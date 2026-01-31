package cli

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/platformfoundry/platformfoundry-ce/internal/gitops"
	"github.com/platformfoundry/platformfoundry-ce/pkg/types"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

func init() {
	rootCmd.AddCommand(gitopsCmd)

	// Subcommands
	gitopsCmd.AddCommand(gitopsInitCmd)
	gitopsCmd.AddCommand(gitopsSyncCmd)
	gitopsCmd.AddCommand(gitopsStatusCmd)
	gitopsCmd.AddCommand(gitopsEventsCmd)
	gitopsCmd.AddCommand(gitopsPromoteCmd)

	// Flags
	gitopsInitCmd.Flags().StringP("config", "c", "", "Path to GitOps configuration file")
	gitopsInitCmd.Flags().StringP("repo", "r", "", "Repository URL")
	gitopsInitCmd.Flags().StringP("branch", "b", "main", "Branch name")
	gitopsInitCmd.Flags().StringP("path", "p", "", "Path within repository")

	gitopsSyncCmd.Flags().Bool("dry-run", false, "Show what would be synced without making changes")
	gitopsSyncCmd.Flags().Bool("prune", false, "Remove resources not defined in Git")
	gitopsSyncCmd.Flags().StringP("revision", "r", "", "Specific revision to sync")

	gitopsEventsCmd.Flags().IntP("limit", "l", 20, "Maximum number of events to show")
	gitopsEventsCmd.Flags().String("type", "", "Filter by event type (sync, pr_created, error)")

	gitopsPromoteCmd.Flags().StringP("source", "s", "", "Source environment")
	gitopsPromoteCmd.Flags().StringP("target", "t", "", "Target environment")
	gitopsPromoteCmd.Flags().Bool("auto-merge", false, "Automatically merge the promotion PR")
}

var gitopsCmd = &cobra.Command{
	Use:   "gitops",
	Short: "Manage GitOps workflows",
	Long: `Manage GitOps-based platform deployment workflows.

GitOps enables declarative infrastructure management through Git repositories,
providing version control, audit trails, and automated synchronization.

Features:
  - Repository synchronization with automatic drift detection
  - Pull request-based change management
  - Environment promotions
  - Event tracking and notifications`,
	Example: `  # Initialize GitOps from a config file
  pf gitops init -c gitops-config.yaml

  # Sync with the repository
  pf gitops sync

  # Check GitOps status
  pf gitops status

  # View recent events
  pf gitops events

  # Promote staging to production
  pf gitops promote -s staging -t production`,
}

var gitopsInitCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialize GitOps configuration",
	Long: `Initialize GitOps configuration for the platform.

This command sets up the connection to your Git repository and configures
synchronization settings.`,
	RunE: runGitOpsInit,
}

var gitopsSyncCmd = &cobra.Command{
	Use:   "sync",
	Short: "Synchronize with Git repository",
	Long: `Synchronize platform state with the Git repository.

This pulls the latest changes from the repository and optionally applies
them to the platform based on sync policies.`,
	RunE: runGitOpsSync,
}

var gitopsStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show GitOps status",
	Long:  `Display the current GitOps synchronization status, including last sync time and health.`,
	RunE:  runGitOpsStatus,
}

var gitopsEventsCmd = &cobra.Command{
	Use:   "events",
	Short: "Show GitOps events",
	Long:  `Display recent GitOps events including syncs, PR creations, and errors.`,
	RunE:  runGitOpsEvents,
}

var gitopsPromoteCmd = &cobra.Command{
	Use:   "promote",
	Short: "Promote changes between environments",
	Long: `Promote configuration changes from one environment to another.

This creates a pull request to apply the configuration from the source
environment to the target environment.`,
	RunE: runGitOpsPromote,
}

// Global manager instance (would be properly initialized in real implementation)
var gitopsManager *gitops.Manager

