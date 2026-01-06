// Package handlers provides HTTP handlers for the API.
package handlers

import (
	"context"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/unifiedui/agent-service/internal/api/dto"
	"github.com/unifiedui/agent-service/internal/api/middleware"
	"github.com/unifiedui/agent-service/internal/core/docdb"
	"github.com/unifiedui/agent-service/internal/domain/errors"
	"github.com/unifiedui/agent-service/internal/domain/models"
	"github.com/unifiedui/agent-service/internal/services/platform"
)

// TracesHandler handles trace-related endpoints.
type TracesHandler struct {
	docDBClient    docdb.Client
	platformClient platform.Client
}

// NewTracesHandler creates a new TracesHandler.
func NewTracesHandler(docDBClient docdb.Client, platformClient platform.Client) *TracesHandler {
	return &TracesHandler{
		docDBClient:    docDBClient,
		platformClient: platformClient,
	}
}

// CreateTrace handles POST /tenants/{tenantId}/traces
// @Summary Create a new trace
// @Description Creates a new trace for a conversation or autonomous agent
// @Tags Traces
// @Accept json
// @Produce json
// @Param tenantId path string true "Tenant ID"
// @Param request body dto.CreateTraceRequest true "Trace creation request"
// @Success 201 {object} dto.CreateTraceResponse
// @Failure 400 {object} dto.ErrorResponse "Bad request - validation error"
// @Failure 401 {object} dto.ErrorResponse "Unauthorized"
// @Failure 404 {object} dto.ErrorResponse "Application, Conversation, or AutonomousAgent not found"
// @Failure 500 {object} dto.ErrorResponse "Internal server error"
// @Security BearerAuth
// @Router /api/v1/agent-service/tenants/{tenantId}/traces [post]
func (h *TracesHandler) CreateTrace(c *gin.Context) {
	ctx := c.Request.Context()
	tenantID := c.Param("tenantId")
	authToken := middleware.GetToken(c)

	// Parse request body
	var req dto.CreateTraceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		middleware.HandleError(c, errors.NewValidationError("invalid request body", err.Error()))
		return
	}

	// Validate context: either (applicationId + conversationId) OR autonomousAgentId
	hasConversationContext := req.ApplicationID != "" && req.ConversationID != ""
	hasAgentContext := req.AutonomousAgentID != ""

	if hasConversationContext && hasAgentContext {
		middleware.HandleError(c, errors.NewValidationError(
			"invalid context",
			"cannot specify both conversation context and autonomous agent context",
		))
		return
	}

	if !hasConversationContext && !hasAgentContext {
		middleware.HandleError(c, errors.NewValidationError(
			"missing context",
			"must specify either (applicationId + conversationId) or autonomousAgentId",
		))
		return
	}

	// Get user info from platform service for created_by
	userID, err := h.getUserID(ctx, authToken)
	if err != nil {
		middleware.HandleError(c, errors.NewInternalError("failed to get user info", err))
		return
	}

	// Validate context with platform service
	if hasConversationContext {
		if err := h.validateConversationContext(ctx, tenantID, req.ApplicationID, req.ConversationID, authToken); err != nil {
			middleware.HandleError(c, err)
			return
		}
	} else {
		if err := h.validateAutonomousAgentContext(ctx, tenantID, req.AutonomousAgentID, authToken); err != nil {
			middleware.HandleError(c, err)
			return
		}
	}

	// Generate ID if not provided
	traceID := req.ID
	if traceID == "" {
		traceID = uuid.New().String()
	}

	// Create trace model
	now := time.Now().UTC()
	trace := &models.Trace{
		ID:                traceID,
		TenantID:          tenantID,
		ReferenceID:       req.ReferenceID,
		ReferenceName:     req.ReferenceName,
		ReferenceMetadata: req.ReferenceMetadata,
		Logs:              models.ConvertLogsToStrings(req.Logs),
		CreatedAt:         now,
		UpdatedAt:         now,
		CreatedBy:         userID,
		UpdatedBy:         userID,
	}

	if hasConversationContext {
		trace.ApplicationID = req.ApplicationID
		trace.ConversationID = req.ConversationID
		trace.ContextType = models.TraceContextConversation
	} else {
		trace.AutonomousAgentID = req.AutonomousAgentID
		trace.ContextType = models.TraceContextAutonomousAgent
	}

	// Convert and add nodes
	if len(req.Nodes) > 0 {
		trace.Nodes = dto.ConvertNodesToModel(req.Nodes, userID)
	} else {
		trace.Nodes = []models.TraceNode{}
	}

	if trace.Logs == nil {
		trace.Logs = []string{}
	}

	// Insert trace
	if err := h.docDBClient.Traces().Create(ctx, trace); err != nil {
		middleware.HandleError(c, errors.NewInternalError("failed to create trace", err))
		return
	}

	c.JSON(http.StatusCreated, dto.CreateTraceResponse{
		ID: trace.ID,
	})
}

