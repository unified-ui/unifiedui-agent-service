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
	// BackendConfig contains backend-specific configuration as a generic map.
	BackendConfig map[string]interface{} `json:"backendConfig,omitempty"`
}

// ImportRequest represents a request to import traces.
// This is the common structure for all importers.
type ImportRequest struct {
	// TenantID is required for all imports.
	TenantID string
	// ConversationID is the internal conversation ID.
	ConversationID string
	// ApplicationID is the application ID.
	ApplicationID string
	// AutonomousAgentID is the autonomous agent ID (for autonomous agent context).
	AutonomousAgentID string
	// ExistingTraceID is set when updating an existing trace (preserves the ID).
	ExistingTraceID string
	// Logs are optional log entries to add to the trace.
	Logs []string
	// UserID is the user who initiated the import.
	UserID string
	// BackendConfig contains backend-specific configuration.
	// Each importer extracts its required fields from this map.
	// For Foundry: ext_conversation_id, project_endpoint, api_version, api_token
	// For N8N: execution_id, workflow_id, instance_url, api_key
	BackendConfig map[string]interface{}
}

// NewImportRequest creates a new ImportRequest with the given parameters.
func NewImportRequest(tenantID, conversationID, applicationID, userID string) *ImportRequest {
	return &ImportRequest{
		TenantID:       tenantID,
		ConversationID: conversationID,
		ApplicationID:  applicationID,
		UserID:         userID,
		Logs:           []string{},
		BackendConfig:  make(map[string]interface{}),
	}
}

// WithBackendConfig adds backend-specific configuration.
func (r *ImportRequest) WithBackendConfig(key string, value interface{}) *ImportRequest {
	if r.BackendConfig == nil {
		r.BackendConfig = make(map[string]interface{})
	}
	r.BackendConfig[key] = value
	return r
}

// WithLogs adds log entries.
func (r *ImportRequest) WithLogs(logs ...string) *ImportRequest {
	r.Logs = append(r.Logs, logs...)
	return r
}
