package ai_test

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/unifiedui/agent-service/internal/services/ai"
)

func TestTraceToHierarchicalText_Nil(t *testing.T) {
	require.Equal(t, "", ai.TraceToHierarchicalText(nil))
}

func TestTraceToHierarchicalText_StringFields(t *testing.T) {
	trace := map[string]interface{}{
		"id":          "t-1",
		"contextType": "conversation",
	}
	result := ai.TraceToHierarchicalText(trace)
	require.Contains(t, result, `id = "t-1"`)
	require.Contains(t, result, `contextType = "conversation"`)
}

func TestTraceToHierarchicalText_LongString(t *testing.T) {
	long := strings.Repeat("x", 600)
	trace := map[string]interface{}{
		"description": long,
	}
	result := ai.TraceToHierarchicalText(trace)
	require.Contains(t, result, "...")
	require.NotContains(t, result, strings.Repeat("x", 600))
}

func TestTraceToHierarchicalText_EmptyString(t *testing.T) {
	trace := map[string]interface{}{
		"name":  "",
		"count": float64(5),
	}
	result := ai.TraceToHierarchicalText(trace)
	require.NotContains(t, result, "name")
	require.Contains(t, result, "count = 5")
}

func TestTraceToHierarchicalText_FloatWholeNumber(t *testing.T) {
	trace := map[string]interface{}{
		"duration": float64(10),
	}
	result := ai.TraceToHierarchicalText(trace)
	require.Contains(t, result, "duration = 10")
}

func TestTraceToHierarchicalText_FloatDecimal(t *testing.T) {
	trace := map[string]interface{}{
		"score": 3.14,
	}
	result := ai.TraceToHierarchicalText(trace)
	require.Contains(t, result, "score = 3.14")
}

func TestTraceToHierarchicalText_Boolean(t *testing.T) {
	trace := map[string]interface{}{
		"active": true,
	}
	result := ai.TraceToHierarchicalText(trace)
	require.Contains(t, result, "active = true")
}

func TestTraceToHierarchicalText_NilField(t *testing.T) {
	trace := map[string]interface{}{
		"id":   "t-1",
		"data": nil,
	}
	result := ai.TraceToHierarchicalText(trace)
	require.Contains(t, result, "id")
	require.NotContains(t, result, "data")
}

func TestTraceToHierarchicalText_NestedMap(t *testing.T) {
	trace := map[string]interface{}{
		"metadata": map[string]interface{}{
			"key1": "val1",
			"key2": float64(42),
		},
	}
	result := ai.TraceToHierarchicalText(trace)
	require.Contains(t, result, "metadata:")
	require.Contains(t, result, `key1 = "val1"`)
	require.Contains(t, result, "key2 = 42")
}

func TestTraceToHierarchicalText_ArrayOfMaps(t *testing.T) {
	trace := map[string]interface{}{
		"logs": []interface{}{
			map[string]interface{}{"level": "info", "msg": "started"},
			map[string]interface{}{"level": "error", "msg": "failed"},
		},
	}
	result := ai.TraceToHierarchicalText(trace)
	require.Contains(t, result, "logs:")
	require.Contains(t, result, "[1]")
	require.Contains(t, result, "[2]")
	require.Contains(t, result, `level = "info"`)
}

func TestTraceToHierarchicalText_ArrayOfPrimitives(t *testing.T) {
	trace := map[string]interface{}{
		"tags": []interface{}{"tag1", "tag2"},
	}
	result := ai.TraceToHierarchicalText(trace)
	require.Contains(t, result, "tags:")
	require.Contains(t, result, "- tag1")
	require.Contains(t, result, "- tag2")
}

func TestTraceToHierarchicalText_EmptyArray(t *testing.T) {
	trace := map[string]interface{}{
		"id":   "t-1",
		"logs": []interface{}{},
	}
	result := ai.TraceToHierarchicalText(trace)
	require.NotContains(t, result, "logs")
}

func TestTraceToHierarchicalText_DefaultType(t *testing.T) {
	trace := map[string]interface{}{
		"custom": struct{ X int }{X: 5},
	}
	result := ai.TraceToHierarchicalText(trace)
	require.Contains(t, result, "custom")
}

func TestTraceToHierarchicalText_WithNodes(t *testing.T) {
	trace := map[string]interface{}{
		"id": "t-1",
		"nodes": []interface{}{
			map[string]interface{}{
				"name": "Agent", "type": "agent", "status": "completed",
			},
		},
	}
	result := ai.TraceToHierarchicalText(trace)
	require.Contains(t, result, "Nodes (1):")
	require.Contains(t, result, "[Node 1] Agent")
}

