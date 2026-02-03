// Package graph provides a resource dependency graph with query capabilities.
package graph

import (
	"fmt"
	"regexp"
	"strings"
	"sync"
)

// Resource represents a node in the resource graph (extends Node with additional fields)
type Resource struct {
	ID        string                 `json:"id"`
	Name      string                 `json:"name"`
	Type      string                 `json:"type"`
	Namespace string                 `json:"namespace,omitempty"`
	Labels    map[string]string      `json:"labels,omitempty"`
	Spec      map[string]interface{} `json:"spec,omitempty"`
	Status    string                 `json:"status,omitempty"`
}

// ResourceGraph represents a directed graph of resources and their relationships
type ResourceGraph struct {
	resources    map[string]*Resource
	edges        map[string][]*Edge // from -> edges
	reverseEdges map[string][]*Edge // to -> edges
	mu           sync.RWMutex
}

// NewResourceGraph creates a new resource graph
func NewResourceGraph() *ResourceGraph {
	return &ResourceGraph{
		resources:    make(map[string]*Resource),
		edges:        make(map[string][]*Edge),
		reverseEdges: make(map[string][]*Edge),
	}
}

// AddResource adds a resource to the graph
func (g *ResourceGraph) AddResource(r *Resource) {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.resources[r.ID] = r
}

// RemoveResource removes a resource and its edges from the graph
func (g *ResourceGraph) RemoveResource(id string) {
	g.mu.Lock()
	defer g.mu.Unlock()

	delete(g.resources, id)
	delete(g.edges, id)
	delete(g.reverseEdges, id)

	// Remove edges pointing to this resource
	for from, edges := range g.edges {
		filtered := make([]*Edge, 0)
		for _, e := range edges {
			if e.To != id {
				filtered = append(filtered, e)
			}
		}
		g.edges[from] = filtered
	}
}

// AddEdge adds an edge between two resources
func (g *ResourceGraph) AddEdge(from, to, edgeType string, required bool) {
	g.mu.Lock()
	defer g.mu.Unlock()

	edge := &Edge{
		From:     from,
		To:       to,
		Type:     edgeType,
		Required: required,
	}

	g.edges[from] = append(g.edges[from], edge)
	g.reverseEdges[to] = append(g.reverseEdges[to], edge)
}

// GetResource returns a resource by ID
func (g *ResourceGraph) GetResource(id string) *Resource {
	g.mu.RLock()
	defer g.mu.RUnlock()
	return g.resources[id]
}

// GetDependencies returns resources that the given resource depends on
func (g *ResourceGraph) GetDependencies(id string) []*Resource {
	g.mu.RLock()
	defer g.mu.RUnlock()

	var deps []*Resource
	for _, edge := range g.edges[id] {
		if r := g.resources[edge.To]; r != nil {
			deps = append(deps, r)
		}
	}
	return deps
}

// GetDependents returns resources that depend on the given resource
func (g *ResourceGraph) GetDependents(id string) []*Resource {
	g.mu.RLock()
	defer g.mu.RUnlock()

	var deps []*Resource
	for _, edge := range g.reverseEdges[id] {
		if r := g.resources[edge.From]; r != nil {
			deps = append(deps, r)
		}
	}
	return deps
}

// FindBySelector returns resources matching a selector
func (g *ResourceGraph) FindBySelector(selector Selector) []*Resource {
	g.mu.RLock()
	defer g.mu.RUnlock()

	var results []*Resource
	for _, r := range g.resources {
		if selector.Matches(r) {
			results = append(results, r)
		}
	}
	return results
}

// Selector represents criteria for selecting resources
type Selector struct {
	Type      string            `json:"type,omitempty"`
	Name      string            `json:"name,omitempty"`
	Namespace string            `json:"namespace,omitempty"`
	Labels    map[string]string `json:"labels,omitempty"`
	NameRegex *regexp.Regexp    `json:"-"`
}

// Matches checks if a resource matches the selector
func (s *Selector) Matches(r *Resource) bool {
	if s.Type != "" && s.Type != "*" && r.Type != s.Type {
		return false
	}

	if s.Name != "" && s.Name != "*" {
		if s.NameRegex != nil {
			if !s.NameRegex.MatchString(r.Name) {
				return false
			}
		} else if r.Name != s.Name {
			return false
		}
	}

	if s.Namespace != "" && s.Namespace != "*" && r.Namespace != s.Namespace {
		return false
	}

	for k, v := range s.Labels {
		if r.Labels[k] != v {
			return false
		}
	}

	return true
}

// QueryEngine executes queries against the resource graph
type QueryEngine struct {
	graph *ResourceGraph
}

