package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/platformfoundry/pf-ce/internal/engine"
	"github.com/platformfoundry/pf-ce/internal/orchestrator"
	"github.com/platformfoundry/pf-ce/internal/plugin"
	"github.com/platformfoundry/pf-ce/internal/plugin/providers"
	"github.com/platformfoundry/pf-ce/internal/state"
	"github.com/platformfoundry/pf-ce/internal/workload"
	"github.com/platformfoundry/pf-ce/pkg/types"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

func init() {
	rootCmd.AddCommand(workloadCmd)

	// Subcommands
	workloadCmd.AddCommand(workloadApplyCmd)
	workloadCmd.AddCommand(workloadPlanCmd)
	workloadCmd.AddCommand(workloadValidateCmd)
	workloadCmd.AddCommand(workloadGenerateCmd)
	workloadCmd.AddCommand(workloadListCmd)
	workloadCmd.AddCommand(workloadDescribeCmd)

	// Flags for apply
	workloadApplyCmd.Flags().StringP("file", "f", "", "Workload specification file")
	workloadApplyCmd.Flags().StringP("environment", "e", "", "Target environment")
	workloadApplyCmd.Flags().Bool("dry-run", false, "Show what would be created without applying")
	workloadApplyCmd.MarkFlagRequired("file")

	// Flags for plan
	workloadPlanCmd.Flags().StringP("file", "f", "", "Workload specification file")
	workloadPlanCmd.Flags().StringP("output", "o", "table", "Output format: table, json, yaml")
	workloadPlanCmd.MarkFlagRequired("file")

	// Flags for validate
	workloadValidateCmd.Flags().StringP("file", "f", "", "Workload specification file")
	workloadValidateCmd.MarkFlagRequired("file")

	// Flags for generate
	workloadGenerateCmd.Flags().StringP("file", "f", "", "Workload specification file")
	workloadGenerateCmd.Flags().StringP("output", "o", "yaml", "Output format: yaml, json")
	workloadGenerateCmd.Flags().String("output-dir", "", "Output directory for generated files")
	workloadGenerateCmd.Flags().Bool("kubernetes", true, "Generate Kubernetes manifests")
	workloadGenerateCmd.Flags().Bool("terraform", true, "Generate Terraform configurations")
	workloadGenerateCmd.MarkFlagRequired("file")

	// Flags for list
	workloadListCmd.Flags().StringP("team", "t", "", "Filter by team")
	workloadListCmd.Flags().StringP("environment", "e", "", "Filter by environment")
	workloadListCmd.Flags().StringP("output", "o", "table", "Output format: table, json, yaml")

	// Flags for describe
	workloadDescribeCmd.Flags().StringP("output", "o", "yaml", "Output format: yaml, json")
}

var workloadCmd = &cobra.Command{
	Use:   "workload",
	Short: "Manage workload specifications",
	Long: `Workloads are developer-friendly abstractions that define applications
without requiring infrastructure knowledge. Platform Foundry translates
workloads into platform resources (Kubernetes manifests, infrastructure, etc.).

Use workloads to:
- Define your application containers and dependencies
- Specify scaling and networking requirements
- Let Platform Foundry handle the infrastructure details`,
	Example: `  # Apply a workload specification
  pf workload apply -f my-workload.yaml

  # Preview what will be created
  pf workload plan -f my-workload.yaml

  # Generate Kubernetes manifests
  pf workload generate -f my-workload.yaml

  # Validate a workload file
  pf workload validate -f my-workload.yaml

  # List all workloads
  pf workload list`,
}

var workloadApplyCmd = &cobra.Command{
	Use:   "apply",
	Short: "Apply a workload specification",
	Long: `Apply a workload specification to provision the application and its dependencies.
This will translate the workload spec to Kubernetes resources and infrastructure,
then provision them through the appropriate plugins.`,
	Example: `  # Apply a workload
  pf workload apply -f my-workload.yaml

  # Apply to a specific environment
  pf workload apply -f my-workload.yaml -e production

  # Dry run - show what would be created
  pf workload apply -f my-workload.yaml --dry-run`,
	RunE: runWorkloadApply,
}

