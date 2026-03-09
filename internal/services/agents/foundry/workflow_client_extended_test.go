package foundry

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// =============================================================================
// Helper Functions
// =============================================================================

// createSSEEvent creates a single SSE formatted event
func createSSEEvent(eventType string, data interface{}) string {
	var sb strings.Builder
	if eventType != "" {
		sb.WriteString("event: ")
		sb.WriteString(eventType)
		sb.WriteString("\n")
	}
	jsonData, _ := json.Marshal(data)
	sb.WriteString("data: ")
	sb.Write(jsonData)
	sb.WriteString("\n\n")
	return sb.String()
}

// createSSEDone creates a [DONE] SSE event
func createSSEDone() string {
	return "data: [DONE]\n\n"
}

// createTestServer creates a test HTTP server with the given handler
func createTestServer(handler http.HandlerFunc) *httptest.Server {
	return httptest.NewServer(handler)
}

// =============================================================================
// NewWorkflowClient Tests
// =============================================================================

func TestNewWorkflowClient_ValidConfig(t *testing.T) {
	config := &WorkflowClientConfig{
		ProjectEndpoint: "https://project.api.azure.com",
		APIVersion:      "2025-11-15-preview",
		AgentName:       "test-agent",
		AgentType:       "AGENT",
		APIToken:        "bearer-token-123",
	}

	client, err := NewWorkflowClient(config)

	require.NoError(t, err)
	require.NotNil(t, client)
	assert.Equal(t, "https://project.api.azure.com", client.projectEndpoint)
	assert.Equal(t, "2025-11-15-preview", client.apiVersion)
	assert.Equal(t, "test-agent", client.agentName)
	assert.Equal(t, "AGENT", client.agentType)
	assert.Equal(t, "bearer-token-123", client.apiToken)
	assert.NotNil(t, client.httpClient)
}

func TestNewWorkflowClient_NilConfig(t *testing.T) {
	client, err := NewWorkflowClient(nil)

	require.Error(t, err)
	assert.Nil(t, client)
	assert.Contains(t, err.Error(), "config is required")
}

func TestNewWorkflowClient_EmptyProjectEndpoint(t *testing.T) {
	config := &WorkflowClientConfig{
		ProjectEndpoint: "",
		AgentName:       "test-agent",
		APIToken:        "token",
	}

	client, err := NewWorkflowClient(config)

	require.Error(t, err)
	assert.Nil(t, client)
	assert.Contains(t, err.Error(), "project endpoint is required")
}

func TestNewWorkflowClient_EmptyAgentName(t *testing.T) {
	config := &WorkflowClientConfig{
		ProjectEndpoint: "https://project.api.azure.com",
		AgentName:       "",
		APIToken:        "token",
	}

	client, err := NewWorkflowClient(config)

	require.Error(t, err)
	assert.Nil(t, client)
	assert.Contains(t, err.Error(), "agent name is required")
}

func TestNewWorkflowClient_EmptyAPIToken(t *testing.T) {
	config := &WorkflowClientConfig{
		ProjectEndpoint: "https://project.api.azure.com",
		AgentName:       "test-agent",
		APIToken:        "",
	}

	client, err := NewWorkflowClient(config)

	require.Error(t, err)
	assert.Nil(t, client)
	assert.Contains(t, err.Error(), "API token is required")
}

func TestNewWorkflowClient_DefaultAPIVersion(t *testing.T) {
	config := &WorkflowClientConfig{
		ProjectEndpoint: "https://project.api.azure.com",
		AgentName:       "test-agent",
		APIToken:        "token",
	}

	client, err := NewWorkflowClient(config)

	require.NoError(t, err)
	require.NotNil(t, client)
	assert.Equal(t, "2025-11-15-preview", client.apiVersion)
}

func TestNewWorkflowClient_TrimsTrailingSlash(t *testing.T) {
	config := &WorkflowClientConfig{
		ProjectEndpoint: "https://project.api.azure.com/",
		AgentName:       "test-agent",
		APIToken:        "token",
	}

	client, err := NewWorkflowClient(config)

	require.NoError(t, err)
	require.NotNil(t, client)
	assert.Equal(t, "https://project.api.azure.com", client.projectEndpoint)
}

func TestNewWorkflowClient_MultiAgent(t *testing.T) {
	config := &WorkflowClientConfig{
		ProjectEndpoint: "https://project.api.azure.com",
		AgentName:       "multi-agent",
		AgentType:       "MULTI_AGENT",
		APIToken:        "test-token",
	}

	client, err := NewWorkflowClient(config)

	require.NoError(t, err)
	require.NotNil(t, client)
	assert.Equal(t, "MULTI_AGENT", client.agentType)
}

// =============================================================================
// Invoke Tests - Valid Responses
// =============================================================================

func TestInvoke_ValidResponse_SingleChunk(t *testing.T) {
	events := []Event{
		{Type: EventOutputTextDelta, Delta: "Hello from Foundry!"},
		{Type: EventResponseCompleted, Response: &Response{
			ID:     "resp-123",
			Status: "completed",
		}},
	}

	server := createTestServer(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)
		for _, event := range events {
			w.Write([]byte(createSSEEvent("", event)))
		}
		w.Write([]byte(createSSEDone()))
	})
	defer server.Close()

	config := &WorkflowClientConfig{
		ProjectEndpoint: server.URL,
		AgentName:       "test-agent",
		APIToken:        "test-token",
	}

	client, err := NewWorkflowClient(config)
	require.NoError(t, err)
	client.httpClient = server.Client()

	req := &InvokeRequest{
		ExtConversationID: "conv-123",
		Message:           "Hello",
		AgentName:         "test-agent",
	}

	resp, err := client.Invoke(context.Background(), req)

	require.NoError(t, err)
	require.NotNil(t, resp)
	assert.Equal(t, "Hello from Foundry!", resp.Content)
	assert.Equal(t, "conv-123", resp.SessionID)
}

