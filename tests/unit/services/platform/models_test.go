package platform_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/unifiedui/agent-service/internal/services/platform"
)

func TestCredentials_GetSecretAsString_String(t *testing.T) {
	cred := &platform.Credentials{Secret: "my-api-key"}
	require.Equal(t, "my-api-key", cred.GetSecretAsString())
}

func TestCredentials_GetSecretAsString_NotString(t *testing.T) {
	cred := &platform.Credentials{Secret: map[string]interface{}{"username": "u"}}
	require.Equal(t, "", cred.GetSecretAsString())
}

func TestCredentials_GetSecretAsString_Nil(t *testing.T) {
	cred := &platform.Credentials{Secret: nil}
	require.Equal(t, "", cred.GetSecretAsString())
}

func TestCredentials_GetSecretAsBasicAuth_Valid(t *testing.T) {
	cred := &platform.Credentials{
		Secret: map[string]interface{}{
			"username": "admin",
			"password": "secret",
		},
	}
	auth := cred.GetSecretAsBasicAuth()
	require.NotNil(t, auth)
	require.Equal(t, "admin", auth.Username)
	require.Equal(t, "secret", auth.Password)
}

func TestCredentials_GetSecretAsBasicAuth_NotMap(t *testing.T) {
	cred := &platform.Credentials{Secret: "just-a-string"}
	require.Nil(t, cred.GetSecretAsBasicAuth())
}

func TestCredentials_GetSecretAsBasicAuth_Nil(t *testing.T) {
	cred := &platform.Credentials{Secret: nil}
	require.Nil(t, cred.GetSecretAsBasicAuth())
}
