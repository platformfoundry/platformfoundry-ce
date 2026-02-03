package slo

import (
	"context"
	"fmt"
	"math"
	"sync"
	"time"
)

// Engine manages SLO definitions and calculations
type Engine struct {
	metricsClient MetricsClient
	alertManager  AlertManager
	stateBackend  StateBackend
	slos          map[string]*SLO
	mu            sync.RWMutex
}

// MetricsClient interface for querying metrics
type MetricsClient interface {
	Query(ctx context.Context, query string, window time.Duration) (*QueryResult, error)
	QueryRange(ctx context.Context, query string, start, end time.Time, step time.Duration) ([]DataPoint, error)
}

// AlertManager interface for managing alerts
type AlertManager interface {
	Fire(ctx context.Context, alert Alert) error
	Resolve(ctx context.Context, alertID string) error
	List(ctx context.Context) ([]Alert, error)
}

// StateBackend interface for persistence
type StateBackend interface {
	Get(ctx context.Context, kind, id string) (interface{}, error)
	Put(ctx context.Context, kind, id string, value interface{}) error
	Delete(ctx context.Context, kind, id string) error
	List(ctx context.Context, kind string) ([]interface{}, error)
}

// QueryResult represents a metrics query result
type QueryResult struct {
	Value     float64
	Timestamp time.Time
	Labels    map[string]string
}

// DataPoint represents a time-series data point
type DataPoint struct {
	Timestamp time.Time
	Value     float64
}

// SLO represents a Service Level Objective
type SLO struct {
	ID             string           `json:"id" yaml:"id"`
	Name           string           `json:"name" yaml:"name"`
	Description    string           `json:"description,omitempty" yaml:"description,omitempty"`
	Service        string           `json:"service" yaml:"service"`
	Indicator      SLI              `json:"indicator" yaml:"indicator"`
	Objective      float64          `json:"objective" yaml:"objective"` // e.g., 99.9
	Window         time.Duration    `json:"window" yaml:"window"`       // e.g., 30 days
	BurnRateAlerts []BurnRateAlert  `json:"burnRateAlerts,omitempty" yaml:"burnRateAlerts,omitempty"`
	ErrorBudget    ErrorBudgetConfig `json:"errorBudget,omitempty" yaml:"errorBudget,omitempty"`
	Labels         map[string]string `json:"labels,omitempty" yaml:"labels,omitempty"`
	CreatedAt      time.Time        `json:"createdAt" yaml:"createdAt"`
	UpdatedAt      time.Time        `json:"updatedAt" yaml:"updatedAt"`
}

// SLI represents a Service Level Indicator
type SLI struct {
	Type        SLIType  `json:"type" yaml:"type"` // availability, latency, throughput, error_rate
	GoodQuery   string   `json:"goodQuery" yaml:"goodQuery"`     // Query for good events
	TotalQuery  string   `json:"totalQuery" yaml:"totalQuery"`   // Query for total events
	Threshold   float64  `json:"threshold,omitempty" yaml:"threshold,omitempty"` // For latency SLIs (e.g., 200ms)
	Percentile  float64  `json:"percentile,omitempty" yaml:"percentile,omitempty"` // For latency SLIs (e.g., 0.99)
}

// SLIType represents the type of SLI
type SLIType string

const (
	SLITypeAvailability SLIType = "availability"
	SLITypeLatency      SLIType = "latency"
	SLITypeThroughput   SLIType = "throughput"
	SLITypeErrorRate    SLIType = "error_rate"
)

// BurnRateAlert defines when to alert based on error budget burn rate
type BurnRateAlert struct {
	Name       string        `json:"name" yaml:"name"`
	ShortWindow time.Duration `json:"shortWindow" yaml:"shortWindow"` // e.g., 5m
	LongWindow  time.Duration `json:"longWindow" yaml:"longWindow"`   // e.g., 1h
	BurnRate   float64       `json:"burnRate" yaml:"burnRate"`       // e.g., 14.4 (will exhaust budget in 5 days)
	Severity   string        `json:"severity" yaml:"severity"`       // critical, warning, info
}

