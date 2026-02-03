package commands

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/platformfoundry/pf-ce/pkg/types"
)

// mockParser implements Parser interface for testing
type mockParser struct {
	resources []types.Resource
	err       error
}

func (m *mockParser) ParseFile(path string) ([]types.Resource, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.resources, nil
}

// mockExecutor implements Executor interface for testing
type mockExecutor struct {
	results []ComponentResult
	err     error
}

func (m *mockExecutor) Apply(ctx context.Context, resources []types.Resource) ([]ComponentResult, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.results, nil
}

func TestNewApplyCommand(t *testing.T) {
	parser := &mockParser{}
	executor := &mockExecutor{}

	cmd := NewApplyCommand(parser, executor)
	if cmd == nil {
		t.Fatal("NewApplyCommand returned nil")
	}
	if cmd.Parser == nil {
		t.Error("Parser should not be nil")
	}
	if cmd.Executor == nil {
		t.Error("Executor should not be nil")
	}
	if cmd.Parallelism != 4 {
		t.Errorf("Expected parallelism 4, got %d", cmd.Parallelism)
	}
}

func TestApplyCommand_Execute_DryRun(t *testing.T) {
	resources := []types.Resource{
		{
			APIVersion: "platformfoundry.io/v1",
			Kind:       "Platform",
			Metadata: types.Metadata{
				Name: "test-platform",
			},
			Spec: map[string]interface{}{},
		},
	}

	cmd := &ApplyCommand{
		Parser:   &mockParser{resources: resources},
		Executor: &mockExecutor{},
	}

	input := ApplyInput{
		FilePath: "test.yaml",
		DryRun:   true,
	}

	output, err := cmd.Execute(context.Background(), input)
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	if !output.Success {
		t.Error("Expected success in dry run mode")
	}
	if output.Platform != "test-platform" {
		t.Errorf("Expected platform test-platform, got %s", output.Platform)
	}
	if len(output.Components) != 0 {
		t.Error("Expected no components in dry run")
	}
}

func TestApplyCommand_Execute_NoPlatform(t *testing.T) {
	resources := []types.Resource{
		{
			APIVersion: "platformfoundry.io/v1",
			Kind:       "Infrastructure",
			Metadata: types.Metadata{
				Name: "test-infra",
			},
		},
	}

	cmd := &ApplyCommand{
		Parser:   &mockParser{resources: resources},
		Executor: &mockExecutor{},
	}

	input := ApplyInput{
		FilePath: "test.yaml",
	}

	_, err := cmd.Execute(context.Background(), input)
	if err == nil {
		t.Error("Expected error when no platform resource")
	}
}

func TestApplyCommand_Execute_EmptyFilePath(t *testing.T) {
	cmd := &ApplyCommand{
		Parser:   &mockParser{},
		Executor: &mockExecutor{},
	}

	input := ApplyInput{
		FilePath: "",
	}

	_, err := cmd.Execute(context.Background(), input)
	if err == nil {
		t.Error("Expected error for empty file path")
	}
}

func TestApplyCommand_Execute_ParseError(t *testing.T) {
	cmd := &ApplyCommand{
		Parser:   &mockParser{err: errors.New("parse error")},
		Executor: &mockExecutor{},
	}

	input := ApplyInput{
		FilePath: "test.yaml",
	}

	_, err := cmd.Execute(context.Background(), input)
	if err == nil {
		t.Error("Expected error on parse failure")
	}
}

func TestApplyCommand_Execute_WithExecutor(t *testing.T) {
	resources := []types.Resource{
		{
			APIVersion: "platformfoundry.io/v1",
			Kind:       "Platform",
			Metadata: types.Metadata{
				Name: "test-platform",
			},
			Spec: map[string]interface{}{},
		},
	}

	results := []ComponentResult{
		{Name: "terraform", Type: "infrastructure", Status: "success"},
		{Name: "argocd", Type: "orchestrator", Status: "success"},
	}

	cmd := &ApplyCommand{
		Parser:   &mockParser{resources: resources},
		Executor: &mockExecutor{results: results},
	}

	input := ApplyInput{
		FilePath: "test.yaml",
	}

	output, err := cmd.Execute(context.Background(), input)
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	if !output.Success {
		t.Error("Expected success")
	}
	if len(output.Components) != 2 {
		t.Errorf("Expected 2 components, got %d", len(output.Components))
	}
	if output.ErrorCount != 0 {
		t.Errorf("Expected 0 errors, got %d", output.ErrorCount)
	}
}

func TestApplyCommand_Execute_WithErrors(t *testing.T) {
	resources := []types.Resource{
		{
			APIVersion: "platformfoundry.io/v1",
			Kind:       "Platform",
			Metadata: types.Metadata{
				Name: "test-platform",
			},
			Spec: map[string]interface{}{},
		},
	}

	results := []ComponentResult{
		{Name: "terraform", Type: "infrastructure", Status: "success"},
		{Name: "argocd", Type: "orchestrator", Status: "failed", Message: "connection refused"},
	}

	cmd := &ApplyCommand{
		Parser:   &mockParser{resources: resources},
		Executor: &mockExecutor{results: results},
	}

	input := ApplyInput{
		FilePath: "test.yaml",
	}

	output, err := cmd.Execute(context.Background(), input)
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	if output.Success {
		t.Error("Expected failure with errors")
	}
	if output.ErrorCount != 1 {
		t.Errorf("Expected 1 error, got %d", output.ErrorCount)
	}
}

