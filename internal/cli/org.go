package cli

import (
	"fmt"

	"github.com/platformfoundry/pf-ce/internal/context"
	"github.com/spf13/cobra"
)

var orgCmd = &cobra.Command{
	Use:   "org",
	Short: "Manage organization context and membership",
	Long:  `Manage organization context, users, and roles.`,
}

var orgSetCmd = &cobra.Command{
	Use:   "set <organization>",
	Short: "Set current organization context",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		ctxMgr, err := context.NewManager()
		if err != nil {
			return err
		}

		if err := ctxMgr.SetOrganization(args[0]); err != nil {
			return err
		}

		fmt.Printf("Switched to organization: %s\n", args[0])
		return nil
	},
}

var orgCurrentCmd = &cobra.Command{
	Use:   "current",
	Short: "Display current organization context",
	RunE: func(cmd *cobra.Command, args []string) error {
		ctxMgr, err := context.NewManager()
		if err != nil {
			return err
		}

		fmt.Printf("Current organization: %s\n", ctxMgr.GetCurrentOrganization())
		if env := ctxMgr.GetCurrentEnvironment(); env != "" {
			fmt.Printf("Current environment: %s\n", env)
		}
		fmt.Printf("Current user: %s\n", ctxMgr.GetCurrentUser())

		return nil
	},
}

var orgSetEnvCmd = &cobra.Command{
	Use:   "set-env <environment>",
	Short: "Set current environment context",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		ctxMgr, err := context.NewManager()
		if err != nil {
			return err
		}

		if err := ctxMgr.SetEnvironment(args[0]); err != nil {
			return err
		}

		fmt.Printf("Switched to environment: %s\n", args[0])
		return nil
	},
}

func init() {
	orgCmd.AddCommand(orgSetCmd)
	orgCmd.AddCommand(orgCurrentCmd)
	orgCmd.AddCommand(orgSetEnvCmd)
}
