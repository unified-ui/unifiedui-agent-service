package ai_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/unifiedui/agent-service/internal/services/ai"
)

func openAIHandler(content string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		resp := map[string]interface{}{
			"choices": []map[string]interface{}{
				{
					"message": map[string]interface{}{
						"content": content,
					},
					"finish_reason": "stop",
				},
			},
			"model": "test-model",
			"usage": map[string]interface{}{
				"prompt_tokens":     10,
				"completion_tokens": 5,
			},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}
}

func ollamaHandler(content string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		resp := map[string]interface{}{
			"message": map[string]interface{}{
				"content": content,
			},
			"model": "llama3",
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}
}

func errorHandler(status int, body string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(status)
		w.Write([]byte(body))
	}
}

// --- NewLLMClient constructor tests ---

func TestNewLLMClient_OpenAI_Success(t *testing.T) {
	config := map[string]interface{}{"model_name": "gpt-4"}
	secret := map[string]interface{}{"api_key": "sk-test"}
	client, err := ai.NewLLMClient("OPENAI", config, secret)
	require.NoError(t, err)
	assert.NotNil(t, client)
}

func TestNewLLMClient_OpenAI_MissingModel(t *testing.T) {
	config := map[string]interface{}{}
	_, err := ai.NewLLMClient("OPENAI", config, nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "model_name")
}

func TestNewLLMClient_AzureOpenAI_Success(t *testing.T) {
	config := map[string]interface{}{
		"endpoint":        "https://myendpoint.openai.azure.com",
		"deployment_name": "gpt-4",
	}
	secret := map[string]interface{}{"api_key": "key123"}
	client, err := ai.NewLLMClient("AZURE_OPENAI", config, secret)
	require.NoError(t, err)
	assert.NotNil(t, client)
}

func TestNewLLMClient_AzureOpenAI_MissingEndpoint(t *testing.T) {
	config := map[string]interface{}{"deployment_name": "gpt-4"}
	_, err := ai.NewLLMClient("AZURE_OPENAI", config, nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "endpoint")
}

func TestNewLLMClient_AzureOpenAI_MissingDeployment(t *testing.T) {
	config := map[string]interface{}{"endpoint": "https://x.openai.azure.com"}
	_, err := ai.NewLLMClient("AZURE_OPENAI", config, nil)
	assert.Error(t, err)
}

func TestNewLLMClient_Anthropic_Success(t *testing.T) {
	config := map[string]interface{}{"model_name": "claude-3"}
	client, err := ai.NewLLMClient("ANTHROPIC", config, nil)
	require.NoError(t, err)
	assert.NotNil(t, client)
}

func TestNewLLMClient_Anthropic_MissingModel(t *testing.T) {
	_, err := ai.NewLLMClient("ANTHROPIC", map[string]interface{}{}, nil)
	assert.Error(t, err)
}

func TestNewLLMClient_GoogleGenAI_Success(t *testing.T) {
	config := map[string]interface{}{"model_name": "gemini-pro"}
	client, err := ai.NewLLMClient("GOOGLE_GENAI", config, nil)
	require.NoError(t, err)
	assert.NotNil(t, client)
}

func TestNewLLMClient_GoogleGenAI_MissingModel(t *testing.T) {
	_, err := ai.NewLLMClient("GOOGLE_GENAI", map[string]interface{}{}, nil)
	assert.Error(t, err)
}

func TestNewLLMClient_Ollama_Success(t *testing.T) {
	config := map[string]interface{}{"model_name": "llama3"}
	client, err := ai.NewLLMClient("OLLAMA", config, nil)
	require.NoError(t, err)
	assert.NotNil(t, client)
}

func TestNewLLMClient_Ollama_MissingModel(t *testing.T) {
	_, err := ai.NewLLMClient("OLLAMA", map[string]interface{}{}, nil)
	assert.Error(t, err)
}

