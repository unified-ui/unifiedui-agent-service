// Package handlers provides HTTP handlers for the API.
package handlers

import (
	"context"
	"io"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/unifiedui/agent-service/internal/api/middleware"
	"github.com/unifiedui/agent-service/internal/api/sse"
	"github.com/unifiedui/agent-service/internal/core/docdb"
	"github.com/unifiedui/agent-service/internal/domain/errors"
	"github.com/unifiedui/agent-service/internal/domain/models"
	"github.com/unifiedui/agent-service/internal/services/agents"
	"github.com/unifiedui/agent-service/internal/services/platform"
	"github.com/unifiedui/agent-service/internal/services/session"
	"github.com/unifiedui/agent-service/internal/services/traceimport"
)

const (
	// DefaultChatHistoryCount is the default number of chat history messages.
	DefaultChatHistoryCount = 30
	// DefaultMessagesLimit is the default limit for listing messages.
	DefaultMessagesLimit = 25
)

// MessagesHandler handles message-related endpoints.
type MessagesHandler struct {
	docDBClient    docdb.Client
	platformClient platform.Client
	agentFactory   *agents.Factory
	sessionService session.Service
	importService  *traceimport.ImportService
}

// NewMessagesHandler creates a new MessagesHandler.
func NewMessagesHandler(
	docDBClient docdb.Client,
	platformClient platform.Client,
	agentFactory *agents.Factory,
	sessionService session.Service,
	importService *traceimport.ImportService,
) *MessagesHandler {
	return &MessagesHandler{
		docDBClient:    docDBClient,
		platformClient: platformClient,
		agentFactory:   agentFactory,
		sessionService: sessionService,
		importService:  importService,
	}
}

// GetMessagesRequest represents the query parameters for getting messages.
type GetMessagesRequest struct {
	ConversationID string `form:"conversationId" binding:"required"`
	Limit          int64  `form:"limit" binding:"omitempty,min=1,max=100"`
	Skip           int64  `form:"skip" binding:"omitempty,min=0"`
}

// GetMessagesResponse represents the response for getting messages.
type GetMessagesResponse struct {
	Messages []MessageResponse `json:"messages"`
}

// MessageResponse represents a message in the API response.
type MessageResponse struct {
	ID             string                    `json:"id"`
	Type           models.MessageType        `json:"type"`
	ConversationID string                    `json:"conversationId"`
	ApplicationID  string                    `json:"applicationId"`
	Content        string                    `json:"content"`
	UserID         string                    `json:"userId,omitempty"`
	UserMessageID  string                    `json:"userMessageId,omitempty"`
	Status         models.MessageStatus      `json:"status,omitempty"`
	ErrorMessage   string                    `json:"errorMessage,omitempty"`
	StatusTraces   []models.StatusTrace      `json:"statusTraces,omitempty"`
	Metadata       *models.AssistantMetadata `json:"metadata,omitempty"`
	CreatedAt      time.Time                 `json:"createdAt"`
	UpdatedAt      time.Time                 `json:"updatedAt"`
}

