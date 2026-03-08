package agents

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

	"github.com/unifiedui/agent-service/internal/services/agents/foundry"
	"github.com/unifiedui/agent-service/internal/services/agents/n8n"
)

// =============================================================================
// Test Helpers
// =============================================================================

// createN8NSSEResponse creates a Server-Sent Events formatted response for n8n
func createN8NSSEResponse(events []n8n.StreamEvent) string {
	var sb strings.Builder
	for _, event := range events {
		data, _ := json.Marshal(event)
		sb.Write(data)
		sb.WriteString("\n")
	}
	return sb.String()
}

// createFoundrySSEEvent creates a single SSE event line for Foundry
func createFoundrySSEEvent(eventType string, data interface{}) string {
	jsonData, _ := json.Marshal(data)
	if eventType != "" {
		return "event: " + eventType + "\ndata: " + string(jsonData) + "\n\n"
	}
	return "data: " + string(jsonData) + "\n\n"
}

// createFoundrySSEDone creates the [DONE] marker for Foundry SSE
func createFoundrySSEDone() string {
	return "data: [DONE]\n\n"
}

// =============================================================================
// n8nWorkflowAdapter.Invoke Tests
// =============================================================================

func TestN8NWorkflowAdapter_Invoke_Success(t *testing.T) {
	events := []n8n.StreamEvent{
		{Type: n8n.StreamTypeItem, Content: "Hello from n8n!"},
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify request
		assert.Equal(t, http.MethodPost, r.Method)
		assert.Equal(t, "application/json", r.Header.Get("Content-Type"))

		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(createN8NSSEResponse(events)))
	}))
	defer server.Close()

	config := &n8n.ChatWorkflowConfig{
		ChatURL:    server.URL,
		HTTPClient: server.Client(),
	}
	client, err := n8n.NewChatWorkflowClient(config)
	require.NoError(t, err)

	adapter := &n8nWorkflowAdapter{
		client:        client,
		fileConverter: n8n.NewFileConverter(),
	}

	req := &InvokeRequest{
		ConversationID: "conv-123",
		Message:        "Hello",
		SessionID:      "session-456",
	}

	resp, err := adapter.Invoke(context.Background(), req)

	require.NoError(t, err)
	require.NotNil(t, resp)
	assert.Equal(t, "Hello from n8n!", resp.Content)
	assert.Equal(t, "session-456", resp.SessionID)
}

func TestN8NWorkflowAdapter_Invoke_WithContextData(t *testing.T) {
	var receivedBody map[string]interface{}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		json.Unmarshal(body, &receivedBody)

		events := []n8n.StreamEvent{
			{Type: n8n.StreamTypeItem, Content: "Response with context"},
		}
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(createN8NSSEResponse(events)))
	}))
	defer server.Close()

	config := &n8n.ChatWorkflowConfig{
		ChatURL:    server.URL,
		HTTPClient: server.Client(),
	}
	client, err := n8n.NewChatWorkflowClient(config)
	require.NoError(t, err)

	adapter := &n8nWorkflowAdapter{
		client:        client,
		fileConverter: n8n.NewFileConverter(),
	}

	req := &InvokeRequest{
		ConversationID: "conv-123",
		Message:        "Hello",
		SessionID:      "session-456",
		ContextData: map[string]string{
			"user_id": "user-789",
			"tenant":  "acme",
		},
	}

	resp, err := adapter.Invoke(context.Background(), req)

	require.NoError(t, err)
	require.NotNil(t, resp)

	// Verify context was prepended to message
	chatInput, ok := receivedBody["chatInput"].(string)
	require.True(t, ok)
	assert.Contains(t, chatInput, "Hello")
	assert.Contains(t, chatInput, "user_id")
	assert.Contains(t, chatInput, "tenant")
}

func TestN8NWorkflowAdapter_Invoke_WithFiles(t *testing.T) {
	var receivedBody map[string]interface{}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		json.Unmarshal(body, &receivedBody)

		events := []n8n.StreamEvent{
			{Type: n8n.StreamTypeItem, Content: "Processed file"},
		}
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(createN8NSSEResponse(events)))
	}))
	defer server.Close()

	config := &n8n.ChatWorkflowConfig{
		ChatURL:    server.URL,
		HTTPClient: server.Client(),
	}
	client, err := n8n.NewChatWorkflowClient(config)
	require.NoError(t, err)

	adapter := &n8nWorkflowAdapter{
		client:        client,
		fileConverter: n8n.NewFileConverter(),
	}

	req := &InvokeRequest{
		ConversationID: "conv-123",
		Message:        "Analyze this image",
		SessionID:      "session-456",
		Files: []FileInput{
			{
				Type:     "image",
				ImageURL: "data:image/png;base64,iVBORw0KGgo=",
				Filename: "test.png",
				MimeType: "image/png",
			},
		},
	}

	resp, err := adapter.Invoke(context.Background(), req)

	require.NoError(t, err)
	require.NotNil(t, resp)
	assert.Equal(t, "Processed file", resp.Content)

	// Verify files were included in request
	files, ok := receivedBody["files"].([]interface{})
	require.True(t, ok)
	assert.Len(t, files, 1)
}

func TestN8NWorkflowAdapter_Invoke_HTTPError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("Internal Server Error"))
	}))
	defer server.Close()

	config := &n8n.ChatWorkflowConfig{
		ChatURL:    server.URL,
		HTTPClient: server.Client(),
	}
	client, err := n8n.NewChatWorkflowClient(config)
	require.NoError(t, err)

	adapter := &n8nWorkflowAdapter{
		client:        client,
		fileConverter: n8n.NewFileConverter(),
	}

	req := &InvokeRequest{
		ConversationID: "conv-123",
		Message:        "Hello",
		SessionID:      "session-456",
	}

	resp, err := adapter.Invoke(context.Background(), req)

	require.Error(t, err)
	assert.Nil(t, resp)
	assert.Contains(t, err.Error(), "500")
}

