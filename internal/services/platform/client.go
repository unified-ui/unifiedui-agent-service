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
	GetChatAgentConfig(ctx context.Context, tenantID, chatAgentID, authToken string) (*ChatAgentConfigResponse, error)
	GetAgentConfig(ctx context.Context, tenantID, chatAgentID, conversationID, authToken string) (*AgentConfig, error)
	GetAgentConfigFromFile(ctx context.Context, tenantID, chatAgentID string) (*AgentConfig, error)
	GetMe(ctx context.Context, authToken string) (*UserInfo, error)
	GetConversation(ctx context.Context, tenantID, conversationID, authToken string) (*ConversationResponse, error)
	ValidateConversation(ctx context.Context, tenantID, conversationID, authToken string) error
	ValidateAutonomousAgent(ctx context.Context, tenantID, autonomousAgentID, authToken string) error
	GetAutonomousAgentConfig(ctx context.Context, tenantID, autonomousAgentID, apiKey string) (*AutonomousAgentConfigResponse, error)
	GetAutonomousAgentConfigWithBearer(ctx context.Context, tenantID, autonomousAgentID, authToken string) (*AutonomousAgentConfigResponse, error)
	ValidateAutonomousAgentAPIKey(ctx context.Context, tenantID, autonomousAgentID, apiKey string) error
	GetAIModelsByPurpose(ctx context.Context, tenantID, purposeGroup, modelType string) ([]AIModelWithSecretResponse, error)
	GetCredentialSecret(ctx context.Context, tenantID, credentialID, authToken string) (string, error)
	UpdateConversationTitle(ctx context.Context, tenantID, conversationID, title, authToken string) error
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

func (c *client) GetChatAgentConfig(ctx context.Context, tenantID, chatAgentID, authToken string) (*ChatAgentConfigResponse, error) {
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
	return doJSONRequest[ChatAgentConfigResponse](ctx, c, requestConfig{method: http.MethodGet, url: url, headers: headers}, "chat agent config not found")
}

func (c *client) GetAgentConfig(ctx context.Context, tenantID, chatAgentID, conversationID, authToken string) (*AgentConfig, error) {
	appConfig, err := c.GetChatAgentConfig(ctx, tenantID, chatAgentID, authToken)
	if err != nil {
		return nil, fmt.Errorf("failed to get chat agent config: %w", err)
	}

	return &AgentConfig{
		DocVersion:     appConfig.DocVersion,
		Type:           appConfig.Type,
		TenantID:       appConfig.TenantID,
		ConversationID: conversationID,
		ChatAgentID:    appConfig.ChatAgentID,
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

func (c *client) ValidateAutonomousAgent(ctx context.Context, tenantID, autonomousAgentID, authToken string) error {
	if c.baseURL == "" {
		return nil
	}
	if authToken == "" {
		return fmt.Errorf("auth token not provided")
	}
	url := fmt.Sprintf("%s/api/v1/platform-service/tenants/%s/autonomous-agents/%s", c.baseURL, tenantID, autonomousAgentID)
	return doValidateRequest(ctx, c, requestConfig{method: http.MethodGet, url: url, headers: bearerHeaders(c, authToken)}, "autonomous agent not found")
}

func (c *client) GetAutonomousAgentConfig(ctx context.Context, tenantID, autonomousAgentID, apiKey string) (*AutonomousAgentConfigResponse, error) {
	if c.baseURL == "" {
		return nil, fmt.Errorf("platform service URL not configured")
	}
	if apiKey == "" {
		return nil, fmt.Errorf("API key not provided")
	}
	url := fmt.Sprintf("%s/api/v1/platform-service/tenants/%s/autonomous-agents/%s/config", c.baseURL, tenantID, autonomousAgentID)
	return doJSONRequest[AutonomousAgentConfigResponse](ctx, c, requestConfig{method: http.MethodGet, url: url, headers: apiKeyHeaders(apiKey)}, "autonomous agent not found")
}

func (c *client) GetAutonomousAgentConfigWithBearer(ctx context.Context, tenantID, autonomousAgentID, authToken string) (*AutonomousAgentConfigResponse, error) {
	if c.baseURL == "" {
		return nil, fmt.Errorf("platform service URL not configured")
	}
	if authToken == "" {
		return nil, fmt.Errorf("auth token not provided")
	}
	url := fmt.Sprintf("%s/api/v1/platform-service/tenants/%s/autonomous-agents/%s/config/bearer", c.baseURL, tenantID, autonomousAgentID)
	return doJSONRequest[AutonomousAgentConfigResponse](ctx, c, requestConfig{method: http.MethodGet, url: url, headers: bearerHeaders(c, authToken)}, "autonomous agent not found")
}

func (c *client) ValidateAutonomousAgentAPIKey(ctx context.Context, tenantID, autonomousAgentID, apiKey string) error {
	if c.baseURL == "" {
		return nil
	}
	if apiKey == "" {
		return fmt.Errorf("API key not provided")
	}
	url := fmt.Sprintf("%s/api/v1/platform-service/tenants/%s/autonomous-agents/%s/validate-api-key", c.baseURL, tenantID, autonomousAgentID)
	return doValidateRequest(ctx, c, requestConfig{method: http.MethodPost, url: url, headers: apiKeyHeaders(apiKey)}, "autonomous agent not found")
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
