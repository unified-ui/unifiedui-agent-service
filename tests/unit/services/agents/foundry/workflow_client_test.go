// Package foundry_test provides tests for the Microsoft Foundry agent client.
package foundry_test

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/unifiedui/agent-service/internal/services/agents/foundry"
)

// TestNewWorkflowClient_Success tests successful client creation.
func TestNewWorkflowClient_Success(t *testing.T) {
	config := &foundry.WorkflowClientConfig{
		ProjectEndpoint: "https://test.services.ai.azure.com/api/projects/test-project",
		APIVersion:      "2025-11-15-preview",
		AgentName:       "TestAgent",
		AgentType:       "AGENT",
		APIToken:        "test-token",
	}

	client, err := foundry.NewWorkflowClient(config)
	require.NoError(t, err)
	require.NotNil(t, client)
}

// TestNewWorkflowClient_NilConfig tests error when config is nil.
func TestNewWorkflowClient_NilConfig(t *testing.T) {
	client, err := foundry.NewWorkflowClient(nil)
	require.Error(t, err)
	assert.Nil(t, client)
	assert.Contains(t, err.Error(), "config is required")
}

// TestNewWorkflowClient_MissingProjectEndpoint tests error when project endpoint is missing.
func TestNewWorkflowClient_MissingProjectEndpoint(t *testing.T) {
	config := &foundry.WorkflowClientConfig{
		AgentName: "TestAgent",
		APIToken:  "test-token",
	}

	client, err := foundry.NewWorkflowClient(config)
	require.Error(t, err)
	assert.Nil(t, client)
	assert.Contains(t, err.Error(), "project endpoint is required")
}

// TestNewWorkflowClient_MissingAgentName tests error when agent name is missing.
func TestNewWorkflowClient_MissingAgentName(t *testing.T) {
	config := &foundry.WorkflowClientConfig{
		ProjectEndpoint: "https://test.services.ai.azure.com/api/projects/test-project",
		APIToken:        "test-token",
	}

	client, err := foundry.NewWorkflowClient(config)
	require.Error(t, err)
	assert.Nil(t, client)
	assert.Contains(t, err.Error(), "agent name is required")
}

// TestNewWorkflowClient_MissingAPIToken tests error when API token is missing.
func TestNewWorkflowClient_MissingAPIToken(t *testing.T) {
	config := &foundry.WorkflowClientConfig{
		ProjectEndpoint: "https://test.services.ai.azure.com/api/projects/test-project",
		AgentName:       "TestAgent",
	}

	client, err := foundry.NewWorkflowClient(config)
	require.Error(t, err)
	assert.Nil(t, client)
	assert.Contains(t, err.Error(), "API token is required")
}

// TestNewWorkflowClient_DefaultAPIVersion tests default API version is set.
func TestNewWorkflowClient_DefaultAPIVersion(t *testing.T) {
	config := &foundry.WorkflowClientConfig{
		ProjectEndpoint: "https://test.services.ai.azure.com/api/projects/test-project",
		AgentName:       "TestAgent",
		APIToken:        "test-token",
		// APIVersion not set - should default
	}

	client, err := foundry.NewWorkflowClient(config)
	require.NoError(t, err)
	require.NotNil(t, client)
}

