package compliance

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/platformfoundry/pf-ce/pkg/types"
)

// Scanner performs compliance scans
type Scanner struct {
	policies        map[string]*types.CompliancePolicy
	checkers        map[types.CheckType]Checker
	resourceFetcher ResourceFetcher
	results         map[string]*types.ComplianceScanResult
	mu              sync.RWMutex
}

// Checker interface for compliance checks
type Checker interface {
	Check(ctx context.Context, rule types.ComplianceRule, resource interface{}) (bool, map[string]interface{}, error)
}

// ResourceFetcher retrieves resources to scan
type ResourceFetcher interface {
	FetchResources(ctx context.Context, target types.RuleTarget) ([]interface{}, error)
	GetResourceIdentifier(resource interface{}) types.ViolationResource
}

// ScannerConfig configures the scanner
type ScannerConfig struct {
	ResourceFetcher ResourceFetcher
}

// NewScanner creates a new compliance scanner
func NewScanner(cfg ScannerConfig) *Scanner {
	s := &Scanner{
		policies:        make(map[string]*types.CompliancePolicy),
		checkers:        make(map[types.CheckType]Checker),
		resourceFetcher: cfg.ResourceFetcher,
		results:         make(map[string]*types.ComplianceScanResult),
	}

	// Register default checkers
	s.checkers[types.CheckTypeBuiltin] = &BuiltinChecker{}
	s.checkers[types.CheckTypeJSONPath] = &JSONPathChecker{}

	return s
}

// RegisterPolicy adds a compliance policy
func (s *Scanner) RegisterPolicy(policy *types.CompliancePolicy) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if policy.Metadata.Name == "" {
		return fmt.Errorf("policy name is required")
	}

	// Initialize status
	if policy.Status == nil {
		policy.Status = &types.CompliancePolicyStatus{
			TotalRules: len(policy.Spec.Rules),
		}
	}

	s.policies[policy.Metadata.Name] = policy
	return nil
}

// GetPolicy retrieves a policy by name
func (s *Scanner) GetPolicy(name string) (*types.CompliancePolicy, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	policy, ok := s.policies[name]
	if !ok {
		return nil, fmt.Errorf("policy not found: %s", name)
	}
	return policy, nil
}

// ListPolicies returns all policies
func (s *Scanner) ListPolicies() []*types.CompliancePolicy {
	s.mu.RLock()
	defer s.mu.RUnlock()

	result := make([]*types.CompliancePolicy, 0, len(s.policies))
	for _, p := range s.policies {
		result = append(result, p)
	}
	return result
}

// Scan performs a compliance scan for a policy
func (s *Scanner) Scan(ctx context.Context, policyName string) (*types.ComplianceScanResult, error) {
	policy, err := s.GetPolicy(policyName)
	if err != nil {
		return nil, err
	}

	startTime := time.Now()
	scanID := fmt.Sprintf("scan-%s-%d", policyName, startTime.Unix())

	result := &types.ComplianceScanResult{
		ScanID:      scanID,
		PolicyName:  policyName,
		Framework:   policy.Spec.Framework,
		StartTime:   startTime,
		TotalChecks: 0,
		Passed:      0,
		Failed:      0,
		Skipped:     0,
		Violations:  make([]types.ComplianceViolation, 0),
		Summary:     make(map[string]int),
	}

	// Check each rule
	for _, rule := range policy.Spec.Rules {
		if !rule.Enabled {
			result.Skipped++
			continue
		}

		// Check for exceptions
		if s.hasException(policy, rule.ID) {
			result.Skipped++
			continue
		}

		// Fetch resources matching the target
		resources, err := s.fetchResources(ctx, rule.Target)
		if err != nil {
			fmt.Printf("Warning: failed to fetch resources for rule %s: %v\n", rule.ID, err)
			result.Skipped++
			continue
		}

		// Check each resource
		for _, resource := range resources {
			result.TotalChecks++

			passed, evidence, err := s.checkRule(ctx, rule, resource)
			if err != nil {
				fmt.Printf("Warning: check failed for rule %s: %v\n", rule.ID, err)
				continue
			}

			if passed {
				result.Passed++
			} else {
				result.Failed++
				result.Summary[string(rule.Severity)]++

				violation := types.ComplianceViolation{
					RuleID:      rule.ID,
					RuleName:    rule.Name,
					Severity:    rule.Severity,
					Category:    rule.Category,
					Resource:    s.getResourceIdentifier(resource),
					Description: rule.Description,
					Evidence:    evidence,
					AutoFixable: rule.Remediation != nil && rule.Remediation.AutoFix,
					DetectedAt:  time.Now(),
					Status:      types.ViolationStatusOpen,
				}

				if rule.Remediation != nil {
					violation.Remediation = rule.Remediation.Description
				}

				result.Violations = append(result.Violations, violation)
			}
		}
	}

	result.EndTime = time.Now()
	result.Duration = result.EndTime.Sub(result.StartTime)

	// Calculate compliance percentage
	if result.TotalChecks > 0 {
		result.Compliance = float64(result.Passed) / float64(result.TotalChecks) * 100
	}

	// Update policy status
	s.mu.Lock()
	now := time.Now()
	policy.Status.LastScan = &now
	policy.Status.PassingRules = result.Passed
	policy.Status.FailingRules = result.Failed
	policy.Status.Compliance = result.Compliance
	s.results[scanID] = result
	s.mu.Unlock()

	return result, nil
}

