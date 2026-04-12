package llm_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/unifiedui/agent-service/internal/services/agents/llm"
)

// --- Factory: NewStreamingClient constructor tests ---

func TestNewStreamingClient_OpenAI_Success(t *testing.T) {
	config := map[string]interface{}{"model_name": "gpt-4o-mini"}
	secret := map[string]interface{}{"api_key": "sk-test"}
	client, err := llm.NewStreamingClient("OPENAI", config, secret)
	require.NoError(t, err)
	assert.NotNil(t, client)
	client.Close()
}

func TestNewStreamingClient_OpenAI_MissingModel(t *testing.T) {
	_, err := llm.NewStreamingClient("OPENAI", map[string]interface{}{}, nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "model_name")
}

func TestNewStreamingClient_OpenAI_CustomBaseURL(t *testing.T) {
	config := map[string]interface{}{
		"model_name": "gpt-4o-mini",
		"base_url":   "https://custom-proxy.example.com",
	}
	client, err := llm.NewStreamingClient("OPENAI", config, nil)
	require.NoError(t, err)
	assert.NotNil(t, client)
	client.Close()
}

func TestNewStreamingClient_AzureOpenAI_Success(t *testing.T) {
	config := map[string]interface{}{
		"endpoint":        "https://my-resource.openai.azure.com",
		"deployment_name": "gpt-5-mini",
	}
	secret := map[string]interface{}{"api_key": "key123"}
	client, err := llm.NewStreamingClient("AZURE_OPENAI", config, secret)
	require.NoError(t, err)
	assert.NotNil(t, client)
	client.Close()
}

func TestNewStreamingClient_AzureOpenAI_MissingEndpoint(t *testing.T) {
	config := map[string]interface{}{"deployment_name": "gpt-5-mini"}
	_, err := llm.NewStreamingClient("AZURE_OPENAI", config, nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "endpoint")
}

func TestNewStreamingClient_AzureOpenAI_MissingDeployment(t *testing.T) {
	config := map[string]interface{}{"endpoint": "https://x.openai.azure.com"}
	_, err := llm.NewStreamingClient("AZURE_OPENAI", config, nil)
	assert.Error(t, err)
}

func TestNewStreamingClient_Anthropic_Success(t *testing.T) {
	config := map[string]interface{}{"model_name": "claude-sonnet-4-20250514"}
	secret := map[string]interface{}{"api_key": "sk-ant-test"}
	client, err := llm.NewStreamingClient("ANTHROPIC", config, secret)
	require.NoError(t, err)
	assert.NotNil(t, client)
	client.Close()
}

func TestNewStreamingClient_Anthropic_MissingModel(t *testing.T) {
	_, err := llm.NewStreamingClient("ANTHROPIC", map[string]interface{}{}, nil)
	assert.Error(t, err)
}

func TestNewStreamingClient_GoogleGenAI_Success(t *testing.T) {
	config := map[string]interface{}{"model_name": "gemini-2.5-flash"}
	client, err := llm.NewStreamingClient("GOOGLE_GENAI", config, nil)
	require.NoError(t, err)
	assert.NotNil(t, client)
	client.Close()
}

func TestNewStreamingClient_GoogleGenAI_MissingModel(t *testing.T) {
	_, err := llm.NewStreamingClient("GOOGLE_GENAI", map[string]interface{}{}, nil)
	assert.Error(t, err)
}

func TestNewStreamingClient_Ollama_Success(t *testing.T) {
	config := map[string]interface{}{"model_name": "llama3.2:3b"}
	client, err := llm.NewStreamingClient("OLLAMA", config, nil)
	require.NoError(t, err)
	assert.NotNil(t, client)
	client.Close()
}

func TestNewStreamingClient_Ollama_MissingModel(t *testing.T) {
	_, err := llm.NewStreamingClient("OLLAMA", map[string]interface{}{}, nil)
	assert.Error(t, err)
}

func TestNewStreamingClient_Ollama_CustomBaseURL(t *testing.T) {
	config := map[string]interface{}{
		"model_name": "llama3.2:3b",
		"base_url":   "http://custom-host:11434",
	}
	client, err := llm.NewStreamingClient("OLLAMA", config, nil)
	require.NoError(t, err)
	assert.NotNil(t, client)
	client.Close()
}

func TestNewStreamingClient_Mistral_Success(t *testing.T) {
	config := map[string]interface{}{"model_name": "mistral-small-latest"}
	secret := map[string]interface{}{"api_key": "test-key"}
	client, err := llm.NewStreamingClient("MISTRAL", config, secret)
	require.NoError(t, err)
	assert.NotNil(t, client)
	client.Close()
}

func TestNewStreamingClient_Mistral_MissingModel(t *testing.T) {
	_, err := llm.NewStreamingClient("MISTRAL", map[string]interface{}{}, nil)
	assert.Error(t, err)
}

func TestNewStreamingClient_Groq_Success(t *testing.T) {
	config := map[string]interface{}{"model_name": "llama-3.3-70b-versatile"}
	secret := map[string]interface{}{"api_key": "gsk-test"}
	client, err := llm.NewStreamingClient("GROQ", config, secret)
	require.NoError(t, err)
	assert.NotNil(t, client)
	client.Close()
}

func TestNewStreamingClient_Groq_MissingModel(t *testing.T) {
	_, err := llm.NewStreamingClient("GROQ", map[string]interface{}{}, nil)
	assert.Error(t, err)
}

func TestNewStreamingClient_UnsupportedProvider(t *testing.T) {
	_, err := llm.NewStreamingClient("UNKNOWN", map[string]interface{}{}, nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unsupported LLM provider")
}

func TestNewStreamingClient_NilCredentialSecret(t *testing.T) {
	config := map[string]interface{}{"model_name": "gpt-4o-mini"}
	client, err := llm.NewStreamingClient("OPENAI", config, nil)
	require.NoError(t, err)
	assert.NotNil(t, client)
	client.Close()
}

func TestNewStreamingClient_EmptyApiKey(t *testing.T) {
	config := map[string]interface{}{"model_name": "gpt-4o-mini"}
	secret := map[string]interface{}{"api_key": ""}
	client, err := llm.NewStreamingClient("OPENAI", config, secret)
	require.NoError(t, err)
	assert.NotNil(t, client)
	client.Close()
}
