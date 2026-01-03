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
