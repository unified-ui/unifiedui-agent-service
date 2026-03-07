// Package foundry contains unit tests for the foundry trace import service.
package foundry

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/unifiedui/agent-service/internal/domain/models"
	"github.com/unifiedui/agent-service/internal/services/traceimport/foundry"
)

func TestFoundryTransformer_Transform_EmptyItems(t *testing.T) {
	transformer := foundry.NewTransformer()

	nodes := transformer.Transform([]foundry.ConversationItem{}, "test-user")

	assert.Empty(t, nodes)
}

func TestFoundryTransformer_Transform_SingleUserMessage(t *testing.T) {
	transformer := foundry.NewTransformer()

	items := []foundry.ConversationItem{
		{
			ID:           "msg_001",
			Type:         "message",
			Status:       "completed",
			Role:         "user",
			PartitionKey: "partition123",
			Content: []interface{}{
				map[string]interface{}{
					"type": "input_text",
					"text": "Hello, how are you?",
				},
			},
		},
	}

	nodes := transformer.Transform(items, "test-user")

	assert.Len(t, nodes, 1)
	assert.Equal(t, "User Message", nodes[0].Name)
	assert.Equal(t, models.NodeTypeLLM, nodes[0].Type)
	assert.Equal(t, models.NodeStatusCompleted, nodes[0].Status)
	assert.Equal(t, "msg_001", nodes[0].ReferenceID)
	assert.NotNil(t, nodes[0].Data)
	assert.NotNil(t, nodes[0].Data.Input)
	assert.Equal(t, "Hello, how are you?", nodes[0].Data.Input.Text)
}

func TestFoundryTransformer_Transform_SingleAssistantMessage(t *testing.T) {
	transformer := foundry.NewTransformer()

	items := []foundry.ConversationItem{
		{
			ID:           "msg_002",
			Type:         "message",
			Status:       "completed",
			Role:         "assistant",
			PartitionKey: "partition123",
			CreatedBy: map[string]interface{}{
				"response_id": "resp_001",
				"agent": map[string]interface{}{
					"type":    "agent_id",
					"name":    "TestAgent",
					"version": "1",
				},
			},
			Content: []interface{}{
				map[string]interface{}{
					"type": "output_text",
					"text": "I'm doing great, thanks!",
				},
			},
		},
	}

	nodes := transformer.Transform(items, "test-user")

	assert.Len(t, nodes, 1)
	assert.Equal(t, "Assistant Response", nodes[0].Name)
	assert.Equal(t, models.NodeTypeLLM, nodes[0].Type)
	assert.Equal(t, models.NodeStatusCompleted, nodes[0].Status)
	assert.NotNil(t, nodes[0].Data)
	assert.NotNil(t, nodes[0].Data.Output)
	assert.Equal(t, "I'm doing great, thanks!", nodes[0].Data.Output.Text)
	assert.NotNil(t, nodes[0].Metadata)
	assert.Equal(t, "resp_001", nodes[0].Metadata["response_id"])
}

func TestFoundryTransformer_Transform_WorkflowAction(t *testing.T) {
	transformer := foundry.NewTransformer()

	items := []foundry.ConversationItem{
		{
			ID:               "wfa_001",
			Type:             "workflow_action",
			Status:           "completed",
			Kind:             "SendActivity",
			ActionID:         "action-123",
			ParentActionID:   "trigger_wf",
			PreviousActionID: "action-100",
			CreatedBy: map[string]interface{}{
				"response_id": "wfresp_001",
				"agent": map[string]interface{}{
					"type":    "agent_id",
					"name":    "BasicWorkflow",
					"version": "2",
				},
			},
		},
	}

	nodes := transformer.Transform(items, "test-user")

	assert.Len(t, nodes, 1)
	assert.Equal(t, "Send Activity", nodes[0].Name)
	assert.Equal(t, models.NodeTypeWorkflow, nodes[0].Type)
	assert.Equal(t, models.NodeStatusCompleted, nodes[0].Status)
	assert.Equal(t, "wfa_001", nodes[0].ReferenceID)
	assert.NotNil(t, nodes[0].Metadata)
	assert.Equal(t, "action-123", nodes[0].Metadata["action_id"])
	assert.Equal(t, "trigger_wf", nodes[0].Metadata["parent_action_id"])
	assert.Equal(t, "SendActivity", nodes[0].Metadata["kind"])
}

func TestFoundryTransformer_Transform_MCPCallWithApproval(t *testing.T) {
	transformer := foundry.NewTransformer()
	approved := true

	items := []foundry.ConversationItem{
		{
			ID:           "mcpr_001",
			Type:         "mcp_approval_request",
			ServerLabel:  "MicrosoftWordFrontier",
			Name:         "WordCreateNewDocument",
			Arguments:    json.RawMessage(`{"fileName":"test.docx"}`),
			PartitionKey: "partition123",
			CreatedBy: map[string]interface{}{
				"response_id": "resp_001",
			},
		},
		{
			ID:                "mcpa_001",
			Type:              "mcp_approval_response",
			ApprovalRequestID: "mcpr_001",
			PartitionKey:      "partition123",
			Approve:           &approved,
		},
		{
			ID:                "mcp_001",
			Type:              "mcp_call",
			Status:            "completed",
			ApprovalRequestID: "mcpr_001",
			ServerLabel:       "MicrosoftWordFrontier",
			Name:              "WordCreateNewDocument",
			Arguments:         json.RawMessage(`{"fileName":"test.docx"}`),
			Output:            json.RawMessage(`{"result":"success"}`),
			PartitionKey:      "partition123",
			CreatedBy: map[string]interface{}{
				"response_id": "resp_001",
			},
		},
	}

	nodes := transformer.Transform(items, "test-user")

	// Should create one parent node with sub-nodes
	assert.Len(t, nodes, 1)
	assert.Equal(t, "WordCreateNewDocument", nodes[0].Name)
	assert.Equal(t, models.NodeTypeTool, nodes[0].Type)
	assert.Equal(t, models.NodeStatusCompleted, nodes[0].Status)
	assert.NotNil(t, nodes[0].Data)
	assert.Equal(t, `{"fileName":"test.docx"}`, nodes[0].Data.Input.Text)
	assert.Equal(t, `{"result":"success"}`, nodes[0].Data.Output.Text)

	// Check sub-nodes
	assert.Len(t, nodes[0].Nodes, 3) // approval_request, approval_response, mcp_call
}

func TestFoundryTransformer_Transform_MCPCallDenied(t *testing.T) {
	transformer := foundry.NewTransformer()
	denied := false

	items := []foundry.ConversationItem{
		{
			ID:           "mcpr_001",
			Type:         "mcp_approval_request",
			ServerLabel:  "MicrosoftWordFrontier",
			Name:         "WordCreateNewDocument",
			Arguments:    json.RawMessage(`{"fileName":"test.docx"}`),
			PartitionKey: "partition123",
		},
		{
			ID:                "mcpa_001",
			Type:              "mcp_approval_response",
			ApprovalRequestID: "mcpr_001",
			PartitionKey:      "partition123",
			Approve:           &denied,
		},
	}

	nodes := transformer.Transform(items, "test-user")

	assert.Len(t, nodes, 1)
	assert.Equal(t, models.NodeStatusCanceled, nodes[0].Status)
}

