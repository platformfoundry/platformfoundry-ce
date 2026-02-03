package types

import (
	"time"
)

// CloudProvider represents a cloud provider type
type CloudProvider string

const (
	CloudProviderAWS          CloudProvider = "aws"
	CloudProviderAzure        CloudProvider = "azure"
	CloudProviderGCP          CloudProvider = "gcp"
	CloudProviderDigitalOcean CloudProvider = "digitalocean"
	CloudProviderOnPrem       CloudProvider = "on-prem"
)

// CloudAccount represents a cloud provider account
type CloudAccount struct {
	APIVersion string              `yaml:"apiVersion" json:"apiVersion"`
	Kind       string              `yaml:"kind" json:"kind"`
	Metadata   CloudMetadata       `yaml:"metadata" json:"metadata"`
	Spec       CloudAccountSpec    `yaml:"spec" json:"spec"`
	Status     *CloudAccountStatus `yaml:"status,omitempty" json:"status,omitempty"`
}

// CloudMetadata contains metadata for cloud resources
type CloudMetadata struct {
	Name        string            `yaml:"name" json:"name"`
	Labels      map[string]string `yaml:"labels,omitempty" json:"labels,omitempty"`
	Annotations map[string]string `yaml:"annotations,omitempty" json:"annotations,omitempty"`
}

// CloudAccountSpec defines cloud account configuration
type CloudAccountSpec struct {
	Provider      CloudProvider     `yaml:"provider" json:"provider"`
	Credentials   CloudCredentials  `yaml:"credentials" json:"credentials"`
	Regions       []string          `yaml:"regions,omitempty" json:"regions,omitempty"`
	DefaultRegion string            `yaml:"defaultRegion,omitempty" json:"defaultRegion,omitempty"`
	Tags          map[string]string `yaml:"tags,omitempty" json:"tags,omitempty"`
}

// CloudCredentials defines how to authenticate with a cloud provider
type CloudCredentials struct {
	Type      string `yaml:"type" json:"type"` // secret, iam-role, service-account, env
	SecretRef string `yaml:"secretRef,omitempty" json:"secretRef,omitempty"`
	RoleARN   string `yaml:"roleArn,omitempty" json:"roleArn,omitempty"`
	ProjectID string `yaml:"projectId,omitempty" json:"projectId,omitempty"`
}

// CloudAccountStatus represents account status
type CloudAccountStatus struct {
	Connected    bool           `yaml:"connected" json:"connected"`
	LastVerified *time.Time     `yaml:"lastVerified,omitempty" json:"lastVerified,omitempty"`
	Regions      []RegionStatus `yaml:"regions,omitempty" json:"regions,omitempty"`
	Message      string         `yaml:"message,omitempty" json:"message,omitempty"`
}

// RegionStatus represents the status of a region
type RegionStatus struct {
	Name      string `yaml:"name" json:"name"`
	Available bool   `yaml:"available" json:"available"`
}

// UnifiedResource represents a cloud-agnostic resource
type UnifiedResource struct {
	APIVersion string                 `yaml:"apiVersion" json:"apiVersion"`
	Kind       string                 `yaml:"kind" json:"kind"`
	Metadata   CloudMetadata          `yaml:"metadata" json:"metadata"`
	Spec       UnifiedResourceSpec    `yaml:"spec" json:"spec"`
	Status     *UnifiedResourceStatus `yaml:"status,omitempty" json:"status,omitempty"`
}

// UnifiedResourceSpec defines a cloud-agnostic resource spec
type UnifiedResourceSpec struct {
	Type              UnifiedResourceType                      `yaml:"type" json:"type"`
	AccountRef        string                                   `yaml:"accountRef" json:"accountRef"`
	Region            string                                   `yaml:"region,omitempty" json:"region,omitempty"`
	Size              string                                   `yaml:"size,omitempty" json:"size,omitempty"`
	Config            map[string]interface{}                   `yaml:"config,omitempty" json:"config,omitempty"`
	ProviderOverrides map[CloudProvider]map[string]interface{} `yaml:"providerOverrides,omitempty" json:"providerOverrides,omitempty"`
}

// UnifiedResourceType represents abstract resource types
type UnifiedResourceType string

const (
	// Compute resources
	UnifiedResourceVM         UnifiedResourceType = "virtual-machine"
	UnifiedResourceContainer  UnifiedResourceType = "container"
	UnifiedResourceKubernetes UnifiedResourceType = "kubernetes-cluster"
	UnifiedResourceServerless UnifiedResourceType = "serverless-function"

	// Database resources
	UnifiedResourceRDBMS UnifiedResourceType = "relational-database"
	UnifiedResourceNoSQL UnifiedResourceType = "nosql-database"
	UnifiedResourceCache UnifiedResourceType = "cache"

	// Storage resources
	UnifiedResourceObjectStorage UnifiedResourceType = "object-storage"
	UnifiedResourceBlockStorage  UnifiedResourceType = "block-storage"
	UnifiedResourceFileStorage   UnifiedResourceType = "file-storage"

	// Network resources
	UnifiedResourceVPC          UnifiedResourceType = "vpc"
	UnifiedResourceLoadBalancer UnifiedResourceType = "load-balancer"
	UnifiedResourceDNS          UnifiedResourceType = "dns-zone"
	UnifiedResourceCDN          UnifiedResourceType = "cdn"

	// Security resources
	UnifiedResourceSecurityGroup UnifiedResourceType = "security-group"
	UnifiedResourceSSLCert       UnifiedResourceType = "ssl-certificate"
	UnifiedResourceKMS           UnifiedResourceType = "key-management"
)

