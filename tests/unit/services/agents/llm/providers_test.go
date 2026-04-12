package llm_test

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/unifiedui/agent-service/internal/services/agents/llm"
)

func openAIStreamHandler(chunks []string) http.HandlerFunc {
	return func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)

		for _, chunk := range chunks {
			resp := map[string]interface{}{
				"choices": []map[string]interface{}{
					{"delta": map[string]interface{}{"content": chunk}},
				},
			}
			data, _ := json.Marshal(resp)
			fmt.Fprintf(w, "data: %s\n\n", data)
		}
		fmt.Fprintf(w, "data: [DONE]\n\n")
	}
}

func openAIStreamErrorHandler(status int, body string) http.HandlerFunc {
	return func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(status)
		_, _ = w.Write([]byte(body))
	}
}

func captureRequestHandler(captured *http.Request, chunks []string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		*captured = *r
		captured.Body = r.Body
		bodyBytes, _ := io.ReadAll(r.Body)
		captured.Body = io.NopCloser(strings.NewReader(string(bodyBytes)))

		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)

		for _, chunk := range chunks {
			resp := map[string]interface{}{
				"choices": []map[string]interface{}{
					{"delta": map[string]interface{}{"content": chunk}},
				},
			}
			data, _ := json.Marshal(resp)
			fmt.Fprintf(w, "data: %s\n\n", data)
		}
		fmt.Fprintf(w, "data: [DONE]\n\n")
	}
}

// --- OpenAI Streaming Tests ---

func TestOpenAI_StreamChatCompletion_Success(t *testing.T) {
	ts := httptest.NewServer(openAIStreamHandler([]string{"Hello", " World"}))
	defer ts.Close()

	config := map[string]interface{}{
		"model_name": "gpt-4o-mini",
		"base_url":   ts.URL,
	}
	secret := map[string]interface{}{"api_key": "sk-test"}
	client, err := llm.NewStreamingClient("OPENAI", config, secret)
	require.NoError(t, err)
	defer client.Close()

	messages := []llm.ChatMessage{{Role: "user", Content: "Hi"}}
	reader, err := client.StreamChatCompletion(messages, llm.StreamOptions{})
	require.NoError(t, err)
	defer reader.Close()

	var result strings.Builder
	for {
		chunk, err := reader.Read()
		if errors.Is(err, io.EOF) {
			break
		}
		require.NoError(t, err)
		result.WriteString(chunk)
	}

	assert.Equal(t, "Hello World", result.String())
}

func TestOpenAI_StreamChatCompletion_WithOptions(t *testing.T) {
	var captured http.Request
	ts := httptest.NewServer(captureRequestHandler(&captured, []string{"OK"}))
	defer ts.Close()

	config := map[string]interface{}{
		"model_name": "gpt-4o-mini",
		"base_url":   ts.URL,
	}
	temp := 0.5
	maxTokens := 1024
	client, err := llm.NewStreamingClient("OPENAI", config, nil)
	require.NoError(t, err)
	defer client.Close()

	messages := []llm.ChatMessage{{Role: "user", Content: "Hi"}}
	opts := llm.StreamOptions{Temperature: &temp, MaxTokens: &maxTokens}
	reader, err := client.StreamChatCompletion(messages, opts)
	require.NoError(t, err)
	defer reader.Close()

	bodyBytes, _ := io.ReadAll(captured.Body)
	var body map[string]interface{}
	require.NoError(t, json.Unmarshal(bodyBytes, &body))

	assert.Equal(t, 0.5, body["temperature"])
	assert.Equal(t, float64(1024), body["max_tokens"])
	assert.Equal(t, "gpt-4o-mini", body["model"])
}

