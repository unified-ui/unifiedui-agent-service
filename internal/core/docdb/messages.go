// Package docdb provides the messages collection interface.
package docdb

import (
	"context"

	"github.com/unifiedui/agent-service/internal/domain/models"
)

// SortOrder represents the sort direction.
type SortOrder string

const (
	// SortOrderAsc represents ascending order.
	SortOrderAsc SortOrder = "asc"
	// SortOrderDesc represents descending order.
	SortOrderDesc SortOrder = "desc"
)

// ListMessagesOptions contains options for listing messages.
type ListMessagesOptions struct {
	ConversationID string
	TenantID       string
	Limit          int64
	Skip           int64
	OrderBy        SortOrder // Order by createdAt
}

// DeleteMessagesOptions contains options for deleting messages.
type DeleteMessagesOptions struct {
	MessageID      string // Delete single message by ID
	ConversationID string // Delete all messages in conversation
	TenantID       string // Required for tenant isolation
}

// MessagesCollection defines the interface for message collection operations.
type MessagesCollection interface {
	// Add inserts a new message (UserMessage or AssistantMessage).
	AddUserMessage(ctx context.Context, message *models.UserMessage) error

	// AddAssistantMessage inserts a new assistant message.
	AddAssistantMessage(ctx context.Context, message *models.AssistantMessage) error

	// GetUserMessage retrieves a user message by ID.
	GetUserMessage(ctx context.Context, id string) (*models.UserMessage, error)

	// GetAssistantMessage retrieves an assistant message by ID.
	GetAssistantMessage(ctx context.Context, id string) (*models.AssistantMessage, error)

	// GetAssistantMessageByUserMessageID retrieves assistant message by user message ID.
	GetAssistantMessageByUserMessageID(ctx context.Context, userMessageID string) (*models.AssistantMessage, error)

	// ListUserMessages lists user messages with pagination and sorting.
	ListUserMessages(ctx context.Context, opts *ListMessagesOptions) ([]*models.UserMessage, error)

	// ListAssistantMessages lists assistant messages with pagination and sorting.
	ListAssistantMessages(ctx context.Context, opts *ListMessagesOptions) ([]*models.AssistantMessage, error)

	// ListChatHistory retrieves interleaved chat history for a conversation.
	// Returns messages ordered by createdAt (configurable order).
	ListChatHistory(ctx context.Context, opts *ListMessagesOptions) ([]models.ChatHistoryEntry, error)

	// UpdateUserMessage updates an existing user message.
	UpdateUserMessage(ctx context.Context, message *models.UserMessage) error

	// UpdateAssistantMessage updates an existing assistant message.
	UpdateAssistantMessage(ctx context.Context, message *models.AssistantMessage) error

	// Delete removes a message or all messages in a conversation.
	Delete(ctx context.Context, opts *DeleteMessagesOptions) (int64, error)

	// CountByConversation returns the count of messages in a conversation.
	CountByConversation(ctx context.Context, tenantID, conversationID string) (int64, error)

	// EnsureIndexes creates necessary indexes for the collection.
	EnsureIndexes(ctx context.Context) error
}
