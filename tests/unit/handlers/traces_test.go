// Package handlers_test provides unit tests for trace handlers.
package handlers_test

import (
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

// createTestTracesHandler creates a TracesHandler with mocks for testing.
func createTestTracesHandler(mockDocDB *mocks.MockDocDBClient, mockPlatform *mocks.MockPlatformClient) *handlers.TracesHandler {
	importService := traceimport.NewImportService(mockDocDB)
	return handlers.NewTracesHandler(mockDocDB, mockPlatform, importService)
}

// autonomousAgentAPIKeyMiddleware is a test middleware that extracts the API key
// from the X-Unified-UI-Autonomous-Agent-API-Key header and sets it in the context.
func autonomousAgentAPIKeyMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		apiKey := c.GetHeader("X-Unified-UI-Autonomous-Agent-API-Key")
		if apiKey != "" {
			c.Set("autonomous_agent_api_key", apiKey)
		}
		c.Next()
	}
}

func TestTracesHandler_CreateTrace_Conversation_Success(t *testing.T) {
	// Setup
	mockDocDB := mocks.NewMockDocDBClient()
	mockPlatform := &mocks.MockPlatformClient{}

	now := time.Now().UTC()
	startAt := now.Add(-100 * time.Millisecond)

	createReq := dto.CreateTraceRequest{
		ApplicationID:  testutils.TestApplicationID,
		ConversationID: testutils.TestConversationID,
		ReferenceID:    "workflow-123",
		ReferenceName:  "Test Workflow",
		Nodes: []dto.TraceNodeRequest{
			{
				ID:       "node-1",
				Name:     "test-node",
				Type:     "llm",
				Status:   "completed",
				StartAt:  &startAt,
				EndAt:    &now,
				Duration: 0.1,
			},
		},
	}

	// Mock platform client responses - use mock.Anything for all params
	mockPlatform.On("GetMe", mock.Anything, mock.Anything).Return(&platform.UserInfo{ID: testutils.TestUserID}, nil)
	mockPlatform.On("ValidateConversation", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil)

	// Mock traces collection - no existing trace for this conversation
	mockDocDB.GetTracesCollection().On("GetByConversation", mock.Anything, testutils.TestTenantID, testutils.TestConversationID).Return(nil, nil)
	mockDocDB.GetTracesCollection().On("Create", mock.Anything, mock.Anything).Return(nil)

	handler := createTestTracesHandler(mockDocDB, mockPlatform)

	router := testutils.SetupTestRouter()
	router.POST("/tenants/:tenantId/traces", handler.CreateTrace)

	// Execute
	headers := map[string]string{"Authorization": "Bearer test-token"}
	w := testutils.PerformRequest(router, "POST", "/tenants/"+testutils.TestTenantID+"/traces", createReq, headers)

	// Assert
	testutils.AssertStatusCode(t, http.StatusCreated, w)

	var response dto.CreateTraceResponse
	testutils.ParseJSONResponse(t, w, &response)

	assert.NotEmpty(t, response.ID)

	mockDocDB.GetTracesCollection().AssertExpectations(t)
	mockPlatform.AssertExpectations(t)
}

func TestTracesHandler_CreateTrace_AutonomousAgent_Success(t *testing.T) {
	// Setup
	mockDocDB := mocks.NewMockDocDBClient()
	mockPlatform := &mocks.MockPlatformClient{}

	createReq := dto.CreateTraceRequest{
		AutonomousAgentID: "auto-agent-123",
		ReferenceID:       "scheduled-run-456",
		ReferenceName:     "Scheduled Agent Run",
	}

	// Mock platform client responses
	mockPlatform.On("GetMe", mock.Anything, mock.Anything).Return(&platform.UserInfo{ID: testutils.TestUserID}, nil)
	mockPlatform.On("ValidateAutonomousAgent", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil)

	// Mock traces collection
	mockDocDB.GetTracesCollection().On("Create", mock.Anything, mock.Anything).Return(nil)

	handler := createTestTracesHandler(mockDocDB, mockPlatform)

	router := testutils.SetupTestRouter()
	router.POST("/tenants/:tenantId/traces", handler.CreateTrace)

	// Execute
	headers := map[string]string{"Authorization": "Bearer test-token"}
	w := testutils.PerformRequest(router, "POST", "/tenants/"+testutils.TestTenantID+"/traces", createReq, headers)

	// Assert
	testutils.AssertStatusCode(t, http.StatusCreated, w)

	var response dto.CreateTraceResponse
	testutils.ParseJSONResponse(t, w, &response)

	assert.NotEmpty(t, response.ID)

	mockDocDB.GetTracesCollection().AssertExpectations(t)
	mockPlatform.AssertExpectations(t)
}

