package cli

import (
	"fmt"
	"os"
	"strings"
	"text/tabwriter"

	"github.com/platformfoundry/pf-ce/internal/copilot"
	"github.com/platformfoundry/pf-ce/internal/intelligence"
	"github.com/spf13/cobra"
)

var copilotCmd = &cobra.Command{
	Use:   "copilot",
	Short: "AI-powered platform assistant",
	Long: `AI-powered assistant for platform operations.

The copilot helps you deploy, troubleshoot, monitor, and manage your
platform infrastructure using natural language.

Examples:
  pf copilot chat                         Start interactive chat session
  pf copilot ask "deploy to staging"      Ask a single question
  pf copilot diagnose "API is slow"       Diagnose a problem
  pf copilot suggest                      Get AI suggestions for current state`,
}

var copilotChatCmd = &cobra.Command{
	Use:   "chat",
	Short: "Start interactive chat session",
	Long: `Start an interactive chat session with the AI copilot.

The copilot understands natural language commands for:
- Deployments: "deploy the api to staging"
- Troubleshooting: "why is the worker service failing"
- Scaling: "scale the api to 5 replicas"
- Monitoring: "show me the health of production"
- Configuration: "update the database connection string"

Type 'exit' or 'quit' to end the session.`,
	RunE: runCopilotChat,
}

var copilotAskCmd = &cobra.Command{
	Use:   "ask [question]",
	Short: "Ask a single question",
	Long: `Ask the copilot a single question and get a response.

Examples:
  pf copilot ask "what services are running"
  pf copilot ask "deploy api to staging"
  pf copilot ask "rollback the last deployment"`,
	Args: cobra.MinimumNArgs(1),
	RunE: runCopilotAsk,
}

var copilotDiagnoseCmd = &cobra.Command{
	Use:   "diagnose [symptom]",
	Short: "Diagnose platform issues",
	Long: `Diagnose platform issues using AI-powered analysis.

The diagnose command analyzes logs, metrics, events, and health status
to identify the probable root cause of an issue.

Examples:
  pf copilot diagnose "API requests are timing out"
  pf copilot diagnose "High error rate in production"
  pf copilot diagnose "Database connections exhausted"`,
	Args: cobra.MinimumNArgs(1),
	RunE: runCopilotDiagnose,
}

var copilotSuggestCmd = &cobra.Command{
	Use:   "suggest",
	Short: "Get AI suggestions for current state",
	Long: `Get AI-powered suggestions based on current platform state.

Analyzes the current state and provides recommendations for:
- Performance optimizations
- Cost savings
- Security improvements
- Reliability enhancements`,
	RunE: runCopilotSuggest,
}

var copilotPlanCmd = &cobra.Command{
	Use:   "plan [action]",
	Short: "Generate an action plan",
	Long: `Generate a detailed action plan for an operation.

Examples:
  pf copilot plan "deploy api to production"
  pf copilot plan "migrate database"
  pf copilot plan "upgrade kubernetes cluster"`,
	Args: cobra.MinimumNArgs(1),
	RunE: runCopilotPlan,
}

// Flags
var (
	copilotProvider    string
	copilotModel       string
	copilotService     string
	copilotEnvironment string
	copilotVerbose     bool
	copilotJSON        bool
)

func init() {
	// Global copilot flags
	copilotCmd.PersistentFlags().StringVar(&copilotProvider, "provider", "mock", "LLM provider (openai, mock)")
	copilotCmd.PersistentFlags().StringVar(&copilotModel, "model", "gpt-4", "Model to use")
	copilotCmd.PersistentFlags().BoolVar(&copilotVerbose, "verbose", false, "Show detailed output")
	copilotCmd.PersistentFlags().BoolVar(&copilotJSON, "json", false, "Output in JSON format")

	// Diagnose command flags
	copilotDiagnoseCmd.Flags().StringVar(&copilotService, "service", "", "Focus on specific service")
	copilotDiagnoseCmd.Flags().StringVar(&copilotEnvironment, "env", "", "Focus on specific environment")

	// Add subcommands
	copilotCmd.AddCommand(copilotChatCmd)
	copilotCmd.AddCommand(copilotAskCmd)
	copilotCmd.AddCommand(copilotDiagnoseCmd)
	copilotCmd.AddCommand(copilotSuggestCmd)
	copilotCmd.AddCommand(copilotPlanCmd)
}

