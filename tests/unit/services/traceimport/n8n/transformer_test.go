package n8n_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/unifiedui/agent-service/internal/domain/models"
	n8n "github.com/unifiedui/agent-service/internal/services/traceimport/n8n"
)

func TestN8NTransformer_TransformExecution_EmptyResponse(t *testing.T) {
	transformer := n8n.NewTransformer()

	// Test nil execution
	nodes := transformer.TransformExecution(nil, "test-user")
	assert.Empty(t, nodes)

	// Test execution with no data
	nodes = transformer.TransformExecution(&n8n.ExecutionResponse{}, "test-user")
	assert.Empty(t, nodes)

	// Test execution with empty result data
	nodes = transformer.TransformExecution(&n8n.ExecutionResponse{
		Data: &n8n.ExecutionData{
			ResultData: &n8n.ResultData{},
		},
	}, "test-user")
	assert.Empty(t, nodes)
}

func TestN8NTransformer_TransformExecution_SimpleWorkflow(t *testing.T) {
	transformer := n8n.NewTransformer()

	execution := &n8n.ExecutionResponse{
		ID:         "1648",
		Status:     n8n.ExecutionStatusSuccess,
		Mode:       "manual",
		WorkflowID: "JI29YxoB4n0D4mcU",
		StartedAt:  "2025-01-03T09:35:53.844Z",
		StoppedAt:  "2025-01-03T09:35:57.000Z",
		Data: &n8n.ExecutionData{
			ResultData: &n8n.ResultData{
				RunData: map[string][]n8n.NodeExecution{
					"When clicking 'Test workflow'": {
						{
							StartTime:       1735900553844,
							ExecutionTime:   1,
							ExecutionStatus: n8n.NodeExecutionStatusSuccess,
							Data: n8n.NodeOutputData{
								Main: [][]n8n.NodeOutputItem{
									{
										{
											JSON: map[string]interface{}{},
										},
									},
								},
							},
						},
					},
					"HTTP Request": {
						{
							StartTime:       1735900553845,
							ExecutionTime:   1500,
							ExecutionStatus: n8n.NodeExecutionStatusSuccess,
							Source: []n8n.NodeExecutionSource{
								{PreviousNode: "When clicking 'Test workflow'"},
							},
							Data: n8n.NodeOutputData{
								Main: [][]n8n.NodeOutputItem{
									{
										{
											JSON: map[string]interface{}{
												"response": "test data",
											},
										},
									},
								},
							},
						},
					},
				},
				LastNodeExecuted: "HTTP Request",
			},
		},
		WorkflowData: &n8n.WorkflowData{
			ID:   "JI29YxoB4n0D4mcU",
			Name: "Test Workflow",
			Nodes: []n8n.WorkflowNode{
				{
					ID:   "node-1",
					Name: "When clicking 'Test workflow'",
					Type: "n8n-nodes-base.manualTrigger",
				},
				{
					ID:   "node-2",
					Name: "HTTP Request",
					Type: "n8n-nodes-base.httpRequest",
				},
			},
		},
	}

	nodes := transformer.TransformExecution(execution, "test-user")

	require.Len(t, nodes, 2)

	// Find nodes by name (they should be sorted by start time)
	var triggerNode, httpNode *models.TraceNode
	for i := range nodes {
		switch nodes[i].Name {
		case "When clicking 'Test workflow'":
			triggerNode = &nodes[i]
		case "HTTP Request":
			httpNode = &nodes[i]
		}
	}

	require.NotNil(t, triggerNode, "Trigger node should exist")
	require.NotNil(t, httpNode, "HTTP node should exist")

	// Check trigger node
	assert.Equal(t, models.NodeTypeWorkflow, triggerNode.Type)
	assert.Equal(t, models.NodeStatusCompleted, triggerNode.Status)
	assert.NotNil(t, triggerNode.StartAt)

	// Check HTTP request node
	assert.Equal(t, models.NodeTypeHTTP, httpNode.Type)
	assert.Equal(t, models.NodeStatusCompleted, httpNode.Status)
	assert.NotNil(t, httpNode.Data)
	assert.NotNil(t, httpNode.Data.Output)
}

