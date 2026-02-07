package auth

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestNewUserStore(t *testing.T) {
	tmpDir := t.TempDir()

	store, err := NewUserStore(tmpDir)
	if err != nil {
		t.Fatalf("NewUserStore failed: %v", err)
	}

	if store == nil {
		t.Fatal("NewUserStore returned nil")
	}

	expectedPath := filepath.Join(tmpDir, "users.json")
	if store.filePath != expectedPath {
		t.Errorf("Expected filePath %s, got %s", expectedPath, store.filePath)
	}

	if store.users == nil {
		t.Fatal("Users map should be initialized")
	}
}

func TestCreateUser(t *testing.T) {
	tmpDir := t.TempDir()
	store, err := NewUserStore(tmpDir)
	if err != nil {
		t.Fatalf("NewUserStore failed: %v", err)
	}

	username := "testuser"
	email := "test@example.com"
	password := "Test123!@#"
	roles := []string{"admin", "developer"}
	organization := "test-org"

	err = store.CreateUser(username, email, password, roles, organization)
	if err != nil {
		t.Fatalf("CreateUser failed: %v", err)
	}

	// Verify user was created
	user, err := store.GetUser(username)
	if err != nil {
		t.Fatalf("GetUser failed: %v", err)
	}

	if user.Username != username {
		t.Errorf("Expected username %s, got %s", username, user.Username)
	}

	if user.Email != email {
		t.Errorf("Expected email %s, got %s", email, user.Email)
	}

	if len(user.Roles) != len(roles) {
		t.Errorf("Expected %d roles, got %d", len(roles), len(user.Roles))
	}

	if user.Organization != organization {
		t.Errorf("Expected organization %s, got %s", organization, user.Organization)
	}

	if !user.Enabled {
		t.Error("User should be enabled by default")
	}

	if user.CreatedAt.IsZero() {
		t.Error("CreatedAt should be set")
	}

	if user.UpdatedAt.IsZero() {
		t.Error("UpdatedAt should be set")
	}

	// Verify password was hashed
	if user.PasswordHash == password {
		t.Error("Password should be hashed, not stored in plaintext")
	}

	if len(user.PasswordHash) < 20 {
		t.Error("Password hash seems too short")
	}
}

func TestCreateUser_DuplicateUsername(t *testing.T) {
	tmpDir := t.TempDir()
	store, err := NewUserStore(tmpDir)
	if err != nil {
		t.Fatalf("NewUserStore failed: %v", err)
	}

	username := "testuser"
	password := "Test123!@#"

	// Create first user
	err = store.CreateUser(username, "test1@example.com", password, []string{"admin"}, "org1")
	if err != nil {
		t.Fatalf("First CreateUser failed: %v", err)
	}

	// Try to create duplicate
	err = store.CreateUser(username, "test2@example.com", password, []string{"developer"}, "org2")
	if err == nil {
		t.Error("Expected error when creating duplicate user")
	}
}

func TestGetUser(t *testing.T) {
	tmpDir := t.TempDir()
	store, err := NewUserStore(tmpDir)
	if err != nil {
		t.Fatalf("NewUserStore failed: %v", err)
	}

	username := "testuser"
	err = store.CreateUser(username, "test@example.com", "password", []string{"admin"}, "test-org")
	if err != nil {
		t.Fatalf("CreateUser failed: %v", err)
	}

	user, err := store.GetUser(username)
	if err != nil {
		t.Fatalf("GetUser failed: %v", err)
	}

	if user.Username != username {
		t.Error("Username mismatch")
	}
}

func TestGetUser_NotFound(t *testing.T) {
	tmpDir := t.TempDir()
	store, err := NewUserStore(tmpDir)
	if err != nil {
		t.Fatalf("NewUserStore failed: %v", err)
	}

	_, err = store.GetUser("nonexistent")
	if err == nil {
		t.Error("Expected error for non-existent user")
	}
}

func TestListUsers(t *testing.T) {
	tmpDir := t.TempDir()
	store, err := NewUserStore(tmpDir)
	if err != nil {
		t.Fatalf("NewUserStore failed: %v", err)
	}

	// Initially empty
	users := store.ListUsers()
	if len(users) != 0 {
		t.Errorf("Expected 0 users, got %d", len(users))
	}

	// Create multiple users
	usernames := []string{"user1", "user2", "user3"}
	for _, username := range usernames {
		err = store.CreateUser(username, username+"@example.com", "password", []string{"user"}, "org")
		if err != nil {
			t.Fatalf("CreateUser failed for %s: %v", username, err)
		}
	}

	// List should return all users
	users = store.ListUsers()
	if len(users) != len(usernames) {
		t.Errorf("Expected %d users, got %d", len(usernames), len(users))
	}
}

