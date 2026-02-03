package finops

import (
	"context"
	"fmt"
	"math"
	"sort"
	"time"
)

// CostTracker interface for tracking costs
type CostTracker interface {
	GetHistory(ctx context.Context, duration time.Duration) (map[string][]CostDataPoint, error)
	GetCurrent(ctx context.Context) (map[string]float64, error)
	GetByTeam(ctx context.Context, team string, duration time.Duration) ([]CostDataPoint, error)
}

// Notifier interface for sending notifications
type Notifier interface {
	Send(ctx context.Context, notification Notification) error
}

// CostDataPoint represents a cost data point in time
type CostDataPoint struct {
	Timestamp time.Time `json:"timestamp"`
	Cost      float64   `json:"cost"`
	Resource  string    `json:"resource,omitempty"`
	Team      string    `json:"team,omitempty"`
}

// Notification represents a notification to send
type Notification struct {
	Type       string                 `json:"type"`
	Severity   string                 `json:"severity"`
	Title      string                 `json:"title"`
	Message    string                 `json:"message"`
	Recipients []string               `json:"recipients"`
	Data       map[string]interface{} `json:"data,omitempty"`
}

// CostAnomaly represents a detected cost anomaly
type CostAnomaly struct {
	ID           string    `json:"id"`
	DetectedAt   time.Time `json:"detectedAt"`
	Resource     string    `json:"resource"`
	ResourceType string    `json:"resourceType,omitempty"`
	Team         string    `json:"team,omitempty"`
	ExpectedCost float64   `json:"expectedCost"`
	ActualCost   float64   `json:"actualCost"`
	Deviation    float64   `json:"deviation"`      // Percentage deviation
	DeviationAbs float64   `json:"deviationAbs"`   // Absolute deviation
	Severity     string    `json:"severity"`       // info, warning, critical
	Type         string    `json:"type"`           // spike, drop, trend_change
	Description  string    `json:"description"`
	Acknowledged bool      `json:"acknowledged"`
}

// Baseline represents statistical baseline for a resource
type Baseline struct {
	Mean      float64   `json:"mean"`
	StdDev    float64   `json:"stdDev"`
	Min       float64   `json:"min"`
	Max       float64   `json:"max"`
	Trend     float64   `json:"trend"`     // Daily trend (positive = increasing)
	DataPoints int      `json:"dataPoints"`
	UpdatedAt time.Time `json:"updatedAt"`
}

// AnomalyDetectorConfig contains configuration for anomaly detection
type AnomalyDetectorConfig struct {
	BaselinePeriod      time.Duration `json:"baselinePeriod"`      // Period for baseline calculation (default 30 days)
	SigmaThreshold      float64       `json:"sigmaThreshold"`      // Standard deviations for anomaly (default 2)
	MinDataPoints       int           `json:"minDataPoints"`       // Minimum data points for baseline (default 7)
	CriticalThreshold   float64       `json:"criticalThreshold"`   // Deviation % for critical (default 100)
	WarningThreshold    float64       `json:"warningThreshold"`    // Deviation % for warning (default 50)
	CheckInterval       time.Duration `json:"checkInterval"`       // How often to check (default 1 hour)
	EnableNotifications bool          `json:"enableNotifications"`
}

// AnomalyDetector detects cost anomalies
type AnomalyDetector struct {
	costTracker CostTracker
	notifier    Notifier
	config      AnomalyDetectorConfig
	baselines   map[string]*Baseline
}

// NewAnomalyDetector creates a new anomaly detector
func NewAnomalyDetector(tracker CostTracker, notifier Notifier, config AnomalyDetectorConfig) *AnomalyDetector {
	// Set defaults
	if config.BaselinePeriod == 0 {
		config.BaselinePeriod = 30 * 24 * time.Hour
	}
	if config.SigmaThreshold == 0 {
		config.SigmaThreshold = 2
	}
	if config.MinDataPoints == 0 {
		config.MinDataPoints = 7
	}
	if config.CriticalThreshold == 0 {
		config.CriticalThreshold = 100
	}
	if config.WarningThreshold == 0 {
		config.WarningThreshold = 50
	}
	if config.CheckInterval == 0 {
		config.CheckInterval = 1 * time.Hour
	}

	return &AnomalyDetector{
		costTracker: tracker,
		notifier:    notifier,
		config:      config,
		baselines:   make(map[string]*Baseline),
	}
}

