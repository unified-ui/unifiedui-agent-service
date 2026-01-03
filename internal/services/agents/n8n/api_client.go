// Package n8n provides N8N-specific agent client implementations.
package n8n

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

// APIClientConfig holds the configuration for the N8N API client.
type APIClientConfig struct {
	BaseURL    string
	APIKey     string
	HTTPClient *http.Client
}

// ExecutionInfo represents information about an N8N execution.
type ExecutionInfo struct {
	ID        string                 `json:"id"`
	Status    string                 `json:"status"`
	StartedAt string                 `json:"startedAt"`
	StoppedAt string                 `json:"stoppedAt,omitempty"`
	Data      map[string]interface{} `json:"data,omitempty"`
}

// APIClient implements the API client for N8N.
type APIClient struct {
	baseURL    string
	apiKey     string
	httpClient *http.Client
}

// NewAPIClient creates a new N8N API client.
func NewAPIClient(config *APIClientConfig) (*APIClient, error) {
	if config == nil {
		return nil, fmt.Errorf("config is required")
	}

	httpClient := config.HTTPClient
	if httpClient == nil {
		httpClient = &http.Client{
			Timeout: 30 * time.Second,
		}
	}

	return &APIClient{
		baseURL:    config.BaseURL,
		apiKey:     config.APIKey,
		httpClient: httpClient,
	}, nil
}

// GetExecution retrieves execution details by ID.
func (c *APIClient) GetExecution(ctx context.Context, executionID string) (*ExecutionInfo, error) {
	url := fmt.Sprintf("%s/api/v1/executions/%s", c.baseURL, executionID)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	c.setHeaders(req)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to execute request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	var execResp ExecutionResponse
	if err := json.NewDecoder(resp.Body).Decode(&execResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &ExecutionInfo{
		ID:        execResp.ID,
		Status:    execResp.Status,
		StartedAt: execResp.StartedAt,
		StoppedAt: execResp.StoppedAt,
		Data:      execResp.Data,
	}, nil
}

// GetExecutionsBySession retrieves executions for a session.
func (c *APIClient) GetExecutionsBySession(ctx context.Context, sessionID string) ([]*ExecutionInfo, error) {
	// N8N doesn't have a direct session-to-execution mapping
	// This would require filtering executions by metadata
	return []*ExecutionInfo{}, nil
}

// Close releases any resources held by the client.
func (c *APIClient) Close() error {
	return nil
}

// setHeaders sets the required headers for N8N API requests.
func (c *APIClient) setHeaders(req *http.Request) {
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	if c.apiKey != "" {
		req.Header.Set("X-N8N-API-KEY", c.apiKey)
	}
}
