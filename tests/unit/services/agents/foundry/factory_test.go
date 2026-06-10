// Package foundry_test provides tests for the Microsoft Foundry agent factory.
package foundry_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/unifiedui/agent-service/internal/services/agents/foundry"
	"github.com/unifiedui/agent-service/internal/services/platform"
)

// TestCreateWorkflowClient_Success tests successful factory creation from config.
func TestCreateWorkflowClient_Success(t *testing.T) {
	config := &platform.AgentConfig{
		Type:        platform.AgentTypeFoundry,
		TenantID:    "tenant-123",
		ChatAgentID: "app-456",
		Settings: platform.AgentSettings{
			APIVersion:      "2025-11-15-preview",
			AgentType:       "AGENT",
			ProjectEndpoint: "https://test.services.ai.azure.com/api/projects/test-project",
			AgentName:       "TestAgent",
		},
	}

	client, err := foundry.NewFactory().CreateWorkflowClient(config, "test-api-token")
	require.NoError(t, err)
	require.NotNil(t, client)
	client.Close()
}

// TestCreateWorkflowClient_NilConfig tests error when config is nil.
func TestCreateWorkflowClient_NilConfig(t *testing.T) {
	client, err := foundry.NewFactory().CreateWorkflowClient(nil, "test-token")
	require.Error(t, err)
	assert.Nil(t, client)
	assert.Contains(t, err.Error(), "config is required")
}

// TestCreateWorkflowClient_MissingToken tests error when token is empty.
func TestCreateWorkflowClient_MissingToken(t *testing.T) {
	config := &platform.AgentConfig{
		Type:        platform.AgentTypeFoundry,
		TenantID:    "tenant-123",
		ChatAgentID: "app-456",
		Settings: platform.AgentSettings{
			ProjectEndpoint: "https://test.services.ai.azure.com/api/projects/test-project",
			AgentName:       "TestAgent",
		},
	}

	client, err := foundry.NewFactory().CreateWorkflowClient(config, "")
	require.Error(t, err)
	assert.Nil(t, client)
	assert.Contains(t, err.Error(), "user token is required")
}

// TestCreateWorkflowClient_MissingProjectEndpoint tests error when project endpoint is missing.
func TestCreateWorkflowClient_MissingProjectEndpoint(t *testing.T) {
	config := &platform.AgentConfig{
		Type:        platform.AgentTypeFoundry,
		TenantID:    "tenant-123",
		ChatAgentID: "app-456",
		Settings: platform.AgentSettings{
			AgentName: "TestAgent",
		},
	}

	client, err := foundry.NewFactory().CreateWorkflowClient(config, "test-token")
	require.Error(t, err)
	assert.Nil(t, client)
	assert.Contains(t, err.Error(), "project endpoint is required")
}

// TestCreateWorkflowClient_MissingAgentName tests error when agent name is missing.
func TestCreateWorkflowClient_MissingAgentName(t *testing.T) {
	config := &platform.AgentConfig{
		Type:        platform.AgentTypeFoundry,
		TenantID:    "tenant-123",
		ChatAgentID: "app-456",
		Settings: platform.AgentSettings{
			ProjectEndpoint: "https://test.services.ai.azure.com/api/projects/test-project",
		},
	}

	client, err := foundry.NewFactory().CreateWorkflowClient(config, "test-token")
	require.Error(t, err)
	assert.Nil(t, client)
	assert.Contains(t, err.Error(), "agent name is required")
}

// TestCreateWorkflowClient_DefaultAPIVersion tests that API version defaults when not set.
func TestCreateWorkflowClient_DefaultAPIVersion(t *testing.T) {
	config := &platform.AgentConfig{
		Type:        platform.AgentTypeFoundry,
		TenantID:    "tenant-123",
		ChatAgentID: "app-456",
		Settings: platform.AgentSettings{
			ProjectEndpoint: "https://test.services.ai.azure.com/api/projects/test-project",
			AgentName:       "TestAgent",
			// No APIVersion set - should default
		},
	}

	client, err := foundry.NewFactory().CreateWorkflowClient(config, "test-token")
	require.NoError(t, err)
	require.NotNil(t, client)
	client.Close()
}
