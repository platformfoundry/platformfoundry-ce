package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"text/tabwriter"
	"time"

	"github.com/spf13/cobra"
)

var federationCmd = &cobra.Command{
	Use:     "federation",
	Aliases: []string{"fed"},
	Short:   "Manage cluster federation",
	Long: `Manage federated clusters and multi-cloud deployments.

Federation enables:
- Multi-cluster workload orchestration
- Global traffic management
- Automated failover
- Cross-cluster replication`,
}

var fedClusterCmd = &cobra.Command{
	Use:   "cluster",
	Short: "Manage federated clusters",
}

var fedClusterListCmd = &cobra.Command{
	Use:   "list",
	Short: "List federated clusters",
	RunE: func(cmd *cobra.Command, args []string) error {
		format, _ := cmd.Flags().GetString("output")
		region, _ := cmd.Flags().GetString("region")

		clusters := getSampleFederatedClusters()

		if region != "" {
			filtered := make([]federatedClusterInfo, 0)
			for _, c := range clusters {
				if c.Region == region {
					filtered = append(filtered, c)
				}
			}
			clusters = filtered
		}

		if format == "json" {
			data, _ := json.MarshalIndent(clusters, "", "  ")
			fmt.Println(string(data))
			return nil
		}

		w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
		fmt.Fprintln(w, "NAME\tPROVIDER\tREGION\tROLE\tSTATUS\tNODES\tLAST HEALTHY")
		for _, c := range clusters {
			fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\t%d\t%s\n",
				c.Name, c.Provider, c.Region, c.Role, c.Status, c.Nodes, c.LastHealthy)
		}
		w.Flush()

		return nil
	},
}

var fedClusterRegisterCmd = &cobra.Command{
	Use:   "register",
	Short: "Register a cluster with the federation",
	Long: `Register a new cluster with the federation.

Examples:
  # Register a cluster using kubeconfig
  pf federation cluster register --name prod-us-east \
    --provider aws --region us-east-1 --role primary \
    --kubeconfig ~/.kube/prod-us-east.yaml

  # Register a cluster using endpoint
  pf federation cluster register --name prod-eu-west \
    --provider gcp --region europe-west1 --role secondary \
    --endpoint https://cluster.example.com`,
	RunE: func(cmd *cobra.Command, args []string) error {
		name, _ := cmd.Flags().GetString("name")
		provider, _ := cmd.Flags().GetString("provider")
		region, _ := cmd.Flags().GetString("region")
		role, _ := cmd.Flags().GetString("role")
		endpoint, _ := cmd.Flags().GetString("endpoint")
		kubeconfig, _ := cmd.Flags().GetString("kubeconfig")

		if name == "" || provider == "" || region == "" {
			return fmt.Errorf("--name, --provider, and --region are required")
		}

		fmt.Printf("Registering cluster '%s'...\n", name)
		fmt.Printf("  Provider: %s\n", provider)
		fmt.Printf("  Region: %s\n", region)
		fmt.Printf("  Role: %s\n", role)
		if endpoint != "" {
			fmt.Printf("  Endpoint: %s\n", endpoint)
		}
		if kubeconfig != "" {
			fmt.Printf("  Kubeconfig: %s\n", kubeconfig)
		}
		fmt.Println()
		fmt.Println("Validating cluster connectivity...")
		fmt.Println("Starting health monitoring...")
		fmt.Printf("Cluster '%s' registered successfully.\n", name)

		return nil
	},
}

var fedClusterUnregisterCmd = &cobra.Command{
	Use:   "unregister [name]",
	Short: "Unregister a cluster from the federation",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		name := args[0]
		force, _ := cmd.Flags().GetBool("force")
		drain, _ := cmd.Flags().GetBool("drain")

		if drain {
			fmt.Printf("Draining workloads from cluster '%s'...\n", name)
		}

		if !force {
			fmt.Printf("Are you sure you want to unregister cluster '%s'?\n", name)
			fmt.Print("Type 'yes' to confirm: ")
			var confirm string
			fmt.Scanln(&confirm)
			if confirm != "yes" {
				fmt.Println("Aborted.")
				return nil
			}
		}

		fmt.Printf("Unregistering cluster '%s'...\n", name)
		fmt.Println("Stopping health monitoring...")
		fmt.Printf("Cluster '%s' unregistered successfully.\n", name)

		return nil
	},
}

