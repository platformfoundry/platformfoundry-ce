package chaos

import (
	"context"
	"fmt"
	"math/rand"
	"sync"
	"time"

	"github.com/platformfoundry/platformfoundry-ce/pkg/types"
)

// Engine manages chaos experiments
type Engine struct {
	executor     Executor
	healthChecker HealthChecker
	experiments  map[string]*types.ChaosExperiment
	activeRuns   map[string]*ExperimentRun
	mu           sync.RWMutex
	eventChan    chan ChaosEvent
}

// Executor executes chaos actions
type Executor interface {
	Execute(ctx context.Context, action types.ChaosAction, target types.ChaosTarget) error
	Rollback(ctx context.Context, action types.ChaosAction, target types.ChaosTarget) error
	SupportsAction(actionType types.ChaosActionType) bool
}

// HealthChecker checks target health
type HealthChecker interface {
	Check(ctx context.Context, target types.ChaosTarget) (bool, error)
	CheckEndpoint(ctx context.Context, endpoint string) (bool, error)
}

// ExperimentRun represents an active experiment run
type ExperimentRun struct {
	ID           string
	Experiment   *types.ChaosExperiment
	StartTime    time.Time
	Actions      []types.ActionResult
	CurrentAction int
	Status       string
	StopChan     chan struct{}
}

// ChaosEvent represents an event during chaos execution
type ChaosEvent struct {
	Type       string    `json:"type"` // started, action_started, action_completed, health_check, completed, failed
	Experiment string    `json:"experiment"`
	Action     string    `json:"action,omitempty"`
	Timestamp  time.Time `json:"timestamp"`
	Message    string    `json:"message"`
	Metadata   map[string]interface{} `json:"metadata,omitempty"`
}

// EngineConfig configures the chaos engine
type EngineConfig struct {
	Executor      Executor
	HealthChecker HealthChecker
	EventBufferSize int
}

// NewEngine creates a new chaos engine
func NewEngine(cfg EngineConfig) *Engine {
	bufferSize := cfg.EventBufferSize
	if bufferSize == 0 {
		bufferSize = 100
	}

	return &Engine{
		executor:     cfg.Executor,
		healthChecker: cfg.HealthChecker,
		experiments:  make(map[string]*types.ChaosExperiment),
		activeRuns:   make(map[string]*ExperimentRun),
		eventChan:    make(chan ChaosEvent, bufferSize),
	}
}

// RegisterExperiment adds an experiment to the engine
func (e *Engine) RegisterExperiment(exp *types.ChaosExperiment) error {
	if exp.Metadata.Name == "" {
		return fmt.Errorf("experiment name is required")
	}

	e.mu.Lock()
	defer e.mu.Unlock()

	e.experiments[exp.Metadata.Name] = exp

	if exp.Status == nil {
		exp.Status = &types.ChaosExperimentStatus{
			Phase: types.ChaosPhaseCreated,
		}
	}

	return nil
}

// GetExperiment retrieves an experiment by name
func (e *Engine) GetExperiment(name string) (*types.ChaosExperiment, error) {
	e.mu.RLock()
	defer e.mu.RUnlock()

	exp, ok := e.experiments[name]
	if !ok {
		return nil, fmt.Errorf("experiment not found: %s", name)
	}
	return exp, nil
}

// ListExperiments returns all registered experiments
func (e *Engine) ListExperiments() []*types.ChaosExperiment {
	e.mu.RLock()
	defer e.mu.RUnlock()

	result := make([]*types.ChaosExperiment, 0, len(e.experiments))
	for _, exp := range e.experiments {
		result = append(result, exp)
	}
	return result
}

