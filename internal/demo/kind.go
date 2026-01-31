package demo

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"
)

// KindCluster manages a kind cluster
type KindCluster struct {
	Name string
}

// NewKindCluster creates a new kind cluster manager
func NewKindCluster(name string) *KindCluster {
	return &KindCluster{
		Name: name,
	}
}

// Create creates a new kind cluster
func (k *KindCluster) Create() error {
	// Check if cluster already exists
	if k.Exists() {
		return fmt.Errorf("cluster %s already exists, run 'pf demo clean' first", k.Name)
	}

	// Create kind config
	config := `
kind: Cluster
apiVersion: kind.x-k8s.io/v1alpha4
nodes:
- role: control-plane
  extraPortMappings:
  - containerPort: 30000
    hostPort: 9090
    protocol: TCP
  - containerPort: 30001
    hostPort: 3000
    protocol: TCP
  - containerPort: 30002
    hostPort: 8080
    protocol: TCP
  - containerPort: 30003
    hostPort: 7007
    protocol: TCP
`

	// Write config to temp file
	configFile := fmt.Sprintf(".pf/kind/%s-config.yaml", k.Name)
	if err := os.MkdirAll(".pf/kind", 0755); err != nil {
		return err
	}
	if err := os.WriteFile(configFile, []byte(config), 0644); err != nil {
		return err
	}

	// Create cluster
	cmd := exec.Command("kind", "create", "cluster", "--name", k.Name, "--config", configFile)

	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to create cluster: %s", stderr.String())
	}

	// Set kubectl context
	cmd = exec.Command("kind", "export", "kubeconfig", "--name", k.Name)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to export kubeconfig: %w", err)
	}

	return nil
}

// Delete deletes the kind cluster
func (k *KindCluster) Delete() error {
	if !k.Exists() {
		return nil // Already deleted
	}

	cmd := exec.Command("kind", "delete", "cluster", "--name", k.Name)
	return cmd.Run()
}

// Exists checks if the cluster exists
func (k *KindCluster) Exists() bool {
	cmd := exec.Command("kind", "get", "clusters")
	output, err := cmd.Output()
	if err != nil {
		return false
	}

	clusters := strings.Split(strings.TrimSpace(string(output)), "\n")
	for _, cluster := range clusters {
		if cluster == k.Name {
			return true
		}
	}

	return false
}

// WaitReady waits for the cluster to be ready
func (k *KindCluster) WaitReady() error {
	maxWait := 2 * time.Minute
	interval := 5 * time.Second
	deadline := time.Now().Add(maxWait)

	for time.Now().Before(deadline) {
		// Check if nodes are ready
		cmd := exec.Command("kubectl", "get", "nodes", "-o", "jsonpath={.items[*].status.conditions[?(@.type=='Ready')].status}")
		output, err := cmd.Output()
		if err == nil && strings.Contains(string(output), "True") {
			return nil
		}

		time.Sleep(interval)
	}

	return fmt.Errorf("cluster did not become ready within %v", maxWait)
}

// CreateNamespace creates a namespace in the cluster
func (k *KindCluster) CreateNamespace(name string) error {
	cmd := exec.Command("kubectl", "create", "namespace", name)
	output, err := cmd.CombinedOutput()

	// Ignore if already exists
	if err != nil && !strings.Contains(string(output), "AlreadyExists") {
		return fmt.Errorf("failed to create namespace %s: %s", name, string(output))
	}

	return nil
}

// ApplyManifest applies a Kubernetes manifest
func (k *KindCluster) ApplyManifest(manifest string) error {
	cmd := exec.Command("kubectl", "apply", "-f", "-")
	cmd.Stdin = strings.NewReader(manifest)

	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to apply manifest: %s", stderr.String())
	}

	return nil
}

// WaitForPod waits for a pod to be ready
func (k *KindCluster) WaitForPod(namespace, selector string, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)

	for time.Now().Before(deadline) {
		cmd := exec.Command("kubectl", "get", "pods", "-n", namespace, "-l", selector,
			"-o", "jsonpath={.items[*].status.conditions[?(@.type=='Ready')].status}")
		output, err := cmd.Output()

		if err == nil && strings.Contains(string(output), "True") {
			return nil
		}

		time.Sleep(2 * time.Second)
	}

	return fmt.Errorf("pod with selector %s in namespace %s did not become ready", selector, namespace)
}

// GetServiceURL gets the URL for a service
func (k *KindCluster) GetServiceURL(namespace, service string, port int) string {
	return fmt.Sprintf("http://localhost:%d", port)
}
