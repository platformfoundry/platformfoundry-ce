package scaling

import (
	"context"
	"fmt"
	"math"
	"sync"
	"time"
)

// Scaler interface for executing scaling operations
type Scaler interface {
	// Scale changes the replica count
	Scale(ctx context.Context, target ScalingTarget, replicas int) error
	// GetCurrentReplicas gets current replica count
	GetCurrentReplicas(ctx context.Context, target ScalingTarget) (int, error)
	// GetResourceCost estimates cost per replica per hour
	GetResourceCost(ctx context.Context, target ScalingTarget) (float64, error)
}

// EventRecorder interface for recording scaling events
type EventRecorder interface {
	Record(event ScalingEvent)
	GetHistory(policyName string, limit int) []ScalingEvent
}

// Engine orchestrates predictive scaling
type Engine struct {
	config      *EngineConfig
	predictor   *Predictor
	scaler      Scaler
	metrics     MetricsProvider
	recorder    EventRecorder
	policies    map[string]*ScalingPolicy
	stopCh      map[string]chan struct{}
	mu          sync.RWMutex
	running     bool
}

// EngineConfig configures the scaling engine
type EngineConfig struct {
	EvaluationInterval time.Duration `json:"evaluationInterval"`
	CooldownPeriod     time.Duration `json:"cooldownPeriod"`
	DryRun             bool          `json:"dryRun"`
	EnablePredictive   bool          `json:"enablePredictive"`
	EnableCostAware    bool          `json:"enableCostAware"`
}

// DefaultEngineConfig returns default engine configuration
func DefaultEngineConfig() *EngineConfig {
	return &EngineConfig{
		EvaluationInterval: 30 * time.Second,
		CooldownPeriod:     3 * time.Minute,
		DryRun:             false,
		EnablePredictive:   true,
		EnableCostAware:    true,
	}
}

// NewEngine creates a new scaling engine
func NewEngine(config *EngineConfig, predictor *Predictor, scaler Scaler, metrics MetricsProvider) *Engine {
	if config == nil {
		config = DefaultEngineConfig()
	}
	return &Engine{
		config:    config,
		predictor: predictor,
		scaler:    scaler,
		metrics:   metrics,
		policies:  make(map[string]*ScalingPolicy),
		stopCh:    make(map[string]chan struct{}),
	}
}

// WithEventRecorder sets the event recorder
func (e *Engine) WithEventRecorder(recorder EventRecorder) *Engine {
	e.recorder = recorder
	return e
}

// RegisterPolicy registers a scaling policy
func (e *Engine) RegisterPolicy(policy *ScalingPolicy) error {
	if policy.Metadata.Name == "" {
		return fmt.Errorf("policy name is required")
	}

	e.mu.Lock()
	defer e.mu.Unlock()

	e.policies[policy.Metadata.Name] = policy

	// Initialize status
	if policy.Status == nil {
		policy.Status = &ScalingPolicyStatus{
			Conditions: []PolicyCondition{
				{
					Type:               "Ready",
					Status:             "Unknown",
					LastTransitionTime: time.Now(),
					Reason:             "Initializing",
				},
			},
		}
	}

	return nil
}

// UnregisterPolicy removes a scaling policy
func (e *Engine) UnregisterPolicy(name string) {
	e.mu.Lock()
	defer e.mu.Unlock()

	// Stop evaluation loop if running
	if stopCh, ok := e.stopCh[name]; ok {
		close(stopCh)
		delete(e.stopCh, name)
	}

	delete(e.policies, name)
}

// GetPolicy returns a policy by name
func (e *Engine) GetPolicy(name string) (*ScalingPolicy, bool) {
	e.mu.RLock()
	defer e.mu.RUnlock()
	policy, ok := e.policies[name]
	return policy, ok
}

// ListPolicies returns all registered policies
func (e *Engine) ListPolicies() []*ScalingPolicy {
	e.mu.RLock()
	defer e.mu.RUnlock()

	policies := make([]*ScalingPolicy, 0, len(e.policies))
	for _, p := range e.policies {
		policies = append(policies, p)
	}
	return policies
}

// Start starts the scaling engine for all policies
func (e *Engine) Start(ctx context.Context) error {
	e.mu.Lock()
	if e.running {
		e.mu.Unlock()
		return fmt.Errorf("engine already running")
	}
	e.running = true
	e.mu.Unlock()

	// Start evaluation loop for each policy
	for name, policy := range e.policies {
		stopCh := make(chan struct{})
		e.stopCh[name] = stopCh
		go e.evaluateLoop(ctx, policy, stopCh)
	}

	return nil
}

