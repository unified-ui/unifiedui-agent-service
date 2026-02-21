package n8n

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/unifiedui/agent-service/internal/domain/models"
)

// =============================================================================
// Helper Functions
// =============================================================================

// createSSEResponse creates a Server-Sent Events formatted response
func createSSEResponse(events []N8NStreamEvent) string {
	var sb strings.Builder
	for _, event := range events {
		data, _ := json.Marshal(event)
		sb.Write(data)
		sb.WriteString("\n")
	}
	return sb.String()
}

// createTestServer creates a test HTTP server with the given handler
func createTestServer(handler http.HandlerFunc) *httptest.Server {
	return httptest.NewServer(handler)
}

// =============================================================================
// NewChatWorkflowClient Tests
// =============================================================================

func TestNewChatWorkflowClient_ValidConfig(t *testing.T) {
	config := &ChatWorkflowConfig{
		ChatURL:               "https://n8n.example.com/webhook/chat",
		Username:              "user",
		Password:              "pass",
		UseUnifiedChatHistory: true,
	}

	client, err := NewChatWorkflowClient(config)

	require.NoError(t, err)
	require.NotNil(t, client)
	assert.Equal(t, "https://n8n.example.com/webhook/chat", client.chatURL)
	assert.Equal(t, "user", client.username)
	assert.Equal(t, "pass", client.password)
	assert.True(t, client.useUnifiedChatHistory)
	assert.NotNil(t, client.httpClient)
}

func TestNewChatWorkflowClient_NilConfig(t *testing.T) {
	client, err := NewChatWorkflowClient(nil)

	require.Error(t, err)
	assert.Nil(t, client)
	assert.Contains(t, err.Error(), "config is required")
}

func TestNewChatWorkflowClient_EmptyChatURL(t *testing.T) {
	config := &ChatWorkflowConfig{
		ChatURL:  "",
		Username: "user",
		Password: "pass",
	}

	client, err := NewChatWorkflowClient(config)

	require.Error(t, err)
	assert.Nil(t, client)
	assert.Contains(t, err.Error(), "chat URL is required")
}

func TestNewChatWorkflowClient_CustomHTTPClient(t *testing.T) {
	customClient := &http.Client{Timeout: 10 * time.Second}
	config := &ChatWorkflowConfig{
		ChatURL:    "https://n8n.example.com/webhook/chat",
		HTTPClient: customClient,
	}

	client, err := NewChatWorkflowClient(config)

	require.NoError(t, err)
	require.NotNil(t, client)
	assert.Equal(t, customClient, client.httpClient)
}

func TestNewChatWorkflowClient_WithoutCredentials(t *testing.T) {
	config := &ChatWorkflowConfig{
		ChatURL: "https://n8n.example.com/webhook/chat",
	}

	client, err := NewChatWorkflowClient(config)

	require.NoError(t, err)
	require.NotNil(t, client)
	assert.Empty(t, client.username)
	assert.Empty(t, client.password)
}

// =============================================================================
// Invoke Tests - Valid Responses
// =============================================================================

func TestInvoke_ValidResponse_SingleChunk(t *testing.T) {
	events := []N8NStreamEvent{
		{Type: N8NStreamTypeItem, Content: "Hello, how can I help you?"},
	}

	server := createTestServer(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(createSSEResponse(events)))
	})
	defer server.Close()

	config := &ChatWorkflowConfig{
		ChatURL:    server.URL,
		HTTPClient: server.Client(),
	}
	client, err := NewChatWorkflowClient(config)
	require.NoError(t, err)

	req := &InvokeRequest{
		ConversationID: "conv-123",
		Message:        "Hello",
		SessionID:      "session-456",
	}

	resp, err := client.Invoke(context.Background(), req)

	require.NoError(t, err)
	require.NotNil(t, resp)
	assert.Equal(t, "Hello, how can I help you?", resp.Content)
	assert.Equal(t, "session-456", resp.SessionID)
}

