package react

import (
	"fmt"
	"net/http"
	"time"

	"github.com/unifiedui/agent-service/internal/config"
	"github.com/unifiedui/agent-service/internal/services/platform"
)

// Factory creates ReACT agent clients based on configuration.
type Factory struct {
	reactCfg   config.ReactServiceConfig
	serviceKey string
}

// NewFactory creates a new ReACT agent factory.
func NewFactory(reactCfg config.ReactServiceConfig, serviceKey string) *Factory {
	return &Factory{
		reactCfg:   reactCfg,
		serviceKey: serviceKey,
	}
}

// CreateWorkflowClient creates a ReACT workflow client from agent configuration.
func (f *Factory) CreateWorkflowClient(agentConfig *platform.AgentConfig) (*WorkflowClient, error) {
	if agentConfig == nil {
		return nil, fmt.Errorf("agent config is required")
	}

	if f.reactCfg.URL == "" {
		return nil, fmt.Errorf("ReACT service URL not configured")
	}

	timeout := f.reactCfg.Timeout
	if timeout == 0 {
		timeout = 300 * time.Second
	}

	clientCfg := &WorkflowClientConfig{
		BaseURL:    f.reactCfg.URL,
		ServiceKey: f.serviceKey,
		HTTPClient: &http.Client{
			Timeout: timeout,
		},
	}

	return NewWorkflowClient(clientCfg), nil
}
