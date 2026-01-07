// Package traceimport contains unit tests for the traceimport service.
package traceimport

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/unifiedui/agent-service/internal/services/platform"
	"github.com/unifiedui/agent-service/internal/services/traceimport"
	"github.com/unifiedui/agent-service/tests/mocks"
)

// MockTraceImporter implements TraceImporter for testing.
type MockTraceImporter struct {
	agentType  platform.AgentType
	ImportFunc func(ctx context.Context, req *traceimport.ImportRequest) (string, error)
}

func (m *MockTraceImporter) Type() platform.AgentType {
	return m.agentType
}

func (m *MockTraceImporter) Import(ctx context.Context, req *traceimport.ImportRequest) (string, error) {
	if m.ImportFunc != nil {
		return m.ImportFunc(ctx, req)
	}
	return "test-trace-id", nil
}

func TestNewImporterFactory_CreatesEmptyFactory(t *testing.T) {
	mockDocDB := mocks.NewMockDocDBClient()

	factory := traceimport.NewImporterFactory(mockDocDB)

	require.NotNil(t, factory)
	assert.Empty(t, factory.SupportedTypes())
}

func TestImporterFactory_Register_AddsImporter(t *testing.T) {
	mockDocDB := mocks.NewMockDocDBClient()
	factory := traceimport.NewImporterFactory(mockDocDB)

	importer := &MockTraceImporter{agentType: platform.AgentTypeFoundry}
	factory.Register(importer)

	assert.True(t, factory.HasImporter(platform.AgentTypeFoundry))
	assert.Contains(t, factory.SupportedTypes(), platform.AgentTypeFoundry)
}

func TestImporterFactory_Register_MultipleImporters(t *testing.T) {
	mockDocDB := mocks.NewMockDocDBClient()
	factory := traceimport.NewImporterFactory(mockDocDB)

	foundryImporter := &MockTraceImporter{agentType: platform.AgentTypeFoundry}
	n8nImporter := &MockTraceImporter{agentType: platform.AgentTypeN8N}

	factory.Register(foundryImporter)
	factory.Register(n8nImporter)

	assert.True(t, factory.HasImporter(platform.AgentTypeFoundry))
	assert.True(t, factory.HasImporter(platform.AgentTypeN8N))
	assert.Len(t, factory.SupportedTypes(), 2)
}

func TestImporterFactory_GetImporter_Success(t *testing.T) {
	mockDocDB := mocks.NewMockDocDBClient()
	factory := traceimport.NewImporterFactory(mockDocDB)

	importer := &MockTraceImporter{agentType: platform.AgentTypeFoundry}
	factory.Register(importer)

	retrieved, err := factory.GetImporter(platform.AgentTypeFoundry)

	require.NoError(t, err)
	assert.Equal(t, importer, retrieved)
}

func TestImporterFactory_GetImporter_NotFound(t *testing.T) {
	mockDocDB := mocks.NewMockDocDBClient()
	factory := traceimport.NewImporterFactory(mockDocDB)

	_, err := factory.GetImporter(platform.AgentTypeFoundry)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "no trace importer registered")
}

func TestImporterFactory_HasImporter_False(t *testing.T) {
	mockDocDB := mocks.NewMockDocDBClient()
	factory := traceimport.NewImporterFactory(mockDocDB)

	assert.False(t, factory.HasImporter(platform.AgentTypeFoundry))
	assert.False(t, factory.HasImporter(platform.AgentTypeN8N))
}

func TestImporterFactory_SupportedTypes_Empty(t *testing.T) {
	mockDocDB := mocks.NewMockDocDBClient()
	factory := traceimport.NewImporterFactory(mockDocDB)

	types := factory.SupportedTypes()

	assert.Empty(t, types)
}

func TestImporterFactory_SupportedTypes_MultipleTypes(t *testing.T) {
	mockDocDB := mocks.NewMockDocDBClient()
	factory := traceimport.NewImporterFactory(mockDocDB)

	factory.Register(&MockTraceImporter{agentType: platform.AgentTypeFoundry})
	factory.Register(&MockTraceImporter{agentType: platform.AgentTypeN8N})

	types := factory.SupportedTypes()

	assert.Len(t, types, 2)
	assert.Contains(t, types, platform.AgentTypeFoundry)
	assert.Contains(t, types, platform.AgentTypeN8N)
}
