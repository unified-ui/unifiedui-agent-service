// Package handlers_test verifies the reactions-to-platform feedback proxy.
package handlers_test

import (
	"fmt"
	"net/http"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"github.com/unifiedui/agent-service/internal/api/handlers"
	"github.com/unifiedui/agent-service/internal/domain/models"
	"github.com/unifiedui/agent-service/internal/services/platform"
	"github.com/unifiedui/agent-service/tests/mocks"
	"github.com/unifiedui/agent-service/tests/testutils"
)

func TestReactionsHandler_UpsertReaction_ProxiesToPlatform(t *testing.T) {
	mockDocDB := mocks.NewMockDocDBClient()
	mockPlatform := &mocks.MockPlatformClient{}

	testMessage := testutils.NewTestAssistantMessage()

	mockPlatform.On("GetMe", mock.Anything, mock.Anything).Return(&platform.UserInfo{ID: testutils.TestUserID}, nil)
	mockDocDB.GetMessagesCollection().On("Get", mock.Anything, testutils.TestMessageID).Return(testMessage, nil)
	mockDocDB.GetReactionsCollection().On("Upsert", mock.Anything, mock.Anything).Return(nil)

	var wg sync.WaitGroup
	wg.Add(1)
	mockPlatform.On(
		"UpsertMessageFeedback",
		mock.Anything,
		testutils.TestTenantID,
		testutils.TestConversationID,
		testutils.TestMessageID,
		"test-token",
		mock.MatchedBy(func(p platform.UpsertMessageFeedbackRequest) bool {
			return p.Rating == "THUMBS_UP" && p.Comment == "Great"
		}),
	).Return(nil).Run(func(_ mock.Arguments) { wg.Done() })

	handler := handlers.NewReactionsHandler(mockDocDB, mockPlatform)

	router := testutils.SetupTestRouter()
	router.Use(authTestMiddleware())
	router.POST("/tenants/:tenantId/conversations/:conversationId/messages/:messageId/reactions", handler.UpsertReaction)

	reqBody := handlers.UpsertReactionRequest{
		Reaction:     models.ReactionThumbsUp,
		FeedbackText: "Great",
	}
	headers := map[string]string{"Authorization": "Bearer test-token"}

	w := testutils.PerformRequest(
		router, "POST",
		fmt.Sprintf("/tenants/%s/conversations/%s/messages/%s/reactions", testutils.TestTenantID, testutils.TestConversationID, testutils.TestMessageID),
		reqBody, headers,
	)

	testutils.AssertStatusCode(t, http.StatusOK, w)

	done := make(chan struct{})
	go func() { wg.Wait(); close(done) }()
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("UpsertMessageFeedback was not called within timeout")
	}

	mockPlatform.AssertCalled(t, "UpsertMessageFeedback", mock.Anything, testutils.TestTenantID, testutils.TestConversationID, testutils.TestMessageID, "test-token", mock.AnythingOfType("platform.UpsertMessageFeedbackRequest"))
}

func TestReactionsHandler_DeleteReaction_ProxiesToPlatform(t *testing.T) {
	mockDocDB := mocks.NewMockDocDBClient()
	mockPlatform := &mocks.MockPlatformClient{}

	mockPlatform.On("GetMe", mock.Anything, mock.Anything).Return(&platform.UserInfo{ID: testutils.TestUserID}, nil)
	mockDocDB.GetReactionsCollection().On("Delete", mock.Anything, mock.Anything).Return(nil)

	var wg sync.WaitGroup
	wg.Add(1)
	mockPlatform.On(
		"DeleteMessageFeedback",
		mock.Anything,
		testutils.TestTenantID,
		testutils.TestConversationID,
		testutils.TestMessageID,
		"test-token",
	).Return(nil).Run(func(_ mock.Arguments) { wg.Done() })

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

	assert.Equal(t, http.StatusNoContent, w.Code)

	done := make(chan struct{})
	go func() { wg.Wait(); close(done) }()
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("DeleteMessageFeedback was not called within timeout")
	}
}
