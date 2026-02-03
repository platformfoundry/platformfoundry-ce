package service

import (
	"testing"
	"time"

	"github.com/platformfoundry/pf-ce/pkg/types"
)

func TestNewScorecardEngine(t *testing.T) {
	engine := NewScorecardEngine()
	if engine == nil {
		t.Fatal("NewScorecardEngine() returned nil")
	}

	// Should have 11 default checks
	if len(engine.checks) != 11 {
		t.Errorf("Expected 11 default checks, got %d", len(engine.checks))
	}
}

func TestScorecardEngine_AddCheck(t *testing.T) {
	engine := NewScorecardEngine()
	initialCount := len(engine.checks)

	// Add a custom check
	customCheck := &ReadmeCheck{}
	engine.AddCheck(customCheck)

	if len(engine.checks) != initialCount+1 {
		t.Errorf("Expected %d checks after adding custom check, got %d", initialCount+1, len(engine.checks))
	}
}

func TestScorecardEngine_Evaluate(t *testing.T) {
	engine := NewScorecardEngine()

	// Create a test service
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

	context := &CheckContext{
		HasReadme:    true,
		ReadmeLength: 1000,
		HasTests:     true,
		TestCoverage: 80.0,
	}

	// Evaluate scorecard
	scorecard, err := engine.Evaluate(service, context)
	if err != nil {
		t.Fatalf("Evaluate() failed: %v", err)
	}

	// Verify scorecard structure
	if scorecard.Metadata.Name != "test-service" {
		t.Errorf("Expected scorecard name 'test-service', got '%s'", scorecard.Metadata.Name)
	}

	if scorecard.Spec.ServiceRef != "test-service" {
		t.Errorf("Expected service ref 'test-service', got '%s'", scorecard.Spec.ServiceRef)
	}

	if len(scorecard.Spec.Checks) != 11 {
		t.Errorf("Expected 11 checks in scorecard, got %d", len(scorecard.Spec.Checks))
	}

	if scorecard.Status.TotalChecks != 11 {
		t.Errorf("Expected total checks 11, got %d", scorecard.Status.TotalChecks)
	}

	// Score should be between 0 and 100
	if scorecard.Status.Score < 0 || scorecard.Status.Score > 100 {
		t.Errorf("Score out of range: %d", scorecard.Status.Score)
	}
}

func TestCalculateGrade(t *testing.T) {
	tests := []struct {
		score int
		grade types.ScorecardGrade
	}{
		{95, types.GradeA},
		{90, types.GradeA},
		{85, types.GradeB},
		{80, types.GradeB},
		{75, types.GradeC},
		{70, types.GradeC},
		{65, types.GradeD},
		{60, types.GradeD},
		{55, types.GradeF},
		{0, types.GradeF},
	}

	for _, tt := range tests {
		t.Run(string(tt.grade), func(t *testing.T) {
			grade := calculateGrade(tt.score)
			if grade != tt.grade {
				t.Errorf("calculateGrade(%d) = %s, want %s", tt.score, grade, tt.grade)
			}
		})
	}
}

// Test individual checks

func TestReadmeCheck_Evaluate(t *testing.T) {
	check := &ReadmeCheck{}

	service := &types.Service{
		Metadata: types.Metadata{Name: "test-service"},
	}

	tests := []struct {
		name     string
		context  *CheckContext
		status   types.CheckStatus
		minScore int
	}{
		{
			name:     "Good README",
			context:  &CheckContext{HasReadme: true, ReadmeLength: 1000},
			status:   types.CheckStatusPassed,
			minScore: 80,
		},
		{
			name:     "Short README",
			context:  &CheckContext{HasReadme: true, ReadmeLength: 50},
			status:   types.CheckStatusWarning,
			minScore: 50,
		},
		{
			name:     "No README",
			context:  &CheckContext{HasReadme: false},
			status:   types.CheckStatusFailed,
			minScore: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := check.Evaluate(service, tt.context)
			if result.Status != tt.status {
				t.Errorf("Expected status %s, got %s", tt.status, result.Status)
			}
			if result.Score < tt.minScore {
				t.Errorf("Expected score >= %d, got %d", tt.minScore, result.Score)
			}
		})
	}
}

