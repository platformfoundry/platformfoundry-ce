package cost

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// Provider represents a cloud provider
type Provider string

const (
	ProviderAWS   Provider = "aws"
	ProviderGCP   Provider = "gcp"
	ProviderAzure Provider = "azure"
)

// ResourceType represents a resource type for cost estimation
type ResourceType string

const (
	ResourceTypeCompute      ResourceType = "compute"
	ResourceTypeStorage      ResourceType = "storage"
	ResourceTypeDatabase     ResourceType = "database"
	ResourceTypeNetwork      ResourceType = "network"
	ResourceTypeLoadBalancer ResourceType = "load_balancer"
	ResourceTypeKubernetes   ResourceType = "kubernetes"
)

// Resource represents a cloud resource with cost information
type Resource struct {
	ID           string                 `json:"id"`
	Name         string                 `json:"name"`
	Type         ResourceType           `json:"type"`
	Provider     Provider               `json:"provider"`
	Region       string                 `json:"region"`
	Spec         map[string]interface{} `json:"spec"`
	HourlyCost   float64                `json:"hourlyCost"`
	MonthlyCost  float64                `json:"monthlyCost"`
	YearlyCost   float64                `json:"yearlyCost"`
	Currency     string                 `json:"currency"`
	PricingModel string                 `json:"pricingModel"` // on-demand, reserved, spot
}

// Estimate represents a cost estimate for a platform
type Estimate struct {
	ID              string      `json:"id"`
	PlatformName    string      `json:"platformName"`
	Provider        Provider    `json:"provider"`
	Timestamp       time.Time   `json:"timestamp"`
	Resources       []*Resource `json:"resources"`
	TotalHourly     float64     `json:"totalHourly"`
	TotalMonthly    float64     `json:"totalMonthly"`
	TotalYearly     float64     `json:"totalYearly"`
	Currency        string      `json:"currency"`
	Recommendations []string    `json:"recommendations,omitempty"`
}

// Budget represents a cost budget
type Budget struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Amount      float64   `json:"amount"`
	Period      string    `json:"period"`    // hourly, daily, monthly, yearly
	Threshold   float64   `json:"threshold"` // percentage for alerts (e.g., 80%)
	Currency    string    `json:"currency"`
	NotifyEmail string    `json:"notifyEmail,omitempty"`
	CreatedAt   time.Time `json:"createdAt"`
}

// Estimator handles cost estimation
type Estimator struct {
	pricingDB    *PricingDB
	estimatesDir string
	budgetsDir   string
}

// Config represents cost estimator configuration
type Config struct {
	PricingFile  string `yaml:"pricingFile" json:"pricingFile"`
	EstimatesDir string `yaml:"estimatesDir" json:"estimatesDir"`
	BudgetsDir   string `yaml:"budgetsDir" json:"budgetsDir"`
	Currency     string `yaml:"currency" json:"currency"`
}

// DefaultConfig returns default cost estimator configuration
func DefaultConfig() *Config {
	return &Config{
		PricingFile:  "/etc/platformfoundry/pricing/prices.json",
		EstimatesDir: "/var/lib/platformfoundry/cost/estimates",
		BudgetsDir:   "/var/lib/platformfoundry/cost/budgets",
		Currency:     "USD",
	}
}

// NewEstimator creates a new cost estimator
func NewEstimator(config *Config) (*Estimator, error) {
	if config == nil {
		config = DefaultConfig()
	}

	// Ensure directories exist
	if err := os.MkdirAll(config.EstimatesDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create estimates directory: %w", err)
	}

	if err := os.MkdirAll(config.BudgetsDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create budgets directory: %w", err)
	}

	// Load pricing database
	pricingDB, err := LoadPricingDB(config.PricingFile)
	if err != nil {
		// Create default pricing DB if file doesn't exist
		pricingDB = NewDefaultPricingDB()
	}

	return &Estimator{
		pricingDB:    pricingDB,
		estimatesDir: config.EstimatesDir,
		budgetsDir:   config.BudgetsDir,
	}, nil
}

