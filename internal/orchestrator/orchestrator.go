package orchestrator

import (
	"fmt"

	"github.com/platformfoundry/platformfoundry-ce/internal/environment"
	"github.com/platformfoundry/platformfoundry-ce/internal/plugin"
	"github.com/platformfoundry/platformfoundry-ce/internal/store"
	"github.com/platformfoundry/platformfoundry-ce/pkg/types"
)

// Orchestrator coordinates resource provisioning
type Orchestrator struct {
	pluginManager *plugin.Manager
	store         *store.Store
}

// New creates a new Orchestrator
func New(pm *plugin.Manager, s *store.Store) *Orchestrator {
	return &Orchestrator{
		pluginManager: pm,
		store:         s,
	}
}

// Apply applies resources in dependency order
func (o *Orchestrator) Apply(resources []interface{}) error {
	// Convert to typed resources
	typedResources := make([]types.Resource, 0, len(resources))
	for _, res := range resources {
		switch r := res.(type) {
		case types.Resource:
			typedResources = append(typedResources, r)
		default:
			// Try to handle as generic resource
			if tr, ok := o.convertToResource(res); ok {
				typedResources = append(typedResources, tr)
			} else {
				return fmt.Errorf("unsupported resource type: %T", res)
			}
		}
	}

	// Resolve dependencies and get execution order
	ordered, err := o.resolveDependencies(typedResources)
	if err != nil {
		return err
	}

	// Apply each resource in order
	for _, resource := range ordered {
		if err := o.applyResource(resource); err != nil {
			return fmt.Errorf("failed to apply resource %s: %w", resource.Metadata.Name, err)
		}
	}

	return nil
}

// ApplyPlatform applies a complete platform with all its components
func (o *Orchestrator) ApplyPlatform(platform *types.Platform, env *types.Environment) error {
	// Apply environment overrides if environment specified
	resolvedPlatform := platform
	if env != nil {
		resolver := environment.NewResolver()
		resolved, err := resolver.Resolve(platform, env)
		if err != nil {
			return fmt.Errorf("failed to resolve environment overrides: %w", err)
		}
		resolvedPlatform = resolved
		fmt.Printf("Applying Platform: %s (Environment: %s)\n", resolvedPlatform.Metadata.Name, env.Metadata.Name)
	} else {
		fmt.Printf("Applying Platform: %s\n", resolvedPlatform.Metadata.Name)
	}
	fmt.Println()

	// Apply infrastructure components
	if resolvedPlatform.Spec.Components.Infrastructure != "" {
		if err := o.applyComponentByRef(resolvedPlatform.Spec.Components.Infrastructure, "Infrastructure"); err != nil {
			return fmt.Errorf("failed to apply infrastructure: %w", err)
		}
	}

	// Apply orchestrator components
	if resolvedPlatform.Spec.Components.Orchestrator != "" {
		if err := o.applyComponentByRef(resolvedPlatform.Spec.Components.Orchestrator, "Orchestrator"); err != nil {
			return fmt.Errorf("failed to apply orchestrator: %w", err)
		}
	}

	// Apply observability components
	if resolvedPlatform.Spec.Components.Observability != "" {
		if err := o.applyComponentByRef(resolvedPlatform.Spec.Components.Observability, "Observability"); err != nil {
			return fmt.Errorf("failed to apply observability: %w", err)
		}
	}

	// Apply DevEx components
	if resolvedPlatform.Spec.Components.DevEx != "" {
		if err := o.applyComponentByRef(resolvedPlatform.Spec.Components.DevEx, "DevEx"); err != nil {
			return fmt.Errorf("failed to apply DevEx: %w", err)
		}
	}

	fmt.Println("\nPlatform applied successfully!")
	return nil
}

// applyComponentByRef applies a component by its reference
func (o *Orchestrator) applyComponentByRef(ref string, componentType string) error {
	fmt.Printf("Applying %s component: %s\n", componentType, ref)
	// In a real implementation, we'd fetch the component definition and apply it
	return nil
}

// convertToResource converts interface{} to types.Resource
func (o *Orchestrator) convertToResource(res interface{}) (types.Resource, bool) {
	// Try to access common fields through reflection or type assertion
	type resourceLike interface {
		GetMetadata() types.Metadata
		GetKind() string
		GetSpec() map[string]interface{}
	}

	if r, ok := res.(resourceLike); ok {
		return types.Resource{
			Metadata: r.GetMetadata(),
			Kind:     r.GetKind(),
			Spec:     r.GetSpec(),
		}, true
	}

	return types.Resource{}, false
}

// applyResource applies a single resource
func (o *Orchestrator) applyResource(resource types.Resource) error {
	// Get provider from spec
	provider, ok := resource.Spec["provider"].(string)
	if !ok {
		return fmt.Errorf("resource %s missing provider in spec", resource.Metadata.Name)
	}

	// Get plugin for this resource type and provider
	p, err := o.pluginManager.Get(resource.Kind, provider)
	if err != nil {
		return err
	}

	fmt.Printf("Applying %s/%s (provider: %s)...\n", resource.Kind, resource.Metadata.Name, provider)

	// Validate resource spec
	if err := p.Validate(resource.Spec); err != nil {
		return fmt.Errorf("validation failed: %w", err)
	}

	// Generate plan
	plan, err := p.Plan(resource.Spec)
	if err != nil {
		return fmt.Errorf("planning failed: %w", err)
	}

	fmt.Printf("  Plan:\n")
	for _, action := range plan.Actions {
		fmt.Printf("    - %s\n", action)
	}

	// Apply resource
	result, err := p.Apply(resource.Spec)
	if err != nil {
		return fmt.Errorf("apply failed: %w", err)
	}

	fmt.Printf("  Status: %s - %s\n", result.Status, result.Message)

	// Save state
	if err := o.store.Save(resource.Metadata.Name, resource.Kind, provider, resource.Spec, result.Status); err != nil {
		return fmt.Errorf("failed to save state: %w", err)
	}

	return nil
}

// resolveDependencies orders resources based on dependencies using topological sort
func (o *Orchestrator) resolveDependencies(resources []types.Resource) ([]types.Resource, error) {
	// Use advanced dependency resolver
	resolver := NewDependencyResolver()

	// Add all resources to resolver
	for _, res := range resources {
		resolver.AddResource(res)
	}

	// Resolve dependencies and get execution order
	ordered, err := resolver.Resolve()
	if err != nil {
		return nil, fmt.Errorf("dependency resolution failed: %w", err)
	}

	return ordered, nil
}

// Delete deletes a resource
func (o *Orchestrator) Delete(name, kind string) error {
	// Get resource from state
	state, err := o.store.Get(name)
	if err != nil {
		return fmt.Errorf("resource %s not found in state", name)
	}

	// Get plugin
	p, err := o.pluginManager.Get(state.Kind, state.Provider)
	if err != nil {
		return err
	}

	fmt.Printf("Deleting %s/%s...\n", state.Kind, name)

	// Delete resource
	if err := p.Delete(name); err != nil {
		return fmt.Errorf("delete failed: %w", err)
	}

	// Remove from state
	if err := o.store.Delete(name); err != nil {
		return fmt.Errorf("failed to remove from state: %w", err)
	}

	fmt.Printf("Deleted %s/%s\n", state.Kind, name)

	return nil
}
