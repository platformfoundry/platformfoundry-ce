package service

import (
	"testing"
	"time"

	"github.com/platformfoundry/pf-ce/pkg/types"
)

func TestNewScorecardCalculator(t *testing.T) {
	backend := NewMockBackend()
	calculator := NewScorecardCalculator(backend)

	if calculator == nil {
		t.Fatal("NewScorecardCalculator() returned nil")
	}

	if calculator.backend == nil {
		t.Error("Calculator backend is nil")
	}

	if calculator.engine == nil {
		t.Error("Calculator engine is nil")
	}
}

func TestScorecardCalculator_Calculate(t *testing.T) {
	backend := NewMockBackend()
	calculator := NewScorecardCalculator(backend)

	// Create a test service first
	service := &types.Service{
		APIVersion: "v1",
		Kind:       "Service",
		Metadata: types.Metadata{
			Name:         "test-service",
			Organization: "test-org",
		},
		Spec: types.ServiceSpec{
			Type: types.ServiceTypeMicroservice,
			Owner: types.ServiceOwner{
				Team:  "platform-team",
				Email: "platform@example.com",
			},
		},
		Status: types.ServiceStatus{
			State:  types.ServiceStateRunning,
			Health: types.ServiceHealthHealthy,
		},
	}

	// Convert service to resource and save
	manager := NewManager(backend)
	err := manager.Create(service)
	if err != nil {
		t.Fatalf("Failed to create service: %v", err)
	}

	// Calculate scorecard
	context := &CheckContext{
		HasReadme:    true,
		ReadmeLength: 1000,
		HasTests:     true,
		TestCoverage: 80.0,
	}

	scorecard, err := calculator.Calculate("test-service", "test-org", context)
	if err != nil {
		t.Fatalf("Calculate() failed: %v", err)
	}

	// Verify scorecard
	if scorecard.Metadata.Name != "test-service" {
		t.Errorf("Expected name 'test-service', got '%s'", scorecard.Metadata.Name)
	}

	if scorecard.Metadata.Organization != "test-org" {
		t.Errorf("Expected organization 'test-org', got '%s'", scorecard.Metadata.Organization)
	}

	if scorecard.Spec.ServiceRef != "test-service" {
		t.Errorf("Expected service ref 'test-service', got '%s'", scorecard.Spec.ServiceRef)
	}

	if len(scorecard.Spec.Checks) != 11 {
		t.Errorf("Expected 11 checks, got %d", len(scorecard.Spec.Checks))
	}

	// Verify scorecard is saved
	retrieved, err := calculator.Get("test-service", "test-org")
	if err != nil {
		t.Fatalf("Failed to get saved scorecard: %v", err)
	}

	if retrieved.Status.Score != scorecard.Status.Score {
		t.Errorf("Retrieved scorecard score mismatch: got %d, want %d", retrieved.Status.Score, scorecard.Status.Score)
	}
}

func TestScorecardCalculator_CalculateWithDefaultContext(t *testing.T) {
	backend := NewMockBackend()
	calculator := NewScorecardCalculator(backend)

	// Create a service with repository
	service := &types.Service{
		APIVersion: "v1",
		Kind:       "Service",
		Metadata: types.Metadata{
			Name:         "test-service",
			Organization: "test-org",
		},
		Spec: types.ServiceSpec{
			Type: types.ServiceTypeMicroservice,
			Owner: types.ServiceOwner{
				Team: "platform-team",
			},
			Repository: &types.RepositoryConfig{
				URL:    "https://github.com/test/repo",
				Branch: "main",
			},
			Links: []types.ServiceLink{
				{Name: "Metrics", URL: "http://metrics", Type: "metrics"},
				{Name: "Logs", URL: "http://logs", Type: "logs"},
			},
		},
		Status: types.ServiceStatus{
			State:  types.ServiceStateRunning,
			Health: types.ServiceHealthHealthy,
		},
	}

	// Save service
	manager := NewManager(backend)
	err := manager.Create(service)
	if err != nil {
		t.Fatalf("Failed to create service: %v", err)
	}

	// Calculate with nil context (should use default)
	scorecard, err := calculator.Calculate("test-service", "test-org", nil)
	if err != nil {
		t.Fatalf("Calculate() with default context failed: %v", err)
	}

	if scorecard == nil {
		t.Fatal("Calculate() returned nil scorecard")
	}

	// Should still have checks
	if len(scorecard.Spec.Checks) != 11 {
		t.Errorf("Expected 11 checks, got %d", len(scorecard.Spec.Checks))
	}
}

