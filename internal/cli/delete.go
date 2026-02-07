package cli

import (
	"fmt"
	"strings"

	"github.com/platformfoundry/pf-ce/internal/orchestrator"
	"github.com/platformfoundry/pf-ce/internal/plugin"
	"github.com/platformfoundry/pf-ce/internal/plugins/clusterexisting"
	"github.com/platformfoundry/pf-ce/internal/store"
	"github.com/spf13/cobra"
)

var deleteCmd = &cobra.Command{
	Use:   "delete [resource-type] [name]",
	Short: "Delete a resource",
	Long:  `Delete a specific resource by type and name.`,
	Example: `  pf delete cluster dev-cluster
  pf delete pipeline jenkins-ci`,
	Args: cobra.ExactArgs(2),
	RunE: runDelete,
}

func runDelete(cmd *cobra.Command, args []string) error {
	resourceType := args[0]
	name := args[1]

	// Capitalize first letter for Kind matching
	kind := strings.ToUpper(resourceType[:1]) + resourceType[1:]
	if strings.HasSuffix(kind, "s") {
		kind = kind[:len(kind)-1] // Remove plural 's'
	}

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

	// Delete resource
	if err := orch.Delete(name, kind); err != nil {
		return fmt.Errorf("failed to delete resource: %w", err)
	}

	return nil
}
