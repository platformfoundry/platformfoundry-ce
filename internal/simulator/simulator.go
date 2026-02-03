package simulator

import (
	"context"
	"fmt"
	"reflect"
	"strings"
	"time"

	"github.com/platformfoundry/pf-ce/pkg/types"
)

// Simulator provides dry-run and what-if analysis capabilities
type Simulator struct {
	policyChecker     PolicyChecker
	costEstimator     CostEstimator
	complianceChecker ComplianceChecker
	validator         Validator
}

// PolicyChecker interface for checking policies
type PolicyChecker interface {
	Check(ctx context.Context, resource *types.SimulatedResource) ([]types.PolicyViolation, error)
}

// CostEstimator interface for estimating costs
type CostEstimator interface {
	Estimate(ctx context.Context, changes []types.ResourceChange) (*types.CostEstimate, error)
}

// ComplianceChecker interface for checking compliance
type ComplianceChecker interface {
	Check(ctx context.Context, resource *types.SimulatedResource) ([]types.ComplianceIssue, error)
}

// Validator interface for validating resources
type Validator interface {
	Validate(ctx context.Context, resource *types.SimulatedResource) ([]types.ValidationError, error)
}

// Config configures the simulator
type Config struct {
	PolicyChecker     PolicyChecker
	CostEstimator     CostEstimator
	ComplianceChecker ComplianceChecker
	Validator         Validator
	DefaultTimeout    time.Duration
}

// NewSimulator creates a new simulator
func NewSimulator(config Config) *Simulator {
	return &Simulator{
		policyChecker:     config.PolicyChecker,
		costEstimator:     config.CostEstimator,
		complianceChecker: config.ComplianceChecker,
		validator:         config.Validator,
	}
}

// Simulate executes a simulation
func (s *Simulator) Simulate(ctx context.Context, req *types.SimulationRequest) (*types.SimulationReport, error) {
	if req == nil {
		return nil, fmt.Errorf("simulation request cannot be nil")
	}

	startTime := time.Now()

	// Apply timeout if specified
	if req.Options.Timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, req.Options.Timeout)
		defer cancel()
	}

	report := &types.SimulationReport{
		ID:              generateID("sim"),
		Mode:            req.Mode,
		Status:          "completed",
		Timestamp:       startTime,
		ResourceChanges: make([]types.ResourceChange, 0),
		CanProceed:      true,
	}

	// Process each resource
	for _, resource := range req.Resources {
		change, err := s.processResource(ctx, &resource, req.Options)
		if err != nil {
			report.Warnings = append(report.Warnings, fmt.Sprintf("Error processing %s/%s: %v", resource.Kind, resource.Name, err))
			continue
		}
		report.ResourceChanges = append(report.ResourceChanges, *change)

		// Collect validation errors
		report.ValidationErrors = append(report.ValidationErrors, change.ValidationErrors...)

		// Collect policy violations
		report.PolicyViolations = append(report.PolicyViolations, change.PolicyViolations...)
	}

	// Run compliance checks if requested
	if req.Options.IncludeComplianceCheck && s.complianceChecker != nil {
		for _, resource := range req.Resources {
			issues, err := s.complianceChecker.Check(ctx, &resource)
			if err != nil {
				report.Warnings = append(report.Warnings, fmt.Sprintf("Compliance check error: %v", err))
			} else {
				report.ComplianceIssues = append(report.ComplianceIssues, issues...)
			}
		}
	}

	// Estimate costs if requested
	if req.Options.IncludeCostEstimate && s.costEstimator != nil {
		estimate, err := s.costEstimator.Estimate(ctx, report.ResourceChanges)
		if err != nil {
			report.Warnings = append(report.Warnings, fmt.Sprintf("Cost estimation error: %v", err))
		} else {
			report.CostEstimate = estimate
		}
	}

	// Impact analysis requires external graph engine (removed)
	// Use kubectl or ArgoCD for dependency visualization
	if req.Options.IncludeImpactAnalysis {
		report.Warnings = append(report.Warnings, "Impact analysis feature removed - use kubectl for dependency analysis")
	}

	// Calculate summary
	report.Summary = s.calculateSummary(report)

	// Determine if we can proceed
	s.determineCanProceed(report)

	// Generate recommendations
	report.Recommendations = s.generateRecommendations(report)

	report.Duration = time.Since(startTime)

	return report, nil
}

