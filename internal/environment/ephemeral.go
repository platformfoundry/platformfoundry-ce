// Package environment provides environment management including ephemeral environments
package environment

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
)

// EphemeralEnvironmentStatus represents the status of an ephemeral environment
type EphemeralEnvironmentStatus string

const (
	EphemeralStatusPending     EphemeralEnvironmentStatus = "pending"
	EphemeralStatusProvisioning EphemeralEnvironmentStatus = "provisioning"
	EphemeralStatusReady       EphemeralEnvironmentStatus = "ready"
	EphemeralStatusFailed      EphemeralEnvironmentStatus = "failed"
	EphemeralStatusExpired     EphemeralEnvironmentStatus = "expired"
	EphemeralStatusDeleting    EphemeralEnvironmentStatus = "deleting"
	EphemeralStatusDeleted     EphemeralEnvironmentStatus = "deleted"
)

// EphemeralEnvironment represents a temporary environment
type EphemeralEnvironment struct {
	ID           string                     `json:"id"`
	Name         string                     `json:"name"`
	Organization string                     `json:"organization"`
	Status       EphemeralEnvironmentStatus `json:"status"`

	// Source information
	Source       EphemeralSource            `json:"source"`

	// Configuration
	TTL          time.Duration              `json:"ttl"`
	ExpiresAt    time.Time                  `json:"expiresAt"`

	// Resources
	Resources    []EphemeralResource        `json:"resources"`

	// Access
	PreviewURL   string                     `json:"previewUrl,omitempty"`
	Namespace    string                     `json:"namespace"`

	// Lifecycle
	CreatedAt    time.Time                  `json:"createdAt"`
	ReadyAt      *time.Time                 `json:"readyAt,omitempty"`
	DeletedAt    *time.Time                 `json:"deletedAt,omitempty"`

	// Cleanup configuration
	Cleanup      CleanupConfig              `json:"cleanup"`

	// Metadata
	Labels       map[string]string          `json:"labels,omitempty"`
	Annotations  map[string]string          `json:"annotations,omitempty"`

	// Error information
	Error        string                     `json:"error,omitempty"`
}

// EphemeralSource defines where the environment comes from
type EphemeralSource struct {
	Type       string `json:"type"` // pull-request, branch, manual
	URL        string `json:"url,omitempty"`
	Branch     string `json:"branch,omitempty"`
	CommitSHA  string `json:"commitSha,omitempty"`
	PRNumber   int    `json:"prNumber,omitempty"`
	Repository string `json:"repository,omitempty"`
}

// EphemeralResource represents a resource in the ephemeral environment
type EphemeralResource struct {
	Service    string                 `json:"service"`
	Image      string                 `json:"image,omitempty"`
	Size       string                 `json:"size,omitempty"`
	Replicas   int                    `json:"replicas,omitempty"`
	Data       *DataConfig            `json:"data,omitempty"`
	Status     string                 `json:"status"`
	URL        string                 `json:"url,omitempty"`
	Config     map[string]interface{} `json:"config,omitempty"`
}

// DataConfig defines data seeding configuration
type DataConfig struct {
	Source    string `json:"source"`    // production, staging, fixture
	Sanitize  bool   `json:"sanitize"`  // Remove PII
	Snapshot  string `json:"snapshot"`  // latest, specific snapshot ID
}

// CleanupConfig defines when to clean up the environment
type CleanupConfig struct {
	OnMerge   bool `json:"onMerge"`
	OnClose   bool `json:"onClose"`
	AfterTTL  bool `json:"afterTtl"`
}

// EphemeralManager manages ephemeral environments
type EphemeralManager struct {
	environments map[string]*EphemeralEnvironment
	mu           sync.RWMutex

	// Configuration
	defaultTTL    time.Duration
	maxTTL        time.Duration
	baseURL       string

	// Cleanup ticker
	cleanupTicker *time.Ticker
	stopCleanup   chan struct{}

	// Event listeners
	listeners     []EphemeralEventListener
	listenerMu    sync.RWMutex
}

// EphemeralEventListener receives ephemeral environment events
type EphemeralEventListener interface {
	OnEphemeralEvent(env *EphemeralEnvironment, event string)
}

