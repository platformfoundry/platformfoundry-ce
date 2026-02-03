package compliance

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/platformfoundry/pf-ce/pkg/types"
)

// ContinuousMonitor provides continuous compliance monitoring
type ContinuousMonitor struct {
	scanner      *Scanner
	config       *MonitorConfig
	alerts       []ComplianceAlert
	driftHistory map[string]*DriftRecord
	callbacks    []AlertCallback
	mu           sync.RWMutex
	stopCh       chan struct{}
	running      bool
}

// MonitorConfig configures continuous monitoring
type MonitorConfig struct {
	ScanInterval      time.Duration `json:"scanInterval" yaml:"scanInterval"`
	DriftCheckEnabled bool          `json:"driftCheckEnabled" yaml:"driftCheckEnabled"`
	AlertThreshold    float64       `json:"alertThreshold" yaml:"alertThreshold"`
	CriticalThreshold float64       `json:"criticalThreshold" yaml:"criticalThreshold"`
	RetainAlertDays   int           `json:"retainAlertDays" yaml:"retainAlertDays"`
}

// DefaultMonitorConfig returns default monitoring configuration
func DefaultMonitorConfig() *MonitorConfig {
	return &MonitorConfig{
		ScanInterval:      30 * time.Minute,
		DriftCheckEnabled: true,
		AlertThreshold:    90.0,
		CriticalThreshold: 80.0,
		RetainAlertDays:   30,
	}
}

// ComplianceAlert represents a compliance alert
type ComplianceAlert struct {
	ID           string                 `json:"id"`
	Type         AlertType              `json:"type"`
	Severity     AlertSeverity          `json:"severity"`
	PolicyName   string                 `json:"policyName"`
	Framework    string                 `json:"framework"`
	Message      string                 `json:"message"`
	Details      map[string]interface{} `json:"details,omitempty"`
	Timestamp    time.Time              `json:"timestamp"`
	Acknowledged bool                   `json:"acknowledged"`
	AckedBy      string                 `json:"ackedBy,omitempty"`
	AckedAt      *time.Time             `json:"ackedAt,omitempty"`
}

// AlertType defines types of compliance alerts
type AlertType string

const (
	AlertTypeThresholdBreach AlertType = "threshold_breach"
	AlertTypeDriftDetected   AlertType = "drift_detected"
	AlertTypeNewViolation    AlertType = "new_violation"
	AlertTypePolicyFailed    AlertType = "policy_failed"
	AlertTypeFrameworkUpdate AlertType = "framework_update"
)

// AlertSeverity defines alert severity levels
type AlertSeverity string

const (
	AlertSeverityCritical AlertSeverity = "critical"
	AlertSeverityHigh     AlertSeverity = "high"
	AlertSeverityMedium   AlertSeverity = "medium"
	AlertSeverityLow      AlertSeverity = "low"
	AlertSeverityInfo     AlertSeverity = "info"
)

// DriftRecord tracks compliance drift over time
type DriftRecord struct {
	PolicyName      string           `json:"policyName"`
	BaselineScore   float64          `json:"baselineScore"`
	CurrentScore    float64          `json:"currentScore"`
	DriftPercentage float64          `json:"driftPercentage"`
	Direction       DriftDirection   `json:"direction"`
	History         []DriftDataPoint `json:"history"`
	LastUpdated     time.Time        `json:"lastUpdated"`
}

// DriftDirection indicates compliance trend
type DriftDirection string

const (
	DriftDirectionImproving DriftDirection = "improving"
	DriftDirectionDegrading DriftDirection = "degrading"
	DriftDirectionStable    DriftDirection = "stable"
)

// DriftDataPoint represents a point in drift history
type DriftDataPoint struct {
	Timestamp  time.Time `json:"timestamp"`
	Compliance float64   `json:"compliance"`
	Passed     int       `json:"passed"`
	Failed     int       `json:"failed"`
}

// AlertCallback is called when alerts are generated
type AlertCallback func(alert ComplianceAlert)

// NewContinuousMonitor creates a new continuous monitor
func NewContinuousMonitor(scanner *Scanner, config *MonitorConfig) *ContinuousMonitor {
	if config == nil {
		config = DefaultMonitorConfig()
	}

	return &ContinuousMonitor{
		scanner:      scanner,
		config:       config,
		alerts:       make([]ComplianceAlert, 0),
		driftHistory: make(map[string]*DriftRecord),
		callbacks:    make([]AlertCallback, 0),
		stopCh:       make(chan struct{}),
	}
}

// OnAlert registers an alert callback
func (m *ContinuousMonitor) OnAlert(callback AlertCallback) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.callbacks = append(m.callbacks, callback)
}

// Start begins continuous monitoring
func (m *ContinuousMonitor) Start(ctx context.Context) error {
	m.mu.Lock()
	if m.running {
		m.mu.Unlock()
		return fmt.Errorf("monitor already running")
	}
	m.running = true
	m.stopCh = make(chan struct{})
	m.mu.Unlock()

	go m.monitorLoop(ctx)
	return nil
}

