// Package cosmosdb provides the traces collection implementation for CosmosDB.
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

// TracesCollection implements the docdb.TracesCollection interface for CosmosDB.
type TracesCollection struct {
	containerClient *azcosmos.ContainerClient
}

// NewTracesCollection creates a new traces collection wrapper.
func NewTracesCollection(db *azcosmos.DatabaseClient) *TracesCollection {
	containerClient, _ := db.NewContainer(TracesContainerName)
	return &TracesCollection{
		containerClient: containerClient,
	}
}

// Create inserts a new trace.
func (c *TracesCollection) Create(ctx context.Context, trace *models.Trace) error {
	if trace.ID == "" {
		return fmt.Errorf("trace ID is required")
	}

	trace.CreatedAt = time.Now().UTC()
	trace.UpdatedAt = trace.CreatedAt

	doc := c.traceToDoc(trace)
	data, err := json.Marshal(doc)
	if err != nil {
		return fmt.Errorf("failed to marshal trace: %w", err)
	}

	pk := azcosmos.NewPartitionKeyString(trace.TenantID)
	_, err = c.containerClient.CreateItem(ctx, pk, data, nil)
	if err != nil {
		return fmt.Errorf("failed to insert trace: %w", err)
	}

	return nil
}

// Get retrieves a trace by ID.
func (c *TracesCollection) Get(ctx context.Context, id string) (*models.Trace, error) {
	query := "SELECT * FROM c WHERE c.id = @id"
	params := []azcosmos.QueryParameter{{Name: "@id", Value: sanitizeValue(id)}}

	pager := c.containerClient.NewQueryItemsPager(query, azcosmos.PartitionKey{}, &azcosmos.QueryOptions{
		QueryParameters: params,
	})

	if pager.More() {
		resp, err := pager.NextPage(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to get trace: %w", err)
		}
		if len(resp.Items) > 0 {
			var trace models.Trace
			if err := json.Unmarshal(resp.Items[0], &trace); err != nil {
				return nil, fmt.Errorf("failed to decode trace: %w", err)
			}
			return &trace, nil
		}
	}

	return nil, nil
}

// GetByConversation retrieves a trace by conversation ID (for internal use like conflict check).
func (c *TracesCollection) GetByConversation(ctx context.Context, tenantID, conversationID string) (*models.Trace, error) {
	query := `SELECT * FROM c WHERE c.tenantId = @tenantId AND c.conversationId = @conversationId AND c.contextType = @contextType`
	params := []azcosmos.QueryParameter{
		{Name: "@tenantId", Value: sanitizeValue(tenantID)},
		{Name: "@conversationId", Value: sanitizeValue(conversationID)},
		{Name: "@contextType", Value: string(models.TraceContextConversation)},
	}

	pk := azcosmos.NewPartitionKeyString(tenantID)
	pager := c.containerClient.NewQueryItemsPager(query, pk, &azcosmos.QueryOptions{
		QueryParameters: params,
	})

	if pager.More() {
		resp, err := pager.NextPage(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to get trace by conversation: %w", err)
		}
		if len(resp.Items) > 0 {
			var trace models.Trace
			if err := json.Unmarshal(resp.Items[0], &trace); err != nil {
				return nil, fmt.Errorf("failed to decode trace: %w", err)
			}
			return &trace, nil
		}
	}

	return nil, nil
}

// GetByReferenceID retrieves a trace by its external reference ID.
func (c *TracesCollection) GetByReferenceID(ctx context.Context, tenantID, referenceID string) (*models.Trace, error) {
	query := `SELECT * FROM c WHERE c.tenantId = @tenantId AND c.referenceId = @referenceId`
	params := []azcosmos.QueryParameter{
		{Name: "@tenantId", Value: sanitizeValue(tenantID)},
		{Name: "@referenceId", Value: sanitizeValue(referenceID)},
	}

	pk := azcosmos.NewPartitionKeyString(tenantID)
	pager := c.containerClient.NewQueryItemsPager(query, pk, &azcosmos.QueryOptions{
		QueryParameters: params,
	})

	if pager.More() {
		resp, err := pager.NextPage(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to get trace by reference ID: %w", err)
		}
		if len(resp.Items) > 0 {
			var trace models.Trace
			if err := json.Unmarshal(resp.Items[0], &trace); err != nil {
				return nil, fmt.Errorf("failed to decode trace: %w", err)
			}
			return &trace, nil
		}
	}

	return nil, nil
}

