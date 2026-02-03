package copilot

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/platformfoundry/platformfoundry-ce/internal/state"
)

var (
	ErrRequiresApproval = errors.New("action requires approval")
	ErrDangerousAction  = errors.New("dangerous action blocked")
	ErrSafetyViolation  = errors.New("safety rule violation")
)

// ActionPlan represents a plan of actions to execute
type ActionPlan struct {
	ID               string        `json:"id"`
	Description      string        `json:"description"`
	Steps            []Action      `json:"steps"`
	EstimatedTime    time.Duration `json:"estimatedTime"`
	RiskLevel        RiskLevel     `json:"riskLevel"`
	Reversible       bool          `json:"reversible"`
	RequiresApproval bool          `json:"requiresApproval"`
	SafetyChecks     []SafetyCheck `json:"safetyChecks"`
	CreatedAt        time.Time     `json:"createdAt"`
}

// SafetyCheck represents a safety check result
type SafetyCheck struct {
	Rule    string `json:"rule"`
	Passed  bool   `json:"passed"`
	Message string `json:"message,omitempty"`
}

// SafetyRule defines a safety rule for actions
type SafetyRule struct {
	Name        string
	Description string
	Check       func(action Action, ctx *PlatformContext) error
}

// ActionPlanner creates and validates action plans
type ActionPlanner struct {
	stateBackend  state.Backend
	safetyRules   []SafetyRule
	safetyEnabled bool
}

// NewActionPlanner creates a new action planner
func NewActionPlanner(backend state.Backend, safetyEnabled bool) *ActionPlanner {
	planner := &ActionPlanner{
		stateBackend:  backend,
		safetyEnabled: safetyEnabled,
		safetyRules:   DefaultSafetyRules,
	}

	return planner
}

// DefaultSafetyRules defines the default safety rules
var DefaultSafetyRules = []SafetyRule{
	{
		Name:        "no-production-delete-without-approval",
		Description: "Prevents deletion in production without explicit approval",
		Check: func(a Action, ctx *PlatformContext) error {
			if a.Type == "delete" && ctx.CurrentEnv == "production" {
				return ErrRequiresApproval
			}
			return nil
		},
	},
	{
		Name:        "no-scale-to-zero-production",
		Description: "Prevents scaling to zero replicas in production",
		Check: func(a Action, ctx *PlatformContext) error {
			if a.Type == "scale" && ctx.CurrentEnv == "production" {
				if replicas, ok := a.Params["replicas"].(int); ok && replicas == 0 {
					return ErrDangerousAction
				}
			}
			return nil
		},
	},
	{
		Name:        "require-approval-for-destructive-actions",
		Description: "Requires approval for destructive actions",
		Check: func(a Action, ctx *PlatformContext) error {
			destructiveTypes := map[string]bool{
				"delete":   true,
				"rollback": true,
				"restart":  true,
				"truncate": true,
			}
			if destructiveTypes[a.Type] {
				return ErrRequiresApproval
			}
			return nil
		},
	},
	{
		Name:        "limit-scale-magnitude",
		Description: "Prevents scaling by more than 10x in a single operation",
		Check: func(a Action, ctx *PlatformContext) error {
			if a.Type == "scale" {
				current, hasCurrent := a.Params["current_replicas"].(int)
				target, hasTarget := a.Params["replicas"].(int)
				if hasCurrent && hasTarget && current > 0 {
					if target > current*10 || (target < current/10 && target > 0) {
						return ErrRequiresApproval
					}
				}
			}
			return nil
		},
	},
	{
		Name:        "no-direct-database-modification",
		Description: "Prevents direct database modifications",
		Check: func(a Action, ctx *PlatformContext) error {
			if a.Type == "database-modify" || a.Type == "database-delete" {
				return ErrDangerousAction
			}
			return nil
		},
	},
}

// CreatePlan creates an action plan from an intent
func (p *ActionPlanner) CreatePlan(ctx context.Context, intent *Intent, platformCtx *PlatformContext) (*ActionPlan, error) {
	plan := &ActionPlan{
		ID:          generatePlanID(),
		Description: fmt.Sprintf("Plan for %s operation", intent.Type),
		Steps:       make([]Action, 0),
		CreatedAt:   time.Now(),
		Reversible:  true,
	}

	// Generate steps based on intent
	switch intent.Type {
	case IntentDeploy:
		p.planDeploy(plan, intent, platformCtx)
	case IntentScale:
		p.planScale(plan, intent, platformCtx)
	case IntentRollback:
		p.planRollback(plan, intent, platformCtx)
	case IntentConfigure:
		p.planConfigure(plan, intent, platformCtx)
	default:
		return nil, fmt.Errorf("unsupported intent type: %s", intent.Type)
	}

	// Run safety checks
	if p.safetyEnabled {
		if err := p.runSafetyChecks(plan, platformCtx); err != nil {
			if errors.Is(err, ErrRequiresApproval) {
				plan.RequiresApproval = true
			} else if errors.Is(err, ErrDangerousAction) {
				return nil, err
			}
		}
	}

	// Calculate overall risk level
	plan.RiskLevel = p.calculateRiskLevel(plan, platformCtx)

	// Estimate time
	plan.EstimatedTime = p.estimateTime(plan)

	return plan, nil
}

