package contextformat_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/unifiedui/agent-service/internal/pkg/contextformat"
)

func TestFormatContextDataAsTOML_Empty(t *testing.T) {
	result := contextformat.FormatContextDataAsTOML(nil)
	assert.Equal(t, "", result)
}

func TestFormatContextDataAsTOML_EmptyMap(t *testing.T) {
	result := contextformat.FormatContextDataAsTOML(map[string]string{})
	assert.Equal(t, "", result)
}

func TestFormatContextDataAsTOML_SingleEntry(t *testing.T) {
	result := contextformat.FormatContextDataAsTOML(map[string]string{"key": "value"})
	assert.Equal(t, "key=value", result)
}

func TestFormatContextDataAsTOML_MultipleEntries_Sorted(t *testing.T) {
	data := map[string]string{
		"zebra":  "z",
		"alpha":  "a",
		"middle": "m",
	}
	result := contextformat.FormatContextDataAsTOML(data)
	assert.Equal(t, "alpha=a;middle=m;zebra=z", result)
}

func TestWrapWithContextTag_Empty(t *testing.T) {
	result := contextformat.WrapWithContextTag("")
	assert.Equal(t, "", result)
}

func TestWrapWithContextTag_NonEmpty(t *testing.T) {
	result := contextformat.WrapWithContextTag("key=value")
	assert.Equal(t, "<context>key=value</context>", result)
}

func TestPrependContextToMessage_EmptyContext(t *testing.T) {
	result := contextformat.PrependContextToMessage(nil, "hello")
	assert.Equal(t, "hello", result)
}

func TestPrependContextToMessage_WithContext(t *testing.T) {
	data := map[string]string{"env": "prod"}
	result := contextformat.PrependContextToMessage(data, "hello")
	assert.Equal(t, "<context>env=prod</context>\n\nhello", result)
}

// Additional comprehensive tests for contextformat

func TestFormatContextDataAsTOML_SpecialCharacters(t *testing.T) {
	data := map[string]string{
		"key_with_underscore": "value",
		"key-with-dash":       "value-dash",
		"key.with.dot":        "value.dot",
	}
	result := contextformat.FormatContextDataAsTOML(data)
	// Keys should be sorted alphabetically
	assert.Equal(t, "key-with-dash=value-dash;key.with.dot=value.dot;key_with_underscore=value", result)
}

func TestFormatContextDataAsTOML_NumericValues(t *testing.T) {
	data := map[string]string{
		"count":   "123",
		"percent": "99.5",
		"version": "1.0.0",
	}
	result := contextformat.FormatContextDataAsTOML(data)
	assert.Equal(t, "count=123;percent=99.5;version=1.0.0", result)
}

func TestFormatContextDataAsTOML_EmptyValues(t *testing.T) {
	data := map[string]string{
		"empty":    "",
		"nonempty": "value",
	}
	result := contextformat.FormatContextDataAsTOML(data)
	assert.Equal(t, "empty=;nonempty=value", result)
}

func TestFormatContextDataAsTOML_LongValues(t *testing.T) {
	longValue := "This is a very long value that contains multiple words and sentences. It should be preserved exactly as-is."
	data := map[string]string{"description": longValue}
	result := contextformat.FormatContextDataAsTOML(data)
	assert.Equal(t, "description="+longValue, result)
}

func TestFormatContextDataAsTOML_UnicodeValues(t *testing.T) {
	data := map[string]string{
		"greeting": "Héllo Wörld 你好",
		"emoji":    "🚀",
	}
	result := contextformat.FormatContextDataAsTOML(data)
	assert.Contains(t, result, "greeting=Héllo Wörld 你好")
	assert.Contains(t, result, "emoji=🚀")
}

func TestWrapWithContextTag_WithSpecialCharacters(t *testing.T) {
	result := contextformat.WrapWithContextTag("key=value<>&\"'")
	assert.Equal(t, "<context>key=value<>&\"'</context>", result)
}

func TestWrapWithContextTag_WithNewlines(t *testing.T) {
	result := contextformat.WrapWithContextTag("key=line1\nline2")
	assert.Equal(t, "<context>key=line1\nline2</context>", result)
}

func TestPrependContextToMessage_EmptyMapContext(t *testing.T) {
	result := contextformat.PrependContextToMessage(map[string]string{}, "hello")
	assert.Equal(t, "hello", result)
}

func TestPrependContextToMessage_EmptyMessage(t *testing.T) {
	data := map[string]string{"env": "prod"}
	result := contextformat.PrependContextToMessage(data, "")
	assert.Equal(t, "<context>env=prod</context>\n\n", result)
}

func TestPrependContextToMessage_MultipleContextEntries(t *testing.T) {
	data := map[string]string{
		"env":     "production",
		"region":  "us-west-2",
		"service": "api",
	}
	result := contextformat.PrependContextToMessage(data, "Process request")
	assert.Equal(t, "<context>env=production;region=us-west-2;service=api</context>\n\nProcess request", result)
}

func TestPrependContextToMessage_MessageWithNewlines(t *testing.T) {
	data := map[string]string{"user": "alice"}
	message := "Line 1\nLine 2\nLine 3"
	result := contextformat.PrependContextToMessage(data, message)
	expected := "<context>user=alice</context>\n\nLine 1\nLine 2\nLine 3"
	assert.Equal(t, expected, result)
}

func TestPrependContextToMessage_IntegrationWithFormatAndWrap(t *testing.T) {
	// Test that PrependContextToMessage produces the same result as manual composition
	data := map[string]string{"key": "value"}
	message := "test message"

	// Using PrependContextToMessage
	result1 := contextformat.PrependContextToMessage(data, message)

	// Manual composition
	toml := contextformat.FormatContextDataAsTOML(data)
	wrapped := contextformat.WrapWithContextTag(toml)
	result2 := wrapped + "\n\n" + message

	assert.Equal(t, result1, result2)
}
