package types

// Observability represents observability resources
// Implements US-1.4: Observability Resource Definition
type Observability struct {
	APIVersion string              `yaml:"apiVersion" json:"apiVersion"`
	Kind       string              `yaml:"kind" json:"kind"`
	Metadata   Metadata            `yaml:"metadata" json:"metadata"`
	Spec       ObservabilitySpec   `yaml:"spec" json:"spec"`
	Status     ObservabilityStatus `yaml:"status,omitempty" json:"status,omitempty"`
}

// ObservabilitySpec defines observability specification
type ObservabilitySpec struct {
	ClusterRef    string               `yaml:"clusterRef" json:"clusterRef"`
	Monitoring    *MonitoringConfig    `yaml:"monitoring,omitempty" json:"monitoring,omitempty"`
	Visualization *VisualizationConfig `yaml:"visualization,omitempty" json:"visualization,omitempty"`
	Logging       *LoggingConfig       `yaml:"logging,omitempty" json:"logging,omitempty"`
	Ingress       *IngressConfig       `yaml:"ingress,omitempty" json:"ingress,omitempty"`
}

// MonitoringConfig defines monitoring configuration
type MonitoringConfig struct {
	Provider       string `yaml:"provider" json:"provider"` // prometheus
	Retention      string `yaml:"retention,omitempty" json:"retention,omitempty"`
	StorageSize    string `yaml:"storageSize,omitempty" json:"storageSize,omitempty"`
	ScrapeInterval string `yaml:"scrapeInterval,omitempty" json:"scrapeInterval,omitempty"`
}

// VisualizationConfig defines visualization configuration
type VisualizationConfig struct {
	Provider   string   `yaml:"provider" json:"provider"` // grafana
	Dashboards []string `yaml:"dashboards,omitempty" json:"dashboards,omitempty"`
}

// LoggingConfig defines logging configuration
type LoggingConfig struct {
	Provider    string `yaml:"provider" json:"provider"` // loki
	Retention   string `yaml:"retention,omitempty" json:"retention,omitempty"`
	StorageSize string `yaml:"storageSize,omitempty" json:"storageSize,omitempty"`
}

// IngressConfig defines ingress configuration
type IngressConfig struct {
	Provider string   `yaml:"provider" json:"provider"` // traefik, nginx
	Domains  []string `yaml:"domains,omitempty" json:"domains,omitempty"`
}

// ObservabilityStatus represents observability status
type ObservabilityStatus struct {
	Phase   Phase  `json:"phase"`
	Message string `json:"message,omitempty"`
}

// Validate validates the observability resource
func (o *Observability) Validate() error {
	if o.APIVersion == "" {
		return ErrMissingAPIVersion
	}
	if o.Kind != "Observability" {
		return ErrInvalidKind
	}
	if o.Metadata.Name == "" {
		return ErrMissingName
	}
	if o.Spec.ClusterRef == "" {
		return ErrMissingClusterRef
	}
	return nil
}
