package platform_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/unifiedui/agent-service/internal/services/platform"
)

func newTestClient(ts *httptest.Server) platform.Client {
	return platform.NewClient(&platform.ClientConfig{
		BaseURL:    ts.URL,
		ServiceKey: "test-service-key",
		Timeout:    0,
	})
}

func jsonHandler(data interface{}) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(data)
	}
}

func statusHandler(code int, body string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(code)
		w.Write([]byte(body))
	}
}

// --- NewClient ---

func TestNewClient_DefaultTimeout(t *testing.T) {
	client := platform.NewClient(&platform.ClientConfig{
		BaseURL:    "http://localhost:8080",
		ServiceKey: "key",
	})
	assert.NotNil(t, client)
}

func TestNewClient_CustomTimeout(t *testing.T) {
	client := platform.NewClient(&platform.ClientConfig{
		BaseURL:    "http://localhost:8080",
		ServiceKey: "key",
		Timeout:    5000000000,
	})
	assert.NotNil(t, client)
}

// --- GetChatAgentConfig ---

func TestGetChatAgentConfig_Success(t *testing.T) {
	resp := platform.ChatAgentConfigResponse{
		DocVersion:  "1",
		Type:        platform.AgentTypeN8N,
		TenantID:    "tenant1",
		ChatAgentID: "agent1",
	}
	ts := httptest.NewServer(jsonHandler(resp))
	defer ts.Close()

	client := newTestClient(ts)
	result, err := client.GetChatAgentConfig(context.Background(), "tenant1", "agent1", "auth-token")
	require.NoError(t, err)
	assert.Equal(t, "tenant1", result.TenantID)
	assert.Equal(t, "agent1", result.ChatAgentID)
}

func TestGetChatAgentConfig_MissingBaseURL(t *testing.T) {
	client := platform.NewClient(&platform.ClientConfig{ServiceKey: "key"})
	_, err := client.GetChatAgentConfig(context.Background(), "t", "a", "token")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not configured")
}

func TestGetChatAgentConfig_MissingServiceKey(t *testing.T) {
	client := platform.NewClient(&platform.ClientConfig{BaseURL: "http://localhost"})
	_, err := client.GetChatAgentConfig(context.Background(), "t", "a", "token")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "service key")
}

func TestGetChatAgentConfig_MissingAuthToken(t *testing.T) {
	client := platform.NewClient(&platform.ClientConfig{BaseURL: "http://localhost", ServiceKey: "key"})
	_, err := client.GetChatAgentConfig(context.Background(), "t", "a", "")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "auth token")
}

func TestGetChatAgentConfig_NotFound(t *testing.T) {
	ts := httptest.NewServer(statusHandler(404, "not found"))
	defer ts.Close()
	client := newTestClient(ts)
	_, err := client.GetChatAgentConfig(context.Background(), "t", "a", "token")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not_found")
}

func TestGetChatAgentConfig_Unauthorized(t *testing.T) {
	ts := httptest.NewServer(statusHandler(401, "unauthorized"))
	defer ts.Close()
	client := newTestClient(ts)
	_, err := client.GetChatAgentConfig(context.Background(), "t", "a", "token")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unauthorized")
}

func TestGetChatAgentConfig_Forbidden(t *testing.T) {
	ts := httptest.NewServer(statusHandler(403, "forbidden"))
	defer ts.Close()
	client := newTestClient(ts)
	_, err := client.GetChatAgentConfig(context.Background(), "t", "a", "token")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "forbidden")
}

func TestGetChatAgentConfig_ServerError(t *testing.T) {
	ts := httptest.NewServer(statusHandler(500, "server error"))
	defer ts.Close()
	client := newTestClient(ts)
	_, err := client.GetChatAgentConfig(context.Background(), "t", "a", "token")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "500")
}

// --- GetAgentConfig ---

func TestGetAgentConfig_Success(t *testing.T) {
	resp := platform.ChatAgentConfigResponse{
		DocVersion:  "1",
		Type:        platform.AgentTypeN8N,
		TenantID:    "tenant1",
		ChatAgentID: "agent1",
		Settings: platform.AgentSettings{
			ChatURL: "http://n8n.local/webhook/chat",
		},
	}
	ts := httptest.NewServer(jsonHandler(resp))
	defer ts.Close()

	client := newTestClient(ts)
	result, err := client.GetAgentConfig(context.Background(), "tenant1", "agent1", "conv1", "auth-token")
	require.NoError(t, err)
	assert.Equal(t, "conv1", result.ConversationID)
	assert.Equal(t, "agent1", result.ChatAgentID)
}

