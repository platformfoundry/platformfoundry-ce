package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/platformfoundry/pf-ce/internal/score"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

var scoreCmd = &cobra.Command{
	Use:   "score",
	Short: "Score workload specification commands",
	Long:  `Manage Score workload specifications - validate, generate, and convert.`,
}

var scoreValidateCmd = &cobra.Command{
	Use:   "validate [file]",
	Short: "Validate a Score specification file",
	Long:  `Validate a Score workload specification file for correctness.`,
	Args:  cobra.ExactArgs(1),
	RunE:  runScoreValidate,
}

var scoreGenerateCmd = &cobra.Command{
	Use:   "generate [file]",
	Short: "Generate platform manifests from Score spec",
	Long:  `Generate Kubernetes, Docker Compose, or Helm manifests from a Score specification.`,
	Args:  cobra.ExactArgs(1),
	RunE:  runScoreGenerate,
}

var scoreConvertCmd = &cobra.Command{
	Use:   "convert [file]",
	Short: "Convert existing manifests to Score format",
	Long:  `Convert existing Kubernetes or Docker Compose manifests to Score specification.`,
	Args:  cobra.ExactArgs(1),
	RunE:  runScoreConvert,
}

var scoreResourcesCmd = &cobra.Command{
	Use:   "resources",
	Short: "List available resource types",
	Long:  `List all available resource types that can be used in Score specifications.`,
	RunE:  runScoreResources,
}

var scoreInitCmd = &cobra.Command{
	Use:   "init [name]",
	Short: "Initialize a new Score specification",
	Long:  `Create a new Score specification file with common defaults.`,
	Args:  cobra.MaximumNArgs(1),
	RunE:  runScoreInit,
}

// Flags
var (
	scoreTarget      string
	scoreOutput      string
	scoreFormat      string
	scoreEnvironment string
	scoreStrict      bool
)

func init() {
	// Validate flags
	scoreValidateCmd.Flags().BoolVar(&scoreStrict, "strict", false, "Treat warnings as errors")
	scoreValidateCmd.Flags().StringVar(&scoreFormat, "format", "text", "Output format (text, json)")

	// Generate flags
	scoreGenerateCmd.Flags().StringVarP(&scoreTarget, "target", "t", "kubernetes", "Target platform (kubernetes, compose, helm)")
	scoreGenerateCmd.Flags().StringVarP(&scoreOutput, "output", "o", "", "Output file or directory")
	scoreGenerateCmd.Flags().StringVarP(&scoreEnvironment, "env", "e", "development", "Environment name")
	scoreGenerateCmd.Flags().StringVar(&scoreFormat, "format", "yaml", "Output format (yaml, json)")

	// Convert flags
	scoreConvertCmd.Flags().StringVarP(&scoreOutput, "output", "o", "", "Output file")

	// Init flags
	scoreInitCmd.Flags().StringVarP(&scoreOutput, "output", "o", "score.yaml", "Output file")

	// Resources flags
	scoreResourcesCmd.Flags().StringVar(&scoreFormat, "format", "table", "Output format (table, json, yaml)")

	scoreCmd.AddCommand(scoreValidateCmd)
	scoreCmd.AddCommand(scoreGenerateCmd)
	scoreCmd.AddCommand(scoreConvertCmd)
	scoreCmd.AddCommand(scoreResourcesCmd)
	scoreCmd.AddCommand(scoreInitCmd)
}

func runScoreValidate(cmd *cobra.Command, args []string) error {
	parser := score.NewParser()

	data, err := os.ReadFile(args[0])
	if err != nil {
		return fmt.Errorf("failed to read file: %w", err)
	}

	workload, errors, err := parser.ParseAndValidate(data)
	if err != nil {
		return fmt.Errorf("failed to parse: %w", err)
	}

	if scoreFormat == "json" {
		result := struct {
			Valid    bool                     `json:"valid"`
			Workload *score.Workload          `json:"workload,omitempty"`
			Errors   []score.ValidationError  `json:"errors,omitempty"`
		}{
			Valid:    len(errors) == 0 || (!scoreStrict && !hasErrors(errors)),
			Workload: workload,
			Errors:   errors,
		}
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(result)
	}

	// Text output
	fmt.Printf("Validating: %s\n", args[0])
	fmt.Printf("Workload: %s\n", workload.Metadata.Name)
	fmt.Printf("API Version: %s\n", workload.APIVersion)
	fmt.Printf("Containers: %d\n", len(workload.Containers))
	fmt.Printf("Resources: %d\n", len(workload.Resources))
	fmt.Println()

	if len(errors) == 0 {
		fmt.Println("✓ Validation passed")
		return nil
	}

	fmt.Println("Validation issues:")
	errorCount := 0
	warningCount := 0

	for _, e := range errors {
		prefix := "⚠"
		if e.Severity == "error" {
			prefix = "✗"
			errorCount++
		} else {
			warningCount++
		}
		fmt.Printf("  %s [%s] %s: %s\n", prefix, e.Severity, e.Field, e.Message)
	}

	fmt.Println()
	fmt.Printf("Summary: %d errors, %d warnings\n", errorCount, warningCount)

	if errorCount > 0 || (scoreStrict && warningCount > 0) {
		return fmt.Errorf("validation failed")
	}

	fmt.Println("✓ Validation passed (with warnings)")
	return nil
}

