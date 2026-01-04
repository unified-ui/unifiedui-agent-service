// Package mocks provides mock implementations for testing.
package mocks

import (
	"context"
	"time"

	"github.com/stretchr/testify/mock"

	"github.com/unifiedui/agent-service/internal/core/cache"
)

// MockCache is a mock implementation of cache.Cache.
type MockCache struct {
	mock.Mock
}

// Get retrieves a value from the cache.
func (m *MockCache) Get(ctx context.Context, key string) ([]byte, error) {
	args := m.Called(ctx, key)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]byte), args.Error(1)
}

// Set stores a value in the cache.
func (m *MockCache) Set(ctx context.Context, key string, value []byte, ttl time.Duration) error {
	args := m.Called(ctx, key, value, ttl)
	return args.Error(0)
}

// Delete removes a value from the cache.
func (m *MockCache) Delete(ctx context.Context, key string) (bool, error) {
	args := m.Called(ctx, key)
	return args.Bool(0), args.Error(1)
}

// DeletePattern removes all values matching the pattern.
func (m *MockCache) DeletePattern(ctx context.Context, pattern string) (int64, error) {
	args := m.Called(ctx, pattern)
	return args.Get(0).(int64), args.Error(1)
}

// Ping checks the cache connection.
func (m *MockCache) Ping(ctx context.Context) error {
	args := m.Called(ctx)
	return args.Error(0)
}

// Close closes the cache connection.
func (m *MockCache) Close() error {
	args := m.Called()
	return args.Error(0)
}

// MockCacheClient is a mock implementation of cache.Client.
type MockCacheClient struct {
	mock.Mock
	cache *MockCache
}

// NewMockCacheClient creates a new MockCacheClient.
func NewMockCacheClient() *MockCacheClient {
	return &MockCacheClient{
		cache: &MockCache{},
	}
}

// GetCache returns the underlying cache.
func (m *MockCacheClient) GetCache() cache.Cache {
	return m.cache
}

// Get retrieves a value from the cache.
func (m *MockCacheClient) Get(ctx context.Context, key string) ([]byte, error) {
	args := m.Called(ctx, key)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]byte), args.Error(1)
}

// Set stores a value in the cache.
func (m *MockCacheClient) Set(ctx context.Context, key string, value []byte, ttl time.Duration) error {
	args := m.Called(ctx, key, value, ttl)
	return args.Error(0)
}

// Delete removes a value from the cache.
func (m *MockCacheClient) Delete(ctx context.Context, key string) (bool, error) {
	args := m.Called(ctx, key)
	return args.Bool(0), args.Error(1)
}

// DeletePattern removes all values matching the pattern.
func (m *MockCacheClient) DeletePattern(ctx context.Context, pattern string) (int64, error) {
	args := m.Called(ctx, pattern)
	return args.Get(0).(int64), args.Error(1)
}

// Ping checks the cache connection.
func (m *MockCacheClient) Ping(ctx context.Context) error {
	args := m.Called(ctx)
	return args.Error(0)
}

// Close closes the cache connection.
func (m *MockCacheClient) Close() error {
	args := m.Called()
	return args.Error(0)
}
