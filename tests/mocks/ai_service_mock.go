// Package mocks provides mock implementations for testing.
package mocks

import (
	"context"

	"github.com/unifiedui/agent-service/internal/services/ai"

	"github.com/stretchr/testify/mock"
)

// MockAIService is a mock implementation of ai.Service.
type MockAIService struct {
	mock.Mock
}

// GenerateTitle mocks the GenerateTitle method.
func (m *MockAIService) GenerateTitle(ctx context.Context, tenantID, userMessage, assistantResponse string) (string, error) {
	args := m.Called(ctx, tenantID, userMessage, assistantResponse)
	return args.String(0), args.Error(1)
}

// GenerateDescription mocks the GenerateDescription method.
func (m *MockAIService) GenerateDescription(ctx context.Context, tenantID, entityType, entityName, existingDescription string, entityContext map[string]interface{}) (string, error) {
	args := m.Called(ctx, tenantID, entityType, entityName, existingDescription, entityContext)
	return args.String(0), args.Error(1)
}

// AnalyzeTrace mocks the AnalyzeTrace method.
func (m *MockAIService) AnalyzeTrace(ctx context.Context, tenantID string, request ai.AnalyzeTraceInput) (string, error) {
	args := m.Called(ctx, tenantID, request)
	return args.String(0), args.Error(1)
}

// SummarizeTrace mocks the SummarizeTrace method.
func (m *MockAIService) SummarizeTrace(ctx context.Context, tenantID string, request ai.SummarizeTraceInput) (string, error) {
	args := m.Called(ctx, tenantID, request)
	return args.String(0), args.Error(1)
}

// TestModel mocks the TestModel method.
func (m *MockAIService) TestModel(ctx context.Context, provider string, config map[string]interface{}, credentialSecret map[string]interface{}) (*ai.TestModelResult, error) {
	args := m.Called(ctx, provider, config, credentialSecret)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*ai.TestModelResult), args.Error(1)
}

// GetCapabilities mocks the GetCapabilities method.
func (m *MockAIService) GetCapabilities(ctx context.Context, tenantID string) (*ai.Capabilities, error) {
	args := m.Called(ctx, tenantID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*ai.Capabilities), args.Error(1)
}

// Ensure MockAIService implements ai.Service interface.
var _ ai.Service = (*MockAIService)(nil)
