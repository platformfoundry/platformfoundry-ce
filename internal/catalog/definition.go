// Package catalog implements a self-service resource catalog for developers
// to provision resources without platform team involvement.
package catalog

import (
	"context"
	"fmt"
	"regexp"
	"sync"
	"time"
)

// ResourceDefinition defines a type of resource that can be provisioned
type ResourceDefinition struct {
	APIVersion string             `json:"apiVersion" yaml:"apiVersion"`
	Kind       string             `json:"kind" yaml:"kind"`
	Metadata   DefinitionMetadata `json:"metadata" yaml:"metadata"`
	Spec       ResourceSpec       `json:"spec" yaml:"spec"`
}

// DefinitionMetadata contains resource definition metadata
type DefinitionMetadata struct {
	Name        string            `json:"name" yaml:"name"`
	Description string            `json:"description,omitempty" yaml:"description,omitempty"`
	Category    string            `json:"category,omitempty" yaml:"category,omitempty"`
	Labels      map[string]string `json:"labels,omitempty" yaml:"labels,omitempty"`
	Annotations map[string]string `json:"annotations,omitempty" yaml:"annotations,omitempty"`
}

// ResourceSpec defines the resource specification
type ResourceSpec struct {
	Type     string                 `json:"type" yaml:"type"`
	Driver   string                 `json:"driver" yaml:"driver"`
	Inputs   []InputDef             `json:"inputs,omitempty" yaml:"inputs,omitempty"`
	Outputs  []OutputDef            `json:"outputs,omitempty" yaml:"outputs,omitempty"`
	Policies []PolicyRef            `json:"policies,omitempty" yaml:"policies,omitempty"`
	Defaults map[string]interface{} `json:"defaults,omitempty" yaml:"defaults,omitempty"`
}

// InputDef defines an input parameter
type InputDef struct {
	Name        string           `json:"name" yaml:"name"`
	Type        string           `json:"type" yaml:"type"` // string, number, boolean, enum
	Description string           `json:"description,omitempty" yaml:"description,omitempty"`
	Required    bool             `json:"required,omitempty" yaml:"required,omitempty"`
	Default     interface{}      `json:"default,omitempty" yaml:"default,omitempty"`
	Enum        []interface{}    `json:"enum,omitempty" yaml:"enum,omitempty"`
	Validation  *InputValidation `json:"validation,omitempty" yaml:"validation,omitempty"`
}

// InputValidation defines validation rules for an input
type InputValidation struct {
	Pattern   string   `json:"pattern,omitempty" yaml:"pattern,omitempty"`
	Min       *float64 `json:"min,omitempty" yaml:"min,omitempty"`
	Max       *float64 `json:"max,omitempty" yaml:"max,omitempty"`
	MinLength *int     `json:"minLength,omitempty" yaml:"minLength,omitempty"`
	MaxLength *int     `json:"maxLength,omitempty" yaml:"maxLength,omitempty"`
}

// OutputDef defines an output value
type OutputDef struct {
	Name        string `json:"name" yaml:"name"`
	Type        string `json:"type,omitempty" yaml:"type,omitempty"`
	Description string `json:"description,omitempty" yaml:"description,omitempty"`
	Secret      bool   `json:"secret,omitempty" yaml:"secret,omitempty"`
}

// PolicyRef references a policy to apply
type PolicyRef struct {
	Name  string      `json:"name" yaml:"name"`
	Value interface{} `json:"value,omitempty" yaml:"value,omitempty"`
}

// ResourceRequest represents a request to provision a resource
type ResourceRequest struct {
	ID             string                 `json:"id"`
	DefinitionName string                 `json:"definitionName"`
	Name           string                 `json:"name"`
	Application    string                 `json:"application"`
	Environment    string                 `json:"environment"`
	Team           string                 `json:"team"`
	Inputs         map[string]interface{} `json:"inputs"`
	RequestedBy    string                 `json:"requestedBy"`
	Status         RequestStatus          `json:"status"`
	StatusMessage  string                 `json:"statusMessage,omitempty"`
	ApprovedBy     string                 `json:"approvedBy,omitempty"`
	CreatedAt      time.Time              `json:"createdAt"`
	UpdatedAt      time.Time              `json:"updatedAt"`
	ProvisionedAt  *time.Time             `json:"provisionedAt,omitempty"`
	Outputs        map[string]string      `json:"outputs,omitempty"`
	EstimatedCost  *CostEstimate          `json:"estimatedCost,omitempty"`
}

