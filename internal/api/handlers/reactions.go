// Package handlers provides HTTP handlers for the API.
package handlers

import (
	"context"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/unifiedui/agent-service/internal/api/middleware"
	"github.com/unifiedui/agent-service/internal/core/docdb"
	"github.com/unifiedui/agent-service/internal/domain/errors"
	"github.com/unifiedui/agent-service/internal/domain/models"
	"github.com/unifiedui/agent-service/internal/services/platform"
)

// ReactionsHandler handles message reaction endpoints.
type ReactionsHandler struct {
	docDBClient    docdb.Client
	platformClient platform.Client
}

// NewReactionsHandler creates a new ReactionsHandler.
func NewReactionsHandler(docDBClient docdb.Client, platformClient platform.Client) *ReactionsHandler {
	return &ReactionsHandler{
		docDBClient:    docDBClient,
		platformClient: platformClient,
	}
}

// UpsertReactionRequest represents the request body for creating/updating a reaction.
type UpsertReactionRequest struct {
	Reaction     models.ReactionType `json:"reaction" binding:"required"`
	FeedbackText string              `json:"feedbackText,omitempty"`
}

// ReactionResponse represents a reaction in the API response.
type ReactionResponse struct {
	ID             string              `json:"id"`
	TenantID       string              `json:"tenantId"`
	ConversationID string              `json:"conversationId"`
	MessageID      string              `json:"messageId"`
	UserID         string              `json:"userId"`
	Reaction       models.ReactionType `json:"reaction"`
	FeedbackText   string              `json:"feedbackText,omitempty"`
	CreatedAt      time.Time           `json:"createdAt"`
	UpdatedAt      time.Time           `json:"updatedAt"`
}

// ListReactionsResponse represents the response for listing reactions.
type ListReactionsResponse struct {
	Reactions []ReactionResponse `json:"reactions"`
}

// UpsertReaction handles POST /tenants/{tenantId}/conversations/{conversationId}/messages/{messageId}/reactions
// @Summary Create or update a reaction
// @Description Creates or updates a user's reaction to a message (one per user per message)
// @Tags Reactions
// @Accept json
// @Produce json
// @Param tenantId path string true "Tenant ID"
// @Param conversationId path string true "Conversation ID"
// @Param messageId path string true "Message ID"
// @Param request body UpsertReactionRequest true "Reaction data"
// @Success 200 {object} ReactionResponse
// @Failure 400 {object} dto.ErrorResponse
// @Failure 401 {object} dto.ErrorResponse
// @Failure 404 {object} dto.ErrorResponse
// @Failure 500 {object} dto.ErrorResponse
// @Security BearerAuth
// @Router /api/v1/agent-service/tenants/{tenantId}/conversations/{conversationId}/messages/{messageId}/reactions [post]
func (h *ReactionsHandler) UpsertReaction(c *gin.Context) {
	ctx := c.Request.Context()
	tenantCtx := middleware.GetTenantContext(c)
	conversationID := middleware.SanitizePathParam(c, "conversationId")
	messageID := middleware.SanitizePathParam(c, "messageId")

	var req UpsertReactionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		middleware.HandleError(c, errors.NewValidationError("invalid request body", err.Error()))
		return
	}

	if !models.IsValidReactionType(req.Reaction) {
		middleware.HandleError(c, errors.NewValidationError("invalid reaction type", "must be 'thumbs_up' or 'thumbs_down'"))
		return
	}

	userID, err := h.getUserID(ctx, middleware.GetToken(c))
	if err != nil {
		middleware.HandleError(c, errors.NewUnauthorizedError("failed to resolve user identity"))
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

	reaction := models.NewMessageReaction(
		tenantCtx.TenantID,
		conversationID,
		messageID,
		userID,
		req.Reaction,
		req.FeedbackText,
	)
	reaction.ID = uuid.New().String()

	if err := h.docDBClient.Reactions().Upsert(ctx, reaction); err != nil {
		middleware.HandleError(c, errors.NewInternalError("failed to upsert reaction", err))
		return
	}

	c.JSON(http.StatusOK, h.toReactionResponse(reaction))
}