func TestN8NTransformer_TransformExecution_AIAgentWorkflow(t *testing.T) {
	transformer := n8n.NewTransformer()

	execution := &n8n.ExecutionResponse{
		ID:         "119",
		Status:     n8n.ExecutionStatusSuccess,
		Mode:       "webhook",
		WorkflowID: "workflow-ai-agent",
		Data: &n8n.ExecutionData{
			ResultData: &n8n.ResultData{
				RunData: map[string][]n8n.NodeExecution{
					"When chat message received": {
						{
							StartTime:       1735900000000,
							ExecutionTime:   10,
							ExecutionStatus: n8n.NodeExecutionStatusSuccess,
							Data: n8n.NodeOutputData{
								Main: [][]n8n.NodeOutputItem{
									{
										{
											JSON: map[string]interface{}{
												"chatInput": "Hello, what can you do?",
												"sessionId": "dc812e23-58c9-4cae-bf11-833925982810",
											},
										},
									},
								},
							},
						},
					},
					"AI Agent": {
						{
							StartTime:       1735900000010,
							ExecutionTime:   5000,
							ExecutionStatus: n8n.NodeExecutionStatusSuccess,
							Source: []n8n.NodeExecutionSource{
								{PreviousNode: "When chat message received"},
							},
							Data: n8n.NodeOutputData{
								Main: [][]n8n.NodeOutputItem{
									{
										{
											JSON: map[string]interface{}{
												"output": "I can help you with various tasks!",
											},
										},
									},
								},
							},
							Metadata: &n8n.NodeExecutionMetadata{
								TokenUsage: &n8n.TokenUsage{
									PromptTokens:     100,
									CompletionTokens: 50,
									TotalTokens:      150,
								},
							},
						},
					},
					"Azure OpenAI Chat Model": {
						{
							StartTime:       1735900000010,
							ExecutionTime:   4500,
							ExecutionStatus: n8n.NodeExecutionStatusSuccess,
							Metadata: &n8n.NodeExecutionMetadata{
								TokenUsage: &n8n.TokenUsage{
									PromptTokens:     100,
									CompletionTokens: 50,
									TotalTokens:      150,
								},
							},
						},
					},
				},
				LastNodeExecuted: "AI Agent",
			},
		},
		WorkflowData: &n8n.WorkflowData{
			ID:   "workflow-ai-agent",
			Name: "AI Agent Workflow",
			Nodes: []n8n.WorkflowNode{
				{
					ID:   "chat-trigger",
					Name: "When chat message received",
					Type: "@n8n/n8n-nodes-langchain.chatTrigger",
				},
				{
					ID:   "agent",
					Name: "AI Agent",
					Type: "@n8n/n8n-nodes-langchain.agent",
				},
				{
					ID:   "llm",
					Name: "Azure OpenAI Chat Model",
					Type: "@n8n/n8n-nodes-langchain.lmChatAzureOpenAi",
				},
			},
			Connections: map[string]interface{}{
				"When chat message received": map[string]interface{}{
					"main": []interface{}{
						[]interface{}{
							map[string]interface{}{"node": "AI Agent", "type": "main", "index": 0},
						},
					},
				},
				"Azure OpenAI Chat Model": map[string]interface{}{
					"ai_languageModel": []interface{}{
						[]interface{}{
							map[string]interface{}{"node": "AI Agent", "type": "ai_languageModel", "index": 0},
						},
					},
				},
			},
		},
	}

	nodes := transformer.TransformExecution(execution, "test-user")

	require.Len(t, nodes, 2)

	var triggerNode, agentNode *models.TraceNode
	for i := range nodes {
		switch nodes[i].Name {
		case "When chat message received":
			triggerNode = &nodes[i]
		case "AI Agent":
			agentNode = &nodes[i]
		}
	}

	require.NotNil(t, triggerNode)
	require.NotNil(t, agentNode)

	assert.Equal(t, models.NodeTypeWorkflow, triggerNode.Type)
	assert.Equal(t, models.NodeStatusCompleted, triggerNode.Status)
	assert.NotNil(t, triggerNode.Data)
	assert.NotNil(t, triggerNode.Data.Input)
	assert.Equal(t, "Hello, what can you do?", triggerNode.Data.Input.Text)

	assert.Equal(t, models.NodeTypeAgent, agentNode.Type)
	assert.Equal(t, models.NodeStatusCompleted, agentNode.Status)
	assert.NotNil(t, agentNode.Data)
	assert.NotNil(t, agentNode.Data.Output)
	assert.Contains(t, agentNode.Data.Output.Text, "I can help you with various tasks!")

	require.NotNil(t, agentNode.Metadata)
	tokenUsage, ok := agentNode.Metadata["token_usage"].(map[string]interface{})
	require.True(t, ok)
	assert.Equal(t, 150, tokenUsage["total_tokens"])

	require.Len(t, agentNode.Nodes, 1)
	llmNode := agentNode.Nodes[0]
	assert.Equal(t, "Azure OpenAI Chat Model", llmNode.Name)
	assert.Equal(t, models.NodeTypeLLM, llmNode.Type)
	assert.Equal(t, models.NodeStatusCompleted, llmNode.Status)
	assert.Equal(t, "ai_languageModel", llmNode.Metadata["connection_type"])
}

func TestN8NTransformer_TransformExecution_ErrorExecution(t *testing.T) {
	transformer := n8n.NewTransformer()

	execution := &n8n.ExecutionResponse{
		ID:         "1001",
		Status:     n8n.ExecutionStatusError,
		Mode:       "manual",
		WorkflowID: "error-workflow",
		Data: &n8n.ExecutionData{
			ResultData: &n8n.ResultData{
				RunData: map[string][]n8n.NodeExecution{
					"Manual Trigger": {
						{
							StartTime:       1735900000000,
							ExecutionTime:   5,
							ExecutionStatus: n8n.NodeExecutionStatusSuccess,
						},
					},
					"Failing Node": {
						{
							StartTime:       1735900000010,
							ExecutionTime:   100,
							ExecutionStatus: n8n.NodeExecutionStatusError,
							Error: &n8n.NodeExecutionError{
								Name:        "NodeOperationError",
								Message:     "Connection refused",
								Description: "Could not connect to the server",
							},
						},
					},
				},
				Error: &n8n.ExecutionError{
					Name:    "NodeOperationError",
					Message: "Connection refused",
				},
			},
		},
		WorkflowData: &n8n.WorkflowData{
			ID:   "error-workflow",
			Name: "Error Workflow",
			Nodes: []n8n.WorkflowNode{
				{
					ID:   "trigger",
					Name: "Manual Trigger",
					Type: "n8n-nodes-base.manualTrigger",
				},
				{
					ID:   "failing",
					Name: "Failing Node",
					Type: "n8n-nodes-base.httpRequest",
				},
			},
		},
	}

	nodes := transformer.TransformExecution(execution, "test-user")

	require.Len(t, nodes, 2)

	// Find failing node
	var failingNode *models.TraceNode
	for i := range nodes {
		if nodes[i].Name == "Failing Node" {
			failingNode = &nodes[i]
			break
		}
	}

	require.NotNil(t, failingNode)
	assert.Equal(t, models.NodeStatusFailed, failingNode.Status)
	assert.NotNil(t, failingNode.Metadata)
	errorInfo, ok := failingNode.Metadata["error"].(map[string]interface{})
	require.True(t, ok)
	assert.Equal(t, "NodeOperationError", errorInfo["name"])
	assert.Equal(t, "Connection refused", errorInfo["message"])
}

