// Package middleware provides HTTP middleware for the API.
package middleware

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

// AuthMiddleware handles authentication by forwarding tokens to Platform Service.
type AuthMiddleware struct {
	// platformServiceURL is the URL of the Platform Service.
	platformServiceURL string
}

// NewAuthMiddleware creates a new AuthMiddleware.
func NewAuthMiddleware(platformServiceURL string) *AuthMiddleware {
	return &AuthMiddleware{
		platformServiceURL: platformServiceURL,
	}
}

// Authenticate returns a gin middleware that validates the Bearer token.
// It extracts the token and stores it in the context for downstream handlers.
func (m *AuthMiddleware) Authenticate() gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"code":    "UNAUTHORIZED",
				"message": "missing authorization header",
			})
			return
		}

		// Extract Bearer token
		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || !strings.EqualFold(parts[0], "bearer") {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"code":    "UNAUTHORIZED",
				"message": "invalid authorization header format",
			})
			return
		}

		token := parts[1]
		if token == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"code":    "UNAUTHORIZED",
				"message": "empty token",
			})
			return
		}

		// Store token in context for downstream handlers
		c.Set("auth_token", token)

		c.Next()
	}
}

// GetToken retrieves the auth token from the gin context.
func GetToken(c *gin.Context) string {
	if token, exists := c.Get("auth_token"); exists {
		return token.(string)
	}
	return ""
}

// GetAutonomousAgentAPIKey retrieves the autonomous agent API key from the gin context.
func GetAutonomousAgentAPIKey(c *gin.Context) string {
	if key, exists := c.Get("autonomous_agent_api_key"); exists {
		return key.(string)
	}
	return ""
}

// AuthenticateFlexible returns a gin middleware that accepts EITHER a Bearer token OR
// an X-Unified-UI-Autonomous-Agent-API-Key header. This enables endpoints like POST /traces
// to serve both user-initiated requests (Bearer) and external agent requests (API key).
// The handler is responsible for checking which auth type was provided and validating accordingly.
func (m *AuthMiddleware) AuthenticateFlexible() gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		apiKey := c.GetHeader("X-Unified-UI-Autonomous-Agent-API-Key")

		if authHeader == "" && apiKey == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"code":    "UNAUTHORIZED",
				"message": "missing authentication: provide either Authorization header or X-Unified-UI-Autonomous-Agent-API-Key header",
			})
			return
		}

		// Try Bearer token first
		if authHeader != "" {
			parts := strings.SplitN(authHeader, " ", 2)
			if len(parts) == 2 && strings.EqualFold(parts[0], "bearer") && parts[1] != "" {
				c.Set("auth_token", parts[1])
				c.Next()
				return
			}
		}

		// Fall back to API key
		if apiKey != "" {
			c.Set("autonomous_agent_api_key", apiKey)
			c.Next()
			return
		}

		c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
			"code":    "UNAUTHORIZED",
			"message": "invalid authentication credentials",
		})
	}
}

// AuthenticateAutonomousAgentAPIKey returns a gin middleware that validates the autonomous agent API key.
// It extracts the X-Unified-UI-Autonomous-Agent-API-Key header and stores it in the context.
// Unlike Bearer token auth, this API key will be validated against the platform service's
// autonomous agent config endpoint (which validates against primary/secondary keys).
func (m *AuthMiddleware) AuthenticateAutonomousAgentAPIKey() gin.HandlerFunc {
	return func(c *gin.Context) {
		apiKey := c.GetHeader("X-Unified-UI-Autonomous-Agent-API-Key")
		if apiKey == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"code":    "UNAUTHORIZED",
				"message": "missing X-Unified-UI-Autonomous-Agent-API-Key header",
			})
			return
		}

		// Store API key in context for downstream handlers
		c.Set("autonomous_agent_api_key", apiKey)

		c.Next()
	}
}
