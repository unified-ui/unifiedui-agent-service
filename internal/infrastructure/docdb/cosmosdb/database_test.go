// Package cosmosdb provides unit tests for database helper functions.
package cosmosdb

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestJoinConditions_Empty(t *testing.T) {
	result := joinConditions([]string{}, " AND ")
	assert.Empty(t, result)
}

func TestJoinConditions_Single(t *testing.T) {
	result := joinConditions([]string{"c.id = @p0"}, " AND ")
	assert.Equal(t, "c.id = @p0", result)
}

func TestJoinConditions_Multiple(t *testing.T) {
	conditions := []string{"c.id = @p0", "c.tenantId = @p1", "c.status = @p2"}
	result := joinConditions(conditions, " AND ")
	assert.Equal(t, "c.id = @p0 AND c.tenantId = @p1 AND c.status = @p2", result)
}

func TestJoinConditions_DifferentSeparator(t *testing.T) {
	conditions := []string{"@p0", "@p1", "@p2"}
	result := joinConditions(conditions, ", ")
	assert.Equal(t, "@p0, @p1, @p2", result)
}

func TestSingleResult_DecodeSuccess(t *testing.T) {
	data := []byte(`{"id": "test-123", "name": "Test"}`)
	sr := &SingleResult{data: data}

	var result map[string]interface{}
	err := sr.Decode(&result)

	require.NoError(t, err)
	assert.Equal(t, "test-123", result["id"])
	assert.Equal(t, "Test", result["name"])
}

func TestSingleResult_DecodeError(t *testing.T) {
	sr := &SingleResult{err: ErrNotFound}

	var result map[string]interface{}
	err := sr.Decode(&result)

	assert.Error(t, err)
	assert.Equal(t, ErrNotFound, err)
}

func TestSingleResult_DecodeNilData(t *testing.T) {
	sr := &SingleResult{data: nil}

	var result map[string]interface{}
	err := sr.Decode(&result)

	assert.Error(t, err)
	assert.Equal(t, ErrNotFound, err)
}

func TestSingleResult_Err(t *testing.T) {
	sr := &SingleResult{err: ErrNotFound}
	assert.Equal(t, ErrNotFound, sr.Err())
}

func TestSingleResult_ErrNil(t *testing.T) {
	sr := &SingleResult{}
	assert.Nil(t, sr.Err())
}

func TestCollection_ApplyUpdate_SetOperator(t *testing.T) {
	col := &Collection{}
	doc := map[string]interface{}{
		"id":       "doc-123",
		"name":     "Original",
		"tenantId": "tenant-1",
	}
	update := map[string]interface{}{
		"$set": map[string]interface{}{
			"name":   "Updated",
			"status": "active",
		},
	}

	result, err := col.applyUpdate(doc, update)

	require.NoError(t, err)
	assert.Equal(t, "doc-123", result["id"])
	assert.Equal(t, "Updated", result["name"])
	assert.Equal(t, "active", result["status"])
	assert.Equal(t, "tenant-1", result["tenantId"])
}

func TestCollection_ApplyUpdate_PushOperator(t *testing.T) {
	col := &Collection{}
	doc := map[string]interface{}{
		"id":   "doc-123",
		"tags": []interface{}{"tag1", "tag2"},
	}
	update := map[string]interface{}{
		"$push": map[string]interface{}{
			"tags": "tag3",
		},
	}

	result, err := col.applyUpdate(doc, update)

	require.NoError(t, err)
	tags := result["tags"].([]interface{})
	assert.Len(t, tags, 3)
	assert.Equal(t, "tag1", tags[0])
	assert.Equal(t, "tag2", tags[1])
	assert.Equal(t, "tag3", tags[2])
}

func TestCollection_ApplyUpdate_PushWithEach(t *testing.T) {
	col := &Collection{}
	doc := map[string]interface{}{
		"id":    "doc-123",
		"nodes": []interface{}{},
	}
	update := map[string]interface{}{
		"$push": map[string]interface{}{
			"nodes": map[string]interface{}{
				"$each": []interface{}{"node1", "node2", "node3"},
			},
		},
	}

	result, err := col.applyUpdate(doc, update)

	require.NoError(t, err)
	nodes := result["nodes"].([]interface{})
	assert.Len(t, nodes, 3)
}

