package finops

import (
	"context"
	"fmt"
	"math"
	"sort"
	"sync"
	"time"
)

// Manager handles FinOps operations
type Manager struct {
	policies        map[string]*FinOpsPolicy
	resourceCosts   map[string]*ResourceCost
	recommendations map[string]*Recommendation
	anomalies       map[string]*CostAnomaly
	costProvider    CostProvider
	mu              sync.RWMutex
	stopCh          chan struct{}
	running         bool
}

// CostProvider interface for fetching cost data
type CostProvider interface {
	GetResourceCosts(ctx context.Context, start, end time.Time) ([]ResourceCost, error)
	GetCostsByTag(ctx context.Context, tag string, start, end time.Time) (map[string]float64, error)
}

// NewManager creates a new FinOps manager
func NewManager(provider CostProvider) *Manager {
	return &Manager{
		policies:        make(map[string]*FinOpsPolicy),
		resourceCosts:   make(map[string]*ResourceCost),
		recommendations: make(map[string]*Recommendation),
		anomalies:       make(map[string]*CostAnomaly),
		costProvider:    provider,
		stopCh:          make(chan struct{}),
	}
}

// RegisterPolicy registers a FinOps policy
func (m *Manager) RegisterPolicy(ctx context.Context, policy *FinOpsPolicy) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if policy.Metadata.Name == "" {
		return fmt.Errorf("policy name is required")
	}

	// Set defaults
	if policy.APIVersion == "" {
		policy.APIVersion = "platformfoundry.io/v1"
	}
	if policy.Kind == "" {
		policy.Kind = "FinOpsPolicy"
	}

	now := time.Now()
	policy.Metadata.CreatedAt = now
	policy.Metadata.UpdatedAt = now

	// Initialize status
	policy.Status = &FinOpsPolicyStatus{
		LastUpdated:     now,
		BudgetStatus:    make([]BudgetStatus, 0),
		Recommendations: make([]Recommendation, 0),
		Anomalies:       make([]CostAnomaly, 0),
	}

	// Initialize budget status
	for _, budget := range policy.Spec.Budgets {
		status := BudgetStatus{
			Name:        budget.Name,
			Scope:       string(budget.Scope),
			Amount:      budget.Amount,
			Spent:       0,
			SpentPercent: 0,
			Status:      "on_track",
			PeriodStart: getBudgetPeriodStart(budget.Period),
			PeriodEnd:   getBudgetPeriodEnd(budget.Period),
		}
		policy.Status.BudgetStatus = append(policy.Status.BudgetStatus, status)
	}

	m.policies[policy.Metadata.Name] = policy
	return nil
}

// GetPolicy retrieves a policy by name
func (m *Manager) GetPolicy(name string) (*FinOpsPolicy, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	policy, ok := m.policies[name]
	if !ok {
		return nil, fmt.Errorf("policy not found: %s", name)
	}
	return policy, nil
}

// ListPolicies returns all policies
func (m *Manager) ListPolicies() []*FinOpsPolicy {
	m.mu.RLock()
	defer m.mu.RUnlock()

	result := make([]*FinOpsPolicy, 0, len(m.policies))
	for _, p := range m.policies {
		result = append(result, p)
	}
	return result
}

// DeletePolicy removes a policy
func (m *Manager) DeletePolicy(name string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, ok := m.policies[name]; !ok {
		return fmt.Errorf("policy not found: %s", name)
	}

	delete(m.policies, name)
	return nil
}

// Start begins cost monitoring
func (m *Manager) Start(ctx context.Context) error {
	m.mu.Lock()
	if m.running {
		m.mu.Unlock()
		return fmt.Errorf("manager already running")
	}
	m.running = true
	m.stopCh = make(chan struct{})
	m.mu.Unlock()

	go m.monitorLoop(ctx)
	return nil
}

// Stop stops cost monitoring
func (m *Manager) Stop() {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.running {
		close(m.stopCh)
		m.running = false
	}
}