var fedClusterHealthCmd = &cobra.Command{
	Use:   "health [name]",
	Short: "Get cluster health status",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		name := args[0]

		health := getSampleClusterHealth(name)

		fmt.Printf("Cluster Health: %s\n", name)
		fmt.Println(string(repeatChar('-', 40)))
		fmt.Printf("Status:       %s\n", health.Status)
		fmt.Printf("Latency:      %s\n", health.Latency)
		fmt.Printf("Nodes:        %d/%d healthy\n", health.HealthyNodes, health.TotalNodes)
		fmt.Printf("CPU Usage:    %.1f%%\n", health.CPUUsage)
		fmt.Printf("Memory Usage: %.1f%%\n", health.MemoryUsage)
		fmt.Printf("Disk Usage:   %.1f%%\n", health.DiskUsage)
		fmt.Printf("Last Check:   %s\n", health.LastCheck)

		return nil
	},
}

var fedCreateCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a federation",
	Long: `Create a new federation of clusters.

Examples:
  pf federation create --name global-prod \
    --clusters prod-us-east,prod-eu-west,prod-ap-southeast \
    --failover automatic --traffic weighted`,
	RunE: func(cmd *cobra.Command, args []string) error {
		name, _ := cmd.Flags().GetString("name")
		clusters, _ := cmd.Flags().GetStringSlice("clusters")
		failover, _ := cmd.Flags().GetString("failover")
		traffic, _ := cmd.Flags().GetString("traffic")

		if name == "" || len(clusters) == 0 {
			return fmt.Errorf("--name and --clusters are required")
		}

		fmt.Printf("Creating federation '%s'...\n", name)
		fmt.Printf("Clusters: %v\n", clusters)
		fmt.Printf("Failover: %s\n", failover)
		fmt.Printf("Traffic: %s\n", traffic)
		fmt.Println()
		fmt.Printf("Federation '%s' created successfully.\n", name)
		fmt.Println("Configure traffic policies with: pf federation traffic set")

		return nil
	},
}

var fedListCmd = &cobra.Command{
	Use:   "list",
	Short: "List federations",
	RunE: func(cmd *cobra.Command, args []string) error {
		format, _ := cmd.Flags().GetString("output")

		federations := getSampleFederations()

		if format == "json" {
			data, _ := json.MarshalIndent(federations, "", "  ")
			fmt.Println(string(data))
			return nil
		}

		w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
		fmt.Fprintln(w, "NAME\tCLUSTERS\tFAILOVER\tTRAFFIC\tCREATED")
		for _, f := range federations {
			fmt.Fprintf(w, "%s\t%d\t%s\t%s\t%s\n",
				f.Name, f.ClusterCount, f.Failover, f.Traffic, f.Created)
		}
		w.Flush()

		return nil
	},
}

var fedTrafficCmd = &cobra.Command{
	Use:   "traffic",
	Short: "Manage traffic policies",
}

var fedTrafficGetCmd = &cobra.Command{
	Use:   "get [service]",
	Short: "Get traffic distribution for a service",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		service := args[0]

		distribution := getSampleTrafficDistribution(service)

		fmt.Printf("Traffic Distribution: %s\n", service)
		fmt.Println(string(repeatChar('-', 40)))
		fmt.Printf("Policy: %s\n", distribution.Policy)
		fmt.Println()
		fmt.Println("Cluster Weights:")

		w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
		fmt.Fprintln(w, "  CLUSTER\tWEIGHT\tLATENCY\tERROR RATE\tRPS")
		for _, c := range distribution.Clusters {
			fmt.Fprintf(w, "  %s\t%d%%\t%s\t%.2f%%\t%.0f\n",
				c.Name, c.Weight, c.Latency, c.ErrorRate*100, c.RPS)
		}
		w.Flush()

		return nil
	},
}

