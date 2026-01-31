// Package engine provides the component engine architecture for parallel execution
// of platform components with dependency management and isolation.
package engine

import (
	"context"
	"time"
)

// EngineState represents the current state of an engine
type EngineState string

const (
	EngineStateIdle         EngineState = "idle"
	EngineStateInitializing EngineState = "initializing"
	EngineStateWaiting      EngineState = "waiting_dependencies"
	EngineStateRunning      EngineState = "running"
	EngineStateCompleted    EngineState = "completed"
	EngineStateFailed       EngineState = "failed"
	EngineStateCancelled    EngineState = "cancelled"
	EngineStateRollingBack  EngineState = "rolling_back"
	EngineStatePaused       EngineState = "paused"
)

// String returns the string representation of EngineState
func (s EngineState) String() string {
	return string(s)
}

// IsTerminal returns true if the state is a terminal state (completed, failed, or cancelled)
func (s EngineState) IsTerminal() bool {
	return s == EngineStateCompleted || s == EngineStateFailed || s == EngineStateCancelled
}

// IsRunning returns true if the engine is actively working
func (s EngineState) IsRunning() bool {
	return s == EngineStateRunning || s == EngineStateInitializing || s == EngineStateRollingBack
}

// EventType represents the type of engine event
type EventType string

const (
	EventTypeStateChange    EventType = "state_change"
	EventTypeProgress       EventType = "progress"
	EventTypeLog            EventType = "log"
	EventTypeError          EventType = "error"
	EventTypeToolSelected   EventType = "tool_selected"
	EventTypeRollbackStart  EventType = "rollback_started"
	EventTypeRollbackFailed EventType = "rollback_failed"
	EventTypeOutputSet      EventType = "output_set"
	EventTypeDependencyMet  EventType = "dependency_met"
)

// EngineEvent represents events emitted by engines
type EngineEvent struct {
	EngineID  string
	Type      EventType
	Component string
	Progress  int
	Message   string
	Error     error
	Timestamp time.Time
	Metadata  map[string]interface{}
}

// ProgressInfo provides detailed progress information
type ProgressInfo struct {
	Percentage    int
	CurrentPhase  string
	PhasesTotal   int
	PhasesCurrent int
	Message       string
	StartedAt     time.Time
	EstimatedEnd  time.Time
	SubProgress   map[string]int
}

// ToolInfo describes a tool within an engine
type ToolInfo struct {
	Name          string
	DisplayName   string
	Description   string
	Version       string
	LearnMoreURL  string
	QuickStart    string
	Compatibility []CompatibilityRule
}

// CompatibilityRule defines compatibility with other tools
type CompatibilityRule struct {
	Tool       string
	Compatible bool
	Notes      string
}

// Plan represents an execution plan
type Plan struct {
	Actions     []PlanAction
	Resources   []string
	Description string
}

// PlanAction represents a single action in a plan
type PlanAction struct {
	Type        string // create, update, delete, no-op
	Resource    string
	Description string
	Details     map[string]interface{}
}

// Result represents the result of an engine operation
type Result struct {
	Status    string
	Message   string
	Resources []string
	Outputs   map[string]interface{}
	Duration  time.Duration
}

// HealthStatus represents the health of an engine
type HealthStatus struct {
	Healthy bool
	Message string
	Checks  []HealthCheck
}

// HealthCheck represents a single health check
type HealthCheck struct {
	Name    string
	Status  string // pass, warn, fail
	Message string
}

// EngineConfig holds configuration for an engine
type EngineConfig struct {
	Name          string
	Category      string
	Dependencies  []string
	MockMode      bool
	MockConfig    *MockConfig
	Timeout       time.Duration
	RetryCount    int
	RetryDelay    time.Duration
	PluginManager interface{}
	Store         interface{}
}

// MockConfig configures mock behavior
type MockConfig struct {
	Mode            MockMode
	SimulatedDelay  time.Duration
	FailureRate     float64
	FailureTools    []string
	ResponseOverride map[string]interface{}
}

// MockMode defines the mock behavior
type MockMode string

const (
	MockModeInstant   MockMode = "instant"
	MockModeRealistic MockMode = "realistic"
	MockModeRecorded  MockMode = "recorded"
	MockModeChaos     MockMode = "chaos"
)

// Engine interface for component-specific engines
type Engine interface {
	// Identity
	ID() string
	Name() string
	Category() string

	// Lifecycle
	Initialize(ctx context.Context, config EngineConfig) error
	Start(ctx context.Context) error
	Stop() error
	Pause() error
	Resume() error

	// State
	State() EngineState
	Progress() ProgressInfo
	Outputs() map[string]interface{}

	// Dependencies
	DependsOn() []string
	SetDependencyResolver(resolver DependencyResolver)

	// Execution
	Validate(spec map[string]interface{}) error
	Plan(spec map[string]interface{}) (*Plan, error)
	Apply(spec map[string]interface{}) (*Result, error)
	Delete() error
	Rollback() error

	// Health
	HealthCheck() (*HealthStatus, error)

	// Events
	Subscribe(listener EventListener)
	Unsubscribe(listener EventListener)
}

// EventListener receives engine events
type EventListener interface {
	OnEvent(event EngineEvent)
}

// DependencyResolver checks if dependencies are satisfied
type DependencyResolver interface {
	IsSatisfied(engineID string, dependencies []string) bool
	WaitFor(ctx context.Context, dependencies []string) error
	GetOutput(engineID string, key string) (interface{}, error)
	MarkCompleted(engineID string, outputs map[string]interface{})
}

// MockableEngine is an engine that can be put in mock mode
type MockableEngine interface {
	Engine
	SetMockMode(enabled bool, config *MockConfig)
	IsMockMode() bool
}

// EngineStatus represents the status of an engine
type EngineStatus struct {
	ID        string
	Name      string
	Category  string
	State     EngineState
	Progress  int
	Message   string
	Tool      *ToolInfo
	StartedAt *time.Time
	Duration  time.Duration
	Error     error
}

// RollbackPlan contains information needed to rollback an engine's changes
type RollbackPlan struct {
	EngineID string
	Actions  []RollbackAction
}

// RollbackAction represents a single rollback action
type RollbackAction struct {
	Type          RollbackActionType
	ResourceName  string
	ResourceKind  string
	PreviousState map[string]interface{}
}

// RollbackActionType defines the type of rollback action
type RollbackActionType string

const (
	RollbackActionDelete  RollbackActionType = "delete"
	RollbackActionRestore RollbackActionType = "restore"
)
