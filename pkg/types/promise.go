package types

import (
	"fmt"
	"regexp"
	"time"
)

// Promise represents a platform capability that can be self-serviced by developers
// Promises define what infrastructure/services can be requested and the contract
// for how they will be provisioned
type Promise struct {
	APIVersion string          `yaml:"apiVersion" json:"apiVersion"`
	Kind       string          `yaml:"kind" json:"kind"` // "Promise"
	Metadata   PromiseMetadata `yaml:"metadata" json:"metadata"`
	Spec       PromiseSpec     `yaml:"spec" json:"spec"`
}

// PromiseMetadata contains promise metadata
type PromiseMetadata struct {
	Name        string            `yaml:"name" json:"name"`
	Labels      map[string]string `yaml:"labels,omitempty" json:"labels,omitempty"`
	Annotations map[string]string `yaml:"annotations,omitempty" json:"annotations,omitempty"`
	Description string            `yaml:"description,omitempty" json:"description,omitempty"`
}

// PromiseSpec defines the promise specification
type PromiseSpec struct {
	// Description of what this promise provides
	Description string `yaml:"description" json:"description"`

	// Provider identifies which plugin fulfills this promise
	Provider string `yaml:"provider" json:"provider"`

	// Category for organizing promises (database, cache, queue, storage, etc.)
	Category string `yaml:"category" json:"category"`

	// Inputs define the parameters that requesters can provide
	Inputs []PromiseInput `yaml:"inputs" json:"inputs"`

	// Outputs define what will be returned after provisioning
	Outputs []PromiseOutput `yaml:"outputs" json:"outputs"`

	// Policies that apply to this promise
	Policies []string `yaml:"policies,omitempty" json:"policies,omitempty"`

	// Approval configuration
	Approval *PromiseApproval `yaml:"approval,omitempty" json:"approval,omitempty"`

	// Defaults for common configurations
	Defaults map[string]interface{} `yaml:"defaults,omitempty" json:"defaults,omitempty"`

	// ProviderConfig contains provider-specific configuration templates
	ProviderConfig map[string]interface{} `yaml:"providerConfig,omitempty" json:"providerConfig,omitempty"`
}

// PromiseInput defines an input parameter for a promise
type PromiseInput struct {
	Name        string      `yaml:"name" json:"name"`
	Type        string      `yaml:"type" json:"type"` // string, number, boolean, enum
	Description string      `yaml:"description" json:"description"`
	Required    bool        `yaml:"required" json:"required"`
	Default     interface{} `yaml:"default,omitempty" json:"default,omitempty"`
	Enum        []string    `yaml:"enum,omitempty" json:"enum,omitempty"`
	Validation  string      `yaml:"validation,omitempty" json:"validation,omitempty"` // regex pattern
	Min         *float64    `yaml:"min,omitempty" json:"min,omitempty"`
	Max         *float64    `yaml:"max,omitempty" json:"max,omitempty"`
}

// PromiseOutput defines an output from a promise
type PromiseOutput struct {
	Name        string `yaml:"name" json:"name"`
	Type        string `yaml:"type" json:"type"` // string, number, secret
	Description string `yaml:"description" json:"description"`
}

// PromiseApproval defines approval requirements
type PromiseApproval struct {
	Required     bool     `yaml:"required" json:"required"`
	Policy       string   `yaml:"policy,omitempty" json:"policy,omitempty"`
	Approvers    []string `yaml:"approvers,omitempty" json:"approvers,omitempty"`
	Environments []string `yaml:"environments,omitempty" json:"environments,omitempty"` // Only require for these envs
}

// PromiseRequest represents a request to fulfill a promise
type PromiseRequest struct {
	APIVersion string                `yaml:"apiVersion" json:"apiVersion"`
	Kind       string                `yaml:"kind" json:"kind"` // "PromiseRequest"
	Metadata   PromiseRequestMetadata `yaml:"metadata" json:"metadata"`
	Spec       PromiseRequestSpec    `yaml:"spec" json:"spec"`
	Status     *PromiseRequestStatus `yaml:"status,omitempty" json:"status,omitempty"`
}

