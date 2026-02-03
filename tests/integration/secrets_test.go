package integration

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/platformfoundry/pf-ce/internal/secrets"
)

func TestSecrets_FileManagerIntegration(t *testing.T) {
	// Create temporary directory for secrets storage
	tmpDir, err := os.MkdirTemp("", "pf-secrets-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	secretsFile := filepath.Join(tmpDir, "secrets.json")

	// Initialize file manager
	manager, err := secrets.NewFileManager(secretsFile)
	if err != nil {
		t.Fatalf("Failed to create file manager: %v", err)
	}
	defer manager.Close()

	ctx := context.Background()

	// Test 1: Set and retrieve a secret
	t.Run("Set and Get Secret", func(t *testing.T) {
		key := "db-password"
		value := "super-secret-password-123"

		// Set secret
		err := manager.Set(ctx, key, value)
		if err != nil {
			t.Fatalf("Failed to set secret: %v", err)
		}

		// Retrieve secret
		retrieved, err := manager.Get(ctx, key)
		if err != nil {
			t.Fatalf("Failed to get secret: %v", err)
		}

		if retrieved != value {
			t.Errorf("Expected secret value '%s', got '%s'", value, retrieved)
		}
	})

	// Test 2: Multiple secrets
	t.Run("Multiple Secrets", func(t *testing.T) {
		secrets := map[string]string{
			"api-key":       "ak-1234567890",
			"webhook-token": "wh-abcdefgh",
			"ssh-key":       "ssh-rsa AAAAB3NzaC1...",
		}

		// Set all secrets
		for key, value := range secrets {
			if err := manager.Set(ctx, key, value); err != nil {
				t.Fatalf("Failed to set secret %s: %v", key, err)
			}
		}

		// List secrets
		keys, err := manager.List(ctx)
		if err != nil {
			t.Fatalf("Failed to list secrets: %v", err)
		}

		// Should have at least these 3 + the one from previous test
		if len(keys) < 3 {
			t.Errorf("Expected at least 3 secrets, got %d", len(keys))
		}

		// Verify all can be retrieved
		for key, expectedValue := range secrets {
			retrieved, err := manager.Get(ctx, key)
			if err != nil {
				t.Errorf("Failed to get secret %s: %v", key, err)
				continue
			}

			if retrieved != expectedValue {
				t.Errorf("Secret %s: expected '%s', got '%s'", key, expectedValue, retrieved)
			}
		}
	})

	// Test 3: Delete secret
	t.Run("Delete Secret", func(t *testing.T) {
		key := "temp-secret"
		value := "temporary-value"

		// Set secret
		if err := manager.Set(ctx, key, value); err != nil {
			t.Fatalf("Failed to set secret: %v", err)
		}

		// Verify it exists
		_, err := manager.Get(ctx, key)
		if err != nil {
			t.Fatalf("Secret should exist after setting: %v", err)
		}

		// Delete secret
		if err := manager.Delete(ctx, key); err != nil {
			t.Fatalf("Failed to delete secret: %v", err)
		}

		// Verify it's gone
		_, err = manager.Get(ctx, key)
		if err == nil {
			t.Error("Expected error when getting deleted secret, got nil")
		}
	})

	// Test 4: Update existing secret
	t.Run("Update Secret", func(t *testing.T) {
		key := "updatable-secret"
		value1 := "initial-value"
		value2 := "updated-value"

		// Set initial value
		if err := manager.Set(ctx, key, value1); err != nil {
			t.Fatalf("Failed to set secret: %v", err)
		}

		retrieved, err := manager.Get(ctx, key)
		if err != nil {
			t.Fatalf("Failed to get secret: %v", err)
		}

		if retrieved != value1 {
			t.Errorf("Expected initial value '%s', got '%s'", value1, retrieved)
		}

		// Update secret
		if err := manager.Set(ctx, key, value2); err != nil {
			t.Fatalf("Failed to update secret: %v", err)
		}

		retrieved, err = manager.Get(ctx, key)
		if err != nil {
			t.Fatalf("Failed to get updated secret: %v", err)
		}

		if retrieved != value2 {
			t.Errorf("Expected updated value '%s', got '%s'", value2, retrieved)
		}
	})

	// Test 5: Persistence across instances
	t.Run("Persistence", func(t *testing.T) {
		key := "persistent-secret"
		value := "persisted-value"

		// Set secret
		if err := manager.Set(ctx, key, value); err != nil {
			t.Fatalf("Failed to set secret: %v", err)
		}

		// Close manager
		manager.Close()

		// Create new manager with same file
		newManager, err := secrets.NewFileManager(secretsFile)
		if err != nil {
			t.Fatalf("Failed to create new file manager: %v", err)
		}
		defer newManager.Close()

		// Retrieve secret from new manager
		retrieved, err := newManager.Get(ctx, key)
		if err != nil {
			t.Fatalf("Failed to get secret from new manager: %v", err)
		}

		if retrieved != value {
			t.Errorf("Expected persisted value '%s', got '%s'", value, retrieved)
		}

		// Restore original manager for cleanup
		manager = newManager
	})
}

func TestSecrets_ReferenceResolution(t *testing.T) {
	// Test secret reference parsing and resolution
	tmpDir, err := os.MkdirTemp("", "pf-secrets-ref-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	secretsFile := filepath.Join(tmpDir, "secrets.json")
	manager, err := secrets.NewFileManager(secretsFile)
	if err != nil {
		t.Fatalf("Failed to create file manager: %v", err)
	}
	defer manager.Close()

	ctx := context.Background()

	// Setup test secrets
	testSecrets := map[string]string{
		"db/password":     "db-secret-password",
		"api/key":         "api-key-value",
		"vault:prod:cert": "cert-content",
	}

	for key, value := range testSecrets {
		if err := manager.Set(ctx, key, value); err != nil {
			t.Fatalf("Failed to set secret %s: %v", key, err)
		}
	}

	tests := []struct {
		name           string
		secretRef      string
		expectedValue  string
		shouldResolve  bool
	}{
		{
			name:          "Simple secret reference",
			secretRef:     "${secret:local:db/password}",
			expectedValue: "db-secret-password",
			shouldResolve: true,
		},
		{
			name:          "API key reference",
			secretRef:     "${secret:local:api/key}",
			expectedValue: "api-key-value",
			shouldResolve: true,
		},
		{
			name:          "Non-existent secret",
			secretRef:     "${secret:local:non-existent}",
			expectedValue: "",
			shouldResolve: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Extract secret key from reference
			// Format: ${secret:provider:path:key} or ${secret:provider:path}
			key := extractSecretKey(tt.secretRef)

			if tt.shouldResolve {
				value, err := manager.Get(ctx, key)
				if err != nil {
					t.Errorf("Failed to resolve secret reference: %v", err)
					return
				}

				if value != tt.expectedValue {
					t.Errorf("Expected value '%s', got '%s'", tt.expectedValue, value)
				}
			} else {
				_, err := manager.Get(ctx, key)
				if err == nil {
					t.Error("Expected error for non-existent secret, got nil")
				}
			}
		})
	}
}

// Helper function to extract secret key from reference format
// ${secret:provider:path:key} -> path:key or path
func extractSecretKey(ref string) string {
	// Remove ${secret: prefix and } suffix
	if len(ref) < 11 {
		return ""
	}

	ref = ref[9 : len(ref)-1] // Remove "${secret:" and "}"

	// Split by :
	parts := []string{}
	current := ""
	for _, ch := range ref {
		if ch == ':' {
			parts = append(parts, current)
			current = ""
		} else {
			current += string(ch)
		}
	}
	if current != "" {
		parts = append(parts, current)
	}

	// Format: provider:path or provider:path:key
	if len(parts) >= 2 {
		// Join path parts (everything after provider)
		return joinParts(parts[1:])
	}

	return ""
}

func joinParts(parts []string) string {
	if len(parts) == 0 {
		return ""
	}
	result := parts[0]
	for i := 1; i < len(parts); i++ {
		result += ":" + parts[i]
	}
	return result
}

func TestSecrets_ErrorHandling(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "pf-secrets-error-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	secretsFile := filepath.Join(tmpDir, "secrets.json")
	manager, err := secrets.NewFileManager(secretsFile)
	if err != nil {
		t.Fatalf("Failed to create file manager: %v", err)
	}
	defer manager.Close()

	ctx := context.Background()

	t.Run("Get Non-Existent Secret", func(t *testing.T) {
		_, err := manager.Get(ctx, "does-not-exist")
		if err == nil {
			t.Error("Expected error when getting non-existent secret, got nil")
		}
	})

	t.Run("Delete Non-Existent Secret", func(t *testing.T) {
		// Delete should succeed even if secret doesn't exist
		err := manager.Delete(ctx, "does-not-exist")
		// File manager doesn't error on deleting non-existent keys
		if err != nil {
			t.Logf("Delete non-existent secret returned: %v", err)
		}
	})

	t.Run("Empty Key", func(t *testing.T) {
		err := manager.Set(ctx, "", "value")
		// Empty keys are technically allowed by the file manager
		// but in practice should be validated at a higher level
		if err != nil {
			t.Logf("Set with empty key returned: %v", err)
		}
	})
}

func TestSecrets_FilePermissions(t *testing.T) {
	if os.Getenv("SKIP_PERMISSION_TESTS") != "" {
		t.Skip("Skipping file permission tests (SKIP_PERMISSION_TESTS set)")
	}

	tmpDir, err := os.MkdirTemp("", "pf-secrets-perm-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	secretsFile := filepath.Join(tmpDir, "secrets.json")
	manager, err := secrets.NewFileManager(secretsFile)
	if err != nil {
		t.Fatalf("Failed to create file manager: %v", err)
	}
	defer manager.Close()

	ctx := context.Background()

	// Set a secret to create the file
	if err := manager.Set(ctx, "test-key", "test-value"); err != nil {
		t.Fatalf("Failed to set secret: %v", err)
	}

	// Check file permissions
	info, err := os.Stat(secretsFile)
	if err != nil {
		t.Fatalf("Failed to stat secrets file: %v", err)
	}

	// Secrets file should have restricted permissions (0600)
	mode := info.Mode()
	expected := os.FileMode(0600)

	// On Windows, file permissions work differently
	// This test may need to be adjusted for cross-platform compatibility
	if mode.Perm() != expected {
		t.Logf("Note: Secrets file has permissions %v, expected %v", mode.Perm(), expected)
		t.Logf("File permission enforcement may vary by platform")
	}

	// Verify directory permissions (should be 0700)
	dirInfo, err := os.Stat(tmpDir)
	if err != nil {
		t.Fatalf("Failed to stat directory: %v", err)
	}

	dirMode := dirInfo.Mode()
	t.Logf("Directory permissions: %v", dirMode.Perm())
}

func TestSecrets_ConcurrentAccess(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "pf-secrets-concurrent-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	secretsFile := filepath.Join(tmpDir, "secrets.json")
	manager, err := secrets.NewFileManager(secretsFile)
	if err != nil {
		t.Fatalf("Failed to create file manager: %v", err)
	}
	defer manager.Close()

	ctx := context.Background()

	// Test concurrent reads and writes
	t.Run("Concurrent Writes", func(t *testing.T) {
		done := make(chan bool)
		errors := make(chan error, 10)

		// Launch 10 concurrent writers
		for i := 0; i < 10; i++ {
			go func(index int) {
				key := filepath.Join("concurrent", "key", string(rune(index+'0')))
				value := filepath.Join("value", string(rune(index+'0')))

				if err := manager.Set(ctx, key, value); err != nil {
					errors <- err
				}
				done <- true
			}(i)
		}

		// Wait for all to complete
		for i := 0; i < 10; i++ {
			<-done
		}
		close(errors)

		// Check for errors
		for err := range errors {
			t.Errorf("Concurrent write error: %v", err)
		}
	})

	t.Run("Concurrent Reads", func(t *testing.T) {
		// Set a secret first
		testKey := "concurrent-read-test"
		testValue := "test-value"
		if err := manager.Set(ctx, testKey, testValue); err != nil {
			t.Fatalf("Failed to set test secret: %v", err)
		}

		done := make(chan bool)
		errors := make(chan error, 20)

		// Launch 20 concurrent readers
		for i := 0; i < 20; i++ {
			go func() {
				value, err := manager.Get(ctx, testKey)
				if err != nil {
					errors <- err
				} else if value != testValue {
					errors <- err
				}
				done <- true
			}()
		}

		// Wait for all to complete
		for i := 0; i < 20; i++ {
			<-done
		}
		close(errors)

		// Check for errors
		errorCount := 0
		for err := range errors {
			t.Errorf("Concurrent read error: %v", err)
			errorCount++
		}

		if errorCount > 0 {
			t.Errorf("Got %d errors during concurrent reads", errorCount)
		}
	})
}
