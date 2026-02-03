package cli

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/platformfoundry/pf-ce/internal/mesh"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

var meshManager *mesh.Manager

func init() {
	meshManager = mesh.NewManager()
}

var meshCmd = &cobra.Command{
	Use:   "mesh",
	Short: "Service mesh management",
	Long: `Manage service mesh configuration including mTLS, traffic policies,
circuit breakers, and observability settings.`,
}

var meshApplyCmd = &cobra.Command{
	Use:   "apply -f <file>",
	Short: "Apply mesh configuration from file",
	RunE:  runMeshApply,
}

var meshGetCmd = &cobra.Command{
	Use:   "get <name>",
	Short: "Get mesh configuration",
	Args:  cobra.ExactArgs(1),
	RunE:  runMeshGet,
}

var meshListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all mesh configurations",
	RunE:  runMeshList,
}

var meshStatusCmd = &cobra.Command{
	Use:   "status [name]",
	Short: "Show mesh status",
	RunE:  runMeshStatus,
}

var meshDeleteCmd = &cobra.Command{
	Use:   "delete <name>",
	Short: "Delete mesh configuration",
	Args:  cobra.ExactArgs(1),
	RunE:  runMeshDelete,
}

// Traffic commands
var meshTrafficCmd = &cobra.Command{
	Use:   "traffic",
	Short: "Traffic management commands",
}

var meshTrafficSplitCmd = &cobra.Command{
	Use:   "split <service>",
	Short: "Configure traffic split",
	Args:  cobra.ExactArgs(1),
	RunE:  runMeshTrafficSplit,
}

var meshTrafficMetricsCmd = &cobra.Command{
	Use:   "metrics <service>",
	Short: "Show traffic metrics",
	Args:  cobra.ExactArgs(1),
	RunE:  runMeshTrafficMetrics,
}

// mTLS commands
var meshMTLSCmd = &cobra.Command{
	Use:   "mtls",
	Short: "mTLS configuration commands",
}

var meshMTLSStatusCmd = &cobra.Command{
	Use:   "status [mesh]",
	Short: "Show mTLS status",
	RunE:  runMeshMTLSStatus,
}

var meshMTLSSetCmd = &cobra.Command{
	Use:   "set <mesh> <mode>",
	Short: "Set mTLS mode (strict, permissive, disabled)",
	Args:  cobra.ExactArgs(2),
	RunE:  runMeshMTLSSet,
}

// Fault injection commands
var meshFaultCmd = &cobra.Command{
	Use:   "fault",
	Short: "Fault injection commands",
}

var meshFaultInjectCmd = &cobra.Command{
	Use:   "inject <service>",
	Short: "Inject fault into service",
	Args:  cobra.ExactArgs(1),
	RunE:  runMeshFaultInject,
}

var meshFaultRemoveCmd = &cobra.Command{
	Use:   "remove <service>",
	Short: "Remove fault injection",
	Args:  cobra.ExactArgs(1),
	RunE:  runMeshFaultRemove,
}

// Graph command
var meshGraphCmd = &cobra.Command{
	Use:   "graph",
	Short: "Show service dependency graph",
	RunE:  runMeshGraph,
}

var (
	meshFile        string
	meshWeights     []string
	meshFaultDelay  string
	meshFaultAbort  int
	meshFaultPct    float64
)

func init() {
	meshCmd.AddCommand(meshApplyCmd)
	meshCmd.AddCommand(meshGetCmd)
	meshCmd.AddCommand(meshListCmd)
	meshCmd.AddCommand(meshStatusCmd)
	meshCmd.AddCommand(meshDeleteCmd)
	meshCmd.AddCommand(meshTrafficCmd)
	meshCmd.AddCommand(meshMTLSCmd)
	meshCmd.AddCommand(meshFaultCmd)
	meshCmd.AddCommand(meshGraphCmd)

	// Traffic subcommands
	meshTrafficCmd.AddCommand(meshTrafficSplitCmd)
	meshTrafficCmd.AddCommand(meshTrafficMetricsCmd)

	// mTLS subcommands
	meshMTLSCmd.AddCommand(meshMTLSStatusCmd)
	meshMTLSCmd.AddCommand(meshMTLSSetCmd)

	// Fault subcommands
	meshFaultCmd.AddCommand(meshFaultInjectCmd)
	meshFaultCmd.AddCommand(meshFaultRemoveCmd)

	// Flags
	meshApplyCmd.Flags().StringVarP(&meshFile, "file", "f", "", "Path to mesh configuration file")
	meshApplyCmd.MarkFlagRequired("file")

	meshTrafficSplitCmd.Flags().StringArrayVar(&meshWeights, "weight", nil, "Version weights (version=weight)")

	meshFaultInjectCmd.Flags().StringVar(&meshFaultDelay, "delay", "", "Delay duration (e.g., 500ms)")
	meshFaultInjectCmd.Flags().IntVar(&meshFaultAbort, "abort", 0, "HTTP status code to abort with")
	meshFaultInjectCmd.Flags().Float64Var(&meshFaultPct, "percentage", 100, "Percentage of requests to affect")
}