func TestScorecardCalculator_Get(t *testing.T) {
	backend := NewMockBackend()
	calculator := NewScorecardCalculator(backend)

	// Try to get non-existent scorecard
	_, err := calculator.Get("non-existent", "test-org")
	if err == nil {
		t.Error("Expected error when getting non-existent scorecard")
	}

	// Create and save a scorecard
	scorecard := &types.ServiceScorecard{
		APIVersion: "v1",
		Kind:       "ServiceScorecard",
		Metadata: types.Metadata{
			Name:         "test-service",
			Organization: "test-org",
		},
		Spec: types.ServiceScorecardSpec{
			ServiceRef: "test-service",
			Checks:     []types.Check{},
		},
		Status: types.ServiceScorecardStatus{
			Score:        75,
			Grade:        types.GradeC,
			TotalChecks:  11,
			PassedChecks: 8,
			FailedChecks: 3,
			EvaluatedAt:  time.Now(),
		},
	}

	err = calculator.Save(scorecard)
	if err != nil {
		t.Fatalf("Failed to save scorecard: %v", err)
	}

	// Get the scorecard
	retrieved, err := calculator.Get("test-service", "test-org")
	if err != nil {
		t.Fatalf("Failed to get scorecard: %v", err)
	}

	if retrieved.Status.Score != 75 {
		t.Errorf("Expected score 75, got %d", retrieved.Status.Score)
	}

	if retrieved.Status.Grade != types.GradeC {
		t.Errorf("Expected grade C, got %s", retrieved.Status.Grade)
	}
}

func TestScorecardCalculator_List(t *testing.T) {
	backend := NewMockBackend()
	calculator := NewScorecardCalculator(backend)

	// Initially empty
	scorecards, err := calculator.List("test-org")
	if err != nil {
		t.Fatalf("List() failed: %v", err)
	}

	if len(scorecards) != 0 {
		t.Errorf("Expected 0 scorecards, got %d", len(scorecards))
	}

	// Add some scorecards
	for i := 1; i <= 3; i++ {
		scorecard := &types.ServiceScorecard{
			APIVersion: "v1",
			Kind:       "ServiceScorecard",
			Metadata: types.Metadata{
				Name:         "test-service-" + string(rune('0'+i)),
				Organization: "test-org",
			},
			Spec: types.ServiceScorecardSpec{
				ServiceRef: "test-service-" + string(rune('0'+i)),
				Checks:     []types.Check{},
			},
			Status: types.ServiceScorecardStatus{
				Score:       75,
				Grade:       types.GradeC,
				EvaluatedAt: time.Now(),
			},
		}
		err = calculator.Save(scorecard)
		if err != nil {
			t.Fatalf("Failed to save scorecard %d: %v", i, err)
		}
	}

	// Add a scorecard for different org
	otherOrgScorecard := &types.ServiceScorecard{
		APIVersion: "v1",
		Kind:       "ServiceScorecard",
		Metadata: types.Metadata{
			Name:         "other-service",
			Organization: "other-org",
		},
		Spec: types.ServiceScorecardSpec{
			ServiceRef: "other-service",
			Checks:     []types.Check{},
		},
		Status: types.ServiceScorecardStatus{
			Score:       80,
			Grade:       types.GradeB,
			EvaluatedAt: time.Now(),
		},
	}
	err = calculator.Save(otherOrgScorecard)
	if err != nil {
		t.Fatalf("Failed to save other org scorecard: %v", err)
	}

	// List scorecards for test-org
	scorecards, err = calculator.List("test-org")
	if err != nil {
		t.Fatalf("List() failed: %v", err)
	}

	if len(scorecards) != 3 {
		t.Errorf("Expected 3 scorecards for test-org, got %d", len(scorecards))
	}

	// List scorecards for other-org
	otherScorecards, err := calculator.List("other-org")
	if err != nil {
		t.Fatalf("List() failed for other-org: %v", err)
	}

	if len(otherScorecards) != 1 {
		t.Errorf("Expected 1 scorecard for other-org, got %d", len(otherScorecards))
	}
}