// EstimateResource estimates cost for a single resource
func (e *Estimator) EstimateResource(ctx context.Context, resource *Resource) (*Resource, error) {
	// Get pricing for resource
	pricing, err := e.pricingDB.GetPricing(resource.Provider, resource.Type, resource.Region)
	if err != nil {
		return nil, fmt.Errorf("failed to get pricing: %w", err)
	}

	// Calculate costs based on resource spec
	hourlyCost := e.calculateResourceCost(resource, pricing)

	resource.HourlyCost = hourlyCost
	resource.MonthlyCost = hourlyCost * 730 // Average hours per month
	resource.YearlyCost = hourlyCost * 8760 // Hours per year
	resource.Currency = pricing.Currency

	return resource, nil
}

// calculateResourceCost calculates hourly cost for a resource
func (e *Estimator) calculateResourceCost(resource *Resource, pricing *Pricing) float64 {
	cost := 0.0

	switch resource.Type {
	case ResourceTypeCompute:
		// CPU and memory-based pricing
		if cpus, ok := resource.Spec["cpus"].(float64); ok {
			cost += cpus * pricing.CPUPerHour
		}
		if memory, ok := resource.Spec["memory_gb"].(float64); ok {
			cost += memory * pricing.MemoryGBPerHour
		}

	case ResourceTypeStorage:
		// Storage-based pricing
		if storage, ok := resource.Spec["size_gb"].(float64); ok {
			cost += (storage / 730) * pricing.StorageGBPerMonth
		}

	case ResourceTypeDatabase:
		// Database pricing (similar to compute)
		if cpus, ok := resource.Spec["cpus"].(float64); ok {
			cost += cpus * pricing.CPUPerHour * 1.5 // DB premium
		}
		if memory, ok := resource.Spec["memory_gb"].(float64); ok {
			cost += memory * pricing.MemoryGBPerHour * 1.5
		}
		if storage, ok := resource.Spec["storage_gb"].(float64); ok {
			cost += (storage / 730) * pricing.StorageGBPerMonth
		}

	case ResourceTypeLoadBalancer:
		cost = pricing.LoadBalancerPerHour

	case ResourceTypeKubernetes:
		// Cluster management cost
		cost = pricing.K8sClusterPerHour
	}

	return cost
}

// EstimatePlatform creates a cost estimate for an entire platform
func (e *Estimator) EstimatePlatform(ctx context.Context, platformName string, provider Provider, resources []*Resource) (*Estimate, error) {
	estimate := &Estimate{
		ID:              generateEstimateID(),
		PlatformName:    platformName,
		Provider:        provider,
		Timestamp:       time.Now(),
		Resources:       make([]*Resource, 0, len(resources)),
		Currency:        "USD",
		Recommendations: make([]string, 0),
	}

	// Estimate each resource
	for _, resource := range resources {
		estimatedResource, err := e.EstimateResource(ctx, resource)
		if err != nil {
			return nil, fmt.Errorf("failed to estimate resource %s: %w", resource.Name, err)
		}

		estimate.Resources = append(estimate.Resources, estimatedResource)
		estimate.TotalHourly += estimatedResource.HourlyCost
		estimate.TotalMonthly += estimatedResource.MonthlyCost
		estimate.TotalYearly += estimatedResource.YearlyCost
	}

	// Generate recommendations
	estimate.Recommendations = e.generateRecommendations(estimate)

	// Save estimate
	if err := e.saveEstimate(estimate); err != nil {
		fmt.Printf("Warning: failed to save estimate: %v\n", err)
	}

	return estimate, nil
}

