// Package connections provides connection testing functionality for external services.
package connections

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"

	"github.com/unifiedui/agent-service/internal/services/agents/foundry"
	"github.com/unifiedui/agent-service/internal/services/agents/n8n"
	"github.com/unifiedui/agent-service/internal/services/agents/restapi"
	"github.com/unifiedui/agent-service/internal/services/platform"
)

var validIdentifier = regexp.MustCompile(`^[A-Za-z0-9_\-.:]{1,512}$`)

// TestResult holds the result of a connection test.
type TestResult struct {
	Success        bool
	Message        string
	ResponseTimeMs int64
}

// Service defines the interface for connection testing.
type Service interface {
	TestConnection(ctx context.Context, testType, rawURL string, config map[string]interface{}, credential *platform.Credentials, userToken string) (*TestResult, error)
}

type service struct{}

// NewService creates a new connection testing service.
func NewService() Service {
	return &service{}
}

// TestConnection dispatches to the appropriate test method based on test type.
func (s *service) TestConnection(ctx context.Context, testType, rawURL string, config map[string]interface{}, credential *platform.Credentials, userToken string) (*TestResult, error) {
	validatedURL, validationErr := validateHTTPURL(rawURL)
	if validationErr != nil {
		return &TestResult{Success: false, Message: fmt.Sprintf("Invalid URL: %s", validationErr.Error())}, nil
	}

	start := time.Now()
	result, testErr := s.dispatchTest(ctx, testType, validatedURL, config, credential, userToken)

	elapsed := time.Since(start).Milliseconds()
	if testErr != nil {
		return &TestResult{Success: false, Message: testErr.Error(), ResponseTimeMs: elapsed}, nil //nolint:nilerr // intentional: convert error to user-facing result message
	}
	if result != nil {
		result.ResponseTimeMs = elapsed
	}
	return result, nil
}

func (s *service) dispatchTest(ctx context.Context, testType, validatedURL string, config map[string]interface{}, credential *platform.Credentials, userToken string) (*TestResult, error) {
	switch testType {
	case "N8N_CHAT_URL":
		return s.testN8NChatURL(ctx, validatedURL, credential)
	case "N8N_WORKFLOW":
		return s.testN8NWorkflow(ctx, validatedURL, config, credential)
	case "N8N_WEBHOOK":
		return s.testN8NWebhook(ctx, validatedURL, credential)
	case "FOUNDRY_AGENT":
		return s.testFoundryAgent(ctx, validatedURL, config, userToken)
	case "REST_API_INVOKE":
		return s.testRestAPIInvoke(ctx, validatedURL, config, credential, userToken)
	case "REST_API_CONVERSATION":
		return s.testRestAPIConversation(ctx, validatedURL, config, credential, userToken)
	default:
		return nil, fmt.Errorf("unsupported test type: %s", testType)
	}
}

func (s *service) testN8NChatURL(ctx context.Context, chatURL string, credential *platform.Credentials) (*TestResult, error) {
	testCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	var username, password string
	if credential != nil {
		secret := credential.GetSecretAsBasicAuth()
		if secret != nil {
			username = secret.Username
			password = secret.Password
		}
	}

	body, err := json.Marshal(&n8n.ChatRequest{
		ChatInput: "ping",
		SessionID: "unified-ui-connection-test",
	})
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(testCtx, http.MethodPost, chatURL, bytes.NewBuffer(body))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "text/event-stream")
	if username != "" && password != "" {
		req.SetBasicAuth(username, password)
	}

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("connection failed: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return &TestResult{
			Success: false,
			Message: fmt.Sprintf("n8n chat URL returned status %d", resp.StatusCode),
		}, nil
	}

	scanner := bufio.NewScanner(resp.Body)
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "data:") || strings.HasPrefix(line, "{") {
			return &TestResult{
				Success: true,
				Message: "n8n chat URL is reachable and responding (test execution triggered)",
			}, nil
		}
	}

	return &TestResult{
		Success: true,
		Message: "n8n chat URL is reachable (200 OK)",
	}, nil
}

