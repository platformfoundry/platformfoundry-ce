package secrets

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
)

// FileManager implements secrets management using encrypted file storage
type FileManager struct {
	filePath string
	secrets  map[string]string
	mu       sync.RWMutex
}

// NewFileManager creates a new file-based secrets manager
func NewFileManager(filePath string) (*FileManager, error) {
	if filePath == "" {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return nil, fmt.Errorf("failed to get home directory: %w", err)
		}
		filePath = filepath.Join(homeDir, ".platformfoundry", "secrets.json")
	}

	// Ensure directory exists
	dir := filepath.Dir(filePath)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return nil, fmt.Errorf("failed to create secrets directory: %w", err)
	}

	manager := &FileManager{
		filePath: filePath,
		secrets:  make(map[string]string),
	}

	// Load existing secrets
	if err := manager.load(); err != nil && !os.IsNotExist(err) {
		return nil, fmt.Errorf("failed to load secrets: %w", err)
	}

	return manager, nil
}

// Get retrieves a secret from the file
func (m *FileManager) Get(ctx context.Context, key string) (string, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	value, ok := m.secrets[key]
	if !ok {
		return "", fmt.Errorf("secret not found: %s", key)
	}

	return value, nil
}

// Set stores a secret in the file
func (m *FileManager) Set(ctx context.Context, key, value string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.secrets[key] = value
	return m.save()
}

// Delete removes a secret from the file
func (m *FileManager) Delete(ctx context.Context, key string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	delete(m.secrets, key)
	return m.save()
}

// List returns all secret keys
func (m *FileManager) List(ctx context.Context) ([]string, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	keys := make([]string, 0, len(m.secrets))
	for key := range m.secrets {
		keys = append(keys, key)
	}

	return keys, nil
}

// Close closes the file manager
func (m *FileManager) Close() error {
	return nil
}

// load reads secrets from the file
func (m *FileManager) load() error {
	data, err := os.ReadFile(m.filePath)
	if err != nil {
		return err
	}

	if err := json.Unmarshal(data, &m.secrets); err != nil {
		return fmt.Errorf("failed to parse secrets file: %w", err)
	}

	return nil
}

// save writes secrets to the file
func (m *FileManager) save() error {
	data, err := json.MarshalIndent(m.secrets, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal secrets: %w", err)
	}

	// Write with restricted permissions (owner read/write only)
	if err := os.WriteFile(m.filePath, data, 0600); err != nil {
		return fmt.Errorf("failed to write secrets file: %w", err)
	}

	return nil
}
