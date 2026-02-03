package chaos

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// Manager handles chaos engineering operations
type Manager struct {
	experiments map[string]*ChaosExperiment
	executions  map[string]*ChaosExperimentStatus
	gameDays    map[string]*GameDay
	templates   map[string]*ExperimentTemplate
	mu          sync.RWMutex
}

// NewManager creates a new chaos engineering manager
func NewManager() *Manager {
	m := &Manager{
		experiments: make(map[string]*ChaosExperiment),
		executions:  make(map[string]*ChaosExperimentStatus),
		gameDays:    make(map[string]*GameDay),
		templates:   make(map[string]*ExperimentTemplate),
	}

	// Initialize built-in templates
	m.initializeTemplates()

	return m
}

// initializeTemplates creates built-in experiment templates
func (m *Manager) initializeTemplates() {
	m.templates["pod-failure-basic"] = &ExperimentTemplate{
		Name:        "pod-failure-basic",
		Description: "Kill random pods to test resilience",
		Category:    "pod",
		Spec: ChaosExperimentSpec{
			Target: ExperimentTarget{
				Kind:       "Deployment",
				Percentage: 30,
			},
			Experiments: []ExperimentAction{
				{
					Type:     ExperimentTypePodFailure,
					Duration: "5m",
				},
			},
			SteadyState: []SteadyStateCheck{
				{
					Name:      "error-rate",
					Metric:    "error_rate",
					Threshold: "< 1%",
				},
			},
		},
	}

	m.templates["network-latency-test"] = &ExperimentTemplate{
		Name:        "network-latency-test",
		Description: "Add network latency to test timeout handling",
		Category:    "network",
		Spec: ChaosExperimentSpec{
			Target: ExperimentTarget{
				Kind: "Service",
			},
			Experiments: []ExperimentAction{
				{
					Type:     ExperimentTypeNetworkLatency,
					Duration: "10m",
					Parameters: map[string]interface{}{
						"latency":    "500ms",
						"jitter":     "100ms",
						"percentage": 50,
					},
				},
			},
			SteadyState: []SteadyStateCheck{
				{
					Name:      "latency-p99",
					Metric:    "latency_p99",
					Threshold: "< 2s",
				},
			},
		},
	}

	m.templates["cpu-stress-test"] = &ExperimentTemplate{
		Name:        "cpu-stress-test",
		Description: "Stress CPU to test auto-scaling",
		Category:    "resource",
		Spec: ChaosExperimentSpec{
			Target: ExperimentTarget{
				Kind: "Deployment",
			},
			Experiments: []ExperimentAction{
				{
					Type:     ExperimentTypeCPUStress,
					Duration: "5m",
					Parameters: map[string]interface{}{
						"workers": 4,
						"load":    80,
					},
				},
			},
			SteadyState: []SteadyStateCheck{
				{
					Name:      "response-time",
					Metric:    "response_time_avg",
					Threshold: "< 500ms",
				},
			},
		},
	}

	m.templates["memory-stress-test"] = &ExperimentTemplate{
		Name:        "memory-stress-test",
		Description: "Stress memory to test OOM handling",
		Category:    "resource",
		Spec: ChaosExperimentSpec{
			Target: ExperimentTarget{
				Kind: "Deployment",
			},
			Experiments: []ExperimentAction{
				{
					Type:     ExperimentTypeMemoryStress,
					Duration: "5m",
					Parameters: map[string]interface{}{
						"percentage": 80,
					},
				},
			},
		},
	}

	m.templates["dns-failure-test"] = &ExperimentTemplate{
		Name:        "dns-failure-test",
		Description: "Cause DNS failures to test fallback behavior",
		Category:    "network",
		Spec: ChaosExperimentSpec{
			Target: ExperimentTarget{
				Kind: "Pod",
			},
			Experiments: []ExperimentAction{
				{
					Type:     ExperimentTypeDNSFailure,
					Duration: "3m",
					Parameters: map[string]interface{}{
						"hosts": []string{"external-api.com"},
					},
				},
			},
		},
	}
}

