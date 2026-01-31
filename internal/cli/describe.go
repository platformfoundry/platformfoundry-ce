package cli

import (
	"encoding/json"
	"fmt"

	"github.com/platformfoundry/platformfoundry-ce/internal/store"
	"github.com/spf13/cobra"
)

var describeCmd = &cobra.Command{
	Use:   "describe [resource-type] [name]",
	Short: "Show detailed information about a resource",
	Long:  `Display detailed information about a specific resource.`,
	Example: `  pf describe cluster dev-cluster
  pf describe pipeline jenkins-ci`,
	Args: cobra.ExactArgs(2),
	RunE: runDescribe,
}

func runDescribe(cmd *cobra.Command, args []string) error {
	name := args[1]

	// Initialize store
	st, err := store.New()
	if err != nil {
		return fmt.Errorf("failed to initialize store: %w", err)
	}

	// Get resource
	resource, err := st.Get(name)
	if err != nil {
		return fmt.Errorf("resource not found: %w", err)
	}

	// Print resource details
	fmt.Printf("Name:      %s\n", resource.Name)
	fmt.Printf("Kind:      %s\n", resource.Kind)
	fmt.Printf("Provider:  %s\n", resource.Provider)
	fmt.Printf("Status:    %s\n", resource.Status)
	fmt.Printf("Created:   %s\n", resource.CreatedAt.Format("2006-01-02 15:04:05"))
	fmt.Printf("Updated:   %s\n", resource.UpdatedAt.Format("2006-01-02 15:04:05"))
	fmt.Println("\nSpec:")

	// Pretty print spec JSON
	var spec map[string]interface{}
	if err := json.Unmarshal([]byte(resource.Spec), &spec); err == nil {
		specJSON, _ := json.MarshalIndent(spec, "", "  ")
		fmt.Println(string(specJSON))
	}

	return nil
}
