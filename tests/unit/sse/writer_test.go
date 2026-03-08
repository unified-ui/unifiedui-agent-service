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

type mockFlusher struct {
	*httptest.ResponseRecorder
	flushed int
}

func (f *mockFlusher) Flush() {
	f.flushed++
}

func newMockFlusher() *mockFlusher {
	return &mockFlusher{ResponseRecorder: httptest.NewRecorder()}
}

func TestNewWriter_Success(t *testing.T) {
	f := newMockFlusher()
	w, err := sse.NewWriter(f)
	require.NoError(t, err)
	require.NotNil(t, w)
	require.Equal(t, "text/event-stream", f.Header().Get("Content-Type"))
	require.Equal(t, "no-cache", f.Header().Get("Cache-Control"))
}

func TestWriter_WriteEvent(t *testing.T) {
	f := newMockFlusher()
	w, _ := sse.NewWriter(f)

	err := w.WriteEvent(sse.EventMessage, "hello world")
	require.NoError(t, err)
	require.Contains(t, f.Body.String(), "event: message")
	require.Contains(t, f.Body.String(), "data: hello world")
	require.Equal(t, 1, f.flushed)
}

func TestWriter_WriteEventWithID(t *testing.T) {
	f := newMockFlusher()
	w, _ := sse.NewWriter(f)

	err := w.WriteEventWithID(sse.EventMessage, "evt-1", "data here")
	require.NoError(t, err)
	require.Contains(t, f.Body.String(), "id: evt-1")
	require.Contains(t, f.Body.String(), "event: message")
	require.Contains(t, f.Body.String(), "data: data here")
}

func TestWriter_WriteJSON(t *testing.T) {
	f := newMockFlusher()
	w, _ := sse.NewWriter(f)

	data := map[string]string{"key": "value"}
	err := w.WriteJSON(sse.EventMessage, data)
	require.NoError(t, err)

	body := f.Body.String()
	lines := strings.Split(body, "\n")
	var dataLine string
	for _, line := range lines {
		if strings.HasPrefix(line, "data: ") {
			dataLine = strings.TrimPrefix(line, "data: ")
		}
	}
	var parsed map[string]string
	require.NoError(t, json.Unmarshal([]byte(dataLine), &parsed))
	require.Equal(t, "value", parsed["key"])
}

func TestWriter_WriteMessage(t *testing.T) {
	f := newMockFlusher()
	w, _ := sse.NewWriter(f)

	err := w.WriteMessage("chunk content")
	require.NoError(t, err)
	require.Contains(t, f.Body.String(), "chunk content")
}

func TestWriter_WriteStreamStart(t *testing.T) {
	f := newMockFlusher()
	w, _ := sse.NewWriter(f)

	err := w.WriteStreamStart("msg-1", "conv-1")
	require.NoError(t, err)
	require.Contains(t, f.Body.String(), "STREAM_START")
	require.Contains(t, f.Body.String(), "msg-1")
	require.Contains(t, f.Body.String(), "conv-1")
}

func TestWriter_WriteTextStream(t *testing.T) {
	f := newMockFlusher()
	w, _ := sse.NewWriter(f)

	err := w.WriteTextStream("Hello, ")
	require.NoError(t, err)
	require.Contains(t, f.Body.String(), "TEXT_STREAM")
	require.Contains(t, f.Body.String(), "Hello, ")
}

func TestWriter_WriteStreamEnd(t *testing.T) {
	f := newMockFlusher()
	w, _ := sse.NewWriter(f)

	err := w.WriteStreamEnd()
	require.NoError(t, err)
	require.Contains(t, f.Body.String(), "STREAM_END")
}

func TestWriter_WriteMessageComplete(t *testing.T) {
	f := newMockFlusher()
	w, _ := sse.NewWriter(f)

	msg := map[string]string{"id": "msg-1", "content": "hi"}
	err := w.WriteMessageComplete(msg)
	require.NoError(t, err)
	require.Contains(t, f.Body.String(), "MESSAGE_COMPLETE")
}

