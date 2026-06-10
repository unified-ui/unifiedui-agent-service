// Package cosmosdb provides the messages collection implementation for CosmosDB.
package cosmosdb

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/data/azcosmos"

	"github.com/unifiedui/agent-service/internal/core/docdb"
	"github.com/unifiedui/agent-service/internal/domain/models"
)

// MessagesCollection implements the docdb.MessagesCollection interface for CosmosDB.
type MessagesCollection struct {
	containerClient *azcosmos.ContainerClient
}

// NewMessagesCollection creates a new messages collection wrapper.
func NewMessagesCollection(db *azcosmos.DatabaseClient) *MessagesCollection {
	containerClient, _ := db.NewContainer(MessagesContainerName)
	return &MessagesCollection{
		containerClient: containerClient,
	}
}

// Add inserts a new message (user or assistant).
func (c *MessagesCollection) Add(ctx context.Context, message *models.Message) error {
	if message.ID == "" {
		return fmt.Errorf("message ID is required")
	}

	message.CreatedAt = time.Now().UTC()
	message.UpdatedAt = message.CreatedAt

	doc := c.messageToDoc(message)
	data, err := json.Marshal(doc)
	if err != nil {
		return fmt.Errorf("failed to marshal message: %w", err)
	}

	pk := azcosmos.NewPartitionKeyString(message.TenantID)
	_, err = c.containerClient.CreateItem(ctx, pk, data, nil)
	if err != nil {
		return fmt.Errorf("failed to insert message: %w", err)
	}

	return nil
}

// Get retrieves a message by ID.
func (c *MessagesCollection) Get(ctx context.Context, id string) (*models.Message, error) {
	query := "SELECT * FROM c WHERE c.id = @id"
	params := []azcosmos.QueryParameter{{Name: "@id", Value: sanitizeValue(id)}}

	pager := c.containerClient.NewQueryItemsPager(query, azcosmos.PartitionKey{}, &azcosmos.QueryOptions{
		QueryParameters: params,
	})

	if pager.More() {
		resp, err := pager.NextPage(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to get message: %w", err)
		}
		if len(resp.Items) > 0 {
			var message models.Message
			if err := json.Unmarshal(resp.Items[0], &message); err != nil {
				return nil, fmt.Errorf("failed to decode message: %w", err)
			}
			return &message, nil
		}
	}

	return nil, nil
}

// GetByUserMessageID retrieves assistant message by user message ID.
func (c *MessagesCollection) GetByUserMessageID(ctx context.Context, userMessageID string) (*models.Message, error) {
	query := `SELECT * FROM c WHERE c.userMessageId = @userMessageId AND c.type = @type`
	params := []azcosmos.QueryParameter{
		{Name: "@userMessageId", Value: sanitizeValue(userMessageID)},
		{Name: "@type", Value: string(models.MessageTypeAssistant)},
	}

	pager := c.containerClient.NewQueryItemsPager(query, azcosmos.PartitionKey{}, &azcosmos.QueryOptions{
		QueryParameters: params,
	})

	if pager.More() {
		resp, err := pager.NextPage(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to get assistant message by user message ID: %w", err)
		}
		if len(resp.Items) > 0 {
			var message models.Message
			if err := json.Unmarshal(resp.Items[0], &message); err != nil {
				return nil, fmt.Errorf("failed to decode message: %w", err)
			}
			return &message, nil
		}
	}

	return nil, nil
}

