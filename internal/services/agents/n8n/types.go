// Package n8n provides N8N-specific agent client implementations.
package n8n

import (
	"fmt"
	"strings"

	"github.com/unifiedui/agent-service/internal/domain/models"
)

// APIVersion represents the N8N API version.
type APIVersion string

const (
	APIVersionV1 APIVersion = "v1"
)

// WorkflowType represents the type of N8N workflow.
type WorkflowType string

const (
	WorkflowTypeChatAgent   WorkflowType = "N8N_CHAT_AGENT_WORKFLOW"
	WorkflowTypeHumanInLoop WorkflowType = "N8N_HUMAN_IN_THE_LOOP"
)

// ChatRequest represents a request to the N8N chat webhook.
type ChatRequest struct {
	ChatInput string `json:"chatInput"`
	SessionID string `json:"sessionId,omitempty"`
}

// N8NStreamType represents the type of N8N stream event.
type N8NStreamType string

const (
	N8NStreamTypeBegin N8NStreamType = "begin"
	N8NStreamTypeItem  N8NStreamType = "item"
	N8NStreamTypeEnd   N8NStreamType = "end"
)

// N8NStreamMetadata represents metadata in N8N stream events.
type N8NStreamMetadata struct {
	NodeID    string `json:"nodeId,omitempty"`
	NodeName  string `json:"nodeName,omitempty"`
	ItemIndex int    `json:"itemIndex,omitempty"`
	RunIndex  int    `json:"runIndex,omitempty"`
	Timestamp int64  `json:"timestamp,omitempty"`
}

// N8NStreamEvent represents a stream event from N8N.
type N8NStreamEvent struct {
	Type     N8NStreamType     `json:"type"`
	Content  string            `json:"content,omitempty"`
	Metadata N8NStreamMetadata `json:"metadata,omitempty"`
}

// ChatStreamChunk represents a chunk from the N8N streaming response (legacy format).
type ChatStreamChunk struct {
	Content      string                 `json:"content,omitempty"`
	ExecutionID  string                 `json:"executionId,omitempty"`
	RunInfo      map[string]interface{} `json:"runInfo,omitempty"`
	InputTokens  int                    `json:"inputTokens,omitempty"`
	OutputTokens int                    `json:"outputTokens,omitempty"`
}

// ExecutionResponse represents the response from the N8N executions API.
type ExecutionResponse struct {
	ID        string                 `json:"id"`
	Finished  bool                   `json:"finished"`
	Mode      string                 `json:"mode"`
	StartedAt string                 `json:"startedAt"`
	StoppedAt string                 `json:"stoppedAt,omitempty"`
	Status    string                 `json:"status"`
	Data      map[string]interface{} `json:"data,omitempty"`
}

// ExecutionsListResponse represents the response from listing executions.
type ExecutionsListResponse struct {
	Data       []*ExecutionResponse `json:"data"`
	NextCursor string               `json:"nextCursor,omitempty"`
}

// BuildChatHistoryMarkdown converts chat history entries to a markdown-formatted string.
// This is specifically for N8N workflows that expect chat history in markdown format.
// Format:
// ### Chat History
//
// **User:** message content
// **Assistant:** response content
// ...
//
// ### Current Message
// message content
func BuildChatHistoryMarkdown(history []models.ChatHistoryEntry, currentMessage string) string {
	var sb strings.Builder

	if len(history) > 0 {
		sb.WriteString("### Chat History\n\n")

		for _, entry := range history {
			switch entry.Role {
			case models.MessageTypeUser:
				sb.WriteString(fmt.Sprintf("**User:** %s\n\n", entry.Content))
			case models.MessageTypeAssistant:
				sb.WriteString(fmt.Sprintf("**Assistant:** %s\n\n", entry.Content))
			}
		}
	}

	sb.WriteString("### Current Message\n\n")
	sb.WriteString(currentMessage)

	return sb.String()
}

// BuildSimpleChatHistoryMarkdown creates a simpler markdown format for chat history.
// Format:
// User: message
// Assistant: response
// User: current message
func BuildSimpleChatHistoryMarkdown(history []models.ChatHistoryEntry, currentMessage string) string {
	var sb strings.Builder

	for _, entry := range history {
		switch entry.Role {
		case models.MessageTypeUser:
			sb.WriteString(fmt.Sprintf("User: %s\n", entry.Content))
		case models.MessageTypeAssistant:
			sb.WriteString(fmt.Sprintf("Assistant: %s\n", entry.Content))
		}
	}

	sb.WriteString(fmt.Sprintf("User: %s", currentMessage))

	return sb.String()
}
