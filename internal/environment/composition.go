// Package environment provides environment management with inheritance and composition.
package environment

import (
	"context"
	"fmt"
	"strings"
	"time"
)

// StateBackend interface for state operations
type StateBackend interface {
	Get(ctx context.Context, kind, id string) (interface{}, error)
	List(ctx context.Context, kind string) ([]interface{}, error)
	Put(ctx context.Context, kind, id string, value interface{}) error
}

// Environment represents an environment configuration
type Environment struct {
	Name        string                 `json:"name" yaml:"name"`
	Description string                 `json:"description,omitempty" yaml:"description,omitempty"`
	Inherits    []string               `json:"inherits,omitempty" yaml:"inherits,omitempty"`
	Components  []ComponentRef         `json:"components,omitempty" yaml:"components,omitempty"`
	Resources   map[string]Resource    `json:"resources,omitempty" yaml:"resources,omitempty"`
	Config      map[string]interface{} `json:"config,omitempty" yaml:"config,omitempty"`
	Overrides   map[string]interface{} `json:"overrides,omitempty" yaml:"overrides,omitempty"`
	Labels      map[string]string      `json:"labels,omitempty" yaml:"labels,omitempty"`
	CreatedAt   time.Time              `json:"createdAt" yaml:"createdAt"`
	UpdatedAt   time.Time              `json:"updatedAt" yaml:"updatedAt"`
}

// ComponentRef references a component to include in the environment
type ComponentRef struct {
	Name    string                 `json:"name" yaml:"name"`
	Version string                 `json:"version,omitempty" yaml:"version,omitempty"`
	Config  map[string]interface{} `json:"config,omitempty" yaml:"config,omitempty"`
	Enabled bool                   `json:"enabled" yaml:"enabled"`
}

// Resource represents a resource in an environment
type Resource struct {
	Name      string                 `json:"name" yaml:"name"`
	Type      string                 `json:"type" yaml:"type"`
	Namespace string                 `json:"namespace,omitempty" yaml:"namespace,omitempty"`
	Spec      map[string]interface{} `json:"spec,omitempty" yaml:"spec,omitempty"`
	Labels    map[string]string      `json:"labels,omitempty" yaml:"labels,omitempty"`
}

// Component represents a reusable component definition
type Component struct {
	Name         string                 `json:"name" yaml:"name"`
	Version      string                 `json:"version" yaml:"version"`
	Description  string                 `json:"description,omitempty" yaml:"description,omitempty"`
	Resources    []Resource             `json:"resources" yaml:"resources"`
	Config       map[string]interface{} `json:"config,omitempty" yaml:"config,omitempty"`
	Dependencies []string               `json:"dependencies,omitempty" yaml:"dependencies,omitempty"`
}

// ResolvedEnvironment represents a fully resolved environment with all inheritance applied
type ResolvedEnvironment struct {
	Name           string                 `json:"name"`
	InheritedFrom  []string               `json:"inheritedFrom,omitempty"`
	Resources      map[string]Resource    `json:"resources"`
	Config         map[string]interface{} `json:"config"`
	Labels         map[string]string      `json:"labels"`
	Components     []string               `json:"components,omitempty"`
	ResolutionPath []string               `json:"resolutionPath,omitempty"`
}

// CompositionEngine handles environment inheritance and composition
type CompositionEngine struct {
	stateBackend StateBackend
	components   map[string]*Component
}

// NewCompositionEngine creates a new composition engine
func NewCompositionEngine(backend StateBackend) *CompositionEngine {
	return &CompositionEngine{
		stateBackend: backend,
		components:   make(map[string]*Component),
	}
}

// RegisterComponent registers a component for use in environments
func (e *CompositionEngine) RegisterComponent(component *Component) {
	key := fmt.Sprintf("%s:%s", component.Name, component.Version)
	e.components[key] = component
}

// GetComponent retrieves a registered component
func (e *CompositionEngine) GetComponent(name, version string) (*Component, error) {
	key := fmt.Sprintf("%s:%s", name, version)
	comp, ok := e.components[key]
	if !ok {
		// Try without version
		for k, c := range e.components {
			if strings.HasPrefix(k, name+":") {
				return c, nil
			}
		}
		return nil, fmt.Errorf("component %s not found", name)
	}
	return comp, nil
}