func TestOpenAI_StreamChatCompletion_ServerError(t *testing.T) {
	ts := httptest.NewServer(openAIStreamErrorHandler(500, "internal error"))
	defer ts.Close()

	config := map[string]interface{}{
		"model_name": "gpt-4o-mini",
		"base_url":   ts.URL,
	}
	client, err := llm.NewStreamingClient("OPENAI", config, nil)
	require.NoError(t, err)
	defer client.Close()

	_, err = client.StreamChatCompletion(
		[]llm.ChatMessage{{Role: "user", Content: "Hi"}},
		llm.StreamOptions{},
	)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "500")
}

func TestOpenAI_StreamChatCompletion_UserAgent(t *testing.T) {
	var captured http.Request
	ts := httptest.NewServer(captureRequestHandler(&captured, []string{"OK"}))
	defer ts.Close()

	config := map[string]interface{}{
		"model_name": "gpt-4o-mini",
		"base_url":   ts.URL,
	}
	client, err := llm.NewStreamingClient("OPENAI", config, nil)
	require.NoError(t, err)
	defer client.Close()

	reader, err := client.StreamChatCompletion(
		[]llm.ChatMessage{{Role: "user", Content: "Hi"}},
		llm.StreamOptions{},
	)
	require.NoError(t, err)
	defer reader.Close()

	assert.Equal(t, "unified-ui-agent-service/1.0", captured.Header.Get("User-Agent"))
}

// --- Azure OpenAI Streaming Tests ---

func TestAzureOpenAI_StreamChatCompletion_NoTemperature(t *testing.T) {
	var capturedBody []byte
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedBody, _ = io.ReadAll(r.Body)
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)
		resp := map[string]interface{}{
			"choices": []map[string]interface{}{
				{"delta": map[string]interface{}{"content": "OK"}},
			},
		}
		data, _ := json.Marshal(resp)
		fmt.Fprintf(w, "data: %s\n\ndata: [DONE]\n\n", data)
	}))
	defer ts.Close()

	config := map[string]interface{}{
		"endpoint":        ts.URL,
		"deployment_name": "gpt-5-mini",
	}
	temp := 0.7
	client, err := llm.NewStreamingClient("AZURE_OPENAI", config, map[string]interface{}{"api_key": "key"})
	require.NoError(t, err)
	defer client.Close()

	reader, err := client.StreamChatCompletion(
		[]llm.ChatMessage{{Role: "user", Content: "Hi"}},
		llm.StreamOptions{Temperature: &temp},
	)
	require.NoError(t, err)
	defer reader.Close()

	var body map[string]interface{}
	require.NoError(t, json.Unmarshal(capturedBody, &body))

	_, hasTemperature := body["temperature"]
	assert.False(t, hasTemperature, "Azure OpenAI should not send temperature")
}

func TestAzureOpenAI_StreamChatCompletion_UsesMaxCompletionTokens(t *testing.T) {
	var capturedBody []byte
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedBody, _ = io.ReadAll(r.Body)
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)
		resp := map[string]interface{}{
			"choices": []map[string]interface{}{
				{"delta": map[string]interface{}{"content": "OK"}},
			},
		}
		data, _ := json.Marshal(resp)
		fmt.Fprintf(w, "data: %s\n\ndata: [DONE]\n\n", data)
	}))
	defer ts.Close()

	config := map[string]interface{}{
		"endpoint":        ts.URL,
		"deployment_name": "gpt-5-mini",
	}
	maxTokens := 4096
	client, err := llm.NewStreamingClient("AZURE_OPENAI", config, map[string]interface{}{"api_key": "key"})
	require.NoError(t, err)
	defer client.Close()

	reader, err := client.StreamChatCompletion(
		[]llm.ChatMessage{{Role: "user", Content: "Hi"}},
		llm.StreamOptions{MaxTokens: &maxTokens},
	)
	require.NoError(t, err)
	defer reader.Close()

	var body map[string]interface{}
	require.NoError(t, json.Unmarshal(capturedBody, &body))

	_, hasMaxTokens := body["max_tokens"]
	assert.False(t, hasMaxTokens, "Azure OpenAI should not send max_tokens")
	assert.Equal(t, float64(4096), body["max_completion_tokens"])
}

