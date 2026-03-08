package configcache_test

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/unifiedui/agent-service/internal/services/configcache"
	"github.com/unifiedui/agent-service/internal/services/platform"
	"github.com/unifiedui/agent-service/tests/mocks"
)

// =============================================================================
// Set - Encrypt Error
// =============================================================================

func TestSet_EncryptError(t *testing.T) {
	mockCache := &mocks.MockCacheClient{}
	mockEncryptor := &mocks.MockEncryptor{}

	svc, err := configcache.NewService(&configcache.Config{
		CacheClient: mockCache,
		Encryptor:   mockEncryptor,
		TTL:         5 * time.Minute,
	})
	require.NoError(t, err)

	config := &platform.AgentConfig{
		Type:        platform.AgentTypeN8N,
		TenantID:    "t1",
		ChatAgentID: "a1",
	}

	mockEncryptor.On("Encrypt", mock.AnythingOfType("[]uint8")).Return("", errors.New("encryption failed"))

	err = svc.Set(context.Background(), "t1", "u1", "a1", config)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to encrypt config")

	mockEncryptor.AssertExpectations(t)
}

// =============================================================================
// Set - Cache Set Error
// =============================================================================

func TestSet_CacheSetError(t *testing.T) {
	mockCache := &mocks.MockCacheClient{}
	mockEncryptor := &mocks.MockEncryptor{}

	svc, err := configcache.NewService(&configcache.Config{
		CacheClient: mockCache,
		Encryptor:   mockEncryptor,
		TTL:         5 * time.Minute,
	})
	require.NoError(t, err)

	config := &platform.AgentConfig{
		Type:        platform.AgentTypeN8N,
		TenantID:    "t1",
		ChatAgentID: "a1",
	}

	mockEncryptor.On("Encrypt", mock.AnythingOfType("[]uint8")).Return("encrypted-data", nil)
	mockCache.On("Set", mock.Anything, mock.AnythingOfType("string"), mock.Anything, mock.AnythingOfType("time.Duration")).
		Return(errors.New("redis connection refused"))

	err = svc.Set(context.Background(), "t1", "u1", "a1", config)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to store config in cache")

	mockEncryptor.AssertExpectations(t)
	mockCache.AssertExpectations(t)
}

// =============================================================================
// Get - Decrypt Error With MockEncryptor
// =============================================================================

func TestGet_DecryptError_WithMockEncryptor(t *testing.T) {
	mockCache := &mocks.MockCacheClient{}
	mockEncryptor := &mocks.MockEncryptor{}

	svc, err := configcache.NewService(&configcache.Config{
		CacheClient: mockCache,
		Encryptor:   mockEncryptor,
		TTL:         5 * time.Minute,
	})
	require.NoError(t, err)

	ctx := context.Background()
	cacheKey := "config:t1:u1:a1"

	mockCache.On("Get", ctx, cacheKey).Return([]byte("some-encrypted-data"), nil)
	mockEncryptor.On("Decrypt", "some-encrypted-data").Return(nil, errors.New("decryption failed"))
	mockCache.On("Delete", ctx, cacheKey).Return(true, nil)

	result, err := svc.Get(ctx, "t1", "u1", "a1")
	require.NoError(t, err)
	assert.Nil(t, result)

	mockCache.AssertCalled(t, "Delete", ctx, cacheKey)
}

// =============================================================================
// Get - Unmarshal Error With MockEncryptor
// =============================================================================

func TestGet_UnmarshalError_WithMockEncryptor(t *testing.T) {
	mockCache := &mocks.MockCacheClient{}
	mockEncryptor := &mocks.MockEncryptor{}

	svc, err := configcache.NewService(&configcache.Config{
		CacheClient: mockCache,
		Encryptor:   mockEncryptor,
		TTL:         5 * time.Minute,
	})
	require.NoError(t, err)

	ctx := context.Background()
	cacheKey := "config:t1:u1:a1"

	mockCache.On("Get", ctx, cacheKey).Return([]byte("some-encrypted-data"), nil)
	mockEncryptor.On("Decrypt", "some-encrypted-data").Return([]byte("not-valid-json{{{"), nil)
	mockCache.On("Delete", ctx, cacheKey).Return(true, nil)

	result, err := svc.Get(ctx, "t1", "u1", "a1")
	require.NoError(t, err)
	assert.Nil(t, result)

	mockCache.AssertCalled(t, "Delete", ctx, cacheKey)
}

// =============================================================================
// Set and Get - Roundtrip With Full AgentConfig
// =============================================================================

