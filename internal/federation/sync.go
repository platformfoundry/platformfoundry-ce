package federation

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"sync"
	"time"
)

// ConfigSyncManager manages federated configuration synchronization
type ConfigSyncManager struct {
	controller    *Controller
	syncPolicies  map[string]*SyncPolicy
	syncStatus    map[string]*SyncStatus
	conflictHandler ConflictHandler
	mu            sync.RWMutex
	stopCh        chan struct{}
	running       bool
}

// SyncPolicy defines how resources should be synchronized
type SyncPolicy struct {
	ID              string            `json:"id" yaml:"id"`
	Name            string            `json:"name" yaml:"name"`
	SourceCluster   string            `json:"sourceCluster" yaml:"sourceCluster"`
	TargetClusters  []string          `json:"targetClusters" yaml:"targetClusters"`
	Resources       []SyncResource    `json:"resources" yaml:"resources"`
	Mode            SyncMode          `json:"mode" yaml:"mode"`
	Interval        time.Duration     `json:"interval" yaml:"interval"`
	ConflictPolicy  ConflictPolicy    `json:"conflictPolicy" yaml:"conflictPolicy"`
	Enabled         bool              `json:"enabled" yaml:"enabled"`
	CreatedAt       time.Time         `json:"createdAt" yaml:"createdAt"`
	UpdatedAt       time.Time         `json:"updatedAt" yaml:"updatedAt"`
}

// SyncResource defines a resource to synchronize
type SyncResource struct {
	Kind           string            `json:"kind" yaml:"kind"`       // ConfigMap, Secret, etc.
	Namespace      string            `json:"namespace" yaml:"namespace"`
	NamePattern    string            `json:"namePattern,omitempty" yaml:"namePattern,omitempty"` // Regex pattern
	LabelSelector  map[string]string `json:"labelSelector,omitempty" yaml:"labelSelector,omitempty"`
	ExcludeNames   []string          `json:"excludeNames,omitempty" yaml:"excludeNames,omitempty"`
	Transform      *TransformConfig  `json:"transform,omitempty" yaml:"transform,omitempty"`
}

// TransformConfig defines transformations to apply during sync
type TransformConfig struct {
	RenameNamespace string            `json:"renameNamespace,omitempty" yaml:"renameNamespace,omitempty"`
	AddLabels       map[string]string `json:"addLabels,omitempty" yaml:"addLabels,omitempty"`
	AddAnnotations  map[string]string `json:"addAnnotations,omitempty" yaml:"addAnnotations,omitempty"`
	RemoveFields    []string          `json:"removeFields,omitempty" yaml:"removeFields,omitempty"`
}

// SyncMode defines the synchronization mode
type SyncMode string

const (
	SyncModePush       SyncMode = "push"       // Push from source to targets
	SyncModePull       SyncMode = "pull"       // Targets pull from source
	SyncModeBidirectional SyncMode = "bidirectional" // Two-way sync
)

// ConflictPolicy defines how to handle conflicts
type ConflictPolicy string

const (
	ConflictPolicySourceWins ConflictPolicy = "source_wins"
	ConflictPolicyTargetWins ConflictPolicy = "target_wins"
	ConflictPolicyNewerWins  ConflictPolicy = "newer_wins"
	ConflictPolicyManual     ConflictPolicy = "manual"
)

// SyncStatus tracks synchronization status
type SyncStatus struct {
	PolicyID       string       `json:"policyId"`
	LastSyncTime   *time.Time   `json:"lastSyncTime,omitempty"`
	NextSyncTime   *time.Time   `json:"nextSyncTime,omitempty"`
	Status         string       `json:"status"` // syncing, synced, error, pending
	Error          string       `json:"error,omitempty"`
	ResourcesSync  int          `json:"resourcesSynced"`
	Conflicts      []Conflict   `json:"conflicts,omitempty"`
	ClusterStatus  map[string]string `json:"clusterStatus"`
}

// Conflict represents a sync conflict
type Conflict struct {
	Resource      string    `json:"resource"`
	SourceVersion string    `json:"sourceVersion"`
	TargetVersion string    `json:"targetVersion"`
	TargetCluster string    `json:"targetCluster"`
	DetectedAt    time.Time `json:"detectedAt"`
	Resolved      bool      `json:"resolved"`
	Resolution    string    `json:"resolution,omitempty"`
}

