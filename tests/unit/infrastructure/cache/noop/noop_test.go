// Package noopcache_test provides unit tests for the NoOp cache package.
package noopcache_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/unifiedui/agent-service/internal/core/cache"
	noopcache "github.com/unifiedui/agent-service/internal/infrastructure/cache/noop"
)

func TestNewClient_ImplementsInterface(t *testing.T) {
	client := noopcache.NewClient()
	assert.NotNil(t, client)

	var _ cache.Client = client
}

func TestNoOpCache_Get_ReturnsCacheMiss(t *testing.T) {
	client := noopcache.NewClient()
	ctx := context.Background()

	result, err := client.Get(ctx, "any-key")
	assert.NoError(t, err)
	assert.Nil(t, result)
}

func TestNoOpCache_Set_DoesNothing(t *testing.T) {
	client := noopcache.NewClient()
	ctx := context.Background()

	err := client.Set(ctx, "any-key", []byte("any-value"), 1*time.Minute)
	assert.NoError(t, err)

	result, err := client.Get(ctx, "any-key")
	assert.NoError(t, err)
	assert.Nil(t, result)
}

func TestNoOpCache_Delete_ReturnsFalse(t *testing.T) {
	client := noopcache.NewClient()
	ctx := context.Background()

	deleted, err := client.Delete(ctx, "any-key")
	assert.NoError(t, err)
	assert.False(t, deleted)
}

func TestNoOpCache_DeletePattern_ReturnsZero(t *testing.T) {
	client := noopcache.NewClient()
	ctx := context.Background()

	deleted, err := client.DeletePattern(ctx, "pattern:*")
	assert.NoError(t, err)
	assert.Equal(t, int64(0), deleted)
}

func TestNoOpCache_Ping_ReturnsNil(t *testing.T) {
	client := noopcache.NewClient()
	ctx := context.Background()

	err := client.Ping(ctx)
	assert.NoError(t, err)
}

func TestNoOpCache_Close_ReturnsNil(t *testing.T) {
	client := noopcache.NewClient()

	err := client.Close()
	assert.NoError(t, err)
}

func TestNoOpCache_GetCache_ReturnsCache(t *testing.T) {
	client := noopcache.NewClient()

	underlyingCache := client.GetCache()
	assert.NotNil(t, underlyingCache)
}
