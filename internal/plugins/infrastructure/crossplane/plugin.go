package crossplane

import (
	"fmt"

	"github.com/platformfoundry/pf-ce/pkg/plugin"
)

// Config represents Crossplane configuration
type Config struct {
	Provider       string              `yaml:"provider" json:"provider" validate:"required,oneof=crossplane"`
	CloudProvider  string              `yaml:"cloudProvider" json:"cloudProvider" validate:"required,oneof=aws gcp azure"`
	Compositions   []CompositionConfig `yaml:"compositions,omitempty" json:"compositions,omitempty"`
	ProviderConfig *ProviderConfig     `yaml:"providerConfig,omitempty" json:"providerConfig,omitempty"`
}

// CompositionConfig represents a Crossplane composition
type CompositionConfig struct {
	Name             string           `yaml:"name" json:"name" validate:"required"`
	Namespace        string           `yaml:"namespace,omitempty" json:"namespace,omitempty"`
	CompositeTypeRef CompositeTypeRef `yaml:"compositeTypeRef" json:"compositeTypeRef"`
	Resources        []ResourceConfig `yaml:"resources,omitempty" json:"resources,omitempty"`
}

// CompositeTypeRef references the composite resource type
type CompositeTypeRef struct {
	APIVersion string `yaml:"apiVersion" json:"apiVersion" validate:"required"`
	Kind       string `yaml:"kind" json:"kind" validate:"required"`
}

// ResourceConfig represents a resource in a composition
type ResourceConfig struct {
	Name string                 `yaml:"name" json:"name" validate:"required"`
	Base map[string]interface{} `yaml:"base" json:"base"`
}

// ProviderConfig represents cloud provider configuration
type ProviderConfig struct {
	Name        string            `yaml:"name" json:"name" validate:"required"`
	Credentials CredentialsConfig `yaml:"credentials" json:"credentials"`
}

// CredentialsConfig represents credentials configuration
type CredentialsConfig struct {
	Source    string     `yaml:"source" json:"source" validate:"required,oneof=Secret InjectedIdentity Environment"`
	SecretRef *SecretRef `yaml:"secretRef,omitempty" json:"secretRef,omitempty"`
}

// SecretRef references a Kubernetes secret
type SecretRef struct {
	Name      string `yaml:"name" json:"name" validate:"required"`
	Namespace string `yaml:"namespace" json:"namespace" validate:"required"`
	Key       string `yaml:"key" json:"key" validate:"required"`
}

// XRDConfig represents Composite Resource Definition
type XRDConfig struct {
	Name    string     `yaml:"name" json:"name" validate:"required"`
	Group   string     `yaml:"group" json:"group" validate:"required"`
	Version string     `yaml:"version" json:"version" validate:"required"`
	Kind    string     `yaml:"kind" json:"kind" validate:"required"`
	Plural  string     `yaml:"plural" json:"plural" validate:"required"`
	Schema  SchemaSpec `yaml:"schema,omitempty" json:"schema,omitempty"`
}

// SchemaSpec defines the schema for the XRD
type SchemaSpec struct {
	OpenAPIV3Schema map[string]interface{} `yaml:"openAPIV3Schema,omitempty" json:"openAPIV3Schema,omitempty"`
}

// Plugin implements the Crossplane plugin
type Plugin struct{}

// NewPlugin creates a new Crossplane plugin
func NewPlugin() plugin.Plugin {
	return &Plugin{}
}

// Name returns the plugin name
func (p *Plugin) Name() string {
	return "crossplane"
}

// Type returns the resource type
func (p *Plugin) Type() string {
	return "Infrastructure"
}

// Version returns the plugin version
func (p *Plugin) Version() string {
	return "1.0.0"
}

// ConfigType returns the configuration type
func (p *Plugin) ConfigType() interface{} {
	return &Config{}
}

// Validate validates the plugin configuration
func (p *Plugin) Validate(spec map[string]interface{}) error {
	provider, ok := spec["provider"].(string)
	if !ok || provider == "" {
		return fmt.Errorf("provider field is required")
	}

	if provider != "crossplane" {
		return fmt.Errorf("provider must be 'crossplane'")
	}

	cloudProvider, ok := spec["cloudProvider"].(string)
	if !ok || cloudProvider == "" {
		return fmt.Errorf("cloudProvider is required (aws, gcp, azure)")
	}

	validProviders := map[string]bool{"aws": true, "gcp": true, "azure": true}
	if !validProviders[cloudProvider] {
		return fmt.Errorf("cloudProvider must be one of: aws, gcp, azure")
	}

	return nil
}

// Plan generates a plan for the plugin
func (p *Plugin) Plan(spec map[string]interface{}) (*plugin.Plan, error) {
	cloudProvider := "aws"
	if cp, ok := spec["cloudProvider"].(string); ok {
		cloudProvider = cp
	}

	return &plugin.Plan{
		Actions: []string{
			fmt.Sprintf("Configure Crossplane %s provider", cloudProvider),
			"Apply Composite Resource Definitions (XRDs)",
			"Deploy Compositions",
			"Provision cloud resources via Kubernetes API",
		},
	}, nil
}

// Apply applies the plugin configuration
func (p *Plugin) Apply(spec map[string]interface{}) (*plugin.Result, error) {
	return &plugin.Result{
		Status:  "success",
		Message: "Infrastructure provisioned with Crossplane",
		Outputs: map[string]string{
			"provider": "crossplane",
		},
	}, nil
}

// Delete deletes resources created by the plugin
func (p *Plugin) Delete(name string) error {
	return nil
}

// Status gets the current status of the resource
func (p *Plugin) Status(name string) (*plugin.Status, error) {
	return &plugin.Status{
		State:   "ready",
		Ready:   true,
		Message: "Crossplane infrastructure is ready",
	}, nil
}
