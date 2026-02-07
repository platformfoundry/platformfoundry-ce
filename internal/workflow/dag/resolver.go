package dag

import (
	"context"
	"fmt"
	"regexp"
	"strings"
	"sync"
)

// Resolver resolves template variables in workflow step configurations
type Resolver struct {
	inputs       map[string]interface{}
	stepOutputs  map[string]map[string]interface{}
	stepStatuses map[string]string
	mu           sync.RWMutex
}

// NewResolver creates a new resolver with the given workflow inputs
func NewResolver(inputs map[string]interface{}) *Resolver {
	if inputs == nil {
		inputs = make(map[string]interface{})
	}
	return &Resolver{
		inputs:       inputs,
		stepOutputs:  make(map[string]map[string]interface{}),
		stepStatuses: make(map[string]string),
	}
}

// SetInput sets an input value
func (r *Resolver) SetInput(key string, value interface{}) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.inputs[key] = value
}

// GetInput retrieves an input value
func (r *Resolver) GetInput(key string) (interface{}, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	val, ok := r.inputs[key]
	return val, ok
}

// SetStepOutputs sets the outputs for a step
func (r *Resolver) SetStepOutputs(stepID string, outputs map[string]interface{}) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.stepOutputs[stepID] = outputs
}

// GetStepOutput retrieves a specific output from a step
func (r *Resolver) GetStepOutput(stepID, key string) (interface{}, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	if outputs, ok := r.stepOutputs[stepID]; ok {
		val, exists := outputs[key]
		return val, exists
	}
	return nil, false
}

// GetAllStepOutputs returns all outputs for a step
func (r *Resolver) GetAllStepOutputs(stepID string) (map[string]interface{}, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	outputs, ok := r.stepOutputs[stepID]
	if !ok {
		return nil, false
	}
	// Return a copy
	result := make(map[string]interface{})
	for k, v := range outputs {
		result[k] = v
	}
	return result, true
}

// SetStepStatus sets the status for a step
func (r *Resolver) SetStepStatus(stepID, status string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.stepStatuses[stepID] = status
}

// GetStepStatus retrieves the status of a step
func (r *Resolver) GetStepStatus(stepID string) (string, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	status, ok := r.stepStatuses[stepID]
	return status, ok
}

// Template variable patterns
var (
	// ${inputs.name}
	inputPattern = regexp.MustCompile(`\$\{inputs\.([a-zA-Z_][a-zA-Z0-9_]*)\}`)
	// ${steps.stepId.outputs.key}
	stepOutputPattern = regexp.MustCompile(`\$\{steps\.([a-zA-Z_][a-zA-Z0-9_-]*)\.outputs\.([a-zA-Z_][a-zA-Z0-9_]*)\}`)
	// ${steps.stepId.status}
	stepStatusPattern = regexp.MustCompile(`\$\{steps\.([a-zA-Z_][a-zA-Z0-9_-]*)\.status\}`)
	// ${env.VAR_NAME}
	envPattern = regexp.MustCompile(`\$\{env\.([a-zA-Z_][a-zA-Z0-9_]*)\}`)
)

// Resolve resolves template variables in a string
func (r *Resolver) Resolve(ctx context.Context, template string) (string, error) {
	if template == "" {
		return template, nil
	}

	result := template

	// Resolve input references
	result = inputPattern.ReplaceAllStringFunc(result, func(match string) string {
		matches := inputPattern.FindStringSubmatch(match)
		if len(matches) < 2 {
			return match
		}
		key := matches[1]
		if val, ok := r.GetInput(key); ok {
			return fmt.Sprintf("%v", val)
		}
		return match
	})

	// Resolve step output references
	result = stepOutputPattern.ReplaceAllStringFunc(result, func(match string) string {
		matches := stepOutputPattern.FindStringSubmatch(match)
		if len(matches) < 3 {
			return match
		}
		stepID := matches[1]
		key := matches[2]
		if val, ok := r.GetStepOutput(stepID, key); ok {
			return fmt.Sprintf("%v", val)
		}
		return match
	})

	// Resolve step status references
	result = stepStatusPattern.ReplaceAllStringFunc(result, func(match string) string {
		matches := stepStatusPattern.FindStringSubmatch(match)
		if len(matches) < 2 {
			return match
		}
		stepID := matches[1]
		if status, ok := r.GetStepStatus(stepID); ok {
			return status
		}
		return match
	})

	return result, nil
}