// Stop stops continuous monitoring
func (m *ContinuousMonitor) Stop() {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.running {
		close(m.stopCh)
		m.running = false
	}
}

// monitorLoop runs the monitoring loop
func (m *ContinuousMonitor) monitorLoop(ctx context.Context) {
	ticker := time.NewTicker(m.config.ScanInterval)
	defer ticker.Stop()

	// Run initial scan
	m.runScanCycle(ctx)

	for {
		select {
		case <-ctx.Done():
			return
		case <-m.stopCh:
			return
		case <-ticker.C:
			m.runScanCycle(ctx)
		}
	}
}

// runScanCycle performs a complete scan cycle
func (m *ContinuousMonitor) runScanCycle(ctx context.Context) {
	results, err := m.scanner.ScanAll(ctx)
	if err != nil {
		m.emitAlert(ComplianceAlert{
			ID:        fmt.Sprintf("alert-%d", time.Now().UnixNano()),
			Type:      AlertTypePolicyFailed,
			Severity:  AlertSeverityHigh,
			Message:   fmt.Sprintf("Compliance scan failed: %v", err),
			Timestamp: time.Now(),
		})
		return
	}

	for _, result := range results {
		m.processResult(result)
	}
}

// processResult analyzes a scan result and generates alerts
func (m *ContinuousMonitor) processResult(result *types.ComplianceScanResult) {
	// Check threshold breach
	if result.Compliance < m.config.CriticalThreshold {
		m.emitAlert(ComplianceAlert{
			ID:         fmt.Sprintf("alert-%d", time.Now().UnixNano()),
			Type:       AlertTypeThresholdBreach,
			Severity:   AlertSeverityCritical,
			PolicyName: result.PolicyName,
			Framework:  string(result.Framework),
			Message:    fmt.Sprintf("Compliance dropped below critical threshold: %.1f%% (threshold: %.1f%%)", result.Compliance, m.config.CriticalThreshold),
			Details: map[string]interface{}{
				"compliance": result.Compliance,
				"passed":     result.Passed,
				"failed":     result.Failed,
			},
			Timestamp: time.Now(),
		})
	} else if result.Compliance < m.config.AlertThreshold {
		m.emitAlert(ComplianceAlert{
			ID:         fmt.Sprintf("alert-%d", time.Now().UnixNano()),
			Type:       AlertTypeThresholdBreach,
			Severity:   AlertSeverityHigh,
			PolicyName: result.PolicyName,
			Framework:  string(result.Framework),
			Message:    fmt.Sprintf("Compliance below alert threshold: %.1f%% (threshold: %.1f%%)", result.Compliance, m.config.AlertThreshold),
			Timestamp:  time.Now(),
		})
	}

	// Check for drift
	if m.config.DriftCheckEnabled {
		m.checkDrift(result)
	}

	// Check for new violations
	for _, violation := range result.Violations {
		if violation.Severity == types.SeverityCritical {
			m.emitAlert(ComplianceAlert{
				ID:         fmt.Sprintf("alert-%d", time.Now().UnixNano()),
				Type:       AlertTypeNewViolation,
				Severity:   AlertSeverityCritical,
				PolicyName: result.PolicyName,
				Framework:  string(result.Framework),
				Message:    fmt.Sprintf("Critical violation: %s", violation.RuleName),
				Details: map[string]interface{}{
					"ruleID":      violation.RuleID,
					"resource":    violation.Resource,
					"remediation": violation.Remediation,
				},
				Timestamp: time.Now(),
			})
		}
	}
}

// checkDrift analyzes compliance drift
func (m *ContinuousMonitor) checkDrift(result *types.ComplianceScanResult) {
	m.mu.Lock()
	defer m.mu.Unlock()

	record, exists := m.driftHistory[result.PolicyName]
	if !exists {
		// First record - establish baseline
		record = &DriftRecord{
			PolicyName:    result.PolicyName,
			BaselineScore: result.Compliance,
			CurrentScore:  result.Compliance,
			Direction:     DriftDirectionStable,
			History:       make([]DriftDataPoint, 0),
		}
		m.driftHistory[result.PolicyName] = record
	}

	// Add data point
	dataPoint := DriftDataPoint{
		Timestamp:  time.Now(),
		Compliance: result.Compliance,
		Passed:     result.Passed,
		Failed:     result.Failed,
	}
	record.History = append(record.History, dataPoint)

	// Keep only last 100 data points
	if len(record.History) > 100 {
		record.History = record.History[len(record.History)-100:]
	}

	// Calculate drift
	previousScore := record.CurrentScore
	record.CurrentScore = result.Compliance
	record.DriftPercentage = record.CurrentScore - record.BaselineScore
	record.LastUpdated = time.Now()

	// Determine direction
	diff := result.Compliance - previousScore
	if diff > 1.0 {
		record.Direction = DriftDirectionImproving
	} else if diff < -1.0 {
		record.Direction = DriftDirectionDegrading
	} else {
		record.Direction = DriftDirectionStable
	}

	// Alert on significant degradation
	if record.Direction == DriftDirectionDegrading && diff < -5.0 {
		m.emitAlertUnlocked(ComplianceAlert{
			ID:         fmt.Sprintf("alert-%d", time.Now().UnixNano()),
			Type:       AlertTypeDriftDetected,
			Severity:   AlertSeverityHigh,
			PolicyName: result.PolicyName,
			Message:    fmt.Sprintf("Compliance degrading: %.1f%% -> %.1f%% (%.1f%% drop)", previousScore, result.Compliance, -diff),
			Details: map[string]interface{}{
				"baseline": record.BaselineScore,
				"current":  record.CurrentScore,
				"drift":    record.DriftPercentage,
			},
			Timestamp: time.Now(),
		})
	}
}

