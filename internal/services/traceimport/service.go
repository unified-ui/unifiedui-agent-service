// Package traceimport provides functionality for importing traces from external systems.
package traceimport

import (
	"context"
	"fmt"

	"github.com/unifiedui/agent-service/internal/core/docdb"
	"github.com/unifiedui/agent-service/internal/services/platform"
)

// ImportService handles trace import operations using the factory pattern.
// It is backend-agnostic and delegates to specific importers via ImporterFactory.
type ImportService struct {
	factory *ImporterFactory
	queue   *JobQueue
}

// NewImportService creates a new import service.
// Use RegisterImporter to add backend-specific importers after creation.
func NewImportService(docDB docdb.Client) *ImportService {
	factory := NewImporterFactory(docDB)

	service := &ImportService{
		factory: factory,
	}

	// Create job queue with worker function
	service.queue = NewJobQueue(100, service.processJob)

	return service
}

// RegisterImporter registers a trace importer with the service.
// This should be called during initialization to add backend-specific importers.
func (s *ImportService) RegisterImporter(importer TraceImporter) {
	s.factory.Register(importer)
}

// Start starts the import service workers.
func (s *ImportService) Start(workerCount int) {
	s.queue.Start(workerCount)
}

// Stop stops the import service gracefully.
func (s *ImportService) Stop() {
	s.queue.Stop()
}

// HasImporter checks if an importer is registered for the specified agent type.
func (s *ImportService) HasImporter(agentType platform.AgentType) bool {
	return s.factory.HasImporter(agentType)
}

// Import imports traces from the specified backend synchronously.
// This is the main entry point for trace import operations.
func (s *ImportService) Import(ctx context.Context, agentType platform.AgentType, req *ImportRequest) (string, error) {
	importer, err := s.factory.GetImporter(agentType)
	if err != nil {
		return "", err
	}

	return importer.Import(ctx, req)
}

// EnqueueImport adds an import job to the background queue.
func (s *ImportService) EnqueueImport(agentType platform.AgentType, req *ImportRequest) error {
	if !s.factory.HasImporter(agentType) {
		return fmt.Errorf("no trace importer registered for agent type: %s", agentType)
	}

	job := &ImportJob{
		Type:   jobTypeFromAgentType(agentType),
		Action: JobActionImportConversationTraces,
		Config: JobConfig{
			TenantID:       req.TenantID,
			ConversationID: req.ConversationID,
			ApplicationID:  req.ApplicationID,
			Logs:           req.Logs,
			UserID:         req.UserID,
			BackendConfig:  req.BackendConfig,
		},
	}

	s.queue.Enqueue(job)
	return nil
}

// processJob processes an import job from the queue.
func (s *ImportService) processJob(ctx context.Context, job *ImportJob) error {
	agentType := agentTypeFromJobType(job.Type)

	req := &ImportRequest{
		TenantID:       job.Config.TenantID,
		ConversationID: job.Config.ConversationID,
		ApplicationID:  job.Config.ApplicationID,
		Logs:           job.Config.Logs,
		UserID:         job.Config.UserID,
		BackendConfig:  job.Config.BackendConfig,
	}

	_, err := s.Import(ctx, agentType, req)
	return err
}

// GetFactory returns the importer factory for direct access if needed.
func (s *ImportService) GetFactory() *ImporterFactory {
	return s.factory
}

// SupportedTypes returns a list of all supported agent types for trace import.
func (s *ImportService) SupportedTypes() []platform.AgentType {
	return s.factory.SupportedTypes()
}

// Helper functions for type conversion

func jobTypeFromAgentType(agentType platform.AgentType) JobType {
	switch agentType {
	case platform.AgentTypeFoundry:
		return JobTypeMicrosoftFoundry
	case platform.AgentTypeN8N:
		return JobTypeN8N
	default:
		return JobType(agentType)
	}
}

func agentTypeFromJobType(jobType JobType) platform.AgentType {
	switch jobType {
	case JobTypeMicrosoftFoundry:
		return platform.AgentTypeFoundry
	case JobTypeN8N:
		return platform.AgentTypeN8N
	default:
		return platform.AgentType(jobType)
	}
}