// ResolveMap resolves template variables in a map recursively
func (r *Resolver) ResolveMap(ctx context.Context, data map[string]interface{}) (map[string]interface{}, error) {
	if data == nil {
		return nil, nil
	}

	result := make(map[string]interface{})
	for key, value := range data {
		resolved, err := r.resolveValue(ctx, value)
		if err != nil {
			return nil, fmt.Errorf("failed to resolve %s: %w", key, err)
		}
		result[key] = resolved
	}
	return result, nil
}

// resolveValue resolves a single value which can be string, map, or slice
func (r *Resolver) resolveValue(ctx context.Context, value interface{}) (interface{}, error) {
	switch v := value.(type) {
	case string:
		return r.Resolve(ctx, v)

	case map[string]interface{}:
		return r.ResolveMap(ctx, v)

	case map[interface{}]interface{}:
		// Convert to map[string]interface{} first
		strMap := make(map[string]interface{})
		for k, val := range v {
			strKey := fmt.Sprintf("%v", k)
			strMap[strKey] = val
		}
		return r.ResolveMap(ctx, strMap)

	case []interface{}:
		result := make([]interface{}, len(v))
		for i, item := range v {
			resolved, err := r.resolveValue(ctx, item)
			if err != nil {
				return nil, err
			}
			result[i] = resolved
		}
		return result, nil

	default:
		return value, nil
	}
}

// ResolveCondition evaluates a condition expression
// Supports: ${steps.X.status} == "completed", ${inputs.Y} == "value", etc.
func (r *Resolver) ResolveCondition(ctx context.Context, condition string) (bool, error) {
	// First resolve all variables
	resolved, err := r.Resolve(ctx, condition)
	if err != nil {
		return false, err
	}

	// Handle simple boolean values
	resolved = strings.TrimSpace(resolved)
	switch strings.ToLower(resolved) {
	case "true", "1", "yes":
		return true, nil
	case "false", "0", "no", "":
		return false, nil
	}

	// Handle comparison expressions
	// Format: "value1 == value2" or "value1 != value2"
	if strings.Contains(resolved, "==") {
		parts := strings.SplitN(resolved, "==", 2)
		if len(parts) == 2 {
			left := strings.TrimSpace(parts[0])
			right := strings.TrimSpace(parts[1])
			// Remove quotes if present
			left = strings.Trim(left, `"'`)
			right = strings.Trim(right, `"'`)
			return left == right, nil
		}
	}

	if strings.Contains(resolved, "!=") {
		parts := strings.SplitN(resolved, "!=", 2)
		if len(parts) == 2 {
			left := strings.TrimSpace(parts[0])
			right := strings.TrimSpace(parts[1])
			left = strings.Trim(left, `"'`)
			right = strings.Trim(right, `"'`)
			return left != right, nil
		}
	}

	// Non-empty resolved value is truthy
	return resolved != "", nil
}

// Clone creates a copy of the resolver
func (r *Resolver) Clone() *Resolver {
	r.mu.RLock()
	defer r.mu.RUnlock()

	newInputs := make(map[string]interface{})
	for k, v := range r.inputs {
		newInputs[k] = v
	}

	newOutputs := make(map[string]map[string]interface{})
	for stepID, outputs := range r.stepOutputs {
		newOutputs[stepID] = make(map[string]interface{})
		for k, v := range outputs {
			newOutputs[stepID][k] = v
		}
	}

	newStatuses := make(map[string]string)
	for k, v := range r.stepStatuses {
		newStatuses[k] = v
	}

	return &Resolver{
		inputs:       newInputs,
		stepOutputs:  newOutputs,
		stepStatuses: newStatuses,
	}
}
