package state

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"sync"

	"golang.org/x/crypto/pbkdf2"
)

// EncryptionConfig configures state encryption
type EncryptionConfig struct {
	Enabled     bool   `yaml:"enabled" json:"enabled"`
	KeyProvider string `yaml:"keyProvider" json:"keyProvider"` // env, file, vault, awskms
	KeyID       string `yaml:"keyId,omitempty" json:"keyId,omitempty"`
	KeyFile     string `yaml:"keyFile,omitempty" json:"keyFile,omitempty"`
	KeyEnvVar   string `yaml:"keyEnvVar,omitempty" json:"keyEnvVar,omitempty"`
}

// Encryptor handles state encryption and decryption
type Encryptor struct {
	config      *EncryptionConfig
	key         []byte
	gcm         cipher.AEAD
	mu          sync.RWMutex
	initialized bool
}

// EncryptedData represents encrypted state data
type EncryptedData struct {
	Version   int    `json:"version"`
	Algorithm string `json:"algorithm"`
	Nonce     string `json:"nonce"`
	Data      string `json:"data"`
	KeyID     string `json:"keyId,omitempty"`
}

const (
	encryptionVersion = 1
	algorithm         = "AES-256-GCM"
	keyLength         = 32 // 256 bits
	nonceLength       = 12
	saltLength        = 16
	pbkdf2Iterations  = 100000
)

// NewEncryptor creates a new state encryptor
func NewEncryptor(config *EncryptionConfig) (*Encryptor, error) {
	if config == nil || !config.Enabled {
		return &Encryptor{config: config}, nil
	}

	e := &Encryptor{
		config: config,
	}

	if err := e.initialize(); err != nil {
		return nil, err
	}

	return e, nil
}

func (e *Encryptor) initialize() error {
	e.mu.Lock()
	defer e.mu.Unlock()

	if e.initialized {
		return nil
	}

	// Get the encryption key based on the provider
	key, err := e.loadKey()
	if err != nil {
		return fmt.Errorf("failed to load encryption key: %w", err)
	}

	if len(key) < keyLength {
		// Derive a key using PBKDF2 if the provided key is too short
		salt := make([]byte, saltLength)
		if _, err := io.ReadFull(rand.Reader, salt); err != nil {
			return fmt.Errorf("failed to generate salt: %w", err)
		}
		key = pbkdf2.Key(key, salt, pbkdf2Iterations, keyLength, sha256.New)
	} else if len(key) > keyLength {
		// Hash the key to get the correct length
		hash := sha256.Sum256(key)
		key = hash[:]
	}

	e.key = key

	// Initialize AES-GCM
	block, err := aes.NewCipher(key)
	if err != nil {
		return fmt.Errorf("failed to create cipher: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return fmt.Errorf("failed to create GCM: %w", err)
	}

	e.gcm = gcm
	e.initialized = true

	return nil
}

func (e *Encryptor) loadKey() ([]byte, error) {
	switch e.config.KeyProvider {
	case "env":
		envVar := e.config.KeyEnvVar
		if envVar == "" {
			envVar = "PF_STATE_ENCRYPTION_KEY"
		}
		key := os.Getenv(envVar)
		if key == "" {
			return nil, fmt.Errorf("encryption key environment variable %s not set", envVar)
		}
		return []byte(key), nil

	case "file":
		if e.config.KeyFile == "" {
			return nil, errors.New("key file path not specified")
		}
		data, err := os.ReadFile(e.config.KeyFile)
		if err != nil {
			return nil, fmt.Errorf("failed to read key file: %w", err)
		}
		// Decode if base64 encoded
		if decoded, err := base64.StdEncoding.DecodeString(string(data)); err == nil {
			return decoded, nil
		}
		return data, nil

	case "vault":
		return e.loadKeyFromVault()

	case "awskms":
		return e.loadKeyFromAWSKMS()

	default:
		// Try environment variable as default
		key := os.Getenv("PF_STATE_ENCRYPTION_KEY")
		if key != "" {
			return []byte(key), nil
		}
		return nil, errors.New("no encryption key provider configured")
	}
}

func (e *Encryptor) loadKeyFromVault() ([]byte, error) {
	// In production, this would use HashiCorp Vault API
	// For now, fall back to environment variable
	vaultAddr := os.Getenv("VAULT_ADDR")
	vaultToken := os.Getenv("VAULT_TOKEN")

	if vaultAddr == "" || vaultToken == "" {
		return nil, errors.New("VAULT_ADDR and VAULT_TOKEN must be set for Vault key provider")
	}

	// The actual Vault integration would go here
	// This is a placeholder that uses the key ID to look up the secret
	keyPath := e.config.KeyID
	if keyPath == "" {
		keyPath = "secret/data/platformfoundry/encryption-key"
	}

	// For now, return an error indicating Vault integration is needed
	return nil, fmt.Errorf("Vault key provider requires configuration at %s", keyPath)
}

func (e *Encryptor) loadKeyFromAWSKMS() ([]byte, error) {
	// In production, this would use AWS KMS to generate/retrieve a data key
	keyID := e.config.KeyID
	if keyID == "" {
		return nil, errors.New("AWS KMS key ID must be specified")
	}

	// The actual AWS KMS integration would go here
	// This would typically:
	// 1. Call KMS GenerateDataKey or Decrypt
	// 2. Return the plaintext data key

	return nil, fmt.Errorf("AWS KMS key provider requires key ID: %s", keyID)
}

// Encrypt encrypts the given data
func (e *Encryptor) Encrypt(data []byte) ([]byte, error) {
	if !e.config.Enabled {
		return data, nil
	}

	e.mu.RLock()
	defer e.mu.RUnlock()

	if !e.initialized {
		return nil, errors.New("encryptor not initialized")
	}

	// Generate a random nonce
	nonce := make([]byte, nonceLength)
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, fmt.Errorf("failed to generate nonce: %w", err)
	}

	// Encrypt the data
	ciphertext := e.gcm.Seal(nil, nonce, data, nil)

	// Package the encrypted data
	encrypted := EncryptedData{
		Version:   encryptionVersion,
		Algorithm: algorithm,
		Nonce:     base64.StdEncoding.EncodeToString(nonce),
		Data:      base64.StdEncoding.EncodeToString(ciphertext),
		KeyID:     e.config.KeyID,
	}

	return json.Marshal(encrypted)
}

