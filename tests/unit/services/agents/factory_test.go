package agents_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/unifiedui/agent-service/internal/config"
	"github.com/unifiedui/agent-service/internal/services/agents"
	"github.com/unifiedui/agent-service/internal/services/platform"
)

type mockWorkflowClient struct {
	closeErr error
}

func (m *mockWorkflowClient) Invoke(_ context.Context, _ *agents.InvokeRequest) (*agents.InvokeResponse, error) {
	return nil, nil
}

func (m *mockWorkflowClient) InvokeStream(_ context.Context, _ *agents.InvokeRequest) (<-chan *agents.StreamChunk, error) {
	return nil, nil
}

func (m *mockWorkflowClient) InvokeStreamReader(_ context.Context, _ *agents.InvokeRequest) (agents.StreamReader, error) {
	return nil, nil
}

func (m *mockWorkflowClient) Close() error {
	return m.closeErr
}

type mockAPIClient struct {
	closeErr error
}

func (m *mockAPIClient) GetExecution(_ context.Context, _ string) (*agents.ExecutionInfo, error) {
	return nil, nil
}

func (m *mockAPIClient) GetExecutionsBySession(_ context.Context, _ string) ([]*agents.ExecutionInfo, error) {
	return nil, nil
}

func (m *mockAPIClient) Close() error {
	return m.closeErr
}

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
	clients.Close()
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
	clients.Close()
}

func TestCreateClients_N8N_WithCredentials(t *testing.T) {
	f := agents.NewFactory()
	cfg := &platform.AgentConfig{
		Type: platform.AgentTypeN8N,
		Settings: platform.AgentSettings{
			WorkflowType: "N8N_CHAT_AGENT_WORKFLOW",
			ChatURL:      "http://n8n.local/webhook/chat",
			APIVersion:   "v1",
			ChatCredentials: &platform.Credentials{
				ID:   "cred-1",
				Type: "basic_auth",
				Secret: map[string]interface{}{
					"username": "user",
					"password": "pass",
				},
			},
		},
	}
	clients, err := f.CreateClients(cfg)
	require.NoError(t, err)
	assert.NotNil(t, clients)
	assert.NotNil(t, clients.WorkflowClient)
	assert.NotNil(t, clients.APIClient)
	clients.Close()
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

func TestNewFactoryWithReact(t *testing.T) {
	reactCfg := config.ReactServiceConfig{
		URL:     "http://react.local:8086",
		Timeout: 30 * time.Second,
	}
	f := agents.NewFactoryWithReact(reactCfg, "test-service-key")
	assert.NotNil(t, f)
}

func TestCreateClients_ReActAgent_NoReactFactory(t *testing.T) {
	f := agents.NewFactory()
	cfg := &platform.AgentConfig{
		Type: platform.AgentTypeReactAgent,
	}
	_, err := f.CreateClients(cfg)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "ReACT service not configured")
}

func TestCreateClients_ReActAgent_Success(t *testing.T) {
	reactCfg := config.ReactServiceConfig{
		URL:     "http://react.local:8086",
		Timeout: 30 * time.Second,
	}
	f := agents.NewFactoryWithReact(reactCfg, "test-key")
	cfg := &platform.AgentConfig{
		Type:        platform.AgentTypeReactAgent,
		TenantID:    "tenant-1",
		ChatAgentID: "agent-1",
		Settings: platform.AgentSettings{
			ReActAgentID: "react-agent-1",
			SystemPrompt: "You are a test agent.",
		},
	}
	clients, err := f.CreateClients(cfg)
	require.NoError(t, err)
	assert.NotNil(t, clients)
	assert.NotNil(t, clients.WorkflowClient)
	clients.Close()
}

func TestCreateClients_ReActAgent_MissingURL(t *testing.T) {
	reactCfg := config.ReactServiceConfig{
		URL: "",
	}
	f := agents.NewFactoryWithReact(reactCfg, "test-key")
	cfg := &platform.AgentConfig{
		Type: platform.AgentTypeReactAgent,
	}
	_, err := f.CreateClients(cfg)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to create ReACT workflow client")
}

func TestAgentClients_Close_WorkflowClientError(t *testing.T) {
	clients := &agents.AgentClients{
		WorkflowClient: &mockWorkflowClient{closeErr: fmt.Errorf("workflow close error")},
	}
	err := clients.Close()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "workflow close error")
}

func TestAgentClients_Close_APIClientError(t *testing.T) {
	clients := &agents.AgentClients{
		APIClient: &mockAPIClient{closeErr: fmt.Errorf("api close error")},
	}
	err := clients.Close()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "api close error")
}

func TestAgentClients_Close_BothClients(t *testing.T) {
	clients := &agents.AgentClients{
		WorkflowClient: &mockWorkflowClient{closeErr: nil},
		APIClient:      &mockAPIClient{closeErr: nil},
	}
	err := clients.Close()
	assert.NoError(t, err)
}

func TestAgentClients_Close_BothClientsError(t *testing.T) {
	clients := &agents.AgentClients{
		WorkflowClient: &mockWorkflowClient{closeErr: fmt.Errorf("workflow error")},
		APIClient:      &mockAPIClient{closeErr: fmt.Errorf("api error")},
	}
	err := clients.Close()
	assert.Error(t, err)
}
