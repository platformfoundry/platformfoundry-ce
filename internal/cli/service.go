package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/platformfoundry/platformfoundry-ce/pkg/types"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

var serviceCmd = &cobra.Command{
	Use:   "service",
	Short: "Manage services",
	Long:  `Create, list, get, update, and delete services.`,
	Example: `  pf service list
  pf service get user-api
  pf service create -f service.yaml
  pf service delete user-api
  pf service scaffold nodejs-express --name my-api`,
}

var serviceListCmd = &cobra.Command{
	Use:     "list",
	Short:   "List services",
	Long:    `List all services in the current organization.`,
	Example: `  pf service list
  pf service list --org acme-corp
  pf service list --team platform-team
  pf service list --type microservice`,
	RunE: RunServiceList,
}

var serviceGetCmd = &cobra.Command{
	Use:     "get [name]",
	Short:   "Get a service",
	Long:    `Get detailed information about a specific service.`,
	Example: `  pf service get user-api
  pf service get user-api --org acme-corp`,
	Args: cobra.ExactArgs(1),
	RunE: RunServiceGet,
}

var serviceCreateCmd = &cobra.Command{
	Use:     "create",
	Short:   "Create a service",
	Long:    `Create a new service from a YAML file.`,
	Example: `  pf service create -f service.yaml
  pf service create -f service.yaml --org acme-corp`,
	RunE: RunServiceCreate,
}

var serviceUpdateCmd = &cobra.Command{
	Use:     "update",
	Short:   "Update a service",
	Long:    `Update an existing service from a YAML file.`,
	Example: `  pf service update -f service.yaml
  pf service update -f service.yaml --org acme-corp`,
	RunE: RunServiceUpdate,
}

var serviceDeleteCmd = &cobra.Command{
	Use:     "delete [name]",
	Short:   "Delete a service",
	Long:    `Delete a service by name.`,
	Example: `  pf service delete user-api
  pf service delete user-api --org acme-corp`,
	Args: cobra.ExactArgs(1),
	RunE: RunServiceDelete,
}

var serviceScorecardCmd = &cobra.Command{
	Use:     "scorecard [name]",
	Short:   "View or calculate service scorecard",
	Long:    `View the quality scorecard for a service, showing scores across various quality dimensions.`,
	Example: `  pf service scorecard user-api
  pf service scorecard user-api --org acme-corp
  pf service scorecard user-api --calculate
  pf service scorecard user-api --format json`,
	Args: cobra.ExactArgs(1),
	RunE: runServiceScorecard,
}

var serviceScaffoldCmd = &cobra.Command{
	Use:     "scaffold [template]",
	Short:   "Scaffold a service from a template",
	Long:    `Generate a new service from a template with parameters.`,
	Example: `  pf service scaffold nodejs-express --name my-api --team platform-team
  pf service scaffold python-fastapi --name analytics-api --port 8080
  pf service scaffold --list-templates`,
	RunE: runServiceScaffold,
}

// Flags
var (
	ServiceFile    string
	serviceOrg     string
	serviceTeam    string
	serviceType    string
	serviceName    string
	serviceParams  []string
	listTemplates  bool
	outputDir      string
	calculateScore bool
	outputFormat   string
)

