// Package mongodb provides the messages collection implementation.
package mongodb

import (
	"context"
	"fmt"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

	"github.com/unifiedui/agent-service/internal/core/docdb"
	"github.com/unifiedui/agent-service/internal/domain/models"
)

const (
	// MessagesCollectionName is the name of the unified messages collection.
	MessagesCollectionName = "messages"
)

// MessagesCollection implements the docdb.MessagesCollection interface for MongoDB.
// All messages (user and assistant) are stored in a SINGLE collection.
type MessagesCollection struct {
	collection *mongo.Collection
}

// NewMessagesCollection creates a new messages collection wrapper.
func NewMessagesCollection(db *mongo.Database) *MessagesCollection {
	return &MessagesCollection{
		collection: db.Collection(MessagesCollectionName),
	}
}

// Add inserts a new message (user or assistant).
func (c *MessagesCollection) Add(ctx context.Context, message *models.Message) error {
	if message.ID == "" {
		return fmt.Errorf("message ID is required")
	}

	message.CreatedAt = time.Now().UTC()
	message.UpdatedAt = message.CreatedAt

	_, err := c.collection.InsertOne(ctx, message)
	if err != nil {
		return fmt.Errorf("failed to insert message: %w", err)
	}

	return nil
}

// Get retrieves a message by ID.
func (c *MessagesCollection) Get(ctx context.Context, id string) (*models.Message, error) {
	var message models.Message
	err := c.collection.FindOne(ctx, bson.M{"_id": id}).Decode(&message)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get message: %w", err)
	}
	return &message, nil
}

// GetByUserMessageID retrieves assistant message by user message ID.
func (c *MessagesCollection) GetByUserMessageID(ctx context.Context, userMessageID string) (*models.Message, error) {
	filter := bson.M{
		"userMessageId": userMessageID,
		"type":          models.MessageTypeAssistant,
	}

	var message models.Message
	err := c.collection.FindOne(ctx, filter).Decode(&message)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get assistant message by user message ID: %w", err)
	}
	return &message, nil
}

// List retrieves messages with pagination and sorting.
func (c *MessagesCollection) List(ctx context.Context, opts *docdb.ListMessagesOptions) ([]*models.Message, error) {
	filter := c.buildFilter(opts)
	findOpts := c.buildFindOptions(opts)

	cursor, err := c.collection.Find(ctx, filter, findOpts)
	if err != nil {
		return nil, fmt.Errorf("failed to list messages: %w", err)
	}
	defer cursor.Close(ctx)

	var messages []*models.Message
	if err := cursor.All(ctx, &messages); err != nil {
		return nil, fmt.Errorf("failed to decode messages: %w", err)
	}

	return messages, nil
}

// ListChatHistory retrieves chat history as entries for a conversation.
func (c *MessagesCollection) ListChatHistory(ctx context.Context, opts *docdb.ListMessagesOptions) ([]models.ChatHistoryEntry, error) {
	messages, err := c.List(ctx, opts)
	if err != nil {
		return nil, fmt.Errorf("failed to list messages for chat history: %w", err)
	}

	entries := make([]models.ChatHistoryEntry, 0, len(messages))
	for _, msg := range messages {
		entries = append(entries, msg.ToChatHistoryEntry())
	}

	return entries, nil
}

// Update updates an existing message.
func (c *MessagesCollection) Update(ctx context.Context, message *models.Message) error {
	message.UpdatedAt = time.Now().UTC()

	result, err := c.collection.ReplaceOne(ctx, bson.M{"_id": message.ID}, message)
	if err != nil {
		return fmt.Errorf("failed to update message: %w", err)
	}

	if result.MatchedCount == 0 {
		return fmt.Errorf("message not found: %s", message.ID)
	}

	return nil
}

