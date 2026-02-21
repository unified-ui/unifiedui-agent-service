// Package handlers_test provides extended unit tests for message handlers.
package handlers_test

import (
	"errors"
	"fmt"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"github.com/unifiedui/agent-service/internal/api/handlers"
	"github.com/unifiedui/agent-service/internal/core/docdb"
	"github.com/unifiedui/agent-service/internal/domain/models"
	"github.com/unifiedui/agent-service/internal/services/agents"
	"github.com/unifiedui/agent-service/internal/services/traceimport"
	"github.com/unifiedui/agent-service/tests/mocks"
	"github.com/unifiedui/agent-service/tests/testutils"
)

func createMessagesHandlerForExtendedTests(mockDocDB *mocks.MockDocDBClient) *handlers.MessagesHandler {
	mockPlatform := &mocks.MockPlatformClient{}
	mockSession := &mocks.MockSessionService{}
	mockAI := &mocks.MockAIService{}
	agentFactory := agents.NewFactory()
	importService := traceimport.NewImportService(mockDocDB)
	return handlers.NewMessagesHandler(mockDocDB, mockPlatform, agentFactory, mockSession, importService, mockAI)
}

// ============================================================================
// DeleteMessage Extended Tests
// ============================================================================

func TestDeleteMessage_GetMessageDBError(t *testing.T) {
	mockDocDB := mocks.NewMockDocDBClient()

	mockDocDB.GetMessagesCollection().On("Get", mock.Anything, testutils.TestMessageID).
		Return(nil, errors.New("database connection error"))

	handler := createMessagesHandlerForExtendedTests(mockDocDB)

	router := testutils.SetupTestRouter()
	router.DELETE("/tenants/:tenantId/conversations/:conversationId/messages/:messageId", handler.DeleteMessage)

	w := testutils.PerformRequest(
		router, "DELETE",
		fmt.Sprintf("/tenants/%s/conversations/%s/messages/%s", testutils.TestTenantID, testutils.TestConversationID, testutils.TestMessageID),
		nil, nil,
	)

	testutils.AssertStatusCode(t, http.StatusInternalServerError, w)
	mockDocDB.GetMessagesCollection().AssertExpectations(t)
}

func TestDeleteMessage_DeleteDBError(t *testing.T) {
	mockDocDB := mocks.NewMockDocDBClient()
	testMessage := testutils.NewTestUserMessage()

	mockDocDB.GetMessagesCollection().On("Get", mock.Anything, testutils.TestMessageID).Return(testMessage, nil)
	mockDocDB.GetMessagesCollection().On("Delete", mock.Anything, &docdb.DeleteMessagesOptions{
		MessageID: testutils.TestMessageID,
		TenantID:  testutils.TestTenantID,
	}).Return(int64(0), errors.New("database delete error"))

	handler := createMessagesHandlerForExtendedTests(mockDocDB)

	router := testutils.SetupTestRouter()
	router.DELETE("/tenants/:tenantId/conversations/:conversationId/messages/:messageId", handler.DeleteMessage)

	w := testutils.PerformRequest(
		router, "DELETE",
		fmt.Sprintf("/tenants/%s/conversations/%s/messages/%s", testutils.TestTenantID, testutils.TestConversationID, testutils.TestMessageID),
		nil, nil,
	)

	testutils.AssertStatusCode(t, http.StatusInternalServerError, w)
	mockDocDB.GetMessagesCollection().AssertExpectations(t)
}

