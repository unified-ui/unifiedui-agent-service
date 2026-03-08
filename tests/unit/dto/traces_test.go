package dto

import (
	"testing"
	"time"

	"github.com/unifiedui/agent-service/internal/api/dto"
	"github.com/unifiedui/agent-service/internal/domain/models"

	"github.com/stretchr/testify/require"
)

func TestTraceNodeRequest_ToTraceNode(t *testing.T) {
	now := time.Now().UTC()
	req := &dto.TraceNodeRequest{
		ID:          "node-1",
		Name:        "Test Node",
		Type:        "llm",
		ReferenceID: "ref-1",
		StartAt:     &now,
		EndAt:       &now,
		Duration:    1.5,
		Status:      "completed",
		Logs:        []interface{}{"log entry"},
		Data: &dto.NodeDataRequest{
			Input: &dto.NodeDataIORequest{
				Text:      "input text",
				ExtraData: map[string]interface{}{"key": "val"},
				Metadata:  map[string]interface{}{"m": "v"},
			},
			Output: &dto.NodeDataIORequest{
				Text: "output text",
			},
		},
		Metadata: map[string]interface{}{"meta": "data"},
	}

	node := req.ToTraceNode("user-1")
	require.Equal(t, "node-1", node.ID)
	require.Equal(t, "Test Node", node.Name)
	require.Equal(t, models.NodeType("llm"), node.Type)
	require.Equal(t, "ref-1", node.ReferenceID)
	require.Equal(t, models.NodeStatus("completed"), node.Status)
	require.Equal(t, "user-1", node.CreatedBy)
	require.NotNil(t, node.Data)
	require.Equal(t, "input text", node.Data.Input.Text)
	require.Equal(t, "output text", node.Data.Output.Text)
	require.Len(t, node.Logs, 1)
}

func TestTraceNodeRequest_ToTraceNode_Minimal(t *testing.T) {
	req := &dto.TraceNodeRequest{
		ID:     "n1",
		Name:   "N",
		Type:   "tool",
		Status: "pending",
	}
	node := req.ToTraceNode("u")
	require.Equal(t, "n1", node.ID)
	require.Nil(t, node.Data)
	require.Empty(t, node.Nodes)
}

func TestTraceNodeRequest_ToTraceNode_WithSubNodes(t *testing.T) {
	req := &dto.TraceNodeRequest{
		ID:     "parent",
		Name:   "Parent",
		Type:   "agent",
		Status: "completed",
		Nodes: []dto.TraceNodeRequest{
			{ID: "child", Name: "Child", Type: "tool", Status: "completed"},
		},
	}
	node := req.ToTraceNode("u")
	require.Len(t, node.Nodes, 1)
	require.Equal(t, "child", node.Nodes[0].ID)
}

func TestTraceNodeToResponse(t *testing.T) {
	now := time.Now().UTC()
	node := models.TraceNode{
		ID:        "n1",
		Name:      "LLM Call",
		Type:      models.NodeTypeLLM,
		Status:    models.NodeStatusCompleted,
		StartAt:   &now,
		EndAt:     &now,
		Duration:  2.0,
		Logs:      []string{"log1"},
		CreatedAt: now,
		UpdatedAt: now,
		CreatedBy: "u1",
		Data: &models.NodeData{
			Input:  &models.NodeDataIO{Text: "in"},
			Output: &models.NodeDataIO{Text: "out"},
		},
		Nodes: []models.TraceNode{
			{ID: "sub", Name: "Sub", Type: models.NodeTypeTool, Status: models.NodeStatusCompleted, CreatedAt: now, UpdatedAt: now},
		},
	}

	resp := dto.TraceNodeToResponse(node)
	require.Equal(t, "n1", resp.ID)
	require.Equal(t, "llm", resp.Type)
	require.Equal(t, "completed", resp.Status)
	require.NotNil(t, resp.Data)
	require.Equal(t, "in", resp.Data.Input.Text)
	require.Len(t, resp.Nodes, 1)
}

func TestTraceNodeToResponse_NilData(t *testing.T) {
	node := models.TraceNode{
		ID:     "n1",
		Name:   "N",
		Type:   models.NodeTypeTool,
		Status: models.NodeStatusCompleted,
	}
	resp := dto.TraceNodeToResponse(node)
	require.Nil(t, resp.Data)
}

func TestTraceToResponse(t *testing.T) {
	now := time.Now().UTC()
	trace := &models.Trace{
		ID:             "t1",
		TenantID:       "ten-1",
		ChatAgentID:    "a1",
		ConversationID: "c1",
		ContextType:    models.TraceContextConversation,
		CreatedAt:      now,
		UpdatedAt:      now,
		Nodes: []models.TraceNode{
			{ID: "n1", Name: "Node", Type: models.NodeTypeLLM, Status: models.NodeStatusCompleted, CreatedAt: now, UpdatedAt: now},
		},
		Logs: []string{"log"},
	}

	resp := dto.TraceToResponse(trace)
	require.NotNil(t, resp)
	require.Equal(t, "t1", resp.ID)
	require.Equal(t, "conversation", resp.ContextType)
	require.Len(t, resp.Nodes, 1)
}

func TestTraceToResponse_Nil(t *testing.T) {
	require.Nil(t, dto.TraceToResponse(nil))
}

func TestTracesToResponse(t *testing.T) {
	traces := []*models.Trace{
		{ID: "t1", TenantID: "ten-1", ContextType: models.TraceContextConversation, ChatAgentID: "a", ConversationID: "c"},
		{ID: "t2", TenantID: "ten-1", ContextType: models.TraceContextAutonomousAgent, AutonomousAgentID: "aa"},
	}
	resp := dto.TracesToResponse(traces)
	require.Len(t, resp, 2)
	require.Equal(t, "t1", resp[0].ID)
}

func TestTracesToResponse_Nil(t *testing.T) {
	resp := dto.TracesToResponse(nil)
	require.NotNil(t, resp)
	require.Empty(t, resp)
}

func TestConvertNodesToModel(t *testing.T) {
	nodes := []dto.TraceNodeRequest{
		{ID: "n1", Name: "N1", Type: "llm", Status: "completed"},
		{ID: "n2", Name: "N2", Type: "tool", Status: "pending"},
	}
	result := dto.ConvertNodesToModel(nodes, "user1")
	require.Len(t, result, 2)
	require.Equal(t, "n1", result[0].ID)
	require.Equal(t, "user1", result[0].CreatedBy)
}

func TestConvertNodesToModel_Nil(t *testing.T) {
	result := dto.ConvertNodesToModel(nil, "u")
	require.NotNil(t, result)
	require.Empty(t, result)
}
