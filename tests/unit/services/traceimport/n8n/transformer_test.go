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
		if nodes[i].Name == "When clicking 'Test workflow'" {
			triggerNode = &nodes[i]
		} else if nodes[i].Name == "HTTP Request" {
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
		},
	}

	nodes := transformer.TransformExecution(execution, "test-user")

	require.Len(t, nodes, 3)

	// Find nodes by name
	var triggerNode, agentNode, llmNode *models.TraceNode
	for i := range nodes {
		switch nodes[i].Name {
		case "When chat message received":
			triggerNode = &nodes[i]
		case "AI Agent":
			agentNode = &nodes[i]
		case "Azure OpenAI Chat Model":
			llmNode = &nodes[i]
		}
	}

	require.NotNil(t, triggerNode)
	require.NotNil(t, agentNode)
	require.NotNil(t, llmNode)

	// Check trigger node has the user input
	assert.Equal(t, models.NodeTypeWorkflow, triggerNode.Type)
	assert.Equal(t, models.NodeStatusCompleted, triggerNode.Status)
	assert.NotNil(t, triggerNode.Data)
	assert.NotNil(t, triggerNode.Data.Input)
	assert.Equal(t, "Hello, what can you do?", triggerNode.Data.Input.Text)

	// Check agent node
	assert.Equal(t, models.NodeTypeAgent, agentNode.Type)
	assert.Equal(t, models.NodeStatusCompleted, agentNode.Status)
	assert.NotNil(t, agentNode.Data)
	assert.NotNil(t, agentNode.Data.Output)
	assert.Contains(t, agentNode.Data.Output.Text, "I can help you with various tasks!")

	// Check token usage in metadata
	require.NotNil(t, agentNode.Metadata)
	tokenUsage, ok := agentNode.Metadata["token_usage"].(map[string]interface{})
	require.True(t, ok)
	assert.Equal(t, 150, tokenUsage["total_tokens"])

	// Check LLM node
	assert.Equal(t, models.NodeTypeLLM, llmNode.Type)
	assert.Equal(t, models.NodeStatusCompleted, llmNode.Status)
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
		{"n8n-nodes-base.manualTrigger", models.NodeTypeWorkflow},
		{"@n8n/n8n-nodes-langchain.chatTrigger", models.NodeTypeWorkflow},
		{"@n8n/n8n-nodes-langchain.agent", models.NodeTypeAgent},
		{"@n8n/n8n-nodes-langchain.lmChatAzureOpenAi", models.NodeTypeLLM},
		{"@n8n/n8n-nodes-langchain.lmChatOpenAi", models.NodeTypeLLM},
		{"n8n-nodes-base.httpRequest", models.NodeTypeHTTP},
		{"n8n-nodes-base.code", models.NodeTypeCode},
		{"n8n-nodes-base.function", models.NodeTypeCode},
		{"n8n-nodes-base.switch", models.NodeTypeConditional},
		{"n8n-nodes-base.if", models.NodeTypeConditional},
		{"n8n-nodes-base.postgres", models.NodeTypeTool},
		{"n8n-nodes-base.mongoDb", models.NodeTypeTool},
		{"@n8n/n8n-nodes-langchain.toolWorkflow", models.NodeTypeTool},
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
