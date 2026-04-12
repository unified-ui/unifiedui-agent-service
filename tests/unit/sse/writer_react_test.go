package sse

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/unifiedui/agent-service/internal/api/sse"
)

func parseSSEDataLine(t *testing.T, body string) sse.StreamMessage {
	t.Helper()
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
	return msg
}

func TestWriter_WriteReasoningStart(t *testing.T) {
	f := newMockFlusher()
	w, _ := sse.NewWriter(f)
	err := w.WriteReasoningStart()
	require.NoError(t, err)
	msg := parseSSEDataLine(t, f.Body.String())
	require.Equal(t, sse.StreamTypeReasoningStart, msg.Type)
}

func TestWriter_WriteReasoningStream(t *testing.T) {
	f := newMockFlusher()
	w, _ := sse.NewWriter(f)
	err := w.WriteReasoningStream("thinking about the problem...")
	require.NoError(t, err)
	msg := parseSSEDataLine(t, f.Body.String())
	require.Equal(t, sse.StreamTypeReasoningStream, msg.Type)
	require.Equal(t, "thinking about the problem...", msg.Content)
}

func TestWriter_WriteReasoningEnd(t *testing.T) {
	f := newMockFlusher()
	w, _ := sse.NewWriter(f)
	err := w.WriteReasoningEnd()
	require.NoError(t, err)
	msg := parseSSEDataLine(t, f.Body.String())
	require.Equal(t, sse.StreamTypeReasoningEnd, msg.Type)
}

func TestWriter_WriteToolCallStart(t *testing.T) {
	f := newMockFlusher()
	w, _ := sse.NewWriter(f)
	err := w.WriteToolCallStart("search_web", nil)
	require.NoError(t, err)
	msg := parseSSEDataLine(t, f.Body.String())
	require.Equal(t, sse.StreamTypeToolCallStart, msg.Type)
	require.Equal(t, "search_web", msg.Config["toolName"])
}

func TestWriter_WriteToolCallStart_WithConfig(t *testing.T) {
	f := newMockFlusher()
	w, _ := sse.NewWriter(f)
	config := map[string]interface{}{"timeout": 30}
	err := w.WriteToolCallStart("calculator", config)
	require.NoError(t, err)
	msg := parseSSEDataLine(t, f.Body.String())
	require.Equal(t, sse.StreamTypeToolCallStart, msg.Type)
	require.Equal(t, "calculator", msg.Config["toolName"])
	require.Equal(t, float64(30), msg.Config["timeout"])
}

func TestWriter_WriteToolCallStream(t *testing.T) {
	f := newMockFlusher()
	w, _ := sse.NewWriter(f)
	err := w.WriteToolCallStream("executing query...")
	require.NoError(t, err)
	msg := parseSSEDataLine(t, f.Body.String())
	require.Equal(t, sse.StreamTypeToolCallStream, msg.Type)
	require.Equal(t, "executing query...", msg.Content)
}

func TestWriter_WriteToolCallEnd(t *testing.T) {
	f := newMockFlusher()
	w, _ := sse.NewWriter(f)
	err := w.WriteToolCallEnd(nil)
	require.NoError(t, err)
	msg := parseSSEDataLine(t, f.Body.String())
	require.Equal(t, sse.StreamTypeToolCallEnd, msg.Type)
}

func TestWriter_WriteToolCallEnd_WithConfig(t *testing.T) {
	f := newMockFlusher()
	w, _ := sse.NewWriter(f)
	config := map[string]interface{}{"result": "42"}
	err := w.WriteToolCallEnd(config)
	require.NoError(t, err)
	msg := parseSSEDataLine(t, f.Body.String())
	require.Equal(t, sse.StreamTypeToolCallEnd, msg.Type)
	require.Equal(t, "42", msg.Config["result"])
}

func TestWriter_WritePlanStart(t *testing.T) {
	f := newMockFlusher()
	w, _ := sse.NewWriter(f)
	err := w.WritePlanStart()
	require.NoError(t, err)
	msg := parseSSEDataLine(t, f.Body.String())
	require.Equal(t, sse.StreamTypePlanStart, msg.Type)
}

func TestWriter_WritePlanStream(t *testing.T) {
	f := newMockFlusher()
	w, _ := sse.NewWriter(f)
	err := w.WritePlanStream("Step 1: Analyze the data")
	require.NoError(t, err)
	msg := parseSSEDataLine(t, f.Body.String())
	require.Equal(t, sse.StreamTypePlanStream, msg.Type)
	require.Equal(t, "Step 1: Analyze the data", msg.Content)
}

func TestWriter_WritePlanComplete(t *testing.T) {
	f := newMockFlusher()
	w, _ := sse.NewWriter(f)
	err := w.WritePlanComplete(nil)
	require.NoError(t, err)
	msg := parseSSEDataLine(t, f.Body.String())
	require.Equal(t, sse.StreamTypePlanComplete, msg.Type)
}

