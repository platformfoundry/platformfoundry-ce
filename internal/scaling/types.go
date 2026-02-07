// Package scaling provides predictive scaling and optimization capabilities
package scaling

import (
	"time"
)

// ScalingStrategy defines the scaling approach
type ScalingStrategy string

const (
	StrategyReactive   ScalingStrategy = "reactive"   // Scale based on current metrics
	StrategyPredictive ScalingStrategy = "predictive" // Scale based on predictions
	StrategyScheduled  ScalingStrategy = "scheduled"  // Scale based on schedule
	StrategyHybrid     ScalingStrategy = "hybrid"     // Combination of strategies
)

// PredictionModel defines the ML model type for predictions
type PredictionModel string

const (
	ModelProphet    PredictionModel = "prophet"    // Facebook Prophet
	ModelARIMA      PredictionModel = "arima"      // Auto-regressive integrated moving average
	ModelLSTM       PredictionModel = "lstm"       // Long short-term memory neural network
	ModelHoltWinter PredictionModel = "holtwinter" // Holt-Winters exponential smoothing
	ModelLinear     PredictionModel = "linear"     // Simple linear regression
)

// ScalingDirection indicates scale direction
type ScalingDirection string

const (
	DirectionUp   ScalingDirection = "up"
	DirectionDown ScalingDirection = "down"
	DirectionNone ScalingDirection = "none"
)

// ScalingPolicy defines auto-scaling behavior for a workload
type ScalingPolicy struct {
	APIVersion string               `yaml:"apiVersion" json:"apiVersion"`
	Kind       string               `yaml:"kind" json:"kind"`
	Metadata   ScalingPolicyMeta    `yaml:"metadata" json:"metadata"`
	Spec       ScalingPolicySpec    `yaml:"spec" json:"spec"`
	Status     *ScalingPolicyStatus `yaml:"status,omitempty" json:"status,omitempty"`
}

// ScalingPolicyMeta contains policy metadata
type ScalingPolicyMeta struct {
	Name        string            `yaml:"name" json:"name"`
	Namespace   string            `yaml:"namespace,omitempty" json:"namespace,omitempty"`
	Labels      map[string]string `yaml:"labels,omitempty" json:"labels,omitempty"`
	Annotations map[string]string `yaml:"annotations,omitempty" json:"annotations,omitempty"`
}

// ScalingPolicySpec defines the scaling policy specification
type ScalingPolicySpec struct {
	Target      ScalingTarget      `yaml:"target" json:"target"`
	Strategy    ScalingStrategy    `yaml:"strategy" json:"strategy"`
	Metrics     []MetricSource     `yaml:"metrics" json:"metrics"`
	Predictions *PredictionConfig  `yaml:"predictions,omitempty" json:"predictions,omitempty"`
	Constraints ScalingConstraints `yaml:"constraints" json:"constraints"`
	Behavior    *ScalingBehavior   `yaml:"behavior,omitempty" json:"behavior,omitempty"`
	CostPolicy  *CostPolicy        `yaml:"costPolicy,omitempty" json:"costPolicy,omitempty"`
	Schedule    []ScheduledScaling `yaml:"schedule,omitempty" json:"schedule,omitempty"`
}

// ScalingTarget defines what to scale
type ScalingTarget struct {
	Kind      string `yaml:"kind" json:"kind"` // Deployment, StatefulSet, etc.
	Name      string `yaml:"name" json:"name"`
	Namespace string `yaml:"namespace,omitempty" json:"namespace,omitempty"`
	Cluster   string `yaml:"cluster,omitempty" json:"cluster,omitempty"`
}

// MetricSource defines a metric to use for scaling decisions
type MetricSource struct {
	Type   string            `yaml:"type" json:"type"`     // cpu, memory, custom, external
	Name   string            `yaml:"name" json:"name"`     // Metric name
	Source string            `yaml:"source" json:"source"` // prometheus, datadog, cloudwatch
	Query  string            `yaml:"query,omitempty" json:"query,omitempty"`
	Target MetricTarget      `yaml:"target" json:"target"`
	Weight float64           `yaml:"weight,omitempty" json:"weight,omitempty"` // For multi-metric decisions
	Labels map[string]string `yaml:"labels,omitempty" json:"labels,omitempty"`
}

// MetricTarget defines the target value for a metric
type MetricTarget struct {
	Type               string  `yaml:"type" json:"type"` // Utilization, Value, AverageValue
	Value              float64 `yaml:"value,omitempty" json:"value,omitempty"`
	AverageValue       float64 `yaml:"averageValue,omitempty" json:"averageValue,omitempty"`
	AverageUtilization int     `yaml:"averageUtilization,omitempty" json:"averageUtilization,omitempty"`
}