// Stop stops the scaling engine
func (e *Engine) Stop() {
	e.mu.Lock()
	defer e.mu.Unlock()

	for name, stopCh := range e.stopCh {
		close(stopCh)
		delete(e.stopCh, name)
	}
	e.running = false
}

// evaluateLoop runs the evaluation loop for a policy
func (e *Engine) evaluateLoop(ctx context.Context, policy *ScalingPolicy, stopCh chan struct{}) {
	ticker := time.NewTicker(e.config.EvaluationInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-stopCh:
			return
		case <-ticker.C:
			if err := e.Evaluate(ctx, policy.Metadata.Name); err != nil {
				e.updatePolicyCondition(policy, "Error", "False", "EvaluationFailed", err.Error())
			}
		}
	}
}

// Evaluate evaluates a policy and scales if necessary
func (e *Engine) Evaluate(ctx context.Context, policyName string) error {
	e.mu.RLock()
	policy, ok := e.policies[policyName]
	e.mu.RUnlock()

	if !ok {
		return fmt.Errorf("policy %s not found", policyName)
	}

	// Get current state
	currentReplicas, err := e.scaler.GetCurrentReplicas(ctx, policy.Spec.Target)
	if err != nil {
		return fmt.Errorf("failed to get current replicas: %w", err)
	}

	policy.Status.CurrentReplicas = currentReplicas

	// Collect current metrics
	metricsValues, err := e.collectMetrics(ctx, policy)
	if err != nil {
		return fmt.Errorf("failed to collect metrics: %w", err)
	}
	policy.Status.CurrentMetrics = metricsValues

	// Calculate desired replicas
	desiredReplicas, reason, err := e.calculateDesiredReplicas(ctx, policy, currentReplicas, metricsValues)
	if err != nil {
		return fmt.Errorf("failed to calculate desired replicas: %w", err)
	}

	policy.Status.DesiredReplicas = desiredReplicas

	// Apply constraints
	desiredReplicas = e.applyConstraints(policy, desiredReplicas)

	// Check cost policy
	if e.config.EnableCostAware && policy.Spec.CostPolicy != nil {
		desiredReplicas, err = e.applyCostConstraints(ctx, policy, desiredReplicas)
		if err != nil {
			// Log but don't fail
			e.updatePolicyCondition(policy, "CostCheck", "False", "CostCheckFailed", err.Error())
		}
	}

	// Check cooldown
	if policy.Status.LastScaleTime != nil {
		cooldown := e.config.CooldownPeriod
		if time.Since(*policy.Status.LastScaleTime) < cooldown {
			return nil // Still in cooldown
		}
	}

	// Scale if needed
	if desiredReplicas != currentReplicas {
		if err := e.executeScale(ctx, policy, currentReplicas, desiredReplicas, reason); err != nil {
			return err
		}
	}

	e.updatePolicyCondition(policy, "Ready", "True", "ScalingActive", "Policy is actively scaling")
	return nil
}

// collectMetrics collects current metric values
func (e *Engine) collectMetrics(ctx context.Context, policy *ScalingPolicy) ([]MetricValue, error) {
	if e.metrics == nil {
		return nil, nil
	}

	values := make([]MetricValue, 0, len(policy.Spec.Metrics))
	for _, metric := range policy.Spec.Metrics {
		current, err := e.metrics.GetCurrent(ctx, metric.Query)
		if err != nil {
			continue // Skip failed metrics
		}

		values = append(values, MetricValue{
			Name:         metric.Name,
			CurrentValue: current,
			TargetValue:  metric.Target.Value,
			Timestamp:    time.Now(),
		})
	}

	return values, nil
}

// calculateDesiredReplicas determines the desired replica count
func (e *Engine) calculateDesiredReplicas(ctx context.Context, policy *ScalingPolicy, current int, metrics []MetricValue) (int, string, error) {
	var desiredReplicas int
	var reason string

	switch policy.Spec.Strategy {
	case StrategyPredictive:
		if e.config.EnablePredictive && policy.Spec.Predictions != nil && policy.Spec.Predictions.Enabled {
			return e.calculatePredictiveReplicas(ctx, policy, current)
		}
		fallthrough // Fall back to reactive

	case StrategyReactive:
		desiredReplicas, reason = e.calculateReactiveReplicas(policy, current, metrics)

	case StrategyScheduled:
		desiredReplicas, reason = e.calculateScheduledReplicas(policy, current)

	case StrategyHybrid:
		// Combine predictive and reactive
		predictive, predReason, _ := e.calculatePredictiveReplicas(ctx, policy, current)
		reactive, _ := e.calculateReactiveReplicas(policy, current, metrics)
		desiredReplicas = int(math.Max(float64(predictive), float64(reactive)))
		reason = fmt.Sprintf("hybrid: max(predictive=%d [%s], reactive=%d)", predictive, predReason, reactive)

	default:
		desiredReplicas, reason = e.calculateReactiveReplicas(policy, current, metrics)
	}

	return desiredReplicas, reason, nil
}