func TestInvoke_ValidResponse_MultipleChunks(t *testing.T) {
	events := []Event{
		{Type: EventOutputTextDelta, Delta: "Hello, "},
		{Type: EventOutputTextDelta, Delta: "I am a "},
		{Type: EventOutputTextDelta, Delta: "Foundry agent!"},
		{Type: EventResponseCompleted, Response: &Response{
			ID:     "resp-456",
			Status: "completed",
		}},
	}

	server := createTestServer(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)
		for _, event := range events {
			w.Write([]byte(createSSEEvent("", event)))
		}
		w.Write([]byte(createSSEDone()))
	})
	defer server.Close()

	config := &WorkflowClientConfig{
		ProjectEndpoint: server.URL,
		AgentName:       "test-agent",
		APIToken:        "test-token",
	}

	client, err := NewWorkflowClient(config)
	require.NoError(t, err)
	client.httpClient = server.Client()

	req := &InvokeRequest{
		ExtConversationID: "conv-456",
		Message:           "Hello",
	}

	resp, err := client.Invoke(context.Background(), req)

	require.NoError(t, err)
	require.NotNil(t, resp)
	assert.Equal(t, "Hello, I am a Foundry agent!", resp.Content)
}

func TestInvoke_WithMetadata(t *testing.T) {
	events := []Event{
		{Type: EventOutputTextDelta, Delta: "Response"},
		{Type: EventResponseCompleted, Response: &Response{
			ID:     "resp-meta",
			Status: "completed",
			Usage: &Usage{
				InputTokens:  100,
				OutputTokens: 50,
				TotalTokens:  150,
			},
			Agent: &AgentRef{
				Name: "test-agent",
			},
			Conversation: &Conversation{
				ID: "conv-meta",
			},
		}},
	}

	server := createTestServer(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)
		for _, event := range events {
			w.Write([]byte(createSSEEvent("", event)))
		}
		w.Write([]byte(createSSEDone()))
	})
	defer server.Close()

	config := &WorkflowClientConfig{
		ProjectEndpoint: server.URL,
		AgentName:       "test-agent",
		APIToken:        "test-token",
	}

	client, err := NewWorkflowClient(config)
	require.NoError(t, err)
	client.httpClient = server.Client()

	req := &InvokeRequest{
		ExtConversationID: "conv-meta",
		Message:           "Test",
	}

	resp, err := client.Invoke(context.Background(), req)

	require.NoError(t, err)
	require.NotNil(t, resp)
	assert.NotNil(t, resp.Metadata)
	assert.Equal(t, "resp-meta", resp.ExecutionID)
}

// =============================================================================
// InvokeStream Tests - SSE Responses
// =============================================================================

func TestInvokeStream_ValidSSEResponse(t *testing.T) {
	events := []Event{
		{Type: EventOutputTextDelta, Delta: "First "},
		{Type: EventOutputTextDelta, Delta: "Second "},
		{Type: EventOutputTextDelta, Delta: "Third"},
		{Type: EventResponseCompleted, Response: &Response{
			ID:     "resp-stream",
			Status: "completed",
		}},
	}

	server := createTestServer(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)
		for _, event := range events {
			w.Write([]byte(createSSEEvent("", event)))
		}
		w.Write([]byte(createSSEDone()))
	})
	defer server.Close()

	config := &WorkflowClientConfig{
		ProjectEndpoint: server.URL,
		AgentName:       "test-agent",
		APIToken:        "test-token",
	}

	client, err := NewWorkflowClient(config)
	require.NoError(t, err)
	client.httpClient = server.Client()

	req := &InvokeRequest{
		ExtConversationID: "conv-stream",
		Message:           "Stream test",
	}

	ch, err := client.InvokeStream(context.Background(), req)

	require.NoError(t, err)
	require.NotNil(t, ch)

	var chunks []*StreamChunk //nolint:prealloc // length unknown: reading from channel
	for chunk := range ch {
		chunks = append(chunks, chunk)
	}

	// Should have 3 content chunks + 1 done chunk
	require.Len(t, chunks, 4)
	assert.Equal(t, ChunkTypeContent, chunks[0].Type)
	assert.Equal(t, "First ", chunks[0].Content)
	assert.Equal(t, ChunkTypeContent, chunks[1].Type)
	assert.Equal(t, "Second ", chunks[1].Content)
	assert.Equal(t, ChunkTypeContent, chunks[2].Type)
	assert.Equal(t, "Third", chunks[2].Content)
	assert.Equal(t, ChunkTypeDone, chunks[3].Type)
}

func TestInvokeStream_WithNewMessage(t *testing.T) {
	events := []Event{
		{
			Type: EventOutputItemAdded,
			Item: &OutputItem{
				Type: "message",
				ID:   "msg-1",
				Role: "assistant",
			},
		},
		{Type: EventOutputTextDelta, Delta: "First message"},
		{
			Type: EventOutputItemAdded,
			Item: &OutputItem{
				Type: "message",
				ID:   "msg-2",
				Role: "assistant",
			},
		},
		{Type: EventOutputTextDelta, Delta: "Second message"},
		{Type: EventResponseCompleted, Response: &Response{
			ID:     "resp-multi",
			Status: "completed",
		}},
	}

	server := createTestServer(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)
		for _, event := range events {
			w.Write([]byte(createSSEEvent("", event)))
		}
		w.Write([]byte(createSSEDone()))
	})
	defer server.Close()

	config := &WorkflowClientConfig{
		ProjectEndpoint: server.URL,
		AgentName:       "test-agent",
		APIToken:        "test-token",
	}

	client, err := NewWorkflowClient(config)
	require.NoError(t, err)
	client.httpClient = server.Client()

	req := &InvokeRequest{
		ExtConversationID: "conv-multi",
		Message:           "Test",
	}

	ch, err := client.InvokeStream(context.Background(), req)
	require.NoError(t, err)

	var chunks []*StreamChunk //nolint:prealloc // length unknown: reading from channel
	for chunk := range ch {
		chunks = append(chunks, chunk)
	}

	// Should contain new message chunk
	hasNewMessage := false
	for _, chunk := range chunks {
		if chunk.Type == ChunkTypeNewMessage {
			hasNewMessage = true
			break
		}
	}
	assert.True(t, hasNewMessage, "Expected a new message chunk")
}

