package gitops

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// DeployStrategy defines how an application should be deployed
type DeployStrategy struct {
	Type     string         `yaml:"type" json:"type"` // canary, blue-green, rolling, recreate
	Steps    []ProgressStep `yaml:"steps,omitempty" json:"steps,omitempty"`
	Pause    PauseConfig    `yaml:"pause,omitempty" json:"pause,omitempty"`
	Rollback RollbackConfig `yaml:"rollback,omitempty" json:"rollback,omitempty"`
}

// ProgressStep represents a step in progressive delivery
type ProgressStep struct {
	Weight    int           `yaml:"weight" json:"weight"`                   // Traffic percentage (0-100)
	WaitTime  time.Duration `yaml:"waitTime" json:"waitTime"`               // Time to wait before next step
	Analysis  bool          `yaml:"analysis" json:"analysis"`               // Whether to run analysis
	SetHeader string        `yaml:"setHeader,omitempty" json:"setHeader"`   // Header-based routing
	SetWeight int           `yaml:"setWeight,omitempty" json:"setWeight"`   // Explicit weight override
	Pause     *PauseConfig  `yaml:"pause,omitempty" json:"pause,omitempty"` // Step-specific pause
}

// PauseConfig defines when to pause during deployment
type PauseConfig struct {
	Duration        time.Duration `yaml:"duration,omitempty" json:"duration,omitempty"`
	RequireApproval bool          `yaml:"requireApproval,omitempty" json:"requireApproval,omitempty"`
}

// RollbackConfig defines rollback behavior
type RollbackConfig struct {
	Automatic    bool          `yaml:"automatic" json:"automatic"`
	MaxRetries   int           `yaml:"maxRetries" json:"maxRetries"`
	BackoffDelay time.Duration `yaml:"backoffDelay" json:"backoffDelay"`
	OnFailure    string        `yaml:"onFailure" json:"onFailure"` // abort, rollback, pause
}

// AnalysisRule defines a metric-based analysis rule
type AnalysisRule struct {
	Name         string            `yaml:"name" json:"name"`
	MetricName   string            `yaml:"metricName" json:"metricName"`
	Query        string            `yaml:"query" json:"query"` // PromQL or custom query
	Threshold    float64           `yaml:"threshold" json:"threshold"`
	Operator     string            `yaml:"operator" json:"operator"` // lt, lte, gt, gte, eq
	FailureLimit int               `yaml:"failureLimit" json:"failureLimit"`
	Labels       map[string]string `yaml:"labels,omitempty" json:"labels,omitempty"`
}

// DeploymentIntent represents the desired deployment state
type DeploymentIntent struct {
	ID                string         `json:"id"`
	Application       string         `json:"application"`
	Namespace         string         `json:"namespace"`
	TargetEnvironment string         `json:"targetEnvironment"`
	SourceRevision    string         `json:"sourceRevision"`
	TargetRevision    string         `json:"targetRevision"`
	Strategy          DeployStrategy `json:"strategy"`
	Analysis          []AnalysisRule `json:"analysis"`
	CreatedAt         time.Time      `json:"createdAt"`
	UpdatedAt         time.Time      `json:"updatedAt"`
	Status            IntentStatus   `json:"status"`
	CurrentStep       int            `json:"currentStep"`
	Message           string         `json:"message"`
}

// IntentStatus represents the status of a deployment intent
type IntentStatus string

const (
	IntentStatusPending    IntentStatus = "pending"
	IntentStatusRunning    IntentStatus = "running"
	IntentStatusPaused     IntentStatus = "paused"
	IntentStatusSucceeded  IntentStatus = "succeeded"
	IntentStatusFailed     IntentStatus = "failed"
	IntentStatusRolledBack IntentStatus = "rolled_back"
	IntentStatusCancelled  IntentStatus = "cancelled"
)

// IntentController manages intent-based deployments
type IntentController struct {
	provider       Provider
	stateBackend   StateBackend
	progressEngine ProgressiveDeliveryEngine
	metricsClient  MetricsClient
	notifier       IntentNotifier
	intents        map[string]*DeploymentIntent
	mu             sync.RWMutex
}