// planDeploy creates a deployment plan
func (p *ActionPlanner) planDeploy(plan *ActionPlan, intent *Intent, ctx *PlatformContext) {
	env := intent.Entities["environment"]
	if env == "" {
		env = "staging"
	}

	service := intent.Entities["service"]
	if service == "" {
		service = "all"
	}

	plan.Description = fmt.Sprintf("Deploy %s to %s", service, env)

	// Add deployment steps
	plan.Steps = append(plan.Steps,
		Action{
			ID:          "validate",
			Type:        "validate",
			Description: "Validate deployment configuration",
			RiskLevel:   RiskLow,
			Reversible:  true,
		},
		Action{
			ID:          "build",
			Type:        "build",
			Description: "Build application artifacts",
			RiskLevel:   RiskLow,
			Reversible:  true,
		},
		Action{
			ID:          "pre-check",
			Type:        "health-check",
			Description: "Run pre-deployment health checks",
			RiskLevel:   RiskLow,
			Reversible:  true,
		},
		Action{
			ID:          "deploy",
			Type:        "deploy",
			Description: fmt.Sprintf("Deploy to %s with rolling update", env),
			RiskLevel:   p.getRiskLevelForEnv(env),
			Reversible:  true,
			Params: map[string]interface{}{
				"environment": env,
				"service":     service,
				"strategy":    "rolling",
			},
		},
		Action{
			ID:          "post-check",
			Type:        "health-check",
			Description: "Verify deployment health",
			RiskLevel:   RiskLow,
			Reversible:  true,
		},
	)
}

// planScale creates a scaling plan
func (p *ActionPlanner) planScale(plan *ActionPlan, intent *Intent, ctx *PlatformContext) {
	service := intent.Entities["service"]
	count := intent.Entities["count"]

	plan.Description = fmt.Sprintf("Scale %s to %s replicas", service, count)

	plan.Steps = append(plan.Steps,
		Action{
			ID:          "validate",
			Type:        "validate",
			Description: "Validate scaling request",
			RiskLevel:   RiskLow,
			Reversible:  true,
		},
		Action{
			ID:          "scale",
			Type:        "scale",
			Description: fmt.Sprintf("Scale %s to %s replicas", service, count),
			RiskLevel:   RiskMedium,
			Reversible:  true,
			Params: map[string]interface{}{
				"service":  service,
				"replicas": count,
			},
		},
		Action{
			ID:          "verify",
			Type:        "health-check",
			Description: "Verify scaled instances are healthy",
			RiskLevel:   RiskLow,
			Reversible:  true,
		},
	)
}

// planRollback creates a rollback plan
func (p *ActionPlanner) planRollback(plan *ActionPlan, intent *Intent, ctx *PlatformContext) {
	service := intent.Entities["service"]
	if service == "" {
		service = "unknown"
	}

	plan.Description = fmt.Sprintf("Rollback %s to previous version", service)
	plan.RequiresApproval = true

	plan.Steps = append(plan.Steps,
		Action{
			ID:          "identify",
			Type:        "query",
			Description: "Identify previous stable version",
			RiskLevel:   RiskLow,
			Reversible:  true,
		},
		Action{
			ID:          "rollback",
			Type:        "rollback",
			Description: fmt.Sprintf("Rollback %s to previous version", service),
			RiskLevel:   RiskHigh,
			Reversible:  true,
			Params: map[string]interface{}{
				"service": service,
			},
		},
		Action{
			ID:          "verify",
			Type:        "health-check",
			Description: "Verify rollback success",
			RiskLevel:   RiskLow,
			Reversible:  true,
		},
	)
}

