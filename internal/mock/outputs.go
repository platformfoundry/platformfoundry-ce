package mock

import "fmt"

// generateOutputs creates realistic mock outputs based on tool type
func generateOutputs(m *MockPlugin, spec map[string]interface{}) map[string]string {
	outputs := make(map[string]string)

	// Get name from spec
	name := "platform"
	if n, ok := spec["name"].(string); ok {
		name = n
	}

	// Generate outputs based on tool type
	switch m.toolType {
	case "Infrastructure", "infrastructure":
		generateInfrastructureOutputs(m, outputs, name, spec)
	case "Orchestrator", "orchestrator":
		generateOrchestratorOutputs(m, outputs, name, spec)
	case "Observability", "observability":
		generateObservabilityOutputs(m, outputs, name, spec)
	case "DevEx", "devex":
		generateDevExOutputs(m, outputs, name, spec)
	case "Security", "security":
		generateSecurityOutputs(m, outputs, name, spec)
	case "Pipeline", "pipeline":
		generatePipelineOutputs(m, outputs, name, spec)
	default:
		outputs["status"] = "mock-success"
		outputs["id"] = fmt.Sprintf("mock-%s", name)
	}

	// Apply any overrides
	if override, ok := m.config.ResponseOverride[m.tool]; ok {
		if overrideMap, ok := override.(map[string]string); ok {
			for k, v := range overrideMap {
				outputs[k] = v
			}
		}
	}

	return outputs
}

func generateInfrastructureOutputs(m *MockPlugin, outputs map[string]string, name string, spec map[string]interface{}) {
	provider := "aws"
	if p, ok := spec["provider"].(string); ok {
		provider = p
	}

	switch provider {
	case "aws":
		outputs["cluster_endpoint"] = fmt.Sprintf("https://%s.us-west-2.eks.amazonaws.com", name)
		outputs["cluster_name"] = fmt.Sprintf("%s-eks", name)
		outputs["cluster_arn"] = fmt.Sprintf("arn:aws:eks:us-west-2:123456789:cluster/%s", name)
		outputs["vpc_id"] = fmt.Sprintf("vpc-%s-12345", name)
		outputs["subnet_ids"] = "subnet-mock-1,subnet-mock-2,subnet-mock-3"
		outputs["security_group_id"] = fmt.Sprintf("sg-%s-67890", name)
		outputs["oidc_issuer"] = fmt.Sprintf("https://oidc.eks.us-west-2.amazonaws.com/id/%s", name)
		outputs["endpoint"] = outputs["cluster_endpoint"]
		outputs["resource_id"] = outputs["cluster_arn"]

	case "gcp":
		outputs["cluster_endpoint"] = fmt.Sprintf("https://%s.us-central1.gke.io", name)
		outputs["cluster_name"] = fmt.Sprintf("%s-gke", name)
		outputs["network_name"] = fmt.Sprintf("%s-network", name)
		outputs["project_id"] = fmt.Sprintf("%s-project", name)
		outputs["endpoint"] = outputs["cluster_endpoint"]
		outputs["resource_id"] = outputs["cluster_name"]

	case "azure":
		outputs["cluster_endpoint"] = fmt.Sprintf("https://%s.eastus.azmk8s.io", name)
		outputs["cluster_name"] = fmt.Sprintf("%s-aks", name)
		outputs["resource_group"] = fmt.Sprintf("%s-rg", name)
		outputs["endpoint"] = outputs["cluster_endpoint"]
		outputs["resource_id"] = outputs["cluster_name"]
	}
}

func generateOrchestratorOutputs(m *MockPlugin, outputs map[string]string, name string, spec map[string]interface{}) {
	switch m.tool {
	case "argocd":
		outputs["argocd_url"] = fmt.Sprintf("https://argocd.%s.local", name)
		outputs["argocd_admin_password"] = "mock-admin-password"
		outputs["argocd_namespace"] = "argocd"
		outputs["argocd_version"] = "2.9.3"

	case "flux":
		outputs["flux_namespace"] = "flux-system"
		outputs["flux_version"] = "2.2.0"

	default:
		outputs["orchestrator_url"] = fmt.Sprintf("https://orchestrator.%s.local", name)
		outputs["orchestrator_namespace"] = m.tool
	}
}

func generateObservabilityOutputs(m *MockPlugin, outputs map[string]string, name string, spec map[string]interface{}) {
	switch m.tool {
	case "prometheus", "prometheus-stack":
		outputs["prometheus_url"] = fmt.Sprintf("https://prometheus.%s.local", name)
		outputs["prometheus_namespace"] = "monitoring"
		outputs["grafana_url"] = fmt.Sprintf("https://grafana.%s.local", name)
		outputs["grafana_admin_password"] = "mock-grafana-password"
		outputs["alertmanager_url"] = fmt.Sprintf("https://alertmanager.%s.local", name)
		outputs["dashboard_url"] = outputs["grafana_url"]
		outputs["metrics_endpoint"] = outputs["prometheus_url"]

	case "datadog":
		outputs["datadog_namespace"] = "datadog"
		outputs["dashboard_url"] = fmt.Sprintf("https://app.datadoghq.com/dashboard/%s", name)
		outputs["metrics_endpoint"] = "https://api.datadoghq.com"

	default:
		outputs["observability_namespace"] = "monitoring"
		outputs["dashboard_url"] = fmt.Sprintf("https://dashboard.%s.local", name)
		outputs["metrics_endpoint"] = fmt.Sprintf("https://metrics.%s.local", name)
	}
}

func generateDevExOutputs(m *MockPlugin, outputs map[string]string, name string, spec map[string]interface{}) {
	switch m.tool {
	case "backstage":
		outputs["backstage_url"] = fmt.Sprintf("https://backstage.%s.local", name)
		outputs["backstage_namespace"] = "backstage"
		outputs["catalog_url"] = fmt.Sprintf("https://backstage.%s.local/catalog", name)
		outputs["url"] = outputs["backstage_url"]
		outputs["api_key"] = "mock-backstage-api-key"

	default:
		outputs["devex_url"] = fmt.Sprintf("https://devex.%s.local", name)
		outputs["devex_namespace"] = m.tool
		outputs["url"] = outputs["devex_url"]
		outputs["api_key"] = "mock-api-key-12345"
	}
}

func generateSecurityOutputs(m *MockPlugin, outputs map[string]string, name string, spec map[string]interface{}) {
	switch m.tool {
	case "vault":
		outputs["vault_url"] = fmt.Sprintf("https://vault.%s.local", name)
		outputs["vault_root_token"] = "mock-root-token"
		outputs["vault_namespace"] = "vault"

	case "external-secrets":
		outputs["external_secrets_namespace"] = "external-secrets"

	default:
		outputs["security_namespace"] = m.tool
	}
}

func generatePipelineOutputs(m *MockPlugin, outputs map[string]string, name string, spec map[string]interface{}) {
	switch m.tool {
	case "jenkins":
		outputs["jenkins_url"] = fmt.Sprintf("https://jenkins.%s.local", name)
		outputs["jenkins_namespace"] = "jenkins"

	case "tekton":
		outputs["tekton_namespace"] = "tekton-pipelines"

	default:
		outputs["pipeline_namespace"] = m.tool
	}
}
