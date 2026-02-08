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
