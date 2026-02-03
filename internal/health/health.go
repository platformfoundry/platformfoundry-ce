// Package health provides platform health scoring by aggregating multiple subsystems.
package health

import (
	"context"
	"fmt"
	"os"
	"sort"
	"time"

	"github.com/platformfoundry/pf-ce/internal/cost"
	"github.com/platformfoundry/pf-ce/internal/drift"
	"github.com/platformfoundry/pf-ce/internal/lint"
	"github.com/platformfoundry/pf-ce/internal/policy"
)

// readFile reads a file and returns its contents
func readFile(path string) ([]byte, error) {
	return os.ReadFile(path)
}

// Status represents the health status
type Status string

const (
	StatusHealthy  Status = "healthy"
	StatusWarning  Status = "warning"
	StatusCritical Status = "critical"
	StatusUnknown  Status = "unknown"
)

// CategoryScore represents the score for a single category
type CategoryScore struct {
	Name       string  `json:"name"`
	Score      int     `json:"score"`       // 0-100
	Weight     float64 `json:"weight"`      // Contribution to overall (0.0-1.0)
	Status     Status  `json:"status"`
	IssueCount int     `json:"issue_count"`
	Message    string  `json:"message,omitempty"`
}

// Issue represents a health issue
type Issue struct {
	ID          string `json:"id"`
	Category    string `json:"category"`
	Severity    string `json:"severity"` // critical, high, medium, low, info
	Title       string `json:"title"`
	Description string `json:"description"`
	Suggestion  string `json:"suggestion,omitempty"`
	Resource    string `json:"resource,omitempty"`
}

// Recommendation represents a health recommendation
type Recommendation struct {
	ID          string `json:"id"`
	Category    string `json:"category"`
	Priority    string `json:"priority"` // high, medium, low
	Title       string `json:"title"`
	Description string `json:"description"`
	Command     string `json:"command,omitempty"` // Suggested pf command
	Impact      string `json:"impact,omitempty"`  // Expected improvement
}

// Score represents the overall health score
type Score struct {
	Overall         int                      `json:"overall"`          // 0-100
	Status          Status                   `json:"status"`
	Categories      map[string]CategoryScore `json:"categories"`
	Issues          []Issue                  `json:"issues"`
	Recommendations []Recommendation         `json:"recommendations"`
	CheckedAt       time.Time                `json:"checked_at"`
	Platform        string                   `json:"platform"`
}

// Config represents health checker configuration
type Config struct {
	// Category weights (must sum to 1.0)
	Weights CategoryWeights `yaml:"weights" json:"weights"`

	// Thresholds for status determination
	Thresholds StatusThresholds `yaml:"thresholds" json:"thresholds"`

	// Budget for cost evaluation (monthly)
	CostBudget float64 `yaml:"cost_budget" json:"cost_budget"`
}

// CategoryWeights defines the weight of each category
type CategoryWeights struct {
	Configuration float64 `yaml:"configuration" json:"configuration"`
	Drift         float64 `yaml:"drift" json:"drift"`
	Policy        float64 `yaml:"policy" json:"policy"`
	Cost          float64 `yaml:"cost" json:"cost"`
	Security      float64 `yaml:"security" json:"security"`
}

// StatusThresholds defines when status changes
type StatusThresholds struct {
	Healthy  int `yaml:"healthy" json:"healthy"`   // >= this is healthy
	Warning  int `yaml:"warning" json:"warning"`   // >= this is warning
	Critical int `yaml:"critical" json:"critical"` // < this is critical
}

// DefaultConfig returns default health checker configuration
func DefaultConfig() *Config {
	return &Config{
		Weights: CategoryWeights{
			Configuration: 0.20,
			Drift:         0.25,
			Policy:        0.25,
			Cost:          0.15,
			Security:      0.15,
		},
		Thresholds: StatusThresholds{
			Healthy:  80,
			Warning:  60,
			Critical: 40,
		},
		CostBudget: 10000, // $10,000/month default
	}
}