// PromiseRequestMetadata contains request metadata
type PromiseRequestMetadata struct {
	Name         string            `yaml:"name" json:"name"`
	Team         string            `yaml:"team" json:"team"`
	Labels       map[string]string `yaml:"labels,omitempty" json:"labels,omitempty"`
	Annotations  map[string]string `yaml:"annotations,omitempty" json:"annotations,omitempty"`
	Organization string            `yaml:"organization,omitempty" json:"organization,omitempty"`
	Environment  string            `yaml:"environment,omitempty" json:"environment,omitempty"`
}

// PromiseRequestSpec defines the request specification
type PromiseRequestSpec struct {
	Promise string                 `yaml:"promise" json:"promise"`
	Inputs  map[string]interface{} `yaml:"inputs" json:"inputs"`
}

// PromiseRequestStatus represents the status of a promise request
type PromiseRequestStatus struct {
	State       PromiseRequestState    `yaml:"state" json:"state"`
	Message     string                 `yaml:"message,omitempty" json:"message,omitempty"`
	Outputs     map[string]interface{} `yaml:"outputs,omitempty" json:"outputs,omitempty"`
	CreatedAt   time.Time              `yaml:"createdAt" json:"createdAt"`
	UpdatedAt   time.Time              `yaml:"updatedAt" json:"updatedAt"`
	CompletedAt *time.Time             `yaml:"completedAt,omitempty" json:"completedAt,omitempty"`
	ApprovalInfo *ApprovalInfo         `yaml:"approvalInfo,omitempty" json:"approvalInfo,omitempty"`
}

// PromiseRequestState represents the state of a promise request
type PromiseRequestState string

const (
	PromiseRequestStatePending         PromiseRequestState = "pending"
	PromiseRequestStateAwaitingApproval PromiseRequestState = "awaiting_approval"
	PromiseRequestStateApproved        PromiseRequestState = "approved"
	PromiseRequestStateRejected        PromiseRequestState = "rejected"
	PromiseRequestStateProvisioning    PromiseRequestState = "provisioning"
	PromiseRequestStateReady           PromiseRequestState = "ready"
	PromiseRequestStateFailed          PromiseRequestState = "failed"
	PromiseRequestStateDeleting        PromiseRequestState = "deleting"
	PromiseRequestStateDeleted         PromiseRequestState = "deleted"
)

// ApprovalInfo contains information about the approval process
type ApprovalInfo struct {
	Required    bool       `yaml:"required" json:"required"`
	RequestedAt *time.Time `yaml:"requestedAt,omitempty" json:"requestedAt,omitempty"`
	ApprovedAt  *time.Time `yaml:"approvedAt,omitempty" json:"approvedAt,omitempty"`
	ApprovedBy  string     `yaml:"approvedBy,omitempty" json:"approvedBy,omitempty"`
	RejectedAt  *time.Time `yaml:"rejectedAt,omitempty" json:"rejectedAt,omitempty"`
	RejectedBy  string     `yaml:"rejectedBy,omitempty" json:"rejectedBy,omitempty"`
	Reason      string     `yaml:"reason,omitempty" json:"reason,omitempty"`
}

// PromiseInstance represents a fulfilled promise instance
type PromiseInstance struct {
	Name        string                 `yaml:"name" json:"name"`
	Promise     string                 `yaml:"promise" json:"promise"`
	Team        string                 `yaml:"team" json:"team"`
	Environment string                 `yaml:"environment" json:"environment"`
	Inputs      map[string]interface{} `yaml:"inputs" json:"inputs"`
	Outputs     map[string]interface{} `yaml:"outputs" json:"outputs"`
	State       PromiseRequestState    `yaml:"state" json:"state"`
	CreatedAt   time.Time              `yaml:"createdAt" json:"createdAt"`
	UpdatedAt   time.Time              `yaml:"updatedAt" json:"updatedAt"`
}

var (
	promiseNameRegex = regexp.MustCompile(`^[a-z][a-z0-9-]*[a-z0-9]$`)
)