func hasErrors(errors []score.ValidationError) bool {
	for _, e := range errors {
		if e.Severity == "error" {
			return true
		}
	}
	return false
}

func runScoreGenerate(cmd *cobra.Command, args []string) error {
	parser := score.NewParser()

	workload, err := parser.ParseFile(args[0])
	if err != nil {
		return fmt.Errorf("failed to parse file: %w", err)
	}

	// Validate first
	errors := parser.Validate(workload)
	if hasErrors(errors) {
		fmt.Println("Validation errors:")
		for _, e := range errors {
			if e.Severity == "error" {
				fmt.Printf("  ✗ %s: %s\n", e.Field, e.Message)
			}
		}
		return fmt.Errorf("fix validation errors before generating")
	}

	// Translate
	translator := score.NewTranslator(parser, scoreEnvironment)

	var target score.TranslationTarget
	switch strings.ToLower(scoreTarget) {
	case "kubernetes", "k8s":
		target = score.TargetKubernetes
	case "compose", "docker-compose":
		target = score.TargetCompose
	case "helm":
		target = score.TargetHelm
	default:
		return fmt.Errorf("unknown target: %s", scoreTarget)
	}

	result, err := translator.Translate(workload, target)
	if err != nil {
		return fmt.Errorf("translation failed: %w", err)
	}

	// Output warnings
	for _, warning := range result.Warnings {
		fmt.Fprintf(os.Stderr, "Warning: %s\n", warning)
	}

	// Output manifests
	if scoreOutput != "" {
		return writeManifests(result, scoreOutput)
	}

	// Print to stdout
	return printManifests(result)
}

func writeManifests(result *score.TranslationResult, outputPath string) error {
	// Check if output is a directory
	info, err := os.Stat(outputPath)
	isDir := err == nil && info.IsDir()

	if isDir || strings.HasSuffix(outputPath, string(os.PathSeparator)) {
		// Write each manifest to a separate file
		if err := os.MkdirAll(outputPath, 0755); err != nil {
			return err
		}

		for name, manifest := range result.Manifests {
			filename := filepath.Join(outputPath, name+".yaml")
			data, err := yaml.Marshal(manifest)
			if err != nil {
				return err
			}
			if err := os.WriteFile(filename, data, 0644); err != nil {
				return err
			}
			fmt.Printf("Generated: %s\n", filename)
		}

		for name, resource := range result.Resources {
			filename := filepath.Join(outputPath, "resource-"+name+".yaml")
			data, err := yaml.Marshal(resource)
			if err != nil {
				return err
			}
			if err := os.WriteFile(filename, data, 0644); err != nil {
				return err
			}
			fmt.Printf("Generated: %s\n", filename)
		}
	} else {
		// Write all manifests to a single file
		f, err := os.Create(outputPath)
		if err != nil {
			return err
		}
		defer f.Close()

		enc := yaml.NewEncoder(f)
		for _, manifest := range result.Manifests {
			if err := enc.Encode(manifest); err != nil {
				return err
			}
		}
		enc.Close()
		fmt.Printf("Generated: %s\n", outputPath)
	}

	return nil
}

func printManifests(result *score.TranslationResult) error {
	enc := yaml.NewEncoder(os.Stdout)
	defer enc.Close()

	for name, manifest := range result.Manifests {
		fmt.Printf("# %s\n", name)
		if err := enc.Encode(manifest); err != nil {
			return err
		}
		fmt.Println("---")
	}

	return nil
}

