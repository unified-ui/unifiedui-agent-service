package middleware

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"

	"github.com/unifiedui/agent-service/internal/api/middleware"
	domainerrors "github.com/unifiedui/agent-service/internal/domain/errors"
)

func TestErrorMiddleware_Recovery(t *testing.T) {
	router := gin.New()
	em := middleware.NewErrorMiddleware()
	router.Use(em.Recovery())
	router.GET("/panic", func(c *gin.Context) {
		panic("test panic")
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/panic", http.NoBody)
	router.ServeHTTP(w, req)

	require.Equal(t, http.StatusInternalServerError, w.Code)
	require.Contains(t, w.Body.String(), "INTERNAL_ERROR")
}

func TestHandleError_NilError(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/", http.NoBody)

	middleware.HandleError(c, nil)
	require.Equal(t, http.StatusOK, w.Code)
}

func TestHandleError_DomainError(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/", http.NoBody)

	err := domainerrors.NewNotFoundError("Trace", "t-1")
	middleware.HandleError(c, err)

	require.Equal(t, http.StatusNotFound, w.Code)
	require.Contains(t, w.Body.String(), "NOT_FOUND")
}

func TestHandleError_WrappedDomainError(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/", http.NoBody)

	inner := domainerrors.NewValidationError("bad input", "field x")
	wrapped := fmt.Errorf("handler: %w", inner)
	middleware.HandleError(c, wrapped)

	require.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandleError_PlainError(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/", http.NoBody)

	middleware.HandleError(c, fmt.Errorf("something broke"))

	require.Equal(t, http.StatusInternalServerError, w.Code)
	require.Contains(t, w.Body.String(), "INTERNAL_ERROR")
}

func TestNotFound(t *testing.T) {
	router := gin.New()
	router.NoRoute(middleware.NotFound())

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/nonexistent", http.NoBody)
	router.ServeHTTP(w, req)

	require.Equal(t, http.StatusNotFound, w.Code)
	require.Contains(t, w.Body.String(), "NOT_FOUND")
}

func TestMethodNotAllowed(t *testing.T) {
	router := gin.New()
	router.HandleMethodNotAllowed = true
	router.NoMethod(middleware.MethodNotAllowed())
	router.GET("/test", func(c *gin.Context) { c.Status(http.StatusOK) })

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/test", http.NoBody)
	router.ServeHTTP(w, req)

	require.Equal(t, http.StatusMethodNotAllowed, w.Code)
}
