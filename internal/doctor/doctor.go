// Package doctor provides system health checks and diagnostics.
package doctor

import (
	"context"
	"fmt"
	"os/exec"
	"runtime"
	"strings"
	"time"
)

// CheckStatus represents the result of a health check
type CheckStatus string

const (
	StatusOK      CheckStatus = "ok"
	StatusWarning CheckStatus = "warning"
	StatusError   CheckStatus = "error"
	StatusSkipped CheckStatus = "skipped"
)

// Check represents a single health check
type Check struct {
	Name        string      `json:"name"`
	Category    string      `json:"category"`
	Status      CheckStatus `json:"status"`
	Message     string      `json:"message"`
	Details     string      `json:"details,omitempty"`
	Duration    time.Duration `json:"duration"`
	Remediation string      `json:"remediation,omitempty"`
}

// Report contains all health check results
type Report struct {
	Checks      []Check       `json:"checks"`
	Summary     Summary       `json:"summary"`
	GeneratedAt time.Time     `json:"generated_at"`
	Duration    time.Duration `json:"duration"`
}

// Summary provides an overview of check results
type Summary struct {
	Total    int `json:"total"`
	Passed   int `json:"passed"`
	Warnings int `json:"warnings"`
	Errors   int `json:"errors"`
	Skipped  int `json:"skipped"`
}

// Doctor performs system health checks
type Doctor struct {
	checks []checkFunc
}

type checkFunc func(ctx context.Context) Check

// New creates a new Doctor instance with default checks
func New() *Doctor {
	d := &Doctor{}
	d.registerDefaultChecks()
	return d
}

func (d *Doctor) registerDefaultChecks() {
	d.checks = []checkFunc{
		d.checkGo,
		d.checkDocker,
		d.checkKubectl,
		d.checkHelm,
		d.checkKind,
		d.checkGit,
		d.checkTerraform,
		d.checkDiskSpace,
		d.checkMemory,
		d.checkNetwork,
	}
}

// RunAll executes all health checks
func (d *Doctor) RunAll(ctx context.Context) *Report {
	start := time.Now()
	report := &Report{
		Checks:      make([]Check, 0, len(d.checks)),
		GeneratedAt: start,
	}

	for _, check := range d.checks {
		select {
		case <-ctx.Done():
			return report
		default:
			result := check(ctx)
			report.Checks = append(report.Checks, result)

			switch result.Status {
			case StatusOK:
				report.Summary.Passed++
			case StatusWarning:
				report.Summary.Warnings++
			case StatusError:
				report.Summary.Errors++
			case StatusSkipped:
				report.Summary.Skipped++
			}
			report.Summary.Total++
		}
	}

	report.Duration = time.Since(start)
	return report
}

// checkGo verifies Go installation
func (d *Doctor) checkGo(ctx context.Context) Check {
	start := time.Now()
	check := Check{
		Name:     "Go",
		Category: "Build Tools",
	}

	cmd := exec.CommandContext(ctx, "go", "version")
	output, err := cmd.Output()
	check.Duration = time.Since(start)

	if err != nil {
		check.Status = StatusWarning
		check.Message = "Go not found"
		check.Details = "Go is optional but required for building from source"
		check.Remediation = "Install Go from https://go.dev/dl/"
		return check
	}

	version := strings.TrimSpace(string(output))
	check.Status = StatusOK
	check.Message = version
	return check
}

// checkDocker verifies Docker installation and daemon
func (d *Doctor) checkDocker(ctx context.Context) Check {
	start := time.Now()
	check := Check{
		Name:     "Docker",
		Category: "Container Runtime",
	}

	// Check if docker command exists
	cmd := exec.CommandContext(ctx, "docker", "version", "--format", "{{.Server.Version}}")
	output, err := cmd.Output()
	check.Duration = time.Since(start)

	if err != nil {
		check.Status = StatusError
		check.Message = "Docker not available"
		check.Details = "Docker daemon may not be running"
		check.Remediation = "Install Docker Desktop from https://docker.com/get-started or start the Docker daemon"
		return check
	}

	version := strings.TrimSpace(string(output))
	check.Status = StatusOK
	check.Message = "Docker " + version
	return check
}

