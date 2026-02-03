// Package preview provides ephemeral preview environment management for PR-based workflows.
package preview

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/platformfoundry/pf-ce/internal/state"
)

// PreviewStatus represents the status of a preview environment
type PreviewStatus string

const (
	StatusPending     PreviewStatus = "pending"
	StatusProvisioning PreviewStatus = "provisioning"
	StatusReady       PreviewStatus = "ready"
	StatusFailed      PreviewStatus = "failed"
	StatusDeleting    PreviewStatus = "deleting"
	StatusDeleted     PreviewStatus = "deleted"
)

// DatabaseStrategy defines how to handle database for preview environments
type DatabaseStrategy string

const (
	DatabaseStrategyClone DatabaseStrategy = "clone"
	DatabaseStrategyFresh DatabaseStrategy = "fresh"
	DatabaseStrategySeed  DatabaseStrategy = "seed"
)

// PreviewEnvironment represents an ephemeral environment created for a PR
type PreviewEnvironment struct {
	ID               string            `json:"id"`
	Name             string            `json:"name"`
	SourceRepo       string            `json:"sourceRepo"`
	SourceBranch     string            `json:"sourceBranch"`
	PullRequest      int               `json:"pullRequest"`
	BaseEnvironment  string            `json:"baseEnvironment"`
	TTL              time.Duration     `json:"ttl"`
	URL              string            `json:"url"`
	IngressIP        string            `json:"ingressIP,omitempty"`
	Status           PreviewStatus     `json:"status"`
	StatusMessage    string            `json:"statusMessage,omitempty"`
	CreatedAt        time.Time         `json:"createdAt"`
	UpdatedAt        time.Time         `json:"updatedAt"`
	ExpiresAt        time.Time         `json:"expiresAt"`
	Resources        []Resource        `json:"resources,omitempty"`
	DatabaseStrategy DatabaseStrategy  `json:"databaseStrategy,omitempty"`
	SeedDataPath     string            `json:"seedDataPath,omitempty"`
	Labels           map[string]string `json:"labels,omitempty"`
	Metadata         map[string]string `json:"metadata,omitempty"`
}

// Resource represents a deployed resource in a preview environment
type Resource struct {
	Name      string                 `json:"name"`
	Type      string                 `json:"type"`
	Namespace string                 `json:"namespace"`
	Status    string                 `json:"status"`
	Spec      map[string]interface{} `json:"spec,omitempty"`
}

// CreatePreviewOpts contains options for creating a preview environment
type CreatePreviewOpts struct {
	Repository       string
	PullRequest      int
	SourceBranch     string
	BaseEnvironment  string
	TTL              time.Duration
	DatabaseStrategy DatabaseStrategy
	SeedDataPath     string
	Labels           map[string]string
}

// DNSProvider defines the interface for DNS operations
type DNSProvider interface {
	CreateRecord(ctx context.Context, hostname, target string) error
	DeleteRecord(ctx context.Context, hostname string) error
}

// Orchestrator defines the interface for resource orchestration
type Orchestrator interface {
	Apply(ctx context.Context, resources []Resource) error
	Delete(ctx context.Context, resources []Resource) error
	GetStatus(ctx context.Context, namespace string) (string, error)
}

// ManagerConfig contains configuration for the preview manager
type ManagerConfig struct {
	DefaultTTL       time.Duration
	MaxTTL           time.Duration
	URLPattern       string // e.g., "pr-{{.PR}}.preview.example.com"
	CleanupInterval  time.Duration
	MaxConcurrent    int
}

// Manager manages preview environments
type Manager struct {
	config        ManagerConfig
	stateBackend  state.Backend
	dnsProvider   DNSProvider
	orchestrator  Orchestrator
	cleanupWorker *CleanupWorker
	mu            sync.RWMutex
}

