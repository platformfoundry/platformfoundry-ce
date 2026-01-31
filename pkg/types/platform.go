package types

import "time"

// Platform represents the top-level platform resource
// Implements US-1.1: Platform Resource Definition
type Platform struct {
	APIVersion string         `yaml:"apiVersion" json:"apiVersion"`
	Kind       string         `yaml:"kind" json:"kind"`
	Metadata   Metadata       `yaml:"metadata" json:"metadata"`
	Spec       PlatformSpec   `yaml:"spec" json:"spec"`
	Status     PlatformStatus `yaml:"status,omitempty" json:"status,omitempty"`
}

// PlatformSpec defines the platform specification
type PlatformSpec struct {
	// Component references
	Components ComponentReferences `yaml:"components" json:"components"`

	// Platform-level configuration
	Global GlobalConfig `yaml:"global,omitempty" json:"global,omitempty"`
}

// ComponentReferences defines references to platform components
type ComponentReferences struct {
	Infrastructure string `yaml:"infrastructure,omitempty" json:"infrastructure,omitempty"`
	Orchestrator   string `yaml:"orchestrator,omitempty" json:"orchestrator,omitempty"`
	Pipeline       string `yaml:"pipeline,omitempty" json:"pipeline,omitempty"`
	Observability  string `yaml:"observability,omitempty" json:"observability,omitempty"`
	Mesh           string `yaml:"mesh,omitempty" json:"mesh,omitempty"`
	DevEx          string `yaml:"devex,omitempty" json:"devex,omitempty"`
	Security       string `yaml:"security,omitempty" json:"security,omitempty"`
	Compliance     string `yaml:"compliance,omitempty" json:"compliance,omitempty"`
}

// GlobalConfig defines global settings
type GlobalConfig struct {
	Region string            `yaml:"region,omitempty" json:"region,omitempty"`
	Tags   map[string]string `yaml:"tags,omitempty" json:"tags,omitempty"`
}

// PlatformStatus represents the platform status
type PlatformStatus struct {
	Phase      Phase              `json:"phase"`
	Conditions []PlatformCondition `json:"conditions,omitempty"`
	Message    string             `json:"message,omitempty"`
	LastApplied *time.Time        `json:"lastApplied,omitempty"`
}

// Phase represents the platform lifecycle phase
type Phase string

const (
	PhasePending      Phase = "Pending"
	PhaseProvisioning Phase = "Provisioning"
	PhaseReady        Phase = "Ready"
	PhaseFailed       Phase = "Failed"
)

// PlatformCondition represents a platform condition
type PlatformCondition struct {
	Type    string    `json:"type"`
	Status  string    `json:"status"`
	Message string    `json:"message,omitempty"`
	LastTransitionTime time.Time `json:"lastTransitionTime,omitempty"`
}

// Validate validates the platform resource
func (p *Platform) Validate() error {
	if p.APIVersion == "" {
		return ErrMissingAPIVersion
	}
	if p.Kind != "Platform" {
		return ErrInvalidKind
	}
	if p.Metadata.Name == "" {
		return ErrMissingName
	}
	if !p.hasComponents() {
		return ErrNoComponents
	}
	return nil
}

// hasComponents checks if at least one component is defined
func (p *Platform) hasComponents() bool {
	c := p.Spec.Components
	return c.Infrastructure != "" ||
		c.Orchestrator != "" ||
		c.Pipeline != "" ||
		c.Observability != "" ||
		c.Mesh != "" ||
		c.DevEx != "" ||
		c.Security != "" ||
		c.Compliance != ""
}

// GetComponentNames returns list of defined component names
func (p *Platform) GetComponentNames() []string {
	var names []string
	c := p.Spec.Components

	if c.Infrastructure != "" {
		names = append(names, c.Infrastructure)
	}
	if c.Orchestrator != "" {
		names = append(names, c.Orchestrator)
	}
	if c.Pipeline != "" {
		names = append(names, c.Pipeline)
	}
	if c.Observability != "" {
		names = append(names, c.Observability)
	}
	if c.Mesh != "" {
		names = append(names, c.Mesh)
	}
	if c.DevEx != "" {
		names = append(names, c.DevEx)
	}
	if c.Security != "" {
		names = append(names, c.Security)
	}
	if c.Compliance != "" {
		names = append(names, c.Compliance)
	}

	return names
}