// =============================================================================
// n8nWorkflowAdapter.InvokeStream Tests
// =============================================================================

func TestN8NWorkflowAdapter_InvokeStream_Success(t *testing.T) {
	events := []n8n.StreamEvent{
		{Type: n8n.StreamTypeItem, Content: "Chunk 1"},
		{Type: n8n.StreamTypeItem, Content: "Chunk 2"},
		{Type: n8n.StreamTypeItem, Content: "Chunk 3"},
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(createN8NSSEResponse(events)))
	}))
	defer server.Close()

	config := &n8n.ChatWorkflowConfig{
		ChatURL:    server.URL,
		HTTPClient: server.Client(),
	}
	client, err := n8n.NewChatWorkflowClient(config)
	require.NoError(t, err)

	adapter := &n8nWorkflowAdapter{
		client:        client,
		fileConverter: n8n.NewFileConverter(),
	}

	req := &InvokeRequest{
		ConversationID: "conv-123",
		Message:        "Hello",
	}

	ch, err := adapter.InvokeStream(context.Background(), req)

	require.NoError(t, err)
	require.NotNil(t, ch)

	var chunks []*StreamChunk //nolint:prealloc // length unknown: reading from channel
	for chunk := range ch {
		chunks = append(chunks, chunk)
	}

	assert.Len(t, chunks, 3)
	assert.Equal(t, "Chunk 1", chunks[0].Content)
	assert.Equal(t, "Chunk 2", chunks[1].Content)
	assert.Equal(t, "Chunk 3", chunks[2].Content)
}

func TestN8NWorkflowAdapter_InvokeStream_WithFiles(t *testing.T) {
	events := []n8n.StreamEvent{
		{Type: n8n.StreamTypeItem, Content: "Image analysis result"},
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(createN8NSSEResponse(events)))
	}))
	defer server.Close()

	config := &n8n.ChatWorkflowConfig{
		ChatURL:    server.URL,
		HTTPClient: server.Client(),
	}
	client, err := n8n.NewChatWorkflowClient(config)
	require.NoError(t, err)

	adapter := &n8nWorkflowAdapter{
		client:        client,
		fileConverter: n8n.NewFileConverter(),
	}

	req := &InvokeRequest{
		Message: "Describe this image",
		Files: []FileInput{
			{
				Type:     "image",
				ImageURL: "https://example.com/image.jpg",
				Filename: "image.jpg",
				MimeType: "image/jpeg",
			},
		},
	}

	ch, err := adapter.InvokeStream(context.Background(), req)

	require.NoError(t, err)
	require.NotNil(t, ch)

	var chunks []*StreamChunk //nolint:prealloc // length unknown: reading from channel
	for chunk := range ch {
		chunks = append(chunks, chunk)
	}

	assert.Len(t, chunks, 1)
	assert.Equal(t, "Image analysis result", chunks[0].Content)
}

func TestN8NWorkflowAdapter_InvokeStream_HTTPError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
	}))
	defer server.Close()

	config := &n8n.ChatWorkflowConfig{
		ChatURL:    server.URL,
		HTTPClient: server.Client(),
	}
	client, err := n8n.NewChatWorkflowClient(config)
	require.NoError(t, err)

	adapter := &n8nWorkflowAdapter{
		client:        client,
		fileConverter: n8n.NewFileConverter(),
	}

	req := &InvokeRequest{
		Message: "Hello",
	}

	ch, err := adapter.InvokeStream(context.Background(), req)

	require.Error(t, err)
	assert.Nil(t, ch)
}

// =============================================================================
// n8nWorkflowAdapter.InvokeStreamReader Tests
// =============================================================================

func TestN8NWorkflowAdapter_InvokeStreamReader_Success(t *testing.T) {
	events := []n8n.StreamEvent{
		{Type: n8n.StreamTypeItem, Content: "Hello"},
		{Type: n8n.StreamTypeItem, Content: " World"},
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(createN8NSSEResponse(events)))
	}))
	defer server.Close()

	config := &n8n.ChatWorkflowConfig{
		ChatURL:    server.URL,
		HTTPClient: server.Client(),
	}
	client, err := n8n.NewChatWorkflowClient(config)
	require.NoError(t, err)

	adapter := &n8nWorkflowAdapter{
		client:        client,
		fileConverter: n8n.NewFileConverter(),
	}

	req := &InvokeRequest{
		Message: "Greet me",
	}

	reader, err := adapter.InvokeStreamReader(context.Background(), req)

	require.NoError(t, err)
	require.NotNil(t, reader)
	defer reader.Close()

	// Read first chunk
	chunk1, err := reader.Read()
	require.NoError(t, err)
	assert.Equal(t, ChunkTypeContent, chunk1.Type)
	assert.Equal(t, "Hello", chunk1.Content)

	// Read second chunk
	chunk2, err := reader.Read()
	require.NoError(t, err)
	assert.Equal(t, ChunkTypeContent, chunk2.Type)
	assert.Equal(t, " World", chunk2.Content)

	// Verify EOF
	_, err = reader.Read()
	assert.Equal(t, io.EOF, err)
}

func TestN8NWorkflowAdapter_InvokeStreamReader_WithContextData(t *testing.T) {
	var receivedBody map[string]interface{}

	events := []n8n.StreamEvent{
		{Type: n8n.StreamTypeItem, Content: "Response"},
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		json.Unmarshal(body, &receivedBody)

		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(createN8NSSEResponse(events)))
	}))
	defer server.Close()

	config := &n8n.ChatWorkflowConfig{
		ChatURL:    server.URL,
		HTTPClient: server.Client(),
	}
	client, err := n8n.NewChatWorkflowClient(config)
	require.NoError(t, err)

	adapter := &n8nWorkflowAdapter{
		client:        client,
		fileConverter: n8n.NewFileConverter(),
	}

	req := &InvokeRequest{
		Message: "Test message",
		ContextData: map[string]string{
			"env": "production",
		},
	}

	reader, err := adapter.InvokeStreamReader(context.Background(), req)

	require.NoError(t, err)
	require.NotNil(t, reader)
	defer reader.Close()

	// Verify context was prepended
	chatInput, ok := receivedBody["chatInput"].(string)
	require.True(t, ok)
	assert.Contains(t, chatInput, "env")
	assert.Contains(t, chatInput, "production")
}

