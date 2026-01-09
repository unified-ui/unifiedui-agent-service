// Package platform provides the platform service client for configuration retrieval.
package platform

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"time"
)

// Client defines the interface for the platform service client.
type Client interface {
	// GetApplicationConfig retrieves the application configuration from the platform service.
	// This calls the /config endpoint with service key AND bearer token authentication.
	// The authToken is the user's bearer token forwarded from the incoming request.
	GetApplicationConfig(ctx context.Context, tenantID, applicationID, authToken string) (*ApplicationConfigResponse, error)

	// GetAgentConfig retrieves the agent configuration for a given application.
	// This converts ApplicationConfigResponse to AgentConfig with conversation ID.
	GetAgentConfig(ctx context.Context, tenantID, applicationID, conversationID, authToken string) (*AgentConfig, error)

	// GetAgentConfigFromFile reads agent configuration from a local file (for development).
	GetAgentConfigFromFile(ctx context.Context, tenantID, applicationID string) (*AgentConfig, error)

	// GetMe retrieves the current user information from the platform service.
	// Note: The identity/me endpoint doesn't require tenantId.
	GetMe(ctx context.Context, authToken string) (*UserInfo, error)

	// GetConversation retrieves conversation details from the platform service.
	GetConversation(ctx context.Context, tenantID, conversationID, authToken string) (*ConversationResponse, error)

	// ValidateConversation validates that a conversation exists and user has access.
	ValidateConversation(ctx context.Context, tenantID, conversationID, authToken string) error

	// ValidateAutonomousAgent validates that an autonomous agent exists and user has access.
	ValidateAutonomousAgent(ctx context.Context, tenantID, autonomousAgentID, authToken string) error

	// GetAutonomousAgentConfig retrieves the autonomous agent configuration from the platform service.
	// This uses X-Unified-UI-Autonomous-Agent-API-Key header for authentication (NOT Bearer token).
	// apiKey is the autonomous agent's API key that will be validated against primary/secondary keys.
	GetAutonomousAgentConfig(ctx context.Context, tenantID, autonomousAgentID, apiKey string) (*AutonomousAgentConfigResponse, error)
}

// client implements the Client interface.
type client struct {
	configPath string
	baseURL    string
	serviceKey string
	httpClient *http.Client
}

// ClientConfig holds the configuration for the platform client.
type ClientConfig struct {
	// BaseURL is the URL of the Platform Service
	BaseURL string
	// ConfigPath is the path to the local config.json file (for development fallback)
	ConfigPath string
	// ServiceKey is the X_AGENT_SERVICE_KEY for service-to-service authentication
	ServiceKey string
	// Timeout for HTTP requests
	Timeout time.Duration
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

// GetApplicationConfig retrieves the application configuration from the platform service.
// It requires both X-Service-Key AND Bearer token for authentication.
func (c *client) GetApplicationConfig(ctx context.Context, tenantID, applicationID, authToken string) (*ApplicationConfigResponse, error) {
	if c.baseURL == "" {
		return nil, fmt.Errorf("platform service URL not configured")
	}

	if c.serviceKey == "" {
		return nil, fmt.Errorf("service key not configured")
	}

	if authToken == "" {
		return nil, fmt.Errorf("auth token not provided")
	}

	// Build request URL - use /config endpoint
	url := fmt.Sprintf("%s/api/v1/platform-service/tenants/%s/applications/%s/config", c.baseURL, tenantID, applicationID)

	// Create request
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers - both X-Service-Key AND Bearer token required
	req.Header.Set("X-Service-Key", c.serviceKey)
	req.Header.Set("Authorization", "Bearer "+authToken)
	req.Header.Set("Accept", "application/json")

	// Execute request
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to call platform service: %w", err)
	}
	defer resp.Body.Close()

	// Read response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	// Check status code
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("platform service returned status %d: %s", resp.StatusCode, string(body))
	}

	// Parse response
	var config ApplicationConfigResponse
	if err := json.Unmarshal(body, &config); err != nil {
		return nil, fmt.Errorf("failed to parse application config response: %w", err)
	}

	return &config, nil
}

