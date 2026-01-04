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
	Type           models.MessageType // Optional: filter by message type
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
// All messages (user and assistant) are stored in a SINGLE collection.
type MessagesCollection interface {
	// Add inserts a new message (user or assistant).
	Add(ctx context.Context, message *models.Message) error

	// Get retrieves a message by ID.
	Get(ctx context.Context, id string) (*models.Message, error)

	// GetByUserMessageID retrieves assistant message by user message ID.
	GetByUserMessageID(ctx context.Context, userMessageID string) (*models.Message, error)

	// List retrieves messages with pagination and sorting.
	// Can filter by message type (user/assistant) via opts.Type.
	List(ctx context.Context, opts *ListMessagesOptions) ([]*models.Message, error)

	// ListChatHistory retrieves chat history as entries for a conversation.
	// Returns messages ordered by createdAt (configurable order).
	ListChatHistory(ctx context.Context, opts *ListMessagesOptions) ([]models.ChatHistoryEntry, error)

	// Update updates an existing message.
	Update(ctx context.Context, message *models.Message) error

	// Delete removes a message or all messages in a conversation.
	Delete(ctx context.Context, opts *DeleteMessagesOptions) (int64, error)

	// CountByConversation returns the count of messages in a conversation.
	CountByConversation(ctx context.Context, tenantID, conversationID string) (int64, error)

	// EnsureIndexes creates necessary indexes for the collection.
	EnsureIndexes(ctx context.Context) error
}
