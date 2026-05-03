// Package handlers provides HTTP handlers for the API.
package handlers

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/unifiedui/agent-service/internal/api/middleware"
	"github.com/unifiedui/agent-service/internal/core/docdb"
	"github.com/unifiedui/agent-service/internal/domain/errors"
	"github.com/unifiedui/agent-service/internal/domain/models"
	"github.com/unifiedui/agent-service/internal/services/agents"
	"github.com/unifiedui/agent-service/internal/services/ai"
	"github.com/unifiedui/agent-service/internal/services/configcache"
	"github.com/unifiedui/agent-service/internal/services/platform"
	"github.com/unifiedui/agent-service/internal/services/session"
	"github.com/unifiedui/agent-service/internal/services/telemetry"
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
	configCache    configcache.Service
	importService  *traceimport.ImportService
	aiService      ai.Service
	telemetry      *telemetry.Emitter
}

// NewMessagesHandler creates a new MessagesHandler.
func NewMessagesHandler(
	docDBClient docdb.Client,
	platformClient platform.Client,
	agentFactory *agents.Factory,
	sessionService session.Service,
	configCache configcache.Service,
	importService *traceimport.ImportService,
	aiService ai.Service,
) *MessagesHandler {
	return &MessagesHandler{
		docDBClient:    docDBClient,
		platformClient: platformClient,
		agentFactory:   agentFactory,
		sessionService: sessionService,
		configCache:    configCache,
		importService:  importService,
		aiService:      aiService,
	}
}

// WithTelemetry attaches a telemetry emitter to the handler. Safe to omit;
// when nil, EmitMetric becomes a no-op.
func (h *MessagesHandler) WithTelemetry(emitter *telemetry.Emitter) *MessagesHandler {
	h.telemetry = emitter
	return h
}

// EmitMetric forwards a per-message metric event to the configured telemetry
// emitter. The call is non-blocking and silently drops when no emitter is
// configured or when the buffer is saturated.
func (h *MessagesHandler) EmitMetric(event telemetry.MetricEvent) {
	if h.telemetry == nil {
		return
	}
	h.telemetry.Emit(event)
}

// emitSendMetric constructs a MetricEvent for a completed SendMessage request
// and forwards it to the telemetry emitter. Safe to call with a partially
// populated assistant message (e.g. on error or cancellation).
func (h *MessagesHandler) emitSendMetric(
	tenantCtx *middleware.TenantContext,
	agentConfig *platform.AgentConfig,
	assistantMessage *models.Message,
	startedAt time.Time,
) {
	if h.telemetry == nil || assistantMessage == nil {
		return
	}
	latencyMs := int(time.Since(startedAt).Milliseconds())
	status := "success"
	errorCode := ""
	switch assistantMessage.Status {
	case models.MessageStatusFailed:
		status = "failed"
		errorCode = "STREAM_ERROR"
	case models.MessageStatusCanceled:
		status = "cancelled"
	case models.MessageStatusPending:
		status = "pending"
	case models.MessageStatusSuccess:
		status = "success"
	}
	event := telemetry.MetricEvent{
		TenantID:       tenantCtx.TenantID,
		MessageID:      assistantMessage.ID,
		ChatAgentID:    assistantMessage.ChatAgentID,
		ConversationID: assistantMessage.ConversationID,
		UserID:         tenantCtx.UserID,
		LatencyMs:      latencyMs,
		Status:         status,
		ErrorCode:      errorCode,
	}
	if agentConfig != nil {
		event.Provider = string(agentConfig.Type)
		event.AgentType = string(agentConfig.Type)
	}
	if assistantMessage.Metadata != nil {
		event.TokensInput = assistantMessage.Metadata.TokensInput
		event.TokensOutput = assistantMessage.Metadata.TokensOutput
		event.Model = assistantMessage.Metadata.Model
		if assistantMessage.Metadata.AgentType != "" {
			event.AgentType = assistantMessage.Metadata.AgentType
		}
		if assistantMessage.Metadata.LatencyMs > 0 {
			event.LatencyMs = int(assistantMessage.Metadata.LatencyMs)
		}
	}
	h.telemetry.Emit(event)
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
	HasMore  bool              `json:"hasMore"`
}

