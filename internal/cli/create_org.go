package cli

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/platformfoundry/pf-ce/internal/orchestrator"
	"github.com/platformfoundry/pf-ce/internal/parser"
	"github.com/platformfoundry/pf-ce/internal/plugin"
	"github.com/platformfoundry/pf-ce/internal/store"
	"github.com/spf13/cobra"
)

var (
	createOrgName        string
	createOrgDisplayName string
	createOrgOwner       string
	createOrgEmail       string
	createOrgApply       bool
	createOrgOutput      string
)

var createOrgCmd = &cobra.Command{
	Use:   "organization",
	Short: "Create an organization",
	Long:  `Generate an organization YAML or create it directly with --apply.`,
	Example: `  pf create organization
  pf create organization --name acme --display-name "Acme Corp" --owner john
  pf create organization --name acme --apply
  pf create organization --output org.yaml`,
	RunE: runCreateOrg,
}

func init() {
	createOrgCmd.Flags().StringVar(&createOrgName, "name", "", "Organization name")
	createOrgCmd.Flags().StringVar(&createOrgDisplayName, "display-name", "", "Organization display name")
	createOrgCmd.Flags().StringVar(&createOrgOwner, "owner", "", "Organization owner username")
	createOrgCmd.Flags().StringVar(&createOrgEmail, "email", "", "Organization contact email")
	createOrgCmd.Flags().BoolVar(&createOrgApply, "apply", false, "Apply the organization directly")
	createOrgCmd.Flags().StringVarP(&createOrgOutput, "output", "o", "", "Output file (default: stdout)")
}

func runCreateOrg(cmd *cobra.Command, args []string) error {
	// Interactive mode if no flags provided
	if createOrgName == "" {
		var err error
		createOrgName, err = promptString("Organization name", "my-org")
		if err != nil {
			return err
		}
	}

	if createOrgDisplayName == "" {
		var err error
		createOrgDisplayName, err = promptString("Display name", createOrgName)
		if err != nil {
			return err
		}
	}

	if createOrgOwner == "" {
		var err error
		createOrgOwner, err = promptString("Owner username", "admin")
		if err != nil {
			return err
		}
	}

	// Generate organization YAML
	yaml := generateOrgYAML(createOrgName, createOrgDisplayName, createOrgOwner, createOrgEmail)

	// Apply mode - create organization directly
	if createOrgApply {
		fmt.Println("Creating organization...")

		// Parse the generated YAML
		p := parser.New()
		resources, err := p.ParseString(yaml)
		if err != nil {
			return fmt.Errorf("failed to parse generated YAML: %w", err)
		}

		// Initialize components
		pm := plugin.NewManager()
		st, err := store.New()
		if err != nil {
			return fmt.Errorf("failed to initialize store: %w", err)
		}
		defer st.Close()

		orch := orchestrator.New(pm, st)

		// Apply organization
		resourcesInterface := make([]interface{}, len(resources))
		for i, r := range resources {
			resourcesInterface[i] = r
		}

		if err := orch.Apply(resourcesInterface); err != nil {
			return fmt.Errorf("failed to create organization: %w", err)
		}

		fmt.Printf("\nOrganization '%s' created successfully!\n", createOrgName)
		fmt.Printf("\nNext steps:\n")
		fmt.Printf("  1. Set as current organization: pf org set %s\n", createOrgName)
		fmt.Printf("  2. Create a platform: pf create platform --org %s\n", createOrgName)

		return nil
	}

	// Output mode - write/print YAML
	if createOrgOutput != "" {
		// Ensure directory exists
		dir := filepath.Dir(createOrgOutput)
		if dir != "." && dir != "" {
			if err := os.MkdirAll(dir, 0755); err != nil {
				return fmt.Errorf("failed to create output directory: %w", err)
			}
		}

		if err := os.WriteFile(createOrgOutput, []byte(yaml), 0644); err != nil {
			return fmt.Errorf("failed to write output file: %w", err)
		}

		fmt.Printf("Organization YAML written to: %s\n", createOrgOutput)
		fmt.Printf("\nTo create the organization:\n")
		fmt.Printf("  pf apply -f %s\n", createOrgOutput)
	} else {
		// Print to stdout
		fmt.Println(yaml)
		fmt.Fprintf(os.Stderr, "\n# Save this to a file and apply with: pf apply -f <file>\n")
		fmt.Fprintf(os.Stderr, "# Or create directly with: pf create organization --apply\n")
	}

	return nil
}

// promptString prompts the user for a string value
func promptString(prompt, defaultValue string) (string, error) {
	fmt.Printf("%s [%s]: ", prompt, defaultValue)
	reader := bufio.NewReader(os.Stdin)
	value, err := reader.ReadString('\n')
	if err != nil {
		return "", err
	}

	value = strings.TrimSpace(value)
	if value == "" {
		return defaultValue, nil
	}

	return value, nil
}

// generateOrgYAML generates organization YAML inline
func generateOrgYAML(name, displayName, owner, email string) string {
	var sb strings.Builder
	sb.WriteString("apiVersion: platformfoundry.io/v1\n")
	sb.WriteString("kind: Organization\n")
	sb.WriteString("metadata:\n")
	sb.WriteString(fmt.Sprintf("  name: %s\n", name))
	if displayName != "" {
		sb.WriteString(fmt.Sprintf("  displayName: %s\n", displayName))
	}
	sb.WriteString("spec:\n")
	if owner != "" {
		sb.WriteString(fmt.Sprintf("  owner: %s\n", owner))
	}
	if email != "" {
		sb.WriteString(fmt.Sprintf("  contactEmail: %s\n", email))
	}
	return sb.String()
}
