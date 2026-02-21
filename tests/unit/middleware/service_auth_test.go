package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/unifiedui/agent-service/internal/api/middleware"
	"github.com/unifiedui/agent-service/internal/config"
	"github.com/unifiedui/agent-service/tests/mocks"
)

func TestServiceKeyMiddleware_MissingHeader(t *testing.T) {
	vault := mocks.NewMockVaultClient()
	cfg := config.AppVaultConfig{PlatformServiceKey: "service-key-name"}
	skm := middleware.NewServiceKeyMiddleware(vault, cfg)

	router := gin.New()
	router.Use(skm.AuthenticateServiceKey())
	router.GET("/test", func(c *gin.Context) { c.Status(http.StatusOK) })

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	router.ServeHTTP(w, req)

	require.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestServiceKeyMiddleware_EmptyKeyName(t *testing.T) {
	vault := mocks.NewMockVaultClient()
	cfg := config.AppVaultConfig{PlatformServiceKey: ""}
	skm := middleware.NewServiceKeyMiddleware(vault, cfg)

	router := gin.New()
	router.Use(skm.AuthenticateServiceKey())
	router.GET("/test", func(c *gin.Context) { c.Status(http.StatusOK) })

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("X-Service-Key", "some-key")
	router.ServeHTTP(w, req)

	require.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestServiceKeyMiddleware_InvalidKey(t *testing.T) {
	vault := mocks.NewMockVaultClient()
	vault.On("GetSecret", mock.Anything, "dotenv://my-key", false).Return("correct-key", nil)
	cfg := config.AppVaultConfig{PlatformServiceKey: "my-key"}
	skm := middleware.NewServiceKeyMiddleware(vault, cfg)

	router := gin.New()
	router.Use(skm.AuthenticateServiceKey())
	router.GET("/test", func(c *gin.Context) { c.Status(http.StatusOK) })

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("X-Service-Key", "wrong-key")
	router.ServeHTTP(w, req)

	require.Equal(t, http.StatusForbidden, w.Code)
}

func TestServiceKeyMiddleware_ValidKey(t *testing.T) {
	vault := mocks.NewMockVaultClient()
	vault.On("GetSecret", mock.Anything, "dotenv://my-key", false).Return("correct-key", nil)
	cfg := config.AppVaultConfig{PlatformServiceKey: "my-key"}
	skm := middleware.NewServiceKeyMiddleware(vault, cfg)

	router := gin.New()
	router.Use(skm.AuthenticateServiceKey())
	router.GET("/test", func(c *gin.Context) { c.Status(http.StatusOK) })

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("X-Service-Key", "correct-key")
	router.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)
}
