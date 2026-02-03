package workload

import (
	"fmt"
	"strings"

	"github.com/platformfoundry/pf-ce/pkg/types"
)

// TranslationResult contains the results of translating a workload
type TranslationResult struct {
	// Kubernetes resources
	Deployment        *DeploymentSpec        `json:"deployment,omitempty"`
	Service           *ServiceSpec           `json:"service,omitempty"`
	HPA               *HPASpec               `json:"hpa,omitempty"`
	Ingress           *IngressResourceSpec   `json:"ingress,omitempty"`
	ConfigMaps        []ConfigMapSpec        `json:"configMaps,omitempty"`
	Secrets           []SecretSpec           `json:"secrets,omitempty"`

	// Infrastructure resources
	InfraResources    []InfraResource        `json:"infraResources,omitempty"`

	// Outputs that will be available after provisioning
	Outputs           map[string]OutputSpec  `json:"outputs,omitempty"`
}

// DeploymentSpec represents a Kubernetes Deployment
type DeploymentSpec struct {
	Name        string                 `json:"name"`
	Namespace   string                 `json:"namespace"`
	Labels      map[string]string      `json:"labels"`
	Annotations map[string]string      `json:"annotations,omitempty"`
	Replicas    int                    `json:"replicas"`
	Containers  []ContainerSpec        `json:"containers"`
	Volumes     []VolumeSpec           `json:"volumes,omitempty"`
}

// ContainerSpec represents a container in a deployment
type ContainerSpec struct {
	Name           string                 `json:"name"`
	Image          string                 `json:"image"`
	Command        []string               `json:"command,omitempty"`
	Args           []string               `json:"args,omitempty"`
	Env            []EnvVarSpec           `json:"env,omitempty"`
	EnvFrom        []EnvFromSpec          `json:"envFrom,omitempty"`
	Resources      ResourceSpec           `json:"resources,omitempty"`
	Ports          []ContainerPortSpec    `json:"ports,omitempty"`
	LivenessProbe  *ProbeSpec             `json:"livenessProbe,omitempty"`
	ReadinessProbe *ProbeSpec             `json:"readinessProbe,omitempty"`
	VolumeMounts   []VolumeMountSpec      `json:"volumeMounts,omitempty"`
}

// EnvVarSpec represents an environment variable
type EnvVarSpec struct {
	Name      string          `json:"name"`
	Value     string          `json:"value,omitempty"`
	ValueFrom *EnvVarSourceSpec `json:"valueFrom,omitempty"`
}

// EnvVarSourceSpec represents the source of an environment variable
type EnvVarSourceSpec struct {
	SecretKeyRef    *SecretKeyRefSpec    `json:"secretKeyRef,omitempty"`
	ConfigMapKeyRef *ConfigMapKeyRefSpec `json:"configMapKeyRef,omitempty"`
}

// SecretKeyRefSpec references a key in a secret
type SecretKeyRefSpec struct {
	Name string `json:"name"`
	Key  string `json:"key"`
}

// ConfigMapKeyRefSpec references a key in a configmap
type ConfigMapKeyRefSpec struct {
	Name string `json:"name"`
	Key  string `json:"key"`
}

// EnvFromSpec represents envFrom in a container
type EnvFromSpec struct {
	SecretRef    string `json:"secretRef,omitempty"`
	ConfigMapRef string `json:"configMapRef,omitempty"`
}

// ResourceSpec represents container resource requirements
type ResourceSpec struct {
	Requests ResourceQuantitySpec `json:"requests,omitempty"`
	Limits   ResourceQuantitySpec `json:"limits,omitempty"`
}

// ResourceQuantitySpec represents resource quantities
type ResourceQuantitySpec struct {
	CPU    string `json:"cpu,omitempty"`
	Memory string `json:"memory,omitempty"`
}

// ContainerPortSpec represents a container port
type ContainerPortSpec struct {
	Name          string `json:"name,omitempty"`
	ContainerPort int    `json:"containerPort"`
	Protocol      string `json:"protocol,omitempty"`
}

