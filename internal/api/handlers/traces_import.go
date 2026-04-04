package handlers

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/unifiedui/agent-service/internal/api/dto"
	"github.com/unifiedui/agent-service/internal/api/middleware"
	"github.com/unifiedui/agent-service/internal/domain/errors"
	"github.com/unifiedui/agent-service/internal/domain/models"
	"github.com/unifiedui/agent-service/internal/services/platform"
	"github.com/unifiedui/agent-service/internal/services/traceimport"
)

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
	tenantID := middleware.SanitizePathParam(c, "tenantId")
	conversationID := middleware.SanitizePathParam(c, "conversationId")
	authToken := middleware.GetToken(c)

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

	appConfig, err := h.platformClient.GetChatAgentConfig(ctx, tenantID, conversation.ChatAgentID, authToken, true)
	if err != nil {
		middleware.HandleError(c, errors.NewInternalError("failed to get chat agent configuration", err))
		return
	}

	userInfo, err := h.platformClient.GetMe(ctx, authToken)
	if err != nil {
		middleware.HandleError(c, errors.NewInternalError("failed to get user info", err))
		return
	}

	if !h.importService.HasImporter(appConfig.Type) {
		middleware.HandleError(c, errors.NewValidationError(
			"unsupported agent type for trace import",
			string(appConfig.Type),
		))
		return
	}

	backendConfig, err := h.buildBackendConfig(c, appConfig, conversation)
	if err != nil {
		middleware.HandleError(c, err)
		return
	}

	req := traceimport.NewImportRequest(
		tenantID,
		conversationID,
		conversation.ChatAgentID,
		userInfo.ID,
	)
	req.BackendConfig = backendConfig

	traceID, err := h.importService.Import(ctx, appConfig.Type, req)
	if err != nil {
		middleware.HandleError(c, errors.NewInternalError("failed to import traces", err))
		return
	}

	c.JSON(http.StatusOK, dto.ImportTraceResponse{
		ID: traceID,
	})
}

func (h *TracesHandler) buildBackendConfig(
	c *gin.Context,
	appConfig *platform.ChatAgentConfigResponse,
	conversation *platform.ConversationResponse,
) (map[string]interface{}, error) {
	switch appConfig.Type {
	case platform.AgentTypeFoundry:
		return h.buildFoundryConfig(c, appConfig, conversation)
	case platform.AgentTypeN8N:
		return h.buildN8NConfig(c, appConfig, conversation)
	default:
		return make(map[string]interface{}), nil
	}
}