func runScoreConvert(cmd *cobra.Command, args []string) error {
	// Read input file
	data, err := os.ReadFile(args[0])
	if err != nil {
		return fmt.Errorf("failed to read file: %w", err)
	}

	// Detect format and convert
	var workload *score.Workload

	// Try parsing as Kubernetes manifest
	var k8sManifest map[string]interface{}
	if err := yaml.Unmarshal(data, &k8sManifest); err == nil {
		if kind, ok := k8sManifest["kind"].(string); ok {
			if kind == "Deployment" || kind == "Pod" {
				workload, err = convertK8sToScore(k8sManifest)
				if err != nil {
					return fmt.Errorf("failed to convert Kubernetes manifest: %w", err)
				}
			}
		}
	}

	// Try parsing as Docker Compose
	if workload == nil {
		var composeFile map[string]interface{}
		if err := yaml.Unmarshal(data, &composeFile); err == nil {
			if _, ok := composeFile["services"]; ok {
				workload, err = convertComposeToScore(composeFile)
				if err != nil {
					return fmt.Errorf("failed to convert Docker Compose: %w", err)
				}
			}
		}
	}

	if workload == nil {
		return fmt.Errorf("could not detect input format (supported: Kubernetes Deployment, Docker Compose)")
	}

	// Output
	output := os.Stdout
	if scoreOutput != "" {
		f, err := os.Create(scoreOutput)
		if err != nil {
			return err
		}
		defer f.Close()
		output = f
	}

	enc := yaml.NewEncoder(output)
	if err := enc.Encode(workload); err != nil {
		return err
	}

	if scoreOutput != "" {
		fmt.Printf("Converted to: %s\n", scoreOutput)
	}

	return nil
}

func convertK8sToScore(manifest map[string]interface{}) (*score.Workload, error) {
	workload := &score.Workload{
		APIVersion: "score.dev/v1b1",
		Containers: make(map[string]score.Container),
		Resources:  make(map[string]score.Resource),
	}

	// Extract metadata
	if metadata, ok := manifest["metadata"].(map[string]interface{}); ok {
		if name, ok := metadata["name"].(string); ok {
			workload.Metadata.Name = name
		}
		if labels, ok := metadata["labels"].(map[string]interface{}); ok {
			workload.Metadata.Labels = make(map[string]string)
			for k, v := range labels {
				if s, ok := v.(string); ok {
					workload.Metadata.Labels[k] = s
				}
			}
		}
	}

	// Extract containers from spec
	if spec, ok := manifest["spec"].(map[string]interface{}); ok {
		var containers []interface{}

		// Handle Deployment
		if template, ok := spec["template"].(map[string]interface{}); ok {
			if podSpec, ok := template["spec"].(map[string]interface{}); ok {
				if c, ok := podSpec["containers"].([]interface{}); ok {
					containers = c
				}
			}
		}

		// Handle Pod
		if c, ok := spec["containers"].([]interface{}); ok {
			containers = c
		}

		for _, c := range containers {
			if container, ok := c.(map[string]interface{}); ok {
				name, _ := container["name"].(string)
				if name == "" {
					name = "main"
				}

				sc := score.Container{}
				if image, ok := container["image"].(string); ok {
					sc.Image = image
				}
				if command, ok := container["command"].([]interface{}); ok {
					for _, cmd := range command {
						if s, ok := cmd.(string); ok {
							sc.Command = append(sc.Command, s)
						}
					}
				}

				// Extract env vars
				if env, ok := container["env"].([]interface{}); ok {
					sc.Variables = make(map[string]string)
					for _, e := range env {
						if envVar, ok := e.(map[string]interface{}); ok {
							envName, _ := envVar["name"].(string)
							envValue, _ := envVar["value"].(string)
							if envName != "" {
								sc.Variables[envName] = envValue
							}
						}
					}
				}

				// Extract resources
				if resources, ok := container["resources"].(map[string]interface{}); ok {
					sc.Resources = &score.ResourceRequirements{}
					if requests, ok := resources["requests"].(map[string]interface{}); ok {
						sc.Resources.Requests = &score.ResourceList{}
						if cpu, ok := requests["cpu"].(string); ok {
							sc.Resources.Requests.CPU = cpu
						}
						if mem, ok := requests["memory"].(string); ok {
							sc.Resources.Requests.Memory = mem
						}
					}
				}

				workload.Containers[name] = sc
			}
		}
	}

	return workload, nil
}