// ProbeSpec represents a health probe
type ProbeSpec struct {
	HTTPGet             *HTTPGetProbeSpec `json:"httpGet,omitempty"`
	TCPSocket           *TCPSocketProbeSpec `json:"tcpSocket,omitempty"`
	InitialDelaySeconds int               `json:"initialDelaySeconds,omitempty"`
	PeriodSeconds       int               `json:"periodSeconds,omitempty"`
	TimeoutSeconds      int               `json:"timeoutSeconds,omitempty"`
	FailureThreshold    int               `json:"failureThreshold,omitempty"`
}

// HTTPGetProbeSpec represents an HTTP GET probe
type HTTPGetProbeSpec struct {
	Path   string `json:"path"`
	Port   int    `json:"port"`
	Scheme string `json:"scheme,omitempty"`
}

// TCPSocketProbeSpec represents a TCP socket probe
type TCPSocketProbeSpec struct {
	Port int `json:"port"`
}

// VolumeMountSpec represents a volume mount
type VolumeMountSpec struct {
	Name      string `json:"name"`
	MountPath string `json:"mountPath"`
	ReadOnly  bool   `json:"readOnly,omitempty"`
}

// VolumeSpec represents a volume
type VolumeSpec struct {
	Name      string           `json:"name"`
	Secret    *SecretVolumeSpec `json:"secret,omitempty"`
	ConfigMap *ConfigMapVolumeSpec `json:"configMap,omitempty"`
	EmptyDir  *EmptyDirVolumeSpec `json:"emptyDir,omitempty"`
}

// SecretVolumeSpec represents a secret volume
type SecretVolumeSpec struct {
	SecretName string `json:"secretName"`
}

// ConfigMapVolumeSpec represents a configmap volume
type ConfigMapVolumeSpec struct {
	Name string `json:"name"`
}

// EmptyDirVolumeSpec represents an emptyDir volume
type EmptyDirVolumeSpec struct {
	Medium string `json:"medium,omitempty"`
}

// ServiceSpec represents a Kubernetes Service
type ServiceSpec struct {
	Name        string            `json:"name"`
	Namespace   string            `json:"namespace"`
	Labels      map[string]string `json:"labels"`
	Type        string            `json:"type"`
	Ports       []ServicePortSpec `json:"ports"`
	Selector    map[string]string `json:"selector"`
}

// ServicePortSpec represents a service port
type ServicePortSpec struct {
	Name       string `json:"name,omitempty"`
	Port       int    `json:"port"`
	TargetPort int    `json:"targetPort"`
	Protocol   string `json:"protocol,omitempty"`
}

// HPASpec represents a HorizontalPodAutoscaler
type HPASpec struct {
	Name                     string `json:"name"`
	Namespace                string `json:"namespace"`
	MinReplicas              int    `json:"minReplicas"`
	MaxReplicas              int    `json:"maxReplicas"`
	TargetCPUUtilization     int    `json:"targetCPUUtilization,omitempty"`
	TargetMemoryUtilization  int    `json:"targetMemoryUtilization,omitempty"`
}

// IngressResourceSpec represents a Kubernetes Ingress
type IngressResourceSpec struct {
	Name        string            `json:"name"`
	Namespace   string            `json:"namespace"`
	Labels      map[string]string `json:"labels"`
	Annotations map[string]string `json:"annotations,omitempty"`
	TLS         bool              `json:"tls"`
	Host        string            `json:"host,omitempty"`
	Path        string            `json:"path"`
	ServiceName string            `json:"serviceName"`
	ServicePort int               `json:"servicePort"`
}

// ConfigMapSpec represents a Kubernetes ConfigMap
type ConfigMapSpec struct {
	Name      string            `json:"name"`
	Namespace string            `json:"namespace"`
	Data      map[string]string `json:"data"`
}

// SecretSpec represents a Kubernetes Secret
type SecretSpec struct {
	Name      string            `json:"name"`
	Namespace string            `json:"namespace"`
	Type      string            `json:"type"`
	Data      map[string]string `json:"data"`
}

// InfraResource represents an infrastructure resource to provision
type InfraResource struct {
	Type     string                 `json:"type"`     // terraform-aws-rds, terraform-aws-elasticache, etc.
	Name     string                 `json:"name"`
	Provider string                 `json:"provider"` // terraform, pulumi, etc.
	Config   map[string]interface{} `json:"config"`
}

