// Package n8n provides N8N-specific agent client implementations.
package n8n

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/unifiedui/agent-service/internal/domain/models"
)

// ChunkType represents the type of stream chunk.
type ChunkType string

// ChunkType constants define the types of stream chunks.
const (
	ChunkTypeContent  ChunkType = "content"
	ChunkTypeMetadata ChunkType = "metadata"
	ChunkTypeError    ChunkType = "error"
	ChunkTypeDone     ChunkType = "done"
)

// StreamChunk represents a chunk of streamed content.
type StreamChunk struct {
	Type        ChunkType
	Content     string
	ExecutionID string
	Metadata    map[string]interface{}
	Error       error
}

// InvokeRequest represents a request to invoke an agent.
type InvokeRequest struct {
	ConversationID string
	Message        string
	SessionID      string
	ChatHistory    []models.ChatHistoryEntry

	// Input is the pre-converted request body (nil for text-only messages)
	Input interface{}
}

// InvokeResponse represents the response from an agent invocation.
type InvokeResponse struct {
	Content     string
	ExecutionID string
	SessionID   string
	Metadata    map[string]interface{}
}

// StreamReader allows reading stream chunks one at a time.
type StreamReader interface {
	Read() (*StreamChunk, error)
	Close() error
}

// ChatWorkflowConfig holds the configuration for the chat workflow client.
type ChatWorkflowConfig struct {
	ChatURL               string
	Username              string
	Password              string
	HTTPClient            *http.Client
	UseUnifiedChatHistory bool
}

// ChatWorkflowClient implements the workflow client for N8N chat workflows.
type ChatWorkflowClient struct {
	chatURL               string
	username              string
	password              string
	httpClient            *http.Client
	useUnifiedChatHistory bool
}

// NewChatWorkflowClient creates a new N8N chat workflow client.
func NewChatWorkflowClient(config *ChatWorkflowConfig) (*ChatWorkflowClient, error) {
	if config == nil {
		return nil, fmt.Errorf("config is required")
	}
	if config.ChatURL == "" {
		return nil, fmt.Errorf("chat URL is required")
	}

	httpClient := config.HTTPClient
	if httpClient == nil {
		httpClient = &http.Client{
			Timeout: 5 * time.Minute, // Longer timeout for streaming
		}
	}

	return &ChatWorkflowClient{
		chatURL:               config.ChatURL,
		username:              config.Username,
		password:              config.Password,
		httpClient:            httpClient,
		useUnifiedChatHistory: config.UseUnifiedChatHistory,
	}, nil
}

// Invoke sends a message and returns the complete response (non-streaming).
func (c *ChatWorkflowClient) Invoke(ctx context.Context, req *InvokeRequest) (*InvokeResponse, error) {
	reader, err := c.InvokeStreamReader(ctx, req)
	if err != nil {
		return nil, err
	}
	defer func() { _ = reader.Close() }()

	var fullContent string
	var lastChunk *StreamChunk

	for {
		chunk, err := reader.Read()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("failed to read stream: %w", err)
		}

		if chunk.Type == ChunkTypeContent {
			fullContent += chunk.Content
		}
		lastChunk = chunk
	}

	response := &InvokeResponse{
		Content:   fullContent,
		SessionID: req.SessionID,
	}

	if lastChunk != nil {
		response.ExecutionID = lastChunk.ExecutionID
		response.Metadata = lastChunk.Metadata
	}

	return response, nil
}

// InvokeStream sends a message and streams the response through a channel.
func (c *ChatWorkflowClient) InvokeStream(ctx context.Context, req *InvokeRequest) (<-chan *StreamChunk, error) {
	reader, err := c.InvokeStreamReader(ctx, req)
	if err != nil {
		return nil, err
	}

	ch := make(chan *StreamChunk, 100)

	go func() {
		defer close(ch)
		defer func() { _ = reader.Close() }()

		for {
			chunk, err := reader.Read()
			if errors.Is(err, io.EOF) {
				return
			}
			if err != nil {
				ch <- &StreamChunk{
					Type:  ChunkTypeError,
					Error: err,
				}
				return
			}

			select {
			case ch <- chunk:
			case <-ctx.Done():
				return
			}
		}
	}()

	return ch, nil
}