// GetMessages handles GET /tenants/{tenantId}/conversation/messages
// @Summary Get messages
// @Description Retrieves messages for a conversation with pagination (descending order by createdAt)
// @Tags Messages
// @Accept json
// @Produce json
// @Param tenantId path string true "Tenant ID"
// @Param conversationId query string true "Conversation ID"
// @Param limit query int false "Maximum number of messages" default(25) minimum(1) maximum(100)
// @Param skip query int false "Offset for pagination" default(0) minimum(0)
// @Success 200 {object} GetMessagesResponse
// @Failure 400 {object} dto.ErrorResponse
// @Failure 401 {object} dto.ErrorResponse
// @Failure 500 {object} dto.ErrorResponse
// @Security BearerAuth
// @Router /api/v1/agent-service/tenants/{tenantId}/conversation/messages [get]
func (h *MessagesHandler) GetMessages(c *gin.Context) {
	ctx := c.Request.Context()
	tenantCtx := middleware.GetTenantContext(c)

	// Parse query parameters
	var req GetMessagesRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		middleware.HandleError(c, errors.NewValidationError("invalid query parameters", err.Error()))
		return
	}

	// Set defaults
	if req.Limit == 0 {
		req.Limit = DefaultMessagesLimit
	}

	// Build list options
	listOpts := &docdb.ListMessagesOptions{
		ConversationID: req.ConversationID,
		TenantID:       tenantCtx.TenantID,
		Limit:          req.Limit,
		Skip:           req.Skip,
		OrderBy:        docdb.SortOrderDesc,
	}

	// Get messages from unified collection
	messages, err := h.docDBClient.Messages().List(ctx, listOpts)
	if err != nil {
		middleware.HandleError(c, errors.NewInternalError("failed to list messages", err))
		return
	}

	// Convert to response
	response := make([]MessageResponse, 0, len(messages))
	for _, msg := range messages {
		response = append(response, h.toMessageResponse(msg))
	}

	c.JSON(http.StatusOK, GetMessagesResponse{
		Messages: response,
	})
}

// toMessageResponse converts a Message to MessageResponse.
func (h *MessagesHandler) toMessageResponse(msg *models.Message) MessageResponse {
	return MessageResponse{
		ID:             msg.ID,
		Type:           msg.Type,
		ConversationID: msg.ConversationID,
		ApplicationID:  msg.ApplicationID,
		Content:        msg.Content,
		UserID:         msg.UserID,
		UserMessageID:  msg.UserMessageID,
		Status:         msg.Status,
		ErrorMessage:   msg.ErrorMessage,
		StatusTraces:   msg.StatusTraces,
		Metadata:       msg.Metadata,
		CreatedAt:      msg.CreatedAt,
		UpdatedAt:      msg.UpdatedAt,
	}
}

// MessageContent represents the message content in the request.
type MessageContent struct {
	Content     string   `json:"content" binding:"required,min=1,max=32000"`
	Attachments []string `json:"attachments,omitempty"`
}

// InvokeConfig represents configuration options for agent invocation.
type InvokeConfig struct {
	ChatHistoryMessageCount int `json:"chatHistoryMessageCount,omitempty"`
}

// SendMessageRequest represents the request body for sending a message.
type SendMessageRequest struct {
	ConversationID    string         `json:"conversationId,omitempty"`
	ApplicationID     string         `json:"applicationId" binding:"required"`
	ExtConversationID string         `json:"extConversationId,omitempty"`
	Message           MessageContent `json:"message" binding:"required"`
	InvokeConfig      InvokeConfig   `json:"invokeConfig,omitempty"`
}

// SendMessageResponse represents the response for sending a message.
type SendMessageResponse struct {
	UserMessageID      string `json:"userMessageId"`
	AssistantMessageID string `json:"assistantMessageId"`
	ConversationID     string `json:"conversationId"`
}