// OutputSpec represents an output value
type OutputSpec struct {
	Type        string `json:"type"`        // string, secret, number
	Description string `json:"description"`
	Value       string `json:"value,omitempty"`
}

// Translator translates Workload specs to platform resources
type Translator struct {
	defaultMappings map[string]DependencyMapping
	cloudProvider   string
	region          string
	namespace       string
}

// DependencyMapping defines how a dependency type maps to infrastructure
type DependencyMapping struct {
	Provider       string                 `json:"provider"`       // terraform, pulumi
	ResourceType   string                 `json:"resourceType"`   // aws-rds, aws-elasticache
	DefaultConfig  map[string]interface{} `json:"defaultConfig"`
	SizeMapping    map[string]SizeSpec    `json:"sizeMapping"`
}

// SizeSpec defines resource specifications for a size tier
type SizeSpec struct {
	InstanceType string `json:"instanceType"`
	Storage      int    `json:"storage,omitempty"`
	Memory       string `json:"memory,omitempty"`
}

// NewTranslator creates a new workload translator
func NewTranslator(cloudProvider, region, namespace string) *Translator {
	t := &Translator{
		cloudProvider:   cloudProvider,
		region:          region,
		namespace:       namespace,
		defaultMappings: make(map[string]DependencyMapping),
	}
	t.initDefaultMappings()
	return t
}

