package cli

import (
	"fmt"
	"strings"

	"github.com/platformfoundry/pf-ce/internal/store"
	"github.com/spf13/cobra"
)

var getCmd = &cobra.Command{
	Use:   "get [resource-type|all]",
	Short: "Get resources",
	Long:  `List resources of a specific type or all resources.`,
	Example: `  pf get clusters
  pf get pipelines
  pf get all`,
	Args:              cobra.ExactArgs(1),
	ValidArgsFunction: resourceTypeCompletion,
	RunE:              runGet,
}

func runGet(cmd *cobra.Command, args []string) error {
	resourceType := args[0]

	// Initialize store
	st, err := store.New()
	if err != nil {
		return fmt.Errorf("failed to initialize store: %w", err)
	}

	var resources []store.ResourceState

	if resourceType == "all" {
		resources, err = st.List()
	} else {
		// Capitalize first letter for Kind matching
		kind := strings.ToUpper(resourceType[:1]) + resourceType[1:]
		if strings.HasSuffix(kind, "s") {
			kind = kind[:len(kind)-1] // Remove plural 's'
		}
		resources, err = st.ListByKind(kind)
	}

	if err != nil {
		return fmt.Errorf("failed to list resources: %w", err)
	}

	if len(resources) == 0 {
		fmt.Println("No resources found")
		return nil
	}

	// Print table header
	fmt.Printf("%-20s %-15s %-15s %-15s\n", "NAME", "KIND", "PROVIDER", "STATUS")
	fmt.Println(strings.Repeat("-", 65))

	// Print resources
	for _, r := range resources {
		fmt.Printf("%-20s %-15s %-15s %-15s\n", r.Name, r.Kind, r.Provider, r.Status)
	}

	return nil
}