func (h *TracesHandler) buildFoundryConfig(
	c *gin.Context,
	appConfig *platform.ChatAgentConfigResponse,
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
			"chat agent configuration missing project endpoint",
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

func (h *TracesHandler) buildN8NConfig(
	_ *gin.Context,
	_ *platform.ChatAgentConfigResponse,
	_ *platform.ConversationResponse,
) (map[string]interface{}, error) {
	return nil, errors.NewValidationError(
		"N8N trace import not yet implemented",
		"",
	)
}

// ImportWorkflowTrace handles PUT /tenants/{tenantId}/workflows/{agentId}/traces/import
// @Summary Import or update traces for an workflow (upsert by executionId)
// @Description Imports traces from an external system (N8N, etc.) for an workflow. If a trace with the same executionId already exists, it will be updated; otherwise a new trace is created.
// @Tags Traces
// @Accept json
// @Produce json
// @Param tenantId path string true "Tenant ID"
// @Param agentId path string true "Autonomous Agent ID"
// @Param Authorization header string false "Bearer token (requires WRITE permission on workflow)"
// @Param X-Unified-UI-Workflow-API-Key header string false "Autonomous Agent API Key"
// @Param request body dto.WorkflowImportTraceRequest true "Import request"
// @Success 200 {object} dto.ImportTraceResponse "Trace updated"
// @Success 201 {object} dto.ImportTraceResponse "Trace created"
// @Failure 400 {object} dto.ErrorResponse "Bad request - validation error"
// @Failure 401 {object} dto.ErrorResponse "Unauthorized - invalid credentials"
// @Failure 403 {object} dto.ErrorResponse "Forbidden - insufficient permissions"
// @Failure 404 {object} dto.ErrorResponse "Autonomous agent not found"
// @Failure 500 {object} dto.ErrorResponse "Internal server error"
// @Security BearerAuth
// @Security ApiKeyAuth
// @Router /api/v1/agent-service/tenants/{tenantId}/workflows/{agentId}/traces/import [put]
func (h *TracesHandler) ImportWorkflowTrace(c *gin.Context) {
	ctx := c.Request.Context()
	tenantID := middleware.SanitizePathParam(c, "tenantId")
	agentID := middleware.SanitizePathParam(c, "agentId")
	authToken := middleware.GetToken(c)
	apiKey := middleware.GetWorkflowAPIKey(c)

	var req dto.WorkflowImportTraceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		middleware.HandleError(c, errors.NewValidationError("invalid request body", err.Error()))
		return
	}

	var agentConfig *platform.WorkflowConfigResponse
	var userID string

	switch {
	case authToken != "":
		config, err := h.platformClient.GetWorkflowConfigWithBearer(ctx, tenantID, agentID, authToken)
		if err != nil {
			errStr := err.Error()
			if strings.HasPrefix(errStr, "unauthorized") {
				middleware.HandleError(c, errors.NewUnauthorizedError("unauthorized access to workflow"))
				return
			}
			if strings.HasPrefix(errStr, "forbidden") {
				middleware.HandleError(c, errors.NewForbiddenError("no WRITE permission on workflow"))
				return
			}
			if strings.HasPrefix(errStr, "not_found") {
				middleware.HandleError(c, errors.NewNotFoundError("workflow", agentID))
				return
			}
			middleware.HandleError(c, errors.NewInternalError("failed to get workflow config", err))
			return
		}
		agentConfig = config

		uid, err := h.getUserID(ctx, authToken)
		if err != nil {
			middleware.HandleError(c, errors.NewInternalError("failed to get user info", err))
			return
		}
		userID = uid
	case apiKey != "":
		config, err := h.platformClient.GetWorkflowConfig(ctx, tenantID, agentID, apiKey)
		if err != nil {
			errStr := err.Error()
			if strings.HasPrefix(errStr, "unauthorized") {
				middleware.HandleError(c, errors.NewUnauthorizedError("invalid API key"))
				return
			}
			if strings.HasPrefix(errStr, "not_found") {
				middleware.HandleError(c, errors.NewNotFoundError("workflow", agentID))
				return
			}
			middleware.HandleError(c, errors.NewInternalError("failed to get workflow config", err))
			return
		}
		agentConfig = config
		userID = "workflow-" + agentID
	default:
		middleware.HandleError(c, errors.NewUnauthorizedError("Bearer token or X-Unified-UI-Workflow-API-Key header required"))
		return
	}

	agentType, err := h.mapImportType(req.Type)
	if err != nil {
		middleware.HandleError(c, errors.NewValidationError("invalid agent type", err.Error()))
		return
	}

	if !h.importService.HasImporter(agentType) {
		middleware.HandleError(c, errors.NewValidationError(
			"unsupported agent type for trace import",
			string(agentType),
		))
		return
	}

	existingTrace, err := h.docDBClient.Traces().GetByReferenceID(ctx, tenantID, req.ExecutionID)
	if err != nil {
		middleware.HandleError(c, errors.NewInternalError("failed to check for existing trace", err))
		return
	}
	isUpdate := existingTrace != nil

	backendConfig, err := h.buildWorkflowBackendConfig(c, agentConfig, req)
	if err != nil {
		middleware.HandleError(c, err)
		return
	}

	importReq := &traceimport.ImportRequest{
		TenantID:      tenantID,
		WorkflowID:    agentID,
		UserID:        userID,
		BackendConfig: backendConfig,
	}

	if isUpdate {
		importReq.ExistingTraceID = existingTrace.ID
	}

	traceID, err := h.importService.Import(ctx, agentType, importReq)
	if err != nil {
		middleware.HandleError(c, errors.NewInternalError("failed to import traces", err))
		return
	}

	statusCode := http.StatusCreated
	if isUpdate {
		statusCode = http.StatusOK
	}

	c.JSON(statusCode, dto.ImportTraceResponse{
		ID: traceID,
	})
}

