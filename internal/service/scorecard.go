package service

import (
	"fmt"
	"strings"
	"time"

	"github.com/platformfoundry/pf-ce/pkg/types"
)

// ScorecardEngine evaluates services against quality checks
type ScorecardEngine struct {
	checks []Check
}

// Check represents a scorecard check
type Check interface {
	Name() string
	Category() types.CheckCategory
	Weight() int
	Evaluate(service *types.Service, context *CheckContext) types.CheckResult
}

// CheckContext provides additional context for check evaluation
type CheckContext struct {
	// Repository information
	HasReadme       bool
	ReadmeLength    int
	HasTests        bool
	TestCoverage    float64
	HasCI           bool

	// Security information
	HasSecurityScan bool
	VulnCount       int

	// Dependencies
	DependencyCount int
	OutdatedDeps    int

	// Deployment information
	LastDeployTime  *time.Time
	DeployFrequency float64 // deploys per week
	MTTR            *time.Duration

	// Observability
	HasMetrics      bool
	HasLogs         bool
	HasTraces       bool
	HasAlerts       bool
	AlertCount      int
}

// NewScorecardEngine creates a new scorecard engine with default checks
func NewScorecardEngine() *ScorecardEngine {
	return &ScorecardEngine{
		checks: []Check{
			&ReadmeCheck{},
			&TestCoverageCheck{},
			&SecurityScanCheck{},
			&DependencyCheck{},
			&SLODefinitionCheck{},
			&ObservabilityCheck{},
			&DeploymentFrequencyCheck{},
			&MTTRCheck{},
			&DocumentationCheck{},
			&OwnershipCheck{},
			&HealthCheck{},
		},
	}
}

// AddCheck adds a custom check to the engine
func (se *ScorecardEngine) AddCheck(check Check) {
	se.checks = append(se.checks, check)
}

// Evaluate runs all checks against a service and returns a scorecard
func (se *ScorecardEngine) Evaluate(service *types.Service, context *CheckContext) (*types.ServiceScorecard, error) {
	if service == nil {
		return nil, fmt.Errorf("service cannot be nil")
	}

	if context == nil {
		context = &CheckContext{}
	}

	scorecard := &types.ServiceScorecard{
		APIVersion: "platformfoundry.io/v1",
		Kind:       "ServiceScorecard",
		Metadata: types.Metadata{
			Name:         service.Metadata.Name,
			Organization: service.Metadata.Organization,
			Labels:       make(map[string]string),
		},
		Spec: types.ServiceScorecardSpec{
			ServiceRef: service.Metadata.Name,
			Checks:     make([]types.Check, 0, len(se.checks)),
		},
		Status: types.ServiceScorecardStatus{
			EvaluatedAt: time.Now(),
		},
	}

	// Run all checks
	totalScore := 0
	totalWeight := 0
	passedCount := 0
	failedCount := 0

	for _, check := range se.checks {
		result := check.Evaluate(service, context)

		checkObj := types.Check{
			Name:        check.Name(),
			Category:    check.Category(),
			Weight:      check.Weight(),
			Status:      result.Status,
			Score:       result.Score,
			Message:     result.Message,
			Details:     result.Details,
			EvaluatedAt: time.Now(),
		}

		scorecard.Spec.Checks = append(scorecard.Spec.Checks, checkObj)

		totalScore += result.Score * check.Weight()
		totalWeight += check.Weight()

		if result.Status == types.CheckStatusPassed {
			passedCount++
		} else if result.Status == types.CheckStatusFailed {
			failedCount++
		}
	}

	// Calculate overall score (0-100)
	if totalWeight > 0 {
		scorecard.Status.Score = totalScore / totalWeight
	}

	// Assign grade
	scorecard.Status.Grade = calculateGrade(scorecard.Status.Score)
	scorecard.Status.PassedChecks = passedCount
	scorecard.Status.FailedChecks = failedCount
	scorecard.Status.TotalChecks = len(se.checks)

	return scorecard, nil
}

// calculateGrade assigns a letter grade based on score
func calculateGrade(score int) types.ScorecardGrade {
	if score >= 90 {
		return types.GradeA
	} else if score >= 80 {
		return types.GradeB
	} else if score >= 70 {
		return types.GradeC
	} else if score >= 60 {
		return types.GradeD
	}
	return types.GradeF
}

// ========== Built-in Checks ==========

// ReadmeCheck verifies presence and quality of README
type ReadmeCheck struct{}

