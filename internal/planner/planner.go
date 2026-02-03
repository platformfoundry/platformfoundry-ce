package planner

import (
	"fmt"

	"github.com/platformfoundry/pf-ce/internal/plugin"
	"github.com/platformfoundry/pf-ce/internal/store"
	"github.com/platformfoundry/pf-ce/pkg/types"
)

// Planner generates execution plans
type Planner struct {
	pluginManager *plugin.Manager
	store         *store.Store
}

// Plan represents an execution plan
type Plan struct {
	ToCreate       []ResourcePlan
	ToUpdate       []ResourcePlan
	ToDelete       []ResourcePlan
	Unchanged      []ResourcePlan
	ExecutionOrder []string
}

// ResourcePlan represents a planned action for a resource
type ResourcePlan struct {
	Name    string
	Kind    string
	Action  string   // "create", "update", "delete", "unchanged"
	Changes []string // List of changes for updates
}

// New creates a new planner
func New(pm *plugin.Manager, st *store.Store) *Planner {
	return &Planner{
		pluginManager: pm,
		store:         st,
	}
}

// CreatePlan generates an execution plan for resources
func (p *Planner) CreatePlan(resources []interface{}) (*Plan, error) {
	plan := &Plan{
		ToCreate:       []ResourcePlan{},
		ToUpdate:       []ResourcePlan{},
		ToDelete:       []ResourcePlan{},
		Unchanged:      []ResourcePlan{},
		ExecutionOrder: []string{},
	}

	// Analyze each resource
	for _, res := range resources {
		switch r := res.(type) {
		case types.Resource:
			if err := p.analyzeResource(&r, plan); err != nil {
				return nil, err
			}
		default:
			// Try to extract common resource interface
			if resource, ok := p.extractResource(res); ok {
				if err := p.analyzeResource(resource, plan); err != nil {
					return nil, err
				}
			}
		}
	}

	// Determine execution order based on dependencies
	plan.ExecutionOrder = p.determineExecutionOrder(plan)

	return plan, nil
}

// analyzeResource analyzes a single resource and updates the plan
func (p *Planner) analyzeResource(res *types.Resource, plan *Plan) error {
	// Check if resource exists in store
	existing, err := p.store.Get(res.Metadata.Name)
	if err != nil || existing == nil {
		// Resource doesn't exist - will be created
		plan.ToCreate = append(plan.ToCreate, ResourcePlan{
			Name:   res.Metadata.Name,
			Kind:   res.Kind,
			Action: "create",
		})
		return nil
	}

	// Resource exists - check for changes
	changes := p.detectChanges(res.Metadata.Name, res)
	if len(changes) > 0 {
		plan.ToUpdate = append(plan.ToUpdate, ResourcePlan{
			Name:    res.Metadata.Name,
			Kind:    res.Kind,
			Action:  "update",
			Changes: changes,
		})
	} else {
		plan.Unchanged = append(plan.Unchanged, ResourcePlan{
			Name:   res.Metadata.Name,
			Kind:   res.Kind,
			Action: "unchanged",
		})
	}

	return nil
}

// detectChanges detects changes between existing and new resource
func (p *Planner) detectChanges(name string, new *types.Resource) []string {
	changes := []string{}

	// Get existing resource from store
	existing, err := p.store.Get(name)
	if err != nil || existing == nil {
		return changes
	}

	// Compare specs - simple string comparison for now
	// In a real implementation, we'd parse and compare structured data
	newSpec := fmt.Sprintf("%v", new.Spec)
	if existing.Spec != newSpec {
		changes = append(changes, "spec updated")
	}

	return changes
}

// determineExecutionOrder determines the order in which resources should be executed
func (p *Planner) determineExecutionOrder(plan *Plan) []string {
	order := []string{}

	// Simple ordering: create first, then update, then delete
	for _, res := range plan.ToCreate {
		order = append(order, res.Name)
	}

	for _, res := range plan.ToUpdate {
		order = append(order, res.Name)
	}

	for _, res := range plan.ToDelete {
		order = append(order, res.Name)
	}

	return order
}

// extractResource attempts to extract a common resource interface from any type
func (p *Planner) extractResource(res interface{}) (*types.Resource, bool) {
	// Try to access common fields
	type resourceLike interface {
		GetName() string
		GetKind() string
	}

	if r, ok := res.(resourceLike); ok {
		return &types.Resource{
			Metadata: types.Metadata{
				Name: r.GetName(),
			},
			Kind: r.GetKind(),
		}, true
	}

	return nil, false
}

// Helper function to compare maps
func mapsEqual(a, b map[string]string) bool {
	if len(a) != len(b) {
		return false
	}

	for k, v := range a {
		if bv, ok := b[k]; !ok || bv != v {
			return false
		}
	}

	return true
}
