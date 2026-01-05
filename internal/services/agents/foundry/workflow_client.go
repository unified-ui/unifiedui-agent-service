// Package foundry provides Microsoft Foundry agent client implementations.
package foundry

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

// WorkflowClient implements the workflow client for Microsoft Foundry agents.
type WorkflowClient struct {
	projectEndpoint string
	apiVersion      string
	agentName       string
	agentType       string
	apiToken        string
	httpClient      *http.Client
}

// NewWorkflowClient creates a new Microsoft Foundry workflow client.
func NewWorkflowClient(config *WorkflowClientConfig) (*WorkflowClient, error) {
	if config == nil {
		return nil, fmt.Errorf("config is required")
	}
	if config.ProjectEndpoint == "" {
		return nil, fmt.Errorf("project endpoint is required")
	}
	if config.AgentName == "" {
		return nil, fmt.Errorf("agent name is required")
	}
	if config.APIToken == "" {
		return nil, fmt.Errorf("API token is required")
	}

	apiVersion := config.APIVersion
	if apiVersion == "" {
		apiVersion = "2025-11-15-preview"
	}

	return &WorkflowClient{
		projectEndpoint: strings.TrimSuffix(config.ProjectEndpoint, "/"),
		apiVersion:      apiVersion,
		agentName:       config.AgentName,
		agentType:       config.AgentType,
		apiToken:        config.APIToken,
		httpClient: &http.Client{
			Timeout: 10 * time.Minute, // Long timeout for streaming
		},
	}, nil
}

// Invoke sends a message and returns the complete response (non-streaming).
func (c *WorkflowClient) Invoke(ctx context.Context, req *InvokeRequest) (*InvokeResponse, error) {
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
		SessionID: req.ExtConversationID,
	}

	if lastChunk != nil {
		response.ExecutionID = lastChunk.ExecutionID
		response.Metadata = lastChunk.Metadata
	}

	return response, nil
}

// InvokeStream sends a message and streams the response through a channel.
func (c *WorkflowClient) InvokeStream(ctx context.Context, req *InvokeRequest) (<-chan *StreamChunk, error) {
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
func (c *WorkflowClient) InvokeStreamReader(ctx context.Context, req *InvokeRequest) (StreamReader, error) {
	url := fmt.Sprintf("%s/openai/responses?api-version=%s", c.projectEndpoint, c.apiVersion)

	payload := &FoundryRequestPayload{
		Agent: FoundryAgentPayload{
			Type: "agent_reference",
			Name: c.agentName,
		},
		Conversation: req.ExtConversationID,
		Input:        req.Message,
		Stream:       true,
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewBuffer(body))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+c.apiToken)
	httpReq.Header.Set("Accept", "text/event-stream")

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		defer resp.Body.Close()
		bodyBytes, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("foundry API error: status=%d, body=%s", resp.StatusCode, string(bodyBytes))
	}

	return &foundryStreamReader{
		response:  resp,
		scanner:   bufio.NewScanner(resp.Body),
		messages:  make([]*MessageInfo, 0),
		agentType: c.agentType,
	}, nil
}

// Close releases any resources held by the client.
func (c *WorkflowClient) Close() error {
	return nil
}

// foundryStreamReader implements StreamReader for Foundry SSE responses.
type foundryStreamReader struct {
	response       *http.Response
	scanner        *bufio.Scanner
	closed         bool
	messages       []*MessageInfo
	currentContent strings.Builder
	agentType      string
	lastEvent      *FoundryEvent
	lastMessageID  string
}

// Read returns the next chunk from the stream.
func (r *foundryStreamReader) Read() (*StreamChunk, error) {
	if r.closed {
		return nil, io.EOF
	}

	for r.scanner.Scan() {
		line := r.scanner.Text()

		// Skip empty lines
		if line == "" {
			continue
		}

		// Parse SSE event
		if strings.HasPrefix(line, "event: ") {
			// Store event type for next data line
			continue
		}

		// Handle data lines - can be "data: {...}" or just "{...}"
		var jsonData string
		if strings.HasPrefix(line, "data: ") {
			jsonData = strings.TrimPrefix(line, "data: ")
		} else if strings.HasPrefix(line, "{") {
			jsonData = line
		} else {
			continue
		}

		// Check for stream end
		if jsonData == "[DONE]" {
			return nil, io.EOF
		}

		// Parse the JSON event
		var event FoundryEvent
		if err := json.Unmarshal([]byte(jsonData), &event); err != nil {
			// Skip malformed events
			continue
		}

		// Process the event and potentially return a chunk
		chunk := r.processEvent(&event)
		if chunk != nil {
			return chunk, nil
		}
	}

	if err := r.scanner.Err(); err != nil {
		return nil, fmt.Errorf("scanner error: %w", err)
	}

	return nil, io.EOF
}

