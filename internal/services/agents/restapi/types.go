// Package restapi provides REST API agent client implementations.
package restapi

// AuthType represents the authentication type for REST API agents.
type AuthType string

// AuthType constants define the supported authentication types.
const (
	AuthTypeAnonymous              AuthType = "ANONYMOUS"
	AuthTypeBasicAuth              AuthType = "BASIC_AUTH"
	AuthTypeAPIKey                 AuthType = "API_KEY"
	AuthTypeEntraIDUserToken       AuthType = "ENTRA_ID_USER_TOKEN" //nolint:gosec // not a credential
	AuthTypeEntraIDAppRegistration AuthType = "ENTRA_ID_APP_REGISTRATION"
)

// InvokeRequest represents the request body sent to the external REST API agent.
type InvokeRequest struct {
	ConversationID          string                 `json:"conversation_id"`
	UnifiedUIConversationID string                 `json:"unified_ui_conversation_id"`
	MessageHistory          []MessageHistoryEntry  `json:"message_history"`
	Config                  map[string]interface{} `json:"config"`
}

// MessageHistoryEntry represents a single message in the conversation history.
type MessageHistoryEntry struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// CreateConversationRequest represents the request body for creating a conversation.
type CreateConversationRequest struct {
	Config map[string]interface{} `json:"config"`
}

// CreateConversationResponse represents the response from creating a conversation.
type CreateConversationResponse struct {
	ConversationID string `json:"conversation_id"`
}

// StreamEventData represents the parsed data payload of an SSE event.
type StreamEventData struct {
	Type    string                 `json:"type"`
	Content string                 `json:"content"`
	Config  map[string]interface{} `json:"config"`
}
