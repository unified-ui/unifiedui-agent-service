package n8n_test

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/unifiedui/agent-service/internal/domain/models"
	"github.com/unifiedui/agent-service/internal/services/agents/n8n"
)

func n8nStreamHandler(events []map[string]interface{}) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		for _, evt := range events {
			line, _ := json.Marshal(evt)
			w.Write(line)
			w.Write([]byte("\n"))
		}
	}
}

// --- NewChatWorkflowClient ---

func TestNewChatWorkflowClient_NilConfig(t *testing.T) {
	_, err := n8n.NewChatWorkflowClient(nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "config is required")
}

func TestNewChatWorkflowClient_EmptyChatURL(t *testing.T) {
	_, err := n8n.NewChatWorkflowClient(&n8n.ChatWorkflowConfig{})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "chat URL")
}

func TestNewChatWorkflowClient_Success(t *testing.T) {
	client, err := n8n.NewChatWorkflowClient(&n8n.ChatWorkflowConfig{
		ChatURL: "http://n8n.local/webhook/chat",
	})
	require.NoError(t, err)
	assert.NotNil(t, client)
}

func TestNewChatWorkflowClient_WithCustomHTTPClient(t *testing.T) {
	client, err := n8n.NewChatWorkflowClient(&n8n.ChatWorkflowConfig{
		ChatURL:    "http://n8n.local/webhook/chat",
		HTTPClient: &http.Client{},
	})
	require.NoError(t, err)
	assert.NotNil(t, client)
}

// --- InvokeStreamReader / streamReader ---

func TestInvokeStreamReader_Success(t *testing.T) {
	events := []map[string]interface{}{
		{"type": "item", "content": "Hello "},
		{"type": "item", "content": "World"},
	}
	ts := httptest.NewServer(n8nStreamHandler(events))
	defer ts.Close()

	client, err := n8n.NewChatWorkflowClient(&n8n.ChatWorkflowConfig{
		ChatURL:    ts.URL,
		HTTPClient: ts.Client(),
	})
	require.NoError(t, err)

	req := &n8n.InvokeRequest{
		Message:   "Hi",
		SessionID: "sess1",
	}
	reader, err := client.InvokeStreamReader(context.Background(), req)
	require.NoError(t, err)
	defer reader.Close()

	var content string
	for {
		chunk, err := reader.Read()
		if err == io.EOF {
			break
		}
		require.NoError(t, err)
		if chunk.Type == n8n.ChunkTypeContent {
			content += chunk.Content
		}
	}
	assert.Equal(t, "Hello World", content)
}

func TestInvokeStreamReader_MetadataChunk(t *testing.T) {
	events := []map[string]interface{}{
		{"type": "item", "content": `{"executionId":"exec-123"}`},
		{"type": "item", "content": "Text content"},
	}
	ts := httptest.NewServer(n8nStreamHandler(events))
	defer ts.Close()

	client, err := n8n.NewChatWorkflowClient(&n8n.ChatWorkflowConfig{
		ChatURL:    ts.URL,
		HTTPClient: ts.Client(),
	})
	require.NoError(t, err)

	reader, err := client.InvokeStreamReader(context.Background(), &n8n.InvokeRequest{Message: "Hi"})
	require.NoError(t, err)
	defer reader.Close()

	chunk, err := reader.Read()
	require.NoError(t, err)
	assert.Equal(t, n8n.ChunkTypeMetadata, chunk.Type)
	assert.Equal(t, "exec-123", chunk.ExecutionID)

	chunk, err = reader.Read()
	require.NoError(t, err)
	assert.Equal(t, n8n.ChunkTypeContent, chunk.Type)
	assert.Equal(t, "Text content", chunk.Content)
}

