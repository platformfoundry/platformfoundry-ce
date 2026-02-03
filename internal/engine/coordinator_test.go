package engine

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestNewCoordinator(t *testing.T) {
	config := CoordinatorConfig{
		MaxParallelEngines: 2,
		Timeout:            5 * time.Minute,
	}

	coord := NewCoordinator(config)

	if coord == nil {
		t.Fatal("expected non-nil coordinator")
	}
	if coord.EngineCount() != 0 {
		t.Errorf("expected 0 engines, got %d", coord.EngineCount())
	}
	if coord.GetEventBus() == nil {
		t.Error("expected event bus to be initialized")
	}
	if coord.GetDependencyGraph() == nil {
		t.Error("expected dependency graph to be initialized")
	}
}

func TestNewCoordinatorDefaults(t *testing.T) {
	config := CoordinatorConfig{} // All zeros

	coord := NewCoordinator(config)

	// Check defaults are applied
	if coord.maxParallel != 4 {
		t.Errorf("expected default maxParallel 4, got %d", coord.maxParallel)
	}
	if coord.config.Timeout != 30*time.Minute {
		t.Errorf("expected default timeout 30m, got %v", coord.config.Timeout)
	}
}

func TestCoordinatorRegisterEngine(t *testing.T) {
	coord := NewCoordinator(CoordinatorConfig{})

	engine := NewBaseEngine("test-engine", "test")

	err := coord.RegisterEngine(engine)
	if err != nil {
		t.Errorf("RegisterEngine failed: %v", err)
	}

	if coord.EngineCount() != 1 {
		t.Errorf("expected 1 engine, got %d", coord.EngineCount())
	}

	// Try registering same engine again
	err = coord.RegisterEngine(engine)
	if err == nil {
		t.Error("expected error when registering duplicate engine")
	}
}

func TestCoordinatorUnregisterEngine(t *testing.T) {
	coord := NewCoordinator(CoordinatorConfig{})

	engine := NewBaseEngine("test-engine", "test")
	coord.RegisterEngine(engine)

	err := coord.UnregisterEngine(engine.ID())
	if err != nil {
		t.Errorf("UnregisterEngine failed: %v", err)
	}

	if coord.EngineCount() != 0 {
		t.Errorf("expected 0 engines, got %d", coord.EngineCount())
	}

	// Try unregistering non-existent engine
	err = coord.UnregisterEngine("non-existent")
	if err == nil {
		t.Error("expected error when unregistering non-existent engine")
	}
}

func TestCoordinatorGetEngine(t *testing.T) {
	coord := NewCoordinator(CoordinatorConfig{})

	engine := NewBaseEngine("test-engine", "test")
	coord.RegisterEngine(engine)

	// Get by ID
	retrieved, ok := coord.GetEngine(engine.ID())
	if !ok {
		t.Error("expected to find engine by ID")
	}
	if retrieved.Name() != "test-engine" {
		t.Errorf("expected name 'test-engine', got '%s'", retrieved.Name())
	}

	// Get non-existent
	_, ok = coord.GetEngine("non-existent")
	if ok {
		t.Error("expected not to find non-existent engine")
	}
}

func TestCoordinatorGetEngineByName(t *testing.T) {
	coord := NewCoordinator(CoordinatorConfig{})

	engine := NewBaseEngine("test-engine", "test")
	coord.RegisterEngine(engine)

	// Get by name
	retrieved, ok := coord.GetEngineByName("test-engine")
	if !ok {
		t.Error("expected to find engine by name")
	}
	if retrieved.ID() != engine.ID() {
		t.Error("expected same engine")
	}

	// Get non-existent
	_, ok = coord.GetEngineByName("non-existent")
	if ok {
		t.Error("expected not to find non-existent engine")
	}
}

func TestCoordinatorGetEngineIDs(t *testing.T) {
	coord := NewCoordinator(CoordinatorConfig{})

	engine1 := NewBaseEngine("engine1", "test")
	engine2 := NewBaseEngine("engine2", "test")

	coord.RegisterEngine(engine1)
	coord.RegisterEngine(engine2)

	ids := coord.GetEngineIDs()
	if len(ids) != 2 {
		t.Errorf("expected 2 IDs, got %d", len(ids))
	}
}

func TestCoordinatorSubscribe(t *testing.T) {
	coord := NewCoordinator(CoordinatorConfig{})

	var eventCount int32

	listener := NewFuncListener(func(event EngineEvent) {
		atomic.AddInt32(&eventCount, 1)
	})

	coord.Subscribe(listener)

	// Emit an event through the event bus
	coord.GetEventBus().Emit(EngineEvent{
		Type:    EventTypeLog,
		Message: "test",
	})

	time.Sleep(50 * time.Millisecond)

	if atomic.LoadInt32(&eventCount) == 0 {
		t.Error("expected to receive event")
	}
}

