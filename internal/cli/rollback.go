package cli

import (
	"fmt"

	"github.com/platformfoundry/pf-ce/internal/orchestrator"
	"github.com/platformfoundry/pf-ce/internal/plugin"
	"github.com/platformfoundry/pf-ce/internal/plugins/clusterexisting"
	"github.com/platformfoundry/pf-ce/internal/store"
	"github.com/spf13/cobra"
)

var rollbackCmd = &cobra.Command{
	Use:   "rollback <job-id>",
	Short: "Rollback changes from a job",
	Long:  `Rollback changes made by a specific job. This will undo all changes in reverse order.`,
	Args:  cobra.ExactArgs(1),
	RunE:  runRollback,
}

var rollbackListCmd = &cobra.Command{
	Use:   "list",
	Short: "List available rollback plans",
	Long:  `List all available rollback plans.`,
	RunE:  runRollbackList,
}

func init() {
	rollbackCmd.AddCommand(rollbackListCmd)
}

func runRollback(cmd *cobra.Command, args []string) error {
	jobID := args[0]

	// Initialize components
	pm := plugin.NewManager()
	st, err := store.New()
	if err != nil {
		return fmt.Errorf("failed to initialize store: %w", err)
	}

	// Register built-in plugins
	pm.Register(clusterexisting.NewPlugin())

	// Initialize orchestrator
	orch := orchestrator.New(pm, st)

	// Create rollback manager
	rbm := orchestrator.NewRollbackManager(orch)

	// Check if rollback plan exists
	plan, err := rbm.GetRollbackPlan(jobID)
	if err != nil {
		return fmt.Errorf("failed to get rollback plan: %w", err)
	}

	// Show plan details
	fmt.Printf("Rollback Plan for Job: %s\n", plan.JobID)
	fmt.Printf("Created: %s\n", plan.CreatedAt.Format("2006-01-02 15:04:05"))
	fmt.Printf("Resources: %d\n\n", len(plan.Resources))

	// Ask for confirmation
	fmt.Print("Are you sure you want to rollback? This action cannot be undone. (yes/no): ")
	var confirm string
	fmt.Scanln(&confirm)

	if confirm != "yes" && confirm != "y" {
		fmt.Println("Rollback cancelled.")
		return nil
	}

	// Execute rollback
	if err := rbm.ExecuteRollback(jobID); err != nil {
		return fmt.Errorf("rollback failed: %w", err)
	}

	return nil
}

func runRollbackList(cmd *cobra.Command, args []string) error {
	// Initialize components
	pm := plugin.NewManager()
	st, err := store.New()
	if err != nil {
		return fmt.Errorf("failed to initialize store: %w", err)
	}

	// Register built-in plugins
	pm.Register(clusterexisting.NewPlugin())

	// Initialize orchestrator
	orch := orchestrator.New(pm, st)

	// Create rollback manager
	rbm := orchestrator.NewRollbackManager(orch)

	// List plans
	plans := rbm.ListRollbackPlans()

	if len(plans) == 0 {
		fmt.Println("No rollback plans available.")
		return nil
	}

	fmt.Printf("%-36s %-20s %-10s\n", "JOB ID", "CREATED", "RESOURCES")
	fmt.Println("--------------------------------------------------------------------")

	for _, plan := range plans {
		fmt.Printf("%-36s %-20s %d\n",
			plan.JobID,
			plan.CreatedAt.Format("2006-01-02 15:04:05"),
			len(plan.Resources),
		)
	}

	return nil
}
