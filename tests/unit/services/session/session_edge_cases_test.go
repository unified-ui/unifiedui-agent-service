// Package session_test provides additional edge case tests for session service.
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

// =============================================================================
// SetSession Edge Cases
// =============================================================================

func TestSetSession_EncryptError(t *testing.T) {
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

	sessionData := &session.Data{
		TenantID:       "tenant-123",
		UserID:         "user-456",
		ConversationID: "conv-789",
	}

	// Mock encrypt to return error
	mockEncryptor.On("Encrypt", mock.AnythingOfType("[]uint8")).Return("", errors.New("encryption failed"))

	// Act
	err = svc.SetSession(context.Background(), sessionData)

	// Assert
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to encrypt session")

	mockEncryptor.AssertExpectations(t)
}

func TestSetSession_CacheSetError(t *testing.T) {
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

	sessionData := &session.Data{
		TenantID:       "tenant-123",
		UserID:         "user-456",
		ConversationID: "conv-789",
	}

	// Mock encrypt succeeds, cache Set fails
	mockEncryptor.On("Encrypt", mock.AnythingOfType("[]uint8")).Return("encrypted-data", nil)
	mockCache.On("Set", mock.Anything, mock.AnythingOfType("string"), mock.Anything, mock.AnythingOfType("time.Duration")).
		Return(errors.New("redis connection refused"))

	// Act
	err = svc.SetSession(context.Background(), sessionData)

	// Assert
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to store session in cache")

	mockEncryptor.AssertExpectations(t)
	mockCache.AssertExpectations(t)
}

func TestSetSession_SetsCreatedAtWhenZero(t *testing.T) {
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

	// Session with zero CreatedAt
	sessionData := &session.Data{
		TenantID:       "tenant-123",
		UserID:         "user-456",
		ConversationID: "conv-789",
		CreatedAt:      time.Time{}, // Zero value
	}

	beforeSet := time.Now().UTC()

	mockEncryptor.On("Encrypt", mock.AnythingOfType("[]uint8")).Return("encrypted-data", nil)
	mockCache.On("Set", mock.Anything, mock.AnythingOfType("string"), mock.Anything, mock.AnythingOfType("time.Duration")).Return(nil)

	// Act
	err = svc.SetSession(context.Background(), sessionData)

	// Assert
	require.NoError(t, err)
	// CreatedAt should have been set
	assert.True(t, sessionData.CreatedAt.After(beforeSet) || sessionData.CreatedAt.Equal(beforeSet))
	// UpdatedAt should equal CreatedAt for new sessions
	assert.Equal(t, sessionData.CreatedAt, sessionData.UpdatedAt)
}

func TestSetSession_UpdatesUpdatedAtWhenCreatedAtExists(t *testing.T) {
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

	existingCreatedAt := time.Now().UTC().Add(-1 * time.Hour)

	// Session with existing CreatedAt
	sessionData := &session.Data{
		TenantID:       "tenant-123",
		UserID:         "user-456",
		ConversationID: "conv-789",
		CreatedAt:      existingCreatedAt,
	}

	mockEncryptor.On("Encrypt", mock.AnythingOfType("[]uint8")).Return("encrypted-data", nil)
	mockCache.On("Set", mock.Anything, mock.AnythingOfType("string"), mock.Anything, mock.AnythingOfType("time.Duration")).Return(nil)

	// Act
	err = svc.SetSession(context.Background(), sessionData)

	// Assert
	require.NoError(t, err)
	// CreatedAt should remain unchanged
	assert.Equal(t, existingCreatedAt, sessionData.CreatedAt)
	// UpdatedAt should be updated (after CreatedAt)
	assert.True(t, sessionData.UpdatedAt.After(sessionData.CreatedAt))
}

// =============================================================================
// UpdateChatHistory Edge Cases
// =============================================================================

func TestUpdateChatHistory_GetSessionError(t *testing.T) {
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

	// Cache returns an error
	mockCache.On("Get", mock.Anything, mock.AnythingOfType("string")).
		Return(nil, errors.New("cache error"))

	// Act
	err = svc.UpdateChatHistory(context.Background(), "tenant", "user", "conv", nil)

	// Assert
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to get session for update")

	mockCache.AssertExpectations(t)
}

