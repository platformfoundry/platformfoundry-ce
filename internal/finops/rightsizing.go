// Package finops provides FinOps capabilities including cost tracking, rightsizing, and anomaly detection.
package finops

import (
	"context"
	"fmt"
	"sort"
	"time"
)

// MetricsClient interface for querying resource metrics
type MetricsClient interface {
	GetAverage(ctx context.Context, query string, duration time.Duration) (*UsageStats, error)
	GetPercentile(ctx context.Context, query string, percentile float64, duration time.Duration) (float64, error)
}

// CloudProvider interface for cloud resource operations
type CloudProvider interface {
	GetInstanceTypes(ctx context.Context) ([]InstanceType, error)
	GetCurrentCost(ctx context.Context, resourceID string) (float64, error)
	GetResourceDetails(ctx context.Context, resourceID string) (*ResourceDetails, error)
}

// StateBackend interface for state persistence
type StateBackend interface {
	List(ctx context.Context, kind string) ([]interface{}, error)
	Get(ctx context.Context, kind, id string) (interface{}, error)
	Put(ctx context.Context, kind, id string, value interface{}) error
}

// UsageStats represents resource usage statistics
type UsageStats struct {
	Avg float64   `json:"avg"`
	Min float64   `json:"min"`
	Max float64   `json:"max"`
	P50 float64   `json:"p50"`
	P95 float64   `json:"p95"`
	P99 float64   `json:"p99"`
}

// InstanceType represents a cloud instance type
type InstanceType struct {
	Name     string  `json:"name"`
	CPUCores int     `json:"cpuCores"`
	MemoryGB float64 `json:"memoryGB"`
	CostPerHour float64 `json:"costPerHour"`
	Category string  `json:"category"` // general, compute, memory, storage
}

// ResourceDetails contains details about a cloud resource
type ResourceDetails struct {
	ID           string            `json:"id"`
	Name         string            `json:"name"`
	Type         string            `json:"type"`
	InstanceType string            `json:"instanceType"`
	CPUCores     int               `json:"cpuCores"`
	MemoryGB     float64           `json:"memoryGB"`
	Region       string            `json:"region"`
	Tags         map[string]string `json:"tags"`
}

// ResourceSize represents the size/capacity of a resource
type ResourceSize struct {
	InstanceType string  `json:"instanceType"`
	CPUCores     int     `json:"cpuCores"`
	MemoryGB     float64 `json:"memoryGB"`
	Cost         float64 `json:"cost"` // Monthly cost
}

// RightsizingRecommendation represents a recommendation to resize a resource
type RightsizingRecommendation struct {
	ResourceID       string       `json:"resourceId"`
	ResourceName     string       `json:"resourceName"`
	ResourceType     string       `json:"resourceType"`
	Team             string       `json:"team,omitempty"`
	CurrentSize      ResourceSize `json:"currentSize"`
	RecommendedSize  ResourceSize `json:"recommendedSize"`
	Action           string       `json:"action"` // downsize, upsize, terminate
	Confidence       float64      `json:"confidence"`
	EstimatedSavings float64      `json:"estimatedSavings"` // Monthly savings
	Reason           string       `json:"reason"`
	UsageMetrics     *UsageMetrics `json:"usageMetrics,omitempty"`
	CreatedAt        time.Time    `json:"createdAt"`
}

// UsageMetrics contains resource usage metrics
type UsageMetrics struct {
	CPUUsage    *UsageStats `json:"cpuUsage,omitempty"`
	MemoryUsage *UsageStats `json:"memoryUsage,omitempty"`
	DiskUsage   *UsageStats `json:"diskUsage,omitempty"`
	NetworkIO   *UsageStats `json:"networkIO,omitempty"`
}

// RightsizingConfig contains configuration for the rightsizing engine
type RightsizingConfig struct {
	AnalysisPeriod       time.Duration `json:"analysisPeriod"`       // How far back to analyze (default 14 days)
	MinSavingsPercent    float64       `json:"minSavingsPercent"`    // Minimum savings to recommend (default 20%)
	MinConfidence        float64       `json:"minConfidence"`        // Minimum confidence threshold (default 0.7)
	CPUTargetUtilization float64       `json:"cpuTargetUtilization"` // Target CPU utilization (default 70%)
	MemTargetUtilization float64       `json:"memTargetUtilization"` // Target memory utilization (default 80%)
	ExcludeTags          []string      `json:"excludeTags"`          // Tags to exclude from analysis
}

// RightsizingEngine analyzes resources and provides rightsizing recommendations
type RightsizingEngine struct {
	metricsClient MetricsClient
	cloudProvider CloudProvider
	stateBackend  StateBackend
	config        RightsizingConfig
	instanceTypes []InstanceType
}