// AddNodes handles POST /tenants/{tenantId}/traces/{traceId}/nodes
// @Summary Add nodes to a trace
// @Description Appends nodes to an existing trace
// @Tags Traces
// @Accept json
// @Produce json
// @Param tenantId path string true "Tenant ID"
// @Param traceId path string true "Trace ID"
// @Param request body dto.AddNodesRequest true "Nodes to add"
// @Success 200 {object} map[string]string "Success"
// @Failure 400 {object} dto.ErrorResponse "Bad request"
// @Failure 401 {object} dto.ErrorResponse "Unauthorized"
// @Failure 404 {object} dto.ErrorResponse "Trace not found"
// @Failure 500 {object} dto.ErrorResponse "Internal server error"
// @Security BearerAuth
// @Router /api/v1/agent-service/tenants/{tenantId}/traces/{traceId}/nodes [post]
func (h *TracesHandler) AddNodes(c *gin.Context) {
	ctx := c.Request.Context()
	tenantID := c.Param("tenantId")
	traceID := c.Param("traceId")
	authToken := middleware.GetToken(c)

	// Parse request body
	var req dto.AddNodesRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		middleware.HandleError(c, errors.NewValidationError("invalid request body", err.Error()))
		return
	}

	// Get trace to verify it exists and belongs to tenant
	trace, err := h.docDBClient.Traces().Get(ctx, traceID)
	if err != nil {
		middleware.HandleError(c, errors.NewInternalError("failed to get trace", err))
		return
	}
	if trace == nil || trace.TenantID != tenantID {
		middleware.HandleError(c, errors.NewNotFoundError("trace", traceID))
		return
	}

	// Get user info for updated_by
	userID, err := h.getUserID(ctx, authToken)
	if err != nil {
		middleware.HandleError(c, errors.NewInternalError("failed to get user info", err))
		return
	}

	// Convert nodes
	nodes := dto.ConvertNodesToModel(req.Nodes, userID)

	// Add nodes to trace
	if err := h.docDBClient.Traces().AddNodes(ctx, traceID, nodes); err != nil {
		middleware.HandleError(c, errors.NewInternalError("failed to add nodes", err))
		return
	}

	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

// AddLogs handles POST /tenants/{tenantId}/traces/{traceId}/logs
// @Summary Add logs to a trace
// @Description Appends logs to an existing trace
// @Tags Traces
// @Accept json
// @Produce json
// @Param tenantId path string true "Tenant ID"
// @Param traceId path string true "Trace ID"
// @Param request body dto.AddLogsRequest true "Logs to add"
// @Success 200 {object} map[string]string "Success"
// @Failure 400 {object} dto.ErrorResponse "Bad request"
// @Failure 401 {object} dto.ErrorResponse "Unauthorized"
// @Failure 404 {object} dto.ErrorResponse "Trace not found"
// @Failure 500 {object} dto.ErrorResponse "Internal server error"
// @Security BearerAuth
// @Router /api/v1/agent-service/tenants/{tenantId}/traces/{traceId}/logs [post]
func (h *TracesHandler) AddLogs(c *gin.Context) {
	ctx := c.Request.Context()
	tenantID := c.Param("tenantId")
	traceID := c.Param("traceId")

	// Parse request body
	var req dto.AddLogsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		middleware.HandleError(c, errors.NewValidationError("invalid request body", err.Error()))
		return
	}

	// Get trace to verify it exists and belongs to tenant
	trace, err := h.docDBClient.Traces().Get(ctx, traceID)
	if err != nil {
		middleware.HandleError(c, errors.NewInternalError("failed to get trace", err))
		return
	}
	if trace == nil || trace.TenantID != tenantID {
		middleware.HandleError(c, errors.NewNotFoundError("trace", traceID))
		return
	}

	// Add logs to trace
	if err := h.docDBClient.Traces().AddLogs(ctx, traceID, models.ConvertLogsToStrings(req.Logs)); err != nil {
		middleware.HandleError(c, errors.NewInternalError("failed to add logs", err))
		return
	}

	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

