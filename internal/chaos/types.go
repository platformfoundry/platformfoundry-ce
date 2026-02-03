package chaos

import (
	"time"
)

// ChaosExperiment defines a chaos engineering experiment
type ChaosExperiment struct {
	APIVersion string                 `json:"apiVersion" yaml:"apiVersion"`
	Kind       string                 `json:"kind" yaml:"kind"`
	Metadata   ExperimentMetadata     `json:"metadata" yaml:"metadata"`
	Spec       ChaosExperimentSpec    `json:"spec" yaml:"spec"`
	Status     *ChaosExperimentStatus `json:"status,omitempty" yaml:"status,omitempty"`
}

// ExperimentMetadata contains experiment identification
type ExperimentMetadata struct {
	Name        string            `json:"name" yaml:"name"`
	Description string            `json:"description,omitempty" yaml:"description,omitempty"`
	Labels      map[string]string `json:"labels,omitempty" yaml:"labels,omitempty"`
	Annotations map[string]string `json:"annotations,omitempty" yaml:"annotations,omitempty"`
	CreatedAt   time.Time         `json:"createdAt,omitempty" yaml:"createdAt,omitempty"`
}

// ChaosExperimentSpec defines the experiment specification
type ChaosExperimentSpec struct {
	Target        ExperimentTarget     `json:"target" yaml:"target"`
	Experiments   []ExperimentAction   `json:"experiments" yaml:"experiments"`
	SteadyState   []SteadyStateCheck   `json:"steadyState,omitempty" yaml:"steadyState,omitempty"`
	Schedule      *ScheduleConfig      `json:"schedule,omitempty" yaml:"schedule,omitempty"`
	Rollback      *RollbackConfig      `json:"rollback,omitempty" yaml:"rollback,omitempty"`
	Notifications []NotificationConfig `json:"notifications,omitempty" yaml:"notifications,omitempty"`
	DryRun        bool                 `json:"dryRun,omitempty" yaml:"dryRun,omitempty"`
}

// ExperimentTarget defines what to target
type ExperimentTarget struct {
	Kind        string            `json:"kind" yaml:"kind"` // Deployment, Pod, Node, Service
	Name        string            `json:"name" yaml:"name"`
	Namespace   string            `json:"namespace,omitempty" yaml:"namespace,omitempty"`
	Environment string            `json:"environment,omitempty" yaml:"environment,omitempty"`
	Selector    map[string]string `json:"selector,omitempty" yaml:"selector,omitempty"`
	Percentage  int               `json:"percentage,omitempty" yaml:"percentage,omitempty"` // % of targets to affect
}

// ExperimentAction defines a chaos action
type ExperimentAction struct {
	Type       ExperimentType         `json:"type" yaml:"type"`
	Duration   string                 `json:"duration" yaml:"duration"`
	Parameters map[string]interface{} `json:"parameters,omitempty" yaml:"parameters,omitempty"`
}

// ExperimentType defines types of chaos experiments
type ExperimentType string

const (
	// Pod-level experiments
	ExperimentTypePodFailure    ExperimentType = "pod-failure"
	ExperimentTypePodKill       ExperimentType = "pod-kill"
	ExperimentTypeContainerKill ExperimentType = "container-kill"

	// Network experiments
	ExperimentTypeNetworkLatency    ExperimentType = "network-latency"
	ExperimentTypeNetworkLoss       ExperimentType = "network-loss"
	ExperimentTypeNetworkCorruption ExperimentType = "network-corruption"
	ExperimentTypeNetworkPartition  ExperimentType = "network-partition"
	ExperimentTypeNetworkBandwidth  ExperimentType = "network-bandwidth"
	ExperimentTypeDNSFailure        ExperimentType = "dns-failure"

	// Resource experiments
	ExperimentTypeCPUStress    ExperimentType = "cpu-stress"
	ExperimentTypeMemoryStress ExperimentType = "memory-stress"
	ExperimentTypeDiskStress   ExperimentType = "disk-stress"
	ExperimentTypeDiskFill     ExperimentType = "disk-fill"
	ExperimentTypeIOStress     ExperimentType = "io-stress"

	// Time experiments
	ExperimentTypeTimeChaos ExperimentType = "time-chaos"

	// HTTP experiments
	ExperimentTypeHTTPAbort ExperimentType = "http-abort"
	ExperimentTypeHTTPDelay ExperimentType = "http-delay"

	// Process experiments
	ExperimentTypeProcessKill ExperimentType = "process-kill"
	ExperimentTypeProcessStop ExperimentType = "process-stop"

	// Node experiments
	ExperimentTypeNodeDrain   ExperimentType = "node-drain"
	ExperimentTypeNodeCordon  ExperimentType = "node-cordon"
	ExperimentTypeNodeRestart ExperimentType = "node-restart"
)

