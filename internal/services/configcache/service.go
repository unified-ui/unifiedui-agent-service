// Package configcache provides agent config caching to reduce platform service calls.
package configcache

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/unifiedui/agent-service/internal/core/cache"
	"github.com/unifiedui/agent-service/internal/pkg/encryption"
	"github.com/unifiedui/agent-service/internal/services/platform"
)

const (
	// DefaultConfigCacheTTL is the default TTL for config cache (5 minutes).
	DefaultConfigCacheTTL = 5 * time.Minute
)

// Service provides agent config caching.
type Service interface {
	// Get retrieves an agent config from cache, or returns nil if not found.
	Get(ctx context.Context, tenantID, userID, chatAgentID string) (*platform.AgentConfig, error)

	// Set stores an agent config in cache with the configured TTL.
	Set(ctx context.Context, tenantID, userID, chatAgentID string, config *platform.AgentConfig) error

	// Delete removes an agent config from cache.
	Delete(ctx context.Context, tenantID, userID, chatAgentID string) error

	// DeleteByTenant removes all agent configs for a tenant from cache.
	DeleteByTenant(ctx context.Context, tenantID string) error

	// BuildCacheKey generates the cache key for an agent config.
	BuildCacheKey(tenantID, userID, chatAgentID string) string
}

type service struct {
	cacheClient cache.Client
	encryptor   encryption.Encryptor
	ttl         time.Duration
}

// Config holds the configuration for the config cache service.
type Config struct {
	CacheClient cache.Client
	Encryptor   encryption.Encryptor
	TTL         time.Duration
}

// NewService creates a new config cache service.
func NewService(cfg *Config) (Service, error) {
	if cfg == nil {
		return nil, fmt.Errorf("config is required")
	}
	if cfg.CacheClient == nil {
		return nil, fmt.Errorf("cache client is required")
	}
	if cfg.Encryptor == nil {
		return nil, fmt.Errorf("encryptor is required")
	}

	ttl := cfg.TTL
	if ttl == 0 {
		ttl = DefaultConfigCacheTTL
	}

	return &service{
		cacheClient: cfg.CacheClient,
		encryptor:   cfg.Encryptor,
		ttl:         ttl,
	}, nil
}

// Get retrieves an agent config from cache.
func (s *service) Get(ctx context.Context, tenantID, userID, chatAgentID string) (*platform.AgentConfig, error) {
	key := s.BuildCacheKey(tenantID, userID, chatAgentID)

	encrypted, err := s.cacheClient.Get(ctx, key)
	if err != nil {
		return nil, fmt.Errorf("failed to get config from cache: %w", err)
	}

	if encrypted == nil {
		return nil, nil
	}

	decrypted, err := s.encryptor.Decrypt(string(encrypted))
	if err != nil {
		_, _ = s.cacheClient.Delete(ctx, key)
		return nil, nil //nolint:nilerr // graceful fallback: decryption failure means stale cache
	}

	var config platform.AgentConfig
	if err := json.Unmarshal(decrypted, &config); err != nil {
		_, _ = s.cacheClient.Delete(ctx, key)
		return nil, nil //nolint:nilerr // graceful fallback: corrupted cache data
	}

	return &config, nil
}

// Set stores an agent config in cache.
func (s *service) Set(ctx context.Context, tenantID, userID, chatAgentID string, config *platform.AgentConfig) error {
	if config == nil {
		return fmt.Errorf("config is required")
	}

	data, err := json.Marshal(config)
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	encrypted, err := s.encryptor.Encrypt(data)
	if err != nil {
		return fmt.Errorf("failed to encrypt config: %w", err)
	}

	key := s.BuildCacheKey(tenantID, userID, chatAgentID)
	if err := s.cacheClient.Set(ctx, key, []byte(encrypted), s.ttl); err != nil {
		return fmt.Errorf("failed to store config in cache: %w", err)
	}

	return nil
}

// Delete removes an agent config from cache.
func (s *service) Delete(ctx context.Context, tenantID, userID, chatAgentID string) error {
	key := s.BuildCacheKey(tenantID, userID, chatAgentID)
	_, err := s.cacheClient.Delete(ctx, key)
	if err != nil {
		return fmt.Errorf("failed to delete config from cache: %w", err)
	}
	return nil
}

// DeleteByTenant removes all agent configs for a tenant from cache.
func (s *service) DeleteByTenant(ctx context.Context, tenantID string) error {
	pattern := fmt.Sprintf("config:%s:*", tenantID)
	_, err := s.cacheClient.DeletePattern(ctx, pattern)
	if err != nil {
		return fmt.Errorf("failed to delete configs by tenant: %w", err)
	}
	return nil
}

// BuildCacheKey generates the cache key for an agent config.
func (s *service) BuildCacheKey(tenantID, userID, chatAgentID string) string {
	return fmt.Sprintf("config:%s:%s:%s", tenantID, userID, chatAgentID)
}