func TestFoundryTransformer_Transform_MCPListTools(t *testing.T) {
	transformer := foundry.NewTransformer()

	items := []foundry.ConversationItem{
		{
			ID:           "mcpl_001",
			Type:         "mcp_list_tools",
			ServerLabel:  "MicrosoftWordFrontier",
			PartitionKey: "partition123",
			Content: []interface{}{
				map[string]interface{}{
					"name":        "WordCreateNewDocument",
					"description": "Create a new Word document",
				},
			},
			CreatedBy: map[string]interface{}{
				"response_id": "resp_001",
			},
		},
	}

	nodes := transformer.Transform(items, "test-user")

	assert.Len(t, nodes, 1)
	assert.Equal(t, "MCP List Tools: MicrosoftWordFrontier", nodes[0].Name)
	assert.Equal(t, models.NodeTypeTool, nodes[0].Type)
	assert.Equal(t, models.NodeStatusCompleted, nodes[0].Status)
	assert.NotNil(t, nodes[0].Data)
	assert.NotNil(t, nodes[0].Data.Output)
	assert.Contains(t, nodes[0].Data.Output.Text, "WordCreateNewDocument")
}

func TestFoundryTransformer_Transform_MixedConversation(t *testing.T) {
	transformer := foundry.NewTransformer()

	// Simulate a conversation with messages and workflow actions (API returns newest first)
	items := []foundry.ConversationItem{
		{
			ID:     "msg_003",
			Type:   "message",
			Status: "completed",
			Role:   "assistant",
			Content: []interface{}{
				map[string]interface{}{
					"type": "output_text",
					"text": "Goodbye!",
				},
			},
		},
		{
			ID:     "wfa_001",
			Type:   "workflow_action",
			Status: "completed",
			Kind:   "EndConversation",
		},
		{
			ID:     "msg_002",
			Type:   "message",
			Status: "completed",
			Role:   "assistant",
			Content: []interface{}{
				map[string]interface{}{
					"type": "output_text",
					"text": "Hello! How can I help?",
				},
			},
		},
		{
			ID:     "msg_001",
			Type:   "message",
			Status: "completed",
			Role:   "user",
			Content: []interface{}{
				map[string]interface{}{
					"type": "input_text",
					"text": "Hi",
				},
			},
		},
	}

	nodes := transformer.Transform(items, "test-user")

	// Should have 4 nodes in chronological order (reversed from input)
	assert.Len(t, nodes, 4)
	// First should be the user message (oldest)
	assert.Equal(t, "User Message", nodes[0].Name)
	// Last should be the goodbye message (newest)
	assert.Equal(t, "Assistant Response", nodes[3].Name)
}

func TestFoundryTransformer_Transform_UnknownType(t *testing.T) {
	transformer := foundry.NewTransformer()

	items := []foundry.ConversationItem{
		{
			ID:     "unknown_001",
			Type:   "new_unknown_type",
			Status: "completed",
		},
	}

	nodes := transformer.Transform(items, "test-user")

	assert.Len(t, nodes, 1)
	assert.Equal(t, "Unknown: new_unknown_type", nodes[0].Name)
	assert.Equal(t, models.NodeTypeCustom, nodes[0].Type)
	assert.Equal(t, models.NodeStatusCompleted, nodes[0].Status)
}

func TestFoundryTransformer_Transform_StatusMapping(t *testing.T) {
	transformer := foundry.NewTransformer()

	testCases := []struct {
		inputStatus    string
		expectedStatus models.NodeStatus
	}{
		{"completed", models.NodeStatusCompleted},
		{"failed", models.NodeStatusFailed},
		{"cancelled", models.NodeStatusCanceled}, //nolint:misspell // value must stay "cancelled" for external API compatibility
		{"pending", models.NodeStatusPending},
		{"running", models.NodeStatusRunning},
		{"in_progress", models.NodeStatusRunning},
		{"", models.NodeStatusCompleted},        // Empty defaults to completed
		{"unknown", models.NodeStatusCompleted}, // Unknown defaults to completed
	}

	for _, tc := range testCases {
		t.Run(tc.inputStatus, func(t *testing.T) {
			items := []foundry.ConversationItem{
				{
					ID:     "msg_test",
					Type:   "message",
					Status: tc.inputStatus,
					Role:   "user",
				},
			}

			nodes := transformer.Transform(items, "test-user")

			assert.Len(t, nodes, 1)
			assert.Equal(t, tc.expectedStatus, nodes[0].Status)
		})
	}
}

func TestFoundryTransformer_Transform_MessageWithMultipleContentBlocks(t *testing.T) {
	transformer := foundry.NewTransformer()

	items := []foundry.ConversationItem{
		{
			ID:     "msg_001",
			Type:   "message",
			Status: "completed",
			Role:   "assistant",
			Content: []interface{}{
				map[string]interface{}{
					"type": "output_text",
					"text": "First paragraph.",
				},
				map[string]interface{}{
					"type": "output_text",
					"text": "Second paragraph.",
				},
			},
		},
	}

	nodes := transformer.Transform(items, "test-user")

	assert.Len(t, nodes, 1)
	assert.NotNil(t, nodes[0].Data)
	assert.NotNil(t, nodes[0].Data.Output)
	assert.Contains(t, nodes[0].Data.Output.Text, "First paragraph.")
	assert.Contains(t, nodes[0].Data.Output.Text, "Second paragraph.")
}

func TestFoundryTransformer_Transform_WorkflowKindFormatting(t *testing.T) {
	transformer := foundry.NewTransformer()

	testCases := []struct {
		kind         string
		expectedName string
	}{
		{"SendActivity", "Send Activity"},
		{"EndConversation", "End Conversation"},
		{"InvokeAzureAgent", "Invoke Azure Agent"},
		{"SimpleAction", "Simple Action"},
	}

	for _, tc := range testCases {
		t.Run(tc.kind, func(t *testing.T) {
			items := []foundry.ConversationItem{
				{
					ID:     "wfa_test",
					Type:   "workflow_action",
					Status: "completed",
					Kind:   tc.kind,
				},
			}

			nodes := transformer.Transform(items, "test-user")

			assert.Len(t, nodes, 1)
			assert.Equal(t, tc.expectedName, nodes[0].Name)
		})
	}
}

func TestFoundryTransformer_Transform_PreservesChronologicalOrder(t *testing.T) {
	transformer := foundry.NewTransformer()

	// API returns newest first, transformer should reverse to chronological
	items := []foundry.ConversationItem{
		{ID: "msg_005", Type: "message", Role: "assistant"},
		{ID: "msg_004", Type: "message", Role: "user"},
		{ID: "msg_003", Type: "message", Role: "assistant"},
		{ID: "msg_002", Type: "message", Role: "user"},
		{ID: "msg_001", Type: "message", Role: "user"},
	}

	nodes := transformer.Transform(items, "test-user")

	assert.Len(t, nodes, 5)
	// After reversal, first should be msg_001
	assert.Equal(t, "msg_001", nodes[0].ReferenceID)
	// Last should be msg_005
	assert.Equal(t, "msg_005", nodes[4].ReferenceID)
}

func TestFoundryTransformer_Transform_ExtractsAgentMetadata(t *testing.T) {
	transformer := foundry.NewTransformer()

	items := []foundry.ConversationItem{
		{
			ID:     "msg_001",
			Type:   "message",
			Status: "completed",
			Role:   "assistant",
			CreatedBy: map[string]interface{}{
				"response_id": "resp_001",
				"agent": map[string]interface{}{
					"type":    "agent_id",
					"name":    "MyCustomAgent",
					"version": "3",
				},
			},
		},
	}

	nodes := transformer.Transform(items, "test-user")

	assert.Len(t, nodes, 1)
	assert.NotNil(t, nodes[0].Metadata)
	agent, ok := nodes[0].Metadata["agent"].(map[string]interface{})
	assert.True(t, ok)
	assert.Equal(t, "MyCustomAgent", agent["name"])
	assert.Equal(t, "3", agent["version"])
}