var workloadPlanCmd = &cobra.Command{
	Use:   "plan",
	Short: "Show what resources will be created for a workload",
	Long:  `Generate and display an execution plan for a workload without applying it.`,
	Example: `  # Show execution plan
  pf workload plan -f my-workload.yaml

  # Output as JSON
  pf workload plan -f my-workload.yaml -o json`,
	RunE: runWorkloadPlan,
}

var workloadValidateCmd = &cobra.Command{
	Use:   "validate",
	Short: "Validate a workload specification",
	Long:  `Validate a workload specification file for correctness without applying it.`,
	Example: `  # Validate a workload file
  pf workload validate -f my-workload.yaml`,
	RunE: runWorkloadValidate,
}

var workloadGenerateCmd = &cobra.Command{
	Use:   "generate",
	Short: "Generate platform resources from a workload",
	Long: `Generate Kubernetes manifests and infrastructure configurations from
a workload specification without applying them.`,
	Example: `  # Generate and print to stdout
  pf workload generate -f my-workload.yaml

  # Generate to files
  pf workload generate -f my-workload.yaml --output-dir ./generated

  # Generate only Kubernetes manifests
  pf workload generate -f my-workload.yaml --terraform=false`,
	RunE: runWorkloadGenerate,
}

var workloadListCmd = &cobra.Command{
	Use:   "list",
	Short: "List workloads",
	Long:  `List all workloads that have been applied to the platform.`,
	Example: `  # List all workloads
  pf workload list

  # List workloads for a team
  pf workload list --team payments

  # List workloads in an environment
  pf workload list -e production`,
	RunE: runWorkloadList,
}

var workloadDescribeCmd = &cobra.Command{
	Use:   "describe [name]",
	Short: "Describe a workload",
	Long:  `Show detailed information about a specific workload.`,
	Example: `  # Describe a workload
  pf workload describe order-service`,
	Args: cobra.ExactArgs(1),
	RunE: runWorkloadDescribe,
}

func runWorkloadApply(cmd *cobra.Command, args []string) error {
	file, _ := cmd.Flags().GetString("file")
	env, _ := cmd.Flags().GetString("environment")
	dryRun, _ := cmd.Flags().GetBool("dry-run")

	// Load workload
	w, err := loadWorkload(file)
	if err != nil {
		return fmt.Errorf("failed to load workload: %w", err)
	}

	// Override environment if specified
	if env != "" {
		w.Metadata.Environment = env
	}

	// Translate workload
	translator := workload.NewTranslator("aws", "us-east-1", "default")
	result, err := translator.Translate(w)
	if err != nil {
		return fmt.Errorf("failed to translate workload: %w", err)
	}

	if dryRun {
		fmt.Println("Dry run - the following resources would be created:")
		fmt.Println()
		printTranslationSummary(result)
		return nil
	}

	// Initialize plugin manager with builtin providers
	pluginManager := plugin.NewManager()
	if err := providers.RegisterBuiltins(pluginManager); err != nil {
		return fmt.Errorf("failed to register plugins: %w", err)
	}

	// Initialize state backend
	stateBackend, err := state.NewBboltBackend(getStatePath())
	if err != nil {
		return fmt.Errorf("failed to initialize state backend: %w", err)
	}
	defer stateBackend.Close()

	// Create orchestrator service
	svc := orchestrator.NewService(orchestrator.Config{
		MaxParallel:       4,
		Timeout:           30 * time.Minute,
		RollbackOnFailure: true,
	}, pluginManager, stateBackend)

	// Subscribe to events for progress output
	progressReporter := newProgressReporter()
	svc.Subscribe(progressReporter)

	// Create context with timeout
	ctx, cancel := context.WithTimeout(cmd.Context(), 30*time.Minute)
	defer cancel()

	fmt.Printf("Applying workload: %s\n", w.Metadata.Name)
	fmt.Println()
	printTranslationSummary(result)
	fmt.Println()

	// Apply workload via orchestrator
	applyResult, err := svc.ApplyWorkload(ctx, w, result)
	if err != nil {
		fmt.Printf("\nWorkload apply failed: %v\n", err)
		return err
	}

	fmt.Println()
	fmt.Printf("Workload %s applied successfully!\n", applyResult.WorkloadName)
	fmt.Printf("Duration: %s\n", applyResult.Duration.Round(time.Second))
	fmt.Println()

	if len(applyResult.Resources) > 0 {
		fmt.Println("Resources created:")
		for _, res := range applyResult.Resources {
			fmt.Printf("  - %s (%s)\n", res.Name, res.Status)
		}
	}

	if len(applyResult.Outputs) > 0 {
		fmt.Println()
		fmt.Println("Outputs:")
		for k, v := range applyResult.Outputs {
			fmt.Printf("  %s: %v\n", k, v)
		}
	}

	return nil
}

