// Package mongodb provides MongoDB database implementation.
package mongodb

import (
	"context"
	"fmt"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

	"github.com/unifiedui/chat-service/internal/core/docdb"
)

// Collection implements the docdb.Collection interface for MongoDB.
type Collection struct {
	collection *mongo.Collection
}

// NewCollection creates a new MongoDB collection wrapper.
func NewCollection(collection *mongo.Collection) *Collection {
	return &Collection{
		collection: collection,
	}
}

// InsertOne inserts a single document.
func (c *Collection) InsertOne(ctx context.Context, document interface{}) (interface{}, error) {
	result, err := c.collection.InsertOne(ctx, document)
	if err != nil {
		return nil, fmt.Errorf("failed to insert document: %w", err)
	}
	return result.InsertedID, nil
}

// InsertMany inserts multiple documents.
func (c *Collection) InsertMany(ctx context.Context, documents []interface{}) ([]interface{}, error) {
	result, err := c.collection.InsertMany(ctx, documents)
	if err != nil {
		return nil, fmt.Errorf("failed to insert documents: %w", err)
	}
	return result.InsertedIDs, nil
}

// FindOne finds a single document matching the filter.
func (c *Collection) FindOne(ctx context.Context, filter interface{}) docdb.SingleResult {
	return &SingleResult{
		result: c.collection.FindOne(ctx, filter),
	}
}

// Find finds all documents matching the filter.
func (c *Collection) Find(ctx context.Context, filter interface{}, opts *docdb.FindOptions) (docdb.Cursor, error) {
	findOpts := options.Find()
	if opts != nil {
		if opts.Limit > 0 {
			findOpts.SetLimit(opts.Limit)
		}
		if opts.Skip > 0 {
			findOpts.SetSkip(opts.Skip)
		}
		if opts.Sort != nil {
			findOpts.SetSort(opts.Sort)
		}
	}

	cursor, err := c.collection.Find(ctx, filter, findOpts)
	if err != nil {
		return nil, fmt.Errorf("failed to find documents: %w", err)
	}

	return &Cursor{cursor: cursor}, nil
}

// UpdateOne updates a single document matching the filter.
func (c *Collection) UpdateOne(ctx context.Context, filter interface{}, update interface{}) (*docdb.UpdateResult, error) {
	result, err := c.collection.UpdateOne(ctx, filter, update)
	if err != nil {
		return nil, fmt.Errorf("failed to update document: %w", err)
	}

	return &docdb.UpdateResult{
		MatchedCount:  result.MatchedCount,
		ModifiedCount: result.ModifiedCount,
		UpsertedCount: result.UpsertedCount,
		UpsertedID:    result.UpsertedID,
	}, nil
}

// UpdateMany updates all documents matching the filter.
func (c *Collection) UpdateMany(ctx context.Context, filter interface{}, update interface{}) (*docdb.UpdateResult, error) {
	result, err := c.collection.UpdateMany(ctx, filter, update)
	if err != nil {
		return nil, fmt.Errorf("failed to update documents: %w", err)
	}

	return &docdb.UpdateResult{
		MatchedCount:  result.MatchedCount,
		ModifiedCount: result.ModifiedCount,
		UpsertedCount: result.UpsertedCount,
		UpsertedID:    result.UpsertedID,
	}, nil
}

// DeleteOne deletes a single document matching the filter.
func (c *Collection) DeleteOne(ctx context.Context, filter interface{}) (*docdb.DeleteResult, error) {
	result, err := c.collection.DeleteOne(ctx, filter)
	if err != nil {
		return nil, fmt.Errorf("failed to delete document: %w", err)
	}

	return &docdb.DeleteResult{
		DeletedCount: result.DeletedCount,
	}, nil
}

// DeleteMany deletes all documents matching the filter.
func (c *Collection) DeleteMany(ctx context.Context, filter interface{}) (*docdb.DeleteResult, error) {
	result, err := c.collection.DeleteMany(ctx, filter)
	if err != nil {
		return nil, fmt.Errorf("failed to delete documents: %w", err)
	}

	return &docdb.DeleteResult{
		DeletedCount: result.DeletedCount,
	}, nil
}

// CountDocuments counts documents matching the filter.
func (c *Collection) CountDocuments(ctx context.Context, filter interface{}) (int64, error) {
	count, err := c.collection.CountDocuments(ctx, filter)
	if err != nil {
		return 0, fmt.Errorf("failed to count documents: %w", err)
	}
	return count, nil
}

// Database implements the docdb.Database interface for MongoDB.
type Database struct {
	database *mongo.Database
}

// NewDatabase creates a new MongoDB database wrapper.
func NewDatabase(database *mongo.Database) *Database {
	return &Database{
		database: database,
	}
}

// Collection returns a collection from the database.
func (d *Database) Collection(name string) docdb.Collection {
	return NewCollection(d.database.Collection(name))
}

// ListCollectionNames lists all collection names in the database.
func (d *Database) ListCollectionNames(ctx context.Context) ([]string, error) {
	names, err := d.database.ListCollectionNames(ctx, bson.D{})
	if err != nil {
		return nil, fmt.Errorf("failed to list collections: %w", err)
	}
	return names, nil
}

// SingleResult wraps a MongoDB single result.
type SingleResult struct {
	result *mongo.SingleResult
}

// Decode decodes the single result into the provided interface.
func (r *SingleResult) Decode(v interface{}) error {
	return r.result.Decode(v)
}

// Err returns any error from the single result.
func (r *SingleResult) Err() error {
	return r.result.Err()
}

// Cursor wraps a MongoDB cursor.
type Cursor struct {
	cursor *mongo.Cursor
}

// Next advances the cursor.
func (c *Cursor) Next(ctx context.Context) bool {
	return c.cursor.Next(ctx)
}

// Decode decodes the current document.
func (c *Cursor) Decode(v interface{}) error {
	return c.cursor.Decode(v)
}

// All decodes all remaining documents.
func (c *Cursor) All(ctx context.Context, results interface{}) error {
	return c.cursor.All(ctx, results)
}

// Err returns any cursor error.
func (c *Cursor) Err() error {
	return c.cursor.Err()
}

// Close closes the cursor.
func (c *Cursor) Close(ctx context.Context) error {
	return c.cursor.Close(ctx)
}
