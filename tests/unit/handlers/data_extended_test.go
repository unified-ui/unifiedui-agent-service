// Package handlers_test provides extended unit tests for data handlers.
package handlers_test

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"github.com/unifiedui/agent-service/internal/api/handlers"
	"github.com/unifiedui/agent-service/internal/core/docdb"
	"github.com/unifiedui/agent-service/tests/mocks"
	"github.com/unifiedui/agent-service/tests/testutils"
)

// ===========================================================================
// DeleteConversationData Extended Tests
// ===========================================================================

func TestDataHandler_DeleteConversationData_MissingTenantID(t *testing.T) {
	mockDocDB := mocks.NewMockDocDBClient()
	handler := handlers.NewDataHandler(mockDocDB)

	router := testutils.SetupTestRouter()
	// Route without tenantId parameter
	router.DELETE("/conversations/:conversationId/data", handler.DeleteConversationData)

	w := testutils.PerformRequest(router, "DELETE",
		"/conversations/"+testutils.TestConversationID+"/data",
		nil, nil,
	)

	testutils.AssertStatusCode(t, http.StatusBadRequest, w)
}

func TestDataHandler_DeleteConversationData_MissingConversationID(t *testing.T) {
	mockDocDB := mocks.NewMockDocDBClient()
	handler := handlers.NewDataHandler(mockDocDB)

	router := testutils.SetupTestRouter()
	// Route without conversationId parameter
	router.DELETE("/tenants/:tenantId/data", handler.DeleteConversationData)

	w := testutils.PerformRequest(router, "DELETE",
		"/tenants/"+testutils.TestTenantID+"/data",
		nil, nil,
	)

	testutils.AssertStatusCode(t, http.StatusBadRequest, w)
}

func TestDataHandler_DeleteConversationData_EmptyTenantID(t *testing.T) {
	mockDocDB := mocks.NewMockDocDBClient()
	handler := handlers.NewDataHandler(mockDocDB)

	router := testutils.SetupTestRouter()
	router.DELETE("/tenants/:tenantId/conversations/:conversationId/data", handler.DeleteConversationData)

	// Provide empty tenant ID via path
	w := testutils.PerformRequest(router, "DELETE",
		"/tenants//conversations/"+testutils.TestConversationID+"/data",
		nil, nil,
	)

	// Handler validates empty parameters and returns validation error
	testutils.AssertStatusCode(t, http.StatusBadRequest, w)
}

func TestDataHandler_DeleteConversationData_EmptyConversationID(t *testing.T) {
	mockDocDB := mocks.NewMockDocDBClient()
	handler := handlers.NewDataHandler(mockDocDB)

	router := testutils.SetupTestRouter()
	router.DELETE("/tenants/:tenantId/conversations/:conversationId/data", handler.DeleteConversationData)

	// Provide empty conversation ID via path
	w := testutils.PerformRequest(router, "DELETE",
		"/tenants/"+testutils.TestTenantID+"/conversations//data",
		nil, nil,
	)

	// Handler validates empty parameters and returns validation error
	testutils.AssertStatusCode(t, http.StatusBadRequest, w)
}

func TestDataHandler_DeleteConversationData_MultipleConversations(t *testing.T) {
	// Test deleting data for different conversation IDs
	testCases := []struct {
		name           string
		conversationID string
		deletedCount   int64
	}{
		{
			name:           "conversation with few messages",
			conversationID: "conv-small-1",
			deletedCount:   2,
		},
		{
			name:           "conversation with many messages",
			conversationID: "conv-large-2",
			deletedCount:   100,
		},
		{
			name:           "conversation with no messages",
			conversationID: "conv-empty-3",
			deletedCount:   0,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			mockDocDB := mocks.NewMockDocDBClient()
			mockDocDB.GetMessagesCollection().On("Delete", mock.Anything, &docdb.DeleteMessagesOptions{
				ConversationID: tc.conversationID,
				TenantID:       testutils.TestTenantID,
			}).Return(tc.deletedCount, nil)
			mockDocDB.GetTracesCollection().On("DeleteByConversation", mock.Anything, testutils.TestTenantID, tc.conversationID).Return(nil)

			handler := handlers.NewDataHandler(mockDocDB)

			router := testutils.SetupTestRouter()
			router.DELETE("/tenants/:tenantId/conversations/:conversationId/data", handler.DeleteConversationData)

			w := testutils.PerformRequest(router, "DELETE",
				"/tenants/"+testutils.TestTenantID+"/conversations/"+tc.conversationID+"/data",
				nil, nil,
			)

			testutils.AssertStatusCode(t, http.StatusNoContent, w)
			mockDocDB.GetMessagesCollection().AssertExpectations(t)
			mockDocDB.GetTracesCollection().AssertExpectations(t)
		})
	}
}

