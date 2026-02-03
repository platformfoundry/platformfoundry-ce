package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"text/tabwriter"

	"github.com/platformfoundry/pf-ce/internal/platform"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

var platformManager *platform.Manager

func init() {
	platformManager = platform.NewManager()
}

var platformCmd = &cobra.Command{
	Use:     "platform",
	Aliases: []string{"plt"},
	Short:   "Manage platform-as-code definitions",
	Long:    `Manage platform definitions, golden paths, and applications.`,
}

var platformApplyCmd = &cobra.Command{
	Use:   "apply -f <file>",
	Short: "Apply a platform configuration",
	RunE: func(cmd *cobra.Command, args []string) error {
		filePath, _ := cmd.Flags().GetString("file")
		if filePath == "" {
			return fmt.Errorf("file path is required (-f)")
		}

		data, err := os.ReadFile(filePath)
		if err != nil {
			return fmt.Errorf("failed to read file: %w", err)
		}

		var plt platform.Platform
		if strings.HasSuffix(filePath, ".json") {
			err = json.Unmarshal(data, &plt)
		} else {
			err = yaml.Unmarshal(data, &plt)
		}
		if err != nil {
			return fmt.Errorf("failed to parse platform: %w", err)
		}

		// Validate
		errors := platformManager.ValidatePlatform(&plt)
		if len(errors) > 0 {
			fmt.Println("Validation errors:")
			for _, e := range errors {
				fmt.Printf("  - %s\n", e)
			}
			return fmt.Errorf("validation failed")
		}

		ctx := context.Background()
		if err := platformManager.RegisterPlatform(ctx, &plt); err != nil {
			return err
		}

		fmt.Printf("Platform '%s' applied successfully\n", plt.Metadata.Name)
		return nil
	},
}

var platformGetCmd = &cobra.Command{
	Use:   "get <name>",
	Short: "Get platform details",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		name := args[0]
		format, _ := cmd.Flags().GetString("output")

		plt, err := platformManager.GetPlatform(name)
		if err != nil {
			return err
		}

		if format == "json" {
			data, _ := json.MarshalIndent(plt, "", "  ")
			fmt.Println(string(data))
			return nil
		}

		if format == "yaml" {
			data, _ := yaml.Marshal(plt)
			fmt.Println(string(data))
			return nil
		}

		fmt.Printf("Name:        %s\n", plt.Metadata.Name)
		if plt.Metadata.Version != "" {
			fmt.Printf("Version:     %s\n", plt.Metadata.Version)
		}
		if plt.Metadata.Description != "" {
			fmt.Printf("Description: %s\n", plt.Metadata.Description)
		}
		fmt.Printf("Status:      %s\n", plt.Status.Phase)

		if len(plt.Spec.GoldenPaths) > 0 {
			fmt.Printf("\nGolden Paths (%d):\n", len(plt.Spec.GoldenPaths))
			for _, gp := range plt.Spec.GoldenPaths {
				fmt.Printf("  - %s (%s/%s)\n", gp.Name, gp.Language, gp.Framework)
			}
		}

		fmt.Printf("\nCapabilities:\n")
		caps := plt.Spec.Capabilities
		if caps.Secrets != "" {
			fmt.Printf("  Secrets:    %s\n", caps.Secrets)
		}
		if caps.GitOps != "" {
			fmt.Printf("  GitOps:     %s\n", caps.GitOps)
		}
		if caps.CI != "" {
			fmt.Printf("  CI:         %s\n", caps.CI)
		}
		if caps.Monitoring != "" {
			fmt.Printf("  Monitoring: %s\n", caps.Monitoring)
		}
		if caps.Logging != "" {
			fmt.Printf("  Logging:    %s\n", caps.Logging)
		}

		if len(plt.Spec.Environments) > 0 {
			fmt.Printf("\nEnvironments (%d):\n", len(plt.Spec.Environments))
			for _, env := range plt.Spec.Environments {
				fmt.Printf("  - %s (%s)\n", env.Name, env.Type)
			}
		}

		return nil
	},
}

var platformListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all platforms",
	RunE: func(cmd *cobra.Command, args []string) error {
		format, _ := cmd.Flags().GetString("output")

		platforms := platformManager.ListPlatforms()

		if format == "json" {
			data, _ := json.MarshalIndent(platforms, "", "  ")
			fmt.Println(string(data))
			return nil
		}

		if len(platforms) == 0 {
			fmt.Println("No platforms found")
			return nil
		}

		w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
		fmt.Fprintln(w, "NAME\tVERSION\tSTATUS\tGOLDEN PATHS\tENVIRONMENTS")
		for _, p := range platforms {
			version := p.Metadata.Version
			if version == "" {
				version = "-"
			}
			fmt.Fprintf(w, "%s\t%s\t%s\t%d\t%d\n",
				p.Metadata.Name,
				version,
				p.Status.Phase,
				len(p.Spec.GoldenPaths),
				len(p.Spec.Environments),
			)
		}
		w.Flush()

		return nil
	},
}

var platformDeleteCmd = &cobra.Command{
	Use:   "delete <name>",
	Short: "Delete a platform",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		name := args[0]
		force, _ := cmd.Flags().GetBool("force")

		if !force {
			fmt.Printf("Are you sure you want to delete platform '%s'?\n", name)
			fmt.Print("Type 'yes' to confirm: ")
			var confirm string
			fmt.Scanln(&confirm)
			if confirm != "yes" {
				fmt.Println("Aborted.")
				return nil
			}
		}

		ctx := context.Background()
		if err := platformManager.DeletePlatform(ctx, name); err != nil {
			return err
		}

		fmt.Printf("Platform '%s' deleted\n", name)
		return nil
	},
}

var platformDriftCmd = &cobra.Command{
	Use:   "drift <name>",
	Short: "Detect configuration drift",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		name := args[0]

		ctx := context.Background()
		drifts, err := platformManager.DetectDrift(ctx, name)
		if err != nil {
			return err
		}

		if len(drifts) == 0 {
			fmt.Println("No configuration drift detected.")
			return nil
		}

		fmt.Printf("Configuration drift detected (%d):\n", len(drifts))
		for _, d := range drifts {
			fmt.Printf("\n  Component: %s\n", d.Component)
			fmt.Printf("  Expected:  %s\n", d.Expected)
			fmt.Printf("  Actual:    %s\n", d.Actual)
			fmt.Printf("  Severity:  %s\n", d.Severity)
		}

		return nil
	},
}

// Golden path commands
var goldenPathCmd = &cobra.Command{
	Use:     "golden-path",
	Aliases: []string{"gp", "golden-paths"},
	Short:   "Manage golden paths",
}

var goldenPathListCmd = &cobra.Command{
	Use:   "list",
	Short: "List available golden paths",
	RunE: func(cmd *cobra.Command, args []string) error {
		format, _ := cmd.Flags().GetString("output")
		tag, _ := cmd.Flags().GetString("tag")

		var paths []*platform.GoldenPath
		if tag != "" {
			paths = platformManager.ListGoldenPathsByTag(tag)
		} else {
			paths = platformManager.ListGoldenPaths()
		}

		if format == "json" {
			data, _ := json.MarshalIndent(paths, "", "  ")
			fmt.Println(string(data))
			return nil
		}

		if len(paths) == 0 {
			fmt.Println("No golden paths found")
			return nil
		}

		w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
		fmt.Fprintln(w, "NAME\tLANGUAGE\tFRAMEWORK\tDESCRIPTION")
		for _, gp := range paths {
			framework := gp.Framework
			if framework == "" {
				framework = "-"
			}
			desc := gp.Description
			if len(desc) > 50 {
				desc = desc[:47] + "..."
			}
			fmt.Fprintf(w, "%s\t%s\t%s\t%s\n",
				gp.Name,
				gp.Language,
				framework,
				desc,
			)
		}
		w.Flush()

		return nil
	},
}

