package types

import (
	"time"
)

// ComplianceFramework represents a compliance framework (e.g., SOC2, HIPAA, PCI-DSS)
type ComplianceFramework string

const (
	ComplianceSOC2     ComplianceFramework = "soc2"
	ComplianceHIPAA    ComplianceFramework = "hipaa"
	CompliancePCIDSS   ComplianceFramework = "pci-dss"
	ComplianceGDPR     ComplianceFramework = "gdpr"
	ComplianceISO27001 ComplianceFramework = "iso27001"
	ComplianceNIST     ComplianceFramework = "nist"
	ComplianceCIS      ComplianceFramework = "cis"
	ComplianceCustom   ComplianceFramework = "custom"
)

// CompliancePolicy defines a compliance policy
type CompliancePolicy struct {
	APIVersion string                  `yaml:"apiVersion" json:"apiVersion"`
	Kind       string                  `yaml:"kind" json:"kind"`
	Metadata   ComplianceMetadata      `yaml:"metadata" json:"metadata"`
	Spec       CompliancePolicySpec    `yaml:"spec" json:"spec"`
	Status     *CompliancePolicyStatus `yaml:"status,omitempty" json:"status,omitempty"`
}

// ComplianceMetadata contains policy metadata
type ComplianceMetadata struct {
	Name        string            `yaml:"name" json:"name"`
	Labels      map[string]string `yaml:"labels,omitempty" json:"labels,omitempty"`
	Annotations map[string]string `yaml:"annotations,omitempty" json:"annotations,omitempty"`
}

// CompliancePolicySpec defines the policy specification
type CompliancePolicySpec struct {
	Framework     ComplianceFramework   `yaml:"framework" json:"framework"`
	Description   string                `yaml:"description,omitempty" json:"description,omitempty"`
	Version       string                `yaml:"version,omitempty" json:"version,omitempty"`
	Rules         []ComplianceRule      `yaml:"rules" json:"rules"`
	Exceptions    []ComplianceException `yaml:"exceptions,omitempty" json:"exceptions,omitempty"`
	Schedule      *ComplianceSchedule   `yaml:"schedule,omitempty" json:"schedule,omitempty"`
	Notifications ComplianceNotify      `yaml:"notifications,omitempty" json:"notifications,omitempty"`
	Enforcement   EnforcementMode       `yaml:"enforcement" json:"enforcement"`
}

// ComplianceRule defines a single compliance rule
type ComplianceRule struct {
	ID          string           `yaml:"id" json:"id"`
	Name        string           `yaml:"name" json:"name"`
	Description string           `yaml:"description,omitempty" json:"description,omitempty"`
	Severity    RuleSeverity     `yaml:"severity" json:"severity"`
	Category    string           `yaml:"category,omitempty" json:"category,omitempty"`
	Target      RuleTarget       `yaml:"target" json:"target"`
	Check       RuleCheck        `yaml:"check" json:"check"`
	Remediation *RuleRemediation `yaml:"remediation,omitempty" json:"remediation,omitempty"`
	References  []string         `yaml:"references,omitempty" json:"references,omitempty"`
	Enabled     bool             `yaml:"enabled" json:"enabled"`
}

// RuleSeverity represents the severity of a compliance rule
type RuleSeverity string

const (
	SeverityCritical RuleSeverity = "critical"
	SeverityHigh     RuleSeverity = "high"
	SeverityMedium   RuleSeverity = "medium"
	SeverityLow      RuleSeverity = "low"
	SeverityInfo     RuleSeverity = "info"
)

// RuleTarget defines what the rule applies to
type RuleTarget struct {
	Kind       string            `yaml:"kind" json:"kind"` // Service, Deployment, ConfigMap, etc.
	APIVersion string            `yaml:"apiVersion,omitempty" json:"apiVersion,omitempty"`
	Selector   map[string]string `yaml:"selector,omitempty" json:"selector,omitempty"`
	Namespaces []string          `yaml:"namespaces,omitempty" json:"namespaces,omitempty"`
}

// RuleCheck defines how to check compliance
type RuleCheck struct {
	Type       CheckType              `yaml:"type" json:"type"`
	Expression string                 `yaml:"expression,omitempty" json:"expression,omitempty"`
	Path       string                 `yaml:"path,omitempty" json:"path,omitempty"`
	Operator   string                 `yaml:"operator,omitempty" json:"operator,omitempty"`
	Value      interface{}            `yaml:"value,omitempty" json:"value,omitempty"`
	Script     string                 `yaml:"script,omitempty" json:"script,omitempty"`
	Parameters map[string]interface{} `yaml:"parameters,omitempty" json:"parameters,omitempty"`
}

// CheckType represents the type of compliance check
type CheckType string

const (
	CheckTypeExpr     CheckType = "expression" // CEL or Rego expression
	CheckTypeJSONPath CheckType = "jsonpath"   // JSON path check
	CheckTypeScript   CheckType = "script"     // Custom script
	CheckTypeBuiltin  CheckType = "builtin"    // Built-in check
	CheckTypeWebhook  CheckType = "webhook"    // External webhook
)

// RuleRemediation defines how to fix violations
type RuleRemediation struct {
	Description string `yaml:"description" json:"description"`
	AutoFix     bool   `yaml:"autoFix,omitempty" json:"autoFix,omitempty"`
	Script      string `yaml:"script,omitempty" json:"script,omitempty"`
	Link        string `yaml:"link,omitempty" json:"link,omitempty"`
}

