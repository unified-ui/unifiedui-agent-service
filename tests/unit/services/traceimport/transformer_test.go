// Package traceimport contains unit tests for the traceimport service.
package traceimport

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/unifiedui/agent-service/internal/domain/models"
	"github.com/unifiedui/agent-service/internal/services/traceimport"
)

func TestFoundryTransformer_Transform_EmptyItems(t *testing.T) {
	transformer := traceimport.NewFoundryTransformer()

	nodes := transformer.Transform([]traceimport.FoundryConversationItem{}, "test-user")

	assert.Empty(t, nodes)
}

func TestFoundryTransformer_Transform_SingleUserMessage(t *testing.T) {
	transformer := traceimport.NewFoundryTransformer()

	items := []traceimport.FoundryConversationItem{
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
	transformer := traceimport.NewFoundryTransformer()

	items := []traceimport.FoundryConversationItem{
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
	transformer := traceimport.NewFoundryTransformer()

	items := []traceimport.FoundryConversationItem{
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
	assert.Equal(t, "SendActivity", nodes[0].Name) // SendActivity containers use literal name
	assert.Equal(t, models.NodeTypeWorkflow, nodes[0].Type)
	assert.Equal(t, models.NodeStatusCompleted, nodes[0].Status)
	assert.Equal(t, "wfa_001", nodes[0].ReferenceID)
	assert.NotNil(t, nodes[0].Metadata)
	assert.Equal(t, "action-123", nodes[0].Metadata["action_id"])
	assert.Equal(t, "trigger_wf", nodes[0].Metadata["parent_action_id"])
	assert.Equal(t, "SendActivity", nodes[0].Metadata["kind"])
}

func TestFoundryTransformer_Transform_MCPCallWithApproval(t *testing.T) {
	transformer := traceimport.NewFoundryTransformer()
	approved := true

	items := []traceimport.FoundryConversationItem{
		{
			ID:           "mcpr_001",
			Type:         "mcp_approval_request",
			ServerLabel:  "MicrosoftWordFrontier",
			Name:         "WordCreateNewDocument",
			Arguments:    `{"fileName":"test.docx"}`,
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
			Arguments:         `{"fileName":"test.docx"}`,
			Output:            `{"result":"success"}`,
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
	transformer := traceimport.NewFoundryTransformer()
	denied := false

	items := []traceimport.FoundryConversationItem{
		{
			ID:           "mcpr_001",
			Type:         "mcp_approval_request",
			ServerLabel:  "MicrosoftWordFrontier",
			Name:         "WordCreateNewDocument",
			Arguments:    `{"fileName":"test.docx"}`,
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
	assert.Equal(t, models.NodeStatusCancelled, nodes[0].Status)
}

func TestFoundryTransformer_Transform_MCPListTools(t *testing.T) {
	transformer := traceimport.NewFoundryTransformer()

	items := []traceimport.FoundryConversationItem{
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
	transformer := traceimport.NewFoundryTransformer()

	// Simulate a conversation with messages and workflow actions (API returns newest first)
	items := []traceimport.FoundryConversationItem{
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
	transformer := traceimport.NewFoundryTransformer()

	items := []traceimport.FoundryConversationItem{
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
	transformer := traceimport.NewFoundryTransformer()

	testCases := []struct {
		inputStatus    string
		expectedStatus models.NodeStatus
	}{
		{"completed", models.NodeStatusCompleted},
		{"failed", models.NodeStatusFailed},
		{"cancelled", models.NodeStatusCancelled},
		{"pending", models.NodeStatusPending},
		{"running", models.NodeStatusRunning},
		{"in_progress", models.NodeStatusRunning},
		{"", models.NodeStatusCompleted},        // Empty defaults to completed
		{"unknown", models.NodeStatusCompleted}, // Unknown defaults to completed
	}

	for _, tc := range testCases {
		t.Run(tc.inputStatus, func(t *testing.T) {
			items := []traceimport.FoundryConversationItem{
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
	transformer := traceimport.NewFoundryTransformer()

	items := []traceimport.FoundryConversationItem{
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
	transformer := traceimport.NewFoundryTransformer()

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
			items := []traceimport.FoundryConversationItem{
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
	transformer := traceimport.NewFoundryTransformer()

	// API returns newest first, transformer should reverse to chronological
	items := []traceimport.FoundryConversationItem{
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
	transformer := traceimport.NewFoundryTransformer()

	items := []traceimport.FoundryConversationItem{
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
	transformer := traceimport.NewFoundryTransformer()

	// Simulate API response (newest first) with items sharing the same response_id
	items := []traceimport.FoundryConversationItem{
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

	// Second root node should be SendActivity container
	assert.Equal(t, "SendActivity", nodes[1].Name)
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
	transformer := traceimport.NewFoundryTransformer()

	items := []traceimport.FoundryConversationItem{
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

	// Should have 2 root nodes (2 SendActivity containers)
	assert.Len(t, nodes, 2)

	// First: SendActivity1 with 1 child (msg_group1)
	assert.Equal(t, "SendActivity", nodes[0].Name)
	assert.Equal(t, "wfa_sendactivity1", nodes[0].ReferenceID)
	require.Len(t, nodes[0].Nodes, 1)
	assert.Equal(t, "msg_group1", nodes[0].Nodes[0].ReferenceID)

	// Second: SendActivity2 with 3 children (InvokeAzureAgent, msg_group2, EndConversation)
	assert.Equal(t, "SendActivity", nodes[1].Name)
	assert.Equal(t, "wfa_sendactivity2", nodes[1].ReferenceID)
	require.Len(t, nodes[1].Nodes, 3)

	// Children should include the other workflow actions and the message
	childRefIDs := []string{}
	for _, child := range nodes[1].Nodes {
		childRefIDs = append(childRefIDs, child.ReferenceID)
	}
	assert.Contains(t, childRefIDs, "wfa_invoke")
	assert.Contains(t, childRefIDs, "msg_group2")
	assert.Contains(t, childRefIDs, "wfa_end")
}
