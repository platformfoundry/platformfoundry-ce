package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/platformfoundry/pf-ce/internal/dcm"
	"github.com/platformfoundry/pf-ce/internal/score"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

var deployCmd = &cobra.Command{
	Use:   "deploy",
	Short: "Deployment and deployment set commands",
	Long:  `Manage deployments and deployment sets with immutable versioning.`,
}

var deploySetsCmd = &cobra.Command{
	Use:   "sets",
	Short: "Manage deployment sets",
	Long:  `List, view, diff, and manage deployment sets.`,
}

var deploySetsListCmd = &cobra.Command{
	Use:   "list",
	Short: "List deployment sets",
	Long:  `List all deployment sets for an application.`,
	RunE:  runDeploySetslist,
}

var deploySetsShowCmd = &cobra.Command{
	Use:   "show [id]",
	Short: "Show deployment set details",
	Long:  `Show details of a specific deployment set.`,
	Args:  cobra.ExactArgs(1),
	RunE:  runDeploySetsShow,
}

var deploySetsDiffCmd = &cobra.Command{
	Use:   "diff [version1] [version2]",
	Short: "Compare deployment sets",
	Long:  `Compare two deployment sets and show differences.`,
	Args:  cobra.ExactArgs(2),
	RunE:  runDeploySetsDiff,
}

var deploySetsRollbackCmd = &cobra.Command{
	Use:   "rollback [version]",
	Short: "Rollback to a previous deployment set",
	Long:  `Rollback to a specific deployment set version.`,
	Args:  cobra.ExactArgs(1),
	RunE:  runDeploySetsRollback,
}

var deployApplyCmd = &cobra.Command{
	Use:   "apply [file]",
	Short: "Apply a deployment from Score file",
	Long:  `Create and apply a deployment from a Score specification file.`,
	Args:  cobra.ExactArgs(1),
	RunE:  runDeployApply,
}

var deployStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show deployment status",
	Long:  `Show the status of the current deployment.`,
	RunE:  runDeployStatus,
}

// Flags
var (
	deployApp     string
	deployEnv     string
	deployDryRun  bool
	deployFormat  string
	deployWait    bool
	deployTimeout time.Duration
)

func init() {
	// Common flags
	deployCmd.PersistentFlags().StringVarP(&deployApp, "app", "a", "", "Application name")
	deployCmd.PersistentFlags().StringVarP(&deployEnv, "env", "e", "", "Environment name")
	deployCmd.PersistentFlags().StringVar(&deployFormat, "format", "table", "Output format (table, json, yaml)")

	// Apply flags
	deployApplyCmd.Flags().BoolVar(&deployDryRun, "dry-run", false, "Preview changes without applying")
	deployApplyCmd.Flags().BoolVarP(&deployWait, "wait", "w", false, "Wait for deployment to complete")
	deployApplyCmd.Flags().DurationVar(&deployTimeout, "timeout", 5*time.Minute, "Timeout for deployment")

	// Sets subcommands
	deploySetsCmd.AddCommand(deploySetsListCmd)
	deploySetsCmd.AddCommand(deploySetsShowCmd)
	deploySetsCmd.AddCommand(deploySetsDiffCmd)
	deploySetsCmd.AddCommand(deploySetsRollbackCmd)

	deployCmd.AddCommand(deploySetsCmd)
	deployCmd.AddCommand(deployApplyCmd)
	deployCmd.AddCommand(deployStatusCmd)
}

func runDeploySetslist(cmd *cobra.Command, args []string) error {
	if deployApp == "" {
		return fmt.Errorf("--app is required")
	}
	if deployEnv == "" {
		return fmt.Errorf("--env is required")
	}

	ctx := context.Background()
	engine := dcm.NewEngine(nil, dcm.EngineConfig{})

	sets, err := engine.ListDeploymentSets(ctx, deployApp, deployEnv)
	if err != nil {
		return err
	}

	if deployFormat == "json" {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(sets)
	}

	if deployFormat == "yaml" {
		enc := yaml.NewEncoder(os.Stdout)
		return enc.Encode(sets)
	}

	// Table format
	fmt.Printf("Deployment Sets for %s/%s:\n\n", deployApp, deployEnv)
	fmt.Printf("%-30s %-10s %-15s %-20s %s\n", "ID", "VERSION", "STATUS", "DEPLOYED", "HASH")
	fmt.Println(strings.Repeat("-", 100))

	for _, set := range sets {
		deployedAt := "-"
		if set.DeployedAt != nil {
			deployedAt = set.DeployedAt.Format("2006-01-02 15:04:05")
		}
		hash := set.Hash
		if len(hash) > 12 {
			hash = hash[:12]
		}
		fmt.Printf("%-30s %-10d %-15s %-20s %s\n",
			set.ID, set.Version, set.Status, deployedAt, hash)
	}

	return nil
}

