// Package main is the entry point for the UnifiedUI Chat Service.
// @title UnifiedUI Chat Service API
// @version 1.0
// @description Unified abstraction layer for heterogeneous AI agent backends (N8N, Microsoft Foundry, Copilot, LangChain)
// @termsOfService http://swagger.io/terms/

// @contact.name API Support
// @contact.url https://github.com/unifiedui/agent-service
// @contact.email support@unifiedui.io

// @license.name MIT
// @license.url https://opensource.org/licenses/MIT

// @host localhost:8085
// @BasePath /
// @schemes http https

// @securityDefinitions.apikey BearerAuth
// @in header
// @name Authorization
// @description Bearer token authentication (MSAL)
package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"

	_ "github.com/unifiedui/agent-service/docs"
	"github.com/unifiedui/agent-service/internal/api/handlers"
	"github.com/unifiedui/agent-service/internal/api/middleware"
	"github.com/unifiedui/agent-service/internal/api/routes"
	"github.com/unifiedui/agent-service/internal/config"
	"github.com/unifiedui/agent-service/internal/core/cache"
	"github.com/unifiedui/agent-service/internal/core/docdb"
	"github.com/unifiedui/agent-service/internal/core/vault"
	rediscache "github.com/unifiedui/agent-service/internal/infrastructure/cache/redis"
	"github.com/unifiedui/agent-service/internal/infrastructure/docdb/mongodb"
	dotenvvault "github.com/unifiedui/agent-service/internal/infrastructure/vault/dotenv"
	"github.com/unifiedui/agent-service/internal/pkg/encryption"
	"github.com/unifiedui/agent-service/internal/services/agents"
	"github.com/unifiedui/agent-service/internal/services/platform"
	"github.com/unifiedui/agent-service/internal/services/session"
)

func main() {
	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("failed to load configuration: %v", err)
	}

	ctx := context.Background()

	// Initialize vault client using factory pattern
	vaultClient, err := createVaultClient(cfg.Vault)
	if err != nil {
		log.Fatalf("failed to initialize vault client: %v", err)
	}
	defer vaultClient.Close()

	// Initialize cache client using factory pattern
	cacheClient, err := createCacheClient(cfg.Cache)
	if err != nil {
		log.Fatalf("failed to initialize cache client: %v", err)
	}
	defer cacheClient.Close()

	// Initialize document db client using factory pattern
	docDBClient, err := createDocDBClient(ctx, cfg.DocDB)
	if err != nil {
		log.Fatalf("failed to initialize document db client: %v", err)
	}
	defer docDBClient.Close(ctx)

	// Ensure database indexes
	if err := docDBClient.EnsureIndexes(ctx); err != nil {
		log.Printf("warning: failed to ensure indexes: %v", err)
	}

	// Initialize encryptor
	encryptor, err := createEncryptor(cfg.Vault, vaultClient)
	if err != nil {
		log.Fatalf("failed to initialize encryptor: %v", err)
	}

	// Initialize session service
	sessionService, err := session.NewService(&session.Config{
		CacheClient: cacheClient,
		Encryptor:   encryptor,
		TTL:         cfg.Cache.TTL,
	})
	if err != nil {
		log.Fatalf("failed to initialize session service: %v", err)
	}

	// Set Gin mode
	gin.SetMode(cfg.Server.GinMode)

	// Setup router
	router := setupRouter(cfg, cacheClient, docDBClient, vaultClient, sessionService)

	// Create HTTP server
	srv := &http.Server{
		Addr:    cfg.Server.Address(),
		Handler: router,
	}

	// Start server in goroutine
	go func() {
		log.Printf("Starting server on %s", cfg.Server.Address())
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("failed to start server: %v", err)
		}
	}()

	// Wait for interrupt signal for graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Shutting down server...")

	// Graceful shutdown with timeout
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Fatalf("server forced to shutdown: %v", err)
	}

	log.Println("Server exited")
}

