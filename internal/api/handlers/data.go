package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/unifiedui/agent-service/internal/api/middleware"
	"github.com/unifiedui/agent-service/internal/core/docdb"
	"github.com/unifiedui/agent-service/internal/domain/errors"
)

// DataHandler handles bulk data deletion for cascade operations from platform service.
type DataHandler struct {
	docDBClient docdb.Client
}

// NewDataHandler creates a new DataHandler.
func NewDataHandler(docDBClient docdb.Client) *DataHandler {
	return &DataHandler{
		docDBClient: docDBClient,
	}
}

// DeleteConversationData handles DELETE /tenants/{tenantId}/conversations/{conversationId}/data
// @Summary Delete all data for a conversation
// @Description Deletes all messages and traces associated with a conversation. Requires X-Service-Key authentication.
// @Tags Data
// @Produce json
// @Param tenantId path string true "Tenant ID"
// @Param conversationId path string true "Conversation ID"
// @Param X-Service-Key header string true "Service-to-service authentication key"
// @Success 204 "No Content"
// @Failure 401 {object} dto.ErrorResponse "Unauthorized"
// @Failure 403 {object} dto.ErrorResponse "Forbidden"
// @Failure 500 {object} dto.ErrorResponse "Internal server error"
// @Router /api/v1/agent-service/tenants/{tenantId}/conversations/{conversationId}/data [delete]
func (h *DataHandler) DeleteConversationData(c *gin.Context) {
	ctx := c.Request.Context()
	tenantID := middleware.SanitizePathParam(c, "tenantId")
	conversationID := middleware.SanitizePathParam(c, "conversationId")

	if tenantID == "" || conversationID == "" {
		middleware.HandleError(c, errors.NewValidationError(
			"missing path parameters", "tenantId and conversationId are required",
		))
		return
	}

	_, messagesErr := h.docDBClient.Messages().Delete(ctx, &docdb.DeleteMessagesOptions{
		ConversationID: conversationID,
		TenantID:       tenantID,
	})
	if messagesErr != nil {
		middleware.HandleError(c, errors.NewInternalError(
			"failed to delete conversation messages", messagesErr,
		))
		return
	}

	tracesErr := h.docDBClient.Traces().DeleteByConversation(ctx, tenantID, conversationID)
	if tracesErr != nil {
		middleware.HandleError(c, errors.NewInternalError(
			"failed to delete conversation traces", tracesErr,
		))
		return
	}

	c.Status(http.StatusNoContent)
}

// DeleteAutonomousAgentData handles DELETE /tenants/{tenantId}/autonomous-agents/{agentId}/data
// @Summary Delete all data for an autonomous agent
// @Description Deletes all traces associated with an autonomous agent. Requires X-Service-Key authentication.
// @Tags Data
// @Produce json
// @Param tenantId path string true "Tenant ID"
// @Param agentId path string true "Autonomous Agent ID"
// @Param X-Service-Key header string true "Service-to-service authentication key"
// @Success 204 "No Content"
// @Failure 401 {object} dto.ErrorResponse "Unauthorized"
// @Failure 403 {object} dto.ErrorResponse "Forbidden"
// @Failure 500 {object} dto.ErrorResponse "Internal server error"
// @Router /api/v1/agent-service/tenants/{tenantId}/autonomous-agents/{agentId}/data [delete]
func (h *DataHandler) DeleteAutonomousAgentData(c *gin.Context) {
	ctx := c.Request.Context()
	tenantID := middleware.SanitizePathParam(c, "tenantId")
	agentID := middleware.SanitizePathParam(c, "agentId")

	if tenantID == "" || agentID == "" {
		middleware.HandleError(c, errors.NewValidationError(
			"missing path parameters", "tenantId and agentId are required",
		))
		return
	}

	tracesErr := h.docDBClient.Traces().DeleteByAutonomousAgent(ctx, tenantID, agentID)
	if tracesErr != nil {
		middleware.HandleError(c, errors.NewInternalError(
			"failed to delete autonomous agent traces", tracesErr,
		))
		return
	}

	c.Status(http.StatusNoContent)
}
