// Package traceimport provides functionality for importing traces from external systems.
package traceimport

import (
	"context"

	"github.com/unifiedui/agent-service/internal/domain/models"
	"github.com/unifiedui/agent-service/internal/services/platform"
)

// TraceImporter defines the interface for importing traces from external systems.
// Each backend (Foundry, N8N, etc.) implements this interface.
type TraceImporter interface {
	// Type returns the agent type this importer handles.
	Type() platform.AgentType

	// Import imports traces from the external system and returns the trace ID.
	// The BackendConfig in ImportRequest contains backend-specific configuration.
	Import(ctx context.Context, req *ImportRequest) (string, error)
}

// WorkflowRunListResult holds the result of listing workflow runs, including a cursor for pagination.
type WorkflowRunListResult struct {
	Runs       []WorkflowRun
	NextCursor string
}

// WorkflowRunListable defines the interface for importers that can list workflow runs.
type WorkflowRunListable interface {
	// ListExecutions lists recent workflow executions from the external system.
	ListExecutions(ctx context.Context, config map[string]interface{}, limit int, cursor string) (WorkflowRunListResult, error)
}

// WorkflowRun represents a single workflow execution from an external system.
type WorkflowRun struct {
	ID         string `json:"id"`
	Status     string `json:"status"`
	StartedAt  string `json:"startedAt"`
	StoppedAt  string `json:"stoppedAt,omitempty"`
	Mode       string `json:"mode"`
	WorkflowID string `json:"workflowId"`
}

// TraceTransformer defines the interface for transforming external items to TraceNodes.
type TraceTransformer interface {
	// Transform converts external system items into TraceNodes.
	// The items parameter type depends on the backend implementation.
	Transform(items interface{}, createdBy string) []models.TraceNode
}