func TestAzureOpenAI_StreamChatCompletion_HasApiVersion(t *testing.T) {
	var capturedPath string
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedPath = r.URL.String()
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)
		resp := map[string]interface{}{
			"choices": []map[string]interface{}{
				{"delta": map[string]interface{}{"content": "OK"}},
			},
		}
		data, _ := json.Marshal(resp)
		fmt.Fprintf(w, "data: %s\n\ndata: [DONE]\n\n", data)
	}))
	defer ts.Close()

	config := map[string]interface{}{
		"endpoint":        ts.URL,
		"deployment_name": "gpt-5-mini",
	}
	client, err := llm.NewStreamingClient("AZURE_OPENAI", config, map[string]interface{}{"api_key": "key"})
	require.NoError(t, err)
	defer client.Close()

	reader, err := client.StreamChatCompletion(
		[]llm.ChatMessage{{Role: "user", Content: "Hi"}},
		llm.StreamOptions{},
	)
	require.NoError(t, err)
	defer reader.Close()

	assert.Contains(t, capturedPath, "api-version=2024-12-01-preview", "Azure OpenAI URL should contain api-version")
	assert.Contains(t, capturedPath, "/openai/deployments/gpt-5-mini/chat/completions")
}

func TestAzureOpenAI_StreamChatCompletion_CustomApiVersion(t *testing.T) {
	var capturedPath string
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedPath = r.URL.String()
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)
		resp := map[string]interface{}{
			"choices": []map[string]interface{}{
				{"delta": map[string]interface{}{"content": "OK"}},
			},
		}
		data, _ := json.Marshal(resp)
		fmt.Fprintf(w, "data: %s\n\ndata: [DONE]\n\n", data)
	}))
	defer ts.Close()

	config := map[string]interface{}{
		"endpoint":        ts.URL,
		"deployment_name": "gpt-5-mini",
		"api_version":     "2025-03-01-preview",
	}
	client, err := llm.NewStreamingClient("AZURE_OPENAI", config, map[string]interface{}{"api_key": "key"})
	require.NoError(t, err)
	defer client.Close()

	reader, err := client.StreamChatCompletion(
		[]llm.ChatMessage{{Role: "user", Content: "Hi"}},
		llm.StreamOptions{},
	)
	require.NoError(t, err)
	defer reader.Close()

	assert.Contains(t, capturedPath, "api-version=2025-03-01-preview", "Azure OpenAI URL should use custom api-version")
}

func TestAzureOpenAI_StreamChatCompletion_AuthHeader(t *testing.T) {
	var captured http.Request
	ts := httptest.NewServer(captureRequestHandler(&captured, []string{"OK"}))
	defer ts.Close()

	config := map[string]interface{}{
		"endpoint":        ts.URL,
		"deployment_name": "gpt-5-mini",
	}
	client, err := llm.NewStreamingClient("AZURE_OPENAI", config, map[string]interface{}{"api_key": "my-secret-key"})
	require.NoError(t, err)
	defer client.Close()

	reader, err := client.StreamChatCompletion(
		[]llm.ChatMessage{{Role: "user", Content: "Hi"}},
		llm.StreamOptions{},
	)
	require.NoError(t, err)
	defer reader.Close()

	assert.Equal(t, "my-secret-key", captured.Header.Get("api-key"))
}

// --- Groq Streaming Tests ---

func TestGroq_StreamChatCompletion_UserAgent(t *testing.T) {
	var captured http.Request
	ts := httptest.NewServer(captureRequestHandler(&captured, []string{"OK"}))
	defer ts.Close()

	config := map[string]interface{}{
		"model_name": "llama-3.3-70b-versatile",
		"base_url":   ts.URL,
	}
	client, err := llm.NewStreamingClient("GROQ", config, map[string]interface{}{"api_key": "gsk-test"})
	require.NoError(t, err)
	defer client.Close()

	reader, err := client.StreamChatCompletion(
		[]llm.ChatMessage{{Role: "user", Content: "Hi"}},
		llm.StreamOptions{},
	)
	require.NoError(t, err)
	defer reader.Close()

	assert.Equal(t, "unified-ui-agent-service/1.0", captured.Header.Get("User-Agent"))
}

