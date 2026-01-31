package mock

import (
	"errors"
	"testing"
	"time"

	"github.com/platformfoundry/platformfoundry-ce/pkg/plugin"
)

func TestDefaultMockConfig(t *testing.T) {
	config := DefaultMockConfig()

	if config == nil {
		t.Fatal("expected non-nil config")
	}
	if config.Mode != MockModeRealistic {
		t.Errorf("expected mode realistic, got %s", config.Mode)
	}
	if config.DefaultDelay != 2*time.Second {
		t.Errorf("expected default delay 2s, got %v", config.DefaultDelay)
	}
	if config.FailureRate != 0 {
		t.Errorf("expected failure rate 0, got %f", config.FailureRate)
	}
}

func TestNewMockPlugin(t *testing.T) {
	p := NewMockPlugin("terraform", "Infrastructure", nil)

	if p == nil {
		t.Fatal("expected non-nil plugin")
	}
	if p.Name() != "terraform-mock" {
		t.Errorf("expected name 'terraform-mock', got '%s'", p.Name())
	}
	if p.Type() != "Infrastructure" {
		t.Errorf("expected type 'Infrastructure', got '%s'", p.Type())
	}
	if p.Version() != "1.0.0-mock" {
		t.Errorf("expected version '1.0.0-mock', got '%s'", p.Version())
	}
}

func TestNewMockPluginWithConfig(t *testing.T) {
	config := &MockConfig{
		Mode:         MockModeInstant,
		DefaultDelay: 100 * time.Millisecond,
	}

	p := NewMockPlugin("argocd", "Orchestrator", config)

	if p.config.Mode != MockModeInstant {
		t.Errorf("expected mode instant, got %s", p.config.Mode)
	}
}

func TestMockPluginValidate(t *testing.T) {
	p := NewMockPlugin("terraform", "Infrastructure", nil)

	spec := map[string]interface{}{
		"name":     "test-platform",
		"provider": "aws",
	}

	err := p.Validate(spec)
	if err != nil {
		t.Errorf("Validate failed: %v", err)
	}
}

func TestMockPluginPlan(t *testing.T) {
	p := NewMockPlugin("terraform", "Infrastructure", nil)

	spec := map[string]interface{}{
		"name": "test-platform",
	}

	plan, err := p.Plan(spec)
	if err != nil {
		t.Errorf("Plan failed: %v", err)
	}
	if plan == nil {
		t.Fatal("expected non-nil plan")
	}
	if len(plan.Actions) != 1 {
		t.Errorf("expected 1 action, got %d", len(plan.Actions))
	}
}

func TestMockPluginApplyInstant(t *testing.T) {
	config := &MockConfig{Mode: MockModeInstant}
	p := NewMockPlugin("terraform", "Infrastructure", config)

	spec := map[string]interface{}{
		"name":     "test-platform",
		"provider": "aws",
	}

	start := time.Now()
	result, err := p.Apply(spec)
	elapsed := time.Since(start)

	if err != nil {
		t.Errorf("Apply failed: %v", err)
	}
	if result == nil {
		t.Fatal("expected non-nil result")
	}
	if result.Status != "success" {
		t.Errorf("expected status 'success', got '%s'", result.Status)
	}
	// Instant mode should be fast
	if elapsed > 100*time.Millisecond {
		t.Errorf("instant mode should be fast, took %v", elapsed)
	}
}

func TestMockPluginApplyRealistic(t *testing.T) {
	config := &MockConfig{
		Mode:         MockModeRealistic,
		DefaultDelay: 100 * time.Millisecond,
	}
	p := NewMockPlugin("terraform", "Infrastructure", config)

	spec := map[string]interface{}{
		"name":     "test-platform",
		"provider": "aws",
	}

	start := time.Now()
	result, err := p.Apply(spec)
	elapsed := time.Since(start)

	if err != nil {
		t.Errorf("Apply failed: %v", err)
	}
	if result == nil {
		t.Fatal("expected non-nil result")
	}
	// Realistic mode should take at least the delay
	if elapsed < 90*time.Millisecond {
		t.Errorf("realistic mode should have delay, took %v", elapsed)
	}
}