// GetAgentConfig retrieves the agent configuration by calling the platform service
// and enriching it with conversation ID. User info is now included in the response.
func (c *client) GetAgentConfig(ctx context.Context, tenantID, applicationID, conversationID, authToken string) (*AgentConfig, error) {
	// Get application config from platform service (includes user info)
	appConfig, err := c.GetApplicationConfig(ctx, tenantID, applicationID, authToken)
	if err != nil {
		return nil, fmt.Errorf("failed to get application config: %w", err)
	}

	// Convert to AgentConfig
	agentConfig := &AgentConfig{
		DocVersion:     appConfig.DocVersion,
		Type:           appConfig.Type,
		TenantID:       appConfig.TenantID,
		ConversationID: conversationID,
		ApplicationID:  appConfig.ApplicationID,
		Settings:       appConfig.Settings,
		User:           appConfig.User, // User info from platform service response
	}

	return agentConfig, nil
}

// GetAgentConfigFromFile reads agent configuration from a local file.
// This is for development/fallback purposes when platform service is not available.
func (c *client) GetAgentConfigFromFile(ctx context.Context, tenantID, applicationID string) (*AgentConfig, error) {
	if c.configPath == "" {
		return nil, fmt.Errorf("config path not configured")
	}

	// Resolve absolute path
	absPath, err := filepath.Abs(c.configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve config path: %w", err)
	}

	// Read config file
	data, err := os.ReadFile(absPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	// Parse JSON
	var config AgentConfig
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	return &config, nil
}

// GetMe retrieves the current user information from the platform service.
// Note: The identity/me endpoint doesn't require tenantId in the path.
func (c *client) GetMe(ctx context.Context, authToken string) (*UserInfo, error) {
	if c.baseURL == "" {
		return nil, fmt.Errorf("platform service URL not configured")
	}

	if authToken == "" {
		return nil, fmt.Errorf("auth token not provided")
	}

	// Build request URL - identity/me endpoint doesn't need tenantId
	url := fmt.Sprintf("%s/api/v1/platform-service/identity/me", c.baseURL)

	// Create request
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers
	if c.serviceKey != "" {
		req.Header.Set("X-Service-Key", c.serviceKey)
	}
	req.Header.Set("Authorization", "Bearer "+authToken)
	req.Header.Set("Accept", "application/json")

	// Execute request
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to call platform service: %w", err)
	}
	defer resp.Body.Close()

	// Read response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	// Check status code
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("platform service returned status %d: %s", resp.StatusCode, string(body))
	}

	// Parse response
	var userInfo UserInfo
	if err := json.Unmarshal(body, &userInfo); err != nil {
		return nil, fmt.Errorf("failed to parse user info response: %w", err)
	}

	return &userInfo, nil
}

// GetConversation retrieves conversation details from the platform service.
func (c *client) GetConversation(ctx context.Context, tenantID, conversationID, authToken string) (*ConversationResponse, error) {
	if c.baseURL == "" {
		return nil, fmt.Errorf("platform service URL not configured")
	}

	if authToken == "" {
		return nil, fmt.Errorf("auth token not provided")
	}

	// Build request URL
	url := fmt.Sprintf("%s/api/v1/platform-service/tenants/%s/conversations/%s", c.baseURL, tenantID, conversationID)

	// Create request
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers
	if c.serviceKey != "" {
		req.Header.Set("X-Service-Key", c.serviceKey)
	}
	req.Header.Set("Authorization", "Bearer "+authToken)
	req.Header.Set("Accept", "application/json")

	// Execute request
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to call platform service: %w", err)
	}
	defer resp.Body.Close()

	// Read response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	// Check status code - forward specific error types
	if resp.StatusCode == http.StatusUnauthorized {
		return nil, fmt.Errorf("unauthorized: %s", string(body))
	}
	if resp.StatusCode == http.StatusForbidden {
		return nil, fmt.Errorf("forbidden: %s", string(body))
	}
	if resp.StatusCode == http.StatusNotFound {
		return nil, fmt.Errorf("not_found: conversation not found")
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("platform service returned status %d: %s", resp.StatusCode, string(body))
	}

	// Parse response
	var conversation ConversationResponse
	if err := json.Unmarshal(body, &conversation); err != nil {
		return nil, fmt.Errorf("failed to parse conversation response: %w", err)
	}

	return &conversation, nil
}