// MessageResponse represents a message in the API response.
type MessageResponse struct {
	ID                  string                      `json:"id"`
	Type                models.MessageType          `json:"type"`
	ConversationID      string                      `json:"conversationId"`
	ChatAgentID         string                      `json:"chatAgentId"`
	Content             string                      `json:"content"`
	UserID              string                      `json:"userId,omitempty"`
	UserMessageID       string                      `json:"userMessageId,omitempty"`
	Status              models.MessageStatus        `json:"status,omitempty"`
	ErrorMessage        string                      `json:"errorMessage,omitempty"`
	StatusTraces        []models.StatusTrace        `json:"statusTraces,omitempty"`
	Metadata            *models.AssistantMetadata   `json:"metadata,omitempty"`
	AttachmentsMetadata []models.AttachmentMetadata `json:"attachmentsMetadata,omitempty"`
	Extra               map[string]interface{}      `json:"extra,omitempty"`
	CreatedAt           time.Time                   `json:"createdAt"`
	UpdatedAt           time.Time                   `json:"updatedAt"`
}

// FileAttachment represents a file attachment in the message request.
type FileAttachment struct {
	Type     string `json:"type" binding:"required,oneof=image file audio"`
	ImageURL string `json:"imageUrl,omitempty"`
	FileData string `json:"fileData,omitempty"`
	Filename string `json:"filename,omitempty"`
	MimeType string `json:"mimeType,omitempty"`
	Detail   string `json:"detail,omitempty"`
	FileID   string `json:"fileId,omitempty"`
}

// MessageContent represents the message content in the request.
type MessageContent struct {
	Content     string           `json:"content" binding:"required,min=1,max=32000"`
	Attachments []string         `json:"attachments,omitempty"`
	Files       []FileAttachment `json:"files,omitempty"`
}

// InvokeConfig represents configuration options for agent invocation.
type InvokeConfig struct {
	ChatHistoryMessageCount int               `json:"chatHistoryMessageCount,omitempty"`
	ContextData             map[string]string `json:"contextData,omitempty"`
}

// SendMessageRequest represents the request body for sending a message.
type SendMessageRequest struct {
	ConversationID    string                 `json:"conversationId,omitempty"`
	ChatAgentID       string                 `json:"chatAgentId" binding:"required"`
	ExtConversationID string                 `json:"extConversationId,omitempty"`
	Message           MessageContent         `json:"message" binding:"required"`
	InvokeConfig      InvokeConfig           `json:"invokeConfig,omitempty"`
	Extra             map[string]interface{} `json:"extra,omitempty"`
}

// SendMessageResponse represents the response for sending a message.
type SendMessageResponse struct {
	UserMessageID      string `json:"userMessageId"`
	AssistantMessageID string `json:"assistantMessageId"`
	ConversationID     string `json:"conversationId"`
}

// EditMessageRequest represents the request body for editing a message.
type EditMessageRequest struct {
	Content string `json:"content" binding:"required,min=1,max=32000"`
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

	var req GetMessagesRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		middleware.HandleError(c, errors.NewValidationError("invalid query parameters", err.Error()))
		return
	}

	if req.Limit == 0 {
		req.Limit = DefaultMessagesLimit
	}

	listOpts := &docdb.ListMessagesOptions{
		ConversationID: req.ConversationID,
		TenantID:       tenantCtx.TenantID,
		Limit:          req.Limit + 1,
		Skip:           req.Skip,
		OrderBy:        docdb.SortOrderDesc,
	}

	messages, err := h.docDBClient.Messages().List(ctx, listOpts)
	if err != nil {
		middleware.HandleError(c, errors.NewInternalError("failed to list messages", err))
		return
	}

	hasMore := int64(len(messages)) > req.Limit
	if hasMore {
		messages = messages[:req.Limit]
	}

	response := make([]MessageResponse, 0, len(messages))
	for _, msg := range messages {
		response = append(response, h.toMessageResponse(msg))
	}

	c.JSON(http.StatusOK, GetMessagesResponse{
		Messages: response,
		HasMore:  hasMore,
	})
}

