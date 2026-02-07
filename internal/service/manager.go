package service

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/platformfoundry/pf-ce/internal/state"
	"github.com/platformfoundry/pf-ce/pkg/types"
)

// Manager handles CRUD operations for services
type Manager struct {
	backend  state.Backend
	registry *Registry
}

// NewManager creates a new service manager
func NewManager(backend state.Backend) *Manager {
	return &Manager{
		backend:  backend,
		registry: NewRegistry(),
	}
}

// Create creates a new service
func (m *Manager) Create(service *types.Service) error {
	// Validate service
	if err := service.Validate(); err != nil {
		return fmt.Errorf("invalid service: %w", err)
	}

	// Check if service already exists
	existing, err := m.Get(service.Metadata.Name, service.Metadata.Organization)
	if err == nil && existing != nil {
		return fmt.Errorf("service %s already exists in organization %s",
			service.Metadata.Name, service.Metadata.Organization)
	}

	// Set timestamps
	now := time.Now()
	if service.Status.LastDeployed == nil {
		service.Status.LastDeployed = &now
	}

	// Convert service to state resource
	resource, err := m.serviceToResource(service)
	if err != nil {
		return fmt.Errorf("failed to convert service to resource: %w", err)
	}

	// Save to backend
	if err := m.backend.Save(resource); err != nil {
		return fmt.Errorf("failed to save service: %w", err)
	}

	// Add to registry
	m.registry.Add(service)

	return nil
}

// Update updates an existing service
func (m *Manager) Update(service *types.Service) error {
	// Validate service
	if err := service.Validate(); err != nil {
		return fmt.Errorf("invalid service: %w", err)
	}

	// Check if service exists
	existing, err := m.Get(service.Metadata.Name, service.Metadata.Organization)
	if err != nil || existing == nil {
		return fmt.Errorf("service %s not found in organization %s",
			service.Metadata.Name, service.Metadata.Organization)
	}

	// Update timestamp
	now := time.Now()
	if service.Status.LastDeployed == nil {
		service.Status.LastDeployed = &now
	}

	// Convert service to state resource
	resource, err := m.serviceToResource(service)
	if err != nil {
		return fmt.Errorf("failed to convert service to resource: %w", err)
	}

	// Save to backend
	if err := m.backend.Save(resource); err != nil {
		return fmt.Errorf("failed to update service: %w", err)
	}

	// Update registry
	m.registry.Update(service)

	return nil
}

// Get retrieves a service by name and organization
func (m *Manager) Get(name, organization string) (*types.Service, error) {
	// Try registry first (fast path)
	if service := m.registry.Get(name, organization); service != nil {
		return service, nil
	}

	// Fallback to backend
	resourceName := m.buildResourceName(name, organization)
	resource, err := m.backend.Get(resourceName)
	if err != nil {
		return nil, fmt.Errorf("service %s not found: %w", name, err)
	}

	// Convert resource to service
	service, err := m.resourceToService(resource)
	if err != nil {
		return nil, fmt.Errorf("failed to convert resource to service: %w", err)
	}

	// Add to registry for future queries
	m.registry.Add(service)

	return service, nil
}

// List returns all services, optionally filtered by organization
func (m *Manager) List(organization string) ([]*types.Service, error) {
	// Try registry first
	services := m.registry.List(organization)
	if len(services) > 0 {
		return services, nil
	}

	// Fallback to backend
	resources, err := m.backend.List()
	if err != nil {
		return nil, fmt.Errorf("failed to list services: %w", err)
	}

	services = make([]*types.Service, 0)
	for _, resource := range resources {
		// Filter by kind
		if resource.Kind != "Service" {
			continue
		}

		// Filter by organization if specified
		if organization != "" {
			orgValue, ok := resource.Spec["organization"]
			if !ok || orgValue != organization {
				continue
			}
		}

		service, err := m.resourceToService(resource)
		if err != nil {
			// Log error but continue
			continue
		}

		services = append(services, service)
		m.registry.Add(service) // Cache for future queries
	}

	return services, nil
}

// Delete deletes a service
func (m *Manager) Delete(name, organization string) error {
	// Check if service exists
	existing, err := m.Get(name, organization)
	if err != nil || existing == nil {
		return fmt.Errorf("service %s not found in organization %s", name, organization)
	}

	// Delete from backend
	resourceName := m.buildResourceName(name, organization)
	if err := m.backend.Delete(resourceName); err != nil {
		return fmt.Errorf("failed to delete service: %w", err)
	}

	// Remove from registry
	m.registry.Delete(name, organization)

	return nil
}