func TestN8NWorkflowAdapter_InvokeStreamReader_HTTPError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	}))
	defer server.Close()

	config := &n8n.ChatWorkflowConfig{
		ChatURL:    server.URL,
		HTTPClient: server.Client(),
	}
	client, err := n8n.NewChatWorkflowClient(config)
	require.NoError(t, err)

	adapter := &n8nWorkflowAdapter{
		client:        client,
		fileConverter: n8n.NewFileConverter(),
	}

	req := &InvokeRequest{
		Message: "Test",
	}

	reader, err := adapter.InvokeStreamReader(context.Background(), req)

	require.Error(t, err)
	assert.Nil(t, reader)
}

// =============================================================================
// n8nStreamReaderAdapter Tests
// =============================================================================

// mockN8NStreamReader is a mock implementation of n8n.StreamReader for testing
type mockN8NStreamReader struct {
	chunks     []*n8n.StreamChunk
	index      int
	closeError error
}

func (m *mockN8NStreamReader) Read() (*n8n.StreamChunk, error) {
	if m.index >= len(m.chunks) {
		return nil, io.EOF
	}
	chunk := m.chunks[m.index]
	m.index++
	return chunk, nil
}

func (m *mockN8NStreamReader) Close() error {
	return m.closeError
}

func TestN8NStreamReaderAdapter_Read_Success(t *testing.T) {
	mockReader := &mockN8NStreamReader{
		chunks: []*n8n.StreamChunk{
			{Type: n8n.ChunkTypeContent, Content: "Hello", ExecutionID: "exec-1"},
			{Type: n8n.ChunkTypeMetadata, Metadata: map[string]interface{}{"key": "value"}},
		},
	}

	adapter := &n8nStreamReaderAdapter{reader: mockReader}

	// Read first chunk
	chunk1, err := adapter.Read()
	require.NoError(t, err)
	assert.Equal(t, ChunkTypeContent, chunk1.Type)
	assert.Equal(t, "Hello", chunk1.Content)
	assert.Equal(t, "exec-1", chunk1.ExecutionID)

	// Read second chunk
	chunk2, err := adapter.Read()
	require.NoError(t, err)
	assert.Equal(t, ChunkTypeMetadata, chunk2.Type)
	assert.Equal(t, "value", chunk2.Metadata["key"])
}

func TestN8NStreamReaderAdapter_Read_EOF(t *testing.T) {
	mockReader := &mockN8NStreamReader{
		chunks: []*n8n.StreamChunk{},
	}

	adapter := &n8nStreamReaderAdapter{reader: mockReader}

	chunk, err := adapter.Read()

	assert.Equal(t, io.EOF, err)
	assert.Nil(t, chunk)
}

func TestN8NStreamReaderAdapter_Close_Success(t *testing.T) {
	mockReader := &mockN8NStreamReader{
		closeError: nil,
	}

	adapter := &n8nStreamReaderAdapter{reader: mockReader}

	err := adapter.Close()

	assert.NoError(t, err)
}

func TestN8NStreamReaderAdapter_Close_Error(t *testing.T) {
	mockReader := &mockN8NStreamReader{
		closeError: errors.New("close error"),
	}

	adapter := &n8nStreamReaderAdapter{reader: mockReader}

	err := adapter.Close()

	assert.Error(t, err)
	assert.Equal(t, "close error", err.Error())
}

// =============================================================================
// n8nWorkflowAdapter.Close Tests
// =============================================================================

func TestN8NWorkflowAdapter_Close_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	config := &n8n.ChatWorkflowConfig{
		ChatURL:    server.URL,
		HTTPClient: server.Client(),
	}
	client, err := n8n.NewChatWorkflowClient(config)
	require.NoError(t, err)

	adapter := &n8nWorkflowAdapter{
		client:        client,
		fileConverter: n8n.NewFileConverter(),
	}

	err = adapter.Close()

	assert.NoError(t, err)
}

// =============================================================================
// n8nAPIAdapter.GetExecution Tests
// =============================================================================

func TestN8NAPIAdapter_GetExecution_Success(t *testing.T) {
	executionResp := n8n.ExecutionResponse{
		ID:        "exec-123",
		Status:    "success",
		StartedAt: "2024-01-01T10:00:00Z",
		StoppedAt: "2024-01-01T10:00:05Z",
		Data: map[string]interface{}{
			"result": "completed",
		},
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodGet, r.Method)
		assert.Contains(t, r.URL.Path, "/api/v1/executions/exec-123")

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(executionResp)
	}))
	defer server.Close()

	config := &n8n.APIClientConfig{
		BaseURL:    server.URL,
		APIKey:     "test-api-key",
		HTTPClient: server.Client(),
	}
	client, err := n8n.NewAPIClient(config)
	require.NoError(t, err)

	adapter := &n8nAPIAdapter{client: client}

	info, err := adapter.GetExecution(context.Background(), "exec-123")

	require.NoError(t, err)
	require.NotNil(t, info)
	assert.Equal(t, "exec-123", info.ID)
	assert.Equal(t, "success", info.Status)
	assert.Equal(t, "2024-01-01T10:00:00Z", info.StartedAt)
	assert.Equal(t, "2024-01-01T10:00:05Z", info.StoppedAt)
	assert.Equal(t, "completed", info.Data["result"])
}