// GetConversationTrace handles GET /tenants/{tenantId}/conversations/{conversationId}/traces
// @Summary Get trace for a conversation
// @Description Retrieves the trace for a specific conversation
// @Tags Traces
// @Accept json
// @Produce json
// @Param tenantId path string true "Tenant ID"
// @Param conversationId path string true "Conversation ID"
// @Success 200 {object} dto.TraceResponse
// @Failure 401 {object} dto.ErrorResponse "Unauthorized"
// @Failure 404 {object} dto.ErrorResponse "Trace not found"
// @Failure 500 {object} dto.ErrorResponse "Internal server error"
// @Security BearerAuth
// @Router /api/v1/agent-service/tenants/{tenantId}/conversations/{conversationId}/traces [get]
func (h *TracesHandler) GetConversationTrace(c *gin.Context) {
	ctx := c.Request.Context()
	tenantID := c.Param("tenantId")
	conversationID := c.Param("conversationId")

	trace, err := h.docDBClient.Traces().GetByConversation(ctx, tenantID, conversationID)
	if err != nil {
		middleware.HandleError(c, errors.NewInternalError("failed to get trace", err))
		return
	}
	if trace == nil {
		middleware.HandleError(c, errors.NewNotFoundError("trace", conversationID))
		return
	}

	c.JSON(http.StatusOK, dto.TraceToResponse(trace))
}

// RefreshConversationTrace handles PUT /tenants/{tenantId}/conversations/{conversationId}/traces
// @Summary Refresh trace for a conversation
// @Description Replaces the trace for a specific conversation completely
// @Tags Traces
// @Accept json
// @Produce json
// @Param tenantId path string true "Tenant ID"
// @Param conversationId path string true "Conversation ID"
// @Param request body dto.RefreshTraceRequest true "New trace data"
// @Success 200 {object} dto.TraceResponse
// @Failure 400 {object} dto.ErrorResponse "Bad request"
// @Failure 401 {object} dto.ErrorResponse "Unauthorized"
// @Failure 404 {object} dto.ErrorResponse "Trace not found"
// @Failure 500 {object} dto.ErrorResponse "Internal server error"
// @Security BearerAuth
// @Router /api/v1/agent-service/tenants/{tenantId}/conversations/{conversationId}/traces [put]
func (h *TracesHandler) RefreshConversationTrace(c *gin.Context) {
	ctx := c.Request.Context()
	tenantID := c.Param("tenantId")
	conversationID := c.Param("conversationId")
	authToken := middleware.GetToken(c)

	// Parse request body
	var req dto.RefreshTraceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		middleware.HandleError(c, errors.NewValidationError("invalid request body", err.Error()))
		return
	}

	// Get existing trace
	trace, err := h.docDBClient.Traces().GetByConversation(ctx, tenantID, conversationID)
	if err != nil {
		middleware.HandleError(c, errors.NewInternalError("failed to get trace", err))
		return
	}
	if trace == nil {
		middleware.HandleError(c, errors.NewNotFoundError("trace", conversationID))
		return
	}

	// Get user info for updated_by
	userID, err := h.getUserID(ctx, authToken)
	if err != nil {
		middleware.HandleError(c, errors.NewInternalError("failed to get user info", err))
		return
	}

	// Update trace fields
	trace.ReferenceID = req.ReferenceID
	trace.ReferenceName = req.ReferenceName
	trace.ReferenceMetadata = req.ReferenceMetadata
	trace.Logs = models.ConvertLogsToStrings(req.Logs)
	trace.Nodes = dto.ConvertNodesToModel(req.Nodes, userID)
	trace.UpdatedAt = time.Now().UTC()
	trace.UpdatedBy = userID

	if trace.Logs == nil {
		trace.Logs = []string{}
	}

	// Update trace
	if err := h.docDBClient.Traces().Update(ctx, trace); err != nil {
		middleware.HandleError(c, errors.NewInternalError("failed to update trace", err))
		return
	}

	c.JSON(http.StatusOK, dto.TraceToResponse(trace))
}