// NewManager creates a new preview environment manager
func NewManager(cfg ManagerConfig, backend state.Backend, dns DNSProvider, orch Orchestrator) *Manager {
	if cfg.DefaultTTL == 0 {
		cfg.DefaultTTL = 72 * time.Hour
	}
	if cfg.MaxTTL == 0 {
		cfg.MaxTTL = 168 * time.Hour // 7 days
	}
	if cfg.URLPattern == "" {
		cfg.URLPattern = "pr-{{.PR}}-{{.Branch}}.preview.local"
	}
	if cfg.CleanupInterval == 0 {
		cfg.CleanupInterval = 5 * time.Minute
	}
	if cfg.MaxConcurrent == 0 {
		cfg.MaxConcurrent = 10
	}

	m := &Manager{
		config:       cfg,
		stateBackend: backend,
		dnsProvider:  dns,
		orchestrator: orch,
	}

	m.cleanupWorker = NewCleanupWorker(m, cfg.CleanupInterval)

	return m
}

// Create creates a new preview environment
func (m *Manager) Create(ctx context.Context, opts CreatePreviewOpts) (*PreviewEnvironment, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Validate options
	if opts.Repository == "" {
		return nil, fmt.Errorf("repository is required")
	}
	if opts.PullRequest <= 0 {
		return nil, fmt.Errorf("pull request number is required")
	}
	if opts.BaseEnvironment == "" {
		return nil, fmt.Errorf("base environment is required")
	}

	// Set defaults
	if opts.TTL == 0 {
		opts.TTL = m.config.DefaultTTL
	}
	if opts.TTL > m.config.MaxTTL {
		opts.TTL = m.config.MaxTTL
	}
	if opts.DatabaseStrategy == "" {
		opts.DatabaseStrategy = DatabaseStrategyFresh
	}

	// Check if preview already exists for this PR
	existing, _ := m.GetByPullRequest(ctx, opts.Repository, opts.PullRequest)
	if existing != nil && existing.Status != StatusDeleted {
		return nil, fmt.Errorf("preview environment already exists for PR #%d", opts.PullRequest)
	}

	// Get base environment configuration
	baseEnv, err := m.getBaseEnvironment(ctx, opts.BaseEnvironment)
	if err != nil {
		return nil, fmt.Errorf("base environment not found: %w", err)
	}

	// Generate preview environment name
	previewName := m.generatePreviewName(opts)

	// Create preview environment
	now := time.Now()
	preview := &PreviewEnvironment{
		ID:               uuid.New().String(),
		Name:             previewName,
		SourceRepo:       opts.Repository,
		SourceBranch:     opts.SourceBranch,
		PullRequest:      opts.PullRequest,
		BaseEnvironment:  opts.BaseEnvironment,
		TTL:              opts.TTL,
		Status:           StatusPending,
		CreatedAt:        now,
		UpdatedAt:        now,
		ExpiresAt:        now.Add(opts.TTL),
		DatabaseStrategy: opts.DatabaseStrategy,
		SeedDataPath:     opts.SeedDataPath,
		Labels:           opts.Labels,
		Metadata:         make(map[string]string),
	}

	// Store initial state
	if err := m.savePreview(ctx, preview); err != nil {
		return nil, fmt.Errorf("failed to save preview state: %w", err)
	}

	// Deploy preview in background
	go m.deployPreview(context.Background(), preview, baseEnv)

	return preview, nil
}