func TestInvoke_ValidResponse_MultipleChunks(t *testing.T) {
	events := []N8NStreamEvent{
		{Type: N8NStreamTypeBegin},
		{Type: N8NStreamTypeItem, Content: "Hello, "},
		{Type: N8NStreamTypeItem, Content: "I am an AI assistant. "},
		{Type: N8NStreamTypeItem, Content: "How can I help you today?"},
		{Type: N8NStreamTypeEnd},
	}

	server := createTestServer(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(createSSEResponse(events)))
	})
	defer server.Close()

	config := &ChatWorkflowConfig{
		ChatURL:    server.URL,
		HTTPClient: server.Client(),
	}
	client, err := NewChatWorkflowClient(config)
	require.NoError(t, err)

	req := &InvokeRequest{
		Message:   "Hello",
		SessionID: "session-789",
	}

	resp, err := client.Invoke(context.Background(), req)

	require.NoError(t, err)
	require.NotNil(t, resp)
	assert.Equal(t, "Hello, I am an AI assistant. How can I help you today?", resp.Content)
}

func TestInvoke_WithMetadata(t *testing.T) {
	events := []N8NStreamEvent{
		{Type: N8NStreamTypeItem, Content: "Response content"},
		{Type: N8NStreamTypeItem, Content: `{"executionId":"exec-123"}`},
	}

	server := createTestServer(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(createSSEResponse(events)))
	})
	defer server.Close()

	config := &ChatWorkflowConfig{
		ChatURL:    server.URL,
		HTTPClient: server.Client(),
	}
	client, err := NewChatWorkflowClient(config)
	require.NoError(t, err)

	req := &InvokeRequest{
		Message:   "Test message",
		SessionID: "session-test",
	}

	resp, err := client.Invoke(context.Background(), req)

	require.NoError(t, err)
	require.NotNil(t, resp)
	assert.Equal(t, "Response content", resp.Content)
	assert.Equal(t, "exec-123", resp.ExecutionID)
}

func TestInvoke_WithChatHistory(t *testing.T) {
	var receivedBody []byte

	server := createTestServer(func(w http.ResponseWriter, r *http.Request) {
		receivedBody, _ = io.ReadAll(r.Body)
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)
		events := []N8NStreamEvent{
			{Type: N8NStreamTypeItem, Content: "Response"},
		}
		w.Write([]byte(createSSEResponse(events)))
	})
	defer server.Close()

	config := &ChatWorkflowConfig{
		ChatURL:               server.URL,
		HTTPClient:            server.Client(),
		UseUnifiedChatHistory: true,
	}
	client, err := NewChatWorkflowClient(config)
	require.NoError(t, err)

	req := &InvokeRequest{
		Message:   "Current message",
		SessionID: "session-test",
		ChatHistory: []models.ChatHistoryEntry{
			{Role: "user", Content: "Previous user message", Timestamp: time.Now().Add(-time.Minute)},
			{Role: "assistant", Content: "Previous assistant response", Timestamp: time.Now().Add(-30 * time.Second)},
		},
	}

	resp, err := client.Invoke(context.Background(), req)

	require.NoError(t, err)
	require.NotNil(t, resp)

	// Verify the chat history was included in the request
	var chatReq ChatRequest
	err = json.Unmarshal(receivedBody, &chatReq)
	require.NoError(t, err)
	assert.Contains(t, chatReq.ChatInput, "<history>")
	assert.Contains(t, chatReq.ChatInput, "Previous user message")
	assert.Contains(t, chatReq.ChatInput, "Previous assistant response")
	assert.Contains(t, chatReq.ChatInput, "Current message")
}

// =============================================================================
// InvokeStream Tests - SSE Responses
// =============================================================================

