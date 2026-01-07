// Package traceimport provides functionality for importing traces from external systems.
package traceimport

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
)

// TraceImporter defines the interface for trace importers.
type TraceImporter interface {
	// Import imports traces from an external system.
	Import(ctx context.Context, request *ImportRequest) error
}

// FoundryTraceImporter imports traces from Microsoft Foundry.
type FoundryTraceImporter struct {
	httpClient  *http.Client
	docDB       docdb.Client
	transformer *FoundryTransformer
}

// NewFoundryTraceImporter creates a new Foundry trace importer.
func NewFoundryTraceImporter(docDB docdb.Client) *FoundryTraceImporter {
	return &FoundryTraceImporter{
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		docDB:       docDB,
		transformer: NewFoundryTransformer(),
	}
}

// Import imports traces from Microsoft Foundry.
// Returns the trace ID on success.
func (f *FoundryTraceImporter) Import(ctx context.Context, req *FoundryImportRequest) (string, error) {
	// Fetch conversation items from Foundry
	items, err := f.fetchConversationItems(ctx, req)
	if err != nil {
		return "", fmt.Errorf("failed to fetch conversation items: %w", err)
	}

	// Transform Foundry items into TraceNodes
	var nodes []models.TraceNode
	if items != nil && len(items.Data) > 0 {
		nodes = f.transformer.Transform(items.Data, req.UserID)
	}

	// Check if trace already exists for this conversation
	existingTrace, err := f.docDB.Traces().GetByConversation(ctx, req.TenantID, req.ConversationID)
	if err != nil {
		return "", fmt.Errorf("failed to check existing trace: %w", err)
	}

	now := time.Now().UTC()

	if existingTrace != nil {
		// Update existing trace with new data
		existingTrace.ReferenceID = req.FoundryConversationID
		existingTrace.ReferenceName = "Microsoft Foundry Conversation"
		existingTrace.ReferenceMetadata = map[string]interface{}{
			"foundry_conversation_id": req.FoundryConversationID,
			"project_endpoint":        req.ProjectEndpoint,
			"api_version":             req.APIVersion,
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

	// Create new trace
	traceID := "trace_" + uuid.New().String()
	trace := &models.Trace{
		ID:             traceID,
		TenantID:       req.TenantID,
		ApplicationID:  req.ApplicationID,
		ConversationID: req.ConversationID,
		ContextType:    models.TraceContextConversation,
		ReferenceID:    req.FoundryConversationID,
		ReferenceName:  "Microsoft Foundry Conversation",
		ReferenceMetadata: map[string]interface{}{
			"foundry_conversation_id": req.FoundryConversationID,
			"project_endpoint":        req.ProjectEndpoint,
			"api_version":             req.APIVersion,
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
func (f *FoundryTraceImporter) fetchConversationItems(ctx context.Context, req *FoundryImportRequest) (*FoundryConversationItemsResponse, error) {
	// Build URL: {PROJECT_ENDPOINT}/openai/conversations/{FOUNDRY_CONV_ID}/items?api-version={VERSION}
	url := fmt.Sprintf("%s/openai/conversations/%s/items?api-version=%s",
		req.ProjectEndpoint,
		req.FoundryConversationID,
		req.APIVersion,
	)

	// Create request
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+req.FoundryAPIToken)

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
	var items FoundryConversationItemsResponse
	if err := json.Unmarshal(body, &items); err != nil {
		// If parsing fails, store raw JSON as fallback
		var rawData interface{}
		if jsonErr := json.Unmarshal(body, &rawData); jsonErr == nil {
			return &FoundryConversationItemsResponse{
				Data: nil, // Will use raw metadata instead
			}, nil
		}
		return nil, fmt.Errorf("failed to parse Foundry response: %w", err)
	}

	return &items, nil
}

// ImportService handles trace import operations.
type ImportService struct {
	docDB           docdb.Client
	foundryImporter *FoundryTraceImporter
	queue           *JobQueue
}

// NewImportService creates a new import service.
func NewImportService(docDB docdb.Client) *ImportService {
	foundryImporter := NewFoundryTraceImporter(docDB)

	service := &ImportService{
		docDB:           docDB,
		foundryImporter: foundryImporter,
	}

	// Create job queue with worker function
	service.queue = NewJobQueue(100, service.processJob)

	return service
}

// Start starts the import service workers.
func (s *ImportService) Start(workerCount int) {
	s.queue.Start(workerCount)
}

// Stop stops the import service gracefully.
func (s *ImportService) Stop() {
	s.queue.Stop()
}

// EnqueueFoundryImport adds a Foundry import job to the queue.
func (s *ImportService) EnqueueFoundryImport(req *FoundryImportRequest) {
	job := &ImportJob{
		Type:   JobTypeMicrosoftFoundry,
		Action: JobActionImportConversationTraces,
		Config: JobConfig{
			TenantID:       req.TenantID,
			ConversationID: req.ConversationID,
			ApplicationID:  req.ApplicationID,
			Logs:           req.Logs,
			UserID:         req.UserID,
			FoundryConfig: &FoundryJobConfig{
				FoundryConversationID: req.FoundryConversationID,
				ProjectEndpoint:       req.ProjectEndpoint,
				APIVersion:            req.APIVersion,
				FoundryAPIToken:       req.FoundryAPIToken,
			},
		},
	}

	s.queue.Enqueue(job)
}

// ImportFoundryTraces imports traces from Foundry synchronously.
// Use this for the PUT /traces/import/refresh endpoint.
// Returns the trace ID on success.
func (s *ImportService) ImportFoundryTraces(ctx context.Context, req *FoundryImportRequest) (string, error) {
	return s.foundryImporter.Import(ctx, req)
}

// processJob processes an import job from the queue.
func (s *ImportService) processJob(ctx context.Context, job *ImportJob) error {
	switch job.Type {
	case JobTypeMicrosoftFoundry:
		return s.processFoundryJob(ctx, job)
	case JobTypeN8N:
		// TODO: Implement N8N trace import
		return fmt.Errorf("N8N trace import not yet implemented")
	default:
		return fmt.Errorf("unknown job type: %s", job.Type)
	}
}

// processFoundryJob processes a Foundry import job.
func (s *ImportService) processFoundryJob(ctx context.Context, job *ImportJob) error {
	if job.Config.FoundryConfig == nil {
		return fmt.Errorf("foundry config is required for Foundry jobs")
	}

	req := &FoundryImportRequest{
		ImportRequest: ImportRequest{
			TenantID:       job.Config.TenantID,
			ConversationID: job.Config.ConversationID,
			ApplicationID:  job.Config.ApplicationID,
			Logs:           job.Config.Logs,
			UserID:         job.Config.UserID,
		},
		FoundryConversationID: job.Config.FoundryConfig.FoundryConversationID,
		ProjectEndpoint:       job.Config.FoundryConfig.ProjectEndpoint,
		APIVersion:            job.Config.FoundryConfig.APIVersion,
		FoundryAPIToken:       job.Config.FoundryConfig.FoundryAPIToken,
	}

	_, err := s.foundryImporter.Import(ctx, req)
	return err
}

// GetFoundryImporter returns the Foundry importer for direct use.
func (s *ImportService) GetFoundryImporter() *FoundryTraceImporter {
	return s.foundryImporter
}
