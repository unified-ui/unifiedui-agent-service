// Package agents provides the agent client interfaces and factory.
package agents

import (
	"context"

	"github.com/unifiedui/agent-service/internal/domain/models"
	"github.com/unifiedui/agent-service/internal/services/platform"
)

// ChunkType represents the type of stream chunk.
type ChunkType string

const (
	ChunkTypeContent    ChunkType = "content"
	ChunkTypeMetadata   ChunkType = "metadata"
	ChunkTypeError      ChunkType = "error"
	ChunkTypeDone       ChunkType = "done"
	ChunkTypeNewMessage ChunkType = "new_message"
)

// StreamChunk represents a chunk of streamed content.
type StreamChunk struct {
	// Type indicates the type of chunk (content, metadata, error, done)
	Type ChunkType

	// Content contains the text content (for content chunks)
	Content string

	// ExecutionID is the identifier for this execution (if available)
	ExecutionID string

	// Metadata contains additional information
	Metadata map[string]interface{}

	// Error contains error information (for error chunks)
	Error error
}

// InvokeRequest represents a request to invoke an agent.
type InvokeRequest struct {
	// ConversationID is the conversation identifier
	ConversationID string

	// Message is the user's message content
	Message string

	// SessionID is an optional session identifier for the agent
	SessionID string

	// ChatHistory contains the previous messages in the conversation
	// This is used when UseUnifiedChatHistory is enabled
	ChatHistory []models.ChatHistoryEntry
}

// InvokeResponse represents the response from an agent invocation.
type InvokeResponse struct {
	// Content is the full response content
	Content string

	// ExecutionID is the identifier for this execution
	ExecutionID string

	// SessionID is the session identifier
	SessionID string

	// Metadata contains any additional metadata
	Metadata map[string]interface{}
}

// ExecutionInfo represents information about an agent execution.
type ExecutionInfo struct {
	ID        string                 `json:"id"`
	Status    string                 `json:"status"`
	StartedAt string                 `json:"startedAt"`
	StoppedAt string                 `json:"stoppedAt,omitempty"`
	Data      map[string]interface{} `json:"data,omitempty"`
}

// StreamReader allows reading stream chunks one at a time.
type StreamReader interface {
	// Read returns the next chunk from the stream.
	// Returns io.EOF when the stream is exhausted.
	Read() (*StreamChunk, error)

	// Close releases resources associated with the reader.
	Close() error
}

// WorkflowClient defines the interface for invoking agent workflows.
type WorkflowClient interface {
	// Invoke sends a message to the agent and returns the complete response.
	Invoke(ctx context.Context, req *InvokeRequest) (*InvokeResponse, error)

	// InvokeStream sends a message and streams the response through a channel.
	InvokeStream(ctx context.Context, req *InvokeRequest) (<-chan *StreamChunk, error)

	// InvokeStreamReader sends a message and returns a reader for the stream.
	InvokeStreamReader(ctx context.Context, req *InvokeRequest) (StreamReader, error)

	// Close releases any resources held by the client.
	Close() error
}

// APIClient defines the interface for agent management API operations.
type APIClient interface {
	// GetExecution retrieves execution details by ID.
	GetExecution(ctx context.Context, executionID string) (*ExecutionInfo, error)

	// GetExecutionsBySession retrieves executions for a session.
	GetExecutionsBySession(ctx context.Context, sessionID string) ([]*ExecutionInfo, error)

	// Close releases any resources held by the client.
	Close() error
}

// AgentClients holds the clients needed for agent operations.
type AgentClients struct {
	// WorkflowClient is used to invoke agent workflows
	WorkflowClient WorkflowClient

	// APIClient is used for agent management operations
	APIClient APIClient

	// Config is the original configuration
	Config *platform.AgentConfig
}

// Close releases all client resources.
func (c *AgentClients) Close() error {
	var errs []error

	if c.WorkflowClient != nil {
		if err := c.WorkflowClient.Close(); err != nil {
			errs = append(errs, err)
		}
	}

	if c.APIClient != nil {
		if err := c.APIClient.Close(); err != nil {
			errs = append(errs, err)
		}
	}

	if len(errs) > 0 {
		return errs[0]
	}
	return nil
}