func TestN8NTransformer_MapNodeType(t *testing.T) {
	transformer := n8n.NewTransformer()

	testCases := []struct {
		n8nType      string
		expectedType models.NodeType
	}{
		// Triggers
		{"n8n-nodes-base.manualTrigger", models.NodeTypeWorkflow},
		{"@n8n/n8n-nodes-langchain.chatTrigger", models.NodeTypeWorkflow},
		{"n8n-nodes-base.webhook", models.NodeTypeWorkflow},
		{"n8n-nodes-base.scheduleTrigger", models.NodeTypeWorkflow},
		// Agents
		{"@n8n/n8n-nodes-langchain.agent", models.NodeTypeAgent},
		{"@n8n/n8n-nodes-langchain.information-extractor", models.NodeTypeAgent},
		{"@n8n/n8n-nodes-langchain.text-classifier", models.NodeTypeAgent},
		{"@n8n/n8n-nodes-langchain.sentimentAnalysis", models.NodeTypeAgent},
		// Chains
		{"@n8n/n8n-nodes-langchain.chainLlm", models.NodeTypeChain},
		{"@n8n/n8n-nodes-langchain.chainRetrievalQa", models.NodeTypeChain},
		{"@n8n/n8n-nodes-langchain.chainSummarization", models.NodeTypeChain},
		// LLM models
		{"@n8n/n8n-nodes-langchain.lmChatAzureOpenAi", models.NodeTypeLLM},
		{"@n8n/n8n-nodes-langchain.lmChatOpenAi", models.NodeTypeLLM},
		{"@n8n/n8n-nodes-langchain.lmChatAnthropic", models.NodeTypeLLM},
		{"@n8n/n8n-nodes-langchain.lmChatGoogleGemini", models.NodeTypeLLM},
		{"@n8n/n8n-nodes-langchain.lmChatGroq", models.NodeTypeLLM},
		{"@n8n/n8n-nodes-langchain.lmChatOllama", models.NodeTypeLLM},
		{"@n8n/n8n-nodes-langchain.lmChatDeepSeek", models.NodeTypeLLM},
		{"@n8n/n8n-nodes-langchain.lmChatMistralCloud", models.NodeTypeLLM},
		// Memory
		{"@n8n/n8n-nodes-langchain.memoryBufferWindow", models.NodeTypeMemory},
		{"@n8n/n8n-nodes-langchain.memoryPostgresChat", models.NodeTypeMemory},
		{"@n8n/n8n-nodes-langchain.memoryRedisChat", models.NodeTypeMemory},
		// Tools (langchain)
		{"@n8n/n8n-nodes-langchain.toolWorkflow", models.NodeTypeTool},
		{"@n8n/n8n-nodes-langchain.toolCode", models.NodeTypeTool},
		{"@n8n/n8n-nodes-langchain.toolMcp", models.NodeTypeTool},
		{"@n8n/n8n-nodes-langchain.toolThink", models.NodeTypeTool},
		// Vector stores
		{"@n8n/n8n-nodes-langchain.vectorStorePinecone", models.NodeTypeVectorStore},
		{"@n8n/n8n-nodes-langchain.vectorStorePgVector", models.NodeTypeVectorStore},
		{"@n8n/n8n-nodes-langchain.vectorStoreQdrant", models.NodeTypeVectorStore},
		// Embeddings
		{"@n8n/n8n-nodes-langchain.embeddingsOpenAi", models.NodeTypeEmbedding},
		{"@n8n/n8n-nodes-langchain.embeddingsAzureOpenAi", models.NodeTypeEmbedding},
		// Output parsers
		{"@n8n/n8n-nodes-langchain.outputParserStructured", models.NodeTypeOutputParser},
		{"@n8n/n8n-nodes-langchain.outputParserAutofixing", models.NodeTypeOutputParser},
		// Document loaders
		{"@n8n/n8n-nodes-langchain.documentDefaultDataLoader", models.NodeTypeDocument},
		// Text splitters
		{"@n8n/n8n-nodes-langchain.textSplitterRecursiveCharacterTextSplitter", models.NodeTypeTextSplitter},
		// Retrievers
		{"@n8n/n8n-nodes-langchain.retrieverVectorStore", models.NodeTypeRetriever},
		{"@n8n/n8n-nodes-langchain.retrieverMultiQuery", models.NodeTypeRetriever},
		// Core nodes
		{"n8n-nodes-base.httpRequest", models.NodeTypeHTTP},
		{"n8n-nodes-base.code", models.NodeTypeCode},
		{"@n8n/n8n-nodes-langchain.code", models.NodeTypeCode},
		{"n8n-nodes-base.function", models.NodeTypeCode},
		{"n8n-nodes-base.switch", models.NodeTypeConditional},
		{"n8n-nodes-base.if", models.NodeTypeConditional},
		{"n8n-nodes-base.filter", models.NodeTypeConditional},
		{"n8n-nodes-base.merge", models.NodeTypeWorkflow},
		{"n8n-nodes-base.executeWorkflow", models.NodeTypeWorkflow},
		{"n8n-nodes-base.respondToWebhook", models.NodeTypeWorkflow},
		{"n8n-nodes-base.splitInBatches", models.NodeTypeLoop},
		{"n8n-nodes-base.postgres", models.NodeTypeTool},
		{"n8n-nodes-base.mongoDb", models.NodeTypeTool},
		{"n8n-nodes-base.executeCommand", models.NodeTypeTool},
		{"n8n-nodes-base.readWriteFile", models.NodeTypeTool},
		{"n8n-nodes-base.set", models.NodeTypeCustom},
		{"n8n-nodes-base.noOp", models.NodeTypeCustom},
		{"n8n-nodes-base.html", models.NodeTypeCustom},
		{"unknown-node-type", models.NodeTypeCustom},
	}

	// Create a simple execution for each type
	for _, tc := range testCases {
		t.Run(tc.n8nType, func(t *testing.T) {
			execution := &n8n.ExecutionResponse{
				Data: &n8n.ExecutionData{
					ResultData: &n8n.ResultData{
						RunData: map[string][]n8n.NodeExecution{
							"TestNode": {
								{
									StartTime:       1735900000000,
									ExecutionTime:   10,
									ExecutionStatus: n8n.NodeExecutionStatusSuccess,
								},
							},
						},
					},
				},
				WorkflowData: &n8n.WorkflowData{
					Nodes: []n8n.WorkflowNode{
						{
							Name: "TestNode",
							Type: tc.n8nType,
						},
					},
				},
			}

			nodes := transformer.TransformExecution(execution, "test-user")
			require.Len(t, nodes, 1)
			assert.Equal(t, tc.expectedType, nodes[0].Type)
		})
	}
}

