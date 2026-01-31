package intelligence

import (
	"github.com/platformfoundry/platformfoundry-ce/pkg/types"
)

// TechStack represents the analyzed technology stack of a platform
type TechStack struct {
	InfrastructureProvider string   // terraform, crossplane, pulumi
	CloudProvider          string   // aws, gcp, azure, multi-cloud
	Orchestrator           string   // argocd, flux, tekton
	ObservabilityTools     []string // prometheus, grafana, loki, tempo, etc.
	PipelineTools          []string // jenkins, gitlab-ci, github-actions
	MeshTools              []string // istio, linkerd, consul
	SecurityTools          []string // vault, trivy, falco
	ComplianceFrameworks   []string // soc2, pci-dss, hipaa
	HasMonitoring          bool
	HasLogging             bool
	HasTracing             bool
	HasIngress             bool
}

// Analyzer analyzes platform configuration to extract tech stack information
type Analyzer struct{}

// NewAnalyzer creates a new tech stack analyzer
func NewAnalyzer() *Analyzer {
	return &Analyzer{}
}

// AnalyzePlatform analyzes a platform resource and returns the tech stack
func (a *Analyzer) AnalyzePlatform(platform *types.Platform) *TechStack {
	ts := &TechStack{
		ObservabilityTools:   make([]string, 0),
		PipelineTools:        make([]string, 0),
		MeshTools:            make([]string, 0),
		SecurityTools:        make([]string, 0),
		ComplianceFrameworks: make([]string, 0),
	}

	// Extract infrastructure provider from platform metadata or tags
	if provider, ok := platform.Spec.Global.Tags["infrastructure-provider"]; ok {
		ts.InfrastructureProvider = provider
	} else {
		ts.InfrastructureProvider = "terraform" // default
	}

	// Extract cloud provider from global config
	if platform.Spec.Global.Region != "" {
		ts.CloudProvider = detectCloudFromRegion(platform.Spec.Global.Region)
	} else if provider, ok := platform.Spec.Global.Tags["cloud-provider"]; ok {
		ts.CloudProvider = provider
	}

	// Note: In a real implementation, we would load and analyze the referenced
	// component resources. For now, we extract from component references

	// Analyze orchestrator
	if platform.Spec.Components.Orchestrator != "" {
		// In real implementation, we'd load the orchestrator resource
		ts.Orchestrator = "argocd" // default assumption
	}

	// Analyze observability
	if platform.Spec.Components.Observability != "" {
		// In real implementation, we'd load the observability resource
		ts.ObservabilityTools = []string{"prometheus", "grafana"}
		ts.HasMonitoring = true
		ts.HasLogging = true
	}

	// Analyze pipeline
	if platform.Spec.Components.Pipeline != "" {
		ts.PipelineTools = []string{"jenkins"} // default assumption
	}

	// Analyze mesh
	if platform.Spec.Components.Mesh != "" {
		ts.MeshTools = []string{"istio"} // default assumption
	}

	// Analyze security
	if platform.Spec.Components.Security != "" {
		ts.SecurityTools = []string{"vault"} // default assumption
	}

	// Analyze compliance
	if platform.Spec.Components.Compliance != "" {
		ts.ComplianceFrameworks = []string{"soc2"} // default assumption
	}

	return ts
}

// AnalyzeInfrastructure analyzes infrastructure resource
func (a *Analyzer) AnalyzeInfrastructure(infra *types.Infrastructure) *TechStack {
	ts := &TechStack{
		ObservabilityTools:   make([]string, 0),
		PipelineTools:        make([]string, 0),
		MeshTools:            make([]string, 0),
		SecurityTools:        make([]string, 0),
		ComplianceFrameworks: make([]string, 0),
	}

	// Detect infrastructure provider from provider field
	ts.InfrastructureProvider = infra.Spec.Provider

	// Detect cloud provider from configuration
	if infra.Spec.Cloud.Provider != "" {
		ts.CloudProvider = infra.Spec.Cloud.Provider
	}

	return ts
}

// AnalyzeObservability analyzes observability resource
func (a *Analyzer) AnalyzeObservability(obs *types.Observability) *TechStack {
	ts := &TechStack{
		ObservabilityTools:   make([]string, 0),
		PipelineTools:        make([]string, 0),
		MeshTools:            make([]string, 0),
		SecurityTools:        make([]string, 0),
		ComplianceFrameworks: make([]string, 0),
	}

	// Detect monitoring tools
	if obs.Spec.Monitoring != nil {
		if obs.Spec.Monitoring.Provider == "prometheus" {
			ts.ObservabilityTools = append(ts.ObservabilityTools, "prometheus")
			ts.HasMonitoring = true
		}
	}

	// Detect visualization tools
	if obs.Spec.Visualization != nil {
		if obs.Spec.Visualization.Provider == "grafana" {
			ts.ObservabilityTools = append(ts.ObservabilityTools, "grafana")
		}
	}

	// Detect logging tools
	if obs.Spec.Logging != nil {
		if obs.Spec.Logging.Provider == "loki" {
			ts.ObservabilityTools = append(ts.ObservabilityTools, "loki")
			ts.HasLogging = true
		}
	}

	// Detect ingress
	if obs.Spec.Ingress != nil {
		ts.HasIngress = true
	}

	return ts
}

// AnalyzeOrchestrator analyzes orchestrator resource
func (a *Analyzer) AnalyzeOrchestrator(orch *types.Orchestrator) *TechStack {
	ts := &TechStack{
		ObservabilityTools:   make([]string, 0),
		PipelineTools:        make([]string, 0),
		MeshTools:            make([]string, 0),
		SecurityTools:        make([]string, 0),
		ComplianceFrameworks: make([]string, 0),
	}

	ts.Orchestrator = orch.Spec.Provider

	return ts
}

// detectCloudFromRegion detects cloud provider from region string
func detectCloudFromRegion(region string) string {
	// Azure regions (no dashes, all lowercase)
	if region == "eastus" || region == "westus" || region == "centralus" ||
	   region == "northeurope" || region == "westeurope" {
		return "azure"
	}

	// AWS regions have format: region-direction-number (e.g., us-east-1)
	// Count dashes - AWS has exactly 2 dashes
	dashCount := 0
	for _, c := range region {
		if c == '-' {
			dashCount++
		}
	}

	// AWS: us-east-1, eu-west-1, ap-south-1 (2 dashes)
	if dashCount == 2 && len(region) >= 9 {
		prefix := region[:3]
		switch prefix {
		case "us-", "eu-", "ap-", "sa-", "ca-", "me-", "af-":
			return "aws"
		}
	}

	// GCP: us-central1, europe-west1, asia-east1 (1 dash)
	if dashCount == 1 && len(region) > 7 {
		if (region[:3] == "us-") ||
		   (len(region) > 11 && region[:7] == "europe-") ||
		   (len(region) > 9 && region[:5] == "asia-") {
			return "gcp"
		}
	}

	return "unknown"
}
