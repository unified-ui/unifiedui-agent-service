// Package ai provides the AI service for LLM interactions.
package ai

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// httpLLMClient is a base for HTTP-based LLM providers.
type httpLLMClient struct {
	httpClient *http.Client
}

func newHTTPClient() *http.Client {
	return &http.Client{Timeout: 60 * time.Second}
}

// --- Azure OpenAI ---

type azureOpenAIClient struct {
	httpLLMClient
	endpoint       string
	apiVersion     string
	deploymentName string
	apiKey         string
}

func newAzureOpenAIClient(config map[string]interface{}, apiKey string) (*azureOpenAIClient, error) {
	endpoint, _ := config["endpoint"].(string)
	apiVersion, _ := config["api_version"].(string)
	deploymentName, _ := config["deployment_name"].(string)

	if endpoint == "" || deploymentName == "" {
		return nil, fmt.Errorf("azure_openai requires endpoint and deployment_name")
	}
	if apiVersion == "" {
		apiVersion = "2024-12-01-preview"
	}

	return &azureOpenAIClient{
		httpLLMClient:  httpLLMClient{httpClient: newHTTPClient()},
		endpoint:       endpoint,
		apiVersion:     apiVersion,
		deploymentName: deploymentName,
		apiKey:         apiKey,
	}, nil
}

// ChatCompletion sends a chat completion request to Azure OpenAI.
func (c *azureOpenAIClient) ChatCompletion(ctx context.Context, messages []ChatMessage) (*ChatCompletionResult, error) {
	url := fmt.Sprintf("%s/openai/deployments/%s/chat/completions?api-version=%s", c.endpoint, c.deploymentName, c.apiVersion)

	body := map[string]interface{}{
		"messages": messages,
	}

	start := time.Now()
	resp, err := c.doRequest(ctx, url, body, map[string]string{"api-key": c.apiKey})
	if err != nil {
		return nil, err
	}
	latency := time.Since(start).Milliseconds()

	return parseOpenAIResponse(resp, latency)
}

func (c *azureOpenAIClient) doRequest(ctx context.Context, url string, body interface{}, headers map[string]string) ([]byte, error) {
	jsonBody, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(jsonBody))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	for k, v := range headers {
		req.Header.Set(k, v)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API returned status %d: %s", resp.StatusCode, string(respBody))
	}

	return respBody, nil
}

// --- OpenAI ---

type openAIClient struct {
	httpLLMClient
	modelName    string
	apiKey       string
	organization string
	baseURL      string
}

func newOpenAIClient(config map[string]interface{}, apiKey string) (*openAIClient, error) {
	modelName, _ := config["model_name"].(string)
	if modelName == "" {
		return nil, fmt.Errorf("openai requires model_name")
	}

	organization, _ := config["organization"].(string)
	baseURL, _ := config["base_url"].(string)
	if baseURL == "" {
		baseURL = "https://api.openai.com"
	}

	return &openAIClient{
		httpLLMClient: httpLLMClient{httpClient: newHTTPClient()},
		modelName:     modelName,
		apiKey:        apiKey,
		organization:  organization,
		baseURL:       baseURL,
	}, nil
}

// ChatCompletion sends a chat completion request to OpenAI.
func (c *openAIClient) ChatCompletion(ctx context.Context, messages []ChatMessage) (*ChatCompletionResult, error) {
	url := fmt.Sprintf("%s/v1/chat/completions", c.baseURL)

	body := map[string]interface{}{
		"model":    c.modelName,
		"messages": messages,
	}

	headers := map[string]string{
		"Authorization": "Bearer " + c.apiKey,
	}
	if c.organization != "" {
		headers["OpenAI-Organization"] = c.organization
	}

	start := time.Now()
	resp, err := doHTTPRequest(ctx, c.httpClient, url, body, headers)
	if err != nil {
		return nil, err
	}
	latency := time.Since(start).Milliseconds()

	return parseOpenAIResponse(resp, latency)
}

// --- Anthropic ---

type anthropicClient struct {
	httpLLMClient
	modelName string
	apiKey    string
}

func newAnthropicClient(config map[string]interface{}, apiKey string) (*anthropicClient, error) {
	modelName, _ := config["model_name"].(string)
	if modelName == "" {
		return nil, fmt.Errorf("anthropic requires model_name")
	}

	return &anthropicClient{
		httpLLMClient: httpLLMClient{httpClient: newHTTPClient()},
		modelName:     modelName,
		apiKey:        apiKey,
	}, nil
}

