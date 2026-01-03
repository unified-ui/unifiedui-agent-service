// Package cache defines the cache client interface.
package cache

import (
	"context"
	"time"
)

// Client is a higher-level cache client that wraps the Cache interface.
// It provides additional functionality like JSON serialization.
type Client interface {
	// GetCache returns the underlying Cache implementation.
	GetCache() Cache

	// Get retrieves a value from the cache and deserializes it.
	Get(ctx context.Context, key string) ([]byte, error)

	// Set serializes and stores a value in the cache.
	Set(ctx context.Context, key string, value []byte, ttl time.Duration) error

	// Delete removes a key from the cache.
	Delete(ctx context.Context, key string) (bool, error)

	// DeletePattern removes all keys matching the given pattern.
	DeletePattern(ctx context.Context, pattern string) (int64, error)

	// Ping checks if the cache connection is alive.
	Ping(ctx context.Context) error

	// Close closes the cache client connection.
	Close() error
}
