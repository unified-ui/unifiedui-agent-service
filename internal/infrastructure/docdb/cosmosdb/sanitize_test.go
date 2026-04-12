// Package cosmosdb provides unit tests for sanitization functions.
package cosmosdb

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSanitizeValue_ValidInputs(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"simple alphanumeric", "abc123", "abc123"},
		{"UUID format", "550e8400-e29b-41d4-a716-446655440000", "550e8400-e29b-41d4-a716-446655440000"},
		{"with dots", "tenant.id.123", "tenant.id.123"},
		{"with underscores", "tenant_id_123", "tenant_id_123"},
		{"with colons", "prefix:suffix", "prefix:suffix"},
		{"with at sign", "user@domain", "user@domain"},
		{"with plus", "tag+value", "tag+value"},
		{"mixed characters", "tenant-123_abc.def:ghi@jkl+mno", "tenant-123_abc.def:ghi@jkl+mno"},
		{"uppercase", "TENANT123", "TENANT123"},
		{"single character", "a", "a"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := sanitizeValue(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestSanitizeValue_InvalidInputs(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{"SQL injection attempt", "'; DROP TABLE users; --"},
		{"NoSQL injection $where", "{$where: 'this.a > 1'}"},
		{"NoSQL injection $gt", "{$gt: 1}"},
		{"script tag", "<script>alert('xss')</script>"},
		{"shell command", "$(rm -rf /)"},
		{"backticks", "`command`"},
		{"special chars", "a*b?c[d]e"},
		{"unicode", "тест"},
		{"emoji", "test🔥"},
		{"null byte", "test\x00value"},
		{"newline", "test\nvalue"},
		{"tab", "test\tvalue"},
		{"empty string", ""},
		{"spaces only", "   "},
		{"leading space", " test"},
		{"trailing space", "test "},
		{"curly braces", "{test}"},
		{"square brackets", "[test]"},
		{"parentheses", "(test)"},
		{"pipe", "a|b"},
		{"ampersand", "a&b"},
		{"semicolon", "a;b"},
		{"equals", "a=b"},
		{"quotes", `"test"`},
		{"single quotes", "'test'"},
		{"backslash", `test\value`},
		{"forward slash", "test/value"},
		{"hash", "test#value"},
		{"percent", "test%value"},
		{"caret", "test^value"},
		{"tilde", "test~value"},
		{"exclamation", "test!value"},
		{"question", "test?value"},
		{"asterisk", "test*value"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := sanitizeValue(tt.input)
			assert.Empty(t, result, "Expected empty string for input: %q", tt.input)
		})
	}
}

func TestSanitizeValue_LengthLimit(t *testing.T) {
	// Test string at exactly 512 characters (should pass)
	validLong := make([]byte, 512)
	for i := range validLong {
		validLong[i] = 'a'
	}
	result := sanitizeValue(string(validLong))
	assert.Equal(t, string(validLong), result)

	// Test string at 513 characters (should fail)
	invalidLong := make([]byte, 513)
	for i := range invalidLong {
		invalidLong[i] = 'a'
	}
	result = sanitizeValue(string(invalidLong))
	assert.Empty(t, result)
}

func TestSanitizeValue_EdgeCases(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"single dash", "-", "-"},
		{"single underscore", "_", "_"},
		{"single dot", ".", "."},
		{"single colon", ":", ":"},
		{"single at", "@", "@"},
		{"single plus", "+", "+"},
		{"numbers only", "12345", "12345"},
		{"start with number", "1abc", "1abc"},
		{"end with number", "abc1", "abc1"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := sanitizeValue(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}
