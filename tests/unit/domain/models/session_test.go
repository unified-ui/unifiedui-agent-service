package models

import (
	"testing"
	"time"

	"github.com/unifiedui/agent-service/internal/domain/models"

	"github.com/stretchr/testify/require"
)

func TestNewSession(t *testing.T) {
	cfg := &models.SessionConfig{
		AgentID:   "agent-1",
		AgentType: "n8n",
		AgentName: "Test Agent",
	}
	creds := &models.EncryptedCreds{EncryptedData: "enc", KeyVersion: "v1"}
	ttl := 5 * time.Minute

	session := models.NewSession("tenant-1", "user-1", cfg, creds, ttl)

	require.Equal(t, "tenant-1", session.TenantID)
	require.Equal(t, "user-1", session.UserID)
	require.Equal(t, cfg, session.Config)
	require.Equal(t, creds, session.Credentials)
	require.False(t, session.CreatedAt.IsZero())
	require.True(t, session.ExpiresAt.After(session.CreatedAt))
}

func TestSession_IsExpired(t *testing.T) {
	past := &models.Session{ExpiresAt: time.Now().UTC().Add(-time.Hour)}
	require.True(t, past.IsExpired())

	future := &models.Session{ExpiresAt: time.Now().UTC().Add(time.Hour)}
	require.False(t, future.IsExpired())
}

func TestSessionKey(t *testing.T) {
	key := models.SessionKey("tenant-1", "user-1")
	require.Equal(t, "session:tenant-1:user-1", key)
}
