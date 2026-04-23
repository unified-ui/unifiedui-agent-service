// Package middleware provides HTTP middleware for the API.
package middleware

import (
	"context"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/unifiedui/agent-service/internal/services/platform"
)

// TokenValidator validates a bearer token by calling an authoritative source
// (typically the platform service) and returns the resolved user info.
type TokenValidator interface {
	GetMe(ctx context.Context, authToken string) (*platform.UserInfo, error)
}

// tokenCacheEntry holds a cached validation result with its expiry deadline.
type tokenCacheEntry struct {
	user      *platform.UserInfo
	expiresAt time.Time
}

// AuthMiddleware validates Bearer tokens by delegating to the platform service.
// Successful validations are cached for a short TTL to amortise the round-trip.
type AuthMiddleware struct {
	validator TokenValidator
	cacheTTL  time.Duration
	cache     sync.Map
}

// NewAuthMiddleware creates a new AuthMiddleware backed by the given TokenValidator.
// A nil validator disables remote validation (for legacy tests only) and tokens
// will be accepted after the syntactic Bearer check; production callers MUST
// pass a real platform client.
func NewAuthMiddleware(validator TokenValidator) *AuthMiddleware {
	return &AuthMiddleware{
		validator: validator,
		cacheTTL:  60 * time.Second,
	}
}

// SetCacheTTL overrides the default validation cache TTL. Useful for tests.
func (m *AuthMiddleware) SetCacheTTL(ttl time.Duration) {
	m.cacheTTL = ttl
}

// validateToken returns the resolved user, fetching from the validator and
// caching the result. A nil validator silently passes through (legacy mode).
func (m *AuthMiddleware) validateToken(ctx context.Context, token string) (*platform.UserInfo, error) {
	if m.validator == nil {
		return nil, nil
	}

	now := time.Now()
	if cached, ok := m.cache.Load(token); ok {
		entry := cached.(tokenCacheEntry)
		if entry.expiresAt.After(now) {
			return entry.user, nil
		}
		m.cache.Delete(token)
	}

	user, err := m.validator.GetMe(ctx, token)
	if err != nil {
		return nil, err
	}

	m.cache.Store(token, tokenCacheEntry{user: user, expiresAt: now.Add(m.cacheTTL)})
	return user, nil
}

// Authenticate returns a gin middleware that validates the Bearer token by
// calling the platform service. The validated user info is stored in the
// gin context for downstream handlers to consume without re-fetching.
func (m *AuthMiddleware) Authenticate() gin.HandlerFunc {
	return func(c *gin.Context) {
		token, ok := extractBearerToken(c)
		if !ok {
			return
		}

		user, err := m.validateToken(c.Request.Context(), token)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"code":    "UNAUTHORIZED",
				"message": "invalid or expired token",
			})
			return
		}

		c.Set("auth_token", token)
		if user != nil {
			c.Set("auth_user", user)
		}

		c.Next()
	}
}

// extractBearerToken pulls the token from the Authorization header and aborts
// the request with 401 on any structural problem. Returns (token, ok).
func extractBearerToken(c *gin.Context) (string, bool) {
	authHeader := c.GetHeader("Authorization")
	if authHeader == "" {
		c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
			"code":    "UNAUTHORIZED",
			"message": "missing authorization header",
		})
		return "", false
	}

	parts := strings.SplitN(authHeader, " ", 2)
	if len(parts) != 2 || !strings.EqualFold(parts[0], "bearer") {
		c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
			"code":    "UNAUTHORIZED",
			"message": "invalid authorization header format",
		})
		return "", false
	}

	token := parts[1]
	if token == "" {
		c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
			"code":    "UNAUTHORIZED",
			"message": "empty token",
		})
		return "", false
	}

	return token, true
}

// GetToken retrieves the auth token from the gin context.
func GetToken(c *gin.Context) string {
	if token, exists := c.Get("auth_token"); exists {
		return token.(string)
	}
	return ""
}

// GetAuthUser retrieves the validated user info from the gin context, if any.
func GetAuthUser(c *gin.Context) *platform.UserInfo {
	if user, exists := c.Get("auth_user"); exists {
		if u, ok := user.(*platform.UserInfo); ok {
			return u
		}
	}
	return nil
}

// GetWorkflowAPIKey retrieves the workflow API key from the gin context.
func GetWorkflowAPIKey(c *gin.Context) string {
	if key, exists := c.Get("workflow_api_key"); exists {
		return key.(string)
	}
	return ""
}

// AuthenticateFlexible returns a gin middleware that accepts EITHER a Bearer token OR
// an X-Unified-UI-Workflow-API-Key header. Bearer tokens are validated against the
// platform service; API keys are validated downstream by the handler against the
// workflow's primary/secondary keys.
func (m *AuthMiddleware) AuthenticateFlexible() gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		apiKey := c.GetHeader("X-Unified-UI-Workflow-API-Key")

		if authHeader == "" && apiKey == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"code":    "UNAUTHORIZED",
				"message": "missing authentication: provide either Authorization header or X-Unified-UI-Workflow-API-Key header",
			})
			return
		}

		if authHeader != "" {
			parts := strings.SplitN(authHeader, " ", 2)
			if len(parts) == 2 && strings.EqualFold(parts[0], "bearer") && parts[1] != "" {
				token := parts[1]
				user, err := m.validateToken(c.Request.Context(), token)
				if err != nil {
					c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
						"code":    "UNAUTHORIZED",
						"message": "invalid or expired token",
					})
					return
				}
				c.Set("auth_token", token)
				if user != nil {
					c.Set("auth_user", user)
				}
				c.Next()
				return
			}
		}

		if apiKey != "" {
			c.Set("workflow_api_key", apiKey)
			c.Next()
			return
		}

		c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
			"code":    "UNAUTHORIZED",
			"message": "invalid authentication credentials",
		})
	}
}

// AuthenticateWorkflowAPIKey returns a gin middleware that validates the workflow API key.
// It extracts the X-Unified-UI-Workflow-API-Key header and stores it in the context.
// Unlike Bearer token auth, this API key will be validated against the platform service's
// workflow config endpoint (which validates against primary/secondary keys).
func (m *AuthMiddleware) AuthenticateWorkflowAPIKey() gin.HandlerFunc {
	return func(c *gin.Context) {
		apiKey := c.GetHeader("X-Unified-UI-Workflow-API-Key")
		if apiKey == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"code":    "UNAUTHORIZED",
				"message": "missing X-Unified-UI-Workflow-API-Key header",
			})
			return
		}

		c.Set("workflow_api_key", apiKey)

		c.Next()
	}
}
