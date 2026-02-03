package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/platformfoundry/pf-ce/pkg/types"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

var templateCmd = &cobra.Command{
	Use:   "template",
	Short: "Manage service templates",
	Long:  `Create, list, get, and delete service templates.`,
	Example: `  pf template list
  pf template get nodejs-express
  pf template create -f template.yaml
  pf template delete nodejs-express`,
}

var templateListCmd = &cobra.Command{
	Use:   "list",
	Short: "List templates",
	Long:  `List all service templates in the current organization.`,
	Example: `  pf template list
  pf template list --org acme-corp
  pf template list --category backend`,
	RunE: runTemplateList,
}

var templateGetCmd = &cobra.Command{
	Use:   "get [name]",
	Short: "Get a template",
	Long:  `Get detailed information about a specific template.`,
	Example: `  pf template get nodejs-express
  pf template get nodejs-express --org acme-corp`,
	Args: cobra.ExactArgs(1),
	RunE: runTemplateGet,
}

var templateCreateCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a template",
	Long:  `Create a new service template from a YAML file.`,
	Example: `  pf template create -f template.yaml
  pf template create -f template.yaml --org acme-corp`,
	RunE: runTemplateCreate,
}

var templateDeleteCmd = &cobra.Command{
	Use:   "delete [name]",
	Short: "Delete a template",
	Long:  `Delete a template by name.`,
	Example: `  pf template delete nodejs-express
  pf template delete nodejs-express --org acme-corp`,
	Args: cobra.ExactArgs(1),
	RunE: runTemplateDelete,
}

// Flags
var (
	templateFile     string
	templateOrg      string
	templateCategory string
)

func init() {
	// Add subcommands
	templateCmd.AddCommand(templateListCmd)
	templateCmd.AddCommand(templateGetCmd)
	templateCmd.AddCommand(templateCreateCmd)
	templateCmd.AddCommand(templateDeleteCmd)

	// List flags
	templateListCmd.Flags().StringVar(&templateOrg, "org", "", "Organization name")
	templateListCmd.Flags().StringVar(&templateCategory, "category", "", "Filter by category")

	// Get flags
	templateGetCmd.Flags().StringVar(&templateOrg, "org", "", "Organization name")

	// Create flags
	templateCreateCmd.Flags().StringVarP(&templateFile, "file", "f", "", "Template YAML file")
	templateCreateCmd.Flags().StringVar(&templateOrg, "org", "", "Organization name")
	templateCreateCmd.MarkFlagRequired("file")

	// Delete flags
	templateDeleteCmd.Flags().StringVar(&templateOrg, "org", "", "Organization name")
}

func runTemplateList(cmd *cobra.Command, args []string) error {
	client := getAPIClient()

	// Build query parameters
	params := make(map[string]string)
	if templateOrg != "" {
		params["organization"] = templateOrg
	}
	if templateCategory != "" {
		params["category"] = templateCategory
	}

	// Call API
	resp, err := client.Get("/api/templates", params)
	if err != nil {
		return fmt.Errorf("failed to list templates: %w", err)
	}

	var result struct {
		Success bool                    `json:"success"`
		Data    []types.ServiceTemplate `json:"data"`
		Error   string                  `json:"error"`
	}

	if err := json.Unmarshal(resp, &result); err != nil {
		return fmt.Errorf("failed to parse response: %w", err)
	}

	if !result.Success {
		return fmt.Errorf("API error: %s", result.Error)
	}

	if len(result.Data) == 0 {
		fmt.Println("No templates found")
		return nil
	}

	// Print table
	fmt.Printf("%-25s %-15s %-10s %s\n", "NAME", "CATEGORY", "PARAMS", "DESCRIPTION")
	fmt.Println(strings.Repeat("-", 90))

	for _, tmpl := range result.Data {
		desc := tmpl.Spec.Description
		if len(desc) > 35 {
			desc = desc[:32] + "..."
		}
		fmt.Printf("%-25s %-15s %-10d %s\n",
			tmpl.Metadata.Name,
			tmpl.Spec.Category,
			len(tmpl.Spec.Parameters),
			desc,
		)
	}

	return nil
}

