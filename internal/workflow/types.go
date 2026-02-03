// Package workflow provides approval workflow and policy gate functionality
// for production deployments and other sensitive operations.
package workflow

import (
	"time"
)

// StepType defines types of workflow steps
type StepType string

const (
	StepTypeShell    StepType = "shell"
	StepTypeHTTP     StepType = "http"
	StepTypeInfra    StepType = "infra"
	StepTypePolicy   StepType = "policy"
	StepTypeSecrets  StepType = "secrets"
	StepTypeNotify   StepType = "notify"
	StepTypeApproval StepType = "approval"
)

// StepStatus represents the current state of a step execution
type StepStatus string

const (
	StepStatusPending   StepStatus = "pending"
	StepStatusRunning   StepStatus = "running"
	StepStatusCompleted StepStatus = "completed"
	StepStatusFailed    StepStatus = "failed"
	StepStatusSkipped   StepStatus = "skipped"
	StepStatusCancelled StepStatus = "cancelled"
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
	Name          string               `yaml:"name" json:"name"`
	Organization  string               `yaml:"organization" json:"organization"`
	Trigger       WorkflowTrigger      `yaml:"trigger" json:"trigger"`
	Conditions    []WorkflowCondition  `yaml:"conditions" json:"conditions"`
	Approvals     ApprovalConfig       `yaml:"approvals" json:"approvals"`
	ChangeWindow  *ChangeWindowConfig  `yaml:"changeWindow,omitempty" json:"changeWindow,omitempty"`
	Rollback      *RollbackConfig      `yaml:"rollback,omitempty" json:"rollback,omitempty"`
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
	Type        ConditionType    `yaml:"type" json:"type"`
	Required    bool             `yaml:"required" json:"required"`
	Status      string           `yaml:"status,omitempty" json:"status,omitempty"`
	Threshold   int              `yaml:"threshold,omitempty" json:"threshold,omitempty"`
	MaxCritical int              `yaml:"maxCritical,omitempty" json:"maxCritical,omitempty"`
	Custom      *CustomCondition `yaml:"custom,omitempty" json:"custom,omitempty"`
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
	Days  []string `yaml:"days" json:"days"`   // Mon, Tue, etc.
	Hours string   `yaml:"hours" json:"hours"` // "10:00-16:00"
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
	Metric    string  `yaml:"metric" json:"metric"` // error_rate, latency_p99, etc.
	Threshold float64 `yaml:"threshold" json:"threshold"`
	Unit      string  `yaml:"unit" json:"unit"` // percent, ms, etc.
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
	ID               string                 `json:"id"`
	WorkflowName     string                 `json:"workflowName"`
	Status           WorkflowStatus         `json:"status"`
	Requester        string                 `json:"requester"`
	RequestedAt      time.Time              `json:"requestedAt"`
	StartedAt        *time.Time             `json:"startedAt,omitempty"`
	CompletedAt      *time.Time             `json:"completedAt,omitempty"`
	Target           WorkflowTarget         `json:"target"`
	Action           string                 `json:"action"`
	ConditionResults []ConditionResult      `json:"conditionResults"`
	Approvals        []ApprovalRecord       `json:"approvals"`
	RollbackInfo     *RollbackInfo          `json:"rollbackInfo,omitempty"`
	Metadata         map[string]interface{} `json:"metadata,omitempty"`
	Error            string                 `json:"error,omitempty"`
}

// ConditionResult captures the result of a condition check
type ConditionResult struct {
	Type      ConditionType          `json:"type"`
	Status    ConditionStatus        `json:"status"`
	Message   string                 `json:"message"`
	CheckedAt time.Time              `json:"checkedAt"`
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
	Triggered       bool       `json:"triggered"`
	Reason          string     `json:"reason,omitempty"`
	PreviousVersion string     `json:"previousVersion,omitempty"`
	TriggeredAt     *time.Time `json:"triggeredAt,omitempty"`
	CompletedAt     *time.Time `json:"completedAt,omitempty"`
}

