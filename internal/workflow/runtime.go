package workflow

import (
	"context"
	"fmt"
	"regexp"
	"sync"
	"time"

	"github.com/google/uuid"
)

// WorkflowStore interface for workflow persistence (implemented by store.BoltStore)
type WorkflowStore interface {
	SaveWorkflow(ctx context.Context, wf *DAGWorkflow) error
	GetWorkflow(ctx context.Context, name string) (*DAGWorkflow, error)
	ListWorkflows(ctx context.Context) ([]*DAGWorkflow, error)
	DeleteWorkflow(ctx context.Context, name string) error
	SaveExecution(ctx context.Context, exec *DAGExecution) error
	GetExecution(ctx context.Context, id string) (*DAGExecution, error)
	ListExecutions(ctx context.Context, workflowName string, limit int) ([]*DAGExecution, error)
	UpdateExecutionStatus(ctx context.Context, id string, status WorkflowStatus) error
	SaveStepExecution(ctx context.Context, execID string, step *StepExecution) error
	GetStepExecution(ctx context.Context, execID, stepID string) (*StepExecution, error)
	Close() error
}

// StepHandler interface for executing workflow steps
type StepHandler interface {
	Type() StepType
	Validate(config map[string]interface{}) error
	Execute(ctx context.Context, step *StepExecution, config map[string]interface{}, resolver *VariableResolver) (*StepResult, error)
}

// VariableResolver resolves template variables in workflow configurations
type VariableResolver struct {
	inputs       map[string]interface{}
	stepOutputs  map[string]map[string]interface{}
	stepStatuses map[string]string
	mu           sync.RWMutex
}

// NewVariableResolver creates a new resolver
func NewVariableResolver(inputs map[string]interface{}) *VariableResolver {
	if inputs == nil {
		inputs = make(map[string]interface{})
	}
	return &VariableResolver{
		inputs:       inputs,
		stepOutputs:  make(map[string]map[string]interface{}),
		stepStatuses: make(map[string]string),
	}
}

// Template variable patterns
var (
	inputPattern      = regexp.MustCompile(`\$\{inputs\.([a-zA-Z_][a-zA-Z0-9_]*)\}`)
	stepOutputPattern = regexp.MustCompile(`\$\{steps\.([a-zA-Z_][a-zA-Z0-9_-]*)\.outputs\.([a-zA-Z_][a-zA-Z0-9_]*)\}`)
	stepStatusPattern = regexp.MustCompile(`\$\{steps\.([a-zA-Z_][a-zA-Z0-9_-]*)\.status\}`)
)

// GetInput retrieves an input value
func (r *VariableResolver) GetInput(key string) (interface{}, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	val, ok := r.inputs[key]
	return val, ok
}

// SetStepOutputs sets the outputs for a step
func (r *VariableResolver) SetStepOutputs(stepID string, outputs map[string]interface{}) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.stepOutputs[stepID] = outputs
}

// GetStepOutput retrieves a specific output from a step
func (r *VariableResolver) GetStepOutput(stepID, key string) (interface{}, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	if outputs, ok := r.stepOutputs[stepID]; ok {
		val, exists := outputs[key]
		return val, exists
	}
	return nil, false
}

// SetStepStatus sets the status for a step
func (r *VariableResolver) SetStepStatus(stepID, status string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.stepStatuses[stepID] = status
}

// Resolve resolves template variables in a string
func (r *VariableResolver) Resolve(template string) string {
	if template == "" {
		return template
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
		r.mu.RLock()
		status, ok := r.stepStatuses[stepID]
		r.mu.RUnlock()
		if ok {
			return status
		}
		return match
	})

	return result
}

// ResolveMap resolves template variables in a map recursively
func (r *VariableResolver) ResolveMap(data map[string]interface{}) map[string]interface{} {
	if data == nil {
		return nil
	}

	result := make(map[string]interface{})
	for key, value := range data {
		result[key] = r.resolveValue(value)
	}
	return result
}

func (r *VariableResolver) resolveValue(value interface{}) interface{} {
	switch v := value.(type) {
	case string:
		return r.Resolve(v)
	case map[string]interface{}:
		return r.ResolveMap(v)
	case []interface{}:
		result := make([]interface{}, len(v))
		for i, item := range v {
			result[i] = r.resolveValue(item)
		}
		return result
	default:
		return value
	}
}