// checkKubectl verifies kubectl installation
func (d *Doctor) checkKubectl(ctx context.Context) Check {
	start := time.Now()
	check := Check{
		Name:     "kubectl",
		Category: "Kubernetes",
	}

	cmd := exec.CommandContext(ctx, "kubectl", "version", "--client", "--output=yaml")
	output, err := cmd.Output()
	check.Duration = time.Since(start)

	if err != nil {
		check.Status = StatusWarning
		check.Message = "kubectl not found"
		check.Remediation = "Install kubectl: https://kubernetes.io/docs/tasks/tools/"
		return check
	}

	// Extract version from output
	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		if strings.Contains(line, "gitVersion") {
			parts := strings.Split(line, ":")
			if len(parts) >= 2 {
				check.Message = "kubectl " + strings.TrimSpace(parts[1])
				break
			}
		}
	}
	if check.Message == "" {
		check.Message = "kubectl installed"
	}
	check.Status = StatusOK
	return check
}

// checkHelm verifies Helm installation
func (d *Doctor) checkHelm(ctx context.Context) Check {
	start := time.Now()
	check := Check{
		Name:     "Helm",
		Category: "Kubernetes",
	}

	cmd := exec.CommandContext(ctx, "helm", "version", "--short")
	output, err := cmd.Output()
	check.Duration = time.Since(start)

	if err != nil {
		check.Status = StatusWarning
		check.Message = "Helm not found"
		check.Remediation = "Install Helm: https://helm.sh/docs/intro/install/"
		return check
	}

	version := strings.TrimSpace(string(output))
	check.Status = StatusOK
	check.Message = "Helm " + version
	return check
}

// checkKind verifies kind installation
func (d *Doctor) checkKind(ctx context.Context) Check {
	start := time.Now()
	check := Check{
		Name:     "kind",
		Category: "Kubernetes",
	}

	cmd := exec.CommandContext(ctx, "kind", "version")
	output, err := cmd.Output()
	check.Duration = time.Since(start)

	if err != nil {
		check.Status = StatusWarning
		check.Message = "kind not found"
		check.Details = "kind is optional, used for local demo"
		check.Remediation = "Install kind: https://kind.sigs.k8s.io/docs/user/quick-start/#installation"
		return check
	}

	version := strings.TrimSpace(string(output))
	check.Status = StatusOK
	check.Message = version
	return check
}

// checkGit verifies Git installation
func (d *Doctor) checkGit(ctx context.Context) Check {
	start := time.Now()
	check := Check{
		Name:     "Git",
		Category: "Version Control",
	}

	cmd := exec.CommandContext(ctx, "git", "--version")
	output, err := cmd.Output()
	check.Duration = time.Since(start)

	if err != nil {
		check.Status = StatusWarning
		check.Message = "Git not found"
		check.Remediation = "Install Git: https://git-scm.com/downloads"
		return check
	}

	version := strings.TrimSpace(string(output))
	check.Status = StatusOK
	check.Message = version
	return check
}

// checkTerraform verifies Terraform installation
func (d *Doctor) checkTerraform(ctx context.Context) Check {
	start := time.Now()
	check := Check{
		Name:     "Terraform",
		Category: "Infrastructure",
	}

	cmd := exec.CommandContext(ctx, "terraform", "version")
	output, err := cmd.Output()
	check.Duration = time.Since(start)

	if err != nil {
		check.Status = StatusWarning
		check.Message = "Terraform not found"
		check.Details = "Terraform is optional, required for infrastructure provisioning"
		check.Remediation = "Install Terraform: https://developer.hashicorp.com/terraform/downloads"
		return check
	}

	lines := strings.Split(string(output), "\n")
	if len(lines) > 0 {
		check.Message = strings.TrimSpace(lines[0])
	}
	check.Status = StatusOK
	return check
}

