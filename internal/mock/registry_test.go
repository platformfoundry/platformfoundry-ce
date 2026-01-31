package mock

import (
	"sync"
	"testing"
	"time"
)

func TestNewRegistry(t *testing.T) {
	registry := NewRegistry(nil)

	if registry == nil {
		t.Fatal("expected non-nil registry")
	}
	if registry.Count() != 0 {
		t.Errorf("expected 0 mocks, got %d", registry.Count())
	}
	if registry.GetGlobalConfig() == nil {
		t.Error("expected default global config")
	}
}

func TestNewRegistryWithConfig(t *testing.T) {
	config := &MockConfig{
		Mode:         MockModeInstant,
		DefaultDelay: 100 * time.Millisecond,
	}

	registry := NewRegistry(config)

	globalConfig := registry.GetGlobalConfig()
	if globalConfig.Mode != MockModeInstant {
		t.Errorf("expected mode instant, got %s", globalConfig.Mode)
	}
}

func TestRegistryRegisterMock(t *testing.T) {
	registry := NewRegistry(nil)

	registry.RegisterMock("terraform", "Infrastructure", nil)

	if registry.Count() != 1 {
		t.Errorf("expected 1 mock, got %d", registry.Count())
	}

	mock, ok := registry.Get("Infrastructure", "terraform")
	if !ok {
		t.Error("expected to find registered mock")
	}
	if mock == nil {
		t.Fatal("expected non-nil mock")
	}
	if mock.Type() != "Infrastructure" {
		t.Errorf("expected type 'Infrastructure', got '%s'", mock.Type())
	}
}

func TestRegistryRegisterMockWithCustomConfig(t *testing.T) {
	registry := NewRegistry(nil)

	customConfig := &MockConfig{
		Mode: MockModeChaos,
	}

	registry.RegisterMock("terraform", "Infrastructure", customConfig)

	mock, _ := registry.Get("Infrastructure", "terraform")
	if mock.config.Mode != MockModeChaos {
		t.Errorf("expected mode chaos, got %s", mock.config.Mode)
	}
}

func TestRegistryRegisterMockFromPlugin(t *testing.T) {
	registry := NewRegistry(nil)

	realPlugin := &fakePlugin{
		name:    "real-terraform",
		typ:     "Infrastructure",
		version: "1.5.0",
	}

	registry.RegisterMockFromPlugin(realPlugin, nil)

	mock, ok := registry.Get("Infrastructure", "real-terraform")
	if !ok {
		t.Error("expected to find registered mock")
	}
	if mock.realPlugin != realPlugin {
		t.Error("expected mock to wrap real plugin")
	}
}

func TestRegistryGet(t *testing.T) {
	registry := NewRegistry(nil)

	registry.RegisterMock("terraform", "Infrastructure", nil)
	registry.RegisterMock("argocd", "Orchestrator", nil)

	// Get existing
	mock, ok := registry.Get("Infrastructure", "terraform")
	if !ok || mock == nil {
		t.Error("expected to find terraform mock")
	}

	// Get non-existent
	mock, ok = registry.Get("Infrastructure", "nonexistent")
	if ok || mock != nil {
		t.Error("expected not to find nonexistent mock")
	}
}

func TestRegistryGetOrCreate(t *testing.T) {
	registry := NewRegistry(nil)

	// Create new
	mock1 := registry.GetOrCreate("Infrastructure", "terraform")
	if mock1 == nil {
		t.Fatal("expected non-nil mock")
	}
	if registry.Count() != 1 {
		t.Errorf("expected 1 mock, got %d", registry.Count())
	}

	// Get existing
	mock2 := registry.GetOrCreate("Infrastructure", "terraform")
	if mock2 != mock1 {
		t.Error("expected to get same mock instance")
	}
	if registry.Count() != 1 {
		t.Errorf("expected still 1 mock, got %d", registry.Count())
	}
}

func TestRegistryList(t *testing.T) {
	registry := NewRegistry(nil)

	registry.RegisterMock("terraform", "Infrastructure", nil)
	registry.RegisterMock("argocd", "Orchestrator", nil)
	registry.RegisterMock("prometheus", "Observability", nil)

	list := registry.List()
	if len(list) != 3 {
		t.Errorf("expected 3 mocks in list, got %d", len(list))
	}

	// Check keys are correct
	if _, ok := list["Infrastructure:terraform"]; !ok {
		t.Error("expected Infrastructure:terraform in list")
	}
	if _, ok := list["Orchestrator:argocd"]; !ok {
		t.Error("expected Orchestrator:argocd in list")
	}
}

