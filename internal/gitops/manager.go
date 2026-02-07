package gitops

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/platformfoundry/pf-ce/pkg/types"
)

// Manager handles GitOps operations for the platform
type Manager struct {
	config        *types.GitOpsConfigV2
	workDir       string
	gitClient     GitClient
	prClient      PRClient
	notifier      Notifier
	syncScheduler *SyncScheduler
	mu            sync.RWMutex
	events        []types.GitOpsEvent
	maxEvents     int
}

// GitClient interface for Git operations
type GitClient interface {
	Clone(ctx context.Context, url, branch, path string) error
	Pull(ctx context.Context, path string) error
	Checkout(ctx context.Context, path, ref string) error
	GetCurrentCommit(ctx context.Context, path string) (string, error)
	GetDiff(ctx context.Context, path, fromRef, toRef string) (string, error)
	CommitAndPush(ctx context.Context, path, message string, files []string) error
	CreateBranch(ctx context.Context, path, branchName string) error
}

// PRClient interface for Pull Request operations
type PRClient interface {
	CreatePR(ctx context.Context, opts CreatePROptions) (*types.PullRequestState, error)
	GetPR(ctx context.Context, number int) (*types.PullRequestState, error)
	MergePR(ctx context.Context, number int) error
	ListOpenPRs(ctx context.Context) ([]types.PullRequestState, error)
	AddReviewers(ctx context.Context, number int, reviewers []string) error
	AddLabels(ctx context.Context, number int, labels []string) error
}

// CreatePROptions contains options for creating a pull request
type CreatePROptions struct {
	Title        string
	Body         string
	SourceBranch string
	TargetBranch string
	Labels       []string
	Reviewers    []string
}

// Notifier interface for sending notifications
type Notifier interface {
	NotifySync(ctx context.Context, result *types.GitOpsSyncResult) error
	NotifyPRCreated(ctx context.Context, pr *types.PullRequestState) error
	NotifyPRMerged(ctx context.Context, pr *types.PullRequestState) error
	NotifyError(ctx context.Context, err error, context string) error
}

// ManagerConfig contains configuration for the GitOps manager
type ManagerConfig struct {
	Config    *types.GitOpsConfigV2
	WorkDir   string
	GitClient GitClient
	PRClient  PRClient
	Notifier  Notifier
}

// NewManager creates a new GitOps manager
func NewManager(cfg ManagerConfig) (*Manager, error) {
	if cfg.Config == nil {
		return nil, fmt.Errorf("gitops config is required")
	}
	if cfg.WorkDir == "" {
		cfg.WorkDir = filepath.Join(os.TempDir(), "pf-gitops")
	}

	// Create work directory
	if err := os.MkdirAll(cfg.WorkDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create work directory: %w", err)
	}

	// Use default git client if none provided
	gitClient := cfg.GitClient
	if gitClient == nil {
		gitClient = &DefaultGitClient{}
	}

	m := &Manager{
		config:    cfg.Config,
		workDir:   cfg.WorkDir,
		gitClient: gitClient,
		prClient:  cfg.PRClient,
		notifier:  cfg.Notifier,
		events:    make([]types.GitOpsEvent, 0),
		maxEvents: 1000,
	}

	return m, nil
}

// Initialize sets up the GitOps repository
func (m *Manager) Initialize(ctx context.Context) error {
	repoPath := m.getRepoPath()

	// Check if repo already exists
	if _, err := os.Stat(filepath.Join(repoPath, ".git")); err == nil {
		// Repo exists, pull latest
		if err := m.gitClient.Pull(ctx, repoPath); err != nil {
			return fmt.Errorf("failed to pull repository: %w", err)
		}
		return nil
	}

	// Clone the repository
	err := m.gitClient.Clone(
		ctx,
		m.config.Spec.Repository.URL,
		m.config.Spec.Repository.Branch,
		repoPath,
	)
	if err != nil {
		return fmt.Errorf("failed to clone repository: %w", err)
	}

	m.recordEvent(types.GitOpsEvent{
		Type:      "initialized",
		Timestamp: time.Now(),
		ConfigRef: m.config.Metadata.Name,
		Message:   fmt.Sprintf("Initialized GitOps repository from %s", m.config.Spec.Repository.URL),
	})

	return nil
}

