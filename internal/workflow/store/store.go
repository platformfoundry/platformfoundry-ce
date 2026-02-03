package store

import (
	"context"

	"github.com/platformfoundry/pf-ce/internal/workflow"
)

// Store interface for workflow persistence
type Store interface {
	// Workflow operations
	SaveWorkflow(ctx context.Context, wf *workflow.DAGWorkflow) error
	GetWorkflow(ctx context.Context, name string) (*workflow.DAGWorkflow, error)
	ListWorkflows(ctx context.Context) ([]*workflow.DAGWorkflow, error)
	DeleteWorkflow(ctx context.Context, name string) error

	// Execution operations
	SaveExecution(ctx context.Context, exec *workflow.DAGExecution) error
	GetExecution(ctx context.Context, id string) (*workflow.DAGExecution, error)
	ListExecutions(ctx context.Context, workflowName string, limit int) ([]*workflow.DAGExecution, error)
	UpdateExecutionStatus(ctx context.Context, id string, status workflow.WorkflowStatus) error

	// Step execution operations
	SaveStepExecution(ctx context.Context, execID string, step *workflow.StepExecution) error
	GetStepExecution(ctx context.Context, execID, stepID string) (*workflow.StepExecution, error)

	// Close closes the store
	Close() error
}

// QueryOptions defines options for listing executions
type QueryOptions struct {
	WorkflowName string
	Status       workflow.WorkflowStatus
	Limit        int
	Offset       int
}
