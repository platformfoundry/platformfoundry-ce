package planner

import (
	"path/filepath"
	"testing"

	"github.com/platformfoundry/platformfoundry-ce/internal/plugin"
	"github.com/platformfoundry/platformfoundry-ce/internal/state"
	"github.com/platformfoundry/platformfoundry-ce/internal/store"
	"github.com/platformfoundry/platformfoundry-ce/pkg/types"
)

// createTestStore creates a store with a unique temporary database for testing
func createTestStore(t *testing.T) *store.Store {
	t.Helper()

	// Create temporary directory
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	// Create local backend with temp path
	backend, err := state.NewLocalBackend(dbPath)
	if err != nil {
		t.Fatalf("Failed to create test backend: %v", err)
	}

	// Register cleanup to close backend
	t.Cleanup(func() {
		backend.Close()
	})

	return store.NewWithBackend(backend)
}

func TestNew(t *testing.T) {
	pm := plugin.NewManager()
	st := createTestStore(t)

	plnr := New(pm, st)
	if plnr == nil {
		t.Fatal("New returned nil")
	}
}

func TestCreatePlan_NewResources(t *testing.T) {
	pm := plugin.NewManager()
	st := createTestStore(t)
	plnr := New(pm, st)

	resources := []interface{}{
		types.Resource{
			Metadata: types.Metadata{
				Name: "test-cluster",
			},
			Kind: "Cluster",
			Spec: map[string]interface{}{
				"provider": "existing",
			},
		},
	}

	plan, err := plnr.CreatePlan(resources)
	if err != nil {
		t.Fatalf("CreatePlan failed: %v", err)
	}

	if len(plan.ToCreate) != 1 {
		t.Errorf("Expected 1 resource to create, got %d", len(plan.ToCreate))
	}

	if plan.ToCreate[0].Name != "test-cluster" {
		t.Errorf("Expected resource name 'test-cluster', got %s", plan.ToCreate[0].Name)
	}
}

func TestCreatePlan_Empty(t *testing.T) {
	pm := plugin.NewManager()
	st := createTestStore(t)
	plnr := New(pm, st)

	resources := []interface{}{}

	plan, err := plnr.CreatePlan(resources)
	if err != nil {
		t.Fatalf("CreatePlan failed: %v", err)
	}

	if len(plan.ToCreate) != 0 {
		t.Errorf("Expected 0 resources to create, got %d", len(plan.ToCreate))
	}
}

func TestDetectChanges(t *testing.T) {
	pm := plugin.NewManager()
	st := createTestStore(t)
	plnr := New(pm, st)

	existing := &types.Resource{
		Metadata: types.Metadata{
			Name:   "test",
			Labels: map[string]string{"env": "dev"},
		},
		Spec: map[string]interface{}{
			"key": "value1",
		},
	}

	new := &types.Resource{
		Metadata: types.Metadata{
			Name:   "test",
			Labels: map[string]string{"env": "prod"},
		},
		Spec: map[string]interface{}{
			"key": "value2",
		},
	}

	// Store the existing resource first
	st.Save(existing.Metadata.Name, existing.Kind, "", existing.Spec, "active")

	changes := plnr.detectChanges(new.Metadata.Name, new)

	if len(changes) == 0 {
		t.Error("Expected changes to be detected")
	}

	// Check that spec change was detected
	hasSpecChange := false
	for _, change := range changes {
		if change == "spec updated" {
			hasSpecChange = true
		}
	}

	if !hasSpecChange {
		t.Error("Expected spec change to be detected")
	}
}

func TestMapsEqual(t *testing.T) {
	tests := []struct {
		name     string
		a        map[string]string
		b        map[string]string
		expected bool
	}{
		{
			name:     "Equal maps",
			a:        map[string]string{"key1": "value1", "key2": "value2"},
			b:        map[string]string{"key1": "value1", "key2": "value2"},
			expected: true,
		},
		{
			name:     "Different values",
			a:        map[string]string{"key1": "value1"},
			b:        map[string]string{"key1": "value2"},
			expected: false,
		},
		{
			name:     "Different keys",
			a:        map[string]string{"key1": "value1"},
			b:        map[string]string{"key2": "value1"},
			expected: false,
		},
		{
			name:     "Different lengths",
			a:        map[string]string{"key1": "value1"},
			b:        map[string]string{"key1": "value1", "key2": "value2"},
			expected: false,
		},
		{
			name:     "Empty maps",
			a:        map[string]string{},
			b:        map[string]string{},
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := mapsEqual(tt.a, tt.b)
			if result != tt.expected {
				t.Errorf("Expected %v, got %v", tt.expected, result)
			}
		})
	}
}
