package types

import (
	"fmt"
	"time"
)

// ServiceScorecard represents a scorecard for a service
type ServiceScorecard struct {
	APIVersion string                   `yaml:"apiVersion" json:"apiVersion"`
	Kind       string                   `yaml:"kind" json:"kind"`
	Metadata   Metadata                 `yaml:"metadata" json:"metadata"`
	Spec       ServiceScorecardSpec     `yaml:"spec" json:"spec"`
	Status     ServiceScorecardStatus   `yaml:"status,omitempty" json:"status,omitempty"`
}

// ServiceScorecardSpec defines the service scorecard specification
type ServiceScorecardSpec struct {
	ServiceRef  string  `yaml:"serviceRef" json:"serviceRef"` // Reference to service name
	Checks      []Check `yaml:"checks" json:"checks"`
}

// ServiceScorecardStatus represents the current status of the scorecard
type ServiceScorecardStatus struct {
	Score        int           `yaml:"score" json:"score"`                             // Overall score (0-100)
	Grade        ScorecardGrade `yaml:"grade" json:"grade"`                            // Letter grade
	PassedChecks int           `yaml:"passedChecks" json:"passedChecks"`
	FailedChecks int           `yaml:"failedChecks" json:"failedChecks"`
	TotalChecks  int           `yaml:"totalChecks" json:"totalChecks"`
	EvaluatedAt  time.Time     `yaml:"evaluatedAt" json:"evaluatedAt"`
}

// Check represents a single check in the scorecard
type Check struct {
	Name        string        `yaml:"name" json:"name"`
	Category    CheckCategory `yaml:"category" json:"category"`
	Weight      int           `yaml:"weight" json:"weight"`         // Importance weight (1-20)
	Status      CheckStatus   `yaml:"status" json:"status"`
	Score       int           `yaml:"score" json:"score"`           // Score for this check (0-100)
	Message     string        `yaml:"message,omitempty" json:"message,omitempty"`
	Details     string        `yaml:"details,omitempty" json:"details,omitempty"`
	EvaluatedAt time.Time     `yaml:"evaluatedAt" json:"evaluatedAt"`
}

// CheckResult is returned by check evaluation
type CheckResult struct {
	Status  CheckStatus
	Score   int    // 0-100
	Message string
	Details string
}

// ScorecardGrade represents the letter grade
type ScorecardGrade string

const (
	GradeA ScorecardGrade = "A"
	GradeB ScorecardGrade = "B"
	GradeC ScorecardGrade = "C"
	GradeD ScorecardGrade = "D"
	GradeF ScorecardGrade = "F"
)

// Legacy type for backwards compatibility
type ScorecardCheck = Check

// CheckCategory represents the category of a scorecard check
type CheckCategory string

const (
	CategoryDocumentation CheckCategory = "documentation"
	CategoryTesting       CheckCategory = "testing"
	CategoryQuality       CheckCategory = "quality"
	CategorySecurity      CheckCategory = "security"
	CategoryObservability CheckCategory = "observability"
	CategoryReliability   CheckCategory = "reliability"
	CategoryPerformance   CheckCategory = "performance"
	CategoryCompliance    CheckCategory = "compliance"
	CategoryGovernance    CheckCategory = "governance"
	CategoryDelivery      CheckCategory = "delivery"

	// Legacy names for backwards compatibility
	CheckCategoryDocumentation = CategoryDocumentation
	CheckCategoryTesting       = CategoryTesting
	CheckCategorySecurity      = CategorySecurity
	CheckCategoryObservability = CategoryObservability
	CheckCategoryReliability   = CategoryReliability
	CheckCategoryPerformance   = CategoryPerformance
	CheckCategoryCompliance    = CategoryCompliance
	CheckCategoryBestPractices = CategoryGovernance
)

// CheckStatus represents the status of a check
type CheckStatus string

const (
	CheckStatusPassed  CheckStatus = "passed"
	CheckStatusFailed  CheckStatus = "failed"
	CheckStatusWarning CheckStatus = "warning"
	CheckStatusInfo    CheckStatus = "info"
	CheckStatusSkipped CheckStatus = "skipped"

	// Legacy names for backwards compatibility
	CheckStatusPass    = CheckStatusPassed
	CheckStatusFail    = CheckStatusFailed
	CheckStatusUnknown = CheckStatusInfo
)

// Built-in check IDs
const (
	CheckIDReadme             = "readme"
	CheckIDTestCoverage       = "test-coverage"
	CheckIDSLODefined         = "slo-defined"
	CheckIDSecurityScan       = "security-scan"
	CheckIDDependenciesUpdate = "dependencies-updated"
	CheckIDObservability      = "observability"
	CheckIDDeploymentFreq     = "deployment-frequency"
	CheckIDMTTR               = "mttr"
	CheckIDDocumentation      = "documentation"
	CheckIDHealthCheck        = "health-check"
	CheckIDCICD               = "cicd"
	CheckIDCodeReview         = "code-review"
)

