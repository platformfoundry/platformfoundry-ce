package types

import (
	"time"
)

// GitOpsConfigV2 defines an advanced GitOps configuration for platform state management
// This extends the basic GitOpsConfig with full workflow support
type GitOpsConfigV2 struct {
	APIVersion string              `yaml:"apiVersion" json:"apiVersion"`
	Kind       string              `yaml:"kind" json:"kind"`
	Metadata   GitOpsMetadata      `yaml:"metadata" json:"metadata"`
	Spec       GitOpsSpecV2        `yaml:"spec" json:"spec"`
	Status     *GitOpsStatusV2     `yaml:"status,omitempty" json:"status,omitempty"`
}

// GitOpsMetadata contains metadata for the GitOps configuration
type GitOpsMetadata struct {
	Name        string            `yaml:"name" json:"name"`
	Namespace   string            `yaml:"namespace,omitempty" json:"namespace,omitempty"`
	Labels      map[string]string `yaml:"labels,omitempty" json:"labels,omitempty"`
	Annotations map[string]string `yaml:"annotations,omitempty" json:"annotations,omitempty"`
}

// GitOpsSpecV2 defines the desired GitOps configuration (advanced version)
type GitOpsSpecV2 struct {
	Repository       GitOpsRepository      `yaml:"repository" json:"repository"`
	PullRequest      PullRequestConfig     `yaml:"pullRequest,omitempty" json:"pullRequest,omitempty"`
	Sync             GitOpsSyncConfig      `yaml:"sync" json:"sync"`
	Notifications    GitOpsNotifications   `yaml:"notifications,omitempty" json:"notifications,omitempty"`
	Environments     []GitOpsEnvironment   `yaml:"environments,omitempty" json:"environments,omitempty"`
}

// GitOpsRepository defines the Git repository settings
type GitOpsRepository struct {
	URL           string               `yaml:"url" json:"url"`
	Branch        string               `yaml:"branch" json:"branch"`
	Path          string               `yaml:"path,omitempty" json:"path,omitempty"`
	SecretRef     string               `yaml:"secretRef,omitempty" json:"secretRef,omitempty"`
	Provider      string               `yaml:"provider,omitempty" json:"provider,omitempty"` // github, gitlab, bitbucket
	Authentication *GitOpsAuthConfig   `yaml:"authentication,omitempty" json:"authentication,omitempty"`
}

// GitOpsAuthConfig defines authentication settings for the repository
type GitOpsAuthConfig struct {
	Type        string `yaml:"type" json:"type"` // ssh, token, basic
	SecretName  string `yaml:"secretName,omitempty" json:"secretName,omitempty"`
	TokenEnvVar string `yaml:"tokenEnvVar,omitempty" json:"tokenEnvVar,omitempty"`
}

// PullRequestConfig defines PR-based workflow settings
type PullRequestConfig struct {
	Enabled           bool     `yaml:"enabled" json:"enabled"`
	AutoMerge         bool     `yaml:"autoMerge" json:"autoMerge"`
	RequiredApprovals int      `yaml:"requiredApprovals" json:"requiredApprovals"`
	Labels            []string `yaml:"labels,omitempty" json:"labels,omitempty"`
	Reviewers         []string `yaml:"reviewers,omitempty" json:"reviewers,omitempty"`
	BranchPrefix      string   `yaml:"branchPrefix,omitempty" json:"branchPrefix,omitempty"`
	TitleTemplate     string   `yaml:"titleTemplate,omitempty" json:"titleTemplate,omitempty"`
	BodyTemplate      string   `yaml:"bodyTemplate,omitempty" json:"bodyTemplate,omitempty"`
}

// GitOpsSyncConfig defines synchronization settings
type GitOpsSyncConfig struct {
	Interval      string             `yaml:"interval" json:"interval"` // e.g., "5m", "1h"
	Prune         bool               `yaml:"prune" json:"prune"`
	SelfHeal      bool               `yaml:"selfHeal" json:"selfHeal"`
	DryRun        bool               `yaml:"dryRun" json:"dryRun"`
	Timeout       string             `yaml:"timeout,omitempty" json:"timeout,omitempty"`
	RetryPolicy   *GitOpsRetryPolicy `yaml:"retryPolicy,omitempty" json:"retryPolicy,omitempty"`
	IgnoreDiffs   []GitOpsIgnoreDiff `yaml:"ignoreDiffs,omitempty" json:"ignoreDiffs,omitempty"`
}

