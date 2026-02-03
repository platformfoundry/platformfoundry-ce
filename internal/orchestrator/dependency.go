package orchestrator

import (
	"fmt"

	"github.com/platformfoundry/pf-ce/pkg/types"
)

// DependencyResolver resolves resource dependencies
type DependencyResolver struct {
	resources map[string]*types.Resource
	graph     map[string][]string // adjacency list
}

// NewDependencyResolver creates a new dependency resolver
func NewDependencyResolver() *DependencyResolver {
	return &DependencyResolver{
		resources: make(map[string]*types.Resource),
		graph:     make(map[string][]string),
	}
}

// AddResource adds a resource to the dependency graph
func (dr *DependencyResolver) AddResource(resource types.Resource) {
	dr.resources[resource.Metadata.Name] = &resource

	// Extract dependencies from spec
	deps := dr.extractDependencies(&resource)
	dr.graph[resource.Metadata.Name] = deps
}

// extractDependencies extracts dependencies from a resource spec
func (dr *DependencyResolver) extractDependencies(resource *types.Resource) []string {
	deps := []string{}

	// Check for common dependency fields
	if clusterRef, ok := resource.Spec["clusterRef"].(string); ok {
		deps = append(deps, clusterRef)
	}

	if infraRef, ok := resource.Spec["infrastructureRef"].(string); ok {
		deps = append(deps, infraRef)
	}

	// Check for explicit dependencies
	if dependsOn, ok := resource.Spec["dependsOn"].([]interface{}); ok {
		for _, dep := range dependsOn {
			if depStr, ok := dep.(string); ok {
				deps = append(deps, depStr)
			}
		}
	}

	// Resource type specific dependencies
	switch resource.Kind {
	case "Pipeline":
		// Pipelines depend on clusters
		deps = append(deps, dr.findClusterDependency(resource)...)
	case "Application":
		// Applications depend on clusters and pipelines
		deps = append(deps, dr.findClusterDependency(resource)...)
		deps = append(deps, dr.findPipelineDependency(resource)...)
	}

	return deps
}

// findClusterDependency finds cluster dependencies
func (dr *DependencyResolver) findClusterDependency(resource *types.Resource) []string {
	deps := []string{}

	// Check for cluster references in various forms
	if clusterRef, ok := resource.Spec["cluster"].(string); ok {
		deps = append(deps, clusterRef)
	}
	if clusterName, ok := resource.Spec["clusterName"].(string); ok {
		deps = append(deps, clusterName)
	}

	return deps
}

// findPipelineDependency finds pipeline dependencies
func (dr *DependencyResolver) findPipelineDependency(resource *types.Resource) []string {
	deps := []string{}

	if pipelineRef, ok := resource.Spec["pipeline"].(string); ok {
		deps = append(deps, pipelineRef)
	}
	if pipelineName, ok := resource.Spec["pipelineName"].(string); ok {
		deps = append(deps, pipelineName)
	}

	return deps
}

// Resolve performs topological sort to determine execution order
func (dr *DependencyResolver) Resolve() ([]types.Resource, error) {
	// Detect cycles
	if err := dr.detectCycles(); err != nil {
		return nil, err
	}

	// Topological sort using DFS
	visited := make(map[string]bool)
	stack := []string{}

	var visit func(string) error
	visit = func(name string) error {
		if visited[name] {
			return nil
		}

		visited[name] = true

		// Visit dependencies first
		for _, dep := range dr.graph[name] {
			if _, exists := dr.resources[dep]; !exists {
				// Dependency not in current batch - might be in state
				continue
			}
			if err := visit(dep); err != nil {
				return err
			}
		}

		stack = append(stack, name)
		return nil
	}

	// Visit all nodes
	for name := range dr.resources {
		if err := visit(name); err != nil {
			return nil, err
		}
	}

	// Convert to resource list
	result := make([]types.Resource, 0, len(stack))
	for _, name := range stack {
		if res, exists := dr.resources[name]; exists {
			result = append(result, *res)
		}
	}

	return result, nil
}

// detectCycles detects circular dependencies
func (dr *DependencyResolver) detectCycles() error {
	visited := make(map[string]bool)
	recStack := make(map[string]bool)

	var hasCycle func(string) error
	hasCycle = func(name string) error {
		visited[name] = true
		recStack[name] = true

		for _, dep := range dr.graph[name] {
			if _, exists := dr.resources[dep]; !exists {
				continue
			}

			if !visited[dep] {
				if err := hasCycle(dep); err != nil {
					return err
				}
			} else if recStack[dep] {
				return fmt.Errorf("circular dependency detected: %s -> %s", name, dep)
			}
		}

		recStack[name] = false
		return nil
	}

	for name := range dr.resources {
		if !visited[name] {
			if err := hasCycle(name); err != nil {
				return err
			}
		}
	}

	return nil
}

// GetDependencies returns the dependencies of a resource
func (dr *DependencyResolver) GetDependencies(name string) []string {
	return dr.graph[name]
}

// GetDependents returns resources that depend on the given resource
func (dr *DependencyResolver) GetDependents(name string) []string {
	dependents := []string{}

	for resName, deps := range dr.graph {
		for _, dep := range deps {
			if dep == name {
				dependents = append(dependents, resName)
				break
			}
		}
	}

	return dependents
}