func TestInvokeStream_ValidSSEResponse(t *testing.T) {
	events := []N8NStreamEvent{
		{Type: N8NStreamTypeBegin},
		{Type: N8NStreamTypeItem, Content: "First chunk"},
		{Type: N8NStreamTypeItem, Content: "Second chunk"},
		{Type: N8NStreamTypeItem, Content: "Third chunk"},
		{Type: N8NStreamTypeEnd},
	}

	server := createTestServer(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(createSSEResponse(events)))
	})
	defer server.Close()

	config := &ChatWorkflowConfig{
		ChatURL:    server.URL,
		HTTPClient: server.Client(),
	}
	client, err := NewChatWorkflowClient(config)
	require.NoError(t, err)

	req := &InvokeRequest{
		Message:   "Stream test",
		SessionID: "session-stream",
	}

	ch, err := client.InvokeStream(context.Background(), req)

	require.NoError(t, err)
	require.NotNil(t, ch)

	var chunks []*StreamChunk
	for chunk := range ch {
		chunks = append(chunks, chunk)
	}

	assert.Len(t, chunks, 3)
	assert.Equal(t, "First chunk", chunks[0].Content)
	assert.Equal(t, "Second chunk", chunks[1].Content)
	assert.Equal(t, "Third chunk", chunks[2].Content)
}

func TestInvokeStream_ContextCancellation(t *testing.T) {
	server := createTestServer(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)
		// Simulate slow response
		time.Sleep(100 * time.Millisecond)
		events := []N8NStreamEvent{
			{Type: N8NStreamTypeItem, Content: "Should not arrive"},
		}
		w.Write([]byte(createSSEResponse(events)))
	})
	defer server.Close()

	config := &ChatWorkflowConfig{
		ChatURL:    server.URL,
		HTTPClient: server.Client(),
	}
	client, err := NewChatWorkflowClient(config)
	require.NoError(t, err)

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	req := &InvokeRequest{
		Message:   "Test message",
		SessionID: "session-ctx",
	}

	ch, err := client.InvokeStream(ctx, req)

	// The request should fail due to context cancellation
	if err == nil {
		// If we got a channel, it should be closed or have an error
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
	// Use an invalid URL to simulate network error
	config := &ChatWorkflowConfig{
		ChatURL: "http://invalid.local.host:12345/webhook/chat",
		HTTPClient: &http.Client{
			Timeout: 1 * time.Second,
		},
	}
	client, err := NewChatWorkflowClient(config)
	require.NoError(t, err)

	req := &InvokeRequest{
		Message:   "Test",
		SessionID: "session-err",
	}

	resp, err := client.Invoke(context.Background(), req)

	require.Error(t, err)
	assert.Nil(t, resp)
	assert.Contains(t, err.Error(), "failed to execute request")
}

func TestInvoke_HTTPError_Unauthorized(t *testing.T) {
	server := createTestServer(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte(`{"error":"Invalid credentials"}`))
	})
	defer server.Close()

	config := &ChatWorkflowConfig{
		ChatURL:    server.URL,
		HTTPClient: server.Client(),
	}
	client, err := NewChatWorkflowClient(config)
	require.NoError(t, err)

	req := &InvokeRequest{
		Message:   "Test",
		SessionID: "session-auth",
	}

	resp, err := client.Invoke(context.Background(), req)

	require.Error(t, err)
	assert.Nil(t, resp)
	assert.Contains(t, err.Error(), "unexpected status code: 401")
}

func TestInvoke_HTTPError_InternalServerError(t *testing.T) {
	server := createTestServer(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(`{"error":"Internal server error"}`))
	})
	defer server.Close()

	config := &ChatWorkflowConfig{
		ChatURL:    server.URL,
		HTTPClient: server.Client(),
	}
	client, err := NewChatWorkflowClient(config)
	require.NoError(t, err)

	req := &InvokeRequest{
		Message:   "Test",
		SessionID: "session-500",
	}

	resp, err := client.Invoke(context.Background(), req)

	require.Error(t, err)
	assert.Nil(t, resp)
	assert.Contains(t, err.Error(), "unexpected status code: 500")
}

