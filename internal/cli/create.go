package cli

import (
	"github.com/spf13/cobra"
)

var createCmd = &cobra.Command{
	Use:   "create [resource-type]",
	Short: "Create resources interactively or from templates",
	Long:  `Generate YAML definitions for resources interactively. Use --apply to create directly.`,
	Example: `  pf create organization
  pf create platform
  pf create organization --apply
  pf create platform --name my-platform --org acme`,
}

func init() {
	// Add subcommands
	createCmd.AddCommand(createOrgCmd)
	createCmd.AddCommand(createPlatformCmd)
}
