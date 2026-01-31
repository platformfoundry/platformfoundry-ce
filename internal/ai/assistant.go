package ai

import (
	"context"
	"fmt"
	"strings"
	"time"
)

// Assistant provides an AI-powered interface to the platform
type Assistant struct {
	provider      LLMProvider
	toolRegistry  ToolRegistry
	systemPrompt  string
	maxIterations int
	conversation  []Message
}

// ToolRegistry interface for tool management
type ToolRegistry interface {
	GetDefinitions() []ToolDefinition
	Execute(ctx context.Context, name string, args map[string]interface{}) (string, error)
}

// AssistantConfig configures the assistant
type AssistantConfig struct {
	Provider      LLMProvider
	ToolRegistry  ToolRegistry
	SystemPrompt  string
	MaxIterations int
}

// NewAssistant creates a new AI assistant
func NewAssistant(config AssistantConfig) *Assistant {
	systemPrompt := config.SystemPrompt
	if systemPrompt == "" {
		systemPrompt = DefaultSystemPrompt()
	}

	maxIterations := config.MaxIterations
	if maxIterations == 0 {
		maxIterations = 10
	}

	return &Assistant{
		provider:      config.Provider,
		toolRegistry:  config.ToolRegistry,
		systemPrompt:  systemPrompt,
		maxIterations: maxIterations,
		conversation:  make([]Message, 0),
	}
}

// DefaultSystemPrompt returns the default system prompt for the assistant
func DefaultSystemPrompt() string {
	return `You are an expert platform engineering assistant for Platform Foundry, an Internal Developer Platform (IDP) orchestration tool.

Your role is to help users:
- Understand and manage their platform infrastructure
- Monitor service health and troubleshoot issues
- Detect and resolve configuration drift
- Analyze costs and optimize resource usage
- Deploy workloads and provision infrastructure through promises
- Compare environments and track deployments

You have access to tools that can query real-time platform data. Use these tools to provide accurate, up-to-date information.

Guidelines:
1. Be concise but thorough in your explanations
2. When analyzing issues, consider multiple factors (health, drift, costs, security)
3. Provide actionable recommendations when identifying problems
4. Use the appropriate tools to gather information before answering questions
5. If you're uncertain about something, say so and suggest how to investigate further
6. Format your responses for readability using markdown when appropriate

When users ask about the platform:
- Use list_services, get_health_score, or check_drift for operational queries
- Use analyze_costs for cost-related questions
- Use compare_environments to understand differences between environments
- Use list_promises and list_workloads for infrastructure inventory
- Use get_recommendations for improvement suggestions
- Use get_recent_events to understand what's been happening

Always explain your findings in a way that helps users understand the implications and take appropriate action.`
}

// Chat sends a message and returns the assistant's response
func (a *Assistant) Chat(ctx context.Context, userMessage string) (*ChatResponse, error) {
	// Add user message to conversation
	a.conversation = append(a.conversation, NewUserMessage(userMessage))

	// Get tool definitions
	var tools []ToolDefinition
	if a.toolRegistry != nil {
		tools = a.toolRegistry.GetDefinitions()
	}

	// Iterate until we get a final response or hit max iterations
	for i := 0; i < a.maxIterations; i++ {
		// Build completion request
		req := &CompletionRequest{
			Messages:     a.conversation,
			SystemPrompt: a.systemPrompt,
			Tools:        tools,
			Temperature:  0.7,
		}

		// Get completion from provider
		resp, err := a.provider.Complete(ctx, req)
		if err != nil {
			return nil, fmt.Errorf("completion failed: %w", err)
		}

		// If no tool calls, we have our final response
		if !resp.IsToolCallResponse() {
			// Add assistant message to conversation
			a.conversation = append(a.conversation, NewAssistantMessage(resp.Content))

			return &ChatResponse{
				Content:   resp.Content,
				ToolsUsed: a.extractToolsUsed(),
				Usage:     resp.Usage,
			}, nil
		}

		// Execute tool calls
		assistantMsg := Message{
			Role:      "assistant",
			Content:   resp.Content,
			ToolCalls: resp.ToolCalls,
		}
		a.conversation = append(a.conversation, assistantMsg)

		// Process each tool call
		for _, tc := range resp.ToolCalls {
			result, err := a.executeToolCall(ctx, tc)
			if err != nil {
				result = fmt.Sprintf("Error executing tool %s: %s", tc.Name, err.Error())
			}

			// Add tool result to conversation
			a.conversation = append(a.conversation, NewToolResultMessage(tc.ID, result))
		}
	}

	return nil, fmt.Errorf("max iterations (%d) exceeded without final response", a.maxIterations)
}

// StreamChat sends a message and streams the response
func (a *Assistant) StreamChat(ctx context.Context, userMessage string) (<-chan StreamResponse, error) {
	ch := make(chan StreamResponse, 100)

	go func() {
		defer close(ch)

		// For now, use non-streaming chat and send result
		resp, err := a.Chat(ctx, userMessage)
		if err != nil {
			ch <- StreamResponse{Error: err, Done: true}
			return
		}

		ch <- StreamResponse{
			Content:   resp.Content,
			ToolsUsed: resp.ToolsUsed,
			Done:      true,
		}
	}()

	return ch, nil
}

