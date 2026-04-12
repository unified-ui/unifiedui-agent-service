package dto

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/unifiedui/agent-service/internal/api/dto"

	"github.com/stretchr/testify/require"
)

// ============================================
// AI DTO Tests (ai.go)
// ============================================

func TestGenerateDescriptionRequest_JSONSerialization(t *testing.T) {
	req := dto.GenerateDescriptionRequest{
		EntityType:          "agent",
		EntityName:          "CustomerSupport Bot",
		ExistingDescription: "A bot for customer support",
		Context: map[string]interface{}{
			"industry": "retail",
			"features": []string{"chat", "email"},
		},
	}

	data, err := json.Marshal(req)
	require.NoError(t, err)

	var parsed dto.GenerateDescriptionRequest
	require.NoError(t, json.Unmarshal(data, &parsed))
	require.Equal(t, req.EntityType, parsed.EntityType)
	require.Equal(t, req.EntityName, parsed.EntityName)
	require.Equal(t, req.ExistingDescription, parsed.ExistingDescription)
	require.NotNil(t, parsed.Context)
}

func TestGenerateDescriptionRequest_MinimalFields(t *testing.T) {
	jsonStr := `{"entity_type":"workflow","entity_name":"MyWorkflow"}`
	var req dto.GenerateDescriptionRequest
	require.NoError(t, json.Unmarshal([]byte(jsonStr), &req))
	require.Equal(t, "workflow", req.EntityType)
	require.Equal(t, "MyWorkflow", req.EntityName)
	require.Empty(t, req.ExistingDescription)
	require.Nil(t, req.Context)
}

func TestGenerateDescriptionResponse_JSONSerialization(t *testing.T) {
	resp := dto.GenerateDescriptionResponse{
		Description: "This is an auto-generated description for the agent.",
	}

	data, err := json.Marshal(resp)
	require.NoError(t, err)

	var parsed dto.GenerateDescriptionResponse
	require.NoError(t, json.Unmarshal(data, &parsed))
	require.Equal(t, resp.Description, parsed.Description)
}

func TestAnalyzeTraceRequest_JSONSerialization(t *testing.T) {
	req := dto.AnalyzeTraceRequest{
		TraceID:  "trace-123",
		NodeID:   "node-456",
		Error:    "Connection timeout",
		NodeName: "API Call Node",
		NodeType: "http",
		Input: map[string]interface{}{
			"url": "https://api.example.com",
		},
		Output: map[string]interface{}{
			"error": "timeout after 30s",
		},
	}

	data, err := json.Marshal(req)
	require.NoError(t, err)

	var parsed dto.AnalyzeTraceRequest
	require.NoError(t, json.Unmarshal(data, &parsed))
	require.Equal(t, req.TraceID, parsed.TraceID)
	require.Equal(t, req.NodeID, parsed.NodeID)
	require.Equal(t, req.Error, parsed.Error)
	require.Equal(t, req.NodeName, parsed.NodeName)
	require.Equal(t, req.NodeType, parsed.NodeType)
	require.NotNil(t, parsed.Input)
	require.NotNil(t, parsed.Output)
}

func TestAnalyzeTraceResponse_JSONSerialization(t *testing.T) {
	resp := dto.AnalyzeTraceResponse{
		Analysis: "The error indicates a network timeout. Consider increasing retry count.",
	}

	data, err := json.Marshal(resp)
	require.NoError(t, err)

	var parsed dto.AnalyzeTraceResponse
	require.NoError(t, json.Unmarshal(data, &parsed))
	require.Equal(t, resp.Analysis, parsed.Analysis)
}

