package slo

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// BudgetTracker tracks error budget consumption and sends notifications
type BudgetTracker struct {
	engine       *Engine
	notifier     BudgetNotifier
	stateBackend StateBackend
	budgets      map[string]*ErrorBudget
	mu           sync.RWMutex
}

// BudgetNotifier interface for sending budget notifications
type BudgetNotifier interface {
	SendBudgetAlert(ctx context.Context, budget *ErrorBudget, threshold float64) error
	SendBudgetExhausted(ctx context.Context, budget *ErrorBudget) error
	SendBudgetRecovered(ctx context.Context, budget *ErrorBudget) error
}

// ErrorBudget represents the current state of an error budget
type ErrorBudget struct {
	SLOID         string        `json:"sloId"`
	SLOName       string        `json:"sloName"`
	Service       string        `json:"service"`
	TotalBudget   float64       `json:"totalBudget"`   // In minutes or events
	Consumed      float64       `json:"consumed"`      // Consumed budget
	Remaining     float64       `json:"remaining"`     // Remaining budget
	ConsumedPct   float64       `json:"consumedPct"`   // Percentage consumed
	RemainingPct  float64       `json:"remainingPct"`  // Percentage remaining
	BurnRate      float64       `json:"burnRate"`      // Current burn rate multiplier
	TimeToExhaust time.Duration `json:"timeToExhaust"` // At current burn rate
	ResetDate     time.Time     `json:"resetDate"`     // When budget resets
	Status        BudgetStatus  `json:"status"`
	History       []BudgetEvent `json:"history,omitempty"`
	UpdatedAt     time.Time     `json:"updatedAt"`
}

// BudgetStatus represents the status of an error budget
type BudgetStatus string

const (
	BudgetStatusHealthy   BudgetStatus = "healthy"   // < 50% consumed
	BudgetStatusWarning   BudgetStatus = "warning"   // 50-75% consumed
	BudgetStatusCritical  BudgetStatus = "critical"  // 75-90% consumed
	BudgetStatusExhausted BudgetStatus = "exhausted" // > 90% consumed
)

// BudgetEvent represents a significant change in error budget
type BudgetEvent struct {
	Timestamp   time.Time `json:"timestamp"`
	Type        string    `json:"type"` // threshold_crossed, exhausted, recovered, reset
	Description string    `json:"description"`
	ConsumedPct float64   `json:"consumedPct"`
	BurnRate    float64   `json:"burnRate"`
}

// NewBudgetTracker creates a new BudgetTracker
func NewBudgetTracker(engine *Engine, notifier BudgetNotifier, stateBackend StateBackend) *BudgetTracker {
	return &BudgetTracker{
		engine:       engine,
		notifier:     notifier,
		stateBackend: stateBackend,
		budgets:      make(map[string]*ErrorBudget),
	}
}

// Track calculates and tracks the error budget for an SLO
func (t *BudgetTracker) Track(ctx context.Context, slo *SLO) (*ErrorBudget, error) {
	// Calculate current SLI
	result, err := t.engine.CalculateSLI(ctx, slo)
	if err != nil {
		return nil, fmt.Errorf("failed to calculate SLI: %w", err)
	}

	// Calculate total budget (in minutes of allowed downtime)
	windowMinutes := slo.Window.Minutes()
	allowedDowntime := windowMinutes * (1 - slo.Objective/100)

	// Calculate consumed budget
	actualDowntime := windowMinutes * (1 - result.Value/100)

	budget := &ErrorBudget{
		SLOID:        slo.ID,
		SLOName:      slo.Name,
		Service:      slo.Service,
		TotalBudget:  allowedDowntime,
		Consumed:     actualDowntime,
		Remaining:    allowedDowntime - actualDowntime,
		ConsumedPct:  result.BudgetSpent,
		RemainingPct: result.BudgetLeft,
		ResetDate:    t.calculateResetDate(slo),
		UpdatedAt:    time.Now(),
	}

	// Calculate burn rate
	burnRate, err := t.engine.calculateBurnRateForWindow(ctx, slo, 1*time.Hour)
	if err == nil {
		budget.BurnRate = burnRate
		if burnRate > 0 {
			budget.TimeToExhaust = time.Duration(budget.RemainingPct/burnRate) * time.Hour
		}
	}

	// Determine status
	budget.Status = t.determineBudgetStatus(budget.ConsumedPct)

	// Check if we need to send notifications
	t.checkNotifications(ctx, slo, budget)

	// Store budget state
	t.mu.Lock()
	previousBudget := t.budgets[slo.ID]
	t.budgets[slo.ID] = budget
	t.mu.Unlock()

	// Record budget event if status changed
	if previousBudget != nil && previousBudget.Status != budget.Status {
		event := BudgetEvent{
			Timestamp:   time.Now(),
			Type:        "status_changed",
			Description: fmt.Sprintf("Status changed from %s to %s", previousBudget.Status, budget.Status),
			ConsumedPct: budget.ConsumedPct,
			BurnRate:    budget.BurnRate,
		}
		budget.History = append(budget.History, event)

		// Notify on status change
		if budget.Status == BudgetStatusExhausted && t.notifier != nil {
			t.notifier.SendBudgetExhausted(ctx, budget)
		}
		if budget.Status == BudgetStatusHealthy && previousBudget.Status != BudgetStatusHealthy && t.notifier != nil {
			t.notifier.SendBudgetRecovered(ctx, budget)
		}
	}

	if t.stateBackend != nil {
		t.stateBackend.Put(ctx, "ErrorBudget", slo.ID, budget)
	}

	return budget, nil
}

