package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/require"

	"github.com/unifiedui/agent-service/internal/api/middleware"
)

func TestLoggingMiddleware_Logger(t *testing.T) {
	router := gin.New()
	lm := middleware.NewLoggingMiddleware()
	router.Use(lm.Logger())
	router.GET("/test", func(c *gin.Context) { c.Status(http.StatusOK) })

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/test", http.NoBody)
	router.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)
}

func TestLoggingMiddleware_Logger_4xx(t *testing.T) {
	router := gin.New()
	lm := middleware.NewLoggingMiddleware()
	router.Use(lm.Logger())
	router.GET("/test", func(c *gin.Context) { c.Status(http.StatusBadRequest) })

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/test", http.NoBody)
	router.ServeHTTP(w, req)

	require.Equal(t, http.StatusBadRequest, w.Code)
}

func TestLoggingMiddleware_Logger_5xx(t *testing.T) {
	router := gin.New()
	lm := middleware.NewLoggingMiddleware()
	router.Use(lm.Logger())
	router.GET("/test", func(c *gin.Context) { c.Status(http.StatusInternalServerError) })

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/test", http.NoBody)
	router.ServeHTTP(w, req)

	require.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestNewLoggingMiddlewareWithLogger(t *testing.T) {
	logger := zerolog.Nop()
	lm := middleware.NewLoggingMiddlewareWithLogger(logger)
	require.NotNil(t, lm)
}

func TestRequestLogger_SetsRequestID(t *testing.T) {
	router := gin.New()
	lm := middleware.NewLoggingMiddleware()
	router.Use(lm.RequestLogger())

	var requestID string
	router.GET("/test", func(c *gin.Context) {
		requestID = middleware.GetRequestID(c)
		c.Status(http.StatusOK)
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/test", http.NoBody)
	router.ServeHTTP(w, req)

	require.NotEmpty(t, requestID)
	require.NotEmpty(t, w.Header().Get("X-Request-ID"))
}

func TestRequestLogger_UsesProvidedRequestID(t *testing.T) {
	router := gin.New()
	lm := middleware.NewLoggingMiddleware()
	router.Use(lm.RequestLogger())

	var requestID string
	router.GET("/test", func(c *gin.Context) {
		requestID = middleware.GetRequestID(c)
		c.Status(http.StatusOK)
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/test", http.NoBody)
	req.Header.Set("X-Request-ID", "custom-id-123")
	router.ServeHTTP(w, req)

	require.Equal(t, "custom-id-123", requestID)
}

func TestGetRequestLogger_Default(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/", http.NoBody)
	logger := middleware.GetRequestLogger(c)
	require.NotNil(t, logger)
}

func TestGetRequestLogger_WithLoggerSet(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/", http.NoBody)

	// Set a custom logger in context
	customLogger := zerolog.Nop()
	c.Set("logger", customLogger)

	logger := middleware.GetRequestLogger(c)
	require.NotNil(t, logger)
}

func TestGetRequestID_NotSet(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/", http.NoBody)
	require.Equal(t, "", middleware.GetRequestID(c))
}

func TestGetRequestID_WhenSet(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/", http.NoBody)
	c.Set("request_id", "test-request-123")
	require.Equal(t, "test-request-123", middleware.GetRequestID(c))
}
