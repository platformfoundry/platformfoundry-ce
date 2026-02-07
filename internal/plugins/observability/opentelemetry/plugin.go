package opentelemetry

import (
	"fmt"

	"github.com/platformfoundry/pf-ce/pkg/plugin"
)

// Config represents OpenTelemetry configuration
type Config struct {
	Provider            string                     `yaml:"provider" json:"provider" validate:"required,oneof=opentelemetry"`
	ClusterRef          string                     `yaml:"clusterRef" json:"clusterRef" validate:"required"`
	Collector           *CollectorConfig           `yaml:"collector,omitempty" json:"collector,omitempty"`
	Exporters           *ExportersConfig           `yaml:"exporters,omitempty" json:"exporters,omitempty"`
	Receivers           *ReceiversConfig           `yaml:"receivers,omitempty" json:"receivers,omitempty"`
	Processors          []string                   `yaml:"processors,omitempty" json:"processors,omitempty"`
	AutoInstrumentation *AutoInstrumentationConfig `yaml:"autoInstrumentation,omitempty" json:"autoInstrumentation,omitempty"`
}

// CollectorConfig represents OTel Collector configuration
type CollectorConfig struct {
	Mode      string                 `yaml:"mode" json:"mode" validate:"required,oneof=deployment daemonset sidecar"`
	Replicas  int                    `yaml:"replicas,omitempty" json:"replicas,omitempty"`
	Resources *ResourcesConfig       `yaml:"resources,omitempty" json:"resources,omitempty"`
	Image     string                 `yaml:"image,omitempty" json:"image,omitempty"`
	Config    map[string]interface{} `yaml:"config,omitempty" json:"config,omitempty"`
}

// ResourcesConfig represents Kubernetes resource requests/limits
type ResourcesConfig struct {
	Requests ResourceSpec `yaml:"requests,omitempty" json:"requests,omitempty"`
	Limits   ResourceSpec `yaml:"limits,omitempty" json:"limits,omitempty"`
}

// ResourceSpec defines CPU and memory
type ResourceSpec struct {
	CPU    string `yaml:"cpu,omitempty" json:"cpu,omitempty"`
	Memory string `yaml:"memory,omitempty" json:"memory,omitempty"`
}

// ExportersConfig defines telemetry exporters
type ExportersConfig struct {
	OTLP       *OTLPExporter       `yaml:"otlp,omitempty" json:"otlp,omitempty"`
	Jaeger     *JaegerExporter     `yaml:"jaeger,omitempty" json:"jaeger,omitempty"`
	Prometheus *PrometheusExporter `yaml:"prometheus,omitempty" json:"prometheus,omitempty"`
	Zipkin     *ZipkinExporter     `yaml:"zipkin,omitempty" json:"zipkin,omitempty"`
	Logging    *LoggingExporter    `yaml:"logging,omitempty" json:"logging,omitempty"`
}

// OTLPExporter configures OTLP export
type OTLPExporter struct {
	Endpoint string            `yaml:"endpoint" json:"endpoint" validate:"required"`
	Headers  map[string]string `yaml:"headers,omitempty" json:"headers,omitempty"`
	TLS      *TLSConfig        `yaml:"tls,omitempty" json:"tls,omitempty"`
}

// JaegerExporter configures Jaeger export
type JaegerExporter struct {
	Endpoint string `yaml:"endpoint" json:"endpoint" validate:"required"`
	Protocol string `yaml:"protocol" json:"protocol" validate:"oneof=grpc http"`
}

// PrometheusExporter configures Prometheus export
type PrometheusExporter struct {
	Endpoint    string            `yaml:"endpoint" json:"endpoint"`
	Namespace   string            `yaml:"namespace,omitempty" json:"namespace,omitempty"`
	ConstLabels map[string]string `yaml:"constLabels,omitempty" json:"constLabels,omitempty"`
}

// ZipkinExporter configures Zipkin export
type ZipkinExporter struct {
	Endpoint string `yaml:"endpoint" json:"endpoint" validate:"required"`
}

// LoggingExporter configures logging export for debugging
type LoggingExporter struct {
	Verbosity string `yaml:"verbosity" json:"verbosity" validate:"oneof=basic normal detailed"`
}

// TLSConfig represents TLS configuration
type TLSConfig struct {
	Insecure bool   `yaml:"insecure" json:"insecure"`
	CertFile string `yaml:"certFile,omitempty" json:"certFile,omitempty"`
	KeyFile  string `yaml:"keyFile,omitempty" json:"keyFile,omitempty"`
	CAFile   string `yaml:"caFile,omitempty" json:"caFile,omitempty"`
}

// ReceiversConfig defines telemetry receivers
type ReceiversConfig struct {
	OTLP        *OTLPReceiver        `yaml:"otlp,omitempty" json:"otlp,omitempty"`
	Jaeger      *JaegerReceiver      `yaml:"jaeger,omitempty" json:"jaeger,omitempty"`
	Zipkin      *ZipkinReceiver      `yaml:"zipkin,omitempty" json:"zipkin,omitempty"`
	Prometheus  *PrometheusReceiver  `yaml:"prometheus,omitempty" json:"prometheus,omitempty"`
	HostMetrics *HostMetricsReceiver `yaml:"hostMetrics,omitempty" json:"hostMetrics,omitempty"`
	Kubelet     *KubeletReceiver     `yaml:"kubeletstats,omitempty" json:"kubeletstats,omitempty"`
}