// UnifiedResourceStatus represents resource status
type UnifiedResourceStatus struct {
	State            string                 `yaml:"state" json:"state"` // provisioning, running, stopped, failed
	ProviderID       string                 `yaml:"providerId,omitempty" json:"providerId,omitempty"`
	ProviderType     string                 `yaml:"providerType,omitempty" json:"providerType,omitempty"`
	Endpoints        []Endpoint             `yaml:"endpoints,omitempty" json:"endpoints,omitempty"`
	CreatedAt        *time.Time             `yaml:"createdAt,omitempty" json:"createdAt,omitempty"`
	UpdatedAt        *time.Time             `yaml:"updatedAt,omitempty" json:"updatedAt,omitempty"`
	ProviderMetadata map[string]interface{} `yaml:"providerMetadata,omitempty" json:"providerMetadata,omitempty"`
}

// Endpoint represents a resource endpoint
type Endpoint struct {
	Type     string `yaml:"type" json:"type"` // public, private
	Address  string `yaml:"address" json:"address"`
	Port     int    `yaml:"port,omitempty" json:"port,omitempty"`
	Protocol string `yaml:"protocol,omitempty" json:"protocol,omitempty"`
}

// ResourceMapping maps unified resource types to provider-specific types
type ResourceMapping struct {
	UnifiedType   UnifiedResourceType      `yaml:"unifiedType" json:"unifiedType"`
	ProviderTypes map[CloudProvider]string `yaml:"providerTypes" json:"providerTypes"`
	SizeMapping   map[string]ProviderSizes `yaml:"sizeMapping,omitempty" json:"sizeMapping,omitempty"`
}

// ProviderSizes maps size names to provider-specific instance types
type ProviderSizes struct {
	AWS   string `yaml:"aws,omitempty" json:"aws,omitempty"`
	Azure string `yaml:"azure,omitempty" json:"azure,omitempty"`
	GCP   string `yaml:"gcp,omitempty" json:"gcp,omitempty"`
}

// MultiCloudDeployment represents a deployment across multiple clouds
type MultiCloudDeployment struct {
	APIVersion string                      `yaml:"apiVersion" json:"apiVersion"`
	Kind       string                      `yaml:"kind" json:"kind"`
	Metadata   CloudMetadata               `yaml:"metadata" json:"metadata"`
	Spec       MultiCloudDeploymentSpec    `yaml:"spec" json:"spec"`
	Status     *MultiCloudDeploymentStatus `yaml:"status,omitempty" json:"status,omitempty"`
}

// MultiCloudDeploymentSpec defines multi-cloud deployment configuration
type MultiCloudDeploymentSpec struct {
	Strategy     DeploymentStrategy    `yaml:"strategy" json:"strategy"`
	Targets      []DeploymentTarget    `yaml:"targets" json:"targets"`
	Resources    []UnifiedResourceSpec `yaml:"resources" json:"resources"`
	TrafficSplit map[string]int        `yaml:"trafficSplit,omitempty" json:"trafficSplit,omitempty"`
}

// DeploymentStrategy defines how to deploy across clouds
type DeploymentStrategy string

const (
	StrategyPrimary      DeploymentStrategy = "primary-secondary"
	StrategyActiveActive DeploymentStrategy = "active-active"
	StrategyHybrid       DeploymentStrategy = "hybrid"
	StrategyBurst        DeploymentStrategy = "cloud-burst"
)

// DeploymentTarget defines a target cloud for deployment
type DeploymentTarget struct {
	AccountRef string `yaml:"accountRef" json:"accountRef"`
	Region     string `yaml:"region" json:"region"`
	Weight     int    `yaml:"weight,omitempty" json:"weight,omitempty"`
	Primary    bool   `yaml:"primary,omitempty" json:"primary,omitempty"`
}

// MultiCloudDeploymentStatus represents deployment status
type MultiCloudDeploymentStatus struct {
	Phase   string         `yaml:"phase" json:"phase"`
	Targets []TargetStatus `yaml:"targets,omitempty" json:"targets,omitempty"`
	Message string         `yaml:"message,omitempty" json:"message,omitempty"`
}

// TargetStatus represents the status of a deployment target
type TargetStatus struct {
	AccountRef string `yaml:"accountRef" json:"accountRef"`
	Region     string `yaml:"region" json:"region"`
	State      string `yaml:"state" json:"state"`
	Resources  int    `yaml:"resources" json:"resources"`
	Healthy    int    `yaml:"healthy" json:"healthy"`
}

// CloudCost represents cost data from a cloud provider
type CloudCost struct {
	Provider   CloudProvider      `json:"provider"`
	Account    string             `json:"account"`
	Period     string             `json:"period"`
	Total      float64            `json:"total"`
	Currency   string             `json:"currency"`
	ByService  map[string]float64 `json:"byService,omitempty"`
	ByRegion   map[string]float64 `json:"byRegion,omitempty"`
	ByResource map[string]float64 `json:"byResource,omitempty"`
}
