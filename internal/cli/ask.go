package cli

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/platformfoundry/pf-ce/internal/ai"
	"github.com/platformfoundry/pf-ce/internal/ai/providers"
	"github.com/platformfoundry/pf-ce/internal/ai/tools"
	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(askCmd)

	askCmd.Flags().StringP("provider", "p", "claude", "AI provider (claude, openai)")
	askCmd.Flags().StringP("model", "m", "", "Model to use (provider-specific)")
	askCmd.Flags().StringP("environment", "e", "", "Environment context")
	askCmd.Flags().StringP("service", "s", "", "Service context")
	askCmd.Flags().Bool("interactive", false, "Start interactive chat mode")
	askCmd.Flags().Bool("no-tools", false, "Disable tool usage")
	askCmd.Flags().Int("max-iterations", 10, "Maximum tool call iterations")
}

var askCmd = &cobra.Command{
	Use:   "ask [question]",
	Short: "Ask the AI assistant about your platform",
	Long: `Ask the AI assistant questions about your platform infrastructure.

The assistant can help you with:
  - Checking service health and status
  - Detecting and analyzing drift
  - Analyzing costs and optimization opportunities
  - Comparing environments
  - Understanding platform events and alerts
  - Getting recommendations for improvements

Examples:
  # Ask a question directly
  pf ask "What is the health status of the platform?"

  # Ask about a specific service
  pf ask -s order-service "Why is this service degraded?"

  # Start interactive chat mode
  pf ask --interactive

  # Use a specific provider
  pf ask -p openai "List all services with drift"

Environment variables:
  ANTHROPIC_API_KEY - API key for Claude provider
  OPENAI_API_KEY    - API key for OpenAI provider`,
	RunE: runAsk,
}

func runAsk(cmd *cobra.Command, args []string) error {
	interactive, _ := cmd.Flags().GetBool("interactive")
	providerName, _ := cmd.Flags().GetString("provider")
	model, _ := cmd.Flags().GetString("model")
	environment, _ := cmd.Flags().GetString("environment")
	service, _ := cmd.Flags().GetString("service")
	noTools, _ := cmd.Flags().GetBool("no-tools")
	maxIterations, _ := cmd.Flags().GetInt("max-iterations")

	// Get question from args or interactive mode
	var question string
	if len(args) > 0 {
		question = strings.Join(args, " ")
	} else if !interactive {
		fmt.Println("Usage: pf ask [question] or pf ask --interactive")
		fmt.Println("\nUse 'pf ask --help' for more information.")
		return nil
	}

	// Create provider
	provider, err := createProvider(providerName, model)
	if err != nil {
		return fmt.Errorf("failed to create provider: %w", err)
	}

	// Create tool registry
	var registry ai.ToolRegistry
	if !noTools {
		registry = tools.NewToolRegistry()
	}

	// Create assistant
	config := ai.AssistantConfig{
		Provider:      provider,
		ToolRegistry:  registry,
		MaxIterations: maxIterations,
	}

	var assistant *ai.ContextualAssistant
	if environment != "" || service != "" {
		assistant = ai.NewContextualAssistant(config, environment, "", service)
	} else {
		assistant = ai.NewContextualAssistant(config, "", "", "")
	}

	ctx := context.Background()

	if interactive {
		return runInteractiveMode(ctx, assistant)
	}

	// Single question mode
	return askQuestion(ctx, assistant, question)
}

func createProvider(name, model string) (ai.LLMProvider, error) {
	switch name {
	case "claude", "anthropic":
		apiKey := os.Getenv("ANTHROPIC_API_KEY")
		if apiKey == "" {
			return nil, fmt.Errorf("ANTHROPIC_API_KEY environment variable is required for Claude provider")
		}
		config := providers.ClaudeConfig{
			APIKey: apiKey,
			Model:  model,
		}
		return providers.NewClaudeProvider(config)

	case "openai", "gpt":
		apiKey := os.Getenv("OPENAI_API_KEY")
		if apiKey == "" {
			return nil, fmt.Errorf("OPENAI_API_KEY environment variable is required for OpenAI provider")
		}
		config := providers.OpenAIConfig{
			APIKey: apiKey,
			Model:  model,
		}
		return providers.NewOpenAIProvider(config)

	default:
		return nil, fmt.Errorf("unsupported provider: %s (use 'claude' or 'openai')", name)
	}
}

func askQuestion(ctx context.Context, assistant *ai.ContextualAssistant, question string) error {
	fmt.Printf("\n🤔 Asking: %s\n\n", question)

	startTime := time.Now()
	resp, err := assistant.Chat(ctx, question)
	if err != nil {
		return fmt.Errorf("chat failed: %w", err)
	}
	elapsed := time.Since(startTime)

	// Print response
	fmt.Println("📝 Response:")
	fmt.Println(strings.Repeat("-", 60))
	fmt.Println(resp.Content)
	fmt.Println(strings.Repeat("-", 60))

	// Print metadata
	if len(resp.ToolsUsed) > 0 {
		fmt.Printf("\n🔧 Tools used: %s\n", strings.Join(resp.ToolsUsed, ", "))
	}
	fmt.Printf("⏱️  Response time: %s\n", elapsed.Round(time.Millisecond))
	fmt.Printf("📊 Tokens: %d prompt, %d completion, %d total\n",
		resp.Usage.PromptTokens, resp.Usage.CompletionTokens, resp.Usage.TotalTokens)

	return nil
}

