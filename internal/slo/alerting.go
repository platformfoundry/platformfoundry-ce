package slo

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// AlertingEngine manages SLO-based alerting
type AlertingEngine struct {
	engine          *Engine
	budgetTracker   *BudgetTracker
	alertManager    AlertManager
	alertRules      map[string]*AlertRule
	activeAlerts    map[string]*Alert
	silences        map[string]*Silence
	mu              sync.RWMutex
}

// AlertRule defines when to fire alerts
type AlertRule struct {
	ID          string        `json:"id"`
	Name        string        `json:"name"`
	Description string        `json:"description"`
	SLOID       string        `json:"sloId"`
	Type        AlertRuleType `json:"type"`
	Threshold   float64       `json:"threshold"`
	Duration    time.Duration `json:"duration"`  // How long condition must be true
	Severity    string        `json:"severity"`
	Labels      map[string]string `json:"labels,omitempty"`
	Annotations map[string]string `json:"annotations,omitempty"`
	Enabled     bool          `json:"enabled"`
}

// AlertRuleType defines the type of alert rule
type AlertRuleType string

const (
	AlertRuleTypeBudgetThreshold  AlertRuleType = "budget_threshold"
	AlertRuleTypeBurnRate         AlertRuleType = "burn_rate"
	AlertRuleTypeSLIBreach        AlertRuleType = "sli_breach"
	AlertRuleTypeTrendPrediction  AlertRuleType = "trend_prediction"
)

// Silence represents an alert silence period
type Silence struct {
	ID        string            `json:"id"`
	Matchers  map[string]string `json:"matchers"` // Labels to match
	StartsAt  time.Time         `json:"startsAt"`
	EndsAt    time.Time         `json:"endsAt"`
	CreatedBy string            `json:"createdBy"`
	Comment   string            `json:"comment"`
}

// NewAlertingEngine creates a new AlertingEngine
func NewAlertingEngine(engine *Engine, budgetTracker *BudgetTracker, alertManager AlertManager) *AlertingEngine {
	return &AlertingEngine{
		engine:        engine,
		budgetTracker: budgetTracker,
		alertManager:  alertManager,
		alertRules:    make(map[string]*AlertRule),
		activeAlerts:  make(map[string]*Alert),
		silences:      make(map[string]*Silence),
	}
}

// RegisterAlertRule registers a new alert rule
func (e *AlertingEngine) RegisterAlertRule(ctx context.Context, rule *AlertRule) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	if rule.ID == "" {
		rule.ID = fmt.Sprintf("rule-%s-%d", rule.Name, time.Now().UnixNano())
	}

	e.alertRules[rule.ID] = rule
	return nil
}

// DeleteAlertRule removes an alert rule
func (e *AlertingEngine) DeleteAlertRule(ctx context.Context, ruleID string) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	if _, ok := e.alertRules[ruleID]; !ok {
		return fmt.Errorf("alert rule not found: %s", ruleID)
	}

	delete(e.alertRules, ruleID)
	return nil
}

// EvaluateRules evaluates all alert rules and fires/resolves alerts as needed
func (e *AlertingEngine) EvaluateRules(ctx context.Context) ([]Alert, error) {
	e.mu.RLock()
	rules := make([]*AlertRule, 0, len(e.alertRules))
	for _, rule := range e.alertRules {
		if rule.Enabled {
			rules = append(rules, rule)
		}
	}
	e.mu.RUnlock()

	var firedAlerts []Alert

	for _, rule := range rules {
		shouldFire, value, err := e.evaluateRule(ctx, rule)
		if err != nil {
			continue // Log in production
		}

		alertID := fmt.Sprintf("%s-%s", rule.ID, rule.SLOID)

		if shouldFire {
			// Check if already active
			e.mu.RLock()
			_, exists := e.activeAlerts[alertID]
			e.mu.RUnlock()

			if !exists && !e.isSilenced(rule) {
				alert := Alert{
					ID:        alertID,
					SLO:       rule.SLOID,
					Severity:  rule.Severity,
					Message:   e.formatAlertMessage(rule, value),
					Value:     value,
					Threshold: rule.Threshold,
					Labels:    rule.Labels,
					FiredAt:   time.Now(),
				}

				e.mu.Lock()
				e.activeAlerts[alertID] = &alert
				e.mu.Unlock()

				if e.alertManager != nil {
					e.alertManager.Fire(ctx, alert)
				}

				firedAlerts = append(firedAlerts, alert)
			}
		} else {
			// Resolve if previously active
			e.mu.Lock()
			if alert, exists := e.activeAlerts[alertID]; exists {
				now := time.Now()
				alert.ResolvedAt = &now
				delete(e.activeAlerts, alertID)

				if e.alertManager != nil {
					e.alertManager.Resolve(ctx, alertID)
				}
			}
			e.mu.Unlock()
		}
	}

	return firedAlerts, nil
}