func runTemplateGet(cmd *cobra.Command, args []string) error {
	name := args[0]
	client := getAPIClient()

	// Build query parameters
	params := make(map[string]string)
	if templateOrg != "" {
		params["organization"] = templateOrg
	}

	// Call API
	resp, err := client.Get(fmt.Sprintf("/api/templates/%s", name), params)
	if err != nil {
		return fmt.Errorf("failed to get template: %w", err)
	}

	var result struct {
		Success bool                   `json:"success"`
		Data    types.ServiceTemplate  `json:"data"`
		Error   string                 `json:"error"`
	}

	if err := json.Unmarshal(resp, &result); err != nil {
		return fmt.Errorf("failed to parse response: %w", err)
	}

	if !result.Success {
		return fmt.Errorf("API error: %s", result.Error)
	}

	// Print template details
	fmt.Printf("Name:        %s\n", result.Data.Metadata.Name)
	fmt.Printf("Display:     %s\n", result.Data.Spec.DisplayName)
	fmt.Printf("Category:    %s\n", result.Data.Spec.Category)
	fmt.Printf("Description: %s\n", result.Data.Spec.Description)

	if len(result.Data.Spec.Tags) > 0 {
		fmt.Printf("Tags:        %s\n", strings.Join(result.Data.Spec.Tags, ", "))
	}

	if len(result.Data.Spec.Parameters) > 0 {
		fmt.Printf("\nParameters:\n")
		for _, param := range result.Data.Spec.Parameters {
			required := ""
			if param.Required {
				required = " (required)"
			}
			fmt.Printf("  - %s (%s)%s\n", param.Name, param.Type, required)
			if param.Description != "" {
				fmt.Printf("    %s\n", param.Description)
			}
		}
	}

	if len(result.Data.Spec.Files) > 0 {
		fmt.Printf("\nGenerated Files: %d\n", len(result.Data.Spec.Files))
	}

	return nil
}

func runTemplateCreate(cmd *cobra.Command, args []string) error {
	// Read template file
	data, err := os.ReadFile(templateFile)
	if err != nil {
		return fmt.Errorf("failed to read file: %w", err)
	}

	// Parse template
	var tmpl types.ServiceTemplate
	if err := yaml.Unmarshal(data, &tmpl); err != nil {
		return fmt.Errorf("failed to parse YAML: %w", err)
	}

	// Override organization if specified
	if templateOrg != "" {
		tmpl.Metadata.Organization = templateOrg
	}

	// Validate template
	if err := tmpl.Validate(); err != nil {
		return fmt.Errorf("invalid template: %w", err)
	}

	client := getAPIClient()

	// Call API
	resp, err := client.Post("/api/templates", tmpl)
	if err != nil {
		return fmt.Errorf("failed to create template: %w", err)
	}

	var result struct {
		Success bool   `json:"success"`
		Message string `json:"message"`
		Error   string `json:"error"`
	}

	if err := json.Unmarshal(resp, &result); err != nil {
		return fmt.Errorf("failed to parse response: %w", err)
	}

	if !result.Success {
		return fmt.Errorf("API error: %s", result.Error)
	}

	fmt.Printf("✓ Template '%s' created successfully\n", tmpl.Metadata.Name)
	return nil
}

func runTemplateDelete(cmd *cobra.Command, args []string) error {
	name := args[0]
	client := getAPIClient()

	// Build query parameters
	params := make(map[string]string)
	if templateOrg != "" {
		params["organization"] = templateOrg
	}

	// Confirm deletion
	fmt.Printf("Are you sure you want to delete template '%s'? (yes/no): ", name)
	var confirm string
	fmt.Scanln(&confirm)
	if confirm != "yes" {
		fmt.Println("Deletion cancelled")
		return nil
	}

	// Call API
	resp, err := client.Delete(fmt.Sprintf("/api/templates/%s", name), params)
	if err != nil {
		return fmt.Errorf("failed to delete template: %w", err)
	}

	var result struct {
		Success bool   `json:"success"`
		Message string `json:"message"`
		Error   string `json:"error"`
	}

	if err := json.Unmarshal(resp, &result); err != nil {
		return fmt.Errorf("failed to parse response: %w", err)
	}

	if !result.Success {
		return fmt.Errorf("API error: %s", result.Error)
	}

	fmt.Printf("✓ Template '%s' deleted successfully\n", name)
	return nil
}
