package finops

import (
	"time"
)

// FinOpsPolicy defines cost management policies
type FinOpsPolicy struct {
	APIVersion string             `json:"apiVersion" yaml:"apiVersion"`
	Kind       string             `json:"kind" yaml:"kind"`
	Metadata   FinOpsMetadata     `json:"metadata" yaml:"metadata"`
	Spec       FinOpsPolicySpec   `json:"spec" yaml:"spec"`
	Status     *FinOpsPolicyStatus `json:"status,omitempty" yaml:"status,omitempty"`
}

// FinOpsMetadata contains policy identification
type FinOpsMetadata struct {
	Name        string            `json:"name" yaml:"name"`
	Description string            `json:"description,omitempty" yaml:"description,omitempty"`
	Labels      map[string]string `json:"labels,omitempty" yaml:"labels,omitempty"`
	CreatedAt   time.Time         `json:"createdAt,omitempty" yaml:"createdAt,omitempty"`
	UpdatedAt   time.Time         `json:"updatedAt,omitempty" yaml:"updatedAt,omitempty"`
}

// FinOpsPolicySpec defines policy configuration
type FinOpsPolicySpec struct {
	Budgets      []Budget         `json:"budgets,omitempty" yaml:"budgets,omitempty"`
	Optimization OptimizationSpec `json:"optimization,omitempty" yaml:"optimization,omitempty"`
	Showback     ShowbackSpec     `json:"showback,omitempty" yaml:"showback,omitempty"`
	Anomaly      AnomalySpec      `json:"anomaly,omitempty" yaml:"anomaly,omitempty"`
}

// Budget defines a budget configuration
type Budget struct {
	Name          string        `json:"name" yaml:"name"`
	Scope         BudgetScope   `json:"scope" yaml:"scope"`
	ScopeValue    string        `json:"scopeValue,omitempty" yaml:"scopeValue,omitempty"` // team name, app name, etc.
	Amount        float64       `json:"amount" yaml:"amount"`
	Period        BudgetPeriod  `json:"period" yaml:"period"`
	Currency      string        `json:"currency,omitempty" yaml:"currency,omitempty"`
	AlertThresholds []float64   `json:"alertThresholds,omitempty" yaml:"alertThresholds,omitempty"` // e.g., [50, 75, 90, 100]
	NotifyChannels []string     `json:"notifyChannels,omitempty" yaml:"notifyChannels,omitempty"`
	EnforceLimit  bool          `json:"enforceLimit,omitempty" yaml:"enforceLimit,omitempty"`
}

// BudgetScope defines what the budget applies to
type BudgetScope string

const (
	BudgetScopeOrganization BudgetScope = "organization"
	BudgetScopeTeam         BudgetScope = "team"
	BudgetScopeApplication  BudgetScope = "application"
	BudgetScopeEnvironment  BudgetScope = "environment"
	BudgetScopeProject      BudgetScope = "project"
)

// BudgetPeriod defines the budget period
type BudgetPeriod string

const (
	BudgetPeriodDaily    BudgetPeriod = "daily"
	BudgetPeriodWeekly   BudgetPeriod = "weekly"
	BudgetPeriodMonthly  BudgetPeriod = "monthly"
	BudgetPeriodQuarterly BudgetPeriod = "quarterly"
	BudgetPeriodYearly   BudgetPeriod = "yearly"
)

// OptimizationSpec defines cost optimization settings
type OptimizationSpec struct {
	RightSizing      RightSizingSpec      `json:"rightSizing,omitempty" yaml:"rightSizing,omitempty"`
	Reservations     ReservationsSpec     `json:"reservations,omitempty" yaml:"reservations,omitempty"`
	SpotInstances    SpotInstancesSpec    `json:"spotInstances,omitempty" yaml:"spotInstances,omitempty"`
	UnusedResources  UnusedResourcesSpec  `json:"unusedResources,omitempty" yaml:"unusedResources,omitempty"`
	ScheduledScaling ScheduledScalingSpec `json:"scheduledScaling,omitempty" yaml:"scheduledScaling,omitempty"`
}

// RightSizingSpec defines right-sizing configuration
type RightSizingSpec struct {
	Enabled      bool     `json:"enabled" yaml:"enabled"`
	MinSavings   float64  `json:"minSavings,omitempty" yaml:"minSavings,omitempty"` // Minimum monthly savings to recommend
	AutoApply    []string `json:"autoApply,omitempty" yaml:"autoApply,omitempty"` // Environments to auto-apply
	ExcludeApps  []string `json:"excludeApps,omitempty" yaml:"excludeApps,omitempty"`
	CPUThreshold float64  `json:"cpuThreshold,omitempty" yaml:"cpuThreshold,omitempty"` // e.g., 0.5 = 50% avg utilization
	MemThreshold float64  `json:"memThreshold,omitempty" yaml:"memThreshold,omitempty"`
}

