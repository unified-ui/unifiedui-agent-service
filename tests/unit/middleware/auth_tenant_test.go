package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"

	"github.com/unifiedui/agent-service/internal/api/middleware"
)

func init() {
	gin.SetMode(gin.TestMode)
}

func TestTenantMiddleware_ExtractTenant(t *testing.T) {
	router := gin.New()
	tm := middleware.NewTenantMiddleware()
	router.Use(tm.ExtractTenant())

	var capturedTenantID string
	router.GET("/tenants/:tenantId/test", func(c *gin.Context) {
		capturedTenantID = middleware.GetTenantID(c)
		c.Status(http.StatusOK)
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/tenants/t-123/test", http.NoBody)
	router.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)
	require.Equal(t, "t-123", capturedTenantID)
}

func TestGetTenantID_FromContextFirstPriority(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/", http.NoBody)
	// Set tenant_id directly in context (first branch)
	c.Set("tenant_id", "context-tenant-123")
	// Also set path param (should be ignored since context has it)
	c.Params = []gin.Param{{Key: "tenantId", Value: "path-tenant-456"}}

	tenantID := middleware.GetTenantID(c)
	require.Equal(t, "context-tenant-123", tenantID)
}

func TestGetTenantID_FallbackToPathParam(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/", http.NoBody)
	// Don't set tenant_id in context, only in path params
	c.Params = []gin.Param{{Key: "tenantId", Value: "path-tenant-456"}}

	tenantID := middleware.GetTenantID(c)
	require.Equal(t, "path-tenant-456", tenantID)
}

func TestGetTenantID_Empty(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/", http.NoBody)
	// Neither context nor path param set

	tenantID := middleware.GetTenantID(c)
	require.Equal(t, "", tenantID)
}

func TestGetTenantContext(t *testing.T) {
	router := gin.New()
	var tc *middleware.TenantContext
	router.GET("/tenants/:tenantId/conversations/:conversationId", func(c *gin.Context) {
		c.Set("user_id", "user-42")
		tc = middleware.GetTenantContext(c)
		c.Status(http.StatusOK)
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/tenants/t-1/conversations/c-2", http.NoBody)
	router.ServeHTTP(w, req)

	require.Equal(t, "t-1", tc.TenantID)
	require.Equal(t, "c-2", tc.ConversationID)
	require.Equal(t, "user-42", tc.UserID)
}

func TestAuthMiddleware_Authenticate_Success(t *testing.T) {
	router := gin.New()
	am := middleware.NewAuthMiddleware("http://platform")
	router.Use(am.Authenticate())

	var token string
	router.GET("/test", func(c *gin.Context) {
		token = middleware.GetToken(c)
		c.Status(http.StatusOK)
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/test", http.NoBody)
	req.Header.Set("Authorization", "Bearer my-token-123")
	router.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)
	require.Equal(t, "my-token-123", token)
}

func TestAuthMiddleware_Authenticate_MissingHeader(t *testing.T) {
	router := gin.New()
	am := middleware.NewAuthMiddleware("http://platform")
	router.Use(am.Authenticate())
	router.GET("/test", func(c *gin.Context) { c.Status(http.StatusOK) })

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/test", http.NoBody)
	router.ServeHTTP(w, req)

	require.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestAuthMiddleware_Authenticate_InvalidFormat(t *testing.T) {
	router := gin.New()
	am := middleware.NewAuthMiddleware("http://platform")
	router.Use(am.Authenticate())
	router.GET("/test", func(c *gin.Context) { c.Status(http.StatusOK) })

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/test", http.NoBody)
	req.Header.Set("Authorization", "Basic abc")
	router.ServeHTTP(w, req)

	require.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestAuthMiddleware_Authenticate_EmptyToken(t *testing.T) {
	router := gin.New()
	am := middleware.NewAuthMiddleware("http://platform")
	router.Use(am.Authenticate())
	router.GET("/test", func(c *gin.Context) { c.Status(http.StatusOK) })

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/test", http.NoBody)
	req.Header.Set("Authorization", "Bearer ")
	router.ServeHTTP(w, req)

	require.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestAuthMiddleware_AuthenticateFlexible_BearerToken(t *testing.T) {
	router := gin.New()
	am := middleware.NewAuthMiddleware("http://platform")
	router.Use(am.AuthenticateFlexible())
	var token string
	router.GET("/test", func(c *gin.Context) {
		token = middleware.GetToken(c)
		c.Status(http.StatusOK)
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/test", http.NoBody)
	req.Header.Set("Authorization", "Bearer flex-token")
	router.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)
	require.Equal(t, "flex-token", token)
}

func TestAuthMiddleware_AuthenticateFlexible_APIKey(t *testing.T) {
	router := gin.New()
	am := middleware.NewAuthMiddleware("http://platform")
	router.Use(am.AuthenticateFlexible())
	var apiKey string
	router.GET("/test", func(c *gin.Context) {
		apiKey = middleware.GetWorkflowAPIKey(c)
		c.Status(http.StatusOK)
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/test", http.NoBody)
	req.Header.Set("X-Unified-UI-Workflow-API-Key", "my-api-key")
	router.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)
	require.Equal(t, "my-api-key", apiKey)
}

func TestAuthMiddleware_AuthenticateFlexible_NeitherPresent(t *testing.T) {
	router := gin.New()
	am := middleware.NewAuthMiddleware("http://platform")
	router.Use(am.AuthenticateFlexible())
	router.GET("/test", func(c *gin.Context) { c.Status(http.StatusOK) })

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/test", http.NoBody)
	router.ServeHTTP(w, req)

	require.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestAuthMiddleware_AuthenticateFlexible_InvalidAuthHeader_NoAPIKey(t *testing.T) {
	// Tests the "invalid authentication credentials" path when auth header is present but invalid
	// and no API key is provided
	router := gin.New()
	am := middleware.NewAuthMiddleware("http://platform")
	router.Use(am.AuthenticateFlexible())
	router.GET("/test", func(c *gin.Context) { c.Status(http.StatusOK) })

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/test", http.NoBody)
	req.Header.Set("Authorization", "Basic invalid-token") // Not Bearer, and no API key
	router.ServeHTTP(w, req)

	require.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestAuthMiddleware_AuthenticateFlexible_EmptyBearerToken_NoAPIKey(t *testing.T) {
	// Tests when Bearer token is empty and no API key
	router := gin.New()
	am := middleware.NewAuthMiddleware("http://platform")
	router.Use(am.AuthenticateFlexible())
	router.GET("/test", func(c *gin.Context) { c.Status(http.StatusOK) })

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/test", http.NoBody)
	req.Header.Set("Authorization", "Bearer ") // Empty token
	router.ServeHTTP(w, req)

	require.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestAuthMiddleware_AuthenticateFlexible_SinglePartAuth_NoAPIKey(t *testing.T) {
	// Tests when Authorization header has only one part (no space)
	router := gin.New()
	am := middleware.NewAuthMiddleware("http://platform")
	router.Use(am.AuthenticateFlexible())
	router.GET("/test", func(c *gin.Context) { c.Status(http.StatusOK) })

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/test", http.NoBody)
	req.Header.Set("Authorization", "InvalidNoSpace")
	router.ServeHTTP(w, req)

	require.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestAuthMiddleware_AuthenticateFlexible_InvalidAuthHeader_WithAPIKey(t *testing.T) {
	// Tests fallback to API key when auth header is invalid
	router := gin.New()
	am := middleware.NewAuthMiddleware("http://platform")
	router.Use(am.AuthenticateFlexible())
	var apiKey string
	router.GET("/test", func(c *gin.Context) {
		apiKey = middleware.GetWorkflowAPIKey(c)
		c.Status(http.StatusOK)
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/test", http.NoBody)
	req.Header.Set("Authorization", "Basic invalid")                // Invalid auth header
	req.Header.Set("X-Unified-UI-Workflow-API-Key", "fallback-key") // But has API key
	router.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)
	require.Equal(t, "fallback-key", apiKey)
}

func TestAuthMiddleware_AuthenticateWorkflowAPIKey_Success(t *testing.T) {
	router := gin.New()
	am := middleware.NewAuthMiddleware("http://platform")
	router.Use(am.AuthenticateWorkflowAPIKey())
	var apiKey string
	router.GET("/test", func(c *gin.Context) {
		apiKey = middleware.GetWorkflowAPIKey(c)
		c.Status(http.StatusOK)
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/test", http.NoBody)
	req.Header.Set("X-Unified-UI-Workflow-API-Key", "agent-key")
	router.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)
	require.Equal(t, "agent-key", apiKey)
}

func TestAuthMiddleware_AuthenticateWorkflowAPIKey_Missing(t *testing.T) {
	router := gin.New()
	am := middleware.NewAuthMiddleware("http://platform")
	router.Use(am.AuthenticateWorkflowAPIKey())
	router.GET("/test", func(c *gin.Context) { c.Status(http.StatusOK) })

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/test", http.NoBody)
	router.ServeHTTP(w, req)

	require.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestGetToken_NotSet(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/", http.NoBody)
	require.Equal(t, "", middleware.GetToken(c))
}

func TestGetWorkflowAPIKey_NotSet(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/", http.NoBody)
	require.Equal(t, "", middleware.GetWorkflowAPIKey(c))
}