var fedTrafficSetCmd = &cobra.Command{
	Use:   "set [service]",
	Short: "Set traffic distribution for a service",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		service := args[0]
		policy, _ := cmd.Flags().GetString("policy")
		weights, _ := cmd.Flags().GetStringToInt("weights")

		fmt.Printf("Updating traffic policy for '%s'...\n", service)
		fmt.Printf("Policy: %s\n", policy)
		if len(weights) > 0 {
			fmt.Println("Weights:")
			for cluster, weight := range weights {
				fmt.Printf("  %s: %d%%\n", cluster, weight)
			}
		}
		fmt.Println()
		fmt.Printf("Traffic policy updated for '%s'.\n", service)

		return nil
	},
}

var fedTrafficMigrateCmd = &cobra.Command{
	Use:   "migrate [service]",
	Short: "Migrate traffic between clusters",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		service := args[0]
		from, _ := cmd.Flags().GetString("from")
		to, _ := cmd.Flags().GetString("to")
		gradual, _ := cmd.Flags().GetBool("gradual")

		if from == "" || to == "" {
			return fmt.Errorf("--from and --to are required")
		}

		fmt.Printf("Migrating traffic for '%s'...\n", service)
		fmt.Printf("From: %s\n", from)
		fmt.Printf("To: %s\n", to)
		fmt.Printf("Gradual: %v\n", gradual)
		fmt.Println()

		if gradual {
			fmt.Println("Performing gradual migration...")
			steps := []int{10, 25, 50, 75, 100}
			for _, step := range steps {
				fmt.Printf("  Shifting %d%% traffic to %s...\n", step, to)
				time.Sleep(500 * time.Millisecond) // Simulated delay
				fmt.Printf("  Verifying error rates... OK\n")
			}
		}

		fmt.Println()
		fmt.Printf("Traffic migration complete. 100%% traffic now routes to %s.\n", to)

		return nil
	},
}

var fedFailoverCmd = &cobra.Command{
	Use:   "failover",
	Short: "Manage failover",
}

var fedSyncCmd = &cobra.Command{
	Use:   "sync",
	Short: "Manage configuration synchronization",
}

var fedSyncCreateCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a sync policy",
	Long: `Create a new configuration sync policy.

Examples:
  # Create sync policy from YAML file
  pf federation sync create -f sync-policy.yaml

  # Create simple sync policy
  pf federation sync create --name config-sync \
    --source prod-us-east --targets prod-eu-west,prod-ap-southeast \
    --resources ConfigMap,Secret --mode push`,
	RunE: func(cmd *cobra.Command, args []string) error {
		filePath, _ := cmd.Flags().GetString("file")
		name, _ := cmd.Flags().GetString("name")
		source, _ := cmd.Flags().GetString("source")
		targets, _ := cmd.Flags().GetStringSlice("targets")
		resources, _ := cmd.Flags().GetStringSlice("resources")
		mode, _ := cmd.Flags().GetString("mode")

		if filePath != "" {
			fmt.Printf("Creating sync policy from file: %s\n", filePath)
		} else {
			if name == "" || source == "" || len(targets) == 0 {
				return fmt.Errorf("--name, --source, and --targets are required")
			}
			fmt.Printf("Creating sync policy '%s'...\n", name)
			fmt.Printf("  Source: %s\n", source)
			fmt.Printf("  Targets: %v\n", targets)
			fmt.Printf("  Resources: %v\n", resources)
			fmt.Printf("  Mode: %s\n", mode)
		}
		fmt.Println()
		fmt.Printf("Sync policy created successfully.\n")

		return nil
	},
}

