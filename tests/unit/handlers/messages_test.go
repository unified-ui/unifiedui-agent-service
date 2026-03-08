// Package handlers_test provides unit tests for message edit/delete handlers.
package handlers_test

import (
	"fmt"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"github.com/unifiedui/agent-service/internal/api/handlers"
	"github.com/unifiedui/agent-service/internal/core/docdb"
	"github.com/unifiedui/agent-service/internal/services/agents"
	"github.com/unifiedui/agent-service/internal/services/ai"
	"github.com/unifiedui/agent-service/internal/services/traceimport"
	"github.com/unifiedui/agent-service/tests/mocks"
	"github.com/unifiedui/agent-service/tests/testutils"
)

func createTestMessagesHandler(mockDocDB *mocks.MockDocDBClient) *handlers.MessagesHandler {
	mockPlatform := &mocks.MockPlatformClient{}
	mockSession := &mocks.MockSessionService{}
	mockConfigCache := &mocks.MockConfigCacheService{}
	mockAI := &mocks.MockAIService{}
	agentFactory := agents.NewFactory()
	importService := traceimport.NewImportService(mockDocDB)
	return handlers.NewMessagesHandler(mockDocDB, mockPlatform, agentFactory, mockSession, mockConfigCache, importService, mockAI)
}

func TestMessagesHandler_DeleteMessage_Success(t *testing.T) {
	mockDocDB := mocks.NewMockDocDBClient()
	testMessage := testutils.NewTestUserMessage()

	mockDocDB.GetMessagesCollection().On("Get", mock.Anything, testutils.TestMessageID).Return(testMessage, nil)
	mockDocDB.GetMessagesCollection().On("Delete", mock.Anything, &docdb.DeleteMessagesOptions{
		MessageID: testutils.TestMessageID,
		TenantID:  testutils.TestTenantID,
	}).Return(int64(1), nil)

	assistantMsg := testutils.NewTestAssistantMessage()
	mockDocDB.GetMessagesCollection().On("GetByUserMessageID", mock.Anything, testutils.TestMessageID).Return(assistantMsg, nil)
	mockDocDB.GetMessagesCollection().On("Delete", mock.Anything, &docdb.DeleteMessagesOptions{
		MessageID: assistantMsg.ID,
		TenantID:  testutils.TestTenantID,
	}).Return(int64(1), nil)

	handler := createTestMessagesHandler(mockDocDB)

	router := testutils.SetupTestRouter()
	router.DELETE("/tenants/:tenantId/conversations/:conversationId/messages/:messageId", handler.DeleteMessage)

	w := testutils.PerformRequest(
		router, "DELETE",
		fmt.Sprintf("/tenants/%s/conversations/%s/messages/%s", testutils.TestTenantID, testutils.TestConversationID, testutils.TestMessageID),
		nil, nil,
	)

	testutils.AssertStatusCode(t, http.StatusNoContent, w)
	mockDocDB.GetMessagesCollection().AssertExpectations(t)
}

func TestMessagesHandler_DeleteMessage_AssistantOnly_Success(t *testing.T) {
	mockDocDB := mocks.NewMockDocDBClient()
	testMessage := testutils.NewTestAssistantMessage()

	mockDocDB.GetMessagesCollection().On("Get", mock.Anything, testMessage.ID).Return(testMessage, nil)
	mockDocDB.GetMessagesCollection().On("Delete", mock.Anything, &docdb.DeleteMessagesOptions{
		MessageID: testMessage.ID,
		TenantID:  testutils.TestTenantID,
	}).Return(int64(1), nil)

	handler := createTestMessagesHandler(mockDocDB)

	router := testutils.SetupTestRouter()
	router.DELETE("/tenants/:tenantId/conversations/:conversationId/messages/:messageId", handler.DeleteMessage)

	w := testutils.PerformRequest(
		router, "DELETE",
		fmt.Sprintf("/tenants/%s/conversations/%s/messages/%s", testutils.TestTenantID, testutils.TestConversationID, testMessage.ID),
		nil, nil,
	)

	testutils.AssertStatusCode(t, http.StatusNoContent, w)
	mockDocDB.GetMessagesCollection().AssertExpectations(t)
}

func TestMessagesHandler_DeleteMessage_NotFound(t *testing.T) {
	mockDocDB := mocks.NewMockDocDBClient()
	mockDocDB.GetMessagesCollection().On("Get", mock.Anything, "non-existent").Return(nil, nil)

	handler := createTestMessagesHandler(mockDocDB)

	router := testutils.SetupTestRouter()
	router.DELETE("/tenants/:tenantId/conversations/:conversationId/messages/:messageId", handler.DeleteMessage)

	w := testutils.PerformRequest(
		router, "DELETE",
		fmt.Sprintf("/tenants/%s/conversations/%s/messages/%s", testutils.TestTenantID, testutils.TestConversationID, "non-existent"),
		nil, nil,
	)

	testutils.AssertStatusCode(t, http.StatusNotFound, w)
}

func TestMessagesHandler_DeleteMessage_WrongTenant(t *testing.T) {
	mockDocDB := mocks.NewMockDocDBClient()
	testMessage := testutils.NewTestUserMessage()

	mockDocDB.GetMessagesCollection().On("Get", mock.Anything, testutils.TestMessageID).Return(testMessage, nil)

	handler := createTestMessagesHandler(mockDocDB)

	router := testutils.SetupTestRouter()
	router.DELETE("/tenants/:tenantId/conversations/:conversationId/messages/:messageId", handler.DeleteMessage)

	w := testutils.PerformRequest(
		router, "DELETE",
		fmt.Sprintf("/tenants/%s/conversations/%s/messages/%s", "wrong-tenant", testutils.TestConversationID, testutils.TestMessageID),
		nil, nil,
	)

	testutils.AssertStatusCode(t, http.StatusNotFound, w)
}

