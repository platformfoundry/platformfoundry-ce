// Package graph provides resource dependency graph functionality.
package graph

import (
	"fmt"
	"sort"
	"strings"
)

// Node represents a resource in the dependency graph
type Node struct {
	ID           string                 `json:"id"`
	Type         string                 `json:"type"`
	Name         string                 `json:"name"`
	Namespace    string                 `json:"namespace,omitempty"`
	Status       string                 `json:"status,omitempty"`
	Dependencies []string               `json:"dependencies,omitempty"`
	Dependents   []string               `json:"dependents,omitempty"`
	Metadata     map[string]interface{} `json:"metadata,omitempty"`
}

// Edge represents a dependency relationship
type Edge struct {
	From     string `json:"from"`
	To       string `json:"to"`
	Type     string `json:"type"` // depends_on, references, creates
	Required bool   `json:"required"`
}

// Graph represents the complete resource dependency graph
type Graph struct {
	nodes map[string]*Node
	edges []Edge
}

// New creates a new dependency graph
func New() *Graph {
	return &Graph{
		nodes: make(map[string]*Node),
		edges: make([]Edge, 0),
	}
}

// AddNode adds a node to the graph
func (g *Graph) AddNode(node *Node) {
	if node.ID == "" {
		node.ID = fmt.Sprintf("%s/%s", node.Type, node.Name)
	}
	g.nodes[node.ID] = node
}

// AddEdge adds a dependency edge
func (g *Graph) AddEdge(from, to, edgeType string, required bool) {
	g.edges = append(g.edges, Edge{
		From:     from,
		To:       to,
		Type:     edgeType,
		Required: required,
	})

	// Update node dependencies
	if fromNode, ok := g.nodes[from]; ok {
		fromNode.Dependencies = append(fromNode.Dependencies, to)
	}
	if toNode, ok := g.nodes[to]; ok {
		toNode.Dependents = append(toNode.Dependents, from)
	}
}

// GetNode returns a node by ID
func (g *Graph) GetNode(id string) *Node {
	return g.nodes[id]
}

// Nodes returns all nodes
func (g *Graph) Nodes() []*Node {
	result := make([]*Node, 0, len(g.nodes))
	for _, node := range g.nodes {
		result = append(result, node)
	}
	// Sort for consistent output
	sort.Slice(result, func(i, j int) bool {
		return result[i].ID < result[j].ID
	})
	return result
}

// Edges returns all edges
func (g *Graph) Edges() []Edge {
	return g.edges
}

// TopologicalSort returns nodes in dependency order
func (g *Graph) TopologicalSort() ([]*Node, error) {
	// Kahn's algorithm for topological sorting
	inDegree := make(map[string]int)
	for id := range g.nodes {
		inDegree[id] = 0
	}

	for _, edge := range g.edges {
		inDegree[edge.From]++
	}

	// Find all nodes with no dependencies
	queue := make([]string, 0)
	for id, degree := range inDegree {
		if degree == 0 {
			queue = append(queue, id)
		}
	}

	var result []*Node
	for len(queue) > 0 {
		// Pop from queue
		id := queue[0]
		queue = queue[1:]

		result = append(result, g.nodes[id])

		// Reduce in-degree of dependents
		for _, edge := range g.edges {
			if edge.To == id {
				inDegree[edge.From]--
				if inDegree[edge.From] == 0 {
					queue = append(queue, edge.From)
				}
			}
		}
	}

	// Check for cycles
	if len(result) != len(g.nodes) {
		return nil, fmt.Errorf("dependency cycle detected")
	}

	return result, nil
}

