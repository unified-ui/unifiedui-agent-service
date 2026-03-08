// Package handlers_test provides unit tests for GetMessages handler.
package handlers_test

import (
	"errors"
	"fmt"
	"net/http"
	"testing"
	"time"

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

func createGetMessagesHandler(mockDocDB *mocks.MockDocDBClient) *handlers.MessagesHandler {
	mockPlatform := &mocks.MockPlatformClient{}
	mockSession := &mocks.MockSessionService{}
	mockConfigCache := &mocks.MockConfigCacheService{}
	mockAI := &mocks.MockAIService{}
	agentFactory := agents.NewFactory()
	importService := traceimport.NewImportService(mockDocDB)
	return handlers.NewMessagesHandler(mockDocDB, mockPlatform, agentFactory, mockSession, mockConfigCache, importService, mockAI)
}

func createTestMessages(count int) []*models.Message {
	messages := make([]*models.Message, count)
	now := time.Now().UTC()
	for i := 0; i < count; i++ {
		msgType := models.MessageTypeUser
		if i%2 == 1 {
			msgType = models.MessageTypeAssistant
		}
		messages[i] = &models.Message{
			ID:             fmt.Sprintf("msg-%d", i),
			Type:           msgType,
			TenantID:       testutils.TestTenantID,
			ConversationID: testutils.TestConversationID,
			ChatAgentID:    testutils.TestChatAgentID,
			UserID:         testutils.TestUserID,
			Content:        fmt.Sprintf("Message content %d", i),
			CreatedAt:      now.Add(time.Duration(-i) * time.Minute),
			UpdatedAt:      now.Add(time.Duration(-i) * time.Minute),
		}
	}
	return messages
}

func TestGetMessages_Success(t *testing.T) {
	mockDocDB := mocks.NewMockDocDBClient()
	testMessages := createTestMessages(3)

	expectedOpts := &docdb.ListMessagesOptions{
		ConversationID: testutils.TestConversationID,
		TenantID:       testutils.TestTenantID,
		Limit:          25, // DefaultMessagesLimit
		Skip:           0,
		OrderBy:        docdb.SortOrderDesc,
	}

	mockDocDB.GetMessagesCollection().On("List", mock.Anything, expectedOpts).Return(testMessages, nil)

	handler := createGetMessagesHandler(mockDocDB)

	router := testutils.SetupTestRouter()
	router.GET("/tenants/:tenantId/conversation/messages", handler.GetMessages)

	w := testutils.PerformRequest(
		router, "GET",
		fmt.Sprintf("/tenants/%s/conversation/messages?conversationId=%s", testutils.TestTenantID, testutils.TestConversationID),
		nil, nil,
	)

	testutils.AssertStatusCode(t, http.StatusOK, w)

	var response handlers.GetMessagesResponse
	testutils.ParseJSONResponse(t, w, &response)

	assert.Len(t, response.Messages, 3)
	assert.Equal(t, "msg-0", response.Messages[0].ID)
	assert.Equal(t, "Message content 0", response.Messages[0].Content)
	assert.Equal(t, testutils.TestConversationID, response.Messages[0].ConversationID)

	mockDocDB.GetMessagesCollection().AssertExpectations(t)
}

func TestGetMessages_WithLimitPagination(t *testing.T) {
	mockDocDB := mocks.NewMockDocDBClient()
	testMessages := createTestMessages(5)

	expectedOpts := &docdb.ListMessagesOptions{
		ConversationID: testutils.TestConversationID,
		TenantID:       testutils.TestTenantID,
		Limit:          5,
		Skip:           0,
		OrderBy:        docdb.SortOrderDesc,
	}

	mockDocDB.GetMessagesCollection().On("List", mock.Anything, expectedOpts).Return(testMessages, nil)

	handler := createGetMessagesHandler(mockDocDB)

	router := testutils.SetupTestRouter()
	router.GET("/tenants/:tenantId/conversation/messages", handler.GetMessages)

	w := testutils.PerformRequest(
		router, "GET",
		fmt.Sprintf("/tenants/%s/conversation/messages?conversationId=%s&limit=5", testutils.TestTenantID, testutils.TestConversationID),
		nil, nil,
	)

	testutils.AssertStatusCode(t, http.StatusOK, w)

	var response handlers.GetMessagesResponse
	testutils.ParseJSONResponse(t, w, &response)

	assert.Len(t, response.Messages, 5)

	mockDocDB.GetMessagesCollection().AssertExpectations(t)
}

func TestGetMessages_WithSkipPagination(t *testing.T) {
	mockDocDB := mocks.NewMockDocDBClient()
	testMessages := createTestMessages(2)

	expectedOpts := &docdb.ListMessagesOptions{
		ConversationID: testutils.TestConversationID,
		TenantID:       testutils.TestTenantID,
		Limit:          10,
		Skip:           5,
		OrderBy:        docdb.SortOrderDesc,
	}

	mockDocDB.GetMessagesCollection().On("List", mock.Anything, expectedOpts).Return(testMessages, nil)

	handler := createGetMessagesHandler(mockDocDB)

	router := testutils.SetupTestRouter()
	router.GET("/tenants/:tenantId/conversation/messages", handler.GetMessages)

	w := testutils.PerformRequest(
		router, "GET",
		fmt.Sprintf("/tenants/%s/conversation/messages?conversationId=%s&limit=10&skip=5", testutils.TestTenantID, testutils.TestConversationID),
		nil, nil,
	)

	testutils.AssertStatusCode(t, http.StatusOK, w)

	var response handlers.GetMessagesResponse
	testutils.ParseJSONResponse(t, w, &response)

	assert.Len(t, response.Messages, 2)

	mockDocDB.GetMessagesCollection().AssertExpectations(t)
}

func TestGetMessages_EmptyResult(t *testing.T) {
	mockDocDB := mocks.NewMockDocDBClient()
	emptyMessages := []*models.Message{}

	expectedOpts := &docdb.ListMessagesOptions{
		ConversationID: testutils.TestConversationID,
		TenantID:       testutils.TestTenantID,
		Limit:          25,
		Skip:           0,
		OrderBy:        docdb.SortOrderDesc,
	}

	mockDocDB.GetMessagesCollection().On("List", mock.Anything, expectedOpts).Return(emptyMessages, nil)

	handler := createGetMessagesHandler(mockDocDB)

	router := testutils.SetupTestRouter()
	router.GET("/tenants/:tenantId/conversation/messages", handler.GetMessages)

	w := testutils.PerformRequest(
		router, "GET",
		fmt.Sprintf("/tenants/%s/conversation/messages?conversationId=%s", testutils.TestTenantID, testutils.TestConversationID),
		nil, nil,
	)

	testutils.AssertStatusCode(t, http.StatusOK, w)

	var response handlers.GetMessagesResponse
	testutils.ParseJSONResponse(t, w, &response)

	assert.Len(t, response.Messages, 0)
	assert.NotNil(t, response.Messages) // Should be empty array, not null

	mockDocDB.GetMessagesCollection().AssertExpectations(t)
}

func TestGetMessages_MissingConversationID(t *testing.T) {
	mockDocDB := mocks.NewMockDocDBClient()

	handler := createGetMessagesHandler(mockDocDB)

	router := testutils.SetupTestRouter()
	router.GET("/tenants/:tenantId/conversation/messages", handler.GetMessages)

	w := testutils.PerformRequest(
		router, "GET",
		fmt.Sprintf("/tenants/%s/conversation/messages", testutils.TestTenantID),
		nil, nil,
	)

	testutils.AssertStatusCode(t, http.StatusBadRequest, w)
}

func TestGetMessages_InvalidLimit(t *testing.T) {
	mockDocDB := mocks.NewMockDocDBClient()

	handler := createGetMessagesHandler(mockDocDB)

	router := testutils.SetupTestRouter()
	router.GET("/tenants/:tenantId/conversation/messages", handler.GetMessages)

	// Limit exceeds max (100)
	w := testutils.PerformRequest(
		router, "GET",
		fmt.Sprintf("/tenants/%s/conversation/messages?conversationId=%s&limit=200", testutils.TestTenantID, testutils.TestConversationID),
		nil, nil,
	)

	testutils.AssertStatusCode(t, http.StatusBadRequest, w)
}

func TestGetMessages_InvalidSkip(t *testing.T) {
	mockDocDB := mocks.NewMockDocDBClient()

	handler := createGetMessagesHandler(mockDocDB)

	router := testutils.SetupTestRouter()
	router.GET("/tenants/:tenantId/conversation/messages", handler.GetMessages)

	// Negative skip value
	w := testutils.PerformRequest(
		router, "GET",
		fmt.Sprintf("/tenants/%s/conversation/messages?conversationId=%s&skip=-1", testutils.TestTenantID, testutils.TestConversationID),
		nil, nil,
	)

	testutils.AssertStatusCode(t, http.StatusBadRequest, w)
}

func TestGetMessages_DatabaseError(t *testing.T) {
	mockDocDB := mocks.NewMockDocDBClient()

	expectedOpts := &docdb.ListMessagesOptions{
		ConversationID: testutils.TestConversationID,
		TenantID:       testutils.TestTenantID,
		Limit:          25,
		Skip:           0,
		OrderBy:        docdb.SortOrderDesc,
	}

	mockDocDB.GetMessagesCollection().On("List", mock.Anything, expectedOpts).Return(nil, errors.New("database connection failed"))

	handler := createGetMessagesHandler(mockDocDB)

	router := testutils.SetupTestRouter()
	router.GET("/tenants/:tenantId/conversation/messages", handler.GetMessages)

	w := testutils.PerformRequest(
		router, "GET",
		fmt.Sprintf("/tenants/%s/conversation/messages?conversationId=%s", testutils.TestTenantID, testutils.TestConversationID),
		nil, nil,
	)

	testutils.AssertStatusCode(t, http.StatusInternalServerError, w)

	mockDocDB.GetMessagesCollection().AssertExpectations(t)
}

func TestGetMessages_MessageFieldsMapping(t *testing.T) {
	mockDocDB := mocks.NewMockDocDBClient()
	now := time.Now().UTC()

	testMessage := &models.Message{
		ID:             "msg-123",
		Type:           models.MessageTypeAssistant,
		TenantID:       testutils.TestTenantID,
		ConversationID: testutils.TestConversationID,
		ChatAgentID:    testutils.TestChatAgentID,
		UserID:         testutils.TestUserID,
		UserMessageID:  "user-msg-456",
		Content:        "Assistant response content",
		Status:         models.MessageStatusSuccess,
		ErrorMessage:   "",
		StatusTraces:   []models.StatusTrace{{Name: "trace1"}},
		Metadata:       &models.AssistantMetadata{},
		AttachmentsMetadata: []models.AttachmentMetadata{
			{FileName: "file.txt", FileType: "text/plain", FileSize: 1024, FileCategory: "document"},
		},
		CreatedAt: now,
		UpdatedAt: now,
	}

	expectedOpts := &docdb.ListMessagesOptions{
		ConversationID: testutils.TestConversationID,
		TenantID:       testutils.TestTenantID,
		Limit:          25,
		Skip:           0,
		OrderBy:        docdb.SortOrderDesc,
	}

	mockDocDB.GetMessagesCollection().On("List", mock.Anything, expectedOpts).Return([]*models.Message{testMessage}, nil)

	handler := createGetMessagesHandler(mockDocDB)

	router := testutils.SetupTestRouter()
	router.GET("/tenants/:tenantId/conversation/messages", handler.GetMessages)

	w := testutils.PerformRequest(
		router, "GET",
		fmt.Sprintf("/tenants/%s/conversation/messages?conversationId=%s", testutils.TestTenantID, testutils.TestConversationID),
		nil, nil,
	)

	testutils.AssertStatusCode(t, http.StatusOK, w)

	var response handlers.GetMessagesResponse
	testutils.ParseJSONResponse(t, w, &response)

	assert.Len(t, response.Messages, 1)
	msg := response.Messages[0]

	assert.Equal(t, "msg-123", msg.ID)
	assert.Equal(t, models.MessageTypeAssistant, msg.Type)
	assert.Equal(t, testutils.TestConversationID, msg.ConversationID)
	assert.Equal(t, testutils.TestChatAgentID, msg.ChatAgentID)
	assert.Equal(t, "Assistant response content", msg.Content)
	assert.Equal(t, "user-msg-456", msg.UserMessageID)
	assert.Equal(t, models.MessageStatusSuccess, msg.Status)
	assert.Len(t, msg.StatusTraces, 1)
	assert.NotNil(t, msg.Metadata)
	assert.Len(t, msg.AttachmentsMetadata, 1)
	assert.Equal(t, "file.txt", msg.AttachmentsMetadata[0].FileName)

	mockDocDB.GetMessagesCollection().AssertExpectations(t)
}

func TestGetMessages_LimitAtBoundary(t *testing.T) {
	testCases := []struct {
		name           string
		limit          int
		expectedLimit  int64
		expectedStatus int
	}{
		{
			name:           "minimum limit (1)",
			limit:          1,
			expectedLimit:  1,
			expectedStatus: http.StatusOK,
		},
		{
			name:           "maximum limit (100)",
			limit:          100,
			expectedLimit:  100,
			expectedStatus: http.StatusOK,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			mockDocDB := mocks.NewMockDocDBClient()
			testMessages := createTestMessages(1)

			expectedOpts := &docdb.ListMessagesOptions{
				ConversationID: testutils.TestConversationID,
				TenantID:       testutils.TestTenantID,
				Limit:          tc.expectedLimit,
				Skip:           0,
				OrderBy:        docdb.SortOrderDesc,
			}

			mockDocDB.GetMessagesCollection().On("List", mock.Anything, expectedOpts).Return(testMessages, nil)

			handler := createGetMessagesHandler(mockDocDB)

			router := testutils.SetupTestRouter()
			router.GET("/tenants/:tenantId/conversation/messages", handler.GetMessages)

			w := testutils.PerformRequest(
				router, "GET",
				fmt.Sprintf("/tenants/%s/conversation/messages?conversationId=%s&limit=%d", testutils.TestTenantID, testutils.TestConversationID, tc.limit),
				nil, nil,
			)

			testutils.AssertStatusCode(t, tc.expectedStatus, w)
			mockDocDB.GetMessagesCollection().AssertExpectations(t)
		})
	}
}

func TestGetMessages_DifferentTenant(t *testing.T) {
	mockDocDB := mocks.NewMockDocDBClient()
	differentTenantID := "different-tenant-id"
	testMessages := createTestMessages(1)

	expectedOpts := &docdb.ListMessagesOptions{
		ConversationID: testutils.TestConversationID,
		TenantID:       differentTenantID,
		Limit:          25,
		Skip:           0,
		OrderBy:        docdb.SortOrderDesc,
	}

	mockDocDB.GetMessagesCollection().On("List", mock.Anything, expectedOpts).Return(testMessages, nil)

	handler := createGetMessagesHandler(mockDocDB)

	router := testutils.SetupTestRouter()
	router.GET("/tenants/:tenantId/conversation/messages", handler.GetMessages)

	w := testutils.PerformRequest(
		router, "GET",
		fmt.Sprintf("/tenants/%s/conversation/messages?conversationId=%s", differentTenantID, testutils.TestConversationID),
		nil, nil,
	)

	testutils.AssertStatusCode(t, http.StatusOK, w)
	mockDocDB.GetMessagesCollection().AssertExpectations(t)
}
