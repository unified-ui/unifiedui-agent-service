package configcache_test

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/unifiedui/agent-service/internal/pkg/encryption"
	"github.com/unifiedui/agent-service/internal/services/configcache"
	"github.com/unifiedui/agent-service/internal/services/platform"
	"github.com/unifiedui/agent-service/tests/mocks"
)

func setupConfigCacheService(t *testing.T) (configcache.Service, *mocks.MockCacheClient) {
	t.Helper()
	mockCache := mocks.NewMockCacheClient()
	enc := encryption.NewNoOpEncryptor()

	svc, err := configcache.NewService(&configcache.Config{
		CacheClient: mockCache,
		Encryptor:   enc,
		TTL:         5 * time.Minute,
	})
	require.NoError(t, err)
	return svc, mockCache
}

// =============================================================================
// NewService Validation
// =============================================================================

func TestNewService_NilConfig(t *testing.T) {
	_, err := configcache.NewService(nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "config is required")
}

func TestNewService_NilCacheClient(t *testing.T) {
	_, err := configcache.NewService(&configcache.Config{
		Encryptor: encryption.NewNoOpEncryptor(),
	})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "cache client is required")
}

func TestNewService_NilEncryptor(t *testing.T) {
	_, err := configcache.NewService(&configcache.Config{
		CacheClient: mocks.NewMockCacheClient(),
	})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "encryptor is required")
}

func TestNewService_DefaultTTL(t *testing.T) {
	svc, err := configcache.NewService(&configcache.Config{
		CacheClient: mocks.NewMockCacheClient(),
		Encryptor:   encryption.NewNoOpEncryptor(),
	})
	require.NoError(t, err)
	assert.NotNil(t, svc)
}

func TestNewService_CustomTTL(t *testing.T) {
	svc, err := configcache.NewService(&configcache.Config{
		CacheClient: mocks.NewMockCacheClient(),
		Encryptor:   encryption.NewNoOpEncryptor(),
		TTL:         10 * time.Minute,
	})
	require.NoError(t, err)
	assert.NotNil(t, svc)
}

// =============================================================================
// BuildCacheKey
// =============================================================================

func TestBuildCacheKey_Format(t *testing.T) {
	svc, _ := setupConfigCacheService(t)
	key := svc.BuildCacheKey("tenant-1", "user-2", "agent-3")
	assert.Equal(t, "config:tenant-1:user-2:agent-3", key)
}

func TestBuildCacheKey_DifferentUsersDifferentKeys(t *testing.T) {
	svc, _ := setupConfigCacheService(t)

	key1 := svc.BuildCacheKey("tenant-1", "user-A", "agent-1")
	key2 := svc.BuildCacheKey("tenant-1", "user-B", "agent-1")

	assert.NotEqual(t, key1, key2)
	assert.Equal(t, "config:tenant-1:user-A:agent-1", key1)
	assert.Equal(t, "config:tenant-1:user-B:agent-1", key2)
}

func TestBuildCacheKey_DifferentAgentsDifferentKeys(t *testing.T) {
	svc, _ := setupConfigCacheService(t)

	key1 := svc.BuildCacheKey("tenant-1", "user-1", "agent-A")
	key2 := svc.BuildCacheKey("tenant-1", "user-1", "agent-B")

	assert.NotEqual(t, key1, key2)
}

// =============================================================================
// Get - Cache Miss
// =============================================================================

func TestGet_CacheMiss(t *testing.T) {
	svc, mockCache := setupConfigCacheService(t)
	ctx := context.Background()

	mockCache.On("Get", ctx, "config:t1:u1:a1").Return(nil, nil)

	result, err := svc.Get(ctx, "t1", "u1", "a1")
	require.NoError(t, err)
	assert.Nil(t, result)
}

// =============================================================================
// Set and Get - Roundtrip
// =============================================================================

