// Package dotenv provides a dotenv-based vault implementation for development.
package dotenv

import (
	"context"
	"fmt"
	"os"
	"strings"
	"sync"
)

// Vault implements the vault.Vault interface using environment variables.
// This is primarily for local development and testing.
type Vault struct {
	// secrets stores in-memory secrets (for secrets not in env vars)
	secrets map[string]string
	mu      sync.RWMutex
}

// NewVault creates a new DotEnv vault instance.
func NewVault() *Vault {
	return &Vault{
		secrets: make(map[string]string),
	}
}

// StoreSecret stores a secret in memory.
// Returns a URI in the format "dotenv://{key}".
func (v *Vault) StoreSecret(ctx context.Context, key string, value string, metadata map[string]string) (string, error) {
	v.mu.Lock()
	defer v.mu.Unlock()

	v.secrets[key] = value
	return fmt.Sprintf("dotenv://%s", key), nil
}

// GetSecret retrieves a secret from environment variables or in-memory store.
func (v *Vault) GetSecret(ctx context.Context, uri string) (string, error) {
	key := strings.TrimPrefix(uri, "dotenv://")

	// First check environment variables
	if value := os.Getenv(key); value != "" {
		return value, nil
	}

	// Then check in-memory store
	v.mu.RLock()
	defer v.mu.RUnlock()

	if value, ok := v.secrets[key]; ok {
		return value, nil
	}

	return "", fmt.Errorf("secret not found: %s", key)
}

// UpdateSecret updates a secret in memory.
func (v *Vault) UpdateSecret(ctx context.Context, uri string, value string, metadata map[string]string) (bool, error) {
	key := strings.TrimPrefix(uri, "dotenv://")

	v.mu.Lock()
	defer v.mu.Unlock()

	v.secrets[key] = value
	return true, nil
}

// DeleteSecret deletes a secret from memory.
func (v *Vault) DeleteSecret(ctx context.Context, uri string) (bool, error) {
	key := strings.TrimPrefix(uri, "dotenv://")

	v.mu.Lock()
	defer v.mu.Unlock()

	if _, ok := v.secrets[key]; ok {
		delete(v.secrets, key)
		return true, nil
	}

	return false, nil
}

// Ping checks if the vault is available (always returns nil for dotenv).
func (v *Vault) Ping(ctx context.Context) error {
	return nil
}

// Close closes the vault (no-op for dotenv).
func (v *Vault) Close() error {
	return nil
}