// RequestStatus represents the status of a resource request
type RequestStatus string

const (
	RequestPending         RequestStatus = "pending"
	RequestPendingApproval RequestStatus = "pending_approval"
	RequestApproved        RequestStatus = "approved"
	RequestRejected        RequestStatus = "rejected"
	RequestProvisioning    RequestStatus = "provisioning"
	RequestProvisioned     RequestStatus = "provisioned"
	RequestFailed          RequestStatus = "failed"
	RequestDeleting        RequestStatus = "deleting"
	RequestDeleted         RequestStatus = "deleted"
)

// CostEstimate represents an estimated cost for a resource
type CostEstimate struct {
	HourlyCost  float64            `json:"hourlyCost"`
	MonthlyCost float64            `json:"monthlyCost"`
	Currency    string             `json:"currency"`
	Breakdown   map[string]float64 `json:"breakdown,omitempty"`
}

// Quota defines resource quotas for a team or project
type Quota struct {
	ID        string            `json:"id"`
	Team      string            `json:"team"`
	Project   string            `json:"project,omitempty"`
	Limits    map[string]int    `json:"limits"` // e.g., "postgres": 5, "redis": 10
	Used      map[string]int    `json:"used"`
	CostLimit float64           `json:"costLimit,omitempty"` // Monthly cost limit
	CostUsed  float64           `json:"costUsed,omitempty"`
	Labels    map[string]string `json:"labels,omitempty"`
}

// Catalog manages resource definitions and requests
type Catalog struct {
	definitions  map[string]*ResourceDefinition
	requests     map[string]*ResourceRequest
	quotas       map[string]*Quota
	stateBackend StateBackend
	mu           sync.RWMutex
}

// StateBackend interface for persistence
type StateBackend interface {
	Get(ctx context.Context, kind, id string) (interface{}, error)
	Put(ctx context.Context, kind, id string, value interface{}) error
	Delete(ctx context.Context, kind, id string) error
	List(ctx context.Context, kind string) ([]interface{}, error)
}

// NewCatalog creates a new resource catalog
func NewCatalog(backend StateBackend) *Catalog {
	c := &Catalog{
		definitions:  make(map[string]*ResourceDefinition),
		requests:     make(map[string]*ResourceRequest),
		quotas:       make(map[string]*Quota),
		stateBackend: backend,
	}
	c.registerBuiltinDefinitions()
	return c
}

