// Package n8n provides N8N trace import functionality.
package n8n

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/unifiedui/agent-service/internal/core/docdb"
	"github.com/unifiedui/agent-service/internal/domain/models"
	"github.com/unifiedui/agent-service/internal/services/platform"
	"github.com/unifiedui/agent-service/internal/services/traceimport"
)

// TraceImporter imports traces from N8N workflow executions.
type TraceImporter struct {
	httpClient  *http.Client
	docDB       docdb.Client
	transformer *Transformer
}

// NewTraceImporter creates a new N8N trace importer.
func NewTraceImporter(docDB docdb.Client) *TraceImporter {
	return &TraceImporter{
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		docDB:       docDB,
		transformer: NewTransformer(),
	}
}

// Type returns the agent type this importer handles.
func (n *TraceImporter) Type() platform.AgentType {
	return platform.AgentTypeN8N
}

// Import imports traces from N8N.
// Returns the trace ID on success.
func (n *TraceImporter) Import(ctx context.Context, req *traceimport.ImportRequest) (string, error) {
	// Extract N8N-specific config
	n8nConfig, ok := ExtractConfig(req.BackendConfig)
	if !ok {
		return "", fmt.Errorf("invalid or missing N8N configuration in BackendConfig")
	}

	// If we don't have an execution ID, try to find it by session ID
	if n8nConfig.ExecutionID == "" && n8nConfig.SessionID != "" {
		execID, err := n.findExecutionBySessionID(ctx, n8nConfig)
		if err != nil {
			return "", fmt.Errorf("failed to find execution by session ID: %w", err)
		}
		n8nConfig.ExecutionID = execID
	}

	// Ensure we have an execution ID at this point
	if n8nConfig.ExecutionID == "" {
		return "", fmt.Errorf("no execution ID provided and could not find by session ID")
	}

	// Fetch execution data from N8N
	execution, err := n.fetchExecution(ctx, n8nConfig)
	if err != nil {
		return "", fmt.Errorf("failed to fetch execution: %w", err)
	}

	// Transform N8N execution into TraceNodes
	nodes := n.transformer.TransformExecution(execution, req.UserID)

	// Check if trace already exists for this executionId (referenceId)
	// Skip this check if ExistingTraceID is provided (upsert scenario handled by caller)
	// N8N: Each execution creates a new trace, lookup by executionId as referenceId
	var existingTrace *models.Trace
	if req.ExistingTraceID == "" && n8nConfig.ExecutionID != "" {
		existingTrace, err = n.docDB.Traces().GetByReferenceID(ctx, req.TenantID, n8nConfig.ExecutionID)
		if err != nil {
			return "", fmt.Errorf("failed to check existing trace: %w", err)
		}
	}

	now := time.Now().UTC()

	// Build reference metadata
	referenceMetadata := map[string]interface{}{
		"n8n_execution_id": n8nConfig.ExecutionID,
		"n8n_session_id":   n8nConfig.SessionID,
		"n8n_base_url":     n8nConfig.BaseURL,
		"workflow_id":      execution.WorkflowID,
		"workflow_name":    n.getWorkflowName(execution),
		"execution_status": string(execution.Status),
		"execution_mode":   execution.Mode,
		"imported_at":      now.Format(time.RFC3339),
		"node_count":       len(nodes),
	}

	// Add execution time info
	if execution.StartedAt != "" {
		referenceMetadata["started_at"] = execution.StartedAt
	}
	if execution.StoppedAt != "" {
		referenceMetadata["stopped_at"] = execution.StoppedAt
	}

	// Add error info if execution failed
	if execution.Data != nil && execution.Data.ResultData != nil && execution.Data.ResultData.Error != nil {
		referenceMetadata["error"] = map[string]interface{}{
			"name":    execution.Data.ResultData.Error.Name,
			"message": execution.Data.ResultData.Error.Message,
		}
	}

	if existingTrace != nil {
		// Update existing trace with new data (conversation context)
		existingTrace.ReferenceID = n8nConfig.ExecutionID
		existingTrace.ReferenceName = "N8N Workflow Execution"
		existingTrace.ReferenceMetadata = referenceMetadata
		existingTrace.Logs = req.Logs
		existingTrace.Nodes = nodes
		existingTrace.UpdatedAt = now
		existingTrace.UpdatedBy = req.UserID

		if err := n.docDB.Traces().Update(ctx, existingTrace); err != nil {
			return "", fmt.Errorf("failed to update trace: %w", err)
		}
		return existingTrace.ID, nil
	}

	// If ExistingTraceID is provided, we need to update that trace (autonomous agent upsert)
	if req.ExistingTraceID != "" {
		// Fetch the existing trace and update it
		existingTraceByID, err := n.docDB.Traces().Get(ctx, req.ExistingTraceID)
		if err != nil {
			return "", fmt.Errorf("failed to get existing trace for update: %w", err)
		}
		if existingTraceByID != nil {
			existingTraceByID.ReferenceID = n8nConfig.ExecutionID
			existingTraceByID.ReferenceName = "N8N Workflow Execution"
			existingTraceByID.ReferenceMetadata = referenceMetadata
			existingTraceByID.Logs = req.Logs
			existingTraceByID.Nodes = nodes
			existingTraceByID.UpdatedAt = now
			existingTraceByID.UpdatedBy = req.UserID

			if err := n.docDB.Traces().Update(ctx, existingTraceByID); err != nil {
				return "", fmt.Errorf("failed to update trace: %w", err)
			}
			return existingTraceByID.ID, nil
		}
	}

	// Create new trace
	traceID := "trace_" + uuid.New().String()

	// Determine context type
	contextType := models.TraceContextConversation
	if req.AutonomousAgentID != "" {
		contextType = models.TraceContextAutonomousAgent
	}

	// Create new trace
	trace := &models.Trace{
		ID:                traceID,
		TenantID:          req.TenantID,
		ChatAgentID:       req.ChatAgentID,
		ConversationID:    req.ConversationID,
		AutonomousAgentID: req.AutonomousAgentID,
		ContextType:       contextType,
		ReferenceID:       n8nConfig.ExecutionID,
		ReferenceName:     "N8N Workflow Execution",
		ReferenceMetadata: referenceMetadata,
		Logs:              req.Logs,
		Nodes:             nodes,
		CreatedAt:         now,
		UpdatedAt:         now,
		CreatedBy:         req.UserID,
		UpdatedBy:         req.UserID,
	}

	if err := n.docDB.Traces().Create(ctx, trace); err != nil {
		return "", fmt.Errorf("failed to create trace: %w", err)
	}

	return traceID, nil
}

