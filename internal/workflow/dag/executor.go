package dag

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/platformfoundry/pf-ce/internal/workflow"
)

// StepHandler interface for executing workflow steps
type StepHandler interface {
	Type() workflow.StepType
	Validate(config map[string]interface{}) error
	Execute(ctx context.Context, step *workflow.StepExecution, config map[string]interface{}, resolver OutputResolver) (*workflow.StepResult, error)
}

// OutputResolver resolves step outputs and template variables
type OutputResolver interface {
	Resolve(ctx context.Context, template string) (string, error)
	ResolveMap(ctx context.Context, data map[string]interface{}) (map[string]interface{}, error)
	GetStepOutput(stepID, key string) (interface{}, bool)
	GetInput(key string) (interface{}, bool)
}

// ExecutorConfig configures the DAG executor
type ExecutorConfig struct {
	MaxParallel       int
	DefaultTimeout    time.Duration
	RetryDelay        time.Duration
	ContinueOnFailure bool
}

// DefaultExecutorConfig returns default executor configuration
func DefaultExecutorConfig() ExecutorConfig {
	return ExecutorConfig{
		MaxParallel:       4,
		DefaultTimeout:    30 * time.Minute,
		RetryDelay:        5 * time.Second,
		ContinueOnFailure: false,
	}
}

// Executor executes DAG workflows
type Executor struct {
	config    ExecutorConfig
	handlers  map[workflow.StepType]StepHandler
	semaphore chan struct{}
	mu        sync.RWMutex
}

// NewExecutor creates a new DAG executor
func NewExecutor(config ExecutorConfig) *Executor {
	if config.MaxParallel == 0 {
		config.MaxParallel = 4
	}
	if config.DefaultTimeout == 0 {
		config.DefaultTimeout = 30 * time.Minute
	}
	if config.RetryDelay == 0 {
		config.RetryDelay = 5 * time.Second
	}

	return &Executor{
		config:    config,
		handlers:  make(map[workflow.StepType]StepHandler),
		semaphore: make(chan struct{}, config.MaxParallel),
	}
}

// RegisterHandler registers a step handler
func (e *Executor) RegisterHandler(handler StepHandler) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.handlers[handler.Type()] = handler
}

// GetHandler returns a handler for the given step type
func (e *Executor) GetHandler(stepType workflow.StepType) (StepHandler, bool) {
	e.mu.RLock()
	defer e.mu.RUnlock()
	handler, ok := e.handlers[stepType]
	return handler, ok
}

// ExecutionContext holds context for a workflow execution
type ExecutionContext struct {
	Execution     *workflow.DAGExecution
	Workflow      *workflow.DAGWorkflow
	Graph         *Graph
	Resolver      OutputResolver
	OnStepStart   func(stepID string)
	OnStepComplete func(stepID string, result *workflow.StepResult)
}

// Execute executes a DAG workflow
func (e *Executor) Execute(ctx context.Context, execCtx *ExecutionContext) error {
	// Get execution levels
	levels, err := execCtx.Graph.GetParallelExecutionLevels()
	if err != nil {
		return fmt.Errorf("failed to get execution levels: %w", err)
	}

	// Build step map for quick lookup
	stepMap := make(map[string]workflow.StepSpec)
	for _, step := range execCtx.Workflow.Spec.Steps {
		stepMap[step.ID] = step
	}

	// Execute level by level
	for levelNum, level := range levels {
		if err := e.executeLevel(ctx, execCtx, levelNum, level, stepMap); err != nil {
			// Check if we should continue on failure
			if !e.config.ContinueOnFailure {
				return fmt.Errorf("level %d failed: %w", levelNum, err)
			}
		}
	}

	return nil
}

// executeLevel runs all steps in a level concurrently
func (e *Executor) executeLevel(ctx context.Context, execCtx *ExecutionContext, levelNum int, stepIDs []string, stepMap map[string]workflow.StepSpec) error {
	var waitGroup sync.WaitGroup
	stepErrorChan := make(chan error, len(stepIDs))

	for _, stepID := range stepIDs {
		stepSpec, ok := stepMap[stepID]
		if !ok {
			continue
		}

		// Check condition
		if stepSpec.Condition != "" {
			shouldRun, err := e.evaluateCondition(ctx, stepSpec.Condition, execCtx.Resolver)
			if err != nil {
				return fmt.Errorf("failed to evaluate condition for step %s: %w", stepID, err)
			}
			if !shouldRun {
				// Mark as skipped
				stepExec := execCtx.Execution.Steps[stepID]
				if stepExec != nil {
					stepExec.Status = workflow.StepStatusSkipped
					now := time.Now()
					stepExec.CompletedAt = &now
				}
				continue
			}
		}

		waitGroup.Add(1)
		go func(currentStep workflow.StepSpec) {
			defer waitGroup.Done()

			// Acquire semaphore slot
			e.semaphore <- struct{}{}
			defer func() { <-e.semaphore }()

			if err := e.executeStep(ctx, execCtx, currentStep); err != nil {
				stepErrorChan <- fmt.Errorf("step %s failed: %w", currentStep.ID, err)
			}
		}(stepSpec)
	}

	waitGroup.Wait()
	close(stepErrorChan)

	// Collect errors
	var stepErrors []error
	for err := range stepErrorChan {
		stepErrors = append(stepErrors, err)
	}

	if len(stepErrors) > 0 {
		return fmt.Errorf("%d steps failed: %v", len(stepErrors), stepErrors)
	}

	return nil
}

