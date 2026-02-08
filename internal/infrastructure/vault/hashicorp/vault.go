package hashicorp

import (
	"context"
	"fmt"
	"strings"

	vaultapi "github.com/hashicorp/vault/api"
)

// Vault implements the vault.Vault interface using HashiCorp Vault KV v2.
type Vault struct {
	client     *vaultapi.Client
	mountPoint string
	host       string
}

// VaultConfig holds the configuration for HashiCorp Vault.
type VaultConfig struct {
	Address    string
	Token      string
	MountPoint string
}

// NewVault creates a new HashiCorp Vault instance.
func NewVault(cfg *VaultConfig) (*Vault, error) {
	if cfg.Address == "" {
		return nil, fmt.Errorf("vault address is required")
	}
	if cfg.Token == "" {
		return nil, fmt.Errorf("vault token is required")
	}

	mountPoint := cfg.MountPoint
	if mountPoint == "" {
		mountPoint = "secret"
	}

	apiCfg := vaultapi.DefaultConfig()
	apiCfg.Address = cfg.Address

	client, err := vaultapi.NewClient(apiCfg)
	if err != nil {
		return nil, fmt.Errorf("failed to create vault client: %w", err)
	}

	client.SetToken(cfg.Token)

	host := strings.TrimPrefix(cfg.Address, "http://")
	host = strings.TrimPrefix(host, "https://")

	return &Vault{
		client:     client,
		mountPoint: mountPoint,
		host:       host,
	}, nil
}

// BuildSecretURI builds a HashiCorp vault URI for the given key name.
func (v *Vault) BuildSecretURI(keyName string) string {
	return fmt.Sprintf("vault://%s/%s/%s", v.host, v.mountPoint, keyName)
}

// StoreSecret stores a secret in HashiCorp Vault KV v2.
func (v *Vault) StoreSecret(ctx context.Context, key string, value string, metadata map[string]string) (string, error) {
	data := map[string]interface{}{
		"data": map[string]interface{}{
			"value": value,
		},
	}

	path := fmt.Sprintf("%s/data/%s", v.mountPoint, key)

	_, err := v.client.Logical().WriteWithContext(ctx, path, data)
	if err != nil {
		return "", fmt.Errorf("failed to store secret: %w", err)
	}

	return v.BuildSecretURI(key), nil
}

// GetSecret retrieves a secret from HashiCorp Vault KV v2.
func (v *Vault) GetSecret(ctx context.Context, uri string) (string, error) {
	mount, secretPath, err := v.parseURI(uri)
	if err != nil {
		return "", err
	}

	path := fmt.Sprintf("%s/data/%s", mount, secretPath)

	secret, err := v.client.Logical().ReadWithContext(ctx, path)
	if err != nil {
		return "", fmt.Errorf("failed to read secret: %w", err)
	}

	if secret == nil || secret.Data == nil {
		return "", fmt.Errorf("secret not found: %s", uri)
	}

	data, ok := secret.Data["data"].(map[string]interface{})
	if !ok {
		return "", fmt.Errorf("unexpected secret data format: %s", uri)
	}

	value, ok := data["value"].(string)
	if !ok {
		return "", fmt.Errorf("secret value is not a string: %s", uri)
	}

	return value, nil
}

// UpdateSecret updates an existing secret in HashiCorp Vault KV v2.
func (v *Vault) UpdateSecret(ctx context.Context, uri string, value string, metadata map[string]string) (bool, error) {
	mount, secretPath, err := v.parseURI(uri)
	if err != nil {
		return false, err
	}

	data := map[string]interface{}{
		"data": map[string]interface{}{
			"value": value,
		},
	}

	path := fmt.Sprintf("%s/data/%s", mount, secretPath)

	_, err = v.client.Logical().WriteWithContext(ctx, path, data)
	if err != nil {
		return false, fmt.Errorf("failed to update secret: %w", err)
	}

	return true, nil
}

// DeleteSecret deletes a secret from HashiCorp Vault KV v2.
func (v *Vault) DeleteSecret(ctx context.Context, uri string) (bool, error) {
	mount, secretPath, err := v.parseURI(uri)
	if err != nil {
		return false, err
	}

	path := fmt.Sprintf("%s/data/%s", mount, secretPath)

	_, err = v.client.Logical().DeleteWithContext(ctx, path)
	if err != nil {
		return false, fmt.Errorf("failed to delete secret: %w", err)
	}

	return true, nil
}

// Ping checks if the vault connection is alive.
func (v *Vault) Ping(ctx context.Context) error {
	health, err := v.client.Sys().HealthWithContext(ctx)
	if err != nil {
		return fmt.Errorf("vault health check failed: %w", err)
	}

	if health.Sealed {
		return fmt.Errorf("vault is sealed")
	}

	return nil
}

// Close closes the vault connection (no-op for HTTP client).
func (v *Vault) Close() error {
	return nil
}

func (v *Vault) parseURI(uri string) (string, string, error) {
	stripped := strings.TrimPrefix(uri, "vault://")
	parts := strings.SplitN(stripped, "/", 3)

	if len(parts) < 3 {
		return "", "", fmt.Errorf("invalid vault URI: %s (expected vault://host/mount/path)", uri)
	}

	return parts[1], parts[2], nil
}
