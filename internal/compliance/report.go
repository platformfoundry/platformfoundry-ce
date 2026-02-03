package compliance

import (
	"context"
	"fmt"
	"sort"
	"time"
)

// ComplianceStatus represents the compliance status
type ComplianceStatus string

const (
	ComplianceStatusPassed        ComplianceStatus = "passed"
	ComplianceStatusFailed        ComplianceStatus = "failed"
	ComplianceStatusPartial       ComplianceStatus = "partial"
	ComplianceStatusNotApplicable ComplianceStatus = "not_applicable"
	ComplianceStatusNotAssessed   ComplianceStatus = "not_assessed"
)

// Severity represents finding severity
type Severity string

const (
	SeverityCritical Severity = "critical"
	SeverityHigh     Severity = "high"
	SeverityMedium   Severity = "medium"
	SeverityLow      Severity = "low"
	SeverityInfo     Severity = "info"
)

// ReportPeriod represents the time period for a report
type ReportPeriod struct {
	Start time.Time `json:"start"`
	End   time.Time `json:"end"`
}

// CheckResult represents the result of a check evaluation
type CheckResult struct {
	CheckID       string           `json:"checkId"`
	CheckName     string           `json:"checkName"`
	Category      string           `json:"category"`
	Status        ComplianceStatus `json:"status"`
	Severity      Severity         `json:"severity,omitempty"`
	Score         float64          `json:"score"`
	FailureReason string           `json:"failureReason,omitempty"`
	Remediation   string           `json:"remediation,omitempty"`
	Evidence      []Evidence       `json:"evidence,omitempty"`
	EvaluatedAt   time.Time        `json:"evaluatedAt"`
}

// Finding represents a compliance finding
type Finding struct {
	ID          string    `json:"id"`
	CheckID     string    `json:"checkId"`
	Title       string    `json:"title"`
	Description string    `json:"description"`
	Severity    Severity  `json:"severity"`
	Remediation string    `json:"remediation"`
	DetectedAt  time.Time `json:"detectedAt"`
	Status      string    `json:"status"` // open, remediated, accepted
}

// Recommendation represents a compliance recommendation
type Recommendation struct {
	CheckID     string   `json:"checkId"`
	Priority    int      `json:"priority"`
	Title       string   `json:"title"`
	Description string   `json:"description"`
	Impact      string   `json:"impact"`
	Effort      string   `json:"effort"`
	Steps       []string `json:"steps,omitempty"`
}

// AuditReport represents a comprehensive audit report
type AuditReport struct {
	ID              string            `json:"id"`
	Framework       Framework         `json:"framework"`
	GeneratedAt     time.Time         `json:"generatedAt"`
	GeneratedBy     string            `json:"generatedBy"`
	Period          ReportPeriod      `json:"period"`
	OverallScore    float64           `json:"overallScore"`
	OverallStatus   ComplianceStatus  `json:"overallStatus"`
	CheckResults    []CheckResult     `json:"checkResults"`
	Findings        []Finding         `json:"findings"`
	Recommendations []Recommendation  `json:"recommendations"`
	Summary         AuditSummary      `json:"summary"`
	Metadata        map[string]string `json:"metadata,omitempty"`
}

// AuditSummary contains summary statistics for the report
type AuditSummary struct {
	TotalChecks       int     `json:"totalChecks"`
	PassedChecks      int     `json:"passedChecks"`
	FailedChecks      int     `json:"failedChecks"`
	WarningChecks     int     `json:"warningChecks"`
	NotAssessed       int     `json:"notAssessed"`
	CriticalFindings  int     `json:"criticalFindings"`
	HighFindings      int     `json:"highFindings"`
	MediumFindings    int     `json:"mediumFindings"`
	LowFindings       int     `json:"lowFindings"`
	CompliancePercent float64 `json:"compliancePercent"`
}

// ReportOpts contains options for generating a report
type ReportOpts struct {
	Framework       Framework    `json:"framework"`
	Period          ReportPeriod `json:"period"`
	IncludeEvidence bool         `json:"includeEvidence"`
	CheckFilter     []string     `json:"checkFilter,omitempty"`
	CategoryFilter  []string     `json:"categoryFilter,omitempty"`
}

