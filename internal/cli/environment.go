package cli

import (
	"fmt"
	"strings"
	"time"

	"github.com/platformfoundry/pf-ce/internal/environment"
	"github.com/spf13/cobra"
)

var ephemeralManager *environment.EphemeralManager

func init() {
	ephemeralManager = environment.NewEphemeralManager(environment.EphemeralManagerConfig{
		DefaultTTL: 7 * 24 * time.Hour,
		MaxTTL:     14 * 24 * time.Hour,
		BaseURL:    "preview.example.com",
	})
}

var envCmd = &cobra.Command{
	Use:     "env",
	Aliases: []string{"environment"},
	Short:   "Manage environments including ephemeral preview environments",
	Long:    `Create and manage environments, including ephemeral preview environments for pull requests.`,
}

var envListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all ephemeral environments",
	RunE:  runEnvList,
}

var envCreateCmd = &cobra.Command{
	Use:   "create",
	Short: "Create an ephemeral environment",
	RunE:  runEnvCreate,
}

var envStatusCmd = &cobra.Command{
	Use:   "status <env-id>",
	Short: "Get status of an ephemeral environment",
	Args:  cobra.ExactArgs(1),
	RunE:  runEnvStatus,
}

var envDeleteCmd = &cobra.Command{
	Use:   "delete <env-id>",
	Short: "Delete an ephemeral environment",
	Args:  cobra.ExactArgs(1),
	RunE:  runEnvDelete,
}

var envExtendCmd = &cobra.Command{
	Use:   "extend <env-id>",
	Short: "Extend the TTL of an ephemeral environment",
	Args:  cobra.ExactArgs(1),
	RunE:  runEnvExtend,
}

var (
	envOrg          string
	envStatusFilter string
	envBranch       string
	envRepo         string
	envPRNumber     int
	envTTL          time.Duration
	envExtendBy     time.Duration
)

func init() {
	envCmd.AddCommand(envListCmd)
	envCmd.AddCommand(envCreateCmd)
	envCmd.AddCommand(envStatusCmd)
	envCmd.AddCommand(envDeleteCmd)
	envCmd.AddCommand(envExtendCmd)

	// List flags
	envListCmd.Flags().StringVar(&envOrg, "org", "", "Filter by organization")
	envListCmd.Flags().StringVar(&envStatusFilter, "status", "", "Filter by status")

	// Create flags
	envCreateCmd.Flags().StringVar(&envOrg, "org", "", "Organization (required)")
	envCreateCmd.Flags().StringVar(&envRepo, "repo", "", "Repository name (required)")
	envCreateCmd.Flags().StringVar(&envBranch, "branch", "", "Branch name")
	envCreateCmd.Flags().IntVar(&envPRNumber, "pr", 0, "Pull request number")
	envCreateCmd.Flags().DurationVar(&envTTL, "ttl", 7*24*time.Hour, "Time to live")
	envCreateCmd.MarkFlagRequired("org")
	envCreateCmd.MarkFlagRequired("repo")

	// Extend flags
	envExtendCmd.Flags().DurationVar(&envExtendBy, "by", 24*time.Hour, "Extend TTL by this duration")
}

func runEnvList(cmd *cobra.Command, args []string) error {
	var status environment.EphemeralEnvironmentStatus
	if envStatusFilter != "" {
		status = environment.EphemeralEnvironmentStatus(envStatusFilter)
	}

	environments := ephemeralManager.List(envOrg, status)

	if len(environments) == 0 {
		fmt.Println("No ephemeral environments found.")
		return nil
	}

	fmt.Printf("%-12s %-20s %-15s %-12s %-25s %-20s\n",
		"ID", "NAME", "STATUS", "SOURCE", "PREVIEW URL", "EXPIRES")
	fmt.Println(strings.Repeat("-", 110))

	for _, env := range environments {
		shortID := env.ID
		if len(shortID) > 12 {
			shortID = shortID[:12]
		}

		source := env.Source.Type
		if env.Source.PRNumber > 0 {
			source = fmt.Sprintf("PR #%d", env.Source.PRNumber)
		} else if env.Source.Branch != "" {
			branch := env.Source.Branch
			if len(branch) > 10 {
				branch = branch[:10] + "..."
			}
			source = branch
		}

		expiresIn := time.Until(env.ExpiresAt).Round(time.Hour)
		expiresStr := fmt.Sprintf("%s", expiresIn)
		if expiresIn < 0 {
			expiresStr = "expired"
		}

		previewURL := env.PreviewURL
		if len(previewURL) > 25 {
			previewURL = previewURL[:22] + "..."
		}

		fmt.Printf("%-12s %-20s %-15s %-12s %-25s %-20s\n",
			shortID,
			env.Name,
			env.Status,
			source,
			previewURL,
			expiresStr,
		)
	}

	return nil
}