func TestN8NTransformer_ExtractSessionID(t *testing.T) {
	transformer := n8n.NewTransformer()

	t.Run("with session ID in chat trigger", func(t *testing.T) {
		execution := &n8n.ExecutionResponse{
			Data: &n8n.ExecutionData{
				ResultData: &n8n.ResultData{
					RunData: map[string][]n8n.NodeExecution{
						"When chat message received": {
							{
								StartTime:       1735900000000,
								ExecutionTime:   10,
								ExecutionStatus: n8n.NodeExecutionStatusSuccess,
								Data: n8n.NodeOutputData{
									Main: [][]n8n.NodeOutputItem{
										{
											{
												JSON: map[string]interface{}{
													"sessionId": "dc812e23-58c9-4cae-bf11-833925982810",
													"chatInput": "Hello",
												},
											},
										},
									},
								},
							},
						},
					},
				},
			},
		}

		sessionID := transformer.ExtractSessionID(execution)
		assert.Equal(t, "dc812e23-58c9-4cae-bf11-833925982810", sessionID)
	})

	t.Run("no session ID", func(t *testing.T) {
		execution := &n8n.ExecutionResponse{
			Data: &n8n.ExecutionData{
				ResultData: &n8n.ResultData{
					RunData: map[string][]n8n.NodeExecution{
						"HTTP Request": {
							{
								StartTime:       1735900000000,
								ExecutionTime:   100,
								ExecutionStatus: n8n.NodeExecutionStatusSuccess,
							},
						},
					},
				},
			},
		}

		sessionID := transformer.ExtractSessionID(execution)
		assert.Empty(t, sessionID)
	})

	t.Run("nil execution", func(t *testing.T) {
		sessionID := transformer.ExtractSessionID(nil)
		assert.Empty(t, sessionID)
	})
}

func TestN8NTransformer_ChronologicalOrder(t *testing.T) {
	transformer := n8n.NewTransformer()

	execution := &n8n.ExecutionResponse{
		Data: &n8n.ExecutionData{
			ResultData: &n8n.ResultData{
				RunData: map[string][]n8n.NodeExecution{
					"Node C": {
						{
							StartTime:       1735900000300,
							ExecutionTime:   10,
							ExecutionStatus: n8n.NodeExecutionStatusSuccess,
						},
					},
					"Node A": {
						{
							StartTime:       1735900000100,
							ExecutionTime:   10,
							ExecutionStatus: n8n.NodeExecutionStatusSuccess,
						},
					},
					"Node B": {
						{
							StartTime:       1735900000200,
							ExecutionTime:   10,
							ExecutionStatus: n8n.NodeExecutionStatusSuccess,
						},
					},
				},
			},
		},
		WorkflowData: &n8n.WorkflowData{
			Nodes: []n8n.WorkflowNode{
				{Name: "Node A", Type: "n8n-nodes-base.code"},
				{Name: "Node B", Type: "n8n-nodes-base.code"},
				{Name: "Node C", Type: "n8n-nodes-base.code"},
			},
		},
	}

	nodes := transformer.TransformExecution(execution, "test-user")

	require.Len(t, nodes, 3)

	// Nodes should be in chronological order
	assert.Equal(t, "Node A", nodes[0].Name)
	assert.Equal(t, "Node B", nodes[1].Name)
	assert.Equal(t, "Node C", nodes[2].Name)
}

func TestN8NTransformer_Duration(t *testing.T) {
	transformer := n8n.NewTransformer()

	execution := &n8n.ExecutionResponse{
		Data: &n8n.ExecutionData{
			ResultData: &n8n.ResultData{
				RunData: map[string][]n8n.NodeExecution{
					"Test Node": {
						{
							StartTime:       1735900000000,
							ExecutionTime:   2500, // 2500 ms
							ExecutionStatus: n8n.NodeExecutionStatusSuccess,
						},
					},
				},
			},
		},
		WorkflowData: &n8n.WorkflowData{
			Nodes: []n8n.WorkflowNode{
				{Name: "Test Node", Type: "n8n-nodes-base.code"},
			},
		},
	}

	nodes := transformer.TransformExecution(execution, "test-user")

	require.Len(t, nodes, 1)
	assert.InDelta(t, 2.5, nodes[0].Duration, 0.001) // Duration should be in seconds
}

func TestN8NTransformer_Transform_InterfaceWrapper(t *testing.T) {
	transformer := n8n.NewTransformer()

	execution := &n8n.ExecutionResponse{
		Data: &n8n.ExecutionData{
			ResultData: &n8n.ResultData{
				RunData: map[string][]n8n.NodeExecution{
					"Test": {
						{
							StartTime:       1735900000000,
							ExecutionTime:   10,
							ExecutionStatus: n8n.NodeExecutionStatusSuccess,
						},
					},
				},
			},
		},
	}

	// Test the Transform interface method
	nodes := transformer.Transform(execution, "test-user")
	assert.Len(t, nodes, 1)

	// Test with wrong type
	nodes = transformer.Transform("wrong type", "test-user")
	assert.Empty(t, nodes)
}

