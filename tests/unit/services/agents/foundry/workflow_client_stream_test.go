package foundry_test

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/unifiedui/agent-service/internal/services/agents/foundry"
)

func makeSSEEvent(eventType string, data interface{}) string {
	jsonData, _ := json.Marshal(data)
	return fmt.Sprintf("event: %s\ndata: %s\n\n", eventType, string(jsonData))
}

func TestWorkflowClient_InvokeStream_Success(t *testing.T) {
	events := []string{
		makeSSEEvent("response.output_text.delta", map[string]interface{}{
			"type":  "response.output_text.delta",
			"delta": "Hello ",
		}),
		makeSSEEvent("response.output_text.delta", map[string]interface{}{
			"type":  "response.output_text.delta",
			"delta": "World",
		}),
		makeSSEEvent("response.completed", map[string]interface{}{
			"type": "response.completed",
			"response": map[string]interface{}{
				"id":     "resp-1",
				"status": "completed",
				"usage":  map[string]interface{}{"input_tokens": 10, "output_tokens": 20, "total_tokens": 30},
			},
		}),
		"data: [DONE]\n\n",
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "Bearer test-token", r.Header.Get("Authorization"))
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)
		for _, e := range events {
			fmt.Fprint(w, e)
		}
	}))
	defer server.Close()

	client, err := foundry.NewWorkflowClient(&foundry.WorkflowClientConfig{
		ProjectEndpoint: server.URL,
		AgentName:       "test-agent",
		APIToken:        "test-token",
	})
	require.NoError(t, err)

	ch, err := client.InvokeStream(context.Background(), &foundry.InvokeRequest{
		Message:           "hello",
		ExtConversationID: "conv-1",
	})
	require.NoError(t, err)

	var chunks []*foundry.StreamChunk //nolint:prealloc // length unknown: reading from channel
	for chunk := range ch {
		chunks = append(chunks, chunk)
	}

	contentChunks := 0
	for _, c := range chunks {
		if c.Type == foundry.ChunkTypeContent {
			contentChunks++
		}
	}
	assert.GreaterOrEqual(t, contentChunks, 1)
}

func TestWorkflowClient_InvokeStreamReader_ServerError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("internal error"))
	}))
	defer server.Close()

	client, _ := foundry.NewWorkflowClient(&foundry.WorkflowClientConfig{
		ProjectEndpoint: server.URL,
		AgentName:       "test-agent",
		APIToken:        "test-token",
	})

	_, err := client.InvokeStreamReader(context.Background(), &foundry.InvokeRequest{
		Message: "hello",
	})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "foundry API error")
}

func TestWorkflowClient_InvokeStreamReader_ParsesEvents(t *testing.T) {
	sseBody := makeSSEEvent("response.output_text.delta", map[string]interface{}{
		"type":  "response.output_text.delta",
		"delta": "Test content",
	}) + "data: [DONE]\n\n"

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, sseBody)
	}))
	defer server.Close()

	client, _ := foundry.NewWorkflowClient(&foundry.WorkflowClientConfig{
		ProjectEndpoint: server.URL,
		AgentName:       "test-agent",
		APIToken:        "test-token",
	})

	reader, err := client.InvokeStreamReader(context.Background(), &foundry.InvokeRequest{
		Message: "test",
	})
	require.NoError(t, err)
	defer reader.Close()

	chunk, err := reader.Read()
	require.NoError(t, err)
	assert.Equal(t, foundry.ChunkTypeContent, chunk.Type)
	assert.Equal(t, "Test content", chunk.Content)

	// Next should be EOF after [DONE]
	_, err = reader.Read()
	assert.ErrorIs(t, err, io.EOF)
}

func TestWorkflowClient_InvokeStreamReader_WorkflowAction(t *testing.T) {
	item := map[string]interface{}{
		"type":               "workflow_action",
		"id":                 "wa-1",
		"kind":               "InvokeAgent",
		"action_id":          "act-1",
		"parent_action_id":   "parent-1",
		"previous_action_id": "prev-1",
		"status":             "completed",
	}

	sseBody := makeSSEEvent("response.output_item.added", map[string]interface{}{
		"type": "response.output_item.added",
		"item": item,
	}) + "data: [DONE]\n\n"

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, sseBody)
	}))
	defer server.Close()

	client, _ := foundry.NewWorkflowClient(&foundry.WorkflowClientConfig{
		ProjectEndpoint: server.URL,
		AgentName:       "test-agent",
		APIToken:        "test-token",
	})

	reader, err := client.InvokeStreamReader(context.Background(), &foundry.InvokeRequest{Message: "test"})
	require.NoError(t, err)
	defer reader.Close()

	chunk, err := reader.Read()
	require.NoError(t, err)
	assert.Equal(t, foundry.ChunkTypeToolCallStart, chunk.Type)
	assert.Equal(t, "InvokeAgent", chunk.Config["tool_name"])
	assert.Equal(t, "workflow_action", chunk.Config["call_type"])
}