func TestInvokeStreamReader_SkipNonItemEvents(t *testing.T) {
	events := []map[string]interface{}{
		{"type": "begin"},
		{"type": "item", "content": "Only content"},
		{"type": "end"},
	}
	ts := httptest.NewServer(n8nStreamHandler(events))
	defer ts.Close()

	client, err := n8n.NewChatWorkflowClient(&n8n.ChatWorkflowConfig{
		ChatURL:    ts.URL,
		HTTPClient: ts.Client(),
	})
	require.NoError(t, err)

	reader, err := client.InvokeStreamReader(context.Background(), &n8n.InvokeRequest{Message: "Hi"})
	require.NoError(t, err)
	defer reader.Close()

	chunk, err := reader.Read()
	require.NoError(t, err)
	assert.Equal(t, "Only content", chunk.Content)

	_, err = reader.Read()
	assert.Equal(t, io.EOF, err)
}

func TestInvokeStreamReader_SkipEmptyContent(t *testing.T) {
	events := []map[string]interface{}{
		{"type": "item", "content": ""},
		{"type": "item", "content": "actual content"},
	}
	ts := httptest.NewServer(n8nStreamHandler(events))
	defer ts.Close()

	client, err := n8n.NewChatWorkflowClient(&n8n.ChatWorkflowConfig{
		ChatURL:    ts.URL,
		HTTPClient: ts.Client(),
	})
	require.NoError(t, err)

	reader, err := client.InvokeStreamReader(context.Background(), &n8n.InvokeRequest{Message: "Hi"})
	require.NoError(t, err)
	defer reader.Close()

	chunk, err := reader.Read()
	require.NoError(t, err)
	assert.Equal(t, "actual content", chunk.Content)
}

func TestInvokeStreamReader_SkipNonJSONLines(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("not json\n"))
		evt, _ := json.Marshal(map[string]interface{}{"type": "item", "content": "valid"})
		w.Write(evt)
		w.Write([]byte("\n"))
	}))
	defer ts.Close()

	client, err := n8n.NewChatWorkflowClient(&n8n.ChatWorkflowConfig{
		ChatURL:    ts.URL,
		HTTPClient: ts.Client(),
	})
	require.NoError(t, err)

	reader, err := client.InvokeStreamReader(context.Background(), &n8n.InvokeRequest{Message: "Hi"})
	require.NoError(t, err)
	defer reader.Close()

	chunk, err := reader.Read()
	require.NoError(t, err)
	assert.Equal(t, "valid", chunk.Content)
}

func TestInvokeStreamReader_ServerError(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(500)
		w.Write([]byte("internal error"))
	}))
	defer ts.Close()

	client, err := n8n.NewChatWorkflowClient(&n8n.ChatWorkflowConfig{
		ChatURL:    ts.URL,
		HTTPClient: ts.Client(),
	})
	require.NoError(t, err)

	_, err = client.InvokeStreamReader(context.Background(), &n8n.InvokeRequest{Message: "Hi"})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "500")
}

// --- Invoke (non-streaming, wraps InvokeStreamReader) ---

func TestInvoke_Success(t *testing.T) {
	events := []map[string]interface{}{
		{"type": "item", "content": "Hello "},
		{"type": "item", "content": "World"},
	}
	ts := httptest.NewServer(n8nStreamHandler(events))
	defer ts.Close()

	client, err := n8n.NewChatWorkflowClient(&n8n.ChatWorkflowConfig{
		ChatURL:    ts.URL,
		HTTPClient: ts.Client(),
	})
	require.NoError(t, err)

	resp, err := client.Invoke(context.Background(), &n8n.InvokeRequest{
		Message:   "Hi",
		SessionID: "sess1",
	})
	require.NoError(t, err)
	assert.Equal(t, "Hello World", resp.Content)
	assert.Equal(t, "sess1", resp.SessionID)
}