// StateBackend interface for state persistence
type StateBackend interface {
	Get(ctx context.Context, kind, id string) (interface{}, error)
	Put(ctx context.Context, kind, id string, value interface{}) error
	Delete(ctx context.Context, kind, id string) error
	List(ctx context.Context, kind string) ([]interface{}, error)
}

// ProgressiveDeliveryEngine interface for traffic shifting
type ProgressiveDeliveryEngine interface {
	ShiftTraffic(ctx context.Context, app string, weight int) error
	GetCurrentWeight(ctx context.Context, app string) (int, error)
	SetHeaderRouting(ctx context.Context, app, header string) error
	RemoveHeaderRouting(ctx context.Context, app string) error
}

// MetricsClient interface for querying metrics
type MetricsClient interface {
	Query(ctx context.Context, query string, window time.Duration) (*QueryResult, error)
	GetAverage(ctx context.Context, query string, window time.Duration) (*MetricValue, error)
	GetPercentile(ctx context.Context, query string, percentile float64, window time.Duration) (*MetricValue, error)
}

// QueryResult represents metrics query result
type QueryResult struct {
	Value     float64
	Timestamp time.Time
	Labels    map[string]string
}

// MetricValue represents a metric value with metadata
type MetricValue struct {
	Avg   float64
	Min   float64
	Max   float64
	P50   float64
	P95   float64
	P99   float64
	Count int64
}

// IntentNotifier interface for sending deployment notifications
type IntentNotifier interface {
	Send(ctx context.Context, channel string, message string) error
	SendDeploymentUpdate(ctx context.Context, intent *DeploymentIntent) error
}

// AnalysisResult represents the result of metric analysis
type AnalysisResult struct {
	Passed   bool
	Rule     string
	Message  string
	Value    float64
	Expected float64
	Error    error
}

// NewIntentController creates a new IntentController
func NewIntentController(provider Provider, stateBackend StateBackend, progressEngine ProgressiveDeliveryEngine, metricsClient MetricsClient, notifier IntentNotifier) *IntentController {
	return &IntentController{
		provider:       provider,
		stateBackend:   stateBackend,
		progressEngine: progressEngine,
		metricsClient:  metricsClient,
		notifier:       notifier,
		intents:        make(map[string]*DeploymentIntent),
	}
}

// CreateIntent creates a new deployment intent
func (c *IntentController) CreateIntent(ctx context.Context, intent *DeploymentIntent) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if intent.ID == "" {
		intent.ID = generateIntentID()
	}

	intent.CreatedAt = time.Now()
	intent.UpdatedAt = time.Now()
	intent.Status = IntentStatusPending
	intent.CurrentStep = 0

	c.intents[intent.ID] = intent

	if c.stateBackend != nil {
		if err := c.stateBackend.Put(ctx, "DeploymentIntent", intent.ID, intent); err != nil {
			return fmt.Errorf("failed to persist intent: %w", err)
		}
	}

	return nil
}

// ExecuteIntent executes a deployment intent
func (c *IntentController) ExecuteIntent(ctx context.Context, intentID string) error {
	c.mu.Lock()
	intent, ok := c.intents[intentID]
	if !ok {
		c.mu.Unlock()
		return fmt.Errorf("intent not found: %s", intentID)
	}
	intent.Status = IntentStatusRunning
	intent.UpdatedAt = time.Now()
	c.mu.Unlock()

	// Notify deployment started
	if c.notifier != nil {
		c.notifier.SendDeploymentUpdate(ctx, intent)
	}

	// Execute based on strategy type
	var err error
	switch intent.Strategy.Type {
	case "canary":
		err = c.executeCanaryDeployment(ctx, intent)
	case "blue-green":
		err = c.executeBlueGreenDeployment(ctx, intent)
	case "rolling":
		err = c.executeRollingDeployment(ctx, intent)
	case "recreate":
		err = c.executeRecreateDeployment(ctx, intent)
	default:
		err = fmt.Errorf("unknown strategy type: %s", intent.Strategy.Type)
	}

	if err != nil {
		return c.handleDeploymentFailure(ctx, intent, err)
	}

	c.mu.Lock()
	intent.Status = IntentStatusSucceeded
	intent.UpdatedAt = time.Now()
	c.mu.Unlock()

	if c.notifier != nil {
		c.notifier.SendDeploymentUpdate(ctx, intent)
	}

	return nil
}