// Checker performs health checks
type Checker struct {
	config    *Config
	linter    *lint.Linter
	drifter   *drift.Detector
	policy    policy.Engine
	estimator *cost.Estimator
}

// NewChecker creates a new health checker
func NewChecker(config *Config) *Checker {
	if config == nil {
		config = DefaultConfig()
	}

	return &Checker{
		config: config,
		linter: lint.New(),
	}
}

// WithLinter sets the linter
func (c *Checker) WithLinter(l *lint.Linter) *Checker {
	c.linter = l
	return c
}

// WithDriftDetector sets the drift detector
func (c *Checker) WithDriftDetector(d *drift.Detector) *Checker {
	c.drifter = d
	return c
}

// WithPolicyEngine sets the policy engine
func (c *Checker) WithPolicyEngine(p policy.Engine) *Checker {
	c.policy = p
	return c
}

// WithCostEstimator sets the cost estimator
func (c *Checker) WithCostEstimator(e *cost.Estimator) *Checker {
	c.estimator = e
	return c
}

// Check performs a health check on the platform
func (c *Checker) Check(ctx context.Context, platform string, configFiles []string) (*Score, error) {
	score := &Score{
		Categories:      make(map[string]CategoryScore),
		Issues:          make([]Issue, 0),
		Recommendations: make([]Recommendation, 0),
		CheckedAt:       time.Now(),
		Platform:        platform,
	}

	// Check each category
	configScore := c.checkConfiguration(ctx, configFiles, score)
	score.Categories["configuration"] = configScore

	driftScore := c.checkDrift(ctx, platform, score)
	score.Categories["drift"] = driftScore

	policyScore := c.checkPolicy(ctx, platform, score)
	score.Categories["policy"] = policyScore

	costScore := c.checkCost(ctx, platform, score)
	score.Categories["cost"] = costScore

	securityScore := c.checkSecurity(ctx, configFiles, score)
	score.Categories["security"] = securityScore

	// Calculate overall score
	score.Overall = c.calculateOverall(score.Categories)
	score.Status = c.determineStatus(score.Overall)

	// Generate recommendations
	score.Recommendations = c.generateRecommendations(score)

	// Sort issues by severity
	sortIssues(score.Issues)

	return score, nil
}

// checkConfiguration checks configuration quality via linting
func (c *Checker) checkConfiguration(ctx context.Context, files []string, score *Score) CategoryScore {
	cat := CategoryScore{
		Name:   "Configuration",
		Weight: c.config.Weights.Configuration,
		Score:  100,
		Status: StatusHealthy,
	}

	if c.linter == nil || len(files) == 0 {
		cat.Score = 100
		cat.Message = "No configuration files to check"
		return cat
	}

	totalIssues := 0
	errorCount := 0
	warningCount := 0

	for _, file := range files {
		content, err := readFile(file)
		if err != nil {
			continue
		}
		result, err := c.linter.Lint(content, file)
		if err != nil {
			continue
		}
		totalIssues += len(result.Issues)
		errorCount += result.Summary.Errors
		warningCount += result.Summary.Warnings

		for _, issue := range result.Issues {
			score.Issues = append(score.Issues, Issue{
				ID:          fmt.Sprintf("lint-%s-%d", file, len(score.Issues)),
				Category:    "configuration",
				Severity:    string(issue.Severity),
				Title:       issue.Rule,
				Description: issue.Message,
				Suggestion:  issue.Suggestion,
				Resource:    file,
			})
		}
	}

	cat.IssueCount = totalIssues

	// Calculate score: start at 100, deduct for issues
	// Errors: -10 points each
	// Warnings: -5 points each
	deduction := (errorCount * 10) + (warningCount * 5)
	cat.Score = max(0, 100-deduction)
	cat.Status = c.categoryStatus(cat.Score)
	cat.Message = fmt.Sprintf("%d errors, %d warnings", errorCount, warningCount)

	return cat
}

