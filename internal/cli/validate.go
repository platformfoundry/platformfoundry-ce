package cli

import (
	"fmt"

	"github.com/platformfoundry/pf-ce/internal/parser"
	"github.com/spf13/cobra"
)

var validateFile string

var validateCmd = &cobra.Command{
	Use:   "validate -f <file>",
	Short: "Validate a YAML file",
	Long:  `Parse and validate a YAML file without applying resources.`,
	Example: `  pf validate -f platform.yaml`,
	RunE: runValidate,
}

func init() {
	validateCmd.Flags().StringVarP(&validateFile, "file", "f", "", "YAML file to validate (required)")
	validateCmd.MarkFlagRequired("file")
}

func runValidate(cmd *cobra.Command, args []string) error {
	// Parse YAML file
	p := parser.New()
	resources, err := p.ParseFile(validateFile)
	if err != nil {
		return fmt.Errorf("validation failed: %w", err)
	}

	fmt.Printf("✓ File is valid\n")
	fmt.Printf("✓ Found %d resource(s)\n\n", len(resources))

	// Print resources
	for _, r := range resources {
		provider := r.Spec["provider"]
		fmt.Printf("  - %s/%s (provider: %v)\n", r.Kind, r.Metadata.Name, provider)
	}

	return nil
}