func init() {
	// Add subcommands
	serviceCmd.AddCommand(serviceListCmd)
	serviceCmd.AddCommand(serviceGetCmd)
	serviceCmd.AddCommand(serviceCreateCmd)
	serviceCmd.AddCommand(serviceUpdateCmd)
	serviceCmd.AddCommand(serviceDeleteCmd)
	serviceCmd.AddCommand(serviceScorecardCmd)
	serviceCmd.AddCommand(serviceScaffoldCmd)

	// List flags
	serviceListCmd.Flags().StringVar(&serviceOrg, "org", "", "Organization name")
	serviceListCmd.Flags().StringVar(&serviceTeam, "team", "", "Filter by team")
	serviceListCmd.Flags().StringVar(&serviceType, "type", "", "Filter by service type")

	// Get flags
	serviceGetCmd.Flags().StringVar(&serviceOrg, "org", "", "Organization name")

	// Create/Update flags
	serviceCreateCmd.Flags().StringVarP(&ServiceFile, "file", "f", "", "Service YAML file")
	serviceCreateCmd.Flags().StringVar(&serviceOrg, "org", "", "Organization name")
	serviceCreateCmd.MarkFlagRequired("file")

	serviceUpdateCmd.Flags().StringVarP(&ServiceFile, "file", "f", "", "Service YAML file")
	serviceUpdateCmd.Flags().StringVar(&serviceOrg, "org", "", "Organization name")
	serviceUpdateCmd.MarkFlagRequired("file")

	// Delete flags
	serviceDeleteCmd.Flags().StringVar(&serviceOrg, "org", "", "Organization name")

	// Scorecard flags
	serviceScorecardCmd.Flags().StringVar(&serviceOrg, "org", "", "Organization name")
	serviceScorecardCmd.Flags().BoolVar(&calculateScore, "calculate", false, "Calculate/recalculate scorecard")
	serviceScorecardCmd.Flags().StringVarP(&outputFormat, "format", "f", "table", "Output format (table, json, yaml)")

	// Scaffold flags
	serviceScaffoldCmd.Flags().StringVar(&serviceName, "name", "", "Service name")
	serviceScaffoldCmd.Flags().StringVar(&serviceTeam, "team", "", "Owner team")
	serviceScaffoldCmd.Flags().StringSliceVar(&serviceParams, "param", []string{}, "Template parameters (key=value)")
	serviceScaffoldCmd.Flags().BoolVar(&listTemplates, "list-templates", false, "List available templates")
	serviceScaffoldCmd.Flags().StringVarP(&outputDir, "output", "o", ".", "Output directory")
}

func RunServiceList(cmd *cobra.Command, args []string) error {
	client := getAPIClient()

	// Build query parameters
	params := make(map[string]string)
	if serviceOrg != "" {
		params["organization"] = serviceOrg
	}
	if serviceTeam != "" {
		params["team"] = serviceTeam
	}
	if serviceType != "" {
		params["type"] = serviceType
	}

	// Call API
	resp, err := client.Get("/api/services", params)
	if err != nil {
		return fmt.Errorf("failed to list services: %w", err)
	}

	var result struct {
		Success bool            `json:"success"`
		Data    []types.Service `json:"data"`
		Error   string          `json:"error"`
	}

	if err := json.Unmarshal(resp, &result); err != nil {
		return fmt.Errorf("failed to parse response: %w", err)
	}

	if !result.Success {
		return fmt.Errorf("API error: %s", result.Error)
	}

	if len(result.Data) == 0 {
		fmt.Println("No services found")
		return nil
	}

	// Print table
	fmt.Printf("%-25s %-15s %-20s %-15s %-15s\n", "NAME", "TYPE", "TEAM", "STATE", "HEALTH")
	fmt.Println(strings.Repeat("-", 90))

	for _, svc := range result.Data {
		fmt.Printf("%-25s %-15s %-20s %-15s %-15s\n",
			svc.Metadata.Name,
			svc.Spec.Type,
			svc.Spec.Owner.Team,
			svc.Status.State,
			svc.Status.Health,
		)
	}

	return nil
}

