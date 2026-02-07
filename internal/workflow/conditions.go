package workflow

import (
	"context"
	"time"
)

// TestsPassingChecker checks if tests are passing
type TestsPassingChecker struct{}

func (c *TestsPassingChecker) Check(ctx context.Context, condition WorkflowCondition, exec *WorkflowExecution) (*ConditionResult, error) {
	result := &ConditionResult{
		Type:      ConditionTestsPassing,
		CheckedAt: time.Now(),
		Details:   make(map[string]interface{}),
	}

	// In a real implementation, this would check CI/CD status
	// For now, we'll check metadata for test results
	if testsPassing, ok := exec.Metadata["tests_passing"].(bool); ok {
		if testsPassing {
			result.Status = ConditionStatusPassed
			result.Message = "All tests passing"
		} else {
			result.Status = ConditionStatusFailed
			result.Message = "Tests are failing"
		}
	} else {
		// Default to passed if no explicit test result provided
		result.Status = ConditionStatusPassed
		result.Message = "Tests status assumed passing (no explicit result)"
	}

	return result, nil
}

// SecurityScanChecker checks security scan results
type SecurityScanChecker struct{}

func (c *SecurityScanChecker) Check(ctx context.Context, condition WorkflowCondition, exec *WorkflowExecution) (*ConditionResult, error) {
	result := &ConditionResult{
		Type:      ConditionSecurityScan,
		CheckedAt: time.Now(),
		Details:   make(map[string]interface{}),
	}

	// Check for critical vulnerabilities in metadata
	criticalCount := 0
	if critical, ok := exec.Metadata["security_critical_count"].(int); ok {
		criticalCount = critical
	}

	highCount := 0
	if high, ok := exec.Metadata["security_high_count"].(int); ok {
		highCount = high
	}

	result.Details["critical"] = criticalCount
	result.Details["high"] = highCount

	maxCritical := condition.MaxCritical
	if maxCritical == 0 {
		maxCritical = 0 // Default: no critical vulnerabilities allowed
	}

	if criticalCount > maxCritical {
		result.Status = ConditionStatusFailed
		result.Message = "Security scan failed: critical vulnerabilities found"
	} else {
		result.Status = ConditionStatusPassed
		result.Message = "Security scan passed"
	}

	return result, nil
}

// TestCoverageChecker checks test coverage threshold
type TestCoverageChecker struct{}

func (c *TestCoverageChecker) Check(ctx context.Context, condition WorkflowCondition, exec *WorkflowExecution) (*ConditionResult, error) {
	result := &ConditionResult{
		Type:      ConditionTestCoverage,
		CheckedAt: time.Now(),
		Details:   make(map[string]interface{}),
	}

	coverage := 0
	if cov, ok := exec.Metadata["test_coverage"].(int); ok {
		coverage = cov
	} else if cov, ok := exec.Metadata["test_coverage"].(float64); ok {
		coverage = int(cov)
	}

	threshold := condition.Threshold
	if threshold == 0 {
		threshold = 80 // Default 80% threshold
	}

	result.Details["coverage"] = coverage
	result.Details["threshold"] = threshold

	if coverage >= threshold {
		result.Status = ConditionStatusPassed
		result.Message = "Test coverage meets threshold"
	} else {
		result.Status = ConditionStatusFailed
		result.Message = "Test coverage below threshold"
	}

	return result, nil
}

// PerformanceTestChecker checks performance test results
type PerformanceTestChecker struct{}

func (c *PerformanceTestChecker) Check(ctx context.Context, condition WorkflowCondition, exec *WorkflowExecution) (*ConditionResult, error) {
	result := &ConditionResult{
		Type:      ConditionPerformanceTest,
		CheckedAt: time.Now(),
		Details:   make(map[string]interface{}),
	}

	// Check performance test results in metadata
	perfPassing := true
	if passing, ok := exec.Metadata["performance_tests_passing"].(bool); ok {
		perfPassing = passing
	}

	if perfPassing {
		result.Status = ConditionStatusPassed
		result.Message = "Performance tests passing"
	} else {
		result.Status = ConditionStatusFailed
		result.Message = "Performance tests failing"
	}

	// Include latency details if available
	if p50, ok := exec.Metadata["latency_p50"].(float64); ok {
		result.Details["latency_p50_ms"] = p50
	}
	if p95, ok := exec.Metadata["latency_p95"].(float64); ok {
		result.Details["latency_p95_ms"] = p95
	}
	if p99, ok := exec.Metadata["latency_p99"].(float64); ok {
		result.Details["latency_p99_ms"] = p99
	}

	return result, nil
}

// CustomConditionChecker handles custom conditions
type CustomConditionChecker struct {
	httpClient HTTPClient
}

// HTTPClient interface for making HTTP requests (for custom endpoint conditions)
type HTTPClient interface {
	Get(ctx context.Context, url string) ([]byte, error)
}

func NewCustomConditionChecker(client HTTPClient) *CustomConditionChecker {
	return &CustomConditionChecker{httpClient: client}
}

func (c *CustomConditionChecker) Check(ctx context.Context, condition WorkflowCondition, exec *WorkflowExecution) (*ConditionResult, error) {
	result := &ConditionResult{
		Type:      ConditionCustom,
		CheckedAt: time.Now(),
		Details:   make(map[string]interface{}),
	}

	if condition.Custom == nil {
		result.Status = ConditionStatusSkipped
		result.Message = "No custom condition configuration"
		return result, nil
	}

	// Handle endpoint-based custom conditions
	if condition.Custom.Endpoint != "" && c.httpClient != nil {
		resp, err := c.httpClient.Get(ctx, condition.Custom.Endpoint)
		if err != nil {
			result.Status = ConditionStatusFailed
			result.Message = "Custom endpoint check failed"
			result.Details["error"] = err.Error()
			return result, nil
		}

		// Simple check: endpoint returned successfully
		result.Status = ConditionStatusPassed
		result.Message = "Custom endpoint check passed"
		result.Details["response_length"] = len(resp)
		return result, nil
	}

	// Default: skip if no implementation
	result.Status = ConditionStatusSkipped
	result.Message = "Custom condition not fully configured"
	return result, nil
}
