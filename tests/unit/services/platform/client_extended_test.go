package platform_test

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/unifiedui/agent-service/internal/services/platform"
)

// === GetAgentConfig Extended Tests ===

func TestGetAgentConfig_ErrorFromGetChatAgentConfig(t *testing.T) {
	// Server returns 500 error
	ts := httptest.NewServer(statusHandler(500, "internal server error"))
	defer ts.Close()

	client := newTestClient(ts)
	_, err := client.GetAgentConfig(context.Background(), "tenant1", "agent1", "conv1", "auth-token", true)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to get chat agent config")
}

func TestGetAgentConfig_WithUserInfo(t *testing.T) {
	resp := platform.ChatAgentConfigResponse{
		DocVersion:  "2",
		Type:        platform.AgentTypeFoundry,
		TenantID:    "tenant1",
		ChatAgentID: "agent1",
		Settings: platform.AgentSettings{
			APIVersion:            "v1",
			UseUnifiedChatHistory: true,
			ChatHistoryCount:      10,
			ProjectEndpoint:       "https://foundry.azure.com/project",
			AgentName:             "test-agent",
		},
		User: &platform.UserInfo{
			ID:          "user1",
			DisplayName: "Test User",
			Mail:        "test@example.com",
		},
	}
	ts := httptest.NewServer(jsonHandler(resp))
	defer ts.Close()

	client := newTestClient(ts)
	result, err := client.GetAgentConfig(context.Background(), "tenant1", "agent1", "conv1", "auth-token", true)
	require.NoError(t, err)
	assert.Equal(t, "2", result.DocVersion)
	assert.Equal(t, platform.AgentTypeFoundry, result.Type)
	assert.Equal(t, "conv1", result.ConversationID)
	assert.NotNil(t, result.User)
	assert.Equal(t, "user1", result.User.ID)
	assert.Equal(t, "test-agent", result.Settings.AgentName)
}

func TestGetAgentConfig_Unauthorized(t *testing.T) {
	ts := httptest.NewServer(statusHandler(401, "invalid token"))
	defer ts.Close()

	client := newTestClient(ts)
	_, err := client.GetAgentConfig(context.Background(), "tenant1", "agent1", "conv1", "bad-token", true)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unauthorized")
}

// === GetConversation Extended Tests ===

func TestGetConversation_MissingAuthToken(t *testing.T) {
	client := platform.NewClient(&platform.ClientConfig{BaseURL: "http://localhost", ServiceKey: "key"})
	_, err := client.GetConversation(context.Background(), "tenant1", "conv1", "")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "auth token")
}

func TestGetConversation_NotFound(t *testing.T) {
	ts := httptest.NewServer(statusHandler(404, "conversation not found"))
	defer ts.Close()

	client := newTestClient(ts)
	_, err := client.GetConversation(context.Background(), "tenant1", "conv1", "token")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not_found")
}

func TestGetConversation_Forbidden(t *testing.T) {
	ts := httptest.NewServer(statusHandler(403, "forbidden"))
	defer ts.Close()

	client := newTestClient(ts)
	_, err := client.GetConversation(context.Background(), "tenant1", "conv1", "token")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "forbidden")
}

func TestGetConversation_ServerError(t *testing.T) {
	ts := httptest.NewServer(statusHandler(500, "server error"))
	defer ts.Close()

	client := newTestClient(ts)
	_, err := client.GetConversation(context.Background(), "tenant1", "conv1", "token")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "500")
}

func TestGetConversation_WithExtConversationID(t *testing.T) {
	resp := platform.ConversationResponse{
		ID:                "conv1",
		Name:              "Test Conversation",
		TenantID:          "tenant1",
		ChatAgentID:       "agent1",
		ExtConversationID: "ext-conv-123",
	}
	ts := httptest.NewServer(jsonHandler(resp))
	defer ts.Close()

	client := newTestClient(ts)
	result, err := client.GetConversation(context.Background(), "tenant1", "conv1", "token")
	require.NoError(t, err)
	assert.Equal(t, "ext-conv-123", result.ExtConversationID)
}

