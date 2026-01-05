// Package mocks provides mock implementations for testing.
package mocks

import (
	"context"

	"github.com/unifiedui/agent-service/internal/services/platform"

	"github.com/stretchr/testify/mock"
)

// MockPlatformClient is a mock implementation of platform.Client.
type MockPlatformClient struct {
	mock.Mock
}

// GetApplicationConfig mocks the GetApplicationConfig method.
func (m *MockPlatformClient) GetApplicationConfig(ctx context.Context, tenantID, applicationID, authToken string) (*platform.ApplicationConfigResponse, error) {
	args := m.Called(ctx, tenantID, applicationID, authToken)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*platform.ApplicationConfigResponse), args.Error(1)
}

// GetAgentConfig mocks the GetAgentConfig method.
func (m *MockPlatformClient) GetAgentConfig(ctx context.Context, tenantID, applicationID, conversationID, authToken string) (*platform.AgentConfig, error) {
	args := m.Called(ctx, tenantID, applicationID, conversationID, authToken)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*platform.AgentConfig), args.Error(1)
}

// GetAgentConfigFromFile mocks the GetAgentConfigFromFile method.
func (m *MockPlatformClient) GetAgentConfigFromFile(ctx context.Context, tenantID, applicationID string) (*platform.AgentConfig, error) {
	args := m.Called(ctx, tenantID, applicationID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*platform.AgentConfig), args.Error(1)
}

// Ensure MockPlatformClient implements platform.Client interface.
var _ platform.Client = (*MockPlatformClient)(nil)
