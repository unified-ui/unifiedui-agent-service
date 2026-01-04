// Package cache defines the cache interface and factory.
package cache

import (
	"context"
	"time"
)

// Cache defines the interface for cache operations.
type Cache interface {
	// Get retrieves a value from the cache by key.
	// Returns nil if the key does not exist.
	Get(ctx context.Context, key string) ([]byte, error)

	// Set stores a value in the cache with an optional TTL.
	// If ttl is 0, the default TTL is used.
	Set(ctx context.Context, key string, value []byte, ttl time.Duration) error

	// Delete removes a key from the cache.
	// Returns true if the key was deleted, false if it didn't exist.
	Delete(ctx context.Context, key string) (bool, error)

	// DeletePattern removes all keys matching the given pattern.
	// Returns the number of keys deleted.
	DeletePattern(ctx context.Context, pattern string) (int64, error)

	// Ping checks if the cache connection is alive.
	Ping(ctx context.Context) error

	// Close closes the cache connection.
	Close() error
}
