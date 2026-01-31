package types

import (
	"fmt"
	"regexp"
)

// Workload represents a developer workload specification
// Workloads abstract away infrastructure complexity, allowing developers to define
// what they need without knowing the underlying implementation details
type Workload struct {
	APIVersion string           `yaml:"apiVersion" json:"apiVersion"`
	Kind       string           `yaml:"kind" json:"kind"` // "Workload"
	Metadata   WorkloadMetadata `yaml:"metadata" json:"metadata"`
	Spec       WorkloadSpec     `yaml:"spec" json:"spec"`
	Status     *WorkloadStatus  `yaml:"status,omitempty" json:"status,omitempty"`
}

// WorkloadMetadata contains workload metadata
type WorkloadMetadata struct {
	Name         string            `yaml:"name" json:"name"`
	Team         string            `yaml:"team" json:"team"`
	Labels       map[string]string `yaml:"labels,omitempty" json:"labels,omitempty"`
	Annotations  map[string]string `yaml:"annotations,omitempty" json:"annotations,omitempty"`
	Organization string            `yaml:"organization,omitempty" json:"organization,omitempty"`
	Environment  string            `yaml:"environment,omitempty" json:"environment,omitempty"`
}

// WorkloadSpec defines the workload specification
type WorkloadSpec struct {
	// Containers defines the application containers
	Containers []Container `yaml:"containers" json:"containers"`

	// Dependencies defines external dependencies (databases, caches, queues, etc.)
	Dependencies []WorkloadDependency `yaml:"dependencies,omitempty" json:"dependencies,omitempty"`

	// Scaling defines auto-scaling configuration
	Scaling *ScalingSpec `yaml:"scaling,omitempty" json:"scaling,omitempty"`

	// Network defines networking configuration
	Network *NetworkSpec `yaml:"network,omitempty" json:"network,omitempty"`

	// Environment-specific overrides
	Environments map[string]EnvironmentOverride `yaml:"environments,omitempty" json:"environments,omitempty"`

	// ServiceAccount for workload identity
	ServiceAccount string `yaml:"serviceAccount,omitempty" json:"serviceAccount,omitempty"`

	// Lifecycle hooks
	Lifecycle *LifecycleSpec `yaml:"lifecycle,omitempty" json:"lifecycle,omitempty"`
}

// Container defines a container in the workload
type Container struct {
	Name       string               `yaml:"name" json:"name"`
	Image      string               `yaml:"image" json:"image"`
	Command    []string             `yaml:"command,omitempty" json:"command,omitempty"`
	Args       []string             `yaml:"args,omitempty" json:"args,omitempty"`
	Env        map[string]string    `yaml:"env,omitempty" json:"env,omitempty"`
	EnvFrom    []EnvFromSource      `yaml:"envFrom,omitempty" json:"envFrom,omitempty"`
	Resources  *ContainerResources  `yaml:"resources,omitempty" json:"resources,omitempty"`
	Ports      []PortSpec           `yaml:"ports,omitempty" json:"ports,omitempty"`
	LivenessProbe  *ProbeSpec       `yaml:"livenessProbe,omitempty" json:"livenessProbe,omitempty"`
	ReadinessProbe *ProbeSpec       `yaml:"readinessProbe,omitempty" json:"readinessProbe,omitempty"`
	VolumeMounts   []VolumeMount    `yaml:"volumeMounts,omitempty" json:"volumeMounts,omitempty"`
}

// EnvFromSource defines environment variable sources
type EnvFromSource struct {
	SecretRef    string `yaml:"secretRef,omitempty" json:"secretRef,omitempty"`
	ConfigMapRef string `yaml:"configMapRef,omitempty" json:"configMapRef,omitempty"`
}

// ContainerResources defines resource requirements for a container
type ContainerResources struct {
	CPU    string `yaml:"cpu,omitempty" json:"cpu,omitempty"`       // e.g., "500m", "1"
	Memory string `yaml:"memory,omitempty" json:"memory,omitempty"` // e.g., "512Mi", "1Gi"
}

// PortSpec defines a port specification
type PortSpec struct {
	Name          string `yaml:"name" json:"name"`
	Port          int    `yaml:"port" json:"port"`
	Protocol      string `yaml:"protocol,omitempty" json:"protocol,omitempty"` // TCP, UDP
	ContainerPort int    `yaml:"containerPort,omitempty" json:"containerPort,omitempty"`
}

