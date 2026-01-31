package graph

import (
	"strings"
	"testing"
)

func TestNew(t *testing.T) {
	g := New()
	if g == nil {
		t.Fatal("New() returned nil")
	}
	if len(g.nodes) != 0 {
		t.Error("New graph should have no nodes")
	}
}

func TestAddNode(t *testing.T) {
	g := New()

	node := &Node{
		Type: "Deployment",
		Name: "my-app",
	}
	g.AddNode(node)

	if len(g.Nodes()) != 1 {
		t.Errorf("Expected 1 node, got %d", len(g.Nodes()))
	}

	// ID should be auto-generated
	if node.ID != "Deployment/my-app" {
		t.Errorf("Expected ID 'Deployment/my-app', got %s", node.ID)
	}
}

func TestAddEdge(t *testing.T) {
	g := New()

	g.AddNode(&Node{Type: "Service", Name: "api"})
	g.AddNode(&Node{Type: "Deployment", Name: "web"})

	g.AddEdge("Deployment/web", "Service/api", "depends_on", true)

	edges := g.Edges()
	if len(edges) != 1 {
		t.Errorf("Expected 1 edge, got %d", len(edges))
	}

	if edges[0].From != "Deployment/web" {
		t.Error("Edge 'from' incorrect")
	}
	if edges[0].To != "Service/api" {
		t.Error("Edge 'to' incorrect")
	}
}

func TestGetNode(t *testing.T) {
	g := New()
	g.AddNode(&Node{Type: "Cluster", Name: "prod"})

	node := g.GetNode("Cluster/prod")
	if node == nil {
		t.Fatal("GetNode returned nil")
	}
	if node.Name != "prod" {
		t.Error("Wrong node returned")
	}

	// Non-existent node
	if g.GetNode("nonexistent") != nil {
		t.Error("Expected nil for non-existent node")
	}
}

func TestTopologicalSort(t *testing.T) {
	g := New()

	// Create a simple dependency chain: A -> B -> C
	g.AddNode(&Node{ID: "A", Type: "X", Name: "A"})
	g.AddNode(&Node{ID: "B", Type: "X", Name: "B"})
	g.AddNode(&Node{ID: "C", Type: "X", Name: "C"})

	g.AddEdge("B", "A", "depends_on", true)
	g.AddEdge("C", "B", "depends_on", true)

	sorted, err := g.TopologicalSort()
	if err != nil {
		t.Fatalf("TopologicalSort failed: %v", err)
	}

	if len(sorted) != 3 {
		t.Fatalf("Expected 3 nodes, got %d", len(sorted))
	}

	// A should come before B, B before C
	aIdx, bIdx, cIdx := -1, -1, -1
	for i, n := range sorted {
		switch n.ID {
		case "A":
			aIdx = i
		case "B":
			bIdx = i
		case "C":
			cIdx = i
		}
	}

	if aIdx > bIdx || bIdx > cIdx {
		t.Error("Topological order violated")
	}
}

func TestDetectCycles(t *testing.T) {
	g := New()

	// Create a cycle: A -> B -> C -> A
	g.AddNode(&Node{ID: "A", Type: "X", Name: "A"})
	g.AddNode(&Node{ID: "B", Type: "X", Name: "B"})
	g.AddNode(&Node{ID: "C", Type: "X", Name: "C"})

	g.AddEdge("A", "B", "depends_on", true)
	g.AddEdge("B", "C", "depends_on", true)
	g.AddEdge("C", "A", "depends_on", true)

	cycles := g.DetectCycles()
	if len(cycles) == 0 {
		t.Error("Expected cycle to be detected")
	}
}

func TestNoCycles(t *testing.T) {
	g := New()

	g.AddNode(&Node{ID: "A", Type: "X", Name: "A"})
	g.AddNode(&Node{ID: "B", Type: "X", Name: "B"})
	g.AddNode(&Node{ID: "C", Type: "X", Name: "C"})

	g.AddEdge("B", "A", "depends_on", true)
	g.AddEdge("C", "A", "depends_on", true)

	cycles := g.DetectCycles()
	if len(cycles) != 0 {
		t.Errorf("Expected no cycles, found %d", len(cycles))
	}
}

