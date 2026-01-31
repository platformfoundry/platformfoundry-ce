package cli

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/platformfoundry/platformfoundry-ce/internal/context"
	"github.com/platformfoundry/platformfoundry-ce/internal/generator"
	"github.com/platformfoundry/platformfoundry-ce/internal/orchestrator"
	"github.com/platformfoundry/platformfoundry-ce/internal/parser"
	"github.com/platformfoundry/platformfoundry-ce/internal/plugin"
	"github.com/platformfoundry/platformfoundry-ce/internal/state"
	"github.com/platformfoundry/platformfoundry-ce/internal/store"
	"github.com/spf13/cobra"
)

var (
	createPlatformName   string
	createPlatformOrg    string
	createPlatformEnvs   []string
	createPlatformApply  bool
	createPlatformOutput string
	createPlatformType   string
)

var createPlatformCmd = &cobra.Command{
	Use:   "platform",
	Short: "Create a platform with environment profiles",
	Long:  `Generate a platform YAML with environment profiles or create it directly with --apply.`,
	Example: `  pf create platform
  pf create platform --name my-platform --org acme
  pf create platform --name my-platform --envs dev,staging,prod --apply
  pf create platform --type kubernetes --output platform.yaml`,
	RunE: runCreatePlatform,
}

func init() {
	createPlatformCmd.Flags().StringVar(&createPlatformName, "name", "", "Platform name")
	createPlatformCmd.Flags().StringVar(&createPlatformOrg, "org", "", "Organization name")
	createPlatformCmd.Flags().StringSliceVar(&createPlatformEnvs, "envs", []string{"dev", "staging", "prod"}, "Environment names")
	createPlatformCmd.Flags().BoolVar(&createPlatformApply, "apply", false, "Apply the platform directly")
	createPlatformCmd.Flags().StringVarP(&createPlatformOutput, "output", "o", "", "Output file (default: stdout)")
	createPlatformCmd.Flags().StringVar(&createPlatformType, "type", "kubernetes", "Platform type (kubernetes, serverless, hybrid)")
}

func runCreatePlatform(cmd *cobra.Command, args []string) error {
	// Interactive mode if no flags provided
	if createPlatformName == "" {
		var err error
		createPlatformName, err = promptString("Platform name", "my-platform")
		if err != nil {
			return err
		}
	}

	// Use context if org not specified
	if createPlatformOrg == "" {
		ctxMgr, err := context.NewManager()
		if err == nil {
			createPlatformOrg = ctxMgr.GetCurrentOrganization()
		} else {
			createPlatformOrg, err = promptString("Organization", "default")
			if err != nil {
				return err
			}
		}
	}

	// Generate platform YAML with environment profiles
	gen := generator.NewPlatformGenerator()
	yaml, err := gen.Generate(generator.PlatformConfig{
		Name:         createPlatformName,
		Organization: createPlatformOrg,
		Type:         createPlatformType,
		Environments: createPlatformEnvs,
	})
	if err != nil {
		return fmt.Errorf("failed to generate platform YAML: %w", err)
	}

	// Apply mode - create platform and environments directly
	if createPlatformApply {
		fmt.Printf("Creating platform '%s' in organization '%s'...\n", createPlatformName, createPlatformOrg)

		// Parse the generated YAML
		p := parser.New()
		resources, err := p.ParseString(yaml)
		if err != nil {
			return fmt.Errorf("failed to parse generated YAML: %w", err)
		}

		// Initialize components with org context
		pm := plugin.NewManager()
		st, err := store.New()
		if err != nil {
			return fmt.Errorf("failed to initialize store: %w", err)
		}
		defer st.Close()

		// Create org-filtered backend
		orgBackend := state.NewOrgFilteredBackend(st.GetBackend(), createPlatformOrg, "")
		orgStore := store.NewWithBackend(orgBackend)
		defer orgStore.Close()

		orch := orchestrator.New(pm, orgStore)

		// Apply platform and environments
		resourcesInterface := make([]interface{}, len(resources))
		for i, r := range resources {
			resourcesInterface[i] = r
		}

		if err := orch.Apply(resourcesInterface); err != nil {
			return fmt.Errorf("failed to create platform: %w", err)
		}

		fmt.Printf("\nPlatform '%s' created successfully with %d environments!\n",
			createPlatformName, len(createPlatformEnvs))
		fmt.Printf("\nEnvironments created:\n")
		for _, env := range createPlatformEnvs {
			fmt.Printf("  - %s\n", env)
		}
		fmt.Printf("\nNext steps:\n")
		fmt.Printf("  1. View platform: pf get platforms --org %s\n", createPlatformOrg)
		fmt.Printf("  2. Set environment context: pf org set-env dev\n")

		return nil
	}

	// Output mode - write/print YAML
	if createPlatformOutput != "" {
		dir := filepath.Dir(createPlatformOutput)
		if dir != "." && dir != "" {
			if err := os.MkdirAll(dir, 0755); err != nil {
				return fmt.Errorf("failed to create output directory: %w", err)
			}
		}

		if err := os.WriteFile(createPlatformOutput, []byte(yaml), 0644); err != nil {
			return fmt.Errorf("failed to write output file: %w", err)
		}

		fmt.Printf("Platform YAML written to: %s\n", createPlatformOutput)
		fmt.Printf("\nTo create the platform:\n")
		fmt.Printf("  pf apply -f %s\n", createPlatformOutput)
	} else {
		// Print to stdout
		fmt.Println(yaml)
		fmt.Fprintf(os.Stderr, "\n# Save this to a file and apply with: pf apply -f <file>\n")
		fmt.Fprintf(os.Stderr, "# Or create directly with: pf create platform --apply\n")
	}

	return nil
}
