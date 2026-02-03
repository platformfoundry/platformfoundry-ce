package types

import (
	"time"
)

// ChaosExperiment defines a chaos engineering experiment
type ChaosExperiment struct {
	APIVersion string                 `yaml:"apiVersion" json:"apiVersion"`
	Kind       string                 `yaml:"kind" json:"kind"`
	Metadata   ChaosMetadata          `yaml:"metadata" json:"metadata"`
	Spec       ChaosExperimentSpec    `yaml:"spec" json:"spec"`
	Status     *ChaosExperimentStatus `yaml:"status,omitempty" json:"status,omitempty"`
}

// ChaosMetadata contains metadata for the experiment
type ChaosMetadata struct {
	Name        string            `yaml:"name" json:"name"`
	Namespace   string            `yaml:"namespace,omitempty" json:"namespace,omitempty"`
	Labels      map[string]string `yaml:"labels,omitempty" json:"labels,omitempty"`
	Annotations map[string]string `yaml:"annotations,omitempty" json:"annotations,omitempty"`
}

// ChaosExperimentSpec defines the experiment specification
type ChaosExperimentSpec struct {
	Target      ChaosTarget      `yaml:"target" json:"target"`
	Experiments []ChaosAction    `yaml:"experiments" json:"experiments"`
	Schedule    *ChaosSchedule   `yaml:"schedule,omitempty" json:"schedule,omitempty"`
	Safety      ChaosSafetyRules `yaml:"safety" json:"safety"`
	Duration    string           `yaml:"duration,omitempty" json:"duration,omitempty"`
}

// ChaosTarget defines what the experiment targets
type ChaosTarget struct {
	Service     string            `yaml:"service,omitempty" json:"service,omitempty"`
	Namespace   string            `yaml:"namespace,omitempty" json:"namespace,omitempty"`
	Environment string            `yaml:"environment" json:"environment"`
	Selector    map[string]string `yaml:"selector,omitempty" json:"selector,omitempty"`
}

// ChaosAction defines a specific chaos action
type ChaosAction struct {
	Name        string                 `yaml:"name" json:"name"`
	Type        ChaosActionType        `yaml:"type" json:"type"`
	Duration    string                 `yaml:"duration" json:"duration"`
	Probability float64                `yaml:"probability,omitempty" json:"probability,omitempty"`
	Parameters  map[string]interface{} `yaml:"parameters,omitempty" json:"parameters,omitempty"`
}

// ChaosActionType represents the type of chaos action
type ChaosActionType string

const (
	// Pod chaos
	ChaosActionPodKill       ChaosActionType = "pod-kill"
	ChaosActionPodFailure    ChaosActionType = "pod-failure"
	ChaosActionContainerKill ChaosActionType = "container-kill"

	// Network chaos
	ChaosActionNetworkDelay     ChaosActionType = "network-delay"
	ChaosActionNetworkLoss      ChaosActionType = "network-loss"
	ChaosActionNetworkPartition ChaosActionType = "network-partition"
	ChaosActionNetworkCorrupt   ChaosActionType = "network-corrupt"
	ChaosActionNetworkDuplicate ChaosActionType = "network-duplicate"

	// Stress chaos
	ChaosActionCPUStress    ChaosActionType = "cpu-stress"
	ChaosActionMemoryStress ChaosActionType = "memory-stress"
	ChaosActionIOStress     ChaosActionType = "io-stress"

	// Service chaos
	ChaosActionServiceUnavailable ChaosActionType = "service-unavailable"
	ChaosActionHTTPError          ChaosActionType = "http-error"
	ChaosActionHTTPDelay          ChaosActionType = "http-delay"

	// Infrastructure chaos
	ChaosActionNodeDrain   ChaosActionType = "node-drain"
	ChaosActionNodeFailure ChaosActionType = "node-failure"
	ChaosActionZoneFailure ChaosActionType = "zone-failure"

	// DNS chaos
	ChaosActionDNSError ChaosActionType = "dns-error"
	ChaosActionDNSDelay ChaosActionType = "dns-delay"
)

// ChaosSchedule defines when experiments run
type ChaosSchedule struct {
	Cron       string   `yaml:"cron,omitempty" json:"cron,omitempty"`
	Timezone   string   `yaml:"timezone,omitempty" json:"timezone,omitempty"`
	StartTime  string   `yaml:"startTime,omitempty" json:"startTime,omitempty"`
	EndTime    string   `yaml:"endTime,omitempty" json:"endTime,omitempty"`
	DaysOfWeek []string `yaml:"daysOfWeek,omitempty" json:"daysOfWeek,omitempty"`
}

