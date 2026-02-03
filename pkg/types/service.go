package types

import (
	"fmt"
	"regexp"
	"time"
)

// Service represents a developer service (microservice, frontend, database, etc.)
type Service struct {
	APIVersion string        `yaml:"apiVersion" json:"apiVersion"`
	Kind       string        `yaml:"kind" json:"kind"`
	Metadata   Metadata      `yaml:"metadata" json:"metadata"`
	Spec       ServiceSpec   `yaml:"spec" json:"spec"`
	Status     ServiceStatus `yaml:"status,omitempty" json:"status,omitempty"`
}

// ServiceSpec defines the service specification
type ServiceSpec struct {
	Type         ServiceType           `yaml:"type" json:"type"`
	Runtime      string                `yaml:"runtime,omitempty" json:"runtime,omitempty"`
	Owner        ServiceOwner          `yaml:"owner" json:"owner"`
	Repository   *RepositoryConfig     `yaml:"repository,omitempty" json:"repository,omitempty"`
	Dependencies []ServiceDependency   `yaml:"dependencies,omitempty" json:"dependencies,omitempty"`
	Links        []ServiceLink         `yaml:"links,omitempty" json:"links,omitempty"`
	Health       *HealthConfig         `yaml:"health,omitempty" json:"health,omitempty"`
	Resources    *ResourceRequirements `yaml:"resources,omitempty" json:"resources,omitempty"`
	SLO          *SLOConfig            `yaml:"slo,omitempty" json:"slo,omitempty"`
}

// ServiceType represents the type of service
type ServiceType string

const (
	ServiceTypeMicroservice ServiceType = "microservice"
	ServiceTypeFrontend     ServiceType = "frontend"
	ServiceTypeBackend      ServiceType = "backend"
	ServiceTypeDatabase     ServiceType = "database"
	ServiceTypeCache        ServiceType = "cache"
	ServiceTypeQueue        ServiceType = "queue"
	ServiceTypeAPI          ServiceType = "api"
	ServiceTypeWorker       ServiceType = "worker"
)

// ServiceOwner defines the service ownership
type ServiceOwner struct {
	Team  string `yaml:"team" json:"team"`
	Email string `yaml:"email,omitempty" json:"email,omitempty"`
	Slack string `yaml:"slack,omitempty" json:"slack,omitempty"`
}

// RepositoryConfig defines the source code repository
type RepositoryConfig struct {
	URL    string `yaml:"url" json:"url"`
	Branch string `yaml:"branch,omitempty" json:"branch,omitempty"`
	Path   string `yaml:"path,omitempty" json:"path,omitempty"`
}

// ServiceDependency defines a dependency on another service
type ServiceDependency struct {
	Name     string `yaml:"name" json:"name"`
	Type     string `yaml:"type,omitempty" json:"type,omitempty"`
	Required bool   `yaml:"required,omitempty" json:"required,omitempty"`
}

// ServiceLink defines external links (docs, monitoring, etc.)
type ServiceLink struct {
	Name string `yaml:"name" json:"name"`
	URL  string `yaml:"url" json:"url"`
	Type string `yaml:"type,omitempty" json:"type,omitempty"` // docs, dashboard, repo, etc.
}

// HealthConfig defines health check configuration
type HealthConfig struct {
	Endpoint    string        `yaml:"endpoint,omitempty" json:"endpoint,omitempty"`
	Interval    time.Duration `yaml:"interval,omitempty" json:"interval,omitempty"`
	Timeout     time.Duration `yaml:"timeout,omitempty" json:"timeout,omitempty"`
	Retries     int           `yaml:"retries,omitempty" json:"retries,omitempty"`
	StartPeriod time.Duration `yaml:"startPeriod,omitempty" json:"startPeriod,omitempty"`
}

// ResourceRequirements defines resource requirements
type ResourceRequirements struct {
	CPU    string `yaml:"cpu,omitempty" json:"cpu,omitempty"`       // e.g., "500m", "1"
	Memory string `yaml:"memory,omitempty" json:"memory,omitempty"` // e.g., "512Mi", "1Gi"
	Disk   string `yaml:"disk,omitempty" json:"disk,omitempty"`     // e.g., "10Gi"
}

// SLOConfig defines Service Level Objectives
type SLOConfig struct {
	Availability float64     `yaml:"availability,omitempty" json:"availability,omitempty"` // e.g., 99.9
	Latency      *LatencySLO `yaml:"latency,omitempty" json:"latency,omitempty"`
	ErrorRate    float64     `yaml:"errorRate,omitempty" json:"errorRate,omitempty"` // e.g., 0.1 for 0.1%
}