// ChatResponse represents the assistant's response
type ChatResponse struct {
	Content   string
	ToolsUsed []string
	Usage     TokenUsage
}

// StreamResponse represents a streaming response chunk
type StreamResponse struct {
	Content   string
	ToolsUsed []string
	Done      bool
	Error     error
}

// ClearConversation resets the conversation history
func (a *Assistant) ClearConversation() {
	a.conversation = make([]Message, 0)
}

// GetConversation returns the current conversation history
func (a *Assistant) GetConversation() []Message {
	return a.conversation
}

// SetSystemPrompt updates the system prompt
func (a *Assistant) SetSystemPrompt(prompt string) {
	a.systemPrompt = prompt
}

// executeToolCall executes a single tool call
func (a *Assistant) executeToolCall(ctx context.Context, tc ToolCall) (string, error) {
	if a.toolRegistry == nil {
		return "", fmt.Errorf("no tool registry configured")
	}

	// Execute the tool
	result, err := a.toolRegistry.Execute(ctx, tc.Name, tc.Arguments)
	if err != nil {
		return "", err
	}

	return result, nil
}

// extractToolsUsed returns a list of tools used in the conversation
func (a *Assistant) extractToolsUsed() []string {
	toolsUsed := make(map[string]bool)
	for _, msg := range a.conversation {
		for _, tc := range msg.ToolCalls {
			toolsUsed[tc.Name] = true
		}
	}

	result := make([]string, 0, len(toolsUsed))
	for tool := range toolsUsed {
		result = append(result, tool)
	}
	return result
}

// QuickAsk is a convenience method for single-question interactions
func QuickAsk(ctx context.Context, provider LLMProvider, registry ToolRegistry, question string) (*ChatResponse, error) {
	assistant := NewAssistant(AssistantConfig{
		Provider:     provider,
		ToolRegistry: registry,
	})

	return assistant.Chat(ctx, question)
}

// ConversationSummary provides a summary of the conversation
type ConversationSummary struct {
	MessageCount   int
	ToolCallCount  int
	ToolsUsed      []string
	TotalTokens    int
	StartTime      time.Time
	LastUpdateTime time.Time
}

// GetConversationSummary returns a summary of the current conversation
func (a *Assistant) GetConversationSummary() ConversationSummary {
	toolsUsed := make(map[string]bool)
	toolCallCount := 0

	for _, msg := range a.conversation {
		for _, tc := range msg.ToolCalls {
			toolsUsed[tc.Name] = true
			toolCallCount++
		}
	}

	tools := make([]string, 0, len(toolsUsed))
	for tool := range toolsUsed {
		tools = append(tools, tool)
	}

	return ConversationSummary{
		MessageCount:  len(a.conversation),
		ToolCallCount: toolCallCount,
		ToolsUsed:     tools,
	}
}

// ContextualAssistant extends Assistant with platform-specific context
type ContextualAssistant struct {
	*Assistant
	environment string
	team        string
	service     string
}

// NewContextualAssistant creates an assistant with platform context
func NewContextualAssistant(config AssistantConfig, env, team, service string) *ContextualAssistant {
	// Enhance system prompt with context
	contextPrompt := config.SystemPrompt
	if contextPrompt == "" {
		contextPrompt = DefaultSystemPrompt()
	}

	var contextParts []string
	if env != "" {
		contextParts = append(contextParts, fmt.Sprintf("Current environment: %s", env))
	}
	if team != "" {
		contextParts = append(contextParts, fmt.Sprintf("Team context: %s", team))
	}
	if service != "" {
		contextParts = append(contextParts, fmt.Sprintf("Service focus: %s", service))
	}

	if len(contextParts) > 0 {
		contextPrompt += "\n\nContext:\n" + strings.Join(contextParts, "\n")
	}

	config.SystemPrompt = contextPrompt

	return &ContextualAssistant{
		Assistant:   NewAssistant(config),
		environment: env,
		team:        team,
		service:     service,
	}
}

// GetEnvironment returns the current environment context
func (ca *ContextualAssistant) GetEnvironment() string {
	return ca.environment
}

// SetEnvironment updates the environment context
func (ca *ContextualAssistant) SetEnvironment(env string) {
	ca.environment = env
}

// GetTeam returns the current team context
func (ca *ContextualAssistant) GetTeam() string {
	return ca.team
}

// SetTeam updates the team context
func (ca *ContextualAssistant) SetTeam(team string) {
	ca.team = team
}

// GetService returns the current service context
func (ca *ContextualAssistant) GetService() string {
	return ca.service
}

// SetService updates the service context
func (ca *ContextualAssistant) SetService(service string) {
	ca.service = service
}

// SuggestedQuestions returns contextual question suggestions
func (ca *ContextualAssistant) SuggestedQuestions() []string {
	questions := []string{
		"What's the current health status of the platform?",
		"Are there any active drift issues?",
		"What recommendations do you have for improving reliability?",
	}

	if ca.service != "" {
		questions = append(questions,
			fmt.Sprintf("What's the health status of %s?", ca.service),
			fmt.Sprintf("Show me recent events for %s", ca.service),
		)
	}

	if ca.environment != "" {
		questions = append(questions,
			fmt.Sprintf("Compare %s with production", ca.environment),
			fmt.Sprintf("What costs are associated with %s?", ca.environment),
		)
	}

	return questions
}
