package session_test

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/unifiedui/agent-service/internal/domain/models"
	"github.com/unifiedui/agent-service/internal/services/platform"
	"github.com/unifiedui/agent-service/internal/services/session"
	"github.com/unifiedui/agent-service/tests/mocks"
)

func TestNewService_Success(t *testing.T) {
	// Arrange
	mockCache := &mocks.MockCacheClient{}
	mockEncryptor := &mocks.MockEncryptor{}

	cfg := &session.Config{
		CacheClient: mockCache,
		Encryptor:   mockEncryptor,
		TTL:         5 * time.Minute,
	}

	// Act
	svc, err := session.NewService(cfg)

	// Assert
	require.NoError(t, err)
	assert.NotNil(t, svc)
}

func TestNewService_NilConfig(t *testing.T) {
	// Act
	svc, err := session.NewService(nil)

	// Assert
	assert.Error(t, err)
	assert.Nil(t, svc)
	assert.Contains(t, err.Error(), "config is required")
}

func TestNewService_NilCacheClient(t *testing.T) {
	// Arrange
	mockEncryptor := &mocks.MockEncryptor{}

	cfg := &session.Config{
		CacheClient: nil,
		Encryptor:   mockEncryptor,
	}

	// Act
	svc, err := session.NewService(cfg)

	// Assert
	assert.Error(t, err)
	assert.Nil(t, svc)
	assert.Contains(t, err.Error(), "cache client is required")
}

func TestNewService_NilEncryptor(t *testing.T) {
	// Arrange
	mockCache := &mocks.MockCacheClient{}

	cfg := &session.Config{
		CacheClient: mockCache,
		Encryptor:   nil,
	}

	// Act
	svc, err := session.NewService(cfg)

	// Assert
	assert.Error(t, err)
	assert.Nil(t, svc)
	assert.Contains(t, err.Error(), "encryptor is required")
}

func TestNewService_DefaultTTL(t *testing.T) {
	// Arrange
	mockCache := &mocks.MockCacheClient{}
	mockEncryptor := &mocks.MockEncryptor{}

	cfg := &session.Config{
		CacheClient: mockCache,
		Encryptor:   mockEncryptor,
		TTL:         0, // Default should be used
	}

	// Act
	svc, err := session.NewService(cfg)

	// Assert
	require.NoError(t, err)
	assert.NotNil(t, svc)
}

func TestService_GetSession_Success(t *testing.T) {
	// Arrange
	mockCache := &mocks.MockCacheClient{}
	mockEncryptor := &mocks.MockEncryptor{}

	cfg := &session.Config{
		CacheClient: mockCache,
		Encryptor:   mockEncryptor,
	}

	svc, err := session.NewService(cfg)
	require.NoError(t, err)

	tenantID := "tenant-123"
	userID := "user-456"
	conversationID := "conv-789"

	sessionData := &session.SessionData{
		Config:         &platform.AgentConfig{TenantID: tenantID},
		ChatHistory:    []models.ChatHistoryEntry{{Role: models.MessageTypeUser, Content: "Hello"}},
		TenantID:       tenantID,
		UserID:         userID,
		ConversationID: conversationID,
	}

	jsonData, _ := json.Marshal(sessionData)
	encryptedData := "encrypted-data"

	mockCache.On("Get", mock.Anything, mock.AnythingOfType("string")).Return([]byte(encryptedData), nil)
	mockEncryptor.On("Decrypt", encryptedData).Return(jsonData, nil)

	// Act
	result, err := svc.GetSession(context.Background(), tenantID, userID, conversationID)

	// Assert
	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, tenantID, result.TenantID)
	assert.Equal(t, userID, result.UserID)
	assert.Equal(t, conversationID, result.ConversationID)
	assert.Len(t, result.ChatHistory, 1)

	mockCache.AssertExpectations(t)
	mockEncryptor.AssertExpectations(t)
}

func TestService_GetSession_NotFound(t *testing.T) {
	// Arrange
	mockCache := &mocks.MockCacheClient{}
	mockEncryptor := &mocks.MockEncryptor{}

	cfg := &session.Config{
		CacheClient: mockCache,
		Encryptor:   mockEncryptor,
	}

	svc, err := session.NewService(cfg)
	require.NoError(t, err)

	mockCache.On("Get", mock.Anything, mock.AnythingOfType("string")).Return(nil, nil)

	// Act
	result, err := svc.GetSession(context.Background(), "tenant", "user", "conv")

	// Assert
	require.NoError(t, err)
	assert.Nil(t, result)

	mockCache.AssertExpectations(t)
}

func TestService_GetSession_CacheError(t *testing.T) {
	// Arrange
	mockCache := &mocks.MockCacheClient{}
	mockEncryptor := &mocks.MockEncryptor{}

	cfg := &session.Config{
		CacheClient: mockCache,
		Encryptor:   mockEncryptor,
	}

	svc, err := session.NewService(cfg)
	require.NoError(t, err)

	mockCache.On("Get", mock.Anything, mock.AnythingOfType("string")).Return(nil, errors.New("cache error"))

	// Act
	result, err := svc.GetSession(context.Background(), "tenant", "user", "conv")

	// Assert
	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "failed to get session from cache")

	mockCache.AssertExpectations(t)
}

