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

func createTestDataHandler(mockDocDB *mocks.MockDocDBClient) *handlers.DataHandler {
	return handlers.NewDataHandler(mockDocDB)
}

// ===========================================================================
// DeleteConversationData
// ===========================================================================

func TestDataHandler_DeleteConversationData_Success(t *testing.T) {
	mockDocDB := mocks.NewMockDocDBClient()
	mockDocDB.GetMessagesCollection().On("Delete", mock.Anything, &docdb.DeleteMessagesOptions{
		ConversationID: testutils.TestConversationID,
		TenantID:       testutils.TestTenantID,
	}).Return(int64(5), nil)
	mockDocDB.GetTracesCollection().On("DeleteByConversation", mock.Anything, testutils.TestTenantID, testutils.TestConversationID).Return(nil)

	handler := createTestDataHandler(mockDocDB)

	router := testutils.SetupTestRouter()
	router.DELETE("/tenants/:tenantId/conversations/:conversationId/data", handler.DeleteConversationData)

	w := testutils.PerformRequest(router, "DELETE",
		"/tenants/"+testutils.TestTenantID+"/conversations/"+testutils.TestConversationID+"/data",
		nil, nil,
	)

	testutils.AssertStatusCode(t, http.StatusNoContent, w)
	mockDocDB.GetMessagesCollection().AssertExpectations(t)
	mockDocDB.GetTracesCollection().AssertExpectations(t)
}

func TestDataHandler_DeleteConversationData_MessagesDeleteError(t *testing.T) {
	mockDocDB := mocks.NewMockDocDBClient()
	mockDocDB.GetMessagesCollection().On("Delete", mock.Anything, mock.Anything).Return(int64(0), assert.AnError)

	handler := createTestDataHandler(mockDocDB)

	router := testutils.SetupTestRouter()
	router.DELETE("/tenants/:tenantId/conversations/:conversationId/data", handler.DeleteConversationData)

	w := testutils.PerformRequest(router, "DELETE",
		"/tenants/"+testutils.TestTenantID+"/conversations/"+testutils.TestConversationID+"/data",
		nil, nil,
	)

	testutils.AssertStatusCode(t, http.StatusInternalServerError, w)
}

func TestDataHandler_DeleteConversationData_TracesDeleteError(t *testing.T) {
	mockDocDB := mocks.NewMockDocDBClient()
	mockDocDB.GetMessagesCollection().On("Delete", mock.Anything, mock.Anything).Return(int64(3), nil)
	mockDocDB.GetTracesCollection().On("DeleteByConversation", mock.Anything, mock.Anything, mock.Anything).Return(assert.AnError)

	handler := createTestDataHandler(mockDocDB)

	router := testutils.SetupTestRouter()
	router.DELETE("/tenants/:tenantId/conversations/:conversationId/data", handler.DeleteConversationData)

	w := testutils.PerformRequest(router, "DELETE",
		"/tenants/"+testutils.TestTenantID+"/conversations/"+testutils.TestConversationID+"/data",
		nil, nil,
	)

	testutils.AssertStatusCode(t, http.StatusInternalServerError, w)
}

// ===========================================================================
// DeleteWorkflowData
// ===========================================================================

func TestDataHandler_DeleteWorkflowData_Success(t *testing.T) {
	mockDocDB := mocks.NewMockDocDBClient()
	mockDocDB.GetTracesCollection().On("DeleteByWorkflow", mock.Anything, testutils.TestTenantID, "agent-123").Return(nil)

	handler := createTestDataHandler(mockDocDB)

	router := testutils.SetupTestRouter()
	router.DELETE("/tenants/:tenantId/workflows/:agentId/data", handler.DeleteWorkflowData)

	w := testutils.PerformRequest(router, "DELETE",
		"/tenants/"+testutils.TestTenantID+"/workflows/agent-123/data",
		nil, nil,
	)

	testutils.AssertStatusCode(t, http.StatusNoContent, w)
	mockDocDB.GetTracesCollection().AssertExpectations(t)
}

func TestDataHandler_DeleteWorkflowData_TracesDeleteError(t *testing.T) {
	mockDocDB := mocks.NewMockDocDBClient()
	mockDocDB.GetTracesCollection().On("DeleteByWorkflow", mock.Anything, mock.Anything, mock.Anything).Return(assert.AnError)

	handler := createTestDataHandler(mockDocDB)

	router := testutils.SetupTestRouter()
	router.DELETE("/tenants/:tenantId/workflows/:agentId/data", handler.DeleteWorkflowData)

	w := testutils.PerformRequest(router, "DELETE",
		"/tenants/"+testutils.TestTenantID+"/workflows/agent-123/data",
		nil, nil,
	)

	testutils.AssertStatusCode(t, http.StatusInternalServerError, w)
}
