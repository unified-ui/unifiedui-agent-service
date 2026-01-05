// Package foundry provides Microsoft Foundry agent client implementations.
package foundry

import (
	"time"
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
	Type        ChunkType
	Content     string
	ExecutionID string
	Metadata    map[string]interface{}
	Error       error
}

// InvokeRequest represents a request to invoke a Foundry agent.
type InvokeRequest struct {
	// ExtConversationID is the external conversation ID from Foundry
	ExtConversationID string

	// Message is the user's message content
	Message string

	// AgentName is the name of the agent to invoke
	AgentName string
}

// InvokeResponse represents the response from a Foundry agent invocation.
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

// FoundryEventType represents the type of SSE event from Foundry.
type FoundryEventType string

const (
	EventResponseCreated    FoundryEventType = "response.created"
	EventResponseInProgress FoundryEventType = "response.in_progress"
	EventResponseCompleted  FoundryEventType = "response.completed"
	EventOutputItemAdded    FoundryEventType = "response.output_item.added"
	EventOutputItemDone     FoundryEventType = "response.output_item.done"
	EventContentPartAdded   FoundryEventType = "response.content_part.added"
	EventContentPartDone    FoundryEventType = "response.content_part.done"
	EventOutputTextDelta    FoundryEventType = "response.output_text.delta"
	EventOutputTextDone     FoundryEventType = "response.output_text.done"
)

// FoundryEvent represents a parsed SSE event from Foundry.
type FoundryEvent struct {
	Type           FoundryEventType    `json:"type"`
	SequenceNumber int                 `json:"sequence_number"`
	Response       *FoundryResponse    `json:"response,omitempty"`
	Item           *FoundryOutputItem  `json:"item,omitempty"`
	ItemID         string              `json:"item_id,omitempty"`
	OutputIndex    int                 `json:"output_index,omitempty"`
	ContentIndex   int                 `json:"content_index,omitempty"`
	Delta          string              `json:"delta,omitempty"`
	Text           string              `json:"text,omitempty"`
	Part           *FoundryContentPart `json:"part,omitempty"`
}

// FoundryResponse represents the response object in Foundry events.
type FoundryResponse struct {
	ID           string                 `json:"id"`
	Object       string                 `json:"object"`
	Status       string                 `json:"status"`
	CreatedAt    int64                  `json:"created_at"`
	Model        string                 `json:"model,omitempty"`
	Output       []FoundryOutputItem    `json:"output,omitempty"`
	OutputText   string                 `json:"output_text,omitempty"`
	Usage        *FoundryUsage          `json:"usage,omitempty"`
	Conversation *FoundryConversation   `json:"conversation,omitempty"`
	Agent        *FoundryAgentRef       `json:"agent,omitempty"`
	Error        *FoundryError          `json:"error,omitempty"`
	Metadata     map[string]interface{} `json:"metadata,omitempty"`
}

// FoundryOutputItem represents an output item in Foundry response.
type FoundryOutputItem struct {
	Type      string               `json:"type"`
	ID        string               `json:"id"`
	Status    string               `json:"status,omitempty"`
	Role      string               `json:"role,omitempty"`
	Content   []FoundryContentPart `json:"content,omitempty"`
	CreatedBy *FoundryCreatedBy    `json:"created_by,omitempty"`
	// Workflow action specific fields
	Kind             string `json:"kind,omitempty"`
	ActionID         string `json:"action_id,omitempty"`
	ParentActionID   string `json:"parent_action_id,omitempty"`
	PreviousActionID string `json:"previous_action_id,omitempty"`
}

// FoundryContentPart represents a content part in Foundry output.
type FoundryContentPart struct {
	Type        string   `json:"type"`
	Text        string   `json:"text,omitempty"`
	Annotations []string `json:"annotations,omitempty"`
}

// FoundryCreatedBy represents the creator of an output item.
type FoundryCreatedBy struct {
	Agent      *FoundryAgentRef `json:"agent,omitempty"`
	ResponseID string           `json:"response_id,omitempty"`
}

// FoundryAgentRef represents a reference to a Foundry agent.
type FoundryAgentRef struct {
	Type    string `json:"type"`
	Name    string `json:"name"`
	Version string `json:"version,omitempty"`
}

// FoundryConversation represents a conversation reference.
type FoundryConversation struct {
	ID string `json:"id"`
}

// FoundryUsage represents token usage information.
type FoundryUsage struct {
	InputTokens  int `json:"input_tokens"`
	OutputTokens int `json:"output_tokens"`
	TotalTokens  int `json:"total_tokens"`
}

// FoundryError represents an error in Foundry response.
type FoundryError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

// FoundryRequestPayload represents the request payload to Foundry.
type FoundryRequestPayload struct {
	Agent        FoundryAgentPayload `json:"agent"`
	Conversation string              `json:"conversation"`
	Input        string              `json:"input"`
	Stream       bool                `json:"stream"`
}

// FoundryAgentPayload represents the agent reference in a request.
type FoundryAgentPayload struct {
	Type string `json:"type"`
	Name string `json:"name"`
}

// WorkflowClientConfig holds the configuration for the Foundry workflow client.
type WorkflowClientConfig struct {
	ProjectEndpoint string
	APIVersion      string
	AgentName       string
	AgentType       string // "AGENT" or "MULTI_AGENT"
	APIToken        string // Bearer token from X-Microsoft-Foundry-API-Key header
}

// MessageInfo contains information about a parsed message from Foundry.
type MessageInfo struct {
	ID         string
	Role       string
	Content    string
	AgentName  string
	ResponseID string
	Status     string
	Kind       string // For workflow actions
	ActionID   string
	Metadata   map[string]interface{}
	CreatedAt  time.Time
}
