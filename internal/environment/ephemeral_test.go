package environment

import (
	"context"
	"testing"
	"time"
)

func TestNewEphemeralManager(t *testing.T) {
	config := EphemeralManagerConfig{
		DefaultTTL: 24 * time.Hour,
		MaxTTL:     48 * time.Hour,
		BaseURL:    "test.example.com",
	}

	manager := NewEphemeralManager(config)
	defer manager.Stop()

	if manager == nil {
		t.Fatal("NewEphemeralManager returned nil")
	}

	if manager.defaultTTL != config.DefaultTTL {
		t.Errorf("expected defaultTTL %v, got %v", config.DefaultTTL, manager.defaultTTL)
	}

	if manager.baseURL != config.BaseURL {
		t.Errorf("expected baseURL %s, got %s", config.BaseURL, manager.baseURL)
	}
}

func TestCreateForPullRequest(t *testing.T) {
	manager := NewEphemeralManager(EphemeralManagerConfig{
		DefaultTTL: 24 * time.Hour,
		MaxTTL:     48 * time.Hour,
		BaseURL:    "preview.test.com",
	})
	defer manager.Stop()

	req := PREnvironmentRequest{
		Organization: "test-org",
		Repository:   "test-repo",
		PRNumber:     123,
		PRURL:        "https://github.com/test-org/test-repo/pull/123",
		Branch:       "feature/test",
		CommitSHA:    "abc123",
		Services: []ServiceRequest{
			{
				Name:  "api",
				Image: "api:pr-123",
			},
		},
	}

	env, err := manager.CreateForPullRequest(context.Background(), req)
	if err != nil {
		t.Fatalf("CreateForPullRequest failed: %v", err)
	}

	if env.Name != "pr-123" {
		t.Errorf("expected name pr-123, got %s", env.Name)
	}

	if env.Source.PRNumber != 123 {
		t.Errorf("expected PR number 123, got %d", env.Source.PRNumber)
	}

	if env.PreviewURL != "https://pr-123.preview.test.com" {
		t.Errorf("expected preview URL https://pr-123.preview.test.com, got %s", env.PreviewURL)
	}

	if len(env.Resources) != 1 {
		t.Errorf("expected 1 resource, got %d", len(env.Resources))
	}
}

func TestCreateForPullRequestDuplicate(t *testing.T) {
	manager := NewEphemeralManager(EphemeralManagerConfig{})
	defer manager.Stop()

	req := PREnvironmentRequest{
		Organization: "test-org",
		Repository:   "test-repo",
		PRNumber:     456,
	}

	_, err := manager.CreateForPullRequest(context.Background(), req)
	if err != nil {
		t.Fatalf("first CreateForPullRequest failed: %v", err)
	}

	_, err = manager.CreateForPullRequest(context.Background(), req)
	if err == nil {
		t.Error("expected error for duplicate PR environment")
	}
}

func TestCreateForBranch(t *testing.T) {
	manager := NewEphemeralManager(EphemeralManagerConfig{
		BaseURL: "preview.test.com",
	})
	defer manager.Stop()

	env, err := manager.CreateForBranch(context.Background(),
		"test-org", "test-repo", "feature/new-thing", "def456", 24*time.Hour)
	if err != nil {
		t.Fatalf("CreateForBranch failed: %v", err)
	}

	if env.Source.Type != "branch" {
		t.Errorf("expected source type branch, got %s", env.Source.Type)
	}

	if env.Source.Branch != "feature/new-thing" {
		t.Errorf("expected branch feature/new-thing, got %s", env.Source.Branch)
	}
}

func TestGetAndList(t *testing.T) {
	manager := NewEphemeralManager(EphemeralManagerConfig{})
	defer manager.Stop()

	// Create multiple environments
	for i := 1; i <= 3; i++ {
		req := PREnvironmentRequest{
			Organization: "test-org",
			Repository:   "test-repo",
			PRNumber:     i * 100,
		}
		manager.CreateForPullRequest(context.Background(), req)
	}

	// List all
	all := manager.List("", "")
	if len(all) != 3 {
		t.Errorf("expected 3 environments, got %d", len(all))
	}

	// List by org
	byOrg := manager.List("test-org", "")
	if len(byOrg) != 3 {
		t.Errorf("expected 3 environments for test-org, got %d", len(byOrg))
	}

	// List by other org
	byOtherOrg := manager.List("other-org", "")
	if len(byOtherOrg) != 0 {
		t.Errorf("expected 0 environments for other-org, got %d", len(byOtherOrg))
	}

	// Get by ID
	env := all[0]
	retrieved, err := manager.Get(env.ID)
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}
	if retrieved.ID != env.ID {
		t.Errorf("expected ID %s, got %s", env.ID, retrieved.ID)
	}
}

func TestGetByPR(t *testing.T) {
	manager := NewEphemeralManager(EphemeralManagerConfig{})
	defer manager.Stop()

	req := PREnvironmentRequest{
		Organization: "test-org",
		Repository:   "test-repo",
		PRNumber:     789,
	}
	created, _ := manager.CreateForPullRequest(context.Background(), req)

	retrieved, err := manager.GetByPR("test-repo", 789)
	if err != nil {
		t.Fatalf("GetByPR failed: %v", err)
	}

	if retrieved.ID != created.ID {
		t.Errorf("expected ID %s, got %s", created.ID, retrieved.ID)
	}

	// Non-existent PR
	_, err = manager.GetByPR("test-repo", 999)
	if err == nil {
		t.Error("expected error for non-existent PR")
	}
}

