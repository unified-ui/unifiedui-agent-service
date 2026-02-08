// Package platform provides the platform service client for configuration retrieval.
package platform

// AIModelWithSecretResponse represents an AI model with decrypted credential secret.
type AIModelWithSecretResponse struct {
	ID               string                 `json:"id"`
	Type             string                 `json:"type"`
	Provider         string                 `json:"provider"`
	Config           map[string]interface{} `json:"config"`
	CredentialSecret map[string]interface{} `json:"credential_secret"`
	Priority         int                    `json:"priority"`
}

// CredentialSecretResponse represents a decrypted credential secret.
type CredentialSecretResponse struct {
	CredentialID string `json:"credential_id"`
	SecretValue  string `json:"secret_value"`
}