func TestInvoke_HTTPError_BadRequest(t *testing.T) {
	server := createTestServer(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(`{"error":"Invalid request"}`))
	})
	defer server.Close()

	config := &ChatWorkflowConfig{
		ChatURL:    server.URL,
		HTTPClient: server.Client(),
	}
	client, err := NewChatWorkflowClient(config)
	require.NoError(t, err)

	req := &InvokeRequest{
		Message:   "Test",
		SessionID: "session-400",
	}

	resp, err := client.Invoke(context.Background(), req)

	require.Error(t, err)
	assert.Nil(t, resp)
	assert.Contains(t, err.Error(), "unexpected status code: 400")
}

func TestInvokeStream_InvalidJSON(t *testing.T) {
	server := createTestServer(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)
		// Send invalid JSON
		w.Write([]byte("invalid json content\n"))
		w.Write([]byte("{\"type\":\"item\",\"content\":\"valid content\"}\n"))
	})
	defer server.Close()

	config := &ChatWorkflowConfig{
		ChatURL:    server.URL,
		HTTPClient: server.Client(),
	}
	client, err := NewChatWorkflowClient(config)
	require.NoError(t, err)

	req := &InvokeRequest{
		Message:   "Test",
		SessionID: "session-json",
	}

	ch, err := client.InvokeStream(context.Background(), req)
	require.NoError(t, err)

	var chunks []*StreamChunk
	for chunk := range ch {
		chunks = append(chunks, chunk)
	}

	// Should skip invalid JSON and process valid content
	require.Len(t, chunks, 1)
	assert.Equal(t, "valid content", chunks[0].Content)
}

func TestInvokeStream_EmptyResponse(t *testing.T) {
	server := createTestServer(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)
		// Send only empty lines
		w.Write([]byte("\n\n\n"))
	})
	defer server.Close()

	config := &ChatWorkflowConfig{
		ChatURL:    server.URL,
		HTTPClient: server.Client(),
	}
	client, err := NewChatWorkflowClient(config)
	require.NoError(t, err)

	req := &InvokeRequest{
		Message:   "Test",
		SessionID: "session-empty",
	}

	ch, err := client.InvokeStream(context.Background(), req)
	require.NoError(t, err)

	var chunks []*StreamChunk
	for chunk := range ch {
		chunks = append(chunks, chunk)
	}

	assert.Empty(t, chunks)
}

// =============================================================================
// Authentication Tests
// =============================================================================

func TestInvoke_BasicAuth(t *testing.T) {
	var receivedAuth string

	server := createTestServer(func(w http.ResponseWriter, r *http.Request) {
		receivedAuth = r.Header.Get("Authorization")
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)
		events := []N8NStreamEvent{
			{Type: N8NStreamTypeItem, Content: "Authenticated response"},
		}
		w.Write([]byte(createSSEResponse(events)))
	})
	defer server.Close()

	config := &ChatWorkflowConfig{
		ChatURL:    server.URL,
		Username:   "testuser",
		Password:   "testpass",
		HTTPClient: server.Client(),
	}
	client, err := NewChatWorkflowClient(config)
	require.NoError(t, err)

	req := &InvokeRequest{
		Message:   "Test auth",
		SessionID: "session-auth",
	}

	resp, err := client.Invoke(context.Background(), req)

	require.NoError(t, err)
	require.NotNil(t, resp)
	assert.True(t, strings.HasPrefix(receivedAuth, "Basic "))
}