// TestFoundryTransformer_Transform_SendActivityHierarchy tests that items with the same
// response_id are grouped under the SendActivity workflow_action as child nodes.
func TestFoundryTransformer_Transform_SendActivityHierarchy(t *testing.T) {
	transformer := foundry.NewTransformer()

	// Simulate API response (newest first) with items sharing the same response_id
	items := []foundry.ConversationItem{
		// Newer items first (as returned by API)
		{
			ID:     "msg_002",
			Type:   "message",
			Status: "completed",
			Role:   "assistant",
			CreatedBy: map[string]interface{}{
				"response_id": "wfresp_group1",
			},
			Content: []interface{}{
				map[string]interface{}{
					"type": "text",
					"text": "Second response",
				},
			},
		},
		{
			ID:               "wfa_sendactivity",
			Type:             "workflow_action",
			Status:           "completed",
			Kind:             "SendActivity",
			ActionID:         "action-123",
			PreviousActionID: "action-100",
			CreatedBy: map[string]interface{}{
				"response_id": "wfresp_group1",
			},
		},
		{
			ID:     "msg_001",
			Type:   "message",
			Status: "completed",
			Role:   "assistant",
			CreatedBy: map[string]interface{}{
				"response_id": "wfresp_group1",
			},
			Content: []interface{}{
				map[string]interface{}{
					"type": "text",
					"text": "First response",
				},
			},
		},
		{
			ID:     "msg_user",
			Type:   "message",
			Status: "completed",
			Role:   "user",
			// User message has no response_id - should be standalone
			Content: []interface{}{
				map[string]interface{}{
					"type": "text",
					"text": "Hello",
				},
			},
		},
	}

	nodes := transformer.Transform(items, "test-user")

	// Should have 2 root nodes: User message and SendActivity
	assert.Len(t, nodes, 2)

	// First root node should be user message (chronological order)
	assert.Equal(t, "User Message", nodes[0].Name)
	assert.Equal(t, "msg_user", nodes[0].ReferenceID)

	// Second root node should be SendActivity with children
	assert.Equal(t, "Send Activity", nodes[1].Name)
	assert.Equal(t, "wfa_sendactivity", nodes[1].ReferenceID)

	// SendActivity should have 2 child nodes (the two assistant messages)
	require.Len(t, nodes[1].Nodes, 2)
	assert.Equal(t, "Assistant Response", nodes[1].Nodes[0].Name)
	assert.Equal(t, "msg_001", nodes[1].Nodes[0].ReferenceID)
	assert.Equal(t, "Assistant Response", nodes[1].Nodes[1].Name)
	assert.Equal(t, "msg_002", nodes[1].Nodes[1].ReferenceID)
}

// TestFoundryTransformer_Transform_MultipleResponseGroups tests multiple SendActivity groups.
func TestFoundryTransformer_Transform_MultipleResponseGroups(t *testing.T) {
	transformer := foundry.NewTransformer()

	items := []foundry.ConversationItem{
		// Second group (newer)
		{
			ID:               "wfa_end",
			Type:             "workflow_action",
			Status:           "completed",
			Kind:             "EndConversation",
			ActionID:         "action-300",
			PreviousActionID: "action-200",
			CreatedBy: map[string]interface{}{
				"response_id": "wfresp_group2",
			},
		},
		{
			ID:               "wfa_sendactivity2",
			Type:             "workflow_action",
			Status:           "completed",
			Kind:             "SendActivity",
			ActionID:         "action-200",
			PreviousActionID: "action-150",
			CreatedBy: map[string]interface{}{
				"response_id": "wfresp_group2",
			},
		},
		{
			ID:     "msg_group2",
			Type:   "message",
			Status: "completed",
			Role:   "assistant",
			CreatedBy: map[string]interface{}{
				"response_id": "wfresp_group2",
			},
			Content: []interface{}{
				map[string]interface{}{
					"type": "text",
					"text": "Goodbye!",
				},
			},
		},
		{
			ID:               "wfa_invoke",
			Type:             "workflow_action",
			Status:           "completed",
			Kind:             "InvokeAzureAgent",
			ActionID:         "action-150",
			PreviousActionID: "action-100",
			CreatedBy: map[string]interface{}{
				"response_id": "wfresp_group2",
			},
		},
		// First group (older)
		{
			ID:               "wfa_sendactivity1",
			Type:             "workflow_action",
			Status:           "completed",
			Kind:             "SendActivity",
			ActionID:         "action-100",
			PreviousActionID: "",
			CreatedBy: map[string]interface{}{
				"response_id": "wfresp_group1",
			},
		},
		{
			ID:     "msg_group1",
			Type:   "message",
			Status: "completed",
			Role:   "assistant",
			CreatedBy: map[string]interface{}{
				"response_id": "wfresp_group1",
			},
			Content: []interface{}{
				map[string]interface{}{
					"type": "text",
					"text": "Hello!",
				},
			},
		},
	}

	nodes := transformer.Transform(items, "test-user")

	// New algorithm: Each workflow action is a top-level node.
	// Messages are children of their message-producing action (SendActivity/Question).
	// EndConversation is no longer a child of SendActivity — it's a standalone top-level node.
	assert.Len(t, nodes, 4)

	// 1. SendActivity1 with 1 child (msg_group1)
	assert.Equal(t, "Send Activity", nodes[0].Name)
	assert.Equal(t, "wfa_sendactivity1", nodes[0].ReferenceID)
	require.Len(t, nodes[0].Nodes, 1)
	assert.Equal(t, "msg_group1", nodes[0].Nodes[0].ReferenceID)

	// 2. InvokeAzureAgent (standalone, no children)
	assert.Equal(t, "Invoke Azure Agent", nodes[1].Name)
	assert.Equal(t, "wfa_invoke", nodes[1].ReferenceID)
	assert.Empty(t, nodes[1].Nodes)

	// 3. SendActivity2 with 1 child (msg_group2)
	assert.Equal(t, "Send Activity", nodes[2].Name)
	assert.Equal(t, "wfa_sendactivity2", nodes[2].ReferenceID)
	require.Len(t, nodes[2].Nodes, 1)
	assert.Equal(t, "msg_group2", nodes[2].Nodes[0].ReferenceID)

	// 4. EndConversation (standalone top-level — not child of SendActivity)
	assert.Equal(t, "End Conversation", nodes[3].Name)
	assert.Equal(t, "wfa_end", nodes[3].ReferenceID)
	assert.Empty(t, nodes[3].Nodes)
}

