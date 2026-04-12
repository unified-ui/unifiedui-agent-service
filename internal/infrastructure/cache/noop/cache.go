// Package noop provides a no-operation cache implementation for graceful degradation.
package noop

import (
	"context"
	"time"
)

// Cache implements the cache.Cache interface as a no-operation cache.
// All operations are safe no-ops: Get returns nil (cache miss), Set/Delete do nothing.
type Cache struct{}

// NewCache creates a new no-operation cache instance.
func NewCache() *Cache {
	return &Cache{}
}

// Get always returns nil (cache miss) with no error.
func (c *Cache) Get(_ context.Context, _ string) ([]byte, error) {
	return nil, nil
}

// Set does nothing and returns no error.
func (c *Cache) Set(_ context.Context, _ string, _ []byte, _ time.Duration) error {
	return nil
}

// Delete always returns false (key not found) with no error.
func (c *Cache) Delete(_ context.Context, _ string) (bool, error) {
	return false, nil
}

// DeletePattern always returns 0 (no keys deleted) with no error.
func (c *Cache) DeletePattern(_ context.Context, _ string) (int64, error) {
	return 0, nil
}

// Ping always returns nil (healthy).
func (c *Cache) Ping(_ context.Context) error {
	return nil
}

// Close does nothing and returns no error.
func (c *Cache) Close() error {
	return nil
}
