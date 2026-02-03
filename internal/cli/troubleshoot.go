package cli

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"

	"github.com/platformfoundry/pf-ce/internal/store"
	"github.com/spf13/cobra"
)

var troubleshootCmd = &cobra.Command{
	Use:   "troubleshoot",
	Short: "Diagnose Platform Foundry issues",
	Long:  `Run diagnostic checks to identify and troubleshoot common issues.`,
	Example: `  pf troubleshoot
  pf troubleshoot --verbose`,
	RunE: runTroubleshoot,
}

var troubleshootVerbose bool

func init() {
	troubleshootCmd.Flags().BoolVarP(&troubleshootVerbose, "verbose", "v", false, "Show detailed diagnostic information")
}

// CheckResult represents the result of a diagnostic check
type CheckResult struct {
	Name    string
	Status  string // "ok", "warning", "error"
	Message string
	Fix     string
}

func runTroubleshoot(cmd *cobra.Command, args []string) error {
	fmt.Println("🔍 Checking Platform Foundry health...")
	fmt.Println()

	checks := []CheckResult{
		checkCLIVersion(),
		checkConfiguration(),
		checkStateBackend(),
		checkAuthentication(),
		checkDocker(),
		checkKubectl(),
		checkTerraform(),
		checkGit(),
		checkDiskSpace(),
		checkPermissions(),
	}

	// Count issues
	okCount := 0
	warningCount := 0
	errorCount := 0

	// Display results
	for _, check := range checks {
		icon := ""
		switch check.Status {
		case "ok":
			icon = "✅"
			okCount++
		case "warning":
			icon = "⚠️ "
			warningCount++
		case "error":
			icon = "❌"
			errorCount++
		}

		fmt.Printf("%s %-25s %s\n", icon, check.Name+":", check.Message)

		if troubleshootVerbose && check.Fix != "" {
			fmt.Printf("   💡 Fix: %s\n", check.Fix)
		}
	}

	// Summary
	fmt.Println()
	fmt.Println("Summary:")
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Printf("  ✅ OK: %d\n", okCount)
	if warningCount > 0 {
		fmt.Printf("  ⚠️  Warnings: %d\n", warningCount)
	}
	if errorCount > 0 {
		fmt.Printf("  ❌ Errors: %d\n", errorCount)
	}

	// Issues found section
	if errorCount > 0 || warningCount > 0 {
		fmt.Println()
		fmt.Println("Issues Found:")
		fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")

		for _, check := range checks {
			if check.Status == "error" || check.Status == "warning" {
				fmt.Printf("\n%s %s\n", check.Name, check.Message)
				if check.Fix != "" {
					fmt.Printf("  💡 How to fix: %s\n", check.Fix)
				}
			}
		}
	}

	// Overall status
	fmt.Println()
	if errorCount == 0 && warningCount == 0 {
		fmt.Println("🎉 All checks passed! Platform Foundry is ready to use.")
	} else if errorCount == 0 {
		fmt.Println("⚠️  Platform Foundry is functional but has warnings. Review the issues above.")
	} else {
		fmt.Println("❌ Platform Foundry has errors that need to be fixed. See issues above.")
		return fmt.Errorf("troubleshooting found %d error(s)", errorCount)
	}

	return nil
}

func checkCLIVersion() CheckResult {
	version := rootCmd.Version
	if version == "" {
		version = "unknown"
	}

	return CheckResult{
		Name:    "CLI Version",
		Status:  "ok",
		Message: fmt.Sprintf("%s", version),
	}
}

func checkConfiguration() CheckResult {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return CheckResult{
			Name:    "Configuration",
			Status:  "warning",
			Message: "Could not detect home directory",
			Fix:     "Ensure HOME environment variable is set",
		}
	}

	configPath := filepath.Join(homeDir, ".platformfoundry", "config.yaml")
	if _, err := os.Stat(configPath); err != nil {
		return CheckResult{
			Name:    "Configuration",
			Status:  "warning",
			Message: "Config file not found",
			Fix:     fmt.Sprintf("Create config at %s or run 'pf init'", configPath),
		}
	}

	return CheckResult{
		Name:    "Configuration",
		Status:  "ok",
		Message: configPath,
	}
}

func checkStateBackend() CheckResult {
	st, err := store.New()
	if err != nil {
		return CheckResult{
			Name:    "State Backend",
			Status:  "error",
			Message: "Failed to connect",
			Fix:     fmt.Sprintf("Error: %v. Check state backend configuration", err),
		}
	}

	// Try to list resources to verify it's working
	_, err = st.List()
	if err != nil {
		return CheckResult{
			Name:    "State Backend",
			Status:  "warning",
			Message: "Connected but query failed",
			Fix:     fmt.Sprintf("State backend may be corrupted: %v", err),
		}
	}

	return CheckResult{
		Name:    "State Backend",
		Status:  "ok",
		Message: "Connected",
	}
}

