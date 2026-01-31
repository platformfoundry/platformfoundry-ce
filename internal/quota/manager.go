package quota

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/platformfoundry/platformfoundry-ce/pkg/types"
)

// Manager manages resource quotas
type Manager struct {
	mu              sync.RWMutex
	quotas          map[string]*types.ResourceQuota
	usage           map[string]map[string]int64 // quota name -> resource -> usage
	enforcementMode types.QuotaEnforcementMode
	alertHandler    AlertHandler
}

// AlertHandler handles quota alerts
type AlertHandler func(alert *types.QuotaAlert) error

// ManagerConfig configures the quota manager
type ManagerConfig struct {
	EnforcementMode types.QuotaEnforcementMode
	AlertHandler    AlertHandler
}

// NewManager creates a new quota manager
func NewManager(config ManagerConfig) *Manager {
	if config.EnforcementMode == "" {
		config.EnforcementMode = types.EnforcementModeEnforce
	}

	return &Manager{
		quotas:          make(map[string]*types.ResourceQuota),
		usage:           make(map[string]map[string]int64),
		enforcementMode: config.EnforcementMode,
		alertHandler:    config.AlertHandler,
	}
}

// RegisterQuota registers a new quota
func (m *Manager) RegisterQuota(quota *types.ResourceQuota) error {
	if err := quota.Validate(); err != nil {
		return err
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	key := m.quotaKey(quota)
	m.quotas[key] = quota
	m.usage[key] = make(map[string]int64)

	return nil
}

// UnregisterQuota removes a quota
func (m *Manager) UnregisterQuota(name string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, exists := m.quotas[name]; !exists {
		return types.ErrQuotaNotFound
	}

	delete(m.quotas, name)
	delete(m.usage, name)
	return nil
}

// GetQuota retrieves a quota by name
func (m *Manager) GetQuota(name string) (*types.ResourceQuota, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	quota, exists := m.quotas[name]
	if !exists {
		return nil, types.ErrQuotaNotFound
	}
	return quota, nil
}

// ListQuotas lists all quotas
func (m *Manager) ListQuotas() []*types.ResourceQuota {
	m.mu.RLock()
	defer m.mu.RUnlock()

	quotas := make([]*types.ResourceQuota, 0, len(m.quotas))
	for _, q := range m.quotas {
		quotas = append(quotas, q)
	}
	return quotas
}

// Check checks if a resource usage is within quota
func (m *Manager) Check(ctx context.Context, scope types.QuotaScope, resource string, requested int64) (*types.QuotaCheckResult, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	result := &types.QuotaCheckResult{
		Allowed:  true,
		Warnings: make([]string, 0),
	}

	// Find applicable quotas
	for key, quota := range m.quotas {
		if !m.scopeMatches(quota.Spec.Scope, scope) {
			continue
		}

		limit, hasLimit := quota.Spec.Hard[resource]
		if !hasLimit {
			continue
		}

		currentUsage := m.usage[key][resource]
		newUsage := currentUsage + requested

		result.CurrentUsage = currentUsage
		result.Limit = limit
		result.Available = limit - currentUsage

		// Check hard limit
		if newUsage > limit {
			result.Allowed = m.enforcementMode != types.EnforcementModeEnforce
			result.Reason = fmt.Sprintf("quota exceeded for %s: requested %d, limit %d, current usage %d", resource, requested, limit, currentUsage)
			result.ExceededQuota = key
			return result, nil
		}

		// Check soft limit
		if softLimit, hasSoftLimit := quota.Spec.Soft[resource]; hasSoftLimit {
			if newUsage > softLimit {
				result.Warnings = append(result.Warnings, fmt.Sprintf("soft limit exceeded for %s: %d > %d", resource, newUsage, softLimit))
			}
		}

		// Check thresholds for alerts
		if quota.Spec.AlertPolicy != nil {
			usagePercent := float64(newUsage) / float64(limit) * 100
			if usagePercent >= float64(quota.Spec.AlertPolicy.CriticalThreshold) {
				result.Warnings = append(result.Warnings, fmt.Sprintf("critical threshold reached for %s: %.1f%%", resource, usagePercent))
			} else if usagePercent >= float64(quota.Spec.AlertPolicy.WarningThreshold) {
				result.Warnings = append(result.Warnings, fmt.Sprintf("warning threshold reached for %s: %.1f%%", resource, usagePercent))
			}
		}
	}

	return result, nil
}

// Consume records resource usage
func (m *Manager) Consume(ctx context.Context, scope types.QuotaScope, resource string, amount int64) error {
	// First check if allowed
	result, err := m.Check(ctx, scope, resource, amount)
	if err != nil {
		return err
	}

	if !result.Allowed {
		return fmt.Errorf("%s: %s", types.ErrQuotaExceeded, result.Reason)
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	// Find and update applicable quotas
	for key, quota := range m.quotas {
		if !m.scopeMatches(quota.Spec.Scope, scope) {
			continue
		}

		if _, hasLimit := quota.Spec.Hard[resource]; hasLimit {
			if m.usage[key] == nil {
				m.usage[key] = make(map[string]int64)
			}
			m.usage[key][resource] += amount

			// Trigger alerts if needed
			m.checkAndTriggerAlerts(key, quota, resource)
		}
	}

	return nil
}

// Release releases previously consumed resources
func (m *Manager) Release(ctx context.Context, scope types.QuotaScope, resource string, amount int64) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	for key, quota := range m.quotas {
		if !m.scopeMatches(quota.Spec.Scope, scope) {
			continue
		}

		if _, hasLimit := quota.Spec.Hard[resource]; hasLimit {
			if m.usage[key] != nil {
				m.usage[key][resource] -= amount
				if m.usage[key][resource] < 0 {
					m.usage[key][resource] = 0
				}
			}
		}
	}

	return nil
}

// GetUsage returns current usage for a quota
func (m *Manager) GetUsage(quotaName string) (map[string]int64, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	usage, exists := m.usage[quotaName]
	if !exists {
		return nil, types.ErrQuotaNotFound
	}

	// Return a copy
	result := make(map[string]int64)
	for k, v := range usage {
		result[k] = v
	}
	return result, nil
}

// GetStatus returns the status of a quota
func (m *Manager) GetStatus(quotaName string) (*types.ResourceQuotaStatus, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	quota, exists := m.quotas[quotaName]
	if !exists {
		return nil, types.ErrQuotaNotFound
	}

	usage, exists := m.usage[quotaName]
	if !exists {
		usage = make(map[string]int64)
	}

	status := &types.ResourceQuotaStatus{
		Used:           make(map[string]int64),
		UsedPercentage: make(map[string]float64),
		Available:      make(map[string]int64),
		Alerts:         make([]types.QuotaAlert, 0),
		LastUpdated:    time.Now(),
	}

	for resource, limit := range quota.Spec.Hard {
		used := usage[resource]
		status.Used[resource] = used
		status.Available[resource] = limit - used
		if limit > 0 {
			status.UsedPercentage[resource] = float64(used) / float64(limit) * 100
		}

		// Check for alerts
		if quota.Spec.AlertPolicy != nil {
			usedPercent := status.UsedPercentage[resource]
			if usedPercent >= float64(quota.Spec.AlertPolicy.CriticalThreshold) {
				status.Alerts = append(status.Alerts, types.QuotaAlert{
					Resource:    resource,
					Severity:    "critical",
					Message:     fmt.Sprintf("Critical: %s usage at %.1f%%", resource, usedPercent),
					Threshold:   quota.Spec.AlertPolicy.CriticalThreshold,
					Current:     usedPercent,
					TriggeredAt: time.Now(),
				})
			} else if usedPercent >= float64(quota.Spec.AlertPolicy.WarningThreshold) {
				status.Alerts = append(status.Alerts, types.QuotaAlert{
					Resource:    resource,
					Severity:    "warning",
					Message:     fmt.Sprintf("Warning: %s usage at %.1f%%", resource, usedPercent),
					Threshold:   quota.Spec.AlertPolicy.WarningThreshold,
					Current:     usedPercent,
					TriggeredAt: time.Now(),
				})
			}
		}
	}

	return status, nil
}

// GenerateReport generates a usage report for a scope
func (m *Manager) GenerateReport(scope types.QuotaScope) (*types.QuotaUsageReport, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	report := &types.QuotaUsageReport{
		Scope:       scope,
		Quotas:      make([]types.QuotaSummary, 0),
		GeneratedAt: time.Now(),
	}

	for key, quota := range m.quotas {
		if !m.scopeMatches(quota.Spec.Scope, scope) {
			continue
		}

		usage := m.usage[key]
		if usage == nil {
			usage = make(map[string]int64)
		}

		for resource, limit := range quota.Spec.Hard {
			used := usage[resource]
			available := limit - used
			usedPercent := float64(0)
			if limit > 0 {
				usedPercent = float64(used) / float64(limit) * 100
			}

			status := "ok"
			if usedPercent >= 100 {
				status = "exceeded"
				report.OverQuotaCount++
			} else if quota.Spec.AlertPolicy != nil && usedPercent >= float64(quota.Spec.AlertPolicy.CriticalThreshold) {
				status = "critical"
				report.NearLimitCount++
			} else if quota.Spec.AlertPolicy != nil && usedPercent >= float64(quota.Spec.AlertPolicy.WarningThreshold) {
				status = "warning"
				report.NearLimitCount++
			}

			summary := types.QuotaSummary{
				Resource:       resource,
				Hard:           limit,
				Used:           used,
				Available:      available,
				UsedPercentage: usedPercent,
				Status:         status,
			}

			if softLimit, ok := quota.Spec.Soft[resource]; ok {
				summary.Soft = softLimit
			}

			report.Quotas = append(report.Quotas, summary)
			report.TotalResources++
		}
	}

	return report, nil
}

// Helper methods

func (m *Manager) quotaKey(quota *types.ResourceQuota) string {
	return quota.Metadata.Name
}

func (m *Manager) scopeMatches(quotaScope, requestScope types.QuotaScope) bool {
	// Check type matches
	if quotaScope.Type != requestScope.Type {
		return false
	}

	// Check name matches (empty quota name means all)
	if quotaScope.Name != "" && quotaScope.Name != requestScope.Name {
		return false
	}

	// Check selector matches
	if len(quotaScope.Selector) > 0 {
		for k, v := range quotaScope.Selector {
			if requestScope.Selector[k] != v {
				return false
			}
		}
	}

	return true
}

func (m *Manager) checkAndTriggerAlerts(quotaKey string, quota *types.ResourceQuota, resource string) {
	if quota.Spec.AlertPolicy == nil || m.alertHandler == nil {
		return
	}

	usage := m.usage[quotaKey][resource]
	limit := quota.Spec.Hard[resource]
	if limit == 0 {
		return
	}

	usedPercent := float64(usage) / float64(limit) * 100

	var alert *types.QuotaAlert

	if usedPercent >= float64(quota.Spec.AlertPolicy.CriticalThreshold) {
		alert = &types.QuotaAlert{
			Resource:    resource,
			Severity:    "critical",
			Message:     fmt.Sprintf("Critical: %s usage at %.1f%% (threshold: %d%%)", resource, usedPercent, quota.Spec.AlertPolicy.CriticalThreshold),
			Threshold:   quota.Spec.AlertPolicy.CriticalThreshold,
			Current:     usedPercent,
			TriggeredAt: time.Now(),
		}
	} else if usedPercent >= float64(quota.Spec.AlertPolicy.WarningThreshold) {
		alert = &types.QuotaAlert{
			Resource:    resource,
			Severity:    "warning",
			Message:     fmt.Sprintf("Warning: %s usage at %.1f%% (threshold: %d%%)", resource, usedPercent, quota.Spec.AlertPolicy.WarningThreshold),
			Threshold:   quota.Spec.AlertPolicy.WarningThreshold,
			Current:     usedPercent,
			TriggeredAt: time.Now(),
		}
	}

	if alert != nil {
		// Fire and forget alert handling
		go m.alertHandler(alert)
	}
}

// SetEnforcementMode changes the enforcement mode
func (m *Manager) SetEnforcementMode(mode types.QuotaEnforcementMode) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.enforcementMode = mode
}

// GetEnforcementMode returns the current enforcement mode
func (m *Manager) GetEnforcementMode() types.QuotaEnforcementMode {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.enforcementMode
}

// ResetUsage resets usage for a quota
func (m *Manager) ResetUsage(quotaName string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, exists := m.quotas[quotaName]; !exists {
		return types.ErrQuotaNotFound
	}

	m.usage[quotaName] = make(map[string]int64)
	return nil
}

// SetUsage directly sets usage for a quota resource
func (m *Manager) SetUsage(quotaName, resource string, usage int64) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, exists := m.quotas[quotaName]; !exists {
		return types.ErrQuotaNotFound
	}

	if m.usage[quotaName] == nil {
		m.usage[quotaName] = make(map[string]int64)
	}
	m.usage[quotaName][resource] = usage
	return nil
}