// deployPreview deploys the preview environment resources
func (m *Manager) deployPreview(ctx context.Context, preview *PreviewEnvironment, baseEnv *state.Resource) {
	// Update status to provisioning
	preview.Status = StatusProvisioning
	preview.UpdatedAt = time.Now()
	_ = m.savePreview(ctx, preview)

	// Clone resources from base environment
	resources, err := m.cloneResources(ctx, preview, baseEnv)
	if err != nil {
		preview.Status = StatusFailed
		preview.StatusMessage = fmt.Sprintf("failed to clone resources: %v", err)
		preview.UpdatedAt = time.Now()
		_ = m.savePreview(ctx, preview)
		return
	}

	preview.Resources = resources

	// Apply resources via orchestrator
	if m.orchestrator != nil {
		if err := m.orchestrator.Apply(ctx, resources); err != nil {
			preview.Status = StatusFailed
			preview.StatusMessage = fmt.Sprintf("failed to apply resources: %v", err)
			preview.UpdatedAt = time.Now()
			_ = m.savePreview(ctx, preview)
			return
		}
	}

	// Configure DNS
	preview.URL = m.generatePreviewURL(preview)
	if m.dnsProvider != nil && preview.IngressIP != "" {
		if err := m.dnsProvider.CreateRecord(ctx, preview.URL, preview.IngressIP); err != nil {
			preview.Status = StatusFailed
			preview.StatusMessage = fmt.Sprintf("failed to create DNS record: %v", err)
			preview.UpdatedAt = time.Now()
			_ = m.savePreview(ctx, preview)
			return
		}
	}

	// Update status to ready
	preview.Status = StatusReady
	preview.StatusMessage = ""
	preview.UpdatedAt = time.Now()

	// Schedule cleanup
	m.cleanupWorker.Schedule(preview.ID, preview.ExpiresAt)

	_ = m.savePreview(ctx, preview)
}

// cloneResources clones resources from base environment for preview
func (m *Manager) cloneResources(ctx context.Context, preview *PreviewEnvironment, baseEnv *state.Resource) ([]Resource, error) {
	var resources []Resource

	// Get resources from base environment spec
	if baseEnv.Spec == nil {
		return resources, nil
	}

	baseResources, ok := baseEnv.Spec["resources"].([]interface{})
	if !ok {
		return resources, nil
	}

	for _, r := range baseResources {
		resMap, ok := r.(map[string]interface{})
		if !ok {
			continue
		}

		res := Resource{
			Name:      fmt.Sprintf("%s-%s", preview.Name, resMap["name"]),
			Type:      getString(resMap, "type"),
			Namespace: preview.Name,
			Status:    "pending",
			Spec:      make(map[string]interface{}),
		}

		// Copy spec
		if spec, ok := resMap["spec"].(map[string]interface{}); ok {
			for k, v := range spec {
				res.Spec[k] = v
			}
		}

		// Handle database strategy
		if res.Type == "database" {
			switch preview.DatabaseStrategy {
			case DatabaseStrategyClone:
				if snapshot := getString(baseEnv.Spec, "databaseSnapshot"); snapshot != "" {
					res.Spec["sourceSnapshot"] = snapshot
				}
			case DatabaseStrategySeed:
				if preview.SeedDataPath != "" {
					res.Spec["seedData"] = preview.SeedDataPath
				}
			case DatabaseStrategyFresh:
				// Use empty database
			}
		}

		resources = append(resources, res)
	}

	return resources, nil
}

// Get retrieves a preview environment by ID
func (m *Manager) Get(ctx context.Context, id string) (*PreviewEnvironment, error) {
	resource, err := m.stateBackend.Get("preview:" + id)
	if err != nil {
		return nil, err
	}

	return resourceToPreview(resource)
}

// GetByPullRequest retrieves a preview environment by repository and PR number
func (m *Manager) GetByPullRequest(ctx context.Context, repo string, pr int) (*PreviewEnvironment, error) {
	previews, err := m.List(ctx)
	if err != nil {
		return nil, err
	}

	for _, p := range previews {
		if p.SourceRepo == repo && p.PullRequest == pr {
			return p, nil
		}
	}

	return nil, fmt.Errorf("preview environment not found for PR #%d", pr)
}

// List returns all preview environments
func (m *Manager) List(ctx context.Context) ([]*PreviewEnvironment, error) {
	resources, err := m.stateBackend.List()
	if err != nil {
		return nil, err
	}

	var previews []*PreviewEnvironment
	for _, r := range resources {
		if r.Kind == "PreviewEnvironment" {
			p, err := resourceToPreview(r)
			if err == nil && p.Status != StatusDeleted {
				previews = append(previews, p)
			}
		}
	}

	return previews, nil
}