func TestSummarizeTraceRequest_JSONSerialization(t *testing.T) {
	req := dto.SummarizeTraceRequest{
		TraceID:     "trace-789",
		DetailLevel: "medium",
		Nodes: []map[string]interface{}{
			{"id": "n1", "name": "Start", "type": "trigger"},
			{"id": "n2", "name": "Process", "type": "llm"},
			{"id": "n3", "name": "End", "type": "response"},
		},
	}

	data, err := json.Marshal(req)
	require.NoError(t, err)

	var parsed dto.SummarizeTraceRequest
	require.NoError(t, json.Unmarshal(data, &parsed))
	require.Equal(t, req.TraceID, parsed.TraceID)
	require.Equal(t, req.DetailLevel, parsed.DetailLevel)
	require.Len(t, parsed.Nodes, 3)
}

func TestSummarizeTraceRequest_DetailLevels(t *testing.T) {
	tests := []string{"short", "medium", "long"}
	for _, level := range tests {
		t.Run(level, func(t *testing.T) {
			jsonStr := `{"trace_id":"t1","detail_level":"` + level + `","nodes":[{}]}`
			var req dto.SummarizeTraceRequest
			require.NoError(t, json.Unmarshal([]byte(jsonStr), &req))
			require.Equal(t, level, req.DetailLevel)
		})
	}
}

func TestSummarizeTraceResponse_JSONSerialization(t *testing.T) {
	resp := dto.SummarizeTraceResponse{
		Summary: "The trace processed 3 nodes successfully in 2.5 seconds.",
	}

	data, err := json.Marshal(resp)
	require.NoError(t, err)

	var parsed dto.SummarizeTraceResponse
	require.NoError(t, json.Unmarshal(data, &parsed))
	require.Equal(t, resp.Summary, parsed.Summary)
}

func TestTestModelRequest_JSONSerialization(t *testing.T) {
	req := dto.TestModelRequest{
		Provider: "openai",
		Config: map[string]interface{}{
			"model":       "gpt-4",
			"temperature": 0.7,
			"max_tokens":  1000,
		},
		CredentialID: "cred-abc",
	}

	data, err := json.Marshal(req)
	require.NoError(t, err)

	var parsed dto.TestModelRequest
	require.NoError(t, json.Unmarshal(data, &parsed))
	require.Equal(t, req.Provider, parsed.Provider)
	require.NotNil(t, parsed.Config)
	require.Equal(t, req.CredentialID, parsed.CredentialID)
}

func TestTestModelResponse_JSONSerialization(t *testing.T) {
	resp := dto.TestModelResponse{
		Success:        true,
		Message:        "Model responded successfully",
		ResponseTimeMs: 245,
	}

	data, err := json.Marshal(resp)
	require.NoError(t, err)

	var parsed dto.TestModelResponse
	require.NoError(t, json.Unmarshal(data, &parsed))
	require.True(t, parsed.Success)
	require.Equal(t, resp.Message, parsed.Message)
	require.Equal(t, resp.ResponseTimeMs, parsed.ResponseTimeMs)
}

func TestTestModelResponse_Failure(t *testing.T) {
	jsonStr := `{"success":false,"message":"Invalid API key","response_time_ms":50}`
	var resp dto.TestModelResponse
	require.NoError(t, json.Unmarshal([]byte(jsonStr), &resp))
	require.False(t, resp.Success)
	require.Equal(t, "Invalid API key", resp.Message)
}

func TestTraceChatMessage_JSONSerialization(t *testing.T) {
	tests := []struct {
		name    string
		role    string
		content string
	}{
		{"user message", "user", "What does this node do?"},
		{"assistant message", "assistant", "This node processes incoming data."},
		{"empty content", "user", ""},
		{"long content", "assistant", string(make([]byte, 10000))},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			msg := dto.TraceChatMessage{
				Role:    tt.role,
				Content: tt.content,
			}

			data, err := json.Marshal(msg)
			require.NoError(t, err)

			var parsed dto.TraceChatMessage
			require.NoError(t, json.Unmarshal(data, &parsed))
			require.Equal(t, tt.role, parsed.Role)
			require.Equal(t, tt.content, parsed.Content)
		})
	}
}