var fedSyncListCmd = &cobra.Command{
	Use:   "list",
	Short: "List sync policies",
	RunE: func(cmd *cobra.Command, args []string) error {
		format, _ := cmd.Flags().GetString("output")

		policies := getSampleSyncPolicies()

		if format == "json" {
			data, _ := json.MarshalIndent(policies, "", "  ")
			fmt.Println(string(data))
			return nil
		}

		w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
		fmt.Fprintln(w, "NAME\tSOURCE\tTARGETS\tMODE\tSTATUS\tLAST SYNC")
		for _, p := range policies {
			fmt.Fprintf(w, "%s\t%s\t%d\t%s\t%s\t%s\n",
				p.Name, p.Source, p.TargetCount, p.Mode, p.Status, p.LastSync)
		}
		w.Flush()

		return nil
	},
}

var fedSyncStatusCmd = &cobra.Command{
	Use:   "status [policy-name]",
	Short: "Get sync policy status",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		name := args[0]

		fmt.Printf("Sync Policy: %s\n", name)
		fmt.Println(string(repeatChar('-', 40)))
		fmt.Printf("Status:           synced\n")
		fmt.Printf("Last Sync:        2m ago\n")
		fmt.Printf("Next Sync:        3m\n")
		fmt.Printf("Resources Synced: 15\n")
		fmt.Printf("Conflicts:        0\n")
		fmt.Println()
		fmt.Println("Cluster Status:")
		w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
		fmt.Fprintln(w, "  CLUSTER\tSTATUS\tLAST SYNC\tRESOURCES")
		fmt.Fprintln(w, "  prod-eu-west\tsynced\t2m ago\t8")
		fmt.Fprintln(w, "  prod-ap-southeast\tsynced\t2m ago\t7")
		w.Flush()

		return nil
	},
}

var fedSyncTriggerCmd = &cobra.Command{
	Use:   "trigger [policy-name]",
	Short: "Manually trigger synchronization",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		name := args[0]
		wait, _ := cmd.Flags().GetBool("wait")

		fmt.Printf("Triggering sync for policy '%s'...\n", name)

		if wait {
			fmt.Println("Waiting for sync to complete...")
			fmt.Println("  Fetching resources from source...")
			fmt.Println("  Comparing with targets...")
			fmt.Println("  Applying changes...")
		}

		fmt.Printf("Sync completed successfully.\n")
		return nil
	},
}

var fedSyncDeleteCmd = &cobra.Command{
	Use:   "delete [policy-name]",
	Short: "Delete a sync policy",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		name := args[0]
		force, _ := cmd.Flags().GetBool("force")

		if !force {
			fmt.Printf("Are you sure you want to delete sync policy '%s'?\n", name)
			fmt.Print("Type 'yes' to confirm: ")
			var confirm string
			fmt.Scanln(&confirm)
			if confirm != "yes" {
				fmt.Println("Aborted.")
				return nil
			}
		}

		fmt.Printf("Deleting sync policy '%s'...\n", name)
		fmt.Printf("Sync policy '%s' deleted.\n", name)
		return nil
	},
}

var fedSyncConflictsCmd = &cobra.Command{
	Use:   "conflicts [policy-name]",
	Short: "List sync conflicts",
	Args:  cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		if len(args) > 0 {
			fmt.Printf("Conflicts for policy: %s\n", args[0])
		} else {
			fmt.Println("All Sync Conflicts:")
		}
		fmt.Println()
		fmt.Println("No conflicts found.")
		return nil
	},
}

var fedFailoverTriggerCmd = &cobra.Command{
	Use:   "trigger [cluster]",
	Short: "Trigger failover from a cluster",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		cluster := args[0]
		target, _ := cmd.Flags().GetString("target")
		force, _ := cmd.Flags().GetBool("force")

		if !force {
			fmt.Printf("Are you sure you want to trigger failover from '%s'?\n", cluster)
			fmt.Print("Type 'yes' to confirm: ")
			var confirm string
			fmt.Scanln(&confirm)
			if confirm != "yes" {
				fmt.Println("Aborted.")
				return nil
			}
		}

		fmt.Printf("Triggering failover from '%s'...\n", cluster)
		if target != "" {
			fmt.Printf("Target cluster: %s\n", target)
		} else {
			fmt.Println("Selecting best available target...")
		}
		fmt.Println()
		fmt.Println("Steps:")
		fmt.Println("  1. Identifying workloads...")
		fmt.Println("  2. Migrating workloads...")
		fmt.Println("  3. Redirecting traffic...")
		fmt.Println("  4. Updating DNS...")
		fmt.Println()
		fmt.Printf("Failover complete. Traffic redirected to '%s'.\n", target)

		return nil
	},
}