// TestFoundryTransformer_Transform_SubAgentMessageAssignment tests that sub-agent messages
// (no response_id, but with agent info) are assigned to the nearest preceding InvokeAzureAgent.
func TestFoundryTransformer_Transform_SubAgentMessageAssignment(t *testing.T) {
	transformer := foundry.NewTransformer()

	items := []foundry.ConversationItem{
		// Newest first (API order)
		{
			ID:               "wfa_end",
			Type:             "workflow_action",
			Status:           "completed",
			Kind:             "EndConversation",
			ActionID:         "action-300",
			PreviousActionID: "action-200",
			CreatedBy: map[string]interface{}{
				"response_id": "wfresp_001",
			},
		},
		{
			ID:               "wfa_send",
			Type:             "workflow_action",
			Status:           "completed",
			Kind:             "SendActivity",
			ActionID:         "action-200",
			PreviousActionID: "action-150",
			CreatedBy: map[string]interface{}{
				"response_id": "wfresp_001",
			},
		},
		{
			ID:     "msg_send",
			Type:   "message",
			Status: "completed",
			Role:   "assistant",
			CreatedBy: map[string]interface{}{
				"response_id": "wfresp_001",
				"agent":       map[string]interface{}{"name": "BasicWorkflow"},
			},
			Content: []interface{}{
				map[string]interface{}{"type": "output_text", "text": "Goodbye!"},
			},
		},
		{
			ID:     "msg_subagent",
			Type:   "message",
			Status: "completed",
			Role:   "assistant",
			CreatedBy: map[string]interface{}{
				"agent": map[string]interface{}{"name": "BasicAssistantAgent"},
			},
			Content: []interface{}{
				map[string]interface{}{"type": "output_text", "text": "Sub-agent response"},
			},
		},
		{
			ID:               "wfa_invoke",
			Type:             "workflow_action",
			Status:           "completed",
			Kind:             "InvokeAzureAgent",
			ActionID:         "action-150",
			PreviousActionID: "action-100",
			CreatedBy: map[string]interface{}{
				"response_id": "wfresp_001",
			},
		},
		{
			ID:     "msg_user",
			Type:   "message",
			Status: "completed",
			Role:   "user",
			Content: []interface{}{
				map[string]interface{}{"type": "input_text", "text": "Hello"},
			},
		},
	}

	nodes := transformer.Transform(items, "test-user")

	// Expected: user_msg, InvokeAzureAgent(→sub-agent msg), SendActivity(→goodbye msg), EndConversation
	assert.Len(t, nodes, 4)

	assert.Equal(t, "User Message", nodes[0].Name)

	assert.Equal(t, "Invoke Azure Agent", nodes[1].Name)
	require.Len(t, nodes[1].Nodes, 1)
	assert.Equal(t, "msg_subagent", nodes[1].Nodes[0].ReferenceID)
	assert.Contains(t, nodes[1].Nodes[0].Data.Output.Text, "Sub-agent response")

	assert.Equal(t, "Send Activity", nodes[2].Name)
	require.Len(t, nodes[2].Nodes, 1)
	assert.Equal(t, "msg_send", nodes[2].Nodes[0].ReferenceID)

	assert.Equal(t, "End Conversation", nodes[3].Name)
	assert.Empty(t, nodes[3].Nodes)
}

// TestFoundryTransformer_Transform_MCPAssignedToAction tests that MCP groups with response_id
// matching a workflow action become children of that action.
func TestFoundryTransformer_Transform_MCPAssignedToAction(t *testing.T) {
	transformer := foundry.NewTransformer()
	approved := true

	items := []foundry.ConversationItem{
		// Newest first
		{
			ID:               "wfa_send",
			Type:             "workflow_action",
			Status:           "completed",
			Kind:             "SendActivity",
			ActionID:         "action-200",
			PreviousActionID: "action-100",
			CreatedBy: map[string]interface{}{
				"response_id": "wfresp_001",
			},
		},
		{
			ID:     "msg_result",
			Type:   "message",
			Status: "completed",
			Role:   "assistant",
			CreatedBy: map[string]interface{}{
				"response_id": "wfresp_001",
			},
			Content: []interface{}{
				map[string]interface{}{"type": "output_text", "text": "Done!"},
			},
		},
		{
			ID:                "mcp_call_001",
			Type:              "mcp_call",
			Status:            "completed",
			ApprovalRequestID: "mcp_req_001",
			ServerLabel:       "TestServer",
			Name:              "DoSomething",
			Arguments:         json.RawMessage(`{"key":"value"}`),
			Output:            json.RawMessage(`{"result":"ok"}`),
			CreatedBy: map[string]interface{}{
				"response_id": "wfresp_001",
			},
		},
		{
			ID:                "mcp_resp_001",
			Type:              "mcp_approval_response",
			ApprovalRequestID: "mcp_req_001",
			Approve:           &approved,
		},
		{
			ID:          "mcp_req_001",
			Type:        "mcp_approval_request",
			ServerLabel: "TestServer",
			Name:        "DoSomething",
			Arguments:   json.RawMessage(`{"key":"value"}`),
			CreatedBy: map[string]interface{}{
				"response_id": "wfresp_001",
			},
		},
	}

	nodes := transformer.Transform(items, "test-user")

	// MCP group should be child of the first workflow action with matching response_id
	// SendActivity gets the message, and the first action with response_id gets the MCP group
	hasActionWithMCP := false
	for _, node := range nodes {
		if node.Type == models.NodeTypeWorkflow {
			for _, child := range node.Nodes {
				if child.Name == "DoSomething" && child.Type == models.NodeTypeTool {
					hasActionWithMCP = true
				}
			}
		}
	}
	assert.True(t, hasActionWithMCP, "MCP group should be a child of a workflow action")
}

// TestFoundryTransformer_Transform_SimpleAgent tests that simple agents (only messages, no workflow)
// produce a flat list of message nodes.
func TestFoundryTransformer_Transform_SimpleAgent(t *testing.T) {
	transformer := foundry.NewTransformer()

	items := []foundry.ConversationItem{
		{
			ID:     "msg_003",
			Type:   "message",
			Status: "completed",
			Role:   "assistant",
			CreatedBy: map[string]interface{}{
				"response_id": "resp_002",
				"agent":       map[string]interface{}{"name": "BasicAssistantAgent"},
			},
			Content: []interface{}{
				map[string]interface{}{"type": "output_text", "text": "I'm fine, thanks!"},
			},
		},
		{
			ID:     "msg_002",
			Type:   "message",
			Status: "completed",
			Role:   "user",
			Content: []interface{}{
				map[string]interface{}{"type": "input_text", "text": "How are you?"},
			},
		},
		{
			ID:     "msg_001",
			Type:   "message",
			Status: "completed",
			Role:   "assistant",
			CreatedBy: map[string]interface{}{
				"response_id": "resp_001",
				"agent":       map[string]interface{}{"name": "BasicAssistantAgent"},
			},
			Content: []interface{}{
				map[string]interface{}{"type": "output_text", "text": "Hello there!"},
			},
		},
	}

	nodes := transformer.Transform(items, "test-user")

	// No workflow actions → all messages are top-level
	assert.Len(t, nodes, 3)
	assert.Equal(t, "Assistant Response", nodes[0].Name)
	assert.Equal(t, "User Message", nodes[1].Name)
	assert.Equal(t, "Assistant Response", nodes[2].Name)
}

// TestFoundryTransformer_Transform_QuestionAction tests that Question workflow actions
// also capture their associated messages as children.
func TestFoundryTransformer_Transform_QuestionAction(t *testing.T) {
	transformer := foundry.NewTransformer()

	items := []foundry.ConversationItem{
		// Newest first
		{
			ID:     "msg_question",
			Type:   "message",
			Status: "completed",
			Role:   "assistant",
			CreatedBy: map[string]interface{}{
				"response_id": "wfresp_001",
			},
			Content: []interface{}{
				map[string]interface{}{"type": "output_text", "text": "What is your name?"},
			},
		},
		{
			ID:               "wfa_question",
			Type:             "workflow_action",
			Status:           "completed",
			Kind:             "Question",
			ActionID:         "action-100",
			PreviousActionID: "",
			CreatedBy: map[string]interface{}{
				"response_id": "wfresp_001",
			},
		},
	}

	nodes := transformer.Transform(items, "test-user")

	// Question action with its message as child
	assert.Len(t, nodes, 1)
	assert.Equal(t, "Question", nodes[0].Name)
	assert.Equal(t, models.NodeTypeWorkflow, nodes[0].Type)
	require.Len(t, nodes[0].Nodes, 1)
	assert.Equal(t, "Assistant Response", nodes[0].Nodes[0].Name)
	assert.Contains(t, nodes[0].Nodes[0].Data.Output.Text, "What is your name?")
}

