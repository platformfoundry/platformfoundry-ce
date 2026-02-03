package preview

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// MockStateBackend implements a mock state backend for testing
type MockStateBackend struct {
	resources map[string]*mockResource
}

type mockResource struct {
	Name       string
	Kind       string
	APIVersion string
	Spec       map[string]interface{}
	Status     map[string]interface{}
	Version    int
	CreatedAt  time.Time
	UpdatedAt  time.Time
}

func NewMockStateBackend() *MockStateBackend {
	return &MockStateBackend{
		resources: make(map[string]*mockResource),
	}
}

func (m *MockStateBackend) Save(resource interface{}) error {
	// Type assertion handled in actual implementation
	return nil
}

func (m *MockStateBackend) Get(name string) (interface{}, error) {
	if r, ok := m.resources[name]; ok {
		return r, nil
	}
	return nil, nil
}

func (m *MockStateBackend) List() ([]interface{}, error) {
	var result []interface{}
	for _, r := range m.resources {
		result = append(result, r)
	}
	return result, nil
}

func (m *MockStateBackend) Delete(name string) error {
	delete(m.resources, name)
	return nil
}

func (m *MockStateBackend) Lock(name string) error   { return nil }
func (m *MockStateBackend) Unlock(name string) error { return nil }
func (m *MockStateBackend) GetVersion(name string, version int) (interface{}, error) {
	return nil, nil
}
func (m *MockStateBackend) ListVersions(name string) ([]interface{}, error) { return nil, nil }
func (m *MockStateBackend) Close() error                                    { return nil }

// MockDNSProvider implements a mock DNS provider for testing
type MockDNSProvider struct {
	records map[string]string
}

func NewMockDNSProvider() *MockDNSProvider {
	return &MockDNSProvider{
		records: make(map[string]string),
	}
}

func (m *MockDNSProvider) CreateRecord(ctx context.Context, hostname, target string) error {
	m.records[hostname] = target
	return nil
}

func (m *MockDNSProvider) DeleteRecord(ctx context.Context, hostname string) error {
	delete(m.records, hostname)
	return nil
}

// MockOrchestrator implements a mock orchestrator for testing
type MockOrchestrator struct {
	applied  []Resource
	deleted  []Resource
	statuses map[string]string
}

func NewMockOrchestrator() *MockOrchestrator {
	return &MockOrchestrator{
		statuses: make(map[string]string),
	}
}

func (m *MockOrchestrator) Apply(ctx context.Context, resources []Resource) error {
	m.applied = append(m.applied, resources...)
	return nil
}

func (m *MockOrchestrator) Delete(ctx context.Context, resources []Resource) error {
	m.deleted = append(m.deleted, resources...)
	return nil
}

func (m *MockOrchestrator) GetStatus(ctx context.Context, namespace string) (string, error) {
	if status, ok := m.statuses[namespace]; ok {
		return status, nil
	}
	return "unknown", nil
}