// InvokeStreamReader sends a message and returns a reader for streaming response.
func (c *ChatWorkflowClient) InvokeStreamReader(ctx context.Context, req *InvokeRequest) (StreamReader, error) {
	// Build the chat input - use markdown if chat history is provided
	chatInput := req.Message
	now := time.Now()

	if c.useUnifiedChatHistory && len(req.ChatHistory) > 0 {
		chatInput = BuildSimpleChatHistoryMarkdown(req.ChatHistory, req.Message, now)
	}

	// Use pre-converted Input if provided (contains files), otherwise build ChatRequest
	var requestBody interface{}
	if req.Input != nil {
		// Use pre-converted request with files
		switch v := req.Input.(type) {
		case *ChatRequestWithFiles:
			// Update chatInput if chat history was applied
			v.ChatInput = chatInput
			v.SessionID = req.SessionID
			requestBody = v
		case *ChatRequest:
			v.ChatInput = chatInput
			v.SessionID = req.SessionID
			requestBody = v
		default:
			requestBody = req.Input
		}
	} else {
		requestBody = &ChatRequest{
			ChatInput: chatInput,
			SessionID: req.SessionID,
		}
	}

	body, err := json.Marshal(requestBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, c.chatURL, bytes.NewBuffer(body))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	c.setHeaders(httpReq)

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("failed to execute request: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		_ = resp.Body.Close()
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	return &streamReader{
		response: resp,
		scanner:  bufio.NewScanner(resp.Body),
	}, nil
}

// Close releases any resources held by the client.
func (c *ChatWorkflowClient) Close() error {
	return nil
}

// setHeaders sets the required headers for N8N chat requests.
func (c *ChatWorkflowClient) setHeaders(req *http.Request) {
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "text/event-stream")

	if c.username != "" && c.password != "" {
		req.SetBasicAuth(c.username, c.password)
	}
}

// streamReader implements the StreamReader interface.
type streamReader struct {
	response *http.Response
	scanner  *bufio.Scanner
}

// Read reads the next chunk from the stream.
func (r *streamReader) Read() (*StreamChunk, error) {
	for r.scanner.Scan() {
		line := r.scanner.Text()
		if line == "" {
			continue
		}

		// Strip SSE "data: " prefix if present
		parseLine := line
		if strings.HasPrefix(line, "data: ") {
			parseLine = strings.TrimPrefix(line, "data: ")
		}

		// Parse N8N stream event
		var event StreamEvent
		if err := json.Unmarshal([]byte(parseLine), &event); err != nil {
			// Skip non-JSON lines
			continue
		}

		// Handle non-streaming response format: {"output":"text"}
		if event.Type == "" {
			var plainResp map[string]interface{}
			if err := json.Unmarshal([]byte(parseLine), &plainResp); err == nil {
				if output, ok := plainResp["output"].(string); ok && output != "" {
					return &StreamChunk{
						Type:     ChunkTypeContent,
						Content:  output,
						Metadata: make(map[string]interface{}),
					}, nil
				}
				if text, ok := plainResp["text"].(string); ok && text != "" {
					return &StreamChunk{
						Type:     ChunkTypeContent,
						Content:  text,
						Metadata: make(map[string]interface{}),
					}, nil
				}
				if resp, ok := plainResp["response"].(string); ok && resp != "" {
					return &StreamChunk{
						Type:     ChunkTypeContent,
						Content:  resp,
						Metadata: make(map[string]interface{}),
					}, nil
				}
			}
			continue
		}

		// Only process "item" events with content from AI Agent node
		if event.Type != StreamTypeItem {
			continue
		}

		// Skip empty content
		if event.Content == "" {
			continue
		}

		// Check if content is JSON (metadata like executionId)
		if strings.HasPrefix(event.Content, "{") {
			var innerData map[string]interface{}
			if err := json.Unmarshal([]byte(event.Content), &innerData); err == nil {
				if execID, ok := innerData["executionId"].(string); ok {
					return &StreamChunk{
						Type:        ChunkTypeMetadata,
						ExecutionID: execID,
						Metadata:    make(map[string]interface{}),
					}, nil
				}
				continue
			}
		}

		// Regular text content
		return &StreamChunk{
			Type:     ChunkTypeContent,
			Content:  event.Content,
			Metadata: make(map[string]interface{}),
		}, nil
	}

	if err := r.scanner.Err(); err != nil {
		return nil, err
	}

	return nil, io.EOF
}

// Close closes the underlying response body.
func (r *streamReader) Close() error {
	if r.response != nil && r.response.Body != nil {
		return r.response.Body.Close()
	}
	return nil
}