// TestFoundryTransformer_Transform_FunctionCall tests that function_call items
// are correctly transformed into tool nodes with their function_call_output matched.
func TestFoundryTransformer_Transform_FunctionCall(t *testing.T) {
	transformer := foundry.NewTransformer()

	items := []foundry.ConversationItem{
		// Newest first (API order)
		{
			ID:     "fco_001",
			Type:   "function_call_output",
			CallID: "call_abc123",
			Output: json.RawMessage(`{"temperature":"22°C","condition":"sunny"}`),
		},
		{
			ID:        "fc_001",
			Type:      "function_call",
			Status:    "completed",
			Name:      "get_weather",
			CallID:    "call_abc123",
			Arguments: json.RawMessage(`{"location":"Berlin"}`),
			CreatedBy: map[string]interface{}{
				"response_id": "resp_001",
			},
		},
	}

	nodes := transformer.Transform(items, "test-user")

	// function_call should create one node, function_call_output should be absorbed
	assert.Len(t, nodes, 1)
	assert.Equal(t, "get_weather", nodes[0].Name)
	assert.Equal(t, models.NodeTypeTool, nodes[0].Type)
	assert.Equal(t, models.NodeStatusCompleted, nodes[0].Status)
	assert.Equal(t, "fc_001", nodes[0].ReferenceID)
	assert.NotNil(t, nodes[0].Data)
	assert.Equal(t, `{"location":"Berlin"}`, nodes[0].Data.Input.Text)
	assert.Equal(t, `{"temperature":"22°C","condition":"sunny"}`, nodes[0].Data.Output.Text)
}

// TestFoundryTransformer_Transform_FunctionCallWithObjectArguments tests that
// function_call items with JSON object arguments (not strings) are handled correctly.
func TestFoundryTransformer_Transform_FunctionCallWithObjectArguments(t *testing.T) {
	transformer := foundry.NewTransformer()

	items := []foundry.ConversationItem{
		{
			ID:     "fco_001",
			Type:   "function_call_output",
			CallID: "call_xyz",
			Output: json.RawMessage(`{"result": "success"}`),
		},
		{
			ID:        "fc_001",
			Type:      "function_call",
			Status:    "completed",
			Name:      "create_item",
			CallID:    "call_xyz",
			Arguments: json.RawMessage(`{"name": "test", "count": 5}`),
		},
	}

	nodes := transformer.Transform(items, "test-user")

	assert.Len(t, nodes, 1)
	assert.Equal(t, "create_item", nodes[0].Name)
	assert.Equal(t, `{"name": "test", "count": 5}`, nodes[0].Data.Input.Text)
	assert.Equal(t, `{"result": "success"}`, nodes[0].Data.Output.Text)
}

// TestFoundryTransformer_Transform_FunctionCallWithStringArguments tests that
// function_call items with JSON string arguments are handled correctly.
func TestFoundryTransformer_Transform_FunctionCallWithStringArguments(t *testing.T) {
	transformer := foundry.NewTransformer()

	items := []foundry.ConversationItem{
		{
			ID:        "fc_001",
			Type:      "function_call",
			Status:    "completed",
			Name:      "echo",
			CallID:    "call_str",
			Arguments: json.RawMessage(`"hello world"`),
		},
	}

	nodes := transformer.Transform(items, "test-user")

	assert.Len(t, nodes, 1)
	assert.Equal(t, "echo", nodes[0].Name)
	// String arguments should be unquoted
	assert.Equal(t, "hello world", nodes[0].Data.Input.Text)
}

// TestFoundryTransformer_Transform_FunctionCallAsChildOfWorkflowAction tests that
// function_call items with matching response_id become children of workflow actions.
func TestFoundryTransformer_Transform_FunctionCallAsChildOfWorkflowAction(t *testing.T) {
	transformer := foundry.NewTransformer()

	items := []foundry.ConversationItem{
		// Newest first
		{
			ID:     "fco_001",
			Type:   "function_call_output",
			CallID: "call_wf",
			Output: json.RawMessage(`{"status":"done"}`),
		},
		{
			ID:        "fc_001",
			Type:      "function_call",
			Status:    "completed",
			Name:      "do_something",
			CallID:    "call_wf",
			Arguments: json.RawMessage(`{"input":"test"}`),
			CreatedBy: map[string]interface{}{
				"response_id": "wfresp_001",
			},
		},
		{
			ID:               "wfa_001",
			Type:             "workflow_action",
			Status:           "completed",
			Kind:             "SendActivity",
			ActionID:         "action-1",
			PreviousActionID: "",
			CreatedBy: map[string]interface{}{
				"response_id": "wfresp_001",
			},
		},
	}

	nodes := transformer.Transform(items, "test-user")

	// Workflow action should have the function_call as a child
	hasActionWithFunctionCall := false
	for _, node := range nodes {
		if node.Type == models.NodeTypeWorkflow {
			for _, child := range node.Nodes {
				if child.Name == "do_something" && child.Type == models.NodeTypeTool {
					hasActionWithFunctionCall = true
					assert.Equal(t, `{"input":"test"}`, child.Data.Input.Text)
					assert.Equal(t, `{"status":"done"}`, child.Data.Output.Text)
				}
			}
		}
	}
	assert.True(t, hasActionWithFunctionCall, "function_call should be a child of workflow action")
}

// TestFoundryTransformer_Transform_StandaloneFunctionCallOutput tests that
// function_call_output without a matching function_call is handled as standalone.
func TestFoundryTransformer_Transform_StandaloneFunctionCallOutput(t *testing.T) {
	transformer := foundry.NewTransformer()

	items := []foundry.ConversationItem{
		{
			ID:     "fco_001",
			Type:   "function_call_output",
			CallID: "call_orphan",
			Output: json.RawMessage(`{"data":"orphan result"}`),
		},
	}

	nodes := transformer.Transform(items, "test-user")

	assert.Len(t, nodes, 1)
	assert.Equal(t, "Function Call Output", nodes[0].Name)
	assert.Equal(t, models.NodeTypeTool, nodes[0].Type)
	assert.Equal(t, `{"data":"orphan result"}`, nodes[0].Data.Output.Text)
}

// ─── remote_function_call tests ─────────────────────────────────────────────

// TestFoundryTransformer_Transform_RemoteFunctionCall tests that remote_function_call items
// are transformed the same way as function_call items.
func TestFoundryTransformer_Transform_RemoteFunctionCall(t *testing.T) {
	transformer := foundry.NewTransformer()

	items := []foundry.ConversationItem{
		{
			ID:     "rfco_001",
			Type:   "remote_function_call_output",
			CallID: "call_remote_1",
			Output: json.RawMessage(`"{\"planets\":[\"Tatooine\",\"Hoth\"]}"`),
		},
		{
			ID:        "rfc_001",
			Type:      "remote_function_call",
			Status:    "completed",
			Name:      "StarWarsAPI_listPlanets",
			CallID:    "call_remote_1",
			Arguments: json.RawMessage(`{}`),
			CreatedBy: map[string]interface{}{
				"response_id": "resp_r001",
			},
		},
	}

	nodes := transformer.Transform(items, "test-user")

	assert.Len(t, nodes, 1)
	assert.Equal(t, "StarWarsAPI_listPlanets", nodes[0].Name)
	assert.Equal(t, models.NodeTypeTool, nodes[0].Type)
	assert.Equal(t, models.NodeStatusCompleted, nodes[0].Status)
	assert.Equal(t, "rfc_001", nodes[0].ReferenceID)
	assert.NotNil(t, nodes[0].Data)
	assert.Equal(t, `{}`, nodes[0].Data.Input.Text)
	assert.Equal(t, `{"planets":["Tatooine","Hoth"]}`, nodes[0].Data.Output.Text)
}