func TestTracesHandler_CreateTrace_MixedContext_Error(t *testing.T) {
	// Setup
	mockDocDB := mocks.NewMockDocDBClient()
	mockPlatform := &mocks.MockPlatformClient{}

	createReq := dto.CreateTraceRequest{
		ApplicationID:     testutils.TestApplicationID,
		ConversationID:    testutils.TestConversationID,
		AutonomousAgentID: "auto-agent-123", // Both contexts - invalid
	}

	handler := createTestTracesHandler(mockDocDB, mockPlatform)

	router := testutils.SetupTestRouter()
	router.POST("/tenants/:tenantId/traces", handler.CreateTrace)

	// Execute
	headers := map[string]string{"Authorization": "Bearer test-token"}
	w := testutils.PerformRequest(router, "POST", "/tenants/"+testutils.TestTenantID+"/traces", createReq, headers)

	// Assert
	testutils.AssertStatusCode(t, http.StatusBadRequest, w)
}

func TestTracesHandler_CreateTrace_MissingContext_Error(t *testing.T) {
	// Setup
	mockDocDB := mocks.NewMockDocDBClient()
	mockPlatform := &mocks.MockPlatformClient{}

	createReq := dto.CreateTraceRequest{
		ReferenceID:   "workflow-123",
		ReferenceName: "Test Workflow",
		// No context specified - invalid
	}

	handler := createTestTracesHandler(mockDocDB, mockPlatform)

	router := testutils.SetupTestRouter()
	router.POST("/tenants/:tenantId/traces", handler.CreateTrace)

	// Execute
	headers := map[string]string{"Authorization": "Bearer test-token"}
	w := testutils.PerformRequest(router, "POST", "/tenants/"+testutils.TestTenantID+"/traces", createReq, headers)

	// Assert
	testutils.AssertStatusCode(t, http.StatusBadRequest, w)
}

func TestTracesHandler_CreateTrace_ConversationAlreadyExists_Conflict(t *testing.T) {
	// Setup
	mockDocDB := mocks.NewMockDocDBClient()
	mockPlatform := &mocks.MockPlatformClient{}

	createReq := dto.CreateTraceRequest{
		ApplicationID:  testutils.TestApplicationID,
		ConversationID: testutils.TestConversationID,
		ReferenceID:    "workflow-123",
		ReferenceName:  "Test Workflow",
	}

	existingTrace := testutils.NewTestTrace()

	// Mock platform client responses
	mockPlatform.On("GetMe", mock.Anything, mock.Anything).Return(&platform.UserInfo{ID: testutils.TestUserID}, nil)
	mockPlatform.On("ValidateConversation", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil)

	// Mock traces collection - returns existing trace (conflict)
	mockDocDB.GetTracesCollection().On("GetByConversation", mock.Anything, testutils.TestTenantID, testutils.TestConversationID).Return(existingTrace, nil)

	handler := createTestTracesHandler(mockDocDB, mockPlatform)

	router := testutils.SetupTestRouter()
	router.POST("/tenants/:tenantId/traces", handler.CreateTrace)

	// Execute
	headers := map[string]string{"Authorization": "Bearer test-token"}
	w := testutils.PerformRequest(router, "POST", "/tenants/"+testutils.TestTenantID+"/traces", createReq, headers)

	// Assert
	testutils.AssertStatusCode(t, http.StatusConflict, w)

	var errorResp dto.ErrorResponse
	testutils.ParseJSONResponse(t, w, &errorResp)
	assert.Contains(t, errorResp.Message, "trace already exists")

	mockDocDB.GetTracesCollection().AssertExpectations(t)
	mockPlatform.AssertExpectations(t)
}

func TestTracesHandler_AddNodes_Success(t *testing.T) {
	// Setup
	mockDocDB := mocks.NewMockDocDBClient()
	mockPlatform := &mocks.MockPlatformClient{}

	existingTrace := testutils.NewTestTrace()

	addNodesReq := dto.AddNodesRequest{
		Nodes: []dto.TraceNodeRequest{
			{
				ID:     "node-2",
				Name:   "new-node",
				Type:   "tool",
				Status: "completed",
			},
		},
	}

	// Mock platform client for getUserID
	mockPlatform.On("GetMe", mock.Anything, mock.Anything).Return(&platform.UserInfo{ID: testutils.TestUserID}, nil)

	// Mock get existing trace - use mock.Anything for all params
	mockDocDB.GetTracesCollection().On("Get", mock.Anything, mock.Anything).Return(existingTrace, nil)
	mockDocDB.GetTracesCollection().On("AddNodes", mock.Anything, mock.Anything, mock.Anything).Return(nil)

	handler := createTestTracesHandler(mockDocDB, mockPlatform)

	router := testutils.SetupTestRouter()
	router.POST("/tenants/:tenantId/traces/:traceId/nodes", handler.AddNodes)

	// Execute
	headers := map[string]string{"Authorization": "Bearer test-token"}
	w := testutils.PerformRequest(router, "POST", "/tenants/"+testutils.TestTenantID+"/traces/"+existingTrace.ID+"/nodes", addNodesReq, headers)

	// Assert
	testutils.AssertStatusCode(t, http.StatusOK, w)

	mockDocDB.GetTracesCollection().AssertExpectations(t)
}

