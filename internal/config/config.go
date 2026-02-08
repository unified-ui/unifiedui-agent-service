// Package config handles application configuration loading and management.
package config

import (
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/joho/godotenv"
)

// Config holds all configuration for the application.
type Config struct {
	Server   ServerConfig
	Cache    CacheConfig
	DocDB    DocDBConfig
	Vaults   VaultsConfig
	Platform PlatformConfig
	AppVault AppVaultConfig
	Log      LogConfig
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
	Type     string
	Host     string
	Port     string
	Password string
	DB       int
	TTL      time.Duration
}

// DocDBConfig holds document database configuration.
type DocDBConfig struct {
	Type     string
	URI      string
	Database string
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

// AppVaultConfig holds app vault key name configuration.
type AppVaultConfig struct {
	PlatformServiceKey string // Key name in vault for validating incoming platform requests
	AgentToPlatformKey string // Key name in vault for outgoing requests to platform
}

// LogConfig holds logging configuration.
type LogConfig struct {
	Level  string
	Format string
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
			Type:     getEnv("CACHE_TYPE", "redis"),
			Host:     getEnv("REDIS_HOST", "localhost"),
			Port:     getEnv("REDIS_PORT", "6379"),
			Password: getEnv("REDIS_PASSWORD", ""),
			DB:       getEnvAsInt("REDIS_DB", 0),
			TTL:      time.Duration(getEnvAsInt("CACHE_TTL_SECONDS", 180)) * time.Second,
		},
		DocDB: DocDBConfig{
			Type:     getEnv("DOCDB_TYPE", "mongodb"),
			URI:      getEnv("MONGODB_URI", "mongodb://localhost:27017"),
			Database: getEnv("MONGODB_DATABASE", "unifiedui"),
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
		AppVault: AppVaultConfig{
			PlatformServiceKey: getEnv("APP_VAULT_PLATFORM_SERVICE_KEY", "PLATFORM_TO_AGENT_SERVICE_KEY"),
			AgentToPlatformKey: getEnv("APP_VAULT_AGENT_TO_PLATFORM_KEY", "AGENT_TO_PLATFORM_SERVICE_KEY"),
		},
		Log: LogConfig{
			Level:  getEnv("LOG_LEVEL", "info"),
			Format: getEnv("LOG_FORMAT", "json"),
		},
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
