// Package traceimport contains unit tests for the traceimport service.
package traceimport

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/unifiedui/agent-service/internal/services/platform"
	"github.com/unifiedui/agent-service/internal/services/traceimport"
	"github.com/unifiedui/agent-service/tests/mocks"
)

func TestNewImportService_CreatesService(t *testing.T) {
	mockDocDB := mocks.NewMockDocDBClient()

	service := traceimport.NewImportService(mockDocDB)

	require.NotNil(t, service)
}

func TestImportService_RegisterImporter(t *testing.T) {
	mockDocDB := mocks.NewMockDocDBClient()
	service := traceimport.NewImportService(mockDocDB)

	importer := &MockTraceImporter{agentType: platform.AgentTypeFoundry}
	service.RegisterImporter(importer)

	assert.True(t, service.HasImporter(platform.AgentTypeFoundry))
}

func TestImportService_HasImporter_True(t *testing.T) {
	mockDocDB := mocks.NewMockDocDBClient()
	service := traceimport.NewImportService(mockDocDB)

	importer := &MockTraceImporter{agentType: platform.AgentTypeFoundry}
	service.RegisterImporter(importer)

	assert.True(t, service.HasImporter(platform.AgentTypeFoundry))
}

func TestImportService_HasImporter_False(t *testing.T) {
	mockDocDB := mocks.NewMockDocDBClient()
	service := traceimport.NewImportService(mockDocDB)

	assert.False(t, service.HasImporter(platform.AgentTypeFoundry))
}

func TestImportService_Import_Success(t *testing.T) {
	mockDocDB := mocks.NewMockDocDBClient()
	service := traceimport.NewImportService(mockDocDB)

	expectedTraceID := "trace-123"
	importer := &MockTraceImporter{
		agentType: platform.AgentTypeFoundry,
		ImportFunc: func(ctx context.Context, req *traceimport.ImportRequest) (string, error) {
			return expectedTraceID, nil
		},
	}
	service.RegisterImporter(importer)

	req := traceimport.NewImportRequest("tenant-1", "conv-1", "app-1", "user-1")
	traceID, err := service.Import(context.Background(), platform.AgentTypeFoundry, req)

	require.NoError(t, err)
	assert.Equal(t, expectedTraceID, traceID)
}

func TestImportService_Import_NoImporterRegistered(t *testing.T) {
	mockDocDB := mocks.NewMockDocDBClient()
	service := traceimport.NewImportService(mockDocDB)

	req := traceimport.NewImportRequest("tenant-1", "conv-1", "app-1", "user-1")
	_, err := service.Import(context.Background(), platform.AgentTypeFoundry, req)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "no trace importer registered")
}

func TestImportService_Import_ImporterError(t *testing.T) {
	mockDocDB := mocks.NewMockDocDBClient()
	service := traceimport.NewImportService(mockDocDB)

	importer := &MockTraceImporter{
		agentType: platform.AgentTypeFoundry,
		ImportFunc: func(ctx context.Context, req *traceimport.ImportRequest) (string, error) {
			return "", errors.New("import failed")
		},
	}
	service.RegisterImporter(importer)

	req := traceimport.NewImportRequest("tenant-1", "conv-1", "app-1", "user-1")
	_, err := service.Import(context.Background(), platform.AgentTypeFoundry, req)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "import failed")
}

func TestImportService_Import_PassesCorrectRequest(t *testing.T) {
	mockDocDB := mocks.NewMockDocDBClient()
	service := traceimport.NewImportService(mockDocDB)

	var receivedReq *traceimport.ImportRequest
	importer := &MockTraceImporter{
		agentType: platform.AgentTypeFoundry,
		ImportFunc: func(ctx context.Context, req *traceimport.ImportRequest) (string, error) {
			receivedReq = req
			return "trace-id", nil
		},
	}
	service.RegisterImporter(importer)

	req := traceimport.NewImportRequest("tenant-1", "conv-1", "app-1", "user-1")
	req.WithBackendConfig("ext_conversation_id", "ext-conv-123")
	req.WithBackendConfig("project_endpoint", "https://project.ai.azure.com")

	_, err := service.Import(context.Background(), platform.AgentTypeFoundry, req)

	require.NoError(t, err)
	require.NotNil(t, receivedReq)
	assert.Equal(t, "tenant-1", receivedReq.TenantID)
	assert.Equal(t, "conv-1", receivedReq.ConversationID)
	assert.Equal(t, "app-1", receivedReq.ApplicationID)
	assert.Equal(t, "user-1", receivedReq.UserID)
	assert.Equal(t, "ext-conv-123", receivedReq.BackendConfig["ext_conversation_id"])
	assert.Equal(t, "https://project.ai.azure.com", receivedReq.BackendConfig["project_endpoint"])
}

func TestImportService_SupportedTypes(t *testing.T) {
	mockDocDB := mocks.NewMockDocDBClient()
	service := traceimport.NewImportService(mockDocDB)

	service.RegisterImporter(&MockTraceImporter{agentType: platform.AgentTypeFoundry})
	service.RegisterImporter(&MockTraceImporter{agentType: platform.AgentTypeN8N})

	types := service.SupportedTypes()

	assert.Len(t, types, 2)
	assert.Contains(t, types, platform.AgentTypeFoundry)
	assert.Contains(t, types, platform.AgentTypeN8N)
}

func TestImportService_GetFactory(t *testing.T) {
	mockDocDB := mocks.NewMockDocDBClient()
	service := traceimport.NewImportService(mockDocDB)

	factory := service.GetFactory()

	require.NotNil(t, factory)
}
