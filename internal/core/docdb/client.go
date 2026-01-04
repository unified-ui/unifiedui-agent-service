// Package docdb defines the document database client interface.
package docdb

import (
	"context"
)

// Client defines the interface for a document database client.
type Client interface {
	// Database returns the database interface.
	Database() Database

	// Messages returns the messages collection.
	Messages() Collection

	// Traces returns the traces collection.
	Traces() Collection

	// Ping verifies the database connection.
	Ping(ctx context.Context) error

	// Close closes the database connection.
	Close(ctx context.Context) error
}
