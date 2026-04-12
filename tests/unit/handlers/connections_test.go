// Package handlers_test provides unit tests for the API handlers.
package handlers_test

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"github.com/unifiedui/agent-service/internal/api/dto"
	"github.com/unifiedui/agent-service/internal/api/handlers"
	"github.com/unifiedui/agent-service/internal/services/connections"
	"github.com/unifiedui/agent-service/internal/services/platform"
	"github.com/unifiedui/agent-service/tests/mocks"
	"github.com/unifiedui/agent-service/tests/testutils"
)

const testConnTenantID = "tenant-conn-123"

func authMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader != "" && len(authHeader) > 7 {
			c.Set("auth_token", authHeader[7:])
		}
		c.Next()
	}
}

func setupConnectionsRouter(handler *handlers.ConnectionsHandler) *gin.Engine {
	router := testutils.SetupTestRouter()
	router.Use(authMiddleware())
	router.POST("/tenants/:tenantId/connections/test", handler.TestConnection)
	return router
}

func TestConnectionsHandler_TestConnection_Success(t *testing.T) {
	mockService := new(mocks.MockConnectionService)
	mockPlatform := new(mocks.MockPlatformClient)

	mockService.On("TestConnection",
		mock.Anything, "N8N_WORKFLOW", "https://n8n.example.com/workflow/abc123",
		map[string]interface{}(nil), (*platform.Credentials)(nil), "",
	).Return(&connections.TestResult{
		Success:        true,
		Message:        "Workflow 'Test WF' is active",
		ResponseTimeMs: 42,
	}, nil)

	handler := handlers.NewConnectionsHandler(mockService, mockPlatform)
	router := setupConnectionsRouter(handler)

	body := dto.TestConnectionRequest{
		TestType: dto.TestConnectionType("N8N_WORKFLOW"),
		URL:      "https://n8n.example.com/workflow/abc123",
	}

	w := testutils.PerformRequest(router, "POST",
		fmt.Sprintf("/tenants/%s/connections/test", testConnTenantID),
		body, nil)

	testutils.AssertStatusCode(t, http.StatusOK, w)

	var resp dto.TestConnectionResponse
	testutils.ParseJSONResponse(t, w, &resp)
	assert.True(t, resp.Success)
	assert.Equal(t, "Workflow 'Test WF' is active", resp.Message)
	assert.Equal(t, int64(42), resp.ResponseTimeMs)

	mockService.AssertExpectations(t)
}

func TestConnectionsHandler_TestConnection_WithCredential(t *testing.T) {
	mockService := new(mocks.MockConnectionService)
	mockPlatform := new(mocks.MockPlatformClient)

	credentialSecret := `{"username":"admin","password":"secret"}` //nolint:gosec // test data

	mockPlatform.On("GetCredentialSecret",
		mock.Anything, testConnTenantID, "cred-123", "test-token",
	).Return(credentialSecret, nil)

	var expectedSecret map[string]interface{}
	_ = json.Unmarshal([]byte(credentialSecret), &expectedSecret)

	mockService.On("TestConnection",
		mock.Anything, "N8N_CHAT_URL", "https://n8n.example.com/webhook/chat",
		map[string]interface{}(nil),
		mock.MatchedBy(func(cred *platform.Credentials) bool {
			secretMap, ok := cred.Secret.(map[string]interface{})
			return ok && secretMap["username"] == "admin" && secretMap["password"] == "secret"
		}),
		"test-token",
	).Return(&connections.TestResult{
		Success:        true,
		Message:        "Connection successful (response time: 150ms)",
		ResponseTimeMs: 150,
	}, nil)

	handler := handlers.NewConnectionsHandler(mockService, mockPlatform)
	router := setupConnectionsRouter(handler)

	body := dto.TestConnectionRequest{
		TestType:     dto.TestConnectionType("N8N_CHAT_URL"),
		URL:          "https://n8n.example.com/webhook/chat",
		CredentialID: "cred-123",
	}

	w := testutils.PerformRequest(router, "POST",
		fmt.Sprintf("/tenants/%s/connections/test", testConnTenantID),
		body, map[string]string{"Authorization": "Bearer test-token"})

	testutils.AssertStatusCode(t, http.StatusOK, w)

	var resp dto.TestConnectionResponse
	testutils.ParseJSONResponse(t, w, &resp)
	assert.True(t, resp.Success)

	mockService.AssertExpectations(t)
	mockPlatform.AssertExpectations(t)
}

func TestConnectionsHandler_TestConnection_InvalidTestType(t *testing.T) {
	mockService := new(mocks.MockConnectionService)
	mockPlatform := new(mocks.MockPlatformClient)

	handler := handlers.NewConnectionsHandler(mockService, mockPlatform)
	router := setupConnectionsRouter(handler)

	body := dto.TestConnectionRequest{
		TestType: dto.TestConnectionType("INVALID_TYPE"),
		URL:      "https://example.com",
	}

	w := testutils.PerformRequest(router, "POST",
		fmt.Sprintf("/tenants/%s/connections/test", testConnTenantID),
		body, nil)

	testutils.AssertStatusCode(t, http.StatusBadRequest, w)
}

func TestConnectionsHandler_TestConnection_InvalidBody(t *testing.T) {
	mockService := new(mocks.MockConnectionService)
	mockPlatform := new(mocks.MockPlatformClient)

	handler := handlers.NewConnectionsHandler(mockService, mockPlatform)
	router := setupConnectionsRouter(handler)

	w := testutils.PerformRequest(router, "POST",
		fmt.Sprintf("/tenants/%s/connections/test", testConnTenantID),
		"not-json", nil)

	testutils.AssertStatusCode(t, http.StatusBadRequest, w)
}

