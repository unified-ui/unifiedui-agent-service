// Package models_test provides unit tests for trace domain models.
package models_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/unifiedui/agent-service/internal/domain/models"
)

func TestNewConversationTrace(t *testing.T) {
	// Arrange
	tenantID := "tenant-123"
	applicationID := "app-456"
	conversationID := "conv-789"
	createdBy := "user-abc"

	// Act
	trace := models.NewConversationTrace(tenantID, applicationID, conversationID, createdBy)

	// Assert
	assert.Equal(t, tenantID, trace.TenantID)
	assert.Equal(t, applicationID, trace.ApplicationID)
	assert.Equal(t, conversationID, trace.ConversationID)
	assert.Equal(t, "", trace.AutonomousAgentID)
	assert.Equal(t, models.TraceContextConversation, trace.ContextType)
	assert.Equal(t, createdBy, trace.CreatedBy)
	assert.Equal(t, createdBy, trace.UpdatedBy)
	assert.NotZero(t, trace.CreatedAt)
	assert.Equal(t, trace.CreatedAt, trace.UpdatedAt)
	assert.Empty(t, trace.Nodes)
	assert.Empty(t, trace.Logs)
}

func TestNewAutonomousAgentTrace(t *testing.T) {
	// Arrange
	tenantID := "tenant-123"
	autonomousAgentID := "auto-agent-456"
	createdBy := "user-abc"

	// Act
	trace := models.NewAutonomousAgentTrace(tenantID, autonomousAgentID, createdBy)

	// Assert
	assert.Equal(t, tenantID, trace.TenantID)
	assert.Equal(t, "", trace.ApplicationID)
	assert.Equal(t, "", trace.ConversationID)
	assert.Equal(t, autonomousAgentID, trace.AutonomousAgentID)
	assert.Equal(t, models.TraceContextAutonomousAgent, trace.ContextType)
	assert.Equal(t, createdBy, trace.CreatedBy)
	assert.NotZero(t, trace.CreatedAt)
	assert.Empty(t, trace.Nodes)
}

func TestTrace_Validate_ConversationContext_Success(t *testing.T) {
	// Arrange
	trace := models.NewConversationTrace("tenant", "app", "conv", "user")

	// Act
	err := trace.Validate()

	// Assert
	assert.NoError(t, err)
}

func TestTrace_Validate_AutonomousAgentContext_Success(t *testing.T) {
	// Arrange
	trace := models.NewAutonomousAgentTrace("tenant", "auto-agent", "user")

	// Act
	err := trace.Validate()

	// Assert
	assert.NoError(t, err)
}

func TestTrace_Validate_MissingTenantID_Error(t *testing.T) {
	// Arrange
	trace := models.NewConversationTrace("", "app", "conv", "user")

	// Act
	err := trace.Validate()

	// Assert
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "tenantId is required")
}

func TestTrace_Validate_MixedContext_Error(t *testing.T) {
	// Arrange
	trace := &models.Trace{
		TenantID:          "tenant",
		ApplicationID:     "app",
		ConversationID:    "conv",
		AutonomousAgentID: "auto-agent", // Both contexts set - invalid
		ContextType:       models.TraceContextConversation,
	}

	// Act
	err := trace.Validate()

	// Assert
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "cannot have both conversation and autonomous agent context")
}

func TestTrace_Validate_ConversationMissingApplicationID_Error(t *testing.T) {
	// Arrange
	trace := &models.Trace{
		TenantID:       "tenant",
		ConversationID: "conv",
		ContextType:    models.TraceContextConversation,
	}

	// Act
	err := trace.Validate()

	// Assert
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "applicationId is required for conversation context")
}

func TestTrace_Validate_AutonomousAgentMissingAgentID_Error(t *testing.T) {
	// Arrange
	trace := &models.Trace{
		TenantID:    "tenant",
		ContextType: models.TraceContextAutonomousAgent,
	}

	// Act
	err := trace.Validate()

	// Assert
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "autonomousAgentId is required for autonomous agent context")
}

func TestTrace_AddNode(t *testing.T) {
	// Arrange
	trace := models.NewConversationTrace("tenant", "app", "conv", "user")
	node := models.TraceNode{
		ID:     "node-1",
		Name:   "test-node",
		Type:   models.NodeTypeLLM,
		Status: models.NodeStatusCompleted,
	}

	// Act
	trace.AddNode(node)

	// Assert
	assert.Len(t, trace.Nodes, 1)
	assert.Equal(t, "node-1", trace.Nodes[0].ID)
	assert.Equal(t, "test-node", trace.Nodes[0].Name)
}