func TestInvokeStream_ContextCancellation(t *testing.T) {
	server := createTestServer(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)
		// Simulate slow response
		time.Sleep(100 * time.Millisecond)
		event := Event{Type: EventOutputTextDelta, Delta: "Should not arrive"}
		w.Write([]byte(createSSEEvent("", event)))
	})
	defer server.Close()

	config := &WorkflowClientConfig{
		ProjectEndpoint: server.URL,
		AgentName:       "test-agent",
		APIToken:        "test-token",
	}

	client, err := NewWorkflowClient(config)
	require.NoError(t, err)
	client.httpClient = server.Client()

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	req := &InvokeRequest{
		ExtConversationID: "conv-ctx",
		Message:           "Test",
	}

	ch, err := client.InvokeStream(ctx, req)
	// Either error immediately or channel should close
	if err == nil {
		for chunk := range ch {
			if chunk.Type == ChunkTypeError {
				assert.NotNil(t, chunk.Error)
			}
		}
	}
}

// =============================================================================
// Error Handling Tests
// =============================================================================

func TestInvoke_NetworkError(t *testing.T) {
	config := &WorkflowClientConfig{
		ProjectEndpoint: "http://invalid.local.host:12345",
		AgentName:       "test-agent",
		APIToken:        "test-token",
	}

	client, err := NewWorkflowClient(config)
	require.NoError(t, err)
	client.httpClient = &http.Client{Timeout: 1 * time.Second}

	req := &InvokeRequest{
		ExtConversationID: "conv-err",
		Message:           "Test",
	}

	resp, err := client.Invoke(context.Background(), req)

	require.Error(t, err)
	assert.Nil(t, resp)
	assert.Contains(t, err.Error(), "failed to send request")
}

func TestInvoke_HTTPError_Unauthorized(t *testing.T) {
	server := createTestServer(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte(`{"error":{"code":"unauthorized","message":"Invalid token"}}`))
	})
	defer server.Close()

	config := &WorkflowClientConfig{
		ProjectEndpoint: server.URL,
		AgentName:       "test-agent",
		APIToken:        "invalid-token",
	}

	client, err := NewWorkflowClient(config)
	require.NoError(t, err)
	client.httpClient = server.Client()

	req := &InvokeRequest{
		ExtConversationID: "conv-auth",
		Message:           "Test",
	}

	resp, err := client.Invoke(context.Background(), req)

	require.Error(t, err)
	assert.Nil(t, resp)
	assert.Contains(t, err.Error(), "foundry API error: status=401")
}

func TestInvoke_HTTPError_InternalServerError(t *testing.T) {
	server := createTestServer(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(`{"error":{"code":"internal_error","message":"Server error"}}`))
	})
	defer server.Close()

	config := &WorkflowClientConfig{
		ProjectEndpoint: server.URL,
		AgentName:       "test-agent",
		APIToken:        "test-token",
	}

	client, err := NewWorkflowClient(config)
	require.NoError(t, err)
	client.httpClient = server.Client()

	req := &InvokeRequest{
		ExtConversationID: "conv-500",
		Message:           "Test",
	}

	resp, err := client.Invoke(context.Background(), req)

	require.Error(t, err)
	assert.Nil(t, resp)
	assert.Contains(t, err.Error(), "foundry API error: status=500")
}

func TestInvoke_HTTPError_BadRequest(t *testing.T) {
	server := createTestServer(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(`{"error":{"code":"bad_request","message":"Invalid request"}}`))
	})
	defer server.Close()

	config := &WorkflowClientConfig{
		ProjectEndpoint: server.URL,
		AgentName:       "test-agent",
		APIToken:        "test-token",
	}

	client, err := NewWorkflowClient(config)
	require.NoError(t, err)
	client.httpClient = server.Client()

	req := &InvokeRequest{
		ExtConversationID: "conv-400",
		Message:           "Test",
	}

	resp, err := client.Invoke(context.Background(), req)

	require.Error(t, err)
	assert.Nil(t, resp)
	assert.Contains(t, err.Error(), "foundry API error: status=400")
}

func TestInvokeStream_InvalidJSON(t *testing.T) {
	server := createTestServer(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)
		// Invalid JSON followed by valid event
		w.Write([]byte("data: invalid json\n\n"))
		event := Event{Type: EventOutputTextDelta, Delta: "valid content"}
		w.Write([]byte(createSSEEvent("", event)))
		w.Write([]byte(createSSEDone()))
	})
	defer server.Close()

	config := &WorkflowClientConfig{
		ProjectEndpoint: server.URL,
		AgentName:       "test-agent",
		APIToken:        "test-token",
	}

	client, err := NewWorkflowClient(config)
	require.NoError(t, err)
	client.httpClient = server.Client()

	req := &InvokeRequest{
		ExtConversationID: "conv-json",
		Message:           "Test",
	}

	ch, err := client.InvokeStream(context.Background(), req)
	require.NoError(t, err)

	var chunks []*StreamChunk //nolint:prealloc // length unknown: reading from channel
	for chunk := range ch {
		chunks = append(chunks, chunk)
	}

	// Should skip invalid JSON and process valid content
	require.Len(t, chunks, 1)
	assert.Equal(t, "valid content", chunks[0].Content)
}