// EphemeralManagerConfig configures the ephemeral manager
type EphemeralManagerConfig struct {
	DefaultTTL     time.Duration
	MaxTTL         time.Duration
	BaseURL        string
	CleanupInterval time.Duration
}

// NewEphemeralManager creates a new ephemeral environment manager
func NewEphemeralManager(config EphemeralManagerConfig) *EphemeralManager {
	if config.DefaultTTL == 0 {
		config.DefaultTTL = 7 * 24 * time.Hour // 7 days default
	}
	if config.MaxTTL == 0 {
		config.MaxTTL = 14 * 24 * time.Hour // 14 days max
	}
	if config.CleanupInterval == 0 {
		config.CleanupInterval = 5 * time.Minute
	}
	if config.BaseURL == "" {
		config.BaseURL = "preview.example.com"
	}

	m := &EphemeralManager{
		environments:  make(map[string]*EphemeralEnvironment),
		defaultTTL:    config.DefaultTTL,
		maxTTL:        config.MaxTTL,
		baseURL:       config.BaseURL,
		cleanupTicker: time.NewTicker(config.CleanupInterval),
		stopCleanup:   make(chan struct{}),
		listeners:     make([]EphemeralEventListener, 0),
	}

	// Start cleanup goroutine
	go m.runCleanup()

	return m
}

// Stop stops the manager and cleanup goroutine
func (m *EphemeralManager) Stop() {
	close(m.stopCleanup)
	m.cleanupTicker.Stop()
}

// Subscribe adds an event listener
func (m *EphemeralManager) Subscribe(listener EphemeralEventListener) {
	m.listenerMu.Lock()
	defer m.listenerMu.Unlock()
	m.listeners = append(m.listeners, listener)
}

// CreateForPullRequest creates an ephemeral environment for a PR
func (m *EphemeralManager) CreateForPullRequest(ctx context.Context, req PREnvironmentRequest) (*EphemeralEnvironment, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Check if environment already exists for this PR
	for _, env := range m.environments {
		if env.Source.Type == "pull-request" &&
		   env.Source.PRNumber == req.PRNumber &&
		   env.Source.Repository == req.Repository &&
		   env.Status != EphemeralStatusDeleted {
			return nil, fmt.Errorf("environment already exists for PR #%d", req.PRNumber)
		}
	}

	ttl := req.TTL
	if ttl == 0 {
		ttl = m.defaultTTL
	}
	if ttl > m.maxTTL {
		ttl = m.maxTTL
	}

	now := time.Now()
	env := &EphemeralEnvironment{
		ID:           uuid.New().String(),
		Name:         fmt.Sprintf("pr-%d", req.PRNumber),
		Organization: req.Organization,
		Status:       EphemeralStatusPending,
		Source: EphemeralSource{
			Type:       "pull-request",
			URL:        req.PRURL,
			Branch:     req.Branch,
			CommitSHA:  req.CommitSHA,
			PRNumber:   req.PRNumber,
			Repository: req.Repository,
		},
		TTL:       ttl,
		ExpiresAt: now.Add(ttl),
		Namespace: fmt.Sprintf("preview-pr-%d", req.PRNumber),
		CreatedAt: now,
		Cleanup: CleanupConfig{
			OnMerge:  true,
			OnClose:  true,
			AfterTTL: true,
		},
		Labels: map[string]string{
			"type":       "ephemeral",
			"source":     "pull-request",
			"pr":         fmt.Sprintf("%d", req.PRNumber),
			"repository": req.Repository,
		},
	}

	// Generate preview URL
	env.PreviewURL = fmt.Sprintf("https://pr-%d.%s", req.PRNumber, m.baseURL)

	// Add resources
	env.Resources = make([]EphemeralResource, 0, len(req.Services))
	for _, svc := range req.Services {
		resource := EphemeralResource{
			Service:  svc.Name,
			Image:    svc.Image,
			Size:     svc.Size,
			Replicas: svc.Replicas,
			Status:   "pending",
			Config:   svc.Config,
		}
		if svc.Data != nil {
			resource.Data = &DataConfig{
				Source:   svc.Data.Source,
				Sanitize: svc.Data.Sanitize,
				Snapshot: svc.Data.Snapshot,
			}
		}
		env.Resources = append(env.Resources, resource)
	}

	m.environments[env.ID] = env
	m.notifyListeners(env, "created")

	return env, nil
}