// Decrypt decrypts the given data
func (e *Encryptor) Decrypt(data []byte) ([]byte, error) {
	if !e.config.Enabled {
		return data, nil
	}

	e.mu.RLock()
	defer e.mu.RUnlock()

	if !e.initialized {
		return nil, errors.New("encryptor not initialized")
	}

	// Check if data is encrypted (starts with JSON object)
	if len(data) == 0 || data[0] != '{' {
		// Data is not encrypted, return as-is
		return data, nil
	}

	// Parse the encrypted data
	var encrypted EncryptedData
	if err := json.Unmarshal(data, &encrypted); err != nil {
		// Not a valid encrypted format, return as-is
		return data, nil
	}

	// Verify version and algorithm
	if encrypted.Version != encryptionVersion {
		return nil, fmt.Errorf("unsupported encryption version: %d", encrypted.Version)
	}
	if encrypted.Algorithm != algorithm {
		return nil, fmt.Errorf("unsupported encryption algorithm: %s", encrypted.Algorithm)
	}

	// Decode nonce and ciphertext
	nonce, err := base64.StdEncoding.DecodeString(encrypted.Nonce)
	if err != nil {
		return nil, fmt.Errorf("failed to decode nonce: %w", err)
	}

	ciphertext, err := base64.StdEncoding.DecodeString(encrypted.Data)
	if err != nil {
		return nil, fmt.Errorf("failed to decode ciphertext: %w", err)
	}

	// Decrypt the data
	plaintext, err := e.gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return nil, fmt.Errorf("decryption failed: %w", err)
	}

	return plaintext, nil
}

// IsEnabled returns whether encryption is enabled
func (e *Encryptor) IsEnabled() bool {
	return e.config != nil && e.config.Enabled
}

// RotateKey rotates the encryption key
func (e *Encryptor) RotateKey(newKey []byte) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	if len(newKey) < keyLength {
		hash := sha256.Sum256(newKey)
		newKey = hash[:]
	} else if len(newKey) > keyLength {
		hash := sha256.Sum256(newKey)
		newKey = hash[:]
	}

	block, err := aes.NewCipher(newKey)
	if err != nil {
		return fmt.Errorf("failed to create cipher with new key: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return fmt.Errorf("failed to create GCM with new key: %w", err)
	}

	e.key = newKey
	e.gcm = gcm

	return nil
}

// GenerateKey generates a new random encryption key
func GenerateKey() ([]byte, error) {
	key := make([]byte, keyLength)
	if _, err := io.ReadFull(rand.Reader, key); err != nil {
		return nil, fmt.Errorf("failed to generate key: %w", err)
	}
	return key, nil
}

// EncodeKey encodes a key to base64 for storage
func EncodeKey(key []byte) string {
	return base64.StdEncoding.EncodeToString(key)
}

// DecodeKey decodes a base64-encoded key
func DecodeKey(encoded string) ([]byte, error) {
	return base64.StdEncoding.DecodeString(encoded)
}
