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
	DataHandler     *handlers.DataHandler
	AIHandler       *handlers.AIHandler
	AuthMiddleware  *middleware.AuthMiddleware
	ServiceKeyMw    *middleware.ServiceKeyMiddleware
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
				// Get, delete trace by ID
				traces.GET("/:traceId", cfg.TracesHandler.GetTrace)
				traces.DELETE("/:traceId", cfg.TracesHandler.DeleteTrace)
			}

			// --- Conversation Traces Routes ---
			conversations := tenants.Group("/conversations/:conversationId")
			{
				// Get traces for conversation
				conversations.GET("/traces", cfg.TracesHandler.GetConversationTraces)
				// Refresh (replace) trace for conversation
				conversations.PUT("/traces", cfg.TracesHandler.RefreshConversationTrace)
				// Import traces from external system (Foundry, N8N)
				conversations.PUT("/traces/import/refresh", cfg.TracesHandler.ImportConversationTrace)
			}

			// --- Autonomous Agent Routes ---
			// List all autonomous agent traces
			tenants.GET("/autonomous-agents/traces", cfg.TracesHandler.ListAutonomousAgentTraces)

			// --- AI Feature Routes ---
			if cfg.AIHandler != nil {
				aiRoutes := tenants.Group("/ai")
				{
					aiRoutes.POST("/generate-description", cfg.AIHandler.GenerateDescription)
					aiRoutes.POST("/analyze-trace", cfg.AIHandler.AnalyzeTrace)
					aiRoutes.POST("/summarize-trace", cfg.AIHandler.SummarizeTrace)
					aiRoutes.POST("/trace-chat", cfg.AIHandler.TraceChat)
					aiRoutes.POST("/test-model", cfg.AIHandler.TestModel)
					aiRoutes.GET("/capabilities", cfg.AIHandler.GetCapabilities)
				}
			}

			// Specific autonomous agent routes
			agents := tenants.Group("/autonomous-agents/:agentId")
			{
				// Get traces for agent
				agents.GET("/traces", cfg.TracesHandler.GetAutonomousAgentTraces)
				// Refresh (replace) trace for agent
				agents.PUT("/traces", cfg.TracesHandler.RefreshAutonomousAgentTrace)
			}
		}

		// --- Autonomous Agent Import Routes (API Key Auth) ---
		// These routes use X-Unified-UI-Autonomous-Agent-API-Key header instead of Bearer token
		agentImport := v1.Group("/tenants/:tenantId")
		agentImport.Use(cfg.AuthMiddleware.AuthenticateAutonomousAgentAPIKey())
		{
			agentImportRoutes := agentImport.Group("/autonomous-agents/:agentId")
			{
				// Import/upsert traces for an autonomous agent (create or replace by executionId)
				agentImportRoutes.PUT("/traces/import", cfg.TracesHandler.ImportAutonomousAgentTrace)
				// Refresh imported trace for an autonomous agent
				agentImportRoutes.PUT("/traces/:traceId/import/refresh", cfg.TracesHandler.RefreshAutonomousAgentImportTrace)
			}
		}

		// --- Flexible Auth Routes (Bearer OR API Key) ---
		// These routes accept either Bearer token (for user/conversation context) or
		// X-Unified-UI-Autonomous-Agent-API-Key (for autonomous agent context).
		// The handler determines which auth type is required based on request content.
		flexibleAuth := v1.Group("/tenants/:tenantId")
		flexibleAuth.Use(cfg.AuthMiddleware.AuthenticateFlexible())
		{
			flexibleTraces := flexibleAuth.Group("/traces")
			{
				// Create a new trace (Bearer for conversation, API key for agent)
				flexibleTraces.POST("", cfg.TracesHandler.CreateTrace)
				// Add nodes/logs to existing trace
				flexibleTraces.POST("/:traceId/nodes", cfg.TracesHandler.AddNodes)
				flexibleTraces.POST("/:traceId/logs", cfg.TracesHandler.AddLogs)
			}
		}

		// --- Service Key Auth Routes (internal service-to-service) ---
		if cfg.ServiceKeyMw != nil && cfg.DataHandler != nil {
			serviceAuth := v1.Group("/tenants/:tenantId")
			serviceAuth.Use(cfg.ServiceKeyMw.AuthenticateServiceKey())
			{
				serviceAuth.DELETE("/conversations/:conversationId/data", cfg.DataHandler.DeleteConversationData)
				serviceAuth.DELETE("/autonomous-agents/:agentId/data", cfg.DataHandler.DeleteAutonomousAgentData)
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