func runEnvCreate(cmd *cobra.Command, args []string) error {
	ctx := cmd.Context()

	if envPRNumber > 0 {
		// Create PR environment
		req := environment.PREnvironmentRequest{
			Organization: envOrg,
			Repository:   envRepo,
			PRNumber:     envPRNumber,
			Branch:       envBranch,
			TTL:          envTTL,
		}

		env, err := ephemeralManager.CreateForPullRequest(ctx, req)
		if err != nil {
			return err
		}

		fmt.Printf("Created ephemeral environment for PR #%d\n", envPRNumber)
		fmt.Printf("  ID: %s\n", env.ID)
		fmt.Printf("  Preview URL: %s\n", env.PreviewURL)
		fmt.Printf("  Namespace: %s\n", env.Namespace)
		fmt.Printf("  Expires: %s\n", env.ExpiresAt.Format(time.RFC3339))

	} else if envBranch != "" {
		// Create branch environment
		env, err := ephemeralManager.CreateForBranch(ctx, envOrg, envRepo, envBranch, "", envTTL)
		if err != nil {
			return err
		}

		fmt.Printf("Created ephemeral environment for branch %s\n", envBranch)
		fmt.Printf("  ID: %s\n", env.ID)
		fmt.Printf("  Preview URL: %s\n", env.PreviewURL)
		fmt.Printf("  Namespace: %s\n", env.Namespace)
		fmt.Printf("  Expires: %s\n", env.ExpiresAt.Format(time.RFC3339))

	} else {
		return fmt.Errorf("either --pr or --branch must be specified")
	}

	return nil
}

func runEnvStatus(cmd *cobra.Command, args []string) error {
	envID := args[0]

	env, err := ephemeralManager.Get(envID)
	if err != nil {
		return err
	}

	fmt.Printf("Environment: %s (%s)\n", env.Name, env.ID)
	fmt.Printf("Organization: %s\n", env.Organization)
	fmt.Printf("Status: %s\n", getStatusWithIcon(env.Status))

	// Source
	fmt.Println("\nSource:")
	fmt.Printf("  Type: %s\n", env.Source.Type)
	if env.Source.PRNumber > 0 {
		fmt.Printf("  PR: #%d\n", env.Source.PRNumber)
	}
	if env.Source.Branch != "" {
		fmt.Printf("  Branch: %s\n", env.Source.Branch)
	}
	if env.Source.CommitSHA != "" {
		fmt.Printf("  Commit: %s\n", env.Source.CommitSHA)
	}
	if env.Source.Repository != "" {
		fmt.Printf("  Repository: %s\n", env.Source.Repository)
	}

	// Access
	fmt.Println("\nAccess:")
	fmt.Printf("  Preview URL: %s\n", env.PreviewURL)
	fmt.Printf("  Namespace: %s\n", env.Namespace)

	// Resources
	if len(env.Resources) > 0 {
		fmt.Println("\nResources:")
		for _, res := range env.Resources {
			fmt.Printf("  %s: %s", res.Service, res.Status)
			if res.URL != "" {
				fmt.Printf(" (%s)", res.URL)
			}
			fmt.Println()
		}
	}

	// Timing
	fmt.Println("\nTiming:")
	fmt.Printf("  Created: %s\n", env.CreatedAt.Format(time.RFC3339))
	if env.ReadyAt != nil {
		fmt.Printf("  Ready: %s\n", env.ReadyAt.Format(time.RFC3339))
	}
	fmt.Printf("  Expires: %s\n", env.ExpiresAt.Format(time.RFC3339))

	expiresIn := time.Until(env.ExpiresAt).Round(time.Minute)
	if expiresIn > 0 {
		fmt.Printf("  Expires In: %s\n", expiresIn)
	} else {
		fmt.Printf("  Expired: %s ago\n", -expiresIn)
	}

	// Error
	if env.Error != "" {
		fmt.Printf("\nError: %s\n", env.Error)
	}

	return nil
}

func runEnvDelete(cmd *cobra.Command, args []string) error {
	envID := args[0]

	if err := ephemeralManager.Delete(envID); err != nil {
		return err
	}

	fmt.Printf("Environment %s marked for deletion\n", envID)
	return nil
}

func runEnvExtend(cmd *cobra.Command, args []string) error {
	envID := args[0]

	if err := ephemeralManager.ExtendTTL(envID, envExtendBy); err != nil {
		return err
	}

	env, _ := ephemeralManager.Get(envID)
	fmt.Printf("Extended TTL for environment %s\n", envID)
	fmt.Printf("New expiry: %s\n", env.ExpiresAt.Format(time.RFC3339))
	return nil
}

func getStatusWithIcon(status environment.EphemeralEnvironmentStatus) string {
	switch status {
	case environment.EphemeralStatusPending:
		return "Pending"
	case environment.EphemeralStatusProvisioning:
		return "Provisioning..."
	case environment.EphemeralStatusReady:
		return "Ready"
	case environment.EphemeralStatusFailed:
		return "Failed"
	case environment.EphemeralStatusExpired:
		return "Expired"
	case environment.EphemeralStatusDeleting:
		return "Deleting..."
	case environment.EphemeralStatusDeleted:
		return "Deleted"
	default:
		return string(status)
	}
}