// SendMessage handles POST /tenants/{tenantId}/conversation/messages
// @Summary Send a message
// @Description Sends a message to an AI agent and returns the response via SSE streaming
// @Tags Messages
// @Accept json
// @Produce text/event-stream
// @Param tenantId path string true "Tenant ID"
// @Param X-Microsoft-Foundry-API-Key header string false "Microsoft Foundry API Key (required for Foundry agents)"
// @Param request body SendMessageRequest true "Message content with applicationId"
// @Success 200 {object} SendMessageResponse
// @Failure 400 {object} dto.ErrorResponse
// @Failure 401 {object} dto.ErrorResponse
// @Failure 500 {object} dto.ErrorResponse
// @Security BearerAuth
// @Router /api/v1/agent-service/tenants/{tenantId}/conversation/messages [post]
func (h *MessagesHandler) SendMessage(c *gin.Context) {
	ctx := c.Request.Context()
	tenantCtx := middleware.GetTenantContext(c)

	// Parse request body
	var req SendMessageRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		middleware.HandleError(c, errors.NewValidationError("invalid request body", err.Error()))
		return
	}

	// Generate conversation ID if not provided
	conversationID := req.ConversationID
	if conversationID == "" {
		conversationID = generateConversationID()
	}

	// Generate message IDs
	userMessageID := generateMessageID()
	assistantMessageID := generateMessageID()

	// Try to get session from cache
	sessionData, err := h.sessionService.GetSession(ctx, tenantCtx.TenantID, tenantCtx.UserID, conversationID)
	if err != nil {
		// Log error but continue - we'll fetch fresh config
		sessionData = nil
	}

	var agentConfig *platform.AgentConfig
	var chatHistory []models.ChatHistoryEntry

	if sessionData != nil {
		// Use cached config and chat history
		agentConfig = sessionData.Config
		chatHistory = sessionData.ChatHistory
	} else {
		// Get auth token from context for Platform Service call
		authToken := middleware.GetToken(c)
		if authToken == "" {
			middleware.HandleError(c, errors.NewUnauthorizedError("auth token not found in context"))
			return
		}

		// Get agent configuration from Platform Service
		// The /config endpoint requires both X-Service-Key AND Bearer token
		agentConfig, err = h.platformClient.GetAgentConfig(ctx, tenantCtx.TenantID, req.ApplicationID, conversationID, authToken)
		if err != nil {
			middleware.HandleError(c, errors.NewInternalError("failed to get agent configuration", err))
			return
		}

		// Fetch chat history from database if use_unified_chat_history is enabled
		// Skip for Foundry agents - they manage their own conversation history
		if agentConfig.Settings.UseUnifiedChatHistory && agentConfig.Type != platform.AgentTypeFoundry {
			chatHistoryCount := agentConfig.Settings.ChatHistoryCount
			if chatHistoryCount == 0 {
				chatHistoryCount = DefaultChatHistoryCount
			}

			listOpts := &docdb.ListMessagesOptions{
				ConversationID: conversationID,
				TenantID:       tenantCtx.TenantID,
				Limit:          int64(chatHistoryCount),
				OrderBy:        docdb.SortOrderAsc, // Get oldest first for proper conversation order
			}

			chatHistory, err = h.docDBClient.Messages().ListChatHistory(ctx, listOpts)
			if err != nil {
				// Log error but continue without chat history
				chatHistory = []models.ChatHistoryEntry{}
			}
		}
	}

	// Create user message
	userMessage := models.NewUserMessage(
		tenantCtx.TenantID,
		conversationID,
		req.ApplicationID,
		tenantCtx.UserID,
		req.Message.Content,
		req.Message.Attachments,
		&models.MessageRequest{
			ApplicationID:  req.ApplicationID,
			ConversationID: req.ConversationID,
			Message: models.MessageRequestContent{
				Content:     req.Message.Content,
				Attachments: req.Message.Attachments,
			},
			InvokeConfig: models.MessageInvokeConfig{
				ChatHistoryMessageCount: req.InvokeConfig.ChatHistoryMessageCount,
			},
		},
	)
	userMessage.ID = userMessageID

	// Store user message
	if err := h.docDBClient.Messages().Add(ctx, userMessage); err != nil {
		middleware.HandleError(c, errors.NewInternalError("failed to store user message", err))
		return
	}

	// Create agent clients using factory
	var agentClients *agents.AgentClients
	var createErr error

	if agentConfig.Type == platform.AgentTypeFoundry {
		// For Foundry, we need the API key from header
		foundryAPIKey := c.GetHeader("X-Microsoft-Foundry-API-Key")
		if foundryAPIKey == "" {
			middleware.HandleError(c, errors.NewValidationError("X-Microsoft-Foundry-API-Key header is required for Foundry agents", ""))
			return
		}
		agentClients, createErr = h.agentFactory.CreateFoundryClients(agentConfig, foundryAPIKey)
	} else {
		agentClients, createErr = h.agentFactory.CreateClients(agentConfig)
	}

	if createErr != nil {
		middleware.HandleError(c, errors.NewInternalError("failed to create agent clients", createErr))
		return
	}
	defer agentClients.Close()

	// Create assistant message (initially pending)
	assistantMessage := models.NewAssistantMessage(
		tenantCtx.TenantID,
		conversationID,
		userMessageID,
		req.ApplicationID,
		"",
		models.MessageStatusPending,
	)
	assistantMessage.ID = assistantMessageID

	// Handle streaming response
	foundryAPIKey := c.GetHeader("X-Microsoft-Foundry-API-Key")
	h.handleStreamingResponse(c, tenantCtx, agentClients, agentConfig, userMessage, assistantMessage, chatHistory, req.ExtConversationID, foundryAPIKey)
}

