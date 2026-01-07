// Package traceimport contains unit tests for the traceimport service.
package traceimport

import (
	"testing"

	"github.com/stretchr/testify/assert"

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