// DetectCycles returns any cycles in the graph
func (g *Graph) DetectCycles() [][]string {
	var cycles [][]string
	visited := make(map[string]bool)
	recStack := make(map[string]bool)
	path := make([]string, 0)

	var dfs func(id string) bool
	dfs = func(id string) bool {
		visited[id] = true
		recStack[id] = true
		path = append(path, id)

		for _, edge := range g.edges {
			if edge.From == id {
				if !visited[edge.To] {
					if dfs(edge.To) {
						return true
					}
				} else if recStack[edge.To] {
					// Found cycle - extract it
					cycleStart := -1
					for i, p := range path {
						if p == edge.To {
							cycleStart = i
							break
						}
					}
					if cycleStart >= 0 {
						cycle := make([]string, len(path)-cycleStart)
						copy(cycle, path[cycleStart:])
						cycle = append(cycle, edge.To)
						cycles = append(cycles, cycle)
					}
					return true
				}
			}
		}

		recStack[id] = false
		path = path[:len(path)-1]
		return false
	}

	for id := range g.nodes {
		if !visited[id] {
			dfs(id)
		}
	}

	return cycles
}

// GetDependencies returns all dependencies of a node (transitive)
func (g *Graph) GetDependencies(id string, transitive bool) []*Node {
	if !transitive {
		var deps []*Node
		for _, edge := range g.edges {
			if edge.From == id {
				if node := g.nodes[edge.To]; node != nil {
					deps = append(deps, node)
				}
			}
		}
		return deps
	}

	// BFS for transitive dependencies
	visited := make(map[string]bool)
	queue := []string{id}
	var result []*Node

	for len(queue) > 0 {
		current := queue[0]
		queue = queue[1:]

		for _, edge := range g.edges {
			if edge.From == current && !visited[edge.To] {
				visited[edge.To] = true
				if node := g.nodes[edge.To]; node != nil {
					result = append(result, node)
					queue = append(queue, edge.To)
				}
			}
		}
	}

	return result
}

// GetDependents returns all nodes that depend on a node
func (g *Graph) GetDependents(id string, transitive bool) []*Node {
	if !transitive {
		var deps []*Node
		for _, edge := range g.edges {
			if edge.To == id {
				if node := g.nodes[edge.From]; node != nil {
					deps = append(deps, node)
				}
			}
		}
		return deps
	}

	// BFS for transitive dependents
	visited := make(map[string]bool)
	queue := []string{id}
	var result []*Node

	for len(queue) > 0 {
		current := queue[0]
		queue = queue[1:]

		for _, edge := range g.edges {
			if edge.To == current && !visited[edge.From] {
				visited[edge.From] = true
				if node := g.nodes[edge.From]; node != nil {
					result = append(result, node)
					queue = append(queue, edge.From)
				}
			}
		}
	}

	return result
}

// Format returns a text representation of the graph
func (g *Graph) Format() string {
	var sb strings.Builder

	sb.WriteString("Resource Dependency Graph\n")
	sb.WriteString("=========================\n\n")

	nodes := g.Nodes()
	if len(nodes) == 0 {
		sb.WriteString("No resources in graph\n")
		return sb.String()
	}

	sb.WriteString("Resources:\n")
	sb.WriteString("----------\n")

	// Group by type
	byType := make(map[string][]*Node)
	for _, node := range nodes {
		byType[node.Type] = append(byType[node.Type], node)
	}

	types := make([]string, 0, len(byType))
	for t := range byType {
		types = append(types, t)
	}
	sort.Strings(types)

	for _, t := range types {
		sb.WriteString(fmt.Sprintf("\n  %s:\n", t))
		for _, node := range byType[t] {
			status := ""
			if node.Status != "" {
				status = fmt.Sprintf(" [%s]", node.Status)
			}
			sb.WriteString(fmt.Sprintf("    - %s%s\n", node.Name, status))

			if len(node.Dependencies) > 0 {
				sb.WriteString(fmt.Sprintf("      depends on: %s\n", strings.Join(node.Dependencies, ", ")))
			}
		}
	}

	// Show dependency tree
	sb.WriteString("\nDependency Tree:\n")
	sb.WriteString("----------------\n")

	roots := g.findRoots()
	for _, root := range roots {
		g.formatTree(&sb, root.ID, "", true)
	}

	// Check for cycles
	cycles := g.DetectCycles()
	if len(cycles) > 0 {
		sb.WriteString("\n[WARNING] Dependency Cycles Detected:\n")
		for _, cycle := range cycles {
			sb.WriteString(fmt.Sprintf("  %s\n", strings.Join(cycle, " -> ")))
		}
	}

	return sb.String()
}