// ErrorBudgetConfig defines error budget notification settings
type ErrorBudgetConfig struct {
	NotifyAt []float64 `json:"notifyAt" yaml:"notifyAt"` // Percentages, e.g., [50, 75, 90]
	Channels []string  `json:"channels" yaml:"channels"` // Notification channels
}

// SLIResult represents the result of an SLI calculation
type SLIResult struct {
	SLO         string    `json:"slo"`
	Service     string    `json:"service"`
	Value       float64   `json:"value"`       // Current SLI value (e.g., 99.95)
	Objective   float64   `json:"objective"`   // Target (e.g., 99.9)
	InBudget    bool      `json:"inBudget"`
	BudgetSpent float64   `json:"budgetSpent"` // Percentage of error budget consumed
	BudgetLeft  float64   `json:"budgetLeft"`  // Percentage remaining
	Window      time.Duration `json:"window"`
	CalculatedAt time.Time `json:"calculatedAt"`
	GoodEvents   float64   `json:"goodEvents"`
	TotalEvents  float64   `json:"totalEvents"`
}

// Alert represents an SLO alert
type Alert struct {
	ID        string            `json:"id"`
	SLO       string            `json:"slo"`
	Service   string            `json:"service"`
	Severity  string            `json:"severity"`
	Message   string            `json:"message"`
	Value     float64           `json:"value"`
	Threshold float64           `json:"threshold"`
	Labels    map[string]string `json:"labels,omitempty"`
	FiredAt   time.Time         `json:"firedAt"`
	ResolvedAt *time.Time       `json:"resolvedAt,omitempty"`
}

// NewEngine creates a new SLO Engine
func NewEngine(metricsClient MetricsClient, alertManager AlertManager, stateBackend StateBackend) *Engine {
	return &Engine{
		metricsClient: metricsClient,
		alertManager:  alertManager,
		stateBackend:  stateBackend,
		slos:          make(map[string]*SLO),
	}
}

// RegisterSLO registers a new SLO
func (e *Engine) RegisterSLO(ctx context.Context, slo *SLO) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	if slo.ID == "" {
		slo.ID = generateSLOID(slo.Service, slo.Name)
	}

	slo.CreatedAt = time.Now()
	slo.UpdatedAt = time.Now()

	e.slos[slo.ID] = slo

	if e.stateBackend != nil {
		if err := e.stateBackend.Put(ctx, "SLO", slo.ID, slo); err != nil {
			return fmt.Errorf("failed to persist SLO: %w", err)
		}
	}

	return nil
}

// GetSLO returns an SLO by ID
func (e *Engine) GetSLO(ctx context.Context, sloID string) (*SLO, error) {
	e.mu.RLock()
	defer e.mu.RUnlock()

	slo, ok := e.slos[sloID]
	if !ok {
		return nil, fmt.Errorf("SLO not found: %s", sloID)
	}

	return slo, nil
}

// ListSLOs returns all registered SLOs
func (e *Engine) ListSLOs(ctx context.Context) []*SLO {
	e.mu.RLock()
	defer e.mu.RUnlock()

	slos := make([]*SLO, 0, len(e.slos))
	for _, slo := range e.slos {
		slos = append(slos, slo)
	}

	return slos
}

// DeleteSLO removes an SLO
func (e *Engine) DeleteSLO(ctx context.Context, sloID string) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	if _, ok := e.slos[sloID]; !ok {
		return fmt.Errorf("SLO not found: %s", sloID)
	}

	delete(e.slos, sloID)

	if e.stateBackend != nil {
		return e.stateBackend.Delete(ctx, "SLO", sloID)
	}

	return nil
}

