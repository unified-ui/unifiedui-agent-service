// Package dto provides Data Transfer Objects for API requests and responses.
package dto

// GenerateDescriptionRequest represents the request for generating a description.
type GenerateDescriptionRequest struct {
	EntityType          string                 `json:"entity_type" binding:"required"`
	EntityName          string                 `json:"entity_name" binding:"required"`
	ExistingDescription string                 `json:"existing_description"`
	Context             map[string]interface{} `json:"context"`
}

// GenerateDescriptionResponse represents the response for generating a description.
type GenerateDescriptionResponse struct {
	Description string `json:"description"`
}

// AnalyzeTraceRequest represents the request for analyzing a trace error.
type AnalyzeTraceRequest struct {
	TraceID  string                 `json:"trace_id"`
	NodeID   string                 `json:"node_id"`
	Error    string                 `json:"error" binding:"required"`
	NodeName string                 `json:"node_name" binding:"required"`
	NodeType string                 `json:"node_type" binding:"required"`
	Input    map[string]interface{} `json:"input"`
	Output   map[string]interface{} `json:"output"`
}

// AnalyzeTraceResponse represents the response for analyzing a trace error.
type AnalyzeTraceResponse struct {
	Analysis string `json:"analysis"`
}

// SummarizeTraceRequest represents the request for summarizing a trace.
type SummarizeTraceRequest struct {
	TraceID     string                   `json:"trace_id"`
	DetailLevel string                   `json:"detail_level" binding:"required,oneof=short medium long"`
	Nodes       []map[string]interface{} `json:"nodes" binding:"required"`
}

// SummarizeTraceResponse represents the response for summarizing a trace.
type SummarizeTraceResponse struct {
	Summary string `json:"summary"`
}

// TestModelRequest represents the request for testing an AI model.
type TestModelRequest struct {
	Provider     string                 `json:"provider" binding:"required"`
	Config       map[string]interface{} `json:"config" binding:"required"`
	CredentialID string                 `json:"credential_id"`
}

// TestModelResponse represents the response for testing an AI model.
type TestModelResponse struct {
	Success        bool   `json:"success"`
	Message        string `json:"message"`
	ResponseTimeMs int64  `json:"response_time_ms"`
}

// AICapabilitiesResponse represents the available AI capabilities for a tenant.
type AICapabilitiesResponse struct {
	TitleGeneration       bool `json:"title_generation"`
	DescriptionGeneration bool `json:"description_generation"`
	TraceAnalysis         bool `json:"trace_analysis"`
	Summarization         bool `json:"summarization"`
}