func TestTracesHandler_AddNodes_TraceNotFound(t *testing.T) {
	// Setup
	mockDocDB := mocks.NewMockDocDBClient()
	mockPlatform := &mocks.MockPlatformClient{}

	addNodesReq := dto.AddNodesRequest{
		Nodes: []dto.TraceNodeRequest{
			{
				ID:     "node-2",
				Name:   "new-node",
				Type:   "tool",
				Status: "completed",
			},
		},
	}

	// Mock trace not found
	mockDocDB.GetTracesCollection().On("Get", mock.Anything, mock.Anything).Return(nil, nil)

	handler := createTestTracesHandler(mockDocDB, mockPlatform)

	router := testutils.SetupTestRouter()
	router.POST("/tenants/:tenantId/traces/:traceId/nodes", handler.AddNodes)

	// Execute
	headers := map[string]string{"Authorization": "Bearer test-token"}
	w := testutils.PerformRequest(router, "POST", "/tenants/"+testutils.TestTenantID+"/traces/non-existent/nodes", addNodesReq, headers)

	// Assert
	testutils.AssertStatusCode(t, http.StatusNotFound, w)
}

func TestTracesHandler_AddLogs_Success(t *testing.T) {
	// Setup
	mockDocDB := mocks.NewMockDocDBClient()
	mockPlatform := &mocks.MockPlatformClient{}

	existingTrace := testutils.NewTestTrace()

	addLogsReq := dto.AddLogsRequest{
		Logs: []interface{}{
			map[string]interface{}{"level": "info", "message": "test log"},
		},
	}

	// Mock get existing trace
	mockDocDB.GetTracesCollection().On("Get", mock.Anything, mock.Anything).Return(existingTrace, nil)
	mockDocDB.GetTracesCollection().On("AddLogs", mock.Anything, mock.Anything, mock.Anything).Return(nil)

	handler := createTestTracesHandler(mockDocDB, mockPlatform)

	router := testutils.SetupTestRouter()
	router.POST("/tenants/:tenantId/traces/:traceId/logs", handler.AddLogs)

	// Execute
	headers := map[string]string{"Authorization": "Bearer test-token"}
	w := testutils.PerformRequest(router, "POST", "/tenants/"+testutils.TestTenantID+"/traces/"+existingTrace.ID+"/logs", addLogsReq, headers)

	// Assert
	testutils.AssertStatusCode(t, http.StatusOK, w)

	mockDocDB.GetTracesCollection().AssertExpectations(t)
}

func TestTracesHandler_GetTrace_Success(t *testing.T) {
	// Setup
	mockDocDB := mocks.NewMockDocDBClient()
	mockPlatform := &mocks.MockPlatformClient{}

	existingTrace := testutils.NewTestTrace()

	mockDocDB.GetTracesCollection().On("Get", mock.Anything, mock.Anything).Return(existingTrace, nil)

	handler := createTestTracesHandler(mockDocDB, mockPlatform)

	router := testutils.SetupTestRouter()
	router.GET("/tenants/:tenantId/traces/:traceId", handler.GetTrace)

	// Execute
	w := testutils.PerformRequest(router, "GET", "/tenants/"+testutils.TestTenantID+"/traces/"+existingTrace.ID, nil, nil)

	// Assert
	testutils.AssertStatusCode(t, http.StatusOK, w)

	var response dto.TraceResponse
	testutils.ParseJSONResponse(t, w, &response)

	assert.Equal(t, existingTrace.ID, response.ID)
	assert.Equal(t, existingTrace.TenantID, response.TenantID)
	assert.Equal(t, string(models.TraceContextConversation), response.ContextType)

	mockDocDB.GetTracesCollection().AssertExpectations(t)
}

func TestTracesHandler_GetTrace_NotFound(t *testing.T) {
	// Setup
	mockDocDB := mocks.NewMockDocDBClient()
	mockPlatform := &mocks.MockPlatformClient{}

	mockDocDB.GetTracesCollection().On("Get", mock.Anything, mock.Anything).Return(nil, nil)

	handler := createTestTracesHandler(mockDocDB, mockPlatform)

	router := testutils.SetupTestRouter()
	router.GET("/tenants/:tenantId/traces/:traceId", handler.GetTrace)

	// Execute
	w := testutils.PerformRequest(router, "GET", "/tenants/"+testutils.TestTenantID+"/traces/non-existent", nil, nil)

	// Assert
	testutils.AssertStatusCode(t, http.StatusNotFound, w)
}

func TestTracesHandler_DeleteTrace_Success(t *testing.T) {
	// Setup
	mockDocDB := mocks.NewMockDocDBClient()
	mockPlatform := &mocks.MockPlatformClient{}

	existingTrace := testutils.NewTestTrace()

	mockDocDB.GetTracesCollection().On("Get", mock.Anything, mock.Anything).Return(existingTrace, nil)
	mockDocDB.GetTracesCollection().On("Delete", mock.Anything, mock.Anything).Return(nil)

	handler := createTestTracesHandler(mockDocDB, mockPlatform)

	router := testutils.SetupTestRouter()
	router.DELETE("/tenants/:tenantId/traces/:traceId", handler.DeleteTrace)

	// Execute
	w := testutils.PerformRequest(router, "DELETE", "/tenants/"+testutils.TestTenantID+"/traces/"+existingTrace.ID, nil, nil)

	// Assert
	testutils.AssertStatusCode(t, http.StatusNoContent, w)

	mockDocDB.GetTracesCollection().AssertExpectations(t)
}

