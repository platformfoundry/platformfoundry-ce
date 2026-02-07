package engine

import (
	"context"
	"sync"
	"testing"
	"time"
)

func TestNewDependencyGraph(t *testing.T) {
	graph := NewDependencyGraph()

	if graph == nil {
		t.Fatal("expected non-nil graph")
	}
	if graph.nodes == nil {
		t.Error("expected nodes map to be initialized")
	}
}

func TestDependencyGraphAddNode(t *testing.T) {
	graph := NewDependencyGraph()

	graph.AddNode("node1", "Node 1", nil)
	graph.AddNode("node2", "Node 2", []string{"node1"})

	if len(graph.nodes) != 2 {
		t.Errorf("expected 2 nodes, got %d", len(graph.nodes))
	}

	// Verify dependencies are set correctly
	node2 := graph.nodes["node2"]
	if len(node2.dependencies) != 1 || node2.dependencies[0] != "node1" {
		t.Error("expected node2 to depend on node1")
	}

	// Verify dependents are set correctly
	node1 := graph.nodes["node1"]
	if len(node1.dependents) != 1 || node1.dependents[0] != "node2" {
		t.Error("expected node1 to have node2 as dependent")
	}
}

func TestDependencyGraphRemoveNode(t *testing.T) {
	graph := NewDependencyGraph()

	graph.AddNode("node1", "Node 1", nil)
	graph.AddNode("node2", "Node 2", []string{"node1"})

	graph.RemoveNode("node2")

	if len(graph.nodes) != 1 {
		t.Errorf("expected 1 node after removal, got %d", len(graph.nodes))
	}

	// Verify dependents are updated
	node1 := graph.nodes["node1"]
	if len(node1.dependents) != 0 {
		t.Error("expected node1 to have no dependents after node2 removal")
	}

	// Remove non-existent node should not panic
	graph.RemoveNode("non-existent")
}

func TestDependencyGraphGetDependencies(t *testing.T) {
	graph := NewDependencyGraph()

	graph.AddNode("node1", "Node 1", nil)
	graph.AddNode("node2", "Node 2", []string{"node1"})
	graph.AddNode("node3", "Node 3", []string{"node1", "node2"})

	deps := graph.GetDependencies("node3")
	if len(deps) != 2 {
		t.Errorf("expected 2 dependencies, got %d", len(deps))
	}

	// Non-existent node
	deps = graph.GetDependencies("non-existent")
	if deps != nil {
		t.Error("expected nil for non-existent node")
	}
}

func TestDependencyGraphGetDependents(t *testing.T) {
	graph := NewDependencyGraph()

	graph.AddNode("node1", "Node 1", nil)
	graph.AddNode("node2", "Node 2", []string{"node1"})
	graph.AddNode("node3", "Node 3", []string{"node1"})

	dependents := graph.GetDependents("node1")
	if len(dependents) != 2 {
		t.Errorf("expected 2 dependents, got %d", len(dependents))
	}

	// Non-existent node
	dependents = graph.GetDependents("non-existent")
	if dependents != nil {
		t.Error("expected nil for non-existent node")
	}
}

func TestDependencyGraphGetParallelExecutionLevels(t *testing.T) {
	graph := NewDependencyGraph()

	// Level 0: node1, node2 (no dependencies)
	// Level 1: node3 (depends on node1)
	// Level 2: node4 (depends on node2, node3)
	graph.AddNode("node1", "Node 1", nil)
	graph.AddNode("node2", "Node 2", nil)
	graph.AddNode("node3", "Node 3", []string{"node1"})
	graph.AddNode("node4", "Node 4", []string{"node2", "node3"})

	levels, err := graph.GetParallelExecutionLevels()
	if err != nil {
		t.Errorf("GetParallelExecutionLevels failed: %v", err)
	}

	if len(levels) != 3 {
		t.Errorf("expected 3 levels, got %d", len(levels))
	}

	// Level 0 should have node1 and node2
	level0 := levels[0]
	if len(level0) != 2 {
		t.Errorf("expected 2 nodes in level 0, got %d", len(level0))
	}

	// Level 1 should have node3
	level1 := levels[1]
	if len(level1) != 1 {
		t.Errorf("expected 1 node in level 1, got %d", len(level1))
	}

	// Level 2 should have node4
	level2 := levels[2]
	if len(level2) != 1 {
		t.Errorf("expected 1 node in level 2, got %d", len(level2))
	}
}

