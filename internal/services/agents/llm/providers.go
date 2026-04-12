package llm

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

var streamHTTPClient = &http.Client{Timeout: 5 * time.Minute}

// openAICompatibleClient handles OpenAI, Mistral, Groq, and Azure OpenAI streaming.
type openAICompatibleClient struct {
	url                   string
	modelName             string
	authHeader            string
	authValue             string
	extraHeaders          map[string]string
	useMaxCompletionToken bool
	skipTemperature       bool
}

func newOpenAICompatibleStreamClient(config map[string]interface{}, _, baseURL, authHeader, authValue string) (*openAICompatibleClient, error) {
	modelName, _ := config["model_name"].(string)
	if modelName == "" {
		return nil, fmt.Errorf("model_name is required")
	}

	configBaseURL, _ := config["base_url"].(string)
	if configBaseURL != "" {
		baseURL = configBaseURL
	}

	return &openAICompatibleClient{
		url:        fmt.Sprintf("%s/v1/chat/completions", strings.TrimRight(baseURL, "/")),
		modelName:  modelName,
		authHeader: authHeader,
		authValue:  authValue,
	}, nil
}

func newAzureOpenAIStreamClient(config map[string]interface{}, apiKey string) (*openAICompatibleClient, error) {
	endpoint, _ := config["endpoint"].(string)
	deploymentName, _ := config["deployment_name"].(string)

	if endpoint == "" || deploymentName == "" {
		return nil, fmt.Errorf("azure_openai requires endpoint and deployment_name")
	}

	apiVersion, _ := config["api_version"].(string)
	if apiVersion == "" {
		apiVersion = "2024-12-01-preview"
	}

	return &openAICompatibleClient{
		url:                   fmt.Sprintf("%s/openai/deployments/%s/chat/completions?api-version=%s", strings.TrimRight(endpoint, "/"), deploymentName, apiVersion),
		modelName:             "",
		authHeader:            "api-key",
		authValue:             apiKey,
		useMaxCompletionToken: true,
		skipTemperature:       true,
	}, nil
}

func (c *openAICompatibleClient) StreamChatCompletion(messages []ChatMessage, opts StreamOptions) (StreamReader, error) {
	body := map[string]interface{}{
		"messages": messages,
		"stream":   true,
	}
	if c.modelName != "" {
		body["model"] = c.modelName
	}
	if opts.Temperature != nil && !c.skipTemperature {
		body["temperature"] = *opts.Temperature
	}
	if opts.MaxTokens != nil {
		if c.useMaxCompletionToken {
			body["max_completion_tokens"] = *opts.MaxTokens
		} else {
			body["max_tokens"] = *opts.MaxTokens
		}
	}

	resp, err := c.doStreamRequest(body) //nolint:bodyclose // closed by StreamReader.Close()
	if err != nil {
		return nil, err
	}

	return &openAIStreamReader{
		scanner: bufio.NewScanner(resp.Body),
		body:    resp.Body,
	}, nil
}

func (c *openAICompatibleClient) doStreamRequest(body interface{}) (*http.Response, error) {
	jsonBody, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(context.Background(), http.MethodPost, c.url, bytes.NewReader(jsonBody))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "text/event-stream")
	req.Header.Set("User-Agent", "unified-ui-agent-service/1.0")
	if c.authHeader != "" && c.authValue != "" {
		req.Header.Set(c.authHeader, c.authValue)
	}
	for k, v := range c.extraHeaders {
		req.Header.Set(k, v)
	}

	resp, err := streamHTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		_ = resp.Body.Close()
		return nil, fmt.Errorf("API returned status %d: %s", resp.StatusCode, string(respBody))
	}

	return resp, nil
}

func (c *openAICompatibleClient) Close() error { return nil }

// openAIStreamReader reads OpenAI-compatible SSE streams.
type openAIStreamReader struct {
	scanner *bufio.Scanner
	body    io.ReadCloser
}

func (r *openAIStreamReader) Read() (string, error) {
	for r.scanner.Scan() {
		line := r.scanner.Text()
		if !strings.HasPrefix(line, "data: ") {
			continue
		}
		data := strings.TrimPrefix(line, "data: ")
		if data == "[DONE]" {
			return "", io.EOF
		}

		var resp openAIStreamChunk
		if err := json.Unmarshal([]byte(data), &resp); err != nil {
			continue
		}

		if len(resp.Choices) > 0 && resp.Choices[0].Delta.Content != "" {
			return resp.Choices[0].Delta.Content, nil
		}
	}

	if err := r.scanner.Err(); err != nil {
		return "", err
	}
	return "", io.EOF
}

func (r *openAIStreamReader) Close() error {
	return r.body.Close()
}

type openAIStreamChunk struct {
	Choices []struct {
		Delta struct {
			Content string `json:"content"`
		} `json:"delta"`
	} `json:"choices"`
}