// DryRun is a convenience method for dry-run simulation
func (s *Simulator) DryRun(ctx context.Context, resources []types.SimulatedResource) (*types.SimulationReport, error) {
	return s.Simulate(ctx, &types.SimulationRequest{
		Mode:      types.SimulationModeDryRun,
		Resources: resources,
		Options: types.SimulationOptions{
			IncludeCostEstimate:   true,
			IncludePolicyCheck:    true,
			IncludeImpactAnalysis: true,
		},
	})
}

// WhatIf is a convenience method for what-if analysis
func (s *Simulator) WhatIf(ctx context.Context, resources []types.SimulatedResource) (*types.SimulationReport, error) {
	return s.Simulate(ctx, &types.SimulationRequest{
		Mode:      types.SimulationModeWhatIf,
		Resources: resources,
		Options: types.SimulationOptions{
			IncludeCostEstimate:    true,
			IncludePolicyCheck:     true,
			IncludeImpactAnalysis:  true,
			IncludeComplianceCheck: true,
		},
	})
}

// Validate is a convenience method for validation only
func (s *Simulator) Validate(ctx context.Context, resources []types.SimulatedResource) (*types.SimulationReport, error) {
	return s.Simulate(ctx, &types.SimulationRequest{
		Mode:      types.SimulationModeValidate,
		Resources: resources,
		Options: types.SimulationOptions{
			IncludePolicyCheck: true,
		},
	})
}

// processResource processes a single resource for simulation
func (s *Simulator) processResource(ctx context.Context, resource *types.SimulatedResource, options types.SimulationOptions) (*types.ResourceChange, error) {
	change := &types.ResourceChange{
		Action:           resource.Action,
		Kind:             resource.Kind,
		Name:             resource.Name,
		ValidationErrors: make([]types.ValidationError, 0),
		PolicyViolations: make([]types.PolicyViolation, 0),
	}

	// Validate the resource
	if s.validator != nil {
		errors, err := s.validator.Validate(ctx, resource)
		if err != nil {
			return nil, fmt.Errorf("validation failed: %w", err)
		}
		change.ValidationErrors = errors
	}

	// Check policies
	if options.IncludePolicyCheck && s.policyChecker != nil {
		violations, err := s.policyChecker.Check(ctx, resource)
		if err != nil {
			return nil, fmt.Errorf("policy check failed: %w", err)
		}
		change.PolicyViolations = violations
	}

	// Calculate diff for updates
	if resource.Action == "update" && resource.Current != nil && resource.Spec != nil {
		change.Diff = s.calculateDiff(resource.Current, resource.Spec)
	}

	// Dependency analysis removed - use kubectl for dependency information

	return change, nil
}

// calculateDiff calculates the difference between two states
func (s *Simulator) calculateDiff(before, after map[string]interface{}) *types.ResourceDiff {
	diff := &types.ResourceDiff{
		Before:  before,
		After:   after,
		Changes: make([]types.FieldChange, 0),
	}

	// Compare fields
	s.compareFields("", before, after, &diff.Changes)

	return diff
}