// processEvent processes a Foundry SSE event and returns a StreamChunk if applicable.
func (r *foundryStreamReader) processEvent(event *FoundryEvent) *StreamChunk {
	r.lastEvent = event

	switch event.Type {
	case EventOutputTextDelta:
		// Text delta - main content streaming
		if event.Delta != "" {
			return &StreamChunk{
				Type:    ChunkTypeContent,
				Content: event.Delta,
			}
		}

	case EventOutputItemAdded:
		// New output item added - check if it's a new message
		if event.Item != nil {
			// Check if this is a new message (different from last)
			if event.Item.Type == "message" && event.Item.ID != r.lastMessageID {
				// If we had a previous message, signal new message
				if r.lastMessageID != "" {
					r.lastMessageID = event.Item.ID
					return &StreamChunk{
						Type:    ChunkTypeNewMessage,
						Content: "",
						Metadata: map[string]interface{}{
							"message_id": event.Item.ID,
							"role":       event.Item.Role,
						},
					}
				}
				r.lastMessageID = event.Item.ID
			}

			// Handle workflow actions
			if event.Item.Type == "workflow_action" {
				return &StreamChunk{
					Type: ChunkTypeMetadata,
					Metadata: map[string]interface{}{
						"type":               "workflow_action",
						"id":                 event.Item.ID,
						"kind":               event.Item.Kind,
						"action_id":          event.Item.ActionID,
						"parent_action_id":   event.Item.ParentActionID,
						"previous_action_id": event.Item.PreviousActionID,
						"status":             event.Item.Status,
					},
				}
			}
		}

	case EventOutputItemDone:
		// Output item completed
		if event.Item != nil && event.Item.Type == "message" {
			// Extract agent info
			agentName := ""
			responseID := ""
			if event.Item.CreatedBy != nil {
				if event.Item.CreatedBy.Agent != nil {
					agentName = event.Item.CreatedBy.Agent.Name
				}
				responseID = event.Item.CreatedBy.ResponseID
			}

			// Build full content from content parts
			var content string
			for _, part := range event.Item.Content {
				if part.Type == "output_text" {
					content += part.Text
				}
			}

			// Store message info
			msgInfo := &MessageInfo{
				ID:         event.Item.ID,
				Role:       event.Item.Role,
				Content:    content,
				AgentName:  agentName,
				ResponseID: responseID,
				Status:     event.Item.Status,
				CreatedAt:  time.Now(),
				Metadata: map[string]interface{}{
					"output_index": event.OutputIndex,
				},
			}
			r.messages = append(r.messages, msgInfo)

			return &StreamChunk{
				Type: ChunkTypeMetadata,
				Metadata: map[string]interface{}{
					"type":        "message_done",
					"message_id":  event.Item.ID,
					"role":        event.Item.Role,
					"status":      event.Item.Status,
					"agent_name":  agentName,
					"response_id": responseID,
				},
			}
		}

	case EventResponseCompleted:
		// Response completed - send final metadata
		if event.Response != nil {
			metadata := map[string]interface{}{
				"type":        "response_completed",
				"response_id": event.Response.ID,
				"status":      event.Response.Status,
			}

			if event.Response.Usage != nil {
				metadata["usage"] = map[string]interface{}{
					"input_tokens":  event.Response.Usage.InputTokens,
					"output_tokens": event.Response.Usage.OutputTokens,
					"total_tokens":  event.Response.Usage.TotalTokens,
				}
			}

			if event.Response.Agent != nil {
				metadata["agent_name"] = event.Response.Agent.Name
			}

			if event.Response.Conversation != nil {
				metadata["conversation_id"] = event.Response.Conversation.ID
			}

			return &StreamChunk{
				Type:        ChunkTypeDone,
				ExecutionID: event.Response.ID,
				Metadata:    metadata,
			}
		}
	}

	return nil
}

// GetMessages returns all parsed messages from the stream.
func (r *foundryStreamReader) GetMessages() []*MessageInfo {
	return r.messages
}

// Close releases resources associated with the reader.
func (r *foundryStreamReader) Close() error {
	if r.closed {
		return nil
	}
	r.closed = true
	if r.response != nil && r.response.Body != nil {
		return r.response.Body.Close()
	}
	return nil
}