func TestUpdateChatHistory_SetSessionError(t *testing.T) {
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

	existingSession := &session.Data{
		Config:         &platform.AgentConfig{Settings: platform.AgentSettings{ChatHistoryCount: 10}},
		ChatHistory:    []models.ChatHistoryEntry{},
		TenantID:       "tenant",
		UserID:         "user",
		ConversationID: "conv",
	}

	jsonData, _ := json.Marshal(existingSession)

	// Get succeeds
	mockCache.On("Get", mock.Anything, mock.AnythingOfType("string")).Return([]byte("encrypted"), nil)
	mockEncryptor.On("Decrypt", "encrypted").Return(jsonData, nil)

	// Set fails (encrypt error)
	mockEncryptor.On("Encrypt", mock.AnythingOfType("[]uint8")).Return("", errors.New("encrypt error"))

	// Act
	newEntries := []models.ChatHistoryEntry{{Role: models.MessageTypeUser, Content: "New"}}
	err = svc.UpdateChatHistory(context.Background(), "tenant", "user", "conv", newEntries)

	// Assert
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to encrypt session")

	mockCache.AssertExpectations(t)
	mockEncryptor.AssertExpectations(t)
}

func TestUpdateChatHistory_UsesDefaultHistoryCount_WhenNilConfig(t *testing.T) {
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

	// Session with nil Config - should use DefaultChatHistoryCount (30)
	existingSession := &session.Data{
		Config:         nil, // nil config
		ChatHistory:    []models.ChatHistoryEntry{},
		TenantID:       "tenant",
		UserID:         "user",
		ConversationID: "conv",
	}

	jsonData, _ := json.Marshal(existingSession)

	mockCache.On("Get", mock.Anything, mock.AnythingOfType("string")).Return([]byte("encrypted"), nil)
	mockEncryptor.On("Decrypt", "encrypted").Return(jsonData, nil)
	mockEncryptor.On("Encrypt", mock.AnythingOfType("[]uint8")).Return("new-encrypted", nil)
	mockCache.On("Set", mock.Anything, mock.AnythingOfType("string"), mock.Anything, mock.AnythingOfType("time.Duration")).Return(nil)

	// Act
	newEntries := []models.ChatHistoryEntry{{Role: models.MessageTypeUser, Content: "New"}}
	err = svc.UpdateChatHistory(context.Background(), "tenant", "user", "conv", newEntries)

	// Assert
	require.NoError(t, err)

	mockCache.AssertExpectations(t)
	mockEncryptor.AssertExpectations(t)
}

func TestUpdateChatHistory_UsesDefaultHistoryCount_WhenZeroInConfig(t *testing.T) {
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

	// Session with Config but ChatHistoryCount = 0 - should use default (30)
	existingSession := &session.Data{
		Config: &platform.AgentConfig{
			Settings: platform.AgentSettings{
				ChatHistoryCount: 0, // Zero means use default
			},
		},
		ChatHistory:    []models.ChatHistoryEntry{},
		TenantID:       "tenant",
		UserID:         "user",
		ConversationID: "conv",
	}

	jsonData, _ := json.Marshal(existingSession)

	mockCache.On("Get", mock.Anything, mock.AnythingOfType("string")).Return([]byte("encrypted"), nil)
	mockEncryptor.On("Decrypt", "encrypted").Return(jsonData, nil)
	mockEncryptor.On("Encrypt", mock.AnythingOfType("[]uint8")).Return("new-encrypted", nil)
	mockCache.On("Set", mock.Anything, mock.AnythingOfType("string"), mock.Anything, mock.AnythingOfType("time.Duration")).Return(nil)

	// Act
	newEntries := []models.ChatHistoryEntry{{Role: models.MessageTypeUser, Content: "New"}}
	err = svc.UpdateChatHistory(context.Background(), "tenant", "user", "conv", newEntries)

	// Assert
	require.NoError(t, err)

	mockCache.AssertExpectations(t)
	mockEncryptor.AssertExpectations(t)
}

