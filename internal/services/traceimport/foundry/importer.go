// Package foundry provides Microsoft Foundry trace import functionality.
package foundry

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/google/uuid"

	"github.com/unifiedui/agent-service/internal/core/docdb"
	"github.com/unifiedui/agent-service/internal/domain/models"
	"github.com/unifiedui/agent-service/internal/services/platform"
	"github.com/unifiedui/agent-service/internal/services/traceimport"
)

// TraceImporter imports traces from Microsoft Foundry.
type TraceImporter struct {
	httpClient  *http.Client
	docDB       docdb.Client
	transformer *Transformer
}

// NewTraceImporter creates a new Foundry trace importer.
func NewTraceImporter(docDB docdb.Client) *TraceImporter {
	return &TraceImporter{
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		docDB:       docDB,
		transformer: NewTransformer(),
	}
}

// Type returns the agent type this importer handles.
func (f *TraceImporter) Type() platform.AgentType {
	return platform.AgentTypeFoundry
}

// Import imports traces from Microsoft Foundry.
// Returns the trace ID on success.
func (f *TraceImporter) Import(ctx context.Context, req *traceimport.ImportRequest) (string, error) {
	// Extract Foundry-specific config
	foundryConfig, ok := ExtractConfig(req.BackendConfig)
	if !ok {
		return "", fmt.Errorf("invalid or missing Foundry configuration in BackendConfig")
	}

	// Fetch conversation items from Foundry
	items, err := f.fetchConversationItems(ctx, foundryConfig)
	if err != nil {
		return "", fmt.Errorf("failed to fetch conversation items: %w", err)
	}

	// Transform Foundry items into TraceNodes
	var nodes []models.TraceNode
	if items != nil && len(items.Data) > 0 {
		nodes = f.transformer.Transform(items.Data, req.UserID)
	}

	// Check if trace already exists for this conversation
	// Skip this check if ExistingTraceID is provided (upsert scenario handled by caller)
	var existingTrace *models.Trace
	if req.ExistingTraceID == "" && req.ConversationID != "" {
		existingTrace, err = f.docDB.Traces().GetByConversation(ctx, req.TenantID, req.ConversationID)
		if err != nil {
			return "", fmt.Errorf("failed to check existing trace: %w", err)
		}
	}

	now := time.Now().UTC()

	if existingTrace != nil {
		// Update existing trace with new data (conversation context)
		existingTrace.ReferenceID = foundryConfig.FoundryConversationID
		existingTrace.ReferenceName = "Microsoft Foundry Conversation"
		existingTrace.ReferenceMetadata = map[string]interface{}{
			"foundry_conversation_id": foundryConfig.FoundryConversationID,
			"project_endpoint":        foundryConfig.ProjectEndpoint,
			"api_version":             foundryConfig.APIVersion,
			"imported_at":             now.Format(time.RFC3339),
			"item_count":              len(items.Data),
		}
		existingTrace.Logs = req.Logs
		existingTrace.Nodes = nodes
		existingTrace.UpdatedAt = now
		existingTrace.UpdatedBy = req.UserID

		if err := f.docDB.Traces().Update(ctx, existingTrace); err != nil {
			return "", fmt.Errorf("failed to update trace: %w", err)
		}
		return existingTrace.ID, nil
	}

	// If ExistingTraceID is provided, we need to update that trace (autonomous agent upsert)
	if req.ExistingTraceID != "" {
		existingTraceByID, err := f.docDB.Traces().Get(ctx, req.ExistingTraceID)
		if err != nil {
			return "", fmt.Errorf("failed to get existing trace for update: %w", err)
		}
		if existingTraceByID != nil {
			existingTraceByID.ReferenceID = foundryConfig.FoundryConversationID
			existingTraceByID.ReferenceName = "Microsoft Foundry Conversation"
			existingTraceByID.ReferenceMetadata = map[string]interface{}{
				"foundry_conversation_id": foundryConfig.FoundryConversationID,
				"project_endpoint":        foundryConfig.ProjectEndpoint,
				"api_version":             foundryConfig.APIVersion,
				"imported_at":             now.Format(time.RFC3339),
				"item_count":              len(items.Data),
			}
			existingTraceByID.Logs = req.Logs
			existingTraceByID.Nodes = nodes
			existingTraceByID.UpdatedAt = now
			existingTraceByID.UpdatedBy = req.UserID

			if err := f.docDB.Traces().Update(ctx, existingTraceByID); err != nil {
				return "", fmt.Errorf("failed to update trace: %w", err)
			}
			return existingTraceByID.ID, nil
		}
	}

	// Create new trace
	traceID := "trace_" + uuid.New().String()

	// Determine context type
	contextType := models.TraceContextConversation
	if req.AutonomousAgentID != "" {
		contextType = models.TraceContextAutonomousAgent
	}

	// Create new trace
	trace := &models.Trace{
		ID:                traceID,
		TenantID:          req.TenantID,
		ApplicationID:     req.ApplicationID,
		ConversationID:    req.ConversationID,
		AutonomousAgentID: req.AutonomousAgentID,
		ContextType:       contextType,
		ReferenceID:       foundryConfig.FoundryConversationID,
		ReferenceName:     "Microsoft Foundry Conversation",
		ReferenceMetadata: map[string]interface{}{
			"foundry_conversation_id": foundryConfig.FoundryConversationID,
			"project_endpoint":        foundryConfig.ProjectEndpoint,
			"api_version":             foundryConfig.APIVersion,
			"imported_at":             now.Format(time.RFC3339),
			"item_count":              len(items.Data),
		},
		Logs:      req.Logs,
		Nodes:     nodes,
		CreatedAt: now,
		UpdatedAt: now,
		CreatedBy: req.UserID,
		UpdatedBy: req.UserID,
	}

	if err := f.docDB.Traces().Create(ctx, trace); err != nil {
		return "", fmt.Errorf("failed to create trace: %w", err)
	}

	return traceID, nil
}

// fetchConversationItems fetches conversation items from Microsoft Foundry.
func (f *TraceImporter) fetchConversationItems(ctx context.Context, config *FoundryConfig) (*ConversationItemsResponse, error) {
	// Build URL: {PROJECT_ENDPOINT}/openai/conversations/{FOUNDRY_CONV_ID}/items?api-version={VERSION}
	url := fmt.Sprintf("%s/openai/conversations/%s/items?api-version=%s",
		config.ProjectEndpoint,
		config.FoundryConversationID,
		config.APIVersion,
	)

	// Create request
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+config.FoundryAPIToken)

	// Execute request
	resp, err := f.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("failed to call Foundry API: %w", err)
	}
	defer resp.Body.Close()

	// Read response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	// Check status code
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("foundry API returned status %d: %s", resp.StatusCode, string(body))
	}

	// Parse response
	var items ConversationItemsResponse
	if err := json.Unmarshal(body, &items); err != nil {
		var rawData interface{}
		if jsonErr := json.Unmarshal(body, &rawData); jsonErr == nil {
			return &ConversationItemsResponse{
				Data: nil,
			}, nil
		}
		return nil, fmt.Errorf("failed to parse Foundry response: %w", err)
	}

	return &items, nil
}

// GetTransformer returns the transformer for testing purposes.
func (f *TraceImporter) GetTransformer() *Transformer {
	return f.transformer
}