// calculateResetDate calculates when the error budget resets
func (t *BudgetTracker) calculateResetDate(slo *SLO) time.Time {
	// Assuming rolling window, the reset is always window duration from now
	return time.Now().Add(slo.Window)
}

// determineBudgetStatus determines the status based on consumption percentage
func (t *BudgetTracker) determineBudgetStatus(consumedPct float64) BudgetStatus {
	switch {
	case consumedPct >= 90:
		return BudgetStatusExhausted
	case consumedPct >= 75:
		return BudgetStatusCritical
	case consumedPct >= 50:
		return BudgetStatusWarning
	default:
		return BudgetStatusHealthy
	}
}

// checkNotifications checks if we need to send threshold notifications
func (t *BudgetTracker) checkNotifications(ctx context.Context, slo *SLO, budget *ErrorBudget) {
	if t.notifier == nil {
		return
	}

	t.mu.RLock()
	previousBudget := t.budgets[slo.ID]
	t.mu.RUnlock()

	// Check each notification threshold
	for _, threshold := range slo.ErrorBudget.NotifyAt {
		if budget.ConsumedPct >= threshold {
			// Only notify if we just crossed this threshold
			if previousBudget == nil || previousBudget.ConsumedPct < threshold {
				t.notifier.SendBudgetAlert(ctx, budget, threshold)
			}
		}
	}
}

// GetBudget returns the current error budget for an SLO
func (t *BudgetTracker) GetBudget(ctx context.Context, sloID string) (*ErrorBudget, error) {
	t.mu.RLock()
	budget, ok := t.budgets[sloID]
	t.mu.RUnlock()

	if !ok {
		// Try to load from state backend
		if t.stateBackend != nil {
			data, err := t.stateBackend.Get(ctx, "ErrorBudget", sloID)
			if err == nil && data != nil {
				if b, ok := data.(*ErrorBudget); ok {
					return b, nil
				}
			}
		}
		return nil, fmt.Errorf("error budget not found for SLO: %s", sloID)
	}

	return budget, nil
}

// ListBudgets returns all tracked error budgets
func (t *BudgetTracker) ListBudgets(ctx context.Context) []*ErrorBudget {
	t.mu.RLock()
	defer t.mu.RUnlock()

	budgets := make([]*ErrorBudget, 0, len(t.budgets))
	for _, budget := range t.budgets {
		budgets = append(budgets, budget)
	}

	return budgets
}

// TrackAll tracks error budgets for all registered SLOs
func (t *BudgetTracker) TrackAll(ctx context.Context) ([]*ErrorBudget, error) {
	slos := t.engine.ListSLOs(ctx)
	budgets := make([]*ErrorBudget, 0, len(slos))

	for _, slo := range slos {
		budget, err := t.Track(ctx, slo)
		if err != nil {
			continue // Log in production
		}
		budgets = append(budgets, budget)
	}

	return budgets, nil
}

// GetBudgetSummary returns a summary of all error budgets
func (t *BudgetTracker) GetBudgetSummary(ctx context.Context) *BudgetSummary {
	budgets := t.ListBudgets(ctx)

	summary := &BudgetSummary{
		Total:       len(budgets),
		GeneratedAt: time.Now(),
	}

	for _, budget := range budgets {
		switch budget.Status {
		case BudgetStatusHealthy:
			summary.Healthy++
		case BudgetStatusWarning:
			summary.Warning++
		case BudgetStatusCritical:
			summary.Critical++
		case BudgetStatusExhausted:
			summary.Exhausted++
		}
	}

	if summary.Total > 0 {
		summary.HealthScore = float64(summary.Healthy) / float64(summary.Total) * 100
	}

	return summary
}