func TestRegistryListByType(t *testing.T) {
	registry := NewRegistry(nil)

	registry.RegisterMock("terraform", "Infrastructure", nil)
	registry.RegisterMock("aws", "Infrastructure", nil)
	registry.RegisterMock("argocd", "Orchestrator", nil)

	infraMocks := registry.ListByType("Infrastructure")
	if len(infraMocks) != 2 {
		t.Errorf("expected 2 infrastructure mocks, got %d", len(infraMocks))
	}

	orchMocks := registry.ListByType("Orchestrator")
	if len(orchMocks) != 1 {
		t.Errorf("expected 1 orchestrator mock, got %d", len(orchMocks))
	}

	emptyMocks := registry.ListByType("Nonexistent")
	if len(emptyMocks) != 0 {
		t.Errorf("expected 0 mocks for nonexistent type, got %d", len(emptyMocks))
	}
}

func TestRegistrySetGlobalConfig(t *testing.T) {
	registry := NewRegistry(nil)

	newConfig := &MockConfig{
		Mode:        MockModeChaos,
		FailureRate: 0.5,
	}

	registry.SetGlobalConfig(newConfig)

	globalConfig := registry.GetGlobalConfig()
	if globalConfig.Mode != MockModeChaos {
		t.Errorf("expected mode chaos, got %s", globalConfig.Mode)
	}
	if globalConfig.FailureRate != 0.5 {
		t.Errorf("expected failure rate 0.5, got %f", globalConfig.FailureRate)
	}
}

func TestRegistryClear(t *testing.T) {
	registry := NewRegistry(nil)

	registry.RegisterMock("terraform", "Infrastructure", nil)
	registry.RegisterMock("argocd", "Orchestrator", nil)

	if registry.Count() != 2 {
		t.Errorf("expected 2 mocks, got %d", registry.Count())
	}

	registry.Clear()

	if registry.Count() != 0 {
		t.Errorf("expected 0 mocks after clear, got %d", registry.Count())
	}
}

func TestRegistryRemove(t *testing.T) {
	registry := NewRegistry(nil)

	registry.RegisterMock("terraform", "Infrastructure", nil)
	registry.RegisterMock("argocd", "Orchestrator", nil)

	registry.Remove("Infrastructure", "terraform")

	if registry.Count() != 1 {
		t.Errorf("expected 1 mock after remove, got %d", registry.Count())
	}

	_, ok := registry.Get("Infrastructure", "terraform")
	if ok {
		t.Error("expected terraform to be removed")
	}

	// Remove non-existent should not panic
	registry.Remove("Nonexistent", "tool")
}

func TestRegistryCount(t *testing.T) {
	registry := NewRegistry(nil)

	if registry.Count() != 0 {
		t.Errorf("expected count 0, got %d", registry.Count())
	}

	registry.RegisterMock("terraform", "Infrastructure", nil)
	if registry.Count() != 1 {
		t.Errorf("expected count 1, got %d", registry.Count())
	}

	registry.RegisterMock("argocd", "Orchestrator", nil)
	if registry.Count() != 2 {
		t.Errorf("expected count 2, got %d", registry.Count())
	}
}

func TestRegistryRegisterBuiltinMocks(t *testing.T) {
	registry := NewRegistry(nil)

	registry.RegisterBuiltinMocks()

	// Check some expected builtins
	expectedMocks := []struct {
		kind string
		name string
	}{
		{"Infrastructure", "terraform"},
		{"Infrastructure", "aws"},
		{"Infrastructure", "gcp"},
		{"Infrastructure", "azure"},
		{"Orchestrator", "argocd"},
		{"Orchestrator", "flux"},
		{"Observability", "prometheus"},
		{"Observability", "grafana"},
		{"DevEx", "backstage"},
		{"Security", "vault"},
		{"Pipeline", "jenkins"},
		{"Pipeline", "tekton"},
	}

	for _, expected := range expectedMocks {
		mock, ok := registry.Get(expected.kind, expected.name)
		if !ok {
			t.Errorf("expected builtin mock for %s:%s", expected.kind, expected.name)
		}
		if mock == nil {
			t.Errorf("expected non-nil mock for %s:%s", expected.kind, expected.name)
		}
	}
}

func TestDefaultRegistry(t *testing.T) {
	if DefaultRegistry == nil {
		t.Fatal("expected non-nil DefaultRegistry")
	}

	// Should have builtin mocks
	mock, ok := DefaultRegistry.Get("Infrastructure", "terraform")
	if !ok || mock == nil {
		t.Error("expected DefaultRegistry to have builtin mocks")
	}
}

