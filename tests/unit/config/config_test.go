package config

import (
	"os"
	"testing"

	"github.com/unifiedui/agent-service/internal/config"

	"github.com/stretchr/testify/require"
)

func TestLoad_Defaults(t *testing.T) {
	for _, key := range []string{
		"SERVER_HOST", "SERVER_PORT", "GIN_MODE",
		"CACHE_TYPE", "REDIS_HOST", "REDIS_PORT",
		"DOCDB_TYPE", "MONGODB_URI", "MONGODB_DATABASE",
		"VAULT_TYPE", "LOG_LEVEL", "LOG_FORMAT",
	} {
		t.Setenv(key, "")
		os.Unsetenv(key)
	}

	cfg, err := config.Load()
	require.NoError(t, err)
	require.NotNil(t, cfg)
	require.Equal(t, "0.0.0.0", cfg.Server.Host)
	require.Equal(t, 8080, cfg.Server.Port)
	require.Equal(t, "debug", cfg.Server.GinMode)
	require.Equal(t, "redis", cfg.Cache.Type)
	require.Equal(t, "localhost", cfg.Cache.Host)
	require.Equal(t, "6379", cfg.Cache.Port)
	require.Equal(t, "mongodb", cfg.DocDB.Type)
	require.Equal(t, "mongodb://localhost:27017", cfg.DocDB.URI)
	require.Equal(t, "unifiedui", cfg.DocDB.Database)
	require.Equal(t, "dotenv", cfg.Vaults.VaultType)
	require.Equal(t, "info", cfg.Log.Level)
	require.Equal(t, "json", cfg.Log.Format)
}

func TestLoad_CustomEnv(t *testing.T) {
	t.Setenv("SERVER_HOST", "127.0.0.1")
	t.Setenv("SERVER_PORT", "9090")
	t.Setenv("GIN_MODE", "release")
	t.Setenv("REDIS_HOST", "redis.local")
	t.Setenv("MONGODB_URI", "mongodb://remote:27017")
	t.Setenv("MONGODB_DATABASE", "testdb")
	t.Setenv("VAULT_TYPE", "hashicorp")
	t.Setenv("LOG_LEVEL", "debug")

	cfg, err := config.Load()
	require.NoError(t, err)
	require.Equal(t, "127.0.0.1", cfg.Server.Host)
	require.Equal(t, 9090, cfg.Server.Port)
	require.Equal(t, "release", cfg.Server.GinMode)
	require.Equal(t, "redis.local", cfg.Cache.Host)
	require.Equal(t, "mongodb://remote:27017", cfg.DocDB.URI)
	require.Equal(t, "testdb", cfg.DocDB.Database)
	require.Equal(t, "hashicorp", cfg.Vaults.VaultType)
	require.Equal(t, "debug", cfg.Log.Level)
}

func TestLoad_InvalidPort(t *testing.T) {
	t.Setenv("SERVER_PORT", "notanumber")

	cfg, err := config.Load()
	require.NoError(t, err)
	require.Equal(t, 8080, cfg.Server.Port)
}

func TestServerConfig_Address(t *testing.T) {
	cfg := config.ServerConfig{Host: "0.0.0.0", Port: 8080}
	require.Equal(t, "0.0.0.0:8080", cfg.Address())
}

func TestVaultsConfig_ResolvedVaultTypes(t *testing.T) {
	v := config.VaultsConfig{VaultType: "dotenv"}
	require.Equal(t, "dotenv", v.ResolvedAppVaultType())
	require.Equal(t, "dotenv", v.ResolvedSecretsVaultType())

	v.AppVaultType = "hashicorp"
	v.SecretsVaultType = "azurekeyvault"
	require.Equal(t, "hashicorp", v.ResolvedAppVaultType())
	require.Equal(t, "azurekeyvault", v.ResolvedSecretsVaultType())
}

// Additional comprehensive tests for config

func TestLoad_CacheConfig(t *testing.T) {
	t.Setenv("CACHE_TYPE", "memcached")
	t.Setenv("REDIS_PORT", "6380")
	t.Setenv("REDIS_PASSWORD", "secretpass")
	t.Setenv("REDIS_DB", "5")
	t.Setenv("CACHE_TTL_SECONDS", "300")

	cfg, err := config.Load()
	require.NoError(t, err)
	require.Equal(t, "memcached", cfg.Cache.Type)
	require.Equal(t, "6380", cfg.Cache.Port)
	require.Equal(t, "secretpass", cfg.Cache.Password)
	require.Equal(t, 5, cfg.Cache.DB)
	require.Equal(t, 300*1000000000, int(cfg.Cache.TTL)) // 300 seconds in nanoseconds
}

func TestLoad_CacheConfig_InvalidDB(t *testing.T) {
	t.Setenv("REDIS_DB", "notanumber")

	cfg, err := config.Load()
	require.NoError(t, err)
	require.Equal(t, 0, cfg.Cache.DB) // Falls back to default
}

