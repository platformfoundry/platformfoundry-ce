package types

// Resource represents a Platform Foundry resource
type Resource struct {
	APIVersion string                 `yaml:"apiVersion" json:"apiVersion"`
	Kind       string                 `yaml:"kind" json:"kind"`
	Metadata   Metadata               `yaml:"metadata" json:"metadata"`
	Spec       map[string]interface{} `yaml:"spec" json:"spec"`
}

// Metadata contains resource metadata
type Metadata struct {
	Name         string            `yaml:"name" json:"name"`
	Labels       map[string]string `yaml:"labels,omitempty" json:"labels,omitempty"`
	Annotations  map[string]string `yaml:"annotations,omitempty" json:"annotations,omitempty"`
	Organization string            `yaml:"organization,omitempty" json:"organization,omitempty"`
	Environment  string            `yaml:"environment,omitempty" json:"environment,omitempty"`
	Sharing      *SharingConfig    `yaml:"sharing,omitempty" json:"sharing,omitempty"`
}

// SharingConfig defines how a resource is shared across organizations
type SharingConfig struct {
	Enabled       bool                   `yaml:"enabled" json:"enabled"`
	Organizations []OrganizationAccess   `yaml:"organizations,omitempty" json:"organizations,omitempty"`
}

// OrganizationAccess defines organization-specific access
type OrganizationAccess struct {
	Name         string   `yaml:"name" json:"name"`
	AccessLevel  string   `yaml:"accessLevel" json:"accessLevel"` // read, write, admin
	Environments []string `yaml:"environments,omitempty" json:"environments,omitempty"`
}

// ResourceList is a collection of resources
type ResourceList struct {
	Resources []Resource `json:"resources"`
}
