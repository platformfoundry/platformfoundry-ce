package rbac

import (
	"os"
	"testing"
)

func TestNewRBAC(t *testing.T) {
	rbac := NewRBAC("", false)
	if rbac == nil {
		t.Fatal("NewRBAC() returned nil")
	}

	if rbac.permissions == nil {
		t.Error("permissions should be initialized")
	}

	if rbac.users == nil {
		t.Error("users should be initialized")
	}
}

func TestDefaultPermissions(t *testing.T) {
	rbac := NewRBAC("", false)

	// Admin should have all permissions
	adminPerms := rbac.GetRolePermissions(RoleAdmin)
	if len(adminPerms) == 0 {
		t.Error("Admin should have permissions")
	}

	// Check admin has permission on Platform
	platformPerms, ok := adminPerms[ResourcePlatform]
	if !ok {
		t.Error("Admin should have permissions on Platform")
	}

	if !contains(platformPerms, ActionCreate) {
		t.Error("Admin should have create permission")
	}

	// Operator should have permissions but not on RBAC
	operatorPerms := rbac.GetRolePermissions(RoleOperator)
	if len(operatorPerms) == 0 {
		t.Error("Operator should have permissions")
	}

	// Operator should only read RBAC
	rbacPerms, ok := operatorPerms[ResourceRBAC]
	if !ok {
		t.Error("Operator should have some permissions on RBAC")
	}

	if contains(rbacPerms, ActionCreate) {
		t.Error("Operator should not have create permission on RBAC")
	}

	// Viewer should only have read permissions
	viewerPerms := rbac.GetRolePermissions(RoleViewer)
	if len(viewerPerms) == 0 {
		t.Error("Viewer should have permissions")
	}

	platformViewerPerms, ok := viewerPerms[ResourcePlatform]
	if !ok {
		t.Error("Viewer should have permissions on Platform")
	}

	if contains(platformViewerPerms, ActionCreate) {
		t.Error("Viewer should not have create permission")
	}

	if !contains(platformViewerPerms, ActionRead) {
		t.Error("Viewer should have read permission")
	}
}

func TestAddUser(t *testing.T) {
	rbac := NewRBAC("", false)

	err := rbac.AddUser("alice", "alice@example.com", []Role{RoleAdmin})
	if err != nil {
		t.Fatalf("AddUser() error = %v", err)
	}

	user, err := rbac.GetUser("alice")
	if err != nil {
		t.Fatalf("GetUser() error = %v", err)
	}

	if user.Username != "alice" {
		t.Errorf("Expected username 'alice', got '%s'", user.Username)
	}

	if user.Email != "alice@example.com" {
		t.Errorf("Expected email 'alice@example.com', got '%s'", user.Email)
	}

	if len(user.Roles) != 1 || user.Roles[0] != RoleAdmin {
		t.Error("User should have admin role")
	}
}

func TestAddUserInvalidRole(t *testing.T) {
	rbac := NewRBAC("", false)

	err := rbac.AddUser("alice", "alice@example.com", []Role{"invalid"})
	if err == nil {
		t.Error("AddUser() should fail with invalid role")
	}
}

func TestUpdateUser(t *testing.T) {
	rbac := NewRBAC("", false)

	rbac.AddUser("alice", "alice@example.com", []Role{RoleViewer})

	err := rbac.UpdateUser("alice", []Role{RoleOperator})
	if err != nil {
		t.Fatalf("UpdateUser() error = %v", err)
	}

	user, _ := rbac.GetUser("alice")
	if len(user.Roles) != 1 || user.Roles[0] != RoleOperator {
		t.Error("User roles should be updated to operator")
	}
}

func TestUpdateUserNotFound(t *testing.T) {
	rbac := NewRBAC("", false)

	err := rbac.UpdateUser("nonexistent", []Role{RoleOperator})
	if err != ErrUserNotFound {
		t.Error("UpdateUser() should return ErrUserNotFound")
	}
}

