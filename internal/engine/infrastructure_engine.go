package engine

import (
	"fmt"
	"time"
)

// InfrastructureEngine handles infrastructure provisioning
type InfrastructureEngine struct {
	*BaseEngine

	cloudProvider string
	plugin        Plugin
}

// Plugin interface for engine plugins
type Plugin interface {
	Name() string
	Type() string
	Version() string
	Validate(spec map[string]interface{}) error
	Plan(spec map[string]interface{}) (*Plan, error)
	Apply(spec map[string]interface{}) (*Result, error)
	Delete(name string) error
	Status(name string) (*HealthStatus, error)
}

// NewInfrastructureEngine creates a new infrastructure engine
func NewInfrastructureEngine(provider string) *InfrastructureEngine {
	base := NewBaseEngine("infrastructure", "Infrastructure")
	// Infrastructure has no dependencies - it runs first
	base.SetDependencies([]string{})

	return &InfrastructureEngine{
		BaseEngine:    base,
		cloudProvider: provider,
	}
}

// SetPlugin sets the plugin to use for infrastructure operations
func (e *InfrastructureEngine) SetPlugin(plugin Plugin) {
	e.plugin = plugin
}

// DependsOn returns infrastructure dependencies (none - it's the foundation)
func (e *InfrastructureEngine) DependsOn() []string {
	return []string{}
}

// Validate validates the infrastructure specification
func (e *InfrastructureEngine) Validate(spec map[string]interface{}) error {
	if e.plugin != nil {
		return e.plugin.Validate(spec)
	}

	// Basic validation
	if spec == nil {
		return fmt.Errorf("infrastructure spec is required")
	}

	return nil
}

// Plan creates an infrastructure execution plan
func (e *InfrastructureEngine) Plan(spec map[string]interface{}) (*Plan, error) {
	if e.plugin != nil {
		return e.plugin.Plan(spec)
	}

	return &Plan{
		Description: "Infrastructure provisioning plan",
		Actions: []PlanAction{
			{Type: "create", Resource: "vpc", Description: "Create VPC"},
			{Type: "create", Resource: "subnets", Description: "Create subnets"},
			{Type: "create", Resource: "cluster", Description: "Create Kubernetes cluster"},
		},
	}, nil
}

// Apply provisions the infrastructure
func (e *InfrastructureEngine) Apply(spec map[string]interface{}) (*Result, error) {
	startTime := time.Now()

	// Check mock mode
	if e.IsMockMode() {
		return e.mockApply(spec)
	}

	e.SetState(EngineStateRunning)

	// Phase 1: Validate (0-10%)
	e.SetProgress(0, "Validating infrastructure spec", 1, 5)
	e.Log("Validating infrastructure specification...")

	if err := e.Validate(spec); err != nil {
		e.SetState(EngineStateFailed)
		return nil, fmt.Errorf("validation failed: %w", err)
	}
	e.SetProgress(10, "Validation complete", 1, 5)

	// Phase 2: Plan (10-30%)
	e.SetProgress(15, "Creating infrastructure plan", 2, 5)
	e.Log("Creating infrastructure plan...")

	plan, err := e.Plan(spec)
	if err != nil {
		e.SetState(EngineStateFailed)
		return nil, fmt.Errorf("planning failed: %w", err)
	}

	// Store rollback plan
	e.SetRollbackPlan(e.CreateRollbackPlan(plan))
	e.SetProgress(30, "Plan created", 2, 5)

	// Phase 3: Apply with plugin (30-90%)
	e.SetProgress(35, "Applying infrastructure changes", 3, 5)
	e.Log("Applying infrastructure changes...")

	var result *Result
	if e.plugin != nil {
		result, err = e.plugin.Apply(spec)
		if err != nil {
			e.SetState(EngineStateFailed)
			return nil, fmt.Errorf("apply failed: %w", err)
		}
	} else {
		// Simulated apply without plugin
		result = &Result{
			Status:    "success",
			Message:   "Infrastructure provisioned (simulated)",
			Resources: []string{"vpc", "subnets", "cluster"},
			Outputs:   e.generateDefaultOutputs(spec),
		}
	}
	e.SetProgress(90, "Infrastructure applied", 4, 5)

	// Phase 4: Store outputs (90-100%)
	e.SetProgress(95, "Storing outputs", 5, 5)

	// Store commonly needed outputs for dependent engines
	if result.Outputs != nil {
		for key, value := range result.Outputs {
			e.SetOutput(key, value)
		}
	}

	// Ensure critical outputs are set
	e.ensureOutputs(spec, result)

	e.SetProgress(100, "Completed", 5, 5)
	e.SetState(EngineStateCompleted)

	result.Duration = time.Since(startTime)
	return result, nil
}

// mockApply provides instant mock response
func (e *InfrastructureEngine) mockApply(spec map[string]interface{}) (*Result, error) {
	e.SetState(EngineStateRunning)

	phases := []string{"Validating", "Planning", "Applying", "Completing"}
	mockConfig := e.mockConfig

	for i, phase := range phases {
		e.SetProgress((i+1)*25, phase, i+1, len(phases))

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
		Message:   fmt.Sprintf("Mock infrastructure (%s) provisioned", e.cloudProvider),
		Resources: []string{"vpc", "subnets", "cluster"},
		Outputs:   outputs,
	}, nil
}