// Delete removes a preview environment
func (m *Manager) Delete(ctx context.Context, id string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	preview, err := m.Get(ctx, id)
	if err != nil {
		return err
	}

	// Update status
	preview.Status = StatusDeleting
	preview.UpdatedAt = time.Now()
	if err := m.savePreview(ctx, preview); err != nil {
		return err
	}

	// Delete resources
	if m.orchestrator != nil && len(preview.Resources) > 0 {
		if err := m.orchestrator.Delete(ctx, preview.Resources); err != nil {
			return fmt.Errorf("failed to delete resources: %w", err)
		}
	}

	// Delete DNS record
	if m.dnsProvider != nil && preview.URL != "" {
		_ = m.dnsProvider.DeleteRecord(ctx, preview.URL)
	}

	// Update status to deleted
	preview.Status = StatusDeleted
	preview.UpdatedAt = time.Now()

	return m.savePreview(ctx, preview)
}

// DeleteByPullRequest removes a preview environment by repository and PR number
func (m *Manager) DeleteByPullRequest(ctx context.Context, repo string, pr int) error {
	preview, err := m.GetByPullRequest(ctx, repo, pr)
	if err != nil {
		return err
	}

	return m.Delete(ctx, preview.ID)
}

// Extend extends the TTL of a preview environment
func (m *Manager) Extend(ctx context.Context, id string, additionalTTL time.Duration) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	preview, err := m.Get(ctx, id)
	if err != nil {
		return err
	}

	if preview.Status != StatusReady {
		return fmt.Errorf("can only extend ready preview environments")
	}

	newExpiry := preview.ExpiresAt.Add(additionalTTL)
	maxExpiry := preview.CreatedAt.Add(m.config.MaxTTL)

	if newExpiry.After(maxExpiry) {
		newExpiry = maxExpiry
	}

	preview.ExpiresAt = newExpiry
	preview.UpdatedAt = time.Now()

	// Reschedule cleanup
	m.cleanupWorker.Schedule(preview.ID, preview.ExpiresAt)

	return m.savePreview(ctx, preview)
}

// Refresh rebuilds a preview environment with latest changes
func (m *Manager) Refresh(ctx context.Context, id string) error {
	preview, err := m.Get(ctx, id)
	if err != nil {
		return err
	}

	if preview.Status != StatusReady && preview.Status != StatusFailed {
		return fmt.Errorf("can only refresh ready or failed preview environments")
	}

	baseEnv, err := m.getBaseEnvironment(ctx, preview.BaseEnvironment)
	if err != nil {
		return err
	}

	// Redeploy
	go m.deployPreview(context.Background(), preview, baseEnv)

	return nil
}

// StartCleanupWorker starts the background cleanup worker
func (m *Manager) StartCleanupWorker(ctx context.Context) {
	m.cleanupWorker.Start(ctx)
}

// StopCleanupWorker stops the background cleanup worker
func (m *Manager) StopCleanupWorker() {
	m.cleanupWorker.Stop()
}

// savePreview saves a preview environment to state backend
func (m *Manager) savePreview(ctx context.Context, preview *PreviewEnvironment) error {
	resource := &state.Resource{
		Name:       "preview:" + preview.ID,
		Kind:       "PreviewEnvironment",
		APIVersion: "platformfoundry.io/v1",
		Spec: map[string]interface{}{
			"id":               preview.ID,
			"name":             preview.Name,
			"sourceRepo":       preview.SourceRepo,
			"sourceBranch":     preview.SourceBranch,
			"pullRequest":      preview.PullRequest,
			"baseEnvironment":  preview.BaseEnvironment,
			"ttl":              preview.TTL.String(),
			"url":              preview.URL,
			"ingressIP":        preview.IngressIP,
			"status":           string(preview.Status),
			"statusMessage":    preview.StatusMessage,
			"expiresAt":        preview.ExpiresAt.Format(time.RFC3339),
			"databaseStrategy": string(preview.DatabaseStrategy),
			"seedDataPath":     preview.SeedDataPath,
			"labels":           preview.Labels,
			"metadata":         preview.Metadata,
			"resources":        preview.Resources,
		},
		CreatedAt: preview.CreatedAt,
		UpdatedAt: preview.UpdatedAt,
	}

	return m.stateBackend.Save(resource)
}

