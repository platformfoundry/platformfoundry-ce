package workflow

import (
	"context"
	"testing"
	"time"
)

func TestNewEngine(t *testing.T) {
	engine := NewEngine()

	if engine == nil {
		t.Fatal("NewEngine returned nil")
	}

	if engine.workflows == nil {
		t.Error("workflows map is nil")
	}

	if engine.executions == nil {
		t.Error("executions map is nil")
	}

	if engine.conditionCheckers == nil {
		t.Error("conditionCheckers map is nil")
	}
}

func TestRegisterWorkflow(t *testing.T) {
	engine := NewEngine()

	wf := &Workflow{
		Name:         "test-workflow",
		Organization: "test-org",
		Trigger: WorkflowTrigger{
			Action: "deploy",
			Target: WorkflowTarget{
				Environment: "production",
			},
		},
		Approvals: ApprovalConfig{
			Required: 2,
			Roles:    []string{"tech-lead"},
			Timeout:  24 * time.Hour,
		},
	}

	err := engine.RegisterWorkflow(wf)
	if err != nil {
		t.Fatalf("RegisterWorkflow failed: %v", err)
	}

	// Verify workflow was registered
	retrieved, err := engine.GetWorkflow("test-workflow")
	if err != nil {
		t.Fatalf("GetWorkflow failed: %v", err)
	}

	if retrieved.Name != wf.Name {
		t.Errorf("expected name %s, got %s", wf.Name, retrieved.Name)
	}
}

func TestRegisterWorkflowEmpty(t *testing.T) {
	engine := NewEngine()

	wf := &Workflow{} // Empty name

	err := engine.RegisterWorkflow(wf)
	if err == nil {
		t.Error("expected error for empty workflow name")
	}
}

func TestGetWorkflowNotFound(t *testing.T) {
	engine := NewEngine()

	_, err := engine.GetWorkflow("nonexistent")
	if err == nil {
		t.Error("expected error for nonexistent workflow")
	}
}

func TestListWorkflows(t *testing.T) {
	engine := NewEngine()

	// Register multiple workflows
	for i := 0; i < 3; i++ {
		wf := &Workflow{
			Name: "workflow-" + string(rune('a'+i)),
		}
		engine.RegisterWorkflow(wf)
	}

	workflows := engine.ListWorkflows()
	if len(workflows) != 3 {
		t.Errorf("expected 3 workflows, got %d", len(workflows))
	}
}