func (c *ReadmeCheck) Name() string                  { return "README Documentation" }
func (c *ReadmeCheck) Category() types.CheckCategory { return types.CategoryDocumentation }
func (c *ReadmeCheck) Weight() int                   { return 10 }

func (c *ReadmeCheck) Evaluate(service *types.Service, context *CheckContext) types.CheckResult {
	if !context.HasReadme {
		return types.CheckResult{
			Status:  types.CheckStatusFailed,
			Score:   0,
			Message: "README.md not found",
			Details: "Add a README.md file to document your service",
		}
	}

	if context.ReadmeLength < 100 {
		return types.CheckResult{
			Status:  types.CheckStatusWarning,
			Score:   50,
			Message: "README is too short",
			Details: fmt.Sprintf("README is only %d characters. Add more documentation.", context.ReadmeLength),
		}
	}

	return types.CheckResult{
		Status:  types.CheckStatusPassed,
		Score:   100,
		Message: "README documentation present",
		Details: fmt.Sprintf("README contains %d characters of documentation", context.ReadmeLength),
	}
}

// TestCoverageCheck evaluates test coverage
type TestCoverageCheck struct{}

func (c *TestCoverageCheck) Name() string                  { return "Test Coverage" }
func (c *TestCoverageCheck) Category() types.CheckCategory { return types.CategoryQuality }
func (c *TestCoverageCheck) Weight() int                   { return 15 }

func (c *TestCoverageCheck) Evaluate(service *types.Service, context *CheckContext) types.CheckResult {
	if !context.HasTests {
		return types.CheckResult{
			Status:  types.CheckStatusFailed,
			Score:   0,
			Message: "No tests found",
			Details: "Add unit tests for your service",
		}
	}

	if context.TestCoverage < 50 {
		return types.CheckResult{
			Status:  types.CheckStatusFailed,
			Score:   int(context.TestCoverage),
			Message: fmt.Sprintf("Test coverage too low: %.1f%%", context.TestCoverage),
			Details: "Aim for at least 80% test coverage",
		}
	} else if context.TestCoverage < 80 {
		return types.CheckResult{
			Status:  types.CheckStatusWarning,
			Score:   int(context.TestCoverage),
			Message: fmt.Sprintf("Test coverage: %.1f%%", context.TestCoverage),
			Details: "Good progress! Aim for 80%+ coverage",
		}
	}

	return types.CheckResult{
		Status:  types.CheckStatusPassed,
		Score:   100,
		Message: fmt.Sprintf("Excellent test coverage: %.1f%%", context.TestCoverage),
		Details: "Test coverage meets or exceeds 80%",
	}
}

// SecurityScanCheck evaluates security posture
type SecurityScanCheck struct{}

func (c *SecurityScanCheck) Name() string                  { return "Security Scanning" }
func (c *SecurityScanCheck) Category() types.CheckCategory { return types.CategorySecurity }
func (c *SecurityScanCheck) Weight() int                   { return 20 }

func (c *SecurityScanCheck) Evaluate(service *types.Service, context *CheckContext) types.CheckResult {
	if !context.HasSecurityScan {
		return types.CheckResult{
			Status:  types.CheckStatusFailed,
			Score:   0,
			Message: "No security scan configured",
			Details: "Enable automated security scanning (e.g., Snyk, Trivy)",
		}
	}

	if context.VulnCount > 10 {
		return types.CheckResult{
			Status:  types.CheckStatusFailed,
			Score:   20,
			Message: fmt.Sprintf("%d vulnerabilities found", context.VulnCount),
			Details: "Critical: Fix high/critical vulnerabilities immediately",
		}
	} else if context.VulnCount > 0 {
		return types.CheckResult{
			Status:  types.CheckStatusWarning,
			Score:   70,
			Message: fmt.Sprintf("%d vulnerabilities found", context.VulnCount),
			Details: "Address vulnerabilities in dependencies",
		}
	}

	return types.CheckResult{
		Status:  types.CheckStatusPassed,
		Score:   100,
		Message: "No vulnerabilities detected",
		Details: "Security scan passed with no issues",
	}
}

// DependencyCheck evaluates dependency management
type DependencyCheck struct{}

func (c *DependencyCheck) Name() string                  { return "Dependency Health" }
func (c *DependencyCheck) Category() types.CheckCategory { return types.CategoryQuality }
func (c *DependencyCheck) Weight() int                   { return 10 }