// === GetAutonomousAgentConfigWithBearer Extended Tests ===

func TestGetAutonomousAgentConfigWithBearer_MissingBaseURL(t *testing.T) {
	client := platform.NewClient(&platform.ClientConfig{ServiceKey: "key"})
	_, err := client.GetAutonomousAgentConfigWithBearer(context.Background(), "tenant1", "aa1", "token")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not configured")
}

func TestGetAutonomousAgentConfigWithBearer_NotFound(t *testing.T) {
	ts := httptest.NewServer(statusHandler(404, "autonomous agent not found"))
	defer ts.Close()

	client := newTestClient(ts)
	_, err := client.GetAutonomousAgentConfigWithBearer(context.Background(), "tenant1", "aa1", "token")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not_found")
}

func TestGetAutonomousAgentConfigWithBearer_WithFullSettings(t *testing.T) {
	resp := platform.AutonomousAgentConfigResponse{
		DocVersion:        "1",
		Type:              platform.AgentTypeN8N,
		TenantID:          "tenant1",
		AutonomousAgentID: "aa1",
		Settings: platform.AutonomousAgentConfigSettings{
			APIVersion:          "v1",
			N8NHost:             "https://n8n.example.com",
			N8NWorkflowEndpoint: "/webhook/workflow",
			WorkflowID:          "wf123",
		},
	}
	ts := httptest.NewServer(jsonHandler(resp))
	defer ts.Close()

	client := newTestClient(ts)
	result, err := client.GetAutonomousAgentConfigWithBearer(context.Background(), "tenant1", "aa1", "token")
	require.NoError(t, err)
	assert.Equal(t, "wf123", result.Settings.WorkflowID)
	assert.Equal(t, "https://n8n.example.com", result.Settings.N8NHost)
}

// === GetCredentialSecret Extended Tests ===

func TestGetCredentialSecret_NotFound(t *testing.T) {
	ts := httptest.NewServer(statusHandler(404, "credential not found"))
	defer ts.Close()

	client := newTestClient(ts)
	_, err := client.GetCredentialSecret(context.Background(), "tenant1", "cred1", "token")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not_found")
}

func TestGetCredentialSecret_Unauthorized(t *testing.T) {
	ts := httptest.NewServer(statusHandler(401, "unauthorized"))
	defer ts.Close()

	client := newTestClient(ts)
	_, err := client.GetCredentialSecret(context.Background(), "tenant1", "cred1", "token")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unauthorized")
}

func TestGetCredentialSecret_ServerError(t *testing.T) {
	ts := httptest.NewServer(statusHandler(500, "internal server error"))
	defer ts.Close()

	client := newTestClient(ts)
	_, err := client.GetCredentialSecret(context.Background(), "tenant1", "cred1", "token")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "500")
}

// === UpdateConversationTitle Extended Tests ===

func TestUpdateConversationTitle_Unauthorized(t *testing.T) {
	ts := httptest.NewServer(statusHandler(401, "unauthorized"))
	defer ts.Close()

	client := newTestClient(ts)
	err := client.UpdateConversationTitle(context.Background(), "tenant1", "conv1", "New Title", "bad-token")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unauthorized")
}

func TestUpdateConversationTitle_Forbidden(t *testing.T) {
	ts := httptest.NewServer(statusHandler(403, "forbidden"))
	defer ts.Close()

	client := newTestClient(ts)
	err := client.UpdateConversationTitle(context.Background(), "tenant1", "conv1", "New Title", "token")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "forbidden")
}

func TestUpdateConversationTitle_ServerError(t *testing.T) {
	ts := httptest.NewServer(statusHandler(500, "server error"))
	defer ts.Close()

	client := newTestClient(ts)
	err := client.UpdateConversationTitle(context.Background(), "tenant1", "conv1", "New Title", "token")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "500")
}