func TestTraceChatRequest_JSONSerialization(t *testing.T) {
	req := dto.TraceChatRequest{
		Trace:        `{"id":"t1","nodes":[]}`,
		SelectedNode: "node-1",
		Message:      "Explain this error",
		History: []dto.TraceChatMessage{
			{Role: "user", Content: "What happened?"},
			{Role: "assistant", Content: "There was an error."},
		},
	}

	data, err := json.Marshal(req)
	require.NoError(t, err)

	var parsed dto.TraceChatRequest
	require.NoError(t, json.Unmarshal(data, &parsed))
	require.Equal(t, req.Trace, parsed.Trace)
	require.Equal(t, req.SelectedNode, parsed.SelectedNode)
	require.Equal(t, req.Message, parsed.Message)
	require.Len(t, parsed.History, 2)
}

func TestTraceChatRequest_EmptyHistory(t *testing.T) {
	jsonStr := `{"trace":"{}","message":"Hi","history":[]}`
	var req dto.TraceChatRequest
	require.NoError(t, json.Unmarshal([]byte(jsonStr), &req))
	require.Empty(t, req.History)
}

func TestTraceChatResponse_JSONSerialization(t *testing.T) {
	resp := dto.TraceChatResponse{
		Reply: "Based on the trace, the error occurred in the HTTP node.",
	}

	data, err := json.Marshal(resp)
	require.NoError(t, err)

	var parsed dto.TraceChatResponse
	require.NoError(t, json.Unmarshal(data, &parsed))
	require.Equal(t, resp.Reply, parsed.Reply)
}

func TestAICapabilitiesResponse_JSONSerialization(t *testing.T) {
	resp := dto.AICapabilitiesResponse{
		TitleGeneration:       true,
		DescriptionGeneration: true,
		TraceAnalysis:         false,
		Summarization:         true,
	}

	data, err := json.Marshal(resp)
	require.NoError(t, err)

	var parsed dto.AICapabilitiesResponse
	require.NoError(t, json.Unmarshal(data, &parsed))
	require.True(t, parsed.TitleGeneration)
	require.True(t, parsed.DescriptionGeneration)
	require.False(t, parsed.TraceAnalysis)
	require.True(t, parsed.Summarization)
}

func TestAICapabilitiesResponse_AllDisabled(t *testing.T) {
	jsonStr := `{"title_generation":false,"description_generation":false,"trace_analysis":false,"summarization":false}`
	var resp dto.AICapabilitiesResponse
	require.NoError(t, json.Unmarshal([]byte(jsonStr), &resp))
	require.False(t, resp.TitleGeneration)
	require.False(t, resp.DescriptionGeneration)
	require.False(t, resp.TraceAnalysis)
	require.False(t, resp.Summarization)
}

// ============================================
// Request DTO Tests (requests.go)
// ============================================

func TestSendMessageRequest_JSONSerialization(t *testing.T) {
	req := dto.SendMessageRequest{
		Content: "Hello, how can I help?",
		AgentID: "agent-123",
		Stream:  true,
	}

	data, err := json.Marshal(req)
	require.NoError(t, err)

	var parsed dto.SendMessageRequest
	require.NoError(t, json.Unmarshal(data, &parsed))
	require.Equal(t, req.Content, parsed.Content)
	require.Equal(t, req.AgentID, parsed.AgentID)
	require.True(t, parsed.Stream)
}

func TestSendMessageRequest_StreamFalse(t *testing.T) {
	jsonStr := `{"content":"msg","agentId":"a1","stream":false}`
	var req dto.SendMessageRequest
	require.NoError(t, json.Unmarshal([]byte(jsonStr), &req))
	require.False(t, req.Stream)
}

func TestSendMessageRequest_LongContent(t *testing.T) {
	longContent := string(make([]byte, 32000))
	req := dto.SendMessageRequest{
		Content: longContent,
		AgentID: "agent-1",
		Stream:  true,
	}

	data, err := json.Marshal(req)
	require.NoError(t, err)

	var parsed dto.SendMessageRequest
	require.NoError(t, json.Unmarshal(data, &parsed))
	require.Len(t, parsed.Content, 32000)
}