// ValidateConversation validates that a conversation exists and user has access.
func (c *client) ValidateConversation(ctx context.Context, tenantID, conversationID, authToken string) error {
	if c.baseURL == "" {
		// Skip validation if platform service is not configured
		return nil
	}

	if authToken == "" {
		return fmt.Errorf("auth token not provided")
	}

	// Build request URL
	url := fmt.Sprintf("%s/api/v1/platform-service/tenants/%s/conversations/%s", c.baseURL, tenantID, conversationID)

	// Create request
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers
	if c.serviceKey != "" {
		req.Header.Set("X-Service-Key", c.serviceKey)
	}
	req.Header.Set("Authorization", "Bearer "+authToken)
	req.Header.Set("Accept", "application/json")

	// Execute request
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to call platform service: %w", err)
	}
	defer resp.Body.Close()

	// Check status code - forward specific error types
	if resp.StatusCode == http.StatusUnauthorized {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("unauthorized: %s", string(body))
	}
	if resp.StatusCode == http.StatusForbidden {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("forbidden: %s", string(body))
	}
	if resp.StatusCode == http.StatusNotFound {
		return fmt.Errorf("not_found: conversation not found")
	}
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("platform service returned status %d: %s", resp.StatusCode, string(body))
	}

	return nil
}

// ValidateAutonomousAgent validates that an autonomous agent exists and user has access.
func (c *client) ValidateAutonomousAgent(ctx context.Context, tenantID, autonomousAgentID, authToken string) error {
	if c.baseURL == "" {
		// Skip validation if platform service is not configured
		return nil
	}

	if authToken == "" {
		return fmt.Errorf("auth token not provided")
	}

	// Build request URL
	url := fmt.Sprintf("%s/api/v1/platform-service/tenants/%s/autonomous-agents/%s", c.baseURL, tenantID, autonomousAgentID)

	// Create request
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers
	if c.serviceKey != "" {
		req.Header.Set("X-Service-Key", c.serviceKey)
	}
	req.Header.Set("Authorization", "Bearer "+authToken)
	req.Header.Set("Accept", "application/json")

	// Execute request
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to call platform service: %w", err)
	}
	defer resp.Body.Close()

	// Check status code - forward specific error types
	if resp.StatusCode == http.StatusUnauthorized {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("unauthorized: %s", string(body))
	}
	if resp.StatusCode == http.StatusForbidden {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("forbidden: %s", string(body))
	}
	if resp.StatusCode == http.StatusNotFound {
		return fmt.Errorf("not_found: autonomous agent not found")
	}
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("platform service returned status %d: %s", resp.StatusCode, string(body))
	}

	return nil
}

// GetAutonomousAgentConfig retrieves the autonomous agent configuration from the platform service.
// This uses X-Unified-UI-Autonomous-Agent-API-Key header for authentication (NOT Bearer token).
// apiKey is the autonomous agent's API key that will be validated against primary/secondary keys.
func (c *client) GetAutonomousAgentConfig(ctx context.Context, tenantID, autonomousAgentID, apiKey string) (*AutonomousAgentConfigResponse, error) {
	if c.baseURL == "" {
		return nil, fmt.Errorf("platform service URL not configured")
	}

	if apiKey == "" {
		return nil, fmt.Errorf("API key not provided")
	}

	// Build request URL - use /config endpoint
	url := fmt.Sprintf("%s/api/v1/platform-service/tenants/%s/autonomous-agents/%s/config", c.baseURL, tenantID, autonomousAgentID)

	// Create request
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers - API key authentication only (no Bearer token, no service key)
	req.Header.Set("X-Unified-UI-Autonomous-Agent-API-Key", apiKey)
	req.Header.Set("Accept", "application/json")

	// Execute request
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to call platform service: %w", err)
	}
	defer resp.Body.Close()

	// Read response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	// Check status code - forward specific error types
	if resp.StatusCode == http.StatusUnauthorized {
		return nil, fmt.Errorf("unauthorized: invalid API key")
	}
	if resp.StatusCode == http.StatusForbidden {
		return nil, fmt.Errorf("forbidden: %s", string(body))
	}
	if resp.StatusCode == http.StatusNotFound {
		return nil, fmt.Errorf("not_found: autonomous agent not found")
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("platform service returned status %d: %s", resp.StatusCode, string(body))
	}

	// Parse response
	var config AutonomousAgentConfigResponse
	if err := json.Unmarshal(body, &config); err != nil {
		return nil, fmt.Errorf("failed to parse autonomous agent config response: %w", err)
	}

	return &config, nil
}
