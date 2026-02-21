// Package handlers provides HTTP handlers for the API.
// This file contains extended internal tests for trace import functions to improve coverage.
package handlers

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/unifiedui/agent-service/internal/api/dto"
	"github.com/unifiedui/agent-service/internal/domain/models"
	"github.com/unifiedui/agent-service/internal/services/platform"
	"github.com/unifiedui/agent-service/internal/services/traceimport"
	"github.com/unifiedui/agent-service/tests/mocks"
)

// Test constants for extended tests
const (
	testTenantID          = "tenant-test-123"
	testAutonomousAgentID = "auto-agent-123"
	testTraceID           = "trace-test-xyz"
	testExecutionID       = "exec-123"
	testUserID            = "user-test-def"
)

// --- mapImportType Tests ---

func TestMapImportType_N8N(t *testing.T) {
	handler := &TracesHandler{}

	agentType, err := handler.mapImportType("N8N")

	require.NoError(t, err)
	assert.Equal(t, platform.AgentTypeN8N, agentType)
}

func TestMapImportType_N8N_LowerCase(t *testing.T) {
	handler := &TracesHandler{}

	agentType, err := handler.mapImportType("n8n")

	require.NoError(t, err)
	assert.Equal(t, platform.AgentTypeN8N, agentType)
}

func TestMapImportType_MicrosoftFoundry(t *testing.T) {
	handler := &TracesHandler{}

	agentType, err := handler.mapImportType("MICROSOFT_FOUNDRY")

	require.NoError(t, err)
	assert.Equal(t, platform.AgentTypeFoundry, agentType)
}

func TestMapImportType_Foundry(t *testing.T) {
	handler := &TracesHandler{}

	agentType, err := handler.mapImportType("FOUNDRY")

	require.NoError(t, err)
	assert.Equal(t, platform.AgentTypeFoundry, agentType)
}

func TestMapImportType_Foundry_LowerCase(t *testing.T) {
	handler := &TracesHandler{}

	agentType, err := handler.mapImportType("foundry")

	require.NoError(t, err)
	assert.Equal(t, platform.AgentTypeFoundry, agentType)
}

func TestMapImportType_MixedCase(t *testing.T) {
	handler := &TracesHandler{}

	agentType, err := handler.mapImportType("Microsoft_Foundry")

	require.NoError(t, err)
	assert.Equal(t, platform.AgentTypeFoundry, agentType)
}

func TestMapImportType_UnknownType(t *testing.T) {
	handler := &TracesHandler{}

	agentType, err := handler.mapImportType("UNKNOWN")

	require.Error(t, err)
	assert.Contains(t, err.Error(), "unknown import type")
	assert.Equal(t, platform.AgentType(""), agentType)
}

func TestMapImportType_EmptyString(t *testing.T) {
	handler := &TracesHandler{}

	agentType, err := handler.mapImportType("")

	require.Error(t, err)
	assert.Contains(t, err.Error(), "unknown import type")
	assert.Equal(t, platform.AgentType(""), agentType)
}

// --- buildAutonomousAgentBackendConfig Tests ---

func TestBuildAutonomousAgentBackendConfig_N8N_Success(t *testing.T) {
	handler := &TracesHandler{}

	c, _ := setupTestContext(nil)

	agentConfig := &platform.AutonomousAgentConfigResponse{
		Type:              platform.AgentTypeN8N,
		TenantID:          testTenantID,
		AutonomousAgentID: testAutonomousAgentID,
		Settings: platform.AutonomousAgentConfigSettings{
			N8NHost:    "https://n8n.example.com",
			WorkflowID: "workflow-123",
			APICredentials: &platform.Credentials{
				ID:     "cred-123",
				Type:   platform.CredentialTypeN8NAPIKey,
				Secret: "test-api-key",
			},
		},
	}

	req := dto.AutonomousAgentImportTraceRequest{
		Type:        "N8N",
		ExecutionID: testExecutionID,
		SessionID:   "session-456",
	}

	config, err := handler.buildAutonomousAgentBackendConfig(c, agentConfig, req)

	require.NoError(t, err)
	assert.NotNil(t, config)
	assert.Equal(t, testExecutionID, config["execution_id"])
	assert.Equal(t, "session-456", config["session_id"])
	assert.Equal(t, "https://n8n.example.com", config["base_url"])
	assert.Equal(t, "workflow-123", config["workflow_id"])
	assert.Equal(t, "test-api-key", config["api_key"])
}

func TestBuildAutonomousAgentBackendConfig_Foundry_Unsupported(t *testing.T) {
	handler := &TracesHandler{}

	c, _ := setupTestContext(nil)

	agentConfig := &platform.AutonomousAgentConfigResponse{
		Type:              platform.AgentTypeFoundry,
		TenantID:          testTenantID,
		AutonomousAgentID: testAutonomousAgentID,
		Settings:          platform.AutonomousAgentConfigSettings{},
	}

	req := dto.AutonomousAgentImportTraceRequest{
		Type:        "FOUNDRY",
		ExecutionID: testExecutionID,
	}

	config, err := handler.buildAutonomousAgentBackendConfig(c, agentConfig, req)

	assert.Nil(t, config)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unsupported agent type for autonomous agent import")
}

func TestBuildAutonomousAgentBackendConfig_UnknownType(t *testing.T) {
	handler := &TracesHandler{}

	c, _ := setupTestContext(nil)

	agentConfig := &platform.AutonomousAgentConfigResponse{
		Type:              platform.AgentType("custom"),
		TenantID:          testTenantID,
		AutonomousAgentID: testAutonomousAgentID,
		Settings:          platform.AutonomousAgentConfigSettings{},
	}

	req := dto.AutonomousAgentImportTraceRequest{
		Type:        "custom",
		ExecutionID: testExecutionID,
	}

	config, err := handler.buildAutonomousAgentBackendConfig(c, agentConfig, req)

	assert.Nil(t, config)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unsupported agent type for autonomous agent import")
}

// --- buildAutonomousAgentRefreshBackendConfig Tests ---

func TestBuildAutonomousAgentRefreshBackendConfig_N8N_Success(t *testing.T) {
	handler := &TracesHandler{}

	c, _ := setupTestContext(nil)

	agentConfig := &platform.AutonomousAgentConfigResponse{
		Type:              platform.AgentTypeN8N,
		TenantID:          testTenantID,
		AutonomousAgentID: testAutonomousAgentID,
		Settings: platform.AutonomousAgentConfigSettings{
			N8NHost:    "https://n8n.example.com",
			WorkflowID: "workflow-123",
			APICredentials: &platform.Credentials{
				ID:     "cred-123",
				Type:   platform.CredentialTypeN8NAPIKey,
				Secret: "test-api-key",
			},
		},
	}

	config, err := handler.buildAutonomousAgentRefreshBackendConfig(c, agentConfig, testExecutionID)

	require.NoError(t, err)
	assert.NotNil(t, config)
	assert.Equal(t, testExecutionID, config["execution_id"])
	assert.Equal(t, "", config["session_id"]) // Session ID should be empty for refresh
	assert.Equal(t, "https://n8n.example.com", config["base_url"])
	assert.Equal(t, "workflow-123", config["workflow_id"])
}

func TestBuildAutonomousAgentRefreshBackendConfig_Foundry_Unsupported(t *testing.T) {
	handler := &TracesHandler{}

	c, _ := setupTestContext(nil)

	agentConfig := &platform.AutonomousAgentConfigResponse{
		Type:              platform.AgentTypeFoundry,
		TenantID:          testTenantID,
		AutonomousAgentID: testAutonomousAgentID,
		Settings:          platform.AutonomousAgentConfigSettings{},
	}

	config, err := handler.buildAutonomousAgentRefreshBackendConfig(c, agentConfig, testExecutionID)

	assert.Nil(t, config)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unsupported agent type for autonomous agent import")
}

func TestBuildAutonomousAgentRefreshBackendConfig_UnknownType(t *testing.T) {
	handler := &TracesHandler{}

	c, _ := setupTestContext(nil)

	agentConfig := &platform.AutonomousAgentConfigResponse{
		Type:              platform.AgentType("unknown"),
		TenantID:          testTenantID,
		AutonomousAgentID: testAutonomousAgentID,
		Settings:          platform.AutonomousAgentConfigSettings{},
	}

	config, err := handler.buildAutonomousAgentRefreshBackendConfig(c, agentConfig, testExecutionID)

	assert.Nil(t, config)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unsupported agent type for autonomous agent import")
}