// ProbeSpec defines health probe configuration
type ProbeSpec struct {
	HTTPGet             *HTTPGetAction `yaml:"httpGet,omitempty" json:"httpGet,omitempty"`
	TCPSocket           *TCPSocketAction `yaml:"tcpSocket,omitempty" json:"tcpSocket,omitempty"`
	InitialDelaySeconds int            `yaml:"initialDelaySeconds,omitempty" json:"initialDelaySeconds,omitempty"`
	PeriodSeconds       int            `yaml:"periodSeconds,omitempty" json:"periodSeconds,omitempty"`
	TimeoutSeconds      int            `yaml:"timeoutSeconds,omitempty" json:"timeoutSeconds,omitempty"`
	FailureThreshold    int            `yaml:"failureThreshold,omitempty" json:"failureThreshold,omitempty"`
}

// HTTPGetAction describes an action based on HTTP Get requests
type HTTPGetAction struct {
	Path   string `yaml:"path" json:"path"`
	Port   int    `yaml:"port" json:"port"`
	Scheme string `yaml:"scheme,omitempty" json:"scheme,omitempty"` // HTTP, HTTPS
}

// TCPSocketAction describes an action based on TCP socket
type TCPSocketAction struct {
	Port int `yaml:"port" json:"port"`
}

// VolumeMount describes a volume mount
type VolumeMount struct {
	Name      string `yaml:"name" json:"name"`
	MountPath string `yaml:"mountPath" json:"mountPath"`
	ReadOnly  bool   `yaml:"readOnly,omitempty" json:"readOnly,omitempty"`
}

// WorkloadDependency defines an external dependency
type WorkloadDependency struct {
	Type     string                 `yaml:"type" json:"type"`     // postgres, redis, s3, kafka, elasticsearch, etc.
	Name     string                 `yaml:"name" json:"name"`     // Logical name for this dependency
	Config   map[string]interface{} `yaml:"config,omitempty" json:"config,omitempty"`
	Required bool                   `yaml:"required,omitempty" json:"required,omitempty"`
}

// ScalingSpec defines auto-scaling configuration
type ScalingSpec struct {
	Min        int `yaml:"min" json:"min"`
	Max        int `yaml:"max" json:"max"`
	TargetCPU  int `yaml:"targetCPU,omitempty" json:"targetCPU,omitempty"`   // Target CPU utilization %
	TargetMem  int `yaml:"targetMem,omitempty" json:"targetMem,omitempty"`   // Target memory utilization %
}

// NetworkSpec defines networking configuration
type NetworkSpec struct {
	// Ingress configuration
	Ingress *IngressSpec `yaml:"ingress,omitempty" json:"ingress,omitempty"`

	// Internal service configuration
	Service *ServiceNetworkSpec `yaml:"service,omitempty" json:"service,omitempty"`

	// Network policies
	Policies []NetworkPolicySpec `yaml:"policies,omitempty" json:"policies,omitempty"`
}

// IngressSpec defines ingress configuration
type IngressSpec struct {
	Enabled     bool              `yaml:"enabled,omitempty" json:"enabled,omitempty"`
	Path        string            `yaml:"path,omitempty" json:"path,omitempty"`
	Host        string            `yaml:"host,omitempty" json:"host,omitempty"`
	TLS         bool              `yaml:"tls,omitempty" json:"tls,omitempty"`
	Annotations map[string]string `yaml:"annotations,omitempty" json:"annotations,omitempty"`
}

// ServiceNetworkSpec defines internal service networking
type ServiceNetworkSpec struct {
	Type  string `yaml:"type,omitempty" json:"type,omitempty"` // ClusterIP, NodePort, LoadBalancer
	Ports []int  `yaml:"ports,omitempty" json:"ports,omitempty"`
}

// NetworkPolicySpec defines network policy rules
type NetworkPolicySpec struct {
	AllowFrom []string `yaml:"allowFrom,omitempty" json:"allowFrom,omitempty"` // Services/namespaces allowed to connect
	AllowTo   []string `yaml:"allowTo,omitempty" json:"allowTo,omitempty"`     // Services/namespaces this can connect to
}

// EnvironmentOverride defines environment-specific configuration overrides
type EnvironmentOverride struct {
	Replicas  *int                `yaml:"replicas,omitempty" json:"replicas,omitempty"`
	Resources *ContainerResources `yaml:"resources,omitempty" json:"resources,omitempty"`
	Scaling   *ScalingSpec        `yaml:"scaling,omitempty" json:"scaling,omitempty"`
	Env       map[string]string   `yaml:"env,omitempty" json:"env,omitempty"`
}

