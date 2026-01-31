package engine

import (
	"context"
	"sync"
	"testing"
	"time"
)

func TestNewBaseEngine(t *testing.T) {
	engine := NewBaseEngine("test-engine", "test-category")

	if engine.Name() != "test-engine" {
		t.Errorf("expected name 'test-engine', got '%s'", engine.Name())
	}
	if engine.Category() != "test-category" {
		t.Errorf("expected category 'test-category', got '%s'", engine.Category())
	}
	if engine.ID() == "" {
		t.Error("expected non-empty ID")
	}
	if engine.State() != EngineStateIdle {
		t.Errorf("expected initial state to be idle, got '%s'", engine.State())
	}
}

func TestBaseEngineID(t *testing.T) {
	engine1 := NewBaseEngine("engine1", "category")
	engine2 := NewBaseEngine("engine2", "category")

	if engine1.ID() == engine2.ID() {
		t.Error("expected unique IDs for different engines")
	}
}

func TestBaseEngineStateTransitions(t *testing.T) {
	engine := NewBaseEngine("test", "category")

	// Test SetState
	engine.SetState(EngineStateRunning)
	if engine.State() != EngineStateRunning {
		t.Errorf("expected state running, got '%s'", engine.State())
	}

	engine.SetState(EngineStateCompleted)
	if engine.State() != EngineStateCompleted {
		t.Errorf("expected state completed, got '%s'", engine.State())
	}
}

func TestBaseEngineInitialize(t *testing.T) {
	engine := NewBaseEngine("test", "category")

	config := EngineConfig{
		Name:         "test",
		Category:     "category",
		Dependencies: []string{"dep1", "dep2"},
		MockMode:     true,
		MockConfig: &MockConfig{
			Mode: MockModeInstant,
		},
	}

	err := engine.Initialize(context.Background(), config)
	if err != nil {
		t.Errorf("Initialize failed: %v", err)
	}

	if !engine.IsMockMode() {
		t.Error("expected mock mode to be enabled")
	}

	deps := engine.DependsOn()
	if len(deps) != 2 {
		t.Errorf("expected 2 dependencies, got %d", len(deps))
	}
}

func TestBaseEngineStartStop(t *testing.T) {
	engine := NewBaseEngine("test", "category")

	ctx := context.Background()
	err := engine.Start(ctx)
	if err != nil {
		t.Errorf("Start failed: %v", err)
	}

	if engine.State() != EngineStateRunning {
		t.Errorf("expected state running after start, got '%s'", engine.State())
	}

	if engine.GetStartedAt() == nil {
		t.Error("expected startedAt to be set")
	}

	if engine.GetContext() == nil {
		t.Error("expected context to be set")
	}

	err = engine.Stop()
	if err != nil {
		t.Errorf("Stop failed: %v", err)
	}

	if engine.State() != EngineStateCancelled {
		t.Errorf("expected state cancelled after stop, got '%s'", engine.State())
	}
}

func TestBaseEnginePauseResume(t *testing.T) {
	engine := NewBaseEngine("test", "category")
	engine.Start(context.Background())

	err := engine.Pause()
	if err != nil {
		t.Errorf("Pause failed: %v", err)
	}
	if engine.State() != EngineStatePaused {
		t.Errorf("expected state paused, got '%s'", engine.State())
	}

	err = engine.Resume()
	if err != nil {
		t.Errorf("Resume failed: %v", err)
	}
	if engine.State() != EngineStateRunning {
		t.Errorf("expected state running after resume, got '%s'", engine.State())
	}
}

