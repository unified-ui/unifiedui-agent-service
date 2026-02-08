// Package ai provides the AI service for LLM interactions.
package ai

import (
	"fmt"
	"sort"
	"strings"
)

// ToTOML converts a map to a TOML-like string representation.
// This is a simplified TOML serializer optimized for token savings when sending data to LLMs.
func ToTOML(data map[string]interface{}) string {
	if data == nil {
		return ""
	}

	var sb strings.Builder
	writeTOMLMap(&sb, data, "")
	return sb.String()
}

// SliceToTOML converts a slice of maps to TOML array-of-tables format.
func SliceToTOML(items []map[string]interface{}, tableName string) string {
	if len(items) == 0 {
		return ""
	}

	var sb strings.Builder
	for _, item := range items {
		sb.WriteString(fmt.Sprintf("[[%s]]\n", tableName))
		writeTOMLMap(&sb, item, "")
		sb.WriteString("\n")
	}
	return sb.String()
}

func writeTOMLMap(sb *strings.Builder, data map[string]interface{}, prefix string) {
	keys := sortedKeys(data)

	for _, key := range keys {
		value := data[key]
		fullKey := key
		if prefix != "" {
			fullKey = prefix + "." + key
		}

		switch v := value.(type) {
		case map[string]interface{}:
			sb.WriteString(fmt.Sprintf("[%s]\n", fullKey))
			writeTOMLMap(sb, v, fullKey)
		case []interface{}:
			writeTOMLArray(sb, fullKey, v)
		case string:
			sb.WriteString(fmt.Sprintf("%s = %q\n", key, v))
		case float64:
			if v == float64(int64(v)) {
				sb.WriteString(fmt.Sprintf("%s = %d\n", key, int64(v)))
			} else {
				sb.WriteString(fmt.Sprintf("%s = %g\n", key, v))
			}
		case int:
			sb.WriteString(fmt.Sprintf("%s = %d\n", key, v))
		case int64:
			sb.WriteString(fmt.Sprintf("%s = %d\n", key, v))
		case bool:
			sb.WriteString(fmt.Sprintf("%s = %t\n", key, v))
		case nil:
			sb.WriteString(fmt.Sprintf("%s = \"\"\n", key))
		default:
			sb.WriteString(fmt.Sprintf("%s = %q\n", key, fmt.Sprintf("%v", v)))
		}
	}
}

func writeTOMLArray(sb *strings.Builder, key string, arr []interface{}) {
	if len(arr) == 0 {
		sb.WriteString(fmt.Sprintf("%s = []\n", key))
		return
	}

	allMaps := true
	for _, item := range arr {
		if _, ok := item.(map[string]interface{}); !ok {
			allMaps = false
			break
		}
	}

	if allMaps {
		for _, item := range arr {
			m := item.(map[string]interface{})
			sb.WriteString(fmt.Sprintf("[[%s]]\n", key))
			writeTOMLMap(sb, m, "")
		}
		return
	}

	values := make([]string, len(arr))
	for i, item := range arr {
		switch v := item.(type) {
		case string:
			values[i] = fmt.Sprintf("%q", v)
		default:
			values[i] = fmt.Sprintf("%v", v)
		}
	}
	sb.WriteString(fmt.Sprintf("%s = [%s]\n", key, strings.Join(values, ", ")))
}

func sortedKeys(m map[string]interface{}) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

// TraceToHierarchicalText converts a full trace object (including root-level
// fields like logs, metadata, contextType, etc.) into a text representation.
// Nodes are rendered via NodesToHierarchicalText preserving hierarchy.
func TraceToHierarchicalText(trace map[string]interface{}) string {
	if len(trace) == 0 {
		return ""
	}

	var sb strings.Builder

	nodeSkipKeys := map[string]bool{"nodes": true}

	for _, key := range sortedKeys(trace) {
		if nodeSkipKeys[key] {
			continue
		}

		value := trace[key]
		if value == nil {
			continue
		}

		switch v := value.(type) {
		case map[string]interface{}:
			sb.WriteString(fmt.Sprintf("%s:\n", key))
			writeNodeDataMap(&sb, v, "  ")
		case []interface{}:
			if len(v) == 0 {
				continue
			}
			sb.WriteString(fmt.Sprintf("%s:\n", key))
			for i, item := range v {
				switch it := item.(type) {
				case map[string]interface{}:
					sb.WriteString(fmt.Sprintf("  [%d]\n", i+1))
					writeNodeDataMap(&sb, it, "    ")
				default:
					sb.WriteString(fmt.Sprintf("  - %v\n", item))
				}
			}
		case string:
			if v == "" {
				continue
			}
			if len(v) > 500 {
				v = v[:500] + "..."
			}
			sb.WriteString(fmt.Sprintf("%s = %q\n", key, v))
		case float64:
			if v == float64(int64(v)) {
				sb.WriteString(fmt.Sprintf("%s = %d\n", key, int64(v)))
			} else {
				sb.WriteString(fmt.Sprintf("%s = %g\n", key, v))
			}
		case bool:
			sb.WriteString(fmt.Sprintf("%s = %t\n", key, v))
		default:
			sb.WriteString(fmt.Sprintf("%s = %q\n", key, fmt.Sprintf("%v", v)))
		}
	}

	var nodes []map[string]interface{}
	if rawNodes, ok := trace["nodes"]; ok {
		if nodeSlice, ok := rawNodes.([]interface{}); ok {
			for _, n := range nodeSlice {
				if m, ok := n.(map[string]interface{}); ok {
					nodes = append(nodes, m)
				}
			}
		}
	}

	if len(nodes) > 0 {
		sb.WriteString(fmt.Sprintf("\nNodes (%d):\n", len(nodes)))
		nodesText := NodesToHierarchicalText(nodes)
		sb.WriteString(nodesText)
	}

	return sb.String()
}

