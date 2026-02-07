package rbac

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// JITManager manages just-in-time (JIT) access requests
type JITManager struct {
	stateBackend  JITStateBackend
	notifier      JITNotifier
	auditLog      JITAuditLogger
	rbac          *RBAC
	requests      map[string]*JITRequest
	activeGrants  map[string]*JITGrant
	policies      map[string]*JITPolicy
	mu            sync.RWMutex
	cleanupTicker *time.Ticker
}

// JITStateBackend interface for persistence
type JITStateBackend interface {
	Get(ctx context.Context, kind, id string) (interface{}, error)
	Put(ctx context.Context, kind, id string, value interface{}) error
	Delete(ctx context.Context, kind, id string) error
	List(ctx context.Context, kind string) ([]interface{}, error)
}

// JITNotifier interface for notifications
type JITNotifier interface {
	NotifyApprovalRequest(ctx context.Context, request *JITRequest, approvers []string) error
	NotifyRequestApproved(ctx context.Context, request *JITRequest) error
	NotifyRequestDenied(ctx context.Context, request *JITRequest, reason string) error
	NotifyAccessExpiring(ctx context.Context, grant *JITGrant, remainingTime time.Duration) error
	NotifyAccessRevoked(ctx context.Context, grant *JITGrant, reason string) error
}

// JITAuditLogger interface for audit logging
type JITAuditLogger interface {
	Log(ctx context.Context, event JITAuditEvent) error
}

// JITRequest represents a just-in-time access request
type JITRequest struct {
	ID             string            `json:"id"`
	Requester      string            `json:"requester"`
	RequesterEmail string            `json:"requesterEmail,omitempty"`
	Resource       string            `json:"resource"`
	ResourceType   string            `json:"resourceType"`
	Role           string            `json:"role"`
	Justification  string            `json:"justification"`
	Duration       time.Duration     `json:"duration"`
	Status         JITStatus         `json:"status"`
	ApprovedBy     string            `json:"approvedBy,omitempty"`
	DeniedBy       string            `json:"deniedBy,omitempty"`
	DenialReason   string            `json:"denialReason,omitempty"`
	CreatedAt      time.Time         `json:"createdAt"`
	UpdatedAt      time.Time         `json:"updatedAt"`
	ExpiresAt      time.Time         `json:"expiresAt,omitempty"`
	GrantedAt      *time.Time        `json:"grantedAt,omitempty"`
	Labels         map[string]string `json:"labels,omitempty"`
	PolicyID       string            `json:"policyId,omitempty"`
}

// JITStatus represents the status of a JIT request
type JITStatus string

const (
	JITStatusPending   JITStatus = "pending"
	JITStatusApproved  JITStatus = "approved"
	JITStatusDenied    JITStatus = "denied"
	JITStatusExpired   JITStatus = "expired"
	JITStatusRevoked   JITStatus = "revoked"
	JITStatusCancelled JITStatus = "cancelled"
)

// JITGrant represents an active JIT access grant
type JITGrant struct {
	ID          string    `json:"id"`
	RequestID   string    `json:"requestId"`
	User        string    `json:"user"`
	Resource    string    `json:"resource"`
	Role        string    `json:"role"`
	GrantedAt   time.Time `json:"grantedAt"`
	ExpiresAt   time.Time `json:"expiresAt"`
	ApprovedBy  string    `json:"approvedBy"`
	IsActive    bool      `json:"isActive"`
	ExtendCount int       `json:"extendCount"` // Number of times access was extended
}

