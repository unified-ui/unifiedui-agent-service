package encryption_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/unifiedui/agent-service/internal/pkg/encryption"
)

// generateTestKey creates a valid 32-byte key for testing.
// Uses a base64-encoded key to ensure consistent 32 bytes.
func generateTestKey(t *testing.T) string {
	key, err := encryption.GenerateKey()
	require.NoError(t, err)
	return key
}

func TestNewAESEncryptor_ValidBase64Key(t *testing.T) {
	// Arrange - use a generated base64-encoded 32-byte key
	key := generateTestKey(t)

	// Act
	encryptor, err := encryption.NewAESEncryptor(key)

	// Assert
	require.NoError(t, err)
	assert.NotNil(t, encryptor)
}

func TestNewAESEncryptor_Base64Key(t *testing.T) {
	// Arrange - generate a valid base64-encoded 32-byte key
	key, err := encryption.GenerateKey()
	require.NoError(t, err)

	// Act
	encryptor, err := encryption.NewAESEncryptor(key)

	// Assert
	require.NoError(t, err)
	assert.NotNil(t, encryptor)
}

func TestNewAESEncryptor_InvalidKeyLength(t *testing.T) {
	// Arrange - key too short (not valid base64)
	key := "tooshort!!!"

	// Act
	encryptor, err := encryption.NewAESEncryptor(key)

	// Assert
	assert.Error(t, err)
	assert.Nil(t, encryptor)
	assert.Contains(t, err.Error(), "must be 32 bytes")
}

func TestAESEncryptor_EncryptDecrypt(t *testing.T) {
	// Arrange
	key := generateTestKey(t)
	encryptor, err := encryption.NewAESEncryptor(key)
	require.NoError(t, err)

	plaintext := []byte("Hello, World! This is a secret message.")

	// Act
	ciphertext, err := encryptor.Encrypt(plaintext)
	require.NoError(t, err)

	decrypted, err := encryptor.Decrypt(ciphertext)
	require.NoError(t, err)

	// Assert
	assert.Equal(t, plaintext, decrypted)
	assert.NotEqual(t, string(plaintext), ciphertext) // Ciphertext should be different
}

func TestAESEncryptor_EncryptDecryptString(t *testing.T) {
	// Arrange
	key := generateTestKey(t)
	encryptor, err := encryption.NewAESEncryptor(key)
	require.NoError(t, err)

	plaintext := "Hello, World! This is a secret message."

	// Act
	ciphertext, err := encryptor.EncryptString(plaintext)
	require.NoError(t, err)

	decrypted, err := encryptor.DecryptString(ciphertext)
	require.NoError(t, err)

	// Assert
	assert.Equal(t, plaintext, decrypted)
}

func TestAESEncryptor_Decrypt_InvalidCiphertext(t *testing.T) {
	// Arrange
	key := generateTestKey(t)
	encryptor, err := encryption.NewAESEncryptor(key)
	require.NoError(t, err)

	// Act
	_, err = encryptor.Decrypt("not-valid-base64!!!")

	// Assert
	assert.Error(t, err)
}

func TestAESEncryptor_Decrypt_TamperedCiphertext(t *testing.T) {
	// Arrange
	key := generateTestKey(t)
	encryptor, err := encryption.NewAESEncryptor(key)
	require.NoError(t, err)

	plaintext := []byte("Secret message")
	ciphertext, err := encryptor.Encrypt(plaintext)
	require.NoError(t, err)

	// Tamper with the ciphertext by appending characters
	tamperedCiphertext := ciphertext + "X"

	// Act
	_, err = encryptor.Decrypt(tamperedCiphertext)

	// Assert
	assert.Error(t, err)
}

func TestAESEncryptor_EncryptDifferentNonces(t *testing.T) {
	// Arrange
	key := generateTestKey(t)
	encryptor, err := encryption.NewAESEncryptor(key)
	require.NoError(t, err)

	plaintext := []byte("Same message")

	// Act - encrypt same message twice
	ciphertext1, err := encryptor.Encrypt(plaintext)
	require.NoError(t, err)

	ciphertext2, err := encryptor.Encrypt(plaintext)
	require.NoError(t, err)

	// Assert - ciphertexts should be different due to random nonces
	assert.NotEqual(t, ciphertext1, ciphertext2)

	// But both should decrypt to the same plaintext
	decrypted1, err := encryptor.Decrypt(ciphertext1)
	require.NoError(t, err)

	decrypted2, err := encryptor.Decrypt(ciphertext2)
	require.NoError(t, err)

	assert.Equal(t, decrypted1, decrypted2)
}

func TestGenerateKey(t *testing.T) {
	// Act
	key, err := encryption.GenerateKey()

	// Assert
	require.NoError(t, err)
	assert.NotEmpty(t, key)

	// Verify key can be used to create encryptor
	encryptor, err := encryption.NewAESEncryptor(key)
	require.NoError(t, err)
	assert.NotNil(t, encryptor)
}

func TestGenerateKey_Uniqueness(t *testing.T) {
	// Act
	key1, err := encryption.GenerateKey()
	require.NoError(t, err)

	key2, err := encryption.GenerateKey()
	require.NoError(t, err)

	// Assert
	assert.NotEqual(t, key1, key2)
}

func TestNoOpEncryptor(t *testing.T) {
	// Arrange
	encryptor := encryption.NewNoOpEncryptor()
	plaintext := []byte("test message")

	// Act
	encrypted, err := encryptor.Encrypt(plaintext)
	require.NoError(t, err)

	decrypted, err := encryptor.Decrypt(encrypted)
	require.NoError(t, err)

	// Assert
	assert.Equal(t, plaintext, decrypted)
}

func TestNoOpEncryptor_String(t *testing.T) {
	// Arrange
	encryptor := encryption.NewNoOpEncryptor()
	plaintext := "test message"

	// Act
	encrypted, err := encryptor.EncryptString(plaintext)
	require.NoError(t, err)

	decrypted, err := encryptor.DecryptString(encrypted)
	require.NoError(t, err)

	// Assert
	assert.Equal(t, plaintext, decrypted)
}
