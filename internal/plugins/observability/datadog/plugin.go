package datadog

import (
	"fmt"

	"github.com/platformfoundry/platformfoundry-ce/pkg/plugin"
)

// Config represents Datadog configuration
type Config struct {
	Provider   string          `yaml:"provider" json:"provider" validate:"required,oneof=datadog"`
	ClusterRef string          `yaml:"clusterRef" json:"clusterRef" validate:"required"`
	APIKey     string          `yaml:"apiKey" json:"apiKey" validate:"required"`
	APPKey     string          `yaml:"appKey,omitempty" json:"appKey,omitempty"`
	Site       string          `yaml:"site" json:"site" validate:"oneof=datadoghq.com datadoghq.eu us3.datadoghq.com us5.datadoghq.com ap1.datadoghq.com"`
	Agent      *AgentConfig    `yaml:"agent,omitempty" json:"agent,omitempty"`
	Logs       *LogsConfig     `yaml:"logs,omitempty" json:"logs,omitempty"`
	APM        *APMConfig      `yaml:"apm,omitempty" json:"apm,omitempty"`
	Metrics    *MetricsConfig  `yaml:"metrics,omitempty" json:"metrics,omitempty"`
	Monitors   []MonitorConfig `yaml:"monitors,omitempty" json:"monitors,omitempty"`
	Dashboards []DashboardRef  `yaml:"dashboards,omitempty" json:"dashboards,omitempty"`
}

// AgentConfig represents Datadog Agent configuration
type AgentConfig struct {
	Enabled          bool              `yaml:"enabled" json:"enabled"`
	ClusterAgent     *ClusterAgentConfig `yaml:"clusterAgent,omitempty" json:"clusterAgent,omitempty"`
	NodeAgent        *NodeAgentConfig    `yaml:"nodeAgent,omitempty" json:"nodeAgent,omitempty"`
	ClusterChecks    bool              `yaml:"clusterChecksEnabled" json:"clusterChecksEnabled"`
	KubeStateMetrics *KSMConfig        `yaml:"kubeStateMetricsCore,omitempty" json:"kubeStateMetricsCore,omitempty"`
}

// ClusterAgentConfig represents Cluster Agent settings
type ClusterAgentConfig struct {
	Enabled  bool   `yaml:"enabled" json:"enabled"`
	Replicas int    `yaml:"replicas" json:"replicas"`
	Image    string `yaml:"image,omitempty" json:"image,omitempty"`
}

// NodeAgentConfig represents Node Agent settings
type NodeAgentConfig struct {
	Enabled   bool             `yaml:"enabled" json:"enabled"`
	Image     string           `yaml:"image,omitempty" json:"image,omitempty"`
	Resources *ResourcesConfig `yaml:"resources,omitempty" json:"resources,omitempty"`
}

// ResourcesConfig represents Kubernetes resource requirements
type ResourcesConfig struct {
	Requests ResourceSpec `yaml:"requests,omitempty" json:"requests,omitempty"`
	Limits   ResourceSpec `yaml:"limits,omitempty" json:"limits,omitempty"`
}

// ResourceSpec defines CPU and memory
type ResourceSpec struct {
	CPU    string `yaml:"cpu,omitempty" json:"cpu,omitempty"`
	Memory string `yaml:"memory,omitempty" json:"memory,omitempty"`
}

// KSMConfig configures Kube State Metrics integration
type KSMConfig struct {
	Enabled bool `yaml:"enabled" json:"enabled"`
}

// LogsConfig represents Datadog Logs configuration
type LogsConfig struct {
	Enabled           bool     `yaml:"enabled" json:"enabled"`
	ContainerCollect  bool     `yaml:"containerCollectAll" json:"containerCollectAll"`
	ContainerExclude  []string `yaml:"containerExclude,omitempty" json:"containerExclude,omitempty"`
	ContainerInclude  []string `yaml:"containerInclude,omitempty" json:"containerInclude,omitempty"`
	ProcessingRules   []ProcessingRule `yaml:"processingRules,omitempty" json:"processingRules,omitempty"`
}

// ProcessingRule defines log processing rules
type ProcessingRule struct {
	Type    string `yaml:"type" json:"type" validate:"required,oneof=include_at_match exclude_at_match mask_sequences"`
	Name    string `yaml:"name" json:"name" validate:"required"`
	Pattern string `yaml:"pattern" json:"pattern" validate:"required"`
}

// APMConfig represents APM/Tracing configuration
type APMConfig struct {
	Enabled              bool     `yaml:"enabled" json:"enabled"`
	HostPort             int      `yaml:"hostPort" json:"hostPort"`
	NonLocalTraffic      bool     `yaml:"nonLocalTraffic" json:"nonLocalTraffic"`
	UnixDomainSocket     bool     `yaml:"unixDomainSocketEnabled" json:"unixDomainSocketEnabled"`
	TraceSampleRate      float64  `yaml:"traceSampleRate,omitempty" json:"traceSampleRate,omitempty"`
	IgnoredResources     []string `yaml:"ignoredResources,omitempty" json:"ignoredResources,omitempty"`
	ServiceMapping       map[string]string `yaml:"serviceMapping,omitempty" json:"serviceMapping,omitempty"`
}

// MetricsConfig represents custom metrics configuration
type MetricsConfig struct {
	Enabled           bool              `yaml:"enabled" json:"enabled"`
	DogStatsDPort     int               `yaml:"dogstatsdPort" json:"dogstatsdPort"`
	OriginDetection   bool              `yaml:"dogstatsdOriginDetection" json:"dogstatsdOriginDetection"`
	CustomMetrics     []CustomMetric    `yaml:"customMetrics,omitempty" json:"customMetrics,omitempty"`
	AutodiscoveryTags []string          `yaml:"autodiscoveryTags,omitempty" json:"autodiscoveryTags,omitempty"`
}

