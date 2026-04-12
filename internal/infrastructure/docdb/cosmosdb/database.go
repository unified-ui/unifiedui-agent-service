// Package cosmosdb provides Azure CosmosDB NoSQL database implementation.
package cosmosdb

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/runtime"
	"github.com/Azure/azure-sdk-for-go/sdk/data/azcosmos"

	"github.com/unifiedui/agent-service/internal/core/docdb"
)

const (
	// MessagesContainerName is the name of the messages container.
	MessagesContainerName = "messages"
	// TracesContainerName is the name of the traces container.
	TracesContainerName = "traces"
	// ReactionsContainerName is the name of the reactions container.
	ReactionsContainerName = "reactions"
	// SessionsContainerName is the name of the sessions container.
	SessionsContainerName = "sessions"

	// DefaultPartitionKeyPath is the partition key path for tenant isolation.
	DefaultPartitionKeyPath = "/tenantId"
)

// Database implements the docdb.Database interface for CosmosDB.
type Database struct {
	cosmosClient   *azcosmos.Client
	databaseClient *azcosmos.DatabaseClient
	databaseName   string
	containers     map[string]*Collection
}

// NewDatabase creates a new database wrapper.
func NewDatabase(cosmosClient *azcosmos.Client, databaseClient *azcosmos.DatabaseClient, databaseName string) *Database {
	return &Database{
		cosmosClient:   cosmosClient,
		databaseClient: databaseClient,
		databaseName:   databaseName,
		containers:     make(map[string]*Collection),
	}
}

// Collection returns a collection by name.
func (d *Database) Collection(name string) docdb.Collection {
	if col, ok := d.containers[name]; ok {
		return col
	}

	containerClient, err := d.databaseClient.NewContainer(name)
	if err != nil {
		// Return a collection that will error on operations
		return &Collection{containerName: name, err: err}
	}

	col := NewCollection(containerClient, name)
	d.containers[name] = col
	return col
}

// ListCollectionNames lists all container names.
func (d *Database) ListCollectionNames(ctx context.Context) ([]string, error) {
	pager := d.databaseClient.NewQueryContainersPager("SELECT * FROM c", nil)

	var names []string
	for pager.More() {
		resp, err := pager.NextPage(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to list containers: %w", err)
		}
		for _, container := range resp.Containers {
			names = append(names, container.ID)
		}
	}

	return names, nil
}

// Collection implements the docdb.Collection interface for CosmosDB.
type Collection struct {
	containerClient *azcosmos.ContainerClient
	containerName   string
	err             error
}

// NewCollection creates a new collection wrapper.
func NewCollection(containerClient *azcosmos.ContainerClient, containerName string) *Collection {
	return &Collection{
		containerClient: containerClient,
		containerName:   containerName,
	}
}

// InsertOne inserts a single document.
func (c *Collection) InsertOne(ctx context.Context, document interface{}) (interface{}, error) {
	if c.err != nil {
		return nil, c.err
	}

	doc, partitionKey, err := c.prepareDocument(document)
	if err != nil {
		return nil, err
	}

	data, err := json.Marshal(doc)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal document: %w", err)
	}

	resp, err := c.containerClient.CreateItem(ctx, partitionKey, data, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to insert document: %w", err)
	}

	// Return the id from the response, fallback to document id
	var result map[string]interface{}
	if json.Unmarshal(resp.Value, &result) == nil {
		if id, ok := result["id"]; ok {
			return id, nil
		}
	}
	return doc["id"], nil
}

// InsertMany inserts multiple documents.
func (c *Collection) InsertMany(ctx context.Context, documents []interface{}) ([]interface{}, error) {
	if c.err != nil {
		return nil, c.err
	}

	ids := make([]interface{}, 0, len(documents))
	for _, document := range documents {
		id, err := c.InsertOne(ctx, document)
		if err != nil {
			return ids, err
		}
		ids = append(ids, id)
	}
	return ids, nil
}

// FindOne finds a single document.
func (c *Collection) FindOne(ctx context.Context, filter interface{}) docdb.SingleResult {
	if c.err != nil {
		return &SingleResult{err: c.err}
	}

	// Convert filter to query
	query, params, partitionKey := c.filterToQuery(filter)

	pager := c.containerClient.NewQueryItemsPager(query, partitionKey, &azcosmos.QueryOptions{
		QueryParameters: params,
	})

	if pager.More() {
		resp, err := pager.NextPage(ctx)
		if err != nil {
			return &SingleResult{err: err}
		}
		if len(resp.Items) > 0 {
			return &SingleResult{data: resp.Items[0]}
		}
	}

	return &SingleResult{err: ErrNotFound}
}

