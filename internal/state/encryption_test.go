package state

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewEncryptor_Disabled(t *testing.T) {
	enc, err := NewEncryptor(nil)
	require.NoError(t, err)
	assert.NotNil(t, enc)
	assert.False(t, enc.IsEnabled())
}

func TestNewEncryptor_DisabledExplicit(t *testing.T) {
	cfg := &EncryptionConfig{Enabled: false}
	enc, err := NewEncryptor(cfg)
	require.NoError(t, err)
	assert.NotNil(t, enc)
	assert.False(t, enc.IsEnabled())
}

func TestNewEncryptor_WithEnvKey(t *testing.T) {
	// Set test key
	testKey := "test-encryption-key-32-bytes-ok"
	os.Setenv("PF_STATE_ENCRYPTION_KEY", testKey)
	defer os.Unsetenv("PF_STATE_ENCRYPTION_KEY")

	cfg := &EncryptionConfig{
		Enabled:     true,
		KeyProvider: "env",
	}

	enc, err := NewEncryptor(cfg)
	require.NoError(t, err)
	assert.NotNil(t, enc)
	assert.True(t, enc.IsEnabled())
}

func TestNewEncryptor_WithCustomEnvVar(t *testing.T) {
	testKey := "another-test-key-for-encryption"
	os.Setenv("CUSTOM_KEY_VAR", testKey)
	defer os.Unsetenv("CUSTOM_KEY_VAR")

	cfg := &EncryptionConfig{
		Enabled:     true,
		KeyProvider: "env",
		KeyEnvVar:   "CUSTOM_KEY_VAR",
	}

	enc, err := NewEncryptor(cfg)
	require.NoError(t, err)
	assert.True(t, enc.IsEnabled())
}

func TestNewEncryptor_MissingEnvKey(t *testing.T) {
	os.Unsetenv("PF_STATE_ENCRYPTION_KEY")

	cfg := &EncryptionConfig{
		Enabled:     true,
		KeyProvider: "env",
	}

	_, err := NewEncryptor(cfg)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not set")
}

func TestEncryptDecrypt_RoundTrip(t *testing.T) {
	os.Setenv("PF_STATE_ENCRYPTION_KEY", "test-key-for-roundtrip-testing!")
	defer os.Unsetenv("PF_STATE_ENCRYPTION_KEY")

	cfg := &EncryptionConfig{
		Enabled:     true,
		KeyProvider: "env",
	}

	enc, err := NewEncryptor(cfg)
	require.NoError(t, err)

	testCases := []struct {
		name string
		data []byte
	}{
		{"empty", []byte{}},
		{"small", []byte("hello world")},
		{"json", []byte(`{"key": "value", "number": 42}`)},
		{"large", make([]byte, 10000)},
		{"binary", []byte{0x00, 0x01, 0x02, 0xFF, 0xFE, 0xFD}},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			encrypted, err := enc.Encrypt(tc.data)
			require.NoError(t, err)
			if len(tc.data) > 0 {
				assert.NotEqual(t, tc.data, encrypted)
			}

			decrypted, err := enc.Decrypt(encrypted)
			require.NoError(t, err)
			// Handle nil vs empty slice equivalence
			if len(tc.data) == 0 {
				assert.Len(t, decrypted, 0)
			} else {
				assert.Equal(t, tc.data, decrypted)
			}
		})
	}
}

func TestEncrypt_DisabledPassthrough(t *testing.T) {
	enc, err := NewEncryptor(&EncryptionConfig{Enabled: false})
	require.NoError(t, err)

	data := []byte("plaintext data")
	result, err := enc.Encrypt(data)
	require.NoError(t, err)
	assert.Equal(t, data, result)
}

func TestDecrypt_DisabledPassthrough(t *testing.T) {
	enc, err := NewEncryptor(&EncryptionConfig{Enabled: false})
	require.NoError(t, err)

	data := []byte("plaintext data")
	result, err := enc.Decrypt(data)
	require.NoError(t, err)
	assert.Equal(t, data, result)
}

func TestDecrypt_NonEncryptedData(t *testing.T) {
	os.Setenv("PF_STATE_ENCRYPTION_KEY", "test-key-for-non-encrypted-test")
	defer os.Unsetenv("PF_STATE_ENCRYPTION_KEY")

	cfg := &EncryptionConfig{
		Enabled:     true,
		KeyProvider: "env",
	}

	enc, err := NewEncryptor(cfg)
	require.NoError(t, err)

	// Non-JSON data should pass through
	plainData := []byte("not encrypted data")
	result, err := enc.Decrypt(plainData)
	require.NoError(t, err)
	assert.Equal(t, plainData, result)
}

func TestDecrypt_InvalidJSON(t *testing.T) {
	os.Setenv("PF_STATE_ENCRYPTION_KEY", "test-key-for-invalid-json-test")
	defer os.Unsetenv("PF_STATE_ENCRYPTION_KEY")

	cfg := &EncryptionConfig{
		Enabled:     true,
		KeyProvider: "env",
	}

	enc, err := NewEncryptor(cfg)
	require.NoError(t, err)

	// Invalid JSON starting with { should pass through
	invalidJSON := []byte("{invalid json")
	result, err := enc.Decrypt(invalidJSON)
	require.NoError(t, err)
	assert.Equal(t, invalidJSON, result)
}

