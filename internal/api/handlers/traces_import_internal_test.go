// Package handlers provides HTTP handlers for the API.
// This file contains internal tests for trace import helper functions.
package handlers

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/unifiedui/agent-service/internal/services/platform"
)

// setupTestContext creates a test Gin context with optional headers.
func setupTestContext(headers map[string]string) (*gin.Context, *httptest.ResponseRecorder) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPut, "/test", nil)
	for key, value := range headers {
		c.Request.Header.Set(key, value)
	}
	return c, w
}

// createTestTracesHandlerInternal creates a TracesHandler for internal testing.
func createTestTracesHandlerInternal() *TracesHandler {
	return &TracesHandler{}
}

// --- buildFoundryConfig Tests ---

func TestBuildFoundryConfig_ValidConfig(t *testing.T) {
	handler := createTestTracesHandlerInternal()

	headers := map[string]string{
		"X-Microsoft-Foundry-API-Key": "test-foundry-api-key",
	}
	c, _ := setupTestContext(headers)

	appConfig := &platform.ChatAgentConfigResponse{
		Type:        platform.AgentTypeFoundry,
		TenantID:    "test-tenant",
		ChatAgentID: "test-chat-agent",
		Settings: platform.AgentSettings{
			ProjectEndpoint: "https://my-project.api.azureml.ms",
			APIVersion:      "2025-01-01-preview",
		},
	}

	conversation := &platform.ConversationResponse{
		ID:                "conv-123",
		ExtConversationID: "ext-conv-456",
		TenantID:          "test-tenant",
		ChatAgentID:       "test-chat-agent",
	}

	config, err := handler.buildFoundryConfig(c, appConfig, conversation)

	require.NoError(t, err)
	assert.NotNil(t, config)
	assert.Equal(t, "ext-conv-456", config["ext_conversation_id"])
	assert.Equal(t, "https://my-project.api.azureml.ms", config["project_endpoint"])
	assert.Equal(t, "2025-01-01-preview", config["api_version"])
	assert.Equal(t, "test-foundry-api-key", config["api_token"])
}

func TestBuildFoundryConfig_DefaultAPIVersion(t *testing.T) {
	handler := createTestTracesHandlerInternal()

	headers := map[string]string{
		"X-Microsoft-Foundry-API-Key": "test-foundry-api-key",
	}
	c, _ := setupTestContext(headers)

	appConfig := &platform.ChatAgentConfigResponse{
		Type:        platform.AgentTypeFoundry,
		TenantID:    "test-tenant",
		ChatAgentID: "test-chat-agent",
		Settings: platform.AgentSettings{
			ProjectEndpoint: "https://my-project.api.azureml.ms",
			APIVersion:      "", // Empty - should use default
		},
	}

	conversation := &platform.ConversationResponse{
		ID:                "conv-123",
		ExtConversationID: "ext-conv-456",
		TenantID:          "test-tenant",
		ChatAgentID:       "test-chat-agent",
	}

	config, err := handler.buildFoundryConfig(c, appConfig, conversation)

	require.NoError(t, err)
	assert.NotNil(t, config)
	assert.Equal(t, "2025-11-15-preview", config["api_version"], "Should use default API version when not specified")
}

func TestBuildFoundryConfig_MissingAPIKey(t *testing.T) {
	handler := createTestTracesHandlerInternal()

	// No X-Microsoft-Foundry-API-Key header
	c, _ := setupTestContext(nil)

	appConfig := &platform.ChatAgentConfigResponse{
		Type:        platform.AgentTypeFoundry,
		TenantID:    "test-tenant",
		ChatAgentID: "test-chat-agent",
		Settings: platform.AgentSettings{
			ProjectEndpoint: "https://my-project.api.azureml.ms",
			APIVersion:      "2025-01-01-preview",
		},
	}

	conversation := &platform.ConversationResponse{
		ID:                "conv-123",
		ExtConversationID: "ext-conv-456",
		TenantID:          "test-tenant",
		ChatAgentID:       "test-chat-agent",
	}

	config, err := handler.buildFoundryConfig(c, appConfig, conversation)

	assert.Nil(t, config)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "X-Microsoft-Foundry-API-Key header is required")
}

func TestBuildFoundryConfig_MissingExtConversationID(t *testing.T) {
	handler := createTestTracesHandlerInternal()

	headers := map[string]string{
		"X-Microsoft-Foundry-API-Key": "test-foundry-api-key",
	}
	c, _ := setupTestContext(headers)

	appConfig := &platform.ChatAgentConfigResponse{
		Type:        platform.AgentTypeFoundry,
		TenantID:    "test-tenant",
		ChatAgentID: "test-chat-agent",
		Settings: platform.AgentSettings{
			ProjectEndpoint: "https://my-project.api.azureml.ms",
			APIVersion:      "2025-01-01-preview",
		},
	}

	conversation := &platform.ConversationResponse{
		ID:                "conv-123",
		ExtConversationID: "", // Missing
		TenantID:          "test-tenant",
		ChatAgentID:       "test-chat-agent",
	}

	config, err := handler.buildFoundryConfig(c, appConfig, conversation)

	assert.Nil(t, config)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "conversation has no external conversation ID")
}

