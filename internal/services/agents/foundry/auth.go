package foundry

import (
	"context"
	"errors"
	"fmt"
	"net/http"

	"github.com/unifiedui/agent-service/internal/services/auth/clientcredentials"
)

// AuthType represents the authentication mode for Foundry agents.
type AuthType string

// AuthType constants supported by the Foundry workflow client.
const (
	AuthTypeEntraIDUserToken       AuthType = "ENTRA_ID_USER_TOKEN" //nolint:gosec // not a credential
	AuthTypeEntraIDAppRegistration AuthType = "ENTRA_ID_APP_REGISTRATION"
	AuthTypeAPIKey                 AuthType = "API_KEY"
	AuthTypeCustomRestAPI          AuthType = "CUSTOM_REST_API" //nolint:gosec // not a credential
)

// FoundryTokenScope is the AAD scope used to acquire Foundry access tokens.
const FoundryTokenScope = "https://ai.azure.com/.default" // #nosec G101 -- AAD scope, not a credential

// AuthProvider applies the authentication header(s) to a Foundry HTTP request.
type AuthProvider interface {
	Apply(ctx context.Context, req *http.Request) error
	Invalidate(ctx context.Context)
}

// userTokenAuth attaches a pre-supplied bearer token (the caller's user token).
type userTokenAuth struct {
	token string
}

// NewUserTokenAuth returns an AuthProvider for ENTRA_ID_USER_TOKEN mode.
func NewUserTokenAuth(token string) (AuthProvider, error) {
	if token == "" {
		return nil, errors.New("foundry: user token is required for ENTRA_ID_USER_TOKEN auth")
	}
	return &userTokenAuth{token: token}, nil
}

func (a *userTokenAuth) Apply(_ context.Context, req *http.Request) error {
	req.Header.Set("Authorization", "Bearer "+a.token)
	return nil
}

func (a *userTokenAuth) Invalidate(_ context.Context) {}

// apiKeyAuth attaches the static api-key header.
type apiKeyAuth struct {
	apiKey string
}

// NewAPIKeyAuth returns an AuthProvider for API_KEY mode.
func NewAPIKeyAuth(apiKey string) (AuthProvider, error) {
	if apiKey == "" {
		return nil, errors.New("foundry: api key is required for API_KEY auth")
	}
	return &apiKeyAuth{apiKey: apiKey}, nil
}

func (a *apiKeyAuth) Apply(_ context.Context, req *http.Request) error {
	req.Header.Set("api-key", a.apiKey)
	return nil
}

func (a *apiKeyAuth) Invalidate(_ context.Context) {}

// AppRegTokenAcquirer is the subset of CachedTokenClient used by appRegAuth.
type AppRegTokenAcquirer interface {
	Acquire(ctx context.Context, creds clientcredentials.Credentials, scope string) (*clientcredentials.Token, error)
	Invalidate(ctx context.Context, creds clientcredentials.Credentials, scope string)
}

// appRegAuth acquires an AAD token via the client-credentials grant.
type appRegAuth struct {
	creds    clientcredentials.Credentials
	scope    string
	acquirer AppRegTokenAcquirer
}

// NewAppRegistrationAuth returns an AuthProvider for ENTRA_ID_APP_REGISTRATION mode.
func NewAppRegistrationAuth(creds clientcredentials.Credentials, acquirer AppRegTokenAcquirer) (AuthProvider, error) {
	if acquirer == nil {
		return nil, errors.New("foundry: token acquirer is required for ENTRA_ID_APP_REGISTRATION auth")
	}
	if creds.TenantID == "" || creds.ClientID == "" || creds.ClientSecret == "" {
		return nil, errors.New("foundry: tenant_id, client_id and client_secret are required for ENTRA_ID_APP_REGISTRATION auth")
	}
	return &appRegAuth{creds: creds, scope: FoundryTokenScope, acquirer: acquirer}, nil
}

func (a *appRegAuth) Apply(ctx context.Context, req *http.Request) error {
	tok, err := a.acquirer.Acquire(ctx, a.creds, a.scope)
	if err != nil {
		return fmt.Errorf("foundry: acquire app-reg token: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+tok.AccessToken)
	return nil
}

func (a *appRegAuth) Invalidate(ctx context.Context) {
	a.acquirer.Invalidate(ctx, a.creds, a.scope)
}