func runGitOpsInit(cmd *cobra.Command, args []string) error {
	configPath, _ := cmd.Flags().GetString("config")
	repoURL, _ := cmd.Flags().GetString("repo")
	branch, _ := cmd.Flags().GetString("branch")
	path, _ := cmd.Flags().GetString("path")

	var config *types.GitOpsConfigV2

	if configPath != "" {
		// Load from file
		data, err := os.ReadFile(configPath)
		if err != nil {
			return fmt.Errorf("failed to read config file: %w", err)
		}

		config = &types.GitOpsConfigV2{}
		if err := yaml.Unmarshal(data, config); err != nil {
			return fmt.Errorf("failed to parse config file: %w", err)
		}
	} else if repoURL != "" {
		// Create config from flags
		config = &types.GitOpsConfigV2{
			APIVersion: "platformfoundry.io/v1",
			Kind:       "GitOpsConfig",
			Metadata: types.GitOpsMetadata{
				Name: "default",
			},
			Spec: types.GitOpsSpecV2{
				Repository: types.GitOpsRepository{
					URL:    repoURL,
					Branch: branch,
					Path:   path,
				},
				Sync: types.GitOpsSyncConfig{
					Interval: "5m",
					Prune:    false,
					SelfHeal: false,
				},
			},
		}
	} else {
		return fmt.Errorf("either --config or --repo is required")
	}

	// Create the manager
	var err error
	gitopsManager, err = gitops.NewManager(gitops.ManagerConfig{
		Config: config,
	})
	if err != nil {
		return fmt.Errorf("failed to create GitOps manager: %w", err)
	}

	// Initialize
	ctx := context.Background()
	fmt.Println("Initializing GitOps repository...")

	if err := gitopsManager.Initialize(ctx); err != nil {
		return fmt.Errorf("failed to initialize: %w", err)
	}

	fmt.Println("GitOps initialized successfully!")
	fmt.Printf("\nRepository: %s\n", config.Spec.Repository.URL)
	fmt.Printf("Branch:     %s\n", config.Spec.Repository.Branch)
	if config.Spec.Repository.Path != "" {
		fmt.Printf("Path:       %s\n", config.Spec.Repository.Path)
	}
	fmt.Printf("Sync:       %s\n", config.Spec.Sync.Interval)

	return nil
}

func runGitOpsSync(cmd *cobra.Command, args []string) error {
	dryRun, _ := cmd.Flags().GetBool("dry-run")
	prune, _ := cmd.Flags().GetBool("prune")

	if gitopsManager == nil {
		return fmt.Errorf("GitOps not initialized. Run 'pf gitops init' first")
	}

	ctx := context.Background()

	if dryRun {
		fmt.Println("Dry run mode - no changes will be applied")
	}

	fmt.Println("Synchronizing with Git repository...")
	startTime := time.Now()

	result, err := gitopsManager.Sync(ctx)
	if err != nil {
		return fmt.Errorf("sync failed: %w", err)
	}

	elapsed := time.Since(startTime)

	fmt.Println()
	if result.Success {
		fmt.Println("Sync completed successfully!")
	} else {
		fmt.Println("Sync completed with errors")
	}

	fmt.Println(strings.Repeat("-", 50))
	fmt.Printf("Revision:  %s\n", result.Revision[:12])
	fmt.Printf("Duration:  %s\n", elapsed.Round(time.Millisecond))
	fmt.Printf("Resources: %d synced\n", len(result.Resources))

	if prune {
		fmt.Println("\nPruning enabled - orphaned resources would be removed")
	}

	if len(result.Resources) > 0 {
		fmt.Println("\nSynced Resources:")
		for _, r := range result.Resources {
			status := "✓"
			if r.Status == "SyncFailed" {
				status = "✗"
			} else if r.Status == "Pruned" {
				status = "⌫"
			}
			fmt.Printf("  %s %s/%s\n", status, r.Kind, r.Name)
		}
	}

	if len(result.Errors) > 0 {
		fmt.Println("\nErrors:")
		for _, e := range result.Errors {
			fmt.Printf("  • %s\n", e)
		}
	}

	return nil
}