// SyncedResource represents a synchronized resource
type SyncedResource struct {
	Kind         string                 `json:"kind"`
	Name         string                 `json:"name"`
	Namespace    string                 `json:"namespace"`
	Cluster      string                 `json:"cluster"`
	Version      string                 `json:"version"`
	Hash         string                 `json:"hash"`
	Data         map[string]interface{} `json:"data,omitempty"`
	SyncedAt     time.Time              `json:"syncedAt"`
}

// ConflictHandler interface for handling sync conflicts
type ConflictHandler interface {
	HandleConflict(ctx context.Context, conflict Conflict, policy *SyncPolicy) (resolution string, err error)
}

// ClusterClient interface for cluster operations
type ClusterClient interface {
	GetResources(ctx context.Context, kind, namespace string, labelSelector map[string]string) ([]SyncedResource, error)
	ApplyResource(ctx context.Context, resource SyncedResource) error
	DeleteResource(ctx context.Context, kind, name, namespace string) error
	GetResourceVersion(ctx context.Context, kind, name, namespace string) (string, error)
}

// NewConfigSyncManager creates a new config sync manager
func NewConfigSyncManager(controller *Controller) *ConfigSyncManager {
	return &ConfigSyncManager{
		controller:   controller,
		syncPolicies: make(map[string]*SyncPolicy),
		syncStatus:   make(map[string]*SyncStatus),
		stopCh:       make(chan struct{}),
	}
}

// WithConflictHandler sets the conflict handler
func (m *ConfigSyncManager) WithConflictHandler(handler ConflictHandler) *ConfigSyncManager {
	m.conflictHandler = handler
	return m
}

// CreateSyncPolicy creates a new sync policy
func (m *ConfigSyncManager) CreateSyncPolicy(ctx context.Context, policy *SyncPolicy) error {
	if policy.ID == "" {
		policy.ID = fmt.Sprintf("sync-%s-%d", policy.Name, time.Now().UnixNano())
	}
	policy.CreatedAt = time.Now()
	policy.UpdatedAt = time.Now()

	// Validate source cluster exists
	if _, err := m.controller.GetCluster(ctx, policy.SourceCluster); err != nil {
		return fmt.Errorf("source cluster not found: %s", policy.SourceCluster)
	}

	// Validate target clusters exist
	for _, target := range policy.TargetClusters {
		if _, err := m.controller.GetCluster(ctx, target); err != nil {
			return fmt.Errorf("target cluster not found: %s", target)
		}
	}

	m.mu.Lock()
	m.syncPolicies[policy.ID] = policy
	m.syncStatus[policy.ID] = &SyncStatus{
		PolicyID:      policy.ID,
		Status:        "pending",
		ClusterStatus: make(map[string]string),
	}
	m.mu.Unlock()

	return nil
}

// GetSyncPolicy returns a sync policy by ID
func (m *ConfigSyncManager) GetSyncPolicy(policyID string) (*SyncPolicy, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	policy, ok := m.syncPolicies[policyID]
	if !ok {
		return nil, fmt.Errorf("sync policy not found: %s", policyID)
	}
	return policy, nil
}

// ListSyncPolicies returns all sync policies
func (m *ConfigSyncManager) ListSyncPolicies() []*SyncPolicy {
	m.mu.RLock()
	defer m.mu.RUnlock()

	policies := make([]*SyncPolicy, 0, len(m.syncPolicies))
	for _, p := range m.syncPolicies {
		policies = append(policies, p)
	}
	return policies
}

// DeleteSyncPolicy deletes a sync policy
func (m *ConfigSyncManager) DeleteSyncPolicy(policyID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, ok := m.syncPolicies[policyID]; !ok {
		return fmt.Errorf("sync policy not found: %s", policyID)
	}

	delete(m.syncPolicies, policyID)
	delete(m.syncStatus, policyID)
	return nil
}

// GetSyncStatus returns the sync status for a policy
func (m *ConfigSyncManager) GetSyncStatus(policyID string) (*SyncStatus, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	status, ok := m.syncStatus[policyID]
	if !ok {
		return nil, fmt.Errorf("sync status not found: %s", policyID)
	}
	return status, nil
}

