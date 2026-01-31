package scaffold

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestNewGenerator(t *testing.T) {
	gen := NewGenerator()

	if gen == nil {
		t.Fatal("expected non-nil generator")
	}
	if len(gen.templates) == 0 {
		t.Error("expected templates to be loaded")
	}

	// Check all expected templates are loaded
	expectedTemplates := []string{"platform", "infrastructure", "orchestrator", "observability", "devex", "security"}
	for _, name := range expectedTemplates {
		if _, ok := gen.templates[name]; !ok {
			t.Errorf("expected template '%s' to be loaded", name)
		}
	}
}

func TestGenerateUnknownType(t *testing.T) {
	gen := NewGenerator()

	config := ScaffoldConfig{
		Type: ScaffoldType("unknown"),
		Name: "test",
	}

	_, err := gen.Generate(config)
	if err == nil {
		t.Error("expected error for unknown scaffold type")
	}
}

func TestGeneratePlatformDryRun(t *testing.T) {
	gen := NewGenerator()

	config := ScaffoldConfig{
		Type:          ScaffoldPlatform,
		Name:          "test-platform",
		OutputDir:     t.TempDir(),
		CloudProvider: "aws",
		Environment:   "dev",
		DryRun:        true,
	}

	result, err := gen.Generate(config)
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}
	if result == nil {
		t.Fatal("expected non-nil result")
	}
	if len(result.Files) != 1 {
		t.Errorf("expected 1 file, got %d", len(result.Files))
	}

	// In dry run, file should not be created
	file := result.Files[0]
	if file.Created {
		t.Error("expected file not to be created in dry run mode")
	}
	if file.Content == "" {
		t.Error("expected content to be generated")
	}

	// Verify content contains expected values
	if !strings.Contains(file.Content, "test-platform") {
		t.Error("expected content to contain platform name")
	}
}

func TestGeneratePlatformActual(t *testing.T) {
	gen := NewGenerator()
	outputDir := t.TempDir()

	config := ScaffoldConfig{
		Type:          ScaffoldPlatform,
		Name:          "my-platform",
		OutputDir:     outputDir,
		CloudProvider: "aws",
		Environment:   "dev",
		DryRun:        false,
	}

	result, err := gen.Generate(config)
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	// File should be created
	file := result.Files[0]
	if !file.Created {
		t.Error("expected file to be created")
	}

	// Verify file exists
	if _, err := os.Stat(file.Path); os.IsNotExist(err) {
		t.Error("expected file to exist on disk")
	}
}

func TestGenerateInfrastructureAWS(t *testing.T) {
	gen := NewGenerator()

	config := ScaffoldConfig{
		Type:          ScaffoldInfrastructure,
		Name:          "test-infra",
		OutputDir:     t.TempDir(),
		CloudProvider: "aws",
		Environment:   "dev",
		DryRun:        true,
	}

	result, err := gen.Generate(config)
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	content := result.Files[0].Content

	// Verify AWS-specific content
	if !strings.Contains(content, "us-west-2") {
		t.Error("expected AWS region in content")
	}
	if !strings.Contains(content, "eks") {
		t.Error("expected EKS reference in content")
	}
	if !strings.Contains(content, "vpc") {
		t.Error("expected VPC configuration in content")
	}
}

func TestGenerateInfrastructureGCP(t *testing.T) {
	gen := NewGenerator()

	config := ScaffoldConfig{
		Type:          ScaffoldInfrastructure,
		Name:          "test-infra",
		OutputDir:     t.TempDir(),
		CloudProvider: "gcp",
		Environment:   "dev",
		DryRun:        true,
	}

	result, err := gen.Generate(config)
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	content := result.Files[0].Content

	// Verify GCP-specific content
	if !strings.Contains(content, "us-central1") {
		t.Error("expected GCP region in content")
	}
	if !strings.Contains(content, "gke") {
		t.Error("expected GKE reference in content")
	}
}

