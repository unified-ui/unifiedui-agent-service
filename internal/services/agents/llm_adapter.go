package agents

import (
	"context"
	"errors"
	"fmt"
	"io"

	"github.com/unifiedui/agent-service/internal/services/agents/llm"
	"github.com/unifiedui/agent-service/internal/services/platform"
)

// llmWorkflowAdapter adapts an llm.StreamingClient to the WorkflowClient interface.
type llmWorkflowAdapter struct {
	client   llm.StreamingClient
	settings platform.AgentSettings
}

// Invoke is not supported for LLM streaming agents.
func (a *llmWorkflowAdapter) Invoke(_ context.Context, _ *InvokeRequest) (*InvokeResponse, error) {
	return nil, fmt.Errorf("LLM agent only supports streaming via InvokeStreamReader")
}

// InvokeStream is not supported — use InvokeStreamReader instead.
func (a *llmWorkflowAdapter) InvokeStream(_ context.Context, _ *InvokeRequest) (<-chan *StreamChunk, error) {
	return nil, fmt.Errorf("LLM agent only supports InvokeStreamReader")
}

// InvokeStreamReader sends a message to the LLM and returns a stream reader.
func (a *llmWorkflowAdapter) InvokeStreamReader(_ context.Context, req *InvokeRequest) (StreamReader, error) {
	messages := buildLLMMessages(a.settings, req)

	opts := llm.StreamOptions{}

	reader, err := a.client.StreamChatCompletion(messages, opts)
	if err != nil {
		return nil, fmt.Errorf("failed to start LLM stream: %w", err)
	}

	return &llmStreamReaderAdapter{reader: reader}, nil
}

// Close releases resources held by the LLM client.
func (a *llmWorkflowAdapter) Close() error {
	return a.client.Close()
}

func buildLLMMessages(settings platform.AgentSettings, req *InvokeRequest) []llm.ChatMessage {
	var messages []llm.ChatMessage

	if settings.SystemPrompt != "" {
		messages = append(messages, llm.ChatMessage{Role: "system", Content: settings.SystemPrompt})
	}

	for _, entry := range req.ChatHistory {
		role := string(entry.Role)
		if (role == "user" || role == "assistant") && entry.Content != "" {
			messages = append(messages, llm.ChatMessage{Role: role, Content: entry.Content})
		}
	}

	messages = append(messages, llm.ChatMessage{Role: "user", Content: req.Message})

	return messages
}

// llmStreamReaderAdapter adapts llm.StreamReader to agents.StreamReader.
type llmStreamReaderAdapter struct {
	reader llm.StreamReader
}

func (a *llmStreamReaderAdapter) Read() (*StreamChunk, error) {
	content, err := a.reader.Read()
	if err != nil {
		if errors.Is(err, io.EOF) {
			return nil, io.EOF
		}
		return &StreamChunk{
			Type:  ChunkTypeError,
			Error: err,
		}, nil
	}

	return &StreamChunk{
		Type:    ChunkTypeContent,
		Content: content,
	}, nil
}

func (a *llmStreamReaderAdapter) Close() error {
	return a.reader.Close()
}