// PredictionConfig configures predictive scaling
type PredictionConfig struct {
	Enabled          bool            `yaml:"enabled" json:"enabled"`
	Model            PredictionModel `yaml:"model" json:"model"`
	Horizon          string          `yaml:"horizon" json:"horizon"`                                       // How far ahead to predict (e.g., "1h", "6h")
	TrainingWindow   string          `yaml:"trainingWindow,omitempty" json:"trainingWindow,omitempty"`     // Historical data window
	UpdateInterval   string          `yaml:"updateInterval,omitempty" json:"updateInterval,omitempty"`     // Model update frequency
	ConfidenceLevel  float64         `yaml:"confidenceLevel,omitempty" json:"confidenceLevel,omitempty"`   // Min confidence to act
	SeasonalityMode  string          `yaml:"seasonalityMode,omitempty" json:"seasonalityMode,omitempty"`   // daily, weekly, yearly
	BufferPercentage float64         `yaml:"bufferPercentage,omitempty" json:"bufferPercentage,omitempty"` // Extra capacity buffer
}

// ScalingConstraints defines scaling limits
type ScalingConstraints struct {
	MinReplicas     int     `yaml:"minReplicas" json:"minReplicas"`
	MaxReplicas     int     `yaml:"maxReplicas" json:"maxReplicas"`
	MaxCPUPerPod    string  `yaml:"maxCPUPerPod,omitempty" json:"maxCPUPerPod,omitempty"`
	MaxMemoryPerPod string  `yaml:"maxMemoryPerPod,omitempty" json:"maxMemoryPerPod,omitempty"`
	MaxCostPerHour  float64 `yaml:"maxCostPerHour,omitempty" json:"maxCostPerHour,omitempty"`
}

// ScalingBehavior controls scaling rate
type ScalingBehavior struct {
	ScaleUp   *ScalingRules `yaml:"scaleUp,omitempty" json:"scaleUp,omitempty"`
	ScaleDown *ScalingRules `yaml:"scaleDown,omitempty" json:"scaleDown,omitempty"`
}

// ScalingRules defines scaling rate limits
type ScalingRules struct {
	StabilizationWindow string        `yaml:"stabilizationWindow,omitempty" json:"stabilizationWindow,omitempty"`
	Policies            []ScalePolicy `yaml:"policies,omitempty" json:"policies,omitempty"`
}

// ScalePolicy defines a specific scaling policy
type ScalePolicy struct {
	Type          string `yaml:"type" json:"type"` // Pods, Percent
	Value         int    `yaml:"value" json:"value"`
	PeriodSeconds int    `yaml:"periodSeconds" json:"periodSeconds"`
}

// CostPolicy defines cost-aware scaling rules
type CostPolicy struct {
	MaxHourlyCost   float64 `yaml:"maxHourlyCost,omitempty" json:"maxHourlyCost,omitempty"`
	MaxDailyCost    float64 `yaml:"maxDailyCost,omitempty" json:"maxDailyCost,omitempty"`
	MaxMonthlyCost  float64 `yaml:"maxMonthlyCost,omitempty" json:"maxMonthlyCost,omitempty"`
	PreferSpot      bool    `yaml:"preferSpot,omitempty" json:"preferSpot,omitempty"`
	SpotPercentage  int     `yaml:"spotPercentage,omitempty" json:"spotPercentage,omitempty"`
	OptimizeForCost bool    `yaml:"optimizeForCost,omitempty" json:"optimizeForCost,omitempty"`
}

// ScheduledScaling defines time-based scaling
type ScheduledScaling struct {
	Name        string `yaml:"name" json:"name"`
	Cron        string `yaml:"cron" json:"cron"`
	MinReplicas int    `yaml:"minReplicas,omitempty" json:"minReplicas,omitempty"`
	MaxReplicas int    `yaml:"maxReplicas,omitempty" json:"maxReplicas,omitempty"`
	Replicas    int    `yaml:"replicas,omitempty" json:"replicas,omitempty"`
	Duration    string `yaml:"duration,omitempty" json:"duration,omitempty"`
	Timezone    string `yaml:"timezone,omitempty" json:"timezone,omitempty"`
}

// ScalingPolicyStatus holds current policy status
type ScalingPolicyStatus struct {
	CurrentReplicas int               `json:"currentReplicas"`
	DesiredReplicas int               `json:"desiredReplicas"`
	LastScaleTime   *time.Time        `json:"lastScaleTime,omitempty"`
	LastPrediction  *PredictionResult `json:"lastPrediction,omitempty"`
	Conditions      []PolicyCondition `json:"conditions,omitempty"`
	CurrentMetrics  []MetricValue     `json:"currentMetrics,omitempty"`
	CostEstimate    *CostEstimate     `json:"costEstimate,omitempty"`
}

