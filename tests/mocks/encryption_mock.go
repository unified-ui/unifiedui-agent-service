package mocks

import (
	"github.com/stretchr/testify/mock"
)

// MockEncryptor is a mock implementation of encryption.Encryptor.
type MockEncryptor struct {
	mock.Mock
}

// Encrypt encrypts the given plaintext and returns base64-encoded ciphertext.
func (m *MockEncryptor) Encrypt(plaintext []byte) (string, error) {
	args := m.Called(plaintext)
	return args.String(0), args.Error(1)
}

// Decrypt decrypts base64-encoded ciphertext and returns plaintext.
func (m *MockEncryptor) Decrypt(ciphertext string) ([]byte, error) {
	args := m.Called(ciphertext)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]byte), args.Error(1)
}

// EncryptString encrypts a string and returns base64-encoded ciphertext.
func (m *MockEncryptor) EncryptString(plaintext string) (string, error) {
	args := m.Called(plaintext)
	return args.String(0), args.Error(1)
}

// DecryptString decrypts base64-encoded ciphertext and returns a string.
func (m *MockEncryptor) DecryptString(ciphertext string) (string, error) {
	args := m.Called(ciphertext)
	return args.String(0), args.Error(1)
}
