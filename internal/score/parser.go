// Package score implements the Score workload specification parser and validator.
// Score is a platform-agnostic workload specification that enables portable definitions.
// See: https://score.dev
package score

import (
	"fmt"
	"os"
	"regexp"
	"strings"

	"gopkg.in/yaml.v3"
)

// Workload represents a Score workload specification
type Workload struct {
	APIVersion string               `yaml:"apiVersion" json:"apiVersion"`
	Metadata   WorkloadMetadata     `yaml:"metadata" json:"metadata"`
	Containers map[string]Container `yaml:"containers" json:"containers"`
	Resources  map[string]Resource  `yaml:"resources,omitempty" json:"resources,omitempty"`
	Service    *ServiceSpec         `yaml:"service,omitempty" json:"service,omitempty"`
}

// WorkloadMetadata contains workload metadata
type WorkloadMetadata struct {
	Name        string            `yaml:"name" json:"name"`
	Labels      map[string]string `yaml:"labels,omitempty" json:"labels,omitempty"`
	Annotations map[string]string `yaml:"annotations,omitempty" json:"annotations,omitempty"`
}

// Container defines a container within the workload
type Container struct {
	Image          string                `yaml:"image" json:"image"`
	Command        []string              `yaml:"command,omitempty" json:"command,omitempty"`
	Args           []string              `yaml:"args,omitempty" json:"args,omitempty"`
	Variables      map[string]string     `yaml:"variables,omitempty" json:"variables,omitempty"`
	Files          []FileMount           `yaml:"files,omitempty" json:"files,omitempty"`
	Volumes        []VolumeMount         `yaml:"volumes,omitempty" json:"volumes,omitempty"`
	Resources      *ResourceRequirements `yaml:"resources,omitempty" json:"resources,omitempty"`
	LivenessProbe  *Probe                `yaml:"livenessProbe,omitempty" json:"livenessProbe,omitempty"`
	ReadinessProbe *Probe                `yaml:"readinessProbe,omitempty" json:"readinessProbe,omitempty"`
}

// FileMount defines a file to be mounted in the container
type FileMount struct {
	Target   string  `yaml:"target" json:"target"`
	Mode     string  `yaml:"mode,omitempty" json:"mode,omitempty"`
	Source   *string `yaml:"source,omitempty" json:"source,omitempty"`
	Content  *string `yaml:"content,omitempty" json:"content,omitempty"`
	NoExpand bool    `yaml:"noExpand,omitempty" json:"noExpand,omitempty"`
}

// VolumeMount defines a volume mount
type VolumeMount struct {
	Source   string `yaml:"source" json:"source"`
	Target   string `yaml:"target" json:"target"`
	Path     string `yaml:"path,omitempty" json:"path,omitempty"`
	ReadOnly bool   `yaml:"read_only,omitempty" json:"read_only,omitempty"`
}

// ResourceRequirements defines compute resource requirements
type ResourceRequirements struct {
	Requests *ResourceList `yaml:"requests,omitempty" json:"requests,omitempty"`
	Limits   *ResourceList `yaml:"limits,omitempty" json:"limits,omitempty"`
}

// ResourceList is a set of resource quantities
type ResourceList struct {
	CPU    string `yaml:"cpu,omitempty" json:"cpu,omitempty"`
	Memory string `yaml:"memory,omitempty" json:"memory,omitempty"`
}

// Probe defines a health check probe
type Probe struct {
	HTTPGet *HTTPGetAction `yaml:"httpGet,omitempty" json:"httpGet,omitempty"`
}

// HTTPGetAction describes an action based on HTTP Get requests
type HTTPGetAction struct {
	Path   string `yaml:"path" json:"path"`
	Port   int    `yaml:"port" json:"port"`
	Scheme string `yaml:"scheme,omitempty" json:"scheme,omitempty"`
}