// SteadyStateCheck defines a steady state hypothesis check
type SteadyStateCheck struct {
	Name      string  `json:"name" yaml:"name"`
	Metric    string  `json:"metric" yaml:"metric"`
	Threshold string  `json:"threshold" yaml:"threshold"`
	Tolerance float64 `json:"tolerance,omitempty" yaml:"tolerance,omitempty"`
	Provider  string  `json:"provider,omitempty" yaml:"provider,omitempty"` // prometheus, datadog, etc.
}

// ScheduleConfig defines experiment scheduling
type ScheduleConfig struct {
	Cron              string `json:"cron,omitempty" yaml:"cron,omitempty"`
	Timezone          string `json:"timezone,omitempty" yaml:"timezone,omitempty"`
	ConcurrencyPolicy string `json:"concurrencyPolicy,omitempty" yaml:"concurrencyPolicy,omitempty"` // Allow, Forbid, Replace
	StartingDeadline  string `json:"startingDeadline,omitempty" yaml:"startingDeadline,omitempty"`
}

// RollbackConfig defines rollback behavior
type RollbackConfig struct {
	OnFailure              bool `json:"onFailure" yaml:"onFailure"`
	OnSteadyStateViolation bool `json:"onSteadyStateViolation,omitempty" yaml:"onSteadyStateViolation,omitempty"`
	Manual                 bool `json:"manual,omitempty" yaml:"manual,omitempty"`
}

// NotificationConfig defines experiment notifications
type NotificationConfig struct {
	Channel string   `json:"channel" yaml:"channel"` // slack, webhook, email
	Target  string   `json:"target" yaml:"target"`
	Events  []string `json:"events,omitempty" yaml:"events,omitempty"` // started, completed, failed, steady-state-violated
}

// ChaosExperimentStatus tracks experiment status
type ChaosExperimentStatus struct {
	Phase              ExperimentPhase       `json:"phase" yaml:"phase"`
	StartedAt          *time.Time            `json:"startedAt,omitempty" yaml:"startedAt,omitempty"`
	CompletedAt        *time.Time            `json:"completedAt,omitempty" yaml:"completedAt,omitempty"`
	Duration           string                `json:"duration,omitempty" yaml:"duration,omitempty"`
	SteadyStateResults []SteadyStateResult   `json:"steadyStateResults,omitempty" yaml:"steadyStateResults,omitempty"`
	ExperimentResults  []ExperimentResult    `json:"experimentResults,omitempty" yaml:"experimentResults,omitempty"`
	RolledBack         bool                  `json:"rolledBack,omitempty" yaml:"rolledBack,omitempty"`
	Error              string                `json:"error,omitempty" yaml:"error,omitempty"`
	Conditions         []ExperimentCondition `json:"conditions,omitempty" yaml:"conditions,omitempty"`
}

// ExperimentPhase defines experiment phase
type ExperimentPhase string

const (
	ExperimentPhasePending    ExperimentPhase = "Pending"
	ExperimentPhaseRunning    ExperimentPhase = "Running"
	ExperimentPhasePaused     ExperimentPhase = "Paused"
	ExperimentPhaseCompleted  ExperimentPhase = "Completed"
	ExperimentPhaseFailed     ExperimentPhase = "Failed"
	ExperimentPhaseRolledBack ExperimentPhase = "RolledBack"
)

// SteadyStateResult captures steady state check results
type SteadyStateResult struct {
	Name      string    `json:"name" yaml:"name"`
	Passed    bool      `json:"passed" yaml:"passed"`
	Value     float64   `json:"value,omitempty" yaml:"value,omitempty"`
	Threshold string    `json:"threshold" yaml:"threshold"`
	Message   string    `json:"message,omitempty" yaml:"message,omitempty"`
	CheckedAt time.Time `json:"checkedAt" yaml:"checkedAt"`
}

// ExperimentResult captures individual experiment results
type ExperimentResult struct {
	Type        ExperimentType     `json:"type" yaml:"type"`
	StartedAt   time.Time          `json:"startedAt" yaml:"startedAt"`
	CompletedAt *time.Time         `json:"completedAt,omitempty" yaml:"completedAt,omitempty"`
	Targets     []string           `json:"targets,omitempty" yaml:"targets,omitempty"`
	Success     bool               `json:"success" yaml:"success"`
	Error       string             `json:"error,omitempty" yaml:"error,omitempty"`
	Metrics     map[string]float64 `json:"metrics,omitempty" yaml:"metrics,omitempty"`
}

// ExperimentCondition represents an experiment condition
type ExperimentCondition struct {
	Type               string    `json:"type" yaml:"type"`
	Status             string    `json:"status" yaml:"status"`
	LastTransitionTime time.Time `json:"lastTransitionTime" yaml:"lastTransitionTime"`
	Reason             string    `json:"reason,omitempty" yaml:"reason,omitempty"`
	Message            string    `json:"message,omitempty" yaml:"message,omitempty"`
}