func RunServiceGet(cmd *cobra.Command, args []string) error {
	name := args[0]
	client := getAPIClient()

	// Build query parameters
	params := make(map[string]string)
	if serviceOrg != "" {
		params["organization"] = serviceOrg
	}

	// Call API
	resp, err := client.Get(fmt.Sprintf("/api/services/%s", name), params)
	if err != nil {
		return fmt.Errorf("failed to get service: %w", err)
	}

	var result struct {
		Success bool          `json:"success"`
		Data    types.Service `json:"data"`
		Error   string        `json:"error"`
	}

	if err := json.Unmarshal(resp, &result); err != nil {
		return fmt.Errorf("failed to parse response: %w", err)
	}

	if !result.Success {
		return fmt.Errorf("API error: %s", result.Error)
	}

	// Print service details
	fmt.Printf("Name:         %s\n", result.Data.Metadata.Name)
	fmt.Printf("Organization: %s\n", result.Data.Metadata.Organization)
	fmt.Printf("Type:         %s\n", result.Data.Spec.Type)
	fmt.Printf("Team:         %s\n", result.Data.Spec.Owner.Team)
	if result.Data.Spec.Owner.Email != "" {
		fmt.Printf("Contact:      %s\n", result.Data.Spec.Owner.Email)
	}
	fmt.Printf("State:        %s\n", result.Data.Status.State)
	fmt.Printf("Health:       %s\n", result.Data.Status.Health)

	if result.Data.Spec.Repository != nil {
		fmt.Printf("\nRepository:\n")
		fmt.Printf("  URL:    %s\n", result.Data.Spec.Repository.URL)
		if result.Data.Spec.Repository.Branch != "" {
			fmt.Printf("  Branch: %s\n", result.Data.Spec.Repository.Branch)
		}
	}

	if len(result.Data.Spec.Dependencies) > 0 {
		fmt.Printf("\nDependencies:\n")
		for _, dep := range result.Data.Spec.Dependencies {
			fmt.Printf("  - %s (%s)\n", dep.Name, dep.Type)
		}
	}

	if len(result.Data.Spec.Links) > 0 {
		fmt.Printf("\nLinks:\n")
		for _, link := range result.Data.Spec.Links {
			fmt.Printf("  - %s: %s\n", link.Name, link.URL)
		}
	}

	return nil
}