func TestNewLLMClient_Mistral_Success(t *testing.T) {
	config := map[string]interface{}{"model_name": "mistral-large"}
	client, err := ai.NewLLMClient("MISTRAL", config, nil)
	require.NoError(t, err)
	assert.NotNil(t, client)
}

func TestNewLLMClient_Mistral_MissingModel(t *testing.T) {
	_, err := ai.NewLLMClient("MISTRAL", map[string]interface{}{}, nil)
	assert.Error(t, err)
}

func TestNewLLMClient_Groq_Success(t *testing.T) {
	config := map[string]interface{}{"model_name": "llama3-70b"}
	client, err := ai.NewLLMClient("GROQ", config, nil)
	require.NoError(t, err)
	assert.NotNil(t, client)
}

func TestNewLLMClient_Groq_MissingModel(t *testing.T) {
	_, err := ai.NewLLMClient("GROQ", map[string]interface{}{}, nil)
	assert.Error(t, err)
}

func TestNewLLMClient_UnsupportedProvider(t *testing.T) {
	_, err := ai.NewLLMClient("UNKNOWN", map[string]interface{}{}, nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unsupported provider")
}

func TestNewLLMClient_NilCredentialSecret(t *testing.T) {
	config := map[string]interface{}{"model_name": "gpt-4"}
	client, err := ai.NewLLMClient("OPENAI", config, nil)
	require.NoError(t, err)
	assert.NotNil(t, client)
}

// --- OpenAI ChatCompletion via httptest ---

func TestOpenAI_ChatCompletion_Success(t *testing.T) {
	ts := httptest.NewServer(openAIHandler("Hello from OpenAI"))
	defer ts.Close()

	config := map[string]interface{}{
		"model_name": "gpt-4",
		"base_url":   ts.URL,
	}
	secret := map[string]interface{}{"api_key": "sk-test"}
	client, err := ai.NewLLMClient("OPENAI", config, secret)
	require.NoError(t, err)

	messages := []ai.ChatMessage{{Role: "user", Content: "Hi"}}
	result, err := client.ChatCompletion(context.Background(), messages)
	require.NoError(t, err)
	assert.Equal(t, "Hello from OpenAI", result.Content)
	assert.Equal(t, "test-model", result.Model)
	assert.Equal(t, 10, result.TokensInput)
	assert.Equal(t, 5, result.TokensOutput)
	assert.Greater(t, result.LatencyMs, int64(-1))
}

func TestOpenAI_ChatCompletion_ServerError(t *testing.T) {
	ts := httptest.NewServer(errorHandler(500, "internal error"))
	defer ts.Close()

	config := map[string]interface{}{
		"model_name": "gpt-4",
		"base_url":   ts.URL,
	}
	client, err := ai.NewLLMClient("OPENAI", config, nil)
	require.NoError(t, err)

	_, err = client.ChatCompletion(context.Background(), []ai.ChatMessage{{Role: "user", Content: "Hi"}})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "500")
}

func TestOpenAI_ChatCompletion_InvalidJSON(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("not json"))
	}))
	defer ts.Close()

	config := map[string]interface{}{
		"model_name": "gpt-4",
		"base_url":   ts.URL,
	}
	client, err := ai.NewLLMClient("OPENAI", config, nil)
	require.NoError(t, err)

	_, err = client.ChatCompletion(context.Background(), []ai.ChatMessage{{Role: "user", Content: "Hi"}})
	assert.Error(t, err)
}