// CalculateSLI calculates the current SLI value for an SLO
func (e *Engine) CalculateSLI(ctx context.Context, slo *SLO) (*SLIResult, error) {
	if e.metricsClient == nil {
		return nil, fmt.Errorf("metrics client not configured")
	}

	var sliValue float64
	var goodEvents, totalEvents float64

	switch slo.Indicator.Type {
	case SLITypeAvailability, SLITypeErrorRate:
		good, err := e.metricsClient.Query(ctx, slo.Indicator.GoodQuery, slo.Window)
		if err != nil {
			return nil, fmt.Errorf("failed to query good events: %w", err)
		}
		goodEvents = good.Value

		total, err := e.metricsClient.Query(ctx, slo.Indicator.TotalQuery, slo.Window)
		if err != nil {
			return nil, fmt.Errorf("failed to query total events: %w", err)
		}
		totalEvents = total.Value

		if totalEvents > 0 {
			if slo.Indicator.Type == SLITypeErrorRate {
				// Error rate: (bad/total) * 100
				sliValue = 100 - (goodEvents/totalEvents)*100
			} else {
				// Availability: (good/total) * 100
				sliValue = (goodEvents / totalEvents) * 100
			}
		}

	case SLITypeLatency:
		// For latency, we query the percentile directly
		result, err := e.metricsClient.Query(ctx, slo.Indicator.GoodQuery, slo.Window)
		if err != nil {
			return nil, fmt.Errorf("failed to query latency: %w", err)
		}
		// Calculate percentage of requests under threshold
		if slo.Indicator.Threshold > 0 {
			sliValue = result.Value // Assuming query returns percentage under threshold
		}

	case SLITypeThroughput:
		result, err := e.metricsClient.Query(ctx, slo.Indicator.GoodQuery, slo.Window)
		if err != nil {
			return nil, fmt.Errorf("failed to query throughput: %w", err)
		}
		sliValue = result.Value
	}

	// Calculate error budget consumption
	budgetSpent := e.calculateBudgetSpent(sliValue, slo.Objective)
	budgetLeft := 100 - budgetSpent

	return &SLIResult{
		SLO:          slo.ID,
		Service:      slo.Service,
		Value:        sliValue,
		Objective:    slo.Objective,
		InBudget:     sliValue >= slo.Objective,
		BudgetSpent:  budgetSpent,
		BudgetLeft:   budgetLeft,
		Window:       slo.Window,
		CalculatedAt: time.Now(),
		GoodEvents:   goodEvents,
		TotalEvents:  totalEvents,
	}, nil
}

// calculateBudgetSpent calculates the percentage of error budget consumed
func (e *Engine) calculateBudgetSpent(currentSLI, objective float64) float64 {
	// Error budget = 100 - objective (e.g., 0.1% for 99.9% SLO)
	errorBudget := 100 - objective

	if errorBudget == 0 {
		return 0
	}

	// Actual errors = 100 - current SLI
	actualErrors := 100 - currentSLI

	// Budget spent = (actual errors / allowed errors) * 100
	spent := (actualErrors / errorBudget) * 100

	// Clamp to 0-100
	if spent < 0 {
		return 0
	}
	if spent > 100 {
		return 100
	}

	return spent
}

// CheckBurnRate calculates the burn rate and generates alerts if needed
func (e *Engine) CheckBurnRate(ctx context.Context, slo *SLO) ([]Alert, error) {
	if e.metricsClient == nil {
		return nil, fmt.Errorf("metrics client not configured")
	}

	var alerts []Alert

	for _, burnRateAlert := range slo.BurnRateAlerts {
		// Calculate burn rate for short window
		shortRate, err := e.calculateBurnRateForWindow(ctx, slo, burnRateAlert.ShortWindow)
		if err != nil {
			continue
		}

		// Calculate burn rate for long window
		longRate, err := e.calculateBurnRateForWindow(ctx, slo, burnRateAlert.LongWindow)
		if err != nil {
			continue
		}

		// Multi-window approach: both must exceed threshold
		if shortRate >= burnRateAlert.BurnRate && longRate >= burnRateAlert.BurnRate {
			alert := Alert{
				ID:        fmt.Sprintf("%s-%s-%d", slo.ID, burnRateAlert.Name, time.Now().Unix()),
				SLO:       slo.ID,
				Service:   slo.Service,
				Severity:  burnRateAlert.Severity,
				Message:   fmt.Sprintf("SLO %s burn rate %.2fx exceeds threshold %.2fx (short: %.2fx, long: %.2fx)", slo.Name, longRate, burnRateAlert.BurnRate, shortRate, longRate),
				Value:     longRate,
				Threshold: burnRateAlert.BurnRate,
				Labels:    slo.Labels,
				FiredAt:   time.Now(),
			}
			alerts = append(alerts, alert)

			if e.alertManager != nil {
				e.alertManager.Fire(ctx, alert)
			}
		}
	}

	return alerts, nil
}