func TestMessagesHandler_EditMessage_Success(t *testing.T) {
	mockDocDB := mocks.NewMockDocDBClient()
	testMessage := testutils.NewTestUserMessage()

	mockDocDB.GetMessagesCollection().On("Get", mock.Anything, testutils.TestMessageID).Return(testMessage, nil)
	mockDocDB.GetMessagesCollection().On("Update", mock.Anything, mock.Anything).Return(nil)

	handler := createTestMessagesHandler(mockDocDB)

	router := testutils.SetupTestRouter()
	router.PUT("/tenants/:tenantId/conversations/:conversationId/messages/:messageId", handler.EditMessage)

	reqBody := handlers.EditMessageRequest{Content: "Updated message content"}

	w := testutils.PerformRequest(
		router, "PUT",
		fmt.Sprintf("/tenants/%s/conversations/%s/messages/%s", testutils.TestTenantID, testutils.TestConversationID, testutils.TestMessageID),
		reqBody, nil,
	)

	testutils.AssertStatusCode(t, http.StatusOK, w)

	var response handlers.MessageResponse
	testutils.ParseJSONResponse(t, w, &response)

	assert.Equal(t, "Updated message content", response.Content)
	assert.Equal(t, testutils.TestMessageID, response.ID)
	mockDocDB.GetMessagesCollection().AssertExpectations(t)
}

func TestMessagesHandler_EditMessage_NotFound(t *testing.T) {
	mockDocDB := mocks.NewMockDocDBClient()
	mockDocDB.GetMessagesCollection().On("Get", mock.Anything, "non-existent").Return(nil, nil)

	handler := createTestMessagesHandler(mockDocDB)

	router := testutils.SetupTestRouter()
	router.PUT("/tenants/:tenantId/conversations/:conversationId/messages/:messageId", handler.EditMessage)

	reqBody := handlers.EditMessageRequest{Content: "Updated content"}

	w := testutils.PerformRequest(
		router, "PUT",
		fmt.Sprintf("/tenants/%s/conversations/%s/messages/%s", testutils.TestTenantID, testutils.TestConversationID, "non-existent"),
		reqBody, nil,
	)

	testutils.AssertStatusCode(t, http.StatusNotFound, w)
}

func TestMessagesHandler_EditMessage_WrongTenant(t *testing.T) {
	mockDocDB := mocks.NewMockDocDBClient()
	testMessage := testutils.NewTestUserMessage()

	mockDocDB.GetMessagesCollection().On("Get", mock.Anything, testutils.TestMessageID).Return(testMessage, nil)

	handler := createTestMessagesHandler(mockDocDB)

	router := testutils.SetupTestRouter()
	router.PUT("/tenants/:tenantId/conversations/:conversationId/messages/:messageId", handler.EditMessage)

	reqBody := handlers.EditMessageRequest{Content: "Updated content"}

	w := testutils.PerformRequest(
		router, "PUT",
		fmt.Sprintf("/tenants/%s/conversations/%s/messages/%s", "wrong-tenant", testutils.TestConversationID, testutils.TestMessageID),
		reqBody, nil,
	)

	testutils.AssertStatusCode(t, http.StatusNotFound, w)
}

func TestMessagesHandler_EditMessage_AssistantForbidden(t *testing.T) {
	mockDocDB := mocks.NewMockDocDBClient()
	testMessage := testutils.NewTestAssistantMessage()

	mockDocDB.GetMessagesCollection().On("Get", mock.Anything, testMessage.ID).Return(testMessage, nil)

	handler := createTestMessagesHandler(mockDocDB)

	router := testutils.SetupTestRouter()
	router.PUT("/tenants/:tenantId/conversations/:conversationId/messages/:messageId", handler.EditMessage)

	reqBody := handlers.EditMessageRequest{Content: "Trying to edit assistant message"}

	w := testutils.PerformRequest(
		router, "PUT",
		fmt.Sprintf("/tenants/%s/conversations/%s/messages/%s", testutils.TestTenantID, testutils.TestConversationID, testMessage.ID),
		reqBody, nil,
	)

	testutils.AssertStatusCode(t, http.StatusForbidden, w)
}

func TestMessagesHandler_EditMessage_EmptyContent(t *testing.T) {
	mockDocDB := mocks.NewMockDocDBClient()

	handler := createTestMessagesHandler(mockDocDB)

	router := testutils.SetupTestRouter()
	router.PUT("/tenants/:tenantId/conversations/:conversationId/messages/:messageId", handler.EditMessage)

	reqBody := handlers.EditMessageRequest{Content: ""}

	w := testutils.PerformRequest(
		router, "PUT",
		fmt.Sprintf("/tenants/%s/conversations/%s/messages/%s", testutils.TestTenantID, testutils.TestConversationID, testutils.TestMessageID),
		reqBody, nil,
	)

	testutils.AssertStatusCode(t, http.StatusBadRequest, w)
}

// Ensure mocks satisfy interfaces at compile time.
var _ ai.Service = (*mocks.MockAIService)(nil)
