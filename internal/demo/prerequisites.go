package demo

import (
	"fmt"
	"os/exec"
	"runtime"
	"strings"
)

// CheckPrerequisites checks if all prerequisites are met
func CheckPrerequisites() error {
	checks := []struct {
		name  string
		check func() error
	}{
		{"Docker", checkDocker},
		{"kubectl", checkKubectl},
		{"kind", checkKind},
	}

	var errors []string

	for _, c := range checks {
		if err := c.check(); err != nil {
			errors = append(errors, fmt.Sprintf("  ❌ %s: %v", c.name, err))
		} else {
			fmt.Printf("  ✓ %s\n", c.name)
		}
	}

	if len(errors) > 0 {
		return fmt.Errorf("\nMissing prerequisites:\n%s\n\nInstallation instructions:\n%s",
			strings.Join(errors, "\n"), getInstallInstructions())
	}

	return nil
}

func checkDocker() error {
	// Check if docker is installed
	if _, err := exec.LookPath("docker"); err != nil {
		return fmt.Errorf("docker not found")
	}

	// Check if docker daemon is running
	cmd := exec.Command("docker", "ps")
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("docker daemon not running - please start Docker Desktop")
	}

	return nil
}

func checkKubectl() error {
	if _, err := exec.LookPath("kubectl"); err != nil {
		return fmt.Errorf("kubectl not found")
	}

	// Check version
	cmd := exec.Command("kubectl", "version", "--client", "--short")
	output, err := cmd.Output()
	if err != nil {
		return fmt.Errorf("kubectl not working properly")
	}

	// Basic version check
	if !strings.Contains(string(output), "Client Version") {
		return fmt.Errorf("kubectl version check failed")
	}

	return nil
}

func checkKind() error {
	if _, err := exec.LookPath("kind"); err != nil {
		return fmt.Errorf("kind not found")
	}

	// Check version
	cmd := exec.Command("kind", "version")
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("kind not working properly")
	}

	return nil
}

func getInstallInstructions() string {
	var instructions string

	switch runtime.GOOS {
	case "windows":
		instructions = `
Windows Installation:

1. Docker Desktop:
   Download from: https://www.docker.com/products/docker-desktop

2. kubectl:
   choco install kubernetes-cli
   OR download from: https://kubernetes.io/docs/tasks/tools/install-kubectl-windows/

3. kind:
   choco install kind
   OR: curl.exe -Lo kind-windows-amd64.exe https://kind.sigs.k8s.io/dl/latest/kind-windows-amd64
       move kind-windows-amd64.exe C:\Windows\System32\kind.exe
`

	case "darwin":
		instructions = `
macOS Installation:

Using Homebrew:
   brew install docker kubectl kind

Or install Docker Desktop from: https://www.docker.com/products/docker-desktop
`

	case "linux":
		instructions = `
Linux Installation:

1. Docker:
   curl -fsSL https://get.docker.com -o get-docker.sh
   sudo sh get-docker.sh

2. kubectl:
   curl -LO "https://dl.k8s.io/release/$(curl -L -s https://dl.k8s.io/release/stable.txt)/bin/linux/amd64/kubectl"
   sudo install -o root -g root -m 0755 kubectl /usr/local/bin/kubectl

3. kind:
   curl -Lo ./kind https://kind.sigs.k8s.io/dl/latest/kind-linux-amd64
   chmod +x ./kind
   sudo mv ./kind /usr/local/bin/kind
`

	default:
		instructions = "Please install Docker, kubectl, and kind for your operating system."
	}

	return instructions
}

// CleanupCluster removes a kind cluster
func CleanupCluster(clusterName string) error {
	cluster := NewKindCluster(clusterName)
	if !cluster.Exists() {
		return fmt.Errorf("cluster %s does not exist", clusterName)
	}

	return cluster.Delete()
}