func (c *Catalog) registerBuiltinDefinitions() {
	// PostgreSQL
	c.RegisterDefinition(&ResourceDefinition{
		APIVersion: "platformfoundry.io/v1",
		Kind:       "ResourceDefinition",
		Metadata: DefinitionMetadata{
			Name:        "postgres-standard",
			Description: "PostgreSQL database - standard configuration",
			Category:    "database",
		},
		Spec: ResourceSpec{
			Type:   "postgres",
			Driver: "terraform-aws-rds",
			Inputs: []InputDef{
				{Name: "version", Type: "enum", Default: "15", Enum: []interface{}{"13", "14", "15", "16"}},
				{Name: "size", Type: "enum", Default: "small", Enum: []interface{}{"small", "medium", "large"}},
				{Name: "storage_gb", Type: "number", Default: 20, Validation: &InputValidation{Min: ptr(10.0), Max: ptr(1000.0)}},
				{Name: "backup_retention_days", Type: "number", Default: 7},
			},
			Outputs: []OutputDef{
				{Name: "host", Type: "string", Description: "Database hostname"},
				{Name: "port", Type: "string", Description: "Database port"},
				{Name: "name", Type: "string", Description: "Database name"},
				{Name: "username", Type: "string", Description: "Database username"},
				{Name: "password", Type: "string", Description: "Database password", Secret: true},
				{Name: "connection_string", Type: "string", Description: "Connection string", Secret: true},
			},
			Policies: []PolicyRef{
				{Name: "max-cost", Value: "$500/month"},
				{Name: "approved-regions", Value: []string{"us-east-1", "us-west-2", "eu-west-1"}},
			},
		},
	})

	// Redis
	c.RegisterDefinition(&ResourceDefinition{
		APIVersion: "platformfoundry.io/v1",
		Kind:       "ResourceDefinition",
		Metadata: DefinitionMetadata{
			Name:        "redis-standard",
			Description: "Redis cache - standard configuration",
			Category:    "cache",
		},
		Spec: ResourceSpec{
			Type:   "redis",
			Driver: "terraform-aws-elasticache",
			Inputs: []InputDef{
				{Name: "version", Type: "enum", Default: "7.0", Enum: []interface{}{"6.2", "7.0"}},
				{Name: "size", Type: "enum", Default: "small", Enum: []interface{}{"small", "medium", "large"}},
				{Name: "cluster_mode", Type: "boolean", Default: false},
			},
			Outputs: []OutputDef{
				{Name: "host", Type: "string"},
				{Name: "port", Type: "string"},
				{Name: "password", Type: "string", Secret: true},
				{Name: "url", Type: "string", Secret: true},
			},
		},
	})

	// S3 Bucket
	c.RegisterDefinition(&ResourceDefinition{
		APIVersion: "platformfoundry.io/v1",
		Kind:       "ResourceDefinition",
		Metadata: DefinitionMetadata{
			Name:        "s3-bucket",
			Description: "S3 object storage bucket",
			Category:    "storage",
		},
		Spec: ResourceSpec{
			Type:   "s3",
			Driver: "terraform-aws-s3",
			Inputs: []InputDef{
				{Name: "versioning", Type: "boolean", Default: true},
				{Name: "encryption", Type: "enum", Default: "AES256", Enum: []interface{}{"AES256", "aws:kms"}},
				{Name: "lifecycle_days", Type: "number", Default: 0, Description: "Days before archival (0 = disabled)"},
			},
			Outputs: []OutputDef{
				{Name: "bucket", Type: "string"},
				{Name: "arn", Type: "string"},
				{Name: "region", Type: "string"},
				{Name: "endpoint", Type: "string"},
			},
		},
	})

	// MySQL
	c.RegisterDefinition(&ResourceDefinition{
		APIVersion: "platformfoundry.io/v1",
		Kind:       "ResourceDefinition",
		Metadata: DefinitionMetadata{
			Name:        "mysql-standard",
			Description: "MySQL database - standard configuration",
			Category:    "database",
		},
		Spec: ResourceSpec{
			Type:   "mysql",
			Driver: "terraform-aws-rds",
			Inputs: []InputDef{
				{Name: "version", Type: "enum", Default: "8.0", Enum: []interface{}{"5.7", "8.0"}},
				{Name: "size", Type: "enum", Default: "small", Enum: []interface{}{"small", "medium", "large"}},
				{Name: "storage_gb", Type: "number", Default: 20},
			},
			Outputs: []OutputDef{
				{Name: "host", Type: "string"},
				{Name: "port", Type: "string"},
				{Name: "name", Type: "string"},
				{Name: "username", Type: "string"},
				{Name: "password", Type: "string", Secret: true},
				{Name: "connection_string", Type: "string", Secret: true},
			},
		},
	})

	// MongoDB
	c.RegisterDefinition(&ResourceDefinition{
		APIVersion: "platformfoundry.io/v1",
		Kind:       "ResourceDefinition",
		Metadata: DefinitionMetadata{
			Name:        "mongodb-standard",
			Description: "MongoDB document database",
			Category:    "database",
		},
		Spec: ResourceSpec{
			Type:   "mongodb",
			Driver: "terraform-mongodb-atlas",
			Inputs: []InputDef{
				{Name: "version", Type: "enum", Default: "6.0", Enum: []interface{}{"5.0", "6.0", "7.0"}},
				{Name: "size", Type: "enum", Default: "M10", Enum: []interface{}{"M10", "M20", "M30"}},
			},
			Outputs: []OutputDef{
				{Name: "connection_string", Type: "string", Secret: true},
				{Name: "host", Type: "string"},
			},
		},
	})

	// RabbitMQ
	c.RegisterDefinition(&ResourceDefinition{
		APIVersion: "platformfoundry.io/v1",
		Kind:       "ResourceDefinition",
		Metadata: DefinitionMetadata{
			Name:        "rabbitmq-standard",
			Description: "RabbitMQ message broker",
			Category:    "messaging",
		},
		Spec: ResourceSpec{
			Type:   "rabbitmq",
			Driver: "terraform-aws-mq",
			Inputs: []InputDef{
				{Name: "size", Type: "enum", Default: "mq.t3.micro", Enum: []interface{}{"mq.t3.micro", "mq.m5.large"}},
			},
			Outputs: []OutputDef{
				{Name: "host", Type: "string"},
				{Name: "port", Type: "string"},
				{Name: "username", Type: "string"},
				{Name: "password", Type: "string", Secret: true},
				{Name: "url", Type: "string", Secret: true},
			},
		},
	})
}