func TestGenerateInfrastructureAzure(t *testing.T) {
	gen := NewGenerator()

	config := ScaffoldConfig{
		Type:          ScaffoldInfrastructure,
		Name:          "test-infra",
		OutputDir:     t.TempDir(),
		CloudProvider: "azure",
		Environment:   "dev",
		DryRun:        true,
	}

	result, err := gen.Generate(config)
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	content := result.Files[0].Content

	// Verify Azure-specific content
	if !strings.Contains(content, "eastus") {
		t.Error("expected Azure region in content")
	}
	if !strings.Contains(content, "aks") {
		t.Error("expected AKS reference in content")
	}
}

func TestGenerateOrchestrator(t *testing.T) {
	gen := NewGenerator()

	config := ScaffoldConfig{
		Type:          ScaffoldOrchestrator,
		Name:          "test-orch",
		OutputDir:     t.TempDir(),
		CloudProvider: "aws",
		Environment:   "dev",
		DryRun:        true,
	}

	result, err := gen.Generate(config)
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	content := result.Files[0].Content

	// Verify ArgoCD content
	if !strings.Contains(content, "argocd") {
		t.Error("expected argocd in content")
	}
	if !strings.Contains(content, "clusterRef") {
		t.Error("expected cluster reference in content")
	}
}

func TestGenerateObservability(t *testing.T) {
	gen := NewGenerator()

	config := ScaffoldConfig{
		Type:          ScaffoldObservability,
		Name:          "test-obs",
		OutputDir:     t.TempDir(),
		CloudProvider: "aws",
		Environment:   "dev",
		DryRun:        true,
	}

	result, err := gen.Generate(config)
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	content := result.Files[0].Content

	// Verify observability content
	if !strings.Contains(content, "prometheus") {
		t.Error("expected prometheus in content")
	}
	if !strings.Contains(content, "grafana") {
		t.Error("expected grafana in content")
	}
}

func TestGenerateObservabilityProdEnvironment(t *testing.T) {
	gen := NewGenerator()

	config := ScaffoldConfig{
		Type:          ScaffoldObservability,
		Name:          "test-obs",
		OutputDir:     t.TempDir(),
		CloudProvider: "aws",
		Environment:   "prod",
		DryRun:        true,
	}

	result, err := gen.Generate(config)
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	content := result.Files[0].Content

	// Verify production-specific values
	if !strings.Contains(content, "30d") {
		t.Error("expected 30d retention for prod")
	}
	if !strings.Contains(content, "100Gi") {
		t.Error("expected 100Gi storage for prod")
	}
}

func TestGenerateDevEx(t *testing.T) {
	gen := NewGenerator()

	config := ScaffoldConfig{
		Type:          ScaffoldDevEx,
		Name:          "test-devex",
		OutputDir:     t.TempDir(),
		CloudProvider: "aws",
		Environment:   "dev",
		DryRun:        true,
	}

	result, err := gen.Generate(config)
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	content := result.Files[0].Content

	// Verify DevEx content
	if !strings.Contains(content, "backstage") {
		t.Error("expected backstage in content")
	}
	if !strings.Contains(content, "catalog") {
		t.Error("expected catalog reference in content")
	}
}

func TestGenerateSecurity(t *testing.T) {
	gen := NewGenerator()

	config := ScaffoldConfig{
		Type:          ScaffoldSecurity,
		Name:          "test-sec",
		OutputDir:     t.TempDir(),
		CloudProvider: "aws",
		Environment:   "dev",
		DryRun:        true,
	}

	result, err := gen.Generate(config)
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	content := result.Files[0].Content

	// Verify security content
	if !strings.Contains(content, "vault") {
		t.Error("expected vault in content")
	}
	if !strings.Contains(content, "externalSecrets") {
		t.Error("expected externalSecrets in content")
	}
}

