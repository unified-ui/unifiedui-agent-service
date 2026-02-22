// Package models contains domain models for the UnifiedUI Chat Service.
package models

import (
	"time"
)

// MessageType represents the type of message (user or assistant).
type MessageType string

const (
	// MessageTypeUser represents a user message.
	MessageTypeUser MessageType = "user"
	// MessageTypeAssistant represents an assistant message.
	MessageTypeAssistant MessageType = "assistant"
)

// MessageStatus represents the status of a message (mainly for assistant messages).
type MessageStatus string

const (
	// MessageStatusPending indicates the message is being processed.
	MessageStatusPending MessageStatus = "pending"
	// MessageStatusSuccess indicates the message was processed successfully.
	MessageStatusSuccess MessageStatus = "success"
	// MessageStatusFailed indicates the message processing failed.
	MessageStatusFailed MessageStatus = "failed"
	// MessageStatusCanceled indicates the message generation was canceled by the user.
	MessageStatusCanceled MessageStatus = "cancelled" //nolint:misspell // value must stay "cancelled" for external API compatibility
)

// MessageRequest represents the original request that triggered a message.
type MessageRequest struct {
	ChatAgentID    string                 `json:"chatAgentId" bson:"chatAgentId"`
	ConversationID string                 `json:"conversationId,omitempty" bson:"conversationId,omitempty"`
	Message        MessageRequestContent  `json:"message" bson:"message"`
	InvokeConfig   MessageInvokeConfig    `json:"invokeConfig,omitempty" bson:"invokeConfig,omitempty"`
	Extra          map[string]interface{} `json:"extra,omitempty" bson:"extra,omitempty"`
}

// MessageRequestContent represents the message content in a request.
type MessageRequestContent struct {
	Content     string   `json:"content" bson:"content"`
	Attachments []string `json:"attachments,omitempty" bson:"attachments,omitempty"`
}

// MessageInvokeConfig represents configuration for agent invocation.
type MessageInvokeConfig struct {
	ChatHistoryMessageCount int               `json:"chatHistoryMessageCount,omitempty" bson:"chatHistoryMessageCount,omitempty"`
	ContextData             map[string]string `json:"contextData,omitempty" bson:"contextData,omitempty"`
}

// AssistantMetadata holds metadata about an assistant response.
type AssistantMetadata struct {
	Model        string `json:"model,omitempty" bson:"model,omitempty"`
	TokensInput  int    `json:"tokensInput,omitempty" bson:"tokensInput,omitempty"`
	TokensOutput int    `json:"tokensOutput,omitempty" bson:"tokensOutput,omitempty"`
	LatencyMs    int64  `json:"latencyMs,omitempty" bson:"latencyMs,omitempty"`
	ExecutionID  string `json:"executionId,omitempty" bson:"executionId,omitempty"`
	AgentType    string `json:"agentType,omitempty" bson:"agentType,omitempty"`
	// ExtMessageID is the external message ID from the backend (e.g., Foundry message ID).
	// This allows mapping between chat messages and trace nodes.
	ExtMessageID string `json:"extMessageId,omitempty" bson:"extMessageId,omitempty"`
}

// AttachmentMetadata holds metadata about a file attached to a user message.
type AttachmentMetadata struct {
	FileName     string `json:"fileName" bson:"fileName"`
	FileType     string `json:"fileType" bson:"fileType"`
	FileSize     int64  `json:"fileSize" bson:"fileSize"`
	FileCategory string `json:"fileCategory" bson:"fileCategory"`
}

// StatusTrace represents a trace entry during message processing.
type StatusTrace struct {
	Type      string                 `json:"type" bson:"type"`
	Name      string                 `json:"name,omitempty" bson:"name,omitempty"`
	Content   string                 `json:"content,omitempty" bson:"content,omitempty"`
	Timestamp time.Time              `json:"timestamp" bson:"timestamp"`
	Data      map[string]interface{} `json:"data,omitempty" bson:"data,omitempty"`
}

