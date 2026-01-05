package middleware

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// CORSConfig contains the configuration for CORS middleware.
type CORSConfig struct {
	AllowOrigins     []string
	AllowMethods     []string
	AllowHeaders     []string
	ExposeHeaders    []string
	AllowCredentials bool
	MaxAge           int
}

// DefaultCORSConfig returns a default CORS configuration.
func DefaultCORSConfig() CORSConfig {
	return CORSConfig{
		AllowOrigins: []string{
			"http://localhost:5173",
			"http://localhost:5174",
			"http://localhost:3000",
		},
		AllowMethods: []string{
			http.MethodGet,
			http.MethodPost,
			http.MethodPut,
			http.MethodPatch,
			http.MethodDelete,
			http.MethodOptions,
			http.MethodHead,
		},
		AllowHeaders: []string{
			"Origin",
			"Content-Type",
			"Content-Length",
			"Accept",
			"Accept-Encoding",
			"Accept-Language",
			"Authorization",
			"X-Requested-With",
			"X-Request-ID",
			"X-Correlation-ID",
			"Cache-Control",
		},
		ExposeHeaders: []string{
			"Content-Length",
			"Content-Type",
			"X-Request-ID",
			"X-Correlation-ID",
		},
		AllowCredentials: true,
		MaxAge:           86400, // 24 hours
	}
}

// NewCORSMiddleware creates a new CORS middleware with the given configuration.
func NewCORSMiddleware(cfg CORSConfig) gin.HandlerFunc {
	return func(c *gin.Context) {
		origin := c.Request.Header.Get("Origin")

		// Check if origin is allowed
		allowedOrigin := ""
		for _, o := range cfg.AllowOrigins {
			if o == "*" || o == origin {
				allowedOrigin = origin
				break
			}
		}

		// Always set CORS headers if origin is allowed
		if allowedOrigin != "" {
			c.Header("Access-Control-Allow-Origin", allowedOrigin)
			c.Header("Access-Control-Allow-Credentials", "true")
			c.Header("Access-Control-Allow-Headers", joinStrings(cfg.AllowHeaders))
			c.Header("Access-Control-Allow-Methods", joinStrings(cfg.AllowMethods))
			c.Header("Access-Control-Expose-Headers", joinStrings(cfg.ExposeHeaders))
			c.Header("Access-Control-Max-Age", "86400")
			c.Header("Vary", "Origin")
		}

		// Handle preflight request - must abort with 204 before reaching route handlers
		if c.Request.Method == http.MethodOptions {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}

		c.Next()
	}
}

// SetupCORSRoutes adds explicit OPTIONS handlers for all routes to ensure CORS preflight works.
// This is necessary because Gin's middleware doesn't run for 404 routes.
func SetupCORSRoutes(router *gin.Engine, cfg CORSConfig) {
	handler := NewCORSMiddleware(cfg)

	// Handle OPTIONS for any route
	router.OPTIONS("/*path", handler)
}

// joinStrings joins a slice of strings with comma separator.
func joinStrings(strs []string) string {
	if len(strs) == 0 {
		return ""
	}
	result := strs[0]
	for i := 1; i < len(strs); i++ {
		result += ", " + strs[i]
	}
	return result
}