// calculateBurnRateForWindow calculates the error budget burn rate for a specific window
func (e *Engine) calculateBurnRateForWindow(ctx context.Context, slo *SLO, window time.Duration) (float64, error) {
	// Query good and total events for the window
	good, err := e.metricsClient.Query(ctx, slo.Indicator.GoodQuery, window)
	if err != nil {
		return 0, err
	}

	total, err := e.metricsClient.Query(ctx, slo.Indicator.TotalQuery, window)
	if err != nil {
		return 0, err
	}

	if total.Value == 0 {
		return 0, nil
	}

	// Calculate current error rate
	errorRate := 1 - (good.Value / total.Value)

	// Calculate allowed error rate
	allowedErrorRate := 1 - (slo.Objective / 100)

	if allowedErrorRate == 0 {
		return 0, nil
	}

	// Burn rate = actual error rate / allowed error rate
	burnRate := errorRate / allowedErrorRate

	return burnRate, nil
}

// GetBurnRateAlertThresholds returns recommended burn rate alert configurations
// Based on Google SRE multi-window, multi-burn-rate alerting
func GetBurnRateAlertThresholds() []BurnRateAlert {
	return []BurnRateAlert{
		{
			Name:        "page-critical",
			ShortWindow: 5 * time.Minute,
			LongWindow:  1 * time.Hour,
			BurnRate:    14.4, // Will exhaust 30-day budget in 2 days
			Severity:    "critical",
		},
		{
			Name:        "page-high",
			ShortWindow: 30 * time.Minute,
			LongWindow:  6 * time.Hour,
			BurnRate:    6.0, // Will exhaust 30-day budget in 5 days
			Severity:    "warning",
		},
		{
			Name:        "ticket-medium",
			ShortWindow: 2 * time.Hour,
			LongWindow:  24 * time.Hour,
			BurnRate:    3.0, // Will exhaust 30-day budget in 10 days
			Severity:    "warning",
		},
		{
			Name:        "ticket-low",
			ShortWindow: 6 * time.Hour,
			LongWindow:  3 * 24 * time.Hour,
			BurnRate:    1.0, // On track to exhaust budget in 30 days
			Severity:    "info",
		},
	}
}

// CalculateAllSLIs calculates SLI values for all registered SLOs
func (e *Engine) CalculateAllSLIs(ctx context.Context) ([]*SLIResult, error) {
	e.mu.RLock()
	slos := make([]*SLO, 0, len(e.slos))
	for _, slo := range e.slos {
		slos = append(slos, slo)
	}
	e.mu.RUnlock()

	results := make([]*SLIResult, 0, len(slos))
	for _, slo := range slos {
		result, err := e.CalculateSLI(ctx, slo)
		if err != nil {
			continue // Log error in production
		}
		results = append(results, result)
	}

	return results, nil
}

// GenerateSLOReport generates a report for all SLOs
func (e *Engine) GenerateSLOReport(ctx context.Context) (*SLOReport, error) {
	results, err := e.CalculateAllSLIs(ctx)
	if err != nil {
		return nil, err
	}

	report := &SLOReport{
		GeneratedAt: time.Now(),
		SLOs:        results,
		Summary:     SLOSummary{},
	}

	for _, result := range results {
		report.Summary.Total++
		if result.InBudget {
			report.Summary.InBudget++
		} else {
			report.Summary.OutOfBudget++
		}
		if result.BudgetSpent >= 50 {
			report.Summary.AtRisk++
		}
	}

	report.Summary.HealthScore = float64(report.Summary.InBudget) / float64(report.Summary.Total) * 100

	return report, nil
}

