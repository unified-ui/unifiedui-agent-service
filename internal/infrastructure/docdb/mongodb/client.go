// Package mongodb provides MongoDB client implementation.
package mongodb

import (
	"context"
	"fmt"

	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

	"github.com/unifiedui/agent-service/internal/core/docdb"
)

const (
	// TracesCollection is the name of the traces collection.
	TracesCollection = "traces"
)

// Client implements the docdb.Client interface for MongoDB.
type Client struct {
	client             *mongo.Client
	database           *Database
	messagesCollection *MessagesCollection
}

// ClientConfig holds MongoDB connection configuration.
type ClientConfig struct {
	URI          string
	DatabaseName string
}

// NewClient creates a new MongoDB client.
func NewClient(ctx context.Context, config *ClientConfig) (*Client, error) {
	if config == nil {
		return nil, fmt.Errorf("config cannot be nil")
	}
	if config.URI == "" {
		return nil, fmt.Errorf("mongodb URI is required")
	}
	if config.DatabaseName == "" {
		return nil, fmt.Errorf("database name is required")
	}

	clientOpts := options.Client().ApplyURI(config.URI)
	client, err := mongo.Connect(ctx, clientOpts)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to mongodb: %w", err)
	}

	// Verify connection
	if err := client.Ping(ctx, nil); err != nil {
		return nil, fmt.Errorf("failed to ping mongodb: %w", err)
	}

	db := client.Database(config.DatabaseName)
	database := NewDatabase(db)
	messagesCollection := NewMessagesCollection(db)

	return &Client{
		client:             client,
		database:           database,
		messagesCollection: messagesCollection,
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
	return c.database.Collection(MessagesCollectionName)
}

// Traces returns the traces collection.
func (c *Client) Traces() docdb.Collection {
	return c.database.Collection(TracesCollection)
}

// Ping verifies the connection to MongoDB.
func (c *Client) Ping(ctx context.Context) error {
	if err := c.client.Ping(ctx, nil); err != nil {
		return fmt.Errorf("mongodb ping failed: %w", err)
	}
	return nil
}

// Close closes the MongoDB connection.
func (c *Client) Close(ctx context.Context) error {
	if err := c.client.Disconnect(ctx); err != nil {
		return fmt.Errorf("failed to disconnect from mongodb: %w", err)
	}
	return nil
}

// EnsureIndexes creates all necessary indexes for all collections.
func (c *Client) EnsureIndexes(ctx context.Context) error {
	if err := c.messagesCollection.EnsureIndexes(ctx); err != nil {
		return fmt.Errorf("failed to ensure messages indexes: %w", err)
	}
	return nil
}
