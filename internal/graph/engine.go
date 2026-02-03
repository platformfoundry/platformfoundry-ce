package graph

import (
	"context"
	"fmt"
	"sort"
	"sync"
	"time"

	"github.com/platformfoundry/pf-ce/pkg/types"
)

// Engine provides resource graph operations
type Engine struct {
	mu       sync.RWMutex
	graph    *types.ResourceGraph
	inEdges  map[string][]*types.ResourceEdge // target -> edges
	outEdges map[string][]*types.ResourceEdge // source -> edges
}

// NewEngine creates a new graph engine
func NewEngine() *Engine {
	return &Engine{
		graph: &types.ResourceGraph{
			Version:   "1.0",
			Nodes:     make(map[string]*types.ResourceNode),
			Edges:     make([]*types.ResourceEdge, 0),
			UpdatedAt: time.Now(),
		},
		inEdges:  make(map[string][]*types.ResourceEdge),
		outEdges: make(map[string][]*types.ResourceEdge),
	}
}

// AddNode adds a node to the graph
func (e *Engine) AddNode(node *types.ResourceNode) error {
	if node == nil {
		return fmt.Errorf("node cannot be nil")
	}
	if node.ID == "" {
		return fmt.Errorf("node ID cannot be empty")
	}

	e.mu.Lock()
	defer e.mu.Unlock()

	e.graph.Nodes[node.ID] = node
	e.graph.UpdatedAt = time.Now()
	return nil
}

// RemoveNode removes a node and its edges
func (e *Engine) RemoveNode(id string) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	if _, exists := e.graph.Nodes[id]; !exists {
		return fmt.Errorf("node %s not found", id)
	}

	// Remove the node
	delete(e.graph.Nodes, id)

	// Remove all edges involving this node
	var remaining []*types.ResourceEdge
	for _, edge := range e.graph.Edges {
		if edge.Source != id && edge.Target != id {
			remaining = append(remaining, edge)
		}
	}
	e.graph.Edges = remaining

	// Rebuild edge indices
	e.rebuildIndices()
	e.graph.UpdatedAt = time.Now()
	return nil
}

// GetNode retrieves a node by ID
func (e *Engine) GetNode(id string) (*types.ResourceNode, error) {
	e.mu.RLock()
	defer e.mu.RUnlock()

	node, exists := e.graph.Nodes[id]
	if !exists {
		return nil, fmt.Errorf("node %s not found", id)
	}
	return node, nil
}

// AddEdge adds an edge to the graph
func (e *Engine) AddEdge(edge *types.ResourceEdge) error {
	if edge == nil {
		return fmt.Errorf("edge cannot be nil")
	}
	if edge.Source == "" || edge.Target == "" {
		return fmt.Errorf("edge source and target cannot be empty")
	}

	e.mu.Lock()
	defer e.mu.Unlock()

	// Verify nodes exist
	if _, exists := e.graph.Nodes[edge.Source]; !exists {
		return fmt.Errorf("source node %s not found", edge.Source)
	}
	if _, exists := e.graph.Nodes[edge.Target]; !exists {
		return fmt.Errorf("target node %s not found", edge.Target)
	}

	// Generate ID if needed
	if edge.ID == "" {
		edge.ID = fmt.Sprintf("%s->%s:%s", edge.Source, edge.Target, edge.Type)
	}

	e.graph.Edges = append(e.graph.Edges, edge)
	e.outEdges[edge.Source] = append(e.outEdges[edge.Source], edge)
	e.inEdges[edge.Target] = append(e.inEdges[edge.Target], edge)
	e.graph.UpdatedAt = time.Now()
	return nil
}

// RemoveEdge removes an edge
func (e *Engine) RemoveEdge(id string) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	var found bool
	var remaining []*types.ResourceEdge
	for _, edge := range e.graph.Edges {
		if edge.ID == id {
			found = true
		} else {
			remaining = append(remaining, edge)
		}
	}

	if !found {
		return fmt.Errorf("edge %s not found", id)
	}

	e.graph.Edges = remaining
	e.rebuildIndices()
	e.graph.UpdatedAt = time.Now()
	return nil
}

