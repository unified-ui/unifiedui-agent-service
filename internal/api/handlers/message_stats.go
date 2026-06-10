package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/unifiedui/agent-service/internal/api/dto"
	"github.com/unifiedui/agent-service/internal/api/middleware"
	"github.com/unifiedui/agent-service/internal/core/docdb"
	"github.com/unifiedui/agent-service/internal/domain/models"
)

// MessageStatsHandler handles message statistics endpoints.
type MessageStatsHandler struct {
	docDBClient docdb.Client
}

// NewMessageStatsHandler creates a new MessageStatsHandler.
func NewMessageStatsHandler(docDBClient docdb.Client) *MessageStatsHandler {
	return &MessageStatsHandler{docDBClient: docDBClient}
}

// GetMessageStats returns aggregated message counts by status for a tenant,
// grouped by chat agent.
func (h *MessageStatsHandler) GetMessageStats(c *gin.Context) {
	tenantCtx := middleware.GetTenantContext(c)
	if tenantCtx == nil {
		middleware.HandleError(c, nil)
		return
	}

	var req dto.MessageStatsRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid query parameters"})
		return
	}

	from, to, err := req.ParseTimeRange()
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid date format, use RFC3339"})
		return
	}

	filter := &models.MessageStatsFilter{
		ChatAgentIDs: req.ParseChatAgentIDs(),
		From:         from,
		To:           to,
	}

	result, err := h.docDBClient.Messages().GetMessageStats(c.Request.Context(), tenantCtx.TenantID, filter)
	if err != nil {
		middleware.HandleError(c, err)
		return
	}

	perAgent := make([]dto.MessageStatsPerAgent, 0, len(result.PerAgent))
	for _, row := range result.PerAgent {
		perAgent = append(perAgent, dto.MessageStatsPerAgent{
			ChatAgentID:   row.ChatAgentID,
			TotalMessages: row.TotalMessages,
			SuccessCount:  row.SuccessCount,
			FailedCount:   row.FailedCount,
		})
	}

	c.JSON(http.StatusOK, dto.MessageStatsResponse{
		Aggregate: dto.MessageStatsAggregate{
			TotalMessages: result.Aggregate.TotalMessages,
			SuccessCount:  result.Aggregate.SuccessCount,
			FailedCount:   result.Aggregate.FailedCount,
		},
		PerAgent: perAgent,
	})
}