func TestOpenAI_ChatCompletion_EmptyChoices(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := map[string]interface{}{
			"choices": []interface{}{},
			"model":   "gpt-4",
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer ts.Close()

	config := map[string]interface{}{
		"model_name": "gpt-4",
		"base_url":   ts.URL,
	}
	client, err := ai.NewLLMClient("OPENAI", config, nil)
	require.NoError(t, err)

	_, err = client.ChatCompletion(context.Background(), []ai.ChatMessage{{Role: "user", Content: "Hi"}})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no choices")
}

func TestOpenAI_ChatCompletion_EmptyContent(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := map[string]interface{}{
			"choices": []map[string]interface{}{
				{
					"message":       map[string]interface{}{"content": "", "refusal": "content_filter"},
					"finish_reason": "content_filter",
				},
			},
			"model": "gpt-4",
			"usage": map[string]interface{}{"prompt_tokens": 10, "completion_tokens": 0},
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer ts.Close()

	config := map[string]interface{}{
		"model_name": "gpt-4",
		"base_url":   ts.URL,
	}
	client, err := ai.NewLLMClient("OPENAI", config, nil)
	require.NoError(t, err)

	result, err := client.ChatCompletion(context.Background(), []ai.ChatMessage{{Role: "user", Content: "Hi"}})
	require.NoError(t, err)
	assert.Equal(t, "", result.Content)
}

func TestOpenAI_ChatCompletion_WithOrganization(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "org-test", r.Header.Get("OpenAI-Organization"))
		openAIHandler("OK")(w, r)
	}))
	defer ts.Close()

	config := map[string]interface{}{
		"model_name":   "gpt-4",
		"base_url":     ts.URL,
		"organization": "org-test",
	}
	client, err := ai.NewLLMClient("OPENAI", config, map[string]interface{}{"api_key": "sk-test"})
	require.NoError(t, err)

	result, err := client.ChatCompletion(context.Background(), []ai.ChatMessage{{Role: "user", Content: "Hi"}})
	require.NoError(t, err)
	assert.Equal(t, "OK", result.Content)
}

func TestOpenAI_ChatCompletion_CancelledContext(t *testing.T) {
	ts := httptest.NewServer(openAIHandler("OK"))
	defer ts.Close()

	config := map[string]interface{}{
		"model_name": "gpt-4",
		"base_url":   ts.URL,
	}
	client, err := ai.NewLLMClient("OPENAI", config, nil)
	require.NoError(t, err)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err = client.ChatCompletion(ctx, []ai.ChatMessage{{Role: "user", Content: "Hi"}})
	assert.Error(t, err)
}

// --- Azure OpenAI ChatCompletion via httptest ---

func TestAzureOpenAI_ChatCompletion_Success(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Contains(t, r.URL.Path, "/openai/deployments/gpt-4/chat/completions")
		assert.Contains(t, r.URL.RawQuery, "api-version=")
		assert.Equal(t, "test-key", r.Header.Get("api-key"))
		openAIHandler("Azure response")(w, r)
	}))
	defer ts.Close()

	config := map[string]interface{}{
		"endpoint":        ts.URL,
		"deployment_name": "gpt-4",
		"api_version":     "2024-12-01-preview",
	}
	secret := map[string]interface{}{"api_key": "test-key"}
	client, err := ai.NewLLMClient("AZURE_OPENAI", config, secret)
	require.NoError(t, err)

	result, err := client.ChatCompletion(context.Background(), []ai.ChatMessage{{Role: "user", Content: "Hi"}})
	require.NoError(t, err)
	assert.Equal(t, "Azure response", result.Content)
}

func TestAzureOpenAI_ChatCompletion_DefaultApiVersion(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Contains(t, r.URL.RawQuery, "api-version=2024-12-01-preview")
		openAIHandler("OK")(w, r)
	}))
	defer ts.Close()

	config := map[string]interface{}{
		"endpoint":        ts.URL,
		"deployment_name": "gpt-4",
	}
	client, err := ai.NewLLMClient("AZURE_OPENAI", config, nil)
	require.NoError(t, err)

	_, err = client.ChatCompletion(context.Background(), []ai.ChatMessage{{Role: "user", Content: "Hi"}})
	require.NoError(t, err)
}