func TestUpdateStatus(t *testing.T) {
	manager := NewEphemeralManager(EphemeralManagerConfig{})
	defer manager.Stop()

	req := PREnvironmentRequest{
		Organization: "test-org",
		Repository:   "test-repo",
		PRNumber:     111,
	}
	env, _ := manager.CreateForPullRequest(context.Background(), req)

	// Update to provisioning
	err := manager.UpdateStatus(env.ID, EphemeralStatusProvisioning, "")
	if err != nil {
		t.Fatalf("UpdateStatus failed: %v", err)
	}

	updated, _ := manager.Get(env.ID)
	if updated.Status != EphemeralStatusProvisioning {
		t.Errorf("expected status provisioning, got %s", updated.Status)
	}

	// Update to ready
	err = manager.UpdateStatus(env.ID, EphemeralStatusReady, "")
	if err != nil {
		t.Fatalf("UpdateStatus failed: %v", err)
	}

	updated, _ = manager.Get(env.ID)
	if updated.Status != EphemeralStatusReady {
		t.Errorf("expected status ready, got %s", updated.Status)
	}
	if updated.ReadyAt == nil {
		t.Error("expected ReadyAt to be set")
	}
}

func TestExtendTTL(t *testing.T) {
	manager := NewEphemeralManager(EphemeralManagerConfig{
		DefaultTTL: 24 * time.Hour,
		MaxTTL:     48 * time.Hour,
	})
	defer manager.Stop()

	req := PREnvironmentRequest{
		Organization: "test-org",
		Repository:   "test-repo",
		PRNumber:     222,
	}
	env, _ := manager.CreateForPullRequest(context.Background(), req)
	originalExpiry := env.ExpiresAt

	// Extend by 12 hours
	err := manager.ExtendTTL(env.ID, 12*time.Hour)
	if err != nil {
		t.Fatalf("ExtendTTL failed: %v", err)
	}

	updated, _ := manager.Get(env.ID)
	if !updated.ExpiresAt.After(originalExpiry) {
		t.Error("expected ExpiresAt to be extended")
	}
}

func TestExtendTTLBeyondMax(t *testing.T) {
	manager := NewEphemeralManager(EphemeralManagerConfig{
		DefaultTTL: 24 * time.Hour,
		MaxTTL:     48 * time.Hour,
	})
	defer manager.Stop()

	req := PREnvironmentRequest{
		Organization: "test-org",
		Repository:   "test-repo",
		PRNumber:     333,
	}
	env, _ := manager.CreateForPullRequest(context.Background(), req)
	maxExpiry := env.CreatedAt.Add(48 * time.Hour)

	// Try to extend beyond max
	err := manager.ExtendTTL(env.ID, 100*time.Hour)
	if err != nil {
		t.Fatalf("ExtendTTL failed: %v", err)
	}

	updated, _ := manager.Get(env.ID)
	if updated.ExpiresAt.After(maxExpiry.Add(time.Second)) {
		t.Error("ExpiresAt should not exceed max TTL")
	}
}

func TestDelete(t *testing.T) {
	manager := NewEphemeralManager(EphemeralManagerConfig{})
	defer manager.Stop()

	req := PREnvironmentRequest{
		Organization: "test-org",
		Repository:   "test-repo",
		PRNumber:     444,
	}
	env, _ := manager.CreateForPullRequest(context.Background(), req)

	err := manager.Delete(env.ID)
	if err != nil {
		t.Fatalf("Delete failed: %v", err)
	}

	updated, _ := manager.Get(env.ID)
	if updated.Status != EphemeralStatusDeleting {
		t.Errorf("expected status deleting, got %s", updated.Status)
	}
}

func TestHandlePREvent(t *testing.T) {
	manager := NewEphemeralManager(EphemeralManagerConfig{})
	defer manager.Stop()

	req := PREnvironmentRequest{
		Organization: "test-org",
		Repository:   "test-repo",
		PRNumber:     555,
	}
	env, _ := manager.CreateForPullRequest(context.Background(), req)

	// Handle merge event
	err := manager.HandlePREvent("test-repo", 555, "merged")
	if err != nil {
		t.Fatalf("HandlePREvent failed: %v", err)
	}

	updated, _ := manager.Get(env.ID)
	if updated.Status != EphemeralStatusDeleting {
		t.Errorf("expected status deleting after merge, got %s", updated.Status)
	}
}

func TestSanitizeName(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"feature/test", "feature-test"},
		{"UPPERCASE", "uppercase"},
		{"with_underscore", "with-underscore"},
		{"with.dot", "with-dot"},
		{"-leading-dash", "leading-dash"},
		{"trailing-dash-", "trailing-dash"},
		{"a-very-long-branch-name-that-exceeds-the-maximum-length-allowed", "a-very-long-branch-name-that-exceeds-the-maximum-leng"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := sanitizeName(tt.input)
			if result != tt.expected {
				t.Errorf("sanitizeName(%s) = %s, want %s", tt.input, result, tt.expected)
			}
		})
	}
}
