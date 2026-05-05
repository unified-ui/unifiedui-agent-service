// Package mongodb provides the messages collection implementation.
package mongodb

import (
	"context"
	"errors"
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
	err := c.collection.FindOne(ctx, bson.M{"_id": sanitizeValue(id)}).Decode(&message)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get message: %w", err)
	}
	return &message, nil
}

// GetByUserMessageID retrieves assistant message by user message ID.
func (c *MessagesCollection) GetByUserMessageID(ctx context.Context, userMessageID string) (*models.Message, error) {
	filter := bson.M{
		"userMessageId": sanitizeValue(userMessageID),
		"type":          models.MessageTypeAssistant,
	}

	var message models.Message
	err := c.collection.FindOne(ctx, filter).Decode(&message)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
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
	defer func() { _ = cursor.Close(ctx) }()

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

// Search searches messages by content text using case-insensitive regex matching.
func (c *MessagesCollection) Search(ctx context.Context, opts *docdb.SearchMessagesOptions) ([]*models.Message, error) {
	filter := bson.M{
		"tenantId": sanitizeValue(opts.TenantID),
		"content":  bson.M{"$regex": sanitizeRegex(opts.Query), "$options": "i"},
	}

	findOpts := options.Find().SetSort(bson.D{{Key: "createdAt", Value: -1}})
	if opts.Limit > 0 {
		findOpts.SetLimit(opts.Limit)
	}
	if opts.Skip > 0 {
		findOpts.SetSkip(opts.Skip)
	}

	cursor, err := c.collection.Find(ctx, filter, findOpts)
	if err != nil {
		return nil, fmt.Errorf("failed to search messages: %w", err)
	}
	defer func() { _ = cursor.Close(ctx) }()

	var messages []*models.Message
	if err := cursor.All(ctx, &messages); err != nil {
		return nil, fmt.Errorf("failed to decode search results: %w", err)
	}

	return messages, nil
}

// Update updates an existing message.
func (c *MessagesCollection) Update(ctx context.Context, message *models.Message) error {
	message.UpdatedAt = time.Now().UTC()

	result, err := c.collection.ReplaceOne(ctx, bson.M{"_id": sanitizeValue(message.ID)}, message)
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
			"_id":      sanitizeValue(opts.MessageID),
			"tenantId": sanitizeValue(opts.TenantID),
		})
		if err != nil {
			return 0, fmt.Errorf("failed to delete message: %w", err)
		}
		return result.DeletedCount, nil
	}

	if opts.ConversationID != "" {
		// Delete all messages in conversation
		filter := bson.M{
			"conversationId": sanitizeValue(opts.ConversationID),
			"tenantId":       sanitizeValue(opts.TenantID),
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
		"conversationId": sanitizeValue(conversationID),
		"tenantId":       sanitizeValue(tenantID),
	}

	count, err := c.collection.CountDocuments(ctx, filter)
	if err != nil {
		return 0, fmt.Errorf("failed to count messages: %w", err)
	}

	return count, nil
}

// GetMessageStats returns aggregated message counts by status for a tenant,
// grouped by chat agent. Uses a single $match + $group aggregation pipeline.
func (c *MessagesCollection) GetMessageStats(ctx context.Context, tenantID string, filter *models.MessageStatsFilter) (*models.MessageStatsResult, error) {
	match := bson.M{
		"tenantId": sanitizeValue(tenantID),
		"type":     "assistant",
		"status":   bson.M{"$in": bson.A{"success", "failed"}},
	}
	if filter != nil {
		if len(filter.ChatAgentIDs) > 0 {
			ids := make(bson.A, 0, len(filter.ChatAgentIDs))
			for _, id := range filter.ChatAgentIDs {
				ids = append(ids, sanitizeValue(id))
			}
			match["chatAgentId"] = bson.M{"$in": ids}
		}
		if !filter.From.IsZero() || !filter.To.IsZero() {
			createdAt := bson.M{}
			if !filter.From.IsZero() {
				createdAt["$gte"] = filter.From
			}
			if !filter.To.IsZero() {
				createdAt["$lte"] = filter.To
			}
			match["createdAt"] = createdAt
		}
	}

	pipeline := mongo.Pipeline{
		{{Key: "$match", Value: match}},
		{{Key: "$group", Value: bson.M{
			"_id":   "$chatAgentId",
			"total": bson.M{"$sum": 1},
			"failed": bson.M{"$sum": bson.M{"$cond": bson.A{
				bson.M{"$eq": bson.A{"$status", "failed"}},
				1,
				0,
			}}},
		}}},
	}

	cursor, err := c.collection.Aggregate(ctx, pipeline)
	if err != nil {
		return nil, fmt.Errorf("failed to aggregate message stats: %w", err)
	}
	defer cursor.Close(ctx)

	type groupRow struct {
		ID     interface{} `bson:"_id"`
		Total  int64       `bson:"total"`
		Failed int64       `bson:"failed"`
	}

	result := &models.MessageStatsResult{
		PerAgent: make([]models.MessageStatsPerAgent, 0),
	}

	for cursor.Next(ctx) {
		var row groupRow
		if err := cursor.Decode(&row); err != nil {
			return nil, fmt.Errorf("failed to decode message stats row: %w", err)
		}
		success := row.Total - row.Failed
		result.Aggregate.TotalMessages += row.Total
		result.Aggregate.SuccessCount += success
		result.Aggregate.FailedCount += row.Failed

		agentID, ok := row.ID.(string)
		if !ok || agentID == "" {
			continue
		}
		result.PerAgent = append(result.PerAgent, models.MessageStatsPerAgent{
			ChatAgentID:   agentID,
			TotalMessages: row.Total,
			SuccessCount:  success,
			FailedCount:   row.Failed,
		})
	}
	if err := cursor.Err(); err != nil {
		return nil, fmt.Errorf("cursor error during message stats aggregation: %w", err)
	}

	return result, nil
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
		{
			Keys: bson.D{
				{Key: "tenantId", Value: 1},
				{Key: "content", Value: 1},
				{Key: "createdAt", Value: -1},
			},
			Options: options.Index().SetName("idx_tenant_content_search"),
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
		filter["tenantId"] = sanitizeValue(opts.TenantID)
	}
	if opts.ConversationID != "" {
		filter["conversationId"] = sanitizeValue(opts.ConversationID)
	}
	if opts.Type != "" {
		filter["type"] = sanitizeValue(string(opts.Type))
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
