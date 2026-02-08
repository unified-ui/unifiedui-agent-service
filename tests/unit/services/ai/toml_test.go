// Package ai_test provides unit tests for the AI service package.
package ai_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/unifiedui/agent-service/internal/services/ai"
)

func TestToTOML_SimpleMap(t *testing.T) {
	data := map[string]interface{}{
		"name":   "test-node",
		"status": "completed",
		"type":   "llm",
	}

	result := ai.ToTOML(data)

	assert.Contains(t, result, `name = "test-node"`)
	assert.Contains(t, result, `status = "completed"`)
	assert.Contains(t, result, `type = "llm"`)
}

func TestToTOML_WithNumbers(t *testing.T) {
	data := map[string]interface{}{
		"duration": 2.5,
		"count":    float64(10),
		"active":   true,
	}

	result := ai.ToTOML(data)

	assert.Contains(t, result, "duration = 2.5")
	assert.Contains(t, result, "count = 10")
	assert.Contains(t, result, "active = true")
}

func TestToTOML_NestedMap(t *testing.T) {
	data := map[string]interface{}{
		"name": "node-1",
		"data": map[string]interface{}{
			"input":  "test query",
			"output": "test result",
		},
	}

	result := ai.ToTOML(data)

	assert.Contains(t, result, `name = "node-1"`)
	assert.Contains(t, result, "[data]")
	assert.Contains(t, result, `input = "test query"`)
}

func TestToTOML_NilMap(t *testing.T) {
	result := ai.ToTOML(nil)
	assert.Equal(t, "", result)
}

func TestToTOML_EmptyMap(t *testing.T) {
	result := ai.ToTOML(map[string]interface{}{})
	assert.Equal(t, "", result)
}

func TestSliceToTOML_MultipleItems(t *testing.T) {
	items := []map[string]interface{}{
		{"name": "Agent", "status": "completed", "duration": 2.3},
		{"name": "Tool", "status": "completed", "duration": 0.8},
	}

	result := ai.SliceToTOML(items, "nodes")

	assert.Contains(t, result, "[[nodes]]")
	assert.Contains(t, result, `name = "Agent"`)
	assert.Contains(t, result, `name = "Tool"`)
}

func TestSliceToTOML_EmptySlice(t *testing.T) {
	result := ai.SliceToTOML(nil, "nodes")
	assert.Equal(t, "", result)
}

func TestToTOML_WithNilValue(t *testing.T) {
	data := map[string]interface{}{
		"name":  "test",
		"value": nil,
	}

	result := ai.ToTOML(data)

	assert.Contains(t, result, `name = "test"`)
	assert.Contains(t, result, `value = ""`)
}

func TestToTOML_WithArray(t *testing.T) {
	data := map[string]interface{}{
		"tags": []interface{}{"tag1", "tag2", "tag3"},
	}

	result := ai.ToTOML(data)

	assert.Contains(t, result, `tags = ["tag1", "tag2", "tag3"]`)
}
