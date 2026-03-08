package foundry_test

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
	"github.com/unifiedui/agent-service/internal/services/traceimport"
	foundryimport "github.com/unifiedui/agent-service/internal/services/traceimport/foundry"
	"github.com/unifiedui/agent-service/tests/mocks"
)

func makeFoundryItemsResponse() map[string]interface{} {
	return map[string]interface{}{
		"object":   "list",
		"has_more": false,
		"last_id":  "item-2",
		"first_id": "item-1",
		"data": []map[string]interface{}{
			{
				"id":   "item-1",
				"type": "message",
				"role": "user",
				"content": []map[string]interface{}{
					{"type": "input_text", "text": "Hello"},
				},
			},
			{
				"id":   "item-2",
				"type": "message",
				"role": "assistant",
				"content": []map[string]interface{}{
					{"type": "output_text", "text": "Hi there!"},
				},
			},
		},
	}
}

func TestNewTraceImporter(t *testing.T) {
	docDB := mocks.NewMockDocDBClient()
	importer := foundryimport.NewTraceImporter(docDB)
	assert.NotNil(t, importer)
}

func TestTraceImporter_Type(t *testing.T) {
	docDB := mocks.NewMockDocDBClient()
	importer := foundryimport.NewTraceImporter(docDB)
	assert.Equal(t, platform.AgentTypeFoundry, importer.Type())
}

func TestTraceImporter_GetTransformer(t *testing.T) {
	docDB := mocks.NewMockDocDBClient()
	importer := foundryimport.NewTraceImporter(docDB)
	assert.NotNil(t, importer.GetTransformer())
}

func TestTraceImporter_Import_InvalidConfig(t *testing.T) {
	docDB := mocks.NewMockDocDBClient()
	importer := foundryimport.NewTraceImporter(docDB)

	_, err := importer.Import(context.Background(), &traceimport.ImportRequest{
		TenantID:      "t1",
		BackendConfig: map[string]interface{}{},
	})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid or missing Foundry configuration")
}

func TestTraceImporter_Import_FetchError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("error"))
	}))
	defer server.Close()

	docDB := mocks.NewMockDocDBClient()
	importer := foundryimport.NewTraceImporter(docDB)

	_, err := importer.Import(context.Background(), &traceimport.ImportRequest{
		TenantID: "t1",
		BackendConfig: map[string]interface{}{
			"ext_conversation_id": "conv-ext-1",
			"project_endpoint":    server.URL,
			"api_version":         "2025-01-01",
			"api_token":           "token",
		},
	})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to fetch conversation items")
}

func TestTraceImporter_Import_CreateNewTrace(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Contains(t, r.URL.Path, "/openai/conversations/conv-ext-1/items")
		assert.Equal(t, "Bearer token", r.Header.Get("Authorization"))

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(makeFoundryItemsResponse())
	}))
	defer server.Close()

	docDB := mocks.NewMockDocDBClient()
	tracesCol := docDB.GetTracesCollection()

	tracesCol.On("GetByReferenceID", mock.Anything, "t1", "conv-ext-1").
		Return((*models.Trace)(nil), nil)
	tracesCol.On("Create", mock.Anything, mock.AnythingOfType("*models.Trace")).
		Return(nil)

	importer := foundryimport.NewTraceImporter(docDB)

	traceID, err := importer.Import(context.Background(), &traceimport.ImportRequest{
		TenantID:       "t1",
		ConversationID: "conv-1",
		ChatAgentID:    "agent-1",
		UserID:         "user-1",
		BackendConfig: map[string]interface{}{
			"ext_conversation_id": "conv-ext-1",
			"project_endpoint":    server.URL,
			"api_version":         "2025-01-01",
			"api_token":           "token",
		},
	})
	require.NoError(t, err)
	assert.Contains(t, traceID, "trace_")
	tracesCol.AssertCalled(t, "Create", mock.Anything, mock.AnythingOfType("*models.Trace"))
}

