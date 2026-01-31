package engine

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// CoordinatorConfig configures the coordinator
type CoordinatorConfig struct {
	MaxParallelEngines int
	Timeout            time.Duration
	MockMode           bool
	MockConfig         *MockConfig
	RollbackOnFailure  bool
	RetryCount         int
	RetryDelay         time.Duration
}

// Coordinator manages multiple engines and their execution
type Coordinator struct {
	engines   map[string]Engine
	enginesMu sync.RWMutex

	depGraph    *DependencyGraph
	depResolver *DependencyGraphResolver
	eventBus    *EventBus

	// Execution control
	maxParallel int
	semaphore   chan struct{}

	// State
	ctx    context.Context
	cancel context.CancelFunc

	// Results
	results     map[string]*Result
	resultsMu   sync.RWMutex
	errors      map[string]error
	errorsMu    sync.RWMutex

	// Configuration
	config CoordinatorConfig

	// Rollback tracking
	completedEngines []string
	completedMu      sync.Mutex
}

// NewCoordinator creates a new coordinator
func NewCoordinator(config CoordinatorConfig) *Coordinator {
	if config.MaxParallelEngines == 0 {
		config.MaxParallelEngines = 4
	}
	if config.Timeout == 0 {
		config.Timeout = 30 * time.Minute
	}

	depGraph := NewDependencyGraph()
	return &Coordinator{
		engines:          make(map[string]Engine),
		depGraph:         depGraph,
		depResolver:      NewDependencyGraphResolver(depGraph),
		eventBus:         NewEventBus(),
		maxParallel:      config.MaxParallelEngines,
		semaphore:        make(chan struct{}, config.MaxParallelEngines),
		results:          make(map[string]*Result),
		errors:           make(map[string]error),
		config:           config,
		completedEngines: make([]string, 0),
	}
}

// RegisterEngine adds an engine to the coordinator
func (c *Coordinator) RegisterEngine(engine Engine) error {
	c.enginesMu.Lock()
	defer c.enginesMu.Unlock()

	if _, exists := c.engines[engine.ID()]; exists {
		return fmt.Errorf("engine %s already registered", engine.ID())
	}

	c.engines[engine.ID()] = engine
	c.depGraph.AddNode(engine.ID(), engine.Name(), engine.DependsOn())
	engine.Subscribe(c.eventBus)
	engine.SetDependencyResolver(c.depResolver)

	return nil
}

// UnregisterEngine removes an engine from the coordinator
func (c *Coordinator) UnregisterEngine(engineID string) error {
	c.enginesMu.Lock()
	defer c.enginesMu.Unlock()

	if _, exists := c.engines[engineID]; !exists {
		return fmt.Errorf("engine %s not found", engineID)
	}

	delete(c.engines, engineID)
	c.depGraph.RemoveNode(engineID)

	return nil
}

// GetEngine returns an engine by ID
func (c *Coordinator) GetEngine(engineID string) (Engine, bool) {
	c.enginesMu.RLock()
	defer c.enginesMu.RUnlock()
	engine, ok := c.engines[engineID]
	return engine, ok
}

// GetEngineByName returns an engine by name
func (c *Coordinator) GetEngineByName(name string) (Engine, bool) {
	c.enginesMu.RLock()
	defer c.enginesMu.RUnlock()

	for _, engine := range c.engines {
		if engine.Name() == name {
			return engine, true
		}
	}
	return nil, false
}

// Subscribe adds an event listener to the coordinator
func (c *Coordinator) Subscribe(listener EventListener) {
	c.eventBus.Subscribe(listener)
}

// Apply executes all registered engines respecting dependencies
func (c *Coordinator) Apply(ctx context.Context, specs map[string]map[string]interface{}) error {
	c.ctx, c.cancel = context.WithTimeout(ctx, c.config.Timeout)
	defer c.cancel()

	// Reset state
	c.depResolver.Reset()
	c.results = make(map[string]*Result)
	c.errors = make(map[string]error)
	c.completedEngines = make([]string, 0)

	// Get parallel execution levels
	levels, err := c.depGraph.GetParallelExecutionLevels()
	if err != nil {
		return fmt.Errorf("dependency resolution failed: %w", err)
	}

	c.eventBus.EmitCoordinatorEvent("execution_started",
		fmt.Sprintf("Starting execution with %d levels", len(levels)),
		map[string]interface{}{
			"total_engines": len(c.engines),
			"levels":        len(levels),
			"mock_mode":     c.config.MockMode,
		})

	// Execute level by level
	for levelNum, level := range levels {
		c.eventBus.EmitCoordinatorEvent("level_started",
			fmt.Sprintf("Starting level %d with %d engines", levelNum, len(level)),
			map[string]interface{}{
				"level":   levelNum,
				"engines": level,
			})

		if err := c.executeLevel(levelNum, level, specs); err != nil {
			// Rollback completed engines on failure
			if c.config.RollbackOnFailure {
				c.rollbackCompleted()
			}
			return fmt.Errorf("level %d failed: %w", levelNum, err)
		}

		c.eventBus.EmitCoordinatorEvent("level_completed",
			fmt.Sprintf("Completed level %d", levelNum),
			map[string]interface{}{"level": levelNum})
	}

	c.eventBus.EmitCoordinatorEvent("execution_completed",
		"All engines completed successfully",
		map[string]interface{}{
			"total_engines": len(c.engines),
			"results":       len(c.results),
		})

	return nil
}