// Detect detects cost anomalies
func (d *AnomalyDetector) Detect(ctx context.Context) ([]CostAnomaly, error) {
	var anomalies []CostAnomaly

	// Get historical cost data
	history, err := d.costTracker.GetHistory(ctx, d.config.BaselinePeriod)
	if err != nil {
		return nil, fmt.Errorf("failed to get cost history: %w", err)
	}

	// Calculate baselines
	d.calculateBaselines(history)

	// Get current costs
	current, err := d.costTracker.GetCurrent(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get current costs: %w", err)
	}

	// Detect anomalies
	for resource, cost := range current {
		baseline, ok := d.baselines[resource]
		if !ok || baseline.DataPoints < d.config.MinDataPoints {
			continue // Skip resources without enough baseline data
		}

		anomaly := d.checkAnomaly(resource, cost, baseline)
		if anomaly != nil {
			anomalies = append(anomalies, *anomaly)

			// Send notification if enabled
			if d.config.EnableNotifications && d.notifier != nil {
				d.sendNotification(ctx, *anomaly)
			}
		}
	}

	// Sort by severity (critical first) then by deviation
	sort.Slice(anomalies, func(i, j int) bool {
		if anomalies[i].Severity != anomalies[j].Severity {
			return severityOrder(anomalies[i].Severity) < severityOrder(anomalies[j].Severity)
		}
		return math.Abs(anomalies[i].Deviation) > math.Abs(anomalies[j].Deviation)
	})

	return anomalies, nil
}

// calculateBaselines calculates baselines from historical data
func (d *AnomalyDetector) calculateBaselines(history map[string][]CostDataPoint) {
	for resource, dataPoints := range history {
		if len(dataPoints) < d.config.MinDataPoints {
			continue
		}

		var sum, sumSq float64
		minVal := math.MaxFloat64
		maxVal := -math.MaxFloat64

		for _, dp := range dataPoints {
			sum += dp.Cost
			sumSq += dp.Cost * dp.Cost
			if dp.Cost < minVal {
				minVal = dp.Cost
			}
			if dp.Cost > maxVal {
				maxVal = dp.Cost
			}
		}

		n := float64(len(dataPoints))
		mean := sum / n
		variance := (sumSq / n) - (mean * mean)
		stdDev := math.Sqrt(variance)

		// Calculate trend (simple linear regression slope)
		trend := d.calculateTrend(dataPoints)

		d.baselines[resource] = &Baseline{
			Mean:       mean,
			StdDev:     stdDev,
			Min:        minVal,
			Max:        maxVal,
			Trend:      trend,
			DataPoints: len(dataPoints),
			UpdatedAt:  time.Now(),
		}
	}
}

// calculateTrend calculates the daily cost trend
func (d *AnomalyDetector) calculateTrend(dataPoints []CostDataPoint) float64 {
	if len(dataPoints) < 2 {
		return 0
	}

	// Sort by timestamp
	sorted := make([]CostDataPoint, len(dataPoints))
	copy(sorted, dataPoints)
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].Timestamp.Before(sorted[j].Timestamp)
	})

	// Simple linear regression
	n := float64(len(sorted))
	var sumX, sumY, sumXY, sumX2 float64

	for i, dp := range sorted {
		x := float64(i)
		y := dp.Cost
		sumX += x
		sumY += y
		sumXY += x * y
		sumX2 += x * x
	}

	// Slope = (n*sumXY - sumX*sumY) / (n*sumX2 - sumX*sumX)
	denominator := n*sumX2 - sumX*sumX
	if denominator == 0 {
		return 0
	}

	return (n*sumXY - sumX*sumY) / denominator
}

// checkAnomaly checks if a cost is anomalous
func (d *AnomalyDetector) checkAnomaly(resource string, cost float64, baseline *Baseline) *CostAnomaly {
	if baseline.Mean == 0 {
		return nil
	}

	// Calculate deviation
	deviation := (cost - baseline.Mean) / baseline.Mean * 100
	deviationAbs := cost - baseline.Mean

	// Check if deviation exceeds threshold (using standard deviations)
	threshold := baseline.StdDev * d.config.SigmaThreshold
	if baseline.StdDev == 0 {
		threshold = baseline.Mean * 0.2 // Use 20% if no variance
	}

	if math.Abs(deviationAbs) < threshold {
		return nil // Within normal range
	}

	// Determine severity
	severity := "info"
	absDeviation := math.Abs(deviation)
	if absDeviation >= d.config.CriticalThreshold {
		severity = "critical"
	} else if absDeviation >= d.config.WarningThreshold {
		severity = "warning"
	}

	// Determine anomaly type
	anomalyType := "spike"
	if deviation < 0 {
		anomalyType = "drop"
	}

	// Check for trend change
	if baseline.Trend != 0 {
		expectedWithTrend := baseline.Mean + baseline.Trend
		if math.Abs(cost-expectedWithTrend) > threshold*2 {
			anomalyType = "trend_change"
		}
	}

	return &CostAnomaly{
		ID:           fmt.Sprintf("anomaly-%s-%d", resource, time.Now().Unix()),
		DetectedAt:   time.Now(),
		Resource:     resource,
		ExpectedCost: baseline.Mean,
		ActualCost:   cost,
		Deviation:    deviation,
		DeviationAbs: deviationAbs,
		Severity:     severity,
		Type:         anomalyType,
		Description:  d.generateDescription(resource, cost, baseline, deviation, anomalyType),
	}
}

