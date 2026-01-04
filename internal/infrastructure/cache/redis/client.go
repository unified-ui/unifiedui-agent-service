// Package redis provides the Redis cache client implementation.
package redis

import (
	"context"
	"time"

	"github.com/unifiedui/agent-service/internal/core/cache"
)

// Client implements the cache.Client interface for Redis.
type Client struct {
	cache *Cache
}

// NewClient creates a new Redis cache client.
func NewClient(cfg Config) (*Client, error) {
	c, err := NewCache(cfg)
	if err != nil {
		return nil, err
	}

	return &Client{
		cache: c,
	}, nil
}

// GetCache returns the underlying Cache implementation.
func (c *Client) GetCache() cache.Cache {
	return c.cache
}

// Get retrieves a value from the cache.
func (c *Client) Get(ctx context.Context, key string) ([]byte, error) {
	return c.cache.Get(ctx, key)
}

// Set stores a value in the cache.
func (c *Client) Set(ctx context.Context, key string, value []byte, ttl time.Duration) error {
	return c.cache.Set(ctx, key, value, ttl)
}

// Delete removes a key from the cache.
func (c *Client) Delete(ctx context.Context, key string) (bool, error) {
	return c.cache.Delete(ctx, key)
}

// DeletePattern removes all keys matching the given pattern.
func (c *Client) DeletePattern(ctx context.Context, pattern string) (int64, error) {
	return c.cache.DeletePattern(ctx, pattern)
}

// Ping checks if the cache connection is alive.
func (c *Client) Ping(ctx context.Context) error {
	return c.cache.Ping(ctx)
}

// Close closes the cache client connection.
func (c *Client) Close() error {
	return c.cache.Close()
}
