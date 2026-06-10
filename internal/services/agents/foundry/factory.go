// Package foundry provides Microsoft Foundry agent client implementations.
package foundry

import (
	"encoding/json"
	"fmt"

	"github.com/unifiedui/agent-service/internal/services/auth/clientcredentials"
	"github.com/unifiedui/agent-service/internal/services/platform"
)

// Factory creates Foundry-specific agent clients.
type Factory struct {
	tokenAcquirer AppRegTokenAcquirer
}

// NewFactory creates a new Foundry factory without app-registration support.
func NewFactory() *Factory {
	return &Factory{}
}

// NewFactoryWithTokenAcquirer creates a Foundry factory wired to a token
// acquirer for ENTRA_ID_APP_REGISTRATION authentication.
func NewFactoryWithTokenAcquirer(acquirer AppRegTokenAcquirer) *Factory {
	return &Factory{tokenAcquirer: acquirer}
}

// CreateWorkflowClient creates a Foundry workflow client from platform configuration.
//
// userToken is the caller's Entra ID bearer token; it is used for
// ENTRA_ID_USER_TOKEN auth and as the legacy fallback when no auth_type is set.
func (f *Factory) CreateWorkflowClient(config *platform.AgentConfig, userToken string) (*WorkflowClient, error) {
	if config == nil {
		return nil, fmt.Errorf("config is required")
	}
	authProvider, err := f.buildAuthProvider(config, userToken)
	if err != nil {
		return nil, err
	}
	return NewWorkflowClient(&WorkflowClientConfig{
		ProjectEndpoint: config.Settings.ProjectEndpoint,
		APIVersion:      config.Settings.APIVersion,
		AgentName:       config.Settings.AgentName,
		AgentType:       config.Settings.AgentType,
		AuthProvider:    authProvider,
	})
}

// buildAuthProvider returns an AuthProvider matching the agent's configured auth_type.
// When auth_type is empty, the legacy ENTRA_ID_USER_TOKEN behavior is used.
func (f *Factory) buildAuthProvider(config *platform.AgentConfig, userToken string) (AuthProvider, error) {
	authType := AuthType(config.Settings.AuthType)
	if authType == "" {
		authType = AuthTypeEntraIDUserToken
	}

	switch authType {
	case AuthTypeEntraIDUserToken:
		return NewUserTokenAuth(userToken)

	case AuthTypeAPIKey:
		secret, err := extractCredentialSecret(config.Settings.Credential)
		if err != nil {
			return nil, fmt.Errorf("foundry API_KEY auth: %w", err)
		}
		return NewAPIKeyAuth(secret)

	case AuthTypeEntraIDAppRegistration:
		creds, err := extractAppRegistrationCredentials(config.Settings.Credential)
		if err != nil {
			return nil, fmt.Errorf("foundry ENTRA_ID_APP_REGISTRATION auth: %w", err)
		}
		if f.tokenAcquirer == nil {
			return nil, fmt.Errorf("foundry: token acquirer is required for ENTRA_ID_APP_REGISTRATION auth")
		}
		return NewAppRegistrationAuth(creds, f.tokenAcquirer)

	default:
		return nil, fmt.Errorf("foundry: unsupported auth_type %q", authType)
	}
}

// extractCredentialSecret returns a string secret (api key) from a credential.
func extractCredentialSecret(cred *platform.Credentials) (string, error) {
	if cred == nil {
		return "", fmt.Errorf("credential is required")
	}
	switch v := cred.Secret.(type) {
	case string:
		return v, nil
	case map[string]interface{}:
		if s, ok := v["api_key"].(string); ok && s != "" {
			return s, nil
		}
		return "", fmt.Errorf("credential secret object missing api_key")
	default:
		return "", fmt.Errorf("credential secret has unsupported type %T", cred.Secret)
	}
}

// extractAppRegistrationCredentials parses the credential secret as a JSON object
// containing tenant_id, client_id and client_secret fields.
func extractAppRegistrationCredentials(cred *platform.Credentials) (clientcredentials.Credentials, error) {
	if cred == nil {
		return clientcredentials.Credentials{}, fmt.Errorf("credential is required")
	}

	var raw map[string]interface{}
	switch v := cred.Secret.(type) {
	case string:
		if err := json.Unmarshal([]byte(v), &raw); err != nil {
			return clientcredentials.Credentials{}, fmt.Errorf("parse credential secret: %w", err)
		}
	case map[string]interface{}:
		raw = v
	default:
		return clientcredentials.Credentials{}, fmt.Errorf("credential secret has unsupported type %T", cred.Secret)
	}

	tenantID, _ := raw["tenant_id"].(string)
	clientID, _ := raw["client_id"].(string)
	clientSecret, _ := raw["client_secret"].(string)

	if tenantID == "" || clientID == "" || clientSecret == "" {
		return clientcredentials.Credentials{}, fmt.Errorf("credential secret missing tenant_id, client_id or client_secret")
	}

	return clientcredentials.Credentials{
		TenantID:     tenantID,
		ClientID:     clientID,
		ClientSecret: clientSecret,
	}, nil
}

// NewFromConfig creates a Foundry workflow client from platform configuration.
//
// Deprecated: prefer Factory.CreateWorkflowClient, which honors
// config.Settings.AuthType. This shim preserves the legacy behavior where the
// caller-supplied bearer token is used unconditionally.
func NewFromConfig(config *platform.AgentConfig, apiToken string) (*WorkflowClient, error) {
	if config == nil {
		return nil, fmt.Errorf("config is required")
	}
	return NewWorkflowClient(&WorkflowClientConfig{
		ProjectEndpoint: config.Settings.ProjectEndpoint,
		APIVersion:      config.Settings.APIVersion,
		AgentName:       config.Settings.AgentName,
		AgentType:       config.Settings.AgentType,
		APIToken:        apiToken,
	})
}