// DAGWorkflow represents a YAML-defined DAG-based workflow
type DAGWorkflow struct {
	APIVersion string           `yaml:"apiVersion" json:"apiVersion"`
	Kind       string           `yaml:"kind" json:"kind"`
	Metadata   WorkflowMetadata `yaml:"metadata" json:"metadata"`
	Spec       DAGWorkflowSpec  `yaml:"spec" json:"spec"`
}

// WorkflowMetadata contains workflow metadata
type WorkflowMetadata struct {
	Name        string            `yaml:"name" json:"name"`
	Namespace   string            `yaml:"namespace,omitempty" json:"namespace,omitempty"`
	Labels      map[string]string `yaml:"labels,omitempty" json:"labels,omitempty"`
	Annotations map[string]string `yaml:"annotations,omitempty" json:"annotations,omitempty"`
}

// DAGWorkflowSpec defines the workflow specification
type DAGWorkflowSpec struct {
	Description   string               `yaml:"description,omitempty" json:"description,omitempty"`
	Triggers      []TriggerSpec        `yaml:"triggers,omitempty" json:"triggers,omitempty"`
	Inputs        []InputSpec          `yaml:"inputs,omitempty" json:"inputs,omitempty"`
	Timeout       string               `yaml:"timeout,omitempty" json:"timeout,omitempty"`
	Steps         []StepSpec           `yaml:"steps" json:"steps"`
	OnSuccess     []StepSpec           `yaml:"onSuccess,omitempty" json:"onSuccess,omitempty"`
	OnFailure     []StepSpec           `yaml:"onFailure,omitempty" json:"onFailure,omitempty"`
	Notifications []NotificationConfig `yaml:"notifications,omitempty" json:"notifications,omitempty"`
	Concurrency   *ConcurrencyConfig   `yaml:"concurrency,omitempty" json:"concurrency,omitempty"`
}

// TriggerSpec defines a workflow trigger
type TriggerSpec struct {
	Type     string          `yaml:"type" json:"type"` // manual, schedule, webhook, event
	Name     string          `yaml:"name,omitempty" json:"name,omitempty"`
	Cron     string          `yaml:"cron,omitempty" json:"cron,omitempty"`
	Webhook  *WebhookTrigger `yaml:"webhook,omitempty" json:"webhook,omitempty"`
	Event    *EventTrigger   `yaml:"event,omitempty" json:"event,omitempty"`
	Disabled bool            `yaml:"disabled,omitempty" json:"disabled,omitempty"`
}

// WebhookTrigger defines webhook trigger configuration
type WebhookTrigger struct {
	Path    string            `yaml:"path,omitempty" json:"path,omitempty"`
	Secret  string            `yaml:"secret,omitempty" json:"secret,omitempty"`
	Headers map[string]string `yaml:"headers,omitempty" json:"headers,omitempty"`
}

// EventTrigger defines event-based trigger configuration
type EventTrigger struct {
	Source string   `yaml:"source" json:"source"`
	Types  []string `yaml:"types" json:"types"`
}

// InputSpec defines a workflow input parameter
type InputSpec struct {
	Name        string      `yaml:"name" json:"name"`
	Type        string      `yaml:"type" json:"type"` // string, number, boolean, array, object
	Required    bool        `yaml:"required,omitempty" json:"required,omitempty"`
	Default     interface{} `yaml:"default,omitempty" json:"default,omitempty"`
	Description string      `yaml:"description,omitempty" json:"description,omitempty"`
	Enum        []string    `yaml:"enum,omitempty" json:"enum,omitempty"`
}

