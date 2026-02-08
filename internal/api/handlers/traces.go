// Package handlers provides HTTP handlers for the API.
package handlers

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/unifiedui/agent-service/internal/api/dto"
	"github.com/unifiedui/agent-service/internal/api/middleware"
	"github.com/unifiedui/agent-service/internal/core/docdb"
	"github.com/unifiedui/agent-service/internal/domain/errors"
	"github.com/unifiedui/agent-service/internal/domain/models"
	"github.com/unifiedui/agent-service/internal/services/platform"
	"github.com/unifiedui/agent-service/internal/services/traceimport"
)

// TracesHandler handles trace-related endpoints.
type TracesHandler struct {
	docDBClient    docdb.Client
	platformClient platform.Client
	importService  *traceimport.ImportService
}

// NewTracesHandler creates a new TracesHandler.
func NewTracesHandler(docDBClient docdb.Client, platformClient platform.Client, importService *traceimport.ImportService) *TracesHandler {
	return &TracesHandler{
		docDBClient:    docDBClient,
		platformClient: platformClient,
		importService:  importService,
	}
}

// GetImportService returns the import service for testing purposes.
func (h *TracesHandler) GetImportService() *traceimport.ImportService {
	return h.importService
}

// CreateTrace handles POST /tenants/{tenantId}/traces
// @Summary Create a new trace
// @Description Creates a new trace for a conversation or autonomous agent. Uses Bearer token for conversation context, API key for autonomous agent context.
// @Tags Traces
// @Accept json
// @Produce json
// @Param tenantId path string true "Tenant ID"
// @Param request body dto.CreateTraceRequest true "Trace creation request"
// @Param Authorization header string false "Bearer token (required for conversation traces)"
// @Param X-Unified-UI-Autonomous-Agent-API-Key header string false "API key (required for autonomous agent traces)"
// @Success 201 {object} dto.CreateTraceResponse
// @Failure 400 {object} dto.ErrorResponse "Bad request - validation error"
// @Failure 401 {object} dto.ErrorResponse "Unauthorized"
// @Failure 404 {object} dto.ErrorResponse "Application, Conversation, or AutonomousAgent not found"
// @Failure 500 {object} dto.ErrorResponse "Internal server error"
// @Router /api/v1/agent-service/tenants/{tenantId}/traces [post]
func (h *TracesHandler) CreateTrace(c *gin.Context) {
	ctx := c.Request.Context()
	tenantID := c.Param("tenantId")

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

	var userID string

	if hasConversationContext {
		// Conversation context: require Bearer token
		authToken := middleware.GetToken(c)
		if authToken == "" {
			middleware.HandleError(c, errors.NewUnauthorizedError("bearer token required for conversation traces"))
			return
		}

		uid, err := h.getUserID(ctx, authToken)
		if err != nil {
			middleware.HandleError(c, errors.NewInternalError("failed to get user info", err))
			return
		}
		userID = uid

		if err := h.validateConversationContext(ctx, tenantID, req.ApplicationID, req.ConversationID, authToken); err != nil {
			middleware.HandleError(c, err)
			return
		}

		// Check if trace already exists for this conversation (only one trace per conversation allowed)
		existingTrace, err := h.docDBClient.Traces().GetByConversation(ctx, tenantID, req.ConversationID)
		if err != nil {
			middleware.HandleError(c, errors.NewInternalError("failed to check existing trace", err))
			return
		}
		if existingTrace != nil {
			middleware.HandleError(c, errors.NewConflictError(
				"trace already exists",
				"a trace already exists for this conversation; use PUT to update it",
			))
			return
		}
	} else {
		// Autonomous agent context: require Bearer token or API key
		authToken := middleware.GetToken(c)
		apiKey := middleware.GetAutonomousAgentAPIKey(c)

		if authToken != "" {
			// Bearer token auth: validate access to autonomous agent via platform
			if domainErr := h.resolveAutonomousAgentFromBearer(ctx, tenantID, req.AutonomousAgentID, authToken); domainErr != nil {
				middleware.HandleError(c, domainErr)
				return
			}

			uid, err := h.getUserID(ctx, authToken)
			if err != nil {
				middleware.HandleError(c, errors.NewInternalError("failed to get user info", err))
				return
			}
			userID = uid
		} else if apiKey != "" {
			// API key auth: validate via platform (platform checks allow_api_keys)
			if domainErr := h.resolveUserIDFromAPIKey(ctx, tenantID, req.AutonomousAgentID, apiKey); domainErr != nil {
				middleware.HandleError(c, domainErr)
				return
			}
			userID = "autonomous-agent-" + req.AutonomousAgentID
		} else {
			middleware.HandleError(c, errors.NewUnauthorizedError("Bearer token or X-Unified-UI-Autonomous-Agent-API-Key header required for autonomous agent traces"))
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
// @Description Appends nodes to an existing trace. Uses Bearer token for conversation traces, API key for autonomous agent traces.
// @Tags Traces
// @Accept json
// @Produce json
// @Param tenantId path string true "Tenant ID"
// @Param traceId path string true "Trace ID"
// @Param request body dto.AddNodesRequest true "Nodes to add"
// @Param Authorization header string false "Bearer token (required for conversation traces)"
// @Param X-Unified-UI-Autonomous-Agent-API-Key header string false "API key (required for autonomous agent traces)"
// @Success 200 {object} map[string]string "Success"
// @Failure 400 {object} dto.ErrorResponse "Bad request"
// @Failure 401 {object} dto.ErrorResponse "Unauthorized"
// @Failure 404 {object} dto.ErrorResponse "Trace not found"
// @Failure 500 {object} dto.ErrorResponse "Internal server error"
// @Router /api/v1/agent-service/tenants/{tenantId}/traces/{traceId}/nodes [post]
func (h *TracesHandler) AddNodes(c *gin.Context) {
	ctx := c.Request.Context()
	tenantID := c.Param("tenantId")
	traceID := c.Param("traceId")

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

	// Resolve user ID based on auth type and trace context
	userID, domainErr := h.resolveUserIDForTrace(ctx, c, tenantID, trace)
	if domainErr != nil {
		middleware.HandleError(c, domainErr)
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
// @Description Appends logs to an existing trace. Accepts Bearer token or API key authentication.
// @Tags Traces
// @Accept json
// @Produce json
// @Param tenantId path string true "Tenant ID"
// @Param traceId path string true "Trace ID"
// @Param request body dto.AddLogsRequest true "Logs to add"
// @Param Authorization header string false "Bearer token"
// @Param X-Unified-UI-Autonomous-Agent-API-Key header string false "API key"
// @Success 200 {object} map[string]string "Success"
// @Failure 400 {object} dto.ErrorResponse "Bad request"
// @Failure 401 {object} dto.ErrorResponse "Unauthorized"
// @Failure 404 {object} dto.ErrorResponse "Trace not found"
// @Failure 500 {object} dto.ErrorResponse "Internal server error"
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

// GetConversationTraces handles GET /tenants/{tenantId}/conversations/{conversationId}/traces
// @Summary List traces for a conversation
// @Description Retrieves all traces for a specific conversation
// @Tags Traces
// @Accept json
// @Produce json
// @Param tenantId path string true "Tenant ID"
// @Param conversationId path string true "Conversation ID"
// @Success 200 {object} dto.ListTracesResponse
// @Failure 401 {object} dto.ErrorResponse "Unauthorized"
// @Failure 500 {object} dto.ErrorResponse "Internal server error"
// @Security BearerAuth
// @Router /api/v1/agent-service/tenants/{tenantId}/conversations/{conversationId}/traces [get]
func (h *TracesHandler) GetConversationTraces(c *gin.Context) {
	ctx := c.Request.Context()
	tenantID := c.Param("tenantId")
	conversationID := c.Param("conversationId")

	traces, err := h.docDBClient.Traces().ListByConversation(ctx, tenantID, conversationID)
	if err != nil {
		middleware.HandleError(c, errors.NewInternalError("failed to list traces", err))
		return
	}

	c.JSON(http.StatusOK, dto.ListTracesResponse{
		Traces: dto.TracesToResponse(traces),
	})
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

// GetAutonomousAgentTraces handles GET /tenants/{tenantId}/autonomous-agents/{agentId}/traces
// @Summary List traces for an autonomous agent
// @Description Retrieves traces for a specific autonomous agent with pagination, sorting, and filtering
// @Tags Traces
// @Accept json
// @Produce json
// @Param tenantId path string true "Tenant ID"
// @Param agentId path string true "Autonomous Agent ID"
// @Param limit query int false "Maximum number of results (default: 20, max: 100)"
// @Param skip query int false "Number of results to skip (default: 0)"
// @Param order query string false "Sort order: asc or desc (default: desc)"
// @Param order_by query string false "Sort field: created_at or updated_at (default: created_at)"
// @Param created_after query string false "Filter: traces created after this ISO 8601 timestamp"
// @Param created_before query string false "Filter: traces created before this ISO 8601 timestamp"
// @Param expand query bool false "Include nodes and logs in response (default: false)"
// @Success 200 {object} dto.ListTracesResponse
// @Failure 400 {object} dto.ErrorResponse "Bad request"
// @Failure 401 {object} dto.ErrorResponse "Unauthorized"
// @Failure 500 {object} dto.ErrorResponse "Internal server error"
// @Security BearerAuth
// @Router /api/v1/agent-service/tenants/{tenantId}/autonomous-agents/{agentId}/traces [get]
func (h *TracesHandler) GetAutonomousAgentTraces(c *gin.Context) {
	ctx := c.Request.Context()
	tenantID := c.Param("tenantId")
	agentID := c.Param("agentId")

	opts, err := h.parseListTracesQueryParams(c)
	if err != nil {
		middleware.HandleError(c, err)
		return
	}
	opts.TenantID = tenantID
	opts.AutonomousAgentID = agentID
	opts.ContextType = models.TraceContextAutonomousAgent

	traces, err2 := h.docDBClient.Traces().List(ctx, opts)
	if err2 != nil {
		middleware.HandleError(c, errors.NewInternalError("failed to list traces", err2))
		return
	}

	total, err2 := h.docDBClient.Traces().Count(ctx, opts)
	if err2 != nil {
		middleware.HandleError(c, errors.NewInternalError("failed to count traces", err2))
		return
	}

	c.JSON(http.StatusOK, dto.ListTracesResponse{
		Traces: dto.TracesToResponse(traces),
		Total:  total,
	})
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
// @Description Retrieves a list of traces for autonomous agents with pagination, sorting, and filtering
// @Tags Traces
// @Accept json
// @Produce json
// @Param tenantId path string true "Tenant ID"
// @Param autonomousAgentId query string false "Filter by autonomous agent ID"
// @Param limit query int false "Maximum number of results (default: 20, max: 100)"
// @Param skip query int false "Number of results to skip (default: 0)"
// @Param order query string false "Sort order: asc or desc (default: desc)"
// @Param order_by query string false "Sort field: created_at or updated_at (default: created_at)"
// @Param created_after query string false "Filter: traces created after this ISO 8601 timestamp"
// @Param created_before query string false "Filter: traces created before this ISO 8601 timestamp"
// @Param expand query bool false "Include nodes and logs in response (default: false)"
// @Success 200 {object} dto.ListTracesResponse
// @Failure 400 {object} dto.ErrorResponse "Bad request"
// @Failure 401 {object} dto.ErrorResponse "Unauthorized"
// @Failure 500 {object} dto.ErrorResponse "Internal server error"
// @Security BearerAuth
// @Router /api/v1/agent-service/tenants/{tenantId}/autonomous-agents/traces [get]
func (h *TracesHandler) ListAutonomousAgentTraces(c *gin.Context) {
	ctx := c.Request.Context()
	tenantID := c.Param("tenantId")

	opts, err := h.parseListTracesQueryParams(c)
	if err != nil {
		middleware.HandleError(c, err)
		return
	}
	opts.TenantID = tenantID
	opts.ContextType = models.TraceContextAutonomousAgent

	// Optional filter by specific autonomous agent ID
	autonomousAgentID := c.Query("autonomousAgentId")
	if autonomousAgentID != "" {
		opts.AutonomousAgentID = autonomousAgentID
	}

	traces, err2 := h.docDBClient.Traces().List(ctx, opts)
	if err2 != nil {
		middleware.HandleError(c, errors.NewInternalError("failed to list traces", err2))
		return
	}

	total, err2 := h.docDBClient.Traces().Count(ctx, opts)
	if err2 != nil {
		middleware.HandleError(c, errors.NewInternalError("failed to count traces", err2))
		return
	}

	c.JSON(http.StatusOK, dto.ListTracesResponse{
		Traces: dto.TracesToResponse(traces),
		Total:  total,
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

// parseListTracesQueryParams parses common query parameters for trace listing endpoints.
func (h *TracesHandler) parseListTracesQueryParams(c *gin.Context) (*docdb.ListTracesOptions, *errors.DomainError) {
	limitStr := c.DefaultQuery("limit", "20")
	skipStr := c.DefaultQuery("skip", "0")
	order := c.DefaultQuery("order", "desc")
	orderByParam := c.DefaultQuery("order_by", "created_at")
	createdAfterStr := c.Query("created_after")
	createdBeforeStr := c.Query("created_before")
	expandStr := c.DefaultQuery("expand", "false")

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

	sortField := docdb.SortFieldCreatedAt
	if orderByParam == "updated_at" {
		sortField = docdb.SortFieldUpdatedAt
	}

	expand := expandStr == "true"

	opts := &docdb.ListTracesOptions{
		Limit:   limit,
		Skip:    skip,
		OrderBy: sortOrder,
		SortBy:  sortField,
		Expand:  expand,
	}

	// Parse date filters
	if createdAfterStr != "" {
		t, err := time.Parse(time.RFC3339, createdAfterStr)
		if err != nil {
			return nil, errors.NewValidationError("invalid created_after format", "must be ISO 8601 / RFC 3339")
		}
		opts.CreatedAfter = &t
	}
	if createdBeforeStr != "" {
		t, err := time.Parse(time.RFC3339, createdBeforeStr)
		if err != nil {
			return nil, errors.NewValidationError("invalid created_before format", "must be ISO 8601 / RFC 3339")
		}
		opts.CreatedBefore = &t
	}

	return opts, nil
}

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

// resolveAutonomousAgentFromBearer validates Bearer token access to an autonomous agent
// via the platform service. This is used when creating traces with autonomous agent context
// using a service principal Bearer token instead of an API key.
func (h *TracesHandler) resolveAutonomousAgentFromBearer(ctx context.Context, tenantID, agentID, authToken string) *errors.DomainError {
	if h.platformClient == nil {
		return nil
	}

	err := h.platformClient.ValidateAutonomousAgent(ctx, tenantID, agentID, authToken)
	if err != nil {
		errStr := err.Error()
		if strings.HasPrefix(errStr, "unauthorized") {
			return errors.NewUnauthorizedError("unauthorized access to autonomous agent")
		}
		if strings.HasPrefix(errStr, "forbidden") {
			return errors.NewForbiddenError("no permission to access autonomous agent")
		}
		if strings.HasPrefix(errStr, "not_found") {
			return errors.NewNotFoundError("autonomous agent", agentID)
		}
		return errors.NewInternalError("failed to validate autonomous agent access", err)
	}

	return nil
}

// resolveUserIDFromAPIKey validates the API key against the platform service and returns
// the autonomous agent user ID. This calls ValidateAutonomousAgentAPIKey which validates
// the API key against the stored primary/secondary keys without loading credential secrets.
func (h *TracesHandler) resolveUserIDFromAPIKey(ctx context.Context, tenantID, agentID, apiKey string) *errors.DomainError {
	if h.platformClient == nil {
		return nil
	}

	err := h.platformClient.ValidateAutonomousAgentAPIKey(ctx, tenantID, agentID, apiKey)
	if err != nil {
		errStr := err.Error()
		if strings.HasPrefix(errStr, "unauthorized") {
			return errors.NewUnauthorizedError("invalid API key")
		}
		if strings.HasPrefix(errStr, "not_found") {
			return errors.NewNotFoundError("autonomous agent", agentID)
		}
		return errors.NewInternalError("failed to validate autonomous agent API key", err)
	}

	return nil
}

// resolveUserIDForTrace determines the user ID based on the auth type and the trace's context.
// For Bearer token auth, it resolves the user via GetMe.
// For API key auth, it validates the key against the trace's autonomous agent.
func (h *TracesHandler) resolveUserIDForTrace(ctx context.Context, c *gin.Context, tenantID string, trace *models.Trace) (string, *errors.DomainError) {
	authToken := middleware.GetToken(c)
	apiKey := middleware.GetAutonomousAgentAPIKey(c)

	if authToken != "" {
		uid, err := h.getUserID(ctx, authToken)
		if err != nil {
			return "", errors.NewInternalError("failed to get user info", err)
		}
		return uid, nil
	}

	if apiKey != "" {
		if trace.ContextType != models.TraceContextAutonomousAgent || trace.AutonomousAgentID == "" {
			return "", errors.NewForbiddenError("API key authentication is only allowed for autonomous agent traces")
		}

		if domainErr := h.resolveUserIDFromAPIKey(ctx, tenantID, trace.AutonomousAgentID, apiKey); domainErr != nil {
			return "", domainErr
		}
		return "autonomous-agent-" + trace.AutonomousAgentID, nil
	}

	return "", errors.NewUnauthorizedError("no valid authentication provided")
}

// ImportConversationTrace handles PUT /tenants/{tenantId}/conversations/{conversationId}/traces/import/refresh
// @Summary Import and refresh traces for a conversation
// @Description Imports traces from an external system (Microsoft Foundry, N8N) for a conversation
// @Tags Traces
// @Accept json
// @Produce json
// @Param tenantId path string true "Tenant ID"
// @Param conversationId path string true "Conversation ID"
// @Param X-Microsoft-Foundry-API-Key header string false "Microsoft Foundry API Key (required for Foundry agents)"
// @Success 200 {object} dto.ImportTraceResponse
// @Failure 400 {object} dto.ErrorResponse "Bad request - missing required header or configuration"
// @Failure 401 {object} dto.ErrorResponse "Unauthorized"
// @Failure 404 {object} dto.ErrorResponse "Conversation not found"
// @Failure 500 {object} dto.ErrorResponse "Internal server error"
// @Security BearerAuth
// @Router /api/v1/agent-service/tenants/{tenantId}/conversations/{conversationId}/traces/import/refresh [put]
func (h *TracesHandler) ImportConversationTrace(c *gin.Context) {
	ctx := c.Request.Context()
	tenantID := c.Param("tenantId")
	conversationID := c.Param("conversationId")
	authToken := middleware.GetToken(c)

	// Get conversation details from platform service
	conversation, err := h.platformClient.GetConversation(ctx, tenantID, conversationID, authToken)
	if err != nil {
		errStr := err.Error()
		if strings.HasPrefix(errStr, "unauthorized") {
			middleware.HandleError(c, errors.NewUnauthorizedError("invalid or expired token"))
			return
		}
		if strings.HasPrefix(errStr, "forbidden") {
			middleware.HandleError(c, errors.NewForbiddenError("access denied to conversation"))
			return
		}
		if strings.HasPrefix(errStr, "not_found") {
			middleware.HandleError(c, errors.NewNotFoundError("conversation", conversationID))
			return
		}
		middleware.HandleError(c, errors.NewInternalError("failed to get conversation", err))
		return
	}

	// Get application configuration to determine agent type
	appConfig, err := h.platformClient.GetApplicationConfig(ctx, tenantID, conversation.ApplicationID, authToken)
	if err != nil {
		middleware.HandleError(c, errors.NewInternalError("failed to get application configuration", err))
		return
	}

	// Get user info for created_by field
	userInfo, err := h.platformClient.GetMe(ctx, authToken)
	if err != nil {
		middleware.HandleError(c, errors.NewInternalError("failed to get user info", err))
		return
	}

	// Check if importer is registered for this agent type
	if !h.importService.HasImporter(appConfig.Type) {
		middleware.HandleError(c, errors.NewValidationError(
			"unsupported agent type for trace import",
			string(appConfig.Type),
		))
		return
	}

	// Build backend-specific configuration based on agent type
	backendConfig, err := h.buildBackendConfig(c, appConfig, conversation)
	if err != nil {
		middleware.HandleError(c, err)
		return
	}

	// Create import request
	req := traceimport.NewImportRequest(
		tenantID,
		conversationID,
		conversation.ApplicationID,
		userInfo.ID,
	)
	req.BackendConfig = backendConfig

	// Import traces using factory pattern - no switch statement needed
	traceID, err := h.importService.Import(ctx, appConfig.Type, req)
	if err != nil {
		middleware.HandleError(c, errors.NewInternalError("failed to import traces", err))
		return
	}

	c.JSON(http.StatusOK, dto.ImportTraceResponse{
		ID: traceID,
	})
}

// buildBackendConfig builds the backend-specific configuration based on agent type.
// This is the ONLY place where agent-type specific logic exists in the handler.
func (h *TracesHandler) buildBackendConfig(
	c *gin.Context,
	appConfig *platform.ApplicationConfigResponse,
	conversation *platform.ConversationResponse,
) (map[string]interface{}, error) {
	switch appConfig.Type {
	case platform.AgentTypeFoundry:
		return h.buildFoundryConfig(c, appConfig, conversation)
	case platform.AgentTypeN8N:
		return h.buildN8NConfig(c, appConfig, conversation)
	default:
		// Return empty config for unknown types - importer will handle validation
		return make(map[string]interface{}), nil
	}
}

// buildFoundryConfig builds the Foundry-specific backend configuration.
func (h *TracesHandler) buildFoundryConfig(
	c *gin.Context,
	appConfig *platform.ApplicationConfigResponse,
	conversation *platform.ConversationResponse,
) (map[string]interface{}, error) {
	foundryAPIKey := c.GetHeader("X-Microsoft-Foundry-API-Key")

	if foundryAPIKey == "" {
		return nil, errors.NewValidationError(
			"X-Microsoft-Foundry-API-Key header is required for Foundry agents",
			"",
		)
	}

	if conversation.ExtConversationID == "" {
		return nil, errors.NewValidationError(
			"conversation has no external conversation ID",
			"",
		)
	}

	if appConfig.Settings.ProjectEndpoint == "" {
		return nil, errors.NewValidationError(
			"application configuration missing project endpoint",
			"",
		)
	}

	apiVersion := appConfig.Settings.APIVersion
	if apiVersion == "" {
		apiVersion = "2025-11-15-preview"
	}

	return map[string]interface{}{
		"ext_conversation_id": conversation.ExtConversationID,
		"project_endpoint":    appConfig.Settings.ProjectEndpoint,
		"api_version":         apiVersion,
		"api_token":           foundryAPIKey,
	}, nil
}

// buildN8NConfig builds the N8N-specific backend configuration.
func (h *TracesHandler) buildN8NConfig(
	c *gin.Context,
	appConfig *platform.ApplicationConfigResponse,
	conversation *platform.ConversationResponse,
) (map[string]interface{}, error) {
	// TODO: Implement N8N config extraction when N8N importer is added
	// Expected keys: execution_id, workflow_id, instance_url, api_key
	return nil, errors.NewValidationError(
		"N8N trace import not yet implemented",
		"",
	)
}

// --- Autonomous Agent Import Handlers ---

// ImportAutonomousAgentTrace handles PUT /tenants/{tenantId}/autonomous-agents/{agentId}/traces/import
// @Summary Import or update traces for an autonomous agent (upsert by executionId)
// @Description Imports traces from an external system (N8N, etc.) for an autonomous agent. If a trace with the same executionId already exists, it will be updated; otherwise a new trace is created.
// @Tags Traces
// @Accept json
// @Produce json
// @Param tenantId path string true "Tenant ID"
// @Param agentId path string true "Autonomous Agent ID"
// @Param X-Unified-UI-Autonomous-Agent-API-Key header string true "Autonomous Agent API Key"
// @Param request body dto.AutonomousAgentImportTraceRequest true "Import request"
// @Success 200 {object} dto.ImportTraceResponse "Trace updated"
// @Success 201 {object} dto.ImportTraceResponse "Trace created"
// @Failure 400 {object} dto.ErrorResponse "Bad request - validation error"
// @Failure 401 {object} dto.ErrorResponse "Unauthorized - invalid API key"
// @Failure 404 {object} dto.ErrorResponse "Autonomous agent not found"
// @Failure 500 {object} dto.ErrorResponse "Internal server error"
// @Security ApiKeyAuth
// @Router /api/v1/agent-service/tenants/{tenantId}/autonomous-agents/{agentId}/traces/import [put]
func (h *TracesHandler) ImportAutonomousAgentTrace(c *gin.Context) {
	ctx := c.Request.Context()
	tenantID := c.Param("tenantId")
	agentID := c.Param("agentId")
	apiKey := middleware.GetAutonomousAgentAPIKey(c)

	// Parse request body
	var req dto.AutonomousAgentImportTraceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		middleware.HandleError(c, errors.NewValidationError("invalid request body", err.Error()))
		return
	}

	// Get autonomous agent config from platform service (validates API key)
	agentConfig, err := h.platformClient.GetAutonomousAgentConfig(ctx, tenantID, agentID, apiKey)
	if err != nil {
		errStr := err.Error()
		if strings.HasPrefix(errStr, "unauthorized") {
			middleware.HandleError(c, errors.NewUnauthorizedError("invalid API key"))
			return
		}
		if strings.HasPrefix(errStr, "not_found") {
			middleware.HandleError(c, errors.NewNotFoundError("autonomous agent", agentID))
			return
		}
		middleware.HandleError(c, errors.NewInternalError("failed to get autonomous agent config", err))
		return
	}

	// Map request type to platform agent type
	agentType, err := h.mapImportType(req.Type)
	if err != nil {
		middleware.HandleError(c, errors.NewValidationError("invalid agent type", err.Error()))
		return
	}

	// Check if importer is registered for this agent type
	if !h.importService.HasImporter(agentType) {
		middleware.HandleError(c, errors.NewValidationError(
			"unsupported agent type for trace import",
			string(agentType),
		))
		return
	}

	// Check if a trace with this executionId (referenceId) already exists
	existingTrace, err := h.docDBClient.Traces().GetByReferenceID(ctx, tenantID, req.ExecutionID)
	if err != nil {
		middleware.HandleError(c, errors.NewInternalError("failed to check for existing trace", err))
		return
	}
	isUpdate := existingTrace != nil

	// Build backend-specific configuration
	backendConfig, err := h.buildAutonomousAgentBackendConfig(c, agentConfig, req)
	if err != nil {
		middleware.HandleError(c, err)
		return
	}

	// Create import request using autonomous agent context
	importReq := &traceimport.ImportRequest{
		TenantID:          tenantID,
		AutonomousAgentID: agentID,
		UserID:            "autonomous-agent-" + agentID, // Special user ID for autonomous agents
		BackendConfig:     backendConfig,
	}

	// If trace exists, set the existing trace ID to preserve it during upsert
	// The importer will handle the update logic using this ID
	if isUpdate {
		importReq.ExistingTraceID = existingTrace.ID
	}

	// Import traces using factory pattern
	traceID, err := h.importService.Import(ctx, agentType, importReq)
	if err != nil {
		middleware.HandleError(c, errors.NewInternalError("failed to import traces", err))
		return
	}

	// Return appropriate status code based on create vs update
	statusCode := http.StatusCreated
	if isUpdate {
		statusCode = http.StatusOK
	}

	c.JSON(statusCode, dto.ImportTraceResponse{
		ID: traceID,
	})
}

// RefreshAutonomousAgentImportTrace handles PUT /tenants/{tenantId}/autonomous-agents/{agentId}/traces/{traceId}/import/refresh
// @Summary Refresh an imported trace for an autonomous agent
// @Description Re-imports traces from the external system using the existing trace's reference ID
// @Tags Traces
// @Accept json
// @Produce json
// @Param tenantId path string true "Tenant ID"
// @Param agentId path string true "Autonomous Agent ID"
// @Param traceId path string true "Trace ID"
// @Param X-Unified-UI-Autonomous-Agent-API-Key header string true "Autonomous Agent API Key"
// @Success 200 {object} dto.ImportTraceResponse
// @Failure 400 {object} dto.ErrorResponse "Bad request - trace has no reference ID"
// @Failure 401 {object} dto.ErrorResponse "Unauthorized - invalid API key"
// @Failure 404 {object} dto.ErrorResponse "Trace or autonomous agent not found"
// @Failure 500 {object} dto.ErrorResponse "Internal server error"
// @Security ApiKeyAuth
// @Router /api/v1/agent-service/tenants/{tenantId}/autonomous-agents/{agentId}/traces/{traceId}/import/refresh [put]
func (h *TracesHandler) RefreshAutonomousAgentImportTrace(c *gin.Context) {
	ctx := c.Request.Context()
	tenantID := c.Param("tenantId")
	agentID := c.Param("agentId")
	traceID := c.Param("traceId")
	apiKey := middleware.GetAutonomousAgentAPIKey(c)

	// Get autonomous agent config from platform service (validates API key)
	agentConfig, err := h.platformClient.GetAutonomousAgentConfig(ctx, tenantID, agentID, apiKey)
	if err != nil {
		errStr := err.Error()
		if strings.HasPrefix(errStr, "unauthorized") {
			middleware.HandleError(c, errors.NewUnauthorizedError("invalid API key"))
			return
		}
		if strings.HasPrefix(errStr, "not_found") {
			middleware.HandleError(c, errors.NewNotFoundError("autonomous agent", agentID))
			return
		}
		middleware.HandleError(c, errors.NewInternalError("failed to get autonomous agent config", err))
		return
	}

	// Get existing trace
	trace, err := h.docDBClient.Traces().Get(ctx, traceID)
	if err != nil {
		middleware.HandleError(c, errors.NewInternalError("failed to get trace", err))
		return
	}
	if trace == nil {
		middleware.HandleError(c, errors.NewNotFoundError("trace", traceID))
		return
	}

	// Verify trace belongs to this autonomous agent
	if trace.AutonomousAgentID != agentID {
		middleware.HandleError(c, errors.NewForbiddenError("trace does not belong to this autonomous agent"))
		return
	}

	// Get execution ID from trace reference metadata
	executionID := h.getExecutionIDFromTrace(trace)
	if executionID == "" {
		middleware.HandleError(c, errors.NewValidationError(
			"trace has no execution ID",
			"cannot refresh trace without original execution reference",
		))
		return
	}

	// Determine agent type from config
	agentType := agentConfig.Type

	// Check if importer is registered for this agent type
	if !h.importService.HasImporter(agentType) {
		middleware.HandleError(c, errors.NewValidationError(
			"unsupported agent type for trace import",
			string(agentType),
		))
		return
	}

	// Build backend-specific configuration for refresh
	backendConfig, err := h.buildAutonomousAgentRefreshBackendConfig(c, agentConfig, executionID)
	if err != nil {
		middleware.HandleError(c, err)
		return
	}

	// Create import request
	importReq := &traceimport.ImportRequest{
		TenantID:      tenantID,
		UserID:        "autonomous-agent-" + agentID,
		BackendConfig: backendConfig,
	}

	// Import traces using factory pattern - this will update the existing trace
	newTraceID, err := h.importService.Import(ctx, agentType, importReq)
	if err != nil {
		middleware.HandleError(c, errors.NewInternalError("failed to refresh traces", err))
		return
	}

	// Ensure the trace is linked to the autonomous agent
	updatedTrace, err := h.docDBClient.Traces().Get(ctx, newTraceID)
	if err == nil && updatedTrace != nil {
		updatedTrace.AutonomousAgentID = agentID
		updatedTrace.ContextType = models.TraceContextAutonomousAgent
		_ = h.docDBClient.Traces().Update(ctx, updatedTrace)
	}

	c.JSON(http.StatusOK, dto.ImportTraceResponse{
		ID: newTraceID,
	})
}

// mapImportType maps the request type string to platform.AgentType.
func (h *TracesHandler) mapImportType(importType string) (platform.AgentType, error) {
	switch strings.ToUpper(importType) {
	case "N8N":
		return platform.AgentTypeN8N, nil
	case "MICROSOFT_FOUNDRY", "FOUNDRY":
		return platform.AgentTypeFoundry, nil
	default:
		return "", fmt.Errorf("unknown import type: %s", importType)
	}
}

// buildAutonomousAgentBackendConfig builds the backend-specific configuration for autonomous agent imports.
func (h *TracesHandler) buildAutonomousAgentBackendConfig(
	c *gin.Context,
	agentConfig *platform.AutonomousAgentConfigResponse,
	req dto.AutonomousAgentImportTraceRequest,
) (map[string]interface{}, error) {
	switch agentConfig.Type {
	case platform.AgentTypeN8N:
		return h.buildN8NAutonomousAgentConfig(c, agentConfig, req.ExecutionID, req.SessionID)
	default:
		return nil, errors.NewValidationError(
			"unsupported agent type for autonomous agent import",
			string(agentConfig.Type),
		)
	}
}

// buildAutonomousAgentRefreshBackendConfig builds the backend config for refresh operations.
func (h *TracesHandler) buildAutonomousAgentRefreshBackendConfig(
	c *gin.Context,
	agentConfig *platform.AutonomousAgentConfigResponse,
	executionID string,
) (map[string]interface{}, error) {
	switch agentConfig.Type {
	case platform.AgentTypeN8N:
		return h.buildN8NAutonomousAgentConfig(c, agentConfig, executionID, "")
	default:
		return nil, errors.NewValidationError(
			"unsupported agent type for autonomous agent import",
			string(agentConfig.Type),
		)
	}
}

// buildN8NAutonomousAgentConfig builds the N8N-specific backend configuration for autonomous agents.
func (h *TracesHandler) buildN8NAutonomousAgentConfig(
	c *gin.Context,
	agentConfig *platform.AutonomousAgentConfigResponse,
	executionID string,
	sessionID string,
) (map[string]interface{}, error) {
	settings := agentConfig.Settings

	if settings.N8NHost == "" {
		return nil, errors.NewValidationError(
			"autonomous agent configuration missing N8N host",
			"",
		)
	}

	// Get API key from credentials
	apiKey := ""
	if settings.APICredentials != nil {
		apiKey = settings.APICredentials.GetSecretAsString()
	}

	if apiKey == "" {
		return nil, errors.NewValidationError(
			"autonomous agent configuration missing API credentials",
			"",
		)
	}

	return map[string]interface{}{
		"execution_id": executionID,
		"session_id":   sessionID,
		"base_url":     settings.N8NHost,
		"workflow_id":  settings.WorkflowID,
		"api_key":      apiKey,
	}, nil
}

// getExecutionIDFromTrace extracts the execution ID from a trace's reference metadata.
func (h *TracesHandler) getExecutionIDFromTrace(trace *models.Trace) string {
	if trace.ReferenceMetadata == nil {
		return trace.ReferenceID // Fallback to referenceId
	}

	// Try to get N8N execution ID
	if execID, ok := trace.ReferenceMetadata["n8n_execution_id"].(string); ok && execID != "" {
		return execID
	}

	// Try to get Foundry conversation ID
	if extConvID, ok := trace.ReferenceMetadata["ext_conversation_id"].(string); ok && extConvID != "" {
		return extConvID
	}

	// Fallback to referenceId
	return trace.ReferenceID
}