// executeStep runs a single step with retry logic
func (e *Executor) executeStep(ctx context.Context, execCtx *ExecutionContext, stepSpec workflow.StepSpec) error {
	stepExecution := execCtx.Execution.Steps[stepSpec.ID]
	if stepExecution == nil {
		stepExecution = &workflow.StepExecution{
			ID:      fmt.Sprintf("%s-%s", execCtx.Execution.ID, stepSpec.ID),
			StepID:  stepSpec.ID,
			Status:  workflow.StepStatusPending,
			Attempt: 0,
		}
		execCtx.Execution.Steps[stepSpec.ID] = stepExecution
	}

	// Get handler
	handler, ok := e.GetHandler(stepSpec.Type)
	if !ok {
		stepExecution.Status = workflow.StepStatusFailed
		stepExecution.Error = fmt.Sprintf("no handler registered for step type: %s", stepSpec.Type)
		return fmt.Errorf("%s", stepExecution.Error)
	}

	// Parse timeout
	stepTimeout := e.config.DefaultTimeout
	if stepSpec.Timeout != "" {
		if parsed, err := time.ParseDuration(stepSpec.Timeout); err == nil {
			stepTimeout = parsed
		}
	}

	// Resolve config variables
	resolvedConfig, err := execCtx.Resolver.ResolveMap(ctx, stepSpec.Config)
	if err != nil {
		stepExecution.Status = workflow.StepStatusFailed
		stepExecution.Error = fmt.Sprintf("failed to resolve config: %v", err)
		return fmt.Errorf("%s", stepExecution.Error)
	}

	// Determine retry count
	maxAttempts := 1
	retryDelay := e.config.RetryDelay
	if stepSpec.Retries != nil {
		maxAttempts = stepSpec.Retries.MaxAttempts
		if stepSpec.Retries.Delay != "" {
			if parsed, err := time.ParseDuration(stepSpec.Retries.Delay); err == nil {
				retryDelay = parsed
			}
		}
	}

	// Notify step start
	if execCtx.OnStepStart != nil {
		execCtx.OnStepStart(stepSpec.ID)
	}

	var lastError error
	for attemptNum := 1; attemptNum <= maxAttempts; attemptNum++ {
		stepExecution.Attempt = attemptNum
		stepExecution.Status = workflow.StepStatusRunning
		now := time.Now()
		stepExecution.StartedAt = &now

		// Create step context with timeout
		stepCtx, cancel := context.WithTimeout(ctx, stepTimeout)

		// Execute step
		result, err := handler.Execute(stepCtx, stepExecution, resolvedConfig, execCtx.Resolver)
		cancel()

		if err != nil {
			lastError = err
			stepExecution.Logs = append(stepExecution.Logs, workflow.StepLog{
				Time:    time.Now(),
				Level:   "error",
				Message: fmt.Sprintf("Attempt %d failed: %v", attemptNum, err),
			})

			if attemptNum < maxAttempts {
				time.Sleep(retryDelay)
				// Exponential backoff if configured
				if stepSpec.Retries != nil && stepSpec.Retries.Backoff == "exponential" {
					retryDelay *= 2
				}
				continue
			}
		}

		// Process result
		if result != nil {
			stepExecution.Status = result.Status
			stepExecution.Outputs = result.Outputs
			stepExecution.Logs = append(stepExecution.Logs, result.Logs...)
			if result.Error != nil {
				stepExecution.Error = result.Error.Error()
			} else if result.ErrorMsg != "" {
				stepExecution.Error = result.ErrorMsg
			}
		}

		completedAt := time.Now()
		stepExecution.CompletedAt = &completedAt

		// Notify step complete
		if execCtx.OnStepComplete != nil {
			execCtx.OnStepComplete(stepSpec.ID, result)
		}

		// Check if step succeeded
		if result != nil && result.Status == workflow.StepStatusCompleted {
			return nil
		}

		// Check continueOn settings
		if stepSpec.ContinueOn != nil && stepSpec.ContinueOn.Failure && stepExecution.Status == workflow.StepStatusFailed {
			return nil
		}

		break
	}

	if lastError != nil {
		stepExecution.Status = workflow.StepStatusFailed
		stepExecution.Error = lastError.Error()
		return lastError
	}

	if stepExecution.Status != workflow.StepStatusCompleted {
		return fmt.Errorf("step %s did not complete successfully: %s", stepSpec.ID, stepExecution.Status)
	}

	return nil
}

// evaluateCondition evaluates a step condition expression
func (e *Executor) evaluateCondition(ctx context.Context, condition string, resolver OutputResolver) (bool, error) {
	// Simple condition evaluation
	// Supports: ${steps.X.status} == "completed", ${inputs.Y} == "value"
	resolved, err := resolver.Resolve(ctx, condition)
	if err != nil {
		return false, err
	}

	// Basic evaluation - if the condition resolves to "true" or non-empty, it passes
	switch resolved {
	case "true", "1", "yes":
		return true, nil
	case "false", "0", "no", "":
		return false, nil
	default:
		// Non-empty resolved value is truthy
		return resolved != "", nil
	}
}

// Cancel cancels execution (to be implemented by caller managing context)
func (e *Executor) Cancel() {
	// Cancellation is handled via context
}
