package traceimport_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/unifiedui/agent-service/internal/services/platform"
	"github.com/unifiedui/agent-service/internal/services/traceimport"
	"github.com/unifiedui/agent-service/tests/mocks"
)

func TestNewImportService(t *testing.T) {
	docDB := &mocks.MockDocDBClient{}
	service := traceimport.NewImportService(docDB)
	require.NotNil(t, service)
	require.NotNil(t, service.GetFactory())
}

func TestImportService_RegisterAndHasImporter(t *testing.T) {
	docDB := &mocks.MockDocDBClient{}
	service := traceimport.NewImportService(docDB)

	require.False(t, service.HasImporter(platform.AgentTypeN8N))

	service.RegisterImporter(&mockImporter{agentType: platform.AgentTypeN8N})
	require.True(t, service.HasImporter(platform.AgentTypeN8N))
}

func TestImportService_SupportedTypes(t *testing.T) {
	docDB := &mocks.MockDocDBClient{}
	service := traceimport.NewImportService(docDB)

	service.RegisterImporter(&mockImporter{agentType: platform.AgentTypeN8N})
	service.RegisterImporter(&mockImporter{agentType: platform.AgentTypeFoundry})

	types := service.SupportedTypes()
	require.Len(t, types, 2)
}

func TestImportService_Import_Success(t *testing.T) {
	docDB := &mocks.MockDocDBClient{}
	service := traceimport.NewImportService(docDB)

	service.RegisterImporter(&mockImporter{agentType: platform.AgentTypeN8N})

	req := traceimport.NewImportRequest("t1", "c1", "a1", "u1")
	traceID, err := service.Import(t.Context(), platform.AgentTypeN8N, req)
	require.NoError(t, err)
	require.Equal(t, "trace-id-123", traceID)
}

func TestImportService_Import_NoImporter(t *testing.T) {
	docDB := &mocks.MockDocDBClient{}
	service := traceimport.NewImportService(docDB)

	req := traceimport.NewImportRequest("t1", "c1", "a1", "u1")
	_, err := service.Import(t.Context(), platform.AgentTypeFoundry, req)
	require.Error(t, err)
}

func TestImportService_EnqueueImport_NoImporter(t *testing.T) {
	docDB := &mocks.MockDocDBClient{}
	service := traceimport.NewImportService(docDB)

	req := traceimport.NewImportRequest("t1", "c1", "a1", "u1")
	err := service.EnqueueImport(platform.AgentTypeN8N, req)
	require.Error(t, err)
}

func TestImportService_EnqueueImport_Success(t *testing.T) {
	docDB := &mocks.MockDocDBClient{}
	service := traceimport.NewImportService(docDB)

	service.RegisterImporter(&mockImporter{agentType: platform.AgentTypeN8N})
	service.Start(1)
	defer service.Stop()

	req := traceimport.NewImportRequest("t1", "c1", "a1", "u1")
	err := service.EnqueueImport(platform.AgentTypeN8N, req)
	require.NoError(t, err)
}