// RegisterExperiment registers a chaos experiment
func (m *Manager) RegisterExperiment(ctx context.Context, exp *ChaosExperiment) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if exp.Metadata.Name == "" {
		return fmt.Errorf("experiment name is required")
	}

	if exp.APIVersion == "" {
		exp.APIVersion = "platformfoundry.io/v1"
	}
	if exp.Kind == "" {
		exp.Kind = "ChaosExperiment"
	}

	exp.Metadata.CreatedAt = time.Now()

	// Initialize status
	exp.Status = &ChaosExperimentStatus{
		Phase: ExperimentPhasePending,
		Conditions: []ExperimentCondition{
			{
				Type:               "Registered",
				Status:             "True",
				LastTransitionTime: time.Now(),
				Reason:             "ExperimentCreated",
				Message:            "Experiment registered and ready to run",
			},
		},
	}

	m.experiments[exp.Metadata.Name] = exp
	return nil
}

// GetExperiment retrieves an experiment by name
func (m *Manager) GetExperiment(name string) (*ChaosExperiment, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	exp, ok := m.experiments[name]
	if !ok {
		return nil, fmt.Errorf("experiment not found: %s", name)
	}
	return exp, nil
}

// ListExperiments returns all experiments
func (m *Manager) ListExperiments() []*ChaosExperiment {
	m.mu.RLock()
	defer m.mu.RUnlock()

	result := make([]*ChaosExperiment, 0, len(m.experiments))
	for _, exp := range m.experiments {
		result = append(result, exp)
	}
	return result
}

// DeleteExperiment removes an experiment
func (m *Manager) DeleteExperiment(name string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, ok := m.experiments[name]; !ok {
		return fmt.Errorf("experiment not found: %s", name)
	}

	delete(m.experiments, name)
	return nil
}

// RunExperiment executes a chaos experiment
func (m *Manager) RunExperiment(ctx context.Context, name string) error {
	m.mu.Lock()
	exp, ok := m.experiments[name]
	if !ok {
		m.mu.Unlock()
		return fmt.Errorf("experiment not found: %s", name)
	}

	// Update status
	now := time.Now()
	exp.Status.Phase = ExperimentPhaseRunning
	exp.Status.StartedAt = &now
	exp.Status.ExperimentResults = make([]ExperimentResult, 0)
	exp.Status.SteadyStateResults = make([]SteadyStateResult, 0)
	m.mu.Unlock()

	// Run steady state check before
	if len(exp.Spec.SteadyState) > 0 {
		results := m.checkSteadyState(ctx, exp)
		m.mu.Lock()
		exp.Status.SteadyStateResults = append(exp.Status.SteadyStateResults, results...)
		m.mu.Unlock()

		// Check if any failed
		for _, r := range results {
			if !r.Passed {
				m.mu.Lock()
				exp.Status.Phase = ExperimentPhaseFailed
				exp.Status.Error = fmt.Sprintf("Pre-experiment steady state check failed: %s", r.Name)
				m.mu.Unlock()
				return fmt.Errorf("steady state check failed before experiment: %s", r.Name)
			}
		}
	}

	// Run experiments
	for _, action := range exp.Spec.Experiments {
		result := m.runExperimentAction(ctx, exp, action)

		m.mu.Lock()
		exp.Status.ExperimentResults = append(exp.Status.ExperimentResults, result)
		m.mu.Unlock()

		if !result.Success && exp.Spec.Rollback != nil && exp.Spec.Rollback.OnFailure {
			m.rollback(ctx, exp)
			return fmt.Errorf("experiment action failed: %s", result.Error)
		}

		// Check steady state during experiment
		if len(exp.Spec.SteadyState) > 0 {
			results := m.checkSteadyState(ctx, exp)
			m.mu.Lock()
			exp.Status.SteadyStateResults = append(exp.Status.SteadyStateResults, results...)
			m.mu.Unlock()

			for _, r := range results {
				if !r.Passed {
					if exp.Spec.Rollback != nil && exp.Spec.Rollback.OnSteadyStateViolation {
						m.rollback(ctx, exp)
						return fmt.Errorf("steady state violated during experiment: %s", r.Name)
					}
				}
			}
		}
	}

	// Mark as completed
	m.mu.Lock()
	completedAt := time.Now()
	exp.Status.Phase = ExperimentPhaseCompleted
	exp.Status.CompletedAt = &completedAt
	exp.Status.Duration = completedAt.Sub(*exp.Status.StartedAt).String()
	m.mu.Unlock()

	return nil
}

