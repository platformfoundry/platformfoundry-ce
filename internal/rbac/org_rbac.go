package rbac

import (
	"fmt"
	"time"
)

// OrgRoleBinding represents a user's role in an organization
type OrgRoleBinding struct {
	Username     string    `json:"username"`
	Organization string    `json:"organization"`
	Role         Role      `json:"role"`
	CreatedAt    time.Time `json:"created_at"`
}

// AddOrganizationRole adds a role for a user in a specific organization
func (r *RBAC) AddOrganizationRole(username, organization string, role Role) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	user, ok := r.users[username]
	if !ok {
		return ErrUserNotFound
	}

	if !r.isValidOrgRole(role) {
		return fmt.Errorf("%w: %s", ErrInvalidRole, role)
	}

	if user.OrganizationRoles == nil {
		user.OrganizationRoles = make(map[string][]Role)
	}

	// Add role if not already present
	roles := user.OrganizationRoles[organization]
	for _, r := range roles {
		if r == role {
			return nil // Already has this role
		}
	}

	user.OrganizationRoles[organization] = append(roles, role)
	user.UpdatedAt = time.Now()

	return nil
}

// RemoveOrganizationRole removes a role from a user in an organization
func (r *RBAC) RemoveOrganizationRole(username, organization string, role Role) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	user, ok := r.users[username]
	if !ok {
		return ErrUserNotFound
	}

	if user.OrganizationRoles == nil {
		return nil
	}

	roles := user.OrganizationRoles[organization]
	newRoles := make([]Role, 0, len(roles))
	for _, r := range roles {
		if r != role {
			newRoles = append(newRoles, r)
		}
	}

	if len(newRoles) == 0 {
		delete(user.OrganizationRoles, organization)
	} else {
		user.OrganizationRoles[organization] = newRoles
	}

	user.UpdatedAt = time.Now()
	return nil
}

// CheckOrganizationPermission checks if a user has permission in an organization context
func (r *RBAC) CheckOrganizationPermission(username, organization string, resourceType ResourceType, action Action) error {
	r.mu.RLock()
	user, ok := r.users[username]
	r.mu.RUnlock()

	if !ok {
		reason := "user not found"
		r.logPermissionCheck(username, "", resourceType, organization, action, false, reason)
		return ErrUserNotFound
	}

	// Global admins have access to everything
	for _, role := range user.Roles {
		if role == RoleAdmin {
			r.logPermissionCheck(username, role, resourceType, organization, action, true, "")
			return nil
		}
	}

	// Check organization-specific roles
	if user.OrganizationRoles != nil {
		if orgRoles, ok := user.OrganizationRoles[organization]; ok {
			for _, role := range orgRoles {
				if r.hasOrgPermission(role, resourceType, action) {
					r.logPermissionCheck(username, role, resourceType, organization, action, true, "")
					return nil
				}
			}
		}
	}

	reason := fmt.Sprintf("user %s does not have permission to %s on %s in organization %s",
		username, action, resourceType, organization)
	r.logPermissionCheck(username, "", resourceType, organization, action, false, reason)
	return ErrPermissionDenied
}

