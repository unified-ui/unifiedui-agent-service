// Package platform provides the platform service client for configuration retrieval.
package platform

// AgentType represents the type of agent backend.
type AgentType string

// AgentType constants define the supported agent backend types.
const (
	AgentTypeN8N        AgentType = "N8N"
	AgentTypeFoundry    AgentType = "MICROSOFT_FOUNDRY"
	AgentTypeReactAgent AgentType = "REACT_AGENT"
	AgentTypeCopilot    AgentType = "COPILOT"
	AgentTypeCustom     AgentType = "CUSTOM"
	AgentTypeRestAPI    AgentType = "REST_API"
	AgentTypeLLM        AgentType = "LLM"
)

// CredentialType represents the type of credentials.
type CredentialType string

// CredentialType constants define the supported credential types.
const (
	CredentialTypeN8NAPIKey    CredentialType = "N8N_API_KEY"    //nolint:gosec // credential type name, not a credential
	CredentialTypeN8NBasicAuth CredentialType = "N8N_BASIC_AUTH" //nolint:gosec // credential type name, not a credential
	CredentialTypeBearerToken  CredentialType = "BEARER_TOKEN"
)

// N8NWorkflowType represents the type of N8N workflow.
type N8NWorkflowType string

// N8NWorkflowType constants define the types of N8N workflows.
const (
	N8NWorkflowTypeChatAgent   N8NWorkflowType = "N8N_CHAT_AGENT_WORKFLOW"
	N8NWorkflowTypeHumanInLoop N8NWorkflowType = "N8N_HUMAN_IN_THE_LOOP"
)

// ServiceConfigResponse represents the config response from platform service (without user data).
// Deprecated: Use ChatAgentConfigResponse instead.
// This is kept for backwards compatibility.
type ServiceConfigResponse struct {
	DocVersion  string        `json:"docversion"`
	Type        AgentType     `json:"type"`
	TenantID    string        `json:"tenant_id"`
	ChatAgentID string        `json:"chat_agent_id"`
	Settings    AgentSettings `json:"settings"`
}

// ChatAgentConfigResponse represents the config response from platform service.
// This is the response from GET /tenants/{tenant_id}/chat-agents/{chat_agent_id}/config
// and includes user information.
type ChatAgentConfigResponse struct {
	DocVersion  string        `json:"docversion"`
	Type        AgentType     `json:"type"`
	TenantID    string        `json:"tenant_id"`
	ChatAgentID string        `json:"chat_agent_id"`
	Settings    AgentSettings `json:"settings"`
	User        *UserInfo     `json:"user,omitempty"`
}

// AgentConfig represents the complete configuration for a chat agent.
// This includes user data and is used internally when user context is available.
type AgentConfig struct {
	DocVersion     string        `json:"docversion"`
	Type           AgentType     `json:"type"`
	TenantID       string        `json:"tenant_id"`
	ConversationID string        `json:"conversation_id"`
	ChatAgentID    string        `json:"chat_agent_id"`
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

	// ReACT Agent specific settings
	ReActAgentID string            `json:"react_agent_id,omitempty"`
	Tools        []ReActTool       `json:"tools,omitempty"`
	SystemPrompt string            `json:"system_prompt,omitempty"`
	AIModelIDs   []string          `json:"ai_model_ids,omitempty"`
	AIModels     []ResolvedAIModel `json:"ai_models,omitempty"`

	// REST API specific settings
	AuthType                   string       `json:"auth_type,omitempty"`
	InvokeEndpoint             string       `json:"invoke_endpoint,omitempty"`
	CreateConversationEndpoint string       `json:"create_conversation_endpoint,omitempty"`
	Credential                 *Credentials `json:"credential,omitempty"`
	AccessToken                string       `json:"access_token,omitempty"`
	APIKeyHeaderName           string       `json:"api_key_header_name,omitempty"`

	// LLM specific settings
	AIModelID string           `json:"ai_model_id,omitempty"`
	AIModel   *ResolvedAIModel `json:"ai_model,omitempty"`
}

// ResolvedAIModel represents a fully resolved AI model with decrypted credentials
// as returned by the platform service config endpoint.
type ResolvedAIModel struct {
	ID               string                 `json:"id"`
	Provider         string                 `json:"provider"`
	Config           map[string]interface{} `json:"config"`
	CredentialSecret map[string]interface{} `json:"credential_secret"`
	Priority         int                    `json:"priority"`
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

// ReActTool represents a tool definition returned in the ReACT agent config.
type ReActTool struct {
	ID          string       `json:"id"`
	Name        string       `json:"name"`
	Description string       `json:"description"`
	Type        string       `json:"type"` // "MCP_SERVER" or "OPENAPI_DEFINITION"
	Config      interface{}  `json:"config"`
	IsActive    bool         `json:"is_active"`
	Credentials *Credentials `json:"credentials,omitempty"`
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
	ChatAgentID       string `json:"chat_agent_id"`
	ExtConversationID string `json:"ext_conversation_id,omitempty"`
}

// WorkflowConfigResponse represents the config response from platform service
// for workflows. This is the response from GET /tenants/{tenant_id}/workflows/{id}/config
// and uses API key authentication (not Bearer token).
type WorkflowConfigResponse struct {
	DocVersion string                 `json:"docversion"`
	Type       AgentType              `json:"type"`
	TenantID   string                 `json:"tenant_id"`
	WorkflowID string                 `json:"workflow_id"`
	Settings   WorkflowConfigSettings `json:"settings"`
}

// WorkflowConfigSettings contains the workflow-specific settings.
type WorkflowConfigSettings struct {
	// API version for the workflow config format
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

// EntraIDAppRegistrationSecret represents Entra ID app registration credentials.
type EntraIDAppRegistrationSecret struct {
	TenantID     string `json:"tenant_id"`
	ClientID     string `json:"client_id"`
	ClientSecret string `json:"client_secret"`
}

// GetSecretAsEntraIDAppReg returns the secret as EntraIDAppRegistrationSecret.
func (c *Credentials) GetSecretAsEntraIDAppReg() *EntraIDAppRegistrationSecret {
	m, ok := c.Secret.(map[string]interface{})
	if !ok {
		return nil
	}
	tID, _ := m["tenant_id"].(string)
	cID, _ := m["client_id"].(string)
	cSecret, _ := m["client_secret"].(string)
	if tID == "" || cID == "" || cSecret == "" {
		return nil
	}
	return &EntraIDAppRegistrationSecret{
		TenantID:     tID,
		ClientID:     cID,
		ClientSecret: cSecret,
	}
}
