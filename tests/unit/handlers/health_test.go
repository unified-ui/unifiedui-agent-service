// Package handlers_test provides unit tests for the API handlers.
package handlers_test

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"github.com/unifiedui/agent-service/internal/api/handlers"
	"github.com/unifiedui/agent-service/tests/mocks"
	"github.com/unifiedui/agent-service/tests/testutils"
)

func TestHealthHandler_Health_AllHealthy(t *testing.T) {
	// Setup
	mockCache := mocks.NewMockCacheClient()
	mockDocDB := mocks.NewMockDocDBClient()

	mockCache.On("Ping", mock.Anything).Return(nil)
	mockDocDB.On("Ping", mock.Anything).Return(nil)

	handler := handlers.NewHealthHandler(mockCache, mockDocDB)

	router := testutils.SetupTestRouter()
	router.GET("/health", handler.Health)

	// Execute
	w := testutils.PerformRequest(router, "GET", "/health", nil, nil)

	// Assert
	testutils.AssertStatusCode(t, http.StatusOK, w)

	var response handlers.HealthResponse
	testutils.ParseJSONResponse(t, w, &response)

	assert.Equal(t, "healthy", response.Status)
	assert.Equal(t, "healthy", response.Components["cache"])
	assert.Equal(t, "healthy", response.Components["docdb"])

	mockCache.AssertExpectations(t)
	mockDocDB.AssertExpectations(t)
}

func TestHealthHandler_Health_CacheUnhealthy(t *testing.T) {
	// Setup
	mockCache := mocks.NewMockCacheClient()
	mockDocDB := mocks.NewMockDocDBClient()

	mockCache.On("Ping", mock.Anything).Return(assert.AnError)
	mockDocDB.On("Ping", mock.Anything).Return(nil)

	handler := handlers.NewHealthHandler(mockCache, mockDocDB)

	router := testutils.SetupTestRouter()
	router.GET("/health", handler.Health)

	// Execute
	w := testutils.PerformRequest(router, "GET", "/health", nil, nil)

	// Assert
	testutils.AssertStatusCode(t, http.StatusServiceUnavailable, w)

	var response handlers.HealthResponse
	testutils.ParseJSONResponse(t, w, &response)

	assert.Equal(t, "unhealthy", response.Status)
	assert.Equal(t, "unhealthy", response.Components["cache"])
	assert.Equal(t, "healthy", response.Components["docdb"])
}

func TestHealthHandler_Ready_AllReady(t *testing.T) {
	// Setup
	mockCache := mocks.NewMockCacheClient()
	mockDocDB := mocks.NewMockDocDBClient()

	mockCache.On("Ping", mock.Anything).Return(nil)
	mockDocDB.On("Ping", mock.Anything).Return(nil)

	handler := handlers.NewHealthHandler(mockCache, mockDocDB)

	router := testutils.SetupTestRouter()
	router.GET("/ready", handler.Ready)

	// Execute
	w := testutils.PerformRequest(router, "GET", "/ready", nil, nil)

	// Assert
	testutils.AssertStatusCode(t, http.StatusOK, w)
}

func TestHealthHandler_Ready_NotReady(t *testing.T) {
	// Setup
	mockCache := mocks.NewMockCacheClient()
	mockDocDB := mocks.NewMockDocDBClient()

	mockCache.On("Ping", mock.Anything).Return(assert.AnError)

	handler := handlers.NewHealthHandler(mockCache, mockDocDB)

	router := testutils.SetupTestRouter()
	router.GET("/ready", handler.Ready)

	// Execute
	w := testutils.PerformRequest(router, "GET", "/ready", nil, nil)

	// Assert
	testutils.AssertStatusCode(t, http.StatusServiceUnavailable, w)
}

func TestHealthHandler_Live(t *testing.T) {
	// Setup
	mockCache := mocks.NewMockCacheClient()
	mockDocDB := mocks.NewMockDocDBClient()

	handler := handlers.NewHealthHandler(mockCache, mockDocDB)

	router := testutils.SetupTestRouter()
	router.GET("/live", handler.Live)

	// Execute
	w := testutils.PerformRequest(router, "GET", "/live", nil, nil)

	// Assert
	testutils.AssertStatusCode(t, http.StatusOK, w)
}
