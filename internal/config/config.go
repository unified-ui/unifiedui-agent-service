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
	Vault    VaultConfig
	Platform PlatformConfig
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

// VaultConfig holds vault configuration.
type VaultConfig struct {
	Type             string
	AzureKeyVaultURL string
	HashiCorpAddr    string
	HashiCorpToken   string
	EncryptionKey    string
}

// PlatformConfig holds platform service configuration.
type PlatformConfig struct {
	URL     string
	Timeout time.Duration
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
		Vault: VaultConfig{
			Type:             getEnv("VAULT_TYPE", "dotenv"),
			AzureKeyVaultURL: getEnv("AZURE_KEYVAULT_URL", ""),
			HashiCorpAddr:    getEnv("HASHICORP_VAULT_ADDR", ""),
			HashiCorpToken:   getEnv("HASHICORP_VAULT_TOKEN", ""),
			EncryptionKey:    getEnv("SECRETS_ENCRYPTION_KEY", ""),
		},
		Platform: PlatformConfig{
			URL:     getEnv("PLATFORM_SERVICE_URL", "http://localhost:8081"),
			Timeout: time.Duration(getEnvAsInt("PLATFORM_SERVICE_TIMEOUT_SECONDS", 30)) * time.Second,
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
