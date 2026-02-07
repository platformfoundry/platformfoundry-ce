package catalog

import (
	"context"
	"fmt"
	"time"
)

// CreateRequest creates a new resource request
func (c *Catalog) CreateRequest(ctx context.Context, req *ResourceRequest) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Validate definition exists
	def, ok := c.definitions[req.DefinitionName]
	if !ok {
		return fmt.Errorf("definition not found: %s", req.DefinitionName)
	}

	// Validate inputs
	errors := c.ValidateInputs(def, req.Inputs)
	if len(errors) > 0 {
		return fmt.Errorf("validation failed: %v", errors)
	}

	// Check quota
	if err := c.checkQuota(ctx, req); err != nil {
		return fmt.Errorf("quota check failed: %w", err)
	}

	// Generate ID
	if req.ID == "" {
		req.ID = fmt.Sprintf("req-%s-%s-%d", req.Application, req.Name, time.Now().UnixNano())
	}

	// Set timestamps
	req.CreatedAt = time.Now()
	req.UpdatedAt = time.Now()

	// Determine if approval is required
	if c.requiresApproval(def, req) {
		req.Status = RequestPendingApproval
	} else {
		req.Status = RequestPending
	}

	// Estimate cost
	req.EstimatedCost = c.estimateCost(def, req.Inputs)

	// Store
	c.requests[req.ID] = req

	if c.stateBackend != nil {
		return c.stateBackend.Put(ctx, "ResourceRequest", req.ID, req)
	}

	return nil
}

// GetRequest returns a request by ID
func (c *Catalog) GetRequest(id string) (*ResourceRequest, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	req, ok := c.requests[id]
	if !ok {
		return nil, fmt.Errorf("request not found: %s", id)
	}
	return req, nil
}

// ListRequests returns all requests, optionally filtered
func (c *Catalog) ListRequests(filters RequestFilters) []*ResourceRequest {
	c.mu.RLock()
	defer c.mu.RUnlock()

	var results []*ResourceRequest
	for _, req := range c.requests {
		if filters.matches(req) {
			results = append(results, req)
		}
	}
	return results
}

// RequestFilters defines filters for listing requests
type RequestFilters struct {
	Team        string
	Application string
	Environment string
	Status      RequestStatus
	RequestedBy string
}

func (f RequestFilters) matches(req *ResourceRequest) bool {
	if f.Team != "" && req.Team != f.Team {
		return false
	}
	if f.Application != "" && req.Application != f.Application {
		return false
	}
	if f.Environment != "" && req.Environment != f.Environment {
		return false
	}
	if f.Status != "" && req.Status != f.Status {
		return false
	}
	if f.RequestedBy != "" && req.RequestedBy != f.RequestedBy {
		return false
	}
	return true
}

// ApproveRequest approves a pending request
func (c *Catalog) ApproveRequest(ctx context.Context, id, approvedBy string) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	req, ok := c.requests[id]
	if !ok {
		return fmt.Errorf("request not found: %s", id)
	}

	if req.Status != RequestPendingApproval {
		return fmt.Errorf("request is not pending approval: %s", req.Status)
	}

	req.Status = RequestApproved
	req.ApprovedBy = approvedBy
	req.UpdatedAt = time.Now()

	if c.stateBackend != nil {
		return c.stateBackend.Put(ctx, "ResourceRequest", req.ID, req)
	}

	return nil
}

// RejectRequest rejects a pending request
func (c *Catalog) RejectRequest(ctx context.Context, id, rejectedBy, reason string) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	req, ok := c.requests[id]
	if !ok {
		return fmt.Errorf("request not found: %s", id)
	}

	if req.Status != RequestPendingApproval {
		return fmt.Errorf("request is not pending approval: %s", req.Status)
	}

	req.Status = RequestRejected
	req.StatusMessage = reason
	req.UpdatedAt = time.Now()

	if c.stateBackend != nil {
		return c.stateBackend.Put(ctx, "ResourceRequest", req.ID, req)
	}

	return nil
}

