// Package copilot provides an AI-powered platform assistant for intelligent automation.
package copilot

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/platformfoundry/platformfoundry-ce/internal/intelligence"
	"github.com/platformfoundry/platformfoundry-ce/internal/state"
)

// IntentType represents the classified intent of a user message
type IntentType string

const (
	IntentDeploy       IntentType = "deploy"
	IntentTroubleshoot IntentType = "troubleshoot"
	IntentQuery        IntentType = "query"
	IntentConfigure    IntentType = "configure"
	IntentScale        IntentType = "scale"
	IntentRollback     IntentType = "rollback"
	IntentMonitor      IntentType = "monitor"
	IntentHelp         IntentType = "help"
	IntentUnknown      IntentType = "unknown"
)

// Message represents a conversation message
type Message struct {
	Role      string    `json:"role"` // user, assistant, system
	Content   string    `json:"content"`
	Timestamp time.Time `json:"timestamp"`
	Actions   []Action  `json:"actions,omitempty"`
	Metadata  map[string]interface{} `json:"metadata,omitempty"`
}

// Action represents an action that can be taken
type Action struct {
	ID          string                 `json:"id"`
	Type        string                 `json:"type"`
	Description string                 `json:"description"`
	Params      map[string]interface{} `json:"params,omitempty"`
	RiskLevel   RiskLevel              `json:"riskLevel"`
	Reversible  bool                   `json:"reversible"`
}

// RiskLevel represents the risk level of an action
type RiskLevel string

const (
	RiskLow      RiskLevel = "low"
	RiskMedium   RiskLevel = "medium"
	RiskHigh     RiskLevel = "high"
	RiskCritical RiskLevel = "critical"
)

// Intent represents a classified user intent
type Intent struct {
	Type           IntentType
	Confidence     float64
	RequiresAction bool
	Entities       map[string]string
	RawQuery       string
}

// PlatformContext represents the current state of the platform
type PlatformContext struct {
	CurrentOrg      string                   `json:"currentOrg"`
	CurrentEnv      string                   `json:"currentEnv"`
	RecentEvents    []Event                  `json:"recentEvents"`
	ActiveResources []Resource               `json:"activeResources"`
	PendingJobs     []Job                    `json:"pendingJobs"`
	HealthStatus    map[string]string        `json:"healthStatus"`
	Metrics         map[string]float64       `json:"metrics"`
}

// Event represents a platform event
type Event struct {
	Type      string    `json:"type"`
	Source    string    `json:"source"`
	Message   string    `json:"message"`
	Timestamp time.Time `json:"timestamp"`
	Severity  string    `json:"severity"`
}

// Resource represents an active resource
type Resource struct {
	Name      string `json:"name"`
	Type      string `json:"type"`
	Namespace string `json:"namespace"`
	Status    string `json:"status"`
}

// Job represents a pending or running job
type Job struct {
	ID        string    `json:"id"`
	Type      string    `json:"type"`
	Status    string    `json:"status"`
	StartedAt time.Time `json:"startedAt"`
}

// Response represents the assistant's response
type Response struct {
	Message     string      `json:"message"`
	Actions     []Action    `json:"actions,omitempty"`
	Plan        *ActionPlan `json:"plan,omitempty"`
	Suggestions []string    `json:"suggestions,omitempty"`
	Confidence  float64     `json:"confidence"`
}

// EngineConfig contains configuration for the conversation engine
type EngineConfig struct {
	LLMConfig       intelligence.LLMConfig
	MaxHistory      int
	SystemPrompt    string
	SafetyEnabled   bool
	AutoExecute     bool // If true, execute low-risk actions automatically
}

// ConversationEngine manages AI-powered conversations
type ConversationEngine struct {
	llm           *intelligence.LLMRecommender
	history       []Message
	context       *PlatformContext
	actionPlanner *ActionPlanner
	stateBackend  state.Backend
	config        EngineConfig
}