// TriggerSync manually triggers synchronization for a policy
func (m *ConfigSyncManager) TriggerSync(ctx context.Context, policyID string) error {
	m.mu.RLock()
	policy, ok := m.syncPolicies[policyID]
	if !ok {
		m.mu.RUnlock()
		return fmt.Errorf("sync policy not found: %s", policyID)
	}
	m.mu.RUnlock()

	return m.syncPolicy(ctx, policy)
}

// syncPolicy performs synchronization for a policy
func (m *ConfigSyncManager) syncPolicy(ctx context.Context, policy *SyncPolicy) error {
	m.mu.Lock()
	status := m.syncStatus[policy.ID]
	status.Status = "syncing"
	m.mu.Unlock()

	defer func() {
		m.mu.Lock()
		now := time.Now()
		status.LastSyncTime = &now
		if policy.Interval > 0 {
			next := now.Add(policy.Interval)
			status.NextSyncTime = &next
		}
		m.mu.Unlock()
	}()

	// In a real implementation, this would:
	// 1. Fetch resources from source cluster
	// 2. Compare with target clusters
	// 3. Detect and handle conflicts
	// 4. Apply changes to target clusters

	resourcesSynced := 0

	for _, target := range policy.TargetClusters {
		m.mu.Lock()
		status.ClusterStatus[target] = "syncing"
		m.mu.Unlock()

		// Simulate sync (in real implementation, use ClusterClient)
		for _, res := range policy.Resources {
			// Sync each resource type
			_ = res // Would sync resources of this type
			resourcesSynced++
		}

		m.mu.Lock()
		status.ClusterStatus[target] = "synced"
		m.mu.Unlock()
	}

	m.mu.Lock()
	status.Status = "synced"
	status.ResourcesSync = resourcesSynced
	status.Error = ""
	m.mu.Unlock()

	return nil
}

// Start starts the background sync loop
func (m *ConfigSyncManager) Start(ctx context.Context) error {
	m.mu.Lock()
	if m.running {
		m.mu.Unlock()
		return fmt.Errorf("sync manager already running")
	}
	m.running = true
	m.stopCh = make(chan struct{})
	m.mu.Unlock()

	go m.syncLoop(ctx)
	return nil
}

// Stop stops the sync manager
func (m *ConfigSyncManager) Stop() {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.running {
		close(m.stopCh)
		m.running = false
	}
}

// syncLoop runs periodic synchronization
func (m *ConfigSyncManager) syncLoop(ctx context.Context) {
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-m.stopCh:
			return
		case <-ticker.C:
			m.runScheduledSyncs(ctx)
		}
	}
}

// runScheduledSyncs runs any scheduled syncs
func (m *ConfigSyncManager) runScheduledSyncs(ctx context.Context) {
	m.mu.RLock()
	policies := make([]*SyncPolicy, 0)
	for _, p := range m.syncPolicies {
		if p.Enabled {
			policies = append(policies, p)
		}
	}
	m.mu.RUnlock()

	now := time.Now()
	for _, policy := range policies {
		m.mu.RLock()
		status := m.syncStatus[policy.ID]
		shouldSync := status.NextSyncTime == nil || now.After(*status.NextSyncTime)
		m.mu.RUnlock()

		if shouldSync {
			go m.syncPolicy(ctx, policy)
		}
	}
}

// HashResource calculates a hash of resource data for comparison
func HashResource(data map[string]interface{}) string {
	h := sha256.New()
	h.Write([]byte(fmt.Sprintf("%v", data)))
	return hex.EncodeToString(h.Sum(nil))[:16]
}

// DefaultSyncPolicy returns a default sync policy for common configs
func DefaultSyncPolicy(name, sourceCluster string, targetClusters []string) *SyncPolicy {
	return &SyncPolicy{
		Name:           name,
		SourceCluster:  sourceCluster,
		TargetClusters: targetClusters,
		Resources: []SyncResource{
			{Kind: "ConfigMap", Namespace: "default", LabelSelector: map[string]string{"sync": "true"}},
			{Kind: "Secret", Namespace: "default", LabelSelector: map[string]string{"sync": "true"}},
		},
		Mode:           SyncModePush,
		Interval:       5 * time.Minute,
		ConflictPolicy: ConflictPolicySourceWins,
		Enabled:        true,
	}
}