func TestTracesHandler_GetConversationTraces_Success(t *testing.T) {
	// Setup
	mockDocDB := mocks.NewMockDocDBClient()
	mockPlatform := &mocks.MockPlatformClient{}

	existingTrace := testutils.NewTestTrace()
	traces := []*models.Trace{existingTrace}

	// GetConversationTraces uses ListByConversation which returns a list
	mockDocDB.GetTracesCollection().On("ListByConversation", mock.Anything, mock.Anything, mock.Anything).Return(traces, nil)

	handler := createTestTracesHandler(mockDocDB, mockPlatform)

	router := testutils.SetupTestRouter()
	router.GET("/tenants/:tenantId/conversations/:conversationId/traces", handler.GetConversationTraces)

	// Execute
	headers := map[string]string{"Authorization": "Bearer test-token"}
	path := "/tenants/" + testutils.TestTenantID + "/conversations/" + testutils.TestConversationID + "/traces"
	w := testutils.PerformRequest(router, "GET", path, nil, headers)

	// Assert
	testutils.AssertStatusCode(t, http.StatusOK, w)

	var response dto.ListTracesResponse
	testutils.ParseJSONResponse(t, w, &response)

	assert.Len(t, response.Traces, 1)
	assert.Equal(t, existingTrace.ID, response.Traces[0].ID)

	mockDocDB.GetTracesCollection().AssertExpectations(t)
}

func TestTracesHandler_ListAutonomousAgentTraces_Success(t *testing.T) {
	// Setup
	mockDocDB := mocks.NewMockDocDBClient()
	mockPlatform := &mocks.MockPlatformClient{}

	traces := testutils.NewTestTraces(3)
	for _, trace := range traces {
		trace.ContextType = models.TraceContextAutonomousAgent
		trace.AutonomousAgentID = "auto-agent-123"
		trace.ApplicationID = ""
		trace.ConversationID = ""
	}

	// ListAutonomousAgentTraces does NOT call ValidateAutonomousAgent - it just lists traces
	mockDocDB.GetTracesCollection().On("List", mock.Anything, mock.Anything).Return(traces, nil)

	handler := createTestTracesHandler(mockDocDB, mockPlatform)

	router := testutils.SetupTestRouter()
	router.GET("/tenants/:tenantId/autonomous-agents/traces", handler.ListAutonomousAgentTraces)

	// Execute
	headers := map[string]string{"Authorization": "Bearer test-token"}
	path := "/tenants/" + testutils.TestTenantID + "/autonomous-agents/traces?skip=0&limit=10"
	w := testutils.PerformRequest(router, "GET", path, nil, headers)

	// Assert
	testutils.AssertStatusCode(t, http.StatusOK, w)

	var response dto.ListTracesResponse
	testutils.ParseJSONResponse(t, w, &response)

	assert.Len(t, response.Traces, 3)

	mockDocDB.GetTracesCollection().AssertExpectations(t)
}

func TestTracesHandler_RefreshConversationTrace_Success(t *testing.T) {
	// Setup
	mockDocDB := mocks.NewMockDocDBClient()
	mockPlatform := &mocks.MockPlatformClient{}

	existingTrace := testutils.NewTestTrace()

	refreshReq := dto.RefreshTraceRequest{
		ReferenceID:   "updated-workflow-123",
		ReferenceName: "Updated Workflow",
		Nodes: []dto.TraceNodeRequest{
			{
				ID:     "node-new",
				Name:   "refreshed-node",
				Type:   "agent",
				Status: "completed",
			},
		},
	}

	mockPlatform.On("GetMe", mock.Anything, mock.Anything).Return(&platform.UserInfo{ID: testutils.TestUserID}, nil)
	mockDocDB.GetTracesCollection().On("GetByConversation", mock.Anything, mock.Anything, mock.Anything).Return(existingTrace, nil)
	mockDocDB.GetTracesCollection().On("Update", mock.Anything, mock.Anything).Return(nil)

	handler := createTestTracesHandler(mockDocDB, mockPlatform)

	router := testutils.SetupTestRouter()
	router.PUT("/tenants/:tenantId/conversations/:conversationId/traces", handler.RefreshConversationTrace)

	// Execute
	headers := map[string]string{"Authorization": "Bearer test-token"}
	path := "/tenants/" + testutils.TestTenantID + "/conversations/" + testutils.TestConversationID + "/traces"
	w := testutils.PerformRequest(router, "PUT", path, refreshReq, headers)

	// Assert
	testutils.AssertStatusCode(t, http.StatusOK, w)

	mockDocDB.GetTracesCollection().AssertExpectations(t)
	mockPlatform.AssertExpectations(t)
}

