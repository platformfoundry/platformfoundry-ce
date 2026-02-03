package cli

import (
	"fmt"

	"github.com/platformfoundry/pf-ce/internal/scaffold"
	"github.com/spf13/cobra"
)

func NewScaffoldCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "scaffold",
		Short: "Generate scaffold configurations for platform components",
		Long:  `Generate starter configurations for rapid platform setup`,
	}

	cmd.AddCommand(newScaffoldPlatformCmd())
	cmd.AddCommand(newScaffoldComponentCmd())
	cmd.AddCommand(newScaffoldFullCmd())

	return cmd
}

func newScaffoldFullCmd() *cobra.Command {
	var (
		name        string
		outputDir   string
		provider    string
		mockMode    bool
		environment string
	)

	cmd := &cobra.Command{
		Use:   "full",
		Short: "Generate a complete platform scaffold",
		Example: `  # Generate a full platform scaffold with mock mode
  pf scaffold full --name my-platform --provider aws --mock

  # Generate production-ready scaffold
  pf scaffold full --name my-platform --provider aws --env prod`,
		RunE: func(cmd *cobra.Command, args []string) error {
			gen := scaffold.NewGenerator()

			config := scaffold.ScaffoldConfig{
				Type:          scaffold.ScaffoldFull,
				Name:          name,
				OutputDir:     outputDir,
				CloudProvider: provider,
				MockMode:      mockMode,
				Environment:   environment,
			}

			if _, err := gen.Generate(config); err != nil {
				return err
			}

			fmt.Printf("Generated platform scaffold in %s\n", outputDir)
			fmt.Println("\nNext steps:")
			if mockMode {
				fmt.Println("  1. Review generated files in", outputDir)
				fmt.Println("  2. Run: pf apply -f", outputDir+"/platform.yaml", "--mock")
				fmt.Println("  3. Test your platform locally")
			} else {
				fmt.Println("  1. Review and customize generated files in", outputDir)
				fmt.Println("  2. Set required environment variables")
				fmt.Println("  3. Run: pf plan -f", outputDir+"/platform.yaml")
				fmt.Println("  4. Run: pf apply -f", outputDir+"/platform.yaml")
			}

			return nil
		},
	}

	cmd.Flags().StringVar(&name, "name", "", "Platform name (required)")
	cmd.Flags().StringVar(&outputDir, "output", "./platform", "Output directory")
	cmd.Flags().StringVar(&provider, "provider", "aws", "Cloud provider (aws, gcp, azure)")
	cmd.Flags().BoolVar(&mockMode, "mock", false, "Generate with mock providers")
	cmd.Flags().StringVar(&environment, "env", "dev", "Target environment (dev, staging, prod)")

	cmd.MarkFlagRequired("name")

	return cmd
}

func newScaffoldPlatformCmd() *cobra.Command {
	var (
		name        string
		outputDir   string
		provider    string
		mockMode    bool
		environment string
	)

	cmd := &cobra.Command{
		Use:   "platform",
		Short: "Generate a platform.yaml file",
		RunE: func(cmd *cobra.Command, args []string) error {
			gen := scaffold.NewGenerator()

			config := scaffold.ScaffoldConfig{
				Type:          scaffold.ScaffoldPlatform,
				Name:          name,
				OutputDir:     outputDir,
				CloudProvider: provider,
				MockMode:      mockMode,
				Environment:   environment,
			}

			if _, err := gen.Generate(config); err != nil {
				return err
			}

			fmt.Printf("Generated platform.yaml in %s\n", outputDir)
			return nil
		},
	}

	cmd.Flags().StringVar(&name, "name", "", "Platform name (required)")
	cmd.Flags().StringVar(&outputDir, "output", "./platform", "Output directory")
	cmd.Flags().StringVar(&provider, "provider", "aws", "Cloud provider (aws, gcp, azure)")
	cmd.Flags().BoolVar(&mockMode, "mock", false, "Generate with mock providers")
	cmd.Flags().StringVar(&environment, "env", "dev", "Target environment (dev, staging, prod)")

	cmd.MarkFlagRequired("name")

	return cmd
}

func newScaffoldComponentCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "component [type]",
		Short: "Generate a component configuration file",
	}

	types := []scaffold.ScaffoldType{
		scaffold.ScaffoldInfrastructure,
		scaffold.ScaffoldOrchestrator,
		scaffold.ScaffoldObservability,
		scaffold.ScaffoldDevEx,
		scaffold.ScaffoldSecurity,
	}

	for _, t := range types {
		componentCmd := newComponentScaffoldCmd(t)
		cmd.AddCommand(componentCmd)
	}

	return cmd
}

func newComponentScaffoldCmd(scaffoldType scaffold.ScaffoldType) *cobra.Command {
	var (
		name        string
		outputDir   string
		provider    string
		mockMode    bool
		environment string
	)

	cmd := &cobra.Command{
		Use:   string(scaffoldType),
		Short: fmt.Sprintf("Generate a %s component configuration", scaffoldType),
		RunE: func(cmd *cobra.Command, args []string) error {
			gen := scaffold.NewGenerator()

			config := scaffold.ScaffoldConfig{
				Type:          scaffoldType,
				Name:          name,
				OutputDir:     outputDir,
				CloudProvider: provider,
				MockMode:      mockMode,
				Environment:   environment,
			}

			if _, err := gen.Generate(config); err != nil {
				return err
			}

			fmt.Printf("Generated %s configuration in %s\n", scaffoldType, outputDir)
			return nil
		},
	}

	cmd.Flags().StringVar(&name, "name", "my-platform", "Platform name")
	cmd.Flags().StringVar(&outputDir, "output", "./platform", "Output directory")
	cmd.Flags().StringVar(&provider, "provider", "aws", "Cloud provider (aws, gcp, azure)")
	cmd.Flags().BoolVar(&mockMode, "mock", false, "Generate with mock providers")
	cmd.Flags().StringVar(&environment, "env", "dev", "Target environment (dev, staging, prod)")

	return cmd
}
