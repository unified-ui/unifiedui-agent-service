package vault_test

import (
	"context"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/unifiedui/agent-service/internal/infrastructure/vault/hashicorp"
)

func getHashiCorpTestClient(t *testing.T) *hashicorp.Client {
	t.Helper()

	addr := os.Getenv("VAULT_ADDR")
	token := os.Getenv("VAULT_TOKEN")

	if addr == "" {
		addr = "http://localhost:8200"
	}
	if token == "" {
		token = "admin"
	}

	client, err := hashicorp.NewClient(&hashicorp.VaultConfig{
		Address:    addr,
		Token:      token,
		MountPoint: "secret",
	})
	if err != nil {
		t.Skipf("skipping HashiCorp vault test: %v", err)
	}

	if err := client.Ping(context.Background()); err != nil {
		t.Skipf("skipping HashiCorp vault test (vault not reachable): %v", err)
	}

	return client
}

func TestHashiCorpVault_BuildSecretURI(t *testing.T) {
	client := getHashiCorpTestClient(t)
	uri := client.BuildSecretURI("MY_SECRET_KEY")
	assert.Contains(t, uri, "vault://")
	assert.Contains(t, uri, "/secret/MY_SECRET_KEY")
}

func TestHashiCorpVault_StoreAndGetSecret(t *testing.T) {
	client := getHashiCorpTestClient(t)
	ctx := context.Background()

	uri, err := client.StoreSecret(ctx, "test/store-get", "test-secret-value", nil)
	require.NoError(t, err)
	assert.Contains(t, uri, "vault://")

	value, err := client.GetSecret(ctx, uri, false)
	require.NoError(t, err)
	assert.Equal(t, "test-secret-value", value)

	_, _ = client.DeleteSecret(ctx, uri)
}

func TestHashiCorpVault_UpdateSecret(t *testing.T) {
	client := getHashiCorpTestClient(t)
	ctx := context.Background()

	uri, err := client.StoreSecret(ctx, "test/update", "initial-value", nil)
	require.NoError(t, err)

	ok, err := client.UpdateSecret(ctx, uri, "updated-value", nil)
	require.NoError(t, err)
	assert.True(t, ok)

	value, err := client.GetSecret(ctx, uri, false)
	require.NoError(t, err)
	assert.Equal(t, "updated-value", value)

	_, _ = client.DeleteSecret(ctx, uri)
}

func TestHashiCorpVault_DeleteSecret(t *testing.T) {
	client := getHashiCorpTestClient(t)
	ctx := context.Background()

	uri, err := client.StoreSecret(ctx, "test/delete", "to-be-deleted", nil)
	require.NoError(t, err)

	ok, err := client.DeleteSecret(ctx, uri)
	require.NoError(t, err)
	assert.True(t, ok)
}

func TestHashiCorpVault_GetExistingServiceKeys(t *testing.T) {
	client := getHashiCorpTestClient(t)
	ctx := context.Background()

	uri := client.BuildSecretURI("PLATFORM_TO_AGENT_SERVICE_KEY")
	value, err := client.GetSecret(ctx, uri, false)
	require.NoError(t, err)
	assert.Equal(t, "147155ca38e356265f9627044db2401f802891da3b509ca7f93120e27607e734", value)

	uri2 := client.BuildSecretURI("AGENT_TO_PLATFORM_SERVICE_KEY")
	value2, err := client.GetSecret(ctx, uri2, false)
	require.NoError(t, err)
	assert.Equal(t, "147155ca38e356265f9627044db2401f802891da3b509ca7f93120e27607e734", value2)
}

func TestHashiCorpVault_Ping(t *testing.T) {
	client := getHashiCorpTestClient(t)
	err := client.Ping(context.Background())
	assert.NoError(t, err)
}

func TestHashiCorpVault_NewClientMissingAddress(t *testing.T) {
	_, err := hashicorp.NewClient(&hashicorp.VaultConfig{
		Address: "",
		Token:   "token",
	})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "address is required")
}

func TestHashiCorpVault_NewClientMissingToken(t *testing.T) {
	_, err := hashicorp.NewClient(&hashicorp.VaultConfig{
		Address: "http://localhost:8200",
		Token:   "",
	})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "token is required")
}

func TestHashiCorpVault_GetSecretInvalidURI(t *testing.T) {
	client := getHashiCorpTestClient(t)
	ctx := context.Background()

	_, err := client.GetSecret(ctx, "invalid-uri", false)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid vault URI")
}