func TestUpdateConversationTitle_VerifiesHeaders(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodPatch, r.Method)
		assert.Equal(t, "application/json", r.Header.Get("Content-Type"))
		assert.Equal(t, "Bearer my-token", r.Header.Get("Authorization"))
		assert.Equal(t, "test-service-key", r.Header.Get("X-Service-Key"))
		w.WriteHeader(200)
	}))
	defer ts.Close()

	client := newTestClient(ts)
	err := client.UpdateConversationTitle(context.Background(), "tenant1", "conv1", "Title", "my-token")
	assert.NoError(t, err)
}

// === ValidateConversation Extended Tests ===

func TestValidateConversation_Unauthorized(t *testing.T) {
	ts := httptest.NewServer(statusHandler(401, "unauthorized"))
	defer ts.Close()

	client := newTestClient(ts)
	err := client.ValidateConversation(context.Background(), "tenant1", "conv1", "token")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unauthorized")
}

func TestValidateConversation_Forbidden(t *testing.T) {
	ts := httptest.NewServer(statusHandler(403, "forbidden"))
	defer ts.Close()

	client := newTestClient(ts)
	err := client.ValidateConversation(context.Background(), "tenant1", "conv1", "token")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "forbidden")
}

func TestValidateConversation_ServerError(t *testing.T) {
	ts := httptest.NewServer(statusHandler(500, "server error"))
	defer ts.Close()

	client := newTestClient(ts)
	err := client.ValidateConversation(context.Background(), "tenant1", "conv1", "token")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "500")
}

// === ValidateAutonomousAgent Extended Tests ===

func TestValidateAutonomousAgent_NotFound(t *testing.T) {
	ts := httptest.NewServer(statusHandler(404, "not found"))
	defer ts.Close()

	client := newTestClient(ts)
	err := client.ValidateAutonomousAgent(context.Background(), "tenant1", "aa1", "token")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not_found")
}

func TestValidateAutonomousAgent_Unauthorized(t *testing.T) {
	ts := httptest.NewServer(statusHandler(401, "unauthorized"))
	defer ts.Close()

	client := newTestClient(ts)
	err := client.ValidateAutonomousAgent(context.Background(), "tenant1", "aa1", "token")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unauthorized")
}

// === ValidateAutonomousAgentAPIKey Extended Tests ===

func TestValidateAutonomousAgentAPIKey_NotFound(t *testing.T) {
	ts := httptest.NewServer(statusHandler(404, "autonomous agent not found"))
	defer ts.Close()

	client := newTestClient(ts)
	err := client.ValidateAutonomousAgentAPIKey(context.Background(), "tenant1", "aa1", "api-key")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not_found")
}

func TestValidateAutonomousAgentAPIKey_Unauthorized(t *testing.T) {
	ts := httptest.NewServer(statusHandler(401, "invalid api key"))
	defer ts.Close()

	client := newTestClient(ts)
	err := client.ValidateAutonomousAgentAPIKey(context.Background(), "tenant1", "aa1", "bad-key")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unauthorized")
}

func TestValidateAutonomousAgentAPIKey_Forbidden(t *testing.T) {
	ts := httptest.NewServer(statusHandler(403, "forbidden"))
	defer ts.Close()

	client := newTestClient(ts)
	err := client.ValidateAutonomousAgentAPIKey(context.Background(), "tenant1", "aa1", "api-key")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "forbidden")
}

// === GetAutonomousAgentConfig Extended Tests ===

func TestGetAutonomousAgentConfig_NotFound(t *testing.T) {
	ts := httptest.NewServer(statusHandler(404, "not found"))
	defer ts.Close()

	client := newTestClient(ts)
	_, err := client.GetAutonomousAgentConfig(context.Background(), "tenant1", "aa1", "api-key")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not_found")
}

