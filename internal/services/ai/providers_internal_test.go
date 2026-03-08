package ai

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseOpenAIResponse_Success(t *testing.T) {
	data := []byte(`{
		"choices": [{"message": {"content": "Hello world"}, "finish_reason": "stop"}],
		"model": "gpt-4",
		"usage": {"prompt_tokens": 10, "completion_tokens": 5}
	}`)

	result, err := parseOpenAIResponse(data, 100)
	require.NoError(t, err)
	assert.Equal(t, "Hello world", result.Content)
	assert.Equal(t, "gpt-4", result.Model)
	assert.Equal(t, int64(100), result.LatencyMs)
	assert.Equal(t, 10, result.TokensInput)
	assert.Equal(t, 5, result.TokensOutput)
}

func TestParseOpenAIResponse_NoChoices(t *testing.T) {
	data := []byte(`{"choices": [], "model": "gpt-4"}`)
	_, err := parseOpenAIResponse(data, 100)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "no choices")
}

func TestParseOpenAIResponse_InvalidJSON(t *testing.T) {
	data := []byte(`not json`)
	_, err := parseOpenAIResponse(data, 100)
	require.Error(t, err)
}

func TestParseAnthropicResponse_Success(t *testing.T) {
	data := []byte(`{
		"content": [{"type": "text", "text": "Hello from Claude"}],
		"model": "claude-3",
		"usage": {"input_tokens": 20, "output_tokens": 10}
	}`)

	result, err := parseAnthropicResponse(data, 200)
	require.NoError(t, err)
	assert.Equal(t, "Hello from Claude", result.Content)
	assert.Equal(t, "claude-3", result.Model)
	assert.Equal(t, int64(200), result.LatencyMs)
	assert.Equal(t, 20, result.TokensInput)
	assert.Equal(t, 10, result.TokensOutput)
}

func TestParseAnthropicResponse_MultipleBlocks(t *testing.T) {
	data := []byte(`{
		"content": [
			{"type": "text", "text": "First"},
			{"type": "text", "text": " Second"}
		],
		"model": "claude-3",
		"usage": {"input_tokens": 5, "output_tokens": 10}
	}`)

	result, err := parseAnthropicResponse(data, 150)
	require.NoError(t, err)
	assert.Equal(t, "First Second", result.Content)
}

func TestParseAnthropicResponse_InvalidJSON(t *testing.T) {
	data := []byte(`not json`)
	_, err := parseAnthropicResponse(data, 100)
	require.Error(t, err)
}

func TestParseGeminiResponse_Success(t *testing.T) {
	data := []byte(`{
		"candidates": [{
			"content": {
				"parts": [{"text": "Hello from Gemini"}]
			}
		}]
	}`)

	result, err := parseGeminiResponse(data, 300)
	require.NoError(t, err)
	assert.Equal(t, "Hello from Gemini", result.Content)
	assert.Equal(t, "gemini", result.Model)
	assert.Equal(t, int64(300), result.LatencyMs)
}

func TestParseGeminiResponse_NoCandidates(t *testing.T) {
	data := []byte(`{"candidates": []}`)
	_, err := parseGeminiResponse(data, 100)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "no content")
}

func TestParseGeminiResponse_NoParts(t *testing.T) {
	data := []byte(`{"candidates": [{"content": {"parts": []}}]}`)
	_, err := parseGeminiResponse(data, 100)
	require.Error(t, err)
}

func TestParseGeminiResponse_InvalidJSON(t *testing.T) {
	data := []byte(`not json`)
	_, err := parseGeminiResponse(data, 100)
	require.Error(t, err)
}

func TestParseOllamaResponse_Success(t *testing.T) {
	data := []byte(`{
		"message": {"content": "Hello from Ollama"},
		"model": "llama2"
	}`)

	result, err := parseOllamaResponse(data, 50)
	require.NoError(t, err)
	assert.Equal(t, "Hello from Ollama", result.Content)
	assert.Equal(t, "llama2", result.Model)
	assert.Equal(t, int64(50), result.LatencyMs)
}

func TestParseOllamaResponse_InvalidJSON(t *testing.T) {
	data := []byte(`not json`)
	_, err := parseOllamaResponse(data, 100)
	require.Error(t, err)
}