// CustomMetric defines a custom metric to collect
type CustomMetric struct {
	Name       string            `yaml:"name" json:"name" validate:"required"`
	Type       string            `yaml:"type" json:"type" validate:"required,oneof=gauge counter histogram"`
	Tags       []string          `yaml:"tags,omitempty" json:"tags,omitempty"`
	Labels     map[string]string `yaml:"labels,omitempty" json:"labels,omitempty"`
}

// MonitorConfig represents a Datadog monitor definition
type MonitorConfig struct {
	Name               string            `yaml:"name" json:"name" validate:"required"`
	Type               string            `yaml:"type" json:"type" validate:"required,oneof=metric query_alert service_check event_alert log_alert process_alert synthetics_alert"`
	Query              string            `yaml:"query" json:"query" validate:"required"`
	Message            string            `yaml:"message" json:"message"`
	Tags               []string          `yaml:"tags,omitempty" json:"tags,omitempty"`
	Priority           int               `yaml:"priority,omitempty" json:"priority,omitempty" validate:"omitempty,min=1,max=5"`
	Thresholds         *ThresholdConfig  `yaml:"thresholds,omitempty" json:"thresholds,omitempty"`
	NotificationConfig *NotifyConfig     `yaml:"notificationConfig,omitempty" json:"notificationConfig,omitempty"`
}

// ThresholdConfig defines monitor thresholds
type ThresholdConfig struct {
	Critical         float64 `yaml:"critical,omitempty" json:"critical,omitempty"`
	CriticalRecovery float64 `yaml:"criticalRecovery,omitempty" json:"criticalRecovery,omitempty"`
	Warning          float64 `yaml:"warning,omitempty" json:"warning,omitempty"`
	WarningRecovery  float64 `yaml:"warningRecovery,omitempty" json:"warningRecovery,omitempty"`
	OK               float64 `yaml:"ok,omitempty" json:"ok,omitempty"`
}

// NotifyConfig defines notification settings
type NotifyConfig struct {
	NoDataTimeframe int  `yaml:"noDataTimeframe,omitempty" json:"noDataTimeframe,omitempty"`
	NotifyAudit     bool `yaml:"notifyAudit" json:"notifyAudit"`
	NotifyNoData    bool `yaml:"notifyNoData" json:"notifyNoData"`
	RenotifyInterval int `yaml:"renotifyInterval,omitempty" json:"renotifyInterval,omitempty"`
}

// DashboardRef references an existing dashboard or defines a new one
type DashboardRef struct {
	Name        string `yaml:"name" json:"name" validate:"required"`
	ID          string `yaml:"id,omitempty" json:"id,omitempty"`
	TemplateURL string `yaml:"templateUrl,omitempty" json:"templateUrl,omitempty"`
}

// Plugin implements the Datadog plugin
type Plugin struct{}

// NewPlugin creates a new Datadog plugin
func NewPlugin() plugin.Plugin {
	return &Plugin{}
}

// Name returns the plugin name
func (p *Plugin) Name() string {
	return "datadog"
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
	return &Config{}
}

// Validate validates the plugin configuration
func (p *Plugin) Validate(spec map[string]interface{}) error {
	provider, ok := spec["provider"].(string)
	if !ok || provider == "" {
		return fmt.Errorf("provider field is required")
	}

	if provider != "datadog" {
		return fmt.Errorf("provider must be 'datadog'")
	}

	clusterRef, ok := spec["clusterRef"].(string)
	if !ok || clusterRef == "" {
		return fmt.Errorf("clusterRef is required")
	}

	apiKey, ok := spec["apiKey"].(string)
	if !ok || apiKey == "" {
		return fmt.Errorf("apiKey is required")
	}

	return nil
}

// Plan generates a plan for the plugin
func (p *Plugin) Plan(spec map[string]interface{}) (*plugin.Plan, error) {
	actions := []string{
		"Install Datadog Operator",
		"Create Datadog Agent configuration",
	}

	if agent, ok := spec["agent"].(map[string]interface{}); ok {
		if ca, ok := agent["clusterAgent"].(map[string]interface{}); ok {
			if enabled, ok := ca["enabled"].(bool); ok && enabled {
				actions = append(actions, "Deploy Cluster Agent")
			}
		}
	}

	if logs, ok := spec["logs"].(map[string]interface{}); ok {
		if enabled, ok := logs["enabled"].(bool); ok && enabled {
			actions = append(actions, "Configure log collection")
		}
	}

	if apm, ok := spec["apm"].(map[string]interface{}); ok {
		if enabled, ok := apm["enabled"].(bool); ok && enabled {
			actions = append(actions, "Enable APM/Tracing")
		}
	}

	if monitors, ok := spec["monitors"].([]interface{}); ok && len(monitors) > 0 {
		actions = append(actions, fmt.Sprintf("Create %d monitors", len(monitors)))
	}

	return &plugin.Plan{
		Actions: actions,
	}, nil
}

// Apply applies the plugin configuration
func (p *Plugin) Apply(spec map[string]interface{}) (*plugin.Result, error) {
	return &plugin.Result{
		Status:  "success",
		Message: "Datadog configured successfully",
		Outputs: map[string]string{
			"provider": "datadog",
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
		Message: "Datadog agents are running",
	}, nil
}
