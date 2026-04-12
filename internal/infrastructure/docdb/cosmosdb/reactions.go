// Package cosmosdb provides the reactions collection implementation for CosmosDB.
package cosmosdb

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/data/azcosmos"

	"github.com/unifiedui/agent-service/internal/core/docdb"
	"github.com/unifiedui/agent-service/internal/domain/models"
)

// ReactionsCollection implements the docdb.ReactionsCollection interface for CosmosDB.
type ReactionsCollection struct {
	containerClient *azcosmos.ContainerClient
}

// NewReactionsCollection creates a new reactions collection wrapper.
func NewReactionsCollection(db *azcosmos.DatabaseClient) *ReactionsCollection {
	containerClient, _ := db.NewContainer(ReactionsContainerName)
	return &ReactionsCollection{
		containerClient: containerClient,
	}
}

// Upsert creates or updates a reaction (one per user per message).
func (c *ReactionsCollection) Upsert(ctx context.Context, reaction *models.MessageReaction) error {
	pk := azcosmos.NewPartitionKeyString(reaction.TenantID)

	// Check if reaction already exists
	existing, err := c.Get(ctx, &docdb.UpsertReactionOptions{
		TenantID:  reaction.TenantID,
		MessageID: reaction.MessageID,
		UserID:    reaction.UserID,
	})
	if err != nil {
		return err
	}

	now := time.Now().UTC()
	if existing != nil {
		// Update existing reaction
		existing.Reaction = reaction.Reaction
		existing.FeedbackText = reaction.FeedbackText
		existing.UpdatedAt = now

		doc := c.reactionToDoc(existing)
		data, err := json.Marshal(doc)
		if err != nil {
			return fmt.Errorf("failed to marshal reaction: %w", err)
		}

		_, err = c.containerClient.ReplaceItem(ctx, pk, existing.ID, data, nil)
		if err != nil {
			return fmt.Errorf("failed to update reaction: %w", err)
		}
	} else {
		// Create new reaction
		reaction.CreatedAt = now
		reaction.UpdatedAt = now

		doc := c.reactionToDoc(reaction)
		data, err := json.Marshal(doc)
		if err != nil {
			return fmt.Errorf("failed to marshal reaction: %w", err)
		}

		_, err = c.containerClient.CreateItem(ctx, pk, data, nil)
		if err != nil {
			return fmt.Errorf("failed to create reaction: %w", err)
		}
	}

	return nil
}

// Get retrieves a reaction by tenant, message, and user.
func (c *ReactionsCollection) Get(ctx context.Context, opts *docdb.UpsertReactionOptions) (*models.MessageReaction, error) {
	query := `SELECT * FROM c WHERE c.tenantId = @tenantId AND c.messageId = @messageId AND c.userId = @userId`
	params := []azcosmos.QueryParameter{
		{Name: "@tenantId", Value: sanitizeValue(opts.TenantID)},
		{Name: "@messageId", Value: sanitizeValue(opts.MessageID)},
		{Name: "@userId", Value: sanitizeValue(opts.UserID)},
	}

	pk := azcosmos.NewPartitionKeyString(opts.TenantID)
	pager := c.containerClient.NewQueryItemsPager(query, pk, &azcosmos.QueryOptions{
		QueryParameters: params,
	})

	if pager.More() {
		resp, err := pager.NextPage(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to get reaction: %w", err)
		}
		if len(resp.Items) > 0 {
			var reaction models.MessageReaction
			if err := json.Unmarshal(resp.Items[0], &reaction); err != nil {
				return nil, fmt.Errorf("failed to decode reaction: %w", err)
			}
			return &reaction, nil
		}
	}

	return nil, nil
}

// ListByMessage retrieves all reactions for a message.
func (c *ReactionsCollection) ListByMessage(ctx context.Context, opts *docdb.ListReactionsOptions) ([]*models.MessageReaction, error) {
	query := `SELECT * FROM c WHERE c.tenantId = @tenantId AND c.conversationId = @conversationId AND c.messageId = @messageId ORDER BY c.createdAt ASC`
	params := []azcosmos.QueryParameter{
		{Name: "@tenantId", Value: sanitizeValue(opts.TenantID)},
		{Name: "@conversationId", Value: sanitizeValue(opts.ConversationID)},
		{Name: "@messageId", Value: sanitizeValue(opts.MessageID)},
	}

	pk := azcosmos.NewPartitionKeyString(opts.TenantID)
	return c.queryReactions(ctx, query, params, pk)
}

