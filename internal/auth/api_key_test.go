package auth

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestNewAPIKeyStore(t *testing.T) {
	tmpDir := t.TempDir()

	store, err := NewAPIKeyStore(tmpDir)
	if err != nil {
		t.Fatalf("NewAPIKeyStore failed: %v", err)
	}

	if store == nil {
		t.Fatal("NewAPIKeyStore returned nil")
	}

	expectedPath := filepath.Join(tmpDir, "api_keys.json")
	if store.filePath != expectedPath {
		t.Errorf("Expected filePath %s, got %s", expectedPath, store.filePath)
	}

	if store.keys == nil {
		t.Fatal("Keys map should be initialized")
	}
}

func TestCreateAPIKey(t *testing.T) {
	tmpDir := t.TempDir()
	store, err := NewAPIKeyStore(tmpDir)
	if err != nil {
		t.Fatalf("NewAPIKeyStore failed: %v", err)
	}

	name := "test-key"
	username := "testuser"
	roles := []string{"admin", "developer"}
	organization := "test-org"
	expiresIn := 30 * 24 * time.Hour // 30 days

	rawKey, apiKey, err := store.CreateAPIKey(name, username, roles, organization, expiresIn)
	if err != nil {
		t.Fatalf("CreateAPIKey failed: %v", err)
	}

	// Verify raw key format
	if !strings.HasPrefix(rawKey, "pf_") {
		t.Errorf("Expected raw key to start with 'pf_', got %s", rawKey)
	}

	if len(rawKey) < 20 {
		t.Error("Raw key seems too short")
	}

	// Verify API key object
	if apiKey.Name != name {
		t.Errorf("Expected name %s, got %s", name, apiKey.Name)
	}

	if apiKey.Username != username {
		t.Errorf("Expected username %s, got %s", username, apiKey.Username)
	}

	if len(apiKey.Roles) != len(roles) {
		t.Errorf("Expected %d roles, got %d", len(roles), len(apiKey.Roles))
	}

	if apiKey.Organization != organization {
		t.Errorf("Expected organization %s, got %s", organization, apiKey.Organization)
	}

	if !apiKey.Enabled {
		t.Error("API key should be enabled by default")
	}

	if apiKey.ExpiresAt == nil {
		t.Error("ExpiresAt should be set")
	}

	if apiKey.CreatedAt.IsZero() {
		t.Error("CreatedAt should be set")
	}

	if apiKey.ID == "" {
		t.Error("ID should be set")
	}

	// Verify key is hashed (not equal to raw key)
	if apiKey.Key == rawKey {
		t.Error("Stored key should be hashed, not raw")
	}

	// Verify key is stored
	retrievedKey, err := store.GetAPIKey(apiKey.ID)
	if err != nil {
		t.Fatalf("GetAPIKey failed: %v", err)
	}

	if retrievedKey.ID != apiKey.ID {
		t.Error("Retrieved key ID mismatch")
	}
}

func TestCreateAPIKey_NoExpiration(t *testing.T) {
	tmpDir := t.TempDir()
	store, err := NewAPIKeyStore(tmpDir)
	if err != nil {
		t.Fatalf("NewAPIKeyStore failed: %v", err)
	}

	_, apiKey, err := store.CreateAPIKey("test-key", "testuser", []string{"admin"}, "test-org", 0)
	if err != nil {
		t.Fatalf("CreateAPIKey failed: %v", err)
	}

	if apiKey.ExpiresAt != nil {
		t.Error("ExpiresAt should be nil for keys without expiration")
	}
}

func TestValidateAPIKey_Valid(t *testing.T) {
	tmpDir := t.TempDir()
	store, err := NewAPIKeyStore(tmpDir)
	if err != nil {
		t.Fatalf("NewAPIKeyStore failed: %v", err)
	}

	rawKey, createdKey, err := store.CreateAPIKey("test-key", "testuser", []string{"admin"}, "test-org", 24*time.Hour)
	if err != nil {
		t.Fatalf("CreateAPIKey failed: %v", err)
	}

	// Validate the key
	validatedKey, err := store.ValidateAPIKey(rawKey)
	if err != nil {
		t.Fatalf("ValidateAPIKey failed: %v", err)
	}

	if validatedKey.ID != createdKey.ID {
		t.Error("Validated key ID mismatch")
	}

	if validatedKey.Username != createdKey.Username {
		t.Error("Validated key username mismatch")
	}

	if validatedKey.LastUsedAt == nil {
		t.Error("LastUsedAt should be set after validation")
	}
}

