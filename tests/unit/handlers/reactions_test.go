// Package handlers_test provides unit tests for reaction handlers.
package handlers_test

import (
	"fmt"
	"net/http"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"github.com/unifiedui/agent-service/internal/api/handlers"
	"github.com/unifiedui/agent-service/internal/core/docdb"
	"github.com/unifiedui/agent-service/internal/domain/models"
	"github.com/unifiedui/agent-service/internal/services/platform"
	"github.com/unifiedui/agent-service/tests/mocks"
	"github.com/unifiedui/agent-service/tests/testutils"
)

func createTestReactionsHandler(mockDocDB *mocks.MockDocDBClient, mockPlatform *mocks.MockPlatformClient) *handlers.ReactionsHandler {
	return handlers.NewReactionsHandler(mockDocDB, mockPlatform)
}

func authTestMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader != "" && len(authHeader) > 7 {
			c.Set("auth_token", authHeader[7:])
		}
		c.Next()
	}
}

func TestReactionsHandler_UpsertReaction_Success(t *testing.T) {
	mockDocDB := mocks.NewMockDocDBClient()
	mockPlatform := &mocks.MockPlatformClient{}

	testMessage := testutils.NewTestAssistantMessage()

	mockPlatform.On("GetMe", mock.Anything, mock.Anything).Return(&platform.UserInfo{ID: testutils.TestUserID}, nil)
	mockDocDB.GetMessagesCollection().On("Get", mock.Anything, testutils.TestMessageID).Return(testMessage, nil)
	mockDocDB.GetReactionsCollection().On("Upsert", mock.Anything, mock.Anything).Return(nil)

	handler := createTestReactionsHandler(mockDocDB, mockPlatform)

	router := testutils.SetupTestRouter()
	router.Use(authTestMiddleware())
	router.POST("/tenants/:tenantId/conversations/:conversationId/messages/:messageId/reactions", handler.UpsertReaction)

	reqBody := handlers.UpsertReactionRequest{
		Reaction:     models.ReactionThumbsUp,
		FeedbackText: "Great response!",
	}

	headers := map[string]string{"Authorization": "Bearer test-token"}
	w := testutils.PerformRequest(
		router, "POST",
		fmt.Sprintf("/tenants/%s/conversations/%s/messages/%s/reactions", testutils.TestTenantID, testutils.TestConversationID, testutils.TestMessageID),
		reqBody, headers,
	)

	testutils.AssertStatusCode(t, http.StatusOK, w)

	var response handlers.ReactionResponse
	testutils.ParseJSONResponse(t, w, &response)

	assert.Equal(t, testutils.TestTenantID, response.TenantID)
	assert.Equal(t, testutils.TestConversationID, response.ConversationID)
	assert.Equal(t, testutils.TestMessageID, response.MessageID)
	assert.Equal(t, testutils.TestUserID, response.UserID)
	assert.Equal(t, models.ReactionThumbsUp, response.Reaction)
	assert.Equal(t, "Great response!", response.FeedbackText)
	assert.NotEmpty(t, response.ID)

	mockDocDB.GetMessagesCollection().AssertExpectations(t)
	mockDocDB.GetReactionsCollection().AssertExpectations(t)
	mockPlatform.AssertExpectations(t)
}