func TestN8NTransformer_MultipleSubNodes(t *testing.T) {
	transformer := n8n.NewTransformer()

	execution := &n8n.ExecutionResponse{
		ID:     "200",
		Status: n8n.ExecutionStatusSuccess,
		Data: &n8n.ExecutionData{
			ResultData: &n8n.ResultData{
				RunData: map[string][]n8n.NodeExecution{
					"Chat Trigger": {
						{
							StartTime:       1735900000000,
							ExecutionTime:   5,
							ExecutionStatus: n8n.NodeExecutionStatusSuccess,
						},
					},
					"AI Agent": {
						{
							StartTime:       1735900000010,
							ExecutionTime:   8000,
							ExecutionStatus: n8n.NodeExecutionStatusSuccess,
						},
					},
					"Azure OpenAI Chat Model": {
						{
							StartTime:       1735900000015,
							ExecutionTime:   6000,
							ExecutionStatus: n8n.NodeExecutionStatusSuccess,
						},
					},
					"Memory Buffer": {
						{
							StartTime:       1735900000012,
							ExecutionTime:   50,
							ExecutionStatus: n8n.NodeExecutionStatusSuccess,
						},
					},
					"Search Tool": {
						{
							StartTime:       1735900000020,
							ExecutionTime:   2000,
							ExecutionStatus: n8n.NodeExecutionStatusSuccess,
						},
					},
				},
			},
		},
		WorkflowData: &n8n.WorkflowData{
			Nodes: []n8n.WorkflowNode{
				{Name: "Chat Trigger", Type: "@n8n/n8n-nodes-langchain.chatTrigger"},
				{Name: "AI Agent", Type: "@n8n/n8n-nodes-langchain.agent"},
				{Name: "Azure OpenAI Chat Model", Type: "@n8n/n8n-nodes-langchain.lmChatAzureOpenAi"},
				{Name: "Memory Buffer", Type: "@n8n/n8n-nodes-langchain.memoryBufferWindow"},
				{Name: "Search Tool", Type: "@n8n/n8n-nodes-langchain.toolWorkflow"},
			},
			Connections: map[string]interface{}{
				"Chat Trigger": map[string]interface{}{
					"main": []interface{}{
						[]interface{}{
							map[string]interface{}{"node": "AI Agent", "type": "main", "index": 0},
						},
					},
				},
				"Azure OpenAI Chat Model": map[string]interface{}{
					"ai_languageModel": []interface{}{
						[]interface{}{
							map[string]interface{}{"node": "AI Agent", "type": "ai_languageModel", "index": 0},
						},
					},
				},
				"Memory Buffer": map[string]interface{}{
					"ai_memory": []interface{}{
						[]interface{}{
							map[string]interface{}{"node": "AI Agent", "type": "ai_memory", "index": 0},
						},
					},
				},
				"Search Tool": map[string]interface{}{
					"ai_tool": []interface{}{
						[]interface{}{
							map[string]interface{}{"node": "AI Agent", "type": "ai_tool", "index": 0},
						},
					},
				},
			},
		},
	}

	nodes := transformer.TransformExecution(execution, "test-user")

	require.Len(t, nodes, 2)
	assert.Equal(t, "Chat Trigger", nodes[0].Name)
	assert.Equal(t, "AI Agent", nodes[1].Name)

	require.Len(t, nodes[1].Nodes, 3)
	assert.Equal(t, "Memory Buffer", nodes[1].Nodes[0].Name)
	assert.Equal(t, "ai_memory", nodes[1].Nodes[0].Metadata["connection_type"])
	assert.Equal(t, "Azure OpenAI Chat Model", nodes[1].Nodes[1].Name)
	assert.Equal(t, "ai_languageModel", nodes[1].Nodes[1].Metadata["connection_type"])
	assert.Equal(t, "Search Tool", nodes[1].Nodes[2].Name)
	assert.Equal(t, "ai_tool", nodes[1].Nodes[2].Metadata["connection_type"])
}

func TestN8NTransformer_NoConnectionsFlatOutput(t *testing.T) {
	transformer := n8n.NewTransformer()

	execution := &n8n.ExecutionResponse{
		Data: &n8n.ExecutionData{
			ResultData: &n8n.ResultData{
				RunData: map[string][]n8n.NodeExecution{
					"Trigger": {
						{
							StartTime:       1735900000000,
							ExecutionTime:   5,
							ExecutionStatus: n8n.NodeExecutionStatusSuccess,
						},
					},
					"Agent": {
						{
							StartTime:       1735900000010,
							ExecutionTime:   1000,
							ExecutionStatus: n8n.NodeExecutionStatusSuccess,
						},
					},
					"LLM": {
						{
							StartTime:       1735900000015,
							ExecutionTime:   800,
							ExecutionStatus: n8n.NodeExecutionStatusSuccess,
						},
					},
				},
			},
		},
		WorkflowData: &n8n.WorkflowData{
			Nodes: []n8n.WorkflowNode{
				{Name: "Trigger", Type: "n8n-nodes-base.manualTrigger"},
				{Name: "Agent", Type: "@n8n/n8n-nodes-langchain.agent"},
				{Name: "LLM", Type: "@n8n/n8n-nodes-langchain.lmChatOpenAi"},
			},
		},
	}

	nodes := transformer.TransformExecution(execution, "test-user")

	require.Len(t, nodes, 3)
	assert.Equal(t, "Trigger", nodes[0].Name)
	assert.Equal(t, "Agent", nodes[1].Name)
	assert.Equal(t, "LLM", nodes[2].Name)

	for _, node := range nodes {
		assert.Empty(t, node.Nodes)
	}
}

func TestN8NTransformer_BranchingWorkflowStaysFlat(t *testing.T) {
	transformer := n8n.NewTransformer()

	execution := &n8n.ExecutionResponse{
		Data: &n8n.ExecutionData{
			ResultData: &n8n.ResultData{
				RunData: map[string][]n8n.NodeExecution{
					"Trigger": {
						{
							StartTime:       1735900000000,
							ExecutionTime:   5,
							ExecutionStatus: n8n.NodeExecutionStatusSuccess,
						},
					},
					"Switch": {
						{
							StartTime:       1735900000010,
							ExecutionTime:   10,
							ExecutionStatus: n8n.NodeExecutionStatusSuccess,
						},
					},
					"HTTP Request": {
						{
							StartTime:       1735900000020,
							ExecutionTime:   500,
							ExecutionStatus: n8n.NodeExecutionStatusSuccess,
						},
					},
					"Code": {
						{
							StartTime:       1735900000030,
							ExecutionTime:   100,
							ExecutionStatus: n8n.NodeExecutionStatusSuccess,
						},
					},
				},
			},
		},
		WorkflowData: &n8n.WorkflowData{
			Nodes: []n8n.WorkflowNode{
				{Name: "Trigger", Type: "n8n-nodes-base.manualTrigger"},
				{Name: "Switch", Type: "n8n-nodes-base.switch"},
				{Name: "HTTP Request", Type: "n8n-nodes-base.httpRequest"},
				{Name: "Code", Type: "n8n-nodes-base.code"},
			},
			Connections: map[string]interface{}{
				"Trigger": map[string]interface{}{
					"main": []interface{}{
						[]interface{}{
							map[string]interface{}{"node": "Switch", "type": "main", "index": 0},
						},
					},
				},
				"Switch": map[string]interface{}{
					"main": []interface{}{
						[]interface{}{
							map[string]interface{}{"node": "HTTP Request", "type": "main", "index": 0},
						},
						[]interface{}{
							map[string]interface{}{"node": "Code", "type": "main", "index": 0},
						},
					},
				},
			},
		},
	}

	nodes := transformer.TransformExecution(execution, "test-user")

	require.Len(t, nodes, 4)
	assert.Equal(t, "Trigger", nodes[0].Name)
	assert.Equal(t, "Switch", nodes[1].Name)
	assert.Equal(t, "HTTP Request", nodes[2].Name)
	assert.Equal(t, "Code", nodes[3].Name)

	for _, node := range nodes {
		assert.Empty(t, node.Nodes)
	}
}

