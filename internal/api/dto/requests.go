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

// Note: Trace-related DTOs are defined in traces.go