func TestLoad_CacheConfig_InvalidTTL(t *testing.T) {
	t.Setenv("CACHE_TTL_SECONDS", "invalid")

	cfg, err := config.Load()
	require.NoError(t, err)
	require.Equal(t, 180*1000000000, int(cfg.Cache.TTL)) // Falls back to default (180 seconds)
}

func TestLoad_PlatformConfig(t *testing.T) {
	t.Setenv("PLATFORM_SERVICE_URL", "http://platform:9000")
	t.Setenv("PLATFORM_CONFIG_PATH", "/custom/path/config.json")
	t.Setenv("PLATFORM_SERVICE_TIMEOUT_SECONDS", "60")
	t.Setenv("X_AGENT_SERVICE_KEY", "my-service-key")

	cfg, err := config.Load()
	require.NoError(t, err)
	require.Equal(t, "http://platform:9000", cfg.Platform.URL)
	require.Equal(t, "/custom/path/config.json", cfg.Platform.ConfigPath)
	require.Equal(t, 60*1000000000, int(cfg.Platform.Timeout)) // 60 seconds in nanoseconds
	require.Equal(t, "my-service-key", cfg.Platform.ServiceKey)
}

func TestLoad_PlatformConfig_InvalidTimeout(t *testing.T) {
	t.Setenv("PLATFORM_SERVICE_TIMEOUT_SECONDS", "invalid")

	cfg, err := config.Load()
	require.NoError(t, err)
	require.Equal(t, 30*1000000000, int(cfg.Platform.Timeout)) // Falls back to default
}

func TestLoad_AppVaultConfig(t *testing.T) {
	t.Setenv("APP_VAULT_PLATFORM_SERVICE_KEY", "CUSTOM_PLATFORM_KEY")
	t.Setenv("APP_VAULT_AGENT_TO_PLATFORM_KEY", "CUSTOM_AGENT_KEY")

	cfg, err := config.Load()
	require.NoError(t, err)
	require.Equal(t, "CUSTOM_PLATFORM_KEY", cfg.AppVault.PlatformServiceKey)
	require.Equal(t, "CUSTOM_AGENT_KEY", cfg.AppVault.AgentToPlatformKey)
}

func TestLoad_AppVaultConfig_Defaults(t *testing.T) {
	os.Unsetenv("APP_VAULT_PLATFORM_SERVICE_KEY")
	os.Unsetenv("APP_VAULT_AGENT_TO_PLATFORM_KEY")

	cfg, err := config.Load()
	require.NoError(t, err)
	require.Equal(t, "PLATFORM_TO_AGENT_SERVICE_KEY", cfg.AppVault.PlatformServiceKey)
	require.Equal(t, "AGENT_TO_PLATFORM_SERVICE_KEY", cfg.AppVault.AgentToPlatformKey)
}

func TestLoad_VaultsConfig(t *testing.T) {
	t.Setenv("APP_AZURE_KEYVAULT_URL", "https://myvault.vault.azure.net")
	t.Setenv("APP_HASHICORP_VAULT_ADDR", "http://vault:8200")
	t.Setenv("APP_HASHICORP_VAULT_TOKEN", "s.token123")
	t.Setenv("SECRETS_AZURE_KEYVAULT_URL", "https://secretsvault.vault.azure.net")
	t.Setenv("SECRETS_HASHICORP_VAULT_ADDR", "http://secretsvault:8200")
	t.Setenv("SECRETS_HASHICORP_VAULT_TOKEN", "s.secrettoken")
	t.Setenv("SECRETS_ENCRYPTION_KEY", "myencryptionkey")

	cfg, err := config.Load()
	require.NoError(t, err)
	require.Equal(t, "https://myvault.vault.azure.net", cfg.Vaults.App.AzureKeyVaultURL)
	require.Equal(t, "http://vault:8200", cfg.Vaults.App.HashiCorpAddr)
	require.Equal(t, "s.token123", cfg.Vaults.App.HashiCorpToken)
	require.Equal(t, "https://secretsvault.vault.azure.net", cfg.Vaults.Secrets.AzureKeyVaultURL)
	require.Equal(t, "http://secretsvault:8200", cfg.Vaults.Secrets.HashiCorpAddr)
	require.Equal(t, "s.secrettoken", cfg.Vaults.Secrets.HashiCorpToken)
	require.Equal(t, "myencryptionkey", cfg.Vaults.EncryptionKey)
}

func TestLoad_LogConfig(t *testing.T) {
	t.Setenv("LOG_LEVEL", "debug")
	t.Setenv("LOG_FORMAT", "text")

	cfg, err := config.Load()
	require.NoError(t, err)
	require.Equal(t, "debug", cfg.Log.Level)
	require.Equal(t, "text", cfg.Log.Format)
}

func TestServerConfig_Address_CustomValues(t *testing.T) {
	cfg := config.ServerConfig{Host: "192.168.1.100", Port: 3000}
	require.Equal(t, "192.168.1.100:3000", cfg.Address())
}

