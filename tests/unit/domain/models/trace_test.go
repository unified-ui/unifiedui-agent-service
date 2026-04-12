package models

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/unifiedui/agent-service/internal/domain/models"
)

func TestNewConversationTrace(t *testing.T) {
	trace := models.NewConversationTrace("tenant-1", "agent-1", "conv-1", "user-1")

	require.Equal(t, "tenant-1", trace.TenantID)
	require.Equal(t, "agent-1", trace.ChatAgentID)
	require.Equal(t, "conv-1", trace.ConversationID)
	require.Equal(t, models.TraceContextConversation, trace.ContextType)
	require.Equal(t, "user-1", trace.CreatedBy)
	require.NotNil(t, trace.Nodes)
	require.NotNil(t, trace.Logs)
	require.False(t, trace.CreatedAt.IsZero())
}

func TestNewWorkflowTrace(t *testing.T) {
	trace := models.NewWorkflowTrace("tenant-1", "auto-1", "user-1")

	require.Equal(t, "tenant-1", trace.TenantID)
	require.Equal(t, "auto-1", trace.WorkflowID)
	require.Empty(t, trace.ChatAgentID)
	require.Equal(t, models.TraceContextWorkflow, trace.ContextType)
}

func TestNewTraceNode(t *testing.T) {
	node := models.NewTraceNode("node-1", "Test Node", models.NodeTypeLLM, "user-1")

	require.Equal(t, "node-1", node.ID)
	require.Equal(t, "Test Node", node.Name)
	require.Equal(t, models.NodeTypeLLM, node.Type)
	require.Equal(t, models.NodeStatusPending, node.Status)
	require.Equal(t, "user-1", node.CreatedBy)
	require.NotNil(t, node.Nodes)
	require.NotNil(t, node.Logs)
}

func TestTrace_AddNode(t *testing.T) {
	trace := models.NewConversationTrace("t", "a", "c", "u")
	initialLen := len(trace.Nodes)
	node := models.NewTraceNode("n1", "Node 1", models.NodeTypeAgent, "u")

	trace.AddNode(node)

	require.Len(t, trace.Nodes, initialLen+1)
}

func TestTrace_AddNodes(t *testing.T) {
	trace := models.NewConversationTrace("t", "a", "c", "u")
	initialLen := len(trace.Nodes)
	nodes := []models.TraceNode{
		models.NewTraceNode("n1", "Node 1", models.NodeTypeAgent, "u"),
		models.NewTraceNode("n2", "Node 2", models.NodeTypeTool, "u"),
	}

	trace.AddNodes(nodes)

	require.Len(t, trace.Nodes, initialLen+2)
}

func TestTrace_AddLog(t *testing.T) {
	trace := models.NewConversationTrace("t", "a", "c", "u")
	initialLen := len(trace.Logs)

	trace.AddLog("test log entry")

	require.Len(t, trace.Logs, initialLen+1)
	require.Contains(t, trace.Logs, "test log entry")
}

func TestTrace_AddLogs(t *testing.T) {
	trace := models.NewConversationTrace("t", "a", "c", "u")
	initialLen := len(trace.Logs)

	trace.AddLogs([]string{"log1", "log2"})

	require.Len(t, trace.Logs, initialLen+2)
}

func TestTrace_SetUpdatedBy(t *testing.T) {
	trace := models.NewConversationTrace("t", "a", "c", "u1")
	time.Sleep(time.Millisecond)

	trace.SetUpdatedBy("u2")

	require.Equal(t, "u2", trace.UpdatedBy)
}

func TestTrace_IsConversationContext(t *testing.T) {
	conv := models.NewConversationTrace("t", "a", "c", "u")

	require.True(t, conv.IsConversationContext())
	require.False(t, conv.IsWorkflowContext())
}

func TestTrace_IsWorkflowContext(t *testing.T) {
	auto := models.NewWorkflowTrace("t", "a", "u")

	require.True(t, auto.IsWorkflowContext())
	require.False(t, auto.IsConversationContext())
}

func TestTrace_ValidateContext_Conversation(t *testing.T) {
	trace := models.NewConversationTrace("t", "agent-1", "conv-1", "u")

	require.True(t, trace.ValidateContext())
}

func TestTrace_ValidateContext_Workflow(t *testing.T) {
	trace := models.NewWorkflowTrace("t", "auto-1", "u")

	require.True(t, trace.ValidateContext())
}

func TestTrace_ValidateContext_Invalid(t *testing.T) {
	trace := &models.Trace{ContextType: "invalid"}

	require.False(t, trace.ValidateContext())
}

func TestTrace_Validate_Valid_Conversation(t *testing.T) {
	trace := models.NewConversationTrace("t", "agent-1", "conv-1", "u")

	require.NoError(t, trace.Validate())
}