// Runtime orchestrates DAG workflow execution
type Runtime struct {
	store    WorkflowStore
	handlers map[StepType]StepHandler
	parser   *Parser

	// In-memory workflow cache
	workflows map[string]*DAGWorkflow
	mu        sync.RWMutex

	// Execution tracking
	executions map[string]*DAGExecution
	execMu     sync.RWMutex

	// Configuration
	maxParallel    int
	defaultTimeout time.Duration

	// Event listeners
	listeners  []RuntimeListener
	listenerMu sync.RWMutex

	ctx    context.Context
	cancel context.CancelFunc
}

// RuntimeListener receives workflow events
type RuntimeListener interface {
	OnWorkflowStart(exec *DAGExecution)
	OnWorkflowComplete(exec *DAGExecution)
	OnStepStart(exec *DAGExecution, stepID string)
	OnStepComplete(exec *DAGExecution, stepID string, result *StepResult)
}

// RuntimeConfig configures the runtime
type RuntimeConfig struct {
	Store          WorkflowStore
	MaxParallel    int
	DefaultTimeout time.Duration
}

// NewRuntime creates a new workflow runtime
func NewRuntime(config RuntimeConfig) *Runtime {
	if config.MaxParallel == 0 {
		config.MaxParallel = 4
	}
	if config.DefaultTimeout == 0 {
		config.DefaultTimeout = 30 * time.Minute
	}

	return &Runtime{
		store:          config.Store,
		handlers:       make(map[StepType]StepHandler),
		parser:         NewParser(),
		workflows:      make(map[string]*DAGWorkflow),
		executions:     make(map[string]*DAGExecution),
		maxParallel:    config.MaxParallel,
		defaultTimeout: config.DefaultTimeout,
		listeners:      make([]RuntimeListener, 0),
	}
}

// Start starts the runtime
func (r *Runtime) Start(ctx context.Context) error {
	r.ctx, r.cancel = context.WithCancel(ctx)

	if r.store == nil {
		return nil
	}

	// Load workflows from store
	workflows, err := r.store.ListWorkflows(ctx)
	if err != nil {
		return fmt.Errorf("failed to load workflows: %w", err)
	}

	r.mu.Lock()
	for _, wf := range workflows {
		r.workflows[wf.Metadata.Name] = wf
	}
	r.mu.Unlock()

	return nil
}

// Stop stops the runtime
func (r *Runtime) Stop() error {
	if r.cancel != nil {
		r.cancel()
	}

	if r.store != nil {
		return r.store.Close()
	}
	return nil
}

// RegisterHandler registers a step handler
func (r *Runtime) RegisterHandler(handler StepHandler) {
	r.handlers[handler.Type()] = handler
}

// Subscribe adds a runtime listener
func (r *Runtime) Subscribe(listener RuntimeListener) {
	r.listenerMu.Lock()
	defer r.listenerMu.Unlock()
	r.listeners = append(r.listeners, listener)
}

// ApplyWorkflow creates or updates a workflow
func (r *Runtime) ApplyWorkflow(ctx context.Context, wf *DAGWorkflow) error {
	// Validate workflow
	if err := r.parser.validate(wf); err != nil {
		return fmt.Errorf("workflow validation failed: %w", err)
	}

	// Save to store
	if r.store != nil {
		if err := r.store.SaveWorkflow(ctx, wf); err != nil {
			return fmt.Errorf("failed to save workflow: %w", err)
		}
	}

	// Update cache
	r.mu.Lock()
	r.workflows[wf.Metadata.Name] = wf
	r.mu.Unlock()

	return nil
}

// GetWorkflow retrieves a workflow by name
func (r *Runtime) GetWorkflow(ctx context.Context, name string) (*DAGWorkflow, error) {
	r.mu.RLock()
	wf, ok := r.workflows[name]
	r.mu.RUnlock()

	if ok {
		return wf, nil
	}

	if r.store != nil {
		return r.store.GetWorkflow(ctx, name)
	}

	return nil, fmt.Errorf("workflow not found: %s", name)
}