// BudgetSummary summarizes error budget health across all SLOs
type BudgetSummary struct {
	Total       int       `json:"total"`
	Healthy     int       `json:"healthy"`
	Warning     int       `json:"warning"`
	Critical    int       `json:"critical"`
	Exhausted   int       `json:"exhausted"`
	HealthScore float64   `json:"healthScore"`
	GeneratedAt time.Time `json:"generatedAt"`
}

// StartPeriodicTracking starts a goroutine that periodically tracks all error budgets
func (t *BudgetTracker) StartPeriodicTracking(ctx context.Context, interval time.Duration) {
	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				t.TrackAll(ctx)
			}
		}
	}()
}

// BudgetForecast represents a forecast of error budget consumption
type BudgetForecast struct {
	SLOID           string    `json:"sloId"`
	CurrentBurnRate float64   `json:"currentBurnRate"`
	ProjectedDate   time.Time `json:"projectedDate"` // When budget will be exhausted
	DaysRemaining   float64   `json:"daysRemaining"`
	Recommendation  string    `json:"recommendation"`
}

// ForecastBudget forecasts when the error budget will be exhausted
func (t *BudgetTracker) ForecastBudget(ctx context.Context, sloID string) (*BudgetForecast, error) {
	budget, err := t.GetBudget(ctx, sloID)
	if err != nil {
		return nil, err
	}

	forecast := &BudgetForecast{
		SLOID:           sloID,
		CurrentBurnRate: budget.BurnRate,
	}

	if budget.BurnRate <= 0 {
		forecast.Recommendation = "Budget consumption rate is healthy"
		return forecast, nil
	}

	// Calculate when budget will be exhausted
	hoursRemaining := budget.RemainingPct / budget.BurnRate
	forecast.DaysRemaining = hoursRemaining / 24
	forecast.ProjectedDate = time.Now().Add(time.Duration(hoursRemaining) * time.Hour)

	// Generate recommendation based on forecast
	switch {
	case forecast.DaysRemaining < 1:
		forecast.Recommendation = "CRITICAL: Budget will be exhausted within 24 hours. Immediate action required."
	case forecast.DaysRemaining < 7:
		forecast.Recommendation = "WARNING: Budget will be exhausted within a week. Investigate and address reliability issues."
	case forecast.DaysRemaining < 14:
		forecast.Recommendation = "CAUTION: Budget consumption is elevated. Monitor closely."
	default:
		forecast.Recommendation = "Budget consumption is within acceptable limits."
	}

	return forecast, nil
}

// BudgetPolicy defines policies for error budget governance
type BudgetPolicy struct {
	Name                string   `json:"name"`
	Description         string   `json:"description"`
	FreezeThreshold     float64  `json:"freezeThreshold"`  // Percentage at which to freeze deploys
	RestoreThreshold    float64  `json:"restoreThreshold"` // Percentage at which to restore deploys
	NotifyChannels      []string `json:"notifyChannels"`
	EnforceDeployFreeze bool     `json:"enforceDeployFreeze"`
}

// DefaultBudgetPolicy returns the default error budget policy
func DefaultBudgetPolicy() *BudgetPolicy {
	return &BudgetPolicy{
		Name:                "default",
		Description:         "Standard error budget policy",
		FreezeThreshold:     90.0, // Freeze at 90% consumed
		RestoreThreshold:    75.0, // Restore at 75% consumed
		EnforceDeployFreeze: true,
	}
}

// CheckDeploymentAllowed checks if deployments are allowed based on error budget
func (t *BudgetTracker) CheckDeploymentAllowed(ctx context.Context, sloID string, policy *BudgetPolicy) (bool, string) {
	budget, err := t.GetBudget(ctx, sloID)
	if err != nil {
		return true, "Unable to check budget, allowing deployment"
	}

	if policy == nil {
		policy = DefaultBudgetPolicy()
	}

	if budget.ConsumedPct >= policy.FreezeThreshold && policy.EnforceDeployFreeze {
		return false, fmt.Sprintf("Deployment frozen: error budget %.1f%% consumed (threshold: %.1f%%)",
			budget.ConsumedPct, policy.FreezeThreshold)
	}

	return true, ""
}
