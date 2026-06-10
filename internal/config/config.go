// Package config handles application configuration loading and management.
package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/joho/godotenv"
)

// Config holds all configuration for the application.
type Config struct {
	Server        ServerConfig
	Cache         CacheConfig
	DocDB         DocDBConfig
	Vaults        VaultsConfig
	Platform      PlatformConfig
	ReactService  ReactServiceConfig
	AppVault      AppVaultConfig
	Log           LogConfig
	CORS          CORSConfig
	Deployment    DeploymentConfig
	DebugBackdoor DebugBackdoorConfig
}

// ServerConfig holds server-related configuration.
type ServerConfig struct {
	Host    string
	Port    int
	GinMode string
}

// Address returns the server address in host:port format.
func (c ServerConfig) Address() string {
	return fmt.Sprintf("%s:%d", c.Host, c.Port)
}

// CacheConfig holds cache-related configuration.
type CacheConfig struct {
	Type           string
	Host           string
	Port           string
	Password       string
	DB             int
	TTL            time.Duration
	ConfigCacheTTL time.Duration
}

// DocDBConfig holds document database configuration.
type DocDBConfig struct {
	Type               string
	URI                string // MongoDB URI (for mongodb type)
	Database           string
	CosmosDBEndpoint   string // CosmosDB endpoint (for cosmosdb type)
	CosmosDBKey        string // CosmosDB key (optional, empty = use managed identity)
	UseManagedIdentity bool   // Use managed identity for CosmosDB auth
}

// VaultConfig holds vault configuration for a specific vault purpose.
type VaultConfig struct {
	Type             string
	AzureKeyVaultURL string
	HashiCorpAddr    string
	HashiCorpToken   string
}

// VaultsConfig holds configuration for all vault instances.
type VaultsConfig struct {
	VaultType        string
	AppVaultType     string
	SecretsVaultType string
	App              VaultConfig
	Secrets          VaultConfig
	EncryptionKey    string
}

// ResolvedAppVaultType returns the effective vault type for the app vault.
func (v VaultsConfig) ResolvedAppVaultType() string {
	if v.AppVaultType != "" {
		return v.AppVaultType
	}
	return v.VaultType
}

// ResolvedSecretsVaultType returns the effective vault type for the secrets vault.
func (v VaultsConfig) ResolvedSecretsVaultType() string {
	if v.SecretsVaultType != "" {
		return v.SecretsVaultType
	}
	return v.VaultType
}

// PlatformConfig holds platform service configuration.
type PlatformConfig struct {
	URL        string
	ConfigPath string
	Timeout    time.Duration
	ServiceKey string // X_AGENT_SERVICE_KEY for service-to-service authentication
}

// ReactServiceConfig holds ReACT agent service configuration.
type ReactServiceConfig struct {
	URL     string
	Timeout time.Duration
}

// AppVaultConfig holds app vault key name configuration.
type AppVaultConfig struct {
	PlatformServiceKey   string // Key name in vault for validating incoming platform requests
	AgentToPlatformKey   string // Key name in vault for outgoing requests to platform
	AgentToReactAgentKey string // Key name in vault for outgoing requests to ReACT agent service
}

// LogConfig holds logging configuration.
type LogConfig struct {
	Level  string
	Format string
}

// CORSConfig holds CORS configuration.
type CORSConfig struct {
	AllowOrigins []string
}

// DeploymentConfig holds deployment-mode information used for runtime safety guards.
type DeploymentConfig struct {
	Mode string
}

// DebugBackdoorConfig holds the debug backdoor configuration (REQ 007).
type DebugBackdoorConfig struct {
	Enabled bool
	Secret  string
}

// Validate enforces secret length and production guard rules.
func (d DebugBackdoorConfig) Validate(deploymentMode string) error {
	if !d.Enabled {
		return nil
	}
	if deploymentMode == "production" {
		return fmt.Errorf("ENABLE_DEBUG_BACK_DOOR MUST be false when DEPLOYMENT_MODE=production")
	}
	if len(d.Secret) < 32 {
		return fmt.Errorf("DEBUG_BACK_DOOR_SECRET MUST be set and at least 32 characters when ENABLE_DEBUG_BACK_DOOR=true")
	}
	return nil
}