// ListByConversation retrieves traces for a conversation as a list.
func (c *TracesCollection) ListByConversation(ctx context.Context, tenantID, conversationID string) ([]*models.Trace, error) {
	query := `SELECT * FROM c WHERE c.tenantId = @tenantId AND c.conversationId = @conversationId AND c.contextType = @contextType ORDER BY c.createdAt DESC`
	params := []azcosmos.QueryParameter{
		{Name: "@tenantId", Value: sanitizeValue(tenantID)},
		{Name: "@conversationId", Value: sanitizeValue(conversationID)},
		{Name: "@contextType", Value: string(models.TraceContextConversation)},
	}

	pk := azcosmos.NewPartitionKeyString(tenantID)
	return c.queryTraces(ctx, query, params, pk)
}

// GetByWorkflow retrieves the most recent trace for a workflow.
func (c *TracesCollection) GetByWorkflow(ctx context.Context, tenantID, workflowID string) (*models.Trace, error) {
	query := `SELECT TOP 1 * FROM c WHERE c.tenantId = @tenantId AND c.workflowId = @workflowId AND c.contextType = @contextType ORDER BY c.createdAt DESC`
	params := []azcosmos.QueryParameter{
		{Name: "@tenantId", Value: sanitizeValue(tenantID)},
		{Name: "@workflowId", Value: sanitizeValue(workflowID)},
		{Name: "@contextType", Value: string(models.TraceContextWorkflow)},
	}

	pk := azcosmos.NewPartitionKeyString(tenantID)
	pager := c.containerClient.NewQueryItemsPager(query, pk, &azcosmos.QueryOptions{
		QueryParameters: params,
	})

	if pager.More() {
		resp, err := pager.NextPage(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to get trace by workflow: %w", err)
		}
		if len(resp.Items) > 0 {
			var trace models.Trace
			if err := json.Unmarshal(resp.Items[0], &trace); err != nil {
				return nil, fmt.Errorf("failed to decode trace: %w", err)
			}
			return &trace, nil
		}
	}

	return nil, nil
}

// ListByWorkflow retrieves traces for a workflow as a list.
func (c *TracesCollection) ListByWorkflow(ctx context.Context, tenantID, workflowID string) ([]*models.Trace, error) {
	query := `SELECT * FROM c WHERE c.tenantId = @tenantId AND c.workflowId = @workflowId AND c.contextType = @contextType ORDER BY c.createdAt DESC`
	params := []azcosmos.QueryParameter{
		{Name: "@tenantId", Value: sanitizeValue(tenantID)},
		{Name: "@workflowId", Value: sanitizeValue(workflowID)},
		{Name: "@contextType", Value: string(models.TraceContextWorkflow)},
	}

	pk := azcosmos.NewPartitionKeyString(tenantID)
	return c.queryTraces(ctx, query, params, pk)
}

// List retrieves traces with pagination and filtering.
func (c *TracesCollection) List(ctx context.Context, opts *docdb.ListTracesOptions) ([]*models.Trace, error) {
	query, params, pk := c.buildQuery(opts)
	return c.queryTraces(ctx, query, params, pk)
}

// Count returns the total number of traces matching the filter options.
func (c *TracesCollection) Count(ctx context.Context, opts *docdb.ListTracesOptions) (int64, error) {
	query, params, pk := c.buildCountQuery(opts)

	pager := c.containerClient.NewQueryItemsPager(query, pk, &azcosmos.QueryOptions{
		QueryParameters: params,
	})

	if pager.More() {
		resp, err := pager.NextPage(ctx)
		if err != nil {
			return 0, fmt.Errorf("failed to count traces: %w", err)
		}
		if len(resp.Items) > 0 {
			var result struct {
				Count int64 `json:"$1"`
			}
			if err := json.Unmarshal(resp.Items[0], &result); err != nil {
				return 0, err
			}
			return result.Count, nil
		}
	}

	return 0, nil
}