// TestInvokeStreamReader_Success tests successful streaming invocation.
func TestInvokeStreamReader_Success(t *testing.T) {
	// Create SSE response
	sseResponse := strings.Join([]string{
		`event: response.created`,
		`{"type":"response.created","sequence_number":1,"response":{"id":"resp_123","status":"in_progress"}}`,
		``,
		`event: response.output_text.delta`,
		`{"type":"response.output_text.delta","sequence_number":2,"delta":"Hello"}`,
		``,
		`event: response.output_text.delta`,
		`{"type":"response.output_text.delta","sequence_number":3,"delta":" World"}`,
		``,
		`event: response.completed`,
		`{"type":"response.completed","sequence_number":4,"response":{"id":"resp_123","status":"completed","conversation":{"id":"conv_456"},"agent":{"name":"TestAgent"}}}`,
		``,
	}, "\n")

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify request
		assert.Equal(t, http.MethodPost, r.Method)
		assert.Contains(t, r.URL.Path, "/openai/responses")
		assert.Equal(t, "Bearer test-token", r.Header.Get("Authorization"))
		assert.Equal(t, "text/event-stream", r.Header.Get("Accept"))

		// Verify request body
		body, err := io.ReadAll(r.Body)
		require.NoError(t, err)

		var payload foundry.FoundryRequestPayload
		err = json.Unmarshal(body, &payload)
		require.NoError(t, err)
		assert.Equal(t, "agent_reference", payload.Agent.Type)
		assert.Equal(t, "TestAgent", payload.Agent.Name)
		assert.Equal(t, "conv_123", payload.Conversation)
		assert.Equal(t, "Hello", payload.Input)
		assert.True(t, payload.Stream)

		// Send SSE response
		w.Header().Set("Content-Type", "text/event-stream")
		w.Write([]byte(sseResponse))
	}))
	defer server.Close()

	config := &foundry.WorkflowClientConfig{
		ProjectEndpoint: server.URL,
		APIVersion:      "2025-11-15-preview",
		AgentName:       "TestAgent",
		AgentType:       "AGENT",
		APIToken:        "test-token",
	}

	client, err := foundry.NewWorkflowClient(config)
	require.NoError(t, err)

	req := &foundry.InvokeRequest{
		ExtConversationID: "conv_123",
		Message:           "Hello",
	}

	reader, err := client.InvokeStreamReader(context.Background(), req)
	require.NoError(t, err)
	defer reader.Close()

	var chunks []*foundry.StreamChunk
	for {
		chunk, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		chunks = append(chunks, chunk)
	}

	// Verify chunks
	require.GreaterOrEqual(t, len(chunks), 2)

	// Check for content chunks
	var fullContent string
	for _, chunk := range chunks {
		if chunk.Type == foundry.ChunkTypeContent {
			fullContent += chunk.Content
		}
	}
	assert.Equal(t, "Hello World", fullContent)
}

// TestInvokeStreamReader_HTTPError tests handling of HTTP errors.
func TestInvokeStreamReader_HTTPError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte(`{"error": "unauthorized"}`))
	}))
	defer server.Close()

	config := &foundry.WorkflowClientConfig{
		ProjectEndpoint: server.URL,
		AgentName:       "TestAgent",
		APIToken:        "test-token",
	}

	client, err := foundry.NewWorkflowClient(config)
	require.NoError(t, err)

	req := &foundry.InvokeRequest{
		ExtConversationID: "conv_123",
		Message:           "Hello",
	}

	reader, err := client.InvokeStreamReader(context.Background(), req)
	require.Error(t, err)
	assert.Nil(t, reader)
	assert.Contains(t, err.Error(), "401")
}

// TestInvoke_Success tests the non-streaming invoke method.
func TestInvoke_Success(t *testing.T) {
	sseResponse := strings.Join([]string{
		`event: response.output_text.delta`,
		`{"type":"response.output_text.delta","delta":"Test response"}`,
		``,
		`event: response.completed`,
		`{"type":"response.completed","response":{"id":"resp_123","status":"completed"}}`,
		``,
	}, "\n")

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		w.Write([]byte(sseResponse))
	}))
	defer server.Close()

	config := &foundry.WorkflowClientConfig{
		ProjectEndpoint: server.URL,
		AgentName:       "TestAgent",
		APIToken:        "test-token",
	}

	client, err := foundry.NewWorkflowClient(config)
	require.NoError(t, err)

	req := &foundry.InvokeRequest{
		ExtConversationID: "conv_123",
		Message:           "Hello",
	}

	resp, err := client.Invoke(context.Background(), req)
	require.NoError(t, err)
	require.NotNil(t, resp)
	assert.Equal(t, "Test response", resp.Content)
}

// TestWorkflowClient_Close tests the Close method.
func TestWorkflowClient_Close(t *testing.T) {
	config := &foundry.WorkflowClientConfig{
		ProjectEndpoint: "https://test.services.ai.azure.com/api/projects/test-project",
		AgentName:       "TestAgent",
		APIToken:        "test-token",
	}

	client, err := foundry.NewWorkflowClient(config)
	require.NoError(t, err)

	err = client.Close()
	assert.NoError(t, err)
}