func TestMessageResponse_JSONSerialization(t *testing.T) {
	now := time.Now().UTC()
	resp := dto.MessageResponse{
		ID:             "msg-001",
		ConversationID: "conv-001",
		Role:           "assistant",
		Content:        "Here is your answer.",
		AgentID:        "agent-001",
		UserID:         "user-001",
		CreatedAt:      now,
		Metadata: &dto.MetadataResponse{
			Model:        "gpt-4",
			TokensInput:  100,
			TokensOutput: 50,
			LatencyMs:    250,
			AgentType:    "chat",
			Custom: map[string]interface{}{
				"source": "api",
			},
		},
	}

	data, err := json.Marshal(resp)
	require.NoError(t, err)

	var parsed dto.MessageResponse
	require.NoError(t, json.Unmarshal(data, &parsed))
	require.Equal(t, resp.ID, parsed.ID)
	require.Equal(t, resp.ConversationID, parsed.ConversationID)
	require.Equal(t, resp.Role, parsed.Role)
	require.Equal(t, resp.Content, parsed.Content)
	require.NotNil(t, parsed.Metadata)
	require.Equal(t, "gpt-4", parsed.Metadata.Model)
}

func TestMessageResponse_NoMetadata(t *testing.T) {
	jsonStr := `{"id":"m1","conversationId":"c1","role":"user","content":"hi","createdAt":"2024-01-01T00:00:00Z"}`
	var resp dto.MessageResponse
	require.NoError(t, json.Unmarshal([]byte(jsonStr), &resp))
	require.Nil(t, resp.Metadata)
}

func TestMetadataResponse_JSONSerialization(t *testing.T) {
	meta := dto.MetadataResponse{
		Model:        "claude-3",
		TokensInput:  200,
		TokensOutput: 100,
		LatencyMs:    500,
		AgentType:    "assistant",
		Custom: map[string]interface{}{
			"key1": "value1",
			"key2": 42,
		},
	}

	data, err := json.Marshal(meta)
	require.NoError(t, err)

	var parsed dto.MetadataResponse
	require.NoError(t, json.Unmarshal(data, &parsed))
	require.Equal(t, meta.Model, parsed.Model)
	require.Equal(t, meta.TokensInput, parsed.TokensInput)
	require.Equal(t, meta.TokensOutput, parsed.TokensOutput)
	require.Equal(t, meta.LatencyMs, parsed.LatencyMs)
	require.NotNil(t, parsed.Custom)
}

func TestMetadataResponse_AllEmpty(t *testing.T) {
	jsonStr := `{}`
	var meta dto.MetadataResponse
	require.NoError(t, json.Unmarshal([]byte(jsonStr), &meta))
	require.Empty(t, meta.Model)
	require.Zero(t, meta.TokensInput)
	require.Nil(t, meta.Custom)
}

// ============================================
// Response DTO Tests (responses.go)
// ============================================

func TestErrorResponse_JSONSerialization(t *testing.T) {
	resp := dto.ErrorResponse{
		Code:    "VALIDATION_ERROR",
		Message: "Invalid input",
		Details: "Field 'name' is required",
	}

	data, err := json.Marshal(resp)
	require.NoError(t, err)

	var parsed dto.ErrorResponse
	require.NoError(t, json.Unmarshal(data, &parsed))
	require.Equal(t, resp.Code, parsed.Code)
	require.Equal(t, resp.Message, parsed.Message)
	require.Equal(t, resp.Details, parsed.Details)
}

func TestErrorResponse_NoDetails(t *testing.T) {
	jsonStr := `{"code":"NOT_FOUND","message":"Resource not found"}`
	var resp dto.ErrorResponse
	require.NoError(t, json.Unmarshal([]byte(jsonStr), &resp))
	require.Empty(t, resp.Details)
}

