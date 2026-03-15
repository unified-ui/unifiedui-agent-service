// Package restapi provides REST API agent client implementations.
package restapi

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
	"github.com/unifiedui/agent-service/internal/services/platform"
)

// ChunkType represents the type of stream chunk.
type ChunkType string

// ChunkType constants for REST API stream events.
const (
	ChunkTypeContent         ChunkType = "content"
	ChunkTypeError           ChunkType = "error"
	ChunkTypeDone            ChunkType = "done"
	ChunkTypeReasoningStart  ChunkType = "reasoning_start"
	ChunkTypeReasoningStream ChunkType = "reasoning_stream"
	ChunkTypeReasoningEnd    ChunkType = "reasoning_end"
	ChunkTypeToolCallStart   ChunkType = "tool_call_start"
	ChunkTypeToolCallStream  ChunkType = "tool_call_stream"
	ChunkTypeToolCallEnd     ChunkType = "tool_call_end"
	ChunkTypePlanStart       ChunkType = "plan_start"
	ChunkTypePlanStream      ChunkType = "plan_stream"
	ChunkTypePlanComplete    ChunkType = "plan_complete"
	ChunkTypeSubAgentStart   ChunkType = "sub_agent_start"
	ChunkTypeSubAgentStream  ChunkType = "sub_agent_stream"
	ChunkTypeSubAgentEnd     ChunkType = "sub_agent_end"
	ChunkTypeSynthesisStart  ChunkType = "synthesis_start"
	ChunkTypeSynthesisStream ChunkType = "synthesis_stream"
	ChunkTypeTrace           ChunkType = "trace"
)

// StreamChunk represents a parsed stream chunk from the external REST API agent.
type StreamChunk struct {
	Type    ChunkType
	Content string
	Config  map[string]interface{}
	Error   error
}

// WorkflowClientConfig holds configuration for the REST API workflow client.
type WorkflowClientConfig struct {
	InvokeEndpoint             string
	CreateConversationEndpoint string
	AuthType                   AuthType
	Credential                 *platform.Credentials
	AccessToken                string
	UserToken                  string
	APIKeyHeaderName           string
	UseUnifiedChatHistory      bool
	ChatHistoryCount           int
	HTTPClient                 *http.Client
}

// WorkflowClient implements the workflow client for REST API agents.
type WorkflowClient struct {
	invokeEndpoint             string
	createConversationEndpoint string
	authType                   AuthType
	credential                 *platform.Credentials
	accessToken                string
	userToken                  string
	apiKeyHeaderName           string
	useUnifiedChatHistory      bool
	chatHistoryCount           int
	httpClient                 *http.Client
}

// NewWorkflowClient creates a new REST API workflow client.
func NewWorkflowClient(config *WorkflowClientConfig) (*WorkflowClient, error) {
	if config == nil {
		return nil, fmt.Errorf("config is required")
	}
	if config.InvokeEndpoint == "" {
		return nil, fmt.Errorf("invoke endpoint is required")
	}

	httpClient := config.HTTPClient
	if httpClient == nil {
		httpClient = &http.Client{
			Timeout: 5 * time.Minute,
		}
	}

	return &WorkflowClient{
		invokeEndpoint:             config.InvokeEndpoint,
		createConversationEndpoint: config.CreateConversationEndpoint,
		authType:                   config.AuthType,
		credential:                 config.Credential,
		accessToken:                config.AccessToken,
		userToken:                  config.UserToken,
		apiKeyHeaderName:           config.APIKeyHeaderName,
		useUnifiedChatHistory:      config.UseUnifiedChatHistory,
		chatHistoryCount:           config.ChatHistoryCount,
		httpClient:                 httpClient,
	}, nil
}

// CreateConversation creates a new conversation/session on the external agent service.
func (c *WorkflowClient) CreateConversation(ctx context.Context) (string, error) {
	if c.createConversationEndpoint == "" {
		return "", fmt.Errorf("create conversation endpoint is not configured")
	}

	reqBody := &CreateConversationRequest{
		Config: map[string]interface{}{},
	}

	body, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("failed to marshal create conversation request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, c.createConversationEndpoint, bytes.NewBuffer(body))
	if err != nil {
		return "", fmt.Errorf("failed to create HTTP request: %w", err)
	}

	c.setHeaders(httpReq)

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return "", fmt.Errorf("failed to create conversation: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		errBody, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		return "", fmt.Errorf("conversation creation failed with status %d — %s", resp.StatusCode, strings.TrimSpace(string(errBody)))
	}

	var respBody CreateConversationResponse
	if err := json.NewDecoder(resp.Body).Decode(&respBody); err != nil {
		return "", fmt.Errorf("failed to decode create conversation response: %w", err)
	}

	return respBody.ConversationID, nil
}

// InvokeStreamReader sends a message to the external agent and returns a stream reader.
func (c *WorkflowClient) InvokeStreamReader(
	ctx context.Context,
	conversationID string,
	unifiedUIConversationID string,
	message string,
	chatHistory []models.ChatHistoryEntry,
) (*StreamReader, error) {
	var messageHistory []MessageHistoryEntry

	if c.useUnifiedChatHistory && len(chatHistory) > 0 {
		for _, entry := range chatHistory {
			messageHistory = append(messageHistory, MessageHistoryEntry{
				Role:    string(entry.Role),
				Content: entry.Content,
			})
		}
	}

	messageHistory = append(messageHistory, MessageHistoryEntry{
		Role:    "user",
		Content: message,
	})

	reqBody := &InvokeRequest{
		ConversationID:          conversationID,
		UnifiedUIConversationID: unifiedUIConversationID,
		MessageHistory:          messageHistory,
		Config:                  map[string]interface{}{},
	}

	body, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal invoke request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, c.invokeEndpoint, bytes.NewBuffer(body))
	if err != nil {
		return nil, fmt.Errorf("failed to create HTTP request: %w", err)
	}

	c.setHeaders(httpReq)
	httpReq.Header.Set("Accept", "text/event-stream")

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("failed to invoke agent: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		errBody, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		_ = resp.Body.Close()
		return nil, fmt.Errorf("unexpected status code from agent: %d — %s", resp.StatusCode, strings.TrimSpace(string(errBody)))
	}

	return &StreamReader{
		response: resp,
		scanner:  bufio.NewScanner(resp.Body),
	}, nil
}