// progressReporter implements engine.EventListener for CLI progress output
type progressReporter struct {
	lastProgress map[string]int
}

func newProgressReporter() *progressReporter {
	return &progressReporter{
		lastProgress: make(map[string]int),
	}
}

func (p *progressReporter) OnEvent(event engine.EngineEvent) {
	switch event.Type {
	case engine.EventTypeProgress:
		if p.lastProgress[event.EngineID] != event.Progress {
			p.lastProgress[event.EngineID] = event.Progress
			fmt.Printf("[%s] %d%% - %s\n", event.Component, event.Progress, event.Message)
		}
	case engine.EventTypeError:
		fmt.Printf("[%s] ERROR: %v\n", event.Component, event.Error)
	case engine.EventTypeLog:
		fmt.Printf("[%s] %s\n", event.Component, event.Message)
	}
}

func getStatePath() string {
	// Use platform-specific state path
	home, err := os.UserHomeDir()
	if err != nil {
		return ".platformfoundry/state.db"
	}
	stateDir := fmt.Sprintf("%s/.platformfoundry", home)
	os.MkdirAll(stateDir, 0755)
	return fmt.Sprintf("%s/state.db", stateDir)
}

func runWorkloadPlan(cmd *cobra.Command, args []string) error {
	file, _ := cmd.Flags().GetString("file")
	output, _ := cmd.Flags().GetString("output")

	// Load workload
	w, err := loadWorkload(file)
	if err != nil {
		return fmt.Errorf("failed to load workload: %w", err)
	}

	// Translate workload
	translator := workload.NewTranslator("aws", "us-east-1", "default")
	result, err := translator.Translate(w)
	if err != nil {
		return fmt.Errorf("failed to translate workload: %w", err)
	}

	switch output {
	case "json":
		data, err := json.MarshalIndent(result, "", "  ")
		if err != nil {
			return err
		}
		fmt.Println(string(data))
	case "yaml":
		data, err := yaml.Marshal(result)
		if err != nil {
			return err
		}
		fmt.Println(string(data))
	default:
		printWorkloadPlan(w, result)
	}

	return nil
}

func runWorkloadValidate(cmd *cobra.Command, args []string) error {
	file, _ := cmd.Flags().GetString("file")

	// Load workload
	w, err := loadWorkload(file)
	if err != nil {
		return fmt.Errorf("failed to load workload: %w", err)
	}

	// Validate
	if err := w.Validate(); err != nil {
		fmt.Printf("Validation failed: %s\n", err)
		return err
	}

	fmt.Printf("Workload '%s' is valid!\n", w.Metadata.Name)
	fmt.Println()
	fmt.Println("Summary:")
	fmt.Printf("  Name: %s\n", w.Metadata.Name)
	fmt.Printf("  Team: %s\n", w.Metadata.Team)
	fmt.Printf("  Containers: %d\n", len(w.Spec.Containers))
	fmt.Printf("  Dependencies: %d\n", len(w.Spec.Dependencies))
	if w.Spec.Scaling != nil {
		fmt.Printf("  Scaling: %d-%d replicas\n", w.Spec.Scaling.Min, w.Spec.Scaling.Max)
	}
	if w.Spec.Network != nil && w.Spec.Network.Ingress != nil {
		fmt.Printf("  Ingress: %s\n", w.Spec.Network.Ingress.Path)
	}

	return nil
}