// Find finds multiple documents.
func (c *Collection) Find(ctx context.Context, filter interface{}, opts *docdb.FindOptions) (docdb.Cursor, error) {
	if c.err != nil {
		return nil, c.err
	}

	query, params, partitionKey := c.filterToQuery(filter)

	// Add ORDER BY, OFFSET, and LIMIT if specified
	if opts != nil {
		if opts.Sort != nil {
			query = c.addOrderBy(query, opts.Sort)
		}
		if opts.Skip > 0 || opts.Limit > 0 {
			query = fmt.Sprintf("%s OFFSET %d LIMIT %d", query, opts.Skip, opts.Limit)
		}
	}

	pager := c.containerClient.NewQueryItemsPager(query, partitionKey, &azcosmos.QueryOptions{
		QueryParameters: params,
	})

	return &Cursor{pager: pager, ctx: ctx}, nil
}

// UpdateOne updates a single document.
func (c *Collection) UpdateOne(ctx context.Context, filter interface{}, update interface{}) (*docdb.UpdateResult, error) {
	if c.err != nil {
		return nil, c.err
	}

	// First find the document
	result := c.FindOne(ctx, filter)
	var existing map[string]interface{}
	if err := result.Decode(&existing); err != nil {
		if errors.Is(err, ErrNotFound) {
			return &docdb.UpdateResult{MatchedCount: 0, ModifiedCount: 0}, nil
		}
		return nil, err
	}

	// Apply the update
	updated, err := c.applyUpdate(existing, update)
	if err != nil {
		return nil, err
	}

	// Get ID and partition key
	id, ok := updated["id"].(string)
	if !ok {
		id, ok = updated["_id"].(string)
		if !ok {
			return nil, fmt.Errorf("document must have an id field")
		}
		// Normalize to id
		updated["id"] = id
		delete(updated, "_id")
	}

	tenantID, _ := updated["tenantId"].(string)
	pk := azcosmos.NewPartitionKeyString(tenantID)

	data, err := json.Marshal(updated)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal updated document: %w", err)
	}

	_, err = c.containerClient.ReplaceItem(ctx, pk, id, data, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to update document: %w", err)
	}

	return &docdb.UpdateResult{MatchedCount: 1, ModifiedCount: 1}, nil
}

// UpdateMany updates multiple documents.
func (c *Collection) UpdateMany(ctx context.Context, filter interface{}, update interface{}) (*docdb.UpdateResult, error) {
	if c.err != nil {
		return nil, c.err
	}

	// Find all matching documents
	cursor, err := c.Find(ctx, filter, nil)
	if err != nil {
		return nil, err
	}
	defer func() { _ = cursor.Close(ctx) }()

	var matched, modified int64
	for cursor.Next(ctx) {
		var doc map[string]interface{}
		if err := cursor.Decode(&doc); err != nil {
			continue
		}
		matched++

		updated, err := c.applyUpdate(doc, update)
		if err != nil {
			continue
		}

		id, _ := updated["id"].(string)
		if id == "" {
			id, _ = updated["_id"].(string)
		}
		tenantID, _ := updated["tenantId"].(string)
		pk := azcosmos.NewPartitionKeyString(tenantID)

		data, err := json.Marshal(updated)
		if err != nil {
			continue
		}

		_, err = c.containerClient.ReplaceItem(ctx, pk, id, data, nil)
		if err == nil {
			modified++
		}
	}

	return &docdb.UpdateResult{MatchedCount: matched, ModifiedCount: modified}, nil
}

// DeleteOne deletes a single document.
func (c *Collection) DeleteOne(ctx context.Context, filter interface{}) (*docdb.DeleteResult, error) {
	if c.err != nil {
		return nil, c.err
	}

	// Find the document first
	result := c.FindOne(ctx, filter)
	var doc map[string]interface{}
	if err := result.Decode(&doc); err != nil {
		if errors.Is(err, ErrNotFound) {
			return &docdb.DeleteResult{DeletedCount: 0}, nil
		}
		return nil, err
	}

	id, ok := doc["id"].(string)
	if !ok {
		id, ok = doc["_id"].(string)
	}
	if !ok {
		return nil, fmt.Errorf("document must have an id field")
	}

	tenantID, _ := doc["tenantId"].(string)
	pk := azcosmos.NewPartitionKeyString(tenantID)

	_, err := c.containerClient.DeleteItem(ctx, pk, id, nil)
	if err != nil {
		var respErr *azcore.ResponseError
		if errors.As(err, &respErr) && respErr.StatusCode == http.StatusNotFound {
			return &docdb.DeleteResult{DeletedCount: 0}, nil
		}
		return nil, fmt.Errorf("failed to delete document: %w", err)
	}

	return &docdb.DeleteResult{DeletedCount: 1}, nil
}