func TestDataHandler_DeleteConversationData_DifferentTenants(t *testing.T) {
	testCases := []struct {
		name     string
		tenantID string
	}{
		{name: "tenant1", tenantID: "tenant-001"},
		{name: "tenant2", tenantID: "tenant-002"},
		{name: "tenant with special chars", tenantID: "tenant_special-123"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			mockDocDB := mocks.NewMockDocDBClient()
			mockDocDB.GetMessagesCollection().On("Delete", mock.Anything, &docdb.DeleteMessagesOptions{
				ConversationID: testutils.TestConversationID,
				TenantID:       tc.tenantID,
			}).Return(int64(1), nil)
			mockDocDB.GetTracesCollection().On("DeleteByConversation", mock.Anything, tc.tenantID, testutils.TestConversationID).Return(nil)

			handler := handlers.NewDataHandler(mockDocDB)

			router := testutils.SetupTestRouter()
			router.DELETE("/tenants/:tenantId/conversations/:conversationId/data", handler.DeleteConversationData)

			w := testutils.PerformRequest(router, "DELETE",
				"/tenants/"+tc.tenantID+"/conversations/"+testutils.TestConversationID+"/data",
				nil, nil,
			)

			testutils.AssertStatusCode(t, http.StatusNoContent, w)
			mockDocDB.GetMessagesCollection().AssertExpectations(t)
			mockDocDB.GetTracesCollection().AssertExpectations(t)
		})
	}
}

func TestDataHandler_DeleteConversationData_MessagesDeleteSuccessTracesDeleteError(t *testing.T) {
	// This tests the scenario where messages delete succeeds but traces delete fails
	mockDocDB := mocks.NewMockDocDBClient()
	mockDocDB.GetMessagesCollection().On("Delete", mock.Anything, &docdb.DeleteMessagesOptions{
		ConversationID: testutils.TestConversationID,
		TenantID:       testutils.TestTenantID,
	}).Return(int64(10), nil)
	mockDocDB.GetTracesCollection().On("DeleteByConversation", mock.Anything, testutils.TestTenantID, testutils.TestConversationID).Return(assert.AnError)

	handler := handlers.NewDataHandler(mockDocDB)

	router := testutils.SetupTestRouter()
	router.DELETE("/tenants/:tenantId/conversations/:conversationId/data", handler.DeleteConversationData)

	w := testutils.PerformRequest(router, "DELETE",
		"/tenants/"+testutils.TestTenantID+"/conversations/"+testutils.TestConversationID+"/data",
		nil, nil,
	)

	testutils.AssertStatusCode(t, http.StatusInternalServerError, w)
	mockDocDB.GetMessagesCollection().AssertExpectations(t)
	mockDocDB.GetTracesCollection().AssertExpectations(t)
}

// ===========================================================================
// DeleteAutonomousAgentData Extended Tests
// ===========================================================================

func TestDataHandler_DeleteAutonomousAgentData_MissingTenantID(t *testing.T) {
	mockDocDB := mocks.NewMockDocDBClient()
	handler := handlers.NewDataHandler(mockDocDB)

	router := testutils.SetupTestRouter()
	// Route without tenantId parameter
	router.DELETE("/autonomous-agents/:agentId/data", handler.DeleteAutonomousAgentData)

	w := testutils.PerformRequest(router, "DELETE",
		"/autonomous-agents/agent-123/data",
		nil, nil,
	)

	testutils.AssertStatusCode(t, http.StatusBadRequest, w)
}

func TestDataHandler_DeleteAutonomousAgentData_MissingAgentID(t *testing.T) {
	mockDocDB := mocks.NewMockDocDBClient()
	handler := handlers.NewDataHandler(mockDocDB)

	router := testutils.SetupTestRouter()
	// Route without agentId parameter
	router.DELETE("/tenants/:tenantId/autonomous-agents-data", handler.DeleteAutonomousAgentData)

	w := testutils.PerformRequest(router, "DELETE",
		"/tenants/"+testutils.TestTenantID+"/autonomous-agents-data",
		nil, nil,
	)

	testutils.AssertStatusCode(t, http.StatusBadRequest, w)
}

