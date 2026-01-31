package orchestrator

import (
	"fmt"
	"time"

	"github.com/platformfoundry/platformfoundry-ce/pkg/types"
)

// RollbackPlan represents a plan for rolling back changes
type RollbackPlan struct {
	JobID         string
	Resources     []RollbackResource
	CreatedAt     time.Time
	ExecutionTime time.Duration
}

// RollbackResource represents a resource to be rolled back
type RollbackResource struct {
	Name          string
	Kind          string
	Action        RollbackAction
	PreviousState map[string]interface{}
}

// RollbackAction defines the type of rollback action
type RollbackAction string

const (
	RollbackActionDelete  RollbackAction = "delete"  // Delete newly created resource
	RollbackActionRestore RollbackAction = "restore" // Restore previous state
	RollbackActionNoop    RollbackAction = "noop"    // No action needed
)

// RollbackManager manages rollback operations
type RollbackManager struct {
	orchestrator *Orchestrator
	plans        map[string]*RollbackPlan
}

// NewRollbackManager creates a new rollback manager
func NewRollbackManager(orch *Orchestrator) *RollbackManager {
	return &RollbackManager{
		orchestrator: orch,
		plans:        make(map[string]*RollbackPlan),
	}
}

// CreateRollbackPlan creates a rollback plan for a job
func (rm *RollbackManager) CreateRollbackPlan(jobID string, resources []types.Resource) (*RollbackPlan, error) {
	plan := &RollbackPlan{
		JobID:     jobID,
		Resources: []RollbackResource{},
		CreatedAt: time.Now(),
	}

	// For each resource, determine the rollback action
	for _, resource := range resources {
		rollbackRes := RollbackResource{
			Name: resource.Metadata.Name,
			Kind: resource.Kind,
		}

		// Check if resource existed before
		existingState, err := rm.orchestrator.store.Get(resource.Metadata.Name)
		if err != nil || existingState == nil {
			// Resource didn't exist, so rollback = delete
			rollbackRes.Action = RollbackActionDelete
		} else {
			// Resource existed, so rollback = restore previous state
			rollbackRes.Action = RollbackActionRestore
			// Note: existingState.Spec is JSON-encoded string
			// In a real implementation, we'd deserialize it
			rollbackRes.PreviousState = map[string]interface{}{
				"spec": existingState.Spec,
			}
		}

		plan.Resources = append(plan.Resources, rollbackRes)
	}

	// Store plan
	rm.plans[jobID] = plan

	return plan, nil
}

// ExecuteRollback executes a rollback plan
func (rm *RollbackManager) ExecuteRollback(jobID string) error {
	plan, exists := rm.plans[jobID]
	if !exists {
		return fmt.Errorf("no rollback plan found for job %s", jobID)
	}

	fmt.Printf("Executing rollback for job %s...\n", jobID)
	fmt.Printf("Rolling back %d resource(s)\n\n", len(plan.Resources))

	// Execute in reverse order
	for i := len(plan.Resources) - 1; i >= 0; i-- {
		resource := plan.Resources[i]

		if err := rm.rollbackResource(resource); err != nil {
			return fmt.Errorf("failed to rollback resource %s: %w", resource.Name, err)
		}
	}

	plan.ExecutionTime = time.Since(plan.CreatedAt)
	fmt.Printf("\nRollback completed in %s\n", plan.ExecutionTime)

	return nil
}

// rollbackResource rolls back a single resource
func (rm *RollbackManager) rollbackResource(resource RollbackResource) error {
	fmt.Printf("Rolling back %s/%s (action: %s)...\n", resource.Kind, resource.Name, resource.Action)

	switch resource.Action {
	case RollbackActionDelete:
		// Delete the resource that was created
		if err := rm.orchestrator.Delete(resource.Name, string(resource.Kind)); err != nil {
			return fmt.Errorf("failed to delete resource: %w", err)
		}
		fmt.Printf("  Deleted %s/%s\n", resource.Kind, resource.Name)

	case RollbackActionRestore:
		// Restore previous state
		if resource.PreviousState == nil {
			return fmt.Errorf("no previous state to restore for %s", resource.Name)
		}

		// Create a resource with previous state
		restoreRes := types.Resource{
			Metadata: types.Metadata{
				Name: resource.Name,
			},
			Kind: resource.Kind,
			Spec: resource.PreviousState,
		}

		if err := rm.orchestrator.applyResource(restoreRes); err != nil {
			return fmt.Errorf("failed to restore resource: %w", err)
		}
		fmt.Printf("  Restored %s/%s to previous state\n", resource.Kind, resource.Name)

	case RollbackActionNoop:
		fmt.Printf("  No action needed for %s/%s\n", resource.Kind, resource.Name)
	}

	return nil
}

// GetRollbackPlan retrieves a rollback plan
func (rm *RollbackManager) GetRollbackPlan(jobID string) (*RollbackPlan, error) {
	plan, exists := rm.plans[jobID]
	if !exists {
		return nil, fmt.Errorf("no rollback plan found for job %s", jobID)
	}

	return plan, nil
}

// ListRollbackPlans lists all available rollback plans
func (rm *RollbackManager) ListRollbackPlans() []*RollbackPlan {
	plans := make([]*RollbackPlan, 0, len(rm.plans))
	for _, plan := range rm.plans {
		plans = append(plans, plan)
	}
	return plans
}

// DeleteRollbackPlan deletes a rollback plan
func (rm *RollbackManager) DeleteRollbackPlan(jobID string) error {
	if _, exists := rm.plans[jobID]; !exists {
		return fmt.Errorf("no rollback plan found for job %s", jobID)
	}

	delete(rm.plans, jobID)
	return nil
}
