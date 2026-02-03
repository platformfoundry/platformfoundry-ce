package cli

import (
	"fmt"

	"github.com/platformfoundry/pf-ce/internal/parser"
	"github.com/platformfoundry/pf-ce/internal/planner"
	"github.com/platformfoundry/pf-ce/internal/plugin"
	"github.com/platformfoundry/pf-ce/internal/plugins/clusterexisting"
	"github.com/platformfoundry/pf-ce/internal/store"
	"github.com/spf13/cobra"
)

var planFile string

var planCmd = &cobra.Command{
	Use:   "plan -f <file>",
	Short: "Show execution plan for resources",
	Long:  `Generate and display an execution plan showing what actions will be taken without making any changes.`,
	Example: `  pf plan -f platform.yaml
  pf plan -f cluster.yaml`,
	RunE: runPlan,
}

func init() {
	planCmd.Flags().StringVarP(&planFile, "file", "f", "", "YAML file containing resources (required)")
	planCmd.MarkFlagRequired("file")
}

func runPlan(cmd *cobra.Command, args []string) error {
	// Initialize components
	pm := plugin.NewManager()
	st, err := store.New()
	if err != nil {
		return fmt.Errorf("failed to initialize store: %w", err)
	}

	// Register built-in plugins
	pm.Register(clusterexisting.NewPlugin())

	// Parse YAML file
	p := parser.New()
	resources, err := p.ParseFile(planFile)
	if err != nil {
		return fmt.Errorf("failed to parse file: %w", err)
	}

	fmt.Printf("Parsed %d resource(s) from %s\n\n", len(resources), planFile)

	// Convert to []interface{}
	resourcesInterface := make([]interface{}, len(resources))
	for i, r := range resources {
		resourcesInterface[i] = r
	}

	// Create execution plan
	plnr := planner.New(pm, st)
	plan, err := plnr.CreatePlan(resourcesInterface)
	if err != nil {
		return fmt.Errorf("failed to create plan: %w", err)
	}

	// Display plan
	fmt.Println("EXECUTION PLAN")
	fmt.Println("==============")
	fmt.Println()

	if len(plan.ToCreate) > 0 {
		fmt.Println("Resources to CREATE:")
		for i, res := range plan.ToCreate {
			fmt.Printf("  %d. %s (kind: %s)\n", i+1, res.Name, res.Kind)
		}
		fmt.Println()
	}

	if len(plan.ToUpdate) > 0 {
		fmt.Println("Resources to UPDATE:")
		for i, res := range plan.ToUpdate {
			fmt.Printf("  %d. %s (kind: %s)\n", i+1, res.Name, res.Kind)
			if len(res.Changes) > 0 {
				for _, change := range res.Changes {
					fmt.Printf("     ~ %s\n", change)
				}
			}
		}
		fmt.Println()
	}

	if len(plan.ToDelete) > 0 {
		fmt.Println("Resources to DELETE:")
		for i, res := range plan.ToDelete {
			fmt.Printf("  %d. %s (kind: %s)\n", i+1, res.Name, res.Kind)
		}
		fmt.Println()
	}

	if len(plan.Unchanged) > 0 {
		fmt.Println("Resources UNCHANGED:")
		for i, res := range plan.Unchanged {
			fmt.Printf("  %d. %s (kind: %s)\n", i+1, res.Name, res.Kind)
		}
		fmt.Println()
	}

	// Show execution order
	if len(plan.ExecutionOrder) > 0 {
		fmt.Println("Execution Order:")
		for i, name := range plan.ExecutionOrder {
			fmt.Printf("  %d. %s\n", i+1, name)
		}
		fmt.Println()
	}

	// Summary
	fmt.Printf("Plan: %d to create, %d to update, %d to delete, %d unchanged\n",
		len(plan.ToCreate),
		len(plan.ToUpdate),
		len(plan.ToDelete),
		len(plan.Unchanged),
	)

	fmt.Println("\nTo apply this plan, run: pf apply -f " + planFile)

	return nil
}
