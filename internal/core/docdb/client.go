// Package docdb defines the document database client interface.
package docdb

import (
	"context"
)

// Client defines the interface for a document database client.
type Client interface {
	// Database returns the database interface.
	Database() Database

	// Messages returns the typed messages collection with domain methods.
	Messages() MessagesCollection

	// MessagesRaw returns the raw messages collection for direct operations.
	MessagesRaw() Collection

	// Traces returns the traces collection.
	Traces() Collection

	// Ping verifies the database connection.
	Ping(ctx context.Context) error

	// Close closes the database connection.
	Close(ctx context.Context) error

	// EnsureIndexes creates all necessary indexes for all collections.
	EnsureIndexes(ctx context.Context) error
}
