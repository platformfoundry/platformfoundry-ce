package tenancy

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// Manager handles multi-tenant operations
type Manager struct {
	tenants       map[string]*Tenant
	roles         map[string]map[string]*Role        // tenant -> role name -> role
	bindings      map[string]map[string]*RoleBinding // tenant -> binding name -> binding
	jitRequests   map[string]*JITAccessRequest
	accessReviews map[string]*AccessReview
	mu            sync.RWMutex
}

// NewManager creates a new tenancy manager
func NewManager() *Manager {
	m := &Manager{
		tenants:       make(map[string]*Tenant),
		roles:         make(map[string]map[string]*Role),
		bindings:      make(map[string]map[string]*RoleBinding),
		jitRequests:   make(map[string]*JITAccessRequest),
		accessReviews: make(map[string]*AccessReview),
	}

	// Load built-in roles
	m.loadBuiltInRoles()

	return m
}

// loadBuiltInRoles creates default roles
func (m *Manager) loadBuiltInRoles() {
	// Global built-in roles
	builtInRoles := []*Role{
		{
			APIVersion: "platformfoundry.io/v1",
			Kind:       "Role",
			Metadata: RoleMetadata{
				Name:        "admin",
				Description: "Full administrative access",
			},
			Spec: RoleSpec{
				Permissions: []Permission{
					{
						Resources: []string{"*"},
						Verbs:     []string{"*"},
					},
				},
			},
		},
		{
			APIVersion: "platformfoundry.io/v1",
			Kind:       "Role",
			Metadata: RoleMetadata{
				Name:        "developer",
				Description: "Developer access to non-production environments",
			},
			Spec: RoleSpec{
				Permissions: []Permission{
					{
						Resources:    []string{"deployments", "services", "configmaps", "secrets"},
						Verbs:        []string{"get", "list", "create", "update", "delete"},
						Environments: []string{"dev", "staging"},
					},
					{
						Resources:    []string{"deployments", "services"},
						Verbs:        []string{"get", "list"},
						Environments: []string{"production"},
					},
				},
				Constraints: []Constraint{
					{Type: "maxReplicas", Value: "10"},
					{Type: "deniedImages", Value: "*:latest"},
				},
			},
		},
		{
			APIVersion: "platformfoundry.io/v1",
			Kind:       "Role",
			Metadata: RoleMetadata{
				Name:        "viewer",
				Description: "Read-only access",
			},
			Spec: RoleSpec{
				Permissions: []Permission{
					{
						Resources: []string{"deployments", "services", "configmaps"},
						Verbs:     []string{"get", "list"},
					},
				},
			},
		},
		{
			APIVersion: "platformfoundry.io/v1",
			Kind:       "Role",
			Metadata: RoleMetadata{
				Name:        "deployer",
				Description: "Can deploy to production",
			},
			Spec: RoleSpec{
				Permissions: []Permission{
					{
						Resources:    []string{"deployments"},
						Verbs:        []string{"get", "list", "create", "update"},
						Environments: []string{"production"},
					},
				},
				InheritFrom: []string{"developer"},
			},
		},
	}

	// Store in global context
	m.roles["_global"] = make(map[string]*Role)
	for _, role := range builtInRoles {
		m.roles["_global"][role.Metadata.Name] = role
	}
}

// CreateTenant creates a new tenant
func (m *Manager) CreateTenant(ctx context.Context, tenant *Tenant) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if tenant.Metadata.Name == "" {
		return fmt.Errorf("tenant name is required")
	}

	if _, exists := m.tenants[tenant.Metadata.Name]; exists {
		return fmt.Errorf("tenant already exists: %s", tenant.Metadata.Name)
	}

	// Set defaults
	if tenant.APIVersion == "" {
		tenant.APIVersion = "platformfoundry.io/v1"
	}
	if tenant.Kind == "" {
		tenant.Kind = "Tenant"
	}
	if tenant.Metadata.ID == "" {
		tenant.Metadata.ID = fmt.Sprintf("tenant-%d", time.Now().UnixNano())
	}
	tenant.Metadata.CreatedAt = time.Now()
	tenant.Metadata.UpdatedAt = time.Now()

	// Initialize status
	tenant.Status = &TenantStatus{
		Phase: TenantPhaseActive,
		Conditions: []TenantCondition{
			{
				Type:               "Ready",
				Status:             "True",
				LastTransitionTime: time.Now(),
				Reason:             "Created",
				Message:            "Tenant created successfully",
			},
		},
	}

	m.tenants[tenant.Metadata.Name] = tenant
	m.roles[tenant.Metadata.Name] = make(map[string]*Role)
	m.bindings[tenant.Metadata.Name] = make(map[string]*RoleBinding)

	return nil
}

