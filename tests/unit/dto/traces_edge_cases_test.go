package dto

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/unifiedui/agent-service/internal/api/dto"
	"github.com/unifiedui/agent-service/internal/domain/models"

	"github.com/stretchr/testify/require"
)

// ============================================
// CreateTraceRequest Tests
// ============================================

func TestCreateTraceRequest_JSONSerialization(t *testing.T) {
	req := dto.CreateTraceRequest{
		ID:             "trace-001",
		ChatAgentID:    "agent-1",
		ConversationID: "conv-1",
		ReferenceID:    "ref-abc",
		ReferenceName:  "MyWorkflow",
		ReferenceMetadata: map[string]interface{}{
			"version": "1.0",
		},
		Logs: []interface{}{"log1", "log2"},
		Nodes: []dto.TraceNodeRequest{
			{ID: "n1", Name: "Start", Type: "trigger", Status: "completed"},
		},
	}

	data, err := json.Marshal(req)
	require.NoError(t, err)

	var parsed dto.CreateTraceRequest
	require.NoError(t, json.Unmarshal(data, &parsed))
	require.Equal(t, req.ID, parsed.ID)
	require.Equal(t, req.ChatAgentID, parsed.ChatAgentID)
	require.Equal(t, req.ConversationID, parsed.ConversationID)
	require.Len(t, parsed.Nodes, 1)
}

func TestCreateTraceRequest_Workflow(t *testing.T) {
	jsonStr := `{"workflowId":"aa-1","referenceId":"exec-123"}`
	var req dto.CreateTraceRequest
	require.NoError(t, json.Unmarshal([]byte(jsonStr), &req))
	require.Equal(t, "aa-1", req.WorkflowID)
	require.Empty(t, req.ChatAgentID)
	require.Empty(t, req.ConversationID)
}

func TestCreateTraceRequest_Minimal(t *testing.T) {
	jsonStr := `{}`
	var req dto.CreateTraceRequest
	require.NoError(t, json.Unmarshal([]byte(jsonStr), &req))
	require.Empty(t, req.ID)
	require.Empty(t, req.Nodes)
}

// ============================================
// AddNodesRequest Tests
// ============================================

func TestAddNodesRequest_JSONSerialization(t *testing.T) {
	now := time.Now().UTC()
	req := dto.AddNodesRequest{
		Nodes: []dto.TraceNodeRequest{
			{
				ID:      "n1",
				Name:    "Node 1",
				Type:    "llm",
				Status:  "completed",
				StartAt: &now,
				EndAt:   &now,
			},
			{
				ID:     "n2",
				Name:   "Node 2",
				Type:   "tool",
				Status: "pending",
			},
		},
	}

	data, err := json.Marshal(req)
	require.NoError(t, err)

	var parsed dto.AddNodesRequest
	require.NoError(t, json.Unmarshal(data, &parsed))
	require.Len(t, parsed.Nodes, 2)
}

// ============================================
// AddLogsRequest Tests
// ============================================

func TestAddLogsRequest_JSONSerialization(t *testing.T) {
	req := dto.AddLogsRequest{
		Logs: []interface{}{"log entry 1", "log entry 2", 123, true},
	}

	data, err := json.Marshal(req)
	require.NoError(t, err)

	var parsed dto.AddLogsRequest
	require.NoError(t, json.Unmarshal(data, &parsed))
	require.Len(t, parsed.Logs, 4)
}

// ============================================
// RefreshTraceRequest Tests
// ============================================

func TestRefreshTraceRequest_JSONSerialization(t *testing.T) {
	req := dto.RefreshTraceRequest{
		ReferenceID:   "ref-new",
		ReferenceName: "UpdatedWorkflow",
		ReferenceMetadata: map[string]interface{}{
			"updated": true,
		},
		Logs: []interface{}{"new log"},
		Nodes: []dto.TraceNodeRequest{
			{ID: "n1", Name: "Updated", Type: "llm", Status: "completed"},
		},
	}

	data, err := json.Marshal(req)
	require.NoError(t, err)

	var parsed dto.RefreshTraceRequest
	require.NoError(t, json.Unmarshal(data, &parsed))
	require.Equal(t, req.ReferenceID, parsed.ReferenceID)
	require.Len(t, parsed.Nodes, 1)
}

// ============================================
// TraceNodeRequest Edge Cases
// ============================================