// generateDescription generates a human-readable description
func (d *AnomalyDetector) generateDescription(resource string, cost float64, baseline *Baseline, deviation float64, anomalyType string) string {
	direction := "increased"
	if deviation < 0 {
		direction = "decreased"
	}

	return fmt.Sprintf(
		"Cost for %s has %s by %.1f%% (from $%.2f to $%.2f). This is a %s anomaly based on %d days of historical data.",
		resource,
		direction,
		math.Abs(deviation),
		baseline.Mean,
		cost,
		anomalyType,
		baseline.DataPoints,
	)
}

// sendNotification sends a notification for an anomaly
func (d *AnomalyDetector) sendNotification(ctx context.Context, anomaly CostAnomaly) {
	notification := Notification{
		Type:     "cost_anomaly",
		Severity: anomaly.Severity,
		Title:    fmt.Sprintf("Cost Anomaly Detected: %s", anomaly.Resource),
		Message:  anomaly.Description,
		Data: map[string]interface{}{
			"anomaly": anomaly,
		},
	}

	d.notifier.Send(ctx, notification)
}

// GetBaseline returns the baseline for a resource
func (d *AnomalyDetector) GetBaseline(resource string) *Baseline {
	return d.baselines[resource]
}

// GetAllBaselines returns all baselines
func (d *AnomalyDetector) GetAllBaselines() map[string]*Baseline {
	return d.baselines
}

// AcknowledgeAnomaly acknowledges an anomaly (marks it as reviewed)
func (d *AnomalyDetector) AcknowledgeAnomaly(anomalyID string) error {
	// In production, this would update the anomaly in persistent storage
	return nil
}

// DetectByTeam detects anomalies for a specific team
func (d *AnomalyDetector) DetectByTeam(ctx context.Context, team string) ([]CostAnomaly, error) {
	dataPoints, err := d.costTracker.GetByTeam(ctx, team, d.config.BaselinePeriod)
	if err != nil {
		return nil, err
	}

	if len(dataPoints) < d.config.MinDataPoints {
		return nil, nil // Not enough data
	}

	// Calculate team baseline
	var sum, sumSq float64
	for _, dp := range dataPoints {
		sum += dp.Cost
		sumSq += dp.Cost * dp.Cost
	}
	n := float64(len(dataPoints))
	mean := sum / n
	variance := (sumSq / n) - (mean * mean)
	stdDev := math.Sqrt(variance)

	baseline := &Baseline{
		Mean:       mean,
		StdDev:     stdDev,
		DataPoints: len(dataPoints),
		UpdatedAt:  time.Now(),
	}

	// Get current team cost
	current, err := d.costTracker.GetCurrent(ctx)
	if err != nil {
		return nil, err
	}

	var teamCost float64
	for resource, cost := range current {
		// In production, filter by team tag
		_ = resource
		teamCost += cost
	}

	var anomalies []CostAnomaly
	if anomaly := d.checkAnomaly(team, teamCost, baseline); anomaly != nil {
		anomaly.Team = team
		anomalies = append(anomalies, *anomaly)
	}

	return anomalies, nil
}

// severityOrder returns the order for sorting severities
func severityOrder(severity string) int {
	switch severity {
	case "critical":
		return 0
	case "warning":
		return 1
	case "info":
		return 2
	default:
		return 3
	}
}

// ForecastCost forecasts future cost based on trend
func (d *AnomalyDetector) ForecastCost(resource string, days int) (float64, error) {
	baseline, ok := d.baselines[resource]
	if !ok {
		return 0, fmt.Errorf("no baseline for resource: %s", resource)
	}

	forecast := baseline.Mean + (baseline.Trend * float64(days))
	if forecast < 0 {
		forecast = 0
	}

	return forecast, nil
}

// GetAnomalySummary returns a summary of anomalies
type AnomalySummary struct {
	Total     int     `json:"total"`
	Critical  int     `json:"critical"`
	Warning   int     `json:"warning"`
	Info      int     `json:"info"`
	TotalCostImpact float64 `json:"totalCostImpact"`
}

// SummarizeAnomalies summarizes a list of anomalies
func SummarizeAnomalies(anomalies []CostAnomaly) AnomalySummary {
	summary := AnomalySummary{Total: len(anomalies)}

	for _, a := range anomalies {
		switch a.Severity {
		case "critical":
			summary.Critical++
		case "warning":
			summary.Warning++
		case "info":
			summary.Info++
		}
		summary.TotalCostImpact += a.DeviationAbs
	}

	return summary
}