// TemplateEngineInterface interface for rendering reports
type TemplateEngineInterface interface {
	RenderPDF(template string, data interface{}) ([]byte, error)
	RenderHTML(template string, data interface{}) (string, error)
}

// ReportGenerator generates compliance audit reports
type ReportGenerator struct {
	evidenceCollector *EvidenceCollector
	manager           *Manager
	templateEngine    TemplateEngineInterface
}

// NewReportGenerator creates a new report generator
func NewReportGenerator(collector *EvidenceCollector, mgr *Manager, templates TemplateEngineInterface) *ReportGenerator {
	return &ReportGenerator{
		evidenceCollector: collector,
		manager:           mgr,
		templateEngine:    templates,
	}
}

// Generate generates a compliance audit report
func (g *ReportGenerator) Generate(ctx context.Context, opts ReportOpts) (*AuditReport, error) {
	report := &AuditReport{
		ID:              fmt.Sprintf("report-%s-%d", opts.Framework, time.Now().Unix()),
		Framework:       opts.Framework,
		GeneratedAt:     time.Now(),
		Period:          opts.Period,
		CheckResults:    make([]CheckResult, 0),
		Findings:        make([]Finding, 0),
		Recommendations: make([]Recommendation, 0),
	}

	// Get checks for the framework
	checks, err := g.manager.GetChecks(opts.Framework)
	if err != nil {
		return nil, fmt.Errorf("failed to get checks: %w", err)
	}

	// Evaluate each check
	var passedChecks, totalChecks int
	for _, check := range checks {
		// Apply filters
		if len(opts.CheckFilter) > 0 && !containsStr(opts.CheckFilter, check.ID) {
			continue
		}
		if len(opts.CategoryFilter) > 0 && !containsStr(opts.CategoryFilter, check.Category) {
			continue
		}

		result := g.evaluateCheck(ctx, check, opts)
		report.CheckResults = append(report.CheckResults, result)

		totalChecks++
		if result.Status == ComplianceStatusPassed {
			passedChecks++
		} else if result.Status == ComplianceStatusFailed {
			finding := Finding{
				ID:          fmt.Sprintf("finding-%s-%d", check.ID, time.Now().Unix()),
				CheckID:     check.ID,
				Title:       check.Title,
				Description: result.FailureReason,
				Severity:    result.Severity,
				Remediation: check.Remediation,
				DetectedAt:  time.Now(),
				Status:      "open",
			}
			report.Findings = append(report.Findings, finding)
		}
	}

	// Calculate overall score
	if totalChecks > 0 {
		report.OverallScore = float64(passedChecks) / float64(totalChecks) * 100
	}

	// Determine overall status
	report.OverallStatus = g.determineOverallStatus(report.OverallScore)

	// Generate summary
	report.Summary = g.generateSummary(report)

	// Generate recommendations
	report.Recommendations = g.generateRecommendations(report)

	// Sort findings by severity
	sort.Slice(report.Findings, func(i, j int) bool {
		return severityRank(report.Findings[i].Severity) < severityRank(report.Findings[j].Severity)
	})

	return report, nil
}

// evaluateCheck evaluates a single check
func (g *ReportGenerator) evaluateCheck(ctx context.Context, check *Check, opts ReportOpts) CheckResult {
	result := CheckResult{
		CheckID:     check.ID,
		CheckName:   check.Title,
		Category:    check.Category,
		EvaluatedAt: time.Now(),
	}

	// Use existing check status from the framework
	switch check.Status {
	case StatusPass:
		result.Status = ComplianceStatusPassed
		result.Score = 100
	case StatusFail:
		result.Status = ComplianceStatusFailed
		result.Score = 0
		result.FailureReason = check.Message
	case StatusWarning:
		result.Status = ComplianceStatusPartial
		result.Score = 50
		result.FailureReason = check.Message
	case StatusSkipped:
		result.Status = ComplianceStatusNotAssessed
		result.Score = 0
	default:
		result.Status = ComplianceStatusPassed
		result.Score = 100
	}

	result.Severity = g.determineSeverity(check, result)
	result.Remediation = check.Remediation

	// Collect evidence if requested
	if opts.IncludeEvidence && g.evidenceCollector != nil {
		evidences, err := g.evidenceCollector.CollectForCheck(ctx, check)
		if err == nil {
			result.Evidence = evidences
		}
	}

	return result
}

