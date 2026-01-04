// Package platform provides the platform service client for configuration retrieval.
package platform

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// Client defines the interface for the platform service client.
type Client interface {
	// GetAgentConfig retrieves the agent configuration for a given application.
	// In production, this will call the Platform Service API.
	// Currently, it reads from a local config.json file.
	GetAgentConfig(ctx context.Context, tenantID, applicationID string) (*AgentConfig, error)
}

// client implements the Client interface.
type client struct {
	configPath string
	baseURL    string
}

// ClientConfig holds the configuration for the platform client.
type ClientConfig struct {
	// BaseURL is the URL of the Platform Service (for future use)
	BaseURL string
	// ConfigPath is the path to the local config.json file (for development)
	ConfigPath string
}

// NewClient creates a new platform service client.
func NewClient(cfg *ClientConfig) Client {
	return &client{
		configPath: cfg.ConfigPath,
		baseURL:    cfg.BaseURL,
	}
}

// GetAgentConfig retrieves the agent configuration.
// Currently reads from local config.json file.
// TODO: Implement actual Platform Service API call.
func (c *client) GetAgentConfig(ctx context.Context, tenantID, applicationID string) (*AgentConfig, error) {
	// For now, read from local config file
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
