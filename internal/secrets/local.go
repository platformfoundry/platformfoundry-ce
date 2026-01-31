package secrets

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// LocalManager implements Manager using local encrypted file storage
type LocalManager struct {
	filePath string
	gcm      cipher.AEAD
	secrets  map[string]*Secret
	mu       sync.RWMutex
}

// NewLocalManager creates a new local secrets manager
func NewLocalManager(config *LocalConfig) (*LocalManager, error) {
	// Set default path
	filePath := config.Path
	if filePath == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return nil, fmt.Errorf("failed to get home directory: %w", err)
		}
		filePath = filepath.Join(home, ".platformfoundry", "secrets.enc")
	}

	// Ensure directory exists
	dir := filepath.Dir(filePath)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return nil, fmt.Errorf("failed to create secrets directory: %w", err)
	}

	// Get or generate encryption key
	encryptionKey := config.EncryptionKey
	if encryptionKey == "" {
		// Check environment variable
		encryptionKey = os.Getenv("PF_SECRETS_KEY")
		if encryptionKey == "" {
			// Generate a new key for development
			keyFile := filepath.Join(dir, "secrets.key")
			key, err := loadOrGenerateKey(keyFile)
			if err != nil {
				return nil, fmt.Errorf("failed to load encryption key: %w", err)
			}
			encryptionKey = key
		}
	}

	// Create AES cipher
	hash := sha256.Sum256([]byte(encryptionKey))
	block, err := aes.NewCipher(hash[:])
	if err != nil {
		return nil, fmt.Errorf("failed to create cipher: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("failed to create GCM: %w", err)
	}

	manager := &LocalManager{
		filePath: filePath,
		gcm:      gcm,
		secrets:  make(map[string]*Secret),
	}

	// Load existing secrets
	if err := manager.load(); err != nil {
		if !os.IsNotExist(err) {
			return nil, err
		}
	}

	return manager, nil
}

// GetSecret retrieves a secret by path
func (m *LocalManager) GetSecret(ctx context.Context, path string) (*Secret, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	secret, exists := m.secrets[path]
	if !exists {
		return nil, fmt.Errorf("secret not found: %s", path)
	}

	return secret, nil
}

// PutSecret stores a secret at the given path
func (m *LocalManager) PutSecret(ctx context.Context, path string, data map[string]string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	now := time.Now()
	secret := &Secret{
		Path:      path,
		Data:      data,
		Version:   1,
		CreatedAt: now,
		UpdatedAt: now,
	}

	// Increment version if updating
	if existing, exists := m.secrets[path]; exists {
		secret.Version = existing.Version + 1
		secret.CreatedAt = existing.CreatedAt
	}

	m.secrets[path] = secret
	return m.save()
}

// DeleteSecret removes a secret
func (m *LocalManager) DeleteSecret(ctx context.Context, path string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, exists := m.secrets[path]; !exists {
		return fmt.Errorf("secret not found: %s", path)
	}

	delete(m.secrets, path)
	return m.save()
}

// ListSecrets lists all secret paths
func (m *LocalManager) ListSecrets(ctx context.Context, prefix string) ([]string, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	paths := make([]string, 0)
	for path := range m.secrets {
		if prefix == "" || strings.HasPrefix(path, prefix) {
			paths = append(paths, path)
		}
	}

	return paths, nil
}

// Close closes the manager
func (m *LocalManager) Close() error {
	return nil
}

// load reads and decrypts secrets from disk
func (m *LocalManager) load() error {
	data, err := os.ReadFile(m.filePath)
	if err != nil {
		return err
	}

	// Decrypt
	decrypted, err := m.decrypt(data)
	if err != nil {
		return fmt.Errorf("failed to decrypt secrets: %w", err)
	}

	// Unmarshal
	var secrets map[string]*Secret
	if err := json.Unmarshal(decrypted, &secrets); err != nil {
		return fmt.Errorf("failed to unmarshal secrets: %w", err)
	}

	m.secrets = secrets
	return nil
}

// save encrypts and writes secrets to disk
func (m *LocalManager) save() error {
	// Marshal
	data, err := json.MarshalIndent(m.secrets, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal secrets: %w", err)
	}

	// Encrypt
	encrypted, err := m.encrypt(data)
	if err != nil {
		return fmt.Errorf("failed to encrypt secrets: %w", err)
	}

	// Write
	if err := os.WriteFile(m.filePath, encrypted, 0600); err != nil {
		return fmt.Errorf("failed to write secrets: %w", err)
	}

	return nil
}

// encrypt encrypts data using AES-GCM
func (m *LocalManager) encrypt(plaintext []byte) ([]byte, error) {
	nonce := make([]byte, m.gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, err
	}

	ciphertext := m.gcm.Seal(nonce, nonce, plaintext, nil)
	return ciphertext, nil
}

// decrypt decrypts data using AES-GCM
func (m *LocalManager) decrypt(ciphertext []byte) ([]byte, error) {
	nonceSize := m.gcm.NonceSize()
	if len(ciphertext) < nonceSize {
		return nil, fmt.Errorf("ciphertext too short")
	}

	nonce, ciphertext := ciphertext[:nonceSize], ciphertext[nonceSize:]
	plaintext, err := m.gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return nil, err
	}

	return plaintext, nil
}

// loadOrGenerateKey loads or generates an encryption key
func loadOrGenerateKey(keyFile string) (string, error) {
	// Try to load existing key
	if data, err := os.ReadFile(keyFile); err == nil {
		return string(data), nil
	}

	// Generate new key
	key := make([]byte, 32)
	if _, err := rand.Read(key); err != nil {
		return "", err
	}

	keyStr := fmt.Sprintf("%x", key)

	// Save key
	if err := os.WriteFile(keyFile, []byte(keyStr), 0600); err != nil {
		return "", err
	}

	fmt.Printf("Generated new secrets encryption key: %s\n", keyFile)
	return keyStr, nil
}