// JITPolicy defines policies for JIT access
type JITPolicy struct {
	ID                   string        `json:"id" yaml:"id"`
	Name                 string        `json:"name" yaml:"name"`
	Description          string        `json:"description,omitempty" yaml:"description,omitempty"`
	ResourcePattern      string        `json:"resourcePattern" yaml:"resourcePattern"` // Regex pattern for resources
	AllowedRoles         []string      `json:"allowedRoles" yaml:"allowedRoles"`
	MaxDuration          time.Duration `json:"maxDuration" yaml:"maxDuration"`
	MaxExtensions        int           `json:"maxExtensions" yaml:"maxExtensions"`
	RequireJustification bool          `json:"requireJustification" yaml:"requireJustification"`
	RequireApproval      bool          `json:"requireApproval" yaml:"requireApproval"`
	AutoApproveFor       []string      `json:"autoApproveFor,omitempty" yaml:"autoApproveFor,omitempty"` // Roles that get auto-approved
	Approvers            []string      `json:"approvers,omitempty" yaml:"approvers,omitempty"`           // Who can approve
	ApproverGroups       []string      `json:"approverGroups,omitempty" yaml:"approverGroups,omitempty"`
	ExpiryWarningMinutes int           `json:"expiryWarningMinutes" yaml:"expiryWarningMinutes"` // Warn before expiry
	Enabled              bool          `json:"enabled" yaml:"enabled"`
}

// JITAuditEvent represents an audit event for JIT access
type JITAuditEvent struct {
	Type      string                 `json:"type"` // request_created, request_approved, request_denied, access_granted, access_revoked, access_expired
	Timestamp time.Time              `json:"timestamp"`
	Actor     string                 `json:"actor"`
	RequestID string                 `json:"requestId,omitempty"`
	GrantID   string                 `json:"grantId,omitempty"`
	Resource  string                 `json:"resource"`
	Role      string                 `json:"role"`
	Details   map[string]interface{} `json:"details,omitempty"`
}

// NewJITManager creates a new JITManager
func NewJITManager(stateBackend JITStateBackend, notifier JITNotifier, auditLog JITAuditLogger, rbac *RBAC) *JITManager {
	return &JITManager{
		stateBackend: stateBackend,
		notifier:     notifier,
		auditLog:     auditLog,
		rbac:         rbac,
		requests:     make(map[string]*JITRequest),
		activeGrants: make(map[string]*JITGrant),
		policies:     make(map[string]*JITPolicy),
	}
}

// RegisterPolicy registers a JIT policy
func (m *JITManager) RegisterPolicy(ctx context.Context, policy *JITPolicy) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if policy.ID == "" {
		policy.ID = fmt.Sprintf("jit-policy-%s-%d", policy.Name, time.Now().UnixNano())
	}

	m.policies[policy.ID] = policy

	if m.stateBackend != nil {
		return m.stateBackend.Put(ctx, "JITPolicy", policy.ID, policy)
	}

	return nil
}

// RequestAccess creates a new JIT access request
func (m *JITManager) RequestAccess(ctx context.Context, req *JITRequest) error {
	// Validate request
	if req.Requester == "" {
		return fmt.Errorf("requester is required")
	}
	if req.Resource == "" {
		return fmt.Errorf("resource is required")
	}
	if req.Role == "" {
		return fmt.Errorf("role is required")
	}

	// Find matching policy
	policy := m.findMatchingPolicy(req.Resource, req.Role)
	if policy != nil {
		req.PolicyID = policy.ID

		// Validate against policy
		if policy.RequireJustification && req.Justification == "" {
			return fmt.Errorf("justification is required by policy")
		}
		if req.Duration > policy.MaxDuration {
			return fmt.Errorf("requested duration exceeds maximum allowed: %v", policy.MaxDuration)
		}
	}

	// Generate ID and set timestamps
	req.ID = generateJITRequestID()
	req.Status = JITStatusPending
	req.CreatedAt = time.Now()
	req.UpdatedAt = time.Now()

	m.mu.Lock()
	m.requests[req.ID] = req
	m.mu.Unlock()

	// Persist request
	if m.stateBackend != nil {
		if err := m.stateBackend.Put(ctx, "JITRequest", req.ID, req); err != nil {
			return fmt.Errorf("failed to persist request: %w", err)
		}
	}

	// Audit log
	if m.auditLog != nil {
		m.auditLog.Log(ctx, JITAuditEvent{
			Type:      "request_created",
			Timestamp: time.Now(),
			Actor:     req.Requester,
			RequestID: req.ID,
			Resource:  req.Resource,
			Role:      req.Role,
			Details: map[string]interface{}{
				"justification": req.Justification,
				"duration":      req.Duration.String(),
			},
		})
	}

	// Check for auto-approval
	if policy != nil && m.shouldAutoApprove(req, policy) {
		return m.autoApprove(ctx, req)
	}

	// Find approvers and notify
	approvers := m.findApprovers(ctx, req, policy)
	if m.notifier != nil && len(approvers) > 0 {
		if err := m.notifier.NotifyApprovalRequest(ctx, req, approvers); err != nil {
			// Log but don't fail the request
			fmt.Printf("failed to notify approvers: %v\n", err)
		}
	}

	return nil
}