func TestN8NAPIAdapter_GetExecution_NotFound(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	config := &n8n.APIClientConfig{
		BaseURL:    server.URL,
		APIKey:     "test-api-key",
		HTTPClient: server.Client(),
	}
	client, err := n8n.NewAPIClient(config)
	require.NoError(t, err)

	adapter := &n8nAPIAdapter{client: client}

	info, err := adapter.GetExecution(context.Background(), "nonexistent")

	require.Error(t, err)
	assert.Nil(t, info)
	assert.Contains(t, err.Error(), "404")
}

func TestN8NAPIAdapter_GetExecution_HTTPError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	config := &n8n.APIClientConfig{
		BaseURL:    server.URL,
		APIKey:     "test-api-key",
		HTTPClient: server.Client(),
	}
	client, err := n8n.NewAPIClient(config)
	require.NoError(t, err)

	adapter := &n8nAPIAdapter{client: client}

	info, err := adapter.GetExecution(context.Background(), "exec-123")

	require.Error(t, err)
	assert.Nil(t, info)
}

// =============================================================================
// n8nAPIAdapter.GetExecutionsBySession Tests
// =============================================================================

func TestN8NAPIAdapter_GetExecutionsBySession_Success(t *testing.T) {
	// N8N returns empty since it doesn't have native session-to-execution mapping
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	config := &n8n.APIClientConfig{
		BaseURL:    server.URL,
		APIKey:     "test-api-key",
		HTTPClient: server.Client(),
	}
	client, err := n8n.NewAPIClient(config)
	require.NoError(t, err)

	adapter := &n8nAPIAdapter{client: client}

	infos, err := adapter.GetExecutionsBySession(context.Background(), "session-123")

	require.NoError(t, err)
	assert.NotNil(t, infos)
	assert.Empty(t, infos)
}

// =============================================================================
// n8nAPIAdapter.Close Tests
// =============================================================================

func TestN8NAPIAdapter_Close_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	config := &n8n.APIClientConfig{
		BaseURL:    server.URL,
		HTTPClient: server.Client(),
	}
	client, err := n8n.NewAPIClient(config)
	require.NoError(t, err)

	adapter := &n8nAPIAdapter{client: client}

	err = adapter.Close()

	assert.NoError(t, err)
}

// =============================================================================
// foundryWorkflowAdapter.Invoke Tests
// =============================================================================

func TestFoundryWorkflowAdapter_Invoke_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify request
		assert.Equal(t, http.MethodPost, r.Method)
		assert.Equal(t, "application/json", r.Header.Get("Content-Type"))
		assert.Contains(t, r.Header.Get("Authorization"), "Bearer")

		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)

		// Foundry SSE format
		event := foundry.Event{
			Type:  foundry.EventOutputTextDelta,
			Delta: "Hello from Foundry!",
		}
		w.Write([]byte(createFoundrySSEEvent("", event)))

		completedEvent := foundry.Event{
			Type: foundry.EventResponseCompleted,
			Response: &foundry.Response{
				ID:     "resp-123",
				Status: "completed",
			},
		}
		w.Write([]byte(createFoundrySSEEvent("", completedEvent)))
		w.Write([]byte(createFoundrySSEDone()))
	}))
	defer server.Close()

	config := &foundry.WorkflowClientConfig{
		ProjectEndpoint: server.URL,
		AgentName:       "test-agent",
		APIToken:        "test-token",
	}
	client, err := foundry.NewWorkflowClient(config)
	require.NoError(t, err)
	// Inject test HTTP client
	client.SetHTTPClient(server.Client())

	adapter := &foundryWorkflowAdapter{
		client:        client,
		fileConverter: foundry.NewFileConverter(),
	}

	req := &InvokeRequest{
		ConversationID: "conv-123",
		Message:        "Hello",
	}

	resp, err := adapter.Invoke(context.Background(), req)

	require.NoError(t, err)
	require.NotNil(t, resp)
	assert.Equal(t, "Hello from Foundry!", resp.Content)
}

func TestFoundryWorkflowAdapter_Invoke_WithContextData(t *testing.T) {
	var receivedBody map[string]interface{}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		json.Unmarshal(body, &receivedBody)

		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)

		event := foundry.Event{
			Type:  foundry.EventOutputTextDelta,
			Delta: "Response",
		}
		w.Write([]byte(createFoundrySSEEvent("", event)))
		w.Write([]byte(createFoundrySSEDone()))
	}))
	defer server.Close()

	config := &foundry.WorkflowClientConfig{
		ProjectEndpoint: server.URL,
		AgentName:       "test-agent",
		APIToken:        "test-token",
	}
	client, err := foundry.NewWorkflowClient(config)
	require.NoError(t, err)
	client.SetHTTPClient(server.Client())

	adapter := &foundryWorkflowAdapter{
		client:        client,
		fileConverter: foundry.NewFileConverter(),
	}

	req := &InvokeRequest{
		ConversationID: "conv-123",
		Message:        "Hello",
		ContextData: map[string]string{
			"user_id": "user-789",
		},
	}

	resp, err := adapter.Invoke(context.Background(), req)

	require.NoError(t, err)
	require.NotNil(t, resp)

	// Verify input contains context data prepended to message
	input, ok := receivedBody["input"].(string)
	if ok {
		assert.Contains(t, input, "user_id")
	}
}

