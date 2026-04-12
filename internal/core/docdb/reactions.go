// Package docdb provides the reactions collection interface.
package docdb

import (
	"context"

	"github.com/unifiedui/agent-service/internal/domain/models"
)

// UpsertReactionOptions contains options for upserting a reaction.
type UpsertReactionOptions struct {
	TenantID       string
	ConversationID string
	MessageID      string
	UserID         string
}

// DeleteReactionOptions contains options for deleting reactions.
type DeleteReactionOptions struct {
	TenantID       string
	ConversationID string
	MessageID      string
	UserID         string
}

// ListReactionsOptions contains options for listing reactions.
type ListReactionsOptions struct {
	TenantID       string
	ConversationID string
	MessageID      string
}

// ListBulkReactionsOptions contains options for listing reactions for multiple messages.
type ListBulkReactionsOptions struct {
	TenantID       string
	ConversationID string
	MessageIDs     []string
}

// ReactionsCollection defines the interface for reaction collection operations.
type ReactionsCollection interface {
	// Upsert creates or updates a reaction (one per user per message).
	Upsert(ctx context.Context, reaction *models.MessageReaction) error

	// Get retrieves a reaction by tenant, message, and user.
	Get(ctx context.Context, opts *UpsertReactionOptions) (*models.MessageReaction, error)

	// ListByMessage retrieves all reactions for a message.
	ListByMessage(ctx context.Context, opts *ListReactionsOptions) ([]*models.MessageReaction, error)

	// ListByMessages retrieves all reactions for multiple messages in a single query.
	ListByMessages(ctx context.Context, opts *ListBulkReactionsOptions) ([]*models.MessageReaction, error)

	// Delete removes a user's reaction from a message.
	Delete(ctx context.Context, opts *DeleteReactionOptions) error

	// DeleteByConversation removes all reactions in a conversation (for cleanup).
	DeleteByConversation(ctx context.Context, tenantID, conversationID string) error

	// EnsureIndexes creates necessary indexes for the collection.
	EnsureIndexes(ctx context.Context) error
}
