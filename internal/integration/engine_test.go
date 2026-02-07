package integration

import (
	"testing"

	"github.com/platformfoundry/pf-ce/pkg/types"
)

func TestNewEngine(t *testing.T) {
	e := NewEngine()
	if e == nil {
		t.Fatal("NewEngine() returned nil")
	}

	if e.integrations == nil {
		t.Error("integrations should be initialized")
	}
}

func TestDetectIntegrationsGrafanaPrometheus(t *testing.T) {
	e := NewEngine()

	platform := &types.Platform{
		Metadata: types.Metadata{Name: "test-platform"},
	}

	components := map[string]interface{}{
		"prometheus": true,
		"grafana":    true,
	}

	err := e.detectIntegrations(platform, components)
	if err != nil {
		t.Fatalf("detectIntegrations() error = %v", err)
	}

	// Should create Grafana -> Prometheus datasource integration
	found := false
	for _, integ := range e.integrations {
		if integ.Source == "grafana" && integ.Target == "prometheus" && integ.Type == "datasource" {
			found = true
			break
		}
	}

	if !found {
		t.Error("Should detect Grafana -> Prometheus datasource integration")
	}
}

func TestDetectIntegrationsArgoCD(t *testing.T) {
	e := NewEngine()

	platform := &types.Platform{
		Metadata: types.Metadata{Name: "test-platform"},
	}

	orchestrator := &types.Orchestrator{
		Metadata: types.Metadata{Name: "test-argocd"},
		Spec: types.OrchestratorSpec{
			Provider: "argocd",
			GitOps: &types.GitOpsConfig{
				RepoURL: "https://github.com/example/repo",
				Branch:  "main",
			},
		},
	}

	components := map[string]interface{}{
		"orchestrator": orchestrator,
	}

	err := e.detectIntegrations(platform, components)
	if err != nil {
		t.Fatalf("detectIntegrations() error = %v", err)
	}

	// Should create ArgoCD -> Git repository integration
	found := false
	for _, integ := range e.integrations {
		if integ.Source == "argocd" && integ.Target == "git" && integ.Type == "repository" {
			found = true
			if repoURL, ok := integ.Config["repoURL"].(string); !ok || repoURL != "https://github.com/example/repo" {
				t.Error("Integration should have correct repoURL")
			}
			break
		}
	}

	if !found {
		t.Error("Should detect ArgoCD -> Git repository integration")
	}
}

func TestDetectIntegrationsBackstage(t *testing.T) {
	e := NewEngine()

	platform := &types.Platform{
		Metadata: types.Metadata{Name: "test-platform"},
	}

	devex := &types.DevEx{
		Metadata: types.Metadata{Name: "test-backstage"},
		Spec: types.DevExSpec{
			Provider: "backstage",
			Portal: &types.PortalConfig{
				Integrations: []types.Integration{
					{Type: "github", Enabled: true, Config: map[string]interface{}{"token": "fake"}},
				},
			},
		},
	}

	components := map[string]interface{}{
		"devex":      devex,
		"prometheus": true,
		"grafana":    true,
	}

	orchestrator := &types.Orchestrator{
		Spec: types.OrchestratorSpec{Provider: "argocd"},
	}
	components["orchestrator"] = orchestrator

	err := e.detectIntegrations(platform, components)
	if err != nil {
		t.Fatalf("detectIntegrations() error = %v", err)
	}

	// Should create multiple Backstage integrations
	backstageIntegrations := 0
	for _, integ := range e.integrations {
		if integ.Source == "backstage" {
			backstageIntegrations++
		}
	}

	if backstageIntegrations < 3 {
		t.Errorf("Should detect at least 3 Backstage integrations, got %d", backstageIntegrations)
	}
}

func TestDetectIntegrationsPrometheusCluster(t *testing.T) {
	e := NewEngine()

	platform := &types.Platform{
		Metadata: types.Metadata{Name: "test-platform"},
	}

	infrastructure := &types.Infrastructure{
		Metadata: types.Metadata{Name: "test-cluster"},
	}

	components := map[string]interface{}{
		"prometheus":     true,
		"infrastructure": infrastructure,
	}

	err := e.detectIntegrations(platform, components)
	if err != nil {
		t.Fatalf("detectIntegrations() error = %v", err)
	}

	// Should create Prometheus -> Cluster monitoring integration
	found := false
	for _, integ := range e.integrations {
		if integ.Source == "prometheus" && integ.Target == "cluster" && integ.Type == "monitoring" {
			found = true
			break
		}
	}

	if !found {
		t.Error("Should detect Prometheus -> Cluster monitoring integration")
	}
}

func TestApplyDataSourceIntegration(t *testing.T) {
	e := NewEngine()

	integ := &Integration{
		Source: "grafana",
		Target: "prometheus",
		Type:   "datasource",
		Config: map[string]interface{}{
			"url": "http://prometheus:9090",
		},
	}

	err := e.applyDataSourceIntegration(integ)
	if err != nil {
		t.Errorf("applyDataSourceIntegration() error = %v", err)
	}
}

