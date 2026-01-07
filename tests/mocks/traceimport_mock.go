// Package mocks provides mock implementations for testing.
package mocks

import (
	"github.com/unifiedui/agent-service/internal/services/traceimport"
)

// MockImportService is a mock implementation of traceimport.ImportService for testing.
// Since ImportService is a concrete type (not an interface), we provide a helper function
// that creates a real but non-started ImportService suitable for testing.
func NewMockImportService(mockDocDB *MockDocDBClient) *traceimport.ImportService {
	return traceimport.NewImportService(mockDocDB)
}