// Delete removes a message or all messages in a conversation.
func (c *MessagesCollection) Delete(ctx context.Context, opts *docdb.DeleteMessagesOptions) (int64, error) {
	if opts.MessageID != "" {
		// Delete specific message
		result, err := c.collection.DeleteOne(ctx, bson.M{
			"_id":      opts.MessageID,
			"tenantId": opts.TenantID,
		})
		if err != nil {
			return 0, fmt.Errorf("failed to delete message: %w", err)
		}
		return result.DeletedCount, nil
	}

	if opts.ConversationID != "" {
		// Delete all messages in conversation
		filter := bson.M{
			"conversationId": opts.ConversationID,
			"tenantId":       opts.TenantID,
		}

		result, err := c.collection.DeleteMany(ctx, filter)
		if err != nil {
			return 0, fmt.Errorf("failed to delete messages: %w", err)
		}
		return result.DeletedCount, nil
	}

	return 0, nil
}

// CountByConversation returns the count of messages in a conversation.
func (c *MessagesCollection) CountByConversation(ctx context.Context, tenantID, conversationID string) (int64, error) {
	filter := bson.M{
		"conversationId": conversationID,
		"tenantId":       tenantID,
	}

	count, err := c.collection.CountDocuments(ctx, filter)
	if err != nil {
		return 0, fmt.Errorf("failed to count messages: %w", err)
	}

	return count, nil
}

// EnsureIndexes creates necessary indexes for the messages collection.
func (c *MessagesCollection) EnsureIndexes(ctx context.Context) error {
	indexes := []mongo.IndexModel{
		{
			Keys: bson.D{
				{Key: "conversationId", Value: 1},
				{Key: "createdAt", Value: -1},
			},
			Options: options.Index().SetName("idx_conversation_created"),
		},
		{
			Keys: bson.D{
				{Key: "tenantId", Value: 1},
				{Key: "conversationId", Value: 1},
			},
			Options: options.Index().SetName("idx_tenant_conversation"),
		},
		{
			Keys: bson.D{
				{Key: "type", Value: 1},
				{Key: "createdAt", Value: -1},
			},
			Options: options.Index().SetName("idx_type_created"),
		},
		{
			Keys:    bson.D{{Key: "createdAt", Value: -1}},
			Options: options.Index().SetName("idx_created_at"),
		},
		{
			Keys:    bson.D{{Key: "userId", Value: 1}},
			Options: options.Index().SetName("idx_user_id"),
		},
		{
			Keys:    bson.D{{Key: "userMessageId", Value: 1}},
			Options: options.Index().SetName("idx_user_message_id"),
		},
		{
			Keys:    bson.D{{Key: "status", Value: 1}},
			Options: options.Index().SetName("idx_status"),
		},
	}

	_, err := c.collection.Indexes().CreateMany(ctx, indexes)
	if err != nil {
		return fmt.Errorf("failed to create messages indexes: %w", err)
	}

	return nil
}

// buildFilter creates a MongoDB filter from list options.
func (c *MessagesCollection) buildFilter(opts *docdb.ListMessagesOptions) bson.M {
	filter := bson.M{}

	if opts == nil {
		return filter
	}

	if opts.TenantID != "" {
		filter["tenantId"] = opts.TenantID
	}
	if opts.ConversationID != "" {
		filter["conversationId"] = opts.ConversationID
	}
	if opts.Type != "" {
		filter["type"] = opts.Type
	}

	return filter
}

// buildFindOptions creates MongoDB find options from list options.
func (c *MessagesCollection) buildFindOptions(opts *docdb.ListMessagesOptions) *options.FindOptions {
	findOpts := options.Find()

	if opts == nil {
		return findOpts
	}

	if opts.Limit > 0 {
		findOpts.SetLimit(opts.Limit)
	}
	if opts.Skip > 0 {
		findOpts.SetSkip(opts.Skip)
	}

	// Default to descending order by createdAt
	sortOrder := -1
	if opts.OrderBy == docdb.SortOrderAsc {
		sortOrder = 1
	}
	findOpts.SetSort(bson.D{{Key: "createdAt", Value: sortOrder}})

	return findOpts
}
