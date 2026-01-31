// Package remediation provides auto-remediation for detected drift and policy violations.
package remediation

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// ActionType represents the type of remediation action
type ActionType string

const (
	ActionAutoFix         ActionType = "auto_fix"
	ActionAlertOnly       ActionType = "alert_only"
	ActionRequireApproval ActionType = "require_approval"
	ActionCreateTicket    ActionType = "create_ticket"
)

// TriggerType represents what triggers remediation
type TriggerType string

const (
	TriggerDrift           TriggerType = "drift"
	TriggerPolicyViolation TriggerType = "policy_violation"
	TriggerHealthScore     TriggerType = "health_score"
)

// Severity represents issue severity
type Severity string

const (
	SeverityCritical Severity = "critical"
	SeverityHigh     Severity = "high"
	SeverityMedium   Severity = "medium"
	SeverityLow      Severity = "low"
)

// Rule defines a remediation rule
type Rule struct {
	Name       string            `yaml:"name" json:"name"`
	Trigger    RuleTrigger       `yaml:"trigger" json:"trigger"`
	Conditions []RuleCondition   `yaml:"conditions,omitempty" json:"conditions,omitempty"`
	Action     RuleAction        `yaml:"action" json:"action"`
	Notify     []string          `yaml:"notify,omitempty" json:"notify,omitempty"`
	Escalate   *EscalationPolicy `yaml:"escalate,omitempty" json:"escalate,omitempty"`
	Enabled    bool              `yaml:"enabled" json:"enabled"`
}

// RuleTrigger defines when a rule is triggered
type RuleTrigger struct {
	Type     TriggerType `yaml:"type" json:"type"`
	Severity []Severity  `yaml:"severity,omitempty" json:"severity,omitempty"`
	MaxAge   string      `yaml:"max_age,omitempty" json:"max_age,omitempty"`
	Resource string      `yaml:"resource,omitempty" json:"resource,omitempty"`
}

// RuleCondition defines additional conditions for a rule
type RuleCondition struct {
	Environment []string `yaml:"environment,omitempty" json:"environment,omitempty"`
	Resource    string   `yaml:"resource,omitempty" json:"resource,omitempty"`
	Tag         string   `yaml:"tag,omitempty" json:"tag,omitempty"`
}

// RuleAction defines what action to take
type RuleAction struct {
	Type           ActionType `yaml:"type" json:"type"`
	ApprovalPolicy string     `yaml:"approval_policy,omitempty" json:"approval_policy,omitempty"`
	TicketProject  string     `yaml:"ticket_project,omitempty" json:"ticket_project,omitempty"`
}

// EscalationPolicy defines escalation behavior
type EscalationPolicy struct {
	After string   `yaml:"after" json:"after"`
	To    []string `yaml:"to" json:"to"`
}