func TestBaseEngineProgress(t *testing.T) {
	engine := NewBaseEngine("test", "category")

	engine.SetProgress(50, "Processing", 2, 4)

	progress := engine.Progress()
	if progress.Percentage != 50 {
		t.Errorf("expected progress 50%%, got %d%%", progress.Percentage)
	}
	if progress.Message != "Processing" {
		t.Errorf("expected message 'Processing', got '%s'", progress.Message)
	}
	if progress.PhasesCurrent != 2 {
		t.Errorf("expected current phase 2, got %d", progress.PhasesCurrent)
	}
	if progress.PhasesTotal != 4 {
		t.Errorf("expected total phases 4, got %d", progress.PhasesTotal)
	}
}

func TestBaseEngineOutputs(t *testing.T) {
	engine := NewBaseEngine("test", "category")

	// Test SetOutput and GetOutput
	engine.SetOutput("key1", "value1")
	engine.SetOutput("key2", 42)

	val1, ok := engine.GetOutput("key1")
	if !ok || val1 != "value1" {
		t.Errorf("expected 'value1', got '%v'", val1)
	}

	val2, ok := engine.GetOutput("key2")
	if !ok || val2 != 42 {
		t.Errorf("expected 42, got '%v'", val2)
	}

	// Test non-existent key
	_, ok = engine.GetOutput("nonexistent")
	if ok {
		t.Error("expected ok=false for nonexistent key")
	}

	// Test Outputs returns a copy
	outputs := engine.Outputs()
	if len(outputs) != 2 {
		t.Errorf("expected 2 outputs, got %d", len(outputs))
	}
}

func TestBaseEngineDependencies(t *testing.T) {
	engine := NewBaseEngine("test", "category")

	engine.SetDependencies([]string{"dep1", "dep2", "dep3"})

	deps := engine.DependsOn()
	if len(deps) != 3 {
		t.Errorf("expected 3 dependencies, got %d", len(deps))
	}
}

func TestBaseEngineEventSubscription(t *testing.T) {
	engine := NewBaseEngine("test", "category")

	var receivedEvents []EngineEvent
	var mu sync.Mutex

	listener := NewFuncListener(func(event EngineEvent) {
		mu.Lock()
		receivedEvents = append(receivedEvents, event)
		mu.Unlock()
	})

	engine.Subscribe(listener)

	// Trigger events
	engine.SetState(EngineStateRunning)
	engine.Log("test message")
	engine.SetOutput("key", "value")

	// Wait for async events to be processed
	time.Sleep(50 * time.Millisecond)

	mu.Lock()
	defer mu.Unlock()

	if len(receivedEvents) < 3 {
		t.Errorf("expected at least 3 events, got %d", len(receivedEvents))
	}
}

func TestBaseEngineUnsubscribe(t *testing.T) {
	engine := NewBaseEngine("test", "category")

	var eventCount int
	var mu sync.Mutex

	listener := NewFuncListener(func(event EngineEvent) {
		mu.Lock()
		eventCount++
		mu.Unlock()
	})

	engine.Subscribe(listener)
	engine.SetState(EngineStateRunning)

	time.Sleep(50 * time.Millisecond)

	engine.Unsubscribe(listener)

	mu.Lock()
	countBefore := eventCount
	mu.Unlock()

	engine.SetState(EngineStateCompleted)
	time.Sleep(50 * time.Millisecond)

	mu.Lock()
	countAfter := eventCount
	mu.Unlock()

	if countAfter != countBefore {
		t.Error("expected no events after unsubscribe")
	}
}

func TestBaseEngineLog(t *testing.T) {
	engine := NewBaseEngine("test", "category")

	var logEvent EngineEvent
	var received bool
	var mu sync.Mutex

	listener := NewFuncListener(func(event EngineEvent) {
		mu.Lock()
		if event.Type == EventTypeLog {
			logEvent = event
			received = true
		}
		mu.Unlock()
	})

	engine.Subscribe(listener)
	engine.Log("test log message")

	time.Sleep(50 * time.Millisecond)

	mu.Lock()
	defer mu.Unlock()

	if !received {
		t.Error("expected log event")
	}
	if logEvent.Message != "test log message" {
		t.Errorf("expected 'test log message', got '%s'", logEvent.Message)
	}
}