func TestReactionsHandler_UpsertReaction_ThumbsDown_Success(t *testing.T) {
	mockDocDB := mocks.NewMockDocDBClient()
	mockPlatform := &mocks.MockPlatformClient{}

	testMessage := testutils.NewTestAssistantMessage()

	mockPlatform.On("GetMe", mock.Anything, mock.Anything).Return(&platform.UserInfo{ID: testutils.TestUserID}, nil)
	mockDocDB.GetMessagesCollection().On("Get", mock.Anything, testutils.TestMessageID).Return(testMessage, nil)
	mockDocDB.GetReactionsCollection().On("Upsert", mock.Anything, mock.Anything).Return(nil)

	handler := createTestReactionsHandler(mockDocDB, mockPlatform)

	router := testutils.SetupTestRouter()
	router.Use(authTestMiddleware())
	router.POST("/tenants/:tenantId/conversations/:conversationId/messages/:messageId/reactions", handler.UpsertReaction)

	reqBody := handlers.UpsertReactionRequest{
		Reaction:     models.ReactionThumbsDown,
		FeedbackText: "Not helpful",
	}

	headers := map[string]string{"Authorization": "Bearer test-token"}
	w := testutils.PerformRequest(
		router, "POST",
		fmt.Sprintf("/tenants/%s/conversations/%s/messages/%s/reactions", testutils.TestTenantID, testutils.TestConversationID, testutils.TestMessageID),
		reqBody, headers,
	)

	testutils.AssertStatusCode(t, http.StatusOK, w)

	var response handlers.ReactionResponse
	testutils.ParseJSONResponse(t, w, &response)

	assert.Equal(t, models.ReactionThumbsDown, response.Reaction)
	assert.Equal(t, "Not helpful", response.FeedbackText)

	mockDocDB.GetReactionsCollection().AssertExpectations(t)
}

func TestReactionsHandler_UpsertReaction_InvalidType(t *testing.T) {
	mockDocDB := mocks.NewMockDocDBClient()
	mockPlatform := &mocks.MockPlatformClient{}

	handler := createTestReactionsHandler(mockDocDB, mockPlatform)

	router := testutils.SetupTestRouter()
	router.Use(authTestMiddleware())
	router.POST("/tenants/:tenantId/conversations/:conversationId/messages/:messageId/reactions", handler.UpsertReaction)

	reqBody := map[string]interface{}{
		"reaction": "invalid_type",
	}

	headers := map[string]string{"Authorization": "Bearer test-token"}
	w := testutils.PerformRequest(
		router, "POST",
		fmt.Sprintf("/tenants/%s/conversations/%s/messages/%s/reactions", testutils.TestTenantID, testutils.TestConversationID, testutils.TestMessageID),
		reqBody, headers,
	)

	testutils.AssertStatusCode(t, http.StatusBadRequest, w)
}

func TestReactionsHandler_UpsertReaction_MessageNotFound(t *testing.T) {
	mockDocDB := mocks.NewMockDocDBClient()
	mockPlatform := &mocks.MockPlatformClient{}

	mockPlatform.On("GetMe", mock.Anything, mock.Anything).Return(&platform.UserInfo{ID: testutils.TestUserID}, nil)
	mockDocDB.GetMessagesCollection().On("Get", mock.Anything, "non-existent-msg").Return(nil, nil)

	handler := createTestReactionsHandler(mockDocDB, mockPlatform)

	router := testutils.SetupTestRouter()
	router.Use(authTestMiddleware())
	router.POST("/tenants/:tenantId/conversations/:conversationId/messages/:messageId/reactions", handler.UpsertReaction)

	reqBody := handlers.UpsertReactionRequest{
		Reaction: models.ReactionThumbsUp,
	}

	headers := map[string]string{"Authorization": "Bearer test-token"}
	w := testutils.PerformRequest(
		router, "POST",
		fmt.Sprintf("/tenants/%s/conversations/%s/messages/%s/reactions", testutils.TestTenantID, testutils.TestConversationID, "non-existent-msg"),
		reqBody, headers,
	)

	testutils.AssertStatusCode(t, http.StatusNotFound, w)

	mockDocDB.GetMessagesCollection().AssertExpectations(t)
	mockPlatform.AssertExpectations(t)
}

func TestReactionsHandler_UpsertReaction_MissingBody(t *testing.T) {
	mockDocDB := mocks.NewMockDocDBClient()
	mockPlatform := &mocks.MockPlatformClient{}

	handler := createTestReactionsHandler(mockDocDB, mockPlatform)

	router := testutils.SetupTestRouter()
	router.Use(authTestMiddleware())
	router.POST("/tenants/:tenantId/conversations/:conversationId/messages/:messageId/reactions", handler.UpsertReaction)

	headers := map[string]string{"Authorization": "Bearer test-token"}
	w := testutils.PerformRequest(
		router, "POST",
		fmt.Sprintf("/tenants/%s/conversations/%s/messages/%s/reactions", testutils.TestTenantID, testutils.TestConversationID, testutils.TestMessageID),
		nil, headers,
	)

	testutils.AssertStatusCode(t, http.StatusBadRequest, w)
}