// Issue represents an issue that may need remediation
type Issue struct {
	ID          string                 `json:"id"`
	Type        TriggerType            `json:"type"`
	Severity    Severity               `json:"severity"`
	Resource    string                 `json:"resource"`
	Environment string                 `json:"environment"`
	Description string                 `json:"description"`
	DetectedAt  time.Time              `json:"detected_at"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
}

// RemediationResult represents the result of a remediation attempt
type RemediationResult struct {
	Issue      Issue      `json:"issue"`
	Rule       string     `json:"rule"`
	Action     ActionType `json:"action"`
	Success    bool       `json:"success"`
	Message    string     `json:"message"`
	ExecutedAt time.Time  `json:"executed_at"`
	Duration   string     `json:"duration"`
}

// Config holds remediation configuration
type Config struct {
	Enabled       bool          `yaml:"enabled" json:"enabled"`
	Rules         []Rule        `yaml:"rules" json:"rules"`
	DefaultAction ActionType    `yaml:"default_action" json:"default_action"`
	CheckInterval time.Duration `yaml:"check_interval" json:"check_interval"`
}

// DefaultConfig returns default remediation configuration
func DefaultConfig() *Config {
	return &Config{
		Enabled:       true,
		DefaultAction: ActionAlertOnly,
		CheckInterval: 5 * time.Minute,
		Rules: []Rule{
			{
				Name:    "auto-sync-low-severity",
				Enabled: true,
				Trigger: RuleTrigger{
					Type:     TriggerDrift,
					Severity: []Severity{SeverityLow},
					MaxAge:   "1h",
				},
				Conditions: []RuleCondition{
					{Environment: []string{"dev", "staging"}},
				},
				Action: RuleAction{
					Type: ActionAutoFix,
				},
				Notify: []string{"slack"},
			},
			{
				Name:    "alert-production-drift",
				Enabled: true,
				Trigger: RuleTrigger{
					Type:     TriggerDrift,
					Severity: []Severity{SeverityMedium, SeverityHigh},
				},
				Conditions: []RuleCondition{
					{Environment: []string{"production"}},
				},
				Action: RuleAction{
					Type:           ActionRequireApproval,
					ApprovalPolicy: "platform-leads",
				},
				Notify: []string{"pagerduty", "slack"},
			},
			{
				Name:    "critical-immediate-alert",
				Enabled: true,
				Trigger: RuleTrigger{
					Type:     TriggerDrift,
					Severity: []Severity{SeverityCritical},
				},
				Action: RuleAction{
					Type: ActionAlertOnly,
				},
				Notify: []string{"pagerduty"},
				Escalate: &EscalationPolicy{
					After: "15m",
					To:    []string{"on-call-lead"},
				},
			},
		},
	}
}

// Notifier interface for sending notifications
type Notifier interface {
	Send(ctx context.Context, channel string, message string) error
}

// Applier interface for applying fixes
type Applier interface {
	Fix(ctx context.Context, issue Issue) error
}

// ApprovalRequester interface for requesting approvals
type ApprovalRequester interface {
	RequestApproval(ctx context.Context, policy string, issue Issue) (string, error)
}

// Engine handles auto-remediation
type Engine struct {
	config    *Config
	notifier  Notifier
	applier   Applier
	approvals ApprovalRequester
	history   []RemediationResult
	mu        sync.RWMutex
	running   bool
	stopCh    chan struct{}
}

// NewEngine creates a new remediation engine
func NewEngine(config *Config) *Engine {
	if config == nil {
		config = DefaultConfig()
	}
	return &Engine{
		config:  config,
		history: make([]RemediationResult, 0),
		stopCh:  make(chan struct{}),
	}
}

// WithNotifier sets the notifier
func (e *Engine) WithNotifier(n Notifier) *Engine {
	e.notifier = n
	return e
}

// WithApplier sets the applier
func (e *Engine) WithApplier(a Applier) *Engine {
	e.applier = a
	return e
}

// WithApprovalRequester sets the approval requester
func (e *Engine) WithApprovalRequester(ar ApprovalRequester) *Engine {
	e.approvals = ar
	return e
}

// ProcessIssue processes a single issue and determines remediation action
func (e *Engine) ProcessIssue(ctx context.Context, issue Issue) (*RemediationResult, error) {
	start := time.Now()

	result := &RemediationResult{
		Issue:      issue,
		ExecutedAt: start,
	}

	// Find matching rule
	rule := e.findMatchingRule(issue)
	if rule == nil {
		result.Action = e.config.DefaultAction
		result.Rule = "default"
	} else {
		result.Action = rule.Action.Type
		result.Rule = rule.Name
	}

	// Execute action
	var err error
	switch result.Action {
	case ActionAutoFix:
		err = e.executeAutoFix(ctx, issue, rule)
	case ActionRequireApproval:
		err = e.executeApproval(ctx, issue, rule)
	case ActionCreateTicket:
		err = e.executeCreateTicket(ctx, issue, rule)
	case ActionAlertOnly:
		err = e.executeAlert(ctx, issue, rule)
	}

	result.Duration = time.Since(start).String()

	if err != nil {
		result.Success = false
		result.Message = err.Error()
	} else {
		result.Success = true
		result.Message = fmt.Sprintf("Successfully executed %s action", result.Action)
	}

	// Store in history
	e.mu.Lock()
	e.history = append(e.history, *result)
	// Keep last 1000 results
	if len(e.history) > 1000 {
		e.history = e.history[len(e.history)-1000:]
	}
	e.mu.Unlock()

	return result, err
}

// findMatchingRule finds the first matching rule for an issue
func (e *Engine) findMatchingRule(issue Issue) *Rule {
	for i := range e.config.Rules {
		rule := &e.config.Rules[i]
		if !rule.Enabled {
			continue
		}

		// Check trigger type
		if rule.Trigger.Type != issue.Type {
			continue
		}

		// Check severity
		if len(rule.Trigger.Severity) > 0 {
			matched := false
			for _, s := range rule.Trigger.Severity {
				if s == issue.Severity {
					matched = true
					break
				}
			}
			if !matched {
				continue
			}
		}

		// Check resource filter
		if rule.Trigger.Resource != "" && rule.Trigger.Resource != issue.Resource {
			continue
		}

		// Check conditions
		if !e.checkConditions(rule.Conditions, issue) {
			continue
		}

		return rule
	}
	return nil
}

// checkConditions checks if all conditions are met
func (e *Engine) checkConditions(conditions []RuleCondition, issue Issue) bool {
	if len(conditions) == 0 {
		return true
	}

	for _, cond := range conditions {
		// Check environment
		if len(cond.Environment) > 0 {
			matched := false
			for _, env := range cond.Environment {
				if env == issue.Environment {
					matched = true
					break
				}
			}
			if !matched {
				return false
			}
		}

		// Check resource
		if cond.Resource != "" && cond.Resource != issue.Resource {
			return false
		}
	}

	return true
}

// executeAutoFix attempts to automatically fix the issue
func (e *Engine) executeAutoFix(ctx context.Context, issue Issue, rule *Rule) error {
	if e.applier == nil {
		return fmt.Errorf("no applier configured for auto-fix")
	}

	// Send notification before fix
	if rule != nil && len(rule.Notify) > 0 {
		e.sendNotifications(ctx, rule.Notify, fmt.Sprintf(
			"Auto-fixing %s issue: %s\nResource: %s\nEnvironment: %s",
			issue.Type, issue.Description, issue.Resource, issue.Environment,
		))
	}

	// Apply fix
	if err := e.applier.Fix(ctx, issue); err != nil {
		e.sendNotifications(ctx, rule.Notify, fmt.Sprintf(
			"Auto-fix FAILED for %s: %s\nError: %s",
			issue.Resource, issue.Description, err.Error(),
		))
		return err
	}

	// Send success notification
	e.sendNotifications(ctx, rule.Notify, fmt.Sprintf(
		"Auto-fix SUCCESSFUL for %s: %s",
		issue.Resource, issue.Description,
	))

	return nil
}

// executeApproval requests approval for remediation
func (e *Engine) executeApproval(ctx context.Context, issue Issue, rule *Rule) error {
	if e.approvals == nil {
		return fmt.Errorf("no approval requester configured")
	}

	policy := "default"
	if rule != nil && rule.Action.ApprovalPolicy != "" {
		policy = rule.Action.ApprovalPolicy
	}

	approvalID, err := e.approvals.RequestApproval(ctx, policy, issue)
	if err != nil {
		return err
	}

	// Notify about pending approval
	if rule != nil && len(rule.Notify) > 0 {
		e.sendNotifications(ctx, rule.Notify, fmt.Sprintf(
			"Approval requested for %s remediation\nResource: %s\nEnvironment: %s\nApproval ID: %s\nPolicy: %s",
			issue.Type, issue.Resource, issue.Environment, approvalID, policy,
		))
	}

	return nil
}

// executeCreateTicket creates a ticket for the issue
func (e *Engine) executeCreateTicket(ctx context.Context, issue Issue, rule *Rule) error {
	// In production, this would integrate with Jira, GitHub Issues, etc.
	if rule != nil && len(rule.Notify) > 0 {
		e.sendNotifications(ctx, rule.Notify, fmt.Sprintf(
			"Ticket created for %s issue\nResource: %s\nDescription: %s",
			issue.Type, issue.Resource, issue.Description,
		))
	}
	return nil
}

// executeAlert sends alerts without taking action
func (e *Engine) executeAlert(ctx context.Context, issue Issue, rule *Rule) error {
	channels := []string{"default"}
	if rule != nil && len(rule.Notify) > 0 {
		channels = rule.Notify
	}

	e.sendNotifications(ctx, channels, fmt.Sprintf(
		"ALERT: %s detected\nSeverity: %s\nResource: %s\nEnvironment: %s\nDescription: %s",
		issue.Type, issue.Severity, issue.Resource, issue.Environment, issue.Description,
	))

	return nil
}

// sendNotifications sends notifications to specified channels
func (e *Engine) sendNotifications(ctx context.Context, channels []string, message string) {
	if e.notifier == nil {
		return
	}

	for _, channel := range channels {
		_ = e.notifier.Send(ctx, channel, message)
	}
}

// GetHistory returns remediation history
func (e *Engine) GetHistory() []RemediationResult {
	e.mu.RLock()
	defer e.mu.RUnlock()

	result := make([]RemediationResult, len(e.history))
	copy(result, e.history)
	return result
}

// GetStatus returns current remediation engine status
func (e *Engine) GetStatus() map[string]interface{} {
	e.mu.RLock()
	defer e.mu.RUnlock()

	return map[string]interface{}{
		"enabled":       e.config.Enabled,
		"running":       e.running,
		"rules_count":   len(e.config.Rules),
		"history_count": len(e.history),
	}
}

// AddRule adds a new remediation rule
func (e *Engine) AddRule(rule Rule) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.config.Rules = append(e.config.Rules, rule)
}

// RemoveRule removes a rule by name
func (e *Engine) RemoveRule(name string) bool {
	e.mu.Lock()
	defer e.mu.Unlock()

	for i, rule := range e.config.Rules {
		if rule.Name == name {
			e.config.Rules = append(e.config.Rules[:i], e.config.Rules[i+1:]...)
			return true
		}
	}
	return false
}

// EnableRule enables a rule by name
func (e *Engine) EnableRule(name string) bool {
	e.mu.Lock()
	defer e.mu.Unlock()

	for i := range e.config.Rules {
		if e.config.Rules[i].Name == name {
			e.config.Rules[i].Enabled = true
			return true
		}
	}
	return false
}

// DisableRule disables a rule by name
func (e *Engine) DisableRule(name string) bool {
	e.mu.Lock()
	defer e.mu.Unlock()

	for i := range e.config.Rules {
		if e.config.Rules[i].Name == name {
			e.config.Rules[i].Enabled = false
			return true
		}
	}
	return false
}
