// Package handlers provides HTTP handlers for the API.
package handlers

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/bson"

	"github.com/unifiedui/agent-service/internal/api/middleware"
	"github.com/unifiedui/agent-service/internal/core/docdb"
	"github.com/unifiedui/agent-service/internal/domain/errors"
	"github.com/unifiedui/agent-service/internal/domain/models"
)

// TracesHandler handles trace-related endpoints.
type TracesHandler struct {
	docDBClient docdb.Client
}

// NewTracesHandler creates a new TracesHandler.
func NewTracesHandler(docDBClient docdb.Client) *TracesHandler {
	return &TracesHandler{
		docDBClient: docDBClient,
	}
}

// GetMessageTracesResponse represents the response for getting message traces.
type GetMessageTracesResponse struct {
	Traces []*models.Trace `json:"traces"`
	Total  int64           `json:"total"`
}

// GetMessageTraces handles GET /tenants/{tenantId}/conversations/{conversationId}/messages/{messageId}/traces
// @Summary Get message traces
// @Description Retrieves all traces associated with a specific message
// @Tags Traces
// @Accept json
// @Produce json
// @Param tenantId path string true "Tenant ID"
// @Param conversationId path string true "Conversation ID"
// @Param messageId path string true "Message ID"
// @Success 200 {object} GetMessageTracesResponse
// @Failure 400 {object} dto.ErrorResponse
// @Failure 401 {object} dto.ErrorResponse
// @Failure 500 {object} dto.ErrorResponse
// @Security BearerAuth
// @Router /api/v1/agent-service/tenants/{tenantId}/conversations/{conversationId}/messages/{messageId}/traces [get]
func (h *TracesHandler) GetMessageTraces(c *gin.Context) {
	ctx := c.Request.Context()
	tenantCtx := middleware.GetTenantContext(c)

	// Build filter
	filter := bson.M{
		"tenantId":       tenantCtx.TenantID,
		"conversationId": tenantCtx.ConversationID,
		"messageId":      tenantCtx.MessageID,
	}

	// Get total count
	total, err := h.docDBClient.Traces().CountDocuments(ctx, filter)
	if err != nil {
		middleware.HandleError(c, errors.NewInternalError("failed to count traces", err))
		return
	}

	// Get traces
	opts := &docdb.FindOptions{
		Sort: bson.M{"startedAt": 1},
	}

	cursor, err := h.docDBClient.Traces().Find(ctx, filter, opts)
	if err != nil {
		middleware.HandleError(c, errors.NewInternalError("failed to find traces", err))
		return
	}
	defer cursor.Close(ctx)

	var traces []*models.Trace
	if err := cursor.All(ctx, &traces); err != nil {
		middleware.HandleError(c, errors.NewInternalError("failed to decode traces", err))
		return
	}

	c.JSON(http.StatusOK, GetMessageTracesResponse{
		Traces: traces,
		Total:  total,
	})
}

// UpdateTracesRequest represents the request body for updating traces.
type UpdateTracesRequest struct {
	Traces []*TraceUpdate `json:"traces" binding:"required,min=1"`
}

// TraceUpdate represents a single trace update.
type TraceUpdate struct {
	TraceID        string                `json:"traceId" binding:"required"`
	ConversationID string                `json:"conversationId" binding:"required"`
	MessageID      string                `json:"messageId" binding:"required"`
	ParentTraceID  string                `json:"parentTraceId,omitempty"`
	Type           string                `json:"type" binding:"required"`
	Name           string                `json:"name" binding:"required"`
	Status         string                `json:"status" binding:"required"`
	Input          interface{}           `json:"input,omitempty"`
	Output         interface{}           `json:"output,omitempty"`
	Error          string                `json:"error,omitempty"`
	StartedAt      *time.Time            `json:"startedAt,omitempty"`
	EndedAt        *time.Time            `json:"endedAt,omitempty"`
	DurationMs     int64                 `json:"durationMs,omitempty"`
	Metadata       *models.TraceMetadata `json:"metadata,omitempty"`
}

// UpdateTracesResponse represents the response for updating traces.
type UpdateTracesResponse struct {
	Updated int `json:"updated"`
	Created int `json:"created"`
}

// UpdateTraces handles PUT /tenants/{tenantId}/autonomous-agents/{agentId}/traces
// @Summary Update traces
// @Description Called by autonomous agents to report their trace data
// @Tags Traces
// @Accept json
// @Produce json
// @Param tenantId path string true "Tenant ID"
// @Param agentId path string true "Agent ID"
// @Param request body UpdateTracesRequest true "Trace updates"
// @Success 200 {object} UpdateTracesResponse
// @Failure 400 {object} dto.ErrorResponse
// @Failure 401 {object} dto.ErrorResponse
// @Failure 500 {object} dto.ErrorResponse
// @Security BearerAuth
// @Router /api/v1/agent-service/tenants/{tenantId}/autonomous-agents/{agentId}/traces [put]
func (h *TracesHandler) UpdateTraces(c *gin.Context) {
	ctx := c.Request.Context()
	tenantCtx := middleware.GetTenantContext(c)

	// Parse request body
	var req UpdateTracesRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		middleware.HandleError(c, errors.NewValidationError("invalid request body", err.Error()))
		return
	}

	updated := 0
	created := 0

	for _, traceUpdate := range req.Traces {
		// Check if trace exists
		existingFilter := bson.M{
			"_id":      traceUpdate.TraceID,
			"tenantId": tenantCtx.TenantID,
			"agentId":  tenantCtx.AgentID,
		}

		existingResult := h.docDBClient.Traces().FindOne(ctx, existingFilter)
		if existingResult.Err() == nil {
			// Update existing trace
			update := bson.M{
				"$set": bson.M{
					"status":     traceUpdate.Status,
					"output":     traceUpdate.Output,
					"error":      traceUpdate.Error,
					"endedAt":    traceUpdate.EndedAt,
					"durationMs": traceUpdate.DurationMs,
					"metadata":   traceUpdate.Metadata,
				},
			}

			if _, err := h.docDBClient.Traces().UpdateOne(ctx, existingFilter, update); err != nil {
				middleware.HandleError(c, errors.NewInternalError("failed to update trace", err))
				return
			}
			updated++
		} else {
			// Create new trace
			trace := &models.Trace{
				ID:             traceUpdate.TraceID,
				TenantID:       tenantCtx.TenantID,
				ConversationID: traceUpdate.ConversationID,
				MessageID:      traceUpdate.MessageID,
				AgentID:        tenantCtx.AgentID,
				ParentTraceID:  traceUpdate.ParentTraceID,
				Type:           models.TraceType(traceUpdate.Type),
				Name:           traceUpdate.Name,
				Status:         models.TraceStatus(traceUpdate.Status),
				Input:          traceUpdate.Input,
				Output:         traceUpdate.Output,
				Error:          traceUpdate.Error,
				DurationMs:     traceUpdate.DurationMs,
			}

			if traceUpdate.StartedAt != nil {
				trace.StartedAt = *traceUpdate.StartedAt
			} else {
				trace.StartedAt = time.Now().UTC()
			}

			if traceUpdate.EndedAt != nil {
				trace.EndedAt = traceUpdate.EndedAt
			}

			if traceUpdate.Metadata != nil {
				trace.Metadata = *traceUpdate.Metadata
			}

			if _, err := h.docDBClient.Traces().InsertOne(ctx, trace); err != nil {
				middleware.HandleError(c, errors.NewInternalError("failed to create trace", err))
				return
			}
			created++
		}
	}

	c.JSON(http.StatusOK, UpdateTracesResponse{
		Updated: updated,
		Created: created,
	})
}
