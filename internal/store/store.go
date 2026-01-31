package store

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/platformfoundry/platformfoundry-ce/internal/state"
	"github.com/platformfoundry/platformfoundry-ce/pkg/types"
)

// ResourceState represents the state of a provisioned resource
// Kept for backwards compatibility
type ResourceState struct {
	ID         uint      `json:"id"`
	Name       string    `json:"name"`
	Kind       string    `json:"kind"`
	Provider   string    `json:"provider"`
	Spec       string    `json:"spec"` // JSON-encoded spec
	Status     string    `json:"status"`
	CreatedAt  time.Time `json:"createdAt"`
	UpdatedAt  time.Time `json:"updatedAt"`
}

// Store manages resource state persistence
// Implements US-3.4: Update Store to Use Backends
type Store struct {
	backend state.Backend
}

// New creates a new Store instance with default local backend
func New() (*Store, error) {
	// Create state directory
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("failed to get home directory: %w", err)
	}

	stateDir := filepath.Join(homeDir, ".platformfoundry")
	if err := os.MkdirAll(stateDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create state directory: %w", err)
	}

	dbPath := filepath.Join(stateDir, "state.db")

	// Create local backend
	backend, err := state.NewLocalBackend(dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to create backend: %w", err)
	}

	return &Store{backend: backend}, nil
}

// NewWithBackend creates a new Store with a custom backend
func NewWithBackend(backend state.Backend) *Store {
	return &Store{backend: backend}
}

// NewWithPath creates a new Store with a custom database path
// Useful for testing or when you need isolation between store instances
func NewWithPath(dbPath string) (*Store, error) {
	// Ensure parent directory exists
	dir := filepath.Dir(dbPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create state directory: %w", err)
	}

	// Create local backend
	backend, err := state.NewLocalBackend(dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to create backend: %w", err)
	}

	return &Store{backend: backend}, nil
}

// NewFromConfig creates a new Store from configuration
func NewFromConfig(config *state.BackendConfig) (*Store, error) {
	var backend state.Backend
	var err error

	switch config.Type {
	case "local", "":
		// Local backend is default
		dbPath := "./platformfoundry.db"
		if path, ok := config.Config["path"].(string); ok {
			dbPath = path
		}
		backend, err = state.NewLocalBackend(dbPath)
	case "s3":
		// S3 backend
		s3Config := &state.S3Config{}
		if bucket, ok := config.Config["bucket"].(string); ok {
			s3Config.Bucket = bucket
		}
		if region, ok := config.Config["region"].(string); ok {
			s3Config.Region = region
		}
		if table, ok := config.Config["tableName"].(string); ok {
			s3Config.TableName = table
		}
		backend, err = state.NewS3Backend(s3Config)
	default:
		return nil, fmt.Errorf("unknown backend type: %s", config.Type)
	}

	if err != nil {
		return nil, fmt.Errorf("failed to create backend: %w", err)
	}

	return &Store{backend: backend}, nil
}

// Save saves or updates a resource state
func (s *Store) Save(name, kind, provider string, spec map[string]interface{}, status string) error {
	// Create resource with provider in spec
	specWithProvider := make(map[string]interface{})
	for k, v := range spec {
		specWithProvider[k] = v
	}
	specWithProvider["provider"] = provider

	statusMap := map[string]interface{}{
		"phase": status,
	}

	resource := &state.Resource{
		Name:       name,
		Kind:       kind,
		APIVersion: "platformfoundry.io/v1",
		Spec:       specWithProvider,
		Status:     statusMap,
	}

	return s.backend.Save(resource)
}

// Get retrieves a resource state by name
func (s *Store) Get(name string) (*ResourceState, error) {
	resource, err := s.backend.Get(name)
	if err != nil {
		return nil, err
	}

	return s.resourceToState(resource)
}

// List retrieves all resource states
func (s *Store) List() ([]ResourceState, error) {
	resources, err := s.backend.List()
	if err != nil {
		return nil, err
	}

	states := make([]ResourceState, len(resources))
	for i, resource := range resources {
		state, err := s.resourceToState(resource)
		if err != nil {
			return nil, err
		}
		states[i] = *state
	}

	return states, nil
}