func TestAzureOpenAI_ChatCompletion_ServerError(t *testing.T) {
	ts := httptest.NewServer(errorHandler(429, "rate limited"))
	defer ts.Close()

	config := map[string]interface{}{
		"endpoint":        ts.URL,
		"deployment_name": "gpt-4",
	}
	client, err := ai.NewLLMClient("AZURE_OPENAI", config, nil)
	require.NoError(t, err)

	_, err = client.ChatCompletion(context.Background(), []ai.ChatMessage{{Role: "user", Content: "Hi"}})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "429")
}

// --- Ollama ChatCompletion via httptest ---

func TestOllama_ChatCompletion_Success(t *testing.T) {
	ts := httptest.NewServer(ollamaHandler("Ollama response"))
	defer ts.Close()

	config := map[string]interface{}{
		"model_name": "llama3",
		"base_url":   ts.URL,
	}
	client, err := ai.NewLLMClient("OLLAMA", config, nil)
	require.NoError(t, err)

	result, err := client.ChatCompletion(context.Background(), []ai.ChatMessage{{Role: "user", Content: "Hi"}})
	require.NoError(t, err)
	assert.Equal(t, "Ollama response", result.Content)
	assert.Equal(t, "llama3", result.Model)
}

func TestOllama_ChatCompletion_ServerError(t *testing.T) {
	ts := httptest.NewServer(errorHandler(500, "error"))
	defer ts.Close()

	config := map[string]interface{}{
		"model_name": "llama3",
		"base_url":   ts.URL,
	}
	client, err := ai.NewLLMClient("OLLAMA", config, nil)
	require.NoError(t, err)

	_, err = client.ChatCompletion(context.Background(), []ai.ChatMessage{{Role: "user", Content: "Hi"}})
	assert.Error(t, err)
}

func TestOllama_ChatCompletion_InvalidJSON(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("not json"))
	}))
	defer ts.Close()

	config := map[string]interface{}{
		"model_name": "llama3",
		"base_url":   ts.URL,
	}
	client, err := ai.NewLLMClient("OLLAMA", config, nil)
	require.NoError(t, err)

	_, err = client.ChatCompletion(context.Background(), []ai.ChatMessage{{Role: "user", Content: "Hi"}})
	assert.Error(t, err)
}

func TestOllama_DefaultBaseURL(t *testing.T) {
	config := map[string]interface{}{"model_name": "llama3"}
	client, err := ai.NewLLMClient("OLLAMA", config, nil)
	require.NoError(t, err)
	assert.NotNil(t, client)
}

func TestOllama_ChatCompletion_EmptyContent(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := map[string]interface{}{
			"message": map[string]interface{}{
				"content": "",
			},
			"model": "llama3",
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer ts.Close()

	config := map[string]interface{}{
		"model_name": "llama3",
		"base_url":   ts.URL,
	}
	client, err := ai.NewLLMClient("OLLAMA", config, nil)
	require.NoError(t, err)

	result, err := client.ChatCompletion(context.Background(), []ai.ChatMessage{{Role: "user", Content: "Hi"}})
	require.NoError(t, err)
	assert.Equal(t, "", result.Content)
}

// --- Azure OpenAI ChatCompletion additional tests ---

func TestAzureOpenAI_ChatCompletion_InvalidJSON(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("not json"))
	}))
	defer ts.Close()

	config := map[string]interface{}{
		"endpoint":        ts.URL,
		"deployment_name": "gpt-4",
	}
	client, err := ai.NewLLMClient("AZURE_OPENAI", config, nil)
	require.NoError(t, err)

	_, err = client.ChatCompletion(context.Background(), []ai.ChatMessage{{Role: "user", Content: "Hi"}})
	assert.Error(t, err)
}

