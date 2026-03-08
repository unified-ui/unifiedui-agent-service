// Package handlers provides HTTP handlers for the API.
package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/unifiedui/agent-service/internal/api/dto"
	"github.com/unifiedui/agent-service/internal/api/middleware"
	domainerrors "github.com/unifiedui/agent-service/internal/domain/errors"
	"github.com/unifiedui/agent-service/internal/services/ai"
	"github.com/unifiedui/agent-service/internal/services/platform"
)

// AIHandler handles AI-related endpoints.
type AIHandler struct {
	aiService      ai.Service
	platformClient platform.Client
}

// NewAIHandler creates a new AIHandler.
func NewAIHandler(aiService ai.Service, platformClient platform.Client) *AIHandler {
	return &AIHandler{
		aiService:      aiService,
		platformClient: platformClient,
	}
}

// GenerateDescription handles POST /tenants/{tenantId}/ai/generate-description
// @Summary Generate or improve an entity description using AI
// @Description Uses configured LLM models to generate or polish a description for any entity type
// @Tags AI
// @Accept json
// @Produce json
// @Param tenantId path string true "Tenant ID"
// @Param request body dto.GenerateDescriptionRequest true "Description generation request"
// @Success 200 {object} dto.GenerateDescriptionResponse
// @Failure 400 {object} dto.ErrorResponse "Bad request - validation error"
// @Failure 401 {object} dto.ErrorResponse "Unauthorized"
// @Failure 500 {object} dto.ErrorResponse "Internal server error"
// @Router /api/v1/agent-service/tenants/{tenantId}/ai/generate-description [post]
// @Security BearerAuth
func (h *AIHandler) GenerateDescription(c *gin.Context) {
	tenantID := middleware.SanitizePathParam(c, "tenantId")

	var req dto.GenerateDescriptionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		middleware.HandleError(c, domainerrors.NewValidationError("invalid request body", err.Error()))
		return
	}

	description, err := h.aiService.GenerateDescription(
		c.Request.Context(),
		tenantID,
		req.EntityType,
		req.EntityName,
		req.ExistingDescription,
		req.Context,
	)
	if err != nil {
		middleware.HandleError(c, domainerrors.NewInternalError("failed to generate description", err))
		return
	}

	c.JSON(http.StatusOK, dto.GenerateDescriptionResponse{
		Description: description,
	})
}

// AnalyzeTrace handles POST /tenants/{tenantId}/ai/analyze-trace
// @Summary Analyze a trace error using AI
// @Description Analyzes a failed trace node and provides root cause analysis and suggested fixes
// @Tags AI
// @Accept json
// @Produce json
// @Param tenantId path string true "Tenant ID"
// @Param request body dto.AnalyzeTraceRequest true "Trace analysis request"
// @Success 200 {object} dto.AnalyzeTraceResponse
// @Failure 400 {object} dto.ErrorResponse "Bad request - validation error"
// @Failure 401 {object} dto.ErrorResponse "Unauthorized"
// @Failure 500 {object} dto.ErrorResponse "Internal server error"
// @Router /api/v1/agent-service/tenants/{tenantId}/ai/analyze-trace [post]
// @Security BearerAuth
func (h *AIHandler) AnalyzeTrace(c *gin.Context) {
	tenantID := middleware.SanitizePathParam(c, "tenantId")

	var req dto.AnalyzeTraceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		middleware.HandleError(c, domainerrors.NewValidationError("invalid request body", err.Error()))
		return
	}

	analysis, err := h.aiService.AnalyzeTrace(
		c.Request.Context(),
		tenantID,
		ai.AnalyzeTraceInput{
			TraceID:  req.TraceID,
			NodeID:   req.NodeID,
			Error:    req.Error,
			NodeName: req.NodeName,
			NodeType: req.NodeType,
			Input:    req.Input,
			Output:   req.Output,
		},
	)
	if err != nil {
		middleware.HandleError(c, domainerrors.NewInternalError("failed to analyze trace", err))
		return
	}

	c.JSON(http.StatusOK, dto.AnalyzeTraceResponse{
		Analysis: analysis,
	})
}

// SummarizeTrace handles POST /tenants/{tenantId}/ai/summarize-trace
// @Summary Summarize a trace using AI
// @Description Summarizes trace nodes at the specified detail level (short, medium, long)
// @Tags AI
// @Accept json
// @Produce json
// @Param tenantId path string true "Tenant ID"
// @Param request body dto.SummarizeTraceRequest true "Trace summarization request"
// @Success 200 {object} dto.SummarizeTraceResponse
// @Failure 400 {object} dto.ErrorResponse "Bad request - validation error"
// @Failure 401 {object} dto.ErrorResponse "Unauthorized"
// @Failure 500 {object} dto.ErrorResponse "Internal server error"
// @Router /api/v1/agent-service/tenants/{tenantId}/ai/summarize-trace [post]
// @Security BearerAuth
func (h *AIHandler) SummarizeTrace(c *gin.Context) {
	tenantID := middleware.SanitizePathParam(c, "tenantId")

	var req dto.SummarizeTraceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		middleware.HandleError(c, domainerrors.NewValidationError("invalid request body", err.Error()))
		return
	}

	summary, err := h.aiService.SummarizeTrace(
		c.Request.Context(),
		tenantID,
		ai.SummarizeTraceInput{
			TraceID:     req.TraceID,
			DetailLevel: req.DetailLevel,
			Nodes:       req.Nodes,
		},
	)
	if err != nil {
		middleware.HandleError(c, domainerrors.NewInternalError("failed to summarize trace", err))
		return
	}

	c.JSON(http.StatusOK, dto.SummarizeTraceResponse{
		Summary: summary,
	})
}

