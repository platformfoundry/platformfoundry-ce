package doctor

import (
	"context"
	"testing"
	"time"
)

func TestNew(t *testing.T) {
	doc := New()
	if doc == nil {
		t.Fatal("New() returned nil")
	}
	if len(doc.checks) == 0 {
		t.Error("No default checks registered")
	}
}

func TestRunAll(t *testing.T) {
	doc := New()
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	report := doc.RunAll(ctx)

	if report == nil {
		t.Fatal("RunAll returned nil report")
	}

	if len(report.Checks) == 0 {
		t.Error("No checks were run")
	}

	if report.Summary.Total != len(report.Checks) {
		t.Errorf("Summary total %d doesn't match checks count %d",
			report.Summary.Total, len(report.Checks))
	}

	// Verify summary counts add up
	sum := report.Summary.Passed + report.Summary.Warnings +
		report.Summary.Errors + report.Summary.Skipped
	if sum != report.Summary.Total {
		t.Errorf("Summary counts don't add up: %d != %d", sum, report.Summary.Total)
	}
}

func TestRunAllWithCancellation(t *testing.T) {
	doc := New()
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	report := doc.RunAll(ctx)

	// Should return quickly with partial or no results
	if report == nil {
		t.Fatal("RunAll returned nil report even with cancelled context")
	}
}

func TestCheckStatus(t *testing.T) {
	statuses := []CheckStatus{StatusOK, StatusWarning, StatusError, StatusSkipped}

	for _, status := range statuses {
		icon := getStatusIcon(status)
		if icon == "[?]" {
			t.Errorf("Unknown icon for status %s", status)
		}
	}
}

func TestFormatReport(t *testing.T) {
	report := &Report{
		Checks: []Check{
			{
				Name:     "Test Check",
				Category: "Test",
				Status:   StatusOK,
				Message:  "All good",
			},
			{
				Name:        "Warning Check",
				Category:    "Test",
				Status:      StatusWarning,
				Message:     "Something minor",
				Remediation: "Do this to fix",
			},
		},
		Summary: Summary{
			Total:    2,
			Passed:   1,
			Warnings: 1,
		},
		GeneratedAt: time.Now(),
		Duration:    100 * time.Millisecond,
	}

	output := FormatReport(report)

	if output == "" {
		t.Error("FormatReport returned empty string")
	}

	// Check for expected content
	expectedStrings := []string{
		"Platform Foundry Doctor",
		"Test Check",
		"Warning Check",
		"Passed:",
		"Warnings:",
	}

	for _, expected := range expectedStrings {
		if !contains(output, expected) {
			t.Errorf("Expected output to contain '%s'", expected)
		}
	}
}

func TestCheckCategories(t *testing.T) {
	doc := New()
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	report := doc.RunAll(ctx)

	// Verify checks have categories
	for _, check := range report.Checks {
		if check.Category == "" {
			t.Errorf("Check '%s' has no category", check.Name)
		}
		if check.Name == "" {
			t.Error("Check has no name")
		}
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsHelper(s, substr))
}

func containsHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