// monitorLoop runs periodic cost analysis
func (m *Manager) monitorLoop(ctx context.Context) {
	ticker := time.NewTicker(1 * time.Hour)
	defer ticker.Stop()

	// Initial analysis
	m.runAnalysis(ctx)

	for {
		select {
		case <-ctx.Done():
			return
		case <-m.stopCh:
			return
		case <-ticker.C:
			m.runAnalysis(ctx)
		}
	}
}

// runAnalysis performs cost analysis
func (m *Manager) runAnalysis(ctx context.Context) {
	m.updateBudgetStatus(ctx)
	m.detectAnomalies(ctx)
	m.generateRecommendations(ctx)
}

// updateBudgetStatus updates budget consumption
func (m *Manager) updateBudgetStatus(ctx context.Context) {
	m.mu.Lock()
	defer m.mu.Unlock()

	for _, policy := range m.policies {
		for i := range policy.Status.BudgetStatus {
			status := &policy.Status.BudgetStatus[i]

			// Simulate cost data (in real implementation, fetch from provider)
			spent := simulateBudgetSpend(status.Amount, status.PeriodStart, status.PeriodEnd)
			status.Spent = spent
			status.SpentPercent = (spent / status.Amount) * 100

			// Calculate forecast
			daysElapsed := time.Since(status.PeriodStart).Hours() / 24
			totalDays := status.PeriodEnd.Sub(status.PeriodStart).Hours() / 24
			if daysElapsed > 0 {
				dailyRate := spent / daysElapsed
				status.Forecast = dailyRate * totalDays
			}

			// Determine status
			if status.SpentPercent >= 100 {
				status.Status = "over_budget"
			} else if status.SpentPercent >= 90 || (status.Forecast > status.Amount) {
				status.Status = "at_risk"
			} else {
				status.Status = "on_track"
			}
		}
		policy.Status.LastUpdated = time.Now()
	}
}

// detectAnomalies detects cost anomalies
func (m *Manager) detectAnomalies(ctx context.Context) {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Simulate anomaly detection
	// In real implementation, compare with historical data
	for _, policy := range m.policies {
		if !policy.Spec.Anomaly.Enabled {
			continue
		}

		// Clear old anomalies
		policy.Status.Anomalies = make([]CostAnomaly, 0)
	}
}

// generateRecommendations generates cost optimization recommendations
func (m *Manager) generateRecommendations(ctx context.Context) {
	m.mu.Lock()
	defer m.mu.Unlock()

	for _, policy := range m.policies {
		recommendations := make([]Recommendation, 0)

		// Right-sizing recommendations
		if policy.Spec.Optimization.RightSizing.Enabled {
			recs := m.generateRightSizingRecommendations(policy)
			recommendations = append(recommendations, recs...)
		}

		// Unused resource recommendations
		if policy.Spec.Optimization.UnusedResources.DetectAfter != "" {
			recs := m.generateUnusedResourceRecommendations(policy)
			recommendations = append(recommendations, recs...)
		}

		// Spot instance recommendations
		if policy.Spec.Optimization.SpotInstances.Enabled {
			recs := m.generateSpotRecommendations(policy)
			recommendations = append(recommendations, recs...)
		}

		policy.Status.Recommendations = recommendations
	}
}

// generateRightSizingRecommendations creates right-sizing recommendations
func (m *Manager) generateRightSizingRecommendations(policy *FinOpsPolicy) []Recommendation {
	recommendations := make([]Recommendation, 0)

	// Simulated recommendations
	// In real implementation, analyze actual resource utilization
	recommendations = append(recommendations, Recommendation{
		ID:              fmt.Sprintf("rs-%d", time.Now().UnixNano()),
		Type:            "right_sizing",
		Resource:        "api-deployment",
		ResourceType:    "Deployment",
		CurrentCost:     150.00,
		RecommendedCost: 75.00,
		MonthlySavings:  75.00,
		Description:     "CPU utilization averages 15%. Recommend reducing from 4 vCPU to 2 vCPU.",
		Action:          "Resize deployment to use smaller instance",
		Confidence:      0.85,
		DetectedAt:      time.Now(),
		Status:          "pending",
	})

	return recommendations
}