// Sync synchronizes the platform state with the Git repository
func (m *Manager) Sync(ctx context.Context) (*types.GitOpsSyncResult, error) {
	startTime := time.Now()
	repoPath := m.getRepoPath()

	// Pull latest changes
	if err := m.gitClient.Pull(ctx, repoPath); err != nil {
		m.recordErrorEvent(err, "sync pull")
		return &types.GitOpsSyncResult{
			Success:     false,
			Message:     fmt.Sprintf("Failed to pull: %s", err),
			CompletedAt: time.Now(),
			Duration:    time.Since(startTime),
		}, err
	}

	// Get current commit
	commit, err := m.gitClient.GetCurrentCommit(ctx, repoPath)
	if err != nil {
		m.recordErrorEvent(err, "get commit")
		return nil, err
	}

	// TODO: Parse manifests from repository
	// TODO: Compare with current state
	// TODO: Apply changes if selfHeal enabled

	result := &types.GitOpsSyncResult{
		Success:     true,
		Revision:    commit,
		Message:     "Sync completed successfully",
		Resources:   []types.SyncedResource{},
		CompletedAt: time.Now(),
		Duration:    time.Since(startTime),
	}

	m.recordEvent(types.GitOpsEvent{
		Type:      "sync",
		Timestamp: time.Now(),
		ConfigRef: m.config.Metadata.Name,
		Message:   fmt.Sprintf("Sync completed: %s", commit[:8]),
		Metadata: map[string]interface{}{
			"revision": commit,
			"duration": result.Duration.String(),
		},
	})

	// Send notification
	if m.notifier != nil {
		if err := m.notifier.NotifySync(ctx, result); err != nil {
			// Log but don't fail
			fmt.Printf("Failed to send sync notification: %v\n", err)
		}
	}

	return result, nil
}