func TestBuildFoundryConfig_MissingProjectEndpoint(t *testing.T) {
	handler := createTestTracesHandlerInternal()

	headers := map[string]string{
		"X-Microsoft-Foundry-API-Key": "test-foundry-api-key",
	}
	c, _ := setupTestContext(headers)

	appConfig := &platform.ChatAgentConfigResponse{
		Type:        platform.AgentTypeFoundry,
		TenantID:    "test-tenant",
		ChatAgentID: "test-chat-agent",
		Settings: platform.AgentSettings{
			ProjectEndpoint: "", // Missing
			APIVersion:      "2025-01-01-preview",
		},
	}

	conversation := &platform.ConversationResponse{
		ID:                "conv-123",
		ExtConversationID: "ext-conv-456",
		TenantID:          "test-tenant",
		ChatAgentID:       "test-chat-agent",
	}

	config, err := handler.buildFoundryConfig(c, appConfig, conversation)

	assert.Nil(t, config)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "chat agent configuration missing project endpoint")
}

func TestBuildFoundryConfig_WithMicrosoftGraphCredentials(t *testing.T) {
	// Test with a realistic project endpoint that might be used with Microsoft Graph
	handler := createTestTracesHandlerInternal()

	headers := map[string]string{
		"X-Microsoft-Foundry-API-Key": "ms-graph-integration-key-abc123",
	}
	c, _ := setupTestContext(headers)

	appConfig := &platform.ChatAgentConfigResponse{
		Type:        platform.AgentTypeFoundry,
		TenantID:    "test-tenant",
		ChatAgentID: "test-chat-agent",
		Settings: platform.AgentSettings{
			ProjectEndpoint: "https://contoso.api.azureml.ms/agents/openai/threads",
			APIVersion:      "2025-11-15-preview",
			AgentType:       "AGENT",
			AgentName:       "graph-enabled-agent",
		},
	}

	conversation := &platform.ConversationResponse{
		ID:                "conv-123",
		ExtConversationID: "thread_abc123xyz",
		TenantID:          "test-tenant",
		ChatAgentID:       "test-chat-agent",
	}

	config, err := handler.buildFoundryConfig(c, appConfig, conversation)

	require.NoError(t, err)
	assert.NotNil(t, config)
	assert.Equal(t, "thread_abc123xyz", config["ext_conversation_id"])
	assert.Equal(t, "https://contoso.api.azureml.ms/agents/openai/threads", config["project_endpoint"])
	assert.Equal(t, "2025-11-15-preview", config["api_version"])
	assert.Equal(t, "ms-graph-integration-key-abc123", config["api_token"])
}

// --- buildN8NConfig Tests ---

func TestBuildN8NConfig_ReturnsNotImplementedError(t *testing.T) {
	handler := createTestTracesHandlerInternal()

	c, _ := setupTestContext(nil)

	appConfig := &platform.ChatAgentConfigResponse{
		Type:        platform.AgentTypeN8N,
		TenantID:    "test-tenant",
		ChatAgentID: "test-chat-agent",
		Settings: platform.AgentSettings{
			ChatURL:      "https://n8n.example.com/webhook/chat",
			WorkflowType: platform.N8NWorkflowTypeChatAgent,
		},
	}

	conversation := &platform.ConversationResponse{
		ID:          "conv-123",
		TenantID:    "test-tenant",
		ChatAgentID: "test-chat-agent",
	}

	config, err := handler.buildN8NConfig(c, appConfig, conversation)

	assert.Nil(t, config)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "N8N trace import not yet implemented")
}

func TestBuildN8NConfig_WithAPIKeyAuth(t *testing.T) {
	// Test that even with valid API key credentials, N8N trace import is not implemented
	handler := createTestTracesHandlerInternal()

	headers := map[string]string{
		"Authorization": "Bearer test-user-token",
	}
	c, _ := setupTestContext(headers)

	appConfig := &platform.ChatAgentConfigResponse{
		Type:        platform.AgentTypeN8N,
		TenantID:    "test-tenant",
		ChatAgentID: "test-chat-agent",
		Settings: platform.AgentSettings{
			ChatURL:      "https://n8n.example.com/webhook/chat",
			WorkflowType: platform.N8NWorkflowTypeChatAgent,
			APICredentials: &platform.Credentials{
				ID:   "cred-123",
				Type: platform.CredentialTypeN8NAPIKey,
				Name: "N8N API Key",
			},
		},
	}

	conversation := &platform.ConversationResponse{
		ID:          "conv-123",
		TenantID:    "test-tenant",
		ChatAgentID: "test-chat-agent",
	}

	config, err := handler.buildN8NConfig(c, appConfig, conversation)

	assert.Nil(t, config)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "N8N trace import not yet implemented")
}

