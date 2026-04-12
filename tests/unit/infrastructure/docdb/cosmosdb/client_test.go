// Package cosmosdb_test provides unit tests for the CosmosDB implementation.
package cosmosdb_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/unifiedui/agent-service/internal/infrastructure/docdb/cosmosdb"
)

func TestNewClient_NilConfig(t *testing.T) {
	client, err := cosmosdb.NewClient(context.TODO(), nil)

	assert.Nil(t, client)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "config cannot be nil")
}

func TestNewClient_MissingEndpoint(t *testing.T) {
	config := &cosmosdb.ClientConfig{
		DatabaseName: "testdb",
	}

	client, err := cosmosdb.NewClient(context.TODO(), config)

	assert.Nil(t, client)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "cosmosdb endpoint is required")
}

func TestNewClient_MissingDatabaseName(t *testing.T) {
	config := &cosmosdb.ClientConfig{
		Endpoint: "https://test.documents.azure.com:443/",
	}

	client, err := cosmosdb.NewClient(context.TODO(), config)

	assert.Nil(t, client)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "database name is required")
}

func TestClientConfig_Fields(t *testing.T) {
	config := &cosmosdb.ClientConfig{
		Endpoint:           "https://test.documents.azure.com:443/",
		Key:                "test-key",
		DatabaseName:       "testdb",
		UseManagedIdentity: true,
	}

	assert.Equal(t, "https://test.documents.azure.com:443/", config.Endpoint)
	assert.Equal(t, "test-key", config.Key)
	assert.Equal(t, "testdb", config.DatabaseName)
	assert.True(t, config.UseManagedIdentity)
}

func TestClientConfig_DefaultManagedIdentity(t *testing.T) {
	config := &cosmosdb.ClientConfig{
		Endpoint:     "https://test.documents.azure.com:443/",
		DatabaseName: "testdb",
	}

	assert.False(t, config.UseManagedIdentity)
	assert.Empty(t, config.Key)
}

func TestContainerNames(t *testing.T) {
	assert.Equal(t, "messages", cosmosdb.MessagesContainerName)
	assert.Equal(t, "traces", cosmosdb.TracesContainerName)
	assert.Equal(t, "reactions", cosmosdb.ReactionsContainerName)
	assert.Equal(t, "sessions", cosmosdb.SessionsContainerName)
	assert.Equal(t, "/tenantId", cosmosdb.DefaultPartitionKeyPath)
}

func TestErrNotFound(t *testing.T) {
	assert.NotNil(t, cosmosdb.ErrNotFound)
	assert.Equal(t, "document not found", cosmosdb.ErrNotFound.Error())
}
