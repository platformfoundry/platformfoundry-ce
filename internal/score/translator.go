package score

import (
	"fmt"
	"regexp"
	"strings"
)

// TranslationTarget represents the target platform for translation
type TranslationTarget string

const (
	TargetKubernetes TranslationTarget = "kubernetes"
	TargetCompose    TranslationTarget = "compose"
	TargetHelm       TranslationTarget = "helm"
	TargetTerraform  TranslationTarget = "terraform"
)

// Translator converts Score workloads to platform-specific formats
type Translator struct {
	parser          *Parser
	resourceDrivers map[string]ResourceDriver
	environment     string
}

// ResourceDriver provisions a specific resource type
type ResourceDriver interface {
	Type() string
	Provision(name string, resource *Resource, env string) (map[string]string, error)
	Translate(name string, resource *Resource, target TranslationTarget) (interface{}, error)
}

// TranslationResult contains the translated output
type TranslationResult struct {
	Target      TranslationTarget        `json:"target"`
	Workload    *Workload                `json:"workload"`
	Manifests   map[string]interface{}   `json:"manifests"`
	Resources   map[string]interface{}   `json:"resources"`
	Outputs     map[string]map[string]string `json:"outputs"`
	Warnings    []string                 `json:"warnings,omitempty"`
}

// NewTranslator creates a new Score translator
func NewTranslator(parser *Parser, env string) *Translator {
	return &Translator{
		parser:          parser,
		resourceDrivers: make(map[string]ResourceDriver),
		environment:     env,
	}
}

// RegisterDriver registers a resource driver
func (t *Translator) RegisterDriver(driver ResourceDriver) {
	t.resourceDrivers[driver.Type()] = driver
}

// Translate converts a Score workload to the target format
func (t *Translator) Translate(workload *Workload, target TranslationTarget) (*TranslationResult, error) {
	result := &TranslationResult{
		Target:    target,
		Workload:  workload,
		Manifests: make(map[string]interface{}),
		Resources: make(map[string]interface{}),
		Outputs:   make(map[string]map[string]string),
	}

	// Translate resources first to get outputs
	for name, resource := range workload.Resources {
		outputs, err := t.translateResource(name, &resource, target, result)
		if err != nil {
			return nil, fmt.Errorf("failed to translate resource %s: %w", name, err)
		}
		result.Outputs[name] = outputs
	}

	// Translate workload based on target
	switch target {
	case TargetKubernetes:
		if err := t.translateToKubernetes(workload, result); err != nil {
			return nil, err
		}
	case TargetCompose:
		if err := t.translateToCompose(workload, result); err != nil {
			return nil, err
		}
	case TargetHelm:
		if err := t.translateToHelm(workload, result); err != nil {
			return nil, err
		}
	default:
		return nil, fmt.Errorf("unsupported target: %s", target)
	}

	return result, nil
}

func (t *Translator) translateResource(name string, resource *Resource, target TranslationTarget, result *TranslationResult) (map[string]string, error) {
	// Get resource type definition
	rt, exists := t.parser.GetResourceType(resource.Type)
	if !exists {
		result.Warnings = append(result.Warnings, fmt.Sprintf("unknown resource type: %s", resource.Type))
	}

	// Check for driver
	if driver, ok := t.resourceDrivers[resource.Type]; ok {
		manifest, err := driver.Translate(name, resource, target)
		if err != nil {
			return nil, err
		}
		result.Resources[name] = manifest

		outputs, err := driver.Provision(name, resource, t.environment)
		if err != nil {
			return nil, err
		}
		return outputs, nil
	}

	// Generate placeholder outputs based on resource type definition
	outputs := make(map[string]string)
	if rt != nil {
		for outputName := range rt.Outputs {
			outputs[outputName] = fmt.Sprintf("${%s.%s.%s}", t.environment, name, outputName)
		}
	}

	return outputs, nil
}