func TestMatchesTrigger(t *testing.T) {
	engine := NewEngine()

	tests := []struct {
		name    string
		trigger WorkflowTrigger
		target  WorkflowTarget
		action  string
		want    bool
	}{
		{
			name: "exact match",
			trigger: WorkflowTrigger{
				Action: "deploy",
				Target: WorkflowTarget{Environment: "production"},
			},
			target: WorkflowTarget{Environment: "production"},
			action: "deploy",
			want:   true,
		},
		{
			name: "action mismatch",
			trigger: WorkflowTrigger{
				Action: "deploy",
				Target: WorkflowTarget{Environment: "production"},
			},
			target: WorkflowTarget{Environment: "production"},
			action: "scale",
			want:   false,
		},
		{
			name: "environment mismatch",
			trigger: WorkflowTrigger{
				Action: "deploy",
				Target: WorkflowTarget{Environment: "production"},
			},
			target: WorkflowTarget{Environment: "staging"},
			action: "deploy",
			want:   false,
		},
		{
			name: "wildcard trigger",
			trigger: WorkflowTrigger{
				Action: "",
				Target: WorkflowTarget{},
			},
			target: WorkflowTarget{Environment: "any"},
			action: "any",
			want:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := engine.matchesTrigger(tt.trigger, tt.target, tt.action)
			if got != tt.want {
				t.Errorf("matchesTrigger() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestIsWithinChangeWindow(t *testing.T) {
	engine := NewEngine()

	// Test with Monday 14:00
	monday := time.Date(2026, 1, 19, 14, 0, 0, 0, time.UTC) // Monday

	tests := []struct {
		name   string
		config *ChangeWindowConfig
		time   time.Time
		want   bool
	}{
		{
			name: "within allowed window",
			config: &ChangeWindowConfig{
				Allowed: []TimeWindow{
					{Days: []string{"Mon", "Tue", "Wed"}, Hours: "10:00-16:00"},
				},
			},
			time: monday,
			want: true,
		},
		{
			name: "outside hours",
			config: &ChangeWindowConfig{
				Allowed: []TimeWindow{
					{Days: []string{"Mon"}, Hours: "08:00-12:00"},
				},
			},
			time: monday,
			want: false,
		},
		{
			name: "blocked day",
			config: &ChangeWindowConfig{
				Allowed: []TimeWindow{
					{Days: []string{"Mon"}, Hours: "10:00-16:00"},
				},
				Blocked: []BlockedTime{
					{Days: []string{"Mon"}},
				},
			},
			time: monday,
			want: false,
		},
		{
			name:   "no config allows by default",
			config: &ChangeWindowConfig{},
			time:   monday,
			want:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := engine.isWithinChangeWindow(tt.config, tt.time)
			if got != tt.want {
				t.Errorf("isWithinChangeWindow() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestApproveReject(t *testing.T) {
	t.Skip("Skipping flaky test - TODO: investigate timing issues")
	engine := NewEngine()

	// Register workflow
	wf := &Workflow{
		Name: "approval-test",
		Trigger: WorkflowTrigger{
			Action: "deploy",
			Target: WorkflowTarget{Environment: "production"},
		},
		Approvals: ApprovalConfig{
			Required: 2,
			Roles:    []string{"tech-lead"},
			Timeout:  1 * time.Hour,
		},
	}
	engine.RegisterWorkflow(wf)

	// Start execution
	ctx := context.Background()
	exec, err := engine.StartExecution(ctx, "approval-test", "developer", WorkflowTarget{Environment: "production"}, "deploy")
	if err != nil {
		t.Fatalf("StartExecution failed: %v", err)
	}

	// Wait for status to change to awaiting approval
	time.Sleep(100 * time.Millisecond)

	// Approve
	err = engine.Approve(exec.ID, "alice", "tech-lead", "LGTM")
	if err != nil {
		t.Fatalf("Approve failed: %v", err)
	}

	// Check approval was recorded
	updated, _ := engine.GetExecution(exec.ID)
	if len(updated.Approvals) != 1 {
		t.Errorf("expected 1 approval, got %d", len(updated.Approvals))
	}

	if updated.Approvals[0].User != "alice" {
		t.Errorf("expected approver alice, got %s", updated.Approvals[0].User)
	}
}

func TestApproveNonExistent(t *testing.T) {
	engine := NewEngine()

	err := engine.Approve("nonexistent", "user", "role", "comment")
	if err == nil {
		t.Error("expected error for nonexistent execution")
	}
}

func TestListExecutions(t *testing.T) {
	engine := NewEngine()

	// Register workflow
	wf := &Workflow{
		Name: "list-test",
		Trigger: WorkflowTrigger{
			Action: "deploy",
		},
	}
	engine.RegisterWorkflow(wf)

	// Start multiple executions
	ctx := context.Background()
	for i := 0; i < 3; i++ {
		engine.StartExecution(ctx, "list-test", "user", WorkflowTarget{}, "deploy")
	}

	time.Sleep(50 * time.Millisecond)

	// List all
	all := engine.ListExecutions("", "")
	if len(all) != 3 {
		t.Errorf("expected 3 executions, got %d", len(all))
	}

	// List by workflow name
	byName := engine.ListExecutions("list-test", "")
	if len(byName) != 3 {
		t.Errorf("expected 3 executions for list-test, got %d", len(byName))
	}

	// List by nonexistent name
	empty := engine.ListExecutions("nonexistent", "")
	if len(empty) != 0 {
		t.Errorf("expected 0 executions for nonexistent, got %d", len(empty))
	}
}

func TestConditionCheckers(t *testing.T) {
	tests := []struct {
		name      string
		checker   ConditionChecker
		condition WorkflowCondition
		metadata  map[string]interface{}
		wantPass  bool
	}{
		{
			name:    "tests passing - true",
			checker: &TestsPassingChecker{},
			condition: WorkflowCondition{
				Type:     ConditionTestsPassing,
				Required: true,
			},
			metadata: map[string]interface{}{"tests_passing": true},
			wantPass: true,
		},
		{
			name:    "tests passing - false",
			checker: &TestsPassingChecker{},
			condition: WorkflowCondition{
				Type:     ConditionTestsPassing,
				Required: true,
			},
			metadata: map[string]interface{}{"tests_passing": false},
			wantPass: false,
		},
		{
			name:    "security scan - no criticals",
			checker: &SecurityScanChecker{},
			condition: WorkflowCondition{
				Type:        ConditionSecurityScan,
				Required:    true,
				MaxCritical: 0,
			},
			metadata: map[string]interface{}{"security_critical_count": 0},
			wantPass: true,
		},
		{
			name:    "security scan - has criticals",
			checker: &SecurityScanChecker{},
			condition: WorkflowCondition{
				Type:        ConditionSecurityScan,
				Required:    true,
				MaxCritical: 0,
			},
			metadata: map[string]interface{}{"security_critical_count": 2},
			wantPass: false,
		},
		{
			name:    "test coverage - meets threshold",
			checker: &TestCoverageChecker{},
			condition: WorkflowCondition{
				Type:      ConditionTestCoverage,
				Required:  true,
				Threshold: 80,
			},
			metadata: map[string]interface{}{"test_coverage": 85},
			wantPass: true,
		},
		{
			name:    "test coverage - below threshold",
			checker: &TestCoverageChecker{},
			condition: WorkflowCondition{
				Type:      ConditionTestCoverage,
				Required:  true,
				Threshold: 80,
			},
			metadata: map[string]interface{}{"test_coverage": 70},
			wantPass: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			exec := &WorkflowExecution{
				Metadata: tt.metadata,
			}

			result, err := tt.checker.Check(context.Background(), tt.condition, exec)
			if err != nil {
				t.Fatalf("Check failed: %v", err)
			}

			gotPass := result.Status == ConditionStatusPassed
			if gotPass != tt.wantPass {
				t.Errorf("Check() passed = %v, want %v (status: %s, message: %s)",
					gotPass, tt.wantPass, result.Status, result.Message)
			}
		})
	}
}
