package preview

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/platformfoundry/pf-ce/internal/gitops"
	"github.com/platformfoundry/pf-ce/internal/notifications"
)

// PREventAction represents the action that triggered the PR event
type PREventAction string

const (
	PRActionOpened      PREventAction = "opened"
	PRActionClosed      PREventAction = "closed"
	PRActionReopened    PREventAction = "reopened"
	PRActionSynchronize PREventAction = "synchronize"
	PRActionMerged      PREventAction = "merged"
	PRActionLabeled     PREventAction = "labeled"
)

// PREvent represents a pull request event from a git provider
type PREvent struct {
	Action       PREventAction `json:"action"`
	Number       int           `json:"number"`
	Repository   string        `json:"repository"`
	SourceBranch string        `json:"source_branch"`
	TargetBranch string        `json:"target_branch"`
	Title        string        `json:"title"`
	Author       string        `json:"author"`
	Labels       []string      `json:"labels,omitempty"`
	IsMerged     bool          `json:"is_merged,omitempty"`
}

// WebhookConfig contains configuration for the webhook handler
type WebhookConfig struct {
	Secret             string
	DefaultTTL         time.Duration
	BaseEnvironmentMap map[string]string // target branch -> base environment
	EnabledLabels      []string          // labels that trigger preview creation
	DisabledLabels     []string          // labels that skip preview creation
	AutoDeleteOnMerge  bool
	AutoDeleteOnClose  bool
	CommentOnPR        bool
}

// WebhookHandler handles incoming webhooks from git providers
type WebhookHandler struct {
	manager     *Manager
	notifier    *notifications.Manager
	gitProvider gitops.PRClient
	config      WebhookConfig
}

// PRClient interface for git provider operations
type PRClient interface {
	CreateComment(ctx context.Context, repo string, number int, body string) error
}

// NewWebhookHandler creates a new webhook handler
func NewWebhookHandler(manager *Manager, notifier *notifications.Manager, gitProvider gitops.PRClient, config WebhookConfig) *WebhookHandler {
	if config.DefaultTTL == 0 {
		config.DefaultTTL = 72 * time.Hour
	}
	if config.BaseEnvironmentMap == nil {
		config.BaseEnvironmentMap = map[string]string{
			"main":    "production",
			"master":  "production",
			"develop": "staging",
		}
	}

	return &WebhookHandler{
		manager:     manager,
		notifier:    notifier,
		gitProvider: gitProvider,
		config:      config,
	}
}