// ApproveAccess approves a JIT access request
func (m *JITManager) ApproveAccess(ctx context.Context, requestID, approver string) error {
	m.mu.Lock()
	req, ok := m.requests[requestID]
	if !ok {
		m.mu.Unlock()
		return fmt.Errorf("request not found: %s", requestID)
	}

	if req.Status != JITStatusPending {
		m.mu.Unlock()
		return fmt.Errorf("request is not pending: %s", req.Status)
	}

	// Update request
	now := time.Now()
	req.Status = JITStatusApproved
	req.ApprovedBy = approver
	req.UpdatedAt = now
	req.GrantedAt = &now
	req.ExpiresAt = now.Add(req.Duration)
	m.mu.Unlock()

	// Grant temporary access
	grant, err := m.grantTemporaryAccess(ctx, req)
	if err != nil {
		return fmt.Errorf("failed to grant access: %w", err)
	}

	// Persist
	if m.stateBackend != nil {
		m.stateBackend.Put(ctx, "JITRequest", requestID, req)
		m.stateBackend.Put(ctx, "JITGrant", grant.ID, grant)
	}

	// Audit log
	if m.auditLog != nil {
		m.auditLog.Log(ctx, JITAuditEvent{
			Type:      "request_approved",
			Timestamp: now,
			Actor:     approver,
			RequestID: req.ID,
			GrantID:   grant.ID,
			Resource:  req.Resource,
			Role:      req.Role,
			Details: map[string]interface{}{
				"expiresAt": req.ExpiresAt,
			},
		})
	}

	// Notify requester
	if m.notifier != nil {
		m.notifier.NotifyRequestApproved(ctx, req)
	}

	// Schedule revocation
	m.scheduleRevocation(grant)

	return nil
}

// DenyAccess denies a JIT access request
func (m *JITManager) DenyAccess(ctx context.Context, requestID, denier, reason string) error {
	m.mu.Lock()
	req, ok := m.requests[requestID]
	if !ok {
		m.mu.Unlock()
		return fmt.Errorf("request not found: %s", requestID)
	}

	if req.Status != JITStatusPending {
		m.mu.Unlock()
		return fmt.Errorf("request is not pending: %s", req.Status)
	}

	req.Status = JITStatusDenied
	req.DeniedBy = denier
	req.DenialReason = reason
	req.UpdatedAt = time.Now()
	m.mu.Unlock()

	// Persist
	if m.stateBackend != nil {
		m.stateBackend.Put(ctx, "JITRequest", requestID, req)
	}

	// Audit log
	if m.auditLog != nil {
		m.auditLog.Log(ctx, JITAuditEvent{
			Type:      "request_denied",
			Timestamp: time.Now(),
			Actor:     denier,
			RequestID: req.ID,
			Resource:  req.Resource,
			Role:      req.Role,
			Details: map[string]interface{}{
				"reason": reason,
			},
		})
	}

	// Notify requester
	if m.notifier != nil {
		m.notifier.NotifyRequestDenied(ctx, req, reason)
	}

	return nil
}