func TestEncryptedData_Structure(t *testing.T) {
	os.Setenv("PF_STATE_ENCRYPTION_KEY", "test-key-structure-validation!")
	defer os.Unsetenv("PF_STATE_ENCRYPTION_KEY")

	cfg := &EncryptionConfig{
		Enabled:     true,
		KeyProvider: "env",
		KeyID:       "test-key-id",
	}

	enc, err := NewEncryptor(cfg)
	require.NoError(t, err)

	encrypted, err := enc.Encrypt([]byte("test data"))
	require.NoError(t, err)

	// Should be valid JSON
	assert.True(t, encrypted[0] == '{')

	// Should contain expected fields
	assert.Contains(t, string(encrypted), `"version":1`)
	assert.Contains(t, string(encrypted), `"algorithm":"AES-256-GCM"`)
	assert.Contains(t, string(encrypted), `"nonce":`)
	assert.Contains(t, string(encrypted), `"data":`)
	assert.Contains(t, string(encrypted), `"keyId":"test-key-id"`)
}

func TestRotateKey(t *testing.T) {
	os.Setenv("PF_STATE_ENCRYPTION_KEY", "original-key-for-rotation-test")
	defer os.Unsetenv("PF_STATE_ENCRYPTION_KEY")

	cfg := &EncryptionConfig{
		Enabled:     true,
		KeyProvider: "env",
	}

	enc, err := NewEncryptor(cfg)
	require.NoError(t, err)

	// Encrypt with original key
	originalData := []byte("sensitive data")
	encrypted, err := enc.Encrypt(originalData)
	require.NoError(t, err)

	// Rotate to new key
	newKey := []byte("new-key-after-rotation-testing!")
	err = enc.RotateKey(newKey)
	require.NoError(t, err)

	// Old encrypted data should fail to decrypt
	_, err = enc.Decrypt(encrypted)
	assert.Error(t, err)

	// New encryption should work
	newEncrypted, err := enc.Encrypt(originalData)
	require.NoError(t, err)

	decrypted, err := enc.Decrypt(newEncrypted)
	require.NoError(t, err)
	assert.Equal(t, originalData, decrypted)
}

func TestGenerateKey(t *testing.T) {
	key1, err := GenerateKey()
	require.NoError(t, err)
	assert.Len(t, key1, 32)

	key2, err := GenerateKey()
	require.NoError(t, err)
	assert.Len(t, key2, 32)

	// Keys should be different (random)
	assert.NotEqual(t, key1, key2)
}

func TestEncodeDecodeKey(t *testing.T) {
	key, err := GenerateKey()
	require.NoError(t, err)

	encoded := EncodeKey(key)
	assert.NotEmpty(t, encoded)

	decoded, err := DecodeKey(encoded)
	require.NoError(t, err)
	assert.Equal(t, key, decoded)
}

func TestNewEncryptor_FileProvider_MissingPath(t *testing.T) {
	cfg := &EncryptionConfig{
		Enabled:     true,
		KeyProvider: "file",
	}

	_, err := NewEncryptor(cfg)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "key file path not specified")
}

func TestNewEncryptor_VaultProvider_MissingConfig(t *testing.T) {
	os.Unsetenv("VAULT_ADDR")
	os.Unsetenv("VAULT_TOKEN")

	cfg := &EncryptionConfig{
		Enabled:     true,
		KeyProvider: "vault",
	}

	_, err := NewEncryptor(cfg)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "VAULT_ADDR")
}

func TestNewEncryptor_AWSKMSProvider_MissingKeyID(t *testing.T) {
	cfg := &EncryptionConfig{
		Enabled:     true,
		KeyProvider: "awskms",
	}

	_, err := NewEncryptor(cfg)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "key ID")
}

func TestIsEnabled(t *testing.T) {
	tests := []struct {
		name     string
		config   *EncryptionConfig
		expected bool
	}{
		{"nil config", nil, false},
		{"disabled", &EncryptionConfig{Enabled: false}, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			enc := &Encryptor{config: tt.config}
			assert.Equal(t, tt.expected, enc.IsEnabled())
		})
	}
}

func TestEncrypt_UniqueNonce(t *testing.T) {
	os.Setenv("PF_STATE_ENCRYPTION_KEY", "test-key-for-unique-nonce-test!")
	defer os.Unsetenv("PF_STATE_ENCRYPTION_KEY")

	cfg := &EncryptionConfig{
		Enabled:     true,
		KeyProvider: "env",
	}

	enc, err := NewEncryptor(cfg)
	require.NoError(t, err)

	data := []byte("same data")

	// Encrypt same data multiple times
	encrypted1, err := enc.Encrypt(data)
	require.NoError(t, err)

	encrypted2, err := enc.Encrypt(data)
	require.NoError(t, err)

	// Should produce different ciphertext due to random nonce
	assert.NotEqual(t, encrypted1, encrypted2)

	// But both should decrypt to same plaintext
	decrypted1, err := enc.Decrypt(encrypted1)
	require.NoError(t, err)

	decrypted2, err := enc.Decrypt(encrypted2)
	require.NoError(t, err)

	assert.Equal(t, data, decrypted1)
	assert.Equal(t, data, decrypted2)
}