// initDefaultMappings initializes default dependency mappings
func (t *Translator) initDefaultMappings() {
	t.defaultMappings = map[string]DependencyMapping{
		"postgres": {
			Provider:     "terraform",
			ResourceType: "aws-rds-postgres",
			DefaultConfig: map[string]interface{}{
				"engine":              "postgres",
				"engine_version":      "15",
				"allocated_storage":   20,
				"backup_retention":    7,
				"multi_az":            false,
				"skip_final_snapshot": true,
			},
			SizeMapping: map[string]SizeSpec{
				"small":  {InstanceType: "db.t3.small", Storage: 20},
				"medium": {InstanceType: "db.t3.medium", Storage: 50},
				"large":  {InstanceType: "db.r5.large", Storage: 100},
				"xlarge": {InstanceType: "db.r5.xlarge", Storage: 200},
			},
		},
		"mysql": {
			Provider:     "terraform",
			ResourceType: "aws-rds-mysql",
			DefaultConfig: map[string]interface{}{
				"engine":              "mysql",
				"engine_version":      "8.0",
				"allocated_storage":   20,
				"backup_retention":    7,
				"multi_az":            false,
				"skip_final_snapshot": true,
			},
			SizeMapping: map[string]SizeSpec{
				"small":  {InstanceType: "db.t3.small", Storage: 20},
				"medium": {InstanceType: "db.t3.medium", Storage: 50},
				"large":  {InstanceType: "db.r5.large", Storage: 100},
				"xlarge": {InstanceType: "db.r5.xlarge", Storage: 200},
			},
		},
		"redis": {
			Provider:     "terraform",
			ResourceType: "aws-elasticache-redis",
			DefaultConfig: map[string]interface{}{
				"engine":               "redis",
				"engine_version":       "7.0",
				"num_cache_clusters":   1,
				"automatic_failover":   false,
			},
			SizeMapping: map[string]SizeSpec{
				"small":  {InstanceType: "cache.t3.small"},
				"medium": {InstanceType: "cache.t3.medium"},
				"large":  {InstanceType: "cache.r5.large"},
				"xlarge": {InstanceType: "cache.r5.xlarge"},
			},
		},
		"mongodb": {
			Provider:     "terraform",
			ResourceType: "aws-documentdb",
			DefaultConfig: map[string]interface{}{
				"engine":         "docdb",
				"engine_version": "6.0",
			},
			SizeMapping: map[string]SizeSpec{
				"small":  {InstanceType: "db.t3.medium"},
				"medium": {InstanceType: "db.r5.large"},
				"large":  {InstanceType: "db.r5.xlarge"},
				"xlarge": {InstanceType: "db.r5.2xlarge"},
			},
		},
		"s3": {
			Provider:     "terraform",
			ResourceType: "aws-s3",
			DefaultConfig: map[string]interface{}{
				"versioning": false,
				"encryption": true,
			},
			SizeMapping: map[string]SizeSpec{},
		},
		"kafka": {
			Provider:     "terraform",
			ResourceType: "aws-msk",
			DefaultConfig: map[string]interface{}{
				"kafka_version":     "3.5.1",
				"number_of_nodes":   3,
				"ebs_volume_size":   100,
			},
			SizeMapping: map[string]SizeSpec{
				"small":  {InstanceType: "kafka.t3.small"},
				"medium": {InstanceType: "kafka.m5.large"},
				"large":  {InstanceType: "kafka.m5.xlarge"},
				"xlarge": {InstanceType: "kafka.m5.2xlarge"},
			},
		},
		"rabbitmq": {
			Provider:     "terraform",
			ResourceType: "aws-mq-rabbitmq",
			DefaultConfig: map[string]interface{}{
				"engine_version":    "3.12",
				"deployment_mode":   "SINGLE_INSTANCE",
			},
			SizeMapping: map[string]SizeSpec{
				"small":  {InstanceType: "mq.t3.micro"},
				"medium": {InstanceType: "mq.m5.large"},
				"large":  {InstanceType: "mq.m5.xlarge"},
				"xlarge": {InstanceType: "mq.m5.2xlarge"},
			},
		},
		"elasticsearch": {
			Provider:     "terraform",
			ResourceType: "aws-opensearch",
			DefaultConfig: map[string]interface{}{
				"engine_version": "OpenSearch_2.11",
				"instance_count": 1,
			},
			SizeMapping: map[string]SizeSpec{
				"small":  {InstanceType: "t3.small.search", Storage: 10},
				"medium": {InstanceType: "t3.medium.search", Storage: 50},
				"large":  {InstanceType: "r5.large.search", Storage: 100},
				"xlarge": {InstanceType: "r5.xlarge.search", Storage: 200},
			},
		},
		"memcached": {
			Provider:     "terraform",
			ResourceType: "aws-elasticache-memcached",
			DefaultConfig: map[string]interface{}{
				"engine":             "memcached",
				"engine_version":     "1.6.17",
				"num_cache_nodes":    1,
			},
			SizeMapping: map[string]SizeSpec{
				"small":  {InstanceType: "cache.t3.small"},
				"medium": {InstanceType: "cache.t3.medium"},
				"large":  {InstanceType: "cache.r5.large"},
				"xlarge": {InstanceType: "cache.r5.xlarge"},
			},
		},
		"dynamodb": {
			Provider:     "terraform",
			ResourceType: "aws-dynamodb",
			DefaultConfig: map[string]interface{}{
				"billing_mode": "PAY_PER_REQUEST",
			},
			SizeMapping: map[string]SizeSpec{},
		},
	}
}