// createVaultClient creates a vault client based on the configuration.
func createVaultClient(cfg config.VaultConfig) (vault.Client, error) {
	vaultType := vault.Type(cfg.Type)

	switch vaultType {
	case vault.TypeDotEnv:
		return dotenvvault.NewClient()
	case vault.TypeAzure:
		// TODO: Implement Azure KeyVault client
		return nil, nil
	case vault.TypeHashiCorp:
		// TODO: Implement HashiCorp Vault client
		return nil, nil
	default:
		log.Fatalf("unsupported vault type: %s", cfg.Type)
		return nil, nil
	}
}

// createCacheClient creates a cache client based on the configuration.
func createCacheClient(cfg config.CacheConfig) (cache.Client, error) {
	cacheType := cache.Type(cfg.Type)

	switch cacheType {
	case cache.TypeRedis:
		return rediscache.NewClient(rediscache.Config{
			Host:       cfg.Host,
			Port:       cfg.Port,
			Password:   cfg.Password,
			DB:         cfg.DB,
			DefaultTTL: cfg.TTL,
		})
	default:
		log.Fatalf("unsupported cache type: %s", cfg.Type)
		return nil, nil
	}
}

// createDocDBClient creates a document database client based on the configuration.
func createDocDBClient(ctx context.Context, cfg config.DocDBConfig) (docdb.Client, error) {
	docDBType := docdb.Type(cfg.Type)

	switch docDBType {
	case docdb.TypeMongoDB:
		return mongodb.NewClient(ctx, &mongodb.ClientConfig{
			URI:          cfg.URI,
			DatabaseName: cfg.Database,
		})
	case docdb.TypeCosmosDB:
		// CosmosDB uses MongoDB protocol, so we can use the same client
		return mongodb.NewClient(ctx, &mongodb.ClientConfig{
			URI:          cfg.URI,
			DatabaseName: cfg.Database,
		})
	default:
		log.Fatalf("unsupported docdb type: %s", cfg.Type)
		return nil, nil
	}
}

// createEncryptor creates an encryptor based on the configuration.
func createEncryptor(cfg config.VaultConfig, vaultClient vault.Client) (encryption.Encryptor, error) {
	// Try to get encryption key from vault/env
	encryptionKey := cfg.EncryptionKey
	if encryptionKey == "" {
		// Try to get from vault
		key, err := vaultClient.GetSecret(context.Background(), "dotenv://SECRETS_ENCRYPTION_KEY", false)
		if err == nil && key != "" {
			encryptionKey = key
		}
	}

	if encryptionKey == "" {
		// Use NoOp encryptor in development
		log.Println("warning: SECRETS_ENCRYPTION_KEY not set, using NoOp encryptor")
		return encryption.NewNoOpEncryptor(), nil
	}

	return encryption.NewAESEncryptor(encryptionKey)
}

// setupRouter creates and configures the Gin router.
func setupRouter(cfg *config.Config, cacheClient cache.Client, docDBClient docdb.Client, vaultClient vault.Client, sessionService session.Service) *gin.Engine {
	router := gin.New()

	// Create middleware
	loggingMw := middleware.NewLoggingMiddleware()
	errorMw := middleware.NewErrorMiddleware()
	authMw := middleware.NewAuthMiddleware(cfg.Platform.URL)

	// Create platform client
	platformClient := platform.NewClient(&platform.ClientConfig{
		BaseURL:    cfg.Platform.URL,
		ConfigPath: cfg.Platform.ConfigPath,
		ServiceKey: cfg.Platform.ServiceKey,
		Timeout:    cfg.Platform.Timeout,
	})

	// Create agent factory
	agentFactory := agents.NewFactory()

	// Create handlers
	healthHandler := handlers.NewHealthHandler(cacheClient, docDBClient)
	messagesHandler := handlers.NewMessagesHandler(docDBClient, platformClient, agentFactory, sessionService)
	tracesHandler := handlers.NewTracesHandler(docDBClient)

	// Setup routes
	routesCfg := &routes.Config{
		HealthHandler:   healthHandler,
		MessagesHandler: messagesHandler,
		TracesHandler:   tracesHandler,
		AuthMiddleware:  authMw,
	}

	routes.SetupWithMiddleware(router, routesCfg, loggingMw, errorMw)

	// Swagger documentation endpoint
	router.GET("/docs/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

	return router
}
