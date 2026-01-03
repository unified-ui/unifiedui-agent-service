// Package handlers provides HTTP handlers for the API.
package handlers

import (
	"io"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/bson"

	"github.com/unifiedui/agent-service/internal/api/middleware"
	"github.com/unifiedui/agent-service/internal/api/sse"
	"github.com/unifiedui/agent-service/internal/core/docdb"
	"github.com/unifiedui/agent-service/internal/domain/errors"
	"github.com/unifiedui/agent-service/internal/domain/models"
	"github.com/unifiedui/agent-service/internal/services/agents"
	"github.com/unifiedui/agent-service/internal/services/platform"
)

// MessagesHandler handles message-related endpoints.
type MessagesHandler struct {
	docDBClient    docdb.Client
	platformClient platform.Client
	agentFactory   *agents.Factory
}

// NewMessagesHandler creates a new MessagesHandler.
func NewMessagesHandler(docDBClient docdb.Client, platformClient platform.Client, agentFactory *agents.Factory) *MessagesHandler {
	return &MessagesHandler{
		docDBClient:    docDBClient,
		platformClient: platformClient,
		agentFactory:   agentFactory,
	}
}

// GetMessagesRequest represents the query parameters for getting messages.
type GetMessagesRequest struct {
	ConversationID string `form:"conversationId" binding:"required"`
	Limit          int64  `form:"limit" binding:"omitempty,min=1,max=100"`
	Offset         int64  `form:"offset" binding:"omitempty,min=0"`
}

// GetMessagesResponse represents the response for getting messages.
type GetMessagesResponse struct {
	Messages []*models.Message `json:"messages"`
	Total    int64             `json:"total"`
	Limit    int64             `json:"limit"`
	Offset   int64             `json:"offset"`
}

// GetMessages handles GET /tenants/{tenantId}/conversation/messages
// @Summary Get messages
// @Description Retrieves messages for a conversation with pagination
// @Tags Messages
// @Accept json
// @Produce json
// @Param tenantId path string true "Tenant ID"
// @Param conversationId query string true "Conversation ID"
// @Param limit query int false "Maximum number of messages" default(50) minimum(1) maximum(100)
// @Param offset query int false "Offset for pagination" default(0) minimum(0)
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
		req.Limit = 50
	}

	// Build filter
	filter := bson.M{
		"tenantId":       tenantCtx.TenantID,
		"conversationId": req.ConversationID,
	}

	// Get total count
	total, err := h.docDBClient.Messages().CountDocuments(ctx, filter)
	if err != nil {
		middleware.HandleError(c, errors.NewInternalError("failed to count messages", err))
		return
	}

	// Get messages
	opts := &docdb.FindOptions{
		Limit: req.Limit,
		Skip:  req.Offset,
		Sort:  bson.M{"createdAt": -1},
	}

	cursor, err := h.docDBClient.Messages().Find(ctx, filter, opts)
	if err != nil {
		middleware.HandleError(c, errors.NewInternalError("failed to find messages", err))
		return
	}
	defer cursor.Close(ctx)

	var messages []*models.Message
	if err := cursor.All(ctx, &messages); err != nil {
		middleware.HandleError(c, errors.NewInternalError("failed to decode messages", err))
		return
	}

	c.JSON(http.StatusOK, GetMessagesResponse{
		Messages: messages,
		Total:    total,
		Limit:    req.Limit,
		Offset:   req.Offset,
	})
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
	Message *models.Message `json:"message"`
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

	// Get agent configuration from Platform Service
	agentConfig, err := h.platformClient.GetAgentConfig(ctx, tenantCtx.TenantID, req.ApplicationID)
	if err != nil {
		middleware.HandleError(c, errors.NewInternalError("failed to get agent configuration", err))
		return
	}

	// Create agent clients using factory
	agentClients, err := h.agentFactory.CreateClients(agentConfig)
	if err != nil {
		middleware.HandleError(c, errors.NewInternalError("failed to create agent clients", err))
		return
	}
	defer agentClients.Close()

	// Create user message
	userMessage := models.NewMessage(
		tenantCtx.TenantID,
		conversationID,
		string(models.RoleUser),
		req.Message.Content,
		req.ApplicationID,
		tenantCtx.UserID,
	)
	userMessage.ID = generateMessageID()

	// Store user message
	if _, err := h.docDBClient.Messages().InsertOne(ctx, userMessage); err != nil {
		middleware.HandleError(c, errors.NewInternalError("failed to store user message", err))
		return
	}

	// Handle streaming response
	h.handleStreamingResponse(c, tenantCtx, agentClients, userMessage, conversationID)
}

// handleStreamingResponse handles SSE streaming for message responses.
func (h *MessagesHandler) handleStreamingResponse(
	c *gin.Context,
	tenantCtx *middleware.TenantContext,
	agentClients *agents.AgentClients,
	userMessage *models.Message,
	conversationID string,
) {
	ctx := c.Request.Context()

	// Create SSE writer
	writer, err := sse.NewWriter(c.Writer)
	if err != nil {
		middleware.HandleError(c, errors.NewInternalError("streaming not supported", err))
		return
	}

	// Build invoke request
	invokeReq := &agents.InvokeRequest{
		ConversationID: conversationID,
		Message:        userMessage.Content,
		SessionID:      conversationID, // Use conversation ID as session ID for now
	}

	// Get stream reader from workflow client
	streamReader, err := agentClients.WorkflowClient.InvokeStreamReader(ctx, invokeReq)
	if err != nil {
		writer.WriteError("STREAM_ERROR", "Failed to invoke agent", err.Error())
		writer.WriteDone()
		return
	}
	defer streamReader.Close()

	// Create assistant message ID
	assistantMessageID := generateMessageID()
	var fullContent string

	// Read and forward stream chunks
	for {
		chunk, err := streamReader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			writer.WriteError("STREAM_ERROR", "Error reading stream", err.Error())
			break
		}

		switch chunk.Type {
		case agents.ChunkTypeContent:
			fullContent += chunk.Content
			writer.WriteMessageChunk(&sse.MessageChunk{
				Content:   chunk.Content,
				MessageID: assistantMessageID,
				Done:      false,
			})
		case agents.ChunkTypeMetadata:
			// Could send metadata as separate event type if needed
		case agents.ChunkTypeError:
			if chunk.Error != nil {
				writer.WriteError("CHUNK_ERROR", "Error in chunk", chunk.Error.Error())
			}
		}
	}

	// Send done chunk
	writer.WriteMessageChunk(&sse.MessageChunk{
		Content:   "",
		MessageID: assistantMessageID,
		Done:      true,
	})
	writer.WriteDone()

	// Store assistant message
	assistantMessage := models.NewMessage(
		tenantCtx.TenantID,
		conversationID,
		string(models.RoleAssistant),
		fullContent,
		userMessage.AgentID,
		"",
	)
	assistantMessage.ID = assistantMessageID

	if _, err := h.docDBClient.Messages().InsertOne(ctx, assistantMessage); err != nil {
		// Log error but don't fail - message was already sent to client
		// TODO: Add proper logging
	}
}

// generateMessageID generates a unique message ID.
func generateMessageID() string {
	return "msg_" + time.Now().Format("20060102150405.000000")
}

// generateConversationID generates a unique conversation ID.
func generateConversationID() string {
	return "conv_" + time.Now().Format("20060102150405.000000")
}