func TestFoundryWorkflowAdapter_Invoke_WithFiles(t *testing.T) {
	var receivedBody map[string]interface{}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		json.Unmarshal(body, &receivedBody)

		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)

		event := foundry.Event{
			Type:  foundry.EventOutputTextDelta,
			Delta: "Image analyzed",
		}
		w.Write([]byte(createFoundrySSEEvent("", event)))
		w.Write([]byte(createFoundrySSEDone()))
	}))
	defer server.Close()

	config := &foundry.WorkflowClientConfig{
		ProjectEndpoint: server.URL,
		AgentName:       "test-agent",
		APIToken:        "test-token",
	}
	client, err := foundry.NewWorkflowClient(config)
	require.NoError(t, err)
	client.SetHTTPClient(server.Client())

	adapter := &foundryWorkflowAdapter{
		client:        client,
		fileConverter: foundry.NewFileConverter(),
	}

	req := &InvokeRequest{
		ConversationID: "conv-123",
		Message:        "Analyze this image",
		Files: []FileInput{
			{
				Type:     "image",
				ImageURL: "data:image/png;base64,iVBORw0KGgo=",
				Filename: "test.png",
				MimeType: "image/png",
			},
		},
	}

	resp, err := adapter.Invoke(context.Background(), req)

	require.NoError(t, err)
	require.NotNil(t, resp)
	assert.Equal(t, "Image analyzed", resp.Content)

	// Verify input contains multimodal content
	input, ok := receivedBody["input"].([]interface{})
	if ok {
		assert.Greater(t, len(input), 0)
	}
}

func TestFoundryWorkflowAdapter_Invoke_HTTPError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte("Unauthorized"))
	}))
	defer server.Close()

	config := &foundry.WorkflowClientConfig{
		ProjectEndpoint: server.URL,
		AgentName:       "test-agent",
		APIToken:        "invalid-token",
	}
	client, err := foundry.NewWorkflowClient(config)
	require.NoError(t, err)
	client.SetHTTPClient(server.Client())

	adapter := &foundryWorkflowAdapter{
		client:        client,
		fileConverter: foundry.NewFileConverter(),
	}

	req := &InvokeRequest{
		Message: "Hello",
	}

	resp, err := adapter.Invoke(context.Background(), req)

	require.Error(t, err)
	assert.Nil(t, resp)
	assert.Contains(t, err.Error(), "401")
}

// =============================================================================
// foundryWorkflowAdapter.InvokeStream Tests
// =============================================================================

func TestFoundryWorkflowAdapter_InvokeStream_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)

		events := []foundry.Event{
			{Type: foundry.EventOutputTextDelta, Delta: "Chunk 1"},
			{Type: foundry.EventOutputTextDelta, Delta: "Chunk 2"},
			{Type: foundry.EventOutputTextDelta, Delta: "Chunk 3"},
		}
		for _, event := range events {
			w.Write([]byte(createFoundrySSEEvent("", event)))
		}
		w.Write([]byte(createFoundrySSEDone()))
	}))
	defer server.Close()

	config := &foundry.WorkflowClientConfig{
		ProjectEndpoint: server.URL,
		AgentName:       "test-agent",
		APIToken:        "test-token",
	}
	client, err := foundry.NewWorkflowClient(config)
	require.NoError(t, err)
	client.SetHTTPClient(server.Client())

	adapter := &foundryWorkflowAdapter{
		client:        client,
		fileConverter: foundry.NewFileConverter(),
	}

	req := &InvokeRequest{
		Message: "Stream test",
	}

	ch, err := adapter.InvokeStream(context.Background(), req)

	require.NoError(t, err)
	require.NotNil(t, ch)

	var chunks []*StreamChunk //nolint:prealloc // length unknown: reading from channel
	for chunk := range ch {
		chunks = append(chunks, chunk)
	}

	assert.GreaterOrEqual(t, len(chunks), 3)
}

func TestFoundryWorkflowAdapter_InvokeStream_WithFiles(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)

		event := foundry.Event{
			Type:  foundry.EventOutputTextDelta,
			Delta: "File processed",
		}
		w.Write([]byte(createFoundrySSEEvent("", event)))
		w.Write([]byte(createFoundrySSEDone()))
	}))
	defer server.Close()

	config := &foundry.WorkflowClientConfig{
		ProjectEndpoint: server.URL,
		AgentName:       "test-agent",
		APIToken:        "test-token",
	}
	client, err := foundry.NewWorkflowClient(config)
	require.NoError(t, err)
	client.SetHTTPClient(server.Client())

	adapter := &foundryWorkflowAdapter{
		client:        client,
		fileConverter: foundry.NewFileConverter(),
	}

	req := &InvokeRequest{
		Message: "Process file",
		Files: []FileInput{
			{
				Type:     "file",
				FileData: "SGVsbG8gV29ybGQ=",
				Filename: "test.txt",
				MimeType: "text/plain",
			},
		},
	}

	ch, err := adapter.InvokeStream(context.Background(), req)

	require.NoError(t, err)
	require.NotNil(t, ch)

	var chunks []*StreamChunk //nolint:prealloc // length unknown: reading from channel
	for chunk := range ch {
		chunks = append(chunks, chunk)
	}

	assert.Greater(t, len(chunks), 0)
}

func TestFoundryWorkflowAdapter_InvokeStream_HTTPError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
	}))
	defer server.Close()

	config := &foundry.WorkflowClientConfig{
		ProjectEndpoint: server.URL,
		AgentName:       "test-agent",
		APIToken:        "test-token",
	}
	client, err := foundry.NewWorkflowClient(config)
	require.NoError(t, err)
	client.SetHTTPClient(server.Client())

	adapter := &foundryWorkflowAdapter{
		client:        client,
		fileConverter: foundry.NewFileConverter(),
	}

	req := &InvokeRequest{
		Message: "Hello",
	}

	ch, err := adapter.InvokeStream(context.Background(), req)

	require.Error(t, err)
	assert.Nil(t, ch)
}

// =============================================================================
// foundryWorkflowAdapter.InvokeStreamReader Tests
// =============================================================================