func runInteractiveMode(ctx context.Context, assistant *ai.ContextualAssistant) error {
	fmt.Println("\n" + strings.Repeat("=", 60))
	fmt.Println("Platform Foundry AI Assistant")
	fmt.Println(strings.Repeat("=", 60))
	fmt.Println("\nI'm here to help you understand and manage your platform.")
	fmt.Println("Type 'help' for commands, 'exit' to quit.")

	// Show suggested questions
	suggestions := assistant.SuggestedQuestions()
	if len(suggestions) > 0 {
		fmt.Println("💡 Suggested questions:")
		for i, q := range suggestions[:min(3, len(suggestions))] {
			fmt.Printf("   %d. %s\n", i+1, q)
		}
		fmt.Println()
	}

	reader := bufio.NewReader(os.Stdin)

	for {
		fmt.Print("You: ")
		input, err := reader.ReadString('\n')
		if err != nil {
			return fmt.Errorf("read error: %w", err)
		}

		input = strings.TrimSpace(input)
		if input == "" {
			continue
		}

		// Handle commands
		switch strings.ToLower(input) {
		case "exit", "quit", "q":
			fmt.Println("\nGoodbye! 👋")
			return nil

		case "help", "h", "?":
			printInteractiveHelp()
			continue

		case "clear", "reset":
			assistant.ClearConversation()
			fmt.Println("\n🔄 Conversation cleared.")
			continue

		case "history":
			printConversationHistory(assistant)
			continue

		case "summary":
			printConversationSummary(assistant)
			continue

		case "suggestions":
			printSuggestions(assistant)
			continue
		}

		// Handle numbered suggestions
		if len(input) == 1 && input[0] >= '1' && input[0] <= '9' {
			idx := int(input[0] - '1')
			suggestions := assistant.SuggestedQuestions()
			if idx < len(suggestions) {
				input = suggestions[idx]
				fmt.Printf("You: %s\n", input)
			}
		}

		// Ask the question
		fmt.Println()
		startTime := time.Now()
		resp, err := assistant.Chat(ctx, input)
		elapsed := time.Since(startTime)

		if err != nil {
			fmt.Printf("❌ Error: %s\n\n", err)
			continue
		}

		fmt.Println("Assistant:")
		fmt.Println(strings.Repeat("-", 60))
		fmt.Println(resp.Content)
		fmt.Println(strings.Repeat("-", 60))

		if len(resp.ToolsUsed) > 0 {
			fmt.Printf("🔧 [%s | %s]\n\n", strings.Join(resp.ToolsUsed, ", "), elapsed.Round(time.Millisecond))
		} else {
			fmt.Printf("⏱️  [%s]\n\n", elapsed.Round(time.Millisecond))
		}
	}
}

func printInteractiveHelp() {
	fmt.Print(`
Commands:
  help, h, ?     Show this help message
  clear, reset   Clear conversation history
  history        Show conversation history
  summary        Show conversation summary
  suggestions    Show suggested questions
  exit, quit, q  Exit interactive mode
  1-9            Select a suggested question

Tips:
  - Ask about specific services: "What's the health of order-service?"
  - Check for issues: "Are there any active alerts?"
  - Get recommendations: "How can I improve reliability?"
  - Compare environments: "Compare staging to production"
  - Analyze costs: "What are our biggest cost drivers?"
`)
}

func printConversationHistory(assistant *ai.ContextualAssistant) {
	conversation := assistant.GetConversation()
	if len(conversation) == 0 {
		fmt.Println("\n📜 No conversation history.")
		return
	}

	fmt.Println("\n📜 Conversation History:")
	fmt.Println(strings.Repeat("-", 60))

	for _, msg := range conversation {
		role := strings.Title(msg.Role)
		content := msg.Content
		if len(content) > 200 {
			content = content[:200] + "..."
		}
		fmt.Printf("[%s] %s\n", role, content)

		if len(msg.ToolCalls) > 0 {
			tools := make([]string, len(msg.ToolCalls))
			for i, tc := range msg.ToolCalls {
				tools[i] = tc.Name
			}
			fmt.Printf("  🔧 Tools: %s\n", strings.Join(tools, ", "))
		}
	}

	fmt.Println(strings.Repeat("-", 60))
	fmt.Println()
}

func printConversationSummary(assistant *ai.ContextualAssistant) {
	summary := assistant.GetConversationSummary()

	fmt.Println("\n📊 Conversation Summary:")
	fmt.Println(strings.Repeat("-", 40))
	fmt.Printf("Messages:    %d\n", summary.MessageCount)
	fmt.Printf("Tool calls:  %d\n", summary.ToolCallCount)
	if len(summary.ToolsUsed) > 0 {
		fmt.Printf("Tools used:  %s\n", strings.Join(summary.ToolsUsed, ", "))
	}
	fmt.Println(strings.Repeat("-", 40))
	fmt.Println()
}

func printSuggestions(assistant *ai.ContextualAssistant) {
	suggestions := assistant.SuggestedQuestions()
	fmt.Println("\n💡 Suggested questions:")
	for i, q := range suggestions {
		fmt.Printf("   %d. %s\n", i+1, q)
	}
	fmt.Println()
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
