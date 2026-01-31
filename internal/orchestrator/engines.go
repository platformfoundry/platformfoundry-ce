package orchestrator

import (
	"fmt"

	"github.com/platformfoundry/platformfoundry-ce/internal/engine"
	"github.com/platformfoundry/platformfoundry-ce/internal/plugin"
	"github.com/platformfoundry/platformfoundry-ce/internal/workload"
)

// InfrastructureEngine orchestrates infrastructure resource provisioning
type InfrastructureEngine struct {
	*engine.BaseEngine
	pluginManager *plugin.Manager
}

// NewInfrastructureEngine creates a new infrastructure engine
func NewInfrastructureEngine(name string, pm *plugin.Manager) *InfrastructureEngine {
	return &InfrastructureEngine{
		BaseEngine:    engine.NewBaseEngine(name, "infrastructure"),
		pluginManager: pm,
	}
}

// Validate validates the infrastructure spec
func (e *InfrastructureEngine) Validate(spec map[string]interface{}) error {
	resources, ok := spec["resources"].([]workload.InfraResource)
	if !ok {
		// Try to parse from interface slice
		if rawResources, ok := spec["resources"].([]interface{}); ok {
			for _, r := range rawResources {
				if _, ok := r.(workload.InfraResource); !ok {
					if rm, ok := r.(map[string]interface{}); ok {
						if rm["type"] == nil || rm["name"] == nil {
							return fmt.Errorf("invalid resource: missing type or name")
						}
					}
				}
			}
			return nil
		}
		return fmt.Errorf("invalid resources spec")
	}

	for _, res := range resources {
		if res.Type == "" || res.Name == "" {
			return fmt.Errorf("resource type and name are required")
		}
	}

	return nil
}

// Plan generates an execution plan for infrastructure
func (e *InfrastructureEngine) Plan(spec map[string]interface{}) (*engine.Plan, error) {
	resources := e.extractResources(spec)

	plan := &engine.Plan{
		Description: fmt.Sprintf("Provision %d infrastructure resources", len(resources)),
		Actions:     make([]engine.PlanAction, 0, len(resources)),
	}

	for _, res := range resources {
		plan.Actions = append(plan.Actions, engine.PlanAction{
			Type:     "create",
			Resource: fmt.Sprintf("%s/%s", res.Type, res.Name),
			Details: map[string]interface{}{
				"provider": res.Provider,
				"config":   res.Config,
			},
		})
	}

	return plan, nil
}

// Apply provisions infrastructure resources
func (e *InfrastructureEngine) Apply(spec map[string]interface{}) (*engine.Result, error) {
	// Get context for potential future use
	_ = e.GetContext()

	resources := e.extractResources(spec)
	totalResources := len(resources)

	result := &engine.Result{
		Status:    "success",
		Resources: make([]string, 0),
		Outputs:   make(map[string]interface{}),
	}

	for i, res := range resources {
		progress := ((i + 1) * 100) / totalResources
		e.SetProgress(progress, fmt.Sprintf("Provisioning %s/%s", res.Type, res.Name), i+1, totalResources)

		// Get plugin for this resource
		p, err := e.pluginManager.Get("Infrastructure", res.Provider)
		if err != nil {
			// Try AWS plugin for aws-* resources
			if p, err = e.pluginManager.Get("Infrastructure", "aws"); err != nil {
				e.LogError(err, fmt.Sprintf("Plugin not found for %s", res.Provider))
				result.Status = "partial"
				continue
			}
		}

		// Build spec for the plugin
		pluginSpec := map[string]interface{}{
			"region": getConfigValue(res.Config, "region", "us-east-1"),
			"resources": []map[string]interface{}{
				{
					"type":       mapResourceType(res.Type),
					"name":       res.Name,
					"properties": res.Config,
				},
			},
		}

		// Apply via plugin
		pluginResult, err := p.Apply(pluginSpec)
		if err != nil {
			e.LogError(err, fmt.Sprintf("Failed to provision %s/%s", res.Type, res.Name))
			result.Status = "partial"
			result.Message = fmt.Sprintf("Failed at %s: %v", res.Name, err)
			continue
		}

		result.Resources = append(result.Resources, fmt.Sprintf("%s:%s", res.Type, res.Name))

		// Merge outputs
		for k, v := range pluginResult.Outputs {
			result.Outputs[fmt.Sprintf("%s.%s", res.Name, k)] = v
			e.SetOutput(fmt.Sprintf("%s.%s", res.Name, k), v)
		}

		e.Log(fmt.Sprintf("Provisioned %s/%s", res.Type, res.Name))
	}

	if result.Status == "success" {
		result.Message = fmt.Sprintf("Successfully provisioned %d resources", len(result.Resources))
	}

	return result, nil
}

