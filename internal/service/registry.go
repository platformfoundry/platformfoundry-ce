package service

import (
	"strings"
	"sync"

	"github.com/platformfoundry/platformfoundry-ce/pkg/types"
)

// Registry provides an in-memory cache of services for fast lookups
type Registry struct {
	services map[string]*types.Service // key: "org/name"
	mu       sync.RWMutex
}

// NewRegistry creates a new service registry
func NewRegistry() *Registry {
	return &Registry{
		services: make(map[string]*types.Service),
	}
}

// Add adds or updates a service in the registry
func (r *Registry) Add(service *types.Service) {
	r.mu.Lock()
	defer r.mu.Unlock()

	key := r.buildKey(service.Metadata.Name, service.Metadata.Organization)
	r.services[key] = service
}

// Update updates a service in the registry
func (r *Registry) Update(service *types.Service) {
	// Same as Add for in-memory registry
	r.Add(service)
}

// Get retrieves a service by name and organization
func (r *Registry) Get(name, organization string) *types.Service {
	r.mu.RLock()
	defer r.mu.RUnlock()

	key := r.buildKey(name, organization)
	return r.services[key]
}

// Delete removes a service from the registry
func (r *Registry) Delete(name, organization string) {
	r.mu.Lock()
	defer r.mu.Unlock()

	key := r.buildKey(name, organization)
	delete(r.services, key)
}

// List returns all services, optionally filtered by organization
func (r *Registry) List(organization string) []*types.Service {
	r.mu.RLock()
	defer r.mu.RUnlock()

	result := make([]*types.Service, 0)
	for _, service := range r.services {
		if organization == "" || service.Metadata.Organization == organization {
			result = append(result, service)
		}
	}

	return result
}

// Search searches for services by name (case-insensitive, partial match)
func (r *Registry) Search(query, organization string) []*types.Service {
	r.mu.RLock()
	defer r.mu.RUnlock()

	query = strings.ToLower(query)
	result := make([]*types.Service, 0)

	for _, service := range r.services {
		// Filter by organization if specified
		if organization != "" && service.Metadata.Organization != organization {
			continue
		}

		// Search in name, team, and type
		if strings.Contains(strings.ToLower(service.Metadata.Name), query) ||
			strings.Contains(strings.ToLower(service.Spec.Owner.Team), query) ||
			strings.Contains(strings.ToLower(string(service.Spec.Type)), query) {
			result = append(result, service)
		}
	}

	return result
}

// FilterByLabels filters services by labels
func (r *Registry) FilterByLabels(labels map[string]string, organization string) []*types.Service {
	r.mu.RLock()
	defer r.mu.RUnlock()

	result := make([]*types.Service, 0)

	for _, service := range r.services {
		// Filter by organization if specified
		if organization != "" && service.Metadata.Organization != organization {
			continue
		}

		// Check if all label criteria match
		matches := true
		for key, value := range labels {
			if serviceValue, ok := service.Metadata.Labels[key]; !ok || serviceValue != value {
				matches = false
				break
			}
		}

		if matches {
			result = append(result, service)
		}
	}

	return result
}

// FilterByState filters services by state
func (r *Registry) FilterByState(state types.ServiceState, organization string) []*types.Service {
	r.mu.RLock()
	defer r.mu.RUnlock()

	result := make([]*types.Service, 0)

	for _, service := range r.services {
		// Filter by organization if specified
		if organization != "" && service.Metadata.Organization != organization {
			continue
		}

		if service.Status.State == state {
			result = append(result, service)
		}
	}

	return result
}

// FilterByHealth filters services by health status
func (r *Registry) FilterByHealth(health types.ServiceHealth, organization string) []*types.Service {
	r.mu.RLock()
	defer r.mu.RUnlock()

	result := make([]*types.Service, 0)

	for _, service := range r.services {
		// Filter by organization if specified
		if organization != "" && service.Metadata.Organization != organization {
			continue
		}

		if service.Status.Health == health {
			result = append(result, service)
		}
	}

	return result
}

// GetByType returns all services of a specific type
func (r *Registry) GetByType(serviceType types.ServiceType, organization string) []*types.Service {
	r.mu.RLock()
	defer r.mu.RUnlock()

	result := make([]*types.Service, 0)

	for _, service := range r.services {
		// Filter by organization if specified
		if organization != "" && service.Metadata.Organization != organization {
			continue
		}

		if service.Spec.Type == serviceType {
			result = append(result, service)
		}
	}

	return result
}

// GetByTeam returns all services owned by a specific team
func (r *Registry) GetByTeam(team, organization string) []*types.Service {
	r.mu.RLock()
	defer r.mu.RUnlock()

	result := make([]*types.Service, 0)

	for _, service := range r.services {
		// Filter by organization if specified
		if organization != "" && service.Metadata.Organization != organization {
			continue
		}

		if service.Spec.Owner.Team == team {
			result = append(result, service)
		}
	}

	return result
}

// GetStats returns registry statistics
func (r *Registry) GetStats(organization string) RegistryStats {
	r.mu.RLock()
	defer r.mu.RUnlock()

	stats := RegistryStats{
		Total:    0,
		ByType:   make(map[types.ServiceType]int),
		ByState:  make(map[types.ServiceState]int),
		ByHealth: make(map[types.ServiceHealth]int),
	}

	for _, service := range r.services {
		// Filter by organization if specified
		if organization != "" && service.Metadata.Organization != organization {
			continue
		}

		stats.Total++
		stats.ByType[service.Spec.Type]++
		stats.ByState[service.Status.State]++
		stats.ByHealth[service.Status.Health]++
	}

	return stats
}

// Clear removes all services from the registry
func (r *Registry) Clear() {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.services = make(map[string]*types.Service)
}

// Size returns the number of services in the registry
func (r *Registry) Size() int {
	r.mu.RLock()
	defer r.mu.RUnlock()

	return len(r.services)
}

// buildKey creates a unique key for a service
func (r *Registry) buildKey(name, organization string) string {
	if organization != "" {
		return organization + "/" + name
	}
	return name
}

// RegistryStats holds statistics about the registry
type RegistryStats struct {
	Total    int
	ByType   map[types.ServiceType]int
	ByState  map[types.ServiceState]int
	ByHealth map[types.ServiceHealth]int
}
