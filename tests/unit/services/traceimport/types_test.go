// Package traceimport contains unit tests for the traceimport types.
package traceimport

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/unifiedui/agent-service/internal/services/traceimport"
)

func TestNewImportRequest_CreatesRequest(t *testing.T) {
	req := traceimport.NewImportRequest("tenant-1", "conv-1", "app-1", "user-1")

	require.NotNil(t, req)
	assert.Equal(t, "tenant-1", req.TenantID)
	assert.Equal(t, "conv-1", req.ConversationID)
	assert.Equal(t, "app-1", req.ApplicationID)
	assert.Equal(t, "user-1", req.UserID)
	assert.NotNil(t, req.Logs)
	assert.Empty(t, req.Logs)
	assert.NotNil(t, req.BackendConfig)
	assert.Empty(t, req.BackendConfig)
}

func TestImportRequest_WithBackendConfig_SingleValue(t *testing.T) {
	req := traceimport.NewImportRequest("tenant-1", "conv-1", "app-1", "user-1")

	result := req.WithBackendConfig("key1", "value1")

	assert.Same(t, req, result, "WithBackendConfig should return same instance for chaining")
	assert.Equal(t, "value1", req.BackendConfig["key1"])
}

func TestImportRequest_WithBackendConfig_MultipleValues(t *testing.T) {
	req := traceimport.NewImportRequest("tenant-1", "conv-1", "app-1", "user-1")

	req.WithBackendConfig("key1", "value1").
		WithBackendConfig("key2", 42).
		WithBackendConfig("key3", true)

	assert.Equal(t, "value1", req.BackendConfig["key1"])
	assert.Equal(t, 42, req.BackendConfig["key2"])
	assert.Equal(t, true, req.BackendConfig["key3"])
}

func TestImportRequest_WithBackendConfig_OverwritesValue(t *testing.T) {
	req := traceimport.NewImportRequest("tenant-1", "conv-1", "app-1", "user-1")

	req.WithBackendConfig("key1", "original")
	req.WithBackendConfig("key1", "updated")

	assert.Equal(t, "updated", req.BackendConfig["key1"])
}

func TestImportRequest_WithLogs_SingleLog(t *testing.T) {
	req := traceimport.NewImportRequest("tenant-1", "conv-1", "app-1", "user-1")

	result := req.WithLogs("log entry 1")

	assert.Same(t, req, result, "WithLogs should return same instance for chaining")
	assert.Len(t, req.Logs, 1)
	assert.Equal(t, "log entry 1", req.Logs[0])
}

func TestImportRequest_WithLogs_MultipleLogs(t *testing.T) {
	req := traceimport.NewImportRequest("tenant-1", "conv-1", "app-1", "user-1")

	req.WithLogs("log 1", "log 2", "log 3")

	assert.Len(t, req.Logs, 3)
	assert.Equal(t, "log 1", req.Logs[0])
	assert.Equal(t, "log 2", req.Logs[1])
	assert.Equal(t, "log 3", req.Logs[2])
}

func TestImportRequest_WithLogs_Appends(t *testing.T) {
	req := traceimport.NewImportRequest("tenant-1", "conv-1", "app-1", "user-1")

	req.WithLogs("log 1")
	req.WithLogs("log 2", "log 3")

	assert.Len(t, req.Logs, 3)
	assert.Equal(t, []string{"log 1", "log 2", "log 3"}, req.Logs)
}

func TestImportRequest_Chaining(t *testing.T) {
	req := traceimport.NewImportRequest("tenant-1", "conv-1", "app-1", "user-1").
		WithBackendConfig("ext_conversation_id", "ext-123").
		WithBackendConfig("project_endpoint", "https://example.com").
		WithLogs("Import started")

	assert.Equal(t, "tenant-1", req.TenantID)
	assert.Equal(t, "ext-123", req.BackendConfig["ext_conversation_id"])
	assert.Equal(t, "https://example.com", req.BackendConfig["project_endpoint"])
	assert.Contains(t, req.Logs, "Import started")
}

func TestJobType_Constants(t *testing.T) {
	assert.Equal(t, traceimport.JobType("MICROSOFT_FOUNDRY"), traceimport.JobTypeMicrosoftFoundry)
	assert.Equal(t, traceimport.JobType("N8N"), traceimport.JobTypeN8N)
}

func TestJobAction_Constants(t *testing.T) {
	assert.Equal(t, traceimport.JobAction("IMPORT_CONVERSATION_TRACES"), traceimport.JobActionImportConversationTraces)
}