func TestRegistryConcurrentAccess(t *testing.T) {
	registry := NewRegistry(nil)

	var wg sync.WaitGroup
	iterations := 100

	// Concurrent registrations
	wg.Add(iterations)
	for i := 0; i < iterations; i++ {
		go func(id int) {
			defer wg.Done()
			registry.RegisterMock("tool-"+string(rune('A'+id%26)), "Infrastructure", nil)
		}(i)
	}

	// Concurrent reads
	wg.Add(iterations)
	for i := 0; i < iterations; i++ {
		go func() {
			defer wg.Done()
			registry.List()
			registry.Count()
		}()
	}

	wg.Wait()
}

func TestRegistryConcurrentGetOrCreate(t *testing.T) {
	registry := NewRegistry(nil)

	var wg sync.WaitGroup
	iterations := 50

	// Multiple goroutines trying to get/create the same mock
	results := make([]*MockPlugin, iterations)

	wg.Add(iterations)
	for i := 0; i < iterations; i++ {
		go func(idx int) {
			defer wg.Done()
			results[idx] = registry.GetOrCreate("Infrastructure", "terraform")
		}(i)
	}

	wg.Wait()

	// All should return the same instance
	first := results[0]
	for i := 1; i < iterations; i++ {
		if results[i] != first {
			t.Error("expected all GetOrCreate calls to return same instance")
			break
		}
	}
}

func TestRegistryListReturnsCopy(t *testing.T) {
	registry := NewRegistry(nil)

	registry.RegisterMock("terraform", "Infrastructure", nil)

	list := registry.List()
	delete(list, "Infrastructure:terraform")

	// Original should be unchanged
	if registry.Count() != 1 {
		t.Error("expected List to return a copy")
	}
}

func TestRegistryUsesGlobalConfigForNewMocks(t *testing.T) {
	config := &MockConfig{
		Mode: MockModeChaos,
	}
	registry := NewRegistry(config)

	registry.RegisterMock("terraform", "Infrastructure", nil)

	mock, _ := registry.Get("Infrastructure", "terraform")
	if mock.config.Mode != MockModeChaos {
		t.Error("expected new mock to use global config")
	}
}

func TestRegistryMockOverridesGlobalConfig(t *testing.T) {
	globalConfig := &MockConfig{Mode: MockModeRealistic}
	registry := NewRegistry(globalConfig)

	customConfig := &MockConfig{Mode: MockModeInstant}
	registry.RegisterMock("terraform", "Infrastructure", customConfig)

	mock, _ := registry.Get("Infrastructure", "terraform")
	if mock.config.Mode != MockModeInstant {
		t.Error("expected mock to use custom config over global")
	}
}

func TestRegistryKeyFormat(t *testing.T) {
	registry := NewRegistry(nil)

	registry.RegisterMock("my-tool", "My-Type", nil)

	list := registry.List()
	if _, ok := list["My-Type:my-tool"]; !ok {
		t.Error("expected key format to be 'Type:name'")
	}
}

func TestRegistryFunctionalWorkflow(t *testing.T) {
	// Test a realistic usage workflow
	config := &MockConfig{
		Mode:         MockModeInstant,
		DefaultDelay: 10 * time.Millisecond,
	}
	registry := NewRegistry(config)

	// Register mocks
	registry.RegisterMock("terraform", "Infrastructure", nil)
	registry.RegisterMock("argocd", "Orchestrator", nil)

	// Get mock and use it
	infraMock, ok := registry.Get("Infrastructure", "terraform")
	if !ok {
		t.Fatal("expected to get infrastructure mock")
	}

	spec := map[string]interface{}{
		"name":     "test-platform",
		"provider": "aws",
	}

	result, err := infraMock.Apply(spec)
	if err != nil {
		t.Fatalf("Apply failed: %v", err)
	}
	if result.Status != "success" {
		t.Errorf("expected success status, got %s", result.Status)
	}
	if result.Outputs["cluster_endpoint"] == "" {
		t.Error("expected cluster_endpoint in outputs")
	}

	// Get orchestrator mock
	orchMock, ok := registry.Get("Orchestrator", "argocd")
	if !ok {
		t.Fatal("expected to get orchestrator mock")
	}

	result, err = orchMock.Apply(spec)
	if err != nil {
		t.Fatalf("Apply failed: %v", err)
	}
	if result.Outputs["argocd_url"] == "" {
		t.Error("expected argocd_url in outputs")
	}
}