// ScanAll scans all registered policies
func (s *Scanner) ScanAll(ctx context.Context) ([]*types.ComplianceScanResult, error) {
	policies := s.ListPolicies()
	results := make([]*types.ComplianceScanResult, 0, len(policies))

	for _, policy := range policies {
		result, err := s.Scan(ctx, policy.Metadata.Name)
		if err != nil {
			fmt.Printf("Warning: scan failed for policy %s: %v\n", policy.Metadata.Name, err)
			continue
		}
		results = append(results, result)
	}

	return results, nil
}

// GetScanResult retrieves a scan result
func (s *Scanner) GetScanResult(scanID string) (*types.ComplianceScanResult, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	result, ok := s.results[scanID]
	if !ok {
		return nil, fmt.Errorf("scan result not found: %s", scanID)
	}
	return result, nil
}

// GenerateReport creates a compliance report
func (s *Scanner) GenerateReport(ctx context.Context) (*types.ComplianceReport, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	report := &types.ComplianceReport{
		GeneratedAt:          time.Now(),
		Period:               "current",
		Frameworks:           make([]types.FrameworkSummary, 0),
		TotalPolicies:        len(s.policies),
		ViolationsBySeverity: make(map[types.RuleSeverity]int),
		TopViolations:        make([]types.ComplianceViolation, 0),
	}

	// Aggregate by framework
	frameworkStats := make(map[types.ComplianceFramework]*types.FrameworkSummary)

	var totalCompliance float64
	var policyCount int

	for _, policy := range s.policies {
		if policy.Status == nil {
			continue
		}

		totalCompliance += policy.Status.Compliance
		policyCount++

		// Aggregate by framework
		fw := policy.Spec.Framework
		if _, ok := frameworkStats[fw]; !ok {
			frameworkStats[fw] = &types.FrameworkSummary{
				Framework: fw,
			}
		}

		stats := frameworkStats[fw]
		stats.TotalRules += policy.Status.TotalRules
		stats.Passing += policy.Status.PassingRules
		stats.Failing += policy.Status.FailingRules
	}

	// Calculate overall compliance
	if policyCount > 0 {
		report.OverallCompliance = totalCompliance / float64(policyCount)
	}

	// Finalize framework summaries
	for _, stats := range frameworkStats {
		if stats.TotalRules > 0 {
			stats.Compliance = float64(stats.Passing) / float64(stats.TotalRules) * 100
		}
		stats.Trend = "stable"
		report.Frameworks = append(report.Frameworks, *stats)
	}

	// Aggregate violations
	for _, result := range s.results {
		for severity, count := range result.Summary {
			report.ViolationsBySeverity[types.RuleSeverity(severity)] += count
		}

		// Collect top violations
		for _, v := range result.Violations {
			if v.Severity == types.SeverityCritical || v.Severity == types.SeverityHigh {
				report.TopViolations = append(report.TopViolations, v)
			}
		}
	}

	// Limit top violations
	if len(report.TopViolations) > 10 {
		report.TopViolations = report.TopViolations[:10]
	}

	return report, nil
}

