// Package traceimport provides functionality for importing traces from external systems.
package traceimport

// JobType represents the type of import job.
type JobType string

const (
	// JobTypeMicrosoftFoundry represents a Microsoft Foundry trace import job.
	JobTypeMicrosoftFoundry JobType = "MICROSOFT_FOUNDRY"
	// JobTypeN8N represents an N8N trace import job.
	JobTypeN8N JobType = "N8N"
)

// JobAction represents the action to perform.
type JobAction string

const (
	// JobActionImportConversationTraces imports conversation traces.
	JobActionImportConversationTraces JobAction = "IMPORT_CONVERSATION_TRACES"
)

// ImportJob represents a job to import traces from an external system.
type ImportJob struct {
	Type   JobType   `json:"type"`
	Action JobAction `json:"action"`
	Config JobConfig `json:"config"`
}

// JobConfig contains the configuration for an import job.
type JobConfig struct {
	// TenantID is required for all imports.
	TenantID string `json:"tenantId"`
	// ConversationID is the internal conversation ID.
	ConversationID string `json:"conversationId"`
	// ApplicationID is the application ID.
	ApplicationID string `json:"applicationId"`
	// Logs are optional log entries to add to the trace.
	Logs []string `json:"logs"`
	// UserID is the user who initiated the import.
	UserID string `json:"userId"`
	// FoundryConfig contains Foundry-specific configuration.
	FoundryConfig *FoundryJobConfig `json:"foundryConfig,omitempty"`
}

// FoundryJobConfig contains Foundry-specific job configuration.
type FoundryJobConfig struct {
	// FoundryConversationID is the external Foundry conversation ID.
	FoundryConversationID string `json:"foundryConversationId"`
	// ProjectEndpoint is the Foundry project endpoint.
	ProjectEndpoint string `json:"projectEndpoint"`
	// APIVersion is the Foundry API version.
	APIVersion string `json:"apiVersion"`
	// FoundryAPIToken is the token for Foundry API authentication.
	FoundryAPIToken string `json:"foundryApiToken"`
}

// ImportRequest represents a request to import traces.
// This is the common interface for all importers.
type ImportRequest struct {
	// TenantID is required for all imports.
	TenantID string
	// ConversationID is the internal conversation ID.
	ConversationID string
	// ApplicationID is the application ID.
	ApplicationID string
	// Logs are optional log entries to add to the trace.
	Logs []string
	// UserID is the user who initiated the import.
	UserID string
}

// FoundryImportRequest contains all data needed for a Foundry trace import.
type FoundryImportRequest struct {
	ImportRequest
	// FoundryConversationID is the external Foundry conversation ID.
	FoundryConversationID string
	// ProjectEndpoint is the Foundry project endpoint.
	ProjectEndpoint string
	// APIVersion is the Foundry API version.
	APIVersion string
	// FoundryAPIToken is the token for Foundry API authentication.
	FoundryAPIToken string
}

// FoundryConversationItem represents an item from Foundry's conversation items API.
// This is a flexible structure to handle various item types.
type FoundryConversationItem struct {
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

// FoundryConversationItemsResponse represents the response from Foundry's conversation items API.
type FoundryConversationItemsResponse struct {
	Object  string                    `json:"object"`
	HasMore bool                      `json:"has_more"`
	LastID  string                    `json:"last_id"`
	FirstID string                    `json:"first_id"`
	Data    []FoundryConversationItem `json:"data"`
}