func TestInvokeStream_EmptyDelta(t *testing.T) {
	events := []Event{
		{Type: EventOutputTextDelta, Delta: ""},
		{Type: EventOutputTextDelta, Delta: "actual content"},
		{Type: EventResponseCompleted, Response: &Response{ID: "resp", Status: "completed"}},
	}

	server := createTestServer(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)
		for _, event := range events {
			w.Write([]byte(createSSEEvent("", event)))
		}
		w.Write([]byte(createSSEDone()))
	})
	defer server.Close()

	config := &WorkflowClientConfig{
		ProjectEndpoint: server.URL,
		AgentName:       "test-agent",
		APIToken:        "test-token",
	}

	client, err := NewWorkflowClient(config)
	require.NoError(t, err)
	client.httpClient = server.Client()

	req := &InvokeRequest{
		ExtConversationID: "conv-empty",
		Message:           "Test",
	}

	ch, err := client.InvokeStream(context.Background(), req)
	require.NoError(t, err)

	var contentChunks []*StreamChunk
	for chunk := range ch {
		if chunk.Type == ChunkTypeContent {
			contentChunks = append(contentChunks, chunk)
		}
	}

	// Empty delta should not produce a chunk
	require.Len(t, contentChunks, 1)
	assert.Equal(t, "actual content", contentChunks[0].Content)
}

// =============================================================================
// Authentication Tests - Bearer Token
// =============================================================================

func TestInvoke_BearerToken(t *testing.T) {
	var receivedAuth string

	server := createTestServer(func(w http.ResponseWriter, r *http.Request) {
		receivedAuth = r.Header.Get("Authorization")
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)
		event := Event{Type: EventOutputTextDelta, Delta: "Response"}
		w.Write([]byte(createSSEEvent("", event)))
		w.Write([]byte(createSSEDone()))
	})
	defer server.Close()

	config := &WorkflowClientConfig{
		ProjectEndpoint: server.URL,
		AgentName:       "test-agent",
		APIToken:        "my-bearer-token",
	}

	client, err := NewWorkflowClient(config)
	require.NoError(t, err)
	client.httpClient = server.Client()

	req := &InvokeRequest{
		ExtConversationID: "conv-bearer",
		Message:           "Test",
	}

	_, err = client.Invoke(context.Background(), req)
	require.NoError(t, err)

	assert.Equal(t, "Bearer my-bearer-token", receivedAuth)
}

// =============================================================================
// Headers Verification Tests
// =============================================================================

func TestInvoke_CorrectHeaders(t *testing.T) {
	var receivedHeaders http.Header

	server := createTestServer(func(w http.ResponseWriter, r *http.Request) {
		receivedHeaders = r.Header.Clone()
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)
		event := Event{Type: EventOutputTextDelta, Delta: "Response"}
		w.Write([]byte(createSSEEvent("", event)))
		w.Write([]byte(createSSEDone()))
	})
	defer server.Close()

	config := &WorkflowClientConfig{
		ProjectEndpoint: server.URL,
		AgentName:       "test-agent",
		APIToken:        "test-token",
	}

	client, err := NewWorkflowClient(config)
	require.NoError(t, err)
	client.httpClient = server.Client()

	req := &InvokeRequest{
		ExtConversationID: "conv-headers",
		Message:           "Test",
	}

	_, err = client.Invoke(context.Background(), req)
	require.NoError(t, err)

	assert.Equal(t, "application/json", receivedHeaders.Get("Content-Type"))
	assert.Equal(t, "text/event-stream", receivedHeaders.Get("Accept"))
	assert.Equal(t, "Bearer test-token", receivedHeaders.Get("Authorization"))
}

// =============================================================================
// Request Body Tests
// =============================================================================

func TestInvoke_CorrectRequestBody(t *testing.T) {
	var receivedBody []byte

	server := createTestServer(func(w http.ResponseWriter, r *http.Request) {
		receivedBody, _ = io.ReadAll(r.Body)
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)
		event := Event{Type: EventOutputTextDelta, Delta: "Response"}
		w.Write([]byte(createSSEEvent("", event)))
		w.Write([]byte(createSSEDone()))
	})
	defer server.Close()

	config := &WorkflowClientConfig{
		ProjectEndpoint: server.URL,
		AgentName:       "my-agent",
		APIToken:        "test-token",
	}

	client, err := NewWorkflowClient(config)
	require.NoError(t, err)
	client.httpClient = server.Client()

	req := &InvokeRequest{
		ExtConversationID: "conv-body",
		Message:           "Test message content",
	}

	_, err = client.Invoke(context.Background(), req)
	require.NoError(t, err)

	var payload RequestPayload
	err = json.Unmarshal(receivedBody, &payload)
	require.NoError(t, err)

	assert.Equal(t, "agent_reference", payload.Agent.Type)
	assert.Equal(t, "my-agent", payload.Agent.Name)
	assert.Equal(t, "conv-body", payload.Conversation)
	assert.Equal(t, "Test message content", payload.Input)
	assert.True(t, payload.Stream)
}

func TestInvoke_RequestURL(t *testing.T) {
	var receivedURL string

	server := createTestServer(func(w http.ResponseWriter, r *http.Request) {
		receivedURL = r.URL.String()
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)
		event := Event{Type: EventOutputTextDelta, Delta: "Response"}
		w.Write([]byte(createSSEEvent("", event)))
		w.Write([]byte(createSSEDone()))
	})
	defer server.Close()

	config := &WorkflowClientConfig{
		ProjectEndpoint: server.URL,
		APIVersion:      "2025-11-15-preview",
		AgentName:       "test-agent",
		APIToken:        "test-token",
	}

	client, err := NewWorkflowClient(config)
	require.NoError(t, err)
	client.httpClient = server.Client()

	req := &InvokeRequest{
		ExtConversationID: "conv-url",
		Message:           "Test",
	}

	_, err = client.Invoke(context.Background(), req)
	require.NoError(t, err)

	assert.Contains(t, receivedURL, "/openai/responses")
	assert.Contains(t, receivedURL, "api-version=2025-11-15-preview")
}