// RunExperiment executes an experiment
func (e *Engine) RunExperiment(ctx context.Context, name string) (*types.ChaosReport, error) {
	exp, err := e.GetExperiment(name)
	if err != nil {
		return nil, err
	}

	// Check if already running
	e.mu.Lock()
	if _, exists := e.activeRuns[name]; exists {
		e.mu.Unlock()
		return nil, fmt.Errorf("experiment %s is already running", name)
	}

	// Check safety - is it paused?
	if exp.Spec.Safety.Paused {
		e.mu.Unlock()
		return nil, fmt.Errorf("experiment %s is paused", name)
	}

	// Create run
	runID := fmt.Sprintf("%s-%d", name, time.Now().Unix())
	run := &ExperimentRun{
		ID:         runID,
		Experiment: exp,
		StartTime:  time.Now(),
		Actions:    make([]types.ActionResult, 0),
		Status:     "running",
		StopChan:   make(chan struct{}),
	}
	e.activeRuns[name] = run
	e.mu.Unlock()

	// Update experiment status
	exp.Status.Phase = types.ChaosPhaseRunning
	now := time.Now()
	exp.Status.StartTime = &now

	e.emitEvent(ChaosEvent{
		Type:       "started",
		Experiment: name,
		Timestamp:  time.Now(),
		Message:    fmt.Sprintf("Experiment %s started", name),
	})

	// Execute actions
	report, err := e.executeExperiment(ctx, run)

	// Clean up
	e.mu.Lock()
	delete(e.activeRuns, name)
	e.mu.Unlock()

	// Update status
	endTime := time.Now()
	exp.Status.EndTime = &endTime
	exp.Status.LastRunTime = &endTime

	if err != nil {
		exp.Status.Phase = types.ChaosPhaseFailed
		exp.Status.LastRunResult = "failed"
		exp.Status.FailedRuns++
	} else {
		exp.Status.Phase = types.ChaosPhaseCompleted
		exp.Status.LastRunResult = "success"
		exp.Status.SuccessfulRuns++
	}

	// Record history
	historyEntry := types.ChaosRunHistory{
		RunID:     runID,
		StartTime: run.StartTime,
		EndTime:   &endTime,
		Result:    exp.Status.LastRunResult,
		Actions:   run.Actions,
	}
	if err != nil {
		historyEntry.Message = err.Error()
	}

	if exp.Status.History == nil {
		exp.Status.History = make([]types.ChaosRunHistory, 0)
	}
	exp.Status.History = append(exp.Status.History, historyEntry)

	// Keep only last 10 history entries
	if len(exp.Status.History) > 10 {
		exp.Status.History = exp.Status.History[len(exp.Status.History)-10:]
	}

	return report, err
}

// StopExperiment stops a running experiment
func (e *Engine) StopExperiment(name string) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	run, exists := e.activeRuns[name]
	if !exists {
		return fmt.Errorf("experiment %s is not running", name)
	}

	close(run.StopChan)
	run.Status = "aborted"

	if exp, ok := e.experiments[name]; ok {
		exp.Status.Phase = types.ChaosPhaseAborted
	}

	e.emitEvent(ChaosEvent{
		Type:       "aborted",
		Experiment: name,
		Timestamp:  time.Now(),
		Message:    fmt.Sprintf("Experiment %s aborted", name),
	})

	return nil
}