func TestGetDependencies(t *testing.T) {
	g := New()

	g.AddNode(&Node{ID: "A", Type: "X", Name: "A"})
	g.AddNode(&Node{ID: "B", Type: "X", Name: "B"})
	g.AddNode(&Node{ID: "C", Type: "X", Name: "C"})

	g.AddEdge("B", "A", "depends_on", true)
	g.AddEdge("C", "B", "depends_on", true)

	// Direct dependencies
	deps := g.GetDependencies("B", false)
	if len(deps) != 1 {
		t.Errorf("Expected 1 direct dependency, got %d", len(deps))
	}

	// Transitive dependencies
	deps = g.GetDependencies("C", true)
	if len(deps) != 2 {
		t.Errorf("Expected 2 transitive dependencies, got %d", len(deps))
	}
}

func TestGetDependents(t *testing.T) {
	g := New()

	g.AddNode(&Node{ID: "A", Type: "X", Name: "A"})
	g.AddNode(&Node{ID: "B", Type: "X", Name: "B"})
	g.AddNode(&Node{ID: "C", Type: "X", Name: "C"})

	g.AddEdge("B", "A", "depends_on", true)
	g.AddEdge("C", "A", "depends_on", true)

	// Direct dependents of A
	deps := g.GetDependents("A", false)
	if len(deps) != 2 {
		t.Errorf("Expected 2 dependents, got %d", len(deps))
	}
}

func TestFormat(t *testing.T) {
	g := New()

	g.AddNode(&Node{Type: "Cluster", Name: "prod", Status: "healthy"})
	g.AddNode(&Node{Type: "Deployment", Name: "api"})

	g.AddEdge("Deployment/api", "Cluster/prod", "depends_on", true)

	output := g.Format()

	expected := []string{
		"Resource Dependency Graph",
		"Cluster",
		"prod",
		"Deployment",
		"api",
	}

	for _, exp := range expected {
		if !strings.Contains(output, exp) {
			t.Errorf("Expected output to contain '%s'", exp)
		}
	}
}

func TestExportDOT(t *testing.T) {
	g := New()

	g.AddNode(&Node{Type: "Service", Name: "api"})
	g.AddNode(&Node{Type: "Database", Name: "postgres"})

	g.AddEdge("Service/api", "Database/postgres", "depends_on", true)

	dot := g.ExportDOT()

	if !strings.Contains(dot, "digraph") {
		t.Error("DOT output should contain 'digraph'")
	}
	if !strings.Contains(dot, "Service/api") {
		t.Error("DOT output should contain node ID")
	}
	if !strings.Contains(dot, "->") {
		t.Error("DOT output should contain edges")
	}
}

func TestExportMermaid(t *testing.T) {
	g := New()

	g.AddNode(&Node{Type: "Service", Name: "api"})
	g.AddNode(&Node{Type: "Database", Name: "postgres"})

	g.AddEdge("Service/api", "Database/postgres", "depends_on", true)

	mermaid := g.ExportMermaid()

	if !strings.Contains(mermaid, "graph LR") {
		t.Error("Mermaid output should contain 'graph LR'")
	}
	if !strings.Contains(mermaid, "-->") {
		t.Error("Mermaid output should contain edges")
	}
}

func TestEmptyGraph(t *testing.T) {
	g := New()

	output := g.Format()
	if !strings.Contains(output, "No resources") {
		t.Error("Empty graph format should indicate no resources")
	}

	sorted, err := g.TopologicalSort()
	if err != nil {
		t.Error("Empty graph should sort without error")
	}
	if len(sorted) != 0 {
		t.Error("Empty graph should have no sorted nodes")
	}
}