// Resolve resolves an environment specification into a fully resolved environment
func (e *CompositionEngine) Resolve(ctx context.Context, spec *Environment) (*ResolvedEnvironment, error) {
	resolved := &ResolvedEnvironment{
		Name:           spec.Name,
		InheritedFrom:  make([]string, 0),
		Resources:      make(map[string]Resource),
		Config:         make(map[string]interface{}),
		Labels:         make(map[string]string),
		Components:     make([]string, 0),
		ResolutionPath: []string{spec.Name},
	}

	// Apply inheritance chain (in order)
	for _, baseName := range spec.Inherits {
		baseEnv, err := e.getEnvironment(ctx, baseName)
		if err != nil {
			return nil, fmt.Errorf("base environment %s not found: %w", baseName, err)
		}

		// Recursively resolve base environment
		baseResolved, err := e.Resolve(ctx, baseEnv)
		if err != nil {
			return nil, fmt.Errorf("failed to resolve base %s: %w", baseName, err)
		}

		// Merge base into resolved
		e.mergeEnvironments(resolved, baseResolved)
		resolved.InheritedFrom = append(resolved.InheritedFrom, baseName)
		resolved.ResolutionPath = append(baseResolved.ResolutionPath, resolved.ResolutionPath...)
	}

	// Apply components
	for _, compRef := range spec.Components {
		if !compRef.Enabled {
			continue
		}

		comp, err := e.GetComponent(compRef.Name, compRef.Version)
		if err != nil {
			return nil, fmt.Errorf("component %s not found: %w", compRef.Name, err)
		}

		e.addComponent(resolved, comp, compRef.Config)
		resolved.Components = append(resolved.Components, compRef.Name)
	}

	// Apply spec's own resources
	for name, res := range spec.Resources {
		resolved.Resources[name] = res
	}

	// Apply spec's own config
	for k, v := range spec.Config {
		resolved.Config[k] = v
	}

	// Apply spec's own labels
	for k, v := range spec.Labels {
		resolved.Labels[k] = v
	}

	// Apply overrides last
	e.applyOverrides(resolved, spec.Overrides)

	return resolved, nil
}

// getEnvironment retrieves an environment from state backend
func (e *CompositionEngine) getEnvironment(ctx context.Context, name string) (*Environment, error) {
	if e.stateBackend == nil {
		return nil, fmt.Errorf("state backend not configured")
	}

	data, err := e.stateBackend.Get(ctx, "Environment", name)
	if err != nil {
		return nil, err
	}

	env, ok := data.(*Environment)
	if !ok {
		// Try to convert from map
		if m, ok := data.(map[string]interface{}); ok {
			return e.mapToEnvironment(m), nil
		}
		return nil, fmt.Errorf("invalid environment data")
	}

	return env, nil
}

// mapToEnvironment converts a map to Environment struct
func (e *CompositionEngine) mapToEnvironment(m map[string]interface{}) *Environment {
	env := &Environment{
		Resources: make(map[string]Resource),
		Config:    make(map[string]interface{}),
		Labels:    make(map[string]string),
	}

	if name, ok := m["name"].(string); ok {
		env.Name = name
	}
	if inherits, ok := m["inherits"].([]interface{}); ok {
		for _, i := range inherits {
			if s, ok := i.(string); ok {
				env.Inherits = append(env.Inherits, s)
			}
		}
	}
	if config, ok := m["config"].(map[string]interface{}); ok {
		env.Config = config
	}
	if labels, ok := m["labels"].(map[string]string); ok {
		env.Labels = labels
	}

	return env
}

// mergeEnvironments merges source into target
func (e *CompositionEngine) mergeEnvironments(target, source *ResolvedEnvironment) {
	// Merge resources (source overwrites target)
	for name, res := range source.Resources {
		if _, exists := target.Resources[name]; !exists {
			target.Resources[name] = res
		}
	}

	// Merge config (source overwrites target)
	for k, v := range source.Config {
		if _, exists := target.Config[k]; !exists {
			target.Config[k] = v
		}
	}

	// Merge labels (source overwrites target)
	for k, v := range source.Labels {
		if _, exists := target.Labels[k]; !exists {
			target.Labels[k] = v
		}
	}
}

// addComponent adds component resources to environment
func (e *CompositionEngine) addComponent(env *ResolvedEnvironment, comp *Component, config map[string]interface{}) {
	for _, res := range comp.Resources {
		// Apply component config to resource spec
		resolvedRes := res
		if resolvedRes.Spec == nil {
			resolvedRes.Spec = make(map[string]interface{})
		}

		// Merge component-level config
		for k, v := range comp.Config {
			if _, exists := resolvedRes.Spec[k]; !exists {
				resolvedRes.Spec[k] = v
			}
		}

		// Merge instance-level config
		for k, v := range config {
			resolvedRes.Spec[k] = v
		}

		env.Resources[res.Name] = resolvedRes
	}
}