// Close releases any resources held by the client.
func (c *WorkflowClient) Close() error {
	return nil
}

// setHeaders sets authentication headers based on the auth type.
func (c *WorkflowClient) setHeaders(req *http.Request) {
	req.Header.Set("Content-Type", "application/json")

	switch c.authType {
	case AuthTypeBasicAuth:
		if c.credential != nil {
			secret := c.credential.GetSecretAsBasicAuth()
			if secret != nil {
				req.SetBasicAuth(secret.Username, secret.Password)
			}
		}
	case AuthTypeAPIKey:
		if c.credential != nil {
			headerName := c.apiKeyHeaderName
			if headerName == "" {
				headerName = "X-API-Key"
			}
			req.Header.Set(headerName, c.credential.GetSecretAsString())
		}
	case AuthTypeEntraIDUserToken:
		if c.userToken != "" {
			req.Header.Set("Authorization", "Bearer "+c.userToken)
		}
	case AuthTypeEntraIDAppRegistration:
		if c.accessToken != "" {
			req.Header.Set("Authorization", "Bearer "+c.accessToken)
		}
	case AuthTypeAnonymous:
		// No auth headers
	}
}

// sseEventTypeToChunkType maps unified-ui SSE event types to chunk types.
var sseEventTypeToChunkType = map[string]ChunkType{
	"TEXT_STREAM":      ChunkTypeContent,
	"REASONING_START":  ChunkTypeReasoningStart,
	"REASONING_STREAM": ChunkTypeReasoningStream,
	"REASONING_END":    ChunkTypeReasoningEnd,
	"TOOL_CALL_START":  ChunkTypeToolCallStart,
	"TOOL_CALL_STREAM": ChunkTypeToolCallStream,
	"TOOL_CALL_END":    ChunkTypeToolCallEnd,
	"PLAN_START":       ChunkTypePlanStart,
	"PLAN_STREAM":      ChunkTypePlanStream,
	"PLAN_COMPLETE":    ChunkTypePlanComplete,
	"SUB_AGENT_START":  ChunkTypeSubAgentStart,
	"SUB_AGENT_STREAM": ChunkTypeSubAgentStream,
	"SUB_AGENT_END":    ChunkTypeSubAgentEnd,
	"SYNTHESIS_START":  ChunkTypeSynthesisStart,
	"SYNTHESIS_STREAM": ChunkTypeSynthesisStream,
	"TRACE":            ChunkTypeTrace,
	"ERROR":            ChunkTypeError,
}

// StreamReader implements reading SSE events from the external REST API agent.
type StreamReader struct {
	response  *http.Response
	scanner   *bufio.Scanner
	lastEvent string
}

// Read returns the next stream chunk from the SSE response.
func (r *StreamReader) Read() (*StreamChunk, error) {
	for r.scanner.Scan() {
		line := r.scanner.Text()

		if line == "" {
			continue
		}

		if strings.HasPrefix(line, "event:") {
			r.lastEvent = strings.TrimSpace(strings.TrimPrefix(line, "event:"))
			continue
		}

		if strings.HasPrefix(line, "data:") {
			dataStr := strings.TrimSpace(strings.TrimPrefix(line, "data:"))

			var eventData StreamEventData
			if err := json.Unmarshal([]byte(dataStr), &eventData); err != nil {
				continue
			}

			eventType := r.lastEvent
			if eventType == "" {
				eventType = eventData.Type
			}

			if eventType == "STREAM_START" || eventType == "STREAM_END" || eventType == "MESSAGE_COMPLETE" {
				if eventType == "STREAM_END" {
					return nil, io.EOF
				}
				continue
			}

			chunkType, ok := sseEventTypeToChunkType[eventType]
			if !ok {
				continue
			}

			chunk := &StreamChunk{
				Type:    chunkType,
				Content: eventData.Content,
				Config:  eventData.Config,
			}

			if chunkType == ChunkTypeError {
				chunk.Error = fmt.Errorf("agent error: %s", eventData.Content)
			}

			return chunk, nil
		}
	}

	if err := r.scanner.Err(); err != nil {
		return nil, err
	}

	return nil, io.EOF
}

// Close closes the underlying HTTP response body.
func (r *StreamReader) Close() error {
	if r.response != nil && r.response.Body != nil {
		return r.response.Body.Close()
	}
	return nil
}