// handleStreamingResponse handles SSE streaming for message responses.
func (h *MessagesHandler) handleStreamingResponse(
	c *gin.Context,
	tenantCtx *middleware.TenantContext,
	agentClients *agents.AgentClients,
	agentConfig *platform.AgentConfig,
	userMessage *models.Message,
	assistantMessage *models.Message,
	chatHistory []models.ChatHistoryEntry,
	extConversationID string,
	foundryAPIKey string,
) {
	ctx := c.Request.Context()

	// Create SSE writer
	writer, err := sse.NewWriter(c.Writer)
	if err != nil {
		middleware.HandleError(c, errors.NewInternalError("streaming not supported", err))
		return
	}

	// Build invoke request with chat history
	// For Foundry, use ext_conversation_id as the conversation ID, or empty string for new conversations
	conversationIDForInvoke := userMessage.ConversationID
	if agentConfig.Type == platform.AgentTypeFoundry {
		// Foundry manages its own conversation/thread IDs
		// Use ext_conversation_id if provided, otherwise empty string to create a new thread
		conversationIDForInvoke = extConversationID // Will be empty string for new conversations
	}

	invokeReq := &agents.InvokeRequest{
		ConversationID: conversationIDForInvoke,
		Message:        userMessage.Content,
		SessionID:      userMessage.ConversationID,
		ChatHistory:    chatHistory,
	}

	// Get stream reader from workflow client
	streamReader, err := agentClients.WorkflowClient.InvokeStreamReader(ctx, invokeReq)
	if err != nil {
		writer.WriteStreamError("STREAM_ERROR", "Failed to invoke agent", err.Error())
		writer.WriteStreamEnd()
		h.saveFailedAssistantMessage(ctx, assistantMessage, "Failed to invoke agent: "+err.Error())
		return
	}
	defer streamReader.Close()

	startTime := time.Now()

	// Use Foundry-specific streaming handler if applicable
	if agentConfig.Type == platform.AgentTypeFoundry {
		h.handleFoundryStreaming(ctx, writer, streamReader, tenantCtx, agentConfig, userMessage, assistantMessage, startTime)
		// Cache config for Foundry but with empty chat history (Foundry manages its own conversation history)
		h.updateSessionCacheConfigOnly(ctx, tenantCtx, agentConfig, userMessage.ConversationID)

		// Enqueue trace import job for Foundry after streaming completes
		if h.importService != nil && extConversationID != "" && foundryAPIKey != "" {
			h.enqueueFoundryTraceImport(tenantCtx, agentConfig, userMessage, extConversationID, foundryAPIKey)
		}
	} else {
		h.handleDefaultStreaming(ctx, writer, streamReader, agentConfig, userMessage, assistantMessage, startTime)
		// Update session cache with new chat history (only for non-Foundry agents)
		h.updateSessionCache(ctx, tenantCtx, agentConfig, userMessage, assistantMessage)
	}
}

