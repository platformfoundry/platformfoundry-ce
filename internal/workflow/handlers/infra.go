package handlers

import (
	"context"
	"fmt"
	"time"

	"github.com/platformfoundry/pf-ce/internal/workflow"
	"github.com/platformfoundry/pf-ce/internal/workflow/dag"
)

// InfraHandler executes infrastructure operations via the engine coordinator
type InfraHandler struct {
	BaseHandler
	coordinator InfraCoordinator
}

// InfraCoordinator interface for infrastructure operations
type InfraCoordinator interface {
	// Apply applies infrastructure changes
	Apply(ctx context.Context, specs map[string]map[string]interface{}) error
	// Plan generates an execution plan
	Plan(ctx context.Context, specs map[string]map[string]interface{}) (map[string]interface{}, error)
	// GetStatus returns status of operations
	GetStatus() map[string]interface{}
}

// NewInfraHandler creates a new infrastructure handler
func NewInfraHandler(coordinator InfraCoordinator) *InfraHandler {
	return &InfraHandler{
		BaseHandler: BaseHandler{stepType: workflow.StepTypeInfra},
		coordinator: coordinator,
	}
}

// Validate validates the infra step configuration
func (h *InfraHandler) Validate(config map[string]interface{}) error {
	operation := GetStringConfig(config, "operation", "")
	if operation == "" {
		return fmt.Errorf("infra step requires 'operation' configuration (apply, plan)")
	}

	validOps := map[string]bool{"apply": true, "plan": true, "status": true}
	if !validOps[operation] {
		return fmt.Errorf("invalid infra operation: %s (valid: apply, plan, status)", operation)
	}

	return nil
}

// Execute performs the infrastructure operation
func (h *InfraHandler) Execute(ctx context.Context, step *workflow.StepExecution, config map[string]interface{}, resolver dag.OutputResolver) (*workflow.StepResult, error) {
	result := &workflow.StepResult{
		Status:  workflow.StepStatusRunning,
		Outputs: make(map[string]interface{}),
		Logs:    make([]workflow.StepLog, 0),
	}

	operation := GetStringConfig(config, "operation", "apply")
	specs := GetMapConfig(config, "specs")
	if specs == nil {
		specs = make(map[string]interface{})
	}

	result.Logs = append(result.Logs, workflow.StepLog{
		Time:    time.Now(),
		Level:   "info",
		Message: fmt.Sprintf("Starting infra operation: %s", operation),
	})

	if h.coordinator == nil {
		result.Logs = append(result.Logs, workflow.StepLog{
			Time:    time.Now(),
			Level:   "warn",
			Message: "No infra coordinator configured - simulating operation",
		})
		result.Status = workflow.StepStatusCompleted
		result.Outputs["operation"] = operation
		result.Outputs["simulated"] = true
		return result, nil
	}

	// Convert specs to expected format
	engineSpecs := make(map[string]map[string]interface{})
	for key, val := range specs {
		if specMap, ok := val.(map[string]interface{}); ok {
			engineSpecs[key] = specMap
		}
	}

	switch operation {
	case "apply":
		startTime := time.Now()
		err := h.coordinator.Apply(ctx, engineSpecs)
		duration := time.Since(startTime)

		result.Outputs["duration_ms"] = duration.Milliseconds()

		if err != nil {
			result.Status = workflow.StepStatusFailed
			result.ErrorMsg = fmt.Sprintf("infra apply failed: %v", err)
			result.Logs = append(result.Logs, workflow.StepLog{
				Time:    time.Now(),
				Level:   "error",
				Message: result.ErrorMsg,
			})
			return result, fmt.Errorf("%s", result.ErrorMsg)
		}

		result.Outputs["status"] = h.coordinator.GetStatus()
		result.Logs = append(result.Logs, workflow.StepLog{
			Time:    time.Now(),
			Level:   "info",
			Message: fmt.Sprintf("Infra apply completed in %v", duration),
		})

	case "plan":
		plan, err := h.coordinator.Plan(ctx, engineSpecs)
		if err != nil {
			result.Status = workflow.StepStatusFailed
			result.ErrorMsg = fmt.Sprintf("infra plan failed: %v", err)
			return result, fmt.Errorf("%s", result.ErrorMsg)
		}

		result.Outputs["plan"] = plan
		result.Logs = append(result.Logs, workflow.StepLog{
			Time:    time.Now(),
			Level:   "info",
			Message: "Infra plan generated",
		})

	case "status":
		result.Outputs["status"] = h.coordinator.GetStatus()
		result.Logs = append(result.Logs, workflow.StepLog{
			Time:    time.Now(),
			Level:   "info",
			Message: "Retrieved infra status",
		})
	}

	result.Status = workflow.StepStatusCompleted
	result.Outputs["operation"] = operation

	return result, nil
}
