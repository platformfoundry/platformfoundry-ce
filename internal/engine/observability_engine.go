package engine

import (
	"context"
	"fmt"
	"time"
)

// ObservabilityEngine handles observability stack setup
type ObservabilityEngine struct {
	*BaseEngine

	provider string
	plugin   Plugin
}

// NewObservabilityEngine creates a new observability engine
func NewObservabilityEngine(provider string) *ObservabilityEngine {
	base := NewBaseEngine("observability", "Observability")
	// Observability depends on infrastructure
	base.SetDependencies([]string{"infrastructure"})

	return &ObservabilityEngine{
		BaseEngine: base,
		provider:   provider,
	}
}

// SetPlugin sets the plugin to use for observability operations
func (e *ObservabilityEngine) SetPlugin(plugin Plugin) {
	e.plugin = plugin
}

// DependsOn returns observability dependencies
func (e *ObservabilityEngine) DependsOn() []string {
	return []string{"infrastructure"}
}

// Validate validates the observability specification
func (e *ObservabilityEngine) Validate(spec map[string]interface{}) error {
	if e.plugin != nil {
		return e.plugin.Validate(spec)
	}

	if spec == nil {
		return fmt.Errorf("observability spec is required")
	}

	return nil
}

// Plan creates an observability execution plan
func (e *ObservabilityEngine) Plan(spec map[string]interface{}) (*Plan, error) {
	if e.plugin != nil {
		return e.plugin.Plan(spec)
	}

	actions := []PlanAction{
		{Type: "create", Resource: "namespace", Description: "Create monitoring namespace"},
		{Type: "create", Resource: "prometheus", Description: "Deploy Prometheus"},
		{Type: "create", Resource: "grafana", Description: "Deploy Grafana"},
		{Type: "create", Resource: "alertmanager", Description: "Deploy Alertmanager"},
	}

	return &Plan{
		Description: "Observability stack deployment plan",
		Actions:     actions,
	}, nil
}

// Apply deploys the observability stack
func (e *ObservabilityEngine) Apply(spec map[string]interface{}) (*Result, error) {
	startTime := time.Now()

	// Check mock mode
	if e.IsMockMode() {
		return e.mockApply(spec)
	}

	// Wait for dependencies
	e.SetState(EngineStateWaiting)
	e.SetProgress(0, "Waiting for infrastructure", 1, 6)

	ctx := e.GetContext()
	if ctx == nil {
		ctx = context.Background()
	}

	if err := e.WaitForDependencies(ctx); err != nil {
		e.SetState(EngineStateFailed)
		return nil, fmt.Errorf("dependency wait failed: %w", err)
	}

	// Inject dependency outputs
	if err := e.InjectDependencyOutputs(spec); err != nil {
		e.Log(fmt.Sprintf("Warning: failed to inject dependency outputs: %v", err))
	}

	e.SetState(EngineStateRunning)

	// Phase 1: Validate (10-20%)
	e.SetProgress(10, "Validating observability spec", 2, 6)
	e.Log("Validating observability specification...")

	if err := e.Validate(spec); err != nil {
		e.SetState(EngineStateFailed)
		return nil, fmt.Errorf("validation failed: %w", err)
	}
	e.SetProgress(20, "Validation complete", 2, 6)

	// Phase 2: Plan (20-30%)
	e.SetProgress(25, "Creating observability plan", 3, 6)

	plan, err := e.Plan(spec)
	if err != nil {
		e.SetState(EngineStateFailed)
		return nil, fmt.Errorf("planning failed: %w", err)
	}

	e.SetRollbackPlan(e.CreateRollbackPlan(plan))
	e.SetProgress(30, "Plan created", 3, 6)

	// Phase 3: Apply (30-80%)
	e.SetProgress(35, "Deploying observability stack", 4, 6)
	e.Log("Deploying observability components...")

	var result *Result
	if e.plugin != nil {
		result, err = e.plugin.Apply(spec)
		if err != nil {
			e.SetState(EngineStateFailed)
			return nil, fmt.Errorf("apply failed: %w", err)
		}
	} else {
		result = &Result{
			Status:    "success",
			Message:   "Observability stack deployed (simulated)",
			Resources: []string{"namespace", "prometheus", "grafana", "alertmanager"},
			Outputs:   e.generateDefaultOutputs(spec),
		}
	}
	e.SetProgress(80, "Observability stack deployed", 5, 6)

	// Phase 4: Store outputs (80-100%)
	e.SetProgress(90, "Storing outputs", 6, 6)

	if result.Outputs != nil {
		for key, value := range result.Outputs {
			e.SetOutput(key, value)
		}
	}

	e.ensureOutputs(spec, result)

	e.SetProgress(100, "Completed", 6, 6)
	e.SetState(EngineStateCompleted)

	result.Duration = time.Since(startTime)
	return result, nil
}