func TestValidateAPIKey_Invalid(t *testing.T) {
	tmpDir := t.TempDir()
	store, err := NewAPIKeyStore(tmpDir)
	if err != nil {
		t.Fatalf("NewAPIKeyStore failed: %v", err)
	}

	// Try to validate non-existent key
	_, err = store.ValidateAPIKey("pf_invalid_key_12345")
	if err == nil {
		t.Error("Expected validation to fail for invalid key")
	}
}

func TestValidateAPIKey_Expired(t *testing.T) {
	tmpDir := t.TempDir()
	store, err := NewAPIKeyStore(tmpDir)
	if err != nil {
		t.Fatalf("NewAPIKeyStore failed: %v", err)
	}

	// Create key with very short expiration
	rawKey, _, err := store.CreateAPIKey("test-key", "testuser", []string{"admin"}, "test-org", 1*time.Millisecond)
	if err != nil {
		t.Fatalf("CreateAPIKey failed: %v", err)
	}

	// Wait for expiration
	time.Sleep(10 * time.Millisecond)

	// Try to validate expired key
	_, err = store.ValidateAPIKey(rawKey)
	if err == nil {
		t.Error("Expected validation to fail for expired key")
	}
}

func TestValidateAPIKey_Disabled(t *testing.T) {
	tmpDir := t.TempDir()
	store, err := NewAPIKeyStore(tmpDir)
	if err != nil {
		t.Fatalf("NewAPIKeyStore failed: %v", err)
	}

	rawKey, apiKey, err := store.CreateAPIKey("test-key", "testuser", []string{"admin"}, "test-org", 24*time.Hour)
	if err != nil {
		t.Fatalf("CreateAPIKey failed: %v", err)
	}

	// Disable the key
	err = store.RevokeAPIKey(apiKey.ID)
	if err != nil {
		t.Fatalf("RevokeAPIKey failed: %v", err)
	}

	// Try to validate disabled key
	_, err = store.ValidateAPIKey(rawKey)
	if err == nil {
		t.Error("Expected validation to fail for disabled key")
	}
}

func TestGetAPIKey(t *testing.T) {
	tmpDir := t.TempDir()
	store, err := NewAPIKeyStore(tmpDir)
	if err != nil {
		t.Fatalf("NewAPIKeyStore failed: %v", err)
	}

	_, apiKey, err := store.CreateAPIKey("test-key", "testuser", []string{"admin"}, "test-org", 24*time.Hour)
	if err != nil {
		t.Fatalf("CreateAPIKey failed: %v", err)
	}

	retrievedKey, err := store.GetAPIKey(apiKey.ID)
	if err != nil {
		t.Fatalf("GetAPIKey failed: %v", err)
	}

	if retrievedKey.ID != apiKey.ID {
		t.Error("Retrieved key ID mismatch")
	}
}

func TestGetAPIKey_NotFound(t *testing.T) {
	tmpDir := t.TempDir()
	store, err := NewAPIKeyStore(tmpDir)
	if err != nil {
		t.Fatalf("NewAPIKeyStore failed: %v", err)
	}

	_, err = store.GetAPIKey("non-existent-id")
	if err == nil {
		t.Error("Expected error for non-existent key")
	}
}

