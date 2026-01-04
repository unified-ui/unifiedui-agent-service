package mocks

import (
	"context"

	"github.com/stretchr/testify/mock"

	"github.com/unifiedui/agent-service/internal/domain/models"
	"github.com/unifiedui/agent-service/internal/services/session"
)

// MockSessionService is a mock implementation of session.Service.
type MockSessionService struct {
	mock.Mock
}

// GetSession retrieves a session from cache.
func (m *MockSessionService) GetSession(ctx context.Context, tenantID, userID, conversationID string) (*session.SessionData, error) {
	args := m.Called(ctx, tenantID, userID, conversationID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*session.SessionData), args.Error(1)
}

// SetSession stores a session in cache.
func (m *MockSessionService) SetSession(ctx context.Context, sessionData *session.SessionData) error {
	args := m.Called(ctx, sessionData)
	return args.Error(0)
}

// UpdateChatHistory updates the chat history in an existing session.
func (m *MockSessionService) UpdateChatHistory(ctx context.Context, tenantID, userID, conversationID string, newEntries []models.ChatHistoryEntry) error {
	args := m.Called(ctx, tenantID, userID, conversationID, newEntries)
	return args.Error(0)
}

// DeleteSession removes a session from cache.
func (m *MockSessionService) DeleteSession(ctx context.Context, tenantID, userID, conversationID string) error {
	args := m.Called(ctx, tenantID, userID, conversationID)
	return args.Error(0)
}

// BuildCacheKey generates the cache key for a session.
func (m *MockSessionService) BuildCacheKey(tenantID, userID, conversationID string) string {
	args := m.Called(tenantID, userID, conversationID)
	return args.String(0)
}