// List retrieves messages with pagination and sorting.
func (c *MessagesCollection) List(ctx context.Context, opts *docdb.ListMessagesOptions) ([]*models.Message, error) {
	query, params, pk := c.buildQuery(opts)
	return c.queryMessages(ctx, query, params, pk)
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

// Search searches messages by content text.
func (c *MessagesCollection) Search(ctx context.Context, opts *docdb.SearchMessagesOptions) ([]*models.Message, error) {
	// CosmosDB uses CONTAINS for case-insensitive search
	query := `SELECT * FROM c WHERE c.tenantId = @tenantId AND CONTAINS(LOWER(c.content), LOWER(@query)) ORDER BY c.createdAt DESC`
	params := []azcosmos.QueryParameter{
		{Name: "@tenantId", Value: sanitizeValue(opts.TenantID)},
		{Name: "@query", Value: strings.ToLower(opts.Query)},
	}

	if opts.Limit > 0 {
		query += fmt.Sprintf(" OFFSET %d LIMIT %d", opts.Skip, opts.Limit)
	}

	pk := azcosmos.NewPartitionKeyString(opts.TenantID)
	return c.queryMessages(ctx, query, params, pk)
}

// Update updates an existing message.
func (c *MessagesCollection) Update(ctx context.Context, message *models.Message) error {
	message.UpdatedAt = time.Now().UTC()

	doc := c.messageToDoc(message)
	data, err := json.Marshal(doc)
	if err != nil {
		return fmt.Errorf("failed to marshal message: %w", err)
	}

	pk := azcosmos.NewPartitionKeyString(message.TenantID)
	_, err = c.containerClient.ReplaceItem(ctx, pk, message.ID, data, nil)
	if err != nil {
		var respErr *azcore.ResponseError
		if errors.As(err, &respErr) && respErr.StatusCode == http.StatusNotFound {
			return fmt.Errorf("message not found: %s", message.ID)
		}
		return fmt.Errorf("failed to update message: %w", err)
	}

	return nil
}

// Delete removes a message or all messages in a conversation.
func (c *MessagesCollection) Delete(ctx context.Context, opts *docdb.DeleteMessagesOptions) (int64, error) {
	if opts.MessageID != "" {
		// Delete specific message
		msg, err := c.Get(ctx, opts.MessageID)
		if err != nil {
			return 0, err
		}
		if msg == nil || msg.TenantID != opts.TenantID {
			return 0, nil
		}

		pk := azcosmos.NewPartitionKeyString(opts.TenantID)
		_, err = c.containerClient.DeleteItem(ctx, pk, opts.MessageID, nil)
		if err != nil {
			var respErr *azcore.ResponseError
			if errors.As(err, &respErr) && respErr.StatusCode == http.StatusNotFound {
				return 0, nil
			}
			return 0, fmt.Errorf("failed to delete message: %w", err)
		}
		return 1, nil
	}

	if opts.ConversationID != "" {
		// Delete all messages in conversation
		query := `SELECT c.id FROM c WHERE c.conversationId = @conversationId AND c.tenantId = @tenantId`
		params := []azcosmos.QueryParameter{
			{Name: "@conversationId", Value: sanitizeValue(opts.ConversationID)},
			{Name: "@tenantId", Value: sanitizeValue(opts.TenantID)},
		}

		pk := azcosmos.NewPartitionKeyString(opts.TenantID)
		pager := c.containerClient.NewQueryItemsPager(query, pk, &azcosmos.QueryOptions{
			QueryParameters: params,
		})

		var deleted int64
		for pager.More() {
			resp, err := pager.NextPage(ctx)
			if err != nil {
				return deleted, fmt.Errorf("failed to query messages for deletion: %w", err)
			}
			for _, item := range resp.Items {
				var doc struct {
					ID string `json:"id"`
				}
				if err := json.Unmarshal(item, &doc); err != nil {
					continue
				}
				_, err := c.containerClient.DeleteItem(ctx, pk, doc.ID, nil)
				if err == nil {
					deleted++
				}
			}
		}
		return deleted, nil
	}

	return 0, nil
}

// CountByConversation returns the count of messages in a conversation.
func (c *MessagesCollection) CountByConversation(ctx context.Context, tenantID, conversationID string) (int64, error) {
	query := `SELECT VALUE COUNT(1) FROM c WHERE c.conversationId = @conversationId AND c.tenantId = @tenantId`
	params := []azcosmos.QueryParameter{
		{Name: "@conversationId", Value: sanitizeValue(conversationID)},
		{Name: "@tenantId", Value: sanitizeValue(tenantID)},
	}

	pk := azcosmos.NewPartitionKeyString(tenantID)
	pager := c.containerClient.NewQueryItemsPager(query, pk, &azcosmos.QueryOptions{
		QueryParameters: params,
	})

	if pager.More() {
		resp, err := pager.NextPage(ctx)
		if err != nil {
			return 0, fmt.Errorf("failed to count messages: %w", err)
		}
		if len(resp.Items) > 0 {
			var count int64
			if err := json.Unmarshal(resp.Items[0], &count); err != nil {
				return 0, err
			}
			return count, nil
		}
	}

	return 0, nil
}

// GetMessageStats returns aggregated message counts by status for a tenant,
// grouped by chat agent. Uses a single GROUP BY query.
func (c *MessagesCollection) GetMessageStats(ctx context.Context, tenantID string, filter *models.MessageStatsFilter) (*models.MessageStatsResult, error) {
	whereClause := "WHERE c.tenantId = @tenantId AND c.type = 'assistant' AND c.status IN ('success', 'failed')"
	params := []azcosmos.QueryParameter{
		{Name: "@tenantId", Value: sanitizeValue(tenantID)},
	}

	if filter != nil {
		if len(filter.ChatAgentIDs) > 0 {
			ids := make([]interface{}, 0, len(filter.ChatAgentIDs))
			for _, id := range filter.ChatAgentIDs {
				ids = append(ids, sanitizeValue(id))
			}
			whereClause += " AND ARRAY_CONTAINS(@agentIds, c.chatAgentId)"
			params = append(params, azcosmos.QueryParameter{Name: "@agentIds", Value: ids})
		}
		if !filter.From.IsZero() {
			whereClause += " AND c.createdAt >= @from"
			params = append(params, azcosmos.QueryParameter{Name: "@from", Value: filter.From.Format(time.RFC3339)})
		}
		if !filter.To.IsZero() {
			whereClause += " AND c.createdAt <= @to"
			params = append(params, azcosmos.QueryParameter{Name: "@to", Value: filter.To.Format(time.RFC3339)})
		}
	}

	query := fmt.Sprintf(
		"SELECT c.chatAgentId AS agentId, COUNT(1) AS total, SUM(IIF(c.status = 'failed', 1, 0)) AS failed FROM c %s GROUP BY c.chatAgentId",
		whereClause,
	)

	pk := azcosmos.NewPartitionKeyString(tenantID)
	pager := c.containerClient.NewQueryItemsPager(query, pk, &azcosmos.QueryOptions{
		QueryParameters: params,
	})

	type groupRow struct {
		AgentID string `json:"agentId"`
		Total   int64  `json:"total"`
		Failed  int64  `json:"failed"`
	}

	result := &models.MessageStatsResult{
		PerAgent: make([]models.MessageStatsPerAgent, 0),
	}

	for pager.More() {
		resp, err := pager.NextPage(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to query message stats: %w", err)
		}
		for _, item := range resp.Items {
			var row groupRow
			if err := json.Unmarshal(item, &row); err != nil {
				return nil, fmt.Errorf("failed to decode message stats row: %w", err)
			}
			success := row.Total - row.Failed
			result.Aggregate.TotalMessages += row.Total
			result.Aggregate.SuccessCount += success
			result.Aggregate.FailedCount += row.Failed

			if row.AgentID == "" {
				continue
			}
			result.PerAgent = append(result.PerAgent, models.MessageStatsPerAgent{
				ChatAgentID:   row.AgentID,
				TotalMessages: row.Total,
				SuccessCount:  success,
				FailedCount:   row.Failed,
			})
		}
	}

	return result, nil
}

// EnsureIndexes for CosmosDB verifies container configuration.
// Note: CosmosDB indexes are managed via indexing policy at container creation time.
func (c *MessagesCollection) EnsureIndexes(_ context.Context) error {
	// CosmosDB automatically indexes all properties by default
	return nil
}

// messageToDoc converts a message to a document with CosmosDB-compatible ID field.
func (c *MessagesCollection) messageToDoc(message *models.Message) map[string]interface{} {
	data, _ := json.Marshal(message)
	var doc map[string]interface{}
	_ = json.Unmarshal(data, &doc)

	// CosmosDB requires 'id' field, not '_id'
	if id, ok := doc["_id"]; ok {
		doc["id"] = id
		delete(doc, "_id")
	}

	return doc
}

// buildQuery builds a query from list options.
func (c *MessagesCollection) buildQuery(opts *docdb.ListMessagesOptions) (string, []azcosmos.QueryParameter, azcosmos.PartitionKey) {
	var conditions []string
	var params []azcosmos.QueryParameter
	var pk azcosmos.PartitionKey
	paramIdx := 0

	addParam := func(name, value string) {
		paramName := fmt.Sprintf("@p%d", paramIdx)
		paramIdx++
		conditions = append(conditions, fmt.Sprintf("c.%s = %s", name, paramName))
		params = append(params, azcosmos.QueryParameter{Name: paramName, Value: value})
	}

	if opts != nil {
		if opts.TenantID != "" {
			addParam("tenantId", sanitizeValue(opts.TenantID))
			pk = azcosmos.NewPartitionKeyString(opts.TenantID)
		}
		if opts.ConversationID != "" {
			addParam("conversationId", sanitizeValue(opts.ConversationID))
		}
		if opts.Type != "" {
			addParam("type", string(opts.Type))
		}
	}

	query := "SELECT * FROM c"
	if len(conditions) > 0 {
		query += " WHERE " + joinConditions(conditions, " AND ")
	}

	// Add ORDER BY
	sortDir := "DESC"
	if opts != nil && opts.OrderBy == docdb.SortOrderAsc {
		sortDir = "ASC"
	}
	query += fmt.Sprintf(" ORDER BY c.createdAt %s", sortDir)

	// Add OFFSET/LIMIT
	if opts != nil {
		if opts.Skip > 0 || opts.Limit > 0 {
			query += fmt.Sprintf(" OFFSET %d LIMIT %d", opts.Skip, opts.Limit)
		}
	}

	return query, params, pk
}

// queryMessages executes a query and returns messages.
func (c *MessagesCollection) queryMessages(ctx context.Context, query string, params []azcosmos.QueryParameter, pk azcosmos.PartitionKey) ([]*models.Message, error) {
	pager := c.containerClient.NewQueryItemsPager(query, pk, &azcosmos.QueryOptions{
		QueryParameters: params,
	})

	var messages []*models.Message
	for pager.More() {
		resp, err := pager.NextPage(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to query messages: %w", err)
		}
		for _, item := range resp.Items {
			var message models.Message
			if err := json.Unmarshal(item, &message); err != nil {
				return nil, fmt.Errorf("failed to decode message: %w", err)
			}
			messages = append(messages, &message)
		}
	}

	return messages, nil
}