func TestFoundryWorkflowAdapter_InvokeStreamReader_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)

		events := []foundry.Event{
			{Type: foundry.EventOutputTextDelta, Delta: "Hello"},
			{Type: foundry.EventOutputTextDelta, Delta: " World"},
		}
		for _, event := range events {
			w.Write([]byte(createFoundrySSEEvent("", event)))
		}
		w.Write([]byte(createFoundrySSEDone()))
	}))
	defer server.Close()

	config := &foundry.WorkflowClientConfig{
		ProjectEndpoint: server.URL,
		AgentName:       "test-agent",
		APIToken:        "test-token",
	}
	client, err := foundry.NewWorkflowClient(config)
	require.NoError(t, err)
	client.SetHTTPClient(server.Client())

	adapter := &foundryWorkflowAdapter{
		client:        client,
		fileConverter: foundry.NewFileConverter(),
	}

	req := &InvokeRequest{
		Message: "Greet me",
	}

	reader, err := adapter.InvokeStreamReader(context.Background(), req)

	require.NoError(t, err)
	require.NotNil(t, reader)
	defer reader.Close()

	// Read chunks until EOF
	var contents []string
	for {
		chunk, err := reader.Read()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if chunk.Type == ChunkTypeContent && chunk.Content != "" {
			contents = append(contents, chunk.Content)
		}
	}

	assert.Contains(t, contents, "Hello")
	assert.Contains(t, contents, " World")
}

func TestFoundryWorkflowAdapter_InvokeStreamReader_WithContextData(t *testing.T) {
	var receivedBody map[string]interface{}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		json.Unmarshal(body, &receivedBody)

		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)

		event := foundry.Event{
			Type:  foundry.EventOutputTextDelta,
			Delta: "Response",
		}
		w.Write([]byte(createFoundrySSEEvent("", event)))
		w.Write([]byte(createFoundrySSEDone()))
	}))
	defer server.Close()

	config := &foundry.WorkflowClientConfig{
		ProjectEndpoint: server.URL,
		AgentName:       "test-agent",
		APIToken:        "test-token",
	}
	client, err := foundry.NewWorkflowClient(config)
	require.NoError(t, err)
	client.SetHTTPClient(server.Client())

	adapter := &foundryWorkflowAdapter{
		client:        client,
		fileConverter: foundry.NewFileConverter(),
	}

	req := &InvokeRequest{
		Message: "Test message",
		ContextData: map[string]string{
			"region": "us-west",
		},
	}

	reader, err := adapter.InvokeStreamReader(context.Background(), req)

	require.NoError(t, err)
	require.NotNil(t, reader)
	defer reader.Close()

	// Verify context was prepended to input
	input, ok := receivedBody["input"].(string)
	if ok {
		assert.Contains(t, input, "region")
	}
}

func TestFoundryWorkflowAdapter_InvokeStreamReader_HTTPError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable)
	}))
	defer server.Close()

	config := &foundry.WorkflowClientConfig{
		ProjectEndpoint: server.URL,
		AgentName:       "test-agent",
		APIToken:        "test-token",
	}
	client, err := foundry.NewWorkflowClient(config)
	require.NoError(t, err)
	client.SetHTTPClient(server.Client())

	adapter := &foundryWorkflowAdapter{
		client:        client,
		fileConverter: foundry.NewFileConverter(),
	}

	req := &InvokeRequest{
		Message: "Test",
	}

	reader, err := adapter.InvokeStreamReader(context.Background(), req)

	require.Error(t, err)
	assert.Nil(t, reader)
}

// =============================================================================
// foundryStreamReaderAdapter Tests
// =============================================================================

// mockFoundryStreamReader is a mock implementation of foundry.StreamReader for testing
type mockFoundryStreamReader struct {
	chunks     []*foundry.StreamChunk
	index      int
	closeError error
}

func (m *mockFoundryStreamReader) Read() (*foundry.StreamChunk, error) {
	if m.index >= len(m.chunks) {
		return nil, io.EOF
	}
	chunk := m.chunks[m.index]
	m.index++
	return chunk, nil
}

func (m *mockFoundryStreamReader) Close() error {
	return m.closeError
}

func TestFoundryStreamReaderAdapter_Read_Success(t *testing.T) {
	mockReader := &mockFoundryStreamReader{
		chunks: []*foundry.StreamChunk{
			{Type: foundry.ChunkTypeContent, Content: "Hello", ExecutionID: "exec-1"},
			{Type: foundry.ChunkTypeMetadata, Metadata: map[string]interface{}{"agent": "foundry"}},
			{Type: foundry.ChunkTypeNewMessage},
		},
	}

	adapter := &foundryStreamReaderAdapter{reader: mockReader}

	// Read first chunk
	chunk1, err := adapter.Read()
	require.NoError(t, err)
	assert.Equal(t, ChunkTypeContent, chunk1.Type)
	assert.Equal(t, "Hello", chunk1.Content)
	assert.Equal(t, "exec-1", chunk1.ExecutionID)

	// Read second chunk
	chunk2, err := adapter.Read()
	require.NoError(t, err)
	assert.Equal(t, ChunkTypeMetadata, chunk2.Type)
	assert.Equal(t, "foundry", chunk2.Metadata["agent"])

	// Read third chunk
	chunk3, err := adapter.Read()
	require.NoError(t, err)
	assert.Equal(t, ChunkTypeNewMessage, chunk3.Type)
}

func TestFoundryStreamReaderAdapter_Read_EOF(t *testing.T) {
	mockReader := &mockFoundryStreamReader{
		chunks: []*foundry.StreamChunk{},
	}

	adapter := &foundryStreamReaderAdapter{reader: mockReader}

	chunk, err := adapter.Read()

	assert.Equal(t, io.EOF, err)
	assert.Nil(t, chunk)
}

func TestFoundryStreamReaderAdapter_Close_Success(t *testing.T) {
	mockReader := &mockFoundryStreamReader{
		closeError: nil,
	}

	adapter := &foundryStreamReaderAdapter{reader: mockReader}

	err := adapter.Close()

	assert.NoError(t, err)
}

func TestFoundryStreamReaderAdapter_Close_Error(t *testing.T) {
	mockReader := &mockFoundryStreamReader{
		closeError: errors.New("close failed"),
	}

	adapter := &foundryStreamReaderAdapter{reader: mockReader}

	err := adapter.Close()

	assert.Error(t, err)
	assert.Equal(t, "close failed", err.Error())
}