// GetTenant retrieves a tenant by name
func (m *Manager) GetTenant(name string) (*Tenant, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	tenant, ok := m.tenants[name]
	if !ok {
		return nil, fmt.Errorf("tenant not found: %s", name)
	}
	return tenant, nil
}

// ListTenants returns all tenants
func (m *Manager) ListTenants() []*Tenant {
	m.mu.RLock()
	defer m.mu.RUnlock()

	result := make([]*Tenant, 0, len(m.tenants))
	for _, t := range m.tenants {
		result = append(result, t)
	}
	return result
}

// UpdateTenant updates a tenant
func (m *Manager) UpdateTenant(ctx context.Context, tenant *Tenant) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, exists := m.tenants[tenant.Metadata.Name]; !exists {
		return fmt.Errorf("tenant not found: %s", tenant.Metadata.Name)
	}

	tenant.Metadata.UpdatedAt = time.Now()
	m.tenants[tenant.Metadata.Name] = tenant
	return nil
}

// DeleteTenant removes a tenant
func (m *Manager) DeleteTenant(ctx context.Context, name string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, exists := m.tenants[name]; !exists {
		return fmt.Errorf("tenant not found: %s", name)
	}

	delete(m.tenants, name)
	delete(m.roles, name)
	delete(m.bindings, name)
	return nil
}

// SuspendTenant suspends a tenant
func (m *Manager) SuspendTenant(ctx context.Context, name, reason string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	tenant, ok := m.tenants[name]
	if !ok {
		return fmt.Errorf("tenant not found: %s", name)
	}

	tenant.Status.Phase = TenantPhaseSuspended
	tenant.Status.Conditions = append(tenant.Status.Conditions, TenantCondition{
		Type:               "Suspended",
		Status:             "True",
		LastTransitionTime: time.Now(),
		Reason:             "ManualSuspension",
		Message:            reason,
	})
	return nil
}

// ActivateTenant reactivates a suspended tenant
func (m *Manager) ActivateTenant(ctx context.Context, name string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	tenant, ok := m.tenants[name]
	if !ok {
		return fmt.Errorf("tenant not found: %s", name)
	}

	tenant.Status.Phase = TenantPhaseActive
	tenant.Status.Conditions = append(tenant.Status.Conditions, TenantCondition{
		Type:               "Ready",
		Status:             "True",
		LastTransitionTime: time.Now(),
		Reason:             "Reactivated",
		Message:            "Tenant reactivated",
	})
	return nil
}

// CreateRole creates a role within a tenant
func (m *Manager) CreateRole(ctx context.Context, tenantName string, role *Role) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, exists := m.tenants[tenantName]; !exists && tenantName != "_global" {
		return fmt.Errorf("tenant not found: %s", tenantName)
	}

	if m.roles[tenantName] == nil {
		m.roles[tenantName] = make(map[string]*Role)
	}

	role.Metadata.Tenant = tenantName
	m.roles[tenantName][role.Metadata.Name] = role
	return nil
}

// GetRole retrieves a role
func (m *Manager) GetRole(tenantName, roleName string) (*Role, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	// Check tenant-specific roles first
	if roles, ok := m.roles[tenantName]; ok {
		if role, ok := roles[roleName]; ok {
			return role, nil
		}
	}

	// Fall back to global roles
	if roles, ok := m.roles["_global"]; ok {
		if role, ok := roles[roleName]; ok {
			return role, nil
		}
	}

	return nil, fmt.Errorf("role not found: %s", roleName)
}