// PredictionResult holds prediction output
type PredictionResult struct {
	Timestamp       time.Time       `json:"timestamp"`
	Horizon         string          `json:"horizon"`
	PredictedLoad   float64         `json:"predictedLoad"`
	ConfidenceLevel float64         `json:"confidenceLevel"`
	RecommendedPods int             `json:"recommendedPods"`
	Forecast        []ForecastPoint `json:"forecast,omitempty"`
	ModelMetrics    *ModelMetrics   `json:"modelMetrics,omitempty"`
}

// ForecastPoint represents a single forecast data point
type ForecastPoint struct {
	Timestamp  time.Time `json:"timestamp"`
	Value      float64   `json:"value"`
	LowerBound float64   `json:"lowerBound"`
	UpperBound float64   `json:"upperBound"`
}

// ModelMetrics holds model performance metrics
type ModelMetrics struct {
	MAE         float64 `json:"mae"`         // Mean Absolute Error
	RMSE        float64 `json:"rmse"`        // Root Mean Square Error
	MAPE        float64 `json:"mape"`        // Mean Absolute Percentage Error
	TrainingAge string  `json:"trainingAge"` // Time since last training
}

// PolicyCondition represents a condition of the scaling policy
type PolicyCondition struct {
	Type               string    `json:"type"`
	Status             string    `json:"status"` // True, False, Unknown
	LastTransitionTime time.Time `json:"lastTransitionTime"`
	Reason             string    `json:"reason,omitempty"`
	Message            string    `json:"message,omitempty"`
}

// MetricValue holds current metric value
type MetricValue struct {
	Name         string    `json:"name"`
	CurrentValue float64   `json:"currentValue"`
	TargetValue  float64   `json:"targetValue"`
	Timestamp    time.Time `json:"timestamp"`
}

// CostEstimate holds cost estimation
type CostEstimate struct {
	CurrentHourlyCost   float64 `json:"currentHourlyCost"`
	ProjectedHourlyCost float64 `json:"projectedHourlyCost"`
	DailyEstimate       float64 `json:"dailyEstimate"`
	MonthlyEstimate     float64 `json:"monthlyEstimate"`
	SavingsOpportunity  float64 `json:"savingsOpportunity,omitempty"`
}

// ScalingEvent represents a scaling event
type ScalingEvent struct {
	ID              string           `json:"id"`
	PolicyName      string           `json:"policyName"`
	Timestamp       time.Time        `json:"timestamp"`
	Direction       ScalingDirection `json:"direction"`
	FromReplicas    int              `json:"fromReplicas"`
	ToReplicas      int              `json:"toReplicas"`
	Reason          string           `json:"reason"`
	Trigger         string           `json:"trigger"` // metric, prediction, schedule, manual
	MetricsSnapshot []MetricValue    `json:"metricsSnapshot,omitempty"`
	Success         bool             `json:"success"`
	Error           string           `json:"error,omitempty"`
	Duration        time.Duration    `json:"duration"`
}

// TrafficPattern represents learned traffic patterns
type TrafficPattern struct {
	Name        string           `json:"name"`
	DayOfWeek   []DayPattern     `json:"dayOfWeek,omitempty"`
	Seasonality *SeasonalPattern `json:"seasonality,omitempty"`
	Anomalies   []AnomalyPeriod  `json:"anomalies,omitempty"`
	LastUpdated time.Time        `json:"lastUpdated"`
}

// DayPattern represents traffic pattern for a day
type DayPattern struct {
	Day         string    `json:"day"`        // Monday, Tuesday, etc.
	HourlyLoad  []float64 `json:"hourlyLoad"` // 24 values
	PeakHours   []int     `json:"peakHours"`
	TroughHours []int     `json:"troughHours"`
}

// SeasonalPattern represents seasonal variations
type SeasonalPattern struct {
	Daily   []float64 `json:"daily,omitempty"`   // 24 values
	Weekly  []float64 `json:"weekly,omitempty"`  // 7 values
	Monthly []float64 `json:"monthly,omitempty"` // 12 values
}

// AnomalyPeriod represents a detected anomaly
type AnomalyPeriod struct {
	Start       time.Time `json:"start"`
	End         time.Time `json:"end"`
	Description string    `json:"description"`
	Multiplier  float64   `json:"multiplier"`
}