func runWorkloadGenerate(cmd *cobra.Command, args []string) error {
	file, _ := cmd.Flags().GetString("file")
	output, _ := cmd.Flags().GetString("output")
	outputDir, _ := cmd.Flags().GetString("output-dir")
	genK8s, _ := cmd.Flags().GetBool("kubernetes")
	genTerraform, _ := cmd.Flags().GetBool("terraform")

	// Load workload
	w, err := loadWorkload(file)
	if err != nil {
		return fmt.Errorf("failed to load workload: %w", err)
	}

	// Translate workload
	translator := workload.NewTranslator("aws", "us-east-1", "default")
	result, err := translator.Translate(w)
	if err != nil {
		return fmt.Errorf("failed to translate workload: %w", err)
	}

	if outputDir != "" {
		// Generate to files
		if err := os.MkdirAll(outputDir, 0755); err != nil {
			return fmt.Errorf("failed to create output directory: %w", err)
		}

		if genK8s {
			k8sYAML, err := result.ToKubernetesYAML()
			if err != nil {
				return fmt.Errorf("failed to generate Kubernetes YAML: %w", err)
			}
			k8sFile := fmt.Sprintf("%s/%s-k8s.yaml", outputDir, w.Metadata.Name)
			if err := os.WriteFile(k8sFile, []byte(k8sYAML), 0644); err != nil {
				return fmt.Errorf("failed to write Kubernetes file: %w", err)
			}
			fmt.Printf("Generated: %s\n", k8sFile)
		}

		if genTerraform && len(result.InfraResources) > 0 {
			tfConfig := generateTerraformConfig(result.InfraResources)
			tfFile := fmt.Sprintf("%s/%s-infra.tf", outputDir, w.Metadata.Name)
			if err := os.WriteFile(tfFile, []byte(tfConfig), 0644); err != nil {
				return fmt.Errorf("failed to write Terraform file: %w", err)
			}
			fmt.Printf("Generated: %s\n", tfFile)
		}

		return nil
	}

	// Output to stdout
	switch output {
	case "json":
		data, err := json.MarshalIndent(result, "", "  ")
		if err != nil {
			return err
		}
		fmt.Println(string(data))
	default:
		if genK8s {
			fmt.Println("# Kubernetes Manifests")
			fmt.Println("# -------------------")
			k8sYAML, err := result.ToKubernetesYAML()
			if err != nil {
				return fmt.Errorf("failed to generate Kubernetes YAML: %w", err)
			}
			fmt.Println(k8sYAML)
		}

		if genTerraform && len(result.InfraResources) > 0 {
			fmt.Println("\n# Terraform Configuration")
			fmt.Println("# ----------------------")
			fmt.Println(generateTerraformConfig(result.InfraResources))
		}
	}

	return nil
}