// LatencySLO defines latency SLO targets
type LatencySLO struct {
	P50 time.Duration `yaml:"p50,omitempty" json:"p50,omitempty"`
	P95 time.Duration `yaml:"p95,omitempty" json:"p95,omitempty"`
	P99 time.Duration `yaml:"p99,omitempty" json:"p99,omitempty"`
}

// ServiceStatus represents the runtime status of a service
type ServiceStatus struct {
	State        ServiceState    `json:"state"`
	Health       ServiceHealth   `json:"health"`
	LastDeployed *time.Time      `json:"lastDeployed,omitempty"`
	Version      string          `json:"version,omitempty"`
	Metrics      *ServiceMetrics `json:"metrics,omitempty"`
	Message      string          `json:"message,omitempty"`
}

// ServiceState represents the lifecycle state of a service
type ServiceState string

const (
	ServiceStateDraft      ServiceState = "draft"
	ServiceStateActive     ServiceState = "active"
	ServiceStateDeprecated ServiceState = "deprecated"
	ServiceStateRetired    ServiceState = "retired"
	ServiceStateRunning    ServiceState = "running"
	ServiceStateStopped    ServiceState = "stopped"
	ServiceStateFailed     ServiceState = "failed"
)

// ServiceHealth represents the health status of a service
type ServiceHealth string

const (
	ServiceHealthHealthy  ServiceHealth = "healthy"
	ServiceHealthDegraded ServiceHealth = "degraded"
	ServiceHealthDown     ServiceHealth = "down"
	ServiceHealthUnknown  ServiceHealth = "unknown"
)

// ServiceMetrics represents runtime metrics for a service
type ServiceMetrics struct {
	RequestRate float64 `json:"requestRate,omitempty"` // requests per second
	ErrorRate   float64 `json:"errorRate,omitempty"`   // percentage
	LatencyP50  float64 `json:"latencyP50,omitempty"`  // milliseconds
	LatencyP95  float64 `json:"latencyP95,omitempty"`  // milliseconds
	LatencyP99  float64 `json:"latencyP99,omitempty"`  // milliseconds
	CPUUsage    float64 `json:"cpuUsage,omitempty"`    // percentage
	MemoryUsage float64 `json:"memoryUsage,omitempty"` // bytes
	DiskUsage   float64 `json:"diskUsage,omitempty"`   // bytes
}

var (
	// serviceNameRegex validates service names (lowercase alphanumeric with hyphens)
	serviceNameRegex = regexp.MustCompile(`^[a-z][a-z0-9-]*[a-z0-9]$`)
)

