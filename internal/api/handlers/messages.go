// Package handlers provides HTTP handlers for the API.
package handlers

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/bson"

	"github.com/unifiedui/agent-service/internal/api/middleware"
	"github.com/unifiedui/agent-service/internal/api/sse"
	"github.com/unifiedui/agent-service/internal/core/docdb"
	"github.com/unifiedui/agent-service/internal/domain/errors"
	"github.com/unifiedui/agent-service/internal/domain/models"
)

// MessagesHandler handles message-related endpoints.
type MessagesHandler struct {
	docDBClient docdb.Client
}

// NewMessagesHandler creates a new MessagesHandler.
func NewMessagesHandler(docDBClient docdb.Client) *MessagesHandler {
	return &MessagesHandler{
		docDBClient: docDBClient,
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

// SendMessageRequest represents the request body for sending a message.
type SendMessageRequest struct {
	ConversationID string `json:"conversationId" binding:"required"`
	Content        string `json:"content" binding:"required,min=1,max=32000"`
	AgentID        string `json:"agentId" binding:"required"`
	Stream         bool   `json:"stream"`
}

// SendMessageResponse represents the response for sending a message.
type SendMessageResponse struct {
	Message *models.Message `json:"message"`
}

// SendMessage handles POST /tenants/{tenantId}/conversation/messages
// @Summary Send a message
// @Description Sends a message to an AI agent and returns the response (supports SSE streaming)
// @Tags Messages
// @Accept json
// @Produce json
// @Param tenantId path string true "Tenant ID"
// @Param request body SendMessageRequest true "Message content with conversationId"
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

	// Create user message
	userMessage := models.NewMessage(
		tenantCtx.TenantID,
		req.ConversationID,
		string(models.RoleUser),
		req.Content,
		req.AgentID,
		tenantCtx.UserID,
	)
	userMessage.ID = generateMessageID()

	// Store user message
	if _, err := h.docDBClient.Messages().InsertOne(ctx, userMessage); err != nil {
		middleware.HandleError(c, errors.NewInternalError("failed to store user message", err))
		return
	}

	// Handle streaming response
	if req.Stream {
		h.handleStreamingResponse(c, tenantCtx, userMessage, req.AgentID)
		return
	}

	// Handle non-streaming response
	h.handleNonStreamingResponse(c, tenantCtx, userMessage, req.AgentID)
}

// handleStreamingResponse handles SSE streaming for message responses.
func (h *MessagesHandler) handleStreamingResponse(c *gin.Context, tenantCtx *middleware.TenantContext, userMessage *models.Message, agentID string) {
	writer, err := sse.NewWriter(c.Writer)
	if err != nil {
		middleware.HandleError(c, errors.NewInternalError("streaming not supported", err))
		return
	}

	// TODO: Implement actual agent communication
	// This is a placeholder for demonstration

	// Send message chunks
	writer.WriteMessageChunk(&sse.MessageChunk{
		Content:   "This is a placeholder response. ",
		MessageID: generateMessageID(),
		Done:      false,
	})

	writer.WriteMessageChunk(&sse.MessageChunk{
		Content:   "Agent integration pending.",
		MessageID: "",
		Done:      true,
	})

	writer.WriteDone()
}

// handleNonStreamingResponse handles non-streaming message responses.
func (h *MessagesHandler) handleNonStreamingResponse(c *gin.Context, tenantCtx *middleware.TenantContext, userMessage *models.Message, agentID string) {
	ctx := c.Request.Context()

	// TODO: Implement actual agent communication
	// This is a placeholder response

	assistantMessage := models.NewMessage(
		tenantCtx.TenantID,
		tenantCtx.ConversationID,
		string(models.RoleAssistant),
		"This is a placeholder response. Agent integration pending.",
		agentID,
		"",
	)
	assistantMessage.ID = generateMessageID()

	// Store assistant message
	if _, err := h.docDBClient.Messages().InsertOne(ctx, assistantMessage); err != nil {
		middleware.HandleError(c, errors.NewInternalError("failed to store assistant message", err))
		return
	}

	c.JSON(http.StatusOK, SendMessageResponse{
		Message: assistantMessage,
	})
}

// generateMessageID generates a unique message ID.
func generateMessageID() string {
	return "msg_" + time.Now().Format("20060102150405.000000")
}
