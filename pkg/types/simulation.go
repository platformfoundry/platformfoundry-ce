package types

import (
	"time"
)

// SimulationMode defines how simulation behaves
type SimulationMode string

const (
	// SimulationModeDryRun shows what would happen without making changes
	SimulationModeDryRun SimulationMode = "dry-run"

	// SimulationModeWhatIf shows impact of hypothetical changes
	SimulationModeWhatIf SimulationMode = "what-if"

	// SimulationModeValidate only validates without planning
	SimulationModeValidate SimulationMode = "validate"
)

// SimulationRequest represents a request to simulate changes
type SimulationRequest struct {
	// Mode defines the simulation type
	Mode SimulationMode `json:"mode" yaml:"mode"`

	// Resources to simulate
	Resources []SimulatedResource `json:"resources" yaml:"resources"`

	// Options for the simulation
	Options SimulationOptions `json:"options" yaml:"options"`
}

// SimulatedResource represents a resource to simulate
type SimulatedResource struct {
	// Action to simulate (create, update, delete)
	Action string `json:"action" yaml:"action"`

	// Kind of resource
	Kind string `json:"kind" yaml:"kind"`

	// Name of resource
	Name string `json:"name" yaml:"name"`

	// Spec is the resource specification
	Spec map[string]interface{} `json:"spec,omitempty" yaml:"spec,omitempty"`

	// Current is the current state (for updates)
	Current map[string]interface{} `json:"current,omitempty" yaml:"current,omitempty"`
}

// SimulationOptions configures the simulation
type SimulationOptions struct {
	// IncludeCostEstimate includes cost impact
	IncludeCostEstimate bool `json:"include_cost_estimate" yaml:"include_cost_estimate"`

	// IncludePolicyCheck checks against policies
	IncludePolicyCheck bool `json:"include_policy_check" yaml:"include_policy_check"`

	// IncludeImpactAnalysis includes dependency impact
	IncludeImpactAnalysis bool `json:"include_impact_analysis" yaml:"include_impact_analysis"`

	// IncludeComplianceCheck checks compliance
	IncludeComplianceCheck bool `json:"include_compliance_check" yaml:"include_compliance_check"`

	// TargetEnvironment to simulate against
	TargetEnvironment string `json:"target_environment,omitempty" yaml:"target_environment,omitempty"`

	// Timeout for the simulation
	Timeout time.Duration `json:"timeout,omitempty" yaml:"timeout,omitempty"`
}

// SimulationReport represents the result of a simulation
type SimulationReport struct {
	// ID of the simulation
	ID string `json:"id" yaml:"id"`

	// Mode used for simulation
	Mode SimulationMode `json:"mode" yaml:"mode"`

	// Status of the simulation
	Status string `json:"status" yaml:"status"`

	// Timestamp of the simulation
	Timestamp time.Time `json:"timestamp" yaml:"timestamp"`

	// Duration of the simulation
	Duration time.Duration `json:"duration" yaml:"duration"`

	// Summary of changes
	Summary SimulationSummary `json:"summary" yaml:"summary"`

	// ResourceChanges detailed per-resource changes
	ResourceChanges []ResourceChange `json:"resource_changes" yaml:"resource_changes"`

	// ValidationErrors found during simulation
	ValidationErrors []ValidationError `json:"validation_errors,omitempty" yaml:"validation_errors,omitempty"`

	// PolicyViolations found during simulation
	PolicyViolations []PolicyViolation `json:"policy_violations,omitempty" yaml:"policy_violations,omitempty"`

	// ComplianceIssues found during simulation
	ComplianceIssues []ComplianceIssue `json:"compliance_issues,omitempty" yaml:"compliance_issues,omitempty"`

	// CostEstimate for the changes
	CostEstimate *CostEstimate `json:"cost_estimate,omitempty" yaml:"cost_estimate,omitempty"`

	// ImpactAnalysis for the changes
	ImpactAnalysis *SimulationImpact `json:"impact_analysis,omitempty" yaml:"impact_analysis,omitempty"`

	// Warnings generated during simulation
	Warnings []string `json:"warnings,omitempty" yaml:"warnings,omitempty"`

	// Recommendations for the changes
	Recommendations []string `json:"recommendations,omitempty" yaml:"recommendations,omitempty"`

	// CanProceed indicates if it's safe to apply
	CanProceed bool `json:"can_proceed" yaml:"can_proceed"`

	// BlockedReason if cannot proceed
	BlockedReason string `json:"blocked_reason,omitempty" yaml:"blocked_reason,omitempty"`
}

