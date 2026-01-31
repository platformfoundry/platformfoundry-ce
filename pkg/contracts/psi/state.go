// Package psi defines the Platform State Interface (PSI).
// This is inspired by Terraform backends and handles state persistence.
package psi

import (
	"context"
	"time"
)

// StateBackend is the main interface for state persistence.
// Implementations handle storage and retrieval of platform state.
type StateBackend interface {
	// State Operations
	GetState(ctx context.Context, workspace string) (*State, error)
	PutState(ctx context.Context, workspace string, state *State) error
	DeleteState(ctx context.Context, workspace string) error

	// Locking prevents concurrent modifications
	Lock(ctx context.Context, workspace string, info *LockInfo) (string, error)
	Unlock(ctx context.Context, workspace string, lockID string) error

	// Versioning provides state history
	GetStateVersions(ctx context.Context, workspace string) ([]StateVersion, error)
	GetStateAtVersion(ctx context.Context, workspace string, version int64) (*State, error)

	// Workspaces manages multiple state workspaces
	ListWorkspaces(ctx context.Context) ([]string, error)
	CreateWorkspace(ctx context.Context, workspace string) error
	DeleteWorkspace(ctx context.Context, workspace string) error

	// Close releases backend resources
	Close() error
}

// State represents the complete state of a platform
type State struct {
	// Version is the state format version
	Version int64

	// Serial is incremented on each state change
	Serial int64

	// Lineage is a unique identifier for this state lineage
	Lineage string

	// Resources contains all managed resource states
	Resources []ResourceState

	// Outputs contains output values
	Outputs map[string]OutputValue

	// Metadata contains additional state metadata
	Metadata StateMetadata
}

// ResourceState represents the state of a single resource
type ResourceState struct {
	// Type is the resource type (e.g., "platform", "infrastructure")
	Type string

	// Name is the resource name within its type
	Name string

	// Provider is the provider that manages this resource
	Provider string

	// Instances contains the state of each instance
	Instances []InstanceState
}

// InstanceState represents the state of a resource instance
type InstanceState struct {
	// IndexKey is the index key for list/map resources
	IndexKey interface{}

	// Status is the current status of this instance
	Status string

	// Attributes contains the attribute values
	Attributes map[string]interface{}

	// Private contains provider-internal data
	Private []byte

	// Dependencies lists dependent resources
	Dependencies []string

	// CreateBeforeDestroy indicates replacement ordering
	CreateBeforeDestroy bool
}

// OutputValue represents a state output value
type OutputValue struct {
	// Value is the output value
	Value interface{}

	// Sensitive indicates if this output is sensitive
	Sensitive bool

	// Type describes the value type
	Type string
}

// StateMetadata contains metadata about the state
type StateMetadata struct {
	// CreatedAt is when this state was created
	CreatedAt time.Time

	// UpdatedAt is when this state was last updated
	UpdatedAt time.Time

	// LastApplied is when the last apply completed
	LastApplied time.Time

	// LastAppliedBy is who performed the last apply
	LastAppliedBy string

	// Environment contains environment information
	Environment string

	// Tags contains arbitrary tags
	Tags map[string]string
}