// ListRoles returns roles for a tenant
func (m *Manager) ListRoles(tenantName string) []*Role {
	m.mu.RLock()
	defer m.mu.RUnlock()

	result := make([]*Role, 0)

	// Add tenant-specific roles
	if roles, ok := m.roles[tenantName]; ok {
		for _, r := range roles {
			result = append(result, r)
		}
	}

	// Add global roles
	if roles, ok := m.roles["_global"]; ok {
		for _, r := range roles {
			result = append(result, r)
		}
	}

	return result
}

// CreateRoleBinding creates a role binding
func (m *Manager) CreateRoleBinding(ctx context.Context, tenantName string, binding *RoleBinding) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, exists := m.tenants[tenantName]; !exists {
		return fmt.Errorf("tenant not found: %s", tenantName)
	}

	if m.bindings[tenantName] == nil {
		m.bindings[tenantName] = make(map[string]*RoleBinding)
	}

	binding.Metadata.Tenant = tenantName
	m.bindings[tenantName][binding.Metadata.Name] = binding
	return nil
}

// ListRoleBindings returns bindings for a tenant
func (m *Manager) ListRoleBindings(tenantName string) []*RoleBinding {
	m.mu.RLock()
	defer m.mu.RUnlock()

	result := make([]*RoleBinding, 0)
	if bindings, ok := m.bindings[tenantName]; ok {
		for _, b := range bindings {
			result = append(result, b)
		}
	}
	return result
}

// CheckAccess verifies if a subject has permission
func (m *Manager) CheckAccess(ctx context.Context, tenantName, subjectName, resource, verb string, environment string) (bool, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	// Find all bindings for this subject in the tenant
	bindings, ok := m.bindings[tenantName]
	if !ok {
		return false, nil
	}

	for _, binding := range bindings {
		// Check if subject matches
		subjectMatch := false
		for _, subject := range binding.Spec.Subjects {
			if subject.Name == subjectName {
				subjectMatch = true
				break
			}
		}
		if !subjectMatch {
			continue
		}

		// Get the role
		role, err := m.GetRole(tenantName, binding.Spec.RoleRef.Name)
		if err != nil {
			continue
		}

		// Check permissions
		for _, perm := range role.Spec.Permissions {
			resourceMatch := false
			for _, r := range perm.Resources {
				if r == "*" || r == resource {
					resourceMatch = true
					break
				}
			}
			if !resourceMatch {
				continue
			}

			verbMatch := false
			for _, v := range perm.Verbs {
				if v == "*" || v == verb {
					verbMatch = true
					break
				}
			}
			if !verbMatch {
				continue
			}

			// Check environment restriction
			if len(perm.Environments) > 0 && environment != "" {
				envMatch := false
				for _, e := range perm.Environments {
					if e == environment {
						envMatch = true
						break
					}
				}
				if !envMatch {
					continue
				}
			}

			return true, nil
		}
	}

	return false, nil
}

// RequestJITAccess creates a just-in-time access request
func (m *Manager) RequestJITAccess(ctx context.Context, request *JITAccessRequest) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, exists := m.tenants[request.Tenant]; !exists {
		return fmt.Errorf("tenant not found: %s", request.Tenant)
	}

	request.ID = fmt.Sprintf("jit-%d", time.Now().UnixNano())
	request.Status = JITStatusPending
	request.RequestedAt = time.Now()

	m.jitRequests[request.ID] = request
	return nil
}