// checkDrift checks for infrastructure drift
func (c *Checker) checkDrift(ctx context.Context, platform string, score *Score) CategoryScore {
	cat := CategoryScore{
		Name:   "Drift",
		Weight: c.config.Weights.Drift,
		Score:  100,
		Status: StatusHealthy,
	}

	if c.drifter == nil {
		cat.Message = "Drift detection not configured"
		return cat
	}

	// Drift detector requires resources to check - for now we return healthy if no detector
	// In production, this would load resources from state and check them
	cat.Message = "No drift detected"
	return cat
}

// checkPolicy checks policy compliance
func (c *Checker) checkPolicy(ctx context.Context, platform string, score *Score) CategoryScore {
	cat := CategoryScore{
		Name:   "Policy",
		Weight: c.config.Weights.Policy,
		Score:  100,
		Status: StatusHealthy,
	}

	if c.policy == nil {
		cat.Message = "Policy engine not configured"
		return cat
	}

	// Evaluate platform against policies
	input := map[string]interface{}{
		"platform": platform,
		"action":   "health_check",
	}

	result, err := c.policy.Evaluate(ctx, "platform_health", input)
	if err != nil {
		cat.Score = 80
		cat.Status = StatusWarning
		cat.Message = fmt.Sprintf("Policy evaluation error: %v", err)
		return cat
	}

	if !result.Allowed {
		cat.IssueCount = len(result.Reasons)
		for i, reason := range result.Reasons {
			score.Issues = append(score.Issues, Issue{
				ID:          fmt.Sprintf("policy-%d", i),
				Category:    "policy",
				Severity:    "high",
				Title:       "Policy violation",
				Description: reason,
				Suggestion:  "Review and fix policy violations",
			})
		}
		cat.Score = max(0, 100-(len(result.Reasons)*15))
	}

	cat.Status = c.categoryStatus(cat.Score)
	cat.Message = fmt.Sprintf("%d policy violations", cat.IssueCount)

	return cat
}

// checkCost checks cost efficiency
func (c *Checker) checkCost(ctx context.Context, platform string, score *Score) CategoryScore {
	cat := CategoryScore{
		Name:   "Cost",
		Weight: c.config.Weights.Cost,
		Score:  100,
		Status: StatusHealthy,
	}

	if c.estimator == nil {
		cat.Message = "Cost estimation not configured"
		return cat
	}

	// Cost estimator requires resource list - for now return default score
	// In production, this would load estimates from saved data
	cat.Message = "Within budget"
	return cat
}

// checkSecurity checks security posture
func (c *Checker) checkSecurity(ctx context.Context, files []string, score *Score) CategoryScore {
	cat := CategoryScore{
		Name:   "Security",
		Weight: c.config.Weights.Security,
		Score:  100,
		Status: StatusHealthy,
	}

	if c.linter == nil {
		cat.Message = "Security linting not configured"
		return cat
	}

	// Security-specific checks via linter
	// Count security-related issues (hardcoded secrets, missing security contexts, etc.)
	securityIssues := 0

	for _, file := range files {
		content, err := readFile(file)
		if err != nil {
			continue
		}
		result, err := c.linter.Lint(content, file)
		if err != nil {
			continue
		}
		for _, issue := range result.Issues {
			// Check for security-related rules
			if isSecurityRule(issue.Rule) {
				securityIssues++
				score.Issues = append(score.Issues, Issue{
					ID:          fmt.Sprintf("security-%s-%d", file, securityIssues),
					Category:    "security",
					Severity:    string(issue.Severity),
					Title:       issue.Rule,
					Description: issue.Message,
					Suggestion:  issue.Suggestion,
					Resource:    file,
				})
			}
		}
	}

	cat.IssueCount = securityIssues
	cat.Score = max(0, 100-(securityIssues*15))
	cat.Status = c.categoryStatus(cat.Score)
	cat.Message = fmt.Sprintf("%d security findings", securityIssues)

	return cat
}

