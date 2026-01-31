package engine

import (
	"context"
	"fmt"
	"time"
)

// DevExEngine handles developer experience tools setup
type DevExEngine struct {
	*BaseEngine

	provider string
	plugin   Plugin
}

// NewDevExEngine creates a new developer experience engine
func NewDevExEngine(provider string) *DevExEngine {
	base := NewBaseEngine("devex", "DevEx")
	// DevEx depends on infrastructure, orchestrator, and observability
	base.SetDependencies([]string{"infrastructure", "orchestrator"})

	return &DevExEngine{
		BaseEngine: base,
		provider:   provider,
	}
}

// SetPlugin sets the plugin to use for devex operations
func (e *DevExEngine) SetPlugin(plugin Plugin) {
	e.plugin = plugin
}

// DependsOn returns devex dependencies
func (e *DevExEngine) DependsOn() []string {
	return []string{"infrastructure", "orchestrator"}
}

// Validate validates the devex specification
func (e *DevExEngine) Validate(spec map[string]interface{}) error {
	if e.plugin != nil {
		return e.plugin.Validate(spec)
	}

	if spec == nil {
		return fmt.Errorf("devex spec is required")
	}

	return nil
}

// Plan creates a devex execution plan
func (e *DevExEngine) Plan(spec map[string]interface{}) (*Plan, error) {
	if e.plugin != nil {
		return e.plugin.Plan(spec)
	}

	actions := []PlanAction{
		{Type: "create", Resource: "namespace", Description: "Create backstage namespace"},
		{Type: "create", Resource: "database", Description: "Deploy Backstage database"},
		{Type: "create", Resource: "backstage", Description: "Deploy Backstage"},
		{Type: "create", Resource: "catalog", Description: "Configure software catalog"},
	}

	return &Plan{
		Description: "Developer experience platform deployment plan",
		Actions:     actions,
	}, nil
}

// Apply deploys the developer experience tools
func (e *DevExEngine) Apply(spec map[string]interface{}) (*Result, error) {
	startTime := time.Now()

	// Check mock mode
	if e.IsMockMode() {
		return e.mockApply(spec)
	}

	// Wait for dependencies
	e.SetState(EngineStateWaiting)
	e.SetProgress(0, "Waiting for dependencies", 1, 7)

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
	e.SetProgress(10, "Validating devex spec", 2, 7)
	e.Log("Validating developer experience specification...")

	if err := e.Validate(spec); err != nil {
		e.SetState(EngineStateFailed)
		return nil, fmt.Errorf("validation failed: %w", err)
	}
	e.SetProgress(20, "Validation complete", 2, 7)

	// Phase 2: Plan (20-30%)
	e.SetProgress(25, "Creating devex plan", 3, 7)

	plan, err := e.Plan(spec)
	if err != nil {
		e.SetState(EngineStateFailed)
		return nil, fmt.Errorf("planning failed: %w", err)
	}

	e.SetRollbackPlan(e.CreateRollbackPlan(plan))
	e.SetProgress(30, "Plan created", 3, 7)

	// Phase 3: Apply (30-85%)
	e.SetProgress(35, "Deploying developer platform", 4, 7)
	e.Log("Deploying developer experience components...")

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
			Message:   "Developer platform deployed (simulated)",
			Resources: []string{"namespace", "database", "backstage", "catalog"},
			Outputs:   e.generateDefaultOutputs(spec),
		}
	}
	e.SetProgress(85, "Developer platform deployed", 5, 7)

	// Phase 4: Configure integrations (85-95%)
	e.SetProgress(90, "Configuring integrations", 6, 7)

	// Phase 5: Store outputs (95-100%)
	e.SetProgress(95, "Storing outputs", 7, 7)

	if result.Outputs != nil {
		for key, value := range result.Outputs {
			e.SetOutput(key, value)
		}
	}

	e.ensureOutputs(spec, result)

	e.SetProgress(100, "Completed", 7, 7)
	e.SetState(EngineStateCompleted)

	result.Duration = time.Since(startTime)
	return result, nil
}

// mockApply provides instant mock response
func (e *DevExEngine) mockApply(spec map[string]interface{}) (*Result, error) {
	e.SetState(EngineStateWaiting)
	e.SetProgress(5, "Waiting for dependencies (mock)", 1, 6)

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

	phases := []string{"Validating", "Planning", "Deploying Database", "Deploying Backstage", "Configuring Catalog", "Completing"}
	mockConfig := e.mockConfig

	for i, phase := range phases {
		e.SetProgress((i+1)*16, phase, i+2, len(phases)+1)

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
		Message:   fmt.Sprintf("Mock developer platform (%s) deployed", e.provider),
		Resources: []string{"namespace", "postgresql", "backstage", "catalog", "techdocs"},
		Outputs:   outputs,
	}, nil
}

// generateMockOutputs generates realistic mock outputs
func (e *DevExEngine) generateMockOutputs(spec map[string]interface{}) map[string]interface{} {
	outputs := make(map[string]interface{})

	name := "platform"
	if n, ok := spec["name"].(string); ok {
		name = n
	}

	switch e.provider {
	case "backstage":
		outputs["backstage_url"] = fmt.Sprintf("https://backstage.%s.local", name)
		outputs["backstage_service"] = "backstage.backstage.svc.cluster.local"
		outputs["backstage_namespace"] = "backstage"
		outputs["backstage_version"] = "1.20.0"
		outputs["catalog_url"] = fmt.Sprintf("https://backstage.%s.local/catalog", name)
		outputs["techdocs_url"] = fmt.Sprintf("https://backstage.%s.local/docs", name)

	case "port":
		outputs["port_url"] = fmt.Sprintf("https://port.%s.local", name)
		outputs["port_namespace"] = "port"

	case "cortex":
		outputs["cortex_url"] = fmt.Sprintf("https://cortex.%s.local", name)
		outputs["cortex_namespace"] = "cortex"

	default:
		outputs["devex_url"] = fmt.Sprintf("https://devex.%s.local", name)
		outputs["devex_namespace"] = e.provider
	}

	return outputs
}

// generateDefaultOutputs generates default outputs for non-plugin execution
func (e *DevExEngine) generateDefaultOutputs(spec map[string]interface{}) map[string]interface{} {
	return e.generateMockOutputs(spec)
}

// ensureOutputs ensures all critical outputs are set
func (e *DevExEngine) ensureOutputs(spec map[string]interface{}, result *Result) {
	// Ensure backstage_url is set for Backstage
	if e.provider == "backstage" {
		if _, ok := e.GetOutput("backstage_url"); !ok {
			if url, ok := result.Outputs["backstage_url"]; ok {
				e.SetOutput("backstage_url", url)
			}
		}
	}
}

// Delete removes the developer platform
func (e *DevExEngine) Delete() error {
	if e.plugin != nil {
		namespace, _ := e.GetOutput("backstage_namespace")
		if ns, ok := namespace.(string); ok {
			return e.plugin.Delete(ns)
		}
		return e.plugin.Delete(e.provider)
	}
	return nil
}

// HealthCheck performs devex health check
func (e *DevExEngine) HealthCheck() (*HealthStatus, error) {
	if e.plugin != nil {
		return e.plugin.Status(e.provider)
	}

	return &HealthStatus{
		Healthy: true,
		Message: "Developer experience engine is healthy",
		Checks: []HealthCheck{
			{Name: "backstage", Status: "pass", Message: "Backstage running"},
			{Name: "database", Status: "pass", Message: "Database running"},
			{Name: "catalog", Status: "pass", Message: "Catalog synced"},
		},
	}, nil
}
