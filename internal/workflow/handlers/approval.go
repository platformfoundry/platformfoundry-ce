package handlers

import (
	"context"
	"fmt"
	"time"

	"github.com/platformfoundry/pf-ce/internal/workflow"
	"github.com/platformfoundry/pf-ce/internal/workflow/dag"
)

// ApprovalHandler handles human approval steps
type ApprovalHandler struct {
	BaseHandler
	approvalStore ApprovalStore
}

// ApprovalStore interface for managing approvals
type ApprovalStore interface {
	// RequestApproval creates an approval request
	RequestApproval(ctx context.Context, req *ApprovalRequest) error
	// GetApprovalStatus checks the status of an approval
	GetApprovalStatus(ctx context.Context, stepExecutionID string) (*ApprovalStatus, error)
	// WaitForApproval waits for an approval decision
	WaitForApproval(ctx context.Context, stepExecutionID string, timeout time.Duration) (*ApprovalStatus, error)
}

// ApprovalRequest represents an approval request
type ApprovalRequest struct {
	StepExecutionID string
	WorkflowName    string
	StepName        string
	Requester       string
	Message         string
	RequiredRoles   []string
	RequiredCount   int
	Timeout         time.Duration
	CreatedAt       time.Time
}

// ApprovalStatus represents the status of an approval
type ApprovalStatus struct {
	StepExecutionID string
	Status          string // pending, approved, rejected
	Approvals       []ApprovalRecord
	RejectedBy      string
	RejectedReason  string
	CompletedAt     *time.Time
}

// ApprovalRecord represents a single approval
type ApprovalRecord struct {
	User      string
	Role      string
	Decision  string // approved, rejected
	Comment   string
	Timestamp time.Time
}

// NewApprovalHandler creates a new approval handler
func NewApprovalHandler() *ApprovalHandler {
	return &ApprovalHandler{
		BaseHandler: BaseHandler{stepType: workflow.StepTypeApproval},
	}
}

// NewApprovalHandlerWithStore creates an approval handler with a store
func NewApprovalHandlerWithStore(store ApprovalStore) *ApprovalHandler {
	return &ApprovalHandler{
		BaseHandler:   BaseHandler{stepType: workflow.StepTypeApproval},
		approvalStore: store,
	}
}

// Validate validates the approval step configuration
func (h *ApprovalHandler) Validate(config map[string]interface{}) error {
	// Approval steps can have optional configuration
	return nil
}

// Execute handles the approval step
func (h *ApprovalHandler) Execute(ctx context.Context, step *workflow.StepExecution, config map[string]interface{}, resolver dag.OutputResolver) (*workflow.StepResult, error) {
	result := &workflow.StepResult{
		Status:  workflow.StepStatusRunning,
		Outputs: make(map[string]interface{}),
		Logs:    make([]workflow.StepLog, 0),
	}

	// Get configuration
	message := GetStringConfig(config, "message", "Approval required to continue workflow")
	requiredRoles := GetStringSliceConfig(config, "roles")
	requiredCount := GetIntConfig(config, "required", 1)
	timeoutMinutes := GetIntConfig(config, "timeout", 60) // Default 1 hour
	timeout := time.Duration(timeoutMinutes) * time.Minute

	result.Logs = append(result.Logs, workflow.StepLog{
		Time:    time.Now(),
		Level:   "info",
		Message: fmt.Sprintf("Waiting for approval: %s (requires %d from %v)", message, requiredCount, requiredRoles),
	})

	// If no store is configured, just log and complete (for testing/demo)
	if h.approvalStore == nil {
		result.Logs = append(result.Logs, workflow.StepLog{
			Time:    time.Now(),
			Level:   "warn",
			Message: "No approval store configured - auto-approving for demo/test mode",
		})
		result.Status = workflow.StepStatusCompleted
		result.Outputs["approved"] = true
		result.Outputs["autoApproved"] = true
		return result, nil
	}

	// Create approval request
	req := &ApprovalRequest{
		StepExecutionID: step.ID,
		StepName:        step.StepID,
		Message:         message,
		RequiredRoles:   requiredRoles,
		RequiredCount:   requiredCount,
		Timeout:         timeout,
		CreatedAt:       time.Now(),
	}

	if err := h.approvalStore.RequestApproval(ctx, req); err != nil {
		result.Status = workflow.StepStatusFailed
		result.ErrorMsg = fmt.Sprintf("failed to create approval request: %v", err)
		return result, fmt.Errorf("%s", result.ErrorMsg)
	}

	// Wait for approval
	status, err := h.approvalStore.WaitForApproval(ctx, step.ID, timeout)
	if err != nil {
		result.Status = workflow.StepStatusFailed
		result.ErrorMsg = fmt.Sprintf("approval wait failed: %v", err)
		return result, fmt.Errorf("%s", result.ErrorMsg)
	}

	// Process result
	result.Outputs["status"] = status.Status
	result.Outputs["approvals"] = len(status.Approvals)

	if status.Status == "approved" {
		result.Status = workflow.StepStatusCompleted
		result.Outputs["approved"] = true
		result.Logs = append(result.Logs, workflow.StepLog{
			Time:    time.Now(),
			Level:   "info",
			Message: fmt.Sprintf("Approval granted by %d approvers", len(status.Approvals)),
		})
	} else if status.Status == "rejected" {
		result.Status = workflow.StepStatusFailed
		result.Outputs["approved"] = false
		result.Outputs["rejectedBy"] = status.RejectedBy
		result.Outputs["rejectedReason"] = status.RejectedReason
		result.ErrorMsg = fmt.Sprintf("Approval rejected by %s: %s", status.RejectedBy, status.RejectedReason)
		result.Logs = append(result.Logs, workflow.StepLog{
			Time:    time.Now(),
			Level:   "error",
			Message: result.ErrorMsg,
		})
		return result, fmt.Errorf("%s", result.ErrorMsg)
	} else {
		// Timeout or other
		result.Status = workflow.StepStatusFailed
		result.ErrorMsg = fmt.Sprintf("Approval timed out after %v", timeout)
		result.Logs = append(result.Logs, workflow.StepLog{
			Time:    time.Now(),
			Level:   "error",
			Message: result.ErrorMsg,
		})
		return result, fmt.Errorf("%s", result.ErrorMsg)
	}

	return result, nil
}