// TestFoundryTransformer_Transform_RemoteFunctionCallWithObjectArguments tests that
// remote_function_call items with JSON object arguments are handled correctly.
func TestFoundryTransformer_Transform_RemoteFunctionCallWithObjectArguments(t *testing.T) {
	transformer := foundry.NewTransformer()

	items := []foundry.ConversationItem{
		{
			ID:     "rfco_001",
			Type:   "remote_function_call_output",
			CallID: "call_remote_2",
			Output: json.RawMessage(`{"name":"Luke Skywalker","height":"172"}`),
		},
		{
			ID:        "rfc_001",
			Type:      "remote_function_call",
			Status:    "completed",
			Name:      "StarWarsAPI_getPerson",
			CallID:    "call_remote_2",
			Arguments: json.RawMessage(`{"person_id": 1}`),
		},
	}

	nodes := transformer.Transform(items, "test-user")

	assert.Len(t, nodes, 1)
	assert.Equal(t, "StarWarsAPI_getPerson", nodes[0].Name)
	assert.Equal(t, `{"person_id": 1}`, nodes[0].Data.Input.Text)
	assert.Equal(t, `{"name":"Luke Skywalker","height":"172"}`, nodes[0].Data.Output.Text)
}

// TestFoundryTransformer_Transform_RemoteFunctionCallAsChildOfWorkflowAction tests that
// remote_function_call items with matching response_id become children of workflow actions.
func TestFoundryTransformer_Transform_RemoteFunctionCallAsChildOfWorkflowAction(t *testing.T) {
	transformer := foundry.NewTransformer()

	items := []foundry.ConversationItem{
		{
			ID:     "rfco_001",
			Type:   "remote_function_call_output",
			CallID: "call_wf_remote",
			Output: json.RawMessage(`{"status":"ok"}`),
		},
		{
			ID:        "rfc_001",
			Type:      "remote_function_call",
			Status:    "completed",
			Name:      "API_doSomething",
			CallID:    "call_wf_remote",
			Arguments: json.RawMessage(`{"arg":"value"}`),
			CreatedBy: map[string]interface{}{
				"response_id": "wfresp_remote_001",
			},
		},
		{
			ID:               "wfa_001",
			Type:             "workflow_action",
			Status:           "completed",
			Kind:             "SendActivity",
			ActionID:         "action-1",
			PreviousActionID: "",
			CreatedBy: map[string]interface{}{
				"response_id": "wfresp_remote_001",
			},
		},
	}

	nodes := transformer.Transform(items, "test-user")

	hasActionWithFunctionCall := false
	for _, node := range nodes {
		if node.Type == models.NodeTypeWorkflow {
			for _, child := range node.Nodes {
				if child.Name == "API_doSomething" && child.Type == models.NodeTypeTool {
					hasActionWithFunctionCall = true
					assert.Equal(t, `{"arg":"value"}`, child.Data.Input.Text)
					assert.Equal(t, `{"status":"ok"}`, child.Data.Output.Text)
				}
			}
		}
	}
	assert.True(t, hasActionWithFunctionCall, "remote_function_call should be a child of workflow action")
}

// TestFoundryTransformer_Transform_StandaloneRemoteFunctionCallOutput tests that
// remote_function_call_output without a matching function_call is handled as standalone.
func TestFoundryTransformer_Transform_StandaloneRemoteFunctionCallOutput(t *testing.T) {
	transformer := foundry.NewTransformer()

	items := []foundry.ConversationItem{
		{
			ID:     "rfco_001",
			Type:   "remote_function_call_output",
			CallID: "call_orphan_remote",
			Output: json.RawMessage(`{"data":"orphan remote result"}`),
		},
	}

	nodes := transformer.Transform(items, "test-user")

	assert.Len(t, nodes, 1)
	assert.Equal(t, "Function Call Output", nodes[0].Name)
	assert.Equal(t, models.NodeTypeTool, nodes[0].Type)
	assert.Equal(t, `{"data":"orphan remote result"}`, nodes[0].Data.Output.Text)
}

// ─── Hierarchy tests: function calls as children of assistant messages ────────

// TestFoundryTransformer_Transform_FunctionCallAsChildOfAssistantMessage tests that
// function_call items with the same response_id as an assistant message become children
// of that message (for simple agents without workflow actions).
func TestFoundryTransformer_Transform_FunctionCallAsChildOfAssistantMessage(t *testing.T) {
	transformer := foundry.NewTransformer()

	items := []foundry.ConversationItem{
		// Newest first (API order)
		{
			ID:   "msg_assistant",
			Type: "message",
			Role: "assistant",
			Content: []interface{}{
				map[string]interface{}{"type": "text", "text": "Here are some planets: Tatooine, Hoth..."},
			},
			CreatedBy: map[string]interface{}{
				"response_id": "resp_shared",
			},
		},
		{
			ID:     "fco_001",
			Type:   "function_call_output",
			CallID: "call_fc1",
			Output: json.RawMessage(`{"planets":["Tatooine","Hoth"]}`),
		},
		{
			ID:        "fc_001",
			Type:      "function_call",
			Status:    "completed",
			Name:      "listPlanets",
			CallID:    "call_fc1",
			Arguments: json.RawMessage(`{}`),
			CreatedBy: map[string]interface{}{
				"response_id": "resp_shared",
			},
		},
		{
			ID:   "msg_user",
			Type: "message",
			Role: "user",
			Content: []interface{}{
				map[string]interface{}{"type": "text", "text": "Which planets exist?"},
			},
		},
	}

	nodes := transformer.Transform(items, "test-user")

	// Should have 2 top-level nodes: user message, assistant message (function call nested inside)
	assert.Len(t, nodes, 2, "expected user msg + assistant msg, function call should be a child of assistant msg")

	// First node: user message
	assert.Equal(t, "User Message", nodes[0].Name)
	assert.Equal(t, models.NodeTypeLLM, nodes[0].Type)

	// Second node: assistant message with function call child
	assert.Equal(t, "Assistant Response", nodes[1].Name)
	assert.Equal(t, models.NodeTypeLLM, nodes[1].Type)
	assert.Len(t, nodes[1].Nodes, 1, "assistant message should have 1 function call child")
	assert.Equal(t, "listPlanets", nodes[1].Nodes[0].Name)
	assert.Equal(t, models.NodeTypeTool, nodes[1].Nodes[0].Type)
	assert.Equal(t, `{}`, nodes[1].Nodes[0].Data.Input.Text)
	assert.Equal(t, `{"planets":["Tatooine","Hoth"]}`, nodes[1].Nodes[0].Data.Output.Text)
}