// generateMockOutputs generates realistic mock outputs
func (e *InfrastructureEngine) generateMockOutputs(spec map[string]interface{}) map[string]interface{} {
	outputs := make(map[string]interface{})

	// Get name from spec or generate default
	name := "platform"
	if n, ok := spec["name"].(string); ok {
		name = n
	}

	switch e.cloudProvider {
	case "aws":
		outputs["cluster_endpoint"] = fmt.Sprintf("https://%s.us-west-2.eks.amazonaws.com", name)
		outputs["cluster_name"] = fmt.Sprintf("%s-eks", name)
		outputs["cluster_arn"] = fmt.Sprintf("arn:aws:eks:us-west-2:123456789:cluster/%s", name)
		outputs["vpc_id"] = fmt.Sprintf("vpc-%s-12345", name)
		outputs["subnet_ids"] = []string{"subnet-mock-1", "subnet-mock-2", "subnet-mock-3"}
		outputs["security_group_id"] = fmt.Sprintf("sg-%s-67890", name)
		outputs["oidc_issuer"] = fmt.Sprintf("https://oidc.eks.us-west-2.amazonaws.com/id/%s", name)

	case "gcp":
		outputs["cluster_endpoint"] = fmt.Sprintf("https://%s.us-central1.gke.io", name)
		outputs["cluster_name"] = fmt.Sprintf("%s-gke", name)
		outputs["network_name"] = fmt.Sprintf("%s-network", name)
		outputs["cluster_ca"] = "mock-ca-certificate-data"
		outputs["project_id"] = fmt.Sprintf("%s-project", name)

	case "azure":
		outputs["cluster_endpoint"] = fmt.Sprintf("https://%s.eastus.azmk8s.io", name)
		outputs["cluster_name"] = fmt.Sprintf("%s-aks", name)
		outputs["resource_group"] = fmt.Sprintf("%s-rg", name)
		outputs["vnet_id"] = fmt.Sprintf("/subscriptions/xxx/resourceGroups/%s-rg/providers/Microsoft.Network/virtualNetworks/%s-vnet", name, name)

	default:
		outputs["cluster_endpoint"] = fmt.Sprintf("https://%s.local:6443", name)
		outputs["cluster_name"] = name
		outputs["vpc_id"] = fmt.Sprintf("vpc-%s", name)
	}

	outputs["kubeconfig"] = e.generateMockKubeconfig(outputs)

	return outputs
}

// generateDefaultOutputs generates default outputs for non-plugin execution
func (e *InfrastructureEngine) generateDefaultOutputs(spec map[string]interface{}) map[string]interface{} {
	return e.generateMockOutputs(spec)
}

// ensureOutputs ensures all critical outputs are set
func (e *InfrastructureEngine) ensureOutputs(spec map[string]interface{}, result *Result) {
	// Ensure cluster_endpoint is set
	if _, ok := e.GetOutput("cluster_endpoint"); !ok {
		if endpoint, ok := result.Outputs["cluster_endpoint"]; ok {
			e.SetOutput("cluster_endpoint", endpoint)
		}
	}

	// Ensure vpc_id is set
	if _, ok := e.GetOutput("vpc_id"); !ok {
		if vpcID, ok := result.Outputs["vpc_id"]; ok {
			e.SetOutput("vpc_id", vpcID)
		} else if name, ok := spec["name"].(string); ok {
			e.SetOutput("vpc_id", fmt.Sprintf("vpc-%s", name))
		}
	}
}

// generateMockKubeconfig generates a mock kubeconfig
func (e *InfrastructureEngine) generateMockKubeconfig(outputs map[string]interface{}) string {
	endpoint := outputs["cluster_endpoint"]
	clusterName := outputs["cluster_name"]

	return fmt.Sprintf(`apiVersion: v1
kind: Config
clusters:
- cluster:
    server: %s
    certificate-authority-data: bW9jay1jYS1kYXRh
  name: %s
contexts:
- context:
    cluster: %s
    user: admin
  name: %s
current-context: %s
users:
- name: admin
  user:
    token: mock-token
`, endpoint, clusterName, clusterName, clusterName, clusterName)
}

// Delete removes the infrastructure
func (e *InfrastructureEngine) Delete() error {
	if e.plugin != nil {
		name, _ := e.GetOutput("cluster_name")
		if nameStr, ok := name.(string); ok {
			return e.plugin.Delete(nameStr)
		}
	}
	return nil
}

// HealthCheck performs infrastructure health check
func (e *InfrastructureEngine) HealthCheck() (*HealthStatus, error) {
	if e.plugin != nil {
		name, _ := e.GetOutput("cluster_name")
		if nameStr, ok := name.(string); ok {
			return e.plugin.Status(nameStr)
		}
	}

	return &HealthStatus{
		Healthy: true,
		Message: "Infrastructure engine is healthy",
		Checks: []HealthCheck{
			{Name: "plugin", Status: "pass", Message: "Plugin available"},
		},
	}, nil
}