// Delete removes infrastructure resources
func (e *InfrastructureEngine) Delete() error {
	// Deletion handled by plugin
	return nil
}

func (e *InfrastructureEngine) extractResources(spec map[string]interface{}) []workload.InfraResource {
	resources := make([]workload.InfraResource, 0)

	if rawResources, ok := spec["resources"].([]workload.InfraResource); ok {
		return rawResources
	}

	if rawResources, ok := spec["resources"].([]interface{}); ok {
		for _, r := range rawResources {
			if res, ok := r.(workload.InfraResource); ok {
				resources = append(resources, res)
			} else if rm, ok := r.(map[string]interface{}); ok {
				res := workload.InfraResource{
					Type:     getStringValue(rm, "type"),
					Name:     getStringValue(rm, "name"),
					Provider: getStringValue(rm, "provider"),
				}
				if cfg, ok := rm["config"].(map[string]interface{}); ok {
					res.Config = cfg
				}
				resources = append(resources, res)
			}
		}
	}

	return resources
}

// KubernetesEngine orchestrates Kubernetes resource deployment
type KubernetesEngine struct {
	*engine.BaseEngine
	pluginManager *plugin.Manager
}

// NewKubernetesEngine creates a new Kubernetes engine
func NewKubernetesEngine(name string, pm *plugin.Manager) *KubernetesEngine {
	return &KubernetesEngine{
		BaseEngine:    engine.NewBaseEngine(name, "kubernetes"),
		pluginManager: pm,
	}
}

// Validate validates the Kubernetes spec
func (e *KubernetesEngine) Validate(spec map[string]interface{}) error {
	if spec["namespace"] == nil {
		return fmt.Errorf("namespace is required")
	}
	return nil
}

// Plan generates an execution plan for Kubernetes resources
func (e *KubernetesEngine) Plan(spec map[string]interface{}) (*engine.Plan, error) {
	plan := &engine.Plan{
		Description: "Deploy Kubernetes resources",
		Actions:     make([]engine.PlanAction, 0),
	}

	namespace := getStringValue(spec, "namespace")

	if spec["deployment"] != nil {
		plan.Actions = append(plan.Actions, engine.PlanAction{
			Type:     "create",
			Resource: fmt.Sprintf("Deployment/%s", namespace),
		})
	}

	if spec["service"] != nil {
		plan.Actions = append(plan.Actions, engine.PlanAction{
			Type:     "create",
			Resource: fmt.Sprintf("Service/%s", namespace),
		})
	}

	if spec["hpa"] != nil {
		plan.Actions = append(plan.Actions, engine.PlanAction{
			Type:     "create",
			Resource: fmt.Sprintf("HPA/%s", namespace),
		})
	}

	if spec["ingress"] != nil {
		plan.Actions = append(plan.Actions, engine.PlanAction{
			Type:     "create",
			Resource: fmt.Sprintf("Ingress/%s", namespace),
		})
	}

	return plan, nil
}