// TestStreamReader_ParseWorkflowAction tests parsing workflow action events.
func TestStreamReader_ParseWorkflowAction(t *testing.T) {
	sseResponse := strings.Join([]string{
		`event: response.output_item.added`,
		`{"type":"response.output_item.added","output_index":0,"item":{"type":"workflow_action","id":"wfa_123","kind":"Question","action_id":"action-1","parent_action_id":"trigger_wf","status":"in_progress"}}`,
		``,
		`event: response.output_text.delta`,
		`{"type":"response.output_text.delta","delta":"What is your name?"}`,
		``,
		`event: response.completed`,
		`{"type":"response.completed","response":{"id":"resp_123","status":"completed"}}`,
		``,
	}, "\n")

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		w.Write([]byte(sseResponse))
	}))
	defer server.Close()

	config := &foundry.WorkflowClientConfig{
		ProjectEndpoint: server.URL,
		AgentName:       "TestAgent",
		APIToken:        "test-token",
	}

	client, err := foundry.NewWorkflowClient(config)
	require.NoError(t, err)

	req := &foundry.InvokeRequest{
		ExtConversationID: "conv_123",
		Message:           "Hi",
	}

	reader, err := client.InvokeStreamReader(context.Background(), req)
	require.NoError(t, err)
	defer reader.Close()

	var hasWorkflowAction bool
	var hasContent bool
	for {
		chunk, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if chunk.Type == foundry.ChunkTypeMetadata {
			if chunk.Metadata != nil {
				if chunk.Metadata["type"] == "workflow_action" {
					hasWorkflowAction = true
					assert.Equal(t, "Question", chunk.Metadata["kind"])
					assert.Equal(t, "action-1", chunk.Metadata["action_id"])
				}
			}
		}
		if chunk.Type == foundry.ChunkTypeContent {
			hasContent = true
		}
	}

	assert.True(t, hasWorkflowAction, "Expected workflow action chunk")
	assert.True(t, hasContent, "Expected content chunk")
}

// TestStreamReader_MultipleMessages tests parsing multiple messages in one response.
func TestStreamReader_MultipleMessages(t *testing.T) {
	sseResponse := strings.Join([]string{
		`event: response.output_item.added`,
		`{"type":"response.output_item.added","output_index":0,"item":{"type":"message","id":"msg_1","role":"assistant","status":"in_progress"}}`,
		``,
		`event: response.output_text.delta`,
		`{"type":"response.output_text.delta","item_id":"msg_1","delta":"First message"}`,
		``,
		`event: response.output_item.done`,
		`{"type":"response.output_item.done","output_index":0,"item":{"type":"message","id":"msg_1","role":"assistant","status":"completed","content":[{"type":"output_text","text":"First message"}]}}`,
		``,
		`event: response.output_item.added`,
		`{"type":"response.output_item.added","output_index":1,"item":{"type":"message","id":"msg_2","role":"assistant","status":"in_progress"}}`,
		``,
		`event: response.output_text.delta`,
		`{"type":"response.output_text.delta","item_id":"msg_2","delta":"Second message"}`,
		``,
		`event: response.output_item.done`,
		`{"type":"response.output_item.done","output_index":1,"item":{"type":"message","id":"msg_2","role":"assistant","status":"completed","content":[{"type":"output_text","text":"Second message"}]}}`,
		``,
		`event: response.completed`,
		`{"type":"response.completed","response":{"id":"resp_123","status":"completed"}}`,
		``,
	}, "\n")

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		w.Write([]byte(sseResponse))
	}))
	defer server.Close()

	config := &foundry.WorkflowClientConfig{
		ProjectEndpoint: server.URL,
		AgentName:       "TestAgent",
		APIToken:        "test-token",
	}

	client, err := foundry.NewWorkflowClient(config)
	require.NoError(t, err)

	req := &foundry.InvokeRequest{
		ExtConversationID: "conv_123",
		Message:           "Hi",
	}

	reader, err := client.InvokeStreamReader(context.Background(), req)
	require.NoError(t, err)
	defer reader.Close()

	var newMessageCount int
	var contentChunks []string
	for {
		chunk, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if chunk.Type == foundry.ChunkTypeNewMessage {
			newMessageCount++
		}
		if chunk.Type == foundry.ChunkTypeContent {
			contentChunks = append(contentChunks, chunk.Content)
		}
	}

	// Should have 1 new message signal (between first and second message)
	assert.Equal(t, 1, newMessageCount)
	assert.Contains(t, contentChunks, "First message")
	assert.Contains(t, contentChunks, "Second message")
}