// determineOverallStatus determines overall compliance status from score
func (g *ReportGenerator) determineOverallStatus(score float64) ComplianceStatus {
	if score >= 95 {
		return ComplianceStatusPassed
	}
	if score >= 70 {
		return ComplianceStatusPartial
	}
	return ComplianceStatusFailed
}

// determineSeverity determines severity based on check and result
func (g *ReportGenerator) determineSeverity(check *Check, result CheckResult) Severity {
	if result.Status == ComplianceStatusPassed {
		return SeverityInfo
	}

	switch check.Severity {
	case "critical":
		return SeverityCritical
	case "high":
		return SeverityHigh
	case "medium":
		return SeverityMedium
	case "low":
		return SeverityLow
	default:
		return SeverityMedium
	}
}

// generateSummary generates a summary of the report
func (g *ReportGenerator) generateSummary(report *AuditReport) AuditSummary {
	summary := AuditSummary{
		TotalChecks: len(report.CheckResults),
	}

	for _, result := range report.CheckResults {
		switch result.Status {
		case ComplianceStatusPassed:
			summary.PassedChecks++
		case ComplianceStatusFailed:
			summary.FailedChecks++
		case ComplianceStatusPartial:
			summary.WarningChecks++
		case ComplianceStatusNotAssessed:
			summary.NotAssessed++
		}
	}

	for _, finding := range report.Findings {
		switch finding.Severity {
		case SeverityCritical:
			summary.CriticalFindings++
		case SeverityHigh:
			summary.HighFindings++
		case SeverityMedium:
			summary.MediumFindings++
		case SeverityLow:
			summary.LowFindings++
		}
	}

	if summary.TotalChecks > 0 {
		assessedChecks := summary.TotalChecks - summary.NotAssessed
		if assessedChecks > 0 {
			summary.CompliancePercent = float64(summary.PassedChecks) / float64(assessedChecks) * 100
		}
	}

	return summary
}

// generateRecommendations generates recommendations based on findings
func (g *ReportGenerator) generateRecommendations(report *AuditReport) []Recommendation {
	var recommendations []Recommendation

	priority := 1
	for _, finding := range report.Findings {
		rec := Recommendation{
			CheckID:     finding.CheckID,
			Priority:    priority,
			Title:       fmt.Sprintf("Remediate %s", finding.Title),
			Description: finding.Description,
			Impact:      "Improving compliance posture",
			Effort:      "medium",
			Steps:       []string{finding.Remediation},
		}
		recommendations = append(recommendations, rec)
		priority++
	}

	return recommendations
}

// ExportPDF exports the report as PDF
func (g *ReportGenerator) ExportPDF(ctx context.Context, report *AuditReport) ([]byte, error) {
	if g.templateEngine == nil {
		return nil, fmt.Errorf("template engine not configured")
	}
	return g.templateEngine.RenderPDF("compliance-report", report)
}

// ExportHTML exports the report as HTML
func (g *ReportGenerator) ExportHTML(ctx context.Context, report *AuditReport) (string, error) {
	if g.templateEngine == nil {
		return "", fmt.Errorf("template engine not configured")
	}
	return g.templateEngine.RenderHTML("compliance-report", report)
}

// Helper functions

func containsStr(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

func severityRank(s Severity) int {
	switch s {
	case SeverityCritical:
		return 0
	case SeverityHigh:
		return 1
	case SeverityMedium:
		return 2
	case SeverityLow:
		return 3
	case SeverityInfo:
		return 4
	default:
		return 5
	}
}