// LifecycleSpec defines lifecycle hooks
type LifecycleSpec struct {
	PreDeploy  *LifecycleHook `yaml:"preDeploy,omitempty" json:"preDeploy,omitempty"`
	PostDeploy *LifecycleHook `yaml:"postDeploy,omitempty" json:"postDeploy,omitempty"`
	PreStop    *LifecycleHook `yaml:"preStop,omitempty" json:"preStop,omitempty"`
}

// LifecycleHook defines a lifecycle hook
type LifecycleHook struct {
	Command []string `yaml:"command,omitempty" json:"command,omitempty"`
	Timeout int      `yaml:"timeout,omitempty" json:"timeout,omitempty"` // seconds
}

// WorkloadStatus represents the runtime status of a workload
type WorkloadStatus struct {
	State            WorkloadState  `yaml:"state" json:"state"`
	Replicas         int            `yaml:"replicas" json:"replicas"`
	ReadyReplicas    int            `yaml:"readyReplicas" json:"readyReplicas"`
	AvailableReplicas int           `yaml:"availableReplicas" json:"availableReplicas"`
	DependencyStatus map[string]DependencyStatus `yaml:"dependencyStatus,omitempty" json:"dependencyStatus,omitempty"`
	LastUpdated      string         `yaml:"lastUpdated,omitempty" json:"lastUpdated,omitempty"`
	Message          string         `yaml:"message,omitempty" json:"message,omitempty"`
}

// WorkloadState represents the state of a workload
type WorkloadState string

const (
	WorkloadStatePending   WorkloadState = "pending"
	WorkloadStateRunning   WorkloadState = "running"
	WorkloadStateDegraded  WorkloadState = "degraded"
	WorkloadStateFailed    WorkloadState = "failed"
	WorkloadStateSucceeded WorkloadState = "succeeded"
)

// DependencyStatus represents the status of a dependency
type DependencyStatus struct {
	Ready      bool   `yaml:"ready" json:"ready"`
	Endpoint   string `yaml:"endpoint,omitempty" json:"endpoint,omitempty"`
	Message    string `yaml:"message,omitempty" json:"message,omitempty"`
}

var (
	workloadNameRegex = regexp.MustCompile(`^[a-z][a-z0-9-]*[a-z0-9]$`)
)

// Validate validates the workload resource
func (w *Workload) Validate() error {
	if w.APIVersion == "" {
		return ErrMissingAPIVersion
	}
	if w.Kind != "Workload" {
		return ErrInvalidKind
	}
	if w.Metadata.Name == "" {
		return ErrMissingName
	}

	// Validate workload name format
	if len(w.Metadata.Name) < 2 || len(w.Metadata.Name) > 63 {
		return fmt.Errorf("workload name must be between 2 and 63 characters")
	}
	if !workloadNameRegex.MatchString(w.Metadata.Name) {
		return fmt.Errorf("workload name must be lowercase alphanumeric with hyphens")
	}

	// Validate team
	if w.Metadata.Team == "" {
		return fmt.Errorf("team is required")
	}
	if len(w.Metadata.Team) > 100 {
		return fmt.Errorf("team name must be 100 characters or less")
	}

	// Validate containers
	if len(w.Spec.Containers) == 0 {
		return fmt.Errorf("at least one container is required")
	}
	if len(w.Spec.Containers) > 10 {
		return fmt.Errorf("too many containers (max 10)")
	}

	containerNames := make(map[string]bool)
	for i, c := range w.Spec.Containers {
		if c.Name == "" {
			return fmt.Errorf("container %d: name is required", i)
		}
		if containerNames[c.Name] {
			return fmt.Errorf("container %d: duplicate container name '%s'", i, c.Name)
		}
		containerNames[c.Name] = true

		if c.Image == "" {
			return fmt.Errorf("container %s: image is required", c.Name)
		}
		if len(c.Image) > 500 {
			return fmt.Errorf("container %s: image name must be 500 characters or less", c.Name)
		}

		// Validate ports
		for j, p := range c.Ports {
			if p.Port <= 0 || p.Port > 65535 {
				return fmt.Errorf("container %s: port %d: invalid port number", c.Name, j)
			}
		}
	}

	// Validate dependencies
	if len(w.Spec.Dependencies) > 50 {
		return fmt.Errorf("too many dependencies (max 50)")
	}

	depNames := make(map[string]bool)
	for i, d := range w.Spec.Dependencies {
		if d.Name == "" {
			return fmt.Errorf("dependency %d: name is required", i)
		}
		if depNames[d.Name] {
			return fmt.Errorf("dependency %d: duplicate dependency name '%s'", i, d.Name)
		}
		depNames[d.Name] = true

		if d.Type == "" {
			return fmt.Errorf("dependency %s: type is required", d.Name)
		}
		if !IsValidDependencyType(d.Type) {
			return fmt.Errorf("dependency %s: invalid type '%s'", d.Name, d.Type)
		}
	}

	// Validate scaling
	if w.Spec.Scaling != nil {
		if w.Spec.Scaling.Min < 0 {
			return fmt.Errorf("scaling min must be >= 0")
		}
		if w.Spec.Scaling.Max < w.Spec.Scaling.Min {
			return fmt.Errorf("scaling max must be >= min")
		}
		if w.Spec.Scaling.Max > 1000 {
			return fmt.Errorf("scaling max must be <= 1000")
		}
		if w.Spec.Scaling.TargetCPU < 0 || w.Spec.Scaling.TargetCPU > 100 {
			return fmt.Errorf("scaling targetCPU must be between 0 and 100")
		}
		if w.Spec.Scaling.TargetMem < 0 || w.Spec.Scaling.TargetMem > 100 {
			return fmt.Errorf("scaling targetMem must be between 0 and 100")
		}
	}

	// Validate labels and annotations
	if len(w.Metadata.Labels) > 50 {
		return fmt.Errorf("too many labels (max 50)")
	}
	if len(w.Metadata.Annotations) > 50 {
		return fmt.Errorf("too many annotations (max 50)")
	}

	return nil
}

