package session_test

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/unifiedui/agent-service/internal/domain/models"
	"github.com/unifiedui/agent-service/internal/pkg/encryption"
	"github.com/unifiedui/agent-service/internal/services/platform"
	"github.com/unifiedui/agent-service/internal/services/session"
	"github.com/unifiedui/agent-service/tests/mocks"
)

func setupSessionService(t *testing.T) (session.Service, *mocks.MockCacheClient) {
	t.Helper()
	mockCache := mocks.NewMockCacheClient()
	enc := encryption.NewNoOpEncryptor()

	svc, err := session.NewService(&session.Config{
		CacheClient: mockCache,
		Encryptor:   enc,
		TTL:         5 * time.Minute,
	})
	require.NoError(t, err)
	return svc, mockCache
}

func TestNewService_NilConfig(t *testing.T) {
	_, err := session.NewService(nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "config is required")
}

func TestNewService_NilCacheClient(t *testing.T) {
	_, err := session.NewService(&session.Config{
		Encryptor: encryption.NewNoOpEncryptor(),
	})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "cache client is required")
}

func TestNewService_NilEncryptor(t *testing.T) {
	_, err := session.NewService(&session.Config{
		CacheClient: mocks.NewMockCacheClient(),
	})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "encryptor is required")
}

func TestNewService_DefaultTTL(t *testing.T) {
	svc, err := session.NewService(&session.Config{
		CacheClient: mocks.NewMockCacheClient(),
		Encryptor:   encryption.NewNoOpEncryptor(),
	})
	require.NoError(t, err)
	assert.NotNil(t, svc)
}

func TestGetSession_NotFound(t *testing.T) {
	svc, mockCache := setupSessionService(t)
	ctx := context.Background()

	mockCache.On("Get", ctx, mock.AnythingOfType("string")).Return(nil, nil)

	result, err := svc.GetSession(ctx, "t1", "u1", "c1")
	require.NoError(t, err)
	assert.Nil(t, result)
}

func TestGetSession_CacheError(t *testing.T) {
	svc, mockCache := setupSessionService(t)
	ctx := context.Background()

	mockCache.On("Get", ctx, mock.AnythingOfType("string")).
		Return(nil, assert.AnError)

	_, err := svc.GetSession(ctx, "t1", "u1", "c1")
	assert.Error(t, err)
}

func TestSetSession_Success(t *testing.T) {
	svc, mockCache := setupSessionService(t)
	ctx := context.Background()

	mockCache.On("Set", ctx, mock.AnythingOfType("string"), mock.Anything, mock.Anything).
		Return(nil)

	sd := &session.SessionData{
		TenantID:       "t1",
		UserID:         "u1",
		ConversationID: "c1",
	}

	err := svc.SetSession(ctx, sd)
	require.NoError(t, err)
	mockCache.AssertCalled(t, "Set", ctx, mock.AnythingOfType("string"), mock.Anything, mock.Anything)
}

func TestSetSession_NilSession(t *testing.T) {
	svc, _ := setupSessionService(t)
	ctx := context.Background()
	err := svc.SetSession(ctx, nil)
	assert.Error(t, err)
}

func TestSetAndGetSession_Roundtrip(t *testing.T) {
	mockCache := mocks.NewMockCacheClient()
	enc := encryption.NewNoOpEncryptor()

	svc, err := session.NewService(&session.Config{
		CacheClient: mockCache,
		Encryptor:   enc,
		TTL:         5 * time.Minute,
	})
	require.NoError(t, err)

	ctx := context.Background()
	var storedData []byte

	mockCache.On("Set", ctx, mock.AnythingOfType("string"), mock.Anything, mock.Anything).
		Run(func(args mock.Arguments) {
			storedData = args.Get(2).([]byte)
		}).Return(nil)

	sd := session.NewSessionData(
		&platform.AgentConfig{Settings: platform.AgentSettings{ChatHistoryCount: 10}},
		[]models.ChatHistoryEntry{{Role: "user", Content: "hi"}},
		"t1", "u1", "c1",
	)

	err = svc.SetSession(ctx, sd)
	require.NoError(t, err)
	require.NotEmpty(t, storedData)

	mockCache.On("Get", ctx, mock.AnythingOfType("string")).Return(storedData, nil)

	retrieved, err := svc.GetSession(ctx, "t1", "u1", "c1")
	require.NoError(t, err)
	require.NotNil(t, retrieved)
	assert.Equal(t, "t1", retrieved.TenantID)
	assert.Equal(t, "u1", retrieved.UserID)
	assert.Equal(t, "c1", retrieved.ConversationID)
	assert.Len(t, retrieved.ChatHistory, 1)
}

func TestGetSession_DecryptionFailure_SkipsGracefully(t *testing.T) {
	mockCache := mocks.NewMockCacheClient()
	enc := encryption.NewNoOpEncryptor()

	svc, _ := session.NewService(&session.Config{
		CacheClient: mockCache,
		Encryptor:   enc,
		TTL:         5 * time.Minute,
	})

	ctx := context.Background()
	// Return invalid encrypted data
	mockCache.On("Get", ctx, mock.AnythingOfType("string")).Return([]byte("not-valid-base64!!!!"), nil)
	mockCache.On("Delete", ctx, mock.AnythingOfType("string")).Return(true, nil)

	result, err := svc.GetSession(ctx, "t1", "u1", "c1")
	require.NoError(t, err)
	assert.Nil(t, result)
}

