// Package vault defines the vault interface for secrets management.
package vault

import (
	"context"
)

// Vault defines the interface for vault/secrets operations.
type Vault interface {
	// StoreSecret stores a secret in the vault.
	// Returns the URI/reference to the stored secret.
	StoreSecret(ctx context.Context, key string, value string, metadata map[string]string) (string, error)

	// GetSecret retrieves a secret from the vault by URI.
	// Returns the secret value or an error if not found.
	GetSecret(ctx context.Context, uri string) (string, error)

	// UpdateSecret updates an existing secret in the vault.
	// Returns true if updated successfully.
	UpdateSecret(ctx context.Context, uri string, value string, metadata map[string]string) (bool, error)

	// DeleteSecret deletes a secret from the vault.
	// Returns true if deleted successfully.
	DeleteSecret(ctx context.Context, uri string) (bool, error)

	// Ping checks if the vault connection is alive.
	Ping(ctx context.Context) error

	// Close closes the vault connection.
	Close() error
}
