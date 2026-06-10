package platform

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"time"
)

// Client defines the interface for the platform service client.
type Client interface {
	GetChatAgentConfig(ctx context.Context, tenantID, chatAgentID, authToken string, useCache bool) (*ChatAgentConfigResponse, error)
	GetAgentConfig(ctx context.Context, tenantID, chatAgentID, conversationID, authToken string, useCache bool) (*AgentConfig, error)
	GetAgentConfigFromFile(ctx context.Context, tenantID, chatAgentID string) (*AgentConfig, error)
	GetMe(ctx context.Context, authToken string) (*UserInfo, error)
	GetConversation(ctx context.Context, tenantID, conversationID, authToken string) (*ConversationResponse, error)
	ValidateConversation(ctx context.Context, tenantID, conversationID, authToken string) error
	ValidateWorkflow(ctx context.Context, tenantID, workflowID, authToken string) error
	GetWorkflowConfig(ctx context.Context, tenantID, workflowID, apiKey string) (*WorkflowConfigResponse, error)
	GetWorkflowConfigWithBearer(ctx context.Context, tenantID, workflowID, authToken string) (*WorkflowConfigResponse, error)
	ValidateWorkflowAPIKey(ctx context.Context, tenantID, workflowID, apiKey string) error
	GetAIModelsByPurpose(ctx context.Context, tenantID, purposeGroup, modelType string) ([]AIModelWithSecretResponse, error)
	GetCredentialSecret(ctx context.Context, tenantID, credentialID, authToken string) (string, error)
	UpdateConversationTitle(ctx context.Context, tenantID, conversationID, title, authToken string) error
	UpsertMessageFeedback(ctx context.Context, tenantID, conversationID, messageID, authToken string, payload UpsertMessageFeedbackRequest) error
	DeleteMessageFeedback(ctx context.Context, tenantID, conversationID, messageID, authToken string) error
}

type client struct {
	configPath string
	baseURL    string
	serviceKey string
	httpClient *http.Client
}

// ClientConfig holds the configuration for the platform client.
type ClientConfig struct {
	BaseURL    string
	ConfigPath string
	ServiceKey string
	Timeout    time.Duration
}

// NewClient creates a new platform service client.
func NewClient(cfg *ClientConfig) Client {
	timeout := cfg.Timeout
	if timeout == 0 {
		timeout = 30 * time.Second
	}

	return &client{
		configPath: cfg.ConfigPath,
		baseURL:    cfg.BaseURL,
		serviceKey: cfg.ServiceKey,
		httpClient: &http.Client{
			Timeout: timeout,
		},
	}
}

func (c *client) GetChatAgentConfig(ctx context.Context, tenantID, chatAgentID, authToken string, useCache bool) (*ChatAgentConfigResponse, error) {
	if c.baseURL == "" {
		return nil, fmt.Errorf("platform service URL not configured")
	}
	if c.serviceKey == "" {
		return nil, fmt.Errorf("service key not configured")
	}
	if authToken == "" {
		return nil, fmt.Errorf("auth token not provided")
	}
	url := fmt.Sprintf("%s/api/v1/platform-service/tenants/%s/chat-agents/%s/config", c.baseURL, tenantID, chatAgentID)
	headers := map[string]string{
		"X-Service-Key": c.serviceKey,
		"Authorization": "Bearer " + authToken,
	}
	if !useCache {
		headers["X-Use-Cache"] = "false"
	}
	return doJSONRequest[ChatAgentConfigResponse](ctx, c, requestConfig{method: http.MethodGet, url: url, headers: headers}, "chat agent config not found")
}

func (c *client) GetAgentConfig(ctx context.Context, tenantID, chatAgentID, conversationID, authToken string, useCache bool) (*AgentConfig, error) {
	appConfig, err := c.GetChatAgentConfig(ctx, tenantID, chatAgentID, authToken, useCache)
	if err != nil {
		return nil, fmt.Errorf("failed to get chat agent config: %w", err)
	}

	return &AgentConfig{
		DocVersion:     appConfig.DocVersion,
		Type:           appConfig.Type,
		TenantID:       appConfig.TenantID,
		ConversationID: conversationID,
		ChatAgentID:    appConfig.ChatAgentID,
		IsActive:       appConfig.IsActive,
		Settings:       appConfig.Settings,
		User:           appConfig.User,
	}, nil
}