func TestGetSession_InvalidJSON_SkipsGracefully(t *testing.T) {
	mockCache := mocks.NewMockCacheClient()
	enc := encryption.NewNoOpEncryptor()

	svc, _ := session.NewService(&session.Config{
		CacheClient: mockCache,
		Encryptor:   enc,
		TTL:         5 * time.Minute,
	})

	ctx := context.Background()
	// Encrypt valid bytes but invalid JSON
	encrypted, _ := enc.Encrypt([]byte("not-json"))
	mockCache.On("Get", ctx, mock.AnythingOfType("string")).Return([]byte(encrypted), nil)
	mockCache.On("Delete", ctx, mock.AnythingOfType("string")).Return(true, nil)

	result, err := svc.GetSession(ctx, "t1", "u1", "c1")
	require.NoError(t, err)
	assert.Nil(t, result)
}

func TestDeleteSession_Success(t *testing.T) {
	svc, mockCache := setupSessionService(t)
	ctx := context.Background()

	mockCache.On("Delete", ctx, mock.AnythingOfType("string")).Return(true, nil)

	err := svc.DeleteSession(ctx, "t1", "u1", "c1")
	require.NoError(t, err)
}

func TestDeleteSession_Error(t *testing.T) {
	svc, mockCache := setupSessionService(t)
	ctx := context.Background()

	mockCache.On("Delete", ctx, mock.AnythingOfType("string")).Return(false, assert.AnError)

	err := svc.DeleteSession(ctx, "t1", "u1", "c1")
	assert.Error(t, err)
}

func TestUpdateChatHistory_Success(t *testing.T) {
	mockCache := mocks.NewMockCacheClient()
	enc := encryption.NewNoOpEncryptor()

	svc, _ := session.NewService(&session.Config{
		CacheClient: mockCache,
		Encryptor:   enc,
		TTL:         5 * time.Minute,
	})

	ctx := context.Background()

	existing := &session.SessionData{
		Config:         &platform.AgentConfig{Settings: platform.AgentSettings{ChatHistoryCount: 5}},
		ChatHistory:    []models.ChatHistoryEntry{{Role: "user", Content: "msg1"}},
		TenantID:       "t1",
		UserID:         "u1",
		ConversationID: "c1",
		CreatedAt:      time.Now(),
	}

	existingJSON, _ := json.Marshal(existing)
	existingEncrypted, _ := enc.Encrypt(existingJSON)

	mockCache.On("Get", ctx, mock.AnythingOfType("string")).Return([]byte(existingEncrypted), nil)
	mockCache.On("Set", ctx, mock.AnythingOfType("string"), mock.Anything, mock.Anything).Return(nil)

	newEntries := []models.ChatHistoryEntry{
		{Role: "assistant", Content: "resp1"},
		{Role: "user", Content: "msg2"},
	}

	err := svc.UpdateChatHistory(ctx, "t1", "u1", "c1", newEntries)
	require.NoError(t, err)
}

func TestUpdateChatHistory_SessionNotFound(t *testing.T) {
	svc, mockCache := setupSessionService(t)
	ctx := context.Background()

	mockCache.On("Get", ctx, mock.AnythingOfType("string")).Return(nil, nil)

	err := svc.UpdateChatHistory(ctx, "t1", "u1", "c1", nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "session not found")
}

func TestUpdateChatHistory_TrimExcess(t *testing.T) {
	mockCache := mocks.NewMockCacheClient()
	enc := encryption.NewNoOpEncryptor()

	svc, _ := session.NewService(&session.Config{
		CacheClient: mockCache,
		Encryptor:   enc,
	})

	ctx := context.Background()

	history := make([]models.ChatHistoryEntry, 30)
	for i := range history {
		history[i] = models.ChatHistoryEntry{Role: "user", Content: "m"}
	}

	existing := &session.SessionData{
		Config:         &platform.AgentConfig{Settings: platform.AgentSettings{ChatHistoryCount: 5}},
		ChatHistory:    history,
		TenantID:       "t1",
		UserID:         "u1",
		ConversationID: "c1",
		CreatedAt:      time.Now(),
	}

	existingJSON, _ := json.Marshal(existing)
	existingEncrypted, _ := enc.Encrypt(existingJSON)

	mockCache.On("Get", ctx, mock.AnythingOfType("string")).Return([]byte(existingEncrypted), nil)

	var savedData []byte
	mockCache.On("Set", ctx, mock.AnythingOfType("string"), mock.Anything, mock.Anything).
		Run(func(args mock.Arguments) {
			savedData = args.Get(2).([]byte)
		}).Return(nil)

	err := svc.UpdateChatHistory(ctx, "t1", "u1", "c1", []models.ChatHistoryEntry{{Role: "user", Content: "new"}})
	require.NoError(t, err)
	require.NotEmpty(t, savedData)

	// Decrypt and check trimmed
	decrypted, _ := enc.Decrypt(string(savedData))
	var saved session.SessionData
	json.Unmarshal(decrypted, &saved)
	assert.LessOrEqual(t, len(saved.ChatHistory), 5)
}

func TestNewSessionData(t *testing.T) {
	config := &platform.AgentConfig{Settings: platform.AgentSettings{ChatHistoryCount: 10}}
	history := []models.ChatHistoryEntry{{Role: "user", Content: "hi"}}

	sd := session.NewSessionData(config, history, "t1", "u1", "c1")
	assert.Equal(t, "t1", sd.TenantID)
	assert.Equal(t, "u1", sd.UserID)
	assert.Equal(t, "c1", sd.ConversationID)
	assert.NotNil(t, sd.Config)
	assert.Len(t, sd.ChatHistory, 1)
	assert.False(t, sd.CreatedAt.IsZero())
	assert.False(t, sd.UpdatedAt.IsZero())
}
