package handlers

import (
	"context"
	"fmt"
	"time"

	"github.com/platformfoundry/platformfoundry-ce/internal/policy"
	"github.com/platformfoundry/platformfoundry-ce/internal/workflow"
	"github.com/platformfoundry/platformfoundry-ce/internal/workflow/dag"
)

// PolicyHandler evaluates OPA policies
type PolicyHandler struct {
	BaseHandler
	engine policy.Engine
}

// NewPolicyHandler creates a new policy handler
func NewPolicyHandler(engine policy.Engine) *PolicyHandler {
	return &PolicyHandler{
		BaseHandler: BaseHandler{stepType: workflow.StepTypePolicy},
		engine:      engine,
	}
}

// Validate validates the policy step configuration
func (h *PolicyHandler) Validate(config map[string]interface{}) error {
	policyName := GetStringConfig(config, "policy", "")
	if policyName == "" {
		return fmt.Errorf("policy step requires 'policy' configuration")
	}

	return nil
}

// Execute evaluates the policy
func (h *PolicyHandler) Execute(ctx context.Context, step *workflow.StepExecution, config map[string]interface{}, resolver dag.OutputResolver) (*workflow.StepResult, error) {
	result := &workflow.StepResult{
		Status:  workflow.StepStatusRunning,
		Outputs: make(map[string]interface{}),
		Logs:    make([]workflow.StepLog, 0),
	}

	policyName := GetStringConfig(config, "policy", "")
	input := GetMapConfig(config, "input")
	if input == nil {
		input = make(map[string]interface{})
	}
	failOnDeny := GetBoolConfig(config, "failOnDeny", true)

	result.Logs = append(result.Logs, workflow.StepLog{
		Time:    time.Now(),
		Level:   "info",
		Message: fmt.Sprintf("Evaluating policy: %s", policyName),
	})

	if h.engine == nil {
		result.Logs = append(result.Logs, workflow.StepLog{
			Time:    time.Now(),
			Level:   "warn",
			Message: "No policy engine configured - assuming policy passes",
		})
		result.Status = workflow.StepStatusCompleted
		result.Outputs["allowed"] = true
		result.Outputs["simulated"] = true
		return result, nil
	}

	// Evaluate policy
	startTime := time.Now()
	policyResult, err := h.engine.Evaluate(ctx, policyName, input)
	duration := time.Since(startTime)

	result.Outputs["duration_ms"] = duration.Milliseconds()

	if err != nil {
		result.Status = workflow.StepStatusFailed
		result.ErrorMsg = fmt.Sprintf("policy evaluation failed: %v", err)
		result.Logs = append(result.Logs, workflow.StepLog{
			Time:    time.Now(),
			Level:   "error",
			Message: result.ErrorMsg,
		})
		return result, fmt.Errorf("%s", result.ErrorMsg)
	}

	result.Outputs["allowed"] = policyResult.Allowed
	result.Outputs["reasons"] = policyResult.Reasons
	result.Outputs["data"] = policyResult.Data

	if policyResult.Allowed {
		result.Status = workflow.StepStatusCompleted
		result.Logs = append(result.Logs, workflow.StepLog{
			Time:    time.Now(),
			Level:   "info",
			Message: fmt.Sprintf("Policy %s passed in %v", policyName, duration),
		})
	} else {
		if failOnDeny {
			result.Status = workflow.StepStatusFailed
			result.ErrorMsg = fmt.Sprintf("policy %s denied: %v", policyName, policyResult.Reasons)
			result.Logs = append(result.Logs, workflow.StepLog{
				Time:    time.Now(),
				Level:   "error",
				Message: result.ErrorMsg,
			})
			return result, fmt.Errorf("%s", result.ErrorMsg)
		}

		// Policy denied but we don't fail
		result.Status = workflow.StepStatusCompleted
		result.Logs = append(result.Logs, workflow.StepLog{
			Time:    time.Now(),
			Level:   "warn",
			Message: fmt.Sprintf("Policy %s denied (non-fatal): %v", policyName, policyResult.Reasons),
		})
	}

	return result, nil
}
