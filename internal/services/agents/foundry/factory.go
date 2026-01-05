// Package foundry provides Microsoft Foundry agent client implementations.
package foundry

import (
	"fmt"

	"github.com/unifiedui/agent-service/internal/services/platform"
)

// Factory creates Foundry-specific agent clients.
type Factory struct{}

// NewFactory creates a new Foundry factory.
func NewFactory() *Factory {
	return &Factory{}
}

// CreateWorkflowClient creates a Foundry workflow client from platform configuration.
func (f *Factory) CreateWorkflowClient(config *platform.AgentConfig, apiToken string) (*WorkflowClient, error) {
	return NewFromConfig(config, apiToken)
}

// NewFromConfig creates a Foundry workflow client from platform configuration.
// This is a convenience function that can be used without instantiating a Factory.
func NewFromConfig(config *platform.AgentConfig, apiToken string) (*WorkflowClient, error) {
	if config == nil {
		return nil, fmt.Errorf("config is required")
	}
	return NewWorkflowClient(&WorkflowClientConfig{
		ProjectEndpoint: config.Settings.ProjectEndpoint,
		APIVersion:      config.Settings.APIVersion,
		AgentName:       config.Settings.AgentName,
		AgentType:       config.Settings.AgentType,
		APIToken:        apiToken,
	})
}
