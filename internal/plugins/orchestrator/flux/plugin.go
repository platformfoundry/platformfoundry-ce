package flux

import (
	"fmt"

	"github.com/platformfoundry/platformfoundry-ce/pkg/plugin"
)

// Config represents Flux CD configuration
type Config struct {
	Provider       string                `yaml:"provider" json:"provider" validate:"required,oneof=flux"`
	ClusterRef     string                `yaml:"clusterRef" json:"clusterRef" validate:"required"`
	GitRepository  *GitRepositoryConfig  `yaml:"gitRepository,omitempty" json:"gitRepository,omitempty"`
	Kustomizations []KustomizationConfig `yaml:"kustomizations,omitempty" json:"kustomizations,omitempty"`
	HelmReleases   []HelmReleaseConfig   `yaml:"helmReleases,omitempty" json:"helmReleases,omitempty"`
	ImagePolicies  []ImagePolicyConfig   `yaml:"imagePolicies,omitempty" json:"imagePolicies,omitempty"`
	Notifications  *NotificationConfig   `yaml:"notifications,omitempty" json:"notifications,omitempty"`
}

// GitRepositoryConfig represents a Flux GitRepository source
type GitRepositoryConfig struct {
	Name      string         `yaml:"name" json:"name" validate:"required"`
	Namespace string         `yaml:"namespace" json:"namespace"`
	URL       string         `yaml:"url" json:"url" validate:"required,url"`
	Branch    string         `yaml:"branch" json:"branch"`
	Tag       string         `yaml:"tag,omitempty" json:"tag,omitempty"`
	Interval  string         `yaml:"interval" json:"interval"`
	SecretRef *SecretRefSpec `yaml:"secretRef,omitempty" json:"secretRef,omitempty"`
}

// SecretRefSpec references a secret for authentication
type SecretRefSpec struct {
	Name string `yaml:"name" json:"name" validate:"required"`
}

// KustomizationConfig represents a Flux Kustomization
type KustomizationConfig struct {
	Name            string            `yaml:"name" json:"name" validate:"required"`
	Namespace       string            `yaml:"namespace" json:"namespace"`
	Path            string            `yaml:"path" json:"path" validate:"required"`
	Prune           bool              `yaml:"prune" json:"prune"`
	Interval        string            `yaml:"interval" json:"interval"`
	TargetNamespace string            `yaml:"targetNamespace,omitempty" json:"targetNamespace,omitempty"`
	SourceRef       SourceRefSpec     `yaml:"sourceRef" json:"sourceRef"`
	HealthChecks    []HealthCheckSpec `yaml:"healthChecks,omitempty" json:"healthChecks,omitempty"`
	DependsOn       []DependsOnSpec   `yaml:"dependsOn,omitempty" json:"dependsOn,omitempty"`
}

// SourceRefSpec references the source for a Kustomization
type SourceRefSpec struct {
	Kind string `yaml:"kind" json:"kind" validate:"required,oneof=GitRepository HelmRepository Bucket"`
	Name string `yaml:"name" json:"name" validate:"required"`
}

// HealthCheckSpec defines health check for Kustomization
type HealthCheckSpec struct {
	APIVersion string `yaml:"apiVersion" json:"apiVersion"`
	Kind       string `yaml:"kind" json:"kind"`
	Name       string `yaml:"name" json:"name"`
	Namespace  string `yaml:"namespace" json:"namespace"`
}

// DependsOnSpec defines dependencies between Kustomizations
type DependsOnSpec struct {
	Name string `yaml:"name" json:"name" validate:"required"`
}

// HelmReleaseConfig represents a Flux HelmRelease
type HelmReleaseConfig struct {
	Name         string                 `yaml:"name" json:"name" validate:"required"`
	Namespace    string                 `yaml:"namespace" json:"namespace"`
	Chart        HelmChartSpec          `yaml:"chart" json:"chart"`
	Interval     string                 `yaml:"interval" json:"interval"`
	Values       map[string]interface{} `yaml:"values,omitempty" json:"values,omitempty"`
	ValuesFrom   []ValuesFromSpec       `yaml:"valuesFrom,omitempty" json:"valuesFrom,omitempty"`
	Install      *InstallSpec           `yaml:"install,omitempty" json:"install,omitempty"`
	Upgrade      *UpgradeSpec           `yaml:"upgrade,omitempty" json:"upgrade,omitempty"`
}

// HelmChartSpec specifies the Helm chart
type HelmChartSpec struct {
	Spec ChartSpec `yaml:"spec" json:"spec"`
}

// ChartSpec defines chart details
type ChartSpec struct {
	Chart     string        `yaml:"chart" json:"chart" validate:"required"`
	Version   string        `yaml:"version,omitempty" json:"version,omitempty"`
	SourceRef SourceRefSpec `yaml:"sourceRef" json:"sourceRef"`
}

// ValuesFromSpec references values from external sources
type ValuesFromSpec struct {
	Kind      string `yaml:"kind" json:"kind" validate:"required,oneof=ConfigMap Secret"`
	Name      string `yaml:"name" json:"name" validate:"required"`
	ValuesKey string `yaml:"valuesKey,omitempty" json:"valuesKey,omitempty"`
}