// TestModel handles POST /tenants/{tenantId}/ai/test-model
// @Summary Test an AI model configuration
// @Description Tests LLM connectivity by sending a simple ping message with the provided config and credentials
// @Tags AI
// @Accept json
// @Produce json
// @Param tenantId path string true "Tenant ID"
// @Param request body dto.TestModelRequest true "Model test request"
// @Success 200 {object} dto.TestModelResponse
// @Failure 400 {object} dto.ErrorResponse "Bad request - validation error"
// @Failure 401 {object} dto.ErrorResponse "Unauthorized"
// @Failure 500 {object} dto.ErrorResponse "Internal server error"
// @Router /api/v1/agent-service/tenants/{tenantId}/ai/test-model [post]
// @Security BearerAuth
func (h *AIHandler) TestModel(c *gin.Context) {
	tenantID := middleware.SanitizePathParam(c, "tenantId")

	var req dto.TestModelRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		middleware.HandleError(c, domainerrors.NewValidationError("invalid request body", err.Error()))
		return
	}

	var credentialSecret map[string]interface{}

	if req.CredentialID != "" {
		authToken, _ := c.Get("auth_token")
		token, _ := authToken.(string)

		secretStr, err := h.platformClient.GetCredentialSecret(
			c.Request.Context(),
			tenantID,
			req.CredentialID,
			token,
		)
		if err != nil {
			middleware.HandleError(c, domainerrors.NewInternalError("failed to retrieve credential secret", err))
			return
		}

		if err := json.Unmarshal([]byte(secretStr), &credentialSecret); err != nil {
			credentialSecret = map[string]interface{}{"api_key": secretStr}
		}
	}

	result, err := h.aiService.TestModel(
		c.Request.Context(),
		req.Provider,
		req.Config,
		credentialSecret,
	)
	if err != nil {
		middleware.HandleError(c, domainerrors.NewInternalError("failed to test model", err))
		return
	}

	c.JSON(http.StatusOK, dto.TestModelResponse{
		Success:        result.Success,
		Message:        result.Message,
		ResponseTimeMs: result.ResponseTimeMs,
	})
}

// GetCapabilities handles GET /tenants/{tenantId}/ai/capabilities
// @Summary Get available AI capabilities for a tenant
// @Description Returns which AI features are available based on configured models
// @Tags AI
// @Produce json
// @Param tenantId path string true "Tenant ID"
// @Success 200 {object} dto.AICapabilitiesResponse
// @Failure 401 {object} dto.ErrorResponse "Unauthorized"
// @Failure 500 {object} dto.ErrorResponse "Internal server error"
// @Router /api/v1/agent-service/tenants/{tenantId}/ai/capabilities [get]
// @Security BearerAuth
func (h *AIHandler) GetCapabilities(c *gin.Context) {
	tenantID := middleware.SanitizePathParam(c, "tenantId")

	capabilities, err := h.aiService.GetCapabilities(c.Request.Context(), tenantID)
	if err != nil {
		middleware.HandleError(c, domainerrors.NewInternalError("failed to get capabilities", err))
		return
	}

	c.JSON(http.StatusOK, dto.AICapabilitiesResponse{
		TitleGeneration:       capabilities.TitleGeneration,
		DescriptionGeneration: capabilities.DescriptionGeneration,
		TraceAnalysis:         capabilities.TraceAnalysis,
		Summarization:         capabilities.Summarization,
	})
}

// TraceChat handles POST /tenants/{tenantId}/ai/trace-chat
// @Summary Chat about a trace using AI
// @Description Handles conversational questions about a trace, maintaining chat history for context
// @Tags AI
// @Accept json
// @Produce json
// @Param tenantId path string true "Tenant ID"
// @Param request body dto.TraceChatRequest true "Trace chat request"
// @Success 200 {object} dto.TraceChatResponse
// @Failure 400 {object} dto.ErrorResponse "Bad request - validation error"
// @Failure 401 {object} dto.ErrorResponse "Unauthorized"
// @Failure 500 {object} dto.ErrorResponse "Internal server error"
// @Router /api/v1/agent-service/tenants/{tenantId}/ai/trace-chat [post]
// @Security BearerAuth
func (h *AIHandler) TraceChat(c *gin.Context) {
	tenantID := middleware.SanitizePathParam(c, "tenantId")

	var req dto.TraceChatRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		middleware.HandleError(c, domainerrors.NewValidationError("invalid request body", err.Error()))
		return
	}

	history := make([]ai.ChatMessage, len(req.History))
	for i, msg := range req.History {
		history[i] = ai.ChatMessage{
			Role:    msg.Role,
			Content: msg.Content,
		}
	}

	reply, err := h.aiService.TraceChat(
		c.Request.Context(),
		tenantID,
		ai.TraceChatInput{
			Trace:        req.Trace,
			SelectedNode: req.SelectedNode,
			Message:      req.Message,
			History:      history,
		},
	)
	if err != nil {
		middleware.HandleError(c, domainerrors.NewInternalError("failed to process trace chat", err))
		return
	}

	c.JSON(http.StatusOK, dto.TraceChatResponse{
		Reply: reply,
	})
}
