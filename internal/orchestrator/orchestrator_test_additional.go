package orchestrator

import (
	"testing"

	"github.com/platformfoundry/pf-ce/internal/plugin"
	"github.com/platformfoundry/pf-ce/internal/store"
	"github.com/platformfoundry/pf-ce/pkg/types"
	"github.com/stretchr/testify/assert"
)

func TestOrchestrator_ApplyPlatformWithEnvironment(t *testing.T) {
	pm := plugin.NewManager()
	s, _ := store.New()
	orch := New(pm, s)

	platform := &types.Platform{
		Metadata: types.Metadata{
			Name:         "test-platform",
			Organization: "test-org",
		},
		Spec: types.PlatformSpec{
			Components: types.ComponentReferences{
				Infrastructure: "infra-aws",
				Orchestrator:   "argocd",
			},
		},
	}

	env := &types.Environment{
		Metadata: types.Metadata{
			Name: "production",
		},
		Spec: types.EnvironmentSpec{
			Type: "production",
			Overrides: types.EnvironmentOverrides{
				Infrastructure: map[string]interface{}{
					"replicas": 3,
				},
			},
		},
	}

	err := orch.ApplyPlatform(platform, env)
	// Will error due to missing plugins, but should not panic
	assert.Error(t, err)
}

func TestOrchestrator_ApplyPlatformWithoutEnvironment(t *testing.T) {
	pm := plugin.NewManager()
	s, _ := store.New()
	orch := New(pm, s)

	platform := &types.Platform{
		Metadata: types.Metadata{
			Name:         "test-platform",
			Organization: "test-org",
		},
		Spec: types.PlatformSpec{
			Components: types.ComponentReferences{
				Infrastructure: "infra-aws",
			},
		},
	}

	err := orch.ApplyPlatform(platform, nil)
	// Will error due to missing plugins, but should handle nil environment
	assert.Error(t, err)
}

func TestOrchestrator_ConvertToResource(t *testing.T) {
	pm := plugin.NewManager()
	s, _ := store.New()
	orch := New(pm, s)

	// Test with actual Resource type
	resource := types.Resource{
		Metadata: types.Metadata{
			Name: "test-resource",
		},
		Kind: "Infrastructure",
		Spec: map[string]interface{}{
			"provider": "terraform",
		},
	}

	converted, ok := orch.convertToResource(resource)
	assert.True(t, ok)
	assert.Equal(t, resource.Metadata.Name, converted.Metadata.Name)

	// Test with incompatible type
	_, ok = orch.convertToResource("invalid")
	assert.False(t, ok)
}

func TestOrchestrator_ApplyMultipleResources(t *testing.T) {
	pm := plugin.NewManager()
	s, _ := store.New()
	orch := New(pm, s)

	resources := []interface{}{
		types.Resource{
			Metadata: types.Metadata{Name: "resource-1"},
			Kind:     "Infrastructure",
			Spec:     map[string]interface{}{"provider": "terraform"},
		},
		types.Resource{
			Metadata: types.Metadata{Name: "resource-2"},
			Kind:     "Cluster",
			Spec:     map[string]interface{}{"provider": "kubernetes"},
		},
	}

	err := orch.Apply(resources)
	// Will fail due to missing plugins, but should process all resources
	assert.Error(t, err)
}

func TestOrchestrator_DeleteNonExistent(t *testing.T) {
	pm := plugin.NewManager()
	s, _ := store.New()
	orch := New(pm, s)

	err := orch.Delete("non-existent", "Platform")
	assert.Error(t, err, "should error when deleting non-existent resource")
	assert.Contains(t, err.Error(), "not found")
}