// emitAlert sends an alert
func (m *ContinuousMonitor) emitAlert(alert ComplianceAlert) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.emitAlertUnlocked(alert)
}

// emitAlertUnlocked sends an alert without locking
func (m *ContinuousMonitor) emitAlertUnlocked(alert ComplianceAlert) {
	m.alerts = append(m.alerts, alert)

	// Clean old alerts
	cutoff := time.Now().AddDate(0, 0, -m.config.RetainAlertDays)
	filtered := make([]ComplianceAlert, 0)
	for _, a := range m.alerts {
		if a.Timestamp.After(cutoff) {
			filtered = append(filtered, a)
		}
	}
	m.alerts = filtered

	// Notify callbacks
	for _, cb := range m.callbacks {
		go cb(alert)
	}
}

// GetAlerts returns alerts with optional filtering
func (m *ContinuousMonitor) GetAlerts(severity *AlertSeverity, acknowledged *bool) []ComplianceAlert {
	m.mu.RLock()
	defer m.mu.RUnlock()

	result := make([]ComplianceAlert, 0)
	for _, alert := range m.alerts {
		if severity != nil && alert.Severity != *severity {
			continue
		}
		if acknowledged != nil && alert.Acknowledged != *acknowledged {
			continue
		}
		result = append(result, alert)
	}
	return result
}

// AcknowledgeAlert marks an alert as acknowledged
func (m *ContinuousMonitor) AcknowledgeAlert(alertID, user string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	for i := range m.alerts {
		if m.alerts[i].ID == alertID {
			m.alerts[i].Acknowledged = true
			m.alerts[i].AckedBy = user
			now := time.Now()
			m.alerts[i].AckedAt = &now
			return nil
		}
	}
	return fmt.Errorf("alert not found: %s", alertID)
}

// GetDriftRecords returns all drift records
func (m *ContinuousMonitor) GetDriftRecords() map[string]*DriftRecord {
	m.mu.RLock()
	defer m.mu.RUnlock()

	result := make(map[string]*DriftRecord)
	for k, v := range m.driftHistory {
		result[k] = v
	}
	return result
}

// GetDriftRecord returns drift record for a policy
func (m *ContinuousMonitor) GetDriftRecord(policyName string) (*DriftRecord, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	record, ok := m.driftHistory[policyName]
	if !ok {
		return nil, fmt.Errorf("no drift record for policy: %s", policyName)
	}
	return record, nil
}

// ResetBaseline resets the drift baseline for a policy
func (m *ContinuousMonitor) ResetBaseline(policyName string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	record, ok := m.driftHistory[policyName]
	if !ok {
		return fmt.Errorf("no drift record for policy: %s", policyName)
	}

	record.BaselineScore = record.CurrentScore
	record.DriftPercentage = 0
	record.Direction = DriftDirectionStable
	return nil
}

// GetComplianceTrend returns compliance trend for a policy
func (m *ContinuousMonitor) GetComplianceTrend(policyName string, duration time.Duration) []DriftDataPoint {
	m.mu.RLock()
	defer m.mu.RUnlock()

	record, ok := m.driftHistory[policyName]
	if !ok {
		return nil
	}

	cutoff := time.Now().Add(-duration)
	result := make([]DriftDataPoint, 0)
	for _, dp := range record.History {
		if dp.Timestamp.After(cutoff) {
			result = append(result, dp)
		}
	}
	return result
}

// MonitorStatus returns current monitoring status
func (m *ContinuousMonitor) Status() map[string]interface{} {
	m.mu.RLock()
	defer m.mu.RUnlock()

	unackedCritical := 0
	unackedHigh := 0
	for _, alert := range m.alerts {
		if !alert.Acknowledged {
			switch alert.Severity {
			case AlertSeverityCritical:
				unackedCritical++
			case AlertSeverityHigh:
				unackedHigh++
			}
		}
	}

	degradingPolicies := 0
	for _, record := range m.driftHistory {
		if record.Direction == DriftDirectionDegrading {
			degradingPolicies++
		}
	}

	return map[string]interface{}{
		"running":           m.running,
		"totalAlerts":       len(m.alerts),
		"unackedCritical":   unackedCritical,
		"unackedHigh":       unackedHigh,
		"monitoredPolicies": len(m.driftHistory),
		"degradingPolicies": degradingPolicies,
		"scanInterval":      m.config.ScanInterval.String(),
		"alertThreshold":    m.config.AlertThreshold,
		"criticalThreshold": m.config.CriticalThreshold,
	}
}
