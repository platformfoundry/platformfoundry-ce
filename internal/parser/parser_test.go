package parser

import (
	"os"
	"testing"
)

func TestNewParser(t *testing.T) {
	p := New()
	if p == nil {
		t.Fatal("New() returned nil")
	}
}

func TestParseFile_ValidYAML(t *testing.T) {
	// Create temporary test file
	testYAML := `---
apiVersion: platformfoundry.io/v1
kind: Cluster
metadata:
  name: test-cluster
  description: Test cluster
spec:
  provider: existing
  config:
    kubeconfig: /path/to/kubeconfig
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

	// Parse file
	p := New()
	resources, err := p.ParseFile(tmpFile.Name())

	if err != nil {
		t.Fatalf("ParseFile failed: %v", err)
	}

	if len(resources) != 1 {
		t.Errorf("Expected 1 resource, got %d", len(resources))
	}
}

func TestParseFile_MultiDocument(t *testing.T) {
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

	p := New()
	resources, err := p.ParseFile(tmpFile.Name())

	if err != nil {
		t.Fatalf("ParseFile failed: %v", err)
	}

	if len(resources) != 2 {
		t.Errorf("Expected 2 resources, got %d", len(resources))
	}
}

func TestParseFile_NonExistent(t *testing.T) {
	p := New()
	_, err := p.ParseFile("/nonexistent/file.yaml")

	if err == nil {
		t.Error("Expected error for non-existent file, got nil")
	}
}

func TestParseFile_InvalidYAML(t *testing.T) {
	testYAML := `
invalid: yaml: content:
  - missing
    proper: indentation
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

	p := New()
	_, err = p.ParseFile(tmpFile.Name())

	if err == nil {
		t.Error("Expected error for invalid YAML, got nil")
	}
}

func TestParseString_Valid(t *testing.T) {
	testYAML := `
apiVersion: platformfoundry.io/v1
kind: Cluster
metadata:
  name: test
spec:
  provider: existing
`

	p := New()
	resources, err := p.ParseString(testYAML)

	if err != nil {
		t.Fatalf("ParseString failed: %v", err)
	}

	if len(resources) != 1 {
		t.Errorf("Expected 1 resource, got %d", len(resources))
	}
}

func TestParseString_Empty(t *testing.T) {
	p := New()
	resources, err := p.ParseString("")

	if err != nil {
		t.Fatalf("ParseString failed on empty string: %v", err)
	}

	if len(resources) != 0 {
		t.Errorf("Expected 0 resources for empty string, got %d", len(resources))
	}
}
