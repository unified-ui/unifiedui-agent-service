// Package models contains domain models for the UnifiedUI Chat Service.
package models

import "time"

// Session represents a user session with cached configuration.
type Session struct {
	TenantID    string          `json:"tenantId"`
	UserID      string          `json:"userId"`
	Config      *SessionConfig  `json:"config"`
	Credentials *EncryptedCreds `json:"credentials"`
	CreatedAt   time.Time       `json:"createdAt"`
	ExpiresAt   time.Time       `json:"expiresAt"`
}

// SessionConfig holds the agent configuration from Platform Service.
type SessionConfig struct {
	AgentID   string                 `json:"agentId"`
	AgentType string                 `json:"agentType"`
	AgentName string                 `json:"agentName"`
	Endpoint  string                 `json:"endpoint"`
	Settings  map[string]interface{} `json:"settings,omitempty"`
	Features  *AgentFeatures         `json:"features,omitempty"`
}

// AgentFeatures describes the capabilities of an agent.
type AgentFeatures struct {
	SupportsStreaming     bool `json:"supportsStreaming"`
	SupportsTracing       bool `json:"supportsTracing"`
	SupportsHumanInLoop   bool `json:"supportsHumanInLoop"`
	MaxTokens             int  `json:"maxTokens,omitempty"`
	MaxConversationLength int  `json:"maxConversationLength,omitempty"`
}

// EncryptedCreds holds encrypted credentials for the agent backend.
type EncryptedCreds struct {
	// EncryptedData is the Fernet-encrypted credential blob.
	EncryptedData string `json:"encryptedData"`
	// KeyVersion is the version of the encryption key used.
	KeyVersion string `json:"keyVersion"`
}

// NewSession creates a new session with the given parameters and TTL.
func NewSession(tenantID, userID string, config *SessionConfig, creds *EncryptedCreds, ttl time.Duration) *Session {
	now := time.Now().UTC()
	return &Session{
		TenantID:    tenantID,
		UserID:      userID,
		Config:      config,
		Credentials: creds,
		CreatedAt:   now,
		ExpiresAt:   now.Add(ttl),
	}
}

// IsExpired checks if the session has expired.
func (s *Session) IsExpired() bool {
	return time.Now().UTC().After(s.ExpiresAt)
}

// SessionKey generates a cache key for the session.
func SessionKey(tenantID, userID string) string {
	return "session:" + tenantID + ":" + userID
}