func (t *Translator) translateToKubernetes(workload *Workload, result *TranslationResult) error {
	// Generate Deployment
	deployment := t.generateK8sDeployment(workload, result)
	result.Manifests["deployment"] = deployment

	// Generate Service if ports are exposed
	if workload.Service != nil && len(workload.Service.Ports) > 0 {
		service := t.generateK8sService(workload)
		result.Manifests["service"] = service
	}

	// Generate ConfigMap for non-secret variables
	configMap := t.generateK8sConfigMap(workload, result)
	if configMap != nil {
		result.Manifests["configmap"] = configMap
	}

	// Generate Secret for sensitive data
	secret := t.generateK8sSecret(workload, result)
	if secret != nil {
		result.Manifests["secret"] = secret
	}

	return nil
}

func (t *Translator) generateK8sDeployment(workload *Workload, result *TranslationResult) map[string]interface{} {
	containers := make([]map[string]interface{}, 0)

	for name, container := range workload.Containers {
		c := map[string]interface{}{
			"name":  name,
			"image": container.Image,
		}

		if len(container.Command) > 0 {
			c["command"] = container.Command
		}
		if len(container.Args) > 0 {
			c["args"] = container.Args
		}

		// Environment variables
		envVars := t.resolveEnvVars(container.Variables, result.Outputs)
		if len(envVars) > 0 {
			c["env"] = envVars
		}

		// Resource requirements
		if container.Resources != nil {
			resources := make(map[string]interface{})
			if container.Resources.Requests != nil {
				req := make(map[string]string)
				if container.Resources.Requests.CPU != "" {
					req["cpu"] = container.Resources.Requests.CPU
				}
				if container.Resources.Requests.Memory != "" {
					req["memory"] = container.Resources.Requests.Memory
				}
				if len(req) > 0 {
					resources["requests"] = req
				}
			}
			if container.Resources.Limits != nil {
				lim := make(map[string]string)
				if container.Resources.Limits.CPU != "" {
					lim["cpu"] = container.Resources.Limits.CPU
				}
				if container.Resources.Limits.Memory != "" {
					lim["memory"] = container.Resources.Limits.Memory
				}
				if len(lim) > 0 {
					resources["limits"] = lim
				}
			}
			if len(resources) > 0 {
				c["resources"] = resources
			}
		}

		// Probes
		if container.LivenessProbe != nil && container.LivenessProbe.HTTPGet != nil {
			c["livenessProbe"] = map[string]interface{}{
				"httpGet": map[string]interface{}{
					"path": container.LivenessProbe.HTTPGet.Path,
					"port": container.LivenessProbe.HTTPGet.Port,
				},
			}
		}
		if container.ReadinessProbe != nil && container.ReadinessProbe.HTTPGet != nil {
			c["readinessProbe"] = map[string]interface{}{
				"httpGet": map[string]interface{}{
					"path": container.ReadinessProbe.HTTPGet.Path,
					"port": container.ReadinessProbe.HTTPGet.Port,
				},
			}
		}

		// Volume mounts
		if len(container.Volumes) > 0 {
			mounts := make([]map[string]interface{}, 0)
			for _, vol := range container.Volumes {
				mount := map[string]interface{}{
					"name":      vol.Source,
					"mountPath": vol.Target,
				}
				if vol.ReadOnly {
					mount["readOnly"] = true
				}
				if vol.Path != "" {
					mount["subPath"] = vol.Path
				}
				mounts = append(mounts, mount)
			}
			c["volumeMounts"] = mounts
		}

		containers = append(containers, c)
	}

	deployment := map[string]interface{}{
		"apiVersion": "apps/v1",
		"kind":       "Deployment",
		"metadata": map[string]interface{}{
			"name":   workload.Metadata.Name,
			"labels": t.generateLabels(workload),
		},
		"spec": map[string]interface{}{
			"replicas": 1,
			"selector": map[string]interface{}{
				"matchLabels": map[string]string{
					"app": workload.Metadata.Name,
				},
			},
			"template": map[string]interface{}{
				"metadata": map[string]interface{}{
					"labels": t.generateLabels(workload),
				},
				"spec": map[string]interface{}{
					"containers": containers,
				},
			},
		},
	}

	return deployment
}

