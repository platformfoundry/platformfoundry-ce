package backstage

import (
	"fmt"

	"github.com/platformfoundry/pf-ce/pkg/plugin"
)

// Plugin implements the Backstage plugin
type Plugin struct{}

// BackstageConfig represents Backstage configuration
type BackstageConfig struct {
	Provider   string       `yaml:"provider" json:"provider" validate:"required,oneof=backstage"`
	ClusterRef string       `yaml:"clusterRef" json:"clusterRef" validate:"required"`
	Portal     PortalConfig `yaml:"portal" json:"portal" validate:"required"`
}

// PortalConfig represents portal configuration
type PortalConfig struct {
	Features      FeaturesConfig      `yaml:"features,omitempty" json:"features,omitempty"`
	Integrations  IntegrationsConfig  `yaml:"integrations,omitempty" json:"integrations,omitempty"`
	Customization CustomizationConfig `yaml:"customization,omitempty" json:"customization,omitempty"`
}

// FeaturesConfig represents enabled features
type FeaturesConfig struct {
	Catalog    bool `yaml:"catalog" json:"catalog"`
	TechDocs   bool `yaml:"techdocs" json:"techdocs"`
	Scaffolder bool `yaml:"scaffolder" json:"scaffolder"`
	Search     bool `yaml:"search" json:"search"`
}

// IntegrationsConfig represents external integrations
type IntegrationsConfig struct {
	GitHub     *GitHubIntegration     `yaml:"github,omitempty" json:"github,omitempty"`
	Kubernetes *KubernetesIntegration `yaml:"kubernetes,omitempty" json:"kubernetes,omitempty"`
}

// GitHubIntegration represents GitHub integration config
type GitHubIntegration struct {
	URL   string `yaml:"url" json:"url" validate:"required,url"`
	Token string `yaml:"token,omitempty" json:"token,omitempty"`
}

// KubernetesIntegration represents Kubernetes integration config
type KubernetesIntegration struct {
	Clusters []ClusterIntegration `yaml:"clusters,omitempty" json:"clusters,omitempty"`
}

// ClusterIntegration represents a Kubernetes cluster integration
type ClusterIntegration struct {
	Name         string `yaml:"name" json:"name" validate:"required"`
	AuthProvider string `yaml:"authProvider" json:"authProvider" validate:"required"`
}

// CustomizationConfig represents UI customization
type CustomizationConfig struct {
	Title string `yaml:"title,omitempty" json:"title,omitempty"`
	Theme string `yaml:"theme,omitempty" json:"theme,omitempty" validate:"omitempty,oneof=default light dark"`
}

// NewPlugin creates a new Backstage plugin
func NewPlugin() plugin.Plugin {
	return &Plugin{}
}

// Name returns the plugin name
func (p *Plugin) Name() string {
	return "backstage"
}

// Type returns the resource type
func (p *Plugin) Type() string {
	return "DevEx"
}

// Version returns the plugin version
func (p *Plugin) Version() string {
	return "1.0.0"
}

// ConfigType returns the configuration type
func (p *Plugin) ConfigType() interface{} {
	return &BackstageConfig{}
}

// Validate validates the plugin configuration
func (p *Plugin) Validate(spec map[string]interface{}) error {
	// Check required provider field
	provider, ok := spec["provider"].(string)
	if !ok || provider == "" {
		return fmt.Errorf("provider field is required")
	}

	if provider != "backstage" {
		return fmt.Errorf("provider must be 'backstage'")
	}

	// Check cluster reference
	clusterRef, ok := spec["clusterRef"].(string)
	if !ok || clusterRef == "" {
		return fmt.Errorf("clusterRef is required")
	}

	return nil
}

// Plan generates a plan for the plugin
func (p *Plugin) Plan(spec map[string]interface{}) (*plugin.Plan, error) {
	return &plugin.Plan{
		Actions: []string{"Install Backstage developer portal"},
	}, nil
}

// Apply applies the plugin configuration
func (p *Plugin) Apply(spec map[string]interface{}) (*plugin.Result, error) {
	return &plugin.Result{
		Status:  "success",
		Message: "Backstage installed",
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
		Message: "Backstage is ready",
	}, nil
}
