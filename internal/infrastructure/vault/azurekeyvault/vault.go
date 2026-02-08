package azurekeyvault

import (
	"context"
	"fmt"
	"strings"

	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
	"github.com/Azure/azure-sdk-for-go/sdk/security/keyvault/azsecrets"
)

// Vault implements the vault.Vault interface using Azure Key Vault.
type Vault struct {
	client    *azsecrets.Client
	vaultName string
	vaultURL  string
}

// VaultConfig holds the configuration for Azure Key Vault.
type VaultConfig struct {
	VaultURL string
}

// NewVault creates a new Azure Key Vault instance.
func NewVault(cfg *VaultConfig) (*Vault, error) {
	if cfg.VaultURL == "" {
		return nil, fmt.Errorf("azure key vault URL is required")
	}

	credential, err := azidentity.NewDefaultAzureCredential(nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create Azure credential: %w", err)
	}

	client, err := azsecrets.NewClient(cfg.VaultURL, credential, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create Azure Key Vault client: %w", err)
	}

	vaultName := extractVaultName(cfg.VaultURL)

	return &Vault{
		client:    client,
		vaultName: vaultName,
		vaultURL:  cfg.VaultURL,
	}, nil
}

// BuildSecretURI builds an Azure Key Vault URI for the given key name.
func (v *Vault) BuildSecretURI(keyName string) string {
	secretName := toAzureSecretName(keyName)
	return fmt.Sprintf("azurekv://%s/%s", v.vaultName, secretName)
}

// StoreSecret stores a secret in Azure Key Vault.
func (v *Vault) StoreSecret(ctx context.Context, key string, value string, metadata map[string]string) (string, error) {
	secretName := toAzureSecretName(key)

	params := azsecrets.SetSecretParameters{
		Value: &value,
	}

	if len(metadata) > 0 {
		tags := make(map[string]*string)
		for k, val := range metadata {
			v := val
			tags[k] = &v
		}
		params.Tags = tags
	}

	resp, err := v.client.SetSecret(ctx, secretName, params, nil)
	if err != nil {
		return "", fmt.Errorf("failed to store secret in Azure Key Vault: %w", err)
	}

	version := ""
	if resp.ID != nil {
		version = extractVersion(string(*resp.ID))
	}

	uri := fmt.Sprintf("azurekv://%s/%s", v.vaultName, secretName)
	if version != "" {
		uri = fmt.Sprintf("%s/%s", uri, version)
	}

	return uri, nil
}

// GetSecret retrieves a secret from Azure Key Vault.
func (v *Vault) GetSecret(ctx context.Context, uri string) (string, error) {
	secretName, version, err := v.parseURI(uri)
	if err != nil {
		return "", err
	}

	resp, err := v.client.GetSecret(ctx, secretName, version, nil)
	if err != nil {
		return "", fmt.Errorf("failed to get secret from Azure Key Vault: %w", err)
	}

	if resp.Value == nil {
		return "", fmt.Errorf("secret value is nil: %s", uri)
	}

	return *resp.Value, nil
}

// UpdateSecret updates an existing secret in Azure Key Vault.
func (v *Vault) UpdateSecret(ctx context.Context, uri string, value string, metadata map[string]string) (bool, error) {
	secretName, _, err := v.parseURI(uri)
	if err != nil {
		return false, err
	}

	params := azsecrets.SetSecretParameters{
		Value: &value,
	}

	if len(metadata) > 0 {
		tags := make(map[string]*string)
		for k, val := range metadata {
			v := val
			tags[k] = &v
		}
		params.Tags = tags
	}

	_, err = v.client.SetSecret(ctx, secretName, params, nil)
	if err != nil {
		return false, fmt.Errorf("failed to update secret in Azure Key Vault: %w", err)
	}

	return true, nil
}

// DeleteSecret deletes a secret from Azure Key Vault.
func (v *Vault) DeleteSecret(ctx context.Context, uri string) (bool, error) {
	secretName, _, err := v.parseURI(uri)
	if err != nil {
		return false, err
	}

	_, err = v.client.DeleteSecret(ctx, secretName, nil)
	if err != nil {
		return false, fmt.Errorf("failed to delete secret from Azure Key Vault: %w", err)
	}

	return true, nil
}

// Ping checks if the Azure Key Vault connection is alive by listing secrets (max 1).
func (v *Vault) Ping(ctx context.Context) error {
	pager := v.client.NewListSecretPropertiesPager(nil)

	if pager.More() {
		_, err := pager.NextPage(ctx)
		if err != nil {
			return fmt.Errorf("azure key vault health check failed: %w", err)
		}
	}

	return nil
}

// Close closes the vault connection (no-op for Azure SDK HTTP client).
func (v *Vault) Close() error {
	return nil
}

func (v *Vault) parseURI(uri string) (string, string, error) {
	stripped := strings.TrimPrefix(uri, "azurekv://")
	parts := strings.SplitN(stripped, "/", 3)

	if len(parts) < 2 {
		return "", "", fmt.Errorf("invalid Azure Key Vault URI: %s (expected azurekv://vaultname/secretname[/version])", uri)
	}

	secretName := parts[1]
	version := ""
	if len(parts) > 2 {
		version = parts[2]
	}

	return secretName, version, nil
}

func extractVaultName(vaultURL string) string {
	stripped := strings.TrimPrefix(vaultURL, "https://")
	stripped = strings.TrimPrefix(stripped, "http://")
	parts := strings.SplitN(stripped, ".", 2)
	return parts[0]
}

func extractVersion(secretID string) string {
	parts := strings.Split(secretID, "/")
	if len(parts) > 0 {
		return parts[len(parts)-1]
	}
	return ""
}

func toAzureSecretName(key string) string {
	name := strings.ReplaceAll(key, "_", "-")
	name = strings.ReplaceAll(name, ".", "-")
	return name
}
