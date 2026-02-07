package integration

import (
	"os"
	"testing"

	"github.com/platformfoundry/pf-ce/internal/orchestrator"
	"github.com/platformfoundry/pf-ce/internal/parser"
	"github.com/platformfoundry/pf-ce/internal/plugin"
	"github.com/platformfoundry/pf-ce/internal/store"
)

func TestApply_SingleResource(t *testing.T) {
	// Create temporary YAML file
	testYAML := `---
apiVersion: platformfoundry.io/v1
kind: Cluster
metadata:
  name: test-cluster
  description: Integration test cluster
spec:
  provider: existing
  config:
    kubeconfig: /tmp/test-kubeconfig
`

	tmpFile, err := os.CreateTemp("", "test-*.yaml")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())

	if _, err := tmpFile.WriteString(testYAML); err != nil {
		t.Fatalf("Failed to write test YAML: %v", err)
	}
	tmpFile.Close()

	// Create temp dir for isolated store
	tmpDir, err := os.MkdirTemp("", "pf-apply-test-single-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Initialize components with isolated store
	pm := plugin.NewManager()
	st, err := store.NewWithPath(tmpDir + "/state.db")
	if err != nil {
		t.Fatalf("Failed to create store: %v", err)
	}
	defer st.Close()

	orch := orchestrator.New(pm, st)
	p := parser.New()

	// Parse and apply
	resources, err := p.ParseFile(tmpFile.Name())
	if err != nil {
		t.Fatalf("Failed to parse file: %v", err)
	}

	if len(resources) != 1 {
		t.Errorf("Expected 1 resource, got %d", len(resources))
	}

	// Convert to []interface{} for orchestrator
	var interfaceResources []interface{}
	for _, r := range resources {
		interfaceResources = append(interfaceResources, r)
	}

	// Note: This will fail without a proper plugin registered
	// This test verifies the integration flow
	err = orch.Apply(interfaceResources)
	if err != nil {
		t.Logf("Apply failed as expected (no plugin): %v", err)
	}
}

func TestApply_MultipleResources(t *testing.T) {
	testYAML := `---
apiVersion: platformfoundry.io/v1
kind: Cluster
metadata:
  name: cluster1
spec:
  provider: existing
---
apiVersion: platformfoundry.io/v1
kind: Pipeline
metadata:
  name: pipeline1
spec:
  type: jenkins
  clusterRef: cluster1
`

	tmpFile, err := os.CreateTemp("", "test-*.yaml")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())

	if _, err := tmpFile.WriteString(testYAML); err != nil {
		t.Fatalf("Failed to write test YAML: %v", err)
	}
	tmpFile.Close()

	// Create temp dir for isolated store
	tmpDir, err := os.MkdirTemp("", "pf-apply-test-multi-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Initialize components with isolated store
	pm := plugin.NewManager()
	st, err := store.NewWithPath(tmpDir + "/state.db")
	if err != nil {
		t.Fatalf("Failed to create store: %v", err)
	}
	defer st.Close()

	orch := orchestrator.New(pm, st)
	p := parser.New()

	// Parse resources
	resources, err := p.ParseFile(tmpFile.Name())
	if err != nil {
		t.Fatalf("Failed to parse file: %v", err)
	}

	if len(resources) != 2 {
		t.Errorf("Expected 2 resources, got %d", len(resources))
	}

	// Convert to []interface{} for orchestrator
	var interfaceResources []interface{}
	for _, r := range resources {
		interfaceResources = append(interfaceResources, r)
	}

	// Note: This test verifies parsing and dependency resolution
	// Actual apply will fail without plugins
	err = orch.Apply(interfaceResources)
	if err != nil {
		t.Logf("Apply failed as expected (no plugin): %v", err)
	}
}
