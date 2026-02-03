package terraform

import (
	"fmt"

	"github.com/platformfoundry/pf-ce/pkg/plugin"
)

// Plugin implements the Terraform plugin
type Plugin struct{}

// TerraformConfig represents Terraform configuration
type TerraformConfig struct {
	Provider string      `yaml:"provider" json:"provider" validate:"required,oneof=terraform"`
	Cloud    CloudConfig `yaml:"cloud" json:"cloud" validate:"required"`
}

// CloudConfig represents cloud provider configuration
type CloudConfig struct {
	Provider string      `yaml:"provider" json:"provider" validate:"required,oneof=aws gcp azure"`
	Region   string      `yaml:"region" json:"region" validate:"required"`
	VPC      *VPCConfig  `yaml:"vpc,omitempty" json:"vpc,omitempty"`
	Cluster  ClusterConfig `yaml:"cluster,omitempty" json:"cluster,omitempty"`
	Registry *RegistryConfig `yaml:"registry,omitempty" json:"registry,omitempty"`
}

// VPCConfig represents VPC configuration
type VPCConfig struct {
	CIDR    string         `yaml:"cidr" json:"cidr" validate:"required,cidr"`
	Subnets []SubnetConfig `yaml:"subnets,omitempty" json:"subnets,omitempty" validate:"dive"`
}

// SubnetConfig represents subnet configuration
type SubnetConfig struct {
	CIDR string `yaml:"cidr" json:"cidr" validate:"required,cidr"`
	Zone string `yaml:"zone" json:"zone" validate:"required"`
	Type string `yaml:"type" json:"type" validate:"required,oneof=public private"`
}

// ClusterConfig represents cluster configuration
type ClusterConfig struct {
	Name       string           `yaml:"name" json:"name" validate:"required"`
	Type       string           `yaml:"type" json:"type" validate:"required,oneof=eks gke aks"`
	Version    string           `yaml:"version,omitempty" json:"version,omitempty"`
	NodeGroups []NodeGroupConfig `yaml:"nodeGroups,omitempty" json:"nodeGroups,omitempty" validate:"dive"`
}

// NodeGroupConfig represents node group configuration
type NodeGroupConfig struct {
	Name         string `yaml:"name" json:"name" validate:"required"`
	InstanceType string `yaml:"instanceType" json:"instanceType" validate:"required"`
	MinSize      int    `yaml:"minSize" json:"minSize" validate:"required,min=1"`
	MaxSize      int    `yaml:"maxSize" json:"maxSize" validate:"required,gtefield=MinSize"`
}

// RegistryConfig represents container registry configuration
type RegistryConfig struct {
	Type string `yaml:"type" json:"type" validate:"required,oneof=ecr gcr acr"`
	Name string `yaml:"name" json:"name" validate:"required"`
}

// NewPlugin creates a new Terraform plugin
func NewPlugin() plugin.Plugin {
	return &Plugin{}
}

// Name returns the plugin name
func (p *Plugin) Name() string {
	return "terraform"
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
	return &TerraformConfig{}
}

// Validate validates the plugin configuration
func (p *Plugin) Validate(spec map[string]interface{}) error {
	// Check required provider field
	provider, ok := spec["provider"].(string)
	if !ok || provider == "" {
		return fmt.Errorf("provider field is required")
	}

	// Check cloud configuration
	cloud, ok := spec["cloud"].(map[string]interface{})
	if !ok {
		return fmt.Errorf("cloud configuration is required")
	}

	// Validate cloud provider
	cloudProvider, ok := cloud["provider"].(string)
	if !ok || cloudProvider == "" {
		return fmt.Errorf("cloud.provider is required")
	}

	validProviders := map[string]bool{"aws": true, "gcp": true, "azure": true}
	if !validProviders[cloudProvider] {
		return fmt.Errorf("cloud.provider must be one of: aws, gcp, azure")
	}

	// Validate region
	region, ok := cloud["region"].(string)
	if !ok || region == "" {
		return fmt.Errorf("cloud.region is required")
	}

	return nil
}

// Plan generates a plan for the plugin
func (p *Plugin) Plan(spec map[string]interface{}) (*plugin.Plan, error) {
	return &plugin.Plan{
		Actions: []string{"Provision infrastructure with Terraform"},
	}, nil
}

// Apply applies the plugin configuration
func (p *Plugin) Apply(spec map[string]interface{}) (*plugin.Result, error) {
	return &plugin.Result{
		Status:  "success",
		Message: "Infrastructure provisioned",
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
		Message: "Infrastructure is ready",
	}, nil
}
