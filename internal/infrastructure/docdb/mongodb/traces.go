// Package mongodb provides the traces collection implementation.
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
	// TracesCollectionName is the name of the traces collection.
	TracesCollectionName = "traces"
)

// TracesCollection implements the docdb.TracesCollection interface for MongoDB.
type TracesCollection struct {
	collection *mongo.Collection
}

// NewTracesCollection creates a new traces collection wrapper.
func NewTracesCollection(db *mongo.Database) *TracesCollection {
	return &TracesCollection{
		collection: db.Collection(TracesCollectionName),
	}
}

// Create inserts a new trace.
func (c *TracesCollection) Create(ctx context.Context, trace *models.Trace) error {
	if trace.ID == "" {
		return fmt.Errorf("trace ID is required")
	}

	trace.CreatedAt = time.Now().UTC()
	trace.UpdatedAt = trace.CreatedAt

	_, err := c.collection.InsertOne(ctx, trace)
	if err != nil {
		return fmt.Errorf("failed to insert trace: %w", err)
	}

	return nil
}

// Get retrieves a trace by ID.
func (c *TracesCollection) Get(ctx context.Context, id string) (*models.Trace, error) {
	var trace models.Trace
	err := c.collection.FindOne(ctx, bson.M{"_id": id}).Decode(&trace)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get trace: %w", err)
	}
	return &trace, nil
}

// GetByConversation retrieves a trace by conversation ID.
func (c *TracesCollection) GetByConversation(ctx context.Context, tenantID, conversationID string) (*models.Trace, error) {
	filter := bson.M{
		"tenantId":       tenantID,
		"conversationId": conversationID,
		"contextType":    models.TraceContextConversation,
	}

	var trace models.Trace
	err := c.collection.FindOne(ctx, filter).Decode(&trace)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get trace by conversation: %w", err)
	}
	return &trace, nil
}

// GetByAutonomousAgent retrieves a trace by autonomous agent ID.
func (c *TracesCollection) GetByAutonomousAgent(ctx context.Context, tenantID, autonomousAgentID string) (*models.Trace, error) {
	filter := bson.M{
		"tenantId":          tenantID,
		"autonomousAgentId": autonomousAgentID,
		"contextType":       models.TraceContextAutonomousAgent,
	}

	var trace models.Trace
	err := c.collection.FindOne(ctx, filter).Decode(&trace)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get trace by autonomous agent: %w", err)
	}
	return &trace, nil
}

// List retrieves traces with pagination and filtering.
func (c *TracesCollection) List(ctx context.Context, opts *docdb.ListTracesOptions) ([]*models.Trace, error) {
	filter := c.buildFilter(opts)
	findOpts := c.buildFindOptions(opts)

	cursor, err := c.collection.Find(ctx, filter, findOpts)
	if err != nil {
		return nil, fmt.Errorf("failed to list traces: %w", err)
	}
	defer cursor.Close(ctx)

	var traces []*models.Trace
	if err := cursor.All(ctx, &traces); err != nil {
		return nil, fmt.Errorf("failed to decode traces: %w", err)
	}

	return traces, nil
}

// Update replaces an existing trace completely.
func (c *TracesCollection) Update(ctx context.Context, trace *models.Trace) error {
	trace.UpdatedAt = time.Now().UTC()

	result, err := c.collection.ReplaceOne(ctx, bson.M{"_id": trace.ID}, trace)
	if err != nil {
		return fmt.Errorf("failed to update trace: %w", err)
	}

	if result.MatchedCount == 0 {
		return fmt.Errorf("trace not found: %s", trace.ID)
	}

	return nil
}

// AddNodes appends nodes to an existing trace.
func (c *TracesCollection) AddNodes(ctx context.Context, id string, nodes []models.TraceNode) error {
	update := bson.M{
		"$push": bson.M{
			"nodes": bson.M{
				"$each": nodes,
			},
		},
		"$set": bson.M{
			"updatedAt": time.Now().UTC(),
		},
	}

	result, err := c.collection.UpdateOne(ctx, bson.M{"_id": id}, update)
	if err != nil {
		return fmt.Errorf("failed to add nodes to trace: %w", err)
	}

	if result.MatchedCount == 0 {
		return fmt.Errorf("trace not found: %s", id)
	}

	return nil
}

// AddLogs appends logs to an existing trace.
func (c *TracesCollection) AddLogs(ctx context.Context, id string, logs []interface{}) error {
	update := bson.M{
		"$push": bson.M{
			"logs": bson.M{
				"$each": logs,
			},
		},
		"$set": bson.M{
			"updatedAt": time.Now().UTC(),
		},
	}

	result, err := c.collection.UpdateOne(ctx, bson.M{"_id": id}, update)
	if err != nil {
		return fmt.Errorf("failed to add logs to trace: %w", err)
	}

	if result.MatchedCount == 0 {
		return fmt.Errorf("trace not found: %s", id)
	}

	return nil
}