func TestScorecardCalculator_ListByGrade(t *testing.T) {
	backend := NewMockBackend()
	calculator := NewScorecardCalculator(backend)

	// Add scorecards with different grades
	grades := []types.ScorecardGrade{types.GradeA, types.GradeB, types.GradeC, types.GradeA}
	for i, grade := range grades {
		scorecard := &types.ServiceScorecard{
			APIVersion: "v1",
			Kind:       "ServiceScorecard",
			Metadata: types.Metadata{
				Name:         "test-service-" + string(rune('0'+i)),
				Organization: "test-org",
			},
			Spec: types.ServiceScorecardSpec{
				ServiceRef: "test-service-" + string(rune('0'+i)),
				Checks:     []types.Check{},
			},
			Status: types.ServiceScorecardStatus{
				Grade:       grade,
				EvaluatedAt: time.Now(),
			},
		}
		err := calculator.Save(scorecard)
		if err != nil {
			t.Fatalf("Failed to save scorecard %d: %v", i, err)
		}
	}

	// List by grade A
	aScores, err := calculator.ListByGrade(types.GradeA, "test-org")
	if err != nil {
		t.Fatalf("ListByGrade(A) failed: %v", err)
	}

	if len(aScores) != 2 {
		t.Errorf("Expected 2 A-grade scorecards, got %d", len(aScores))
	}

	// List by grade B
	bScores, err := calculator.ListByGrade(types.GradeB, "test-org")
	if err != nil {
		t.Fatalf("ListByGrade(B) failed: %v", err)
	}

	if len(bScores) != 1 {
		t.Errorf("Expected 1 B-grade scorecard, got %d", len(bScores))
	}

	// List by grade F
	fScores, err := calculator.ListByGrade(types.GradeF, "test-org")
	if err != nil {
		t.Fatalf("ListByGrade(F) failed: %v", err)
	}

	if len(fScores) != 0 {
		t.Errorf("Expected 0 F-grade scorecards, got %d", len(fScores))
	}
}

func TestScorecardCalculator_GetStats(t *testing.T) {
	backend := NewMockBackend()
	calculator := NewScorecardCalculator(backend)

	// Add scorecards with various grades and checks
	scorecardData := []struct {
		grade  types.ScorecardGrade
		score  int
		checks []types.Check
	}{
		{
			grade: types.GradeA,
			score: 95,
			checks: []types.Check{
				{Category: types.CategoryDocumentation, Status: types.CheckStatusPassed},
				{Category: types.CategoryTesting, Status: types.CheckStatusPassed},
			},
		},
		{
			grade: types.GradeB,
			score: 85,
			checks: []types.Check{
				{Category: types.CategoryDocumentation, Status: types.CheckStatusPassed},
				{Category: types.CategoryTesting, Status: types.CheckStatusFailed},
			},
		},
		{
			grade: types.GradeC,
			score: 75,
			checks: []types.Check{
				{Category: types.CategoryDocumentation, Status: types.CheckStatusWarning},
				{Category: types.CategoryTesting, Status: types.CheckStatusFailed},
			},
		},
	}

	for i, data := range scorecardData {
		scorecard := &types.ServiceScorecard{
			APIVersion: "v1",
			Kind:       "ServiceScorecard",
			Metadata: types.Metadata{
				Name:         "test-service-" + string(rune('0'+i)),
				Organization: "test-org",
			},
			Spec: types.ServiceScorecardSpec{
				ServiceRef: "test-service-" + string(rune('0'+i)),
				Checks:     data.checks,
			},
			Status: types.ServiceScorecardStatus{
				Score:       data.score,
				Grade:       data.grade,
				EvaluatedAt: time.Now(),
			},
		}
		err := calculator.Save(scorecard)
		if err != nil {
			t.Fatalf("Failed to save scorecard %d: %v", i, err)
		}
	}

	// Get statistics
	stats, err := calculator.GetStats("test-org")
	if err != nil {
		t.Fatalf("GetStats() failed: %v", err)
	}

	// Verify total
	if stats.Total != 3 {
		t.Errorf("Expected total 3, got %d", stats.Total)
	}

	// Verify average score
	expectedAvg := (95.0 + 85.0 + 75.0) / 3.0
	if stats.AverageScore != expectedAvg {
		t.Errorf("Expected average score %.2f, got %.2f", expectedAvg, stats.AverageScore)
	}

	// Verify grade distribution
	if stats.ByGrade[types.GradeA] != 1 {
		t.Errorf("Expected 1 A grade, got %d", stats.ByGrade[types.GradeA])
	}
	if stats.ByGrade[types.GradeB] != 1 {
		t.Errorf("Expected 1 B grade, got %d", stats.ByGrade[types.GradeB])
	}
	if stats.ByGrade[types.GradeC] != 1 {
		t.Errorf("Expected 1 C grade, got %d", stats.ByGrade[types.GradeC])
	}

	// Verify category stats
	docStats := stats.ByCategory[types.CategoryDocumentation]
	if docStats.Total != 3 {
		t.Errorf("Expected 3 documentation checks, got %d", docStats.Total)
	}
	if docStats.Passed != 2 {
		t.Errorf("Expected 2 passed documentation checks, got %d", docStats.Passed)
	}
	if docStats.Warning != 1 {
		t.Errorf("Expected 1 warning documentation check, got %d", docStats.Warning)
	}

	testStats := stats.ByCategory[types.CategoryTesting]
	if testStats.Failed != 2 {
		t.Errorf("Expected 2 failed testing checks, got %d", testStats.Failed)
	}
}