func TestUpdateUser(t *testing.T) {
	tmpDir := t.TempDir()
	store, err := NewUserStore(tmpDir)
	if err != nil {
		t.Fatalf("NewUserStore failed: %v", err)
	}

	username := "testuser"
	err = store.CreateUser(username, "old@example.com", "password", []string{"user"}, "old-org")
	if err != nil {
		t.Fatalf("CreateUser failed: %v", err)
	}

	// Get and modify user
	user, _ := store.GetUser(username)
	originalUpdatedAt := user.UpdatedAt
	time.Sleep(10 * time.Millisecond) // Ensure timestamp differs

	user.Email = "new@example.com"
	user.Roles = []string{"admin", "developer"}
	user.Organization = "new-org"

	err = store.UpdateUser(user)
	if err != nil {
		t.Fatalf("UpdateUser failed: %v", err)
	}

	// Verify updates
	updatedUser, _ := store.GetUser(username)
	if updatedUser.Email != "new@example.com" {
		t.Error("Email not updated")
	}

	if len(updatedUser.Roles) != 2 {
		t.Error("Roles not updated")
	}

	if updatedUser.Organization != "new-org" {
		t.Error("Organization not updated")
	}

	if !updatedUser.UpdatedAt.After(originalUpdatedAt) {
		t.Error("UpdatedAt should be later after update")
	}
}

func TestUpdateUser_NotFound(t *testing.T) {
	tmpDir := t.TempDir()
	store, err := NewUserStore(tmpDir)
	if err != nil {
		t.Fatalf("NewUserStore failed: %v", err)
	}

	user := &User{
		Username: "nonexistent",
		Email:    "test@example.com",
	}

	err = store.UpdateUser(user)
	if err == nil {
		t.Error("Expected error when updating non-existent user")
	}
}

func TestDeleteUser(t *testing.T) {
	tmpDir := t.TempDir()
	store, err := NewUserStore(tmpDir)
	if err != nil {
		t.Fatalf("NewUserStore failed: %v", err)
	}

	username := "testuser"
	err = store.CreateUser(username, "test@example.com", "password", []string{"user"}, "org")
	if err != nil {
		t.Fatalf("CreateUser failed: %v", err)
	}

	// Verify user exists
	_, err = store.GetUser(username)
	if err != nil {
		t.Fatalf("User should exist before deletion: %v", err)
	}

	// Delete user
	err = store.DeleteUser(username)
	if err != nil {
		t.Fatalf("DeleteUser failed: %v", err)
	}

	// Verify user is gone
	_, err = store.GetUser(username)
	if err == nil {
		t.Error("User should not exist after deletion")
	}
}

func TestDeleteUser_NotFound(t *testing.T) {
	tmpDir := t.TempDir()
	store, err := NewUserStore(tmpDir)
	if err != nil {
		t.Fatalf("NewUserStore failed: %v", err)
	}

	err = store.DeleteUser("nonexistent")
	if err == nil {
		t.Error("Expected error when deleting non-existent user")
	}
}

func TestAuthenticate_Success(t *testing.T) {
	tmpDir := t.TempDir()
	store, err := NewUserStore(tmpDir)
	if err != nil {
		t.Fatalf("NewUserStore failed: %v", err)
	}

	username := "testuser"
	password := "Test123!@#"

	err = store.CreateUser(username, "test@example.com", password, []string{"admin"}, "org")
	if err != nil {
		t.Fatalf("CreateUser failed: %v", err)
	}

	// Authenticate with correct password
	user, err := store.Authenticate(username, password)
	if err != nil {
		t.Fatalf("Authenticate failed: %v", err)
	}

	if user.Username != username {
		t.Error("Authenticated user mismatch")
	}
}

func TestAuthenticate_WrongPassword(t *testing.T) {
	tmpDir := t.TempDir()
	store, err := NewUserStore(tmpDir)
	if err != nil {
		t.Fatalf("NewUserStore failed: %v", err)
	}

	username := "testuser"
	password := "correct"

	err = store.CreateUser(username, "test@example.com", password, []string{"admin"}, "org")
	if err != nil {
		t.Fatalf("CreateUser failed: %v", err)
	}

	// Try wrong password
	_, err = store.Authenticate(username, "wrong")
	if err == nil {
		t.Error("Expected authentication to fail with wrong password")
	}
}

func TestAuthenticate_UserNotFound(t *testing.T) {
	tmpDir := t.TempDir()
	store, err := NewUserStore(tmpDir)
	if err != nil {
		t.Fatalf("NewUserStore failed: %v", err)
	}

	_, err = store.Authenticate("nonexistent", "password")
	if err == nil {
		t.Error("Expected authentication to fail for non-existent user")
	}
}

