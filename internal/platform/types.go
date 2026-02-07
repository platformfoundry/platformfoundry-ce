package platform

import (
	"time"
)

// Platform represents a complete platform definition
type Platform struct {
	APIVersion string           `json:"apiVersion" yaml:"apiVersion"`
	Kind       string           `json:"kind" yaml:"kind"`
	Metadata   PlatformMetadata `json:"metadata" yaml:"metadata"`
	Spec       PlatformSpec     `json:"spec" yaml:"spec"`
	Status     *PlatformStatus  `json:"status,omitempty" yaml:"status,omitempty"`
}

// PlatformMetadata contains platform identification
type PlatformMetadata struct {
	Name        string            `json:"name" yaml:"name"`
	Version     string            `json:"version,omitempty" yaml:"version,omitempty"`
	Description string            `json:"description,omitempty" yaml:"description,omitempty"`
	Labels      map[string]string `json:"labels,omitempty" yaml:"labels,omitempty"`
	Annotations map[string]string `json:"annotations,omitempty" yaml:"annotations,omitempty"`
	Owner       string            `json:"owner,omitempty" yaml:"owner,omitempty"`
	CreatedAt   time.Time         `json:"createdAt,omitempty" yaml:"createdAt,omitempty"`
	UpdatedAt   time.Time         `json:"updatedAt,omitempty" yaml:"updatedAt,omitempty"`
}

// PlatformSpec defines platform configuration
type PlatformSpec struct {
	GoldenPaths  []GoldenPath            `json:"goldenPaths,omitempty" yaml:"goldenPaths,omitempty"`
	Capabilities PlatformCapabilities    `json:"capabilities" yaml:"capabilities"`
	Policies     PlatformPolicies        `json:"policies,omitempty" yaml:"policies,omitempty"`
	Environments []EnvironmentDefinition `json:"environments,omitempty" yaml:"environments,omitempty"`
	Defaults     *PlatformDefaults       `json:"defaults,omitempty" yaml:"defaults,omitempty"`
}

// GoldenPath defines a standardized way to build applications
type GoldenPath struct {
	Name          string                 `json:"name" yaml:"name"`
	Description   string                 `json:"description,omitempty" yaml:"description,omitempty"`
	Template      string                 `json:"template" yaml:"template"`
	Language      string                 `json:"language,omitempty" yaml:"language,omitempty"`
	Framework     string                 `json:"framework,omitempty" yaml:"framework,omitempty"`
	Resources     []ResourceType         `json:"resources,omitempty" yaml:"resources,omitempty"`
	Pipelines     []string               `json:"pipelines,omitempty" yaml:"pipelines,omitempty"`
	Observability []string               `json:"observability,omitempty" yaml:"observability,omitempty"`
	Security      *SecurityConfig        `json:"security,omitempty" yaml:"security,omitempty"`
	Defaults      map[string]interface{} `json:"defaults,omitempty" yaml:"defaults,omitempty"`
	Tags          []string               `json:"tags,omitempty" yaml:"tags,omitempty"`
}

// ResourceType defines a resource that can be provisioned
type ResourceType struct {
	Type      string                 `json:"type" yaml:"type"`
	Name      string                 `json:"name,omitempty" yaml:"name,omitempty"`
	Provider  string                 `json:"provider,omitempty" yaml:"provider,omitempty"`
	Config    map[string]interface{} `json:"config,omitempty" yaml:"config,omitempty"`
	Required  bool                   `json:"required,omitempty" yaml:"required,omitempty"`
	Shareable bool                   `json:"shareable,omitempty" yaml:"shareable,omitempty"`
}

// SecurityConfig defines security requirements
type SecurityConfig struct {
	ImageScanning    bool     `json:"imageScanning,omitempty" yaml:"imageScanning,omitempty"`
	SecretScanning   bool     `json:"secretScanning,omitempty" yaml:"secretScanning,omitempty"`
	SAST             bool     `json:"sast,omitempty" yaml:"sast,omitempty"`
	DAST             bool     `json:"dast,omitempty" yaml:"dast,omitempty"`
	DependencyCheck  bool     `json:"dependencyCheck,omitempty" yaml:"dependencyCheck,omitempty"`
	ComplianceChecks []string `json:"complianceChecks,omitempty" yaml:"complianceChecks,omitempty"`
}

