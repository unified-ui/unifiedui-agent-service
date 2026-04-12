// Package cosmosdb provides Azure CosmosDB NoSQL client implementation.
package cosmosdb

import (
	"context"
	"errors"
	"fmt"

	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
	"github.com/Azure/azure-sdk-for-go/sdk/data/azcosmos"

	"github.com/unifiedui/agent-service/internal/core/docdb"
)

// Client implements the docdb.Client interface for Azure CosmosDB NoSQL.
type Client struct {
	cosmosClient        *azcosmos.Client
	database            *Database
	messagesCollection  *MessagesCollection
	tracesCollection    *TracesCollection
	reactionsCollection *ReactionsCollection
	databaseName        string
}

// ClientConfig holds CosmosDB connection configuration.
type ClientConfig struct {
	Endpoint           string
	Key                string // Optional: If empty, uses managed identity
	DatabaseName       string
	UseManagedIdentity bool
}

// NewClient creates a new CosmosDB NoSQL client.
func NewClient(_ context.Context, config *ClientConfig) (*Client, error) {
	if config == nil {
		return nil, fmt.Errorf("config cannot be nil")
	}
	if config.Endpoint == "" {
		return nil, fmt.Errorf("cosmosdb endpoint is required")
	}
	if config.DatabaseName == "" {
		return nil, fmt.Errorf("database name is required")
	}

	var cosmosClient *azcosmos.Client
	var err error

	if config.UseManagedIdentity || config.Key == "" {
		// Use DefaultAzureCredential for managed identity
		cred, credErr := azidentity.NewDefaultAzureCredential(nil)
		if credErr != nil {
			return nil, fmt.Errorf("failed to create Azure credential: %w", credErr)
		}
		cosmosClient, err = azcosmos.NewClient(config.Endpoint, cred, nil)
	} else {
		// Use key-based authentication
		cred, credErr := azcosmos.NewKeyCredential(config.Key)
		if credErr != nil {
			return nil, fmt.Errorf("failed to create key credential: %w", credErr)
		}
		cosmosClient, err = azcosmos.NewClientWithKey(config.Endpoint, cred, nil)
	}

	if err != nil {
		return nil, fmt.Errorf("failed to create cosmosdb client: %w", err)
	}

	// Get database and container clients
	databaseClient, err := cosmosClient.NewDatabase(config.DatabaseName)
	if err != nil {
		return nil, fmt.Errorf("failed to get database client: %w", err)
	}

	database := NewDatabase(cosmosClient, databaseClient, config.DatabaseName)
	messagesCollection := NewMessagesCollection(databaseClient)
	tracesCollection := NewTracesCollection(databaseClient)
	reactionsCollection := NewReactionsCollection(databaseClient)

	return &Client{
		cosmosClient:        cosmosClient,
		database:            database,
		messagesCollection:  messagesCollection,
		tracesCollection:    tracesCollection,
		reactionsCollection: reactionsCollection,
		databaseName:        config.DatabaseName,
	}, nil
}

// Database returns the database interface.
func (c *Client) Database() docdb.Database {
	return c.database
}

// Messages returns the typed messages collection with domain methods.
func (c *Client) Messages() docdb.MessagesCollection {
	return c.messagesCollection
}

// MessagesRaw returns the raw messages collection for direct operations.
func (c *Client) MessagesRaw() docdb.Collection {
	return c.database.Collection(MessagesContainerName)
}

// Reactions returns the typed reactions collection with domain methods.
func (c *Client) Reactions() docdb.ReactionsCollection {
	return c.reactionsCollection
}

// Traces returns the typed traces collection with domain methods.
func (c *Client) Traces() docdb.TracesCollection {
	return c.tracesCollection
}

// TracesRaw returns the raw traces collection for direct operations.
func (c *Client) TracesRaw() docdb.Collection {
	return c.database.Collection(TracesContainerName)
}

// Ping verifies the connection to CosmosDB.
func (c *Client) Ping(_ context.Context) error {
	// CosmosDB doesn't have a direct ping, but we can try to read database properties
	_, err := c.cosmosClient.NewDatabase(c.databaseName)
	if err != nil {
		return fmt.Errorf("cosmosdb ping failed: %w", err)
	}
	return nil
}

// Close closes the CosmosDB connection.
func (c *Client) Close(_ context.Context) error {
	// CosmosDB SDK doesn't require explicit close
	return nil
}

// EnsureIndexes creates all necessary indexes for all collections.
// Note: CosmosDB NoSQL creates indexes via indexing policy during container creation.
// This method is provided for interface compatibility but indexes should be managed via Terraform.
func (c *Client) EnsureIndexes(ctx context.Context) error {
	// CosmosDB automatically indexes all properties by default
	// Custom composite indexes or excluded paths should be configured via Terraform
	// For compatibility, we call the collection methods which may perform validation
	if err := c.messagesCollection.EnsureIndexes(ctx); err != nil {
		return fmt.Errorf("failed to ensure messages indexes: %w", err)
	}
	if err := c.tracesCollection.EnsureIndexes(ctx); err != nil {
		return fmt.Errorf("failed to ensure traces indexes: %w", err)
	}
	if err := c.reactionsCollection.EnsureIndexes(ctx); err != nil {
		return fmt.Errorf("failed to ensure reactions indexes: %w", err)
	}
	return nil
}

// ErrNotFound is returned when a document is not found.
var ErrNotFound = errors.New("document not found")
