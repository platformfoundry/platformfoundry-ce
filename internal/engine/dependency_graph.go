package engine

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// DependencyGraph manages engine dependencies and execution order
type DependencyGraph struct {
	nodes   map[string]*graphNode
	mu      sync.RWMutex
	outputs map[string]map[string]interface{}
}

// graphNode represents a node in the dependency graph
type graphNode struct {
	id           string
	name         string
	dependencies []string
	dependents   []string
}

// NewDependencyGraph creates a new dependency graph
func NewDependencyGraph() *DependencyGraph {
	return &DependencyGraph{
		nodes:   make(map[string]*graphNode),
		outputs: make(map[string]map[string]interface{}),
	}
}

// AddNode adds a node to the graph
func (g *DependencyGraph) AddNode(id, name string, dependencies []string) {
	g.mu.Lock()
	defer g.mu.Unlock()

	node := &graphNode{
		id:           id,
		name:         name,
		dependencies: dependencies,
		dependents:   make([]string, 0),
	}
	g.nodes[id] = node

	// Update dependents for existing nodes
	for _, depID := range dependencies {
		if depNode, ok := g.nodes[depID]; ok {
			depNode.dependents = append(depNode.dependents, id)
		}
	}
}

// RemoveNode removes a node from the graph
func (g *DependencyGraph) RemoveNode(id string) {
	g.mu.Lock()
	defer g.mu.Unlock()

	node, ok := g.nodes[id]
	if !ok {
		return
	}

	// Remove from dependents of dependencies
	for _, depID := range node.dependencies {
		if depNode, ok := g.nodes[depID]; ok {
			for i, dependent := range depNode.dependents {
				if dependent == id {
					depNode.dependents = append(depNode.dependents[:i], depNode.dependents[i+1:]...)
					break
				}
			}
		}
	}

	delete(g.nodes, id)
}