// PlatformCapabilities defines platform integrations
type PlatformCapabilities struct {
	Secrets       string `json:"secrets,omitempty" yaml:"secrets,omitempty"`             // vault, aws-secrets-manager
	GitOps        string `json:"gitops,omitempty" yaml:"gitops,omitempty"`               // argocd, flux
	CI            string `json:"ci,omitempty" yaml:"ci,omitempty"`                       // github-actions, gitlab-ci
	Monitoring    string `json:"monitoring,omitempty" yaml:"monitoring,omitempty"`       // prometheus, datadog
	Logging       string `json:"logging,omitempty" yaml:"logging,omitempty"`             // elasticsearch, loki
	Tracing       string `json:"tracing,omitempty" yaml:"tracing,omitempty"`             // jaeger, tempo
	ServiceMesh   string `json:"serviceMesh,omitempty" yaml:"serviceMesh,omitempty"`     // istio, linkerd
	DNS           string `json:"dns,omitempty" yaml:"dns,omitempty"`                     // route53, cloudflare
	CDN           string `json:"cdn,omitempty" yaml:"cdn,omitempty"`                     // cloudfront, fastly
	Registry      string `json:"registry,omitempty" yaml:"registry,omitempty"`           // ecr, gcr, dockerhub
	Notifications string `json:"notifications,omitempty" yaml:"notifications,omitempty"` // slack, teams
}

// PlatformPolicies defines global platform policies
type PlatformPolicies struct {
	Security   SecurityPolicies   `json:"security,omitempty" yaml:"security,omitempty"`
	Cost       CostPolicies       `json:"cost,omitempty" yaml:"cost,omitempty"`
	Compliance CompliancePolicies `json:"compliance,omitempty" yaml:"compliance,omitempty"`
	Network    NetworkPolicies    `json:"network,omitempty" yaml:"network,omitempty"`
}

// SecurityPolicies defines security requirements
type SecurityPolicies struct {
	ImageScanning     string   `json:"imageScanning,omitempty" yaml:"imageScanning,omitempty"`     // required, recommended, disabled
	SecretsInEnv      string   `json:"secretsInEnv,omitempty" yaml:"secretsInEnv,omitempty"`       // denied, warn, allowed
	PublicEndpoints   string   `json:"publicEndpoints,omitempty" yaml:"publicEndpoints,omitempty"` // require-auth, allow, deny
	PrivilegedPods    string   `json:"privilegedPods,omitempty" yaml:"privilegedPods,omitempty"`   // denied, restricted, allowed
	RootUser          string   `json:"rootUser,omitempty" yaml:"rootUser,omitempty"`               // denied, warn, allowed
	AllowedRegistries []string `json:"allowedRegistries,omitempty" yaml:"allowedRegistries,omitempty"`
	DeniedImages      []string `json:"deniedImages,omitempty" yaml:"deniedImages,omitempty"`
	RequiredLabels    []string `json:"requiredLabels,omitempty" yaml:"requiredLabels,omitempty"`
}

// CostPolicies defines cost management rules
type CostPolicies struct {
	BudgetAlerts          bool   `json:"budgetAlerts,omitempty" yaml:"budgetAlerts,omitempty"`
	UnusedResourceCleanup string `json:"unusedResourceCleanup,omitempty" yaml:"unusedResourceCleanup,omitempty"` // 7d, 14d, etc.
	MaxCostPerApp         string `json:"maxCostPerApp,omitempty" yaml:"maxCostPerApp,omitempty"`
	MaxCostPerTeam        string `json:"maxCostPerTeam,omitempty" yaml:"maxCostPerTeam,omitempty"`
	RequireSpotForDev     bool   `json:"requireSpotForDev,omitempty" yaml:"requireSpotForDev,omitempty"`
	ShowbackEnabled       bool   `json:"showbackEnabled,omitempty" yaml:"showbackEnabled,omitempty"`
}