// --- GetAgentConfigFromFile ---

func TestGetAgentConfigFromFile_Success(t *testing.T) {
	tmpDir := t.TempDir()
	configFile := filepath.Join(tmpDir, "config.json")
	config := platform.AgentConfig{
		Type:     platform.AgentTypeN8N,
		TenantID: "tenant1",
	}
	data, _ := json.Marshal(config)
	os.WriteFile(configFile, data, 0644)

	client := platform.NewClient(&platform.ClientConfig{ConfigPath: configFile})
	result, err := client.GetAgentConfigFromFile(context.Background(), "tenant1", "agent1")
	require.NoError(t, err)
	assert.Equal(t, "tenant1", result.TenantID)
}

func TestGetAgentConfigFromFile_MissingConfigPath(t *testing.T) {
	client := platform.NewClient(&platform.ClientConfig{})
	_, err := client.GetAgentConfigFromFile(context.Background(), "t", "a")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "config path")
}

func TestGetAgentConfigFromFile_FileNotFound(t *testing.T) {
	client := platform.NewClient(&platform.ClientConfig{ConfigPath: "/nonexistent/config.json"})
	_, err := client.GetAgentConfigFromFile(context.Background(), "t", "a")
	assert.Error(t, err)
}

func TestGetAgentConfigFromFile_InvalidJSON(t *testing.T) {
	tmpDir := t.TempDir()
	configFile := filepath.Join(tmpDir, "config.json")
	os.WriteFile(configFile, []byte("not json"), 0644)

	client := platform.NewClient(&platform.ClientConfig{ConfigPath: configFile})
	_, err := client.GetAgentConfigFromFile(context.Background(), "t", "a")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "parse")
}

// --- GetMe ---

func TestGetMe_Success(t *testing.T) {
	resp := platform.UserInfo{ID: "user1", DisplayName: "Test User"}
	ts := httptest.NewServer(jsonHandler(resp))
	defer ts.Close()

	client := newTestClient(ts)
	result, err := client.GetMe(context.Background(), "auth-token")
	require.NoError(t, err)
	assert.Equal(t, "user1", result.ID)
}

func TestGetMe_MissingBaseURL(t *testing.T) {
	client := platform.NewClient(&platform.ClientConfig{})
	_, err := client.GetMe(context.Background(), "token")
	assert.Error(t, err)
}

func TestGetMe_MissingAuthToken(t *testing.T) {
	client := platform.NewClient(&platform.ClientConfig{BaseURL: "http://localhost"})
	_, err := client.GetMe(context.Background(), "")
	assert.Error(t, err)
}

// --- GetConversation ---

func TestGetConversation_Success(t *testing.T) {
	resp := platform.ConversationResponse{ID: "conv1", Name: "Test Conv"}
	ts := httptest.NewServer(jsonHandler(resp))
	defer ts.Close()

	client := newTestClient(ts)
	result, err := client.GetConversation(context.Background(), "tenant1", "conv1", "token")
	require.NoError(t, err)
	assert.Equal(t, "conv1", result.ID)
}

func TestGetConversation_MissingBaseURL(t *testing.T) {
	client := platform.NewClient(&platform.ClientConfig{})
	_, err := client.GetConversation(context.Background(), "t", "c", "token")
	assert.Error(t, err)
}

// --- ValidateConversation ---

func TestValidateConversation_Success(t *testing.T) {
	ts := httptest.NewServer(statusHandler(200, "{}"))
	defer ts.Close()

	client := newTestClient(ts)
	err := client.ValidateConversation(context.Background(), "tenant1", "conv1", "token")
	assert.NoError(t, err)
}

func TestValidateConversation_EmptyBaseURL(t *testing.T) {
	client := platform.NewClient(&platform.ClientConfig{})
	err := client.ValidateConversation(context.Background(), "t", "c", "token")
	assert.NoError(t, err)
}

func TestValidateConversation_MissingToken(t *testing.T) {
	client := platform.NewClient(&platform.ClientConfig{BaseURL: "http://localhost"})
	err := client.ValidateConversation(context.Background(), "t", "c", "")
	assert.Error(t, err)
}

func TestValidateConversation_NotFound(t *testing.T) {
	ts := httptest.NewServer(statusHandler(404, "not found"))
	defer ts.Close()
	client := newTestClient(ts)
	err := client.ValidateConversation(context.Background(), "t", "c", "token")
	assert.Error(t, err)
}

// --- ValidateAutonomousAgent ---