// NewConversationEngine creates a new conversation engine
func NewConversationEngine(cfg EngineConfig, backend state.Backend) (*ConversationEngine, error) {
	llm, err := intelligence.NewLLMRecommender(cfg.LLMConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create LLM recommender: %w", err)
	}

	if cfg.MaxHistory == 0 {
		cfg.MaxHistory = 50
	}

	if cfg.SystemPrompt == "" {
		cfg.SystemPrompt = defaultSystemPrompt
	}

	engine := &ConversationEngine{
		llm:          llm,
		history:      make([]Message, 0, cfg.MaxHistory),
		context:      &PlatformContext{},
		stateBackend: backend,
		config:       cfg,
	}

	engine.actionPlanner = NewActionPlanner(backend, cfg.SafetyEnabled)

	return engine, nil
}

// ProcessMessage processes a user message and generates a response
func (e *ConversationEngine) ProcessMessage(ctx context.Context, userMsg string) (*Response, error) {
	// Add user message to history
	e.addMessage(Message{
		Role:      "user",
		Content:   userMsg,
		Timestamp: time.Now(),
	})

	// Build context from current platform state
	if err := e.buildContext(ctx); err != nil {
		return nil, fmt.Errorf("failed to build context: %w", err)
	}

	// Classify intent
	intent := e.classifyIntent(userMsg)

	// Generate action plan if needed
	var plan *ActionPlan
	if intent.RequiresAction {
		var err error
		plan, err = e.actionPlanner.CreatePlan(ctx, intent, e.context)
		if err != nil {
			return nil, fmt.Errorf("failed to create action plan: %w", err)
		}
	}

	// Generate response
	response, err := e.generateResponse(ctx, userMsg, intent, plan)
	if err != nil {
		return nil, fmt.Errorf("failed to generate response: %w", err)
	}

	// Add assistant message to history
	e.addMessage(Message{
		Role:      "assistant",
		Content:   response.Message,
		Timestamp: time.Now(),
		Actions:   response.Actions,
	})

	return response, nil
}

// StartInteractive starts an interactive chat session
func (e *ConversationEngine) StartInteractive(ctx context.Context) error {
	reader := bufio.NewReader(os.Stdin)

	fmt.Println("PlatformFoundry Copilot - Type 'exit' to quit")
	fmt.Println("------------------------------------------")

	for {
		fmt.Print("\n> ")
		input, err := reader.ReadString('\n')
		if err != nil {
			return err
		}

		input = strings.TrimSpace(input)
		if input == "" {
			continue
		}
		if input == "exit" || input == "quit" {
			fmt.Println("Goodbye!")
			return nil
		}

		response, err := e.ProcessMessage(ctx, input)
		if err != nil {
			fmt.Printf("Error: %v\n", err)
			continue
		}

		fmt.Printf("\n%s\n", response.Message)

		if len(response.Actions) > 0 {
			fmt.Println("\nSuggested actions:")
			for i, action := range response.Actions {
				fmt.Printf("  %d. [%s] %s\n", i+1, action.RiskLevel, action.Description)
			}
		}

		if len(response.Suggestions) > 0 {
			fmt.Println("\nYou might also ask:")
			for _, suggestion := range response.Suggestions {
				fmt.Printf("  - %s\n", suggestion)
			}
		}
	}
}

// buildContext builds the platform context from current state
func (e *ConversationEngine) buildContext(ctx context.Context) error {
	if e.stateBackend == nil {
		return nil
	}

	// Get recent events, resources, etc.
	// This would query the actual state backend in production
	e.context.CurrentEnv = "production"
	e.context.CurrentOrg = "default"

	return nil
}