func runDeploySetsShow(cmd *cobra.Command, args []string) error {
	ctx := context.Background()
	engine := dcm.NewEngine(nil, dcm.EngineConfig{})

	set, err := engine.GetDeploymentSet(ctx, args[0])
	if err != nil {
		return err
	}

	if deployFormat == "json" {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(set)
	}

	if deployFormat == "yaml" {
		enc := yaml.NewEncoder(os.Stdout)
		return enc.Encode(set)
	}

	// Detailed view
	fmt.Printf("Deployment Set: %s\n", set.ID)
	fmt.Printf("Application:    %s\n", set.Application)
	fmt.Printf("Environment:    %s\n", set.Environment)
	fmt.Printf("Version:        %d\n", set.Version)
	fmt.Printf("Status:         %s\n", set.Status)
	fmt.Printf("Created:        %s\n", set.CreatedAt.Format(time.RFC3339))
	if set.DeployedAt != nil {
		fmt.Printf("Deployed:       %s\n", set.DeployedAt.Format(time.RFC3339))
	}
	fmt.Printf("Hash:           %s\n", set.Hash)
	fmt.Println()

	// Resources
	fmt.Println("Resources:")
	for _, res := range set.Resources {
		fmt.Printf("  - %s (%s): %s\n", res.Name, res.Type, res.Status)
		if len(res.Outputs) > 0 {
			for k, v := range res.Outputs {
				// Mask secrets
				display := v
				if strings.Contains(strings.ToLower(k), "password") || strings.Contains(strings.ToLower(k), "secret") {
					display = "***"
				}
				fmt.Printf("      %s: %s\n", k, display)
			}
		}
	}

	// Config
	if len(set.Config) > 0 {
		fmt.Println()
		fmt.Println("Generated Config:")
		for k, v := range set.Config {
			fmt.Printf("  %s: %v\n", k, v)
		}
	}

	return nil
}

func runDeploySetsDiff(cmd *cobra.Command, args []string) error {
	if deployApp == "" {
		return fmt.Errorf("--app is required")
	}
	if deployEnv == "" {
		return fmt.Errorf("--env is required")
	}

	ctx := context.Background()
	engine := dcm.NewEngine(nil, dcm.EngineConfig{})

	// Get deployment sets
	var version1, version2 int
	fmt.Sscanf(args[0], "%d", &version1)
	fmt.Sscanf(args[1], "%d", &version2)

	sets, err := engine.ListDeploymentSets(ctx, deployApp, deployEnv)
	if err != nil {
		return err
	}

	var set1, set2 *dcm.DeploymentSet
	for _, set := range sets {
		if set.Version == version1 {
			set1 = set
		}
		if set.Version == version2 {
			set2 = set
		}
	}

	if set1 == nil {
		return fmt.Errorf("deployment set version %d not found", version1)
	}
	if set2 == nil {
		return fmt.Errorf("deployment set version %d not found", version2)
	}

	// Compute diff
	delta := engine.DiffDeploymentSets(set1, set2)

	if deployFormat == "json" {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(delta)
	}

	// Display diff
	fmt.Printf("Diff: v%d -> v%d\n\n", version1, version2)

	if len(delta.Added) > 0 {
		fmt.Println("Added:")
		for _, res := range delta.Added {
			fmt.Printf("  + %s (%s)\n", res.Name, res.Type)
		}
	}

	if len(delta.Modified) > 0 {
		fmt.Println("Modified:")
		for _, res := range delta.Modified {
			fmt.Printf("  ~ %s (%s)\n", res.Name, res.Type)
		}
	}

	if len(delta.Removed) > 0 {
		fmt.Println("Removed:")
		for _, res := range delta.Removed {
			fmt.Printf("  - %s (%s)\n", res.Name, res.Type)
		}
	}

	if len(delta.Added) == 0 && len(delta.Modified) == 0 && len(delta.Removed) == 0 {
		fmt.Println("No changes")
	}

	return nil
}

func runDeploySetsRollback(cmd *cobra.Command, args []string) error {
	if deployApp == "" {
		return fmt.Errorf("--app is required")
	}
	if deployEnv == "" {
		return fmt.Errorf("--env is required")
	}

	var version int
	fmt.Sscanf(args[0], "%d", &version)

	ctx := context.Background()
	engine := dcm.NewEngine(nil, dcm.EngineConfig{})
	dcm.RegisterBuiltinDrivers(engine)

	fmt.Printf("Rolling back %s/%s to version %d...\n", deployApp, deployEnv, version)

	newSet, err := engine.Rollback(ctx, deployApp, deployEnv, version)
	if err != nil {
		return fmt.Errorf("rollback failed: %w", err)
	}

	fmt.Printf("Rollback complete. New deployment set: %s (v%d)\n", newSet.ID, newSet.Version)
	return nil
}