func TestServerConfig_Address_LocalhostIPv6(t *testing.T) {
	cfg := config.ServerConfig{Host: "::1", Port: 8080}
	require.Equal(t, "::1:8080", cfg.Address())
}

func TestVaultsConfig_ResolvedVaultTypes_OnlyAppVaultTypeSet(t *testing.T) {
	v := config.VaultsConfig{
		VaultType:    "dotenv",
		AppVaultType: "hashicorp",
	}
	require.Equal(t, "hashicorp", v.ResolvedAppVaultType())
	require.Equal(t, "dotenv", v.ResolvedSecretsVaultType()) // Falls back to VaultType
}

func TestVaultsConfig_ResolvedVaultTypes_OnlySecretsVaultTypeSet(t *testing.T) {
	v := config.VaultsConfig{
		VaultType:        "dotenv",
		SecretsVaultType: "azurekeyvault",
	}
	require.Equal(t, "dotenv", v.ResolvedAppVaultType()) // Falls back to VaultType
	require.Equal(t, "azurekeyvault", v.ResolvedSecretsVaultType())
}

func TestVaultsConfig_ResolvedVaultTypes_EmptyVaultType(t *testing.T) {
	v := config.VaultsConfig{VaultType: ""}
	require.Equal(t, "", v.ResolvedAppVaultType())
	require.Equal(t, "", v.ResolvedSecretsVaultType())
}

func TestLoad_AllDefaultValues(t *testing.T) {
	// Unset all environment variables to test all defaults
	envVars := []string{
		"SERVER_HOST", "SERVER_PORT", "GIN_MODE",
		"CACHE_TYPE", "REDIS_HOST", "REDIS_PORT", "REDIS_PASSWORD", "REDIS_DB", "CACHE_TTL_SECONDS",
		"DOCDB_TYPE", "MONGODB_URI", "MONGODB_DATABASE",
		"VAULT_TYPE", "APP_VAULT_TYPE", "SECRETS_VAULT_TYPE",
		"APP_AZURE_KEYVAULT_URL", "APP_HASHICORP_VAULT_ADDR", "APP_HASHICORP_VAULT_TOKEN",
		"SECRETS_AZURE_KEYVAULT_URL", "SECRETS_HASHICORP_VAULT_ADDR", "SECRETS_HASHICORP_VAULT_TOKEN",
		"SECRETS_ENCRYPTION_KEY",
		"PLATFORM_SERVICE_URL", "PLATFORM_CONFIG_PATH", "PLATFORM_SERVICE_TIMEOUT_SECONDS", "X_AGENT_SERVICE_KEY",
		"APP_VAULT_PLATFORM_SERVICE_KEY", "APP_VAULT_AGENT_TO_PLATFORM_KEY",
		"LOG_LEVEL", "LOG_FORMAT",
	}
	for _, key := range envVars {
		os.Unsetenv(key)
	}

	cfg, err := config.Load()
	require.NoError(t, err)
	require.NotNil(t, cfg)

	// Server defaults
	require.Equal(t, "0.0.0.0", cfg.Server.Host)
	require.Equal(t, 8080, cfg.Server.Port)
	require.Equal(t, "debug", cfg.Server.GinMode)

	// Cache defaults
	require.Equal(t, "redis", cfg.Cache.Type)
	require.Equal(t, "localhost", cfg.Cache.Host)
	require.Equal(t, "6379", cfg.Cache.Port)
	require.Equal(t, "", cfg.Cache.Password)
	require.Equal(t, 0, cfg.Cache.DB)
	require.Equal(t, 180*1000000000, int(cfg.Cache.TTL))

	// DocDB defaults
	require.Equal(t, "mongodb", cfg.DocDB.Type)
	require.Equal(t, "mongodb://localhost:27017", cfg.DocDB.URI)
	require.Equal(t, "unifiedui", cfg.DocDB.Database)

	// Vaults defaults
	require.Equal(t, "dotenv", cfg.Vaults.VaultType)
	require.Equal(t, "", cfg.Vaults.AppVaultType)
	require.Equal(t, "", cfg.Vaults.SecretsVaultType)
	require.Equal(t, "", cfg.Vaults.EncryptionKey)

	// Platform defaults
	require.Equal(t, "http://localhost:8081", cfg.Platform.URL)
	require.Equal(t, "poc/n8n/config.json", cfg.Platform.ConfigPath)
	require.Equal(t, 30*1000000000, int(cfg.Platform.Timeout))
	require.Equal(t, "", cfg.Platform.ServiceKey)

	// AppVault defaults
	require.Equal(t, "PLATFORM_TO_AGENT_SERVICE_KEY", cfg.AppVault.PlatformServiceKey)
	require.Equal(t, "AGENT_TO_PLATFORM_SERVICE_KEY", cfg.AppVault.AgentToPlatformKey)

	// Log defaults
	require.Equal(t, "info", cfg.Log.Level)
	require.Equal(t, "json", cfg.Log.Format)
}