func runWorkloadList(cmd *cobra.Command, args []string) error {
	team, _ := cmd.Flags().GetString("team")
	env, _ := cmd.Flags().GetString("environment")
	output, _ := cmd.Flags().GetString("output")

	// TODO: Load from state backend
	// For now, show sample data
	workloads := []workloadListItem{
		{Name: "order-service", Team: "orders", Environment: "production", Containers: 1, Dependencies: 3, Status: "running"},
		{Name: "payment-api", Team: "payments", Environment: "production", Containers: 2, Dependencies: 2, Status: "running"},
		{Name: "user-service", Team: "users", Environment: "staging", Containers: 1, Dependencies: 1, Status: "running"},
		{Name: "notification-worker", Team: "platform", Environment: "production", Containers: 1, Dependencies: 2, Status: "running"},
	}

	// Filter
	filtered := []workloadListItem{}
	for _, w := range workloads {
		if team != "" && w.Team != team {
			continue
		}
		if env != "" && w.Environment != env {
			continue
		}
		filtered = append(filtered, w)
	}

	switch output {
	case "json":
		data, err := json.MarshalIndent(filtered, "", "  ")
		if err != nil {
			return err
		}
		fmt.Println(string(data))
	case "yaml":
		data, err := yaml.Marshal(filtered)
		if err != nil {
			return err
		}
		fmt.Println(string(data))
	default:
		fmt.Println()
		fmt.Println("+----------------------+-----------+-------------+------------+------+----------+")
		fmt.Println("| NAME                 | TEAM      | ENVIRONMENT | CONTAINERS | DEPS | STATUS   |")
		fmt.Println("+----------------------+-----------+-------------+------------+------+----------+")
		for _, w := range filtered {
			fmt.Printf("| %-20s | %-9s | %-11s | %-10d | %-4d | %-8s |\n",
				workloadTruncateStr(w.Name, 20), workloadTruncateStr(w.Team, 9), workloadTruncateStr(w.Environment, 11),
				w.Containers, w.Dependencies, w.Status)
		}
		fmt.Println("+----------------------+-----------+-------------+------------+------+----------+")
		fmt.Printf("\nTotal: %d workloads\n", len(filtered))
	}

	return nil
}

func runWorkloadDescribe(cmd *cobra.Command, args []string) error {
	name := args[0]
	output, _ := cmd.Flags().GetString("output")

	// TODO: Load from state backend
	// For now, show sample data
	w := &types.Workload{
		APIVersion: "platformfoundry.io/v1",
		Kind:       "Workload",
		Metadata: types.WorkloadMetadata{
			Name:        name,
			Team:        "orders",
			Environment: "production",
			Labels: map[string]string{
				"tier": "backend",
			},
		},
		Spec: types.WorkloadSpec{
			Containers: []types.Container{
				{
					Name:  "api",
					Image: "order-service:v1.2.3",
					Resources: &types.ContainerResources{
						CPU:    "500m",
						Memory: "512Mi",
					},
					Ports: []types.PortSpec{
						{Name: "http", Port: 8080},
					},
				},
			},
			Dependencies: []types.WorkloadDependency{
				{Type: "postgres", Name: "orders-db", Config: map[string]interface{}{"size": "medium"}},
				{Type: "redis", Name: "orders-cache", Config: map[string]interface{}{"size": "small"}},
			},
			Scaling: &types.ScalingSpec{
				Min:       2,
				Max:       10,
				TargetCPU: 70,
			},
		},
		Status: &types.WorkloadStatus{
			State:             types.WorkloadStateRunning,
			Replicas:          3,
			ReadyReplicas:     3,
			AvailableReplicas: 3,
		},
	}

	switch output {
	case "json":
		data, err := json.MarshalIndent(w, "", "  ")
		if err != nil {
			return err
		}
		fmt.Println(string(data))
	default:
		data, err := yaml.Marshal(w)
		if err != nil {
			return err
		}
		fmt.Println(string(data))
	}

	return nil
}

type workloadListItem struct {
	Name         string `json:"name" yaml:"name"`
	Team         string `json:"team" yaml:"team"`
	Environment  string `json:"environment" yaml:"environment"`
	Containers   int    `json:"containers" yaml:"containers"`
	Dependencies int    `json:"dependencies" yaml:"dependencies"`
	Status       string `json:"status" yaml:"status"`
}

func loadWorkload(file string) (*types.Workload, error) {
	data, err := os.ReadFile(file)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	var w types.Workload
	if err := yaml.Unmarshal(data, &w); err != nil {
		return nil, fmt.Errorf("failed to parse YAML: %w", err)
	}

	return &w, nil
}

