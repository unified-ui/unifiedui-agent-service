package telemetry

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// HTTPSink is a Sink implementation that POSTs batches to the platform-service
// internal metrics endpoint.
type HTTPSink struct {
	baseURL    string
	serviceKey string
	httpClient *http.Client
}

// NewHTTPSink constructs an HTTPSink. baseURL should be the platform-service
// root (without the /api/v1/... suffix); serviceKey is the value of the
// X-Service-Key header.
func NewHTTPSink(baseURL, serviceKey string, httpClient *http.Client) *HTTPSink {
	if httpClient == nil {
		httpClient = &http.Client{Timeout: 10 * time.Second}
	}
	return &HTTPSink{
		baseURL:    strings.TrimRight(baseURL, "/"),
		serviceKey: serviceKey,
		httpClient: httpClient,
	}
}

// Send POSTs a batch to /api/v1/platform-service/internal/metrics/messages:batch.
func (s *HTTPSink) Send(ctx context.Context, batch []MetricEvent) error {
	if len(batch) == 0 {
		return nil
	}
	if s.baseURL == "" {
		return fmt.Errorf("telemetry: HTTPSink.baseURL is empty")
	}

	url := s.baseURL + "/api/v1/platform-service/internal/metrics/messages:batch"
	body, err := json.Marshal(map[string]any{"items": batch})
	if err != nil {
		return fmt.Errorf("telemetry: marshal batch: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("telemetry: build request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Service-Key", s.serviceKey)

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("telemetry: send batch: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		return nil
	}

	preview, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
	return fmt.Errorf("telemetry: ingest failed (status=%d): %s", resp.StatusCode, string(preview))
}