func (c *DependencyCheck) Evaluate(service *types.Service, context *CheckContext) types.CheckResult {
	if context.DependencyCount == 0 {
		return types.CheckResult{
			Status:  types.CheckStatusPassed,
			Score:   100,
			Message: "No external dependencies",
			Details: "Service has no external dependencies to manage",
		}
	}

	outdatedPercent := float64(context.OutdatedDeps) / float64(context.DependencyCount) * 100

	if outdatedPercent > 50 {
		return types.CheckResult{
			Status:  types.CheckStatusFailed,
			Score:   30,
			Message: fmt.Sprintf("%d/%d dependencies outdated (%.0f%%)", context.OutdatedDeps, context.DependencyCount, outdatedPercent),
			Details: "Update dependencies to latest stable versions",
		}
	} else if outdatedPercent > 20 {
		return types.CheckResult{
			Status:  types.CheckStatusWarning,
			Score:   70,
			Message: fmt.Sprintf("%d/%d dependencies outdated (%.0f%%)", context.OutdatedDeps, context.DependencyCount, outdatedPercent),
			Details: "Consider updating some dependencies",
		}
	}

	return types.CheckResult{
		Status:  types.CheckStatusPassed,
		Score:   100,
		Message: "Dependencies up to date",
		Details: fmt.Sprintf("All %d dependencies are current", context.DependencyCount),
	}
}

// SLODefinitionCheck verifies SLO configuration
type SLODefinitionCheck struct{}

func (c *SLODefinitionCheck) Name() string                  { return "SLO Definition" }
func (c *SLODefinitionCheck) Category() types.CheckCategory { return types.CategoryReliability }
func (c *SLODefinitionCheck) Weight() int                   { return 15 }

func (c *SLODefinitionCheck) Evaluate(service *types.Service, context *CheckContext) types.CheckResult {
	if service.Spec.SLO == nil {
		return types.CheckResult{
			Status:  types.CheckStatusFailed,
			Score:   0,
			Message: "No SLO defined",
			Details: "Define Service Level Objectives (availability, latency, error rate)",
		}
	}

	slo := service.Spec.SLO
	hasAvailability := slo.Availability > 0
	hasLatency := slo.Latency != nil
	hasErrorRate := slo.ErrorRate > 0

	definedCount := 0
	if hasAvailability {
		definedCount++
	}
	if hasLatency {
		definedCount++
	}
	if hasErrorRate {
		definedCount++
	}

	if definedCount == 0 {
		return types.CheckResult{
			Status:  types.CheckStatusFailed,
			Score:   0,
			Message: "SLO defined but no targets set",
			Details: "Set availability, latency, or error rate targets",
		}
	} else if definedCount < 3 {
		return types.CheckResult{
			Status:  types.CheckStatusWarning,
			Score:   50 + (definedCount * 15),
			Message: fmt.Sprintf("Partial SLO definition (%d/3 metrics)", definedCount),
			Details: "Consider defining all three SLO metrics",
		}
	}

	return types.CheckResult{
		Status:  types.CheckStatusPassed,
		Score:   100,
		Message: "Complete SLO definition",
		Details: "All SLO metrics (availability, latency, error rate) defined",
	}
}

// ObservabilityCheck evaluates observability setup
type ObservabilityCheck struct{}

func (c *ObservabilityCheck) Name() string                  { return "Observability" }
func (c *ObservabilityCheck) Category() types.CheckCategory { return types.CategoryObservability }
func (c *ObservabilityCheck) Weight() int                   { return 15 }

func (c *ObservabilityCheck) Evaluate(service *types.Service, context *CheckContext) types.CheckResult {
	signals := []bool{context.HasMetrics, context.HasLogs, context.HasTraces}
	signalCount := 0
	for _, has := range signals {
		if has {
			signalCount++
		}
	}

	if signalCount == 0 {
		return types.CheckResult{
			Status:  types.CheckStatusFailed,
			Score:   0,
			Message: "No observability configured",
			Details: "Enable metrics, logs, and/or tracing",
		}
	}

	var missing []string
	if !context.HasMetrics {
		missing = append(missing, "metrics")
	}
	if !context.HasLogs {
		missing = append(missing, "logs")
	}
	if !context.HasTraces {
		missing = append(missing, "traces")
	}

	if signalCount < 3 {
		return types.CheckResult{
			Status:  types.CheckStatusWarning,
			Score:   signalCount * 33,
			Message: fmt.Sprintf("Partial observability (%d/3 signals)", signalCount),
			Details: fmt.Sprintf("Missing: %s", strings.Join(missing, ", ")),
		}
	}

	if !context.HasAlerts {
		return types.CheckResult{
			Status:  types.CheckStatusWarning,
			Score:   90,
			Message: "Full observability but no alerts",
			Details: "Configure alerts for critical metrics",
		}
	}

	return types.CheckResult{
		Status:  types.CheckStatusPassed,
		Score:   100,
		Message: fmt.Sprintf("Full observability with %d alerts", context.AlertCount),
		Details: "Metrics, logs, traces, and alerts configured",
	}
}