// ListWorkflows returns all workflows
func (r *Runtime) ListWorkflows(ctx context.Context) ([]*DAGWorkflow, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	result := make([]*DAGWorkflow, 0, len(r.workflows))
	for _, wf := range r.workflows {
		result = append(result, wf)
	}
	return result, nil
}

// DeleteWorkflow removes a workflow
func (r *Runtime) DeleteWorkflow(ctx context.Context, name string) error {
	if r.store != nil {
		if err := r.store.DeleteWorkflow(ctx, name); err != nil {
			return err
		}
	}

	r.mu.Lock()
	delete(r.workflows, name)
	r.mu.Unlock()

	return nil
}

// RunWorkflow starts a workflow execution
func (r *Runtime) RunWorkflow(ctx context.Context, name string, inputs map[string]interface{}) (*DAGExecution, error) {
	wf, err := r.GetWorkflow(ctx, name)
	if err != nil {
		return nil, err
	}

	// Validate and apply default inputs
	inputs = ApplyDefaults(wf, inputs)
	if err := ValidateInputs(wf, inputs); err != nil {
		return nil, fmt.Errorf("input validation failed: %w", err)
	}

	// Create execution
	exec := &DAGExecution{
		ID:           uuid.New().String(),
		WorkflowName: name,
		Status:       WorkflowStatusPending,
		Trigger:      "manual",
		Inputs:       inputs,
		Steps:        make(map[string]*StepExecution),
		StartedAt:    time.Now(),
	}

	// Initialize step executions
	for _, step := range wf.Spec.Steps {
		exec.Steps[step.ID] = &StepExecution{
			ID:     fmt.Sprintf("%s-%s", exec.ID, step.ID),
			StepID: step.ID,
			Status: StepStatusPending,
		}
	}

	// Save execution
	if r.store != nil {
		if err := r.store.SaveExecution(ctx, exec); err != nil {
			return nil, fmt.Errorf("failed to save execution: %w", err)
		}
	}

	// Track execution
	r.execMu.Lock()
	r.executions[exec.ID] = exec
	r.execMu.Unlock()

	// Run async
	go r.executeWorkflow(ctx, exec, wf)

	return exec, nil
}

// executeWorkflow runs the workflow
func (r *Runtime) executeWorkflow(ctx context.Context, exec *DAGExecution, wf *DAGWorkflow) {
	// Notify listeners
	r.notifyWorkflowStart(exec)

	// Update status
	exec.Status = WorkflowStatusExecuting
	if r.store != nil {
		r.store.SaveExecution(ctx, exec)
	}

	// Get parallel execution levels
	levels, err := r.getParallelLevels(wf)
	if err != nil {
		exec.Status = WorkflowStatusFailed
		exec.Error = fmt.Sprintf("failed to build DAG: %v", err)
		now := time.Now()
		exec.CompletedAt = &now
		if r.store != nil {
			r.store.SaveExecution(ctx, exec)
		}
		r.notifyWorkflowComplete(exec)
		return
	}

	// Create resolver
	resolver := NewVariableResolver(exec.Inputs)

	// Build step map
	stepMap := make(map[string]StepSpec)
	for _, step := range wf.Spec.Steps {
		stepMap[step.ID] = step
	}

	// Execute level by level
	for _, level := range levels {
		if err := r.executeLevel(ctx, exec, level, stepMap, resolver); err != nil {
			exec.Status = WorkflowStatusFailed
			exec.Error = err.Error()
			break
		}
	}

	// Update execution status
	now := time.Now()
	exec.CompletedAt = &now

	if exec.Status == WorkflowStatusExecuting {
		exec.Status = WorkflowStatusCompleted
	}

	// Save final state
	if r.store != nil {
		r.store.SaveExecution(ctx, exec)
	}
	r.notifyWorkflowComplete(exec)
}

