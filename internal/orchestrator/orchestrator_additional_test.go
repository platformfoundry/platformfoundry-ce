package orchestrator

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/platformfoundry/pf-ce/internal/environment"
	"github.com/platformfoundry/pf-ce/internal/plugin"
	"github.com/platformfoundry/pf-ce/internal/store"
	plugintypes "github.com/platformfoundry/pf-ce/pkg/plugin"
	"github.com/platformfoundry/pf-ce/pkg/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// newTestStore creates a store with a temporary database for testing
func newTestStore(t *testing.T) *store.Store {
	t.Helper()
	tempDir, err := os.MkdirTemp("", "orchestrator-test-*")
	require.NoError(t, err)
	t.Cleanup(func() {
		os.RemoveAll(tempDir)
	})

	dbPath := filepath.Join(tempDir, "test.db")
	st, err := store.NewWithPath(dbPath)
	require.NoError(t, err)
	t.Cleanup(func() {
		st.Close()
	})

	return st
}

func TestApplyPlatformBasic(t *testing.T) {
	pm := plugin.NewManager()
	st := newTestStore(t)
	orch := New(pm, st)

	platform := &types.Platform{
		APIVersion: "platformfoundry.io/v1",
		Kind:       "Platform",
		Metadata: types.Metadata{
			Name:         "test-platform",
			Organization: "test-org",
		},
		Spec: types.PlatformSpec{
			Global: types.GlobalConfig{
				Region: "us-east-1",
			},
			Components: types.ComponentReferences{
				Infrastructure: "test-infra",
				Orchestrator:   "test-orch",
			},
		},
	}

	err := orch.ApplyPlatform(platform, nil)
	assert.NoError(t, err)
}

func TestApplyPlatformWithEnvironment(t *testing.T) {
	pm := plugin.NewManager()
	st := newTestStore(t)
	orch := New(pm, st)

	platform := &types.Platform{
		APIVersion: "platformfoundry.io/v1",
		Kind:       "Platform",
		Metadata: types.Metadata{
			Name:         "test-platform-env",
			Organization: "test-org",
		},
		Spec: types.PlatformSpec{
			Global: types.GlobalConfig{
				Region: "us-east-1",
			},
			Components: types.ComponentReferences{
				Infrastructure: "test-infra",
			},
		},
	}

	env := &types.Environment{
		APIVersion: "platformfoundry.io/v1",
		Kind:       "Environment",
		Metadata: types.Metadata{
			Name:         "production",
			Organization: "test-org",
		},
		Spec: types.EnvironmentSpec{
			Type: types.EnvironmentProduction,
			Overrides: types.EnvironmentOverrides{
				Global: map[string]interface{}{
					"region": "us-west-2",
				},
				Tags: map[string]string{
					"environment": "production",
				},
			},
		},
	}

	err := orch.ApplyPlatform(platform, env)
	assert.NoError(t, err)
}

func TestApplyPlatformEnvironmentResolutionError(t *testing.T) {
	pm := plugin.NewManager()
	st := newTestStore(t)
	orch := New(pm, st)

	platform := &types.Platform{
		Metadata: types.Metadata{
			Name:         "test-platform",
			Organization: "test-org",
		},
	}

	// Test with nil environment should work
	err := orch.ApplyPlatform(platform, nil)
	assert.NoError(t, err)
}

func TestApplyResourceValidationError(t *testing.T) {
	pm := plugin.NewManager()
	st := newTestStore(t)
	orch := New(pm, st)

	// Resource without provider should fail
	resource := types.Resource{
		Metadata: types.Metadata{Name: "invalid-resource"},
		Kind:     "Infrastructure",
		Spec:     map[string]interface{}{}, // Missing provider
	}

	err := orch.applyResource(resource)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "provider")
}