func (g *Graph) findRoots() []*Node {
	hasParent := make(map[string]bool)
	for _, edge := range g.edges {
		hasParent[edge.From] = true
	}

	var roots []*Node
	for id, node := range g.nodes {
		if !hasParent[id] {
			roots = append(roots, node)
		}
	}

	sort.Slice(roots, func(i, j int) bool {
		return roots[i].ID < roots[j].ID
	})

	return roots
}

func (g *Graph) formatTree(sb *strings.Builder, id string, prefix string, isLast bool) {
	node := g.nodes[id]
	if node == nil {
		return
	}

	connector := "├── "
	if isLast {
		connector = "└── "
	}

	status := ""
	if node.Status != "" {
		status = fmt.Sprintf(" [%s]", node.Status)
	}

	sb.WriteString(fmt.Sprintf("%s%s%s/%s%s\n", prefix, connector, node.Type, node.Name, status))

	newPrefix := prefix
	if isLast {
		newPrefix += "    "
	} else {
		newPrefix += "│   "
	}

	children := g.GetDependents(id, false)
	for i, child := range children {
		g.formatTree(sb, child.ID, newPrefix, i == len(children)-1)
	}
}

// ExportDOT exports the graph in DOT format for visualization with Graphviz
func (g *Graph) ExportDOT() string {
	var sb strings.Builder

	sb.WriteString("digraph dependencies {\n")
	sb.WriteString("  rankdir=LR;\n")
	sb.WriteString("  node [shape=box, style=rounded];\n\n")

	// Define nodes with styling based on type
	typeColors := map[string]string{
		"Cluster":     "#4a90d9",
		"Deployment":  "#7cb342",
		"Service":     "#ff9800",
		"Pipeline":    "#9c27b0",
		"Database":    "#f44336",
		"Environment": "#00bcd4",
	}

	for _, node := range g.Nodes() {
		color := typeColors[node.Type]
		if color == "" {
			color = "#9e9e9e"
		}

		label := fmt.Sprintf("%s\\n%s", node.Type, node.Name)
		sb.WriteString(fmt.Sprintf("  \"%s\" [label=\"%s\", fillcolor=\"%s\", style=filled];\n",
			node.ID, label, color))
	}

	sb.WriteString("\n")

	// Define edges
	for _, edge := range g.edges {
		style := "solid"
		if !edge.Required {
			style = "dashed"
		}
		sb.WriteString(fmt.Sprintf("  \"%s\" -> \"%s\" [style=%s];\n", edge.From, edge.To, style))
	}

	sb.WriteString("}\n")

	return sb.String()
}

// ExportMermaid exports the graph in Mermaid format for documentation
func (g *Graph) ExportMermaid() string {
	var sb strings.Builder

	sb.WriteString("graph LR\n")

	// Define nodes
	for _, node := range g.Nodes() {
		shape := "([%s])" // Default rounded
		switch node.Type {
		case "Cluster":
			shape = "[(%s)]" // Cylinder
		case "Database":
			shape = "[(%s)]" // Cylinder
		case "Pipeline":
			shape = "[[%s]]" // Subroutine
		}

		label := fmt.Sprintf("%s: %s", node.Type, node.Name)
		nodeShape := fmt.Sprintf(shape, label)
		sb.WriteString(fmt.Sprintf("    %s%s\n", sanitizeMermaidID(node.ID), nodeShape))
	}

	sb.WriteString("\n")

	// Define edges
	for _, edge := range g.edges {
		arrow := "-->"
		if !edge.Required {
			arrow = "-.->"
		}
		sb.WriteString(fmt.Sprintf("    %s %s %s\n",
			sanitizeMermaidID(edge.From), arrow, sanitizeMermaidID(edge.To)))
	}

	return sb.String()
}

func sanitizeMermaidID(id string) string {
	// Replace invalid characters for Mermaid
	id = strings.ReplaceAll(id, "/", "_")
	id = strings.ReplaceAll(id, "-", "_")
	id = strings.ReplaceAll(id, " ", "_")
	return id
}