// GitOpsRetryPolicy defines retry behavior for sync operations
type GitOpsRetryPolicy struct {
	Limit       int    `yaml:"limit" json:"limit"`
	BackoffBase string `yaml:"backoffBase" json:"backoffBase"`
	BackoffMax  string `yaml:"backoffMax" json:"backoffMax"`
}

// GitOpsIgnoreDiff defines fields to ignore during diff comparison
type GitOpsIgnoreDiff struct {
	Group     string   `yaml:"group,omitempty" json:"group,omitempty"`
	Kind      string   `yaml:"kind,omitempty" json:"kind,omitempty"`
	Name      string   `yaml:"name,omitempty" json:"name,omitempty"`
	JSONPaths []string `yaml:"jsonPaths,omitempty" json:"jsonPaths,omitempty"`
}

// GitOpsNotifications defines notification settings
type GitOpsNotifications struct {
	Slack     *GitOpsSlackNotification     `yaml:"slack,omitempty" json:"slack,omitempty"`
	Webhook   *GitOpsWebhookNotification   `yaml:"webhook,omitempty" json:"webhook,omitempty"`
	Email     *GitOpsEmailNotification     `yaml:"email,omitempty" json:"email,omitempty"`
	PagerDuty *GitOpsPagerDutyNotification `yaml:"pagerduty,omitempty" json:"pagerduty,omitempty"`
	Events    []string                     `yaml:"events,omitempty" json:"events,omitempty"` // sync, pr_created, pr_merged, error
}

// GitOpsSlackNotification defines Slack notification settings
type GitOpsSlackNotification struct {
	Channel    string `yaml:"channel" json:"channel"`
	WebhookURL string `yaml:"webhookUrl,omitempty" json:"webhookUrl,omitempty"`
	SecretRef  string `yaml:"secretRef,omitempty" json:"secretRef,omitempty"`
}

// GitOpsWebhookNotification defines generic webhook notification settings
type GitOpsWebhookNotification struct {
	URL       string            `yaml:"url" json:"url"`
	Headers   map[string]string `yaml:"headers,omitempty" json:"headers,omitempty"`
	SecretRef string            `yaml:"secretRef,omitempty" json:"secretRef,omitempty"`
}

// GitOpsEmailNotification defines email notification settings
type GitOpsEmailNotification struct {
	Recipients []string `yaml:"recipients" json:"recipients"`
	SMTPConfig string   `yaml:"smtpConfig,omitempty" json:"smtpConfig,omitempty"`
}

// GitOpsPagerDutyNotification defines PagerDuty notification settings
type GitOpsPagerDutyNotification struct {
	ServiceKey string `yaml:"serviceKey,omitempty" json:"serviceKey,omitempty"`
	SecretRef  string `yaml:"secretRef,omitempty" json:"secretRef,omitempty"`
	Severity   string `yaml:"severity,omitempty" json:"severity,omitempty"`
}

// GitOpsEnvironment defines per-environment GitOps settings
type GitOpsEnvironment struct {
	Name            string                 `yaml:"name" json:"name"`
	Path            string                 `yaml:"path,omitempty" json:"path,omitempty"`
	Branch          string                 `yaml:"branch,omitempty" json:"branch,omitempty"`
	AutoSync        bool                   `yaml:"autoSync" json:"autoSync"`
	RequireApproval bool                   `yaml:"requireApproval" json:"requireApproval"`
	Promotions      *GitOpsPromotionConfig `yaml:"promotions,omitempty" json:"promotions,omitempty"`
}

// GitOpsPromotionConfig defines environment promotion settings for GitOps
type GitOpsPromotionConfig struct {
	SourceEnv       string   `yaml:"sourceEnv" json:"sourceEnv"`
	AutoPromote     bool     `yaml:"autoPromote" json:"autoPromote"`
	RequireApproval bool     `yaml:"requireApproval" json:"requireApproval"`
	Approvers       []string `yaml:"approvers,omitempty" json:"approvers,omitempty"`
}

// GitOpsStatusV2 represents the current state of a GitOps configuration
type GitOpsStatusV2 struct {
	Phase           GitOpsPhase         `yaml:"phase" json:"phase"`
	LastSyncTime    *time.Time          `yaml:"lastSyncTime,omitempty" json:"lastSyncTime,omitempty"`
	LastSyncCommit  string              `yaml:"lastSyncCommit,omitempty" json:"lastSyncCommit,omitempty"`
	SyncStatus      GitOpsSyncStatus    `yaml:"syncStatus" json:"syncStatus"`
	HealthStatus    GitOpsHealthStatus  `yaml:"healthStatus" json:"healthStatus"`
	Conditions      []GitOpsCondition   `yaml:"conditions,omitempty" json:"conditions,omitempty"`
	ObservedGen     int64               `yaml:"observedGeneration,omitempty" json:"observedGeneration,omitempty"`
}

