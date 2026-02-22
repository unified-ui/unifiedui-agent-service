// Package mongodb provides the reactions collection implementation.
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
	// ReactionsCollectionName is the name of the message reactions collection.
	ReactionsCollectionName = "message_reactions"
)

// ReactionsCollection implements the docdb.ReactionsCollection interface for MongoDB.
type ReactionsCollection struct {
	collection *mongo.Collection
}

// NewReactionsCollection creates a new reactions collection wrapper.
func NewReactionsCollection(db *mongo.Database) *ReactionsCollection {
	return &ReactionsCollection{
		collection: db.Collection(ReactionsCollectionName),
	}
}

// Upsert creates or updates a reaction (one per user per message).
func (c *ReactionsCollection) Upsert(ctx context.Context, reaction *models.MessageReaction) error {
	filter := bson.M{
		"tenantId":  reaction.TenantID,
		"messageId": reaction.MessageID,
		"userId":    reaction.UserID,
	}

	now := time.Now().UTC()
	update := bson.M{
		"$set": bson.M{
			"reaction":       reaction.Reaction,
			"feedbackText":   reaction.FeedbackText,
			"conversationId": reaction.ConversationID,
			"updatedAt":      now,
		},
		"$setOnInsert": bson.M{
			"_id":       reaction.ID,
			"tenantId":  reaction.TenantID,
			"messageId": reaction.MessageID,
			"userId":    reaction.UserID,
			"createdAt": now,
		},
	}

	opts := options.Update().SetUpsert(true)
	_, err := c.collection.UpdateOne(ctx, filter, update, opts)
	if err != nil {
		return fmt.Errorf("failed to upsert reaction: %w", err)
	}

	return nil
}

// Get retrieves a reaction by tenant, message, and user.
func (c *ReactionsCollection) Get(ctx context.Context, opts *docdb.UpsertReactionOptions) (*models.MessageReaction, error) {
	filter := bson.M{
		"tenantId":  opts.TenantID,
		"messageId": opts.MessageID,
		"userId":    opts.UserID,
	}

	var reaction models.MessageReaction
	err := c.collection.FindOne(ctx, filter).Decode(&reaction)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get reaction: %w", err)
	}

	return &reaction, nil
}

// ListByMessage retrieves all reactions for a message.
func (c *ReactionsCollection) ListByMessage(ctx context.Context, opts *docdb.ListReactionsOptions) ([]*models.MessageReaction, error) {
	filter := bson.M{
		"tenantId":       opts.TenantID,
		"conversationId": opts.ConversationID,
		"messageId":      opts.MessageID,
	}

	findOpts := options.Find().SetSort(bson.D{{Key: "createdAt", Value: 1}})

	cursor, err := c.collection.Find(ctx, filter, findOpts)
	if err != nil {
		return nil, fmt.Errorf("failed to list reactions: %w", err)
	}
	defer cursor.Close(ctx)

	var reactions []*models.MessageReaction
	if err := cursor.All(ctx, &reactions); err != nil {
		return nil, fmt.Errorf("failed to decode reactions: %w", err)
	}

	return reactions, nil
}

// Delete removes a user's reaction from a message.
func (c *ReactionsCollection) Delete(ctx context.Context, opts *docdb.DeleteReactionOptions) error {
	filter := bson.M{
		"tenantId":  opts.TenantID,
		"messageId": opts.MessageID,
		"userId":    opts.UserID,
	}

	_, err := c.collection.DeleteOne(ctx, filter)
	if err != nil {
		return fmt.Errorf("failed to delete reaction: %w", err)
	}

	return nil
}

// DeleteByConversation removes all reactions in a conversation.
func (c *ReactionsCollection) DeleteByConversation(ctx context.Context, tenantID, conversationID string) error {
	filter := bson.M{
		"tenantId":       tenantID,
		"conversationId": conversationID,
	}

	_, err := c.collection.DeleteMany(ctx, filter)
	if err != nil {
		return fmt.Errorf("failed to delete reactions by conversation: %w", err)
	}

	return nil
}

// EnsureIndexes creates necessary indexes for the reactions collection.
func (c *ReactionsCollection) EnsureIndexes(ctx context.Context) error {
	indexes := []mongo.IndexModel{
		{
			Keys: bson.D{
				{Key: "tenantId", Value: 1},
				{Key: "messageId", Value: 1},
				{Key: "userId", Value: 1},
			},
			Options: options.Index().SetUnique(true),
		},
		{
			Keys: bson.D{
				{Key: "tenantId", Value: 1},
				{Key: "conversationId", Value: 1},
				{Key: "messageId", Value: 1},
			},
		},
		{
			Keys: bson.D{
				{Key: "tenantId", Value: 1},
				{Key: "userId", Value: 1},
			},
		},
	}

	_, err := c.collection.Indexes().CreateMany(ctx, indexes)
	if err != nil {
		return fmt.Errorf("failed to create reactions indexes: %w", err)
	}

	return nil
}
