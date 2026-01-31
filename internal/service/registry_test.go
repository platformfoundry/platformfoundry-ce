package service

import (
	"testing"

	"github.com/platformfoundry/platformfoundry-ce/pkg/types"
)

func TestRegistry_AddAndGet(t *testing.T) {
	registry := NewRegistry()

	service := &types.Service{
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

	registry.Add(service)

	retrieved := registry.Get("user-api", "acme-corp")
	if retrieved == nil {
		t.Fatal("Service not found in registry")
	}

	if retrieved.Metadata.Name != "user-api" {
		t.Errorf("Expected name 'user-api', got '%s'", retrieved.Metadata.Name)
	}
}

func TestRegistry_Delete(t *testing.T) {
	registry := NewRegistry()

	service := &types.Service{
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

	registry.Add(service)
	registry.Delete("user-api", "acme-corp")

	retrieved := registry.Get("user-api", "acme-corp")
	if retrieved != nil {
		t.Error("Service still exists after deletion")
	}
}

func TestRegistry_List(t *testing.T) {
	registry := NewRegistry()

	services := []*types.Service{
		{
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
			Metadata: types.Metadata{
				Name:         "auth-api",
				Organization: "acme-corp",
			},
			Spec: types.ServiceSpec{
				Type: types.ServiceTypeAPI,
				Owner: types.ServiceOwner{
					Team: "security-team",
				},
			},
		},
		{
			Metadata: types.Metadata{
				Name:         "billing-api",
				Organization: "other-org",
			},
			Spec: types.ServiceSpec{
				Type: types.ServiceTypeMicroservice,
				Owner: types.ServiceOwner{
					Team: "billing-team",
				},
			},
		},
	}

	for _, svc := range services {
		registry.Add(svc)
	}

	// List all services
	allServices := registry.List("")
	if len(allServices) != 3 {
		t.Errorf("Expected 3 services, got %d", len(allServices))
	}

	// List services by organization
	acmeServices := registry.List("acme-corp")
	if len(acmeServices) != 2 {
		t.Errorf("Expected 2 services for acme-corp, got %d", len(acmeServices))
	}
}

func TestRegistry_Search(t *testing.T) {
	registry := NewRegistry()

	services := []*types.Service{
		{
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
			Metadata: types.Metadata{
				Name:         "auth-api",
				Organization: "acme-corp",
			},
			Spec: types.ServiceSpec{
				Type: types.ServiceTypeAPI,
				Owner: types.ServiceOwner{
					Team: "security-team",
				},
			},
		},
	}

	for _, svc := range services {
		registry.Add(svc)
	}

	// Search by name
	results := registry.Search("user", "acme-corp")
	if len(results) != 1 {
		t.Errorf("Expected 1 result for 'user', got %d", len(results))
	}

	// Search by team
	results = registry.Search("security", "acme-corp")
	if len(results) != 1 {
		t.Errorf("Expected 1 result for 'security', got %d", len(results))
	}

	// Search by type
	results = registry.Search("api", "acme-corp")
	if len(results) != 2 {
		t.Errorf("Expected 2 results for 'api', got %d", len(results))
	}
}

func TestRegistry_FilterByLabels(t *testing.T) {
	registry := NewRegistry()

	services := []*types.Service{
		{
			Metadata: types.Metadata{
				Name:         "user-api",
				Organization: "acme-corp",
				Labels: map[string]string{
					"env":  "production",
					"tier": "backend",
				},
			},
			Spec: types.ServiceSpec{
				Type: types.ServiceTypeMicroservice,
				Owner: types.ServiceOwner{
					Team: "platform-team",
				},
			},
		},
		{
			Metadata: types.Metadata{
				Name:         "auth-api",
				Organization: "acme-corp",
				Labels: map[string]string{
					"env":  "staging",
					"tier": "backend",
				},
			},
			Spec: types.ServiceSpec{
				Type: types.ServiceTypeAPI,
				Owner: types.ServiceOwner{
					Team: "security-team",
				},
			},
		},
	}

	for _, svc := range services {
		registry.Add(svc)
	}

	// Filter by single label
	results := registry.FilterByLabels(map[string]string{"env": "production"}, "acme-corp")
	if len(results) != 1 {
		t.Errorf("Expected 1 result, got %d", len(results))
	}

	// Filter by multiple labels
	results = registry.FilterByLabels(map[string]string{"env": "staging", "tier": "backend"}, "acme-corp")
	if len(results) != 1 {
		t.Errorf("Expected 1 result, got %d", len(results))
	}
}

func TestRegistry_FilterByState(t *testing.T) {
	registry := NewRegistry()

	services := []*types.Service{
		{
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
				State: types.ServiceStateActive,
			},
		},
		{
			Metadata: types.Metadata{
				Name:         "auth-api",
				Organization: "acme-corp",
			},
			Spec: types.ServiceSpec{
				Type: types.ServiceTypeAPI,
				Owner: types.ServiceOwner{
					Team: "security-team",
				},
			},
			Status: types.ServiceStatus{
				State: types.ServiceStateDraft,
			},
		},
	}

	for _, svc := range services {
		registry.Add(svc)
	}

	// Filter by state
	results := registry.FilterByState(types.ServiceStateActive, "acme-corp")
	if len(results) != 1 {
		t.Errorf("Expected 1 active service, got %d", len(results))
	}

	if results[0].Metadata.Name != "user-api" {
		t.Errorf("Expected 'user-api', got '%s'", results[0].Metadata.Name)
	}
}

func TestRegistry_GetByType(t *testing.T) {
	registry := NewRegistry()

	services := []*types.Service{
		{
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
			Metadata: types.Metadata{
				Name:         "auth-api",
				Organization: "acme-corp",
			},
			Spec: types.ServiceSpec{
				Type: types.ServiceTypeAPI,
				Owner: types.ServiceOwner{
					Team: "security-team",
				},
			},
		},
	}

	for _, svc := range services {
		registry.Add(svc)
	}

	// Get by type
	results := registry.GetByType(types.ServiceTypeMicroservice, "acme-corp")
	if len(results) != 1 {
		t.Errorf("Expected 1 microservice, got %d", len(results))
	}

	if results[0].Metadata.Name != "user-api" {
		t.Errorf("Expected 'user-api', got '%s'", results[0].Metadata.Name)
	}
}

func TestRegistry_GetByTeam(t *testing.T) {
	registry := NewRegistry()

	services := []*types.Service{
		{
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
			Metadata: types.Metadata{
				Name:         "auth-api",
				Organization: "acme-corp",
			},
			Spec: types.ServiceSpec{
				Type: types.ServiceTypeAPI,
				Owner: types.ServiceOwner{
					Team: "security-team",
				},
			},
		},
	}

	for _, svc := range services {
		registry.Add(svc)
	}

	// Get by team
	results := registry.GetByTeam("platform-team", "acme-corp")
	if len(results) != 1 {
		t.Errorf("Expected 1 service for platform-team, got %d", len(results))
	}

	if results[0].Metadata.Name != "user-api" {
		t.Errorf("Expected 'user-api', got '%s'", results[0].Metadata.Name)
	}
}

func TestRegistry_GetStats(t *testing.T) {
	registry := NewRegistry()

	services := []*types.Service{
		{
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
				State:  types.ServiceStateActive,
				Health: types.ServiceHealthHealthy,
			},
		},
		{
			Metadata: types.Metadata{
				Name:         "auth-api",
				Organization: "acme-corp",
			},
			Spec: types.ServiceSpec{
				Type: types.ServiceTypeAPI,
				Owner: types.ServiceOwner{
					Team: "security-team",
				},
			},
			Status: types.ServiceStatus{
				State:  types.ServiceStateDraft,
				Health: types.ServiceHealthUnknown,
			},
		},
	}

	for _, svc := range services {
		registry.Add(svc)
	}

	// Get stats
	stats := registry.GetStats("acme-corp")

	if stats.Total != 2 {
		t.Errorf("Expected total 2, got %d", stats.Total)
	}

	if stats.ByType[types.ServiceTypeMicroservice] != 1 {
		t.Errorf("Expected 1 microservice, got %d", stats.ByType[types.ServiceTypeMicroservice])
	}

	if stats.ByState[types.ServiceStateActive] != 1 {
		t.Errorf("Expected 1 active service, got %d", stats.ByState[types.ServiceStateActive])
	}

	if stats.ByHealth[types.ServiceHealthHealthy] != 1 {
		t.Errorf("Expected 1 healthy service, got %d", stats.ByHealth[types.ServiceHealthHealthy])
	}
}

func TestRegistry_Clear(t *testing.T) {
	registry := NewRegistry()

	service := &types.Service{
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

	registry.Add(service)
	registry.Clear()

	if registry.Size() != 0 {
		t.Errorf("Expected size 0 after clear, got %d", registry.Size())
	}
}
