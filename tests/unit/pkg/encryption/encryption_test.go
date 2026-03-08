package encryption_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/unifiedui/agent-service/internal/pkg/encryption"
)

func TestNewAESEncryptor_ValidBase64Key(t *testing.T) {
	key, err := encryption.GenerateKey()
	require.NoError(t, err)
	enc, err := encryption.NewAESEncryptor(key)
	require.NoError(t, err)
	assert.NotNil(t, enc)
}

func TestNewAESEncryptor_ValidRawKey(t *testing.T) {
	rawKey := "!@#$%^&*()_+abcdefghijklmnopqrst" // 32 bytes, not valid base64
	enc, err := encryption.NewAESEncryptor(rawKey)
	require.NoError(t, err)
	assert.NotNil(t, enc)
}

func TestNewAESEncryptor_InvalidKeyLength(t *testing.T) {
	enc, err := encryption.NewAESEncryptor("tooshort")
	assert.Error(t, err)
	assert.Nil(t, enc)
	assert.Contains(t, err.Error(), "32 bytes")
}

func TestAESEncryptor_EncryptDecrypt(t *testing.T) {
	key, _ := encryption.GenerateKey()
	enc, _ := encryption.NewAESEncryptor(key)

	plaintext := []byte("hello world")
	ciphertext, err := enc.Encrypt(plaintext)
	require.NoError(t, err)
	assert.NotEmpty(t, ciphertext)
	assert.NotEqual(t, string(plaintext), ciphertext)

	decrypted, err := enc.Decrypt(ciphertext)
	require.NoError(t, err)
	assert.Equal(t, plaintext, decrypted)
}

func TestAESEncryptor_EncryptStringDecryptString(t *testing.T) {
	key, _ := encryption.GenerateKey()
	enc, _ := encryption.NewAESEncryptor(key)

	original := "secret message"
	encrypted, err := enc.EncryptString(original)
	require.NoError(t, err)
	assert.NotEqual(t, original, encrypted)

	decrypted, err := enc.DecryptString(encrypted)
	require.NoError(t, err)
	assert.Equal(t, original, decrypted)
}

func TestAESEncryptor_Decrypt_InvalidBase64(t *testing.T) {
	key, _ := encryption.GenerateKey()
	enc, _ := encryption.NewAESEncryptor(key)

	_, err := enc.Decrypt("not-valid-base64!!!")
	assert.Error(t, err)
}

func TestAESEncryptor_Decrypt_CiphertextTooShort(t *testing.T) {
	key, _ := encryption.GenerateKey()
	enc, _ := encryption.NewAESEncryptor(key)

	// base64 of just a few bytes (shorter than nonce)
	_, err := enc.Decrypt("AQID")
	assert.Error(t, err)
}

func TestAESEncryptor_Decrypt_TamperedCiphertext(t *testing.T) {
	key, _ := encryption.GenerateKey()
	enc, _ := encryption.NewAESEncryptor(key)

	ct, _ := enc.EncryptString("original")
	// Tamper with ciphertext
	tampered := ct[:len(ct)-2] + "AA"
	_, err := enc.DecryptString(tampered)
	assert.Error(t, err)
}

func TestAESEncryptor_DifferentCiphertextsForSamePlaintext(t *testing.T) {
	key, _ := encryption.GenerateKey()
	enc, _ := encryption.NewAESEncryptor(key)

	ct1, _ := enc.EncryptString("same")
	ct2, _ := enc.EncryptString("same")
	assert.NotEqual(t, ct1, ct2, "Different nonces should produce different ciphertexts")
}

func TestGenerateKey(t *testing.T) {
	key1, err := encryption.GenerateKey()
	require.NoError(t, err)
	assert.NotEmpty(t, key1)

	key2, err := encryption.GenerateKey()
	require.NoError(t, err)
	assert.NotEqual(t, key1, key2, "Keys should be unique")
}

func TestNoOpEncryptor_EncryptDecrypt(t *testing.T) {
	enc := encryption.NewNoOpEncryptor()

	plaintext := []byte("hello")
	ct, err := enc.Encrypt(plaintext)
	require.NoError(t, err)
	assert.NotEmpty(t, ct)

	decrypted, err := enc.Decrypt(ct)
	require.NoError(t, err)
	assert.Equal(t, plaintext, decrypted)
}

