package n8n_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/unifiedui/agent-service/internal/services/agents/n8n"
)

func TestNewAPIClient_NilConfig(t *testing.T) {
	_, err := n8n.NewAPIClient(nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "config is required")
}

func TestNewAPIClient_Success(t *testing.T) {
	client, err := n8n.NewAPIClient(&n8n.APIClientConfig{
		BaseURL: "http://localhost:5678",
		APIKey:  "test-key",
	})
	require.NoError(t, err)
	assert.NotNil(t, client)
}

func TestNewAPIClient_DefaultHTTPClient(t *testing.T) {
	client, err := n8n.NewAPIClient(&n8n.APIClientConfig{
		BaseURL: "http://localhost:5678",
	})
	require.NoError(t, err)
	assert.NotNil(t, client)
}

func TestAPIClient_GetExecution_Success(t *testing.T) {
	execResp := map[string]interface{}{
		"id":        "exec-123",
		"status":    "success",
		"startedAt": "2024-01-01T00:00:00Z",
		"stoppedAt": "2024-01-01T00:01:00Z",
		"data":      map[string]interface{}{},
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/api/v1/executions/exec-123", r.URL.Path)
		assert.Equal(t, "application/json", r.Header.Get("Content-Type"))
		assert.Equal(t, "test-key", r.Header.Get("X-N8N-API-KEY"))

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(execResp)
	}))
	defer server.Close()

	client, _ := n8n.NewAPIClient(&n8n.APIClientConfig{
		BaseURL:    server.URL,
		APIKey:     "test-key",
		HTTPClient: server.Client(),
	})

	info, err := client.GetExecution(context.Background(), "exec-123")
	require.NoError(t, err)
	assert.Equal(t, "exec-123", info.ID)
	assert.Equal(t, "success", info.Status)
}

func TestAPIClient_GetExecution_ServerError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	client, _ := n8n.NewAPIClient(&n8n.APIClientConfig{
		BaseURL:    server.URL,
		APIKey:     "key",
		HTTPClient: server.Client(),
	})

	_, err := client.GetExecution(context.Background(), "exec-123")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unexpected status code")
}

func TestAPIClient_GetExecution_InvalidJSON(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("not-json"))
	}))
	defer server.Close()

	client, _ := n8n.NewAPIClient(&n8n.APIClientConfig{
		BaseURL:    server.URL,
		HTTPClient: server.Client(),
	})

	_, err := client.GetExecution(context.Background(), "exec-123")
	assert.Error(t, err)
}

func TestAPIClient_GetExecution_CancelledContext(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{"id": "exec-123"})
	}))
	defer server.Close()

	client, _ := n8n.NewAPIClient(&n8n.APIClientConfig{
		BaseURL:    server.URL,
		HTTPClient: server.Client(),
	})

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := client.GetExecution(ctx, "exec-123")
	assert.Error(t, err)
}

func TestAPIClient_GetExecutionsBySession(t *testing.T) {
	client, _ := n8n.NewAPIClient(&n8n.APIClientConfig{
		BaseURL: "http://localhost:5678",
	})

	result, err := client.GetExecutionsBySession(context.Background(), "session-1")
	require.NoError(t, err)
	assert.Empty(t, result)
}

func TestAPIClient_Close(t *testing.T) {
	client, _ := n8n.NewAPIClient(&n8n.APIClientConfig{
		BaseURL: "http://localhost:5678",
	})
	err := client.Close()
	assert.NoError(t, err)
}

func TestAPIClient_GetExecution_NoAPIKey(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Empty(t, r.Header.Get("X-N8N-API-KEY"))
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"id":        "exec-1",
			"status":    "success",
			"startedAt": "2024-01-01T00:00:00Z",
		})
	}))
	defer server.Close()

	client, _ := n8n.NewAPIClient(&n8n.APIClientConfig{
		BaseURL:    server.URL,
		HTTPClient: server.Client(),
	})

	info, err := client.GetExecution(context.Background(), "exec-1")
	require.NoError(t, err)
	assert.Equal(t, "exec-1", info.ID)
}