// ApproveJITAccess approves a JIT request
func (m *Manager) ApproveJITAccess(ctx context.Context, requestID, approver string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	request, ok := m.jitRequests[requestID]
	if !ok {
		return fmt.Errorf("JIT request not found: %s", requestID)
	}

	if request.Status != JITStatusPending {
		return fmt.Errorf("JIT request is not pending: %s", request.Status)
	}

	now := time.Now()
	expires := now.Add(request.Duration)

	request.Status = JITStatusApproved
	request.ApprovedBy = approver
	request.ApprovedAt = &now
	request.ExpiresAt = &expires

	// Create temporary role binding
	binding := &RoleBinding{
		APIVersion: "platformfoundry.io/v1",
		Kind:       "RoleBinding",
		Metadata: RoleBindingMetadata{
			Name:   fmt.Sprintf("jit-%s", request.ID),
			Tenant: request.Tenant,
			Labels: map[string]string{
				"jit-request": request.ID,
				"expires":     expires.Format(time.RFC3339),
			},
		},
		Spec: RoleBindingSpec{
			RoleRef: RoleRef{
				Kind: "Role",
				Name: request.Role,
			},
			Subjects: []Subject{
				{
					Kind: "User",
					Name: request.Requester,
				},
			},
		},
	}

	if m.bindings[request.Tenant] == nil {
		m.bindings[request.Tenant] = make(map[string]*RoleBinding)
	}
	m.bindings[request.Tenant][binding.Metadata.Name] = binding

	return nil
}

// DenyJITAccess denies a JIT request
func (m *Manager) DenyJITAccess(ctx context.Context, requestID, denier, reason string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	request, ok := m.jitRequests[requestID]
	if !ok {
		return fmt.Errorf("JIT request not found: %s", requestID)
	}

	request.Status = JITStatusDenied
	return nil
}

// ListJITRequests returns JIT requests for a tenant
func (m *Manager) ListJITRequests(tenantName string, status *JITStatus) []*JITAccessRequest {
	m.mu.RLock()
	defer m.mu.RUnlock()

	result := make([]*JITAccessRequest, 0)
	for _, req := range m.jitRequests {
		if req.Tenant != tenantName {
			continue
		}
		if status != nil && req.Status != *status {
			continue
		}
		result = append(result, req)
	}
	return result
}

// CreateAccessReview creates an access review
func (m *Manager) CreateAccessReview(ctx context.Context, review *AccessReview) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	review.ID = fmt.Sprintf("review-%d", time.Now().UnixNano())
	review.Status = ReviewStatusPending
	review.StartedAt = time.Now()

	// Populate entries from current bindings
	review.Entries = make([]ReviewEntry, 0)
	if bindings, ok := m.bindings[review.Tenant]; ok {
		for _, binding := range bindings {
			for _, subject := range binding.Spec.Subjects {
				review.Entries = append(review.Entries, ReviewEntry{
					Subject: subject,
					Role:    binding.Spec.RoleRef.Name,
				})
			}
		}
	}

	m.accessReviews[review.ID] = review
	return nil
}

// GetAccessReview retrieves an access review
func (m *Manager) GetAccessReview(reviewID string) (*AccessReview, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	review, ok := m.accessReviews[reviewID]
	if !ok {
		return nil, fmt.Errorf("access review not found: %s", reviewID)
	}
	return review, nil
}

// GetResourceUsage calculates resource usage for a tenant
func (m *Manager) GetResourceUsage(ctx context.Context, tenantName string) (*ResourceUsage, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	tenant, ok := m.tenants[tenantName]
	if !ok {
		return nil, fmt.Errorf("tenant not found: %s", tenantName)
	}

	// In a real implementation, this would query actual resource usage
	// For now, return mock data
	usage := &ResourceUsage{
		CPU:            "250m",
		CPUPercent:     25.0,
		Memory:         "512Mi",
		MemoryPercent:  25.0,
		Storage:        "10Gi",
		StoragePercent: 10.0,
		Pods:           15,
		PodsPercent:    15.0,
		UpdatedAt:      time.Now(),
	}

	tenant.Status.ResourceUsage = usage
	return usage, nil
}

// CheckQuota verifies if a resource request is within quota
func (m *Manager) CheckQuota(ctx context.Context, tenantName string, resourceType string, requested int) (bool, string, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	tenant, ok := m.tenants[tenantName]
	if !ok {
		return false, "", fmt.Errorf("tenant not found: %s", tenantName)
	}

	if tenant.Spec.Quotas == nil {
		// No quotas set, allow
		return true, "", nil
	}

	// Check specific quota based on resource type
	// In a real implementation, this would compare against actual usage
	return true, "", nil
}