func TestTraceToHierarchicalText_NodesNotSlice(t *testing.T) {
	trace := map[string]interface{}{
		"id":    "t-1",
		"nodes": "not-a-slice",
	}
	result := ai.TraceToHierarchicalText(trace)
	require.NotContains(t, result, "Nodes (")
}

func TestTraceToHierarchicalText_NodesNotMaps(t *testing.T) {
	trace := map[string]interface{}{
		"id":    "t-1",
		"nodes": []interface{}{"not-a-map"},
	}
	result := ai.TraceToHierarchicalText(trace)
	require.NotContains(t, result, "Nodes (")
}

func TestNodesToHierarchicalText_NodeFields(t *testing.T) {
	nodes := []map[string]interface{}{
		{
			"name":    "Tool",
			"type":    "tool",
			"status":  "failed",
			"error":   "timeout",
			"id":      "n-1",
			"enabled": true,
		},
	}
	result := ai.NodesToHierarchicalText(nodes)
	require.Contains(t, result, "[Node 1] Tool (type=tool, status=failed)")
	require.Contains(t, result, `error = "timeout"`)
	require.Contains(t, result, `id = "n-1"`)
	require.Contains(t, result, "enabled = true")
}

func TestNodesToHierarchicalText_LongStringField(t *testing.T) {
	long := strings.Repeat("y", 400)
	nodes := []map[string]interface{}{
		{
			"name": "N", "type": "llm", "status": "ok",
			"output": long,
		},
	}
	result := ai.NodesToHierarchicalText(nodes)
	require.Contains(t, result, "...")
	require.Less(t, len(result), 400)
}

func TestNodesToHierarchicalText_EmptyStringField(t *testing.T) {
	nodes := []map[string]interface{}{
		{
			"name": "N", "type": "llm", "status": "ok",
			"output": "",
		},
	}
	result := ai.NodesToHierarchicalText(nodes)
	require.NotContains(t, result, "output")
}

func TestNodesToHierarchicalText_NilField(t *testing.T) {
	nodes := []map[string]interface{}{
		{
			"name": "N", "type": "llm", "status": "ok",
			"data": nil,
		},
	}
	result := ai.NodesToHierarchicalText(nodes)
	require.NotContains(t, result, "data")
}

func TestNodesToHierarchicalText_FloatField(t *testing.T) {
	nodes := []map[string]interface{}{
		{
			"name": "N", "type": "llm", "status": "ok",
			"duration": 2.5,
			"count":    float64(10),
		},
	}
	result := ai.NodesToHierarchicalText(nodes)
	require.Contains(t, result, "duration = 2.5")
	require.Contains(t, result, "count = 10")
}

func TestNodesToHierarchicalText_NestedMapField(t *testing.T) {
	nodes := []map[string]interface{}{
		{
			"name": "N", "type": "llm", "status": "ok",
			"data": map[string]interface{}{
				"input": "hello",
			},
		},
	}
	result := ai.NodesToHierarchicalText(nodes)
	require.Contains(t, result, "data:")
	require.Contains(t, result, `input = "hello"`)
}

func TestNodesToHierarchicalText_ArrayField(t *testing.T) {
	nodes := []map[string]interface{}{
		{
			"name": "N", "type": "llm", "status": "ok",
			"logs": []interface{}{"log1", "log2"},
		},
	}
	result := ai.NodesToHierarchicalText(nodes)
	require.Contains(t, result, "logs: [2 items]")
}

func TestNodesToHierarchicalText_EmptyArrayField(t *testing.T) {
	nodes := []map[string]interface{}{
		{
			"name": "N", "type": "llm", "status": "ok",
			"logs": []interface{}{},
		},
	}
	result := ai.NodesToHierarchicalText(nodes)
	require.NotContains(t, result, "logs")
}

func TestNodesToHierarchicalText_DefaultTypeField(t *testing.T) {
	nodes := []map[string]interface{}{
		{
			"name": "N", "type": "llm", "status": "ok",
			"custom": []int{1, 2, 3},
		},
	}
	result := ai.NodesToHierarchicalText(nodes)
	require.Contains(t, result, "custom")
}