// handleDefaultStreaming handles the default streaming response (for N8N etc.)
func (h *MessagesHandler) handleDefaultStreaming(
	ctx context.Context,
	writer *sse.Writer,
	streamReader agents.StreamReader,
	agentConfig *platform.AgentConfig,
	userMessage *models.Message,
	assistantMessage *models.Message,
	startTime time.Time,
) {
	var fullContent string

	// Send STREAM_START with messageId and conversationId
	writer.WriteStreamStart(assistantMessage.ID, userMessage.ConversationID)

	// Read and forward stream chunks
	for {
		chunk, err := streamReader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			writer.WriteStreamError("STREAM_ERROR", "Error reading stream", err.Error())
			h.saveFailedAssistantMessage(ctx, assistantMessage, "Stream error: "+err.Error())
			break
		}

		switch chunk.Type {
		case agents.ChunkTypeContent:
			fullContent += chunk.Content
			writer.WriteTextStream(chunk.Content)
		case agents.ChunkTypeMetadata:
			if chunk.ExecutionID != "" {
				if assistantMessage.Metadata == nil {
					assistantMessage.Metadata = &models.AssistantMetadata{}
				}
				assistantMessage.Metadata.ExecutionID = chunk.ExecutionID
			}
		case agents.ChunkTypeError:
			if chunk.Error != nil {
				writer.WriteStreamError("CHUNK_ERROR", "Error in chunk", chunk.Error.Error())
			}
		}
	}

	// Send STREAM_END
	writer.WriteStreamEnd()

	// Calculate latency
	latencyMs := time.Since(startTime).Milliseconds()

	// Set success and metadata
	assistantMessage.SetSuccess(fullContent)
	if assistantMessage.Metadata == nil {
		assistantMessage.Metadata = &models.AssistantMetadata{}
	}
	assistantMessage.Metadata.LatencyMs = latencyMs
	assistantMessage.Metadata.AgentType = string(agentConfig.Type)

	// Store assistant message
	if err := h.docDBClient.Messages().Add(ctx, assistantMessage); err != nil {
		// Log error but don't fail - message was already sent to client
	}
}

// handleFoundryStreaming handles Microsoft Foundry streaming responses with multiple messages support.
func (h *MessagesHandler) handleFoundryStreaming(
	ctx context.Context,
	writer *sse.Writer,
	streamReader agents.StreamReader,
	tenantCtx *middleware.TenantContext,
	agentConfig *platform.AgentConfig,
	userMessage *models.Message,
	currentMessage *models.Message,
	startTime time.Time,
) {
	var currentContent string
	allMessages := []*models.Message{currentMessage}

	// Helper function to save and start new message
	saveCurrentAndStartNew := func() {
		// Save current message if it has content
		if currentContent != "" {
			currentMessage.SetSuccess(currentContent)
			h.saveAssistantMessageWithMetadata(ctx, currentMessage, agentConfig, startTime)
		}

		// Signal new message to client
		writer.WriteJSON(sse.EventMessage, &sse.StreamMessage{
			Type: "STREAM_NEW_MESSAGE",
		})

		// Create new assistant message
		currentMessage = models.NewAssistantMessage(
			tenantCtx.TenantID,
			userMessage.ConversationID,
			userMessage.ID,
			userMessage.ApplicationID,
			"",
			models.MessageStatusPending,
		)
		currentMessage.ID = generateMessageID()
		allMessages = append(allMessages, currentMessage)
		currentContent = ""

		// Send new STREAM_START
		writer.WriteStreamStart(currentMessage.ID, userMessage.ConversationID)
	}

	// Send STREAM_START with messageId and conversationId
	writer.WriteStreamStart(currentMessage.ID, userMessage.ConversationID)

	// Read and forward stream chunks
	for {
		chunk, err := streamReader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			writer.WriteStreamError("STREAM_ERROR", "Error reading stream", err.Error())
			h.saveFailedAssistantMessage(ctx, currentMessage, "Stream error: "+err.Error())
			break
		}

		switch chunk.Type {
		case agents.ChunkTypeContent:
			currentContent += chunk.Content
			writer.WriteTextStream(chunk.Content)

		case agents.ChunkTypeNewMessage:
			// New message starting - save previous and create new
			saveCurrentAndStartNew()

			// Apply metadata from chunk if available
			if chunk.Metadata != nil {
				currentMessage.Metadata = h.extractFoundryMetadata(chunk.Metadata)
			}

		case agents.ChunkTypeMetadata:
			// Update current message metadata
			if currentMessage.Metadata == nil {
				currentMessage.Metadata = &models.AssistantMetadata{}
			}
			if chunk.ExecutionID != "" {
				currentMessage.Metadata.ExecutionID = chunk.ExecutionID
			}

			// Extract and store Foundry-specific metadata
			if chunk.Metadata != nil {
				h.mergeFoundryMetadata(currentMessage, chunk.Metadata)
			}

		case agents.ChunkTypeDone:
			// Response completed - extract final metadata
			if chunk.Metadata != nil {
				h.mergeFoundryMetadata(currentMessage, chunk.Metadata)
			}
			if chunk.ExecutionID != "" && currentMessage.Metadata != nil {
				currentMessage.Metadata.ExecutionID = chunk.ExecutionID
			}

		case agents.ChunkTypeError:
			if chunk.Error != nil {
				writer.WriteStreamError("CHUNK_ERROR", "Error in chunk", chunk.Error.Error())
			}
		}
	}

	// Send STREAM_END
	writer.WriteStreamEnd()

	// Save the final message (always save the last one)
	currentMessage.SetSuccess(currentContent)
	h.saveAssistantMessageWithMetadata(ctx, currentMessage, agentConfig, startTime)
}