// calculatePredictiveReplicas uses predictions to determine replicas
func (e *Engine) calculatePredictiveReplicas(ctx context.Context, policy *ScalingPolicy, current int) (int, string, error) {
	if e.predictor == nil {
		return current, "no predictor", fmt.Errorf("predictor not configured")
	}

	horizon, err := time.ParseDuration(policy.Spec.Predictions.Horizon)
	if err != nil {
		horizon = 1 * time.Hour
	}

	prediction, err := e.predictor.Predict(ctx, policy.Metadata.Name, horizon)
	if err != nil {
		return current, "prediction failed", err
	}

	policy.Status.LastPrediction = prediction

	// Check confidence threshold
	if prediction.ConfidenceLevel < policy.Spec.Predictions.ConfidenceLevel {
		return current, "low confidence", nil
	}

	// Calculate replicas based on predicted load
	desiredReplicas := prediction.RecommendedPods
	if desiredReplicas == 0 {
		// Estimate based on predicted load and current metrics
		if len(policy.Status.CurrentMetrics) > 0 {
			currentLoad := policy.Status.CurrentMetrics[0].CurrentValue
			targetLoad := policy.Status.CurrentMetrics[0].TargetValue
			if currentLoad > 0 && targetLoad > 0 {
				ratio := prediction.PredictedLoad / currentLoad
				desiredReplicas = int(math.Ceil(float64(current) * ratio))
			}
		}
	}

	// Apply buffer
	if policy.Spec.Predictions.BufferPercentage > 0 {
		buffer := float64(desiredReplicas) * policy.Spec.Predictions.BufferPercentage / 100
		desiredReplicas += int(math.Ceil(buffer))
	}

	if desiredReplicas < 1 {
		desiredReplicas = 1
	}

	reason := fmt.Sprintf("predicted load %.2f (confidence %.2f%%)", prediction.PredictedLoad, prediction.ConfidenceLevel*100)
	return desiredReplicas, reason, nil
}

// calculateReactiveReplicas uses current metrics to determine replicas
func (e *Engine) calculateReactiveReplicas(policy *ScalingPolicy, current int, metrics []MetricValue) (int, string) {
	if len(metrics) == 0 {
		return current, "no metrics"
	}

	var totalRatio float64
	var totalWeight float64

	for i, metric := range policy.Spec.Metrics {
		if i >= len(metrics) {
			break
		}

		weight := metric.Weight
		if weight == 0 {
			weight = 1.0
		}

		target := metric.Target.Value
		if metric.Target.AverageUtilization > 0 {
			target = float64(metric.Target.AverageUtilization)
		}

		if target > 0 {
			ratio := metrics[i].CurrentValue / target
			totalRatio += ratio * weight
			totalWeight += weight
		}
	}

	if totalWeight == 0 {
		return current, "no valid metrics"
	}

	avgRatio := totalRatio / totalWeight
	desiredReplicas := int(math.Ceil(float64(current) * avgRatio))

	if desiredReplicas < 1 {
		desiredReplicas = 1
	}

	reason := fmt.Sprintf("metric ratio %.2f", avgRatio)
	return desiredReplicas, reason
}

// calculateScheduledReplicas uses time-based schedules
func (e *Engine) calculateScheduledReplicas(policy *ScalingPolicy, current int) (int, string) {
	if len(policy.Spec.Schedule) == 0 {
		return current, "no schedule"
	}

	now := time.Now()
	for _, schedule := range policy.Spec.Schedule {
		if matchesCron(schedule.Cron, now, schedule.Timezone) {
			if schedule.Replicas > 0 {
				return schedule.Replicas, fmt.Sprintf("schedule: %s", schedule.Name)
			}
		}
	}

	return current, "no matching schedule"
}

// applyConstraints applies min/max constraints
func (e *Engine) applyConstraints(policy *ScalingPolicy, desired int) int {
	if desired < policy.Spec.Constraints.MinReplicas {
		return policy.Spec.Constraints.MinReplicas
	}
	if desired > policy.Spec.Constraints.MaxReplicas {
		return policy.Spec.Constraints.MaxReplicas
	}
	return desired
}