// GitOpsPhase represents the phase of a GitOps configuration
type GitOpsPhase string

const (
	GitOpsPhaseUnknown     GitOpsPhase = "Unknown"
	GitOpsPhasePending     GitOpsPhase = "Pending"
	GitOpsPhaseRunning     GitOpsPhase = "Running"
	GitOpsPhaseSynced      GitOpsPhase = "Synced"
	GitOpsPhaseOutOfSync   GitOpsPhase = "OutOfSync"
	GitOpsPhaseFailed      GitOpsPhase = "Failed"
	GitOpsPhaseSuspended   GitOpsPhase = "Suspended"
)

// GitOpsSyncStatus represents the synchronization status
type GitOpsSyncStatus struct {
	Status   string `yaml:"status" json:"status"` // Synced, OutOfSync, Unknown
	Revision string `yaml:"revision,omitempty" json:"revision,omitempty"`
	Message  string `yaml:"message,omitempty" json:"message,omitempty"`
}

// GitOpsHealthStatus represents the health of synced resources
type GitOpsHealthStatus struct {
	Status  string `yaml:"status" json:"status"` // Healthy, Degraded, Progressing, Unknown
	Message string `yaml:"message,omitempty" json:"message,omitempty"`
}

// GitOpsCondition represents a condition for a GitOps configuration
type GitOpsCondition struct {
	Type               string    `yaml:"type" json:"type"`
	Status             string    `yaml:"status" json:"status"`
	LastTransitionTime time.Time `yaml:"lastTransitionTime" json:"lastTransitionTime"`
	Reason             string    `yaml:"reason,omitempty" json:"reason,omitempty"`
	Message            string    `yaml:"message,omitempty" json:"message,omitempty"`
}

// GitOpsEvent represents an event in the GitOps workflow
type GitOpsEvent struct {
	Type      string                 `json:"type"` // sync, pr_created, pr_merged, error, promotion
	Timestamp time.Time              `json:"timestamp"`
	ConfigRef string                 `json:"configRef"`
	Message   string                 `json:"message"`
	Metadata  map[string]interface{} `json:"metadata,omitempty"`
}

// PullRequestState represents the state of a GitOps-created PR
type PullRequestState struct {
	Number    int        `json:"number"`
	Title     string     `json:"title"`
	URL       string     `json:"url"`
	State     string     `json:"state"` // open, closed, merged
	Branch    string     `json:"branch"`
	CreatedAt time.Time  `json:"createdAt"`
	UpdatedAt time.Time  `json:"updatedAt"`
	MergedAt  *time.Time `json:"mergedAt,omitempty"`
	Approvals int        `json:"approvals"`
	Reviewers []string   `json:"reviewers,omitempty"`
}

// GitOpsChange represents a change to be applied via GitOps
type GitOpsChange struct {
	ID          string                 `json:"id"`
	Type        string                 `json:"type"` // create, update, delete
	Resource    string                 `json:"resource"`
	Path        string                 `json:"path"`
	Before      map[string]interface{} `json:"before,omitempty"`
	After       map[string]interface{} `json:"after,omitempty"`
	Diff        string                 `json:"diff,omitempty"`
	Environment string                 `json:"environment"`
	CreatedAt   time.Time              `json:"createdAt"`
}

// GitOpsSyncResult represents the result of a sync operation
type GitOpsSyncResult struct {
	Success      bool           `json:"success"`
	Revision     string         `json:"revision"`
	Message      string         `json:"message"`
	Resources    []SyncedResource `json:"resources,omitempty"`
	Errors       []string       `json:"errors,omitempty"`
	Duration     time.Duration  `json:"duration"`
	CompletedAt  time.Time      `json:"completedAt"`
}

// SyncedResource represents a resource that was synced
type SyncedResource struct {
	Name      string `json:"name"`
	Kind      string `json:"kind"`
	Namespace string `json:"namespace,omitempty"`
	Status    string `json:"status"` // Synced, SyncFailed, Pruned, OutOfSync
	Message   string `json:"message,omitempty"`
}