func (h *MessagesHandler) toMessageResponse(msg *models.Message) MessageResponse {
	var extra map[string]interface{}
	if msg.Request != nil && len(msg.Request.Extra) > 0 {
		extra = msg.Request.Extra
	}

	return MessageResponse{
		ID:                  msg.ID,
		Type:                msg.Type,
		ConversationID:      msg.ConversationID,
		ChatAgentID:         msg.ChatAgentID,
		Content:             msg.Content,
		UserID:              msg.UserID,
		UserMessageID:       msg.UserMessageID,
		Status:              msg.Status,
		ErrorMessage:        msg.ErrorMessage,
		StatusTraces:        msg.StatusTraces,
		Metadata:            msg.Metadata,
		AttachmentsMetadata: msg.AttachmentsMetadata,
		Extra:               extra,
		CreatedAt:           msg.CreatedAt,
		UpdatedAt:           msg.UpdatedAt,
	}
}

// DeleteMessage handles DELETE /tenants/{tenantId}/conversations/{conversationId}/messages/{messageId}
// @Summary Delete a message
// @Description Deletes a user message and its associated assistant response
// @Tags Messages
// @Accept json
// @Produce json
// @Param tenantId path string true "Tenant ID"
// @Param conversationId path string true "Conversation ID"
// @Param messageId path string true "Message ID"
// @Success 204 "No Content"
// @Failure 400 {object} dto.ErrorResponse
// @Failure 401 {object} dto.ErrorResponse
// @Failure 404 {object} dto.ErrorResponse
// @Failure 500 {object} dto.ErrorResponse
// @Security BearerAuth
// @Router /api/v1/agent-service/tenants/{tenantId}/conversations/{conversationId}/messages/{messageId} [delete]
func (h *MessagesHandler) DeleteMessage(c *gin.Context) {
	ctx := c.Request.Context()
	tenantCtx := middleware.GetTenantContext(c)
	messageID := middleware.SanitizePathParam(c, "messageId")

	message, err := h.docDBClient.Messages().Get(ctx, messageID)
	if err != nil {
		middleware.HandleError(c, errors.NewInternalError("failed to get message", err))
		return
	}
	if message == nil {
		middleware.HandleError(c, errors.NewNotFoundError("message", messageID))
		return
	}

	if message.TenantID != tenantCtx.TenantID {
		middleware.HandleError(c, errors.NewNotFoundError("message", messageID))
		return
	}

	_, err = h.docDBClient.Messages().Delete(ctx, &docdb.DeleteMessagesOptions{
		MessageID: messageID,
		TenantID:  tenantCtx.TenantID,
	})
	if err != nil {
		middleware.HandleError(c, errors.NewInternalError("failed to delete message", err))
		return
	}

	if message.Type == models.MessageTypeUser {
		assistantMsg, err := h.docDBClient.Messages().GetByUserMessageID(ctx, messageID)
		if err == nil && assistantMsg != nil {
			_, _ = h.docDBClient.Messages().Delete(ctx, &docdb.DeleteMessagesOptions{
				MessageID: assistantMsg.ID,
				TenantID:  tenantCtx.TenantID,
			})
		}
	}

	c.Status(http.StatusNoContent)
}