func TestN8NTransformer_SubNodeNotInRunDataIgnored(t *testing.T) {
	transformer := n8n.NewTransformer()

	execution := &n8n.ExecutionResponse{
		Data: &n8n.ExecutionData{
			ResultData: &n8n.ResultData{
				RunData: map[string][]n8n.NodeExecution{
					"AI Agent": {
						{
							StartTime:       1735900000000,
							ExecutionTime:   5000,
							ExecutionStatus: n8n.NodeExecutionStatusSuccess,
						},
					},
				},
			},
		},
		WorkflowData: &n8n.WorkflowData{
			Nodes: []n8n.WorkflowNode{
				{Name: "AI Agent", Type: "@n8n/n8n-nodes-langchain.agent"},
				{Name: "Missing LLM", Type: "@n8n/n8n-nodes-langchain.lmChatOpenAi"},
			},
			Connections: map[string]interface{}{
				"Missing LLM": map[string]interface{}{
					"ai_languageModel": []interface{}{
						[]interface{}{
							map[string]interface{}{"node": "AI Agent", "type": "ai_languageModel", "index": 0},
						},
					},
				},
			},
		},
	}

	nodes := transformer.TransformExecution(execution, "test-user")

	require.Len(t, nodes, 1)
	assert.Equal(t, "AI Agent", nodes[0].Name)
	assert.Empty(t, nodes[0].Nodes)
}

func TestN8NTransformer_LLMSubNodeOutputExtraction(t *testing.T) {
	transformer := n8n.NewTransformer()

	execution := &n8n.ExecutionResponse{
		ID:     "300",
		Status: n8n.ExecutionStatusSuccess,
		Data: &n8n.ExecutionData{
			ResultData: &n8n.ResultData{
				RunData: map[string][]n8n.NodeExecution{
					"Chat Trigger": {
						{
							StartTime:       1735900000000,
							ExecutionTime:   5,
							ExecutionStatus: n8n.NodeExecutionStatusSuccess,
							Data: n8n.NodeOutputData{
								Main: [][]n8n.NodeOutputItem{
									{{JSON: map[string]interface{}{"chatInput": "Hello", "sessionId": "sess-1"}}},
								},
							},
						},
					},
					"AI Agent": {
						{
							StartTime:       1735900000010,
							ExecutionTime:   5000,
							ExecutionStatus: n8n.NodeExecutionStatusSuccess,
							Data: n8n.NodeOutputData{
								Main: [][]n8n.NodeOutputItem{
									{{JSON: map[string]interface{}{"output": "Agent response"}}},
								},
							},
							Metadata: &n8n.NodeExecutionMetadata{
								TokenUsage: &n8n.TokenUsage{
									PromptTokens: 100, CompletionTokens: 50, TotalTokens: 150,
								},
							},
						},
					},
					"OpenAI Chat Model": {
						{
							StartTime:       1735900000015,
							ExecutionTime:   4500,
							ExecutionStatus: n8n.NodeExecutionStatusSuccess,
							Data: n8n.NodeOutputData{
								AILanguageModel: [][]n8n.NodeOutputItem{
									{{JSON: map[string]interface{}{
										"response": map[string]interface{}{
											"generations": []interface{}{
												[]interface{}{
													map[string]interface{}{
														"text":           "LLM generated text",
														"generationInfo": map[string]interface{}{"finish_reason": "stop"},
													},
												},
											},
										},
										"tokenUsage": map[string]interface{}{
											"completionTokens": 50,
											"promptTokens":     100,
											"totalTokens":      150,
										},
									}}},
								},
							},
							InputOverride: map[string]interface{}{
								"ai_languageModel": []interface{}{
									[]interface{}{
										map[string]interface{}{
											"json": map[string]interface{}{
												"messages": []interface{}{"Human: Hello"},
											},
										},
									},
								},
							},
							Metadata: &n8n.NodeExecutionMetadata{
								SubRun: []interface{}{
									map[string]interface{}{"node": "OpenAI Chat Model", "runIndex": 0},
								},
							},
						},
					},
				},
			},
		},
		WorkflowData: &n8n.WorkflowData{
			Nodes: []n8n.WorkflowNode{
				{Name: "Chat Trigger", Type: "@n8n/n8n-nodes-langchain.chatTrigger"},
				{Name: "AI Agent", Type: "@n8n/n8n-nodes-langchain.agent"},
				{Name: "OpenAI Chat Model", Type: "@n8n/n8n-nodes-langchain.lmChatOpenAi"},
			},
			Connections: map[string]interface{}{
				"Chat Trigger": map[string]interface{}{
					"main": []interface{}{
						[]interface{}{map[string]interface{}{"node": "AI Agent", "type": "main", "index": 0}},
					},
				},
				"OpenAI Chat Model": map[string]interface{}{
					"ai_languageModel": []interface{}{
						[]interface{}{map[string]interface{}{"node": "AI Agent", "type": "ai_languageModel", "index": 0}},
					},
				},
			},
		},
	}

	nodes := transformer.TransformExecution(execution, "test-user")

	require.Len(t, nodes, 2)
	assert.Equal(t, "Chat Trigger", nodes[0].Name)
	assert.Equal(t, "AI Agent", nodes[1].Name)

	// AI Agent should have output text
	require.NotNil(t, nodes[1].Data)
	require.NotNil(t, nodes[1].Data.Output)
	assert.Equal(t, "Agent response", nodes[1].Data.Output.Text)

	// LLM should be a child of AI Agent
	require.Len(t, nodes[1].Nodes, 1)
	llmNode := nodes[1].Nodes[0]
	assert.Equal(t, "OpenAI Chat Model", llmNode.Name)
	assert.Equal(t, models.NodeTypeLLM, llmNode.Type)
	assert.Equal(t, "ai_languageModel", llmNode.Metadata["connection_type"])

	// LLM should have extracted output text from generations
	require.NotNil(t, llmNode.Data)
	require.NotNil(t, llmNode.Data.Output)
	assert.Equal(t, "LLM generated text", llmNode.Data.Output.Text)

	// LLM should have input override captured
	require.NotNil(t, llmNode.Data.Input)
	assert.NotNil(t, llmNode.Data.Input.ExtraData)
}