func TestScorecardCalculator_RecalculateAll(t *testing.T) {
	backend := NewMockBackend()
	calculator := NewScorecardCalculator(backend)
	manager := NewManager(backend)

	// Create multiple services
	for i := 1; i <= 3; i++ {
		service := &types.Service{
			APIVersion: "v1",
			Kind:       "Service",
			Metadata: types.Metadata{
				Name:         "test-service-" + string(rune('0'+i)),
				Organization: "test-org",
			},
			Spec: types.ServiceSpec{
				Type: types.ServiceTypeMicroservice,
				Owner: types.ServiceOwner{
					Team: "platform-team",
				},
			},
			Status: types.ServiceStatus{
				State:  types.ServiceStateRunning,
				Health: types.ServiceHealthHealthy,
			},
		}
		err := manager.Create(service)
		if err != nil {
			t.Fatalf("Failed to create service %d: %v", i, err)
		}
	}

	// Recalculate all scorecards
	err := calculator.RecalculateAll("test-org")
	if err != nil {
		t.Fatalf("RecalculateAll() failed: %v", err)
	}

	// Verify scorecards were created
	scorecards, err := calculator.List("test-org")
	if err != nil {
		t.Fatalf("List() failed: %v", err)
	}

	if len(scorecards) != 3 {
		t.Errorf("Expected 3 scorecards after recalculation, got %d", len(scorecards))
	}

	// Verify each scorecard has checks
	for _, sc := range scorecards {
		if len(sc.Spec.Checks) != 11 {
			t.Errorf("Scorecard %s has %d checks, expected 11", sc.Metadata.Name, len(sc.Spec.Checks))
		}
	}
}

func TestScorecardCalculator_BuildDefaultContext(t *testing.T) {
	backend := NewMockBackend()
	calculator := NewScorecardCalculator(backend)

	lastDeployed := time.Now().Add(-2 * 24 * time.Hour) // 2 days ago

	service := &types.Service{
		Spec: types.ServiceSpec{
			Repository: &types.RepositoryConfig{
				URL: "https://github.com/test/repo",
			},
			Links: []types.ServiceLink{
				{Name: "Metrics", Type: "metrics", URL: "http://metrics"},
				{Name: "Logs", Type: "logs", URL: "http://logs"},
			},
			Dependencies: []types.ServiceDependency{
				{Name: "dep1", Type: "service"},
				{Name: "dep2", Type: "database"},
			},
		},
		Status: types.ServiceStatus{
			LastDeployed: &lastDeployed,
		},
	}

	context := calculator.buildDefaultContext(service)

	// Verify inferred values
	if !context.HasReadme {
		t.Error("Expected HasReadme to be true when repository is present")
	}

	if context.ReadmeLength != 500 {
		t.Errorf("Expected default readme length 500, got %d", context.ReadmeLength)
	}

	if !context.HasMetrics {
		t.Error("Expected HasMetrics to be true")
	}

	if !context.HasLogs {
		t.Error("Expected HasLogs to be true")
	}

	if context.DependencyCount != 2 {
		t.Errorf("Expected 2 dependencies, got %d", context.DependencyCount)
	}

	if context.LastDeployTime == nil {
		t.Error("Expected LastDeployTime to be set")
	}

	// Should have non-zero deploy frequency since recently deployed
	if context.DeployFrequency == 0 {
		t.Error("Expected non-zero deploy frequency for recent deployment")
	}
}

