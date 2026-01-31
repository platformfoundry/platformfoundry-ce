package workflow

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
)

// Engine manages workflow execution
type Engine struct {
	workflows    map[string]*Workflow
	executions   map[string]*WorkflowExecution
	mu           sync.RWMutex

	conditionCheckers map[ConditionType]ConditionChecker
	notifiers         map[string]Notifier

	subscribers []ExecutionListener
	subMu       sync.RWMutex
}

// ConditionChecker interface for checking workflow conditions
type ConditionChecker interface {
	Check(ctx context.Context, condition WorkflowCondition, exec *WorkflowExecution) (*ConditionResult, error)
}

// Notifier interface for sending notifications
type Notifier interface {
	Notify(ctx context.Context, config NotificationConfig, exec *WorkflowExecution, event string) error
}

// ExecutionListener receives workflow execution events
type ExecutionListener interface {
	OnExecutionEvent(exec *WorkflowExecution, event string)
}

// NewEngine creates a new workflow engine
func NewEngine() *Engine {
	e := &Engine{
		workflows:         make(map[string]*Workflow),
		executions:        make(map[string]*WorkflowExecution),
		conditionCheckers: make(map[ConditionType]ConditionChecker),
		notifiers:         make(map[string]Notifier),
		subscribers:       make([]ExecutionListener, 0),
	}

	// Register default condition checkers
	e.RegisterConditionChecker(ConditionTestsPassing, &TestsPassingChecker{})
	e.RegisterConditionChecker(ConditionSecurityScan, &SecurityScanChecker{})
	e.RegisterConditionChecker(ConditionTestCoverage, &TestCoverageChecker{})
	e.RegisterConditionChecker(ConditionPerformanceTest, &PerformanceTestChecker{})

	return e
}

// RegisterWorkflow registers a workflow definition
func (e *Engine) RegisterWorkflow(wf *Workflow) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	if wf.Name == "" {
		return fmt.Errorf("workflow name is required")
	}

	e.workflows[wf.Name] = wf
	return nil
}

// GetWorkflow retrieves a workflow by name
func (e *Engine) GetWorkflow(name string) (*Workflow, error) {
	e.mu.RLock()
	defer e.mu.RUnlock()

	wf, ok := e.workflows[name]
	if !ok {
		return nil, fmt.Errorf("workflow not found: %s", name)
	}
	return wf, nil
}

// ListWorkflows returns all registered workflows
func (e *Engine) ListWorkflows() []*Workflow {
	e.mu.RLock()
	defer e.mu.RUnlock()

	result := make([]*Workflow, 0, len(e.workflows))
	for _, wf := range e.workflows {
		result = append(result, wf)
	}
	return result
}

// RegisterConditionChecker registers a condition checker
func (e *Engine) RegisterConditionChecker(condType ConditionType, checker ConditionChecker) {
	e.conditionCheckers[condType] = checker
}

// RegisterNotifier registers a notifier
func (e *Engine) RegisterNotifier(notifyType string, notifier Notifier) {
	e.notifiers[notifyType] = notifier
}

// Subscribe adds an execution listener
func (e *Engine) Subscribe(listener ExecutionListener) {
	e.subMu.Lock()
	defer e.subMu.Unlock()
	e.subscribers = append(e.subscribers, listener)
}

// StartExecution initiates a workflow execution
func (e *Engine) StartExecution(ctx context.Context, workflowName string, requester string, target WorkflowTarget, action string) (*WorkflowExecution, error) {
	e.mu.Lock()

	wf, ok := e.workflows[workflowName]
	if !ok {
		e.mu.Unlock()
		return nil, fmt.Errorf("workflow not found: %s", workflowName)
	}

	// Check if workflow applies to this target/action
	if !e.matchesTrigger(wf.Trigger, target, action) {
		e.mu.Unlock()
		return nil, fmt.Errorf("workflow %s does not apply to this action/target", workflowName)
	}

	exec := &WorkflowExecution{
		ID:               uuid.New().String(),
		WorkflowName:     workflowName,
		Status:           WorkflowStatusPending,
		Requester:        requester,
		RequestedAt:      time.Now(),
		Target:           target,
		Action:           action,
		ConditionResults: make([]ConditionResult, 0),
		Approvals:        make([]ApprovalRecord, 0),
		Metadata:         make(map[string]interface{}),
	}

	e.executions[exec.ID] = exec
	e.mu.Unlock()

	// Start async execution
	go e.runExecution(ctx, exec, wf)

	return exec, nil
}

// matchesTrigger checks if a workflow trigger matches the target and action
func (e *Engine) matchesTrigger(trigger WorkflowTrigger, target WorkflowTarget, action string) bool {
	if trigger.Action != "" && trigger.Action != action {
		return false
	}

	if trigger.Target.Environment != "" && trigger.Target.Environment != target.Environment {
		return false
	}

	if trigger.Target.Service != "" && trigger.Target.Service != target.Service {
		return false
	}

	if trigger.Target.Team != "" && trigger.Target.Team != target.Team {
		return false
	}

	return true
}

