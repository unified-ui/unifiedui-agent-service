package sse

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/unifiedui/agent-service/internal/api/sse"

	"github.com/stretchr/testify/require"
)

// nonFlusher is a ResponseWriter that doesn't implement http.Flusher
type nonFlusher struct {
	header http.Header
}

func (n *nonFlusher) Header() http.Header {
	if n.header == nil {
		n.header = make(http.Header)
	}
	return n.header
}

func (n *nonFlusher) Write(b []byte) (int, error) {
	return len(b), nil
}

func (n *nonFlusher) WriteHeader(statusCode int) {}

func TestNewWriter_NoFlusherSupport(t *testing.T) {
	nf := &nonFlusher{}
	w, err := sse.NewWriter(nf)
	require.Error(t, err)
	require.Nil(t, w)
	require.Contains(t, err.Error(), "streaming not supported")
}

// errorWriter is a ResponseWriter that returns errors on Write
type errorWriter struct {
	*httptest.ResponseRecorder
	writeErr error
}

func (e *errorWriter) Write(b []byte) (int, error) {
	if e.writeErr != nil {
		return 0, e.writeErr
	}
	return e.ResponseRecorder.Write(b)
}

func (e *errorWriter) Flush() {}

func TestWriter_WriteEvent_Error(t *testing.T) {
	ew := &errorWriter{
		ResponseRecorder: httptest.NewRecorder(),
		writeErr:         nil,
	}

	// First create the writer successfully
	w, err := sse.NewWriter(ew)
	require.NoError(t, err)

	// Then set the error for subsequent writes
	ew.writeErr = &mockWriteError{}
	err = w.WriteEvent(sse.EventMessage, "test")
	require.Error(t, err)
	require.Contains(t, err.Error(), "failed to write event")
}

func TestWriter_WriteEventWithID_Error(t *testing.T) {
	ew := &errorWriter{
		ResponseRecorder: httptest.NewRecorder(),
		writeErr:         nil,
	}

	w, err := sse.NewWriter(ew)
	require.NoError(t, err)

	ew.writeErr = &mockWriteError{}
	err = w.WriteEventWithID(sse.EventMessage, "id-1", "test")
	require.Error(t, err)
	require.Contains(t, err.Error(), "failed to write event with id")
}

type mockWriteError struct{}

func (m *mockWriteError) Error() string {
	return "mock write error"
}

// unmarshalableType cannot be marshaled to JSON
type unmarshalableType struct {
	Ch chan int
}

func TestWriter_WriteJSON_MarshalError(t *testing.T) {
	f := newMockFlusher()
	w, _ := sse.NewWriter(f)

	// Channel types cannot be marshaled to JSON
	err := w.WriteJSON(sse.EventMessage, &unmarshalableType{Ch: make(chan int)})
	require.Error(t, err)
	require.Contains(t, err.Error(), "failed to marshal data")
}

// Test SSE format correctness
func TestWriter_SSEFormat_Correct(t *testing.T) {
	f := newMockFlusher()
	w, _ := sse.NewWriter(f)

	err := w.WriteEvent(sse.EventMessage, "test data")
	require.NoError(t, err)

	body := f.Body.String()
	// SSE format should be: event: <type>\ndata: <data>\n\n
	require.True(t, strings.HasPrefix(body, "event: message\n"))
	require.Contains(t, body, "data: test data\n\n")
}

func TestWriter_SSEFormatWithID_Correct(t *testing.T) {
	f := newMockFlusher()
	w, _ := sse.NewWriter(f)

	err := w.WriteEventWithID(sse.EventTrace, "trace-123", "trace data")
	require.NoError(t, err)

	body := f.Body.String()
	// SSE format with ID should be: id: <id>\nevent: <type>\ndata: <data>\n\n
	require.True(t, strings.HasPrefix(body, "id: trace-123\n"))
	require.Contains(t, body, "event: trace\n")
	require.Contains(t, body, "data: trace data\n\n")
}

// Test special characters in content
func TestWriter_WriteTextStream_SpecialCharacters(t *testing.T) {
	tests := []struct {
		name    string
		content string
	}{
		{"empty string", ""},
		{"newlines", "line1\nline2\nline3"},
		{"tabs", "col1\tcol2\tcol3"},
		{"unicode", "Hello 世界 🌍 مرحبا"},
		{"quotes", `"quoted" and 'single'`},
		{"backslashes", `path\to\file`},
		{"html entities", "<script>alert('xss')</script>"},
		{"json chars", `{"key": "value", "arr": [1,2,3]}`},
		{"long content", strings.Repeat("a", 10000)},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := newMockFlusher()
			w, _ := sse.NewWriter(f)

			err := w.WriteTextStream(tt.content)
			require.NoError(t, err)

			// Parse SSE data line
			body := f.Body.String()
			lines := strings.Split(body, "\n")
			var dataLine string
			for _, line := range lines {
				if strings.HasPrefix(line, "data: ") {
					dataLine = strings.TrimPrefix(line, "data: ")
					break
				}
			}

			// Verify JSON structure
			var msg sse.StreamMessage
			err = json.Unmarshal([]byte(dataLine), &msg)
			require.NoError(t, err)
			require.Equal(t, sse.StreamTypeTextStream, msg.Type)
			require.Equal(t, tt.content, msg.Content)
		})
	}
}