// Delete removes a trace by ID.
func (c *TracesCollection) Delete(ctx context.Context, id string) error {
	result, err := c.collection.DeleteOne(ctx, bson.M{"_id": id})
	if err != nil {
		return fmt.Errorf("failed to delete trace: %w", err)
	}

	if result.DeletedCount == 0 {
		return fmt.Errorf("trace not found: %s", id)
	}

	return nil
}

// DeleteByConversation removes the trace for a conversation.
func (c *TracesCollection) DeleteByConversation(ctx context.Context, tenantID, conversationID string) error {
	filter := bson.M{
		"tenantId":       tenantID,
		"conversationId": conversationID,
		"contextType":    models.TraceContextConversation,
	}

	_, err := c.collection.DeleteOne(ctx, filter)
	if err != nil {
		return fmt.Errorf("failed to delete trace by conversation: %w", err)
	}

	return nil
}

// DeleteByAutonomousAgent removes the trace for an autonomous agent.
func (c *TracesCollection) DeleteByAutonomousAgent(ctx context.Context, tenantID, autonomousAgentID string) error {
	filter := bson.M{
		"tenantId":          tenantID,
		"autonomousAgentId": autonomousAgentID,
		"contextType":       models.TraceContextAutonomousAgent,
	}

	_, err := c.collection.DeleteOne(ctx, filter)
	if err != nil {
		return fmt.Errorf("failed to delete trace by autonomous agent: %w", err)
	}

	return nil
}

// EnsureIndexes creates necessary indexes for the traces collection.
func (c *TracesCollection) EnsureIndexes(ctx context.Context) error {
	indexes := []mongo.IndexModel{
		// Primary ID index (automatic on _id)

		// Tenant isolation index
		{
			Keys:    bson.D{{Key: "tenantId", Value: 1}},
			Options: options.Index().SetName("idx_tenant_id"),
		},
		// Conversation context index (unique per tenant+conversation)
		{
			Keys: bson.D{
				{Key: "tenantId", Value: 1},
				{Key: "conversationId", Value: 1},
			},
			Options: options.Index().SetName("idx_tenant_conversation").SetSparse(true),
		},
		// Application + Conversation compound index
		{
			Keys: bson.D{
				{Key: "tenantId", Value: 1},
				{Key: "applicationId", Value: 1},
				{Key: "conversationId", Value: 1},
			},
			Options: options.Index().SetName("idx_tenant_app_conversation").SetSparse(true),
		},
		// Autonomous agent context index (unique per tenant+agent)
		{
			Keys: bson.D{
				{Key: "tenantId", Value: 1},
				{Key: "autonomousAgentId", Value: 1},
			},
			Options: options.Index().SetName("idx_tenant_autonomous_agent").SetSparse(true),
		},
		// Context type index for filtering
		{
			Keys:    bson.D{{Key: "contextType", Value: 1}},
			Options: options.Index().SetName("idx_context_type"),
		},
		// Created at index for sorting and time-based queries
		{
			Keys:    bson.D{{Key: "createdAt", Value: -1}},
			Options: options.Index().SetName("idx_created_at"),
		},
		// Reference ID index for external system lookup
		{
			Keys:    bson.D{{Key: "referenceId", Value: 1}},
			Options: options.Index().SetName("idx_reference_id").SetSparse(true),
		},
	}

	_, err := c.collection.Indexes().CreateMany(ctx, indexes)
	if err != nil {
		return fmt.Errorf("failed to create traces indexes: %w", err)
	}

	return nil
}

// buildFilter creates a MongoDB filter from list options.
func (c *TracesCollection) buildFilter(opts *docdb.ListTracesOptions) bson.M {
	filter := bson.M{}

	if opts == nil {
		return filter
	}

	if opts.TenantID != "" {
		filter["tenantId"] = opts.TenantID
	}
	if opts.ApplicationID != "" {
		filter["applicationId"] = opts.ApplicationID
	}
	if opts.ConversationID != "" {
		filter["conversationId"] = opts.ConversationID
	}
	if opts.AutonomousAgentID != "" {
		filter["autonomousAgentId"] = opts.AutonomousAgentID
	}
	if opts.ContextType != "" {
		filter["contextType"] = opts.ContextType
	}

	return filter
}

// buildFindOptions creates MongoDB find options from list options.
func (c *TracesCollection) buildFindOptions(opts *docdb.ListTracesOptions) *options.FindOptions {
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
