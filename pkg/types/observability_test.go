package types

import (
	"testing"
)

func TestObservability_Validate(t *testing.T) {
	tests := []struct {
		name    string
		obs     Observability
		wantErr bool
		errType error
	}{
		{
			name: "valid observability with monitoring",
			obs: Observability{
				APIVersion: "platformfoundry.io/v1",
				Kind:       "Observability",
				Metadata: Metadata{
					Name: "test-obs",
				},
				Spec: ObservabilitySpec{
					ClusterRef: "my-cluster",
					Monitoring: &MonitoringConfig{
						Provider:       "prometheus",
						Retention:      "30d",
						StorageSize:    "50Gi",
						ScrapeInterval: "30s",
					},
				},
			},
			wantErr: false,
		},
		{
			name: "valid observability with visualization",
			obs: Observability{
				APIVersion: "platformfoundry.io/v1",
				Kind:       "Observability",
				Metadata: Metadata{
					Name: "viz-obs",
				},
				Spec: ObservabilitySpec{
					ClusterRef: "my-cluster",
					Visualization: &VisualizationConfig{
						Provider: "grafana",
						Dashboards: []string{
							"kubernetes-overview",
							"application-metrics",
						},
					},
				},
			},
			wantErr: false,
		},
		{
			name: "valid observability with logging",
			obs: Observability{
				APIVersion: "platformfoundry.io/v1",
				Kind:       "Observability",
				Metadata: Metadata{
					Name: "logging-obs",
				},
				Spec: ObservabilitySpec{
					ClusterRef: "my-cluster",
					Logging: &LoggingConfig{
						Provider:    "loki",
						Retention:   "7d",
						StorageSize: "20Gi",
					},
				},
			},
			wantErr: false,
		},
		{
			name: "valid observability with ingress",
			obs: Observability{
				APIVersion: "platformfoundry.io/v1",
				Kind:       "Observability",
				Metadata: Metadata{
					Name: "ingress-obs",
				},
				Spec: ObservabilitySpec{
					ClusterRef: "my-cluster",
					Ingress: &IngressConfig{
						Provider: "traefik",
						Domains: []string{
							"metrics.example.com",
							"grafana.example.com",
						},
					},
				},
			},
			wantErr: false,
		},
		{
			name: "valid observability with all components",
			obs: Observability{
				APIVersion: "platformfoundry.io/v1",
				Kind:       "Observability",
				Metadata: Metadata{
					Name: "full-obs",
				},
				Spec: ObservabilitySpec{
					ClusterRef: "my-cluster",
					Monitoring: &MonitoringConfig{
						Provider:       "prometheus",
						Retention:      "90d",
						StorageSize:    "100Gi",
						ScrapeInterval: "15s",
					},
					Visualization: &VisualizationConfig{
						Provider: "grafana",
						Dashboards: []string{
							"default",
						},
					},
					Logging: &LoggingConfig{
						Provider:    "loki",
						Retention:   "30d",
						StorageSize: "50Gi",
					},
					Ingress: &IngressConfig{
						Provider: "nginx",
						Domains: []string{
							"monitoring.example.com",
						},
					},
				},
			},
			wantErr: false,
		},
		{
			name: "missing apiVersion",
			obs: Observability{
				Kind: "Observability",
				Metadata: Metadata{
					Name: "test",
				},
				Spec: ObservabilitySpec{
					ClusterRef: "my-cluster",
				},
			},
			wantErr: true,
			errType: ErrMissingAPIVersion,
		},
		{
			name: "invalid kind",
			obs: Observability{
				APIVersion: "platformfoundry.io/v1",
				Kind:       "InvalidKind",
				Metadata: Metadata{
					Name: "test",
				},
				Spec: ObservabilitySpec{
					ClusterRef: "my-cluster",
				},
			},
			wantErr: true,
			errType: ErrInvalidKind,
		},
		{
			name: "missing name",
			obs: Observability{
				APIVersion: "platformfoundry.io/v1",
				Kind:       "Observability",
				Metadata:   Metadata{},
				Spec: ObservabilitySpec{
					ClusterRef: "my-cluster",
				},
			},
			wantErr: true,
			errType: ErrMissingName,
		},
		{
			name: "missing clusterRef",
			obs: Observability{
				APIVersion: "platformfoundry.io/v1",
				Kind:       "Observability",
				Metadata: Metadata{
					Name: "test",
				},
				Spec: ObservabilitySpec{},
			},
			wantErr: true,
			errType: ErrMissingClusterRef,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.obs.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Observability.Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.wantErr && err != tt.errType {
				t.Errorf("Observability.Validate() error = %v, want %v", err, tt.errType)
			}
		})
	}
}

