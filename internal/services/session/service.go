// Package session provides session management with caching for agent configurations.
package session

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/unifiedui/agent-service/internal/core/cache"
	"github.com/unifiedui/agent-service/internal/domain/models"
	"github.com/unifiedui/agent-service/internal/pkg/encryption"
	"github.com/unifiedui/agent-service/internal/services/platform"
)

const (
	// DefaultSessionTTL is the default TTL for session cache (3 minutes).
	DefaultSessionTTL = 3 * time.Minute

	// DefaultChatHistoryCount is the default number of chat history messages.
	DefaultChatHistoryCount = 30
)

// SessionData represents cached session data.
type SessionData struct {
	Config         *platform.AgentConfig     `json:"config"`
	ChatHistory    []models.ChatHistoryEntry `json:"chatHistory"`
	TenantID       string                    `json:"tenantId"`
	UserID         string                    `json:"userId"`
	ConversationID string                    `json:"conversationId"`
	CreatedAt      time.Time                 `json:"createdAt"`
	UpdatedAt      time.Time                 `json:"updatedAt"`
}

// Service provides session management with caching.
type Service interface {
	// GetSession retrieves a session from cache, or returns nil if not found.
	GetSession(ctx context.Context, tenantID, userID, conversationID string) (*SessionData, error)

	// SetSession stores a session in cache with the configured TTL.
	SetSession(ctx context.Context, session *SessionData) error

	// UpdateChatHistory updates the chat history in an existing session.
	// Removes oldest messages if count exceeds limit, then adds new messages.
	UpdateChatHistory(ctx context.Context, tenantID, userID, conversationID string, newEntries []models.ChatHistoryEntry) error

	// DeleteSession removes a session from cache.
	DeleteSession(ctx context.Context, tenantID, userID, conversationID string) error

	// BuildCacheKey generates the cache key for a session.
	BuildCacheKey(tenantID, userID, conversationID string) string
}

// service implements the Service interface.
type service struct {
	cacheClient cache.Client
	encryptor   encryption.Encryptor
	ttl         time.Duration
}

// Config holds the configuration for the session service.
type Config struct {
	CacheClient cache.Client
	Encryptor   encryption.Encryptor
	TTL         time.Duration
}

// NewService creates a new session service.
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
		ttl = DefaultSessionTTL
	}

	return &service{
		cacheClient: cfg.CacheClient,
		encryptor:   cfg.Encryptor,
		ttl:         ttl,
	}, nil
}

// GetSession retrieves a session from cache.
// Returns nil (not an error) if decryption fails (e.g., key changed) - cache will be skipped.
func (s *service) GetSession(ctx context.Context, tenantID, userID, conversationID string) (*SessionData, error) {
	key := s.BuildCacheKey(tenantID, userID, conversationID)

	encrypted, err := s.cacheClient.Get(ctx, key)
	if err != nil {
		return nil, fmt.Errorf("failed to get session from cache: %w", err)
	}

	if encrypted == nil {
		return nil, nil // Not found
	}

	// Decrypt - if decryption fails (e.g., key changed), skip cache gracefully
	decrypted, err := s.encryptor.Decrypt(string(encrypted))
	if err != nil {
		// Key might have changed - delete stale cache entry and return nil to fetch fresh data
		_, _ = s.cacheClient.Delete(ctx, key)
		return nil, nil
	}

	// Unmarshal - if unmarshal fails, data is corrupted, skip cache
	var session SessionData
	if err := json.Unmarshal(decrypted, &session); err != nil {
		_, _ = s.cacheClient.Delete(ctx, key)
		return nil, nil
	}

	return &session, nil
}

// SetSession stores a session in cache.
func (s *service) SetSession(ctx context.Context, session *SessionData) error {
	if session == nil {
		return fmt.Errorf("session is required")
	}

	session.UpdatedAt = time.Now().UTC()
	if session.CreatedAt.IsZero() {
		session.CreatedAt = session.UpdatedAt
	}

	// Marshal to JSON
	data, err := json.Marshal(session)
	if err != nil {
		return fmt.Errorf("failed to marshal session: %w", err)
	}

	// Encrypt
	encrypted, err := s.encryptor.Encrypt(data)
	if err != nil {
		return fmt.Errorf("failed to encrypt session: %w", err)
	}

	// Store in cache
	key := s.BuildCacheKey(session.TenantID, session.UserID, session.ConversationID)
	if err := s.cacheClient.Set(ctx, key, []byte(encrypted), s.ttl); err != nil {
		return fmt.Errorf("failed to store session in cache: %w", err)
	}

	return nil
}

// UpdateChatHistory updates the chat history in an existing session.
func (s *service) UpdateChatHistory(ctx context.Context, tenantID, userID, conversationID string, newEntries []models.ChatHistoryEntry) error {
	session, err := s.GetSession(ctx, tenantID, userID, conversationID)
	if err != nil {
		return fmt.Errorf("failed to get session for update: %w", err)
	}

	if session == nil {
		return fmt.Errorf("session not found")
	}

	// Get chat history count from config
	maxCount := DefaultChatHistoryCount
	if session.Config != nil && session.Config.Settings.ChatHistoryCount > 0 {
		maxCount = session.Config.Settings.ChatHistoryCount
	}

	// Add new entries
	session.ChatHistory = append(session.ChatHistory, newEntries...)

	// Trim to max count (remove oldest entries)
	if len(session.ChatHistory) > maxCount {
		excess := len(session.ChatHistory) - maxCount
		session.ChatHistory = session.ChatHistory[excess:]
	}

	// Save updated session
	return s.SetSession(ctx, session)
}

// DeleteSession removes a session from cache.
func (s *service) DeleteSession(ctx context.Context, tenantID, userID, conversationID string) error {
	key := s.BuildCacheKey(tenantID, userID, conversationID)
	_, err := s.cacheClient.Delete(ctx, key)
	if err != nil {
		return fmt.Errorf("failed to delete session: %w", err)
	}
	return nil
}

// BuildCacheKey generates the cache key for a session.
func (s *service) BuildCacheKey(tenantID, userID, conversationID string) string {
	return fmt.Sprintf("session:%s:%s:%s", tenantID, userID, conversationID)
}

// NewSessionData creates a new SessionData with the given parameters.
func NewSessionData(
	config *platform.AgentConfig,
	chatHistory []models.ChatHistoryEntry,
	tenantID, userID, conversationID string,
) *SessionData {
	now := time.Now().UTC()
	return &SessionData{
		Config:         config,
		ChatHistory:    chatHistory,
		TenantID:       tenantID,
		UserID:         userID,
		ConversationID: conversationID,
		CreatedAt:      now,
		UpdatedAt:      now,
	}
}
