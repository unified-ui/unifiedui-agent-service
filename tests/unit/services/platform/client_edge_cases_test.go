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

// === GetAgentConfigFromFile Edge Cases ===

func TestGetAgentConfigFromFile_DirectoryInsteadOfFile(t *testing.T) {
	tmpDir := t.TempDir()

	client := platform.NewClient(&platform.ClientConfig{ConfigPath: tmpDir})
	_, err := client.GetAgentConfigFromFile(context.Background(), "tenant1", "agent1")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "read config file")
}

func TestGetAgentConfigFromFile_EmptyFile(t *testing.T) {
	tmpDir := t.TempDir()
	configFile := filepath.Join(tmpDir, "empty.json")
	os.WriteFile(configFile, []byte(""), 0o644)

	client := platform.NewClient(&platform.ClientConfig{ConfigPath: configFile})
	_, err := client.GetAgentConfigFromFile(context.Background(), "tenant1", "agent1")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "parse")
}

func TestGetAgentConfigFromFile_FullConfig(t *testing.T) {
	tmpDir := t.TempDir()
	configFile := filepath.Join(tmpDir, "full.json")
	config := platform.AgentConfig{
		DocVersion:     "2",
		Type:           platform.AgentTypeN8N,
		TenantID:       "my-tenant",
		ConversationID: "my-conv",
		ChatAgentID:    "my-agent",
		Settings: platform.AgentSettings{
			APIVersion:            "v2",
			UseUnifiedChatHistory: true,
			ChatHistoryCount:      25,
			WorkflowType:          platform.N8NWorkflowTypeHumanInLoop,
			ChatURL:               "https://n8n.local/webhook",
		},
		User: &platform.UserInfo{
			ID:          "user-1",
			DisplayName: "Test User",
		},
	}
	data, _ := json.Marshal(config)
	os.WriteFile(configFile, data, 0o644)

	client := platform.NewClient(&platform.ClientConfig{ConfigPath: configFile})
	result, err := client.GetAgentConfigFromFile(context.Background(), "tenant1", "agent1")
	require.NoError(t, err)
	assert.Equal(t, "my-tenant", result.TenantID)
	assert.Equal(t, platform.N8NWorkflowTypeHumanInLoop, result.Settings.WorkflowType)
	assert.NotNil(t, result.User)
}

// === doJSONSliceRequest error path coverage ===

func TestGetAIModelsByPurpose_ServerError(t *testing.T) {
	ts := httptest.NewServer(statusHandler(500, "internal server error"))
	defer ts.Close()

	client := newTestClient(ts)
	_, err := client.GetAIModelsByPurpose(context.Background(), "tenant1", "TITLE_GEN", "")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "500")
}

func TestGetAIModelsByPurpose_Forbidden(t *testing.T) {
	ts := httptest.NewServer(statusHandler(403, "forbidden"))
	defer ts.Close()

	client := newTestClient(ts)
	_, err := client.GetAIModelsByPurpose(context.Background(), "tenant1", "TITLE_GEN", "")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "forbidden")
}

// === Complex Response Data Tests ===

func TestGetAIModelsByPurpose_WithCredentialSecrets(t *testing.T) {
	resp := []platform.AIModelWithSecretResponse{
		{
			ID:       "model-1",
			Type:     "LLM_MODEL",
			Provider: "AZURE_OPENAI",
			Priority: 1,
			Config: map[string]interface{}{
				"deployment_name": "gpt-4-turbo",
				"api_version":     "2024-02-01",
				"endpoint":        "https://my-openai.openai.azure.com",
			},
			CredentialSecret: map[string]interface{}{
				"api_key": "sk-secret-key-12345",
			},
		},
		{
			ID:       "model-2",
			Type:     "EMBEDDING_MODEL",
			Provider: "OPENAI",
			Priority: 2,
			Config: map[string]interface{}{
				"model_name": "text-embedding-ada-002",
			},
			CredentialSecret: map[string]interface{}{
				"api_key": "sk-openai-key",
			},
		},
	}
	ts := httptest.NewServer(jsonHandler(resp))
	defer ts.Close()

	client := newTestClient(ts)
	result, err := client.GetAIModelsByPurpose(context.Background(), "tenant1", "CHAT", "")
	require.NoError(t, err)
	assert.Len(t, result, 2)
	assert.Equal(t, "AZURE_OPENAI", result[0].Provider)
	assert.NotNil(t, result[0].CredentialSecret)
	assert.Equal(t, "sk-secret-key-12345", result[0].CredentialSecret["api_key"])
}

// === Special Characters in IDs ===

func TestGetChatAgentConfig_SpecialCharactersInIDs(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Contains(t, r.URL.Path, "tenant-123")
		assert.Contains(t, r.URL.Path, "agent-456")
		json.NewEncoder(w).Encode(platform.ChatAgentConfigResponse{
			TenantID:    "tenant-123",
			ChatAgentID: "agent-456",
		})
	}))
	defer ts.Close()

	client := newTestClient(ts)
	result, err := client.GetChatAgentConfig(context.Background(), "tenant-123", "agent-456", "token", true)
	require.NoError(t, err)
	assert.Equal(t, "tenant-123", result.TenantID)
}

// === Credentials Helper Methods Additional Tests ===

func TestCredentials_VariousSecretTypes(t *testing.T) {
	// Test with number
	credNum := &platform.Credentials{Secret: 12345}
	assert.Equal(t, "", credNum.GetSecretAsString())
	assert.Nil(t, credNum.GetSecretAsBasicAuth())

	// Test with bool
	credBool := &platform.Credentials{Secret: true}
	assert.Equal(t, "", credBool.GetSecretAsString())
	assert.Nil(t, credBool.GetSecretAsBasicAuth())

	// Test with array
	credArr := &platform.Credentials{Secret: []string{"a", "b"}}
	assert.Equal(t, "", credArr.GetSecretAsString())
	assert.Nil(t, credArr.GetSecretAsBasicAuth())
}

