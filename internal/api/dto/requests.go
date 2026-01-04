// Package dto provides Data Transfer Objects for API requests and responses.
package dto

import "time"

// SendMessageRequest represents the request body for sending a message.
type SendMessageRequest struct {
	Content string `json:"content" binding:"required,min=1,max=32000"`
	AgentID string `json:"agentId" binding:"required"`
	Stream  bool   `json:"stream"`
}

// MessageResponse represents a message in API responses.
type MessageResponse struct {
	ID             string            `json:"id"`
	ConversationID string            `json:"conversationId"`
	Role           string            `json:"role"`
	Content        string            `json:"content"`
	AgentID        string            `json:"agentId,omitempty"`
	UserID         string            `json:"userId,omitempty"`
	CreatedAt      time.Time         `json:"createdAt"`
	Metadata       *MetadataResponse `json:"metadata,omitempty"`
}

// MetadataResponse represents message metadata in API responses.
type MetadataResponse struct {
	Model        string                 `json:"model,omitempty"`
	TokensInput  int                    `json:"tokensInput,omitempty"`
	TokensOutput int                    `json:"tokensOutput,omitempty"`
	LatencyMs    int64                  `json:"latencyMs,omitempty"`
	AgentType    string                 `json:"agentType,omitempty"`
	Custom       map[string]interface{} `json:"custom,omitempty"`
}

// TraceResponse represents a trace in API responses.
type TraceResponse struct {
	ID            string                 `json:"id"`
	MessageID     string                 `json:"messageId"`
	AgentID       string                 `json:"agentId"`
	ParentTraceID string                 `json:"parentTraceId,omitempty"`
	Type          string                 `json:"type"`
	Name          string                 `json:"name"`
	Status        string                 `json:"status"`
	Input         interface{}            `json:"input,omitempty"`
	Output        interface{}            `json:"output,omitempty"`
	Error         string                 `json:"error,omitempty"`
	StartedAt     time.Time              `json:"startedAt"`
	EndedAt       *time.Time             `json:"endedAt,omitempty"`
	DurationMs    int64                  `json:"durationMs,omitempty"`
	Metadata      *TraceMetadataResponse `json:"metadata,omitempty"`
}

// TraceMetadataResponse represents trace metadata in API responses.
type TraceMetadataResponse struct {
	Model        string                 `json:"model,omitempty"`
	TokensInput  int                    `json:"tokensInput,omitempty"`
	TokensOutput int                    `json:"tokensOutput,omitempty"`
	ToolName     string                 `json:"toolName,omitempty"`
	ToolInput    interface{}            `json:"toolInput,omitempty"`
	ToolOutput   interface{}            `json:"toolOutput,omitempty"`
	Custom       map[string]interface{} `json:"custom,omitempty"`
}

// UpdateTraceRequest represents the request for updating a trace.
type UpdateTraceRequest struct {
	TraceID        string                 `json:"traceId" binding:"required"`
	ConversationID string                 `json:"conversationId" binding:"required"`
	MessageID      string                 `json:"messageId" binding:"required"`
	ParentTraceID  string                 `json:"parentTraceId,omitempty"`
	Type           string                 `json:"type" binding:"required"`
	Name           string                 `json:"name" binding:"required"`
	Status         string                 `json:"status" binding:"required"`
	Input          interface{}            `json:"input,omitempty"`
	Output         interface{}            `json:"output,omitempty"`
	Error          string                 `json:"error,omitempty"`
	StartedAt      *time.Time             `json:"startedAt,omitempty"`
	EndedAt        *time.Time             `json:"endedAt,omitempty"`
	DurationMs     int64                  `json:"durationMs,omitempty"`
	Metadata       *TraceMetadataResponse `json:"metadata,omitempty"`
}

// BatchUpdateTracesRequest represents the request for batch updating traces.
type BatchUpdateTracesRequest struct {
	Traces []*UpdateTraceRequest `json:"traces" binding:"required,min=1"`
}
