package service

import (
	"errors"
	"testing"
	"time"

	"github.com/platformfoundry/pf-ce/internal/state"
	"github.com/platformfoundry/pf-ce/pkg/types"
)

var ErrResourceNotFound = errors.New("resource not found")

// MockBackend is a mock implementation of state.Backend for testing
type MockBackend struct {
	resources map[string]*state.Resource
}

func NewMockBackend() *MockBackend {
	return &MockBackend{
		resources: make(map[string]*state.Resource),
	}
}

func (m *MockBackend) Save(resource *state.Resource) error {
	m.resources[resource.Name] = resource
	return nil
}

func (m *MockBackend) Get(name string) (*state.Resource, error) {
	resource, ok := m.resources[name]
	if !ok {
		return nil, ErrResourceNotFound
	}
	return resource, nil
}

func (m *MockBackend) List() ([]*state.Resource, error) {
	resources := make([]*state.Resource, 0, len(m.resources))
	for _, r := range m.resources {
		resources = append(resources, r)
	}
	return resources, nil
}

func (m *MockBackend) Delete(name string) error {
	delete(m.resources, name)
	return nil
}

func (m *MockBackend) Lock(name string) error {
	return nil
}

func (m *MockBackend) Unlock(name string) error {
	return nil
}

func (m *MockBackend) GetVersion(name string, version int) (*state.Resource, error) {
	return m.Get(name)
}

func (m *MockBackend) ListVersions(name string) ([]*state.ResourceVersion, error) {
	return nil, nil
}

func (m *MockBackend) Close() error {
	return nil
}

func TestManager_Create(t *testing.T) {
	backend := NewMockBackend()
	manager := NewManager(backend)

	service := &types.Service{
		APIVersion: "platformfoundry.io/v1",
		Kind:       "Service",
		Metadata: types.Metadata{
			Name:         "user-api",
			Organization: "acme-corp",
		},
		Spec: types.ServiceSpec{
			Type: types.ServiceTypeMicroservice,
			Owner: types.ServiceOwner{
				Team: "platform-team",
			},
		},
	}

	err := manager.Create(service)
	if err != nil {
		t.Fatalf("Create() failed: %v", err)
	}

	// Verify service was added to registry
	retrieved := manager.registry.Get("user-api", "acme-corp")
	if retrieved == nil {
		t.Fatal("Service not found in registry")
	}

	if retrieved.Metadata.Name != "user-api" {
		t.Errorf("Expected name 'user-api', got '%s'", retrieved.Metadata.Name)
	}
}

func TestManager_CreateDuplicate(t *testing.T) {
	backend := NewMockBackend()
	manager := NewManager(backend)

	service := &types.Service{
		APIVersion: "platformfoundry.io/v1",
		Kind:       "Service",
		Metadata: types.Metadata{
			Name:         "user-api",
			Organization: "acme-corp",
		},
		Spec: types.ServiceSpec{
			Type: types.ServiceTypeMicroservice,
			Owner: types.ServiceOwner{
				Team: "platform-team",
			},
		},
	}

	// Create once
	err := manager.Create(service)
	if err != nil {
		t.Fatalf("First Create() failed: %v", err)
	}

	// Try to create again
	err = manager.Create(service)
	if err == nil {
		t.Fatal("Expected error when creating duplicate service, got nil")
	}
}

func TestManager_Get(t *testing.T) {
	backend := NewMockBackend()
	manager := NewManager(backend)

	service := &types.Service{
		APIVersion: "platformfoundry.io/v1",
		Kind:       "Service",
		Metadata: types.Metadata{
			Name:         "user-api",
			Organization: "acme-corp",
		},
		Spec: types.ServiceSpec{
			Type: types.ServiceTypeMicroservice,
			Owner: types.ServiceOwner{
				Team: "platform-team",
			},
		},
	}

	// Create service
	err := manager.Create(service)
	if err != nil {
		t.Fatalf("Create() failed: %v", err)
	}

	// Get service
	retrieved, err := manager.Get("user-api", "acme-corp")
	if err != nil {
		t.Fatalf("Get() failed: %v", err)
	}

	if retrieved.Metadata.Name != "user-api" {
		t.Errorf("Expected name 'user-api', got '%s'", retrieved.Metadata.Name)
	}
}