func TestNoOpEncryptor_EncryptStringDecryptString(t *testing.T) {
	enc := encryption.NewNoOpEncryptor()

	original := "test string"
	encrypted, err := enc.EncryptString(original)
	require.NoError(t, err)

	decrypted, err := enc.DecryptString(encrypted)
	require.NoError(t, err)
	assert.Equal(t, original, decrypted)
}

func TestNoOpEncryptor_Decrypt_InvalidBase64(t *testing.T) {
	enc := encryption.NewNoOpEncryptor()
	_, err := enc.Decrypt("not-valid-base64!!!")
	assert.Error(t, err)
}

// Additional comprehensive tests for encryption

func TestNewAESEncryptor_ExactRaw32ByteKey(t *testing.T) {
	// Exactly 32 bytes, characters that are NOT valid base64 (contains !, @, #, etc.)
	rawKey := "!@#$%^&*()_+abcdefghijklmnopqrs1"
	require.Equal(t, 32, len(rawKey), "Test key must be exactly 32 bytes")
	enc, err := encryption.NewAESEncryptor(rawKey)
	require.NoError(t, err)
	assert.NotNil(t, enc)
}

func TestNewAESEncryptor_29ByteKey(t *testing.T) {
	// 29 bytes - too short
	key := "12345678901234567890123456789"
	enc, err := encryption.NewAESEncryptor(key)
	assert.Error(t, err)
	assert.Nil(t, enc)
}

func TestNewAESEncryptor_33ByteKey(t *testing.T) {
	// 33 bytes - too long
	key := "123456789012345678901234567890123"
	enc, err := encryption.NewAESEncryptor(key)
	assert.Error(t, err)
	assert.Nil(t, enc)
}

func TestNewAESEncryptor_EmptyKey(t *testing.T) {
	enc, err := encryption.NewAESEncryptor("")
	assert.Error(t, err)
	assert.Nil(t, enc)
}

func TestAESEncryptor_EncryptEmptyPlaintext(t *testing.T) {
	key, _ := encryption.GenerateKey()
	enc, _ := encryption.NewAESEncryptor(key)

	ciphertext, err := enc.Encrypt([]byte{})
	require.NoError(t, err)
	assert.NotEmpty(t, ciphertext)

	decrypted, err := enc.Decrypt(ciphertext)
	require.NoError(t, err)
	assert.Empty(t, decrypted)
}

func TestAESEncryptor_EncryptStringEmpty(t *testing.T) {
	key, _ := encryption.GenerateKey()
	enc, _ := encryption.NewAESEncryptor(key)

	ciphertext, err := enc.EncryptString("")
	require.NoError(t, err)
	assert.NotEmpty(t, ciphertext)

	decrypted, err := enc.DecryptString(ciphertext)
	require.NoError(t, err)
	assert.Equal(t, "", decrypted)
}

func TestAESEncryptor_EncryptLargeData(t *testing.T) {
	key, _ := encryption.GenerateKey()
	enc, _ := encryption.NewAESEncryptor(key)

	// Create a large plaintext (1MB)
	largeData := make([]byte, 1024*1024)
	for i := range largeData {
		largeData[i] = byte(i % 256)
	}

	ciphertext, err := enc.Encrypt(largeData)
	require.NoError(t, err)
	assert.NotEmpty(t, ciphertext)

	decrypted, err := enc.Decrypt(ciphertext)
	require.NoError(t, err)
	assert.Equal(t, largeData, decrypted)
}

func TestAESEncryptor_EncryptBinaryData(t *testing.T) {
	key, _ := encryption.GenerateKey()
	enc, _ := encryption.NewAESEncryptor(key)

	// Binary data with null bytes and all possible byte values
	binaryData := make([]byte, 256)
	for i := range binaryData {
		binaryData[i] = byte(i)
	}

	ciphertext, err := enc.Encrypt(binaryData)
	require.NoError(t, err)

	decrypted, err := enc.Decrypt(ciphertext)
	require.NoError(t, err)
	assert.Equal(t, binaryData, decrypted)
}