// =============================================================================
// foundryWorkflowAdapter.Close Tests
// =============================================================================

func TestFoundryWorkflowAdapter_Close_Success(t *testing.T) {
	config := &foundry.WorkflowClientConfig{
		ProjectEndpoint: "https://example.com",
		AgentName:       "test-agent",
		APIToken:        "test-token",
	}
	client, err := foundry.NewWorkflowClient(config)
	require.NoError(t, err)

	adapter := &foundryWorkflowAdapter{
		client:        client,
		fileConverter: foundry.NewFileConverter(),
	}

	err = adapter.Close()

	assert.NoError(t, err)
}

// =============================================================================
// Integration Tests - Multiple Adapters
// =============================================================================

func TestAdapters_ConsistentChunkTypeMapping(t *testing.T) {
	// Verify that chunk types are consistently mapped across adapters

	// N8N chunk
	n8nChunk := &n8n.StreamChunk{
		Type:        n8n.ChunkTypeContent,
		Content:     "test",
		ExecutionID: "exec-1",
	}
	convertedN8N := convertN8NChunk(n8nChunk)
	assert.Equal(t, ChunkTypeContent, convertedN8N.Type)

	// Foundry chunk
	foundryChunk := &foundry.StreamChunk{
		Type:        foundry.ChunkTypeContent,
		Content:     "test",
		ExecutionID: "exec-2",
	}
	convertedFoundry := convertFoundryChunk(foundryChunk)
	assert.Equal(t, ChunkTypeContent, convertedFoundry.Type)

	// Both should produce consistent types
	assert.Equal(t, convertedN8N.Type, convertedFoundry.Type)
}

func TestAdapters_AllChunkTypeMappings(t *testing.T) {
	testCases := []struct {
		name         string
		n8nType      n8n.ChunkType
		foundryType  foundry.ChunkType
		expectedType ChunkType
	}{
		{"Content", n8n.ChunkTypeContent, foundry.ChunkTypeContent, ChunkTypeContent},
		{"Metadata", n8n.ChunkTypeMetadata, foundry.ChunkTypeMetadata, ChunkTypeMetadata},
		{"Error", n8n.ChunkTypeError, foundry.ChunkTypeError, ChunkTypeError},
		{"Done", n8n.ChunkTypeDone, foundry.ChunkTypeDone, ChunkTypeDone},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			n8nChunk := convertN8NChunk(&n8n.StreamChunk{Type: tc.n8nType})
			foundryChunk := convertFoundryChunk(&foundry.StreamChunk{Type: tc.foundryType})

			assert.Equal(t, tc.expectedType, n8nChunk.Type)
			assert.Equal(t, tc.expectedType, foundryChunk.Type)
		})
	}
}

// =============================================================================
// Timeout and Context Tests
// =============================================================================

func TestN8NWorkflowAdapter_Invoke_ContextCancellation(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Simulate slow response
		time.Sleep(100 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	config := &n8n.ChatWorkflowConfig{
		ChatURL:    server.URL,
		HTTPClient: server.Client(),
	}
	client, err := n8n.NewChatWorkflowClient(config)
	require.NoError(t, err)

	adapter := &n8nWorkflowAdapter{
		client:        client,
		fileConverter: n8n.NewFileConverter(),
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
	defer cancel()

	req := &InvokeRequest{
		Message: "Hello",
	}

	_, err = adapter.Invoke(ctx, req)

	require.Error(t, err)
}

func TestFoundryWorkflowAdapter_Invoke_ContextCancellation(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Simulate slow response
		time.Sleep(100 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	config := &foundry.WorkflowClientConfig{
		ProjectEndpoint: server.URL,
		AgentName:       "test-agent",
		APIToken:        "test-token",
	}
	client, err := foundry.NewWorkflowClient(config)
	require.NoError(t, err)
	client.SetHTTPClient(server.Client())

	adapter := &foundryWorkflowAdapter{
		client:        client,
		fileConverter: foundry.NewFileConverter(),
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
	defer cancel()

	req := &InvokeRequest{
		Message: "Hello",
	}

	_, err = adapter.Invoke(ctx, req)

	require.Error(t, err)
}

// =============================================================================
// Additional Coverage Tests
// =============================================================================

func TestN8NWorkflowAdapter_InvokeStreamReader_WithFiles(t *testing.T) {
	events := []n8n.StreamEvent{
		{Type: n8n.StreamTypeItem, Content: "File processed"},
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(createN8NSSEResponse(events)))
	}))
	defer server.Close()

	config := &n8n.ChatWorkflowConfig{
		ChatURL:    server.URL,
		HTTPClient: server.Client(),
	}
	client, err := n8n.NewChatWorkflowClient(config)
	require.NoError(t, err)

	adapter := &n8nWorkflowAdapter{
		client:        client,
		fileConverter: n8n.NewFileConverter(),
	}

	req := &InvokeRequest{
		Message: "Process file",
		Files: []FileInput{
			{
				Type:     "file",
				FileData: "SGVsbG8gV29ybGQ=",
				Filename: "test.txt",
				MimeType: "text/plain",
			},
		},
	}

	reader, err := adapter.InvokeStreamReader(context.Background(), req)

	require.NoError(t, err)
	require.NotNil(t, reader)
	defer reader.Close()

	chunk, err := reader.Read()
	require.NoError(t, err)
	assert.Equal(t, "File processed", chunk.Content)
}

func TestFoundryWorkflowAdapter_InvokeStreamReader_WithFiles(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)

		event := foundry.Event{
			Type:  foundry.EventOutputTextDelta,
			Delta: "File analyzed",
		}
		w.Write([]byte(createFoundrySSEEvent("", event)))
		w.Write([]byte(createFoundrySSEDone()))
	}))
	defer server.Close()

	config := &foundry.WorkflowClientConfig{
		ProjectEndpoint: server.URL,
		AgentName:       "test-agent",
		APIToken:        "test-token",
	}
	client, err := foundry.NewWorkflowClient(config)
	require.NoError(t, err)
	client.SetHTTPClient(server.Client())

	adapter := &foundryWorkflowAdapter{
		client:        client,
		fileConverter: foundry.NewFileConverter(),
	}

	req := &InvokeRequest{
		Message: "Analyze file",
		Files: []FileInput{
			{
				Type:     "file",
				FileData: "cGRmY29udGVudA==",
				Filename: "doc.pdf",
				MimeType: "application/pdf",
			},
		},
	}

	reader, err := adapter.InvokeStreamReader(context.Background(), req)

	require.NoError(t, err)
	require.NotNil(t, reader)
	defer reader.Close()

	// Read chunks
	var contents []string
	for {
		chunk, err := reader.Read()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if chunk.Content != "" {
			contents = append(contents, chunk.Content)
		}
	}

	assert.Contains(t, contents, "File analyzed")
}