// ChatCompletion sends a chat completion request to Anthropic.
func (c *anthropicClient) ChatCompletion(ctx context.Context, messages []ChatMessage) (*ChatCompletionResult, error) {
	anthropicMessages := make([]map[string]string, 0)
	systemPrompt := ""

	for _, msg := range messages {
		if msg.Role == "system" {
			systemPrompt = msg.Content
			continue
		}
		anthropicMessages = append(anthropicMessages, map[string]string{
			"role":    msg.Role,
			"content": msg.Content,
		})
	}

	body := map[string]interface{}{
		"model":      c.modelName,
		"max_tokens": 4096,
		"messages":   anthropicMessages,
	}
	if systemPrompt != "" {
		body["system"] = systemPrompt
	}

	headers := map[string]string{
		"x-api-key":         c.apiKey,
		"anthropic-version": "2023-06-01",
	}

	start := time.Now()
	resp, err := doHTTPRequest(ctx, c.httpClient, "https://api.anthropic.com/v1/messages", body, headers)
	if err != nil {
		return nil, err
	}
	latency := time.Since(start).Milliseconds()

	return parseAnthropicResponse(resp, latency)
}

// --- Google GenAI (Gemini) ---

type googleGenAIClient struct {
	httpLLMClient
	modelName string
	apiKey    string
}

func newGoogleGenAIClient(config map[string]interface{}, apiKey string) (*googleGenAIClient, error) {
	modelName, _ := config["model_name"].(string)
	if modelName == "" {
		return nil, fmt.Errorf("google_genai requires model_name")
	}

	return &googleGenAIClient{
		httpLLMClient: httpLLMClient{httpClient: newHTTPClient()},
		modelName:     modelName,
		apiKey:        apiKey,
	}, nil
}

// ChatCompletion sends a chat completion request to Google Gemini.
func (c *googleGenAIClient) ChatCompletion(ctx context.Context, messages []ChatMessage) (*ChatCompletionResult, error) {
	url := fmt.Sprintf("https://generativelanguage.googleapis.com/v1beta/models/%s:generateContent?key=%s", c.modelName, c.apiKey)

	contents := make([]map[string]interface{}, 0)
	for _, msg := range messages {
		role := msg.Role
		if role == "assistant" {
			role = "model"
		}
		if role == "system" {
			role = "user"
		}
		contents = append(contents, map[string]interface{}{
			"role":  role,
			"parts": []map[string]string{{"text": msg.Content}},
		})
	}

	body := map[string]interface{}{
		"contents": contents,
	}

	start := time.Now()
	resp, err := doHTTPRequest(ctx, c.httpClient, url, body, nil)
	if err != nil {
		return nil, err
	}
	latency := time.Since(start).Milliseconds()

	return parseGeminiResponse(resp, latency)
}

// --- Ollama ---

type ollamaClient struct {
	httpLLMClient
	modelName string
	baseURL   string
}

func newOllamaClient(config map[string]interface{}) (*ollamaClient, error) {
	modelName, _ := config["model_name"].(string)
	if modelName == "" {
		return nil, fmt.Errorf("ollama requires model_name")
	}

	baseURL, _ := config["base_url"].(string)
	if baseURL == "" {
		baseURL = "http://localhost:11434"
	}

	return &ollamaClient{
		httpLLMClient: httpLLMClient{httpClient: newHTTPClient()},
		modelName:     modelName,
		baseURL:       baseURL,
	}, nil
}

// ChatCompletion sends a chat completion request to Ollama.
func (c *ollamaClient) ChatCompletion(ctx context.Context, messages []ChatMessage) (*ChatCompletionResult, error) {
	url := fmt.Sprintf("%s/api/chat", c.baseURL)

	body := map[string]interface{}{
		"model":    c.modelName,
		"messages": messages,
		"stream":   false,
	}

	start := time.Now()
	resp, err := doHTTPRequest(ctx, c.httpClient, url, body, nil)
	if err != nil {
		return nil, err
	}
	latency := time.Since(start).Milliseconds()

	return parseOllamaResponse(resp, latency)
}

// --- Mistral ---

type mistralClient struct {
	httpLLMClient
	modelName string
	apiKey    string
}

func newMistralClient(config map[string]interface{}, apiKey string) (*mistralClient, error) {
	modelName, _ := config["model_name"].(string)
	if modelName == "" {
		return nil, fmt.Errorf("mistral requires model_name")
	}

	return &mistralClient{
		httpLLMClient: httpLLMClient{httpClient: newHTTPClient()},
		modelName:     modelName,
		apiKey:        apiKey,
	}, nil
}