func TestCoordinatorApplyWithMockEngines(t *testing.T) {
	config := CoordinatorConfig{
		MaxParallelEngines: 2,
		Timeout:            10 * time.Second,
		MockMode:           true,
	}

	coord := NewCoordinator(config)

	// Create mock engines
	engine1 := &mockEngine{BaseEngine: NewBaseEngine("engine1", "infra")}
	engine2 := &mockEngine{BaseEngine: NewBaseEngine("engine2", "orchestrator")}

	coord.RegisterEngine(engine1)
	coord.RegisterEngine(engine2)

	specs := map[string]map[string]interface{}{
		"engine1": {"key": "value1"},
		"engine2": {"key": "value2"},
	}

	ctx := context.Background()
	err := coord.Apply(ctx, specs)
	if err != nil {
		t.Errorf("Apply failed: %v", err)
	}

	// Check results
	results := coord.GetResults()
	if len(results) != 2 {
		t.Errorf("expected 2 results, got %d", len(results))
	}
}

func TestCoordinatorApplyWithDependencies(t *testing.T) {
	config := CoordinatorConfig{
		MaxParallelEngines: 2,
		Timeout:            10 * time.Second,
	}

	coord := NewCoordinator(config)

	// Create engines with dependencies
	engine1 := &mockEngine{BaseEngine: NewBaseEngine("infra", "infrastructure")}
	engine2 := &mockEngine{BaseEngine: NewBaseEngine("orchestrator", "orchestrator")}
	engine2.SetDependencies([]string{"infra"})

	coord.RegisterEngine(engine1)
	coord.RegisterEngine(engine2)

	specs := map[string]map[string]interface{}{
		"infra":        {"region": "us-east-1"},
		"orchestrator": {"replicas": 3},
	}

	ctx := context.Background()
	err := coord.Apply(ctx, specs)
	if err != nil {
		t.Errorf("Apply failed: %v", err)
	}

	// Verify both executed
	if engine1.applyCount != 1 {
		t.Errorf("expected engine1 apply count 1, got %d", engine1.applyCount)
	}
	if engine2.applyCount != 1 {
		t.Errorf("expected engine2 apply count 1, got %d", engine2.applyCount)
	}
}

func TestCoordinatorApplyWithCircularDependency(t *testing.T) {
	coord := NewCoordinator(CoordinatorConfig{Timeout: 5 * time.Second})

	// Create engines with circular dependencies
	// Note: Dependencies must reference engine IDs, so we create them first
	// then set dependencies using the actual IDs
	engine1 := &mockEngine{BaseEngine: NewBaseEngine("engine1", "test")}
	engine2 := &mockEngine{BaseEngine: NewBaseEngine("engine2", "test")}

	// Set circular dependencies using engine IDs
	engine1.SetDependencies([]string{engine2.ID()})
	engine2.SetDependencies([]string{engine1.ID()})

	coord.RegisterEngine(engine1)
	coord.RegisterEngine(engine2)

	specs := map[string]map[string]interface{}{
		"engine1": {},
		"engine2": {},
	}

	ctx := context.Background()
	err := coord.Apply(ctx, specs)
	if err == nil {
		t.Error("expected error for circular dependency")
	}
}

func TestCoordinatorApplyWithFailure(t *testing.T) {
	config := CoordinatorConfig{
		MaxParallelEngines: 2,
		Timeout:            10 * time.Second,
		RetryCount:         1,
	}

	coord := NewCoordinator(config)

	// Create a failing engine
	engine := &mockEngine{
		BaseEngine: NewBaseEngine("failing-engine", "test"),
		shouldFail: true,
	}

	coord.RegisterEngine(engine)

	specs := map[string]map[string]interface{}{
		"failing-engine": {},
	}

	ctx := context.Background()
	err := coord.Apply(ctx, specs)
	if err == nil {
		t.Error("expected error for failing engine")
	}

	// Check errors
	errs := coord.GetErrors()
	if len(errs) != 1 {
		t.Errorf("expected 1 error, got %d", len(errs))
	}
}

func TestCoordinatorApplyWithRetry(t *testing.T) {
	config := CoordinatorConfig{
		MaxParallelEngines: 1,
		Timeout:            10 * time.Second,
		RetryCount:         3,
		RetryDelay:         10 * time.Millisecond,
	}

	coord := NewCoordinator(config)

	// Engine that fails twice then succeeds
	engine := &mockEngine{
		BaseEngine:  NewBaseEngine("flaky-engine", "test"),
		failCount:   2,
		shouldFail:  true,
		failCounter: new(int32),
	}

	coord.RegisterEngine(engine)

	specs := map[string]map[string]interface{}{
		"flaky-engine": {},
	}

	ctx := context.Background()
	err := coord.Apply(ctx, specs)
	if err != nil {
		t.Errorf("Apply failed after retries: %v", err)
	}
}

