package ai

import (
	"testing"

	"github.com/unifiedui/agent-service/internal/services/ai"

	"github.com/stretchr/testify/require"
)

func TestBuildTitleGenerationMessages(t *testing.T) {
	msgs := ai.BuildTitleGenerationMessages("Hello world", "Hi there, how can I help?")
	require.Len(t, msgs, 3)
	require.Equal(t, "system", msgs[0].Role)
	require.Equal(t, "user", msgs[1].Role)
	require.Equal(t, "Hello world", msgs[1].Content)
	require.Equal(t, "assistant", msgs[2].Role)
}

func TestBuildTitleGenerationMessages_LongAssistantResponse(t *testing.T) {
	long := make([]byte, 1000)
	for i := range long {
		long[i] = 'a'
	}
	msgs := ai.BuildTitleGenerationMessages("hi", string(long))
	require.Len(t, msgs[2].Content, 500)
}

func TestBuildDescriptionGenerationMessages_WithExisting(t *testing.T) {
	msgs := ai.BuildDescriptionGenerationMessages("ChatAgent", "MyAgent", "old desc", nil)
	require.Len(t, msgs, 1)
	require.Equal(t, "user", msgs[0].Role)
	require.Contains(t, msgs[0].Content, "ChatAgent")
	require.Contains(t, msgs[0].Content, "MyAgent")
	require.Contains(t, msgs[0].Content, "old desc")
}

func TestBuildDescriptionGenerationMessages_WithoutExisting(t *testing.T) {
	msgs := ai.BuildDescriptionGenerationMessages("ChatAgent", "MyAgent", "", map[string]interface{}{"key": "val"})
	require.Len(t, msgs, 1)
	require.Contains(t, msgs[0].Content, "ChatAgent")
	require.Contains(t, msgs[0].Content, "MyAgent")
}

func TestBuildDescriptionGenerationMessages_NilContext(t *testing.T) {
	msgs := ai.BuildDescriptionGenerationMessages("ChatAgent", "MyAgent", "", nil)
	require.Contains(t, msgs[0].Content, "{}")
}

func TestBuildTraceAnalysisMessages(t *testing.T) {
	msgs := ai.BuildTraceAnalysisMessages("LLM Node", "llm", "timeout error", "input data", "output data")
	require.Len(t, msgs, 1)
	require.Equal(t, "user", msgs[0].Role)
	require.Contains(t, msgs[0].Content, "LLM Node")
	require.Contains(t, msgs[0].Content, "timeout error")
}

func TestBuildTraceSummarizeMessages(t *testing.T) {
	msgs := ai.BuildTraceSummarizeMessages("short", "node tree here")
	require.Len(t, msgs, 1)
	require.Contains(t, msgs[0].Content, "short")
	require.Contains(t, msgs[0].Content, "node tree here")
}

func TestBuildTestModelMessages(t *testing.T) {
	msgs := ai.BuildTestModelMessages()
	require.Len(t, msgs, 1)
	require.Equal(t, "user", msgs[0].Role)
	require.Equal(t, "Reply with exactly: OK", msgs[0].Content)
}

func TestBuildTraceChatMessages(t *testing.T) {
	history := []ai.ChatMessage{
		{Role: "user", Content: "What happened?"},
		{Role: "assistant", Content: "The trace shows..."},
	}
	msgs := ai.BuildTraceChatMessages(
		`{"id":"t1"}`, `{"id":"n1"}`, history, "Tell me more",
	)
	require.True(t, len(msgs) >= 4)
	require.Equal(t, "system", msgs[0].Role)
	require.Contains(t, msgs[0].Content, `{"id":"t1"}`)
	require.Contains(t, msgs[0].Content, `{"id":"n1"}`)
	require.Equal(t, "Tell me more", msgs[len(msgs)-1].Content)
}

func TestBuildTraceChatMessages_NoSelectedNode(t *testing.T) {
	msgs := ai.BuildTraceChatMessages(`{"id":"t1"}`, "", nil, "What happened?")
	require.Len(t, msgs, 2)
	require.NotContains(t, msgs[0].Content, "Currently selected")
}