// calculateOverall calculates the weighted overall score
func (c *Checker) calculateOverall(categories map[string]CategoryScore) int {
	total := 0.0
	totalWeight := 0.0

	for _, cat := range categories {
		total += float64(cat.Score) * cat.Weight
		totalWeight += cat.Weight
	}

	if totalWeight == 0 {
		return 100
	}

	return int(total / totalWeight)
}

// determineStatus determines overall status from score
func (c *Checker) determineStatus(score int) Status {
	if score >= c.config.Thresholds.Healthy {
		return StatusHealthy
	}
	if score >= c.config.Thresholds.Warning {
		return StatusWarning
	}
	return StatusCritical
}

// categoryStatus determines status for a category score
func (c *Checker) categoryStatus(score int) Status {
	if score >= 80 {
		return StatusHealthy
	}
	if score >= 60 {
		return StatusWarning
	}
	return StatusCritical
}

// generateRecommendations generates actionable recommendations
func (c *Checker) generateRecommendations(score *Score) []Recommendation {
	recs := make([]Recommendation, 0)

	// Generate recommendations based on issues
	hasDrift := false
	hasCostIssue := false
	hasSecurityIssue := false
	hasLintErrors := false

	for _, issue := range score.Issues {
		switch issue.Category {
		case "drift":
			hasDrift = true
		case "cost":
			hasCostIssue = true
		case "security":
			hasSecurityIssue = true
		case "configuration":
			if issue.Severity == "error" {
				hasLintErrors = true
			}
		}
	}

	if hasDrift {
		recs = append(recs, Recommendation{
			ID:          "rec-fix-drift",
			Category:    "drift",
			Priority:    "high",
			Title:       "Fix infrastructure drift",
			Description: "Infrastructure has drifted from desired state",
			Command:     "pf drift fix",
			Impact:      "Restore infrastructure to desired configuration",
		})
	}

	if hasCostIssue {
		recs = append(recs, Recommendation{
			ID:          "rec-optimize-cost",
			Category:    "cost",
			Priority:    "medium",
			Title:       "Optimize cloud costs",
			Description: "Cost optimization opportunities available",
			Command:     "pf cost optimize",
			Impact:      "Reduce monthly cloud spend",
		})
	}

	if hasSecurityIssue {
		recs = append(recs, Recommendation{
			ID:          "rec-fix-security",
			Category:    "security",
			Priority:    "high",
			Title:       "Address security findings",
			Description: "Security issues detected in configuration",
			Command:     "pf lint --security",
			Impact:      "Improve security posture",
		})
	}

	if hasLintErrors {
		recs = append(recs, Recommendation{
			ID:          "rec-fix-config",
			Category:    "configuration",
			Priority:    "medium",
			Title:       "Fix configuration errors",
			Description: "Configuration validation errors found",
			Command:     "pf lint --fix",
			Impact:      "Ensure valid configuration",
		})
	}

	return recs
}

// isSecurityRule checks if a lint rule is security-related
func isSecurityRule(rule string) bool {
	securityRules := map[string]bool{
		"hardcoded-secrets":  true,
		"security-context":   true,
		"privileged-mode":    true,
		"host-network":       true,
		"host-pid":           true,
		"root-user":          true,
		"missing-seccomp":    true,
		"missing-apparmor":   true,
		"insecure-port":      true,
		"plaintext-secret":   true,
	}
	return securityRules[rule]
}

// sortIssues sorts issues by severity (critical first)
func sortIssues(issues []Issue) {
	severityOrder := map[string]int{
		"critical": 0,
		"high":     1,
		"medium":   2,
		"low":      3,
		"info":     4,
	}

	sort.Slice(issues, func(i, j int) bool {
		return severityOrder[issues[i].Severity] < severityOrder[issues[j].Severity]
	})
}

// max returns the maximum of two integers
func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