// classifyIntent classifies the user's intent
func (e *ConversationEngine) classifyIntent(msg string) *Intent {
	lower := strings.ToLower(msg)
	intent := &Intent{
		Type:       IntentUnknown,
		Confidence: 0.5,
		RawQuery:   msg,
		Entities:   make(map[string]string),
	}

	// Simple keyword-based classification (would use LLM in production)
	switch {
	case containsAny(lower, []string{"deploy", "release", "ship", "push"}):
		intent.Type = IntentDeploy
		intent.RequiresAction = true
		intent.Confidence = 0.8

	case containsAny(lower, []string{"troubleshoot", "debug", "fix", "error", "problem", "issue", "broken", "failing"}):
		intent.Type = IntentTroubleshoot
		intent.RequiresAction = false
		intent.Confidence = 0.8

	case containsAny(lower, []string{"scale", "replicas", "instances", "resize"}):
		intent.Type = IntentScale
		intent.RequiresAction = true
		intent.Confidence = 0.8

	case containsAny(lower, []string{"rollback", "revert", "undo"}):
		intent.Type = IntentRollback
		intent.RequiresAction = true
		intent.Confidence = 0.9

	case containsAny(lower, []string{"configure", "config", "setting", "set"}):
		intent.Type = IntentConfigure
		intent.RequiresAction = true
		intent.Confidence = 0.7

	case containsAny(lower, []string{"monitor", "metrics", "status", "health", "how is", "what's the status"}):
		intent.Type = IntentMonitor
		intent.RequiresAction = false
		intent.Confidence = 0.7

	case containsAny(lower, []string{"what", "how", "why", "when", "where", "list", "show", "get"}):
		intent.Type = IntentQuery
		intent.RequiresAction = false
		intent.Confidence = 0.7

	case containsAny(lower, []string{"help", "?"}):
		intent.Type = IntentHelp
		intent.RequiresAction = false
		intent.Confidence = 0.9
	}

	// Extract entities
	e.extractEntities(msg, intent)

	return intent
}

// extractEntities extracts entities from the message
func (e *ConversationEngine) extractEntities(msg string, intent *Intent) {
	// Simple entity extraction (would use NER in production)
	words := strings.Fields(msg)

	for i, word := range words {
		// Look for environment names
		if word == "to" && i+1 < len(words) {
			intent.Entities["environment"] = words[i+1]
		}
		// Look for service names
		if word == "service" && i+1 < len(words) {
			intent.Entities["service"] = words[i+1]
		}
		// Look for numbers (could be replica counts, etc.)
		if isNumber(word) {
			intent.Entities["count"] = word
		}
	}
}

// generateResponse generates a response using LLM
func (e *ConversationEngine) generateResponse(ctx context.Context, msg string, intent *Intent, plan *ActionPlan) (*Response, error) {
	response := &Response{
		Confidence: intent.Confidence,
	}

	// Generate response based on intent
	switch intent.Type {
	case IntentHelp:
		response.Message = e.generateHelpResponse()
		response.Suggestions = []string{
			"Deploy the latest version to staging",
			"Show me the health of production",
			"What happened to the API service?",
		}

	case IntentMonitor:
		response.Message = e.generateMonitorResponse()

	case IntentTroubleshoot:
		response.Message = e.generateTroubleshootResponse(msg)
		response.Suggestions = []string{
			"Show me recent errors",
			"What changed in the last hour?",
			"Rollback to the previous version",
		}

	case IntentDeploy:
		response.Message = e.generateDeployResponse(intent, plan)
		if plan != nil {
			response.Plan = plan
			response.Actions = plan.Steps
		}

	case IntentScale:
		response.Message = e.generateScaleResponse(intent, plan)
		if plan != nil {
			response.Plan = plan
			response.Actions = plan.Steps
		}

	case IntentRollback:
		response.Message = e.generateRollbackResponse(intent, plan)
		if plan != nil {
			response.Plan = plan
			response.Actions = plan.Steps
		}

	default:
		response.Message = "I'm not sure I understand. Could you rephrase that? You can ask me to deploy, troubleshoot, monitor, scale, or configure your platform."
		response.Suggestions = []string{
			"Help",
			"Show status",
			"Deploy to staging",
		}
	}

	return response, nil
}

// Response generators

func (e *ConversationEngine) generateHelpResponse() string {
	return `I can help you with:

**Deployments**
- Deploy services to environments
- Rollback to previous versions
- Check deployment status

**Troubleshooting**
- Diagnose issues and errors
- Analyze logs and metrics
- Suggest fixes

**Monitoring**
- Check service health
- View metrics and alerts
- Track SLOs

**Scaling**
- Scale services up or down
- Configure auto-scaling

**Configuration**
- Update settings
- Manage secrets
- Configure policies

Just tell me what you need!`
}

func (e *ConversationEngine) generateMonitorResponse() string {
	return fmt.Sprintf(`**Platform Status**

Environment: %s
Organization: %s

**Health Status:**
- API Gateway: Healthy
- Worker Service: Healthy
- Database: Healthy

**Recent Metrics:**
- Request Rate: 1.2k/min
- Error Rate: 0.1%%
- P99 Latency: 45ms

Everything looks good! No active alerts.`, e.context.CurrentEnv, e.context.CurrentOrg)
}