func TestAuthenticate_DisabledUser(t *testing.T) {
	tmpDir := t.TempDir()
	store, err := NewUserStore(tmpDir)
	if err != nil {
		t.Fatalf("NewUserStore failed: %v", err)
	}

	username := "testuser"
	password := "password"

	err = store.CreateUser(username, "test@example.com", password, []string{"admin"}, "org")
	if err != nil {
		t.Fatalf("CreateUser failed: %v", err)
	}

	// Disable user
	user, _ := store.GetUser(username)
	user.Enabled = false
	store.UpdateUser(user)

	// Try to authenticate
	_, err = store.Authenticate(username, password)
	if err == nil {
		t.Error("Expected authentication to fail for disabled user")
	}
}

func TestUpdatePassword(t *testing.T) {
	tmpDir := t.TempDir()
	store, err := NewUserStore(tmpDir)
	if err != nil {
		t.Fatalf("NewUserStore failed: %v", err)
	}

	username := "testuser"
	oldPassword := "old123"
	newPassword := "new456"

	err = store.CreateUser(username, "test@example.com", oldPassword, []string{"admin"}, "org")
	if err != nil {
		t.Fatalf("CreateUser failed: %v", err)
	}

	// Get original password hash
	user, _ := store.GetUser(username)
	oldHash := user.PasswordHash

	// Update password
	err = store.UpdatePassword(username, newPassword)
	if err != nil {
		t.Fatalf("UpdatePassword failed: %v", err)
	}

	// Verify password hash changed
	user, _ = store.GetUser(username)
	if user.PasswordHash == oldHash {
		t.Error("Password hash should have changed")
	}

	// Old password should no longer work
	_, err = store.Authenticate(username, oldPassword)
	if err == nil {
		t.Error("Old password should not work after update")
	}

	// New password should work
	_, err = store.Authenticate(username, newPassword)
	if err != nil {
		t.Fatalf("New password should work: %v", err)
	}
}

func TestUpdatePassword_UserNotFound(t *testing.T) {
	tmpDir := t.TempDir()
	store, err := NewUserStore(tmpDir)
	if err != nil {
		t.Fatalf("NewUserStore failed: %v", err)
	}

	err = store.UpdatePassword("nonexistent", "newpassword")
	if err == nil {
		t.Error("Expected error when updating password for non-existent user")
	}
}

func TestUserPersistence(t *testing.T) {
	tmpDir := t.TempDir()

	// Create store and add user
	store1, err := NewUserStore(tmpDir)
	if err != nil {
		t.Fatalf("NewUserStore failed: %v", err)
	}

	username := "testuser"
	password := "password123"

	err = store1.CreateUser(username, "test@example.com", password, []string{"admin"}, "test-org")
	if err != nil {
		t.Fatalf("CreateUser failed: %v", err)
	}

	// Create new store instance (simulates restart)
	store2, err := NewUserStore(tmpDir)
	if err != nil {
		t.Fatalf("NewUserStore failed: %v", err)
	}

	// Verify user persisted
	user, err := store2.GetUser(username)
	if err != nil {
		t.Fatalf("GetUser failed after reload: %v", err)
	}

	if user.Username != username {
		t.Error("Username mismatch after reload")
	}

	// Verify authentication still works
	_, err = store2.Authenticate(username, password)
	if err != nil {
		t.Fatalf("Authenticate failed after reload: %v", err)
	}
}

func TestMultipleUsers(t *testing.T) {
	tmpDir := t.TempDir()
	store, err := NewUserStore(tmpDir)
	if err != nil {
		t.Fatalf("NewUserStore failed: %v", err)
	}

	// Create 10 users
	for i := 0; i < 10; i++ {
		username := fmt.Sprintf("user%d", i)
		email := fmt.Sprintf("user%d@example.com", i)
		password := fmt.Sprintf("password%d", i)

		err = store.CreateUser(username, email, password, []string{"user"}, "org")
		if err != nil {
			t.Fatalf("CreateUser failed for %s: %v", username, err)
		}
	}

	// Verify all users exist
	users := store.ListUsers()
	if len(users) != 10 {
		t.Errorf("Expected 10 users, got %d", len(users))
	}

	// Authenticate all users
	for i := 0; i < 10; i++ {
		username := fmt.Sprintf("user%d", i)
		password := fmt.Sprintf("password%d", i)

		_, err := store.Authenticate(username, password)
		if err != nil {
			t.Errorf("Authenticate failed for %s: %v", username, err)
		}
	}
}

func TestUserStore_EmptyConfigDir(t *testing.T) {
	// Test with empty config dir (should use home directory)
	store, err := NewUserStore("")
	if err != nil {
		t.Fatalf("NewUserStore with empty config dir failed: %v", err)
	}

	if store == nil {
		t.Fatal("Store should not be nil")
	}

	// Clean up test data
	homeDir, _ := os.UserHomeDir()
	testFile := filepath.Join(homeDir, ".platformfoundry", "users.json")
	os.Remove(testFile)
}