// generateUnusedResourceRecommendations detects unused resources
func (m *Manager) generateUnusedResourceRecommendations(policy *FinOpsPolicy) []Recommendation {
	recommendations := make([]Recommendation, 0)

	// Simulated recommendations
	recommendations = append(recommendations, Recommendation{
		ID:              fmt.Sprintf("ur-%d", time.Now().UnixNano()),
		Type:            "unused_resource",
		Resource:        "old-test-volume",
		ResourceType:    "PersistentVolume",
		CurrentCost:     25.00,
		RecommendedCost: 0,
		MonthlySavings:  25.00,
		Description:     "Volume has not been attached to any pod for 14 days.",
		Action:          "Delete unused volume",
		Confidence:      0.95,
		DetectedAt:      time.Now(),
		Status:          "pending",
	})

	return recommendations
}

// generateSpotRecommendations creates spot instance recommendations
func (m *Manager) generateSpotRecommendations(policy *FinOpsPolicy) []Recommendation {
	recommendations := make([]Recommendation, 0)

	// Simulated recommendations
	recommendations = append(recommendations, Recommendation{
		ID:              fmt.Sprintf("sp-%d", time.Now().UnixNano()),
		Type:            "spot",
		Resource:        "batch-worker-nodes",
		ResourceType:    "NodeGroup",
		CurrentCost:     500.00,
		RecommendedCost: 150.00,
		MonthlySavings:  350.00,
		Description:     "Batch workloads are fault-tolerant and can use spot instances.",
		Action:          "Convert to spot instances with 70% savings",
		Confidence:      0.90,
		DetectedAt:      time.Now(),
		Status:          "pending",
	})

	return recommendations
}

// GetRecommendations returns current recommendations
func (m *Manager) GetRecommendations(policyName string) ([]Recommendation, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	policy, ok := m.policies[policyName]
	if !ok {
		return nil, fmt.Errorf("policy not found: %s", policyName)
	}

	return policy.Status.Recommendations, nil
}

// ApplyRecommendation marks a recommendation as applied
func (m *Manager) ApplyRecommendation(policyName, recommendationID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	policy, ok := m.policies[policyName]
	if !ok {
		return fmt.Errorf("policy not found: %s", policyName)
	}

	for i := range policy.Status.Recommendations {
		if policy.Status.Recommendations[i].ID == recommendationID {
			policy.Status.Recommendations[i].Status = "applied"
			return nil
		}
	}

	return fmt.Errorf("recommendation not found: %s", recommendationID)
}

// DismissRecommendation marks a recommendation as dismissed
func (m *Manager) DismissRecommendation(policyName, recommendationID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	policy, ok := m.policies[policyName]
	if !ok {
		return fmt.Errorf("policy not found: %s", policyName)
	}

	for i := range policy.Status.Recommendations {
		if policy.Status.Recommendations[i].ID == recommendationID {
			policy.Status.Recommendations[i].Status = "dismissed"
			return nil
		}
	}

	return fmt.Errorf("recommendation not found: %s", recommendationID)
}

