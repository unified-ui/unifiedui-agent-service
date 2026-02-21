// Package handlers_test provides extended unit tests for reaction handlers.
package handlers_test

import (
	"errors"
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

// ===========================================================================
// UpsertReaction Extended Tests
// ===========================================================================

func TestReactionsHandler_UpsertReaction_GetMessageDBError(t *testing.T) {
	mockDocDB := mocks.NewMockDocDBClient()
	mockPlatform := &mocks.MockPlatformClient{}

	mockPlatform.On("GetMe", mock.Anything, mock.Anything).Return(&platform.UserInfo{ID: testutils.TestUserID}, nil)
	mockDocDB.GetMessagesCollection().On("Get", mock.Anything, testutils.TestMessageID).Return(nil, errors.New("database connection error"))

	handler := handlers.NewReactionsHandler(mockDocDB, mockPlatform)

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

	testutils.AssertStatusCode(t, http.StatusInternalServerError, w)
	mockDocDB.GetMessagesCollection().AssertExpectations(t)
	mockPlatform.AssertExpectations(t)
}

func TestReactionsHandler_UpsertReaction_UpsertDBError(t *testing.T) {
	mockDocDB := mocks.NewMockDocDBClient()
	mockPlatform := &mocks.MockPlatformClient{}

	testMessage := testutils.NewTestAssistantMessage()

	mockPlatform.On("GetMe", mock.Anything, mock.Anything).Return(&platform.UserInfo{ID: testutils.TestUserID}, nil)
	mockDocDB.GetMessagesCollection().On("Get", mock.Anything, testutils.TestMessageID).Return(testMessage, nil)
	mockDocDB.GetReactionsCollection().On("Upsert", mock.Anything, mock.Anything).Return(errors.New("upsert failed"))

	handler := handlers.NewReactionsHandler(mockDocDB, mockPlatform)

	router := testutils.SetupTestRouter()
	router.Use(authTestMiddleware())
	router.POST("/tenants/:tenantId/conversations/:conversationId/messages/:messageId/reactions", handler.UpsertReaction)

	reqBody := handlers.UpsertReactionRequest{
		Reaction:     models.ReactionThumbsUp,
		FeedbackText: "Test feedback",
	}

	headers := map[string]string{"Authorization": "Bearer test-token"}
	w := testutils.PerformRequest(
		router, "POST",
		fmt.Sprintf("/tenants/%s/conversations/%s/messages/%s/reactions", testutils.TestTenantID, testutils.TestConversationID, testutils.TestMessageID),
		reqBody, headers,
	)

	testutils.AssertStatusCode(t, http.StatusInternalServerError, w)
	mockDocDB.GetMessagesCollection().AssertExpectations(t)
	mockDocDB.GetReactionsCollection().AssertExpectations(t)
	mockPlatform.AssertExpectations(t)
}

func TestReactionsHandler_UpsertReaction_NilPlatformClient(t *testing.T) {
	mockDocDB := mocks.NewMockDocDBClient()

	testMessage := testutils.NewTestAssistantMessage()

	mockDocDB.GetMessagesCollection().On("Get", mock.Anything, testutils.TestMessageID).Return(testMessage, nil)
	mockDocDB.GetReactionsCollection().On("Upsert", mock.Anything, mock.MatchedBy(func(r *models.MessageReaction) bool {
		return r.UserID == "system" // When platformClient is nil, userID should be "system"
	})).Return(nil)

	// Pass nil for platform client
	handler := handlers.NewReactionsHandler(mockDocDB, nil)

	router := testutils.SetupTestRouter()
	router.Use(authTestMiddleware())
	router.POST("/tenants/:tenantId/conversations/:conversationId/messages/:messageId/reactions", handler.UpsertReaction)

	reqBody := handlers.UpsertReactionRequest{
		Reaction: models.ReactionThumbsDown,
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
	assert.Equal(t, "system", response.UserID)
	assert.Equal(t, models.ReactionThumbsDown, response.Reaction)

	mockDocDB.GetMessagesCollection().AssertExpectations(t)
	mockDocDB.GetReactionsCollection().AssertExpectations(t)
}

func TestReactionsHandler_UpsertReaction_PlatformClientGetMeError(t *testing.T) {
	mockDocDB := mocks.NewMockDocDBClient()
	mockPlatform := &mocks.MockPlatformClient{}

	testMessage := testutils.NewTestAssistantMessage()

	// GetMe returns error - should fall back to "system"
	mockPlatform.On("GetMe", mock.Anything, mock.Anything).Return(nil, errors.New("platform service unavailable"))
	mockDocDB.GetMessagesCollection().On("Get", mock.Anything, testutils.TestMessageID).Return(testMessage, nil)
	mockDocDB.GetReactionsCollection().On("Upsert", mock.Anything, mock.MatchedBy(func(r *models.MessageReaction) bool {
		return r.UserID == "system" // Fallback to "system" on error
	})).Return(nil)

	handler := handlers.NewReactionsHandler(mockDocDB, mockPlatform)

	router := testutils.SetupTestRouter()
	router.Use(authTestMiddleware())
	router.POST("/tenants/:tenantId/conversations/:conversationId/messages/:messageId/reactions", handler.UpsertReaction)

	reqBody := handlers.UpsertReactionRequest{
		Reaction:     models.ReactionThumbsUp,
		FeedbackText: "Good answer",
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
	assert.Equal(t, "system", response.UserID)

	mockDocDB.GetMessagesCollection().AssertExpectations(t)
	mockDocDB.GetReactionsCollection().AssertExpectations(t)
	mockPlatform.AssertExpectations(t)
}

func TestReactionsHandler_UpsertReaction_InvalidJSON(t *testing.T) {
	mockDocDB := mocks.NewMockDocDBClient()
	mockPlatform := &mocks.MockPlatformClient{}

	handler := handlers.NewReactionsHandler(mockDocDB, mockPlatform)

	router := testutils.SetupTestRouter()
	router.Use(authTestMiddleware())
	router.POST("/tenants/:tenantId/conversations/:conversationId/messages/:messageId/reactions", handler.UpsertReaction)

	// Send raw invalid JSON string
	headers := map[string]string{
		"Authorization": "Bearer test-token",
		"Content-Type":  "application/json",
	}

	reqBody := map[string]interface{}{
		"reaction": 12345, // Invalid type (should be string)
	}

	w := testutils.PerformRequest(
		router, "POST",
		fmt.Sprintf("/tenants/%s/conversations/%s/messages/%s/reactions", testutils.TestTenantID, testutils.TestConversationID, testutils.TestMessageID),
		reqBody, headers,
	)

	testutils.AssertStatusCode(t, http.StatusBadRequest, w)
}

func TestReactionsHandler_UpsertReaction_EmptyReactionField(t *testing.T) {
	mockDocDB := mocks.NewMockDocDBClient()
	mockPlatform := &mocks.MockPlatformClient{}

	handler := handlers.NewReactionsHandler(mockDocDB, mockPlatform)

	router := testutils.SetupTestRouter()
	router.Use(authTestMiddleware())
	router.POST("/tenants/:tenantId/conversations/:conversationId/messages/:messageId/reactions", handler.UpsertReaction)

	reqBody := map[string]interface{}{
		"feedbackText": "Some feedback without reaction",
	}

	headers := map[string]string{"Authorization": "Bearer test-token"}
	w := testutils.PerformRequest(
		router, "POST",
		fmt.Sprintf("/tenants/%s/conversations/%s/messages/%s/reactions", testutils.TestTenantID, testutils.TestConversationID, testutils.TestMessageID),
		reqBody, headers,
	)

	testutils.AssertStatusCode(t, http.StatusBadRequest, w)
}

func TestReactionsHandler_UpsertReaction_ThumbsDownWithFeedback(t *testing.T) {
	mockDocDB := mocks.NewMockDocDBClient()
	mockPlatform := &mocks.MockPlatformClient{}

	testMessage := testutils.NewTestAssistantMessage()

	mockPlatform.On("GetMe", mock.Anything, mock.Anything).Return(&platform.UserInfo{ID: testutils.TestUserID}, nil)
	mockDocDB.GetMessagesCollection().On("Get", mock.Anything, testutils.TestMessageID).Return(testMessage, nil)
	mockDocDB.GetReactionsCollection().On("Upsert", mock.Anything, mock.MatchedBy(func(r *models.MessageReaction) bool {
		return r.Reaction == models.ReactionThumbsDown && r.FeedbackText == "The response was incorrect."
	})).Return(nil)

	handler := handlers.NewReactionsHandler(mockDocDB, mockPlatform)

	router := testutils.SetupTestRouter()
	router.Use(authTestMiddleware())
	router.POST("/tenants/:tenantId/conversations/:conversationId/messages/:messageId/reactions", handler.UpsertReaction)

	reqBody := handlers.UpsertReactionRequest{
		Reaction:     models.ReactionThumbsDown,
		FeedbackText: "The response was incorrect.",
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
	assert.Equal(t, "The response was incorrect.", response.FeedbackText)

	mockDocDB.GetReactionsCollection().AssertExpectations(t)
}

func TestReactionsHandler_UpsertReaction_WithEmptyFeedbackText(t *testing.T) {
	mockDocDB := mocks.NewMockDocDBClient()
	mockPlatform := &mocks.MockPlatformClient{}

	testMessage := testutils.NewTestAssistantMessage()

	mockPlatform.On("GetMe", mock.Anything, mock.Anything).Return(&platform.UserInfo{ID: testutils.TestUserID}, nil)
	mockDocDB.GetMessagesCollection().On("Get", mock.Anything, testutils.TestMessageID).Return(testMessage, nil)
	mockDocDB.GetReactionsCollection().On("Upsert", mock.Anything, mock.MatchedBy(func(r *models.MessageReaction) bool {
		return r.FeedbackText == ""
	})).Return(nil)

	handler := handlers.NewReactionsHandler(mockDocDB, mockPlatform)

	router := testutils.SetupTestRouter()
	router.Use(authTestMiddleware())
	router.POST("/tenants/:tenantId/conversations/:conversationId/messages/:messageId/reactions", handler.UpsertReaction)

	reqBody := handlers.UpsertReactionRequest{
		Reaction:     models.ReactionThumbsUp,
		FeedbackText: "",
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
	assert.Equal(t, "", response.FeedbackText)
}

// ===========================================================================
// DeleteReaction Extended Tests
// ===========================================================================

func TestReactionsHandler_DeleteReaction_DBError(t *testing.T) {
	mockDocDB := mocks.NewMockDocDBClient()
	mockPlatform := &mocks.MockPlatformClient{}

	mockPlatform.On("GetMe", mock.Anything, mock.Anything).Return(&platform.UserInfo{ID: testutils.TestUserID}, nil)
	mockDocDB.GetReactionsCollection().On("Delete", mock.Anything, &docdb.DeleteReactionOptions{
		TenantID:  testutils.TestTenantID,
		MessageID: testutils.TestMessageID,
		UserID:    testutils.TestUserID,
	}).Return(errors.New("database delete failed"))

	handler := handlers.NewReactionsHandler(mockDocDB, mockPlatform)

	router := testutils.SetupTestRouter()
	router.Use(authTestMiddleware())
	router.DELETE("/tenants/:tenantId/conversations/:conversationId/messages/:messageId/reactions", handler.DeleteReaction)

	headers := map[string]string{"Authorization": "Bearer test-token"}
	w := testutils.PerformRequest(
		router, "DELETE",
		fmt.Sprintf("/tenants/%s/conversations/%s/messages/%s/reactions", testutils.TestTenantID, testutils.TestConversationID, testutils.TestMessageID),
		nil, headers,
	)

	testutils.AssertStatusCode(t, http.StatusInternalServerError, w)
	mockDocDB.GetReactionsCollection().AssertExpectations(t)
	mockPlatform.AssertExpectations(t)
}

func TestReactionsHandler_DeleteReaction_NilPlatformClient(t *testing.T) {
	mockDocDB := mocks.NewMockDocDBClient()

	mockDocDB.GetReactionsCollection().On("Delete", mock.Anything, &docdb.DeleteReactionOptions{
		TenantID:  testutils.TestTenantID,
		MessageID: testutils.TestMessageID,
		UserID:    "system",
	}).Return(nil)

	// Pass nil for platform client
	handler := handlers.NewReactionsHandler(mockDocDB, nil)

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
}

func TestReactionsHandler_DeleteReaction_PlatformGetMeError(t *testing.T) {
	mockDocDB := mocks.NewMockDocDBClient()
	mockPlatform := &mocks.MockPlatformClient{}

	// GetMe returns error, should fallback to "system"
	mockPlatform.On("GetMe", mock.Anything, mock.Anything).Return(nil, errors.New("auth service down"))
	mockDocDB.GetReactionsCollection().On("Delete", mock.Anything, &docdb.DeleteReactionOptions{
		TenantID:  testutils.TestTenantID,
		MessageID: testutils.TestMessageID,
		UserID:    "system",
	}).Return(nil)

	handler := handlers.NewReactionsHandler(mockDocDB, mockPlatform)

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

func TestReactionsHandler_DeleteReaction_DifferentMessageID(t *testing.T) {
	mockDocDB := mocks.NewMockDocDBClient()
	mockPlatform := &mocks.MockPlatformClient{}
	differentMessageID := "different-msg-id-123"

	mockPlatform.On("GetMe", mock.Anything, mock.Anything).Return(&platform.UserInfo{ID: testutils.TestUserID}, nil)
	mockDocDB.GetReactionsCollection().On("Delete", mock.Anything, &docdb.DeleteReactionOptions{
		TenantID:  testutils.TestTenantID,
		MessageID: differentMessageID,
		UserID:    testutils.TestUserID,
	}).Return(nil)

	handler := handlers.NewReactionsHandler(mockDocDB, mockPlatform)

	router := testutils.SetupTestRouter()
	router.Use(authTestMiddleware())
	router.DELETE("/tenants/:tenantId/conversations/:conversationId/messages/:messageId/reactions", handler.DeleteReaction)

	headers := map[string]string{"Authorization": "Bearer test-token"}
	w := testutils.PerformRequest(
		router, "DELETE",
		fmt.Sprintf("/tenants/%s/conversations/%s/messages/%s/reactions", testutils.TestTenantID, testutils.TestConversationID, differentMessageID),
		nil, headers,
	)

	testutils.AssertStatusCode(t, http.StatusNoContent, w)
	mockDocDB.GetReactionsCollection().AssertExpectations(t)
}

// ===========================================================================
// GetReactions Extended Tests
// ===========================================================================

func TestReactionsHandler_GetReactions_DBError(t *testing.T) {
	mockDocDB := mocks.NewMockDocDBClient()
	mockPlatform := &mocks.MockPlatformClient{}

	mockDocDB.GetReactionsCollection().On("ListByMessage", mock.Anything, &docdb.ListReactionsOptions{
		TenantID:       testutils.TestTenantID,
		ConversationID: testutils.TestConversationID,
		MessageID:      testutils.TestMessageID,
	}).Return(nil, errors.New("database query failed"))

	handler := handlers.NewReactionsHandler(mockDocDB, mockPlatform)

	router := testutils.SetupTestRouter()
	router.GET("/tenants/:tenantId/conversations/:conversationId/messages/:messageId/reactions", handler.GetReactions)

	w := testutils.PerformRequest(
		router, "GET",
		fmt.Sprintf("/tenants/%s/conversations/%s/messages/%s/reactions", testutils.TestTenantID, testutils.TestConversationID, testutils.TestMessageID),
		nil, nil,
	)

	testutils.AssertStatusCode(t, http.StatusInternalServerError, w)
	mockDocDB.GetReactionsCollection().AssertExpectations(t)
}

func TestReactionsHandler_GetReactions_MultipleReactions(t *testing.T) {
	mockDocDB := mocks.NewMockDocDBClient()
	mockPlatform := &mocks.MockPlatformClient{}

	reactions := []*models.MessageReaction{
		{
			ID:             "reaction-1",
			TenantID:       testutils.TestTenantID,
			ConversationID: testutils.TestConversationID,
			MessageID:      testutils.TestMessageID,
			UserID:         "user-1",
			Reaction:       models.ReactionThumbsUp,
		},
		{
			ID:             "reaction-2",
			TenantID:       testutils.TestTenantID,
			ConversationID: testutils.TestConversationID,
			MessageID:      testutils.TestMessageID,
			UserID:         "user-2",
			Reaction:       models.ReactionThumbsDown,
			FeedbackText:   "Could be better",
		},
		{
			ID:             "reaction-3",
			TenantID:       testutils.TestTenantID,
			ConversationID: testutils.TestConversationID,
			MessageID:      testutils.TestMessageID,
			UserID:         "user-3",
			Reaction:       models.ReactionThumbsUp,
		},
	}

	mockDocDB.GetReactionsCollection().On("ListByMessage", mock.Anything, &docdb.ListReactionsOptions{
		TenantID:       testutils.TestTenantID,
		ConversationID: testutils.TestConversationID,
		MessageID:      testutils.TestMessageID,
	}).Return(reactions, nil)

	handler := handlers.NewReactionsHandler(mockDocDB, mockPlatform)

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

	assert.Len(t, response.Reactions, 3)
	assert.Equal(t, "user-1", response.Reactions[0].UserID)
	assert.Equal(t, models.ReactionThumbsUp, response.Reactions[0].Reaction)
	assert.Equal(t, "user-2", response.Reactions[1].UserID)
	assert.Equal(t, models.ReactionThumbsDown, response.Reactions[1].Reaction)
	assert.Equal(t, "Could be better", response.Reactions[1].FeedbackText)

	mockDocDB.GetReactionsCollection().AssertExpectations(t)
}

func TestReactionsHandler_GetReactions_DifferentTenant(t *testing.T) {
	mockDocDB := mocks.NewMockDocDBClient()
	mockPlatform := &mocks.MockPlatformClient{}
	otherTenantID := "other-tenant-456"

	mockDocDB.GetReactionsCollection().On("ListByMessage", mock.Anything, &docdb.ListReactionsOptions{
		TenantID:       otherTenantID,
		ConversationID: testutils.TestConversationID,
		MessageID:      testutils.TestMessageID,
	}).Return([]*models.MessageReaction{}, nil)

	handler := handlers.NewReactionsHandler(mockDocDB, mockPlatform)

	router := testutils.SetupTestRouter()
	// Setup a middleware to set tenant context for different tenant
	router.Use(func(c *gin.Context) {
		c.Set("tenant_id", otherTenantID)
		c.Next()
	})
	router.GET("/tenants/:tenantId/conversations/:conversationId/messages/:messageId/reactions", handler.GetReactions)

	w := testutils.PerformRequest(
		router, "GET",
		fmt.Sprintf("/tenants/%s/conversations/%s/messages/%s/reactions", otherTenantID, testutils.TestConversationID, testutils.TestMessageID),
		nil, nil,
	)

	testutils.AssertStatusCode(t, http.StatusOK, w)

	var response handlers.ListReactionsResponse
	testutils.ParseJSONResponse(t, w, &response)
	assert.Empty(t, response.Reactions)

	mockDocDB.GetReactionsCollection().AssertExpectations(t)
}

func TestReactionsHandler_GetReactions_DifferentMessageID(t *testing.T) {
	mockDocDB := mocks.NewMockDocDBClient()
	mockPlatform := &mocks.MockPlatformClient{}
	differentMessageID := "msg-different-789"

	mockDocDB.GetReactionsCollection().On("ListByMessage", mock.Anything, &docdb.ListReactionsOptions{
		TenantID:       testutils.TestTenantID,
		ConversationID: testutils.TestConversationID,
		MessageID:      differentMessageID,
	}).Return([]*models.MessageReaction{}, nil)

	handler := handlers.NewReactionsHandler(mockDocDB, mockPlatform)

	router := testutils.SetupTestRouter()
	router.GET("/tenants/:tenantId/conversations/:conversationId/messages/:messageId/reactions", handler.GetReactions)

	w := testutils.PerformRequest(
		router, "GET",
		fmt.Sprintf("/tenants/%s/conversations/%s/messages/%s/reactions", testutils.TestTenantID, testutils.TestConversationID, differentMessageID),
		nil, nil,
	)

	testutils.AssertStatusCode(t, http.StatusOK, w)

	var response handlers.ListReactionsResponse
	testutils.ParseJSONResponse(t, w, &response)
	assert.Empty(t, response.Reactions)

	mockDocDB.GetReactionsCollection().AssertExpectations(t)
}

// ===========================================================================
// Reaction Validation Tests
// ===========================================================================

func TestReactionsHandler_UpsertReaction_InvalidReactionType_Heart(t *testing.T) {
	mockDocDB := mocks.NewMockDocDBClient()
	mockPlatform := &mocks.MockPlatformClient{}

	handler := handlers.NewReactionsHandler(mockDocDB, mockPlatform)

	router := testutils.SetupTestRouter()
	router.Use(authTestMiddleware())
	router.POST("/tenants/:tenantId/conversations/:conversationId/messages/:messageId/reactions", handler.UpsertReaction)

	reqBody := map[string]interface{}{
		"reaction": "heart", // Invalid - not thumbs_up or thumbs_down
	}

	headers := map[string]string{"Authorization": "Bearer test-token"}
	w := testutils.PerformRequest(
		router, "POST",
		fmt.Sprintf("/tenants/%s/conversations/%s/messages/%s/reactions", testutils.TestTenantID, testutils.TestConversationID, testutils.TestMessageID),
		reqBody, headers,
	)

	testutils.AssertStatusCode(t, http.StatusBadRequest, w)
}

func TestReactionsHandler_UpsertReaction_InvalidReactionType_Empty(t *testing.T) {
	mockDocDB := mocks.NewMockDocDBClient()
	mockPlatform := &mocks.MockPlatformClient{}

	handler := handlers.NewReactionsHandler(mockDocDB, mockPlatform)

	router := testutils.SetupTestRouter()
	router.Use(authTestMiddleware())
	router.POST("/tenants/:tenantId/conversations/:conversationId/messages/:messageId/reactions", handler.UpsertReaction)

	reqBody := map[string]interface{}{
		"reaction": "", // Empty string
	}

	headers := map[string]string{"Authorization": "Bearer test-token"}
	w := testutils.PerformRequest(
		router, "POST",
		fmt.Sprintf("/tenants/%s/conversations/%s/messages/%s/reactions", testutils.TestTenantID, testutils.TestConversationID, testutils.TestMessageID),
		reqBody, headers,
	)

	testutils.AssertStatusCode(t, http.StatusBadRequest, w)
}

func TestReactionsHandler_UpsertReaction_InvalidReactionType_CaseSensitive(t *testing.T) {
	mockDocDB := mocks.NewMockDocDBClient()
	mockPlatform := &mocks.MockPlatformClient{}

	handler := handlers.NewReactionsHandler(mockDocDB, mockPlatform)

	router := testutils.SetupTestRouter()
	router.Use(authTestMiddleware())
	router.POST("/tenants/:tenantId/conversations/:conversationId/messages/:messageId/reactions", handler.UpsertReaction)

	reqBody := map[string]interface{}{
		"reaction": "THUMBS_UP", // Wrong case
	}

	headers := map[string]string{"Authorization": "Bearer test-token"}
	w := testutils.PerformRequest(
		router, "POST",
		fmt.Sprintf("/tenants/%s/conversations/%s/messages/%s/reactions", testutils.TestTenantID, testutils.TestConversationID, testutils.TestMessageID),
		reqBody, headers,
	)

	testutils.AssertStatusCode(t, http.StatusBadRequest, w)
}
