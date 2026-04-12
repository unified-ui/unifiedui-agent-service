// Package handlers provides HTTP handlers for the API.
package handlers

import (
	"net/http"
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
// @Description Creates a new trace for a conversation or workflow. Uses Bearer token for conversation context, API key for workflow context.
// @Tags Traces
// @Accept json
// @Produce json
// @Param tenantId path string true "Tenant ID"
// @Param request body dto.CreateTraceRequest true "Trace creation request"
// @Param Authorization header string false "Bearer token (required for conversation traces)"
// @Param X-Unified-UI-Workflow-API-Key header string false "API key (required for workflow traces)"
// @Success 201 {object} dto.CreateTraceResponse
// @Failure 400 {object} dto.ErrorResponse "Bad request - validation error"
// @Failure 401 {object} dto.ErrorResponse "Unauthorized"
// @Failure 404 {object} dto.ErrorResponse "Application, Conversation, or Workflow not found"
// @Failure 500 {object} dto.ErrorResponse "Internal server error"
// @Router /api/v1/agent-service/tenants/{tenantId}/traces [post]
func (h *TracesHandler) CreateTrace(c *gin.Context) {
	ctx := c.Request.Context()
	tenantID := middleware.SanitizePathParam(c, "tenantId")

	var req dto.CreateTraceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		middleware.HandleError(c, errors.NewValidationError("invalid request body", err.Error()))
		return
	}

	hasConversationContext := req.ChatAgentID != "" && req.ConversationID != ""
	hasAgentContext := req.WorkflowID != ""

	if hasConversationContext && hasAgentContext {
		middleware.HandleError(c, errors.NewValidationError(
			"invalid context",
			"cannot specify both conversation context and workflow context",
		))
		return
	}

	if !hasConversationContext && !hasAgentContext {
		middleware.HandleError(c, errors.NewValidationError(
			"missing context",
			"must specify either (chatAgentId + conversationId) or workflowId",
		))
		return
	}

	var userID string

	if hasConversationContext {
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

		if err := h.validateConversationContext(ctx, tenantID, req.ChatAgentID, req.ConversationID, authToken); err != nil {
			middleware.HandleError(c, err)
			return
		}

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
		authToken := middleware.GetToken(c)
		apiKey := middleware.GetWorkflowAPIKey(c)

		switch {
		case authToken != "":
			if domainErr := h.resolveWorkflowFromBearer(ctx, tenantID, req.WorkflowID, authToken); domainErr != nil {
				middleware.HandleError(c, domainErr)
				return
			}

			uid, err := h.getUserID(ctx, authToken)
			if err != nil {
				middleware.HandleError(c, errors.NewInternalError("failed to get user info", err))
				return
			}
			userID = uid
		case apiKey != "":
			if domainErr := h.resolveUserIDFromAPIKey(ctx, tenantID, req.WorkflowID, apiKey); domainErr != nil {
				middleware.HandleError(c, domainErr)
				return
			}
			userID = "workflow-" + req.WorkflowID
		default:
			middleware.HandleError(c, errors.NewUnauthorizedError("Bearer token or X-Unified-UI-Workflow-API-Key header required for workflow traces"))
			return
		}
	}

	traceID := req.ID
	if traceID == "" {
		traceID = uuid.New().String()
	}

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
		trace.ChatAgentID = req.ChatAgentID
		trace.ConversationID = req.ConversationID
		trace.ContextType = models.TraceContextConversation
	} else {
		trace.WorkflowID = req.WorkflowID
		trace.ContextType = models.TraceContextWorkflow
	}

	if len(req.Nodes) > 0 {
		trace.Nodes = dto.ConvertNodesToModel(req.Nodes, userID)
	} else {
		trace.Nodes = []models.TraceNode{}
	}

	if trace.Logs == nil {
		trace.Logs = []string{}
	}

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
// @Description Appends nodes to an existing trace. Uses Bearer token for conversation traces, API key for workflow traces.
// @Tags Traces
// @Accept json
// @Produce json
// @Param tenantId path string true "Tenant ID"
// @Param traceId path string true "Trace ID"
// @Param request body dto.AddNodesRequest true "Nodes to add"
// @Param Authorization header string false "Bearer token (required for conversation traces)"
// @Param X-Unified-UI-Workflow-API-Key header string false "API key (required for workflow traces)"
// @Success 200 {object} map[string]string "Success"
// @Failure 400 {object} dto.ErrorResponse "Bad request"
// @Failure 401 {object} dto.ErrorResponse "Unauthorized"
// @Failure 404 {object} dto.ErrorResponse "Trace not found"
// @Failure 500 {object} dto.ErrorResponse "Internal server error"
// @Router /api/v1/agent-service/tenants/{tenantId}/traces/{traceId}/nodes [post]
func (h *TracesHandler) AddNodes(c *gin.Context) {
	ctx := c.Request.Context()
	tenantID := middleware.SanitizePathParam(c, "tenantId")
	traceID := middleware.SanitizePathParam(c, "traceId")

	var req dto.AddNodesRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		middleware.HandleError(c, errors.NewValidationError("invalid request body", err.Error()))
		return
	}

	trace, err := h.docDBClient.Traces().Get(ctx, traceID)
	if err != nil {
		middleware.HandleError(c, errors.NewInternalError("failed to get trace", err))
		return
	}
	if trace == nil || trace.TenantID != tenantID {
		middleware.HandleError(c, errors.NewNotFoundError("trace", traceID))
		return
	}

	userID, domainErr := h.resolveUserIDForTrace(ctx, c, tenantID, trace)
	if domainErr != nil {
		middleware.HandleError(c, domainErr)
		return
	}

	nodes := dto.ConvertNodesToModel(req.Nodes, userID)

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
// @Param X-Unified-UI-Workflow-API-Key header string false "API key"
// @Success 200 {object} map[string]string "Success"
// @Failure 400 {object} dto.ErrorResponse "Bad request"
// @Failure 401 {object} dto.ErrorResponse "Unauthorized"
// @Failure 404 {object} dto.ErrorResponse "Trace not found"
// @Failure 500 {object} dto.ErrorResponse "Internal server error"
// @Router /api/v1/agent-service/tenants/{tenantId}/traces/{traceId}/logs [post]
func (h *TracesHandler) AddLogs(c *gin.Context) {
	ctx := c.Request.Context()
	tenantID := middleware.SanitizePathParam(c, "tenantId")
	traceID := middleware.SanitizePathParam(c, "traceId")

	var req dto.AddLogsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		middleware.HandleError(c, errors.NewValidationError("invalid request body", err.Error()))
		return
	}

	trace, err := h.docDBClient.Traces().Get(ctx, traceID)
	if err != nil {
		middleware.HandleError(c, errors.NewInternalError("failed to get trace", err))
		return
	}
	if trace == nil || trace.TenantID != tenantID {
		middleware.HandleError(c, errors.NewNotFoundError("trace", traceID))
		return
	}

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
	tenantID := middleware.SanitizePathParam(c, "tenantId")
	conversationID := middleware.SanitizePathParam(c, "conversationId")

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
	tenantID := middleware.SanitizePathParam(c, "tenantId")
	conversationID := middleware.SanitizePathParam(c, "conversationId")
	authToken := middleware.GetToken(c)

	var req dto.RefreshTraceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		middleware.HandleError(c, errors.NewValidationError("invalid request body", err.Error()))
		return
	}

	trace, err := h.docDBClient.Traces().GetByConversation(ctx, tenantID, conversationID)
	if err != nil {
		middleware.HandleError(c, errors.NewInternalError("failed to get trace", err))
		return
	}
	if trace == nil {
		middleware.HandleError(c, errors.NewNotFoundError("trace", conversationID))
		return
	}

	userID, err := h.getUserID(ctx, authToken)
	if err != nil {
		middleware.HandleError(c, errors.NewInternalError("failed to get user info", err))
		return
	}

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

	if err := h.docDBClient.Traces().Update(ctx, trace); err != nil {
		middleware.HandleError(c, errors.NewInternalError("failed to update trace", err))
		return
	}

	c.JSON(http.StatusOK, dto.TraceToResponse(trace))
}