// StepSpec defines a workflow step
type StepSpec struct {
	ID         string                 `yaml:"id" json:"id"`
	Name       string                 `yaml:"name,omitempty" json:"name,omitempty"`
	Type       StepType               `yaml:"type" json:"type"`
	DependsOn  []string               `yaml:"dependsOn,omitempty" json:"dependsOn,omitempty"`
	Condition  string                 `yaml:"condition,omitempty" json:"condition,omitempty"`
	Config     map[string]interface{} `yaml:"config,omitempty" json:"config,omitempty"`
	Timeout    string                 `yaml:"timeout,omitempty" json:"timeout,omitempty"`
	Retries    *RetryConfig           `yaml:"retries,omitempty" json:"retries,omitempty"`
	ContinueOn *ContinueOnConfig      `yaml:"continueOn,omitempty" json:"continueOn,omitempty"`
	Env        map[string]string      `yaml:"env,omitempty" json:"env,omitempty"`
	Outputs    []OutputSpec           `yaml:"outputs,omitempty" json:"outputs,omitempty"`
}

// RetryConfig defines retry behavior for a step
type RetryConfig struct {
	MaxAttempts int    `yaml:"maxAttempts" json:"maxAttempts"`
	Delay       string `yaml:"delay,omitempty" json:"delay,omitempty"`
	Backoff     string `yaml:"backoff,omitempty" json:"backoff,omitempty"` // linear, exponential
}

// ContinueOnConfig defines when to continue execution despite step status
type ContinueOnConfig struct {
	Failure bool `yaml:"failure,omitempty" json:"failure,omitempty"`
	Error   bool `yaml:"error,omitempty" json:"error,omitempty"`
}

// OutputSpec defines a step output
type OutputSpec struct {
	Name string `yaml:"name" json:"name"`
	From string `yaml:"from,omitempty" json:"from,omitempty"` // JSONPath or expression
}

// ConcurrencyConfig defines workflow concurrency settings
type ConcurrencyConfig struct {
	Group      string `yaml:"group,omitempty" json:"group,omitempty"`
	MaxRunning int    `yaml:"maxRunning,omitempty" json:"maxRunning,omitempty"`
	CancelPrev bool   `yaml:"cancelPrev,omitempty" json:"cancelPrev,omitempty"`
}

// DAGExecution represents an execution of a DAG workflow
type DAGExecution struct {
	ID           string                    `json:"id"`
	WorkflowName string                    `json:"workflowName"`
	Status       WorkflowStatus            `json:"status"`
	Trigger      string                    `json:"trigger"`
	Inputs       map[string]interface{}    `json:"inputs,omitempty"`
	Steps        map[string]*StepExecution `json:"steps"`
	StartedAt    time.Time                 `json:"startedAt"`
	CompletedAt  *time.Time                `json:"completedAt,omitempty"`
	Error        string                    `json:"error,omitempty"`
	Outputs      map[string]interface{}    `json:"outputs,omitempty"`
}

// StepExecution represents the execution state of a single step
type StepExecution struct {
	ID          string                 `json:"id"`
	StepID      string                 `json:"stepId"`
	Status      StepStatus             `json:"status"`
	StartedAt   *time.Time             `json:"startedAt,omitempty"`
	CompletedAt *time.Time             `json:"completedAt,omitempty"`
	Attempt     int                    `json:"attempt"`
	Inputs      map[string]interface{} `json:"inputs,omitempty"`
	Outputs     map[string]interface{} `json:"outputs,omitempty"`
	Logs        []StepLog              `json:"logs,omitempty"`
	Error       string                 `json:"error,omitempty"`
}

// StepLog represents a log entry from step execution
type StepLog struct {
	Time    time.Time `json:"time"`
	Level   string    `json:"level"`
	Message string    `json:"message"`
}

// StepResult represents the result of a step execution
type StepResult struct {
	Status   StepStatus             `json:"status"`
	Outputs  map[string]interface{} `json:"outputs,omitempty"`
	Logs     []StepLog              `json:"logs,omitempty"`
	Error    error                  `json:"-"`
	ErrorMsg string                 `json:"error,omitempty"`
}