func ptr(f float64) *float64 { return &f }

// RegisterDefinition registers a resource definition
func (c *Catalog) RegisterDefinition(def *ResourceDefinition) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if def.Metadata.Name == "" {
		return fmt.Errorf("definition name is required")
	}

	c.definitions[def.Metadata.Name] = def

	if c.stateBackend != nil {
		return c.stateBackend.Put(context.Background(), "ResourceDefinition", def.Metadata.Name, def)
	}

	return nil
}

// GetDefinition returns a resource definition by name
func (c *Catalog) GetDefinition(name string) (*ResourceDefinition, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	def, ok := c.definitions[name]
	if !ok {
		return nil, fmt.Errorf("definition not found: %s", name)
	}
	return def, nil
}

// ListDefinitions returns all resource definitions
func (c *Catalog) ListDefinitions() []*ResourceDefinition {
	c.mu.RLock()
	defer c.mu.RUnlock()

	defs := make([]*ResourceDefinition, 0, len(c.definitions))
	for _, def := range c.definitions {
		defs = append(defs, def)
	}
	return defs
}

// ListDefinitionsByCategory returns definitions filtered by category
func (c *Catalog) ListDefinitionsByCategory(category string) []*ResourceDefinition {
	c.mu.RLock()
	defer c.mu.RUnlock()

	var defs []*ResourceDefinition
	for _, def := range c.definitions {
		if def.Metadata.Category == category {
			defs = append(defs, def)
		}
	}
	return defs
}

// ValidateInputs validates request inputs against definition
func (c *Catalog) ValidateInputs(def *ResourceDefinition, inputs map[string]interface{}) []ValidationError {
	var errors []ValidationError

	for _, inputDef := range def.Spec.Inputs {
		value, provided := inputs[inputDef.Name]

		// Check required
		if inputDef.Required && !provided {
			errors = append(errors, ValidationError{
				Field:   inputDef.Name,
				Message: "required input is missing",
			})
			continue
		}

		if !provided {
			continue
		}

		// Check enum
		if len(inputDef.Enum) > 0 {
			found := false
			for _, allowed := range inputDef.Enum {
				if value == allowed {
					found = true
					break
				}
			}
			if !found {
				errors = append(errors, ValidationError{
					Field:   inputDef.Name,
					Message: fmt.Sprintf("value must be one of: %v", inputDef.Enum),
				})
			}
		}

		// Check validation rules
		if inputDef.Validation != nil {
			if err := c.validateInput(inputDef, value); err != nil {
				errors = append(errors, ValidationError{
					Field:   inputDef.Name,
					Message: err.Error(),
				})
			}
		}
	}

	return errors
}

func (c *Catalog) validateInput(def InputDef, value interface{}) error {
	v := def.Validation
	if v == nil {
		return nil
	}

	// Pattern validation for strings
	if v.Pattern != "" {
		if str, ok := value.(string); ok {
			matched, err := regexp.MatchString(v.Pattern, str)
			if err != nil {
				return fmt.Errorf("invalid pattern: %v", err)
			}
			if !matched {
				return fmt.Errorf("value does not match pattern: %s", v.Pattern)
			}
		}
	}

	// Numeric range validation
	if v.Min != nil || v.Max != nil {
		var num float64
		switch n := value.(type) {
		case int:
			num = float64(n)
		case float64:
			num = n
		default:
			return nil // Not a number, skip numeric validation
		}

		if v.Min != nil && num < *v.Min {
			return fmt.Errorf("value must be >= %v", *v.Min)
		}
		if v.Max != nil && num > *v.Max {
			return fmt.Errorf("value must be <= %v", *v.Max)
		}
	}

	// Length validation for strings
	if v.MinLength != nil || v.MaxLength != nil {
		if str, ok := value.(string); ok {
			if v.MinLength != nil && len(str) < *v.MinLength {
				return fmt.Errorf("value must be at least %d characters", *v.MinLength)
			}
			if v.MaxLength != nil && len(str) > *v.MaxLength {
				return fmt.Errorf("value must be at most %d characters", *v.MaxLength)
			}
		}
	}

	return nil
}

// ValidationError represents an input validation error
type ValidationError struct {
	Field   string `json:"field"`
	Message string `json:"message"`
}

func (e ValidationError) Error() string {
	return fmt.Sprintf("%s: %s", e.Field, e.Message)
}
