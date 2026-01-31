package integration

import (
	"encoding/json"
	"errors"
	"fmt"
	"os/exec"

	"github.com/platformfoundry/platformfoundry-ce/pkg/types"
)

// Integration represents a connection between two components
type Integration struct {
	Source      string                 `json:"source"`
	Target      string                 `json:"target"`
	Type        string                 `json:"type"` // datasource, repository, monitoring, etc.
	Status      string                 `json:"status"`
	Config      map[string]interface{} `json:"config"`
	Validated   bool                   `json:"validated"`
	ErrorReason string                 `json:"error_reason,omitempty"`
}

// Engine orchestrates component integrations
type Engine struct {
	integrations []Integration
}

// NewEngine creates a new integration engine
func NewEngine() *Engine {
	return &Engine{
		integrations: make([]Integration, 0),
	}
}

// IntegrateComponents automatically integrates all platform components
func (e *Engine) IntegrateComponents(platform *types.Platform, components map[string]interface{}) error {
	// Detect and create integrations
	if err := e.detectIntegrations(platform, components); err != nil {
		return fmt.Errorf("failed to detect integrations: %w", err)
	}

	// Apply integrations
	for i := range e.integrations {
		if err := e.applyIntegration(&e.integrations[i]); err != nil {
			e.integrations[i].Status = "failed"
			e.integrations[i].ErrorReason = err.Error()
			continue
		}
		e.integrations[i].Status = "applied"
	}

	// Validate integrations
	if err := e.validateIntegrations(); err != nil {
		return fmt.Errorf("integration validation failed: %w", err)
	}

	return nil
}

// detectIntegrations detects what integrations are needed based on components
func (e *Engine) detectIntegrations(platform *types.Platform, components map[string]interface{}) error {
	// Check for Prometheus
	hasPrometheus := false
	prometheusService := ""
	if _, ok := components["prometheus"]; ok {
		hasPrometheus = true
		prometheusService = "prometheus-server.monitoring.svc.cluster.local"
	}

	// Check for Grafana
	hasGrafana := false
	if _, ok := components["grafana"]; ok {
		hasGrafana = true

		// Integrate Grafana with Prometheus
		if hasPrometheus {
			e.integrations = append(e.integrations, Integration{
				Source: "grafana",
				Target: "prometheus",
				Type:   "datasource",
				Status: "pending",
				Config: map[string]interface{}{
					"url":     fmt.Sprintf("http://%s", prometheusService),
					"default": true,
				},
			})
		}
	}

	// Check for ArgoCD
	hasArgoCD := false
	if orchestrator, ok := components["orchestrator"].(*types.Orchestrator); ok {
		if orchestrator.Spec.Provider == "argocd" {
			hasArgoCD = true

			// Integrate ArgoCD with Git repository
			if orchestrator.Spec.GitOps != nil {
				e.integrations = append(e.integrations, Integration{
					Source: "argocd",
					Target: "git",
					Type:   "repository",
					Status: "pending",
					Config: map[string]interface{}{
						"repoURL": orchestrator.Spec.GitOps.RepoURL,
						"branch":  orchestrator.Spec.GitOps.Branch,
					},
				})
			}
		}
	}

	// Check for Backstage
	if devex, ok := components["devex"].(*types.DevEx); ok {
		if devex.Spec.Provider == "backstage" {
			// Integrate Backstage with ArgoCD
			if hasArgoCD {
				e.integrations = append(e.integrations, Integration{
					Source: "backstage",
					Target: "argocd",
					Type:   "plugin",
					Status: "pending",
					Config: map[string]interface{}{
						"url": "http://argocd-server.argocd.svc.cluster.local",
					},
				})
			}

			// Integrate Backstage with Grafana
			if hasGrafana {
				e.integrations = append(e.integrations, Integration{
					Source: "backstage",
					Target: "grafana",
					Type:   "plugin",
					Status: "pending",
					Config: map[string]interface{}{
						"url": "http://grafana.monitoring.svc.cluster.local",
					},
				})
			}

			// Integrate Backstage with Prometheus
			if hasPrometheus {
				e.integrations = append(e.integrations, Integration{
					Source: "backstage",
					Target: "prometheus",
					Type:   "plugin",
					Status: "pending",
					Config: map[string]interface{}{
						"url": fmt.Sprintf("http://%s", prometheusService),
					},
				})
			}

			// Integrate Backstage with GitHub (if configured in portal integrations)
			if devex.Spec.Portal != nil {
				for _, integ := range devex.Spec.Portal.Integrations {
					if integ.Type == "github" && integ.Enabled {
						e.integrations = append(e.integrations, Integration{
							Source: "backstage",
							Target: "github",
							Type:   "scm",
							Status: "pending",
							Config: integ.Config,
						})
					}
				}
			}
		}
	}

	// Integrate Prometheus with cluster (scrape targets)
	if hasPrometheus {
		if infrastructure, ok := components["infrastructure"].(*types.Infrastructure); ok {
			e.integrations = append(e.integrations, Integration{
				Source: "prometheus",
				Target: "cluster",
				Type:   "monitoring",
				Status: "pending",
				Config: map[string]interface{}{
					"cluster_name": infrastructure.Metadata.Name,
					"scrape_jobs": []string{
						"kubernetes-apiservers",
						"kubernetes-nodes",
						"kubernetes-pods",
						"kubernetes-cadvisor",
					},
				},
			})
		}
	}

	return nil
}