// GetGraph returns the complete graph
func (e *Engine) GetGraph() *types.ResourceGraph {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return e.graph
}

// ImpactAnalysis calculates the impact of changing a resource
func (e *Engine) ImpactAnalysis(ctx context.Context, resourceID string) (*types.ImpactReport, error) {
	e.mu.RLock()
	defer e.mu.RUnlock()

	if _, exists := e.graph.Nodes[resourceID]; !exists {
		return nil, fmt.Errorf("resource %s not found", resourceID)
	}

	report := &types.ImpactReport{
		Resource:             resourceID,
		DirectlyAffected:     make([]string, 0),
		TransitivelyAffected: make([]string, 0),
		AffectedTeams:        make([]string, 0),
		AffectedEnvironments: make([]string, 0),
	}

	// Find all resources that depend on this resource (upstream traversal)
	visited := make(map[string]bool)
	directSet := make(map[string]bool)
	transitiveSet := make(map[string]bool)
	teamSet := make(map[string]bool)
	envSet := make(map[string]bool)

	// BFS to find dependents
	queue := []struct {
		id    string
		depth int
	}{{id: resourceID, depth: 0}}

	for len(queue) > 0 {
		current := queue[0]
		queue = queue[1:]

		if visited[current.id] {
			continue
		}
		visited[current.id] = true

		// Find nodes that depend on current
		for _, edge := range e.inEdges[current.id] {
			if edge.Type == types.EdgeDependsOn {
				node := e.graph.Nodes[edge.Source]
				if node != nil {
					if current.depth == 0 {
						directSet[edge.Source] = true
					} else {
						transitiveSet[edge.Source] = true
					}
					if node.Team != "" {
						teamSet[node.Team] = true
					}
					if node.Environment != "" {
						envSet[node.Environment] = true
					}
					if node.Criticality == "critical" {
						report.CriticalAffected++
					}
					queue = append(queue, struct {
						id    string
						depth int
					}{id: edge.Source, depth: current.depth + 1})
				}
			}
		}
	}

	// Convert sets to slices
	for id := range directSet {
		report.DirectlyAffected = append(report.DirectlyAffected, id)
	}
	for id := range transitiveSet {
		if !directSet[id] {
			report.TransitivelyAffected = append(report.TransitivelyAffected, id)
		}
	}
	for team := range teamSet {
		report.AffectedTeams = append(report.AffectedTeams, team)
	}
	for env := range envSet {
		report.AffectedEnvironments = append(report.AffectedEnvironments, env)
	}

	report.BlastRadius = len(report.DirectlyAffected) + len(report.TransitivelyAffected)

	// Calculate risk level
	report.RiskLevel = e.calculateRiskLevel(report)

	// Generate recommendations
	report.Recommendations = e.generateRecommendations(report)

	return report, nil
}

// BlastRadius calculates the number of affected resources
func (e *Engine) BlastRadius(resourceID string) (int, error) {
	report, err := e.ImpactAnalysis(context.Background(), resourceID)
	if err != nil {
		return 0, err
	}
	return report.BlastRadius, nil
}

// CriticalPath finds the most critical dependency chain
func (e *Engine) CriticalPath() ([]*types.ResourceNode, error) {
	e.mu.RLock()
	defer e.mu.RUnlock()

	// Find nodes with no outgoing edges (leaf nodes)
	var leafNodes []string
	for id := range e.graph.Nodes {
		if len(e.outEdges[id]) == 0 {
			leafNodes = append(leafNodes, id)
		}
	}

	// Find longest path from any root to any leaf
	var longestPath []*types.ResourceNode
	for _, leaf := range leafNodes {
		path := e.findLongestPathTo(leaf)
		if len(path) > len(longestPath) {
			longestPath = path
		}
	}

	return longestPath, nil
}