// --- Anthropic ---

type anthropicStreamClient struct {
	apiKey    string
	modelName string
	baseURL   string
}

func newAnthropicStreamClient(config map[string]interface{}, apiKey string) (*anthropicStreamClient, error) {
	modelName, _ := config["model_name"].(string)
	if modelName == "" {
		return nil, fmt.Errorf("anthropic requires model_name")
	}

	baseURL, _ := config["base_url"].(string)
	if baseURL == "" {
		baseURL = "https://api.anthropic.com"
	}

	return &anthropicStreamClient{
		apiKey:    apiKey,
		modelName: modelName,
		baseURL:   baseURL,
	}, nil
}

func (c *anthropicStreamClient) StreamChatCompletion(messages []ChatMessage, opts StreamOptions) (StreamReader, error) {
	var systemPrompt string
	filteredMessages := make([]ChatMessage, 0, len(messages))
	for _, msg := range messages {
		if msg.Role == "system" {
			systemPrompt = msg.Content
			continue
		}
		filteredMessages = append(filteredMessages, msg)
	}

	body := map[string]interface{}{
		"model":      c.modelName,
		"messages":   filteredMessages,
		"stream":     true,
		"max_tokens": 4096,
	}
	if systemPrompt != "" {
		body["system"] = systemPrompt
	}
	if opts.MaxTokens != nil {
		body["max_tokens"] = *opts.MaxTokens
	}
	if opts.Temperature != nil {
		body["temperature"] = *opts.Temperature
	}

	jsonBody, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	url := fmt.Sprintf("%s/v1/messages", strings.TrimRight(c.baseURL, "/"))
	req, err := http.NewRequestWithContext(context.Background(), http.MethodPost, url, bytes.NewReader(jsonBody))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "text/event-stream")
	req.Header.Set("User-Agent", "unified-ui-agent-service/1.0")
	req.Header.Set("x-api-key", c.apiKey)
	req.Header.Set("anthropic-version", "2023-06-01")

	resp, err := streamHTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		_ = resp.Body.Close()
		return nil, fmt.Errorf("API returned status %d: %s", resp.StatusCode, string(respBody))
	}

	return &anthropicStreamReader{
		scanner: bufio.NewScanner(resp.Body),
		body:    resp.Body,
	}, nil
}

func (c *anthropicStreamClient) Close() error { return nil }

type anthropicStreamReader struct {
	scanner *bufio.Scanner
	body    io.ReadCloser
}

func (r *anthropicStreamReader) Read() (string, error) {
	for r.scanner.Scan() {
		line := r.scanner.Text()
		if !strings.HasPrefix(line, "data: ") {
			continue
		}
		data := strings.TrimPrefix(line, "data: ")

		var event struct {
			Type  string `json:"type"`
			Delta struct {
				Type string `json:"type"`
				Text string `json:"text"`
			} `json:"delta"`
		}
		if err := json.Unmarshal([]byte(data), &event); err != nil {
			continue
		}

		if event.Type == "message_stop" {
			return "", io.EOF
		}
		if event.Type == "content_block_delta" && event.Delta.Text != "" {
			return event.Delta.Text, nil
		}
	}

	if err := r.scanner.Err(); err != nil {
		return "", err
	}
	return "", io.EOF
}

func (r *anthropicStreamReader) Close() error {
	return r.body.Close()
}

// --- Google GenAI (Gemini) ---

type googleGenAIStreamClient struct {
	apiKey    string
	modelName string
	baseURL   string
}

func newGoogleGenAIStreamClient(config map[string]interface{}, apiKey string) (*googleGenAIStreamClient, error) {
	modelName, _ := config["model_name"].(string)
	if modelName == "" {
		return nil, fmt.Errorf("google_genai requires model_name")
	}

	return &googleGenAIStreamClient{
		apiKey:    apiKey,
		modelName: modelName,
		baseURL:   "https://generativelanguage.googleapis.com",
	}, nil
}

