// Package n8n provides N8N trace import functionality.
package n8n

import "strings"

// Config contains N8N-specific configuration for trace import.
type Config struct {
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
func ExtractConfig(backendConfig map[string]interface{}) (*Config, bool) {
	if backendConfig == nil {
		return nil, false
	}

	config := &Config{}

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
// Sub-nodes output via their connection type key (e.g., ai_languageModel, ai_tool).
type NodeOutputData struct {
	Main            [][]NodeOutputItem `json:"main,omitempty"`
	AILanguageModel [][]NodeOutputItem `json:"ai_languageModel,omitempty"`
	AITool          [][]NodeOutputItem `json:"ai_tool,omitempty"`
	AIMemory        [][]NodeOutputItem `json:"ai_memory,omitempty"`
	AIOutputParser  [][]NodeOutputItem `json:"ai_outputParser,omitempty"`
	AIRetriever     [][]NodeOutputItem `json:"ai_retriever,omitempty"`
	AIDocument      [][]NodeOutputItem `json:"ai_document,omitempty"`
	AITextSplitter  [][]NodeOutputItem `json:"ai_textSplitter,omitempty"`
	AIVectorStore   [][]NodeOutputItem `json:"ai_vectorStore,omitempty"`
	AIEmbedding     [][]NodeOutputItem `json:"ai_embedding,omitempty"`
}

// GetOutputItems returns the first non-empty output items from any connection type.
func (d *NodeOutputData) GetOutputItems() [][]NodeOutputItem {
	if len(d.Main) > 0 {
		return d.Main
	}
	if len(d.AILanguageModel) > 0 {
		return d.AILanguageModel
	}
	if len(d.AITool) > 0 {
		return d.AITool
	}
	if len(d.AIMemory) > 0 {
		return d.AIMemory
	}
	if len(d.AIOutputParser) > 0 {
		return d.AIOutputParser
	}
	if len(d.AIRetriever) > 0 {
		return d.AIRetriever
	}
	if len(d.AIDocument) > 0 {
		return d.AIDocument
	}
	if len(d.AITextSplitter) > 0 {
		return d.AITextSplitter
	}
	if len(d.AIVectorStore) > 0 {
		return d.AIVectorStore
	}
	if len(d.AIEmbedding) > 0 {
		return d.AIEmbedding
	}
	return nil
}

// NodeOutputItem represents a single output item.
type NodeOutputItem struct {
	JSON       map[string]interface{} `json:"json,omitempty"`
	Text       string                 `json:"text,omitempty"`
	Binary     map[string]interface{} `json:"binary,omitempty"`
	PairedItem interface{}             `json:"pairedItem,omitempty"`
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

	// Root AI nodes (agents/chains)
	N8NNodeTypeAgent              = "@n8n/n8n-nodes-langchain.agent"
	N8NNodeTypeChainLLM           = "@n8n/n8n-nodes-langchain.chainLlm"
	N8NNodeTypeChainRetrievalQA   = "@n8n/n8n-nodes-langchain.chainRetrievalQa"
	N8NNodeTypeChainSummarization = "@n8n/n8n-nodes-langchain.chainSummarization"
	N8NNodeTypeInfoExtractor      = "@n8n/n8n-nodes-langchain.information-extractor"
	N8NNodeTypeTextClassifier     = "@n8n/n8n-nodes-langchain.text-classifier"
	N8NNodeTypeSentimentAnalysis  = "@n8n/n8n-nodes-langchain.sentimentAnalysis"
	N8NNodeTypeLangChainCode      = "@n8n/n8n-nodes-langchain.code"

	// LLM chat model sub-nodes
	N8NNodeTypeLMChatOpenAI     = "@n8n/n8n-nodes-langchain.lmChatOpenAi"
	N8NNodeTypeLMChatAzure      = "@n8n/n8n-nodes-langchain.lmChatAzureOpenAi"
	N8NNodeTypeLMChatAnthropic  = "@n8n/n8n-nodes-langchain.lmChatAnthropic"
	N8NNodeTypeLMChatGemini     = "@n8n/n8n-nodes-langchain.lmChatGoogleGemini"
	N8NNodeTypeLMChatGroq       = "@n8n/n8n-nodes-langchain.lmChatGroq"
	N8NNodeTypeLMChatDeepSeek   = "@n8n/n8n-nodes-langchain.lmChatDeepSeek"
	N8NNodeTypeLMChatMistral    = "@n8n/n8n-nodes-langchain.lmChatMistralCloud"
	N8NNodeTypeLMChatOllama     = "@n8n/n8n-nodes-langchain.lmChatOllama"
	N8NNodeTypeLMChatBedrock    = "@n8n/n8n-nodes-langchain.lmChatAwsBedrock"
	N8NNodeTypeLMChatOpenRouter = "@n8n/n8n-nodes-langchain.lmChatOpenRouter"
	N8NNodeTypeLMChatXAI        = "@n8n/n8n-nodes-langchain.lmChatXaiGrok"

	// Memory sub-nodes
	N8NNodeTypeMemoryBuffer    = "@n8n/n8n-nodes-langchain.memoryBufferWindow"
	N8NNodeTypeMemoryPostgres  = "@n8n/n8n-nodes-langchain.memoryPostgresChat"
	N8NNodeTypeMemoryRedis     = "@n8n/n8n-nodes-langchain.memoryRedisChat"
	N8NNodeTypeMemoryMongoDB   = "@n8n/n8n-nodes-langchain.memoryMongoDbChat"
	N8NNodeTypeMemoryMotorhead = "@n8n/n8n-nodes-langchain.memoryMotorhead"
	N8NNodeTypeMemoryZep       = "@n8n/n8n-nodes-langchain.memoryZep"
	N8NNodeTypeMemoryXata      = "@n8n/n8n-nodes-langchain.memoryXata"
	N8NNodeTypeMemoryManager   = "@n8n/n8n-nodes-langchain.memoryManager"

	// Tool sub-nodes
	N8NNodeTypeToolWorkflow    = "@n8n/n8n-nodes-langchain.toolWorkflow"
	N8NNodeTypeToolCode        = "@n8n/n8n-nodes-langchain.toolCode"
	N8NNodeTypeToolMCP         = "@n8n/n8n-nodes-langchain.toolMcp"
	N8NNodeTypeToolSerpAPI     = "@n8n/n8n-nodes-langchain.toolSerpApi"
	N8NNodeTypeToolWikipedia   = "@n8n/n8n-nodes-langchain.toolWikipedia"
	N8NNodeTypeToolThink       = "@n8n/n8n-nodes-langchain.toolThink"
	N8NNodeTypeToolVectorStore = "@n8n/n8n-nodes-langchain.toolVectorStore"
	N8NNodeTypeToolCalculator  = "@n8n/n8n-nodes-langchain.toolCalculator"
	N8NNodeTypeToolAIAgent     = "@n8n/n8n-nodes-langchain.toolAiAgent"
	N8NNodeTypeToolSearXNG     = "@n8n/n8n-nodes-langchain.toolSearxng"
	N8NNodeTypeToolWolfram     = "@n8n/n8n-nodes-langchain.toolWolframAlpha"

	// Vector store sub-nodes
	N8NNodeTypeVSPinecone     = "@n8n/n8n-nodes-langchain.vectorStorePinecone"
	N8NNodeTypeVSPGVector     = "@n8n/n8n-nodes-langchain.vectorStorePgVector"
	N8NNodeTypeVSQdrant       = "@n8n/n8n-nodes-langchain.vectorStoreQdrant"
	N8NNodeTypeVSInMemory     = "@n8n/n8n-nodes-langchain.vectorStoreInMemory"
	N8NNodeTypeVSChroma       = "@n8n/n8n-nodes-langchain.vectorStoreChroma"
	N8NNodeTypeVSSupabase     = "@n8n/n8n-nodes-langchain.vectorStoreSupabase"
	N8NNodeTypeVSRedis        = "@n8n/n8n-nodes-langchain.vectorStoreRedis"
	N8NNodeTypeVSWeaviate     = "@n8n/n8n-nodes-langchain.vectorStoreWeaviate"
	N8NNodeTypeVSMongoDBAtlas = "@n8n/n8n-nodes-langchain.vectorStoreMongoDbAtlas"
	N8NNodeTypeVSMilvus       = "@n8n/n8n-nodes-langchain.vectorStoreMilvus"
	N8NNodeTypeVSAzureSearch  = "@n8n/n8n-nodes-langchain.vectorStoreAzureAiSearch"
	N8NNodeTypeVSZep          = "@n8n/n8n-nodes-langchain.vectorStoreZep"

	// Embedding sub-nodes
	N8NNodeTypeEmbeddingsOpenAI      = "@n8n/n8n-nodes-langchain.embeddingsOpenAi"
	N8NNodeTypeEmbeddingsAzure       = "@n8n/n8n-nodes-langchain.embeddingsAzureOpenAi"
	N8NNodeTypeEmbeddingsGemini      = "@n8n/n8n-nodes-langchain.embeddingsGoogleGemini"
	N8NNodeTypeEmbeddingsVertex      = "@n8n/n8n-nodes-langchain.embeddingsGoogleVertex"
	N8NNodeTypeEmbeddingsCohere      = "@n8n/n8n-nodes-langchain.embeddingsCohere"
	N8NNodeTypeEmbeddingsOllama      = "@n8n/n8n-nodes-langchain.embeddingsOllama"
	N8NNodeTypeEmbeddingsMistral     = "@n8n/n8n-nodes-langchain.embeddingsMistralCloud"
	N8NNodeTypeEmbeddingsHuggingFace = "@n8n/n8n-nodes-langchain.embeddingsHuggingFaceInference"
	N8NNodeTypeEmbeddingsBedrock     = "@n8n/n8n-nodes-langchain.embeddingsAwsBedrock"

	// Output parser sub-nodes
	N8NNodeTypeOutputParserStructured = "@n8n/n8n-nodes-langchain.outputParserStructured"
	N8NNodeTypeOutputParserAutofix    = "@n8n/n8n-nodes-langchain.outputParserAutofixing"
	N8NNodeTypeOutputParserItemList   = "@n8n/n8n-nodes-langchain.outputParserItemList"

	// Document loader sub-nodes
	N8NNodeTypeDocDefaultLoader = "@n8n/n8n-nodes-langchain.documentDefaultDataLoader"
	N8NNodeTypeDocGithubLoader  = "@n8n/n8n-nodes-langchain.documentGithubLoader"

	// Text splitter sub-nodes
	N8NNodeTypeTextSplitterRecursive  = "@n8n/n8n-nodes-langchain.textSplitterRecursiveCharacterTextSplitter"
	N8NNodeTypeTextSplitterCharacter  = "@n8n/n8n-nodes-langchain.textSplitterCharacterTextSplitter"
	N8NNodeTypeTextSplitterToken      = "@n8n/n8n-nodes-langchain.textSplitterTokenSplitter"

	// Retriever sub-nodes
	N8NNodeTypeRetrieverVectorStore = "@n8n/n8n-nodes-langchain.retrieverVectorStore"
	N8NNodeTypeRetrieverWorkflow    = "@n8n/n8n-nodes-langchain.retrieverWorkflow"
	N8NNodeTypeRetrieverMultiQuery  = "@n8n/n8n-nodes-langchain.retrieverMultiQuery"
	N8NNodeTypeRetrieverContextual  = "@n8n/n8n-nodes-langchain.retrieverContextualCompression"

	// Core nodes
	N8NNodeTypeHTTPRequest      = "n8n-nodes-base.httpRequest"
	N8NNodeTypeCode             = "n8n-nodes-base.code"
	N8NNodeTypeFunction         = "n8n-nodes-base.function"
	N8NNodeTypeFunctionItem     = "n8n-nodes-base.functionItem"
	N8NNodeTypeSet              = "n8n-nodes-base.set"
	N8NNodeTypeSwitch           = "n8n-nodes-base.switch"
	N8NNodeTypeIf               = "n8n-nodes-base.if"
	N8NNodeTypeFilter           = "n8n-nodes-base.filter"
	N8NNodeTypeMerge            = "n8n-nodes-base.merge"
	N8NNodeTypeNoOp             = "n8n-nodes-base.noOp"
	N8NNodeTypeExecuteWorkflow  = "n8n-nodes-base.executeWorkflow"
	N8NNodeTypeExecuteCommand   = "n8n-nodes-base.executeCommand"
	N8NNodeTypeRespondToWebhook = "n8n-nodes-base.respondToWebhook"
	N8NNodeTypeSplitInBatches   = "n8n-nodes-base.splitInBatches"
	N8NNodeTypeAggregate        = "n8n-nodes-base.aggregate"
	N8NNodeTypeReadWriteFile    = "n8n-nodes-base.readWriteFile"
	N8NNodeTypeHTML             = "n8n-nodes-base.html"
	N8NNodeTypeWait             = "n8n-nodes-base.wait"
	N8NNodeTypeAITransform      = "n8n-nodes-base.aiTransform"

	// Database nodes
	N8NNodeTypePostgres = "n8n-nodes-base.postgres"
	N8NNodeTypeMongoDB  = "n8n-nodes-base.mongoDb"
	N8NNodeTypeMySQL    = "n8n-nodes-base.mySql"
	N8NNodeTypeRedis    = "n8n-nodes-base.redis"

	// Form nodes
	N8NNodeTypeForm = "n8n-nodes-base.form"
)

// extractNodeSuffix extracts the suffix after the last dot from an N8N node type identifier.
func extractNodeSuffix(nodeType string) string {
	if idx := strings.LastIndex(nodeType, "."); idx >= 0 {
		return nodeType[idx+1:]
	}
	return nodeType
}

// GetNodeCategory returns the category of a node based on its type.
func GetNodeCategory(nodeType string) string {
	suffix := extractNodeSuffix(nodeType)

	if suffix == "form" || suffix == "formTrigger" {
		return "form"
	}

	switch {
	case strings.HasPrefix(suffix, "lmChat"):
		return "llm"
	case strings.HasPrefix(suffix, "embeddings"):
		return "embedding"
	case strings.HasPrefix(suffix, "memory"):
		return "memory"
	case strings.HasPrefix(suffix, "vectorStore"):
		return "vectorStore"
	case strings.HasPrefix(suffix, "outputParser"):
		return "outputParser"
	case strings.HasPrefix(suffix, "document"):
		return "document"
	case strings.HasPrefix(suffix, "textSplitter"):
		return "textSplitter"
	case strings.HasPrefix(suffix, "retriever"):
		return "retriever"
	case strings.HasSuffix(suffix, "Trigger") || suffix == "webhook":
		return "trigger"
	case suffix == "agent" || suffix == "information-extractor" ||
		suffix == "text-classifier" || suffix == "textClassifier" ||
		suffix == "sentimentAnalysis":
		return "agent"
	case strings.HasPrefix(suffix, "chain"):
		return "chain"
	case strings.HasPrefix(suffix, "tool"):
		return "tool"
	case suffix == "httpRequest" || suffix == "postgres" || suffix == "mongoDb" ||
		suffix == "mySql" || suffix == "redis" || suffix == "executeCommand" ||
		suffix == "readWriteFile" || suffix == "executeWorkflow":
		return "tool"
	case suffix == "code" || strings.HasPrefix(suffix, "function"):
		return "code"
	case suffix == "switch" || suffix == "if" || suffix == "filter" || suffix == "merge":
		return "conditional"
	case suffix == "splitInBatches":
		return "loop"
	default:
		return "custom"
	}
}
