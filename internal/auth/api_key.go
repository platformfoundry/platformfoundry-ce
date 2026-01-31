package auth

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/platformfoundry/platformfoundry-ce/pkg/configloader"
)

// APIKey represents an API key for programmatic access
type APIKey struct {
	ID           string     `json:"id"`
	Key          string     `json:"key"` // Hashed
	Name         string     `json:"name"`
	Username     string     `json:"username"`
	Organization string     `json:"organization,omitempty"`
	Roles        []string   `json:"roles"`
	CreatedAt    time.Time  `json:"createdAt"`
	ExpiresAt    *time.Time `json:"expiresAt,omitempty"`
	LastUsedAt   *time.Time `json:"lastUsedAt,omitempty"`
	Enabled      bool       `json:"enabled"`
}

// APIKeyStore manages API keys
type APIKeyStore struct {
	filePath string
	keys     map[string]*APIKey // keyed by ID
}

// NewAPIKeyStore creates a new API key store
func NewAPIKeyStore(configDir string) (*APIKeyStore, error) {
	var filePath string

	if configDir != "" {
		// Use provided config directory
		if err := os.MkdirAll(configDir, 0700); err != nil {
			return nil, fmt.Errorf("failed to create config directory: %w", err)
		}
		filePath = filepath.Join(configDir, "api_keys.json")
	} else {
		// Use centralized config loader
		loader := configloader.Global()
		if _, err := loader.ConfigDir(); err != nil {
			return nil, err
		}
		filePath = loader.FilePath(configloader.ConfigFileAPIKeys)
	}

	store := &APIKeyStore{
		filePath: filePath,
		keys:     make(map[string]*APIKey),
	}

	// Load existing keys
	if err := store.load(); err != nil {
		if !os.IsNotExist(err) {
			return nil, err
		}
	}

	return store, nil
}

// CreateAPIKey creates a new API key
func (s *APIKeyStore) CreateAPIKey(name, username string, roles []string, organization string, expiresIn time.Duration) (string, *APIKey, error) {
	// Generate random API key
	keyBytes := make([]byte, 32)
	if _, err := rand.Read(keyBytes); err != nil {
		return "", nil, fmt.Errorf("failed to generate random key: %w", err)
	}

	// Format: pf_<base64-encoded-key>
	rawKey := "pf_" + base64.URLEncoding.EncodeToString(keyBytes)

	// Hash the key for storage (same as password)
	hashedKey, err := hashAPIKey(rawKey)
	if err != nil {
		return "", nil, err
	}

	// Generate ID
	idBytes := make([]byte, 16)
	if _, err := rand.Read(idBytes); err != nil {
		return "", nil, fmt.Errorf("failed to generate ID: %w", err)
	}
	id := base64.URLEncoding.EncodeToString(idBytes)

	now := time.Now()
	var expiresAt *time.Time
	if expiresIn > 0 {
		expires := now.Add(expiresIn)
		expiresAt = &expires
	}

	apiKey := &APIKey{
		ID:           id,
		Key:          hashedKey,
		Name:         name,
		Username:     username,
		Organization: organization,
		Roles:        roles,
		CreatedAt:    now,
		ExpiresAt:    expiresAt,
		Enabled:      true,
	}

	s.keys[id] = apiKey
	if err := s.save(); err != nil {
		return "", nil, err
	}

	// Return the raw key (only time it's available)
	return rawKey, apiKey, nil
}

// ValidateAPIKey validates an API key and returns the associated key info
func (s *APIKeyStore) ValidateAPIKey(rawKey string) (*APIKey, error) {
	// Find matching key by comparing hashes
	for _, apiKey := range s.keys {
		if !apiKey.Enabled {
			continue
		}

		// Check expiration
		if apiKey.ExpiresAt != nil && time.Now().After(*apiKey.ExpiresAt) {
			continue
		}

		// Verify key
		if err := verifyAPIKey(rawKey, apiKey.Key); err == nil {
			// Update last used
			now := time.Now()
			apiKey.LastUsedAt = &now
			s.save() // Save asynchronously
			return apiKey, nil
		}
	}

	return nil, fmt.Errorf("invalid API key")
}

// GetAPIKey retrieves an API key by ID
func (s *APIKeyStore) GetAPIKey(id string) (*APIKey, error) {
	key, exists := s.keys[id]
	if !exists {
		return nil, fmt.Errorf("API key %s not found", id)
	}
	return key, nil
}

// ListAPIKeys returns all API keys for a user
func (s *APIKeyStore) ListAPIKeys(username string) []*APIKey {
	keys := make([]*APIKey, 0)
	for _, key := range s.keys {
		if key.Username == username {
			keys = append(keys, key)
		}
	}
	return keys
}

// RevokeAPIKey disables an API key
func (s *APIKeyStore) RevokeAPIKey(id string) error {
	key, exists := s.keys[id]
	if !exists {
		return fmt.Errorf("API key %s not found", id)
	}

	key.Enabled = false
	return s.save()
}

// DeleteAPIKey removes an API key
func (s *APIKeyStore) DeleteAPIKey(id string) error {
	if _, exists := s.keys[id]; !exists {
		return fmt.Errorf("API key %s not found", id)
	}

	delete(s.keys, id)
	return s.save()
}

// load reads API keys from disk
func (s *APIKeyStore) load() error {
	data, err := os.ReadFile(s.filePath)
	if err != nil {
		return err
	}

	var keys map[string]*APIKey
	if err := json.Unmarshal(data, &keys); err != nil {
		return fmt.Errorf("failed to unmarshal API keys: %w", err)
	}

	s.keys = keys
	return nil
}

// save writes API keys to disk
func (s *APIKeyStore) save() error {
	data, err := json.MarshalIndent(s.keys, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal API keys: %w", err)
	}

	if err := os.WriteFile(s.filePath, data, 0600); err != nil {
		return fmt.Errorf("failed to write API keys file: %w", err)
	}

	return nil
}

// hashAPIKey hashes an API key using bcrypt
func hashAPIKey(key string) (string, error) {
	// Use a lighter cost for API keys since they're already random
	hashedKey, err := bcryptHash([]byte(key), 10)
	if err != nil {
		return "", fmt.Errorf("failed to hash API key: %w", err)
	}
	return string(hashedKey), nil
}

// verifyAPIKey verifies an API key against its hash
func verifyAPIKey(rawKey, hashedKey string) error {
	return bcryptCompare([]byte(hashedKey), []byte(rawKey))
}
