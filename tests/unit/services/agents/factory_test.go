// Package agents_test provides tests for the agent factory.
package agents_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/unifiedui/agent-service/internal/services/agents"
	"github.com/unifiedui/agent-service/internal/services/platform"
)

// TestNewFactory tests factory creation.
func TestNewFactory(t *testing.T) {
	factory := agents.NewFactory()
	assert.NotNil(t, factory)
}

// TestFactory_CreateClients_N8N tests creating N8N clients.
func TestFactory_CreateClients_N8N(t *testing.T) {
	factory := agents.NewFactory()
	config := &platform.AgentConfig{
		Type:          platform.AgentTypeN8N,
		TenantID:      "tenant-123",
		ApplicationID: "app-456",
		Settings: platform.AgentSettings{
			ChatURL:      "https://n8n.example.com/webhook/chat",
			WorkflowType: platform.N8NWorkflowTypeChatAgent,
			APIVersion:   "v1",
		},
	}

	clients, err := factory.CreateClients(config)
	require.NoError(t, err)
	require.NotNil(t, clients)
	assert.NotNil(t, clients.WorkflowClient)
	defer clients.Close()
}

// TestFactory_CreateClients_Foundry_RequiresToken tests that Foundry requires token via separate method.
func TestFactory_CreateClients_Foundry_RequiresToken(t *testing.T) {
	factory := agents.NewFactory()
	config := &platform.AgentConfig{
		Type:          platform.AgentTypeFoundry,
		TenantID:      "tenant-123",
		ApplicationID: "app-456",
		Settings: platform.AgentSettings{
			ProjectEndpoint: "https://test.services.ai.azure.com/api/projects/test-project",
			AgentName:       "TestAgent",
		},
	}

	// CreateClients should fail for Foundry - need to use CreateFoundryClients
	clients, err := factory.CreateClients(config)
	require.Error(t, err)
	assert.Nil(t, clients)
	assert.Contains(t, err.Error(), "use CreateFoundryClients instead")
}

// TestFactory_CreateFoundryClients_Success tests creating Foundry clients.
func TestFactory_CreateFoundryClients_Success(t *testing.T) {
	factory := agents.NewFactory()
	config := &platform.AgentConfig{
		Type:          platform.AgentTypeFoundry,
		TenantID:      "tenant-123",
		ApplicationID: "app-456",
		Settings: platform.AgentSettings{
			APIVersion:      "2025-11-15-preview",
			AgentType:       "AGENT",
			ProjectEndpoint: "https://test.services.ai.azure.com/api/projects/test-project",
			AgentName:       "TestAgent",
		},
	}

	clients, err := factory.CreateFoundryClients(config, "test-api-token")
	require.NoError(t, err)
	require.NotNil(t, clients)
	assert.NotNil(t, clients.WorkflowClient)
	assert.Nil(t, clients.APIClient) // Foundry doesn't have separate API client
	defer clients.Close()
}

// TestFactory_CreateFoundryClients_MissingToken tests error when token is missing.
func TestFactory_CreateFoundryClients_MissingToken(t *testing.T) {
	factory := agents.NewFactory()
	config := &platform.AgentConfig{
		Type:          platform.AgentTypeFoundry,
		TenantID:      "tenant-123",
		ApplicationID: "app-456",
		Settings: platform.AgentSettings{
			ProjectEndpoint: "https://test.services.ai.azure.com/api/projects/test-project",
			AgentName:       "TestAgent",
		},
	}

	clients, err := factory.CreateFoundryClients(config, "")
	require.Error(t, err)
	assert.Nil(t, clients)
	assert.Contains(t, err.Error(), "API token is required")
}

// TestFactory_CreateFoundryClients_NilConfig tests error when config is nil.
func TestFactory_CreateFoundryClients_NilConfig(t *testing.T) {
	factory := agents.NewFactory()

	clients, err := factory.CreateFoundryClients(nil, "test-token")
	require.Error(t, err)
	assert.Nil(t, clients)
	assert.Contains(t, err.Error(), "config is required")
}

// TestFactory_CreateClients_NilConfig tests error when config is nil.
func TestFactory_CreateClients_NilConfig(t *testing.T) {
	factory := agents.NewFactory()

	clients, err := factory.CreateClients(nil)
	require.Error(t, err)
	assert.Nil(t, clients)
	assert.Contains(t, err.Error(), "config is required")
}

// TestFactory_CreateClients_UnsupportedType tests error for unsupported agent type.
func TestFactory_CreateClients_UnsupportedType(t *testing.T) {
	factory := agents.NewFactory()
	config := &platform.AgentConfig{
		Type: platform.AgentType("UNKNOWN"),
	}

	clients, err := factory.CreateClients(config)
	require.Error(t, err)
	assert.Nil(t, clients)
	assert.Contains(t, err.Error(), "unsupported agent type")
}

// TestFactory_CreateClients_Copilot_NotImplemented tests that Copilot is not yet implemented.
func TestFactory_CreateClients_Copilot_NotImplemented(t *testing.T) {
	factory := agents.NewFactory()
	config := &platform.AgentConfig{
		Type: platform.AgentTypeCopilot,
	}

	clients, err := factory.CreateClients(config)
	require.Error(t, err)
	assert.Nil(t, clients)
	assert.Contains(t, err.Error(), "not yet implemented")
}