func TestHealthResponse_JSONSerialization(t *testing.T) {
	resp := dto.HealthResponse{
		Status: "healthy",
		Components: map[string]string{
			"database": "ok",
			"cache":    "ok",
			"vault":    "degraded",
		},
	}

	data, err := json.Marshal(resp)
	require.NoError(t, err)

	var parsed dto.HealthResponse
	require.NoError(t, json.Unmarshal(data, &parsed))
	require.Equal(t, "healthy", parsed.Status)
	require.Len(t, parsed.Components, 3)
	require.Equal(t, "degraded", parsed.Components["vault"])
}

func TestHealthResponse_NoComponents(t *testing.T) {
	jsonStr := `{"status":"healthy"}`
	var resp dto.HealthResponse
	require.NoError(t, json.Unmarshal([]byte(jsonStr), &resp))
	require.Nil(t, resp.Components)
}

func TestPaginatedResponse_JSONSerialization(t *testing.T) {
	resp := dto.PaginatedResponse{
		Data:   []string{"item1", "item2", "item3"},
		Total:  100,
		Limit:  10,
		Offset: 0,
	}

	data, err := json.Marshal(resp)
	require.NoError(t, err)

	var parsed dto.PaginatedResponse
	require.NoError(t, json.Unmarshal(data, &parsed))
	require.Equal(t, resp.Total, parsed.Total)
	require.Equal(t, resp.Limit, parsed.Limit)
	require.Equal(t, resp.Offset, parsed.Offset)
}

func TestPaginatedResponse_EmptyData(t *testing.T) {
	resp := dto.PaginatedResponse{
		Data:   []interface{}{},
		Total:  0,
		Limit:  10,
		Offset: 0,
	}

	data, err := json.Marshal(resp)
	require.NoError(t, err)

	var parsed dto.PaginatedResponse
	require.NoError(t, json.Unmarshal(data, &parsed))
	require.Zero(t, parsed.Total)
}

func TestGetMessagesResponse_JSONSerialization(t *testing.T) {
	now := time.Now().UTC()
	resp := dto.GetMessagesResponse{
		Messages: []*dto.MessageResponse{
			{ID: "m1", ConversationID: "c1", Role: "user", Content: "Hello", CreatedAt: now},
			{ID: "m2", ConversationID: "c1", Role: "assistant", Content: "Hi!", CreatedAt: now},
		},
		Total:  2,
		Limit:  10,
		Offset: 0,
	}

	data, err := json.Marshal(resp)
	require.NoError(t, err)

	var parsed dto.GetMessagesResponse
	require.NoError(t, json.Unmarshal(data, &parsed))
	require.Len(t, parsed.Messages, 2)
	require.Equal(t, int64(2), parsed.Total)
}

func TestGetMessagesResponse_Empty(t *testing.T) {
	jsonStr := `{"messages":[],"total":0,"limit":10,"offset":0}`
	var resp dto.GetMessagesResponse
	require.NoError(t, json.Unmarshal([]byte(jsonStr), &resp))
	require.Empty(t, resp.Messages)
}

func TestSendMessageResponse_JSONSerialization(t *testing.T) {
	now := time.Now().UTC()
	resp := dto.SendMessageResponse{
		Message: &dto.MessageResponse{
			ID:             "msg-new",
			ConversationID: "conv-1",
			Role:           "assistant",
			Content:        "Response content",
			CreatedAt:      now,
		},
	}

	data, err := json.Marshal(resp)
	require.NoError(t, err)

	var parsed dto.SendMessageResponse
	require.NoError(t, json.Unmarshal(data, &parsed))
	require.NotNil(t, parsed.Message)
	require.Equal(t, "msg-new", parsed.Message.ID)
}