// GameDay defines a chaos engineering game day
type GameDay struct {
	APIVersion string             `json:"apiVersion" yaml:"apiVersion"`
	Kind       string             `json:"kind" yaml:"kind"`
	Metadata   ExperimentMetadata `json:"metadata" yaml:"metadata"`
	Spec       GameDaySpec        `json:"spec" yaml:"spec"`
	Status     *GameDayStatus     `json:"status,omitempty" yaml:"status,omitempty"`
}

// GameDaySpec defines game day specification
type GameDaySpec struct {
	Scenario    string           `json:"scenario" yaml:"scenario"`
	Objectives  []string         `json:"objectives" yaml:"objectives"`
	Experiments []string         `json:"experiments" yaml:"experiments"`               // References to ChaosExperiment names
	RunOrder    string           `json:"runOrder,omitempty" yaml:"runOrder,omitempty"` // sequential, parallel
	Teams       []GameDayTeam    `json:"teams,omitempty" yaml:"teams,omitempty"`
	Schedule    *GameDaySchedule `json:"schedule,omitempty" yaml:"schedule,omitempty"`
	Briefing    string           `json:"briefing,omitempty" yaml:"briefing,omitempty"`
	Postmortem  bool             `json:"postmortem,omitempty" yaml:"postmortem,omitempty"`
}

// GameDayTeam defines a participating team
type GameDayTeam struct {
	Name         string   `json:"name" yaml:"name"`
	Participants []string `json:"participants,omitempty" yaml:"participants,omitempty"`
	Role         string   `json:"role" yaml:"role"` // conductor, observer, responder
}

// GameDaySchedule defines game day timing
type GameDaySchedule struct {
	StartTime time.Time `json:"startTime" yaml:"startTime"`
	Duration  string    `json:"duration" yaml:"duration"`
	Timezone  string    `json:"timezone,omitempty" yaml:"timezone,omitempty"`
}

// GameDayStatus tracks game day status
type GameDayStatus struct {
	Phase             GameDayPhase     `json:"phase" yaml:"phase"`
	StartedAt         *time.Time       `json:"startedAt,omitempty" yaml:"startedAt,omitempty"`
	CompletedAt       *time.Time       `json:"completedAt,omitempty" yaml:"completedAt,omitempty"`
	ExperimentsRun    int              `json:"experimentsRun" yaml:"experimentsRun"`
	ExperimentsFailed int              `json:"experimentsFailed" yaml:"experimentsFailed"`
	Findings          []GameDayFinding `json:"findings,omitempty" yaml:"findings,omitempty"`
	PostmortemURL     string           `json:"postmortemUrl,omitempty" yaml:"postmortemUrl,omitempty"`
}

// GameDayPhase defines game day phase
type GameDayPhase string

const (
	GameDayPhaseScheduled  GameDayPhase = "Scheduled"
	GameDayPhaseBriefing   GameDayPhase = "Briefing"
	GameDayPhaseRunning    GameDayPhase = "Running"
	GameDayPhaseDebriefing GameDayPhase = "Debriefing"
	GameDayPhaseCompleted  GameDayPhase = "Completed"
)

// GameDayFinding represents a finding from the game day
type GameDayFinding struct {
	Type        string `json:"type" yaml:"type"` // weakness, improvement, success
	Title       string `json:"title" yaml:"title"`
	Description string `json:"description" yaml:"description"`
	Severity    string `json:"severity,omitempty" yaml:"severity,omitempty"` // high, medium, low
	ActionItem  string `json:"actionItem,omitempty" yaml:"actionItem,omitempty"`
}

// ChaosReport represents a chaos engineering report
type ChaosReport struct {
	GeneratedAt       time.Time        `json:"generatedAt" yaml:"generatedAt"`
	Period            string           `json:"period" yaml:"period"`
	TotalExperiments  int              `json:"totalExperiments" yaml:"totalExperiments"`
	SuccessRate       float64          `json:"successRate" yaml:"successRate"`
	MeanTimeToDetect  string           `json:"meanTimeToDetect,omitempty" yaml:"meanTimeToDetect,omitempty"`
	MeanTimeToRecover string           `json:"meanTimeToRecover,omitempty" yaml:"meanTimeToRecover,omitempty"`
	TopFindings       []GameDayFinding `json:"topFindings,omitempty" yaml:"topFindings,omitempty"`
	ByType            map[string]int   `json:"byType,omitempty" yaml:"byType,omitempty"`
	Recommendations   []string         `json:"recommendations,omitempty" yaml:"recommendations,omitempty"`
}

// ExperimentTemplate defines reusable experiment templates
type ExperimentTemplate struct {
	Name        string              `json:"name" yaml:"name"`
	Description string              `json:"description,omitempty" yaml:"description,omitempty"`
	Category    string              `json:"category,omitempty" yaml:"category,omitempty"`
	Spec        ChaosExperimentSpec `json:"spec" yaml:"spec"`
}
