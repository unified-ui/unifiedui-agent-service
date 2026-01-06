// Package handlers_test provides unit tests for trace handlers.
package handlers_test

import (
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"github.com/unifiedui/agent-service/internal/api/dto"
	"github.com/unifiedui/agent-service/internal/api/handlers"
	"github.com/unifiedui/agent-service/internal/domain/models"
	"github.com/unifiedui/agent-service/internal/services/platform"
	"github.com/unifiedui/agent-service/tests/mocks"
	"github.com/unifiedui/agent-service/tests/testutils"
)

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

	// Mock traces collection
	mockDocDB.GetTracesCollection().On("Create", mock.Anything, mock.Anything).Return(nil)

	handler := handlers.NewTracesHandler(mockDocDB, mockPlatform)

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

	handler := handlers.NewTracesHandler(mockDocDB, mockPlatform)

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

	handler := handlers.NewTracesHandler(mockDocDB, mockPlatform)

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

	handler := handlers.NewTracesHandler(mockDocDB, mockPlatform)

	router := testutils.SetupTestRouter()
	router.POST("/tenants/:tenantId/traces", handler.CreateTrace)

	// Execute
	headers := map[string]string{"Authorization": "Bearer test-token"}
	w := testutils.PerformRequest(router, "POST", "/tenants/"+testutils.TestTenantID+"/traces", createReq, headers)

	// Assert
	testutils.AssertStatusCode(t, http.StatusBadRequest, w)
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

	handler := handlers.NewTracesHandler(mockDocDB, mockPlatform)

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

	handler := handlers.NewTracesHandler(mockDocDB, mockPlatform)

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

	handler := handlers.NewTracesHandler(mockDocDB, mockPlatform)

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

	handler := handlers.NewTracesHandler(mockDocDB, mockPlatform)

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

	handler := handlers.NewTracesHandler(mockDocDB, mockPlatform)

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

	handler := handlers.NewTracesHandler(mockDocDB, mockPlatform)

	router := testutils.SetupTestRouter()
	router.DELETE("/tenants/:tenantId/traces/:traceId", handler.DeleteTrace)

	// Execute
	w := testutils.PerformRequest(router, "DELETE", "/tenants/"+testutils.TestTenantID+"/traces/"+existingTrace.ID, nil, nil)

	// Assert
	testutils.AssertStatusCode(t, http.StatusNoContent, w)

	mockDocDB.GetTracesCollection().AssertExpectations(t)
}

func TestTracesHandler_GetConversationTrace_Success(t *testing.T) {
	// Setup
	mockDocDB := mocks.NewMockDocDBClient()
	mockPlatform := &mocks.MockPlatformClient{}

	existingTrace := testutils.NewTestTrace()

	// GetConversationTrace does NOT call ValidateConversation - it just fetches the trace
	mockDocDB.GetTracesCollection().On("GetByConversation", mock.Anything, mock.Anything, mock.Anything).Return(existingTrace, nil)

	handler := handlers.NewTracesHandler(mockDocDB, mockPlatform)

	router := testutils.SetupTestRouter()
	router.GET("/tenants/:tenantId/conversations/:conversationId/traces", handler.GetConversationTrace)

	// Execute
	headers := map[string]string{"Authorization": "Bearer test-token"}
	path := "/tenants/" + testutils.TestTenantID + "/conversations/" + testutils.TestConversationID + "/traces"
	w := testutils.PerformRequest(router, "GET", path, nil, headers)

	// Assert
	testutils.AssertStatusCode(t, http.StatusOK, w)

	var response dto.TraceResponse
	testutils.ParseJSONResponse(t, w, &response)

	assert.Equal(t, existingTrace.ID, response.ID)

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

	handler := handlers.NewTracesHandler(mockDocDB, mockPlatform)

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

	handler := handlers.NewTracesHandler(mockDocDB, mockPlatform)

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