// compareFields recursively compares fields
func (s *Simulator) compareFields(prefix string, before, after map[string]interface{}, changes *[]types.FieldChange) {
	// Check for modified and deleted fields
	for key, beforeVal := range before {
		path := key
		if prefix != "" {
			path = prefix + "." + key
		}

		afterVal, exists := after[key]
		if !exists {
			*changes = append(*changes, types.FieldChange{
				Path:     path,
				OldValue: beforeVal,
				Type:     "delete",
			})
			continue
		}

		if !reflect.DeepEqual(beforeVal, afterVal) {
			// Check if both are maps for recursive comparison
			beforeMap, beforeIsMap := beforeVal.(map[string]interface{})
			afterMap, afterIsMap := afterVal.(map[string]interface{})
			if beforeIsMap && afterIsMap {
				s.compareFields(path, beforeMap, afterMap, changes)
			} else {
				*changes = append(*changes, types.FieldChange{
					Path:     path,
					OldValue: beforeVal,
					NewValue: afterVal,
					Type:     "modify",
				})
			}
		}
	}

	// Check for added fields
	for key, afterVal := range after {
		path := key
		if prefix != "" {
			path = prefix + "." + key
		}

		if _, exists := before[key]; !exists {
			*changes = append(*changes, types.FieldChange{
				Path:     path,
				NewValue: afterVal,
				Type:     "add",
			})
		}
	}
}

// analyzeImpact performs impact analysis on the changes
// Note: Graph-based impact analysis removed - returns basic impact info
func (s *Simulator) analyzeImpact(ctx context.Context, resources []types.SimulatedResource) (*types.SimulationImpact, error) {
	impact := &types.SimulationImpact{
		DirectlyAffected:     make([]string, 0),
		TransitivelyAffected: make([]string, 0),
		AffectedTeams:        make([]string, 0),
		AffectedEnvironments: make([]string, 0),
		RiskLevel:            "unknown",
	}

	// Basic impact based on resource count
	for _, resource := range resources {
		resourceID := fmt.Sprintf("%s/%s", resource.Kind, resource.Name)
		impact.DirectlyAffected = append(impact.DirectlyAffected, resourceID)
	}

	impact.BlastRadius = len(impact.DirectlyAffected)

	// Determine risk level
	impact.RiskLevel = s.calculateRiskLevel(impact)

	// Check if approval is required
	impact.RequiresApproval, impact.ApprovalReason = s.checkApprovalRequired(impact)

	return impact, nil
}

// calculateSummary calculates the simulation summary
func (s *Simulator) calculateSummary(report *types.SimulationReport) types.SimulationSummary {
	summary := types.SimulationSummary{
		TotalResources:       len(report.ResourceChanges),
		ValidationErrorCount: len(report.ValidationErrors),
		PolicyViolationCount: len(report.PolicyViolations),
		ComplianceIssueCount: len(report.ComplianceIssues),
	}

	for _, change := range report.ResourceChanges {
		switch change.Action {
		case "create":
			summary.ToCreate++
		case "update":
			summary.ToUpdate++
		case "delete":
			summary.ToDelete++
		default:
			summary.Unchanged++
		}
	}

	if report.CostEstimate != nil {
		summary.EstimatedCostChange = report.CostEstimate.MonthlyCostChange
	}

	if report.ImpactAnalysis != nil {
		summary.BlastRadius = report.ImpactAnalysis.BlastRadius
	}

	return summary
}

// determineCanProceed checks if the simulation result allows proceeding
func (s *Simulator) determineCanProceed(report *types.SimulationReport) {
	// Check for blocking validation errors
	for _, err := range report.ValidationErrors {
		if err.Severity == "error" {
			report.CanProceed = false
			report.BlockedReason = fmt.Sprintf("Validation error: %s", err.Message)
			return
		}
	}

	// Check for blocking policy violations
	for _, violation := range report.PolicyViolations {
		if violation.Enforcement == "enforce" && violation.Severity == "error" {
			report.CanProceed = false
			report.BlockedReason = fmt.Sprintf("Policy violation: %s - %s", violation.Policy, violation.Message)
			return
		}
	}

	// Check for critical compliance issues
	for _, issue := range report.ComplianceIssues {
		if issue.Severity == "critical" {
			report.CanProceed = false
			report.BlockedReason = fmt.Sprintf("Compliance issue: %s - %s", issue.Framework, issue.Message)
			return
		}
	}

	report.CanProceed = true
}

