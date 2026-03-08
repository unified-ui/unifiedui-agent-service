// Package handlers_test provides extended unit tests for trace handlers focusing on edge cases and error paths.
package handlers_test

import (
	"errors"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"github.com/unifiedui/agent-service/internal/api/dto"
	"github.com/unifiedui/agent-service/internal/api/handlers"
	"github.com/unifiedui/agent-service/internal/domain/models"
	"github.com/unifiedui/agent-service/internal/services/platform"
	"github.com/unifiedui/agent-service/internal/services/traceimport"
	"github.com/unifiedui/agent-service/tests/mocks"
	"github.com/unifiedui/agent-service/tests/testutils"
)

// =============================================================================
// CreateTrace - Extended Tests for Error Paths
// =============================================================================

func TestTracesHandler_CreateTrace_InvalidRequestBody(t *testing.T) {
	mockDocDB := mocks.NewMockDocDBClient()
	mockPlatform := &mocks.MockPlatformClient{}

	handler := createTracesHandler(mockDocDB, mockPlatform)

	router := testutils.SetupTestRouter()
	router.POST("/tenants/:tenantId/traces", handler.CreateTrace)

	// Send invalid JSON
	w := testutils.PerformRequest(router, "POST", "/tenants/"+testutils.TestTenantID+"/traces", "invalid json", nil)

	testutils.AssertStatusCode(t, http.StatusBadRequest, w)
}

func TestTracesHandler_CreateTrace_ConversationContext_MissingBearerToken(t *testing.T) {
	mockDocDB := mocks.NewMockDocDBClient()
	mockPlatform := &mocks.MockPlatformClient{}

	createReq := dto.CreateTraceRequest{
		ChatAgentID:    testutils.TestChatAgentID,
		ConversationID: testutils.TestConversationID,
		ReferenceID:    "workflow-123",
	}

	handler := createTracesHandler(mockDocDB, mockPlatform)

	router := testutils.SetupTestRouter()
	// No auth middleware - simulates missing token
	router.POST("/tenants/:tenantId/traces", handler.CreateTrace)

	// Execute without Authorization header
	w := testutils.PerformRequest(router, "POST", "/tenants/"+testutils.TestTenantID+"/traces", createReq, nil)

	testutils.AssertStatusCode(t, http.StatusUnauthorized, w)
}

func TestTracesHandler_CreateTrace_ConversationContext_GetUserIDError(t *testing.T) {
	mockDocDB := mocks.NewMockDocDBClient()
	mockPlatform := &mocks.MockPlatformClient{}

	createReq := dto.CreateTraceRequest{
		ChatAgentID:    testutils.TestChatAgentID,
		ConversationID: testutils.TestConversationID,
	}

	// Mock GetMe to return error - getUserID returns "system" on error, so it continues
	mockPlatform.On("GetMe", mock.Anything, mock.Anything).Return(nil, errors.New("user service error"))
	mockPlatform.On("ValidateConversation", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil)
	mockDocDB.GetTracesCollection().On("GetByConversation", mock.Anything, mock.Anything, mock.Anything).Return(nil, nil)
	mockDocDB.GetTracesCollection().On("Create", mock.Anything, mock.Anything).Return(nil)

	handler := createTracesHandler(mockDocDB, mockPlatform)

	router := testutils.SetupTestRouter()
	router.Use(testFlexibleAuthMiddleware())
	router.POST("/tenants/:tenantId/traces", handler.CreateTrace)

	headers := map[string]string{"Authorization": "Bearer test-token"}
	w := testutils.PerformRequest(router, "POST", "/tenants/"+testutils.TestTenantID+"/traces", createReq, headers)

	// Should succeed but with "system" as user
	testutils.AssertStatusCode(t, http.StatusCreated, w)
	mockPlatform.AssertExpectations(t)
}