// ChatCompletion sends a chat completion request to Mistral.
func (c *mistralClient) ChatCompletion(ctx context.Context, messages []ChatMessage) (*ChatCompletionResult, error) {
	body := map[string]interface{}{
		"model":    c.modelName,
		"messages": messages,
	}

	headers := map[string]string{
		"Authorization": "Bearer " + c.apiKey,
	}

	start := time.Now()
	resp, err := doHTTPRequest(ctx, c.httpClient, "https://api.mistral.ai/v1/chat/completions", body, headers)
	if err != nil {
		return nil, err
	}
	latency := time.Since(start).Milliseconds()

	return parseOpenAIResponse(resp, latency)
}

// --- Groq ---

type groqClient struct {
	httpLLMClient
	modelName string
	apiKey    string
}

func newGroqClient(config map[string]interface{}, apiKey string) (*groqClient, error) {
	modelName, _ := config["model_name"].(string)
	if modelName == "" {
		return nil, fmt.Errorf("groq requires model_name")
	}

	return &groqClient{
		httpLLMClient: httpLLMClient{httpClient: newHTTPClient()},
		modelName:     modelName,
		apiKey:        apiKey,
	}, nil
}

// ChatCompletion sends a chat completion request to Groq (OpenAI-compatible).
func (c *groqClient) ChatCompletion(ctx context.Context, messages []ChatMessage) (*ChatCompletionResult, error) {
	body := map[string]interface{}{
		"model":    c.modelName,
		"messages": messages,
	}

	headers := map[string]string{
		"Authorization": "Bearer " + c.apiKey,
	}

	start := time.Now()
	resp, err := doHTTPRequest(ctx, c.httpClient, "https://api.groq.com/openai/v1/chat/completions", body, headers)
	if err != nil {
		return nil, err
	}
	latency := time.Since(start).Milliseconds()

	return parseOpenAIResponse(resp, latency)
}

// --- Shared Helpers ---

func doHTTPRequest(ctx context.Context, client *http.Client, url string, body interface{}, headers map[string]string) ([]byte, error) {
	jsonBody, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(jsonBody))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	for k, v := range headers {
		req.Header.Set(k, v)
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API returned status %d: %s", resp.StatusCode, string(respBody))
	}

	return respBody, nil
}

func parseOpenAIResponse(data []byte, latencyMs int64) (*ChatCompletionResult, error) {
	var resp struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
		Model string `json:"model"`
		Usage struct {
			PromptTokens     int `json:"prompt_tokens"`
			CompletionTokens int `json:"completion_tokens"`
		} `json:"usage"`
	}

	if err := json.Unmarshal(data, &resp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	if len(resp.Choices) == 0 {
		return nil, fmt.Errorf("no choices in response")
	}

	return &ChatCompletionResult{
		Content:      resp.Choices[0].Message.Content,
		Model:        resp.Model,
		LatencyMs:    latencyMs,
		TokensInput:  resp.Usage.PromptTokens,
		TokensOutput: resp.Usage.CompletionTokens,
	}, nil
}

func parseAnthropicResponse(data []byte, latencyMs int64) (*ChatCompletionResult, error) {
	var resp struct {
		Content []struct {
			Type string `json:"type"`
			Text string `json:"text"`
		} `json:"content"`
		Model string `json:"model"`
		Usage struct {
			InputTokens  int `json:"input_tokens"`
			OutputTokens int `json:"output_tokens"`
		} `json:"usage"`
	}

	if err := json.Unmarshal(data, &resp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	content := ""
	for _, block := range resp.Content {
		if block.Type == "text" {
			content += block.Text
		}
	}

	return &ChatCompletionResult{
		Content:      content,
		Model:        resp.Model,
		LatencyMs:    latencyMs,
		TokensInput:  resp.Usage.InputTokens,
		TokensOutput: resp.Usage.OutputTokens,
	}, nil
}

func parseGeminiResponse(data []byte, latencyMs int64) (*ChatCompletionResult, error) {
	var resp struct {
		Candidates []struct {
			Content struct {
				Parts []struct {
					Text string `json:"text"`
				} `json:"parts"`
			} `json:"content"`
		} `json:"candidates"`
	}

	if err := json.Unmarshal(data, &resp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	if len(resp.Candidates) == 0 || len(resp.Candidates[0].Content.Parts) == 0 {
		return nil, fmt.Errorf("no content in response")
	}

	return &ChatCompletionResult{
		Content:   resp.Candidates[0].Content.Parts[0].Text,
		Model:     "gemini",
		LatencyMs: latencyMs,
	}, nil
}

func parseOllamaResponse(data []byte, latencyMs int64) (*ChatCompletionResult, error) {
	var resp struct {
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
		Model string `json:"model"`
	}

	if err := json.Unmarshal(data, &resp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return &ChatCompletionResult{
		Content:   resp.Message.Content,
		Model:     resp.Model,
		LatencyMs: latencyMs,
	}, nil
}
