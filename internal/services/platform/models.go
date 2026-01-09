// Package platform provides the platform service client for configuration retrieval.
package platform

// AgentType represents the type of agent backend.
type AgentType string

const (
	AgentTypeN8N     AgentType = "N8N"
	AgentTypeFoundry AgentType = "MICROSOFT_FOUNDRY"
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
	// Common settings
	APIVersion            string `json:"api_version"`
	UseUnifiedChatHistory bool   `json:"use_unified_chat_history"`
	ChatHistoryCount      int    `json:"chat_history_count"`

	// N8N specific settings
	WorkflowType    N8NWorkflowType `json:"workflow_type,omitempty"`
	ChatURL         string          `json:"chat_url,omitempty"`
	APICredentials  *Credentials    `json:"api_credentials,omitempty"`
	ChatCredentials *Credentials    `json:"chat_credentials,omitempty"`

	// Microsoft Foundry specific settings
	AgentType       string `json:"agent_type,omitempty"`       // "AGENT" or "MULTI_AGENT"
	ProjectEndpoint string `json:"project_endpoint,omitempty"` // Full endpoint URL
	AgentName       string `json:"agent_name,omitempty"`       // Agent name to invoke
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

// UserInfo represents user information from the platform's identity/me endpoint.
// This matches the IdentityUserResponse from the Python platform service.
type UserInfo struct {
	ID               string                   `json:"id"`
	IdentityProvider string                   `json:"identity_provider"`
	IdentityTenantID string                   `json:"identity_tenant_id"`
	DisplayName      string                   `json:"display_name"`
	PrincipalName    string                   `json:"principal_name"`
	Firstname        string                   `json:"firstname"`
	Lastname         string                   `json:"lastname"`
	Mail             string                   `json:"mail"`
	Tenants          []map[string]interface{} `json:"tenants"`
	Groups           []map[string]interface{} `json:"groups"`
}

// ConversationResponse represents a conversation from the platform service.
type ConversationResponse struct {
	ID                string `json:"id"`
	Name              string `json:"name"`
	TenantID          string `json:"tenant_id"`
	ApplicationID     string `json:"application_id"`
	ExtConversationID string `json:"ext_conversation_id,omitempty"`
}

// AutonomousAgentConfigResponse represents the config response from platform service
// for autonomous agents. This is the response from GET /tenants/{tenant_id}/autonomous-agents/{id}/config
// and uses API key authentication (not Bearer token).
type AutonomousAgentConfigResponse struct {
	DocVersion        string                        `json:"docversion"`
	Type              AgentType                     `json:"type"`
	TenantID          string                        `json:"tenant_id"`
	AutonomousAgentID string                        `json:"autonomous_agent_id"`
	Settings          AutonomousAgentConfigSettings `json:"settings"`
}

// AutonomousAgentConfigSettings contains the autonomous agent-specific settings.
type AutonomousAgentConfigSettings struct {
	// API version for the autonomous agent config format
	APIVersion string `json:"api_version"`

	// N8N specific settings
	N8NHost             string       `json:"n8n_host,omitempty"`
	N8NWorkflowEndpoint string       `json:"n8n_workflow_endpoint,omitempty"`
	WorkflowID          string       `json:"workflow_id,omitempty"`
	APICredentials      *Credentials `json:"api_credentials,omitempty"`
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