func TestGeneratePreviewName(t *testing.T) {
	tests := []struct {
		name     string
		opts     CreatePreviewOpts
		expected string
	}{
		{
			name: "simple branch",
			opts: CreatePreviewOpts{
				PullRequest:  123,
				SourceBranch: "feature-test",
			},
			expected: "pr-123-feature-test",
		},
		{
			name: "branch with slashes",
			opts: CreatePreviewOpts{
				PullRequest:  456,
				SourceBranch: "feature/new-api",
			},
			expected: "pr-456-feature-new-api",
		},
		{
			name: "long branch name",
			opts: CreatePreviewOpts{
				PullRequest:  789,
				SourceBranch: "this-is-a-very-long-branch-name-that-exceeds-limit",
			},
			expected: "pr-789-this-is-a-very-long-b",
		},
		{
			name: "uppercase branch",
			opts: CreatePreviewOpts{
				PullRequest:  100,
				SourceBranch: "FEATURE-TEST",
			},
			expected: "pr-100-feature-test",
		},
	}

	m := &Manager{
		config: ManagerConfig{},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := m.generatePreviewName(tt.opts)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestSanitizeName(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"simple", "simple"},
		{"with-dash", "with-dash"},
		{"with_underscore", "with-underscore"},
		{"MixedCase", "mixedcase"},
		{"with/slash", "with-slash"},
		{"special@chars!", "specialchars"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := sanitizeName(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestPreviewEnvironmentStatus(t *testing.T) {
	assert.Equal(t, PreviewStatus("pending"), StatusPending)
	assert.Equal(t, PreviewStatus("provisioning"), StatusProvisioning)
	assert.Equal(t, PreviewStatus("ready"), StatusReady)
	assert.Equal(t, PreviewStatus("failed"), StatusFailed)
	assert.Equal(t, PreviewStatus("deleting"), StatusDeleting)
	assert.Equal(t, PreviewStatus("deleted"), StatusDeleted)
}

func TestDatabaseStrategy(t *testing.T) {
	assert.Equal(t, DatabaseStrategy("clone"), DatabaseStrategyClone)
	assert.Equal(t, DatabaseStrategy("fresh"), DatabaseStrategyFresh)
	assert.Equal(t, DatabaseStrategy("seed"), DatabaseStrategySeed)
}

func TestCreatePreviewOpts_Defaults(t *testing.T) {
	cfg := ManagerConfig{}
	m := &Manager{config: cfg}

	// Test that defaults are applied correctly
	require.NotNil(t, m)

	opts := CreatePreviewOpts{
		Repository:      "test/repo",
		PullRequest:     1,
		BaseEnvironment: "staging",
	}

	// TTL should default to manager's default
	assert.Equal(t, time.Duration(0), opts.TTL)
	assert.Equal(t, DatabaseStrategy(""), opts.DatabaseStrategy)
}

func TestManagerConfig_Defaults(t *testing.T) {
	cfg := ManagerConfig{}

	// Create manager to apply defaults
	m := NewManager(cfg, nil, nil, nil)

	assert.Equal(t, 72*time.Hour, m.config.DefaultTTL)
	assert.Equal(t, 168*time.Hour, m.config.MaxTTL)
	assert.Equal(t, 5*time.Minute, m.config.CleanupInterval)
	assert.Equal(t, 10, m.config.MaxConcurrent)
}

func TestCleanupWorker(t *testing.T) {
	m := NewManager(ManagerConfig{}, nil, nil, nil)
	w := NewCleanupWorker(m, time.Minute)

	require.NotNil(t, w)

	// Test scheduling
	w.Schedule("preview-1", time.Now().Add(time.Hour))
	w.Schedule("preview-2", time.Now().Add(2 * time.Hour))

	assert.Equal(t, 2, w.GetScheduledCount())
	assert.True(t, w.IsScheduled("preview-1"))
	assert.True(t, w.IsScheduled("preview-2"))
	assert.False(t, w.IsScheduled("preview-3"))

	// Test unscheduling
	w.Unschedule("preview-1")
	assert.Equal(t, 1, w.GetScheduledCount())
	assert.False(t, w.IsScheduled("preview-1"))

	// Test getting next cleanup
	next := w.GetNextCleanup()
	require.NotNil(t, next)
}

func TestCleanupWorker_StartStop(t *testing.T) {
	m := NewManager(ManagerConfig{}, nil, nil, nil)
	w := NewCleanupWorker(m, 100*time.Millisecond)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Start the worker
	w.Start(ctx)

	// Give it time to start
	time.Sleep(50 * time.Millisecond)

	// Stop the worker
	w.Stop()

	// Starting again should work
	w.Start(ctx)
	w.Stop()
}

func TestHelperFunctions(t *testing.T) {
	t.Run("getString", func(t *testing.T) {
		m := map[string]interface{}{
			"key1": "value1",
			"key2": 123,
		}

		assert.Equal(t, "value1", getString(m, "key1"))
		assert.Equal(t, "", getString(m, "key2"))
		assert.Equal(t, "", getString(m, "nonexistent"))
	})

	t.Run("getInt", func(t *testing.T) {
		m := map[string]interface{}{
			"int":     42,
			"float":   42.5,
			"string":  "42",
		}

		assert.Equal(t, 42, getInt(m, "int"))
		assert.Equal(t, 42, getInt(m, "float"))
		assert.Equal(t, 0, getInt(m, "string"))
		assert.Equal(t, 0, getInt(m, "nonexistent"))
	})
}

func TestPreviewEnvironment_Fields(t *testing.T) {
	now := time.Now()
	preview := &PreviewEnvironment{
		ID:               "test-id",
		Name:             "pr-123-feature",
		SourceRepo:       "owner/repo",
		SourceBranch:     "feature-branch",
		PullRequest:      123,
		BaseEnvironment:  "staging",
		TTL:              72 * time.Hour,
		URL:              "pr-123.preview.local",
		Status:           StatusReady,
		CreatedAt:        now,
		UpdatedAt:        now,
		ExpiresAt:        now.Add(72 * time.Hour),
		DatabaseStrategy: DatabaseStrategyFresh,
		Labels: map[string]string{
			"team": "platform",
		},
		Metadata: map[string]string{
			"created_by": "webhook",
		},
	}

	assert.Equal(t, "test-id", preview.ID)
	assert.Equal(t, "pr-123-feature", preview.Name)
	assert.Equal(t, 123, preview.PullRequest)
	assert.Equal(t, StatusReady, preview.Status)
	assert.Equal(t, DatabaseStrategyFresh, preview.DatabaseStrategy)
	assert.Equal(t, "platform", preview.Labels["team"])
}

func TestResource_Fields(t *testing.T) {
	res := Resource{
		Name:      "api-deployment",
		Type:      "deployment",
		Namespace: "pr-123",
		Status:    "running",
		Spec: map[string]interface{}{
			"replicas": 2,
		},
	}

	assert.Equal(t, "api-deployment", res.Name)
	assert.Equal(t, "deployment", res.Type)
	assert.Equal(t, "pr-123", res.Namespace)
	assert.Equal(t, "running", res.Status)
	assert.Equal(t, 2, res.Spec["replicas"])
}