func TestWriter_WriteTitleGeneration(t *testing.T) {
	f := newMockFlusher()
	w, _ := sse.NewWriter(f)

	err := w.WriteTitleGeneration("My Conversation Title")
	require.NoError(t, err)
	require.Contains(t, f.Body.String(), "TITLE_GENERATION")
	require.Contains(t, f.Body.String(), "My Conversation Title")
}

func TestWriter_WriteStreamError(t *testing.T) {
	f := newMockFlusher()
	w, _ := sse.NewWriter(f)

	err := w.WriteStreamError("TIMEOUT", "request timed out", "took >30s")
	require.NoError(t, err)
	require.Contains(t, f.Body.String(), "ERROR")
	require.Contains(t, f.Body.String(), "TIMEOUT")
}

func TestWriter_WriteMessageChunk(t *testing.T) {
	f := newMockFlusher()
	w, _ := sse.NewWriter(f)

	err := w.WriteMessageChunk(&sse.MessageChunk{Content: "chunk", MessageID: "m1", Done: false})
	require.NoError(t, err)
	require.Contains(t, f.Body.String(), "chunk")
}

func TestWriter_WriteTrace(t *testing.T) {
	f := newMockFlusher()
	w, _ := sse.NewWriter(f)

	err := w.WriteTrace(&sse.TraceEvent{TraceID: "t-1", Type: "llm", Name: "LLM Call", Status: "completed"})
	require.NoError(t, err)
	require.Contains(t, f.Body.String(), "event: trace")
	require.Contains(t, f.Body.String(), "t-1")
}

func TestWriter_WriteError(t *testing.T) {
	f := newMockFlusher()
	w, _ := sse.NewWriter(f)

	err := w.WriteError("INTERNAL_ERROR", "something broke", "details")
	require.NoError(t, err)
	require.Contains(t, f.Body.String(), "event: error")
	require.Contains(t, f.Body.String(), "INTERNAL_ERROR")
}

func TestWriter_WriteDone(t *testing.T) {
	f := newMockFlusher()
	w, _ := sse.NewWriter(f)

	err := w.WriteDone()
	require.NoError(t, err)
	require.Contains(t, f.Body.String(), "event: done")
	require.Contains(t, f.Body.String(), "stream completed")
}

func TestWriter_Flush(t *testing.T) {
	f := newMockFlusher()
	w, _ := sse.NewWriter(f)

	w.Flush()
	require.Equal(t, 1, f.flushed)
}

func TestEventTypes(t *testing.T) {
	require.Equal(t, sse.EventType("message"), sse.EventMessage)
	require.Equal(t, sse.EventType("trace"), sse.EventTrace)
	require.Equal(t, sse.EventType("error"), sse.EventError)
	require.Equal(t, sse.EventType("done"), sse.EventDone)
}

func TestStreamMessageTypes(t *testing.T) {
	require.Equal(t, sse.StreamMessageType("STREAM_START"), sse.StreamTypeStart)
	require.Equal(t, sse.StreamMessageType("TEXT_STREAM"), sse.StreamTypeTextStream)
	require.Equal(t, sse.StreamMessageType("STREAM_END"), sse.StreamTypeEnd)
	require.Equal(t, sse.StreamMessageType("ERROR"), sse.StreamTypeError)
	require.Equal(t, sse.StreamMessageType("STREAM_NEW_MESSAGE"), sse.StreamTypeNewMessage)
	require.Equal(t, sse.StreamMessageType("MESSAGE_COMPLETE"), sse.StreamTypeMessageComplete)
	require.Equal(t, sse.StreamMessageType("TITLE_GENERATION"), sse.StreamTypeTitleGeneration)
}

func TestWriter_Integration_FullStream(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		writer, err := sse.NewWriter(w)
		require.NoError(t, err)

		require.NoError(t, writer.WriteStreamStart("msg-1", "conv-1"))
		require.NoError(t, writer.WriteTextStream("Hello"))
		require.NoError(t, writer.WriteTextStream(" World"))
		require.NoError(t, writer.WriteStreamEnd())
		require.NoError(t, writer.WriteDone())
	})

	server := httptest.NewServer(handler)
	defer server.Close()

	resp, err := http.Get(server.URL)
	require.NoError(t, err)
	defer resp.Body.Close()

	require.Equal(t, "text/event-stream", resp.Header.Get("Content-Type"))
}