func (c *client) GetAgentConfigFromFile(ctx context.Context, tenantID, chatAgentID string) (*AgentConfig, error) {
	if c.configPath == "" {
		return nil, fmt.Errorf("config path not configured")
	}

	absPath, err := filepath.Abs(c.configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve config path: %w", err)
	}

	data, err := os.ReadFile(absPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var config AgentConfig
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	return &config, nil
}

func (c *client) GetMe(ctx context.Context, authToken string) (*UserInfo, error) {
	if c.baseURL == "" {
		return nil, fmt.Errorf("platform service URL not configured")
	}
	if authToken == "" {
		return nil, fmt.Errorf("auth token not provided")
	}
	url := fmt.Sprintf("%s/api/v1/platform-service/identity/me", c.baseURL)
	return doJSONRequest[UserInfo](ctx, c, requestConfig{method: http.MethodGet, url: url, headers: bearerHeaders(c, authToken)}, "user not found")
}

func (c *client) GetConversation(ctx context.Context, tenantID, conversationID, authToken string) (*ConversationResponse, error) {
	if c.baseURL == "" {
		return nil, fmt.Errorf("platform service URL not configured")
	}
	if authToken == "" {
		return nil, fmt.Errorf("auth token not provided")
	}
	url := fmt.Sprintf("%s/api/v1/platform-service/tenants/%s/conversations/%s", c.baseURL, tenantID, conversationID)
	return doJSONRequest[ConversationResponse](ctx, c, requestConfig{method: http.MethodGet, url: url, headers: bearerHeaders(c, authToken)}, "conversation not found")
}

func (c *client) ValidateConversation(ctx context.Context, tenantID, conversationID, authToken string) error {
	if c.baseURL == "" {
		return nil
	}
	if authToken == "" {
		return fmt.Errorf("auth token not provided")
	}
	url := fmt.Sprintf("%s/api/v1/platform-service/tenants/%s/conversations/%s", c.baseURL, tenantID, conversationID)
	return doValidateRequest(ctx, c, requestConfig{method: http.MethodGet, url: url, headers: bearerHeaders(c, authToken)}, "conversation not found")
}

func (c *client) ValidateWorkflow(ctx context.Context, tenantID, workflowID, authToken string) error {
	if c.baseURL == "" {
		return nil
	}
	if authToken == "" {
		return fmt.Errorf("auth token not provided")
	}
	url := fmt.Sprintf("%s/api/v1/platform-service/tenants/%s/workflows/%s", c.baseURL, tenantID, workflowID)
	return doValidateRequest(ctx, c, requestConfig{method: http.MethodGet, url: url, headers: bearerHeaders(c, authToken)}, "workflow not found")
}

func (c *client) GetWorkflowConfig(ctx context.Context, tenantID, workflowID, apiKey string) (*WorkflowConfigResponse, error) {
	if c.baseURL == "" {
		return nil, fmt.Errorf("platform service URL not configured")
	}
	if apiKey == "" {
		return nil, fmt.Errorf("API key not provided")
	}
	url := fmt.Sprintf("%s/api/v1/platform-service/tenants/%s/workflows/%s/config", c.baseURL, tenantID, workflowID)
	return doJSONRequest[WorkflowConfigResponse](ctx, c, requestConfig{method: http.MethodGet, url: url, headers: apiKeyHeaders(apiKey)}, "workflow not found")
}

func (c *client) GetWorkflowConfigWithBearer(ctx context.Context, tenantID, workflowID, authToken string) (*WorkflowConfigResponse, error) {
	if c.baseURL == "" {
		return nil, fmt.Errorf("platform service URL not configured")
	}
	if authToken == "" {
		return nil, fmt.Errorf("auth token not provided")
	}
	url := fmt.Sprintf("%s/api/v1/platform-service/tenants/%s/workflows/%s/config/bearer", c.baseURL, tenantID, workflowID)
	return doJSONRequest[WorkflowConfigResponse](ctx, c, requestConfig{method: http.MethodGet, url: url, headers: bearerHeaders(c, authToken)}, "workflow not found")
}

func (c *client) ValidateWorkflowAPIKey(ctx context.Context, tenantID, workflowID, apiKey string) error {
	if c.baseURL == "" {
		return nil
	}
	if apiKey == "" {
		return fmt.Errorf("API key not provided")
	}
	url := fmt.Sprintf("%s/api/v1/platform-service/tenants/%s/workflows/%s/validate-api-key", c.baseURL, tenantID, workflowID)
	return doValidateRequest(ctx, c, requestConfig{method: http.MethodPost, url: url, headers: apiKeyHeaders(apiKey)}, "workflow not found")
}

func (c *client) GetAIModelsByPurpose(ctx context.Context, tenantID, purposeGroup, modelType string) ([]AIModelWithSecretResponse, error) {
	if c.baseURL == "" {
		return nil, fmt.Errorf("platform service URL not configured")
	}
	if c.serviceKey == "" {
		return nil, fmt.Errorf("service key not configured")
	}
	url := fmt.Sprintf("%s/api/v1/platform-service/tenants/%s/ai-models/by-purpose/%s", c.baseURL, tenantID, purposeGroup)
	if modelType != "" {
		url += "?model_type=" + modelType
	}
	return doJSONSliceRequest[AIModelWithSecretResponse](ctx, c, requestConfig{method: http.MethodGet, url: url, headers: serviceKeyHeaders(c)}, "AI models not found")
}

func (c *client) GetCredentialSecret(ctx context.Context, tenantID, credentialID, authToken string) (string, error) {
	if c.baseURL == "" {
		return "", fmt.Errorf("platform service URL not configured")
	}
	if authToken == "" {
		return "", fmt.Errorf("auth token not provided")
	}
	url := fmt.Sprintf("%s/api/v1/platform-service/tenants/%s/credentials/%s/secret", c.baseURL, tenantID, credentialID)
	secretResp, err := doJSONRequest[CredentialSecretResponse](ctx, c, requestConfig{method: http.MethodGet, url: url, headers: bearerHeaders(c, authToken)}, "credential not found")
	if err != nil {
		return "", err
	}
	return secretResp.SecretValue, nil
}

func (c *client) UpdateConversationTitle(ctx context.Context, tenantID, conversationID, title, authToken string) error {
	if c.baseURL == "" {
		return fmt.Errorf("platform service URL not configured")
	}
	if authToken == "" {
		return fmt.Errorf("auth token not provided")
	}
	url := fmt.Sprintf("%s/api/v1/platform-service/tenants/%s/conversations/%s", c.baseURL, tenantID, conversationID)
	payload, err := json.Marshal(map[string]string{"name": title})
	if err != nil {
		return fmt.Errorf("failed to marshal request body: %w", err)
	}
	headers := bearerHeaders(c, authToken)
	headers["Content-Type"] = "application/json"
	return doValidateRequest(ctx, c, requestConfig{method: http.MethodPatch, url: url, body: bytes.NewReader(payload), headers: headers}, "conversation not found")
}

// UpsertMessageFeedback proxies a reaction/feedback upsert to the platform service.
func (c *client) UpsertMessageFeedback(ctx context.Context, tenantID, conversationID, messageID, authToken string, payload UpsertMessageFeedbackRequest) error {
	if c.baseURL == "" {
		return fmt.Errorf("platform service URL not configured")
	}
	if authToken == "" {
		return fmt.Errorf("auth token not provided")
	}
	url := fmt.Sprintf("%s/api/v1/platform-service/tenants/%s/conversations/%s/messages/%s/feedback", c.baseURL, tenantID, conversationID, messageID)
	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal feedback payload: %w", err)
	}
	headers := bearerHeaders(c, authToken)
	headers["Content-Type"] = "application/json"
	return doValidateRequest(ctx, c, requestConfig{method: http.MethodPost, url: url, body: bytes.NewReader(body), headers: headers}, "conversation or message not found")
}

// DeleteMessageFeedback proxies a reaction/feedback delete to the platform service.
func (c *client) DeleteMessageFeedback(ctx context.Context, tenantID, conversationID, messageID, authToken string) error {
	if c.baseURL == "" {
		return fmt.Errorf("platform service URL not configured")
	}
	if authToken == "" {
		return fmt.Errorf("auth token not provided")
	}
	url := fmt.Sprintf("%s/api/v1/platform-service/tenants/%s/conversations/%s/messages/%s/feedback", c.baseURL, tenantID, conversationID, messageID)
	return doValidateRequest(ctx, c, requestConfig{method: http.MethodDelete, url: url, headers: bearerHeaders(c, authToken)}, "feedback not found")
}
