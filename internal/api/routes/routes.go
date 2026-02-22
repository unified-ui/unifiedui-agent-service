// Package routes defines the HTTP routes for the UnifiedUI Agent Service.
package routes

import (
	"github.com/gin-gonic/gin"

	"github.com/unifiedui/agent-service/internal/api/handlers"
	"github.com/unifiedui/agent-service/internal/api/middleware"
)

// Config holds the dependencies for setting up routes.
type Config struct {
	HealthHandler    *handlers.HealthHandler
	MessagesHandler  *handlers.MessagesHandler
	ReactionsHandler *handlers.ReactionsHandler
	TracesHandler    *handlers.TracesHandler
	DataHandler      *handlers.DataHandler
	AIHandler        *handlers.AIHandler
	AuthMiddleware   *middleware.AuthMiddleware
	ServiceKeyMw     *middleware.ServiceKeyMiddleware
}

// Setup configures all routes on the Gin engine.
func Setup(r *gin.Engine, cfg *Config) {
	v1 := r.Group("/api/v1/agent-service")

	v1.GET("/health", cfg.HealthHandler.Health)
	v1.GET("/ready", cfg.HealthHandler.Ready)
	v1.GET("/live", cfg.HealthHandler.Live)

	protected := v1.Group("")
	protected.Use(cfg.AuthMiddleware.Authenticate())

	tenants := protected.Group("/tenants/:tenantId")

	conversation := tenants.Group("/conversation")
	conversation.GET("/messages", cfg.MessagesHandler.GetMessages)
	conversation.POST("/messages", cfg.MessagesHandler.SendMessage)

	conversations := tenants.Group("/conversations/:conversationId")
	conversations.PUT("/messages/:messageId", cfg.MessagesHandler.EditMessage)
	conversations.DELETE("/messages/:messageId", cfg.MessagesHandler.DeleteMessage)

	if cfg.ReactionsHandler != nil {
		reactions := conversations.Group("/messages/:messageId/reactions")
		reactions.POST("", cfg.ReactionsHandler.UpsertReaction)
		reactions.DELETE("", cfg.ReactionsHandler.DeleteReaction)
		reactions.GET("", cfg.ReactionsHandler.GetReactions)
	}

	conversations.GET("/traces", cfg.TracesHandler.GetConversationTraces)
	conversations.PUT("/traces", cfg.TracesHandler.RefreshConversationTrace)
	conversations.PUT("/traces/import/refresh", cfg.TracesHandler.ImportConversationTrace)

	traces := tenants.Group("/traces")
	traces.GET("/:traceId", cfg.TracesHandler.GetTrace)
	traces.DELETE("/:traceId", cfg.TracesHandler.DeleteTrace)

	tenants.GET("/autonomous-agents/traces", cfg.TracesHandler.ListAutonomousAgentTraces)

	if cfg.AIHandler != nil {
		aiRoutes := tenants.Group("/ai")
		aiRoutes.POST("/generate-description", cfg.AIHandler.GenerateDescription)
		aiRoutes.POST("/analyze-trace", cfg.AIHandler.AnalyzeTrace)
		aiRoutes.POST("/summarize-trace", cfg.AIHandler.SummarizeTrace)
		aiRoutes.POST("/trace-chat", cfg.AIHandler.TraceChat)
		aiRoutes.POST("/test-model", cfg.AIHandler.TestModel)
		aiRoutes.GET("/capabilities", cfg.AIHandler.GetCapabilities)
	}

	agents := tenants.Group("/autonomous-agents/:agentId")
	agents.GET("/traces", cfg.TracesHandler.GetAutonomousAgentTraces)
	agents.PUT("/traces", cfg.TracesHandler.RefreshAutonomousAgentTrace)

	agentImport := v1.Group("/tenants/:tenantId")
	agentImport.Use(cfg.AuthMiddleware.AuthenticateFlexible())

	agentImportRoutes := agentImport.Group("/autonomous-agents/:agentId")
	agentImportRoutes.PUT("/traces/import", cfg.TracesHandler.ImportAutonomousAgentTrace)
	agentImportRoutes.PUT("/traces/:traceId/import/refresh", cfg.TracesHandler.RefreshAutonomousAgentImportTrace)

	flexibleAuth := v1.Group("/tenants/:tenantId")
	flexibleAuth.Use(cfg.AuthMiddleware.AuthenticateFlexible())

	flexibleTraces := flexibleAuth.Group("/traces")
	flexibleTraces.POST("", cfg.TracesHandler.CreateTrace)
	flexibleTraces.POST("/:traceId/nodes", cfg.TracesHandler.AddNodes)
	flexibleTraces.POST("/:traceId/logs", cfg.TracesHandler.AddLogs)

	if cfg.ServiceKeyMw != nil && cfg.DataHandler != nil {
		serviceAuth := v1.Group("/tenants/:tenantId")
		serviceAuth.Use(cfg.ServiceKeyMw.AuthenticateServiceKey())
		serviceAuth.DELETE("/conversations/:conversationId/data", cfg.DataHandler.DeleteConversationData)
		serviceAuth.DELETE("/autonomous-agents/:agentId/data", cfg.DataHandler.DeleteAutonomousAgentData)
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