func TestSetAndGet_Roundtrip(t *testing.T) {
	mockCache := mocks.NewMockCacheClient()
	enc := encryption.NewNoOpEncryptor()

	svc, err := configcache.NewService(&configcache.Config{
		CacheClient: mockCache,
		Encryptor:   enc,
		TTL:         5 * time.Minute,
	})
	require.NoError(t, err)

	ctx := context.Background()
	var storedData []byte

	agentConfig := &platform.AgentConfig{
		Type:        platform.AgentTypeN8N,
		TenantID:    "tenant-123",
		ChatAgentID: "agent-456",
		Settings: platform.AgentSettings{
			ChatHistoryCount:      20,
			UseUnifiedChatHistory: true,
		},
	}

	mockCache.On("Set", ctx, "config:tenant-123:user-789:agent-456", mock.Anything, 5*time.Minute).
		Run(func(args mock.Arguments) {
			storedData = args.Get(2).([]byte)
		}).Return(nil)

	err = svc.Set(ctx, "tenant-123", "user-789", "agent-456", agentConfig)
	require.NoError(t, err)
	require.NotEmpty(t, storedData)

	mockCache.On("Get", ctx, "config:tenant-123:user-789:agent-456").Return(storedData, nil)

	retrieved, err := svc.Get(ctx, "tenant-123", "user-789", "agent-456")
	require.NoError(t, err)
	require.NotNil(t, retrieved)
	assert.Equal(t, platform.AgentTypeN8N, retrieved.Type)
	assert.Equal(t, "tenant-123", retrieved.TenantID)
	assert.Equal(t, "agent-456", retrieved.ChatAgentID)
	assert.Equal(t, 20, retrieved.Settings.ChatHistoryCount)
	assert.True(t, retrieved.Settings.UseUnifiedChatHistory)
}

// =============================================================================
// Get - Decryption Failure (graceful fallback)
// =============================================================================

func TestGet_DecryptionFailure_ReturnsNilAndDeletesKey(t *testing.T) {
	mockCache := mocks.NewMockCacheClient()
	enc := encryption.NewNoOpEncryptor()

	svc, err := configcache.NewService(&configcache.Config{
		CacheClient: mockCache,
		Encryptor:   enc,
		TTL:         5 * time.Minute,
	})
	require.NoError(t, err)

	ctx := context.Background()

	mockCache.On("Get", ctx, mock.AnythingOfType("string")).Return([]byte("not-valid-base64!!!!"), nil)
	mockCache.On("Delete", ctx, mock.AnythingOfType("string")).Return(true, nil)

	result, err := svc.Get(ctx, "t1", "u1", "a1")
	require.NoError(t, err)
	assert.Nil(t, result)

	mockCache.AssertCalled(t, "Delete", ctx, "config:t1:u1:a1")
}

// =============================================================================
// Get - Invalid JSON (graceful fallback)
// =============================================================================

func TestGet_InvalidJSON_ReturnsNilAndDeletesKey(t *testing.T) {
	mockCache := mocks.NewMockCacheClient()
	enc := encryption.NewNoOpEncryptor()

	svc, err := configcache.NewService(&configcache.Config{
		CacheClient: mockCache,
		Encryptor:   enc,
		TTL:         5 * time.Minute,
	})
	require.NoError(t, err)

	ctx := context.Background()

	encrypted, _ := enc.Encrypt([]byte("not-json"))
	mockCache.On("Get", ctx, mock.AnythingOfType("string")).Return([]byte(encrypted), nil)
	mockCache.On("Delete", ctx, mock.AnythingOfType("string")).Return(true, nil)

	result, err := svc.Get(ctx, "t1", "u1", "a1")
	require.NoError(t, err)
	assert.Nil(t, result)

	mockCache.AssertCalled(t, "Delete", ctx, "config:t1:u1:a1")
}

// =============================================================================
// Get - Cache Error
// =============================================================================

func TestGet_CacheError(t *testing.T) {
	svc, mockCache := setupConfigCacheService(t)
	ctx := context.Background()

	mockCache.On("Get", ctx, mock.AnythingOfType("string")).
		Return(nil, assert.AnError)

	_, err := svc.Get(ctx, "t1", "u1", "a1")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to get config from cache")
}

// =============================================================================
// Set - Nil Config
// =============================================================================

func TestSet_NilConfig(t *testing.T) {
	svc, _ := setupConfigCacheService(t)
	ctx := context.Background()

	err := svc.Set(ctx, "t1", "u1", "a1", nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "config is required")
}