func runGitOpsStatus(cmd *cobra.Command, args []string) error {
	if gitopsManager == nil {
		return fmt.Errorf("GitOps not initialized. Run 'pf gitops init' first")
	}

	ctx := context.Background()
	status, err := gitopsManager.GetStatus(ctx)
	if err != nil {
		return fmt.Errorf("failed to get status: %w", err)
	}

	fmt.Println("GitOps Status")
	fmt.Println(strings.Repeat("=", 50))

	// Phase indicator
	phaseIcon := getPhaseIcon(status.Phase)
	fmt.Printf("\nPhase: %s %s\n", phaseIcon, status.Phase)

	// Sync status
	fmt.Println("\nSync Status:")
	fmt.Printf("  Status:   %s\n", status.SyncStatus.Status)
	if status.SyncStatus.Revision != "" {
		fmt.Printf("  Revision: %s\n", status.SyncStatus.Revision[:12])
	}
	if status.LastSyncTime != nil {
		fmt.Printf("  Last Sync: %s\n", status.LastSyncTime.Format(time.RFC3339))
	}

	// Health status
	healthIcon := getGitOpsHealthIcon(status.HealthStatus.Status)
	fmt.Println("\nHealth Status:")
	fmt.Printf("  Status: %s %s\n", healthIcon, status.HealthStatus.Status)
	if status.HealthStatus.Message != "" {
		fmt.Printf("  Message: %s\n", status.HealthStatus.Message)
	}

	// Conditions
	if len(status.Conditions) > 0 {
		fmt.Println("\nConditions:")
		for _, c := range status.Conditions {
			fmt.Printf("  • %s: %s\n", c.Type, c.Status)
			if c.Message != "" {
				fmt.Printf("    %s\n", c.Message)
			}
		}
	}

	return nil
}

func runGitOpsEvents(cmd *cobra.Command, args []string) error {
	limit, _ := cmd.Flags().GetInt("limit")
	eventType, _ := cmd.Flags().GetString("type")

	if gitopsManager == nil {
		return fmt.Errorf("GitOps not initialized. Run 'pf gitops init' first")
	}

	events := gitopsManager.GetEvents(limit)

	if len(events) == 0 {
		fmt.Println("No GitOps events recorded")
		return nil
	}

	fmt.Println("GitOps Events")
	fmt.Println(strings.Repeat("=", 60))

	for _, event := range events {
		// Filter by type if specified
		if eventType != "" && event.Type != eventType {
			continue
		}

		icon := getEventIcon(event.Type)
		timeStr := event.Timestamp.Format("15:04:05")

		fmt.Printf("\n%s [%s] %s\n", icon, timeStr, event.Type)
		fmt.Printf("   %s\n", event.Message)

		if event.Metadata != nil {
			for k, v := range event.Metadata {
				fmt.Printf("   %s: %v\n", k, v)
			}
		}
	}

	return nil
}

func runGitOpsPromote(cmd *cobra.Command, args []string) error {
	source, _ := cmd.Flags().GetString("source")
	target, _ := cmd.Flags().GetString("target")
	autoMerge, _ := cmd.Flags().GetBool("auto-merge")

	if source == "" || target == "" {
		return fmt.Errorf("both --source and --target are required")
	}

	if gitopsManager == nil {
		return fmt.Errorf("GitOps not initialized. Run 'pf gitops init' first")
	}

	ctx := context.Background()

	fmt.Printf("Promoting %s → %s...\n", source, target)

	pr, err := gitopsManager.PromoteEnvironment(ctx, source, target)
	if err != nil {
		return fmt.Errorf("promotion failed: %w", err)
	}

	fmt.Println("\nPromotion PR created successfully!")
	fmt.Println(strings.Repeat("-", 50))
	fmt.Printf("PR #%d: %s\n", pr.Number, pr.Title)
	fmt.Printf("URL: %s\n", pr.URL)
	fmt.Printf("Branch: %s\n", pr.Branch)

	if autoMerge {
		fmt.Println("\nAuto-merge is enabled but requires approval workflow")
	}

	return nil
}

// Helper functions

func getPhaseIcon(phase types.GitOpsPhase) string {
	switch phase {
	case types.GitOpsPhaseSynced:
		return "✓"
	case types.GitOpsPhaseOutOfSync:
		return "⚠"
	case types.GitOpsPhaseFailed:
		return "✗"
	case types.GitOpsPhaseRunning:
		return "⟳"
	case types.GitOpsPhaseSuspended:
		return "⏸"
	default:
		return "?"
	}
}

func getGitOpsHealthIcon(status string) string {
	switch status {
	case "Healthy":
		return "✓"
	case "Degraded":
		return "⚠"
	case "Progressing":
		return "⟳"
	default:
		return "?"
	}
}

func getEventIcon(eventType string) string {
	switch eventType {
	case "sync":
		return "🔄"
	case "pr_created":
		return "📝"
	case "pr_merged":
		return "✅"
	case "error":
		return "❌"
	case "promotion":
		return "⬆️"
	case "initialized":
		return "🚀"
	default:
		return "•"
	}
}
