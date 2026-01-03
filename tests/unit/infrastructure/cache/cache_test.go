// Package cache_test provides unit tests for the cache package.
package cache_test

import (
	"context"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/unifiedui/chat-service/internal/core/cache"
	rediscache "github.com/unifiedui/chat-service/internal/infrastructure/cache/redis"
)

func setupMiniredis(t *testing.T) (*miniredis.Miniredis, cache.Client) {
	t.Helper()

	mr, err := miniredis.Run()
	require.NoError(t, err)

	client, err := rediscache.NewClient(rediscache.Config{
		Host:     mr.Host(),
		Port:     mr.Port(),
		Password: "",
		DB:       0,
	})
	require.NoError(t, err)

	t.Cleanup(func() {
		client.Close()
		mr.Close()
	})

	return mr, client
}

func TestNewClient_Success(t *testing.T) {
	mr, err := miniredis.Run()
	require.NoError(t, err)
	defer mr.Close()

	client, err := rediscache.NewClient(rediscache.Config{
		Host:     mr.Host(),
		Port:     mr.Port(),
		Password: "",
		DB:       0,
	})

	assert.NoError(t, err)
	assert.NotNil(t, client)

	client.Close()
}

func TestCache_SetAndGet(t *testing.T) {
	_, client := setupMiniredis(t)
	ctx := context.Background()

	key := "test-key"
	value := []byte("test-value")
	ttl := 1 * time.Minute

	// Set
	err := client.Set(ctx, key, value, ttl)
	assert.NoError(t, err)

	// Get
	result, err := client.Get(ctx, key)
	assert.NoError(t, err)
	assert.Equal(t, value, result)
}

func TestCache_GetNotFound(t *testing.T) {
	_, client := setupMiniredis(t)
	ctx := context.Background()

	result, err := client.Get(ctx, "non-existent-key")

	// According to interface: Get returns nil if key does not exist
	assert.NoError(t, err)
	assert.Nil(t, result)
}

func TestCache_Delete(t *testing.T) {
	_, client := setupMiniredis(t)
	ctx := context.Background()

	key := "test-key"
	value := []byte("test-value")

	// Set
	err := client.Set(ctx, key, value, 1*time.Minute)
	require.NoError(t, err)

	// Delete
	deleted, err := client.Delete(ctx, key)
	assert.NoError(t, err)
	assert.True(t, deleted)

	// Verify deleted - Get returns nil when key doesn't exist
	result, err := client.Get(ctx, key)
	assert.NoError(t, err)
	assert.Nil(t, result)
}

func TestCache_DeletePattern(t *testing.T) {
	mr, client := setupMiniredis(t)
	ctx := context.Background()

	// Set multiple keys
	client.Set(ctx, "session:tenant1:user1", []byte("value1"), 1*time.Minute)
	client.Set(ctx, "session:tenant1:user2", []byte("value2"), 1*time.Minute)
	client.Set(ctx, "session:tenant2:user1", []byte("value3"), 1*time.Minute)
	client.Set(ctx, "other:key", []byte("value4"), 1*time.Minute)

	// Delete pattern
	deleted, err := client.DeletePattern(ctx, "session:tenant1:*")
	assert.NoError(t, err)
	assert.Equal(t, int64(2), deleted)

	// Verify only matching keys deleted
	keys := mr.Keys()
	assert.Contains(t, keys, "session:tenant2:user1")
	assert.Contains(t, keys, "other:key")
	assert.NotContains(t, keys, "session:tenant1:user1")
	assert.NotContains(t, keys, "session:tenant1:user2")
}

func TestCache_Ping(t *testing.T) {
	_, client := setupMiniredis(t)
	ctx := context.Background()

	err := client.Ping(ctx)
	assert.NoError(t, err)
}

func TestCache_TTLExpiration(t *testing.T) {
	mr, client := setupMiniredis(t)
	ctx := context.Background()

	key := "expiring-key"
	value := []byte("expiring-value")

	// Set with short TTL
	err := client.Set(ctx, key, value, 1*time.Second)
	require.NoError(t, err)

	// Verify exists
	result, err := client.Get(ctx, key)
	assert.NoError(t, err)
	assert.Equal(t, value, result)

	// Fast-forward time
	mr.FastForward(2 * time.Second)

	// Verify expired - Get returns nil when key doesn't exist
	result, err = client.Get(ctx, key)
	assert.NoError(t, err)
	assert.Nil(t, result)
}
