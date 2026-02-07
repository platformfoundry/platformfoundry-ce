package workflow

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

// Parser parses workflow YAML files
type Parser struct{}

// NewParser creates a new workflow parser
func NewParser() *Parser {
	return &Parser{}
}

// ParseFile parses a workflow from a file
func (p *Parser) ParseFile(path string) (*DAGWorkflow, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read file %s: %w", path, err)
	}

	return p.Parse(data)
}

// Parse parses a workflow from YAML bytes
func (p *Parser) Parse(data []byte) (*DAGWorkflow, error) {
	var wf DAGWorkflow
	if err := yaml.Unmarshal(data, &wf); err != nil {
		return nil, fmt.Errorf("failed to parse YAML: %w", err)
	}

	// Validate and set defaults
	if err := p.validate(&wf); err != nil {
		return nil, err
	}

	return &wf, nil
}

// ParseDir parses all workflows from a directory
func (p *Parser) ParseDir(dir string) ([]*DAGWorkflow, error) {
	var workflows []*DAGWorkflow

	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if info.IsDir() {
			return nil
		}

		ext := strings.ToLower(filepath.Ext(path))
		if ext != ".yaml" && ext != ".yml" {
			return nil
		}

		wf, err := p.ParseFile(path)
		if err != nil {
			return fmt.Errorf("failed to parse %s: %w", path, err)
		}

		workflows = append(workflows, wf)
		return nil
	})

	if err != nil {
		return nil, err
	}

	return workflows, nil
}

// validate validates and sets defaults for a workflow
func (p *Parser) validate(wf *DAGWorkflow) error {
	// Check required fields
	if wf.Metadata.Name == "" {
		return fmt.Errorf("workflow metadata.name is required")
	}

	// Set defaults for apiVersion and kind
	if wf.APIVersion == "" {
		wf.APIVersion = "platformfoundry.io/v1"
	}
	if wf.Kind == "" {
		wf.Kind = "Workflow"
	}

	// Validate kind
	if wf.Kind != "Workflow" {
		return fmt.Errorf("invalid kind: %s (expected Workflow)", wf.Kind)
	}

	// Validate steps
	if len(wf.Spec.Steps) == 0 {
		return fmt.Errorf("workflow must have at least one step")
	}

	// Track step IDs for dependency validation
	stepIDs := make(map[string]bool)
	for _, step := range wf.Spec.Steps {
		if step.ID == "" {
			return fmt.Errorf("step must have an id")
		}
		if stepIDs[step.ID] {
			return fmt.Errorf("duplicate step id: %s", step.ID)
		}
		stepIDs[step.ID] = true

		// Validate step type
		if err := p.validateStepType(step.Type); err != nil {
			return fmt.Errorf("step %s: %w", step.ID, err)
		}
	}

	// Validate dependencies
	for _, step := range wf.Spec.Steps {
		for _, dep := range step.DependsOn {
			if !stepIDs[dep] {
				return fmt.Errorf("step %s depends on non-existent step: %s", step.ID, dep)
			}
		}
	}

	// Validate timeout format
	if wf.Spec.Timeout != "" {
		if _, err := time.ParseDuration(wf.Spec.Timeout); err != nil {
			return fmt.Errorf("invalid timeout format: %s", wf.Spec.Timeout)
		}
	}

	// Validate inputs
	for _, input := range wf.Spec.Inputs {
		if input.Name == "" {
			return fmt.Errorf("input must have a name")
		}
		if input.Type == "" {
			input.Type = "string"
		}
	}

	// Validate triggers
	for i, trigger := range wf.Spec.Triggers {
		if trigger.Type == "" {
			return fmt.Errorf("trigger %d must have a type", i)
		}

		validTypes := map[string]bool{
			"manual": true, "schedule": true, "webhook": true, "event": true,
		}
		if !validTypes[trigger.Type] {
			return fmt.Errorf("invalid trigger type: %s", trigger.Type)
		}

		// Validate cron expression for schedule triggers
		if trigger.Type == "schedule" && trigger.Cron == "" {
			return fmt.Errorf("schedule trigger must have a cron expression")
		}
	}

	return nil
}

// validateStepType validates a step type
func (p *Parser) validateStepType(stepType StepType) error {
	validTypes := map[StepType]bool{
		StepTypeShell:    true,
		StepTypeHTTP:     true,
		StepTypeInfra:    true,
		StepTypePolicy:   true,
		StepTypeSecrets:  true,
		StepTypeNotify:   true,
		StepTypeApproval: true,
	}

	if !validTypes[stepType] {
		return fmt.Errorf("invalid step type: %s", stepType)
	}

	return nil
}

// Marshal serializes a workflow to YAML
func (p *Parser) Marshal(wf *DAGWorkflow) ([]byte, error) {
	return yaml.Marshal(wf)
}

// ValidateInputs validates workflow inputs against the input spec
func ValidateInputs(wf *DAGWorkflow, inputs map[string]interface{}) error {
	for _, spec := range wf.Spec.Inputs {
		value, exists := inputs[spec.Name]

		// Check required inputs
		if spec.Required && !exists {
			if spec.Default == nil {
				return fmt.Errorf("required input %s is missing", spec.Name)
			}
			continue
		}

		if !exists {
			continue
		}

		// Type validation
		if err := validateInputType(spec.Name, spec.Type, value); err != nil {
			return err
		}

		// Enum validation
		if len(spec.Enum) > 0 {
			strVal, ok := value.(string)
			if !ok {
				return fmt.Errorf("input %s must be a string for enum validation", spec.Name)
			}

			valid := false
			for _, enumVal := range spec.Enum {
				if strVal == enumVal {
					valid = true
					break
				}
			}
			if !valid {
				return fmt.Errorf("input %s must be one of: %v", spec.Name, spec.Enum)
			}
		}
	}

	return nil
}

// validateInputType validates the type of an input value
func validateInputType(name, inputType string, value interface{}) error {
	switch inputType {
	case "string":
		if _, ok := value.(string); !ok {
			return fmt.Errorf("input %s must be a string", name)
		}
	case "number":
		switch value.(type) {
		case int, int32, int64, float32, float64:
			// OK
		default:
			return fmt.Errorf("input %s must be a number", name)
		}
	case "boolean":
		if _, ok := value.(bool); !ok {
			return fmt.Errorf("input %s must be a boolean", name)
		}
	case "array":
		if _, ok := value.([]interface{}); !ok {
			return fmt.Errorf("input %s must be an array", name)
		}
	case "object":
		if _, ok := value.(map[string]interface{}); !ok {
			return fmt.Errorf("input %s must be an object", name)
		}
	}

	return nil
}

// ApplyDefaults applies default values to inputs
func ApplyDefaults(wf *DAGWorkflow, inputs map[string]interface{}) map[string]interface{} {
	result := make(map[string]interface{})

	// Copy provided inputs
	for k, v := range inputs {
		result[k] = v
	}

	// Apply defaults for missing inputs
	for _, spec := range wf.Spec.Inputs {
		if _, exists := result[spec.Name]; !exists && spec.Default != nil {
			result[spec.Name] = spec.Default
		}
	}

	return result
}