var goldenPathGetCmd = &cobra.Command{
	Use:   "get <name>",
	Short: "Get golden path details",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		name := args[0]
		format, _ := cmd.Flags().GetString("output")

		gp, err := platformManager.GetGoldenPath(name)
		if err != nil {
			return err
		}

		if format == "json" {
			data, _ := json.MarshalIndent(gp, "", "  ")
			fmt.Println(string(data))
			return nil
		}

		if format == "yaml" {
			data, _ := yaml.Marshal(gp)
			fmt.Println(string(data))
			return nil
		}

		fmt.Printf("Name:        %s\n", gp.Name)
		if gp.Description != "" {
			fmt.Printf("Description: %s\n", gp.Description)
		}
		fmt.Printf("Language:    %s\n", gp.Language)
		if gp.Framework != "" {
			fmt.Printf("Framework:   %s\n", gp.Framework)
		}

		if len(gp.Resources) > 0 {
			fmt.Printf("\nResources:\n")
			for _, r := range gp.Resources {
				required := ""
				if r.Required {
					required = " (required)"
				}
				fmt.Printf("  - %s%s\n", r.Type, required)
			}
		}

		if len(gp.Pipelines) > 0 {
			fmt.Printf("\nPipelines: %s\n", strings.Join(gp.Pipelines, " -> "))
		}

		if len(gp.Observability) > 0 {
			fmt.Printf("Observability: %s\n", strings.Join(gp.Observability, ", "))
		}

		if len(gp.Tags) > 0 {
			fmt.Printf("Tags: %s\n", strings.Join(gp.Tags, ", "))
		}

		if gp.Security != nil {
			fmt.Printf("\nSecurity:\n")
			if gp.Security.ImageScanning {
				fmt.Printf("  - Image scanning enabled\n")
			}
			if gp.Security.DependencyCheck {
				fmt.Printf("  - Dependency check enabled\n")
			}
			if gp.Security.SAST {
				fmt.Printf("  - SAST enabled\n")
			}
			if gp.Security.DAST {
				fmt.Printf("  - DAST enabled\n")
			}
		}

		return nil
	},
}

// Application commands
var appCmd = &cobra.Command{
	Use:     "app",
	Aliases: []string{"apps", "application"},
	Short:   "Manage applications",
}

var appCreateCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a new application",
	RunE: func(cmd *cobra.Command, args []string) error {
		filePath, _ := cmd.Flags().GetString("file")
		name, _ := cmd.Flags().GetString("name")
		goldenPath, _ := cmd.Flags().GetString("golden-path")
		team, _ := cmd.Flags().GetString("team")

		var app *platform.Application

		if filePath != "" {
			data, err := os.ReadFile(filePath)
			if err != nil {
				return fmt.Errorf("failed to read file: %w", err)
			}

			app = &platform.Application{}
			if strings.HasSuffix(filePath, ".json") {
				err = json.Unmarshal(data, app)
			} else {
				err = yaml.Unmarshal(data, app)
			}
			if err != nil {
				return fmt.Errorf("failed to parse application: %w", err)
			}
		} else {
			if name == "" || goldenPath == "" {
				return fmt.Errorf("--name and --golden-path are required")
			}

			app = &platform.Application{
				Metadata: platform.ApplicationMetadata{
					Name: name,
					Team: team,
				},
				Spec: platform.ApplicationSpec{
					GoldenPath: goldenPath,
				},
			}
		}

		// Validate
		errors := platformManager.ValidateApplication(app)
		if len(errors) > 0 {
			fmt.Println("Validation errors:")
			for _, e := range errors {
				fmt.Printf("  - %s\n", e)
			}
			return fmt.Errorf("validation failed")
		}

		ctx := context.Background()
		if err := platformManager.CreateApplication(ctx, app); err != nil {
			return err
		}

		fmt.Printf("Application '%s' created successfully using golden path '%s'\n",
			app.Metadata.Name, app.Spec.GoldenPath)
		return nil
	},
}

var appListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all applications",
	RunE: func(cmd *cobra.Command, args []string) error {
		format, _ := cmd.Flags().GetString("output")
		goldenPath, _ := cmd.Flags().GetString("golden-path")

		var apps []*platform.Application
		if goldenPath != "" {
			apps = platformManager.ListApplicationsByGoldenPath(goldenPath)
		} else {
			apps = platformManager.ListApplications()
		}

		if format == "json" {
			data, _ := json.MarshalIndent(apps, "", "  ")
			fmt.Println(string(data))
			return nil
		}

		if len(apps) == 0 {
			fmt.Println("No applications found")
			return nil
		}

		w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
		fmt.Fprintln(w, "NAME\tGOLDEN PATH\tTEAM\tSTATUS")
		for _, app := range apps {
			team := app.Metadata.Team
			if team == "" {
				team = "-"
			}
			status := "Pending"
			if app.Status != nil {
				status = app.Status.Phase
			}
			fmt.Fprintf(w, "%s\t%s\t%s\t%s\n",
				app.Metadata.Name,
				app.Spec.GoldenPath,
				team,
				status,
			)
		}
		w.Flush()

		return nil
	},
}