// =============================================================================
// Set - Success Verifies Key and TTL
// =============================================================================

func TestSet_UsesCorrectKeyAndTTL(t *testing.T) {
	mockCache := mocks.NewMockCacheClient()
	enc := encryption.NewNoOpEncryptor()
	ttl := 3 * time.Minute

	svc, err := configcache.NewService(&configcache.Config{
		CacheClient: mockCache,
		Encryptor:   enc,
		TTL:         ttl,
	})
	require.NoError(t, err)

	ctx := context.Background()
	config := &platform.AgentConfig{
		Type:        platform.AgentTypeN8N,
		TenantID:    "t1",
		ChatAgentID: "a1",
	}

	mockCache.On("Set", ctx, "config:t1:u1:a1", mock.Anything, ttl).Return(nil)

	err = svc.Set(ctx, "t1", "u1", "a1", config)
	require.NoError(t, err)
	mockCache.AssertCalled(t, "Set", ctx, "config:t1:u1:a1", mock.Anything, ttl)
}

// =============================================================================
// Set - Data is encrypted
// =============================================================================

func TestSet_DataIsEncrypted(t *testing.T) {
	mockCache := mocks.NewMockCacheClient()
	enc := encryption.NewNoOpEncryptor()

	svc, err := configcache.NewService(&configcache.Config{
		CacheClient: mockCache,
		Encryptor:   enc,
		TTL:         5 * time.Minute,
	})
	require.NoError(t, err)

	ctx := context.Background()
	config := &platform.AgentConfig{
		Type:        platform.AgentTypeN8N,
		TenantID:    "t1",
		ChatAgentID: "a1",
	}

	var storedBytes []byte
	mockCache.On("Set", ctx, mock.AnythingOfType("string"), mock.Anything, mock.Anything).
		Run(func(args mock.Arguments) {
			storedBytes = args.Get(2).([]byte)
		}).Return(nil)

	err = svc.Set(ctx, "t1", "u1", "a1", config)
	require.NoError(t, err)

	storedStr := string(storedBytes)

	rawJSON, _ := json.Marshal(config)
	assert.NotEqual(t, string(rawJSON), storedStr)

	decrypted, decErr := enc.Decrypt(storedStr)
	require.NoError(t, decErr)

	var decoded platform.AgentConfig
	require.NoError(t, json.Unmarshal(decrypted, &decoded))
	assert.Equal(t, config.Type, decoded.Type)
	assert.Equal(t, config.TenantID, decoded.TenantID)
}

// =============================================================================
// Delete
// =============================================================================

func TestDelete_Success(t *testing.T) {
	svc, mockCache := setupConfigCacheService(t)
	ctx := context.Background()

	mockCache.On("Delete", ctx, "config:t1:u1:a1").Return(true, nil)

	err := svc.Delete(ctx, "t1", "u1", "a1")
	require.NoError(t, err)
	mockCache.AssertCalled(t, "Delete", ctx, "config:t1:u1:a1")
}

func TestDelete_Error(t *testing.T) {
	svc, mockCache := setupConfigCacheService(t)
	ctx := context.Background()

	mockCache.On("Delete", ctx, mock.AnythingOfType("string")).Return(false, assert.AnError)

	err := svc.Delete(ctx, "t1", "u1", "a1")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to delete config from cache")
}

// =============================================================================
// DeleteByTenant
// =============================================================================

func TestDeleteByTenant_Success(t *testing.T) {
	svc, mockCache := setupConfigCacheService(t)
	ctx := context.Background()

	mockCache.On("DeletePattern", ctx, "config:tenant-123:*").Return(int64(5), nil)

	err := svc.DeleteByTenant(ctx, "tenant-123")
	require.NoError(t, err)
	mockCache.AssertCalled(t, "DeletePattern", ctx, "config:tenant-123:*")
}

func TestDeleteByTenant_Error(t *testing.T) {
	svc, mockCache := setupConfigCacheService(t)
	ctx := context.Background()

	mockCache.On("DeletePattern", ctx, mock.AnythingOfType("string")).Return(int64(0), assert.AnError)

	err := svc.DeleteByTenant(ctx, "tenant-123")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to delete configs by tenant")
}