func TestTracesHandler_CreateTrace_ConversationContext_ValidationUnauthorized(t *testing.T) {
	mockDocDB := mocks.NewMockDocDBClient()
	mockPlatform := &mocks.MockPlatformClient{}

	createReq := dto.CreateTraceRequest{
		ChatAgentID:    testutils.TestChatAgentID,
		ConversationID: testutils.TestConversationID,
	}

	mockPlatform.On("GetMe", mock.Anything, mock.Anything).Return(&platform.UserInfo{ID: testutils.TestUserID}, nil)
	mockPlatform.On("ValidateConversation", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(errors.New("unauthorized: invalid token"))

	handler := createTracesHandler(mockDocDB, mockPlatform)

	router := testutils.SetupTestRouter()
	router.Use(testFlexibleAuthMiddleware())
	router.POST("/tenants/:tenantId/traces", handler.CreateTrace)

	headers := map[string]string{"Authorization": "Bearer invalid-token"}
	w := testutils.PerformRequest(router, "POST", "/tenants/"+testutils.TestTenantID+"/traces", createReq, headers)

	testutils.AssertStatusCode(t, http.StatusUnauthorized, w)
}

func TestTracesHandler_CreateTrace_ConversationContext_ValidationForbidden(t *testing.T) {
	mockDocDB := mocks.NewMockDocDBClient()
	mockPlatform := &mocks.MockPlatformClient{}

	createReq := dto.CreateTraceRequest{
		ChatAgentID:    testutils.TestChatAgentID,
		ConversationID: testutils.TestConversationID,
	}

	mockPlatform.On("GetMe", mock.Anything, mock.Anything).Return(&platform.UserInfo{ID: testutils.TestUserID}, nil)
	mockPlatform.On("ValidateConversation", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(errors.New("forbidden: access denied"))

	handler := createTracesHandler(mockDocDB, mockPlatform)

	router := testutils.SetupTestRouter()
	router.Use(testFlexibleAuthMiddleware())
	router.POST("/tenants/:tenantId/traces", handler.CreateTrace)

	headers := map[string]string{"Authorization": "Bearer test-token"}
	w := testutils.PerformRequest(router, "POST", "/tenants/"+testutils.TestTenantID+"/traces", createReq, headers)

	testutils.AssertStatusCode(t, http.StatusForbidden, w)
}

func TestTracesHandler_CreateTrace_ConversationContext_ValidationNotFound(t *testing.T) {
	mockDocDB := mocks.NewMockDocDBClient()
	mockPlatform := &mocks.MockPlatformClient{}

	createReq := dto.CreateTraceRequest{
		ChatAgentID:    testutils.TestChatAgentID,
		ConversationID: testutils.TestConversationID,
	}

	mockPlatform.On("GetMe", mock.Anything, mock.Anything).Return(&platform.UserInfo{ID: testutils.TestUserID}, nil)
	mockPlatform.On("ValidateConversation", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(errors.New("not_found: conversation not found"))

	handler := createTracesHandler(mockDocDB, mockPlatform)

	router := testutils.SetupTestRouter()
	router.Use(testFlexibleAuthMiddleware())
	router.POST("/tenants/:tenantId/traces", handler.CreateTrace)

	headers := map[string]string{"Authorization": "Bearer test-token"}
	w := testutils.PerformRequest(router, "POST", "/tenants/"+testutils.TestTenantID+"/traces", createReq, headers)

	testutils.AssertStatusCode(t, http.StatusNotFound, w)
}

func TestTracesHandler_CreateTrace_ConversationContext_ValidationInternalError(t *testing.T) {
	mockDocDB := mocks.NewMockDocDBClient()
	mockPlatform := &mocks.MockPlatformClient{}

	createReq := dto.CreateTraceRequest{
		ChatAgentID:    testutils.TestChatAgentID,
		ConversationID: testutils.TestConversationID,
	}

	mockPlatform.On("GetMe", mock.Anything, mock.Anything).Return(&platform.UserInfo{ID: testutils.TestUserID}, nil)
	mockPlatform.On("ValidateConversation", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(errors.New("database connection error"))

	handler := createTracesHandler(mockDocDB, mockPlatform)

	router := testutils.SetupTestRouter()
	router.Use(testFlexibleAuthMiddleware())
	router.POST("/tenants/:tenantId/traces", handler.CreateTrace)

	headers := map[string]string{"Authorization": "Bearer test-token"}
	w := testutils.PerformRequest(router, "POST", "/tenants/"+testutils.TestTenantID+"/traces", createReq, headers)

	testutils.AssertStatusCode(t, http.StatusInternalServerError, w)
}

func TestTracesHandler_CreateTrace_ConversationContext_GetByConversationDBError(t *testing.T) {
	mockDocDB := mocks.NewMockDocDBClient()
	mockPlatform := &mocks.MockPlatformClient{}

	createReq := dto.CreateTraceRequest{
		ChatAgentID:    testutils.TestChatAgentID,
		ConversationID: testutils.TestConversationID,
	}

	mockPlatform.On("GetMe", mock.Anything, mock.Anything).Return(&platform.UserInfo{ID: testutils.TestUserID}, nil)
	mockPlatform.On("ValidateConversation", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil)
	mockDocDB.GetTracesCollection().On("GetByConversation", mock.Anything, mock.Anything, mock.Anything).Return(nil, errors.New("db connection error"))

	handler := createTracesHandler(mockDocDB, mockPlatform)

	router := testutils.SetupTestRouter()
	router.Use(testFlexibleAuthMiddleware())
	router.POST("/tenants/:tenantId/traces", handler.CreateTrace)

	headers := map[string]string{"Authorization": "Bearer test-token"}
	w := testutils.PerformRequest(router, "POST", "/tenants/"+testutils.TestTenantID+"/traces", createReq, headers)

	testutils.AssertStatusCode(t, http.StatusInternalServerError, w)
}

func TestTracesHandler_CreateTrace_ConversationContext_CreateDBError(t *testing.T) {
	mockDocDB := mocks.NewMockDocDBClient()
	mockPlatform := &mocks.MockPlatformClient{}

	createReq := dto.CreateTraceRequest{
		ChatAgentID:    testutils.TestChatAgentID,
		ConversationID: testutils.TestConversationID,
	}

	mockPlatform.On("GetMe", mock.Anything, mock.Anything).Return(&platform.UserInfo{ID: testutils.TestUserID}, nil)
	mockPlatform.On("ValidateConversation", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil)
	mockDocDB.GetTracesCollection().On("GetByConversation", mock.Anything, mock.Anything, mock.Anything).Return(nil, nil)
	mockDocDB.GetTracesCollection().On("Create", mock.Anything, mock.Anything).Return(errors.New("db write error"))

	handler := createTracesHandler(mockDocDB, mockPlatform)

	router := testutils.SetupTestRouter()
	router.Use(testFlexibleAuthMiddleware())
	router.POST("/tenants/:tenantId/traces", handler.CreateTrace)

	headers := map[string]string{"Authorization": "Bearer test-token"}
	w := testutils.PerformRequest(router, "POST", "/tenants/"+testutils.TestTenantID+"/traces", createReq, headers)

	testutils.AssertStatusCode(t, http.StatusInternalServerError, w)
}

func TestTracesHandler_CreateTrace_AutonomousAgent_WithBearerToken(t *testing.T) {
	mockDocDB := mocks.NewMockDocDBClient()
	mockPlatform := &mocks.MockPlatformClient{}

	createReq := dto.CreateTraceRequest{
		AutonomousAgentID: "auto-agent-123",
		ReferenceID:       "scheduled-run-456",
	}

	// Bearer token auth flow for autonomous agent
	mockPlatform.On("ValidateAutonomousAgent", mock.Anything, testutils.TestTenantID, "auto-agent-123", "test-bearer-token").Return(nil)
	mockPlatform.On("GetMe", mock.Anything, mock.Anything).Return(&platform.UserInfo{ID: testutils.TestUserID}, nil)
	mockDocDB.GetTracesCollection().On("Create", mock.Anything, mock.Anything).Return(nil)

	handler := createTracesHandler(mockDocDB, mockPlatform)

	router := testutils.SetupTestRouter()
	router.Use(testFlexibleAuthMiddleware())
	router.POST("/tenants/:tenantId/traces", handler.CreateTrace)

	headers := map[string]string{"Authorization": "Bearer test-bearer-token"}
	w := testutils.PerformRequest(router, "POST", "/tenants/"+testutils.TestTenantID+"/traces", createReq, headers)

	testutils.AssertStatusCode(t, http.StatusCreated, w)
	mockPlatform.AssertExpectations(t)
}

func TestTracesHandler_CreateTrace_AutonomousAgent_BearerToken_Unauthorized(t *testing.T) {
	mockDocDB := mocks.NewMockDocDBClient()
	mockPlatform := &mocks.MockPlatformClient{}

	createReq := dto.CreateTraceRequest{
		AutonomousAgentID: "auto-agent-123",
	}

	mockPlatform.On("ValidateAutonomousAgent", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(errors.New("unauthorized: invalid token"))

	handler := createTracesHandler(mockDocDB, mockPlatform)

	router := testutils.SetupTestRouter()
	router.Use(testFlexibleAuthMiddleware())
	router.POST("/tenants/:tenantId/traces", handler.CreateTrace)

	headers := map[string]string{"Authorization": "Bearer invalid-token"}
	w := testutils.PerformRequest(router, "POST", "/tenants/"+testutils.TestTenantID+"/traces", createReq, headers)

	testutils.AssertStatusCode(t, http.StatusUnauthorized, w)
}

func TestTracesHandler_CreateTrace_AutonomousAgent_BearerToken_Forbidden(t *testing.T) {
	mockDocDB := mocks.NewMockDocDBClient()
	mockPlatform := &mocks.MockPlatformClient{}

	createReq := dto.CreateTraceRequest{
		AutonomousAgentID: "auto-agent-123",
	}

	mockPlatform.On("ValidateAutonomousAgent", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(errors.New("forbidden: no permission"))

	handler := createTracesHandler(mockDocDB, mockPlatform)

	router := testutils.SetupTestRouter()
	router.Use(testFlexibleAuthMiddleware())
	router.POST("/tenants/:tenantId/traces", handler.CreateTrace)

	headers := map[string]string{"Authorization": "Bearer test-token"}
	w := testutils.PerformRequest(router, "POST", "/tenants/"+testutils.TestTenantID+"/traces", createReq, headers)

	testutils.AssertStatusCode(t, http.StatusForbidden, w)
}

func TestTracesHandler_CreateTrace_AutonomousAgent_BearerToken_NotFound(t *testing.T) {
	mockDocDB := mocks.NewMockDocDBClient()
	mockPlatform := &mocks.MockPlatformClient{}

	createReq := dto.CreateTraceRequest{
		AutonomousAgentID: "non-existent-agent",
	}

	mockPlatform.On("ValidateAutonomousAgent", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(errors.New("not_found: agent not found"))

	handler := createTracesHandler(mockDocDB, mockPlatform)

	router := testutils.SetupTestRouter()
	router.Use(testFlexibleAuthMiddleware())
	router.POST("/tenants/:tenantId/traces", handler.CreateTrace)

	headers := map[string]string{"Authorization": "Bearer test-token"}
	w := testutils.PerformRequest(router, "POST", "/tenants/"+testutils.TestTenantID+"/traces", createReq, headers)

	testutils.AssertStatusCode(t, http.StatusNotFound, w)
}

func TestTracesHandler_CreateTrace_AutonomousAgent_BearerToken_InternalError(t *testing.T) {
	mockDocDB := mocks.NewMockDocDBClient()
	mockPlatform := &mocks.MockPlatformClient{}

	createReq := dto.CreateTraceRequest{
		AutonomousAgentID: "auto-agent-123",
	}

	mockPlatform.On("ValidateAutonomousAgent", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(errors.New("database error"))

	handler := createTracesHandler(mockDocDB, mockPlatform)

	router := testutils.SetupTestRouter()
	router.Use(testFlexibleAuthMiddleware())
	router.POST("/tenants/:tenantId/traces", handler.CreateTrace)

	headers := map[string]string{"Authorization": "Bearer test-token"}
	w := testutils.PerformRequest(router, "POST", "/tenants/"+testutils.TestTenantID+"/traces", createReq, headers)

	testutils.AssertStatusCode(t, http.StatusInternalServerError, w)
}

func TestTracesHandler_CreateTrace_AutonomousAgent_APIKey_Unauthorized(t *testing.T) {
	mockDocDB := mocks.NewMockDocDBClient()
	mockPlatform := &mocks.MockPlatformClient{}

	createReq := dto.CreateTraceRequest{
		AutonomousAgentID: "auto-agent-123",
	}

	mockPlatform.On("ValidateAutonomousAgentAPIKey", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(errors.New("unauthorized: invalid api key"))

	handler := createTracesHandler(mockDocDB, mockPlatform)

	router := testutils.SetupTestRouter()
	router.Use(testFlexibleAuthMiddleware())
	router.POST("/tenants/:tenantId/traces", handler.CreateTrace)

	headers := map[string]string{"X-Unified-UI-Autonomous-Agent-API-Key": "invalid-key"}
	w := testutils.PerformRequest(router, "POST", "/tenants/"+testutils.TestTenantID+"/traces", createReq, headers)

	testutils.AssertStatusCode(t, http.StatusUnauthorized, w)
}

func TestTracesHandler_CreateTrace_AutonomousAgent_APIKey_NotFound(t *testing.T) {
	mockDocDB := mocks.NewMockDocDBClient()
	mockPlatform := &mocks.MockPlatformClient{}

	createReq := dto.CreateTraceRequest{
		AutonomousAgentID: "non-existent-agent",
	}

	mockPlatform.On("ValidateAutonomousAgentAPIKey", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(errors.New("not_found: agent not found"))

	handler := createTracesHandler(mockDocDB, mockPlatform)

	router := testutils.SetupTestRouter()
	router.Use(testFlexibleAuthMiddleware())
	router.POST("/tenants/:tenantId/traces", handler.CreateTrace)

	headers := map[string]string{"X-Unified-UI-Autonomous-Agent-API-Key": "test-key"}
	w := testutils.PerformRequest(router, "POST", "/tenants/"+testutils.TestTenantID+"/traces", createReq, headers)

	testutils.AssertStatusCode(t, http.StatusNotFound, w)
}

func TestTracesHandler_CreateTrace_AutonomousAgent_APIKey_InternalError(t *testing.T) {
	mockDocDB := mocks.NewMockDocDBClient()
	mockPlatform := &mocks.MockPlatformClient{}

	createReq := dto.CreateTraceRequest{
		AutonomousAgentID: "auto-agent-123",
	}

	mockPlatform.On("ValidateAutonomousAgentAPIKey", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(errors.New("database connection failed"))

	handler := createTracesHandler(mockDocDB, mockPlatform)

	router := testutils.SetupTestRouter()
	router.Use(testFlexibleAuthMiddleware())
	router.POST("/tenants/:tenantId/traces", handler.CreateTrace)

	headers := map[string]string{"X-Unified-UI-Autonomous-Agent-API-Key": "test-key"}
	w := testutils.PerformRequest(router, "POST", "/tenants/"+testutils.TestTenantID+"/traces", createReq, headers)

	testutils.AssertStatusCode(t, http.StatusInternalServerError, w)
}

func TestTracesHandler_CreateTrace_AutonomousAgent_NoAuth(t *testing.T) {
	mockDocDB := mocks.NewMockDocDBClient()
	mockPlatform := &mocks.MockPlatformClient{}

	createReq := dto.CreateTraceRequest{
		AutonomousAgentID: "auto-agent-123",
	}

	handler := createTracesHandler(mockDocDB, mockPlatform)

	router := testutils.SetupTestRouter()
	// No auth middleware
	router.POST("/tenants/:tenantId/traces", handler.CreateTrace)

	// No Authorization header, no API key
	w := testutils.PerformRequest(router, "POST", "/tenants/"+testutils.TestTenantID+"/traces", createReq, nil)

	testutils.AssertStatusCode(t, http.StatusUnauthorized, w)
}

func TestTracesHandler_CreateTrace_WithProvidedID(t *testing.T) {
	mockDocDB := mocks.NewMockDocDBClient()
	mockPlatform := &mocks.MockPlatformClient{}

	customID := "custom-trace-id-12345"
	createReq := dto.CreateTraceRequest{
		ID:                customID,
		AutonomousAgentID: "auto-agent-123",
	}

	mockPlatform.On("ValidateAutonomousAgentAPIKey", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil)
	mockDocDB.GetTracesCollection().On("Create", mock.Anything, mock.MatchedBy(func(trace *models.Trace) bool {
		return trace.ID == customID
	})).Return(nil)

	handler := createTracesHandler(mockDocDB, mockPlatform)

	router := testutils.SetupTestRouter()
	router.Use(testFlexibleAuthMiddleware())
	router.POST("/tenants/:tenantId/traces", handler.CreateTrace)

	headers := map[string]string{"X-Unified-UI-Autonomous-Agent-API-Key": "test-key"}
	w := testutils.PerformRequest(router, "POST", "/tenants/"+testutils.TestTenantID+"/traces", createReq, headers)

	testutils.AssertStatusCode(t, http.StatusCreated, w)

	var response dto.CreateTraceResponse
	testutils.ParseJSONResponse(t, w, &response)
	assert.Equal(t, customID, response.ID)
}

// =============================================================================
// AddNodes - Extended Tests for Error Paths
// =============================================================================

func TestTracesHandler_AddNodes_InvalidRequestBody(t *testing.T) {
	mockDocDB := mocks.NewMockDocDBClient()
	mockPlatform := &mocks.MockPlatformClient{}

	handler := createTracesHandler(mockDocDB, mockPlatform)

	router := testutils.SetupTestRouter()
	router.POST("/tenants/:tenantId/traces/:traceId/nodes", handler.AddNodes)

	w := testutils.PerformRequest(router, "POST", "/tenants/"+testutils.TestTenantID+"/traces/trace-id/nodes", "invalid json", nil)

	testutils.AssertStatusCode(t, http.StatusBadRequest, w)
}

func TestTracesHandler_AddNodes_DBErrorOnGet(t *testing.T) {
	mockDocDB := mocks.NewMockDocDBClient()
	mockPlatform := &mocks.MockPlatformClient{}

	addNodesReq := dto.AddNodesRequest{
		Nodes: []dto.TraceNodeRequest{
			{ID: "node-1", Name: "test", Type: "llm", Status: "completed"},
		},
	}

	mockDocDB.GetTracesCollection().On("Get", mock.Anything, mock.Anything).Return(nil, errors.New("db connection error"))

	handler := createTracesHandler(mockDocDB, mockPlatform)

	router := testutils.SetupTestRouter()
	router.POST("/tenants/:tenantId/traces/:traceId/nodes", handler.AddNodes)

	w := testutils.PerformRequest(router, "POST", "/tenants/"+testutils.TestTenantID+"/traces/trace-id/nodes", addNodesReq, nil)

	testutils.AssertStatusCode(t, http.StatusInternalServerError, w)
}

func TestTracesHandler_AddNodes_TraceInDifferentTenant(t *testing.T) {
	mockDocDB := mocks.NewMockDocDBClient()
	mockPlatform := &mocks.MockPlatformClient{}

	existingTrace := testutils.NewTestTrace()
	existingTrace.TenantID = "different-tenant" // Different tenant

	addNodesReq := dto.AddNodesRequest{
		Nodes: []dto.TraceNodeRequest{
			{ID: "node-1", Name: "test", Type: "llm", Status: "completed"},
		},
	}

	mockDocDB.GetTracesCollection().On("Get", mock.Anything, mock.Anything).Return(existingTrace, nil)

	handler := createTracesHandler(mockDocDB, mockPlatform)

	router := testutils.SetupTestRouter()
	router.POST("/tenants/:tenantId/traces/:traceId/nodes", handler.AddNodes)

	w := testutils.PerformRequest(router, "POST", "/tenants/"+testutils.TestTenantID+"/traces/"+existingTrace.ID+"/nodes", addNodesReq, nil)

	testutils.AssertStatusCode(t, http.StatusNotFound, w)
}

func TestTracesHandler_AddNodes_NoAuthentication(t *testing.T) {
	mockDocDB := mocks.NewMockDocDBClient()
	mockPlatform := &mocks.MockPlatformClient{}

	existingTrace := testutils.NewTestTrace()
	existingTrace.ContextType = models.TraceContextConversation

	addNodesReq := dto.AddNodesRequest{
		Nodes: []dto.TraceNodeRequest{
			{ID: "node-1", Name: "test", Type: "llm", Status: "completed"},
		},
	}

	mockDocDB.GetTracesCollection().On("Get", mock.Anything, mock.Anything).Return(existingTrace, nil)

	handler := createTracesHandler(mockDocDB, mockPlatform)

	router := testutils.SetupTestRouter()
	// No auth middleware
	router.POST("/tenants/:tenantId/traces/:traceId/nodes", handler.AddNodes)

	// No auth headers
	w := testutils.PerformRequest(router, "POST", "/tenants/"+testutils.TestTenantID+"/traces/"+existingTrace.ID+"/nodes", addNodesReq, nil)

	testutils.AssertStatusCode(t, http.StatusUnauthorized, w)
}

func TestTracesHandler_AddNodes_DBErrorOnAddNodes(t *testing.T) {
	mockDocDB := mocks.NewMockDocDBClient()
	mockPlatform := &mocks.MockPlatformClient{}

	existingTrace := testutils.NewTestTrace()

	addNodesReq := dto.AddNodesRequest{
		Nodes: []dto.TraceNodeRequest{
			{ID: "node-1", Name: "test", Type: "llm", Status: "completed"},
		},
	}

	mockPlatform.On("GetMe", mock.Anything, mock.Anything).Return(&platform.UserInfo{ID: testutils.TestUserID}, nil)
	mockDocDB.GetTracesCollection().On("Get", mock.Anything, mock.Anything).Return(existingTrace, nil)
	mockDocDB.GetTracesCollection().On("AddNodes", mock.Anything, mock.Anything, mock.Anything).Return(errors.New("db write error"))

	handler := createTracesHandler(mockDocDB, mockPlatform)

	router := testutils.SetupTestRouter()
	router.Use(testFlexibleAuthMiddleware())
	router.POST("/tenants/:tenantId/traces/:traceId/nodes", handler.AddNodes)

	headers := map[string]string{"Authorization": "Bearer test-token"}
	w := testutils.PerformRequest(router, "POST", "/tenants/"+testutils.TestTenantID+"/traces/"+existingTrace.ID+"/nodes", addNodesReq, headers)

	testutils.AssertStatusCode(t, http.StatusInternalServerError, w)
}

func TestTracesHandler_AddNodes_WithAPIKeyForAutonomousAgent(t *testing.T) {
	mockDocDB := mocks.NewMockDocDBClient()
	mockPlatform := &mocks.MockPlatformClient{}

	existingTrace := testutils.NewTestAutonomousAgentTrace()

	addNodesReq := dto.AddNodesRequest{
		Nodes: []dto.TraceNodeRequest{
			{ID: "node-1", Name: "test", Type: "llm", Status: "completed"},
		},
	}

	mockPlatform.On("ValidateAutonomousAgentAPIKey", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil)
	mockDocDB.GetTracesCollection().On("Get", mock.Anything, mock.Anything).Return(existingTrace, nil)
	mockDocDB.GetTracesCollection().On("AddNodes", mock.Anything, mock.Anything, mock.Anything).Return(nil)

	handler := createTracesHandler(mockDocDB, mockPlatform)

	router := testutils.SetupTestRouter()
	router.Use(testFlexibleAuthMiddleware())
	router.POST("/tenants/:tenantId/traces/:traceId/nodes", handler.AddNodes)

	headers := map[string]string{"X-Unified-UI-Autonomous-Agent-API-Key": "test-api-key"}
	w := testutils.PerformRequest(router, "POST", "/tenants/"+testutils.TestTenantID+"/traces/"+existingTrace.ID+"/nodes", addNodesReq, headers)

	testutils.AssertStatusCode(t, http.StatusOK, w)
}

func TestTracesHandler_AddNodes_APIKeyForbiddenForConversationTrace(t *testing.T) {
	mockDocDB := mocks.NewMockDocDBClient()
	mockPlatform := &mocks.MockPlatformClient{}

	// Conversation trace (not autonomous agent)
	existingTrace := testutils.NewTestTrace()
	existingTrace.ContextType = models.TraceContextConversation

	addNodesReq := dto.AddNodesRequest{
		Nodes: []dto.TraceNodeRequest{
			{ID: "node-1", Name: "test", Type: "llm", Status: "completed"},
		},
	}

	mockDocDB.GetTracesCollection().On("Get", mock.Anything, mock.Anything).Return(existingTrace, nil)

	handler := createTracesHandler(mockDocDB, mockPlatform)

	router := testutils.SetupTestRouter()
	router.Use(testFlexibleAuthMiddleware())
	router.POST("/tenants/:tenantId/traces/:traceId/nodes", handler.AddNodes)

	// API key for conversation trace should fail
	headers := map[string]string{"X-Unified-UI-Autonomous-Agent-API-Key": "test-api-key"}
	w := testutils.PerformRequest(router, "POST", "/tenants/"+testutils.TestTenantID+"/traces/"+existingTrace.ID+"/nodes", addNodesReq, headers)

	testutils.AssertStatusCode(t, http.StatusForbidden, w)
}

// =============================================================================
// AddLogs - Extended Tests for Error Paths
// =============================================================================

func TestTracesHandler_AddLogs_InvalidRequestBody(t *testing.T) {
	mockDocDB := mocks.NewMockDocDBClient()
	mockPlatform := &mocks.MockPlatformClient{}

	handler := createTracesHandler(mockDocDB, mockPlatform)

	router := testutils.SetupTestRouter()
	router.POST("/tenants/:tenantId/traces/:traceId/logs", handler.AddLogs)

	w := testutils.PerformRequest(router, "POST", "/tenants/"+testutils.TestTenantID+"/traces/trace-id/logs", "invalid json", nil)

	testutils.AssertStatusCode(t, http.StatusBadRequest, w)
}

func TestTracesHandler_AddLogs_DBErrorOnGet(t *testing.T) {
	mockDocDB := mocks.NewMockDocDBClient()
	mockPlatform := &mocks.MockPlatformClient{}

	addLogsReq := dto.AddLogsRequest{
		Logs: []interface{}{"log message 1"},
	}

	mockDocDB.GetTracesCollection().On("Get", mock.Anything, mock.Anything).Return(nil, errors.New("db connection error"))

	handler := createTracesHandler(mockDocDB, mockPlatform)

	router := testutils.SetupTestRouter()
	router.POST("/tenants/:tenantId/traces/:traceId/logs", handler.AddLogs)

	w := testutils.PerformRequest(router, "POST", "/tenants/"+testutils.TestTenantID+"/traces/trace-id/logs", addLogsReq, nil)

	testutils.AssertStatusCode(t, http.StatusInternalServerError, w)
}

func TestTracesHandler_AddLogs_TraceNotFound(t *testing.T) {
	mockDocDB := mocks.NewMockDocDBClient()
	mockPlatform := &mocks.MockPlatformClient{}

	addLogsReq := dto.AddLogsRequest{
		Logs: []interface{}{"log message 1"},
	}

	mockDocDB.GetTracesCollection().On("Get", mock.Anything, mock.Anything).Return(nil, nil)

	handler := createTracesHandler(mockDocDB, mockPlatform)

	router := testutils.SetupTestRouter()
	router.POST("/tenants/:tenantId/traces/:traceId/logs", handler.AddLogs)

	w := testutils.PerformRequest(router, "POST", "/tenants/"+testutils.TestTenantID+"/traces/non-existent/logs", addLogsReq, nil)

	testutils.AssertStatusCode(t, http.StatusNotFound, w)
}

func TestTracesHandler_AddLogs_TraceInDifferentTenant(t *testing.T) {
	mockDocDB := mocks.NewMockDocDBClient()
	mockPlatform := &mocks.MockPlatformClient{}

	existingTrace := testutils.NewTestTrace()
	existingTrace.TenantID = "different-tenant"

	addLogsReq := dto.AddLogsRequest{
		Logs: []interface{}{"log message 1"},
	}

	mockDocDB.GetTracesCollection().On("Get", mock.Anything, mock.Anything).Return(existingTrace, nil)

	handler := createTracesHandler(mockDocDB, mockPlatform)

	router := testutils.SetupTestRouter()
	router.POST("/tenants/:tenantId/traces/:traceId/logs", handler.AddLogs)

	w := testutils.PerformRequest(router, "POST", "/tenants/"+testutils.TestTenantID+"/traces/"+existingTrace.ID+"/logs", addLogsReq, nil)

	testutils.AssertStatusCode(t, http.StatusNotFound, w)
}

func TestTracesHandler_AddLogs_DBErrorOnAddLogs(t *testing.T) {
	mockDocDB := mocks.NewMockDocDBClient()
	mockPlatform := &mocks.MockPlatformClient{}

	existingTrace := testutils.NewTestTrace()

	addLogsReq := dto.AddLogsRequest{
		Logs: []interface{}{"log message 1"},
	}

	mockDocDB.GetTracesCollection().On("Get", mock.Anything, mock.Anything).Return(existingTrace, nil)
	mockDocDB.GetTracesCollection().On("AddLogs", mock.Anything, mock.Anything, mock.Anything).Return(errors.New("db write error"))

	handler := createTracesHandler(mockDocDB, mockPlatform)

	router := testutils.SetupTestRouter()
	router.POST("/tenants/:tenantId/traces/:traceId/logs", handler.AddLogs)

	w := testutils.PerformRequest(router, "POST", "/tenants/"+testutils.TestTenantID+"/traces/"+existingTrace.ID+"/logs", addLogsReq, nil)

	testutils.AssertStatusCode(t, http.StatusInternalServerError, w)
}

func TestTracesHandler_AddLogs_WithComplexLogObjects(t *testing.T) {
	mockDocDB := mocks.NewMockDocDBClient()
	mockPlatform := &mocks.MockPlatformClient{}

	existingTrace := testutils.NewTestTrace()

	addLogsReq := dto.AddLogsRequest{
		Logs: []interface{}{
			map[string]interface{}{"level": "info", "message": "test", "timestamp": time.Now().Unix()},
			map[string]interface{}{"level": "error", "message": "error occurred", "stack": "..."},
		},
	}

	mockDocDB.GetTracesCollection().On("Get", mock.Anything, mock.Anything).Return(existingTrace, nil)
	mockDocDB.GetTracesCollection().On("AddLogs", mock.Anything, mock.Anything, mock.Anything).Return(nil)

	handler := createTracesHandler(mockDocDB, mockPlatform)

	router := testutils.SetupTestRouter()
	router.POST("/tenants/:tenantId/traces/:traceId/logs", handler.AddLogs)

	w := testutils.PerformRequest(router, "POST", "/tenants/"+testutils.TestTenantID+"/traces/"+existingTrace.ID+"/logs", addLogsReq, nil)

	testutils.AssertStatusCode(t, http.StatusOK, w)
}

// =============================================================================
// GetConversationTraces - Extended Tests for Error Paths
// =============================================================================

func TestTracesHandler_GetConversationTraces_DBError(t *testing.T) {
	mockDocDB := mocks.NewMockDocDBClient()
	mockPlatform := &mocks.MockPlatformClient{}

	mockDocDB.GetTracesCollection().On("ListByConversation", mock.Anything, mock.Anything, mock.Anything).Return(nil, errors.New("db connection error"))

	handler := createTracesHandler(mockDocDB, mockPlatform)

	router := testutils.SetupTestRouter()
	router.GET("/tenants/:tenantId/conversations/:conversationId/traces", handler.GetConversationTraces)

	w := testutils.PerformRequest(router, "GET", "/tenants/"+testutils.TestTenantID+"/conversations/"+testutils.TestConversationID+"/traces", nil, nil)

	testutils.AssertStatusCode(t, http.StatusInternalServerError, w)
}

func TestTracesHandler_GetConversationTraces_EmptyResult(t *testing.T) {
	mockDocDB := mocks.NewMockDocDBClient()
	mockPlatform := &mocks.MockPlatformClient{}

	mockDocDB.GetTracesCollection().On("ListByConversation", mock.Anything, mock.Anything, mock.Anything).Return([]*models.Trace{}, nil)

	handler := createTracesHandler(mockDocDB, mockPlatform)

	router := testutils.SetupTestRouter()
	router.GET("/tenants/:tenantId/conversations/:conversationId/traces", handler.GetConversationTraces)

	w := testutils.PerformRequest(router, "GET", "/tenants/"+testutils.TestTenantID+"/conversations/"+testutils.TestConversationID+"/traces", nil, nil)

	testutils.AssertStatusCode(t, http.StatusOK, w)

	var response dto.ListTracesResponse
	testutils.ParseJSONResponse(t, w, &response)
	assert.Empty(t, response.Traces)
}

// =============================================================================
// RefreshConversationTrace - Extended Tests for Error Paths
// =============================================================================

func TestTracesHandler_RefreshConversationTrace_InvalidRequestBody(t *testing.T) {
	mockDocDB := mocks.NewMockDocDBClient()
	mockPlatform := &mocks.MockPlatformClient{}

	handler := createTracesHandler(mockDocDB, mockPlatform)

	router := testutils.SetupTestRouter()
	router.PUT("/tenants/:tenantId/conversations/:conversationId/traces", handler.RefreshConversationTrace)

	w := testutils.PerformRequest(router, "PUT", "/tenants/"+testutils.TestTenantID+"/conversations/"+testutils.TestConversationID+"/traces", "invalid json", nil)

	testutils.AssertStatusCode(t, http.StatusBadRequest, w)
}

func TestTracesHandler_RefreshConversationTrace_DBErrorOnGet(t *testing.T) {
	mockDocDB := mocks.NewMockDocDBClient()
	mockPlatform := &mocks.MockPlatformClient{}

	refreshReq := dto.RefreshTraceRequest{
		ReferenceID:   "updated-ref",
		ReferenceName: "Updated",
	}

	mockDocDB.GetTracesCollection().On("GetByConversation", mock.Anything, mock.Anything, mock.Anything).Return(nil, errors.New("db error"))

	handler := createTracesHandler(mockDocDB, mockPlatform)

	router := testutils.SetupTestRouter()
	router.PUT("/tenants/:tenantId/conversations/:conversationId/traces", handler.RefreshConversationTrace)

	w := testutils.PerformRequest(router, "PUT", "/tenants/"+testutils.TestTenantID+"/conversations/"+testutils.TestConversationID+"/traces", refreshReq, nil)

	testutils.AssertStatusCode(t, http.StatusInternalServerError, w)
}

func TestTracesHandler_RefreshConversationTrace_TraceNotFound(t *testing.T) {
	mockDocDB := mocks.NewMockDocDBClient()
	mockPlatform := &mocks.MockPlatformClient{}

	refreshReq := dto.RefreshTraceRequest{
		ReferenceID:   "updated-ref",
		ReferenceName: "Updated",
	}

	mockDocDB.GetTracesCollection().On("GetByConversation", mock.Anything, mock.Anything, mock.Anything).Return(nil, nil)

	handler := createTracesHandler(mockDocDB, mockPlatform)

	router := testutils.SetupTestRouter()
	router.PUT("/tenants/:tenantId/conversations/:conversationId/traces", handler.RefreshConversationTrace)

	w := testutils.PerformRequest(router, "PUT", "/tenants/"+testutils.TestTenantID+"/conversations/"+testutils.TestConversationID+"/traces", refreshReq, nil)

	testutils.AssertStatusCode(t, http.StatusNotFound, w)
}

func TestTracesHandler_RefreshConversationTrace_DBErrorOnUpdate(t *testing.T) {
	mockDocDB := mocks.NewMockDocDBClient()
	mockPlatform := &mocks.MockPlatformClient{}

	existingTrace := testutils.NewTestTrace()

	refreshReq := dto.RefreshTraceRequest{
		ReferenceID:   "updated-ref",
		ReferenceName: "Updated",
	}

	mockPlatform.On("GetMe", mock.Anything, mock.Anything).Return(&platform.UserInfo{ID: testutils.TestUserID}, nil)
	mockDocDB.GetTracesCollection().On("GetByConversation", mock.Anything, mock.Anything, mock.Anything).Return(existingTrace, nil)
	mockDocDB.GetTracesCollection().On("Update", mock.Anything, mock.Anything).Return(errors.New("db write error"))

	handler := createTracesHandler(mockDocDB, mockPlatform)

	router := testutils.SetupTestRouter()
	router.Use(testFlexibleAuthMiddleware())
	router.PUT("/tenants/:tenantId/conversations/:conversationId/traces", handler.RefreshConversationTrace)

	headers := map[string]string{"Authorization": "Bearer test-token"}
	w := testutils.PerformRequest(router, "PUT", "/tenants/"+testutils.TestTenantID+"/conversations/"+testutils.TestConversationID+"/traces", refreshReq, headers)

	testutils.AssertStatusCode(t, http.StatusInternalServerError, w)
}

// =============================================================================
// GetAutonomousAgentTraces - Extended Tests for Error Paths
// =============================================================================

func TestTracesHandler_GetAutonomousAgentTraces_DBErrorOnList(t *testing.T) {
	mockDocDB := mocks.NewMockDocDBClient()
	mockPlatform := &mocks.MockPlatformClient{}

	mockDocDB.GetTracesCollection().On("List", mock.Anything, mock.Anything).Return(nil, errors.New("db error"))

	handler := createTracesHandler(mockDocDB, mockPlatform)

	router := testutils.SetupTestRouter()
	router.GET("/tenants/:tenantId/autonomous-agents/:agentId/traces", handler.GetAutonomousAgentTraces)

	w := testutils.PerformRequest(router, "GET", "/tenants/"+testutils.TestTenantID+"/autonomous-agents/agent-123/traces", nil, nil)

	testutils.AssertStatusCode(t, http.StatusInternalServerError, w)
}

func TestTracesHandler_GetAutonomousAgentTraces_DBErrorOnCount(t *testing.T) {
	mockDocDB := mocks.NewMockDocDBClient()
	mockPlatform := &mocks.MockPlatformClient{}

	traces := []*models.Trace{}
	mockDocDB.GetTracesCollection().On("List", mock.Anything, mock.Anything).Return(traces, nil)
	mockDocDB.GetTracesCollection().On("Count", mock.Anything, mock.Anything).Return(int64(0), errors.New("db count error"))

	handler := createTracesHandler(mockDocDB, mockPlatform)

	router := testutils.SetupTestRouter()
	router.GET("/tenants/:tenantId/autonomous-agents/:agentId/traces", handler.GetAutonomousAgentTraces)

	w := testutils.PerformRequest(router, "GET", "/tenants/"+testutils.TestTenantID+"/autonomous-agents/agent-123/traces", nil, nil)

	testutils.AssertStatusCode(t, http.StatusInternalServerError, w)
}

func TestTracesHandler_GetAutonomousAgentTraces_InvalidCreatedBeforeDate(t *testing.T) {
	mockDocDB := mocks.NewMockDocDBClient()
	mockPlatform := &mocks.MockPlatformClient{}

	handler := createTracesHandler(mockDocDB, mockPlatform)

	router := testutils.SetupTestRouter()
	router.GET("/tenants/:tenantId/autonomous-agents/:agentId/traces", handler.GetAutonomousAgentTraces)

	path := "/tenants/" + testutils.TestTenantID + "/autonomous-agents/agent-123/traces?created_before=invalid-date"
	w := testutils.PerformRequest(router, "GET", path, nil, nil)

	testutils.AssertStatusCode(t, http.StatusBadRequest, w)
}

func TestTracesHandler_GetAutonomousAgentTraces_WithLimitExceeded(t *testing.T) {
	mockDocDB := mocks.NewMockDocDBClient()
	mockPlatform := &mocks.MockPlatformClient{}

	traces := []*models.Trace{}
	// Should be capped to 100
	mockDocDB.GetTracesCollection().On("List", mock.Anything, mock.Anything).Return(traces, nil)
	mockDocDB.GetTracesCollection().On("Count", mock.Anything, mock.Anything).Return(int64(0), nil)

	handler := createTracesHandler(mockDocDB, mockPlatform)

	router := testutils.SetupTestRouter()
	router.GET("/tenants/:tenantId/autonomous-agents/:agentId/traces", handler.GetAutonomousAgentTraces)

	path := "/tenants/" + testutils.TestTenantID + "/autonomous-agents/agent-123/traces?limit=500"
	w := testutils.PerformRequest(router, "GET", path, nil, nil)

	testutils.AssertStatusCode(t, http.StatusOK, w)
}

// =============================================================================
// ListAutonomousAgentTraces - Extended Tests for Error Paths
// =============================================================================

func TestTracesHandler_ListAutonomousAgentTraces_DBErrorOnList(t *testing.T) {
	mockDocDB := mocks.NewMockDocDBClient()
	mockPlatform := &mocks.MockPlatformClient{}

	mockDocDB.GetTracesCollection().On("List", mock.Anything, mock.Anything).Return(nil, errors.New("db error"))

	handler := createTracesHandler(mockDocDB, mockPlatform)

	router := testutils.SetupTestRouter()
	router.GET("/tenants/:tenantId/autonomous-agents/traces", handler.ListAutonomousAgentTraces)

	w := testutils.PerformRequest(router, "GET", "/tenants/"+testutils.TestTenantID+"/autonomous-agents/traces", nil, nil)

	testutils.AssertStatusCode(t, http.StatusInternalServerError, w)
}

func TestTracesHandler_ListAutonomousAgentTraces_DBErrorOnCount(t *testing.T) {
	mockDocDB := mocks.NewMockDocDBClient()
	mockPlatform := &mocks.MockPlatformClient{}

	traces := []*models.Trace{}
	mockDocDB.GetTracesCollection().On("List", mock.Anything, mock.Anything).Return(traces, nil)
	mockDocDB.GetTracesCollection().On("Count", mock.Anything, mock.Anything).Return(int64(0), errors.New("db count error"))

	handler := createTracesHandler(mockDocDB, mockPlatform)

	router := testutils.SetupTestRouter()
	router.GET("/tenants/:tenantId/autonomous-agents/traces", handler.ListAutonomousAgentTraces)

	w := testutils.PerformRequest(router, "GET", "/tenants/"+testutils.TestTenantID+"/autonomous-agents/traces", nil, nil)

	testutils.AssertStatusCode(t, http.StatusInternalServerError, w)
}

func TestTracesHandler_ListAutonomousAgentTraces_WithAgentIDFilter(t *testing.T) {
	mockDocDB := mocks.NewMockDocDBClient()
	mockPlatform := &mocks.MockPlatformClient{}

	traces := testutils.NewTestTraces(2)
	for _, trace := range traces {
		trace.ContextType = models.TraceContextAutonomousAgent
		trace.AutonomousAgentID = "specific-agent"
	}

	mockDocDB.GetTracesCollection().On("List", mock.Anything, mock.Anything).Return(traces, nil)
	mockDocDB.GetTracesCollection().On("Count", mock.Anything, mock.Anything).Return(int64(2), nil)

	handler := createTracesHandler(mockDocDB, mockPlatform)

	router := testutils.SetupTestRouter()
	router.GET("/tenants/:tenantId/autonomous-agents/traces", handler.ListAutonomousAgentTraces)

	path := "/tenants/" + testutils.TestTenantID + "/autonomous-agents/traces?autonomousAgentId=specific-agent"
	w := testutils.PerformRequest(router, "GET", path, nil, nil)

	testutils.AssertStatusCode(t, http.StatusOK, w)

	var response dto.ListTracesResponse
	testutils.ParseJSONResponse(t, w, &response)
	assert.Len(t, response.Traces, 2)
}

func TestTracesHandler_ListAutonomousAgentTraces_InvalidPagination(t *testing.T) {
	mockDocDB := mocks.NewMockDocDBClient()
	mockPlatform := &mocks.MockPlatformClient{}

	traces := []*models.Trace{}
	mockDocDB.GetTracesCollection().On("List", mock.Anything, mock.Anything).Return(traces, nil)
	mockDocDB.GetTracesCollection().On("Count", mock.Anything, mock.Anything).Return(int64(0), nil)

	handler := createTracesHandler(mockDocDB, mockPlatform)

	router := testutils.SetupTestRouter()
	router.GET("/tenants/:tenantId/autonomous-agents/traces", handler.ListAutonomousAgentTraces)

	// Invalid skip and limit values - should use defaults
	path := "/tenants/" + testutils.TestTenantID + "/autonomous-agents/traces?skip=-10&limit=-5"
	w := testutils.PerformRequest(router, "GET", path, nil, nil)

	testutils.AssertStatusCode(t, http.StatusOK, w)
}

// =============================================================================
// GetTrace - Extended Tests for Error Paths
// =============================================================================

func TestTracesHandler_GetTrace_DBError(t *testing.T) {
	mockDocDB := mocks.NewMockDocDBClient()
	mockPlatform := &mocks.MockPlatformClient{}

	mockDocDB.GetTracesCollection().On("Get", mock.Anything, mock.Anything).Return(nil, errors.New("db connection error"))

	handler := createTracesHandler(mockDocDB, mockPlatform)

	router := testutils.SetupTestRouter()
	router.GET("/tenants/:tenantId/traces/:traceId", handler.GetTrace)

	w := testutils.PerformRequest(router, "GET", "/tenants/"+testutils.TestTenantID+"/traces/trace-id", nil, nil)

	testutils.AssertStatusCode(t, http.StatusInternalServerError, w)
}

func TestTracesHandler_GetTrace_TraceInDifferentTenant(t *testing.T) {
	mockDocDB := mocks.NewMockDocDBClient()
	mockPlatform := &mocks.MockPlatformClient{}

	existingTrace := testutils.NewTestTrace()
	existingTrace.TenantID = "different-tenant"

	mockDocDB.GetTracesCollection().On("Get", mock.Anything, mock.Anything).Return(existingTrace, nil)

	handler := createTracesHandler(mockDocDB, mockPlatform)

	router := testutils.SetupTestRouter()
	router.GET("/tenants/:tenantId/traces/:traceId", handler.GetTrace)

	w := testutils.PerformRequest(router, "GET", "/tenants/"+testutils.TestTenantID+"/traces/"+existingTrace.ID, nil, nil)

	testutils.AssertStatusCode(t, http.StatusNotFound, w)
}

// =============================================================================
// DeleteTrace - Extended Tests for Error Paths
// =============================================================================

func TestTracesHandler_DeleteTrace_DBErrorOnGet(t *testing.T) {
	mockDocDB := mocks.NewMockDocDBClient()
	mockPlatform := &mocks.MockPlatformClient{}

	mockDocDB.GetTracesCollection().On("Get", mock.Anything, mock.Anything).Return(nil, errors.New("db connection error"))

	handler := createTracesHandler(mockDocDB, mockPlatform)

	router := testutils.SetupTestRouter()
	router.DELETE("/tenants/:tenantId/traces/:traceId", handler.DeleteTrace)

	w := testutils.PerformRequest(router, "DELETE", "/tenants/"+testutils.TestTenantID+"/traces/trace-id", nil, nil)

	testutils.AssertStatusCode(t, http.StatusInternalServerError, w)
}

func TestTracesHandler_DeleteTrace_TraceNotFound(t *testing.T) {
	mockDocDB := mocks.NewMockDocDBClient()
	mockPlatform := &mocks.MockPlatformClient{}

	mockDocDB.GetTracesCollection().On("Get", mock.Anything, mock.Anything).Return(nil, nil)

	handler := createTracesHandler(mockDocDB, mockPlatform)

	router := testutils.SetupTestRouter()
	router.DELETE("/tenants/:tenantId/traces/:traceId", handler.DeleteTrace)

	w := testutils.PerformRequest(router, "DELETE", "/tenants/"+testutils.TestTenantID+"/traces/non-existent", nil, nil)

	testutils.AssertStatusCode(t, http.StatusNotFound, w)
}

func TestTracesHandler_DeleteTrace_TraceInDifferentTenant(t *testing.T) {
	mockDocDB := mocks.NewMockDocDBClient()
	mockPlatform := &mocks.MockPlatformClient{}

	existingTrace := testutils.NewTestTrace()
	existingTrace.TenantID = "different-tenant"

	mockDocDB.GetTracesCollection().On("Get", mock.Anything, mock.Anything).Return(existingTrace, nil)

	handler := createTracesHandler(mockDocDB, mockPlatform)

	router := testutils.SetupTestRouter()
	router.DELETE("/tenants/:tenantId/traces/:traceId", handler.DeleteTrace)

	w := testutils.PerformRequest(router, "DELETE", "/tenants/"+testutils.TestTenantID+"/traces/"+existingTrace.ID, nil, nil)

	testutils.AssertStatusCode(t, http.StatusNotFound, w)
}

func TestTracesHandler_DeleteTrace_DBErrorOnDelete(t *testing.T) {
	mockDocDB := mocks.NewMockDocDBClient()
	mockPlatform := &mocks.MockPlatformClient{}

	existingTrace := testutils.NewTestTrace()

	mockDocDB.GetTracesCollection().On("Get", mock.Anything, mock.Anything).Return(existingTrace, nil)
	mockDocDB.GetTracesCollection().On("Delete", mock.Anything, mock.Anything).Return(errors.New("db delete error"))

	handler := createTracesHandler(mockDocDB, mockPlatform)

	router := testutils.SetupTestRouter()
	router.DELETE("/tenants/:tenantId/traces/:traceId", handler.DeleteTrace)

	w := testutils.PerformRequest(router, "DELETE", "/tenants/"+testutils.TestTenantID+"/traces/"+existingTrace.ID, nil, nil)

	testutils.AssertStatusCode(t, http.StatusInternalServerError, w)
}

// =============================================================================
// RefreshAutonomousAgentTrace - Extended Tests for Error Paths
// =============================================================================

func TestTracesHandler_RefreshAutonomousAgentTrace_InvalidRequestBody(t *testing.T) {
	mockDocDB := mocks.NewMockDocDBClient()
	mockPlatform := &mocks.MockPlatformClient{}

	handler := createTracesHandler(mockDocDB, mockPlatform)

	router := testutils.SetupTestRouter()
	router.PUT("/tenants/:tenantId/autonomous-agents/:agentId/traces", handler.RefreshAutonomousAgentTrace)

	w := testutils.PerformRequest(router, "PUT", "/tenants/"+testutils.TestTenantID+"/autonomous-agents/agent-123/traces", "invalid json", nil)

	testutils.AssertStatusCode(t, http.StatusBadRequest, w)
}

func TestTracesHandler_RefreshAutonomousAgentTrace_DBErrorOnGet(t *testing.T) {
	mockDocDB := mocks.NewMockDocDBClient()
	mockPlatform := &mocks.MockPlatformClient{}

	refreshReq := dto.RefreshTraceRequest{
		ReferenceID:   "updated-ref",
		ReferenceName: "Updated",
	}

	mockDocDB.GetTracesCollection().On("GetByAutonomousAgent", mock.Anything, mock.Anything, mock.Anything).Return(nil, errors.New("db error"))

	handler := createTracesHandler(mockDocDB, mockPlatform)

	router := testutils.SetupTestRouter()
	router.PUT("/tenants/:tenantId/autonomous-agents/:agentId/traces", handler.RefreshAutonomousAgentTrace)

	w := testutils.PerformRequest(router, "PUT", "/tenants/"+testutils.TestTenantID+"/autonomous-agents/agent-123/traces", refreshReq, nil)

	testutils.AssertStatusCode(t, http.StatusInternalServerError, w)
}

func TestTracesHandler_RefreshAutonomousAgentTrace_TraceNotFound(t *testing.T) {
	mockDocDB := mocks.NewMockDocDBClient()
	mockPlatform := &mocks.MockPlatformClient{}

	refreshReq := dto.RefreshTraceRequest{
		ReferenceID:   "updated-ref",
		ReferenceName: "Updated",
	}

	mockDocDB.GetTracesCollection().On("GetByAutonomousAgent", mock.Anything, mock.Anything, mock.Anything).Return(nil, nil)

	handler := createTracesHandler(mockDocDB, mockPlatform)

	router := testutils.SetupTestRouter()
	router.PUT("/tenants/:tenantId/autonomous-agents/:agentId/traces", handler.RefreshAutonomousAgentTrace)

	w := testutils.PerformRequest(router, "PUT", "/tenants/"+testutils.TestTenantID+"/autonomous-agents/agent-123/traces", refreshReq, nil)

	testutils.AssertStatusCode(t, http.StatusNotFound, w)
}

func TestTracesHandler_RefreshAutonomousAgentTrace_DBErrorOnUpdate(t *testing.T) {
	mockDocDB := mocks.NewMockDocDBClient()
	mockPlatform := &mocks.MockPlatformClient{}

	existingTrace := testutils.NewTestAutonomousAgentTrace()

	refreshReq := dto.RefreshTraceRequest{
		ReferenceID:   "updated-ref",
		ReferenceName: "Updated",
	}

	mockPlatform.On("GetMe", mock.Anything, mock.Anything).Return(&platform.UserInfo{ID: testutils.TestUserID}, nil)
	mockDocDB.GetTracesCollection().On("GetByAutonomousAgent", mock.Anything, mock.Anything, mock.Anything).Return(existingTrace, nil)
	mockDocDB.GetTracesCollection().On("Update", mock.Anything, mock.Anything).Return(errors.New("db write error"))

	handler := createTracesHandler(mockDocDB, mockPlatform)

	router := testutils.SetupTestRouter()
	router.Use(testFlexibleAuthMiddleware())
	router.PUT("/tenants/:tenantId/autonomous-agents/:agentId/traces", handler.RefreshAutonomousAgentTrace)

	headers := map[string]string{"Authorization": "Bearer test-token"}
	w := testutils.PerformRequest(router, "PUT", "/tenants/"+testutils.TestTenantID+"/autonomous-agents/agent-123/traces", refreshReq, headers)

	testutils.AssertStatusCode(t, http.StatusInternalServerError, w)
}

func TestTracesHandler_RefreshAutonomousAgentTrace_Success(t *testing.T) {
	mockDocDB := mocks.NewMockDocDBClient()
	mockPlatform := &mocks.MockPlatformClient{}

	existingTrace := testutils.NewTestAutonomousAgentTrace()

	refreshReq := dto.RefreshTraceRequest{
		ReferenceID:   "updated-ref",
		ReferenceName: "Updated Name",
		Nodes: []dto.TraceNodeRequest{
			{ID: "new-node", Name: "new-name", Type: "llm", Status: "completed"},
		},
	}

	mockPlatform.On("GetMe", mock.Anything, mock.Anything).Return(&platform.UserInfo{ID: testutils.TestUserID}, nil)
	mockDocDB.GetTracesCollection().On("GetByAutonomousAgent", mock.Anything, mock.Anything, mock.Anything).Return(existingTrace, nil)
	mockDocDB.GetTracesCollection().On("Update", mock.Anything, mock.Anything).Return(nil)

	handler := createTracesHandler(mockDocDB, mockPlatform)

	router := testutils.SetupTestRouter()
	router.Use(testFlexibleAuthMiddleware())
	router.PUT("/tenants/:tenantId/autonomous-agents/:agentId/traces", handler.RefreshAutonomousAgentTrace)

	headers := map[string]string{"Authorization": "Bearer test-token"}
	w := testutils.PerformRequest(router, "PUT", "/tenants/"+testutils.TestTenantID+"/autonomous-agents/agent-123/traces", refreshReq, headers)

	testutils.AssertStatusCode(t, http.StatusOK, w)
}

// =============================================================================
// Helper function tests for resolveUserIDForTrace
// =============================================================================

func TestTracesHandler_ResolveUserIDForTrace_APIKeyWithInvalidAPIKey(t *testing.T) {
	mockDocDB := mocks.NewMockDocDBClient()
	mockPlatform := &mocks.MockPlatformClient{}

	existingTrace := testutils.NewTestAutonomousAgentTrace()

	addNodesReq := dto.AddNodesRequest{
		Nodes: []dto.TraceNodeRequest{
			{ID: "node-1", Name: "test", Type: "llm", Status: "completed"},
		},
	}

	mockDocDB.GetTracesCollection().On("Get", mock.Anything, mock.Anything).Return(existingTrace, nil)
	mockPlatform.On("ValidateAutonomousAgentAPIKey", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(fmt.Errorf("unauthorized: invalid API key"))

	handler := createTracesHandler(mockDocDB, mockPlatform)

	router := testutils.SetupTestRouter()
	router.Use(testFlexibleAuthMiddleware())
	router.POST("/tenants/:tenantId/traces/:traceId/nodes", handler.AddNodes)

	headers := map[string]string{"X-Unified-UI-Autonomous-Agent-API-Key": "invalid-key"}
	w := testutils.PerformRequest(router, "POST", "/tenants/"+testutils.TestTenantID+"/traces/"+existingTrace.ID+"/nodes", addNodesReq, headers)

	testutils.AssertStatusCode(t, http.StatusUnauthorized, w)
}

// =============================================================================
// Test helper functions
// =============================================================================

// createTracesHandler creates a TracesHandler with mocks for testing.
func createTracesHandler(mockDocDB *mocks.MockDocDBClient, mockPlatform *mocks.MockPlatformClient) *handlers.TracesHandler {
	importService := traceimport.NewImportService(mockDocDB)
	return handlers.NewTracesHandler(mockDocDB, mockPlatform, importService)
}

// testFlexibleAuthMiddleware is a test middleware that extracts auth tokens.
func testFlexibleAuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader != "" && len(authHeader) > 7 {
			c.Set("auth_token", authHeader[7:])
		}
		apiKey := c.GetHeader("X-Unified-UI-Autonomous-Agent-API-Key")
		if apiKey != "" {
			c.Set("autonomous_agent_api_key", apiKey)
		}
		c.Next()
	}
}
