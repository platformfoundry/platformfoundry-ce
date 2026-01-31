package cli

import (
	"os"
	"strings"

	"github.com/spf13/cobra"
)

// File completion for YAML files
func yamlFileCompletion(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	return []string{"yaml", "yml"}, cobra.ShellCompDirectiveFilterFileExt
}

// Completion for resource types
func resourceTypeCompletion(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	return []string{
		"all\tAll resources",
		"platforms\tPlatform resources",
		"clusters\tCluster resources",
		"pipelines\tPipeline resources",
		"observability\tObservability resources",
	}, cobra.ShellCompDirectiveNoFileComp
}

// Completion for compliance frameworks
func complianceFrameworkCompletion(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	return []string{
		"SOC2\tSOC 2 Type II compliance",
		"HIPAA\tHIPAA compliance",
		"PCI-DSS\tPCI-DSS compliance",
		"GDPR\tGDPR compliance",
		"ISO27001\tISO 27001 compliance",
		"NIST\tNIST framework",
	}, cobra.ShellCompDirectiveNoFileComp
}

// Completion for cloud providers
func cloudProviderCompletion(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	return []string{
		"aws\tAmazon Web Services",
		"gcp\tGoogle Cloud Platform",
		"azure\tMicrosoft Azure",
	}, cobra.ShellCompDirectiveNoFileComp
}

// Completion for cloud regions (AWS)
func cloudRegionCompletion(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	// Get provider from flags if available
	provider, _ := cmd.Flags().GetString("provider")

	switch strings.ToLower(provider) {
	case "aws":
		return []string{
			"us-east-1\tUS East (N. Virginia)",
			"us-east-2\tUS East (Ohio)",
			"us-west-1\tUS West (N. California)",
			"us-west-2\tUS West (Oregon)",
			"eu-west-1\tEU (Ireland)",
			"eu-central-1\tEU (Frankfurt)",
			"ap-southeast-1\tAsia Pacific (Singapore)",
			"ap-northeast-1\tAsia Pacific (Tokyo)",
		}, cobra.ShellCompDirectiveNoFileComp

	case "gcp":
		return []string{
			"us-central1\tUS Central (Iowa)",
			"us-east1\tUS East (South Carolina)",
			"us-west1\tUS West (Oregon)",
			"europe-west1\tEurope West (Belgium)",
			"asia-east1\tAsia East (Taiwan)",
		}, cobra.ShellCompDirectiveNoFileComp

	case "azure":
		return []string{
			"eastus\tEast US",
			"westus\tWest US",
			"northeurope\tNorth Europe",
			"westeurope\tWest Europe",
			"southeastasia\tSoutheast Asia",
		}, cobra.ShellCompDirectiveNoFileComp

	default:
		return []string{"us-east-1", "us-west-2", "eu-west-1"}, cobra.ShellCompDirectiveNoFileComp
	}
}

// Completion for SSO providers
func ssoProviderCompletion(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	return []string{
		"google\tGoogle OAuth",
		"github\tGitHub OAuth",
		"azuread\tAzure AD OAuth",
		"okta\tOkta OAuth",
	}, cobra.ShellCompDirectiveNoFileComp
}

// Completion for backup IDs
func backupIDCompletion(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	backupsDir := "/var/lib/platformfoundry/backups"
	if bd := os.Getenv("PF_BACKUPS_DIR"); bd != "" {
		backupsDir = bd
	}

	entries, err := os.ReadDir(backupsDir)
	if err != nil {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}

	var backups []string
	for _, entry := range entries {
		if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".tar.gz") {
			backups = append(backups, entry.Name())
		}
	}

	return backups, cobra.ShellCompDirectiveNoFileComp
}