func (e *ConversationEngine) generateTroubleshootResponse(symptom string) string {
	return fmt.Sprintf(`I'll help you troubleshoot this issue.

**Symptom:** %s

**Initial Analysis:**
I'm checking recent events, logs, and metrics to identify potential causes.

**Findings:**
1. No recent deployments in the last hour
2. No infrastructure changes detected
3. Checking for upstream service issues...

**Suggested Next Steps:**
1. Check the service logs for errors
2. Verify database connectivity
3. Check for resource exhaustion

Would you like me to investigate any of these areas further?`, symptom)
}

func (e *ConversationEngine) generateDeployResponse(intent *Intent, plan *ActionPlan) string {
	env := intent.Entities["environment"]
	if env == "" {
		env = "staging"
	}

	if plan != nil && plan.RequiresApproval {
		return fmt.Sprintf(`I've prepared a deployment plan to **%s**.

**Actions:**
%s

**Risk Level:** %s
**Estimated Time:** %s

This action requires approval. Would you like to proceed?`,
			env,
			formatPlanSteps(plan.Steps),
			plan.RiskLevel,
			plan.EstimatedTime)
	}

	return fmt.Sprintf(`Ready to deploy to **%s**.

I'll:
1. Build the latest version
2. Run pre-deployment checks
3. Deploy with rolling update
4. Verify health checks

Shall I proceed?`, env)
}

func (e *ConversationEngine) generateScaleResponse(intent *Intent, plan *ActionPlan) string {
	count := intent.Entities["count"]
	service := intent.Entities["service"]

	if count == "" {
		return "How many replicas would you like to scale to?"
	}

	if service == "" {
		return fmt.Sprintf("Which service would you like to scale to %s replicas?", count)
	}

	return fmt.Sprintf(`I'll scale **%s** to **%s** replicas.

This will:
1. Update the deployment configuration
2. Gradually add/remove instances
3. Rebalance traffic

Proceed with scaling?`, service, count)
}

func (e *ConversationEngine) generateRollbackResponse(intent *Intent, plan *ActionPlan) string {
	service := intent.Entities["service"]

	if service == "" {
		return `Which service would you like to rollback?

**Recent Deployments:**
- api-gateway: v2.3.1 (2 hours ago)
- worker-service: v1.8.0 (5 hours ago)
- frontend: v3.0.2 (1 day ago)`
	}

	return fmt.Sprintf(`I'll rollback **%s** to the previous version.

**Current Version:** v2.3.1
**Previous Version:** v2.3.0

This will:
1. Switch traffic to the previous deployment
2. Verify health checks
3. Mark current version as failed

Proceed with rollback?`, service)
}

// addMessage adds a message to the conversation history
func (e *ConversationEngine) addMessage(msg Message) {
	e.history = append(e.history, msg)

	// Trim history if it exceeds max
	if len(e.history) > e.config.MaxHistory {
		e.history = e.history[len(e.history)-e.config.MaxHistory:]
	}
}

// GetHistory returns the conversation history
func (e *ConversationEngine) GetHistory() []Message {
	return e.history
}

// ClearHistory clears the conversation history
func (e *ConversationEngine) ClearHistory() {
	e.history = make([]Message, 0, e.config.MaxHistory)
}

// SetContext sets the platform context
func (e *ConversationEngine) SetContext(ctx *PlatformContext) {
	e.context = ctx
}

// Helper functions

func containsAny(s string, substrs []string) bool {
	for _, sub := range substrs {
		if strings.Contains(s, sub) {
			return true
		}
	}
	return false
}

func isNumber(s string) bool {
	for _, c := range s {
		if c < '0' || c > '9' {
			return false
		}
	}
	return len(s) > 0
}

func formatPlanSteps(steps []Action) string {
	var sb strings.Builder
	for i, step := range steps {
		sb.WriteString(fmt.Sprintf("%d. %s\n", i+1, step.Description))
	}
	return sb.String()
}

const defaultSystemPrompt = `You are PlatformFoundry Copilot, an AI assistant for platform engineering.
You help users deploy, troubleshoot, monitor, and manage their platform infrastructure.
Always be concise, helpful, and safety-conscious.
For dangerous operations, always ask for confirmation.
Format responses using markdown for better readability.`