// =============================================================================
// Tests for ImportAutonomousAgentTrace handler (PUT /autonomous-agents/{agentId}/traces/import)
// =============================================================================

func TestTracesHandler_ImportAutonomousAgentTrace_Success(t *testing.T) {
	// Setup
	mockDocDB := mocks.NewMockDocDBClient()
	mockPlatform := &mocks.MockPlatformClient{}

	importReq := dto.AutonomousAgentImportTraceRequest{
		Type:        "N8N",
		ExecutionID: "n8n-execution-123",
		SessionID:   "session-456",
	}

	agentConfig := &platform.AutonomousAgentConfigResponse{
		Type:              platform.AgentTypeN8N,
		TenantID:          testutils.TestTenantID,
		AutonomousAgentID: "auto-agent-123",
		Settings: platform.AutonomousAgentConfigSettings{
			APIVersion:          "v1",
			N8NHost:             "https://n8n.example.com",
			N8NWorkflowEndpoint: "https://n8n.example.com/api/v1",
			WorkflowID:          "workflow-123",
			APICredentials: &platform.Credentials{
				ID:     "cred-123",
				Name:   "N8N API Key",
				Type:   platform.CredentialTypeN8NAPIKey,
				Secret: "test-n8n-api-key",
			},
		},
	}

	// Mock platform client responses
	mockPlatform.On("GetAutonomousAgentConfig", mock.Anything, testutils.TestTenantID, "auto-agent-123", "test-api-key").Return(agentConfig, nil)

	// Mock traces collection - GetByReferenceID returns nil (new trace)
	mockDocDB.GetTracesCollection().On("GetByReferenceID", mock.Anything, testutils.TestTenantID, "n8n-execution-123").Return(nil, nil)
	mockDocDB.GetTracesCollection().On("Create", mock.Anything, mock.Anything).Return(nil)
	mockDocDB.GetTracesCollection().On("Get", mock.Anything, mock.Anything).Return(testutils.NewTestTrace(), nil)
	mockDocDB.GetTracesCollection().On("Update", mock.Anything, mock.Anything).Return(nil)

	handler := createTestTracesHandler(mockDocDB, mockPlatform)

	// Register N8N importer (handler needs this)
	handler.GetImportService().RegisterImporter(mocks.NewMockTraceImporter())

	router := testutils.SetupTestRouter()
	router.PUT("/tenants/:tenantId/autonomous-agents/:agentId/traces/import", autonomousAgentAPIKeyMiddleware(), handler.ImportAutonomousAgentTrace)

	// Execute - use API key header instead of Bearer token
	headers := map[string]string{"X-Unified-UI-Autonomous-Agent-API-Key": "test-api-key"}
	w := testutils.PerformRequest(router, "PUT", "/tenants/"+testutils.TestTenantID+"/autonomous-agents/auto-agent-123/traces/import", importReq, headers)

	// Assert
	testutils.AssertStatusCode(t, http.StatusCreated, w)

	var response dto.ImportTraceResponse
	testutils.ParseJSONResponse(t, w, &response)

	assert.NotEmpty(t, response.ID)

	mockPlatform.AssertExpectations(t)
}

func TestTracesHandler_ImportAutonomousAgentTrace_InvalidAPIKey(t *testing.T) {
	// Setup
	mockDocDB := mocks.NewMockDocDBClient()
	mockPlatform := &mocks.MockPlatformClient{}

	importReq := dto.AutonomousAgentImportTraceRequest{
		Type:        "N8N",
		ExecutionID: "n8n-execution-123",
	}

	// Mock platform client returns unauthorized error
	mockPlatform.On("GetAutonomousAgentConfig", mock.Anything, testutils.TestTenantID, "auto-agent-123", "invalid-api-key").
		Return(nil, fmt.Errorf("unauthorized: invalid API key"))

	handler := createTestTracesHandler(mockDocDB, mockPlatform)

	router := testutils.SetupTestRouter()
	router.PUT("/tenants/:tenantId/autonomous-agents/:agentId/traces/import", autonomousAgentAPIKeyMiddleware(), handler.ImportAutonomousAgentTrace)

	// Execute
	headers := map[string]string{"X-Unified-UI-Autonomous-Agent-API-Key": "invalid-api-key"}
	w := testutils.PerformRequest(router, "PUT", "/tenants/"+testutils.TestTenantID+"/autonomous-agents/auto-agent-123/traces/import", importReq, headers)

	// Assert
	testutils.AssertStatusCode(t, http.StatusUnauthorized, w)

	mockPlatform.AssertExpectations(t)
}

