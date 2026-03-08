package mocks

import (
	"context"

	"github.com/stretchr/testify/mock"

	"github.com/unifiedui/agent-service/internal/services/platform"
)

// MockConfigCacheService is a mock implementation of configcache.Service.
type MockConfigCacheService struct {
	mock.Mock
}

// Get retrieves an agent config from cache.
func (m *MockConfigCacheService) Get(ctx context.Context, tenantID, userID, chatAgentID string) (*platform.AgentConfig, error) {
	args := m.Called(ctx, tenantID, userID, chatAgentID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*platform.AgentConfig), args.Error(1)
}

// Set stores an agent config in cache.
func (m *MockConfigCacheService) Set(ctx context.Context, tenantID, userID, chatAgentID string, config *platform.AgentConfig) error {
	args := m.Called(ctx, tenantID, userID, chatAgentID, config)
	return args.Error(0)
}

// Delete removes an agent config from cache.
func (m *MockConfigCacheService) Delete(ctx context.Context, tenantID, userID, chatAgentID string) error {
	args := m.Called(ctx, tenantID, userID, chatAgentID)
	return args.Error(0)
}

// DeleteByTenant removes all agent configs for a tenant from cache.
func (m *MockConfigCacheService) DeleteByTenant(ctx context.Context, tenantID string) error {
	args := m.Called(ctx, tenantID)
	return args.Error(0)
}

// BuildCacheKey generates the cache key for an agent config.
func (m *MockConfigCacheService) BuildCacheKey(tenantID, userID, chatAgentID string) string {
	args := m.Called(tenantID, userID, chatAgentID)
	return args.String(0)
}
