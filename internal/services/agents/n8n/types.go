// Package n8n provides N8N-specific agent client implementations.
package n8n

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

// ChatStreamChunk represents a chunk from the N8N streaming response.
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
