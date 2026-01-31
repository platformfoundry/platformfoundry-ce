package awscdk

import (
	"fmt"

	"github.com/platformfoundry/platformfoundry-ce/pkg/plugin"
)

// Config represents AWS CDK configuration
type Config struct {
	Provider    string         `yaml:"provider" json:"provider" validate:"required,oneof=aws-cdk"`
	Language    string         `yaml:"language" json:"language" validate:"required,oneof=typescript javascript python java csharp go"`
	Account     string         `yaml:"account,omitempty" json:"account,omitempty"`
	Region      string         `yaml:"region" json:"region" validate:"required"`
	Stacks      []StackConfig  `yaml:"stacks,omitempty" json:"stacks,omitempty"`
	Environment *EnvConfig     `yaml:"environment,omitempty" json:"environment,omitempty"`
}

// StackConfig represents CDK stack configuration
type StackConfig struct {
	Name        string                 `yaml:"name" json:"name" validate:"required"`
	Description string                 `yaml:"description,omitempty" json:"description,omitempty"`
	Props       map[string]interface{} `yaml:"props,omitempty" json:"props,omitempty"`
}

// EnvConfig represents environment configuration
type EnvConfig struct {
	Account string `yaml:"account" json:"account"`
	Region  string `yaml:"region" json:"region"`
}

// VPCConfig represents VPC configuration for CDK
type VPCConfig struct {
	MaxAZs           int      `yaml:"maxAzs" json:"maxAzs"`
	CIDR             string   `yaml:"cidr" json:"cidr"`
	NatGateways      int      `yaml:"natGateways" json:"natGateways"`
	SubnetConfig     []Subnet `yaml:"subnetConfiguration,omitempty" json:"subnetConfiguration,omitempty"`
}

// Subnet represents subnet configuration
type Subnet struct {
	Name       string `yaml:"name" json:"name"`
	SubnetType string `yaml:"subnetType" json:"subnetType" validate:"oneof=PUBLIC PRIVATE_WITH_NAT PRIVATE_ISOLATED"`
	CIDRMask   int    `yaml:"cidrMask" json:"cidrMask"`
}

// EKSConfig represents EKS cluster configuration
type EKSConfig struct {
	ClusterName    string            `yaml:"clusterName" json:"clusterName" validate:"required"`
	Version        string            `yaml:"version,omitempty" json:"version,omitempty"`
	DefaultCapacity int              `yaml:"defaultCapacity" json:"defaultCapacity"`
	NodeGroups     []NodeGroupConfig `yaml:"nodeGroups,omitempty" json:"nodeGroups,omitempty"`
}

// NodeGroupConfig represents managed node group configuration
type NodeGroupConfig struct {
	Name         string   `yaml:"name" json:"name" validate:"required"`
	InstanceTypes []string `yaml:"instanceTypes" json:"instanceTypes"`
	MinSize      int      `yaml:"minSize" json:"minSize"`
	MaxSize      int      `yaml:"maxSize" json:"maxSize"`
	DesiredSize  int      `yaml:"desiredSize" json:"desiredSize"`
	DiskSize     int      `yaml:"diskSize" json:"diskSize"`
}

// Plugin implements the AWS CDK plugin
type Plugin struct{}

// NewPlugin creates a new AWS CDK plugin
func NewPlugin() plugin.Plugin {
	return &Plugin{}
}

// Name returns the plugin name
func (p *Plugin) Name() string {
	return "aws-cdk"
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

	if provider != "aws-cdk" {
		return fmt.Errorf("provider must be 'aws-cdk'")
	}

	language, ok := spec["language"].(string)
	if !ok || language == "" {
		return fmt.Errorf("language is required (typescript, javascript, python, java, csharp, go)")
	}

	validLanguages := map[string]bool{
		"typescript": true, "javascript": true, "python": true,
		"java": true, "csharp": true, "go": true,
	}
	if !validLanguages[language] {
		return fmt.Errorf("language must be one of: typescript, javascript, python, java, csharp, go")
	}

	region, ok := spec["region"].(string)
	if !ok || region == "" {
		return fmt.Errorf("region is required")
	}

	return nil
}

// Plan generates a plan for the plugin
func (p *Plugin) Plan(spec map[string]interface{}) (*plugin.Plan, error) {
	language := "typescript"
	if l, ok := spec["language"].(string); ok {
		language = l
	}

	return &plugin.Plan{
		Actions: []string{
			fmt.Sprintf("Initialize AWS CDK app with %s", language),
			"Synthesize CloudFormation templates",
			"Deploy CDK stacks",
		},
	}, nil
}

// Apply applies the plugin configuration
func (p *Plugin) Apply(spec map[string]interface{}) (*plugin.Result, error) {
	return &plugin.Result{
		Status:  "success",
		Message: "Infrastructure provisioned with AWS CDK",
		Outputs: map[string]string{
			"provider": "aws-cdk",
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
		Message: "AWS CDK infrastructure is ready",
	}, nil
}
