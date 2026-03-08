// Package traceimport provides functionality for importing traces from external systems.
package traceimport

import (
	"fmt"
	"sync"

	"github.com/unifiedui/agent-service/internal/core/docdb"
	"github.com/unifiedui/agent-service/internal/services/platform"
)

// ImporterFactory manages trace importer instances for different backend types.
type ImporterFactory struct {
	docDB     docdb.Client
	importers map[platform.AgentType]TraceImporter
	mu        sync.RWMutex
}

// NewImporterFactory creates a new importer factory.
// Use RegisterImporter to add importers after creation.
func NewImporterFactory(docDB docdb.Client) *ImporterFactory {
	return &ImporterFactory{
		docDB:     docDB,
		importers: make(map[platform.AgentType]TraceImporter),
	}
}

// Register registers a trace importer for its agent type.
func (f *ImporterFactory) Register(importer TraceImporter) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.importers[importer.Type()] = importer
}

// GetImporter returns the importer for the specified agent type.
func (f *ImporterFactory) GetImporter(agentType platform.AgentType) (TraceImporter, error) {
	f.mu.RLock()
	defer f.mu.RUnlock()

	importer, ok := f.importers[agentType]
	if !ok {
		return nil, fmt.Errorf("no trace importer registered for agent type: %s", agentType)
	}

	return importer, nil
}

// HasImporter checks if an importer is registered for the specified agent type.
func (f *ImporterFactory) HasImporter(agentType platform.AgentType) bool {
	f.mu.RLock()
	defer f.mu.RUnlock()
	_, ok := f.importers[agentType]
	return ok
}

// SupportedTypes returns a list of all supported agent types.
func (f *ImporterFactory) SupportedTypes() []platform.AgentType {
	f.mu.RLock()
	defer f.mu.RUnlock()

	types := make([]platform.AgentType, 0, len(f.importers))
	for t := range f.importers {
		types = append(types, t)
	}
	return types
}