func RunServiceCreate(cmd *cobra.Command, args []string) error {
	// Read service file
	data, err := os.ReadFile(ServiceFile)
	if err != nil {
		return fmt.Errorf("failed to read file: %w", err)
	}

	// Parse service
	var svc types.Service
	if err := yaml.Unmarshal(data, &svc); err != nil {
		return fmt.Errorf("failed to parse YAML: %w", err)
	}

	// Override organization if specified
	if serviceOrg != "" {
		svc.Metadata.Organization = serviceOrg
	}

	// Validate service
	if err := svc.Validate(); err != nil {
		return fmt.Errorf("invalid service: %w", err)
	}

	client := getAPIClient()

	// Call API
	resp, err := client.Post("/api/services", svc)
	if err != nil {
		return fmt.Errorf("failed to create service: %w", err)
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

	fmt.Printf("✓ Service '%s' created successfully\n", svc.Metadata.Name)
	return nil
}

func RunServiceUpdate(cmd *cobra.Command, args []string) error {
	// Read service file
	data, err := os.ReadFile(ServiceFile)
	if err != nil {
		return fmt.Errorf("failed to read file: %w", err)
	}

	// Parse service
	var svc types.Service
	if err := yaml.Unmarshal(data, &svc); err != nil {
		return fmt.Errorf("failed to parse YAML: %w", err)
	}

	// Override organization if specified
	if serviceOrg != "" {
		svc.Metadata.Organization = serviceOrg
	}

	// Validate service
	if err := svc.Validate(); err != nil {
		return fmt.Errorf("invalid service: %w", err)
	}

	client := getAPIClient()

	// Call API
	resp, err := client.Put(fmt.Sprintf("/api/services/%s", svc.Metadata.Name), svc)
	if err != nil {
		return fmt.Errorf("failed to update service: %w", err)
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

	fmt.Printf("✓ Service '%s' updated successfully\n", svc.Metadata.Name)
	return nil
}

func RunServiceDelete(cmd *cobra.Command, args []string) error {
	name := args[0]
	client := getAPIClient()

	// Build query parameters
	params := make(map[string]string)
	if serviceOrg != "" {
		params["organization"] = serviceOrg
	}

	// Confirm deletion
	fmt.Printf("Are you sure you want to delete service '%s'? (yes/no): ", name)
	var confirm string
	fmt.Scanln(&confirm)
	if confirm != "yes" {
		fmt.Println("Deletion cancelled")
		return nil
	}

	// Call API
	resp, err := client.Delete(fmt.Sprintf("/api/services/%s", name), params)
	if err != nil {
		return fmt.Errorf("failed to delete service: %w", err)
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

	fmt.Printf("✓ Service '%s' deleted successfully\n", name)
	return nil
}

func runServiceScorecard(cmd *cobra.Command, args []string) error {
	name := args[0]
	client := getAPIClient()

	// Build query parameters
	params := make(map[string]string)
	if serviceOrg != "" {
		params["organization"] = serviceOrg
	}

	var resp []byte
	var err error

	if calculateScore {
		// POST to calculate/recalculate scorecard
		resp, err = client.Post(fmt.Sprintf("/api/services/%s/scorecard", name), nil)
		if err != nil {
			return fmt.Errorf("failed to calculate scorecard: %w", err)
		}
	} else {
		// GET existing scorecard
		resp, err = client.Get(fmt.Sprintf("/api/services/%s/scorecard", name), params)
		if err != nil {
			return fmt.Errorf("failed to get scorecard: %w", err)
		}
	}

	var result struct {
		Success bool                   `json:"success"`
		Data    types.ServiceScorecard `json:"data"`
		Error   string                 `json:"error"`
	}

	if err := json.Unmarshal(resp, &result); err != nil {
		return fmt.Errorf("failed to parse response: %w", err)
	}

	if !result.Success {
		return fmt.Errorf("API error: %s", result.Error)
	}

	// Handle output format
	switch outputFormat {
	case "json":
		jsonData, err := json.MarshalIndent(result.Data, "", "  ")
		if err != nil {
			return fmt.Errorf("failed to marshal JSON: %w", err)
		}
		fmt.Println(string(jsonData))

	case "yaml":
		yamlData, err := yaml.Marshal(result.Data)
		if err != nil {
			return fmt.Errorf("failed to marshal YAML: %w", err)
		}
		fmt.Println(string(yamlData))

	case "table":
		fallthrough
	default:
		// Print scorecard summary
		fmt.Printf("Service Scorecard: %s\n", result.Data.Metadata.Name)
		fmt.Println(strings.Repeat("=", 70))
		fmt.Printf("Grade:        %s\n", result.Data.Status.Grade)
		fmt.Printf("Score:        %d/100\n", result.Data.Status.Score)
		fmt.Printf("Passed:       %d\n", result.Data.Status.PassedChecks)
		fmt.Printf("Failed:       %d\n", result.Data.Status.FailedChecks)
		fmt.Printf("Total Checks: %d\n", result.Data.Status.TotalChecks)
		fmt.Printf("Evaluated:    %s\n", result.Data.Status.EvaluatedAt.Format("2006-01-02 15:04:05"))

		// Group checks by category
		checksByCategory := make(map[types.CheckCategory][]types.Check)
		for _, check := range result.Data.Spec.Checks {
			checksByCategory[check.Category] = append(checksByCategory[check.Category], check)
		}

		// Print checks by category
		fmt.Printf("\n%-30s %-10s %-8s %-8s %s\n", "CHECK", "CATEGORY", "STATUS", "SCORE", "MESSAGE")
		fmt.Println(strings.Repeat("-", 100))

		for category, checks := range checksByCategory {
			for _, check := range checks {
				statusSymbol := "✓"
				if check.Status == types.CheckStatusFailed {
					statusSymbol = "✗"
				} else if check.Status == types.CheckStatusWarning {
					statusSymbol = "⚠"
				} else if check.Status == types.CheckStatusSkipped {
					statusSymbol = "○"
				}

				fmt.Printf("%-30s %-10s %-8s %-8d %s\n",
					check.Name,
					category,
					fmt.Sprintf("%s %s", statusSymbol, check.Status),
					check.Score,
					check.Message,
				)

				if check.Details != "" && (check.Status == types.CheckStatusFailed || check.Status == types.CheckStatusWarning) {
					fmt.Printf("  Details: %s\n", check.Details)
				}
			}
		}
	}

	return nil
}

func runServiceScaffold(cmd *cobra.Command, args []string) error {
	client := getAPIClient()

	// List templates if requested
	if listTemplates {
		resp, err := client.Get("/api/templates", nil)
		if err != nil {
			return fmt.Errorf("failed to list templates: %w", err)
		}

		var result struct {
			Success bool                     `json:"success"`
			Data    []types.ServiceTemplate  `json:"data"`
			Error   string                   `json:"error"`
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

		fmt.Printf("%-25s %-15s %s\n", "NAME", "CATEGORY", "DESCRIPTION")
		fmt.Println(strings.Repeat("-", 80))

		for _, tmpl := range result.Data {
			desc := tmpl.Spec.Description
			if len(desc) > 40 {
				desc = desc[:37] + "..."
			}
			fmt.Printf("%-25s %-15s %s\n", tmpl.Metadata.Name, tmpl.Spec.Category, desc)
		}

		return nil
	}

	// Require template name
	if len(args) == 0 {
		return fmt.Errorf("template name required (use --list-templates to see available templates)")
	}

	templateName := args[0]

	// Build parameters
	params := make(map[string]interface{})
	if serviceName != "" {
		params["name"] = serviceName
	}
	if serviceTeam != "" {
		params["team"] = serviceTeam
	}

	// Parse additional parameters
	for _, param := range serviceParams {
		parts := strings.SplitN(param, "=", 2)
		if len(parts) != 2 {
			return fmt.Errorf("invalid parameter format: %s (expected key=value)", param)
		}
		params[parts[0]] = parts[1]
	}

	// Call API to instantiate template
	req := map[string]interface{}{
		"parameters": params,
	}

	resp, err := client.Post(fmt.Sprintf("/api/templates/%s/instantiate", templateName), req)
	if err != nil {
		return fmt.Errorf("failed to instantiate template: %w", err)
	}

	var result struct {
		Success bool          `json:"success"`
		Data    types.Service `json:"data"`
		Error   string        `json:"error"`
	}

	if err := json.Unmarshal(resp, &result); err != nil {
		return fmt.Errorf("failed to parse response: %w", err)
	}

	if !result.Success {
		return fmt.Errorf("API error: %s", result.Error)
	}

	// Write service YAML to output directory
	outputFile := fmt.Sprintf("%s/%s-service.yaml", outputDir, result.Data.Metadata.Name)
	yamlData, err := yaml.Marshal(result.Data)
	if err != nil {
		return fmt.Errorf("failed to marshal service: %w", err)
	}

	if err := os.WriteFile(outputFile, yamlData, 0644); err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}

	fmt.Printf("✓ Service scaffolded successfully\n")
	fmt.Printf("  Service: %s\n", result.Data.Metadata.Name)
	fmt.Printf("  Type:    %s\n", result.Data.Spec.Type)
	fmt.Printf("  Output:  %s\n", outputFile)

	return nil
}

// getAPIClient returns an API client loaded from configuration
func getAPIClient() *APIClient {
	config, err := LoadAPIClientConfig()
	if err != nil {
		// Fallback to environment variables on error
		return NewAPIClient(os.Getenv("PF_API_URL"), os.Getenv("PF_API_TOKEN"))
	}
	return NewAPIClientFromConfig(config)
}