// GetParallelExecutionLevels returns engines grouped by execution level
// Engines in the same level can run in parallel
func (g *DependencyGraph) GetParallelExecutionLevels() ([][]string, error) {
	g.mu.RLock()
	defer g.mu.RUnlock()

	// Check for cycles
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
			for _, dep := range node.dependencies {
				// Only check dependencies that are in the graph
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

// detectCycle detects cycles in the dependency graph using DFS
func (g *DependencyGraph) detectCycle() string {
	visited := make(map[string]bool)
	recStack := make(map[string]bool)

	var dfs func(id string) string
	dfs = func(id string) string {
		visited[id] = true
		recStack[id] = true

		node := g.nodes[id]
		for _, dep := range node.dependencies {
			if depNode, exists := g.nodes[dep]; exists {
				if !visited[dep] {
					if cycle := dfs(dep); cycle != "" {
						return cycle
					}
				} else if recStack[dep] {
					return fmt.Sprintf("%s -> %s (via %s)", id, dep, depNode.name)
				}
			}
		}

		recStack[id] = false
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

// GetDependencies returns the dependencies of a node
func (g *DependencyGraph) GetDependencies(id string) []string {
	g.mu.RLock()
	defer g.mu.RUnlock()

	if node, ok := g.nodes[id]; ok {
		return node.dependencies
	}
	return nil
}

// GetDependents returns the dependents of a node
func (g *DependencyGraph) GetDependents(id string) []string {
	g.mu.RLock()
	defer g.mu.RUnlock()

	if node, ok := g.nodes[id]; ok {
		return node.dependents
	}
	return nil
}

// HasCycle returns true if the graph has a cycle
func (g *DependencyGraph) HasCycle() bool {
	g.mu.RLock()
	defer g.mu.RUnlock()
	return g.detectCycle() != ""
}

// TopologicalSort returns nodes in topological order
func (g *DependencyGraph) TopologicalSort() ([]string, error) {
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

// DependencyGraphResolver implements DependencyResolver interface
type DependencyGraphResolver struct {
	graph      *DependencyGraph
	completed  map[string]bool
	outputs    map[string]map[string]interface{}
	mu         sync.RWMutex
	notifyChan chan string
}

// NewDependencyGraphResolver creates a new resolver
func NewDependencyGraphResolver(graph *DependencyGraph) *DependencyGraphResolver {
	return &DependencyGraphResolver{
		graph:      graph,
		completed:  make(map[string]bool),
		outputs:    make(map[string]map[string]interface{}),
		notifyChan: make(chan string, 100),
	}
}

// IsSatisfied checks if all dependencies are satisfied
func (r *DependencyGraphResolver) IsSatisfied(engineID string, dependencies []string) bool {
	r.mu.RLock()
	defer r.mu.RUnlock()

	for _, dep := range dependencies {
		// Find engine by name in the graph
		depEngineID := r.findEngineByName(dep)
		if depEngineID == "" {
			continue // Dependency not in graph, assume satisfied
		}
		if !r.completed[depEngineID] {
			return false
		}
	}
	return true
}

// findEngineByName finds an engine ID by its name
func (r *DependencyGraphResolver) findEngineByName(name string) string {
	r.graph.mu.RLock()
	defer r.graph.mu.RUnlock()

	for id, node := range r.graph.nodes {
		if node.name == name || id == name {
			return id
		}
	}
	return ""
}

// WaitFor waits for all dependencies to complete
func (r *DependencyGraphResolver) WaitFor(ctx context.Context, dependencies []string) error {
	// Check if already satisfied
	if r.IsSatisfied("", dependencies) {
		return nil
	}

	// Wait with timeout
	timeout := 30 * time.Minute
	timer := time.NewTimer(timeout)
	defer timer.Stop()

	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-timer.C:
			return fmt.Errorf("timeout waiting for dependencies: %v", dependencies)
		case <-ticker.C:
			if r.IsSatisfied("", dependencies) {
				return nil
			}
		case completedID := <-r.notifyChan:
			// Check if this completion satisfies our dependencies
			_ = completedID
			if r.IsSatisfied("", dependencies) {
				return nil
			}
		}
	}
}

// GetOutput retrieves an output from a completed engine
func (r *DependencyGraphResolver) GetOutput(engineID string, key string) (interface{}, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	// Try to find by ID or name
	targetID := engineID
	if _, exists := r.outputs[engineID]; !exists {
		targetID = r.findEngineByName(engineID)
	}

	if outputs, ok := r.outputs[targetID]; ok {
		if val, ok := outputs[key]; ok {
			return val, nil
		}
		return nil, fmt.Errorf("output key %s not found for engine %s", key, engineID)
	}
	return nil, fmt.Errorf("engine %s outputs not found", engineID)
}

// MarkCompleted marks an engine as completed and stores its outputs
func (r *DependencyGraphResolver) MarkCompleted(engineID string, outputs map[string]interface{}) {
	r.mu.Lock()
	r.completed[engineID] = true
	r.outputs[engineID] = outputs
	r.mu.Unlock()

	// Notify waiters
	select {
	case r.notifyChan <- engineID:
	default:
		// Channel full, waiters will check via ticker
	}
}

// IsCompleted checks if an engine is completed
func (r *DependencyGraphResolver) IsCompleted(engineID string) bool {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.completed[engineID]
}

// Reset resets the resolver state
func (r *DependencyGraphResolver) Reset() {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.completed = make(map[string]bool)
	r.outputs = make(map[string]map[string]interface{})
}

// GetAllOutputs returns all outputs from all completed engines
func (r *DependencyGraphResolver) GetAllOutputs() map[string]map[string]interface{} {
	r.mu.RLock()
	defer r.mu.RUnlock()

	result := make(map[string]map[string]interface{})
	for id, outputs := range r.outputs {
		result[id] = make(map[string]interface{})
		for k, v := range outputs {
			result[id][k] = v
		}
	}
	return result
}
