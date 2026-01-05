// Package platform provides the platform service client for configuration retrieval.
package platform

// AgentType represents the type of agent backend.
type AgentType string

const (
	AgentTypeN8N     AgentType = "N8N"
	AgentTypeFoundry AgentType = "FOUNDRY"
	AgentTypeCopilot AgentType = "COPILOT"
	AgentTypeCustom  AgentType = "CUSTOM"
)

// CredentialType represents the type of credentials.
type CredentialType string

const (
	CredentialTypeN8NAPIKey    CredentialType = "N8N_API_KEY"
	CredentialTypeN8NBasicAuth CredentialType = "N8N_BASIC_AUTH"
	CredentialTypeBearerToken  CredentialType = "BEARER_TOKEN"
)

// N8NWorkflowType represents the type of N8N workflow.
type N8NWorkflowType string

const (
	N8NWorkflowTypeChatAgent   N8NWorkflowType = "N8N_CHAT_AGENT_WORKFLOW"
	N8NWorkflowTypeHumanInLoop N8NWorkflowType = "N8N_HUMAN_IN_THE_LOOP"
)

// ServiceConfigResponse represents the config response from platform service (without user data).
// DEPRECATED: Use ApplicationConfigResponse instead.
// This is kept for backwards compatibility.
type ServiceConfigResponse struct {
	DocVersion    string        `json:"docversion"`
	Type          AgentType     `json:"type"`
	TenantID      string        `json:"tenant_id"`
	ApplicationID string        `json:"application_id"`
	Settings      AgentSettings `json:"settings"`
}

// ApplicationConfigResponse represents the config response from platform service.
// This is the response from GET /tenants/{tenant_id}/applications/{application_id}/config
// and includes user information.
type ApplicationConfigResponse struct {
	DocVersion    string        `json:"docversion"`
	Type          AgentType     `json:"type"`
	TenantID      string        `json:"tenant_id"`
	ApplicationID string        `json:"application_id"`
	Settings      AgentSettings `json:"settings"`
	User          *UserInfo     `json:"user,omitempty"`
}

// AgentConfig represents the complete configuration for an agent application.
// This includes user data and is used internally when user context is available.
type AgentConfig struct {
	DocVersion     string        `json:"docversion"`
	Type           AgentType     `json:"type"`
	TenantID       string        `json:"tenant_id"`
	ConversationID string        `json:"conversation_id"`
	ApplicationID  string        `json:"application_id"`
	Settings       AgentSettings `json:"settings"`
	User           *UserInfo     `json:"user,omitempty"`
}

// AgentSettings contains the agent-specific settings.
type AgentSettings struct {
	APIVersion            string          `json:"api_version"`
	WorkflowType          N8NWorkflowType `json:"workflow_type"`
	UseUnifiedChatHistory bool            `json:"use_unified_chat_history"`
	ChatHistoryCount      int             `json:"chat_history_count"`
	ChatURL               string          `json:"chat_url"`
	APICredentials        *Credentials    `json:"api_credentials,omitempty"`
	ChatCredentials       *Credentials    `json:"chat_credentials,omitempty"`
}

// Credentials represents authentication credentials.
type Credentials struct {
	ID             string         `json:"id"`
	CredentialsURI string         `json:"credentials_uri"`
	Name           string         `json:"name"`
	Description    string         `json:"description"`
	Type           CredentialType `json:"type"`
	IsActive       bool           `json:"is_active"`
	Secret         interface{}    `json:"secret"` // Can be string or object
}

// BasicAuthSecret represents basic auth credentials.
type BasicAuthSecret struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

// UserInfo represents user information from the platform.
type UserInfo struct {
	ID            string `json:"id"`
	DisplayName   string `json:"display_name"`
	PrincipalName string `json:"principal_name"`
	Mail          string `json:"mail"`
}

// GetSecretAsString returns the secret as a string (for API keys).
func (c *Credentials) GetSecretAsString() string {
	if s, ok := c.Secret.(string); ok {
		return s
	}
	return ""
}

// GetSecretAsBasicAuth returns the secret as BasicAuthSecret.
func (c *Credentials) GetSecretAsBasicAuth() *BasicAuthSecret {
	if m, ok := c.Secret.(map[string]interface{}); ok {
		return &BasicAuthSecret{
			Username: m["username"].(string),
			Password: m["password"].(string),
		}
	}
	return nil
}