// Update replaces an existing trace completely.
func (c *TracesCollection) Update(ctx context.Context, trace *models.Trace) error {
	trace.UpdatedAt = time.Now().UTC()

	doc := c.traceToDoc(trace)
	data, err := json.Marshal(doc)
	if err != nil {
		return fmt.Errorf("failed to marshal trace: %w", err)
	}

	pk := azcosmos.NewPartitionKeyString(trace.TenantID)
	_, err = c.containerClient.ReplaceItem(ctx, pk, trace.ID, data, nil)
	if err != nil {
		var respErr *azcore.ResponseError
		if errors.As(err, &respErr) && respErr.StatusCode == http.StatusNotFound {
			return fmt.Errorf("trace not found: %s", trace.ID)
		}
		return fmt.Errorf("failed to update trace: %w", err)
	}

	return nil
}

// AddNodes appends nodes to an existing trace.
func (c *TracesCollection) AddNodes(ctx context.Context, id string, nodes []models.TraceNode) error {
	// Get existing trace
	trace, err := c.Get(ctx, id)
	if err != nil {
		return err
	}
	if trace == nil {
		return fmt.Errorf("trace not found: %s", id)
	}

	// Append nodes
	trace.Nodes = append(trace.Nodes, nodes...)
	trace.UpdatedAt = time.Now().UTC()

	return c.Update(ctx, trace)
}

// AddLogs appends logs to an existing trace.
func (c *TracesCollection) AddLogs(ctx context.Context, id string, logs []string) error {
	// Get existing trace
	trace, err := c.Get(ctx, id)
	if err != nil {
		return err
	}
	if trace == nil {
		return fmt.Errorf("trace not found: %s", id)
	}

	// Append logs
	trace.Logs = append(trace.Logs, logs...)
	trace.UpdatedAt = time.Now().UTC()

	return c.Update(ctx, trace)
}

// Delete removes a trace by ID.
func (c *TracesCollection) Delete(ctx context.Context, id string) error {
	// Get trace to find partition key
	trace, err := c.Get(ctx, id)
	if err != nil {
		return err
	}
	if trace == nil {
		return fmt.Errorf("trace not found: %s", id)
	}

	pk := azcosmos.NewPartitionKeyString(trace.TenantID)
	_, err = c.containerClient.DeleteItem(ctx, pk, id, nil)
	if err != nil {
		var respErr *azcore.ResponseError
		if errors.As(err, &respErr) && respErr.StatusCode == http.StatusNotFound {
			return fmt.Errorf("trace not found: %s", id)
		}
		return fmt.Errorf("failed to delete trace: %w", err)
	}

	return nil
}

// DeleteByConversation removes the trace for a conversation.
func (c *TracesCollection) DeleteByConversation(ctx context.Context, tenantID, conversationID string) error {
	trace, err := c.GetByConversation(ctx, tenantID, conversationID)
	if err != nil {
		return err
	}
	if trace == nil {
		return nil
	}

	return c.Delete(ctx, trace.ID)
}

// DeleteByWorkflow removes the trace for a workflow.
func (c *TracesCollection) DeleteByWorkflow(ctx context.Context, tenantID, workflowID string) error {
	traces, err := c.ListByWorkflow(ctx, tenantID, workflowID)
	if err != nil {
		return err
	}

	for _, trace := range traces {
		if err := c.Delete(ctx, trace.ID); err != nil {
			return err
		}
	}

	return nil
}

// EnsureIndexes for CosmosDB verifies container configuration.
// Note: CosmosDB indexes are managed via indexing policy at container creation time.
// This method exists for interface compatibility.
func (c *TracesCollection) EnsureIndexes(_ context.Context) error {
	// CosmosDB automatically indexes all properties by default
	// Custom indexing policies should be configured via Terraform
	return nil
}