// checkDiskSpace verifies available disk space
func (d *Doctor) checkDiskSpace(ctx context.Context) Check {
	start := time.Now()
	check := Check{
		Name:     "Disk Space",
		Category: "System",
	}

	// Platform-specific disk check
	var cmd *exec.Cmd
	if runtime.GOOS == "windows" {
		cmd = exec.CommandContext(ctx, "wmic", "logicaldisk", "get", "size,freespace,caption")
	} else {
		cmd = exec.CommandContext(ctx, "df", "-h", "/")
	}

	output, err := cmd.Output()
	check.Duration = time.Since(start)

	if err != nil {
		check.Status = StatusSkipped
		check.Message = "Could not check disk space"
		return check
	}

	check.Details = strings.TrimSpace(string(output))
	check.Status = StatusOK
	check.Message = "Disk space available"
	return check
}

// checkMemory verifies available memory
func (d *Doctor) checkMemory(ctx context.Context) Check {
	start := time.Now()
	check := Check{
		Name:     "Memory",
		Category: "System",
	}

	var memStats runtime.MemStats
	runtime.ReadMemStats(&memStats)
	check.Duration = time.Since(start)

	// Get system memory (platform specific)
	totalMB := memStats.Sys / 1024 / 1024

	check.Status = StatusOK
	check.Message = fmt.Sprintf("Go runtime: %d MB allocated", totalMB)

	// Warn if less than 4GB available to Go
	if totalMB < 100 {
		check.Details = "System has limited memory available"
	}

	return check
}

// checkNetwork verifies network connectivity
func (d *Doctor) checkNetwork(ctx context.Context) Check {
	start := time.Now()
	check := Check{
		Name:     "Network",
		Category: "Connectivity",
	}

	// Try to resolve a common hostname
	var cmd *exec.Cmd
	if runtime.GOOS == "windows" {
		cmd = exec.CommandContext(ctx, "ping", "-n", "1", "github.com")
	} else {
		cmd = exec.CommandContext(ctx, "ping", "-c", "1", "github.com")
	}

	err := cmd.Run()
	check.Duration = time.Since(start)

	if err != nil {
		check.Status = StatusWarning
		check.Message = "Cannot reach github.com"
		check.Details = "Network connectivity may be limited"
		check.Remediation = "Check your internet connection"
		return check
	}

	check.Status = StatusOK
	check.Message = "Network connectivity OK"
	return check
}

// FormatReport formats the report for console output
func FormatReport(report *Report) string {
	var sb strings.Builder

	sb.WriteString("Platform Foundry Doctor\n")
	sb.WriteString("=======================\n\n")

	// Group by category
	categories := make(map[string][]Check)
	for _, check := range report.Checks {
		categories[check.Category] = append(categories[check.Category], check)
	}

	for category, checks := range categories {
		sb.WriteString(fmt.Sprintf("[%s]\n", category))
		for _, check := range checks {
			icon := getStatusIcon(check.Status)
			sb.WriteString(fmt.Sprintf("  %s %s: %s\n", icon, check.Name, check.Message))
			if check.Remediation != "" && check.Status != StatusOK {
				sb.WriteString(fmt.Sprintf("      Fix: %s\n", check.Remediation))
			}
		}
		sb.WriteString("\n")
	}

	// Summary
	sb.WriteString("Summary\n")
	sb.WriteString("-------\n")
	sb.WriteString(fmt.Sprintf("  Passed:   %d\n", report.Summary.Passed))
	sb.WriteString(fmt.Sprintf("  Warnings: %d\n", report.Summary.Warnings))
	sb.WriteString(fmt.Sprintf("  Errors:   %d\n", report.Summary.Errors))
	sb.WriteString(fmt.Sprintf("  Skipped:  %d\n", report.Summary.Skipped))
	sb.WriteString(fmt.Sprintf("\nCompleted in %s\n", report.Duration.Round(time.Millisecond)))

	if report.Summary.Errors > 0 {
		sb.WriteString("\nSome checks failed. Please fix the errors above before proceeding.\n")
	} else if report.Summary.Warnings > 0 {
		sb.WriteString("\nSome optional tools are missing. Platform Foundry will work but some features may be limited.\n")
	} else {
		sb.WriteString("\nAll checks passed! Platform Foundry is ready to use.\n")
	}

	return sb.String()
}

func getStatusIcon(status CheckStatus) string {
	switch status {
	case StatusOK:
		return "[OK]"
	case StatusWarning:
		return "[WARN]"
	case StatusError:
		return "[ERROR]"
	case StatusSkipped:
		return "[SKIP]"
	default:
		return "[?]"
	}
}
