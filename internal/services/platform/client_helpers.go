package platform

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

type requestConfig struct {
	method  string
	url     string
	body    io.Reader
	headers map[string]string
}

func (c *client) doRawRequest(ctx context.Context, cfg requestConfig) (body []byte, statusCode int, err error) {
	req, err := http.NewRequestWithContext(ctx, cfg.method, cfg.url, cfg.body)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Accept", "application/json")
	for k, v := range cfg.headers {
		req.Header.Set(k, v)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to call platform service: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	body, err = io.ReadAll(resp.Body)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to read response body: %w", err)
	}

	return body, resp.StatusCode, nil
}

func checkStatus(statusCode int, body []byte, notFoundMsg string) error {
	if statusCode == http.StatusOK {
		return nil
	}
	switch statusCode {
	case http.StatusUnauthorized:
		return fmt.Errorf("unauthorized: %s", string(body))
	case http.StatusForbidden:
		return fmt.Errorf("forbidden: %s", string(body))
	case http.StatusNotFound:
		return fmt.Errorf("not_found: %s", notFoundMsg)
	default:
		return fmt.Errorf("platform service returned status %d: %s", statusCode, string(body))
	}
}

func doJSONRequest[T any](ctx context.Context, c *client, cfg requestConfig, notFoundMsg string) (*T, error) {
	body, statusCode, err := c.doRawRequest(ctx, cfg)
	if err != nil {
		return nil, err
	}
	if err := checkStatus(statusCode, body, notFoundMsg); err != nil {
		return nil, err
	}
	var result T
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}
	return &result, nil
}

func doJSONSliceRequest[T any](ctx context.Context, c *client, cfg requestConfig, notFoundMsg string) ([]T, error) {
	body, statusCode, err := c.doRawRequest(ctx, cfg)
	if err != nil {
		return nil, err
	}
	if err := checkStatus(statusCode, body, notFoundMsg); err != nil {
		return nil, err
	}
	var result []T
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}
	return result, nil
}

func doValidateRequest(ctx context.Context, c *client, cfg requestConfig, notFoundMsg string) error {
	body, statusCode, err := c.doRawRequest(ctx, cfg)
	if err != nil {
		return err
	}
	return checkStatus(statusCode, body, notFoundMsg)
}

func bearerHeaders(c *client, authToken string) map[string]string {
	h := make(map[string]string, 2)
	if c.serviceKey != "" {
		h["X-Service-Key"] = c.serviceKey
	}
	h["Authorization"] = "Bearer " + authToken
	return h
}

func serviceKeyHeaders(c *client) map[string]string {
	return map[string]string{
		"X-Service-Key": c.serviceKey,
	}
}

func apiKeyHeaders(apiKey string) map[string]string {
	return map[string]string{
		"X-Unified-UI-Workflow-API-Key": apiKey,
	}
}