// ListByKind retrieves all resources of a specific kind
func (s *Store) ListByKind(kind string) ([]ResourceState, error) {
	resources, err := s.backend.List()
	if err != nil {
		return nil, err
	}

	states := make([]ResourceState, 0)
	for _, resource := range resources {
		if resource.Kind == kind {
			state, err := s.resourceToState(resource)
			if err != nil {
				return nil, err
			}
			states = append(states, *state)
		}
	}

	return states, nil
}

// Delete deletes a resource state by name
func (s *Store) Delete(name string) error {
	return s.backend.Delete(name)
}

// Exists checks if a resource exists
func (s *Store) Exists(name string) (bool, error) {
	_, err := s.backend.Get(name)
	if err != nil {
		return false, nil
	}
	return true, nil
}

// Lock acquires a lock on a resource
func (s *Store) Lock(name string) error {
	return s.backend.Lock(name)
}

// Unlock releases a lock on a resource
func (s *Store) Unlock(name string) error {
	return s.backend.Unlock(name)
}

// Close closes the store backend
func (s *Store) Close() error {
	return s.backend.Close()
}

// GetBackend returns the underlying backend (for org-filtered wrappers)
func (s *Store) GetBackend() state.Backend {
	return s.backend
}

// GetEnvironment retrieves an environment resource by name
func (s *Store) GetEnvironment(ctx context.Context, name string) (*types.Environment, error) {
	resource, err := s.backend.Get(name)
	if err != nil {
		return nil, fmt.Errorf("environment not found: %w", err)
	}

	if resource.Kind != "Environment" {
		return nil, fmt.Errorf("resource '%s' is not an Environment (found: %s)", name, resource.Kind)
	}

	// Convert resource to Environment type
	// Note: state.Resource has flat Name, but types.Environment has Metadata.Name
	env := &types.Environment{
		APIVersion: resource.APIVersion,
		Kind:       resource.Kind,
		Metadata: types.Metadata{
			Name: resource.Name,
		},
	}

	// Convert spec to EnvironmentSpec
	if resource.Spec != nil {
		specJSON, err := json.Marshal(resource.Spec)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal spec: %w", err)
		}
		if err := json.Unmarshal(specJSON, &env.Spec); err != nil {
			return nil, fmt.Errorf("failed to unmarshal spec: %w", err)
		}
	}

	// Convert status to EnvironmentStatus
	if resource.Status != nil {
		statusJSON, err := json.Marshal(resource.Status)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal status: %w", err)
		}
		if err := json.Unmarshal(statusJSON, &env.Status); err != nil {
			return nil, fmt.Errorf("failed to unmarshal status: %w", err)
		}
	}

	return env, nil
}

// SaveEnvironment saves an environment resource
func (s *Store) SaveEnvironment(ctx context.Context, env *types.Environment) error {
	// Convert to state.Resource
	envJSON, err := json.Marshal(env)
	if err != nil {
		return fmt.Errorf("failed to marshal environment: %w", err)
	}

	var resourceMap map[string]interface{}
	if err := json.Unmarshal(envJSON, &resourceMap); err != nil {
		return fmt.Errorf("failed to unmarshal to map: %w", err)
	}

	spec, _ := resourceMap["spec"].(map[string]interface{})
	status, _ := resourceMap["status"].(map[string]interface{})

	resource := &state.Resource{
		Name:       env.Metadata.Name,
		Kind:       "Environment",
		APIVersion: env.APIVersion,
		Spec:       spec,
		Status:     status,
	}

	return s.backend.Save(resource)
}

// resourceToState converts a state.Resource to ResourceState
func (s *Store) resourceToState(resource *state.Resource) (*ResourceState, error) {
	specJSON, err := json.Marshal(resource.Spec)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal spec: %w", err)
	}

	provider := ""
	if p, ok := resource.Spec["provider"].(string); ok {
		provider = p
	}

	status := "pending"
	if phase, ok := resource.Status["phase"].(string); ok {
		status = phase
	}

	return &ResourceState{
		Name:      resource.Name,
		Kind:      resource.Kind,
		Provider:  provider,
		Spec:      string(specJSON),
		Status:    status,
		CreatedAt: resource.CreatedAt,
		UpdatedAt: resource.UpdatedAt,
	}, nil
}
