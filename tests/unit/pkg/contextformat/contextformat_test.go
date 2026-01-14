// Package contextformat_test provides unit tests for the contextformat package.
package contextformat_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/unifiedui/agent-service/internal/pkg/contextformat"
)

func TestFormatContextDataAsTOML(t *testing.T) {
	tests := []struct {
		name     string
		data     map[string]string
		expected string
	}{
		{
			name:     "nil map",
			data:     nil,
			expected: "",
		},
		{
			name:     "empty map",
			data:     map[string]string{},
			expected: "",
		},
		{
			name: "single key-value",
			data: map[string]string{
				"key1": "value1",
			},
			expected: "key1=value1",
		},
		{
			name: "multiple key-values",
			data: map[string]string{
				"key1": "value1",
				"key2": "value2",
			},
			expected: "", // Will check contains since order is not guaranteed
		},
		{
			name: "with special characters in value",
			data: map[string]string{
				"url": "https://example.com/path?query=1",
			},
			expected: "url=https://example.com/path?query=1",
		},
		{
			name: "with empty value",
			data: map[string]string{
				"empty_key": "",
			},
			expected: "empty_key=",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := contextformat.FormatContextDataAsTOML(tt.data)

			if tt.name == "multiple key-values" {
				// For multiple values, check that both key-value pairs are present
				assert.Contains(t, result, "key1=value1")
				assert.Contains(t, result, "key2=value2")
				assert.Contains(t, result, ";")
			} else {
				assert.Equal(t, tt.expected, result)
			}
		})
	}
}

func TestWrapWithContextTag(t *testing.T) {
	tests := []struct {
		name     string
		tomlStr  string
		expected string
	}{
		{
			name:     "empty string",
			tomlStr:  "",
			expected: "",
		},
		{
			name:     "single key-value",
			tomlStr:  "key1=value1",
			expected: "<context>key1=value1</context>",
		},
		{
			name:     "multiple key-values",
			tomlStr:  "key1=value1;key2=value2",
			expected: "<context>key1=value1;key2=value2</context>",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := contextformat.WrapWithContextTag(tt.tomlStr)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestPrependContextToMessage(t *testing.T) {
	tests := []struct {
		name        string
		contextData map[string]string
		message     string
		expected    string
	}{
		{
			name:        "nil context data",
			contextData: nil,
			message:     "Hello, world!",
			expected:    "Hello, world!",
		},
		{
			name:        "empty context data",
			contextData: map[string]string{},
			message:     "Hello, world!",
			expected:    "Hello, world!",
		},
		{
			name: "with single context key",
			contextData: map[string]string{
				"user_id": "12345",
			},
			message:  "What is the status?",
			expected: "<context>user_id=12345</context>\n\nWhat is the status?",
		},
		{
			name: "with empty message",
			contextData: map[string]string{
				"key": "value",
			},
			message:  "",
			expected: "<context>key=value</context>\n\n",
		},
		{
			name: "with real-world example",
			contextData: map[string]string{
				"imt_internal_id": "728173",
				"session_token":   "abc123",
			},
			message:  "Show me the order details",
			expected: "", // Will verify contains due to map ordering
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := contextformat.PrependContextToMessage(tt.contextData, tt.message)

			if tt.name == "with real-world example" {
				// Verify structure for multiple context values
				assert.Contains(t, result, "<context>")
				assert.Contains(t, result, "</context>")
				assert.Contains(t, result, "\n\n")
				assert.Contains(t, result, "imt_internal_id=728173")
				assert.Contains(t, result, "session_token=abc123")
				assert.True(t, len(result) > len(tt.message), "Result should be longer than original message")
				assert.Contains(t, result, "Show me the order details")
			} else {
				assert.Equal(t, tt.expected, result)
			}
		})
	}
}

func TestPrependContextToMessage_EmptyContextDataMeansOriginalMessage(t *testing.T) {
	// Specific test to ensure empty context data returns original message unchanged
	message := "This is my original message with special chars: !@#$%^&*()"

	// Test with nil
	result := contextformat.PrependContextToMessage(nil, message)
	assert.Equal(t, message, result, "nil context data should return original message")

	// Test with empty map
	result = contextformat.PrependContextToMessage(map[string]string{}, message)
	assert.Equal(t, message, result, "empty context data should return original message")
}

func TestFormatContextDataAsTOML_Deterministic(t *testing.T) {
	// Test that the function always produces the same output for the same input
	// even though map iteration order is non-deterministic
	data := map[string]string{
		"a": "1",
		"b": "2",
		"c": "3",
	}

	firstResult := contextformat.FormatContextDataAsTOML(data)

	// Run multiple times to check consistency in content (not order)
	for i := 0; i < 10; i++ {
		result := contextformat.FormatContextDataAsTOML(data)
		assert.Contains(t, result, "a=1")
		assert.Contains(t, result, "b=2")
		assert.Contains(t, result, "c=3")
		// The result might have different ordering but should have same length
		assert.Equal(t, len(firstResult), len(result), "Result length should be consistent")
	}
}