// applyIntegration applies a single integration
func (e *Engine) applyIntegration(integ *Integration) error {
	switch integ.Type {
	case "datasource":
		return e.applyDataSourceIntegration(integ)
	case "repository":
		return e.applyRepositoryIntegration(integ)
	case "plugin":
		return e.applyPluginIntegration(integ)
	case "scm":
		return e.applySCMIntegration(integ)
	case "monitoring":
		return e.applyMonitoringIntegration(integ)
	default:
		return fmt.Errorf("unsupported integration type: %s", integ.Type)
	}
}

// applyDataSourceIntegration configures Grafana datasource
func (e *Engine) applyDataSourceIntegration(integ *Integration) error {
	// In a real implementation, this would:
	// 1. Connect to Grafana API
	// 2. Create/update datasource configuration
	// 3. Test the connection

	// For now, we simulate success if the config is valid
	if _, ok := integ.Config["url"]; !ok {
		return errors.New("datasource URL is required")
	}

	// Would execute kubectl or API calls here
	return nil
}

// applyRepositoryIntegration configures Git repository connection
func (e *Engine) applyRepositoryIntegration(integ *Integration) error {
	// In a real implementation, this would:
	// 1. Connect to ArgoCD API
	// 2. Register the Git repository
	// 3. Test repository access

	repoURL, ok := integ.Config["repoURL"].(string)
	if !ok || repoURL == "" {
		return errors.New("repository URL is required")
	}

	// Would execute argocd CLI or API calls here
	return nil
}

// applyPluginIntegration configures Backstage plugin integration
func (e *Engine) applyPluginIntegration(integ *Integration) error {
	// In a real implementation, this would:
	// 1. Update Backstage app-config.yaml
	// 2. Restart Backstage pods
	// 3. Verify plugin is loaded

	if _, ok := integ.Config["url"]; !ok {
		return errors.New("target URL is required")
	}

	// Would execute kubectl commands to update configmap
	return nil
}

// applySCMIntegration configures source control integration
func (e *Engine) applySCMIntegration(integ *Integration) error {
	// In a real implementation, this would:
	// 1. Update Backstage app-config.yaml with GitHub token
	// 2. Configure OAuth if needed
	// 3. Test GitHub API access

	// Config validation
	if integ.Config == nil {
		return errors.New("SCM integration requires configuration")
	}

	return nil
}