func TestScorecardCalculator_ResourceConversions(t *testing.T) {
	backend := NewMockBackend()
	calculator := NewScorecardCalculator(backend)

	// Create a scorecard
	scorecard := &types.ServiceScorecard{
		APIVersion: "v1",
		Kind:       "ServiceScorecard",
		Metadata: types.Metadata{
			Name:         "test-service",
			Organization: "test-org",
		},
		Spec: types.ServiceScorecardSpec{
			ServiceRef: "test-service",
			Checks: []types.Check{
				{
					Name:     "Test Check",
					Category: types.CategoryTesting,
					Weight:   10,
					Status:   types.CheckStatusPassed,
					Score:    100,
					Message:  "All good",
				},
			},
		},
		Status: types.ServiceScorecardStatus{
			Score:        85,
			Grade:        types.GradeB,
			TotalChecks:  1,
			PassedChecks: 1,
			FailedChecks: 0,
			EvaluatedAt:  time.Now(),
		},
	}

	// Convert to resource
	resource, err := calculator.scorecardToResource(scorecard)
	if err != nil {
		t.Fatalf("scorecardToResource() failed: %v", err)
	}

	if resource.Kind != "ServiceScorecard" {
		t.Errorf("Expected kind 'ServiceScorecard', got '%s'", resource.Kind)
	}

	if resource.Name != "test-org/test-service" {
		t.Errorf("Expected name 'test-org/test-service', got '%s'", resource.Name)
	}

	// Convert back from resource
	converted, err := calculator.resourceToScorecard(resource, "test-org")
	if err != nil {
		t.Fatalf("resourceToScorecard() failed: %v", err)
	}

	if converted.Metadata.Name != "test-service" {
		t.Errorf("Expected name 'test-service', got '%s'", converted.Metadata.Name)
	}

	if converted.Status.Score != 85 {
		t.Errorf("Expected score 85, got %d", converted.Status.Score)
	}

	if converted.Status.Grade != types.GradeB {
		t.Errorf("Expected grade B, got %s", converted.Status.Grade)
	}

	if len(converted.Spec.Checks) != 1 {
		t.Errorf("Expected 1 check, got %d", len(converted.Spec.Checks))
	}
}

func TestScorecardCalculator_ErrorHandling(t *testing.T) {
	backend := NewMockBackend()
	calculator := NewScorecardCalculator(backend)

	// Try to calculate for non-existent service
	_, err := calculator.Calculate("non-existent", "test-org", nil)
	if err == nil {
		t.Error("Expected error when calculating scorecard for non-existent service")
	}

	// Try to get non-existent scorecard
	_, err = calculator.Get("non-existent", "test-org")
	if err == nil {
		t.Error("Expected error when getting non-existent scorecard")
	}

	// Create a service resource (not a scorecard)
	service := &types.Service{
		APIVersion: "v1",
		Kind:       "Service",
		Metadata: types.Metadata{
			Name:         "test-service",
			Organization: "test-org",
		},
		Spec: types.ServiceSpec{
			Type: types.ServiceTypeMicroservice,
			Owner: types.ServiceOwner{
				Team: "platform-team",
			},
		},
	}

	manager := NewManager(backend)
	err = manager.Create(service)
	if err != nil {
		t.Fatalf("Failed to create service: %v", err)
	}

	// Try to get service as scorecard (should fail because it's not a scorecard resource)
	_, err = calculator.Get("test-service", "test-org")
	if err == nil {
		t.Error("Expected error when trying to get service resource as scorecard")
	}
}