func printTranslationSummary(result *workload.TranslationResult) {
	fmt.Println("Translation Summary:")
	fmt.Println(strings.Repeat("-", 50))

	if result.Deployment != nil {
		fmt.Printf("Deployment: %s (%d container(s), %d replica(s))\n",
			result.Deployment.Name, len(result.Deployment.Containers), result.Deployment.Replicas)
	}

	if result.Service != nil {
		fmt.Printf("Service: %s (%s, %d port(s))\n",
			result.Service.Name, result.Service.Type, len(result.Service.Ports))
	}

	if result.HPA != nil {
		fmt.Printf("HPA: %s (min: %d, max: %d)\n",
			result.HPA.Name, result.HPA.MinReplicas, result.HPA.MaxReplicas)
	}

	if result.Ingress != nil {
		tls := "no"
		if result.Ingress.TLS {
			tls = "yes"
		}
		fmt.Printf("Ingress: %s (path: %s, TLS: %s)\n",
			result.Ingress.Name, result.Ingress.Path, tls)
	}

	if len(result.InfraResources) > 0 {
		fmt.Println("\nInfrastructure Resources:")
		for _, res := range result.InfraResources {
			fmt.Printf("  - %s (%s via %s)\n", res.Name, res.Type, res.Provider)
		}
	}

	if len(result.Outputs) > 0 {
		fmt.Println("\nOutputs:")
		for name, out := range result.Outputs {
			fmt.Printf("  - %s (%s): %s\n", name, out.Type, out.Description)
		}
	}
}

func printWorkloadPlan(w *types.Workload, result *workload.TranslationResult) {
	fmt.Printf("\nWorkload Plan: %s\n", w.Metadata.Name)
	fmt.Println(strings.Repeat("=", 65))

	fmt.Println("\nWorkload Info:")
	fmt.Printf("  Name: %s\n", w.Metadata.Name)
	fmt.Printf("  Team: %s\n", w.Metadata.Team)
	if w.Metadata.Environment != "" {
		fmt.Printf("  Environment: %s\n", w.Metadata.Environment)
	}

	fmt.Println("\nResources to be created:")
	fmt.Println(strings.Repeat("-", 50))

	// Kubernetes resources
	fmt.Println("\nKubernetes Resources:")
	if result.Deployment != nil {
		fmt.Printf("  [+] Deployment: %s/%s\n", result.Deployment.Namespace, result.Deployment.Name)
		for _, c := range result.Deployment.Containers {
			fmt.Printf("      Container: %s (%s)\n", c.Name, c.Image)
		}
	}
	if result.Service != nil {
		fmt.Printf("  [+] Service: %s/%s (%s)\n", result.Service.Namespace, result.Service.Name, result.Service.Type)
	}
	if result.HPA != nil {
		fmt.Printf("  [+] HPA: %s/%s (%d-%d replicas)\n", result.HPA.Namespace, result.HPA.Name, result.HPA.MinReplicas, result.HPA.MaxReplicas)
	}
	if result.Ingress != nil {
		fmt.Printf("  [+] Ingress: %s/%s\n", result.Ingress.Namespace, result.Ingress.Name)
	}

	// Infrastructure resources
	if len(result.InfraResources) > 0 {
		fmt.Println("\nInfrastructure Resources:")
		for _, res := range result.InfraResources {
			fmt.Printf("  [+] %s: %s\n", res.Type, res.Name)
			fmt.Printf("      Provider: %s\n", res.Provider)
		}
	}

	// Outputs
	if len(result.Outputs) > 0 {
		fmt.Println("\nOutputs (available after apply):")
		for name, out := range result.Outputs {
			typeStr := out.Type
			if typeStr == "secret" {
				typeStr = "secret (encrypted)"
			}
			fmt.Printf("  - %s [%s]\n", name, typeStr)
		}
	}

	fmt.Println()
	fmt.Println("Run 'pf workload apply -f <file>' to apply this workload.")
}