func TestTraceNodeRequest_WithAllDataFields(t *testing.T) {
	now := time.Now().UTC()
	req := dto.TraceNodeRequest{
		ID:          "node-full",
		Name:        "Full Node",
		Type:        "llm",
		ReferenceID: "ref-123",
		StartAt:     &now,
		EndAt:       &now,
		Duration:    5.25,
		Status:      "completed",
		Logs:        []interface{}{"log1", "log2", "log3"},
		Data: &dto.NodeDataRequest{
			Input: &dto.NodeDataIORequest{
				Text:      "Input prompt text",
				ExtraData: map[string]interface{}{"tokens": 100},
				Metadata:  map[string]interface{}{"source": "api"},
			},
			Output: &dto.NodeDataIORequest{
				Text:      "Output response text",
				ExtraData: map[string]interface{}{"tokens": 200},
				Metadata:  map[string]interface{}{"model": "gpt-4"},
			},
		},
		Metadata: map[string]interface{}{"custom": "value"},
	}

	// Test JSON round-trip
	data, err := json.Marshal(req)
	require.NoError(t, err)

	var parsed dto.TraceNodeRequest
	require.NoError(t, json.Unmarshal(data, &parsed))
	require.Equal(t, req.ID, parsed.ID)
	require.NotNil(t, parsed.Data)
	require.NotNil(t, parsed.Data.Input)
	require.NotNil(t, parsed.Data.Output)
	require.Equal(t, "Input prompt text", parsed.Data.Input.Text)

	// Test conversion to model
	node := req.ToTraceNode("test-user")
	require.Equal(t, "node-full", node.ID)
	require.NotNil(t, node.Data)
	require.NotNil(t, node.Data.Input)
	require.NotNil(t, node.Data.Output)
}

func TestTraceNodeRequest_OnlyInput(t *testing.T) {
	req := dto.TraceNodeRequest{
		ID:     "n1",
		Name:   "Input Only",
		Type:   "tool",
		Status: "pending",
		Data: &dto.NodeDataRequest{
			Input: &dto.NodeDataIORequest{
				Text: "input only",
			},
		},
	}

	node := req.ToTraceNode("u")
	require.NotNil(t, node.Data)
	require.NotNil(t, node.Data.Input)
	require.Nil(t, node.Data.Output)
}

func TestTraceNodeRequest_OnlyOutput(t *testing.T) {
	req := dto.TraceNodeRequest{
		ID:     "n1",
		Name:   "Output Only",
		Type:   "tool",
		Status: "completed",
		Data: &dto.NodeDataRequest{
			Output: &dto.NodeDataIORequest{
				Text: "output only",
			},
		},
	}

	node := req.ToTraceNode("u")
	require.NotNil(t, node.Data)
	require.Nil(t, node.Data.Input)
	require.NotNil(t, node.Data.Output)
}

func TestTraceNodeRequest_DeeplyNestedNodes(t *testing.T) {
	req := dto.TraceNodeRequest{
		ID:     "root",
		Name:   "Root",
		Type:   "agent",
		Status: "completed",
		Nodes: []dto.TraceNodeRequest{
			{
				ID:     "level1",
				Name:   "Level 1",
				Type:   "tool",
				Status: "completed",
				Nodes: []dto.TraceNodeRequest{
					{
						ID:     "level2",
						Name:   "Level 2",
						Type:   "llm",
						Status: "completed",
						Nodes: []dto.TraceNodeRequest{
							{
								ID:     "level3",
								Name:   "Level 3",
								Type:   "tool",
								Status: "completed",
							},
						},
					},
				},
			},
		},
	}

	node := req.ToTraceNode("user")
	require.Len(t, node.Nodes, 1)
	require.Len(t, node.Nodes[0].Nodes, 1)
	require.Len(t, node.Nodes[0].Nodes[0].Nodes, 1)
	require.Equal(t, "level3", node.Nodes[0].Nodes[0].Nodes[0].ID)

	// All levels should have same creator
	require.Equal(t, "user", node.Nodes[0].Nodes[0].Nodes[0].CreatedBy)
}

// ============================================
// TraceNodeToResponse Edge Cases
// ============================================

func TestTraceNodeToResponse_WithPartialData(t *testing.T) {
	now := time.Now().UTC()

	// Only input
	node := models.TraceNode{
		ID:        "n1",
		Name:      "N1",
		Type:      models.NodeTypeLLM,
		Status:    models.NodeStatusCompleted,
		CreatedAt: now,
		UpdatedAt: now,
		Data: &models.NodeData{
			Input: &models.NodeDataIO{Text: "input"},
		},
	}

	resp := dto.TraceNodeToResponse(node)
	require.NotNil(t, resp.Data)
	require.NotNil(t, resp.Data.Input)
	require.Nil(t, resp.Data.Output)

	// Only output
	node.Data = &models.NodeData{
		Output: &models.NodeDataIO{Text: "output"},
	}
	resp = dto.TraceNodeToResponse(node)
	require.NotNil(t, resp.Data)
	require.Nil(t, resp.Data.Input)
	require.NotNil(t, resp.Data.Output)
}