func TestReactionsHandler_DeleteReaction_Success(t *testing.T) {
	mockDocDB := mocks.NewMockDocDBClient()
	mockPlatform := &mocks.MockPlatformClient{}

	mockPlatform.On("GetMe", mock.Anything, mock.Anything).Return(&platform.UserInfo{ID: testutils.TestUserID}, nil)
	mockDocDB.GetReactionsCollection().On("Delete", mock.Anything, &docdb.DeleteReactionOptions{
		TenantID:  testutils.TestTenantID,
		MessageID: testutils.TestMessageID,
		UserID:    testutils.TestUserID,
	}).Return(nil)

	handler := createTestReactionsHandler(mockDocDB, mockPlatform)

	router := testutils.SetupTestRouter()
	router.Use(authTestMiddleware())
	router.DELETE("/tenants/:tenantId/conversations/:conversationId/messages/:messageId/reactions", handler.DeleteReaction)

	headers := map[string]string{"Authorization": "Bearer test-token"}
	w := testutils.PerformRequest(
		router, "DELETE",
		fmt.Sprintf("/tenants/%s/conversations/%s/messages/%s/reactions", testutils.TestTenantID, testutils.TestConversationID, testutils.TestMessageID),
		nil, headers,
	)

	testutils.AssertStatusCode(t, http.StatusNoContent, w)

	mockDocDB.GetReactionsCollection().AssertExpectations(t)
	mockPlatform.AssertExpectations(t)
}

func TestReactionsHandler_GetReactions_Success(t *testing.T) {
	mockDocDB := mocks.NewMockDocDBClient()
	mockPlatform := &mocks.MockPlatformClient{}

	reactions := []*models.MessageReaction{
		testutils.NewTestReaction(),
	}

	mockDocDB.GetReactionsCollection().On("ListByMessage", mock.Anything, &docdb.ListReactionsOptions{
		TenantID:       testutils.TestTenantID,
		ConversationID: testutils.TestConversationID,
		MessageID:      testutils.TestMessageID,
	}).Return(reactions, nil)

	handler := createTestReactionsHandler(mockDocDB, mockPlatform)

	router := testutils.SetupTestRouter()
	router.GET("/tenants/:tenantId/conversations/:conversationId/messages/:messageId/reactions", handler.GetReactions)

	w := testutils.PerformRequest(
		router, "GET",
		fmt.Sprintf("/tenants/%s/conversations/%s/messages/%s/reactions", testutils.TestTenantID, testutils.TestConversationID, testutils.TestMessageID),
		nil, nil,
	)

	testutils.AssertStatusCode(t, http.StatusOK, w)

	var response handlers.ListReactionsResponse
	testutils.ParseJSONResponse(t, w, &response)

	assert.Len(t, response.Reactions, 1)
	assert.Equal(t, testutils.TestUserID, response.Reactions[0].UserID)
	assert.Equal(t, models.ReactionThumbsUp, response.Reactions[0].Reaction)

	mockDocDB.GetReactionsCollection().AssertExpectations(t)
}

func TestReactionsHandler_GetReactions_Empty(t *testing.T) {
	mockDocDB := mocks.NewMockDocDBClient()
	mockPlatform := &mocks.MockPlatformClient{}

	mockDocDB.GetReactionsCollection().On("ListByMessage", mock.Anything, mock.Anything).Return([]*models.MessageReaction{}, nil)

	handler := createTestReactionsHandler(mockDocDB, mockPlatform)

	router := testutils.SetupTestRouter()
	router.GET("/tenants/:tenantId/conversations/:conversationId/messages/:messageId/reactions", handler.GetReactions)

	w := testutils.PerformRequest(
		router, "GET",
		fmt.Sprintf("/tenants/%s/conversations/%s/messages/%s/reactions", testutils.TestTenantID, testutils.TestConversationID, testutils.TestMessageID),
		nil, nil,
	)

	testutils.AssertStatusCode(t, http.StatusOK, w)

	var response handlers.ListReactionsResponse
	testutils.ParseJSONResponse(t, w, &response)

	assert.Empty(t, response.Reactions)
}
