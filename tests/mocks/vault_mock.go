// Package mocks provides mock implementations for testing.
package mocks

import (
	"context"

	"github.com/stretchr/testify/mock"

	"github.com/unifiedui/agent-service/internal/core/vault"
)

// MockVault is a mock implementation of vault.Vault.
type MockVault struct {
	mock.Mock
}

// StoreSecret stores a secret in the vault.
func (m *MockVault) StoreSecret(ctx context.Context, key string, value string, metadata map[string]string) (string, error) {
	args := m.Called(ctx, key, value, metadata)
	return args.String(0), args.Error(1)
}

// GetSecret retrieves a secret from the vault.
func (m *MockVault) GetSecret(ctx context.Context, uri string) (string, error) {
	args := m.Called(ctx, uri)
	return args.String(0), args.Error(1)
}

// UpdateSecret updates an existing secret.
func (m *MockVault) UpdateSecret(ctx context.Context, uri string, value string, metadata map[string]string) (bool, error) {
	args := m.Called(ctx, uri, value, metadata)
	return args.Bool(0), args.Error(1)
}

// DeleteSecret deletes a secret from the vault.
func (m *MockVault) DeleteSecret(ctx context.Context, uri string) (bool, error) {
	args := m.Called(ctx, uri)
	return args.Bool(0), args.Error(1)
}

// Ping checks the vault connection.
func (m *MockVault) Ping(ctx context.Context) error {
	args := m.Called(ctx)
	return args.Error(0)
}

// Close closes the vault connection.
func (m *MockVault) Close() error {
	args := m.Called()
	return args.Error(0)
}

// MockVaultClient is a mock implementation of vault.Client.
type MockVaultClient struct {
	mock.Mock
	vault *MockVault
}

// NewMockVaultClient creates a new MockVaultClient.
func NewMockVaultClient() *MockVaultClient {
	return &MockVaultClient{
		vault: &MockVault{},
	}
}

// GetVault returns the underlying vault.
func (m *MockVaultClient) GetVault() vault.Vault {
	return m.vault
}

// StoreSecret stores a secret in the vault.
func (m *MockVaultClient) StoreSecret(ctx context.Context, key string, value string, metadata map[string]string) (string, error) {
	args := m.Called(ctx, key, value, metadata)
	return args.String(0), args.Error(1)
}

// GetSecret retrieves a secret from the vault.
func (m *MockVaultClient) GetSecret(ctx context.Context, uri string, useCache bool) (string, error) {
	args := m.Called(ctx, uri, useCache)
	return args.String(0), args.Error(1)
}

// UpdateSecret updates an existing secret.
func (m *MockVaultClient) UpdateSecret(ctx context.Context, uri string, value string, metadata map[string]string) (bool, error) {
	args := m.Called(ctx, uri, value, metadata)
	return args.Bool(0), args.Error(1)
}

// DeleteSecret deletes a secret from the vault.
func (m *MockVaultClient) DeleteSecret(ctx context.Context, uri string) (bool, error) {
	args := m.Called(ctx, uri)
	return args.Bool(0), args.Error(1)
}

// Ping checks the vault connection.
func (m *MockVaultClient) Ping(ctx context.Context) error {
	args := m.Called(ctx)
	return args.Error(0)
}

// Close closes the vault connection.
func (m *MockVaultClient) Close() error {
	args := m.Called()
	return args.Error(0)
}
