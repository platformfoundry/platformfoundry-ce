package promise

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/platformfoundry/pf-ce/pkg/types"
)

// Manager handles promise lifecycle management
type Manager struct {
	promises  map[string]*types.Promise
	instances map[string]*types.PromiseInstance
	requests  map[string]*types.PromiseRequest
	mu        sync.RWMutex

	// Callbacks for external integration
	onProvision func(ctx context.Context, req *types.PromiseRequest, promise *types.Promise) (map[string]interface{}, error)
	onDelete    func(ctx context.Context, instance *types.PromiseInstance) error
	onApproval  func(ctx context.Context, req *types.PromiseRequest) error
}

// NewManager creates a new promise manager
func NewManager() *Manager {
	return &Manager{
		promises:  make(map[string]*types.Promise),
		instances: make(map[string]*types.PromiseInstance),
		requests:  make(map[string]*types.PromiseRequest),
	}
}

// SetProvisionCallback sets the callback for provisioning resources
func (m *Manager) SetProvisionCallback(cb func(ctx context.Context, req *types.PromiseRequest, promise *types.Promise) (map[string]interface{}, error)) {
	m.onProvision = cb
}

// SetDeleteCallback sets the callback for deleting resources
func (m *Manager) SetDeleteCallback(cb func(ctx context.Context, instance *types.PromiseInstance) error) {
	m.onDelete = cb
}

// SetApprovalCallback sets the callback for approval requests
func (m *Manager) SetApprovalCallback(cb func(ctx context.Context, req *types.PromiseRequest) error) {
	m.onApproval = cb
}