func TestGenerateFullPlatform(t *testing.T) {
	gen := NewGenerator()
	outputDir := t.TempDir()

	config := ScaffoldConfig{
		Type:          ScaffoldFull,
		Name:          "full-platform",
		OutputDir:     outputDir,
		CloudProvider: "aws",
		Environment:   "dev",
		DryRun:        false,
	}

	result, err := gen.Generate(config)
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	// Should have multiple files
	if len(result.Files) < 5 {
		t.Errorf("expected at least 5 files for full platform, got %d", len(result.Files))
	}

	// Verify directories were created
	expectedDirs := []string{"infrastructure", "orchestrator", "observability", "devex"}
	for _, dir := range expectedDirs {
		path := filepath.Join(outputDir, dir)
		if _, err := os.Stat(path); os.IsNotExist(err) {
			t.Errorf("expected directory %s to exist", dir)
		}
	}

	// Verify next steps are provided
	if len(result.NextSteps) == 0 {
		t.Error("expected next steps to be provided")
	}
}

func TestGenerateFullPlatformMockMode(t *testing.T) {
	gen := NewGenerator()
	outputDir := t.TempDir()

	config := ScaffoldConfig{
		Type:          ScaffoldFull,
		Name:          "mock-platform",
		OutputDir:     outputDir,
		CloudProvider: "aws",
		Environment:   "dev",
		MockMode:      true,
		DryRun:        false,
	}

	result, err := gen.Generate(config)
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	// Check that mock mode instructions are in next steps
	foundMockInstruction := false
	for _, step := range result.NextSteps {
		if strings.Contains(step, "--mock") {
			foundMockInstruction = true
			break
		}
	}
	if !foundMockInstruction {
		t.Error("expected mock mode instructions in next steps")
	}

	// Check platform file contains mock indicator
	for _, file := range result.Files {
		if strings.Contains(file.Path, "platform.yaml") {
			if !strings.Contains(file.Content, "mockEnabled") {
				t.Error("expected mockEnabled in platform file for mock mode")
			}
		}
	}
}

func TestGenerateWithOverwrite(t *testing.T) {
	gen := NewGenerator()
	outputDir := t.TempDir()

	config := ScaffoldConfig{
		Type:          ScaffoldPlatform,
		Name:          "test-platform",
		OutputDir:     outputDir,
		CloudProvider: "aws",
		Environment:   "dev",
		Overwrite:     false,
	}

	// First generation
	_, err := gen.Generate(config)
	if err != nil {
		t.Fatalf("First generate failed: %v", err)
	}

	// Second generation without overwrite should fail
	_, err = gen.Generate(config)
	if err == nil {
		t.Error("expected error when file exists and overwrite is false")
	}

	// Third generation with overwrite should succeed
	config.Overwrite = true
	_, err = gen.Generate(config)
	if err != nil {
		t.Errorf("Generate with overwrite failed: %v", err)
	}
}

func TestBuildTemplateData(t *testing.T) {
	gen := NewGenerator()

	config := ScaffoldConfig{
		Name:          "test-platform",
		CloudProvider: "aws",
		Environment:   "prod",
		MockMode:      true,
	}

	data := gen.buildTemplateData(config)

	if data["Name"] != "test-platform" {
		t.Errorf("expected Name 'test-platform', got '%v'", data["Name"])
	}
	if data["CloudProvider"] != "aws" {
		t.Errorf("expected CloudProvider 'aws', got '%v'", data["CloudProvider"])
	}
	if data["Environment"] != "prod" {
		t.Errorf("expected Environment 'prod', got '%v'", data["Environment"])
	}
	if data["MockMode"] != true {
		t.Error("expected MockMode true")
	}
	if data["IsProd"] != true {
		t.Error("expected IsProd true for prod environment")
	}
}

func TestBuildTemplateDataNonProd(t *testing.T) {
	gen := NewGenerator()

	config := ScaffoldConfig{
		Environment: "dev",
	}

	data := gen.buildTemplateData(config)

	if data["IsProd"] != false {
		t.Error("expected IsProd false for dev environment")
	}
}

