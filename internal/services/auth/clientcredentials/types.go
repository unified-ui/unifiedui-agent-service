// Package clientcredentials provides Microsoft Entra ID client-credentials
// (app registration) token acquisition with optional caching support.
package clientcredentials

import (
	"errors"
	"time"
)

// Credentials holds the Entra ID app registration credentials.
type Credentials struct {
	TenantID     string
	ClientID     string
	ClientSecret string
}

// Token represents a cached access token with expiry.
type Token struct {
	AccessToken string    `json:"access_token"`
	ExpiresAt   time.Time `json:"expires_at"`
}

// IsExpired reports whether the token is expired (with a 30s safety margin).
func (t *Token) IsExpired() bool {
	return time.Now().After(t.ExpiresAt.Add(-30 * time.Second))
}

// ErrInvalidCredentials is returned when AAD rejects the credentials.
var ErrInvalidCredentials = errors.New("clientcredentials: invalid credentials")