func (c *googleGenAIStreamClient) StreamChatCompletion(messages []ChatMessage, opts StreamOptions) (StreamReader, error) {
	contents := make([]map[string]interface{}, 0, len(messages))
	for _, msg := range messages {
		role := msg.Role
		if role == "assistant" {
			role = "model"
		}
		if role == "system" {
			continue
		}
		contents = append(contents, map[string]interface{}{
			"role":  role,
			"parts": []map[string]string{{"text": msg.Content}},
		})
	}

	body := map[string]interface{}{
		"contents": contents,
	}

	generationConfig := map[string]interface{}{}
	if opts.Temperature != nil {
		generationConfig["temperature"] = *opts.Temperature
	}
	if opts.MaxTokens != nil {
		generationConfig["maxOutputTokens"] = *opts.MaxTokens
	}
	if len(generationConfig) > 0 {
		body["generationConfig"] = generationConfig
	}

	var systemInstruction string
	for _, msg := range messages {
		if msg.Role == "system" {
			systemInstruction = msg.Content
			break
		}
	}
	if systemInstruction != "" {
		body["systemInstruction"] = map[string]interface{}{
			"parts": []map[string]string{{"text": systemInstruction}},
		}
	}

	jsonBody, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	url := fmt.Sprintf("%s/v1beta/models/%s:streamGenerateContent?alt=sse&key=%s", c.baseURL, c.modelName, c.apiKey)
	req, err := http.NewRequestWithContext(context.Background(), http.MethodPost, url, bytes.NewReader(jsonBody))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "unified-ui-agent-service/1.0")

	resp, err := streamHTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		_ = resp.Body.Close()
		return nil, fmt.Errorf("API returned status %d: %s", resp.StatusCode, string(respBody))
	}

	return &googleStreamReader{
		scanner: bufio.NewScanner(resp.Body),
		body:    resp.Body,
	}, nil
}

func (c *googleGenAIStreamClient) Close() error { return nil }

type googleStreamReader struct {
	scanner *bufio.Scanner
	body    io.ReadCloser
}

func (r *googleStreamReader) Read() (string, error) {
	for r.scanner.Scan() {
		line := r.scanner.Text()
		if !strings.HasPrefix(line, "data: ") {
			continue
		}
		data := strings.TrimPrefix(line, "data: ")

		var resp struct {
			Candidates []struct {
				Content struct {
					Parts []struct {
						Text string `json:"text"`
					} `json:"parts"`
				} `json:"content"`
			} `json:"candidates"`
		}
		if err := json.Unmarshal([]byte(data), &resp); err != nil {
			continue
		}

		if len(resp.Candidates) > 0 && len(resp.Candidates[0].Content.Parts) > 0 {
			text := resp.Candidates[0].Content.Parts[0].Text
			if text != "" {
				return text, nil
			}
		}
	}

	if err := r.scanner.Err(); err != nil {
		return "", err
	}
	return "", io.EOF
}

func (r *googleStreamReader) Close() error {
	return r.body.Close()
}

// --- Ollama ---

type ollamaStreamClient struct {
	baseURL   string
	modelName string
}

func newOllamaStreamClient(config map[string]interface{}) (*ollamaStreamClient, error) {
	modelName, _ := config["model_name"].(string)
	if modelName == "" {
		return nil, fmt.Errorf("ollama requires model_name")
	}

	baseURL, _ := config["base_url"].(string)
	if baseURL == "" {
		baseURL = "http://localhost:11434"
	}

	return &ollamaStreamClient{
		baseURL:   baseURL,
		modelName: modelName,
	}, nil
}

func (c *ollamaStreamClient) StreamChatCompletion(messages []ChatMessage, opts StreamOptions) (StreamReader, error) {
	ollamaMessages := make([]map[string]string, len(messages))
	for i, msg := range messages {
		ollamaMessages[i] = map[string]string{
			"role":    msg.Role,
			"content": msg.Content,
		}
	}

	body := map[string]interface{}{
		"model":    c.modelName,
		"messages": ollamaMessages,
		"stream":   true,
	}

	ollamaOpts := map[string]interface{}{}
	if opts.Temperature != nil {
		ollamaOpts["temperature"] = *opts.Temperature
	}
	if opts.MaxTokens != nil {
		ollamaOpts["num_predict"] = *opts.MaxTokens
	}
	if len(ollamaOpts) > 0 {
		body["options"] = ollamaOpts
	}

	jsonBody, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	url := fmt.Sprintf("%s/api/chat", strings.TrimRight(c.baseURL, "/"))
	req, err := http.NewRequestWithContext(context.Background(), http.MethodPost, url, bytes.NewReader(jsonBody))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "unified-ui-agent-service/1.0")

	resp, err := streamHTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		_ = resp.Body.Close()
		return nil, fmt.Errorf("API returned status %d: %s", resp.StatusCode, string(respBody))
	}

	return &ollamaStreamReader{
		decoder: json.NewDecoder(resp.Body),
		body:    resp.Body,
	}, nil
}

func (c *ollamaStreamClient) Close() error { return nil }

type ollamaStreamReader struct {
	decoder *json.Decoder
	body    io.ReadCloser
}

func (r *ollamaStreamReader) Read() (string, error) {
	var chunk struct {
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
		Done bool `json:"done"`
	}

	if err := r.decoder.Decode(&chunk); err != nil {
		if err == io.EOF {
			return "", io.EOF
		}
		return "", err
	}

	if chunk.Done {
		return "", io.EOF
	}

	return chunk.Message.Content, nil
}

func (r *ollamaStreamReader) Close() error {
	return r.body.Close()
}