func TestMonitoringConfig(t *testing.T) {
	tests := []struct {
		name          string
		config        *MonitoringConfig
		wantRetention string
		wantStorage   string
	}{
		{
			name: "short retention",
			config: &MonitoringConfig{
				Provider:    "prometheus",
				Retention:   "7d",
				StorageSize: "10Gi",
			},
			wantRetention: "7d",
			wantStorage:   "10Gi",
		},
		{
			name: "long retention",
			config: &MonitoringConfig{
				Provider:    "prometheus",
				Retention:   "90d",
				StorageSize: "200Gi",
			},
			wantRetention: "90d",
			wantStorage:   "200Gi",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.config.Retention != tt.wantRetention {
				t.Errorf("MonitoringConfig.Retention = %v, want %v", tt.config.Retention, tt.wantRetention)
			}
			if tt.config.StorageSize != tt.wantStorage {
				t.Errorf("MonitoringConfig.StorageSize = %v, want %v", tt.config.StorageSize, tt.wantStorage)
			}
		})
	}
}

func TestVisualizationConfig(t *testing.T) {
	tests := []struct {
		name           string
		config         *VisualizationConfig
		wantDashboards int
	}{
		{
			name: "single dashboard",
			config: &VisualizationConfig{
				Provider: "grafana",
				Dashboards: []string{
					"kubernetes-overview",
				},
			},
			wantDashboards: 1,
		},
		{
			name: "multiple dashboards",
			config: &VisualizationConfig{
				Provider: "grafana",
				Dashboards: []string{
					"kubernetes-overview",
					"application-metrics",
					"infrastructure-health",
				},
			},
			wantDashboards: 3,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if len(tt.config.Dashboards) != tt.wantDashboards {
				t.Errorf("VisualizationConfig.Dashboards count = %v, want %v", len(tt.config.Dashboards), tt.wantDashboards)
			}
		})
	}
}

func TestLoggingConfig(t *testing.T) {
	config := &LoggingConfig{
		Provider:    "loki",
		Retention:   "30d",
		StorageSize: "50Gi",
	}

	if config.Provider != "loki" {
		t.Errorf("LoggingConfig.Provider = %v, want loki", config.Provider)
	}
	if config.Retention != "30d" {
		t.Errorf("LoggingConfig.Retention = %v, want 30d", config.Retention)
	}
	if config.StorageSize != "50Gi" {
		t.Errorf("LoggingConfig.StorageSize = %v, want 50Gi", config.StorageSize)
	}
}

func TestIngressConfig(t *testing.T) {
	tests := []struct {
		name         string
		config       *IngressConfig
		wantProvider string
		wantDomains  int
	}{
		{
			name: "traefik with single domain",
			config: &IngressConfig{
				Provider: "traefik",
				Domains: []string{
					"metrics.example.com",
				},
			},
			wantProvider: "traefik",
			wantDomains:  1,
		},
		{
			name: "nginx with multiple domains",
			config: &IngressConfig{
				Provider: "nginx",
				Domains: []string{
					"grafana.example.com",
					"prometheus.example.com",
					"loki.example.com",
				},
			},
			wantProvider: "nginx",
			wantDomains:  3,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.config.Provider != tt.wantProvider {
				t.Errorf("IngressConfig.Provider = %v, want %v", tt.config.Provider, tt.wantProvider)
			}
			if len(tt.config.Domains) != tt.wantDomains {
				t.Errorf("IngressConfig.Domains count = %v, want %v", len(tt.config.Domains), tt.wantDomains)
			}
		})
	}
}

func TestObservabilityStatus(t *testing.T) {
	obs := Observability{
		Status: ObservabilityStatus{
			Phase:   PhaseReady,
			Message: "All monitoring components ready",
		},
	}

	if obs.Status.Phase != PhaseReady {
		t.Errorf("Expected phase %s, got %s", PhaseReady, obs.Status.Phase)
	}

	if obs.Status.Message == "" {
		t.Error("Expected non-empty status message")
	}
}
