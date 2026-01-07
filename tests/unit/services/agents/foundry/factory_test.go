// Package foundry_test provides tests for the Microsoft Foundry agent factory.
package foundry_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/unifiedui/agent-service/internal/services/agents/foundry"
	"github.com/unifiedui/agent-service/internal/services/platform"
)

// TestNewFromConfig_Success tests successful factory creation from config.
func TestNewFromConfig_Success(t *testing.T) {
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

	client, err := foundry.NewFromConfig(config, "test-api-token")
	require.NoError(t, err)
	require.NotNil(t, client)
	defer client.Close()
}

// TestNewFromConfig_NilConfig tests error when config is nil.
func TestNewFromConfig_NilConfig(t *testing.T) {
	client, err := foundry.NewFromConfig(nil, "test-token")
	require.Error(t, err)
	assert.Nil(t, client)
	assert.Contains(t, err.Error(), "config is required")
}

// TestNewFromConfig_MissingToken tests error when token is empty.
func TestNewFromConfig_MissingToken(t *testing.T) {
	config := &platform.AgentConfig{
		Type:          platform.AgentTypeFoundry,
		TenantID:      "tenant-123",
		ApplicationID: "app-456",
		Settings: platform.AgentSettings{
			ProjectEndpoint: "https://test.services.ai.azure.com/api/projects/test-project",
			AgentName:       "TestAgent",
		},
	}

	client, err := foundry.NewFromConfig(config, "")
	require.Error(t, err)
	assert.Nil(t, client)
	assert.Contains(t, err.Error(), "API token is required")
}

// TestNewFromConfig_MissingProjectEndpoint tests error when project endpoint is missing.
func TestNewFromConfig_MissingProjectEndpoint(t *testing.T) {
	config := &platform.AgentConfig{
		Type:          platform.AgentTypeFoundry,
		TenantID:      "tenant-123",
		ApplicationID: "app-456",
		Settings: platform.AgentSettings{
			AgentName: "TestAgent",
		},
	}

	client, err := foundry.NewFromConfig(config, "test-token")
	require.Error(t, err)
	assert.Nil(t, client)
	assert.Contains(t, err.Error(), "project endpoint is required")
}

// TestNewFromConfig_MissingAgentName tests error when agent name is missing.
func TestNewFromConfig_MissingAgentName(t *testing.T) {
	config := &platform.AgentConfig{
		Type:          platform.AgentTypeFoundry,
		TenantID:      "tenant-123",
		ApplicationID: "app-456",
		Settings: platform.AgentSettings{
			ProjectEndpoint: "https://test.services.ai.azure.com/api/projects/test-project",
		},
	}

	client, err := foundry.NewFromConfig(config, "test-token")
	require.Error(t, err)
	assert.Nil(t, client)
	assert.Contains(t, err.Error(), "agent name is required")
}

// TestNewFromConfig_DefaultAPIVersion tests that API version defaults when not set.
func TestNewFromConfig_DefaultAPIVersion(t *testing.T) {
	config := &platform.AgentConfig{
		Type:          platform.AgentTypeFoundry,
		TenantID:      "tenant-123",
		ApplicationID: "app-456",
		Settings: platform.AgentSettings{
			ProjectEndpoint: "https://test.services.ai.azure.com/api/projects/test-project",
			AgentName:       "TestAgent",
			// No APIVersion set - should default
		},
	}

	client, err := foundry.NewFromConfig(config, "test-token")
	require.NoError(t, err)
	require.NotNil(t, client)
	defer client.Close()
}