// GetAutonomousAgentTrace handles GET /tenants/{tenantId}/autonomous-agents/{agentId}/traces
// @Summary Get trace for an autonomous agent
// @Description Retrieves the trace for a specific autonomous agent
// @Tags Traces
// @Accept json
// @Produce json
// @Param tenantId path string true "Tenant ID"
// @Param agentId path string true "Autonomous Agent ID"
// @Success 200 {object} dto.TraceResponse
// @Failure 401 {object} dto.ErrorResponse "Unauthorized"
// @Failure 404 {object} dto.ErrorResponse "Trace not found"
// @Failure 500 {object} dto.ErrorResponse "Internal server error"
// @Security BearerAuth
// @Router /api/v1/agent-service/tenants/{tenantId}/autonomous-agents/{agentId}/traces [get]
func (h *TracesHandler) GetAutonomousAgentTrace(c *gin.Context) {
	ctx := c.Request.Context()
	tenantID := c.Param("tenantId")
	agentID := c.Param("agentId")

	trace, err := h.docDBClient.Traces().GetByAutonomousAgent(ctx, tenantID, agentID)
	if err != nil {
		middleware.HandleError(c, errors.NewInternalError("failed to get trace", err))
		return
	}
	if trace == nil {
		middleware.HandleError(c, errors.NewNotFoundError("trace", agentID))
		return
	}

	c.JSON(http.StatusOK, dto.TraceToResponse(trace))
}

// RefreshAutonomousAgentTrace handles PUT /tenants/{tenantId}/autonomous-agents/{agentId}/traces
// @Summary Refresh trace for an autonomous agent
// @Description Replaces the trace for a specific autonomous agent completely
// @Tags Traces
// @Accept json
// @Produce json
// @Param tenantId path string true "Tenant ID"
// @Param agentId path string true "Autonomous Agent ID"
// @Param request body dto.RefreshTraceRequest true "New trace data"
// @Success 200 {object} dto.TraceResponse
// @Failure 400 {object} dto.ErrorResponse "Bad request"
// @Failure 401 {object} dto.ErrorResponse "Unauthorized"
// @Failure 404 {object} dto.ErrorResponse "Trace not found"
// @Failure 500 {object} dto.ErrorResponse "Internal server error"
// @Security BearerAuth
// @Router /api/v1/agent-service/tenants/{tenantId}/autonomous-agents/{agentId}/traces [put]
func (h *TracesHandler) RefreshAutonomousAgentTrace(c *gin.Context) {
	ctx := c.Request.Context()
	tenantID := c.Param("tenantId")
	agentID := c.Param("agentId")
	authToken := middleware.GetToken(c)

	// Parse request body
	var req dto.RefreshTraceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		middleware.HandleError(c, errors.NewValidationError("invalid request body", err.Error()))
		return
	}

	// Get existing trace
	trace, err := h.docDBClient.Traces().GetByAutonomousAgent(ctx, tenantID, agentID)
	if err != nil {
		middleware.HandleError(c, errors.NewInternalError("failed to get trace", err))
		return
	}
	if trace == nil {
		middleware.HandleError(c, errors.NewNotFoundError("trace", agentID))
		return
	}

	// Get user info for updated_by
	userID, err := h.getUserID(ctx, authToken)
	if err != nil {
		middleware.HandleError(c, errors.NewInternalError("failed to get user info", err))
		return
	}

	// Update trace fields
	trace.ReferenceID = req.ReferenceID
	trace.ReferenceName = req.ReferenceName
	trace.ReferenceMetadata = req.ReferenceMetadata
	trace.Logs = models.ConvertLogsToStrings(req.Logs)
	trace.Nodes = dto.ConvertNodesToModel(req.Nodes, userID)
	trace.UpdatedAt = time.Now().UTC()
	trace.UpdatedBy = userID

	if trace.Logs == nil {
		trace.Logs = []string{}
	}

	// Update trace
	if err := h.docDBClient.Traces().Update(ctx, trace); err != nil {
		middleware.HandleError(c, errors.NewInternalError("failed to update trace", err))
		return
	}

	c.JSON(http.StatusOK, dto.TraceToResponse(trace))
}