func TestWorkflowClient_InvokeStreamReader_MessageDone(t *testing.T) {
	item := map[string]interface{}{
		"type":   "message",
		"id":     "msg-1",
		"role":   "assistant",
		"status": "completed",
		"content": []map[string]interface{}{
			{"type": "output_text", "text": "Hello there"},
		},
		"created_by": map[string]interface{}{
			"agent":       map[string]interface{}{"name": "my-agent"},
			"response_id": "resp-1",
		},
	}

	sseBody := makeSSEEvent("response.output_item.done", map[string]interface{}{
		"type":         "response.output_item.done",
		"item":         item,
		"output_index": 0,
	}) + "data: [DONE]\n\n"

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, sseBody)
	}))
	defer server.Close()

	client, _ := foundry.NewWorkflowClient(&foundry.WorkflowClientConfig{
		ProjectEndpoint: server.URL,
		AgentName:       "test-agent",
		APIToken:        "test-token",
	})

	reader, err := client.InvokeStreamReader(context.Background(), &foundry.InvokeRequest{
		Message: "test",
	})
	require.NoError(t, err)
	defer reader.Close()

	chunk, err := reader.Read()
	require.NoError(t, err)
	assert.Equal(t, foundry.ChunkTypeMetadata, chunk.Type)
	assert.Equal(t, "message_done", chunk.Metadata["type"])
}

func TestWorkflowClient_Invoke_FullResponse(t *testing.T) {
	sseBody := makeSSEEvent("response.output_text.delta", map[string]interface{}{
		"type":  "response.output_text.delta",
		"delta": "Full ",
	}) + makeSSEEvent("response.output_text.delta", map[string]interface{}{
		"type":  "response.output_text.delta",
		"delta": "response",
	}) + makeSSEEvent("response.completed", map[string]interface{}{
		"type": "response.completed",
		"response": map[string]interface{}{
			"id":     "resp-1",
			"status": "completed",
		},
	}) + "data: [DONE]\n\n"

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, sseBody)
	}))
	defer server.Close()

	client, _ := foundry.NewWorkflowClient(&foundry.WorkflowClientConfig{
		ProjectEndpoint: server.URL,
		AgentName:       "test-agent",
		APIToken:        "test-token",
	})

	resp, err := client.Invoke(context.Background(), &foundry.InvokeRequest{
		Message:           "test",
		ExtConversationID: "conv-1",
	})
	require.NoError(t, err)
	assert.Equal(t, "Full response", resp.Content)
	assert.Equal(t, "conv-1", resp.SessionID)
}

func TestWorkflowClient_CloseReleasesResources(t *testing.T) {
	client, _ := foundry.NewWorkflowClient(&foundry.WorkflowClientConfig{
		ProjectEndpoint: "https://example.com",
		AgentName:       "agent",
		APIToken:        "token",
	})
	err := client.Close()
	assert.NoError(t, err)
}

func TestWorkflowClient_InvokeStreamReader_WithMultimodalInput(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify the request body has Input field
		var payload map[string]interface{}
		json.NewDecoder(r.Body).Decode(&payload)
		assert.NotNil(t, payload["input"])

		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, "data: [DONE]\n\n")
	}))
	defer server.Close()

	client, _ := foundry.NewWorkflowClient(&foundry.WorkflowClientConfig{
		ProjectEndpoint: server.URL,
		AgentName:       "test-agent",
		APIToken:        "test-token",
	})

	reader, err := client.InvokeStreamReader(context.Background(), &foundry.InvokeRequest{
		Input: []map[string]interface{}{
			{"type": "input_text", "text": "hello"},
		},
	})
	require.NoError(t, err)
	reader.Close()
}