func TestAESEncryptor_DecryptString_InvalidBase64(t *testing.T) {
	key, _ := encryption.GenerateKey()
	enc, _ := encryption.NewAESEncryptor(key)

	_, err := enc.DecryptString("not-valid-base64!!!")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "decode ciphertext")
}

func TestAESEncryptor_DifferentKeysCannotDecrypt(t *testing.T) {
	key1, _ := encryption.GenerateKey()
	key2, _ := encryption.GenerateKey()

	enc1, _ := encryption.NewAESEncryptor(key1)
	enc2, _ := encryption.NewAESEncryptor(key2)

	ciphertext, err := enc1.EncryptString("secret")
	require.NoError(t, err)

	// Attempting to decrypt with a different key should fail
	_, err = enc2.DecryptString(ciphertext)
	assert.Error(t, err)
}

func TestAESEncryptor_SameKeyCanDecrypt(t *testing.T) {
	key, _ := encryption.GenerateKey()

	enc1, _ := encryption.NewAESEncryptor(key)
	enc2, _ := encryption.NewAESEncryptor(key) // Same key, different instance

	ciphertext, err := enc1.EncryptString("secret")
	require.NoError(t, err)

	decrypted, err := enc2.DecryptString(ciphertext)
	require.NoError(t, err)
	assert.Equal(t, "secret", decrypted)
}

func TestNoOpEncryptor_EncryptStringDecryptString_SpecialChars(t *testing.T) {
	enc := encryption.NewNoOpEncryptor()

	original := "test string with spaces and symbols !@#$%"
	encrypted, err := enc.EncryptString(original)
	require.NoError(t, err)

	decrypted, err := enc.DecryptString(encrypted)
	require.NoError(t, err)
	assert.Equal(t, original, decrypted)
}

func TestNoOpEncryptor_DecryptString_InvalidBase64(t *testing.T) {
	enc := encryption.NewNoOpEncryptor()
	_, err := enc.DecryptString("invalid-base64!!!")
	assert.Error(t, err)
}

func TestNoOpEncryptor_EmptyInput(t *testing.T) {
	enc := encryption.NewNoOpEncryptor()

	encrypted, err := enc.Encrypt([]byte{})
	require.NoError(t, err)

	decrypted, err := enc.Decrypt(encrypted)
	require.NoError(t, err)
	assert.Empty(t, decrypted)
}

func TestNoOpEncryptor_EmptyStringInput(t *testing.T) {
	enc := encryption.NewNoOpEncryptor()

	encrypted, err := enc.EncryptString("")
	require.NoError(t, err)

	decrypted, err := enc.DecryptString(encrypted)
	require.NoError(t, err)
	assert.Equal(t, "", decrypted)
}

func TestNoOpEncryptor_BinaryData(t *testing.T) {
	enc := encryption.NewNoOpEncryptor()

	binaryData := []byte{0x00, 0x01, 0x02, 0xFF, 0xFE, 0xFD}
	encrypted, err := enc.Encrypt(binaryData)
	require.NoError(t, err)

	decrypted, err := enc.Decrypt(encrypted)
	require.NoError(t, err)
	assert.Equal(t, binaryData, decrypted)
}

func TestGenerateKey_ValidForEncryption(t *testing.T) {
	// Generate multiple keys and verify each can be used
	for i := 0; i < 5; i++ {
		key, err := encryption.GenerateKey()
		require.NoError(t, err)

		enc, err := encryption.NewAESEncryptor(key)
		require.NoError(t, err)
		assert.NotNil(t, enc)

		// Verify encryption/decryption works
		ct, err := enc.EncryptString("test")
		require.NoError(t, err)
		pt, err := enc.DecryptString(ct)
		require.NoError(t, err)
		assert.Equal(t, "test", pt)
	}
}

func TestAESEncryptor_EncryptUnicode(t *testing.T) {
	key, _ := encryption.GenerateKey()
	enc, _ := encryption.NewAESEncryptor(key)

	unicodeText := "Hello 世界 🌍 مرحبا мир"
	ciphertext, err := enc.EncryptString(unicodeText)
	require.NoError(t, err)

	decrypted, err := enc.DecryptString(ciphertext)
	require.NoError(t, err)
	assert.Equal(t, unicodeText, decrypted)
}