// applyCostConstraints applies cost-based constraints
func (e *Engine) applyCostConstraints(ctx context.Context, policy *ScalingPolicy, desired int) (int, error) {
	if e.scaler == nil {
		return desired, nil
	}

	costPerReplica, err := e.scaler.GetResourceCost(ctx, policy.Spec.Target)
	if err != nil {
		return desired, err
	}

	hourlyCost := costPerReplica * float64(desired)
	policy.Status.CostEstimate = &CostEstimate{
		CurrentHourlyCost:   costPerReplica * float64(policy.Status.CurrentReplicas),
		ProjectedHourlyCost: hourlyCost,
		DailyEstimate:       hourlyCost * 24,
		MonthlyEstimate:     hourlyCost * 24 * 30,
	}

	// Check max hourly cost
	if policy.Spec.CostPolicy.MaxHourlyCost > 0 && hourlyCost > policy.Spec.CostPolicy.MaxHourlyCost {
		maxReplicas := int(policy.Spec.CostPolicy.MaxHourlyCost / costPerReplica)
		if maxReplicas < policy.Spec.Constraints.MinReplicas {
			maxReplicas = policy.Spec.Constraints.MinReplicas
		}
		return maxReplicas, nil
	}

	return desired, nil
}

// executeScale performs the scaling operation
func (e *Engine) executeScale(ctx context.Context, policy *ScalingPolicy, from, to int, reason string) error {
	start := time.Now()

	direction := DirectionNone
	if to > from {
		direction = DirectionUp
	} else if to < from {
		direction = DirectionDown
	}

	event := ScalingEvent{
		ID:              fmt.Sprintf("scale-%d", time.Now().UnixNano()),
		PolicyName:      policy.Metadata.Name,
		Timestamp:       start,
		Direction:       direction,
		FromReplicas:    from,
		ToReplicas:      to,
		Reason:          reason,
		Trigger:         string(policy.Spec.Strategy),
		MetricsSnapshot: policy.Status.CurrentMetrics,
	}

	if e.config.DryRun {
		event.Success = true
		event.Duration = time.Since(start)
		if e.recorder != nil {
			e.recorder.Record(event)
		}
		return nil
	}

	err := e.scaler.Scale(ctx, policy.Spec.Target, to)
	event.Duration = time.Since(start)

	if err != nil {
		event.Success = false
		event.Error = err.Error()
		if e.recorder != nil {
			e.recorder.Record(event)
		}
		return err
	}

	event.Success = true
	now := time.Now()
	policy.Status.LastScaleTime = &now
	policy.Status.CurrentReplicas = to

	if e.recorder != nil {
		e.recorder.Record(event)
	}

	return nil
}

// updatePolicyCondition updates a policy condition
func (e *Engine) updatePolicyCondition(policy *ScalingPolicy, condType, status, reason, message string) {
	now := time.Now()

	for i, cond := range policy.Status.Conditions {
		if cond.Type == condType {
			if cond.Status != status {
				policy.Status.Conditions[i].LastTransitionTime = now
			}
			policy.Status.Conditions[i].Status = status
			policy.Status.Conditions[i].Reason = reason
			policy.Status.Conditions[i].Message = message
			return
		}
	}

	// Add new condition
	policy.Status.Conditions = append(policy.Status.Conditions, PolicyCondition{
		Type:               condType,
		Status:             status,
		LastTransitionTime: now,
		Reason:             reason,
		Message:            message,
	})
}

// TrainModels trains prediction models for all policies
func (e *Engine) TrainModels(ctx context.Context) error {
	if e.predictor == nil {
		return fmt.Errorf("predictor not configured")
	}

	e.mu.RLock()
	policies := make([]*ScalingPolicy, 0, len(e.policies))
	for _, p := range e.policies {
		policies = append(policies, p)
	}
	e.mu.RUnlock()

	for _, policy := range policies {
		if policy.Spec.Predictions == nil || !policy.Spec.Predictions.Enabled {
			continue
		}

		// Use first metric for training
		if len(policy.Spec.Metrics) == 0 {
			continue
		}

		metric := policy.Spec.Metrics[0]
		_, err := e.predictor.Train(ctx, policy.Metadata.Name, metric.Query, policy.Spec.Predictions.Model)
		if err != nil {
			e.updatePolicyCondition(policy, "ModelTrained", "False", "TrainingFailed", err.Error())
		} else {
			e.updatePolicyCondition(policy, "ModelTrained", "True", "TrainingSucceeded", "Model trained successfully")
		}
	}

	return nil
}

// GetScalingHistory returns scaling history for a policy
func (e *Engine) GetScalingHistory(policyName string, limit int) []ScalingEvent {
	if e.recorder == nil {
		return nil
	}
	return e.recorder.GetHistory(policyName, limit)
}

// matchesCron checks if current time matches a cron expression (simplified)
func matchesCron(cronExpr string, t time.Time, timezone string) bool {
	// Simplified cron matching - in production use robfig/cron
	// This is a placeholder implementation
	return false
}
