package agents_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/unifiedui/agent-service/internal/services/agents"
	"github.com/unifiedui/agent-service/internal/services/platform"
)

func TestNewFactory(t *testing.T) {
	f := agents.NewFactory()
	assert.NotNil(t, f)
}

func TestCreateClients_NilConfig(t *testing.T) {
	f := agents.NewFactory()
	_, err := f.CreateClients(nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "config is required")
}

func TestCreateClients_N8N_Success(t *testing.T) {
	f := agents.NewFactory()
	config := &platform.AgentConfig{
		Type: platform.AgentTypeN8N,
		Settings: platform.AgentSettings{
			WorkflowType: "N8N_CHAT_AGENT_WORKFLOW",
			ChatURL:      "http://n8n.local/webhook/chat",
			APIVersion:   "v1",
		},
	}
	clients, err := f.CreateClients(config)
	require.NoError(t, err)
	assert.NotNil(t, clients)
	assert.NotNil(t, clients.WorkflowClient)
	assert.NotNil(t, clients.APIClient)
	defer clients.Close()
}

func TestCreateClients_Foundry_RequiresToken(t *testing.T) {
	f := agents.NewFactory()
	config := &platform.AgentConfig{
		Type: platform.AgentTypeFoundry,
	}
	_, err := f.CreateClients(config)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "CreateFoundryClients")
}

func TestCreateClients_Copilot_NotImplemented(t *testing.T) {
	f := agents.NewFactory()
	config := &platform.AgentConfig{
		Type: platform.AgentTypeCopilot,
	}
	_, err := f.CreateClients(config)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not yet implemented")
}

func TestCreateClients_Custom_NotImplemented(t *testing.T) {
	f := agents.NewFactory()
	config := &platform.AgentConfig{
		Type: platform.AgentTypeCustom,
	}
	_, err := f.CreateClients(config)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not yet implemented")
}

func TestCreateClients_UnsupportedType(t *testing.T) {
	f := agents.NewFactory()
	config := &platform.AgentConfig{
		Type: "UNKNOWN_TYPE",
	}
	_, err := f.CreateClients(config)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unsupported")
}

func TestCreateFoundryClients_NilConfig(t *testing.T) {
	f := agents.NewFactory()
	_, err := f.CreateFoundryClients(nil, "token")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "config is required")
}

func TestCreateFoundryClients_EmptyToken(t *testing.T) {
	f := agents.NewFactory()
	config := &platform.AgentConfig{
		Type: platform.AgentTypeFoundry,
	}
	_, err := f.CreateFoundryClients(config, "")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "API token")
}

func TestCreateFoundryClients_Success(t *testing.T) {
	f := agents.NewFactory()
	config := &platform.AgentConfig{
		Type: platform.AgentTypeFoundry,
		Settings: platform.AgentSettings{
			AgentType:       "AGENT",
			ProjectEndpoint: "https://foundry.example.com",
			AgentName:       "my-agent",
		},
	}
	clients, err := f.CreateFoundryClients(config, "api-token")
	require.NoError(t, err)
	assert.NotNil(t, clients)
	assert.NotNil(t, clients.WorkflowClient)
	defer clients.Close()
}

func TestCreateClients_N8N_MissingChatURL(t *testing.T) {
	f := agents.NewFactory()
	config := &platform.AgentConfig{
		Type: platform.AgentTypeN8N,
		Settings: platform.AgentSettings{
			WorkflowType: "N8N_CHAT_AGENT_WORKFLOW",
			ChatURL:      "",
			APIVersion:   "v1",
		},
	}
	_, err := f.CreateClients(config)
	assert.Error(t, err)
}

func TestCreateClients_N8N_UnsupportedWorkflowType(t *testing.T) {
	f := agents.NewFactory()
	config := &platform.AgentConfig{
		Type: platform.AgentTypeN8N,
		Settings: platform.AgentSettings{
			WorkflowType: "N8N_HUMAN_IN_THE_LOOP",
			ChatURL:      "http://n8n.local/webhook/chat",
			APIVersion:   "v1",
		},
	}
	_, err := f.CreateClients(config)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not yet implemented")
}

func TestCreateClients_N8N_UnsupportedAPIVersion(t *testing.T) {
	f := agents.NewFactory()
	config := &platform.AgentConfig{
		Type: platform.AgentTypeN8N,
		Settings: platform.AgentSettings{
			WorkflowType: "N8N_CHAT_AGENT_WORKFLOW",
			ChatURL:      "http://n8n.local/webhook/chat",
			APIVersion:   "v99",
		},
	}
	_, err := f.CreateClients(config)
	assert.Error(t, err)
}

func TestAgentClients_Close_NilClients(t *testing.T) {
	clients := &agents.AgentClients{}
	err := clients.Close()
	assert.NoError(t, err)
}