// executeExperiment runs all actions in an experiment
func (e *Engine) executeExperiment(ctx context.Context, run *ExperimentRun) (*types.ChaosReport, error) {
	exp := run.Experiment
	report := &types.ChaosReport{
		Experiment:  exp.Metadata.Name,
		Environment: exp.Spec.Target.Environment,
		StartTime:   run.StartTime,
		TotalActions: len(exp.Spec.Experiments),
		Findings:    make([]types.ChaosFinding, 0),
	}

	// Initial health check
	if e.healthChecker != nil {
		healthy, err := e.healthChecker.Check(ctx, exp.Spec.Target)
		if err != nil || !healthy {
			return nil, fmt.Errorf("pre-experiment health check failed: %w", err)
		}
	}

	// Execute each action
	for i, action := range exp.Spec.Experiments {
		select {
		case <-ctx.Done():
			return report, ctx.Err()
		case <-run.StopChan:
			return report, fmt.Errorf("experiment stopped")
		default:
		}

		run.CurrentAction = i
		exp.Status.CurrentAction = action.Name

		// Check probability
		if action.Probability > 0 && action.Probability < 1 {
			if rand.Float64() > action.Probability {
				// Skip this action based on probability
				continue
			}
		}

		result, finding := e.executeAction(ctx, run, action)
		run.Actions = append(run.Actions, result)

		if result.Result == "success" {
			report.SuccessfulActions++
		} else {
			report.FailedActions++
			if finding != nil {
				report.Findings = append(report.Findings, *finding)
			}

			// Check if we should stop on failure
			if exp.Spec.Safety.StopOnFailure {
				break
			}
		}

		// Health check between actions
		if e.healthChecker != nil {
			interval, _ := time.ParseDuration(exp.Spec.Safety.HealthCheckInterval)
			if interval == 0 {
				interval = 30 * time.Second
			}
			time.Sleep(interval)

			healthy, _ := e.healthChecker.Check(ctx, exp.Spec.Target)
			if !healthy && exp.Spec.Safety.RollbackOnError {
				e.rollbackAction(ctx, action, exp.Spec.Target)
				report.Findings = append(report.Findings, types.ChaosFinding{
					Severity:    "high",
					Component:   exp.Spec.Target.Service,
					Description: "Health check failed after action " + action.Name,
					Impact:      "Service degradation detected",
					Remediation: "Automatic rollback was triggered",
				})
			}
		}
	}

	report.EndTime = time.Now()
	report.Duration = report.EndTime.Sub(report.StartTime).String()

	// Determine overall result
	if report.FailedActions == 0 {
		report.OverallResult = "success"
	} else if report.SuccessfulActions > 0 {
		report.OverallResult = "partial"
	} else {
		report.OverallResult = "failed"
	}

	// Generate recommendations based on findings
	report.Recommendations = e.generateRecommendations(report.Findings)

	e.emitEvent(ChaosEvent{
		Type:       "completed",
		Experiment: exp.Metadata.Name,
		Timestamp:  time.Now(),
		Message:    fmt.Sprintf("Experiment completed: %s", report.OverallResult),
		Metadata: map[string]interface{}{
			"successful": report.SuccessfulActions,
			"failed":     report.FailedActions,
			"findings":   len(report.Findings),
		},
	})

	return report, nil
}

// executeAction executes a single chaos action
func (e *Engine) executeAction(ctx context.Context, run *ExperimentRun, action types.ChaosAction) (types.ActionResult, *types.ChaosFinding) {
	exp := run.Experiment
	startTime := time.Now()

	e.emitEvent(ChaosEvent{
		Type:       "action_started",
		Experiment: exp.Metadata.Name,
		Action:     action.Name,
		Timestamp:  startTime,
		Message:    fmt.Sprintf("Starting action: %s (%s)", action.Name, action.Type),
	})

	result := types.ActionResult{
		Name:      action.Name,
		Type:      string(action.Type),
		StartTime: startTime,
	}

	// Parse duration
	duration, err := time.ParseDuration(action.Duration)
	if err != nil {
		duration = 1 * time.Minute
	}

	// Execute action
	if e.executor != nil {
		err = e.executor.Execute(ctx, action, exp.Spec.Target)
	} else {
		// Simulate action
		err = e.simulateAction(ctx, action, duration)
	}

	result.EndTime = time.Now()

	var finding *types.ChaosFinding

	if err != nil {
		result.Result = "failed"
		result.Message = err.Error()
		finding = &types.ChaosFinding{
			Severity:    "medium",
			Component:   exp.Spec.Target.Service,
			Description: fmt.Sprintf("Action %s failed: %v", action.Name, err),
			Impact:      "Unknown - action execution failed",
		}
	} else {
		result.Result = "success"
		result.Message = fmt.Sprintf("Action %s completed successfully", action.Name)
	}

	e.emitEvent(ChaosEvent{
		Type:       "action_completed",
		Experiment: exp.Metadata.Name,
		Action:     action.Name,
		Timestamp:  result.EndTime,
		Message:    fmt.Sprintf("Action %s: %s", action.Name, result.Result),
	})

	return result, finding
}