// === HTTP Method Verification Tests ===

func TestGetChatAgentConfig_HTTPMethod(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodGet, r.Method)
		json.NewEncoder(w).Encode(platform.ChatAgentConfigResponse{})
	}))
	defer ts.Close()

	client := newTestClient(ts)
	_, err := client.GetChatAgentConfig(context.Background(), "t", "a", "token", true)
	assert.NoError(t, err)
}

func TestGetConversation_HTTPMethod(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodGet, r.Method)
		json.NewEncoder(w).Encode(platform.ConversationResponse{})
	}))
	defer ts.Close()

	client := newTestClient(ts)
	_, err := client.GetConversation(context.Background(), "t", "c", "token")
	assert.NoError(t, err)
}

func TestValidateAutonomousAgentAPIKey_HTTPMethod(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodPost, r.Method)
		w.WriteHeader(200)
	}))
	defer ts.Close()

	client := newTestClient(ts)
	err := client.ValidateAutonomousAgentAPIKey(context.Background(), "t", "a", "key")
	assert.NoError(t, err)
}

// === ServiceConfigResponse backward compatibility ===

func TestServiceConfigResponse_BackwardCompatibility(t *testing.T) {
	resp := platform.ServiceConfigResponse{
		DocVersion:  "1",
		Type:        platform.AgentTypeN8N,
		TenantID:    "tenant1",
		ChatAgentID: "agent1",
		Settings: platform.AgentSettings{
			ChatURL: "http://n8n.local",
		},
	}

	data, err := json.Marshal(resp)
	require.NoError(t, err)

	var parsed platform.ServiceConfigResponse
	err = json.Unmarshal(data, &parsed)
	require.NoError(t, err)

	assert.Equal(t, "tenant1", parsed.TenantID)
	assert.Equal(t, platform.AgentTypeN8N, parsed.Type)
}

// === AutonomousAgentConfigSettings Additional Tests ===

func TestAutonomousAgentConfigSettings_AllFields(t *testing.T) {
	settings := platform.AutonomousAgentConfigSettings{
		APIVersion:          "v2",
		N8NHost:             "https://n8n.example.com",
		N8NWorkflowEndpoint: "/webhook/autonomous",
		WorkflowID:          "workflow-123-abc",
		APICredentials: &platform.Credentials{
			ID:             "cred-456",
			CredentialsURI: "/api/v1/credentials/cred-456",
			Name:           "Test Credential",
			Description:    "Test API Key",
			Type:           platform.CredentialTypeN8NAPIKey,
			IsActive:       true,
			Secret:         "api-key-secret-value",
		},
	}

	data, err := json.Marshal(settings)
	require.NoError(t, err)

	var parsed platform.AutonomousAgentConfigSettings
	err = json.Unmarshal(data, &parsed)
	require.NoError(t, err)

	assert.Equal(t, "v2", parsed.APIVersion)
	assert.Equal(t, "workflow-123-abc", parsed.WorkflowID)
	assert.NotNil(t, parsed.APICredentials)
	assert.Equal(t, "api-key-secret-value", parsed.APICredentials.GetSecretAsString())
}

// === Response Parsing Edge Cases ===

func TestGetChatAgentConfig_NullUserField(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{
			"docversion": "1",
			"type": "N8N",
			"tenant_id": "t1",
			"chat_agent_id": "a1",
			"settings": {},
			"user": null
		}`))
	}))
	defer ts.Close()

	client := newTestClient(ts)
	result, err := client.GetChatAgentConfig(context.Background(), "t1", "a1", "token", true)
	require.NoError(t, err)
	assert.Nil(t, result.User)
}

func TestGetConversation_EmptyExtConversationID(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{
			"id": "conv-1",
			"name": "My Conversation",
			"tenant_id": "t1",
			"chat_agent_id": "a1"
		}`))
	}))
	defer ts.Close()

	client := newTestClient(ts)
	result, err := client.GetConversation(context.Background(), "t1", "conv-1", "token")
	require.NoError(t, err)
	assert.Empty(t, result.ExtConversationID)
}

// === Client Configuration Tests ===

func TestNewClient_AllConfigOptions(t *testing.T) {
	client := platform.NewClient(&platform.ClientConfig{
		BaseURL:    "https://platform.example.com",
		ConfigPath: "/etc/unified-ui/config.json",
		ServiceKey: "service-key-abc-123",
		Timeout:    60000000000,
	})

	assert.NotNil(t, client)
}

func TestNewClient_ZeroTimeout(t *testing.T) {
	client := platform.NewClient(&platform.ClientConfig{
		BaseURL:    "http://localhost:8080",
		ServiceKey: "key",
		Timeout:    0,
	})
	assert.NotNil(t, client)
}

// === Large Response Handling ===

func TestGetAIModelsByPurpose_LargeResponse(t *testing.T) {
	models := make([]platform.AIModelWithSecretResponse, 100)
	for i := 0; i < 100; i++ {
		models[i] = platform.AIModelWithSecretResponse{
			ID:       "model-" + string(rune('0'+i%10)),
			Provider: "OPENAI",
			Priority: i,
			Config: map[string]interface{}{
				"model":     "gpt-4",
				"index":     i,
				"extra_key": "some extra data to make payload larger",
			},
		}
	}

	ts := httptest.NewServer(jsonHandler(models))
	defer ts.Close()

	client := newTestClient(ts)
	result, err := client.GetAIModelsByPurpose(context.Background(), "tenant1", "CHAT", "")
	require.NoError(t, err)
	assert.Len(t, result, 100)
}