func TestTracesHandler_ImportAutonomousAgentTrace_AgentNotFound(t *testing.T) {
	// Setup
	mockDocDB := mocks.NewMockDocDBClient()
	mockPlatform := &mocks.MockPlatformClient{}

	importReq := dto.AutonomousAgentImportTraceRequest{
		Type:        "N8N",
		ExecutionID: "n8n-execution-123",
	}

	// Mock platform client returns not found error
	mockPlatform.On("GetAutonomousAgentConfig", mock.Anything, testutils.TestTenantID, "non-existent-agent", "test-api-key").
		Return(nil, fmt.Errorf("not_found: autonomous agent not found"))

	handler := createTestTracesHandler(mockDocDB, mockPlatform)

	router := testutils.SetupTestRouter()
	router.PUT("/tenants/:tenantId/autonomous-agents/:agentId/traces/import", autonomousAgentAPIKeyMiddleware(), handler.ImportAutonomousAgentTrace)

	// Execute
	headers := map[string]string{"X-Unified-UI-Autonomous-Agent-API-Key": "test-api-key"}
	w := testutils.PerformRequest(router, "PUT", "/tenants/"+testutils.TestTenantID+"/autonomous-agents/non-existent-agent/traces/import", importReq, headers)

	// Assert
	testutils.AssertStatusCode(t, http.StatusNotFound, w)

	mockPlatform.AssertExpectations(t)
}

func TestTracesHandler_ImportAutonomousAgentTrace_UnsupportedAgentType(t *testing.T) {
	// Setup
	mockDocDB := mocks.NewMockDocDBClient()
	mockPlatform := &mocks.MockPlatformClient{}

	importReq := dto.AutonomousAgentImportTraceRequest{
		Type:        "UNKNOWN_TYPE",
		ExecutionID: "execution-123",
	}

	agentConfig := &platform.AutonomousAgentConfigResponse{
		Type:              platform.AgentTypeN8N,
		TenantID:          testutils.TestTenantID,
		AutonomousAgentID: "auto-agent-123",
	}

	// Mock platform client responses
	mockPlatform.On("GetAutonomousAgentConfig", mock.Anything, testutils.TestTenantID, "auto-agent-123", "test-api-key").Return(agentConfig, nil)

	handler := createTestTracesHandler(mockDocDB, mockPlatform)
	// Note: No importer registered for the requested type

	router := testutils.SetupTestRouter()
	router.PUT("/tenants/:tenantId/autonomous-agents/:agentId/traces/import", autonomousAgentAPIKeyMiddleware(), handler.ImportAutonomousAgentTrace)

	// Execute
	headers := map[string]string{"X-Unified-UI-Autonomous-Agent-API-Key": "test-api-key"}
	w := testutils.PerformRequest(router, "PUT", "/tenants/"+testutils.TestTenantID+"/autonomous-agents/auto-agent-123/traces/import", importReq, headers)

	// Assert - should fail due to unsupported/invalid type
	testutils.AssertStatusCode(t, http.StatusBadRequest, w)

	mockPlatform.AssertExpectations(t)
}

func TestTracesHandler_ImportAutonomousAgentTrace_MissingExecutionID(t *testing.T) {
	// Setup
	mockDocDB := mocks.NewMockDocDBClient()
	mockPlatform := &mocks.MockPlatformClient{}

	importReq := dto.AutonomousAgentImportTraceRequest{
		Type: "N8N",
		// ExecutionID is missing (required field)
	}

	handler := createTestTracesHandler(mockDocDB, mockPlatform)

	router := testutils.SetupTestRouter()
	router.PUT("/tenants/:tenantId/autonomous-agents/:agentId/traces/import", autonomousAgentAPIKeyMiddleware(), handler.ImportAutonomousAgentTrace)

	// Execute
	headers := map[string]string{"X-Unified-UI-Autonomous-Agent-API-Key": "test-api-key"}
	w := testutils.PerformRequest(router, "PUT", "/tenants/"+testutils.TestTenantID+"/autonomous-agents/auto-agent-123/traces/import", importReq, headers)

	// Assert - should fail validation
	testutils.AssertStatusCode(t, http.StatusBadRequest, w)
}

