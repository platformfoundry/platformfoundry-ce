// Package services provides a dependency injection container for Platform Foundry.
package services

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// LifecycleManager manages service lifecycle
type LifecycleManager struct {
	mu       sync.Mutex
	services []managedService
	started  bool
	stopped  bool
}

type managedService struct {
	name      string
	lifecycle ServiceLifecycle
	priority  int
}

// NewLifecycleManager creates a new lifecycle manager
func NewLifecycleManager() *LifecycleManager {
	return &LifecycleManager{}
}

// Register registers a service with lifecycle management
func (m *LifecycleManager) Register(name string, lifecycle ServiceLifecycle, priority int) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.services = append(m.services, managedService{
		name:      name,
		lifecycle: lifecycle,
		priority:  priority,
	})
}

// Start starts all registered services in priority order
func (m *LifecycleManager) Start(ctx context.Context) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.started {
		return nil
	}

	// Sort by priority (lower = first)
	services := m.sortByPriority(false)

	for _, svc := range services {
		if err := m.startService(ctx, svc); err != nil {
			// Stop already started services
			m.stopStartedServices(ctx, services)
			return err
		}
	}

	m.started = true
	return nil
}

// Stop stops all registered services in reverse priority order
func (m *LifecycleManager) Stop(ctx context.Context) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if !m.started || m.stopped {
		return nil
	}

	// Sort by priority (higher = first for stopping)
	services := m.sortByPriority(true)

	var errs []error
	for _, svc := range services {
		if err := m.stopService(ctx, svc); err != nil {
			errs = append(errs, err)
		}
	}

	m.stopped = true

	if len(errs) > 0 {
		return &LifecycleError{Phase: "stop", Errors: errs}
	}

	return nil
}

func (m *LifecycleManager) startService(ctx context.Context, svc managedService) error {
	// Add timeout for service startup
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	if err := svc.lifecycle.Start(ctx); err != nil {
		return &LifecycleError{
			Phase:   "start",
			Service: svc.name,
			Errors:  []error{err},
		}
	}

	return nil
}

func (m *LifecycleManager) stopService(ctx context.Context, svc managedService) error {
	// Add timeout for service shutdown
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	if err := svc.lifecycle.Stop(ctx); err != nil {
		return &LifecycleError{
			Phase:   "stop",
			Service: svc.name,
			Errors:  []error{err},
		}
	}

	return nil
}

func (m *LifecycleManager) stopStartedServices(ctx context.Context, services []managedService) {
	// Stop in reverse order
	for i := len(services) - 1; i >= 0; i-- {
		_ = m.stopService(ctx, services[i])
	}
}

func (m *LifecycleManager) sortByPriority(reverse bool) []managedService {
	result := make([]managedService, len(m.services))
	copy(result, m.services)

	// Simple insertion sort
	for i := 1; i < len(result); i++ {
		j := i
		for j > 0 {
			shouldSwap := result[j].priority < result[j-1].priority
			if reverse {
				shouldSwap = result[j].priority > result[j-1].priority
			}
			if shouldSwap {
				result[j], result[j-1] = result[j-1], result[j]
				j--
			} else {
				break
			}
		}
	}

	return result
}

// LifecycleError represents a lifecycle error
type LifecycleError struct {
	Phase   string
	Service string
	Errors  []error
}

func (e *LifecycleError) Error() string {
	if e.Service != "" {
		return fmt.Sprintf("lifecycle %s error for service %q: %v", e.Phase, e.Service, e.Errors[0])
	}
	return fmt.Sprintf("lifecycle %s error: %d errors", e.Phase, len(e.Errors))
}

// ServicePriorities defines standard service priorities
const (
	PriorityFirst   = 0    // Start first, stop last
	PriorityCore    = 100  // Core infrastructure services
	PriorityDefault = 500  // Default priority
	PriorityPlugin  = 800  // Plugin services
	PriorityLast    = 1000 // Start last, stop first
)

// HealthCheck represents a health check for a service
type HealthCheck interface {
	// Check performs a health check
	Check(ctx context.Context) HealthStatus
}

// HealthStatus represents the health status of a service
type HealthStatus struct {
	// Healthy indicates if the service is healthy
	Healthy bool

	// Message provides details about the health status
	Message string

	// Details contains additional health details
	Details map[string]interface{}

	// Checks contains results of individual checks
	Checks []CheckResult
}

// CheckResult represents the result of an individual health check
type CheckResult struct {
	// Name is the check name
	Name string

	// Healthy indicates if the check passed
	Healthy bool

	// Message provides details
	Message string

	// Duration is how long the check took
	Duration time.Duration
}

// HealthAggregator aggregates health checks from multiple services
type HealthAggregator struct {
	mu     sync.RWMutex
	checks map[string]HealthCheck
}

// NewHealthAggregator creates a new health aggregator
func NewHealthAggregator() *HealthAggregator {
	return &HealthAggregator{
		checks: make(map[string]HealthCheck),
	}
}

// Register registers a health check
func (a *HealthAggregator) Register(name string, check HealthCheck) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.checks[name] = check
}

// Check performs all registered health checks
func (a *HealthAggregator) Check(ctx context.Context) HealthStatus {
	a.mu.RLock()
	defer a.mu.RUnlock()

	status := HealthStatus{
		Healthy: true,
		Details: make(map[string]interface{}),
	}

	for name, check := range a.checks {
		start := time.Now()
		result := check.Check(ctx)
		duration := time.Since(start)

		status.Checks = append(status.Checks, CheckResult{
			Name:     name,
			Healthy:  result.Healthy,
			Message:  result.Message,
			Duration: duration,
		})

		if !result.Healthy {
			status.Healthy = false
		}

		status.Details[name] = result
	}

	if status.Healthy {
		status.Message = "All checks passed"
	} else {
		status.Message = "One or more checks failed"
	}

	return status
}

// ReadinessCheck checks if the service is ready to receive traffic
type ReadinessCheck interface {
	// Ready checks if the service is ready
	Ready(ctx context.Context) bool
}

// LivenessCheck checks if the service is alive
type LivenessCheck interface {
	// Alive checks if the service is alive
	Alive(ctx context.Context) bool
}
