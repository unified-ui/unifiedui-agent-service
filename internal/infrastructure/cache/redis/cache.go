// Package redis provides the Redis cache implementation.
package redis

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

// Config holds Redis connection configuration.
type Config struct {
	Host       string
	Port       string
	Password   string
	DB         int
	DefaultTTL time.Duration
}

// Cache implements the cache.Cache interface for Redis.
type Cache struct {
	client     *redis.Client
	defaultTTL time.Duration
}

// NewCache creates a new Redis cache instance.
func NewCache(cfg Config) (*Cache, error) {
	client := redis.NewClient(&redis.Options{
		Addr:     fmt.Sprintf("%s:%s", cfg.Host, cfg.Port),
		Password: cfg.Password,
		DB:       cfg.DB,
	})

	// Test connection
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := client.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("failed to connect to redis: %w", err)
	}

	return &Cache{
		client:     client,
		defaultTTL: cfg.DefaultTTL,
	}, nil
}

// Get retrieves a value from Redis by key.
func (c *Cache) Get(ctx context.Context, key string) ([]byte, error) {
	val, err := c.client.Get(ctx, key).Bytes()
	if err == redis.Nil {
		return nil, nil // Key not found
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get key %s: %w", key, err)
	}
	return val, nil
}

// Set stores a value in Redis with an optional TTL.
func (c *Cache) Set(ctx context.Context, key string, value []byte, ttl time.Duration) error {
	if ttl == 0 {
		ttl = c.defaultTTL
	}

	if err := c.client.Set(ctx, key, value, ttl).Err(); err != nil {
		return fmt.Errorf("failed to set key %s: %w", key, err)
	}
	return nil
}

// Delete removes a key from Redis.
func (c *Cache) Delete(ctx context.Context, key string) (bool, error) {
	result, err := c.client.Del(ctx, key).Result()
	if err != nil {
		return false, fmt.Errorf("failed to delete key %s: %w", key, err)
	}
	return result > 0, nil
}

// DeletePattern removes all keys matching the given pattern.
func (c *Cache) DeletePattern(ctx context.Context, pattern string) (int64, error) {
	var cursor uint64
	var deleted int64

	for {
		keys, nextCursor, err := c.client.Scan(ctx, cursor, pattern, 100).Result()
		if err != nil {
			return deleted, fmt.Errorf("failed to scan keys with pattern %s: %w", pattern, err)
		}

		if len(keys) > 0 {
			result, err := c.client.Del(ctx, keys...).Result()
			if err != nil {
				return deleted, fmt.Errorf("failed to delete keys: %w", err)
			}
			deleted += result
		}

		cursor = nextCursor
		if cursor == 0 {
			break
		}
	}

	return deleted, nil
}

// Ping checks if the Redis connection is alive.
func (c *Cache) Ping(ctx context.Context) error {
	if err := c.client.Ping(ctx).Err(); err != nil {
		return fmt.Errorf("redis ping failed: %w", err)
	}
	return nil
}

// Close closes the Redis connection.
func (c *Cache) Close() error {
	if err := c.client.Close(); err != nil {
		return fmt.Errorf("failed to close redis connection: %w", err)
	}
	return nil
}

// GetClient returns the underlying Redis client (for testing purposes).
func (c *Cache) GetClient() *redis.Client {
	return c.client
}