func TestBaseEngineLogError(t *testing.T) {
	engine := NewBaseEngine("test", "category")

	var errorEvent EngineEvent
	var received bool
	var mu sync.Mutex

	listener := NewFuncListener(func(event EngineEvent) {
		mu.Lock()
		if event.Type == EventTypeError {
			errorEvent = event
			received = true
		}
		mu.Unlock()
	})

	engine.Subscribe(listener)

	testErr := &testError{msg: "test error"}
	engine.LogError(testErr, "error occurred")

	time.Sleep(50 * time.Millisecond)

	mu.Lock()
	defer mu.Unlock()

	if !received {
		t.Error("expected error event")
	}
	if errorEvent.Error == nil {
		t.Error("expected error to be set")
	}
}

type testError struct {
	msg string
}

func (e *testError) Error() string {
	return e.msg
}

func TestBaseEngineMockMode(t *testing.T) {
	engine := NewBaseEngine("test", "category")

	if engine.IsMockMode() {
		t.Error("expected mock mode to be disabled by default")
	}

	mockConfig := &MockConfig{
		Mode:           MockModeRealistic,
		SimulatedDelay: 100 * time.Millisecond,
		FailureRate:    0.1,
	}

	engine.SetMockMode(true, mockConfig)

	if !engine.IsMockMode() {
		t.Error("expected mock mode to be enabled")
	}

	engine.SetMockMode(false, nil)

	if engine.IsMockMode() {
		t.Error("expected mock mode to be disabled")
	}
}

func TestBaseEngineRollbackPlan(t *testing.T) {
	engine := NewBaseEngine("test", "category")

	plan := &RollbackPlan{
		EngineID: engine.ID(),
		Actions: []RollbackAction{
			{
				Type:         RollbackActionDelete,
				ResourceName: "resource1",
			},
		},
	}

	engine.SetRollbackPlan(plan)

	retrieved := engine.GetRollbackPlan()
	if retrieved == nil {
		t.Error("expected rollback plan to be set")
	}
	if len(retrieved.Actions) != 1 {
		t.Errorf("expected 1 action, got %d", len(retrieved.Actions))
	}
}

func TestBaseEngineRollback(t *testing.T) {
	engine := NewBaseEngine("test", "category")

	// Test rollback without plan
	err := engine.Rollback()
	if err == nil {
		t.Error("expected error when rolling back without plan")
	}

	// Test rollback with plan
	plan := &RollbackPlan{
		EngineID: engine.ID(),
		Actions: []RollbackAction{
			{Type: RollbackActionDelete, ResourceName: "resource1"},
			{Type: RollbackActionRestore, ResourceName: "resource2"},
		},
	}
	engine.SetRollbackPlan(plan)

	err = engine.Rollback()
	if err != nil {
		t.Errorf("Rollback failed: %v", err)
	}

	if engine.State() != EngineStateIdle {
		t.Errorf("expected state idle after rollback, got '%s'", engine.State())
	}
}

func TestBaseEngineCreateRollbackPlan(t *testing.T) {
	engine := NewBaseEngine("test", "category")

	plan := &Plan{
		Actions: []PlanAction{
			{Type: "create", Resource: "vpc"},
			{Type: "update", Resource: "security-group", Details: map[string]interface{}{"previous": "value"}},
			{Type: "delete", Resource: "old-resource"},
		},
	}

	rollback := engine.CreateRollbackPlan(plan)

	if rollback.EngineID != engine.ID() {
		t.Error("expected rollback plan to have engine ID")
	}

	// Should have 2 actions (create -> delete, update -> restore)
	if len(rollback.Actions) != 2 {
		t.Errorf("expected 2 rollback actions, got %d", len(rollback.Actions))
	}

	// Check create -> delete
	if rollback.Actions[0].Type != RollbackActionDelete {
		t.Errorf("expected delete action for create, got %s", rollback.Actions[0].Type)
	}
	if rollback.Actions[0].ResourceName != "vpc" {
		t.Errorf("expected resource 'vpc', got '%s'", rollback.Actions[0].ResourceName)
	}

	// Check update -> restore
	if rollback.Actions[1].Type != RollbackActionRestore {
		t.Errorf("expected restore action for update, got %s", rollback.Actions[1].Type)
	}
}