func TestInvoke_NoAuth(t *testing.T) {
	var receivedAuth string

	server := createTestServer(func(w http.ResponseWriter, r *http.Request) {
		receivedAuth = r.Header.Get("Authorization")
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)
		events := []N8NStreamEvent{
			{Type: N8NStreamTypeItem, Content: "Response without auth"},
		}
		w.Write([]byte(createSSEResponse(events)))
	})
	defer server.Close()

	config := &ChatWorkflowConfig{
		ChatURL:    server.URL,
		HTTPClient: server.Client(),
	}
	client, err := NewChatWorkflowClient(config)
	require.NoError(t, err)

	req := &InvokeRequest{
		Message:   "Test no auth",
		SessionID: "session-noauth",
	}

	resp, err := client.Invoke(context.Background(), req)

	require.NoError(t, err)
	require.NotNil(t, resp)
	assert.Empty(t, receivedAuth)
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
		events := []N8NStreamEvent{
			{Type: N8NStreamTypeItem, Content: "Response"},
		}
		w.Write([]byte(createSSEResponse(events)))
	})
	defer server.Close()

	config := &ChatWorkflowConfig{
		ChatURL:    server.URL,
		HTTPClient: server.Client(),
	}
	client, err := NewChatWorkflowClient(config)
	require.NoError(t, err)

	req := &InvokeRequest{
		Message:   "Test headers",
		SessionID: "session-headers",
	}

	_, err = client.Invoke(context.Background(), req)
	require.NoError(t, err)

	assert.Equal(t, "application/json", receivedHeaders.Get("Content-Type"))
	assert.Equal(t, "text/event-stream", receivedHeaders.Get("Accept"))
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
		events := []N8NStreamEvent{
			{Type: N8NStreamTypeItem, Content: "Response"},
		}
		w.Write([]byte(createSSEResponse(events)))
	})
	defer server.Close()

	config := &ChatWorkflowConfig{
		ChatURL:    server.URL,
		HTTPClient: server.Client(),
	}
	client, err := NewChatWorkflowClient(config)
	require.NoError(t, err)

	req := &InvokeRequest{
		Message:   "Test message content",
		SessionID: "session-body",
	}

	_, err = client.Invoke(context.Background(), req)
	require.NoError(t, err)

	var chatReq ChatRequest
	err = json.Unmarshal(receivedBody, &chatReq)
	require.NoError(t, err)
	assert.Equal(t, "Test message content", chatReq.ChatInput)
	assert.Equal(t, "session-body", chatReq.SessionID)
}

func TestInvoke_WithFileAttachments(t *testing.T) {
	var receivedBody []byte

	server := createTestServer(func(w http.ResponseWriter, r *http.Request) {
		receivedBody, _ = io.ReadAll(r.Body)
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)
		events := []N8NStreamEvent{
			{Type: N8NStreamTypeItem, Content: "Response with files"},
		}
		w.Write([]byte(createSSEResponse(events)))
	})
	defer server.Close()

	config := &ChatWorkflowConfig{
		ChatURL:    server.URL,
		HTTPClient: server.Client(),
	}
	client, err := NewChatWorkflowClient(config)
	require.NoError(t, err)

	reqWithFiles := &ChatRequestWithFiles{
		ChatInput: "Message with files",
		SessionID: "session-files",
		Files: []FileAttachment{
			{
				Type:     "image",
				Data:     "base64encodeddata",
				Filename: "test.png",
				MimeType: "image/png",
			},
		},
	}

	req := &InvokeRequest{
		Message:   "Message with files",
		SessionID: "session-files",
		Input:     reqWithFiles,
	}

	resp, err := client.Invoke(context.Background(), req)
	require.NoError(t, err)
	require.NotNil(t, resp)

	var chatReqWithFiles ChatRequestWithFiles
	err = json.Unmarshal(receivedBody, &chatReqWithFiles)
	require.NoError(t, err)
	assert.Len(t, chatReqWithFiles.Files, 1)
	assert.Equal(t, "test.png", chatReqWithFiles.Files[0].Filename)
}

// =============================================================================
// StreamReader Tests
// =============================================================================

