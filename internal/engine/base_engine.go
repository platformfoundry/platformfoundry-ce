package engine

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
)

// BaseEngine provides common functionality for all engines
type BaseEngine struct {
	id       string
	name     string
	category string

	// State management
	state    EngineState
	progress ProgressInfo
	outputs  map[string]interface{}
	stateMu  sync.RWMutex

	// Context management
	ctx    context.Context
	cancel context.CancelFunc

	// Dependencies
	dependencies []string
	depResolver  DependencyResolver

	// Event handling
	listeners   []EventListener
	listenersMu sync.RWMutex

	// Configuration
	config     EngineConfig
	mockMode   bool
	mockConfig *MockConfig

	// Rollback
	rollbackPlan *RollbackPlan

	// Start time for duration tracking
	startedAt *time.Time
}

// NewBaseEngine creates a new base engine
func NewBaseEngine(name, category string) *BaseEngine {
	return &BaseEngine{
		id:        uuid.New().String(),
		name:      name,
		category:  category,
		state:     EngineStateIdle,
		outputs:   make(map[string]interface{}),
		listeners: make([]EventListener, 0),
		progress: ProgressInfo{
			Percentage:  0,
			SubProgress: make(map[string]int),
		},
	}
}

// ID returns the unique engine identifier
func (e *BaseEngine) ID() string {
	return e.id
}

// Name returns the engine name
func (e *BaseEngine) Name() string {
	return e.name
}

// Category returns the engine category
func (e *BaseEngine) Category() string {
	return e.category
}

// State returns the current engine state
func (e *BaseEngine) State() EngineState {
	e.stateMu.RLock()
	defer e.stateMu.RUnlock()
	return e.state
}

// Progress returns the current progress info
func (e *BaseEngine) Progress() ProgressInfo {
	e.stateMu.RLock()
	defer e.stateMu.RUnlock()
	return e.progress
}

// Outputs returns the engine outputs
func (e *BaseEngine) Outputs() map[string]interface{} {
	e.stateMu.RLock()
	defer e.stateMu.RUnlock()

	// Return a copy to prevent modification
	result := make(map[string]interface{})
	for k, v := range e.outputs {
		result[k] = v
	}
	return result
}

// DependsOn returns the list of engine dependencies
func (e *BaseEngine) DependsOn() []string {
	return e.dependencies
}

// SetDependencies sets the engine dependencies
func (e *BaseEngine) SetDependencies(deps []string) {
	e.dependencies = deps
}

// SetDependencyResolver sets the dependency resolver
func (e *BaseEngine) SetDependencyResolver(resolver DependencyResolver) {
	e.depResolver = resolver
}

// Initialize prepares the engine for execution
func (e *BaseEngine) Initialize(ctx context.Context, config EngineConfig) error {
	e.config = config
	e.mockMode = config.MockMode
	e.mockConfig = config.MockConfig

	if len(config.Dependencies) > 0 {
		e.dependencies = config.Dependencies
	}

	e.SetState(EngineStateIdle)
	return nil
}

// Start begins engine execution
func (e *BaseEngine) Start(ctx context.Context) error {
	e.ctx, e.cancel = context.WithCancel(ctx)
	now := time.Now()
	e.startedAt = &now
	e.SetState(EngineStateRunning)
	return nil
}

// Stop cancels engine execution
func (e *BaseEngine) Stop() error {
	if e.cancel != nil {
		e.cancel()
	}
	e.SetState(EngineStateCancelled)
	return nil
}

// Pause pauses the engine
func (e *BaseEngine) Pause() error {
	e.SetState(EngineStatePaused)
	return nil
}

// Resume resumes a paused engine
func (e *BaseEngine) Resume() error {
	e.SetState(EngineStateRunning)
	return nil
}