var fedFailoverTestCmd = &cobra.Command{
	Use:   "test [cluster]",
	Short: "Test failover for a cluster (dry run)",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		cluster := args[0]

		fmt.Printf("Testing failover from '%s'...\n", cluster)
		fmt.Println()
		fmt.Println("Validation Results:")
		fmt.Println("  [OK] Secondary clusters available")
		fmt.Println("  [OK] Workload manifests synced")
		fmt.Println("  [OK] DNS failover configured")
		fmt.Println("  [OK] Load balancer health checks passing")
		fmt.Println()
		fmt.Println("Estimated failover time: 45 seconds")
		fmt.Println("Target cluster: prod-eu-west")
		fmt.Println()
		fmt.Println("Failover test PASSED. System is ready for failover.")

		return nil
	},
}

var fedSummaryCmd = &cobra.Command{
	Use:   "summary",
	Short: "Show federation summary",
	RunE: func(cmd *cobra.Command, args []string) error {
		summary := getSampleFederationSummary()

		fmt.Println("Federation Summary")
		fmt.Println("==================")
		fmt.Printf("Total Clusters:      %d\n", summary.TotalClusters)
		fmt.Printf("Healthy Clusters:    %d\n", summary.HealthyClusters)
		fmt.Printf("Total Federations:   %d\n", summary.TotalFederations)
		fmt.Println()
		fmt.Println("Clusters by Status:")
		for status, count := range summary.ByStatus {
			fmt.Printf("  %s: %d\n", status, count)
		}
		fmt.Println()
		fmt.Println("Clusters by Region:")
		for region, count := range summary.ByRegion {
			fmt.Printf("  %s: %d\n", region, count)
		}

		return nil
	},
}

// Types for CLI display
type federatedClusterInfo struct {
	Name        string `json:"name"`
	Provider    string `json:"provider"`
	Region      string `json:"region"`
	Role        string `json:"role"`
	Status      string `json:"status"`
	Nodes       int    `json:"nodes"`
	LastHealthy string `json:"lastHealthy"`
}

type clusterHealthInfo struct {
	Status       string  `json:"status"`
	Latency      string  `json:"latency"`
	TotalNodes   int     `json:"totalNodes"`
	HealthyNodes int     `json:"healthyNodes"`
	CPUUsage     float64 `json:"cpuUsage"`
	MemoryUsage  float64 `json:"memoryUsage"`
	DiskUsage    float64 `json:"diskUsage"`
	LastCheck    string  `json:"lastCheck"`
}

type federationInfo struct {
	Name         string `json:"name"`
	ClusterCount int    `json:"clusterCount"`
	Failover     string `json:"failover"`
	Traffic      string `json:"traffic"`
	Created      string `json:"created"`
}

type trafficDistributionInfo struct {
	Service  string               `json:"service"`
	Policy   string               `json:"policy"`
	Clusters []clusterTrafficInfo `json:"clusters"`
}

type clusterTrafficInfo struct {
	Name      string  `json:"name"`
	Weight    int     `json:"weight"`
	Latency   string  `json:"latency"`
	ErrorRate float64 `json:"errorRate"`
	RPS       float64 `json:"rps"`
}

type federationSummaryInfo struct {
	TotalClusters    int            `json:"totalClusters"`
	HealthyClusters  int            `json:"healthyClusters"`
	TotalFederations int            `json:"totalFederations"`
	ByStatus         map[string]int `json:"byStatus"`
	ByRegion         map[string]int `json:"byRegion"`
}