// ListAutonomousAgentTraces handles GET /tenants/{tenantId}/autonomous-agents/traces
// @Summary List traces for autonomous agents
// @Description Retrieves a list of traces for autonomous agents with pagination
// @Tags Traces
// @Accept json
// @Produce json
// @Param tenantId path string true "Tenant ID"
// @Param autonomousAgentId query string false "Filter by autonomous agent ID"
// @Param limit query int false "Maximum number of results (default: 20, max: 100)"
// @Param skip query int false "Number of results to skip"
// @Param order query string false "Sort order: asc or desc (default: desc)"
// @Success 200 {object} dto.ListTracesResponse
// @Failure 400 {object} dto.ErrorResponse "Bad request"
// @Failure 401 {object} dto.ErrorResponse "Unauthorized"
// @Failure 500 {object} dto.ErrorResponse "Internal server error"
// @Security BearerAuth
// @Router /api/v1/agent-service/tenants/{tenantId}/autonomous-agents/traces [get]
func (h *TracesHandler) ListAutonomousAgentTraces(c *gin.Context) {
	ctx := c.Request.Context()
	tenantID := c.Param("tenantId")

	// Parse query parameters
	autonomousAgentID := c.Query("autonomousAgentId")
	limitStr := c.DefaultQuery("limit", "20")
	skipStr := c.DefaultQuery("skip", "0")
	order := c.DefaultQuery("order", "desc")

	limit, err := strconv.ParseInt(limitStr, 10, 64)
	if err != nil || limit < 1 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}

	skip, err := strconv.ParseInt(skipStr, 10, 64)
	if err != nil || skip < 0 {
		skip = 0
	}

	sortOrder := docdb.SortOrderDesc
	if order == "asc" {
		sortOrder = docdb.SortOrderAsc
	}

	// Build list options
	opts := &docdb.ListTracesOptions{
		TenantID:          tenantID,
		AutonomousAgentID: autonomousAgentID,
		ContextType:       models.TraceContextAutonomousAgent,
		Limit:             limit,
		Skip:              skip,
		OrderBy:           sortOrder,
	}

	traces, err := h.docDBClient.Traces().List(ctx, opts)
	if err != nil {
		middleware.HandleError(c, errors.NewInternalError("failed to list traces", err))
		return
	}

	c.JSON(http.StatusOK, dto.ListTracesResponse{
		Traces: dto.TracesToResponse(traces),
	})
}

// GetTrace handles GET /tenants/{tenantId}/traces/{traceId}
// @Summary Get a trace by ID
// @Description Retrieves a specific trace by its ID
// @Tags Traces
// @Accept json
// @Produce json
// @Param tenantId path string true "Tenant ID"
// @Param traceId path string true "Trace ID"
// @Success 200 {object} dto.TraceResponse
// @Failure 401 {object} dto.ErrorResponse "Unauthorized"
// @Failure 404 {object} dto.ErrorResponse "Trace not found"
// @Failure 500 {object} dto.ErrorResponse "Internal server error"
// @Security BearerAuth
// @Router /api/v1/agent-service/tenants/{tenantId}/traces/{traceId} [get]
func (h *TracesHandler) GetTrace(c *gin.Context) {
	ctx := c.Request.Context()
	tenantID := c.Param("tenantId")
	traceID := c.Param("traceId")

	trace, err := h.docDBClient.Traces().Get(ctx, traceID)
	if err != nil {
		middleware.HandleError(c, errors.NewInternalError("failed to get trace", err))
		return
	}
	if trace == nil || trace.TenantID != tenantID {
		middleware.HandleError(c, errors.NewNotFoundError("trace", traceID))
		return
	}

	c.JSON(http.StatusOK, dto.TraceToResponse(trace))
}