func TestGetAutonomousAgentConfig_Unauthorized(t *testing.T) {
	ts := httptest.NewServer(statusHandler(401, "unauthorized"))
	defer ts.Close()

	client := newTestClient(ts)
	_, err := client.GetAutonomousAgentConfig(context.Background(), "tenant1", "aa1", "bad-key")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unauthorized")
}

func TestGetAutonomousAgentConfig_WithCredentials(t *testing.T) {
	resp := platform.AutonomousAgentConfigResponse{
		DocVersion:        "1",
		Type:              platform.AgentTypeN8N,
		TenantID:          "tenant1",
		AutonomousAgentID: "aa1",
		Settings: platform.AutonomousAgentConfigSettings{
			APIVersion: "v1",
			APICredentials: &platform.Credentials{
				ID:   "cred1",
				Name: "N8N API Key",
				Type: platform.CredentialTypeN8NAPIKey,
			},
		},
	}
	ts := httptest.NewServer(jsonHandler(resp))
	defer ts.Close()

	client := newTestClient(ts)
	result, err := client.GetAutonomousAgentConfig(context.Background(), "tenant1", "aa1", "api-key")
	require.NoError(t, err)
	require.NotNil(t, result.Settings.APICredentials)
	assert.Equal(t, "cred1", result.Settings.APICredentials.ID)
}

// === GetAIModelsByPurpose Extended Tests ===

func TestGetAIModelsByPurpose_NotFound(t *testing.T) {
	ts := httptest.NewServer(statusHandler(404, "not found"))
	defer ts.Close()

	client := newTestClient(ts)
	_, err := client.GetAIModelsByPurpose(context.Background(), "tenant1", "UNKNOWN_PURPOSE", "")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not_found")
}

func TestGetAIModelsByPurpose_EmptyResult(t *testing.T) {
	ts := httptest.NewServer(jsonHandler([]platform.AIModelWithSecretResponse{}))
	defer ts.Close()

	client := newTestClient(ts)
	result, err := client.GetAIModelsByPurpose(context.Background(), "tenant1", "TITLE_GEN", "")
	require.NoError(t, err)
	assert.Empty(t, result)
}

func TestGetAIModelsByPurpose_MultipleModels(t *testing.T) {
	resp := []platform.AIModelWithSecretResponse{
		{ID: "m1", Provider: "OPENAI", Priority: 1, Config: map[string]interface{}{"model_name": "gpt-4"}},
		{ID: "m2", Provider: "AZURE_OPENAI", Priority: 2, Config: map[string]interface{}{"deployment_name": "gpt-4-deployment"}},
	}
	ts := httptest.NewServer(jsonHandler(resp))
	defer ts.Close()

	client := newTestClient(ts)
	result, err := client.GetAIModelsByPurpose(context.Background(), "tenant1", "TITLE_GEN", "")
	require.NoError(t, err)
	assert.Len(t, result, 2)
	assert.Equal(t, "m1", result[0].ID)
	assert.Equal(t, "m2", result[1].ID)
}

func TestGetAIModelsByPurpose_WithoutModelType(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.NotContains(t, r.URL.RawQuery, "model_type")
		json.NewEncoder(w).Encode([]platform.AIModelWithSecretResponse{})
	}))
	defer ts.Close()

	client := newTestClient(ts)
	_, err := client.GetAIModelsByPurpose(context.Background(), "tenant1", "TITLE_GEN", "")
	assert.NoError(t, err)
}

func TestGetAIModelsByPurpose_Unauthorized(t *testing.T) {
	ts := httptest.NewServer(statusHandler(401, "unauthorized"))
	defer ts.Close()

	client := newTestClient(ts)
	_, err := client.GetAIModelsByPurpose(context.Background(), "tenant1", "TITLE_GEN", "")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unauthorized")
}

// === GetMe Extended Tests ===