// NewQueryEngine creates a new query engine
func NewQueryEngine(graph *ResourceGraph) *QueryEngine {
	return &QueryEngine{graph: graph}
}

// Query represents a parsed graph query
type Query struct {
	Source    Selector
	Target    Selector
	EdgeType  string // depends, owns, references, or empty for any
	Direction string // forward (->), reverse (<-), or bidirectional (<->)
	Depth     int    // max traversal depth, 0 = unlimited
}

// Execute executes a query and returns matching resources
func (e *QueryEngine) Execute(query *Query) ([]*Resource, error) {
	var results []*Resource

	// Find source resources
	sources := e.graph.FindBySelector(query.Source)

	switch query.Direction {
	case "forward", "->":
		// A -> B: Find B resources that A depends on
		for _, src := range sources {
			deps := e.traverseForward(src.ID, query.Target, query.EdgeType, query.Depth, make(map[string]bool))
			results = append(results, deps...)
		}

	case "reverse", "<-":
		// A <- B: Find B resources that depend on A
		for _, src := range sources {
			deps := e.traverseReverse(src.ID, query.Target, query.EdgeType, query.Depth, make(map[string]bool))
			results = append(results, deps...)
		}

	case "bidirectional", "<->":
		// Both directions
		for _, src := range sources {
			deps := e.traverseForward(src.ID, query.Target, query.EdgeType, query.Depth, make(map[string]bool))
			results = append(results, deps...)
			deps = e.traverseReverse(src.ID, query.Target, query.EdgeType, query.Depth, make(map[string]bool))
			results = append(results, deps...)
		}

	default:
		// Default to forward
		for _, src := range sources {
			deps := e.traverseForward(src.ID, query.Target, query.EdgeType, query.Depth, make(map[string]bool))
			results = append(results, deps...)
		}
	}

	// Deduplicate results
	return e.deduplicate(results), nil
}

// traverseForward traverses edges in the forward direction
func (e *QueryEngine) traverseForward(id string, target Selector, edgeType string, depth int, visited map[string]bool) []*Resource {
	if visited[id] {
		return nil
	}
	if depth == 0 {
		depth = 100 // Default max depth
	}

	visited[id] = true
	var results []*Resource

	e.graph.mu.RLock()
	edges := e.graph.edges[id]
	e.graph.mu.RUnlock()

	for _, edge := range edges {
		if edgeType != "" && edge.Type != edgeType {
			continue
		}

		r := e.graph.GetResource(edge.To)
		if r == nil {
			continue
		}

		if target.Matches(r) {
			results = append(results, r)
		}

		// Recursive traversal
		if depth > 1 {
			results = append(results, e.traverseForward(edge.To, target, edgeType, depth-1, visited)...)
		}
	}

	return results
}

// traverseReverse traverses edges in the reverse direction
func (e *QueryEngine) traverseReverse(id string, target Selector, edgeType string, depth int, visited map[string]bool) []*Resource {
	if visited[id] {
		return nil
	}
	if depth == 0 {
		depth = 100
	}

	visited[id] = true
	var results []*Resource

	e.graph.mu.RLock()
	edges := e.graph.reverseEdges[id]
	e.graph.mu.RUnlock()

	for _, edge := range edges {
		if edgeType != "" && edge.Type != edgeType {
			continue
		}

		r := e.graph.GetResource(edge.From)
		if r == nil {
			continue
		}

		if target.Matches(r) {
			results = append(results, r)
		}

		if depth > 1 {
			results = append(results, e.traverseReverse(edge.From, target, edgeType, depth-1, visited)...)
		}
	}

	return results
}

// deduplicate removes duplicate resources from results
func (e *QueryEngine) deduplicate(resources []*Resource) []*Resource {
	seen := make(map[string]bool)
	var unique []*Resource

	for _, r := range resources {
		if !seen[r.ID] {
			seen[r.ID] = true
			unique = append(unique, r)
		}
	}

	return unique
}