func TestTrace_Validate_Valid_Workflow(t *testing.T) {
	trace := models.NewWorkflowTrace("t", "auto-1", "u")

	require.NoError(t, trace.Validate())
}

func TestTrace_Validate_MissingTenantID(t *testing.T) {
	trace := &models.Trace{TenantID: "", ContextType: models.TraceContextConversation}

	require.Error(t, trace.Validate())
}

func TestTrace_Validate_BothContexts(t *testing.T) {
	trace := &models.Trace{
		TenantID:       "t",
		ChatAgentID:    "a",
		ConversationID: "c",
		WorkflowID:     "auto",
		ContextType:    models.TraceContextConversation,
	}

	require.Error(t, trace.Validate())
}

func TestTrace_Validate_ConversationMissingAgent(t *testing.T) {
	trace := &models.Trace{
		TenantID:       "t",
		ConversationID: "c",
		ContextType:    models.TraceContextConversation,
	}

	require.Error(t, trace.Validate())
}

func TestTrace_Validate_ConversationMissingConversation(t *testing.T) {
	trace := &models.Trace{
		TenantID:    "t",
		ChatAgentID: "a",
		ContextType: models.TraceContextConversation,
	}

	require.Error(t, trace.Validate())
}

func TestTrace_Validate_AutonomousMissingAgent(t *testing.T) {
	trace := &models.Trace{
		TenantID:    "t",
		ContextType: models.TraceContextWorkflow,
	}

	require.Error(t, trace.Validate())
}

func TestTrace_Validate_InvalidContextType(t *testing.T) {
	trace := &models.Trace{TenantID: "t", ContextType: "invalid"}

	require.Error(t, trace.Validate())
}

func TestTrace_Validate_InvalidNode(t *testing.T) {
	trace := models.NewConversationTrace("t", "a", "c", "u")
	trace.AddNode(models.TraceNode{ID: "", Name: "bad"})

	require.Error(t, trace.Validate())
}

func TestNodeStatus_IsValid(t *testing.T) {
	valid := []models.NodeStatus{
		models.NodeStatusPending, models.NodeStatusRunning, models.NodeStatusCompleted,
		models.NodeStatusFailed, models.NodeStatusSkipped, models.NodeStatusCanceled,
	}

	for _, s := range valid {
		require.True(t, s.IsValid(), "expected %s to be valid", s)
	}

	require.False(t, models.NodeStatus("unknown").IsValid())
}

func TestNodeType_IsValid(t *testing.T) {
	valid := []models.NodeType{
		models.NodeTypeAgent, models.NodeTypeTool, models.NodeTypeLLM,
		models.NodeTypeChain, models.NodeTypeRetriever, models.NodeTypeWorkflow,
		models.NodeTypeFunction, models.NodeTypeHTTP, models.NodeTypeCode,
		models.NodeTypeConditional, models.NodeTypeLoop, models.NodeTypeCustom,
	}

	for _, nt := range valid {
		require.True(t, nt.IsValid(), "expected %s to be valid", nt)
	}

	require.False(t, models.NodeType("unknown").IsValid())
}

func TestTraceNode_Validate_Valid(t *testing.T) {
	node := models.NewTraceNode("n1", "Node", models.NodeTypeLLM, "u")

	require.NoError(t, node.Validate())
}

func TestTraceNode_Validate_MissingID(t *testing.T) {
	node := &models.TraceNode{Name: "x", Type: models.NodeTypeLLM, Status: models.NodeStatusCompleted}

	require.Error(t, node.Validate())
}

func TestTraceNode_Validate_MissingName(t *testing.T) {
	node := &models.TraceNode{ID: "x", Type: models.NodeTypeLLM, Status: models.NodeStatusCompleted}

	require.Error(t, node.Validate())
}

func TestTraceNode_Validate_InvalidType(t *testing.T) {
	node := &models.TraceNode{ID: "x", Name: "y", Type: "bad", Status: models.NodeStatusCompleted}

	require.Error(t, node.Validate())
}

func TestTraceNode_Validate_InvalidStatus(t *testing.T) {
	node := &models.TraceNode{ID: "x", Name: "y", Type: models.NodeTypeLLM, Status: "bad"}

	require.Error(t, node.Validate())
}

func TestTraceNode_Validate_InvalidSubNode(t *testing.T) {
	node := models.NewTraceNode("n1", "Node", models.NodeTypeLLM, "u")
	node.Nodes = append(node.Nodes, models.TraceNode{ID: ""})

	require.Error(t, node.Validate())
}

func TestConvertLogsToStrings(t *testing.T) {
	require.Equal(t, []string{}, models.ConvertLogsToStrings(nil))

	logs := []interface{}{"hello", 42, map[string]string{"key": "val"}}
	result := models.ConvertLogsToStrings(logs)

	require.Len(t, result, 3)
	require.Equal(t, "hello", result[0])
	require.Contains(t, result[1], "42")
	require.Contains(t, result[2], "key")
}