func TestListAPIKeys(t *testing.T) {
	tmpDir := t.TempDir()
	store, err := NewAPIKeyStore(tmpDir)
	if err != nil {
		t.Fatalf("NewAPIKeyStore failed: %v", err)
	}

	// Create keys for different users
	user1 := "user1"
	user2 := "user2"

	_, _, err = store.CreateAPIKey("key1", user1, []string{"admin"}, "org1", 24*time.Hour)
	if err != nil {
		t.Fatalf("CreateAPIKey failed: %v", err)
	}

	_, _, err = store.CreateAPIKey("key2", user1, []string{"developer"}, "org1", 24*time.Hour)
	if err != nil {
		t.Fatalf("CreateAPIKey failed: %v", err)
	}

	_, _, err = store.CreateAPIKey("key3", user2, []string{"viewer"}, "org2", 24*time.Hour)
	if err != nil {
		t.Fatalf("CreateAPIKey failed: %v", err)
	}

	// List keys for user1
	user1Keys := store.ListAPIKeys(user1)
	if len(user1Keys) != 2 {
		t.Errorf("Expected 2 keys for user1, got %d", len(user1Keys))
	}

	// List keys for user2
	user2Keys := store.ListAPIKeys(user2)
	if len(user2Keys) != 1 {
		t.Errorf("Expected 1 key for user2, got %d", len(user2Keys))
	}

	// List keys for non-existent user
	nonExistentKeys := store.ListAPIKeys("non-existent")
	if len(nonExistentKeys) != 0 {
		t.Errorf("Expected 0 keys for non-existent user, got %d", len(nonExistentKeys))
	}
}

func TestRevokeAPIKey(t *testing.T) {
	tmpDir := t.TempDir()
	store, err := NewAPIKeyStore(tmpDir)
	if err != nil {
		t.Fatalf("NewAPIKeyStore failed: %v", err)
	}

	rawKey, apiKey, err := store.CreateAPIKey("test-key", "testuser", []string{"admin"}, "test-org", 24*time.Hour)
	if err != nil {
		t.Fatalf("CreateAPIKey failed: %v", err)
	}

	// Key should be valid before revocation
	_, err = store.ValidateAPIKey(rawKey)
	if err != nil {
		t.Fatalf("Key should be valid before revocation: %v", err)
	}

	// Revoke the key
	err = store.RevokeAPIKey(apiKey.ID)
	if err != nil {
		t.Fatalf("RevokeAPIKey failed: %v", err)
	}

	// Key should be invalid after revocation
	_, err = store.ValidateAPIKey(rawKey)
	if err == nil {
		t.Error("Key should be invalid after revocation")
	}

	// Verify key is disabled in store
	revokedKey, err := store.GetAPIKey(apiKey.ID)
	if err != nil {
		t.Fatalf("GetAPIKey failed: %v", err)
	}

	if revokedKey.Enabled {
		t.Error("Key should be disabled after revocation")
	}
}

func TestRevokeAPIKey_NotFound(t *testing.T) {
	tmpDir := t.TempDir()
	store, err := NewAPIKeyStore(tmpDir)
	if err != nil {
		t.Fatalf("NewAPIKeyStore failed: %v", err)
	}

	err = store.RevokeAPIKey("non-existent-id")
	if err == nil {
		t.Error("Expected error when revoking non-existent key")
	}
}

func TestDeleteAPIKey(t *testing.T) {
	tmpDir := t.TempDir()
	store, err := NewAPIKeyStore(tmpDir)
	if err != nil {
		t.Fatalf("NewAPIKeyStore failed: %v", err)
	}

	_, apiKey, err := store.CreateAPIKey("test-key", "testuser", []string{"admin"}, "test-org", 24*time.Hour)
	if err != nil {
		t.Fatalf("CreateAPIKey failed: %v", err)
	}

	// Verify key exists
	_, err = store.GetAPIKey(apiKey.ID)
	if err != nil {
		t.Fatalf("Key should exist before deletion: %v", err)
	}

	// Delete the key
	err = store.DeleteAPIKey(apiKey.ID)
	if err != nil {
		t.Fatalf("DeleteAPIKey failed: %v", err)
	}

	// Verify key is gone
	_, err = store.GetAPIKey(apiKey.ID)
	if err == nil {
		t.Error("Key should not exist after deletion")
	}
}

func TestDeleteAPIKey_NotFound(t *testing.T) {
	tmpDir := t.TempDir()
	store, err := NewAPIKeyStore(tmpDir)
	if err != nil {
		t.Fatalf("NewAPIKeyStore failed: %v", err)
	}

	err = store.DeleteAPIKey("non-existent-id")
	if err == nil {
		t.Error("Expected error when deleting non-existent key")
	}
}

