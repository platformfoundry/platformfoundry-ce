// Package workflow provides approval workflow and policy gate functionality
// for production deployments and other sensitive operations.
package workflow

import (
	"time"
)

// WorkflowStatus represents the current state of a workflow execution
type WorkflowStatus string

const (
	WorkflowStatusPending       WorkflowStatus = "pending"
	WorkflowStatusConditions    WorkflowStatus = "checking_conditions"
	WorkflowStatusAwaitApproval WorkflowStatus = "awaiting_approval"
	WorkflowStatusApproved      WorkflowStatus = "approved"
	WorkflowStatusRejected      WorkflowStatus = "rejected"
	WorkflowStatusExecuting     WorkflowStatus = "executing"
	WorkflowStatusCompleted     WorkflowStatus = "completed"
	WorkflowStatusFailed        WorkflowStatus = "failed"
	WorkflowStatusRolledBack    WorkflowStatus = "rolled_back"
	WorkflowStatusTimedOut      WorkflowStatus = "timed_out"
	WorkflowStatusBlocked       WorkflowStatus = "blocked"
)

// ConditionType defines types of pre-deployment conditions
type ConditionType string

const (
	ConditionTestsPassing    ConditionType = "tests-passing"
	ConditionSecurityScan    ConditionType = "security-scan"
	ConditionTestCoverage    ConditionType = "test-coverage"
	ConditionPerformanceTest ConditionType = "performance-tests"
	ConditionCustom          ConditionType = "custom"
)

// ConditionStatus represents the result of a condition check
type ConditionStatus string

const (
	ConditionStatusPending ConditionStatus = "pending"
	ConditionStatusPassed  ConditionStatus = "passed"
	ConditionStatusFailed  ConditionStatus = "failed"
	ConditionStatusSkipped ConditionStatus = "skipped"
)

// Workflow defines an approval workflow configuration
type Workflow struct {
	Name         string              `yaml:"name" json:"name"`
	Organization string              `yaml:"organization" json:"organization"`
	Trigger      WorkflowTrigger     `yaml:"trigger" json:"trigger"`
	Conditions   []WorkflowCondition `yaml:"conditions" json:"conditions"`
	Approvals    ApprovalConfig      `yaml:"approvals" json:"approvals"`
	ChangeWindow *ChangeWindowConfig `yaml:"changeWindow,omitempty" json:"changeWindow,omitempty"`
	Rollback     *RollbackConfig     `yaml:"rollback,omitempty" json:"rollback,omitempty"`
	Notifications []NotificationConfig `yaml:"notifications,omitempty" json:"notifications,omitempty"`
}

// WorkflowTrigger defines when a workflow is triggered
type WorkflowTrigger struct {
	Action string            `yaml:"action" json:"action"` // deploy, scale, delete
	Target WorkflowTarget    `yaml:"target" json:"target"`
	Labels map[string]string `yaml:"labels,omitempty" json:"labels,omitempty"`
}

// WorkflowTarget specifies what the workflow applies to
type WorkflowTarget struct {
	Environment string `yaml:"environment,omitempty" json:"environment,omitempty"`
	Service     string `yaml:"service,omitempty" json:"service,omitempty"`
	Team        string `yaml:"team,omitempty" json:"team,omitempty"`
}

// WorkflowCondition defines a pre-deployment condition
type WorkflowCondition struct {
	Type       ConditionType `yaml:"type" json:"type"`
	Required   bool          `yaml:"required" json:"required"`
	Status     string        `yaml:"status,omitempty" json:"status,omitempty"`
	Threshold  int           `yaml:"threshold,omitempty" json:"threshold,omitempty"`
	MaxCritical int          `yaml:"maxCritical,omitempty" json:"maxCritical,omitempty"`
	Custom     *CustomCondition `yaml:"custom,omitempty" json:"custom,omitempty"`
}

// CustomCondition allows custom condition logic
type CustomCondition struct {
	Script   string `yaml:"script,omitempty" json:"script,omitempty"`
	Endpoint string `yaml:"endpoint,omitempty" json:"endpoint,omitempty"`
	Timeout  string `yaml:"timeout,omitempty" json:"timeout,omitempty"`
}