func TestInvoke_WithExecutionID(t *testing.T) {
	events := []map[string]interface{}{
		{"type": "item", "content": "response text"},
		{"type": "item", "content": `{"executionId":"exec-456"}`},
	}
	ts := httptest.NewServer(n8nStreamHandler(events))
	defer ts.Close()

	client, err := n8n.NewChatWorkflowClient(&n8n.ChatWorkflowConfig{
		ChatURL:    ts.URL,
		HTTPClient: ts.Client(),
	})
	require.NoError(t, err)

	resp, err := client.Invoke(context.Background(), &n8n.InvokeRequest{Message: "Hi"})
	require.NoError(t, err)
	assert.Equal(t, "response text", resp.Content)
	assert.Equal(t, "exec-456", resp.ExecutionID)
}

// --- InvokeStream ---

func TestInvokeStream_Success(t *testing.T) {
	events := []map[string]interface{}{
		{"type": "item", "content": "chunk1"},
		{"type": "item", "content": "chunk2"},
	}
	ts := httptest.NewServer(n8nStreamHandler(events))
	defer ts.Close()

	client, err := n8n.NewChatWorkflowClient(&n8n.ChatWorkflowConfig{
		ChatURL:    ts.URL,
		HTTPClient: ts.Client(),
	})
	require.NoError(t, err)

	ch, err := client.InvokeStream(context.Background(), &n8n.InvokeRequest{Message: "Hi"})
	require.NoError(t, err)

	var chunks []string
	for chunk := range ch {
		if chunk.Type == n8n.ChunkTypeContent {
			chunks = append(chunks, chunk.Content)
		}
	}
	assert.Equal(t, []string{"chunk1", "chunk2"}, chunks)
}

// --- ChatHistory integration ---

func TestInvokeStreamReader_WithChatHistory(t *testing.T) {
	var receivedBody map[string]interface{}
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewDecoder(r.Body).Decode(&receivedBody)
		evt, _ := json.Marshal(map[string]interface{}{"type": "item", "content": "OK"})
		w.Write(evt)
		w.Write([]byte("\n"))
	}))
	defer ts.Close()

	client, err := n8n.NewChatWorkflowClient(&n8n.ChatWorkflowConfig{
		ChatURL:               ts.URL,
		HTTPClient:            ts.Client(),
		UseUnifiedChatHistory: true,
	})
	require.NoError(t, err)

	reader, err := client.InvokeStreamReader(context.Background(), &n8n.InvokeRequest{
		Message:   "New message",
		SessionID: "sess1",
		ChatHistory: []models.ChatHistoryEntry{
			{Role: "user", Content: "Previous"},
			{Role: "assistant", Content: "Previous response"},
		},
	})
	require.NoError(t, err)
	defer reader.Close()

	chatInput, ok := receivedBody["chatInput"].(string)
	require.True(t, ok)
	assert.Contains(t, chatInput, "Previous")
	assert.Contains(t, chatInput, "New message")
}

// --- Close ---

func TestClose(t *testing.T) {
	client, _ := n8n.NewChatWorkflowClient(&n8n.ChatWorkflowConfig{
		ChatURL: "http://test",
	})
	err := client.Close()
	assert.NoError(t, err)
}

// --- Basic auth headers ---

func TestInvokeStreamReader_WithBasicAuth(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		user, pass, ok := r.BasicAuth()
		assert.True(t, ok)
		assert.Equal(t, "admin", user)
		assert.Equal(t, "secret", pass)
		evt, _ := json.Marshal(map[string]interface{}{"type": "item", "content": "OK"})
		w.Write(evt)
		w.Write([]byte("\n"))
	}))
	defer ts.Close()

	client, err := n8n.NewChatWorkflowClient(&n8n.ChatWorkflowConfig{
		ChatURL:    ts.URL,
		HTTPClient: ts.Client(),
		Username:   "admin",
		Password:   "secret",
	})
	require.NoError(t, err)

	reader, err := client.InvokeStreamReader(context.Background(), &n8n.InvokeRequest{Message: "Hi"})
	require.NoError(t, err)
	reader.Close()
}

// --- Pre-converted input ---