func (t *Translator) generateK8sService(workload *Workload) map[string]interface{} {
	ports := make([]map[string]interface{}, 0)
	for _, port := range workload.Service.Ports {
		p := map[string]interface{}{
			"port": port.Port,
		}
		if port.Name != "" {
			p["name"] = port.Name
		}
		if port.TargetPort != 0 {
			p["targetPort"] = port.TargetPort
		}
		if port.Protocol != "" {
			p["protocol"] = port.Protocol
		} else {
			p["protocol"] = "TCP"
		}
		ports = append(ports, p)
	}

	return map[string]interface{}{
		"apiVersion": "v1",
		"kind":       "Service",
		"metadata": map[string]interface{}{
			"name":   workload.Metadata.Name,
			"labels": t.generateLabels(workload),
		},
		"spec": map[string]interface{}{
			"selector": map[string]string{
				"app": workload.Metadata.Name,
			},
			"ports": ports,
		},
	}
}

func (t *Translator) generateK8sConfigMap(workload *Workload, result *TranslationResult) map[string]interface{} {
	data := make(map[string]string)

	for _, container := range workload.Containers {
		for varName, varValue := range container.Variables {
			// Skip secret references
			if t.isSecretReference(varValue, result.Outputs) {
				continue
			}
			resolved := t.resolveVariableValue(varValue, result.Outputs)
			data[varName] = resolved
		}
	}

	if len(data) == 0 {
		return nil
	}

	return map[string]interface{}{
		"apiVersion": "v1",
		"kind":       "ConfigMap",
		"metadata": map[string]interface{}{
			"name":   workload.Metadata.Name + "-config",
			"labels": t.generateLabels(workload),
		},
		"data": data,
	}
}

func (t *Translator) generateK8sSecret(workload *Workload, result *TranslationResult) map[string]interface{} {
	data := make(map[string]string)

	for _, container := range workload.Containers {
		for varName, varValue := range container.Variables {
			// Only include secret references
			if !t.isSecretReference(varValue, result.Outputs) {
				continue
			}
			resolved := t.resolveVariableValue(varValue, result.Outputs)
			data[varName] = resolved
		}
	}

	if len(data) == 0 {
		return nil
	}

	return map[string]interface{}{
		"apiVersion": "v1",
		"kind":       "Secret",
		"metadata": map[string]interface{}{
			"name":   workload.Metadata.Name + "-secret",
			"labels": t.generateLabels(workload),
		},
		"type":       "Opaque",
		"stringData": data,
	}
}

func (t *Translator) translateToCompose(workload *Workload, result *TranslationResult) error {
	services := make(map[string]interface{})

	for name, container := range workload.Containers {
		service := map[string]interface{}{
			"image": container.Image,
		}

		if len(container.Command) > 0 {
			service["command"] = container.Command
		}

		// Environment
		env := make([]string, 0)
		for varName, varValue := range container.Variables {
			resolved := t.resolveVariableValue(varValue, result.Outputs)
			env = append(env, fmt.Sprintf("%s=%s", varName, resolved))
		}
		if len(env) > 0 {
			service["environment"] = env
		}

		// Volumes
		if len(container.Volumes) > 0 {
			volumes := make([]string, 0)
			for _, vol := range container.Volumes {
				mount := fmt.Sprintf("%s:%s", vol.Source, vol.Target)
				if vol.ReadOnly {
					mount += ":ro"
				}
				volumes = append(volumes, mount)
			}
			service["volumes"] = volumes
		}

		services[name] = service
	}

	// Add resource services (databases, caches, etc.)
	for name, resource := range workload.Resources {
		if resourceService := t.resourceToComposeService(name, &resource); resourceService != nil {
			services[name] = resourceService
		}
	}

	result.Manifests["docker-compose"] = map[string]interface{}{
		"version":  "3.8",
		"services": services,
	}

	return nil
}