// executeLevel runs all engines in a level concurrently
func (c *Coordinator) executeLevel(levelNum int, engineIDs []string, specs map[string]map[string]interface{}) error {
	var wg sync.WaitGroup
	errChan := make(chan error, len(engineIDs))

	for _, engineID := range engineIDs {
		engine, ok := c.engines[engineID]
		if !ok {
			continue
		}

		// Find spec by engine name or category
		spec := c.findSpec(engine, specs)
		if spec == nil {
			c.eventBus.EmitCoordinatorEvent("engine_skipped",
				fmt.Sprintf("No spec found for engine %s, skipping", engine.Name()),
				map[string]interface{}{"engine": engine.Name()})
			continue
		}

		wg.Add(1)
		go func(eng Engine, s map[string]interface{}) {
			defer wg.Done()

			// Acquire semaphore slot
			c.semaphore <- struct{}{}
			defer func() { <-c.semaphore }()

			err := c.executeEngine(eng, s)
			if err != nil {
				errChan <- fmt.Errorf("engine %s failed: %w", eng.Name(), err)
			}
		}(engine, spec)
	}

	// Wait for all engines in level
	wg.Wait()
	close(errChan)

	// Collect errors
	var errs []error
	for err := range errChan {
		errs = append(errs, err)
	}

	if len(errs) > 0 {
		return fmt.Errorf("%d engines failed: %v", len(errs), errs)
	}

	return nil
}

// findSpec finds the specification for an engine
func (c *Coordinator) findSpec(engine Engine, specs map[string]map[string]interface{}) map[string]interface{} {
	// Try by name first
	if spec, ok := specs[engine.Name()]; ok {
		return spec
	}
	// Try by category
	if spec, ok := specs[engine.Category()]; ok {
		return spec
	}
	return nil
}

// executeEngine runs a single engine with retry logic
func (c *Coordinator) executeEngine(engine Engine, spec map[string]interface{}) error {
	// Set mock mode if enabled
	if c.config.MockMode {
		if mockable, ok := engine.(MockableEngine); ok {
			mockable.SetMockMode(true, c.config.MockConfig)
		}
	}

	var lastErr error
	retries := c.config.RetryCount
	if retries == 0 {
		retries = 1 // At least one attempt
	}

	for attempt := 0; attempt < retries; attempt++ {
		if attempt > 0 {
			c.eventBus.EmitCoordinatorEvent("engine_retry",
				fmt.Sprintf("Retrying engine %s (attempt %d/%d)", engine.Name(), attempt+1, retries),
				map[string]interface{}{
					"engine":  engine.Name(),
					"attempt": attempt + 1,
				})
			time.Sleep(c.config.RetryDelay)
		}

		err := c.runEngine(engine, spec)
		if err == nil {
			return nil
		}
		lastErr = err
	}

	// Store error
	c.errorsMu.Lock()
	c.errors[engine.ID()] = lastErr
	c.errorsMu.Unlock()

	return lastErr
}

// runEngine executes a single engine
func (c *Coordinator) runEngine(engine Engine, spec map[string]interface{}) error {
	// Initialize
	config := EngineConfig{
		Name:       engine.Name(),
		Category:   engine.Category(),
		MockMode:   c.config.MockMode,
		MockConfig: c.config.MockConfig,
	}
	if err := engine.Initialize(c.ctx, config); err != nil {
		return fmt.Errorf("initialization failed: %w", err)
	}

	// Start engine
	if err := engine.Start(c.ctx); err != nil {
		return fmt.Errorf("start failed: %w", err)
	}

	// Execute
	result, err := engine.Apply(spec)
	if err != nil {
		return fmt.Errorf("apply failed: %w", err)
	}

	// Store result
	c.resultsMu.Lock()
	c.results[engine.ID()] = result
	c.resultsMu.Unlock()

	// Mark completed in dependency resolver
	c.depResolver.MarkCompleted(engine.ID(), engine.Outputs())

	// Track for potential rollback
	c.completedMu.Lock()
	c.completedEngines = append(c.completedEngines, engine.ID())
	c.completedMu.Unlock()

	return nil
}