func TestMockPluginApplyRealisticWithToolDelay(t *testing.T) {
	config := &MockConfig{
		Mode:         MockModeRealistic,
		DefaultDelay: 500 * time.Millisecond,
		PerToolDelay: map[string]time.Duration{
			"terraform": 100 * time.Millisecond,
		},
	}
	p := NewMockPlugin("terraform", "Infrastructure", config)

	spec := map[string]interface{}{"name": "test"}

	start := time.Now()
	_, err := p.Apply(spec)
	elapsed := time.Since(start)

	if err != nil {
		t.Errorf("Apply failed: %v", err)
	}
	// Should use tool-specific delay, not default
	if elapsed > 200*time.Millisecond {
		t.Errorf("should use tool-specific delay (100ms), took %v", elapsed)
	}
}

func TestMockPluginApplyChaos(t *testing.T) {
	config := &MockConfig{
		Mode:         MockModeChaos,
		FailureTools: []string{"terraform"},
		DefaultDelay: 10 * time.Millisecond,
	}
	p := NewMockPlugin("terraform", "Infrastructure", config)

	spec := map[string]interface{}{"name": "test"}

	_, err := p.Apply(spec)
	if err == nil {
		t.Error("expected failure for tool in FailureTools")
	}
}

func TestMockPluginApplyChaosRandomFailure(t *testing.T) {
	config := &MockConfig{
		Mode:         MockModeChaos,
		FailureRate:  1.0, // Always fail
		DefaultDelay: 10 * time.Millisecond,
	}
	p := NewMockPlugin("terraform", "Infrastructure", config)

	spec := map[string]interface{}{"name": "test"}

	_, err := p.Apply(spec)
	if err == nil {
		t.Error("expected random failure with 100% failure rate")
	}
}

func TestMockPluginApplyChaosNoFailure(t *testing.T) {
	config := &MockConfig{
		Mode:         MockModeChaos,
		FailureRate:  0, // Never fail
		DefaultDelay: 10 * time.Millisecond,
	}
	p := NewMockPlugin("argocd", "Orchestrator", config)

	spec := map[string]interface{}{"name": "test"}

	result, err := p.Apply(spec)
	if err != nil {
		t.Errorf("unexpected failure: %v", err)
	}
	if result == nil {
		t.Error("expected non-nil result")
	}
}

func TestMockPluginApplyRecorded(t *testing.T) {
	config := &MockConfig{
		Mode:            MockModeRecorded,
		RecordResponses: true,
	}
	p := NewMockPlugin("terraform", "Infrastructure", config)

	// First apply in instant mode to record
	p.config.Mode = MockModeInstant
	spec := map[string]interface{}{"name": "test"}
	_, _ = p.Apply(spec)

	// Switch to recorded mode
	p.config.Mode = MockModeRecorded
	result, err := p.Apply(spec)

	if err != nil {
		t.Errorf("Apply failed: %v", err)
	}
	if result == nil {
		t.Fatal("expected non-nil result from recording")
	}
}

func TestMockPluginApplyRecordedNoRecording(t *testing.T) {
	config := &MockConfig{Mode: MockModeRecorded}
	p := NewMockPlugin("terraform", "Infrastructure", config)

	spec := map[string]interface{}{"name": "test"}

	// Should fall back to instant when no recording
	result, err := p.Apply(spec)
	if err != nil {
		t.Errorf("Apply failed: %v", err)
	}
	if result == nil {
		t.Fatal("expected non-nil result")
	}
}

func TestMockPluginDelete(t *testing.T) {
	p := NewMockPlugin("terraform", "Infrastructure", nil)

	err := p.Delete("test-resource")
	if err != nil {
		t.Errorf("Delete failed: %v", err)
	}
}

func TestMockPluginStatus(t *testing.T) {
	p := NewMockPlugin("terraform", "Infrastructure", nil)

	status, err := p.Status("test-resource")
	if err != nil {
		t.Errorf("Status failed: %v", err)
	}
	if status == nil {
		t.Fatal("expected non-nil status")
	}
	if !status.Ready {
		t.Error("expected ready status")
	}
	if status.Details["mock"] != "true" {
		t.Error("expected mock=true in details")
	}
}

func TestMockPluginInfrastructureOutputs(t *testing.T) {
	config := &MockConfig{Mode: MockModeInstant}
	p := NewMockPlugin("terraform", "infrastructure", config)

	spec := map[string]interface{}{
		"name":     "test",
		"provider": "aws",
	}

	result, err := p.Apply(spec)
	if err != nil {
		t.Fatalf("Apply failed: %v", err)
	}

	if _, ok := result.Outputs["endpoint"]; !ok {
		t.Error("expected output 'endpoint' for infrastructure type")
	}
	if _, ok := result.Outputs["resource_id"]; !ok {
		t.Error("expected output 'resource_id' for infrastructure type")
	}
}