// NodesToHierarchicalText converts a hierarchical slice of trace node maps into
// an indented text representation that preserves the parent-child tree structure.
func NodesToHierarchicalText(nodes []map[string]interface{}) string {
	if len(nodes) == 0 {
		return ""
	}

	var sb strings.Builder
	writeNodeTree(&sb, nodes, 0)
	return sb.String()
}

func writeNodeTree(sb *strings.Builder, nodes []map[string]interface{}, depth int) {
	indent := strings.Repeat("  ", depth)

	for i, node := range nodes {
		name, _ := node["name"].(string)
		nodeType, _ := node["type"].(string)
		status, _ := node["status"].(string)

		sb.WriteString(fmt.Sprintf("%s[Node %d] %s (type=%s, status=%s)\n", indent, i+1, name, nodeType, status))

		fieldIndent := indent + "  "
		for _, key := range sortedKeys(node) {
			if key == "name" || key == "type" || key == "status" || key == "nodes" {
				continue
			}
			value := node[key]
			if value == nil {
				continue
			}

			switch v := value.(type) {
			case map[string]interface{}:
				sb.WriteString(fmt.Sprintf("%s%s:\n", fieldIndent, key))
				writeNodeDataMap(sb, v, fieldIndent+"  ")
			case []interface{}:
				if len(v) == 0 {
					continue
				}
				sb.WriteString(fmt.Sprintf("%s%s: [%d items]\n", fieldIndent, key, len(v)))
			case string:
				if v == "" {
					continue
				}
				if len(v) > 300 {
					v = v[:300] + "..."
				}
				sb.WriteString(fmt.Sprintf("%s%s = %q\n", fieldIndent, key, v))
			case float64:
				if v == float64(int64(v)) {
					sb.WriteString(fmt.Sprintf("%s%s = %d\n", fieldIndent, key, int64(v)))
				} else {
					sb.WriteString(fmt.Sprintf("%s%s = %g\n", fieldIndent, key, v))
				}
			case bool:
				sb.WriteString(fmt.Sprintf("%s%s = %t\n", fieldIndent, key, v))
			default:
				sb.WriteString(fmt.Sprintf("%s%s = %q\n", fieldIndent, key, fmt.Sprintf("%v", v)))
			}
		}

		var childNodes []map[string]interface{}
		if rawChildren, ok := node["nodes"]; ok {
			if children, ok := rawChildren.([]interface{}); ok && len(children) > 0 {
				for _, child := range children {
					if m, ok := child.(map[string]interface{}); ok {
						childNodes = append(childNodes, m)
					}
				}
			}
		}

		if len(childNodes) > 0 {
			sb.WriteString(fmt.Sprintf("%s  children: %d\n", indent, len(childNodes)))
			writeNodeTree(sb, childNodes, depth+1)
		}
	}
}

func writeNodeDataMap(sb *strings.Builder, data map[string]interface{}, indent string) {
	for _, key := range sortedKeys(data) {
		value := data[key]
		if value == nil {
			continue
		}
		switch v := value.(type) {
		case map[string]interface{}:
			sb.WriteString(fmt.Sprintf("%s%s:\n", indent, key))
			writeNodeDataMap(sb, v, indent+"  ")
		case string:
			if len(v) > 300 {
				v = v[:300] + "..."
			}
			sb.WriteString(fmt.Sprintf("%s%s = %q\n", indent, key, v))
		case float64:
			if v == float64(int64(v)) {
				sb.WriteString(fmt.Sprintf("%s%s = %d\n", indent, key, int64(v)))
			} else {
				sb.WriteString(fmt.Sprintf("%s%s = %g\n", indent, key, v))
			}
		case bool:
			sb.WriteString(fmt.Sprintf("%s%s = %t\n", indent, key, v))
		default:
			sb.WriteString(fmt.Sprintf("%s%s = %q\n", indent, key, fmt.Sprintf("%v", v)))
		}
	}
}