// ProposeChange creates a PR for a platform change
func (m *Manager) ProposeChange(ctx context.Context, change types.GitOpsChange) (*types.PullRequestState, error) {
	if !m.config.Spec.PullRequest.Enabled {
		return nil, fmt.Errorf("pull request workflow is not enabled")
	}

	if m.prClient == nil {
		return nil, fmt.Errorf("no PR client configured")
	}

	repoPath := m.getRepoPath()

	// Create a new branch
	branchPrefix := m.config.Spec.PullRequest.BranchPrefix
	if branchPrefix == "" {
		branchPrefix = "pf-change"
	}
	branchName := fmt.Sprintf("%s/%s-%d", branchPrefix, change.Type, time.Now().Unix())

	if err := m.gitClient.CreateBranch(ctx, repoPath, branchName); err != nil {
		return nil, fmt.Errorf("failed to create branch: %w", err)
	}

	// Write the change to the appropriate file
	changePath := filepath.Join(repoPath, change.Path)
	if err := os.MkdirAll(filepath.Dir(changePath), 0755); err != nil {
		return nil, fmt.Errorf("failed to create directory: %w", err)
	}

	// TODO: Write the change content to the file

	// Commit and push
	commitMsg := fmt.Sprintf("[Platform Foundry] %s %s", change.Type, change.Resource)
	if err := m.gitClient.CommitAndPush(ctx, repoPath, commitMsg, []string{change.Path}); err != nil {
		return nil, fmt.Errorf("failed to commit and push: %w", err)
	}

	// Create the PR
	title := m.formatPRTitle(change)
	body := m.formatPRBody(change)

	pr, err := m.prClient.CreatePR(ctx, CreatePROptions{
		Title:        title,
		Body:         body,
		SourceBranch: branchName,
		TargetBranch: m.config.Spec.Repository.Branch,
		Labels:       m.config.Spec.PullRequest.Labels,
		Reviewers:    m.config.Spec.PullRequest.Reviewers,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create PR: %w", err)
	}

	m.recordEvent(types.GitOpsEvent{
		Type:      "pr_created",
		Timestamp: time.Now(),
		ConfigRef: m.config.Metadata.Name,
		Message:   fmt.Sprintf("PR #%d created: %s", pr.Number, pr.Title),
		Metadata: map[string]interface{}{
			"pr_number": pr.Number,
			"pr_url":    pr.URL,
			"branch":    branchName,
		},
	})

	// Send notification
	if m.notifier != nil {
		if err := m.notifier.NotifyPRCreated(ctx, pr); err != nil {
			fmt.Printf("Failed to send PR notification: %v\n", err)
		}
	}

	return pr, nil
}

// PromoteEnvironment promotes changes from one environment to another
func (m *Manager) PromoteEnvironment(ctx context.Context, sourceEnv, targetEnv string) (*types.PullRequestState, error) {
	// Find environment configs
	var sourceConfig, targetConfig *types.GitOpsEnvironment
	for i := range m.config.Spec.Environments {
		if m.config.Spec.Environments[i].Name == sourceEnv {
			sourceConfig = &m.config.Spec.Environments[i]
		}
		if m.config.Spec.Environments[i].Name == targetEnv {
			targetConfig = &m.config.Spec.Environments[i]
		}
	}

	if sourceConfig == nil {
		return nil, fmt.Errorf("source environment %s not found", sourceEnv)
	}
	if targetConfig == nil {
		return nil, fmt.Errorf("target environment %s not found", targetEnv)
	}

	// Get diff between environments
	repoPath := m.getRepoPath()
	sourcePath := filepath.Join(repoPath, sourceConfig.Path)
	targetPath := filepath.Join(repoPath, targetConfig.Path)

	diff, err := m.gitClient.GetDiff(ctx, repoPath, sourcePath, targetPath)
	if err != nil {
		return nil, fmt.Errorf("failed to get diff: %w", err)
	}

	if diff == "" {
		return nil, fmt.Errorf("no differences found between environments")
	}

	// Create promotion change
	change := types.GitOpsChange{
		ID:          fmt.Sprintf("promote-%s-%s-%d", sourceEnv, targetEnv, time.Now().Unix()),
		Type:        "promotion",
		Resource:    fmt.Sprintf("%s -> %s", sourceEnv, targetEnv),
		Path:        targetConfig.Path,
		Diff:        diff,
		Environment: targetEnv,
		CreatedAt:   time.Now(),
	}

	// Create PR for the promotion
	return m.ProposeChange(ctx, change)
}

// GetStatus returns the current GitOps status
func (m *Manager) GetStatus(ctx context.Context) (*types.GitOpsStatusV2, error) {
	repoPath := m.getRepoPath()

	commit, err := m.gitClient.GetCurrentCommit(ctx, repoPath)
	if err != nil {
		return &types.GitOpsStatusV2{
			Phase: types.GitOpsPhaseUnknown,
			SyncStatus: types.GitOpsSyncStatus{
				Status:  "Unknown",
				Message: err.Error(),
			},
		}, nil
	}

	now := time.Now()
	return &types.GitOpsStatusV2{
		Phase:          types.GitOpsPhaseSynced,
		LastSyncTime:   &now,
		LastSyncCommit: commit,
		SyncStatus: types.GitOpsSyncStatus{
			Status:   "Synced",
			Revision: commit,
		},
		HealthStatus: types.GitOpsHealthStatus{
			Status: "Healthy",
		},
	}, nil
}

// GetEvents returns recent GitOps events
func (m *Manager) GetEvents(limit int) []types.GitOpsEvent {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if limit <= 0 || limit > len(m.events) {
		limit = len(m.events)
	}

	// Return most recent events
	start := len(m.events) - limit
	if start < 0 {
		start = 0
	}

	result := make([]types.GitOpsEvent, limit)
	copy(result, m.events[start:])
	return result
}

// StartSyncScheduler starts the background sync scheduler
func (m *Manager) StartSyncScheduler(ctx context.Context) error {
	interval, err := time.ParseDuration(m.config.Spec.Sync.Interval)
	if err != nil {
		return fmt.Errorf("invalid sync interval: %w", err)
	}

	m.syncScheduler = &SyncScheduler{
		manager:  m,
		interval: interval,
		stopCh:   make(chan struct{}),
	}

	go m.syncScheduler.Run(ctx)
	return nil
}

// StopSyncScheduler stops the background sync scheduler
func (m *Manager) StopSyncScheduler() {
	if m.syncScheduler != nil {
		close(m.syncScheduler.stopCh)
	}
}

// Helper methods

func (m *Manager) getRepoPath() string {
	return filepath.Join(m.workDir, m.config.Metadata.Name)
}

func (m *Manager) recordEvent(event types.GitOpsEvent) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.events = append(m.events, event)

	// Trim old events
	if len(m.events) > m.maxEvents {
		m.events = m.events[len(m.events)-m.maxEvents:]
	}
}