func TestDependencyGraphGetParallelExecutionLevelsEmpty(t *testing.T) {
	graph := NewDependencyGraph()

	levels, err := graph.GetParallelExecutionLevels()
	if err != nil {
		t.Errorf("GetParallelExecutionLevels failed: %v", err)
	}

	if len(levels) != 0 {
		t.Errorf("expected 0 levels for empty graph, got %d", len(levels))
	}
}

func TestDependencyGraphCircularDependency(t *testing.T) {
	graph := NewDependencyGraph()

	// Create circular dependency: A -> B -> C -> A
	graph.AddNode("nodeA", "Node A", []string{"nodeC"})
	graph.AddNode("nodeB", "Node B", []string{"nodeA"})
	graph.AddNode("nodeC", "Node C", []string{"nodeB"})

	_, err := graph.GetParallelExecutionLevels()
	if err == nil {
		t.Error("expected error for circular dependency")
	}
}

func TestDependencyGraphSelfDependency(t *testing.T) {
	graph := NewDependencyGraph()

	graph.AddNode("node1", "Node 1", []string{"node1"})

	_, err := graph.GetParallelExecutionLevels()
	if err == nil {
		t.Error("expected error for self-dependency")
	}
}

func TestDependencyGraphHasCycle(t *testing.T) {
	graph := NewDependencyGraph()

	// No cycle
	graph.AddNode("node1", "Node 1", nil)
	graph.AddNode("node2", "Node 2", []string{"node1"})

	if graph.HasCycle() {
		t.Error("expected no cycle")
	}

	// Add cycle
	graph.AddNode("node3", "Node 3", []string{"node2"})
	graph.nodes["node1"].dependencies = []string{"node3"}

	if !graph.HasCycle() {
		t.Error("expected cycle to be detected")
	}
}

func TestDependencyGraphTopologicalSort(t *testing.T) {
	graph := NewDependencyGraph()

	graph.AddNode("node1", "Node 1", nil)
	graph.AddNode("node2", "Node 2", []string{"node1"})
	graph.AddNode("node3", "Node 3", []string{"node2"})

	sorted, err := graph.TopologicalSort()
	if err != nil {
		t.Errorf("TopologicalSort failed: %v", err)
	}

	if len(sorted) != 3 {
		t.Errorf("expected 3 nodes, got %d", len(sorted))
	}

	// node1 should come before node2, node2 before node3
	node1Idx, node2Idx, node3Idx := -1, -1, -1
	for i, id := range sorted {
		switch id {
		case "node1":
			node1Idx = i
		case "node2":
			node2Idx = i
		case "node3":
			node3Idx = i
		}
	}

	if node1Idx > node2Idx || node2Idx > node3Idx {
		t.Error("expected proper topological order")
	}
}

func TestDependencyGraphExternalDependencies(t *testing.T) {
	graph := NewDependencyGraph()

	// Node with external dependency (not in graph)
	graph.AddNode("node1", "Node 1", []string{"external-dep"})

	// Should still work - external deps are assumed satisfied
	levels, err := graph.GetParallelExecutionLevels()
	if err != nil {
		t.Errorf("GetParallelExecutionLevels failed: %v", err)
	}

	if len(levels) != 1 {
		t.Errorf("expected 1 level, got %d", len(levels))
	}
}

func TestDependencyGraphConcurrentAccess(t *testing.T) {
	graph := NewDependencyGraph()

	var wg sync.WaitGroup
	iterations := 100

	// Concurrent adds
	for i := 0; i < iterations; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			graph.AddNode(
				string(rune('A'+id%26))+"-"+string(rune('0'+id%10)),
				"Node",
				nil,
			)
		}(i)
	}

	wg.Wait()

	// Concurrent reads
	for i := 0; i < iterations; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			graph.HasCycle()
			graph.GetParallelExecutionLevels()
		}()
	}

	wg.Wait()
}

// Tests for DependencyGraphResolver

func TestNewDependencyGraphResolver(t *testing.T) {
	graph := NewDependencyGraph()
	resolver := NewDependencyGraphResolver(graph)

	if resolver == nil {
		t.Fatal("expected non-nil resolver")
	}
	if resolver.graph != graph {
		t.Error("expected resolver to reference the graph")
	}
}