func TestTraceImporter_Import_UpdateExistingTrace(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(makeFoundryItemsResponse())
	}))
	defer server.Close()

	existingTrace := &models.Trace{
		ID:       "trace-existing",
		TenantID: "t1",
	}

	docDB := mocks.NewMockDocDBClient()
	tracesCol := docDB.GetTracesCollection()

	tracesCol.On("GetByReferenceID", mock.Anything, "t1", "conv-ext-1").
		Return(existingTrace, nil)
	tracesCol.On("Update", mock.Anything, mock.AnythingOfType("*models.Trace")).Return(nil)

	importer := foundryimport.NewTraceImporter(docDB)

	traceID, err := importer.Import(context.Background(), &traceimport.ImportRequest{
		TenantID: "t1",
		UserID:   "user-1",
		BackendConfig: map[string]interface{}{
			"ext_conversation_id": "conv-ext-1",
			"project_endpoint":    server.URL,
			"api_version":         "2025-01-01",
			"api_token":           "token",
		},
	})
	require.NoError(t, err)
	assert.Equal(t, "trace-existing", traceID)
}

func TestTraceImporter_Import_UpdateByExistingTraceID(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(makeFoundryItemsResponse())
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

	importer := foundryimport.NewTraceImporter(docDB)

	traceID, err := importer.Import(context.Background(), &traceimport.ImportRequest{
		TenantID:        "t1",
		ExistingTraceID: "trace-by-id",
		UserID:          "user-1",
		BackendConfig: map[string]interface{}{
			"ext_conversation_id": "conv-ext-1",
			"project_endpoint":    server.URL,
			"api_version":         "2025-01-01",
			"api_token":           "token",
		},
	})
	require.NoError(t, err)
	assert.Equal(t, "trace-by-id", traceID)
}

func TestTraceImporter_Import_AutonomousAgentContext(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(makeFoundryItemsResponse())
	}))
	defer server.Close()

	docDB := mocks.NewMockDocDBClient()
	tracesCol := docDB.GetTracesCollection()

	tracesCol.On("GetByReferenceID", mock.Anything, "t1", "conv-ext-1").
		Return((*models.Trace)(nil), nil)
	tracesCol.On("Create", mock.Anything, mock.AnythingOfType("*models.Trace")).
		Run(func(args mock.Arguments) {
			trace := args.Get(1).(*models.Trace)
			assert.Equal(t, models.TraceContextAutonomousAgent, trace.ContextType)
		}).Return(nil)

	importer := foundryimport.NewTraceImporter(docDB)

	_, err := importer.Import(context.Background(), &traceimport.ImportRequest{
		TenantID:          "t1",
		AutonomousAgentID: "auto-1",
		UserID:            "user-1",
		BackendConfig: map[string]interface{}{
			"ext_conversation_id": "conv-ext-1",
			"project_endpoint":    server.URL,
			"api_version":         "2025-01-01",
			"api_token":           "token",
		},
	})
	require.NoError(t, err)
}

func TestTraceImporter_Import_EmptyItems(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"object":   "list",
			"has_more": false,
			"data":     []interface{}{},
		})
	}))
	defer server.Close()

	docDB := mocks.NewMockDocDBClient()
	tracesCol := docDB.GetTracesCollection()

	tracesCol.On("GetByReferenceID", mock.Anything, "t1", "conv-ext-1").
		Return((*models.Trace)(nil), nil)
	tracesCol.On("Create", mock.Anything, mock.AnythingOfType("*models.Trace")).Return(nil)

	importer := foundryimport.NewTraceImporter(docDB)

	_, err := importer.Import(context.Background(), &traceimport.ImportRequest{
		TenantID: "t1",
		UserID:   "user-1",
		BackendConfig: map[string]interface{}{
			"ext_conversation_id": "conv-ext-1",
			"project_endpoint":    server.URL,
			"api_version":         "2025-01-01",
			"api_token":           "token",
		},
	})
	require.NoError(t, err)
}

func TestExtractConfig_Success(t *testing.T) {
	config, ok := foundryimport.ExtractConfig(map[string]interface{}{
		"ext_conversation_id": "conv-1",
		"project_endpoint":    "https://test.openai.azure.com",
		"api_version":         "2025-01-01",
		"api_token":           "token",
	})
	assert.True(t, ok)
	assert.Equal(t, "conv-1", config.FoundryConversationID)
	assert.Equal(t, "https://test.openai.azure.com", config.ProjectEndpoint)
}

func TestExtractConfig_NilMap(t *testing.T) {
	_, ok := foundryimport.ExtractConfig(nil)
	assert.False(t, ok)
}

func TestExtractConfig_MissingRequiredFields(t *testing.T) {
	_, ok := foundryimport.ExtractConfig(map[string]interface{}{
		"ext_conversation_id": "conv-1",
	})
	assert.False(t, ok)
}