// fetchExecution fetches execution details from N8N API.
func (n *TraceImporter) fetchExecution(ctx context.Context, config *Config) (*ExecutionResponse, error) {
	baseURL, err := validateHTTPURL(config.BaseURL)
	if err != nil {
		return nil, fmt.Errorf("invalid base URL: %w", err)
	}

	u, err := url.Parse(baseURL + "/api/v1/executions/" + url.PathEscape(validateIdentifier(config.ExecutionID)))
	if err != nil {
		return nil, fmt.Errorf("failed to construct request URL: %w", err)
	}

	q := u.Query()
	q.Set("includeData", "true")
	u.RawQuery = q.Encode()

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), http.NoBody)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Accept", "application/json")
	httpReq.Header.Set("X-N8N-API-KEY", config.APIKey)

	// Execute request
	resp, err := n.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("failed to call N8N API: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	// Read response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	// Check status code
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("N8N API returned status %d: %s", resp.StatusCode, string(body))
	}

	// Parse response
	var execution ExecutionResponse
	if err := json.Unmarshal(body, &execution); err != nil {
		return nil, fmt.Errorf("failed to parse N8N response: %w", err)
	}

	return &execution, nil
}

// findExecutionBySessionID searches for an execution with the given session ID.
// This is used when the execution ID is not available in the stream response.
func (n *TraceImporter) findExecutionBySessionID(ctx context.Context, config *Config) (string, error) {
	baseURL, err := validateHTTPURL(config.BaseURL)
	if err != nil {
		return "", fmt.Errorf("invalid base URL: %w", err)
	}

	u, err := url.Parse(baseURL + "/api/v1/executions")
	if err != nil {
		return "", fmt.Errorf("failed to construct request URL: %w", err)
	}

	q := u.Query()
	q.Set("status", "success")
	q.Set("limit", "100")
	q.Set("includeData", "true")
	if config.WorkflowID != "" {
		q.Set("workflowId", validateIdentifier(config.WorkflowID))
	}
	u.RawQuery = q.Encode()

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), http.NoBody)
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Accept", "application/json")
	httpReq.Header.Set("X-N8N-API-KEY", config.APIKey)

	// Execute request
	resp, err := n.httpClient.Do(httpReq)
	if err != nil {
		return "", fmt.Errorf("failed to call N8N API: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	// Read response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response body: %w", err)
	}

	// Check status code
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("N8N API returned status %d: %s", resp.StatusCode, string(body))
	}

	// Parse response
	var execList ExecutionsListResponse
	if err := json.Unmarshal(body, &execList); err != nil {
		return "", fmt.Errorf("failed to parse N8N response: %w", err)
	}

	// Search for execution with matching session ID
	for i := range execList.Data {
		sessionID := n.transformer.ExtractSessionID(&execList.Data[i])
		if sessionID == config.SessionID {
			return execList.Data[i].ID, nil
		}
	}

	return "", fmt.Errorf("no execution found with session ID: %s", config.SessionID)
}