// HandleWebhook handles incoming webhook requests
func (h *WebhookHandler) HandleWebhook(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Read body
	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Failed to read body", http.StatusBadRequest)
		return
	}

	// Verify signature if secret is configured
	if h.config.Secret != "" {
		signature := r.Header.Get("X-Hub-Signature-256")
		if signature == "" {
			signature = r.Header.Get("X-Gitlab-Token")
		}
		if !h.verifySignature(body, signature) {
			http.Error(w, "Invalid signature", http.StatusUnauthorized)
			return
		}
	}

	// Determine provider and parse event
	event, err := h.parseEvent(r, body)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to parse event: %v", err), http.StatusBadRequest)
		return
	}

	// Handle the event
	ctx := r.Context()
	if err := h.HandlePREvent(ctx, event); err != nil {
		http.Error(w, fmt.Sprintf("Failed to handle event: %v", err), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write([]byte("OK"))
}

// HandlePREvent processes a pull request event
func (h *WebhookHandler) HandlePREvent(ctx context.Context, event *PREvent) error {
	switch event.Action {
	case PRActionOpened, PRActionReopened:
		return h.handlePROpened(ctx, event)

	case PRActionSynchronize:
		return h.handlePRSynchronize(ctx, event)

	case PRActionClosed:
		if event.IsMerged {
			return h.handlePRMerged(ctx, event)
		}
		return h.handlePRClosed(ctx, event)

	case PRActionLabeled:
		return h.handlePRLabeled(ctx, event)
	}

	return nil
}

// handlePROpened creates a preview environment for a new PR
func (h *WebhookHandler) handlePROpened(ctx context.Context, event *PREvent) error {
	// Check if preview should be skipped
	if h.shouldSkipPreview(event) {
		return nil
	}

	// Determine base environment
	baseEnv := h.determineBaseEnvironment(event)

	// Create preview
	preview, err := h.manager.Create(ctx, CreatePreviewOpts{
		Repository:      event.Repository,
		PullRequest:     event.Number,
		SourceBranch:    event.SourceBranch,
		BaseEnvironment: baseEnv,
		TTL:             h.config.DefaultTTL,
		Labels: map[string]string{
			"pr":     fmt.Sprintf("%d", event.Number),
			"branch": event.SourceBranch,
			"author": event.Author,
		},
	})
	if err != nil {
		return fmt.Errorf("failed to create preview: %w", err)
	}

	// Comment on PR
	if h.config.CommentOnPR && h.gitProvider != nil {
		comment := h.formatPreviewComment(preview, "created")
		h.createPRComment(ctx, event, comment)
	}

	// Send notification
	h.sendNotification(ctx, event, preview, "created")

	return nil
}

// handlePRSynchronize updates a preview environment when PR is updated
func (h *WebhookHandler) handlePRSynchronize(ctx context.Context, event *PREvent) error {
	preview, err := h.manager.GetByPullRequest(ctx, event.Repository, event.Number)
	if err != nil {
		// Preview doesn't exist, create one if needed
		if !h.shouldSkipPreview(event) {
			return h.handlePROpened(ctx, event)
		}
		return nil
	}

	// Refresh the preview
	if err := h.manager.Refresh(ctx, preview.ID); err != nil {
		return fmt.Errorf("failed to refresh preview: %w", err)
	}

	// Comment on PR
	if h.config.CommentOnPR && h.gitProvider != nil {
		comment := h.formatPreviewComment(preview, "updated")
		h.createPRComment(ctx, event, comment)
	}

	return nil
}

// handlePRClosed deletes preview environment when PR is closed
func (h *WebhookHandler) handlePRClosed(ctx context.Context, event *PREvent) error {
	if !h.config.AutoDeleteOnClose {
		return nil
	}

	preview, err := h.manager.GetByPullRequest(ctx, event.Repository, event.Number)
	if err != nil {
		return nil // Preview doesn't exist, nothing to do
	}

	if err := h.manager.Delete(ctx, preview.ID); err != nil {
		return fmt.Errorf("failed to delete preview: %w", err)
	}

	// Comment on PR
	if h.config.CommentOnPR && h.gitProvider != nil {
		comment := h.formatPreviewComment(preview, "deleted")
		h.createPRComment(ctx, event, comment)
	}

	// Send notification
	h.sendNotification(ctx, event, preview, "deleted")

	return nil
}

// handlePRMerged deletes preview environment when PR is merged
func (h *WebhookHandler) handlePRMerged(ctx context.Context, event *PREvent) error {
	if !h.config.AutoDeleteOnMerge {
		return nil
	}

	return h.handlePRClosed(ctx, event)
}

// handlePRLabeled handles label changes that might enable/disable preview
func (h *WebhookHandler) handlePRLabeled(ctx context.Context, event *PREvent) error {
	// Check if any enabled label was added
	for _, label := range event.Labels {
		for _, enabled := range h.config.EnabledLabels {
			if label == enabled {
				// Create preview if it doesn't exist
				_, err := h.manager.GetByPullRequest(ctx, event.Repository, event.Number)
				if err != nil {
					return h.handlePROpened(ctx, event)
				}
			}
		}
	}

	return nil
}

// shouldSkipPreview determines if preview creation should be skipped
func (h *WebhookHandler) shouldSkipPreview(event *PREvent) bool {
	// Skip if any disabled label is present
	for _, label := range event.Labels {
		for _, disabled := range h.config.DisabledLabels {
			if label == disabled {
				return true
			}
		}
	}

	// If enabled labels are configured, require at least one
	if len(h.config.EnabledLabels) > 0 {
		hasEnabledLabel := false
		for _, label := range event.Labels {
			for _, enabled := range h.config.EnabledLabels {
				if label == enabled {
					hasEnabledLabel = true
					break
				}
			}
		}
		if !hasEnabledLabel {
			return true
		}
	}

	return false
}

// determineBaseEnvironment determines which base environment to use
func (h *WebhookHandler) determineBaseEnvironment(event *PREvent) string {
	if baseEnv, ok := h.config.BaseEnvironmentMap[event.TargetBranch]; ok {
		return baseEnv
	}
	return "staging" // default
}

// formatPreviewComment formats a comment for the PR
func (h *WebhookHandler) formatPreviewComment(preview *PreviewEnvironment, action string) string {
	switch action {
	case "created":
		return fmt.Sprintf(
			"Preview environment is being deployed.\n\n"+
				"**URL:** %s\n"+
				"**Expires:** %s\n\n"+
				"The environment will be available shortly.",
			preview.URL,
			preview.ExpiresAt.Format(time.RFC3339),
		)
	case "updated":
		return fmt.Sprintf(
			"Preview environment is being updated.\n\n"+
				"**URL:** %s\n"+
				"**Expires:** %s",
			preview.URL,
			preview.ExpiresAt.Format(time.RFC3339),
		)
	case "deleted":
		return "Preview environment has been deleted."
	default:
		return fmt.Sprintf("Preview environment status: %s", action)
	}
}

// createPRComment creates a comment on the PR
func (h *WebhookHandler) createPRComment(ctx context.Context, event *PREvent, body string) {
	// This would require extending the gitops.PRClient interface
	// For now, log the intent
	fmt.Printf("Would comment on PR #%d: %s\n", event.Number, body)
}

// sendNotification sends a notification about the preview environment
func (h *WebhookHandler) sendNotification(ctx context.Context, event *PREvent, preview *PreviewEnvironment, action string) {
	if h.notifier == nil {
		return
	}

	eventType := notifications.EventEnvCreated
	if action == "deleted" {
		eventType = notifications.EventEnvDeleted
	}

	h.notifier.Notify(ctx, &notifications.Event{
		Type:    eventType,
		Source:  "preview-manager",
		Subject: fmt.Sprintf("Preview environment %s for PR #%d", action, event.Number),
		Time:    time.Now(),
		Data: map[string]interface{}{
			"repository":  event.Repository,
			"pullRequest": event.Number,
			"branch":      event.SourceBranch,
			"url":         preview.URL,
			"action":      action,
		},
	})
}

// verifySignature verifies the webhook signature
func (h *WebhookHandler) verifySignature(body []byte, signature string) bool {
	if signature == "" {
		return false
	}

	// Handle GitHub signature format
	if strings.HasPrefix(signature, "sha256=") {
		expected := h.computeHMAC(body)
		return hmac.Equal([]byte(signature[7:]), []byte(expected))
	}

	// Handle GitLab token format
	return signature == h.config.Secret
}

// computeHMAC computes the HMAC-SHA256 signature
func (h *WebhookHandler) computeHMAC(body []byte) string {
	mac := hmac.New(sha256.New, []byte(h.config.Secret))
	mac.Write(body)
	return hex.EncodeToString(mac.Sum(nil))
}

// parseEvent parses the webhook payload into a PREvent
func (h *WebhookHandler) parseEvent(r *http.Request, body []byte) (*PREvent, error) {
	// Detect provider from headers
	if r.Header.Get("X-GitHub-Event") != "" {
		return h.parseGitHubEvent(r, body)
	}
	if r.Header.Get("X-Gitlab-Event") != "" {
		return h.parseGitLabEvent(r, body)
	}

	// Try generic format
	var event PREvent
	if err := json.Unmarshal(body, &event); err != nil {
		return nil, fmt.Errorf("failed to parse event: %w", err)
	}

	return &event, nil
}

// parseGitHubEvent parses a GitHub webhook payload
func (h *WebhookHandler) parseGitHubEvent(r *http.Request, body []byte) (*PREvent, error) {
	eventType := r.Header.Get("X-GitHub-Event")
	if eventType != "pull_request" {
		return nil, fmt.Errorf("unsupported event type: %s", eventType)
	}

	var payload struct {
		Action      string `json:"action"`
		Number      int    `json:"number"`
		PullRequest struct {
			Merged bool   `json:"merged"`
			Title  string `json:"title"`
			Head   struct {
				Ref string `json:"ref"`
			} `json:"head"`
			Base struct {
				Ref string `json:"ref"`
			} `json:"base"`
			User struct {
				Login string `json:"login"`
			} `json:"user"`
			Labels []struct {
				Name string `json:"name"`
			} `json:"labels"`
		} `json:"pull_request"`
		Repository struct {
			FullName string `json:"full_name"`
		} `json:"repository"`
	}

	if err := json.Unmarshal(body, &payload); err != nil {
		return nil, fmt.Errorf("failed to parse GitHub payload: %w", err)
	}

	event := &PREvent{
		Action:       PREventAction(payload.Action),
		Number:       payload.Number,
		Repository:   payload.Repository.FullName,
		SourceBranch: payload.PullRequest.Head.Ref,
		TargetBranch: payload.PullRequest.Base.Ref,
		Title:        payload.PullRequest.Title,
		Author:       payload.PullRequest.User.Login,
		IsMerged:     payload.PullRequest.Merged,
	}

	for _, label := range payload.PullRequest.Labels {
		event.Labels = append(event.Labels, label.Name)
	}

	return event, nil
}

// parseGitLabEvent parses a GitLab webhook payload
func (h *WebhookHandler) parseGitLabEvent(r *http.Request, body []byte) (*PREvent, error) {
	eventType := r.Header.Get("X-Gitlab-Event")
	if eventType != "Merge Request Hook" {
		return nil, fmt.Errorf("unsupported event type: %s", eventType)
	}

	var payload struct {
		ObjectKind       string `json:"object_kind"`
		ObjectAttributes struct {
			Action       string `json:"action"`
			IID          int    `json:"iid"`
			Title        string `json:"title"`
			SourceBranch string `json:"source_branch"`
			TargetBranch string `json:"target_branch"`
			State        string `json:"state"`
		} `json:"object_attributes"`
		User struct {
			Username string `json:"username"`
		} `json:"user"`
		Project struct {
			PathWithNamespace string `json:"path_with_namespace"`
		} `json:"project"`
		Labels []struct {
			Title string `json:"title"`
		} `json:"labels"`
	}

	if err := json.Unmarshal(body, &payload); err != nil {
		return nil, fmt.Errorf("failed to parse GitLab payload: %w", err)
	}

	// Map GitLab actions to our actions
	action := PREventAction(payload.ObjectAttributes.Action)
	if action == "open" {
		action = PRActionOpened
	} else if action == "update" {
		action = PRActionSynchronize
	} else if action == "close" {
		action = PRActionClosed
	} else if action == "merge" {
		action = PRActionClosed
	}

	event := &PREvent{
		Action:       action,
		Number:       payload.ObjectAttributes.IID,
		Repository:   payload.Project.PathWithNamespace,
		SourceBranch: payload.ObjectAttributes.SourceBranch,
		TargetBranch: payload.ObjectAttributes.TargetBranch,
		Title:        payload.ObjectAttributes.Title,
		Author:       payload.User.Username,
		IsMerged:     payload.ObjectAttributes.State == "merged",
	}

	for _, label := range payload.Labels {
		event.Labels = append(event.Labels, label.Title)
	}

	return event, nil
}

// PRClient adapter for gitops.GitHubPRClient
type gitHubPRClientAdapter struct {
	client *gitops.GitHubPRClient
}

// CreateComment implements comment creation for GitHub
func (a *gitHubPRClientAdapter) CreateComment(ctx context.Context, repo string, number int, body string) error {
	// This would need to be added to the GitHubPRClient
	return nil
}