// Validate validates the service scorecard with security checks
func (sc *ServiceScorecard) Validate() error {
	if sc.APIVersion == "" {
		return ErrMissingAPIVersion
	}
	if sc.Kind != "ServiceScorecard" {
		return ErrInvalidKind
	}
	if sc.Metadata.Name == "" {
		return ErrMissingName
	}

	// Validate service reference
	if sc.Spec.ServiceRef == "" {
		return fmt.Errorf("serviceRef is required")
	}
	if len(sc.Spec.ServiceRef) > 253 {
		return fmt.Errorf("serviceRef must be 253 characters or less")
	}

	// Security: Limit number of checks
	if len(sc.Spec.Checks) > 100 {
		return fmt.Errorf("too many checks (max 100)")
	}

	// Validate score
	if sc.Status.Score < 0 || sc.Status.Score > 100 {
		return fmt.Errorf("score must be between 0 and 100")
	}

	// Validate grade
	if sc.Status.Grade != "" {
		if !IsValidGrade(string(sc.Status.Grade)) {
			return fmt.Errorf("invalid grade: %s (must be A, B, C, D, or F)", sc.Status.Grade)
		}
	}

	// Validate checks
	totalWeight := 0
	for i, check := range sc.Spec.Checks {
		if check.Name == "" {
			return fmt.Errorf("check %d: name is required", i)
		}
		if len(check.Name) > 200 {
			return fmt.Errorf("check %d: name must be 200 characters or less", i)
		}
		if check.Category == "" {
			return fmt.Errorf("check %d: category is required", i)
		}
		if !IsValidCheckCategory(check.Category) {
			return fmt.Errorf("check %d: invalid category %s", i, check.Category)
		}
		if check.Weight < 0 || check.Weight > 100 {
			return fmt.Errorf("check %d: weight must be between 0 and 100", i)
		}
		totalWeight += check.Weight
		if check.Status == "" {
			return fmt.Errorf("check %d: status is required", i)
		}
		if !IsValidCheckStatus(check.Status) {
			return fmt.Errorf("check %d: invalid status %s", i, check.Status)
		}
		if len(check.Message) > 1000 {
			return fmt.Errorf("check %d: message must be 1000 characters or less", i)
		}
		if len(check.Details) > 2000 {
			return fmt.Errorf("check %d: details must be 2000 characters or less", i)
		}
	}

	return nil
}

// IsValidCheckCategory checks if a check category is valid
func IsValidCheckCategory(category CheckCategory) bool {
	return category == CategoryDocumentation ||
		category == CategoryTesting ||
		category == CategoryQuality ||
		category == CategorySecurity ||
		category == CategoryObservability ||
		category == CategoryReliability ||
		category == CategoryPerformance ||
		category == CategoryCompliance ||
		category == CategoryGovernance ||
		category == CategoryDelivery
}

// IsValidCheckStatus checks if a check status is valid
func IsValidCheckStatus(status CheckStatus) bool {
	return status == CheckStatusPassed ||
		status == CheckStatusFailed ||
		status == CheckStatusWarning ||
		status == CheckStatusInfo ||
		status == CheckStatusSkipped
}

// IsValidGrade checks if a grade is valid
func IsValidGrade(grade string) bool {
	return grade == "A" || grade == "B" || grade == "C" || grade == "D" || grade == "F"
}

// CalculateScore calculates the weighted score from checks
func (sc *ServiceScorecard) CalculateScore() {
	if len(sc.Spec.Checks) == 0 {
		sc.Status.Score = 0
		sc.Status.Grade = GradeF
		return
	}

	totalScore := 0
	totalWeight := 0
	passedCount := 0
	failedCount := 0

	for _, check := range sc.Spec.Checks {
		totalScore += check.Score * check.Weight
		totalWeight += check.Weight

		if check.Status == CheckStatusPassed {
			passedCount++
		} else if check.Status == CheckStatusFailed {
			failedCount++
		}
	}

	if totalWeight == 0 {
		sc.Status.Score = 0
	} else {
		sc.Status.Score = totalScore / totalWeight
	}

	// Calculate grade
	if sc.Status.Score >= 90 {
		sc.Status.Grade = GradeA
	} else if sc.Status.Score >= 80 {
		sc.Status.Grade = GradeB
	} else if sc.Status.Score >= 70 {
		sc.Status.Grade = GradeC
	} else if sc.Status.Score >= 60 {
		sc.Status.Grade = GradeD
	} else {
		sc.Status.Grade = GradeF
	}

	sc.Status.PassedChecks = passedCount
	sc.Status.FailedChecks = failedCount
	sc.Status.TotalChecks = len(sc.Spec.Checks)
}
