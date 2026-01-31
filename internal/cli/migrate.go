package cli

import (
	"fmt"

	"github.com/platformfoundry/platformfoundry-ce/internal/state"
	"github.com/platformfoundry/platformfoundry-ce/internal/store"
	"github.com/spf13/cobra"
)

var (
	migrateDefaultOrg string
)

var migrateCmd = &cobra.Command{
	Use:   "migrate",
	Short: "Migrate existing resources to organization model",
	Long:  `Migrates existing resources without organization to the default organization.`,
	RunE:  runMigrate,
}

func init() {
	migrateCmd.Flags().StringVar(&migrateDefaultOrg, "default-org", "default",
		"Default organization for existing resources")
	rootCmd.AddCommand(migrateCmd)
}

func runMigrate(cmd *cobra.Command, args []string) error {
	fmt.Println("Starting migration to organization model...")

	// Initialize store
	st, err := store.New()
	if err != nil {
		return fmt.Errorf("failed to initialize store: %w", err)
	}
	defer st.Close()

	// Create default organization
	if err := state.CreateDefaultOrganization(st.GetBackend()); err != nil {
		return err
	}

	// Migrate resources
	if lb, ok := st.GetBackend().(*state.LocalBackend); ok {
		if err := state.MigrateToOrgModel(lb.DB(), migrateDefaultOrg); err != nil {
			return err
		}
	}

	fmt.Println("\nMigration completed successfully!")
	fmt.Printf("All resources assigned to organization: %s\n", migrateDefaultOrg)
	fmt.Printf("\nNext steps:\n")
	fmt.Printf("  1. Review migrated resources: pf get all --org %s\n", migrateDefaultOrg)
	fmt.Printf("  2. Create additional organizations: pf create organization\n")
	fmt.Printf("  3. Set organization context: pf org set %s\n", migrateDefaultOrg)

	return nil
}