// Translate converts a Workload to platform resources
func (t *Translator) Translate(w *types.Workload) (*TranslationResult, error) {
	if err := w.Validate(); err != nil {
		return nil, fmt.Errorf("invalid workload: %w", err)
	}

	result := &TranslationResult{
		Outputs: make(map[string]OutputSpec),
	}

	// Determine namespace
	namespace := t.namespace
	if w.Metadata.Environment != "" {
		namespace = w.Metadata.Environment
	}

	// Create labels
	labels := map[string]string{
		"app":                          w.Metadata.Name,
		"team":                         w.Metadata.Team,
		"platformfoundry.io/workload":  w.Metadata.Name,
		"platformfoundry.io/managed":   "true",
	}
	for k, v := range w.Metadata.Labels {
		labels[k] = v
	}

	// Translate containers to deployment
	deployment, err := t.translateDeployment(w, namespace, labels)
	if err != nil {
		return nil, fmt.Errorf("failed to translate deployment: %w", err)
	}
	result.Deployment = deployment

	// Translate service
	if len(w.Spec.Containers) > 0 && hasExposedPorts(w.Spec.Containers) {
		service, err := t.translateService(w, namespace, labels)
		if err != nil {
			return nil, fmt.Errorf("failed to translate service: %w", err)
		}
		result.Service = service
	}

	// Translate HPA if scaling is defined
	if w.Spec.Scaling != nil {
		result.HPA = t.translateHPA(w, namespace)
	}

	// Translate ingress if network.ingress is defined
	if w.Spec.Network != nil && w.Spec.Network.Ingress != nil && w.Spec.Network.Ingress.Enabled {
		ingress, err := t.translateIngress(w, namespace, labels)
		if err != nil {
			return nil, fmt.Errorf("failed to translate ingress: %w", err)
		}
		result.Ingress = ingress
	}

	// Translate dependencies to infrastructure resources
	for _, dep := range w.Spec.Dependencies {
		infraRes, err := t.translateDependency(dep, w.Metadata.Name)
		if err != nil {
			return nil, fmt.Errorf("failed to translate dependency %s: %w", dep.Name, err)
		}
		result.InfraResources = append(result.InfraResources, infraRes)

		// Add outputs for the dependency
		t.addDependencyOutputs(result, dep)
	}

	return result, nil
}

// translateDeployment creates a Kubernetes Deployment spec
func (t *Translator) translateDeployment(w *types.Workload, namespace string, labels map[string]string) (*DeploymentSpec, error) {
	replicas := 1
	if w.Spec.Scaling != nil {
		replicas = w.Spec.Scaling.Min
	}

	deployment := &DeploymentSpec{
		Name:        w.Metadata.Name,
		Namespace:   namespace,
		Labels:      labels,
		Annotations: w.Metadata.Annotations,
		Replicas:    replicas,
		Containers:  make([]ContainerSpec, 0, len(w.Spec.Containers)),
	}

	for _, c := range w.Spec.Containers {
		container := ContainerSpec{
			Name:    c.Name,
			Image:   c.Image,
			Command: c.Command,
			Args:    c.Args,
		}

		// Translate environment variables
		for name, value := range c.Env {
			container.Env = append(container.Env, EnvVarSpec{
				Name:  name,
				Value: value,
			})
		}

		// Translate envFrom
		for _, ef := range c.EnvFrom {
			container.EnvFrom = append(container.EnvFrom, EnvFromSpec{
				SecretRef:    ef.SecretRef,
				ConfigMapRef: ef.ConfigMapRef,
			})
		}

		// Translate resources
		if c.Resources != nil {
			container.Resources = ResourceSpec{
				Requests: ResourceQuantitySpec{
					CPU:    c.Resources.CPU,
					Memory: c.Resources.Memory,
				},
				Limits: ResourceQuantitySpec{
					CPU:    c.Resources.CPU,
					Memory: c.Resources.Memory,
				},
			}
		}

		// Translate ports
		for _, p := range c.Ports {
			port := p.Port
			if p.ContainerPort > 0 {
				port = p.ContainerPort
			}
			container.Ports = append(container.Ports, ContainerPortSpec{
				Name:          p.Name,
				ContainerPort: port,
				Protocol:      coalesce(p.Protocol, "TCP"),
			})
		}

		// Translate probes
		if c.LivenessProbe != nil {
			container.LivenessProbe = t.translateProbe(c.LivenessProbe)
		}
		if c.ReadinessProbe != nil {
			container.ReadinessProbe = t.translateProbe(c.ReadinessProbe)
		}

		// Translate volume mounts
		for _, vm := range c.VolumeMounts {
			container.VolumeMounts = append(container.VolumeMounts, VolumeMountSpec{
				Name:      vm.Name,
				MountPath: vm.MountPath,
				ReadOnly:  vm.ReadOnly,
			})
		}

		deployment.Containers = append(deployment.Containers, container)
	}

	return deployment, nil
}