// Test multiple events in sequence
func TestWriter_MultipleEventsSequence(t *testing.T) {
	f := newMockFlusher()
	w, _ := sse.NewWriter(f)

	// Simulate a complete message stream
	require.NoError(t, w.WriteStreamStart("msg-1", "conv-1"))
	require.NoError(t, w.WriteTextStream("Hello "))
	require.NoError(t, w.WriteTextStream("World"))
	require.NoError(t, w.WriteTextStream("!"))
	require.NoError(t, w.WriteStreamEnd())
	require.NoError(t, w.WriteMessageComplete(map[string]interface{}{
		"id":      "msg-1",
		"content": "Hello World!",
	}))
	require.NoError(t, w.WriteDone())

	body := f.Body.String()

	// Count event occurrences
	messageCount := strings.Count(body, "event: message")
	doneCount := strings.Count(body, "event: done")

	// 6 message events: start, 3 text streams, end, complete
	require.Equal(t, 6, messageCount)
	require.Equal(t, 1, doneCount)

	// Verify flush count (one per event)
	require.Equal(t, 7, f.flushed)
}

// Test StreamMessage JSON structure
func TestStreamMessage_JSONSerialization(t *testing.T) {
	tests := []struct {
		name     string
		msg      sse.StreamMessage
		expected map[string]interface{}
	}{
		{
			name: "stream start",
			msg: sse.StreamMessage{
				Type: sse.StreamTypeStart,
				Config: map[string]interface{}{
					"messageId":      "m1",
					"conversationId": "c1",
				},
			},
			expected: map[string]interface{}{
				"type": "STREAM_START",
				"config": map[string]interface{}{
					"messageId":      "m1",
					"conversationId": "c1",
				},
			},
		},
		{
			name: "text stream with content",
			msg: sse.StreamMessage{
				Type:    sse.StreamTypeTextStream,
				Content: "Hello",
			},
			expected: map[string]interface{}{
				"type":    "TEXT_STREAM",
				"content": "Hello",
			},
		},
		{
			name: "stream end",
			msg: sse.StreamMessage{
				Type:   sse.StreamTypeEnd,
				Config: map[string]interface{}{},
			},
			expected: map[string]interface{}{
				"type":   "STREAM_END",
				"config": map[string]interface{}{},
			},
		},
		{
			name: "error with details",
			msg: sse.StreamMessage{
				Type: sse.StreamTypeError,
				Config: map[string]interface{}{
					"code":    "TIMEOUT",
					"message": "Request timed out",
					"details": "Operation exceeded 30s limit",
				},
			},
			expected: map[string]interface{}{
				"type": "ERROR",
				"config": map[string]interface{}{
					"code":    "TIMEOUT",
					"message": "Request timed out",
					"details": "Operation exceeded 30s limit",
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data, err := json.Marshal(tt.msg)
			require.NoError(t, err)

			var parsed map[string]interface{}
			require.NoError(t, json.Unmarshal(data, &parsed))

			require.Equal(t, tt.expected["type"], parsed["type"])
			if tt.expected["content"] != nil {
				require.Equal(t, tt.expected["content"], parsed["content"])
			}
			if expectedConfig, ok := tt.expected["config"].(map[string]interface{}); ok && len(expectedConfig) > 0 {
				parsedConfig, ok := parsed["config"].(map[string]interface{})
				require.True(t, ok, "expected config to be a map")
				for k, v := range expectedConfig {
					require.Equal(t, v, parsedConfig[k])
				}
			}
		})
	}
}

// Test TraceEvent JSON structure
func TestTraceEvent_JSONSerialization(t *testing.T) {
	trace := sse.TraceEvent{
		TraceID: "trace-abc",
		Type:    "llm",
		Name:    "GPT-4 Call",
		Status:  "completed",
		Data: map[string]interface{}{
			"tokensUsed": 150,
			"model":      "gpt-4",
		},
	}

	data, err := json.Marshal(trace)
	require.NoError(t, err)

	var parsed map[string]interface{}
	require.NoError(t, json.Unmarshal(data, &parsed))

	require.Equal(t, "trace-abc", parsed["traceId"])
	require.Equal(t, "llm", parsed["type"])
	require.Equal(t, "GPT-4 Call", parsed["name"])
	require.Equal(t, "completed", parsed["status"])
	require.NotNil(t, parsed["data"])
}

// Test ErrorEvent JSON structure
func TestErrorEvent_JSONSerialization(t *testing.T) {
	errEvent := sse.ErrorEvent{
		Code:    "INTERNAL_ERROR",
		Message: "An error occurred",
		Details: "Stack trace here",
	}

	data, err := json.Marshal(errEvent)
	require.NoError(t, err)

	var parsed map[string]interface{}
	require.NoError(t, json.Unmarshal(data, &parsed))

	require.Equal(t, "INTERNAL_ERROR", parsed["code"])
	require.Equal(t, "An error occurred", parsed["message"])
	require.Equal(t, "Stack trace here", parsed["details"])
}

// Test MessageChunk JSON structure (legacy format)
func TestMessageChunk_JSONSerialization(t *testing.T) {
	chunk := sse.MessageChunk{
		Content:   "Hello world",
		MessageID: "msg-123",
		Done:      false,
	}

	data, err := json.Marshal(chunk)
	require.NoError(t, err)

	var parsed map[string]interface{}
	require.NoError(t, json.Unmarshal(data, &parsed))

	require.Equal(t, "Hello world", parsed["content"])
	require.Equal(t, "msg-123", parsed["messageId"])
	require.Equal(t, false, parsed["done"])
}

// Test all headers set correctly
func TestNewWriter_HeadersSet(t *testing.T) {
	f := newMockFlusher()
	_, err := sse.NewWriter(f)
	require.NoError(t, err)

	headers := f.Header()
	require.Equal(t, "text/event-stream", headers.Get("Content-Type"))
	require.Equal(t, "no-cache", headers.Get("Cache-Control"))
	require.Equal(t, "keep-alive", headers.Get("Connection"))
	require.Equal(t, "no", headers.Get("X-Accel-Buffering"))
}

// Test title generation
func TestWriter_WriteTitleGeneration_EdgeCases(t *testing.T) {
	tests := []struct {
		name  string
		title string
	}{
		{"empty title", ""},
		{"short title", "Hi"},
		{"long title", strings.Repeat("Very Long Title ", 100)},
		{"title with special chars", "Title: with <special> & \"chars\""},
		{"unicode title", "聊天标题 🗣️"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := newMockFlusher()
			w, _ := sse.NewWriter(f)

			err := w.WriteTitleGeneration(tt.title)
			require.NoError(t, err)

			body := f.Body.String()
			require.Contains(t, body, "TITLE_GENERATION")

			// Parse and verify
			lines := strings.Split(body, "\n")
			var dataLine string
			for _, line := range lines {
				if strings.HasPrefix(line, "data: ") {
					dataLine = strings.TrimPrefix(line, "data: ")
					break
				}
			}

			var msg sse.StreamMessage
			require.NoError(t, json.Unmarshal([]byte(dataLine), &msg))
			require.Equal(t, tt.title, msg.Content)
		})
	}
}

