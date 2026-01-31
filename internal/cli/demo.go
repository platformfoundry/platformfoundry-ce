package cli

import (
	"fmt"
	"os"
	"time"

	"github.com/platformfoundry/platformfoundry-ce/internal/demo"
	"github.com/spf13/cobra"
)

var demoCmd = &cobra.Command{
	Use:   "demo",
	Short: "Run a local demo of Platform Foundry",
	Long: `Run a complete local demo of Platform Foundry.

This command will:
  1. Create a local Kubernetes cluster using kind
  2. Install ArgoCD, Prometheus, Grafana, and Backstage
  3. Configure all integrations automatically
  4. Provide access URLs for all components

Prerequisites:
  - Docker must be running
  - 8GB RAM available

This demo runs entirely on your local machine - no cloud account needed!`,
	RunE: runDemo,
}

var demoQuickCmd = &cobra.Command{
	Use:   "quick",
	Short: "Run a quick demo with minimal components",
	Long:  "Run a quick demo with only Prometheus and Grafana (faster, lower resources)",
	RunE:  runQuickDemo,
}

var demoCleanCmd = &cobra.Command{
	Use:   "clean",
	Short: "Clean up demo resources",
	Long:  "Remove the demo kind cluster and all resources",
	RunE:  cleanDemo,
}

func init() {
	demoCmd.AddCommand(demoQuickCmd)
	demoCmd.AddCommand(demoCleanCmd)
	rootCmd.AddCommand(demoCmd)
}

func runDemo(cmd *cobra.Command, args []string) error {
	fmt.Println("🎬 Platform Foundry - Local Demo")
	fmt.Println("===============================================")
	fmt.Println("")
	fmt.Println("This will create a complete platform on your local machine.")
	fmt.Println("Estimated time: 5-7 minutes")
	fmt.Println("")

	// Check prerequisites
	fmt.Println("🔍 Checking prerequisites...")
	if err := demo.CheckPrerequisites(); err != nil {
		return fmt.Errorf("prerequisites check failed: %w", err)
	}
	fmt.Println("✓ All prerequisites met")
	fmt.Println("")

	// Create demo manager
	mgr := demo.NewManager(&demo.Config{
		ClusterName: "pf-demo",
		Components: []string{
			"prometheus",
			"grafana",
			"argocd",
			"backstage",
		},
		QuickMode: false,
	})

	// Start demo
	startTime := time.Now()
	fmt.Println("📦 Starting demo setup...")
	fmt.Println("")

	if err := mgr.Setup(); err != nil {
		fmt.Printf("\n❌ Demo setup failed: %v\n", err)
		fmt.Println("\nTo clean up, run: pf demo clean")
		return err
	}

	duration := time.Since(startTime)
	fmt.Println("")
	fmt.Println("===============================================")
	fmt.Printf("✅ Platform ready in %s!\n", duration.Round(time.Second))
	fmt.Println("===============================================")
	fmt.Println("")

	// Show access information
	mgr.ShowAccessInfo()

	fmt.Println("")
	fmt.Println("🎯 Try it out:")
	fmt.Println("  1. Open Backstage → See all tools integrated")
	fmt.Println("  2. Open Grafana → See Prometheus datasource auto-configured")
	fmt.Println("  3. Open ArgoCD → Ready for GitOps deployments")
	fmt.Println("")
	fmt.Println("💡 What sets Platform Foundry apart:")
	fmt.Println("  ✓ No manual configuration needed")
	fmt.Println("  ✓ Components auto-discover each other")
	fmt.Println("  ✓ Platform ready in minutes, not days")
	fmt.Println("")
	fmt.Println("🧹 When done, run: pf demo clean")
	fmt.Println("")

	return nil
}

func runQuickDemo(cmd *cobra.Command, args []string) error {
	fmt.Println("🎬 Platform Foundry - Quick Demo")
	fmt.Println("===============================================")
	fmt.Println("")
	fmt.Println("This will create a lightweight platform with Prometheus + Grafana.")
	fmt.Println("Estimated time: 2-3 minutes")
	fmt.Println("")

	// Check prerequisites
	fmt.Println("🔍 Checking prerequisites...")
	if err := demo.CheckPrerequisites(); err != nil {
		return fmt.Errorf("prerequisites check failed: %w", err)
	}
	fmt.Println("✓ All prerequisites met")
	fmt.Println("")

	// Create demo manager
	mgr := demo.NewManager(&demo.Config{
		ClusterName: "pf-demo-quick",
		Components: []string{
			"prometheus",
			"grafana",
		},
		QuickMode: true,
	})

	// Start demo
	startTime := time.Now()
	fmt.Println("📦 Starting quick demo setup...")
	fmt.Println("")

	if err := mgr.Setup(); err != nil {
		fmt.Printf("\n❌ Demo setup failed: %v\n", err)
		fmt.Println("\nTo clean up, run: pf demo clean")
		return err
	}

	duration := time.Since(startTime)
	fmt.Println("")
	fmt.Println("===============================================")
	fmt.Printf("✅ Platform ready in %s!\n", duration.Round(time.Second))
	fmt.Println("===============================================")
	fmt.Println("")

	// Show access information
	mgr.ShowAccessInfo()

	fmt.Println("")
	fmt.Println("🧹 When done, run: pf demo clean")
	fmt.Println("")

	return nil
}

func cleanDemo(cmd *cobra.Command, args []string) error {
	fmt.Println("🧹 Cleaning up demo resources...")
	fmt.Println("")

	// Try both cluster names
	for _, clusterName := range []string{"pf-demo", "pf-demo-quick"} {
		if err := demo.CleanupCluster(clusterName); err != nil {
			// Continue to next cluster if this one doesn't exist
			continue
		}
		fmt.Printf("✓ Removed cluster: %s\n", clusterName)
	}

	// Clean up local directories
	cleanupDirs := []string{
		".pf/demo",
		".pf/kind",
	}

	for _, dir := range cleanupDirs {
		if err := os.RemoveAll(dir); err != nil {
			fmt.Printf("⚠ Warning: Could not remove %s: %v\n", dir, err)
		}
	}

	fmt.Println("")
	fmt.Println("✅ Demo cleanup complete!")
	fmt.Println("")

	return nil
}