// EditMessage handles PUT /tenants/{tenantId}/conversations/{conversationId}/messages/{messageId}
// @Summary Edit a message
// @Description Updates the content of an existing user message
// @Tags Messages
// @Accept json
// @Produce json
// @Param tenantId path string true "Tenant ID"
// @Param conversationId path string true "Conversation ID"
// @Param messageId path string true "Message ID"
// @Param request body EditMessageRequest true "Updated message content"
// @Success 200 {object} MessageResponse
// @Failure 400 {object} dto.ErrorResponse
// @Failure 401 {object} dto.ErrorResponse
// @Failure 403 {object} dto.ErrorResponse
// @Failure 404 {object} dto.ErrorResponse
// @Failure 500 {object} dto.ErrorResponse
// @Security BearerAuth
// @Router /api/v1/agent-service/tenants/{tenantId}/conversations/{conversationId}/messages/{messageId} [put]
func (h *MessagesHandler) EditMessage(c *gin.Context) {
	ctx := c.Request.Context()
	tenantCtx := middleware.GetTenantContext(c)
	messageID := middleware.SanitizePathParam(c, "messageId")

	var req EditMessageRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		middleware.HandleError(c, errors.NewValidationError("invalid request body", err.Error()))
		return
	}

	message, err := h.docDBClient.Messages().Get(ctx, messageID)
	if err != nil {
		middleware.HandleError(c, errors.NewInternalError("failed to get message", err))
		return
	}
	if message == nil {
		middleware.HandleError(c, errors.NewNotFoundError("message", messageID))
		return
	}

	if message.TenantID != tenantCtx.TenantID {
		middleware.HandleError(c, errors.NewNotFoundError("message", messageID))
		return
	}

	if message.Type != models.MessageTypeUser {
		middleware.HandleError(c, errors.NewForbiddenError("only user messages can be edited"))
		return
	}

	message.Content = req.Content
	message.UpdatedAt = time.Now().UTC()

	if err := h.docDBClient.Messages().Update(ctx, message); err != nil {
		middleware.HandleError(c, errors.NewInternalError("failed to update message", err))
		return
	}

	c.JSON(http.StatusOK, h.toMessageResponse(message))
}

// SearchMessagesRequest represents the query parameters for searching messages.
type SearchMessagesRequest struct {
	Query string `form:"query" binding:"required,min=1,max=500"`
	Limit int64  `form:"limit" binding:"omitempty,min=1,max=50"`
	Skip  int64  `form:"skip" binding:"omitempty,min=0"`
}

// SearchMessagesResponse represents the response for searching messages.
type SearchMessagesResponse struct {
	Messages []MessageResponse `json:"messages"`
}

// SearchMessages handles GET /tenants/{tenantId}/conversation/messages/search
// @Summary Search messages
// @Description Searches messages by content text using case-insensitive matching across all conversations
// @Tags Messages
// @Accept json
// @Produce json
// @Param tenantId path string true "Tenant ID"
// @Param query query string true "Search query text"
// @Param limit query int false "Maximum number of messages" default(20) minimum(1) maximum(50)
// @Param skip query int false "Offset for pagination" default(0) minimum(0)
// @Success 200 {object} SearchMessagesResponse
// @Failure 400 {object} dto.ErrorResponse
// @Failure 401 {object} dto.ErrorResponse
// @Failure 500 {object} dto.ErrorResponse
// @Security BearerAuth
// @Router /api/v1/agent-service/tenants/{tenantId}/conversation/messages/search [get]
func (h *MessagesHandler) SearchMessages(c *gin.Context) {
	ctx := c.Request.Context()
	tenantCtx := middleware.GetTenantContext(c)

	var req SearchMessagesRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		middleware.HandleError(c, errors.NewValidationError("invalid query parameters", err.Error()))
		return
	}

	if req.Limit == 0 {
		req.Limit = 20
	}

	searchOpts := &docdb.SearchMessagesOptions{
		TenantID: tenantCtx.TenantID,
		Query:    req.Query,
		Limit:    req.Limit,
		Skip:     req.Skip,
	}

	messages, err := h.docDBClient.Messages().Search(ctx, searchOpts)
	if err != nil {
		middleware.HandleError(c, errors.NewInternalError("failed to search messages", err))
		return
	}

	response := make([]MessageResponse, 0, len(messages))
	for _, msg := range messages {
		response = append(response, h.toMessageResponse(msg))
	}

	c.JSON(http.StatusOK, SearchMessagesResponse{
		Messages: response,
	})
}
