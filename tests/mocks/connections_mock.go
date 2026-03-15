// Package mocks provides mock implementations for testing.
package mocks

import (
	"context"

	"github.com/stretchr/testify/mock"

	"github.com/unifiedui/agent-service/internal/services/connections"
	"github.com/unifiedui/agent-service/internal/services/platform"
)

// MockConnectionService is a mock implementation of connections.Service.
type MockConnectionService struct {
	mock.Mock
}

// TestConnection mocks the TestConnection method.
func (m *MockConnectionService) TestConnection(ctx context.Context, testType string, rawURL string, config map[string]interface{}, credential *platform.Credentials, userToken string) (*connections.TestResult, error) {
	args := m.Called(ctx, testType, rawURL, config, credential, userToken)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*connections.TestResult), args.Error(1)
}