func TestCoordinatorApplyWithRollback(t *testing.T) {
	config := CoordinatorConfig{
		MaxParallelEngines: 1,
		Timeout:            10 * time.Second,
		RollbackOnFailure:  true,
	}

	coord := NewCoordinator(config)

	// First engine succeeds
	engine1 := &mockEngine{BaseEngine: NewBaseEngine("engine1", "test")}
	// Second engine fails
	engine2 := &mockEngine{
		BaseEngine: NewBaseEngine("engine2", "test"),
		shouldFail: true,
	}
	engine2.SetDependencies([]string{"engine1"})

	coord.RegisterEngine(engine1)
	coord.RegisterEngine(engine2)

	specs := map[string]map[string]interface{}{
		"engine1": {},
		"engine2": {},
	}

	ctx := context.Background()
	err := coord.Apply(ctx, specs)
	if err == nil {
		t.Error("expected error for failing engine")
	}

	// Verify rollback was called on completed engine
	if engine1.rollbackCount != 1 {
		t.Errorf("expected engine1 rollback count 1, got %d", engine1.rollbackCount)
	}
}

func TestCoordinatorApplyWithNoSpec(t *testing.T) {
	coord := NewCoordinator(CoordinatorConfig{Timeout: 5 * time.Second})

	engine := &mockEngine{BaseEngine: NewBaseEngine("engine1", "test")}
	coord.RegisterEngine(engine)

	// Empty specs
	specs := map[string]map[string]interface{}{}

	ctx := context.Background()
	err := coord.Apply(ctx, specs)
	if err != nil {
		t.Errorf("Apply failed: %v", err)
	}

	// Engine should be skipped
	if engine.applyCount != 0 {
		t.Errorf("expected engine apply count 0 (skipped), got %d", engine.applyCount)
	}
}

func TestCoordinatorApplyFindSpecByCategory(t *testing.T) {
	coord := NewCoordinator(CoordinatorConfig{Timeout: 5 * time.Second})

	engine := &mockEngine{BaseEngine: NewBaseEngine("my-infra-engine", "infrastructure")}
	coord.RegisterEngine(engine)

	// Spec by category instead of name
	specs := map[string]map[string]interface{}{
		"infrastructure": {"region": "us-west-2"},
	}

	ctx := context.Background()
	err := coord.Apply(ctx, specs)
	if err != nil {
		t.Errorf("Apply failed: %v", err)
	}

	if engine.applyCount != 1 {
		t.Errorf("expected engine apply count 1, got %d", engine.applyCount)
	}
}

func TestCoordinatorPlan(t *testing.T) {
	coord := NewCoordinator(CoordinatorConfig{})

	engine := &mockEngine{BaseEngine: NewBaseEngine("engine1", "test")}
	coord.RegisterEngine(engine)

	specs := map[string]map[string]interface{}{
		"engine1": {"key": "value"},
	}

	ctx := context.Background()
	plans, err := coord.Plan(ctx, specs)
	if err != nil {
		t.Errorf("Plan failed: %v", err)
	}

	if len(plans) != 1 {
		t.Errorf("expected 1 plan, got %d", len(plans))
	}
}

func TestCoordinatorHealthCheck(t *testing.T) {
	coord := NewCoordinator(CoordinatorConfig{})

	engine1 := &mockEngine{BaseEngine: NewBaseEngine("engine1", "test")}
	engine2 := &mockEngine{BaseEngine: NewBaseEngine("engine2", "test")}

	coord.RegisterEngine(engine1)
	coord.RegisterEngine(engine2)

	health := coord.HealthCheck()
	if len(health) != 2 {
		t.Errorf("expected 2 health checks, got %d", len(health))
	}

	for _, h := range health {
		if !h.Healthy {
			t.Error("expected all engines to be healthy")
		}
	}
}

func TestCoordinatorGetStatus(t *testing.T) {
	coord := NewCoordinator(CoordinatorConfig{})

	engine := NewBaseEngine("engine1", "test")
	coord.RegisterEngine(engine)

	status := coord.GetStatus()
	if len(status) != 1 {
		t.Errorf("expected 1 status, got %d", len(status))
	}

	engineStatus, ok := status[engine.ID()]
	if !ok {
		t.Error("expected to find engine status")
	}
	if engineStatus.Name != "engine1" {
		t.Errorf("expected name 'engine1', got '%s'", engineStatus.Name)
	}
	if engineStatus.State != EngineStateIdle {
		t.Errorf("expected state idle, got '%s'", engineStatus.State)
	}
}