// GetWorkflowTraces handles GET /tenants/{tenantId}/workflows/{agentId}/traces
// @Summary List traces for an workflow
// @Description Retrieves traces for a specific workflow with pagination, sorting, and filtering
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
// @Router /api/v1/agent-service/tenants/{tenantId}/workflows/{agentId}/traces [get]
func (h *TracesHandler) GetWorkflowTraces(c *gin.Context) {
	ctx := c.Request.Context()
	tenantID := middleware.SanitizePathParam(c, "tenantId")
	agentID := middleware.SanitizePathParam(c, "agentId")

	opts, err := h.parseListTracesQueryParams(c)
	if err != nil {
		middleware.HandleError(c, err)
		return
	}
	opts.TenantID = tenantID
	opts.WorkflowID = agentID
	opts.ContextType = models.TraceContextWorkflow

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

// RefreshWorkflowTrace handles PUT /tenants/{tenantId}/workflows/{agentId}/traces
// @Summary Refresh trace for an workflow
// @Description Replaces the trace for a specific workflow completely
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
// @Router /api/v1/agent-service/tenants/{tenantId}/workflows/{agentId}/traces [put]
func (h *TracesHandler) RefreshWorkflowTrace(c *gin.Context) {
	ctx := c.Request.Context()
	tenantID := middleware.SanitizePathParam(c, "tenantId")
	agentID := middleware.SanitizePathParam(c, "agentId")
	authToken := middleware.GetToken(c)

	var req dto.RefreshTraceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		middleware.HandleError(c, errors.NewValidationError("invalid request body", err.Error()))
		return
	}

	trace, err := h.docDBClient.Traces().GetByWorkflow(ctx, tenantID, agentID)
	if err != nil {
		middleware.HandleError(c, errors.NewInternalError("failed to get trace", err))
		return
	}
	if trace == nil {
		middleware.HandleError(c, errors.NewNotFoundError("trace", agentID))
		return
	}

	userID, err := h.getUserID(ctx, authToken)
	if err != nil {
		middleware.HandleError(c, errors.NewInternalError("failed to get user info", err))
		return
	}

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

	if err := h.docDBClient.Traces().Update(ctx, trace); err != nil {
		middleware.HandleError(c, errors.NewInternalError("failed to update trace", err))
		return
	}

	c.JSON(http.StatusOK, dto.TraceToResponse(trace))
}

// ListWorkflowTraces handles GET /tenants/{tenantId}/workflows/traces
// @Summary List traces for workflows
// @Description Retrieves a list of traces for workflows with pagination, sorting, and filtering
// @Tags Traces
// @Accept json
// @Produce json
// @Param tenantId path string true "Tenant ID"
// @Param workflowId query string false "Filter by workflow ID"
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
// @Router /api/v1/agent-service/tenants/{tenantId}/workflows/traces [get]
func (h *TracesHandler) ListWorkflowTraces(c *gin.Context) {
	ctx := c.Request.Context()
	tenantID := middleware.SanitizePathParam(c, "tenantId")

	opts, err := h.parseListTracesQueryParams(c)
	if err != nil {
		middleware.HandleError(c, err)
		return
	}
	opts.TenantID = tenantID
	opts.ContextType = models.TraceContextWorkflow

	workflowID := c.Query("workflowId")
	if workflowID != "" {
		opts.WorkflowID = workflowID
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
	tenantID := middleware.SanitizePathParam(c, "tenantId")
	traceID := middleware.SanitizePathParam(c, "traceId")

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
	tenantID := middleware.SanitizePathParam(c, "tenantId")
	traceID := middleware.SanitizePathParam(c, "traceId")

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