func TestAzureOpenAI_ChatCompletion_EmptyChoices(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := map[string]interface{}{
			"choices": []interface{}{},
			"model":   "gpt-4",
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer ts.Close()

	config := map[string]interface{}{
		"endpoint":        ts.URL,
		"deployment_name": "gpt-4",
	}
	client, err := ai.NewLLMClient("AZURE_OPENAI", config, nil)
	require.NoError(t, err)

	_, err = client.ChatCompletion(context.Background(), []ai.ChatMessage{{Role: "user", Content: "Hi"}})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no choices")
}

func TestAzureOpenAI_ChatCompletion_EmptyContent(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := map[string]interface{}{
			"choices": []map[string]interface{}{
				{
					"message":       map[string]interface{}{"content": ""},
					"finish_reason": "stop",
				},
			},
			"model": "gpt-4",
			"usage": map[string]interface{}{"prompt_tokens": 10, "completion_tokens": 0},
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer ts.Close()

	config := map[string]interface{}{
		"endpoint":        ts.URL,
		"deployment_name": "gpt-4",
	}
	client, err := ai.NewLLMClient("AZURE_OPENAI", config, nil)
	require.NoError(t, err)

	result, err := client.ChatCompletion(context.Background(), []ai.ChatMessage{{Role: "user", Content: "Hi"}})
	require.NoError(t, err)
	assert.Equal(t, "", result.Content)
}

// --- Google GenAI ChatCompletion tests ---

func geminiHandler(content string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		resp := map[string]interface{}{
			"candidates": []map[string]interface{}{
				{
					"content": map[string]interface{}{
						"parts": []map[string]interface{}{
							{"text": content},
						},
					},
				},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}
}

func TestGoogleGenAI_ChatCompletion_Success(t *testing.T) {
	ts := httptest.NewServer(geminiHandler("Gemini response"))
	defer ts.Close()

	config := map[string]interface{}{
		"model_name": "gemini-pro",
		"base_url":   ts.URL,
	}
	client, err := ai.NewLLMClientWithBaseURL("GOOGLE_GENAI", config, map[string]interface{}{"api_key": "test-api-key"}, ts.URL)
	require.NoError(t, err)

	result, err := client.ChatCompletion(context.Background(), []ai.ChatMessage{{Role: "user", Content: "Hi"}})
	require.NoError(t, err)
	assert.Equal(t, "Gemini response", result.Content)
	assert.Equal(t, "gemini", result.Model)
}

func TestGoogleGenAI_ChatCompletion_ServerError(t *testing.T) {
	ts := httptest.NewServer(errorHandler(500, "internal error"))
	defer ts.Close()

	config := map[string]interface{}{
		"model_name": "gemini-pro",
	}
	client, err := ai.NewLLMClientWithBaseURL("GOOGLE_GENAI", config, nil, ts.URL)
	require.NoError(t, err)

	_, err = client.ChatCompletion(context.Background(), []ai.ChatMessage{{Role: "user", Content: "Hi"}})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "500")
}

func TestGoogleGenAI_ChatCompletion_InvalidJSON(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("not json"))
	}))
	defer ts.Close()

	config := map[string]interface{}{
		"model_name": "gemini-pro",
	}
	client, err := ai.NewLLMClientWithBaseURL("GOOGLE_GENAI", config, nil, ts.URL)
	require.NoError(t, err)

	_, err = client.ChatCompletion(context.Background(), []ai.ChatMessage{{Role: "user", Content: "Hi"}})
	assert.Error(t, err)
}

