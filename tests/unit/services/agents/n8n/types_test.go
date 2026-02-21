package n8n_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/unifiedui/agent-service/internal/domain/models"
	"github.com/unifiedui/agent-service/internal/services/agents/n8n"
)

func TestBuildChatHistoryMarkdown_NoHistory(t *testing.T) {
	ts := time.Date(2026, 1, 4, 13, 20, 0, 0, time.UTC)
	result := n8n.BuildChatHistoryMarkdown(nil, "Hello", ts)

	require.NotContains(t, result, "## Chat History")
	require.Contains(t, result, "## Current Message")
	require.Contains(t, result, "Hello")
}

func TestBuildChatHistoryMarkdown_WithHistory(t *testing.T) {
	ts := time.Date(2026, 1, 4, 13, 18, 0, 0, time.UTC)
	history := []models.ChatHistoryEntry{
		{Role: models.MessageTypeUser, Content: "Hi", Timestamp: ts},
		{Role: models.MessageTypeAssistant, Content: "Hello!", Timestamp: ts.Add(time.Second * 3)},
	}
	current := ts.Add(time.Minute * 2)
	result := n8n.BuildChatHistoryMarkdown(history, "How are you?", current)

	require.Contains(t, result, "## Chat History")
	require.Contains(t, result, "[2026-01-04 13:18:00 | user]:")
	require.Contains(t, result, "Hi")
	require.Contains(t, result, "Hello!")
	require.Contains(t, result, "## Current Message")
	require.Contains(t, result, "How are you?")
}

func TestBuildSimpleChatHistoryMarkdown_NoHistory(t *testing.T) {
	ts := time.Date(2026, 1, 4, 13, 20, 0, 0, time.UTC)
	result := n8n.BuildSimpleChatHistoryMarkdown(nil, "Hello", ts)

	require.NotContains(t, result, "<history>")
	require.Contains(t, result, "<current>")
	require.Contains(t, result, "Hello")
	require.Contains(t, result, "</current>")
}

func TestBuildSimpleChatHistoryMarkdown_WithHistory(t *testing.T) {
	ts := time.Date(2026, 1, 4, 13, 18, 0, 0, time.UTC)
	history := []models.ChatHistoryEntry{
		{Role: models.MessageTypeUser, Content: "Hi", Timestamp: ts},
	}
	result := n8n.BuildSimpleChatHistoryMarkdown(history, "Bye", ts.Add(time.Minute))

	require.Contains(t, result, "<history>")
	require.Contains(t, result, "</history>")
	require.Contains(t, result, "[2026-01-04 13:18:00|user]: Hi")
	require.Contains(t, result, "<current>")
	require.Contains(t, result, "Bye")
}