func TestValidateAutonomousAgent_Success(t *testing.T) {
	ts := httptest.NewServer(statusHandler(200, "{}"))
	defer ts.Close()
	client := newTestClient(ts)
	err := client.ValidateAutonomousAgent(context.Background(), "tenant1", "aa1", "token")
	assert.NoError(t, err)
}

func TestValidateAutonomousAgent_EmptyBaseURL(t *testing.T) {
	client := platform.NewClient(&platform.ClientConfig{})
	err := client.ValidateAutonomousAgent(context.Background(), "t", "a", "token")
	assert.NoError(t, err)
}

func TestValidateAutonomousAgent_MissingToken(t *testing.T) {
	client := platform.NewClient(&platform.ClientConfig{BaseURL: "http://localhost"})
	err := client.ValidateAutonomousAgent(context.Background(), "t", "a", "")
	assert.Error(t, err)
}

// --- GetAutonomousAgentConfig ---

func TestGetAutonomousAgentConfig_Success(t *testing.T) {
	resp := platform.AutonomousAgentConfigResponse{
		TenantID:         "tenant1",
		AutonomousAgentID: "aa1",
	}
	ts := httptest.NewServer(jsonHandler(resp))
	defer ts.Close()

	client := newTestClient(ts)
	result, err := client.GetAutonomousAgentConfig(context.Background(), "tenant1", "aa1", "api-key")
	require.NoError(t, err)
	assert.Equal(t, "aa1", result.AutonomousAgentID)
}

func TestGetAutonomousAgentConfig_MissingBaseURL(t *testing.T) {
	client := platform.NewClient(&platform.ClientConfig{})
	_, err := client.GetAutonomousAgentConfig(context.Background(), "t", "a", "key")
	assert.Error(t, err)
}

func TestGetAutonomousAgentConfig_MissingAPIKey(t *testing.T) {
	client := platform.NewClient(&platform.ClientConfig{BaseURL: "http://localhost"})
	_, err := client.GetAutonomousAgentConfig(context.Background(), "t", "a", "")
	assert.Error(t, err)
}

// --- GetAutonomousAgentConfigWithBearer ---

func TestGetAutonomousAgentConfigWithBearer_Success(t *testing.T) {
	resp := platform.AutonomousAgentConfigResponse{
		TenantID:         "tenant1",
		AutonomousAgentID: "aa1",
	}
	ts := httptest.NewServer(jsonHandler(resp))
	defer ts.Close()

	client := newTestClient(ts)
	result, err := client.GetAutonomousAgentConfigWithBearer(context.Background(), "tenant1", "aa1", "bearer-token")
	require.NoError(t, err)
	assert.Equal(t, "aa1", result.AutonomousAgentID)
}

func TestGetAutonomousAgentConfigWithBearer_MissingAuth(t *testing.T) {
	client := platform.NewClient(&platform.ClientConfig{BaseURL: "http://localhost"})
	_, err := client.GetAutonomousAgentConfigWithBearer(context.Background(), "t", "a", "")
	assert.Error(t, err)
}

// --- ValidateAutonomousAgentAPIKey ---

func TestValidateAutonomousAgentAPIKey_Success(t *testing.T) {
	ts := httptest.NewServer(statusHandler(200, "{}"))
	defer ts.Close()
	client := newTestClient(ts)
	err := client.ValidateAutonomousAgentAPIKey(context.Background(), "tenant1", "aa1", "api-key")
	assert.NoError(t, err)
}

func TestValidateAutonomousAgentAPIKey_EmptyBaseURL(t *testing.T) {
	client := platform.NewClient(&platform.ClientConfig{})
	err := client.ValidateAutonomousAgentAPIKey(context.Background(), "t", "a", "key")
	assert.NoError(t, err)
}

func TestValidateAutonomousAgentAPIKey_MissingKey(t *testing.T) {
	client := platform.NewClient(&platform.ClientConfig{BaseURL: "http://localhost"})
	err := client.ValidateAutonomousAgentAPIKey(context.Background(), "t", "a", "")
	assert.Error(t, err)
}

// --- GetAIModelsByPurpose ---

func TestGetAIModelsByPurpose_Success(t *testing.T) {
	resp := []platform.AIModelWithSecretResponse{
		{ID: "m1", Provider: "OPENAI", Config: map[string]interface{}{"model_name": "gpt-4"}},
	}
	ts := httptest.NewServer(jsonHandler(resp))
	defer ts.Close()

	client := newTestClient(ts)
	result, err := client.GetAIModelsByPurpose(context.Background(), "tenant1", "TITLE_GEN", "LLM_MODEL")
	require.NoError(t, err)
	assert.Len(t, result, 1)
	assert.Equal(t, "m1", result[0].ID)
}

