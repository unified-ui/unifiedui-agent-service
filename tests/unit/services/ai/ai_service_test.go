package ai_test

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/unifiedui/agent-service/internal/services/ai"
	"github.com/unifiedui/agent-service/internal/services/platform"
	"github.com/unifiedui/agent-service/tests/mocks"
)

func newOpenAITestServer(content string) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := map[string]interface{}{
			"choices": []map[string]interface{}{
				{"message": map[string]interface{}{"content": content}, "finish_reason": "stop"},
			},
			"model": "test-model",
			"usage": map[string]interface{}{"prompt_tokens": 10, "completion_tokens": 5},
		}
		json.NewEncoder(w).Encode(resp)
	}))
}

func mockModelsForServer(ts *httptest.Server) []platform.AIModelWithSecretResponse {
	return []platform.AIModelWithSecretResponse{
		{
			ID:       "model-1",
			Type:     "LLM_MODEL",
			Provider: "OPENAI",
			Config: map[string]interface{}{
				"model_name": "gpt-4",
				"base_url":   ts.URL,
			},
			CredentialSecret: map[string]interface{}{"api_key": "sk-test"},
			Priority:         1,
		},
	}
}

func TestAIService_GenerateTitle_Success(t *testing.T) {
	ts := newOpenAITestServer("Chat Title")
	defer ts.Close()

	mockPlatform := new(mocks.MockPlatformClient)
	mockPlatform.On("GetAIModelsByPurpose", mock.Anything, "tenant1", "CONVERSATION_TITLE_GENERATION", "LLM_MODEL").
		Return(mockModelsForServer(ts), nil)

	svc := ai.NewService(mockPlatform)
	title, err := svc.GenerateTitle(context.Background(), "tenant1", "Hello", "Hi there")
	require.NoError(t, err)
	assert.Equal(t, "Chat Title", title)
	mockPlatform.AssertExpectations(t)
}

func TestAIService_GenerateTitle_LongTitleTruncated(t *testing.T) {
	longTitle := strings.Repeat("A", 100)
	ts := newOpenAITestServer(longTitle)
	defer ts.Close()

	mockPlatform := new(mocks.MockPlatformClient)
	mockPlatform.On("GetAIModelsByPurpose", mock.Anything, "tenant1", "CONVERSATION_TITLE_GENERATION", "LLM_MODEL").
		Return(mockModelsForServer(ts), nil)

	svc := ai.NewService(mockPlatform)
	title, err := svc.GenerateTitle(context.Background(), "tenant1", "Hello", "Hi")
	require.NoError(t, err)
	assert.Len(t, title, 50)
}

func TestAIService_GenerateTitle_NoModels(t *testing.T) {
	mockPlatform := new(mocks.MockPlatformClient)
	mockPlatform.On("GetAIModelsByPurpose", mock.Anything, "tenant1", "CONVERSATION_TITLE_GENERATION", "LLM_MODEL").
		Return([]platform.AIModelWithSecretResponse{}, nil)

	svc := ai.NewService(mockPlatform)
	_, err := svc.GenerateTitle(context.Background(), "tenant1", "Hello", "Hi")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no active AI models")
}

func TestAIService_GenerateTitle_PlatformError(t *testing.T) {
	mockPlatform := new(mocks.MockPlatformClient)
	mockPlatform.On("GetAIModelsByPurpose", mock.Anything, "tenant1", "CONVERSATION_TITLE_GENERATION", "LLM_MODEL").
		Return(nil, fmt.Errorf("connection refused"))

	svc := ai.NewService(mockPlatform)
	_, err := svc.GenerateTitle(context.Background(), "tenant1", "Hello", "Hi")
	assert.Error(t, err)
}

func TestAIService_GenerateDescription_Success(t *testing.T) {
	ts := newOpenAITestServer("Generated description")
	defer ts.Close()

	mockPlatform := new(mocks.MockPlatformClient)
	mockPlatform.On("GetAIModelsByPurpose", mock.Anything, "tenant1", "DESCRIPTION_GENERATION", "LLM_MODEL").
		Return(mockModelsForServer(ts), nil)

	svc := ai.NewService(mockPlatform)
	desc, err := svc.GenerateDescription(context.Background(), "tenant1", "chat_agent", "My Agent", "old desc", map[string]interface{}{"key": "val"})
	require.NoError(t, err)
	assert.Equal(t, "Generated description", desc)
}