func generateTerraformConfig(resources []workload.InfraResource) string {
	var sb strings.Builder

	sb.WriteString("# Generated by Platform Foundry\n")
	sb.WriteString("# Terraform configuration for workload infrastructure\n\n")

	for _, res := range resources {
		sb.WriteString(fmt.Sprintf("# Resource: %s\n", res.Name))
		sb.WriteString(fmt.Sprintf("# Type: %s\n", res.Type))

		// Generate resource block based on type
		switch {
		case strings.Contains(res.Type, "rds"):
			sb.WriteString(generateRDSConfig(res))
		case strings.Contains(res.Type, "elasticache"):
			sb.WriteString(generateElasticacheConfig(res))
		case strings.Contains(res.Type, "s3"):
			sb.WriteString(generateS3Config(res))
		case strings.Contains(res.Type, "dynamodb"):
			sb.WriteString(generateDynamoDBConfig(res))
		default:
			// Generic resource placeholder
			sb.WriteString(fmt.Sprintf(`
resource "%s" "%s" {
  # Configuration would go here
}
`, res.Type, res.Name))
		}
		sb.WriteString("\n")
	}

	return sb.String()
}

func generateRDSConfig(res workload.InfraResource) string {
	name := getString(res.Config, "name", res.Name)
	engine := getString(res.Config, "engine", "postgres")
	version := getString(res.Config, "engine_version", "15")
	instanceType := getString(res.Config, "instance_type", "db.t3.small")
	storage := getInt(res.Config, "allocated_storage", 20)

	return fmt.Sprintf(`
resource "aws_db_instance" "%s" {
  identifier           = "%s"
  engine               = "%s"
  engine_version       = "%s"
  instance_class       = "%s"
  allocated_storage    = %d
  skip_final_snapshot  = true

  tags = {
    managed-by = "platformfoundry"
  }
}
`, res.Name, name, engine, version, instanceType, storage)
}

func generateElasticacheConfig(res workload.InfraResource) string {
	name := getString(res.Config, "name", res.Name)
	nodeType := getString(res.Config, "instance_type", "cache.t3.small")

	return fmt.Sprintf(`
resource "aws_elasticache_cluster" "%s" {
  cluster_id           = "%s"
  engine               = "redis"
  node_type            = "%s"
  num_cache_nodes      = 1
  port                 = 6379

  tags = {
    managed-by = "platformfoundry"
  }
}
`, res.Name, name, nodeType)
}

func generateS3Config(res workload.InfraResource) string {
	name := getString(res.Config, "name", res.Name)

	return fmt.Sprintf(`
resource "aws_s3_bucket" "%s" {
  bucket = "%s"

  tags = {
    managed-by = "platformfoundry"
  }
}

resource "aws_s3_bucket_server_side_encryption_configuration" "%s" {
  bucket = aws_s3_bucket.%s.id

  rule {
    apply_server_side_encryption_by_default {
      sse_algorithm = "AES256"
    }
  }
}
`, res.Name, name, res.Name, res.Name)
}

func generateDynamoDBConfig(res workload.InfraResource) string {
	name := getString(res.Config, "name", res.Name)
	billingMode := getString(res.Config, "billing_mode", "PAY_PER_REQUEST")

	return fmt.Sprintf(`
resource "aws_dynamodb_table" "%s" {
  name         = "%s"
  billing_mode = "%s"
  hash_key     = "id"

  attribute {
    name = "id"
    type = "S"
  }

  tags = {
    managed-by = "platformfoundry"
  }
}
`, res.Name, name, billingMode)
}

func getString(m map[string]interface{}, key, defaultVal string) string {
	if v, ok := m[key].(string); ok {
		return v
	}
	return defaultVal
}

func getInt(m map[string]interface{}, key string, defaultVal int) int {
	if v, ok := m[key].(int); ok {
		return v
	}
	if v, ok := m[key].(float64); ok {
		return int(v)
	}
	return defaultVal
}

func workloadTruncateStr(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}
