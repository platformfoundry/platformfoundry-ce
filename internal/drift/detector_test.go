package drift

import (
	"context"
	"testing"
)

// MockStateProvider for testing
type MockStateProvider struct {
	states map[string]map[string]interface{}
}

func (m *MockStateProvider) GetActualState(ctx context.Context, resourceType, resourceID string) (map[string]interface{}, error) {
	key := resourceType + "/" + resourceID
	if state, ok := m.states[key]; ok {
		return state, nil
	}
	return nil, nil
}

func TestNewDetector(t *testing.T) {
	d := NewDetector(DetectorConfig{}, nil)
	if d == nil {
		t.Fatal("NewDetector returned nil")
	}
	if d.config.Concurrency != 5 {
		t.Errorf("Expected default concurrency 5, got %d", d.config.Concurrency)
	}
}

func TestDetectNoDrift(t *testing.T) {
	d := NewDetector(DetectorConfig{}, nil)

	resources := []Resource{
		{
			ID:   "res-1",
			Type: "Deployment",
			Name: "my-app",
			Desired: map[string]interface{}{
				"replicas": 3,
				"image":    "nginx:1.19",
			},
			Actual: map[string]interface{}{
				"replicas": 3,
				"image":    "nginx:1.19",
			},
		},
	}

	report, err := d.Detect(context.Background(), resources)
	if err != nil {
		t.Fatalf("Detect failed: %v", err)
	}

	if report.Summary.Total != 0 {
		t.Errorf("Expected 0 drifts, got %d", report.Summary.Total)
	}
}

func TestDetectModifiedDrift(t *testing.T) {
	d := NewDetector(DetectorConfig{}, nil)

	resources := []Resource{
		{
			ID:   "res-1",
			Type: "Deployment",
			Name: "my-app",
			Desired: map[string]interface{}{
				"replicas": 3,
				"image":    "nginx:1.19",
			},
			Actual: map[string]interface{}{
				"replicas": 5, // Drifted
				"image":    "nginx:1.19",
			},
		},
	}

	report, err := d.Detect(context.Background(), resources)
	if err != nil {
		t.Fatalf("Detect failed: %v", err)
	}

	if report.Summary.Total != 1 {
		t.Errorf("Expected 1 drift, got %d", report.Summary.Total)
	}

	if len(report.Drifts) != 1 {
		t.Fatalf("Expected 1 drift in list")
	}

	drift := report.Drifts[0]
	if drift.Type != DriftModified {
		t.Errorf("Expected drift type modified, got %s", drift.Type)
	}
	if drift.Path != "replicas" {
		t.Errorf("Expected path 'replicas', got %s", drift.Path)
	}
}

func TestDetectDeletedDrift(t *testing.T) {
	d := NewDetector(DetectorConfig{}, nil)

	resources := []Resource{
		{
			ID:   "res-1",
			Type: "Deployment",
			Name: "my-app",
			Desired: map[string]interface{}{
				"replicas": 3,
			},
			Actual: nil, // Resource doesn't exist
		},
	}

	report, err := d.Detect(context.Background(), resources)
	if err != nil {
		t.Fatalf("Detect failed: %v", err)
	}

	if len(report.Drifts) != 1 {
		t.Fatalf("Expected 1 drift, got %d", len(report.Drifts))
	}

	if report.Drifts[0].Type != DriftDeleted {
		t.Errorf("Expected drift type deleted, got %s", report.Drifts[0].Type)
	}
}

func TestDetectAddedDrift(t *testing.T) {
	d := NewDetector(DetectorConfig{}, nil)

	resources := []Resource{
		{
			ID:   "res-1",
			Type: "Deployment",
			Name: "my-app",
			Desired: map[string]interface{}{
				"replicas": 3,
			},
			Actual: map[string]interface{}{
				"replicas":  3,
				"extraField": "unexpected", // Added field
			},
		},
	}

	report, err := d.Detect(context.Background(), resources)
	if err != nil {
		t.Fatalf("Detect failed: %v", err)
	}

	if len(report.Drifts) != 1 {
		t.Fatalf("Expected 1 drift, got %d", len(report.Drifts))
	}

	if report.Drifts[0].Type != DriftAdded {
		t.Errorf("Expected drift type added, got %s", report.Drifts[0].Type)
	}
}

