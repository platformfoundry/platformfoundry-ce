package devex

import (
	"time"
)

// DeveloperMetrics defines developer experience metrics configuration
type DeveloperMetrics struct {
	APIVersion string                  `json:"apiVersion" yaml:"apiVersion"`
	Kind       string                  `json:"kind" yaml:"kind"`
	Metadata   MetricsMetadata         `json:"metadata" yaml:"metadata"`
	Spec       DeveloperMetricsSpec    `json:"spec" yaml:"spec"`
	Status     *DeveloperMetricsStatus `json:"status,omitempty" yaml:"status,omitempty"`
}

// MetricsMetadata contains metrics configuration identification
type MetricsMetadata struct {
	Name      string            `json:"name" yaml:"name"`
	Team      string            `json:"team,omitempty" yaml:"team,omitempty"`
	Labels    map[string]string `json:"labels,omitempty" yaml:"labels,omitempty"`
	CreatedAt time.Time         `json:"createdAt,omitempty" yaml:"createdAt,omitempty"`
}

// DeveloperMetricsSpec defines the metrics to track
type DeveloperMetricsSpec struct {
	Team      string          `json:"team" yaml:"team"`
	Metrics   []MetricTarget  `json:"metrics" yaml:"metrics"`
	Reporting ReportingConfig `json:"reporting,omitempty" yaml:"reporting,omitempty"`
}

// MetricTarget defines a metric and its target
type MetricTarget struct {
	Name   string `json:"name" yaml:"name"`
	Type   string `json:"type,omitempty" yaml:"type,omitempty"` // dora, custom, survey
	Target string `json:"target" yaml:"target"`
	Source string `json:"source,omitempty" yaml:"source,omitempty"`
}

// ReportingConfig defines reporting settings
type ReportingConfig struct {
	Dashboard    bool   `json:"dashboard,omitempty" yaml:"dashboard,omitempty"`
	WeeklyDigest bool   `json:"weeklyDigest,omitempty" yaml:"weeklyDigest,omitempty"`
	SlackChannel string `json:"slackChannel,omitempty" yaml:"slackChannel,omitempty"`
}

// DeveloperMetricsStatus tracks current metric values
type DeveloperMetricsStatus struct {
	LastUpdated    time.Time       `json:"lastUpdated" yaml:"lastUpdated"`
	MetricValues   []MetricValue   `json:"metricValues,omitempty" yaml:"metricValues,omitempty"`
	Score          *DeveloperScore `json:"score,omitempty" yaml:"score,omitempty"`
	Trends         []MetricTrend   `json:"trends,omitempty" yaml:"trends,omitempty"`
	FrictionPoints []FrictionPoint `json:"frictionPoints,omitempty" yaml:"frictionPoints,omitempty"`
}

// MetricValue represents a metric measurement
type MetricValue struct {
	Name      string    `json:"name" yaml:"name"`
	Value     float64   `json:"value" yaml:"value"`
	Unit      string    `json:"unit,omitempty" yaml:"unit,omitempty"`
	Target    string    `json:"target" yaml:"target"`
	OnTarget  bool      `json:"onTarget" yaml:"onTarget"`
	Timestamp time.Time `json:"timestamp" yaml:"timestamp"`
}

// MetricTrend represents the trend of a metric
type MetricTrend struct {
	Name      string  `json:"name" yaml:"name"`
	Direction string  `json:"direction" yaml:"direction"` // improving, stable, degrading
	Change    float64 `json:"change" yaml:"change"`       // Percentage change
	Period    string  `json:"period" yaml:"period"`       // e.g., "7d", "30d"
}

// DeveloperScore represents the overall developer experience score
type DeveloperScore struct {
	Overall    float64            `json:"overall" yaml:"overall"`
	Categories map[string]float64 `json:"categories,omitempty" yaml:"categories,omitempty"`
	Percentile int                `json:"percentile,omitempty" yaml:"percentile,omitempty"` // vs other teams
}