func TestGetMe_NotFound(t *testing.T) {
	ts := httptest.NewServer(statusHandler(404, "user not found"))
	defer ts.Close()

	client := newTestClient(ts)
	_, err := client.GetMe(context.Background(), "token")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not_found")
}

func TestGetMe_Unauthorized(t *testing.T) {
	ts := httptest.NewServer(statusHandler(401, "invalid token"))
	defer ts.Close()

	client := newTestClient(ts)
	_, err := client.GetMe(context.Background(), "bad-token")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unauthorized")
}

func TestGetMe_Forbidden(t *testing.T) {
	ts := httptest.NewServer(statusHandler(403, "forbidden"))
	defer ts.Close()

	client := newTestClient(ts)
	_, err := client.GetMe(context.Background(), "token")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "forbidden")
}

func TestGetMe_WithFullUserInfo(t *testing.T) {
	resp := platform.UserInfo{
		ID:               "user-123",
		IdentityProvider: "azure-ad",
		IdentityTenantID: "tenant-abc",
		DisplayName:      "John Doe",
		PrincipalName:    "john.doe@example.com",
		Firstname:        "John",
		Lastname:         "Doe",
		Mail:             "john.doe@example.com",
		Tenants: []map[string]interface{}{
			{"id": "tenant1", "name": "Tenant One"},
		},
		Groups: []map[string]interface{}{
			{"id": "group1", "name": "Admins"},
		},
	}
	ts := httptest.NewServer(jsonHandler(resp))
	defer ts.Close()

	client := newTestClient(ts)
	result, err := client.GetMe(context.Background(), "token")
	require.NoError(t, err)
	assert.Equal(t, "John Doe", result.DisplayName)
	assert.Equal(t, "azure-ad", result.IdentityProvider)
	assert.Len(t, result.Tenants, 1)
	assert.Len(t, result.Groups, 1)
}

// === GetChatAgentConfig Extended Tests ===

func TestGetChatAgentConfig_WithFullSettings(t *testing.T) {
	resp := platform.ChatAgentConfigResponse{
		DocVersion:  "3",
		Type:        platform.AgentTypeN8N,
		TenantID:    "tenant1",
		ChatAgentID: "agent1",
		Settings: platform.AgentSettings{
			APIVersion:            "v2",
			UseUnifiedChatHistory: true,
			ChatHistoryCount:      50,
			WorkflowType:          platform.N8NWorkflowTypeChatAgent,
			ChatURL:               "https://n8n.example.com/webhook/chat",
			APICredentials: &platform.Credentials{
				ID:             "cred1",
				CredentialsURI: "/credentials/cred1",
				Name:           "API Key",
				Type:           platform.CredentialTypeN8NAPIKey,
				IsActive:       true,
			},
			ChatCredentials: &platform.Credentials{
				ID:             "cred2",
				CredentialsURI: "/credentials/cred2",
				Name:           "Basic Auth",
				Type:           platform.CredentialTypeN8NBasicAuth,
				IsActive:       true,
			},
		},
		User: &platform.UserInfo{
			ID:          "user1",
			DisplayName: "Test User",
		},
	}
	ts := httptest.NewServer(jsonHandler(resp))
	defer ts.Close()

	client := newTestClient(ts)
	result, err := client.GetChatAgentConfig(context.Background(), "tenant1", "agent1", "auth-token", true)
	require.NoError(t, err)
	assert.Equal(t, "3", result.DocVersion)
	assert.Equal(t, platform.N8NWorkflowTypeChatAgent, result.Settings.WorkflowType)
	assert.NotNil(t, result.Settings.APICredentials)
	assert.NotNil(t, result.Settings.ChatCredentials)
	assert.Equal(t, 50, result.Settings.ChatHistoryCount)
}

