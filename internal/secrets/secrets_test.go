package secrets

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewManager(t *testing.T) {
	tests := []struct {
		name    string
		config  *Config
		wantErr bool
		errMsg  string
	}{
		{
			name:    "nil config returns error",
			config:  nil,
			wantErr: true,
			errMsg:  "configuration is required",
		},
		{
			name:    "unsupported provider returns error",
			config:  &Config{Provider: "unsupported"},
			wantErr: true,
			errMsg:  "unsupported secrets provider",
		},
		{
			name:    "vault without config returns error",
			config:  &Config{Provider: "vault"},
			wantErr: true,
			errMsg:  "vault configuration is required",
		},
		{
			name:    "aws without config returns error",
			config:  &Config{Provider: "aws"},
			wantErr: true,
			errMsg:  "AWS configuration is required",
		},
		{
			name: "local with default config succeeds",
			config: &Config{
				Provider: "local",
				Local:    &LocalConfig{EncryptionKey: "test-key-12345"},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.config != nil && tt.config.Provider == "local" {
				// Create temp directory for local provider
				tempDir, err := os.MkdirTemp("", "secrets-test-*")
				require.NoError(t, err)
				defer os.RemoveAll(tempDir)
				tt.config.Local.Path = filepath.Join(tempDir, "secrets.enc")
			}

			manager, err := NewManager(tt.config)

			if tt.wantErr {
				assert.Error(t, err)
				if tt.errMsg != "" {
					assert.Contains(t, err.Error(), tt.errMsg)
				}
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, manager)
				if manager != nil {
					manager.Close()
				}
			}
		})
	}
}

func TestParseSecretReference(t *testing.T) {
	tests := []struct {
		name      string
		ref       string
		wantErr   bool
		provider  string
		path      string
		key       string
	}{
		{
			name:     "valid reference with key",
			ref:      "${secret:vault:database/prod:password}",
			wantErr:  false,
			provider: "vault",
			path:     "database/prod",
			key:      "password",
		},
		{
			name:     "valid reference without key uses default",
			ref:      "${secret:aws:prod/db}",
			wantErr:  false,
			provider: "aws",
			path:     "prod/db",
			key:      "value",
		},
		{
			name:     "valid local reference",
			ref:      "${secret:local:api:token}",
			wantErr:  false,
			provider: "local",
			path:     "api",
			key:      "token",
		},
		{
			name:    "invalid format - no prefix",
			ref:     "secret:vault:path:key",
			wantErr: true,
		},
		{
			name:    "invalid format - incomplete",
			ref:     "${secret:vault}",
			wantErr: true,
		},
		{
			name:    "invalid format - random string",
			ref:     "just-a-string",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ref, err := ParseSecretReference(tt.ref)

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, ref)
				assert.Equal(t, tt.provider, ref.Provider)
				assert.Equal(t, tt.path, ref.Path)
				assert.Equal(t, tt.key, ref.Key)
				assert.Equal(t, tt.ref, ref.Raw)
			}
		})
	}
}

func TestFindSecretReferences(t *testing.T) {
	tests := []struct {
		name     string
		text     string
		expected int
		paths    []string
	}{
		{
			name:     "no references",
			text:     "just a plain string",
			expected: 0,
		},
		{
			name:     "single reference",
			text:     "password: ${secret:vault:db:pass}",
			expected: 1,
			paths:    []string{"db"},
		},
		{
			name:     "multiple references",
			text:     "user: ${secret:aws:app:user} pass: ${secret:vault:db:pass}",
			expected: 2,
			paths:    []string{"app", "db"},
		},
		{
			name:     "reference in multiline text",
			text:     "config:\n  api_key: ${secret:local:keys:api}\n  other: value",
			expected: 1,
			paths:    []string{"keys"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			refs := FindSecretReferences(tt.text)
			assert.Len(t, refs, tt.expected)

			if tt.paths != nil {
				for i, ref := range refs {
					assert.Equal(t, tt.paths[i], ref.Path)
				}
			}
		})
	}
}