func TestDeleteMessage_UserMessage_NoAssociatedAssistantMessage(t *testing.T) {
	mockDocDB := mocks.NewMockDocDBClient()
	testMessage := testutils.NewTestUserMessage()

	mockDocDB.GetMessagesCollection().On("Get", mock.Anything, testutils.TestMessageID).Return(testMessage, nil)
	mockDocDB.GetMessagesCollection().On("Delete", mock.Anything, &docdb.DeleteMessagesOptions{
		MessageID: testutils.TestMessageID,
		TenantID:  testutils.TestTenantID,
	}).Return(int64(1), nil)

	// Return nil for no associated assistant message
	mockDocDB.GetMessagesCollection().On("GetByUserMessageID", mock.Anything, testutils.TestMessageID).
		Return(nil, nil)

	handler := createMessagesHandlerForExtendedTests(mockDocDB)

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

func TestDeleteMessage_UserMessage_GetAssistantMessageError(t *testing.T) {
	mockDocDB := mocks.NewMockDocDBClient()
	testMessage := testutils.NewTestUserMessage()

	mockDocDB.GetMessagesCollection().On("Get", mock.Anything, testutils.TestMessageID).Return(testMessage, nil)
	mockDocDB.GetMessagesCollection().On("Delete", mock.Anything, &docdb.DeleteMessagesOptions{
		MessageID: testutils.TestMessageID,
		TenantID:  testutils.TestTenantID,
	}).Return(int64(1), nil)

	// Return error when getting associated assistant message - should still succeed
	mockDocDB.GetMessagesCollection().On("GetByUserMessageID", mock.Anything, testutils.TestMessageID).
		Return(nil, errors.New("database error"))

	handler := createMessagesHandlerForExtendedTests(mockDocDB)

	router := testutils.SetupTestRouter()
	router.DELETE("/tenants/:tenantId/conversations/:conversationId/messages/:messageId", handler.DeleteMessage)

	w := testutils.PerformRequest(
		router, "DELETE",
		fmt.Sprintf("/tenants/%s/conversations/%s/messages/%s", testutils.TestTenantID, testutils.TestConversationID, testutils.TestMessageID),
		nil, nil,
	)

	// Should still succeed - error getting assistant message is ignored
	testutils.AssertStatusCode(t, http.StatusNoContent, w)
	mockDocDB.GetMessagesCollection().AssertExpectations(t)
}

func TestDeleteMessage_UserMessage_DeleteAssistantMessageError(t *testing.T) {
	mockDocDB := mocks.NewMockDocDBClient()
	testMessage := testutils.NewTestUserMessage()
	assistantMsg := testutils.NewTestAssistantMessage()

	mockDocDB.GetMessagesCollection().On("Get", mock.Anything, testutils.TestMessageID).Return(testMessage, nil)
	mockDocDB.GetMessagesCollection().On("Delete", mock.Anything, &docdb.DeleteMessagesOptions{
		MessageID: testutils.TestMessageID,
		TenantID:  testutils.TestTenantID,
	}).Return(int64(1), nil)

	mockDocDB.GetMessagesCollection().On("GetByUserMessageID", mock.Anything, testutils.TestMessageID).
		Return(assistantMsg, nil)

	// Error deleting assistant message - should still succeed
	mockDocDB.GetMessagesCollection().On("Delete", mock.Anything, &docdb.DeleteMessagesOptions{
		MessageID: assistantMsg.ID,
		TenantID:  testutils.TestTenantID,
	}).Return(int64(0), errors.New("delete assistant error"))

	handler := createMessagesHandlerForExtendedTests(mockDocDB)

	router := testutils.SetupTestRouter()
	router.DELETE("/tenants/:tenantId/conversations/:conversationId/messages/:messageId", handler.DeleteMessage)

	w := testutils.PerformRequest(
		router, "DELETE",
		fmt.Sprintf("/tenants/%s/conversations/%s/messages/%s", testutils.TestTenantID, testutils.TestConversationID, testutils.TestMessageID),
		nil, nil,
	)

	// Should still succeed - error deleting assistant message is ignored
	testutils.AssertStatusCode(t, http.StatusNoContent, w)
	mockDocDB.GetMessagesCollection().AssertExpectations(t)
}

func TestDeleteMessage_DifferentTenant_Unauthorized(t *testing.T) {
	mockDocDB := mocks.NewMockDocDBClient()
	testMessage := testutils.NewTestUserMessage()
	testMessage.TenantID = "different-tenant-id"

	mockDocDB.GetMessagesCollection().On("Get", mock.Anything, testutils.TestMessageID).Return(testMessage, nil)

	handler := createMessagesHandlerForExtendedTests(mockDocDB)

	router := testutils.SetupTestRouter()
	router.DELETE("/tenants/:tenantId/conversations/:conversationId/messages/:messageId", handler.DeleteMessage)

	w := testutils.PerformRequest(
		router, "DELETE",
		fmt.Sprintf("/tenants/%s/conversations/%s/messages/%s", testutils.TestTenantID, testutils.TestConversationID, testutils.TestMessageID),
		nil, nil,
	)

	// Should return NotFound (not exposing that message exists for other tenant)
	testutils.AssertStatusCode(t, http.StatusNotFound, w)
	mockDocDB.GetMessagesCollection().AssertExpectations(t)
}

// ============================================================================
// EditMessage Extended Tests
// ============================================================================

func TestEditMessage_InvalidJSON(t *testing.T) {
	mockDocDB := mocks.NewMockDocDBClient()

	handler := createMessagesHandlerForExtendedTests(mockDocDB)

	router := testutils.SetupTestRouter()
	router.PUT("/tenants/:tenantId/conversations/:conversationId/messages/:messageId", handler.EditMessage)

	// nil body triggers validation error
	w := testutils.PerformRequest(
		router, "PUT",
		fmt.Sprintf("/tenants/%s/conversations/%s/messages/%s", testutils.TestTenantID, testutils.TestConversationID, testutils.TestMessageID),
		nil, nil,
	)

	testutils.AssertStatusCode(t, http.StatusBadRequest, w)
}

func TestEditMessage_MalformedJSON(t *testing.T) {
	mockDocDB := mocks.NewMockDocDBClient()

	handler := createMessagesHandlerForExtendedTests(mockDocDB)

	router := testutils.SetupTestRouter()
	router.PUT("/tenants/:tenantId/conversations/:conversationId/messages/:messageId", handler.EditMessage)

	// Send malformed JSON - wrong type for content
	reqBody := map[string]interface{}{
		"content": 12345, // Wrong type - should be string
	}

	w := testutils.PerformRequest(
		router, "PUT",
		fmt.Sprintf("/tenants/%s/conversations/%s/messages/%s", testutils.TestTenantID, testutils.TestConversationID, testutils.TestMessageID),
		reqBody, nil,
	)

	testutils.AssertStatusCode(t, http.StatusBadRequest, w)
}

func TestEditMessage_GetMessageDBError(t *testing.T) {
	mockDocDB := mocks.NewMockDocDBClient()

	mockDocDB.GetMessagesCollection().On("Get", mock.Anything, testutils.TestMessageID).
		Return(nil, errors.New("database connection error"))

	handler := createMessagesHandlerForExtendedTests(mockDocDB)

	router := testutils.SetupTestRouter()
	router.PUT("/tenants/:tenantId/conversations/:conversationId/messages/:messageId", handler.EditMessage)

	reqBody := handlers.EditMessageRequest{Content: "Updated content"}

	w := testutils.PerformRequest(
		router, "PUT",
		fmt.Sprintf("/tenants/%s/conversations/%s/messages/%s", testutils.TestTenantID, testutils.TestConversationID, testutils.TestMessageID),
		reqBody, nil,
	)

	testutils.AssertStatusCode(t, http.StatusInternalServerError, w)
	mockDocDB.GetMessagesCollection().AssertExpectations(t)
}

func TestEditMessage_UpdateDBError(t *testing.T) {
	mockDocDB := mocks.NewMockDocDBClient()
	testMessage := testutils.NewTestUserMessage()

	mockDocDB.GetMessagesCollection().On("Get", mock.Anything, testutils.TestMessageID).Return(testMessage, nil)
	mockDocDB.GetMessagesCollection().On("Update", mock.Anything, mock.Anything).
		Return(errors.New("database update error"))

	handler := createMessagesHandlerForExtendedTests(mockDocDB)

	router := testutils.SetupTestRouter()
	router.PUT("/tenants/:tenantId/conversations/:conversationId/messages/:messageId", handler.EditMessage)

	reqBody := handlers.EditMessageRequest{Content: "Updated content"}

	w := testutils.PerformRequest(
		router, "PUT",
		fmt.Sprintf("/tenants/%s/conversations/%s/messages/%s", testutils.TestTenantID, testutils.TestConversationID, testutils.TestMessageID),
		reqBody, nil,
	)

	testutils.AssertStatusCode(t, http.StatusInternalServerError, w)
	mockDocDB.GetMessagesCollection().AssertExpectations(t)
}

func TestEditMessage_ContentTooLong(t *testing.T) {
	mockDocDB := mocks.NewMockDocDBClient()

	handler := createMessagesHandlerForExtendedTests(mockDocDB)

	router := testutils.SetupTestRouter()
	router.PUT("/tenants/:tenantId/conversations/:conversationId/messages/:messageId", handler.EditMessage)

	// Create content that exceeds the max of 32000 characters
	longContent := make([]byte, 32001)
	for i := range longContent {
		longContent[i] = 'a'
	}

	reqBody := handlers.EditMessageRequest{Content: string(longContent)}

	w := testutils.PerformRequest(
		router, "PUT",
		fmt.Sprintf("/tenants/%s/conversations/%s/messages/%s", testutils.TestTenantID, testutils.TestConversationID, testutils.TestMessageID),
		reqBody, nil,
	)

	testutils.AssertStatusCode(t, http.StatusBadRequest, w)
}

func TestEditMessage_DifferentTenant_Unauthorized(t *testing.T) {
	mockDocDB := mocks.NewMockDocDBClient()
	testMessage := testutils.NewTestUserMessage()
	testMessage.TenantID = "different-tenant-id"

	mockDocDB.GetMessagesCollection().On("Get", mock.Anything, testutils.TestMessageID).Return(testMessage, nil)

	handler := createMessagesHandlerForExtendedTests(mockDocDB)

	router := testutils.SetupTestRouter()
	router.PUT("/tenants/:tenantId/conversations/:conversationId/messages/:messageId", handler.EditMessage)

	reqBody := handlers.EditMessageRequest{Content: "Trying to update other tenant's message"}

	w := testutils.PerformRequest(
		router, "PUT",
		fmt.Sprintf("/tenants/%s/conversations/%s/messages/%s", testutils.TestTenantID, testutils.TestConversationID, testutils.TestMessageID),
		reqBody, nil,
	)

	// Should return NotFound (not exposing that message exists for other tenant)
	testutils.AssertStatusCode(t, http.StatusNotFound, w)
	mockDocDB.GetMessagesCollection().AssertExpectations(t)
}

func TestEditMessage_VerifyUpdatedAtTimestamp(t *testing.T) {
	mockDocDB := mocks.NewMockDocDBClient()
	testMessage := testutils.NewTestUserMessage()
	originalUpdatedAt := testMessage.UpdatedAt

	mockDocDB.GetMessagesCollection().On("Get", mock.Anything, testutils.TestMessageID).Return(testMessage, nil)
	mockDocDB.GetMessagesCollection().On("Update", mock.Anything, mock.MatchedBy(func(msg *models.Message) bool {
		// Verify that UpdatedAt was updated
		return msg.UpdatedAt.After(originalUpdatedAt) || msg.UpdatedAt.Equal(originalUpdatedAt)
	})).Return(nil)

	handler := createMessagesHandlerForExtendedTests(mockDocDB)

	router := testutils.SetupTestRouter()
	router.PUT("/tenants/:tenantId/conversations/:conversationId/messages/:messageId", handler.EditMessage)

	reqBody := handlers.EditMessageRequest{Content: "New content"}

	w := testutils.PerformRequest(
		router, "PUT",
		fmt.Sprintf("/tenants/%s/conversations/%s/messages/%s", testutils.TestTenantID, testutils.TestConversationID, testutils.TestMessageID),
		reqBody, nil,
	)

	testutils.AssertStatusCode(t, http.StatusOK, w)

	var response handlers.MessageResponse
	testutils.ParseJSONResponse(t, w, &response)

	assert.Equal(t, "New content", response.Content)
	assert.Equal(t, testutils.TestMessageID, response.ID)
	mockDocDB.GetMessagesCollection().AssertExpectations(t)
}

func TestEditMessage_VerifyContentUpdated(t *testing.T) {
	mockDocDB := mocks.NewMockDocDBClient()
	testMessage := testutils.NewTestUserMessage()
	newContent := "This is the updated message content"

	mockDocDB.GetMessagesCollection().On("Get", mock.Anything, testutils.TestMessageID).Return(testMessage, nil)
	mockDocDB.GetMessagesCollection().On("Update", mock.Anything, mock.MatchedBy(func(msg *models.Message) bool {
		return msg.Content == newContent
	})).Return(nil)

	handler := createMessagesHandlerForExtendedTests(mockDocDB)

	router := testutils.SetupTestRouter()
	router.PUT("/tenants/:tenantId/conversations/:conversationId/messages/:messageId", handler.EditMessage)

	reqBody := handlers.EditMessageRequest{Content: newContent}

	w := testutils.PerformRequest(
		router, "PUT",
		fmt.Sprintf("/tenants/%s/conversations/%s/messages/%s", testutils.TestTenantID, testutils.TestConversationID, testutils.TestMessageID),
		reqBody, nil,
	)

	testutils.AssertStatusCode(t, http.StatusOK, w)

	var response handlers.MessageResponse
	testutils.ParseJSONResponse(t, w, &response)

	assert.Equal(t, newContent, response.Content)
	mockDocDB.GetMessagesCollection().AssertExpectations(t)
}

func TestEditMessage_OnlyUserMessagesAllowed(t *testing.T) {
	mockDocDB := mocks.NewMockDocDBClient()

	// Create an assistant message (editing should be forbidden)
	testMessage := testutils.NewTestAssistantMessage()

	mockDocDB.GetMessagesCollection().On("Get", mock.Anything, testMessage.ID).Return(testMessage, nil)

	handler := createMessagesHandlerForExtendedTests(mockDocDB)

	router := testutils.SetupTestRouter()
	router.PUT("/tenants/:tenantId/conversations/:conversationId/messages/:messageId", handler.EditMessage)

	reqBody := handlers.EditMessageRequest{Content: "Trying to edit assistant message"}

	w := testutils.PerformRequest(
		router, "PUT",
		fmt.Sprintf("/tenants/%s/conversations/%s/messages/%s", testutils.TestTenantID, testutils.TestConversationID, testMessage.ID),
		reqBody, nil,
	)

	testutils.AssertStatusCode(t, http.StatusForbidden, w)
	mockDocDB.GetMessagesCollection().AssertExpectations(t)
}

func TestEditMessage_ResponseContainsAllFields(t *testing.T) {
	mockDocDB := mocks.NewMockDocDBClient()
	testMessage := testutils.NewTestUserMessage()
	testMessage.StatusTraces = []models.StatusTrace{
		{Type: "processing", Timestamp: testMessage.CreatedAt},
	}

	mockDocDB.GetMessagesCollection().On("Get", mock.Anything, testutils.TestMessageID).Return(testMessage, nil)
	mockDocDB.GetMessagesCollection().On("Update", mock.Anything, mock.Anything).Return(nil)

	handler := createMessagesHandlerForExtendedTests(mockDocDB)

	router := testutils.SetupTestRouter()
	router.PUT("/tenants/:tenantId/conversations/:conversationId/messages/:messageId", handler.EditMessage)

	reqBody := handlers.EditMessageRequest{Content: "Updated content"}

	w := testutils.PerformRequest(
		router, "PUT",
		fmt.Sprintf("/tenants/%s/conversations/%s/messages/%s", testutils.TestTenantID, testutils.TestConversationID, testutils.TestMessageID),
		reqBody, nil,
	)

	testutils.AssertStatusCode(t, http.StatusOK, w)

	var response handlers.MessageResponse
	testutils.ParseJSONResponse(t, w, &response)

	assert.Equal(t, testutils.TestMessageID, response.ID)
	assert.Equal(t, models.MessageTypeUser, response.Type)
	assert.Equal(t, testutils.TestConversationID, response.ConversationID)
	assert.Equal(t, testutils.TestChatAgentID, response.ChatAgentID)
	assert.Equal(t, "Updated content", response.Content)
	assert.Equal(t, testutils.TestUserID, response.UserID)
	assert.NotZero(t, response.CreatedAt)
	assert.NotZero(t, response.UpdatedAt)

	mockDocDB.GetMessagesCollection().AssertExpectations(t)
}

func TestEditMessage_SpecialCharactersInContent(t *testing.T) {
	mockDocDB := mocks.NewMockDocDBClient()
	testMessage := testutils.NewTestUserMessage()

	mockDocDB.GetMessagesCollection().On("Get", mock.Anything, testutils.TestMessageID).Return(testMessage, nil)
	mockDocDB.GetMessagesCollection().On("Update", mock.Anything, mock.Anything).Return(nil)

	handler := createMessagesHandlerForExtendedTests(mockDocDB)

	router := testutils.SetupTestRouter()
	router.PUT("/tenants/:tenantId/conversations/:conversationId/messages/:messageId", handler.EditMessage)

	// Content with special characters, unicode, and newlines
	specialContent := "Hello! How are you?\n\nLine with quotes and apostrophes\nSpecial chars"

	reqBody := handlers.EditMessageRequest{Content: specialContent}

	w := testutils.PerformRequest(
		router, "PUT",
		fmt.Sprintf("/tenants/%s/conversations/%s/messages/%s", testutils.TestTenantID, testutils.TestConversationID, testutils.TestMessageID),
		reqBody, nil,
	)

	testutils.AssertStatusCode(t, http.StatusOK, w)

	var response handlers.MessageResponse
	testutils.ParseJSONResponse(t, w, &response)

	assert.Equal(t, specialContent, response.Content)
	mockDocDB.GetMessagesCollection().AssertExpectations(t)
}