func TestInvokeStreamReader_WithPreConvertedInput(t *testing.T) {
	var receivedBody map[string]interface{}
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewDecoder(r.Body).Decode(&receivedBody)
		evt, _ := json.Marshal(map[string]interface{}{"type": "item", "content": "OK"})
		w.Write(evt)
		w.Write([]byte("\n"))
	}))
	defer ts.Close()

	client, err := n8n.NewChatWorkflowClient(&n8n.ChatWorkflowConfig{
		ChatURL:    ts.URL,
		HTTPClient: ts.Client(),
	})
	require.NoError(t, err)

	req := &n8n.InvokeRequest{
		Message:   "Hi with files",
		SessionID: "sess1",
		Input: &n8n.ChatRequestWithFiles{
			ChatInput: "placeholder",
			SessionID: "placeholder",
			Files: []n8n.FileAttachment{
				{Type: "image", Data: "base64data", Filename: "img.png"},
			},
		},
	}
	reader, err := client.InvokeStreamReader(context.Background(), req)
	require.NoError(t, err)
	defer reader.Close()

	assert.Equal(t, "Hi with files", receivedBody["chatInput"])
	assert.Equal(t, "sess1", receivedBody["sessionId"])
}

func TestInvokeStreamReader_WithChatRequestInput(t *testing.T) {
	var receivedBody map[string]interface{}
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewDecoder(r.Body).Decode(&receivedBody)
		evt, _ := json.Marshal(map[string]interface{}{"type": "item", "content": "OK"})
		w.Write(evt)
		w.Write([]byte("\n"))
	}))
	defer ts.Close()

	client, err := n8n.NewChatWorkflowClient(&n8n.ChatWorkflowConfig{
		ChatURL:    ts.URL,
		HTTPClient: ts.Client(),
	})
	require.NoError(t, err)

	req := &n8n.InvokeRequest{
		Message:   "Simple message",
		SessionID: "sess2",
		Input:     &n8n.ChatRequest{ChatInput: "placeholder", SessionID: "placeholder"},
	}
	reader, err := client.InvokeStreamReader(context.Background(), req)
	require.NoError(t, err)
	defer reader.Close()

	assert.Equal(t, "Simple message", receivedBody["chatInput"])
	assert.Equal(t, "sess2", receivedBody["sessionId"])
}

// --- SkipOtherJSONMetadata ---

func TestInvokeStreamReader_SkipOtherJSONMetadata(t *testing.T) {
	events := []map[string]interface{}{
		{"type": "item", "content": `{"someKey":"someValue"}`},
		{"type": "item", "content": "actual text"},
	}
	ts := httptest.NewServer(n8nStreamHandler(events))
	defer ts.Close()

	client, err := n8n.NewChatWorkflowClient(&n8n.ChatWorkflowConfig{
		ChatURL:    ts.URL,
		HTTPClient: ts.Client(),
	})
	require.NoError(t, err)

	reader, err := client.InvokeStreamReader(context.Background(), &n8n.InvokeRequest{Message: "Hi"})
	require.NoError(t, err)
	defer reader.Close()

	chunk, err := reader.Read()
	require.NoError(t, err)
	assert.Equal(t, n8n.ChunkTypeContent, chunk.Type)
	assert.Equal(t, "actual text", chunk.Content)
}

// --- CancelledContext ---

func TestInvokeStreamReader_CancelledContext(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		evt, _ := json.Marshal(map[string]interface{}{"type": "item", "content": "OK"})
		w.Write(evt)
		w.Write([]byte("\n"))
	}))
	defer ts.Close()

	client, err := n8n.NewChatWorkflowClient(&n8n.ChatWorkflowConfig{
		ChatURL:    ts.URL,
		HTTPClient: ts.Client(),
	})
	require.NoError(t, err)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err = client.InvokeStreamReader(ctx, &n8n.InvokeRequest{Message: "Hi"})
	assert.Error(t, err)
}