func TestLocalManager(t *testing.T) {
	// Create temp directory
	tempDir, err := os.MkdirTemp("", "local-manager-test-*")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	config := &LocalConfig{
		Path:          filepath.Join(tempDir, "secrets.enc"),
		EncryptionKey: "test-encryption-key-32chars!!!!",
	}

	manager, err := NewLocalManager(config)
	require.NoError(t, err)
	defer manager.Close()

	ctx := context.Background()

	t.Run("PutSecret and GetSecret", func(t *testing.T) {
		path := "test/secret"
		data := map[string]string{
			"username": "admin",
			"password": "secret123",
		}

		// Put secret
		err := manager.PutSecret(ctx, path, data)
		require.NoError(t, err)

		// Get secret
		secret, err := manager.GetSecret(ctx, path)
		require.NoError(t, err)
		assert.Equal(t, path, secret.Path)
		assert.Equal(t, "admin", secret.Data["username"])
		assert.Equal(t, "secret123", secret.Data["password"])
		assert.Equal(t, 1, secret.Version)
	})

	t.Run("Update increments version", func(t *testing.T) {
		path := "version/test"
		data := map[string]string{"key": "value1"}

		err := manager.PutSecret(ctx, path, data)
		require.NoError(t, err)

		secret1, _ := manager.GetSecret(ctx, path)
		assert.Equal(t, 1, secret1.Version)

		// Update
		data["key"] = "value2"
		err = manager.PutSecret(ctx, path, data)
		require.NoError(t, err)

		secret2, _ := manager.GetSecret(ctx, path)
		assert.Equal(t, 2, secret2.Version)
		assert.Equal(t, "value2", secret2.Data["key"])
	})

	t.Run("GetSecret returns error for non-existent", func(t *testing.T) {
		_, err := manager.GetSecret(ctx, "non/existent")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "not found")
	})

	t.Run("ListSecrets", func(t *testing.T) {
		// Add some secrets
		_ = manager.PutSecret(ctx, "app/config", map[string]string{"key": "val"})
		_ = manager.PutSecret(ctx, "app/credentials", map[string]string{"key": "val"})
		_ = manager.PutSecret(ctx, "db/password", map[string]string{"key": "val"})

		// List all
		paths, err := manager.ListSecrets(ctx, "")
		require.NoError(t, err)
		assert.GreaterOrEqual(t, len(paths), 3)

		// List with prefix
		paths, err = manager.ListSecrets(ctx, "app/")
		require.NoError(t, err)
		assert.Equal(t, 2, len(paths))
	})

	t.Run("DeleteSecret", func(t *testing.T) {
		path := "delete/test"
		_ = manager.PutSecret(ctx, path, map[string]string{"key": "val"})

		// Verify exists
		_, err := manager.GetSecret(ctx, path)
		require.NoError(t, err)

		// Delete
		err = manager.DeleteSecret(ctx, path)
		require.NoError(t, err)

		// Verify deleted
		_, err = manager.GetSecret(ctx, path)
		assert.Error(t, err)
	})

	t.Run("DeleteSecret returns error for non-existent", func(t *testing.T) {
		err := manager.DeleteSecret(ctx, "non/existent/secret")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "not found")
	})
}

func TestLocalManager_Persistence(t *testing.T) {
	// Create temp directory
	tempDir, err := os.MkdirTemp("", "local-persist-test-*")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	secretsPath := filepath.Join(tempDir, "secrets.enc")
	encryptionKey := "persistence-test-key-32chars!!!"

	// Create first manager and add secret
	manager1, err := NewLocalManager(&LocalConfig{
		Path:          secretsPath,
		EncryptionKey: encryptionKey,
	})
	require.NoError(t, err)

	ctx := context.Background()
	err = manager1.PutSecret(ctx, "persist/test", map[string]string{"key": "persisted-value"})
	require.NoError(t, err)
	manager1.Close()

	// Create second manager and verify secret persisted
	manager2, err := NewLocalManager(&LocalConfig{
		Path:          secretsPath,
		EncryptionKey: encryptionKey,
	})
	require.NoError(t, err)
	defer manager2.Close()

	secret, err := manager2.GetSecret(ctx, "persist/test")
	require.NoError(t, err)
	assert.Equal(t, "persisted-value", secret.Data["key"])
}