// RevokeAccess revokes an active JIT grant
func (m *JITManager) RevokeAccess(ctx context.Context, grantID, revoker, reason string) error {
	m.mu.Lock()
	grant, ok := m.activeGrants[grantID]
	if !ok {
		m.mu.Unlock()
		return fmt.Errorf("grant not found: %s", grantID)
	}

	if !grant.IsActive {
		m.mu.Unlock()
		return fmt.Errorf("grant is not active")
	}

	grant.IsActive = false
	delete(m.activeGrants, grantID)
	m.mu.Unlock()

	// Remove RBAC access
	if err := m.removeTemporaryAccess(ctx, grant); err != nil {
		return fmt.Errorf("failed to remove access: %w", err)
	}

	// Update request status
	m.mu.Lock()
	if req, ok := m.requests[grant.RequestID]; ok {
		req.Status = JITStatusRevoked
		req.UpdatedAt = time.Now()
	}
	m.mu.Unlock()

	// Persist
	if m.stateBackend != nil {
		m.stateBackend.Put(ctx, "JITGrant", grantID, grant)
	}

	// Audit log
	if m.auditLog != nil {
		m.auditLog.Log(ctx, JITAuditEvent{
			Type:      "access_revoked",
			Timestamp: time.Now(),
			Actor:     revoker,
			RequestID: grant.RequestID,
			GrantID:   grant.ID,
			Resource:  grant.Resource,
			Role:      grant.Role,
			Details: map[string]interface{}{
				"reason": reason,
			},
		})
	}

	// Notify user
	if m.notifier != nil {
		m.notifier.NotifyAccessRevoked(ctx, grant, reason)
	}

	return nil
}

// ExtendAccess extends an active JIT grant
func (m *JITManager) ExtendAccess(ctx context.Context, grantID string, additionalDuration time.Duration, approver string) error {
	m.mu.Lock()
	grant, ok := m.activeGrants[grantID]
	if !ok {
		m.mu.Unlock()
		return fmt.Errorf("grant not found: %s", grantID)
	}

	if !grant.IsActive {
		m.mu.Unlock()
		return fmt.Errorf("grant is not active")
	}

	// Check extension limits
	req := m.requests[grant.RequestID]
	if req != nil && req.PolicyID != "" {
		policy := m.policies[req.PolicyID]
		if policy != nil && grant.ExtendCount >= policy.MaxExtensions {
			m.mu.Unlock()
			return fmt.Errorf("maximum extensions (%d) exceeded", policy.MaxExtensions)
		}
	}

	grant.ExpiresAt = grant.ExpiresAt.Add(additionalDuration)
	grant.ExtendCount++
	m.mu.Unlock()

	// Persist
	if m.stateBackend != nil {
		m.stateBackend.Put(ctx, "JITGrant", grantID, grant)
	}

	// Audit log
	if m.auditLog != nil {
		m.auditLog.Log(ctx, JITAuditEvent{
			Type:      "access_extended",
			Timestamp: time.Now(),
			Actor:     approver,
			GrantID:   grant.ID,
			Resource:  grant.Resource,
			Role:      grant.Role,
			Details: map[string]interface{}{
				"newExpiresAt":   grant.ExpiresAt,
				"extensionCount": grant.ExtendCount,
			},
		})
	}

	// Reschedule revocation
	m.scheduleRevocation(grant)

	return nil
}

// GetRequest returns a JIT request by ID
func (m *JITManager) GetRequest(ctx context.Context, requestID string) (*JITRequest, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	req, ok := m.requests[requestID]
	if !ok {
		return nil, fmt.Errorf("request not found: %s", requestID)
	}

	return req, nil
}

// ListPendingRequests returns all pending JIT requests
func (m *JITManager) ListPendingRequests(ctx context.Context) []*JITRequest {
	m.mu.RLock()
	defer m.mu.RUnlock()

	requests := make([]*JITRequest, 0)
	for _, req := range m.requests {
		if req.Status == JITStatusPending {
			requests = append(requests, req)
		}
	}

	return requests
}

// ListActiveGrants returns all active JIT grants
func (m *JITManager) ListActiveGrants(ctx context.Context) []*JITGrant {
	m.mu.RLock()
	defer m.mu.RUnlock()

	grants := make([]*JITGrant, 0, len(m.activeGrants))
	for _, grant := range m.activeGrants {
		if grant.IsActive {
			grants = append(grants, grant)
		}
	}

	return grants
}

