package types

import (
	"fmt"
	"regexp"
)

// Environment represents an environment profile (dev, staging, prod)
type Environment struct {
	APIVersion string            `yaml:"apiVersion" json:"apiVersion"`
	Kind       string            `yaml:"kind" json:"kind"`
	Metadata   Metadata          `yaml:"metadata" json:"metadata"`
	Spec       EnvironmentSpec   `yaml:"spec" json:"spec"`
	Status     EnvironmentStatus `yaml:"status,omitempty" json:"status,omitempty"`
}

// EnvironmentSpec defines environment specification
type EnvironmentSpec struct {
	Type        EnvironmentType      `yaml:"type" json:"type"`
	PlatformRef string               `yaml:"platformRef" json:"platformRef"`
	Overrides   EnvironmentOverrides `yaml:"overrides,omitempty" json:"overrides,omitempty"`
	Promotion   *PromotionConfig     `yaml:"promotion,omitempty" json:"promotion,omitempty"`
}

// EnvironmentType represents the type of environment
type EnvironmentType string

const (
	EnvironmentDev        EnvironmentType = "development"
	EnvironmentStaging    EnvironmentType = "staging"
	EnvironmentProduction EnvironmentType = "production"
)

// EnvironmentOverrides defines environment-specific overrides
type EnvironmentOverrides struct {
	Infrastructure map[string]interface{} `yaml:"infrastructure,omitempty" json:"infrastructure,omitempty"`
	Orchestrator   map[string]interface{} `yaml:"orchestrator,omitempty" json:"orchestrator,omitempty"`
	Observability  map[string]interface{} `yaml:"observability,omitempty" json:"observability,omitempty"`
	DevEx          map[string]interface{} `yaml:"devex,omitempty" json:"devex,omitempty"`
	Pipeline       map[string]interface{} `yaml:"pipeline,omitempty" json:"pipeline,omitempty"`
	Global         map[string]interface{} `yaml:"global,omitempty" json:"global,omitempty"`
	Tags           map[string]string      `yaml:"tags,omitempty" json:"tags,omitempty"`
}

// PromotionConfig defines promotion settings between environments
type PromotionConfig struct {
	Auto             bool     `yaml:"auto,omitempty" json:"auto,omitempty"`
	PromotesTo       string   `yaml:"promotesTo,omitempty" json:"promotesTo,omitempty"`
	RequiresApproval bool     `yaml:"requiresApproval,omitempty" json:"requiresApproval,omitempty"`
	Approvers        []string `yaml:"approvers,omitempty" json:"approvers,omitempty"`
}

// EnvironmentStatus represents environment status
type EnvironmentStatus struct {
	Phase   Phase  `json:"phase"`
	Message string `json:"message,omitempty"`
}

var (
	// environmentNameRegex validates environment names
	environmentNameRegex = regexp.MustCompile(`^[a-z0-9]([-a-z0-9]*[a-z0-9])?$`)
)

// Validate validates the environment resource with security checks
func (e *Environment) Validate() error {
	if e.APIVersion == "" {
		return ErrMissingAPIVersion
	}
	if e.Kind != "Environment" {
		return ErrInvalidKind
	}
	if e.Metadata.Name == "" {
		return ErrMissingName
	}

	// Security: Validate environment name format
	if len(e.Metadata.Name) > 63 {
		return fmt.Errorf("environment name must be 63 characters or less")
	}
	if !environmentNameRegex.MatchString(e.Metadata.Name) {
		return fmt.Errorf("environment name must be lowercase alphanumeric with hyphens")
	}

	if e.Spec.Type == "" {
		return ErrMissingEnvironmentType
	}

	// Security: Validate environment type is one of the allowed values
	if !IsValidEnvironmentType(e.Spec.Type) {
		return ErrInvalidEnvironmentType
	}

	if e.Spec.PlatformRef == "" {
		return ErrMissingPlatformRef
	}

	// Security: Validate platformRef format
	if len(e.Spec.PlatformRef) > 253 {
		return fmt.Errorf("platform reference must be 253 characters or less")
	}

	// Validate promotion config if present
	if e.Spec.Promotion != nil {
		if e.Spec.Promotion.PromotesTo != "" && len(e.Spec.Promotion.PromotesTo) > 253 {
			return fmt.Errorf("promotion target must be 253 characters or less")
		}

		// Security: Limit number of approvers
		if len(e.Spec.Promotion.Approvers) > 50 {
			return fmt.Errorf("too many approvers (max 50)")
		}

		// Validate approver names
		for _, approver := range e.Spec.Promotion.Approvers {
			if len(approver) > 128 {
				return fmt.Errorf("approver name must be 128 characters or less")
			}
		}
	}

	return nil
}

// IsValidEnvironmentType checks if an environment type is valid
func IsValidEnvironmentType(envType EnvironmentType) bool {
	return envType == EnvironmentDev ||
		envType == EnvironmentStaging ||
		envType == EnvironmentProduction
}
