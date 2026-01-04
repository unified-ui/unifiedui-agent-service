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
}

// NewMessagesHandler creates a new MessagesHandler.
func NewMessagesHandler(
	docDBClient docdb.Client,
	platformClient platform.Client,
	agentFactory *agents.Factory,
	sessionService session.Service,
) *MessagesHandler {
	return &MessagesHandler{
		docDBClient:    docDBClient,
		platformClient: platformClient,
		agentFactory:   agentFactory,
		sessionService: sessionService,
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
	ID             string                  `json:"id"`
	ConversationID string                  `json:"conversationId"`
	ApplicationID  string                  `json:"applicationId"`
	Message        models.MessageContent   `json:"message"`
	Status         models.MessageStatus    `json:"status,omitempty"`
	ErrorMessage   string                  `json:"errorMessage,omitempty"`
	StatusTraces   []models.StatusTrace    `json:"statusTraces,omitempty"`
	Metadata       *models.AssistantMetadata `json:"metadata,omitempty"`
	CreatedAt      time.Time               `json:"createdAt"`
	UpdatedAt      time.Time               `json:"updatedAt"`
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

	// Get user messages
	userMessages, err := h.docDBClient.Messages().ListUserMessages(ctx, listOpts)
	if err != nil {
		middleware.HandleError(c, errors.NewInternalError("failed to list user messages", err))
		return
	}

	// Get assistant messages
	assistantMessages, err := h.docDBClient.Messages().ListAssistantMessages(ctx, listOpts)
	if err != nil {
		middleware.HandleError(c, errors.NewInternalError("failed to list assistant messages", err))
		return
	}

	// Merge and convert to response
	messages := h.mergeMessages(userMessages, assistantMessages)

	c.JSON(http.StatusOK, GetMessagesResponse{
		Messages: messages,
	})
}

// mergeMessages merges user and assistant messages, sorted by createdAt descending.
func (h *MessagesHandler) mergeMessages(
	userMsgs []*models.UserMessage,
	assistantMsgs []*models.AssistantMessage,
) []MessageResponse {
	result := make([]MessageResponse, 0, len(userMsgs)+len(assistantMsgs))

	// Convert user messages
	for _, msg := range userMsgs {
		result = append(result, MessageResponse{
			ID:             msg.ID,
			ConversationID: msg.ConversationID,
			ApplicationID:  msg.ApplicationID,
			Message:        msg.Message,
			CreatedAt:      msg.CreatedAt,
			UpdatedAt:      msg.UpdatedAt,
		})
	}

	// Convert assistant messages
	for _, msg := range assistantMsgs {
		result = append(result, MessageResponse{
			ID:             msg.ID,
			ConversationID: msg.ConversationID,
			ApplicationID:  msg.ApplicationID,
			Message:        msg.Message,
			Status:         msg.Status,
			ErrorMessage:   msg.ErrorMessage,
			StatusTraces:   msg.StatusTraces,
			Metadata:       msg.Metadata,
			CreatedAt:      msg.CreatedAt,
			UpdatedAt:      msg.UpdatedAt,
		})
	}

	// Sort by createdAt descending
	sortMessagesByCreatedAt(result)

	return result
}

// sortMessagesByCreatedAt sorts messages by createdAt in descending order.
func sortMessagesByCreatedAt(messages []MessageResponse) {
	n := len(messages)
	for i := 0; i < n-1; i++ {
		for j := 0; j < n-i-1; j++ {
			if messages[j].CreatedAt.Before(messages[j+1].CreatedAt) {
				messages[j], messages[j+1] = messages[j+1], messages[j]
			}
		}
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
	ConversationID string         `json:"conversationId,omitempty"`
	ApplicationID  string         `json:"applicationId" binding:"required"`
	Message        MessageContent `json:"message" binding:"required"`
	InvokeConfig   InvokeConfig   `json:"invokeConfig,omitempty"`
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
		// Get agent configuration from Platform Service
		agentConfig, err = h.platformClient.GetAgentConfig(ctx, tenantCtx.TenantID, req.ApplicationID)
		if err != nil {
			middleware.HandleError(c, errors.NewInternalError("failed to get agent configuration", err))
			return
		}

		// Fetch chat history from database if use_unified_chat_history is enabled
		if agentConfig.Settings.UseUnifiedChatHistory {
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
	if err := h.docDBClient.Messages().AddUserMessage(ctx, userMessage); err != nil {
		middleware.HandleError(c, errors.NewInternalError("failed to store user message", err))
		return
	}

	// Create agent clients using factory
	agentClients, err := h.agentFactory.CreateClients(agentConfig)
	if err != nil {
		middleware.HandleError(c, errors.NewInternalError("failed to create agent clients", err))
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
	h.handleStreamingResponse(c, tenantCtx, agentClients, agentConfig, userMessage, assistantMessage, chatHistory)
}

// handleStreamingResponse handles SSE streaming for message responses.
func (h *MessagesHandler) handleStreamingResponse(
	c *gin.Context,
	tenantCtx *middleware.TenantContext,
	agentClients *agents.AgentClients,
	agentConfig *platform.AgentConfig,
	userMessage *models.UserMessage,
	assistantMessage *models.AssistantMessage,
	chatHistory []models.ChatHistoryEntry,
) {
	ctx := c.Request.Context()

	// Create SSE writer
	writer, err := sse.NewWriter(c.Writer)
	if err != nil {
		middleware.HandleError(c, errors.NewInternalError("streaming not supported", err))
		return
	}

	// Build invoke request with chat history
	invokeReq := &agents.InvokeRequest{
		ConversationID: userMessage.ConversationID,
		Message:        userMessage.Message.Content,
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

	var fullContent string
	startTime := time.Now()

	// Send STREAM_START
	writer.WriteStreamStart(assistantMessage.ID)

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
	if err := h.docDBClient.Messages().AddAssistantMessage(ctx, assistantMessage); err != nil {
		// Log error but don't fail - message was already sent to client
	}

	// Update session cache with new chat history
	h.updateSessionCache(ctx, tenantCtx, agentConfig, userMessage, assistantMessage)
}

// saveFailedAssistantMessage saves an assistant message with failed status.
func (h *MessagesHandler) saveFailedAssistantMessage(ctx context.Context, assistantMessage *models.AssistantMessage, errorMsg string) {
	assistantMessage.SetError(errorMsg)
	_ = h.docDBClient.Messages().AddAssistantMessage(ctx, assistantMessage)
}

// updateSessionCache updates the session cache with new messages.
func (h *MessagesHandler) updateSessionCache(
	ctx context.Context,
	tenantCtx *middleware.TenantContext,
	agentConfig *platform.AgentConfig,
	userMessage *models.UserMessage,
	assistantMessage *models.AssistantMessage,
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

// generateMessageID generates a unique message ID.
func generateMessageID() string {
	return "msg_" + uuid.New().String()
}

// generateConversationID generates a unique conversation ID.
func generateConversationID() string {
	return "conv_" + uuid.New().String()
}