func TestGetChatAgentConfig_WithFoundrySettings(t *testing.T) {
	resp := platform.ChatAgentConfigResponse{
		DocVersion:  "1",
		Type:        platform.AgentTypeFoundry,
		TenantID:    "tenant1",
		ChatAgentID: "agent1",
		Settings: platform.AgentSettings{
			AgentType:       "MULTI_AGENT",
			ProjectEndpoint: "https://eastus.api.azureml.ms/agents/v1.0/projects/my-project",
			AgentName:       "multi-agent-orchestrator",
		},
	}
	ts := httptest.NewServer(jsonHandler(resp))
	defer ts.Close()

	client := newTestClient(ts)
	result, err := client.GetChatAgentConfig(context.Background(), "tenant1", "agent1", "auth-token", true)
	require.NoError(t, err)
	assert.Equal(t, platform.AgentTypeFoundry, result.Type)
	assert.Equal(t, "MULTI_AGENT", result.Settings.AgentType)
	assert.Contains(t, result.Settings.ProjectEndpoint, "azureml.ms")
}

// === JSON Parsing Error Tests ===

func TestGetChatAgentConfig_InvalidJSON(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		w.Write([]byte("not valid json"))
	}))
	defer ts.Close()

	client := newTestClient(ts)
	_, err := client.GetChatAgentConfig(context.Background(), "tenant1", "agent1", "token", true)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "parse")
}

func TestGetConversation_InvalidJSON(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		w.Write([]byte("{invalid"))
	}))
	defer ts.Close()

	client := newTestClient(ts)
	_, err := client.GetConversation(context.Background(), "tenant1", "conv1", "token")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "parse")
}

func TestGetAIModelsByPurpose_InvalidJSON(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		w.Write([]byte("not a json array"))
	}))
	defer ts.Close()

	client := newTestClient(ts)
	_, err := client.GetAIModelsByPurpose(context.Background(), "tenant1", "TITLE_GEN", "")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "parse")
}

func TestGetMe_InvalidJSON(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		w.Write([]byte(":::invalid:::"))
	}))
	defer ts.Close()

	client := newTestClient(ts)
	_, err := client.GetMe(context.Background(), "token")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "parse")
}

func TestGetCredentialSecret_InvalidJSON(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		w.Write([]byte("not json"))
	}))
	defer ts.Close()

	client := newTestClient(ts)
	_, err := client.GetCredentialSecret(context.Background(), "tenant1", "cred1", "token")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "parse")
}

func TestGetAutonomousAgentConfig_InvalidJSON(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		w.Write([]byte("bad json"))
	}))
	defer ts.Close()

	client := newTestClient(ts)
	_, err := client.GetAutonomousAgentConfig(context.Background(), "tenant1", "aa1", "api-key")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "parse")
}

// === Context Cancellation Tests ===

func TestGetChatAgentConfig_ContextCanceled(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(100 * time.Millisecond)
		json.NewEncoder(w).Encode(platform.ChatAgentConfigResponse{})
	}))
	defer ts.Close()

	client := newTestClient(ts)
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	_, err := client.GetChatAgentConfig(ctx, "tenant1", "agent1", "token", true)
	require.Error(t, err)
}

func TestGetMe_ContextTimeout(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(100 * time.Millisecond)
		json.NewEncoder(w).Encode(platform.UserInfo{})
	}))
	defer ts.Close()

	client := newTestClient(ts)
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Millisecond)
	defer cancel()

	_, err := client.GetMe(ctx, "token")
	require.Error(t, err)
}

// === URL endpoint verification ===

func TestGetChatAgentConfig_URLConstruction(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/api/v1/platform-service/tenants/my-tenant/chat-agents/my-agent/config", r.URL.Path)
		assert.Equal(t, http.MethodGet, r.Method)
		json.NewEncoder(w).Encode(platform.ChatAgentConfigResponse{})
	}))
	defer ts.Close()

	client := newTestClient(ts)
	_, err := client.GetChatAgentConfig(context.Background(), "my-tenant", "my-agent", "token", true)
	assert.NoError(t, err)
}