// runExperimentAction executes a single experiment action
func (m *Manager) runExperimentAction(ctx context.Context, exp *ChaosExperiment, action ExperimentAction) ExperimentResult {
	result := ExperimentResult{
		Type:      action.Type,
		StartedAt: time.Now(),
		Success:   true,
		Targets:   []string{exp.Spec.Target.Name},
		Metrics:   make(map[string]float64),
	}

	// Simulate experiment execution
	// In real implementation, this would inject actual faults
	fmt.Printf("[Chaos] Running %s experiment on %s for %s\n",
		action.Type, exp.Spec.Target.Name, action.Duration)

	// Parse duration and wait (simulated)
	duration, err := time.ParseDuration(action.Duration)
	if err != nil {
		duration = 1 * time.Second // Default for simulation
	}

	// For simulation, we'll use a shorter duration
	simDuration := duration
	if simDuration > 5*time.Second {
		simDuration = 1 * time.Second
	}

	select {
	case <-ctx.Done():
		result.Success = false
		result.Error = "experiment cancelled"
	case <-time.After(simDuration):
		// Simulate metrics
		result.Metrics["affected_pods"] = float64(exp.Spec.Target.Percentage)
		result.Metrics["duration_seconds"] = duration.Seconds()
	}

	now := time.Now()
	result.CompletedAt = &now
	return result
}

// checkSteadyState checks steady state hypotheses
func (m *Manager) checkSteadyState(ctx context.Context, exp *ChaosExperiment) []SteadyStateResult {
	results := make([]SteadyStateResult, 0, len(exp.Spec.SteadyState))

	for _, check := range exp.Spec.SteadyState {
		result := SteadyStateResult{
			Name:      check.Name,
			Threshold: check.Threshold,
			CheckedAt: time.Now(),
			Passed:    true, // Simulated - in real implementation, query metrics
		}

		// Simulate metric value
		switch check.Name {
		case "error-rate":
			result.Value = 0.5 // 0.5% error rate
		case "latency-p99":
			result.Value = 250 // 250ms
		case "response-time":
			result.Value = 100 // 100ms
		default:
			result.Value = 0
		}

		results = append(results, result)
	}

	return results
}

// rollback rolls back an experiment
func (m *Manager) rollback(ctx context.Context, exp *ChaosExperiment) {
	m.mu.Lock()
	defer m.mu.Unlock()

	exp.Status.Phase = ExperimentPhaseRolledBack
	exp.Status.RolledBack = true

	now := time.Now()
	exp.Status.CompletedAt = &now
	exp.Status.Duration = now.Sub(*exp.Status.StartedAt).String()

	fmt.Printf("[Chaos] Rolling back experiment %s\n", exp.Metadata.Name)
}

// PauseExperiment pauses a running experiment
func (m *Manager) PauseExperiment(name string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	exp, ok := m.experiments[name]
	if !ok {
		return fmt.Errorf("experiment not found: %s", name)
	}

	if exp.Status.Phase != ExperimentPhaseRunning {
		return fmt.Errorf("experiment is not running")
	}

	exp.Status.Phase = ExperimentPhasePaused
	return nil
}

// ResumeExperiment resumes a paused experiment
func (m *Manager) ResumeExperiment(ctx context.Context, name string) error {
	m.mu.Lock()
	exp, ok := m.experiments[name]
	if !ok {
		m.mu.Unlock()
		return fmt.Errorf("experiment not found: %s", name)
	}

	if exp.Status.Phase != ExperimentPhasePaused {
		m.mu.Unlock()
		return fmt.Errorf("experiment is not paused")
	}

	exp.Status.Phase = ExperimentPhaseRunning
	m.mu.Unlock()

	return nil
}

