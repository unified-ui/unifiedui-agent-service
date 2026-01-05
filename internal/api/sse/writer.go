// Package sse provides Server-Sent Events support for streaming responses.
package sse

import (
	"encoding/json"
	"fmt"
	"net/http"
)

// EventType represents the type of SSE event.
type EventType string

const (
	// EventMessage is a chat message chunk event.
	EventMessage EventType = "message"
	// EventTrace is a trace update event.
	EventTrace EventType = "trace"
	// EventError is an error event.
	EventError EventType = "error"
	// EventDone is a stream completion event.
	EventDone EventType = "done"
)

// StreamMessageType represents the type of stream message in the data payload.
type StreamMessageType string

const (
	// StreamTypeStart indicates the start of a stream.
	StreamTypeStart StreamMessageType = "STREAM_START"
	// StreamTypeTextStream indicates a text content chunk.
	StreamTypeTextStream StreamMessageType = "TEXT_STREAM"
	// StreamTypeEnd indicates the end of a stream.
	StreamTypeEnd StreamMessageType = "STREAM_END"
	// StreamTypeError indicates an error in the stream.
	StreamTypeError StreamMessageType = "ERROR"
)

// StreamMessage represents a unified stream message format.
type StreamMessage struct {
	Type    StreamMessageType      `json:"type"`
	Content string                 `json:"content,omitempty"`
	Config  map[string]interface{} `json:"config,omitempty"`
}

// Writer writes Server-Sent Events to an HTTP response.
type Writer struct {
	writer  http.ResponseWriter
	flusher http.Flusher
}

// NewWriter creates a new SSE writer.
func NewWriter(w http.ResponseWriter) (*Writer, error) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		return nil, fmt.Errorf("streaming not supported")
	}

	// Set SSE headers
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no")

	return &Writer{
		writer:  w,
		flusher: flusher,
	}, nil
}

// WriteEvent writes an SSE event with the given type and data.
func (w *Writer) WriteEvent(eventType EventType, data string) error {
	_, err := fmt.Fprintf(w.writer, "event: %s\ndata: %s\n\n", eventType, data)
	if err != nil {
		return fmt.Errorf("failed to write event: %w", err)
	}
	w.flusher.Flush()
	return nil
}

// WriteEventWithID writes an SSE event with an ID.
func (w *Writer) WriteEventWithID(eventType EventType, id string, data string) error {
	_, err := fmt.Fprintf(w.writer, "id: %s\nevent: %s\ndata: %s\n\n", id, eventType, data)
	if err != nil {
		return fmt.Errorf("failed to write event with id: %w", err)
	}
	w.flusher.Flush()
	return nil
}

// WriteJSON writes an SSE event with JSON-encoded data.
func (w *Writer) WriteJSON(eventType EventType, data interface{}) error {
	jsonData, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("failed to marshal data: %w", err)
	}
	return w.WriteEvent(eventType, string(jsonData))
}

// WriteMessage writes a message event.
func (w *Writer) WriteMessage(content string) error {
	return w.WriteEvent(EventMessage, content)
}

// WriteStreamStart writes the STREAM_START message with messageId and conversationId.
func (w *Writer) WriteStreamStart(messageID, conversationID string) error {
	return w.WriteJSON(EventMessage, &StreamMessage{
		Type: StreamTypeStart,
		Config: map[string]interface{}{
			"messageId":      messageID,
			"conversationId": conversationID,
		},
	})
}

// WriteTextStream writes a TEXT_STREAM message with content.
func (w *Writer) WriteTextStream(content string) error {
	return w.WriteJSON(EventMessage, &StreamMessage{
		Type:    StreamTypeTextStream,
		Content: content,
	})
}

// WriteStreamEnd writes the STREAM_END message.
func (w *Writer) WriteStreamEnd() error {
	return w.WriteJSON(EventMessage, &StreamMessage{
		Type:   StreamTypeEnd,
		Config: map[string]interface{}{},
	})
}

// WriteStreamError writes an error message in stream format.
func (w *Writer) WriteStreamError(code, message, details string) error {
	return w.WriteJSON(EventMessage, &StreamMessage{
		Type: StreamTypeError,
		Config: map[string]interface{}{
			"code":    code,
			"message": message,
			"details": details,
		},
	})
}

// MessageChunk is kept for backward compatibility.
type MessageChunk struct {
	Content   string `json:"content"`
	MessageID string `json:"messageId,omitempty"`
	Done      bool   `json:"done"`
}

// WriteMessageChunk writes a message chunk event (legacy format).
func (w *Writer) WriteMessageChunk(chunk *MessageChunk) error {
	return w.WriteJSON(EventMessage, chunk)
}

// TraceEvent represents a trace update event.
type TraceEvent struct {
	TraceID string      `json:"traceId"`
	Type    string      `json:"type"`
	Name    string      `json:"name"`
	Status  string      `json:"status"`
	Data    interface{} `json:"data,omitempty"`
}

// WriteTrace writes a trace event.
func (w *Writer) WriteTrace(trace *TraceEvent) error {
	return w.WriteJSON(EventTrace, trace)
}

// ErrorEvent represents an error event.
type ErrorEvent struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Details string `json:"details,omitempty"`
}

// WriteError writes an error event.
func (w *Writer) WriteError(code, message string, details string) error {
	return w.WriteJSON(EventError, &ErrorEvent{
		Code:    code,
		Message: message,
		Details: details,
	})
}

// WriteDone writes a done event to signal stream completion.
func (w *Writer) WriteDone() error {
	return w.WriteEvent(EventDone, "stream completed")
}

// Flush flushes the response writer.
func (w *Writer) Flush() {
	w.flusher.Flush()
}