func TestN8NTransformer_ToolSubNodeOutputExtraction(t *testing.T) {
	transformer := n8n.NewTransformer()

	execution := &n8n.ExecutionResponse{
		ID:     "301",
		Status: n8n.ExecutionStatusSuccess,
		Data: &n8n.ExecutionData{
			ResultData: &n8n.ResultData{
				RunData: map[string][]n8n.NodeExecution{
					"AI Agent": {
						{
							StartTime:       1735900000000,
							ExecutionTime:   5000,
							ExecutionStatus: n8n.NodeExecutionStatusSuccess,
							Data: n8n.NodeOutputData{
								Main: [][]n8n.NodeOutputItem{
									{{JSON: map[string]interface{}{"output": "Agent used a tool"}}},
								},
							},
						},
					},
					"Search Tool": {
						{
							StartTime:       1735900000100,
							ExecutionTime:   1000,
							ExecutionStatus: n8n.NodeExecutionStatusSuccess,
							Data: n8n.NodeOutputData{
								AITool: [][]n8n.NodeOutputItem{
									{{JSON: map[string]interface{}{"response": "Search result text"}}},
								},
							},
							Metadata: &n8n.NodeExecutionMetadata{
								SubRun: []interface{}{
									map[string]interface{}{"node": "Search Tool", "runIndex": 0},
								},
							},
						},
					},
				},
			},
		},
		WorkflowData: &n8n.WorkflowData{
			Nodes: []n8n.WorkflowNode{
				{Name: "AI Agent", Type: "@n8n/n8n-nodes-langchain.agent"},
				{Name: "Search Tool", Type: "@n8n/n8n-nodes-langchain.toolSerpApi"},
			},
			Connections: map[string]interface{}{
				"Search Tool": map[string]interface{}{
					"ai_tool": []interface{}{
						[]interface{}{map[string]interface{}{"node": "AI Agent", "type": "ai_tool", "index": 0}},
					},
				},
			},
		},
	}

	nodes := transformer.TransformExecution(execution, "test-user")

	require.Len(t, nodes, 1)
	assert.Equal(t, "AI Agent", nodes[0].Name)

	// Tool should be a child of AI Agent
	require.Len(t, nodes[0].Nodes, 1)
	toolNode := nodes[0].Nodes[0]
	assert.Equal(t, "Search Tool", toolNode.Name)
	assert.Equal(t, models.NodeTypeTool, toolNode.Type)
	assert.Equal(t, "ai_tool", toolNode.Metadata["connection_type"])

	// Tool should have extracted response text
	require.NotNil(t, toolNode.Data)
	require.NotNil(t, toolNode.Data.Output)
	assert.Equal(t, "Search result text", toolNode.Data.Output.Text)
}

func TestN8NTransformer_MemorySubNodeOutputExtraction(t *testing.T) {
	transformer := n8n.NewTransformer()

	execution := &n8n.ExecutionResponse{
		ID:     "302",
		Status: n8n.ExecutionStatusSuccess,
		Data: &n8n.ExecutionData{
			ResultData: &n8n.ResultData{
				RunData: map[string][]n8n.NodeExecution{
					"AI Agent": {
						{
							StartTime:       1735900000000,
							ExecutionTime:   5000,
							ExecutionStatus: n8n.NodeExecutionStatusSuccess,
						},
					},
					"Memory Buffer": {
						{
							StartTime:       1735900000010,
							ExecutionTime:   50,
							ExecutionStatus: n8n.NodeExecutionStatusSuccess,
							Data: n8n.NodeOutputData{
								AIMemory: [][]n8n.NodeOutputItem{
									{{JSON: map[string]interface{}{
										"action": "load",
										"chatHistory": []interface{}{
											map[string]interface{}{"type": "human", "data": map[string]interface{}{"content": "Hello"}},
										},
									}}},
								},
							},
						},
						{
							StartTime:       1735900005000,
							ExecutionTime:   30,
							ExecutionStatus: n8n.NodeExecutionStatusSuccess,
							Data: n8n.NodeOutputData{
								AIMemory: [][]n8n.NodeOutputItem{
									{{JSON: map[string]interface{}{"action": "save"}}},
								},
							},
						},
					},
				},
			},
		},
		WorkflowData: &n8n.WorkflowData{
			Nodes: []n8n.WorkflowNode{
				{Name: "AI Agent", Type: "@n8n/n8n-nodes-langchain.agent"},
				{Name: "Memory Buffer", Type: "@n8n/n8n-nodes-langchain.memoryBufferWindow"},
			},
			Connections: map[string]interface{}{
				"Memory Buffer": map[string]interface{}{
					"ai_memory": []interface{}{
						[]interface{}{map[string]interface{}{"node": "AI Agent", "type": "ai_memory", "index": 0}},
					},
				},
			},
		},
	}

	nodes := transformer.TransformExecution(execution, "test-user")

	require.Len(t, nodes, 1)
	assert.Equal(t, "AI Agent", nodes[0].Name)

	// Memory should be child nodes (2 runs: load + save)
	require.Len(t, nodes[0].Nodes, 2)
	memLoadNode := nodes[0].Nodes[0]
	assert.Equal(t, "Memory Buffer", memLoadNode.Name)
	assert.Equal(t, models.NodeTypeMemory, memLoadNode.Type)
	assert.Equal(t, "ai_memory", memLoadNode.Metadata["connection_type"])

	// Memory load should have output data with action
	require.NotNil(t, memLoadNode.Data)
	require.NotNil(t, memLoadNode.Data.Output)
	assert.NotNil(t, memLoadNode.Data.Output.ExtraData)
	assert.Equal(t, "load", memLoadNode.Data.Output.ExtraData["action"])

	// Second memory run (save)
	memSaveNode := nodes[0].Nodes[1]
	assert.Equal(t, "Memory Buffer", memSaveNode.Name)
	require.NotNil(t, memSaveNode.Data)
	require.NotNil(t, memSaveNode.Data.Output)
	assert.Equal(t, "save", memSaveNode.Data.Output.ExtraData["action"])
}