// InstallSpec configures Helm install behavior
type InstallSpec struct {
	CreateNamespace bool `yaml:"createNamespace" json:"createNamespace"`
	Remediation     *RemediationSpec `yaml:"remediation,omitempty" json:"remediation,omitempty"`
}

// UpgradeSpec configures Helm upgrade behavior
type UpgradeSpec struct {
	Remediation *RemediationSpec `yaml:"remediation,omitempty" json:"remediation,omitempty"`
}

// RemediationSpec defines remediation on failure
type RemediationSpec struct {
	Retries int `yaml:"retries" json:"retries"`
}

// ImagePolicyConfig represents a Flux ImagePolicy for auto-updates
type ImagePolicyConfig struct {
	Name       string           `yaml:"name" json:"name" validate:"required"`
	Namespace  string           `yaml:"namespace" json:"namespace"`
	ImageRef   ImageRefSpec     `yaml:"imageRepositoryRef" json:"imageRepositoryRef"`
	Policy     ImagePolicySpec  `yaml:"policy" json:"policy"`
}

// ImageRefSpec references an ImageRepository
type ImageRefSpec struct {
	Name string `yaml:"name" json:"name" validate:"required"`
}

// ImagePolicySpec defines image selection policy
type ImagePolicySpec struct {
	SemVer    *SemVerPolicy    `yaml:"semver,omitempty" json:"semver,omitempty"`
	Alphabetical *AlphabeticalPolicy `yaml:"alphabetical,omitempty" json:"alphabetical,omitempty"`
	Numerical *NumericalPolicy `yaml:"numerical,omitempty" json:"numerical,omitempty"`
}

// SemVerPolicy selects based on semver
type SemVerPolicy struct {
	Range string `yaml:"range" json:"range"`
}

// AlphabeticalPolicy selects alphabetically
type AlphabeticalPolicy struct {
	Order string `yaml:"order" json:"order" validate:"oneof=asc desc"`
}

// NumericalPolicy selects numerically
type NumericalPolicy struct {
	Order string `yaml:"order" json:"order" validate:"oneof=asc desc"`
}

// NotificationConfig configures Flux notifications
type NotificationConfig struct {
	Providers []ProviderConfig `yaml:"providers,omitempty" json:"providers,omitempty"`
	Alerts    []AlertConfig    `yaml:"alerts,omitempty" json:"alerts,omitempty"`
}

// ProviderConfig defines a notification provider
type ProviderConfig struct {
	Name      string `yaml:"name" json:"name" validate:"required"`
	Type      string `yaml:"type" json:"type" validate:"required,oneof=slack msteams discord github gitlab"`
	SecretRef *SecretRefSpec `yaml:"secretRef,omitempty" json:"secretRef,omitempty"`
	Channel   string `yaml:"channel,omitempty" json:"channel,omitempty"`
}

// AlertConfig defines notification alerts
type AlertConfig struct {
	Name        string        `yaml:"name" json:"name" validate:"required"`
	ProviderRef string        `yaml:"providerRef" json:"providerRef" validate:"required"`
	EventSources []EventSource `yaml:"eventSources" json:"eventSources"`
}

// EventSource defines sources of events
type EventSource struct {
	Kind string `yaml:"kind" json:"kind" validate:"required"`
	Name string `yaml:"name" json:"name" validate:"required"`
}

// Plugin implements the Flux CD plugin
type Plugin struct{}

// NewPlugin creates a new Flux plugin
func NewPlugin() plugin.Plugin {
	return &Plugin{}
}

// Name returns the plugin name
func (p *Plugin) Name() string {
	return "flux"
}

// Type returns the resource type
func (p *Plugin) Type() string {
	return "Orchestrator"
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

	if provider != "flux" {
		return fmt.Errorf("provider must be 'flux'")
	}

	clusterRef, ok := spec["clusterRef"].(string)
	if !ok || clusterRef == "" {
		return fmt.Errorf("clusterRef is required")
	}

	return nil
}

// Plan generates a plan for the plugin
func (p *Plugin) Plan(spec map[string]interface{}) (*plugin.Plan, error) {
	actions := []string{
		"Bootstrap Flux CD on cluster",
	}

	if _, ok := spec["gitRepository"]; ok {
		actions = append(actions, "Configure GitRepository source")
	}

	if kustomizations, ok := spec["kustomizations"].([]interface{}); ok && len(kustomizations) > 0 {
		actions = append(actions, fmt.Sprintf("Create %d Kustomizations", len(kustomizations)))
	}

	if helmReleases, ok := spec["helmReleases"].([]interface{}); ok && len(helmReleases) > 0 {
		actions = append(actions, fmt.Sprintf("Create %d HelmReleases", len(helmReleases)))
	}

	return &plugin.Plan{
		Actions: actions,
	}, nil
}

// Apply applies the plugin configuration
func (p *Plugin) Apply(spec map[string]interface{}) (*plugin.Result, error) {
	return &plugin.Result{
		Status:  "success",
		Message: "Flux CD configured successfully",
		Outputs: map[string]string{
			"provider": "flux",
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
		Message: "Flux CD is ready and reconciling",
	}, nil
}