func TestCollection_ApplyUpdate_UnsetOperator(t *testing.T) {
	col := &Collection{}
	doc := map[string]interface{}{
		"id":       "doc-123",
		"name":     "Test",
		"toRemove": "value",
	}
	update := map[string]interface{}{
		"$unset": map[string]interface{}{
			"toRemove": "",
		},
	}

	result, err := col.applyUpdate(doc, update)

	require.NoError(t, err)
	assert.Equal(t, "doc-123", result["id"])
	assert.Equal(t, "Test", result["name"])
	_, exists := result["toRemove"]
	assert.False(t, exists)
}

func TestCollection_ApplyUpdate_IncOperator_Float(t *testing.T) {
	col := &Collection{}
	doc := map[string]interface{}{
		"id":    "doc-123",
		"count": float64(10),
	}
	update := map[string]interface{}{
		"$inc": map[string]interface{}{
			"count": float64(5),
		},
	}

	result, err := col.applyUpdate(doc, update)

	require.NoError(t, err)
	assert.Equal(t, float64(15), result["count"])
}

func TestCollection_ApplyUpdate_IncOperator_Int(t *testing.T) {
	col := &Collection{}
	doc := map[string]interface{}{
		"id":    "doc-123",
		"count": float64(10),
	}
	update := map[string]interface{}{
		"$inc": map[string]interface{}{
			"count": 5,
		},
	}

	result, err := col.applyUpdate(doc, update)

	require.NoError(t, err)
	assert.Equal(t, float64(15), result["count"])
}

func TestCollection_ApplyUpdate_InvalidUpdate(t *testing.T) {
	col := &Collection{}
	doc := map[string]interface{}{
		"id": "doc-123",
	}
	update := "invalid"

	_, err := col.applyUpdate(doc, update)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "update must be a map")
}

func TestCollection_ApplyUpdate_CombinedOperators(t *testing.T) {
	col := &Collection{}
	doc := map[string]interface{}{
		"id":       "doc-123",
		"name":     "Original",
		"count":    float64(0),
		"toRemove": "value",
	}
	update := map[string]interface{}{
		"$set": map[string]interface{}{
			"name": "Updated",
		},
		"$inc": map[string]interface{}{
			"count": float64(1),
		},
		"$unset": map[string]interface{}{
			"toRemove": "",
		},
	}

	result, err := col.applyUpdate(doc, update)

	require.NoError(t, err)
	assert.Equal(t, "Updated", result["name"])
	assert.Equal(t, float64(1), result["count"])
	_, exists := result["toRemove"]
	assert.False(t, exists)
}

func TestCollection_AddOrderBy_Ascending(t *testing.T) {
	col := &Collection{}
	query := "SELECT * FROM c WHERE c.tenantId = @p0"
	sort := map[string]interface{}{
		"createdAt": 1,
	}

	result := col.addOrderBy(query, sort)

	assert.Contains(t, result, "ORDER BY c.createdAt ASC")
}

func TestCollection_AddOrderBy_Descending(t *testing.T) {
	col := &Collection{}
	query := "SELECT * FROM c WHERE c.tenantId = @p0"
	sort := map[string]interface{}{
		"createdAt": -1,
	}

	result := col.addOrderBy(query, sort)

	assert.Contains(t, result, "ORDER BY c.createdAt DESC")
}

func TestCollection_AddOrderBy_DescendingFloat(t *testing.T) {
	col := &Collection{}
	query := "SELECT * FROM c WHERE c.tenantId = @p0"
	sort := map[string]interface{}{
		"createdAt": float64(-1),
	}

	result := col.addOrderBy(query, sort)

	assert.Contains(t, result, "ORDER BY c.createdAt DESC")
}