func TestTraceNodeToResponse_DeeplyNested(t *testing.T) {
	now := time.Now().UTC()
	node := models.TraceNode{
		ID:        "root",
		Name:      "Root",
		Type:      models.NodeTypeAgent,
		Status:    models.NodeStatusCompleted,
		CreatedAt: now,
		UpdatedAt: now,
		Nodes: []models.TraceNode{
			{
				ID:        "child1",
				Name:      "Child 1",
				Type:      models.NodeTypeTool,
				Status:    models.NodeStatusCompleted,
				CreatedAt: now,
				UpdatedAt: now,
				Nodes: []models.TraceNode{
					{
						ID:        "grandchild",
						Name:      "Grandchild",
						Type:      models.NodeTypeLLM,
						Status:    models.NodeStatusCompleted,
						CreatedAt: now,
						UpdatedAt: now,
					},
				},
			},
		},
	}

	resp := dto.TraceNodeToResponse(node)
	require.Len(t, resp.Nodes, 1)
	require.Len(t, resp.Nodes[0].Nodes, 1)
	require.Equal(t, "grandchild", resp.Nodes[0].Nodes[0].ID)
}

// ============================================
// TraceResponse Edge Cases
// ============================================

func TestTraceToResponse_ChatContext(t *testing.T) {
	now := time.Now().UTC()
	trace := &models.Trace{
		ID:             "t1",
		TenantID:       "ten-1",
		ChatAgentID:    "agent-1",
		ConversationID: "conv-1",
		ContextType:    models.TraceContextConversation,
		CreatedAt:      now,
		UpdatedAt:      now,
	}

	resp := dto.TraceToResponse(trace)
	require.Equal(t, "conversation", resp.ContextType)
	require.NotEmpty(t, resp.ChatAgentID)
	require.NotEmpty(t, resp.ConversationID)
	require.Empty(t, resp.WorkflowID)
}