// executeCanaryDeployment performs canary deployment with gradual traffic shift
func (c *IntentController) executeCanaryDeployment(ctx context.Context, intent *DeploymentIntent) error {
	for i, step := range intent.Strategy.Steps {
		c.mu.Lock()
		intent.CurrentStep = i
		intent.UpdatedAt = time.Now()
		c.mu.Unlock()

		// Shift traffic to new version
		weight := step.Weight
		if step.SetWeight > 0 {
			weight = step.SetWeight
		}

		if c.progressEngine != nil {
			if err := c.progressEngine.ShiftTraffic(ctx, intent.Application, weight); err != nil {
				return fmt.Errorf("failed to shift traffic at step %d: %w", i, err)
			}
		}

		// Set header-based routing if configured
		if step.SetHeader != "" && c.progressEngine != nil {
			if err := c.progressEngine.SetHeaderRouting(ctx, intent.Application, step.SetHeader); err != nil {
				return fmt.Errorf("failed to set header routing at step %d: %w", i, err)
			}
		}

		// Run analysis if enabled
		if step.Analysis {
			result := c.analyzeMetrics(ctx, intent.Analysis, step.WaitTime)
			if !result.Passed {
				return fmt.Errorf("analysis failed at step %d: %s", i, result.Message)
			}
		}

		// Handle pause configuration
		pauseConfig := step.Pause
		if pauseConfig == nil {
			pauseConfig = &intent.Strategy.Pause
		}

		if pauseConfig != nil && pauseConfig.RequireApproval {
			c.mu.Lock()
			intent.Status = IntentStatusPaused
			intent.Message = fmt.Sprintf("Waiting for approval at step %d (weight: %d%%)", i, weight)
			intent.UpdatedAt = time.Now()
			c.mu.Unlock()

			if c.notifier != nil {
				c.notifier.SendDeploymentUpdate(ctx, intent)
			}

			// Wait for approval (in real implementation, this would be async)
			return fmt.Errorf("deployment paused for approval at step %d", i)
		}

		// Wait before proceeding to next step
		if step.WaitTime > 0 {
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(step.WaitTime):
			}
		}
	}

	// Final step: 100% traffic to new version
	if c.progressEngine != nil {
		if err := c.progressEngine.ShiftTraffic(ctx, intent.Application, 100); err != nil {
			return fmt.Errorf("failed to complete traffic shift: %w", err)
		}
	}

	return nil
}

// executeBlueGreenDeployment performs blue-green deployment
func (c *IntentController) executeBlueGreenDeployment(ctx context.Context, intent *DeploymentIntent) error {
	// Blue-green is essentially a two-step canary: 0% -> 100%
	// 1. Deploy new version (green) alongside old (blue)
	// 2. Run analysis on green
	// 3. Switch traffic atomically

	c.mu.Lock()
	intent.CurrentStep = 0
	intent.Message = "Deploying green version"
	intent.UpdatedAt = time.Now()
	c.mu.Unlock()

	// Run analysis on green before switch (using header-based routing)
	if c.progressEngine != nil {
		if err := c.progressEngine.SetHeaderRouting(ctx, intent.Application, "X-Canary: true"); err != nil {
			return fmt.Errorf("failed to set header routing for analysis: %w", err)
		}
	}

	// Analyze green version
	if len(intent.Analysis) > 0 {
		waitTime := 5 * time.Minute
		if len(intent.Strategy.Steps) > 0 && intent.Strategy.Steps[0].WaitTime > 0 {
			waitTime = intent.Strategy.Steps[0].WaitTime
		}

		result := c.analyzeMetrics(ctx, intent.Analysis, waitTime)
		if !result.Passed {
			return fmt.Errorf("green version analysis failed: %s", result.Message)
		}
	}

	c.mu.Lock()
	intent.CurrentStep = 1
	intent.Message = "Switching traffic to green"
	intent.UpdatedAt = time.Now()
	c.mu.Unlock()

	// Atomic switch to green
	if c.progressEngine != nil {
		if err := c.progressEngine.RemoveHeaderRouting(ctx, intent.Application); err != nil {
			return fmt.Errorf("failed to remove header routing: %w", err)
		}
		if err := c.progressEngine.ShiftTraffic(ctx, intent.Application, 100); err != nil {
			return fmt.Errorf("failed to switch traffic to green: %w", err)
		}
	}

	return nil
}

