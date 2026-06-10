package foundry_test

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/unifiedui/agent-service/internal/services/agents/foundry"
	"github.com/unifiedui/agent-service/internal/services/auth/clientcredentials"
	"github.com/unifiedui/agent-service/internal/services/platform"
)

type stubAcquirer struct {
	tok      *clientcredentials.Token
	err      error
	calls    int
	invalids int
}

func (s *stubAcquirer) Acquire(_ context.Context, _ clientcredentials.Credentials, _ string) (*clientcredentials.Token, error) {
	s.calls++
	if s.err != nil {
		return nil, s.err
	}
	return s.tok, nil
}

func (s *stubAcquirer) Invalidate(_ context.Context, _ clientcredentials.Credentials, _ string) {
	s.invalids++
}

func TestNewUserTokenAuth_AppliesBearer(t *testing.T) {
	prov, err := foundry.NewUserTokenAuth("user-tok")
	require.NoError(t, err)
	req, _ := http.NewRequestWithContext(context.Background(), http.MethodGet, "https://x", nil)
	require.NoError(t, prov.Apply(context.Background(), req))
	assert.Equal(t, "Bearer user-tok", req.Header.Get("Authorization"))
}

func TestNewUserTokenAuth_RequiresToken(t *testing.T) {
	_, err := foundry.NewUserTokenAuth("")
	require.Error(t, err)
}

func TestNewAPIKeyAuth_AppliesHeader(t *testing.T) {
	prov, err := foundry.NewAPIKeyAuth("k-123")
	require.NoError(t, err)
	req, _ := http.NewRequestWithContext(context.Background(), http.MethodGet, "https://x", nil)
	require.NoError(t, prov.Apply(context.Background(), req))
	assert.Equal(t, "k-123", req.Header.Get("api-key"))
	assert.Empty(t, req.Header.Get("Authorization"))
}

func TestNewAPIKeyAuth_RequiresKey(t *testing.T) {
	_, err := foundry.NewAPIKeyAuth("")
	require.Error(t, err)
}

func TestNewAppRegistrationAuth_AppliesAcquiredToken(t *testing.T) {
	stub := &stubAcquirer{tok: &clientcredentials.Token{AccessToken: "AAD-TOK", ExpiresAt: time.Now().Add(time.Hour)}}
	prov, err := foundry.NewAppRegistrationAuth(clientcredentials.Credentials{TenantID: "t", ClientID: "c", ClientSecret: "s"}, stub)
	require.NoError(t, err)

	req, _ := http.NewRequestWithContext(context.Background(), http.MethodGet, "https://x", nil)
	require.NoError(t, prov.Apply(context.Background(), req))
	assert.Equal(t, "Bearer AAD-TOK", req.Header.Get("Authorization"))
	assert.Equal(t, 1, stub.calls)

	prov.Invalidate(context.Background())
	assert.Equal(t, 1, stub.invalids)
}

func TestNewAppRegistrationAuth_RejectsMissingFields(t *testing.T) {
	_, err := foundry.NewAppRegistrationAuth(clientcredentials.Credentials{}, &stubAcquirer{})
	require.Error(t, err)
	_, err = foundry.NewAppRegistrationAuth(clientcredentials.Credentials{TenantID: "t", ClientID: "c", ClientSecret: "s"}, nil)
	require.Error(t, err)
}

func TestNewAppRegistrationAuth_PropagatesAcquireError(t *testing.T) {
	stub := &stubAcquirer{err: errors.New("aad down")}
	prov, err := foundry.NewAppRegistrationAuth(clientcredentials.Credentials{TenantID: "t", ClientID: "c", ClientSecret: "s"}, stub)
	require.NoError(t, err)
	req, _ := http.NewRequestWithContext(context.Background(), http.MethodGet, "https://x", nil)
	err = prov.Apply(context.Background(), req)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "aad down")
}

func TestFactory_CreateWorkflowClient_UserToken(t *testing.T) {
	cfg := &platform.AgentConfig{Settings: platform.AgentSettings{
		ProjectEndpoint: "https://x", AgentName: "a", AgentType: "AGENT", AuthType: "ENTRA_ID_USER_TOKEN",
	}}
	cli, err := foundry.NewFactory().CreateWorkflowClient(cfg, "u-tok")
	require.NoError(t, err)
	assert.NotNil(t, cli)
}

func TestFactory_CreateWorkflowClient_DefaultsToUserToken(t *testing.T) {
	cfg := &platform.AgentConfig{Settings: platform.AgentSettings{
		ProjectEndpoint: "https://x", AgentName: "a", AgentType: "AGENT",
	}}
	cli, err := foundry.NewFactory().CreateWorkflowClient(cfg, "u-tok")
	require.NoError(t, err)
	assert.NotNil(t, cli)
}

func TestFactory_CreateWorkflowClient_APIKey(t *testing.T) {
	cfg := &platform.AgentConfig{Settings: platform.AgentSettings{
		ProjectEndpoint: "https://x", AgentName: "a", AgentType: "AGENT", AuthType: "API_KEY",
		Credential: &platform.Credentials{Secret: "k-1"},
	}}
	cli, err := foundry.NewFactory().CreateWorkflowClient(cfg, "")
	require.NoError(t, err)
	assert.NotNil(t, cli)
}

func TestFactory_CreateWorkflowClient_APIKey_FromObject(t *testing.T) {
	cfg := &platform.AgentConfig{Settings: platform.AgentSettings{
		ProjectEndpoint: "https://x", AgentName: "a", AgentType: "AGENT", AuthType: "API_KEY",
		Credential: &platform.Credentials{Secret: map[string]interface{}{"api_key": "k-2"}},
	}}
	_, err := foundry.NewFactory().CreateWorkflowClient(cfg, "")
	require.NoError(t, err)
}

