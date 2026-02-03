package types

import (
	"time"
)

// ResourceQuota defines resource limits for a scope (org, team, environment)
type ResourceQuota struct {
	APIVersion string               `yaml:"apiVersion" json:"apiVersion"`
	Kind       string               `yaml:"kind" json:"kind"`
	Metadata   QuotaMetadata        `yaml:"metadata" json:"metadata"`
	Spec       ResourceQuotaSpec    `yaml:"spec" json:"spec"`
	Status     *ResourceQuotaStatus `yaml:"status,omitempty" json:"status,omitempty"`
}

// QuotaMetadata contains quota metadata
type QuotaMetadata struct {
	Name        string            `yaml:"name" json:"name"`
	Labels      map[string]string `yaml:"labels,omitempty" json:"labels,omitempty"`
	Annotations map[string]string `yaml:"annotations,omitempty" json:"annotations,omitempty"`
}

// ResourceQuotaSpec defines the quota specification
type ResourceQuotaSpec struct {
	// Scope defines what this quota applies to
	Scope QuotaScope `yaml:"scope" json:"scope"`

	// Hard limits that cannot be exceeded
	Hard map[string]int64 `yaml:"hard" json:"hard"`

	// Soft limits that generate warnings
	Soft map[string]int64 `yaml:"soft,omitempty" json:"soft,omitempty"`

	// AlertPolicy defines how to alert on quota usage
	AlertPolicy *QuotaAlertPolicy `yaml:"alertPolicy,omitempty" json:"alertPolicy,omitempty"`

	// Priority for resolving conflicts
	Priority int `yaml:"priority,omitempty" json:"priority,omitempty"`
}

// QuotaScope defines the scope of a quota
type QuotaScope struct {
	// Type of scope (organization, team, environment, namespace)
	Type string `yaml:"type" json:"type"`

	// Name of the scope (org name, team name, etc.)
	Name string `yaml:"name" json:"name"`

	// Selector for more specific scoping
	Selector map[string]string `yaml:"selector,omitempty" json:"selector,omitempty"`
}

// QuotaAlertPolicy defines when to alert
type QuotaAlertPolicy struct {
	// WarningThreshold percentage (0-100)
	WarningThreshold int `yaml:"warningThreshold" json:"warningThreshold"`

	// CriticalThreshold percentage (0-100)
	CriticalThreshold int `yaml:"criticalThreshold" json:"criticalThreshold"`

	// NotifyChannels to send alerts to
	NotifyChannels []string `yaml:"notifyChannels,omitempty" json:"notifyChannels,omitempty"`
}

// ResourceQuotaStatus represents the current quota usage
type ResourceQuotaStatus struct {
	// Used resources
	Used map[string]int64 `yaml:"used" json:"used"`

	// Percentage used
	UsedPercentage map[string]float64 `yaml:"usedPercentage" json:"usedPercentage"`

	// Available resources
	Available map[string]int64 `yaml:"available" json:"available"`

	// Alerts triggered
	Alerts []QuotaAlert `yaml:"alerts,omitempty" json:"alerts,omitempty"`

	// LastUpdated timestamp
	LastUpdated time.Time `yaml:"lastUpdated" json:"lastUpdated"`
}

// QuotaAlert represents a quota alert
type QuotaAlert struct {
	Resource    string    `yaml:"resource" json:"resource"`
	Severity    string    `yaml:"severity" json:"severity"`
	Message     string    `yaml:"message" json:"message"`
	Threshold   int       `yaml:"threshold" json:"threshold"`
	Current     float64   `yaml:"current" json:"current"`
	TriggeredAt time.Time `yaml:"triggeredAt" json:"triggeredAt"`
}

// Common resource quota names
const (
	QuotaPlatforms           = "platforms"
	QuotaServices            = "services"
	QuotaPromises            = "promises"
	QuotaWorkloads           = "workloads"
	QuotaDatabases           = "databases"
	QuotaCaches              = "caches"
	QuotaQueues              = "queues"
	QuotaStorageGB           = "storage.gb"
	QuotaCPUCores            = "cpu.cores"
	QuotaMemoryGB            = "memory.gb"
	QuotaMonthlyCostUSD      = "cost.monthly.usd"
	QuotaChaosExperiments    = "chaos.experiments"
	QuotaAPIRequestsPerHour  = "api.requests.per_hour"
	QuotaConcurrentWorkflows = "workflows.concurrent"
)