// Test WriteStreamError with various error scenarios
func TestWriter_WriteStreamError_EdgeCases(t *testing.T) {
	tests := []struct {
		name    string
		code    string
		message string
		details string
	}{
		{"basic error", "ERROR", "Something went wrong", "Details here"},
		{"empty details", "NOT_FOUND", "Resource not found", ""},
		{"all empty", "", "", ""},
		{"long error", "LONG_ERROR", strings.Repeat("error ", 500), strings.Repeat("detail ", 500)},
		{"special chars", "PARSE_ERROR", "Invalid JSON: {\"bad\": }", "<stack>trace</stack>"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := newMockFlusher()
			w, _ := sse.NewWriter(f)

			err := w.WriteStreamError(tt.code, tt.message, tt.details)
			require.NoError(t, err)

			body := f.Body.String()

			// Parse and verify
			lines := strings.Split(body, "\n")
			var dataLine string
			for _, line := range lines {
				if strings.HasPrefix(line, "data: ") {
					dataLine = strings.TrimPrefix(line, "data: ")
					break
				}
			}

			var msg sse.StreamMessage
			require.NoError(t, json.Unmarshal([]byte(dataLine), &msg))
			require.Equal(t, sse.StreamTypeError, msg.Type)
			require.Equal(t, tt.code, msg.Config["code"])
			require.Equal(t, tt.message, msg.Config["message"])
			require.Equal(t, tt.details, msg.Config["details"])
		})
	}
}