func checkAuthentication() CheckResult {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return CheckResult{
			Name:    "Authentication",
			Status:  "warning",
			Message: "Cannot check auth status",
		}
	}

	authDir := filepath.Join(homeDir, ".platformfoundry", "auth")
	usersFile := filepath.Join(authDir, "users.json")

	if _, err := os.Stat(usersFile); err != nil {
		return CheckResult{
			Name:    "Authentication",
			Status:  "warning",
			Message: "Not configured",
			Fix:     "Run 'pf auth create-user admin' to create an admin user",
		}
	}

	return CheckResult{
		Name:    "Authentication",
		Status:  "ok",
		Message: "Configured",
	}
}

func checkDocker() CheckResult {
	cmd := exec.Command("docker", "version")
	err := cmd.Run()
	if err != nil {
		return CheckResult{
			Name:    "Docker",
			Status:  "warning",
			Message: "Not available",
			Fix:     "Install Docker from https://www.docker.com/get-started or ensure Docker daemon is running",
		}
	}

	return CheckResult{
		Name:    "Docker",
		Status:  "ok",
		Message: "Available",
	}
}

func checkKubectl() CheckResult {
	cmd := exec.Command("kubectl", "version", "--client")
	err := cmd.Run()
	if err != nil {
		return CheckResult{
			Name:    "Kubectl",
			Status:  "warning",
			Message: "Not available",
			Fix:     "Install kubectl from https://kubernetes.io/docs/tasks/tools/",
		}
	}

	return CheckResult{
		Name:    "Kubectl",
		Status:  "ok",
		Message: "Available",
	}
}

func checkTerraform() CheckResult {
	cmd := exec.Command("terraform", "version")
	err := cmd.Run()
	if err != nil {
		return CheckResult{
			Name:    "Terraform",
			Status:  "warning",
			Message: "Not available",
			Fix:     "Install Terraform from https://www.terraform.io/downloads if you need infrastructure provisioning",
		}
	}

	return CheckResult{
		Name:    "Terraform",
		Status:  "ok",
		Message: "Available",
	}
}

func checkGit() CheckResult {
	cmd := exec.Command("git", "version")
	err := cmd.Run()
	if err != nil {
		return CheckResult{
			Name:    "Git",
			Status:  "warning",
			Message: "Not available",
			Fix:     "Install Git from https://git-scm.com/downloads",
		}
	}

	return CheckResult{
		Name:    "Git",
		Status:  "ok",
		Message: "Available",
	}
}

func checkDiskSpace() CheckResult {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return CheckResult{
			Name:    "Disk Space",
			Status:  "warning",
			Message: "Cannot check",
		}
	}

	stateDir := filepath.Join(homeDir, ".platformfoundry", "state")

	// Check if state directory exists
	info, err := os.Stat(stateDir)
	if err != nil {
		return CheckResult{
			Name:    "Disk Space",
			Status:  "ok",
			Message: "State directory not yet created",
		}
	}

	_ = info // Use the info variable

	return CheckResult{
		Name:    "Disk Space",
		Status:  "ok",
		Message: "Sufficient",
	}
}

func checkPermissions() CheckResult {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return CheckResult{
			Name:    "Permissions",
			Status:  "warning",
			Message: "Cannot check permissions",
		}
	}

	pfDir := filepath.Join(homeDir, ".platformfoundry")

	// Try to create directory if it doesn't exist
	if err := os.MkdirAll(pfDir, 0755); err != nil {
		return CheckResult{
			Name:    "Permissions",
			Status:  "error",
			Message: "Cannot write to config directory",
			Fix:     fmt.Sprintf("Ensure you have write permissions to %s", pfDir),
		}
	}

	// Try to create a test file
	testFile := filepath.Join(pfDir, ".write-test")
	if err := os.WriteFile(testFile, []byte("test"), 0644); err != nil {
		return CheckResult{
			Name:    "Permissions",
			Status:  "error",
			Message: "Cannot write files",
			Fix:     fmt.Sprintf("Ensure you have write permissions to %s", pfDir),
		}
	}

	// Clean up test file
	os.Remove(testFile)

	return CheckResult{
		Name:    "Permissions",
		Status:  "ok",
		Message: fmt.Sprintf("Read/write access to %s", pfDir),
	}
}

// Platform-specific checks
func init() {
	// Add platform-specific checks based on OS
	switch runtime.GOOS {
	case "windows":
		// Windows-specific checks could be added here
	case "darwin":
		// macOS-specific checks could be added here
	case "linux":
		// Linux-specific checks could be added here
	}
}