func TestAIService_AnalyzeTrace_Success(t *testing.T) {
	ts := newOpenAITestServer("Error analysis result")
	defer ts.Close()

	mockPlatform := new(mocks.MockPlatformClient)
	mockPlatform.On("GetAIModelsByPurpose", mock.Anything, "tenant1", "TRACE_ANALYSIS", "LLM_MODEL").
		Return(mockModelsForServer(ts), nil)

	svc := ai.NewService(mockPlatform)
	result, err := svc.AnalyzeTrace(context.Background(), "tenant1", ai.AnalyzeTraceInput{
		TraceID:  "t1",
		NodeID:   "n1",
		Error:    "connection timeout",
		NodeName: "HTTP Request",
		NodeType: "httpRequest",
		Input:    map[string]interface{}{"url": "https://api.example.com"},
		Output:   map[string]interface{}{"error": "timeout"},
	})
	require.NoError(t, err)
	assert.Equal(t, "Error analysis result", result)
}

func TestAIService_AnalyzeTrace_NilInputOutput(t *testing.T) {
	ts := newOpenAITestServer("Analysis")
	defer ts.Close()

	mockPlatform := new(mocks.MockPlatformClient)
	mockPlatform.On("GetAIModelsByPurpose", mock.Anything, "tenant1", "TRACE_ANALYSIS", "LLM_MODEL").
		Return(mockModelsForServer(ts), nil)

	svc := ai.NewService(mockPlatform)
	result, err := svc.AnalyzeTrace(context.Background(), "tenant1", ai.AnalyzeTraceInput{
		TraceID:  "t1",
		NodeID:   "n1",
		Error:    "error",
		NodeName: "Node",
		NodeType: "type",
	})
	require.NoError(t, err)
	assert.Equal(t, "Analysis", result)
}

func TestAIService_SummarizeTrace_Success(t *testing.T) {
	ts := newOpenAITestServer("Trace summary")
	defer ts.Close()

	mockPlatform := new(mocks.MockPlatformClient)
	mockPlatform.On("GetAIModelsByPurpose", mock.Anything, "tenant1", "TRACE_ANALYSIS", "LLM_MODEL").
		Return(mockModelsForServer(ts), nil)

	svc := ai.NewService(mockPlatform)
	result, err := svc.SummarizeTrace(context.Background(), "tenant1", ai.SummarizeTraceInput{
		TraceID:     "t1",
		DetailLevel: "high",
		Nodes:       []map[string]interface{}{{"name": "Node1", "type": "http"}},
	})
	require.NoError(t, err)
	assert.Equal(t, "Trace summary", result)
}

func TestAIService_TraceChat_Success(t *testing.T) {
	ts := newOpenAITestServer("Chat response about trace")
	defer ts.Close()

	mockPlatform := new(mocks.MockPlatformClient)
	mockPlatform.On("GetAIModelsByPurpose", mock.Anything, "tenant1", "TRACE_ANALYSIS", "LLM_MODEL").
		Return(mockModelsForServer(ts), nil)

	svc := ai.NewService(mockPlatform)
	result, err := svc.TraceChat(context.Background(), "tenant1", ai.TraceChatInput{
		Trace:        "trace data",
		SelectedNode: "node1",
		Message:      "What went wrong?",
		History:      []ai.ChatMessage{{Role: "user", Content: "Previous question"}},
	})
	require.NoError(t, err)
	assert.Equal(t, "Chat response about trace", result)
}

func TestAIService_TraceChat_EmptyContent(t *testing.T) {
	ts := newOpenAITestServer("")
	defer ts.Close()

	mockPlatform := new(mocks.MockPlatformClient)
	mockPlatform.On("GetAIModelsByPurpose", mock.Anything, "tenant1", "TRACE_ANALYSIS", "LLM_MODEL").
		Return(mockModelsForServer(ts), nil)

	svc := ai.NewService(mockPlatform)
	result, err := svc.TraceChat(context.Background(), "tenant1", ai.TraceChatInput{
		Trace:   "trace data",
		Message: "What went wrong?",
	})
	require.NoError(t, err)
	assert.Contains(t, result, "content filter")
}

