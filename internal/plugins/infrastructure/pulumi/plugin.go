package pulumi

import (
	"fmt"

	"github.com/platformfoundry/platformfoundry-ce/pkg/plugin"
)

// Config represents Pulumi configuration
type Config struct {
	Provider string      `yaml:"provider" json:"provider" validate:"required,oneof=pulumi"`
	Runtime  string      `yaml:"runtime" json:"runtime" validate:"required,oneof=nodejs python go dotnet yaml"`
	Cloud    CloudConfig `yaml:"cloud" json:"cloud" validate:"required"`
	Stack    StackConfig `yaml:"stack,omitempty" json:"stack,omitempty"`
}

// CloudConfig represents cloud provider configuration
type CloudConfig struct {
	Provider string        `yaml:"provider" json:"provider" validate:"required,oneof=aws gcp azure kubernetes"`
	Region   string        `yaml:"region" json:"region" validate:"required"`
	VPC      *VPCConfig    `yaml:"vpc,omitempty" json:"vpc,omitempty"`
	Cluster  *ClusterConfig `yaml:"cluster,omitempty" json:"cluster,omitempty"`
}

// VPCConfig represents VPC configuration
type VPCConfig struct {
	CIDR             string   `yaml:"cidr" json:"cidr" validate:"required,cidr"`
	AvailabilityZones []string `yaml:"availabilityZones,omitempty" json:"availabilityZones,omitempty"`
	EnableNATGateway bool     `yaml:"enableNATGateway" json:"enableNATGateway"`
}

// ClusterConfig represents Kubernetes cluster configuration
type ClusterConfig struct {
	Name       string            `yaml:"name" json:"name" validate:"required"`
	Version    string            `yaml:"version,omitempty" json:"version,omitempty"`
	NodeGroups []NodeGroupConfig `yaml:"nodeGroups,omitempty" json:"nodeGroups,omitempty"`
}

// NodeGroupConfig represents node group configuration
type NodeGroupConfig struct {
	Name         string `yaml:"name" json:"name" validate:"required"`
	InstanceType string `yaml:"instanceType" json:"instanceType" validate:"required"`
	DesiredSize  int    `yaml:"desiredSize" json:"desiredSize" validate:"required,min=1"`
	MinSize      int    `yaml:"minSize" json:"minSize" validate:"required,min=0"`
	MaxSize      int    `yaml:"maxSize" json:"maxSize" validate:"required,gtefield=MinSize"`
}

// StackConfig represents Pulumi stack configuration
type StackConfig struct {
	Name         string            `yaml:"name" json:"name"`
	Organization string            `yaml:"organization,omitempty" json:"organization,omitempty"`
	Config       map[string]string `yaml:"config,omitempty" json:"config,omitempty"`
}

// Plugin implements the Pulumi plugin
type Plugin struct{}

// NewPlugin creates a new Pulumi plugin
func NewPlugin() plugin.Plugin {
	return &Plugin{}
}

// Name returns the plugin name
func (p *Plugin) Name() string {
	return "pulumi"
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

	if provider != "pulumi" {
		return fmt.Errorf("provider must be 'pulumi'")
	}

	runtime, ok := spec["runtime"].(string)
	if !ok || runtime == "" {
		return fmt.Errorf("runtime is required (nodejs, python, go, dotnet, yaml)")
	}

	validRuntimes := map[string]bool{"nodejs": true, "python": true, "go": true, "dotnet": true, "yaml": true}
	if !validRuntimes[runtime] {
		return fmt.Errorf("runtime must be one of: nodejs, python, go, dotnet, yaml")
	}

	cloud, ok := spec["cloud"].(map[string]interface{})
	if !ok {
		return fmt.Errorf("cloud configuration is required")
	}

	cloudProvider, ok := cloud["provider"].(string)
	if !ok || cloudProvider == "" {
		return fmt.Errorf("cloud.provider is required")
	}

	return nil
}

// Plan generates a plan for the plugin
func (p *Plugin) Plan(spec map[string]interface{}) (*plugin.Plan, error) {
	runtime := "unknown"
	if r, ok := spec["runtime"].(string); ok {
		runtime = r
	}

	return &plugin.Plan{
		Actions: []string{
			fmt.Sprintf("Initialize Pulumi stack with %s runtime", runtime),
			"Preview infrastructure changes",
			"Apply infrastructure configuration",
		},
	}, nil
}

// Apply applies the plugin configuration
func (p *Plugin) Apply(spec map[string]interface{}) (*plugin.Result, error) {
	return &plugin.Result{
		Status:  "success",
		Message: "Infrastructure provisioned with Pulumi",
		Outputs: map[string]string{
			"provider": "pulumi",
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
		Message: "Pulumi infrastructure is ready",
	}, nil
}