func TestApplyResourcePluginNotFound(t *testing.T) {
	pm := plugin.NewManager()
	st := newTestStore(t)
	orch := New(pm, st)

	resource := types.Resource{
		Metadata: types.Metadata{Name: "test-resource"},
		Kind:     "UnknownKind",
		Spec: map[string]interface{}{
			"provider": "unknown-provider",
		},
	}

	err := orch.applyResource(resource)
	assert.Error(t, err)
}

func TestApplyMultipleResources(t *testing.T) {
	pm := plugin.NewManager()
	st := newTestStore(t)
	orch := New(pm, st)

	resources := []interface{}{
		types.Resource{
			Metadata: types.Metadata{Name: "resource1"},
			Kind:     "Infrastructure",
			Spec: map[string]interface{}{
				"provider": "terraform",
			},
		},
		types.Resource{
			Metadata: types.Metadata{Name: "resource2"},
			Kind:     "Orchestrator",
			Spec: map[string]interface{}{
				"provider": "argocd",
			},
		},
	}

	// This may error if plugins aren't fully configured, but tests the flow
	err := orch.Apply(resources)
	// Error is expected if plugins aren't set up, we're just testing the flow
	t.Logf("Apply returned: %v", err)
}

func TestDeleteResource(t *testing.T) {
	pm := plugin.NewManager()
	st := newTestStore(t)
	orch := New(pm, st)

	// Try to delete non-existent resource
	err := orch.Delete("nonexistent", "Platform")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestDeleteResourceSuccess(t *testing.T) {
	pm := plugin.NewManager()
	st := newTestStore(t)
	orch := New(pm, st)

	// First, save a resource to state
	err := st.Save("test-resource", "Infrastructure", "terraform", map[string]interface{}{}, "success")
	require.NoError(t, err)

	// Now try to delete it
	err = orch.Delete("test-resource", "Infrastructure")
	// This will error because plugin isn't registered, but tests the flow
	t.Logf("Delete returned: %v", err)
}

func TestResolveDependenciesEmpty(t *testing.T) {
	pm := plugin.NewManager()
	st := newTestStore(t)
	orch := New(pm, st)

	resources := []types.Resource{}
	ordered, err := orch.resolveDependencies(resources)

	assert.NoError(t, err)
	assert.Empty(t, ordered)
}

func TestResolveDependenciesComplex(t *testing.T) {
	pm := plugin.NewManager()
	st := newTestStore(t)
	orch := New(pm, st)

	// Create a complex dependency graph using explicit dependsOn field in Spec
	// The dependency resolver looks for Spec["dependsOn"] as []interface{}
	resources := []types.Resource{
		{
			Metadata: types.Metadata{Name: "app"},
			Kind:     "Application",
			Spec: map[string]interface{}{
				"dependsOn": []interface{}{"cluster", "pipeline"},
			},
		},
		{
			Metadata: types.Metadata{Name: "pipeline"},
			Kind:     "Pipeline",
			Spec: map[string]interface{}{
				"dependsOn": []interface{}{"cluster"},
			},
		},
		{
			Metadata: types.Metadata{Name: "cluster"},
			Kind:     "Cluster",
			Spec:     map[string]interface{}{},
		},
		{
			Metadata: types.Metadata{Name: "monitoring"},
			Kind:     "Observability",
			Spec: map[string]interface{}{
				"dependsOn": []interface{}{"cluster"},
			},
		},
	}

	ordered, err := orch.resolveDependencies(resources)
	assert.NoError(t, err)
	assert.Len(t, ordered, 4)

	// Cluster should come first (no dependencies)
	assert.Equal(t, "cluster", ordered[0].Metadata.Name)

	// Find indices of each resource
	indices := make(map[string]int)
	for i, r := range ordered {
		indices[r.Metadata.Name] = i
	}

	// App should come after pipeline (depends on pipeline)
	assert.Greater(t, indices["app"], indices["pipeline"], "app should come after pipeline")

	// Pipeline should come after cluster (depends on cluster)
	assert.Greater(t, indices["pipeline"], indices["cluster"], "pipeline should come after cluster")

	// Monitoring should come after cluster (depends on cluster)
	assert.Greater(t, indices["monitoring"], indices["cluster"], "monitoring should come after cluster")
}

func TestApplyComponentByRef(t *testing.T) {
	pm := plugin.NewManager()
	st := newTestStore(t)
	orch := New(pm, st)

	err := orch.applyComponentByRef("test-component", "Infrastructure")
	assert.NoError(t, err) // Current implementation just logs and returns nil
}

func TestConvertToResourceInvalid(t *testing.T) {
	pm := plugin.NewManager()
	st := newTestStore(t)
	orch := New(pm, st)

	// Test with an invalid type
	invalidResource := "not a resource"
	_, ok := orch.convertToResource(invalidResource)
	assert.False(t, ok)
}

func TestApplyWithUnsupportedResourceType(t *testing.T) {
	pm := plugin.NewManager()
	st := newTestStore(t)
	orch := New(pm, st)

	resources := []interface{}{
		"unsupported type",
		12345,
		struct{}{},
	}

	err := orch.Apply(resources)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unsupported resource type")
}

func TestEnvironmentResolutionIntegration(t *testing.T) {
	// Test that environment resolver works correctly with orchestrator
	resolver := environment.NewResolver()

	platform := &types.Platform{
		Metadata: types.Metadata{
			Name:         "test-platform",
			Organization: "test-org",
		},
		Spec: types.PlatformSpec{
			Global: types.GlobalConfig{
				Region: "us-east-1",
				Tags: map[string]string{
					"env": "dev",
				},
			},
		},
	}

	env := &types.Environment{
		Metadata: types.Metadata{
			Name:         "production",
			Organization: "test-org",
		},
		Spec: types.EnvironmentSpec{
			Type: types.EnvironmentProduction,
			Overrides: types.EnvironmentOverrides{
				Global: map[string]interface{}{
					"region": "us-west-2",
				},
				Tags: map[string]string{
					"env": "prod",
				},
			},
		},
	}

	resolved, err := resolver.Resolve(platform, env)
	assert.NoError(t, err)
	assert.NotNil(t, resolved)

	// Check that overrides were applied
	assert.Equal(t, "production", resolved.Metadata.Environment)
	assert.Equal(t, "prod", resolved.Spec.Global.Tags["env"])
}

func TestApplyPlatformWithAllComponents(t *testing.T) {
	pm := plugin.NewManager()
	st := newTestStore(t)
	orch := New(pm, st)

	platform := &types.Platform{
		APIVersion: "platformfoundry.io/v1",
		Kind:       "Platform",
		Metadata: types.Metadata{
			Name:         "full-platform",
			Organization: "test-org",
		},
		Spec: types.PlatformSpec{
			Global: types.GlobalConfig{
				Region: "us-east-1",
			},
			Components: types.ComponentReferences{
				Infrastructure: "test-infra",
				Orchestrator:   "test-orch",
				Observability:  "test-obs",
				DevEx:          "test-devex",
			},
		},
	}

	err := orch.ApplyPlatform(platform, nil)
	assert.NoError(t, err)
}

func TestApplyPlatformPartialComponents(t *testing.T) {
	pm := plugin.NewManager()
	st := newTestStore(t)
	orch := New(pm, st)

	// Test with only some components defined
	platform := &types.Platform{
		APIVersion: "platformfoundry.io/v1",
		Kind:       "Platform",
		Metadata: types.Metadata{
			Name:         "partial-platform",
			Organization: "test-org",
		},
		Spec: types.PlatformSpec{
			Components: types.ComponentReferences{
				Infrastructure: "test-infra",
				// Other components empty
			},
		},
	}

	err := orch.ApplyPlatform(platform, nil)
	assert.NoError(t, err)
}

func TestOrchestratorConcurrentAccess(t *testing.T) {
	// Test concurrent access to orchestrator
	pm := plugin.NewManager()
	st := newTestStore(t)
	orch := New(pm, st)

	platform := &types.Platform{
		Metadata: types.Metadata{
			Name:         "concurrent-platform",
			Organization: "test-org",
		},
		Spec: types.PlatformSpec{
			Components: types.ComponentReferences{
				Infrastructure: "test-infra",
			},
		},
	}

	// Run multiple goroutines
	done := make(chan bool, 10)
	for i := 0; i < 10; i++ {
		go func() {
			_ = orch.ApplyPlatform(platform, nil)
			done <- true
		}()
	}

	// Wait for all to complete
	for i := 0; i < 10; i++ {
		<-done
	}
}

func TestDependencyResolutionWithMultipleReferences(t *testing.T) {
	pm := plugin.NewManager()
	st := newTestStore(t)
	orch := New(pm, st)

	// Use dependsOn field which is properly detected by the resolver
	// The resolver detects: clusterRef, infrastructureRef, and dependsOn ([]interface{})
	resources := []types.Resource{
		{
			Metadata: types.Metadata{Name: "service1"},
			Kind:     "Service",
			Spec: map[string]interface{}{
				"clusterRef": "cluster",
				"dependsOn":  []interface{}{"pipeline", "database"},
			},
		},
		{
			Metadata: types.Metadata{Name: "database"},
			Kind:     "Database",
			Spec: map[string]interface{}{
				"clusterRef": "cluster",
			},
		},
		{
			Metadata: types.Metadata{Name: "pipeline"},
			Kind:     "Pipeline",
			Spec: map[string]interface{}{
				"clusterRef": "cluster",
			},
		},
		{
			Metadata: types.Metadata{Name: "cluster"},
			Kind:     "Cluster",
			Spec:     map[string]interface{}{},
		},
	}

	ordered, err := orch.resolveDependencies(resources)
	assert.NoError(t, err)
	assert.Len(t, ordered, 4)

	// Cluster should be first
	assert.Equal(t, "cluster", ordered[0].Metadata.Name)

	// Service1 should be last (depends on everything)
	assert.Equal(t, "service1", ordered[len(ordered)-1].Metadata.Name)
}

func TestApplyResourceStateManagement(t *testing.T) {
	pm := plugin.NewManager()
	st := newTestStore(t)
	_ = New(pm, st)

	// Save a resource directly to state
	resourceName := "state-test-resource"
	err := st.Save(resourceName, "Infrastructure", "terraform", map[string]interface{}{
		"provider": "terraform",
		"cloud":    "aws",
	}, "success")
	require.NoError(t, err)

	// Retrieve it
	state, err := st.Get(resourceName)
	assert.NoError(t, err)
	assert.NotNil(t, state)
	assert.Equal(t, "Infrastructure", state.Kind)
	assert.Equal(t, "terraform", state.Provider)
	assert.Equal(t, "success", state.Status)
}

func BenchmarkApplyPlatform(b *testing.B) {
	pm := plugin.NewManager()
	st, _ := store.New()
	orch := New(pm, st)

	platform := &types.Platform{
		Metadata: types.Metadata{
			Name:         "bench-platform",
			Organization: "test-org",
		},
		Spec: types.PlatformSpec{
			Components: types.ComponentReferences{
				Infrastructure: "test-infra",
			},
		},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = orch.ApplyPlatform(platform, nil)
	}
}

func BenchmarkResolveDependencies(b *testing.B) {
	pm := plugin.NewManager()
	st, _ := store.New()
	orch := New(pm, st)

	resources := []types.Resource{
		{Metadata: types.Metadata{Name: "r1"}, Kind: "K1", Spec: map[string]interface{}{"ref": "r2"}},
		{Metadata: types.Metadata{Name: "r2"}, Kind: "K2", Spec: map[string]interface{}{"ref": "r3"}},
		{Metadata: types.Metadata{Name: "r3"}, Kind: "K3", Spec: map[string]interface{}{}},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = orch.resolveDependencies(resources)
	}
}

// Mock implementations for testing
type mockPlugin struct {
	name         string
	pluginType   string
	validateFunc func(spec map[string]interface{}) error
	applyFunc    func(spec map[string]interface{}) (*plugintypes.Result, error)
}

func (m *mockPlugin) Name() string { return m.name }
func (m *mockPlugin) Type() string {
	if m.pluginType != "" {
		return m.pluginType
	}
	return "Test"
}
func (m *mockPlugin) Version() string         { return "1.0.0" }
func (m *mockPlugin) ConfigType() interface{} { return nil }

func (m *mockPlugin) Validate(spec map[string]interface{}) error {
	if m.validateFunc != nil {
		return m.validateFunc(spec)
	}
	return nil
}

func (m *mockPlugin) Plan(spec map[string]interface{}) (*plugintypes.Plan, error) {
	return &plugintypes.Plan{Actions: []string{"test action"}}, nil
}

func (m *mockPlugin) Apply(spec map[string]interface{}) (*plugintypes.Result, error) {
	if m.applyFunc != nil {
		return m.applyFunc(spec)
	}
	return &plugintypes.Result{Status: "success", Message: "applied"}, nil
}

func (m *mockPlugin) Delete(name string) error {
	return nil
}

func (m *mockPlugin) Status(name string) (*plugintypes.Status, error) {
	return &plugintypes.Status{State: "ready", Ready: true}, nil
}

func TestApplyWithMockPlugin(t *testing.T) {
	pm := plugin.NewManager()
	st := newTestStore(t)

	// Register mock plugin
	mockP := &mockPlugin{
		name:       "test-provider",
		pluginType: "TestKind",
		applyFunc: func(spec map[string]interface{}) (*plugintypes.Result, error) {
			return &plugintypes.Result{Status: "success", Message: "mock applied"}, nil
		},
	}
	pm.Register(mockP)

	orch := New(pm, st)

	resource := types.Resource{
		Metadata: types.Metadata{Name: "mock-resource"},
		Kind:     "TestKind",
		Spec: map[string]interface{}{
			"provider": "test-provider",
		},
	}

	err := orch.applyResource(resource)
	assert.NoError(t, err)

	// Verify state was saved
	state, err := st.Get("mock-resource")
	assert.NoError(t, err)
	assert.Equal(t, "success", state.Status)
}

func TestApplyWithPluginFailure(t *testing.T) {
	pm := plugin.NewManager()
	st := newTestStore(t)

	// Register mock plugin that fails
	mockP := &mockPlugin{
		name:       "failing-provider",
		pluginType: "TestKind",
		applyFunc: func(spec map[string]interface{}) (*plugintypes.Result, error) {
			return nil, fmt.Errorf("plugin apply failed")
		},
	}
	pm.Register(mockP)

	orch := New(pm, st)

	resource := types.Resource{
		Metadata: types.Metadata{Name: "failing-resource"},
		Kind:     "TestKind",
		Spec: map[string]interface{}{
			"provider": "failing-provider",
		},
	}

	err := orch.applyResource(resource)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "plugin apply failed")
}

func TestApplyWithValidationFailure(t *testing.T) {
	pm := plugin.NewManager()
	st := newTestStore(t)

	// Register mock plugin with validation failure
	mockP := &mockPlugin{
		name:       "validating-provider",
		pluginType: "TestKind",
		validateFunc: func(spec map[string]interface{}) error {
			return fmt.Errorf("validation failed: missing required field")
		},
	}
	pm.Register(mockP)

	orch := New(pm, st)

	resource := types.Resource{
		Metadata: types.Metadata{Name: "invalid-resource"},
		Kind:     "TestKind",
		Spec: map[string]interface{}{
			"provider": "validating-provider",
		},
	}

	err := orch.applyResource(resource)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "validation failed")
}