func TestAPIKeyPersistence(t *testing.T) {
	tmpDir := t.TempDir()

	// Create store and add keys
	store1, err := NewAPIKeyStore(tmpDir)
	if err != nil {
		t.Fatalf("NewAPIKeyStore failed: %v", err)
	}

	rawKey, apiKey1, err := store1.CreateAPIKey("test-key", "testuser", []string{"admin"}, "test-org", 24*time.Hour)
	if err != nil {
		t.Fatalf("CreateAPIKey failed: %v", err)
	}

	// Create new store instance (simulates restart)
	store2, err := NewAPIKeyStore(tmpDir)
	if err != nil {
		t.Fatalf("NewAPIKeyStore failed: %v", err)
	}

	// Verify key persisted
	retrievedKey, err := store2.GetAPIKey(apiKey1.ID)
	if err != nil {
		t.Fatalf("GetAPIKey failed after reload: %v", err)
	}

	if retrievedKey.ID != apiKey1.ID {
		t.Error("Key ID mismatch after reload")
	}

	if retrievedKey.Name != apiKey1.Name {
		t.Error("Key name mismatch after reload")
	}

	// Verify key still validates
	_, err = store2.ValidateAPIKey(rawKey)
	if err != nil {
		t.Fatalf("ValidateAPIKey failed after reload: %v", err)
	}
}

func TestAPIKeyFormat(t *testing.T) {
	tmpDir := t.TempDir()
	store, err := NewAPIKeyStore(tmpDir)
	if err != nil {
		t.Fatalf("NewAPIKeyStore failed: %v", err)
	}

	// Create multiple keys and verify format
	for i := 0; i < 5; i++ {
		rawKey, _, err := store.CreateAPIKey("test-key", "testuser", []string{"admin"}, "test-org", 24*time.Hour)
		if err != nil {
			t.Fatalf("CreateAPIKey failed: %v", err)
		}

		if !strings.HasPrefix(rawKey, "pf_") {
			t.Errorf("Key should start with 'pf_', got: %s", rawKey)
		}

		// Verify it's base64-like (URL-safe)
		keyPart := strings.TrimPrefix(rawKey, "pf_")
		if len(keyPart) < 20 {
			t.Errorf("Key part seems too short: %s", keyPart)
		}

		// Should only contain base64 URL-safe characters
		for _, ch := range keyPart {
			if !((ch >= 'A' && ch <= 'Z') || (ch >= 'a' && ch <= 'z') ||
				 (ch >= '0' && ch <= '9') || ch == '-' || ch == '_' || ch == '=') {
				t.Errorf("Invalid character in key: %c", ch)
			}
		}
	}
}

func TestMultipleKeys(t *testing.T) {
	tmpDir := t.TempDir()
	store, err := NewAPIKeyStore(tmpDir)
	if err != nil {
		t.Fatalf("NewAPIKeyStore failed: %v", err)
	}

	keys := make(map[string]string) // ID -> rawKey

	// Create 10 keys
	for i := 0; i < 10; i++ {
		rawKey, apiKey, err := store.CreateAPIKey("test-key", "testuser", []string{"admin"}, "test-org", 24*time.Hour)
		if err != nil {
			t.Fatalf("CreateAPIKey failed: %v", err)
		}
		keys[apiKey.ID] = rawKey
	}

	// Validate all keys
	for id, rawKey := range keys {
		validatedKey, err := store.ValidateAPIKey(rawKey)
		if err != nil {
			t.Fatalf("ValidateAPIKey failed for %s: %v", id, err)
		}

		if validatedKey.ID != id {
			t.Errorf("Key ID mismatch: expected %s, got %s", id, validatedKey.ID)
		}
	}
}

func TestAPIKeyStore_EmptyConfigDir(t *testing.T) {
	// Test with empty config dir (should use home directory)
	store, err := NewAPIKeyStore("")
	if err != nil {
		t.Fatalf("NewAPIKeyStore with empty config dir failed: %v", err)
	}

	if store == nil {
		t.Fatal("Store should not be nil")
	}

	// Clean up test data
	homeDir, _ := os.UserHomeDir()
	testFile := filepath.Join(homeDir, ".platformfoundry", "api_keys.json")
	os.Remove(testFile)
}