// --- buildN8NAutonomousAgentConfig Tests ---

func TestBuildN8NAutonomousAgentConfig_Success(t *testing.T) {
	handler := &TracesHandler{}

	c, _ := setupTestContext(nil)

	agentConfig := &platform.AutonomousAgentConfigResponse{
		Type:              platform.AgentTypeN8N,
		TenantID:          testTenantID,
		AutonomousAgentID: testAutonomousAgentID,
		Settings: platform.AutonomousAgentConfigSettings{
			N8NHost:    "https://n8n.example.com",
			WorkflowID: "workflow-123",
			APICredentials: &platform.Credentials{
				ID:     "cred-123",
				Type:   platform.CredentialTypeN8NAPIKey,
				Secret: "test-api-key",
			},
		},
	}

	config, err := handler.buildN8NAutonomousAgentConfig(c, agentConfig, testExecutionID, "session-456")

	require.NoError(t, err)
	assert.NotNil(t, config)
	assert.Equal(t, testExecutionID, config["execution_id"])
	assert.Equal(t, "session-456", config["session_id"])
	assert.Equal(t, "https://n8n.example.com", config["base_url"])
	assert.Equal(t, "workflow-123", config["workflow_id"])
	assert.Equal(t, "test-api-key", config["api_key"])
}

func TestBuildN8NAutonomousAgentConfig_MissingN8NHost(t *testing.T) {
	handler := &TracesHandler{}

	c, _ := setupTestContext(nil)

	agentConfig := &platform.AutonomousAgentConfigResponse{
		Type:              platform.AgentTypeN8N,
		TenantID:          testTenantID,
		AutonomousAgentID: testAutonomousAgentID,
		Settings: platform.AutonomousAgentConfigSettings{
			N8NHost:    "", // Missing
			WorkflowID: "workflow-123",
			APICredentials: &platform.Credentials{
				ID:     "cred-123",
				Type:   platform.CredentialTypeN8NAPIKey,
				Secret: "test-api-key",
			},
		},
	}

	config, err := handler.buildN8NAutonomousAgentConfig(c, agentConfig, testExecutionID, "session-456")

	assert.Nil(t, config)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "autonomous agent configuration missing N8N host")
}

func TestBuildN8NAutonomousAgentConfig_MissingAPICredentials(t *testing.T) {
	handler := &TracesHandler{}

	c, _ := setupTestContext(nil)

	agentConfig := &platform.AutonomousAgentConfigResponse{
		Type:              platform.AgentTypeN8N,
		TenantID:          testTenantID,
		AutonomousAgentID: testAutonomousAgentID,
		Settings: platform.AutonomousAgentConfigSettings{
			N8NHost:        "https://n8n.example.com",
			WorkflowID:     "workflow-123",
			APICredentials: nil, // Missing
		},
	}

	config, err := handler.buildN8NAutonomousAgentConfig(c, agentConfig, testExecutionID, "session-456")

	assert.Nil(t, config)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "autonomous agent configuration missing API credentials")
}

func TestBuildN8NAutonomousAgentConfig_EmptySecretString(t *testing.T) {
	handler := &TracesHandler{}

	c, _ := setupTestContext(nil)

	agentConfig := &platform.AutonomousAgentConfigResponse{
		Type:              platform.AgentTypeN8N,
		TenantID:          testTenantID,
		AutonomousAgentID: testAutonomousAgentID,
		Settings: platform.AutonomousAgentConfigSettings{
			N8NHost:    "https://n8n.example.com",
			WorkflowID: "workflow-123",
			APICredentials: &platform.Credentials{
				ID:     "cred-123",
				Type:   platform.CredentialTypeN8NAPIKey,
				Secret: "", // Empty secret
			},
		},
	}

	config, err := handler.buildN8NAutonomousAgentConfig(c, agentConfig, testExecutionID, "session-456")

	assert.Nil(t, config)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "autonomous agent configuration missing API credentials")
}

func TestBuildN8NAutonomousAgentConfig_NonStringSecret(t *testing.T) {
	handler := &TracesHandler{}

	c, _ := setupTestContext(nil)

	// Test with non-string secret (e.g., map for basic auth)
	agentConfig := &platform.AutonomousAgentConfigResponse{
		Type:              platform.AgentTypeN8N,
		TenantID:          testTenantID,
		AutonomousAgentID: testAutonomousAgentID,
		Settings: platform.AutonomousAgentConfigSettings{
			N8NHost:    "https://n8n.example.com",
			WorkflowID: "workflow-123",
			APICredentials: &platform.Credentials{
				ID:   "cred-123",
				Type: platform.CredentialTypeN8NBasicAuth,
				Secret: map[string]interface{}{
					"username": "user",
					"password": "pass",
				},
			},
		},
	}

	config, err := handler.buildN8NAutonomousAgentConfig(c, agentConfig, testExecutionID, "session-456")

	// GetSecretAsString returns empty string for non-string secrets
	assert.Nil(t, config)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "autonomous agent configuration missing API credentials")
}

func TestBuildN8NAutonomousAgentConfig_EmptySessionID(t *testing.T) {
	handler := &TracesHandler{}

	c, _ := setupTestContext(nil)

	agentConfig := &platform.AutonomousAgentConfigResponse{
		Type:              platform.AgentTypeN8N,
		TenantID:          testTenantID,
		AutonomousAgentID: testAutonomousAgentID,
		Settings: platform.AutonomousAgentConfigSettings{
			N8NHost:    "https://n8n.example.com",
			WorkflowID: "workflow-123",
			APICredentials: &platform.Credentials{
				ID:     "cred-123",
				Type:   platform.CredentialTypeN8NAPIKey,
				Secret: "test-api-key",
			},
		},
	}

	// Call with empty session ID (valid scenario for refresh)
	config, err := handler.buildN8NAutonomousAgentConfig(c, agentConfig, testExecutionID, "")

	require.NoError(t, err)
	assert.NotNil(t, config)
	assert.Equal(t, testExecutionID, config["execution_id"])
	assert.Equal(t, "", config["session_id"])
}

// --- getExecutionIDFromTrace Tests ---

func TestGetExecutionIDFromTrace_WithN8NExecutionID(t *testing.T) {
	handler := &TracesHandler{}

	trace := &models.Trace{
		ID:          testTraceID,
		ReferenceID: "reference-123",
		ReferenceMetadata: map[string]interface{}{
			"n8n_execution_id": "n8n-exec-456",
		},
	}

	executionID := handler.getExecutionIDFromTrace(trace)

	assert.Equal(t, "n8n-exec-456", executionID)
}

func TestGetExecutionIDFromTrace_WithExtConversationID(t *testing.T) {
	handler := &TracesHandler{}

	trace := &models.Trace{
		ID:          testTraceID,
		ReferenceID: "reference-123",
		ReferenceMetadata: map[string]interface{}{
			"ext_conversation_id": "ext-conv-789",
		},
	}

	executionID := handler.getExecutionIDFromTrace(trace)

	assert.Equal(t, "ext-conv-789", executionID)
}

func TestGetExecutionIDFromTrace_FallbackToReferenceID(t *testing.T) {
	handler := &TracesHandler{}

	trace := &models.Trace{
		ID:                testTraceID,
		ReferenceID:       "reference-123",
		ReferenceMetadata: map[string]interface{}{},
	}

	executionID := handler.getExecutionIDFromTrace(trace)

	assert.Equal(t, "reference-123", executionID)
}

func TestGetExecutionIDFromTrace_NilReferenceMetadata(t *testing.T) {
	handler := &TracesHandler{}

	trace := &models.Trace{
		ID:                testTraceID,
		ReferenceID:       "reference-123",
		ReferenceMetadata: nil,
	}

	executionID := handler.getExecutionIDFromTrace(trace)

	assert.Equal(t, "reference-123", executionID)
}

func TestGetExecutionIDFromTrace_EmptyN8NExecutionID(t *testing.T) {
	handler := &TracesHandler{}

	trace := &models.Trace{
		ID:          testTraceID,
		ReferenceID: "reference-123",
		ReferenceMetadata: map[string]interface{}{
			"n8n_execution_id": "", // Empty string
		},
	}

	executionID := handler.getExecutionIDFromTrace(trace)

	// Should fall back to reference ID when n8n_execution_id is empty
	assert.Equal(t, "reference-123", executionID)
}

