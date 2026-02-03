package cli

import (
	"fmt"
	"os"

	"github.com/platformfoundry/pf-ce/internal/graph"
	"github.com/spf13/cobra"
)

var graphCmd = &cobra.Command{
	Use:   "graph",
	Short: "Resource dependency graph commands",
	Long:  "Visualize and analyze resource dependencies.",
}

var graphShowCmd = &cobra.Command{
	Use:   "show",
	Short: "Show the resource dependency graph",
	Long: `Display the dependency graph of all managed resources.

Examples:
  pf graph show                  # Show text-based graph
  pf graph show --format dot     # Export as Graphviz DOT
  pf graph show --format mermaid # Export as Mermaid diagram
  pf graph show --output deps.dot --format dot  # Save to file`,
	RunE: runGraphShow,
}

var graphCheckCmd = &cobra.Command{
	Use:   "check",
	Short: "Check for dependency issues",
	Long: `Analyze the dependency graph for potential issues:
- Circular dependencies
- Missing dependencies
- Orphaned resources

Examples:
  pf graph check              # Check all resources
  pf graph check --strict     # Fail on any warning`,
	RunE: runGraphCheck,
}

var (
	graphFormat string
	graphOutput string
	graphStrict bool
)

func init() {
	graphCmd.AddCommand(graphShowCmd)
	graphCmd.AddCommand(graphCheckCmd)

	graphShowCmd.Flags().StringVar(&graphFormat, "format", "text", "Output format (text, dot, mermaid)")
	graphShowCmd.Flags().StringVarP(&graphOutput, "output", "o", "", "Output file (default: stdout)")

	graphCheckCmd.Flags().BoolVar(&graphStrict, "strict", false, "Fail on any warning")
}

func runGraphShow(cmd *cobra.Command, args []string) error {
	// Build graph from resources
	g := buildResourceGraph()

	var output string
	switch graphFormat {
	case "dot":
		output = g.ExportDOT()
	case "mermaid":
		output = g.ExportMermaid()
	default:
		output = g.Format()
	}

	if graphOutput != "" {
		if err := os.WriteFile(graphOutput, []byte(output), 0644); err != nil {
			return fmt.Errorf("failed to write output: %w", err)
		}
		fmt.Printf("Graph exported to %s\n", graphOutput)
		return nil
	}

	fmt.Print(output)
	return nil
}

func runGraphCheck(cmd *cobra.Command, args []string) error {
	fmt.Println("Dependency Graph Check")
	fmt.Println("======================")
	fmt.Println()

	g := buildResourceGraph()

	hasIssues := false
	hasWarnings := false

	// Check for cycles
	cycles := g.DetectCycles()
	if len(cycles) > 0 {
		hasIssues = true
		fmt.Println("[ERROR] Circular dependencies detected:")
		for _, cycle := range cycles {
			fmt.Printf("  - %v\n", cycle)
		}
		fmt.Println()
	}

	// Check for missing dependencies
	missingDeps := checkMissingDependencies(g)
	if len(missingDeps) > 0 {
		hasWarnings = true
		fmt.Println("[WARN] Missing dependencies:")
		for _, dep := range missingDeps {
			fmt.Printf("  - %s references non-existent %s\n", dep.from, dep.to)
		}
		fmt.Println()
	}

	// Check for orphaned resources
	orphans := findOrphanedResources(g)
	if len(orphans) > 0 {
		hasWarnings = true
		fmt.Println("[INFO] Orphaned resources (no dependencies or dependents):")
		for _, orphan := range orphans {
			fmt.Printf("  - %s\n", orphan)
		}
		fmt.Println()
	}

	// Summary
	nodes := g.Nodes()
	edges := g.Edges()

	fmt.Println("Summary")
	fmt.Println("-------")
	fmt.Printf("  Resources: %d\n", len(nodes))
	fmt.Printf("  Dependencies: %d\n", len(edges))
	fmt.Printf("  Cycles: %d\n", len(cycles))
	fmt.Printf("  Warnings: %d\n", len(missingDeps)+len(orphans))
	fmt.Println()

	if hasIssues {
		fmt.Println("[FAILED] Dependency graph has critical issues")
		os.Exit(1)
	}

	if hasWarnings && graphStrict {
		fmt.Println("[FAILED] Dependency graph has warnings (strict mode)")
		os.Exit(1)
	}

	if !hasIssues && !hasWarnings {
		fmt.Println("[OK] Dependency graph is healthy")
	} else {
		fmt.Println("[OK] No critical issues found")
	}

	return nil
}

func buildResourceGraph() *graph.Graph {
	g := graph.New()

	// In production, this would:
	// 1. Read all platform configuration files
	// 2. Parse dependencies from YAML
	// 3. Query current resource state
	// For now, return sample data for demonstration

	// Add sample resources for demonstration
	// When real resources are available, remove this
	if len(g.Nodes()) == 0 {
		addSampleData(g)
	}

	return g
}

func addSampleData(g *graph.Graph) {
	// Sample graph showing typical platform structure
	g.AddNode(&graph.Node{
		Type:   "Cluster",
		Name:   "production",
		Status: "healthy",
	})

	g.AddNode(&graph.Node{
		Type:   "Cluster",
		Name:   "staging",
		Status: "healthy",
	})

	g.AddNode(&graph.Node{
		Type:   "Deployment",
		Name:   "api-server",
		Status: "running",
	})

	g.AddNode(&graph.Node{
		Type:   "Deployment",
		Name:   "web-frontend",
		Status: "running",
	})

	g.AddNode(&graph.Node{
		Type:   "Database",
		Name:   "postgres",
		Status: "healthy",
	})

	g.AddNode(&graph.Node{
		Type:   "Service",
		Name:   "api-gateway",
		Status: "active",
	})

	// Add dependencies
	g.AddEdge("Deployment/api-server", "Cluster/production", "runs_on", true)
	g.AddEdge("Deployment/web-frontend", "Cluster/production", "runs_on", true)
	g.AddEdge("Deployment/api-server", "Database/postgres", "connects_to", true)
	g.AddEdge("Service/api-gateway", "Deployment/api-server", "routes_to", true)
	g.AddEdge("Deployment/web-frontend", "Service/api-gateway", "calls", true)
}

type missingDep struct {
	from string
	to   string
}

func checkMissingDependencies(g *graph.Graph) []missingDep {
	var missing []missingDep

	nodeSet := make(map[string]bool)
	for _, node := range g.Nodes() {
		nodeSet[node.ID] = true
	}

	for _, edge := range g.Edges() {
		if !nodeSet[edge.To] {
			missing = append(missing, missingDep{from: edge.From, to: edge.To})
		}
	}

	return missing
}

func findOrphanedResources(g *graph.Graph) []string {
	var orphans []string

	hasDependency := make(map[string]bool)
	hasDependent := make(map[string]bool)

	for _, edge := range g.Edges() {
		hasDependency[edge.From] = true
		hasDependent[edge.To] = true
	}

	for _, node := range g.Nodes() {
		if !hasDependency[node.ID] && !hasDependent[node.ID] {
			orphans = append(orphans, node.ID)
		}
	}

	return orphans
}