// ProvisionRequest provisions an approved request
func (c *Catalog) ProvisionRequest(ctx context.Context, id string, provisioner ResourceProvisioner) error {
	c.mu.Lock()
	req, ok := c.requests[id]
	if !ok {
		c.mu.Unlock()
		return fmt.Errorf("request not found: %s", id)
	}

	if req.Status != RequestPending && req.Status != RequestApproved {
		c.mu.Unlock()
		return fmt.Errorf("request cannot be provisioned: %s", req.Status)
	}

	def, ok := c.definitions[req.DefinitionName]
	if !ok {
		c.mu.Unlock()
		return fmt.Errorf("definition not found: %s", req.DefinitionName)
	}

	req.Status = RequestProvisioning
	req.UpdatedAt = time.Now()
	c.mu.Unlock()

	// Provision
	outputs, err := provisioner.Provision(ctx, def, req)

	c.mu.Lock()
	defer c.mu.Unlock()

	if err != nil {
		req.Status = RequestFailed
		req.StatusMessage = err.Error()
		req.UpdatedAt = time.Now()
		if c.stateBackend != nil {
			c.stateBackend.Put(ctx, "ResourceRequest", req.ID, req)
		}
		return err
	}

	now := time.Now()
	req.Status = RequestProvisioned
	req.Outputs = outputs
	req.ProvisionedAt = &now
	req.UpdatedAt = now

	// Update quota usage
	c.updateQuotaUsage(ctx, req, 1)

	if c.stateBackend != nil {
		return c.stateBackend.Put(ctx, "ResourceRequest", req.ID, req)
	}

	return nil
}

// DeleteRequest deletes a provisioned resource
func (c *Catalog) DeleteRequest(ctx context.Context, id string, provisioner ResourceProvisioner) error {
	c.mu.Lock()
	req, ok := c.requests[id]
	if !ok {
		c.mu.Unlock()
		return fmt.Errorf("request not found: %s", id)
	}

	if req.Status != RequestProvisioned {
		c.mu.Unlock()
		return fmt.Errorf("request is not provisioned: %s", req.Status)
	}

	def, ok := c.definitions[req.DefinitionName]
	if !ok {
		c.mu.Unlock()
		return fmt.Errorf("definition not found: %s", req.DefinitionName)
	}

	req.Status = RequestDeleting
	req.UpdatedAt = time.Now()
	c.mu.Unlock()

	// Delete
	err := provisioner.Delete(ctx, def, req)

	c.mu.Lock()
	defer c.mu.Unlock()

	if err != nil {
		req.Status = RequestFailed
		req.StatusMessage = err.Error()
		req.UpdatedAt = time.Now()
		if c.stateBackend != nil {
			c.stateBackend.Put(ctx, "ResourceRequest", req.ID, req)
		}
		return err
	}

	req.Status = RequestDeleted
	req.UpdatedAt = time.Now()

	// Update quota usage
	c.updateQuotaUsage(ctx, req, -1)

	if c.stateBackend != nil {
		return c.stateBackend.Put(ctx, "ResourceRequest", req.ID, req)
	}

	return nil
}

// ResourceProvisioner provisions resources
type ResourceProvisioner interface {
	Provision(ctx context.Context, def *ResourceDefinition, req *ResourceRequest) (map[string]string, error)
	Delete(ctx context.Context, def *ResourceDefinition, req *ResourceRequest) error
}

// checkQuota checks if the request is within quota
func (c *Catalog) checkQuota(ctx context.Context, req *ResourceRequest) error {
	quota := c.getQuota(req.Team, req.Application)
	if quota == nil {
		return nil // No quota defined
	}

	def := c.definitions[req.DefinitionName]
	if def == nil {
		return nil
	}

	// Check resource type limit
	resourceType := def.Spec.Type
	limit, hasLimit := quota.Limits[resourceType]
	if hasLimit {
		used := quota.Used[resourceType]
		if used >= limit {
			return fmt.Errorf("quota exceeded for %s: %d/%d", resourceType, used, limit)
		}
	}

	// Check cost limit
	if quota.CostLimit > 0 {
		estimated := c.estimateCost(def, req.Inputs)
		if estimated != nil && quota.CostUsed+estimated.MonthlyCost > quota.CostLimit {
			return fmt.Errorf("cost limit exceeded: $%.2f + $%.2f > $%.2f",
				quota.CostUsed, estimated.MonthlyCost, quota.CostLimit)
		}
	}

	return nil
}

