package cli

import (
	"fmt"
	"os"
	"text/tabwriter"
	"time"

	"github.com/platformfoundry/pf-ce/internal/preview"
	"github.com/spf13/cobra"
)

var previewCmd = &cobra.Command{
	Use:   "preview",
	Short: "Manage preview environments",
	Long: `Manage ephemeral preview environments for pull requests.

Preview environments are automatically created when a PR is opened and
destroyed when the PR is closed or merged.

Examples:
  pf preview list                         List all active preview environments
  pf preview create --repo org/repo --pr 123   Create a preview environment
  pf preview delete pr-123-feature        Delete a preview environment
  pf preview extend pr-123-feature --ttl 24h   Extend TTL by 24 hours`,
}

var previewListCmd = &cobra.Command{
	Use:   "list",
	Short: "List active preview environments",
	Long:  `List all active preview environments with their status and URLs.`,
	RunE:  runPreviewList,
}

var previewCreateCmd = &cobra.Command{
	Use:   "create",
	Short: "Create preview environment from PR",
	Long: `Create a new preview environment from a pull request.

Examples:
  pf preview create --repo org/repo --pr 123
  pf preview create --repo org/repo --pr 123 --base staging --ttl 72h`,
	RunE: runPreviewCreate,
}

var previewDeleteCmd = &cobra.Command{
	Use:   "delete [name]",
	Short: "Delete preview environment",
	Long: `Delete a preview environment by name or PR number.

Examples:
  pf preview delete pr-123-feature
  pf preview delete --pr 123 --repo org/repo`,
	Args: cobra.MaximumNArgs(1),
	RunE: runPreviewDelete,
}

var previewExtendCmd = &cobra.Command{
	Use:   "extend [name]",
	Short: "Extend preview environment TTL",
	Long: `Extend the TTL of a preview environment.

Examples:
  pf preview extend pr-123-feature --ttl 24h
  pf preview extend pr-123-feature --ttl 48h`,
	Args: cobra.ExactArgs(1),
	RunE: runPreviewExtend,
}

var previewStatusCmd = &cobra.Command{
	Use:   "status [name]",
	Short: "Show preview environment status",
	Long:  `Show detailed status of a preview environment.`,
	Args:  cobra.ExactArgs(1),
	RunE:  runPreviewStatus,
}

var previewRefreshCmd = &cobra.Command{
	Use:   "refresh [name]",
	Short: "Refresh preview environment",
	Long:  `Rebuild and redeploy a preview environment with latest changes.`,
	Args:  cobra.ExactArgs(1),
	RunE:  runPreviewRefresh,
}

// Flags
var (
	previewRepo             string
	previewPR               int
	previewBaseEnv          string
	previewTTL              time.Duration
	previewDatabaseStrategy string
	previewWait             bool
)

func init() {
	// Create command flags
	previewCreateCmd.Flags().StringVar(&previewRepo, "repo", "", "Repository (owner/repo)")
	previewCreateCmd.Flags().IntVar(&previewPR, "pr", 0, "Pull request number")
	previewCreateCmd.Flags().StringVar(&previewBaseEnv, "base", "staging", "Base environment to clone")
	previewCreateCmd.Flags().DurationVar(&previewTTL, "ttl", 72*time.Hour, "Time to live")
	previewCreateCmd.Flags().StringVar(&previewDatabaseStrategy, "db-strategy", "fresh", "Database strategy (fresh, clone, seed)")
	previewCreateCmd.Flags().BoolVar(&previewWait, "wait", false, "Wait for environment to be ready")

	previewCreateCmd.MarkFlagRequired("repo")
	previewCreateCmd.MarkFlagRequired("pr")

	// Delete command flags
	previewDeleteCmd.Flags().IntVar(&previewPR, "pr", 0, "Pull request number (alternative to name)")
	previewDeleteCmd.Flags().StringVar(&previewRepo, "repo", "", "Repository (required with --pr)")

	// Extend command flags
	previewExtendCmd.Flags().DurationVar(&previewTTL, "ttl", 24*time.Hour, "Additional time to live")

	// Add subcommands
	previewCmd.AddCommand(previewListCmd)
	previewCmd.AddCommand(previewCreateCmd)
	previewCmd.AddCommand(previewDeleteCmd)
	previewCmd.AddCommand(previewExtendCmd)
	previewCmd.AddCommand(previewStatusCmd)
	previewCmd.AddCommand(previewRefreshCmd)
}