// ComplianceException defines an exception to a rule
type ComplianceException struct {
	RuleID     string     `yaml:"ruleId" json:"ruleId"`
	Reason     string     `yaml:"reason" json:"reason"`
	ApprovedBy string     `yaml:"approvedBy" json:"approvedBy"`
	ExpiresAt  *time.Time `yaml:"expiresAt,omitempty" json:"expiresAt,omitempty"`
	Resources  []string   `yaml:"resources,omitempty" json:"resources,omitempty"`
}

// ComplianceSchedule defines when to run compliance scans
type ComplianceSchedule struct {
	Cron     string `yaml:"cron,omitempty" json:"cron,omitempty"`
	Interval string `yaml:"interval,omitempty" json:"interval,omitempty"`
	Timezone string `yaml:"timezone,omitempty" json:"timezone,omitempty"`
}

// ComplianceNotify defines notification settings
type ComplianceNotify struct {
	OnViolation   bool     `yaml:"onViolation" json:"onViolation"`
	OnRemediation bool     `yaml:"onRemediation" json:"onRemediation"`
	Channels      []string `yaml:"channels,omitempty" json:"channels,omitempty"`
}

// EnforcementMode defines how violations are handled
type EnforcementMode string

const (
	EnforcementAudit   EnforcementMode = "audit"   // Log only
	EnforcementWarn    EnforcementMode = "warn"    // Warn but allow
	EnforcementDeny    EnforcementMode = "deny"    // Block violations
	EnforcementAutoFix EnforcementMode = "autofix" // Automatically remediate
)

// CompliancePolicyStatus represents policy status
type CompliancePolicyStatus struct {
	LastScan     *time.Time            `yaml:"lastScan,omitempty" json:"lastScan,omitempty"`
	NextScan     *time.Time            `yaml:"nextScan,omitempty" json:"nextScan,omitempty"`
	TotalRules   int                   `yaml:"totalRules" json:"totalRules"`
	PassingRules int                   `yaml:"passingRules" json:"passingRules"`
	FailingRules int                   `yaml:"failingRules" json:"failingRules"`
	Compliance   float64               `yaml:"compliance" json:"compliance"` // Percentage
	Conditions   []ComplianceCondition `yaml:"conditions,omitempty" json:"conditions,omitempty"`
}

// ComplianceCondition represents a status condition
type ComplianceCondition struct {
	Type               string    `yaml:"type" json:"type"`
	Status             string    `yaml:"status" json:"status"`
	LastTransitionTime time.Time `yaml:"lastTransitionTime" json:"lastTransitionTime"`
	Reason             string    `yaml:"reason,omitempty" json:"reason,omitempty"`
	Message            string    `yaml:"message,omitempty" json:"message,omitempty"`
}

// ComplianceScanResult represents a scan result
type ComplianceScanResult struct {
	ScanID      string                `json:"scanId"`
	PolicyName  string                `json:"policyName"`
	Framework   ComplianceFramework   `json:"framework"`
	StartTime   time.Time             `json:"startTime"`
	EndTime     time.Time             `json:"endTime"`
	Duration    time.Duration         `json:"duration"`
	TotalChecks int                   `json:"totalChecks"`
	Passed      int                   `json:"passed"`
	Failed      int                   `json:"failed"`
	Skipped     int                   `json:"skipped"`
	Compliance  float64               `json:"compliance"`
	Violations  []ComplianceViolation `json:"violations"`
	Summary     map[string]int        `json:"summary"` // By severity
}

// ComplianceViolation represents a specific violation
type ComplianceViolation struct {
	RuleID      string                 `json:"ruleId"`
	RuleName    string                 `json:"ruleName"`
	Severity    RuleSeverity           `json:"severity"`
	Category    string                 `json:"category,omitempty"`
	Resource    ViolationResource      `json:"resource"`
	Description string                 `json:"description"`
	Evidence    map[string]interface{} `json:"evidence,omitempty"`
	Remediation string                 `json:"remediation,omitempty"`
	AutoFixable bool                   `json:"autoFixable"`
	DetectedAt  time.Time              `json:"detectedAt"`
	Status      ViolationStatus        `json:"status"`
}

// ViolationResource identifies the violating resource
type ViolationResource struct {
	Kind       string `json:"kind"`
	Name       string `json:"name"`
	Namespace  string `json:"namespace,omitempty"`
	APIVersion string `json:"apiVersion,omitempty"`
}

// ViolationStatus represents the status of a violation
type ViolationStatus string

const (
	ViolationStatusOpen         ViolationStatus = "open"
	ViolationStatusAcknowledged ViolationStatus = "acknowledged"
	ViolationStatusRemediated   ViolationStatus = "remediated"
	ViolationStatusExcepted     ViolationStatus = "excepted"
)

// ComplianceReport summarizes compliance status
type ComplianceReport struct {
	GeneratedAt          time.Time             `json:"generatedAt"`
	Period               string                `json:"period"`
	Frameworks           []FrameworkSummary    `json:"frameworks"`
	TotalPolicies        int                   `json:"totalPolicies"`
	OverallCompliance    float64               `json:"overallCompliance"`
	ViolationsBySeverity map[RuleSeverity]int  `json:"violationsBySeverity"`
	TopViolations        []ComplianceViolation `json:"topViolations"`
	Trends               []ComplianceTrend     `json:"trends,omitempty"`
}

// FrameworkSummary summarizes compliance by framework
type FrameworkSummary struct {
	Framework  ComplianceFramework `json:"framework"`
	Compliance float64             `json:"compliance"`
	TotalRules int                 `json:"totalRules"`
	Passing    int                 `json:"passing"`
	Failing    int                 `json:"failing"`
	Trend      string              `json:"trend"` // improving, stable, declining
}

// ComplianceTrend represents compliance over time
type ComplianceTrend struct {
	Date       time.Time `json:"date"`
	Compliance float64   `json:"compliance"`
	Violations int       `json:"violations"`
}
