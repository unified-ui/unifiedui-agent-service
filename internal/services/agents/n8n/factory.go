// Package n8n provides N8N-specific agent client implementations.
package n8n

import (
	"fmt"

	"github.com/unifiedui/agent-service/internal/services/platform"
)

// Factory creates N8N-specific agent clients.
type Factory struct{}

// NewFactory creates a new N8N factory.
func NewFactory() *Factory {
	return &Factory{}
}

// CreateWorkflowClient creates a workflow client based on the workflow type.
func (f *Factory) CreateWorkflowClient(config *platform.AgentConfig) (*ChatWorkflowClient, error) {
	if config == nil {
		return nil, fmt.Errorf("config is required")
	}

	workflowType := WorkflowType(config.Settings.WorkflowType)

	switch workflowType {
	case WorkflowTypeChatAgent:
		return f.createChatWorkflowClient(config)
	case WorkflowTypeHumanInLoop:
		return nil, fmt.Errorf("human-in-the-loop workflow not yet implemented")
	default:
		return nil, fmt.Errorf("unsupported N8N workflow type: %s", workflowType)
	}
}

// CreateAPIClient creates an API client based on the API version.
func (f *Factory) CreateAPIClient(config *platform.AgentConfig) (*APIClient, error) {
	if config == nil {
		return nil, fmt.Errorf("config is required")
	}

	apiVersion := APIVersion(config.Settings.APIVersion)

	switch apiVersion {
	case APIVersionV1:
		return f.createAPIClientV1(config)
	default:
		return nil, fmt.Errorf("unsupported N8N API version: %s", apiVersion)
	}
}

// createChatWorkflowClient creates a chat workflow client.
func (f *Factory) createChatWorkflowClient(config *platform.AgentConfig) (*ChatWorkflowClient, error) {
	if config.Settings.ChatURL == "" {
		return nil, fmt.Errorf("chat_url is required for chat workflow")
	}

	// Get credentials
	var username, password string
	if config.Settings.ChatCredentials != nil {
		basicAuth := config.Settings.ChatCredentials.GetSecretAsBasicAuth()
		if basicAuth != nil {
			username = basicAuth.Username
			password = basicAuth.Password
		}
	}

	clientConfig := &ChatWorkflowConfig{
		ChatURL:  config.Settings.ChatURL,
		Username: username,
		Password: password,
	}

	return NewChatWorkflowClient(clientConfig)
}

// createAPIClientV1 creates an API client v1.
func (f *Factory) createAPIClientV1(config *platform.AgentConfig) (*APIClient, error) {
	// Extract base URL from chat URL (remove webhook path)
	// For now, we'll use a placeholder since API operations are secondary
	baseURL := "http://localhost:5678" // TODO: Extract from config

	var apiKey string
	if config.Settings.APICredentials != nil {
		apiKey = config.Settings.APICredentials.GetSecretAsString()
	}

	clientConfig := &APIClientConfig{
		BaseURL: baseURL,
		APIKey:  apiKey,
	}

	return NewAPIClient(clientConfig)
}
