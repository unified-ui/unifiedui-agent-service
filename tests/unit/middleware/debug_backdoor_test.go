package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"

	"github.com/unifiedui/agent-service/internal/api/middleware"
	"github.com/unifiedui/agent-service/internal/config"
)

const validBackdoorSecret = "this-secret-is-at-least-32-chars!!"

func backdoorCfg(enabled bool) config.DebugBackdoorConfig {
	return config.DebugBackdoorConfig{Enabled: enabled, Secret: validBackdoorSecret}
}

func TestDebugBackdoor_DisabledIgnoresHeaders(t *testing.T) {
	router := gin.New()
	am := middleware.NewAuthMiddleware(nil)
	router.Use(am.Authenticate())
	router.GET("/x", func(c *gin.Context) { c.Status(http.StatusOK) })

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/x", http.NoBody)
	req.Header.Set(middleware.DebugHeaderSecret, validBackdoorSecret)
	req.Header.Set(middleware.DebugHeaderUserID, "u1")
	req.Header.Set(middleware.DebugHeaderUserUPN, "u1@example.com")
	router.ServeHTTP(w, req)

	require.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestDebugBackdoor_WrongSecret(t *testing.T) {
	router := gin.New()
	am := middleware.NewAuthMiddleware(nil)
	am.SetDebugBackdoor(backdoorCfg(true))
	router.Use(am.Authenticate())
	router.GET("/x", func(c *gin.Context) { c.Status(http.StatusOK) })

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/x", http.NoBody)
	req.Header.Set(middleware.DebugHeaderSecret, "wrong")
	req.Header.Set(middleware.DebugHeaderUserID, "u1")
	req.Header.Set(middleware.DebugHeaderUserUPN, "u1@example.com")
	router.ServeHTTP(w, req)

	require.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestDebugBackdoor_MissingUserHeaders(t *testing.T) {
	router := gin.New()
	am := middleware.NewAuthMiddleware(nil)
	am.SetDebugBackdoor(backdoorCfg(true))
	router.Use(am.Authenticate())
	router.GET("/x", func(c *gin.Context) { c.Status(http.StatusOK) })

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/x", http.NoBody)
	req.Header.Set(middleware.DebugHeaderSecret, validBackdoorSecret)
	router.ServeHTTP(w, req)

	require.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestDebugBackdoor_ValidSynthesisesUser(t *testing.T) {
	router := gin.New()
	am := middleware.NewAuthMiddleware(nil)
	am.SetDebugBackdoor(backdoorCfg(true))
	router.Use(am.Authenticate())

	var capturedID, capturedToken string
	router.GET("/x", func(c *gin.Context) {
		u := middleware.GetAuthUser(c)
		require.NotNil(t, u)
		capturedID = u.ID
		capturedToken = middleware.GetToken(c)
		require.True(t, middleware.IsBackdoorRequest(c))
		c.Status(http.StatusOK)
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/x", http.NoBody)
	req.Header.Set(middleware.DebugHeaderSecret, validBackdoorSecret)
	req.Header.Set(middleware.DebugHeaderUserID, "back-user-1")
	req.Header.Set(middleware.DebugHeaderUserUPN, "back@example.com")
	req.Header.Set(middleware.DebugHeaderTenantID, "t-1")
	req.Header.Set(middleware.DebugHeaderGroups, "g1, g2 ,g3")
	router.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)
	require.Equal(t, "back-user-1", capturedID)
	require.NotEmpty(t, capturedToken)
}

func TestDebugBackdoor_FlexibleAuthAccepts(t *testing.T) {
	router := gin.New()
	am := middleware.NewAuthMiddleware(nil)
	am.SetDebugBackdoor(backdoorCfg(true))
	router.Use(am.AuthenticateFlexible())
	router.GET("/x", func(c *gin.Context) {
		require.True(t, middleware.IsBackdoorRequest(c))
		c.Status(http.StatusOK)
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/x", http.NoBody)
	req.Header.Set(middleware.DebugHeaderSecret, validBackdoorSecret)
	req.Header.Set(middleware.DebugHeaderUserID, "u1")
	req.Header.Set(middleware.DebugHeaderUserUPN, "u1@example.com")
	router.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)
}

func TestDebugBackdoorConfig_Validate(t *testing.T) {
	require.NoError(t, config.DebugBackdoorConfig{Enabled: false}.Validate("self-hosted"))
	require.Error(t, config.DebugBackdoorConfig{Enabled: true, Secret: "short"}.Validate("self-hosted"))
	require.Error(t, config.DebugBackdoorConfig{Enabled: true, Secret: validBackdoorSecret}.Validate("production"))
	require.NoError(t, config.DebugBackdoorConfig{Enabled: true, Secret: validBackdoorSecret}.Validate("self-hosted"))
}