// ChaosSafetyRules defines safety controls
type ChaosSafetyRules struct {
	MaxImpact           string            `yaml:"maxImpact" json:"maxImpact"` // percentage
	RollbackOnError     bool              `yaml:"rollbackOnError" json:"rollbackOnError"`
	HealthCheckInterval string            `yaml:"healthCheckInterval" json:"healthCheckInterval"`
	HealthCheck         *ChaosHealthCheck `yaml:"healthCheck,omitempty" json:"healthCheck,omitempty"`
	Paused              bool              `yaml:"paused,omitempty" json:"paused,omitempty"`
	StopOnFailure       bool              `yaml:"stopOnFailure,omitempty" json:"stopOnFailure,omitempty"`
	ConcurrencyPolicy   string            `yaml:"concurrencyPolicy,omitempty" json:"concurrencyPolicy,omitempty"` // Forbid, Allow
}

// ChaosHealthCheck defines how to verify system health
type ChaosHealthCheck struct {
	Type     string `yaml:"type" json:"type"` // http, tcp, exec
	Endpoint string `yaml:"endpoint,omitempty" json:"endpoint,omitempty"`
	Timeout  string `yaml:"timeout,omitempty" json:"timeout,omitempty"`
	Interval string `yaml:"interval,omitempty" json:"interval,omitempty"`
}

// ChaosExperimentStatus represents the current state
type ChaosExperimentStatus struct {
	Phase          ChaosPhase        `yaml:"phase" json:"phase"`
	StartTime      *time.Time        `yaml:"startTime,omitempty" json:"startTime,omitempty"`
	EndTime        *time.Time        `yaml:"endTime,omitempty" json:"endTime,omitempty"`
	SuccessfulRuns int               `yaml:"successfulRuns" json:"successfulRuns"`
	FailedRuns     int               `yaml:"failedRuns" json:"failedRuns"`
	LastRunTime    *time.Time        `yaml:"lastRunTime,omitempty" json:"lastRunTime,omitempty"`
	LastRunResult  string            `yaml:"lastRunResult,omitempty" json:"lastRunResult,omitempty"`
	CurrentAction  string            `yaml:"currentAction,omitempty" json:"currentAction,omitempty"`
	Conditions     []ChaosCondition  `yaml:"conditions,omitempty" json:"conditions,omitempty"`
	History        []ChaosRunHistory `yaml:"history,omitempty" json:"history,omitempty"`
}

// ChaosPhase represents the experiment phase
type ChaosPhase string

const (
	ChaosPhaseCreated   ChaosPhase = "Created"
	ChaosPhaseScheduled ChaosPhase = "Scheduled"
	ChaosPhaseRunning   ChaosPhase = "Running"
	ChaosPhasePaused    ChaosPhase = "Paused"
	ChaosPhaseCompleted ChaosPhase = "Completed"
	ChaosPhaseFailed    ChaosPhase = "Failed"
	ChaosPhaseAborted   ChaosPhase = "Aborted"
)

// ChaosCondition represents a status condition
type ChaosCondition struct {
	Type               string    `yaml:"type" json:"type"`
	Status             string    `yaml:"status" json:"status"`
	LastTransitionTime time.Time `yaml:"lastTransitionTime" json:"lastTransitionTime"`
	Reason             string    `yaml:"reason,omitempty" json:"reason,omitempty"`
	Message            string    `yaml:"message,omitempty" json:"message,omitempty"`
}

// ChaosRunHistory records past experiment runs
type ChaosRunHistory struct {
	RunID     string         `yaml:"runId" json:"runId"`
	StartTime time.Time      `yaml:"startTime" json:"startTime"`
	EndTime   *time.Time     `yaml:"endTime,omitempty" json:"endTime,omitempty"`
	Result    string         `yaml:"result" json:"result"` // success, failed, aborted
	Actions   []ActionResult `yaml:"actions" json:"actions"`
	Message   string         `yaml:"message,omitempty" json:"message,omitempty"`
}

// ActionResult records the result of a chaos action
type ActionResult struct {
	Name      string                 `yaml:"name" json:"name"`
	Type      string                 `yaml:"type" json:"type"`
	StartTime time.Time              `yaml:"startTime" json:"startTime"`
	EndTime   time.Time              `yaml:"endTime" json:"endTime"`
	Result    string                 `yaml:"result" json:"result"`
	Message   string                 `yaml:"message,omitempty" json:"message,omitempty"`
	Metrics   map[string]interface{} `yaml:"metrics,omitempty" json:"metrics,omitempty"`
}

// ChaosReport summarizes experiment results
type ChaosReport struct {
	Experiment        string         `json:"experiment"`
	Environment       string         `json:"environment"`
	StartTime         time.Time      `json:"startTime"`
	EndTime           time.Time      `json:"endTime"`
	Duration          string         `json:"duration"`
	TotalActions      int            `json:"totalActions"`
	SuccessfulActions int            `json:"successfulActions"`
	FailedActions     int            `json:"failedActions"`
	OverallResult     string         `json:"overallResult"`
	Findings          []ChaosFinding `json:"findings"`
	Recommendations   []string       `json:"recommendations"`
}

// ChaosFinding represents a discovery from the experiment
type ChaosFinding struct {
	Severity    string `json:"severity"` // critical, high, medium, low
	Component   string `json:"component"`
	Description string `json:"description"`
	Impact      string `json:"impact"`
	Remediation string `json:"remediation,omitempty"`
}