// evaluateRule evaluates a single alert rule
func (e *AlertingEngine) evaluateRule(ctx context.Context, rule *AlertRule) (bool, float64, error) {
	switch rule.Type {
	case AlertRuleTypeBudgetThreshold:
		return e.evaluateBudgetThreshold(ctx, rule)
	case AlertRuleTypeBurnRate:
		return e.evaluateBurnRate(ctx, rule)
	case AlertRuleTypeSLIBreach:
		return e.evaluateSLIBreach(ctx, rule)
	case AlertRuleTypeTrendPrediction:
		return e.evaluateTrendPrediction(ctx, rule)
	default:
		return false, 0, fmt.Errorf("unknown rule type: %s", rule.Type)
	}
}

// evaluateBudgetThreshold checks if error budget consumption exceeds threshold
func (e *AlertingEngine) evaluateBudgetThreshold(ctx context.Context, rule *AlertRule) (bool, float64, error) {
	budget, err := e.budgetTracker.GetBudget(ctx, rule.SLOID)
	if err != nil {
		return false, 0, err
	}

	return budget.ConsumedPct >= rule.Threshold, budget.ConsumedPct, nil
}

// evaluateBurnRate checks if burn rate exceeds threshold
func (e *AlertingEngine) evaluateBurnRate(ctx context.Context, rule *AlertRule) (bool, float64, error) {
	budget, err := e.budgetTracker.GetBudget(ctx, rule.SLOID)
	if err != nil {
		return false, 0, err
	}

	return budget.BurnRate >= rule.Threshold, budget.BurnRate, nil
}

// evaluateSLIBreach checks if SLI is below objective
func (e *AlertingEngine) evaluateSLIBreach(ctx context.Context, rule *AlertRule) (bool, float64, error) {
	slo, err := e.engine.GetSLO(ctx, rule.SLOID)
	if err != nil {
		return false, 0, err
	}

	result, err := e.engine.CalculateSLI(ctx, slo)
	if err != nil {
		return false, 0, err
	}

	return result.Value < slo.Objective, result.Value, nil
}

// evaluateTrendPrediction checks if trend predicts budget exhaustion
func (e *AlertingEngine) evaluateTrendPrediction(ctx context.Context, rule *AlertRule) (bool, float64, error) {
	forecast, err := e.budgetTracker.ForecastBudget(ctx, rule.SLOID)
	if err != nil {
		return false, 0, err
	}

	// Threshold represents days until exhaustion
	return forecast.DaysRemaining <= rule.Threshold && forecast.DaysRemaining > 0, forecast.DaysRemaining, nil
}

// isSilenced checks if an alert rule matches any active silences
func (e *AlertingEngine) isSilenced(rule *AlertRule) bool {
	e.mu.RLock()
	defer e.mu.RUnlock()

	now := time.Now()
	for _, silence := range e.silences {
		if now.Before(silence.StartsAt) || now.After(silence.EndsAt) {
			continue
		}

		// Check if all matchers match
		allMatch := true
		for key, value := range silence.Matchers {
			if rule.Labels[key] != value {
				allMatch = false
				break
			}
		}

		if allMatch {
			return true
		}
	}

	return false
}

// formatAlertMessage formats the alert message based on rule type
func (e *AlertingEngine) formatAlertMessage(rule *AlertRule, value float64) string {
	switch rule.Type {
	case AlertRuleTypeBudgetThreshold:
		return fmt.Sprintf("Error budget for %s: %.1f%% consumed (threshold: %.1f%%)",
			rule.SLOID, value, rule.Threshold)
	case AlertRuleTypeBurnRate:
		return fmt.Sprintf("Error budget burn rate for %s: %.2fx (threshold: %.2fx)",
			rule.SLOID, value, rule.Threshold)
	case AlertRuleTypeSLIBreach:
		return fmt.Sprintf("SLI breach for %s: %.2f%% (objective: %.2f%%)",
			rule.SLOID, value, rule.Threshold)
	case AlertRuleTypeTrendPrediction:
		return fmt.Sprintf("Error budget for %s predicted to exhaust in %.1f days",
			rule.SLOID, value)
	default:
		return rule.Description
	}
}