// ParseQuery parses a query string into a Query struct
// Query syntax examples:
//   "Workload[name=api] -> Database"
//   "* -> Secret[name=*-credentials]"
//   "Workload[team=platform] -[depends]-> *"
func ParseQuery(queryStr string) (*Query, error) {
	query := &Query{
		Direction: "forward",
		Depth:     0,
	}

	// Determine direction
	var parts []string
	if strings.Contains(queryStr, "<->") {
		query.Direction = "bidirectional"
		parts = strings.Split(queryStr, "<->")
	} else if strings.Contains(queryStr, "<-") {
		query.Direction = "reverse"
		parts = strings.Split(queryStr, "<-")
	} else if strings.Contains(queryStr, "->") {
		query.Direction = "forward"
		parts = strings.Split(queryStr, "->")
	} else {
		return nil, fmt.Errorf("invalid query: missing direction operator (-> or <-)")
	}

	if len(parts) != 2 {
		return nil, fmt.Errorf("invalid query: expected source -> target format")
	}

	// Parse source selector
	source, err := parseSelector(strings.TrimSpace(parts[0]))
	if err != nil {
		return nil, fmt.Errorf("invalid source selector: %w", err)
	}
	query.Source = *source

	// Parse target selector (may include edge type)
	targetStr := strings.TrimSpace(parts[1])

	// Check for edge type: -[type]->
	if strings.HasPrefix(targetStr, "[") {
		endBracket := strings.Index(targetStr, "]")
		if endBracket > 0 {
			query.EdgeType = targetStr[1:endBracket]
			targetStr = strings.TrimSpace(targetStr[endBracket+1:])
		}
	}

	target, err := parseSelector(targetStr)
	if err != nil {
		return nil, fmt.Errorf("invalid target selector: %w", err)
	}
	query.Target = *target

	return query, nil
}

// parseSelector parses a selector string like "Type[key=value]"
func parseSelector(s string) (*Selector, error) {
	selector := &Selector{
		Labels: make(map[string]string),
	}

	s = strings.TrimSpace(s)
	if s == "*" {
		return selector, nil
	}

	// Parse Type[conditions]
	bracketStart := strings.Index(s, "[")
	if bracketStart == -1 {
		selector.Type = s
		return selector, nil
	}

	selector.Type = s[:bracketStart]

	bracketEnd := strings.LastIndex(s, "]")
	if bracketEnd == -1 {
		return nil, fmt.Errorf("unclosed bracket in selector")
	}

	conditions := s[bracketStart+1 : bracketEnd]
	for _, cond := range strings.Split(conditions, ",") {
		parts := strings.SplitN(strings.TrimSpace(cond), "=", 2)
		if len(parts) != 2 {
			continue
		}

		key := strings.TrimSpace(parts[0])
		value := strings.TrimSpace(parts[1])

		switch key {
		case "name":
			if strings.Contains(value, "*") {
				// Convert glob to regex
				pattern := strings.ReplaceAll(value, "*", ".*")
				selector.NameRegex = regexp.MustCompile("^" + pattern + "$")
			}
			selector.Name = value
		case "namespace":
			selector.Namespace = value
		default:
			selector.Labels[key] = value
		}
	}

	return selector, nil
}

// GetAllPaths returns all paths between two resources
func (e *QueryEngine) GetAllPaths(fromID, toID string, maxDepth int) [][]string {
	if maxDepth == 0 {
		maxDepth = 10
	}

	var paths [][]string
	e.findPaths(fromID, toID, []string{fromID}, maxDepth, &paths)
	return paths
}

// findPaths recursively finds all paths
func (e *QueryEngine) findPaths(current, target string, path []string, depth int, paths *[][]string) {
	if depth == 0 {
		return
	}

	if current == target {
		pathCopy := make([]string, len(path))
		copy(pathCopy, path)
		*paths = append(*paths, pathCopy)
		return
	}

	e.graph.mu.RLock()
	edges := e.graph.edges[current]
	e.graph.mu.RUnlock()

	for _, edge := range edges {
		// Check if already in path (avoid cycles)
		inPath := false
		for _, p := range path {
			if p == edge.To {
				inPath = true
				break
			}
		}

		if !inPath {
			e.findPaths(edge.To, target, append(path, edge.To), depth-1, paths)
		}
	}
}

// TopologicalSort returns resources in dependency order
func (g *ResourceGraph) TopologicalSort() ([]*Resource, error) {
	g.mu.RLock()
	defer g.mu.RUnlock()

	visited := make(map[string]bool)
	recStack := make(map[string]bool)
	var result []*Resource

	var visit func(id string) error
	visit = func(id string) error {
		if recStack[id] {
			return fmt.Errorf("cycle detected involving %s", id)
		}
		if visited[id] {
			return nil
		}

		recStack[id] = true

		for _, edge := range g.edges[id] {
			if err := visit(edge.To); err != nil {
				return err
			}
		}

		recStack[id] = false
		visited[id] = true
		if r := g.resources[id]; r != nil {
			result = append([]*Resource{r}, result...)
		}

		return nil
	}

	for id := range g.resources {
		if err := visit(id); err != nil {
			return nil, err
		}
	}

	return result, nil
}