func TestGroq_StreamChatCompletion_SendsTemperature(t *testing.T) {
	var capturedBody []byte
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedBody, _ = io.ReadAll(r.Body)
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)
		resp := map[string]interface{}{
			"choices": []map[string]interface{}{
				{"delta": map[string]interface{}{"content": "OK"}},
			},
		}
		data, _ := json.Marshal(resp)
		fmt.Fprintf(w, "data: %s\n\ndata: [DONE]\n\n", data)
	}))
	defer ts.Close()

	config := map[string]interface{}{
		"model_name": "llama-3.3-70b-versatile",
		"base_url":   ts.URL,
	}
	temp := 0.5
	client, err := llm.NewStreamingClient("GROQ", config, nil)
	require.NoError(t, err)
	defer client.Close()

	reader, err := client.StreamChatCompletion(
		[]llm.ChatMessage{{Role: "user", Content: "Hi"}},
		llm.StreamOptions{Temperature: &temp},
	)
	require.NoError(t, err)
	defer reader.Close()

	var body map[string]interface{}
	require.NoError(t, json.Unmarshal(capturedBody, &body))

	assert.Equal(t, 0.5, body["temperature"], "Non-Azure providers should send temperature")
}

// --- Mistral Streaming Tests ---

func TestMistral_StreamChatCompletion_Success(t *testing.T) {
	ts := httptest.NewServer(openAIStreamHandler([]string{"Bonjour"}))
	defer ts.Close()

	config := map[string]interface{}{
		"model_name": "mistral-small-latest",
		"base_url":   ts.URL,
	}
	client, err := llm.NewStreamingClient("MISTRAL", config, map[string]interface{}{"api_key": "test"})
	require.NoError(t, err)
	defer client.Close()

	reader, err := client.StreamChatCompletion(
		[]llm.ChatMessage{{Role: "user", Content: "Hi"}},
		llm.StreamOptions{},
	)
	require.NoError(t, err)
	defer reader.Close()

	chunk, err := reader.Read()
	require.NoError(t, err)
	assert.Equal(t, "Bonjour", chunk)
}

// --- Anthropic System Message Test ---

func TestAnthropic_StreamChatCompletion_SystemMessage(t *testing.T) {
	var capturedBody []byte
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedBody, _ = io.ReadAll(r.Body)
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, "event: content_block_delta\ndata: {\"type\":\"content_block_delta\",\"delta\":{\"type\":\"text_delta\",\"text\":\"Hi\"}}\n\n")
		fmt.Fprint(w, "event: message_stop\ndata: {\"type\":\"message_stop\"}\n\n")
	}))
	defer ts.Close()

	config := map[string]interface{}{
		"model_name": "claude-sonnet-4-20250514",
		"base_url":   ts.URL,
	}
	client, err := llm.NewStreamingClient("ANTHROPIC", config, map[string]interface{}{"api_key": "test-key"})
	require.NoError(t, err)
	defer client.Close()

	reader, err := client.StreamChatCompletion(
		[]llm.ChatMessage{
			{Role: "system", Content: "You are Malek"},
			{Role: "user", Content: "Who are you?"},
		},
		llm.StreamOptions{},
	)
	require.NoError(t, err)
	defer reader.Close()

	var body map[string]interface{}
	require.NoError(t, json.Unmarshal(capturedBody, &body))

	assert.Equal(t, "You are Malek", body["system"], "System message should be top-level parameter")

	messages, ok := body["messages"].([]interface{})
	require.True(t, ok)
	for _, msg := range messages {
		m := msg.(map[string]interface{})
		assert.NotEqual(t, "system", m["role"], "Messages array should not contain system role")
	}
}
