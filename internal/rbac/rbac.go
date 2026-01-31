package rbac

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// Role represents a user role
type Role string

const (
	// Global roles
	RoleAdmin    Role = "admin"    // Full access to all resources
	RoleOperator Role = "operator" // Create, update, delete resources but not modify RBAC
	RoleViewer   Role = "viewer"   // Read-only access

	// Organization-scoped roles
	RoleOrgAdmin         Role = "org-admin"          // Full access within organization
	RolePlatformEngineer Role = "platform-engineer"  // Manage platforms and components
	RoleDeveloper        Role = "developer"          // Read-only + manage DevEx resources
)

// Action represents an action that can be performed on a resource
type Action string

const (
	ActionCreate   Action = "create"
	ActionRead     Action = "read"
	ActionUpdate   Action = "update"
	ActionDelete   Action = "delete"
	ActionList     Action = "list"
	ActionValidate Action = "validate"
	ActionPlan     Action = "plan"
)

// ResourceType represents the type of resource
type ResourceType string

const (
	ResourcePlatform       ResourceType = "Platform"
	ResourceInfrastructure ResourceType = "Infrastructure"
	ResourceOrchestrator   ResourceType = "Orchestrator"
	ResourceObservability  ResourceType = "Observability"
	ResourceDevEx          ResourceType = "DevEx"
	ResourcePipeline       ResourceType = "Pipeline"
	ResourceMesh           ResourceType = "Mesh"
	ResourceSecurity       ResourceType = "Security"
	ResourceCompliance     ResourceType = "Compliance"
	ResourceJob            ResourceType = "Job"
	ResourcePlugin         ResourceType = "Plugin"
	ResourceRBAC           ResourceType = "RBAC"
	ResourceOrganization   ResourceType = "Organization"
	ResourceEnvironment    ResourceType = "Environment"
	ResourceService        ResourceType = "Service"
	ResourceServiceTemplate ResourceType = "ServiceTemplate"
	ResourceServiceAction   ResourceType = "ServiceAction"
	ResourceServiceScorecard ResourceType = "ServiceScorecard"
)

// Permission represents a permission for a role on a resource type
type Permission struct {
	Role         Role         `json:"role"`
	ResourceType ResourceType `json:"resource_type"`
	Actions      []Action     `json:"actions"`
}

// User represents a user with roles
type User struct {
	Username          string              `json:"username"`
	Email             string              `json:"email"`
	Roles             []Role              `json:"roles"` // Global roles
	OrganizationRoles map[string][]Role   `json:"organization_roles,omitempty"` // Org-specific roles
	CreatedAt         time.Time           `json:"created_at"`
	UpdatedAt         time.Time           `json:"updated_at"`
}

// PermissionCheck represents a permission check event
type PermissionCheck struct {
	Username     string       `json:"username"`
	Role         Role         `json:"role"`
	ResourceType ResourceType `json:"resource_type"`
	ResourceName string       `json:"resource_name"`
	Action       Action       `json:"action"`
	Allowed      bool         `json:"allowed"`
	Timestamp    time.Time    `json:"timestamp"`
	Reason       string       `json:"reason,omitempty"`
}

// RBAC manages role-based access control
type RBAC struct {
	permissions     map[Role]map[ResourceType][]Action
	users           map[string]*User
	permissionLogs  []PermissionCheck
	mu              sync.RWMutex
	auditLogPath    string
	enableAuditLog  bool
}

var (
	// ErrPermissionDenied is returned when a user doesn't have permission
	ErrPermissionDenied = errors.New("permission denied")
	// ErrUserNotFound is returned when a user is not found
	ErrUserNotFound = errors.New("user not found")
	// ErrInvalidRole is returned when a role is invalid
	ErrInvalidRole = errors.New("invalid role")
)

// NewRBAC creates a new RBAC instance with default permissions
func NewRBAC(auditLogPath string, enableAuditLog bool) *RBAC {
	rbac := &RBAC{
		permissions:    make(map[Role]map[ResourceType][]Action),
		users:          make(map[string]*User),
		permissionLogs: make([]PermissionCheck, 0),
		auditLogPath:   auditLogPath,
		enableAuditLog: enableAuditLog,
	}

	// Initialize default permissions
	rbac.initializeDefaultPermissions()

	return rbac
}