// CreateSilence creates a new alert silence
func (e *AlertingEngine) CreateSilence(ctx context.Context, silence *Silence) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	if silence.ID == "" {
		silence.ID = fmt.Sprintf("silence-%d", time.Now().UnixNano())
	}

	e.silences[silence.ID] = silence
	return nil
}

// DeleteSilence removes a silence
func (e *AlertingEngine) DeleteSilence(ctx context.Context, silenceID string) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	if _, ok := e.silences[silenceID]; !ok {
		return fmt.Errorf("silence not found: %s", silenceID)
	}

	delete(e.silences, silenceID)
	return nil
}

// ListSilences returns all silences
func (e *AlertingEngine) ListSilences(ctx context.Context) []*Silence {
	e.mu.RLock()
	defer e.mu.RUnlock()

	silences := make([]*Silence, 0, len(e.silences))
	for _, silence := range e.silences {
		silences = append(silences, silence)
	}

	return silences
}

// GetActiveAlerts returns all active alerts
func (e *AlertingEngine) GetActiveAlerts(ctx context.Context) []*Alert {
	e.mu.RLock()
	defer e.mu.RUnlock()

	alerts := make([]*Alert, 0, len(e.activeAlerts))
	for _, alert := range e.activeAlerts {
		alerts = append(alerts, alert)
	}

	return alerts
}

// StartPeriodicEvaluation starts a goroutine that periodically evaluates alert rules
func (e *AlertingEngine) StartPeriodicEvaluation(ctx context.Context, interval time.Duration) {
	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				e.EvaluateRules(ctx)
			}
		}
	}()
}

// DefaultAlertRules returns commonly used alert rules for an SLO
func DefaultAlertRules(sloID string) []*AlertRule {
	return []*AlertRule{
		{
			Name:        "budget-critical",
			Description: "Error budget critically low",
			SLOID:       sloID,
			Type:        AlertRuleTypeBudgetThreshold,
			Threshold:   90,
			Severity:    "critical",
			Enabled:     true,
		},
		{
			Name:        "budget-warning",
			Description: "Error budget consumption elevated",
			SLOID:       sloID,
			Type:        AlertRuleTypeBudgetThreshold,
			Threshold:   75,
			Severity:    "warning",
			Enabled:     true,
		},
		{
			Name:        "burn-rate-critical",
			Description: "High error budget burn rate",
			SLOID:       sloID,
			Type:        AlertRuleTypeBurnRate,
			Threshold:   14.4, // Will exhaust 30-day budget in 2 days
			Severity:    "critical",
			Enabled:     true,
		},
		{
			Name:        "burn-rate-warning",
			Description: "Elevated error budget burn rate",
			SLOID:       sloID,
			Type:        AlertRuleTypeBurnRate,
			Threshold:   6.0, // Will exhaust 30-day budget in 5 days
			Severity:    "warning",
			Enabled:     true,
		},
		{
			Name:        "sli-breach",
			Description: "SLI below objective",
			SLOID:       sloID,
			Type:        AlertRuleTypeSLIBreach,
			Duration:    5 * time.Minute,
			Severity:    "critical",
			Enabled:     true,
		},
		{
			Name:        "budget-exhaustion-prediction",
			Description: "Error budget predicted to exhaust soon",
			SLOID:       sloID,
			Type:        AlertRuleTypeTrendPrediction,
			Threshold:   7, // Days until exhaustion
			Severity:    "warning",
			Enabled:     true,
		},
	}
}

// AlertSummary summarizes alert status
type AlertSummary struct {
	TotalActive int            `json:"totalActive"`
	BySeverity  map[string]int `json:"bySeverity"`
	BySLO       map[string]int `json:"bySLO"`
	GeneratedAt time.Time      `json:"generatedAt"`
}

// GetAlertSummary returns a summary of active alerts
func (e *AlertingEngine) GetAlertSummary(ctx context.Context) *AlertSummary {
	alerts := e.GetActiveAlerts(ctx)

	summary := &AlertSummary{
		TotalActive: len(alerts),
		BySeverity:  make(map[string]int),
		BySLO:       make(map[string]int),
		GeneratedAt: time.Now(),
	}

	for _, alert := range alerts {
		summary.BySeverity[alert.Severity]++
		summary.BySLO[alert.SLO]++
	}

	return summary
}
