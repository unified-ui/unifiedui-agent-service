// Package foundry provides Microsoft Foundry agent client implementations.
package foundry

import (
	"time"
)

// InputType represents the type of multimodal input content.
type InputType string

const (
	InputTypeText  InputType = "input_text"
	InputTypeImage InputType = "input_image"
	InputTypeFile  InputType = "input_file"
	InputTypeAudio InputType = "input_audio"
)

// InputContent represents a content item for multimodal Foundry requests.
type InputContent struct {
	Type InputType `json:"type"`

	// For input_text
	Text string `json:"text,omitempty"`

	// For input_image
	ImageURL string `json:"image_url,omitempty"`
	Detail   string `json:"detail,omitempty"`

	// For input_file
	FileData string `json:"file_data,omitempty"`
	Filename string `json:"filename,omitempty"`

	// For input_audio
	Data   string `json:"data,omitempty"`
	Format string `json:"format,omitempty"`
}

// InputMessage represents a message with multimodal content.
type InputMessage struct {
	Role    string         `json:"role"`
	Content []InputContent `json:"content"`
}

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

	// Input is the multimodal input content (nil for text-only messages)
	Input interface{}
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

// EventType represents the type of SSE event from Foundry.
type EventType string

const (
	EventResponseCreated    EventType = "response.created"
	EventResponseInProgress EventType = "response.in_progress"
	EventResponseCompleted  EventType = "response.completed"
	EventOutputItemAdded    EventType = "response.output_item.added"
	EventOutputItemDone     EventType = "response.output_item.done"
	EventContentPartAdded   EventType = "response.content_part.added"
	EventContentPartDone    EventType = "response.content_part.done"
	EventOutputTextDelta    EventType = "response.output_text.delta"
	EventOutputTextDone     EventType = "response.output_text.done"
)

// Event represents a parsed SSE event from Foundry.
type Event struct {
	Type           EventType    `json:"type"`
	SequenceNumber int          `json:"sequence_number"`
	Response       *Response    `json:"response,omitempty"`
	Item           *OutputItem  `json:"item,omitempty"`
	ItemID         string       `json:"item_id,omitempty"`
	OutputIndex    int          `json:"output_index,omitempty"`
	ContentIndex   int          `json:"content_index,omitempty"`
	Delta          string       `json:"delta,omitempty"`
	Text           string       `json:"text,omitempty"`
	Part           *ContentPart `json:"part,omitempty"`
}

// Response represents the response object in Foundry events.
type Response struct {
	ID           string                 `json:"id"`
	Object       string                 `json:"object"`
	Status       string                 `json:"status"`
	CreatedAt    int64                  `json:"created_at"`
	Model        string                 `json:"model,omitempty"`
	Output       []OutputItem           `json:"output,omitempty"`
	OutputText   string                 `json:"output_text,omitempty"`
	Usage        *Usage                 `json:"usage,omitempty"`
	Conversation *Conversation          `json:"conversation,omitempty"`
	Agent        *AgentRef              `json:"agent,omitempty"`
	Error        *Error                 `json:"error,omitempty"`
	Metadata     map[string]interface{} `json:"metadata,omitempty"`
}

// OutputItem represents an output item in Foundry response.
type OutputItem struct {
	Type      string        `json:"type"`
	ID        string        `json:"id"`
	Status    string        `json:"status,omitempty"`
	Role      string        `json:"role,omitempty"`
	Content   []ContentPart `json:"content,omitempty"`
	CreatedBy *CreatedBy    `json:"created_by,omitempty"`
	// Workflow action specific fields
	Kind             string `json:"kind,omitempty"`
	ActionID         string `json:"action_id,omitempty"`
	ParentActionID   string `json:"parent_action_id,omitempty"`
	PreviousActionID string `json:"previous_action_id,omitempty"`
}

// ContentPart represents a content part in Foundry output.
type ContentPart struct {
	Type        string   `json:"type"`
	Text        string   `json:"text,omitempty"`
	Annotations []string `json:"annotations,omitempty"`
}

// CreatedBy represents the creator of an output item.
type CreatedBy struct {
	Agent      *AgentRef `json:"agent,omitempty"`
	ResponseID string    `json:"response_id,omitempty"`
}

// AgentRef represents a reference to a Foundry agent.
type AgentRef struct {
	Type    string `json:"type"`
	Name    string `json:"name"`
	Version string `json:"version,omitempty"`
}

// Conversation represents a conversation reference.
type Conversation struct {
	ID string `json:"id"`
}

// Usage represents token usage information.
type Usage struct {
	InputTokens  int `json:"input_tokens"`
	OutputTokens int `json:"output_tokens"`
	TotalTokens  int `json:"total_tokens"`
}

// Error represents an error in Foundry response.
type Error struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

// RequestPayload represents the request payload to Foundry.
type RequestPayload struct {
	Agent        AgentPayload `json:"agent"`
	Conversation string       `json:"conversation,omitempty"` // Omit when empty to create new conversation
	Input        interface{}  `json:"input"`                  // string or []InputMessage for multimodal
	Stream       bool         `json:"stream"`
}

// AgentPayload represents the agent reference in a request.
type AgentPayload struct {
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
