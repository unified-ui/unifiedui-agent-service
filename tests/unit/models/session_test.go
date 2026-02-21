// Package models_test provides unit tests for domain models.
package models_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/unifiedui/agent-service/internal/domain/models"
)

func TestNewSession(t *testing.T) {
	// Arrange
	tenantID := "tenant-123"
	userID := "user-456"
	ttl := 5 * time.Minute

	config := &models.SessionConfig{
		AgentID:   "agent-1",
		AgentType: "n8n",
		AgentName: "Test Agent",
		Endpoint:  "http://localhost:8080",
		Settings:  map[string]interface{}{"key": "value"},
		Features: &models.AgentFeatures{
			SupportsStreaming:     true,
			SupportsTracing:       true,
			SupportsHumanInLoop:   false,
			MaxTokens:             4096,
			MaxConversationLength: 100,
		},
	}

	creds := &models.EncryptedCreds{
		EncryptedData: "encrypted-secret-data",
		KeyVersion:    "v1",
	}

	// Act
	before := time.Now().UTC()
	session := models.NewSession(tenantID, userID, config, creds, ttl)
	after := time.Now().UTC()

	// Assert
	require.NotNil(t, session)
	assert.Equal(t, tenantID, session.TenantID)
	assert.Equal(t, userID, session.UserID)
	assert.Equal(t, config, session.Config)
	assert.Equal(t, creds, session.Credentials)

	// Check CreatedAt is within expected range
	assert.True(t, session.CreatedAt.After(before) || session.CreatedAt.Equal(before))
	assert.True(t, session.CreatedAt.Before(after) || session.CreatedAt.Equal(after))

	// Check ExpiresAt is CreatedAt + TTL
	expectedExpiry := session.CreatedAt.Add(ttl)
	assert.Equal(t, expectedExpiry, session.ExpiresAt)
}

func TestNewSession_NilConfigAndCreds(t *testing.T) {
	// Arrange
	tenantID := "tenant-123"
	userID := "user-456"
	ttl := 10 * time.Minute

	// Act
	session := models.NewSession(tenantID, userID, nil, nil, ttl)

	// Assert
	require.NotNil(t, session)
	assert.Equal(t, tenantID, session.TenantID)
	assert.Equal(t, userID, session.UserID)
	assert.Nil(t, session.Config)
	assert.Nil(t, session.Credentials)
}

func TestNewSession_ZeroTTL(t *testing.T) {
	// Arrange
	tenantID := "tenant-123"
	userID := "user-456"
	ttl := time.Duration(0)

	// Act
	session := models.NewSession(tenantID, userID, nil, nil, ttl)

	// Assert
	require.NotNil(t, session)
	// ExpiresAt should be same as CreatedAt when TTL is 0
	assert.Equal(t, session.CreatedAt, session.ExpiresAt)
}

func TestSession_IsExpired_NotExpired(t *testing.T) {
	// Arrange
	session := models.NewSession("tenant", "user", nil, nil, 5*time.Minute)

	// Act
	expired := session.IsExpired()

	// Assert
	assert.False(t, expired)
}

func TestSession_IsExpired_Expired(t *testing.T) {
	// Arrange
	session := &models.Session{
		TenantID:  "tenant",
		UserID:    "user",
		CreatedAt: time.Now().UTC().Add(-10 * time.Minute),
		ExpiresAt: time.Now().UTC().Add(-5 * time.Minute), // Expired 5 minutes ago
	}

	// Act
	expired := session.IsExpired()

	// Assert
	assert.True(t, expired)
}

func TestSession_IsExpired_JustExpired(t *testing.T) {
	// Arrange - session that expired 1 second ago
	session := &models.Session{
		TenantID:  "tenant",
		UserID:    "user",
		CreatedAt: time.Now().UTC().Add(-1 * time.Hour),
		ExpiresAt: time.Now().UTC().Add(-1 * time.Second),
	}

	// Act
	expired := session.IsExpired()

	// Assert
	assert.True(t, expired)
}