// DeleteMany deletes multiple documents.
func (c *Collection) DeleteMany(ctx context.Context, filter interface{}) (*docdb.DeleteResult, error) {
	if c.err != nil {
		return nil, c.err
	}

	// Find all matching documents
	cursor, err := c.Find(ctx, filter, nil)
	if err != nil {
		return nil, err
	}
	defer func() { _ = cursor.Close(ctx) }()

	var deleted int64
	for cursor.Next(ctx) {
		var doc map[string]interface{}
		if err := cursor.Decode(&doc); err != nil {
			continue
		}

		id, _ := doc["id"].(string)
		if id == "" {
			id, _ = doc["_id"].(string)
		}
		tenantID, _ := doc["tenantId"].(string)
		pk := azcosmos.NewPartitionKeyString(tenantID)

		_, err := c.containerClient.DeleteItem(ctx, pk, id, nil)
		if err == nil {
			deleted++
		}
	}

	return &docdb.DeleteResult{DeletedCount: deleted}, nil
}

// CountDocuments counts documents matching the filter.
func (c *Collection) CountDocuments(ctx context.Context, filter interface{}) (int64, error) {
	if c.err != nil {
		return 0, c.err
	}

	query, params, partitionKey := c.filterToCountQuery(filter)

	pager := c.containerClient.NewQueryItemsPager(query, partitionKey, &azcosmos.QueryOptions{
		QueryParameters: params,
	})

	if pager.More() {
		resp, err := pager.NextPage(ctx)
		if err != nil {
			return 0, fmt.Errorf("failed to count documents: %w", err)
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

// prepareDocument prepares a document for insertion.
func (c *Collection) prepareDocument(document interface{}) (map[string]interface{}, azcosmos.PartitionKey, error) {
	// Convert to map
	data, err := json.Marshal(document)
	if err != nil {
		return nil, azcosmos.PartitionKey{}, fmt.Errorf("failed to marshal document: %w", err)
	}

	var doc map[string]interface{}
	if err := json.Unmarshal(data, &doc); err != nil {
		return nil, azcosmos.PartitionKey{}, fmt.Errorf("failed to unmarshal document: %w", err)
	}

	// CosmosDB requires 'id' field, not '_id'
	if id, ok := doc["_id"]; ok {
		doc["id"] = id
		delete(doc, "_id")
	}

	// Get partition key value (tenantId)
	tenantID, _ := doc["tenantId"].(string)
	pk := azcosmos.NewPartitionKeyString(tenantID)

	return doc, pk, nil
}

// filterToQuery converts a MongoDB-style filter to a SQL query.
func (c *Collection) filterToQuery(filter interface{}) (string, []azcosmos.QueryParameter, azcosmos.PartitionKey) {
	filterMap, ok := filter.(map[string]interface{})
	if !ok {
		return "SELECT * FROM c", nil, azcosmos.PartitionKey{}
	}

	conditions := make([]string, 0, len(filterMap))
	params := make([]azcosmos.QueryParameter, 0, len(filterMap))
	var partitionKey azcosmos.PartitionKey
	paramIndex := 0

	for key, value := range filterMap {
		// Normalize _id to id
		if key == "_id" {
			key = "id"
		}

		paramName := fmt.Sprintf("@p%d", paramIndex)
		paramIndex++

		conditions = append(conditions, fmt.Sprintf("c.%s = %s", key, paramName))
		params = append(params, azcosmos.QueryParameter{Name: paramName, Value: value})

		// Extract partition key
		if key == "tenantId" {
			if strVal, ok := value.(string); ok {
				partitionKey = azcosmos.NewPartitionKeyString(strVal)
			}
		}
	}

	query := "SELECT * FROM c"
	if len(conditions) > 0 {
		query = fmt.Sprintf("SELECT * FROM c WHERE %s", joinConditions(conditions, " AND "))
	}

	return query, params, partitionKey
}

// filterToCountQuery converts a filter to a COUNT query.
func (c *Collection) filterToCountQuery(filter interface{}) (string, []azcosmos.QueryParameter, azcosmos.PartitionKey) {
	query, params, pk := c.filterToQuery(filter)
	// Replace SELECT * with SELECT VALUE COUNT(1)
	if len(query) >= 15 && query[:15] == "SELECT * FROM c" {
		query = "SELECT VALUE COUNT(1) FROM c" + query[15:]
	}
	return query, params, pk
}

// addOrderBy adds ORDER BY clause to query.
func (c *Collection) addOrderBy(query string, sort interface{}) string {
	sortMap, ok := sort.(map[string]interface{})
	if !ok {
		return query
	}

	for field, direction := range sortMap {
		dir := "ASC"
		if d, ok := direction.(int); ok && d < 0 {
			dir = "DESC"
		}
		if d, ok := direction.(float64); ok && d < 0 {
			dir = "DESC"
		}
		query = fmt.Sprintf("%s ORDER BY c.%s %s", query, field, dir)
		break // Only use first sort field
	}

	return query
}

// applyUpdate applies MongoDB-style update operators to a document.
func (c *Collection) applyUpdate(doc map[string]interface{}, update interface{}) (map[string]interface{}, error) {
	updateMap, ok := update.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("update must be a map")
	}

	// Handle $set operator
	if setOp, ok := updateMap["$set"]; ok {
		if setMap, ok := setOp.(map[string]interface{}); ok {
			for key, value := range setMap {
				doc[key] = value
			}
		}
	}

	// Handle $push operator
	if pushOp, ok := updateMap["$push"]; ok {
		if pushMap, ok := pushOp.(map[string]interface{}); ok {
			for key, value := range pushMap {
				// Handle $each modifier
				if valueMap, ok := value.(map[string]interface{}); ok {
					if each, ok := valueMap["$each"].([]interface{}); ok {
						existing, _ := doc[key].([]interface{})
						doc[key] = append(existing, each...)
						continue
					}
				}
				existing, _ := doc[key].([]interface{})
				doc[key] = append(existing, value)
			}
		}
	}

	// Handle $unset operator
	if unsetOp, ok := updateMap["$unset"]; ok {
		if unsetMap, ok := unsetOp.(map[string]interface{}); ok {
			for key := range unsetMap {
				delete(doc, key)
			}
		}
	}

	// Handle $inc operator
	if incOp, ok := updateMap["$inc"]; ok {
		if incMap, ok := incOp.(map[string]interface{}); ok {
			for key, value := range incMap {
				existing, _ := doc[key].(float64)
				if incVal, ok := value.(float64); ok {
					doc[key] = existing + incVal
				} else if incVal, ok := value.(int); ok {
					doc[key] = existing + float64(incVal)
				}
			}
		}
	}

	return doc, nil
}

// joinConditions joins conditions with a separator.
func joinConditions(conditions []string, sep string) string {
	if len(conditions) == 0 {
		return ""
	}
	result := conditions[0]
	for i := 1; i < len(conditions); i++ {
		result += sep + conditions[i]
	}
	return result
}

// SingleResult implements docdb.SingleResult for CosmosDB.
type SingleResult struct {
	data []byte
	err  error
}

// Decode decodes the result into the provided interface.
func (r *SingleResult) Decode(v interface{}) error {
	if r.err != nil {
		return r.err
	}
	if r.data == nil {
		return ErrNotFound
	}
	return json.Unmarshal(r.data, v)
}

// Err returns any error from the operation.
func (r *SingleResult) Err() error {
	return r.err
}

// Cursor implements docdb.Cursor for CosmosDB.
type Cursor struct {
	pager    *runtime.Pager[azcosmos.QueryItemsResponse]
	ctx      context.Context
	items    [][]byte
	index    int
	err      error
	pageRead bool
}

// Next advances the cursor to the next document.
func (c *Cursor) Next(ctx context.Context) bool {
	// If we have items in the current page, advance
	if c.index < len(c.items)-1 {
		c.index++
		return true
	}

	// Try to get next page
	if c.pager.More() {
		resp, err := c.pager.NextPage(ctx)
		if err != nil {
			c.err = err
			return false
		}
		c.items = resp.Items
		c.index = 0
		c.pageRead = true
		return len(c.items) > 0
	}

	return false
}

// Decode decodes the current document.
func (c *Cursor) Decode(v interface{}) error {
	if c.err != nil {
		return c.err
	}
	if c.index < 0 || c.index >= len(c.items) {
		return ErrNotFound
	}
	return json.Unmarshal(c.items[c.index], v)
}

// All decodes all remaining documents.
func (c *Cursor) All(ctx context.Context, results interface{}) error {
	// Collect all items
	var allItems [][]byte

	// Add current page items (if any remain)
	if c.pageRead && c.index < len(c.items) {
		allItems = append(allItems, c.items[c.index:]...)
	}

	// Get remaining pages
	for c.pager.More() {
		resp, err := c.pager.NextPage(ctx)
		if err != nil {
			return err
		}
		allItems = append(allItems, resp.Items...)
	}

	// Combine into JSON array
	combinedJSON := []byte("[")
	for i, item := range allItems {
		if i > 0 {
			combinedJSON = append(combinedJSON, ',')
		}
		combinedJSON = append(combinedJSON, item...)
	}
	combinedJSON = append(combinedJSON, ']')

	return json.Unmarshal(combinedJSON, results)
}

// Err returns any cursor error.
func (c *Cursor) Err() error {
	return c.err
}

// Close closes the cursor.
func (c *Cursor) Close(_ context.Context) error {
	// No explicit close needed for CosmosDB pager
	return nil
}