func TestGetConversation_URLConstruction(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/api/v1/platform-service/tenants/tenant-x/conversations/conv-y", r.URL.Path)
		json.NewEncoder(w).Encode(platform.ConversationResponse{})
	}))
	defer ts.Close()

	client := newTestClient(ts)
	_, err := client.GetConversation(context.Background(), "tenant-x", "conv-y", "token")
	assert.NoError(t, err)
}

func TestGetMe_URLConstruction(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/api/v1/platform-service/identity/me", r.URL.Path)
		json.NewEncoder(w).Encode(platform.UserInfo{})
	}))
	defer ts.Close()

	client := newTestClient(ts)
	_, err := client.GetMe(context.Background(), "token")
	assert.NoError(t, err)
}

func TestGetAIModelsByPurpose_URLConstruction(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/api/v1/platform-service/tenants/t1/ai-models/by-purpose/CHAT_COMPLETION", r.URL.Path)
		assert.Equal(t, "model_type=LLM_MODEL", r.URL.RawQuery)
		json.NewEncoder(w).Encode([]platform.AIModelWithSecretResponse{})
	}))
	defer ts.Close()

	client := newTestClient(ts)
	_, err := client.GetAIModelsByPurpose(context.Background(), "t1", "CHAT_COMPLETION", "LLM_MODEL")
	assert.NoError(t, err)
}

func TestGetCredentialSecret_URLConstruction(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/api/v1/platform-service/tenants/t1/credentials/cred-123/secret", r.URL.Path)
		json.NewEncoder(w).Encode(platform.CredentialSecretResponse{SecretValue: "secret"})
	}))
	defer ts.Close()

	client := newTestClient(ts)
	_, err := client.GetCredentialSecret(context.Background(), "t1", "cred-123", "token")
	assert.NoError(t, err)
}

func TestValidateAutonomousAgentAPIKey_URLConstruction(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/api/v1/platform-service/tenants/t1/autonomous-agents/aa1/validate-api-key", r.URL.Path)
		assert.Equal(t, http.MethodPost, r.Method)
		w.WriteHeader(200)
	}))
	defer ts.Close()

	client := newTestClient(ts)
	err := client.ValidateAutonomousAgentAPIKey(context.Background(), "t1", "aa1", "key")
	assert.NoError(t, err)
}

func TestUpdateConversationTitle_URLConstruction(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/api/v1/platform-service/tenants/t1/conversations/c1", r.URL.Path)
		assert.Equal(t, http.MethodPatch, r.Method)
		body, _ := io.ReadAll(r.Body)
		var payload map[string]string
		json.Unmarshal(body, &payload)
		assert.Equal(t, "My New Title", payload["name"])
		w.WriteHeader(200)
	}))
	defer ts.Close()

	client := newTestClient(ts)
	err := client.UpdateConversationTitle(context.Background(), "t1", "c1", "My New Title", "token")
	assert.NoError(t, err)
}

// === Service Key Header Tests ===

func TestGetAIModelsByPurpose_UsesServiceKeyHeader(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "test-service-key", r.Header.Get("X-Service-Key"))
		assert.Empty(t, r.Header.Get("Authorization")) // No bearer token for service key auth
		json.NewEncoder(w).Encode([]platform.AIModelWithSecretResponse{})
	}))
	defer ts.Close()

	client := newTestClient(ts)
	_, err := client.GetAIModelsByPurpose(context.Background(), "tenant1", "TITLE_GEN", "")
	assert.NoError(t, err)
}

// === Network Error Tests ===

func TestGetChatAgentConfig_NetworkError(t *testing.T) {
	// Create server and close it immediately to simulate network error
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	ts.Close()

	client := newTestClient(ts)
	_, err := client.GetChatAgentConfig(context.Background(), "tenant1", "agent1", "token", true)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to call platform service")
}

func TestGetMe_NetworkError(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	ts.Close()

	client := newTestClient(ts)
	_, err := client.GetMe(context.Background(), "token")
	require.Error(t, err)
}

