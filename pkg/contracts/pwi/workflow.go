// Package pwi defines the Platform Workflow Interface (PWI).
// This handles approval and workflow engine abstraction.
package pwi

import (
	"context"
	"time"
)

// WorkflowEngine is the main interface for workflow management.
// Implementations handle workflow definition, execution, and approvals.
type WorkflowEngine interface {
	// Workflow Management
	CreateWorkflow(ctx context.Context, def *WorkflowDefinition) (*Workflow, error)
	GetWorkflow(ctx context.Context, id string) (*Workflow, error)
	ListWorkflows(ctx context.Context, filter *WorkflowFilter) ([]*Workflow, error)
	DeleteWorkflow(ctx context.Context, id string) error

	// Execution
	StartWorkflow(ctx context.Context, id string, input map[string]interface{}) (*WorkflowExecution, error)
	GetExecution(ctx context.Context, executionID string) (*WorkflowExecution, error)
	CancelExecution(ctx context.Context, executionID string, reason string) error

	// Step Operations
	ExecuteStep(ctx context.Context, workflowID string, stepID string, input interface{}) (*StepResult, error)
	RetryStep(ctx context.Context, executionID string, stepID string) error
	SkipStep(ctx context.Context, executionID string, stepID string, reason string) error

	// Approvals
	Approve(ctx context.Context, workflowID string, approverID string, decision Decision) error
	Reject(ctx context.Context, workflowID string, approverID string, reason string) error
	GetPendingApprovals(ctx context.Context, approverID string) ([]PendingApproval, error)

	// Close releases workflow engine resources
	Close() error
}

// WorkflowDefinition defines a workflow structure
type WorkflowDefinition struct {
	// Name is the workflow name
	Name string

	// Description describes the workflow
	Description string

	// Version is the workflow version
	Version string

	// Steps defines the workflow steps
	Steps []StepDefinition

	// Triggers defines what triggers this workflow
	Triggers []Trigger

	// Timeout is the maximum workflow duration
	Timeout time.Duration

	// RetryPolicy defines retry behavior
	RetryPolicy *RetryPolicy

	// Metadata contains additional workflow metadata
	Metadata map[string]string
}

// StepDefinition defines a single workflow step
type StepDefinition struct {
	// ID is the unique step identifier
	ID string

	// Name is the step name
	Name string

	// Type is the step type
	Type StepType

	// Config contains step-specific configuration
	Config map[string]interface{}

	// DependsOn lists step IDs this step depends on
	DependsOn []string

	// Condition is evaluated to determine if the step should run
	Condition string

	// Timeout is the maximum step duration
	Timeout time.Duration

	// RetryPolicy defines retry behavior for this step
	RetryPolicy *RetryPolicy

	// ApprovalConfig is set for approval steps
	ApprovalConfig *ApprovalConfig
}

// StepType represents the type of workflow step
type StepType string

const (
	StepTypeAction    StepType = "action"
	StepTypeApproval  StepType = "approval"
	StepTypeWait      StepType = "wait"
	StepTypeCondition StepType = "condition"
	StepTypeParallel  StepType = "parallel"
	StepTypeSubflow   StepType = "subflow"
)

// ApprovalConfig configures approval step behavior
type ApprovalConfig struct {
	// Approvers lists who can approve
	Approvers []string

	// MinApprovals is the minimum number of approvals needed
	MinApprovals int

	// Timeout is how long to wait for approval
	Timeout time.Duration

	// EscalationPolicy defines escalation behavior
	EscalationPolicy *EscalationPolicy
}

// EscalationPolicy defines how approvals escalate
type EscalationPolicy struct {
	// After is the duration before escalation
	After time.Duration

	// EscalateTo lists who to escalate to
	EscalateTo []string

	// AutoApprove indicates if auto-approval occurs on timeout
	AutoApprove bool
}

// RetryPolicy defines retry behavior
type RetryPolicy struct {
	// MaxAttempts is the maximum number of attempts
	MaxAttempts int

	// InitialInterval is the initial retry interval
	InitialInterval time.Duration

	// MaxInterval is the maximum retry interval
	MaxInterval time.Duration

	// Multiplier is the backoff multiplier
	Multiplier float64

	// RetryableErrors lists error types that trigger retries
	RetryableErrors []string
}

// Trigger defines what triggers a workflow
type Trigger struct {
	// Type is the trigger type
	Type TriggerType

	// Config contains trigger-specific configuration
	Config map[string]interface{}
}

// TriggerType represents the type of trigger
type TriggerType string

const (
	TriggerTypeManual   TriggerType = "manual"
	TriggerTypeSchedule TriggerType = "schedule"
	TriggerTypeWebhook  TriggerType = "webhook"
	TriggerTypeEvent    TriggerType = "event"
)

