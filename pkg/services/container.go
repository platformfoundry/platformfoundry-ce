// Package services provides a dependency injection container for Platform Foundry.
// This is inspired by Backstage's backend system and provides service registration,
// resolution, and lifecycle management.
package services

import (
	"context"
	"fmt"
	"reflect"
	"sync"
)

// Container is the main dependency injection container
type Container interface {
	// Registration
	Register(ref ServiceRef, factory ServiceFactory) error
	RegisterSingleton(ref ServiceRef, instance interface{}) error

	// Resolution
	Get(ctx context.Context, ref ServiceRef) (interface{}, error)
	GetAll(ctx context.Context, refs ...ServiceRef) ([]interface{}, error)
	MustGet(ctx context.Context, ref ServiceRef) interface{}

	// Inspection
	Has(ref ServiceRef) bool
	List() []ServiceRef

	// Lifecycle
	Start(ctx context.Context) error
	Stop(ctx context.Context) error

	// Scoping
	CreateScope() Container
}

// ServiceRef is a reference to a service
type ServiceRef struct {
	// ID is the unique identifier for this service
	ID string

	// Type is the Go type of the service (for type checking)
	Type reflect.Type
}

// NewServiceRef creates a new service reference
func NewServiceRef[T any](id string) ServiceRef {
	var t T
	return ServiceRef{
		ID:   id,
		Type: reflect.TypeOf(&t).Elem(),
	}
}

// ServiceFactory creates service instances
type ServiceFactory func(ctx context.Context, container Container) (interface{}, error)

// ServiceLifecycle defines optional lifecycle methods for services
type ServiceLifecycle interface {
	// Start is called when the container starts
	Start(ctx context.Context) error

	// Stop is called when the container stops
	Stop(ctx context.Context) error
}

// DefaultContainer is the default implementation of Container
type DefaultContainer struct {
	mu         sync.RWMutex
	factories  map[string]ServiceFactory
	singletons map[string]interface{}
	parent     *DefaultContainer
	started    bool
	instances  []ServiceLifecycle
}

// NewContainer creates a new container
func NewContainer() *DefaultContainer {
	return &DefaultContainer{
		factories:  make(map[string]ServiceFactory),
		singletons: make(map[string]interface{}),
	}
}

// Register registers a factory for a service
func (c *DefaultContainer) Register(ref ServiceRef, factory ServiceFactory) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if _, exists := c.factories[ref.ID]; exists {
		return fmt.Errorf("service %q is already registered", ref.ID)
	}

	c.factories[ref.ID] = factory
	return nil
}

// RegisterSingleton registers a singleton instance
func (c *DefaultContainer) RegisterSingleton(ref ServiceRef, instance interface{}) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if _, exists := c.singletons[ref.ID]; exists {
		return fmt.Errorf("service %q is already registered", ref.ID)
	}

	// Type check if type is specified
	if ref.Type != nil {
		instanceType := reflect.TypeOf(instance)
		if !instanceType.AssignableTo(ref.Type) {
			return fmt.Errorf("instance type %v is not assignable to service type %v", instanceType, ref.Type)
		}
	}

	c.singletons[ref.ID] = instance
	return nil
}

// Get resolves a service by reference
func (c *DefaultContainer) Get(ctx context.Context, ref ServiceRef) (interface{}, error) {
	c.mu.RLock()

	// Check singletons first
	if instance, ok := c.singletons[ref.ID]; ok {
		c.mu.RUnlock()
		return instance, nil
	}

	// Check factories
	factory, ok := c.factories[ref.ID]
	if !ok {
		// Check parent container
		if c.parent != nil {
			c.mu.RUnlock()
			return c.parent.Get(ctx, ref)
		}
		c.mu.RUnlock()
		return nil, &ServiceNotFoundError{ID: ref.ID}
	}

	c.mu.RUnlock()

	// Create instance from factory
	instance, err := factory(ctx, c)
	if err != nil {
		return nil, &ServiceCreationError{ID: ref.ID, Cause: err}
	}

	// Store as singleton
	c.mu.Lock()
	c.singletons[ref.ID] = instance
	c.mu.Unlock()

	return instance, nil
}