// rollbackCompleted rolls back all completed engines in reverse order
func (c *Coordinator) rollbackCompleted() {
	c.completedMu.Lock()
	engines := make([]string, len(c.completedEngines))
	copy(engines, c.completedEngines)
	c.completedMu.Unlock()

	c.eventBus.EmitCoordinatorEvent("rollback_started",
		fmt.Sprintf("Rolling back %d completed engines", len(engines)),
		map[string]interface{}{"count": len(engines)})

	// Rollback in reverse order
	for i := len(engines) - 1; i >= 0; i-- {
		engineID := engines[i]
		engine, ok := c.engines[engineID]
		if !ok {
			continue
		}

		c.eventBus.EmitCoordinatorEvent("engine_rollback_started",
			fmt.Sprintf("Rolling back engine %s", engine.Name()),
			map[string]interface{}{"engine": engine.Name()})

		if err := engine.Rollback(); err != nil {
			c.eventBus.EmitCoordinatorEvent("rollback_error",
				fmt.Sprintf("Failed to rollback %s: %v", engine.Name(), err),
				map[string]interface{}{
					"engine": engine.Name(),
					"error":  err.Error(),
				})
		}
	}

	c.eventBus.EmitCoordinatorEvent("rollback_completed",
		"Rollback completed",
		nil)
}

// GetStatus returns status of all engines
func (c *Coordinator) GetStatus() map[string]EngineStatus {
	c.enginesMu.RLock()
	defer c.enginesMu.RUnlock()

	status := make(map[string]EngineStatus)
	for id, engine := range c.engines {
		progress := engine.Progress()

		var duration time.Duration
		var startedAt *time.Time
		if be, ok := engine.(*BaseEngine); ok {
			startedAt = be.GetStartedAt()
			duration = be.GetDuration()
		}

		c.errorsMu.RLock()
		engineErr := c.errors[id]
		c.errorsMu.RUnlock()

		status[id] = EngineStatus{
			ID:        id,
			Name:      engine.Name(),
			Category:  engine.Category(),
			State:     engine.State(),
			Progress:  progress.Percentage,
			Message:   progress.Message,
			StartedAt: startedAt,
			Duration:  duration,
			Error:     engineErr,
		}
	}
	return status
}

// GetResults returns all engine results
func (c *Coordinator) GetResults() map[string]*Result {
	c.resultsMu.RLock()
	defer c.resultsMu.RUnlock()

	results := make(map[string]*Result)
	for id, result := range c.results {
		results[id] = result
	}
	return results
}

// GetErrors returns all engine errors
func (c *Coordinator) GetErrors() map[string]error {
	c.errorsMu.RLock()
	defer c.errorsMu.RUnlock()

	errors := make(map[string]error)
	for id, err := range c.errors {
		errors[id] = err
	}
	return errors
}

// GetEventBus returns the event bus
func (c *Coordinator) GetEventBus() *EventBus {
	return c.eventBus
}

// GetDependencyGraph returns the dependency graph
func (c *Coordinator) GetDependencyGraph() *DependencyGraph {
	return c.depGraph
}

// Stop cancels all running engines
func (c *Coordinator) Stop() {
	if c.cancel != nil {
		c.cancel()
	}

	c.enginesMu.RLock()
	defer c.enginesMu.RUnlock()

	for _, engine := range c.engines {
		engine.Stop()
	}
}

// Plan generates execution plans for all engines
func (c *Coordinator) Plan(ctx context.Context, specs map[string]map[string]interface{}) (map[string]*Plan, error) {
	plans := make(map[string]*Plan)

	c.enginesMu.RLock()
	defer c.enginesMu.RUnlock()

	for _, engine := range c.engines {
		spec := c.findSpec(engine, specs)
		if spec == nil {
			continue
		}

		plan, err := engine.Plan(spec)
		if err != nil {
			return nil, fmt.Errorf("planning failed for %s: %w", engine.Name(), err)
		}
		plans[engine.ID()] = plan
	}

	return plans, nil
}

// HealthCheck performs health checks on all engines
func (c *Coordinator) HealthCheck() map[string]*HealthStatus {
	c.enginesMu.RLock()
	defer c.enginesMu.RUnlock()

	status := make(map[string]*HealthStatus)
	for id, engine := range c.engines {
		health, err := engine.HealthCheck()
		if err != nil {
			status[id] = &HealthStatus{
				Healthy: false,
				Message: err.Error(),
			}
		} else {
			status[id] = health
		}
	}
	return status
}

// EngineCount returns the number of registered engines
func (c *Coordinator) EngineCount() int {
	c.enginesMu.RLock()
	defer c.enginesMu.RUnlock()
	return len(c.engines)
}

// GetEngineIDs returns all engine IDs
func (c *Coordinator) GetEngineIDs() []string {
	c.enginesMu.RLock()
	defer c.enginesMu.RUnlock()

	ids := make([]string, 0, len(c.engines))
	for id := range c.engines {
		ids = append(ids, id)
	}
	return ids
}