type syncPolicyInfo struct {
	Name        string `json:"name"`
	Source      string `json:"source"`
	TargetCount int    `json:"targetCount"`
	Mode        string `json:"mode"`
	Status      string `json:"status"`
	LastSync    string `json:"lastSync"`
}

// Sample data functions
func getSampleFederatedClusters() []federatedClusterInfo {
	return []federatedClusterInfo{
		{Name: "prod-us-east", Provider: "aws", Region: "us-east-1", Role: "primary", Status: "healthy", Nodes: 12, LastHealthy: "30s ago"},
		{Name: "prod-eu-west", Provider: "gcp", Region: "europe-west1", Role: "secondary", Status: "healthy", Nodes: 8, LastHealthy: "30s ago"},
		{Name: "prod-ap-southeast", Provider: "azure", Region: "southeastasia", Role: "secondary", Status: "degraded", Nodes: 6, LastHealthy: "5m ago"},
	}
}

func getSampleClusterHealth(name string) clusterHealthInfo {
	return clusterHealthInfo{
		Status:       "healthy",
		Latency:      "45ms",
		TotalNodes:   12,
		HealthyNodes: 12,
		CPUUsage:     42.5,
		MemoryUsage:  65.2,
		DiskUsage:    38.1,
		LastCheck:    "30s ago",
	}
}

func getSampleFederations() []federationInfo {
	return []federationInfo{
		{Name: "global-prod", ClusterCount: 3, Failover: "automatic", Traffic: "weighted", Created: "30d ago"},
		{Name: "regional-staging", ClusterCount: 2, Failover: "manual", Traffic: "failover", Created: "15d ago"},
	}
}

func getSampleTrafficDistribution(service string) trafficDistributionInfo {
	return trafficDistributionInfo{
		Service: service,
		Policy:  "weighted",
		Clusters: []clusterTrafficInfo{
			{Name: "prod-us-east", Weight: 50, Latency: "12ms", ErrorRate: 0.001, RPS: 1500},
			{Name: "prod-eu-west", Weight: 30, Latency: "45ms", ErrorRate: 0.002, RPS: 900},
			{Name: "prod-ap-southeast", Weight: 20, Latency: "120ms", ErrorRate: 0.003, RPS: 600},
		},
	}
}

func getSampleFederationSummary() federationSummaryInfo {
	return federationSummaryInfo{
		TotalClusters:    5,
		HealthyClusters:  4,
		TotalFederations: 2,
		ByStatus: map[string]int{
			"healthy":  4,
			"degraded": 1,
		},
		ByRegion: map[string]int{
			"us-east-1":     2,
			"europe-west1":  2,
			"southeastasia": 1,
		},
	}
}

func getSampleSyncPolicies() []syncPolicyInfo {
	return []syncPolicyInfo{
		{Name: "config-sync", Source: "prod-us-east", TargetCount: 2, Mode: "push", Status: "synced", LastSync: "2m ago"},
		{Name: "secrets-sync", Source: "prod-us-east", TargetCount: 2, Mode: "push", Status: "synced", LastSync: "5m ago"},
	}
}