// SimulationSummary summarizes the simulation results
type SimulationSummary struct {
	// TotalResources involved
	TotalResources int `json:"total_resources" yaml:"total_resources"`

	// ToCreate count
	ToCreate int `json:"to_create" yaml:"to_create"`

	// ToUpdate count
	ToUpdate int `json:"to_update" yaml:"to_update"`

	// ToDelete count
	ToDelete int `json:"to_delete" yaml:"to_delete"`

	// Unchanged count
	Unchanged int `json:"unchanged" yaml:"unchanged"`

	// ValidationErrorCount
	ValidationErrorCount int `json:"validation_error_count" yaml:"validation_error_count"`

	// PolicyViolationCount
	PolicyViolationCount int `json:"policy_violation_count" yaml:"policy_violation_count"`

	// ComplianceIssueCount
	ComplianceIssueCount int `json:"compliance_issue_count" yaml:"compliance_issue_count"`

	// EstimatedCostChange monthly
	EstimatedCostChange float64 `json:"estimated_cost_change,omitempty" yaml:"estimated_cost_change,omitempty"`

	// BlastRadius total affected resources
	BlastRadius int `json:"blast_radius" yaml:"blast_radius"`
}

// ResourceChange represents a change to a resource
type ResourceChange struct {
	// Action (create, update, delete, unchanged)
	Action string `json:"action" yaml:"action"`

	// Kind of resource
	Kind string `json:"kind" yaml:"kind"`

	// Name of resource
	Name string `json:"name" yaml:"name"`

	// Provider handling this resource
	Provider string `json:"provider,omitempty" yaml:"provider,omitempty"`

	// Diff showing what changed
	Diff *ResourceDiff `json:"diff,omitempty" yaml:"diff,omitempty"`

	// Plan from the provider
	Plan *ProviderPlan `json:"plan,omitempty" yaml:"plan,omitempty"`

	// ValidationErrors for this resource
	ValidationErrors []ValidationError `json:"validation_errors,omitempty" yaml:"validation_errors,omitempty"`

	// PolicyViolations for this resource
	PolicyViolations []PolicyViolation `json:"policy_violations,omitempty" yaml:"policy_violations,omitempty"`

	// EstimatedCost for this resource
	EstimatedCost *ResourceCost `json:"estimated_cost,omitempty" yaml:"estimated_cost,omitempty"`

	// Dependencies of this resource
	Dependencies []string `json:"dependencies,omitempty" yaml:"dependencies,omitempty"`

	// Dependents affected by this change
	Dependents []string `json:"dependents,omitempty" yaml:"dependents,omitempty"`
}

// ResourceDiff shows the difference between current and desired state
type ResourceDiff struct {
	// Before state
	Before map[string]interface{} `json:"before,omitempty" yaml:"before,omitempty"`

	// After state
	After map[string]interface{} `json:"after,omitempty" yaml:"after,omitempty"`

	// Changes list of individual changes
	Changes []FieldChange `json:"changes" yaml:"changes"`
}

// FieldChange represents a change to a field
type FieldChange struct {
	// Path to the field (e.g., "spec.replicas")
	Path string `json:"path" yaml:"path"`

	// OldValue before change
	OldValue interface{} `json:"old_value,omitempty" yaml:"old_value,omitempty"`

	// NewValue after change
	NewValue interface{} `json:"new_value,omitempty" yaml:"new_value,omitempty"`

	// Type of change (add, modify, delete)
	Type string `json:"type" yaml:"type"`
}

// ProviderPlan represents the plan from a provider
type ProviderPlan struct {
	// Provider name
	Provider string `json:"provider" yaml:"provider"`

	// Actions to be taken
	Actions []string `json:"actions" yaml:"actions"`

	// ResourcesAffected count
	ResourcesAffected int `json:"resources_affected" yaml:"resources_affected"`

	// Output from the provider's plan
	Output string `json:"output,omitempty" yaml:"output,omitempty"`
}