// getParallelLevels computes parallel execution levels using topological sort
func (r *Runtime) getParallelLevels(wf *DAGWorkflow) ([][]string, error) {
	// Build adjacency info
	deps := make(map[string][]string)
	for _, step := range wf.Spec.Steps {
		deps[step.ID] = step.DependsOn
	}

	// Kahn's algorithm for topological sort with levels
	inDegree := make(map[string]int)
	for _, step := range wf.Spec.Steps {
		if _, ok := inDegree[step.ID]; !ok {
			inDegree[step.ID] = 0
		}
		for _, dep := range step.DependsOn {
			inDegree[dep] = inDegree[dep] // ensure exists
		}
	}

	for _, step := range wf.Spec.Steps {
		for _, dep := range step.DependsOn {
			inDegree[step.ID]++
			_ = dep
		}
	}

	var levels [][]string
	completed := make(map[string]bool)

	for len(completed) < len(wf.Spec.Steps) {
		var currentLevel []string

		for _, step := range wf.Spec.Steps {
			if completed[step.ID] {
				continue
			}

			// Check if all dependencies are completed
			allDepsCompleted := true
			for _, dep := range step.DependsOn {
				if !completed[dep] {
					allDepsCompleted = false
					break
				}
			}

			if allDepsCompleted {
				currentLevel = append(currentLevel, step.ID)
			}
		}

		if len(currentLevel) == 0 && len(completed) < len(wf.Spec.Steps) {
			return nil, fmt.Errorf("circular dependency detected")
		}

		if len(currentLevel) > 0 {
			levels = append(levels, currentLevel)
			for _, id := range currentLevel {
				completed[id] = true
			}
		}
	}

	return levels, nil
}

// executeLevel runs all steps in a level concurrently
func (r *Runtime) executeLevel(ctx context.Context, exec *DAGExecution, stepIDs []string, stepMap map[string]StepSpec, resolver *VariableResolver) error {
	var wg sync.WaitGroup
	errChan := make(chan error, len(stepIDs))
	sem := make(chan struct{}, r.maxParallel)

	for _, stepID := range stepIDs {
		step, ok := stepMap[stepID]
		if !ok {
			continue
		}

		wg.Add(1)
		go func(s StepSpec) {
			defer wg.Done()

			sem <- struct{}{}
			defer func() { <-sem }()

			if err := r.executeStep(ctx, exec, s, resolver); err != nil {
				errChan <- fmt.Errorf("step %s failed: %w", s.ID, err)
			}
		}(step)
	}

	wg.Wait()
	close(errChan)

	var errs []error
	for err := range errChan {
		errs = append(errs, err)
	}

	if len(errs) > 0 {
		return fmt.Errorf("%d steps failed: %v", len(errs), errs)
	}

	return nil
}

// executeStep runs a single step
func (r *Runtime) executeStep(ctx context.Context, exec *DAGExecution, step StepSpec, resolver *VariableResolver) error {
	stepExec := exec.Steps[step.ID]
	if stepExec == nil {
		stepExec = &StepExecution{
			ID:     fmt.Sprintf("%s-%s", exec.ID, step.ID),
			StepID: step.ID,
			Status: StepStatusPending,
		}
		exec.Steps[step.ID] = stepExec
	}

	// Get handler
	handler, ok := r.handlers[step.Type]
	if !ok {
		stepExec.Status = StepStatusFailed
		stepExec.Error = fmt.Sprintf("no handler registered for step type: %s", step.Type)
		return fmt.Errorf("%s", stepExec.Error)
	}

	// Notify step start
	r.notifyStepStart(exec, step.ID)

	// Parse timeout
	timeout := r.defaultTimeout
	if step.Timeout != "" {
		if parsed, err := time.ParseDuration(step.Timeout); err == nil {
			timeout = parsed
		}
	}

	// Resolve config variables
	resolvedConfig := resolver.ResolveMap(step.Config)

	// Update step status
	stepExec.Status = StepStatusRunning
	now := time.Now()
	stepExec.StartedAt = &now

	// Create step context with timeout
	stepCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	// Execute step
	result, err := handler.Execute(stepCtx, stepExec, resolvedConfig, resolver)

	// Process result
	completedAt := time.Now()
	stepExec.CompletedAt = &completedAt

	if result != nil {
		stepExec.Status = result.Status
		stepExec.Outputs = result.Outputs
		stepExec.Logs = append(stepExec.Logs, result.Logs...)
		if result.Error != nil {
			stepExec.Error = result.Error.Error()
		} else if result.ErrorMsg != "" {
			stepExec.Error = result.ErrorMsg
		}

		// Update resolver with outputs
		if result.Outputs != nil {
			resolver.SetStepOutputs(step.ID, result.Outputs)
		}
		resolver.SetStepStatus(step.ID, string(result.Status))
	}

	// Save step execution
	if r.store != nil {
		r.store.SaveStepExecution(ctx, exec.ID, stepExec)
	}

	// Notify step complete
	r.notifyStepComplete(exec, step.ID, result)

	if err != nil {
		return err
	}

	if stepExec.Status != StepStatusCompleted {
		return fmt.Errorf("step %s did not complete successfully: %s", step.ID, stepExec.Status)
	}

	return nil
}

