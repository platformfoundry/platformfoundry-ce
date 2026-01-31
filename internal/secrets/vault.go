package secrets

import (
	"context"
	"fmt"
	"os"
	"time"

	vault "github.com/hashicorp/vault/api"
)

// VaultManager implements Manager using HashiCorp Vault
type VaultManager struct {
	client    *vault.Client
	mount     string
	namespace string
}

// NewVaultManager creates a new Vault secrets manager
func NewVaultManager(config *VaultConfig) (*VaultManager, error) {
	// Create Vault client configuration
	vaultConfig := vault.DefaultConfig()
	vaultConfig.Address = config.Address

	client, err := vault.NewClient(vaultConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create Vault client: %w", err)
	}

	// Set token
	token := config.Token
	if token == "" && config.TokenFile != "" {
		tokenBytes, err := os.ReadFile(config.TokenFile)
		if err != nil {
			return nil, fmt.Errorf("failed to read token file: %w", err)
		}
		token = string(tokenBytes)
	}
	if token == "" {
		token = os.Getenv("VAULT_TOKEN")
	}
	if token == "" {
		return nil, fmt.Errorf("Vault token is required")
	}

	client.SetToken(token)

	// Set namespace if provided
	if config.Namespace != "" {
		client.SetNamespace(config.Namespace)
	}

	// Set mount path
	mount := config.Mount
	if mount == "" {
		mount = "secret"
	}

	return &VaultManager{
		client:    client,
		mount:     mount,
		namespace: config.Namespace,
	}, nil
}

// GetSecret retrieves a secret by path
func (m *VaultManager) GetSecret(ctx context.Context, path string) (*Secret, error) {
	fullPath := m.mount + "/data/" + path

	// Read secret from Vault
	vaultSecret, err := m.client.Logical().ReadWithContext(ctx, fullPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read secret from Vault: %w", err)
	}

	if vaultSecret == nil {
		return nil, fmt.Errorf("secret not found: %s", path)
	}

	// Extract data
	dataInterface, ok := vaultSecret.Data["data"]
	if !ok {
		return nil, fmt.Errorf("secret data not found in response")
	}

	dataMap, ok := dataInterface.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("invalid secret data format")
	}

	// Convert to string map
	data := make(map[string]string)
	for k, v := range dataMap {
		if strVal, ok := v.(string); ok {
			data[k] = strVal
		} else {
			data[k] = fmt.Sprintf("%v", v)
		}
	}

	// Extract metadata
	metadata := make(map[string]string)
	if metadataInterface, ok := vaultSecret.Data["metadata"]; ok {
		if metadataMap, ok := metadataInterface.(map[string]interface{}); ok {
			for k, v := range metadataMap {
				metadata[k] = fmt.Sprintf("%v", v)
			}
		}
	}

	// Extract version
	version := 0
	if versionInterface, ok := vaultSecret.Data["version"]; ok {
		if versionFloat, ok := versionInterface.(float64); ok {
			version = int(versionFloat)
		}
	}

	secret := &Secret{
		Path:     path,
		Data:     data,
		Metadata: metadata,
		Version:  version,
	}

	// Parse timestamps if available
	if createdTime, ok := metadata["created_time"]; ok {
		if t, err := time.Parse(time.RFC3339, createdTime); err == nil {
			secret.CreatedAt = t
		}
	}
	if updatedTime, ok := metadata["updated_time"]; ok {
		if t, err := time.Parse(time.RFC3339, updatedTime); err == nil {
			secret.UpdatedAt = t
		}
	}

	return secret, nil
}

// PutSecret stores a secret at the given path
func (m *VaultManager) PutSecret(ctx context.Context, path string, data map[string]string) error {
	fullPath := m.mount + "/data/" + path

	// Convert to interface{} map
	dataInterface := make(map[string]interface{})
	for k, v := range data {
		dataInterface[k] = v
	}

	// Write to Vault
	payload := map[string]interface{}{
		"data": dataInterface,
	}

	_, err := m.client.Logical().WriteWithContext(ctx, fullPath, payload)
	if err != nil {
		return fmt.Errorf("failed to write secret to Vault: %w", err)
	}

	return nil
}

// DeleteSecret removes a secret
func (m *VaultManager) DeleteSecret(ctx context.Context, path string) error {
	fullPath := m.mount + "/metadata/" + path

	_, err := m.client.Logical().DeleteWithContext(ctx, fullPath)
	if err != nil {
		return fmt.Errorf("failed to delete secret from Vault: %w", err)
	}

	return nil
}

// ListSecrets lists all secret paths
func (m *VaultManager) ListSecrets(ctx context.Context, prefix string) ([]string, error) {
	fullPath := m.mount + "/metadata/" + prefix

	vaultSecret, err := m.client.Logical().ListWithContext(ctx, fullPath)
	if err != nil {
		return nil, fmt.Errorf("failed to list secrets from Vault: %w", err)
	}

	if vaultSecret == nil {
		return []string{}, nil
	}

	// Extract keys
	keysInterface, ok := vaultSecret.Data["keys"]
	if !ok {
		return []string{}, nil
	}

	keysSlice, ok := keysInterface.([]interface{})
	if !ok {
		return nil, fmt.Errorf("invalid keys format in response")
	}

	paths := make([]string, 0, len(keysSlice))
	for _, keyInterface := range keysSlice {
		if key, ok := keyInterface.(string); ok {
			paths = append(paths, prefix+key)
		}
	}

	return paths, nil
}

// Close closes the Vault client
func (m *VaultManager) Close() error {
	// Vault client doesn't require explicit closing
	return nil
}