// SetState updates the engine state and emits an event
func (e *BaseEngine) SetState(state EngineState) {
	e.stateMu.Lock()
	e.state = state
	e.stateMu.Unlock()

	e.Emit(EngineEvent{
		EngineID:  e.id,
		Type:      EventTypeStateChange,
		Component: e.category,
		Message:   string(state),
		Timestamp: time.Now(),
	})
}

// SetProgress updates the engine progress and emits an event
func (e *BaseEngine) SetProgress(percentage int, message string, phaseCurrent, phaseTotal int) {
	e.stateMu.Lock()
	e.progress.Percentage = percentage
	e.progress.Message = message
	e.progress.PhasesCurrent = phaseCurrent
	e.progress.PhasesTotal = phaseTotal
	e.progress.CurrentPhase = message
	e.stateMu.Unlock()

	e.Emit(EngineEvent{
		EngineID:  e.id,
		Type:      EventTypeProgress,
		Component: e.category,
		Progress:  percentage,
		Message:   message,
		Timestamp: time.Now(),
	})
}

// SetOutput stores an output value
func (e *BaseEngine) SetOutput(key string, value interface{}) {
	e.stateMu.Lock()
	e.outputs[key] = value
	e.stateMu.Unlock()

	e.Emit(EngineEvent{
		EngineID:  e.id,
		Type:      EventTypeOutputSet,
		Component: e.category,
		Message:   fmt.Sprintf("Output set: %s", key),
		Timestamp: time.Now(),
		Metadata:  map[string]interface{}{"key": key},
	})
}

// GetOutput retrieves an output value
func (e *BaseEngine) GetOutput(key string) (interface{}, bool) {
	e.stateMu.RLock()
	defer e.stateMu.RUnlock()
	val, ok := e.outputs[key]
	return val, ok
}

// Subscribe adds an event listener
func (e *BaseEngine) Subscribe(listener EventListener) {
	e.listenersMu.Lock()
	e.listeners = append(e.listeners, listener)
	e.listenersMu.Unlock()
}

// Unsubscribe removes an event listener
func (e *BaseEngine) Unsubscribe(listener EventListener) {
	e.listenersMu.Lock()
	defer e.listenersMu.Unlock()

	for i, l := range e.listeners {
		if l == listener {
			e.listeners = append(e.listeners[:i], e.listeners[i+1:]...)
			return
		}
	}
}

// Emit broadcasts an event to all listeners
func (e *BaseEngine) Emit(event EngineEvent) {
	e.listenersMu.RLock()
	listeners := make([]EventListener, len(e.listeners))
	copy(listeners, e.listeners)
	e.listenersMu.RUnlock()

	for _, listener := range listeners {
		// Run in goroutine to avoid blocking
		go listener.OnEvent(event)
	}
}

// Log emits a log event
func (e *BaseEngine) Log(message string) {
	e.Emit(EngineEvent{
		EngineID:  e.id,
		Type:      EventTypeLog,
		Component: e.category,
		Message:   message,
		Timestamp: time.Now(),
	})
}

// LogError emits an error event
func (e *BaseEngine) LogError(err error, message string) {
	e.Emit(EngineEvent{
		EngineID:  e.id,
		Type:      EventTypeError,
		Component: e.category,
		Message:   message,
		Error:     err,
		Timestamp: time.Now(),
	})
}

// WaitForDependencies waits for all dependencies to be satisfied
func (e *BaseEngine) WaitForDependencies(ctx context.Context) error {
	if len(e.dependencies) == 0 {
		return nil
	}

	if e.depResolver == nil {
		return fmt.Errorf("dependency resolver not set")
	}

	e.SetState(EngineStateWaiting)
	e.Log(fmt.Sprintf("Waiting for dependencies: %v", e.dependencies))

	return e.depResolver.WaitFor(ctx, e.dependencies)
}

