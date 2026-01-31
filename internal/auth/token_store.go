package auth

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/platformfoundry/platformfoundry-ce/pkg/configloader"
)

// TokenStore manages authentication tokens
type TokenStore struct {
	filePath string
}

// TokenInfo represents stored token information
type TokenInfo struct {
	Token     string    `json:"token"`
	Username  string    `json:"username"`
	ExpiresAt time.Time `json:"expiresAt"`
	IssuedAt  time.Time `json:"issuedAt"`
}

// NewTokenStore creates a new token store
func NewTokenStore(configDir string) (*TokenStore, error) {
	var filePath string

	if configDir != "" {
		// Use provided config directory
		if err := os.MkdirAll(configDir, 0700); err != nil {
			return nil, fmt.Errorf("failed to create config directory: %w", err)
		}
		filePath = filepath.Join(configDir, "credentials")
	} else {
		// Use centralized config loader
		loader := configloader.Global()
		if _, err := loader.ConfigDir(); err != nil {
			return nil, err
		}
		filePath = loader.FilePath(configloader.ConfigFileCreds)
	}

	return &TokenStore{
		filePath: filePath,
	}, nil
}

// SaveToken stores an authentication token
func (s *TokenStore) SaveToken(token string, username string, expiresAt time.Time) error {
	info := TokenInfo{
		Token:     token,
		Username:  username,
		ExpiresAt: expiresAt,
		IssuedAt:  time.Now(),
	}

	data, err := json.MarshalIndent(info, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal token info: %w", err)
	}

	// Write with restricted permissions (owner only)
	if err := os.WriteFile(s.filePath, data, 0600); err != nil {
		return fmt.Errorf("failed to write credentials file: %w", err)
	}

	return nil
}

// GetToken retrieves the stored authentication token
func (s *TokenStore) GetToken() (*TokenInfo, error) {
	data, err := os.ReadFile(s.filePath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("not logged in: no credentials found")
		}
		return nil, fmt.Errorf("failed to read credentials: %w", err)
	}

	var info TokenInfo
	if err := json.Unmarshal(data, &info); err != nil {
		return nil, fmt.Errorf("failed to unmarshal token info: %w", err)
	}

	// Check if token is expired
	if time.Now().After(info.ExpiresAt) {
		return nil, fmt.Errorf("token expired: please login again")
	}

	return &info, nil
}

// ClearToken removes the stored authentication token
func (s *TokenStore) ClearToken() error {
	if err := os.Remove(s.filePath); err != nil {
		if os.IsNotExist(err) {
			return nil // Already cleared
		}
		return fmt.Errorf("failed to remove credentials: %w", err)
	}
	return nil
}

// HasValidToken checks if there's a valid token stored
func (s *TokenStore) HasValidToken() bool {
	info, err := s.GetToken()
	if err != nil {
		return false
	}
	return time.Now().Before(info.ExpiresAt)
}