// applyOverrides applies override values to the resolved environment
func (e *CompositionEngine) applyOverrides(env *ResolvedEnvironment, overrides map[string]interface{}) {
	if overrides == nil {
		return
	}

	// Override config values
	if configOverrides, ok := overrides["config"].(map[string]interface{}); ok {
		for k, v := range configOverrides {
			env.Config[k] = v
		}
	}

	// Override resource specs
	if resourceOverrides, ok := overrides["resources"].(map[string]interface{}); ok {
		for resName, resOverride := range resourceOverrides {
			if res, exists := env.Resources[resName]; exists {
				if overrideMap, ok := resOverride.(map[string]interface{}); ok {
					if res.Spec == nil {
						res.Spec = make(map[string]interface{})
					}
					for k, v := range overrideMap {
						res.Spec[k] = v
					}
					env.Resources[resName] = res
				}
			}
		}
	}

	// Override labels
	if labelOverrides, ok := overrides["labels"].(map[string]string); ok {
		for k, v := range labelOverrides {
			env.Labels[k] = v
		}
	}
}

// ValidateInheritance validates that inheritance doesn't have cycles
func (e *CompositionEngine) ValidateInheritance(ctx context.Context, envName string, visited map[string]bool) error {
	if visited == nil {
		visited = make(map[string]bool)
	}

	if visited[envName] {
		return fmt.Errorf("circular inheritance detected: %s", envName)
	}

	visited[envName] = true

	env, err := e.getEnvironment(ctx, envName)
	if err != nil {
		return err
	}

	for _, baseName := range env.Inherits {
		if err := e.ValidateInheritance(ctx, baseName, visited); err != nil {
			return err
		}
	}

	return nil
}

// GetInheritanceChain returns the full inheritance chain for an environment
func (e *CompositionEngine) GetInheritanceChain(ctx context.Context, envName string) ([]string, error) {
	var chain []string
	visited := make(map[string]bool)

	var walk func(name string) error
	walk = func(name string) error {
		if visited[name] {
			return nil
		}
		visited[name] = true

		env, err := e.getEnvironment(ctx, name)
		if err != nil {
			return err
		}

		for _, baseName := range env.Inherits {
			if err := walk(baseName); err != nil {
				return err
			}
		}

		chain = append(chain, name)
		return nil
	}

	if err := walk(envName); err != nil {
		return nil, err
	}

	return chain, nil
}

// Diff compares two environments and returns differences
func (e *CompositionEngine) Diff(env1, env2 *ResolvedEnvironment) *EnvironmentDiff {
	diff := &EnvironmentDiff{
		AddedResources:    make([]string, 0),
		RemovedResources:  make([]string, 0),
		ModifiedResources: make([]string, 0),
		ConfigChanges:     make(map[string]ConfigChange),
	}

	// Check for added/removed resources
	for name := range env2.Resources {
		if _, exists := env1.Resources[name]; !exists {
			diff.AddedResources = append(diff.AddedResources, name)
		}
	}

	for name := range env1.Resources {
		if _, exists := env2.Resources[name]; !exists {
			diff.RemovedResources = append(diff.RemovedResources, name)
		} else {
			// Check if modified (simplified comparison)
			diff.ModifiedResources = append(diff.ModifiedResources, name)
		}
	}

	// Check config changes
	for k, v2 := range env2.Config {
		if v1, exists := env1.Config[k]; exists {
			if fmt.Sprintf("%v", v1) != fmt.Sprintf("%v", v2) {
				diff.ConfigChanges[k] = ConfigChange{Old: v1, New: v2}
			}
		} else {
			diff.ConfigChanges[k] = ConfigChange{New: v2}
		}
	}

	return diff
}

// EnvironmentDiff represents differences between two environments
type EnvironmentDiff struct {
	AddedResources    []string                `json:"addedResources"`
	RemovedResources  []string                `json:"removedResources"`
	ModifiedResources []string                `json:"modifiedResources"`
	ConfigChanges     map[string]ConfigChange `json:"configChanges"`
}

// ConfigChange represents a configuration change
type ConfigChange struct {
	Old interface{} `json:"old,omitempty"`
	New interface{} `json:"new,omitempty"`
}