// InjectDependencyOutputs retrieves outputs from dependency engines
func (e *BaseEngine) InjectDependencyOutputs(spec map[string]interface{}) error {
	if e.depResolver == nil {
		return nil
	}

	for _, depID := range e.dependencies {
		// Try to get common outputs from dependencies
		commonKeys := []string{"cluster_endpoint", "vpc_id", "namespace", "argocd_url"}
		for _, key := range commonKeys {
			if val, err := e.depResolver.GetOutput(depID, key); err == nil && val != nil {
				spec[key] = val
			}
		}
	}

	return nil
}

// SetMockMode enables or disables mock mode
func (e *BaseEngine) SetMockMode(enabled bool, config *MockConfig) {
	e.mockMode = enabled
	e.mockConfig = config
}

// IsMockMode returns whether mock mode is enabled
func (e *BaseEngine) IsMockMode() bool {
	return e.mockMode
}

// GetContext returns the engine's context
func (e *BaseEngine) GetContext() context.Context {
	return e.ctx
}

// SetRollbackPlan sets the rollback plan
func (e *BaseEngine) SetRollbackPlan(plan *RollbackPlan) {
	e.rollbackPlan = plan
}

// GetRollbackPlan returns the rollback plan
func (e *BaseEngine) GetRollbackPlan() *RollbackPlan {
	return e.rollbackPlan
}

// GetStartedAt returns when the engine started
func (e *BaseEngine) GetStartedAt() *time.Time {
	return e.startedAt
}

// GetDuration returns how long the engine has been running
func (e *BaseEngine) GetDuration() time.Duration {
	if e.startedAt == nil {
		return 0
	}
	return time.Since(*e.startedAt)
}

// Validate is a placeholder for validation - should be overridden by specific engines
func (e *BaseEngine) Validate(spec map[string]interface{}) error {
	return nil
}

// Plan is a placeholder for planning - should be overridden by specific engines
func (e *BaseEngine) Plan(spec map[string]interface{}) (*Plan, error) {
	return &Plan{
		Actions:     []PlanAction{},
		Description: "No plan available",
	}, nil
}

// Apply is a placeholder for applying - should be overridden by specific engines
func (e *BaseEngine) Apply(spec map[string]interface{}) (*Result, error) {
	return &Result{
		Status:  "success",
		Message: "No operation performed",
	}, nil
}

// Delete is a placeholder for deletion - should be overridden by specific engines
func (e *BaseEngine) Delete() error {
	return nil
}

// Rollback attempts to rollback changes made by this engine
func (e *BaseEngine) Rollback() error {
	if e.rollbackPlan == nil {
		return fmt.Errorf("no rollback plan available")
	}

	e.SetState(EngineStateRollingBack)
	e.Log("Starting rollback")

	// Rollback actions in reverse order
	for i := len(e.rollbackPlan.Actions) - 1; i >= 0; i-- {
		action := e.rollbackPlan.Actions[i]
		e.Log(fmt.Sprintf("Rolling back: %s %s", action.Type, action.ResourceName))
	}

	e.SetState(EngineStateIdle)
	return nil
}

// HealthCheck performs a basic health check
func (e *BaseEngine) HealthCheck() (*HealthStatus, error) {
	return &HealthStatus{
		Healthy: true,
		Message: "Engine is healthy",
		Checks:  []HealthCheck{},
	}, nil
}

// CreateRollbackPlan creates a rollback plan from the current state
func (e *BaseEngine) CreateRollbackPlan(plan *Plan) *RollbackPlan {
	rollback := &RollbackPlan{
		EngineID: e.id,
		Actions:  make([]RollbackAction, 0),
	}

	for _, action := range plan.Actions {
		switch action.Type {
		case "create":
			rollback.Actions = append(rollback.Actions, RollbackAction{
				Type:         RollbackActionDelete,
				ResourceName: action.Resource,
			})
		case "update":
			rollback.Actions = append(rollback.Actions, RollbackAction{
				Type:          RollbackActionRestore,
				ResourceName:  action.Resource,
				PreviousState: action.Details,
			})
		}
	}

	return rollback
}
