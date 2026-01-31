package environment

import (
	"fmt"

	"github.com/platformfoundry/platformfoundry-ce/pkg/types"
)

// Resolver handles environment profile resolution and merging
type Resolver struct{}

// NewResolver creates a new environment resolver
func NewResolver() *Resolver {
	return &Resolver{}
}

// Resolve merges base platform spec with environment overrides
func (r *Resolver) Resolve(platform *types.Platform, env *types.Environment) (*types.Platform, error) {
	if env == nil {
		return platform, nil
	}

	// Create a copy of the platform
	resolved := *platform

	// Apply environment overrides
	if err := r.applyOverrides(&resolved, env); err != nil {
		return nil, fmt.Errorf("failed to apply environment overrides: %w", err)
	}

	// Update metadata to reflect environment
	resolved.Metadata.Environment = env.Metadata.Name
	if resolved.Metadata.Labels == nil {
		resolved.Metadata.Labels = make(map[string]string)
	}
	resolved.Metadata.Labels["environment"] = string(env.Spec.Type)
	resolved.Metadata.Labels["environment-profile"] = env.Metadata.Name

	return &resolved, nil
}

// applyOverrides applies environment overrides to platform
func (r *Resolver) applyOverrides(platform *types.Platform, env *types.Environment) error {
	overrides := env.Spec.Overrides

	// Merge global config overrides
	if len(overrides.Global) > 0 {
		// Merge region if specified
		if region, ok := overrides.Global["region"].(string); ok {
			platform.Spec.Global.Region = region
		}

		// Merge other global config fields (stored as annotations for extensibility)
		if platform.Metadata.Annotations == nil {
			platform.Metadata.Annotations = make(map[string]string)
		}
		for k, v := range overrides.Global {
			if k != "region" && k != "tags" {
				if strVal, ok := v.(string); ok {
					platform.Metadata.Annotations["env.override."+k] = strVal
				}
			}
		}
	}

	// Store component overrides in annotations for plugins to use
	if platform.Metadata.Annotations == nil {
		platform.Metadata.Annotations = make(map[string]string)
	}

	// Store infrastructure overrides
	if len(overrides.Infrastructure) > 0 {
		for k, v := range overrides.Infrastructure {
			if strVal, ok := v.(string); ok {
				platform.Metadata.Annotations["env.infrastructure."+k] = strVal
			}
		}
	}

	// Store orchestrator overrides
	if len(overrides.Orchestrator) > 0 {
		for k, v := range overrides.Orchestrator {
			if strVal, ok := v.(string); ok {
				platform.Metadata.Annotations["env.orchestrator."+k] = strVal
			}
		}
	}

	// Store observability overrides
	if len(overrides.Observability) > 0 {
		for k, v := range overrides.Observability {
			if strVal, ok := v.(string); ok {
				platform.Metadata.Annotations["env.observability."+k] = strVal
			}
		}
	}

	// Store DevEx overrides
	if len(overrides.DevEx) > 0 {
		for k, v := range overrides.DevEx {
			if strVal, ok := v.(string); ok {
				platform.Metadata.Annotations["env.devex."+k] = strVal
			}
		}
	}

	// Merge tags
	if len(overrides.Tags) > 0 {
		if platform.Spec.Global.Tags == nil {
			platform.Spec.Global.Tags = make(map[string]string)
		}
		for k, v := range overrides.Tags {
			platform.Spec.Global.Tags[k] = v
		}
	}

	return nil
}

// ValidateEnvironmentChain validates environment promotion chain
func (r *Resolver) ValidateEnvironmentChain(environments []*types.Environment) error {
	// Build promotion graph
	promotionMap := make(map[string]string)
	for _, env := range environments {
		if env.Spec.Promotion != nil && env.Spec.Promotion.PromotesTo != "" {
			promotionMap[env.Metadata.Name] = env.Spec.Promotion.PromotesTo
		}
	}

	// Check for cycles
	visited := make(map[string]bool)
	for env := range promotionMap {
		if err := r.detectPromotionCycle(env, promotionMap, visited, make(map[string]bool)); err != nil {
			return err
		}
	}

	return nil
}

// detectPromotionCycle detects cycles in promotion chain
func (r *Resolver) detectPromotionCycle(env string, promotionMap map[string]string,
	visited, recStack map[string]bool) error {

	visited[env] = true
	recStack[env] = true

	if next, ok := promotionMap[env]; ok {
		if !visited[next] {
			if err := r.detectPromotionCycle(next, promotionMap, visited, recStack); err != nil {
				return err
			}
		} else if recStack[next] {
			return fmt.Errorf("circular promotion detected: %s -> %s", env, next)
		}
	}

	recStack[env] = false
	return nil
}