// FindPath finds a path between two nodes
func (e *Engine) FindPath(source, target string) (*types.PathResult, error) {
	e.mu.RLock()
	defer e.mu.RUnlock()

	if _, exists := e.graph.Nodes[source]; !exists {
		return nil, fmt.Errorf("source node %s not found", source)
	}
	if _, exists := e.graph.Nodes[target]; !exists {
		return nil, fmt.Errorf("target node %s not found", target)
	}

	// BFS to find shortest path
	visited := make(map[string]bool)
	parent := make(map[string]string)
	parentEdge := make(map[string]*types.ResourceEdge)

	queue := []string{source}
	visited[source] = true

	for len(queue) > 0 {
		current := queue[0]
		queue = queue[1:]

		if current == target {
			// Reconstruct path
			path := []string{target}
			var edges []*types.ResourceEdge
			for current != source {
				if edge := parentEdge[current]; edge != nil {
					edges = append([]*types.ResourceEdge{edge}, edges...)
				}
				current = parent[current]
				path = append([]string{current}, path...)
			}
			return &types.PathResult{
				Source: source,
				Target: target,
				Path:   path,
				Edges:  edges,
				Length: len(path) - 1,
			}, nil
		}

		for _, edge := range e.outEdges[current] {
			if !visited[edge.Target] {
				visited[edge.Target] = true
				parent[edge.Target] = current
				parentEdge[edge.Target] = edge
				queue = append(queue, edge.Target)
			}
		}
	}

	return nil, fmt.Errorf("no path found from %s to %s", source, target)
}

// GetDependencies returns all dependencies of a resource
func (e *Engine) GetDependencies(resourceID string, depth int) ([]*types.ResourceNode, error) {
	e.mu.RLock()
	defer e.mu.RUnlock()

	if _, exists := e.graph.Nodes[resourceID]; !exists {
		return nil, fmt.Errorf("resource %s not found", resourceID)
	}

	visited := make(map[string]bool)
	var dependencies []*types.ResourceNode

	e.collectDependencies(resourceID, depth, 0, visited, &dependencies)

	return dependencies, nil
}

// GetDependents returns all resources that depend on this resource
func (e *Engine) GetDependents(resourceID string, depth int) ([]*types.ResourceNode, error) {
	e.mu.RLock()
	defer e.mu.RUnlock()

	if _, exists := e.graph.Nodes[resourceID]; !exists {
		return nil, fmt.Errorf("resource %s not found", resourceID)
	}

	visited := make(map[string]bool)
	var dependents []*types.ResourceNode

	e.collectDependents(resourceID, depth, 0, visited, &dependents)

	return dependents, nil
}

// DetectCycles finds all cycles in the graph
func (e *Engine) DetectCycles() ([]*types.CycleResult, error) {
	e.mu.RLock()
	defer e.mu.RUnlock()

	var cycles []*types.CycleResult
	visited := make(map[string]bool)
	recStack := make(map[string]bool)
	path := make([]string, 0)

	for id := range e.graph.Nodes {
		if !visited[id] {
			e.detectCyclesDFS(id, visited, recStack, path, &cycles)
		}
	}

	return cycles, nil
}

