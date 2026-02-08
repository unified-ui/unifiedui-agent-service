// Package ai provides the AI service for LLM interactions.
package ai

import (
	"context"
	"fmt"
)

// ChatMessage represents a message in a chat completion request.
type ChatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// ChatCompletionResult represents the result of a chat completion.
type ChatCompletionResult struct {
	Content      string `json:"content"`
	Model        string `json:"model,omitempty"`
	LatencyMs    int64  `json:"latency_ms"`
	TokensInput  int    `json:"tokens_input,omitempty"`
	TokensOutput int    `json:"tokens_output,omitempty"`
}

// LLMClient defines the interface for LLM provider interactions.
type LLMClient interface {
	// ChatCompletion sends a chat completion request and returns the response.
	ChatCompletion(ctx context.Context, messages []ChatMessage) (*ChatCompletionResult, error)
}

// Service defines the interface for the AI service.
type Service interface {
	// GenerateTitle generates a concise conversation title from the first message exchange.
	GenerateTitle(ctx context.Context, tenantID, userMessage, assistantResponse string) (string, error)

	// GenerateDescription generates or improves a description for an entity.
	GenerateDescription(ctx context.Context, tenantID, entityType, entityName, existingDescription string, entityContext map[string]interface{}) (string, error)

	// AnalyzeTrace analyzes a failed trace node and returns error analysis.
	AnalyzeTrace(ctx context.Context, tenantID string, request AnalyzeTraceInput) (string, error)

	// SummarizeTrace summarizes trace nodes at the specified detail level.
	SummarizeTrace(ctx context.Context, tenantID string, request SummarizeTraceInput) (string, error)

	// TraceChat handles a conversational chat about a trace.
	TraceChat(ctx context.Context, tenantID string, request TraceChatInput) (string, error)

	// TestModel tests an LLM model configuration by sending a simple ping.
	TestModel(ctx context.Context, provider string, config map[string]interface{}, credentialSecret map[string]interface{}) (*TestModelResult, error)

	// GetCapabilities returns the available AI capabilities for a tenant.
	GetCapabilities(ctx context.Context, tenantID string) (*Capabilities, error)
}

// AnalyzeTraceInput holds the input for trace error analysis.
type AnalyzeTraceInput struct {
	TraceID  string                 `json:"trace_id"`
	NodeID   string                 `json:"node_id"`
	Error    string                 `json:"error"`
	NodeName string                 `json:"node_name"`
	NodeType string                 `json:"node_type"`
	Input    map[string]interface{} `json:"input"`
	Output   map[string]interface{} `json:"output"`
}

// SummarizeTraceInput holds the input for trace summarization.
type SummarizeTraceInput struct {
	TraceID     string                   `json:"trace_id"`
	DetailLevel string                   `json:"detail_level"`
	Nodes       []map[string]interface{} `json:"nodes"`
}

// TraceChatInput holds the input for trace chat.
type TraceChatInput struct {
	Trace        string        `json:"trace"`
	SelectedNode string        `json:"selected_node"`
	Message      string        `json:"message"`
	History      []ChatMessage `json:"history"`
}

// TestModelResult holds the result of a model test.
type TestModelResult struct {
	Success        bool   `json:"success"`
	Message        string `json:"message"`
	ResponseTimeMs int64  `json:"response_time_ms"`
}

// Capabilities represents the available AI features for a tenant.
type Capabilities struct {
	TitleGeneration       bool `json:"title_generation"`
	DescriptionGeneration bool `json:"description_generation"`
	TraceAnalysis         bool `json:"trace_analysis"`
	Summarization         bool `json:"summarization"`
}

// NewLLMClient creates a new LLM client based on the provider configuration.
func NewLLMClient(provider string, config map[string]interface{}, credentialSecret map[string]interface{}) (LLMClient, error) {
	apiKey := ""
	if credentialSecret != nil {
		if key, ok := credentialSecret["api_key"].(string); ok {
			apiKey = key
		}
	}

	switch provider {
	case "AZURE_OPENAI":
		return newAzureOpenAIClient(config, apiKey)
	case "OPENAI":
		return newOpenAIClient(config, apiKey)
	case "ANTHROPIC":
		return newAnthropicClient(config, apiKey)
	case "GOOGLE_GENAI":
		return newGoogleGenAIClient(config, apiKey)
	case "OLLAMA":
		return newOllamaClient(config)
	case "MISTRAL":
		return newMistralClient(config, apiKey)
	case "GROQ":
		return newGroqClient(config, apiKey)
	default:
		return nil, fmt.Errorf("unsupported provider: %s", provider)
	}
}