func TestTrace_AddLog(t *testing.T) {
	// Arrange
	trace := models.NewConversationTrace("tenant", "app", "conv", "user")
	log := "test log message"

	// Act
	trace.AddLog(log)

	// Assert
	assert.Len(t, trace.Logs, 1)
	assert.Equal(t, "test log message", trace.Logs[0])
}

func TestTraceNode_Validate_Success(t *testing.T) {
	// Arrange
	node := models.TraceNode{
		ID:     "node-1",
		Name:   "test-node",
		Type:   models.NodeTypeLLM,
		Status: models.NodeStatusCompleted,
	}

	// Act
	err := node.Validate()

	// Assert
	assert.NoError(t, err)
}

func TestTraceNode_Validate_MissingID_Error(t *testing.T) {
	// Arrange
	node := models.TraceNode{
		Name:   "test-node",
		Type:   models.NodeTypeLLM,
		Status: models.NodeStatusCompleted,
	}

	// Act
	err := node.Validate()

	// Assert
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "node id is required")
}

func TestTraceNode_Validate_MissingName_Error(t *testing.T) {
	// Arrange
	node := models.TraceNode{
		ID:     "node-1",
		Type:   models.NodeTypeLLM,
		Status: models.NodeStatusCompleted,
	}

	// Act
	err := node.Validate()

	// Assert
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "node name is required")
}

func TestTraceNode_Validate_InvalidType_Error(t *testing.T) {
	// Arrange
	node := models.TraceNode{
		ID:     "node-1",
		Name:   "test-node",
		Type:   "invalid-type",
		Status: models.NodeStatusCompleted,
	}

	// Act
	err := node.Validate()

	// Assert
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid node type")
}

func TestTraceNode_Validate_InvalidStatus_Error(t *testing.T) {
	// Arrange
	node := models.TraceNode{
		ID:     "node-1",
		Name:   "test-node",
		Type:   models.NodeTypeLLM,
		Status: "invalid-status",
	}

	// Act
	err := node.Validate()

	// Assert
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid node status")
}

func TestTraceNode_Validate_WithSubNodes_Success(t *testing.T) {
	// Arrange
	now := time.Now()
	node := models.TraceNode{
		ID:      "node-1",
		Name:    "parent-node",
		Type:    models.NodeTypeAgent,
		Status:  models.NodeStatusCompleted,
		StartAt: &now,
		Nodes: []models.TraceNode{
			{
				ID:     "node-1-1",
				Name:   "child-node",
				Type:   models.NodeTypeTool,
				Status: models.NodeStatusCompleted,
			},
		},
	}

	// Act
	err := node.Validate()

	// Assert
	assert.NoError(t, err)
}

func TestTraceNode_Validate_InvalidSubNode_Error(t *testing.T) {
	// Arrange
	node := models.TraceNode{
		ID:     "node-1",
		Name:   "parent-node",
		Type:   models.NodeTypeAgent,
		Status: models.NodeStatusCompleted,
		Nodes: []models.TraceNode{
			{
				ID:     "", // Invalid - missing ID
				Name:   "child-node",
				Type:   models.NodeTypeTool,
				Status: models.NodeStatusCompleted,
			},
		},
	}

	// Act
	err := node.Validate()

	// Assert
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "node id is required")
}

func TestNodeStatus_IsValid(t *testing.T) {
	validStatuses := []models.NodeStatus{
		models.NodeStatusPending,
		models.NodeStatusRunning,
		models.NodeStatusCompleted,
		models.NodeStatusFailed,
		models.NodeStatusSkipped,
		models.NodeStatusCancelled,
	}

	for _, status := range validStatuses {
		assert.True(t, status.IsValid(), "status %s should be valid", status)
	}

	invalidStatus := models.NodeStatus("invalid")
	assert.False(t, invalidStatus.IsValid())
}

func TestNodeType_IsValid(t *testing.T) {
	validTypes := []models.NodeType{
		models.NodeTypeAgent,
		models.NodeTypeTool,
		models.NodeTypeLLM,
		models.NodeTypeChain,
		models.NodeTypeRetriever,
		models.NodeTypeWorkflow,
		models.NodeTypeFunction,
		models.NodeTypeHTTP,
		models.NodeTypeCode,
		models.NodeTypeConditional,
		models.NodeTypeLoop,
		models.NodeTypeCustom,
	}

	for _, nodeType := range validTypes {
		assert.True(t, nodeType.IsValid(), "type %s should be valid", nodeType)
	}

	invalidType := models.NodeType("invalid")
	assert.False(t, invalidType.IsValid())
}