// DependencyType represents supported dependency types
type DependencyType string

const (
	DependencyTypePostgres      DependencyType = "postgres"
	DependencyTypeMySQL         DependencyType = "mysql"
	DependencyTypeRedis         DependencyType = "redis"
	DependencyTypeMongoDB       DependencyType = "mongodb"
	DependencyTypeS3            DependencyType = "s3"
	DependencyTypeKafka         DependencyType = "kafka"
	DependencyTypeRabbitMQ      DependencyType = "rabbitmq"
	DependencyTypeElasticsearch DependencyType = "elasticsearch"
	DependencyTypeMemcached     DependencyType = "memcached"
	DependencyTypeDynamoDB      DependencyType = "dynamodb"
)

// IsValidDependencyType checks if a dependency type is supported
func IsValidDependencyType(depType string) bool {
	validTypes := map[string]bool{
		"postgres":      true,
		"mysql":         true,
		"redis":         true,
		"mongodb":       true,
		"s3":            true,
		"kafka":         true,
		"rabbitmq":      true,
		"elasticsearch": true,
		"memcached":     true,
		"dynamodb":      true,
	}
	return validTypes[depType]
}

// GetFullName returns the fully qualified workload name
func (w *Workload) GetFullName() string {
	if w.Metadata.Organization != "" {
		return fmt.Sprintf("%s/%s", w.Metadata.Organization, w.Metadata.Name)
	}
	return w.Metadata.Name
}

// DefaultDependencyConfig returns default configuration for a dependency type
func DefaultDependencyConfig(depType string) map[string]interface{} {
	defaults := map[string]map[string]interface{}{
		"postgres": {
			"version": "15",
			"size":    "small",
			"backup":  "daily",
		},
		"mysql": {
			"version": "8.0",
			"size":    "small",
			"backup":  "daily",
		},
		"redis": {
			"version": "7",
			"size":    "small",
		},
		"mongodb": {
			"version": "6.0",
			"size":    "small",
		},
		"s3": {
			"versioning": false,
			"encryption": true,
		},
		"kafka": {
			"version":    "3.5",
			"partitions": 3,
			"replicas":   2,
		},
		"rabbitmq": {
			"version": "3.12",
			"size":    "small",
		},
		"elasticsearch": {
			"version": "8",
			"size":    "small",
		},
		"memcached": {
			"version": "1.6",
			"size":    "small",
		},
		"dynamodb": {
			"billingMode": "PAY_PER_REQUEST",
		},
	}
	return defaults[depType]
}
