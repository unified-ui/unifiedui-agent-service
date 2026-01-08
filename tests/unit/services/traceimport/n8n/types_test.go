// Package n8n_test contains unit tests for the N8N trace import types.
package n8n_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	n8n "github.com/unifiedui/agent-service/internal/services/traceimport/n8n"
)

func TestExtractConfig_ValidConfig(t *testing.T) {
	backendConfig := map[string]interface{}{
		"execution_id": "exec-123",
		"session_id":   "session-456",
		"base_url":     "http://localhost:5678",
		"api_key":      "test-api-key",
		"workflow_id":  "workflow-789",
	}

	config, ok := n8n.ExtractConfig(backendConfig)

	require.True(t, ok)
	assert.Equal(t, "exec-123", config.ExecutionID)
	assert.Equal(t, "session-456", config.SessionID)
	assert.Equal(t, "http://localhost:5678", config.BaseURL)
	assert.Equal(t, "test-api-key", config.APIKey)
	assert.Equal(t, "workflow-789", config.WorkflowID)
}

func TestExtractConfig_MinimalConfigWithExecutionID(t *testing.T) {
	backendConfig := map[string]interface{}{
		"execution_id": "exec-123",
		"base_url":     "http://localhost:5678",
		"api_key":      "test-api-key",
	}

	config, ok := n8n.ExtractConfig(backendConfig)

	require.True(t, ok)
	assert.Equal(t, "exec-123", config.ExecutionID)
	assert.Empty(t, config.SessionID)
	assert.Equal(t, "http://localhost:5678", config.BaseURL)
	assert.Equal(t, "test-api-key", config.APIKey)
}

func TestExtractConfig_MinimalConfigWithSessionID(t *testing.T) {
	backendConfig := map[string]interface{}{
		"session_id": "session-456",
		"base_url":   "http://localhost:5678",
		"api_key":    "test-api-key",
	}

	config, ok := n8n.ExtractConfig(backendConfig)

	require.True(t, ok)
	assert.Empty(t, config.ExecutionID)
	assert.Equal(t, "session-456", config.SessionID)
}

func TestExtractConfig_NilConfig(t *testing.T) {
	config, ok := n8n.ExtractConfig(nil)

	assert.False(t, ok)
	assert.Nil(t, config)
}

func TestExtractConfig_MissingBaseURL(t *testing.T) {
	backendConfig := map[string]interface{}{
		"execution_id": "exec-123",
		"api_key":      "test-api-key",
	}

	config, ok := n8n.ExtractConfig(backendConfig)

	assert.False(t, ok)
	assert.Nil(t, config)
}

func TestExtractConfig_MissingAPIKey(t *testing.T) {
	backendConfig := map[string]interface{}{
		"execution_id": "exec-123",
		"base_url":     "http://localhost:5678",
	}

	config, ok := n8n.ExtractConfig(backendConfig)

	assert.False(t, ok)
	assert.Nil(t, config)
}

func TestExtractConfig_MissingExecutionIDAndSessionID(t *testing.T) {
	backendConfig := map[string]interface{}{
		"base_url": "http://localhost:5678",
		"api_key":  "test-api-key",
	}

	config, ok := n8n.ExtractConfig(backendConfig)

	assert.False(t, ok)
	assert.Nil(t, config)
}

func TestGetNodeCategory(t *testing.T) {
	testCases := []struct {
		nodeType string
		expected string
	}{
		{"n8n-nodes-base.manualTrigger", "trigger"},
		{"@n8n/n8n-nodes-langchain.chatTrigger", "trigger"},
		{"@n8n/n8n-nodes-langchain.agent", "agent"},
		{"@n8n/n8n-nodes-langchain.lmChatAzureOpenAi", "llm"},
		{"n8n-nodes-base.httpRequest", "tool"},
		{"n8n-nodes-base.postgres", "tool"},
		{"n8n-nodes-base.code", "code"},
		{"n8n-nodes-base.function", "code"},
		{"n8n-nodes-base.switch", "conditional"},
		{"n8n-nodes-base.if", "conditional"},
		{"n8n-nodes-base.form", "form"},
		{"n8n-nodes-base.formTrigger", "form"},
		{"@n8n/n8n-nodes-langchain.memoryBufferWindow", "memory"},
		{"@n8n/n8n-nodes-langchain.toolWorkflow", "tool"},
		{"some-unknown-type", "custom"},
	}

	for _, tc := range testCases {
		t.Run(tc.nodeType, func(t *testing.T) {
			result := n8n.GetNodeCategory(tc.nodeType)
			assert.Equal(t, tc.expected, result)
		})
	}
}
