package handlers

import (
	"context"
	"fmt"
	"sync"

	"github.com/platformfoundry/platformfoundry-ce/internal/workflow"
	"github.com/platformfoundry/platformfoundry-ce/internal/workflow/dag"
)

// Handler defines the interface for workflow step handlers
type Handler interface {
	// Type returns the step type this handler handles
	Type() workflow.StepType

	// Validate validates the step configuration
	Validate(config map[string]interface{}) error

	// Execute executes the step and returns the result
	Execute(ctx context.Context, step *workflow.StepExecution, config map[string]interface{}, resolver dag.OutputResolver) (*workflow.StepResult, error)
}

// Registry manages step handlers
type Registry struct {
	handlers map[workflow.StepType]Handler
	mu       sync.RWMutex
}

// NewRegistry creates a new handler registry
func NewRegistry() *Registry {
	return &Registry{
		handlers: make(map[workflow.StepType]Handler),
	}
}

// Register registers a handler for a step type
func (r *Registry) Register(handler Handler) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	stepType := handler.Type()
	if _, exists := r.handlers[stepType]; exists {
		return fmt.Errorf("handler for step type %s already registered", stepType)
	}

	r.handlers[stepType] = handler
	return nil
}

// Get returns the handler for a step type
func (r *Registry) Get(stepType workflow.StepType) (Handler, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	handler, ok := r.handlers[stepType]
	return handler, ok
}

// Unregister removes a handler
func (r *Registry) Unregister(stepType workflow.StepType) {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.handlers, stepType)
}

// Types returns all registered step types
func (r *Registry) Types() []workflow.StepType {
	r.mu.RLock()
	defer r.mu.RUnlock()

	types := make([]workflow.StepType, 0, len(r.handlers))
	for t := range r.handlers {
		types = append(types, t)
	}
	return types
}

// DefaultRegistry creates a registry with all default handlers
func DefaultRegistry() *Registry {
	registry := NewRegistry()

	// Register default handlers
	registry.Register(NewShellHandler())
	registry.Register(NewHTTPHandler())
	registry.Register(NewNotifyHandler(nil))
	registry.Register(NewApprovalHandler())

	return registry
}

// BaseHandler provides common functionality for handlers
type BaseHandler struct {
	stepType workflow.StepType
}

// Type returns the step type
func (h *BaseHandler) Type() workflow.StepType {
	return h.stepType
}

// GetStringConfig retrieves a string value from config
func GetStringConfig(config map[string]interface{}, key string, defaultValue string) string {
	if val, ok := config[key]; ok {
		if str, ok := val.(string); ok {
			return str
		}
	}
	return defaultValue
}

// GetIntConfig retrieves an int value from config
func GetIntConfig(config map[string]interface{}, key string, defaultValue int) int {
	if val, ok := config[key]; ok {
		switch v := val.(type) {
		case int:
			return v
		case int64:
			return int(v)
		case float64:
			return int(v)
		}
	}
	return defaultValue
}

// GetBoolConfig retrieves a bool value from config
func GetBoolConfig(config map[string]interface{}, key string, defaultValue bool) bool {
	if val, ok := config[key]; ok {
		if b, ok := val.(bool); ok {
			return b
		}
	}
	return defaultValue
}

// GetStringSliceConfig retrieves a string slice from config
func GetStringSliceConfig(config map[string]interface{}, key string) []string {
	if val, ok := config[key]; ok {
		switch v := val.(type) {
		case []string:
			return v
		case []interface{}:
			result := make([]string, 0, len(v))
			for _, item := range v {
				if str, ok := item.(string); ok {
					result = append(result, str)
				}
			}
			return result
		}
	}
	return nil
}

// GetMapConfig retrieves a map from config
func GetMapConfig(config map[string]interface{}, key string) map[string]interface{} {
	if val, ok := config[key]; ok {
		if m, ok := val.(map[string]interface{}); ok {
			return m
		}
	}
	return nil
}
