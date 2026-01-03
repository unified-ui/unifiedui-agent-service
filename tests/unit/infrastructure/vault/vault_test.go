// Package vault_test provides unit tests for the vault package.
package vault_test

import (
	"context"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/unifiedui/chat-service/internal/infrastructure/vault/dotenv"
)

func TestDotEnvVault_StoreAndGetSecret(t *testing.T) {
	client, err := dotenv.NewClient()
	require.NoError(t, err)

	ctx := context.Background()

	// Store secret
	uri, err := client.StoreSecret(ctx, "test-secret", "secret-value", nil)
	assert.NoError(t, err)
	assert.Equal(t, "dotenv://test-secret", uri)

	// Get secret
	value, err := client.GetSecret(ctx, uri, false)
	assert.NoError(t, err)
	assert.Equal(t, "secret-value", value)
}

func TestDotEnvVault_GetSecretFromEnv(t *testing.T) {
	// Set environment variable
	os.Setenv("TEST_ENV_SECRET", "env-secret-value")
	defer os.Unsetenv("TEST_ENV_SECRET")

	client, err := dotenv.NewClient()
	require.NoError(t, err)

	ctx := context.Background()

	// Get secret from env
	value, err := client.GetSecret(ctx, "dotenv://TEST_ENV_SECRET", false)
	assert.NoError(t, err)
	assert.Equal(t, "env-secret-value", value)
}

func TestDotEnvVault_GetSecretNotFound(t *testing.T) {
	client, err := dotenv.NewClient()
	require.NoError(t, err)

	ctx := context.Background()

	value, err := client.GetSecret(ctx, "dotenv://non-existent", false)
	assert.Error(t, err)
	assert.Empty(t, value)
	assert.Contains(t, err.Error(), "secret not found")
}

func TestDotEnvVault_UpdateSecret(t *testing.T) {
	client, err := dotenv.NewClient()
	require.NoError(t, err)

	ctx := context.Background()

	// Store initial secret
	uri, err := client.StoreSecret(ctx, "update-test", "initial-value", nil)
	require.NoError(t, err)

	// Update secret
	updated, err := client.UpdateSecret(ctx, uri, "updated-value", nil)
	assert.NoError(t, err)
	assert.True(t, updated)

	// Verify update
	value, err := client.GetSecret(ctx, uri, false)
	assert.NoError(t, err)
	assert.Equal(t, "updated-value", value)
}

func TestDotEnvVault_DeleteSecret(t *testing.T) {
	client, err := dotenv.NewClient()
	require.NoError(t, err)

	ctx := context.Background()

	// Store secret
	uri, err := client.StoreSecret(ctx, "delete-test", "delete-value", nil)
	require.NoError(t, err)

	// Delete secret
	deleted, err := client.DeleteSecret(ctx, uri)
	assert.NoError(t, err)
	assert.True(t, deleted)

	// Verify deleted
	value, err := client.GetSecret(ctx, uri, false)
	assert.Error(t, err)
	assert.Empty(t, value)
}

func TestDotEnvVault_DeleteNonExistent(t *testing.T) {
	client, err := dotenv.NewClient()
	require.NoError(t, err)

	ctx := context.Background()

	deleted, err := client.DeleteSecret(ctx, "dotenv://non-existent")
	assert.NoError(t, err)
	assert.False(t, deleted)
}

func TestDotEnvVault_Ping(t *testing.T) {
	client, err := dotenv.NewClient()
	require.NoError(t, err)

	ctx := context.Background()

	err = client.Ping(ctx)
	assert.NoError(t, err)
}