func TestAIService_TestModel_Success(t *testing.T) {
	ts := newOpenAITestServer("pong")
	defer ts.Close()

	mockPlatform := new(mocks.MockPlatformClient)
	svc := ai.NewService(mockPlatform)

	config := map[string]interface{}{
		"model_name": "gpt-4",
		"base_url":   ts.URL,
	}
	result, err := svc.TestModel(context.Background(), "OPENAI", config, map[string]interface{}{"api_key": "sk-test"})
	require.NoError(t, err)
	assert.True(t, result.Success)
	assert.Contains(t, result.Message, "successfully")
	assert.Greater(t, result.ResponseTimeMs, int64(-1))
}

func TestAIService_TestModel_InvalidProvider(t *testing.T) {
	mockPlatform := new(mocks.MockPlatformClient)
	svc := ai.NewService(mockPlatform)

	result, err := svc.TestModel(context.Background(), "INVALID", map[string]interface{}{}, nil)
	require.NoError(t, err)
	assert.False(t, result.Success)
	assert.Contains(t, result.Message, "Failed to create LLM client")
}

func TestAIService_TestModel_LLMError(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(500)
		w.Write([]byte("error"))
	}))
	defer ts.Close()

	mockPlatform := new(mocks.MockPlatformClient)
	svc := ai.NewService(mockPlatform)

	config := map[string]interface{}{
		"model_name": "gpt-4",
		"base_url":   ts.URL,
	}
	result, err := svc.TestModel(context.Background(), "OPENAI", config, nil)
	require.NoError(t, err)
	assert.False(t, result.Success)
	assert.Contains(t, result.Message, "Model test failed")
}

func TestAIService_GetCapabilities_AllEnabled(t *testing.T) {
	ts := newOpenAITestServer("OK")
	defer ts.Close()

	mockPlatform := new(mocks.MockPlatformClient)
	models := mockModelsForServer(ts)
	mockPlatform.On("GetAIModelsByPurpose", mock.Anything, "tenant1", "CONVERSATION_TITLE_GENERATION", "LLM_MODEL").Return(models, nil)
	mockPlatform.On("GetAIModelsByPurpose", mock.Anything, "tenant1", "DESCRIPTION_GENERATION", "LLM_MODEL").Return(models, nil)
	mockPlatform.On("GetAIModelsByPurpose", mock.Anything, "tenant1", "TRACE_ANALYSIS", "LLM_MODEL").Return(models, nil)
	mockPlatform.On("GetAIModelsByPurpose", mock.Anything, "tenant1", "CONVERSATION_SUMMARIZATION", "LLM_MODEL").Return(models, nil)

	svc := ai.NewService(mockPlatform)
	caps, err := svc.GetCapabilities(context.Background(), "tenant1")
	require.NoError(t, err)
	assert.True(t, caps.TitleGeneration)
	assert.True(t, caps.DescriptionGeneration)
	assert.True(t, caps.TraceAnalysis)
	assert.True(t, caps.Summarization)
}

func TestAIService_GetCapabilities_NoneEnabled(t *testing.T) {
	mockPlatform := new(mocks.MockPlatformClient)
	empty := []platform.AIModelWithSecretResponse{}
	mockPlatform.On("GetAIModelsByPurpose", mock.Anything, "tenant1", mock.Anything, "LLM_MODEL").Return(empty, nil)

	svc := ai.NewService(mockPlatform)
	caps, err := svc.GetCapabilities(context.Background(), "tenant1")
	require.NoError(t, err)
	assert.False(t, caps.TitleGeneration)
	assert.False(t, caps.DescriptionGeneration)
	assert.False(t, caps.TraceAnalysis)
	assert.False(t, caps.Summarization)
}

func TestAIService_GetCapabilities_ErrorHandling(t *testing.T) {
	mockPlatform := new(mocks.MockPlatformClient)
	mockPlatform.On("GetAIModelsByPurpose", mock.Anything, "tenant1", mock.Anything, "LLM_MODEL").
		Return(nil, fmt.Errorf("service unavailable"))

	svc := ai.NewService(mockPlatform)
	caps, err := svc.GetCapabilities(context.Background(), "tenant1")
	require.NoError(t, err)
	assert.False(t, caps.TitleGeneration)
}
