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
		{"n8n-nodes-base.postgres", "database"},
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
		// Communication & Messaging
		{"n8n-nodes-base.slack", "messaging"},
		{"n8n-nodes-base.discord", "messaging"},
		{"n8n-nodes-base.telegram", "messaging"},
		{"n8n-nodes-base.microsoftTeams", "messaging"},
		{"n8n-nodes-base.gmail", "messaging"},
		{"n8n-nodes-base.sendEmail", "messaging"},
		// Productivity & Spreadsheets
		{"n8n-nodes-base.googleSheets", "spreadsheet"},
		{"n8n-nodes-base.airtable", "spreadsheet"},
		{"n8n-nodes-base.notion", "spreadsheet"},
		{"n8n-nodes-base.microsoftExcel", "spreadsheet"},
		// Project Management
		{"n8n-nodes-base.jira", "project_mgmt"},
		{"n8n-nodes-base.trello", "project_mgmt"},
		{"n8n-nodes-base.linear", "project_mgmt"},
		// CRM & Sales
		{"n8n-nodes-base.hubSpot", "crm"},
		{"n8n-nodes-base.salesforce", "crm"},
		// File Storage & Cloud
		{"n8n-nodes-base.googleDrive", "storage"},
		{"n8n-nodes-base.s3", "storage"},
		{"n8n-nodes-base.azureStorage", "storage"},
		// Developer Tools
		{"n8n-nodes-base.github", "devops"},
		{"n8n-nodes-base.gitlab", "devops"},
		// Additional Database
		{"n8n-nodes-base.microsoftSql", "database"},
		{"n8n-nodes-base.elasticsearch", "database"},
		{"n8n-nodes-base.azureCosmosDb", "database"},
		// Message Queues
		{"n8n-nodes-base.kafka", "queue"},
		{"n8n-nodes-base.rabbitMq", "queue"},
		// E-Commerce & Payments
		{"n8n-nodes-base.stripe", "payment"},
		{"n8n-nodes-base.shopify", "payment"},
		// Customer Support
		{"n8n-nodes-base.zendesk", "support"},
		{"n8n-nodes-base.serviceNow", "support"},
		// Marketing
		{"n8n-nodes-base.mailchimp", "marketing"},
		{"n8n-nodes-base.sendGrid", "marketing"},
		// Data Transformation
		{"n8n-nodes-base.dateTime", "data_transform"},
		{"n8n-nodes-base.crypto", "data_transform"},
		{"n8n-nodes-base.xml", "data_transform"},
		{"n8n-nodes-base.html", "data_transform"},
		{"n8n-nodes-base.sort", "data_transform"},
		{"n8n-nodes-base.splitOut", "data_transform"},
		{"n8n-nodes-base.summarize", "data_transform"},
		{"n8n-nodes-base.itemLists", "data_transform"},
		{"n8n-nodes-base.rssFeedRead", "data_transform"},
		// File I/O
		{"n8n-nodes-base.extractFromFile", "file_io"},
		{"n8n-nodes-base.spreadsheetFile", "file_io"},
		{"n8n-nodes-base.ftp", "file_io"},
		// Workflow Data
		{"n8n-nodes-base.dataTable", "data_store"},
		// Utility
		{"n8n-nodes-base.graphQl", "tool"},
		{"n8n-nodes-base.ssh", "tool"},
		// Core/Workflow
		{"n8n-nodes-base.stopAndError", "core"},
		{"n8n-nodes-base.executeWorkflowTrigger", "core"},
		// Microsoft & Azure
		{"n8n-nodes-base.microsoftGraphSecurity", "security"},
		{"n8n-nodes-base.microsoftToDo", "productivity"},
		{"n8n-nodes-base.microsoftEntra", "identity"},
		// Trigger variants use existing trigger catch-all
		{"n8n-nodes-base.slackTrigger", "trigger"},
		{"n8n-nodes-base.kafkaTrigger", "trigger"},
		{"n8n-nodes-base.githubTrigger", "trigger"},
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