// ListUserGrants returns all active grants for a user
func (m *JITManager) ListUserGrants(ctx context.Context, userID string) []*JITGrant {
	m.mu.RLock()
	defer m.mu.RUnlock()

	grants := make([]*JITGrant, 0)
	for _, grant := range m.activeGrants {
		if grant.User == userID && grant.IsActive {
			grants = append(grants, grant)
		}
	}

	return grants
}

// findMatchingPolicy finds the policy that matches the resource and role
func (m *JITManager) findMatchingPolicy(resource, role string) *JITPolicy {
	m.mu.RLock()
	defer m.mu.RUnlock()

	for _, policy := range m.policies {
		if !policy.Enabled {
			continue
		}

		// Check role match
		roleMatch := false
		for _, r := range policy.AllowedRoles {
			if r == role || r == "*" {
				roleMatch = true
				break
			}
		}
		if !roleMatch {
			continue
		}

		// Check resource pattern match
		if policy.ResourcePattern == "*" {
			return policy
		}
		// Simple pattern matching (can be enhanced with regex)
		if matchPattern(policy.ResourcePattern, resource) {
			return policy
		}
	}

	return nil
}

// shouldAutoApprove checks if a request should be auto-approved
func (m *JITManager) shouldAutoApprove(req *JITRequest, policy *JITPolicy) bool {
	if !policy.RequireApproval {
		return true
	}

	// Check if requester is in auto-approve list
	for _, role := range policy.AutoApproveFor {
		// In a real implementation, check if user has this role
		if role == req.Role {
			return true
		}
	}

	return false
}

// autoApprove automatically approves a request
func (m *JITManager) autoApprove(ctx context.Context, req *JITRequest) error {
	return m.ApproveAccess(ctx, req.ID, "system:auto-approve")
}

// findApprovers finds who can approve a request
func (m *JITManager) findApprovers(ctx context.Context, req *JITRequest, policy *JITPolicy) []string {
	if policy == nil {
		return nil
	}

	approvers := make([]string, 0)
	approvers = append(approvers, policy.Approvers...)
	// In a real implementation, expand approver groups to individual users

	return approvers
}

// grantTemporaryAccess grants temporary RBAC access
func (m *JITManager) grantTemporaryAccess(ctx context.Context, req *JITRequest) (*JITGrant, error) {
	grant := &JITGrant{
		ID:         fmt.Sprintf("grant-%d", time.Now().UnixNano()),
		RequestID:  req.ID,
		User:       req.Requester,
		Resource:   req.Resource,
		Role:       req.Role,
		GrantedAt:  time.Now(),
		ExpiresAt:  req.ExpiresAt,
		ApprovedBy: req.ApprovedBy,
		IsActive:   true,
	}

	m.mu.Lock()
	m.activeGrants[grant.ID] = grant
	m.mu.Unlock()

	// Grant RBAC role if RBAC is configured
	if m.rbac != nil {
		// In a real implementation, add the user to the role for the specific resource
		// This is simplified - actual implementation would need resource-scoped roles
	}

	return grant, nil
}

// removeTemporaryAccess removes temporary RBAC access
func (m *JITManager) removeTemporaryAccess(ctx context.Context, grant *JITGrant) error {
	// In a real implementation, remove the user from the role
	if m.rbac != nil {
		// Remove role assignment
	}
	return nil
}

// scheduleRevocation schedules automatic revocation when grant expires
func (m *JITManager) scheduleRevocation(grant *JITGrant) {
	duration := time.Until(grant.ExpiresAt)
	if duration <= 0 {
		return
	}

	go func() {
		time.Sleep(duration)

		m.mu.Lock()
		if g, ok := m.activeGrants[grant.ID]; ok && g.IsActive {
			g.IsActive = false
			delete(m.activeGrants, grant.ID)
		}
		m.mu.Unlock()

		// Remove access
		ctx := context.Background()
		m.removeTemporaryAccess(ctx, grant)

		// Audit log
		if m.auditLog != nil {
			m.auditLog.Log(ctx, JITAuditEvent{
				Type:      "access_expired",
				Timestamp: time.Now(),
				Actor:     "system",
				GrantID:   grant.ID,
				Resource:  grant.Resource,
				Role:      grant.Role,
			})
		}
	}()
}

