// Package noop provides a no-operation cache client implementation.
package noop

import (
	"context"
	"time"

	"github.com/unifiedui/agent-service/internal/core/cache"
)

// Client implements the cache.Client interface as a no-operation client.
type Client struct {
	cache *Cache
}

// NewClient creates a new no-operation cache client.
func NewClient() *Client {
	return &Client{
		cache: NewCache(),
	}
}

// GetCache returns the underlying NoOp Cache implementation.
func (c *Client) GetCache() cache.Cache {
	return c.cache
}

// Get always returns nil (cache miss) with no error.
func (c *Client) Get(ctx context.Context, key string) ([]byte, error) {
	return c.cache.Get(ctx, key)
}

// Set does nothing and returns no error.
func (c *Client) Set(ctx context.Context, key string, value []byte, ttl time.Duration) error {
	return c.cache.Set(ctx, key, value, ttl)
}

// Delete always returns false (key not found) with no error.
func (c *Client) Delete(ctx context.Context, key string) (bool, error) {
	return c.cache.Delete(ctx, key)
}

// DeletePattern always returns 0 (no keys deleted) with no error.
func (c *Client) DeletePattern(ctx context.Context, pattern string) (int64, error) {
	return c.cache.DeletePattern(ctx, pattern)
}

// Ping always returns nil (healthy).
func (c *Client) Ping(ctx context.Context) error {
	return c.cache.Ping(ctx)
}

// Close does nothing and returns no error.
func (c *Client) Close() error {
	return c.cache.Close()
}