// Apply deploys Kubernetes resources
func (e *KubernetesEngine) Apply(spec map[string]interface{}) (*engine.Result, error) {
	// Get context for potential future use
	_ = e.GetContext()

	result := &engine.Result{
		Status:    "success",
		Resources: make([]string, 0),
		Outputs:   make(map[string]interface{}),
	}

	namespace := getStringValue(spec, "namespace")

	// Get Kubernetes plugin
	k8sPlugin, err := e.pluginManager.Get("Cluster", "kubernetes")
	if err != nil {
		return nil, fmt.Errorf("kubernetes plugin not found: %w", err)
	}

	// Build manifests from translation specs
	manifests := make([]map[string]interface{}, 0)

	// Add Deployment
	if deployment := spec["deployment"]; deployment != nil {
		if d, ok := deployment.(*workload.DeploymentSpec); ok {
			manifests = append(manifests, map[string]interface{}{
				"kind": "Deployment",
				"name": d.Name,
				"spec": map[string]interface{}{
					"replicas": d.Replicas,
					"image":    getFirstContainerImage(d),
					"port":     getFirstContainerPort(d),
				},
				"labels": d.Labels,
			})
		}
	}

	// Add Service
	if service := spec["service"]; service != nil {
		if s, ok := service.(*workload.ServiceSpec); ok {
			port := 80
			targetPort := 80
			if len(s.Ports) > 0 {
				port = s.Ports[0].Port
				targetPort = s.Ports[0].TargetPort
			}
			manifests = append(manifests, map[string]interface{}{
				"kind": "Service",
				"name": s.Name,
				"spec": map[string]interface{}{
					"type":       s.Type,
					"port":       port,
					"targetPort": targetPort,
				},
				"labels": s.Labels,
			})
		}
	}

	// Add HPA
	if hpa := spec["hpa"]; hpa != nil {
		if h, ok := hpa.(*workload.HPASpec); ok {
			manifests = append(manifests, map[string]interface{}{
				"kind": "HorizontalPodAutoscaler",
				"name": h.Name,
				"spec": map[string]interface{}{
					"minReplicas":          h.MinReplicas,
					"maxReplicas":          h.MaxReplicas,
					"targetCPUUtilization": h.TargetCPUUtilization,
					"scaleTargetRef":       h.Name,
				},
			})
		}
	}

	// Add Ingress
	if ingress := spec["ingress"]; ingress != nil {
		if ing, ok := ingress.(*workload.IngressResourceSpec); ok {
			manifests = append(manifests, map[string]interface{}{
				"kind": "Ingress",
				"name": ing.Name,
				"spec": map[string]interface{}{
					"host":        ing.Host,
					"path":        ing.Path,
					"serviceName": ing.ServiceName,
					"servicePort": ing.ServicePort,
					"tls":         ing.TLS,
				},
				"annotations": ing.Annotations,
			})
		}
	}

	if len(manifests) == 0 {
		result.Message = "No Kubernetes resources to deploy"
		return result, nil
	}

	// Apply via plugin
	pluginSpec := map[string]interface{}{
		"namespace": namespace,
		"manifests": manifests,
	}

	e.SetProgress(50, "Deploying Kubernetes resources", 1, 2)

	pluginResult, err := k8sPlugin.Apply(pluginSpec)
	if err != nil {
		e.LogError(err, "Failed to apply Kubernetes resources")
		result.Status = "failed"
		result.Message = err.Error()
		return result, err
	}

	e.SetProgress(100, "Kubernetes resources deployed", 2, 2)

	result.Resources = pluginResult.Resources
	for k, v := range pluginResult.Outputs {
		result.Outputs[k] = v
		e.SetOutput(k, v)
	}

	result.Message = fmt.Sprintf("Deployed %d Kubernetes resources", len(result.Resources))
	return result, nil
}

// Delete removes Kubernetes resources
func (e *KubernetesEngine) Delete() error {
	return nil
}

// Helper functions
func getStringValue(m map[string]interface{}, key string) string {
	if v, ok := m[key].(string); ok {
		return v
	}
	return ""
}

func getConfigValue(m map[string]interface{}, key, defaultVal string) string {
	if v, ok := m[key].(string); ok {
		return v
	}
	return defaultVal
}

func mapResourceType(resType string) string {
	// Map workload translator resource types to AWS resource types
	switch resType {
	case "aws-rds-postgres", "aws-rds-mysql":
		return "rds"
	case "aws-elasticache-redis", "aws-elasticache-memcached":
		return "elasticache"
	case "aws-s3":
		return "s3"
	case "aws-dynamodb":
		return "dynamodb"
	default:
		return resType
	}
}

func getFirstContainerImage(d *workload.DeploymentSpec) string {
	if len(d.Containers) > 0 {
		return d.Containers[0].Image
	}
	return ""
}

func getFirstContainerPort(d *workload.DeploymentSpec) int {
	if len(d.Containers) > 0 && len(d.Containers[0].Ports) > 0 {
		return d.Containers[0].Ports[0].ContainerPort
	}
	return 80
}