func (s *service) testN8NWorkflow(ctx context.Context, workflowEndpoint string, _ map[string]interface{}, credential *platform.Credentials) (*TestResult, error) {
	testCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	workflowID := extractN8NWorkflowID(workflowEndpoint)
	if workflowID == "" {
		return &TestResult{Success: false, Message: "Could not extract workflow ID from endpoint URL"}, nil
	}

	if !validIdentifier.MatchString(workflowID) {
		return &TestResult{Success: false, Message: "Invalid workflow ID format"}, nil
	}

	baseURL := extractN8NBaseURL(workflowEndpoint)
	if baseURL == "" {
		return &TestResult{Success: false, Message: "Could not extract base URL from workflow endpoint"}, nil
	}

	var apiKey string
	if credential != nil {
		apiKey = credential.GetSecretAsString()
	}

	apiClient, err := n8n.NewAPIClient(&n8n.APIClientConfig{
		BaseURL:    baseURL,
		APIKey:     apiKey,
		HTTPClient: &http.Client{Timeout: 10 * time.Second},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create n8n API client: %w", err)
	}

	workflow, err := apiClient.GetWorkflow(testCtx, workflowID)
	if err != nil {
		return &TestResult{
			Success: false,
			Message: fmt.Sprintf("Failed to reach n8n workflow: %s", err.Error()),
		}, nil
	}

	activeStatus := "inactive"
	if workflow.Active {
		activeStatus = "active"
	}

	return &TestResult{
		Success: true,
		Message: fmt.Sprintf("Workflow '%s' found (%s)", workflow.Name, activeStatus),
	}, nil
}

func (s *service) testN8NWebhook(ctx context.Context, webhookURL string, credential *platform.Credentials) (*TestResult, error) {
	testCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	body, err := json.Marshal(map[string]interface{}{"test": true})
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(testCtx, http.MethodPost, webhookURL, bytes.NewBuffer(body))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	if credential != nil {
		apiKey := credential.GetSecretAsString()
		if apiKey != "" {
			req.Header.Set("X-N8N-API-KEY", apiKey)
		}
	}

	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("webhook unreachable: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode >= http.StatusOK && resp.StatusCode < http.StatusMultipleChoices {
		return &TestResult{
			Success: true,
			Message: fmt.Sprintf("Webhook is reachable (status %d, test execution triggered)", resp.StatusCode),
		}, nil
	}

	return &TestResult{
		Success: false,
		Message: fmt.Sprintf("Webhook returned status %d", resp.StatusCode),
	}, nil
}

func (s *service) testFoundryAgent(ctx context.Context, projectEndpoint string, config map[string]interface{}, userToken string) (*TestResult, error) {
	testCtx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()

	agentName, _ := config["agent_name"].(string)
	apiVersion, _ := config["api_version"].(string)

	if agentName == "" {
		return &TestResult{Success: false, Message: "Agent name is required for Foundry test"}, nil
	}
	if userToken == "" {
		return &TestResult{Success: false, Message: "Authentication token is required for Foundry test"}, nil
	}

	foundryClient, err := foundry.NewWorkflowClient(&foundry.WorkflowClientConfig{
		ProjectEndpoint: projectEndpoint,
		APIVersion:      apiVersion,
		AgentName:       agentName,
		AgentType:       "AGENT",
		APIToken:        userToken,
	})
	if err != nil {
		return &TestResult{
			Success: false,
			Message: fmt.Sprintf("Failed to create Foundry client: %s", err.Error()),
		}, nil
	}
	foundryClient.SetHTTPClient(&http.Client{Timeout: 15 * time.Second})

	if err := foundryClient.TestConnection(testCtx); err != nil {
		return &TestResult{
			Success: false,
			Message: fmt.Sprintf("Foundry agent unreachable: %s", err.Error()),
		}, nil
	}

	return &TestResult{
		Success: true,
		Message: fmt.Sprintf("Foundry agent '%s' is reachable and responding", agentName),
	}, nil
}

func (s *service) testRestAPIInvoke(ctx context.Context, invokeEndpoint string, config map[string]interface{}, credential *platform.Credentials, userToken string) (*TestResult, error) {
	testCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	authType := restapi.AuthType(getStringFromConfig(config, "auth_type"))
	apiKeyHeaderName := getStringFromConfig(config, "api_key_header_name")

	accessToken, err := s.resolveAccessToken(testCtx, authType, credential, userToken)
	if err != nil {
		return &TestResult{
			Success: false,
			Message: fmt.Sprintf("Failed to acquire access token: %s", err.Error()),
		}, nil
	}

	restapiClient, err := restapi.NewWorkflowClient(&restapi.WorkflowClientConfig{
		InvokeEndpoint:   invokeEndpoint,
		AuthType:         authType,
		Credential:       credential,
		AccessToken:      accessToken,
		UserToken:        userToken,
		APIKeyHeaderName: apiKeyHeaderName,
		HTTPClient:       &http.Client{Timeout: 10 * time.Second},
	})
	if err != nil {
		return &TestResult{
			Success: false,
			Message: fmt.Sprintf("Failed to create REST API client: %s", err.Error()),
		}, nil
	}

	reader, err := restapiClient.InvokeStreamReader(testCtx, "", "connection-test", "ping", nil)
	if err != nil {
		return &TestResult{
			Success: false,
			Message: fmt.Sprintf("REST API invoke endpoint unreachable: %s", err.Error()),
		}, nil
	}
	defer func() { _ = reader.Close() }()

	chunk, readErr := reader.Read()
	if readErr != nil && !errors.Is(readErr, io.EOF) {
		return &TestResult{
			Success: false,
			Message: fmt.Sprintf("Failed to read REST API response: %s", readErr.Error()),
		}, nil
	}

	if chunk != nil || errors.Is(readErr, io.EOF) {
		return &TestResult{
			Success: true,
			Message: "REST API invoke endpoint is reachable and responding",
		}, nil
	}

	return &TestResult{
		Success: true,
		Message: "REST API invoke endpoint connection established",
	}, nil
}

func (s *service) testRestAPIConversation(ctx context.Context, conversationEndpoint string, config map[string]interface{}, credential *platform.Credentials, userToken string) (*TestResult, error) {
	testCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	authType := restapi.AuthType(getStringFromConfig(config, "auth_type"))
	apiKeyHeaderName := getStringFromConfig(config, "api_key_header_name")

	accessToken, err := s.resolveAccessToken(testCtx, authType, credential, userToken)
	if err != nil {
		return &TestResult{
			Success: false,
			Message: fmt.Sprintf("Failed to acquire access token: %s", err.Error()),
		}, nil
	}

	restapiClient, err := restapi.NewWorkflowClient(&restapi.WorkflowClientConfig{
		InvokeEndpoint:             "https://placeholder.invalid",
		CreateConversationEndpoint: conversationEndpoint,
		AuthType:                   authType,
		Credential:                 credential,
		AccessToken:                accessToken,
		UserToken:                  userToken,
		APIKeyHeaderName:           apiKeyHeaderName,
		HTTPClient:                 &http.Client{Timeout: 10 * time.Second},
	})
	if err != nil {
		return &TestResult{
			Success: false,
			Message: fmt.Sprintf("Failed to create REST API client: %s", err.Error()),
		}, nil
	}

	conversationID, err := restapiClient.CreateConversation(testCtx)
	if err != nil {
		return &TestResult{
			Success: false,
			Message: fmt.Sprintf("REST API conversation endpoint unreachable: %s", err.Error()),
		}, nil
	}

	return &TestResult{
		Success: true,
		Message: fmt.Sprintf("Conversation created successfully (ID: %s)", conversationID),
	}, nil
}

func validateHTTPURL(rawURL string) (string, error) {
	parsed, err := url.Parse(rawURL)
	if err != nil {
		return "", fmt.Errorf("malformed URL: %w", err)
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return "", fmt.Errorf("URL scheme must be http or https, got: %s", parsed.Scheme)
	}
	if parsed.Host == "" {
		return "", fmt.Errorf("URL must have a host")
	}
	return rawURL, nil
}

func extractN8NWorkflowID(workflowEndpoint string) string {
	parts := strings.Split(workflowEndpoint, "/workflow/")
	if len(parts) < 2 {
		parts = strings.Split(workflowEndpoint, "/workflows/")
		if len(parts) < 2 {
			return ""
		}
	}
	idPart := parts[len(parts)-1]
	idPart = strings.TrimRight(idPart, "/")
	if idx := strings.Index(idPart, "/"); idx >= 0 {
		idPart = idPart[:idx]
	}
	if idx := strings.Index(idPart, "?"); idx >= 0 {
		idPart = idPart[:idx]
	}
	return idPart
}

func extractN8NBaseURL(workflowEndpoint string) string {
	for _, sep := range []string{"/workflow/", "/workflows/"} {
		if idx := strings.Index(workflowEndpoint, sep); idx >= 0 {
			return strings.TrimRight(workflowEndpoint[:idx], "/")
		}
	}
	return ""
}

func getStringFromConfig(config map[string]interface{}, key string) string {
	if config == nil {
		return ""
	}
	val, _ := config[key].(string)
	return val
}

func (s *service) resolveAccessToken(ctx context.Context, authType restapi.AuthType, credential *platform.Credentials, userToken string) (string, error) {
	switch authType {
	case restapi.AuthTypeEntraIDUserToken:
		return userToken, nil
	case restapi.AuthTypeEntraIDAppRegistration:
		if credential == nil {
			return "", fmt.Errorf("credential required for Entra ID App Registration auth")
		}
		appReg := credential.GetSecretAsEntraIDAppReg()
		if appReg == nil {
			return "", fmt.Errorf("invalid credential format for Entra ID App Registration")
		}
		return acquireClientCredentialsToken(ctx, appReg)
	default:
		return "", nil
	}
}

func acquireClientCredentialsToken(ctx context.Context, appReg *platform.EntraIDAppRegistrationSecret) (string, error) {
	tokenURL := fmt.Sprintf("https://login.microsoftonline.com/%s/oauth2/v2.0/token", url.PathEscape(appReg.TenantID))

	body := url.Values{
		"grant_type":    {"client_credentials"},
		"client_id":     {appReg.ClientID},
		"client_secret": {appReg.ClientSecret},
		"scope":         {"https://graph.microsoft.com/.default"},
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, tokenURL, strings.NewReader(body.Encode()))
	if err != nil {
		return "", fmt.Errorf("failed to create token request: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := (&http.Client{Timeout: 10 * time.Second}).Do(req)
	if err != nil {
		return "", fmt.Errorf("token request failed: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read token response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("token endpoint returned %d: %s", resp.StatusCode, string(respBody))
	}

	var tokenResp struct {
		AccessToken string `json:"access_token"`
	}
	if err := json.Unmarshal(respBody, &tokenResp); err != nil {
		return "", fmt.Errorf("failed to parse token response: %w", err)
	}

	if tokenResp.AccessToken == "" {
		return "", fmt.Errorf("empty access_token in token response")
	}

	return tokenResp.AccessToken, nil
}