// Stats returns graph statistics
func (e *Engine) Stats() *types.GraphStats {
	e.mu.RLock()
	defer e.mu.RUnlock()

	stats := &types.GraphStats{
		NodeCount:          len(e.graph.Nodes),
		EdgeCount:          len(e.graph.Edges),
		NodesByKind:        make(map[string]int),
		NodesByEnvironment: make(map[string]int),
		NodesByTeam:        make(map[string]int),
		EdgesByType:        make(map[types.EdgeType]int),
		CriticalNodes:      make([]string, 0),
	}

	// Count nodes by attributes
	for _, node := range e.graph.Nodes {
		stats.NodesByKind[node.Kind]++
		if node.Environment != "" {
			stats.NodesByEnvironment[node.Environment]++
		}
		if node.Team != "" {
			stats.NodesByTeam[node.Team]++
		}
	}

	// Count edges by type
	for _, edge := range e.graph.Edges {
		stats.EdgesByType[edge.Type]++
	}

	// Calculate degrees
	if len(e.graph.Nodes) > 0 {
		var totalOut, totalIn float64
		for id := range e.graph.Nodes {
			totalOut += float64(len(e.outEdges[id]))
			totalIn += float64(len(e.inEdges[id]))
		}
		stats.AverageOutDegree = totalOut / float64(len(e.graph.Nodes))
		stats.AverageInDegree = totalIn / float64(len(e.graph.Nodes))
	}

	// Find critical nodes (high blast radius)
	for id := range e.graph.Nodes {
		radius, _ := e.BlastRadius(id)
		if radius > 5 {
			stats.CriticalNodes = append(stats.CriticalNodes, id)
		}
	}

	// Check for cycles
	cycles, _ := e.DetectCycles()
	stats.HasCycles = len(cycles) > 0
	stats.CycleCount = len(cycles)

	// Calculate max depth
	stats.MaxDepth = e.calculateMaxDepth()

	return stats
}

// Query executes a graph query
func (e *Engine) Query(ctx context.Context, query *types.GraphQuery) ([]*types.ResourceNode, error) {
	e.mu.RLock()
	defer e.mu.RUnlock()

	var results []*types.ResourceNode

	if query.StartNode != "" {
		// Traversal query
		depth := query.MaxDepth
		if depth == 0 {
			depth = 10
		}

		visited := make(map[string]bool)
		switch query.Direction {
		case "downstream", "":
			e.collectDependencies(query.StartNode, depth, 0, visited, &results)
		case "upstream":
			e.collectDependents(query.StartNode, depth, 0, visited, &results)
		case "both":
			e.collectDependencies(query.StartNode, depth, 0, visited, &results)
			e.collectDependents(query.StartNode, depth, 0, visited, &results)
		}
	} else {
		// Filter query
		for _, node := range e.graph.Nodes {
			if e.matchesNodeFilter(node, query.NodeFilter) {
				results = append(results, node)
			}
		}
	}

	return results, nil
}

// TopoSort returns nodes in topological order
func (e *Engine) TopoSort() ([]string, error) {
	e.mu.RLock()
	defer e.mu.RUnlock()

	inDegree := make(map[string]int)
	for id := range e.graph.Nodes {
		inDegree[id] = 0
	}
	for _, edge := range e.graph.Edges {
		if edge.Type == types.EdgeDependsOn {
			inDegree[edge.Source]++
		}
	}

	var queue []string
	for id, degree := range inDegree {
		if degree == 0 {
			queue = append(queue, id)
		}
	}

	var sorted []string
	for len(queue) > 0 {
		current := queue[0]
		queue = queue[1:]
		sorted = append(sorted, current)

		for _, edge := range e.outEdges[current] {
			if edge.Type == types.EdgeDependsOn {
				inDegree[edge.Target]--
				if inDegree[edge.Target] == 0 {
					queue = append(queue, edge.Target)
				}
			}
		}
	}

	if len(sorted) != len(e.graph.Nodes) {
		return nil, fmt.Errorf("graph contains cycles, topological sort not possible")
	}

	return sorted, nil
}

// Helper methods

func (e *Engine) rebuildIndices() {
	e.inEdges = make(map[string][]*types.ResourceEdge)
	e.outEdges = make(map[string][]*types.ResourceEdge)
	for _, edge := range e.graph.Edges {
		e.outEdges[edge.Source] = append(e.outEdges[edge.Source], edge)
		e.inEdges[edge.Target] = append(e.inEdges[edge.Target], edge)
	}
}