// getBaseEnvironment retrieves the base environment configuration
func (m *Manager) getBaseEnvironment(ctx context.Context, name string) (*state.Resource, error) {
	return m.stateBackend.Get("environment:" + name)
}

// generatePreviewName generates a name for the preview environment
func (m *Manager) generatePreviewName(opts CreatePreviewOpts) string {
	branch := opts.SourceBranch
	if len(branch) > 21 {
		branch = branch[:21]
	}
	// Sanitize branch name for use in resource names
	return fmt.Sprintf("pr-%d-%s", opts.PullRequest, sanitizeName(branch))
}

// generatePreviewURL generates the URL for the preview environment
func (m *Manager) generatePreviewURL(preview *PreviewEnvironment) string {
	return fmt.Sprintf("pr-%d.preview.local", preview.PullRequest)
}

// Helper functions

func resourceToPreview(r *state.Resource) (*PreviewEnvironment, error) {
	if r == nil || r.Spec == nil {
		return nil, fmt.Errorf("invalid resource")
	}

	ttl, _ := time.ParseDuration(getString(r.Spec, "ttl"))
	expiresAt, _ := time.Parse(time.RFC3339, getString(r.Spec, "expiresAt"))

	preview := &PreviewEnvironment{
		ID:               getString(r.Spec, "id"),
		Name:             getString(r.Spec, "name"),
		SourceRepo:       getString(r.Spec, "sourceRepo"),
		SourceBranch:     getString(r.Spec, "sourceBranch"),
		PullRequest:      getInt(r.Spec, "pullRequest"),
		BaseEnvironment:  getString(r.Spec, "baseEnvironment"),
		TTL:              ttl,
		URL:              getString(r.Spec, "url"),
		IngressIP:        getString(r.Spec, "ingressIP"),
		Status:           PreviewStatus(getString(r.Spec, "status")),
		StatusMessage:    getString(r.Spec, "statusMessage"),
		CreatedAt:        r.CreatedAt,
		UpdatedAt:        r.UpdatedAt,
		ExpiresAt:        expiresAt,
		DatabaseStrategy: DatabaseStrategy(getString(r.Spec, "databaseStrategy")),
		SeedDataPath:     getString(r.Spec, "seedDataPath"),
	}

	if labels, ok := r.Spec["labels"].(map[string]string); ok {
		preview.Labels = labels
	}
	if metadata, ok := r.Spec["metadata"].(map[string]string); ok {
		preview.Metadata = metadata
	}

	return preview, nil
}

func getString(m map[string]interface{}, key string) string {
	if v, ok := m[key].(string); ok {
		return v
	}
	return ""
}

func getInt(m map[string]interface{}, key string) int {
	switch v := m[key].(type) {
	case int:
		return v
	case float64:
		return int(v)
	}
	return 0
}

func sanitizeName(name string) string {
	result := make([]byte, 0, len(name))
	for i := 0; i < len(name); i++ {
		c := name[i]
		if (c >= 'a' && c <= 'z') || (c >= '0' && c <= '9') || c == '-' {
			result = append(result, c)
		} else if c >= 'A' && c <= 'Z' {
			result = append(result, c+32) // toLowerCase
		} else if c == '_' || c == '/' {
			result = append(result, '-')
		}
	}
	return string(result)
}
