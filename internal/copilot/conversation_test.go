package copilot

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestIntentClassification(t *testing.T) {
	engine := &ConversationEngine{
		config: EngineConfig{},
	}

	tests := []struct {
		name           string
		message        string
		expectedType   IntentType
		requiresAction bool
	}{
		{
			name:           "deploy intent",
			message:        "Deploy the latest version to staging",
			expectedType:   IntentDeploy,
			requiresAction: true,
		},
		{
			name:           "troubleshoot intent",
			message:        "The API is returning errors",
			expectedType:   IntentTroubleshoot,
			requiresAction: false,
		},
		{
			name:           "scale intent",
			message:        "Scale the worker service to 5 replicas",
			expectedType:   IntentScale,
			requiresAction: true,
		},
		{
			name:           "rollback intent",
			message:        "Rollback the last deployment",
			expectedType:   IntentRollback,
			requiresAction: true,
		},
		{
			name:           "monitor intent",
			message:        "What's the status of production?",
			expectedType:   IntentMonitor,
			requiresAction: false,
		},
		{
			name:           "query intent",
			message:        "List all deployments",
			expectedType:   IntentQuery,
			requiresAction: false,
		},
		{
			name:           "help intent",
			message:        "Help",
			expectedType:   IntentHelp,
			requiresAction: false,
		},
		{
			name:           "configure intent",
			message:        "Configure the database connection",
			expectedType:   IntentConfigure,
			requiresAction: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			intent := engine.classifyIntent(tt.message)
			assert.Equal(t, tt.expectedType, intent.Type)
			assert.Equal(t, tt.requiresAction, intent.RequiresAction)
		})
	}
}

func TestEntityExtraction(t *testing.T) {
	engine := &ConversationEngine{
		config: EngineConfig{},
	}

	tests := []struct {
		name     string
		message  string
		expected map[string]string
	}{
		{
			name:    "extract environment",
			message: "Deploy to production",
			expected: map[string]string{
				"environment": "production",
			},
		},
		{
			name:    "extract service",
			message: "Restart service api-gateway",
			expected: map[string]string{
				"service": "api-gateway",
			},
		},
		{
			name:    "extract count",
			message: "Scale to 5 replicas",
			expected: map[string]string{
				"count": "5",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			intent := engine.classifyIntent(tt.message)
			for key, expectedValue := range tt.expected {
				assert.Equal(t, expectedValue, intent.Entities[key])
			}
		})
	}
}

func TestMessageHistory(t *testing.T) {
	engine := &ConversationEngine{
		config: EngineConfig{
			MaxHistory: 5,
		},
		history: make([]Message, 0, 5),
	}

	// Add messages
	for i := 0; i < 10; i++ {
		engine.addMessage(Message{
			Role:      "user",
			Content:   "test message",
			Timestamp: time.Now(),
		})
	}

	// Should only keep MaxHistory messages
	assert.Equal(t, 5, len(engine.history))
}

func TestClearHistory(t *testing.T) {
	engine := &ConversationEngine{
		config: EngineConfig{
			MaxHistory: 10,
		},
		history: make([]Message, 0, 10),
	}

	engine.addMessage(Message{
		Role:    "user",
		Content: "test",
	})

	assert.Equal(t, 1, len(engine.GetHistory()))

	engine.ClearHistory()
	assert.Equal(t, 0, len(engine.GetHistory()))
}

func TestContainsAny(t *testing.T) {
	tests := []struct {
		str      string
		substrs  []string
		expected bool
	}{
		{"deploy to production", []string{"deploy", "release"}, true},
		{"hello world", []string{"deploy", "release"}, false},
		{"DEPLOY NOW", []string{"deploy"}, false}, // case sensitive
		{"let's release it", []string{"deploy", "release"}, true},
	}

	for _, tt := range tests {
		result := containsAny(tt.str, tt.substrs)
		assert.Equal(t, tt.expected, result)
	}
}

func TestIsNumber(t *testing.T) {
	tests := []struct {
		input    string
		expected bool
	}{
		{"123", true},
		{"0", true},
		{"abc", false},
		{"12a", false},
		{"", false},
		{"-5", false}, // negative not supported
	}

	for _, tt := range tests {
		result := isNumber(tt.input)
		assert.Equal(t, tt.expected, result)
	}
}