func TestSetAndGet_FullAgentConfig(t *testing.T) {
	mockCache := &mocks.MockCacheClient{}
	mockEncryptor := &mocks.MockEncryptor{}

	svc, err := configcache.NewService(&configcache.Config{
		CacheClient: mockCache,
		Encryptor:   mockEncryptor,
		TTL:         5 * time.Minute,
	})
	require.NoError(t, err)

	ctx := context.Background()

	config := &platform.AgentConfig{
		Type:        platform.AgentTypeFoundry,
		TenantID:    "tenant-abc",
		ChatAgentID: "agent-xyz",
		Settings: platform.AgentSettings{
			ProjectEndpoint:       "https://project.api.azure.com",
			AgentName:             "test-agent",
			ChatHistoryCount:      50,
			UseUnifiedChatHistory: true,
		},
	}

	configJSON, _ := json.Marshal(config)

	mockEncryptor.On("Encrypt", configJSON).Return("encrypted-payload", nil)
	mockCache.On("Set", ctx, "config:tenant-abc:user-123:agent-xyz", []byte("encrypted-payload"), 5*time.Minute).
		Return(nil)

	err = svc.Set(ctx, "tenant-abc", "user-123", "agent-xyz", config)
	require.NoError(t, err)

	mockCache.On("Get", ctx, "config:tenant-abc:user-123:agent-xyz").Return([]byte("encrypted-payload"), nil)
	mockEncryptor.On("Decrypt", "encrypted-payload").Return(configJSON, nil)

	retrieved, err := svc.Get(ctx, "tenant-abc", "user-123", "agent-xyz")
	require.NoError(t, err)
	require.NotNil(t, retrieved)
	assert.Equal(t, platform.AgentTypeFoundry, retrieved.Type)
	assert.Equal(t, "tenant-abc", retrieved.TenantID)
	assert.Equal(t, "agent-xyz", retrieved.ChatAgentID)
	assert.Equal(t, "https://project.api.azure.com", retrieved.Settings.ProjectEndpoint)
	assert.Equal(t, "test-agent", retrieved.Settings.AgentName)
	assert.Equal(t, 50, retrieved.Settings.ChatHistoryCount)
	assert.True(t, retrieved.Settings.UseUnifiedChatHistory)
}

// =============================================================================
// Cache Isolation - Different Users
// =============================================================================

func TestCacheIsolation_DifferentUsers(t *testing.T) {
	mockCache := &mocks.MockCacheClient{}
	mockEncryptor := &mocks.MockEncryptor{}

	svc, err := configcache.NewService(&configcache.Config{
		CacheClient: mockCache,
		Encryptor:   mockEncryptor,
		TTL:         5 * time.Minute,
	})
	require.NoError(t, err)

	ctx := context.Background()

	configA := &platform.AgentConfig{Type: platform.AgentTypeN8N, TenantID: "t1", ChatAgentID: "a1"}
	configB := &platform.AgentConfig{Type: platform.AgentTypeFoundry, TenantID: "t1", ChatAgentID: "a1"}

	jsonA, _ := json.Marshal(configA)
	jsonB, _ := json.Marshal(configB)

	mockEncryptor.On("Encrypt", jsonA).Return("encrypted-A", nil)
	mockEncryptor.On("Encrypt", jsonB).Return("encrypted-B", nil)
	mockCache.On("Set", ctx, "config:t1:userA:a1", []byte("encrypted-A"), 5*time.Minute).Return(nil)
	mockCache.On("Set", ctx, "config:t1:userB:a1", []byte("encrypted-B"), 5*time.Minute).Return(nil)

	require.NoError(t, svc.Set(ctx, "t1", "userA", "a1", configA))
	require.NoError(t, svc.Set(ctx, "t1", "userB", "a1", configB))

	mockCache.On("Get", ctx, "config:t1:userA:a1").Return([]byte("encrypted-A"), nil)
	mockEncryptor.On("Decrypt", "encrypted-A").Return(jsonA, nil)

	mockCache.On("Get", ctx, "config:t1:userB:a1").Return([]byte("encrypted-B"), nil)
	mockEncryptor.On("Decrypt", "encrypted-B").Return(jsonB, nil)

	resultA, err := svc.Get(ctx, "t1", "userA", "a1")
	require.NoError(t, err)
	assert.Equal(t, platform.AgentTypeN8N, resultA.Type)

	resultB, err := svc.Get(ctx, "t1", "userB", "a1")
	require.NoError(t, err)
	assert.Equal(t, platform.AgentTypeFoundry, resultB.Type)
}

// =============================================================================
// Delete - Correct Key Used
// =============================================================================

func TestDelete_CorrectKeyFormat(t *testing.T) {
	mockCache := &mocks.MockCacheClient{}
	mockEncryptor := &mocks.MockEncryptor{}

	svc, err := configcache.NewService(&configcache.Config{
		CacheClient: mockCache,
		Encryptor:   mockEncryptor,
		TTL:         5 * time.Minute,
	})
	require.NoError(t, err)

	ctx := context.Background()
	mockCache.On("Delete", ctx, "config:t1:u1:a1").Return(true, nil)

	err = svc.Delete(ctx, "t1", "u1", "a1")
	require.NoError(t, err)
	mockCache.AssertCalled(t, "Delete", ctx, "config:t1:u1:a1")
}

// =============================================================================
// DeleteByTenant - Pattern Format
// =============================================================================

func TestDeleteByTenant_PatternFormat(t *testing.T) {
	mockCache := &mocks.MockCacheClient{}
	mockEncryptor := &mocks.MockEncryptor{}

	svc, err := configcache.NewService(&configcache.Config{
		CacheClient: mockCache,
		Encryptor:   mockEncryptor,
		TTL:         5 * time.Minute,
	})
	require.NoError(t, err)

	ctx := context.Background()
	mockCache.On("DeletePattern", ctx, "config:my-tenant:*").Return(int64(3), nil)

	err = svc.DeleteByTenant(ctx, "my-tenant")
	require.NoError(t, err)
	mockCache.AssertCalled(t, "DeletePattern", ctx, "config:my-tenant:*")
}
