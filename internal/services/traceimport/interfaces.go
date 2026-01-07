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

// TraceTransformer defines the interface for transforming external items to TraceNodes.
type TraceTransformer interface {
	// Transform converts external system items into TraceNodes.
	// The items parameter type depends on the backend implementation.
	Transform(items interface{}, createdBy string) []models.TraceNode
}