func TestFactory_CreateWorkflowClient_APIKey_MissingCredential(t *testing.T) {
	cfg := &platform.AgentConfig{Settings: platform.AgentSettings{
		ProjectEndpoint: "https://x", AgentName: "a", AgentType: "AGENT", AuthType: "API_KEY",
	}}
	_, err := foundry.NewFactory().CreateWorkflowClient(cfg, "")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "API_KEY")
}

func TestFactory_CreateWorkflowClient_AppRegistration(t *testing.T) {
	stub := &stubAcquirer{tok: &clientcredentials.Token{AccessToken: "tok", ExpiresAt: time.Now().Add(time.Hour)}}
	cfg := &platform.AgentConfig{Settings: platform.AgentSettings{
		ProjectEndpoint: "https://x", AgentName: "a", AgentType: "AGENT", AuthType: "ENTRA_ID_APP_REGISTRATION",
		Credential: &platform.Credentials{Secret: `{"tenant_id":"t","client_id":"c","client_secret":"s"}`},
	}}
	cli, err := foundry.NewFactoryWithTokenAcquirer(stub).CreateWorkflowClient(cfg, "")
	require.NoError(t, err)
	assert.NotNil(t, cli)
}

func TestFactory_CreateWorkflowClient_AppRegistration_NoAcquirer(t *testing.T) {
	cfg := &platform.AgentConfig{Settings: platform.AgentSettings{
		ProjectEndpoint: "https://x", AgentName: "a", AgentType: "AGENT", AuthType: "ENTRA_ID_APP_REGISTRATION",
		Credential: &platform.Credentials{Secret: `{"tenant_id":"t","client_id":"c","client_secret":"s"}`},
	}}
	_, err := foundry.NewFactory().CreateWorkflowClient(cfg, "")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "token acquirer")
}

func TestFactory_CreateWorkflowClient_AppRegistration_BadSecret(t *testing.T) {
	stub := &stubAcquirer{}
	cfg := &platform.AgentConfig{Settings: platform.AgentSettings{
		ProjectEndpoint: "https://x", AgentName: "a", AgentType: "AGENT", AuthType: "ENTRA_ID_APP_REGISTRATION",
		Credential: &platform.Credentials{Secret: `{"tenant_id":"t"}`},
	}}
	_, err := foundry.NewFactoryWithTokenAcquirer(stub).CreateWorkflowClient(cfg, "")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "missing")
}

func TestFactory_CreateWorkflowClient_UnsupportedAuthType(t *testing.T) {
	cfg := &platform.AgentConfig{Settings: platform.AgentSettings{
		ProjectEndpoint: "https://x", AgentName: "a", AgentType: "AGENT", AuthType: "WHATEVER",
	}}
	_, err := foundry.NewFactory().CreateWorkflowClient(cfg, "tok")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unsupported")
}

func TestWorkflowClient_TestConnection_AuthHeaderSwitching(t *testing.T) {
	for _, tc := range []struct {
		name        string
		auth        string
		secret      interface{}
		userToken   string
		wantHeader  string
		wantHeaderV string
	}{
		{"user_token", "ENTRA_ID_USER_TOKEN", nil, "user-1", "Authorization", "Bearer user-1"},
		{"api_key", "API_KEY", "k-9", "", "api-key", "k-9"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			seen := ""
			seenV := ""
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if v := r.Header.Get("Authorization"); v != "" {
					seen, seenV = "Authorization", v
				} else if v := r.Header.Get("api-key"); v != "" {
					seen, seenV = "api-key", v
				}
				w.WriteHeader(http.StatusBadRequest)
				_, _ = w.Write([]byte(`{"error":{"message":"ping payload rejected"}}`))
			}))
			defer srv.Close()

			cfg := &platform.AgentConfig{Settings: platform.AgentSettings{
				ProjectEndpoint: srv.URL, APIVersion: "v1", AgentName: "BasicAgent", AgentType: "AGENT", AuthType: tc.auth,
			}}
			if tc.secret != nil {
				cfg.Settings.Credential = &platform.Credentials{Secret: tc.secret}
			}

			cli, err := foundry.NewFactory().CreateWorkflowClient(cfg, tc.userToken)
			require.NoError(t, err)
			cli.SetHTTPClient(srv.Client())

			err = cli.TestConnection(context.Background())
			require.NoError(t, err, "400 must be treated as auth-valid")
			assert.Equal(t, tc.wantHeader, seen)
			assert.Equal(t, tc.wantHeaderV, seenV)
		})
	}
}

func TestWorkflowClient_TestConnection_AppRegInjectsToken(t *testing.T) {
	stub := &stubAcquirer{tok: &clientcredentials.Token{AccessToken: "AAD-Z", ExpiresAt: time.Now().Add(time.Hour)}}
	seen := ""
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		seen = r.Header.Get("Authorization")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{}`))
	}))
	defer srv.Close()

	cfg := &platform.AgentConfig{Settings: platform.AgentSettings{
		ProjectEndpoint: srv.URL, APIVersion: "v1", AgentName: "BasicAgent", AgentType: "AGENT", AuthType: "ENTRA_ID_APP_REGISTRATION",
		Credential: &platform.Credentials{Secret: `{"tenant_id":"t","client_id":"c","client_secret":"s"}`},
	}}
	cli, err := foundry.NewFactoryWithTokenAcquirer(stub).CreateWorkflowClient(cfg, "")
	require.NoError(t, err)
	cli.SetHTTPClient(srv.Client())

	require.NoError(t, cli.TestConnection(context.Background()))
	assert.Equal(t, "Bearer AAD-Z", seen)
	assert.GreaterOrEqual(t, stub.calls, 1)
}