func TestResolveSecretReferences(t *testing.T) {
	// Create temp directory and manager
	tempDir, err := os.MkdirTemp("", "resolve-test-*")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	manager, err := NewLocalManager(&LocalConfig{
		Path:          filepath.Join(tempDir, "secrets.enc"),
		EncryptionKey: "resolve-test-key-32chars!!!!!!!",
	})
	require.NoError(t, err)
	defer manager.Close()

	ctx := context.Background()

	// Setup secrets
	_ = manager.PutSecret(ctx, "db", map[string]string{
		"username": "dbuser",
		"password": "dbpass123",
	})

	t.Run("resolve single reference", func(t *testing.T) {
		text := "DB_PASS=${secret:local:db:password}"
		resolved, err := ResolveSecretReferences(ctx, text, manager)
		require.NoError(t, err)
		assert.Equal(t, "DB_PASS=dbpass123", resolved)
	})

	t.Run("resolve multiple references", func(t *testing.T) {
		text := "user=${secret:local:db:username} pass=${secret:local:db:password}"
		resolved, err := ResolveSecretReferences(ctx, text, manager)
		require.NoError(t, err)
		assert.Equal(t, "user=dbuser pass=dbpass123", resolved)
	})

	t.Run("no references returns unchanged", func(t *testing.T) {
		text := "no references here"
		resolved, err := ResolveSecretReferences(ctx, text, manager)
		require.NoError(t, err)
		assert.Equal(t, text, resolved)
	})

	t.Run("missing secret returns error", func(t *testing.T) {
		text := "${secret:local:nonexistent:key}"
		_, err := ResolveSecretReferences(ctx, text, manager)
		assert.Error(t, err)
	})

	t.Run("missing key returns error", func(t *testing.T) {
		text := "${secret:local:db:nonexistent}"
		_, err := ResolveSecretReferences(ctx, text, manager)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "does not contain key")
	})
}

func TestResolveSecretReferencesInMap(t *testing.T) {
	// Create temp directory and manager
	tempDir, err := os.MkdirTemp("", "resolve-map-test-*")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	manager, err := NewLocalManager(&LocalConfig{
		Path:          filepath.Join(tempDir, "secrets.enc"),
		EncryptionKey: "resolve-map-key-32chars!!!!!!!!!",
	})
	require.NoError(t, err)
	defer manager.Close()

	ctx := context.Background()

	// Setup secrets
	_ = manager.PutSecret(ctx, "api", map[string]string{"key": "api-key-123"})

	t.Run("resolve in flat map", func(t *testing.T) {
		data := map[string]interface{}{
			"apiKey": "${secret:local:api:key}",
			"normal": "plain-value",
		}
		err := ResolveSecretReferencesInMap(ctx, data, manager)
		require.NoError(t, err)
		assert.Equal(t, "api-key-123", data["apiKey"])
		assert.Equal(t, "plain-value", data["normal"])
	})

	t.Run("resolve in nested map", func(t *testing.T) {
		data := map[string]interface{}{
			"config": map[string]interface{}{
				"apiKey": "${secret:local:api:key}",
			},
		}
		err := ResolveSecretReferencesInMap(ctx, data, manager)
		require.NoError(t, err)
		config := data["config"].(map[string]interface{})
		assert.Equal(t, "api-key-123", config["apiKey"])
	})

	t.Run("resolve in array", func(t *testing.T) {
		data := map[string]interface{}{
			"keys": []interface{}{"${secret:local:api:key}", "plain"},
		}
		err := ResolveSecretReferencesInMap(ctx, data, manager)
		require.NoError(t, err)
		keys := data["keys"].([]interface{})
		assert.Equal(t, "api-key-123", keys[0])
		assert.Equal(t, "plain", keys[1])
	})
}