// QuotaEnforcementMode defines how quotas are enforced
type QuotaEnforcementMode string

const (
	// EnforcementModeEnforce blocks operations that exceed quota
	EnforcementModeEnforce QuotaEnforcementMode = "enforce"

	// EnforcementModeWarn only warns but allows operations
	EnforcementModeWarn QuotaEnforcementMode = "warn"

	// EnforcementModeAudit only logs but takes no action
	EnforcementModeAudit QuotaEnforcementMode = "audit"
)

// QuotaCheckResult represents the result of a quota check
type QuotaCheckResult struct {
	// Allowed indicates if the operation is allowed
	Allowed bool `json:"allowed" yaml:"allowed"`

	// Reason for denial if not allowed
	Reason string `json:"reason,omitempty" yaml:"reason,omitempty"`

	// Quota that was exceeded
	ExceededQuota string `json:"exceededQuota,omitempty" yaml:"exceededQuota,omitempty"`

	// Current usage
	CurrentUsage int64 `json:"currentUsage" yaml:"currentUsage"`

	// Limit
	Limit int64 `json:"limit" yaml:"limit"`

	// Available capacity
	Available int64 `json:"available" yaml:"available"`

	// Warnings if any soft limits were exceeded
	Warnings []string `json:"warnings,omitempty" yaml:"warnings,omitempty"`
}

// QuotaUsageReport provides a summary of quota usage
type QuotaUsageReport struct {
	// Scope of the report
	Scope QuotaScope `json:"scope" yaml:"scope"`

	// Quotas in this scope
	Quotas []QuotaSummary `json:"quotas" yaml:"quotas"`

	// TotalResources count
	TotalResources int `json:"totalResources" yaml:"totalResources"`

	// OverQuotaCount resources over quota
	OverQuotaCount int `json:"overQuotaCount" yaml:"overQuotaCount"`

	// NearLimitCount resources near limit
	NearLimitCount int `json:"nearLimitCount" yaml:"nearLimitCount"`

	// GeneratedAt timestamp
	GeneratedAt time.Time `json:"generatedAt" yaml:"generatedAt"`
}

// QuotaSummary summarizes a single quota
type QuotaSummary struct {
	Resource       string  `json:"resource" yaml:"resource"`
	Hard           int64   `json:"hard" yaml:"hard"`
	Soft           int64   `json:"soft,omitempty" yaml:"soft,omitempty"`
	Used           int64   `json:"used" yaml:"used"`
	Available      int64   `json:"available" yaml:"available"`
	UsedPercentage float64 `json:"usedPercentage" yaml:"usedPercentage"`
	Status         string  `json:"status" yaml:"status"` // ok, warning, critical, exceeded
}

// NewResourceQuota creates a new ResourceQuota with defaults
func NewResourceQuota(name string, scopeType, scopeName string) *ResourceQuota {
	return &ResourceQuota{
		APIVersion: "platformfoundry.io/v1",
		Kind:       "ResourceQuota",
		Metadata: QuotaMetadata{
			Name:   name,
			Labels: make(map[string]string),
		},
		Spec: ResourceQuotaSpec{
			Scope: QuotaScope{
				Type: scopeType,
				Name: scopeName,
			},
			Hard: make(map[string]int64),
			Soft: make(map[string]int64),
			AlertPolicy: &QuotaAlertPolicy{
				WarningThreshold:  80,
				CriticalThreshold: 95,
			},
		},
	}
}

// SetHardLimit sets a hard limit for a resource
func (q *ResourceQuota) SetHardLimit(resource string, limit int64) {
	q.Spec.Hard[resource] = limit
}

// SetSoftLimit sets a soft limit for a resource
func (q *ResourceQuota) SetSoftLimit(resource string, limit int64) {
	q.Spec.Soft[resource] = limit
}

// GetLimit returns the hard limit for a resource
func (q *ResourceQuota) GetLimit(resource string) (int64, bool) {
	limit, ok := q.Spec.Hard[resource]
	return limit, ok
}

// Validate validates the quota configuration
func (q *ResourceQuota) Validate() error {
	if q.Metadata.Name == "" {
		return ErrMissingName
	}
	if q.Spec.Scope.Type == "" {
		return ErrInvalidScope
	}
	if q.Spec.Scope.Name == "" {
		return ErrInvalidScope
	}
	return nil
}