func TestApplyCommand_Execute_ExecutorError(t *testing.T) {
	resources := []types.Resource{
		{
			APIVersion: "platformfoundry.io/v1",
			Kind:       "Platform",
			Metadata: types.Metadata{
				Name: "test-platform",
			},
			Spec: map[string]interface{}{},
		},
	}

	cmd := &ApplyCommand{
		Parser:   &mockParser{resources: resources},
		Executor: &mockExecutor{err: errors.New("executor failed")},
	}

	input := ApplyInput{
		FilePath: "test.yaml",
	}

	_, err := cmd.Execute(context.Background(), input)
	if err == nil {
		t.Error("Expected error on executor failure")
	}
}

func TestApplyCommand_Execute_WithTimeout(t *testing.T) {
	resources := []types.Resource{
		{
			APIVersion: "platformfoundry.io/v1",
			Kind:       "Platform",
			Metadata: types.Metadata{
				Name: "test-platform",
			},
			Spec: map[string]interface{}{},
		},
	}

	cmd := &ApplyCommand{
		Parser:   &mockParser{resources: resources},
		Executor: &mockExecutor{results: []ComponentResult{}},
	}

	input := ApplyInput{
		FilePath: "test.yaml",
		Timeout:  5 * time.Second,
	}

	output, err := cmd.Execute(context.Background(), input)
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	if !output.Success {
		t.Error("Expected success")
	}
}

func TestApplyCommand_Validate(t *testing.T) {
	resources := []types.Resource{
		{
			APIVersion: "platformfoundry.io/v1",
			Kind:       "Platform",
			Metadata: types.Metadata{
				Name: "test-platform",
			},
			Spec: map[string]interface{}{},
		},
	}

	cmd := &ApplyCommand{
		Parser: &mockParser{resources: resources},
	}

	input := ApplyInput{
		FilePath: "test.yaml",
	}

	err := cmd.Validate(context.Background(), input)
	if err != nil {
		t.Errorf("Validate failed: %v", err)
	}
}

func TestApplyCommand_Validate_NoPlatform(t *testing.T) {
	resources := []types.Resource{
		{
			APIVersion: "platformfoundry.io/v1",
			Kind:       "Infrastructure",
		},
	}

	cmd := &ApplyCommand{
		Parser: &mockParser{resources: resources},
	}

	input := ApplyInput{
		FilePath: "test.yaml",
	}

	err := cmd.Validate(context.Background(), input)
	if err == nil {
		t.Error("Expected error when no platform")
	}
}

func TestApplyCommand_FindPlatform(t *testing.T) {
	cmd := &ApplyCommand{}

	tests := []struct {
		name      string
		resources []types.Resource
		wantErr   bool
		wantName  string
	}{
		{
			name: "valid platform",
			resources: []types.Resource{
				{
					Kind:     "Platform",
					Metadata: types.Metadata{Name: "my-platform"},
				},
			},
			wantErr:  false,
			wantName: "my-platform",
		},
		{
			name: "platform among others",
			resources: []types.Resource{
				{Kind: "Infrastructure"},
				{Kind: "Platform", Metadata: types.Metadata{Name: "found"}},
				{Kind: "Orchestrator"},
			},
			wantErr:  false,
			wantName: "found",
		},
		{
			name: "no platform",
			resources: []types.Resource{
				{Kind: "Infrastructure"},
			},
			wantErr: true,
		},
		{
			name:      "empty resources",
			resources: []types.Resource{},
			wantErr:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			platform, err := cmd.findPlatform(tt.resources)

			if tt.wantErr {
				if err == nil {
					t.Error("Expected error")
				}
				return
			}

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}

			if platform.Metadata.Name != tt.wantName {
				t.Errorf("Expected name %s, got %s", tt.wantName, platform.Metadata.Name)
			}
		})
	}
}

func TestApplyOutput_Fields(t *testing.T) {
	output := &ApplyOutput{
		Platform: "test",
		Components: []ComponentResult{
			{Name: "infra", Status: "success"},
			{Name: "orch", Status: "failed"},
		},
		Duration:   5 * time.Second,
		Success:    false,
		ErrorCount: 1,
	}

	if output.Success {
		t.Error("Expected success to be false with errors")
	}
	if output.ErrorCount != 1 {
		t.Errorf("Expected 1 error, got %d", output.ErrorCount)
	}
	if len(output.Components) != 2 {
		t.Errorf("Expected 2 components, got %d", len(output.Components))
	}
	if output.Duration != 5*time.Second {
		t.Errorf("Expected 5s duration, got %v", output.Duration)
	}
}

func TestComponentResult_Fields(t *testing.T) {
	result := ComponentResult{
		Name:    "terraform",
		Type:    "infrastructure",
		Status:  "success",
		Message: "Applied successfully",
		Outputs: map[string]string{
			"cluster_endpoint": "https://k8s.example.com",
		},
	}

	if result.Name != "terraform" {
		t.Errorf("Expected name terraform, got %s", result.Name)
	}
	if result.Type != "infrastructure" {
		t.Errorf("Expected type infrastructure, got %s", result.Type)
	}
	if result.Outputs["cluster_endpoint"] != "https://k8s.example.com" {
		t.Error("Expected cluster_endpoint output")
	}
}