func (c *Catalog) getQuota(team, project string) *Quota {
	// Try project-specific quota first
	key := fmt.Sprintf("%s/%s", team, project)
	if quota, ok := c.quotas[key]; ok {
		return quota
	}

	// Fall back to team quota
	if quota, ok := c.quotas[team]; ok {
		return quota
	}

	return nil
}

func (c *Catalog) updateQuotaUsage(ctx context.Context, req *ResourceRequest, delta int) {
	quota := c.getQuota(req.Team, req.Application)
	if quota == nil {
		return
	}

	def := c.definitions[req.DefinitionName]
	if def == nil {
		return
	}

	resourceType := def.Spec.Type
	if quota.Used == nil {
		quota.Used = make(map[string]int)
	}
	quota.Used[resourceType] += delta

	// Update cost usage
	if req.EstimatedCost != nil {
		quota.CostUsed += float64(delta) * req.EstimatedCost.MonthlyCost
	}

	if c.stateBackend != nil {
		c.stateBackend.Put(ctx, "Quota", quota.ID, quota)
	}
}

// requiresApproval determines if a request requires approval
func (c *Catalog) requiresApproval(def *ResourceDefinition, req *ResourceRequest) bool {
	// Check policy requirements
	for _, policy := range def.Spec.Policies {
		if policy.Name == "requires-approval" {
			if v, ok := policy.Value.(bool); ok && v {
				return true
			}
		}
	}

	// Check if environment is production
	if req.Environment == "production" || req.Environment == "prod" {
		return true
	}

	// Check estimated cost threshold
	if req.EstimatedCost != nil && req.EstimatedCost.MonthlyCost > 200 {
		return true
	}

	return false
}

// estimateCost estimates the cost of a resource
func (c *Catalog) estimateCost(def *ResourceDefinition, inputs map[string]interface{}) *CostEstimate {
	// Simple cost estimation based on size
	size := "small"
	if s, ok := inputs["size"].(string); ok {
		size = s
	}

	baseCost := 0.0
	switch def.Spec.Type {
	case "postgres", "mysql":
		switch size {
		case "small":
			baseCost = 50.0
		case "medium":
			baseCost = 150.0
		case "large":
			baseCost = 400.0
		}
	case "redis":
		switch size {
		case "small":
			baseCost = 25.0
		case "medium":
			baseCost = 75.0
		case "large":
			baseCost = 200.0
		}
	case "s3":
		baseCost = 5.0 // Base cost, actual depends on usage
	case "mongodb":
		switch size {
		case "M10":
			baseCost = 60.0
		case "M20":
			baseCost = 150.0
		case "M30":
			baseCost = 300.0
		}
	case "rabbitmq":
		baseCost = 30.0
	}

	if baseCost == 0 {
		return nil
	}

	return &CostEstimate{
		MonthlyCost: baseCost,
		HourlyCost:  baseCost / (24 * 30),
		Currency:    "USD",
	}
}

// SetQuota sets a quota for a team or project
func (c *Catalog) SetQuota(ctx context.Context, quota *Quota) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if quota.ID == "" {
		if quota.Project != "" {
			quota.ID = fmt.Sprintf("%s/%s", quota.Team, quota.Project)
		} else {
			quota.ID = quota.Team
		}
	}

	if quota.Used == nil {
		quota.Used = make(map[string]int)
	}

	c.quotas[quota.ID] = quota

	if c.stateBackend != nil {
		return c.stateBackend.Put(ctx, "Quota", quota.ID, quota)
	}

	return nil
}

// GetQuota returns the quota for a team/project
func (c *Catalog) GetQuota(team, project string) *Quota {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.getQuota(team, project)
}