func TestInvoke_WithMultimodalInput(t *testing.T) {
	var receivedBody []byte

	server := createTestServer(func(w http.ResponseWriter, r *http.Request) {
		receivedBody, _ = io.ReadAll(r.Body)
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)
		event := Event{Type: EventOutputTextDelta, Delta: "Response"}
		w.Write([]byte(createSSEEvent("", event)))
		w.Write([]byte(createSSEDone()))
	})
	defer server.Close()

	config := &WorkflowClientConfig{
		ProjectEndpoint: server.URL,
		AgentName:       "test-agent",
		APIToken:        "test-token",
	}

	client, err := NewWorkflowClient(config)
	require.NoError(t, err)
	client.httpClient = server.Client()

	multimodalInput := []InputMessage{
		{
			Role: "user",
			Content: []InputContent{
				{Type: InputTypeText, Text: "What is in this image?"},
				{Type: InputTypeImage, ImageURL: "data:image/png;base64,iVBORw0KGgo=", Detail: "high"},
			},
		},
	}

	req := &InvokeRequest{
		ExtConversationID: "conv-multimodal",
		Message:           "What is in this image?",
		Input:             multimodalInput,
	}

	resp, err := client.Invoke(context.Background(), req)

	require.NoError(t, err)
	require.NotNil(t, resp)

	var payload map[string]interface{}
	err = json.Unmarshal(receivedBody, &payload)
	require.NoError(t, err)

	// Input should be the multimodal content
	input, ok := payload["input"].([]interface{})
	assert.True(t, ok, "Input should be an array")
	assert.Len(t, input, 1)
}

// =============================================================================
// StreamReader Tests
// =============================================================================

func TestStreamReader_Read_ValidContent(t *testing.T) {
	events := []Event{
		{Type: EventOutputTextDelta, Delta: "Test content"},
	}

	server := createTestServer(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)
		for _, event := range events {
			w.Write([]byte(createSSEEvent("", event)))
		}
		w.Write([]byte(createSSEDone()))
	})
	defer server.Close()

	config := &WorkflowClientConfig{
		ProjectEndpoint: server.URL,
		AgentName:       "test-agent",
		APIToken:        "test-token",
	}

	client, err := NewWorkflowClient(config)
	require.NoError(t, err)
	client.httpClient = server.Client()

	req := &InvokeRequest{
		ExtConversationID: "conv-reader",
		Message:           "Test",
	}

	reader, err := client.InvokeStreamReader(context.Background(), req)
	require.NoError(t, err)
	defer reader.Close()

	chunk, err := reader.Read()

	require.NoError(t, err)
	require.NotNil(t, chunk)
	assert.Equal(t, ChunkTypeContent, chunk.Type)
	assert.Equal(t, "Test content", chunk.Content)
}

func TestStreamReader_Read_EOF(t *testing.T) {
	server := createTestServer(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(createSSEDone()))
	})
	defer server.Close()

	config := &WorkflowClientConfig{
		ProjectEndpoint: server.URL,
		AgentName:       "test-agent",
		APIToken:        "test-token",
	}

	client, err := NewWorkflowClient(config)
	require.NoError(t, err)
	client.httpClient = server.Client()

	req := &InvokeRequest{
		ExtConversationID: "conv-eof",
		Message:           "Test",
	}

	reader, err := client.InvokeStreamReader(context.Background(), req)
	require.NoError(t, err)
	defer reader.Close()

	chunk, err := reader.Read()

	assert.Equal(t, io.EOF, err)
	assert.Nil(t, chunk)
}

func TestStreamReader_Close(t *testing.T) {
	server := createTestServer(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)
		event := Event{Type: EventOutputTextDelta, Delta: "Content"}
		w.Write([]byte(createSSEEvent("", event)))
		w.Write([]byte(createSSEDone()))
	})
	defer server.Close()

	config := &WorkflowClientConfig{
		ProjectEndpoint: server.URL,
		AgentName:       "test-agent",
		APIToken:        "test-token",
	}

	client, err := NewWorkflowClient(config)
	require.NoError(t, err)
	client.httpClient = server.Client()

	req := &InvokeRequest{
		ExtConversationID: "conv-close",
		Message:           "Test",
	}

	reader, err := client.InvokeStreamReader(context.Background(), req)
	require.NoError(t, err)

	err = reader.Close()
	assert.NoError(t, err)

	// Double close should be safe
	err = reader.Close()
	assert.NoError(t, err)
}

// =============================================================================
// Event Processing Tests
// =============================================================================

func TestStreamReader_ProcessEvent_WorkflowAction(t *testing.T) {
	events := []Event{
		{
			Type: EventOutputItemAdded,
			Item: &OutputItem{
				Type:     "workflow_action",
				ID:       "action-1",
				Kind:     "tool_call",
				ActionID: "act-123",
				Status:   "in_progress",
			},
		},
		{Type: EventOutputTextDelta, Delta: "Action result"},
		{Type: EventResponseCompleted, Response: &Response{ID: "resp", Status: "completed"}},
	}

	server := createTestServer(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)
		for _, event := range events {
			w.Write([]byte(createSSEEvent("", event)))
		}
		w.Write([]byte(createSSEDone()))
	})
	defer server.Close()

	config := &WorkflowClientConfig{
		ProjectEndpoint: server.URL,
		AgentName:       "test-agent",
		APIToken:        "test-token",
	}

	client, err := NewWorkflowClient(config)
	require.NoError(t, err)
	client.httpClient = server.Client()

	req := &InvokeRequest{
		ExtConversationID: "conv-action",
		Message:           "Test",
	}

	ch, err := client.InvokeStream(context.Background(), req)
	require.NoError(t, err)

	var toolCallChunks []*StreamChunk
	for chunk := range ch {
		if chunk.Type == ChunkTypeToolCallStart {
			toolCallChunks = append(toolCallChunks, chunk)
		}
	}

	// Should have workflow action as tool call
	require.NotEmpty(t, toolCallChunks)
	actionChunk := toolCallChunks[0]
	assert.Equal(t, "tool_call", actionChunk.Config["tool_name"])
	assert.Equal(t, "workflow_action", actionChunk.Config["call_type"])
	assert.Equal(t, "act-123", actionChunk.Config["action_id"])
}