// RefreshWorkflowImportTrace handles PUT /tenants/{tenantId}/workflows/{agentId}/traces/{traceId}/import/refresh
// @Summary Refresh an imported trace for an workflow
// @Description Re-imports traces from the external system using the existing trace's reference ID
// @Tags Traces
// @Accept json
// @Produce json
// @Param tenantId path string true "Tenant ID"
// @Param agentId path string true "Autonomous Agent ID"
// @Param traceId path string true "Trace ID"
// @Param Authorization header string false "Bearer token (requires WRITE permission on workflow)"
// @Param X-Unified-UI-Workflow-API-Key header string false "Autonomous Agent API Key"
// @Success 200 {object} dto.ImportTraceResponse
// @Failure 400 {object} dto.ErrorResponse "Bad request - trace has no reference ID"
// @Failure 401 {object} dto.ErrorResponse "Unauthorized - invalid credentials"
// @Failure 403 {object} dto.ErrorResponse "Forbidden - insufficient permissions"
// @Failure 404 {object} dto.ErrorResponse "Trace or workflow not found"
// @Failure 500 {object} dto.ErrorResponse "Internal server error"
// @Security BearerAuth
// @Security ApiKeyAuth
// @Router /api/v1/agent-service/tenants/{tenantId}/workflows/{agentId}/traces/{traceId}/import/refresh [put]
func (h *TracesHandler) RefreshWorkflowImportTrace(c *gin.Context) {
	ctx := c.Request.Context()
	tenantID := middleware.SanitizePathParam(c, "tenantId")
	agentID := middleware.SanitizePathParam(c, "agentId")
	traceID := middleware.SanitizePathParam(c, "traceId")
	authToken := middleware.GetToken(c)
	apiKey := middleware.GetWorkflowAPIKey(c)

	var agentConfig *platform.WorkflowConfigResponse
	var userID string

	switch {
	case authToken != "":
		config, err := h.platformClient.GetWorkflowConfigWithBearer(ctx, tenantID, agentID, authToken)
		if err != nil {
			errStr := err.Error()
			if strings.HasPrefix(errStr, "unauthorized") {
				middleware.HandleError(c, errors.NewUnauthorizedError("unauthorized access to workflow"))
				return
			}
			if strings.HasPrefix(errStr, "forbidden") {
				middleware.HandleError(c, errors.NewForbiddenError("no WRITE permission on workflow"))
				return
			}
			if strings.HasPrefix(errStr, "not_found") {
				middleware.HandleError(c, errors.NewNotFoundError("workflow", agentID))
				return
			}
			middleware.HandleError(c, errors.NewInternalError("failed to get workflow config", err))
			return
		}
		agentConfig = config

		uid, err := h.getUserID(ctx, authToken)
		if err != nil {
			middleware.HandleError(c, errors.NewInternalError("failed to get user info", err))
			return
		}
		userID = uid
	case apiKey != "":
		config, err := h.platformClient.GetWorkflowConfig(ctx, tenantID, agentID, apiKey)
		if err != nil {
			errStr := err.Error()
			if strings.HasPrefix(errStr, "unauthorized") {
				middleware.HandleError(c, errors.NewUnauthorizedError("invalid API key"))
				return
			}
			if strings.HasPrefix(errStr, "not_found") {
				middleware.HandleError(c, errors.NewNotFoundError("workflow", agentID))
				return
			}
			middleware.HandleError(c, errors.NewInternalError("failed to get workflow config", err))
			return
		}
		agentConfig = config
		userID = "workflow-" + agentID
	default:
		middleware.HandleError(c, errors.NewUnauthorizedError("Bearer token or X-Unified-UI-Workflow-API-Key header required"))
		return
	}

	trace, err := h.docDBClient.Traces().Get(ctx, traceID)
	if err != nil {
		middleware.HandleError(c, errors.NewInternalError("failed to get trace", err))
		return
	}
	if trace == nil {
		middleware.HandleError(c, errors.NewNotFoundError("trace", traceID))
		return
	}

	if trace.WorkflowID != agentID {
		middleware.HandleError(c, errors.NewForbiddenError("trace does not belong to this workflow"))
		return
	}

	executionID := h.getExecutionIDFromTrace(trace)
	if executionID == "" {
		middleware.HandleError(c, errors.NewValidationError(
			"trace has no execution ID",
			"cannot refresh trace without original execution reference",
		))
		return
	}

	agentType := agentConfig.Type

	if !h.importService.HasImporter(agentType) {
		middleware.HandleError(c, errors.NewValidationError(
			"unsupported agent type for trace import",
			string(agentType),
		))
		return
	}

	backendConfig, err := h.buildWorkflowRefreshBackendConfig(c, agentConfig, executionID)
	if err != nil {
		middleware.HandleError(c, err)
		return
	}

	importReq := &traceimport.ImportRequest{
		TenantID:      tenantID,
		UserID:        userID,
		BackendConfig: backendConfig,
	}

	newTraceID, err := h.importService.Import(ctx, agentType, importReq)
	if err != nil {
		middleware.HandleError(c, errors.NewInternalError("failed to refresh traces", err))
		return
	}

	updatedTrace, err := h.docDBClient.Traces().Get(ctx, newTraceID)
	if err == nil && updatedTrace != nil {
		updatedTrace.WorkflowID = agentID
		updatedTrace.ContextType = models.TraceContextWorkflow
		_ = h.docDBClient.Traces().Update(ctx, updatedTrace)
	}

	c.JSON(http.StatusOK, dto.ImportTraceResponse{
		ID: newTraceID,
	})
}

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