// generateRecommendations generates recommendations based on simulation results
func (s *Simulator) generateRecommendations(report *types.SimulationReport) []string {
	var recommendations []string

	// Based on impact
	if report.ImpactAnalysis != nil {
		if report.ImpactAnalysis.RiskLevel == "critical" || report.ImpactAnalysis.RiskLevel == "high" {
			recommendations = append(recommendations, "High-impact change - consider scheduling during maintenance window")
		}
		if report.ImpactAnalysis.RequiresApproval {
			recommendations = append(recommendations, fmt.Sprintf("Approval required: %s", report.ImpactAnalysis.ApprovalReason))
		}
		if len(report.ImpactAnalysis.AffectedTeams) > 1 {
			recommendations = append(recommendations, fmt.Sprintf("Notify affected teams: %s", strings.Join(report.ImpactAnalysis.AffectedTeams, ", ")))
		}
	}

	// Based on cost
	if report.CostEstimate != nil {
		if report.CostEstimate.MonthlyCostChange > 1000 {
			recommendations = append(recommendations, fmt.Sprintf("Significant cost increase: $%.2f/month - consider cost optimization", report.CostEstimate.MonthlyCostChange))
		}
		if report.CostEstimate.PercentageChange > 50 {
			recommendations = append(recommendations, "Cost increase exceeds 50% - review resource sizing")
		}
	}

	// Based on changes
	if report.Summary.ToDelete > 0 {
		recommendations = append(recommendations, fmt.Sprintf("%d resources will be deleted - ensure backups exist", report.Summary.ToDelete))
	}

	// Based on warnings
	if len(report.PolicyViolations) > 0 {
		recommendations = append(recommendations, fmt.Sprintf("Review %d policy violation(s) before proceeding", len(report.PolicyViolations)))
	}

	if len(report.ComplianceIssues) > 0 {
		recommendations = append(recommendations, fmt.Sprintf("Address %d compliance issue(s) for audit readiness", len(report.ComplianceIssues)))
	}

	return recommendations
}

// calculateRiskLevel determines the risk level of the impact
func (s *Simulator) calculateRiskLevel(impact *types.SimulationImpact) string {
	if impact.CriticalResourcesAffected > 0 || impact.BlastRadius > 20 {
		return "critical"
	}
	if impact.BlastRadius > 10 || len(impact.AffectedEnvironments) > 2 {
		return "high"
	}
	if impact.BlastRadius > 5 || len(impact.AffectedTeams) > 2 {
		return "medium"
	}
	return "low"
}

// checkApprovalRequired determines if approval is needed
func (s *Simulator) checkApprovalRequired(impact *types.SimulationImpact) (bool, string) {
	if impact.CriticalResourcesAffected > 0 {
		return true, "Changes affect critical resources"
	}
	if impact.RiskLevel == "critical" {
		return true, "Critical risk level"
	}
	if impact.RiskLevel == "high" && len(impact.AffectedTeams) > 1 {
		return true, "High risk change affecting multiple teams"
	}
	return false, ""
}

// generateID generates a unique ID with a prefix
func generateID(prefix string) string {
	return fmt.Sprintf("%s_%d", prefix, time.Now().UnixNano())
}