func TestRiskLevel(t *testing.T) {
	assert.Equal(t, RiskLevel("low"), RiskLow)
	assert.Equal(t, RiskLevel("medium"), RiskMedium)
	assert.Equal(t, RiskLevel("high"), RiskHigh)
	assert.Equal(t, RiskLevel("critical"), RiskCritical)
}

func TestIntentType(t *testing.T) {
	assert.Equal(t, IntentType("deploy"), IntentDeploy)
	assert.Equal(t, IntentType("troubleshoot"), IntentTroubleshoot)
	assert.Equal(t, IntentType("query"), IntentQuery)
	assert.Equal(t, IntentType("configure"), IntentConfigure)
	assert.Equal(t, IntentType("scale"), IntentScale)
	assert.Equal(t, IntentType("rollback"), IntentRollback)
	assert.Equal(t, IntentType("monitor"), IntentMonitor)
	assert.Equal(t, IntentType("help"), IntentHelp)
	assert.Equal(t, IntentType("unknown"), IntentUnknown)
}

func TestMessage(t *testing.T) {
	now := time.Now()
	msg := Message{
		Role:      "user",
		Content:   "test message",
		Timestamp: now,
		Actions: []Action{
			{
				ID:          "action-1",
				Type:        "deploy",
				Description: "Deploy service",
				RiskLevel:   RiskLow,
			},
		},
		Metadata: map[string]interface{}{
			"source": "cli",
		},
	}

	assert.Equal(t, "user", msg.Role)
	assert.Equal(t, "test message", msg.Content)
	assert.Equal(t, now, msg.Timestamp)
	assert.Len(t, msg.Actions, 1)
	assert.Equal(t, "deploy", msg.Actions[0].Type)
}

func TestAction(t *testing.T) {
	action := Action{
		ID:          "test-action",
		Type:        "deploy",
		Description: "Deploy to production",
		Params: map[string]interface{}{
			"environment": "production",
		},
		RiskLevel:  RiskHigh,
		Reversible: true,
	}

	assert.Equal(t, "test-action", action.ID)
	assert.Equal(t, "deploy", action.Type)
	assert.Equal(t, RiskHigh, action.RiskLevel)
	assert.True(t, action.Reversible)
	assert.Equal(t, "production", action.Params["environment"])
}

func TestPlatformContext(t *testing.T) {
	ctx := PlatformContext{
		CurrentOrg: "my-org",
		CurrentEnv: "production",
		RecentEvents: []Event{
			{
				Type:    "deploy",
				Source:  "api",
				Message: "Deployed v1.0",
			},
		},
		ActiveResources: []Resource{
			{
				Name:   "api-gateway",
				Type:   "deployment",
				Status: "running",
			},
		},
		PendingJobs: []Job{
			{
				ID:     "job-1",
				Type:   "build",
				Status: "pending",
			},
		},
		HealthStatus: map[string]string{
			"api": "healthy",
		},
		Metrics: map[string]float64{
			"cpu": 45.5,
		},
	}

	assert.Equal(t, "my-org", ctx.CurrentOrg)
	assert.Equal(t, "production", ctx.CurrentEnv)
	assert.Len(t, ctx.RecentEvents, 1)
	assert.Len(t, ctx.ActiveResources, 1)
	assert.Len(t, ctx.PendingJobs, 1)
	assert.Equal(t, "healthy", ctx.HealthStatus["api"])
	assert.Equal(t, 45.5, ctx.Metrics["cpu"])
}

func TestFormatPlanSteps(t *testing.T) {
	steps := []Action{
		{
			ID:          "1",
			Description: "Build application",
		},
		{
			ID:          "2",
			Description: "Run tests",
		},
		{
			ID:          "3",
			Description: "Deploy",
		},
	}

	result := formatPlanSteps(steps)

	assert.Contains(t, result, "1. Build application")
	assert.Contains(t, result, "2. Run tests")
	assert.Contains(t, result, "3. Deploy")
}