func runMeshApply(cmd *cobra.Command, args []string) error {
	data, err := os.ReadFile(meshFile)
	if err != nil {
		return fmt.Errorf("failed to read file: %w", err)
	}

	var meshConfig mesh.ServiceMesh
	if err := yaml.Unmarshal(data, &meshConfig); err != nil {
		return fmt.Errorf("failed to parse mesh config: %w", err)
	}

	ctx := context.Background()
	if err := meshManager.RegisterMesh(ctx, &meshConfig); err != nil {
		return err
	}

	fmt.Printf("Mesh '%s' applied successfully\n", meshConfig.Metadata.Name)
	fmt.Printf("  Provider: %s\n", meshConfig.Spec.Provider)
	fmt.Printf("  mTLS:     %s\n", meshConfig.Spec.MTLS.Mode)

	return nil
}

func runMeshGet(cmd *cobra.Command, args []string) error {
	meshConfig, err := meshManager.GetMesh(args[0])
	if err != nil {
		return err
	}

	data, err := yaml.Marshal(meshConfig)
	if err != nil {
		return fmt.Errorf("failed to marshal mesh: %w", err)
	}

	fmt.Println(string(data))
	return nil
}

func runMeshList(cmd *cobra.Command, args []string) error {
	meshes := meshManager.ListMeshes()

	if len(meshes) == 0 {
		fmt.Println("No mesh configurations found")
		return nil
	}

	fmt.Printf("%-20s %-12s %-12s %-12s %s\n", "NAME", "PROVIDER", "MTLS", "STATUS", "SERVICES")
	fmt.Println(strings.Repeat("-", 70))

	for _, m := range meshes {
		status := "Unknown"
		services := 0
		if m.Status != nil {
			status = string(m.Status.Phase)
			services = m.Status.Services
		}

		fmt.Printf("%-20s %-12s %-12s %-12s %d\n",
			m.Metadata.Name,
			m.Spec.Provider,
			m.Spec.MTLS.Mode,
			status,
			services)
	}

	return nil
}