func TestCollection_AddOrderBy_NotMap(t *testing.T) {
	col := &Collection{}
	query := "SELECT * FROM c WHERE c.tenantId = @p0"

	result := col.addOrderBy(query, "invalid")

	assert.Equal(t, query, result)
}

func TestCollection_PrepareDocument_NormalizeID(t *testing.T) {
	col := &Collection{}
	input := map[string]interface{}{
		"_id":      "doc-123",
		"tenantId": "tenant-1",
		"name":     "Test",
	}

	doc, pk, err := col.prepareDocument(input)

	require.NoError(t, err)
	assert.Equal(t, "doc-123", doc["id"])
	_, hasOldID := doc["_id"]
	assert.False(t, hasOldID)
	assert.NotEmpty(t, pk)
}

func TestCollection_PrepareDocument_WithExistingID(t *testing.T) {
	col := &Collection{}
	input := map[string]interface{}{
		"id":       "doc-123",
		"tenantId": "tenant-1",
	}

	doc, pk, err := col.prepareDocument(input)

	require.NoError(t, err)
	assert.Equal(t, "doc-123", doc["id"])
	assert.NotEmpty(t, pk)
}

func TestCollection_FilterToQuery_EmptyFilter(t *testing.T) {
	col := &Collection{}

	query, params, _ := col.filterToQuery(map[string]interface{}{})

	assert.Equal(t, "SELECT * FROM c", query)
	assert.Empty(t, params)
}

func TestCollection_FilterToQuery_SingleCondition(t *testing.T) {
	col := &Collection{}
	filter := map[string]interface{}{
		"tenantId": "tenant-123",
	}

	query, params, pk := col.filterToQuery(filter)

	assert.Contains(t, query, "SELECT * FROM c WHERE")
	assert.Contains(t, query, "c.tenantId = @p0")
	assert.Len(t, params, 1)
	assert.Equal(t, "@p0", params[0].Name)
	assert.Equal(t, "tenant-123", params[0].Value)
	assert.NotEmpty(t, pk)
}

func TestCollection_FilterToQuery_NormalizeIDField(t *testing.T) {
	col := &Collection{}
	filter := map[string]interface{}{
		"_id": "doc-123",
	}

	query, params, _ := col.filterToQuery(filter)

	assert.Contains(t, query, "c.id = @p0")
	assert.Len(t, params, 1)
	assert.Equal(t, "doc-123", params[0].Value)
}

func TestCollection_FilterToQuery_NotMap(t *testing.T) {
	col := &Collection{}

	query, params, pk := col.filterToQuery("invalid")

	assert.Equal(t, "SELECT * FROM c", query)
	assert.Nil(t, params)
	assert.Empty(t, pk)
}

func TestCollection_FilterToCountQuery(t *testing.T) {
	col := &Collection{}
	filter := map[string]interface{}{
		"tenantId": "tenant-123",
	}

	query, params, _ := col.filterToCountQuery(filter)

	assert.Contains(t, query, "SELECT VALUE COUNT(1) FROM c WHERE")
	assert.Len(t, params, 1)
}

func TestCursor_Err(t *testing.T) {
	cursor := &Cursor{err: ErrNotFound}
	assert.Equal(t, ErrNotFound, cursor.Err())
}

func TestCursor_ErrNil(t *testing.T) {
	cursor := &Cursor{}
	assert.Nil(t, cursor.Err())
}

func TestCursor_DecodeError(t *testing.T) {
	cursor := &Cursor{err: ErrNotFound}

	var result map[string]interface{}
	err := cursor.Decode(&result)

	assert.Equal(t, ErrNotFound, err)
}

func TestCursor_DecodeOutOfBounds(t *testing.T) {
	cursor := &Cursor{
		items: [][]byte{},
		index: 0,
	}

	var result map[string]interface{}
	err := cursor.Decode(&result)

	assert.Equal(t, ErrNotFound, err)
}

func TestCursor_Close(t *testing.T) {
	cursor := &Cursor{}
	err := cursor.Close(context.TODO())
	assert.NoError(t, err)
}