// ApprovalConfig defines approval requirements
type ApprovalConfig struct {
	Required      int           `yaml:"required" json:"required"`
	Roles         []string      `yaml:"roles" json:"roles"`
	Users         []string      `yaml:"users,omitempty" json:"users,omitempty"`
	Timeout       time.Duration `yaml:"timeout" json:"timeout"`
	AllowSelf     bool          `yaml:"allowSelf,omitempty" json:"allowSelf,omitempty"`
	RequireReason bool          `yaml:"requireReason,omitempty" json:"requireReason,omitempty"`
}

// ChangeWindowConfig defines allowed deployment windows
type ChangeWindowConfig struct {
	Allowed []TimeWindow  `yaml:"allowed" json:"allowed"`
	Blocked []BlockedTime `yaml:"blocked,omitempty" json:"blocked,omitempty"`
}

// TimeWindow specifies an allowed time window
type TimeWindow struct {
	Days   []string `yaml:"days" json:"days"`     // Mon, Tue, etc.
	Hours  string   `yaml:"hours" json:"hours"`   // "10:00-16:00"
}

// BlockedTime specifies blocked times
type BlockedTime struct {
	Days   []string `yaml:"days,omitempty" json:"days,omitempty"`
	Dates  []string `yaml:"dates,omitempty" json:"dates,omitempty"` // YYYY-MM-DD
	Reason string   `yaml:"reason,omitempty" json:"reason,omitempty"`
}

// RollbackConfig defines auto-rollback settings
type RollbackConfig struct {
	Enabled    bool                `yaml:"enabled" json:"enabled"`
	Automatic  bool                `yaml:"automatic" json:"automatic"`
	Conditions []RollbackCondition `yaml:"conditions" json:"conditions"`
	Window     time.Duration       `yaml:"window" json:"window"`
}

// RollbackCondition defines when to trigger rollback
type RollbackCondition struct {
	Metric    string  `yaml:"metric" json:"metric"`       // error_rate, latency_p99, etc.
	Threshold float64 `yaml:"threshold" json:"threshold"`
	Unit      string  `yaml:"unit" json:"unit"`           // percent, ms, etc.
}

// NotificationConfig defines notification settings
type NotificationConfig struct {
	Type       string   `yaml:"type" json:"type"` // slack, email, webhook
	Channel    string   `yaml:"channel,omitempty" json:"channel,omitempty"`
	Recipients []string `yaml:"recipients,omitempty" json:"recipients,omitempty"`
	URL        string   `yaml:"url,omitempty" json:"url,omitempty"`
	OnEvents   []string `yaml:"onEvents,omitempty" json:"onEvents,omitempty"`
}

// WorkflowExecution represents an instance of a workflow running
type WorkflowExecution struct {
	ID              string                    `json:"id"`
	WorkflowName    string                    `json:"workflowName"`
	Status          WorkflowStatus            `json:"status"`
	Requester       string                    `json:"requester"`
	RequestedAt     time.Time                 `json:"requestedAt"`
	StartedAt       *time.Time                `json:"startedAt,omitempty"`
	CompletedAt     *time.Time                `json:"completedAt,omitempty"`
	Target          WorkflowTarget            `json:"target"`
	Action          string                    `json:"action"`
	ConditionResults []ConditionResult        `json:"conditionResults"`
	Approvals       []ApprovalRecord          `json:"approvals"`
	RollbackInfo    *RollbackInfo             `json:"rollbackInfo,omitempty"`
	Metadata        map[string]interface{}    `json:"metadata,omitempty"`
	Error           string                    `json:"error,omitempty"`
}

// ConditionResult captures the result of a condition check
type ConditionResult struct {
	Type      ConditionType   `json:"type"`
	Status    ConditionStatus `json:"status"`
	Message   string          `json:"message"`
	CheckedAt time.Time       `json:"checkedAt"`
	Details   map[string]interface{} `json:"details,omitempty"`
}

// ApprovalRecord captures an approval or rejection
type ApprovalRecord struct {
	User      string    `json:"user"`
	Role      string    `json:"role"`
	Decision  string    `json:"decision"` // approved, rejected
	Comment   string    `json:"comment,omitempty"`
	Timestamp time.Time `json:"timestamp"`
}

// RollbackInfo contains information about a rollback
type RollbackInfo struct {
	Triggered   bool      `json:"triggered"`
	Reason      string    `json:"reason,omitempty"`
	PreviousVersion string `json:"previousVersion,omitempty"`
	TriggeredAt *time.Time `json:"triggeredAt,omitempty"`
	CompletedAt *time.Time `json:"completedAt,omitempty"`
}