func TestDataHandler_DeleteAutonomousAgentData_EmptyTenantID(t *testing.T) {
	mockDocDB := mocks.NewMockDocDBClient()
	handler := handlers.NewDataHandler(mockDocDB)

	router := testutils.SetupTestRouter()
	router.DELETE("/tenants/:tenantId/autonomous-agents/:agentId/data", handler.DeleteAutonomousAgentData)

	// Provide empty tenant ID via path
	w := testutils.PerformRequest(router, "DELETE",
		"/tenants//autonomous-agents/agent-123/data",
		nil, nil,
	)

	// Handler validates empty parameters and returns validation error
	testutils.AssertStatusCode(t, http.StatusBadRequest, w)
}

func TestDataHandler_DeleteAutonomousAgentData_EmptyAgentID(t *testing.T) {
	mockDocDB := mocks.NewMockDocDBClient()
	handler := handlers.NewDataHandler(mockDocDB)

	router := testutils.SetupTestRouter()
	router.DELETE("/tenants/:tenantId/autonomous-agents/:agentId/data", handler.DeleteAutonomousAgentData)

	// Provide empty agent ID via path
	w := testutils.PerformRequest(router, "DELETE",
		"/tenants/"+testutils.TestTenantID+"/autonomous-agents//data",
		nil, nil,
	)

	// Handler validates empty parameters and returns validation error
	testutils.AssertStatusCode(t, http.StatusBadRequest, w)
}

func TestDataHandler_DeleteAutonomousAgentData_MultipleAgents(t *testing.T) {
	testCases := []struct {
		name    string
		agentID string
	}{
		{name: "standard agent", agentID: "agent-001"},
		{name: "agent with UUID", agentID: "550e8400-e29b-41d4-a716-446655440000"},
		{name: "agent with special chars", agentID: "agent_special-123"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			mockDocDB := mocks.NewMockDocDBClient()
			mockDocDB.GetTracesCollection().On("DeleteByAutonomousAgent", mock.Anything, testutils.TestTenantID, tc.agentID).Return(nil)

			handler := handlers.NewDataHandler(mockDocDB)

			router := testutils.SetupTestRouter()
			router.DELETE("/tenants/:tenantId/autonomous-agents/:agentId/data", handler.DeleteAutonomousAgentData)

			w := testutils.PerformRequest(router, "DELETE",
				"/tenants/"+testutils.TestTenantID+"/autonomous-agents/"+tc.agentID+"/data",
				nil, nil,
			)

			testutils.AssertStatusCode(t, http.StatusNoContent, w)
			mockDocDB.GetTracesCollection().AssertExpectations(t)
		})
	}
}

func TestDataHandler_DeleteAutonomousAgentData_DifferentTenants(t *testing.T) {
	testCases := []struct {
		name     string
		tenantID string
	}{
		{name: "tenant1", tenantID: "tenant-aaa"},
		{name: "tenant2", tenantID: "tenant-bbb"},
		{name: "tenant with UUID", tenantID: "123e4567-e89b-12d3-a456-426614174000"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			mockDocDB := mocks.NewMockDocDBClient()
			mockDocDB.GetTracesCollection().On("DeleteByAutonomousAgent", mock.Anything, tc.tenantID, "agent-123").Return(nil)

			handler := handlers.NewDataHandler(mockDocDB)

			router := testutils.SetupTestRouter()
			router.DELETE("/tenants/:tenantId/autonomous-agents/:agentId/data", handler.DeleteAutonomousAgentData)

			w := testutils.PerformRequest(router, "DELETE",
				"/tenants/"+tc.tenantID+"/autonomous-agents/agent-123/data",
				nil, nil,
			)

			testutils.AssertStatusCode(t, http.StatusNoContent, w)
			mockDocDB.GetTracesCollection().AssertExpectations(t)
		})
	}
}

func TestDataHandler_DeleteAutonomousAgentData_NonExistentAgent(t *testing.T) {
	// Deleting data for non-existent agent should still succeed (no-op)
	mockDocDB := mocks.NewMockDocDBClient()
	mockDocDB.GetTracesCollection().On("DeleteByAutonomousAgent", mock.Anything, testutils.TestTenantID, "non-existent-agent").Return(nil)

	handler := handlers.NewDataHandler(mockDocDB)

	router := testutils.SetupTestRouter()
	router.DELETE("/tenants/:tenantId/autonomous-agents/:agentId/data", handler.DeleteAutonomousAgentData)

	w := testutils.PerformRequest(router, "DELETE",
		"/tenants/"+testutils.TestTenantID+"/autonomous-agents/non-existent-agent/data",
		nil, nil,
	)

	testutils.AssertStatusCode(t, http.StatusNoContent, w)
	mockDocDB.GetTracesCollection().AssertExpectations(t)
}
