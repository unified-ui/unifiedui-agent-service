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
	// StreamTypeNewMessage indicates a new message starts in the stream (for Foundry multi-message responses).
	StreamTypeNewMessage StreamMessageType = "STREAM_NEW_MESSAGE"
	// StreamTypeMessageComplete indicates a complete message with all metadata (sent after STREAM_END).
	StreamTypeMessageComplete StreamMessageType = "MESSAGE_COMPLETE"
	// StreamTypeTitleGeneration indicates an AI-generated conversation title.
	StreamTypeTitleGeneration StreamMessageType = "TITLE_GENERATION"

	// ReACT Agent stream types

	// StreamTypeReasoningStart indicates the start of a reasoning/thinking step.
	StreamTypeReasoningStart StreamMessageType = "REASONING_START"
	// StreamTypeReasoningStream indicates a reasoning content chunk.
	StreamTypeReasoningStream StreamMessageType = "REASONING_STREAM"
	// StreamTypeReasoningEnd indicates the end of a reasoning step.
	StreamTypeReasoningEnd StreamMessageType = "REASONING_END"
	// StreamTypeToolCallStart indicates the start of a tool invocation.
	StreamTypeToolCallStart StreamMessageType = "TOOL_CALL_START"
	// StreamTypeToolCallStream indicates a tool call content chunk.
	StreamTypeToolCallStream StreamMessageType = "TOOL_CALL_STREAM"
	// StreamTypeToolCallEnd indicates the end of a tool invocation.
	StreamTypeToolCallEnd StreamMessageType = "TOOL_CALL_END"
	// StreamTypePlanStart indicates the start of a planning step.
	StreamTypePlanStart StreamMessageType = "PLAN_START"
	// StreamTypePlanStream indicates a plan content chunk.
	StreamTypePlanStream StreamMessageType = "PLAN_STREAM"
	// StreamTypePlanComplete indicates planning is complete.
	StreamTypePlanComplete StreamMessageType = "PLAN_COMPLETE"
	// StreamTypeSubAgentStart indicates the start of a sub-agent delegation.
	StreamTypeSubAgentStart StreamMessageType = "SUB_AGENT_START"
	// StreamTypeSubAgentStream indicates a sub-agent content chunk.
	StreamTypeSubAgentStream StreamMessageType = "SUB_AGENT_STREAM"
	// StreamTypeSubAgentEnd indicates the end of a sub-agent delegation.
	StreamTypeSubAgentEnd StreamMessageType = "SUB_AGENT_END"
	// StreamTypeSynthesisStart indicates the start of a synthesis step.
	StreamTypeSynthesisStart StreamMessageType = "SYNTHESIS_START"
	// StreamTypeSynthesisStream indicates a synthesis content chunk.
	StreamTypeSynthesisStream StreamMessageType = "SYNTHESIS_STREAM"
	// StreamTypeTrace indicates a trace event from the ReACT agent.
	StreamTypeTrace StreamMessageType = "TRACE"
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
func (w *Writer) WriteEventWithID(eventType EventType, id, data string) error {
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

// WriteMessageComplete writes the MESSAGE_COMPLETE message with full message data.
// This is sent after STREAM_END to provide the frontend with complete message metadata.
func (w *Writer) WriteMessageComplete(message interface{}) error {
	return w.WriteJSON(EventMessage, &StreamMessage{
		Type: StreamTypeMessageComplete,
		Config: map[string]interface{}{
			"message": message,
		},
	})
}

// WriteTitleGeneration writes a TITLE_GENERATION message with the AI-generated title.
func (w *Writer) WriteTitleGeneration(title string) error {
	return w.WriteJSON(EventMessage, &StreamMessage{
		Type:    StreamTypeTitleGeneration,
		Content: title,
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

// WriteReasoningStart writes a REASONING_START message.
func (w *Writer) WriteReasoningStart() error {
	return w.WriteJSON(EventMessage, &StreamMessage{
		Type:   StreamTypeReasoningStart,
		Config: map[string]interface{}{},
	})
}

// WriteReasoningStream writes a REASONING_STREAM message with content.
func (w *Writer) WriteReasoningStream(content string) error {
	return w.WriteJSON(EventMessage, &StreamMessage{
		Type:    StreamTypeReasoningStream,
		Content: content,
	})
}

// WriteReasoningEnd writes a REASONING_END message.
func (w *Writer) WriteReasoningEnd() error {
	return w.WriteJSON(EventMessage, &StreamMessage{
		Type:   StreamTypeReasoningEnd,
		Config: map[string]interface{}{},
	})
}

// WriteToolCallStart writes a TOOL_CALL_START message with tool metadata.
func (w *Writer) WriteToolCallStart(toolName string, config map[string]interface{}) error {
	if config == nil {
		config = map[string]interface{}{}
	}
	config["toolName"] = toolName
	if args, ok := config["tool_arguments"]; ok {
		config["toolInput"] = args
	}
	if ct, ok := config["call_type"]; ok {
		config["callType"] = ct
	}
	return w.WriteJSON(EventMessage, &StreamMessage{
		Type:   StreamTypeToolCallStart,
		Config: config,
	})
}

// WriteToolCallStream writes a TOOL_CALL_STREAM message with content.
func (w *Writer) WriteToolCallStream(content string) error {
	return w.WriteJSON(EventMessage, &StreamMessage{
		Type:    StreamTypeToolCallStream,
		Content: content,
	})
}

// WriteToolCallEnd writes a TOOL_CALL_END message with optional result config.
func (w *Writer) WriteToolCallEnd(config map[string]interface{}) error {
	if config == nil {
		config = map[string]interface{}{}
	}
	if result, ok := config["tool_result"]; ok {
		config["toolResult"] = result
	}
	if name, ok := config["tool_name"]; ok {
		config["toolName"] = name
	}
	if ct, ok := config["call_type"]; ok {
		config["callType"] = ct
	}
	return w.WriteJSON(EventMessage, &StreamMessage{
		Type:   StreamTypeToolCallEnd,
		Config: config,
	})
}

// WritePlanStart writes a PLAN_START message.
func (w *Writer) WritePlanStart() error {
	return w.WriteJSON(EventMessage, &StreamMessage{
		Type:   StreamTypePlanStart,
		Config: map[string]interface{}{},
	})
}

// WritePlanStream writes a PLAN_STREAM message with content.
func (w *Writer) WritePlanStream(content string) error {
	return w.WriteJSON(EventMessage, &StreamMessage{
		Type:    StreamTypePlanStream,
		Content: content,
	})
}

// WritePlanComplete writes a PLAN_COMPLETE message.
func (w *Writer) WritePlanComplete(config map[string]interface{}) error {
	if config == nil {
		config = map[string]interface{}{}
	}
	return w.WriteJSON(EventMessage, &StreamMessage{
		Type:   StreamTypePlanComplete,
		Config: config,
	})
}

// WriteSubAgentStart writes a SUB_AGENT_START message with agent name.
func (w *Writer) WriteSubAgentStart(agentName string, config map[string]interface{}) error {
	if config == nil {
		config = map[string]interface{}{}
	}
	config["agentName"] = agentName
	if id, ok := config["sub_agent_id"]; ok {
		config["agentId"] = id
	}
	return w.WriteJSON(EventMessage, &StreamMessage{
		Type:   StreamTypeSubAgentStart,
		Config: config,
	})
}

// WriteSubAgentStream writes a SUB_AGENT_STREAM message with content.
func (w *Writer) WriteSubAgentStream(content string) error {
	return w.WriteJSON(EventMessage, &StreamMessage{
		Type:    StreamTypeSubAgentStream,
		Content: content,
	})
}

// WriteSubAgentEnd writes a SUB_AGENT_END message.
func (w *Writer) WriteSubAgentEnd(config map[string]interface{}) error {
	if config == nil {
		config = map[string]interface{}{}
	}
	return w.WriteJSON(EventMessage, &StreamMessage{
		Type:   StreamTypeSubAgentEnd,
		Config: config,
	})
}

// WriteSynthesisStart writes a SYNTHESIS_START message.
func (w *Writer) WriteSynthesisStart() error {
	return w.WriteJSON(EventMessage, &StreamMessage{
		Type:   StreamTypeSynthesisStart,
		Config: map[string]interface{}{},
	})
}

// WriteSynthesisStream writes a SYNTHESIS_STREAM message with content.
func (w *Writer) WriteSynthesisStream(content string) error {
	return w.WriteJSON(EventMessage, &StreamMessage{
		Type:    StreamTypeSynthesisStream,
		Content: content,
	})
}

// WriteStreamTrace writes a TRACE message from the ReACT agent.
func (w *Writer) WriteStreamTrace(config map[string]interface{}) error {
	if config == nil {
		config = map[string]interface{}{}
	}
	return w.WriteJSON(EventMessage, &StreamMessage{
		Type:   StreamTypeTrace,
		Config: config,
	})
}

// WriteReActStreamMessage writes a generic ReACT stream message by type, content, and config.
func (w *Writer) WriteReActStreamMessage(streamType StreamMessageType, content string, config map[string]interface{}) error {
	return w.WriteJSON(EventMessage, &StreamMessage{
		Type:    streamType,
		Content: content,
		Config:  config,
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
func (w *Writer) WriteError(code, message, details string) error {
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