// FormatReport formats a simulation report for display
func FormatReport(report *types.SimulationReport) string {
	var sb strings.Builder

	sb.WriteString(fmt.Sprintf("Simulation Report: %s\n", report.ID))
	sb.WriteString(fmt.Sprintf("Mode: %s | Status: %s\n", report.Mode, report.Status))
	sb.WriteString(strings.Repeat("=", 60) + "\n\n")

	// Summary
	sb.WriteString("SUMMARY\n")
	sb.WriteString(strings.Repeat("-", 40) + "\n")
	sb.WriteString(fmt.Sprintf("Total Resources: %d\n", report.Summary.TotalResources))
	sb.WriteString(fmt.Sprintf("  To Create: %d\n", report.Summary.ToCreate))
	sb.WriteString(fmt.Sprintf("  To Update: %d\n", report.Summary.ToUpdate))
	sb.WriteString(fmt.Sprintf("  To Delete: %d\n", report.Summary.ToDelete))
	sb.WriteString(fmt.Sprintf("  Unchanged: %d\n", report.Summary.Unchanged))
	sb.WriteString(fmt.Sprintf("Blast Radius: %d\n", report.Summary.BlastRadius))
	sb.WriteString("\n")

	// Validation errors
	if len(report.ValidationErrors) > 0 {
		sb.WriteString(fmt.Sprintf("VALIDATION ERRORS (%d)\n", len(report.ValidationErrors)))
		sb.WriteString(strings.Repeat("-", 40) + "\n")
		for _, err := range report.ValidationErrors {
			sb.WriteString(fmt.Sprintf("  [%s] %s.%s: %s\n", err.Severity, err.Resource, err.Field, err.Message))
		}
		sb.WriteString("\n")
	}

	// Policy violations
	if len(report.PolicyViolations) > 0 {
		sb.WriteString(fmt.Sprintf("POLICY VIOLATIONS (%d)\n", len(report.PolicyViolations)))
		sb.WriteString(strings.Repeat("-", 40) + "\n")
		for _, v := range report.PolicyViolations {
			sb.WriteString(fmt.Sprintf("  [%s] %s: %s\n", v.Severity, v.Policy, v.Message))
		}
		sb.WriteString("\n")
	}

	// Cost estimate
	if report.CostEstimate != nil {
		sb.WriteString("COST ESTIMATE\n")
		sb.WriteString(strings.Repeat("-", 40) + "\n")
		sb.WriteString(fmt.Sprintf("  Current: $%.2f/month\n", report.CostEstimate.CurrentMonthlyCost))
		sb.WriteString(fmt.Sprintf("  Estimated: $%.2f/month\n", report.CostEstimate.EstimatedMonthlyCost))
		sb.WriteString(fmt.Sprintf("  Change: $%.2f/month (%.1f%%)\n", report.CostEstimate.MonthlyCostChange, report.CostEstimate.PercentageChange))
		sb.WriteString("\n")
	}

	// Impact analysis
	if report.ImpactAnalysis != nil {
		sb.WriteString("IMPACT ANALYSIS\n")
		sb.WriteString(strings.Repeat("-", 40) + "\n")
		sb.WriteString(fmt.Sprintf("  Risk Level: %s\n", report.ImpactAnalysis.RiskLevel))
		sb.WriteString(fmt.Sprintf("  Blast Radius: %d resources\n", report.ImpactAnalysis.BlastRadius))
		sb.WriteString(fmt.Sprintf("  Critical Resources: %d\n", report.ImpactAnalysis.CriticalResourcesAffected))
		if len(report.ImpactAnalysis.AffectedTeams) > 0 {
			sb.WriteString(fmt.Sprintf("  Affected Teams: %s\n", strings.Join(report.ImpactAnalysis.AffectedTeams, ", ")))
		}
		sb.WriteString("\n")
	}

	// Recommendations
	if len(report.Recommendations) > 0 {
		sb.WriteString("RECOMMENDATIONS\n")
		sb.WriteString(strings.Repeat("-", 40) + "\n")
		for _, r := range report.Recommendations {
			sb.WriteString(fmt.Sprintf("  • %s\n", r))
		}
		sb.WriteString("\n")
	}

	// Result
	sb.WriteString(strings.Repeat("=", 60) + "\n")
	if report.CanProceed {
		sb.WriteString("✓ SAFE TO PROCEED\n")
	} else {
		sb.WriteString(fmt.Sprintf("✗ BLOCKED: %s\n", report.BlockedReason))
	}

	return sb.String()
}