func TestGetAIModelsByPurpose_WithModelType(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Contains(t, r.URL.RawQuery, "model_type=LLM_MODEL")
		json.NewEncoder(w).Encode([]platform.AIModelWithSecretResponse{})
	}))
	defer ts.Close()

	client := newTestClient(ts)
	_, err := client.GetAIModelsByPurpose(context.Background(), "tenant1", "TITLE_GEN", "LLM_MODEL")
	assert.NoError(t, err)
}

func TestGetAIModelsByPurpose_MissingBaseURL(t *testing.T) {
	client := platform.NewClient(&platform.ClientConfig{})
	_, err := client.GetAIModelsByPurpose(context.Background(), "t", "p", "m")
	assert.Error(t, err)
}

func TestGetAIModelsByPurpose_MissingServiceKey(t *testing.T) {
	client := platform.NewClient(&platform.ClientConfig{BaseURL: "http://localhost"})
	_, err := client.GetAIModelsByPurpose(context.Background(), "t", "p", "m")
	assert.Error(t, err)
}

// --- GetCredentialSecret ---

func TestGetCredentialSecret_Success(t *testing.T) {
	resp := platform.CredentialSecretResponse{CredentialID: "cred1", SecretValue: "secret-value"}
	ts := httptest.NewServer(jsonHandler(resp))
	defer ts.Close()

	client := newTestClient(ts)
	result, err := client.GetCredentialSecret(context.Background(), "tenant1", "cred1", "token")
	require.NoError(t, err)
	assert.Equal(t, "secret-value", result)
}

func TestGetCredentialSecret_MissingBaseURL(t *testing.T) {
	client := platform.NewClient(&platform.ClientConfig{})
	_, err := client.GetCredentialSecret(context.Background(), "t", "c", "token")
	assert.Error(t, err)
}

func TestGetCredentialSecret_MissingAuthToken(t *testing.T) {
	client := platform.NewClient(&platform.ClientConfig{BaseURL: "http://localhost"})
	_, err := client.GetCredentialSecret(context.Background(), "t", "c", "")
	assert.Error(t, err)
}

// --- UpdateConversationTitle ---

func TestUpdateConversationTitle_Success(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodPatch, r.Method)
		assert.Equal(t, "application/json", r.Header.Get("Content-Type"))
		var body map[string]string
		json.NewDecoder(r.Body).Decode(&body)
		assert.Equal(t, "New Title", body["name"])
		w.WriteHeader(200)
	}))
	defer ts.Close()

	client := newTestClient(ts)
	err := client.UpdateConversationTitle(context.Background(), "tenant1", "conv1", "New Title", "token")
	assert.NoError(t, err)
}

func TestUpdateConversationTitle_MissingBaseURL(t *testing.T) {
	client := platform.NewClient(&platform.ClientConfig{})
	err := client.UpdateConversationTitle(context.Background(), "t", "c", "title", "token")
	assert.Error(t, err)
}

func TestUpdateConversationTitle_MissingAuthToken(t *testing.T) {
	client := platform.NewClient(&platform.ClientConfig{BaseURL: "http://localhost"})
	err := client.UpdateConversationTitle(context.Background(), "t", "c", "title", "")
	assert.Error(t, err)
}

func TestUpdateConversationTitle_NotFound(t *testing.T) {
	ts := httptest.NewServer(statusHandler(404, "not found"))
	defer ts.Close()
	client := newTestClient(ts)
	err := client.UpdateConversationTitle(context.Background(), "t", "c", "title", "token")
	assert.Error(t, err)
}

// --- Headers verification ---

func TestBearerHeaders_IncludesServiceKey(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "Bearer auth-token", r.Header.Get("Authorization"))
		assert.Equal(t, "test-service-key", r.Header.Get("X-Service-Key"))
		json.NewEncoder(w).Encode(platform.UserInfo{ID: "u1"})
	}))
	defer ts.Close()

	client := newTestClient(ts)
	_, err := client.GetMe(context.Background(), "auth-token")
	assert.NoError(t, err)
}

func TestAPIKeyHeaders(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "my-api-key", r.Header.Get("X-Unified-UI-Autonomous-Agent-API-Key"))
		json.NewEncoder(w).Encode(platform.AutonomousAgentConfigResponse{TenantID: "t1", AutonomousAgentID: "aa1"})
	}))
	defer ts.Close()

	client := newTestClient(ts)
	_, err := client.GetAutonomousAgentConfig(context.Background(), "t1", "aa1", "my-api-key")
	assert.NoError(t, err)
}