func TestTraceToResponse_AutonomousContext(t *testing.T) {
	now := time.Now().UTC()
	trace := &models.Trace{
		ID:          "t2",
		TenantID:    "ten-1",
		WorkflowID:  "aa-1",
		ContextType: models.TraceContextWorkflow,
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	resp := dto.TraceToResponse(trace)
	require.Equal(t, "workflow", resp.ContextType)
	require.Empty(t, resp.ChatAgentID)
	require.Empty(t, resp.ConversationID)
	require.NotEmpty(t, resp.WorkflowID)
}

func TestTraceToResponse_WithReferenceMetadata(t *testing.T) {
	now := time.Now().UTC()
	trace := &models.Trace{
		ID:             "t3",
		TenantID:       "ten-1",
		ContextType:    models.TraceContextConversation,
		ChatAgentID:    "a",
		ConversationID: "c",
		ReferenceID:    "n8n-exec-123",
		ReferenceName:  "N8N Workflow",
		ReferenceMetadata: map[string]interface{}{
			"workflowId":  "wf-1",
			"executionId": "exec-123",
		},
		CreatedAt: now,
		UpdatedAt: now,
	}

	resp := dto.TraceToResponse(trace)
	require.Equal(t, "n8n-exec-123", resp.ReferenceID)
	require.Equal(t, "N8N Workflow", resp.ReferenceName)
	require.NotNil(t, resp.ReferenceMetadata)
	require.Equal(t, "wf-1", resp.ReferenceMetadata["workflowId"])
}

func TestTraceToResponse_EmptyNodes(t *testing.T) {
	now := time.Now().UTC()
	trace := &models.Trace{
		ID:             "t4",
		TenantID:       "ten-1",
		ContextType:    models.TraceContextConversation,
		ChatAgentID:    "a",
		ConversationID: "c",
		Nodes:          []models.TraceNode{},
		CreatedAt:      now,
		UpdatedAt:      now,
	}

	resp := dto.TraceToResponse(trace)
	require.Nil(t, resp.Nodes)
}

// ============================================
// Autonomous Agent Import DTOs
// ============================================

func TestWorkflowImportTraceRequest_JSONSerialization(t *testing.T) {
	req := dto.WorkflowImportTraceRequest{
		Type:        "N8N",
		ExecutionID: "exec-456",
		SessionID:   "session-789",
	}

	data, err := json.Marshal(req)
	require.NoError(t, err)

	var parsed dto.WorkflowImportTraceRequest
	require.NoError(t, json.Unmarshal(data, &parsed))
	require.Equal(t, req.Type, parsed.Type)
	require.Equal(t, req.ExecutionID, parsed.ExecutionID)
	require.Equal(t, req.SessionID, parsed.SessionID)
}

func TestWorkflowImportTraceRequest_WithoutSessionID(t *testing.T) {
	jsonStr := `{"type":"MICROSOFT_FOUNDRY","executionId":"exec-123"}`
	var req dto.WorkflowImportTraceRequest
	require.NoError(t, json.Unmarshal([]byte(jsonStr), &req))
	require.Equal(t, "MICROSOFT_FOUNDRY", req.Type)
	require.Empty(t, req.SessionID)
}

// ============================================
// Response DTOs JSON Structure
// ============================================

func TestListTracesResponse_JSONSerialization(t *testing.T) {
	now := time.Now().UTC()
	resp := dto.ListTracesResponse{
		Traces: []*dto.TraceResponse{
			{ID: "t1", TenantID: "ten", ContextType: "conversation", CreatedAt: now, UpdatedAt: now},
		},
		Total: 1,
	}

	data, err := json.Marshal(resp)
	require.NoError(t, err)

	var parsed dto.ListTracesResponse
	require.NoError(t, json.Unmarshal(data, &parsed))
	require.Len(t, parsed.Traces, 1)
	require.Equal(t, int64(1), parsed.Total)
}

func TestCreateTraceResponse_JSONSerialization(t *testing.T) {
	resp := dto.CreateTraceResponse{
		ID: "trace-new-123",
	}

	data, err := json.Marshal(resp)
	require.NoError(t, err)

	var parsed dto.CreateTraceResponse
	require.NoError(t, json.Unmarshal(data, &parsed))
	require.Equal(t, resp.ID, parsed.ID)
}

func TestImportTraceResponse_JSONSerialization(t *testing.T) {
	resp := dto.ImportTraceResponse{
		ID: "imported-trace-456",
	}

	data, err := json.Marshal(resp)
	require.NoError(t, err)

	var parsed dto.ImportTraceResponse
	require.NoError(t, json.Unmarshal(data, &parsed))
	require.Equal(t, resp.ID, parsed.ID)
}

// ============================================
// NodeDataIO Response Tests
// ============================================

func TestNodeDataIOResponse_JSONSerialization(t *testing.T) {
	resp := dto.NodeDataIOResponse{
		Text: "Some text content",
		ExtraData: map[string]interface{}{
			"tokens":  150,
			"success": true,
		},
		Metadata: map[string]interface{}{
			"model":   "gpt-4",
			"version": "1.0",
		},
	}

	data, err := json.Marshal(resp)
	require.NoError(t, err)

	var parsed dto.NodeDataIOResponse
	require.NoError(t, json.Unmarshal(data, &parsed))
	require.Equal(t, resp.Text, parsed.Text)
	require.NotNil(t, parsed.ExtraData)
	require.NotNil(t, parsed.Metadata)
}

func TestNodeDataResponse_JSONSerialization(t *testing.T) {
	resp := dto.NodeDataResponse{
		Input: &dto.NodeDataIOResponse{
			Text: "input text",
		},
		Output: &dto.NodeDataIOResponse{
			Text: "output text",
		},
	}

	data, err := json.Marshal(resp)
	require.NoError(t, err)

	var parsed dto.NodeDataResponse
	require.NoError(t, json.Unmarshal(data, &parsed))
	require.NotNil(t, parsed.Input)
	require.NotNil(t, parsed.Output)
}

// ============================================
// TraceNodeResponse JSON Tests
// ============================================

func TestTraceNodeResponse_JSONSerialization(t *testing.T) {
	now := time.Now().UTC()
	resp := dto.TraceNodeResponse{
		ID:          "node-resp-1",
		Name:        "Response Node",
		Type:        "llm",
		ReferenceID: "ref-1",
		StartAt:     &now,
		EndAt:       &now,
		Duration:    2.5,
		Status:      "completed",
		Logs:        []string{"log1", "log2"},
		Data: &dto.NodeDataResponse{
			Input:  &dto.NodeDataIOResponse{Text: "in"},
			Output: &dto.NodeDataIOResponse{Text: "out"},
		},
		Metadata:  map[string]interface{}{"key": "value"},
		CreatedAt: now,
		UpdatedAt: now,
		CreatedBy: "user-1",
		UpdatedBy: "user-2",
	}

	data, err := json.Marshal(resp)
	require.NoError(t, err)

	var parsed dto.TraceNodeResponse
	require.NoError(t, json.Unmarshal(data, &parsed))
	require.Equal(t, resp.ID, parsed.ID)
	require.Equal(t, resp.Name, parsed.Name)
	require.NotNil(t, parsed.Data)
	require.Len(t, parsed.Logs, 2)
}