func TestSession_IsExpired_NearFuture(t *testing.T) {
	// Arrange - session that will expire in 1 second
	session := &models.Session{
		TenantID:  "tenant",
		UserID:    "user",
		CreatedAt: time.Now().UTC().Add(-1 * time.Hour),
		ExpiresAt: time.Now().UTC().Add(1 * time.Second),
	}

	// Act
	expired := session.IsExpired()

	// Assert
	assert.False(t, expired)
}

func TestSessionKey(t *testing.T) {
	testCases := []struct {
		name        string
		tenantID    string
		userID      string
		expectedKey string
	}{
		{
			name:        "standard IDs",
			tenantID:    "tenant-123",
			userID:      "user-456",
			expectedKey: "session:tenant-123:user-456",
		},
		{
			name:        "empty IDs",
			tenantID:    "",
			userID:      "",
			expectedKey: "session::",
		},
		{
			name:        "IDs with special characters",
			tenantID:    "tenant_with-dashes",
			userID:      "user.with.dots",
			expectedKey: "session:tenant_with-dashes:user.with.dots",
		},
		{
			name:        "UUID format",
			tenantID:    "550e8400-e29b-41d4-a716-446655440000",
			userID:      "660f9511-f30c-52e5-b827-557766551111",
			expectedKey: "session:550e8400-e29b-41d4-a716-446655440000:660f9511-f30c-52e5-b827-557766551111",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Act
			key := models.SessionKey(tc.tenantID, tc.userID)

			// Assert
			assert.Equal(t, tc.expectedKey, key)
		})
	}
}

func TestSessionConfig_Fields(t *testing.T) {
	// Arrange
	config := &models.SessionConfig{
		AgentID:   "agent-123",
		AgentType: "foundry",
		AgentName: "My Agent",
		Endpoint:  "https://api.example.com/chat",
		Settings: map[string]interface{}{
			"temperature": 0.7,
			"maxTokens":   2048,
		},
		Features: &models.AgentFeatures{
			SupportsStreaming:     true,
			SupportsTracing:       true,
			SupportsHumanInLoop:   false,
			MaxTokens:             4096,
			MaxConversationLength: 50,
		},
	}

	// Assert - verify all fields are accessible
	assert.Equal(t, "agent-123", config.AgentID)
	assert.Equal(t, "foundry", config.AgentType)
	assert.Equal(t, "My Agent", config.AgentName)
	assert.Equal(t, "https://api.example.com/chat", config.Endpoint)
	assert.Equal(t, 0.7, config.Settings["temperature"])
	assert.Equal(t, 2048, config.Settings["maxTokens"])
	assert.True(t, config.Features.SupportsStreaming)
	assert.True(t, config.Features.SupportsTracing)
	assert.False(t, config.Features.SupportsHumanInLoop)
	assert.Equal(t, 4096, config.Features.MaxTokens)
	assert.Equal(t, 50, config.Features.MaxConversationLength)
}

func TestEncryptedCreds_Fields(t *testing.T) {
	// Arrange
	creds := &models.EncryptedCreds{
		EncryptedData: "base64-encoded-encrypted-data",
		KeyVersion:    "2024-01",
	}

	// Assert
	assert.Equal(t, "base64-encoded-encrypted-data", creds.EncryptedData)
	assert.Equal(t, "2024-01", creds.KeyVersion)
}

func TestAgentFeatures_Fields(t *testing.T) {
	// Arrange
	features := &models.AgentFeatures{
		SupportsStreaming:     true,
		SupportsTracing:       false,
		SupportsHumanInLoop:   true,
		MaxTokens:             8192,
		MaxConversationLength: 200,
	}

	// Assert
	assert.True(t, features.SupportsStreaming)
	assert.False(t, features.SupportsTracing)
	assert.True(t, features.SupportsHumanInLoop)
	assert.Equal(t, 8192, features.MaxTokens)
	assert.Equal(t, 200, features.MaxConversationLength)
}

func TestAgentFeatures_ZeroValues(t *testing.T) {
	// Arrange
	features := &models.AgentFeatures{}

	// Assert - zero values
	assert.False(t, features.SupportsStreaming)
	assert.False(t, features.SupportsTracing)
	assert.False(t, features.SupportsHumanInLoop)
	assert.Equal(t, 0, features.MaxTokens)
	assert.Equal(t, 0, features.MaxConversationLength)
}
