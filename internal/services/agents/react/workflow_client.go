package react

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
)

// WorkflowClientConfig holds the configuration for the ReACT workflow client.
type WorkflowClientConfig struct {
	BaseURL    string
	ServiceKey string
	HTTPClient *http.Client
}

// WorkflowClient communicates with the ReACT Agent Service via HTTP+SSE.
type WorkflowClient struct {
	baseURL    string
	serviceKey string
	httpClient *http.Client
}

// NewWorkflowClient creates a new ReACT workflow client.
func NewWorkflowClient(cfg *WorkflowClientConfig) *WorkflowClient {
	return &WorkflowClient{
		baseURL:    cfg.BaseURL,
		serviceKey: cfg.ServiceKey,
		httpClient: cfg.HTTPClient,
	}
}

// InvokeStreamReader sends a request to the ReACT service and returns a StreamReader.
func (c *WorkflowClient) InvokeStreamReader(ctx context.Context, req *InvokeRequest) (StreamReader, error) {
	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	url := strings.TrimRight(c.baseURL, "/") + "/api/v1/agent/invoke"

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("failed to create HTTP request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Accept", "text/event-stream")
	if c.serviceKey != "" {
		httpReq.Header.Set("X-Service-Key", c.serviceKey)
	}

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("failed to send request to ReACT service: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		defer func() { _ = resp.Body.Close() }()
		respBody, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("ReACT service returned status %d: %s", resp.StatusCode, string(respBody))
	}

	return &reactStreamReader{
		response: resp,
		scanner:  bufio.NewScanner(resp.Body),
	}, nil
}

// Invoke sends a message and returns the complete accumulated response.
func (c *WorkflowClient) Invoke(ctx context.Context, req *InvokeRequest) (string, error) {
	reader, err := c.InvokeStreamReader(ctx, req)
	if err != nil {
		return "", err
	}
	defer func() { _ = reader.Close() }()

	var fullContent strings.Builder

	for {
		chunk, err := reader.Read()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return fullContent.String(), err
		}

		if chunk.Type == ChunkTypeContent || chunk.Type == ChunkTypeReasoningStream ||
			chunk.Type == ChunkTypeToolCallStream || chunk.Type == ChunkTypePlanStream ||
			chunk.Type == ChunkTypeSubAgentStream || chunk.Type == ChunkTypeSynthesisStream {
			fullContent.WriteString(chunk.Content)
		}
	}

	return fullContent.String(), nil
}

// Close releases any resources held by the client.
func (c *WorkflowClient) Close() error {
	return nil
}

// reactStreamReader parses SSE events from the ReACT service response.
type reactStreamReader struct {
	response *http.Response
	scanner  *bufio.Scanner
}

// Read returns the next chunk from the SSE stream.
func (r *reactStreamReader) Read() (*StreamChunk, error) {
	for {
		dataLine, err := r.readNextDataLine()
		if err != nil {
			return nil, err
		}

		var msg StreamMessage
		if err := json.Unmarshal([]byte(dataLine), &msg); err != nil {
			continue
		}

		chunk := r.convertToChunk(&msg)
		if chunk != nil {
			return chunk, nil
		}
	}
}

// Close releases the HTTP response body.
func (r *reactStreamReader) Close() error {
	if r.response != nil && r.response.Body != nil {
		return r.response.Body.Close()
	}
	return nil
}

// readNextDataLine scans SSE lines until it finds a "data:" line.
func (r *reactStreamReader) readNextDataLine() (string, error) {
	for r.scanner.Scan() {
		line := r.scanner.Text()

		if strings.HasPrefix(line, "data:") {
			data := strings.TrimPrefix(line, "data:")
			data = strings.TrimSpace(data)
			if data == "" || data == "[DONE]" {
				continue
			}
			return data, nil
		}
	}

	if err := r.scanner.Err(); err != nil {
		return "", fmt.Errorf("scanner error: %w", err)
	}

	return "", io.EOF
}

// convertToChunk maps a StreamMessage to a StreamChunk based on the event type.
func (r *reactStreamReader) convertToChunk(msg *StreamMessage) *StreamChunk {
	switch msg.Type {
	case StreamTypeTextStream:
		return &StreamChunk{Type: ChunkTypeContent, Content: msg.Content, Config: msg.Config}
	case StreamTypeReasoningStart:
		return &StreamChunk{Type: ChunkTypeReasoningStart, Config: msg.Config}
	case StreamTypeReasoningStream:
		return &StreamChunk{Type: ChunkTypeReasoningStream, Content: msg.Content, Config: msg.Config}
	case StreamTypeReasoningEnd:
		return &StreamChunk{Type: ChunkTypeReasoningEnd, Config: msg.Config}
	case StreamTypeToolCallStart:
		return &StreamChunk{Type: ChunkTypeToolCallStart, Config: msg.Config}
	case StreamTypeToolCallStream:
		return &StreamChunk{Type: ChunkTypeToolCallStream, Content: msg.Content, Config: msg.Config}
	case StreamTypeToolCallEnd:
		return &StreamChunk{Type: ChunkTypeToolCallEnd, Config: msg.Config}
	case StreamTypePlanStart:
		return &StreamChunk{Type: ChunkTypePlanStart, Config: msg.Config}
	case StreamTypePlanStream:
		return &StreamChunk{Type: ChunkTypePlanStream, Content: msg.Content, Config: msg.Config}
	case StreamTypePlanComplete:
		return &StreamChunk{Type: ChunkTypePlanComplete, Config: msg.Config}
	case StreamTypeSubAgentStart:
		return &StreamChunk{Type: ChunkTypeSubAgentStart, Config: msg.Config}
	case StreamTypeSubAgentStream:
		return &StreamChunk{Type: ChunkTypeSubAgentStream, Content: msg.Content, Config: msg.Config}
	case StreamTypeSubAgentEnd:
		return &StreamChunk{Type: ChunkTypeSubAgentEnd, Config: msg.Config}
	case StreamTypeSynthesisStart:
		return &StreamChunk{Type: ChunkTypeSynthesisStart, Config: msg.Config}
	case StreamTypeSynthesisStream:
		return &StreamChunk{Type: ChunkTypeSynthesisStream, Content: msg.Content, Config: msg.Config}
	case StreamTypeTrace:
		return &StreamChunk{Type: ChunkTypeTrace, Config: msg.Config}
	case StreamTypeError:
		errMsg := "unknown error"
		if m, ok := msg.Config["message"]; ok {
			if s, ok := m.(string); ok {
				errMsg = s
			}
		}
		return &StreamChunk{Type: ChunkTypeError, Error: fmt.Errorf("%s", errMsg), Config: msg.Config}
	case StreamTypeStreamEnd:
		return &StreamChunk{Type: ChunkTypeDone, Config: msg.Config}
	default:
		return &StreamChunk{Type: ChunkTypeMetadata, Content: msg.Content, Config: msg.Config}
	}
}