func (h *TracesHandler) buildWorkflowBackendConfig(
	c *gin.Context,
	agentConfig *platform.WorkflowConfigResponse,
	req dto.WorkflowImportTraceRequest,
) (map[string]interface{}, error) {
	switch agentConfig.Type {
	case platform.AgentTypeN8N:
		return h.buildN8NWorkflowConfig(c, agentConfig, req.ExecutionID, req.SessionID)
	default:
		return nil, errors.NewValidationError(
			"unsupported agent type for workflow import",
			string(agentConfig.Type),
		)
	}
}

func (h *TracesHandler) buildWorkflowRefreshBackendConfig(
	c *gin.Context,
	agentConfig *platform.WorkflowConfigResponse,
	executionID string,
) (map[string]interface{}, error) {
	switch agentConfig.Type {
	case platform.AgentTypeN8N:
		return h.buildN8NWorkflowConfig(c, agentConfig, executionID, "")
	default:
		return nil, errors.NewValidationError(
			"unsupported agent type for workflow import",
			string(agentConfig.Type),
		)
	}
}

func (h *TracesHandler) buildN8NWorkflowConfig(
	_ *gin.Context,
	agentConfig *platform.WorkflowConfigResponse,
	executionID string,
	sessionID string,
) (map[string]interface{}, error) {
	settings := agentConfig.Settings

	if settings.N8NHost == "" {
		return nil, errors.NewValidationError(
			"workflow configuration missing N8N host",
			"",
		)
	}

	apiKey := ""
	if settings.APICredentials != nil {
		apiKey = settings.APICredentials.GetSecretAsString()
	}

	if apiKey == "" {
		return nil, errors.NewValidationError(
			"workflow configuration missing API credentials",
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

func (h *TracesHandler) getExecutionIDFromTrace(trace *models.Trace) string {
	if trace.ReferenceMetadata == nil {
		return trace.ReferenceID
	}

	if execID, ok := trace.ReferenceMetadata["n8n_execution_id"].(string); ok && execID != "" {
		return execID
	}

	if extConvID, ok := trace.ReferenceMetadata["ext_conversation_id"].(string); ok && extConvID != "" {
		return extConvID
	}

	return trace.ReferenceID
}

// ListWorkflowRuns handles GET /tenants/{tenantId}/workflows/{agentId}/workflow-runs
// @Summary List workflow runs for an workflow
// @Description Lists recent workflow executions from the external system (e.g., N8N) for an workflow
// @Tags Traces
// @Produce json
// @Param tenantId path string true "Tenant ID"
// @Param agentId path string true "Autonomous Agent ID"
// @Param limit query int false "Maximum number of runs to return (default: 50, max: 100)"
// @Param cursor query string false "Pagination cursor from previous response"
// @Success 200 {object} dto.ListWorkflowRunsResponse
// @Failure 400 {object} dto.ErrorResponse "Bad request - unsupported agent type"
// @Failure 401 {object} dto.ErrorResponse "Unauthorized"
// @Failure 403 {object} dto.ErrorResponse "Forbidden"
// @Failure 404 {object} dto.ErrorResponse "Autonomous agent not found"
// @Failure 500 {object} dto.ErrorResponse "Internal server error"
// @Security BearerAuth
// @Router /api/v1/agent-service/tenants/{tenantId}/workflows/{agentId}/workflow-runs [get]
func (h *TracesHandler) ListWorkflowRuns(c *gin.Context) {
	ctx := c.Request.Context()
	tenantID := middleware.SanitizePathParam(c, "tenantId")
	agentID := middleware.SanitizePathParam(c, "agentId")
	authToken := middleware.GetToken(c)
	cursor := c.Query("cursor")

	agentConfig, err := h.platformClient.GetWorkflowConfigWithBearer(ctx, tenantID, agentID, authToken)
	if err != nil {
		errStr := err.Error()
		if strings.HasPrefix(errStr, "unauthorized") {
			middleware.HandleError(c, errors.NewUnauthorizedError("unauthorized access to workflow"))
			return
		}
		if strings.HasPrefix(errStr, "forbidden") {
			middleware.HandleError(c, errors.NewForbiddenError("no permission on workflow"))
			return
		}
		if strings.HasPrefix(errStr, "not_found") {
			middleware.HandleError(c, errors.NewNotFoundError("workflow", agentID))
			return
		}
		middleware.HandleError(c, errors.NewInternalError("failed to get workflow config", err))
		return
	}

	if !h.importService.HasImporter(agentConfig.Type) {
		middleware.HandleError(c, errors.NewValidationError(
			"unsupported agent type for listing workflow runs",
			string(agentConfig.Type),
		))
		return
	}

	limit := 50
	if limitStr := c.Query("limit"); limitStr != "" {
		if parsed, parseErr := fmt.Sscanf(limitStr, "%d", &limit); parseErr != nil || parsed != 1 {
			limit = 50
		}
		if limit <= 0 || limit > 100 {
			limit = 50
		}
	}

	backendConfig, err := h.buildListWorkflowRunsConfig(agentConfig)
	if err != nil {
		middleware.HandleError(c, err)
		return
	}

	result, err := h.importService.ListExecutions(ctx, agentConfig.Type, backendConfig, limit, cursor)
	if err != nil {
		middleware.HandleError(c, errors.NewInternalError("failed to list workflow runs", err))
		return
	}

	response := dto.ListWorkflowRunsResponse{
		Runs:       make([]dto.WorkflowRunResponse, 0, len(result.Runs)),
		NextCursor: result.NextCursor,
	}
	for _, run := range result.Runs {
		response.Runs = append(response.Runs, dto.WorkflowRunResponse{
			ID:        run.ID,
			Status:    run.Status,
			StartedAt: run.StartedAt,
			StoppedAt: run.StoppedAt,
			Mode:      run.Mode,
		})
	}

	c.JSON(http.StatusOK, response)
}

func (h *TracesHandler) buildListWorkflowRunsConfig(
	agentConfig *platform.WorkflowConfigResponse,
) (map[string]interface{}, error) {
	settings := agentConfig.Settings

	if settings.N8NHost == "" {
		return nil, errors.NewValidationError(
			"workflow configuration missing N8N host",
			"",
		)
	}

	apiKey := ""
	if settings.APICredentials != nil {
		apiKey = settings.APICredentials.GetSecretAsString()
	}

	if apiKey == "" {
		return nil, errors.NewValidationError(
			"workflow configuration missing API credentials",
			"",
		)
	}

	return map[string]interface{}{
		"base_url":    settings.N8NHost,
		"workflow_id": settings.WorkflowID,
		"api_key":     apiKey,
	}, nil
}
