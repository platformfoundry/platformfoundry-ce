package dag

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/platformfoundry/platformfoundry-ce/internal/workflow"
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
	var wg sync.WaitGroup
	errChan := make(chan error, len(stepIDs))

	for _, stepID := range stepIDs {
		step, ok := stepMap[stepID]
		if !ok {
			continue
		}

		// Check condition
		if step.Condition != "" {
			shouldRun, err := e.evaluateCondition(ctx, step.Condition, execCtx.Resolver)
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

		wg.Add(1)
		go func(s workflow.StepSpec) {
			defer wg.Done()

			// Acquire semaphore slot
			e.semaphore <- struct{}{}
			defer func() { <-e.semaphore }()

			if err := e.executeStep(ctx, execCtx, s); err != nil {
				errChan <- fmt.Errorf("step %s failed: %w", s.ID, err)
			}
		}(step)
	}

	wg.Wait()
	close(errChan)

	// Collect errors
	var errs []error
	for err := range errChan {
		errs = append(errs, err)
	}

	if len(errs) > 0 {
		return fmt.Errorf("%d steps failed: %v", len(errs), errs)
	}

	return nil
}

// executeStep runs a single step with retry logic
func (e *Executor) executeStep(ctx context.Context, execCtx *ExecutionContext, step workflow.StepSpec) error {
	stepExec := execCtx.Execution.Steps[step.ID]
	if stepExec == nil {
		stepExec = &workflow.StepExecution{
			ID:      fmt.Sprintf("%s-%s", execCtx.Execution.ID, step.ID),
			StepID:  step.ID,
			Status:  workflow.StepStatusPending,
			Attempt: 0,
		}
		execCtx.Execution.Steps[step.ID] = stepExec
	}

	// Get handler
	handler, ok := e.GetHandler(step.Type)
	if !ok {
		stepExec.Status = workflow.StepStatusFailed
		stepExec.Error = fmt.Sprintf("no handler registered for step type: %s", step.Type)
		return fmt.Errorf("%s", stepExec.Error)
	}

	// Parse timeout
	timeout := e.config.DefaultTimeout
	if step.Timeout != "" {
		if parsed, err := time.ParseDuration(step.Timeout); err == nil {
			timeout = parsed
		}
	}

	// Resolve config variables
	resolvedConfig, err := execCtx.Resolver.ResolveMap(ctx, step.Config)
	if err != nil {
		stepExec.Status = workflow.StepStatusFailed
		stepExec.Error = fmt.Sprintf("failed to resolve config: %v", err)
		return fmt.Errorf("%s", stepExec.Error)
	}

	// Determine retry count
	maxAttempts := 1
	retryDelay := e.config.RetryDelay
	if step.Retries != nil {
		maxAttempts = step.Retries.MaxAttempts
		if step.Retries.Delay != "" {
			if parsed, err := time.ParseDuration(step.Retries.Delay); err == nil {
				retryDelay = parsed
			}
		}
	}

	// Notify step start
	if execCtx.OnStepStart != nil {
		execCtx.OnStepStart(step.ID)
	}

	var lastErr error
	for attempt := 1; attempt <= maxAttempts; attempt++ {
		stepExec.Attempt = attempt
		stepExec.Status = workflow.StepStatusRunning
		now := time.Now()
		stepExec.StartedAt = &now

		// Create step context with timeout
		stepCtx, cancel := context.WithTimeout(ctx, timeout)

		// Execute step
		result, err := handler.Execute(stepCtx, stepExec, resolvedConfig, execCtx.Resolver)
		cancel()

		if err != nil {
			lastErr = err
			stepExec.Logs = append(stepExec.Logs, workflow.StepLog{
				Time:    time.Now(),
				Level:   "error",
				Message: fmt.Sprintf("Attempt %d failed: %v", attempt, err),
			})

			if attempt < maxAttempts {
				time.Sleep(retryDelay)
				// Exponential backoff if configured
				if step.Retries != nil && step.Retries.Backoff == "exponential" {
					retryDelay *= 2
				}
				continue
			}
		}

		// Process result
		if result != nil {
			stepExec.Status = result.Status
			stepExec.Outputs = result.Outputs
			stepExec.Logs = append(stepExec.Logs, result.Logs...)
			if result.Error != nil {
				stepExec.Error = result.Error.Error()
			} else if result.ErrorMsg != "" {
				stepExec.Error = result.ErrorMsg
			}
		}

		completedAt := time.Now()
		stepExec.CompletedAt = &completedAt

		// Notify step complete
		if execCtx.OnStepComplete != nil {
			execCtx.OnStepComplete(step.ID, result)
		}

		// Check if step succeeded
		if result != nil && result.Status == workflow.StepStatusCompleted {
			return nil
		}

		// Check continueOn settings
		if step.ContinueOn != nil && step.ContinueOn.Failure && stepExec.Status == workflow.StepStatusFailed {
			return nil
		}

		break
	}

	if lastErr != nil {
		stepExec.Status = workflow.StepStatusFailed
		stepExec.Error = lastErr.Error()
		return lastErr
	}

	if stepExec.Status != workflow.StepStatusCompleted {
		return fmt.Errorf("step %s did not complete successfully: %s", step.ID, stepExec.Status)
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
