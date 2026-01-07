// Package platform provides tests for the platform service client.
package platform_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/unifiedui/agent-service/internal/services/platform"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestGetApplicationConfig_Success tests successful application config retrieval.
func TestGetApplicationConfig_Success(t *testing.T) {
	// Create test response (now includes user info)
	expectedResponse := &platform.ApplicationConfigResponse{
		DocVersion:    "v1",
		Type:          platform.AgentTypeN8N,
		TenantID:      "tenant-123",
		ApplicationID: "app-456",
		Settings: platform.AgentSettings{
			APIVersion:            "v1",
			WorkflowType:          platform.N8NWorkflowTypeChatAgent,
			UseUnifiedChatHistory: true,
			ChatHistoryCount:      5,
			ChatURL:               "https://n8n.example.com/webhook/chat",
		},
		User: &platform.UserInfo{
			ID:            "user-789",
			DisplayName:   "Test User",
			PrincipalName: "test@example.com",
			Mail:          "test@example.com",
		},
	}

	// Create test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify request
		assert.Equal(t, http.MethodGet, r.Method)
		assert.Equal(t, "/api/v1/platform-service/tenants/tenant-123/applications/app-456/config", r.URL.Path)
		assert.Equal(t, "test-service-key", r.Header.Get("X-Service-Key"))
		assert.Equal(t, "Bearer test-auth-token", r.Header.Get("Authorization"))
		assert.Equal(t, "application/json", r.Header.Get("Accept"))

		// Send response
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(expectedResponse)
	}))
	defer server.Close()

	// Create client
	client := platform.NewClient(&platform.ClientConfig{
		BaseURL:    server.URL,
		ServiceKey: "test-service-key",
		Timeout:    5 * time.Second,
	})

	// Call GetApplicationConfig
	config, err := client.GetApplicationConfig(context.Background(), "tenant-123", "app-456", "test-auth-token")

	// Verify result
	require.NoError(t, err)
	require.NotNil(t, config)
	assert.Equal(t, expectedResponse.DocVersion, config.DocVersion)
	assert.Equal(t, expectedResponse.Type, config.Type)
	assert.Equal(t, expectedResponse.TenantID, config.TenantID)
	assert.Equal(t, expectedResponse.ApplicationID, config.ApplicationID)
	assert.Equal(t, expectedResponse.Settings.ChatURL, config.Settings.ChatURL)
	assert.NotNil(t, config.User)
	assert.Equal(t, "user-789", config.User.ID)
}

// TestGetApplicationConfig_MissingBaseURL tests error when base URL is not configured.
func TestGetApplicationConfig_MissingBaseURL(t *testing.T) {
	client := platform.NewClient(&platform.ClientConfig{
		ServiceKey: "test-key",
	})

	config, err := client.GetApplicationConfig(context.Background(), "tenant-123", "app-456", "test-token")

	require.Error(t, err)
	assert.Nil(t, config)
	assert.Contains(t, err.Error(), "platform service URL not configured")
}

// TestGetApplicationConfig_MissingServiceKey tests error when service key is not configured.
func TestGetApplicationConfig_MissingServiceKey(t *testing.T) {
	client := platform.NewClient(&platform.ClientConfig{
		BaseURL: "http://localhost:8081",
	})

	config, err := client.GetApplicationConfig(context.Background(), "tenant-123", "app-456", "test-token")

	require.Error(t, err)
	assert.Nil(t, config)
	assert.Contains(t, err.Error(), "service key not configured")
}

// TestGetApplicationConfig_MissingAuthToken tests error when auth token is not provided.
func TestGetApplicationConfig_MissingAuthToken(t *testing.T) {
	client := platform.NewClient(&platform.ClientConfig{
		BaseURL:    "http://localhost:8081",
		ServiceKey: "test-key",
	})

	config, err := client.GetApplicationConfig(context.Background(), "tenant-123", "app-456", "")

	require.Error(t, err)
	assert.Nil(t, config)
	assert.Contains(t, err.Error(), "auth token not provided")
}

// TestGetApplicationConfig_Unauthorized tests handling of 401 response.
func TestGetApplicationConfig_Unauthorized(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte(`{"error": "invalid service key or token"}`))
	}))
	defer server.Close()

	client := platform.NewClient(&platform.ClientConfig{
		BaseURL:    server.URL,
		ServiceKey: "invalid-key",
		Timeout:    5 * time.Second,
	})

	config, err := client.GetApplicationConfig(context.Background(), "tenant-123", "app-456", "invalid-token")

	require.Error(t, err)
	assert.Nil(t, config)
	assert.Contains(t, err.Error(), "status 401")
}

// TestGetApplicationConfig_Forbidden tests handling of 403 response (invalid service key).
func TestGetApplicationConfig_Forbidden(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
		w.Write([]byte(`{"error": "invalid service key"}`))
	}))
	defer server.Close()

	client := platform.NewClient(&platform.ClientConfig{
		BaseURL:    server.URL,
		ServiceKey: "wrong-key",
		Timeout:    5 * time.Second,
	})

	config, err := client.GetApplicationConfig(context.Background(), "tenant-123", "app-456", "test-token")

	require.Error(t, err)
	assert.Nil(t, config)
	assert.Contains(t, err.Error(), "status 403")
}

// TestGetApplicationConfig_NotFound tests handling of 404 response.
func TestGetApplicationConfig_NotFound(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte(`{"error": "application not found"}`))
	}))
	defer server.Close()

	client := platform.NewClient(&platform.ClientConfig{
		BaseURL:    server.URL,
		ServiceKey: "test-key",
		Timeout:    5 * time.Second,
	})

	config, err := client.GetApplicationConfig(context.Background(), "tenant-123", "nonexistent-app", "test-token")

	require.Error(t, err)
	assert.Nil(t, config)
	assert.Contains(t, err.Error(), "status 404")
}

