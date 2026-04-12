// Package dto provides Data Transfer Objects for API requests and responses.
package dto

// TestConnectionType represents the type of connection test.
type TestConnectionType string

// TestConnectionType constants define the supported connection test types.
const (
	TestConnectionTypeN8NChatURL          TestConnectionType = "N8N_CHAT_URL"
	TestConnectionTypeN8NWorkflow         TestConnectionType = "N8N_WORKFLOW"
	TestConnectionTypeN8NWebhook          TestConnectionType = "N8N_WEBHOOK"
	TestConnectionTypeFoundryAgent        TestConnectionType = "FOUNDRY_AGENT"
	TestConnectionTypeRestAPIInvoke       TestConnectionType = "REST_API_INVOKE"
	TestConnectionTypeRestAPIConversation TestConnectionType = "REST_API_CONVERSATION"
)

// ValidTestConnectionTypes contains all valid test connection types.
var ValidTestConnectionTypes = map[TestConnectionType]bool{
	TestConnectionTypeN8NChatURL:          true,
	TestConnectionTypeN8NWorkflow:         true,
	TestConnectionTypeN8NWebhook:          true,
	TestConnectionTypeFoundryAgent:        true,
	TestConnectionTypeRestAPIInvoke:       true,
	TestConnectionTypeRestAPIConversation: true,
}

// TestConnectionRequest represents the request for testing a connection.
type TestConnectionRequest struct {
	TestType     TestConnectionType     `json:"test_type" binding:"required"`
	URL          string                 `json:"url" binding:"required"`
	Config       map[string]interface{} `json:"config"`
	CredentialID string                 `json:"credential_id"`
}

// TestConnectionResponse represents the response for testing a connection.
type TestConnectionResponse struct {
	Success        bool   `json:"success"`
	Message        string `json:"message"`
	ResponseTimeMs int64  `json:"response_time_ms"`
}