// GetExecution retrieves an execution by ID
func (r *Runtime) GetExecution(ctx context.Context, id string) (*DAGExecution, error) {
	r.execMu.RLock()
	exec, ok := r.executions[id]
	r.execMu.RUnlock()

	if ok {
		return exec, nil
	}

	if r.store != nil {
		return r.store.GetExecution(ctx, id)
	}

	return nil, fmt.Errorf("execution not found: %s", id)
}

// ListExecutions returns executions for a workflow
func (r *Runtime) ListExecutions(ctx context.Context, workflowName string, limit int) ([]*DAGExecution, error) {
	if r.store != nil {
		return r.store.ListExecutions(ctx, workflowName, limit)
	}

	r.execMu.RLock()
	defer r.execMu.RUnlock()

	result := make([]*DAGExecution, 0)
	for _, exec := range r.executions {
		if workflowName == "" || exec.WorkflowName == workflowName {
			result = append(result, exec)
			if limit > 0 && len(result) >= limit {
				break
			}
		}
	}
	return result, nil
}

// CancelExecution cancels a running execution
func (r *Runtime) CancelExecution(ctx context.Context, id string) error {
	r.execMu.Lock()
	exec, ok := r.executions[id]
	r.execMu.Unlock()

	if !ok {
		return fmt.Errorf("execution not found: %s", id)
	}

	if exec.Status != WorkflowStatusPending && exec.Status != WorkflowStatusExecuting {
		return fmt.Errorf("cannot cancel execution in status: %s", exec.Status)
	}

	exec.Status = WorkflowStatusFailed
	exec.Error = "cancelled by user"
	now := time.Now()
	exec.CompletedAt = &now

	if r.store != nil {
		return r.store.SaveExecution(ctx, exec)
	}
	return nil
}

// GetStepLogs retrieves logs for a step
func (r *Runtime) GetStepLogs(ctx context.Context, execID, stepID string) ([]StepLog, error) {
	exec, err := r.GetExecution(ctx, execID)
	if err != nil {
		return nil, err
	}

	stepExec, ok := exec.Steps[stepID]
	if !ok {
		return nil, fmt.Errorf("step not found: %s", stepID)
	}

	return stepExec.Logs, nil
}

// Notification helpers

func (r *Runtime) notifyWorkflowStart(exec *DAGExecution) {
	r.listenerMu.RLock()
	listeners := make([]RuntimeListener, len(r.listeners))
	copy(listeners, r.listeners)
	r.listenerMu.RUnlock()

	for _, l := range listeners {
		go l.OnWorkflowStart(exec)
	}
}

func (r *Runtime) notifyWorkflowComplete(exec *DAGExecution) {
	r.listenerMu.RLock()
	listeners := make([]RuntimeListener, len(r.listeners))
	copy(listeners, r.listeners)
	r.listenerMu.RUnlock()

	for _, l := range listeners {
		go l.OnWorkflowComplete(exec)
	}
}

func (r *Runtime) notifyStepStart(exec *DAGExecution, stepID string) {
	r.listenerMu.RLock()
	listeners := make([]RuntimeListener, len(r.listeners))
	copy(listeners, r.listeners)
	r.listenerMu.RUnlock()

	for _, l := range listeners {
		go l.OnStepStart(exec, stepID)
	}
}

func (r *Runtime) notifyStepComplete(exec *DAGExecution, stepID string, result *StepResult) {
	r.listenerMu.RLock()
	listeners := make([]RuntimeListener, len(r.listeners))
	copy(listeners, r.listeners)
	r.listenerMu.RUnlock()

	for _, l := range listeners {
		go l.OnStepComplete(exec, stepID, result)
	}
}
