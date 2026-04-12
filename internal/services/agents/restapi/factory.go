// Package restapi provides REST API agent client implementations.
package restapi

import (
	"fmt"

	"github.com/unifiedui/agent-service/internal/services/platform"
)

// Factory creates REST API agent clients.
type Factory struct{}

// NewFactory creates a new REST API factory.
func NewFactory() *Factory {
	return &Factory{}
}

// CreateWorkflowClient creates a REST API workflow client from agent configuration.
func (f *Factory) CreateWorkflowClient(config *platform.AgentConfig, userToken string) (*WorkflowClient, error) {
	if config == nil {
		return nil, fmt.Errorf("config is required")
	}

	settings := config.Settings
	if settings.InvokeEndpoint == "" {
		return nil, fmt.Errorf("invoke endpoint is required")
	}

	return NewWorkflowClient(&WorkflowClientConfig{
		InvokeEndpoint:             settings.InvokeEndpoint,
		CreateConversationEndpoint: settings.CreateConversationEndpoint,
		AuthType:                   AuthType(settings.AuthType),
		Credential:                 settings.Credential,
		AccessToken:                settings.AccessToken,
		UserToken:                  userToken,
		APIKeyHeaderName:           settings.APIKeyHeaderName,
		UseUnifiedChatHistory:      settings.UseUnifiedChatHistory,
		ChatHistoryCount:           settings.ChatHistoryCount,
	})
}
