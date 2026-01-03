// Package dotenv provides the dotenv vault client implementation.
package dotenv

import (
	"context"

	"github.com/unifiedui/chat-service/internal/core/vault"
)

// Client implements the vault.Client interface for DotEnv.
type Client struct {
	vault *Vault
}

// NewClient creates a new DotEnv vault client.
func NewClient() (*Client, error) {
	return &Client{
		vault: NewVault(),
	}, nil
}

// GetVault returns the underlying Vault implementation.
func (c *Client) GetVault() vault.Vault {
	return c.vault
}

// StoreSecret stores a secret in the vault.
func (c *Client) StoreSecret(ctx context.Context, key string, value string, metadata map[string]string) (string, error) {
	return c.vault.StoreSecret(ctx, key, value, metadata)
}

// GetSecret retrieves a secret from the vault.
func (c *Client) GetSecret(ctx context.Context, uri string, useCache bool) (string, error) {
	// DotEnv vault doesn't support caching
	return c.vault.GetSecret(ctx, uri)
}

// UpdateSecret updates an existing secret.
func (c *Client) UpdateSecret(ctx context.Context, uri string, value string, metadata map[string]string) (bool, error) {
	return c.vault.UpdateSecret(ctx, uri, value, metadata)
}

// DeleteSecret deletes a secret from the vault.
func (c *Client) DeleteSecret(ctx context.Context, uri string) (bool, error) {
	return c.vault.DeleteSecret(ctx, uri)
}

// Ping checks if the vault connection is alive.
func (c *Client) Ping(ctx context.Context) error {
	return c.vault.Ping(ctx)
}

// Close closes the vault client connection.
func (c *Client) Close() error {
	return c.vault.Close()
}
