// Package foundry provides Microsoft Foundry trace import functionality.
package foundry

// FoundryConfig contains Foundry-specific configuration for trace import.
type FoundryConfig struct {
	// FoundryConversationID is the external Foundry conversation ID.
	FoundryConversationID string `json:"foundryConversationId"`
	// ProjectEndpoint is the Foundry project endpoint.
	ProjectEndpoint string `json:"projectEndpoint"`
	// APIVersion is the Foundry API version.
	APIVersion string `json:"apiVersion"`
	// FoundryAPIToken is the token for Foundry API authentication.
	FoundryAPIToken string `json:"foundryApiToken"`
}

// ConversationItem represents an item from Foundry's conversation items API.
// This is a flexible structure to handle various item types.
type ConversationItem struct {
	ID     string `json:"id"`
	Type   string `json:"type"`
	Status string `json:"status,omitempty"`
	Role   string `json:"role,omitempty"`
	Kind   string `json:"kind,omitempty"`
	// CreatedBy contains information about who/what created the item.
	CreatedBy map[string]interface{} `json:"created_by,omitempty"`
	// Content can be various types depending on the item type.
	Content []interface{} `json:"content,omitempty"`
	// Workflow action fields
	ActionID         string `json:"action_id,omitempty"`
	ParentActionID   string `json:"parent_action_id,omitempty"`
	PreviousActionID string `json:"previous_action_id,omitempty"`
	// MCP fields
	ServerLabel string `json:"server_label,omitempty"`
	Name        string `json:"name,omitempty"`
	Arguments   string `json:"arguments,omitempty"`
	Output      string `json:"output,omitempty"`
	// Approval fields
	ApprovalRequestID string `json:"approval_request_id,omitempty"`
	Approve           *bool  `json:"approve,omitempty"`
	// Partition key for CosmosDB
	PartitionKey string `json:"partition_key,omitempty"`
}

// ConversationItemsResponse represents the response from Foundry's conversation items API.
type ConversationItemsResponse struct {
	Object  string             `json:"object"`
	HasMore bool               `json:"has_more"`
	LastID  string             `json:"last_id"`
	FirstID string             `json:"first_id"`
	Data    []ConversationItem `json:"data"`
}

// BackendConfigKey is the key used in ImportRequest.BackendConfig for Foundry settings.
const BackendConfigKey = "foundry"

// BackendConfigKeys for accessing Foundry-specific config from BackendConfig map.
const (
	ConfigKeyConversationID = "ext_conversation_id"
	ConfigKeyProjectEndpoint = "project_endpoint"
	ConfigKeyAPIVersion      = "api_version"
	ConfigKeyAPIToken        = "api_token"
)

// ExtractConfig extracts Foundry configuration from a BackendConfig map.
func ExtractConfig(backendConfig map[string]interface{}) (*FoundryConfig, bool) {
	if backendConfig == nil {
		return nil, false
	}

	config := &FoundryConfig{}

	if v, ok := backendConfig[ConfigKeyConversationID].(string); ok {
		config.FoundryConversationID = v
	}
	if v, ok := backendConfig[ConfigKeyProjectEndpoint].(string); ok {
		config.ProjectEndpoint = v
	}
	if v, ok := backendConfig[ConfigKeyAPIVersion].(string); ok {
		config.APIVersion = v
	}
	if v, ok := backendConfig[ConfigKeyAPIToken].(string); ok {
		config.FoundryAPIToken = v
	}

	// Validate required fields
	if config.FoundryConversationID == "" || config.ProjectEndpoint == "" || config.FoundryAPIToken == "" {
		return nil, false
	}

	return config, true
}