func TestStreamReader_Read_ValidContent(t *testing.T) {
	events := []N8NStreamEvent{
		{Type: N8NStreamTypeItem, Content: "Test content"},
	}
	responseBody := createSSEResponse(events)

	// Create mock response with proper body handling
	server := createTestServer(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(responseBody))
	})
	defer server.Close()

	config := &ChatWorkflowConfig{
		ChatURL:    server.URL,
		HTTPClient: server.Client(),
	}
	client, err := NewChatWorkflowClient(config)
	require.NoError(t, err)

	req := &InvokeRequest{
		Message:   "Test",
		SessionID: "session-reader",
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
		// Empty response
	})
	defer server.Close()

	config := &ChatWorkflowConfig{
		ChatURL:    server.URL,
		HTTPClient: server.Client(),
	}
	client, err := NewChatWorkflowClient(config)
	require.NoError(t, err)

	req := &InvokeRequest{
		Message:   "Test",
		SessionID: "session-eof",
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
		events := []N8NStreamEvent{
			{Type: N8NStreamTypeItem, Content: "Content"},
		}
		w.Write([]byte(createSSEResponse(events)))
	})
	defer server.Close()

	config := &ChatWorkflowConfig{
		ChatURL:    server.URL,
		HTTPClient: server.Client(),
	}
	client, err := NewChatWorkflowClient(config)
	require.NoError(t, err)

	req := &InvokeRequest{
		Message:   "Test",
		SessionID: "session-close",
	}

	reader, err := client.InvokeStreamReader(context.Background(), req)
	require.NoError(t, err)

	err = reader.Close()
	assert.NoError(t, err)
}

// =============================================================================
// Close Tests
// =============================================================================

func TestChatWorkflowClient_Close(t *testing.T) {
	config := &ChatWorkflowConfig{
		ChatURL: "https://n8n.example.com/webhook/chat",
	}
	client, err := NewChatWorkflowClient(config)
	require.NoError(t, err)

	err = client.Close()
	assert.NoError(t, err)
}

// =============================================================================
// API Client Tests
// =============================================================================

func TestNewAPIClient_ValidConfig(t *testing.T) {
	config := &APIClientConfig{
		BaseURL: "https://n8n.example.com",
		APIKey:  "api-key-123",
	}

	client, err := NewAPIClient(config)

	require.NoError(t, err)
	require.NotNil(t, client)
	assert.Equal(t, "https://n8n.example.com", client.baseURL)
	assert.Equal(t, "api-key-123", client.apiKey)
	assert.NotNil(t, client.httpClient)
}

func TestNewAPIClient_NilConfig(t *testing.T) {
	client, err := NewAPIClient(nil)

	require.Error(t, err)
	assert.Nil(t, client)
	assert.Contains(t, err.Error(), "config is required")
}

func TestNewAPIClient_CustomHTTPClient(t *testing.T) {
	customClient := &http.Client{Timeout: 5 * time.Second}
	config := &APIClientConfig{
		BaseURL:    "https://n8n.example.com",
		APIKey:     "api-key",
		HTTPClient: customClient,
	}

	client, err := NewAPIClient(config)

	require.NoError(t, err)
	require.NotNil(t, client)
	assert.Equal(t, customClient, client.httpClient)
}

func TestGetExecution_ValidResponse(t *testing.T) {
	server := createTestServer(func(w http.ResponseWriter, r *http.Request) {
		assert.Contains(t, r.URL.Path, "/api/v1/executions/")
		assert.Equal(t, "api-key-123", r.Header.Get("X-N8N-API-KEY"))

		response := ExecutionResponse{
			ID:        "exec-123",
			Status:    "success",
			StartedAt: "2024-01-01T10:00:00Z",
			StoppedAt: "2024-01-01T10:01:00Z",
			Data:      map[string]interface{}{"result": "ok"},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	})
	defer server.Close()

	config := &APIClientConfig{
		BaseURL:    server.URL,
		APIKey:     "api-key-123",
		HTTPClient: server.Client(),
	}
	client, err := NewAPIClient(config)
	require.NoError(t, err)

	execInfo, err := client.GetExecution(context.Background(), "exec-123")

	require.NoError(t, err)
	require.NotNil(t, execInfo)
	assert.Equal(t, "exec-123", execInfo.ID)
	assert.Equal(t, "success", execInfo.Status)
	assert.Equal(t, "2024-01-01T10:00:00Z", execInfo.StartedAt)
	assert.Equal(t, "2024-01-01T10:01:00Z", execInfo.StoppedAt)
	assert.Equal(t, "ok", execInfo.Data["result"])
}

func TestGetExecution_HTTPError(t *testing.T) {
	server := createTestServer(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	})
	defer server.Close()

	config := &APIClientConfig{
		BaseURL:    server.URL,
		APIKey:     "api-key",
		HTTPClient: server.Client(),
	}
	client, err := NewAPIClient(config)
	require.NoError(t, err)

	execInfo, err := client.GetExecution(context.Background(), "nonexistent")

	require.Error(t, err)
	assert.Nil(t, execInfo)
	assert.Contains(t, err.Error(), "unexpected status code: 404")
}