func (e *Engine) collectDependencies(id string, maxDepth, currentDepth int, visited map[string]bool, result *[]*types.ResourceNode) {
	if currentDepth >= maxDepth || visited[id] {
		return
	}
	visited[id] = true

	for _, edge := range e.outEdges[id] {
		if node, exists := e.graph.Nodes[edge.Target]; exists {
			*result = append(*result, node)
			e.collectDependencies(edge.Target, maxDepth, currentDepth+1, visited, result)
		}
	}
}

func (e *Engine) collectDependents(id string, maxDepth, currentDepth int, visited map[string]bool, result *[]*types.ResourceNode) {
	if currentDepth >= maxDepth || visited[id] {
		return
	}
	visited[id] = true

	for _, edge := range e.inEdges[id] {
		if node, exists := e.graph.Nodes[edge.Source]; exists {
			*result = append(*result, node)
			e.collectDependents(edge.Source, maxDepth, currentDepth+1, visited, result)
		}
	}
}

func (e *Engine) detectCyclesDFS(id string, visited, recStack map[string]bool, path []string, cycles *[]*types.CycleResult) {
	visited[id] = true
	recStack[id] = true
	path = append(path, id)

	for _, edge := range e.outEdges[id] {
		if !visited[edge.Target] {
			e.detectCyclesDFS(edge.Target, visited, recStack, path, cycles)
		} else if recStack[edge.Target] {
			// Found a cycle
			cycleStart := -1
			for i, node := range path {
				if node == edge.Target {
					cycleStart = i
					break
				}
			}
			if cycleStart >= 0 {
				cycle := &types.CycleResult{
					Nodes: append([]string{}, path[cycleStart:]...),
				}
				*cycles = append(*cycles, cycle)
			}
		}
	}

	recStack[id] = false
}

func (e *Engine) findLongestPathTo(target string) []*types.ResourceNode {
	memo := make(map[string][]*types.ResourceNode)
	return e.dfsLongestPath(target, memo)
}

func (e *Engine) dfsLongestPath(id string, memo map[string][]*types.ResourceNode) []*types.ResourceNode {
	if cached, exists := memo[id]; exists {
		return cached
	}

	node := e.graph.Nodes[id]
	if node == nil {
		return nil
	}

	var longest []*types.ResourceNode
	for _, edge := range e.inEdges[id] {
		path := e.dfsLongestPath(edge.Source, memo)
		if len(path) > len(longest) {
			longest = path
		}
	}

	result := append([]*types.ResourceNode{}, longest...)
	result = append(result, node)
	memo[id] = result
	return result
}

func (e *Engine) calculateMaxDepth() int {
	maxDepth := 0
	for id := range e.graph.Nodes {
		depth := e.calculateDepth(id, make(map[string]bool))
		if depth > maxDepth {
			maxDepth = depth
		}
	}
	return maxDepth
}

func (e *Engine) calculateDepth(id string, visited map[string]bool) int {
	if visited[id] {
		return 0
	}
	visited[id] = true

	maxChildDepth := 0
	for _, edge := range e.outEdges[id] {
		childDepth := e.calculateDepth(edge.Target, visited)
		if childDepth > maxChildDepth {
			maxChildDepth = childDepth
		}
	}
	return maxChildDepth + 1
}

func (e *Engine) calculateRiskLevel(report *types.ImpactReport) string {
	if report.CriticalAffected > 0 || report.BlastRadius > 10 {
		return "critical"
	}
	if report.BlastRadius > 5 {
		return "high"
	}
	if report.BlastRadius > 2 {
		return "medium"
	}
	return "low"
}

func (e *Engine) generateRecommendations(report *types.ImpactReport) []string {
	var recommendations []string

	if report.RiskLevel == "critical" || report.RiskLevel == "high" {
		recommendations = append(recommendations, "Consider scheduling this change during a maintenance window")
		recommendations = append(recommendations, "Notify all affected teams before proceeding")
	}

	if report.CriticalAffected > 0 {
		recommendations = append(recommendations, fmt.Sprintf("Warning: %d critical resources will be affected", report.CriticalAffected))
	}

	if len(report.AffectedEnvironments) > 1 {
		recommendations = append(recommendations, "Multiple environments affected - consider staged rollout")
	}

	if report.BlastRadius > 5 {
		recommendations = append(recommendations, "High blast radius - ensure rollback plan is ready")
	}

	return recommendations
}