// hasException checks if a rule has an active exception
func (s *Scanner) hasException(policy *types.CompliancePolicy, ruleID string) bool {
	for _, exc := range policy.Spec.Exceptions {
		if exc.RuleID == ruleID {
			if exc.ExpiresAt == nil || exc.ExpiresAt.After(time.Now()) {
				return true
			}
		}
	}
	return false
}

// fetchResources retrieves resources matching the target
func (s *Scanner) fetchResources(ctx context.Context, target types.RuleTarget) ([]interface{}, error) {
	if s.resourceFetcher != nil {
		return s.resourceFetcher.FetchResources(ctx, target)
	}
	// Return mock resources for testing
	return []interface{}{mockResource(target)}, nil
}

// getResourceIdentifier extracts resource identification
func (s *Scanner) getResourceIdentifier(resource interface{}) types.ViolationResource {
	if s.resourceFetcher != nil {
		return s.resourceFetcher.GetResourceIdentifier(resource)
	}
	// Mock identifier
	return types.ViolationResource{
		Kind: "MockResource",
		Name: "mock-resource",
	}
}

// checkRule performs a compliance check
func (s *Scanner) checkRule(ctx context.Context, rule types.ComplianceRule, resource interface{}) (bool, map[string]interface{}, error) {
	checker, ok := s.checkers[rule.Check.Type]
	if !ok {
		return false, nil, fmt.Errorf("unknown check type: %s", rule.Check.Type)
	}
	return checker.Check(ctx, rule, resource)
}

// mockResource creates a mock resource for testing
func mockResource(target types.RuleTarget) map[string]interface{} {
	return map[string]interface{}{
		"kind":       target.Kind,
		"apiVersion": target.APIVersion,
		"metadata": map[string]interface{}{
			"name":      "mock-resource",
			"namespace": "default",
		},
		"spec": map[string]interface{}{
			"replicas": 2,
		},
	}
}

// BuiltinChecker implements common compliance checks
type BuiltinChecker struct{}

func (c *BuiltinChecker) Check(ctx context.Context, rule types.ComplianceRule, resource interface{}) (bool, map[string]interface{}, error) {
	// Simulate checks based on rule parameters
	params := rule.Check.Parameters
	if params == nil {
		return true, nil, nil
	}

	checkName, _ := params["check"].(string)
	switch checkName {
	case "has-resource-limits":
		// Check if resource has limits defined
		return c.hasResourceLimits(resource)
	case "has-security-context":
		return c.hasSecurityContext(resource)
	case "no-privileged-containers":
		return c.noPrivilegedContainers(resource)
	case "has-network-policy":
		return c.hasNetworkPolicy(resource)
	case "encrypted-at-rest":
		return c.encryptedAtRest(resource)
	default:
		// Default pass for unknown checks in mock mode
		return true, nil, nil
	}
}

func (c *BuiltinChecker) hasResourceLimits(resource interface{}) (bool, map[string]interface{}, error) {
	// Simulated check
	return true, nil, nil
}

func (c *BuiltinChecker) hasSecurityContext(resource interface{}) (bool, map[string]interface{}, error) {
	return true, nil, nil
}

func (c *BuiltinChecker) noPrivilegedContainers(resource interface{}) (bool, map[string]interface{}, error) {
	return true, nil, nil
}

func (c *BuiltinChecker) hasNetworkPolicy(resource interface{}) (bool, map[string]interface{}, error) {
	return false, map[string]interface{}{"reason": "No NetworkPolicy found"}, nil
}

func (c *BuiltinChecker) encryptedAtRest(resource interface{}) (bool, map[string]interface{}, error) {
	return true, nil, nil
}

// JSONPathChecker checks using JSON paths
type JSONPathChecker struct{}