// saveAssistantMessageWithMetadata saves an assistant message with timing metadata.
func (h *MessagesHandler) saveAssistantMessageWithMetadata(
	ctx context.Context,
	msg *models.Message,
	agentConfig *platform.AgentConfig,
	startTime time.Time,
) {
	latencyMs := time.Since(startTime).Milliseconds()

	if msg.Metadata == nil {
		msg.Metadata = &models.AssistantMetadata{}
	}
	msg.Metadata.LatencyMs = latencyMs
	msg.Metadata.AgentType = string(agentConfig.Type)

	if err := h.docDBClient.Messages().Add(ctx, msg); err != nil {
		// Log error but don't fail - message was already sent to client
	}
}

// extractFoundryMetadata extracts Foundry-specific metadata into AssistantMetadata.
func (h *MessagesHandler) extractFoundryMetadata(metadata map[string]interface{}) *models.AssistantMetadata {
	result := &models.AssistantMetadata{}

	if messageID, ok := metadata["message_id"].(string); ok {
		result.ExecutionID = messageID
	}

	return result
}

// mergeFoundryMetadata merges Foundry metadata into the message's metadata.
func (h *MessagesHandler) mergeFoundryMetadata(msg *models.Message, metadata map[string]interface{}) {
	if msg.Metadata == nil {
		msg.Metadata = &models.AssistantMetadata{}
	}

	// Extract execution ID
	if responseID, ok := metadata["response_id"].(string); ok && msg.Metadata.ExecutionID == "" {
		msg.Metadata.ExecutionID = responseID
	}

	// Extract model if available
	if model, ok := metadata["model"].(string); ok {
		msg.Metadata.Model = model
	}

	// Extract agent name
	if agentName, ok := metadata["agent_name"].(string); ok {
		msg.Metadata.AgentType = agentName
	}

	// Extract token usage
	if usage, ok := metadata["usage"].(map[string]interface{}); ok {
		if inputTokens, ok := usage["input_tokens"].(int); ok {
			msg.Metadata.TokensInput = inputTokens
		}
		if outputTokens, ok := usage["output_tokens"].(int); ok {
			msg.Metadata.TokensOutput = outputTokens
		}
	}

	// Store additional workflow-specific data in status traces
	if workflowType, ok := metadata["type"].(string); ok {
		if workflowType == "workflow_action" {
			msg.AddStatusTrace(
				"workflow_action",
				getStringFromMap(metadata, "kind"),
				"",
				map[string]interface{}{
					"action_id":          getStringFromMap(metadata, "action_id"),
					"parent_action_id":   getStringFromMap(metadata, "parent_action_id"),
					"previous_action_id": getStringFromMap(metadata, "previous_action_id"),
					"status":             getStringFromMap(metadata, "status"),
				},
			)
		}
	}
}

// getStringFromMap safely extracts a string from a map.
func getStringFromMap(m map[string]interface{}, key string) string {
	if val, ok := m[key].(string); ok {
		return val
	}
	return ""
}

