// Package foundry provides Microsoft Foundry agent client implementations.
package foundry

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

// SetHTTPClient sets a custom HTTP client (used for testing).
func (c *WorkflowClient) SetHTTPClient(client *http.Client) {
	c.httpClient = client
}

// TestConnection verifies that the Foundry endpoint and API token are valid
// by sending a ping message to the agent. A 400 response about payload format
// still proves connectivity, authentication, and agent existence.
func (c *WorkflowClient) TestConnection(ctx context.Context) error {
	url := fmt.Sprintf("%s/openai/responses?api-version=%s", c.projectEndpoint, c.apiVersion)

	payload := &RequestPayload{
		Agent: AgentPayload{
			Type: "agent_reference",
			Name: c.agentName,
		},
		Input:  "ping",
		Stream: false,
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewBuffer(body))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+c.apiToken)

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return fmt.Errorf("failed to send request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	// 200 = full success, 400 = payload rejected but endpoint+auth+agent are valid
	if resp.StatusCode == http.StatusOK || resp.StatusCode == http.StatusBadRequest {
		return nil
	}

	bodyBytes, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
	return fmt.Errorf("foundry API error: status=%d, body=%s", resp.StatusCode, string(bodyBytes))
}

// Invoke sends a message and returns the complete response (non-streaming).
func (c *WorkflowClient) Invoke(ctx context.Context, req *InvokeRequest) (*InvokeResponse, error) {
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
func (c *WorkflowClient) InvokeStreamReader(ctx context.Context, req *InvokeRequest) (StreamReader, error) {
	url := fmt.Sprintf("%s/openai/responses?api-version=%s", c.projectEndpoint, c.apiVersion)

	// Use multimodal Input if provided, otherwise use plain text Message
	var input interface{}
	if req.Input != nil {
		input = req.Input
	} else {
		input = req.Message
	}

	var conversationPtr *string
	if req.ExtConversationID != "" {
		conversationPtr = &req.ExtConversationID
	}

	payload := &RequestPayload{
		Agent: AgentPayload{
			Type: "agent_reference",
			Name: c.agentName,
		},
		Conversation: conversationPtr,
		Input:        input,
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
		defer func() { _ = resp.Body.Close() }()
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
	response      *http.Response
	scanner       *bufio.Scanner
	closed        bool
	messages      []*MessageInfo
	agentType     string
	lastEvent     *Event
	lastMessageID string
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
		switch {
		case strings.HasPrefix(line, "data: "):
			jsonData = strings.TrimPrefix(line, "data: ")
		case strings.HasPrefix(line, "{"):
			jsonData = line
		default:
			continue
		}

		// Check for stream end
		if jsonData == "[DONE]" {
			return nil, io.EOF
		}

		// Parse the JSON event
		var event Event
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

// isToolCall checks if the item type matches the *_call pattern (e.g. openapi_call, mcp_call).
func isToolCall(itemType string) bool {
	return strings.HasSuffix(itemType, "_call")
}

// isToolCallOutput checks if the item type matches the *_call_output pattern.
func isToolCallOutput(itemType string) bool {
	return strings.HasSuffix(itemType, "_call_output")
}

// processEvent processes a Foundry SSE event and returns a StreamChunk if applicable.
func (r *foundryStreamReader) processEvent(event *Event) *StreamChunk {
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
		if event.Item != nil {
			return r.handleOutputItemAdded(event)
		}

	case EventOutputItemDone:
		if event.Item != nil {
			return r.handleOutputItemDone(event)
		}

	case EventResponseCompleted:
		// Response completed - send final metadata
		if event.Response != nil {
			return r.handleResponseCompleted(event)
		}
	}

	return nil
}

// handleOutputItemAdded processes output_item.added events.
func (r *foundryStreamReader) handleOutputItemAdded(event *Event) *StreamChunk {
	item := event.Item

	// Generic *_call_output: skip on added (wait for done)
	if isToolCallOutput(item.Type) {
		return nil
	}

	// Generic *_call: emit tool_call_start
	if isToolCall(item.Type) {
		return &StreamChunk{
			Type: ChunkTypeToolCallStart,
			Config: map[string]interface{}{
				"tool_call_id": item.CallID,
				"tool_name":    item.Name,
				"call_type":    item.Type,
			},
		}
	}

	// Workflow actions
	if item.Type == "workflow_action" {
		return r.handleWorkflowActionAdded(item)
	}

	// New message
	if item.Type == "message" && item.ID != r.lastMessageID {
		if r.lastMessageID != "" {
			r.lastMessageID = item.ID
			return &StreamChunk{
				Type:    ChunkTypeNewMessage,
				Content: "",
				Metadata: map[string]interface{}{
					"message_id": item.ID,
					"role":       item.Role,
				},
			}
		}
		r.lastMessageID = item.ID
	}

	return nil
}

// handleWorkflowActionAdded processes workflow_action items on added.
func (r *foundryStreamReader) handleWorkflowActionAdded(item *OutputItem) *StreamChunk {
	switch item.Kind {
	case "InvokeAzureAgent":
		agentName := ""
		if item.CreatedBy != nil && item.CreatedBy.Agent != nil {
			agentName = item.CreatedBy.Agent.Name
		}
		return &StreamChunk{
			Type: ChunkTypeSubAgentStart,
			Config: map[string]interface{}{
				"agent_name": agentName,
				"action_id":  item.ActionID,
			},
		}
	case "EndConversation":
		return nil
	default:
		return &StreamChunk{
			Type: ChunkTypeToolCallStart,
			Config: map[string]interface{}{
				"tool_name": item.Kind,
				"call_type": "workflow_action",
				"action_id": item.ActionID,
			},
		}
	}
}

// handleOutputItemDone processes output_item.done events.
func (r *foundryStreamReader) handleOutputItemDone(event *Event) *StreamChunk {
	item := event.Item

	// Generic *_call_output done: emit tool_call_end with result
	if isToolCallOutput(item.Type) {
		callType := strings.TrimSuffix(item.Type, "_output")
		return &StreamChunk{
			Type: ChunkTypeToolCallEnd,
			Config: map[string]interface{}{
				"tool_call_id": item.CallID,
				"tool_name":    item.Name,
				"tool_result":  item.Output,
				"call_type":    callType,
			},
		}
	}

	// Generic *_call done: emit tool_call_stream with arguments
	if isToolCall(item.Type) {
		return &StreamChunk{
			Type:    ChunkTypeToolCallStream,
			Content: item.Arguments,
		}
	}

	// Workflow action done
	if item.Type == "workflow_action" {
		return r.handleWorkflowActionDone(item)
	}

	// Message done
	if item.Type == "message" {
		return r.handleMessageDone(event)
	}

	return nil
}

// handleWorkflowActionDone processes workflow_action items on done.
func (r *foundryStreamReader) handleWorkflowActionDone(item *OutputItem) *StreamChunk {
	switch item.Kind {
	case "InvokeAzureAgent":
		return &StreamChunk{
			Type:   ChunkTypeSubAgentEnd,
			Config: map[string]interface{}{},
		}
	case "EndConversation":
		return nil
	default:
		return &StreamChunk{
			Type:   ChunkTypeToolCallEnd,
			Config: map[string]interface{}{},
		}
	}
}

// handleMessageDone processes message items on done.
func (r *foundryStreamReader) handleMessageDone(event *Event) *StreamChunk {
	item := event.Item
	agentName := ""
	responseID := ""
	if item.CreatedBy != nil {
		if item.CreatedBy.Agent != nil {
			agentName = item.CreatedBy.Agent.Name
		}
		responseID = item.CreatedBy.ResponseID
	}

	var content string
	for _, part := range item.Content {
		if part.Type == "output_text" {
			content += part.Text
		}
	}

	msgInfo := &MessageInfo{
		ID:         item.ID,
		Role:       item.Role,
		Content:    content,
		AgentName:  agentName,
		ResponseID: responseID,
		Status:     item.Status,
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
			"message_id":  item.ID,
			"role":        item.Role,
			"status":      item.Status,
			"agent_name":  agentName,
			"response_id": responseID,
		},
	}
}

// handleResponseCompleted processes response.completed events.
func (r *foundryStreamReader) handleResponseCompleted(event *Event) *StreamChunk {
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