// mockApply provides instant mock response
func (e *ObservabilityEngine) mockApply(spec map[string]interface{}) (*Result, error) {
	e.SetState(EngineStateWaiting)
	e.SetProgress(5, "Waiting for infrastructure (mock)", 1, 5)

	ctx := e.GetContext()
	if ctx == nil {
		ctx = context.Background()
	}

	// Wait for mock dependencies
	if err := e.WaitForDependencies(ctx); err != nil {
		e.SetState(EngineStateFailed)
		return nil, fmt.Errorf("dependency wait failed: %w", err)
	}

	e.SetState(EngineStateRunning)

	phases := []string{"Validating", "Planning", "Deploying Prometheus", "Deploying Grafana", "Completing"}
	mockConfig := e.mockConfig

	for i, phase := range phases {
		e.SetProgress((i+1)*20, phase, i+2, len(phases)+1)

		if mockConfig != nil && mockConfig.SimulatedDelay > 0 {
			time.Sleep(mockConfig.SimulatedDelay / time.Duration(len(phases)))
		} else {
			time.Sleep(200 * time.Millisecond)
		}
	}

	outputs := e.generateMockOutputs(spec)
	for key, value := range outputs {
		e.SetOutput(key, value)
	}

	e.SetState(EngineStateCompleted)

	return &Result{
		Status:    "success",
		Message:   fmt.Sprintf("Mock observability stack (%s) deployed", e.provider),
		Resources: []string{"namespace", "prometheus", "grafana", "alertmanager", "node-exporter"},
		Outputs:   outputs,
	}, nil
}

// generateMockOutputs generates realistic mock outputs
func (e *ObservabilityEngine) generateMockOutputs(spec map[string]interface{}) map[string]interface{} {
	outputs := make(map[string]interface{})

	name := "platform"
	if n, ok := spec["name"].(string); ok {
		name = n
	}

	switch e.provider {
	case "prometheus-stack", "prometheus":
		outputs["prometheus_url"] = fmt.Sprintf("https://prometheus.%s.local", name)
		outputs["prometheus_service"] = "prometheus-server.monitoring.svc.cluster.local"
		outputs["prometheus_namespace"] = "monitoring"
		outputs["prometheus_version"] = "2.47.0"

		outputs["grafana_url"] = fmt.Sprintf("https://grafana.%s.local", name)
		outputs["grafana_service"] = "grafana.monitoring.svc.cluster.local"
		outputs["grafana_admin_user"] = "admin"
		outputs["grafana_admin_password"] = "mock-grafana-password"

		outputs["alertmanager_url"] = fmt.Sprintf("https://alertmanager.%s.local", name)
		outputs["alertmanager_service"] = "alertmanager.monitoring.svc.cluster.local"

	case "datadog":
		outputs["datadog_site"] = "datadoghq.com"
		outputs["datadog_namespace"] = "datadog"
		outputs["datadog_cluster_agent"] = "datadog-cluster-agent"

	case "newrelic":
		outputs["newrelic_namespace"] = "newrelic"
		outputs["newrelic_account_id"] = "mock-account-id"

	default:
		outputs["observability_namespace"] = "monitoring"
		outputs["metrics_endpoint"] = fmt.Sprintf("https://metrics.%s.local", name)
	}

	return outputs
}

// generateDefaultOutputs generates default outputs for non-plugin execution
func (e *ObservabilityEngine) generateDefaultOutputs(spec map[string]interface{}) map[string]interface{} {
	return e.generateMockOutputs(spec)
}

// ensureOutputs ensures all critical outputs are set
func (e *ObservabilityEngine) ensureOutputs(spec map[string]interface{}, result *Result) {
	// Ensure prometheus_url is set
	if _, ok := e.GetOutput("prometheus_url"); !ok {
		if url, ok := result.Outputs["prometheus_url"]; ok {
			e.SetOutput("prometheus_url", url)
		}
	}

	// Ensure grafana_url is set
	if _, ok := e.GetOutput("grafana_url"); !ok {
		if url, ok := result.Outputs["grafana_url"]; ok {
			e.SetOutput("grafana_url", url)
		}
	}
}

// Delete removes the observability stack
func (e *ObservabilityEngine) Delete() error {
	if e.plugin != nil {
		namespace, _ := e.GetOutput("prometheus_namespace")
		if ns, ok := namespace.(string); ok {
			return e.plugin.Delete(ns)
		}
		return e.plugin.Delete("monitoring")
	}
	return nil
}

// HealthCheck performs observability health check
func (e *ObservabilityEngine) HealthCheck() (*HealthStatus, error) {
	if e.plugin != nil {
		return e.plugin.Status(e.provider)
	}

	return &HealthStatus{
		Healthy: true,
		Message: "Observability engine is healthy",
		Checks: []HealthCheck{
			{Name: "prometheus", Status: "pass", Message: "Prometheus running"},
			{Name: "grafana", Status: "pass", Message: "Grafana running"},
			{Name: "alertmanager", Status: "pass", Message: "Alertmanager running"},
		},
	}, nil
}
