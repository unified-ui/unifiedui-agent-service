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
	TestAgentID        = "agent-test-abc"
	TestUserID         = "user-test-def"
	TestTraceID        = "trace-test-xyz"
)

// NewTestMessage creates a test message with default values.
func NewTestMessage() *models.Message {
	return &models.Message{
		ID:             TestMessageID,
		TenantID:       TestTenantID,
		ConversationID: TestConversationID,
		Role:           models.RoleUser,
		Content:        "Test message content",
		AgentID:        TestAgentID,
		UserID:         TestUserID,
		CreatedAt:      time.Now().UTC(),
		UpdatedAt:      time.Now().UTC(),
	}
}

// NewTestAssistantMessage creates a test assistant message.
func NewTestAssistantMessage() *models.Message {
	return &models.Message{
		ID:             TestMessageID + "-assistant",
		TenantID:       TestTenantID,
		ConversationID: TestConversationID,
		Role:           models.RoleAssistant,
		Content:        "Test assistant response",
		AgentID:        TestAgentID,
		CreatedAt:      time.Now().UTC(),
		UpdatedAt:      time.Now().UTC(),
	}
}

// NewTestTrace creates a test trace with default values.
func NewTestTrace() *models.Trace {
	return &models.Trace{
		ID:             TestTraceID,
		TenantID:       TestTenantID,
		ConversationID: TestConversationID,
		MessageID:      TestMessageID,
		AgentID:        TestAgentID,
		Type:           models.TraceTypeLLM,
		Name:           "test-llm-call",
		Status:         models.TraceStatusCompleted,
		Input:          "Test input",
		Output:         "Test output",
		StartedAt:      time.Now().UTC().Add(-100 * time.Millisecond),
		DurationMs:     100,
	}
}

// NewTestToolTrace creates a test tool trace.
func NewTestToolTrace() *models.Trace {
	now := time.Now().UTC()
	return &models.Trace{
		ID:             TestTraceID + "-tool",
		TenantID:       TestTenantID,
		ConversationID: TestConversationID,
		MessageID:      TestMessageID,
		AgentID:        TestAgentID,
		ParentTraceID:  TestTraceID,
		Type:           models.TraceTypeTool,
		Name:           "search-tool",
		Status:         models.TraceStatusCompleted,
		Input:          map[string]interface{}{"query": "test search"},
		Output:         map[string]interface{}{"results": []string{"result1", "result2"}},
		StartedAt:      now.Add(-50 * time.Millisecond),
		DurationMs:     50,
		Metadata: models.TraceMetadata{
			ToolName:   "search-tool",
			ToolInput:  map[string]interface{}{"query": "test search"},
			ToolOutput: map[string]interface{}{"results": []string{"result1", "result2"}},
		},
	}
}

// NewTestSession creates a test session with default values.
func NewTestSession() *models.Session {
	return &models.Session{
		TenantID: TestTenantID,
		UserID:   TestUserID,
		Config: &models.SessionConfig{
			AgentID:   TestAgentID,
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

// NewTestMessages creates a slice of test messages.
func NewTestMessages(count int) []*models.Message {
	messages := make([]*models.Message, count)
	for i := 0; i < count; i++ {
		msg := NewTestMessage()
		msg.ID = TestMessageID + "-" + string(rune('0'+i))
		if i%2 == 1 {
			msg.Role = models.RoleAssistant
		}
		messages[i] = msg
	}
	return messages
}

// NewTestTraces creates a slice of test traces.
func NewTestTraces(count int) []*models.Trace {
	traces := make([]*models.Trace, count)
	for i := 0; i < count; i++ {
		trace := NewTestTrace()
		trace.ID = TestTraceID + "-" + string(rune('0'+i))
		traces[i] = trace
	}
	return traces
}