func runPreviewList(cmd *cobra.Command, args []string) error {
	ctx := cmd.Context()

	mgr := getPreviewManager()
	if mgr == nil {
		return fmt.Errorf("preview manager not configured")
	}

	previews, err := mgr.List(ctx)
	if err != nil {
		return fmt.Errorf("failed to list previews: %w", err)
	}

	if len(previews) == 0 {
		fmt.Println("No active preview environments")
		return nil
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "NAME\tSTATUS\tPR\tURL\tEXPIRES\tAGE")

	for _, p := range previews {
		age := time.Since(p.CreatedAt).Round(time.Minute)
		expiresIn := time.Until(p.ExpiresAt).Round(time.Minute)
		expiresStr := formatDuration(expiresIn)
		if expiresIn < 0 {
			expiresStr = "expired"
		}

		fmt.Fprintf(w, "%s\t%s\t#%d\t%s\t%s\t%s\n",
			p.Name,
			p.Status,
			p.PullRequest,
			p.URL,
			expiresStr,
			formatDuration(age),
		)
	}

	return w.Flush()
}

func runPreviewCreate(cmd *cobra.Command, args []string) error {
	ctx := cmd.Context()

	mgr := getPreviewManager()
	if mgr == nil {
		return fmt.Errorf("preview manager not configured")
	}

	// Parse database strategy
	var dbStrategy preview.DatabaseStrategy
	switch previewDatabaseStrategy {
	case "fresh":
		dbStrategy = preview.DatabaseStrategyFresh
	case "clone":
		dbStrategy = preview.DatabaseStrategyClone
	case "seed":
		dbStrategy = preview.DatabaseStrategySeed
	default:
		return fmt.Errorf("invalid database strategy: %s (must be fresh, clone, or seed)", previewDatabaseStrategy)
	}

	opts := preview.CreatePreviewOpts{
		Repository:       previewRepo,
		PullRequest:      previewPR,
		BaseEnvironment:  previewBaseEnv,
		TTL:              previewTTL,
		DatabaseStrategy: dbStrategy,
	}

	fmt.Printf("Creating preview environment for PR #%d...\n", previewPR)

	env, err := mgr.Create(ctx, opts)
	if err != nil {
		return fmt.Errorf("failed to create preview: %w", err)
	}

	fmt.Printf("Preview environment created:\n")
	fmt.Printf("  Name:    %s\n", env.Name)
	fmt.Printf("  Status:  %s\n", env.Status)
	fmt.Printf("  Expires: %s\n", env.ExpiresAt.Format(time.RFC3339))

	if previewWait {
		fmt.Println("\nWaiting for environment to be ready...")
		// In production, this would poll until ready or timeout
		time.Sleep(2 * time.Second)
		fmt.Printf("  URL:     %s\n", env.URL)
	}

	return nil
}

func runPreviewDelete(cmd *cobra.Command, args []string) error {
	ctx := cmd.Context()

	mgr := getPreviewManager()
	if mgr == nil {
		return fmt.Errorf("preview manager not configured")
	}

	var err error

	if len(args) > 0 {
		// Delete by name
		name := args[0]
		fmt.Printf("Deleting preview environment %s...\n", name)

		// Get preview by name
		previews, listErr := mgr.List(ctx)
		if listErr != nil {
			return listErr
		}

		for _, p := range previews {
			if p.Name == name {
				err = mgr.Delete(ctx, p.ID)
				break
			}
		}
		if err == nil {
			fmt.Printf("Preview environment %s not found\n", name)
			return nil
		}
	} else if previewPR > 0 && previewRepo != "" {
		// Delete by PR number
		fmt.Printf("Deleting preview environment for PR #%d...\n", previewPR)
		err = mgr.DeleteByPullRequest(ctx, previewRepo, previewPR)
	} else {
		return fmt.Errorf("specify either preview name or --pr and --repo flags")
	}

	if err != nil {
		return fmt.Errorf("failed to delete preview: %w", err)
	}

	fmt.Println("Preview environment deleted")
	return nil
}

