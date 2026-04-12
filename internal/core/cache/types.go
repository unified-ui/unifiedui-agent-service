// Package cache provides the cache type constants.
package cache

// Type represents the type of cache.
type Type string

const (
	// TypeRedis represents a Redis cache.
	TypeRedis Type = "redis"

	// TypeNone represents a no-operation cache (disabled caching).
	TypeNone Type = "none"
)