func TestTracesHandler_ImportAutonomousAgentTrace_UpdateExisting(t *testing.T) {
	// Setup - test upsert scenario where trace already exists (should return 200)
	mockDocDB := mocks.NewMockDocDBClient()
	mockPlatform := &mocks.MockPlatformClient{}

	importReq := dto.AutonomousAgentImportTraceRequest{
		Type:        "N8N",
		ExecutionID: "n8n-execution-123",
		SessionID:   "session-456",
	}

	agentConfig := &platform.AutonomousAgentConfigResponse{
		Type:              platform.AgentTypeN8N,
		TenantID:          testutils.TestTenantID,
		AutonomousAgentID: "auto-agent-123",
		Settings: platform.AutonomousAgentConfigSettings{
			APIVersion:          "v1",
			N8NHost:             "https://n8n.example.com",
			N8NWorkflowEndpoint: "https://n8n.example.com/api/v1",
			WorkflowID:          "workflow-123",
			APICredentials: &platform.Credentials{
				ID:     "cred-123",
				Name:   "N8N API Key",
				Type:   platform.CredentialTypeN8NAPIKey,
				Secret: "test-n8n-api-key",
			},
		},
	}

	// Existing trace to be replaced
	existingTrace := testutils.NewTestTrace()
	existingTrace.ID = "existing-trace-id"
	existingTrace.ReferenceID = "n8n-execution-123"
	existingTrace.AutonomousAgentID = "auto-agent-123"
	existingTrace.ContextType = models.TraceContextAutonomousAgent

	// Mock platform client responses
	mockPlatform.On("GetAutonomousAgentConfig", mock.Anything, testutils.TestTenantID, "auto-agent-123", "test-api-key").Return(agentConfig, nil)

	// Mock traces collection - GetByReferenceID returns existing trace
	mockDocDB.GetTracesCollection().On("GetByReferenceID", mock.Anything, testutils.TestTenantID, "n8n-execution-123").Return(existingTrace, nil)
	// No Delete needed - the importer will update the existing trace
	// Update is called by the mock importer
	mockDocDB.GetTracesCollection().On("Update", mock.Anything, mock.Anything).Return(nil)

	handler := createTestTracesHandler(mockDocDB, mockPlatform)

	// Register N8N importer (handler needs this)
	handler.GetImportService().RegisterImporter(mocks.NewMockTraceImporter())

	router := testutils.SetupTestRouter()
	router.PUT("/tenants/:tenantId/autonomous-agents/:agentId/traces/import", autonomousAgentAPIKeyMiddleware(), handler.ImportAutonomousAgentTrace)

	// Execute - use API key header instead of Bearer token
	headers := map[string]string{"X-Unified-UI-Autonomous-Agent-API-Key": "test-api-key"}
	w := testutils.PerformRequest(router, "PUT", "/tenants/"+testutils.TestTenantID+"/autonomous-agents/auto-agent-123/traces/import", importReq, headers)

	// Assert - should return 200 for update (not 201 for create)
	testutils.AssertStatusCode(t, http.StatusOK, w)

	var response dto.ImportTraceResponse
	testutils.ParseJSONResponse(t, w, &response)

	// IMPORTANT: The trace ID should be preserved during upsert
	assert.Equal(t, "existing-trace-id", response.ID, "trace ID should be preserved during upsert")

	mockPlatform.AssertExpectations(t)
}

// =============================================================================
// Tests for RefreshAutonomousAgentImportTrace handler (PUT /autonomous-agents/{agentId}/traces/{traceId}/import/refresh)
// =============================================================================

func TestTracesHandler_RefreshAutonomousAgentImportTrace_Success(t *testing.T) {
	// Setup
	mockDocDB := mocks.NewMockDocDBClient()
	mockPlatform := &mocks.MockPlatformClient{}

	existingTrace := testutils.NewTestTrace()
	existingTrace.AutonomousAgentID = "auto-agent-123"
	existingTrace.ContextType = models.TraceContextAutonomousAgent
	existingTrace.ReferenceID = "n8n-execution-original"
	existingTrace.ReferenceMetadata = map[string]interface{}{
		"execution_id": "n8n-execution-original",
	}

	agentConfig := &platform.AutonomousAgentConfigResponse{
		Type:              platform.AgentTypeN8N,
		TenantID:          testutils.TestTenantID,
		AutonomousAgentID: "auto-agent-123",
		Settings: platform.AutonomousAgentConfigSettings{
			APIVersion:          "v1",
			N8NHost:             "https://n8n.example.com",
			N8NWorkflowEndpoint: "https://n8n.example.com/api/v1",
			APICredentials: &platform.Credentials{
				ID:     "cred-123",
				Name:   "N8N API Key",
				Type:   platform.CredentialTypeN8NAPIKey,
				Secret: "test-n8n-api-key",
			},
		},
	}

	// Mock platform client responses
	mockPlatform.On("GetAutonomousAgentConfig", mock.Anything, testutils.TestTenantID, "auto-agent-123", "test-api-key").Return(agentConfig, nil)

	// Mock traces collection:
	// 1. First Get call to validate existing trace belongs to this agent
	mockDocDB.GetTracesCollection().On("Get", mock.Anything, existingTrace.ID).Return(existingTrace, nil).Once()
	// 2. Second Get call after import returns the new trace ID
	newTrace := testutils.NewTestTrace()
	newTrace.ID = "mock-trace-id"
	mockDocDB.GetTracesCollection().On("Get", mock.Anything, "mock-trace-id").Return(newTrace, nil).Once()
	// 3. Update called after linking to autonomous agent
	mockDocDB.GetTracesCollection().On("Update", mock.Anything, mock.Anything).Return(nil)

	handler := createTestTracesHandler(mockDocDB, mockPlatform)

	// Register N8N importer
	handler.GetImportService().RegisterImporter(mocks.NewMockTraceImporter())

	router := testutils.SetupTestRouter()
	router.PUT("/tenants/:tenantId/autonomous-agents/:agentId/traces/:traceId/import/refresh", autonomousAgentAPIKeyMiddleware(), handler.RefreshAutonomousAgentImportTrace)

	// Execute
	headers := map[string]string{"X-Unified-UI-Autonomous-Agent-API-Key": "test-api-key"}
	path := "/tenants/" + testutils.TestTenantID + "/autonomous-agents/auto-agent-123/traces/" + existingTrace.ID + "/import/refresh"
	w := testutils.PerformRequest(router, "PUT", path, nil, headers)

	// Assert
	testutils.AssertStatusCode(t, http.StatusOK, w)

	mockPlatform.AssertExpectations(t)
}