func TestDeleteUser(t *testing.T) {
	rbac := NewRBAC("", false)

	rbac.AddUser("alice", "alice@example.com", []Role{RoleAdmin})

	err := rbac.DeleteUser("alice")
	if err != nil {
		t.Fatalf("DeleteUser() error = %v", err)
	}

	_, err = rbac.GetUser("alice")
	if err != ErrUserNotFound {
		t.Error("GetUser() should return ErrUserNotFound after deletion")
	}
}

func TestListUsers(t *testing.T) {
	rbac := NewRBAC("", false)

	rbac.AddUser("alice", "alice@example.com", []Role{RoleAdmin})
	rbac.AddUser("bob", "bob@example.com", []Role{RoleOperator})
	rbac.AddUser("charlie", "charlie@example.com", []Role{RoleViewer})

	users := rbac.ListUsers()
	if len(users) != 3 {
		t.Errorf("Expected 3 users, got %d", len(users))
	}
}

func TestCheckPermissionAdmin(t *testing.T) {
	rbac := NewRBAC("", false)
	rbac.AddUser("alice", "alice@example.com", []Role{RoleAdmin})

	// Admin should have all permissions
	err := rbac.CheckPermission("alice", ResourcePlatform, "test-platform", ActionCreate)
	if err != nil {
		t.Error("Admin should have create permission on Platform")
	}

	err = rbac.CheckPermission("alice", ResourceRBAC, "test-rbac", ActionCreate)
	if err != nil {
		t.Error("Admin should have create permission on RBAC")
	}
}

func TestCheckPermissionOperator(t *testing.T) {
	rbac := NewRBAC("", false)
	rbac.AddUser("bob", "bob@example.com", []Role{RoleOperator})

	// Operator should have permissions on resources
	err := rbac.CheckPermission("bob", ResourcePlatform, "test-platform", ActionCreate)
	if err != nil {
		t.Error("Operator should have create permission on Platform")
	}

	// Operator should NOT have create permission on RBAC
	err = rbac.CheckPermission("bob", ResourceRBAC, "test-rbac", ActionCreate)
	if err != ErrPermissionDenied {
		t.Error("Operator should not have create permission on RBAC")
	}

	// Operator should have read permission on RBAC
	err = rbac.CheckPermission("bob", ResourceRBAC, "test-rbac", ActionRead)
	if err != nil {
		t.Error("Operator should have read permission on RBAC")
	}
}

func TestCheckPermissionViewer(t *testing.T) {
	rbac := NewRBAC("", false)
	rbac.AddUser("charlie", "charlie@example.com", []Role{RoleViewer})

	// Viewer should have read permission
	err := rbac.CheckPermission("charlie", ResourcePlatform, "test-platform", ActionRead)
	if err != nil {
		t.Error("Viewer should have read permission on Platform")
	}

	// Viewer should NOT have create permission
	err = rbac.CheckPermission("charlie", ResourcePlatform, "test-platform", ActionCreate)
	if err != ErrPermissionDenied {
		t.Error("Viewer should not have create permission on Platform")
	}
}

func TestCheckPermissionUserNotFound(t *testing.T) {
	rbac := NewRBAC("", false)

	err := rbac.CheckPermission("nonexistent", ResourcePlatform, "test", ActionRead)
	if err != ErrUserNotFound {
		t.Error("CheckPermission() should return ErrUserNotFound")
	}
}

func TestCheckPermissionMultipleRoles(t *testing.T) {
	rbac := NewRBAC("", false)
	rbac.AddUser("alice", "alice@example.com", []Role{RoleViewer, RoleOperator})

	// User with multiple roles should have permissions from all roles
	err := rbac.CheckPermission("alice", ResourcePlatform, "test-platform", ActionCreate)
	if err != nil {
		t.Error("User with operator role should have create permission")
	}
}

func TestPermissionLogs(t *testing.T) {
	rbac := NewRBAC("", false)
	rbac.AddUser("alice", "alice@example.com", []Role{RoleAdmin})
	rbac.AddUser("bob", "bob@example.com", []Role{RoleViewer})

	// Perform some permission checks
	rbac.CheckPermission("alice", ResourcePlatform, "test1", ActionCreate)
	rbac.CheckPermission("bob", ResourcePlatform, "test2", ActionRead)
	rbac.CheckPermission("bob", ResourcePlatform, "test3", ActionCreate) // Should be denied

	logs := rbac.GetPermissionLogs()
	if len(logs) != 3 {
		t.Errorf("Expected 3 permission logs, got %d", len(logs))
	}

	// Check denied log
	deniedFound := false
	for _, log := range logs {
		if !log.Allowed {
			deniedFound = true
			if log.Username != "bob" {
				t.Error("Denied log should be for bob")
			}
			if log.Action != ActionCreate {
				t.Error("Denied log should be for create action")
			}
		}
	}

	if !deniedFound {
		t.Error("Should have found a denied permission log")
	}
}

