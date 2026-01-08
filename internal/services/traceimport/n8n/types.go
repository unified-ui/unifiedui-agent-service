// Package n8n provides N8N trace import functionality.
package n8n

import "strings"

// N8NConfig contains N8N-specific configuration for trace import.
type N8NConfig struct {
	// ExecutionID is the N8N execution ID.
	ExecutionID string `json:"executionId"`
	// SessionID is the session ID used for fallback search.
	SessionID string `json:"sessionId"`
	// BaseURL is the N8N instance base URL (without trailing slash).
	BaseURL string `json:"baseUrl"`
	// APIKey is the N8N API key for authentication.
	APIKey string `json:"apiKey"`
	// WorkflowID is the optional workflow ID for reference.
	WorkflowID string `json:"workflowId,omitempty"`
}

// BackendConfigKeys for accessing N8N-specific config from BackendConfig map.
const (
	ConfigKeyExecutionID = "execution_id"
	ConfigKeySessionID   = "session_id"
	ConfigKeyBaseURL     = "base_url"
	ConfigKeyAPIKey      = "api_key"
	ConfigKeyWorkflowID  = "workflow_id"
)

// ExtractConfig extracts N8N configuration from a BackendConfig map.
func ExtractConfig(backendConfig map[string]interface{}) (*N8NConfig, bool) {
	if backendConfig == nil {
		return nil, false
	}

	config := &N8NConfig{}

	if v, ok := backendConfig[ConfigKeyExecutionID].(string); ok {
		config.ExecutionID = v
	}
	if v, ok := backendConfig[ConfigKeySessionID].(string); ok {
		config.SessionID = v
	}
	if v, ok := backendConfig[ConfigKeyBaseURL].(string); ok {
		config.BaseURL = v
	}
	if v, ok := backendConfig[ConfigKeyAPIKey].(string); ok {
		config.APIKey = v
	}
	if v, ok := backendConfig[ConfigKeyWorkflowID].(string); ok {
		config.WorkflowID = v
	}

	// Validate required fields
	// We need either ExecutionID or SessionID, plus BaseURL and APIKey
	if config.BaseURL == "" || config.APIKey == "" {
		return nil, false
	}
	if config.ExecutionID == "" && config.SessionID == "" {
		return nil, false
	}

	return config, true
}

// ExecutionStatus represents the status of an N8N execution.
type ExecutionStatus string

const (
	ExecutionStatusSuccess ExecutionStatus = "success"
	ExecutionStatusError   ExecutionStatus = "error"
	ExecutionStatusWaiting ExecutionStatus = "waiting"
	ExecutionStatusRunning ExecutionStatus = "running"
	ExecutionStatusCrashed ExecutionStatus = "crashed"
	ExecutionStatusNew     ExecutionStatus = "new"
)

// NodeExecutionStatus represents the status of a node execution within an N8N workflow.
type NodeExecutionStatus string

const (
	NodeExecutionStatusSuccess NodeExecutionStatus = "success"
	NodeExecutionStatusError   NodeExecutionStatus = "error"
)

// ExecutionResponse represents the full response from the N8N executions API.
type ExecutionResponse struct {
	ID           string                 `json:"id"`
	Finished     bool                   `json:"finished"`
	Mode         string                 `json:"mode"`
	Status       ExecutionStatus        `json:"status"`
	CreatedAt    string                 `json:"createdAt"`
	StartedAt    string                 `json:"startedAt"`
	StoppedAt    string                 `json:"stoppedAt,omitempty"`
	WorkflowID   string                 `json:"workflowId"`
	WaitTill     *string                `json:"waitTill,omitempty"`
	Data         *ExecutionData         `json:"data,omitempty"`
	WorkflowData *WorkflowData          `json:"workflowData,omitempty"`
	Metadata     map[string]interface{} `json:"metadata,omitempty"`
}

// ExecutionData contains the execution result data.
type ExecutionData struct {
	ResultData        *ResultData            `json:"resultData,omitempty"`
	ExecutionData     map[string]interface{} `json:"executionData,omitempty"`
	StartData         map[string]interface{} `json:"startData,omitempty"`
	WaitingExecution  map[string]interface{} `json:"waitingExecution,omitempty"`
	WaitingExecutions map[string]interface{} `json:"waitingExecutions,omitempty"`
}

// ResultData contains the actual execution results per node.
type ResultData struct {
	// RunData is a map of node name to list of executions
	// Each node can be executed multiple times (e.g., in a loop)
	RunData map[string][]NodeExecution `json:"runData,omitempty"`
	// LastNodeExecuted is the name of the last node that was executed
	LastNodeExecuted string `json:"lastNodeExecuted,omitempty"`
	// Error contains error information if the execution failed
	Error *ExecutionError `json:"error,omitempty"`
}