func TestWriter_WritePlanComplete_WithConfig(t *testing.T) {
	f := newMockFlusher()
	w, _ := sse.NewWriter(f)
	config := map[string]interface{}{"steps": 3}
	err := w.WritePlanComplete(config)
	require.NoError(t, err)
	msg := parseSSEDataLine(t, f.Body.String())
	require.Equal(t, sse.StreamTypePlanComplete, msg.Type)
	require.Equal(t, float64(3), msg.Config["steps"])
}

func TestWriter_WriteSubAgentStart(t *testing.T) {
	f := newMockFlusher()
	w, _ := sse.NewWriter(f)
	err := w.WriteSubAgentStart("research_agent", nil)
	require.NoError(t, err)
	msg := parseSSEDataLine(t, f.Body.String())
	require.Equal(t, sse.StreamTypeSubAgentStart, msg.Type)
	require.Equal(t, "research_agent", msg.Config["agentName"])
}

func TestWriter_WriteSubAgentStart_WithConfig(t *testing.T) {
	f := newMockFlusher()
	w, _ := sse.NewWriter(f)
	config := map[string]interface{}{"delegated_task": "search"}
	err := w.WriteSubAgentStart("code_agent", config)
	require.NoError(t, err)
	msg := parseSSEDataLine(t, f.Body.String())
	require.Equal(t, sse.StreamTypeSubAgentStart, msg.Type)
	require.Equal(t, "code_agent", msg.Config["agentName"])
	require.Equal(t, "search", msg.Config["delegated_task"])
}

func TestWriter_WriteSubAgentStream(t *testing.T) {
	f := newMockFlusher()
	w, _ := sse.NewWriter(f)
	err := w.WriteSubAgentStream("sub-agent is working...")
	require.NoError(t, err)
	msg := parseSSEDataLine(t, f.Body.String())
	require.Equal(t, sse.StreamTypeSubAgentStream, msg.Type)
	require.Equal(t, "sub-agent is working...", msg.Content)
}

func TestWriter_WriteSubAgentEnd(t *testing.T) {
	f := newMockFlusher()
	w, _ := sse.NewWriter(f)
	err := w.WriteSubAgentEnd(nil)
	require.NoError(t, err)
	msg := parseSSEDataLine(t, f.Body.String())
	require.Equal(t, sse.StreamTypeSubAgentEnd, msg.Type)
}

func TestWriter_WriteSubAgentEnd_WithConfig(t *testing.T) {
	f := newMockFlusher()
	w, _ := sse.NewWriter(f)
	config := map[string]interface{}{"status": "completed"}
	err := w.WriteSubAgentEnd(config)
	require.NoError(t, err)
	msg := parseSSEDataLine(t, f.Body.String())
	require.Equal(t, sse.StreamTypeSubAgentEnd, msg.Type)
	require.Equal(t, "completed", msg.Config["status"])
}

func TestWriter_WriteSynthesisStart(t *testing.T) {
	f := newMockFlusher()
	w, _ := sse.NewWriter(f)
	err := w.WriteSynthesisStart()
	require.NoError(t, err)
	msg := parseSSEDataLine(t, f.Body.String())
	require.Equal(t, sse.StreamTypeSynthesisStart, msg.Type)
}

func TestWriter_WriteSynthesisStream(t *testing.T) {
	f := newMockFlusher()
	w, _ := sse.NewWriter(f)
	err := w.WriteSynthesisStream("synthesizing final answer...")
	require.NoError(t, err)
	msg := parseSSEDataLine(t, f.Body.String())
	require.Equal(t, sse.StreamTypeSynthesisStream, msg.Type)
	require.Equal(t, "synthesizing final answer...", msg.Content)
}

func TestWriter_WriteStreamTrace(t *testing.T) {
	f := newMockFlusher()
	w, _ := sse.NewWriter(f)
	err := w.WriteStreamTrace(nil)
	require.NoError(t, err)
	msg := parseSSEDataLine(t, f.Body.String())
	require.Equal(t, sse.StreamTypeTrace, msg.Type)
}

func TestWriter_WriteStreamTrace_WithConfig(t *testing.T) {
	f := newMockFlusher()
	w, _ := sse.NewWriter(f)
	config := map[string]interface{}{"node_id": "n1", "status": "completed"}
	err := w.WriteStreamTrace(config)
	require.NoError(t, err)
	msg := parseSSEDataLine(t, f.Body.String())
	require.Equal(t, sse.StreamTypeTrace, msg.Type)
	require.Equal(t, "n1", msg.Config["node_id"])
	require.Equal(t, "completed", msg.Config["status"])
}