// translateProbe converts a types.ProbeSpec to ProbeSpec
func (t *Translator) translateProbe(p *types.ProbeSpec) *ProbeSpec {
	probe := &ProbeSpec{
		InitialDelaySeconds: p.InitialDelaySeconds,
		PeriodSeconds:       p.PeriodSeconds,
		TimeoutSeconds:      p.TimeoutSeconds,
		FailureThreshold:    p.FailureThreshold,
	}

	if p.HTTPGet != nil {
		probe.HTTPGet = &HTTPGetProbeSpec{
			Path:   p.HTTPGet.Path,
			Port:   p.HTTPGet.Port,
			Scheme: coalesce(p.HTTPGet.Scheme, "HTTP"),
		}
	}

	if p.TCPSocket != nil {
		probe.TCPSocket = &TCPSocketProbeSpec{
			Port: p.TCPSocket.Port,
		}
	}

	return probe
}

// translateService creates a Kubernetes Service spec
func (t *Translator) translateService(w *types.Workload, namespace string, labels map[string]string) (*ServiceSpec, error) {
	service := &ServiceSpec{
		Name:      w.Metadata.Name,
		Namespace: namespace,
		Labels:    labels,
		Type:      "ClusterIP",
		Selector:  labels,
		Ports:     make([]ServicePortSpec, 0),
	}

	if w.Spec.Network != nil && w.Spec.Network.Service != nil {
		service.Type = coalesce(w.Spec.Network.Service.Type, "ClusterIP")
	}

	// Collect all ports from containers
	for _, c := range w.Spec.Containers {
		for _, p := range c.Ports {
			port := p.Port
			if p.ContainerPort > 0 {
				port = p.ContainerPort
			}
			service.Ports = append(service.Ports, ServicePortSpec{
				Name:       p.Name,
				Port:       p.Port,
				TargetPort: port,
				Protocol:   coalesce(p.Protocol, "TCP"),
			})
		}
	}

	return service, nil
}

// translateHPA creates a HorizontalPodAutoscaler spec
func (t *Translator) translateHPA(w *types.Workload, namespace string) *HPASpec {
	hpa := &HPASpec{
		Name:        w.Metadata.Name,
		Namespace:   namespace,
		MinReplicas: w.Spec.Scaling.Min,
		MaxReplicas: w.Spec.Scaling.Max,
	}

	if w.Spec.Scaling.TargetCPU > 0 {
		hpa.TargetCPUUtilization = w.Spec.Scaling.TargetCPU
	}
	if w.Spec.Scaling.TargetMem > 0 {
		hpa.TargetMemoryUtilization = w.Spec.Scaling.TargetMem
	}

	return hpa
}

// translateIngress creates a Kubernetes Ingress spec
func (t *Translator) translateIngress(w *types.Workload, namespace string, labels map[string]string) (*IngressResourceSpec, error) {
	ing := w.Spec.Network.Ingress

	// Find the first service port
	servicePort := 80
	if len(w.Spec.Containers) > 0 && len(w.Spec.Containers[0].Ports) > 0 {
		servicePort = w.Spec.Containers[0].Ports[0].Port
	}

	ingress := &IngressResourceSpec{
		Name:        w.Metadata.Name,
		Namespace:   namespace,
		Labels:      labels,
		Annotations: ing.Annotations,
		TLS:         ing.TLS,
		Host:        ing.Host,
		Path:        coalesce(ing.Path, "/"),
		ServiceName: w.Metadata.Name,
		ServicePort: servicePort,
	}

	return ingress, nil
}

// translateDependency creates an infrastructure resource for a dependency
func (t *Translator) translateDependency(dep types.WorkloadDependency, workloadName string) (InfraResource, error) {
	mapping, ok := t.defaultMappings[dep.Type]
	if !ok {
		return InfraResource{}, fmt.Errorf("unsupported dependency type: %s", dep.Type)
	}

	// Start with default config
	config := make(map[string]interface{})
	for k, v := range mapping.DefaultConfig {
		config[k] = v
	}

	// Apply user-specified config
	for k, v := range dep.Config {
		config[k] = v
	}

	// Apply size mapping if size is specified
	if size, ok := dep.Config["size"].(string); ok {
		if sizeSpec, ok := mapping.SizeMapping[size]; ok {
			config["instance_type"] = sizeSpec.InstanceType
			if sizeSpec.Storage > 0 {
				config["allocated_storage"] = sizeSpec.Storage
			}
		}
	}

	// Add naming and tagging
	resourceName := fmt.Sprintf("%s-%s", workloadName, dep.Name)
	config["name"] = resourceName
	config["tags"] = map[string]string{
		"workload":   workloadName,
		"dependency": dep.Name,
		"managed-by": "platformfoundry",
	}

	return InfraResource{
		Type:     mapping.ResourceType,
		Name:     dep.Name,
		Provider: mapping.Provider,
		Config:   config,
	}, nil
}