// Resource defines a resource dependency
type Resource struct {
	Type       string                 `yaml:"type" json:"type"`
	Class      string                 `yaml:"class,omitempty" json:"class,omitempty"`
	ID         string                 `yaml:"id,omitempty" json:"id,omitempty"`
	Metadata   map[string]string      `yaml:"metadata,omitempty" json:"metadata,omitempty"`
	Params     map[string]interface{} `yaml:"params,omitempty" json:"params,omitempty"`
	Properties map[string]interface{} `yaml:"properties,omitempty" json:"properties,omitempty"`
}

// ServiceSpec defines the service configuration
type ServiceSpec struct {
	Ports []ServicePort `yaml:"ports,omitempty" json:"ports,omitempty"`
}

// ServicePort defines a port exposed by the service
type ServicePort struct {
	Name       string `yaml:"name,omitempty" json:"name,omitempty"`
	Port       int    `yaml:"port" json:"port"`
	TargetPort int    `yaml:"targetPort,omitempty" json:"targetPort,omitempty"`
	Protocol   string `yaml:"protocol,omitempty" json:"protocol,omitempty"`
}

// Parser parses Score workload specifications
type Parser struct {
	resourceTypes map[string]*ResourceType
}

// ResourceType defines a type of resource that can be used in Score specs
type ResourceType struct {
	Type        string            `yaml:"type" json:"type"`
	Name        string            `yaml:"name" json:"name"`
	Description string            `yaml:"description,omitempty" json:"description,omitempty"`
	Outputs     map[string]Output `yaml:"outputs,omitempty" json:"outputs,omitempty"`
	Params      map[string]Param  `yaml:"params,omitempty" json:"params,omitempty"`
}

// Output defines an output from a resource
type Output struct {
	Type        string `yaml:"type" json:"type"`
	Description string `yaml:"description,omitempty" json:"description,omitempty"`
	Secret      bool   `yaml:"secret,omitempty" json:"secret,omitempty"`
}

// Param defines an input parameter for a resource
type Param struct {
	Type        string      `yaml:"type" json:"type"`
	Description string      `yaml:"description,omitempty" json:"description,omitempty"`
	Default     interface{} `yaml:"default,omitempty" json:"default,omitempty"`
	Required    bool        `yaml:"required,omitempty" json:"required,omitempty"`
}

// NewParser creates a new Score parser with default resource types
func NewParser() *Parser {
	p := &Parser{
		resourceTypes: make(map[string]*ResourceType),
	}
	p.registerDefaultTypes()
	return p
}