func TestBuildTemplateDataProduction(t *testing.T) {
	gen := NewGenerator()

	config := ScaffoldConfig{
		Environment: "production",
	}

	data := gen.buildTemplateData(config)

	if data["IsProd"] != true {
		t.Error("expected IsProd true for production environment")
	}
}

func TestExecuteTemplateNotFound(t *testing.T) {
	gen := NewGenerator()

	_, err := gen.executeTemplate("nonexistent", map[string]interface{}{})
	if err == nil {
		t.Error("expected error for nonexistent template")
	}
}

func TestWriteFileNoOverwrite(t *testing.T) {
	gen := NewGenerator()
	outputDir := t.TempDir()
	filePath := filepath.Join(outputDir, "test.yaml")

	// Write first time
	err := gen.writeFile(filePath, "content1", false)
	if err != nil {
		t.Fatalf("writeFile failed: %v", err)
	}

	// Write second time without overwrite should fail
	err = gen.writeFile(filePath, "content2", false)
	if err == nil {
		t.Error("expected error when file exists and overwrite is false")
	}

	// Write with overwrite should succeed
	err = gen.writeFile(filePath, "content2", true)
	if err != nil {
		t.Errorf("writeFile with overwrite failed: %v", err)
	}

	// Verify content was overwritten
	content, _ := os.ReadFile(filePath)
	if string(content) != "content2" {
		t.Error("expected content to be overwritten")
	}
}

func TestWriteFileCreatesDirectory(t *testing.T) {
	gen := NewGenerator()
	outputDir := t.TempDir()
	filePath := filepath.Join(outputDir, "nested", "dir", "test.yaml")

	err := gen.writeFile(filePath, "content", false)
	if err != nil {
		t.Fatalf("writeFile failed: %v", err)
	}

	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		t.Error("expected file to be created")
	}
}

// Test template helper functions

func TestIndent(t *testing.T) {
	result := indent(2, "line1\nline2")
	expected := "  line1\n  line2"
	if result != expected {
		t.Errorf("expected '%s', got '%s'", expected, result)
	}
}

func TestQuote(t *testing.T) {
	result := quote("test string")
	if result != "\"test string\"" {
		t.Errorf("expected quoted string, got '%s'", result)
	}
}

func TestDefaultValue(t *testing.T) {
	// Non-empty value should be returned
	result := defaultValue("default", "actual")
	if result != "actual" {
		t.Errorf("expected 'actual', got '%v'", result)
	}

	// Empty value should return default
	result = defaultValue("default", "")
	if result != "default" {
		t.Errorf("expected 'default', got '%v'", result)
	}

	// Nil value should return default
	result = defaultValue("default", nil)
	if result != "default" {
		t.Errorf("expected 'default', got '%v'", result)
	}
}

func TestToYaml(t *testing.T) {
	data := map[string]string{"key": "value"}
	result := toYaml(data)
	if !strings.Contains(result, "key: value") {
		t.Errorf("expected yaml output, got '%s'", result)
	}
}

func TestScaffoldTypes(t *testing.T) {
	// Test all scaffold types are valid
	types := []ScaffoldType{
		ScaffoldPlatform,
		ScaffoldInfrastructure,
		ScaffoldOrchestrator,
		ScaffoldObservability,
		ScaffoldDevEx,
		ScaffoldSecurity,
		ScaffoldFull,
	}

	gen := NewGenerator()

	for _, scaffoldType := range types {
		config := ScaffoldConfig{
			Type:          scaffoldType,
			Name:          "test",
			OutputDir:     t.TempDir(),
			CloudProvider: "aws",
			Environment:   "dev",
			DryRun:        true,
		}

		_, err := gen.Generate(config)
		if err != nil {
			t.Errorf("Generate failed for type %s: %v", scaffoldType, err)
		}
	}
}
