package traceimport_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/unifiedui/agent-service/internal/services/platform"
	"github.com/unifiedui/agent-service/internal/services/traceimport"
	"github.com/unifiedui/agent-service/tests/mocks"
)

type mockImporter struct {
	agentType platform.AgentType
}

func (m *mockImporter) Type() platform.AgentType {
	return m.agentType
}

func (m *mockImporter) Import(ctx context.Context, req *traceimport.ImportRequest) (string, error) {
	return "trace-id-123", nil
}

func TestImporterFactory_Register_And_GetImporter(t *testing.T) {
	docDB := &mocks.MockDocDBClient{}
	factory := traceimport.NewImporterFactory(docDB)

	imp := &mockImporter{agentType: platform.AgentTypeN8N}
	factory.Register(imp)

	got, err := factory.GetImporter(platform.AgentTypeN8N)
	require.NoError(t, err)
	require.NotNil(t, got)
}

func TestImporterFactory_GetImporter_NotRegistered(t *testing.T) {
	docDB := &mocks.MockDocDBClient{}
	factory := traceimport.NewImporterFactory(docDB)

	_, err := factory.GetImporter(platform.AgentTypeFoundry)
	require.Error(t, err)
	require.Contains(t, err.Error(), "no trace importer")
}

func TestImporterFactory_HasImporter(t *testing.T) {
	docDB := &mocks.MockDocDBClient{}
	factory := traceimport.NewImporterFactory(docDB)

	require.False(t, factory.HasImporter(platform.AgentTypeN8N))

	factory.Register(&mockImporter{agentType: platform.AgentTypeN8N})
	require.True(t, factory.HasImporter(platform.AgentTypeN8N))
}

func TestImporterFactory_SupportedTypes(t *testing.T) {
	docDB := &mocks.MockDocDBClient{}
	factory := traceimport.NewImporterFactory(docDB)

	factory.Register(&mockImporter{agentType: platform.AgentTypeN8N})
	factory.Register(&mockImporter{agentType: platform.AgentTypeFoundry})

	types := factory.SupportedTypes()
	require.Len(t, types, 2)
	require.Contains(t, types, platform.AgentTypeN8N)
	require.Contains(t, types, platform.AgentTypeFoundry)
}