func TestBuildN8NConfig_WithBasicAuth(t *testing.T) {
	// Test that even with basic auth credentials, N8N trace import is not implemented
	handler := createTestTracesHandlerInternal()

	c, _ := setupTestContext(nil)

	appConfig := &platform.ChatAgentConfigResponse{
		Type:        platform.AgentTypeN8N,
		TenantID:    "test-tenant",
		ChatAgentID: "test-chat-agent",
		Settings: platform.AgentSettings{
			ChatURL:      "https://n8n.example.com/webhook/chat",
			WorkflowType: platform.N8NWorkflowTypeChatAgent,
			APICredentials: &platform.Credentials{
				ID:   "cred-456",
				Type: platform.CredentialTypeN8NBasicAuth,
				Name: "N8N Basic Auth",
			},
		},
	}

	conversation := &platform.ConversationResponse{
		ID:                "conv-123",
		ExtConversationID: "n8n-session-789",
		TenantID:          "test-tenant",
		ChatAgentID:       "test-chat-agent",
	}

	config, err := handler.buildN8NConfig(c, appConfig, conversation)

	assert.Nil(t, config)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "N8N trace import not yet implemented")
}

// --- buildBackendConfig Tests ---

func TestBuildBackendConfig_FoundryType(t *testing.T) {
	handler := createTestTracesHandlerInternal()

	headers := map[string]string{
		"X-Microsoft-Foundry-API-Key": "test-foundry-api-key",
	}
	c, _ := setupTestContext(headers)

	appConfig := &platform.ChatAgentConfigResponse{
		Type:        platform.AgentTypeFoundry,
		TenantID:    "test-tenant",
		ChatAgentID: "test-chat-agent",
		Settings: platform.AgentSettings{
			ProjectEndpoint: "https://my-project.api.azureml.ms",
			APIVersion:      "2025-01-01-preview",
		},
	}

	conversation := &platform.ConversationResponse{
		ID:                "conv-123",
		ExtConversationID: "ext-conv-456",
		TenantID:          "test-tenant",
		ChatAgentID:       "test-chat-agent",
	}

	config, err := handler.buildBackendConfig(c, appConfig, conversation)

	require.NoError(t, err)
	assert.NotNil(t, config)
	assert.Equal(t, "ext-conv-456", config["ext_conversation_id"])
	assert.Equal(t, "https://my-project.api.azureml.ms", config["project_endpoint"])
}

func TestBuildBackendConfig_N8NType(t *testing.T) {
	handler := createTestTracesHandlerInternal()

	c, _ := setupTestContext(nil)

	appConfig := &platform.ChatAgentConfigResponse{
		Type:        platform.AgentTypeN8N,
		TenantID:    "test-tenant",
		ChatAgentID: "test-chat-agent",
		Settings: platform.AgentSettings{
			ChatURL: "https://n8n.example.com/webhook/chat",
		},
	}

	conversation := &platform.ConversationResponse{
		ID:          "conv-123",
		TenantID:    "test-tenant",
		ChatAgentID: "test-chat-agent",
	}

	config, err := handler.buildBackendConfig(c, appConfig, conversation)

	assert.Nil(t, config)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "N8N trace import not yet implemented")
}

func TestBuildBackendConfig_UnsupportedType(t *testing.T) {
	handler := createTestTracesHandlerInternal()

	c, _ := setupTestContext(nil)

	appConfig := &platform.ChatAgentConfigResponse{
		Type:        "", // Empty/unsupported type
		TenantID:    "test-tenant",
		ChatAgentID: "test-chat-agent",
	}

	conversation := &platform.ConversationResponse{
		ID:          "conv-123",
		TenantID:    "test-tenant",
		ChatAgentID: "test-chat-agent",
	}

	config, err := handler.buildBackendConfig(c, appConfig, conversation)

	require.NoError(t, err)
	assert.NotNil(t, config)
	assert.Empty(t, config, "Unsupported types should return empty config")
}

func TestBuildBackendConfig_CustomAgentType(t *testing.T) {
	handler := createTestTracesHandlerInternal()

	c, _ := setupTestContext(nil)

	appConfig := &platform.ChatAgentConfigResponse{
		Type:        platform.AgentType("custom"), // Custom/unknown type
		TenantID:    "test-tenant",
		ChatAgentID: "test-chat-agent",
	}

	conversation := &platform.ConversationResponse{
		ID:          "conv-123",
		TenantID:    "test-tenant",
		ChatAgentID: "test-chat-agent",
	}

	config, err := handler.buildBackendConfig(c, appConfig, conversation)

	require.NoError(t, err)
	assert.NotNil(t, config)
	assert.Empty(t, config, "Custom types should return empty config")
}