// FrictionPoint represents identified developer friction
type FrictionPoint struct {
	ID          string    `json:"id" yaml:"id"`
	Category    string    `json:"category" yaml:"category"` // build, deploy, test, debug, etc.
	Description string    `json:"description" yaml:"description"`
	Impact      string    `json:"impact" yaml:"impact"` // high, medium, low
	Occurrences int       `json:"occurrences" yaml:"occurrences"`
	Suggestion  string    `json:"suggestion,omitempty" yaml:"suggestion,omitempty"`
	DetectedAt  time.Time `json:"detectedAt" yaml:"detectedAt"`
}

// DORAMetrics represents DORA metrics
type DORAMetrics struct {
	DeploymentFrequency *DeploymentFrequency `json:"deploymentFrequency,omitempty" yaml:"deploymentFrequency,omitempty"`
	LeadTime            *LeadTime            `json:"leadTime,omitempty" yaml:"leadTime,omitempty"`
	ChangeFailureRate   *ChangeFailureRate   `json:"changeFailureRate,omitempty" yaml:"changeFailureRate,omitempty"`
	TimeToRestore       *TimeToRestore       `json:"timeToRestore,omitempty" yaml:"timeToRestore,omitempty"`
	Rating              string               `json:"rating" yaml:"rating"` // elite, high, medium, low
}

// DeploymentFrequency measures how often deployments occur
type DeploymentFrequency struct {
	Value            float64        `json:"value" yaml:"value"` // Deployments per day
	Period           string         `json:"period" yaml:"period"`
	TotalDeployments int            `json:"totalDeployments" yaml:"totalDeployments"`
	ByEnvironment    map[string]int `json:"byEnvironment,omitempty" yaml:"byEnvironment,omitempty"`
	Rating           string         `json:"rating" yaml:"rating"` // elite: multiple/day, high: daily-weekly
}

// LeadTime measures time from commit to production
type LeadTime struct {
	Value     time.Duration      `json:"value" yaml:"value"`
	P50       time.Duration      `json:"p50" yaml:"p50"`
	P90       time.Duration      `json:"p90" yaml:"p90"`
	P95       time.Duration      `json:"p95" yaml:"p95"`
	Rating    string             `json:"rating" yaml:"rating"` // elite: <1hr, high: <1day
	Breakdown *LeadTimeBreakdown `json:"breakdown,omitempty" yaml:"breakdown,omitempty"`
}

// LeadTimeBreakdown shows where time is spent
type LeadTimeBreakdown struct {
	CodeReview time.Duration `json:"codeReview" yaml:"codeReview"`
	Build      time.Duration `json:"build" yaml:"build"`
	Test       time.Duration `json:"test" yaml:"test"`
	Staging    time.Duration `json:"staging" yaml:"staging"`
	Approval   time.Duration `json:"approval" yaml:"approval"`
	Production time.Duration `json:"production" yaml:"production"`
}

// ChangeFailureRate measures failed deployments
type ChangeFailureRate struct {
	Value         float64           `json:"value" yaml:"value"` // Percentage
	TotalChanges  int               `json:"totalChanges" yaml:"totalChanges"`
	FailedChanges int               `json:"failedChanges" yaml:"failedChanges"`
	Rating        string            `json:"rating" yaml:"rating"` // elite: <15%, high: <30%
	TopFailures   []FailureCategory `json:"topFailures,omitempty" yaml:"topFailures,omitempty"`
}

// FailureCategory categorizes deployment failures
type FailureCategory struct {
	Category   string  `json:"category" yaml:"category"`
	Count      int     `json:"count" yaml:"count"`
	Percentage float64 `json:"percentage" yaml:"percentage"`
}

// TimeToRestore measures recovery time from incidents
type TimeToRestore struct {
	Value     time.Duration `json:"value" yaml:"value"`
	P50       time.Duration `json:"p50" yaml:"p50"`
	P90       time.Duration `json:"p90" yaml:"p90"`
	Rating    string        `json:"rating" yaml:"rating"` // elite: <1hr, high: <1day
	Incidents int           `json:"incidents" yaml:"incidents"`
}