// ListByTeam returns services owned by a specific team
func (m *Manager) ListByTeam(team, organization string) ([]*types.Service, error) {
	allServices, err := m.List(organization)
	if err != nil {
		return nil, err
	}

	teamServices := make([]*types.Service, 0)
	for _, service := range allServices {
		if service.Spec.Owner.Team == team {
			teamServices = append(teamServices, service)
		}
	}

	return teamServices, nil
}

// ListByType returns services of a specific type
func (m *Manager) ListByType(serviceType types.ServiceType, organization string) ([]*types.Service, error) {
	allServices, err := m.List(organization)
	if err != nil {
		return nil, err
	}

	filteredServices := make([]*types.Service, 0)
	for _, service := range allServices {
		if service.Spec.Type == serviceType {
			filteredServices = append(filteredServices, service)
		}
	}

	return filteredServices, nil
}

// UpdateStatus updates only the status of a service
func (m *Manager) UpdateStatus(name, organization string, status types.ServiceStatus) error {
	service, err := m.Get(name, organization)
	if err != nil {
		return err
	}

	service.Status = status
	return m.Update(service)
}

// GetDependencies returns all dependencies of a service
func (m *Manager) GetDependencies(name, organization string) ([]*types.Service, error) {
	service, err := m.Get(name, organization)
	if err != nil {
		return nil, err
	}

	dependencies := make([]*types.Service, 0)
	for _, dep := range service.Spec.Dependencies {
		depService, err := m.Get(dep.Name, organization)
		if err != nil {
			// Dependency might not exist yet
			continue
		}
		dependencies = append(dependencies, depService)
	}

	return dependencies, nil
}

// GetDependents returns all services that depend on the given service
func (m *Manager) GetDependents(name, organization string) ([]*types.Service, error) {
	allServices, err := m.List(organization)
	if err != nil {
		return nil, err
	}

	dependents := make([]*types.Service, 0)
	for _, service := range allServices {
		for _, dep := range service.Spec.Dependencies {
			if dep.Name == name {
				dependents = append(dependents, service)
				break
			}
		}
	}

	return dependents, nil
}

// Helper functions

func (m *Manager) buildResourceName(name, organization string) string {
	if organization != "" {
		return fmt.Sprintf("%s/%s", organization, name)
	}
	return name
}

func (m *Manager) serviceToResource(service *types.Service) (*state.Resource, error) {
	// Marshal spec
	specMap, err := structToMap(service.Spec)
	if err != nil {
		return nil, err
	}

	// Marshal status
	statusMap, err := structToMap(service.Status)
	if err != nil {
		return nil, err
	}

	// Add metadata to spec for filtering
	specMap["organization"] = service.Metadata.Organization
	specMap["name"] = service.Metadata.Name

	now := time.Now()
	return &state.Resource{
		Name:       m.buildResourceName(service.Metadata.Name, service.Metadata.Organization),
		Kind:       service.Kind,
		APIVersion: service.APIVersion,
		Spec:       specMap,
		Status:     statusMap,
		CreatedAt:  now,
		UpdatedAt:  now,
	}, nil
}

func (m *Manager) resourceToService(resource *state.Resource) (*types.Service, error) {
	// Marshal and unmarshal to convert maps to structs
	data, err := json.Marshal(map[string]interface{}{
		"apiVersion": resource.APIVersion,
		"kind":       resource.Kind,
		"metadata": map[string]interface{}{
			"name":         resource.Spec["name"],
			"organization": resource.Spec["organization"],
		},
		"spec":   resource.Spec,
		"status": resource.Status,
	})
	if err != nil {
		return nil, err
	}

	var service types.Service
	if err := json.Unmarshal(data, &service); err != nil {
		return nil, err
	}

	return &service, nil
}

func structToMap(v interface{}) (map[string]interface{}, error) {
	data, err := json.Marshal(v)
	if err != nil {
		return nil, err
	}

	var m map[string]interface{}
	if err := json.Unmarshal(data, &m); err != nil {
		return nil, err
	}

	return m, nil
}
