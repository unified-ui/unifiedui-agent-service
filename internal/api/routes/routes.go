// Package routes defines the HTTP routes for the UnifiedUI Agent Service.
package routes

import (
	"github.com/gin-gonic/gin"

	"github.com/unifiedui/agent-service/internal/api/handlers"
	"github.com/unifiedui/agent-service/internal/api/middleware"
)

// Config holds the dependencies for setting up routes.
type Config struct {
	HealthHandler   *handlers.HealthHandler
	MessagesHandler *handlers.MessagesHandler
	TracesHandler   *handlers.TracesHandler
	AuthMiddleware  *middleware.AuthMiddleware
}

// Setup configures all routes on the Gin engine.
func Setup(r *gin.Engine, cfg *Config) {
	// API v1 routes - all routes under /api/v1/agent-service
	v1 := r.Group("/api/v1/agent-service")
	{
		// Health check routes (no auth required)
		v1.GET("/health", cfg.HealthHandler.Health)
		v1.GET("/ready", cfg.HealthHandler.Ready)
		v1.GET("/live", cfg.HealthHandler.Live)

		// Apply auth middleware to protected API routes
		protected := v1.Group("")
		protected.Use(cfg.AuthMiddleware.Authenticate())

		// Tenant-scoped routes
		tenants := protected.Group("/tenants/:tenantId")
		{
			// Conversation routes (conversationId in request body)
			conversation := tenants.Group("/conversation")
			{
				// Messages
				conversation.GET("/messages", cfg.MessagesHandler.GetMessages)
				conversation.POST("/messages", cfg.MessagesHandler.SendMessage)

				// Message traces
				conversation.GET("/messages/:messageId/traces", cfg.TracesHandler.GetMessageTraces)
			}

			// Autonomous agent routes
			agents := tenants.Group("/autonomous-agents/:agentId")
			{
				// Trace updates from agents
				agents.PUT("/traces", cfg.TracesHandler.UpdateTraces)
			}
		}
	}
}

// SetupWithMiddleware sets up routes with common middleware.
func SetupWithMiddleware(r *gin.Engine, cfg *Config, loggingMw *middleware.LoggingMiddleware, errorMw *middleware.ErrorMiddleware) {
	// Apply global middleware
	r.Use(loggingMw.Logger())
	r.Use(errorMw.Recovery())
	r.Use(gin.Recovery())

	// Setup routes
	Setup(r, cfg)
}