// simulateAction simulates a chaos action (for testing)
func (e *Engine) simulateAction(ctx context.Context, action types.ChaosAction, duration time.Duration) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-time.After(duration):
		return nil
	}
}

// rollbackAction rolls back a chaos action
func (e *Engine) rollbackAction(ctx context.Context, action types.ChaosAction, target types.ChaosTarget) error {
	if e.executor != nil {
		return e.executor.Rollback(ctx, action, target)
	}
	return nil
}

// generateRecommendations creates recommendations based on findings
func (e *Engine) generateRecommendations(findings []types.ChaosFinding) []string {
	recommendations := make([]string, 0)

	hasPodFailures := false
	hasNetworkIssues := false
	hasResourceIssues := false

	for _, f := range findings {
		switch f.Severity {
		case "critical", "high":
			if containsAny(f.Description, "pod", "container") {
				hasPodFailures = true
			}
			if containsAny(f.Description, "network", "latency", "timeout") {
				hasNetworkIssues = true
			}
			if containsAny(f.Description, "cpu", "memory", "resource") {
				hasResourceIssues = true
			}
		}
	}

	if hasPodFailures {
		recommendations = append(recommendations,
			"Consider implementing pod disruption budgets to ensure minimum availability",
			"Review health check and readiness probe configurations",
		)
	}

	if hasNetworkIssues {
		recommendations = append(recommendations,
			"Implement circuit breakers for external dependencies",
			"Consider adding retry logic with exponential backoff",
			"Review timeout configurations for all network calls",
		)
	}

	if hasResourceIssues {
		recommendations = append(recommendations,
			"Review resource limits and requests for all containers",
			"Implement horizontal pod autoscaling",
			"Consider adding resource quotas",
		)
	}

	if len(findings) > 0 {
		recommendations = append(recommendations,
			"Document findings and create action items for remediation",
			"Schedule follow-up chaos experiments after implementing fixes",
		)
	}

	return recommendations
}

// emitEvent sends an event to the event channel
func (e *Engine) emitEvent(event ChaosEvent) {
	select {
	case e.eventChan <- event:
	default:
		// Channel full, drop event
	}
}

// Events returns the event channel
func (e *Engine) Events() <-chan ChaosEvent {
	return e.eventChan
}

// GetActiveRuns returns all active experiment runs
func (e *Engine) GetActiveRuns() map[string]*ExperimentRun {
	e.mu.RLock()
	defer e.mu.RUnlock()

	result := make(map[string]*ExperimentRun)
	for k, v := range e.activeRuns {
		result[k] = v
	}
	return result
}

// Helper function
func containsAny(s string, substrs ...string) bool {
	for _, sub := range substrs {
		if len(s) >= len(sub) {
			for i := 0; i <= len(s)-len(sub); i++ {
				if s[i:i+len(sub)] == sub {
					return true
				}
			}
		}
	}
	return false
}

// MockExecutor provides a simulated chaos executor for testing
type MockExecutor struct{}

func (m *MockExecutor) Execute(ctx context.Context, action types.ChaosAction, target types.ChaosTarget) error {
	// Simulate action execution
	duration, _ := time.ParseDuration(action.Duration)
	if duration == 0 {
		duration = 1 * time.Second
	}
	time.Sleep(duration / 10) // Shortened for testing
	return nil
}

func (m *MockExecutor) Rollback(ctx context.Context, action types.ChaosAction, target types.ChaosTarget) error {
	return nil
}

func (m *MockExecutor) SupportsAction(actionType types.ChaosActionType) bool {
	return true
}

// MockHealthChecker provides a simulated health checker for testing
type MockHealthChecker struct {
	HealthyByDefault bool
}

func (m *MockHealthChecker) Check(ctx context.Context, target types.ChaosTarget) (bool, error) {
	return m.HealthyByDefault, nil
}

func (m *MockHealthChecker) CheckEndpoint(ctx context.Context, endpoint string) (bool, error) {
	return m.HealthyByDefault, nil
}
