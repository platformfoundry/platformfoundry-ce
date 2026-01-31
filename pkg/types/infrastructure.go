package types

// Infrastructure represents infrastructure resources
// Implements US-1.2: Infrastructure Resource Definition
type Infrastructure struct {
	APIVersion string               `yaml:"apiVersion" json:"apiVersion"`
	Kind       string               `yaml:"kind" json:"kind"`
	Metadata   Metadata             `yaml:"metadata" json:"metadata"`
	Spec       InfrastructureSpec   `yaml:"spec" json:"spec"`
	Status     InfrastructureStatus `yaml:"status,omitempty" json:"status,omitempty"`
}

// InfrastructureSpec defines infrastructure specification
type InfrastructureSpec struct {
	Provider string                 `yaml:"provider" json:"provider"` // terraform, crossplane, pulumi
	Cloud    CloudConfig            `yaml:"cloud,omitempty" json:"cloud,omitempty"`
	Clusters []ClusterConfig        `yaml:"clusters,omitempty" json:"clusters,omitempty"`
	Tags     map[string]string      `yaml:"tags,omitempty" json:"tags,omitempty"`
}

// CloudConfig defines cloud provider configuration
type CloudConfig struct {
	Provider string      `yaml:"provider" json:"provider"` // aws, gcp, azure
	Region   string      `yaml:"region,omitempty" json:"region,omitempty"`
	VPC      *VPCConfig  `yaml:"vpc,omitempty" json:"vpc,omitempty"`
}

// VPCConfig defines VPC configuration
type VPCConfig struct {
	CIDR string `yaml:"cidr" json:"cidr"`
}

// ClusterConfig defines Kubernetes cluster configuration
type ClusterConfig struct {
	Name    string `yaml:"name" json:"name"`
	Type    string `yaml:"type" json:"type"` // eks, gke, aks
	Version string `yaml:"version,omitempty" json:"version,omitempty"`
}

// InfrastructureStatus represents infrastructure status
type InfrastructureStatus struct {
	Phase   Phase  `json:"phase"`
	Message string `json:"message,omitempty"`
}

// Validate validates the infrastructure resource
func (i *Infrastructure) Validate() error {
	if i.APIVersion == "" {
		return ErrMissingAPIVersion
	}
	if i.Kind != "Infrastructure" {
		return ErrInvalidKind
	}
	if i.Metadata.Name == "" {
		return ErrMissingName
	}
	if i.Spec.Provider == "" {
		return ErrInvalidProvider
	}
	return nil
}