func TestStreamReader_ProcessEvent_MessageDone(t *testing.T) {
	events := []Event{
		{
			Type: EventOutputItemDone,
			Item: &OutputItem{
				Type:   "message",
				ID:     "msg-done",
				Status: "completed",
				Role:   "assistant",
				Content: []ContentPart{
					{Type: "output_text", Text: "Complete message"},
				},
				CreatedBy: &CreatedBy{
					Agent:      &AgentRef{Name: "test-agent"},
					ResponseID: "resp-123",
				},
			},
		},
		{Type: EventResponseCompleted, Response: &Response{ID: "resp-123", Status: "completed"}},
	}

	server := createTestServer(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)
		for _, event := range events {
			w.Write([]byte(createSSEEvent("", event)))
		}
		w.Write([]byte(createSSEDone()))
	})
	defer server.Close()

	config := &WorkflowClientConfig{
		ProjectEndpoint: server.URL,
		AgentName:       "test-agent",
		APIToken:        "test-token",
	}

	client, err := NewWorkflowClient(config)
	require.NoError(t, err)
	client.httpClient = server.Client()

	req := &InvokeRequest{
		ExtConversationID: "conv-done",
		Message:           "Test",
	}

	ch, err := client.InvokeStream(context.Background(), req)
	require.NoError(t, err)

	var chunks []*StreamChunk //nolint:prealloc // length unknown: reading from channel
	for chunk := range ch {
		chunks = append(chunks, chunk)
	}

	// Should have message_done metadata
	hasMessageDone := false
	for _, chunk := range chunks {
		if chunk.Type == ChunkTypeMetadata {
			if msgType, ok := chunk.Metadata["type"].(string); ok && msgType == "message_done" {
				hasMessageDone = true
				assert.Equal(t, "msg-done", chunk.Metadata["message_id"])
				assert.Equal(t, "assistant", chunk.Metadata["role"])
				assert.Equal(t, "test-agent", chunk.Metadata["agent_name"])
			}
		}
	}
	assert.True(t, hasMessageDone, "Expected message_done metadata")
}

// =============================================================================
// Close Tests
// =============================================================================

func TestWorkflowClient_Close(t *testing.T) {
	config := &WorkflowClientConfig{
		ProjectEndpoint: "https://project.api.azure.com",
		AgentName:       "test-agent",
		APIToken:        "test-token",
	}

	client, err := NewWorkflowClient(config)
	require.NoError(t, err)

	err = client.Close()
	assert.NoError(t, err)
}

// =============================================================================
// Factory Tests
// =============================================================================

func TestFactory_NewFactory(t *testing.T) {
	factory := NewFactory()
	assert.NotNil(t, factory)
}

// =============================================================================
// Empty Conversation ID Tests
// =============================================================================

func TestInvoke_EmptyConversationID(t *testing.T) {
	var receivedBody []byte

	server := createTestServer(func(w http.ResponseWriter, r *http.Request) {
		receivedBody, _ = io.ReadAll(r.Body)
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)
		event := Event{Type: EventOutputTextDelta, Delta: "Response"}
		w.Write([]byte(createSSEEvent("", event)))
		w.Write([]byte(createSSEDone()))
	})
	defer server.Close()

	config := &WorkflowClientConfig{
		ProjectEndpoint: server.URL,
		AgentName:       "test-agent",
		APIToken:        "test-token",
	}

	client, err := NewWorkflowClient(config)
	require.NoError(t, err)
	client.httpClient = server.Client()

	req := &InvokeRequest{
		ExtConversationID: "", // Empty - should create new conversation
		Message:           "Test",
	}

	_, err = client.Invoke(context.Background(), req)
	require.NoError(t, err)

	var payload map[string]interface{}
	err = json.Unmarshal(receivedBody, &payload)
	require.NoError(t, err)

	// Conversation should be empty or omitted
	conv, exists := payload["conversation"]
	if exists {
		assert.Equal(t, "", conv)
	}
}

// =============================================================================
// GetMessages Tests
// =============================================================================

func TestStreamReader_GetMessages_Empty(t *testing.T) {
	// Create a stream reader that has no messages
	server := createTestServer(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)
		// Only content chunks, no message_done events
		event := Event{Type: EventOutputTextDelta, Delta: "Some content"}
		w.Write([]byte(createSSEEvent("", event)))
		w.Write([]byte(createSSEDone()))
	})
	defer server.Close()

	config := &WorkflowClientConfig{
		ProjectEndpoint: server.URL,
		AgentName:       "test-agent",
		APIToken:        "test-token",
	}

	client, err := NewWorkflowClient(config)
	require.NoError(t, err)
	client.httpClient = server.Client()

	req := &InvokeRequest{
		ExtConversationID: "conv-getmsg-empty",
		Message:           "Test",
	}

	reader, err := client.InvokeStreamReader(context.Background(), req)
	require.NoError(t, err)
	defer reader.Close()

	// Read all chunks to completion
	for {
		_, err := reader.Read()
		if errors.Is(err, io.EOF) {
			break
		}
		require.NoError(t, err)
	}

	// Cast to foundryStreamReader to access GetMessages
	fsr, ok := reader.(*foundryStreamReader)
	require.True(t, ok, "Expected foundryStreamReader type")

	messages := fsr.GetMessages()
	assert.Empty(t, messages, "Expected no messages when no message_done events")
}

