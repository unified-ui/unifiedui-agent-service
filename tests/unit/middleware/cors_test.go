package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"

	"github.com/unifiedui/agent-service/internal/api/middleware"
)

func TestDefaultCORSConfig(t *testing.T) {
	cfg := middleware.DefaultCORSConfig()
	require.NotEmpty(t, cfg.AllowOrigins)
	require.NotEmpty(t, cfg.AllowMethods)
	require.NotEmpty(t, cfg.AllowHeaders)
	require.True(t, cfg.AllowCredentials)
	require.Equal(t, 86400, cfg.MaxAge)
}

func TestNewCORSMiddleware_AllowedOrigin(t *testing.T) {
	cfg := middleware.DefaultCORSConfig()
	router := gin.New()
	router.Use(middleware.NewCORSMiddleware(cfg))
	router.GET("/test", func(c *gin.Context) { c.Status(http.StatusOK) })

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/test", http.NoBody)
	req.Header.Set("Origin", "http://localhost:5173")
	router.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)
	require.Equal(t, "http://localhost:5173", w.Header().Get("Access-Control-Allow-Origin"))
	require.Equal(t, "true", w.Header().Get("Access-Control-Allow-Credentials"))
}

func TestNewCORSMiddleware_DisallowedOrigin(t *testing.T) {
	cfg := middleware.DefaultCORSConfig()
	router := gin.New()
	router.Use(middleware.NewCORSMiddleware(cfg))
	router.GET("/test", func(c *gin.Context) { c.Status(http.StatusOK) })

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/test", http.NoBody)
	req.Header.Set("Origin", "http://evil.com")
	router.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)
	require.Empty(t, w.Header().Get("Access-Control-Allow-Origin"))
}

func TestNewCORSMiddleware_Preflight(t *testing.T) {
	cfg := middleware.DefaultCORSConfig()
	router := gin.New()
	router.Use(middleware.NewCORSMiddleware(cfg))
	router.GET("/test", func(c *gin.Context) { c.Status(http.StatusOK) })

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodOptions, "/test", http.NoBody)
	req.Header.Set("Origin", "http://localhost:5173")
	router.ServeHTTP(w, req)

	require.Equal(t, http.StatusNoContent, w.Code)
	require.Equal(t, "http://localhost:5173", w.Header().Get("Access-Control-Allow-Origin"))
}

func TestNewCORSMiddleware_WildcardOrigin(t *testing.T) {
	cfg := middleware.CORSConfig{
		AllowOrigins: []string{"*"},
		AllowMethods: []string{http.MethodGet},
	}
	router := gin.New()
	router.Use(middleware.NewCORSMiddleware(cfg))
	router.GET("/test", func(c *gin.Context) { c.Status(http.StatusOK) })

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/test", http.NoBody)
	req.Header.Set("Origin", "http://anything.com")
	router.ServeHTTP(w, req)

	require.Equal(t, "http://anything.com", w.Header().Get("Access-Control-Allow-Origin"))
}

func TestSetupCORSRoutes(t *testing.T) {
	cfg := middleware.DefaultCORSConfig()
	router := gin.New()
	middleware.SetupCORSRoutes(router, cfg)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodOptions, "/any/path", http.NoBody)
	req.Header.Set("Origin", "http://localhost:5173")
	router.ServeHTTP(w, req)

	require.Equal(t, http.StatusNoContent, w.Code)
}
