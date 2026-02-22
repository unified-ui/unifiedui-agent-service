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
		// Triggers
		{"n8n-nodes-base.manualTrigger", "trigger"},
		{"@n8n/n8n-nodes-langchain.chatTrigger", "trigger"},
		{"n8n-nodes-base.webhook", "trigger"},
		{"n8n-nodes-base.scheduleTrigger", "trigger"},
		// Forms
		{"n8n-nodes-base.form", "form"},
		{"n8n-nodes-base.formTrigger", "form"},
		// Agents
		{"@n8n/n8n-nodes-langchain.agent", "agent"},
		{"@n8n/n8n-nodes-langchain.information-extractor", "agent"},
		{"@n8n/n8n-nodes-langchain.text-classifier", "agent"},
		{"@n8n/n8n-nodes-langchain.sentimentAnalysis", "agent"},
		// Chains
		{"@n8n/n8n-nodes-langchain.chainLlm", "chain"},
		{"@n8n/n8n-nodes-langchain.chainRetrievalQa", "chain"},
		{"@n8n/n8n-nodes-langchain.chainSummarization", "chain"},
		// LLM
		{"@n8n/n8n-nodes-langchain.lmChatAzureOpenAi", "llm"},
		{"@n8n/n8n-nodes-langchain.lmChatOpenAi", "llm"},
		{"@n8n/n8n-nodes-langchain.lmChatAnthropic", "llm"},
		{"@n8n/n8n-nodes-langchain.lmChatGoogleGemini", "llm"},
		// Memory
		{"@n8n/n8n-nodes-langchain.memoryBufferWindow", "memory"},
		{"@n8n/n8n-nodes-langchain.memoryPostgresChat", "memory"},
		// Tools
		{"@n8n/n8n-nodes-langchain.toolWorkflow", "tool"},
		{"@n8n/n8n-nodes-langchain.toolCode", "tool"},
		{"@n8n/n8n-nodes-langchain.toolMcp", "tool"},
		{"n8n-nodes-base.httpRequest", "tool"},
		{"n8n-nodes-base.postgres", "tool"},
		{"n8n-nodes-base.executeWorkflow", "tool"},
		{"n8n-nodes-base.executeCommand", "tool"},
		// Vector stores
		{"@n8n/n8n-nodes-langchain.vectorStorePinecone", "vectorStore"},
		{"@n8n/n8n-nodes-langchain.vectorStorePgVector", "vectorStore"},
		// Embeddings
		{"@n8n/n8n-nodes-langchain.embeddingsOpenAi", "embedding"},
		{"@n8n/n8n-nodes-langchain.embeddingsAzureOpenAi", "embedding"},
		// Output parsers
		{"@n8n/n8n-nodes-langchain.outputParserStructured", "outputParser"},
		// Document loaders
		{"@n8n/n8n-nodes-langchain.documentDefaultDataLoader", "document"},
		// Text splitters
		{"@n8n/n8n-nodes-langchain.textSplitterRecursiveCharacterTextSplitter", "textSplitter"},
		// Retrievers
		{"@n8n/n8n-nodes-langchain.retrieverVectorStore", "retriever"},
		// Code
		{"n8n-nodes-base.code", "code"},
		{"n8n-nodes-base.function", "code"},
		{"@n8n/n8n-nodes-langchain.code", "code"},
		// Conditional
		{"n8n-nodes-base.switch", "conditional"},
		{"n8n-nodes-base.if", "conditional"},
		{"n8n-nodes-base.filter", "conditional"},
		{"n8n-nodes-base.merge", "conditional"},
		// Loop
		{"n8n-nodes-base.splitInBatches", "loop"},
		// Custom/default
		{"n8n-nodes-base.set", "custom"},
		{"n8n-nodes-base.noOp", "custom"},
		{"some-unknown-type", "custom"},
	}

	for _, tc := range testCases {
		t.Run(tc.nodeType, func(t *testing.T) {
			result := n8n.GetNodeCategory(tc.nodeType)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestGetNodeCategory_EmbeddingsNotLLM(t *testing.T) {
	assert.Equal(t, "embedding", n8n.GetNodeCategory("@n8n/n8n-nodes-langchain.embeddingsOpenAi"))
	assert.Equal(t, "llm", n8n.GetNodeCategory("@n8n/n8n-nodes-langchain.lmChatOpenAi"))
}

func TestNodeOutputData_GetOutputItems_AllKeys(t *testing.T) {
	t.Run("ai_outputParser", func(t *testing.T) {
		data := n8n.NodeOutputData{
			AIOutputParser: [][]n8n.NodeOutputItem{{{JSON: map[string]interface{}{"parsed": true}}}},
		}
		items := data.GetOutputItems()
		require.NotNil(t, items)
		assert.Equal(t, true, items[0][0].JSON["parsed"])
	})

	t.Run("ai_document", func(t *testing.T) {
		data := n8n.NodeOutputData{
			AIDocument: [][]n8n.NodeOutputItem{{{JSON: map[string]interface{}{"pageContent": "doc text"}}}},
		}
		items := data.GetOutputItems()
		require.NotNil(t, items)
		assert.Equal(t, "doc text", items[0][0].JSON["pageContent"])
	})

	t.Run("ai_textSplitter", func(t *testing.T) {
		data := n8n.NodeOutputData{
			AITextSplitter: [][]n8n.NodeOutputItem{{{JSON: map[string]interface{}{"chunks": 5}}}},
		}
		items := data.GetOutputItems()
		require.NotNil(t, items)
	})
}
