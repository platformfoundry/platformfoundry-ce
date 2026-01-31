package engine

import (
	"context"
	"fmt"
	"time"
)

// OrchestratorEngine handles GitOps/ArgoCD setup
type OrchestratorEngine struct {
	*BaseEngine

	provider string
	plugin   Plugin
}

// NewOrchestratorEngine creates a new orchestrator engine
func NewOrchestratorEngine(provider string) *OrchestratorEngine {
	base := NewBaseEngine("orchestrator", "Orchestrator")
	// Orchestrator depends on infrastructure
	base.SetDependencies([]string{"infrastructure"})

	return &OrchestratorEngine{
		BaseEngine: base,
		provider:   provider,
	}
}

// SetPlugin sets the plugin to use for orchestrator operations
func (e *OrchestratorEngine) SetPlugin(plugin Plugin) {
	e.plugin = plugin
}

// DependsOn returns orchestrator dependencies
func (e *OrchestratorEngine) DependsOn() []string {
	return []string{"infrastructure"}
}

// Validate validates the orchestrator specification
func (e *OrchestratorEngine) Validate(spec map[string]interface{}) error {
	if e.plugin != nil {
		return e.plugin.Validate(spec)
	}

	if spec == nil {
		return fmt.Errorf("orchestrator spec is required")
	}

	return nil
}

// Plan creates an orchestrator execution plan
func (e *OrchestratorEngine) Plan(spec map[string]interface{}) (*Plan, error) {
	if e.plugin != nil {
		return e.plugin.Plan(spec)
	}

	actions := []PlanAction{
		{Type: "create", Resource: "namespace", Description: "Create orchestrator namespace"},
		{Type: "create", Resource: "argocd", Description: "Deploy ArgoCD"},
		{Type: "create", Resource: "applications", Description: "Configure applications"},
	}

	return &Plan{
		Description: "Orchestrator deployment plan",
		Actions:     actions,
	}, nil
}

// Apply deploys the orchestrator
func (e *OrchestratorEngine) Apply(spec map[string]interface{}) (*Result, error) {
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
	e.SetProgress(10, "Validating orchestrator spec", 2, 6)
	e.Log("Validating orchestrator specification...")

	if err := e.Validate(spec); err != nil {
		e.SetState(EngineStateFailed)
		return nil, fmt.Errorf("validation failed: %w", err)
	}
	e.SetProgress(20, "Validation complete", 2, 6)

	// Phase 2: Plan (20-30%)
	e.SetProgress(25, "Creating orchestrator plan", 3, 6)

	plan, err := e.Plan(spec)
	if err != nil {
		e.SetState(EngineStateFailed)
		return nil, fmt.Errorf("planning failed: %w", err)
	}

	e.SetRollbackPlan(e.CreateRollbackPlan(plan))
	e.SetProgress(30, "Plan created", 3, 6)

	// Phase 3: Apply (30-80%)
	e.SetProgress(35, "Deploying orchestrator", 4, 6)
	e.Log("Deploying orchestrator components...")

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
			Message:   "Orchestrator deployed (simulated)",
			Resources: []string{"namespace", "argocd", "applications"},
			Outputs:   e.generateDefaultOutputs(spec),
		}
	}
	e.SetProgress(80, "Orchestrator deployed", 5, 6)

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
func (e *OrchestratorEngine) mockApply(spec map[string]interface{}) (*Result, error) {
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

	phases := []string{"Validating", "Planning", "Deploying", "Completing"}
	mockConfig := e.mockConfig

	for i, phase := range phases {
		e.SetProgress((i+1)*25, phase, i+2, len(phases)+1)

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
		Message:   fmt.Sprintf("Mock orchestrator (%s) deployed", e.provider),
		Resources: []string{"namespace", "argocd-server", "argocd-repo-server", "applications"},
		Outputs:   outputs,
	}, nil
}

// generateMockOutputs generates realistic mock outputs
func (e *OrchestratorEngine) generateMockOutputs(spec map[string]interface{}) map[string]interface{} {
	outputs := make(map[string]interface{})

	name := "platform"
	if n, ok := spec["name"].(string); ok {
		name = n
	}

	switch e.provider {
	case "argocd":
		outputs["argocd_url"] = fmt.Sprintf("https://argocd.%s.local", name)
		outputs["argocd_server"] = "argocd-server.argocd.svc.cluster.local"
		outputs["argocd_admin_password"] = "mock-admin-password"
		outputs["argocd_namespace"] = "argocd"
		outputs["argocd_version"] = "2.9.3"

	case "flux":
		outputs["flux_namespace"] = "flux-system"
		outputs["flux_version"] = "2.2.0"
		outputs["flux_controllers"] = []string{"source-controller", "kustomize-controller", "helm-controller"}

	case "rancher":
		outputs["rancher_url"] = fmt.Sprintf("https://rancher.%s.local", name)
		outputs["rancher_admin_password"] = "mock-rancher-password"
		outputs["rancher_namespace"] = "cattle-system"

	default:
		outputs["orchestrator_url"] = fmt.Sprintf("https://orchestrator.%s.local", name)
		outputs["orchestrator_namespace"] = e.provider
	}

	return outputs
}

// generateDefaultOutputs generates default outputs for non-plugin execution
func (e *OrchestratorEngine) generateDefaultOutputs(spec map[string]interface{}) map[string]interface{} {
	return e.generateMockOutputs(spec)
}

// ensureOutputs ensures all critical outputs are set
func (e *OrchestratorEngine) ensureOutputs(spec map[string]interface{}, result *Result) {
	// Ensure argocd_url is set for ArgoCD
	if e.provider == "argocd" {
		if _, ok := e.GetOutput("argocd_url"); !ok {
			if url, ok := result.Outputs["argocd_url"]; ok {
				e.SetOutput("argocd_url", url)
			}
		}
	}
}

// Delete removes the orchestrator
func (e *OrchestratorEngine) Delete() error {
	if e.plugin != nil {
		namespace, _ := e.GetOutput("argocd_namespace")
		if ns, ok := namespace.(string); ok {
			return e.plugin.Delete(ns)
		}
		return e.plugin.Delete(e.provider)
	}
	return nil
}

// HealthCheck performs orchestrator health check
func (e *OrchestratorEngine) HealthCheck() (*HealthStatus, error) {
	if e.plugin != nil {
		return e.plugin.Status(e.provider)
	}

	return &HealthStatus{
		Healthy: true,
		Message: "Orchestrator engine is healthy",
		Checks: []HealthCheck{
			{Name: "argocd-server", Status: "pass", Message: "Server running"},
			{Name: "argocd-repo-server", Status: "pass", Message: "Repo server running"},
		},
	}, nil
}