func runMeshStatus(cmd *cobra.Command, args []string) error {
	name := "production-mesh"
	if len(args) > 0 {
		name = args[0]
	}

	meshConfig, err := meshManager.GetMesh(name)
	if err != nil {
		return err
	}

	fmt.Printf("Service Mesh: %s\n", meshConfig.Metadata.Name)
	fmt.Println(strings.Repeat("=", 50))

	fmt.Printf("\nProvider:   %s\n", meshConfig.Spec.Provider)
	fmt.Printf("Namespace:  %s\n", meshConfig.Metadata.Namespace)

	if meshConfig.Status != nil {
		status := meshConfig.Status
		fmt.Printf("\nStatus:     %s\n", status.Phase)
		fmt.Printf("Services:   %d\n", status.Services)
		fmt.Printf("Proxies:    %d/%d healthy\n", status.ProxiesHealthy, status.ProxiesTotal)

		if len(status.Conditions) > 0 {
			fmt.Println("\nConditions:")
			for _, c := range status.Conditions {
				icon := "?"
				if c.Status == "True" {
					icon = "V"
				} else if c.Status == "False" {
					icon = "X"
				}
				fmt.Printf("  %s %s: %s\n", icon, c.Type, c.Message)
			}
		}
	}

	// mTLS status
	fmt.Println("\nmTLS Configuration:")
	fmt.Printf("  Mode:        %s\n", meshConfig.Spec.MTLS.Mode)
	if meshConfig.Spec.MTLS.MinTLSVersion != "" {
		fmt.Printf("  Min TLS:     %s\n", meshConfig.Spec.MTLS.MinTLSVersion)
	}
	if meshConfig.Spec.MTLS.WorkloadCertTTL != "" {
		fmt.Printf("  Cert TTL:    %s\n", meshConfig.Spec.MTLS.WorkloadCertTTL)
	}

	// Traffic config
	fmt.Println("\nTraffic Configuration:")
	traffic := meshConfig.Spec.Traffic
	fmt.Printf("  Retries:           %d attempts, %s timeout\n",
		traffic.Retries.Attempts, traffic.Retries.PerTryTimeout)
	fmt.Printf("  Circuit Breaker:   %d errors, %s interval\n",
		traffic.CircuitBreaker.ConsecutiveErrors, traffic.CircuitBreaker.Interval)
	fmt.Printf("  Request Timeout:   %s\n", traffic.Timeout.Request)
	fmt.Printf("  Load Balancing:    %s\n", traffic.LoadBalancing.Algorithm)

	// Observability
	fmt.Println("\nObservability:")
	obs := meshConfig.Spec.Observability
	fmt.Printf("  Tracing:    %v (%.1f%% sampling)\n", obs.Tracing.Enabled, obs.Tracing.Sampling)
	fmt.Printf("  Metrics:    %v\n", obs.Metrics.Enabled)
	fmt.Printf("  Logging:    %v (%s)\n", obs.Logging.Enabled, obs.Logging.Format)

	return nil
}

func runMeshDelete(cmd *cobra.Command, args []string) error {
	if err := meshManager.DeleteMesh(args[0]); err != nil {
		return err
	}
	fmt.Printf("Mesh '%s' deleted\n", args[0])
	return nil
}

func runMeshTrafficSplit(cmd *cobra.Command, args []string) error {
	service := args[0]

	if len(meshWeights) == 0 {
		return fmt.Errorf("at least one --weight is required (e.g., --weight v1=80 --weight v2=20)")
	}

	weights := make(map[string]int)
	for _, w := range meshWeights {
		parts := strings.SplitN(w, "=", 2)
		if len(parts) != 2 {
			return fmt.Errorf("invalid weight format: %s (expected version=weight)", w)
		}
		var weight int
		fmt.Sscanf(parts[1], "%d", &weight)
		weights[parts[0]] = weight
	}

	ctx := context.Background()
	if err := meshManager.ConfigureTrafficSplit(ctx, service, weights); err != nil {
		return err
	}

	fmt.Printf("Traffic split configured for %s:\n", service)
	for version, weight := range weights {
		fmt.Printf("  %s: %d%%\n", version, weight)
	}

	return nil
}

func runMeshTrafficMetrics(cmd *cobra.Command, args []string) error {
	service := args[0]
	metrics := meshManager.GetTrafficMetrics(service)

	fmt.Printf("Traffic Metrics for %s\n", service)
	fmt.Println(strings.Repeat("=", 50))

	fmt.Printf("\nThroughput:\n")
	fmt.Printf("  Requests/sec:     %.1f\n", metrics["requestsPerSecond"])
	fmt.Printf("  Active Conns:     %.0f\n", metrics["activeConnections"])

	fmt.Printf("\nLatency:\n")
	fmt.Printf("  P50:              %.1f ms\n", metrics["p50LatencyMs"])
	fmt.Printf("  P99:              %.1f ms\n", metrics["p99LatencyMs"])

	fmt.Printf("\nReliability:\n")
	fmt.Printf("  Success Rate:     %.2f%%\n", metrics["successRate"])
	fmt.Printf("  Error Rate:       %.2f%%\n", metrics["errorRate"].(float64)*100)

	fmt.Printf("\nBandwidth:\n")
	fmt.Printf("  Bytes In:         %.1f MB\n", float64(metrics["bytesIn"].(int))/1024/1024)
	fmt.Printf("  Bytes Out:        %.1f MB\n", float64(metrics["bytesOut"].(int))/1024/1024)

	return nil
}