// Workflow represents a workflow instance
type Workflow struct {
	// ID is the unique workflow identifier
	ID string

	// Definition is the workflow definition
	Definition *WorkflowDefinition

	// Status is the current workflow status
	Status WorkflowStatus

	// CreatedAt is when the workflow was created
	CreatedAt time.Time

	// UpdatedAt is when the workflow was last updated
	UpdatedAt time.Time
}

// WorkflowStatus represents the status of a workflow
type WorkflowStatus string

const (
	WorkflowStatusDraft    WorkflowStatus = "draft"
	WorkflowStatusActive   WorkflowStatus = "active"
	WorkflowStatusPaused   WorkflowStatus = "paused"
	WorkflowStatusArchived WorkflowStatus = "archived"
)

// WorkflowExecution represents a workflow execution instance
type WorkflowExecution struct {
	// ID is the unique execution identifier
	ID string

	// WorkflowID is the workflow being executed
	WorkflowID string

	// Status is the current execution status
	Status ExecutionStatus

	// Input is the execution input
	Input map[string]interface{}

	// Output is the execution output
	Output map[string]interface{}

	// Steps contains step execution states
	Steps []StepExecution

	// StartedAt is when the execution started
	StartedAt time.Time

	// CompletedAt is when the execution completed
	CompletedAt *time.Time

	// Error contains error information if failed
	Error string
}

// ExecutionStatus represents the status of an execution
type ExecutionStatus string

const (
	ExecutionStatusPending   ExecutionStatus = "pending"
	ExecutionStatusRunning   ExecutionStatus = "running"
	ExecutionStatusWaiting   ExecutionStatus = "waiting"
	ExecutionStatusSucceeded ExecutionStatus = "succeeded"
	ExecutionStatusFailed    ExecutionStatus = "failed"
	ExecutionStatusCancelled ExecutionStatus = "cancelled"
	ExecutionStatusTimedOut  ExecutionStatus = "timed_out"
)

// StepExecution represents the execution state of a step
type StepExecution struct {
	// StepID is the step identifier
	StepID string

	// Status is the step execution status
	Status ExecutionStatus

	// Input is the step input
	Input interface{}

	// Output is the step output
	Output interface{}

	// StartedAt is when the step started
	StartedAt time.Time

	// CompletedAt is when the step completed
	CompletedAt *time.Time

	// Attempts is the number of execution attempts
	Attempts int

	// Error contains error information if failed
	Error string
}

// StepResult represents the result of a step execution
type StepResult struct {
	// Success indicates if the step succeeded
	Success bool

	// Output is the step output
	Output interface{}

	// Error is set if the step failed
	Error string

	// Metadata contains additional result metadata
	Metadata map[string]interface{}
}

// Decision represents an approval decision
type Decision string

const (
	DecisionApproved Decision = "approved"
	DecisionRejected Decision = "rejected"
)

// PendingApproval represents a pending approval request
type PendingApproval struct {
	// WorkflowID is the workflow awaiting approval
	WorkflowID string

	// ExecutionID is the execution awaiting approval
	ExecutionID string

	// StepID is the step awaiting approval
	StepID string

	// RequestedAt is when the approval was requested
	RequestedAt time.Time

	// ExpiresAt is when the approval request expires
	ExpiresAt time.Time

	// Context contains approval context
	Context map[string]interface{}

	// Approvers lists who can approve
	Approvers []string
}

// WorkflowFilter filters workflow queries
type WorkflowFilter struct {
	// Status filters by status
	Status []WorkflowStatus

	// Name filters by name pattern
	Name string

	// CreatedAfter filters by creation date
	CreatedAfter *time.Time

	// CreatedBefore filters by creation date
	CreatedBefore *time.Time

	// Limit is the maximum results
	Limit int

	// Offset is the result offset
	Offset int
}

// Common errors for workflow operations
var (
	// ErrWorkflowNotFound indicates the workflow doesn't exist
	ErrWorkflowNotFound = workflowError("workflow not found")

	// ErrExecutionNotFound indicates the execution doesn't exist
	ErrExecutionNotFound = workflowError("execution not found")

	// ErrStepNotFound indicates the step doesn't exist
	ErrStepNotFound = workflowError("step not found")

	// ErrApprovalNotPending indicates no pending approval
	ErrApprovalNotPending = workflowError("no pending approval")

	// ErrUnauthorizedApprover indicates the user cannot approve
	ErrUnauthorizedApprover = workflowError("unauthorized approver")

	// ErrWorkflowTimeout indicates the workflow timed out
	ErrWorkflowTimeout = workflowError("workflow timed out")

	// ErrStepTimeout indicates a step timed out
	ErrStepTimeout = workflowError("step timed out")
)

type workflowError string

func (e workflowError) Error() string {
	return string(e)
}