func (e *Engine) matchesNodeFilter(node *types.ResourceNode, filter *types.NodeFilter) bool {
	if filter == nil {
		return true
	}

	if len(filter.Kinds) > 0 && !contains(filter.Kinds, node.Kind) {
		return false
	}
	if len(filter.Names) > 0 && !contains(filter.Names, node.Name) {
		return false
	}
	if len(filter.Teams) > 0 && !contains(filter.Teams, node.Team) {
		return false
	}
	if len(filter.Environments) > 0 && !contains(filter.Environments, node.Environment) {
		return false
	}
	if len(filter.Statuses) > 0 && !contains(filter.Statuses, node.Status) {
		return false
	}
	if len(filter.Labels) > 0 {
		for k, v := range filter.Labels {
			if node.Labels[k] != v {
				return false
			}
		}
	}

	return true
}

func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

// ExportDOT exports the graph in DOT format for visualization
func (e *Engine) ExportDOT() string {
	e.mu.RLock()
	defer e.mu.RUnlock()

	var output string
	output += "digraph ResourceGraph {\n"
	output += "  rankdir=LR;\n"
	output += "  node [shape=box];\n\n"

	// Add nodes grouped by kind
	nodesByKind := make(map[string][]*types.ResourceNode)
	for _, node := range e.graph.Nodes {
		nodesByKind[node.Kind] = append(nodesByKind[node.Kind], node)
	}

	for kind, nodes := range nodesByKind {
		output += fmt.Sprintf("  subgraph cluster_%s {\n", kind)
		output += fmt.Sprintf("    label=\"%s\";\n", kind)
		for _, node := range nodes {
			color := "black"
			if node.Criticality == "critical" {
				color = "red"
			}
			output += fmt.Sprintf("    \"%s\" [label=\"%s\" color=%s];\n", node.ID, node.Name, color)
		}
		output += "  }\n\n"
	}

	// Add edges
	for _, edge := range e.graph.Edges {
		style := "solid"
		if !edge.Required {
			style = "dashed"
		}
		output += fmt.Sprintf("  \"%s\" -> \"%s\" [label=\"%s\" style=%s];\n", edge.Source, edge.Target, edge.Type, style)
	}

	output += "}\n"
	return output
}

// ExportMermaid exports the graph in Mermaid format
func (e *Engine) ExportMermaid() string {
	e.mu.RLock()
	defer e.mu.RUnlock()

	var output string
	output += "graph LR\n"

	// Sort nodes for consistent output
	var nodeIDs []string
	for id := range e.graph.Nodes {
		nodeIDs = append(nodeIDs, id)
	}
	sort.Strings(nodeIDs)

	// Add nodes
	for _, id := range nodeIDs {
		node := e.graph.Nodes[id]
		shape := "(%s)"
		if node.Kind == "Service" {
			shape = "[%s]"
		} else if node.Kind == "Database" {
			shape = "[(%s)]"
		}
		output += fmt.Sprintf("  %s"+shape+"\n", sanitizeID(id), node.Name)
	}

	// Add edges
	for _, edge := range e.graph.Edges {
		arrow := "-->"
		if !edge.Required {
			arrow = "-.->"
		}
		output += fmt.Sprintf("  %s %s|%s| %s\n", sanitizeID(edge.Source), arrow, edge.Type, sanitizeID(edge.Target))
	}

	return output
}

func sanitizeID(id string) string {
	// Replace characters that Mermaid doesn't like
	result := ""
	for _, c := range id {
		if (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9') || c == '_' {
			result += string(c)
		} else {
			result += "_"
		}
	}
	return result
}
