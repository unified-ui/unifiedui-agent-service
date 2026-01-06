// Package testutils provides test utilities and helpers.
package testutils

import (
	"time"

	"github.com/unifiedui/agent-service/internal/domain/models"
)

// Test constants
const (
	TestTenantID       = "tenant-test-123"
	TestConversationID = "conv-test-456"
	TestMessageID      = "msg-test-789"
	TestApplicationID  = "app-test-abc"
	TestUserID         = "user-test-def"
	TestTraceID        = "trace-test-xyz"
)

// NewTestUserMessage creates a test user message with default values.
func NewTestUserMessage() *models.Message {
	now := time.Now().UTC()
	return &models.Message{
		ID:             TestMessageID,
		Type:           models.MessageTypeUser,
		TenantID:       TestTenantID,
		ConversationID: TestConversationID,
		ApplicationID:  TestApplicationID,
		UserID:         TestUserID,
		Content:        "Test message content",
		CreatedAt:      now,
		UpdatedAt:      now,
	}
}

// NewTestAssistantMessage creates a test assistant message.
func NewTestAssistantMessage() *models.Message {
	now := time.Now().UTC()
	return &models.Message{
		ID:             TestMessageID + "-assistant",
		Type:           models.MessageTypeAssistant,
		TenantID:       TestTenantID,
		ConversationID: TestConversationID,
		ApplicationID:  TestApplicationID,
		UserMessageID:  TestMessageID,
		Content:        "Test assistant response",
		Status:         models.MessageStatusSuccess,
		StatusTraces:   []models.StatusTrace{},
		CreatedAt:      now,
		UpdatedAt:      now,
	}
}

// NewTestTrace creates a test trace with default values.
func NewTestTrace() *models.Trace {
	now := time.Now().UTC()
	return &models.Trace{
		ID:             TestTraceID,
		TenantID:       TestTenantID,
		ApplicationID:  TestApplicationID,
		ConversationID: TestConversationID,
		ContextType:    models.TraceContextConversation,
		ReferenceID:    "workflow-execution-123",
		ReferenceName:  "Test Workflow Run",
		Nodes: []models.TraceNode{
			{
				ID:        "node-1",
				Name:      "test-llm-call",
				Type:      models.NodeTypeLLM,
				Status:    models.NodeStatusCompleted,
				StartAt:   timePtr(now.Add(-100 * time.Millisecond)),
				EndAt:     timePtr(now),
				Duration:  0.1,
				CreatedAt: now,
				UpdatedAt: now,
			},
		},
		Logs:      []string{},
		CreatedAt: now,
		UpdatedAt: now,
		CreatedBy: TestUserID,
		UpdatedBy: TestUserID,
	}
}

// NewTestToolTrace creates a test trace with a tool node.
func NewTestToolTrace() *models.Trace {
	now := time.Now().UTC()
	return &models.Trace{
		ID:             TestTraceID + "-tool",
		TenantID:       TestTenantID,
		ApplicationID:  TestApplicationID,
		ConversationID: TestConversationID,
		ContextType:    models.TraceContextConversation,
		ReferenceID:    "workflow-execution-456",
		ReferenceName:  "Tool Workflow Run",
		Nodes: []models.TraceNode{
			{
				ID:       "node-tool-1",
				Name:     "search-tool",
				Type:     models.NodeTypeTool,
				Status:   models.NodeStatusCompleted,
				StartAt:  timePtr(now.Add(-50 * time.Millisecond)),
				EndAt:    timePtr(now),
				Duration: 0.05,
				Data: &models.NodeData{
					Input: &models.NodeDataIO{
						Text:      "test search query",
						ExtraData: map[string]interface{}{"query": "test search"},
					},
					Output: &models.NodeDataIO{
						Text:      "search results",
						ExtraData: map[string]interface{}{"results": []string{"result1", "result2"}},
					},
				},
				Metadata:  map[string]interface{}{"toolName": "search-tool"},
				CreatedAt: now,
				UpdatedAt: now,
			},
		},
		Logs:      []string{},
		CreatedAt: now,
		UpdatedAt: now,
		CreatedBy: TestUserID,
		UpdatedBy: TestUserID,
	}
}

// timePtr returns a pointer to the given time.
func timePtr(t time.Time) *time.Time {
	return &t
}

// NewTestSession creates a test session with default values.
func NewTestSession() *models.Session {
	return &models.Session{
		TenantID: TestTenantID,
		UserID:   TestUserID,
		Config: &models.SessionConfig{
			AgentID:   TestApplicationID,
			AgentType: "n8n",
			AgentName: "Test Agent",
			Endpoint:  "http://localhost:5678/webhook/test",
			Features: &models.AgentFeatures{
				SupportsStreaming:   true,
				SupportsTracing:     true,
				SupportsHumanInLoop: false,
			},
		},
		Credentials: &models.EncryptedCreds{
			EncryptedData: "encrypted-test-data",
			KeyVersion:    "v1",
		},
		CreatedAt: time.Now().UTC(),
		ExpiresAt: time.Now().UTC().Add(3 * time.Minute),
	}
}

// NewTestMessages creates a slice of test messages (alternating user/assistant).
func NewTestMessages(count int) []*models.Message {
	messages := make([]*models.Message, count)
	for i := 0; i < count; i++ {
		if i%2 == 0 {
			msg := NewTestUserMessage()
			msg.ID = TestMessageID + "-" + string(rune('0'+i))
			messages[i] = msg
		} else {
			msg := NewTestAssistantMessage()
			msg.ID = TestMessageID + "-" + string(rune('0'+i))
			msg.UserMessageID = TestMessageID + "-" + string(rune('0'+i-1))
			messages[i] = msg
		}
	}
	return messages
}

// NewTestTraces creates a slice of test traces.
func NewTestTraces(count int) []*models.Trace {
	traces := make([]*models.Trace, count)
	for i := 0; i < count; i++ {
		trace := NewTestTrace()
		trace.ID = TestTraceID + "-" + string(rune('0'+i))
		trace.ReferenceID = "workflow-" + string(rune('0'+i))
		traces[i] = trace
	}
	return traces
}

// NewTestAutonomousAgentTrace creates a test trace for autonomous agent context.
func NewTestAutonomousAgentTrace() *models.Trace {
	now := time.Now().UTC()
	return &models.Trace{
		ID:                TestTraceID + "-auto",
		TenantID:          TestTenantID,
		AutonomousAgentID: "auto-agent-123",
		ContextType:       models.TraceContextAutonomousAgent,
		ReferenceID:       "scheduled-run-123",
		ReferenceName:     "Scheduled Agent Run",
		Nodes: []models.TraceNode{
			{
				ID:        "node-auto-1",
				Name:      "agent-execution",
				Type:      models.NodeTypeAgent,
				Status:    models.NodeStatusCompleted,
				StartAt:   timePtr(now.Add(-200 * time.Millisecond)),
				EndAt:     timePtr(now),
				Duration:  0.2,
				CreatedAt: now,
				UpdatedAt: now,
			},
		},
		Logs:      []string{},
		CreatedAt: now,
		UpdatedAt: now,
		CreatedBy: TestUserID,
		UpdatedBy: TestUserID,
	}
}