func TestApplyDataSourceIntegrationMissingURL(t *testing.T) {
	e := NewEngine()

	integ := &Integration{
		Source: "grafana",
		Target: "prometheus",
		Type:   "datasource",
		Config: map[string]interface{}{},
	}

	err := e.applyDataSourceIntegration(integ)
	if err == nil {
		t.Error("applyDataSourceIntegration() should fail without URL")
	}
}

func TestApplyRepositoryIntegration(t *testing.T) {
	e := NewEngine()

	integ := &Integration{
		Source: "argocd",
		Target: "git",
		Type:   "repository",
		Config: map[string]interface{}{
			"repoURL": "https://github.com/example/repo",
			"branch":  "main",
		},
	}

	err := e.applyRepositoryIntegration(integ)
	if err != nil {
		t.Errorf("applyRepositoryIntegration() error = %v", err)
	}
}

func TestApplyRepositoryIntegrationMissingURL(t *testing.T) {
	e := NewEngine()

	integ := &Integration{
		Source: "argocd",
		Target: "git",
		Type:   "repository",
		Config: map[string]interface{}{},
	}

	err := e.applyRepositoryIntegration(integ)
	if err == nil {
		t.Error("applyRepositoryIntegration() should fail without repoURL")
	}
}

func TestApplyMonitoringIntegration(t *testing.T) {
	e := NewEngine()

	integ := &Integration{
		Source: "prometheus",
		Target: "cluster",
		Type:   "monitoring",
		Config: map[string]interface{}{
			"scrape_jobs": []string{"kubernetes-nodes", "kubernetes-pods"},
		},
	}

	err := e.applyMonitoringIntegration(integ)
	if err != nil {
		t.Errorf("applyMonitoringIntegration() error = %v", err)
	}
}

func TestGetIntegrationStatus(t *testing.T) {
	e := NewEngine()

	e.integrations = []Integration{
		{Source: "grafana", Target: "prometheus", Status: "applied", Validated: true},
		{Source: "argocd", Target: "git", Status: "applied", Validated: true},
		{Source: "backstage", Target: "argocd", Status: "applied", Validated: false},
		{Source: "backstage", Target: "github", Status: "failed", Validated: false},
	}

	status := e.GetIntegrationStatus()

	if total, ok := status["total"].(int); !ok || total != 4 {
		t.Errorf("Expected total 4, got %v", status["total"])
	}

	if applied, ok := status["applied"].(int); !ok || applied != 3 {
		t.Errorf("Expected applied 3, got %v", status["applied"])
	}

	if validated, ok := status["validated"].(int); !ok || validated != 2 {
		t.Errorf("Expected validated 2, got %v", status["validated"])
	}

	if failed, ok := status["failed"].(int); !ok || failed != 1 {
		t.Errorf("Expected failed 1, got %v", status["failed"])
	}

	if rate, ok := status["success_rate"].(float64); !ok || rate != 0.5 {
		t.Errorf("Expected success_rate 0.5, got %v", status["success_rate"])
	}
}

func TestGetIntegrationStatusEmpty(t *testing.T) {
	e := NewEngine()

	status := e.GetIntegrationStatus()

	if total, ok := status["total"].(int); !ok || total != 0 {
		t.Errorf("Expected total 0, got %v", status["total"])
	}

	if rate, ok := status["success_rate"].(float64); !ok || rate != 1.0 {
		t.Errorf("Expected success_rate 1.0 for empty list, got %v", status["success_rate"])
	}
}

func TestExportIntegrations(t *testing.T) {
	e := NewEngine()

	e.integrations = []Integration{
		{Source: "grafana", Target: "prometheus", Type: "datasource", Status: "applied"},
	}

	json, err := e.ExportIntegrations()
	if err != nil {
		t.Fatalf("ExportIntegrations() error = %v", err)
	}

	if json == "" {
		t.Error("ExportIntegrations() should return non-empty JSON")
	}

	if !contains(json, "grafana") {
		t.Error("JSON should contain 'grafana'")
	}

	if !contains(json, "prometheus") {
		t.Error("JSON should contain 'prometheus'")
	}
}

func TestIntegrateComponents(t *testing.T) {
	e := NewEngine()

	platform := &types.Platform{
		Metadata: types.Metadata{Name: "test-platform"},
	}

	components := map[string]interface{}{
		"prometheus": true,
		"grafana":    true,
	}

	err := e.IntegrateComponents(platform, components)
	if err != nil {
		t.Fatalf("IntegrateComponents() error = %v", err)
	}

	integrations := e.GetIntegrations()
	if len(integrations) == 0 {
		t.Error("IntegrateComponents() should create integrations")
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) && (s[:len(substr)] == substr || s[len(s)-len(substr):] == substr || containsMiddle(s, substr)))
}

func containsMiddle(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