func TestMockPluginObservabilityOutputs(t *testing.T) {
	config := &MockConfig{Mode: MockModeInstant}
	p := NewMockPlugin("prometheus", "observability", config)

	spec := map[string]interface{}{"name": "test"}

	result, err := p.Apply(spec)
	if err != nil {
		t.Fatalf("Apply failed: %v", err)
	}

	if _, ok := result.Outputs["dashboard_url"]; !ok {
		t.Error("expected output 'dashboard_url' for observability type")
	}
	if _, ok := result.Outputs["metrics_endpoint"]; !ok {
		t.Error("expected output 'metrics_endpoint' for observability type")
	}
}

func TestMockPluginDevExOutputs(t *testing.T) {
	config := &MockConfig{Mode: MockModeInstant}
	p := NewMockPlugin("backstage", "devex", config)

	spec := map[string]interface{}{"name": "test"}

	result, err := p.Apply(spec)
	if err != nil {
		t.Fatalf("Apply failed: %v", err)
	}

	if _, ok := result.Outputs["url"]; !ok {
		t.Error("expected url output for devex type")
	}
	if _, ok := result.Outputs["api_key"]; !ok {
		t.Error("expected api_key output for devex type")
	}
}

func TestMockPluginResponseOverride(t *testing.T) {
	config := &MockConfig{
		Mode: MockModeInstant,
		ResponseOverride: map[string]interface{}{
			"terraform": map[string]string{
				"custom_output": "custom_value",
			},
		},
	}
	p := NewMockPlugin("terraform", "Infrastructure", config)

	spec := map[string]interface{}{"name": "test", "provider": "aws"}

	result, err := p.Apply(spec)
	if err != nil {
		t.Fatalf("Apply failed: %v", err)
	}

	if result.Outputs["custom_output"] != "custom_value" {
		t.Error("expected response override to be applied")
	}
}

// Tests for WrapPlugin

type fakePlugin struct {
	name    string
	typ     string
	version string
}

func (p *fakePlugin) Name() string                               { return p.name }
func (p *fakePlugin) Type() string                               { return p.typ }
func (p *fakePlugin) Version() string                            { return p.version }
func (p *fakePlugin) ConfigType() interface{}                    { return nil }
func (p *fakePlugin) Validate(spec map[string]interface{}) error { return nil }
func (p *fakePlugin) Plan(spec map[string]interface{}) (*plugin.Plan, error) {
	return &plugin.Plan{Actions: []string{"fake plan"}}, nil
}
func (p *fakePlugin) Apply(spec map[string]interface{}) (*plugin.Result, error) {
	return &plugin.Result{Status: "real"}, nil
}
func (p *fakePlugin) Delete(name string) error { return nil }
func (p *fakePlugin) Status(name string) (*plugin.Status, error) {
	return &plugin.Status{Ready: true}, nil
}

func TestWrapPlugin(t *testing.T) {
	realPlugin := &fakePlugin{
		name:    "real-terraform",
		typ:     "Infrastructure",
		version: "1.5.0",
	}

	wrapped := WrapPlugin(realPlugin, nil)

	if wrapped.Name() != "real-terraform-mock" {
		t.Errorf("expected name 'real-terraform-mock', got '%s'", wrapped.Name())
	}
	if wrapped.Type() != "Infrastructure" {
		t.Errorf("expected type 'Infrastructure', got '%s'", wrapped.Type())
	}
	if wrapped.Version() != "1.5.0-mock" {
		t.Errorf("expected version '1.5.0-mock', got '%s'", wrapped.Version())
	}
}

func TestWrapPluginUsesRealPluginForPlan(t *testing.T) {
	realPlugin := &fakePlugin{name: "real", typ: "Test", version: "1.0"}
	wrapped := WrapPlugin(realPlugin, nil)

	plan, err := wrapped.Plan(nil)
	if err != nil {
		t.Fatalf("Plan failed: %v", err)
	}
	if len(plan.Actions) != 1 || plan.Actions[0] != "fake plan" {
		t.Error("expected wrapped plugin to use real plugin's Plan")
	}
}

// Tests for ResponseRecorder