// Validate validates the service resource with security checks
func (s *Service) Validate() error {
	if s.APIVersion == "" {
		return ErrMissingAPIVersion
	}
	if s.Kind != "Service" {
		return ErrInvalidKind
	}
	if s.Metadata.Name == "" {
		return ErrMissingName
	}

	// Security: Validate service name format
	if len(s.Metadata.Name) < 2 || len(s.Metadata.Name) > 63 {
		return fmt.Errorf("service name must be between 2 and 63 characters")
	}
	if !serviceNameRegex.MatchString(s.Metadata.Name) {
		return fmt.Errorf("service name must be lowercase alphanumeric with hyphens (start and end with alphanumeric)")
	}

	// Validate service type
	if s.Spec.Type == "" {
		return fmt.Errorf("service type is required")
	}
	if !IsValidServiceType(s.Spec.Type) {
		return fmt.Errorf("invalid service type: %s", s.Spec.Type)
	}

	// Validate owner
	if s.Spec.Owner.Team == "" {
		return fmt.Errorf("owner team is required")
	}
	if len(s.Spec.Owner.Team) > 100 {
		return fmt.Errorf("owner team name must be 100 characters or less")
	}
	if s.Spec.Owner.Email != "" {
		if len(s.Spec.Owner.Email) > 254 {
			return fmt.Errorf("owner email must be 254 characters or less")
		}
		if !emailRegex.MatchString(s.Spec.Owner.Email) {
			return fmt.Errorf("invalid owner email format")
		}
	}

	// Validate repository config if present
	if s.Spec.Repository != nil {
		if s.Spec.Repository.URL == "" {
			return fmt.Errorf("repository URL is required when repository is specified")
		}
		if len(s.Spec.Repository.URL) > 2048 {
			return fmt.Errorf("repository URL must be 2048 characters or less")
		}
		if s.Spec.Repository.Branch != "" && len(s.Spec.Repository.Branch) > 255 {
			return fmt.Errorf("repository branch name must be 255 characters or less")
		}
		if s.Spec.Repository.Path != "" && len(s.Spec.Repository.Path) > 1024 {
			return fmt.Errorf("repository path must be 1024 characters or less")
		}
	}

	// Security: Limit number of dependencies
	if len(s.Spec.Dependencies) > 100 {
		return fmt.Errorf("too many dependencies (max 100)")
	}

	// Validate dependencies
	for i, dep := range s.Spec.Dependencies {
		if dep.Name == "" {
			return fmt.Errorf("dependency %d: name is required", i)
		}
		if len(dep.Name) > 63 {
			return fmt.Errorf("dependency %d: name must be 63 characters or less", i)
		}
		if dep.Type != "" && len(dep.Type) > 50 {
			return fmt.Errorf("dependency %d: type must be 50 characters or less", i)
		}
	}

	// Security: Limit number of links
	if len(s.Spec.Links) > 50 {
		return fmt.Errorf("too many links (max 50)")
	}

	// Validate links
	for i, link := range s.Spec.Links {
		if link.Name == "" {
			return fmt.Errorf("link %d: name is required", i)
		}
		if link.URL == "" {
			return fmt.Errorf("link %d: URL is required", i)
		}
		if len(link.Name) > 100 {
			return fmt.Errorf("link %d: name must be 100 characters or less", i)
		}
		if len(link.URL) > 2048 {
			return fmt.Errorf("link %d: URL must be 2048 characters or less", i)
		}
	}

	// Validate health config if present
	if s.Spec.Health != nil {
		if s.Spec.Health.Endpoint != "" && len(s.Spec.Health.Endpoint) > 1024 {
			return fmt.Errorf("health endpoint must be 1024 characters or less")
		}
		if s.Spec.Health.Retries < 0 || s.Spec.Health.Retries > 10 {
			return fmt.Errorf("health retries must be between 0 and 10")
		}
	}

	// Validate SLO config if present
	if s.Spec.SLO != nil {
		if s.Spec.SLO.Availability < 0 || s.Spec.SLO.Availability > 100 {
			return fmt.Errorf("SLO availability must be between 0 and 100")
		}
		if s.Spec.SLO.ErrorRate < 0 || s.Spec.SLO.ErrorRate > 100 {
			return fmt.Errorf("SLO error rate must be between 0 and 100")
		}
	}

	// Validate labels and annotations (inherited from Metadata)
	if len(s.Metadata.Labels) > 50 {
		return fmt.Errorf("too many labels (max 50)")
	}
	if len(s.Metadata.Annotations) > 50 {
		return fmt.Errorf("too many annotations (max 50)")
	}

	return nil
}

// IsValidServiceType checks if a service type is valid
func IsValidServiceType(serviceType ServiceType) bool {
	return serviceType == ServiceTypeMicroservice ||
		serviceType == ServiceTypeFrontend ||
		serviceType == ServiceTypeBackend ||
		serviceType == ServiceTypeDatabase ||
		serviceType == ServiceTypeCache ||
		serviceType == ServiceTypeQueue ||
		serviceType == ServiceTypeAPI ||
		serviceType == ServiceTypeWorker
}

// IsValidServiceState checks if a service state is valid
func IsValidServiceState(state ServiceState) bool {
	return state == ServiceStateDraft ||
		state == ServiceStateActive ||
		state == ServiceStateDeprecated ||
		state == ServiceStateRetired ||
		state == ServiceStateRunning ||
		state == ServiceStateStopped ||
		state == ServiceStateFailed
}

// IsValidServiceHealth checks if a service health is valid
func IsValidServiceHealth(health ServiceHealth) bool {
	return health == ServiceHealthHealthy ||
		health == ServiceHealthDegraded ||
		health == ServiceHealthDown ||
		health == ServiceHealthUnknown
}

// GetFullName returns the fully qualified service name (org/service)
func (s *Service) GetFullName() string {
	if s.Metadata.Organization != "" {
		return fmt.Sprintf("%s/%s", s.Metadata.Organization, s.Metadata.Name)
	}
	return s.Metadata.Name
}