func TestStreamReader_GetMessages_SingleMessage(t *testing.T) {
	events := []Event{
		{
			Type: EventOutputItemAdded,
			Item: &OutputItem{
				Type: "message",
				ID:   "msg-single",
				Role: "assistant",
			},
		},
		{Type: EventOutputTextDelta, Delta: "Hello there!"},
		{
			Type: EventOutputItemDone,
			Item: &OutputItem{
				Type:   "message",
				ID:     "msg-single",
				Status: "completed",
				Role:   "assistant",
				Content: []ContentPart{
					{Type: "output_text", Text: "Hello there!"},
				},
				CreatedBy: &CreatedBy{
					Agent:      &AgentRef{Name: "test-agent"},
					ResponseID: "resp-single",
				},
			},
			OutputIndex: 0,
		},
		{Type: EventResponseCompleted, Response: &Response{ID: "resp-single", Status: "completed"}},
	}

	server := createTestServer(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)
		for _, event := range events {
			w.Write([]byte(createSSEEvent("", event)))
		}
		w.Write([]byte(createSSEDone()))
	})
	defer server.Close()

	config := &WorkflowClientConfig{
		ProjectEndpoint: server.URL,
		AgentName:       "test-agent",
		APIToken:        "test-token",
	}

	client, err := NewWorkflowClient(config)
	require.NoError(t, err)
	client.httpClient = server.Client()

	req := &InvokeRequest{
		ExtConversationID: "conv-getmsg-single",
		Message:           "Test",
	}

	reader, err := client.InvokeStreamReader(context.Background(), req)
	require.NoError(t, err)
	defer reader.Close()

	// Read all chunks to completion
	for {
		_, err := reader.Read()
		if errors.Is(err, io.EOF) {
			break
		}
		require.NoError(t, err)
	}

	fsr, ok := reader.(*foundryStreamReader)
	require.True(t, ok)

	messages := fsr.GetMessages()
	require.Len(t, messages, 1, "Expected exactly one message")

	msg := messages[0]
	assert.Equal(t, "msg-single", msg.ID)
	assert.Equal(t, "assistant", msg.Role)
	assert.Equal(t, "Hello there!", msg.Content)
	assert.Equal(t, "test-agent", msg.AgentName)
	assert.Equal(t, "resp-single", msg.ResponseID)
	assert.Equal(t, "completed", msg.Status)
}

func TestStreamReader_GetMessages_MultipleMessages(t *testing.T) {
	events := []Event{
		// First message
		{
			Type: EventOutputItemAdded,
			Item: &OutputItem{
				Type: "message",
				ID:   "msg-1",
				Role: "assistant",
			},
		},
		{Type: EventOutputTextDelta, Delta: "First message"},
		{
			Type: EventOutputItemDone,
			Item: &OutputItem{
				Type:   "message",
				ID:     "msg-1",
				Status: "completed",
				Role:   "assistant",
				Content: []ContentPart{
					{Type: "output_text", Text: "First message"},
				},
				CreatedBy: &CreatedBy{
					Agent:      &AgentRef{Name: "agent-1"},
					ResponseID: "resp-1",
				},
			},
			OutputIndex: 0,
		},
		// Second message
		{
			Type: EventOutputItemAdded,
			Item: &OutputItem{
				Type: "message",
				ID:   "msg-2",
				Role: "assistant",
			},
		},
		{Type: EventOutputTextDelta, Delta: "Second message"},
		{
			Type: EventOutputItemDone,
			Item: &OutputItem{
				Type:   "message",
				ID:     "msg-2",
				Status: "completed",
				Role:   "assistant",
				Content: []ContentPart{
					{Type: "output_text", Text: "Second message"},
				},
				CreatedBy: &CreatedBy{
					Agent:      &AgentRef{Name: "agent-2"},
					ResponseID: "resp-2",
				},
			},
			OutputIndex: 1,
		},
		{Type: EventResponseCompleted, Response: &Response{ID: "resp-2", Status: "completed"}},
	}

	server := createTestServer(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)
		for _, event := range events {
			w.Write([]byte(createSSEEvent("", event)))
		}
		w.Write([]byte(createSSEDone()))
	})
	defer server.Close()

	config := &WorkflowClientConfig{
		ProjectEndpoint: server.URL,
		AgentName:       "test-agent",
		APIToken:        "test-token",
	}

	client, err := NewWorkflowClient(config)
	require.NoError(t, err)
	client.httpClient = server.Client()

	req := &InvokeRequest{
		ExtConversationID: "conv-getmsg-multi",
		Message:           "Test",
	}

	reader, err := client.InvokeStreamReader(context.Background(), req)
	require.NoError(t, err)
	defer reader.Close()

	// Read all chunks
	for {
		_, err := reader.Read()
		if errors.Is(err, io.EOF) {
			break
		}
		require.NoError(t, err)
	}

	fsr, ok := reader.(*foundryStreamReader)
	require.True(t, ok)

	messages := fsr.GetMessages()
	require.Len(t, messages, 2, "Expected two messages")

	// Verify first message
	assert.Equal(t, "msg-1", messages[0].ID)
	assert.Equal(t, "First message", messages[0].Content)
	assert.Equal(t, "agent-1", messages[0].AgentName)

	// Verify second message
	assert.Equal(t, "msg-2", messages[1].ID)
	assert.Equal(t, "Second message", messages[1].Content)
	assert.Equal(t, "agent-2", messages[1].AgentName)
}