// registerDefaultTypes registers the built-in Score resource types
func (p *Parser) registerDefaultTypes() {
	// Database types
	p.RegisterResourceType(&ResourceType{
		Type:        "postgres",
		Name:        "PostgreSQL Database",
		Description: "PostgreSQL relational database",
		Outputs: map[string]Output{
			"host":              {Type: "string", Description: "Database host"},
			"port":              {Type: "integer", Description: "Database port"},
			"name":              {Type: "string", Description: "Database name"},
			"username":          {Type: "string", Description: "Database username"},
			"password":          {Type: "string", Description: "Database password", Secret: true},
			"connection_string": {Type: "string", Description: "Full connection string", Secret: true},
		},
		Params: map[string]Param{
			"version": {Type: "string", Description: "PostgreSQL version", Default: "15"},
		},
	})

	p.RegisterResourceType(&ResourceType{
		Type:        "mysql",
		Name:        "MySQL Database",
		Description: "MySQL relational database",
		Outputs: map[string]Output{
			"host":              {Type: "string", Description: "Database host"},
			"port":              {Type: "integer", Description: "Database port"},
			"name":              {Type: "string", Description: "Database name"},
			"username":          {Type: "string", Description: "Database username"},
			"password":          {Type: "string", Description: "Database password", Secret: true},
			"connection_string": {Type: "string", Description: "Full connection string", Secret: true},
		},
		Params: map[string]Param{
			"version": {Type: "string", Description: "MySQL version", Default: "8.0"},
		},
	})

	p.RegisterResourceType(&ResourceType{
		Type:        "mongodb",
		Name:        "MongoDB Database",
		Description: "MongoDB document database",
		Outputs: map[string]Output{
			"host":              {Type: "string", Description: "Database host"},
			"port":              {Type: "integer", Description: "Database port"},
			"name":              {Type: "string", Description: "Database name"},
			"username":          {Type: "string", Description: "Database username"},
			"password":          {Type: "string", Description: "Database password", Secret: true},
			"connection_string": {Type: "string", Description: "Full connection string", Secret: true},
		},
	})

	// Cache types
	p.RegisterResourceType(&ResourceType{
		Type:        "redis",
		Name:        "Redis Cache",
		Description: "Redis in-memory cache",
		Outputs: map[string]Output{
			"host":     {Type: "string", Description: "Redis host"},
			"port":     {Type: "integer", Description: "Redis port"},
			"password": {Type: "string", Description: "Redis password", Secret: true},
			"url":      {Type: "string", Description: "Redis URL", Secret: true},
		},
	})

	p.RegisterResourceType(&ResourceType{
		Type:        "memcached",
		Name:        "Memcached",
		Description: "Memcached distributed cache",
		Outputs: map[string]Output{
			"host": {Type: "string", Description: "Memcached host"},
			"port": {Type: "integer", Description: "Memcached port"},
		},
	})

	// Message queue types
	p.RegisterResourceType(&ResourceType{
		Type:        "rabbitmq",
		Name:        "RabbitMQ",
		Description: "RabbitMQ message broker",
		Outputs: map[string]Output{
			"host":     {Type: "string", Description: "RabbitMQ host"},
			"port":     {Type: "integer", Description: "RabbitMQ port"},
			"username": {Type: "string", Description: "RabbitMQ username"},
			"password": {Type: "string", Description: "RabbitMQ password", Secret: true},
			"vhost":    {Type: "string", Description: "Virtual host"},
			"url":      {Type: "string", Description: "AMQP URL", Secret: true},
		},
	})

	p.RegisterResourceType(&ResourceType{
		Type:        "kafka",
		Name:        "Apache Kafka",
		Description: "Apache Kafka event streaming",
		Outputs: map[string]Output{
			"brokers": {Type: "string", Description: "Kafka broker addresses"},
		},
	})

	// Storage types
	p.RegisterResourceType(&ResourceType{
		Type:        "s3",
		Name:        "S3 Bucket",
		Description: "S3-compatible object storage",
		Outputs: map[string]Output{
			"bucket":            {Type: "string", Description: "Bucket name"},
			"region":            {Type: "string", Description: "Bucket region"},
			"endpoint":          {Type: "string", Description: "S3 endpoint"},
			"access_key_id":     {Type: "string", Description: "Access key ID", Secret: true},
			"secret_access_key": {Type: "string", Description: "Secret access key", Secret: true},
		},
	})

	p.RegisterResourceType(&ResourceType{
		Type:        "volume",
		Name:        "Persistent Volume",
		Description: "Persistent storage volume",
		Outputs: map[string]Output{
			"source": {Type: "string", Description: "Volume source identifier"},
		},
		Params: map[string]Param{
			"size":  {Type: "string", Description: "Volume size", Default: "1Gi"},
			"class": {Type: "string", Description: "Storage class"},
		},
	})

	// DNS/Routing
	p.RegisterResourceType(&ResourceType{
		Type:        "dns",
		Name:        "DNS Record",
		Description: "DNS record configuration",
		Outputs: map[string]Output{
			"host": {Type: "string", Description: "DNS hostname"},
		},
	})

	p.RegisterResourceType(&ResourceType{
		Type:        "route",
		Name:        "HTTP Route",
		Description: "HTTP routing/ingress",
		Outputs: map[string]Output{
			"host": {Type: "string", Description: "Route hostname"},
			"path": {Type: "string", Description: "Route path"},
		},
	})

	// Environment
	p.RegisterResourceType(&ResourceType{
		Type:        "environment",
		Name:        "Environment",
		Description: "Environment variables",
		Outputs: map[string]Output{
			"name": {Type: "string", Description: "Environment name"},
		},
	})
}