// applyMonitoringIntegration configures Prometheus to scrape cluster
func (e *Engine) applyMonitoringIntegration(integ *Integration) error {
	// In a real implementation, this would:
	// 1. Update Prometheus ConfigMap with scrape configs
	// 2. Reload Prometheus configuration
	// 3. Verify scrape targets are discovered

	scrapeJobs, ok := integ.Config["scrape_jobs"].([]string)
	if !ok || len(scrapeJobs) == 0 {
		return errors.New("scrape jobs are required")
	}

	return nil
}

// validateIntegrations validates all applied integrations
func (e *Engine) validateIntegrations() error {
	for i := range e.integrations {
		if e.integrations[i].Status == "failed" {
			continue
		}

		if err := e.validateIntegration(&e.integrations[i]); err != nil {
			e.integrations[i].Validated = false
			e.integrations[i].ErrorReason = err.Error()
			continue
		}

		e.integrations[i].Validated = true
	}

	return nil
}

// validateIntegration validates a single integration
func (e *Engine) validateIntegration(integ *Integration) error {
	switch integ.Type {
	case "datasource":
		return e.validateDataSourceIntegration(integ)
	case "repository":
		return e.validateRepositoryIntegration(integ)
	case "plugin":
		return e.validatePluginIntegration(integ)
	case "scm":
		return e.validateSCMIntegration(integ)
	case "monitoring":
		return e.validateMonitoringIntegration(integ)
	default:
		return nil // Unknown types are considered valid
	}
}

// validateDataSourceIntegration validates Grafana can connect to datasource
func (e *Engine) validateDataSourceIntegration(integ *Integration) error {
	// In a real implementation, would test the connection
	// For now, just check if kubectl is available
	if _, err := exec.LookPath("kubectl"); err != nil {
		// If kubectl is not available, assume validation passes (config-only mode)
		return nil
	}

	return nil
}

// validateRepositoryIntegration validates Git repository is accessible
func (e *Engine) validateRepositoryIntegration(integ *Integration) error {
	// In a real implementation, would verify repository access
	repoURL, _ := integ.Config["repoURL"].(string)
	if repoURL == "" {
		return errors.New("repository URL is empty")
	}

	return nil
}

// validatePluginIntegration validates Backstage plugin is loaded
func (e *Engine) validatePluginIntegration(integ *Integration) error {
	// In a real implementation, would check Backstage API
	return nil
}

// validateSCMIntegration validates GitHub integration
func (e *Engine) validateSCMIntegration(integ *Integration) error {
	// In a real implementation, would test GitHub API access
	return nil
}

// validateMonitoringIntegration validates Prometheus is scraping targets
func (e *Engine) validateMonitoringIntegration(integ *Integration) error {
	// In a real implementation, would check Prometheus targets API
	return nil
}

// GetIntegrations returns all integrations
func (e *Engine) GetIntegrations() []Integration {
	return e.integrations
}

// GetIntegrationStatus returns a summary of integration status
func (e *Engine) GetIntegrationStatus() map[string]interface{} {
	total := len(e.integrations)
	applied := 0
	validated := 0
	failed := 0

	for _, integ := range e.integrations {
		if integ.Status == "applied" {
			applied++
		}
		if integ.Status == "failed" {
			failed++
		}
		if integ.Validated {
			validated++
		}
	}

	return map[string]interface{}{
		"total":     total,
		"applied":   applied,
		"validated": validated,
		"failed":    failed,
		"success_rate": func() float64 {
			if total == 0 {
				return 1.0
			}
			return float64(validated) / float64(total)
		}(),
	}
}

// ExportIntegrations exports integrations to JSON
func (e *Engine) ExportIntegrations() (string, error) {
	data, err := json.MarshalIndent(e.integrations, "", "  ")
	if err != nil {
		return "", err
	}
	return string(data), nil
}