func (t *Translator) resourceToComposeService(name string, resource *Resource) map[string]interface{} {
	switch resource.Type {
	case "postgres":
		version := "15"
		if v, ok := resource.Params["version"].(string); ok {
			version = v
		}
		return map[string]interface{}{
			"image": fmt.Sprintf("postgres:%s", version),
			"environment": []string{
				"POSTGRES_DB=" + name,
				"POSTGRES_USER=postgres",
				"POSTGRES_PASSWORD=postgres",
			},
		}
	case "redis":
		return map[string]interface{}{
			"image": "redis:7-alpine",
		}
	case "mysql":
		version := "8.0"
		if v, ok := resource.Params["version"].(string); ok {
			version = v
		}
		return map[string]interface{}{
			"image": fmt.Sprintf("mysql:%s", version),
			"environment": []string{
				"MYSQL_DATABASE=" + name,
				"MYSQL_ROOT_PASSWORD=root",
			},
		}
	case "mongodb":
		return map[string]interface{}{
			"image": "mongo:6",
		}
	case "rabbitmq":
		return map[string]interface{}{
			"image": "rabbitmq:3-management",
		}
	}
	return nil
}

func (t *Translator) translateToHelm(workload *Workload, result *TranslationResult) error {
	// Generate values.yaml
	values := map[string]interface{}{
		"replicaCount": 1,
		"image": map[string]interface{}{
			"pullPolicy": "IfNotPresent",
		},
		"service": map[string]interface{}{
			"type": "ClusterIP",
		},
		"resources": map[string]interface{}{},
	}

	// Get first container's image (primary container)
	for _, container := range workload.Containers {
		parts := strings.Split(container.Image, ":")
		repo := parts[0]
		tag := "latest"
		if len(parts) > 1 {
			tag = parts[1]
		}
		values["image"].(map[string]interface{})["repository"] = repo
		values["image"].(map[string]interface{})["tag"] = tag

		if container.Resources != nil && container.Resources.Requests != nil {
			values["resources"].(map[string]interface{})["requests"] = map[string]string{
				"cpu":    container.Resources.Requests.CPU,
				"memory": container.Resources.Requests.Memory,
			}
		}
		break // Use first container
	}

	// Service ports
	if workload.Service != nil && len(workload.Service.Ports) > 0 {
		values["service"].(map[string]interface{})["port"] = workload.Service.Ports[0].Port
	}

	result.Manifests["values.yaml"] = values

	return nil
}

func (t *Translator) generateLabels(workload *Workload) map[string]string {
	labels := map[string]string{
		"app":                          workload.Metadata.Name,
		"app.kubernetes.io/name":       workload.Metadata.Name,
		"app.kubernetes.io/managed-by": "platformfoundry",
	}

	for k, v := range workload.Metadata.Labels {
		labels[k] = v
	}

	return labels
}

func (t *Translator) resolveEnvVars(variables map[string]string, outputs map[string]map[string]string) []map[string]interface{} {
	envVars := make([]map[string]interface{}, 0)

	for name, value := range variables {
		resolved := t.resolveVariableValue(value, outputs)
		envVars = append(envVars, map[string]interface{}{
			"name":  name,
			"value": resolved,
		})
	}

	return envVars
}

var resourceRefPattern = regexp.MustCompile(`\$\{resources\.([^.]+)\.([^}]+)\}`)

func (t *Translator) resolveVariableValue(value string, outputs map[string]map[string]string) string {
	return resourceRefPattern.ReplaceAllStringFunc(value, func(match string) string {
		parts := resourceRefPattern.FindStringSubmatch(match)
		if len(parts) != 3 {
			return match
		}
		resourceName := parts[1]
		outputName := parts[2]

		if resourceOutputs, ok := outputs[resourceName]; ok {
			if outputValue, ok := resourceOutputs[outputName]; ok {
				return outputValue
			}
		}
		return match
	})
}

func (t *Translator) isSecretReference(value string, outputs map[string]map[string]string) bool {
	matches := resourceRefPattern.FindAllStringSubmatch(value, -1)
	for _, match := range matches {
		if len(match) != 3 {
			continue
		}
		resourceName := match[1]
		outputName := match[2]

		// Check if the referenced output is marked as secret
		if rt, ok := t.parser.resourceTypes[resourceName]; ok {
			if output, ok := rt.Outputs[outputName]; ok && output.Secret {
				return true
			}
		}

		// Common secret output names
		secretOutputs := []string{"password", "secret", "token", "key", "credential", "connection_string"}
		for _, secretOutput := range secretOutputs {
			if strings.Contains(strings.ToLower(outputName), secretOutput) {
				return true
			}
		}
	}
	return false
}