// RegisterPromise registers a new promise
func (m *Manager) RegisterPromise(promise *types.Promise) error {
	if err := promise.Validate(); err != nil {
		return fmt.Errorf("invalid promise: %w", err)
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	m.promises[promise.Metadata.Name] = promise
	return nil
}

// UnregisterPromise removes a promise
func (m *Manager) UnregisterPromise(name string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, exists := m.promises[name]; !exists {
		return fmt.Errorf("promise not found: %s", name)
	}

	// Check for active instances
	for _, inst := range m.instances {
		if inst.Promise == name && inst.State != types.PromiseRequestStateDeleted {
			return fmt.Errorf("cannot unregister promise with active instances")
		}
	}

	delete(m.promises, name)
	return nil
}

// GetPromise returns a promise by name
func (m *Manager) GetPromise(name string) (*types.Promise, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	promise, exists := m.promises[name]
	if !exists {
		return nil, fmt.Errorf("promise not found: %s", name)
	}

	return promise, nil
}

// ListPromises returns all registered promises
func (m *Manager) ListPromises() []*types.Promise {
	m.mu.RLock()
	defer m.mu.RUnlock()

	result := make([]*types.Promise, 0, len(m.promises))
	for _, p := range m.promises {
		result = append(result, p)
	}
	return result
}

// ListPromisesByCategory returns promises filtered by category
func (m *Manager) ListPromisesByCategory(category string) []*types.Promise {
	m.mu.RLock()
	defer m.mu.RUnlock()

	result := []*types.Promise{}
	for _, p := range m.promises {
		if p.Spec.Category == category {
			result = append(result, p)
		}
	}
	return result
}

// Request creates a new promise request
func (m *Manager) Request(ctx context.Context, req *types.PromiseRequest) (*types.PromiseRequest, error) {
	// Validate basic request structure
	if err := req.Validate(); err != nil {
		return nil, fmt.Errorf("invalid request: %w", err)
	}

	// Get the promise
	promise, err := m.GetPromise(req.Spec.Promise)
	if err != nil {
		return nil, err
	}

	// Apply defaults
	req.ApplyDefaults(promise)

	// Validate inputs against promise
	if err := req.ValidateInputs(promise); err != nil {
		return nil, fmt.Errorf("input validation failed: %w", err)
	}

	// Initialize status
	now := time.Now()
	req.Status = &types.PromiseRequestStatus{
		State:     types.PromiseRequestStatePending,
		CreatedAt: now,
		UpdatedAt: now,
	}

	// Check if approval is needed
	if req.NeedsApproval(promise) {
		req.Status.State = types.PromiseRequestStateAwaitingApproval
		req.Status.ApprovalInfo = &types.ApprovalInfo{
			Required:    true,
			RequestedAt: &now,
		}
		req.Status.Message = "Awaiting approval"

		// Trigger approval callback if set
		if m.onApproval != nil {
			if err := m.onApproval(ctx, req); err != nil {
				return nil, fmt.Errorf("failed to initiate approval: %w", err)
			}
		}
	}

	// Store the request
	m.mu.Lock()
	m.requests[req.Metadata.Name] = req
	m.mu.Unlock()

	// If no approval needed, provision immediately
	if req.Status.State == types.PromiseRequestStatePending {
		return m.provision(ctx, req, promise)
	}

	return req, nil
}

// Approve approves a pending request
func (m *Manager) Approve(ctx context.Context, requestName, approver, reason string) (*types.PromiseRequest, error) {
	m.mu.Lock()
	req, exists := m.requests[requestName]
	if !exists {
		m.mu.Unlock()
		return nil, fmt.Errorf("request not found: %s", requestName)
	}
	m.mu.Unlock()

	if req.Status.State != types.PromiseRequestStateAwaitingApproval {
		return nil, fmt.Errorf("request is not awaiting approval (current state: %s)", req.Status.State)
	}

	// Get the promise
	promise, err := m.GetPromise(req.Spec.Promise)
	if err != nil {
		return nil, err
	}

	// Update approval info
	now := time.Now()
	req.Status.ApprovalInfo.ApprovedAt = &now
	req.Status.ApprovalInfo.ApprovedBy = approver
	req.Status.ApprovalInfo.Reason = reason
	req.Status.State = types.PromiseRequestStateApproved
	req.Status.UpdatedAt = now
	req.Status.Message = fmt.Sprintf("Approved by %s", approver)

	// Provision
	return m.provision(ctx, req, promise)
}

// Reject rejects a pending request
func (m *Manager) Reject(ctx context.Context, requestName, rejector, reason string) (*types.PromiseRequest, error) {
	m.mu.Lock()
	req, exists := m.requests[requestName]
	if !exists {
		m.mu.Unlock()
		return nil, fmt.Errorf("request not found: %s", requestName)
	}
	m.mu.Unlock()

	if req.Status.State != types.PromiseRequestStateAwaitingApproval {
		return nil, fmt.Errorf("request is not awaiting approval (current state: %s)", req.Status.State)
	}

	// Update status
	now := time.Now()
	req.Status.ApprovalInfo.RejectedAt = &now
	req.Status.ApprovalInfo.RejectedBy = rejector
	req.Status.ApprovalInfo.Reason = reason
	req.Status.State = types.PromiseRequestStateRejected
	req.Status.UpdatedAt = now
	req.Status.Message = fmt.Sprintf("Rejected by %s: %s", rejector, reason)

	return req, nil
}

// provision performs the actual provisioning
func (m *Manager) provision(ctx context.Context, req *types.PromiseRequest, promise *types.Promise) (*types.PromiseRequest, error) {
	now := time.Now()
	req.Status.State = types.PromiseRequestStateProvisioning
	req.Status.UpdatedAt = now
	req.Status.Message = "Provisioning resources..."

	var outputs map[string]interface{}
	var err error

	if m.onProvision != nil {
		outputs, err = m.onProvision(ctx, req, promise)
		if err != nil {
			req.Status.State = types.PromiseRequestStateFailed
			req.Status.Message = fmt.Sprintf("Provisioning failed: %v", err)
			req.Status.UpdatedAt = time.Now()
			return req, fmt.Errorf("provisioning failed: %w", err)
		}
	} else {
		// Generate sample outputs
		outputs = m.generateSampleOutputs(promise)
	}

	// Update status to ready
	completedAt := time.Now()
	req.Status.State = types.PromiseRequestStateReady
	req.Status.Outputs = outputs
	req.Status.CompletedAt = &completedAt
	req.Status.UpdatedAt = completedAt
	req.Status.Message = "Ready"

	// Create instance record
	instance := &types.PromiseInstance{
		Name:        req.Metadata.Name,
		Promise:     req.Spec.Promise,
		Team:        req.Metadata.Team,
		Environment: req.Metadata.Environment,
		Inputs:      req.Spec.Inputs,
		Outputs:     outputs,
		State:       types.PromiseRequestStateReady,
		CreatedAt:   req.Status.CreatedAt,
		UpdatedAt:   completedAt,
	}

	m.mu.Lock()
	m.instances[req.Metadata.Name] = instance
	m.mu.Unlock()

	return req, nil
}

// generateSampleOutputs generates sample outputs based on promise definition
func (m *Manager) generateSampleOutputs(promise *types.Promise) map[string]interface{} {
	outputs := make(map[string]interface{})

	for _, out := range promise.Spec.Outputs {
		switch out.Type {
		case "string":
			outputs[out.Name] = fmt.Sprintf("<%s-value>", out.Name)
		case "number":
			outputs[out.Name] = 0
		case "secret":
			outputs[out.Name] = "<secret>"
		}
	}

	return outputs
}

// Delete deletes a promise instance
func (m *Manager) Delete(ctx context.Context, instanceName string) error {
	m.mu.Lock()
	instance, exists := m.instances[instanceName]
	if !exists {
		m.mu.Unlock()
		return fmt.Errorf("instance not found: %s", instanceName)
	}

	instance.State = types.PromiseRequestStateDeleting
	instance.UpdatedAt = time.Now()
	m.mu.Unlock()

	// Trigger delete callback if set
	if m.onDelete != nil {
		if err := m.onDelete(ctx, instance); err != nil {
			m.mu.Lock()
			instance.State = types.PromiseRequestStateFailed
			instance.UpdatedAt = time.Now()
			m.mu.Unlock()
			return fmt.Errorf("delete failed: %w", err)
		}
	}

	// Mark as deleted
	m.mu.Lock()
	instance.State = types.PromiseRequestStateDeleted
	instance.UpdatedAt = time.Now()

	// Also update the request if it exists
	if req, exists := m.requests[instanceName]; exists {
		req.Status.State = types.PromiseRequestStateDeleted
		req.Status.UpdatedAt = time.Now()
		req.Status.Message = "Deleted"
	}
	m.mu.Unlock()

	return nil
}

// GetRequest returns a request by name
func (m *Manager) GetRequest(name string) (*types.PromiseRequest, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	req, exists := m.requests[name]
	if !exists {
		return nil, fmt.Errorf("request not found: %s", name)
	}

	return req, nil
}

// GetInstance returns an instance by name
func (m *Manager) GetInstance(name string) (*types.PromiseInstance, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	instance, exists := m.instances[name]
	if !exists {
		return nil, fmt.Errorf("instance not found: %s", name)
	}

	return instance, nil
}

// ListInstances returns all instances, optionally filtered
func (m *Manager) ListInstances(filters InstanceFilters) []*types.PromiseInstance {
	m.mu.RLock()
	defer m.mu.RUnlock()

	result := []*types.PromiseInstance{}
	for _, inst := range m.instances {
		if filters.Match(inst) {
			result = append(result, inst)
		}
	}
	return result
}

// ListRequests returns all requests, optionally filtered
func (m *Manager) ListRequests(filters RequestFilters) []*types.PromiseRequest {
	m.mu.RLock()
	defer m.mu.RUnlock()

	result := []*types.PromiseRequest{}
	for _, req := range m.requests {
		if filters.Match(req) {
			result = append(result, req)
		}
	}
	return result
}

// InstanceFilters defines filters for listing instances
type InstanceFilters struct {
	Promise     string
	Team        string
	Environment string
	State       types.PromiseRequestState
}

// Match checks if an instance matches the filters
func (f InstanceFilters) Match(inst *types.PromiseInstance) bool {
	if f.Promise != "" && inst.Promise != f.Promise {
		return false
	}
	if f.Team != "" && inst.Team != f.Team {
		return false
	}
	if f.Environment != "" && inst.Environment != f.Environment {
		return false
	}
	if f.State != "" && inst.State != f.State {
		return false
	}
	return true
}

// RequestFilters defines filters for listing requests
type RequestFilters struct {
	Promise     string
	Team        string
	Environment string
	State       types.PromiseRequestState
}

// Match checks if a request matches the filters
func (f RequestFilters) Match(req *types.PromiseRequest) bool {
	if f.Promise != "" && req.Spec.Promise != f.Promise {
		return false
	}
	if f.Team != "" && req.Metadata.Team != f.Team {
		return false
	}
	if f.Environment != "" && req.Metadata.Environment != f.Environment {
		return false
	}
	if f.State != "" && req.Status.State != f.State {
		return false
	}
	return true
}

// PromiseStats contains statistics about promises
type PromiseStats struct {
	TotalPromises   int            `json:"totalPromises"`
	TotalInstances  int            `json:"totalInstances"`
	TotalRequests   int            `json:"totalRequests"`
	ByCategory      map[string]int `json:"byCategory"`
	ByState         map[string]int `json:"byState"`
	PendingApproval int            `json:"pendingApproval"`
}

// GetStats returns statistics about promises
func (m *Manager) GetStats() *PromiseStats {
	m.mu.RLock()
	defer m.mu.RUnlock()

	stats := &PromiseStats{
		TotalPromises:  len(m.promises),
		TotalInstances: len(m.instances),
		TotalRequests:  len(m.requests),
		ByCategory:     make(map[string]int),
		ByState:        make(map[string]int),
	}

	for _, p := range m.promises {
		stats.ByCategory[p.Spec.Category]++
	}

	for _, inst := range m.instances {
		stats.ByState[string(inst.State)]++
	}

	for _, req := range m.requests {
		if req.Status.State == types.PromiseRequestStateAwaitingApproval {
			stats.PendingApproval++
		}
	}

	return stats
}

// LoadBuiltinPromises loads the default built-in promises
func (m *Manager) LoadBuiltinPromises() error {
	builtins := getBuiltinPromises()
	for _, p := range builtins {
		if err := m.RegisterPromise(p); err != nil {
			return fmt.Errorf("failed to register builtin promise %s: %w", p.Metadata.Name, err)
		}
	}
	return nil
}

// getBuiltinPromises returns the default promises
func getBuiltinPromises() []*types.Promise {
	return []*types.Promise{
		{
			APIVersion: "platformfoundry.io/v1",
			Kind:       "Promise",
			Metadata: types.PromiseMetadata{
				Name:        "postgresql-database",
				Description: "Production-ready PostgreSQL database with backups and monitoring",
				Labels: map[string]string{
					"tier": "production-ready",
				},
			},
			Spec: types.PromiseSpec{
				Description: "Production-ready PostgreSQL database with automated backups, monitoring, and high availability options",
				Provider:    "terraform-aws",
				Category:    "database",
				Inputs: []types.PromiseInput{
					{Name: "name", Type: "string", Description: "Database name", Required: true, Validation: "^[a-z][a-z0-9-]{2,20}$"},
					{Name: "size", Type: "enum", Description: "Database size", Required: true, Enum: []string{"small", "medium", "large", "xlarge"}, Default: "medium"},
					{Name: "version", Type: "enum", Description: "PostgreSQL version", Required: false, Enum: []string{"13", "14", "15", "16"}, Default: "15"},
					{Name: "backup_retention", Type: "number", Description: "Backup retention in days", Required: false, Default: float64(7)},
					{Name: "multi_az", Type: "boolean", Description: "Enable multi-AZ deployment", Required: false, Default: false},
				},
				Outputs: []types.PromiseOutput{
					{Name: "connection_string", Type: "secret", Description: "Database connection string"},
					{Name: "host", Type: "string", Description: "Database hostname"},
					{Name: "port", Type: "number", Description: "Database port"},
					{Name: "readonly_endpoint", Type: "string", Description: "Read replica endpoint"},
				},
				Policies: []string{"require-team-label", "cost-limit-database"},
				Approval: &types.PromiseApproval{
					Required:     true,
					Policy:       "database-provisioning",
					Environments: []string{"production"},
				},
			},
		},
		{
			APIVersion: "platformfoundry.io/v1",
			Kind:       "Promise",
			Metadata: types.PromiseMetadata{
				Name:        "redis-cache",
				Description: "Managed Redis cache for high-performance caching",
			},
			Spec: types.PromiseSpec{
				Description: "Managed Redis cache with automatic failover and persistence options",
				Provider:    "terraform-aws",
				Category:    "cache",
				Inputs: []types.PromiseInput{
					{Name: "name", Type: "string", Description: "Cache name", Required: true, Validation: "^[a-z][a-z0-9-]{2,20}$"},
					{Name: "size", Type: "enum", Description: "Cache size", Required: true, Enum: []string{"small", "medium", "large"}, Default: "small"},
					{Name: "version", Type: "enum", Description: "Redis version", Required: false, Enum: []string{"6", "7"}, Default: "7"},
					{Name: "cluster_mode", Type: "boolean", Description: "Enable cluster mode", Required: false, Default: false},
				},
				Outputs: []types.PromiseOutput{
					{Name: "endpoint", Type: "string", Description: "Cache endpoint"},
					{Name: "port", Type: "number", Description: "Cache port"},
					{Name: "auth_token", Type: "secret", Description: "Authentication token"},
				},
			},
		},
		{
			APIVersion: "platformfoundry.io/v1",
			Kind:       "Promise",
			Metadata: types.PromiseMetadata{
				Name:        "s3-bucket",
				Description: "S3 bucket with encryption and versioning",
			},
			Spec: types.PromiseSpec{
				Description: "S3 bucket with server-side encryption and optional versioning",
				Provider:    "terraform-aws",
				Category:    "storage",
				Inputs: []types.PromiseInput{
					{Name: "name", Type: "string", Description: "Bucket name suffix", Required: true, Validation: "^[a-z][a-z0-9-]{2,30}$"},
					{Name: "versioning", Type: "boolean", Description: "Enable versioning", Required: false, Default: false},
					{Name: "lifecycle_days", Type: "number", Description: "Days before transitioning to cheaper storage", Required: false, Default: float64(90)},
				},
				Outputs: []types.PromiseOutput{
					{Name: "bucket_name", Type: "string", Description: "Full bucket name"},
					{Name: "bucket_arn", Type: "string", Description: "Bucket ARN"},
					{Name: "bucket_url", Type: "string", Description: "Bucket URL"},
				},
			},
		},
		{
			APIVersion: "platformfoundry.io/v1",
			Kind:       "Promise",
			Metadata: types.PromiseMetadata{
				Name:        "kafka-topic",
				Description: "Kafka topic on managed MSK cluster",
			},
			Spec: types.PromiseSpec{
				Description: "Kafka topic with configurable partitions and retention",
				Provider:    "terraform-aws",
				Category:    "queue",
				Inputs: []types.PromiseInput{
					{Name: "name", Type: "string", Description: "Topic name", Required: true, Validation: "^[a-zA-Z][a-zA-Z0-9._-]{2,50}$"},
					{Name: "partitions", Type: "number", Description: "Number of partitions", Required: false, Default: float64(3)},
					{Name: "replication_factor", Type: "number", Description: "Replication factor", Required: false, Default: float64(2)},
					{Name: "retention_hours", Type: "number", Description: "Message retention in hours", Required: false, Default: float64(168)},
				},
				Outputs: []types.PromiseOutput{
					{Name: "topic_name", Type: "string", Description: "Created topic name"},
					{Name: "bootstrap_servers", Type: "string", Description: "Kafka bootstrap servers"},
				},
			},
		},
		{
			APIVersion: "platformfoundry.io/v1",
			Kind:       "Promise",
			Metadata: types.PromiseMetadata{
				Name:        "dynamodb-table",
				Description: "DynamoDB table with on-demand scaling",
			},
			Spec: types.PromiseSpec{
				Description: "DynamoDB table with configurable billing mode and indexes",
				Provider:    "terraform-aws",
				Category:    "database",
				Inputs: []types.PromiseInput{
					{Name: "name", Type: "string", Description: "Table name", Required: true, Validation: "^[a-zA-Z][a-zA-Z0-9_-]{2,50}$"},
					{Name: "hash_key", Type: "string", Description: "Hash key attribute name", Required: true},
					{Name: "range_key", Type: "string", Description: "Range key attribute name", Required: false},
					{Name: "billing_mode", Type: "enum", Description: "Billing mode", Required: false, Enum: []string{"PAY_PER_REQUEST", "PROVISIONED"}, Default: "PAY_PER_REQUEST"},
				},
				Outputs: []types.PromiseOutput{
					{Name: "table_name", Type: "string", Description: "Created table name"},
					{Name: "table_arn", Type: "string", Description: "Table ARN"},
					{Name: "stream_arn", Type: "string", Description: "DynamoDB Streams ARN"},
				},
			},
		},
	}
}
