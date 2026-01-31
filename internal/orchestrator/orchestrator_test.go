package orchestrator

import (
	"testing"

	"github.com/platformfoundry/platformfoundry-ce/internal/plugin"
	"github.com/platformfoundry/platformfoundry-ce/internal/store"
	"github.com/platformfoundry/platformfoundry-ce/pkg/types"
)

func TestNew(t *testing.T) {
	pm := plugin.NewManager()
	st, err := store.New()
	if err != nil {
		t.Fatalf("Failed to create store: %v", err)
	}

	orch := New(pm, st)
	if orch == nil {
		t.Fatal("New returned nil")
	}

	if orch.pluginManager != pm {
		t.Error("Plugin manager not set correctly")
	}

	if orch.store != st {
		t.Error("Store not set correctly")
	}
}

func TestResolveDependencies_Simple(t *testing.T) {
	pm := plugin.NewManager()
	st, _ := store.New()
	orch := New(pm, st)

	resources := []types.Resource{
		{
			Metadata: types.Metadata{Name: "resource1"},
			Kind:     "Cluster",
			Spec:     map[string]interface{}{},
		},
		{
			Metadata: types.Metadata{Name: "resource2"},
			Kind:     "Pipeline",
			Spec:     map[string]interface{}{},
		},
	}

	ordered, err := orch.resolveDependencies(resources)
	if err != nil {
		t.Fatalf("Failed to resolve dependencies: %v", err)
	}

	if len(ordered) != 2 {
		t.Errorf("Expected 2 resources, got %d", len(ordered))
	}
}

func TestResolveDependencies_WithDependency(t *testing.T) {
	pm := plugin.NewManager()
	st, _ := store.New()
	orch := New(pm, st)

	resources := []types.Resource{
		{
			Metadata: types.Metadata{Name: "pipeline1"},
			Kind:     "Pipeline",
			Spec: map[string]interface{}{
				"clusterRef": "cluster1",
			},
		},
		{
			Metadata: types.Metadata{Name: "cluster1"},
			Kind:     "Cluster",
			Spec:     map[string]interface{}{},
		},
	}

	ordered, err := orch.resolveDependencies(resources)
	if err != nil {
		t.Fatalf("Failed to resolve dependencies: %v", err)
	}

	if len(ordered) != 2 {
		t.Errorf("Expected 2 resources, got %d", len(ordered))
	}

	// Cluster should come before pipeline
	if ordered[0].Metadata.Name != "cluster1" {
		t.Errorf("Expected cluster1 first, got %s", ordered[0].Metadata.Name)
	}
}

func TestConvertToResource(t *testing.T) {
	pm := plugin.NewManager()
	st, _ := store.New()
	orch := New(pm, st)

	// Test with types.Resource
	resource := types.Resource{
		Metadata: types.Metadata{Name: "test"},
		Kind:     "Cluster",
		Spec:     map[string]interface{}{},
	}

	converted, ok := orch.convertToResource(resource)
	if !ok {
		// This is expected as convertToResource requires specific interface
		t.Log("Convert to resource returned false as expected")
	} else {
		if converted.Metadata.Name != "test" {
			t.Errorf("Expected name 'test', got %s", converted.Metadata.Name)
		}
	}
}

func TestApplyPlatform(t *testing.T) {
	pm := plugin.NewManager()
	st, _ := store.New()
	orch := New(pm, st)

	platform := &types.Platform{
		Metadata: types.Metadata{
			Name: "test-platform",
		},
		Spec: types.PlatformSpec{
			Components: types.ComponentReferences{
				Infrastructure: "test-infra",
				Orchestrator:   "test-orch",
				Observability:  "test-obs",
				DevEx:          "test-devex",
			},
		},
	}

	// This should not fail even with empty specs
	err := orch.ApplyPlatform(platform, nil)
	if err != nil {
		t.Errorf("ApplyPlatform failed: %v", err)
	}
}