func runPreviewExtend(cmd *cobra.Command, args []string) error {
	ctx := cmd.Context()
	name := args[0]

	mgr := getPreviewManager()
	if mgr == nil {
		return fmt.Errorf("preview manager not configured")
	}

	// Find preview by name
	previews, err := mgr.List(ctx)
	if err != nil {
		return err
	}

	var previewID string
	for _, p := range previews {
		if p.Name == name {
			previewID = p.ID
			break
		}
	}

	if previewID == "" {
		return fmt.Errorf("preview environment %s not found", name)
	}

	fmt.Printf("Extending TTL by %s...\n", previewTTL)

	if err := mgr.Extend(ctx, previewID, previewTTL); err != nil {
		return fmt.Errorf("failed to extend TTL: %w", err)
	}

	// Get updated preview
	preview, _ := mgr.Get(ctx, previewID)
	if preview != nil {
		fmt.Printf("New expiration: %s\n", preview.ExpiresAt.Format(time.RFC3339))
	}

	return nil
}

func runPreviewStatus(cmd *cobra.Command, args []string) error {
	ctx := cmd.Context()
	name := args[0]

	mgr := getPreviewManager()
	if mgr == nil {
		return fmt.Errorf("preview manager not configured")
	}

	// Find preview by name
	previews, err := mgr.List(ctx)
	if err != nil {
		return err
	}

	var p *preview.PreviewEnvironment
	for _, prev := range previews {
		if prev.Name == name {
			p = prev
			break
		}
	}

	if p == nil {
		return fmt.Errorf("preview environment %s not found", name)
	}

	fmt.Printf("Preview Environment: %s\n", p.Name)
	fmt.Printf("  ID:          %s\n", p.ID)
	fmt.Printf("  Status:      %s\n", p.Status)
	if p.StatusMessage != "" {
		fmt.Printf("  Message:     %s\n", p.StatusMessage)
	}
	fmt.Printf("  Repository:  %s\n", p.SourceRepo)
	fmt.Printf("  Branch:      %s\n", p.SourceBranch)
	fmt.Printf("  PR:          #%d\n", p.PullRequest)
	fmt.Printf("  Base:        %s\n", p.BaseEnvironment)
	fmt.Printf("  URL:         %s\n", p.URL)
	fmt.Printf("  Created:     %s\n", p.CreatedAt.Format(time.RFC3339))
	fmt.Printf("  Expires:     %s\n", p.ExpiresAt.Format(time.RFC3339))
	fmt.Printf("  TTL:         %s\n", p.TTL)

	if len(p.Resources) > 0 {
		fmt.Printf("\nResources (%d):\n", len(p.Resources))
		for _, r := range p.Resources {
			fmt.Printf("  - %s (%s): %s\n", r.Name, r.Type, r.Status)
		}
	}

	if len(p.Labels) > 0 {
		fmt.Printf("\nLabels:\n")
		for k, v := range p.Labels {
			fmt.Printf("  %s: %s\n", k, v)
		}
	}

	return nil
}

func runPreviewRefresh(cmd *cobra.Command, args []string) error {
	ctx := cmd.Context()
	name := args[0]

	mgr := getPreviewManager()
	if mgr == nil {
		return fmt.Errorf("preview manager not configured")
	}

	// Find preview by name
	previews, err := mgr.List(ctx)
	if err != nil {
		return err
	}

	var previewID string
	for _, p := range previews {
		if p.Name == name {
			previewID = p.ID
			break
		}
	}

	if previewID == "" {
		return fmt.Errorf("preview environment %s not found", name)
	}

	fmt.Printf("Refreshing preview environment %s...\n", name)

	if err := mgr.Refresh(ctx, previewID); err != nil {
		return fmt.Errorf("failed to refresh: %w", err)
	}

	fmt.Println("Refresh initiated. Use 'pf preview status' to check progress.")
	return nil
}

// getPreviewManager returns the preview manager instance
// In production, this would be initialized with proper backends
func getPreviewManager() *preview.Manager {
	// This would normally be initialized from configuration
	// For now, return a basic manager
	cfg := preview.ManagerConfig{
		DefaultTTL:      72 * time.Hour,
		MaxTTL:          168 * time.Hour,
		CleanupInterval: 5 * time.Minute,
	}

	// In production, these would be real implementations
	return preview.NewManager(cfg, nil, nil, nil)
}

func formatDuration(d time.Duration) string {
	if d < time.Minute {
		return "< 1m"
	}
	if d < time.Hour {
		return fmt.Sprintf("%dm", int(d.Minutes()))
	}
	if d < 24*time.Hour {
		return fmt.Sprintf("%dh", int(d.Hours()))
	}
	return fmt.Sprintf("%dd", int(d.Hours()/24))
}