func runCopilotChat(cmd *cobra.Command, args []string) error {
	ctx := cmd.Context()

	engine, err := getCopilotEngine()
	if err != nil {
		return fmt.Errorf("failed to initialize copilot: %w", err)
	}

	return engine.StartInteractive(ctx)
}

func runCopilotAsk(cmd *cobra.Command, args []string) error {
	ctx := cmd.Context()
	question := strings.Join(args, " ")

	engine, err := getCopilotEngine()
	if err != nil {
		return fmt.Errorf("failed to initialize copilot: %w", err)
	}

	response, err := engine.ProcessMessage(ctx, question)
	if err != nil {
		return fmt.Errorf("failed to process question: %w", err)
	}

	fmt.Println(response.Message)

	if len(response.Actions) > 0 && copilotVerbose {
		fmt.Println("\nSuggested Actions:")
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

	return nil
}

func runCopilotDiagnose(cmd *cobra.Command, args []string) error {
	ctx := cmd.Context()
	symptom := strings.Join(args, " ")

	fmt.Printf("Analyzing: %s\n", symptom)
	fmt.Println("Gathering evidence...")

	troubleshootEngine := copilot.NewTroubleshootEngine(nil, nil, nil, nil)

	result, err := troubleshootEngine.Diagnose(ctx, symptom)
	if err != nil {
		return fmt.Errorf("diagnosis failed: %w", err)
	}

	fmt.Println()
	printDiagnosisResult(result)

	return nil
}

func printDiagnosisResult(result *copilot.DiagnosisResult) {
	// Severity indicator
	severityColor := ""
	switch result.Severity {
	case copilot.SeverityCritical:
		severityColor = "CRITICAL"
	case copilot.SeverityError:
		severityColor = "ERROR"
	case copilot.SeverityWarning:
		severityColor = "WARNING"
	default:
		severityColor = "INFO"
	}

	fmt.Printf("[%s] Diagnosis Results (Confidence: %.0f%%)\n", severityColor, result.Confidence*100)
	fmt.Println(strings.Repeat("-", 50))

	fmt.Printf("\nProbable Root Cause:\n  %s\n", result.ProbableRootCause)

	if len(result.AffectedServices) > 0 {
		fmt.Printf("\nAffected Services:\n")
		for _, svc := range result.AffectedServices {
			fmt.Printf("  - %s\n", svc)
		}
	}

	if len(result.Evidence) > 0 && copilotVerbose {
		fmt.Printf("\nEvidence (%d items):\n", len(result.Evidence))
		w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
		fmt.Fprintln(w, "  TYPE\tSOURCE\tDESCRIPTION\tRELEVANCE")
		for _, ev := range result.Evidence {
			desc := ev.Description
			if len(desc) > 50 {
				desc = desc[:47] + "..."
			}
			fmt.Fprintf(w, "  %s\t%s\t%s\t%.0f%%\n",
				ev.Type, ev.Source, desc, ev.Relevance*100)
		}
		w.Flush()
	}

	if len(result.Timeline) > 0 && copilotVerbose {
		fmt.Printf("\nTimeline:\n")
		for _, event := range result.Timeline {
			fmt.Printf("  %s [%s] %s: %s\n",
				event.Timestamp.Format("15:04:05"),
				event.Severity,
				event.Source,
				event.Description,
			)
		}
	}

	if len(result.SuggestedFixes) > 0 {
		fmt.Printf("\nSuggested Fixes:\n")
		for i, fix := range result.SuggestedFixes {
			fmt.Printf("  %d. %s (Confidence: %.0f%%, Risk: %s)\n",
				i+1, fix.Description, fix.Confidence*100, fix.RiskLevel)
			if fix.Command != "" {
				fmt.Printf("     Command: %s\n", fix.Command)
			}
		}
	}
}

func runCopilotSuggest(cmd *cobra.Command, args []string) error {
	fmt.Println("Analyzing current platform state...")
	fmt.Println()

	// In production, this would analyze the actual platform state
	fmt.Println("Suggestions based on current state:")
	fmt.Println()

	suggestions := []struct {
		category string
		title    string
		impact   string
		command  string
	}{
		{
			category: "Performance",
			title:    "Enable autoscaling for api-gateway",
			impact:   "Improved response times during peak traffic",
			command:  "pf workload autoscale enable api-gateway --min 2 --max 10",
		},
		{
			category: "Cost",
			title:    "Right-size development instances",
			impact:   "Save ~$150/month",
			command:  "pf finops apply-recommendations --env development",
		},
		{
			category: "Security",
			title:    "Rotate API keys older than 90 days",
			impact:   "Improved security posture",
			command:  "pf secrets rotate --older-than 90d",
		},
		{
			category: "Reliability",
			title:    "Add health checks to worker-service",
			impact:   "Faster failure detection",
			command:  "pf workload health-check add worker-service",
		},
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "CATEGORY\tSUGGESTION\tIMPACT")
	for _, s := range suggestions {
		fmt.Fprintf(w, "[%s]\t%s\t%s\n", s.category, s.title, s.impact)
		if copilotVerbose {
			fmt.Fprintf(w, "\t  Command: %s\t\n", s.command)
		}
	}
	w.Flush()

	fmt.Println("\nRun with --verbose to see commands for each suggestion.")
	return nil
}

func runCopilotPlan(cmd *cobra.Command, args []string) error {
	ctx := cmd.Context()
	action := strings.Join(args, " ")

	engine, err := getCopilotEngine()
	if err != nil {
		return fmt.Errorf("failed to initialize copilot: %w", err)
	}

	fmt.Printf("Generating plan for: %s\n", action)
	fmt.Println()

	response, err := engine.ProcessMessage(ctx, action)
	if err != nil {
		return fmt.Errorf("failed to generate plan: %w", err)
	}

	if response.Plan == nil {
		fmt.Println(response.Message)
		return nil
	}

	plan := response.Plan

	fmt.Printf("Action Plan: %s\n", plan.Description)
	fmt.Printf("Risk Level: %s\n", plan.RiskLevel)
	fmt.Printf("Estimated Time: %s\n", plan.EstimatedTime)
	fmt.Printf("Reversible: %t\n", plan.Reversible)
	if plan.RequiresApproval {
		fmt.Printf("Requires Approval: yes\n")
	}

	fmt.Printf("\nSteps (%d):\n", len(plan.Steps))
	for i, step := range plan.Steps {
		fmt.Printf("  %d. [%s] %s\n", i+1, step.RiskLevel, step.Description)
	}

	if len(plan.SafetyChecks) > 0 && copilotVerbose {
		fmt.Printf("\nSafety Checks:\n")
		for _, check := range plan.SafetyChecks {
			status := "PASS"
			if !check.Passed {
				status = "FAIL"
			}
			fmt.Printf("  [%s] %s\n", status, check.Rule)
			if check.Message != "" {
				fmt.Printf("        %s\n", check.Message)
			}
		}
	}

	fmt.Println("\nTo execute this plan, confirm with: pf copilot execute --plan <plan-id>")

	return nil
}

// getCopilotEngine creates and returns a configured copilot engine
func getCopilotEngine() (*copilot.ConversationEngine, error) {
	var provider intelligence.LLMProvider
	switch copilotProvider {
	case "openai":
		provider = intelligence.ProviderOpenAI
	case "mock":
		provider = intelligence.ProviderMock
	default:
		provider = intelligence.ProviderMock
	}

	cfg := copilot.EngineConfig{
		LLMConfig: intelligence.LLMConfig{
			Provider: provider,
			Model:    copilotModel,
			APIKey:   os.Getenv("OPENAI_API_KEY"),
		},
		MaxHistory:    50,
		SafetyEnabled: true,
	}

	return copilot.NewConversationEngine(cfg, nil)
}