// DeploymentFrequencyCheck evaluates deployment cadence
type DeploymentFrequencyCheck struct{}

func (c *DeploymentFrequencyCheck) Name() string                  { return "Deployment Frequency" }
func (c *DeploymentFrequencyCheck) Category() types.CheckCategory { return types.CategoryDelivery }
func (c *DeploymentFrequencyCheck) Weight() int                   { return 10 }

func (c *DeploymentFrequencyCheck) Evaluate(service *types.Service, context *CheckContext) types.CheckResult {
	if context.LastDeployTime == nil {
		return types.CheckResult{
			Status:  types.CheckStatusWarning,
			Score:   50,
			Message: "No deployment history",
			Details: "Service has not been deployed yet",
		}
	}

	// Check if service hasn't been deployed in 30 days
	daysSinceLastDeploy := time.Since(*context.LastDeployTime).Hours() / 24
	if daysSinceLastDeploy > 30 {
		return types.CheckResult{
			Status:  types.CheckStatusWarning,
			Score:   40,
			Message: fmt.Sprintf("Last deployed %.0f days ago", daysSinceLastDeploy),
			Details: "Consider increasing deployment frequency",
		}
	}

	// Evaluate deployment frequency (deploys per week)
	if context.DeployFrequency < 0.5 {
		return types.CheckResult{
			Status:  types.CheckStatusWarning,
			Score:   60,
			Message: fmt.Sprintf("Low deployment frequency: %.1f/week", context.DeployFrequency),
			Details: "Aim for at least 1 deploy per week",
		}
	} else if context.DeployFrequency >= 5 {
		return types.CheckResult{
			Status:  types.CheckStatusPassed,
			Score:   100,
			Message: fmt.Sprintf("Excellent deployment frequency: %.1f/week", context.DeployFrequency),
			Details: "Deploying multiple times per week",
		}
	}

	return types.CheckResult{
		Status:  types.CheckStatusPassed,
		Score:   80,
		Message: fmt.Sprintf("Good deployment frequency: %.1f/week", context.DeployFrequency),
		Details: "Regular deployment cadence",
	}
}

// MTTRCheck evaluates mean time to recovery
type MTTRCheck struct{}

func (c *MTTRCheck) Name() string                  { return "Mean Time to Recovery" }
func (c *MTTRCheck) Category() types.CheckCategory { return types.CategoryReliability }
func (c *MTTRCheck) Weight() int                   { return 10 }

func (c *MTTRCheck) Evaluate(service *types.Service, context *CheckContext) types.CheckResult {
	if context.MTTR == nil {
		return types.CheckResult{
			Status:  types.CheckStatusInfo,
			Score:   75,
			Message: "No incident history",
			Details: "No incidents recorded yet",
		}
	}

	mttrHours := context.MTTR.Hours()

	if mttrHours > 4 {
		return types.CheckResult{
			Status:  types.CheckStatusWarning,
			Score:   40,
			Message: fmt.Sprintf("High MTTR: %.1f hours", mttrHours),
			Details: "Focus on reducing recovery time. Target: <1 hour",
		}
	} else if mttrHours > 1 {
		return types.CheckResult{
			Status:  types.CheckStatusWarning,
			Score:   70,
			Message: fmt.Sprintf("Moderate MTTR: %.1f hours", mttrHours),
			Details: "Good progress. Aim for <1 hour MTTR",
		}
	}

	return types.CheckResult{
		Status:  types.CheckStatusPassed,
		Score:   100,
		Message: fmt.Sprintf("Excellent MTTR: %.0f minutes", mttrHours*60),
		Details: "Recovery time is under 1 hour",
	}
}

// DocumentationCheck evaluates overall documentation
type DocumentationCheck struct{}