func TestN8NTransformer_ChainNodeTypes(t *testing.T) {
	transformer := n8n.NewTransformer()

	chainTypes := []struct {
		nodeType     string
		expectedType models.NodeType
	}{
		{"@n8n/n8n-nodes-langchain.chainLlm", models.NodeTypeChain},
		{"@n8n/n8n-nodes-langchain.chainRetrievalQa", models.NodeTypeChain},
		{"@n8n/n8n-nodes-langchain.chainSummarization", models.NodeTypeChain},
	}

	for _, tc := range chainTypes {
		t.Run(tc.nodeType, func(t *testing.T) {
			execution := &n8n.ExecutionResponse{
				Data: &n8n.ExecutionData{
					ResultData: &n8n.ResultData{
						RunData: map[string][]n8n.NodeExecution{
							"Chain Node": {
								{
									StartTime: 1735900000000, ExecutionTime: 1000,
									ExecutionStatus: n8n.NodeExecutionStatusSuccess,
									Data: n8n.NodeOutputData{
										Main: [][]n8n.NodeOutputItem{
											{{JSON: map[string]interface{}{
												"response": map[string]interface{}{"text": "Chain output"},
											}}},
										},
									},
								},
							},
						},
					},
				},
				WorkflowData: &n8n.WorkflowData{
					Nodes: []n8n.WorkflowNode{
						{Name: "Chain Node", Type: tc.nodeType},
					},
				},
			}

			nodes := transformer.TransformExecution(execution, "test-user")
			require.Len(t, nodes, 1)
			assert.Equal(t, tc.expectedType, nodes[0].Type)
		})
	}
}

func TestNodeOutputData_GetOutputItems(t *testing.T) {
	t.Run("returns main when present", func(t *testing.T) {
		data := n8n.NodeOutputData{
			Main: [][]n8n.NodeOutputItem{{{JSON: map[string]interface{}{"key": "val"}}}},
		}
		items := data.GetOutputItems()
		require.Len(t, items, 1)
		assert.Equal(t, "val", items[0][0].JSON["key"])
	})

	t.Run("returns ai_languageModel when main is empty", func(t *testing.T) {
		data := n8n.NodeOutputData{
			AILanguageModel: [][]n8n.NodeOutputItem{{{JSON: map[string]interface{}{"llm": true}}}},
		}
		items := data.GetOutputItems()
		require.Len(t, items, 1)
		assert.Equal(t, true, items[0][0].JSON["llm"])
	})

	t.Run("returns ai_tool when main is empty", func(t *testing.T) {
		data := n8n.NodeOutputData{
			AITool: [][]n8n.NodeOutputItem{{{JSON: map[string]interface{}{"response": "tool output"}}}},
		}
		items := data.GetOutputItems()
		require.Len(t, items, 1)
		assert.Equal(t, "tool output", items[0][0].JSON["response"])
	})

	t.Run("returns ai_memory when main is empty", func(t *testing.T) {
		data := n8n.NodeOutputData{
			AIMemory: [][]n8n.NodeOutputItem{{{JSON: map[string]interface{}{"action": "load"}}}},
		}
		items := data.GetOutputItems()
		require.Len(t, items, 1)
		assert.Equal(t, "load", items[0][0].JSON["action"])
	})

	t.Run("returns ai_retriever when main is empty", func(t *testing.T) {
		data := n8n.NodeOutputData{
			AIRetriever: [][]n8n.NodeOutputItem{{{JSON: map[string]interface{}{"docs": "retrieved"}}}},
		}
		items := data.GetOutputItems()
		require.Len(t, items, 1)
	})

	t.Run("returns ai_vectorStore when main is empty", func(t *testing.T) {
		data := n8n.NodeOutputData{
			AIVectorStore: [][]n8n.NodeOutputItem{{{JSON: map[string]interface{}{"vs": true}}}},
		}
		items := data.GetOutputItems()
		require.Len(t, items, 1)
	})

	t.Run("returns ai_embedding when main is empty", func(t *testing.T) {
		data := n8n.NodeOutputData{
			AIEmbedding: [][]n8n.NodeOutputItem{{{JSON: map[string]interface{}{"emb": true}}}},
		}
		items := data.GetOutputItems()
		require.Len(t, items, 1)
	})

	t.Run("returns nil when all empty", func(t *testing.T) {
		data := n8n.NodeOutputData{}
		items := data.GetOutputItems()
		assert.Nil(t, items)
	})

	t.Run("main takes priority over sub-node keys", func(t *testing.T) {
		data := n8n.NodeOutputData{
			Main:            [][]n8n.NodeOutputItem{{{JSON: map[string]interface{}{"from": "main"}}}},
			AILanguageModel: [][]n8n.NodeOutputItem{{{JSON: map[string]interface{}{"from": "llm"}}}},
		}
		items := data.GetOutputItems()
		require.Len(t, items, 1)
		assert.Equal(t, "main", items[0][0].JSON["from"])
	})
}

func TestN8NTransformer_StructuredOutputExtraction(t *testing.T) {
	transformer := n8n.NewTransformer()

	execution := &n8n.ExecutionResponse{
		Data: &n8n.ExecutionData{
			ResultData: &n8n.ResultData{
				RunData: map[string][]n8n.NodeExecution{
					"Info Extractor": {
						{
							StartTime: 1735900000000, ExecutionTime: 2000,
							ExecutionStatus: n8n.NodeExecutionStatusSuccess,
							Data: n8n.NodeOutputData{
								Main: [][]n8n.NodeOutputItem{
									{{JSON: map[string]interface{}{
										"output": map[string]interface{}{
											"name":  "John Doe",
											"email": "john@example.com",
										},
									}}},
								},
							},
						},
					},
				},
			},
		},
		WorkflowData: &n8n.WorkflowData{
			Nodes: []n8n.WorkflowNode{
				{Name: "Info Extractor", Type: "@n8n/n8n-nodes-langchain.information-extractor"},
			},
		},
	}

	nodes := transformer.TransformExecution(execution, "test-user")

	require.Len(t, nodes, 1)
	assert.Equal(t, models.NodeTypeAgent, nodes[0].Type)
	require.NotNil(t, nodes[0].Data)
	require.NotNil(t, nodes[0].Data.Output)
	// Structured output goes to ExtraData
	outputMap, ok := nodes[0].Data.Output.ExtraData["output"].(map[string]interface{})
	require.True(t, ok)
	assert.Equal(t, "John Doe", outputMap["name"])
}