func TestConnectionsHandler_TestConnection_CredentialFetchError(t *testing.T) {
	mockService := new(mocks.MockConnectionService)
	mockPlatform := new(mocks.MockPlatformClient)

	mockPlatform.On("GetCredentialSecret",
		mock.Anything, testConnTenantID, "bad-cred", "test-token",
	).Return("", errors.New("credential not found"))

	handler := handlers.NewConnectionsHandler(mockService, mockPlatform)
	router := setupConnectionsRouter(handler)

	body := dto.TestConnectionRequest{
		TestType:     dto.TestConnectionType("REST_API_INVOKE"),
		URL:          "https://api.example.com/invoke",
		CredentialID: "bad-cred",
	}

	w := testutils.PerformRequest(router, "POST",
		fmt.Sprintf("/tenants/%s/connections/test", testConnTenantID),
		body, map[string]string{"Authorization": "Bearer test-token"})

	testutils.AssertStatusCode(t, http.StatusInternalServerError, w)

	mockPlatform.AssertExpectations(t)
}

func TestConnectionsHandler_TestConnection_ServiceError(t *testing.T) {
	mockService := new(mocks.MockConnectionService)
	mockPlatform := new(mocks.MockPlatformClient)

	mockService.On("TestConnection",
		mock.Anything, "FOUNDRY_AGENT", "https://foundry.example.com",
		mock.Anything, (*platform.Credentials)(nil), "",
	).Return(nil, errors.New("connection timeout"))

	handler := handlers.NewConnectionsHandler(mockService, mockPlatform)
	router := setupConnectionsRouter(handler)

	body := dto.TestConnectionRequest{
		TestType: dto.TestConnectionType("FOUNDRY_AGENT"),
		URL:      "https://foundry.example.com",
		Config: map[string]interface{}{
			"agent_name":  "test-agent",
			"api_version": "2025-11-15-preview",
		},
	}

	w := testutils.PerformRequest(router, "POST",
		fmt.Sprintf("/tenants/%s/connections/test", testConnTenantID),
		body, nil)

	testutils.AssertStatusCode(t, http.StatusInternalServerError, w)

	mockService.AssertExpectations(t)
}

func TestConnectionsHandler_TestConnection_ConnectionFailed(t *testing.T) {
	mockService := new(mocks.MockConnectionService)
	mockPlatform := new(mocks.MockPlatformClient)

	mockService.On("TestConnection",
		mock.Anything, "REST_API_CONVERSATION", "https://api.example.com/conversations",
		mock.Anything, (*platform.Credentials)(nil), "",
	).Return(&connections.TestResult{
		Success:        false,
		Message:        "Connection refused",
		ResponseTimeMs: 5,
	}, nil)

	handler := handlers.NewConnectionsHandler(mockService, mockPlatform)
	router := setupConnectionsRouter(handler)

	body := dto.TestConnectionRequest{
		TestType: dto.TestConnectionType("REST_API_CONVERSATION"),
		URL:      "https://api.example.com/conversations",
		Config: map[string]interface{}{
			"auth_type": "ANONYMOUS",
		},
	}

	w := testutils.PerformRequest(router, "POST",
		fmt.Sprintf("/tenants/%s/connections/test", testConnTenantID),
		body, nil)

	testutils.AssertStatusCode(t, http.StatusOK, w)

	var resp dto.TestConnectionResponse
	testutils.ParseJSONResponse(t, w, &resp)
	assert.False(t, resp.Success)
	assert.Equal(t, "Connection refused", resp.Message)

	mockService.AssertExpectations(t)
}

func TestConnectionsHandler_TestConnection_StringCredential(t *testing.T) {
	mockService := new(mocks.MockConnectionService)
	mockPlatform := new(mocks.MockPlatformClient)

	mockPlatform.On("GetCredentialSecret",
		mock.Anything, testConnTenantID, "api-key-cred", "test-token",
	).Return("my-api-key-value", nil)

	mockService.On("TestConnection",
		mock.Anything, "N8N_WEBHOOK", "https://n8n.example.com/webhook/test",
		map[string]interface{}(nil),
		mock.MatchedBy(func(cred *platform.Credentials) bool {
			return cred.Secret == "my-api-key-value"
		}),
		"test-token",
	).Return(&connections.TestResult{
		Success:        true,
		Message:        "Webhook responded with status 200",
		ResponseTimeMs: 80,
	}, nil)

	handler := handlers.NewConnectionsHandler(mockService, mockPlatform)
	router := setupConnectionsRouter(handler)

	body := dto.TestConnectionRequest{
		TestType:     dto.TestConnectionType("N8N_WEBHOOK"),
		URL:          "https://n8n.example.com/webhook/test",
		CredentialID: "api-key-cred",
	}

	w := testutils.PerformRequest(router, "POST",
		fmt.Sprintf("/tenants/%s/connections/test", testConnTenantID),
		body, map[string]string{"Authorization": "Bearer test-token"})

	testutils.AssertStatusCode(t, http.StatusOK, w)

	var resp dto.TestConnectionResponse
	testutils.ParseJSONResponse(t, w, &resp)
	assert.True(t, resp.Success)

	mockService.AssertExpectations(t)
	mockPlatform.AssertExpectations(t)
}
