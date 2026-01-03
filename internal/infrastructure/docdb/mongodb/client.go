// Package mongodb provides MongoDB client implementation.
package mongodb

import (
	"context"
	"fmt"

	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

	"github.com/unifiedui/chat-service/internal/core/docdb"
)

const (
	// MessagesCollection is the name of the messages collection.
	MessagesCollection = "messages"
	// TracesCollection is the name of the traces collection.
	TracesCollection = "traces"
)

// Client implements the docdb.Client interface for MongoDB.
type Client struct {
	client   *mongo.Client
	database *Database
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

	database := NewDatabase(client.Database(config.DatabaseName))

	return &Client{
		client:   client,
		database: database,
	}, nil
}

// Database returns the database interface.
func (c *Client) Database() docdb.Database {
	return c.database
}

// Messages returns the messages collection.
func (c *Client) Messages() docdb.Collection {
	return c.database.Collection(MessagesCollection)
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
