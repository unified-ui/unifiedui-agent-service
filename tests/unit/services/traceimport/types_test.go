package traceimport_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/unifiedui/agent-service/internal/services/traceimport"
)

func TestNewImportRequest(t *testing.T) {
	req := traceimport.NewImportRequest("t1", "c1", "a1", "u1")

	require.Equal(t, "t1", req.TenantID)
	require.Equal(t, "c1", req.ConversationID)
	require.Equal(t, "a1", req.ChatAgentID)
	require.Equal(t, "u1", req.UserID)
	require.NotNil(t, req.Logs)
	require.Empty(t, req.Logs)
	require.NotNil(t, req.BackendConfig)
}

func TestImportRequest_WithBackendConfig(t *testing.T) {
	req := traceimport.NewImportRequest("t1", "c1", "a1", "u1")
	result := req.WithBackendConfig("key1", "val1").WithBackendConfig("key2", 42)

	require.Equal(t, "val1", result.BackendConfig["key1"])
	require.Equal(t, 42, result.BackendConfig["key2"])
}

func TestImportRequest_WithBackendConfig_NilMap(t *testing.T) {
	req := &traceimport.ImportRequest{}
	result := req.WithBackendConfig("key", "value")
	require.Equal(t, "value", result.BackendConfig["key"])
}

func TestImportRequest_WithLogs(t *testing.T) {
	req := traceimport.NewImportRequest("t1", "c1", "a1", "u1")
	result := req.WithLogs("log1", "log2")

	require.Len(t, result.Logs, 2)
	require.Equal(t, "log1", result.Logs[0])
}