// DeleteReaction handles DELETE /tenants/{tenantId}/conversations/{conversationId}/messages/{messageId}/reactions
// @Summary Delete a reaction
// @Description Removes the current user's reaction from a message
// @Tags Reactions
// @Accept json
// @Produce json
// @Param tenantId path string true "Tenant ID"
// @Param conversationId path string true "Conversation ID"
// @Param messageId path string true "Message ID"
// @Success 204 "No Content"
// @Failure 401 {object} dto.ErrorResponse
// @Failure 500 {object} dto.ErrorResponse
// @Security BearerAuth
// @Router /api/v1/agent-service/tenants/{tenantId}/conversations/{conversationId}/messages/{messageId}/reactions [delete]
func (h *ReactionsHandler) DeleteReaction(c *gin.Context) {
	ctx := c.Request.Context()
	tenantCtx := middleware.GetTenantContext(c)
	messageID := middleware.SanitizePathParam(c, "messageId")

	userID, err := h.getUserID(ctx, middleware.GetToken(c))
	if err != nil {
		middleware.HandleError(c, errors.NewUnauthorizedError("failed to resolve user identity"))
		return
	}

	opts := &docdb.DeleteReactionOptions{
		TenantID:  tenantCtx.TenantID,
		MessageID: messageID,
		UserID:    userID,
	}

	if err := h.docDBClient.Reactions().Delete(ctx, opts); err != nil {
		middleware.HandleError(c, errors.NewInternalError("failed to delete reaction", err))
		return
	}

	c.Status(http.StatusNoContent)
}

// GetReactions handles GET /tenants/{tenantId}/conversations/{conversationId}/messages/{messageId}/reactions
// @Summary Get reactions for a message
// @Description Retrieves all reactions for a specific message
// @Tags Reactions
// @Accept json
// @Produce json
// @Param tenantId path string true "Tenant ID"
// @Param conversationId path string true "Conversation ID"
// @Param messageId path string true "Message ID"
// @Success 200 {object} ListReactionsResponse
// @Failure 401 {object} dto.ErrorResponse
// @Failure 500 {object} dto.ErrorResponse
// @Security BearerAuth
// @Router /api/v1/agent-service/tenants/{tenantId}/conversations/{conversationId}/messages/{messageId}/reactions [get]
func (h *ReactionsHandler) GetReactions(c *gin.Context) {
	ctx := c.Request.Context()
	tenantCtx := middleware.GetTenantContext(c)
	conversationID := middleware.SanitizePathParam(c, "conversationId")
	messageID := middleware.SanitizePathParam(c, "messageId")

	opts := &docdb.ListReactionsOptions{
		TenantID:       tenantCtx.TenantID,
		ConversationID: conversationID,
		MessageID:      messageID,
	}

	reactions, err := h.docDBClient.Reactions().ListByMessage(ctx, opts)
	if err != nil {
		middleware.HandleError(c, errors.NewInternalError("failed to list reactions", err))
		return
	}

	response := make([]ReactionResponse, 0, len(reactions))
	for _, r := range reactions {
		response = append(response, h.toReactionResponse(r))
	}

	c.JSON(http.StatusOK, ListReactionsResponse{Reactions: response})
}

// toReactionResponse converts a MessageReaction to ReactionResponse.
func (h *ReactionsHandler) toReactionResponse(r *models.MessageReaction) ReactionResponse {
	return ReactionResponse{
		ID:             r.ID,
		TenantID:       r.TenantID,
		ConversationID: r.ConversationID,
		MessageID:      r.MessageID,
		UserID:         r.UserID,
		Reaction:       r.Reaction,
		FeedbackText:   r.FeedbackText,
		CreatedAt:      r.CreatedAt,
		UpdatedAt:      r.UpdatedAt,
	}
}

// getUserID resolves the user ID from the auth token via the platform service.
func (h *ReactionsHandler) getUserID(ctx context.Context, authToken string) (string, error) {
	if h.platformClient == nil {
		return "system", nil
	}

	userInfo, err := h.platformClient.GetMe(ctx, authToken)
	if err != nil {
		return "system", nil //nolint:nilerr // graceful fallback to "system" user when platform is unavailable
	}

	return userInfo.ID, nil
}