// StartCleanup starts a background goroutine to clean up expired requests and grants
func (m *JITManager) StartCleanup(ctx context.Context, interval time.Duration) {
	m.cleanupTicker = time.NewTicker(interval)

	go func() {
		for {
			select {
			case <-ctx.Done():
				m.cleanupTicker.Stop()
				return
			case <-m.cleanupTicker.C:
				m.cleanup(ctx)
			}
		}
	}()
}

// cleanup removes expired requests and grants
func (m *JITManager) cleanup(ctx context.Context) {
	now := time.Now()

	m.mu.Lock()
	defer m.mu.Unlock()

	// Clean up expired grants
	for id, grant := range m.activeGrants {
		if grant.ExpiresAt.Before(now) && grant.IsActive {
			grant.IsActive = false
			delete(m.activeGrants, id)

			// Remove access
			m.removeTemporaryAccess(ctx, grant)

			// Update request status
			if req, ok := m.requests[grant.RequestID]; ok {
				req.Status = JITStatusExpired
				req.UpdatedAt = now
			}
		}
	}

	// Clean up old requests (older than 30 days)
	cutoff := now.Add(-30 * 24 * time.Hour)
	for id, req := range m.requests {
		if req.UpdatedAt.Before(cutoff) && req.Status != JITStatusPending && req.Status != JITStatusApproved {
			delete(m.requests, id)
		}
	}
}

// Helper functions

func generateJITRequestID() string {
	return fmt.Sprintf("jit-%d", time.Now().UnixNano())
}

func matchPattern(pattern, value string) bool {
	// Simple wildcard matching
	if pattern == "*" {
		return true
	}
	if pattern == value {
		return true
	}
	// Add more sophisticated pattern matching as needed
	return false
}

// DefaultJITPolicy returns a default JIT policy
func DefaultJITPolicy() *JITPolicy {
	return &JITPolicy{
		Name:                 "default",
		Description:          "Default JIT access policy",
		ResourcePattern:      "*",
		AllowedRoles:         []string{"viewer", "operator", "admin"},
		MaxDuration:          8 * time.Hour,
		MaxExtensions:        2,
		RequireJustification: true,
		RequireApproval:      true,
		ExpiryWarningMinutes: 30,
		Enabled:              true,
	}
}

// ProductionJITPolicy returns a stricter JIT policy for production
func ProductionJITPolicy() *JITPolicy {
	return &JITPolicy{
		Name:                 "production",
		Description:          "Strict JIT access policy for production resources",
		ResourcePattern:      "*production*",
		AllowedRoles:         []string{"viewer", "operator"},
		MaxDuration:          4 * time.Hour,
		MaxExtensions:        1,
		RequireJustification: true,
		RequireApproval:      true,
		ApproverGroups:       []string{"platform-admins", "security-team"},
		ExpiryWarningMinutes: 15,
		Enabled:              true,
	}
}

// JITSummary provides a summary of JIT access state
type JITSummary struct {
	PendingRequests int            `json:"pendingRequests"`
	ActiveGrants    int            `json:"activeGrants"`
	GrantsByRole    map[string]int `json:"grantsByRole"`
	ExpiringNoon    int            `json:"expiringSoon"` // Expiring in next hour
	GeneratedAt     time.Time      `json:"generatedAt"`
}

// GetSummary returns a summary of JIT access state
func (m *JITManager) GetSummary(ctx context.Context) *JITSummary {
	m.mu.RLock()
	defer m.mu.RUnlock()

	summary := &JITSummary{
		GrantsByRole: make(map[string]int),
		GeneratedAt:  time.Now(),
	}

	// Count pending requests
	for _, req := range m.requests {
		if req.Status == JITStatusPending {
			summary.PendingRequests++
		}
	}

	// Count active grants
	oneHourFromNow := time.Now().Add(time.Hour)
	for _, grant := range m.activeGrants {
		if grant.IsActive {
			summary.ActiveGrants++
			summary.GrantsByRole[grant.Role]++
			if grant.ExpiresAt.Before(oneHourFromNow) {
				summary.ExpiringNoon++
			}
		}
	}

	return summary
}