func TestGetTracesResponse_JSONSerialization(t *testing.T) {
	now := time.Now().UTC()
	resp := dto.GetTracesResponse{
		Traces: []*dto.TraceResponse{
			{ID: "t1", TenantID: "ten-1", ContextType: "conversation", CreatedAt: now, UpdatedAt: now},
			{ID: "t2", TenantID: "ten-1", ContextType: "workflow", CreatedAt: now, UpdatedAt: now},
		},
		Total: 2,
	}

	data, err := json.Marshal(resp)
	require.NoError(t, err)

	var parsed dto.GetTracesResponse
	require.NoError(t, json.Unmarshal(data, &parsed))
	require.Len(t, parsed.Traces, 2)
	require.Equal(t, int64(2), parsed.Total)
}

func TestUpdateTracesResponse_JSONSerialization(t *testing.T) {
	resp := dto.UpdateTracesResponse{
		Updated: 5,
		Created: 3,
	}

	data, err := json.Marshal(resp)
	require.NoError(t, err)

	var parsed dto.UpdateTracesResponse
	require.NoError(t, json.Unmarshal(data, &parsed))
	require.Equal(t, 5, parsed.Updated)
	require.Equal(t, 3, parsed.Created)
}

// ============================================
// SSE Event DTO Tests (from responses.go)
// ============================================

func TestSSEMessageEvent_JSONSerialization(t *testing.T) {
	evt := dto.SSEMessageEvent{
		Content:   "Hello world",
		MessageID: "msg-123",
		Done:      false,
	}

	data, err := json.Marshal(evt)
	require.NoError(t, err)

	var parsed dto.SSEMessageEvent
	require.NoError(t, json.Unmarshal(data, &parsed))
	require.Equal(t, evt.Content, parsed.Content)
	require.Equal(t, evt.MessageID, parsed.MessageID)
	require.False(t, parsed.Done)
}

func TestSSEMessageEvent_Done(t *testing.T) {
	jsonStr := `{"content":"final","messageId":"m1","done":true}`
	var evt dto.SSEMessageEvent
	require.NoError(t, json.Unmarshal([]byte(jsonStr), &evt))
	require.True(t, evt.Done)
}

func TestSSETraceEvent_JSONSerialization(t *testing.T) {
	evt := dto.SSETraceEvent{
		TraceID: "trace-abc",
		Type:    "llm",
		Name:    "GPT-4 Call",
		Status:  "completed",
		Data: map[string]interface{}{
			"tokensUsed": float64(150),
		},
	}

	data, err := json.Marshal(evt)
	require.NoError(t, err)

	var parsed dto.SSETraceEvent
	require.NoError(t, json.Unmarshal(data, &parsed))
	require.Equal(t, evt.TraceID, parsed.TraceID)
	require.Equal(t, evt.Type, parsed.Type)
	require.Equal(t, evt.Name, parsed.Name)
	require.Equal(t, evt.Status, parsed.Status)
	require.NotNil(t, parsed.Data)
}

func TestSSETraceEvent_NoData(t *testing.T) {
	jsonStr := `{"traceId":"t1","type":"tool","name":"HTTP","status":"pending"}`
	var evt dto.SSETraceEvent
	require.NoError(t, json.Unmarshal([]byte(jsonStr), &evt))
	require.Nil(t, evt.Data)
}

func TestSSEErrorEvent_JSONSerialization(t *testing.T) {
	evt := dto.SSEErrorEvent{
		Code:    "INTERNAL_ERROR",
		Message: "Something went wrong",
		Details: "Stack trace here",
	}

	data, err := json.Marshal(evt)
	require.NoError(t, err)

	var parsed dto.SSEErrorEvent
	require.NoError(t, json.Unmarshal(data, &parsed))
	require.Equal(t, evt.Code, parsed.Code)
	require.Equal(t, evt.Message, parsed.Message)
	require.Equal(t, evt.Details, parsed.Details)
}

func TestSSEErrorEvent_NoDetails(t *testing.T) {
	jsonStr := `{"code":"TIMEOUT","message":"Request timed out"}`
	var evt dto.SSEErrorEvent
	require.NoError(t, json.Unmarshal([]byte(jsonStr), &evt))
	require.Empty(t, evt.Details)
}