// addDependencyOutputs adds output specifications for a dependency
func (t *Translator) addDependencyOutputs(result *TranslationResult, dep types.WorkloadDependency) {
	prefix := strings.ToUpper(strings.ReplaceAll(dep.Name, "-", "_"))

	switch dep.Type {
	case "postgres", "mysql":
		result.Outputs[prefix+"_HOST"] = OutputSpec{
			Type:        "string",
			Description: fmt.Sprintf("%s database host", dep.Name),
		}
		result.Outputs[prefix+"_PORT"] = OutputSpec{
			Type:        "number",
			Description: fmt.Sprintf("%s database port", dep.Name),
		}
		result.Outputs[prefix+"_CONNECTION_STRING"] = OutputSpec{
			Type:        "secret",
			Description: fmt.Sprintf("%s connection string", dep.Name),
		}
	case "redis", "memcached":
		result.Outputs[prefix+"_HOST"] = OutputSpec{
			Type:        "string",
			Description: fmt.Sprintf("%s cache host", dep.Name),
		}
		result.Outputs[prefix+"_PORT"] = OutputSpec{
			Type:        "number",
			Description: fmt.Sprintf("%s cache port", dep.Name),
		}
	case "s3":
		result.Outputs[prefix+"_BUCKET"] = OutputSpec{
			Type:        "string",
			Description: fmt.Sprintf("%s bucket name", dep.Name),
		}
		result.Outputs[prefix+"_ARN"] = OutputSpec{
			Type:        "string",
			Description: fmt.Sprintf("%s bucket ARN", dep.Name),
		}
	case "kafka":
		result.Outputs[prefix+"_BOOTSTRAP_SERVERS"] = OutputSpec{
			Type:        "string",
			Description: fmt.Sprintf("%s bootstrap servers", dep.Name),
		}
	case "elasticsearch":
		result.Outputs[prefix+"_ENDPOINT"] = OutputSpec{
			Type:        "string",
			Description: fmt.Sprintf("%s endpoint", dep.Name),
		}
	case "dynamodb":
		result.Outputs[prefix+"_TABLE_NAME"] = OutputSpec{
			Type:        "string",
			Description: fmt.Sprintf("%s table name", dep.Name),
		}
		result.Outputs[prefix+"_TABLE_ARN"] = OutputSpec{
			Type:        "string",
			Description: fmt.Sprintf("%s table ARN", dep.Name),
		}
	}
}

// hasExposedPorts checks if any container has exposed ports
func hasExposedPorts(containers []types.Container) bool {
	for _, c := range containers {
		if len(c.Ports) > 0 {
			return true
		}
	}
	return false
}

// coalesce returns the first non-empty string
func coalesce(values ...string) string {
	for _, v := range values {
		if v != "" {
			return v
		}
	}
	return ""
}