func TestWriter_WriteReActStreamMessage(t *testing.T) {
	f := newMockFlusher()
	w, _ := sse.NewWriter(f)
	config := map[string]interface{}{"key": "value"}
	err := w.WriteReActStreamMessage(sse.StreamTypeReasoningStream, "thinking...", config)
	require.NoError(t, err)
	msg := parseSSEDataLine(t, f.Body.String())
	require.Equal(t, sse.StreamTypeReasoningStream, msg.Type)
	require.Equal(t, "thinking...", msg.Content)
	require.Equal(t, "value", msg.Config["key"])
}

func TestWriter_ReActFullStream_Integration(t *testing.T) {
	f := newMockFlusher()
	w, _ := sse.NewWriter(f)

	require.NoError(t, w.WriteStreamStart("msg-1", "conv-1"))
	require.NoError(t, w.WriteReasoningStart())
	require.NoError(t, w.WriteReasoningStream("Let me think..."))
	require.NoError(t, w.WriteReasoningEnd())
	require.NoError(t, w.WriteToolCallStart("search", nil))
	require.NoError(t, w.WriteToolCallStream("searching..."))
	require.NoError(t, w.WriteToolCallEnd(map[string]interface{}{"result": "found"}))
	require.NoError(t, w.WritePlanStart())
	require.NoError(t, w.WritePlanStream("Step 1: Do this"))
	require.NoError(t, w.WritePlanComplete(nil))
	require.NoError(t, w.WriteSubAgentStart("helper", nil))
	require.NoError(t, w.WriteSubAgentStream("delegating..."))
	require.NoError(t, w.WriteSubAgentEnd(nil))
	require.NoError(t, w.WriteSynthesisStart())
	require.NoError(t, w.WriteSynthesisStream("Final answer"))
	require.NoError(t, w.WriteTextStream("Here is the answer."))
	require.NoError(t, w.WriteStreamEnd())
	require.NoError(t, w.WriteDone())

	body := f.Body.String()
	require.Contains(t, body, "REASONING_START")
	require.Contains(t, body, "REASONING_STREAM")
	require.Contains(t, body, "REASONING_END")
	require.Contains(t, body, "TOOL_CALL_START")
	require.Contains(t, body, "TOOL_CALL_STREAM")
	require.Contains(t, body, "TOOL_CALL_END")
	require.Contains(t, body, "PLAN_START")
	require.Contains(t, body, "PLAN_STREAM")
	require.Contains(t, body, "PLAN_COMPLETE")
	require.Contains(t, body, "SUB_AGENT_START")
	require.Contains(t, body, "SUB_AGENT_STREAM")
	require.Contains(t, body, "SUB_AGENT_END")
	require.Contains(t, body, "SYNTHESIS_START")
	require.Contains(t, body, "SYNTHESIS_STREAM")
	require.Equal(t, 18, f.flushed)
}

func TestStreamMessageType_ReActConstants(t *testing.T) {
	require.Equal(t, sse.StreamMessageType("REASONING_START"), sse.StreamTypeReasoningStart)
	require.Equal(t, sse.StreamMessageType("REASONING_STREAM"), sse.StreamTypeReasoningStream)
	require.Equal(t, sse.StreamMessageType("REASONING_END"), sse.StreamTypeReasoningEnd)
	require.Equal(t, sse.StreamMessageType("TOOL_CALL_START"), sse.StreamTypeToolCallStart)
	require.Equal(t, sse.StreamMessageType("TOOL_CALL_STREAM"), sse.StreamTypeToolCallStream)
	require.Equal(t, sse.StreamMessageType("TOOL_CALL_END"), sse.StreamTypeToolCallEnd)
	require.Equal(t, sse.StreamMessageType("PLAN_START"), sse.StreamTypePlanStart)
	require.Equal(t, sse.StreamMessageType("PLAN_STREAM"), sse.StreamTypePlanStream)
	require.Equal(t, sse.StreamMessageType("PLAN_COMPLETE"), sse.StreamTypePlanComplete)
	require.Equal(t, sse.StreamMessageType("SUB_AGENT_START"), sse.StreamTypeSubAgentStart)
	require.Equal(t, sse.StreamMessageType("SUB_AGENT_STREAM"), sse.StreamTypeSubAgentStream)
	require.Equal(t, sse.StreamMessageType("SUB_AGENT_END"), sse.StreamTypeSubAgentEnd)
	require.Equal(t, sse.StreamMessageType("SYNTHESIS_START"), sse.StreamTypeSynthesisStart)
	require.Equal(t, sse.StreamMessageType("SYNTHESIS_STREAM"), sse.StreamTypeSynthesisStream)
	require.Equal(t, sse.StreamMessageType("TRACE"), sse.StreamTypeTrace)
}