func convertComposeToScore(compose map[string]interface{}) (*score.Workload, error) {
	workload := &score.Workload{
		APIVersion: "score.dev/v1b1",
		Containers: make(map[string]score.Container),
		Resources:  make(map[string]score.Resource),
	}

	services, ok := compose["services"].(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("no services found in compose file")
	}

	// Use first service as main workload
	for name, svc := range services {
		service, ok := svc.(map[string]interface{})
		if !ok {
			continue
		}

		// Check if this is a resource (database, cache, etc.)
		if image, ok := service["image"].(string); ok {
			if resourceType := detectResourceType(image); resourceType != "" {
				workload.Resources[name] = score.Resource{
					Type: resourceType,
				}
				continue
			}
		}

		// It's an application container
		if workload.Metadata.Name == "" {
			workload.Metadata.Name = name
		}

		container := score.Container{}
		if image, ok := service["image"].(string); ok {
			container.Image = image
		}
		if command, ok := service["command"].([]interface{}); ok {
			for _, cmd := range command {
				if s, ok := cmd.(string); ok {
					container.Command = append(container.Command, s)
				}
			}
		}
		if command, ok := service["command"].(string); ok {
			container.Command = strings.Fields(command)
		}

		// Environment
		container.Variables = make(map[string]string)
		if env, ok := service["environment"].([]interface{}); ok {
			for _, e := range env {
				if s, ok := e.(string); ok {
					parts := strings.SplitN(s, "=", 2)
					if len(parts) == 2 {
						container.Variables[parts[0]] = parts[1]
					}
				}
			}
		}
		if env, ok := service["environment"].(map[string]interface{}); ok {
			for k, v := range env {
				if s, ok := v.(string); ok {
					container.Variables[k] = s
				}
			}
		}

		workload.Containers[name] = container
	}

	return workload, nil
}

func detectResourceType(image string) string {
	image = strings.ToLower(image)
	switch {
	case strings.Contains(image, "postgres"):
		return "postgres"
	case strings.Contains(image, "mysql"):
		return "mysql"
	case strings.Contains(image, "mariadb"):
		return "mysql"
	case strings.Contains(image, "mongo"):
		return "mongodb"
	case strings.Contains(image, "redis"):
		return "redis"
	case strings.Contains(image, "memcached"):
		return "memcached"
	case strings.Contains(image, "rabbitmq"):
		return "rabbitmq"
	case strings.Contains(image, "kafka"):
		return "kafka"
	case strings.Contains(image, "elasticsearch"):
		return "elasticsearch"
	}
	return ""
}

func runScoreResources(cmd *cobra.Command, args []string) error {
	parser := score.NewParser()
	types := parser.ListResourceTypes()

	if scoreFormat == "json" {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(types)
	}

	if scoreFormat == "yaml" {
		enc := yaml.NewEncoder(os.Stdout)
		return enc.Encode(types)
	}

	// Table format
	fmt.Printf("%-15s %-25s %s\n", "TYPE", "NAME", "OUTPUTS")
	fmt.Println(strings.Repeat("-", 80))

	for _, rt := range types {
		outputs := make([]string, 0, len(rt.Outputs))
		for name := range rt.Outputs {
			outputs = append(outputs, name)
		}
		outputStr := strings.Join(outputs, ", ")
		if len(outputStr) > 35 {
			outputStr = outputStr[:32] + "..."
		}
		fmt.Printf("%-15s %-25s %s\n", rt.Type, rt.Name, outputStr)
	}

	return nil
}

func runScoreInit(cmd *cobra.Command, args []string) error {
	name := "my-workload"
	if len(args) > 0 {
		name = args[0]
	}

	workload := &score.Workload{
		APIVersion: "score.dev/v1b1",
		Metadata: score.WorkloadMetadata{
			Name: name,
		},
		Containers: map[string]score.Container{
			"main": {
				Image: "nginx:latest",
				Variables: map[string]string{
					"LOG_LEVEL": "info",
				},
				Resources: &score.ResourceRequirements{
					Requests: &score.ResourceList{
						CPU:    "100m",
						Memory: "128Mi",
					},
				},
				LivenessProbe: &score.Probe{
					HTTPGet: &score.HTTPGetAction{
						Path: "/health",
						Port: 8080,
					},
				},
			},
		},
		Service: &score.ServiceSpec{
			Ports: []score.ServicePort{
				{
					Port:       8080,
					TargetPort: 8080,
				},
			},
		},
		Resources: map[string]score.Resource{
			"db": {
				Type: "postgres",
				Params: map[string]interface{}{
					"version": "15",
				},
			},
		},
	}

	output := os.Stdout
	if scoreOutput != "" && scoreOutput != "-" {
		f, err := os.Create(scoreOutput)
		if err != nil {
			return err
		}
		defer f.Close()
		output = f
	}

	// Add header comment
	fmt.Fprintln(output, "# Score Workload Specification")
	fmt.Fprintln(output, "# https://score.dev")
	fmt.Fprintln(output, "")

	enc := yaml.NewEncoder(output)
	enc.SetIndent(2)
	if err := enc.Encode(workload); err != nil {
		return err
	}

	if scoreOutput != "" && scoreOutput != "-" {
		fmt.Printf("Created: %s\n", scoreOutput)
	}

	return nil
}

// GetScoreCmd returns the score command for registration
func GetScoreCmd() *cobra.Command {
	return scoreCmd
}