func TestGetPermissionLogsForUser(t *testing.T) {
	rbac := NewRBAC("", false)
	rbac.AddUser("alice", "alice@example.com", []Role{RoleAdmin})
	rbac.AddUser("bob", "bob@example.com", []Role{RoleViewer})

	rbac.CheckPermission("alice", ResourcePlatform, "test1", ActionCreate)
	rbac.CheckPermission("alice", ResourcePlatform, "test2", ActionDelete)
	rbac.CheckPermission("bob", ResourcePlatform, "test3", ActionRead)

	aliceLogs := rbac.GetPermissionLogsForUser("alice")
	if len(aliceLogs) != 2 {
		t.Errorf("Expected 2 logs for alice, got %d", len(aliceLogs))
	}

	bobLogs := rbac.GetPermissionLogsForUser("bob")
	if len(bobLogs) != 1 {
		t.Errorf("Expected 1 log for bob, got %d", len(bobLogs))
	}
}

func TestClearPermissionLogs(t *testing.T) {
	rbac := NewRBAC("", false)
	rbac.AddUser("alice", "alice@example.com", []Role{RoleAdmin})

	rbac.CheckPermission("alice", ResourcePlatform, "test", ActionCreate)

	logs := rbac.GetPermissionLogs()
	if len(logs) == 0 {
		t.Error("Should have logs before clearing")
	}

	rbac.ClearPermissionLogs()

	logs = rbac.GetPermissionLogs()
	if len(logs) != 0 {
		t.Error("Logs should be cleared")
	}
}

func TestSetRolePermissions(t *testing.T) {
	rbac := NewRBAC("", false)

	err := rbac.SetRolePermissions(RoleViewer, ResourcePlatform, []Action{ActionCreate, ActionRead})
	if err != nil {
		t.Fatalf("SetRolePermissions() error = %v", err)
	}

	perms := rbac.GetRolePermissions(RoleViewer)
	platformPerms := perms[ResourcePlatform]

	if !contains(platformPerms, ActionCreate) {
		t.Error("Viewer should now have create permission on Platform")
	}
}

func TestExportUsers(t *testing.T) {
	rbac := NewRBAC("", false)
	rbac.AddUser("alice", "alice@example.com", []Role{RoleAdmin})
	rbac.AddUser("bob", "bob@example.com", []Role{RoleOperator})

	json, err := rbac.ExportUsers()
	if err != nil {
		t.Fatalf("ExportUsers() error = %v", err)
	}

	if json == "" {
		t.Error("ExportUsers() should return non-empty JSON")
	}

	if !containsStr(json, "alice") {
		t.Error("JSON should contain alice")
	}

	if !containsStr(json, "bob") {
		t.Error("JSON should contain bob")
	}
}

func TestAuditLogFile(t *testing.T) {
	// Create temp directory for test
	tmpDir := os.TempDir()
	auditPath := tmpDir + "/rbac_test_audit.log"
	defer os.Remove(auditPath)

	rbac := NewRBAC(auditPath, true)
	rbac.AddUser("alice", "alice@example.com", []Role{RoleAdmin})

	rbac.CheckPermission("alice", ResourcePlatform, "test", ActionCreate)

	// Check if audit log file was created
	if _, err := os.Stat(auditPath); os.IsNotExist(err) {
		t.Error("Audit log file should be created")
	}
}

func contains(actions []Action, action Action) bool {
	for _, a := range actions {
		if a == action {
			return true
		}
	}
	return false
}

func containsStr(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) && (s[:len(substr)] == substr || s[len(s)-len(substr):] == substr || containsMiddle(s, substr)))
}

func containsMiddle(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
