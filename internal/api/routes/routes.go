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
			}

			// --- Traces CRUD Routes ---
			traces := tenants.Group("/traces")
			{
				// Create a new trace
				traces.POST("", cfg.TracesHandler.CreateTrace)

				// Get, delete trace by ID
				traces.GET("/:traceId", cfg.TracesHandler.GetTrace)
				traces.DELETE("/:traceId", cfg.TracesHandler.DeleteTrace)

				// Add nodes/logs to existing trace
				traces.POST("/:traceId/nodes", cfg.TracesHandler.AddNodes)
				traces.POST("/:traceId/logs", cfg.TracesHandler.AddLogs)
			}

			// --- Conversation Traces Routes ---
			conversations := tenants.Group("/conversations/:conversationId")
			{
				// Get traces for conversation
				conversations.GET("/traces", cfg.TracesHandler.GetConversationTraces)
				// Refresh (replace) trace for conversation
				conversations.PUT("/traces", cfg.TracesHandler.RefreshConversationTrace)
			}

			// --- Autonomous Agent Routes ---
			// List all autonomous agent traces
			tenants.GET("/autonomous-agents/traces", cfg.TracesHandler.ListAutonomousAgentTraces)

			// Specific autonomous agent routes
			agents := tenants.Group("/autonomous-agents/:agentId")
			{
				// Get traces for agent
				agents.GET("/traces", cfg.TracesHandler.GetAutonomousAgentTraces)
				// Refresh (replace) trace for agent
				agents.PUT("/traces", cfg.TracesHandler.RefreshAutonomousAgentTrace)
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