// hasOrgPermission checks if an org role has permission
func (r *RBAC) hasOrgPermission(role Role, resourceType ResourceType, action Action) bool {
	permissions := r.getOrgRolePermissions(role)

	actions, ok := permissions[resourceType]
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

// getOrgRolePermissions returns permissions for org-scoped roles
func (r *RBAC) getOrgRolePermissions(role Role) map[ResourceType][]Action {
	permissions := make(map[ResourceType][]Action)

	switch role {
	case RoleOrgAdmin:
		// Full access to all resources within the organization
		resources := []ResourceType{
			ResourcePlatform, ResourceInfrastructure, ResourceOrchestrator,
			ResourceObservability, ResourceDevEx, ResourcePipeline,
			ResourceMesh, ResourceSecurity, ResourceCompliance,
			ResourceEnvironment,
			ResourceService, ResourceServiceTemplate, ResourceServiceAction,
			ResourceServiceScorecard,
		}
		actions := []Action{
			ActionCreate, ActionRead, ActionUpdate, ActionDelete,
			ActionList, ActionValidate, ActionPlan,
		}
		for _, res := range resources {
			permissions[res] = actions
		}
		// Org admin can manage org itself but not create/delete
		permissions[ResourceOrganization] = []Action{ActionRead, ActionUpdate, ActionList}

	case RolePlatformEngineer:
		// Can manage platforms and components but not organization settings
		resources := []ResourceType{
			ResourcePlatform, ResourceInfrastructure, ResourceOrchestrator,
			ResourceObservability, ResourceDevEx, ResourcePipeline,
			ResourceMesh, ResourceSecurity, ResourceCompliance,
			ResourceEnvironment,
			ResourceService, ResourceServiceTemplate, ResourceServiceAction,
			ResourceServiceScorecard,
		}
		actions := []Action{
			ActionCreate, ActionRead, ActionUpdate, ActionDelete,
			ActionList, ActionValidate, ActionPlan,
		}
		for _, res := range resources {
			permissions[res] = actions
		}
		// Can only read organization
		permissions[ResourceOrganization] = []Action{ActionRead, ActionList}

	case RoleDeveloper:
		// Read-only access plus ability to manage DevEx resources
		readOnlyResources := []ResourceType{
			ResourcePlatform, ResourceInfrastructure, ResourceOrchestrator,
			ResourceObservability, ResourceEnvironment, ResourceOrganization,
		}
		readOnlyActions := []Action{ActionRead, ActionList, ActionValidate, ActionPlan}

		for _, res := range readOnlyResources {
			permissions[res] = readOnlyActions
		}

		// Can manage DevEx resources
		permissions[ResourceDevEx] = []Action{
			ActionCreate, ActionRead, ActionUpdate, ActionDelete,
			ActionList, ActionValidate, ActionPlan,
		}

		// Can manage Service resources (create, read, update for own team; deploy to dev/staging)
		permissions[ResourceService] = []Action{
			ActionCreate, ActionRead, ActionUpdate, ActionList,
			ActionValidate, ActionPlan,
		}
		permissions[ResourceServiceAction] = []Action{
			ActionCreate, ActionRead, ActionList,
		}
		// Can read service templates and scorecards
		permissions[ResourceServiceTemplate] = []Action{ActionRead, ActionList}
		permissions[ResourceServiceScorecard] = []Action{ActionRead, ActionList}
	}

	return permissions
}

// isValidOrgRole checks if a role is a valid organization-scoped role
func (r *RBAC) isValidOrgRole(role Role) bool {
	return role == RoleOrgAdmin || role == RolePlatformEngineer || role == RoleDeveloper
}

// GetUserOrganizations returns all organizations a user has access to
func (r *RBAC) GetUserOrganizations(username string) []string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	user, ok := r.users[username]
	if !ok {
		return []string{}
	}

	orgs := make([]string, 0, len(user.OrganizationRoles))
	for org := range user.OrganizationRoles {
		orgs = append(orgs, org)
	}

	return orgs
}

// GetOrganizationMembers returns all users with roles in an organization
func (r *RBAC) GetOrganizationMembers(organization string) []OrgRoleBinding {
	r.mu.RLock()
	defer r.mu.RUnlock()

	members := make([]OrgRoleBinding, 0)

	for username, user := range r.users {
		if user.OrganizationRoles != nil {
			if roles, ok := user.OrganizationRoles[organization]; ok {
				for _, role := range roles {
					members = append(members, OrgRoleBinding{
						Username:     username,
						Organization: organization,
						Role:         role,
						CreatedAt:    user.CreatedAt,
					})
				}
			}
		}
	}

	return members
}

// GetUserRolesInOrganization returns all roles a user has in an organization
func (r *RBAC) GetUserRolesInOrganization(username, organization string) []Role {
	r.mu.RLock()
	defer r.mu.RUnlock()

	user, ok := r.users[username]
	if !ok {
		return []Role{}
	}

	// Global admin has implicit access
	for _, role := range user.Roles {
		if role == RoleAdmin {
			return []Role{RoleAdmin}
		}
	}

	if user.OrganizationRoles == nil {
		return []Role{}
	}

	if roles, ok := user.OrganizationRoles[organization]; ok {
		return roles
	}

	return []Role{}
}