func (m *Manager) recordErrorEvent(err error, errContext string) {
	m.recordEvent(types.GitOpsEvent{
		Type:      "error",
		Timestamp: time.Now(),
		ConfigRef: m.config.Metadata.Name,
		Message:   fmt.Sprintf("Error during %s: %s", errContext, err.Error()),
		Metadata: map[string]interface{}{
			"context": errContext,
			"error":   err.Error(),
		},
	})

	if m.notifier != nil {
		ctx := context.Background()
		m.notifier.NotifyError(ctx, err, errContext)
	}
}

func (m *Manager) formatPRTitle(change types.GitOpsChange) string {
	template := m.config.Spec.PullRequest.TitleTemplate
	if template == "" {
		template = "[Platform Foundry] {{.Type}} {{.Resource}}"
	}

	// Simple template replacement
	title := strings.ReplaceAll(template, "{{.Type}}", change.Type)
	title = strings.ReplaceAll(title, "{{.Resource}}", change.Resource)
	title = strings.ReplaceAll(title, "{{.Environment}}", change.Environment)

	return title
}

func (m *Manager) formatPRBody(change types.GitOpsChange) string {
	template := m.config.Spec.PullRequest.BodyTemplate
	if template == "" {
		template = `## Platform Foundry Change

**Type:** {{.Type}}
**Resource:** {{.Resource}}
**Environment:** {{.Environment}}

### Changes

` + "```diff\n{{.Diff}}\n```" + `

---
*This PR was automatically created by Platform Foundry*
`
	}

	body := strings.ReplaceAll(template, "{{.Type}}", change.Type)
	body = strings.ReplaceAll(body, "{{.Resource}}", change.Resource)
	body = strings.ReplaceAll(body, "{{.Environment}}", change.Environment)
	body = strings.ReplaceAll(body, "{{.Diff}}", change.Diff)

	return body
}

// SyncScheduler handles periodic sync operations
type SyncScheduler struct {
	manager  *Manager
	interval time.Duration
	stopCh   chan struct{}
}

// Run starts the sync scheduler loop
func (s *SyncScheduler) Run(ctx context.Context) {
	ticker := time.NewTicker(s.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-s.stopCh:
			return
		case <-ticker.C:
			if _, err := s.manager.Sync(ctx); err != nil {
				fmt.Printf("Scheduled sync failed: %v\n", err)
			}
		}
	}
}

// DefaultGitClient implements GitClient using system git
type DefaultGitClient struct{}

func (c *DefaultGitClient) Clone(ctx context.Context, url, branch, path string) error {
	args := []string{"clone", "--branch", branch, "--single-branch", url, path}
	cmd := exec.CommandContext(ctx, "git", args...)
	return cmd.Run()
}

func (c *DefaultGitClient) Pull(ctx context.Context, path string) error {
	cmd := exec.CommandContext(ctx, "git", "-C", path, "pull", "--ff-only")
	return cmd.Run()
}

func (c *DefaultGitClient) Checkout(ctx context.Context, path, ref string) error {
	cmd := exec.CommandContext(ctx, "git", "-C", path, "checkout", ref)
	return cmd.Run()
}

func (c *DefaultGitClient) GetCurrentCommit(ctx context.Context, path string) (string, error) {
	cmd := exec.CommandContext(ctx, "git", "-C", path, "rev-parse", "HEAD")
	output, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(output)), nil
}

func (c *DefaultGitClient) GetDiff(ctx context.Context, path, fromRef, toRef string) (string, error) {
	cmd := exec.CommandContext(ctx, "git", "-C", path, "diff", fromRef, toRef)
	output, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return string(output), nil
}

func (c *DefaultGitClient) CommitAndPush(ctx context.Context, path, message string, files []string) error {
	// Add files
	args := append([]string{"-C", path, "add"}, files...)
	if err := exec.CommandContext(ctx, "git", args...).Run(); err != nil {
		return err
	}

	// Commit
	if err := exec.CommandContext(ctx, "git", "-C", path, "commit", "-m", message).Run(); err != nil {
		return err
	}

	// Push
	return exec.CommandContext(ctx, "git", "-C", path, "push").Run()
}

func (c *DefaultGitClient) CreateBranch(ctx context.Context, path, branchName string) error {
	if err := exec.CommandContext(ctx, "git", "-C", path, "checkout", "-b", branchName).Run(); err != nil {
		return err
	}
	return nil
}