// initializeDefaultPermissions sets up default permissions for each role
func (r *RBAC) initializeDefaultPermissions() {
	// Admin has all permissions on all resources
	adminResources := []ResourceType{
		ResourcePlatform, ResourceInfrastructure, ResourceOrchestrator,
		ResourceObservability, ResourceDevEx, ResourcePipeline,
		ResourceMesh, ResourceSecurity, ResourceCompliance,
		ResourceJob, ResourcePlugin, ResourceRBAC,
		ResourceOrganization, ResourceEnvironment,
		ResourceService, ResourceServiceTemplate, ResourceServiceAction,
		ResourceServiceScorecard,
	}
	adminActions := []Action{
		ActionCreate, ActionRead, ActionUpdate, ActionDelete,
		ActionList, ActionValidate, ActionPlan,
	}

	r.permissions[RoleAdmin] = make(map[ResourceType][]Action)
	for _, resource := range adminResources {
		r.permissions[RoleAdmin][resource] = adminActions
	}

	// Operator can perform most actions except RBAC management
	operatorResources := []ResourceType{
		ResourcePlatform, ResourceInfrastructure, ResourceOrchestrator,
		ResourceObservability, ResourceDevEx, ResourcePipeline,
		ResourceMesh, ResourceSecurity, ResourceCompliance,
		ResourceJob, ResourcePlugin,
		ResourceService, ResourceServiceTemplate, ResourceServiceAction,
		ResourceServiceScorecard,
	}
	operatorActions := []Action{
		ActionCreate, ActionRead, ActionUpdate, ActionDelete,
		ActionList, ActionValidate, ActionPlan,
	}

	r.permissions[RoleOperator] = make(map[ResourceType][]Action)
	for _, resource := range operatorResources {
		r.permissions[RoleOperator][resource] = operatorActions
	}
	// Operator can only read RBAC
	r.permissions[RoleOperator][ResourceRBAC] = []Action{ActionRead, ActionList}

	// Viewer can only read and list resources
	viewerResources := []ResourceType{
		ResourcePlatform, ResourceInfrastructure, ResourceOrchestrator,
		ResourceObservability, ResourceDevEx, ResourcePipeline,
		ResourceMesh, ResourceSecurity, ResourceCompliance,
		ResourceJob, ResourcePlugin, ResourceRBAC,
		ResourceService, ResourceServiceTemplate, ResourceServiceAction,
		ResourceServiceScorecard,
	}
	viewerActions := []Action{ActionRead, ActionList, ActionValidate, ActionPlan}

	r.permissions[RoleViewer] = make(map[ResourceType][]Action)
	for _, resource := range viewerResources {
		r.permissions[RoleViewer][resource] = viewerActions
	}
}

// AddUser adds a new user with specified roles
func (r *RBAC) AddUser(username, email string, roles []Role) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	// Validate roles
	for _, role := range roles {
		if !r.isValidRole(role) {
			return fmt.Errorf("%w: %s", ErrInvalidRole, role)
		}
	}

	user := &User{
		Username:  username,
		Email:     email,
		Roles:     roles,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	r.users[username] = user
	return nil
}

// UpdateUser updates a user's roles
func (r *RBAC) UpdateUser(username string, roles []Role) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	user, ok := r.users[username]
	if !ok {
		return ErrUserNotFound
	}

	// Validate roles
	for _, role := range roles {
		if !r.isValidRole(role) {
			return fmt.Errorf("%w: %s", ErrInvalidRole, role)
		}
	}

	user.Roles = roles
	user.UpdatedAt = time.Now()
	return nil
}

// GetUser returns a user by username
func (r *RBAC) GetUser(username string) (*User, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	user, ok := r.users[username]
	if !ok {
		return nil, ErrUserNotFound
	}

	return user, nil
}

// ListUsers returns all users
func (r *RBAC) ListUsers() []*User {
	r.mu.RLock()
	defer r.mu.RUnlock()

	users := make([]*User, 0, len(r.users))
	for _, user := range r.users {
		users = append(users, user)
	}

	return users
}

// DeleteUser removes a user
func (r *RBAC) DeleteUser(username string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, ok := r.users[username]; !ok {
		return ErrUserNotFound
	}

	delete(r.users, username)
	return nil
}

// CheckPermission checks if a user has permission to perform an action on a resource
func (r *RBAC) CheckPermission(username string, resourceType ResourceType, resourceName string, action Action) error {
	r.mu.RLock()
	user, ok := r.users[username]
	r.mu.RUnlock()

	if !ok {
		r.logPermissionCheck(username, "", resourceType, resourceName, action, false, "user not found")
		return ErrUserNotFound
	}

	// Check each role the user has
	for _, role := range user.Roles {
		if r.hasPermission(role, resourceType, action) {
			r.logPermissionCheck(username, role, resourceType, resourceName, action, true, "")
			return nil
		}
	}

	reason := fmt.Sprintf("user %s with roles %v does not have permission to %s on %s", username, user.Roles, action, resourceType)
	r.logPermissionCheck(username, user.Roles[0], resourceType, resourceName, action, false, reason)
	return ErrPermissionDenied
}

