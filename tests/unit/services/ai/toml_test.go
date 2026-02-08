// Package ai_test provides unit tests for the AI service package.
package ai_test

import (
	"strings"
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

func TestNodesToHierarchicalText_FlatNodes(t *testing.T) {
	nodes := []map[string]interface{}{
		{"name": "Agent", "type": "agent", "status": "completed", "duration": 2.3},
		{"name": "Tool Call", "type": "tool", "status": "failed", "error": "timeout"},
	}

	result := ai.NodesToHierarchicalText(nodes)

	assert.Contains(t, result, "[Node 1] Agent (type=agent, status=completed)")
	assert.Contains(t, result, "[Node 2] Tool Call (type=tool, status=failed)")
	assert.Contains(t, result, `error = "timeout"`)
}

func TestNodesToHierarchicalText_NestedNodes(t *testing.T) {
	nodes := []map[string]interface{}{
		{
			"name":   "Root Agent",
			"type":   "agent",
			"status": "completed",
			"nodes": []interface{}{
				map[string]interface{}{
					"name":   "LLM Call",
					"type":   "llm",
					"status": "completed",
				},
				map[string]interface{}{
					"name":   "Tool Call",
					"type":   "tool",
					"status": "completed",
					"nodes": []interface{}{
						map[string]interface{}{
							"name":   "HTTP Request",
							"type":   "http",
							"status": "completed",
						},
					},
				},
			},
		},
	}

	result := ai.NodesToHierarchicalText(nodes)

	assert.Contains(t, result, "[Node 1] Root Agent (type=agent, status=completed)")
	assert.Contains(t, result, "children: 2")
	assert.Contains(t, result, "  [Node 1] LLM Call (type=llm, status=completed)")
	assert.Contains(t, result, "  [Node 2] Tool Call (type=tool, status=completed)")
	assert.Contains(t, result, "    [Node 1] HTTP Request (type=http, status=completed)")
}

func TestNodesToHierarchicalText_Empty(t *testing.T) {
	result := ai.NodesToHierarchicalText(nil)
	assert.Equal(t, "", result)
}

func TestNodesToHierarchicalText_PreservesDepth(t *testing.T) {
	nodes := []map[string]interface{}{
		{
			"name": "A", "type": "agent", "status": "completed",
			"nodes": []interface{}{
				map[string]interface{}{
					"name": "B", "type": "llm", "status": "completed",
					"nodes": []interface{}{
						map[string]interface{}{
							"name": "C", "type": "tool", "status": "completed",
						},
					},
				},
			},
		},
		{
			"name": "D", "type": "agent", "status": "completed",
		},
	}

	result := ai.NodesToHierarchicalText(nodes)

	lines := splitNonEmpty(result)
	foundA := false
	foundB := false
	foundC := false
	foundD := false
	for _, line := range lines {
		if contains(line, "[Node 1] A") && !startsWith(line, " ") {
			foundA = true
		}
		if contains(line, "[Node 1] B") && startsWith(line, "  ") && !startsWith(line, "    ") {
			foundB = true
		}
		if contains(line, "[Node 1] C") && startsWith(line, "    ") {
			foundC = true
		}
		if contains(line, "[Node 2] D") && !startsWith(line, " ") {
			foundD = true
		}
	}
	assert.True(t, foundA, "Root node A should be at depth 0")
	assert.True(t, foundB, "Child node B should be at depth 1")
	assert.True(t, foundC, "Grandchild node C should be at depth 2")
	assert.True(t, foundD, "Sibling node D should be at depth 0")
}

func splitNonEmpty(s string) []string {
	lines := []string{}
	for _, line := range strings.Split(s, "\n") {
		if line != "" {
			lines = append(lines, line)
		}
	}
	return lines
}

func contains(s, substr string) bool {
	return strings.Contains(s, substr)
}

func startsWith(s, prefix string) bool {
	return strings.HasPrefix(s, prefix)
}

func TestTraceToHierarchicalText_EmptyTrace(t *testing.T) {
	result := ai.TraceToHierarchicalText(map[string]interface{}{})
	assert.Empty(t, result)
}

func TestTraceToHierarchicalText_RootFieldsOnly(t *testing.T) {
	trace := map[string]interface{}{
		"id":            "trace-abc",
		"contextType":   "conversation",
		"referenceName": "my-agent",
	}

	result := ai.TraceToHierarchicalText(trace)

	assert.Contains(t, result, `contextType = "conversation"`)
	assert.Contains(t, result, `id = "trace-abc"`)
	assert.Contains(t, result, `referenceName = "my-agent"`)
	assert.NotContains(t, result, "Nodes")
}

func TestTraceToHierarchicalText_WithNodesAndMetadata(t *testing.T) {
	trace := map[string]interface{}{
		"id":          "trace-123",
		"contextType": "autonomous",
		"logs": []interface{}{
			map[string]interface{}{"level": "info", "message": "started"},
		},
		"nodes": []interface{}{
			map[string]interface{}{
				"name":   "Agent",
				"type":   "agent",
				"status": "completed",
				"nodes": []interface{}{
					map[string]interface{}{
						"name":   "LLM",
						"type":   "llm",
						"status": "completed",
					},
				},
			},
		},
	}

	result := ai.TraceToHierarchicalText(trace)

	assert.Contains(t, result, `contextType = "autonomous"`)
	assert.Contains(t, result, `id = "trace-123"`)
	assert.Contains(t, result, "logs:")
	assert.Contains(t, result, "Nodes (1):")
	assert.Contains(t, result, "[Node 1] Agent")
	assert.Contains(t, result, "[Node 1] LLM")
}

func TestTraceToHierarchicalText_NodesFieldExcludedFromRootFields(t *testing.T) {
	trace := map[string]interface{}{
		"id": "trace-1",
		"nodes": []interface{}{
			map[string]interface{}{"name": "Step1", "type": "tool", "status": "ok"},
		},
	}

	result := ai.TraceToHierarchicalText(trace)

	lines := splitNonEmpty(result)
	nodesHeaderCount := 0
	for _, line := range lines {
		if contains(line, "nodes") && !contains(line, "Nodes (") && !contains(line, "[Node") {
			nodesHeaderCount++
		}
	}
	assert.Equal(t, 0, nodesHeaderCount, "nodes should not appear as a root-level field")
	assert.Contains(t, result, "Nodes (1):")
}