// PREnvironmentRequest contains data for creating a PR environment
type PREnvironmentRequest struct {
	Organization string
	Repository   string
	PRNumber     int
	PRURL        string
	Branch       string
	CommitSHA    string
	TTL          time.Duration
	Services     []ServiceRequest
}

// ServiceRequest defines a service to deploy in the ephemeral environment
type ServiceRequest struct {
	Name     string
	Image    string
	Size     string
	Replicas int
	Data     *DataConfig
	Config   map[string]interface{}
}

// CreateForBranch creates an ephemeral environment for a branch
func (m *EphemeralManager) CreateForBranch(ctx context.Context, org, repo, branch, commitSHA string, ttl time.Duration) (*EphemeralEnvironment, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if ttl == 0 {
		ttl = m.defaultTTL
	}
	if ttl > m.maxTTL {
		ttl = m.maxTTL
	}

	// Generate safe branch name for namespace
	safeBranch := sanitizeName(branch)

	now := time.Now()
	env := &EphemeralEnvironment{
		ID:           uuid.New().String(),
		Name:         fmt.Sprintf("branch-%s", safeBranch),
		Organization: org,
		Status:       EphemeralStatusPending,
		Source: EphemeralSource{
			Type:       "branch",
			Branch:     branch,
			CommitSHA:  commitSHA,
			Repository: repo,
		},
		TTL:       ttl,
		ExpiresAt: now.Add(ttl),
		Namespace: fmt.Sprintf("preview-%s", safeBranch),
		CreatedAt: now,
		Cleanup: CleanupConfig{
			AfterTTL: true,
		},
		Labels: map[string]string{
			"type":       "ephemeral",
			"source":     "branch",
			"branch":     branch,
			"repository": repo,
		},
	}

	env.PreviewURL = fmt.Sprintf("https://%s.%s", safeBranch, m.baseURL)

	m.environments[env.ID] = env
	m.notifyListeners(env, "created")

	return env, nil
}

// Get retrieves an ephemeral environment by ID
func (m *EphemeralManager) Get(id string) (*EphemeralEnvironment, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	env, ok := m.environments[id]
	if !ok {
		return nil, fmt.Errorf("ephemeral environment not found: %s", id)
	}
	return env, nil
}

// GetByPR retrieves an ephemeral environment by PR number
func (m *EphemeralManager) GetByPR(repo string, prNumber int) (*EphemeralEnvironment, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	for _, env := range m.environments {
		if env.Source.Type == "pull-request" &&
		   env.Source.PRNumber == prNumber &&
		   env.Source.Repository == repo &&
		   env.Status != EphemeralStatusDeleted {
			return env, nil
		}
	}
	return nil, fmt.Errorf("no environment found for PR #%d in %s", prNumber, repo)
}

// List returns all ephemeral environments, optionally filtered
func (m *EphemeralManager) List(org string, status EphemeralEnvironmentStatus) []*EphemeralEnvironment {
	m.mu.RLock()
	defer m.mu.RUnlock()

	result := make([]*EphemeralEnvironment, 0)
	for _, env := range m.environments {
		if org != "" && env.Organization != org {
			continue
		}
		if status != "" && env.Status != status {
			continue
		}
		result = append(result, env)
	}
	return result
}

// UpdateStatus updates the status of an ephemeral environment
func (m *EphemeralManager) UpdateStatus(id string, status EphemeralEnvironmentStatus, errorMsg string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	env, ok := m.environments[id]
	if !ok {
		return fmt.Errorf("ephemeral environment not found: %s", id)
	}

	oldStatus := env.Status
	env.Status = status
	env.Error = errorMsg

	if status == EphemeralStatusReady && env.ReadyAt == nil {
		now := time.Now()
		env.ReadyAt = &now
	}

	if status == EphemeralStatusDeleted {
		now := time.Now()
		env.DeletedAt = &now
	}

	if oldStatus != status {
		m.notifyListeners(env, "status_changed")
	}

	return nil
}

