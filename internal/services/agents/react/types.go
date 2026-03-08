// Package react provides the ReACT agent service client for streaming AI agent invocations.
package react

import (
	"github.com/unifiedui/agent-service/internal/domain/models"
)

// StreamMessageType represents the type of stream message from the ReACT service.
type StreamMessageType string

// StreamMessageType constants matching the 22 SDK SSE event types.
const (
	StreamTypeStreamStart     StreamMessageType = "STREAM_START"
	StreamTypeTextStream      StreamMessageType = "TEXT_STREAM"
	StreamTypeStreamNewMsg    StreamMessageType = "STREAM_NEW_MESSAGE"
	StreamTypeStreamEnd       StreamMessageType = "STREAM_END"
	StreamTypeMessageComplete StreamMessageType = "MESSAGE_COMPLETE"
	StreamTypeTitleGeneration StreamMessageType = "TITLE_GENERATION"
	StreamTypeError           StreamMessageType = "ERROR"
	StreamTypeReasoningStart  StreamMessageType = "REASONING_START"
	StreamTypeReasoningStream StreamMessageType = "REASONING_STREAM"
	StreamTypeReasoningEnd    StreamMessageType = "REASONING_END"
	StreamTypeToolCallStart   StreamMessageType = "TOOL_CALL_START"
	StreamTypeToolCallStream  StreamMessageType = "TOOL_CALL_STREAM"
	StreamTypeToolCallEnd     StreamMessageType = "TOOL_CALL_END"
	StreamTypePlanStart       StreamMessageType = "PLAN_START"
	StreamTypePlanStream      StreamMessageType = "PLAN_STREAM"
	StreamTypePlanComplete    StreamMessageType = "PLAN_COMPLETE"
	StreamTypeSubAgentStart   StreamMessageType = "SUB_AGENT_START"
	StreamTypeSubAgentStream  StreamMessageType = "SUB_AGENT_STREAM"
	StreamTypeSubAgentEnd     StreamMessageType = "SUB_AGENT_END"
	StreamTypeSynthesisStart  StreamMessageType = "SYNTHESIS_START"
	StreamTypeSynthesisStream StreamMessageType = "SYNTHESIS_STREAM"
	StreamTypeTrace           StreamMessageType = "TRACE"
)

// ChunkType represents the type of chunk from the ReACT service.
type ChunkType string

// ChunkType constants.
const (
	ChunkTypeContent         ChunkType = "content"
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
	ChunkTypeMetadata        ChunkType = "metadata"
	ChunkTypeError           ChunkType = "error"
	ChunkTypeDone            ChunkType = "done"
)

// StreamChunk represents a chunk of streamed content from the ReACT service.
type StreamChunk struct {
	Type    ChunkType
	Content string
	Config  map[string]interface{}
	Error   error
}

// StreamReader provides sequential access to ReACT stream chunks.
type StreamReader interface {
	Read() (*StreamChunk, error)
	Close() error
}

// StreamMessage represents the JSON payload of each SSE data line from the ReACT service.
type StreamMessage struct {
	Type    StreamMessageType      `json:"type"`
	Content string                 `json:"content,omitempty"`
	Config  map[string]interface{} `json:"config,omitempty"`
}

// InvokeRequest represents the payload sent to the ReACT agent service.
type InvokeRequest struct {
	TenantID       string                    `json:"tenant_id"`
	ChatAgentID    string                    `json:"chat_agent_id"`
	ConversationID string                    `json:"conversation_id"`
	Message        string                    `json:"message"`
	History        []models.ChatHistoryEntry `json:"history"`
	AgentConfig    AgentConfigPayload        `json:"agent_config"`
}

// AgentConfigPayload is the agent_config sent to the ReACT service.
type AgentConfigPayload struct {
	ReactAgentID      string           `json:"react_agent_id"`
	Version           string           `json:"version,omitempty"`
	Prompts           PromptsConfig    `json:"prompts"`
	AIModels          []AIModelConfig  `json:"ai_models"`
	Tools             []ToolDefinition `json:"tools"`
	MultiAgentEnabled bool             `json:"multi_agent_enabled"`
}

// PromptsConfig holds prompt configuration.
type PromptsConfig struct {
	SystemPrompt string `json:"system_prompt"`
}

// AIModelConfig holds AI model configuration for the ReACT service.
type AIModelConfig struct {
	Provider    string                 `json:"provider"`
	ModelName   string                 `json:"model_name"`
	APIKey      string                 `json:"api_key"`
	BaseURL     string                 `json:"base_url,omitempty"`
	APIVersion  string                 `json:"api_version,omitempty"`
	ExtraConfig map[string]interface{} `json:"extra_config,omitempty"`
}

// ToolDefinition holds a tool definition for the ReACT service.
type ToolDefinition struct {
	Name        string           `json:"name"`
	Description string           `json:"description"`
	Type        string           `json:"type"`
	Config      interface{}      `json:"config"`
	Credentials []ToolCredential `json:"credentials,omitempty"`
}

// ToolCredential holds credential information for a tool.
type ToolCredential struct {
	Type  string `json:"type"`
	Value string `json:"value"`
}