// executeRollingDeployment performs rolling update
func (c *IntentController) executeRollingDeployment(ctx context.Context, intent *DeploymentIntent) error {
	// Rolling updates incrementally replace instances
	// Default: 25% increments
	steps := intent.Strategy.Steps
	if len(steps) == 0 {
		steps = []ProgressStep{
			{Weight: 25, WaitTime: 30 * time.Second, Analysis: true},
			{Weight: 50, WaitTime: 30 * time.Second, Analysis: true},
			{Weight: 75, WaitTime: 30 * time.Second, Analysis: true},
			{Weight: 100, WaitTime: 0, Analysis: false},
		}
	}

	for i, step := range steps {
		c.mu.Lock()
		intent.CurrentStep = i
		intent.Message = fmt.Sprintf("Rolling update: %d%% complete", step.Weight)
		intent.UpdatedAt = time.Now()
		c.mu.Unlock()

		if c.progressEngine != nil {
			if err := c.progressEngine.ShiftTraffic(ctx, intent.Application, step.Weight); err != nil {
				return fmt.Errorf("failed to update at %d%%: %w", step.Weight, err)
			}
		}

		if step.Analysis {
			result := c.analyzeMetrics(ctx, intent.Analysis, step.WaitTime)
			if !result.Passed {
				return fmt.Errorf("rolling update failed at %d%%: %s", step.Weight, result.Message)
			}
		}

		if step.WaitTime > 0 {
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(step.WaitTime):
			}
		}
	}

	return nil
}

// executeRecreateDeployment performs recreate deployment (downtime)
func (c *IntentController) executeRecreateDeployment(ctx context.Context, intent *DeploymentIntent) error {
	c.mu.Lock()
	intent.CurrentStep = 0
	intent.Message = "Terminating old version"
	intent.UpdatedAt = time.Now()
	c.mu.Unlock()

	// Scale down old version
	if c.progressEngine != nil {
		if err := c.progressEngine.ShiftTraffic(ctx, intent.Application, 0); err != nil {
			return fmt.Errorf("failed to terminate old version: %w", err)
		}
	}

	c.mu.Lock()
	intent.CurrentStep = 1
	intent.Message = "Deploying new version"
	intent.UpdatedAt = time.Now()
	c.mu.Unlock()

	// Deploy new version
	if c.progressEngine != nil {
		if err := c.progressEngine.ShiftTraffic(ctx, intent.Application, 100); err != nil {
			return fmt.Errorf("failed to deploy new version: %w", err)
		}
	}

	return nil
}

// analyzeMetrics runs analysis rules against metrics
func (c *IntentController) analyzeMetrics(ctx context.Context, rules []AnalysisRule, window time.Duration) *AnalysisResult {
	if c.metricsClient == nil || len(rules) == 0 {
		return &AnalysisResult{Passed: true}
	}

	for _, rule := range rules {
		result, err := c.metricsClient.Query(ctx, rule.Query, window)
		if err != nil {
			return &AnalysisResult{
				Passed:  false,
				Rule:    rule.Name,
				Message: fmt.Sprintf("failed to query metrics: %v", err),
				Error:   err,
			}
		}

		passed := c.evaluateRule(result.Value, rule.Threshold, rule.Operator)
		if !passed {
			return &AnalysisResult{
				Passed:   false,
				Rule:     rule.Name,
				Message:  fmt.Sprintf("metric %s: value %.2f %s threshold %.2f", rule.MetricName, result.Value, rule.Operator, rule.Threshold),
				Value:    result.Value,
				Expected: rule.Threshold,
			}
		}
	}

	return &AnalysisResult{Passed: true}
}

