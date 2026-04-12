// Package handlers provides HTTP handlers for the API.
package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/unifiedui/agent-service/internal/api/dto"
	"github.com/unifiedui/agent-service/internal/api/middleware"
	domainerrors "github.com/unifiedui/agent-service/internal/domain/errors"
	"github.com/unifiedui/agent-service/internal/services/connections"
	"github.com/unifiedui/agent-service/internal/services/platform"
)

// ConnectionsHandler handles connection testing endpoints.
type ConnectionsHandler struct {
	connectionService connections.Service
	platformClient    platform.Client
}

// NewConnectionsHandler creates a new ConnectionsHandler.
func NewConnectionsHandler(connectionService connections.Service, platformClient platform.Client) *ConnectionsHandler {
	return &ConnectionsHandler{
		connectionService: connectionService,
		platformClient:    platformClient,
	}
}

// TestConnection handles POST /tenants/{tenantId}/connections/test
// @Summary Test an external service connection
// @Description Tests connectivity to an external service (n8n, Foundry, REST API) with the provided URL, credentials and configuration
// @Tags Connections
// @Accept json
// @Produce json
// @Param tenantId path string true "Tenant ID"
// @Param request body dto.TestConnectionRequest true "Connection test request"
// @Success 200 {object} dto.TestConnectionResponse
// @Failure 400 {object} dto.ErrorResponse "Bad request - validation error"
// @Failure 401 {object} dto.ErrorResponse "Unauthorized"
// @Failure 500 {object} dto.ErrorResponse "Internal server error"
// @Router /api/v1/agent-service/tenants/{tenantId}/connections/test [post]
// @Security BearerAuth
func (h *ConnectionsHandler) TestConnection(c *gin.Context) {
	tenantID := middleware.SanitizePathParam(c, "tenantId")

	var req dto.TestConnectionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		middleware.HandleError(c, domainerrors.NewValidationError("invalid request body", err.Error()))
		return
	}

	if !dto.ValidTestConnectionTypes[req.TestType] {
		middleware.HandleError(c, domainerrors.NewValidationError("invalid test_type", string(req.TestType)))
		return
	}

	var credential *platform.Credentials

	authToken, _ := c.Get("auth_token")
	userToken, _ := authToken.(string)

	foundryToken := c.GetHeader("X-Microsoft-Foundry-API-Key")

	if req.CredentialID != "" {
		secretStr, err := h.platformClient.GetCredentialSecret(
			c.Request.Context(),
			tenantID,
			req.CredentialID,
			userToken,
		)
		if err != nil {
			middleware.HandleError(c, domainerrors.NewInternalError("failed to retrieve credential secret", err))
			return
		}

		credential = &platform.Credentials{}

		var secretObj map[string]interface{}
		if err := json.Unmarshal([]byte(secretStr), &secretObj); err != nil {
			credential.Secret = secretStr
		} else {
			credential.Secret = secretObj
		}
	}

	effectiveToken := userToken
	if foundryToken != "" {
		effectiveToken = foundryToken
	}

	result, err := h.connectionService.TestConnection(
		c.Request.Context(),
		string(req.TestType),
		req.URL,
		req.Config,
		credential,
		effectiveToken,
	)
	if err != nil {
		middleware.HandleError(c, domainerrors.NewInternalError("failed to test connection", err))
		return
	}

	c.JSON(http.StatusOK, dto.TestConnectionResponse{
		Success:        result.Success,
		Message:        result.Message,
		ResponseTimeMs: result.ResponseTimeMs,
	})
}
