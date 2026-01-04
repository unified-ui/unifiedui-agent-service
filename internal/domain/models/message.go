// Package models contains domain models for the UnifiedUI Chat Service.
package models

import "time"

// MessageRole represents the role of a message sender.
type MessageRole string

const (
	// RoleUser represents a message from the user.
	RoleUser MessageRole = "user"
	// RoleAssistant represents a message from the assistant.
	RoleAssistant MessageRole = "assistant"
	// RoleSystem represents a system message.
	RoleSystem MessageRole = "system"
)

// Message represents a chat message in a conversation.
type Message struct {
	ID             string      `json:"id" bson:"_id"`
	TenantID       string      `json:"tenantId" bson:"tenantId"`
	ConversationID string      `json:"conversationId" bson:"conversationId"`
	Role           MessageRole `json:"role" bson:"role"`
	Content        string      `json:"content" bson:"content"`
	AgentID        string      `json:"agentId,omitempty" bson:"agentId,omitempty"`
	UserID         string      `json:"userId,omitempty" bson:"userId,omitempty"`
	CreatedAt      time.Time   `json:"createdAt" bson:"createdAt"`
	UpdatedAt      time.Time   `json:"updatedAt" bson:"updatedAt"`
	Metadata       Metadata    `json:"metadata,omitempty" bson:"metadata,omitempty"`
}

// Metadata holds additional message metadata.
type Metadata struct {
	// Model is the model used for generating the response.
	Model string `json:"model,omitempty" bson:"model,omitempty"`
	// TokensInput is the number of input tokens.
	TokensInput int `json:"tokensInput,omitempty" bson:"tokensInput,omitempty"`
	// TokensOutput is the number of output tokens.
	TokensOutput int `json:"tokensOutput,omitempty" bson:"tokensOutput,omitempty"`
	// Latency is the response latency in milliseconds.
	LatencyMs int64 `json:"latencyMs,omitempty" bson:"latencyMs,omitempty"`
	// AgentType is the type of agent that processed the message.
	AgentType string `json:"agentType,omitempty" bson:"agentType,omitempty"`
	// Custom holds additional custom metadata.
	Custom map[string]interface{} `json:"custom,omitempty" bson:"custom,omitempty"`
}

// NewMessage creates a new message with the given parameters.
func NewMessage(tenantID, conversationID, role, content, agentID, userID string) *Message {
	now := time.Now().UTC()
	return &Message{
		TenantID:       tenantID,
		ConversationID: conversationID,
		Role:           MessageRole(role),
		Content:        content,
		AgentID:        agentID,
		UserID:         userID,
		CreatedAt:      now,
		UpdatedAt:      now,
	}
}