// GenerateReport creates a cost report
func (m *Manager) GenerateReport(ctx context.Context, start, end time.Time) (*CostReport, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	report := &CostReport{
		GeneratedAt:  time.Now(),
		PeriodStart:  start,
		PeriodEnd:    end,
		Currency:     "USD",
		ByTeam:       make(map[string]float64),
		ByApplication: make(map[string]float64),
		ByEnvironment: make(map[string]float64),
		ByService:    make(map[string]float64),
		TopSpenders:  make([]CostItem, 0),
		Recommendations: make([]Recommendation, 0),
	}

	// Simulate cost data
	report.TotalCost = 15000.00
	report.PreviousCost = 14200.00
	report.CostChange = report.TotalCost - report.PreviousCost
	report.ChangePercent = (report.CostChange / report.PreviousCost) * 100

	// By team
	report.ByTeam["platform"] = 5000.00
	report.ByTeam["backend"] = 6000.00
	report.ByTeam["frontend"] = 2500.00
	report.ByTeam["data"] = 1500.00

	// By environment
	report.ByEnvironment["production"] = 10000.00
	report.ByEnvironment["staging"] = 3000.00
	report.ByEnvironment["dev"] = 2000.00

	// By service
	report.ByService["compute"] = 8000.00
	report.ByService["storage"] = 3000.00
	report.ByService["network"] = 2000.00
	report.ByService["database"] = 2000.00

	// Top spenders
	report.TopSpenders = []CostItem{
		{Name: "api-production", Type: "Deployment", Cost: 3500.00, Owner: "backend"},
		{Name: "database-primary", Type: "RDS", Cost: 2000.00, Owner: "platform"},
		{Name: "worker-nodes", Type: "EC2", Cost: 1800.00, Owner: "backend"},
		{Name: "cdn-distribution", Type: "CloudFront", Cost: 1200.00, Owner: "frontend"},
		{Name: "data-lake", Type: "S3", Cost: 1000.00, Owner: "data"},
	}

	// Aggregate recommendations
	for _, policy := range m.policies {
		for _, rec := range policy.Status.Recommendations {
			if rec.Status == "pending" {
				report.Recommendations = append(report.Recommendations, rec)
			}
		}
	}

	// Sort recommendations by savings
	sort.Slice(report.Recommendations, func(i, j int) bool {
		return report.Recommendations[i].MonthlySavings > report.Recommendations[j].MonthlySavings
	})

	// Forecast
	report.Forecast = &CostForecast{
		NextMonth:   report.TotalCost * 1.05,
		NextQuarter: report.TotalCost * 3 * 1.08,
		EndOfYear:   report.TotalCost * 12 * 1.10,
		Confidence:  0.75,
		Trend:       "increasing",
	}

	return report, nil
}

// GetBudgetStatus returns current budget status
func (m *Manager) GetBudgetStatus(policyName string) ([]BudgetStatus, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	policy, ok := m.policies[policyName]
	if !ok {
		return nil, fmt.Errorf("policy not found: %s", policyName)
	}

	return policy.Status.BudgetStatus, nil
}

// GetTotalSavingsOpportunity calculates total potential savings
func (m *Manager) GetTotalSavingsOpportunity() float64 {
	m.mu.RLock()
	defer m.mu.RUnlock()

	total := 0.0
	for _, policy := range m.policies {
		for _, rec := range policy.Status.Recommendations {
			if rec.Status == "pending" {
				total += rec.MonthlySavings
			}
		}
	}
	return total
}

// Helper functions

func getBudgetPeriodStart(period BudgetPeriod) time.Time {
	now := time.Now()
	switch period {
	case BudgetPeriodDaily:
		return time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	case BudgetPeriodWeekly:
		weekday := int(now.Weekday())
		return now.AddDate(0, 0, -weekday).Truncate(24 * time.Hour)
	case BudgetPeriodMonthly:
		return time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())
	case BudgetPeriodQuarterly:
		quarter := (int(now.Month()) - 1) / 3
		return time.Date(now.Year(), time.Month(quarter*3+1), 1, 0, 0, 0, 0, now.Location())
	case BudgetPeriodYearly:
		return time.Date(now.Year(), 1, 1, 0, 0, 0, 0, now.Location())
	default:
		return time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())
	}
}

func getBudgetPeriodEnd(period BudgetPeriod) time.Time {
	start := getBudgetPeriodStart(period)
	switch period {
	case BudgetPeriodDaily:
		return start.AddDate(0, 0, 1)
	case BudgetPeriodWeekly:
		return start.AddDate(0, 0, 7)
	case BudgetPeriodMonthly:
		return start.AddDate(0, 1, 0)
	case BudgetPeriodQuarterly:
		return start.AddDate(0, 3, 0)
	case BudgetPeriodYearly:
		return start.AddDate(1, 0, 0)
	default:
		return start.AddDate(0, 1, 0)
	}
}

func simulateBudgetSpend(amount float64, start, end time.Time) float64 {
	totalDays := end.Sub(start).Hours() / 24
	elapsedDays := time.Since(start).Hours() / 24
	if elapsedDays < 0 {
		elapsedDays = 0
	}
	if elapsedDays > totalDays {
		elapsedDays = totalDays
	}

	// Simulate spending slightly above linear rate
	progress := elapsedDays / totalDays
	variance := (math.Sin(progress*math.Pi) * 0.1) + 1.0 // 10% variance
	return amount * progress * variance
}