// ListByMessages retrieves all reactions for multiple messages in a single query.
func (c *ReactionsCollection) ListByMessages(ctx context.Context, opts *docdb.ListBulkReactionsOptions) ([]*models.MessageReaction, error) {
	if len(opts.MessageIDs) == 0 {
		return []*models.MessageReaction{}, nil
	}

	// Build IN clause with parameters
	inParts := make([]string, 0, len(opts.MessageIDs))
	params := make([]azcosmos.QueryParameter, 0, len(opts.MessageIDs)+2)
	params = append(params, azcosmos.QueryParameter{Name: "@tenantId", Value: sanitizeValue(opts.TenantID)})
	params = append(params, azcosmos.QueryParameter{Name: "@conversationId", Value: sanitizeValue(opts.ConversationID)})

	for i, id := range opts.MessageIDs {
		paramName := fmt.Sprintf("@msgId%d", i)
		inParts = append(inParts, paramName)
		params = append(params, azcosmos.QueryParameter{Name: paramName, Value: sanitizeValue(id)})
	}

	query := fmt.Sprintf(`SELECT * FROM c WHERE c.tenantId = @tenantId AND c.conversationId = @conversationId AND c.messageId IN (%s) ORDER BY c.createdAt ASC`,
		joinConditions(inParts, ", "))

	pk := azcosmos.NewPartitionKeyString(opts.TenantID)
	return c.queryReactions(ctx, query, params, pk)
}

// Delete removes a user's reaction from a message.
func (c *ReactionsCollection) Delete(ctx context.Context, opts *docdb.DeleteReactionOptions) error {
	// Find the reaction first
	reaction, err := c.Get(ctx, &docdb.UpsertReactionOptions{
		TenantID:  opts.TenantID,
		MessageID: opts.MessageID,
		UserID:    opts.UserID,
	})
	if err != nil {
		return err
	}
	if reaction == nil {
		return nil
	}

	pk := azcosmos.NewPartitionKeyString(opts.TenantID)
	_, err = c.containerClient.DeleteItem(ctx, pk, reaction.ID, nil)
	if err != nil {
		var respErr *azcore.ResponseError
		if errors.As(err, &respErr) && respErr.StatusCode == http.StatusNotFound {
			return nil
		}
		return fmt.Errorf("failed to delete reaction: %w", err)
	}

	return nil
}

// DeleteByConversation removes all reactions in a conversation.
func (c *ReactionsCollection) DeleteByConversation(ctx context.Context, tenantID, conversationID string) error {
	query := `SELECT c.id FROM c WHERE c.tenantId = @tenantId AND c.conversationId = @conversationId`
	params := []azcosmos.QueryParameter{
		{Name: "@tenantId", Value: sanitizeValue(tenantID)},
		{Name: "@conversationId", Value: sanitizeValue(conversationID)},
	}

	pk := azcosmos.NewPartitionKeyString(tenantID)
	pager := c.containerClient.NewQueryItemsPager(query, pk, &azcosmos.QueryOptions{
		QueryParameters: params,
	})

	for pager.More() {
		resp, err := pager.NextPage(ctx)
		if err != nil {
			return fmt.Errorf("failed to query reactions for deletion: %w", err)
		}
		for _, item := range resp.Items {
			var doc struct {
				ID string `json:"id"`
			}
			if err := json.Unmarshal(item, &doc); err != nil {
				continue
			}
			_, _ = c.containerClient.DeleteItem(ctx, pk, doc.ID, nil)
		}
	}

	return nil
}

// EnsureIndexes for CosmosDB verifies container configuration.
// Note: CosmosDB indexes are managed via indexing policy at container creation time.
func (c *ReactionsCollection) EnsureIndexes(_ context.Context) error {
	// CosmosDB automatically indexes all properties by default
	// Unique constraints in CosmosDB require a unique key policy at container creation
	return nil
}

// reactionToDoc converts a reaction to a document with CosmosDB-compatible ID field.
func (c *ReactionsCollection) reactionToDoc(reaction *models.MessageReaction) map[string]interface{} {
	data, _ := json.Marshal(reaction)
	var doc map[string]interface{}
	_ = json.Unmarshal(data, &doc)

	// CosmosDB requires 'id' field, not '_id'
	if id, ok := doc["_id"]; ok {
		doc["id"] = id
		delete(doc, "_id")
	}

	return doc
}

// queryReactions executes a query and returns reactions.
func (c *ReactionsCollection) queryReactions(ctx context.Context, query string, params []azcosmos.QueryParameter, pk azcosmos.PartitionKey) ([]*models.MessageReaction, error) {
	pager := c.containerClient.NewQueryItemsPager(query, pk, &azcosmos.QueryOptions{
		QueryParameters: params,
	})

	var reactions []*models.MessageReaction
	for pager.More() {
		resp, err := pager.NextPage(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to query reactions: %w", err)
		}
		for _, item := range resp.Items {
			var reaction models.MessageReaction
			if err := json.Unmarshal(item, &reaction); err != nil {
				return nil, fmt.Errorf("failed to decode reaction: %w", err)
			}
			reactions = append(reactions, &reaction)
		}
	}

	return reactions, nil
}