func TestService_SetSession_Success(t *testing.T) {
	// Arrange
	mockCache := &mocks.MockCacheClient{}
	mockEncryptor := &mocks.MockEncryptor{}

	cfg := &session.Config{
		CacheClient: mockCache,
		Encryptor:   mockEncryptor,
		TTL:         5 * time.Minute,
	}

	svc, err := session.NewService(cfg)
	require.NoError(t, err)

	sessionData := &session.SessionData{
		Config:         &platform.AgentConfig{TenantID: "tenant-123"},
		TenantID:       "tenant-123",
		UserID:         "user-456",
		ConversationID: "conv-789",
	}

	mockEncryptor.On("Encrypt", mock.AnythingOfType("[]uint8")).Return("encrypted-data", nil)
	mockCache.On("Set", mock.Anything, mock.AnythingOfType("string"), mock.Anything, mock.AnythingOfType("time.Duration")).Return(nil)

	// Act
	err = svc.SetSession(context.Background(), sessionData)

	// Assert
	require.NoError(t, err)
	assert.NotZero(t, sessionData.UpdatedAt)

	mockEncryptor.AssertExpectations(t)
	mockCache.AssertExpectations(t)
}

func TestService_SetSession_NilSession(t *testing.T) {
	// Arrange
	mockCache := &mocks.MockCacheClient{}
	mockEncryptor := &mocks.MockEncryptor{}

	cfg := &session.Config{
		CacheClient: mockCache,
		Encryptor:   mockEncryptor,
	}

	svc, err := session.NewService(cfg)
	require.NoError(t, err)

	// Act
	err = svc.SetSession(context.Background(), nil)

	// Assert
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "session is required")
}

func TestService_DeleteSession_Success(t *testing.T) {
	// Arrange
	mockCache := &mocks.MockCacheClient{}
	mockEncryptor := &mocks.MockEncryptor{}

	cfg := &session.Config{
		CacheClient: mockCache,
		Encryptor:   mockEncryptor,
	}

	svc, err := session.NewService(cfg)
	require.NoError(t, err)

	mockCache.On("Delete", mock.Anything, mock.AnythingOfType("string")).Return(true, nil)

	// Act
	err = svc.DeleteSession(context.Background(), "tenant", "user", "conv")

	// Assert
	require.NoError(t, err)

	mockCache.AssertExpectations(t)
}

func TestService_BuildCacheKey(t *testing.T) {
	// Arrange
	mockCache := &mocks.MockCacheClient{}
	mockEncryptor := &mocks.MockEncryptor{}

	cfg := &session.Config{
		CacheClient: mockCache,
		Encryptor:   mockEncryptor,
	}

	svc, err := session.NewService(cfg)
	require.NoError(t, err)

	// Act
	key := svc.BuildCacheKey("tenant-123", "user-456", "conv-789")

	// Assert
	assert.Equal(t, "session:tenant-123:user-456:conv-789", key)
}

func TestService_UpdateChatHistory_Success(t *testing.T) {
	// Arrange
	mockCache := &mocks.MockCacheClient{}
	mockEncryptor := &mocks.MockEncryptor{}

	cfg := &session.Config{
		CacheClient: mockCache,
		Encryptor:   mockEncryptor,
	}

	svc, err := session.NewService(cfg)
	require.NoError(t, err)

	tenantID := "tenant-123"
	userID := "user-456"
	conversationID := "conv-789"

	existingSession := &session.SessionData{
		Config: &platform.AgentConfig{
			TenantID: tenantID,
			Settings: platform.AgentSettings{
				ChatHistoryCount: 5, // Max 5 entries
			},
		},
		ChatHistory: []models.ChatHistoryEntry{
			{Role: models.MessageTypeUser, Content: "Old 1"},
			{Role: models.MessageTypeAssistant, Content: "Old 2"},
		},
		TenantID:       tenantID,
		UserID:         userID,
		ConversationID: conversationID,
	}

	newEntries := []models.ChatHistoryEntry{
		{Role: models.MessageTypeUser, Content: "New 1"},
		{Role: models.MessageTypeAssistant, Content: "New 2"},
	}

	jsonData, _ := json.Marshal(existingSession)
	encryptedData := "encrypted-data"

	mockCache.On("Get", mock.Anything, mock.AnythingOfType("string")).Return([]byte(encryptedData), nil)
	mockEncryptor.On("Decrypt", encryptedData).Return(jsonData, nil)
	mockEncryptor.On("Encrypt", mock.AnythingOfType("[]uint8")).Return("new-encrypted-data", nil)
	mockCache.On("Set", mock.Anything, mock.AnythingOfType("string"), mock.Anything, mock.AnythingOfType("time.Duration")).Return(nil)

	// Act
	err = svc.UpdateChatHistory(context.Background(), tenantID, userID, conversationID, newEntries)

	// Assert
	require.NoError(t, err)

	mockCache.AssertExpectations(t)
	mockEncryptor.AssertExpectations(t)
}

func TestService_UpdateChatHistory_SessionNotFound(t *testing.T) {
	// Arrange
	mockCache := &mocks.MockCacheClient{}
	mockEncryptor := &mocks.MockEncryptor{}

	cfg := &session.Config{
		CacheClient: mockCache,
		Encryptor:   mockEncryptor,
	}

	svc, err := session.NewService(cfg)
	require.NoError(t, err)

	mockCache.On("Get", mock.Anything, mock.AnythingOfType("string")).Return(nil, nil)

	// Act
	err = svc.UpdateChatHistory(context.Background(), "tenant", "user", "conv", nil)

	// Assert
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "session not found")

	mockCache.AssertExpectations(t)
}