// getWorkflowName extracts the workflow name from execution response.
func (n *TraceImporter) getWorkflowName(execution *ExecutionResponse) string {
	if execution.WorkflowData != nil && execution.WorkflowData.Name != "" {
		return execution.WorkflowData.Name
	}
	return "Unknown Workflow"
}

// GetTransformer returns the transformer for testing purposes.
func (n *TraceImporter) GetTransformer() *Transformer {
	return n.transformer
}

// ListExecutions lists recent workflow executions from the N8N API.
func (n *TraceImporter) ListExecutions(ctx context.Context, backendConfig map[string]interface{}, limit int, cursor string) (traceimport.WorkflowRunListResult, error) {
	config, ok := ExtractListConfig(backendConfig)
	if !ok {
		return traceimport.WorkflowRunListResult{}, fmt.Errorf("invalid or missing N8N configuration for listing executions")
	}

	baseURL, err := validateHTTPURL(config.BaseURL)
	if err != nil {
		return traceimport.WorkflowRunListResult{}, fmt.Errorf("invalid base URL: %w", err)
	}

	u, err := url.Parse(baseURL + "/api/v1/executions")
	if err != nil {
		return traceimport.WorkflowRunListResult{}, fmt.Errorf("failed to construct request URL: %w", err)
	}

	if limit <= 0 || limit > 100 {
		limit = 100
	}

	q := u.Query()
	q.Set("limit", fmt.Sprintf("%d", limit))
	if config.WorkflowID != "" {
		q.Set("workflowId", validateIdentifier(config.WorkflowID))
	}
	if cursor != "" {
		q.Set("cursor", cursor)
	}
	u.RawQuery = q.Encode()

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), http.NoBody)
	if err != nil {
		return traceimport.WorkflowRunListResult{}, fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Accept", "application/json")
	httpReq.Header.Set("X-N8N-API-KEY", config.APIKey)

	resp, err := n.httpClient.Do(httpReq)
	if err != nil {
		return traceimport.WorkflowRunListResult{}, fmt.Errorf("failed to call N8N API: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return traceimport.WorkflowRunListResult{}, fmt.Errorf("failed to read response body: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return traceimport.WorkflowRunListResult{}, fmt.Errorf("N8N API returned status %d: %s", resp.StatusCode, string(body))
	}

	var execList ExecutionsListResponse
	if err := json.Unmarshal(body, &execList); err != nil {
		return traceimport.WorkflowRunListResult{}, fmt.Errorf("failed to parse N8N response: %w", err)
	}

	runs := make([]traceimport.WorkflowRun, 0, len(execList.Data))
	for _, exec := range execList.Data {
		runs = append(runs, traceimport.WorkflowRun{
			ID:         exec.ID,
			Status:     string(exec.Status),
			StartedAt:  exec.StartedAt,
			StoppedAt:  exec.StoppedAt,
			Mode:       exec.Mode,
			WorkflowID: exec.WorkflowID,
		})
	}

	return traceimport.WorkflowRunListResult{
		Runs:       runs,
		NextCursor: execList.NextCursor,
	}, nil
}

func validateHTTPURL(rawURL string) (string, error) {
	parsed, err := url.Parse(strings.TrimRight(rawURL, "/"))
	if err != nil {
		return "", fmt.Errorf("invalid URL: %w", err)
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return "", fmt.Errorf("unsupported URL scheme: %s", parsed.Scheme)
	}
	if parsed.Host == "" {
		return "", fmt.Errorf("URL missing host")
	}
	return parsed.String(), nil
}