func TestTestCoverageCheck_Evaluate(t *testing.T) {
	check := &TestCoverageCheck{}

	service := &types.Service{
		Metadata: types.Metadata{Name: "test-service"},
	}

	tests := []struct {
		name     string
		context  *CheckContext
		minScore int
	}{
		{
			name:     "Excellent coverage",
			context:  &CheckContext{HasTests: true, TestCoverage: 90.0},
			minScore: 85,
		},
		{
			name:     "Good coverage",
			context:  &CheckContext{HasTests: true, TestCoverage: 75.0},
			minScore: 65,
		},
		{
			name:     "Low coverage",
			context:  &CheckContext{HasTests: true, TestCoverage: 40.0},
			minScore: 20,
		},
		{
			name:     "No tests",
			context:  &CheckContext{HasTests: false},
			minScore: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := check.Evaluate(service, tt.context)
			if result.Score < tt.minScore {
				t.Errorf("Expected score >= %d, got %d", tt.minScore, result.Score)
			}
		})
	}
}

func TestSecurityScanCheck_Evaluate(t *testing.T) {
	check := &SecurityScanCheck{}

	service := &types.Service{
		Metadata: types.Metadata{Name: "test-service"},
	}

	tests := []struct {
		name     string
		context  *CheckContext
		status   types.CheckStatus
		minScore int
	}{
		{
			name:     "No vulnerabilities",
			context:  &CheckContext{HasSecurityScan: true, VulnCount: 0},
			status:   types.CheckStatusPassed,
			minScore: 90,
		},
		{
			name:     "Few vulnerabilities",
			context:  &CheckContext{HasSecurityScan: true, VulnCount: 3},
			status:   types.CheckStatusWarning,
			minScore: 50,
		},
		{
			name:     "Many vulnerabilities",
			context:  &CheckContext{HasSecurityScan: true, VulnCount: 15},
			status:   types.CheckStatusFailed,
			minScore: 0,
		},
		{
			name:     "No scan",
			context:  &CheckContext{HasSecurityScan: false},
			status:   types.CheckStatusFailed,
			minScore: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := check.Evaluate(service, tt.context)
			if result.Status != tt.status {
				t.Errorf("Expected status %s, got %s", tt.status, result.Status)
			}
			if result.Score < tt.minScore {
				t.Errorf("Expected score >= %d, got %d", tt.minScore, result.Score)
			}
		})
	}
}

func TestDependencyCheck_Evaluate(t *testing.T) {
	check := &DependencyCheck{}

	service := &types.Service{
		Metadata: types.Metadata{Name: "test-service"},
	}

	tests := []struct {
		name     string
		context  *CheckContext
		status   types.CheckStatus
		minScore int
	}{
		{
			name:     "No outdated dependencies",
			context:  &CheckContext{DependencyCount: 10, OutdatedDeps: 0},
			status:   types.CheckStatusPassed,
			minScore: 90,
		},
		{
			name:     "Few outdated",
			context:  &CheckContext{DependencyCount: 10, OutdatedDeps: 2},
			status:   types.CheckStatusPassed, // 20% is NOT > 20%, so returns Passed
			minScore: 90,
		},
		{
			name:     "Some outdated",
			context:  &CheckContext{DependencyCount: 10, OutdatedDeps: 3},
			status:   types.CheckStatusWarning, // 30% > 20%, returns Warning
			minScore: 60,
		},
		{
			name:     "Many outdated",
			context:  &CheckContext{DependencyCount: 10, OutdatedDeps: 8},
			status:   types.CheckStatusFailed,
			minScore: 0,
		},
		{
			name:     "No dependencies",
			context:  &CheckContext{DependencyCount: 0},
			status:   types.CheckStatusPassed, // Returns Passed for no dependencies
			minScore: 90,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := check.Evaluate(service, tt.context)
			if result.Status != tt.status {
				t.Errorf("Expected status %s, got %s", tt.status, result.Status)
			}
			if result.Score < tt.minScore {
				t.Errorf("Expected score >= %d, got %d", tt.minScore, result.Score)
			}
		})
	}
}