func TestManager_Update(t *testing.T) {
	backend := NewMockBackend()
	manager := NewManager(backend)

	service := &types.Service{
		APIVersion: "platformfoundry.io/v1",
		Kind:       "Service",
		Metadata: types.Metadata{
			Name:         "user-api",
			Organization: "acme-corp",
		},
		Spec: types.ServiceSpec{
			Type: types.ServiceTypeMicroservice,
			Owner: types.ServiceOwner{
				Team: "platform-team",
			},
		},
	}

	// Create service
	err := manager.Create(service)
	if err != nil {
		t.Fatalf("Create() failed: %v", err)
	}

	// Update service
	service.Spec.Owner.Team = "new-team"
	err = manager.Update(service)
	if err != nil {
		t.Fatalf("Update() failed: %v", err)
	}

	// Verify update
	retrieved, err := manager.Get("user-api", "acme-corp")
	if err != nil {
		t.Fatalf("Get() failed: %v", err)
	}

	if retrieved.Spec.Owner.Team != "new-team" {
		t.Errorf("Expected team 'new-team', got '%s'", retrieved.Spec.Owner.Team)
	}
}

func TestManager_Delete(t *testing.T) {
	backend := NewMockBackend()
	manager := NewManager(backend)

	service := &types.Service{
		APIVersion: "platformfoundry.io/v1",
		Kind:       "Service",
		Metadata: types.Metadata{
			Name:         "user-api",
			Organization: "acme-corp",
		},
		Spec: types.ServiceSpec{
			Type: types.ServiceTypeMicroservice,
			Owner: types.ServiceOwner{
				Team: "platform-team",
			},
		},
	}

	// Create service
	err := manager.Create(service)
	if err != nil {
		t.Fatalf("Create() failed: %v", err)
	}

	// Delete service
	err = manager.Delete("user-api", "acme-corp")
	if err != nil {
		t.Fatalf("Delete() failed: %v", err)
	}

	// Verify deletion
	retrieved := manager.registry.Get("user-api", "acme-corp")
	if retrieved != nil {
		t.Error("Service still exists in registry after deletion")
	}
}

func TestManager_List(t *testing.T) {
	backend := NewMockBackend()
	manager := NewManager(backend)

	// Create multiple services
	services := []*types.Service{
		{
			APIVersion: "platformfoundry.io/v1",
			Kind:       "Service",
			Metadata: types.Metadata{
				Name:         "user-api",
				Organization: "acme-corp",
			},
			Spec: types.ServiceSpec{
				Type: types.ServiceTypeMicroservice,
				Owner: types.ServiceOwner{
					Team: "platform-team",
				},
			},
		},
		{
			APIVersion: "platformfoundry.io/v1",
			Kind:       "Service",
			Metadata: types.Metadata{
				Name:         "auth-api",
				Organization: "acme-corp",
			},
			Spec: types.ServiceSpec{
				Type: types.ServiceTypeAPI,
				Owner: types.ServiceOwner{
					Team: "platform-team",
				},
			},
		},
	}

	for _, svc := range services {
		err := manager.Create(svc)
		if err != nil {
			t.Fatalf("Create() failed: %v", err)
		}
	}

	// List services
	listed, err := manager.List("acme-corp")
	if err != nil {
		t.Fatalf("List() failed: %v", err)
	}

	if len(listed) != 2 {
		t.Errorf("Expected 2 services, got %d", len(listed))
	}
}

func TestManager_ListByTeam(t *testing.T) {
	backend := NewMockBackend()
	manager := NewManager(backend)

	// Create services for different teams
	services := []*types.Service{
		{
			APIVersion: "platformfoundry.io/v1",
			Kind:       "Service",
			Metadata: types.Metadata{
				Name:         "user-api",
				Organization: "acme-corp",
			},
			Spec: types.ServiceSpec{
				Type: types.ServiceTypeMicroservice,
				Owner: types.ServiceOwner{
					Team: "team-a",
				},
			},
		},
		{
			APIVersion: "platformfoundry.io/v1",
			Kind:       "Service",
			Metadata: types.Metadata{
				Name:         "auth-api",
				Organization: "acme-corp",
			},
			Spec: types.ServiceSpec{
				Type: types.ServiceTypeAPI,
				Owner: types.ServiceOwner{
					Team: "team-b",
				},
			},
		},
	}

	for _, svc := range services {
		err := manager.Create(svc)
		if err != nil {
			t.Fatalf("Create() failed: %v", err)
		}
	}

	// List services by team
	teamAServices, err := manager.ListByTeam("team-a", "acme-corp")
	if err != nil {
		t.Fatalf("ListByTeam() failed: %v", err)
	}

	if len(teamAServices) != 1 {
		t.Errorf("Expected 1 service for team-a, got %d", len(teamAServices))
	}

	if teamAServices[0].Metadata.Name != "user-api" {
		t.Errorf("Expected service 'user-api', got '%s'", teamAServices[0].Metadata.Name)
	}
}

