// Package routes defines the HTTP routes for the UnifiedUI Chat Service.
package routes

import (
	"github.com/gin-gonic/gin"

	"github.com/unifiedui/chat-service/internal/api/handlers"
	"github.com/unifiedui/chat-service/internal/api/middleware"
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
	// Health check routes (no auth required)
	r.GET("/health", cfg.HealthHandler.Health)
	r.GET("/ready", cfg.HealthHandler.Ready)
	r.GET("/live", cfg.HealthHandler.Live)

	// API v1 routes
	v1 := r.Group("/api/v1/agent-service")
	{
		// Apply auth middleware to all API routes
		v1.Use(cfg.AuthMiddleware.Authenticate())

		// Tenant-scoped routes
		tenants := v1.Group("/tenants/:tenantId")
		{
			// Conversation routes
			conversations := tenants.Group("/conversations/:conversationId")
			{
				// Messages
				conversations.GET("/messages", cfg.MessagesHandler.GetMessages)
				conversations.POST("/messages", cfg.MessagesHandler.SendMessage)

				// Message traces
				conversations.GET("/messages/:messageId/traces", cfg.TracesHandler.GetMessageTraces)
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