// Validate validates the promise resource
func (p *Promise) Validate() error {
	if p.APIVersion == "" {
		return ErrMissingAPIVersion
	}
	if p.Kind != "Promise" {
		return ErrInvalidKind
	}
	if p.Metadata.Name == "" {
		return ErrMissingName
	}

	// Validate promise name format
	if len(p.Metadata.Name) < 2 || len(p.Metadata.Name) > 63 {
		return fmt.Errorf("promise name must be between 2 and 63 characters")
	}
	if !promiseNameRegex.MatchString(p.Metadata.Name) {
		return fmt.Errorf("promise name must be lowercase alphanumeric with hyphens")
	}

	// Validate description
	if p.Spec.Description == "" {
		return fmt.Errorf("description is required")
	}

	// Validate provider
	if p.Spec.Provider == "" {
		return fmt.Errorf("provider is required")
	}

	// Validate category
	if p.Spec.Category == "" {
		return fmt.Errorf("category is required")
	}
	if !IsValidPromiseCategory(p.Spec.Category) {
		return fmt.Errorf("invalid category: %s", p.Spec.Category)
	}

	// Validate inputs
	if len(p.Spec.Inputs) > 50 {
		return fmt.Errorf("too many inputs (max 50)")
	}

	inputNames := make(map[string]bool)
	for i, input := range p.Spec.Inputs {
		if input.Name == "" {
			return fmt.Errorf("input %d: name is required", i)
		}
		if inputNames[input.Name] {
			return fmt.Errorf("input %d: duplicate input name '%s'", i, input.Name)
		}
		inputNames[input.Name] = true

		if input.Type == "" {
			return fmt.Errorf("input %s: type is required", input.Name)
		}
		if !IsValidInputType(input.Type) {
			return fmt.Errorf("input %s: invalid type '%s'", input.Name, input.Type)
		}

		if input.Type == "enum" && len(input.Enum) == 0 {
			return fmt.Errorf("input %s: enum values are required for enum type", input.Name)
		}

		if input.Validation != "" {
			if _, err := regexp.Compile(input.Validation); err != nil {
				return fmt.Errorf("input %s: invalid validation regex: %w", input.Name, err)
			}
		}
	}

	// Validate outputs
	if len(p.Spec.Outputs) > 50 {
		return fmt.Errorf("too many outputs (max 50)")
	}

	outputNames := make(map[string]bool)
	for i, output := range p.Spec.Outputs {
		if output.Name == "" {
			return fmt.Errorf("output %d: name is required", i)
		}
		if outputNames[output.Name] {
			return fmt.Errorf("output %d: duplicate output name '%s'", i, output.Name)
		}
		outputNames[output.Name] = true

		if output.Type == "" {
			return fmt.Errorf("output %s: type is required", output.Name)
		}
		if !IsValidOutputType(output.Type) {
			return fmt.Errorf("output %s: invalid type '%s'", output.Name, output.Type)
		}
	}

	return nil
}

// Validate validates a promise request
func (r *PromiseRequest) Validate() error {
	if r.APIVersion == "" {
		return ErrMissingAPIVersion
	}
	if r.Kind != "PromiseRequest" {
		return ErrInvalidKind
	}
	if r.Metadata.Name == "" {
		return ErrMissingName
	}

	// Validate name format
	if len(r.Metadata.Name) < 2 || len(r.Metadata.Name) > 63 {
		return fmt.Errorf("request name must be between 2 and 63 characters")
	}
	if !promiseNameRegex.MatchString(r.Metadata.Name) {
		return fmt.Errorf("request name must be lowercase alphanumeric with hyphens")
	}

	// Validate team
	if r.Metadata.Team == "" {
		return fmt.Errorf("team is required")
	}

	// Validate promise reference
	if r.Spec.Promise == "" {
		return fmt.Errorf("promise is required")
	}

	return nil
}

// ValidateInputs validates request inputs against promise definition
func (r *PromiseRequest) ValidateInputs(promise *Promise) error {
	// Check required inputs
	for _, input := range promise.Spec.Inputs {
		val, exists := r.Spec.Inputs[input.Name]

		if input.Required && !exists {
			return fmt.Errorf("required input '%s' is missing", input.Name)
		}

		if !exists {
			continue
		}

		// Validate type
		if err := validateInputValue(input, val); err != nil {
			return fmt.Errorf("input '%s': %w", input.Name, err)
		}
	}

	// Check for unknown inputs
	for name := range r.Spec.Inputs {
		found := false
		for _, input := range promise.Spec.Inputs {
			if input.Name == name {
				found = true
				break
			}
		}
		if !found {
			return fmt.Errorf("unknown input '%s'", name)
		}
	}

	return nil
}

