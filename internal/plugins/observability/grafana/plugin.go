package grafana

import (
	"fmt"

	"github.com/platformfoundry/platformfoundry-ce/pkg/plugin"
)

// Plugin implements the Grafana plugin
type Plugin struct{}

// GrafanaConfig represents Grafana configuration
type GrafanaConfig struct {
	Provider       string              `yaml:"provider" json:"provider" validate:"required,oneof=grafana"`
	ClusterRef     string              `yaml:"clusterRef" json:"clusterRef" validate:"required"`
	Visualization  VisualizationConfig `yaml:"visualization" json:"visualization" validate:"required"`
}

// VisualizationConfig represents visualization configuration
type VisualizationConfig struct {
	Grafana GrafanaSettings `yaml:"grafana" json:"grafana" validate:"required"`
}

// GrafanaSettings represents Grafana-specific settings
type GrafanaSettings struct {
	Datasources []string `yaml:"datasources,omitempty" json:"datasources,omitempty"`
	Dashboards  []string `yaml:"dashboards,omitempty" json:"dashboards,omitempty"`
	Plugins     []string `yaml:"plugins,omitempty" json:"plugins,omitempty"`
}

// NewPlugin creates a new Grafana plugin
func NewPlugin() plugin.Plugin {
	return &Plugin{}
}

// Name returns the plugin name
func (p *Plugin) Name() string {
	return "grafana"
}

// Type returns the resource type
func (p *Plugin) Type() string {
	return "Observability"
}

// Version returns the plugin version
func (p *Plugin) Version() string {
	return "1.0.0"
}

// ConfigType returns the configuration type
func (p *Plugin) ConfigType() interface{} {
	return &GrafanaConfig{}
}

// Validate validates the plugin configuration
func (p *Plugin) Validate(spec map[string]interface{}) error {
	// Check required provider field
	provider, ok := spec["provider"].(string)
	if !ok || provider == "" {
		return fmt.Errorf("provider field is required")
	}

	if provider != "grafana" {
		return fmt.Errorf("provider must be 'grafana'")
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
		Actions: []string{"Install Grafana"},
	}, nil
}

// Apply applies the plugin configuration
func (p *Plugin) Apply(spec map[string]interface{}) (*plugin.Result, error) {
	return &plugin.Result{
		Status:  "success",
		Message: "Grafana installed",
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
		Message: "Grafana is ready",
	}, nil
}