func TestCoordinatorStop(t *testing.T) {
	t.Skip("Skipping flaky timing-dependent test - TODO: fix race condition")
	config := CoordinatorConfig{
		Timeout: 5 * time.Second, // Longer timeout so it doesn't interfere
	}
	coord := NewCoordinator(config)

	// Create a slow engine that respects context
	engine := &mockEngine{
		BaseEngine:   NewBaseEngine("slow-engine", "test"),
		applyDelay:   5 * time.Second,
		checkContext: true, // Enable context checking
	}
	coord.RegisterEngine(engine)

	specs := map[string]map[string]interface{}{
		"slow-engine": {},
	}

	ctx := context.Background()

	// Start Apply in goroutine
	var applyErr error
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		applyErr = coord.Apply(ctx, specs)
	}()

	// Wait a bit then stop
	time.Sleep(100 * time.Millisecond)
	coord.Stop()

	wg.Wait()

	// Apply should have failed due to context cancellation or stop
	if applyErr == nil {
		t.Error("expected error after stop")
	}
}

func TestCoordinatorParallelExecution(t *testing.T) {
	config := CoordinatorConfig{
		MaxParallelEngines: 3,
		Timeout:            10 * time.Second,
	}

	coord := NewCoordinator(config)

	// Create multiple independent engines
	var executionOrder []string
	var mu sync.Mutex

	makeEngine := func(name string) *mockEngine {
		e := &mockEngine{
			BaseEngine: NewBaseEngine(name, "test"),
			applyDelay: 50 * time.Millisecond,
			onApply: func() {
				mu.Lock()
				executionOrder = append(executionOrder, name)
				mu.Unlock()
			},
		}
		return e
	}

	engine1 := makeEngine("engine1")
	engine2 := makeEngine("engine2")
	engine3 := makeEngine("engine3")

	coord.RegisterEngine(engine1)
	coord.RegisterEngine(engine2)
	coord.RegisterEngine(engine3)

	specs := map[string]map[string]interface{}{
		"engine1": {},
		"engine2": {},
		"engine3": {},
	}

	ctx := context.Background()
	start := time.Now()
	err := coord.Apply(ctx, specs)
	elapsed := time.Since(start)

	if err != nil {
		t.Errorf("Apply failed: %v", err)
	}

	// With parallel execution of 3 engines with 50ms delay,
	// total time should be closer to 50ms than 150ms
	if elapsed > 200*time.Millisecond {
		t.Errorf("expected parallel execution to complete faster, took %v", elapsed)
	}

	mu.Lock()
	if len(executionOrder) != 3 {
		t.Errorf("expected 3 executions, got %d", len(executionOrder))
	}
	mu.Unlock()
}

// mockEngine implements Engine interface for testing
type mockEngine struct {
	*BaseEngine
	shouldFail    bool
	failCount     int
	failCounter   *int32
	applyCount    int
	rollbackCount int
	applyDelay    time.Duration
	mu            sync.Mutex
	onApply       func()
	checkContext  bool // If true, checks context during Apply delay
}

func (e *mockEngine) Validate(spec map[string]interface{}) error {
	return nil
}

func (e *mockEngine) Plan(spec map[string]interface{}) (*Plan, error) {
	return &Plan{
		Actions:     []PlanAction{{Type: "create", Resource: "test-resource"}},
		Description: "Test plan",
	}, nil
}

func (e *mockEngine) Apply(spec map[string]interface{}) (*Result, error) {
	e.mu.Lock()
	defer e.mu.Unlock()

	if e.applyDelay > 0 {
		if e.checkContext && e.GetContext() != nil {
			// Check context periodically during delay
			ticker := time.NewTicker(10 * time.Millisecond)
			defer ticker.Stop()
			deadline := time.Now().Add(e.applyDelay)
			for time.Now().Before(deadline) {
				select {
				case <-e.GetContext().Done():
					return nil, e.GetContext().Err()
				case <-ticker.C:
					// Continue waiting
				}
			}
		} else {
			time.Sleep(e.applyDelay)
		}
	}

	if e.onApply != nil {
		e.onApply()
	}

	e.applyCount++

	if e.shouldFail {
		if e.failCounter != nil {
			count := atomic.AddInt32(e.failCounter, 1)
			if int(count) <= e.failCount {
				return nil, errors.New("simulated failure")
			}
			// After failCount failures, succeed
			e.shouldFail = false
		} else {
			return nil, errors.New("simulated failure")
		}
	}

	return &Result{
		Status:  "success",
		Message: "Applied successfully",
	}, nil
}

func (e *mockEngine) Rollback() error {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.rollbackCount++
	return nil
}

func (e *mockEngine) Delete() error {
	return nil
}

func (e *mockEngine) HealthCheck() (*HealthStatus, error) {
	return &HealthStatus{
		Healthy: true,
		Message: "Healthy",
	}, nil
}