// NewRightsizingEngine creates a new rightsizing engine
func NewRightsizingEngine(metrics MetricsClient, cloud CloudProvider, state StateBackend, config RightsizingConfig) *RightsizingEngine {
	// Set defaults
	if config.AnalysisPeriod == 0 {
		config.AnalysisPeriod = 14 * 24 * time.Hour
	}
	if config.MinSavingsPercent == 0 {
		config.MinSavingsPercent = 20
	}
	if config.MinConfidence == 0 {
		config.MinConfidence = 0.7
	}
	if config.CPUTargetUtilization == 0 {
		config.CPUTargetUtilization = 70
	}
	if config.MemTargetUtilization == 0 {
		config.MemTargetUtilization = 80
	}

	return &RightsizingEngine{
		metricsClient: metrics,
		cloudProvider: cloud,
		stateBackend:  state,
		config:        config,
	}
}

// Analyze analyzes all workloads and returns rightsizing recommendations
func (e *RightsizingEngine) Analyze(ctx context.Context) ([]RightsizingRecommendation, error) {
	var recommendations []RightsizingRecommendation

	// Load instance types if not cached
	if len(e.instanceTypes) == 0 && e.cloudProvider != nil {
		types, err := e.cloudProvider.GetInstanceTypes(ctx)
		if err == nil {
			e.instanceTypes = types
		}
	}

	// Get all workloads
	workloads, err := e.stateBackend.List(ctx, "Workload")
	if err != nil {
		return nil, fmt.Errorf("failed to list workloads: %w", err)
	}

	for _, w := range workloads {
		workload, ok := w.(map[string]interface{})
		if !ok {
			continue
		}

		rec, err := e.analyzeWorkload(ctx, workload)
		if err != nil {
			continue // Log error in production
		}

		if rec != nil {
			recommendations = append(recommendations, *rec)
		}
	}

	// Sort by estimated savings (highest first)
	sort.Slice(recommendations, func(i, j int) bool {
		return recommendations[i].EstimatedSavings > recommendations[j].EstimatedSavings
	})

	return recommendations, nil
}

// AnalyzeResource analyzes a specific resource
func (e *RightsizingEngine) AnalyzeResource(ctx context.Context, resourceID string) (*RightsizingRecommendation, error) {
	// Get resource details
	var details *ResourceDetails
	if e.cloudProvider != nil {
		var err error
		details, err = e.cloudProvider.GetResourceDetails(ctx, resourceID)
		if err != nil {
			return nil, fmt.Errorf("failed to get resource details: %w", err)
		}
	} else {
		// Mock for testing
		details = &ResourceDetails{
			ID:           resourceID,
			Name:         "mock-resource",
			Type:         "compute",
			InstanceType: "m5.large",
			CPUCores:     2,
			MemoryGB:     8,
		}
	}

	// Get usage metrics
	usage, err := e.getUsageMetrics(ctx, resourceID)
	if err != nil {
		return nil, fmt.Errorf("failed to get usage metrics: %w", err)
	}

	// Calculate optimal size
	current := ResourceSize{
		InstanceType: details.InstanceType,
		CPUCores:     details.CPUCores,
		MemoryGB:     details.MemoryGB,
		Cost:         e.getMonthlyCost(details.InstanceType),
	}

	optimal := e.calculateOptimalSize(usage, current)
	if optimal == nil {
		return nil, nil // No recommendation
	}

	// Check if savings meet threshold
	savingsPercent := (current.Cost - optimal.Cost) / current.Cost * 100
	if savingsPercent < e.config.MinSavingsPercent {
		return nil, nil
	}

	confidence := e.calculateConfidence(usage)
	if confidence < e.config.MinConfidence {
		return nil, nil
	}

	action := "downsize"
	if optimal.Cost > current.Cost {
		action = "upsize"
		savingsPercent = -savingsPercent
	}

	return &RightsizingRecommendation{
		ResourceID:       resourceID,
		ResourceName:     details.Name,
		ResourceType:     details.Type,
		CurrentSize:      current,
		RecommendedSize:  *optimal,
		Action:           action,
		Confidence:       confidence,
		EstimatedSavings: (current.Cost - optimal.Cost),
		Reason:           e.generateReason(usage, current, *optimal),
		UsageMetrics:     usage,
		CreatedAt:        time.Now(),
	}, nil
}

// analyzeWorkload analyzes a single workload
func (e *RightsizingEngine) analyzeWorkload(ctx context.Context, workload map[string]interface{}) (*RightsizingRecommendation, error) {
	resourceID, _ := workload["id"].(string)
	if resourceID == "" {
		return nil, fmt.Errorf("workload has no ID")
	}

	return e.AnalyzeResource(ctx, resourceID)
}

