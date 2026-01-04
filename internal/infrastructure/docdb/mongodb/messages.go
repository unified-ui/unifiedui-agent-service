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
	// UserMessagesCollection is the name of the user messages collection.
	UserMessagesCollection = "user_messages"
	// AssistantMessagesCollection is the name of the assistant messages collection.
	AssistantMessagesCollection = "assistant_messages"
)

// MessagesCollection implements the docdb.MessagesCollection interface for MongoDB.
type MessagesCollection struct {
	userMessages      *mongo.Collection
	assistantMessages *mongo.Collection
}

// NewMessagesCollection creates a new messages collection wrapper.
func NewMessagesCollection(db *mongo.Database) *MessagesCollection {
	return &MessagesCollection{
		userMessages:      db.Collection(UserMessagesCollection),
		assistantMessages: db.Collection(AssistantMessagesCollection),
	}
}

// AddUserMessage inserts a new user message.
func (c *MessagesCollection) AddUserMessage(ctx context.Context, message *models.UserMessage) error {
	if message.ID == "" {
		return fmt.Errorf("message ID is required")
	}

	message.CreatedAt = time.Now().UTC()
	message.UpdatedAt = message.CreatedAt

	_, err := c.userMessages.InsertOne(ctx, message)
	if err != nil {
		return fmt.Errorf("failed to insert user message: %w", err)
	}

	return nil
}

// AddAssistantMessage inserts a new assistant message.
func (c *MessagesCollection) AddAssistantMessage(ctx context.Context, message *models.AssistantMessage) error {
	if message.ID == "" {
		return fmt.Errorf("message ID is required")
	}

	message.CreatedAt = time.Now().UTC()
	message.UpdatedAt = message.CreatedAt

	_, err := c.assistantMessages.InsertOne(ctx, message)
	if err != nil {
		return fmt.Errorf("failed to insert assistant message: %w", err)
	}

	return nil
}

// GetUserMessage retrieves a user message by ID.
func (c *MessagesCollection) GetUserMessage(ctx context.Context, id string) (*models.UserMessage, error) {
	var message models.UserMessage
	err := c.userMessages.FindOne(ctx, bson.M{"_id": id}).Decode(&message)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get user message: %w", err)
	}
	return &message, nil
}

// GetAssistantMessage retrieves an assistant message by ID.
func (c *MessagesCollection) GetAssistantMessage(ctx context.Context, id string) (*models.AssistantMessage, error) {
	var message models.AssistantMessage
	err := c.assistantMessages.FindOne(ctx, bson.M{"_id": id}).Decode(&message)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get assistant message: %w", err)
	}
	return &message, nil
}

// GetAssistantMessageByUserMessageID retrieves assistant message by user message ID.
func (c *MessagesCollection) GetAssistantMessageByUserMessageID(ctx context.Context, userMessageID string) (*models.AssistantMessage, error) {
	var message models.AssistantMessage
	err := c.assistantMessages.FindOne(ctx, bson.M{"userMessageId": userMessageID}).Decode(&message)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get assistant message by user message ID: %w", err)
	}
	return &message, nil
}

// ListUserMessages lists user messages with pagination and sorting.
func (c *MessagesCollection) ListUserMessages(ctx context.Context, opts *docdb.ListMessagesOptions) ([]*models.UserMessage, error) {
	filter := c.buildFilter(opts)
	findOpts := c.buildFindOptions(opts)

	cursor, err := c.userMessages.Find(ctx, filter, findOpts)
	if err != nil {
		return nil, fmt.Errorf("failed to list user messages: %w", err)
	}
	defer cursor.Close(ctx)

	var messages []*models.UserMessage
	if err := cursor.All(ctx, &messages); err != nil {
		return nil, fmt.Errorf("failed to decode user messages: %w", err)
	}

	return messages, nil
}

// ListAssistantMessages lists assistant messages with pagination and sorting.
func (c *MessagesCollection) ListAssistantMessages(ctx context.Context, opts *docdb.ListMessagesOptions) ([]*models.AssistantMessage, error) {
	filter := c.buildFilter(opts)
	findOpts := c.buildFindOptions(opts)

	cursor, err := c.assistantMessages.Find(ctx, filter, findOpts)
	if err != nil {
		return nil, fmt.Errorf("failed to list assistant messages: %w", err)
	}
	defer cursor.Close(ctx)

	var messages []*models.AssistantMessage
	if err := cursor.All(ctx, &messages); err != nil {
		return nil, fmt.Errorf("failed to decode assistant messages: %w", err)
	}

	return messages, nil
}

// ListChatHistory retrieves interleaved chat history for a conversation.
func (c *MessagesCollection) ListChatHistory(ctx context.Context, opts *docdb.ListMessagesOptions) ([]models.ChatHistoryEntry, error) {
	// Get user messages
	userMessages, err := c.ListUserMessages(ctx, opts)
	if err != nil {
		return nil, fmt.Errorf("failed to list user messages for chat history: %w", err)
	}

	// Get assistant messages
	assistantMessages, err := c.ListAssistantMessages(ctx, opts)
	if err != nil {
		return nil, fmt.Errorf("failed to list assistant messages for chat history: %w", err)
	}

	// Convert to chat history entries
	entries := make([]models.ChatHistoryEntry, 0, len(userMessages)+len(assistantMessages))

	for _, msg := range userMessages {
		entries = append(entries, msg.ToChatHistoryEntry())
	}
	for _, msg := range assistantMessages {
		entries = append(entries, msg.ToChatHistoryEntry())
	}

	// Sort by timestamp
	sortChatHistory(entries, opts.OrderBy)

	// Apply limit after merging
	if opts.Limit > 0 && int64(len(entries)) > opts.Limit {
		entries = entries[:opts.Limit]
	}

	return entries, nil
}

