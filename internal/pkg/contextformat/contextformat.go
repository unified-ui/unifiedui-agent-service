// Package contextformat provides utilities for formatting context data.
package contextformat

import (
	"fmt"
	"sort"
	"strings"
)

// FormatContextDataAsTOML converts a map of context data to a token-optimized string format.
// Format: key1=value1;key2=value2
// The keys are sorted alphabetically for consistent output.
func FormatContextDataAsTOML(contextData map[string]string) string {
	if len(contextData) == 0 {
		return ""
	}

	// Sort keys for consistent output
	keys := make([]string, 0, len(contextData))
	for k := range contextData {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	// Build TOML-like string (key=value pairs separated by semicolons)
	parts := make([]string, 0, len(keys))
	for _, k := range keys {
		parts = append(parts, fmt.Sprintf("%s=%s", k, contextData[k]))
	}

	return strings.Join(parts, ";")
}

// WrapWithContextTag wraps a TOML string with <context> tags.
// Returns empty string if tomlStr is empty.
func WrapWithContextTag(tomlStr string) string {
	if tomlStr == "" {
		return ""
	}
	return fmt.Sprintf("<context>%s</context>", tomlStr)
}

// PrependContextToMessage converts context data to TOML format and prepends it to a message.
// Returns the original message if context data is empty.
// Format: <context>TOML_STR</context>\n\noriginal_message
func PrependContextToMessage(contextData map[string]string, message string) string {
	tomlStr := FormatContextDataAsTOML(contextData)
	if tomlStr == "" {
		return message
	}
	return fmt.Sprintf("<context>%s</context>\n\n%s", tomlStr, message)
}