func TestNodesToHierarchicalText_ChildNodes(t *testing.T) {
	nodes := []map[string]interface{}{
		{
			"name": "Parent", "type": "agent", "status": "completed",
			"nodes": []interface{}{
				map[string]interface{}{
					"name": "Child1", "type": "llm", "status": "completed",
				},
				map[string]interface{}{
					"name": "Child2", "type": "tool", "status": "completed",
				},
			},
		},
	}
	result := ai.NodesToHierarchicalText(nodes)
	require.Contains(t, result, "children: 2")
	require.Contains(t, result, "[Node 1] Child1")
	require.Contains(t, result, "[Node 2] Child2")
}

func TestNodesToHierarchicalText_DeepNesting(t *testing.T) {
	nodes := []map[string]interface{}{
		{
			"name": "L0", "type": "agent", "status": "ok",
			"nodes": []interface{}{
				map[string]interface{}{
					"name": "L1", "type": "llm", "status": "ok",
					"nodes": []interface{}{
						map[string]interface{}{
							"name": "L2", "type": "tool", "status": "ok",
						},
					},
				},
			},
		},
	}
	result := ai.NodesToHierarchicalText(nodes)
	require.Contains(t, result, "[Node 1] L0")
	require.Contains(t, result, "[Node 1] L1")
	require.Contains(t, result, "[Node 1] L2")
}

func TestWriteNodeDataMap_NestedMap(t *testing.T) {
	trace := map[string]interface{}{
		"data": map[string]interface{}{
			"nested": map[string]interface{}{
				"deep": "value",
			},
			"flag": true,
			"num":  float64(10),
			"dec":  1.5,
		},
	}
	result := ai.TraceToHierarchicalText(trace)
	require.Contains(t, result, `deep = "value"`)
	require.Contains(t, result, "flag = true")
	require.Contains(t, result, "num = 10")
	require.Contains(t, result, "dec = 1.5")
}

func TestWriteNodeDataMap_LongString(t *testing.T) {
	long := strings.Repeat("z", 400)
	trace := map[string]interface{}{
		"data": map[string]interface{}{
			"content": long,
		},
	}
	result := ai.TraceToHierarchicalText(trace)
	require.Contains(t, result, "...")
}

func TestWriteNodeDataMap_DefaultType(t *testing.T) {
	trace := map[string]interface{}{
		"data": map[string]interface{}{
			"custom": struct{ X int }{X: 1},
		},
	}
	result := ai.TraceToHierarchicalText(trace)
	require.Contains(t, result, "custom")
}

func TestWriteNodeDataMap_NilValue(t *testing.T) {
	trace := map[string]interface{}{
		"data": map[string]interface{}{
			"key": nil,
		},
	}
	result := ai.TraceToHierarchicalText(trace)
	require.NotContains(t, result, "key")
}

func TestToTOML_IntValue(t *testing.T) {
	data := map[string]interface{}{
		"count": 42,
	}
	result := ai.ToTOML(data)
	require.Contains(t, result, "count = 42")
}

func TestToTOML_Int64Value(t *testing.T) {
	data := map[string]interface{}{
		"bignum": int64(1234567890),
	}
	result := ai.ToTOML(data)
	require.Contains(t, result, "bignum = 1234567890")
}

func TestToTOML_DefaultTypeValue(t *testing.T) {
	data := map[string]interface{}{
		"custom": struct{ X int }{X: 1},
	}
	result := ai.ToTOML(data)
	require.Contains(t, result, "custom")
}

func TestToTOML_ArrayOfMaps(t *testing.T) {
	data := map[string]interface{}{
		"items": []interface{}{
			map[string]interface{}{"name": "a"},
			map[string]interface{}{"name": "b"},
		},
	}
	result := ai.ToTOML(data)
	require.Contains(t, result, "[[items]]")
	require.Contains(t, result, `name = "a"`)
	require.Contains(t, result, `name = "b"`)
}

func TestToTOML_EmptyArray(t *testing.T) {
	data := map[string]interface{}{
		"items": []interface{}{},
	}
	result := ai.ToTOML(data)
	require.Contains(t, result, "items = []")
}

func TestToTOML_MixedArray(t *testing.T) {
	data := map[string]interface{}{
		"items": []interface{}{"str", 42, true},
	}
	result := ai.ToTOML(data)
	require.Contains(t, result, `"str"`)
	require.Contains(t, result, "42")
	require.Contains(t, result, "true")
}

func TestSliceToTOML_SingleItem(t *testing.T) {
	items := []map[string]interface{}{
		{"key": "value"},
	}
	result := ai.SliceToTOML(items, "test")
	require.Contains(t, result, "[[test]]")
	require.Contains(t, result, `key = "value"`)
}