func TestUpdateChatHistory_DoesNotTrimWhenUnderLimit(t *testing.T) {
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

	existingSession := &session.Data{
		Config: &platform.AgentConfig{
			Settings: platform.AgentSettings{
				ChatHistoryCount: 10,
			},
		},
		ChatHistory: []models.ChatHistoryEntry{
			{Role: models.MessageTypeUser, Content: "Msg1"},
			{Role: models.MessageTypeAssistant, Content: "Msg2"},
		},
		TenantID:       "tenant",
		UserID:         "user",
		ConversationID: "conv",
	}

	jsonData, _ := json.Marshal(existingSession)

	var savedData []byte
	mockCache.On("Get", mock.Anything, mock.AnythingOfType("string")).Return([]byte("encrypted"), nil)
	mockEncryptor.On("Decrypt", "encrypted").Return(jsonData, nil)
	mockEncryptor.On("Encrypt", mock.AnythingOfType("[]uint8")).Run(func(args mock.Arguments) {
		savedData = args.Get(0).([]byte)
	}).Return("new-encrypted", nil)
	mockCache.On("Set", mock.Anything, mock.AnythingOfType("string"), mock.Anything, mock.AnythingOfType("time.Duration")).Return(nil)

	// Act - add 2 new entries (total 4, under limit of 10)
	newEntries := []models.ChatHistoryEntry{
		{Role: models.MessageTypeUser, Content: "New1"},
		{Role: models.MessageTypeAssistant, Content: "New2"},
	}
	err = svc.UpdateChatHistory(context.Background(), "tenant", "user", "conv", newEntries)

	// Assert
	require.NoError(t, err)

	// Verify all 4 entries are preserved
	var saved session.Data
	err = json.Unmarshal(savedData, &saved)
	require.NoError(t, err)
	assert.Len(t, saved.ChatHistory, 4)

	mockCache.AssertExpectations(t)
	mockEncryptor.AssertExpectations(t)
}

func TestUpdateChatHistory_TrimsOldestWhenOverLimit(t *testing.T) {
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

	// Start with 4 entries, limit is 5
	existingSession := &session.Data{
		Config: &platform.AgentConfig{
			Settings: platform.AgentSettings{
				ChatHistoryCount: 5,
			},
		},
		ChatHistory: []models.ChatHistoryEntry{
			{Role: models.MessageTypeUser, Content: "Oldest"},
			{Role: models.MessageTypeAssistant, Content: "OldResponse"},
			{Role: models.MessageTypeUser, Content: "Middle"},
			{Role: models.MessageTypeAssistant, Content: "MiddleResponse"},
		},
		TenantID:       "tenant",
		UserID:         "user",
		ConversationID: "conv",
	}

	jsonData, _ := json.Marshal(existingSession)

	var savedData []byte
	mockCache.On("Get", mock.Anything, mock.AnythingOfType("string")).Return([]byte("encrypted"), nil)
	mockEncryptor.On("Decrypt", "encrypted").Return(jsonData, nil)
	mockEncryptor.On("Encrypt", mock.AnythingOfType("[]uint8")).Run(func(args mock.Arguments) {
		savedData = args.Get(0).([]byte)
	}).Return("new-encrypted", nil)
	mockCache.On("Set", mock.Anything, mock.AnythingOfType("string"), mock.Anything, mock.AnythingOfType("time.Duration")).Return(nil)

	// Act - add 3 new entries (total would be 7, trimmed to 5)
	newEntries := []models.ChatHistoryEntry{
		{Role: models.MessageTypeUser, Content: "New1"},
		{Role: models.MessageTypeAssistant, Content: "New2"},
		{Role: models.MessageTypeUser, Content: "Newest"},
	}
	err = svc.UpdateChatHistory(context.Background(), "tenant", "user", "conv", newEntries)

	// Assert
	require.NoError(t, err)

	var saved session.Data
	err = json.Unmarshal(savedData, &saved)
	require.NoError(t, err)

	// Should have exactly 5 entries (the limit)
	assert.Len(t, saved.ChatHistory, 5)
	// Oldest entries should be trimmed, newest should remain
	assert.Equal(t, "Middle", saved.ChatHistory[0].Content) // First 2 oldest removed
	assert.Equal(t, "Newest", saved.ChatHistory[4].Content)

	mockCache.AssertExpectations(t)
	mockEncryptor.AssertExpectations(t)
}

// =============================================================================
// DeleteSession Edge Cases
// =============================================================================

