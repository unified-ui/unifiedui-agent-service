package handlers_test

import (
	"fmt"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"github.com/unifiedui/agent-service/internal/api/handlers"
	"github.com/unifiedui/agent-service/internal/core/docdb"
	"github.com/unifiedui/agent-service/internal/domain/models"
	"github.com/unifiedui/agent-service/tests/mocks"
	"github.com/unifiedui/agent-service/tests/testutils"
)

func TestMessagesHandler_SearchMessages_Success(t *testing.T) {
	mockDocDB := mocks.NewMockDocDBClient()
	testMessage := testutils.NewTestAssistantMessage()

	mockDocDB.GetMessagesCollection().On("Search", mock.Anything, &docdb.SearchMessagesOptions{
		TenantID: testutils.TestTenantID,
		Query:    "hello",
		Limit:    20,
		Skip:     0,
	}).Return([]*models.Message{testMessage}, nil)

	handler := createTestMessagesHandler(mockDocDB)

	router := testutils.SetupTestRouter()
	router.GET("/tenants/:tenantId/conversation/messages/search", handler.SearchMessages)

	w := testutils.PerformRequest(
		router, "GET",
		fmt.Sprintf("/tenants/%s/conversation/messages/search?query=hello", testutils.TestTenantID),
		nil, nil,
	)

	testutils.AssertStatusCode(t, http.StatusOK, w)

	var response handlers.SearchMessagesResponse
	testutils.ParseJSONResponse(t, w, &response)

	assert.Len(t, response.Messages, 1)
	assert.Equal(t, testMessage.ID, response.Messages[0].ID)

	mockDocDB.GetMessagesCollection().AssertExpectations(t)
}

func TestMessagesHandler_SearchMessages_WithLimitAndSkip(t *testing.T) {
	mockDocDB := mocks.NewMockDocDBClient()

	mockDocDB.GetMessagesCollection().On("Search", mock.Anything, &docdb.SearchMessagesOptions{
		TenantID: testutils.TestTenantID,
		Query:    "test",
		Limit:    10,
		Skip:     5,
	}).Return([]*models.Message{}, nil)

	handler := createTestMessagesHandler(mockDocDB)

	router := testutils.SetupTestRouter()
	router.GET("/tenants/:tenantId/conversation/messages/search", handler.SearchMessages)

	w := testutils.PerformRequest(
		router, "GET",
		fmt.Sprintf("/tenants/%s/conversation/messages/search?query=test&limit=10&skip=5", testutils.TestTenantID),
		nil, nil,
	)

	testutils.AssertStatusCode(t, http.StatusOK, w)

	var response handlers.SearchMessagesResponse
	testutils.ParseJSONResponse(t, w, &response)

	assert.Empty(t, response.Messages)

	mockDocDB.GetMessagesCollection().AssertExpectations(t)
}

func TestMessagesHandler_SearchMessages_MissingQuery(t *testing.T) {
	mockDocDB := mocks.NewMockDocDBClient()
	handler := createTestMessagesHandler(mockDocDB)

	router := testutils.SetupTestRouter()
	router.GET("/tenants/:tenantId/conversation/messages/search", handler.SearchMessages)

	w := testutils.PerformRequest(
		router, "GET",
		fmt.Sprintf("/tenants/%s/conversation/messages/search", testutils.TestTenantID),
		nil, nil,
	)

	testutils.AssertStatusCode(t, http.StatusBadRequest, w)
}

func TestMessagesHandler_SearchMessages_InternalError(t *testing.T) {
	mockDocDB := mocks.NewMockDocDBClient()

	mockDocDB.GetMessagesCollection().On("Search", mock.Anything, mock.Anything).
		Return(([]*models.Message)(nil), fmt.Errorf("database error"))

	handler := createTestMessagesHandler(mockDocDB)

	router := testutils.SetupTestRouter()
	router.GET("/tenants/:tenantId/conversation/messages/search", handler.SearchMessages)

	w := testutils.PerformRequest(
		router, "GET",
		fmt.Sprintf("/tenants/%s/conversation/messages/search?query=hello", testutils.TestTenantID),
		nil, nil,
	)

	testutils.AssertStatusCode(t, http.StatusInternalServerError, w)

	mockDocDB.GetMessagesCollection().AssertExpectations(t)
}

func TestMessagesHandler_SearchMessages_EmptyResults(t *testing.T) {
	mockDocDB := mocks.NewMockDocDBClient()

	mockDocDB.GetMessagesCollection().On("Search", mock.Anything, &docdb.SearchMessagesOptions{
		TenantID: testutils.TestTenantID,
		Query:    "nonexistent",
		Limit:    20,
		Skip:     0,
	}).Return([]*models.Message{}, nil)

	handler := createTestMessagesHandler(mockDocDB)

	router := testutils.SetupTestRouter()
	router.GET("/tenants/:tenantId/conversation/messages/search", handler.SearchMessages)

	w := testutils.PerformRequest(
		router, "GET",
		fmt.Sprintf("/tenants/%s/conversation/messages/search?query=nonexistent", testutils.TestTenantID),
		nil, nil,
	)

	testutils.AssertStatusCode(t, http.StatusOK, w)

	var response handlers.SearchMessagesResponse
	testutils.ParseJSONResponse(t, w, &response)

	assert.Empty(t, response.Messages)

	mockDocDB.GetMessagesCollection().AssertExpectations(t)
}
