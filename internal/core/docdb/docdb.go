// Package docdb defines the document database interface.
package docdb

import (
	"context"
)

// SingleResult represents the result of a FindOne operation.
type SingleResult interface {
	// Decode decodes the result into the provided interface.
	Decode(v interface{}) error
	// Err returns any error from the operation.
	Err() error
}

// Cursor represents a cursor for iterating over query results.
type Cursor interface {
	// Next advances the cursor to the next document.
	Next(ctx context.Context) bool
	// Decode decodes the current document.
	Decode(v interface{}) error
	// All decodes all remaining documents.
	All(ctx context.Context, results interface{}) error
	// Err returns any cursor error.
	Err() error
	// Close closes the cursor.
	Close(ctx context.Context) error
}

// FindOptions represents options for Find operations.
type FindOptions struct {
	Limit int64
	Skip  int64
	Sort  interface{}
}

// UpdateResult represents the result of an update operation.
type UpdateResult struct {
	MatchedCount  int64
	ModifiedCount int64
	UpsertedCount int64
	UpsertedID    interface{}
}

// DeleteResult represents the result of a delete operation.
type DeleteResult struct {
	DeletedCount int64
}

// Collection defines the interface for document collection operations.
type Collection interface {
	// InsertOne inserts a single document.
	InsertOne(ctx context.Context, document interface{}) (interface{}, error)

	// InsertMany inserts multiple documents.
	InsertMany(ctx context.Context, documents []interface{}) ([]interface{}, error)

	// FindOne finds a single document.
	FindOne(ctx context.Context, filter interface{}) SingleResult

	// Find finds multiple documents.
	Find(ctx context.Context, filter interface{}, opts *FindOptions) (Cursor, error)

	// UpdateOne updates a single document.
	UpdateOne(ctx context.Context, filter interface{}, update interface{}) (*UpdateResult, error)

	// UpdateMany updates multiple documents.
	UpdateMany(ctx context.Context, filter interface{}, update interface{}) (*UpdateResult, error)

	// DeleteOne deletes a single document.
	DeleteOne(ctx context.Context, filter interface{}) (*DeleteResult, error)

	// DeleteMany deletes multiple documents.
	DeleteMany(ctx context.Context, filter interface{}) (*DeleteResult, error)

	// CountDocuments counts documents matching the filter.
	CountDocuments(ctx context.Context, filter interface{}) (int64, error)
}

// Database defines the interface for database operations.
type Database interface {
	// Collection returns a collection by name.
	Collection(name string) Collection

	// ListCollectionNames lists all collection names.
	ListCollectionNames(ctx context.Context) ([]string, error)
}