// sortChatHistory sorts chat history entries by timestamp.
func sortChatHistory(entries []models.ChatHistoryEntry, order docdb.SortOrder) {
	// Simple bubble sort for small arrays (chat history is typically < 100 entries)
	n := len(entries)
	for i := 0; i < n-1; i++ {
		for j := 0; j < n-i-1; j++ {
			shouldSwap := false
			if order == docdb.SortOrderDesc {
				shouldSwap = entries[j].Timestamp.Before(entries[j+1].Timestamp)
			} else {
				shouldSwap = entries[j].Timestamp.After(entries[j+1].Timestamp)
			}
			if shouldSwap {
				entries[j], entries[j+1] = entries[j+1], entries[j]
			}
		}
	}
}

// UpdateUserMessage updates an existing user message.
func (c *MessagesCollection) UpdateUserMessage(ctx context.Context, message *models.UserMessage) error {
	message.UpdatedAt = time.Now().UTC()

	result, err := c.userMessages.ReplaceOne(ctx, bson.M{"_id": message.ID}, message)
	if err != nil {
		return fmt.Errorf("failed to update user message: %w", err)
	}

	if result.MatchedCount == 0 {
		return fmt.Errorf("user message not found: %s", message.ID)
	}

	return nil
}

// UpdateAssistantMessage updates an existing assistant message.
func (c *MessagesCollection) UpdateAssistantMessage(ctx context.Context, message *models.AssistantMessage) error {
	message.UpdatedAt = time.Now().UTC()

	result, err := c.assistantMessages.ReplaceOne(ctx, bson.M{"_id": message.ID}, message)
	if err != nil {
		return fmt.Errorf("failed to update assistant message: %w", err)
	}

	if result.MatchedCount == 0 {
		return fmt.Errorf("assistant message not found: %s", message.ID)
	}

	return nil
}

// Delete removes a message or all messages in a conversation.
func (c *MessagesCollection) Delete(ctx context.Context, opts *docdb.DeleteMessagesOptions) (int64, error) {
	var totalDeleted int64

	if opts.MessageID != "" {
		// Delete specific message from both collections
		userResult, err := c.userMessages.DeleteOne(ctx, bson.M{
			"_id":      opts.MessageID,
			"tenantId": opts.TenantID,
		})
		if err != nil {
			return 0, fmt.Errorf("failed to delete user message: %w", err)
		}
		totalDeleted += userResult.DeletedCount

		assistantResult, err := c.assistantMessages.DeleteOne(ctx, bson.M{
			"_id":      opts.MessageID,
			"tenantId": opts.TenantID,
		})
		if err != nil {
			return totalDeleted, fmt.Errorf("failed to delete assistant message: %w", err)
		}
		totalDeleted += assistantResult.DeletedCount

	} else if opts.ConversationID != "" {
		// Delete all messages in conversation
		filter := bson.M{
			"conversationId": opts.ConversationID,
			"tenantId":       opts.TenantID,
		}

		userResult, err := c.userMessages.DeleteMany(ctx, filter)
		if err != nil {
			return 0, fmt.Errorf("failed to delete user messages: %w", err)
		}
		totalDeleted += userResult.DeletedCount

		assistantResult, err := c.assistantMessages.DeleteMany(ctx, filter)
		if err != nil {
			return totalDeleted, fmt.Errorf("failed to delete assistant messages: %w", err)
		}
		totalDeleted += assistantResult.DeletedCount
	}

	return totalDeleted, nil
}

// CountByConversation returns the count of messages in a conversation.
func (c *MessagesCollection) CountByConversation(ctx context.Context, tenantID, conversationID string) (int64, error) {
	filter := bson.M{
		"conversationId": conversationID,
		"tenantId":       tenantID,
	}

	userCount, err := c.userMessages.CountDocuments(ctx, filter)
	if err != nil {
		return 0, fmt.Errorf("failed to count user messages: %w", err)
	}

	assistantCount, err := c.assistantMessages.CountDocuments(ctx, filter)
	if err != nil {
		return userCount, fmt.Errorf("failed to count assistant messages: %w", err)
	}

	return userCount + assistantCount, nil
}

// EnsureIndexes creates necessary indexes for the messages collections.
func (c *MessagesCollection) EnsureIndexes(ctx context.Context) error {
	// Indexes for user_messages collection
	userIndexes := []mongo.IndexModel{
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
			Keys:    bson.D{{Key: "createdAt", Value: -1}},
			Options: options.Index().SetName("idx_created_at"),
		},
		{
			Keys:    bson.D{{Key: "userId", Value: 1}},
			Options: options.Index().SetName("idx_user_id"),
		},
	}

	_, err := c.userMessages.Indexes().CreateMany(ctx, userIndexes)
	if err != nil {
		return fmt.Errorf("failed to create user messages indexes: %w", err)
	}

	// Indexes for assistant_messages collection
	assistantIndexes := []mongo.IndexModel{
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
			Keys:    bson.D{{Key: "createdAt", Value: -1}},
			Options: options.Index().SetName("idx_created_at"),
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

	_, err = c.assistantMessages.Indexes().CreateMany(ctx, assistantIndexes)
	if err != nil {
		return fmt.Errorf("failed to create assistant messages indexes: %w", err)
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