var appGetCmd = &cobra.Command{
	Use:   "get <name>",
	Short: "Get application details",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		name := args[0]
		format, _ := cmd.Flags().GetString("output")

		app, err := platformManager.GetApplication(name)
		if err != nil {
			return err
		}

		if format == "json" {
			data, _ := json.MarshalIndent(app, "", "  ")
			fmt.Println(string(data))
			return nil
		}

		if format == "yaml" {
			data, _ := yaml.Marshal(app)
			fmt.Println(string(data))
			return nil
		}

		fmt.Printf("Name:        %s\n", app.Metadata.Name)
		fmt.Printf("Golden Path: %s\n", app.Spec.GoldenPath)
		if app.Metadata.Team != "" {
			fmt.Printf("Team:        %s\n", app.Metadata.Team)
		}
		if app.Spec.Repository != "" {
			fmt.Printf("Repository:  %s\n", app.Spec.Repository)
		}

		if app.Status != nil {
			fmt.Printf("\nStatus:      %s\n", app.Status.Phase)

			if len(app.Status.Deployments) > 0 {
				fmt.Printf("\nDeployments:\n")
				for env, dep := range app.Status.Deployments {
					fmt.Printf("  %s: %s (%d/%d ready)\n",
						env, dep.Version, dep.Ready, dep.Replicas)
				}
			}

			if len(app.Status.Resources) > 0 {
				fmt.Printf("\nResources:\n")
				for _, r := range app.Status.Resources {
					fmt.Printf("  - %s (%s): %s\n", r.Name, r.Type, r.Status)
				}
			}
		}

		return nil
	},
}

var appDeleteCmd = &cobra.Command{
	Use:   "delete <name>",
	Short: "Delete an application",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		name := args[0]
		force, _ := cmd.Flags().GetBool("force")

		if !force {
			fmt.Printf("Are you sure you want to delete application '%s'?\n", name)
			fmt.Print("Type 'yes' to confirm: ")
			var confirm string
			fmt.Scanln(&confirm)
			if confirm != "yes" {
				fmt.Println("Aborted.")
				return nil
			}
		}

		ctx := context.Background()
		if err := platformManager.DeleteApplication(ctx, name); err != nil {
			return err
		}

		fmt.Printf("Application '%s' deleted\n", name)
		return nil
	},
}

func init() {
	rootCmd.AddCommand(platformCmd)

	// Platform commands
	platformApplyCmd.Flags().StringP("file", "f", "", "Path to platform configuration file")
	platformCmd.AddCommand(platformApplyCmd)

	platformGetCmd.Flags().StringP("output", "o", "", "Output format (json, yaml)")
	platformCmd.AddCommand(platformGetCmd)

	platformListCmd.Flags().StringP("output", "o", "", "Output format (json)")
	platformCmd.AddCommand(platformListCmd)

	platformDeleteCmd.Flags().BoolP("force", "f", false, "Force delete without confirmation")
	platformCmd.AddCommand(platformDeleteCmd)

	platformCmd.AddCommand(platformDriftCmd)

	// Golden path commands
	platformCmd.AddCommand(goldenPathCmd)

	goldenPathListCmd.Flags().StringP("output", "o", "", "Output format (json)")
	goldenPathListCmd.Flags().String("tag", "", "Filter by tag")
	goldenPathCmd.AddCommand(goldenPathListCmd)

	goldenPathGetCmd.Flags().StringP("output", "o", "", "Output format (json, yaml)")
	goldenPathCmd.AddCommand(goldenPathGetCmd)

	// Application commands
	platformCmd.AddCommand(appCmd)

	appCreateCmd.Flags().StringP("file", "f", "", "Path to application configuration file")
	appCreateCmd.Flags().StringP("name", "n", "", "Application name")
	appCreateCmd.Flags().StringP("golden-path", "g", "", "Golden path to use")
	appCreateCmd.Flags().String("team", "", "Team owning the application")
	appCmd.AddCommand(appCreateCmd)

	appListCmd.Flags().StringP("output", "o", "", "Output format (json)")
	appListCmd.Flags().String("golden-path", "", "Filter by golden path")
	appCmd.AddCommand(appListCmd)

	appGetCmd.Flags().StringP("output", "o", "", "Output format (json, yaml)")
	appCmd.AddCommand(appGetCmd)

	appDeleteCmd.Flags().BoolP("force", "f", false, "Force delete without confirmation")
	appCmd.AddCommand(appDeleteCmd)
}