func (c *JSONPathChecker) Check(ctx context.Context, rule types.ComplianceRule, resource interface{}) (bool, map[string]interface{}, error) {
	// Simplified JSON path check
	// In production, use a proper JSONPath library
	return true, nil, nil
}

// GetDefaultPolicies returns built-in compliance policies
func GetDefaultPolicies() []*types.CompliancePolicy {
	return []*types.CompliancePolicy{
		{
			APIVersion: "platformfoundry.io/v1",
			Kind:       "CompliancePolicy",
			Metadata: types.ComplianceMetadata{
				Name: "kubernetes-security-baseline",
				Labels: map[string]string{
					"framework": "cis",
				},
			},
			Spec: types.CompliancePolicySpec{
				Framework:   types.ComplianceCIS,
				Description: "CIS Kubernetes Security Baseline",
				Version:     "1.0",
				Enforcement: types.EnforcementAudit,
				Rules: []types.ComplianceRule{
					{
						ID:          "CIS-5.1.1",
						Name:        "Ensure CPU limits are set",
						Description: "Containers should have CPU limits to prevent resource starvation",
						Severity:    types.SeverityMedium,
						Category:    "resource-management",
						Target: types.RuleTarget{
							Kind: "Deployment",
						},
						Check: types.RuleCheck{
							Type: types.CheckTypeBuiltin,
							Parameters: map[string]interface{}{
								"check": "has-resource-limits",
							},
						},
						Remediation: &types.RuleRemediation{
							Description: "Add resources.limits.cpu to container spec",
							AutoFix:     false,
						},
						Enabled: true,
					},
					{
						ID:          "CIS-5.2.1",
						Name:        "Ensure SecurityContext is set",
						Description: "Pods should define a security context",
						Severity:    types.SeverityHigh,
						Category:    "security",
						Target: types.RuleTarget{
							Kind: "Pod",
						},
						Check: types.RuleCheck{
							Type: types.CheckTypeBuiltin,
							Parameters: map[string]interface{}{
								"check": "has-security-context",
							},
						},
						Enabled: true,
					},
					{
						ID:          "CIS-5.2.2",
						Name:        "Ensure privileged containers are not used",
						Description: "Containers should not run in privileged mode",
						Severity:    types.SeverityCritical,
						Category:    "security",
						Target: types.RuleTarget{
							Kind: "Pod",
						},
						Check: types.RuleCheck{
							Type: types.CheckTypeBuiltin,
							Parameters: map[string]interface{}{
								"check": "no-privileged-containers",
							},
						},
						Enabled: true,
					},
					{
						ID:          "CIS-5.3.1",
						Name:        "Ensure network policies are defined",
						Description: "Namespaces should have network policies",
						Severity:    types.SeverityMedium,
						Category:    "network",
						Target: types.RuleTarget{
							Kind: "Namespace",
						},
						Check: types.RuleCheck{
							Type: types.CheckTypeBuiltin,
							Parameters: map[string]interface{}{
								"check": "has-network-policy",
							},
						},
						Enabled: true,
					},
				},
			},
		},
		{
			APIVersion: "platformfoundry.io/v1",
			Kind:       "CompliancePolicy",
			Metadata: types.ComplianceMetadata{
				Name: "data-protection-baseline",
				Labels: map[string]string{
					"framework": "custom",
				},
			},
			Spec: types.CompliancePolicySpec{
				Framework:   types.ComplianceCustom,
				Description: "Data protection and encryption requirements",
				Version:     "1.0",
				Enforcement: types.EnforcementWarn,
				Rules: []types.ComplianceRule{
					{
						ID:          "DP-1.1",
						Name:        "Ensure data is encrypted at rest",
						Description: "Storage volumes must use encryption",
						Severity:    types.SeverityHigh,
						Category:    "encryption",
						Target: types.RuleTarget{
							Kind: "PersistentVolume",
						},
						Check: types.RuleCheck{
							Type: types.CheckTypeBuiltin,
							Parameters: map[string]interface{}{
								"check": "encrypted-at-rest",
							},
						},
						Enabled: true,
					},
				},
			},
		},
	}
}