// TestFoundryTransformer_Transform_RemoteFunctionCallAsChildOfAssistantMessage tests that
// remote_function_call items are nested under the assistant message with the same response_id.
func TestFoundryTransformer_Transform_RemoteFunctionCallAsChildOfAssistantMessage(t *testing.T) {
	transformer := foundry.NewTransformer()

	items := []foundry.ConversationItem{
		// Newest first (API order) - StarWarsAgent scenario
		{
			ID:   "msg_resp2",
			Type: "message",
			Role: "assistant",
			Content: []interface{}{
				map[string]interface{}{"type": "text", "text": "Hier sind einige Planeten aus dem Star Wars Universum..."},
			},
			CreatedBy: map[string]interface{}{
				"response_id": "resp_sw_002",
			},
		},
		{
			ID:     "rfco_sw1",
			Type:   "remote_function_call_output",
			CallID: "call_sw_planets",
			Output: json.RawMessage(`"{\"count\":60,\"results\":[{\"name\":\"Tatooine\"},{\"name\":\"Alderaan\"}]}"`),
		},
		{
			ID:        "rfc_sw1",
			Type:      "remote_function_call",
			Status:    "completed",
			Name:      "StarWarsAPI_listPlanets",
			CallID:    "call_sw_planets",
			Arguments: json.RawMessage(`{}`),
			CreatedBy: map[string]interface{}{
				"response_id": "resp_sw_002",
			},
		},
		{
			ID:   "msg_user2",
			Type: "message",
			Role: "user",
			Content: []interface{}{
				map[string]interface{}{"type": "text", "text": "welche planeten gibts so?"},
			},
		},
		{
			ID:   "msg_resp1",
			Type: "message",
			Role: "assistant",
			Content: []interface{}{
				map[string]interface{}{"type": "text", "text": "Hello there!"},
			},
			CreatedBy: map[string]interface{}{
				"response_id": "resp_sw_001",
			},
		},
		{
			ID:   "msg_user1",
			Type: "message",
			Role: "user",
			Content: []interface{}{
				map[string]interface{}{"type": "text", "text": "hey"},
			},
		},
	}

	nodes := transformer.Transform(items, "test-user")

	// Should have 4 top-level nodes: user->assistant->user->assistant(with child)
	assert.Len(t, nodes, 4, "expected 4 top-level nodes")

	// Node 0: user "hey"
	assert.Equal(t, "User Message", nodes[0].Name)
	assert.Len(t, nodes[0].Nodes, 0)

	// Node 1: assistant "Hello there!" (no function calls)
	assert.Equal(t, "Assistant Response", nodes[1].Name)
	assert.Len(t, nodes[1].Nodes, 0)

	// Node 2: user "welche planeten gibts so?"
	assert.Equal(t, "User Message", nodes[2].Name)
	assert.Len(t, nodes[2].Nodes, 0)

	// Node 3: assistant with nested function call
	assert.Equal(t, "Assistant Response", nodes[3].Name)
	assert.Len(t, nodes[3].Nodes, 1, "assistant msg should have 1 remote_function_call child")
	assert.Equal(t, "StarWarsAPI_listPlanets", nodes[3].Nodes[0].Name)
	assert.Equal(t, models.NodeTypeTool, nodes[3].Nodes[0].Type)
	assert.Equal(t, `{}`, nodes[3].Nodes[0].Data.Input.Text)
	// remote_function_call_output is a JSON string (double-encoded); RawMessageToString unquotes it
	assert.Equal(t, `{"count":60,"results":[{"name":"Tatooine"},{"name":"Alderaan"}]}`, nodes[3].Nodes[0].Data.Output.Text)
}

// TestFoundryTransformer_Transform_MultipleFunctionCallsAsChildOfAssistantMessage tests that
// multiple function calls with the same response_id are all nested under the assistant message.
func TestFoundryTransformer_Transform_MultipleFunctionCallsAsChildOfAssistantMessage(t *testing.T) {
	transformer := foundry.NewTransformer()

	items := []foundry.ConversationItem{
		{
			ID:   "msg_assistant",
			Type: "message",
			Role: "assistant",
			Content: []interface{}{
				map[string]interface{}{"type": "text", "text": "Here's the weather and time."},
			},
			CreatedBy: map[string]interface{}{
				"response_id": "resp_multi",
			},
		},
		{
			ID:     "fco_002",
			Type:   "remote_function_call_output",
			CallID: "call_time_1",
			Output: json.RawMessage(`"14:30 UTC"`),
		},
		{
			ID:        "fc_002",
			Type:      "remote_function_call",
			Status:    "completed",
			Name:      "getCurrentTime",
			CallID:    "call_time_1",
			Arguments: json.RawMessage(`{"timezone":"UTC"}`),
			CreatedBy: map[string]interface{}{
				"response_id": "resp_multi",
			},
		},
		{
			ID:     "fco_001",
			Type:   "remote_function_call_output",
			CallID: "call_weather_1",
			Output: json.RawMessage(`{"temp":"22°C"}`),
		},
		{
			ID:        "fc_001",
			Type:      "remote_function_call",
			Status:    "completed",
			Name:      "getWeather",
			CallID:    "call_weather_1",
			Arguments: json.RawMessage(`{"city":"Berlin"}`),
			CreatedBy: map[string]interface{}{
				"response_id": "resp_multi",
			},
		},
		{
			ID:   "msg_user",
			Type: "message",
			Role: "user",
			Content: []interface{}{
				map[string]interface{}{"type": "text", "text": "What's the weather and time?"},
			},
		},
	}

	nodes := transformer.Transform(items, "test-user")

	assert.Len(t, nodes, 2, "expected user msg + assistant msg with children")
	assert.Equal(t, "Assistant Response", nodes[1].Name)
	assert.Len(t, nodes[1].Nodes, 2, "assistant should have 2 function call children")

	// Children should be: getWeather, getCurrentTime (chronological order)
	childNames := []string{nodes[1].Nodes[0].Name, nodes[1].Nodes[1].Name}
	assert.Contains(t, childNames, "getWeather")
	assert.Contains(t, childNames, "getCurrentTime")
}

// TestFoundryTransformer_Transform_FunctionCallPreferWorkflowActionOverMessage tests that
// when both a workflow action and an assistant message have the same response_id,
// the function call is assigned to the workflow action (not duplicated on message).
func TestFoundryTransformer_Transform_FunctionCallPreferWorkflowActionOverMessage(t *testing.T) {
	transformer := foundry.NewTransformer()

	items := []foundry.ConversationItem{
		{
			ID:   "msg_assistant",
			Type: "message",
			Role: "assistant",
			Content: []interface{}{
				map[string]interface{}{"type": "text", "text": "Done."},
			},
			CreatedBy: map[string]interface{}{
				"response_id": "resp_both",
			},
		},
		{
			ID:     "fco_001",
			Type:   "function_call_output",
			CallID: "call_both_1",
			Output: json.RawMessage(`{"result":"ok"}`),
		},
		{
			ID:        "fc_001",
			Type:      "function_call",
			Status:    "completed",
			Name:      "doAction",
			CallID:    "call_both_1",
			Arguments: json.RawMessage(`{}`),
			CreatedBy: map[string]interface{}{
				"response_id": "resp_both",
			},
		},
		{
			ID:               "wfa_001",
			Type:             "workflow_action",
			Status:           "completed",
			Kind:             "SendActivity",
			ActionID:         "action-1",
			PreviousActionID: "",
			CreatedBy: map[string]interface{}{
				"response_id": "resp_both",
			},
		},
	}

	nodes := transformer.Transform(items, "test-user")

	// Function call should be on workflow action (already processed), NOT on assistant message.
	// The SendActivity workflow action also parents the assistant message (via messageParent),
	// so the workflow action has 2 children: assistant message + function call.
	for _, node := range nodes {
		if node.Type == models.NodeTypeWorkflow {
			assert.Len(t, node.Nodes, 2, "workflow action should have the message and the function call")
			hasFunctionCall := false
			for _, child := range node.Nodes {
				if child.Name == "doAction" && child.Type == models.NodeTypeTool {
					hasFunctionCall = true
				}
				// The assistant message (child of workflow action) should NOT have nested function calls
				if child.Name == "Assistant Response" {
					assert.Len(t, child.Nodes, 0, "assistant message should NOT have function call (already on workflow action)")
				}
			}
			assert.True(t, hasFunctionCall, "workflow action should contain the function call")
		}
	}
}

// ─── MCP hierarchy tests: MCP items as children of assistant messages ────────

