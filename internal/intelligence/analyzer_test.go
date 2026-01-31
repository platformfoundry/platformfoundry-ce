package intelligence

import (
	"testing"

	"github.com/platformfoundry/platformfoundry-ce/pkg/types"
)

func TestAnalyzer_AnalyzePlatform(t *testing.T) {
	analyzer := NewAnalyzer()

	tests := []struct {
		name     string
		platform *types.Platform
		want     *TechStack
	}{
		{
			name: "AWS platform with full stack",
			platform: &types.Platform{
				Metadata: types.Metadata{
					Name: "my-platform",
				},
				Spec: types.PlatformSpec{
					Components: types.ComponentReferences{
						Infrastructure: "aws-infra",
						Orchestrator:   "argocd",
						Observability:  "monitoring",
						Pipeline:       "jenkins",
					},
					Global: types.GlobalConfig{
						Region: "us-east-1",
						Tags: map[string]string{
							"infrastructure-provider": "terraform",
						},
					},
				},
			},
			want: &TechStack{
				InfrastructureProvider: "terraform",
				CloudProvider:          "aws",
				Orchestrator:           "argocd",
				ObservabilityTools:     []string{"prometheus", "grafana"},
				PipelineTools:          []string{"jenkins"},
				HasMonitoring:          true,
				HasLogging:             true,
			},
		},
		{
			name: "GCP platform",
			platform: &types.Platform{
				Metadata: types.Metadata{
					Name: "gcp-platform",
				},
				Spec: types.PlatformSpec{
					Components: types.ComponentReferences{
						Infrastructure: "gcp-infra",
					},
					Global: types.GlobalConfig{
						Region: "us-central1",
					},
				},
			},
			want: &TechStack{
				InfrastructureProvider: "terraform",
				CloudProvider:          "gcp",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := analyzer.AnalyzePlatform(tt.platform)

			if got.InfrastructureProvider != tt.want.InfrastructureProvider {
				t.Errorf("InfrastructureProvider = %v, want %v", got.InfrastructureProvider, tt.want.InfrastructureProvider)
			}
			if got.CloudProvider != tt.want.CloudProvider {
				t.Errorf("CloudProvider = %v, want %v", got.CloudProvider, tt.want.CloudProvider)
			}
			if got.Orchestrator != tt.want.Orchestrator {
				t.Errorf("Orchestrator = %v, want %v", got.Orchestrator, tt.want.Orchestrator)
			}
		})
	}
}

func TestAnalyzer_AnalyzeInfrastructure(t *testing.T) {
	analyzer := NewAnalyzer()

	tests := []struct {
		name  string
		infra *types.Infrastructure
		want  *TechStack
	}{
		{
			name: "AWS infrastructure",
			infra: &types.Infrastructure{
				Metadata: types.Metadata{
					Name: "aws-infra",
				},
				Spec: types.InfrastructureSpec{
					Provider: "terraform",
					Cloud: types.CloudConfig{
						Provider: "aws",
						VPC: &types.VPCConfig{
							CIDR: "10.0.0.0/16",
						},
					},
				},
			},
			want: &TechStack{
				InfrastructureProvider: "terraform",
				CloudProvider:          "aws",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := analyzer.AnalyzeInfrastructure(tt.infra)

			if got.InfrastructureProvider != tt.want.InfrastructureProvider {
				t.Errorf("InfrastructureProvider = %v, want %v", got.InfrastructureProvider, tt.want.InfrastructureProvider)
			}
			if got.CloudProvider != tt.want.CloudProvider {
				t.Errorf("CloudProvider = %v, want %v", got.CloudProvider, tt.want.CloudProvider)
			}
		})
	}
}

func TestAnalyzer_AnalyzeObservability(t *testing.T) {
	analyzer := NewAnalyzer()

	tests := []struct {
		name string
		obs  *types.Observability
		want *TechStack
	}{
		{
			name: "Full observability stack",
			obs: &types.Observability{
				Metadata: types.Metadata{
					Name: "monitoring",
				},
				Spec: types.ObservabilitySpec{
					ClusterRef: "my-cluster",
					Monitoring: &types.MonitoringConfig{
						Provider:  "prometheus",
						Retention: "30d",
					},
					Visualization: &types.VisualizationConfig{
						Provider: "grafana",
					},
					Logging: &types.LoggingConfig{
						Provider:  "loki",
						Retention: "7d",
					},
					Ingress: &types.IngressConfig{
						Provider: "traefik",
					},
				},
			},
			want: &TechStack{
				ObservabilityTools: []string{"prometheus", "grafana", "loki"},
				HasMonitoring:      true,
				HasLogging:         true,
				HasIngress:         true,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := analyzer.AnalyzeObservability(tt.obs)

			if got.HasMonitoring != tt.want.HasMonitoring {
				t.Errorf("HasMonitoring = %v, want %v", got.HasMonitoring, tt.want.HasMonitoring)
			}
			if got.HasLogging != tt.want.HasLogging {
				t.Errorf("HasLogging = %v, want %v", got.HasLogging, tt.want.HasLogging)
			}
			if got.HasIngress != tt.want.HasIngress {
				t.Errorf("HasIngress = %v, want %v", got.HasIngress, tt.want.HasIngress)
			}
		})
	}
}

func TestDetectCloudFromRegion(t *testing.T) {
	tests := []struct {
		region string
		want   string
	}{
		{"us-east-1", "aws"},
		{"eu-west-1", "aws"},
		{"ap-south-1", "aws"},
		{"us-central1", "gcp"},
		{"europe-west1", "gcp"},
		{"eastus", "azure"},
		{"westus", "azure"},
		{"unknown-region", "unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.region, func(t *testing.T) {
			got := detectCloudFromRegion(tt.region)
			if got != tt.want {
				t.Errorf("detectCloudFromRegion(%s) = %v, want %v", tt.region, got, tt.want)
			}
		})
	}
}