func TestDependencyGraphResolverIsSatisfied(t *testing.T) {
	graph := NewDependencyGraph()
	graph.AddNode("engine1", "Engine 1", nil)
	graph.AddNode("engine2", "Engine 2", []string{"engine1"})

	resolver := NewDependencyGraphResolver(graph)

	// Initially, dependencies not satisfied
	if resolver.IsSatisfied("engine2", []string{"engine1"}) {
		t.Error("expected dependencies not satisfied initially")
	}

	// Mark engine1 as completed
	resolver.MarkCompleted("engine1", map[string]interface{}{"key": "value"})

	// Now dependencies should be satisfied
	if !resolver.IsSatisfied("engine2", []string{"engine1"}) {
		t.Error("expected dependencies to be satisfied after completion")
	}

	// Empty dependencies should be satisfied
	if !resolver.IsSatisfied("engine1", []string{}) {
		t.Error("expected empty dependencies to be satisfied")
	}

	// External dependency (not in graph) should be assumed satisfied
	if !resolver.IsSatisfied("engine1", []string{"external"}) {
		t.Error("expected external dependency to be assumed satisfied")
	}
}

func TestDependencyGraphResolverMarkCompleted(t *testing.T) {
	graph := NewDependencyGraph()
	graph.AddNode("engine1", "Engine 1", nil)

	resolver := NewDependencyGraphResolver(graph)

	outputs := map[string]interface{}{
		"endpoint": "http://localhost:8080",
		"port":     8080,
	}

	resolver.MarkCompleted("engine1", outputs)

	if !resolver.IsCompleted("engine1") {
		t.Error("expected engine1 to be marked completed")
	}

	// Check outputs
	val, err := resolver.GetOutput("engine1", "endpoint")
	if err != nil {
		t.Errorf("GetOutput failed: %v", err)
	}
	if val != "http://localhost:8080" {
		t.Errorf("expected 'http://localhost:8080', got '%v'", val)
	}
}

func TestDependencyGraphResolverGetOutput(t *testing.T) {
	graph := NewDependencyGraph()
	graph.AddNode("engine1", "Engine 1", nil)

	resolver := NewDependencyGraphResolver(graph)

	// Output not found
	_, err := resolver.GetOutput("engine1", "key")
	if err == nil {
		t.Error("expected error for engine with no outputs")
	}

	// Mark completed with outputs
	resolver.MarkCompleted("engine1", map[string]interface{}{"key": "value"})

	// Key not found
	_, err = resolver.GetOutput("engine1", "nonexistent")
	if err == nil {
		t.Error("expected error for nonexistent key")
	}

	// Valid key
	val, err := resolver.GetOutput("engine1", "key")
	if err != nil {
		t.Errorf("GetOutput failed: %v", err)
	}
	if val != "value" {
		t.Errorf("expected 'value', got '%v'", val)
	}

	// Get by name
	val, err = resolver.GetOutput("Engine 1", "key")
	if err != nil {
		t.Errorf("GetOutput by name failed: %v", err)
	}
	if val != "value" {
		t.Errorf("expected 'value', got '%v'", val)
	}
}

func TestDependencyGraphResolverWaitFor(t *testing.T) {
	graph := NewDependencyGraph()
	graph.AddNode("engine1", "Engine 1", nil)

	resolver := NewDependencyGraphResolver(graph)

	ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel()

	// Complete in background after delay
	go func() {
		time.Sleep(100 * time.Millisecond)
		resolver.MarkCompleted("engine1", nil)
	}()

	err := resolver.WaitFor(ctx, []string{"engine1"})
	if err != nil {
		t.Errorf("WaitFor failed: %v", err)
	}
}

func TestDependencyGraphResolverWaitForAlreadySatisfied(t *testing.T) {
	graph := NewDependencyGraph()
	graph.AddNode("engine1", "Engine 1", nil)

	resolver := NewDependencyGraphResolver(graph)
	resolver.MarkCompleted("engine1", nil)

	ctx := context.Background()
	err := resolver.WaitFor(ctx, []string{"engine1"})
	if err != nil {
		t.Errorf("WaitFor failed: %v", err)
	}
}