// Message represents a unified message in the chat (user or assistant).
// All messages are stored in a SINGLE collection, differentiated by the Type field.
type Message struct {
	// Common fields for all message types
	ID             string      `json:"id" bson:"_id"`
	Type           MessageType `json:"type" bson:"type"`
	ConversationID string      `json:"conversationId" bson:"conversationId"`
	ChatAgentID    string      `json:"chatAgentId" bson:"chatAgentId"`
	TenantID       string      `json:"tenantId" bson:"tenantId"`
	Content        string      `json:"content" bson:"content"`
	CreatedAt      time.Time   `json:"createdAt" bson:"createdAt"`
	UpdatedAt      time.Time   `json:"updatedAt" bson:"updatedAt"`

	// User message specific fields (only set when Type == MessageTypeUser)
	UserID              string               `json:"userId,omitempty" bson:"userId,omitempty"`
	Request             *MessageRequest      `json:"request,omitempty" bson:"request,omitempty"`
	Attachments         []string             `json:"attachments,omitempty" bson:"attachments,omitempty"`
	AttachmentsMetadata []AttachmentMetadata `json:"attachmentsMetadata,omitempty" bson:"attachmentsMetadata,omitempty"`

	// Assistant message specific fields (only set when Type == MessageTypeAssistant)
	UserMessageID string             `json:"userMessageId,omitempty" bson:"userMessageId,omitempty"`
	Status        MessageStatus      `json:"status,omitempty" bson:"status,omitempty"`
	StatusTraces  []StatusTrace      `json:"statusTraces,omitempty" bson:"statusTraces,omitempty"`
	ErrorMessage  string             `json:"errorMessage,omitempty" bson:"errorMessage,omitempty"`
	Metadata      *AssistantMetadata `json:"metadata,omitempty" bson:"metadata,omitempty"`
}

// ChatHistoryEntry represents a single entry in chat history.
type ChatHistoryEntry struct {
	Role      MessageType `json:"role" bson:"role"`
	Content   string      `json:"content" bson:"content"`
	Timestamp time.Time   `json:"timestamp" bson:"timestamp"`
}

// NewUserMessage creates a new user Message.
func NewUserMessage(
	tenantID string,
	conversationID string,
	chatAgentID string,
	userID string,
	content string,
	attachments []string,
	request *MessageRequest,
) *Message {
	now := time.Now().UTC()
	return &Message{
		Type:           MessageTypeUser,
		ConversationID: conversationID,
		ChatAgentID:    chatAgentID,
		TenantID:       tenantID,
		UserID:         userID,
		Content:        content,
		Request:        request,
		Attachments:    attachments,
		CreatedAt:      now,
		UpdatedAt:      now,
	}
}

// NewAssistantMessage creates a new assistant Message.
func NewAssistantMessage(
	tenantID string,
	conversationID string,
	userMessageID string,
	chatAgentID string,
	content string,
	status MessageStatus,
) *Message {
	now := time.Now().UTC()
	return &Message{
		Type:           MessageTypeAssistant,
		ConversationID: conversationID,
		UserMessageID:  userMessageID,
		ChatAgentID:    chatAgentID,
		TenantID:       tenantID,
		Content:        content,
		Status:         status,
		StatusTraces:   []StatusTrace{},
		CreatedAt:      now,
		UpdatedAt:      now,
	}
}

// IsUserMessage returns true if this is a user message.
func (m *Message) IsUserMessage() bool {
	return m.Type == MessageTypeUser
}

// IsAssistantMessage returns true if this is an assistant message.
func (m *Message) IsAssistantMessage() bool {
	return m.Type == MessageTypeAssistant
}

// SetError sets the error message and updates the status to failed.
func (m *Message) SetError(errorMessage string) {
	m.Status = MessageStatusFailed
	m.ErrorMessage = errorMessage
	m.UpdatedAt = time.Now().UTC()
}

// SetCanceled sets the content and updates the status to canceled.
func (m *Message) SetCanceled(content string) {
	m.Content = content
	m.Status = MessageStatusCanceled
	m.UpdatedAt = time.Now().UTC()
}

// SetSuccess sets the content and updates the status to success.
func (m *Message) SetSuccess(content string) {
	m.Content = content
	m.Status = MessageStatusSuccess
	m.UpdatedAt = time.Now().UTC()
}

// AddStatusTrace adds a status trace entry.
func (m *Message) AddStatusTrace(traceType, name, content string, data map[string]interface{}) {
	m.StatusTraces = append(m.StatusTraces, StatusTrace{
		Type:      traceType,
		Name:      name,
		Content:   content,
		Timestamp: time.Now().UTC(),
		Data:      data,
	})
	m.UpdatedAt = time.Now().UTC()
}

// SetMetadata sets the assistant metadata.
func (m *Message) SetMetadata(metadata *AssistantMetadata) {
	m.Metadata = metadata
	m.UpdatedAt = time.Now().UTC()
}

// ToChatHistoryEntry converts a Message to a ChatHistoryEntry.
func (m *Message) ToChatHistoryEntry() ChatHistoryEntry {
	return ChatHistoryEntry{
		Role:      m.Type,
		Content:   m.Content,
		Timestamp: m.CreatedAt,
	}
}