func TestValidateConversation_NetworkError(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	ts.Close()

	client := newTestClient(ts)
	err := client.ValidateConversation(context.Background(), "t", "c", "token")
	require.Error(t, err)
}

func TestUpdateConversationTitle_NetworkError(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	ts.Close()

	client := newTestClient(ts)
	err := client.UpdateConversationTitle(context.Background(), "t", "c", "title", "token")
	require.Error(t, err)
}

// === Accept Header Verification ===

func TestDoRawRequest_SetsAcceptHeader(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "application/json", r.Header.Get("Accept"))
		json.NewEncoder(w).Encode(platform.UserInfo{ID: "u1"})
	}))
	defer ts.Close()

	client := newTestClient(ts)
	_, err := client.GetMe(context.Background(), "token")
	assert.NoError(t, err)
}

// === Empty Response Body Handling ===

func TestValidateConversation_EmptyBody(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		// Don't write any body
	}))
	defer ts.Close()

	client := newTestClient(ts)
	err := client.ValidateConversation(context.Background(), "tenant1", "conv1", "token")
	assert.NoError(t, err)
}

func TestValidateAutonomousAgent_EmptyBody(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
	}))
	defer ts.Close()

	client := newTestClient(ts)
	err := client.ValidateAutonomousAgent(context.Background(), "tenant1", "aa1", "token")
	assert.NoError(t, err)
}

func TestValidateAutonomousAgentAPIKey_EmptyBody(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
	}))
	defer ts.Close()

	client := newTestClient(ts)
	err := client.ValidateAutonomousAgentAPIKey(context.Background(), "tenant1", "aa1", "api-key")
	assert.NoError(t, err)
}

func TestUpdateConversationTitle_EmptyBody(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
	}))
	defer ts.Close()

	client := newTestClient(ts)
	err := client.UpdateConversationTitle(context.Background(), "tenant1", "conv1", "New Title", "token")
	assert.NoError(t, err)
}

// === Client without service key (bearerHeaders behavior) ===

func TestBearerHeaders_WithoutServiceKey(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "Bearer my-token", r.Header.Get("Authorization"))
		assert.Empty(t, r.Header.Get("X-Service-Key"))
		json.NewEncoder(w).Encode(platform.UserInfo{ID: "u1"})
	}))
	defer ts.Close()

	client := platform.NewClient(&platform.ClientConfig{
		BaseURL: ts.URL,
		Timeout: 0,
	})
	_, err := client.GetMe(context.Background(), "my-token")
	assert.NoError(t, err)
}

// === Agent Type Constants Tests ===

func TestAgentTypes(t *testing.T) {
	assert.Equal(t, platform.AgentType("N8N"), platform.AgentTypeN8N)
	assert.Equal(t, platform.AgentType("MICROSOFT_FOUNDRY"), platform.AgentTypeFoundry)
	assert.Equal(t, platform.AgentType("COPILOT"), platform.AgentTypeCopilot)
	assert.Equal(t, platform.AgentType("CUSTOM"), platform.AgentTypeCustom)
}

func TestCredentialTypes(t *testing.T) {
	assert.Equal(t, platform.CredentialType("N8N_API_KEY"), platform.CredentialTypeN8NAPIKey)
	assert.Equal(t, platform.CredentialType("N8N_BASIC_AUTH"), platform.CredentialTypeN8NBasicAuth)
	assert.Equal(t, platform.CredentialType("BEARER_TOKEN"), platform.CredentialTypeBearerToken)
}

func TestN8NWorkflowTypes(t *testing.T) {
	assert.Equal(t, platform.N8NWorkflowType("N8N_CHAT_AGENT_WORKFLOW"), platform.N8NWorkflowTypeChatAgent)
	assert.Equal(t, platform.N8NWorkflowType("N8N_HUMAN_IN_THE_LOOP"), platform.N8NWorkflowTypeHumanInLoop)
}