func TestGetExecution_InvalidJSON(t *testing.T) {
	server := createTestServer(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("invalid json"))
	})
	defer server.Close()

	config := &APIClientConfig{
		BaseURL:    server.URL,
		APIKey:     "api-key",
		HTTPClient: server.Client(),
	}
	client, err := NewAPIClient(config)
	require.NoError(t, err)

	execInfo, err := client.GetExecution(context.Background(), "exec-123")

	require.Error(t, err)
	assert.Nil(t, execInfo)
	assert.Contains(t, err.Error(), "failed to decode response")
}

func TestGetExecutionsBySession(t *testing.T) {
	config := &APIClientConfig{
		BaseURL: "https://n8n.example.com",
		APIKey:  "api-key",
	}
	client, err := NewAPIClient(config)
	require.NoError(t, err)

	// This currently returns an empty slice
	executions, err := client.GetExecutionsBySession(context.Background(), "session-123")

	require.NoError(t, err)
	assert.Empty(t, executions)
}

func TestAPIClient_Close(t *testing.T) {
	config := &APIClientConfig{
		BaseURL: "https://n8n.example.com",
		APIKey:  "api-key",
	}
	client, err := NewAPIClient(config)
	require.NoError(t, err)

	err = client.Close()
	assert.NoError(t, err)
}

// =============================================================================
// Build Chat History Tests
// =============================================================================

func TestBuildChatHistoryMarkdown_EmptyHistory(t *testing.T) {
	now := time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC)
	result := BuildChatHistoryMarkdown(nil, "Current message", now)

	assert.Contains(t, result, "## Current Message")
	assert.Contains(t, result, "Current message")
	assert.Contains(t, result, "2024-01-15 10:30:00")
	assert.NotContains(t, result, "## Chat History")
}

func TestBuildChatHistoryMarkdown_WithHistory(t *testing.T) {
	now := time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC)
	history := []models.ChatHistoryEntry{
		{Role: "user", Content: "Hello", Timestamp: now.Add(-2 * time.Minute)},
		{Role: "assistant", Content: "Hi there!", Timestamp: now.Add(-time.Minute)},
	}
	result := BuildChatHistoryMarkdown(history, "How are you?", now)

	assert.Contains(t, result, "## Chat History")
	assert.Contains(t, result, "Hello")
	assert.Contains(t, result, "Hi there!")
	assert.Contains(t, result, "## Current Message")
	assert.Contains(t, result, "How are you?")
}

func TestBuildSimpleChatHistoryMarkdown_EmptyHistory(t *testing.T) {
	now := time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC)
	result := BuildSimpleChatHistoryMarkdown(nil, "Current message", now)

	assert.Contains(t, result, "<current>")
	assert.Contains(t, result, "Current message")
	assert.NotContains(t, result, "<history>")
}

func TestBuildSimpleChatHistoryMarkdown_WithHistory(t *testing.T) {
	now := time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC)
	history := []models.ChatHistoryEntry{
		{Role: "user", Content: "Hello", Timestamp: now.Add(-2 * time.Minute)},
		{Role: "assistant", Content: "Hi there!", Timestamp: now.Add(-time.Minute)},
	}
	result := BuildSimpleChatHistoryMarkdown(history, "How are you?", now)

	assert.Contains(t, result, "<history>")
	assert.Contains(t, result, "</history>")
	assert.Contains(t, result, "Hello")
	assert.Contains(t, result, "Hi there!")
	assert.Contains(t, result, "<current>")
	assert.Contains(t, result, "How are you?")
}