// Load loads configuration from environment variables.
func Load() (*Config, error) {
	// Load .env file if it exists (ignore error if not found)
	_ = godotenv.Load()

	cfg := &Config{
		Server: ServerConfig{
			Host:    getEnv("SERVER_HOST", "0.0.0.0"),
			Port:    getEnvAsInt("SERVER_PORT", 8080),
			GinMode: getEnv("GIN_MODE", "debug"),
		},
		Cache: CacheConfig{
			Type:           getEnv("CACHE_TYPE", "redis"),
			Host:           getEnv("REDIS_HOST", "localhost"),
			Port:           getEnv("REDIS_PORT", "6379"),
			Password:       getEnv("REDIS_PASSWORD", ""),
			DB:             getEnvAsInt("REDIS_DB", 0),
			TTL:            time.Duration(getEnvAsInt("CACHE_TTL_SECONDS", 180)) * time.Second,
			ConfigCacheTTL: time.Duration(getEnvAsInt("CONFIG_CACHE_TTL_SECONDS", 300)) * time.Second,
		},
		DocDB: DocDBConfig{
			Type:               getEnv("DOCDB_TYPE", "mongodb"),
			URI:                getEnv("MONGODB_URI", "mongodb://localhost:27017"),
			Database:           getEnv("MONGODB_DATABASE", "unifiedui"),
			CosmosDBEndpoint:   getEnv("COSMOSDB_ENDPOINT", ""),
			CosmosDBKey:        getEnv("COSMOSDB_KEY", ""),
			UseManagedIdentity: getEnvAsBool("COSMOSDB_USE_MANAGED_IDENTITY", true),
		},
		Vaults: VaultsConfig{
			VaultType:        getEnv("VAULT_TYPE", "dotenv"),
			AppVaultType:     getEnv("APP_VAULT_TYPE", ""),
			SecretsVaultType: getEnv("SECRETS_VAULT_TYPE", ""),
			App: VaultConfig{
				AzureKeyVaultURL: getEnv("APP_AZURE_KEYVAULT_URL", ""),
				HashiCorpAddr:    getEnv("APP_HASHICORP_VAULT_ADDR", ""),
				HashiCorpToken:   getEnv("APP_HASHICORP_VAULT_TOKEN", ""),
			},
			Secrets: VaultConfig{
				AzureKeyVaultURL: getEnv("SECRETS_AZURE_KEYVAULT_URL", ""),
				HashiCorpAddr:    getEnv("SECRETS_HASHICORP_VAULT_ADDR", ""),
				HashiCorpToken:   getEnv("SECRETS_HASHICORP_VAULT_TOKEN", ""),
			},
			EncryptionKey: getEnv("SECRETS_ENCRYPTION_KEY", ""),
		},
		Platform: PlatformConfig{
			URL:        getEnv("PLATFORM_SERVICE_URL", "http://localhost:8081"),
			ConfigPath: getEnv("PLATFORM_CONFIG_PATH", "poc/n8n/config.json"),
			Timeout:    time.Duration(getEnvAsInt("PLATFORM_SERVICE_TIMEOUT_SECONDS", 30)) * time.Second,
			ServiceKey: getEnv("X_AGENT_SERVICE_KEY", ""),
		},
		ReactService: ReactServiceConfig{
			URL:     getEnv("REACT_SERVICE_URL", "http://localhost:8086"),
			Timeout: time.Duration(getEnvAsInt("REACT_SERVICE_TIMEOUT_SECONDS", 300)) * time.Second,
		},
		AppVault: AppVaultConfig{
			PlatformServiceKey:   getEnv("APP_VAULT_PLATFORM_SERVICE_KEY", "PLATFORM_TO_AGENT_SERVICE_KEY"),
			AgentToPlatformKey:   getEnv("APP_VAULT_AGENT_TO_PLATFORM_KEY", "AGENT_TO_PLATFORM_SERVICE_KEY"),
			AgentToReactAgentKey: getEnv("APP_VAULT_AGENT_TO_REACT_KEY", "AGENT_TO_REACT_SERVICE_KEY"),
		},
		Log: LogConfig{
			Level:  getEnv("LOG_LEVEL", "info"),
			Format: getEnv("LOG_FORMAT", "json"),
		},
		CORS: CORSConfig{
			AllowOrigins: getEnvAsStringSlice("CORS_ORIGINS", []string{
				"http://localhost:5173",
				"http://localhost:5174",
				"http://localhost:3000",
			}),
		},
		Deployment: DeploymentConfig{
			Mode: getEnv("DEPLOYMENT_MODE", "self-hosted"),
		},
		DebugBackdoor: DebugBackdoorConfig{
			Enabled: getEnvAsBool("ENABLE_DEBUG_BACK_DOOR", false),
			Secret:  getEnv("DEBUG_BACK_DOOR_SECRET", ""),
		},
	}

	if err := cfg.DebugBackdoor.Validate(cfg.Deployment.Mode); err != nil {
		return nil, err
	}

	return cfg, nil
}

// getEnv gets an environment variable with a default value.
func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// getEnvAsInt gets an environment variable as an integer with a default value.
func getEnvAsInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if intValue, err := strconv.Atoi(value); err == nil {
			return intValue
		}
	}
	return defaultValue
}

// getEnvAsBool gets an environment variable as a boolean with a default value.
func getEnvAsBool(key string, defaultValue bool) bool {
	if value := os.Getenv(key); value != "" {
		boolValue, err := strconv.ParseBool(value)
		if err == nil {
			return boolValue
		}
	}
	return defaultValue
}

// getEnvAsStringSlice gets an environment variable as a comma-separated string slice.
func getEnvAsStringSlice(key string, defaultValue []string) []string {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}
	parts := strings.Split(value, ",")
	result := make([]string, 0, len(parts))
	for _, part := range parts {
		trimmed := strings.TrimSpace(part)
		if trimmed != "" {
			result = append(result, trimmed)
		}
	}
	if len(result) == 0 {
		return defaultValue
	}
	return result
}
