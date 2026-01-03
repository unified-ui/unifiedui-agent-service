// Package vault provides the vault type constants.
package vault

// Type represents the type of vault.
type Type string

const (
	// TypeDotEnv represents a DotEnv vault (for development).
	TypeDotEnv Type = "dotenv"
	// TypeAzure represents an Azure Key Vault.
	TypeAzure Type = "azure"
	// TypeHashiCorp represents a HashiCorp Vault.
	TypeHashiCorp Type = "hashicorp"
)