// RegisterResourceType registers a custom resource type
func (p *Parser) RegisterResourceType(rt *ResourceType) {
	p.resourceTypes[rt.Type] = rt
}

// GetResourceType returns a registered resource type
func (p *Parser) GetResourceType(typeName string) (*ResourceType, bool) {
	rt, ok := p.resourceTypes[typeName]
	return rt, ok
}

// ListResourceTypes returns all registered resource types
func (p *Parser) ListResourceTypes() []*ResourceType {
	types := make([]*ResourceType, 0, len(p.resourceTypes))
	for _, rt := range p.resourceTypes {
		types = append(types, rt)
	}
	return types
}

// ParseFile parses a Score workload file
func (p *Parser) ParseFile(path string) (*Workload, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}
	return p.Parse(data)
}

// Parse parses Score workload YAML
func (p *Parser) Parse(data []byte) (*Workload, error) {
	var workload Workload
	if err := yaml.Unmarshal(data, &workload); err != nil {
		return nil, fmt.Errorf("failed to parse YAML: %w", err)
	}
	return &workload, nil
}

// ParseAndValidate parses and validates a Score workload
func (p *Parser) ParseAndValidate(data []byte) (*Workload, []ValidationError, error) {
	workload, err := p.Parse(data)
	if err != nil {
		return nil, nil, err
	}

	errors := p.Validate(workload)
	return workload, errors, nil
}

// ValidationError represents a validation error
type ValidationError struct {
	Field    string `json:"field"`
	Message  string `json:"message"`
	Severity string `json:"severity"` // error, warning
}

func (e ValidationError) Error() string {
	return fmt.Sprintf("%s: %s", e.Field, e.Message)
}

// Validate validates a parsed Score workload
func (p *Parser) Validate(w *Workload) []ValidationError {
	var errors []ValidationError

	// Validate API version
	if w.APIVersion == "" {
		errors = append(errors, ValidationError{
			Field:    "apiVersion",
			Message:  "apiVersion is required",
			Severity: "error",
		})
	} else if !strings.HasPrefix(w.APIVersion, "score.dev/") {
		errors = append(errors, ValidationError{
			Field:    "apiVersion",
			Message:  fmt.Sprintf("unsupported apiVersion: %s (expected score.dev/*)", w.APIVersion),
			Severity: "error",
		})
	}

	// Validate metadata
	if w.Metadata.Name == "" {
		errors = append(errors, ValidationError{
			Field:    "metadata.name",
			Message:  "metadata.name is required",
			Severity: "error",
		})
	} else if !isValidName(w.Metadata.Name) {
		errors = append(errors, ValidationError{
			Field:    "metadata.name",
			Message:  "metadata.name must be lowercase alphanumeric with dashes",
			Severity: "error",
		})
	}

	// Validate containers
	if len(w.Containers) == 0 {
		errors = append(errors, ValidationError{
			Field:    "containers",
			Message:  "at least one container is required",
			Severity: "error",
		})
	}

	for name, container := range w.Containers {
		errors = append(errors, p.validateContainer(name, &container)...)
	}

	// Validate resources
	for name, resource := range w.Resources {
		errors = append(errors, p.validateResource(name, &resource)...)
	}

	// Validate variable references
	errors = append(errors, p.validateVariableReferences(w)...)

	return errors
}