func TestGoogleGenAI_ChatCompletion_EmptyCandidates(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := map[string]interface{}{
			"candidates": []interface{}{},
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer ts.Close()

	config := map[string]interface{}{
		"model_name": "gemini-pro",
	}
	client, err := ai.NewLLMClientWithBaseURL("GOOGLE_GENAI", config, nil, ts.URL)
	require.NoError(t, err)

	_, err = client.ChatCompletion(context.Background(), []ai.ChatMessage{{Role: "user", Content: "Hi"}})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no content")
}

func TestGoogleGenAI_ChatCompletion_EmptyParts(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := map[string]interface{}{
			"candidates": []map[string]interface{}{
				{
					"content": map[string]interface{}{
						"parts": []interface{}{},
					},
				},
			},
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer ts.Close()

	config := map[string]interface{}{
		"model_name": "gemini-pro",
	}
	client, err := ai.NewLLMClientWithBaseURL("GOOGLE_GENAI", config, nil, ts.URL)
	require.NoError(t, err)

	_, err = client.ChatCompletion(context.Background(), []ai.ChatMessage{{Role: "user", Content: "Hi"}})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no content")
}

// --- Anthropic ChatCompletion tests ---

func anthropicHandler(content string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		resp := map[string]interface{}{
			"content": []map[string]interface{}{
				{
					"type": "text",
					"text": content,
				},
			},
			"model": "claude-3",
			"usage": map[string]interface{}{
				"input_tokens":  15,
				"output_tokens": 20,
			},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}
}

func TestAnthropic_ChatCompletion_Success(t *testing.T) {
	ts := httptest.NewServer(anthropicHandler("Anthropic response"))
	defer ts.Close()

	config := map[string]interface{}{
		"model_name": "claude-3",
	}
	client, err := ai.NewLLMClientWithBaseURL("ANTHROPIC", config, map[string]interface{}{"api_key": "test-key"}, ts.URL)
	require.NoError(t, err)

	result, err := client.ChatCompletion(context.Background(), []ai.ChatMessage{
		{Role: "system", Content: "You are helpful"},
		{Role: "user", Content: "Hi"},
	})
	require.NoError(t, err)
	assert.Equal(t, "Anthropic response", result.Content)
	assert.Equal(t, "claude-3", result.Model)
	assert.Equal(t, 15, result.TokensInput)
	assert.Equal(t, 20, result.TokensOutput)
}

func TestAnthropic_ChatCompletion_ServerError(t *testing.T) {
	ts := httptest.NewServer(errorHandler(500, "internal error"))
	defer ts.Close()

	config := map[string]interface{}{
		"model_name": "claude-3",
	}
	client, err := ai.NewLLMClientWithBaseURL("ANTHROPIC", config, nil, ts.URL)
	require.NoError(t, err)

	_, err = client.ChatCompletion(context.Background(), []ai.ChatMessage{{Role: "user", Content: "Hi"}})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "500")
}

func TestAnthropic_ChatCompletion_InvalidJSON(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("not json"))
	}))
	defer ts.Close()

	config := map[string]interface{}{
		"model_name": "claude-3",
	}
	client, err := ai.NewLLMClientWithBaseURL("ANTHROPIC", config, nil, ts.URL)
	require.NoError(t, err)

	_, err = client.ChatCompletion(context.Background(), []ai.ChatMessage{{Role: "user", Content: "Hi"}})
	assert.Error(t, err)
}

func TestAnthropic_ChatCompletion_EmptyContent(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := map[string]interface{}{
			"content": []interface{}{},
			"model":   "claude-3",
			"usage":   map[string]interface{}{"input_tokens": 10, "output_tokens": 0},
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer ts.Close()

	config := map[string]interface{}{
		"model_name": "claude-3",
	}
	client, err := ai.NewLLMClientWithBaseURL("ANTHROPIC", config, nil, ts.URL)
	require.NoError(t, err)

	result, err := client.ChatCompletion(context.Background(), []ai.ChatMessage{{Role: "user", Content: "Hi"}})
	require.NoError(t, err)
	assert.Equal(t, "", result.Content)
}

// --- Mistral ChatCompletion tests ---

func TestMistral_ChatCompletion_Success(t *testing.T) {
	ts := httptest.NewServer(openAIHandler("Mistral response"))
	defer ts.Close()

	config := map[string]interface{}{
		"model_name": "mistral-large",
	}
	client, err := ai.NewLLMClientWithBaseURL("MISTRAL", config, map[string]interface{}{"api_key": "test-key"}, ts.URL)
	require.NoError(t, err)

	result, err := client.ChatCompletion(context.Background(), []ai.ChatMessage{{Role: "user", Content: "Hi"}})
	require.NoError(t, err)
	assert.Equal(t, "Mistral response", result.Content)
}

func TestMistral_ChatCompletion_ServerError(t *testing.T) {
	ts := httptest.NewServer(errorHandler(500, "internal error"))
	defer ts.Close()

	config := map[string]interface{}{
		"model_name": "mistral-large",
	}
	client, err := ai.NewLLMClientWithBaseURL("MISTRAL", config, nil, ts.URL)
	require.NoError(t, err)

	_, err = client.ChatCompletion(context.Background(), []ai.ChatMessage{{Role: "user", Content: "Hi"}})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "500")
}