func TestStreamReader_GetMessages_WithMetadata(t *testing.T) {
	events := []Event{
		{
			Type: EventOutputItemDone,
			Item: &OutputItem{
				Type:   "message",
				ID:     "msg-meta",
				Status: "completed",
				Role:   "assistant",
				Content: []ContentPart{
					{Type: "output_text", Text: "Message with metadata"},
				},
				CreatedBy: &CreatedBy{
					Agent:      &AgentRef{Name: "meta-agent"},
					ResponseID: "resp-meta",
				},
			},
			OutputIndex: 5,
		},
		{Type: EventResponseCompleted, Response: &Response{ID: "resp-meta", Status: "completed"}},
	}

	server := createTestServer(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)
		for _, event := range events {
			w.Write([]byte(createSSEEvent("", event)))
		}
		w.Write([]byte(createSSEDone()))
	})
	defer server.Close()

	config := &WorkflowClientConfig{
		ProjectEndpoint: server.URL,
		AgentName:       "test-agent",
		APIToken:        "test-token",
	}

	client, err := NewWorkflowClient(config)
	require.NoError(t, err)
	client.httpClient = server.Client()

	req := &InvokeRequest{
		ExtConversationID: "conv-meta",
		Message:           "Test",
	}

	reader, err := client.InvokeStreamReader(context.Background(), req)
	require.NoError(t, err)
	defer reader.Close()

	for {
		_, err := reader.Read()
		if errors.Is(err, io.EOF) {
			break
		}
		require.NoError(t, err)
	}

	fsr, ok := reader.(*foundryStreamReader)
	require.True(t, ok)

	messages := fsr.GetMessages()
	require.Len(t, messages, 1)

	msg := messages[0]
	assert.Equal(t, "msg-meta", msg.ID)
	assert.NotNil(t, msg.Metadata, "Expected metadata to be set")
	assert.Equal(t, 5, msg.Metadata["output_index"])
	assert.False(t, msg.CreatedAt.IsZero(), "Expected CreatedAt to be set")
}

func TestStreamReader_GetMessages_MultipleContentParts(t *testing.T) {
	events := []Event{
		{
			Type: EventOutputItemDone,
			Item: &OutputItem{
				Type:   "message",
				ID:     "msg-parts",
				Status: "completed",
				Role:   "assistant",
				Content: []ContentPart{
					{Type: "output_text", Text: "Part 1. "},
					{Type: "output_text", Text: "Part 2. "},
					{Type: "output_text", Text: "Part 3."},
				},
				CreatedBy: &CreatedBy{
					Agent:      &AgentRef{Name: "parts-agent"},
					ResponseID: "resp-parts",
				},
			},
		},
		{Type: EventResponseCompleted, Response: &Response{ID: "resp-parts", Status: "completed"}},
	}

	server := createTestServer(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)
		for _, event := range events {
			w.Write([]byte(createSSEEvent("", event)))
		}
		w.Write([]byte(createSSEDone()))
	})
	defer server.Close()

	config := &WorkflowClientConfig{
		ProjectEndpoint: server.URL,
		AgentName:       "test-agent",
		APIToken:        "test-token",
	}

	client, err := NewWorkflowClient(config)
	require.NoError(t, err)
	client.httpClient = server.Client()

	req := &InvokeRequest{
		ExtConversationID: "conv-parts",
		Message:           "Test",
	}

	reader, err := client.InvokeStreamReader(context.Background(), req)
	require.NoError(t, err)
	defer reader.Close()

	for {
		_, err := reader.Read()
		if errors.Is(err, io.EOF) {
			break
		}
		require.NoError(t, err)
	}

	fsr, ok := reader.(*foundryStreamReader)
	require.True(t, ok)

	messages := fsr.GetMessages()
	require.Len(t, messages, 1)

	// Content parts should be concatenated
	assert.Equal(t, "Part 1. Part 2. Part 3.", messages[0].Content)
}

func TestStreamReader_GetMessages_NoCreatedBy(t *testing.T) {
	events := []Event{
		{
			Type: EventOutputItemDone,
			Item: &OutputItem{
				Type:   "message",
				ID:     "msg-nocreator",
				Status: "completed",
				Role:   "assistant",
				Content: []ContentPart{
					{Type: "output_text", Text: "Message without creator"},
				},
				// No CreatedBy field
			},
		},
		{Type: EventResponseCompleted, Response: &Response{ID: "resp", Status: "completed"}},
	}

	server := createTestServer(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)
		for _, event := range events {
			w.Write([]byte(createSSEEvent("", event)))
		}
		w.Write([]byte(createSSEDone()))
	})
	defer server.Close()

	config := &WorkflowClientConfig{
		ProjectEndpoint: server.URL,
		AgentName:       "test-agent",
		APIToken:        "test-token",
	}

	client, err := NewWorkflowClient(config)
	require.NoError(t, err)
	client.httpClient = server.Client()

	req := &InvokeRequest{
		ExtConversationID: "conv-nocreator",
		Message:           "Test",
	}

	reader, err := client.InvokeStreamReader(context.Background(), req)
	require.NoError(t, err)
	defer reader.Close()

	for {
		_, err := reader.Read()
		if errors.Is(err, io.EOF) {
			break
		}
		require.NoError(t, err)
	}

	fsr, ok := reader.(*foundryStreamReader)
	require.True(t, ok)

	messages := fsr.GetMessages()
	require.Len(t, messages, 1)

	msg := messages[0]
	assert.Equal(t, "msg-nocreator", msg.ID)
	assert.Equal(t, "", msg.AgentName, "Expected empty agent name when no CreatedBy")
	assert.Equal(t, "", msg.ResponseID, "Expected empty response ID when no CreatedBy")
}