// ReservationsSpec defines reservation management
type ReservationsSpec struct {
	Analyze      bool   `json:"analyze" yaml:"analyze"`
	Recommend    bool   `json:"recommend" yaml:"recommend"`
	AutoPurchase bool   `json:"autoPurchase,omitempty" yaml:"autoPurchase,omitempty"`
	MinTerm      string `json:"minTerm,omitempty" yaml:"minTerm,omitempty"` // 1yr, 3yr
	MaxCommitment float64 `json:"maxCommitment,omitempty" yaml:"maxCommitment,omitempty"`
}

// SpotInstancesSpec defines spot instance usage
type SpotInstancesSpec struct {
	Enabled         bool     `json:"enabled" yaml:"enabled"`
	Workloads       []string `json:"workloads,omitempty" yaml:"workloads,omitempty"` // batch, dev, ci, etc.
	FallbackOnDemand bool    `json:"fallbackOnDemand,omitempty" yaml:"fallbackOnDemand,omitempty"`
	MaxSpotPercent  int      `json:"maxSpotPercent,omitempty" yaml:"maxSpotPercent,omitempty"`
}

// UnusedResourcesSpec defines unused resource detection
type UnusedResourcesSpec struct {
	DetectAfter  string   `json:"detectAfter" yaml:"detectAfter"` // Duration like "7d"
	NotifyOwner  bool     `json:"notifyOwner" yaml:"notifyOwner"`
	AutoDelete   []string `json:"autoDelete,omitempty" yaml:"autoDelete,omitempty"` // Environments to auto-delete
	ExcludeTypes []string `json:"excludeTypes,omitempty" yaml:"excludeTypes,omitempty"`
}

// ScheduledScalingSpec defines scheduled scaling for cost savings
type ScheduledScalingSpec struct {
	Enabled   bool              `json:"enabled" yaml:"enabled"`
	Schedules []ScalingSchedule `json:"schedules,omitempty" yaml:"schedules,omitempty"`
}

// ScalingSchedule defines a scaling schedule
type ScalingSchedule struct {
	Name        string   `json:"name" yaml:"name"`
	Environments []string `json:"environments" yaml:"environments"`
	Cron        string   `json:"cron" yaml:"cron"`
	ScaleDown   bool     `json:"scaleDown,omitempty" yaml:"scaleDown,omitempty"`
	MinReplicas int      `json:"minReplicas,omitempty" yaml:"minReplicas,omitempty"`
	Timezone    string   `json:"timezone,omitempty" yaml:"timezone,omitempty"`
}

// ShowbackSpec defines cost allocation settings
type ShowbackSpec struct {
	Enabled     bool          `json:"enabled" yaml:"enabled"`
	Granularity []string      `json:"granularity,omitempty" yaml:"granularity,omitempty"` // team, application, environment
	Reports     ReportSpec    `json:"reports,omitempty" yaml:"reports,omitempty"`
	Allocation  AllocationSpec `json:"allocation,omitempty" yaml:"allocation,omitempty"`
}

// ReportSpec defines reporting configuration
type ReportSpec struct {
	Schedule   string   `json:"schedule,omitempty" yaml:"schedule,omitempty"` // daily, weekly, monthly
	Recipients []string `json:"recipients,omitempty" yaml:"recipients,omitempty"`
	Format     []string `json:"format,omitempty" yaml:"format,omitempty"` // pdf, csv, json
}

// AllocationSpec defines how to allocate shared costs
type AllocationSpec struct {
	SharedCostMethod string            `json:"sharedCostMethod,omitempty" yaml:"sharedCostMethod,omitempty"` // proportional, even, custom
	CustomRules      map[string]float64 `json:"customRules,omitempty" yaml:"customRules,omitempty"`
}

// AnomalySpec defines cost anomaly detection
type AnomalySpec struct {
	Enabled        bool    `json:"enabled" yaml:"enabled"`
	Threshold      float64 `json:"threshold,omitempty" yaml:"threshold,omitempty"` // Percentage deviation
	LookbackDays   int     `json:"lookbackDays,omitempty" yaml:"lookbackDays,omitempty"`
	NotifyChannels []string `json:"notifyChannels,omitempty" yaml:"notifyChannels,omitempty"`
}

// FinOpsPolicyStatus tracks policy state
type FinOpsPolicyStatus struct {
	LastUpdated     time.Time        `json:"lastUpdated" yaml:"lastUpdated"`
	BudgetStatus    []BudgetStatus   `json:"budgetStatus,omitempty" yaml:"budgetStatus,omitempty"`
	Recommendations []Recommendation `json:"recommendations,omitempty" yaml:"recommendations,omitempty"`
	Anomalies       []CostAnomaly    `json:"anomalies,omitempty" yaml:"anomalies,omitempty"`
}