func TestGetExecutionIDFromTrace_EmptyExtConversationID(t *testing.T) {
	handler := &TracesHandler{}

	trace := &models.Trace{
		ID:          testTraceID,
		ReferenceID: "reference-123",
		ReferenceMetadata: map[string]interface{}{
			"ext_conversation_id": "", // Empty string
		},
	}

	executionID := handler.getExecutionIDFromTrace(trace)

	// Should fall back to reference ID when ext_conversation_id is empty
	assert.Equal(t, "reference-123", executionID)
}

func TestGetExecutionIDFromTrace_N8NExecutionIDPriority(t *testing.T) {
	handler := &TracesHandler{}

	trace := &models.Trace{
		ID:          testTraceID,
		ReferenceID: "reference-123",
		ReferenceMetadata: map[string]interface{}{
			"n8n_execution_id":    "n8n-exec-456",
			"ext_conversation_id": "ext-conv-789",
		},
	}

	executionID := handler.getExecutionIDFromTrace(trace)

	// n8n_execution_id should have priority
	assert.Equal(t, "n8n-exec-456", executionID)
}

func TestGetExecutionIDFromTrace_WrongType(t *testing.T) {
	handler := &TracesHandler{}

	trace := &models.Trace{
		ID:          testTraceID,
		ReferenceID: "reference-123",
		ReferenceMetadata: map[string]interface{}{
			"n8n_execution_id": 12345, // Wrong type (int instead of string)
		},
	}

	executionID := handler.getExecutionIDFromTrace(trace)

	// Should fall back to reference ID when type assertion fails
	assert.Equal(t, "reference-123", executionID)
}

func TestGetExecutionIDFromTrace_AllEmpty(t *testing.T) {
	handler := &TracesHandler{}

	trace := &models.Trace{
		ID:                testTraceID,
		ReferenceID:       "",
		ReferenceMetadata: nil,
	}

	executionID := handler.getExecutionIDFromTrace(trace)

	assert.Equal(t, "", executionID)
}

// --- ImportAutonomousAgentTrace Handler Tests ---

// createTestTracesHandlerWithMocks creates a TracesHandler with mocks for handler testing.
func createTestTracesHandlerWithMocks(mockDocDB *mocks.MockDocDBClient, mockPlatform *mocks.MockPlatformClient) *TracesHandler {
	importService := traceimport.NewImportService(mockDocDB)
	return NewTracesHandler(mockDocDB, mockPlatform, importService)
}

// setupTestContextWithBody creates a test Gin context with a request body and optional headers.
func setupTestContextWithBody(method, path string, body interface{}, headers map[string]string) (*gin.Context, *httptest.ResponseRecorder) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	var req *http.Request
	if body != nil {
		jsonBody, _ := json.Marshal(body)
		req = httptest.NewRequest(method, path, bytes.NewReader(jsonBody))
		req.Header.Set("Content-Type", "application/json")
	} else {
		req = httptest.NewRequest(method, path, nil)
	}

	for key, value := range headers {
		req.Header.Set(key, value)
	}

	c.Request = req
	return c, w
}

// setContextParams sets path parameters on a Gin context.
func setContextParams(c *gin.Context, params map[string]string) {
	var ginParams []gin.Param
	for key, value := range params {
		ginParams = append(ginParams, gin.Param{Key: key, Value: value})
	}
	c.Params = ginParams
}