// runExecution executes the workflow asynchronously
func (e *Engine) runExecution(ctx context.Context, exec *WorkflowExecution, wf *Workflow) {
	now := time.Now()
	exec.StartedAt = &now

	// Step 1: Check change window
	if wf.ChangeWindow != nil {
		if !e.isWithinChangeWindow(wf.ChangeWindow, time.Now()) {
			exec.Status = WorkflowStatusBlocked
			exec.Error = "deployment blocked: outside of allowed change window"
			e.notifyListeners(exec, "blocked")
			e.sendNotifications(ctx, wf.Notifications, exec, "blocked")
			return
		}
	}

	// Step 2: Check conditions
	exec.Status = WorkflowStatusConditions
	e.notifyListeners(exec, "conditions_started")

	allConditionsPassed := true
	for _, cond := range wf.Conditions {
		result := e.checkCondition(ctx, cond, exec)
		exec.ConditionResults = append(exec.ConditionResults, *result)

		if result.Status == ConditionStatusFailed && cond.Required {
			allConditionsPassed = false
		}
	}

	if !allConditionsPassed {
		exec.Status = WorkflowStatusFailed
		exec.Error = "one or more required conditions failed"
		completedAt := time.Now()
		exec.CompletedAt = &completedAt
		e.notifyListeners(exec, "conditions_failed")
		e.sendNotifications(ctx, wf.Notifications, exec, "conditions_failed")
		return
	}

	e.notifyListeners(exec, "conditions_passed")

	// Step 3: Request approvals (if required)
	if wf.Approvals.Required > 0 {
		exec.Status = WorkflowStatusAwaitApproval
		e.notifyListeners(exec, "approvals_requested")
		e.sendNotifications(ctx, wf.Notifications, exec, "approval_requested")

		// Wait for approvals with timeout
		if err := e.waitForApprovals(ctx, exec, wf.Approvals); err != nil {
			if exec.Status != WorkflowStatusRejected {
				exec.Status = WorkflowStatusTimedOut
				exec.Error = fmt.Sprintf("approval timeout: %v", err)
			}
			completedAt := time.Now()
			exec.CompletedAt = &completedAt
			e.notifyListeners(exec, "approval_timeout")
			e.sendNotifications(ctx, wf.Notifications, exec, "approval_timeout")
			return
		}
	}

	// Step 4: Execute the action
	exec.Status = WorkflowStatusExecuting
	e.notifyListeners(exec, "executing")
	e.sendNotifications(ctx, wf.Notifications, exec, "executing")

	// Mark as completed (actual execution is handled by caller)
	exec.Status = WorkflowStatusCompleted
	completedAt := time.Now()
	exec.CompletedAt = &completedAt
	e.notifyListeners(exec, "completed")
	e.sendNotifications(ctx, wf.Notifications, exec, "completed")
}

// checkCondition checks a single condition
func (e *Engine) checkCondition(ctx context.Context, cond WorkflowCondition, exec *WorkflowExecution) *ConditionResult {
	checker, ok := e.conditionCheckers[cond.Type]
	if !ok {
		return &ConditionResult{
			Type:      cond.Type,
			Status:    ConditionStatusSkipped,
			Message:   fmt.Sprintf("no checker registered for condition type: %s", cond.Type),
			CheckedAt: time.Now(),
		}
	}

	result, err := checker.Check(ctx, cond, exec)
	if err != nil {
		return &ConditionResult{
			Type:      cond.Type,
			Status:    ConditionStatusFailed,
			Message:   fmt.Sprintf("condition check error: %v", err),
			CheckedAt: time.Now(),
		}
	}

	return result
}

// isWithinChangeWindow checks if the current time is within allowed change windows
func (e *Engine) isWithinChangeWindow(config *ChangeWindowConfig, t time.Time) bool {
	// Check blocked times first
	for _, blocked := range config.Blocked {
		if e.isBlocked(blocked, t) {
			return false
		}
	}

	// Check allowed windows
	for _, window := range config.Allowed {
		if e.isInWindow(window, t) {
			return true
		}
	}

	// If no allowed windows defined, allow by default
	return len(config.Allowed) == 0
}

// isBlocked checks if time is in a blocked period
func (e *Engine) isBlocked(blocked BlockedTime, t time.Time) bool {
	dayName := t.Weekday().String()[:3]

	// Check blocked days
	for _, day := range blocked.Days {
		if day == dayName {
			return true
		}
	}

	// Check blocked dates
	dateStr := t.Format("2006-01-02")
	for _, date := range blocked.Dates {
		if date == dateStr {
			return true
		}
	}

	return false
}