// SLOReport represents an SLO status report
type SLOReport struct {
	GeneratedAt time.Time    `json:"generatedAt"`
	SLOs        []*SLIResult `json:"slos"`
	Summary     SLOSummary   `json:"summary"`
}

// SLOSummary summarizes SLO health
type SLOSummary struct {
	Total       int     `json:"total"`
	InBudget    int     `json:"inBudget"`
	OutOfBudget int     `json:"outOfBudget"`
	AtRisk      int     `json:"atRisk"`      // >50% budget consumed
	HealthScore float64 `json:"healthScore"` // Percentage of SLOs in budget
}

// generateSLOID generates a unique SLO ID
func generateSLOID(service, name string) string {
	return fmt.Sprintf("%s-%s-%d", service, name, time.Now().UnixNano())
}

// CommonSLOTemplates returns commonly used SLO definitions
func CommonSLOTemplates() map[string]*SLO {
	return map[string]*SLO{
		"availability-99.9": {
			Name:        "availability",
			Description: "99.9% availability SLO",
			Indicator: SLI{
				Type:       SLITypeAvailability,
				GoodQuery:  `sum(increase(http_requests_total{status!~"5.."}[{{.window}}]))`,
				TotalQuery: `sum(increase(http_requests_total[{{.window}}]))`,
			},
			Objective:      99.9,
			Window:         30 * 24 * time.Hour,
			BurnRateAlerts: GetBurnRateAlertThresholds(),
			ErrorBudget: ErrorBudgetConfig{
				NotifyAt: []float64{50, 75, 90, 100},
			},
		},
		"latency-p99-200ms": {
			Name:        "latency-p99",
			Description: "99% of requests under 200ms",
			Indicator: SLI{
				Type:       SLITypeLatency,
				GoodQuery:  `sum(increase(http_request_duration_seconds_bucket{le="0.2"}[{{.window}}]))`,
				TotalQuery: `sum(increase(http_request_duration_seconds_count[{{.window}}]))`,
				Threshold:  0.2,
				Percentile: 0.99,
			},
			Objective:      99.0,
			Window:         30 * 24 * time.Hour,
			BurnRateAlerts: GetBurnRateAlertThresholds(),
		},
		"error-rate-0.1": {
			Name:        "error-rate",
			Description: "Less than 0.1% error rate",
			Indicator: SLI{
				Type:       SLITypeErrorRate,
				GoodQuery:  `sum(increase(http_requests_total{status=~"5.."}[{{.window}}]))`,
				TotalQuery: `sum(increase(http_requests_total[{{.window}}]))`,
			},
			Objective:      99.9, // 0.1% error rate = 99.9% success
			Window:         30 * 24 * time.Hour,
			BurnRateAlerts: GetBurnRateAlertThresholds(),
		},
	}
}

// CalculateErrorBudgetMinutes calculates error budget in minutes
func CalculateErrorBudgetMinutes(objective float64, window time.Duration) float64 {
	// Error budget = (100 - objective) / 100 * window_in_minutes
	windowMinutes := window.Minutes()
	errorBudgetPercent := (100 - objective) / 100
	return windowMinutes * errorBudgetPercent
}

// FormatErrorBudget formats error budget as human-readable string
func FormatErrorBudget(minutes float64) string {
	if minutes < 60 {
		return fmt.Sprintf("%.1f minutes", minutes)
	}
	if minutes < 60*24 {
		return fmt.Sprintf("%.1f hours", minutes/60)
	}
	return fmt.Sprintf("%.1f days", minutes/(60*24))
}

// CalculateRemainingBudgetTime calculates remaining time before budget exhaustion at current burn rate
func CalculateRemainingBudgetTime(budgetRemaining float64, burnRate float64) time.Duration {
	if burnRate <= 0 {
		return math.MaxInt64 * time.Nanosecond // Effectively infinite
	}

	// Time to exhaust = remaining budget / burn rate
	// If burn rate is 2x, we're consuming budget twice as fast
	hours := budgetRemaining / burnRate

	return time.Duration(hours * float64(time.Hour))
}