func TestSLODefinitionCheck_Evaluate(t *testing.T) {
	check := &SLODefinitionCheck{}

	tests := []struct {
		name     string
		service  *types.Service
		status   types.CheckStatus
		minScore int
	}{
		{
			name: "Complete SLO",
			service: &types.Service{
				Metadata: types.Metadata{Name: "test-service"},
				Spec: types.ServiceSpec{
					SLO: &types.SLOConfig{
						Availability: 99.9,
						Latency:      &types.LatencySLO{P95: 200, P99: 500},
						ErrorRate:    0.1,
					},
				},
			},
			status:   types.CheckStatusPassed,
			minScore: 90,
		},
		{
			name: "Partial SLO",
			service: &types.Service{
				Metadata: types.Metadata{Name: "test-service"},
				Spec: types.ServiceSpec{
					SLO: &types.SLOConfig{
						Availability: 99.9,
					},
				},
			},
			status:   types.CheckStatusWarning,
			minScore: 50,
		},
		{
			name: "No SLO",
			service: &types.Service{
				Metadata: types.Metadata{Name: "test-service"},
				Spec:     types.ServiceSpec{},
			},
			status:   types.CheckStatusFailed,
			minScore: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := check.Evaluate(tt.service, &CheckContext{})
			if result.Status != tt.status {
				t.Errorf("Expected status %s, got %s", tt.status, result.Status)
			}
			if result.Score < tt.minScore {
				t.Errorf("Expected score >= %d, got %d", tt.minScore, result.Score)
			}
		})
	}
}

func TestHealthCheck_Evaluate(t *testing.T) {
	check := &HealthCheck{}

	tests := []struct {
		name     string
		service  *types.Service
		status   types.CheckStatus
		minScore int
	}{
		{
			name: "Healthy service",
			service: &types.Service{
				Metadata: types.Metadata{Name: "test-service"},
				Status: types.ServiceStatus{
					Health: types.ServiceHealthHealthy,
				},
			},
			status:   types.CheckStatusPassed,
			minScore: 90,
		},
		{
			name: "Degraded service",
			service: &types.Service{
				Metadata: types.Metadata{Name: "test-service"},
				Status: types.ServiceStatus{
					Health: types.ServiceHealthDegraded,
				},
			},
			status:   types.CheckStatusWarning,
			minScore: 50,
		},
		{
			name: "Unhealthy service",
			service: &types.Service{
				Metadata: types.Metadata{Name: "test-service"},
				Status: types.ServiceStatus{
					Health: types.ServiceHealthDown,
				},
			},
			status:   types.CheckStatusFailed,
			minScore: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := check.Evaluate(tt.service, &CheckContext{})
			if result.Status != tt.status {
				t.Errorf("Expected status %s, got %s", tt.status, result.Status)
			}
			if result.Score < tt.minScore {
				t.Errorf("Expected score >= %d, got %d", tt.minScore, result.Score)
			}
		})
	}
}

func TestDeploymentFrequencyCheck_Evaluate(t *testing.T) {
	check := &DeploymentFrequencyCheck{}

	service := &types.Service{
		Metadata: types.Metadata{Name: "test-service"},
	}

	// Recent deployment time for tests that check DeployFrequency
	recentDeploy := time.Now().Add(-24 * time.Hour)

	tests := []struct {
		name     string
		context  *CheckContext
		status   types.CheckStatus
		minScore int
	}{
		{
			name:     "Frequent deployments",
			context:  &CheckContext{LastDeployTime: &recentDeploy, DeployFrequency: 10.0}, // >= 5/week
			status:   types.CheckStatusPassed,
			minScore: 90,
		},
		{
			name:     "Moderate deployments",
			context:  &CheckContext{LastDeployTime: &recentDeploy, DeployFrequency: 3.0}, // 0.5 <= x < 5
			status:   types.CheckStatusPassed,
			minScore: 70,
		},
		{
			name:     "Infrequent deployments",
			context:  &CheckContext{LastDeployTime: &recentDeploy, DeployFrequency: 0.3}, // < 0.5
			status:   types.CheckStatusWarning,
			minScore: 50,
		},
		{
			name:     "No deployment history",
			context:  &CheckContext{DeployFrequency: 0}, // LastDeployTime is nil
			status:   types.CheckStatusWarning,          // Implementation returns Warning for no deploy history
			minScore: 40,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := check.Evaluate(service, tt.context)
			if result.Status != tt.status {
				t.Errorf("Expected status %s, got %s", tt.status, result.Status)
			}
			if result.Score < tt.minScore {
				t.Errorf("Expected score >= %d, got %d", tt.minScore, result.Score)
			}
		})
	}
}