// isInWindow checks if time is within an allowed window
func (e *Engine) isInWindow(window TimeWindow, t time.Time) bool {
	dayName := t.Weekday().String()[:3]

	// Check if day is allowed
	dayAllowed := false
	for _, day := range window.Days {
		if day == dayName {
			dayAllowed = true
			break
		}
	}
	if !dayAllowed {
		return false
	}

	// Parse hours range (e.g., "10:00-16:00")
	if window.Hours == "" {
		return true // No hour restriction
	}

	var startHour, startMin, endHour, endMin int
	_, err := fmt.Sscanf(window.Hours, "%d:%d-%d:%d", &startHour, &startMin, &endHour, &endMin)
	if err != nil {
		return false
	}

	currentMinutes := t.Hour()*60 + t.Minute()
	startMinutes := startHour*60 + startMin
	endMinutes := endHour*60 + endMin

	return currentMinutes >= startMinutes && currentMinutes <= endMinutes
}

// waitForApprovals waits for required approvals
func (e *Engine) waitForApprovals(ctx context.Context, exec *WorkflowExecution, config ApprovalConfig) error {
	timeout := time.After(config.Timeout)
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-timeout:
			return fmt.Errorf("approval timeout after %v", config.Timeout)
		case <-ticker.C:
			e.mu.RLock()
			approvalCount := 0
			rejected := false
			for _, approval := range exec.Approvals {
				if approval.Decision == "approved" {
					approvalCount++
				} else if approval.Decision == "rejected" {
					rejected = true
				}
			}
			e.mu.RUnlock()

			if rejected {
				exec.Status = WorkflowStatusRejected
				return fmt.Errorf("workflow rejected by approver")
			}

			if approvalCount >= config.Required {
				exec.Status = WorkflowStatusApproved
				return nil
			}
		}
	}
}

// Approve records an approval for a workflow execution
func (e *Engine) Approve(executionID string, user string, role string, comment string) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	exec, ok := e.executions[executionID]
	if !ok {
		return fmt.Errorf("execution not found: %s", executionID)
	}

	if exec.Status != WorkflowStatusAwaitApproval {
		return fmt.Errorf("execution is not awaiting approval: %s", exec.Status)
	}

	// Check for duplicate approval
	for _, existing := range exec.Approvals {
		if existing.User == user {
			return fmt.Errorf("user %s has already responded to this workflow", user)
		}
	}

	exec.Approvals = append(exec.Approvals, ApprovalRecord{
		User:      user,
		Role:      role,
		Decision:  "approved",
		Comment:   comment,
		Timestamp: time.Now(),
	})

	return nil
}

// Reject records a rejection for a workflow execution
func (e *Engine) Reject(executionID string, user string, role string, comment string) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	exec, ok := e.executions[executionID]
	if !ok {
		return fmt.Errorf("execution not found: %s", executionID)
	}

	if exec.Status != WorkflowStatusAwaitApproval {
		return fmt.Errorf("execution is not awaiting approval: %s", exec.Status)
	}

	exec.Approvals = append(exec.Approvals, ApprovalRecord{
		User:      user,
		Role:      role,
		Decision:  "rejected",
		Comment:   comment,
		Timestamp: time.Now(),
	})

	exec.Status = WorkflowStatusRejected

	return nil
}

// GetExecution retrieves an execution by ID
func (e *Engine) GetExecution(id string) (*WorkflowExecution, error) {
	e.mu.RLock()
	defer e.mu.RUnlock()

	exec, ok := e.executions[id]
	if !ok {
		return nil, fmt.Errorf("execution not found: %s", id)
	}
	return exec, nil
}

// ListExecutions returns all executions, optionally filtered
func (e *Engine) ListExecutions(workflowName string, status WorkflowStatus) []*WorkflowExecution {
	e.mu.RLock()
	defer e.mu.RUnlock()

	result := make([]*WorkflowExecution, 0)
	for _, exec := range e.executions {
		if workflowName != "" && exec.WorkflowName != workflowName {
			continue
		}
		if status != "" && exec.Status != status {
			continue
		}
		result = append(result, exec)
	}
	return result
}

// notifyListeners notifies all registered listeners
func (e *Engine) notifyListeners(exec *WorkflowExecution, event string) {
	e.subMu.RLock()
	listeners := make([]ExecutionListener, len(e.subscribers))
	copy(listeners, e.subscribers)
	e.subMu.RUnlock()

	for _, listener := range listeners {
		go listener.OnExecutionEvent(exec, event)
	}
}

// sendNotifications sends notifications for an event
func (e *Engine) sendNotifications(ctx context.Context, configs []NotificationConfig, exec *WorkflowExecution, event string) {
	for _, config := range configs {
		// Check if notification should be sent for this event
		if len(config.OnEvents) > 0 {
			shouldNotify := false
			for _, ev := range config.OnEvents {
				if ev == event {
					shouldNotify = true
					break
				}
			}
			if !shouldNotify {
				continue
			}
		}

		notifier, ok := e.notifiers[config.Type]
		if !ok {
			continue
		}

		go notifier.Notify(ctx, config, exec, event)
	}
}
