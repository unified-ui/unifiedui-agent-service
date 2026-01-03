// Package vault defines the vault client interface.
package vault

import (
	"context"
)

// Client is a higher-level vault client that wraps the Vault interface.
// It provides additional functionality like encrypted caching.
type Client interface {
	// GetVault returns the underlying Vault implementation.
	GetVault() Vault

	// StoreSecret stores a secret in the vault.
	StoreSecret(ctx context.Context, key string, value string, metadata map[string]string) (string, error)

	// GetSecret retrieves a secret from the vault.
	// If useCache is true and caching is available, it will use the cache.
	GetSecret(ctx context.Context, uri string, useCache bool) (string, error)

	// UpdateSecret updates an existing secret.
	UpdateSecret(ctx context.Context, uri string, value string, metadata map[string]string) (bool, error)

	// DeleteSecret deletes a secret from the vault.
	DeleteSecret(ctx context.Context, uri string) (bool, error)

	// Ping checks if the vault connection is alive.
	Ping(ctx context.Context) error

	// Close closes the vault client connection.
	Close() error
}
