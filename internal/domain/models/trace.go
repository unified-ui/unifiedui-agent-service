// Package models contains domain models for the UnifiedUI Chat Service.
package models

import "time"

// TraceStatus represents the status of a trace.
type TraceStatus string

const (
	// TraceStatusPending indicates the trace is pending.
	TraceStatusPending TraceStatus = "pending"
	// TraceStatusRunning indicates the trace is currently running.
	TraceStatusRunning TraceStatus = "running"
	// TraceStatusCompleted indicates the trace has completed successfully.
	TraceStatusCompleted TraceStatus = "completed"
	// TraceStatusFailed indicates the trace has failed.
	TraceStatusFailed TraceStatus = "failed"
)

// TraceType represents the type of trace.
type TraceType string

const (
	// TraceTypeLLM represents an LLM call trace.
	TraceTypeLLM TraceType = "llm"
	// TraceTypeTool represents a tool call trace.
	TraceTypeTool TraceType = "tool"
	// TraceTypeAgent represents an agent step trace.
	TraceTypeAgent TraceType = "agent"
	// TraceTypeChain represents a chain execution trace.
	TraceTypeChain TraceType = "chain"
	// TraceTypeRetriever represents a retriever call trace.
	TraceTypeRetriever TraceType = "retriever"
)

// Trace represents a trace entry for agent execution.
type Trace struct {
	ID             string        `json:"id" bson:"_id"`
	TenantID       string        `json:"tenantId" bson:"tenantId"`
	ConversationID string        `json:"conversationId" bson:"conversationId"`
	MessageID      string        `json:"messageId" bson:"messageId"`
	AgentID        string        `json:"agentId" bson:"agentId"`
	ParentTraceID  string        `json:"parentTraceId,omitempty" bson:"parentTraceId,omitempty"`
	Type           TraceType     `json:"type" bson:"type"`
	Name           string        `json:"name" bson:"name"`
	Status         TraceStatus   `json:"status" bson:"status"`
	Input          interface{}   `json:"input,omitempty" bson:"input,omitempty"`
	Output         interface{}   `json:"output,omitempty" bson:"output,omitempty"`
	Error          string        `json:"error,omitempty" bson:"error,omitempty"`
	StartedAt      time.Time     `json:"startedAt" bson:"startedAt"`
	EndedAt        *time.Time    `json:"endedAt,omitempty" bson:"endedAt,omitempty"`
	DurationMs     int64         `json:"durationMs,omitempty" bson:"durationMs,omitempty"`
	Metadata       TraceMetadata `json:"metadata,omitempty" bson:"metadata,omitempty"`
}

// TraceMetadata holds additional trace metadata.
type TraceMetadata struct {
	// Model is the model used (for LLM traces).
	Model string `json:"model,omitempty" bson:"model,omitempty"`
	// TokensInput is the number of input tokens.
	TokensInput int `json:"tokensInput,omitempty" bson:"tokensInput,omitempty"`
	// TokensOutput is the number of output tokens.
	TokensOutput int `json:"tokensOutput,omitempty" bson:"tokensOutput,omitempty"`
	// ToolName is the name of the tool (for tool traces).
	ToolName string `json:"toolName,omitempty" bson:"toolName,omitempty"`
	// ToolInput is the input to the tool.
	ToolInput interface{} `json:"toolInput,omitempty" bson:"toolInput,omitempty"`
	// ToolOutput is the output from the tool.
	ToolOutput interface{} `json:"toolOutput,omitempty" bson:"toolOutput,omitempty"`
	// Custom holds additional custom metadata.
	Custom map[string]interface{} `json:"custom,omitempty" bson:"custom,omitempty"`
}

// NewTrace creates a new trace with the given parameters.
func NewTrace(tenantID, conversationID, messageID, agentID, name string, traceType TraceType) *Trace {
	return &Trace{
		TenantID:       tenantID,
		ConversationID: conversationID,
		MessageID:      messageID,
		AgentID:        agentID,
		Type:           traceType,
		Name:           name,
		Status:         TraceStatusPending,
		StartedAt:      time.Now().UTC(),
	}
}

// Start marks the trace as running.
func (t *Trace) Start() {
	t.Status = TraceStatusRunning
	t.StartedAt = time.Now().UTC()
}

// Complete marks the trace as completed.
func (t *Trace) Complete(output interface{}) {
	t.Status = TraceStatusCompleted
	t.Output = output
	now := time.Now().UTC()
	t.EndedAt = &now
	t.DurationMs = now.Sub(t.StartedAt).Milliseconds()
}

// Fail marks the trace as failed.
func (t *Trace) Fail(err string) {
	t.Status = TraceStatusFailed
	t.Error = err
	now := time.Now().UTC()
	t.EndedAt = &now
	t.DurationMs = now.Sub(t.StartedAt).Milliseconds()
}