// ValidationError represents a validation failure
type ValidationError struct {
	// Resource that failed validation
	Resource string `json:"resource" yaml:"resource"`

	// Field that failed
	Field string `json:"field" yaml:"field"`

	// Message describing the error
	Message string `json:"message" yaml:"message"`

	// Severity (error, warning)
	Severity string `json:"severity" yaml:"severity"`

	// Code for the error
	Code string `json:"code,omitempty" yaml:"code,omitempty"`
}

// PolicyViolation represents a policy violation
type PolicyViolation struct {
	// Resource that violated policy
	Resource string `json:"resource" yaml:"resource"`

	// Policy that was violated
	Policy string `json:"policy" yaml:"policy"`

	// Rule that was violated
	Rule string `json:"rule" yaml:"rule"`

	// Message describing the violation
	Message string `json:"message" yaml:"message"`

	// Severity (error, warning)
	Severity string `json:"severity" yaml:"severity"`

	// Enforcement (enforce, warn)
	Enforcement string `json:"enforcement" yaml:"enforcement"`
}

// ComplianceIssue represents a compliance issue
type ComplianceIssue struct {
	// Resource with the issue
	Resource string `json:"resource" yaml:"resource"`

	// Framework (SOC2, HIPAA, etc.)
	Framework string `json:"framework" yaml:"framework"`

	// Control that would be violated
	Control string `json:"control" yaml:"control"`

	// Message describing the issue
	Message string `json:"message" yaml:"message"`

	// Severity
	Severity string `json:"severity" yaml:"severity"`
}

// CostEstimate represents estimated cost impact
type CostEstimate struct {
	// Currency for all amounts
	Currency string `json:"currency" yaml:"currency"`

	// CurrentMonthlyCost before changes
	CurrentMonthlyCost float64 `json:"current_monthly_cost" yaml:"current_monthly_cost"`

	// EstimatedMonthlyCost after changes
	EstimatedMonthlyCost float64 `json:"estimated_monthly_cost" yaml:"estimated_monthly_cost"`

	// MonthlyCostChange difference
	MonthlyCostChange float64 `json:"monthly_cost_change" yaml:"monthly_cost_change"`

	// PercentageChange
	PercentageChange float64 `json:"percentage_change" yaml:"percentage_change"`

	// ByResource cost breakdown
	ByResource map[string]*ResourceCost `json:"by_resource,omitempty" yaml:"by_resource,omitempty"`

	// ByCategory cost breakdown
	ByCategory map[string]float64 `json:"by_category,omitempty" yaml:"by_category,omitempty"`

	// Confidence level (0-1)
	Confidence float64 `json:"confidence" yaml:"confidence"`

	// Notes about the estimate
	Notes []string `json:"notes,omitempty" yaml:"notes,omitempty"`
}

// ResourceCost represents cost for a single resource
type ResourceCost struct {
	// Resource name
	Resource string `json:"resource" yaml:"resource"`

	// CurrentCost monthly
	CurrentCost float64 `json:"current_cost" yaml:"current_cost"`

	// EstimatedCost monthly
	EstimatedCost float64 `json:"estimated_cost" yaml:"estimated_cost"`

	// CostChange
	CostChange float64 `json:"cost_change" yaml:"cost_change"`

	// Category (compute, storage, network, etc.)
	Category string `json:"category" yaml:"category"`
}

// SimulationImpact represents the impact analysis
type SimulationImpact struct {
	// DirectlyAffected resources
	DirectlyAffected []string `json:"directly_affected" yaml:"directly_affected"`

	// TransitivelyAffected resources
	TransitivelyAffected []string `json:"transitively_affected" yaml:"transitively_affected"`

	// BlastRadius total
	BlastRadius int `json:"blast_radius" yaml:"blast_radius"`

	// CriticalResourcesAffected count
	CriticalResourcesAffected int `json:"critical_resources_affected" yaml:"critical_resources_affected"`

	// AffectedTeams list
	AffectedTeams []string `json:"affected_teams" yaml:"affected_teams"`

	// AffectedEnvironments list
	AffectedEnvironments []string `json:"affected_environments" yaml:"affected_environments"`

	// RiskLevel (low, medium, high, critical)
	RiskLevel string `json:"risk_level" yaml:"risk_level"`

	// RequiresApproval based on impact
	RequiresApproval bool `json:"requires_approval" yaml:"requires_approval"`

	// ApprovalReason if required
	ApprovalReason string `json:"approval_reason,omitempty" yaml:"approval_reason,omitempty"`
}