func TestN8NWorkflowAdapter_Invoke_MultipleChunks(t *testing.T) {
	events := []n8n.StreamEvent{
		{Type: n8n.StreamTypeItem, Content: "Hello, "},
		{Type: n8n.StreamTypeItem, Content: "World!"},
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(createN8NSSEResponse(events)))
	}))
	defer server.Close()

	config := &n8n.ChatWorkflowConfig{
		ChatURL:    server.URL,
		HTTPClient: server.Client(),
	}
	client, err := n8n.NewChatWorkflowClient(config)
	require.NoError(t, err)

	adapter := &n8nWorkflowAdapter{
		client:        client,
		fileConverter: n8n.NewFileConverter(),
	}

	req := &InvokeRequest{
		Message: "Greet",
	}

	resp, err := adapter.Invoke(context.Background(), req)

	require.NoError(t, err)
	require.NotNil(t, resp)
	assert.Equal(t, "Hello, World!", resp.Content)
}

func TestFoundryWorkflowAdapter_Invoke_MultipleChunks(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)

		events := []foundry.Event{
			{Type: foundry.EventOutputTextDelta, Delta: "Hello, "},
			{Type: foundry.EventOutputTextDelta, Delta: "World!"},
		}
		for _, event := range events {
			w.Write([]byte(createFoundrySSEEvent("", event)))
		}
		w.Write([]byte(createFoundrySSEDone()))
	}))
	defer server.Close()

	config := &foundry.WorkflowClientConfig{
		ProjectEndpoint: server.URL,
		AgentName:       "test-agent",
		APIToken:        "test-token",
	}
	client, err := foundry.NewWorkflowClient(config)
	require.NoError(t, err)
	client.SetHTTPClient(server.Client())

	adapter := &foundryWorkflowAdapter{
		client:        client,
		fileConverter: foundry.NewFileConverter(),
	}

	req := &InvokeRequest{
		Message: "Greet",
	}

	resp, err := adapter.Invoke(context.Background(), req)

	require.NoError(t, err)
	require.NotNil(t, resp)
	assert.Equal(t, "Hello, World!", resp.Content)
}

func TestN8NWorkflowAdapter_InvokeStream_ContextCancellation(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(100 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	config := &n8n.ChatWorkflowConfig{
		ChatURL:    server.URL,
		HTTPClient: server.Client(),
	}
	client, err := n8n.NewChatWorkflowClient(config)
	require.NoError(t, err)

	adapter := &n8nWorkflowAdapter{
		client:        client,
		fileConverter: n8n.NewFileConverter(),
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
	defer cancel()

	req := &InvokeRequest{
		Message: "Hello",
	}

	_, err = adapter.InvokeStream(ctx, req)

	require.Error(t, err)
}

func TestFoundryWorkflowAdapter_InvokeStream_ContextCancellation(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(100 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	config := &foundry.WorkflowClientConfig{
		ProjectEndpoint: server.URL,
		AgentName:       "test-agent",
		APIToken:        "test-token",
	}
	client, err := foundry.NewWorkflowClient(config)
	require.NoError(t, err)
	client.SetHTTPClient(server.Client())

	adapter := &foundryWorkflowAdapter{
		client:        client,
		fileConverter: foundry.NewFileConverter(),
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
	defer cancel()

	req := &InvokeRequest{
		Message: "Hello",
	}

	_, err = adapter.InvokeStream(ctx, req)

	require.Error(t, err)
}

func TestN8NWorkflowAdapter_InvokeStreamReader_ContextCancellation(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(100 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	config := &n8n.ChatWorkflowConfig{
		ChatURL:    server.URL,
		HTTPClient: server.Client(),
	}
	client, err := n8n.NewChatWorkflowClient(config)
	require.NoError(t, err)

	adapter := &n8nWorkflowAdapter{
		client:        client,
		fileConverter: n8n.NewFileConverter(),
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
	defer cancel()

	req := &InvokeRequest{
		Message: "Hello",
	}

	_, err = adapter.InvokeStreamReader(ctx, req)

	require.Error(t, err)
}

func TestFoundryWorkflowAdapter_InvokeStreamReader_ContextCancellation(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(100 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	config := &foundry.WorkflowClientConfig{
		ProjectEndpoint: server.URL,
		AgentName:       "test-agent",
		APIToken:        "test-token",
	}
	client, err := foundry.NewWorkflowClient(config)
	require.NoError(t, err)
	client.SetHTTPClient(server.Client())

	adapter := &foundryWorkflowAdapter{
		client:        client,
		fileConverter: foundry.NewFileConverter(),
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
	defer cancel()

	req := &InvokeRequest{
		Message: "Hello",
	}

	_, err = adapter.InvokeStreamReader(ctx, req)

	require.Error(t, err)
}