// NodeExecution represents a single execution of a node.
type NodeExecution struct {
	// StartTime is the timestamp when the node started (milliseconds)
	StartTime int64 `json:"startTime"`
	// ExecutionTime is the duration in milliseconds
	ExecutionTime int64 `json:"executionTime"`
	// ExecutionStatus is "success" or "error"
	ExecutionStatus NodeExecutionStatus `json:"executionStatus"`
	// Source contains source node information
	Source []NodeExecutionSource `json:"source,omitempty"`
	// Data contains the output data from the node
	Data NodeOutputData `json:"data,omitempty"`
	// InputOverride contains input override data if applicable
	InputOverride map[string]interface{} `json:"inputOverride,omitempty"`
	// Error contains error details if the node failed
	Error *NodeExecutionError `json:"error,omitempty"`
	// Hints contains execution hints
	Hints []interface{} `json:"hints,omitempty"`
	// Metadata contains additional metadata like tokenUsage
	Metadata *NodeExecutionMetadata `json:"metadata,omitempty"`
}

// NodeExecutionSource represents the source of execution.
type NodeExecutionSource struct {
	PreviousNode       string `json:"previousNode"`
	PreviousNodeRun    int    `json:"previousNodeRun,omitempty"`
	PreviousNodeOutput int    `json:"previousNodeOutput,omitempty"`
}

// NodeOutputData contains the output data structure.
type NodeOutputData struct {
	// Main is the main output, usually an array of item arrays
	Main [][]NodeOutputItem `json:"main,omitempty"`
}

// NodeOutputItem represents a single output item.
type NodeOutputItem struct {
	JSON       map[string]interface{} `json:"json,omitempty"`
	Text       string                 `json:"text,omitempty"`
	Binary     map[string]interface{} `json:"binary,omitempty"`
	PairedItem map[string]interface{} `json:"pairedItem,omitempty"`
}

// NodeExecutionError represents an error in node execution.
type NodeExecutionError struct {
	Name        string                 `json:"name,omitempty"`
	Message     string                 `json:"message,omitempty"`
	Description string                 `json:"description,omitempty"`
	Stack       string                 `json:"stack,omitempty"`
	Context     map[string]interface{} `json:"context,omitempty"`
}

// NodeExecutionMetadata contains metadata about node execution.
type NodeExecutionMetadata struct {
	SubRun       []interface{}     `json:"subRun,omitempty"`
	TokenUsage   *TokenUsage       `json:"tokenUsage,omitempty"`
	SubExecution *SubExecutionInfo `json:"subExecution,omitempty"`
}

// TokenUsage contains token usage information for LLM nodes.
type TokenUsage struct {
	CompletionTokens int `json:"completionTokens,omitempty"`
	PromptTokens     int `json:"promptTokens,omitempty"`
	TotalTokens      int `json:"totalTokens,omitempty"`
}

// SubExecutionInfo contains information about sub-executions.
type SubExecutionInfo struct {
	WorkflowID  string `json:"workflowId,omitempty"`
	ExecutionID string `json:"executionId,omitempty"`
}

// ExecutionError represents a top-level execution error.
type ExecutionError struct {
	Name        string               `json:"name,omitempty"`
	Message     string               `json:"message,omitempty"`
	Stack       string               `json:"stack,omitempty"`
	Description string               `json:"description,omitempty"`
	Cause       *ExecutionErrorCause `json:"cause,omitempty"`
}

// ExecutionErrorCause contains the cause of an execution error.
type ExecutionErrorCause struct {
	Level string    `json:"level,omitempty"`
	Node  *NodeInfo `json:"node,omitempty"`
}

// WorkflowData contains the workflow definition.
type WorkflowData struct {
	ID          string                 `json:"id"`
	Name        string                 `json:"name"`
	Active      bool                   `json:"active"`
	Nodes       []WorkflowNode         `json:"nodes"`
	Connections map[string]interface{} `json:"connections,omitempty"`
	Settings    map[string]interface{} `json:"settings,omitempty"`
	StaticData  interface{}            `json:"staticData,omitempty"`
	Meta        *WorkflowMeta          `json:"meta,omitempty"`
	Tags        []interface{}          `json:"tags,omitempty"`
	PinnedData  map[string]interface{} `json:"pinnedData,omitempty"`
	VersionID   string                 `json:"versionId,omitempty"`
}

// WorkflowNode represents a node in the workflow definition.
type WorkflowNode struct {
	ID          string                 `json:"id"`
	Name        string                 `json:"name"`
	Type        string                 `json:"type"`
	TypeVersion interface{}            `json:"typeVersion"` // Can be int or float
	Position    []int                  `json:"position,omitempty"`
	Parameters  map[string]interface{} `json:"parameters,omitempty"`
	Credentials map[string]interface{} `json:"credentials,omitempty"`
	Disabled    bool                   `json:"disabled,omitempty"`
	OnError     string                 `json:"onError,omitempty"`
	WebhookID   string                 `json:"webhookId,omitempty"`
	Notes       string                 `json:"notes,omitempty"`
}

