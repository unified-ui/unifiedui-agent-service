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

// GetChatAgentConfig mocks the GetChatAgentConfig method.
func (m *MockPlatformClient) GetChatAgentConfig(ctx context.Context, tenantID, chatAgentID, authToken string, useCache bool) (*platform.ChatAgentConfigResponse, error) {
	args := m.Called(ctx, tenantID, chatAgentID, authToken, useCache)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*platform.ChatAgentConfigResponse), args.Error(1)
}

// GetAgentConfig mocks the GetAgentConfig method.
func (m *MockPlatformClient) GetAgentConfig(ctx context.Context, tenantID, chatAgentID, conversationID, authToken string, useCache bool) (*platform.AgentConfig, error) {
	args := m.Called(ctx, tenantID, chatAgentID, conversationID, authToken, useCache)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*platform.AgentConfig), args.Error(1)
}

// GetAgentConfigFromFile mocks the GetAgentConfigFromFile method.
func (m *MockPlatformClient) GetAgentConfigFromFile(ctx context.Context, tenantID, chatAgentID string) (*platform.AgentConfig, error) {
	args := m.Called(ctx, tenantID, chatAgentID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*platform.AgentConfig), args.Error(1)
}

// GetMe mocks the GetMe method.
// Note: The identity/me endpoint doesn't require tenantId.
func (m *MockPlatformClient) GetMe(ctx context.Context, authToken string) (*platform.UserInfo, error) {
	args := m.Called(ctx, authToken)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*platform.UserInfo), args.Error(1)
}

// ValidateConversation mocks the ValidateConversation method.
func (m *MockPlatformClient) ValidateConversation(ctx context.Context, tenantID, conversationID, authToken string) error {
	args := m.Called(ctx, tenantID, conversationID, authToken)
	return args.Error(0)
}

// ValidateAutonomousAgent mocks the ValidateAutonomousAgent method.
func (m *MockPlatformClient) ValidateAutonomousAgent(ctx context.Context, tenantID, autonomousAgentID, authToken string) error {
	args := m.Called(ctx, tenantID, autonomousAgentID, authToken)
	return args.Error(0)
}

// GetConversation mocks the GetConversation method.
func (m *MockPlatformClient) GetConversation(ctx context.Context, tenantID, conversationID, authToken string) (*platform.ConversationResponse, error) {
	args := m.Called(ctx, tenantID, conversationID, authToken)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*platform.ConversationResponse), args.Error(1)
}

// GetAutonomousAgentConfig mocks the GetAutonomousAgentConfig method.
func (m *MockPlatformClient) GetAutonomousAgentConfig(ctx context.Context, tenantID, autonomousAgentID, apiKey string) (*platform.AutonomousAgentConfigResponse, error) {
	args := m.Called(ctx, tenantID, autonomousAgentID, apiKey)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*platform.AutonomousAgentConfigResponse), args.Error(1)
}

// GetAutonomousAgentConfigWithBearer mocks the GetAutonomousAgentConfigWithBearer method.
func (m *MockPlatformClient) GetAutonomousAgentConfigWithBearer(ctx context.Context, tenantID, autonomousAgentID, authToken string) (*platform.AutonomousAgentConfigResponse, error) {
	args := m.Called(ctx, tenantID, autonomousAgentID, authToken)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*platform.AutonomousAgentConfigResponse), args.Error(1)
}

// ValidateAutonomousAgentAPIKey mocks the ValidateAutonomousAgentAPIKey method.
func (m *MockPlatformClient) ValidateAutonomousAgentAPIKey(ctx context.Context, tenantID, autonomousAgentID, apiKey string) error {
	args := m.Called(ctx, tenantID, autonomousAgentID, apiKey)
	return args.Error(0)
}

// GetAIModelsByPurpose mocks the GetAIModelsByPurpose method.
func (m *MockPlatformClient) GetAIModelsByPurpose(ctx context.Context, tenantID, purposeGroup, modelType string) ([]platform.AIModelWithSecretResponse, error) {
	args := m.Called(ctx, tenantID, purposeGroup, modelType)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]platform.AIModelWithSecretResponse), args.Error(1)
}

// GetCredentialSecret mocks the GetCredentialSecret method.
func (m *MockPlatformClient) GetCredentialSecret(ctx context.Context, tenantID, credentialID, authToken string) (string, error) {
	args := m.Called(ctx, tenantID, credentialID, authToken)
	return args.String(0), args.Error(1)
}

// UpdateConversationTitle mocks the UpdateConversationTitle method.
func (m *MockPlatformClient) UpdateConversationTitle(ctx context.Context, tenantID, conversationID, title, authToken string) error {
	args := m.Called(ctx, tenantID, conversationID, title, authToken)
	return args.Error(0)
}

// Ensure MockPlatformClient implements platform.Client interface.
var _ platform.Client = (*MockPlatformClient)(nil)
