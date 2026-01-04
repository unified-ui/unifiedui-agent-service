// Package n8n provides N8N-specific agent client implementations.
package n8n

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

	"github.com/unifiedui/agent-service/internal/domain/models"
)

// ChunkType represents the type of stream chunk.
type ChunkType string

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
	defer reader.Close()

	var fullContent string
	var lastChunk *StreamChunk

	for {
		chunk, err := reader.Read()
		if err == io.EOF {
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
		defer reader.Close()

		for {
			chunk, err := reader.Read()
			if err == io.EOF {
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

	chatReq := &ChatRequest{
		ChatInput: chatInput,
		SessionID: req.SessionID,
	}

	body, err := json.Marshal(chatReq)
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
		resp.Body.Close()
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

		// Parse N8N stream event
		var event N8NStreamEvent
		if err := json.Unmarshal([]byte(line), &event); err != nil {
			// Skip non-JSON lines
			continue
		}

		// Only process "item" events with content from AI Agent node
		if event.Type != N8NStreamTypeItem {
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
				// This is metadata, skip it for now
				// Could be executionId, runInfo, etc.
				if execID, ok := innerData["executionId"].(string); ok {
					return &StreamChunk{
						Type:        ChunkTypeMetadata,
						ExecutionID: execID,
						Metadata:    make(map[string]interface{}),
					}, nil
				}
				// Skip other JSON metadata
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