// traceToDoc converts a trace to a document with CosmosDB-compatible ID field.
func (c *TracesCollection) traceToDoc(trace *models.Trace) map[string]interface{} {
	data, _ := json.Marshal(trace)
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
func (c *TracesCollection) buildQuery(opts *docdb.ListTracesOptions) (string, []azcosmos.QueryParameter, azcosmos.PartitionKey) {
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
		if opts.ChatAgentID != "" {
			addParam("chatAgentId", sanitizeValue(opts.ChatAgentID))
		}
		if opts.ConversationID != "" {
			addParam("conversationId", sanitizeValue(opts.ConversationID))
		}
		if opts.WorkflowID != "" {
			addParam("workflowId", sanitizeValue(opts.WorkflowID))
		}
		if opts.ContextType != "" {
			addParam("contextType", string(opts.ContextType))
		}
	}

	// Build select clause with optional projection
	selectClause := "SELECT * FROM c"
	if opts != nil && !opts.Expand {
		// Exclude nodes and logs for compact response
		// CosmosDB doesn't have native projection exclusion, so we select specific fields
		selectClause = `SELECT c.id, c.tenantId, c.chatAgentId, c.conversationId, c.workflowId,
			c.contextType, c.referenceId, c.status, c.startTime, c.endTime,
			c.durationMs, c.tokenUsage, c.metadata, c.createdAt, c.updatedAt FROM c`
	}

	query := selectClause
	if len(conditions) > 0 {
		query += " WHERE " + joinConditions(conditions, " AND ")
	}

	// Add ORDER BY
	sortField := "createdAt"
	if opts != nil && opts.SortBy == docdb.SortFieldUpdatedAt {
		sortField = "updatedAt"
	}
	sortDir := "DESC"
	if opts != nil && opts.OrderBy == docdb.SortOrderAsc {
		sortDir = "ASC"
	}
	query += fmt.Sprintf(" ORDER BY c.%s %s", sortField, sortDir)

	// Add OFFSET/LIMIT
	if opts != nil {
		if opts.Skip > 0 || opts.Limit > 0 {
			query += fmt.Sprintf(" OFFSET %d LIMIT %d", opts.Skip, opts.Limit)
		}
	}

	return query, params, pk
}

// buildCountQuery builds a count query from list options.
func (c *TracesCollection) buildCountQuery(opts *docdb.ListTracesOptions) (string, []azcosmos.QueryParameter, azcosmos.PartitionKey) {
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
		if opts.ChatAgentID != "" {
			addParam("chatAgentId", sanitizeValue(opts.ChatAgentID))
		}
		if opts.ConversationID != "" {
			addParam("conversationId", sanitizeValue(opts.ConversationID))
		}
		if opts.WorkflowID != "" {
			addParam("workflowId", sanitizeValue(opts.WorkflowID))
		}
		if opts.ContextType != "" {
			addParam("contextType", string(opts.ContextType))
		}
	}

	query := "SELECT VALUE COUNT(1) FROM c"
	if len(conditions) > 0 {
		query += " WHERE " + joinConditions(conditions, " AND ")
	}

	return query, params, pk
}

// queryTraces executes a query and returns traces.
func (c *TracesCollection) queryTraces(ctx context.Context, query string, params []azcosmos.QueryParameter, pk azcosmos.PartitionKey) ([]*models.Trace, error) {
	pager := c.containerClient.NewQueryItemsPager(query, pk, &azcosmos.QueryOptions{
		QueryParameters: params,
	})

	var traces []*models.Trace
	for pager.More() {
		resp, err := pager.NextPage(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to query traces: %w", err)
		}
		for _, item := range resp.Items {
			var trace models.Trace
			if err := json.Unmarshal(item, &trace); err != nil {
				return nil, fmt.Errorf("failed to decode trace: %w", err)
			}
			traces = append(traces, &trace)
		}
	}

	return traces, nil
}
