// Package drivers provides a pluggable resource driver system for custom resource types.
package drivers

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// ProvisionResult represents the result of a provisioning operation
type ProvisionResult struct {
	ID        string                 `json:"id"`
	Status    string                 `json:"status"`
	Outputs   map[string]interface{} `json:"outputs,omitempty"`
	Message   string                 `json:"message,omitempty"`
	CreatedAt time.Time              `json:"createdAt"`
}

// ResourceStatus represents the current status of a resource
type ResourceStatus struct {
	ID        string                 `json:"id"`
	Status    string                 `json:"status"` // pending, provisioning, ready, failed, deleting
	Message   string                 `json:"message,omitempty"`
	Health    string                 `json:"health,omitempty"` // healthy, degraded, unhealthy
	Outputs   map[string]interface{} `json:"outputs,omitempty"`
	UpdatedAt time.Time              `json:"updatedAt"`
}

// Driver defines the interface for resource drivers
type Driver interface {
	// Name returns the unique name of the driver
	Name() string

	// ResourceType returns the type of resource this driver handles
	ResourceType() string

	// Provision creates a new resource
	Provision(ctx context.Context, spec map[string]interface{}) (*ProvisionResult, error)

	// Update modifies an existing resource
	Update(ctx context.Context, id string, spec map[string]interface{}) error

	// Delete removes a resource
	Delete(ctx context.Context, id string) error

	// GetStatus returns the current status of a resource
	GetStatus(ctx context.Context, id string) (*ResourceStatus, error)

	// GetOutputs returns the outputs of a provisioned resource
	GetOutputs(ctx context.Context, id string) (map[string]interface{}, error)

	// Validate validates a resource spec before provisioning
	Validate(ctx context.Context, spec map[string]interface{}) error
}

// DriverConfig contains common configuration for drivers
type DriverConfig struct {
	Name        string                 `json:"name"`
	Type        string                 `json:"type"`
	Endpoint    string                 `json:"endpoint,omitempty"`
	Credentials map[string]string      `json:"credentials,omitempty"`
	Options     map[string]interface{} `json:"options,omitempty"`
}

// Registry manages registered drivers
type Registry struct {
	drivers map[string]Driver
	mu      sync.RWMutex
}

// NewRegistry creates a new driver registry
func NewRegistry() *Registry {
	return &Registry{
		drivers: make(map[string]Driver),
	}
}

// Register registers a driver
func (r *Registry) Register(driver Driver) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	name := driver.Name()
	if _, exists := r.drivers[name]; exists {
		return fmt.Errorf("driver %s already registered", name)
	}

	r.drivers[name] = driver
	return nil
}

// Unregister removes a driver
func (r *Registry) Unregister(name string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.drivers, name)
}

// Get returns a driver by name
func (r *Registry) Get(name string) (Driver, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	driver, ok := r.drivers[name]
	if !ok {
		return nil, fmt.Errorf("driver %s not found", name)
	}

	return driver, nil
}

// GetByResourceType returns a driver that handles the given resource type
func (r *Registry) GetByResourceType(resourceType string) (Driver, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	for _, driver := range r.drivers {
		if driver.ResourceType() == resourceType {
			return driver, nil
		}
	}

	return nil, fmt.Errorf("no driver found for resource type %s", resourceType)
}

// List returns all registered drivers
func (r *Registry) List() []Driver {
	r.mu.RLock()
	defer r.mu.RUnlock()

	drivers := make([]Driver, 0, len(r.drivers))
	for _, d := range r.drivers {
		drivers = append(drivers, d)
	}

	return drivers
}

// ListNames returns names of all registered drivers
func (r *Registry) ListNames() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	names := make([]string, 0, len(r.drivers))
	for name := range r.drivers {
		names = append(names, name)
	}

	return names
}

// Has checks if a driver is registered
func (r *Registry) Has(name string) bool {
	r.mu.RLock()
	defer r.mu.RUnlock()
	_, ok := r.drivers[name]
	return ok
}

// Provision provisions a resource using the appropriate driver
func (r *Registry) Provision(ctx context.Context, driverName string, spec map[string]interface{}) (*ProvisionResult, error) {
	driver, err := r.Get(driverName)
	if err != nil {
		return nil, err
	}

	if err := driver.Validate(ctx, spec); err != nil {
		return nil, fmt.Errorf("validation failed: %w", err)
	}

	return driver.Provision(ctx, spec)
}

// Update updates a resource using the appropriate driver
func (r *Registry) Update(ctx context.Context, driverName, resourceID string, spec map[string]interface{}) error {
	driver, err := r.Get(driverName)
	if err != nil {
		return err
	}

	return driver.Update(ctx, resourceID, spec)
}

// Delete deletes a resource using the appropriate driver
func (r *Registry) Delete(ctx context.Context, driverName, resourceID string) error {
	driver, err := r.Get(driverName)
	if err != nil {
		return err
	}

	return driver.Delete(ctx, resourceID)
}

// GetStatus gets the status of a resource using the appropriate driver
func (r *Registry) GetStatus(ctx context.Context, driverName, resourceID string) (*ResourceStatus, error) {
	driver, err := r.Get(driverName)
	if err != nil {
		return nil, err
	}

	return driver.GetStatus(ctx, resourceID)
}