func TestMistral_ChatCompletion_InvalidJSON(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("not json"))
	}))
	defer ts.Close()

	config := map[string]interface{}{
		"model_name": "mistral-large",
	}
	client, err := ai.NewLLMClientWithBaseURL("MISTRAL", config, nil, ts.URL)
	require.NoError(t, err)

	_, err = client.ChatCompletion(context.Background(), []ai.ChatMessage{{Role: "user", Content: "Hi"}})
	assert.Error(t, err)
}

func TestMistral_ChatCompletion_EmptyChoices(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := map[string]interface{}{
			"choices": []interface{}{},
			"model":   "mistral-large",
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer ts.Close()

	config := map[string]interface{}{
		"model_name": "mistral-large",
	}
	client, err := ai.NewLLMClientWithBaseURL("MISTRAL", config, nil, ts.URL)
	require.NoError(t, err)

	_, err = client.ChatCompletion(context.Background(), []ai.ChatMessage{{Role: "user", Content: "Hi"}})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no choices")
}

// --- Groq ChatCompletion tests ---

func TestGroq_ChatCompletion_Success(t *testing.T) {
	ts := httptest.NewServer(openAIHandler("Groq response"))
	defer ts.Close()

	config := map[string]interface{}{
		"model_name": "llama3-70b",
	}
	client, err := ai.NewLLMClientWithBaseURL("GROQ", config, map[string]interface{}{"api_key": "test-key"}, ts.URL)
	require.NoError(t, err)

	result, err := client.ChatCompletion(context.Background(), []ai.ChatMessage{{Role: "user", Content: "Hi"}})
	require.NoError(t, err)
	assert.Equal(t, "Groq response", result.Content)
}

func TestGroq_ChatCompletion_ServerError(t *testing.T) {
	ts := httptest.NewServer(errorHandler(500, "internal error"))
	defer ts.Close()

	config := map[string]interface{}{
		"model_name": "llama3-70b",
	}
	client, err := ai.NewLLMClientWithBaseURL("GROQ", config, nil, ts.URL)
	require.NoError(t, err)

	_, err = client.ChatCompletion(context.Background(), []ai.ChatMessage{{Role: "user", Content: "Hi"}})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "500")
}

func TestGroq_ChatCompletion_InvalidJSON(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("not json"))
	}))
	defer ts.Close()

	config := map[string]interface{}{
		"model_name": "llama3-70b",
	}
	client, err := ai.NewLLMClientWithBaseURL("GROQ", config, nil, ts.URL)
	require.NoError(t, err)

	_, err = client.ChatCompletion(context.Background(), []ai.ChatMessage{{Role: "user", Content: "Hi"}})
	assert.Error(t, err)
}

func TestGroq_ChatCompletion_EmptyChoices(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := map[string]interface{}{
			"choices": []interface{}{},
			"model":   "llama3-70b",
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer ts.Close()

	config := map[string]interface{}{
		"model_name": "llama3-70b",
	}
	client, err := ai.NewLLMClientWithBaseURL("GROQ", config, nil, ts.URL)
	require.NoError(t, err)

	_, err = client.ChatCompletion(context.Background(), []ai.ChatMessage{{Role: "user", Content: "Hi"}})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no choices")
}