// CompliancePolicies defines compliance requirements
type CompliancePolicies struct {
	Frameworks    []string `json:"frameworks,omitempty" yaml:"frameworks,omitempty"` // SOC2, HIPAA, PCI-DSS
	DataResidency []string `json:"dataResidency,omitempty" yaml:"dataResidency,omitempty"`
	AuditLogging  bool     `json:"auditLogging,omitempty" yaml:"auditLogging,omitempty"`
	Encryption    bool     `json:"encryption,omitempty" yaml:"encryption,omitempty"`
}

// NetworkPolicies defines network requirements
type NetworkPolicies struct {
	DefaultDeny   bool     `json:"defaultDeny,omitempty" yaml:"defaultDeny,omitempty"`
	AllowedEgress []string `json:"allowedEgress,omitempty" yaml:"allowedEgress,omitempty"`
	RequireTLS    bool     `json:"requireTLS,omitempty" yaml:"requireTLS,omitempty"`
	MinTLSVersion string   `json:"minTLSVersion,omitempty" yaml:"minTLSVersion,omitempty"`
}

// EnvironmentDefinition defines an environment
type EnvironmentDefinition struct {
	Name      string                 `json:"name" yaml:"name"`
	Type      string                 `json:"type" yaml:"type"` // dev, staging, production
	Cluster   string                 `json:"cluster,omitempty" yaml:"cluster,omitempty"`
	Namespace string                 `json:"namespace,omitempty" yaml:"namespace,omitempty"`
	Overrides map[string]interface{} `json:"overrides,omitempty" yaml:"overrides,omitempty"`
	Promotion *PromotionConfig       `json:"promotion,omitempty" yaml:"promotion,omitempty"`
}

// PromotionConfig defines how to promote to this environment
type PromotionConfig struct {
	From          string   `json:"from,omitempty" yaml:"from,omitempty"`
	RequireTests  bool     `json:"requireTests,omitempty" yaml:"requireTests,omitempty"`
	RequireReview bool     `json:"requireReview,omitempty" yaml:"requireReview,omitempty"`
	Reviewers     []string `json:"reviewers,omitempty" yaml:"reviewers,omitempty"`
	AutoPromote   bool     `json:"autoPromote,omitempty" yaml:"autoPromote,omitempty"`
}

// PlatformDefaults defines default values
type PlatformDefaults struct {
	Replicas     int               `json:"replicas,omitempty" yaml:"replicas,omitempty"`
	CPU          string            `json:"cpu,omitempty" yaml:"cpu,omitempty"`
	Memory       string            `json:"memory,omitempty" yaml:"memory,omitempty"`
	Labels       map[string]string `json:"labels,omitempty" yaml:"labels,omitempty"`
	Annotations  map[string]string `json:"annotations,omitempty" yaml:"annotations,omitempty"`
	NodeSelector map[string]string `json:"nodeSelector,omitempty" yaml:"nodeSelector,omitempty"`
}

// PlatformStatus tracks platform state
type PlatformStatus struct {
	Phase            PlatformPhase       `json:"phase" yaml:"phase"`
	Version          string              `json:"version,omitempty" yaml:"version,omitempty"`
	LastApplied      time.Time           `json:"lastApplied,omitempty" yaml:"lastApplied,omitempty"`
	Conditions       []PlatformCondition `json:"conditions,omitempty" yaml:"conditions,omitempty"`
	DriftDetected    bool                `json:"driftDetected,omitempty" yaml:"driftDetected,omitempty"`
	DriftDetails     []DriftDetail       `json:"driftDetails,omitempty" yaml:"driftDetails,omitempty"`
	ApplicationCount int                 `json:"applicationCount,omitempty" yaml:"applicationCount,omitempty"`
	TeamCount        int                 `json:"teamCount,omitempty" yaml:"teamCount,omitempty"`
}

// PlatformPhase indicates platform lifecycle phase
type PlatformPhase string

const (
	PlatformPhaseActive    PlatformPhase = "Active"
	PlatformPhasePending   PlatformPhase = "Pending"
	PlatformPhaseUpgrading PlatformPhase = "Upgrading"
	PlatformPhaseDegraded  PlatformPhase = "Degraded"
	PlatformPhaseFailed    PlatformPhase = "Failed"
)