// UpdateResourceStatus updates the status of a resource in an environment
func (m *EphemeralManager) UpdateResourceStatus(envID, serviceName, status, url string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	env, ok := m.environments[envID]
	if !ok {
		return fmt.Errorf("ephemeral environment not found: %s", envID)
	}

	for i, res := range env.Resources {
		if res.Service == serviceName {
			env.Resources[i].Status = status
			if url != "" {
				env.Resources[i].URL = url
			}
			return nil
		}
	}

	return fmt.Errorf("resource not found: %s", serviceName)
}

// ExtendTTL extends the TTL of an ephemeral environment
func (m *EphemeralManager) ExtendTTL(id string, extension time.Duration) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	env, ok := m.environments[id]
	if !ok {
		return fmt.Errorf("ephemeral environment not found: %s", id)
	}

	newExpiry := env.ExpiresAt.Add(extension)
	maxExpiry := env.CreatedAt.Add(m.maxTTL)

	if newExpiry.After(maxExpiry) {
		newExpiry = maxExpiry
	}

	env.ExpiresAt = newExpiry
	m.notifyListeners(env, "ttl_extended")

	return nil
}

// Delete marks an ephemeral environment for deletion
func (m *EphemeralManager) Delete(id string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	env, ok := m.environments[id]
	if !ok {
		return fmt.Errorf("ephemeral environment not found: %s", id)
	}

	env.Status = EphemeralStatusDeleting
	m.notifyListeners(env, "deleting")

	return nil
}

// HandlePREvent handles PR events (merge, close)
func (m *EphemeralManager) HandlePREvent(repo string, prNumber int, event string) error {
	env, err := m.GetByPR(repo, prNumber)
	if err != nil {
		return err // No environment for this PR
	}

	switch event {
	case "merged":
		if env.Cleanup.OnMerge {
			return m.Delete(env.ID)
		}
	case "closed":
		if env.Cleanup.OnClose {
			return m.Delete(env.ID)
		}
	}

	return nil
}

// runCleanup runs the cleanup loop
func (m *EphemeralManager) runCleanup() {
	for {
		select {
		case <-m.stopCleanup:
			return
		case <-m.cleanupTicker.C:
			m.cleanupExpired()
		}
	}
}

// cleanupExpired marks expired environments for deletion
func (m *EphemeralManager) cleanupExpired() {
	m.mu.Lock()
	defer m.mu.Unlock()

	now := time.Now()
	for _, env := range m.environments {
		if env.Status != EphemeralStatusDeleted &&
		   env.Status != EphemeralStatusDeleting &&
		   env.Cleanup.AfterTTL &&
		   now.After(env.ExpiresAt) {
			env.Status = EphemeralStatusExpired
			m.notifyListeners(env, "expired")
		}
	}
}

// notifyListeners notifies all listeners of an event
func (m *EphemeralManager) notifyListeners(env *EphemeralEnvironment, event string) {
	m.listenerMu.RLock()
	listeners := make([]EphemeralEventListener, len(m.listeners))
	copy(listeners, m.listeners)
	m.listenerMu.RUnlock()

	for _, listener := range listeners {
		go listener.OnEphemeralEvent(env, event)
	}
}

// sanitizeName creates a safe name for Kubernetes resources
func sanitizeName(name string) string {
	result := make([]byte, 0, len(name))
	for _, c := range name {
		if (c >= 'a' && c <= 'z') || (c >= '0' && c <= '9') || c == '-' {
			result = append(result, byte(c))
		} else if c >= 'A' && c <= 'Z' {
			result = append(result, byte(c+32)) // lowercase
		} else if c == '/' || c == '_' || c == '.' {
			result = append(result, '-')
		}
	}

	// Trim leading/trailing dashes
	name = string(result)
	for len(name) > 0 && name[0] == '-' {
		name = name[1:]
	}
	for len(name) > 0 && name[len(name)-1] == '-' {
		name = name[:len(name)-1]
	}

	// Limit length
	if len(name) > 53 {
		name = name[:53]
	}

	return name
}