// evaluateRule evaluates a metric value against a threshold
func (c *IntentController) evaluateRule(value, threshold float64, operator string) bool {
	switch operator {
	case "lt", "<":
		return value < threshold
	case "lte", "<=":
		return value <= threshold
	case "gt", ">":
		return value > threshold
	case "gte", ">=":
		return value >= threshold
	case "eq", "==":
		return value == threshold
	default:
		return false
	}
}

// handleDeploymentFailure handles deployment failure based on rollback config
func (c *IntentController) handleDeploymentFailure(ctx context.Context, intent *DeploymentIntent, deployErr error) error {
	c.mu.Lock()
	intent.Status = IntentStatusFailed
	intent.Message = deployErr.Error()
	intent.UpdatedAt = time.Now()
	c.mu.Unlock()

	rollbackConfig := intent.Strategy.Rollback
	switch rollbackConfig.OnFailure {
	case "rollback":
		if rollbackConfig.Automatic {
			if err := c.rollback(ctx, intent); err != nil {
				return fmt.Errorf("deployment failed and rollback failed: %w (original: %v)", err, deployErr)
			}
			c.mu.Lock()
			intent.Status = IntentStatusRolledBack
			intent.Message = fmt.Sprintf("Rolled back due to: %s", deployErr.Error())
			intent.UpdatedAt = time.Now()
			c.mu.Unlock()
		}
	case "pause":
		c.mu.Lock()
		intent.Status = IntentStatusPaused
		intent.Message = fmt.Sprintf("Paused due to failure: %s", deployErr.Error())
		intent.UpdatedAt = time.Now()
		c.mu.Unlock()
	case "abort":
		// Already set to failed above
	}

	if c.notifier != nil {
		c.notifier.SendDeploymentUpdate(ctx, intent)
	}

	return deployErr
}

// rollback performs a rollback to the previous version
func (c *IntentController) rollback(ctx context.Context, intent *DeploymentIntent) error {
	if c.progressEngine == nil {
		return nil
	}

	// Shift all traffic back to old version
	return c.progressEngine.ShiftTraffic(ctx, intent.Application, 0)
}

// GetIntent returns an intent by ID
func (c *IntentController) GetIntent(ctx context.Context, intentID string) (*DeploymentIntent, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	intent, ok := c.intents[intentID]
	if !ok {
		return nil, fmt.Errorf("intent not found: %s", intentID)
	}

	return intent, nil
}

// ListIntents returns all intents
func (c *IntentController) ListIntents(ctx context.Context) []*DeploymentIntent {
	c.mu.RLock()
	defer c.mu.RUnlock()

	intents := make([]*DeploymentIntent, 0, len(c.intents))
	for _, intent := range c.intents {
		intents = append(intents, intent)
	}

	return intents
}

// CancelIntent cancels a running intent
func (c *IntentController) CancelIntent(ctx context.Context, intentID string) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	intent, ok := c.intents[intentID]
	if !ok {
		return fmt.Errorf("intent not found: %s", intentID)
	}

	if intent.Status != IntentStatusRunning && intent.Status != IntentStatusPaused {
		return fmt.Errorf("cannot cancel intent in status: %s", intent.Status)
	}

	intent.Status = IntentStatusCancelled
	intent.Message = "Cancelled by user"
	intent.UpdatedAt = time.Now()

	return nil
}

// ApproveIntent approves a paused intent to continue
func (c *IntentController) ApproveIntent(ctx context.Context, intentID string) error {
	c.mu.Lock()
	intent, ok := c.intents[intentID]
	if !ok {
		c.mu.Unlock()
		return fmt.Errorf("intent not found: %s", intentID)
	}

	if intent.Status != IntentStatusPaused {
		c.mu.Unlock()
		return fmt.Errorf("intent is not paused: %s", intent.Status)
	}

	intent.Status = IntentStatusRunning
	intent.Message = "Approved, continuing deployment"
	intent.UpdatedAt = time.Now()
	c.mu.Unlock()

	// Continue execution from current step
	return c.ExecuteIntent(ctx, intentID)
}