// saveFailedAssistantMessage saves an assistant message with failed status.
func (h *MessagesHandler) saveFailedAssistantMessage(ctx context.Context, assistantMessage *models.Message, errorMsg string) {
	assistantMessage.SetError(errorMsg)
	_ = h.docDBClient.Messages().Add(ctx, assistantMessage)
}

// updateSessionCache updates the session cache with new messages.
func (h *MessagesHandler) updateSessionCache(
	ctx context.Context,
	tenantCtx *middleware.TenantContext,
	agentConfig *platform.AgentConfig,
	userMessage *models.Message,
	assistantMessage *models.Message,
) {
	// Only update cache if unified chat history is enabled
	if !agentConfig.Settings.UseUnifiedChatHistory {
		return
	}

	// Get existing session
	sessionData, err := h.sessionService.GetSession(ctx, tenantCtx.TenantID, tenantCtx.UserID, userMessage.ConversationID)
	if err != nil || sessionData == nil {
		// Create new session
		chatHistory := []models.ChatHistoryEntry{
			userMessage.ToChatHistoryEntry(),
			assistantMessage.ToChatHistoryEntry(),
		}
		sessionData = session.NewSessionData(
			agentConfig,
			chatHistory,
			tenantCtx.TenantID,
			tenantCtx.UserID,
			userMessage.ConversationID,
		)
		_ = h.sessionService.SetSession(ctx, sessionData)
		return
	}

	// Update existing session with new messages
	newEntries := []models.ChatHistoryEntry{
		userMessage.ToChatHistoryEntry(),
		assistantMessage.ToChatHistoryEntry(),
	}
	_ = h.sessionService.UpdateChatHistory(ctx, tenantCtx.TenantID, tenantCtx.UserID, userMessage.ConversationID, newEntries)
}

// updateSessionCacheConfigOnly caches only the agent config without chat history.
// Used for Foundry agents which manage their own conversation history.
func (h *MessagesHandler) updateSessionCacheConfigOnly(
	ctx context.Context,
	tenantCtx *middleware.TenantContext,
	agentConfig *platform.AgentConfig,
	conversationID string,
) {
	// Check if session already exists - if so, no need to update (config doesn't change)
	existingSession, _ := h.sessionService.GetSession(ctx, tenantCtx.TenantID, tenantCtx.UserID, conversationID)
	if existingSession != nil {
		return
	}

	// Create new session with config but empty chat history
	sessionData := session.NewSessionData(
		agentConfig,
		[]models.ChatHistoryEntry{}, // Empty chat history for Foundry
		tenantCtx.TenantID,
		tenantCtx.UserID,
		conversationID,
	)
	_ = h.sessionService.SetSession(ctx, sessionData)
}

// enqueueFoundryTraceImport enqueues a trace import job for Foundry agents.
func (h *MessagesHandler) enqueueFoundryTraceImport(
	tenantCtx *middleware.TenantContext,
	agentConfig *platform.AgentConfig,
	userMessage *models.Message,
	extConversationID string,
	foundryAPIKey string,
) {
	req := &traceimport.FoundryImportRequest{
		ImportRequest: traceimport.ImportRequest{
			TenantID:       tenantCtx.TenantID,
			ConversationID: userMessage.ConversationID,
			ApplicationID:  userMessage.ApplicationID,
			Logs:           []string{},
			UserID:         tenantCtx.UserID,
		},
		FoundryConversationID: extConversationID,
		ProjectEndpoint:       agentConfig.Settings.ProjectEndpoint,
		APIVersion:            agentConfig.Settings.APIVersion,
		FoundryAPIToken:       foundryAPIKey,
	}

	h.importService.EnqueueFoundryImport(req)
}

// generateMessageID generates a unique message ID.
func generateMessageID() string {
	return "msg_" + uuid.New().String()
}

// generateConversationID generates a unique conversation ID.
func generateConversationID() string {
	return "conv_" + uuid.New().String()
}