// ToKubernetesYAML generates Kubernetes manifests from the translation result
func (r *TranslationResult) ToKubernetesYAML() (string, error) {
	var sb strings.Builder

	// Generate Deployment YAML
	if r.Deployment != nil {
		sb.WriteString(fmt.Sprintf(`apiVersion: apps/v1
kind: Deployment
metadata:
  name: %s
  namespace: %s
  labels:
`, r.Deployment.Name, r.Deployment.Namespace))
		for k, v := range r.Deployment.Labels {
			sb.WriteString(fmt.Sprintf("    %s: %s\n", k, v))
		}
		sb.WriteString(fmt.Sprintf(`spec:
  replicas: %d
  selector:
    matchLabels:
      app: %s
  template:
    metadata:
      labels:
`, r.Deployment.Replicas, r.Deployment.Name))
		for k, v := range r.Deployment.Labels {
			sb.WriteString(fmt.Sprintf("        %s: %s\n", k, v))
		}
		sb.WriteString("    spec:\n      containers:\n")
		for _, c := range r.Deployment.Containers {
			sb.WriteString(fmt.Sprintf("      - name: %s\n        image: %s\n", c.Name, c.Image))
			if c.Resources.Requests.CPU != "" || c.Resources.Requests.Memory != "" {
				sb.WriteString("        resources:\n          requests:\n")
				if c.Resources.Requests.CPU != "" {
					sb.WriteString(fmt.Sprintf("            cpu: %s\n", c.Resources.Requests.CPU))
				}
				if c.Resources.Requests.Memory != "" {
					sb.WriteString(fmt.Sprintf("            memory: %s\n", c.Resources.Requests.Memory))
				}
			}
			if len(c.Ports) > 0 {
				sb.WriteString("        ports:\n")
				for _, p := range c.Ports {
					sb.WriteString(fmt.Sprintf("        - containerPort: %d\n", p.ContainerPort))
					if p.Name != "" {
						sb.WriteString(fmt.Sprintf("          name: %s\n", p.Name))
					}
				}
			}
		}
		sb.WriteString("---\n")
	}

	// Generate Service YAML
	if r.Service != nil {
		sb.WriteString(fmt.Sprintf(`apiVersion: v1
kind: Service
metadata:
  name: %s
  namespace: %s
spec:
  type: %s
  selector:
    app: %s
  ports:
`, r.Service.Name, r.Service.Namespace, r.Service.Type, r.Service.Name))
		for _, p := range r.Service.Ports {
			sb.WriteString(fmt.Sprintf("  - port: %d\n    targetPort: %d\n", p.Port, p.TargetPort))
			if p.Name != "" {
				sb.WriteString(fmt.Sprintf("    name: %s\n", p.Name))
			}
		}
		sb.WriteString("---\n")
	}

	// Generate HPA YAML
	if r.HPA != nil {
		sb.WriteString(fmt.Sprintf(`apiVersion: autoscaling/v2
kind: HorizontalPodAutoscaler
metadata:
  name: %s
  namespace: %s
spec:
  scaleTargetRef:
    apiVersion: apps/v1
    kind: Deployment
    name: %s
  minReplicas: %d
  maxReplicas: %d
  metrics:
`, r.HPA.Name, r.HPA.Namespace, r.HPA.Name, r.HPA.MinReplicas, r.HPA.MaxReplicas))
		if r.HPA.TargetCPUUtilization > 0 {
			sb.WriteString(fmt.Sprintf(`  - type: Resource
    resource:
      name: cpu
      target:
        type: Utilization
        averageUtilization: %d
`, r.HPA.TargetCPUUtilization))
		}
		sb.WriteString("---\n")
	}

	// Generate Ingress YAML
	if r.Ingress != nil {
		sb.WriteString(fmt.Sprintf(`apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  name: %s
  namespace: %s
`, r.Ingress.Name, r.Ingress.Namespace))
		if len(r.Ingress.Annotations) > 0 {
			sb.WriteString("  annotations:\n")
			for k, v := range r.Ingress.Annotations {
				sb.WriteString(fmt.Sprintf("    %s: %s\n", k, v))
			}
		}
		sb.WriteString(`spec:
  rules:
`)
		if r.Ingress.Host != "" {
			sb.WriteString(fmt.Sprintf("  - host: %s\n    http:\n", r.Ingress.Host))
		} else {
			sb.WriteString("  - http:\n")
		}
		sb.WriteString(fmt.Sprintf(`      paths:
      - path: %s
        pathType: Prefix
        backend:
          service:
            name: %s
            port:
              number: %d
`, r.Ingress.Path, r.Ingress.ServiceName, r.Ingress.ServicePort))
		if r.Ingress.TLS && r.Ingress.Host != "" {
			sb.WriteString(fmt.Sprintf(`  tls:
  - hosts:
    - %s
    secretName: %s-tls
`, r.Ingress.Host, r.Ingress.Name))
		}
	}

	return sb.String(), nil
}
