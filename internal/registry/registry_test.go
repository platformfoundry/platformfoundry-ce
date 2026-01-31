package registry

import (
	"sync"
	"testing"
)

func TestNewToolRegistry(t *testing.T) {
	registry := NewToolRegistry()

	if registry == nil {
		t.Fatal("expected non-nil registry")
	}

	// Should have built-in tools registered
	tools := registry.List()
	if len(tools) == 0 {
		t.Error("expected built-in tools to be registered")
	}
}

func TestToolRegistryRegister(t *testing.T) {
	registry := NewToolRegistry()

	tool := &RegisteredTool{
		Name:        "custom-tool",
		DisplayName: "Custom Tool",
		Description: "A custom test tool",
		Category:    "custom",
	}

	err := registry.Register(tool)
	if err != nil {
		t.Errorf("Register failed: %v", err)
	}

	// Verify tool is registered
	retrieved, ok := registry.Get("custom-tool")
	if !ok {
		t.Error("expected to find registered tool")
	}
	if retrieved.DisplayName != "Custom Tool" {
		t.Errorf("expected DisplayName 'Custom Tool', got '%s'", retrieved.DisplayName)
	}

	// Category should be updated
	categories := registry.ListCategories()
	found := false
	for _, cat := range categories {
		if cat == "custom" {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected 'custom' category to be added")
	}
}

func TestToolRegistryRegisterDuplicate(t *testing.T) {
	registry := NewToolRegistry()

	tool := &RegisteredTool{
		Name:     "test-tool",
		Category: "test",
	}

	err := registry.Register(tool)
	if err != nil {
		t.Fatalf("First register failed: %v", err)
	}

	// Registering again should fail
	err = registry.Register(tool)
	if err == nil {
		t.Error("expected error when registering duplicate tool")
	}
}

func TestToolRegistryGet(t *testing.T) {
	registry := NewToolRegistry()

	// Get existing built-in tool
	tool, ok := registry.Get("argocd")
	if !ok {
		t.Error("expected to find argocd")
	}
	if tool.DisplayName != "Argo CD" {
		t.Errorf("expected DisplayName 'Argo CD', got '%s'", tool.DisplayName)
	}

	// Get non-existent tool
	_, ok = registry.Get("nonexistent")
	if ok {
		t.Error("expected not to find nonexistent tool")
	}
}

func TestToolRegistryGetByCategory(t *testing.T) {
	registry := NewToolRegistry()

	// Get orchestration tools
	tools := registry.GetByCategory("orchestration")
	if len(tools) == 0 {
		t.Error("expected orchestration tools")
	}

	// Verify argocd and flux are in orchestration
	foundArgocd := false
	foundFlux := false
	for _, tool := range tools {
		if tool.Name == "argocd" {
			foundArgocd = true
		}
		if tool.Name == "flux" {
			foundFlux = true
		}
	}
	if !foundArgocd {
		t.Error("expected argocd in orchestration category")
	}
	if !foundFlux {
		t.Error("expected flux in orchestration category")
	}

	// Get empty category
	tools = registry.GetByCategory("nonexistent")
	if len(tools) != 0 {
		t.Errorf("expected 0 tools for nonexistent category, got %d", len(tools))
	}
}

func TestToolRegistryGetScaffold(t *testing.T) {
	registry := NewToolRegistry()

	// Get existing scaffold
	scaffold, err := registry.GetScaffold("argocd", "basic")
	if err != nil {
		t.Errorf("GetScaffold failed: %v", err)
	}
	if scaffold == nil {
		t.Fatal("expected non-nil scaffold")
	}
	if scaffold.Name != "basic" {
		t.Errorf("expected scaffold name 'basic', got '%s'", scaffold.Name)
	}

	// Get scaffold for non-existent tool
	_, err = registry.GetScaffold("nonexistent", "basic")
	if err == nil {
		t.Error("expected error for nonexistent tool")
	}

	// Get non-existent scaffold
	_, err = registry.GetScaffold("argocd", "nonexistent")
	if err == nil {
		t.Error("expected error for nonexistent scaffold")
	}
}

func TestToolRegistryList(t *testing.T) {
	registry := NewToolRegistry()

	tools := registry.List()
	if len(tools) == 0 {
		t.Error("expected tools in list")
	}

	// Verify some expected tools
	expectedTools := []string{"argocd", "flux", "prometheus", "grafana", "backstage", "vault"}
	for _, expected := range expectedTools {
		found := false
		for _, tool := range tools {
			if tool.Name == expected {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("expected tool '%s' in list", expected)
		}
	}
}

func TestToolRegistryListCategories(t *testing.T) {
	registry := NewToolRegistry()

	categories := registry.ListCategories()
	if len(categories) == 0 {
		t.Error("expected categories")
	}

	// Verify expected categories
	expectedCategories := []string{"orchestration", "observability", "devex", "security", "infrastructure"}
	for _, expected := range expectedCategories {
		found := false
		for _, cat := range categories {
			if cat == expected {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("expected category '%s'", expected)
		}
	}
}

func TestToolRegistryCheckCompatibility(t *testing.T) {
	registry := NewToolRegistry()

	// Compatible tools
	compatible, notes := registry.CheckCompatibility("prometheus", "grafana")
	if !compatible {
		t.Error("expected prometheus and grafana to be compatible")
	}
	if notes == "" {
		t.Error("expected compatibility notes")
	}

	// Incompatible tools
	compatible, _ = registry.CheckCompatibility("argocd", "flux")
	if compatible {
		t.Error("expected argocd and flux to be incompatible")
	}
}

func TestToolRegistryValidateToolSet(t *testing.T) {
	registry := NewToolRegistry()

	// Valid tool set
	issues := registry.ValidateToolSet([]string{"argocd", "prometheus", "grafana"})
	if len(issues) != 0 {
		t.Errorf("expected no issues, got %d", len(issues))
	}

	// Invalid tool set
	issues = registry.ValidateToolSet([]string{"argocd", "flux", "prometheus"})
	if len(issues) == 0 {
		t.Error("expected issues for argocd + flux")
	}
}

func TestToolRegistryGetCompatibility(t *testing.T) {
	registry := NewToolRegistry()

	matrix := registry.GetCompatibility()
	if matrix == nil {
		t.Error("expected non-nil compatibility matrix")
	}
}

func TestToolRegistryConcurrentAccess(t *testing.T) {
	registry := NewToolRegistry()

	var wg sync.WaitGroup
	iterations := 100

	// Concurrent registrations
	wg.Add(iterations)
	for i := 0; i < iterations; i++ {
		go func(id int) {
			defer wg.Done()
			tool := &RegisteredTool{
				Name:     "tool-" + string(rune('A'+id%26)),
				Category: "test",
			}
			registry.Register(tool)
		}(i)
	}

	// Concurrent reads
	wg.Add(iterations)
	for i := 0; i < iterations; i++ {
		go func() {
			defer wg.Done()
			registry.List()
			registry.ListCategories()
			registry.Get("argocd")
		}()
	}

	wg.Wait()
}

func TestToolVersions(t *testing.T) {
	registry := NewToolRegistry()

	tool, _ := registry.Get("argocd")
	if len(tool.Versions) == 0 {
		t.Error("expected tool to have versions")
	}

	// Verify latest version
	if tool.LatestVersion == "" {
		t.Error("expected latest version to be set")
	}

	// Check version is in list
	found := false
	for _, v := range tool.Versions {
		if v.Version == tool.LatestVersion {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected latest version to be in versions list")
	}
}

func TestScaffoldParameters(t *testing.T) {
	registry := NewToolRegistry()

	scaffold, _ := registry.GetScaffold("argocd", "basic")

	if len(scaffold.Parameters) == 0 {
		t.Error("expected scaffold to have parameters")
	}

	// Check for common parameters
	foundNamespace := false
	for _, param := range scaffold.Parameters {
		if param.Name == "namespace" {
			foundNamespace = true
			if param.Default != "argocd" {
				t.Errorf("expected namespace default 'argocd', got '%v'", param.Default)
			}
		}
	}
	if !foundNamespace {
		t.Error("expected namespace parameter")
	}
}

func TestToolDocumentation(t *testing.T) {
	registry := NewToolRegistry()

	tool, _ := registry.Get("argocd")

	if tool.Documentation.Overview == "" {
		t.Error("expected overview in documentation")
	}
	if tool.Documentation.LearnMore == "" {
		t.Error("expected learn more URL in documentation")
	}
}

func TestToolTags(t *testing.T) {
	registry := NewToolRegistry()

	tool, _ := registry.Get("argocd")

	if len(tool.Tags) == 0 {
		t.Error("expected tool to have tags")
	}

	// Check for expected tags
	foundGitops := false
	for _, tag := range tool.Tags {
		if tag == "gitops" {
			foundGitops = true
			break
		}
	}
	if !foundGitops {
		t.Error("expected 'gitops' tag for argocd")
	}
}

func TestToolRequirements(t *testing.T) {
	registry := NewToolRegistry()

	tool, _ := registry.Get("argocd")

	if len(tool.Requirements) == 0 {
		t.Error("expected tool to have requirements")
	}

	// Check kubernetes requirement
	foundKube := false
	for _, req := range tool.Requirements {
		if req.Type == "kubernetes" {
			foundKube = true
			if req.MinVersion == "" {
				t.Error("expected min kubernetes version")
			}
		}
	}
	if !foundKube {
		t.Error("expected kubernetes requirement")
	}
}

// Compatibility Matrix Tests

func TestNewCompatibilityMatrix(t *testing.T) {
	matrix := NewCompatibilityMatrix()

	if matrix == nil {
		t.Fatal("expected non-nil matrix")
	}

	rules := matrix.GetRules()
	if len(rules) == 0 {
		t.Error("expected built-in rules to be loaded")
	}
}

func TestCompatibilityMatrixAddRule(t *testing.T) {
	matrix := NewCompatibilityMatrix()
	initialCount := len(matrix.GetRules())

	rule := CompatibilityRule{
		ToolA:      "custom-a",
		ToolB:      "custom-b",
		Compatible: true,
		Notes:      "Test rule",
	}

	matrix.AddRule(rule)

	if len(matrix.GetRules()) != initialCount+1 {
		t.Error("expected rule to be added")
	}
}

func TestCompatibilityMatrixCheck(t *testing.T) {
	matrix := NewCompatibilityMatrix()

	// Check known incompatible pair
	compatible, notes := matrix.Check("argocd", "flux")
	if compatible {
		t.Error("expected argocd and flux to be incompatible")
	}
	if notes == "" {
		t.Error("expected notes for incompatibility")
	}

	// Check in reverse order (should be same)
	compatible, _ = matrix.Check("flux", "argocd")
	if compatible {
		t.Error("expected flux and argocd to be incompatible (reverse)")
	}

	// Check known compatible pair
	compatible, notes = matrix.Check("prometheus", "grafana")
	if !compatible {
		t.Error("expected prometheus and grafana to be compatible")
	}
	if notes == "" {
		t.Error("expected notes for compatibility")
	}

	// Check unknown pair (should default to compatible)
	compatible, notes = matrix.Check("unknown-a", "unknown-b")
	if !compatible {
		t.Error("expected unknown pair to default to compatible")
	}
	if notes != "" {
		t.Error("expected no notes for unknown pair")
	}
}

func TestCompatibilityMatrixValidateSet(t *testing.T) {
	matrix := NewCompatibilityMatrix()

	// Valid set
	issues := matrix.ValidateSet([]string{"argocd", "prometheus", "grafana", "backstage"})
	if len(issues) != 0 {
		t.Errorf("expected no issues for valid set, got %d", len(issues))
	}

	// Set with incompatible tools
	issues = matrix.ValidateSet([]string{"argocd", "flux", "prometheus"})
	if len(issues) != 1 {
		t.Errorf("expected 1 issue for argocd+flux, got %d", len(issues))
	}
	if issues[0].Severity != "error" {
		t.Errorf("expected severity 'error', got '%s'", issues[0].Severity)
	}

	// Multiple incompatibilities
	issues = matrix.ValidateSet([]string{"argocd", "flux", "istio", "linkerd"})
	if len(issues) < 2 {
		t.Errorf("expected at least 2 issues, got %d", len(issues))
	}
}

func TestCompatibilityMatrixGetRulesFor(t *testing.T) {
	matrix := NewCompatibilityMatrix()

	rules := matrix.GetRulesFor("argocd")
	if len(rules) == 0 {
		t.Error("expected rules involving argocd")
	}

	// Verify all returned rules involve argocd
	for _, rule := range rules {
		if rule.ToolA != "argocd" && rule.ToolB != "argocd" {
			t.Error("expected all returned rules to involve argocd")
		}
	}
}

func TestCompatibilityMatrixGetIncompatibleTools(t *testing.T) {
	matrix := NewCompatibilityMatrix()

	incompatible := matrix.GetIncompatibleTools("argocd")
	if len(incompatible) == 0 {
		t.Error("expected incompatible tools for argocd")
	}

	// Verify flux is in the list
	found := false
	for _, tool := range incompatible {
		if tool == "flux" {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected flux to be incompatible with argocd")
	}
}

func TestCompatibilityMatrixGetCompatibleTools(t *testing.T) {
	matrix := NewCompatibilityMatrix()

	compatible := matrix.GetCompatibleTools("prometheus")
	if len(compatible) == 0 {
		t.Error("expected compatible tools for prometheus")
	}

	// Verify grafana is in the list
	found := false
	for _, tool := range compatible {
		if tool == "grafana" {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected grafana to be compatible with prometheus")
	}
}

func TestCompatibilityMatrixSuggestAlternatives(t *testing.T) {
	matrix := NewCompatibilityMatrix()

	// For incompatible pair, should suggest alternatives
	alternatives := matrix.SuggestAlternatives("argocd", "flux")
	if len(alternatives) == 0 {
		t.Error("expected alternatives when tools are incompatible")
	}

	// For compatible pair, should return nil
	alternatives = matrix.SuggestAlternatives("prometheus", "grafana")
	if alternatives != nil {
		t.Error("expected no alternatives for compatible pair")
	}
}

func TestCompatibilityMatrixConcurrentAccess(t *testing.T) {
	matrix := NewCompatibilityMatrix()

	var wg sync.WaitGroup
	iterations := 100

	// Concurrent reads
	wg.Add(iterations)
	for i := 0; i < iterations; i++ {
		go func() {
			defer wg.Done()
			matrix.Check("argocd", "flux")
			matrix.GetRules()
			matrix.GetRulesFor("prometheus")
		}()
	}

	// Concurrent writes
	wg.Add(iterations)
	for i := 0; i < iterations; i++ {
		go func(id int) {
			defer wg.Done()
			matrix.AddRule(CompatibilityRule{
				ToolA:      "tool-" + string(rune('A'+id%26)),
				ToolB:      "tool-" + string(rune('a'+id%26)),
				Compatible: true,
			})
		}(i)
	}

	wg.Wait()
}

func TestBuiltinToolsHaveScaffolds(t *testing.T) {
	registry := NewToolRegistry()

	tools := registry.List()
	for _, tool := range tools {
		if len(tool.Scaffolds) == 0 {
			t.Errorf("tool '%s' has no scaffolds", tool.Name)
		}

		// Verify each scaffold has required fields
		for _, scaffold := range tool.Scaffolds {
			if scaffold.Name == "" {
				t.Errorf("scaffold in tool '%s' has no name", tool.Name)
			}
			if scaffold.Description == "" {
				t.Errorf("scaffold '%s' in tool '%s' has no description", scaffold.Name, tool.Name)
			}
		}
	}
}

func TestBuiltinCompatibilityRulesAreComplete(t *testing.T) {
	matrix := NewCompatibilityMatrix()

	rules := matrix.GetRules()
	for _, rule := range rules {
		if rule.ToolA == "" {
			t.Error("rule has empty ToolA")
		}
		if rule.ToolB == "" {
			t.Error("rule has empty ToolB")
		}
		if rule.Notes == "" {
			t.Errorf("rule for %s <-> %s has no notes", rule.ToolA, rule.ToolB)
		}
	}
}