func (c *DocumentationCheck) Name() string                  { return "Documentation" }
func (c *DocumentationCheck) Category() types.CheckCategory { return types.CategoryDocumentation }
func (c *DocumentationCheck) Weight() int                   { return 10 }

func (c *DocumentationCheck) Evaluate(service *types.Service, context *CheckContext) types.CheckResult {
	docScore := 0
	details := []string{}

	// Check for documentation links
	hasDocsLink := false
	for _, link := range service.Spec.Links {
		if link.Type == "docs" || link.Type == "documentation" {
			hasDocsLink = true
			break
		}
	}

	if hasDocsLink {
		docScore += 25
		details = append(details, "✓ Documentation link provided")
	} else {
		details = append(details, "✗ No documentation link")
	}

	if len(service.Spec.Links) > 0 {
		docScore += 25
		details = append(details, fmt.Sprintf("✓ %d links", len(service.Spec.Links)))
	} else {
		details = append(details, "✗ No links")
	}

	if context.HasReadme {
		docScore += 25
		details = append(details, "✓ README present")
	} else {
		details = append(details, "✗ No README")
	}

	if context.HasCI {
		docScore += 25
		details = append(details, "✓ CI/CD documented")
	} else {
		details = append(details, "✗ No CI/CD documentation")
	}

	status := types.CheckStatusPassed
	if docScore < 50 {
		status = types.CheckStatusFailed
	} else if docScore < 75 {
		status = types.CheckStatusWarning
	}

	return types.CheckResult{
		Status:  status,
		Score:   docScore,
		Message: fmt.Sprintf("Documentation score: %d/100", docScore),
		Details: strings.Join(details, "\n"),
	}
}

// OwnershipCheck validates ownership information
type OwnershipCheck struct{}

func (c *OwnershipCheck) Name() string                  { return "Ownership" }
func (c *OwnershipCheck) Category() types.CheckCategory { return types.CategoryGovernance }
func (c *OwnershipCheck) Weight() int                   { return 10 }

func (c *OwnershipCheck) Evaluate(service *types.Service, context *CheckContext) types.CheckResult {
	ownerScore := 0
	details := []string{}

	if service.Spec.Owner.Team != "" {
		ownerScore += 40
		details = append(details, fmt.Sprintf("✓ Team: %s", service.Spec.Owner.Team))
	} else {
		details = append(details, "✗ No team assigned")
	}

	if service.Spec.Owner.Email != "" {
		ownerScore += 30
		details = append(details, fmt.Sprintf("✓ Contact: %s", service.Spec.Owner.Email))
	} else {
		details = append(details, "✗ No contact email")
	}

	if service.Spec.Owner.Slack != "" {
		ownerScore += 30
		details = append(details, fmt.Sprintf("✓ Slack: %s", service.Spec.Owner.Slack))
	} else {
		details = append(details, "✗ No Slack channel")
	}

	status := types.CheckStatusPassed
	if ownerScore < 40 {
		status = types.CheckStatusFailed
	} else if ownerScore < 70 {
		status = types.CheckStatusWarning
	}

	return types.CheckResult{
		Status:  status,
		Score:   ownerScore,
		Message: fmt.Sprintf("Ownership information: %d%%", ownerScore),
		Details: strings.Join(details, "\n"),
	}
}

// HealthCheck evaluates current service health
type HealthCheck struct{}

func (c *HealthCheck) Name() string                  { return "Service Health" }
func (c *HealthCheck) Category() types.CheckCategory { return types.CategoryReliability }
func (c *HealthCheck) Weight() int                   { return 5 }

func (c *HealthCheck) Evaluate(service *types.Service, context *CheckContext) types.CheckResult {
	switch service.Status.Health {
	case types.ServiceHealthHealthy:
		return types.CheckResult{
			Status:  types.CheckStatusPassed,
			Score:   100,
			Message: "Service is healthy",
			Details: fmt.Sprintf("Current state: %s", service.Status.State),
		}
	case types.ServiceHealthDegraded:
		return types.CheckResult{
			Status:  types.CheckStatusWarning,
			Score:   50,
			Message: "Service is degraded",
			Details: service.Status.Message,
		}
	case types.ServiceHealthDown:
		return types.CheckResult{
			Status:  types.CheckStatusFailed,
			Score:   0,
			Message: "Service is down",
			Details: service.Status.Message,
		}
	default:
		return types.CheckResult{
			Status:  types.CheckStatusInfo,
			Score:   75,
			Message: "Health status unknown",
			Details: "Service health has not been reported",
		}
	}
}