func TestManager_UpdateStatus(t *testing.T) {
	backend := NewMockBackend()
	manager := NewManager(backend)

	service := &types.Service{
		APIVersion: "platformfoundry.io/v1",
		Kind:       "Service",
		Metadata: types.Metadata{
			Name:         "user-api",
			Organization: "acme-corp",
		},
		Spec: types.ServiceSpec{
			Type: types.ServiceTypeMicroservice,
			Owner: types.ServiceOwner{
				Team: "platform-team",
			},
		},
		Status: types.ServiceStatus{
			State:  types.ServiceStateDraft,
			Health: types.ServiceHealthUnknown,
		},
	}

	// Create service
	err := manager.Create(service)
	if err != nil {
		t.Fatalf("Create() failed: %v", err)
	}

	// Update status
	now := time.Now()
	newStatus := types.ServiceStatus{
		State:        types.ServiceStateActive,
		Health:       types.ServiceHealthHealthy,
		LastDeployed: &now,
		Version:      "v1.0.0",
	}

	err = manager.UpdateStatus("user-api", "acme-corp", newStatus)
	if err != nil {
		t.Fatalf("UpdateStatus() failed: %v", err)
	}

	// Verify status update
	retrieved, err := manager.Get("user-api", "acme-corp")
	if err != nil {
		t.Fatalf("Get() failed: %v", err)
	}

	if retrieved.Status.State != types.ServiceStateActive {
		t.Errorf("Expected state 'active', got '%s'", retrieved.Status.State)
	}

	if retrieved.Status.Health != types.ServiceHealthHealthy {
		t.Errorf("Expected health 'healthy', got '%s'", retrieved.Status.Health)
	}

	if retrieved.Status.Version != "v1.0.0" {
		t.Errorf("Expected version 'v1.0.0', got '%s'", retrieved.Status.Version)
	}
}

func TestManager_GetDependencies(t *testing.T) {
	backend := NewMockBackend()
	manager := NewManager(backend)

	// Create database service
	database := &types.Service{
		APIVersion: "platformfoundry.io/v1",
		Kind:       "Service",
		Metadata: types.Metadata{
			Name:         "postgres-db",
			Organization: "acme-corp",
		},
		Spec: types.ServiceSpec{
			Type: types.ServiceTypeDatabase,
			Owner: types.ServiceOwner{
				Team: "platform-team",
			},
		},
	}

	err := manager.Create(database)
	if err != nil {
		t.Fatalf("Create() failed: %v", err)
	}

	// Create service with dependency
	service := &types.Service{
		APIVersion: "platformfoundry.io/v1",
		Kind:       "Service",
		Metadata: types.Metadata{
			Name:         "user-api",
			Organization: "acme-corp",
		},
		Spec: types.ServiceSpec{
			Type: types.ServiceTypeMicroservice,
			Owner: types.ServiceOwner{
				Team: "platform-team",
			},
			Dependencies: []types.ServiceDependency{
				{
					Name:     "postgres-db",
					Type:     "database",
					Required: true,
				},
			},
		},
	}

	err = manager.Create(service)
	if err != nil {
		t.Fatalf("Create() failed: %v", err)
	}

	// Get dependencies
	dependencies, err := manager.GetDependencies("user-api", "acme-corp")
	if err != nil {
		t.Fatalf("GetDependencies() failed: %v", err)
	}

	if len(dependencies) != 1 {
		t.Errorf("Expected 1 dependency, got %d", len(dependencies))
	}

	if dependencies[0].Metadata.Name != "postgres-db" {
		t.Errorf("Expected dependency 'postgres-db', got '%s'", dependencies[0].Metadata.Name)
	}
}