// BudgetStatus tracks budget consumption
type BudgetStatus struct {
	Name          string    `json:"name" yaml:"name"`
	Scope         string    `json:"scope" yaml:"scope"`
	Amount        float64   `json:"amount" yaml:"amount"`
	Spent         float64   `json:"spent" yaml:"spent"`
	SpentPercent  float64   `json:"spentPercent" yaml:"spentPercent"`
	Forecast      float64   `json:"forecast" yaml:"forecast"`
	Status        string    `json:"status" yaml:"status"` // on_track, at_risk, over_budget
	PeriodStart   time.Time `json:"periodStart" yaml:"periodStart"`
	PeriodEnd     time.Time `json:"periodEnd" yaml:"periodEnd"`
}

// Recommendation represents a cost optimization recommendation
type Recommendation struct {
	ID              string    `json:"id" yaml:"id"`
	Type            string    `json:"type" yaml:"type"` // right_sizing, unused_resource, reservation, spot
	Resource        string    `json:"resource" yaml:"resource"`
	ResourceType    string    `json:"resourceType" yaml:"resourceType"`
	CurrentCost     float64   `json:"currentCost" yaml:"currentCost"`
	RecommendedCost float64   `json:"recommendedCost" yaml:"recommendedCost"`
	MonthlySavings  float64   `json:"monthlySavings" yaml:"monthlySavings"`
	Description     string    `json:"description" yaml:"description"`
	Action          string    `json:"action" yaml:"action"`
	Confidence      float64   `json:"confidence" yaml:"confidence"`
	DetectedAt      time.Time `json:"detectedAt" yaml:"detectedAt"`
	Status          string    `json:"status" yaml:"status"` // pending, applied, dismissed
}

// CostReport represents a cost report
type CostReport struct {
	GeneratedAt    time.Time         `json:"generatedAt" yaml:"generatedAt"`
	PeriodStart    time.Time         `json:"periodStart" yaml:"periodStart"`
	PeriodEnd      time.Time         `json:"periodEnd" yaml:"periodEnd"`
	TotalCost      float64           `json:"totalCost" yaml:"totalCost"`
	PreviousCost   float64           `json:"previousCost" yaml:"previousCost"`
	CostChange     float64           `json:"costChange" yaml:"costChange"`
	ChangePercent  float64           `json:"changePercent" yaml:"changePercent"`
	Currency       string            `json:"currency" yaml:"currency"`
	ByTeam         map[string]float64 `json:"byTeam,omitempty" yaml:"byTeam,omitempty"`
	ByApplication  map[string]float64 `json:"byApplication,omitempty" yaml:"byApplication,omitempty"`
	ByEnvironment  map[string]float64 `json:"byEnvironment,omitempty" yaml:"byEnvironment,omitempty"`
	ByService      map[string]float64 `json:"byService,omitempty" yaml:"byService,omitempty"`
	TopSpenders    []CostItem        `json:"topSpenders,omitempty" yaml:"topSpenders,omitempty"`
	Recommendations []Recommendation `json:"recommendations,omitempty" yaml:"recommendations,omitempty"`
	Forecast       *CostForecast     `json:"forecast,omitempty" yaml:"forecast,omitempty"`
}

// CostItem represents a cost line item
type CostItem struct {
	Name        string  `json:"name" yaml:"name"`
	Type        string  `json:"type" yaml:"type"`
	Cost        float64 `json:"cost" yaml:"cost"`
	Change      float64 `json:"change,omitempty" yaml:"change,omitempty"`
	Owner       string  `json:"owner,omitempty" yaml:"owner,omitempty"`
}

// CostForecast predicts future costs
type CostForecast struct {
	NextMonth     float64 `json:"nextMonth" yaml:"nextMonth"`
	NextQuarter   float64 `json:"nextQuarter" yaml:"nextQuarter"`
	EndOfYear     float64 `json:"endOfYear" yaml:"endOfYear"`
	Confidence    float64 `json:"confidence" yaml:"confidence"`
	Trend         string  `json:"trend" yaml:"trend"` // increasing, stable, decreasing
}

// ResourceCost tracks cost for a specific resource
type ResourceCost struct {
	ResourceID   string    `json:"resourceId" yaml:"resourceId"`
	ResourceType string    `json:"resourceType" yaml:"resourceType"`
	Name         string    `json:"name" yaml:"name"`
	Team         string    `json:"team,omitempty" yaml:"team,omitempty"`
	Application  string    `json:"application,omitempty" yaml:"application,omitempty"`
	Environment  string    `json:"environment,omitempty" yaml:"environment,omitempty"`
	HourlyCost   float64   `json:"hourlyCost" yaml:"hourlyCost"`
	DailyCost    float64   `json:"dailyCost" yaml:"dailyCost"`
	MonthlyCost  float64   `json:"monthlyCost" yaml:"monthlyCost"`
	Currency     string    `json:"currency" yaml:"currency"`
	LastUpdated  time.Time `json:"lastUpdated" yaml:"lastUpdated"`
	Tags         map[string]string `json:"tags,omitempty" yaml:"tags,omitempty"`
}
