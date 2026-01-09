// Package mocks provides mock implementations for testing.
package mocks

import (
	"context"

	"github.com/unifiedui/agent-service/internal/services/platform"
	"github.com/unifiedui/agent-service/internal/services/traceimport"
)

// MockImportService is a mock implementation of traceimport.ImportService for testing.
// Since ImportService is a concrete type (not an interface), we provide a helper function
// that creates a real but non-started ImportService suitable for testing.
func NewMockImportService(mockDocDB *MockDocDBClient) *traceimport.ImportService {
	return traceimport.NewImportService(mockDocDB)
}

// MockTraceImporter implements TraceImporter for testing.
// This mock can be used to simulate trace import operations without actual external calls.
type MockTraceImporter struct {
	AgentType  platform.AgentType
	ImportFunc func(ctx context.Context, req *traceimport.ImportRequest) (string, error)
}

// NewMockTraceImporter creates a new mock trace importer that returns a success result.
// If ExistingTraceID is set in the request, it returns that ID (simulating upsert behavior).
func NewMockTraceImporter() *MockTraceImporter {
	return &MockTraceImporter{
		AgentType: platform.AgentTypeN8N,
		ImportFunc: func(ctx context.Context, req *traceimport.ImportRequest) (string, error) {
			// Preserve existing trace ID if set (upsert behavior)
			if req.ExistingTraceID != "" {
				return req.ExistingTraceID, nil
			}
			return "mock-trace-id", nil
		},
	}
}

// Type returns the agent type this importer handles.
func (m *MockTraceImporter) Type() platform.AgentType {
	return m.AgentType
}

// Import simulates importing traces.
func (m *MockTraceImporter) Import(ctx context.Context, req *traceimport.ImportRequest) (string, error) {
	if m.ImportFunc != nil {
		return m.ImportFunc(ctx, req)
	}
	return "mock-trace-id", nil
}