// StopExperiment stops an experiment
func (m *Manager) StopExperiment(name string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	exp, ok := m.experiments[name]
	if !ok {
		return fmt.Errorf("experiment not found: %s", name)
	}

	now := time.Now()
	exp.Status.Phase = ExperimentPhaseCompleted
	exp.Status.CompletedAt = &now
	if exp.Status.StartedAt != nil {
		exp.Status.Duration = now.Sub(*exp.Status.StartedAt).String()
	}

	return nil
}

// GetTemplate retrieves an experiment template
func (m *Manager) GetTemplate(name string) (*ExperimentTemplate, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	template, ok := m.templates[name]
	if !ok {
		return nil, fmt.Errorf("template not found: %s", name)
	}
	return template, nil
}

// ListTemplates returns all templates
func (m *Manager) ListTemplates() []*ExperimentTemplate {
	m.mu.RLock()
	defer m.mu.RUnlock()

	result := make([]*ExperimentTemplate, 0, len(m.templates))
	for _, t := range m.templates {
		result = append(result, t)
	}
	return result
}

// CreateFromTemplate creates an experiment from a template
func (m *Manager) CreateFromTemplate(ctx context.Context, templateName, experimentName string, target ExperimentTarget) (*ChaosExperiment, error) {
	template, err := m.GetTemplate(templateName)
	if err != nil {
		return nil, err
	}

	exp := &ChaosExperiment{
		APIVersion: "platformfoundry.io/v1",
		Kind:       "ChaosExperiment",
		Metadata: ExperimentMetadata{
			Name:        experimentName,
			Description: template.Description,
			Labels: map[string]string{
				"template": templateName,
			},
		},
		Spec: template.Spec,
	}

	// Override target
	exp.Spec.Target = target

	if err := m.RegisterExperiment(ctx, exp); err != nil {
		return nil, err
	}

	return exp, nil
}

// RegisterGameDay registers a game day
func (m *Manager) RegisterGameDay(ctx context.Context, gd *GameDay) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if gd.Metadata.Name == "" {
		return fmt.Errorf("game day name is required")
	}

	gd.Metadata.CreatedAt = time.Now()

	gd.Status = &GameDayStatus{
		Phase: GameDayPhaseScheduled,
	}

	m.gameDays[gd.Metadata.Name] = gd
	return nil
}

// GetGameDay retrieves a game day
func (m *Manager) GetGameDay(name string) (*GameDay, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	gd, ok := m.gameDays[name]
	if !ok {
		return nil, fmt.Errorf("game day not found: %s", name)
	}
	return gd, nil
}

// ListGameDays returns all game days
func (m *Manager) ListGameDays() []*GameDay {
	m.mu.RLock()
	defer m.mu.RUnlock()

	result := make([]*GameDay, 0, len(m.gameDays))
	for _, gd := range m.gameDays {
		result = append(result, gd)
	}
	return result
}

// GenerateReport generates a chaos engineering report
func (m *Manager) GenerateReport(ctx context.Context, period string) (*ChaosReport, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	report := &ChaosReport{
		GeneratedAt:      time.Now(),
		Period:           period,
		TotalExperiments: len(m.experiments),
		ByType:           make(map[string]int),
		Recommendations:  make([]string, 0),
	}

	// Count by type and calculate success rate
	successCount := 0
	for _, exp := range m.experiments {
		for _, action := range exp.Spec.Experiments {
			report.ByType[string(action.Type)]++
		}
		if exp.Status != nil && exp.Status.Phase == ExperimentPhaseCompleted {
			successCount++
		}
	}

	if report.TotalExperiments > 0 {
		report.SuccessRate = float64(successCount) / float64(report.TotalExperiments) * 100
	}

	// Add recommendations
	if report.SuccessRate < 80 {
		report.Recommendations = append(report.Recommendations,
			"Consider reviewing failed experiments and addressing root causes")
	}
	if len(m.experiments) < 5 {
		report.Recommendations = append(report.Recommendations,
			"Increase chaos experiment coverage across services")
	}

	return report, nil
}
