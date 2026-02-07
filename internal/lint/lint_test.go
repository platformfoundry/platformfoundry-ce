package lint

import (
	"strings"
	"testing"
)

func TestNew(t *testing.T) {
	l := New()
	if l == nil {
		t.Fatal("New() returned nil")
	}
	if len(l.rules) == 0 {
		t.Error("New linter should have default rules")
	}
}

func TestLintValidConfig(t *testing.T) {
	l := New()

	config := `
apiVersion: platform.io/v1
kind: Deployment
metadata:
  name: my-app
  namespace: production
  labels:
    app: my-app
spec:
  replicas: 3
  resources:
    limits:
      cpu: "1"
      memory: "512Mi"
  securityContext:
    runAsNonRoot: true
`

	result, err := l.Lint([]byte(config), "test.yaml")
	if err != nil {
		t.Fatalf("Lint failed: %v", err)
	}

	if result.Summary.Errors > 0 {
		t.Errorf("Expected no errors, got %d", result.Summary.Errors)
		for _, issue := range result.Issues {
			if issue.Severity == SeverityError {
				t.Logf("Error: %s - %s", issue.Rule, issue.Message)
			}
		}
	}
}

func TestLintMissingAPIVersion(t *testing.T) {
	l := New()

	config := `
kind: Deployment
metadata:
  name: my-app
`

	result, err := l.Lint([]byte(config), "test.yaml")
	if err != nil {
		t.Fatalf("Lint failed: %v", err)
	}

	found := false
	for _, issue := range result.Issues {
		if issue.Rule == "require-api-version" {
			found = true
			break
		}
	}

	if !found {
		t.Error("Expected require-api-version error")
	}
}

func TestLintMissingKind(t *testing.T) {
	l := New()

	config := `
apiVersion: platform.io/v1
metadata:
  name: my-app
`

	result, err := l.Lint([]byte(config), "test.yaml")
	if err != nil {
		t.Fatalf("Lint failed: %v", err)
	}

	found := false
	for _, issue := range result.Issues {
		if issue.Rule == "require-kind" {
			found = true
			break
		}
	}

	if !found {
		t.Error("Expected require-kind error")
	}
}

func TestLintMissingMetadata(t *testing.T) {
	l := New()

	config := `
apiVersion: platform.io/v1
kind: Deployment
`

	result, err := l.Lint([]byte(config), "test.yaml")
	if err != nil {
		t.Fatalf("Lint failed: %v", err)
	}

	found := false
	for _, issue := range result.Issues {
		if issue.Rule == "require-metadata" {
			found = true
			break
		}
	}

	if !found {
		t.Error("Expected require-metadata error")
	}
}

func TestLintInvalidResourceName(t *testing.T) {
	l := New()

	config := `
apiVersion: platform.io/v1
kind: Deployment
metadata:
  name: My_Invalid-Name!
`

	result, err := l.Lint([]byte(config), "test.yaml")
	if err != nil {
		t.Fatalf("Lint failed: %v", err)
	}

	found := false
	for _, issue := range result.Issues {
		if issue.Rule == "valid-resource-name" {
			found = true
			break
		}
	}

	if !found {
		t.Error("Expected valid-resource-name warning")
	}
}

func TestLintLatestImageTag(t *testing.T) {
	l := New()

	config := `
apiVersion: platform.io/v1
kind: Deployment
metadata:
  name: my-app
spec:
  image: nginx:latest
`

	result, err := l.Lint([]byte(config), "test.yaml")
	if err != nil {
		t.Fatalf("Lint failed: %v", err)
	}

	found := false
	for _, issue := range result.Issues {
		if issue.Rule == "image-tag" {
			found = true
			break
		}
	}

	if !found {
		t.Error("Expected image-tag warning for :latest")
	}
}

func TestLintNoImageTag(t *testing.T) {
	l := New()

	config := `
apiVersion: platform.io/v1
kind: Deployment
metadata:
  name: my-app
spec:
  image: nginx
`

	result, err := l.Lint([]byte(config), "test.yaml")
	if err != nil {
		t.Fatalf("Lint failed: %v", err)
	}

	found := false
	for _, issue := range result.Issues {
		if issue.Rule == "image-tag" {
			found = true
			break
		}
	}

	if !found {
		t.Error("Expected image-tag warning for missing tag")
	}
}