func TestDeleteSession_CacheError(t *testing.T) {
	// Arrange
	mockCache := &mocks.MockCacheClient{}
	mockEncryptor := &mocks.MockEncryptor{}

	cfg := &session.Config{
		CacheClient: mockCache,
		Encryptor:   mockEncryptor,
	}

	svc, err := session.NewService(cfg)
	require.NoError(t, err)

	mockCache.On("Delete", mock.Anything, mock.AnythingOfType("string")).
		Return(false, errors.New("redis error"))

	// Act
	err = svc.DeleteSession(context.Background(), "tenant", "user", "conv")

	// Assert
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to delete session")

	mockCache.AssertExpectations(t)
}

func TestDeleteSession_NonExistent_NoError(t *testing.T) {
	// Arrange
	mockCache := &mocks.MockCacheClient{}
	mockEncryptor := &mocks.MockEncryptor{}

	cfg := &session.Config{
		CacheClient: mockCache,
		Encryptor:   mockEncryptor,
	}

	svc, err := session.NewService(cfg)
	require.NoError(t, err)

	// Delete returns false (nothing deleted) but no error
	mockCache.On("Delete", mock.Anything, mock.AnythingOfType("string")).Return(false, nil)

	// Act
	err = svc.DeleteSession(context.Background(), "tenant", "user", "conv")

	// Assert - should succeed even if nothing was deleted
	assert.NoError(t, err)

	mockCache.AssertExpectations(t)
}

// =============================================================================
// BuildCacheKey Tests
// =============================================================================

func TestBuildCacheKey_Format(t *testing.T) {
	// Arrange
	mockCache := &mocks.MockCacheClient{}
	mockEncryptor := &mocks.MockEncryptor{}

	cfg := &session.Config{
		CacheClient: mockCache,
		Encryptor:   mockEncryptor,
	}

	svc, err := session.NewService(cfg)
	require.NoError(t, err)

	// Test various key combinations
	testCases := []struct {
		tenant   string
		user     string
		conv     string
		expected string
	}{
		{"t1", "u1", "c1", "session:t1:u1:c1"},
		{"tenant-abc", "user-xyz", "conv-123", "session:tenant-abc:user-xyz:conv-123"},
		{"", "", "", "session:::"},
		{"a", "b", "c", "session:a:b:c"},
	}

	for _, tc := range testCases {
		t.Run(tc.expected, func(t *testing.T) {
			key := svc.BuildCacheKey(tc.tenant, tc.user, tc.conv)
			assert.Equal(t, tc.expected, key)
		})
	}
}

// =============================================================================
// NewSessionData Tests
// =============================================================================

func TestNewData_AllFieldsPopulated(t *testing.T) {
	// Arrange
	config := &platform.AgentConfig{
		TenantID: "tenant-123",
		Settings: platform.AgentSettings{
			ChatHistoryCount: 50,
		},
	}
	history := []models.ChatHistoryEntry{
		{Role: models.MessageTypeUser, Content: "Hello"},
		{Role: models.MessageTypeAssistant, Content: "Hi there"},
	}
	tenantID := "tenant-123"
	userID := "user-456"
	conversationID := "conv-789"

	before := time.Now().UTC()

	// Act
	sd := session.NewData(config, history, tenantID, userID, conversationID)

	after := time.Now().UTC()

	// Assert
	assert.Equal(t, config, sd.Config)
	assert.Equal(t, history, sd.ChatHistory)
	assert.Equal(t, tenantID, sd.TenantID)
	assert.Equal(t, userID, sd.UserID)
	assert.Equal(t, conversationID, sd.ConversationID)

	// Check timestamps are reasonable
	assert.True(t, sd.CreatedAt.After(before) || sd.CreatedAt.Equal(before))
	assert.True(t, sd.CreatedAt.Before(after) || sd.CreatedAt.Equal(after))
	assert.Equal(t, sd.CreatedAt, sd.UpdatedAt)
}

func TestNewData_NilConfig(t *testing.T) {
	// Act
	sd := session.NewData(nil, nil, "t", "u", "c")

	// Assert
	assert.Nil(t, sd.Config)
	assert.Nil(t, sd.ChatHistory)
}

func TestNewData_EmptyHistory(t *testing.T) {
	// Act
	sd := session.NewData(nil, []models.ChatHistoryEntry{}, "t", "u", "c")

	// Assert
	assert.Empty(t, sd.ChatHistory)
	assert.Len(t, sd.ChatHistory, 0)
}