func TestImportAutonomousAgentTrace_InvalidRequestBody(t *testing.T) {
	mockDocDB := mocks.NewMockDocDBClient()
	mockPlatform := &mocks.MockPlatformClient{}

	handler := createTestTracesHandlerWithMocks(mockDocDB, mockPlatform)

	// Setup context with invalid JSON body
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPut, "/test", bytes.NewReader([]byte("invalid json")))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Request.Header.Set("X-Unified-UI-Autonomous-Agent-API-Key", "test-api-key")
	c.Set("autonomous_agent_api_key", "test-api-key")

	setContextParams(c, map[string]string{
		"tenantId": testTenantID,
		"agentId":  testAutonomousAgentID,
	})

	handler.ImportAutonomousAgentTrace(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestImportAutonomousAgentTrace_NoAuthProvided(t *testing.T) {
	mockDocDB := mocks.NewMockDocDBClient()
	mockPlatform := &mocks.MockPlatformClient{}

	handler := createTestTracesHandlerWithMocks(mockDocDB, mockPlatform)

	importReq := dto.AutonomousAgentImportTraceRequest{
		Type:        "N8N",
		ExecutionID: testExecutionID,
	}

	c, w := setupTestContextWithBody(http.MethodPut, "/test", importReq, nil)
	setContextParams(c, map[string]string{
		"tenantId": testTenantID,
		"agentId":  testAutonomousAgentID,
	})
	// No auth token or API key set

	handler.ImportAutonomousAgentTrace(c)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestImportAutonomousAgentTrace_BearerToken_Unauthorized(t *testing.T) {
	mockDocDB := mocks.NewMockDocDBClient()
	mockPlatform := &mocks.MockPlatformClient{}

	mockPlatform.On("GetAutonomousAgentConfigWithBearer", mock.Anything, testTenantID, testAutonomousAgentID, "test-bearer-token").
		Return(nil, fmt.Errorf("unauthorized: invalid token"))

	handler := createTestTracesHandlerWithMocks(mockDocDB, mockPlatform)

	importReq := dto.AutonomousAgentImportTraceRequest{
		Type:        "N8N",
		ExecutionID: testExecutionID,
	}

	c, w := setupTestContextWithBody(http.MethodPut, "/test", importReq, map[string]string{
		"Authorization": "Bearer test-bearer-token",
	})
	c.Set("auth_token", "test-bearer-token")
	setContextParams(c, map[string]string{
		"tenantId": testTenantID,
		"agentId":  testAutonomousAgentID,
	})

	handler.ImportAutonomousAgentTrace(c)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
	mockPlatform.AssertExpectations(t)
}

func TestImportAutonomousAgentTrace_BearerToken_Forbidden(t *testing.T) {
	mockDocDB := mocks.NewMockDocDBClient()
	mockPlatform := &mocks.MockPlatformClient{}

	mockPlatform.On("GetAutonomousAgentConfigWithBearer", mock.Anything, testTenantID, testAutonomousAgentID, "test-bearer-token").
		Return(nil, fmt.Errorf("forbidden: insufficient permissions"))

	handler := createTestTracesHandlerWithMocks(mockDocDB, mockPlatform)

	importReq := dto.AutonomousAgentImportTraceRequest{
		Type:        "N8N",
		ExecutionID: testExecutionID,
	}

	c, w := setupTestContextWithBody(http.MethodPut, "/test", importReq, map[string]string{
		"Authorization": "Bearer test-bearer-token",
	})
	c.Set("auth_token", "test-bearer-token")
	setContextParams(c, map[string]string{
		"tenantId": testTenantID,
		"agentId":  testAutonomousAgentID,
	})

	handler.ImportAutonomousAgentTrace(c)

	assert.Equal(t, http.StatusForbidden, w.Code)
	mockPlatform.AssertExpectations(t)
}

func TestImportAutonomousAgentTrace_BearerToken_NotFound(t *testing.T) {
	mockDocDB := mocks.NewMockDocDBClient()
	mockPlatform := &mocks.MockPlatformClient{}

	mockPlatform.On("GetAutonomousAgentConfigWithBearer", mock.Anything, testTenantID, testAutonomousAgentID, "test-bearer-token").
		Return(nil, fmt.Errorf("not_found: autonomous agent not found"))

	handler := createTestTracesHandlerWithMocks(mockDocDB, mockPlatform)

	importReq := dto.AutonomousAgentImportTraceRequest{
		Type:        "N8N",
		ExecutionID: testExecutionID,
	}

	c, w := setupTestContextWithBody(http.MethodPut, "/test", importReq, map[string]string{
		"Authorization": "Bearer test-bearer-token",
	})
	c.Set("auth_token", "test-bearer-token")
	setContextParams(c, map[string]string{
		"tenantId": testTenantID,
		"agentId":  testAutonomousAgentID,
	})

	handler.ImportAutonomousAgentTrace(c)

	assert.Equal(t, http.StatusNotFound, w.Code)
	mockPlatform.AssertExpectations(t)
}

func TestImportAutonomousAgentTrace_BearerToken_InternalError(t *testing.T) {
	mockDocDB := mocks.NewMockDocDBClient()
	mockPlatform := &mocks.MockPlatformClient{}

	mockPlatform.On("GetAutonomousAgentConfigWithBearer", mock.Anything, testTenantID, testAutonomousAgentID, "test-bearer-token").
		Return(nil, fmt.Errorf("database connection failed"))

	handler := createTestTracesHandlerWithMocks(mockDocDB, mockPlatform)

	importReq := dto.AutonomousAgentImportTraceRequest{
		Type:        "N8N",
		ExecutionID: testExecutionID,
	}

	c, w := setupTestContextWithBody(http.MethodPut, "/test", importReq, map[string]string{
		"Authorization": "Bearer test-bearer-token",
	})
	c.Set("auth_token", "test-bearer-token")
	setContextParams(c, map[string]string{
		"tenantId": testTenantID,
		"agentId":  testAutonomousAgentID,
	})

	handler.ImportAutonomousAgentTrace(c)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
	mockPlatform.AssertExpectations(t)
}

func TestImportAutonomousAgentTrace_BearerToken_GetUserFallbackToSystem(t *testing.T) {
	// When GetMe fails, getUserID returns "system" as fallback, not an error
	mockDocDB := mocks.NewMockDocDBClient()
	mockPlatform := &mocks.MockPlatformClient{}

	agentConfig := &platform.AutonomousAgentConfigResponse{
		Type:              platform.AgentTypeN8N,
		TenantID:          testTenantID,
		AutonomousAgentID: testAutonomousAgentID,
		Settings: platform.AutonomousAgentConfigSettings{
			N8NHost:    "https://n8n.example.com",
			WorkflowID: "workflow-123",
			APICredentials: &platform.Credentials{
				ID:     "cred-123",
				Type:   platform.CredentialTypeN8NAPIKey,
				Secret: "test-api-key",
			},
		},
	}

	mockPlatform.On("GetAutonomousAgentConfigWithBearer", mock.Anything, testTenantID, testAutonomousAgentID, "test-bearer-token").
		Return(agentConfig, nil)
	mockPlatform.On("GetMe", mock.Anything, "test-bearer-token").
		Return(nil, fmt.Errorf("failed to get user info"))
	// GetMe failure leads to "system" user ID fallback, so import continues
	mockDocDB.GetTracesCollection().On("GetByReferenceID", mock.Anything, testTenantID, testExecutionID).
		Return(nil, nil)
	mockDocDB.GetTracesCollection().On("Create", mock.Anything, mock.Anything).
		Return(nil)
	mockDocDB.GetTracesCollection().On("Get", mock.Anything, mock.Anything).
		Return(&models.Trace{ID: "mock-trace-id"}, nil)
	mockDocDB.GetTracesCollection().On("Update", mock.Anything, mock.Anything).
		Return(nil)

	handler := createTestTracesHandlerWithMocks(mockDocDB, mockPlatform)
	handler.GetImportService().RegisterImporter(mocks.NewMockTraceImporter())

	importReq := dto.AutonomousAgentImportTraceRequest{
		Type:        "N8N",
		ExecutionID: testExecutionID,
	}

	c, w := setupTestContextWithBody(http.MethodPut, "/test", importReq, map[string]string{
		"Authorization": "Bearer test-bearer-token",
	})
	c.Set("auth_token", "test-bearer-token")
	setContextParams(c, map[string]string{
		"tenantId": testTenantID,
		"agentId":  testAutonomousAgentID,
	})

	handler.ImportAutonomousAgentTrace(c)

	// Should succeed with fallback user ID
	assert.Equal(t, http.StatusCreated, w.Code)
	mockPlatform.AssertExpectations(t)
}

func TestImportAutonomousAgentTrace_APIKey_InternalError(t *testing.T) {
	mockDocDB := mocks.NewMockDocDBClient()
	mockPlatform := &mocks.MockPlatformClient{}

	mockPlatform.On("GetAutonomousAgentConfig", mock.Anything, testTenantID, testAutonomousAgentID, "test-api-key").
		Return(nil, fmt.Errorf("database connection failed"))

	handler := createTestTracesHandlerWithMocks(mockDocDB, mockPlatform)

	importReq := dto.AutonomousAgentImportTraceRequest{
		Type:        "N8N",
		ExecutionID: testExecutionID,
	}

	c, w := setupTestContextWithBody(http.MethodPut, "/test", importReq, map[string]string{
		"X-Unified-UI-Autonomous-Agent-API-Key": "test-api-key",
	})
	c.Set("autonomous_agent_api_key", "test-api-key")
	setContextParams(c, map[string]string{
		"tenantId": testTenantID,
		"agentId":  testAutonomousAgentID,
	})

	handler.ImportAutonomousAgentTrace(c)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
	mockPlatform.AssertExpectations(t)
}

func TestImportAutonomousAgentTrace_GetByReferenceIDError(t *testing.T) {
	mockDocDB := mocks.NewMockDocDBClient()
	mockPlatform := &mocks.MockPlatformClient{}

	agentConfig := &platform.AutonomousAgentConfigResponse{
		Type:              platform.AgentTypeN8N,
		TenantID:          testTenantID,
		AutonomousAgentID: testAutonomousAgentID,
		Settings: platform.AutonomousAgentConfigSettings{
			N8NHost:    "https://n8n.example.com",
			WorkflowID: "workflow-123",
			APICredentials: &platform.Credentials{
				ID:     "cred-123",
				Type:   platform.CredentialTypeN8NAPIKey,
				Secret: "test-api-key",
			},
		},
	}

	mockPlatform.On("GetAutonomousAgentConfig", mock.Anything, testTenantID, testAutonomousAgentID, "test-api-key").
		Return(agentConfig, nil)
	mockDocDB.GetTracesCollection().On("GetByReferenceID", mock.Anything, testTenantID, testExecutionID).
		Return(nil, fmt.Errorf("database error"))

	handler := createTestTracesHandlerWithMocks(mockDocDB, mockPlatform)
	handler.GetImportService().RegisterImporter(mocks.NewMockTraceImporter())

	importReq := dto.AutonomousAgentImportTraceRequest{
		Type:        "N8N",
		ExecutionID: testExecutionID,
	}

	c, w := setupTestContextWithBody(http.MethodPut, "/test", importReq, map[string]string{
		"X-Unified-UI-Autonomous-Agent-API-Key": "test-api-key",
	})
	c.Set("autonomous_agent_api_key", "test-api-key")
	setContextParams(c, map[string]string{
		"tenantId": testTenantID,
		"agentId":  testAutonomousAgentID,
	})

	handler.ImportAutonomousAgentTrace(c)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
	mockPlatform.AssertExpectations(t)
}

func TestImportAutonomousAgentTrace_NoImporterRegistered(t *testing.T) {
	mockDocDB := mocks.NewMockDocDBClient()
	mockPlatform := &mocks.MockPlatformClient{}

	agentConfig := &platform.AutonomousAgentConfigResponse{
		Type:              platform.AgentTypeN8N,
		TenantID:          testTenantID,
		AutonomousAgentID: testAutonomousAgentID,
		Settings:          platform.AutonomousAgentConfigSettings{},
	}

	mockPlatform.On("GetAutonomousAgentConfig", mock.Anything, testTenantID, testAutonomousAgentID, "test-api-key").
		Return(agentConfig, nil)

	handler := createTestTracesHandlerWithMocks(mockDocDB, mockPlatform)
	// Note: No importer registered

	importReq := dto.AutonomousAgentImportTraceRequest{
		Type:        "N8N",
		ExecutionID: testExecutionID,
	}

	c, w := setupTestContextWithBody(http.MethodPut, "/test", importReq, map[string]string{
		"X-Unified-UI-Autonomous-Agent-API-Key": "test-api-key",
	})
	c.Set("autonomous_agent_api_key", "test-api-key")
	setContextParams(c, map[string]string{
		"tenantId": testTenantID,
		"agentId":  testAutonomousAgentID,
	})

	handler.ImportAutonomousAgentTrace(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	mockPlatform.AssertExpectations(t)
}

func TestImportAutonomousAgentTrace_BackendConfigError(t *testing.T) {
	mockDocDB := mocks.NewMockDocDBClient()
	mockPlatform := &mocks.MockPlatformClient{}

	// Config without required N8N host
	agentConfig := &platform.AutonomousAgentConfigResponse{
		Type:              platform.AgentTypeN8N,
		TenantID:          testTenantID,
		AutonomousAgentID: testAutonomousAgentID,
		Settings: platform.AutonomousAgentConfigSettings{
			N8NHost: "", // Missing
		},
	}

	mockPlatform.On("GetAutonomousAgentConfig", mock.Anything, testTenantID, testAutonomousAgentID, "test-api-key").
		Return(agentConfig, nil)
	mockDocDB.GetTracesCollection().On("GetByReferenceID", mock.Anything, testTenantID, testExecutionID).
		Return(nil, nil)

	handler := createTestTracesHandlerWithMocks(mockDocDB, mockPlatform)
	handler.GetImportService().RegisterImporter(mocks.NewMockTraceImporter())

	importReq := dto.AutonomousAgentImportTraceRequest{
		Type:        "N8N",
		ExecutionID: testExecutionID,
	}

	c, w := setupTestContextWithBody(http.MethodPut, "/test", importReq, map[string]string{
		"X-Unified-UI-Autonomous-Agent-API-Key": "test-api-key",
	})
	c.Set("autonomous_agent_api_key", "test-api-key")
	setContextParams(c, map[string]string{
		"tenantId": testTenantID,
		"agentId":  testAutonomousAgentID,
	})

	handler.ImportAutonomousAgentTrace(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	mockPlatform.AssertExpectations(t)
}

// --- RefreshAutonomousAgentImportTrace Handler Tests ---

func TestRefreshAutonomousAgentImportTrace_NoAuthProvided(t *testing.T) {
	mockDocDB := mocks.NewMockDocDBClient()
	mockPlatform := &mocks.MockPlatformClient{}

	handler := createTestTracesHandlerWithMocks(mockDocDB, mockPlatform)

	c, w := setupTestContextWithBody(http.MethodPut, "/test", nil, nil)
	setContextParams(c, map[string]string{
		"tenantId": testTenantID,
		"agentId":  testAutonomousAgentID,
		"traceId":  testTraceID,
	})

	handler.RefreshAutonomousAgentImportTrace(c)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestRefreshAutonomousAgentImportTrace_BearerToken_Unauthorized(t *testing.T) {
	mockDocDB := mocks.NewMockDocDBClient()
	mockPlatform := &mocks.MockPlatformClient{}

	mockPlatform.On("GetAutonomousAgentConfigWithBearer", mock.Anything, testTenantID, testAutonomousAgentID, "test-bearer-token").
		Return(nil, fmt.Errorf("unauthorized: invalid token"))

	handler := createTestTracesHandlerWithMocks(mockDocDB, mockPlatform)

	c, w := setupTestContextWithBody(http.MethodPut, "/test", nil, map[string]string{
		"Authorization": "Bearer test-bearer-token",
	})
	c.Set("auth_token", "test-bearer-token")
	setContextParams(c, map[string]string{
		"tenantId": testTenantID,
		"agentId":  testAutonomousAgentID,
		"traceId":  testTraceID,
	})

	handler.RefreshAutonomousAgentImportTrace(c)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
	mockPlatform.AssertExpectations(t)
}

func TestRefreshAutonomousAgentImportTrace_BearerToken_Forbidden(t *testing.T) {
	mockDocDB := mocks.NewMockDocDBClient()
	mockPlatform := &mocks.MockPlatformClient{}

	mockPlatform.On("GetAutonomousAgentConfigWithBearer", mock.Anything, testTenantID, testAutonomousAgentID, "test-bearer-token").
		Return(nil, fmt.Errorf("forbidden: no write permission"))

	handler := createTestTracesHandlerWithMocks(mockDocDB, mockPlatform)

	c, w := setupTestContextWithBody(http.MethodPut, "/test", nil, map[string]string{
		"Authorization": "Bearer test-bearer-token",
	})
	c.Set("auth_token", "test-bearer-token")
	setContextParams(c, map[string]string{
		"tenantId": testTenantID,
		"agentId":  testAutonomousAgentID,
		"traceId":  testTraceID,
	})

	handler.RefreshAutonomousAgentImportTrace(c)

	assert.Equal(t, http.StatusForbidden, w.Code)
	mockPlatform.AssertExpectations(t)
}

func TestRefreshAutonomousAgentImportTrace_BearerToken_NotFound(t *testing.T) {
	mockDocDB := mocks.NewMockDocDBClient()
	mockPlatform := &mocks.MockPlatformClient{}

	mockPlatform.On("GetAutonomousAgentConfigWithBearer", mock.Anything, testTenantID, testAutonomousAgentID, "test-bearer-token").
		Return(nil, fmt.Errorf("not_found: agent not found"))

	handler := createTestTracesHandlerWithMocks(mockDocDB, mockPlatform)

	c, w := setupTestContextWithBody(http.MethodPut, "/test", nil, map[string]string{
		"Authorization": "Bearer test-bearer-token",
	})
	c.Set("auth_token", "test-bearer-token")
	setContextParams(c, map[string]string{
		"tenantId": testTenantID,
		"agentId":  testAutonomousAgentID,
		"traceId":  testTraceID,
	})

	handler.RefreshAutonomousAgentImportTrace(c)

	assert.Equal(t, http.StatusNotFound, w.Code)
	mockPlatform.AssertExpectations(t)
}

func TestRefreshAutonomousAgentImportTrace_APIKey_Unauthorized(t *testing.T) {
	mockDocDB := mocks.NewMockDocDBClient()
	mockPlatform := &mocks.MockPlatformClient{}

	mockPlatform.On("GetAutonomousAgentConfig", mock.Anything, testTenantID, testAutonomousAgentID, "test-api-key").
		Return(nil, fmt.Errorf("unauthorized: invalid API key"))

	handler := createTestTracesHandlerWithMocks(mockDocDB, mockPlatform)

	c, w := setupTestContextWithBody(http.MethodPut, "/test", nil, map[string]string{
		"X-Unified-UI-Autonomous-Agent-API-Key": "test-api-key",
	})
	c.Set("autonomous_agent_api_key", "test-api-key")
	setContextParams(c, map[string]string{
		"tenantId": testTenantID,
		"agentId":  testAutonomousAgentID,
		"traceId":  testTraceID,
	})

	handler.RefreshAutonomousAgentImportTrace(c)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
	mockPlatform.AssertExpectations(t)
}

func TestRefreshAutonomousAgentImportTrace_APIKey_NotFound(t *testing.T) {
	mockDocDB := mocks.NewMockDocDBClient()
	mockPlatform := &mocks.MockPlatformClient{}

	mockPlatform.On("GetAutonomousAgentConfig", mock.Anything, testTenantID, testAutonomousAgentID, "test-api-key").
		Return(nil, fmt.Errorf("not_found: agent not found"))

	handler := createTestTracesHandlerWithMocks(mockDocDB, mockPlatform)

	c, w := setupTestContextWithBody(http.MethodPut, "/test", nil, map[string]string{
		"X-Unified-UI-Autonomous-Agent-API-Key": "test-api-key",
	})
	c.Set("autonomous_agent_api_key", "test-api-key")
	setContextParams(c, map[string]string{
		"tenantId": testTenantID,
		"agentId":  testAutonomousAgentID,
		"traceId":  testTraceID,
	})

	handler.RefreshAutonomousAgentImportTrace(c)

	assert.Equal(t, http.StatusNotFound, w.Code)
	mockPlatform.AssertExpectations(t)
}

func TestRefreshAutonomousAgentImportTrace_TraceGetError(t *testing.T) {
	mockDocDB := mocks.NewMockDocDBClient()
	mockPlatform := &mocks.MockPlatformClient{}

	agentConfig := &platform.AutonomousAgentConfigResponse{
		Type:              platform.AgentTypeN8N,
		TenantID:          testTenantID,
		AutonomousAgentID: testAutonomousAgentID,
		Settings:          platform.AutonomousAgentConfigSettings{},
	}

	mockPlatform.On("GetAutonomousAgentConfig", mock.Anything, testTenantID, testAutonomousAgentID, "test-api-key").
		Return(agentConfig, nil)
	mockDocDB.GetTracesCollection().On("Get", mock.Anything, testTraceID).
		Return(nil, fmt.Errorf("database error"))

	handler := createTestTracesHandlerWithMocks(mockDocDB, mockPlatform)

	c, w := setupTestContextWithBody(http.MethodPut, "/test", nil, map[string]string{
		"X-Unified-UI-Autonomous-Agent-API-Key": "test-api-key",
	})
	c.Set("autonomous_agent_api_key", "test-api-key")
	setContextParams(c, map[string]string{
		"tenantId": testTenantID,
		"agentId":  testAutonomousAgentID,
		"traceId":  testTraceID,
	})

	handler.RefreshAutonomousAgentImportTrace(c)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
	mockPlatform.AssertExpectations(t)
}

func TestRefreshAutonomousAgentImportTrace_TraceNotFound(t *testing.T) {
	mockDocDB := mocks.NewMockDocDBClient()
	mockPlatform := &mocks.MockPlatformClient{}

	agentConfig := &platform.AutonomousAgentConfigResponse{
		Type:              platform.AgentTypeN8N,
		TenantID:          testTenantID,
		AutonomousAgentID: testAutonomousAgentID,
		Settings:          platform.AutonomousAgentConfigSettings{},
	}

	mockPlatform.On("GetAutonomousAgentConfig", mock.Anything, testTenantID, testAutonomousAgentID, "test-api-key").
		Return(agentConfig, nil)
	mockDocDB.GetTracesCollection().On("Get", mock.Anything, testTraceID).
		Return(nil, nil) // Trace not found

	handler := createTestTracesHandlerWithMocks(mockDocDB, mockPlatform)

	c, w := setupTestContextWithBody(http.MethodPut, "/test", nil, map[string]string{
		"X-Unified-UI-Autonomous-Agent-API-Key": "test-api-key",
	})
	c.Set("autonomous_agent_api_key", "test-api-key")
	setContextParams(c, map[string]string{
		"tenantId": testTenantID,
		"agentId":  testAutonomousAgentID,
		"traceId":  testTraceID,
	})

	handler.RefreshAutonomousAgentImportTrace(c)

	assert.Equal(t, http.StatusNotFound, w.Code)
	mockPlatform.AssertExpectations(t)
}

func TestRefreshAutonomousAgentImportTrace_TraceBelongsToDifferentAgent(t *testing.T) {
	mockDocDB := mocks.NewMockDocDBClient()
	mockPlatform := &mocks.MockPlatformClient{}

	agentConfig := &platform.AutonomousAgentConfigResponse{
		Type:              platform.AgentTypeN8N,
		TenantID:          testTenantID,
		AutonomousAgentID: testAutonomousAgentID,
		Settings:          platform.AutonomousAgentConfigSettings{},
	}

	trace := &models.Trace{
		ID:                testTraceID,
		TenantID:          testTenantID,
		AutonomousAgentID: "different-agent-id", // Different agent
		ReferenceID:       testExecutionID,
	}

	mockPlatform.On("GetAutonomousAgentConfig", mock.Anything, testTenantID, testAutonomousAgentID, "test-api-key").
		Return(agentConfig, nil)
	mockDocDB.GetTracesCollection().On("Get", mock.Anything, testTraceID).
		Return(trace, nil)

	handler := createTestTracesHandlerWithMocks(mockDocDB, mockPlatform)

	c, w := setupTestContextWithBody(http.MethodPut, "/test", nil, map[string]string{
		"X-Unified-UI-Autonomous-Agent-API-Key": "test-api-key",
	})
	c.Set("autonomous_agent_api_key", "test-api-key")
	setContextParams(c, map[string]string{
		"tenantId": testTenantID,
		"agentId":  testAutonomousAgentID,
		"traceId":  testTraceID,
	})

	handler.RefreshAutonomousAgentImportTrace(c)

	assert.Equal(t, http.StatusForbidden, w.Code)
	mockPlatform.AssertExpectations(t)
}

func TestRefreshAutonomousAgentImportTrace_NoExecutionID(t *testing.T) {
	mockDocDB := mocks.NewMockDocDBClient()
	mockPlatform := &mocks.MockPlatformClient{}

	agentConfig := &platform.AutonomousAgentConfigResponse{
		Type:              platform.AgentTypeN8N,
		TenantID:          testTenantID,
		AutonomousAgentID: testAutonomousAgentID,
		Settings:          platform.AutonomousAgentConfigSettings{},
	}

	trace := &models.Trace{
		ID:                testTraceID,
		TenantID:          testTenantID,
		AutonomousAgentID: testAutonomousAgentID,
		ReferenceID:       "", // No execution ID
		ReferenceMetadata: nil,
	}

	mockPlatform.On("GetAutonomousAgentConfig", mock.Anything, testTenantID, testAutonomousAgentID, "test-api-key").
		Return(agentConfig, nil)
	mockDocDB.GetTracesCollection().On("Get", mock.Anything, testTraceID).
		Return(trace, nil)

	handler := createTestTracesHandlerWithMocks(mockDocDB, mockPlatform)

	c, w := setupTestContextWithBody(http.MethodPut, "/test", nil, map[string]string{
		"X-Unified-UI-Autonomous-Agent-API-Key": "test-api-key",
	})
	c.Set("autonomous_agent_api_key", "test-api-key")
	setContextParams(c, map[string]string{
		"tenantId": testTenantID,
		"agentId":  testAutonomousAgentID,
		"traceId":  testTraceID,
	})

	handler.RefreshAutonomousAgentImportTrace(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	mockPlatform.AssertExpectations(t)
}

func TestRefreshAutonomousAgentImportTrace_NoImporterRegistered(t *testing.T) {
	mockDocDB := mocks.NewMockDocDBClient()
	mockPlatform := &mocks.MockPlatformClient{}

	agentConfig := &platform.AutonomousAgentConfigResponse{
		Type:              platform.AgentTypeN8N,
		TenantID:          testTenantID,
		AutonomousAgentID: testAutonomousAgentID,
		Settings:          platform.AutonomousAgentConfigSettings{},
	}

	trace := &models.Trace{
		ID:                testTraceID,
		TenantID:          testTenantID,
		AutonomousAgentID: testAutonomousAgentID,
		ReferenceID:       testExecutionID,
	}

	mockPlatform.On("GetAutonomousAgentConfig", mock.Anything, testTenantID, testAutonomousAgentID, "test-api-key").
		Return(agentConfig, nil)
	mockDocDB.GetTracesCollection().On("Get", mock.Anything, testTraceID).
		Return(trace, nil)

	handler := createTestTracesHandlerWithMocks(mockDocDB, mockPlatform)
	// Note: No importer registered

	c, w := setupTestContextWithBody(http.MethodPut, "/test", nil, map[string]string{
		"X-Unified-UI-Autonomous-Agent-API-Key": "test-api-key",
	})
	c.Set("autonomous_agent_api_key", "test-api-key")
	setContextParams(c, map[string]string{
		"tenantId": testTenantID,
		"agentId":  testAutonomousAgentID,
		"traceId":  testTraceID,
	})

	handler.RefreshAutonomousAgentImportTrace(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	mockPlatform.AssertExpectations(t)
}

func TestRefreshAutonomousAgentImportTrace_BackendConfigError(t *testing.T) {
	mockDocDB := mocks.NewMockDocDBClient()
	mockPlatform := &mocks.MockPlatformClient{}

	// Config without required N8N host
	agentConfig := &platform.AutonomousAgentConfigResponse{
		Type:              platform.AgentTypeN8N,
		TenantID:          testTenantID,
		AutonomousAgentID: testAutonomousAgentID,
		Settings: platform.AutonomousAgentConfigSettings{
			N8NHost: "", // Missing
		},
	}

	trace := &models.Trace{
		ID:                testTraceID,
		TenantID:          testTenantID,
		AutonomousAgentID: testAutonomousAgentID,
		ReferenceID:       testExecutionID,
	}

	mockPlatform.On("GetAutonomousAgentConfig", mock.Anything, testTenantID, testAutonomousAgentID, "test-api-key").
		Return(agentConfig, nil)
	mockDocDB.GetTracesCollection().On("Get", mock.Anything, testTraceID).
		Return(trace, nil)

	handler := createTestTracesHandlerWithMocks(mockDocDB, mockPlatform)
	handler.GetImportService().RegisterImporter(mocks.NewMockTraceImporter())

	c, w := setupTestContextWithBody(http.MethodPut, "/test", nil, map[string]string{
		"X-Unified-UI-Autonomous-Agent-API-Key": "test-api-key",
	})
	c.Set("autonomous_agent_api_key", "test-api-key")
	setContextParams(c, map[string]string{
		"tenantId": testTenantID,
		"agentId":  testAutonomousAgentID,
		"traceId":  testTraceID,
	})

	handler.RefreshAutonomousAgentImportTrace(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	mockPlatform.AssertExpectations(t)
}

func TestRefreshAutonomousAgentImportTrace_BearerToken_GetUserFallbackToSystem(t *testing.T) {
	// When GetMe fails, getUserID returns "system" as fallback, not an error
	mockDocDB := mocks.NewMockDocDBClient()
	mockPlatform := &mocks.MockPlatformClient{}

	agentConfig := &platform.AutonomousAgentConfigResponse{
		Type:              platform.AgentTypeN8N,
		TenantID:          testTenantID,
		AutonomousAgentID: testAutonomousAgentID,
		Settings: platform.AutonomousAgentConfigSettings{
			N8NHost:    "https://n8n.example.com",
			WorkflowID: "workflow-123",
			APICredentials: &platform.Credentials{
				ID:     "cred-123",
				Type:   platform.CredentialTypeN8NAPIKey,
				Secret: "test-api-key",
			},
		},
	}

	now := time.Now().UTC()
	trace := &models.Trace{
		ID:                testTraceID,
		TenantID:          testTenantID,
		AutonomousAgentID: testAutonomousAgentID,
		ContextType:       models.TraceContextAutonomousAgent,
		ReferenceID:       testExecutionID,
		CreatedAt:         now,
		UpdatedAt:         now,
	}

	mockReturnedTrace := &models.Trace{
		ID:                "mock-trace-id",
		TenantID:          testTenantID,
		AutonomousAgentID: testAutonomousAgentID,
		ContextType:       models.TraceContextAutonomousAgent,
		ReferenceID:       testExecutionID,
		CreatedAt:         now,
		UpdatedAt:         now,
	}

	mockPlatform.On("GetAutonomousAgentConfigWithBearer", mock.Anything, testTenantID, testAutonomousAgentID, "test-bearer-token").
		Return(agentConfig, nil)
	mockPlatform.On("GetMe", mock.Anything, "test-bearer-token").
		Return(nil, fmt.Errorf("failed to get user"))
	// GetMe failure leads to "system" user ID fallback, so refresh continues
	// First Get is for fetching the original trace by traceID
	mockDocDB.GetTracesCollection().On("Get", mock.Anything, testTraceID).
		Return(trace, nil)
	mockDocDB.GetTracesCollection().On("Create", mock.Anything, mock.Anything).
		Return(nil)
	// Second Get is for fetching the newly created trace by newTraceID (returned by importer)
	mockDocDB.GetTracesCollection().On("Get", mock.Anything, "mock-trace-id").
		Return(mockReturnedTrace, nil)
	mockDocDB.GetTracesCollection().On("Update", mock.Anything, mock.Anything).
		Return(nil)

	handler := createTestTracesHandlerWithMocks(mockDocDB, mockPlatform)
	handler.GetImportService().RegisterImporter(mocks.NewMockTraceImporter())

	c, w := setupTestContextWithBody(http.MethodPut, "/test", nil, map[string]string{
		"Authorization": "Bearer test-bearer-token",
	})
	c.Set("auth_token", "test-bearer-token")
	setContextParams(c, map[string]string{
		"tenantId": testTenantID,
		"agentId":  testAutonomousAgentID,
		"traceId":  testTraceID,
	})

	handler.RefreshAutonomousAgentImportTrace(c)

	// Should succeed with fallback user ID
	assert.Equal(t, http.StatusOK, w.Code)
	mockPlatform.AssertExpectations(t)
}

func TestRefreshAutonomousAgentImportTrace_Success(t *testing.T) {
	mockDocDB := mocks.NewMockDocDBClient()
	mockPlatform := &mocks.MockPlatformClient{}

	agentConfig := &platform.AutonomousAgentConfigResponse{
		Type:              platform.AgentTypeN8N,
		TenantID:          testTenantID,
		AutonomousAgentID: testAutonomousAgentID,
		Settings: platform.AutonomousAgentConfigSettings{
			N8NHost:    "https://n8n.example.com",
			WorkflowID: "workflow-123",
			APICredentials: &platform.Credentials{
				ID:     "cred-123",
				Type:   platform.CredentialTypeN8NAPIKey,
				Secret: "test-api-key",
			},
		},
	}

	now := time.Now().UTC()
	trace := &models.Trace{
		ID:                testTraceID,
		TenantID:          testTenantID,
		AutonomousAgentID: testAutonomousAgentID,
		ContextType:       models.TraceContextAutonomousAgent,
		ReferenceID:       testExecutionID,
		CreatedAt:         now,
		UpdatedAt:         now,
	}

	mockReturnedTrace := &models.Trace{
		ID:                "mock-trace-id",
		TenantID:          testTenantID,
		AutonomousAgentID: testAutonomousAgentID,
		ContextType:       models.TraceContextAutonomousAgent,
		ReferenceID:       testExecutionID,
		CreatedAt:         now,
		UpdatedAt:         now,
	}

	mockPlatform.On("GetAutonomousAgentConfig", mock.Anything, testTenantID, testAutonomousAgentID, "test-api-key").
		Return(agentConfig, nil)
	// First Get is for fetching the original trace by traceID
	mockDocDB.GetTracesCollection().On("Get", mock.Anything, testTraceID).
		Return(trace, nil)
	mockDocDB.GetTracesCollection().On("Create", mock.Anything, mock.Anything).
		Return(nil)
	// Second Get is for fetching the newly created trace by newTraceID (returned by importer)
	mockDocDB.GetTracesCollection().On("Get", mock.Anything, "mock-trace-id").
		Return(mockReturnedTrace, nil)
	mockDocDB.GetTracesCollection().On("Update", mock.Anything, mock.Anything).
		Return(nil)

	handler := createTestTracesHandlerWithMocks(mockDocDB, mockPlatform)
	handler.GetImportService().RegisterImporter(mocks.NewMockTraceImporter())

	c, w := setupTestContextWithBody(http.MethodPut, "/test", nil, map[string]string{
		"X-Unified-UI-Autonomous-Agent-API-Key": "test-api-key",
	})
	c.Set("autonomous_agent_api_key", "test-api-key")
	setContextParams(c, map[string]string{
		"tenantId": testTenantID,
		"agentId":  testAutonomousAgentID,
		"traceId":  testTraceID,
	})

	handler.RefreshAutonomousAgentImportTrace(c)

	assert.Equal(t, http.StatusOK, w.Code)

	var response dto.ImportTraceResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)
	assert.NotEmpty(t, response.ID)

	mockPlatform.AssertExpectations(t)
}

// --- ImportConversationTrace Tests ---

func TestImportConversationTrace_GetConversationUnauthorized(t *testing.T) {
	mockDocDB := mocks.NewMockDocDBClient()
	mockPlatform := &mocks.MockPlatformClient{}

	mockPlatform.On("GetConversation", mock.Anything, testTenantID, "conv-123", "test-token").
		Return(nil, fmt.Errorf("unauthorized: invalid token"))

	handler := createTestTracesHandlerWithMocks(mockDocDB, mockPlatform)

	c, w := setupTestContextWithBody(http.MethodPut, "/test", nil, map[string]string{
		"Authorization": "Bearer test-token",
	})
	c.Set("auth_token", "test-token")
	setContextParams(c, map[string]string{
		"tenantId":       testTenantID,
		"conversationId": "conv-123",
	})

	handler.ImportConversationTrace(c)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
	mockPlatform.AssertExpectations(t)
}

func TestImportConversationTrace_GetConversationForbidden(t *testing.T) {
	mockDocDB := mocks.NewMockDocDBClient()
	mockPlatform := &mocks.MockPlatformClient{}

	mockPlatform.On("GetConversation", mock.Anything, testTenantID, "conv-123", "test-token").
		Return(nil, fmt.Errorf("forbidden: access denied"))

	handler := createTestTracesHandlerWithMocks(mockDocDB, mockPlatform)

	c, w := setupTestContextWithBody(http.MethodPut, "/test", nil, map[string]string{
		"Authorization": "Bearer test-token",
	})
	c.Set("auth_token", "test-token")
	setContextParams(c, map[string]string{
		"tenantId":       testTenantID,
		"conversationId": "conv-123",
	})

	handler.ImportConversationTrace(c)

	assert.Equal(t, http.StatusForbidden, w.Code)
	mockPlatform.AssertExpectations(t)
}

func TestImportConversationTrace_GetConversationNotFound(t *testing.T) {
	mockDocDB := mocks.NewMockDocDBClient()
	mockPlatform := &mocks.MockPlatformClient{}

	mockPlatform.On("GetConversation", mock.Anything, testTenantID, "conv-123", "test-token").
		Return(nil, fmt.Errorf("not_found: conversation not found"))

	handler := createTestTracesHandlerWithMocks(mockDocDB, mockPlatform)

	c, w := setupTestContextWithBody(http.MethodPut, "/test", nil, map[string]string{
		"Authorization": "Bearer test-token",
	})
	c.Set("auth_token", "test-token")
	setContextParams(c, map[string]string{
		"tenantId":       testTenantID,
		"conversationId": "conv-123",
	})

	handler.ImportConversationTrace(c)

	assert.Equal(t, http.StatusNotFound, w.Code)
	mockPlatform.AssertExpectations(t)
}

func TestImportConversationTrace_GetConversationInternalError(t *testing.T) {
	mockDocDB := mocks.NewMockDocDBClient()
	mockPlatform := &mocks.MockPlatformClient{}

	mockPlatform.On("GetConversation", mock.Anything, testTenantID, "conv-123", "test-token").
		Return(nil, fmt.Errorf("database error"))

	handler := createTestTracesHandlerWithMocks(mockDocDB, mockPlatform)

	c, w := setupTestContextWithBody(http.MethodPut, "/test", nil, map[string]string{
		"Authorization": "Bearer test-token",
	})
	c.Set("auth_token", "test-token")
	setContextParams(c, map[string]string{
		"tenantId":       testTenantID,
		"conversationId": "conv-123",
	})

	handler.ImportConversationTrace(c)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
	mockPlatform.AssertExpectations(t)
}

func TestImportConversationTrace_GetChatAgentConfigError(t *testing.T) {
	mockDocDB := mocks.NewMockDocDBClient()
	mockPlatform := &mocks.MockPlatformClient{}

	conversation := &platform.ConversationResponse{
		ID:                "conv-123",
		TenantID:          testTenantID,
		ChatAgentID:       "chat-agent-123",
		ExtConversationID: "ext-conv-456",
	}

	mockPlatform.On("GetConversation", mock.Anything, testTenantID, "conv-123", "test-token").
		Return(conversation, nil)
	mockPlatform.On("GetChatAgentConfig", mock.Anything, testTenantID, "chat-agent-123", "test-token").
		Return(nil, fmt.Errorf("config error"))

	handler := createTestTracesHandlerWithMocks(mockDocDB, mockPlatform)

	c, w := setupTestContextWithBody(http.MethodPut, "/test", nil, map[string]string{
		"Authorization": "Bearer test-token",
	})
	c.Set("auth_token", "test-token")
	setContextParams(c, map[string]string{
		"tenantId":       testTenantID,
		"conversationId": "conv-123",
	})

	handler.ImportConversationTrace(c)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
	mockPlatform.AssertExpectations(t)
}

func TestImportConversationTrace_GetMeError(t *testing.T) {
	mockDocDB := mocks.NewMockDocDBClient()
	mockPlatform := &mocks.MockPlatformClient{}

	conversation := &platform.ConversationResponse{
		ID:                "conv-123",
		TenantID:          testTenantID,
		ChatAgentID:       "chat-agent-123",
		ExtConversationID: "ext-conv-456",
	}

	appConfig := &platform.ChatAgentConfigResponse{
		Type:        platform.AgentTypeFoundry,
		TenantID:    testTenantID,
		ChatAgentID: "chat-agent-123",
		Settings:    platform.AgentSettings{},
	}

	mockPlatform.On("GetConversation", mock.Anything, testTenantID, "conv-123", "test-token").
		Return(conversation, nil)
	mockPlatform.On("GetChatAgentConfig", mock.Anything, testTenantID, "chat-agent-123", "test-token").
		Return(appConfig, nil)
	mockPlatform.On("GetMe", mock.Anything, "test-token").
		Return(nil, fmt.Errorf("user info error"))

	handler := createTestTracesHandlerWithMocks(mockDocDB, mockPlatform)

	c, w := setupTestContextWithBody(http.MethodPut, "/test", nil, map[string]string{
		"Authorization": "Bearer test-token",
	})
	c.Set("auth_token", "test-token")
	setContextParams(c, map[string]string{
		"tenantId":       testTenantID,
		"conversationId": "conv-123",
	})

	handler.ImportConversationTrace(c)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
	mockPlatform.AssertExpectations(t)
}

func TestImportConversationTrace_UnsupportedAgentType(t *testing.T) {
	mockDocDB := mocks.NewMockDocDBClient()
	mockPlatform := &mocks.MockPlatformClient{}

	conversation := &platform.ConversationResponse{
		ID:                "conv-123",
		TenantID:          testTenantID,
		ChatAgentID:       "chat-agent-123",
		ExtConversationID: "ext-conv-456",
	}

	appConfig := &platform.ChatAgentConfigResponse{
		Type:        platform.AgentType("unsupported"),
		TenantID:    testTenantID,
		ChatAgentID: "chat-agent-123",
		Settings:    platform.AgentSettings{},
	}

	userInfo := &platform.UserInfo{
		ID:   testUserID,
		Mail: "test@example.com",
	}

	mockPlatform.On("GetConversation", mock.Anything, testTenantID, "conv-123", "test-token").
		Return(conversation, nil)
	mockPlatform.On("GetChatAgentConfig", mock.Anything, testTenantID, "chat-agent-123", "test-token").
		Return(appConfig, nil)
	mockPlatform.On("GetMe", mock.Anything, "test-token").
		Return(userInfo, nil)

	handler := createTestTracesHandlerWithMocks(mockDocDB, mockPlatform)
	// No importer registered

	c, w := setupTestContextWithBody(http.MethodPut, "/test", nil, map[string]string{
		"Authorization": "Bearer test-token",
	})
	c.Set("auth_token", "test-token")
	setContextParams(c, map[string]string{
		"tenantId":       testTenantID,
		"conversationId": "conv-123",
	})

	handler.ImportConversationTrace(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	mockPlatform.AssertExpectations(t)
}

func TestImportConversationTrace_N8NNotImplemented(t *testing.T) {
	mockDocDB := mocks.NewMockDocDBClient()
	mockPlatform := &mocks.MockPlatformClient{}

	conversation := &platform.ConversationResponse{
		ID:                "conv-123",
		TenantID:          testTenantID,
		ChatAgentID:       "chat-agent-123",
		ExtConversationID: "ext-conv-456",
	}

	appConfig := &platform.ChatAgentConfigResponse{
		Type:        platform.AgentTypeN8N,
		TenantID:    testTenantID,
		ChatAgentID: "chat-agent-123",
		Settings:    platform.AgentSettings{},
	}

	userInfo := &platform.UserInfo{
		ID:   testUserID,
		Mail: "test@example.com",
	}

	mockPlatform.On("GetConversation", mock.Anything, testTenantID, "conv-123", "test-token").
		Return(conversation, nil)
	mockPlatform.On("GetChatAgentConfig", mock.Anything, testTenantID, "chat-agent-123", "test-token").
		Return(appConfig, nil)
	mockPlatform.On("GetMe", mock.Anything, "test-token").
		Return(userInfo, nil)

	handler := createTestTracesHandlerWithMocks(mockDocDB, mockPlatform)
	handler.GetImportService().RegisterImporter(mocks.NewMockTraceImporter())

	c, w := setupTestContextWithBody(http.MethodPut, "/test", nil, map[string]string{
		"Authorization": "Bearer test-token",
	})
	c.Set("auth_token", "test-token")
	setContextParams(c, map[string]string{
		"tenantId":       testTenantID,
		"conversationId": "conv-123",
	})

	handler.ImportConversationTrace(c)

	// Should fail because buildN8NConfig returns not implemented error
	assert.Equal(t, http.StatusBadRequest, w.Code)
	mockPlatform.AssertExpectations(t)
}