func TestTracesHandler_RefreshAutonomousAgentImportTrace_TraceNotFound(t *testing.T) {
	// Setup
	mockDocDB := mocks.NewMockDocDBClient()
	mockPlatform := &mocks.MockPlatformClient{}

	agentConfig := &platform.AutonomousAgentConfigResponse{
		Type:              platform.AgentTypeN8N,
		TenantID:          testutils.TestTenantID,
		AutonomousAgentID: "auto-agent-123",
	}

	// Mock platform client responses
	mockPlatform.On("GetAutonomousAgentConfig", mock.Anything, testutils.TestTenantID, "auto-agent-123", "test-api-key").Return(agentConfig, nil)

	// Mock traces collection - trace not found
	mockDocDB.GetTracesCollection().On("Get", mock.Anything, "non-existent-trace").Return(nil, nil)

	handler := createTestTracesHandler(mockDocDB, mockPlatform)

	router := testutils.SetupTestRouter()
	router.PUT("/tenants/:tenantId/autonomous-agents/:agentId/traces/:traceId/import/refresh", autonomousAgentAPIKeyMiddleware(), handler.RefreshAutonomousAgentImportTrace)

	// Execute
	headers := map[string]string{"X-Unified-UI-Autonomous-Agent-API-Key": "test-api-key"}
	path := "/tenants/" + testutils.TestTenantID + "/autonomous-agents/auto-agent-123/traces/non-existent-trace/import/refresh"
	w := testutils.PerformRequest(router, "PUT", path, nil, headers)

	// Assert
	testutils.AssertStatusCode(t, http.StatusNotFound, w)

	mockPlatform.AssertExpectations(t)
}

func TestTracesHandler_RefreshAutonomousAgentImportTrace_WrongAgent(t *testing.T) {
	// Setup
	mockDocDB := mocks.NewMockDocDBClient()
	mockPlatform := &mocks.MockPlatformClient{}

	existingTrace := testutils.NewTestTrace()
	existingTrace.AutonomousAgentID = "different-agent-id" // Trace belongs to different agent
	existingTrace.ContextType = models.TraceContextAutonomousAgent

	agentConfig := &platform.AutonomousAgentConfigResponse{
		Type:              platform.AgentTypeN8N,
		TenantID:          testutils.TestTenantID,
		AutonomousAgentID: "auto-agent-123",
	}

	// Mock platform client responses
	mockPlatform.On("GetAutonomousAgentConfig", mock.Anything, testutils.TestTenantID, "auto-agent-123", "test-api-key").Return(agentConfig, nil)

	// Mock traces collection
	mockDocDB.GetTracesCollection().On("Get", mock.Anything, existingTrace.ID).Return(existingTrace, nil)

	handler := createTestTracesHandler(mockDocDB, mockPlatform)

	router := testutils.SetupTestRouter()
	router.PUT("/tenants/:tenantId/autonomous-agents/:agentId/traces/:traceId/import/refresh", autonomousAgentAPIKeyMiddleware(), handler.RefreshAutonomousAgentImportTrace)

	// Execute
	headers := map[string]string{"X-Unified-UI-Autonomous-Agent-API-Key": "test-api-key"}
	path := "/tenants/" + testutils.TestTenantID + "/autonomous-agents/auto-agent-123/traces/" + existingTrace.ID + "/import/refresh"
	w := testutils.PerformRequest(router, "PUT", path, nil, headers)

	// Assert - should fail because trace belongs to different agent
	testutils.AssertStatusCode(t, http.StatusForbidden, w)

	mockPlatform.AssertExpectations(t)
}

func TestTracesHandler_RefreshAutonomousAgentImportTrace_InvalidAPIKey(t *testing.T) {
	// Setup
	mockDocDB := mocks.NewMockDocDBClient()
	mockPlatform := &mocks.MockPlatformClient{}

	// Mock platform client returns unauthorized error
	mockPlatform.On("GetAutonomousAgentConfig", mock.Anything, testutils.TestTenantID, "auto-agent-123", "invalid-api-key").
		Return(nil, fmt.Errorf("unauthorized: invalid API key"))

	handler := createTestTracesHandler(mockDocDB, mockPlatform)

	router := testutils.SetupTestRouter()
	router.PUT("/tenants/:tenantId/autonomous-agents/:agentId/traces/:traceId/import/refresh", autonomousAgentAPIKeyMiddleware(), handler.RefreshAutonomousAgentImportTrace)

	// Execute
	headers := map[string]string{"X-Unified-UI-Autonomous-Agent-API-Key": "invalid-api-key"}
	path := "/tenants/" + testutils.TestTenantID + "/autonomous-agents/auto-agent-123/traces/some-trace-id/import/refresh"
	w := testutils.PerformRequest(router, "PUT", path, nil, headers)

	// Assert
	testutils.AssertStatusCode(t, http.StatusUnauthorized, w)

	mockPlatform.AssertExpectations(t)
}