// hasPermission checks if a role has permission for an action on a resource type
func (r *RBAC) hasPermission(role Role, resourceType ResourceType, action Action) bool {
	r.mu.RLock()
	defer r.mu.RUnlock()

	resourcePerms, ok := r.permissions[role]
	if !ok {
		return false
	}

	actions, ok := resourcePerms[resourceType]
	if !ok {
		return false
	}

	for _, a := range actions {
		if a == action {
			return true
		}
	}

	return false
}

// isValidRole checks if a role is valid
func (r *RBAC) isValidRole(role Role) bool {
	return role == RoleAdmin || role == RoleOperator || role == RoleViewer ||
		role == RoleOrgAdmin || role == RolePlatformEngineer || role == RoleDeveloper
}

// logPermissionCheck logs a permission check event
func (r *RBAC) logPermissionCheck(username string, role Role, resourceType ResourceType, resourceName string, action Action, allowed bool, reason string) {
	check := PermissionCheck{
		Username:     username,
		Role:         role,
		ResourceType: resourceType,
		ResourceName: resourceName,
		Action:       action,
		Allowed:      allowed,
		Timestamp:    time.Now(),
		Reason:       reason,
	}

	r.mu.Lock()
	r.permissionLogs = append(r.permissionLogs, check)
	r.mu.Unlock()

	// Write to audit log if enabled
	if r.enableAuditLog && r.auditLogPath != "" {
		r.writeAuditLog(check)
	}
}

// writeAuditLog writes a permission check to the audit log file
func (r *RBAC) writeAuditLog(check PermissionCheck) {
	// Ensure directory exists
	dir := filepath.Dir(r.auditLogPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return
	}

	// Open file in append mode
	f, err := os.OpenFile(r.auditLogPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return
	}
	defer f.Close()

	// Write JSON line
	data, err := json.Marshal(check)
	if err != nil {
		return
	}

	f.Write(data)
	f.Write([]byte("\n"))
}

// GetPermissionLogs returns all permission check logs
func (r *RBAC) GetPermissionLogs() []PermissionCheck {
	r.mu.RLock()
	defer r.mu.RUnlock()

	logs := make([]PermissionCheck, len(r.permissionLogs))
	copy(logs, r.permissionLogs)
	return logs
}

// GetPermissionLogsForUser returns permission check logs for a specific user
func (r *RBAC) GetPermissionLogsForUser(username string) []PermissionCheck {
	r.mu.RLock()
	defer r.mu.RUnlock()

	logs := make([]PermissionCheck, 0)
	for _, log := range r.permissionLogs {
		if log.Username == username {
			logs = append(logs, log)
		}
	}

	return logs
}

// ClearPermissionLogs clears all permission check logs
func (r *RBAC) ClearPermissionLogs() {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.permissionLogs = make([]PermissionCheck, 0)
}

// GetRolePermissions returns all permissions for a role
func (r *RBAC) GetRolePermissions(role Role) map[ResourceType][]Action {
	r.mu.RLock()
	defer r.mu.RUnlock()

	perms, ok := r.permissions[role]
	if !ok {
		return nil
	}

	// Deep copy to prevent modification
	result := make(map[ResourceType][]Action)
	for resourceType, actions := range perms {
		result[resourceType] = make([]Action, len(actions))
		copy(result[resourceType], actions)
	}

	return result
}

// SetRolePermissions sets permissions for a role (admin only)
func (r *RBAC) SetRolePermissions(role Role, resourceType ResourceType, actions []Action) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if !r.isValidRole(role) {
		return ErrInvalidRole
	}

	if r.permissions[role] == nil {
		r.permissions[role] = make(map[ResourceType][]Action)
	}

	r.permissions[role][resourceType] = actions
	return nil
}

// ExportUsers exports users to JSON
func (r *RBAC) ExportUsers() (string, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	users := make([]*User, 0, len(r.users))
	for _, user := range r.users {
		users = append(users, user)
	}

	data, err := json.MarshalIndent(users, "", "  ")
	if err != nil {
		return "", err
	}

	return string(data), nil
}
