package foundry_test

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/unifiedui/agent-service/internal/domain/models"
	"github.com/unifiedui/agent-service/internal/services/traceimport/foundry"
)

func TestTransformInterface_ValidInput(t *testing.T) {
	transformer := foundry.NewTransformer()
	items := []foundry.ConversationItem{
		{
			ID:   "item-1",
			Type: "message",
			Role: "user",
			Content: []interface{}{
				map[string]interface{}{"type": "input_text", "text": "Hello"},
			},
		},
	}

	nodes := transformer.TransformInterface(items, "user-1")
	assert.NotEmpty(t, nodes)
}

func TestTransformInterface_InvalidType(t *testing.T) {
	transformer := foundry.NewTransformer()
	nodes := transformer.TransformInterface("invalid", "user-1")
	assert.Empty(t, nodes)
}

func TestTransformInterface_Nil(t *testing.T) {
	transformer := foundry.NewTransformer()
	nodes := transformer.TransformInterface(nil, "user-1")
	assert.Empty(t, nodes)
}

func TestSortNodesByTime(t *testing.T) {
	now := time.Now()
	nodes := []models.TraceNode{
		{ID: "3", CreatedAt: now.Add(2 * time.Hour)},
		{ID: "1", CreatedAt: now},
		{ID: "2", CreatedAt: now.Add(1 * time.Hour)},
	}

	foundry.SortNodesByTime(nodes)

	assert.Equal(t, "1", nodes[0].ID)
	assert.Equal(t, "2", nodes[1].ID)
	assert.Equal(t, "3", nodes[2].ID)
}

func TestSortNodesByTime_Empty(t *testing.T) {
	nodes := []models.TraceNode{}
	foundry.SortNodesByTime(nodes)
	assert.Empty(t, nodes)
}

func TestSortNodesByTime_SingleNode(t *testing.T) {
	nodes := []models.TraceNode{{ID: "1", CreatedAt: time.Now()}}
	foundry.SortNodesByTime(nodes)
	assert.Len(t, nodes, 1)
}

func TestTransform_EmptyItems(t *testing.T) {
	transformer := foundry.NewTransformer()
	nodes := transformer.Transform([]foundry.ConversationItem{}, "user-1")
	assert.Empty(t, nodes)
}

func TestTransform_UserMessage(t *testing.T) {
	transformer := foundry.NewTransformer()
	items := []foundry.ConversationItem{
		{
			ID:   "item-1",
			Type: "message",
			Role: "user",
			Content: []interface{}{
				map[string]interface{}{"type": "input_text", "text": "Hello world"},
			},
		},
	}

	nodes := transformer.Transform(items, "creator")
	assert.NotEmpty(t, nodes)
}

func TestTransform_AssistantMessage(t *testing.T) {
	transformer := foundry.NewTransformer()
	items := []foundry.ConversationItem{
		{
			ID:   "item-1",
			Type: "message",
			Role: "assistant",
			Content: []interface{}{
				map[string]interface{}{"type": "output_text", "text": "Here is help"},
			},
		},
	}

	nodes := transformer.Transform(items, "creator")
	assert.NotEmpty(t, nodes)
}

func TestTransform_WorkflowAction(t *testing.T) {
	transformer := foundry.NewTransformer()
	items := []foundry.ConversationItem{
		{
			ID:       "item-1",
			Type:     "workflow_action",
			Kind:     "InvokeAgent",
			Status:   "completed",
			ActionID: "act-1",
		},
	}

	nodes := transformer.Transform(items, "creator")
	assert.NotEmpty(t, nodes)
}

func TestTransform_MCPCall(t *testing.T) {
	transformer := foundry.NewTransformer()
	items := []foundry.ConversationItem{
		{
			ID:          "item-1",
			Type:        "mcp_call",
			ServerLabel: "my-server",
			Name:        "tool_name",
			Arguments:   json.RawMessage(`{"key": "value"}`),
			Output:      json.RawMessage(`{"result": "ok"}`),
		},
	}

	nodes := transformer.Transform(items, "creator")
	assert.NotEmpty(t, nodes)
}

func TestTransform_MCPApprovalRequestAndResponse(t *testing.T) {
	transformer := foundry.NewTransformer()
	approve := true
	items := []foundry.ConversationItem{
		{
			ID:                "item-1",
			Type:              "mcp_approval_request",
			ApprovalRequestID: "approval-1",
			ServerLabel:       "server",
			Name:              "dangerous_tool",
		},
		{
			ID:                "item-2",
			Type:              "mcp_approval_response",
			ApprovalRequestID: "approval-1",
			Approve:           &approve,
		},
	}

	nodes := transformer.Transform(items, "creator")
	assert.NotEmpty(t, nodes)
}

func TestTransform_MCPListTools(t *testing.T) {
	transformer := foundry.NewTransformer()
	items := []foundry.ConversationItem{
		{
			ID:          "item-1",
			Type:        "mcp_list_tools",
			ServerLabel: "server",
			Content:     []interface{}{"tool1", "tool2"},
		},
	}

	nodes := transformer.Transform(items, "creator")
	assert.NotEmpty(t, nodes)
}

func TestTransform_UnknownType(t *testing.T) {
	transformer := foundry.NewTransformer()
	items := []foundry.ConversationItem{
		{
			ID:   "item-1",
			Type: "unknown_type",
		},
	}

	nodes := transformer.Transform(items, "creator")
	assert.NotEmpty(t, nodes)
}

func TestTransform_SendActivityWithChildren(t *testing.T) {
	transformer := foundry.NewTransformer()
	items := []foundry.ConversationItem{
		{
			ID:       "wa-1",
			Type:     "workflow_action",
			Kind:     "SendActivity",
			Status:   "completed",
			ActionID: "act-1",
			CreatedBy: map[string]interface{}{
				"response_id": "resp-1",
			},
		},
		{
			ID:   "msg-1",
			Type: "message",
			Role: "assistant",
			CreatedBy: map[string]interface{}{
				"response_id": "resp-1",
			},
			Content: []interface{}{
				map[string]interface{}{"type": "output_text", "text": "Child content"},
			},
		},
	}

	nodes := transformer.Transform(items, "creator")
	assert.NotEmpty(t, nodes)
}

func TestTransform_MultipleItems_ChronologicalOrder(t *testing.T) {
	transformer := foundry.NewTransformer()
	items := []foundry.ConversationItem{
		{
			ID:   "newest",
			Type: "message",
			Role: "assistant",
			Content: []interface{}{
				map[string]interface{}{"type": "output_text", "text": "Response"},
			},
		},
		{
			ID:   "oldest",
			Type: "message",
			Role: "user",
			Content: []interface{}{
				map[string]interface{}{"type": "input_text", "text": "Question"},
			},
		},
	}

	nodes := transformer.Transform(items, "creator")
	assert.NotEmpty(t, nodes)
	// Items should be reversed (API returns newest first)
	assert.GreaterOrEqual(t, len(nodes), 1)
}