func init() {
	// Main federation command
	rootCmd.AddCommand(federationCmd)

	// Cluster subcommands
	federationCmd.AddCommand(fedClusterCmd)
	fedClusterCmd.AddCommand(fedClusterListCmd)
	fedClusterListCmd.Flags().StringP("output", "o", "table", "Output format (table, json)")
	fedClusterListCmd.Flags().String("region", "", "Filter by region")

	fedClusterCmd.AddCommand(fedClusterRegisterCmd)
	fedClusterRegisterCmd.Flags().StringP("name", "n", "", "Cluster name (required)")
	fedClusterRegisterCmd.Flags().StringP("provider", "p", "", "Cloud provider (aws, gcp, azure, on-prem) (required)")
	fedClusterRegisterCmd.Flags().StringP("region", "r", "", "Region (required)")
	fedClusterRegisterCmd.Flags().String("role", "secondary", "Cluster role (primary, secondary, standby, edge)")
	fedClusterRegisterCmd.Flags().String("endpoint", "", "Cluster API endpoint")
	fedClusterRegisterCmd.Flags().String("kubeconfig", "", "Path to kubeconfig file")

	fedClusterCmd.AddCommand(fedClusterUnregisterCmd)
	fedClusterUnregisterCmd.Flags().BoolP("force", "f", false, "Force unregister without confirmation")
	fedClusterUnregisterCmd.Flags().Bool("drain", false, "Drain workloads before unregistering")

	fedClusterCmd.AddCommand(fedClusterHealthCmd)

	// Create federation
	federationCmd.AddCommand(fedCreateCmd)
	fedCreateCmd.Flags().StringP("name", "n", "", "Federation name (required)")
	fedCreateCmd.Flags().StringSlice("clusters", nil, "Cluster names (required)")
	fedCreateCmd.Flags().String("failover", "automatic", "Failover policy (automatic, manual)")
	fedCreateCmd.Flags().String("traffic", "weighted", "Traffic policy (weighted, latency, geo, failover)")

	// List federations
	federationCmd.AddCommand(fedListCmd)
	fedListCmd.Flags().StringP("output", "o", "table", "Output format (table, json)")

	// Traffic subcommands
	federationCmd.AddCommand(fedTrafficCmd)
	fedTrafficCmd.AddCommand(fedTrafficGetCmd)
	fedTrafficCmd.AddCommand(fedTrafficSetCmd)
	fedTrafficSetCmd.Flags().String("policy", "weighted", "Traffic policy (weighted, latency, geo, failover)")
	fedTrafficSetCmd.Flags().StringToInt("weights", nil, "Cluster weights (e.g., cluster1=50,cluster2=50)")

	fedTrafficCmd.AddCommand(fedTrafficMigrateCmd)
	fedTrafficMigrateCmd.Flags().String("from", "", "Source cluster (required)")
	fedTrafficMigrateCmd.Flags().String("to", "", "Target cluster (required)")
	fedTrafficMigrateCmd.Flags().Bool("gradual", true, "Perform gradual migration")

	// Failover subcommands
	federationCmd.AddCommand(fedFailoverCmd)
	fedFailoverCmd.AddCommand(fedFailoverTriggerCmd)
	fedFailoverTriggerCmd.Flags().String("target", "", "Target cluster for failover")
	fedFailoverTriggerCmd.Flags().BoolP("force", "f", false, "Force failover without confirmation")

	fedFailoverCmd.AddCommand(fedFailoverTestCmd)

	// Summary
	federationCmd.AddCommand(fedSummaryCmd)

	// Sync subcommands
	federationCmd.AddCommand(fedSyncCmd)
	fedSyncCmd.AddCommand(fedSyncCreateCmd)
	fedSyncCreateCmd.Flags().StringP("file", "f", "", "Path to sync policy file")
	fedSyncCreateCmd.Flags().StringP("name", "n", "", "Sync policy name")
	fedSyncCreateCmd.Flags().StringP("source", "s", "", "Source cluster")
	fedSyncCreateCmd.Flags().StringSlice("targets", nil, "Target clusters")
	fedSyncCreateCmd.Flags().StringSlice("resources", []string{"ConfigMap", "Secret"}, "Resource types to sync")
	fedSyncCreateCmd.Flags().String("mode", "push", "Sync mode (push, pull, bidirectional)")

	fedSyncCmd.AddCommand(fedSyncListCmd)
	fedSyncListCmd.Flags().StringP("output", "o", "table", "Output format (table, json)")

	fedSyncCmd.AddCommand(fedSyncStatusCmd)
	fedSyncCmd.AddCommand(fedSyncTriggerCmd)
	fedSyncTriggerCmd.Flags().BoolP("wait", "w", false, "Wait for sync to complete")

	fedSyncCmd.AddCommand(fedSyncDeleteCmd)
	fedSyncDeleteCmd.Flags().BoolP("force", "f", false, "Force delete without confirmation")

	fedSyncCmd.AddCommand(fedSyncConflictsCmd)
}