// planConfigure creates a configuration plan
func (p *ActionPlanner) planConfigure(plan *ActionPlan, intent *Intent, ctx *PlatformContext) {
	plan.Description = "Update configuration"

	plan.Steps = append(plan.Steps,
		Action{
			ID:          "backup",
			Type:        "backup",
			Description: "Backup current configuration",
			RiskLevel:   RiskLow,
			Reversible:  true,
		},
		Action{
			ID:          "validate",
			Type:        "validate",
			Description: "Validate new configuration",
			RiskLevel:   RiskLow,
			Reversible:  true,
		},
		Action{
			ID:          "apply",
			Type:        "configure",
			Description: "Apply configuration changes",
			RiskLevel:   RiskMedium,
			Reversible:  true,
		},
		Action{
			ID:          "verify",
			Type:        "health-check",
			Description: "Verify configuration is working",
			RiskLevel:   RiskLow,
			Reversible:  true,
		},
	)
}

// runSafetyChecks runs all safety rules against the plan
func (p *ActionPlanner) runSafetyChecks(plan *ActionPlan, ctx *PlatformContext) error {
	var lastErr error

	for _, step := range plan.Steps {
		for _, rule := range p.safetyRules {
			check := SafetyCheck{
				Rule:   rule.Name,
				Passed: true,
			}

			if err := rule.Check(step, ctx); err != nil {
				check.Passed = false
				check.Message = err.Error()
				lastErr = err
			}

			plan.SafetyChecks = append(plan.SafetyChecks, check)
		}
	}

	return lastErr
}

// calculateRiskLevel calculates the overall risk level of the plan
func (p *ActionPlanner) calculateRiskLevel(plan *ActionPlan, ctx *PlatformContext) RiskLevel {
	maxRisk := RiskLow

	for _, step := range plan.Steps {
		if compareRiskLevel(step.RiskLevel, maxRisk) > 0 {
			maxRisk = step.RiskLevel
		}
	}

	// Increase risk level for production
	if ctx.CurrentEnv == "production" && maxRisk != RiskCritical {
		switch maxRisk {
		case RiskLow:
			maxRisk = RiskMedium
		case RiskMedium:
			maxRisk = RiskHigh
		case RiskHigh:
			maxRisk = RiskCritical
		}
	}

	return maxRisk
}

// estimateTime estimates the total time for the plan
func (p *ActionPlanner) estimateTime(plan *ActionPlan) time.Duration {
	var total time.Duration

	for _, step := range plan.Steps {
		switch step.Type {
		case "validate":
			total += 10 * time.Second
		case "build":
			total += 2 * time.Minute
		case "deploy":
			total += 5 * time.Minute
		case "scale":
			total += 2 * time.Minute
		case "rollback":
			total += 3 * time.Minute
		case "health-check":
			total += 30 * time.Second
		default:
			total += 1 * time.Minute
		}
	}

	return total
}

// getRiskLevelForEnv returns the base risk level for an environment
func (p *ActionPlanner) getRiskLevelForEnv(env string) RiskLevel {
	switch env {
	case "production", "prod":
		return RiskHigh
	case "staging", "stage":
		return RiskMedium
	default:
		return RiskLow
	}
}

// ValidatePlan validates a plan before execution
func (p *ActionPlanner) ValidatePlan(ctx context.Context, plan *ActionPlan) error {
	if len(plan.Steps) == 0 {
		return fmt.Errorf("plan has no steps")
	}

	for _, step := range plan.Steps {
		if step.Type == "" {
			return fmt.Errorf("step %s has no type", step.ID)
		}
	}

	return nil
}

// ExecutePlan executes a plan (placeholder - actual execution would be handled by orchestrator)
func (p *ActionPlanner) ExecutePlan(ctx context.Context, plan *ActionPlan) error {
	if plan.RequiresApproval {
		return ErrRequiresApproval
	}

	if err := p.ValidatePlan(ctx, plan); err != nil {
		return err
	}

	// In production, this would delegate to the orchestrator
	return nil
}

// AddSafetyRule adds a custom safety rule
func (p *ActionPlanner) AddSafetyRule(rule SafetyRule) {
	p.safetyRules = append(p.safetyRules, rule)
}

// SetSafetyEnabled enables or disables safety checks
func (p *ActionPlanner) SetSafetyEnabled(enabled bool) {
	p.safetyEnabled = enabled
}

// Helper functions

func generatePlanID() string {
	return fmt.Sprintf("plan-%d", time.Now().UnixNano())
}

func compareRiskLevel(a, b RiskLevel) int {
	levels := map[RiskLevel]int{
		RiskLow:      1,
		RiskMedium:   2,
		RiskHigh:     3,
		RiskCritical: 4,
	}

	return levels[a] - levels[b]
}