// TestGetAgentConfig_Success tests successful agent config retrieval.
func TestGetAgentConfig_Success(t *testing.T) {
	// Create test response (includes user info from platform)
	appResponse := &platform.ApplicationConfigResponse{
		DocVersion:    "v1",
		Type:          platform.AgentTypeN8N,
		TenantID:      "tenant-123",
		ApplicationID: "app-456",
		Settings: platform.AgentSettings{
			APIVersion:            "v1",
			WorkflowType:          platform.N8NWorkflowTypeChatAgent,
			UseUnifiedChatHistory: true,
			ChatHistoryCount:      5,
			ChatURL:               "https://n8n.example.com/webhook/chat",
		},
		User: &platform.UserInfo{
			ID:            "user-789",
			DisplayName:   "Test User",
			PrincipalName: "test@example.com",
			Mail:          "test@example.com",
		},
	}

	// Create test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(appResponse)
	}))
	defer server.Close()

	// Create client
	client := platform.NewClient(&platform.ClientConfig{
		BaseURL:    server.URL,
		ServiceKey: "test-service-key",
		Timeout:    5 * time.Second,
	})

	// Call GetAgentConfig (now takes authToken instead of user)
	config, err := client.GetAgentConfig(context.Background(), "tenant-123", "app-456", "conv-abc", "test-auth-token")

	// Verify result
	require.NoError(t, err)
	require.NotNil(t, config)
	assert.Equal(t, "v1", config.DocVersion)
	assert.Equal(t, platform.AgentTypeN8N, config.Type)
	assert.Equal(t, "tenant-123", config.TenantID)
	assert.Equal(t, "app-456", config.ApplicationID)
	assert.Equal(t, "conv-abc", config.ConversationID)
	// User info comes from platform response now
	assert.NotNil(t, config.User)
	assert.Equal(t, "user-789", config.User.ID)
	assert.Equal(t, "Test User", config.User.DisplayName)
}

// TestGetAgentConfig_PlatformError tests error handling when platform call fails.
func TestGetAgentConfig_PlatformError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(`{"error": "internal error"}`))
	}))
	defer server.Close()

	client := platform.NewClient(&platform.ClientConfig{
		BaseURL:    server.URL,
		ServiceKey: "test-service-key",
		Timeout:    5 * time.Second,
	})

	config, err := client.GetAgentConfig(context.Background(), "tenant-123", "app-456", "conv-abc", "test-token")

	require.Error(t, err)
	assert.Nil(t, config)
	assert.Contains(t, err.Error(), "failed to get application config")
}

// TestGetAgentConfigFromFile_Success tests successful config retrieval from file.
func TestGetAgentConfigFromFile_Success(t *testing.T) {
	// Create temp config file
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "config.json")

	configData := &platform.AgentConfig{
		DocVersion:     "v1",
		Type:           platform.AgentTypeN8N,
		TenantID:       "tenant-123",
		ApplicationID:  "app-456",
		ConversationID: "conv-789",
		Settings: platform.AgentSettings{
			APIVersion:   "v1",
			WorkflowType: platform.N8NWorkflowTypeChatAgent,
			ChatURL:      "https://n8n.example.com/webhook/chat",
		},
	}

	data, err := json.Marshal(configData)
	require.NoError(t, err)
	err = os.WriteFile(configPath, data, 0644)
	require.NoError(t, err)

	client := platform.NewClient(&platform.ClientConfig{
		ConfigPath: configPath,
	})

	config, err := client.GetAgentConfigFromFile(context.Background(), "tenant-123", "app-456")

	require.NoError(t, err)
	require.NotNil(t, config)
	assert.Equal(t, "v1", config.DocVersion)
	assert.Equal(t, platform.AgentTypeN8N, config.Type)
	assert.Equal(t, "tenant-123", config.TenantID)
}

// TestGetAgentConfigFromFile_MissingPath tests error when config path is not set.
func TestGetAgentConfigFromFile_MissingPath(t *testing.T) {
	client := platform.NewClient(&platform.ClientConfig{})

	config, err := client.GetAgentConfigFromFile(context.Background(), "tenant-123", "app-456")

	require.Error(t, err)
	assert.Nil(t, config)
	assert.Contains(t, err.Error(), "config path not configured")
}

// TestGetAgentConfigFromFile_FileNotFound tests error when config file doesn't exist.
func TestGetAgentConfigFromFile_FileNotFound(t *testing.T) {
	client := platform.NewClient(&platform.ClientConfig{
		ConfigPath: "/nonexistent/path/config.json",
	})

	config, err := client.GetAgentConfigFromFile(context.Background(), "tenant-123", "app-456")

	require.Error(t, err)
	assert.Nil(t, config)
	assert.Contains(t, err.Error(), "failed to read config file")
}

// TestGetAgentConfigFromFile_InvalidJSON tests error when config file has invalid JSON.
func TestGetAgentConfigFromFile_InvalidJSON(t *testing.T) {
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "config.json")

	// Write invalid JSON
	err := os.WriteFile(configPath, []byte("invalid json {"), 0644)
	require.NoError(t, err)

	client := platform.NewClient(&platform.ClientConfig{
		ConfigPath: configPath,
	})

	config, err := client.GetAgentConfigFromFile(context.Background(), "tenant-123", "app-456")

	require.Error(t, err)
	assert.Nil(t, config)
	assert.Contains(t, err.Error(), "failed to parse config file")
}