func TestDetectNestedDrift(t *testing.T) {
	d := NewDetector(DetectorConfig{}, nil)

	resources := []Resource{
		{
			ID:   "res-1",
			Type: "Deployment",
			Name: "my-app",
			Desired: map[string]interface{}{
				"spec": map[string]interface{}{
					"replicas": 3,
					"template": map[string]interface{}{
						"image": "nginx:1.19",
					},
				},
			},
			Actual: map[string]interface{}{
				"spec": map[string]interface{}{
					"replicas": 3,
					"template": map[string]interface{}{
						"image": "nginx:1.20", // Drifted
					},
				},
			},
		},
	}

	report, err := d.Detect(context.Background(), resources)
	if err != nil {
		t.Fatalf("Detect failed: %v", err)
	}

	if len(report.Drifts) != 1 {
		t.Fatalf("Expected 1 drift, got %d", len(report.Drifts))
	}

	if report.Drifts[0].Path != "spec.template.image" {
		t.Errorf("Expected path 'spec.template.image', got %s", report.Drifts[0].Path)
	}
}

func TestIgnorePaths(t *testing.T) {
	d := NewDetector(DetectorConfig{
		IgnorePaths: []string{"metadata.generation", "status*"},
	}, nil)

	resources := []Resource{
		{
			ID:   "res-1",
			Type: "Deployment",
			Name: "my-app",
			Desired: map[string]interface{}{
				"metadata": map[string]interface{}{
					"name":       "my-app",
					"generation": 1,
				},
				"status": map[string]interface{}{
					"ready": true,
				},
			},
			Actual: map[string]interface{}{
				"metadata": map[string]interface{}{
					"name":       "my-app",
					"generation": 5, // Different but ignored
				},
				"status": map[string]interface{}{
					"ready": false, // Different but ignored
				},
			},
		},
	}

	report, err := d.Detect(context.Background(), resources)
	if err != nil {
		t.Fatalf("Detect failed: %v", err)
	}

	if report.Summary.Total != 0 {
		t.Errorf("Expected 0 drifts (paths ignored), got %d", report.Summary.Total)
	}
}

func TestSeverityMapping(t *testing.T) {
	d := NewDetector(DetectorConfig{
		SeverityMapping: map[string]DriftSeverity{
			"replicas": SeverityCritical,
		},
	}, nil)

	resources := []Resource{
		{
			ID:   "res-1",
			Type: "Deployment",
			Name: "my-app",
			Desired: map[string]interface{}{
				"replicas": 3,
			},
			Actual: map[string]interface{}{
				"replicas": 1,
			},
		},
	}

	report, err := d.Detect(context.Background(), resources)
	if err != nil {
		t.Fatalf("Detect failed: %v", err)
	}

	if len(report.Drifts) != 1 {
		t.Fatalf("Expected 1 drift")
	}

	if report.Drifts[0].Severity != SeverityCritical {
		t.Errorf("Expected critical severity, got %s", report.Drifts[0].Severity)
	}
}

func TestFormatReport(t *testing.T) {
	report := &Report{
		ID:               "test-report",
		ResourcesChecked: 5,
		Drifts: []Drift{
			{
				ID:           "drift-1",
				ResourceType: "Deployment",
				ResourceName: "my-app",
				Type:         DriftModified,
				Severity:     SeverityHigh,
				Description:  "Field changed",
				Path:         "spec.replicas",
				DesiredValue: 3,
				ActualValue:  5,
			},
		},
		Summary: Summary{
			Total: 1,
			High:  1,
			ByType: map[DriftType]int{
				DriftModified: 1,
			},
		},
	}

	output := FormatReport(report)

	expectedStrings := []string{
		"Drift Detection Report",
		"test-report",
		"my-app",
		"spec.replicas",
		"Total Drifts: 1",
	}

	for _, expected := range expectedStrings {
		if !containsString(output, expected) {
			t.Errorf("Expected output to contain '%s'", expected)
		}
	}
}

func TestMultipleResources(t *testing.T) {
	d := NewDetector(DetectorConfig{Concurrency: 2}, nil)

	resources := []Resource{
		{
			ID:      "res-1",
			Type:    "Deployment",
			Name:    "app-1",
			Desired: map[string]interface{}{"replicas": 3},
			Actual:  map[string]interface{}{"replicas": 5},
		},
		{
			ID:      "res-2",
			Type:    "Deployment",
			Name:    "app-2",
			Desired: map[string]interface{}{"replicas": 2},
			Actual:  map[string]interface{}{"replicas": 2},
		},
		{
			ID:      "res-3",
			Type:    "Service",
			Name:    "svc-1",
			Desired: map[string]interface{}{"port": 80},
			Actual:  map[string]interface{}{"port": 8080},
		},
	}

	report, err := d.Detect(context.Background(), resources)
	if err != nil {
		t.Fatalf("Detect failed: %v", err)
	}

	if report.ResourcesChecked != 3 {
		t.Errorf("Expected 3 resources checked, got %d", report.ResourcesChecked)
	}

	if report.Summary.Total != 2 {
		t.Errorf("Expected 2 drifts, got %d", report.Summary.Total)
	}
}

func containsString(s, substr string) bool {
	return len(s) >= len(substr) && containsHelper(s, substr)
}

func containsHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