// WorkflowMeta contains workflow metadata.
type WorkflowMeta struct {
	InstanceID                  string `json:"instanceId,omitempty"`
	TemplateID                  string `json:"templateId,omitempty"`
	TemplateCredsSetupCompleted bool   `json:"templateCredsSetupCompleted,omitempty"`
}

// NodeInfo contains basic node information.
type NodeInfo struct {
	Name string `json:"name,omitempty"`
	Type string `json:"type,omitempty"`
}

// ExecutionsListResponse represents the response from listing executions.
type ExecutionsListResponse struct {
	Data       []ExecutionResponse `json:"data"`
	NextCursor string              `json:"nextCursor,omitempty"`
}

// N8N node type constants for type mapping.
const (
	// Trigger nodes
	N8NNodeTypeChatTrigger     = "@n8n/n8n-nodes-langchain.chatTrigger"
	N8NNodeTypeManualTrigger   = "n8n-nodes-base.manualTrigger"
	N8NNodeTypeWebhook         = "n8n-nodes-base.webhook"
	N8NNodeTypeFormTrigger     = "n8n-nodes-base.formTrigger"
	N8NNodeTypeScheduleTrigger = "n8n-nodes-base.scheduleTrigger"

	// LLM nodes
	N8NNodeTypeAgent           = "@n8n/n8n-nodes-langchain.agent"
	N8NNodeTypeLMChatOpenAI    = "@n8n/n8n-nodes-langchain.lmChatOpenAi"
	N8NNodeTypeLMChatAzure     = "@n8n/n8n-nodes-langchain.lmChatAzureOpenAi"
	N8NNodeTypeLMChatAnthropic = "@n8n/n8n-nodes-langchain.lmChatAnthropic"
	N8NNodeTypeTextClassifier  = "@n8n/n8n-nodes-langchain.textClassifier"

	// Tool nodes
	N8NNodeTypeHTTPRequest  = "n8n-nodes-base.httpRequest"
	N8NNodeTypeCode         = "n8n-nodes-base.code"
	N8NNodeTypeFunction     = "n8n-nodes-base.function"
	N8NNodeTypeFunctionItem = "n8n-nodes-base.functionItem"

	// Database nodes
	N8NNodeTypePostgres = "n8n-nodes-base.postgres"
	N8NNodeTypeMongoDB  = "n8n-nodes-base.mongoDb"
	N8NNodeTypeMySQL    = "n8n-nodes-base.mySql"
	N8NNodeTypeRedis    = "n8n-nodes-base.redis"

	// Control flow nodes
	N8NNodeTypeSwitch = "n8n-nodes-base.switch"
	N8NNodeTypeIf     = "n8n-nodes-base.if"
	N8NNodeTypeMerge  = "n8n-nodes-base.merge"
	N8NNodeTypeNoOp   = "n8n-nodes-base.noOp"

	// Form nodes
	N8NNodeTypeForm = "n8n-nodes-base.form"

	// Memory/Tool nodes for Langchain
	N8NNodeTypeToolWorkflow = "@n8n/n8n-nodes-langchain.toolWorkflow"
	N8NNodeTypeMemoryBuffer = "@n8n/n8n-nodes-langchain.memoryBufferWindow"
)

// GetNodeCategory returns the category of a node based on its type.
func GetNodeCategory(nodeType string) string {
	switch {
	case strings.Contains(nodeType, "form") || strings.Contains(nodeType, "Form"):
		return "form"
	case strings.Contains(nodeType, "Trigger") || strings.Contains(nodeType, "trigger"):
		return "trigger"
	case strings.Contains(nodeType, "agent") || strings.Contains(nodeType, "Agent"):
		return "agent"
	case strings.Contains(nodeType, "lmChat") || strings.Contains(nodeType, "LmChat"):
		return "llm"
	case strings.Contains(nodeType, "httpRequest") || strings.Contains(nodeType, "postgres") ||
		strings.Contains(nodeType, "mongo") || strings.Contains(nodeType, "mysql") ||
		strings.Contains(nodeType, "redis"):
		return "tool"
	case strings.Contains(nodeType, "code") || strings.Contains(nodeType, "function"):
		return "code"
	case strings.Contains(nodeType, "switch") || strings.Contains(nodeType, "if") ||
		strings.Contains(nodeType, "merge"):
		return "conditional"
	case strings.Contains(nodeType, "memory") || strings.Contains(nodeType, "Memory"):
		return "memory"
	case strings.Contains(nodeType, "tool") || strings.Contains(nodeType, "Tool"):
		return "tool"
	default:
		return "custom"
	}
}
