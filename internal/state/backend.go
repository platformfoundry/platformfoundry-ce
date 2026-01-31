package state

import (
	"context"
	"time"
)

// Backend defines the interface for state storage backends
// Implements US-3.1: State Backend Interface
type Backend interface {
	// Save stores a resource in the backend
	Save(resource *Resource) error

	// Get retrieves a resource by name
	Get(name string) (*Resource, error)

	// List returns all resources
	List() ([]*Resource, error)

	// Delete removes a resource by name
	Delete(name string) error

	// Lock acquires a lock on a resource
	Lock(name string) error

	// Unlock releases a lock on a resource
	Unlock(name string) error

	// GetVersion retrieves a specific version of a resource
	GetVersion(name string, version int) (*Resource, error)

	// ListVersions returns all versions of a resource
	ListVersions(name string) ([]*ResourceVersion, error)

	// Close closes the backend connection
	Close() error
}

// ContextBackend extends Backend with context-aware methods
// This interface allows for proper request cancellation and timeout handling
type ContextBackend interface {
	Backend

	// SaveWithContext stores a resource with context for cancellation/timeout
	SaveWithContext(ctx context.Context, resource *Resource) error

	// GetWithContext retrieves a resource by name with context
	GetWithContext(ctx context.Context, name string) (*Resource, error)

	// ListWithContext returns all resources with context
	ListWithContext(ctx context.Context) ([]*Resource, error)

	// DeleteWithContext removes a resource by name with context
	DeleteWithContext(ctx context.Context, name string) error

	// LockWithContext acquires a lock on a resource with context
	LockWithContext(ctx context.Context, name string) error

	// UnlockWithContext releases a lock on a resource with context
	UnlockWithContext(ctx context.Context, name string) error

	// GetVersionWithContext retrieves a specific version with context
	GetVersionWithContext(ctx context.Context, name string, version int) (*Resource, error)

	// ListVersionsWithContext returns all versions with context
	ListVersionsWithContext(ctx context.Context, name string) ([]*ResourceVersion, error)
}

// Resource represents a stored resource
type Resource struct {
	Name       string                 `json:"name"`
	Kind       string                 `json:"kind"`
	APIVersion string                 `json:"apiVersion"`
	Spec       map[string]interface{} `json:"spec"`
	Status     map[string]interface{} `json:"status,omitempty"`
	Version    int                    `json:"version"`
	CreatedAt  time.Time              `json:"createdAt"`
	UpdatedAt  time.Time              `json:"updatedAt"`
}

// ResourceVersion represents a versioned resource
type ResourceVersion struct {
	Version   int                    `json:"version"`
	Spec      map[string]interface{} `json:"spec"`
	Status    map[string]interface{} `json:"status,omitempty"`
	CreatedAt time.Time              `json:"createdAt"`
}

// Lock represents a resource lock
type Lock struct {
	ResourceName string    `json:"resourceName"`
	Owner        string    `json:"owner"`
	AcquiredAt   time.Time `json:"acquiredAt"`
	ExpiresAt    time.Time `json:"expiresAt"`
}

// BackendConfig represents backend configuration
type BackendConfig struct {
	Type   string                 `yaml:"type" json:"type"` // local, s3
	Config map[string]interface{} `yaml:"config,omitempty" json:"config,omitempty"`
}
