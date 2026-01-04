// Package encryption provides AES-256-GCM encryption utilities for secrets management.
package encryption

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"io"
)

// Encryptor provides methods for encrypting and decrypting data.
type Encryptor interface {
	// Encrypt encrypts the given plaintext and returns base64-encoded ciphertext.
	Encrypt(plaintext []byte) (string, error)

	// Decrypt decrypts base64-encoded ciphertext and returns plaintext.
	Decrypt(ciphertext string) ([]byte, error)

	// EncryptString encrypts a string and returns base64-encoded ciphertext.
	EncryptString(plaintext string) (string, error)

	// DecryptString decrypts base64-encoded ciphertext and returns a string.
	DecryptString(ciphertext string) (string, error)
}

// AESEncryptor implements Encryptor using AES-256-GCM.
type AESEncryptor struct {
	gcm cipher.AEAD
}

// NewAESEncryptor creates a new AES-256-GCM encryptor.
// The key must be exactly 32 bytes (256 bits) for AES-256.
// Key can be provided as raw bytes or base64-encoded.
func NewAESEncryptor(key string) (*AESEncryptor, error) {
	// Try to decode as base64 first
	keyBytes, err := base64.StdEncoding.DecodeString(key)
	if err != nil {
		// If not base64, use raw bytes
		keyBytes = []byte(key)
	}

	if len(keyBytes) != 32 {
		return nil, fmt.Errorf("encryption key must be 32 bytes, got %d", len(keyBytes))
	}

	block, err := aes.NewCipher(keyBytes)
	if err != nil {
		return nil, fmt.Errorf("failed to create AES cipher: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("failed to create GCM: %w", err)
	}

	return &AESEncryptor{gcm: gcm}, nil
}

// Encrypt encrypts the given plaintext and returns base64-encoded ciphertext.
// The ciphertext includes the nonce prepended to the encrypted data.
func (e *AESEncryptor) Encrypt(plaintext []byte) (string, error) {
	// Generate a random nonce
	nonce := make([]byte, e.gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", fmt.Errorf("failed to generate nonce: %w", err)
	}

	// Encrypt and append nonce to ciphertext
	ciphertext := e.gcm.Seal(nonce, nonce, plaintext, nil)

	return base64.StdEncoding.EncodeToString(ciphertext), nil
}

// Decrypt decrypts base64-encoded ciphertext and returns plaintext.
func (e *AESEncryptor) Decrypt(ciphertext string) ([]byte, error) {
	// Decode from base64
	data, err := base64.StdEncoding.DecodeString(ciphertext)
	if err != nil {
		return nil, fmt.Errorf("failed to decode ciphertext: %w", err)
	}

	nonceSize := e.gcm.NonceSize()
	if len(data) < nonceSize {
		return nil, fmt.Errorf("ciphertext too short")
	}

	// Extract nonce and ciphertext
	nonce, ciphertextBytes := data[:nonceSize], data[nonceSize:]

	// Decrypt
	plaintext, err := e.gcm.Open(nil, nonce, ciphertextBytes, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to decrypt: %w", err)
	}

	return plaintext, nil
}

// EncryptString encrypts a string and returns base64-encoded ciphertext.
func (e *AESEncryptor) EncryptString(plaintext string) (string, error) {
	return e.Encrypt([]byte(plaintext))
}

// DecryptString decrypts base64-encoded ciphertext and returns a string.
func (e *AESEncryptor) DecryptString(ciphertext string) (string, error) {
	plaintext, err := e.Decrypt(ciphertext)
	if err != nil {
		return "", err
	}
	return string(plaintext), nil
}

// GenerateKey generates a new random 32-byte key for AES-256.
// Returns the key as base64-encoded string.
func GenerateKey() (string, error) {
	key := make([]byte, 32)
	if _, err := io.ReadFull(rand.Reader, key); err != nil {
		return "", fmt.Errorf("failed to generate key: %w", err)
	}
	return base64.StdEncoding.EncodeToString(key), nil
}

// NoOpEncryptor is a no-operation encryptor for development/testing.
// It returns the plaintext as-is (base64 encoded for Encrypt operations).
type NoOpEncryptor struct{}

// NewNoOpEncryptor creates a new no-operation encryptor.
func NewNoOpEncryptor() *NoOpEncryptor {
	return &NoOpEncryptor{}
}

// Encrypt returns the plaintext as base64.
func (e *NoOpEncryptor) Encrypt(plaintext []byte) (string, error) {
	return base64.StdEncoding.EncodeToString(plaintext), nil
}

// Decrypt decodes base64 and returns the plaintext.
func (e *NoOpEncryptor) Decrypt(ciphertext string) ([]byte, error) {
	return base64.StdEncoding.DecodeString(ciphertext)
}

// EncryptString returns the plaintext as base64.
func (e *NoOpEncryptor) EncryptString(plaintext string) (string, error) {
	return e.Encrypt([]byte(plaintext))
}

// DecryptString decodes base64 and returns the plaintext.
func (e *NoOpEncryptor) DecryptString(ciphertext string) (string, error) {
	plaintext, err := e.Decrypt(ciphertext)
	if err != nil {
		return "", err
	}
	return string(plaintext), nil
}