func TestDependencyGraphResolverWaitForTimeout(t *testing.T) {
	graph := NewDependencyGraph()
	graph.AddNode("engine1", "Engine 1", nil)

	resolver := NewDependencyGraphResolver(graph)

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	err := resolver.WaitFor(ctx, []string{"engine1"})
	if err == nil {
		t.Error("expected timeout error")
	}
}

func TestDependencyGraphResolverWaitForContextCancelled(t *testing.T) {
	graph := NewDependencyGraph()
	graph.AddNode("engine1", "Engine 1", nil)

	resolver := NewDependencyGraphResolver(graph)

	ctx, cancel := context.WithCancel(context.Background())

	// Cancel immediately
	go func() {
		time.Sleep(50 * time.Millisecond)
		cancel()
	}()

	err := resolver.WaitFor(ctx, []string{"engine1"})
	if err == nil {
		t.Error("expected context cancelled error")
	}
}

func TestDependencyGraphResolverReset(t *testing.T) {
	graph := NewDependencyGraph()
	graph.AddNode("engine1", "Engine 1", nil)

	resolver := NewDependencyGraphResolver(graph)
	resolver.MarkCompleted("engine1", map[string]interface{}{"key": "value"})

	if !resolver.IsCompleted("engine1") {
		t.Error("expected engine1 to be completed")
	}

	resolver.Reset()

	if resolver.IsCompleted("engine1") {
		t.Error("expected engine1 to not be completed after reset")
	}

	_, err := resolver.GetOutput("engine1", "key")
	if err == nil {
		t.Error("expected error for output after reset")
	}
}

func TestDependencyGraphResolverGetAllOutputs(t *testing.T) {
	graph := NewDependencyGraph()
	graph.AddNode("engine1", "Engine 1", nil)
	graph.AddNode("engine2", "Engine 2", nil)

	resolver := NewDependencyGraphResolver(graph)

	resolver.MarkCompleted("engine1", map[string]interface{}{"key1": "value1"})
	resolver.MarkCompleted("engine2", map[string]interface{}{"key2": "value2"})

	allOutputs := resolver.GetAllOutputs()

	if len(allOutputs) != 2 {
		t.Errorf("expected 2 engine outputs, got %d", len(allOutputs))
	}

	if allOutputs["engine1"]["key1"] != "value1" {
		t.Error("expected engine1 outputs")
	}
	if allOutputs["engine2"]["key2"] != "value2" {
		t.Error("expected engine2 outputs")
	}

	// Verify it's a copy
	allOutputs["engine1"]["key1"] = "modified"
	original, _ := resolver.GetOutput("engine1", "key1")
	if original != "value1" {
		t.Error("expected GetAllOutputs to return a copy")
	}
}

func TestDependencyGraphResolverFindEngineByName(t *testing.T) {
	graph := NewDependencyGraph()
	graph.AddNode("id1", "Engine 1", nil)
	graph.AddNode("id2", "Engine 2", nil)

	resolver := NewDependencyGraphResolver(graph)

	// Find by name
	id := resolver.findEngineByName("Engine 1")
	if id != "id1" {
		t.Errorf("expected 'id1', got '%s'", id)
	}

	// Find by ID
	id = resolver.findEngineByName("id2")
	if id != "id2" {
		t.Errorf("expected 'id2', got '%s'", id)
	}

	// Not found
	id = resolver.findEngineByName("nonexistent")
	if id != "" {
		t.Errorf("expected empty string, got '%s'", id)
	}
}

func TestDependencyGraphResolverConcurrentAccess(t *testing.T) {
	graph := NewDependencyGraph()
	for i := 0; i < 10; i++ {
		graph.AddNode(string(rune('A'+i)), "Node", nil)
	}

	resolver := NewDependencyGraphResolver(graph)

	var wg sync.WaitGroup
	iterations := 100

	// Concurrent completions
	for i := 0; i < iterations; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			nodeID := string(rune('A' + id%10))
			resolver.MarkCompleted(nodeID, map[string]interface{}{"iteration": id})
		}(i)
	}

	// Concurrent checks
	for i := 0; i < iterations; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			nodeID := string(rune('A' + id%10))
			resolver.IsCompleted(nodeID)
			resolver.IsSatisfied(nodeID, []string{})
		}(i)
	}

	wg.Wait()
}