// getUsageMetrics retrieves usage metrics for a resource
func (e *RightsizingEngine) getUsageMetrics(ctx context.Context, resourceID string) (*UsageMetrics, error) {
	metrics := &UsageMetrics{}

	if e.metricsClient == nil {
		// Return mock metrics for testing
		metrics.CPUUsage = &UsageStats{Avg: 15, Min: 5, Max: 45, P50: 12, P95: 35, P99: 42}
		metrics.MemoryUsage = &UsageStats{Avg: 35, Min: 25, Max: 55, P50: 33, P95: 50, P99: 53}
		return metrics, nil
	}

	// Query CPU usage
	cpuQuery := fmt.Sprintf("cpu_usage{resource=%q}", resourceID)
	cpuStats, err := e.metricsClient.GetAverage(ctx, cpuQuery, e.config.AnalysisPeriod)
	if err == nil {
		metrics.CPUUsage = cpuStats
	}

	// Query memory usage
	memQuery := fmt.Sprintf("memory_usage{resource=%q}", resourceID)
	memStats, err := e.metricsClient.GetAverage(ctx, memQuery, e.config.AnalysisPeriod)
	if err == nil {
		metrics.MemoryUsage = memStats
	}

	return metrics, nil
}

// calculateOptimalSize calculates the optimal resource size based on usage
func (e *RightsizingEngine) calculateOptimalSize(usage *UsageMetrics, current ResourceSize) *ResourceSize {
	if usage == nil || usage.CPUUsage == nil || usage.MemoryUsage == nil {
		return nil
	}

	// Calculate required resources based on P95 usage with target utilization
	requiredCPU := float64(current.CPUCores) * (usage.CPUUsage.P95 / 100) / (e.config.CPUTargetUtilization / 100)
	requiredMem := current.MemoryGB * (usage.MemoryUsage.P95 / 100) / (e.config.MemTargetUtilization / 100)

	// Find best matching instance type
	var bestMatch *InstanceType
	var bestCost float64 = -1

	for i := range e.instanceTypes {
		it := &e.instanceTypes[i]
		if float64(it.CPUCores) >= requiredCPU && it.MemoryGB >= requiredMem {
			if bestCost < 0 || it.CostPerHour < bestCost {
				bestMatch = it
				bestCost = it.CostPerHour
			}
		}
	}

	if bestMatch == nil {
		// Use a default smaller size if no match found
		return &ResourceSize{
			InstanceType: "m5.medium",
			CPUCores:     1,
			MemoryGB:     4,
			Cost:         current.Cost * 0.5,
		}
	}

	return &ResourceSize{
		InstanceType: bestMatch.Name,
		CPUCores:     bestMatch.CPUCores,
		MemoryGB:     bestMatch.MemoryGB,
		Cost:         bestMatch.CostPerHour * 24 * 30, // Monthly cost
	}
}

// calculateConfidence calculates confidence in the recommendation
func (e *RightsizingEngine) calculateConfidence(usage *UsageMetrics) float64 {
	if usage == nil || usage.CPUUsage == nil {
		return 0.5
	}

	// Higher confidence when usage is consistent (low variance)
	cpuVariance := usage.CPUUsage.Max - usage.CPUUsage.Min
	if cpuVariance < 20 {
		return 0.9
	}
	if cpuVariance < 40 {
		return 0.8
	}
	if cpuVariance < 60 {
		return 0.7
	}
	return 0.6
}

// generateReason generates a human-readable reason for the recommendation
func (e *RightsizingEngine) generateReason(usage *UsageMetrics, current, recommended ResourceSize) string {
	if usage == nil || usage.CPUUsage == nil {
		return "Insufficient usage data for detailed analysis"
	}

	return fmt.Sprintf(
		"CPU avg %.1f%% (P95: %.1f%%), Memory avg %.1f%% (P95: %.1f%%). Recommend %s (%d vCPU, %.1fGB) instead of %s (%d vCPU, %.1fGB)",
		usage.CPUUsage.Avg, usage.CPUUsage.P95,
		usage.MemoryUsage.Avg, usage.MemoryUsage.P95,
		recommended.InstanceType, recommended.CPUCores, recommended.MemoryGB,
		current.InstanceType, current.CPUCores, current.MemoryGB,
	)
}

// getMonthlyCost gets the monthly cost for an instance type
func (e *RightsizingEngine) getMonthlyCost(instanceType string) float64 {
	for _, it := range e.instanceTypes {
		if it.Name == instanceType {
			return it.CostPerHour * 24 * 30
		}
	}
	return 100 // Default cost
}

// GetTotalPotentialSavings calculates total potential monthly savings
func (e *RightsizingEngine) GetTotalPotentialSavings(recommendations []RightsizingRecommendation) float64 {
	var total float64
	for _, r := range recommendations {
		if r.EstimatedSavings > 0 {
			total += r.EstimatedSavings
		}
	}
	return total
}

// FilterByConfidence filters recommendations by minimum confidence
func FilterByConfidence(recommendations []RightsizingRecommendation, minConfidence float64) []RightsizingRecommendation {
	var filtered []RightsizingRecommendation
	for _, r := range recommendations {
		if r.Confidence >= minConfidence {
			filtered = append(filtered, r)
		}
	}
	return filtered
}

// FilterByAction filters recommendations by action type
func FilterByAction(recommendations []RightsizingRecommendation, action string) []RightsizingRecommendation {
	var filtered []RightsizingRecommendation
	for _, r := range recommendations {
		if r.Action == action {
			filtered = append(filtered, r)
		}
	}
	return filtered
}
