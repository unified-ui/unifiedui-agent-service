package clientcredentials

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// HTTPDoer is the minimal interface required from an HTTP client.
type HTTPDoer interface {
	Do(req *http.Request) (*http.Response, error)
}

// TokenClient acquires access tokens from Microsoft Entra ID using the
// client credentials grant. It is safe for concurrent use.
type TokenClient struct {
	httpClient HTTPDoer
	authority  string
}

// NewTokenClient creates a TokenClient pointing at the public AAD authority.
func NewTokenClient(httpClient HTTPDoer) *TokenClient {
	if httpClient == nil {
		httpClient = &http.Client{Timeout: 30 * time.Second}
	}
	return &TokenClient{
		httpClient: httpClient,
		authority:  "https://login.microsoftonline.com",
	}
}

// SetAuthority overrides the default authority (for testing).
func (c *TokenClient) SetAuthority(authority string) {
	c.authority = strings.TrimSuffix(authority, "/")
}

type tokenResponse struct {
	AccessToken string `json:"access_token"`
	ExpiresIn   int    `json:"expires_in"`
	Error       string `json:"error,omitempty"`
	ErrorDesc   string `json:"error_description,omitempty"`
}

// Acquire requests a new access token for the given scope.
func (c *TokenClient) Acquire(ctx context.Context, creds Credentials, scope string) (*Token, error) {
	if creds.TenantID == "" || creds.ClientID == "" || creds.ClientSecret == "" {
		return nil, ErrInvalidCredentials
	}
	if scope == "" {
		return nil, fmt.Errorf("clientcredentials: scope is required")
	}

	endpoint := fmt.Sprintf("%s/%s/oauth2/v2.0/token", c.authority, creds.TenantID)
	form := url.Values{}
	form.Set("grant_type", "client_credentials")
	form.Set("client_id", creds.ClientID)
	form.Set("client_secret", creds.ClientSecret)
	form.Set("scope", scope)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, strings.NewReader(form.Encode()))
	if err != nil {
		return nil, fmt.Errorf("clientcredentials: build request: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("clientcredentials: request failed: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	body, err := io.ReadAll(io.LimitReader(resp.Body, 64*1024))
	if err != nil {
		return nil, fmt.Errorf("clientcredentials: read response: %w", err)
	}

	var tr tokenResponse
	if jerr := json.Unmarshal(body, &tr); jerr != nil {
		return nil, fmt.Errorf("clientcredentials: parse response (status=%d): %w", resp.StatusCode, jerr)
	}

	if resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden {
		return nil, fmt.Errorf("%w: %s: %s", ErrInvalidCredentials, tr.Error, tr.ErrorDesc)
	}
	if resp.StatusCode >= 400 || tr.AccessToken == "" {
		return nil, fmt.Errorf("clientcredentials: token request failed (status=%d): %s: %s", resp.StatusCode, tr.Error, tr.ErrorDesc)
	}

	return &Token{
		AccessToken: tr.AccessToken,
		ExpiresAt:   time.Now().Add(time.Duration(tr.ExpiresIn) * time.Second),
	}, nil
}
