package n8n_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/unifiedui/agent-service/internal/domain/models"
	"github.com/unifiedui/agent-service/internal/services/platform"
	n8nimport "github.com/unifiedui/agent-service/internal/services/traceimport/n8n"
	"github.com/unifiedui/agent-service/internal/services/traceimport"
	"github.com/unifiedui/agent-service/tests/mocks"
)

func makeN8NExecutionResponse() map[string]interface{} {
	return map[string]interface{}{
		"id":         "exec-123",
		"finished":   true,
		"mode":       "manual",
		"status":     "success",
		"createdAt":  "2024-01-01T00:00:00Z",
		"startedAt":  "2024-01-01T00:00:00Z",
		"stoppedAt":  "2024-01-01T00:01:00Z",
		"workflowId": "wf-1",
		"data": map[string]interface{}{
			"resultData": map[string]interface{}{
				"runData": map[string]interface{}{},
			},
		},
		"workflowData": map[string]interface{}{
			"name":  "Test Workflow",
			"nodes": []interface{}{},
		},
	}
}

func TestNewTraceImporter(t *testing.T) {
	docDB := mocks.NewMockDocDBClient()
	importer := n8nimport.NewTraceImporter(docDB)
	assert.NotNil(t, importer)
}

func TestTraceImporter_Type(t *testing.T) {
	docDB := mocks.NewMockDocDBClient()
	importer := n8nimport.NewTraceImporter(docDB)
	assert.Equal(t, platform.AgentTypeN8N, importer.Type())
}

func TestTraceImporter_GetTransformer(t *testing.T) {
	docDB := mocks.NewMockDocDBClient()
	importer := n8nimport.NewTraceImporter(docDB)
	assert.NotNil(t, importer.GetTransformer())
}

func TestTraceImporter_Import_InvalidConfig(t *testing.T) {
	docDB := mocks.NewMockDocDBClient()
	importer := n8nimport.NewTraceImporter(docDB)

	_, err := importer.Import(context.Background(), &traceimport.ImportRequest{
		TenantID:      "t1",
		BackendConfig: map[string]interface{}{},
	})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid or missing N8N configuration")
}

func TestTraceImporter_Import_NoExecutionID(t *testing.T) {
	docDB := mocks.NewMockDocDBClient()
	importer := n8nimport.NewTraceImporter(docDB)

	_, err := importer.Import(context.Background(), &traceimport.ImportRequest{
		TenantID: "t1",
		BackendConfig: map[string]interface{}{
			"base_url":   "http://localhost:5678",
			"api_key":    "key",
			"session_id": "sess-1",
		},
	})
	// Will try to find by session ID and fail
	assert.Error(t, err)
}

func TestTraceImporter_Import_FetchExecution_ServerError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("server error"))
	}))
	defer server.Close()

	docDB := mocks.NewMockDocDBClient()
	importer := n8nimport.NewTraceImporter(docDB)

	_, err := importer.Import(context.Background(), &traceimport.ImportRequest{
		TenantID: "t1",
		BackendConfig: map[string]interface{}{
			"execution_id": "exec-123",
			"base_url":     server.URL,
			"api_key":      "key",
		},
	})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to fetch execution")
}

func TestTraceImporter_Import_CreateNewTrace(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(makeN8NExecutionResponse())
	}))
	defer server.Close()

	docDB := mocks.NewMockDocDBClient()
	tracesCol := docDB.GetTracesCollection()

	tracesCol.On("GetByReferenceID", mock.Anything, "t1", "exec-123").
		Return((*models.Trace)(nil), nil)
	tracesCol.On("Create", mock.Anything, mock.AnythingOfType("*models.Trace")).
		Return(nil)

	importer := n8nimport.NewTraceImporter(docDB)

	traceID, err := importer.Import(context.Background(), &traceimport.ImportRequest{
		TenantID:       "t1",
		ConversationID: "conv-1",
		ChatAgentID:    "agent-1",
		UserID:         "user-1",
		BackendConfig: map[string]interface{}{
			"execution_id": "exec-123",
			"base_url":     server.URL,
			"api_key":      "key",
		},
	})
	require.NoError(t, err)
	assert.Contains(t, traceID, "trace_")
	tracesCol.AssertCalled(t, "Create", mock.Anything, mock.AnythingOfType("*models.Trace"))
}

func TestTraceImporter_Import_UpdateExistingTrace(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(makeN8NExecutionResponse())
	}))
	defer server.Close()

	existingTrace := &models.Trace{
		ID:       "trace-existing",
		TenantID: "t1",
	}

	docDB := mocks.NewMockDocDBClient()
	tracesCol := docDB.GetTracesCollection()

	tracesCol.On("GetByReferenceID", mock.Anything, "t1", "exec-123").
		Return(existingTrace, nil)
	tracesCol.On("Update", mock.Anything, mock.AnythingOfType("*models.Trace")).
		Return(nil)

	importer := n8nimport.NewTraceImporter(docDB)

	traceID, err := importer.Import(context.Background(), &traceimport.ImportRequest{
		TenantID:       "t1",
		ConversationID: "conv-1",
		UserID:         "user-1",
		BackendConfig: map[string]interface{}{
			"execution_id": "exec-123",
			"base_url":     server.URL,
			"api_key":      "key",
		},
	})
	require.NoError(t, err)
	assert.Equal(t, "trace-existing", traceID)
}

func TestTraceImporter_Import_UpdateByExistingTraceID(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(makeN8NExecutionResponse())
	}))
	defer server.Close()

	existingTrace := &models.Trace{
		ID:       "trace-by-id",
		TenantID: "t1",
	}

	docDB := mocks.NewMockDocDBClient()
	tracesCol := docDB.GetTracesCollection()

	tracesCol.On("Get", mock.Anything, "trace-by-id").Return(existingTrace, nil)
	tracesCol.On("Update", mock.Anything, mock.AnythingOfType("*models.Trace")).Return(nil)

	importer := n8nimport.NewTraceImporter(docDB)

	traceID, err := importer.Import(context.Background(), &traceimport.ImportRequest{
		TenantID:        "t1",
		ExistingTraceID: "trace-by-id",
		UserID:          "user-1",
		BackendConfig: map[string]interface{}{
			"execution_id": "exec-123",
			"base_url":     server.URL,
			"api_key":      "key",
		},
	})
	require.NoError(t, err)
	assert.Equal(t, "trace-by-id", traceID)
}

func TestTraceImporter_Import_AutonomousAgentContext(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(makeN8NExecutionResponse())
	}))
	defer server.Close()

	docDB := mocks.NewMockDocDBClient()
	tracesCol := docDB.GetTracesCollection()

	tracesCol.On("GetByReferenceID", mock.Anything, "t1", "exec-123").
		Return((*models.Trace)(nil), nil)
	tracesCol.On("Create", mock.Anything, mock.AnythingOfType("*models.Trace")).
		Run(func(args mock.Arguments) {
			trace := args.Get(1).(*models.Trace)
			assert.Equal(t, models.TraceContextAutonomousAgent, trace.ContextType)
			assert.Equal(t, "auto-1", trace.AutonomousAgentID)
		}).Return(nil)

	importer := n8nimport.NewTraceImporter(docDB)

	_, err := importer.Import(context.Background(), &traceimport.ImportRequest{
		TenantID:          "t1",
		AutonomousAgentID: "auto-1",
		UserID:            "user-1",
		BackendConfig: map[string]interface{}{
			"execution_id": "exec-123",
			"base_url":     server.URL,
			"api_key":      "key",
		},
	})
	require.NoError(t, err)
}