// DeleteIntent deletes an intent
func (c *IntentController) DeleteIntent(ctx context.Context, intentID string) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if _, ok := c.intents[intentID]; !ok {
		return fmt.Errorf("intent not found: %s", intentID)
	}

	delete(c.intents, intentID)

	if c.stateBackend != nil {
		return c.stateBackend.Delete(ctx, "DeploymentIntent", intentID)
	}

	return nil
}

// generateIntentID generates a unique intent ID
func generateIntentID() string {
	return fmt.Sprintf("intent-%d", time.Now().UnixNano())
}

// DefaultCanaryStrategy returns a default canary deployment strategy
func DefaultCanaryStrategy() DeployStrategy {
	return DeployStrategy{
		Type: "canary",
		Steps: []ProgressStep{
			{Weight: 10, WaitTime: 5 * time.Minute, Analysis: true},
			{Weight: 25, WaitTime: 5 * time.Minute, Analysis: true},
			{Weight: 50, WaitTime: 10 * time.Minute, Analysis: true},
			{Weight: 75, WaitTime: 10 * time.Minute, Analysis: true},
			{Weight: 100, WaitTime: 0, Analysis: false},
		},
		Rollback: RollbackConfig{
			Automatic:  true,
			MaxRetries: 3,
			OnFailure:  "rollback",
		},
	}
}

// DefaultBlueGreenStrategy returns a default blue-green deployment strategy
func DefaultBlueGreenStrategy() DeployStrategy {
	return DeployStrategy{
		Type: "blue-green",
		Pause: PauseConfig{
			RequireApproval: true,
		},
		Rollback: RollbackConfig{
			Automatic:  true,
			MaxRetries: 1,
			OnFailure:  "rollback",
		},
	}
}

// DefaultRollingStrategy returns a default rolling update strategy
func DefaultRollingStrategy() DeployStrategy {
	return DeployStrategy{
		Type: "rolling",
		Steps: []ProgressStep{
			{Weight: 25, WaitTime: 30 * time.Second, Analysis: true},
			{Weight: 50, WaitTime: 30 * time.Second, Analysis: true},
			{Weight: 75, WaitTime: 30 * time.Second, Analysis: true},
			{Weight: 100, WaitTime: 0, Analysis: false},
		},
		Rollback: RollbackConfig{
			Automatic:  true,
			MaxRetries: 3,
			OnFailure:  "rollback",
		},
	}
}

// CommonAnalysisRules returns common analysis rules for deployments
func CommonAnalysisRules(app string) []AnalysisRule {
	return []AnalysisRule{
		{
			Name:         "error-rate",
			MetricName:   "http_requests_errors_total",
			Query:        fmt.Sprintf(`sum(rate(http_requests_total{app="%s",status=~"5.."}[5m])) / sum(rate(http_requests_total{app="%s"}[5m])) * 100`, app, app),
			Threshold:    1.0, // 1% error rate
			Operator:     "lt",
			FailureLimit: 3,
		},
		{
			Name:         "latency-p99",
			MetricName:   "http_request_duration_seconds",
			Query:        fmt.Sprintf(`histogram_quantile(0.99, sum(rate(http_request_duration_seconds_bucket{app="%s"}[5m])) by (le))`, app),
			Threshold:    0.5, // 500ms
			Operator:     "lt",
			FailureLimit: 3,
		},
		{
			Name:         "success-rate",
			MetricName:   "http_requests_success_total",
			Query:        fmt.Sprintf(`sum(rate(http_requests_total{app="%s",status=~"2.."}[5m])) / sum(rate(http_requests_total{app="%s"}[5m])) * 100`, app, app),
			Threshold:    99.0, // 99% success rate
			Operator:     "gte",
			FailureLimit: 3,
		},
	}
}