func runMeshMTLSStatus(cmd *cobra.Command, args []string) error {
	name := "production-mesh"
	if len(args) > 0 {
		name = args[0]
	}

	meshConfig, err := meshManager.GetMesh(name)
	if err != nil {
		return err
	}

	mtls := meshConfig.Spec.MTLS

	fmt.Printf("mTLS Status for %s\n", name)
	fmt.Println(strings.Repeat("=", 50))

	fmt.Printf("\nMode:                 %s\n", mtls.Mode)
	fmt.Printf("Min TLS Version:      %s\n", mtls.MinTLSVersion)
	fmt.Printf("Workload Cert TTL:    %s\n", mtls.WorkloadCertTTL)

	if mtls.CertificateAuthority != "" {
		fmt.Printf("Certificate Authority: %s\n", mtls.CertificateAuthority)
	}

	// Status interpretation
	fmt.Println()
	switch mtls.Mode {
	case mesh.MTLSModeStrict:
		fmt.Println("Status: All traffic is encrypted with mTLS")
	case mesh.MTLSModePermissive:
		fmt.Println("Status: mTLS is enabled but allows plain text traffic")
	case mesh.MTLSModeDisabled:
		fmt.Println("Status: mTLS is disabled - traffic is not encrypted")
	}

	return nil
}

func runMeshMTLSSet(cmd *cobra.Command, args []string) error {
	meshName := args[0]
	modeStr := args[1]

	var mode mesh.MTLSMode
	switch strings.ToLower(modeStr) {
	case "strict":
		mode = mesh.MTLSModeStrict
	case "permissive":
		mode = mesh.MTLSModePermissive
	case "disabled":
		mode = mesh.MTLSModeDisabled
	default:
		return fmt.Errorf("invalid mTLS mode: %s (use: strict, permissive, disabled)", modeStr)
	}

	if err := meshManager.UpdateMTLS(meshName, mode); err != nil {
		return err
	}

	fmt.Printf("mTLS mode set to '%s' for mesh '%s'\n", mode, meshName)
	return nil
}

func runMeshFaultInject(cmd *cobra.Command, args []string) error {
	service := args[0]

	if meshFaultDelay == "" && meshFaultAbort == 0 {
		return fmt.Errorf("at least --delay or --abort is required")
	}

	fault := mesh.FaultInjection{}

	if meshFaultDelay != "" {
		fault.Delay = &mesh.FaultDelay{
			FixedDelay: meshFaultDelay,
			Percentage: meshFaultPct,
		}
	}

	if meshFaultAbort > 0 {
		fault.Abort = &mesh.FaultAbort{
			HTTPStatus: meshFaultAbort,
			Percentage: meshFaultPct,
		}
	}

	ctx := context.Background()
	if err := meshManager.InjectFault(ctx, service, fault); err != nil {
		return err
	}

	fmt.Printf("Fault injection configured for %s:\n", service)
	if fault.Delay != nil {
		fmt.Printf("  Delay: %s (%.1f%% of requests)\n", fault.Delay.FixedDelay, fault.Delay.Percentage)
	}
	if fault.Abort != nil {
		fmt.Printf("  Abort: HTTP %d (%.1f%% of requests)\n", fault.Abort.HTTPStatus, fault.Abort.Percentage)
	}

	return nil
}

func runMeshFaultRemove(cmd *cobra.Command, args []string) error {
	service := args[0]
	ctx := context.Background()

	if err := meshManager.RemoveFault(ctx, service); err != nil {
		return err
	}

	fmt.Printf("Fault injection removed from %s\n", service)
	return nil
}

func runMeshGraph(cmd *cobra.Command, args []string) error {
	graph := meshManager.GetServiceGraph()

	fmt.Println("Service Dependency Graph")
	fmt.Println(strings.Repeat("=", 50))
	fmt.Println()

	for service, deps := range graph {
		fmt.Printf("%s\n", service)
		for i, dep := range deps {
			prefix := "├──"
			if i == len(deps)-1 {
				prefix = "└──"
			}
			fmt.Printf("  %s %s\n", prefix, dep)
		}
		fmt.Println()
	}

	return nil
}

// GetMeshCmd returns the mesh command for registration
func GetMeshCmd() *cobra.Command {
	return meshCmd
}