func (p *Parser) validateContainer(name string, c *Container) []ValidationError {
	var errors []ValidationError
	prefix := fmt.Sprintf("containers.%s", name)

	if c.Image == "" {
		errors = append(errors, ValidationError{
			Field:    prefix + ".image",
			Message:  "image is required",
			Severity: "error",
		})
	}

	// Validate resource requirements
	if c.Resources != nil {
		if c.Resources.Requests != nil {
			if c.Resources.Requests.CPU != "" && !isValidCPU(c.Resources.Requests.CPU) {
				errors = append(errors, ValidationError{
					Field:    prefix + ".resources.requests.cpu",
					Message:  "invalid CPU format",
					Severity: "error",
				})
			}
			if c.Resources.Requests.Memory != "" && !isValidMemory(c.Resources.Requests.Memory) {
				errors = append(errors, ValidationError{
					Field:    prefix + ".resources.requests.memory",
					Message:  "invalid memory format",
					Severity: "error",
				})
			}
		}
	}

	// Validate file mounts
	for i, file := range c.Files {
		if file.Target == "" {
			errors = append(errors, ValidationError{
				Field:    fmt.Sprintf("%s.files[%d].target", prefix, i),
				Message:  "target is required",
				Severity: "error",
			})
		}
		if file.Source == nil && file.Content == nil {
			errors = append(errors, ValidationError{
				Field:    fmt.Sprintf("%s.files[%d]", prefix, i),
				Message:  "either source or content is required",
				Severity: "error",
			})
		}
	}

	return errors
}

func (p *Parser) validateResource(name string, r *Resource) []ValidationError {
	var errors []ValidationError
	prefix := fmt.Sprintf("resources.%s", name)

	if r.Type == "" {
		errors = append(errors, ValidationError{
			Field:    prefix + ".type",
			Message:  "type is required",
			Severity: "error",
		})
		return errors
	}

	// Check if resource type is known
	if _, ok := p.resourceTypes[r.Type]; !ok {
		errors = append(errors, ValidationError{
			Field:    prefix + ".type",
			Message:  fmt.Sprintf("unknown resource type: %s", r.Type),
			Severity: "warning",
		})
	}

	return errors
}

func (p *Parser) validateVariableReferences(w *Workload) []ValidationError {
	var errors []ValidationError

	// Extract all resource output references from container variables
	varRefPattern := regexp.MustCompile(`\$\{resources\.([^.]+)\.([^}]+)\}`)

	for containerName, container := range w.Containers {
		for varName, varValue := range container.Variables {
			matches := varRefPattern.FindAllStringSubmatch(varValue, -1)
			for _, match := range matches {
				resourceName := match[1]
				outputName := match[2]

				// Check if resource exists
				resource, exists := w.Resources[resourceName]
				if !exists {
					errors = append(errors, ValidationError{
						Field:    fmt.Sprintf("containers.%s.variables.%s", containerName, varName),
						Message:  fmt.Sprintf("references undefined resource: %s", resourceName),
						Severity: "error",
					})
					continue
				}

				// Check if output exists for the resource type
				if rt, ok := p.resourceTypes[resource.Type]; ok {
					if _, outputExists := rt.Outputs[outputName]; !outputExists {
						errors = append(errors, ValidationError{
							Field:    fmt.Sprintf("containers.%s.variables.%s", containerName, varName),
							Message:  fmt.Sprintf("resource %s (type %s) has no output: %s", resourceName, resource.Type, outputName),
							Severity: "warning",
						})
					}
				}
			}
		}
	}

	return errors
}

// Helper validation functions

var namePattern = regexp.MustCompile(`^[a-z][a-z0-9-]*[a-z0-9]$|^[a-z]$`)

func isValidName(name string) bool {
	return namePattern.MatchString(name)
}

var cpuPattern = regexp.MustCompile(`^(\d+)(m)?$`)

func isValidCPU(cpu string) bool {
	return cpuPattern.MatchString(cpu)
}

var memoryPattern = regexp.MustCompile(`^(\d+)(Ki|Mi|Gi|Ti|Pi|Ei|K|M|G|T|P|E)?$`)

func isValidMemory(memory string) bool {
	return memoryPattern.MatchString(memory)
}