// GetAll resolves multiple services
func (c *DefaultContainer) GetAll(ctx context.Context, refs ...ServiceRef) ([]interface{}, error) {
	results := make([]interface{}, len(refs))
	for i, ref := range refs {
		instance, err := c.Get(ctx, ref)
		if err != nil {
			return nil, err
		}
		results[i] = instance
	}
	return results, nil
}

// MustGet resolves a service or panics.
// Deprecated: Use Get with proper error handling, or GetOrDefault for fallback behavior.
func (c *DefaultContainer) MustGet(ctx context.Context, ref ServiceRef) interface{} {
	instance, err := c.Get(ctx, ref)
	if err != nil {
		panic(err)
	}
	return instance
}

// GetOrDefault resolves a service or returns the default value if not found or on error
func (c *DefaultContainer) GetOrDefault(ctx context.Context, ref ServiceRef, defaultValue interface{}) interface{} {
	instance, err := c.Get(ctx, ref)
	if err != nil {
		return defaultValue
	}
	return instance
}

// Has checks if a service is registered
func (c *DefaultContainer) Has(ref ServiceRef) bool {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if _, ok := c.singletons[ref.ID]; ok {
		return true
	}
	if _, ok := c.factories[ref.ID]; ok {
		return true
	}
	if c.parent != nil {
		return c.parent.Has(ref)
	}
	return false
}

// List returns all registered service references
func (c *DefaultContainer) List() []ServiceRef {
	c.mu.RLock()
	defer c.mu.RUnlock()

	seen := make(map[string]bool)
	var refs []ServiceRef

	for id := range c.singletons {
		if !seen[id] {
			seen[id] = true
			refs = append(refs, ServiceRef{ID: id})
		}
	}

	for id := range c.factories {
		if !seen[id] {
			seen[id] = true
			refs = append(refs, ServiceRef{ID: id})
		}
	}

	return refs
}

// Start starts the container and all registered services
func (c *DefaultContainer) Start(ctx context.Context) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.started {
		return nil
	}

	// Start all singleton instances that implement ServiceLifecycle
	for _, instance := range c.singletons {
		if lifecycle, ok := instance.(ServiceLifecycle); ok {
			if err := lifecycle.Start(ctx); err != nil {
				return &ServiceStartError{Cause: err}
			}
			c.instances = append(c.instances, lifecycle)
		}
	}

	c.started = true
	return nil
}

// Stop stops the container and all registered services
func (c *DefaultContainer) Stop(ctx context.Context) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if !c.started {
		return nil
	}

	// Stop in reverse order
	var errs []error
	for i := len(c.instances) - 1; i >= 0; i-- {
		if err := c.instances[i].Stop(ctx); err != nil {
			errs = append(errs, err)
		}
	}

	c.started = false
	c.instances = nil

	if len(errs) > 0 {
		return &ServiceStopError{Errors: errs}
	}

	return nil
}

// CreateScope creates a child container that inherits from this one
func (c *DefaultContainer) CreateScope() Container {
	return &DefaultContainer{
		factories:  make(map[string]ServiceFactory),
		singletons: make(map[string]interface{}),
		parent:     c,
	}
}

// Service errors

// ServiceNotFoundError indicates a service was not found
type ServiceNotFoundError struct {
	ID string
}

func (e *ServiceNotFoundError) Error() string {
	return fmt.Sprintf("service %q not found", e.ID)
}

// ServiceCreationError indicates an error creating a service
type ServiceCreationError struct {
	ID    string
	Cause error
}

func (e *ServiceCreationError) Error() string {
	return fmt.Sprintf("failed to create service %q: %v", e.ID, e.Cause)
}

func (e *ServiceCreationError) Unwrap() error {
	return e.Cause
}

// ServiceStartError indicates an error starting a service
type ServiceStartError struct {
	Cause error
}

func (e *ServiceStartError) Error() string {
	return fmt.Sprintf("failed to start service: %v", e.Cause)
}

func (e *ServiceStartError) Unwrap() error {
	return e.Cause
}

// ServiceStopError indicates errors stopping services
type ServiceStopError struct {
	Errors []error
}

func (e *ServiceStopError) Error() string {
	return fmt.Sprintf("failed to stop %d services", len(e.Errors))
}