// TestFoundryTransformer_Transform_MCPListToolsAsChildOfAssistantMessage tests that
// mcp_list_tools items with the same response_id as an assistant message become children
// of that message (WordAgent / simple MCP agent scenario).
func TestFoundryTransformer_Transform_MCPListToolsAsChildOfAssistantMessage(t *testing.T) {
	transformer := foundry.NewTransformer()

	items := []foundry.ConversationItem{
		// Newest first (API order)
		{
			ID:   "msg_assistant",
			Type: "message",
			Role: "assistant",
			Content: []interface{}{
				map[string]interface{}{"type": "text", "text": "Dynamics ist eine Suite von Microsoft..."},
			},
			CreatedBy: map[string]interface{}{
				"response_id": "resp_mcp_001",
				"agent": map[string]interface{}{
					"type": "agent_id",
					"name": "WordAgent",
				},
			},
		},
		{
			ID:          "mcpl_001",
			Type:        "mcp_list_tools",
			ServerLabel: "MicrosoftLearnMCPserver",
			CreatedBy: map[string]interface{}{
				"response_id": "resp_mcp_001",
			},
		},
		{
			ID:   "msg_user",
			Type: "message",
			Role: "user",
			Content: []interface{}{
				map[string]interface{}{"type": "input_text", "text": "was ist dynamics?"},
			},
		},
	}

	nodes := transformer.Transform(items, "test-user")

	// Should have 2 top-level nodes: user message + assistant message (MCP list tools nested inside)
	assert.Len(t, nodes, 2, "expected user msg + assistant msg, MCP list tools should be a child of assistant msg")

	assert.Equal(t, "User Message", nodes[0].Name)
	assert.Len(t, nodes[0].Nodes, 0)

	assert.Equal(t, "Assistant Response", nodes[1].Name)
	assert.Len(t, nodes[1].Nodes, 1, "assistant message should have 1 MCP list tools child")
	assert.Equal(t, "MCP List Tools: MicrosoftLearnMCPserver", nodes[1].Nodes[0].Name)
	assert.Equal(t, models.NodeTypeTool, nodes[1].Nodes[0].Type)
}

// TestFoundryTransformer_Transform_MCPCallAsChildOfAssistantMessage tests that
// standalone mcp_call items (without approval) with the same response_id as an assistant
// message become children of that message.
func TestFoundryTransformer_Transform_MCPCallAsChildOfAssistantMessage(t *testing.T) {
	transformer := foundry.NewTransformer()

	items := []foundry.ConversationItem{
		{
			ID:   "msg_assistant",
			Type: "message",
			Role: "assistant",
			Content: []interface{}{
				map[string]interface{}{"type": "text", "text": "Here's the result."},
			},
			CreatedBy: map[string]interface{}{
				"response_id": "resp_mcp_call_001",
			},
		},
		{
			ID:          "mcp_call_001",
			Type:        "mcp_call",
			Name:        "search_docs",
			ServerLabel: "LearnMCP",
			Arguments:   json.RawMessage(`{"query":"dynamics"}`),
			Output:      json.RawMessage(`{"results":["doc1","doc2"]}`),
			CreatedBy: map[string]interface{}{
				"response_id": "resp_mcp_call_001",
			},
		},
		{
			ID:   "msg_user",
			Type: "message",
			Role: "user",
			Content: []interface{}{
				map[string]interface{}{"type": "input_text", "text": "search for dynamics"},
			},
		},
	}

	nodes := transformer.Transform(items, "test-user")

	assert.Len(t, nodes, 2, "expected user msg + assistant msg with MCP call nested")
	assert.Equal(t, "Assistant Response", nodes[1].Name)
	assert.Len(t, nodes[1].Nodes, 1, "assistant message should have 1 MCP call child")
	assert.Equal(t, "MCP Call: search_docs", nodes[1].Nodes[0].Name)
	assert.Equal(t, models.NodeTypeTool, nodes[1].Nodes[0].Type)
}

// TestFoundryTransformer_Transform_MCPApprovalGroupAsChildOfAssistantMessage tests that
// mcp_approval_request groups with the same response_id as an assistant message become
// children of that message.
func TestFoundryTransformer_Transform_MCPApprovalGroupAsChildOfAssistantMessage(t *testing.T) {
	transformer := foundry.NewTransformer()

	items := []foundry.ConversationItem{
		{
			ID:   "msg_assistant",
			Type: "message",
			Role: "assistant",
			Content: []interface{}{
				map[string]interface{}{"type": "text", "text": "Done."},
			},
			CreatedBy: map[string]interface{}{
				"response_id": "resp_approve_001",
			},
		},
		{
			ID:                "mcp_call_appr",
			Type:              "mcp_call",
			Name:              "run_query",
			ServerLabel:       "DBServer",
			ApprovalRequestID: "mcp_req_001",
			Arguments:         json.RawMessage(`{"sql":"SELECT 1"}`),
			Output:            json.RawMessage(`{"rows":1}`),
			CreatedBy: map[string]interface{}{
				"response_id": "resp_approve_001",
			},
		},
		{
			ID:                "mcp_resp_001",
			Type:              "mcp_approval_response",
			ApprovalRequestID: "mcp_req_001",
			Approve:           boolPtr(true),
			CreatedBy: map[string]interface{}{
				"response_id": "resp_approve_001",
			},
		},
		{
			ID:          "mcp_req_001",
			Type:        "mcp_approval_request",
			Name:        "run_query",
			ServerLabel: "DBServer",
			Arguments:   json.RawMessage(`{"sql":"SELECT 1"}`),
			CreatedBy: map[string]interface{}{
				"response_id": "resp_approve_001",
			},
		},
		{
			ID:   "msg_user",
			Type: "message",
			Role: "user",
			Content: []interface{}{
				map[string]interface{}{"type": "input_text", "text": "run a query"},
			},
		},
	}

	nodes := transformer.Transform(items, "test-user")

	assert.Len(t, nodes, 2, "expected user msg + assistant msg with MCP group nested")
	assert.Equal(t, "Assistant Response", nodes[1].Name)
	assert.Len(t, nodes[1].Nodes, 1, "assistant message should have 1 MCP approval group child")
	assert.Equal(t, "run_query", nodes[1].Nodes[0].Name)
	assert.Equal(t, models.NodeTypeTool, nodes[1].Nodes[0].Type)
}

// TestFoundryTransformer_Transform_MCPAndFunctionCallAsChildOfAssistantMessage tests that
// both MCP items and function calls with the same response_id are nested under the assistant message.
func TestFoundryTransformer_Transform_MCPAndFunctionCallAsChildOfAssistantMessage(t *testing.T) {
	transformer := foundry.NewTransformer()

	items := []foundry.ConversationItem{
		{
			ID:   "msg_assistant",
			Type: "message",
			Role: "assistant",
			Content: []interface{}{
				map[string]interface{}{"type": "text", "text": "Here's the info."},
			},
			CreatedBy: map[string]interface{}{
				"response_id": "resp_mixed_001",
			},
		},
		{
			ID:     "fco_001",
			Type:   "remote_function_call_output",
			CallID: "call_api_1",
			Output: json.RawMessage(`{"data":"result"}`),
		},
		{
			ID:        "fc_001",
			Type:      "remote_function_call",
			Status:    "completed",
			Name:      "API_getData",
			CallID:    "call_api_1",
			Arguments: json.RawMessage(`{}`),
			CreatedBy: map[string]interface{}{
				"response_id": "resp_mixed_001",
			},
		},
		{
			ID:          "mcpl_001",
			Type:        "mcp_list_tools",
			ServerLabel: "LearnMCP",
			CreatedBy: map[string]interface{}{
				"response_id": "resp_mixed_001",
			},
		},
		{
			ID:   "msg_user",
			Type: "message",
			Role: "user",
			Content: []interface{}{
				map[string]interface{}{"type": "input_text", "text": "get data"},
			},
		},
	}

	nodes := transformer.Transform(items, "test-user")

	assert.Len(t, nodes, 2, "expected user msg + assistant msg with children")
	assert.Equal(t, "Assistant Response", nodes[1].Name)
	assert.Len(t, nodes[1].Nodes, 2, "assistant should have function call + MCP list tools as children")

	childNames := []string{nodes[1].Nodes[0].Name, nodes[1].Nodes[1].Name}
	assert.Contains(t, childNames, "API_getData")
	assert.Contains(t, childNames, "MCP List Tools: LearnMCP")
}

// boolPtr is a helper to create a pointer to a bool value.
func boolPtr(b bool) *bool {
	return &b
}