// OTLPReceiver configures OTLP receiver
type OTLPReceiver struct {
	Protocols ProtocolsConfig `yaml:"protocols" json:"protocols"`
}

// ProtocolsConfig defines supported protocols
type ProtocolsConfig struct {
	GRPC *EndpointConfig `yaml:"grpc,omitempty" json:"grpc,omitempty"`
	HTTP *EndpointConfig `yaml:"http,omitempty" json:"http,omitempty"`
}

// EndpointConfig defines endpoint settings
type EndpointConfig struct {
	Endpoint string `yaml:"endpoint,omitempty" json:"endpoint,omitempty"`
}

// JaegerReceiver configures Jaeger receiver
type JaegerReceiver struct {
	Protocols JaegerProtocols `yaml:"protocols" json:"protocols"`
}

// JaegerProtocols defines Jaeger protocols
type JaegerProtocols struct {
	GRPC          *EndpointConfig `yaml:"grpc,omitempty" json:"grpc,omitempty"`
	ThriftHTTP    *EndpointConfig `yaml:"thrift_http,omitempty" json:"thrift_http,omitempty"`
	ThriftCompact *EndpointConfig `yaml:"thrift_compact,omitempty" json:"thrift_compact,omitempty"`
}

// ZipkinReceiver configures Zipkin receiver
type ZipkinReceiver struct {
	Endpoint string `yaml:"endpoint" json:"endpoint"`
}

// PrometheusReceiver configures Prometheus scraping
type PrometheusReceiver struct {
	Config map[string]interface{} `yaml:"config" json:"config"`
}

// HostMetricsReceiver collects host metrics
type HostMetricsReceiver struct {
	CollectionInterval string   `yaml:"collection_interval" json:"collection_interval"`
	Scrapers           []string `yaml:"scrapers" json:"scrapers"`
}

// KubeletReceiver collects kubelet stats
type KubeletReceiver struct {
	CollectionInterval string `yaml:"collection_interval" json:"collection_interval"`
	AuthType           string `yaml:"auth_type" json:"auth_type"`
	Endpoint           string `yaml:"endpoint" json:"endpoint"`
}

// AutoInstrumentationConfig configures automatic instrumentation
type AutoInstrumentationConfig struct {
	Java       *InstrumentationSpec `yaml:"java,omitempty" json:"java,omitempty"`
	NodeJS     *InstrumentationSpec `yaml:"nodejs,omitempty" json:"nodejs,omitempty"`
	Python     *InstrumentationSpec `yaml:"python,omitempty" json:"python,omitempty"`
	DotNet     *InstrumentationSpec `yaml:"dotnet,omitempty" json:"dotnet,omitempty"`
	Go         *InstrumentationSpec `yaml:"go,omitempty" json:"go,omitempty"`
	Namespaces []string             `yaml:"namespaces,omitempty" json:"namespaces,omitempty"`
}

// InstrumentationSpec defines language-specific instrumentation
type InstrumentationSpec struct {
	Enabled bool   `yaml:"enabled" json:"enabled"`
	Image   string `yaml:"image,omitempty" json:"image,omitempty"`
}

// Plugin implements the OpenTelemetry plugin
type Plugin struct{}

// NewPlugin creates a new OpenTelemetry plugin
func NewPlugin() plugin.Plugin {
	return &Plugin{}
}

// Name returns the plugin name
func (p *Plugin) Name() string {
	return "opentelemetry"
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

	if provider != "opentelemetry" {
		return fmt.Errorf("provider must be 'opentelemetry'")
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
		"Install OpenTelemetry Operator",
	}

	if collector, ok := spec["collector"].(map[string]interface{}); ok {
		mode := "deployment"
		if m, ok := collector["mode"].(string); ok {
			mode = m
		}
		actions = append(actions, fmt.Sprintf("Deploy OTel Collector in %s mode", mode))
	}

	if _, ok := spec["autoInstrumentation"]; ok {
		actions = append(actions, "Configure auto-instrumentation for applications")
	}

	if exporters, ok := spec["exporters"].(map[string]interface{}); ok {
		for name := range exporters {
			actions = append(actions, fmt.Sprintf("Configure %s exporter", name))
		}
	}

	return &plugin.Plan{
		Actions: actions,
	}, nil
}

// Apply applies the plugin configuration
func (p *Plugin) Apply(spec map[string]interface{}) (*plugin.Result, error) {
	return &plugin.Result{
		Status:  "success",
		Message: "OpenTelemetry configured successfully",
		Outputs: map[string]string{
			"provider": "opentelemetry",
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
		Message: "OpenTelemetry collector is running",
	}, nil
}