// validateInputValue validates a single input value
func validateInputValue(input PromiseInput, value interface{}) error {
	switch input.Type {
	case "string":
		str, ok := value.(string)
		if !ok {
			return fmt.Errorf("expected string, got %T", value)
		}
		if input.Validation != "" {
			re := regexp.MustCompile(input.Validation)
			if !re.MatchString(str) {
				return fmt.Errorf("value does not match validation pattern")
			}
		}

	case "number":
		var num float64
		switch v := value.(type) {
		case float64:
			num = v
		case int:
			num = float64(v)
		case int64:
			num = float64(v)
		default:
			return fmt.Errorf("expected number, got %T", value)
		}
		if input.Min != nil && num < *input.Min {
			return fmt.Errorf("value %v is less than minimum %v", num, *input.Min)
		}
		if input.Max != nil && num > *input.Max {
			return fmt.Errorf("value %v is greater than maximum %v", num, *input.Max)
		}

	case "boolean":
		if _, ok := value.(bool); !ok {
			return fmt.Errorf("expected boolean, got %T", value)
		}

	case "enum":
		str, ok := value.(string)
		if !ok {
			return fmt.Errorf("expected string for enum, got %T", value)
		}
		valid := false
		for _, allowed := range input.Enum {
			if str == allowed {
				valid = true
				break
			}
		}
		if !valid {
			return fmt.Errorf("value '%s' is not in allowed values: %v", str, input.Enum)
		}
	}

	return nil
}

// PromiseCategory represents valid promise categories
type PromiseCategory string

const (
	PromiseCategoryDatabase   PromiseCategory = "database"
	PromiseCategoryCache      PromiseCategory = "cache"
	PromiseCategoryQueue      PromiseCategory = "queue"
	PromiseCategoryStorage    PromiseCategory = "storage"
	PromiseCategoryCompute    PromiseCategory = "compute"
	PromiseCategoryNetwork    PromiseCategory = "network"
	PromiseCategoryMonitoring PromiseCategory = "monitoring"
	PromiseCategoryIdentity   PromiseCategory = "identity"
	PromiseCategoryOther      PromiseCategory = "other"
)

// IsValidPromiseCategory checks if a category is valid
func IsValidPromiseCategory(category string) bool {
	validCategories := map[string]bool{
		"database":   true,
		"cache":      true,
		"queue":      true,
		"storage":    true,
		"compute":    true,
		"network":    true,
		"monitoring": true,
		"identity":   true,
		"other":      true,
	}
	return validCategories[category]
}

// IsValidInputType checks if an input type is valid
func IsValidInputType(inputType string) bool {
	validTypes := map[string]bool{
		"string":  true,
		"number":  true,
		"boolean": true,
		"enum":    true,
	}
	return validTypes[inputType]
}

// IsValidOutputType checks if an output type is valid
func IsValidOutputType(outputType string) bool {
	validTypes := map[string]bool{
		"string": true,
		"number": true,
		"secret": true,
	}
	return validTypes[outputType]
}

// GetFullName returns the fully qualified promise name
func (p *Promise) GetFullName() string {
	return p.Metadata.Name
}

// ApplyDefaults applies default values to a request's inputs
func (r *PromiseRequest) ApplyDefaults(promise *Promise) {
	if r.Spec.Inputs == nil {
		r.Spec.Inputs = make(map[string]interface{})
	}

	for _, input := range promise.Spec.Inputs {
		if _, exists := r.Spec.Inputs[input.Name]; !exists && input.Default != nil {
			r.Spec.Inputs[input.Name] = input.Default
		}
	}
}

// NeedsApproval checks if a request needs approval
func (r *PromiseRequest) NeedsApproval(promise *Promise) bool {
	if promise.Spec.Approval == nil || !promise.Spec.Approval.Required {
		return false
	}

	// Check if approval is only required for certain environments
	if len(promise.Spec.Approval.Environments) > 0 {
		for _, env := range promise.Spec.Approval.Environments {
			if env == r.Metadata.Environment {
				return true
			}
		}
		return false
	}

	return true
}