// generateRecommendations generates cost optimization recommendations
func (e *Estimator) generateRecommendations(estimate *Estimate) []string {
	recommendations := make([]string, 0)

	// Check if costs are high
	if estimate.TotalMonthly > 10000 {
		recommendations = append(recommendations, "Consider using reserved instances for 30-50% savings on long-term workloads")
	}

	// Check for expensive resources
	for _, resource := range estimate.Resources {
		if resource.MonthlyCost > 1000 && resource.Type == ResourceTypeCompute {
			recommendations = append(recommendations,
				fmt.Sprintf("Resource '%s' is expensive - consider right-sizing or spot instances", resource.Name))
		}

		if resource.Type == ResourceTypeStorage && resource.MonthlyCost > 500 {
			recommendations = append(recommendations,
				fmt.Sprintf("Storage '%s' - consider lifecycle policies to move to cheaper tiers", resource.Name))
		}
	}

	if len(recommendations) == 0 {
		recommendations = append(recommendations, "Cost optimization: Platform costs are within expected range")
	}

	return recommendations
}

// saveEstimate saves a cost estimate
func (e *Estimator) saveEstimate(estimate *Estimate) error {
	filename := fmt.Sprintf("%s-%s.json", estimate.PlatformName, estimate.Timestamp.Format("20060102-150405"))
	path := filepath.Join(e.estimatesDir, filename)

	data, err := json.MarshalIndent(estimate, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(path, data, 0644)
}

// LoadEstimate loads a cost estimate
func (e *Estimator) LoadEstimate(estimateID string) (*Estimate, error) {
	path := filepath.Join(e.estimatesDir, estimateID)
	if filepath.Ext(path) != ".json" {
		path += ".json"
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read estimate: %w", err)
	}

	var estimate Estimate
	if err := json.Unmarshal(data, &estimate); err != nil {
		return nil, fmt.Errorf("failed to parse estimate: %w", err)
	}

	return &estimate, nil
}

// ListEstimates lists all cost estimates
func (e *Estimator) ListEstimates() ([]*Estimate, error) {
	entries, err := os.ReadDir(e.estimatesDir)
	if err != nil {
		return nil, err
	}

	estimates := make([]*Estimate, 0)

	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".json" {
			continue
		}

		path := filepath.Join(e.estimatesDir, entry.Name())
		data, err := os.ReadFile(path)
		if err != nil {
			continue
		}

		var estimate Estimate
		if err := json.Unmarshal(data, &estimate); err != nil {
			continue
		}

		estimates = append(estimates, &estimate)
	}

	return estimates, nil
}

// CreateBudget creates a new budget
func (e *Estimator) CreateBudget(budget *Budget) error {
	if budget.ID == "" {
		budget.ID = generateBudgetID()
	}

	budget.CreatedAt = time.Now()

	filename := fmt.Sprintf("%s.json", budget.ID)
	path := filepath.Join(e.budgetsDir, filename)

	data, err := json.MarshalIndent(budget, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(path, data, 0644)
}

// ListBudgets lists all budgets
func (e *Estimator) ListBudgets() ([]*Budget, error) {
	entries, err := os.ReadDir(e.budgetsDir)
	if err != nil {
		return nil, err
	}

	budgets := make([]*Budget, 0)

	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".json" {
			continue
		}

		path := filepath.Join(e.budgetsDir, entry.Name())
		data, err := os.ReadFile(path)
		if err != nil {
			continue
		}

		var budget Budget
		if err := json.Unmarshal(data, &budget); err != nil {
			continue
		}

		budgets = append(budgets, &budget)
	}

	return budgets, nil
}

// CheckBudget checks if spending is within budget
func (e *Estimator) CheckBudget(budgetID string, actualCost float64) (bool, float64, error) {
	path := filepath.Join(e.budgetsDir, budgetID+".json")

	data, err := os.ReadFile(path)
	if err != nil {
		return false, 0, fmt.Errorf("failed to read budget: %w", err)
	}

	var budget Budget
	if err := json.Unmarshal(data, &budget); err != nil {
		return false, 0, fmt.Errorf("failed to parse budget: %w", err)
	}

	percentage := (actualCost / budget.Amount) * 100
	withinBudget := actualCost <= budget.Amount

	return withinBudget, percentage, nil
}

// Helper functions

func generateEstimateID() string {
	return fmt.Sprintf("estimate_%d", time.Now().UnixNano())
}

func generateBudgetID() string {
	return fmt.Sprintf("budget_%d", time.Now().UnixNano())
}
