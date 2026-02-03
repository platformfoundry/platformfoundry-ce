package dag

import (
	"fmt"
	"sync"
)

// Graph represents a directed acyclic graph for workflow step execution
type Graph struct {
	nodes map[string]*Node
	mu    sync.RWMutex
}

// Node represents a node in the DAG
type Node struct {
	ID           string
	Name         string
	Dependencies []string
	Dependents   []string
}

// NewGraph creates a new DAG
func NewGraph() *Graph {
	return &Graph{
		nodes: make(map[string]*Node),
	}
}

// AddNode adds a node to the graph
func (g *Graph) AddNode(id, name string, dependencies []string) error {
	g.mu.Lock()
	defer g.mu.Unlock()

	if _, exists := g.nodes[id]; exists {
		return fmt.Errorf("node %s already exists", id)
	}

	node := &Node{
		ID:           id,
		Name:         name,
		Dependencies: dependencies,
		Dependents:   make([]string, 0),
	}
	g.nodes[id] = node

	// Update dependents for existing dependency nodes
	for _, depID := range dependencies {
		if depNode, ok := g.nodes[depID]; ok {
			depNode.Dependents = append(depNode.Dependents, id)
		}
	}

	return nil
}

// RemoveNode removes a node from the graph
func (g *Graph) RemoveNode(id string) {
	g.mu.Lock()
	defer g.mu.Unlock()

	node, ok := g.nodes[id]
	if !ok {
		return
	}

	// Remove from dependents of dependencies
	for _, depID := range node.Dependencies {
		if depNode, ok := g.nodes[depID]; ok {
			for i, dependent := range depNode.Dependents {
				if dependent == id {
					depNode.Dependents = append(depNode.Dependents[:i], depNode.Dependents[i+1:]...)
					break
				}
			}
		}
	}

	delete(g.nodes, id)
}

// GetNode returns a node by ID
func (g *Graph) GetNode(id string) (*Node, bool) {
	g.mu.RLock()
	defer g.mu.RUnlock()
	node, ok := g.nodes[id]
	return node, ok
}

// GetDependencies returns the dependencies of a node
func (g *Graph) GetDependencies(id string) []string {
	g.mu.RLock()
	defer g.mu.RUnlock()

	if node, ok := g.nodes[id]; ok {
		deps := make([]string, len(node.Dependencies))
		copy(deps, node.Dependencies)
		return deps
	}
	return nil
}

// GetDependents returns the nodes that depend on this node
func (g *Graph) GetDependents(id string) []string {
	g.mu.RLock()
	defer g.mu.RUnlock()

	if node, ok := g.nodes[id]; ok {
		deps := make([]string, len(node.Dependents))
		copy(deps, node.Dependents)
		return deps
	}
	return nil
}

// Validate validates the graph structure
func (g *Graph) Validate() error {
	g.mu.RLock()
	defer g.mu.RUnlock()

	// Check that all dependencies exist
	for id, node := range g.nodes {
		for _, depID := range node.Dependencies {
			if _, exists := g.nodes[depID]; !exists {
				return fmt.Errorf("node %s depends on non-existent node %s", id, depID)
			}
		}
	}

	// Check for cycles
	if cycle := g.detectCycle(); cycle != "" {
		return fmt.Errorf("circular dependency detected: %s", cycle)
	}

	return nil
}

// DetectCycle checks for cycles and returns cycle info if found
func (g *Graph) DetectCycle() string {
	g.mu.RLock()
	defer g.mu.RUnlock()
	return g.detectCycle()
}

// detectCycle detects cycles using DFS (internal, must hold lock)
func (g *Graph) detectCycle() string {
	visited := make(map[string]bool)
	recStack := make(map[string]bool)
	path := make([]string, 0)

	var dfs func(id string) string
	dfs = func(id string) string {
		visited[id] = true
		recStack[id] = true
		path = append(path, id)

		node := g.nodes[id]
		for _, dep := range node.Dependencies {
			if _, exists := g.nodes[dep]; !exists {
				continue
			}
			if !visited[dep] {
				if cycle := dfs(dep); cycle != "" {
					return cycle
				}
			} else if recStack[dep] {
				// Found cycle - build cycle path
				cycleStart := -1
				for i, p := range path {
					if p == dep {
						cycleStart = i
						break
					}
				}
				if cycleStart >= 0 {
					cyclePath := append(path[cycleStart:], dep)
					return fmt.Sprintf("%v", cyclePath)
				}
				return fmt.Sprintf("%s -> %s", id, dep)
			}
		}

		recStack[id] = false
		path = path[:len(path)-1]
		return ""
	}

	for id := range g.nodes {
		if !visited[id] {
			if cycle := dfs(id); cycle != "" {
				return cycle
			}
		}
	}

	return ""
}

// HasCycle returns true if the graph contains a cycle
func (g *Graph) HasCycle() bool {
	return g.DetectCycle() != ""
}

// GetParallelExecutionLevels returns nodes grouped by execution level
// Nodes in the same level have all their dependencies satisfied and can run in parallel
func (g *Graph) GetParallelExecutionLevels() ([][]string, error) {
	g.mu.RLock()
	defer g.mu.RUnlock()

	if cycle := g.detectCycle(); cycle != "" {
		return nil, fmt.Errorf("circular dependency detected: %s", cycle)
	}

	var levels [][]string
	completed := make(map[string]bool)

	for len(completed) < len(g.nodes) {
		var currentLevel []string

		for id, node := range g.nodes {
			if completed[id] {
				continue
			}

			// Check if all dependencies are completed
			allDepsCompleted := true
			for _, dep := range node.Dependencies {
				if _, exists := g.nodes[dep]; exists && !completed[dep] {
					allDepsCompleted = false
					break
				}
			}

			if allDepsCompleted {
				currentLevel = append(currentLevel, id)
			}
		}

		if len(currentLevel) == 0 && len(completed) < len(g.nodes) {
			return nil, fmt.Errorf("unable to resolve remaining dependencies")
		}

		if len(currentLevel) > 0 {
			levels = append(levels, currentLevel)
			for _, id := range currentLevel {
				completed[id] = true
			}
		}
	}

	return levels, nil
}

// TopologicalSort returns nodes in topological order
func (g *Graph) TopologicalSort() ([]string, error) {
	levels, err := g.GetParallelExecutionLevels()
	if err != nil {
		return nil, err
	}

	var result []string
	for _, level := range levels {
		result = append(result, level...)
	}
	return result, nil
}

// Size returns the number of nodes in the graph
func (g *Graph) Size() int {
	g.mu.RLock()
	defer g.mu.RUnlock()
	return len(g.nodes)
}

// NodeIDs returns all node IDs
func (g *Graph) NodeIDs() []string {
	g.mu.RLock()
	defer g.mu.RUnlock()

	ids := make([]string, 0, len(g.nodes))
	for id := range g.nodes {
		ids = append(ids, id)
	}
	return ids
}

// BuildFromSteps builds a graph from step specifications
func BuildFromSteps(steps []StepInfo) (*Graph, error) {
	g := NewGraph()

	for _, step := range steps {
		if err := g.AddNode(step.ID, step.Name, step.DependsOn); err != nil {
			return nil, fmt.Errorf("failed to add step %s: %w", step.ID, err)
		}
	}

	if err := g.Validate(); err != nil {
		return nil, err
	}

	return g, nil
}

// StepInfo contains minimal step information for graph building
type StepInfo struct {
	ID        string
	Name      string
	DependsOn []string
}