func TestNewResponseRecorder(t *testing.T) {
	recorder := NewResponseRecorder()

	if recorder == nil {
		t.Fatal("expected non-nil recorder")
	}
	if len(recorder.responses) != 0 {
		t.Error("expected empty responses")
	}
}

func TestResponseRecorderRecord(t *testing.T) {
	recorder := NewResponseRecorder()

	input := map[string]interface{}{"name": "test"}
	outputs := map[string]string{"endpoint": "http://test"}

	recorder.Record("terraform", "apply", input, outputs, nil)

	all := recorder.GetAll()
	if len(all) != 1 {
		t.Errorf("expected 1 recording, got %d", len(all))
	}

	responses := all["terraform:apply"]
	if len(responses) != 1 {
		t.Errorf("expected 1 response, got %d", len(responses))
	}
	if responses[0].Tool != "terraform" {
		t.Errorf("expected tool 'terraform', got '%s'", responses[0].Tool)
	}
	if responses[0].Operation != "apply" {
		t.Errorf("expected operation 'apply', got '%s'", responses[0].Operation)
	}
}

func TestResponseRecorderRecordWithError(t *testing.T) {
	recorder := NewResponseRecorder()

	testErr := errors.New("test error")
	recorder.Record("terraform", "apply", nil, nil, testErr)

	all := recorder.GetAll()
	responses := all["terraform:apply"]
	if responses[0].Error != testErr {
		t.Error("expected error to be recorded")
	}
}

func TestResponseRecorderPlayback(t *testing.T) {
	recorder := NewResponseRecorder()

	outputs := map[string]string{"key": "value"}
	recorder.Record("terraform", "apply", nil, outputs, nil)

	response, err := recorder.Playback("terraform", "apply")
	if err != nil {
		t.Errorf("Playback failed: %v", err)
	}
	if response.Outputs["key"] != "value" {
		t.Error("expected playback to return recorded output")
	}
}

func TestResponseRecorderPlaybackNotFound(t *testing.T) {
	recorder := NewResponseRecorder()

	_, err := recorder.Playback("nonexistent", "apply")
	if err == nil {
		t.Error("expected error for nonexistent recording")
	}
}

func TestResponseRecorderPlaybackReturnsLatest(t *testing.T) {
	recorder := NewResponseRecorder()

	recorder.Record("terraform", "apply", nil, map[string]string{"version": "1"}, nil)
	recorder.Record("terraform", "apply", nil, map[string]string{"version": "2"}, nil)
	recorder.Record("terraform", "apply", nil, map[string]string{"version": "3"}, nil)

	response, err := recorder.Playback("terraform", "apply")
	if err != nil {
		t.Fatalf("Playback failed: %v", err)
	}
	if response.Outputs["version"] != "3" {
		t.Error("expected playback to return latest recording")
	}
}

func TestResponseRecorderClear(t *testing.T) {
	recorder := NewResponseRecorder()

	recorder.Record("terraform", "apply", nil, nil, nil)
	recorder.Record("argocd", "apply", nil, nil, nil)

	recorder.Clear()

	all := recorder.GetAll()
	if len(all) != 0 {
		t.Errorf("expected 0 recordings after clear, got %d", len(all))
	}
}

func TestResponseRecorderGetAll(t *testing.T) {
	recorder := NewResponseRecorder()

	recorder.Record("terraform", "apply", nil, nil, nil)
	recorder.Record("terraform", "delete", nil, nil, nil)
	recorder.Record("argocd", "apply", nil, nil, nil)

	all := recorder.GetAll()
	if len(all) != 3 {
		t.Errorf("expected 3 different keys, got %d", len(all))
	}
}

func TestResponseRecorderGetAllReturnsSliceCopy(t *testing.T) {
	recorder := NewResponseRecorder()

	recorder.Record("terraform", "apply", nil, map[string]string{"key": "original"}, nil)

	all := recorder.GetAll()

	// Verify it's a copy of the slice (not affecting the original slice)
	originalLen := len(all["terraform:apply"])
	all["terraform:apply"] = append(all["terraform:apply"], RecordedResponse{})

	all2 := recorder.GetAll()
	if len(all2["terraform:apply"]) != originalLen {
		t.Error("expected GetAll to return a copy of the slice")
	}
}

// Test that MockPlugin implements plugin.Plugin interface
func TestMockPluginImplementsInterface(t *testing.T) {
	var _ plugin.Plugin = (*MockPlugin)(nil)
}