func TestLintHardcodedSecrets(t *testing.T) {
	l := New()

	config := `
apiVersion: platform.io/v1
kind: Deployment
metadata:
  name: my-app
spec:
  database_password: "super-secret-123"
`

	result, err := l.Lint([]byte(config), "test.yaml")
	if err != nil {
		t.Fatalf("Lint failed: %v", err)
	}

	found := false
	for _, issue := range result.Issues {
		if issue.Rule == "hardcoded-secrets" {
			found = true
			break
		}
	}

	if !found {
		t.Error("Expected hardcoded-secrets error")
	}
}

func TestLintSecretReference(t *testing.T) {
	l := New()

	// Environment variable reference should be OK
	config := `
apiVersion: platform.io/v1
kind: Deployment
metadata:
  name: my-app
spec:
  database_password: "${DB_PASSWORD}"
`

	result, err := l.Lint([]byte(config), "test.yaml")
	if err != nil {
		t.Fatalf("Lint failed: %v", err)
	}

	for _, issue := range result.Issues {
		if issue.Rule == "hardcoded-secrets" {
			t.Error("Should not flag environment variable references as secrets")
		}
	}
}

func TestLintMultipleDocuments(t *testing.T) {
	l := New()

	config := `
apiVersion: platform.io/v1
kind: Deployment
metadata:
  name: app-1
---
apiVersion: platform.io/v1
kind: Service
metadata:
  name: svc-1
`

	result, err := l.LintMultiple([]byte(config), "multi.yaml")
	if err != nil {
		t.Fatalf("LintMultiple failed: %v", err)
	}

	// Both documents should be linted
	if result.File != "multi.yaml" {
		t.Error("File name not set correctly")
	}
}

func TestLintInvalidYAML(t *testing.T) {
	l := New()

	config := `
this is: not: valid: yaml: [
`

	result, err := l.Lint([]byte(config), "invalid.yaml")
	if err != nil {
		t.Fatalf("Lint should not return error for invalid YAML: %v", err)
	}

	found := false
	for _, issue := range result.Issues {
		if issue.Rule == "yaml-parse" {
			found = true
			break
		}
	}

	if !found {
		t.Error("Expected yaml-parse error for invalid YAML")
	}
}

func TestResultFormat(t *testing.T) {
	result := &Result{
		File: "test.yaml",
		Issues: []Issue{
			{
				Severity:   SeverityError,
				Rule:       "test-rule",
				Message:    "Test error",
				Suggestion: "Fix it",
			},
			{
				Severity: SeverityWarning,
				Rule:     "warn-rule",
				Message:  "Test warning",
			},
		},
	}
	result.Summary.Errors = 1
	result.Summary.Warnings = 1

	output := result.Format()

	expectedStrings := []string{
		"test.yaml",
		"[ERR]",
		"[WARN]",
		"test-rule",
		"Test error",
		"Fix it",
		"1 errors",
		"1 warnings",
	}

	for _, expected := range expectedStrings {
		if !strings.Contains(output, expected) {
			t.Errorf("Expected output to contain '%s'", expected)
		}
	}
}

func TestNoIssuesFormat(t *testing.T) {
	result := &Result{
		File:   "clean.yaml",
		Issues: []Issue{},
	}

	output := result.Format()

	if !strings.Contains(output, "No issues found") {
		t.Error("Expected 'No issues found' message")
	}
}

func TestRegisterRule(t *testing.T) {
	l := New()
	initialCount := len(l.rules)

	customRule := Rule{
		ID:          "custom-rule",
		Name:        "Custom Rule",
		Description: "A custom test rule",
		Severity:    SeverityInfo,
		Check: func(config map[string]interface{}, file string) []Issue {
			return nil
		},
	}

	l.RegisterRule(customRule)

	if len(l.rules) != initialCount+1 {
		t.Error("Rule was not registered")
	}
}

func TestLintReplicaCount(t *testing.T) {
	l := New()

	config := `
apiVersion: platform.io/v1
kind: Deployment
metadata:
  name: my-app
spec:
  replicas: 1
`

	result, err := l.Lint([]byte(config), "test.yaml")
	if err != nil {
		t.Fatalf("Lint failed: %v", err)
	}

	found := false
	for _, issue := range result.Issues {
		if issue.Rule == "replica-count" {
			found = true
			break
		}
	}

	if !found {
		t.Error("Expected replica-count info for single replica")
	}
}