// DeleteTrace handles DELETE /tenants/{tenantId}/traces/{traceId}
// @Summary Delete a trace
// @Description Deletes a specific trace by its ID
// @Tags Traces
// @Accept json
// @Produce json
// @Param tenantId path string true "Tenant ID"
// @Param traceId path string true "Trace ID"
// @Success 204 "No Content"
// @Failure 401 {object} dto.ErrorResponse "Unauthorized"
// @Failure 404 {object} dto.ErrorResponse "Trace not found"
// @Failure 500 {object} dto.ErrorResponse "Internal server error"
// @Security BearerAuth
// @Router /api/v1/agent-service/tenants/{tenantId}/traces/{traceId} [delete]
func (h *TracesHandler) DeleteTrace(c *gin.Context) {
	ctx := c.Request.Context()
	tenantID := c.Param("tenantId")
	traceID := c.Param("traceId")

	// Verify trace exists and belongs to tenant
	trace, err := h.docDBClient.Traces().Get(ctx, traceID)
	if err != nil {
		middleware.HandleError(c, errors.NewInternalError("failed to get trace", err))
		return
	}
	if trace == nil || trace.TenantID != tenantID {
		middleware.HandleError(c, errors.NewNotFoundError("trace", traceID))
		return
	}

	if err := h.docDBClient.Traces().Delete(ctx, traceID); err != nil {
		middleware.HandleError(c, errors.NewInternalError("failed to delete trace", err))
		return
	}

	c.Status(http.StatusNoContent)
}

// --- Helper Methods ---

// getUserID retrieves the user ID from the platform service.
// The identity/me endpoint doesn't require tenantId.
func (h *TracesHandler) getUserID(ctx context.Context, authToken string) (string, error) {
	if h.platformClient == nil {
		// Fallback for when platform client is not configured
		return "system", nil
	}

	// Use GetMe endpoint from platform service
	userInfo, err := h.platformClient.GetMe(ctx, authToken)
	if err != nil {
		// Fallback to "system" if we can't get user info
		return "system", nil
	}

	return userInfo.ID, nil
}

// validateConversationContext validates that the application and conversation exist.
func (h *TracesHandler) validateConversationContext(ctx context.Context, tenantID, applicationID, conversationID, authToken string) *errors.DomainError {
	if h.platformClient == nil {
		// Skip validation if platform client is not configured
		return nil
	}

	// Validate by fetching the conversation (which also validates the application)
	if err := h.platformClient.ValidateConversation(ctx, tenantID, conversationID, authToken); err != nil {
		errStr := err.Error()
		if len(errStr) > 12 && errStr[:12] == "unauthorized" {
			return errors.NewUnauthorizedError("invalid or expired token")
		}
		if len(errStr) > 9 && errStr[:9] == "forbidden" {
			return errors.NewForbiddenError("access denied to conversation")
		}
		if len(errStr) > 9 && errStr[:9] == "not_found" {
			return errors.NewNotFoundError("conversation", conversationID)
		}
		return errors.NewInternalError("failed to validate conversation", err)
	}

	return nil
}

// validateAutonomousAgentContext validates that the autonomous agent exists.
func (h *TracesHandler) validateAutonomousAgentContext(ctx context.Context, tenantID, autonomousAgentID, authToken string) *errors.DomainError {
	if h.platformClient == nil {
		// Skip validation if platform client is not configured
		return nil
	}

	if err := h.platformClient.ValidateAutonomousAgent(ctx, tenantID, autonomousAgentID, authToken); err != nil {
		errStr := err.Error()
		if len(errStr) > 12 && errStr[:12] == "unauthorized" {
			return errors.NewUnauthorizedError("invalid or expired token")
		}
		if len(errStr) > 9 && errStr[:9] == "forbidden" {
			return errors.NewForbiddenError("access denied to autonomous agent")
		}
		if len(errStr) > 9 && errStr[:9] == "not_found" {
			return errors.NewNotFoundError("autonomous agent", autonomousAgentID)
		}
		return errors.NewInternalError("failed to validate autonomous agent", err)
	}

	return nil
}