// PlatformAdoption tracks platform usage metrics
type PlatformAdoption struct {
	SelfServiceRatio     float64        `json:"selfServiceRatio" yaml:"selfServiceRatio"`         // % self-service vs manual
	GoldenPathAdoption   float64        `json:"goldenPathAdoption" yaml:"goldenPathAdoption"`     // % using golden paths
	AutomatedDeployments float64        `json:"automatedDeployments" yaml:"automatedDeployments"` // % automated deploys
	ActiveUsers          int            `json:"activeUsers" yaml:"activeUsers"`
	TotalApplications    int            `json:"totalApplications" yaml:"totalApplications"`
	FeatureUsage         map[string]int `json:"featureUsage,omitempty" yaml:"featureUsage,omitempty"`
}

// DeveloperJourney tracks developer onboarding/productivity
type DeveloperJourney struct {
	OnboardingTime    time.Duration      `json:"onboardingTime" yaml:"onboardingTime"`
	TimeToFirstDeploy time.Duration      `json:"timeToFirstDeploy" yaml:"timeToFirstDeploy"`
	TimeToFirstPR     time.Duration      `json:"timeToFirstPR" yaml:"timeToFirstPR"`
	ProductiveTime    time.Duration      `json:"productiveTime" yaml:"productiveTime"` // Time to full productivity
	Milestones        []JourneyMilestone `json:"milestones,omitempty" yaml:"milestones,omitempty"`
}

// JourneyMilestone represents a developer journey milestone
type JourneyMilestone struct {
	Name        string        `json:"name" yaml:"name"`
	CompletedAt *time.Time    `json:"completedAt,omitempty" yaml:"completedAt,omitempty"`
	Duration    time.Duration `json:"duration,omitempty" yaml:"duration,omitempty"`
}

// Survey represents a developer survey response
type Survey struct {
	ID          string           `json:"id" yaml:"id"`
	Type        string           `json:"type" yaml:"type"` // nps, satisfaction, pulse
	Team        string           `json:"team,omitempty" yaml:"team,omitempty"`
	Responses   []SurveyResponse `json:"responses" yaml:"responses"`
	Summary     *SurveySummary   `json:"summary,omitempty" yaml:"summary,omitempty"`
	CollectedAt time.Time        `json:"collectedAt" yaml:"collectedAt"`
}

// SurveyResponse represents a single survey response
type SurveyResponse struct {
	Question string      `json:"question" yaml:"question"`
	Answer   interface{} `json:"answer" yaml:"answer"`
	Score    *int        `json:"score,omitempty" yaml:"score,omitempty"`
}

// SurveySummary summarizes survey results
type SurveySummary struct {
	NPS          *int     `json:"nps,omitempty" yaml:"nps,omitempty"`
	AverageScore float64  `json:"averageScore,omitempty" yaml:"averageScore,omitempty"`
	ResponseRate float64  `json:"responseRate,omitempty" yaml:"responseRate,omitempty"`
	TopConcerns  []string `json:"topConcerns,omitempty" yaml:"topConcerns,omitempty"`
	TopPraises   []string `json:"topPraises,omitempty" yaml:"topPraises,omitempty"`
}

// AnalyticsReport represents a developer analytics report
type AnalyticsReport struct {
	GeneratedAt     time.Time         `json:"generatedAt" yaml:"generatedAt"`
	Period          string            `json:"period" yaml:"period"`
	Team            string            `json:"team,omitempty" yaml:"team,omitempty"`
	DORA            *DORAMetrics      `json:"dora,omitempty" yaml:"dora,omitempty"`
	Adoption        *PlatformAdoption `json:"adoption,omitempty" yaml:"adoption,omitempty"`
	FrictionPoints  []FrictionPoint   `json:"frictionPoints,omitempty" yaml:"frictionPoints,omitempty"`
	Recommendations []Recommendation  `json:"recommendations,omitempty" yaml:"recommendations,omitempty"`
	Score           *DeveloperScore   `json:"score,omitempty" yaml:"score,omitempty"`
}

// Recommendation represents a platform improvement recommendation
type Recommendation struct {
	ID          string `json:"id" yaml:"id"`
	Category    string `json:"category" yaml:"category"`
	Title       string `json:"title" yaml:"title"`
	Description string `json:"description" yaml:"description"`
	Impact      string `json:"impact" yaml:"impact"` // high, medium, low
	Effort      string `json:"effort" yaml:"effort"` // high, medium, low
	Priority    int    `json:"priority" yaml:"priority"`
}