func TestOwnershipCheck_Evaluate(t *testing.T) {
	check := &OwnershipCheck{}

	tests := []struct {
		name     string
		service  *types.Service
		status   types.CheckStatus
		minScore int
	}{
		{
			name: "Full ownership",
			service: &types.Service{
				Metadata: types.Metadata{Name: "test-service"},
				Spec: types.ServiceSpec{
					Owner: types.ServiceOwner{
						Team:  "platform-team",
						Email: "platform@example.com",
						Slack: "#platform-alerts",
					},
				},
			},
			status:   types.CheckStatusPassed,
			minScore: 90, // Team(40) + Email(30) + Slack(30) = 100
		},
		{
			name: "Team and email only",
			service: &types.Service{
				Metadata: types.Metadata{Name: "test-service"},
				Spec: types.ServiceSpec{
					Owner: types.ServiceOwner{
						Team:  "platform-team",
						Email: "platform@example.com",
					},
				},
			},
			status:   types.CheckStatusPassed, // Score 70, which is >= 70 threshold
			minScore: 60,                      // Team(40) + Email(30) = 70
		},
		{
			name: "Team only",
			service: &types.Service{
				Metadata: types.Metadata{Name: "test-service"},
				Spec: types.ServiceSpec{
					Owner: types.ServiceOwner{
						Team: "platform-team",
					},
				},
			},
			status:   types.CheckStatusWarning, // Score 40, which is < 70
			minScore: 30,                       // Team(40) only
		},
		{
			name: "No ownership",
			service: &types.Service{
				Metadata: types.Metadata{Name: "test-service"},
				Spec:     types.ServiceSpec{},
			},
			status:   types.CheckStatusFailed,
			minScore: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := check.Evaluate(tt.service, &CheckContext{})
			if result.Status != tt.status {
				t.Errorf("Expected status %s, got %s", tt.status, result.Status)
			}
			if result.Score < tt.minScore {
				t.Errorf("Expected score >= %d, got %d", tt.minScore, result.Score)
			}
		})
	}
}

func TestObservabilityCheck_Evaluate(t *testing.T) {
	check := &ObservabilityCheck{}

	service := &types.Service{
		Metadata: types.Metadata{Name: "test-service"},
	}

	tests := []struct {
		name     string
		context  *CheckContext
		status   types.CheckStatus
		minScore int
	}{
		{
			name: "Full observability",
			context: &CheckContext{
				HasMetrics: true,
				HasLogs:    true,
				HasTraces:  true,
				HasAlerts:  true,
			},
			status:   types.CheckStatusPassed,
			minScore: 90,
		},
		{
			name: "Partial observability",
			context: &CheckContext{
				HasMetrics: true,
				HasLogs:    true,
			},
			status:   types.CheckStatusWarning,
			minScore: 40,
		},
		{
			name:     "No observability",
			context:  &CheckContext{},
			status:   types.CheckStatusFailed,
			minScore: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := check.Evaluate(service, tt.context)
			if result.Status != tt.status {
				t.Errorf("Expected status %s, got %s", tt.status, result.Status)
			}
			if result.Score < tt.minScore {
				t.Errorf("Expected score >= %d, got %d", tt.minScore, result.Score)
			}
		})
	}
}

func TestMTTRCheck_Evaluate(t *testing.T) {
	check := &MTTRCheck{}

	service := &types.Service{
		Metadata: types.Metadata{Name: "test-service"},
	}

	thirtyMin := 30 * time.Minute
	threeHours := 3 * time.Hour
	tenHours := 10 * time.Hour

	tests := []struct {
		name     string
		context  *CheckContext
		status   types.CheckStatus
		minScore int
	}{
		{
			name:     "Excellent MTTR",
			context:  &CheckContext{MTTR: &thirtyMin}, // <= 1 hour → Passed
			status:   types.CheckStatusPassed,
			minScore: 90,
		},
		{
			name:     "Moderate MTTR",
			context:  &CheckContext{MTTR: &threeHours}, // 1 < x <= 4 hours → Warning score 70
			status:   types.CheckStatusWarning,
			minScore: 60,
		},
		{
			name:     "High MTTR",
			context:  &CheckContext{MTTR: &tenHours}, // > 4 hours → Warning score 40
			status:   types.CheckStatusWarning,
			minScore: 30,
		},
		{
			name:     "No MTTR data",
			context:  &CheckContext{},
			status:   types.CheckStatusInfo,
			minScore: 60,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := check.Evaluate(service, tt.context)
			if result.Status != tt.status {
				t.Errorf("Expected status %s, got %s", tt.status, result.Status)
			}
			if result.Score < tt.minScore {
				t.Errorf("Expected score >= %d, got %d", tt.minScore, result.Score)
			}
		})
	}
}