// PlatformCondition represents a platform condition
type PlatformCondition struct {
	Type               string    `json:"type" yaml:"type"`
	Status             string    `json:"status" yaml:"status"`
	LastTransitionTime time.Time `json:"lastTransitionTime" yaml:"lastTransitionTime"`
	Reason             string    `json:"reason,omitempty" yaml:"reason,omitempty"`
	Message            string    `json:"message,omitempty" yaml:"message,omitempty"`
}

// DriftDetail describes a configuration drift
type DriftDetail struct {
	Component string `json:"component" yaml:"component"`
	Expected  string `json:"expected" yaml:"expected"`
	Actual    string `json:"actual" yaml:"actual"`
	Severity  string `json:"severity" yaml:"severity"`
}

// Application represents an application using a golden path
type Application struct {
	APIVersion string              `json:"apiVersion" yaml:"apiVersion"`
	Kind       string              `json:"kind" yaml:"kind"`
	Metadata   ApplicationMetadata `json:"metadata" yaml:"metadata"`
	Spec       ApplicationSpec     `json:"spec" yaml:"spec"`
	Status     *ApplicationStatus  `json:"status,omitempty" yaml:"status,omitempty"`
}

// ApplicationMetadata contains application identification
type ApplicationMetadata struct {
	Name        string            `json:"name" yaml:"name"`
	Namespace   string            `json:"namespace,omitempty" yaml:"namespace,omitempty"`
	Team        string            `json:"team,omitempty" yaml:"team,omitempty"`
	Labels      map[string]string `json:"labels,omitempty" yaml:"labels,omitempty"`
	Annotations map[string]string `json:"annotations,omitempty" yaml:"annotations,omitempty"`
}

// ApplicationSpec defines application configuration
type ApplicationSpec struct {
	GoldenPath   string                 `json:"goldenPath" yaml:"goldenPath"`
	Platform     string                 `json:"platform,omitempty" yaml:"platform,omitempty"`
	Repository   string                 `json:"repository,omitempty" yaml:"repository,omitempty"`
	Branch       string                 `json:"branch,omitempty" yaml:"branch,omitempty"`
	Config       map[string]interface{} `json:"config,omitempty" yaml:"config,omitempty"`
	Resources    []ResourceInstance     `json:"resources,omitempty" yaml:"resources,omitempty"`
	Environments []string               `json:"environments,omitempty" yaml:"environments,omitempty"`
}

// ResourceInstance represents a provisioned resource
type ResourceInstance struct {
	Name       string                 `json:"name" yaml:"name"`
	Type       string                 `json:"type" yaml:"type"`
	Config     map[string]interface{} `json:"config,omitempty" yaml:"config,omitempty"`
	Shared     bool                   `json:"shared,omitempty" yaml:"shared,omitempty"`
	SharedFrom string                 `json:"sharedFrom,omitempty" yaml:"sharedFrom,omitempty"`
}

// ApplicationStatus tracks application state
type ApplicationStatus struct {
	Phase        string                    `json:"phase" yaml:"phase"`
	Deployments  map[string]DeploymentInfo `json:"deployments,omitempty" yaml:"deployments,omitempty"`
	Resources    []ResourceStatus          `json:"resources,omitempty" yaml:"resources,omitempty"`
	LastDeployed time.Time                 `json:"lastDeployed,omitempty" yaml:"lastDeployed,omitempty"`
}

// DeploymentInfo tracks deployment in an environment
type DeploymentInfo struct {
	Environment string    `json:"environment" yaml:"environment"`
	Version     string    `json:"version" yaml:"version"`
	Status      string    `json:"status" yaml:"status"`
	Replicas    int       `json:"replicas" yaml:"replicas"`
	Ready       int       `json:"ready" yaml:"ready"`
	UpdatedAt   time.Time `json:"updatedAt" yaml:"updatedAt"`
}

// ResourceStatus tracks resource provisioning
type ResourceStatus struct {
	Name      string    `json:"name" yaml:"name"`
	Type      string    `json:"type" yaml:"type"`
	Status    string    `json:"status" yaml:"status"`
	Endpoint  string    `json:"endpoint,omitempty" yaml:"endpoint,omitempty"`
	CreatedAt time.Time `json:"createdAt,omitempty" yaml:"createdAt,omitempty"`
}