func runDeployApply(cmd *cobra.Command, args []string) error {
	if deployApp == "" {
		return fmt.Errorf("--app is required")
	}
	if deployEnv == "" {
		return fmt.Errorf("--env is required")
	}

	// Parse Score file
	parser := score.NewParser()
	workload, err := parser.ParseFile(args[0])
	if err != nil {
		return fmt.Errorf("failed to parse Score file: %w", err)
	}

	// Validate
	errors := parser.Validate(workload)
	for _, e := range errors {
		if e.Severity == "error" {
			return fmt.Errorf("validation error: %s: %s", e.Field, e.Message)
		}
	}

	// Convert Score resources to DCM resources
	resources := make([]*dcm.ResourceNode, 0)

	for name, res := range workload.Resources {
		node := &dcm.ResourceNode{
			ID:          fmt.Sprintf("%s-%s-%s", deployApp, deployEnv, name),
			Type:        res.Type,
			Name:        name,
			Class:       res.Class,
			Params:      res.Params,
			Status:      dcm.StatusPending,
			Application: deployApp,
			Environment: deployEnv,
			CreatedAt:   time.Now(),
		}
		resources = append(resources, node)
	}

	// Create DCM engine
	ctx := context.Background()
	engine := dcm.NewEngine(nil, dcm.EngineConfig{})
	dcm.RegisterBuiltinDrivers(engine)

	// Preview mode
	if deployDryRun {
		fmt.Println("Dry-run mode - no changes will be applied")
		fmt.Println()
		fmt.Printf("Application: %s\n", deployApp)
		fmt.Printf("Environment: %s\n", deployEnv)
		fmt.Printf("Workload:    %s\n", workload.Metadata.Name)
		fmt.Println()
		fmt.Println("Resources to provision:")
		for _, res := range resources {
			fmt.Printf("  - %s (%s)\n", res.Name, res.Type)
		}
		return nil
	}

	// Create deployment set
	fmt.Printf("Creating deployment for %s/%s...\n", deployApp, deployEnv)

	set, err := engine.CreateDeploymentSet(ctx, deployApp, deployEnv, resources)
	if err != nil {
		return fmt.Errorf("failed to create deployment set: %w", err)
	}

	fmt.Printf("Deployment set created: %s (v%d)\n", set.ID, set.Version)

	// Deploy
	fmt.Println("Deploying resources...")

	if deployWait {
		ctx, cancel := context.WithTimeout(ctx, deployTimeout)
		defer cancel()

		if err := engine.Deploy(ctx, set); err != nil {
			return fmt.Errorf("deployment failed: %w", err)
		}

		fmt.Println()
		fmt.Printf("Deployment complete: %s\n", set.Status)
		fmt.Println()
		fmt.Println("Resources:")
		for _, res := range set.Resources {
			fmt.Printf("  - %s (%s): %s\n", res.Name, res.Type, res.Status)
		}
	} else {
		go engine.Deploy(context.Background(), set)
		fmt.Println("Deployment started in background")
		fmt.Printf("Check status with: pf deploy status --app %s --env %s\n", deployApp, deployEnv)
	}

	return nil
}

func runDeployStatus(cmd *cobra.Command, args []string) error {
	if deployApp == "" {
		return fmt.Errorf("--app is required")
	}
	if deployEnv == "" {
		return fmt.Errorf("--env is required")
	}

	ctx := context.Background()
	engine := dcm.NewEngine(nil, dcm.EngineConfig{})

	sets, err := engine.ListDeploymentSets(ctx, deployApp, deployEnv)
	if err != nil {
		return err
	}

	if len(sets) == 0 {
		fmt.Printf("No deployments found for %s/%s\n", deployApp, deployEnv)
		return nil
	}

	// Get latest
	latest := sets[len(sets)-1]

	fmt.Printf("Application: %s\n", deployApp)
	fmt.Printf("Environment: %s\n", deployEnv)
	fmt.Println()
	fmt.Printf("Current Deployment: %s (v%d)\n", latest.ID, latest.Version)
	fmt.Printf("Status:             %s\n", latest.Status)
	if latest.DeployedAt != nil {
		fmt.Printf("Deployed:           %s\n", latest.DeployedAt.Format(time.RFC3339))
	}
	fmt.Println()

	// Resource status
	fmt.Println("Resources:")
	for _, res := range latest.Resources {
		statusIcon := "○"
		switch res.Status {
		case dcm.StatusReady:
			statusIcon = "●"
		case dcm.StatusFailed:
			statusIcon = "✗"
		case dcm.StatusProvisioning:
			statusIcon = "◐"
		}
		fmt.Printf("  %s %s (%s): %s\n", statusIcon, res.Name, res.Type, res.Status)
	}

	return nil
}

// GetDeployCmd returns the deploy command for registration
func GetDeployCmd() *cobra.Command {
	return deployCmd
}