func TestBaseEngineHealthCheck(t *testing.T) {
	engine := NewBaseEngine("test", "category")

	health, err := engine.HealthCheck()
	if err != nil {
		t.Errorf("HealthCheck failed: %v", err)
	}

	if !health.Healthy {
		t.Error("expected engine to be healthy")
	}
}

func TestBaseEngineDuration(t *testing.T) {
	engine := NewBaseEngine("test", "category")

	// Before start, duration should be 0
	if engine.GetDuration() != 0 {
		t.Error("expected duration 0 before start")
	}

	engine.Start(context.Background())
	time.Sleep(50 * time.Millisecond)

	duration := engine.GetDuration()
	if duration < 50*time.Millisecond {
		t.Errorf("expected duration >= 50ms, got %v", duration)
	}
}

func TestBaseEngineValidatePlanApplyDelete(t *testing.T) {
	engine := NewBaseEngine("test", "category")

	spec := map[string]interface{}{"key": "value"}

	// Test Validate (placeholder)
	err := engine.Validate(spec)
	if err != nil {
		t.Errorf("Validate failed: %v", err)
	}

	// Test Plan (placeholder)
	plan, err := engine.Plan(spec)
	if err != nil {
		t.Errorf("Plan failed: %v", err)
	}
	if plan == nil {
		t.Error("expected non-nil plan")
	}

	// Test Apply (placeholder)
	result, err := engine.Apply(spec)
	if err != nil {
		t.Errorf("Apply failed: %v", err)
	}
	if result == nil {
		t.Error("expected non-nil result")
	}
	if result.Status != "success" {
		t.Errorf("expected status 'success', got '%s'", result.Status)
	}

	// Test Delete (placeholder)
	err = engine.Delete()
	if err != nil {
		t.Errorf("Delete failed: %v", err)
	}
}

func TestBaseEngineWaitForDependencies(t *testing.T) {
	engine := NewBaseEngine("test", "category")
	engine.SetDependencies([]string{"dep1"})

	ctx := context.Background()

	// Test without resolver
	err := engine.WaitForDependencies(ctx)
	if err == nil {
		t.Error("expected error when dependency resolver not set")
	}

	// Test with no dependencies
	engine.SetDependencies(nil)
	err = engine.WaitForDependencies(ctx)
	if err != nil {
		t.Errorf("WaitForDependencies failed: %v", err)
	}
}

func TestBaseEngineConcurrentAccess(t *testing.T) {
	engine := NewBaseEngine("test", "category")

	var wg sync.WaitGroup
	iterations := 100

	// Concurrent state changes
	wg.Add(1)
	go func() {
		defer wg.Done()
		for i := 0; i < iterations; i++ {
			engine.SetState(EngineStateRunning)
			engine.SetState(EngineStateIdle)
		}
	}()

	// Concurrent progress updates
	wg.Add(1)
	go func() {
		defer wg.Done()
		for i := 0; i < iterations; i++ {
			engine.SetProgress(i%100, "progress", i%5, 5)
		}
	}()

	// Concurrent output operations
	wg.Add(1)
	go func() {
		defer wg.Done()
		for i := 0; i < iterations; i++ {
			engine.SetOutput("key", i)
			engine.GetOutput("key")
		}
	}()

	// Concurrent reads
	wg.Add(1)
	go func() {
		defer wg.Done()
		for i := 0; i < iterations; i++ {
			engine.State()
			engine.Progress()
			engine.Outputs()
		}
	}()

	wg.Wait()
}
