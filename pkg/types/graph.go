package types

import (
	"time"
)

// ResourceNode represents a node in the resource graph
type ResourceNode struct {
	// ID is the unique identifier (kind/name)
	ID string `json:"id" yaml:"id"`

	// Kind is the resource type (Service, Platform, Database, etc.)
	Kind string `json:"kind" yaml:"kind"`

	// Name is the resource name
	Name string `json:"name" yaml:"name"`

	// Namespace or organization
	Namespace string `json:"namespace,omitempty" yaml:"namespace,omitempty"`

	// Provider that manages this resource
	Provider string `json:"provider,omitempty" yaml:"provider,omitempty"`

	// Status of the resource
	Status string `json:"status,omitempty" yaml:"status,omitempty"`

	// Health status
	Health string `json:"health,omitempty" yaml:"health,omitempty"`

	// Criticality level (critical, high, medium, low)
	Criticality string `json:"criticality,omitempty" yaml:"criticality,omitempty"`

	// Team that owns this resource
	Team string `json:"team,omitempty" yaml:"team,omitempty"`

	// Environment
	Environment string `json:"environment,omitempty" yaml:"environment,omitempty"`

	// Labels for filtering
	Labels map[string]string `json:"labels,omitempty" yaml:"labels,omitempty"`

	// Metadata
	Metadata map[string]interface{} `json:"metadata,omitempty" yaml:"metadata,omitempty"`

	// CreatedAt timestamp
	CreatedAt time.Time `json:"created_at,omitempty" yaml:"created_at,omitempty"`

	// UpdatedAt timestamp
	UpdatedAt time.Time `json:"updated_at,omitempty" yaml:"updated_at,omitempty"`
}

// EdgeType represents the type of relationship between resources
type EdgeType string

const (
	// EdgeDependsOn means source depends on target
	EdgeDependsOn EdgeType = "depends_on"

	// EdgeProvisions means source provisions target
	EdgeProvisions EdgeType = "provisions"

	// EdgeContains means source contains target
	EdgeContains EdgeType = "contains"

	// EdgeConnectsTo means source connects to target
	EdgeConnectsTo EdgeType = "connects_to"

	// EdgeReadsFrom means source reads from target
	EdgeReadsFrom EdgeType = "reads_from"

	// EdgeWritesTo means source writes to target
	EdgeWritesTo EdgeType = "writes_to"

	// EdgeManagedBy means source is managed by target
	EdgeManagedBy EdgeType = "managed_by"

	// EdgeMonitors means source monitors target
	EdgeMonitors EdgeType = "monitors"

	// EdgeDeploysTo means source deploys to target
	EdgeDeploysTo EdgeType = "deploys_to"
)

// ResourceEdge represents a directed edge between resources
type ResourceEdge struct {
	// ID is the unique identifier
	ID string `json:"id" yaml:"id"`

	// Source node ID
	Source string `json:"source" yaml:"source"`

	// Target node ID
	Target string `json:"target" yaml:"target"`

	// Type of relationship
	Type EdgeType `json:"type" yaml:"type"`

	// Required indicates if this dependency is required
	Required bool `json:"required" yaml:"required"`

	// Labels for filtering
	Labels map[string]string `json:"labels,omitempty" yaml:"labels,omitempty"`

	// Metadata
	Metadata map[string]interface{} `json:"metadata,omitempty" yaml:"metadata,omitempty"`
}

// ResourceGraph represents the complete dependency graph
type ResourceGraph struct {
	// Version of the graph schema
	Version string `json:"version" yaml:"version"`

	// Nodes in the graph
	Nodes map[string]*ResourceNode `json:"nodes" yaml:"nodes"`

	// Edges in the graph
	Edges []*ResourceEdge `json:"edges" yaml:"edges"`

	// UpdatedAt is when the graph was last updated
	UpdatedAt time.Time `json:"updated_at" yaml:"updated_at"`
}

// ImpactReport represents the impact of changing a resource
type ImpactReport struct {
	// Resource being changed
	Resource string `json:"resource" yaml:"resource"`

	// DirectlyAffected are resources that directly depend on the changed resource
	DirectlyAffected []string `json:"directly_affected" yaml:"directly_affected"`

	// TransitivelyAffected are resources affected through dependency chains
	TransitivelyAffected []string `json:"transitively_affected" yaml:"transitively_affected"`

	// BlastRadius is the total number of affected resources
	BlastRadius int `json:"blast_radius" yaml:"blast_radius"`

	// CriticalAffected count of critical resources affected
	CriticalAffected int `json:"critical_affected" yaml:"critical_affected"`

	// AffectedTeams lists teams whose resources are affected
	AffectedTeams []string `json:"affected_teams" yaml:"affected_teams"`

	// AffectedEnvironments lists environments that are affected
	AffectedEnvironments []string `json:"affected_environments" yaml:"affected_environments"`

	// RiskLevel based on criticality and blast radius
	RiskLevel string `json:"risk_level" yaml:"risk_level"`

	// Recommendations for the change
	Recommendations []string `json:"recommendations,omitempty" yaml:"recommendations,omitempty"`
}

// PathResult represents a path between two resources
type PathResult struct {
	// Source node
	Source string `json:"source" yaml:"source"`

	// Target node
	Target string `json:"target" yaml:"target"`

	// Path is the sequence of node IDs
	Path []string `json:"path" yaml:"path"`

	// Edges are the edges traversed
	Edges []*ResourceEdge `json:"edges" yaml:"edges"`

	// Length is the path length
	Length int `json:"length" yaml:"length"`
}

// CycleResult represents a detected cycle in the graph
type CycleResult struct {
	// Nodes involved in the cycle
	Nodes []string `json:"nodes" yaml:"nodes"`

	// Edges forming the cycle
	Edges []*ResourceEdge `json:"edges" yaml:"edges"`
}

// GraphStats provides statistics about the graph
type GraphStats struct {
	// NodeCount is the total number of nodes
	NodeCount int `json:"node_count" yaml:"node_count"`

	// EdgeCount is the total number of edges
	EdgeCount int `json:"edge_count" yaml:"edge_count"`

	// NodesByKind counts nodes by kind
	NodesByKind map[string]int `json:"nodes_by_kind" yaml:"nodes_by_kind"`

	// NodesByEnvironment counts nodes by environment
	NodesByEnvironment map[string]int `json:"nodes_by_environment" yaml:"nodes_by_environment"`

	// NodesByTeam counts nodes by team
	NodesByTeam map[string]int `json:"nodes_by_team" yaml:"nodes_by_team"`

	// EdgesByType counts edges by type
	EdgesByType map[EdgeType]int `json:"edges_by_type" yaml:"edges_by_type"`

	// AverageOutDegree is the average number of outgoing edges
	AverageOutDegree float64 `json:"avg_out_degree" yaml:"avg_out_degree"`

	// AverageInDegree is the average number of incoming edges
	AverageInDegree float64 `json:"avg_in_degree" yaml:"avg_in_degree"`

	// MaxDepth is the maximum dependency chain length
	MaxDepth int `json:"max_depth" yaml:"max_depth"`

	// HasCycles indicates if the graph has cycles
	HasCycles bool `json:"has_cycles" yaml:"has_cycles"`

	// CycleCount is the number of cycles detected
	CycleCount int `json:"cycle_count" yaml:"cycle_count"`

	// CriticalNodes are nodes with high blast radius
	CriticalNodes []string `json:"critical_nodes" yaml:"critical_nodes"`
}

// GraphQuery defines a query against the graph
type GraphQuery struct {
	// NodeFilter filters nodes
	NodeFilter *NodeFilter `json:"node_filter,omitempty" yaml:"node_filter,omitempty"`

	// EdgeFilter filters edges
	EdgeFilter *EdgeFilter `json:"edge_filter,omitempty" yaml:"edge_filter,omitempty"`

	// StartNode for traversal queries
	StartNode string `json:"start_node,omitempty" yaml:"start_node,omitempty"`

	// EndNode for path queries
	EndNode string `json:"end_node,omitempty" yaml:"end_node,omitempty"`

	// MaxDepth limits traversal depth
	MaxDepth int `json:"max_depth,omitempty" yaml:"max_depth,omitempty"`

	// Direction for traversal (upstream, downstream, both)
	Direction string `json:"direction,omitempty" yaml:"direction,omitempty"`
}

// NodeFilter filters nodes in queries
type NodeFilter struct {
	Kinds        []string          `json:"kinds,omitempty" yaml:"kinds,omitempty"`
	Names        []string          `json:"names,omitempty" yaml:"names,omitempty"`
	Teams        []string          `json:"teams,omitempty" yaml:"teams,omitempty"`
	Environments []string          `json:"environments,omitempty" yaml:"environments,omitempty"`
	Statuses     []string          `json:"statuses,omitempty" yaml:"statuses,omitempty"`
	Labels       map[string]string `json:"labels,omitempty" yaml:"labels,omitempty"`
}

// EdgeFilter filters edges in queries
type EdgeFilter struct {
	Types    []EdgeType `json:"types,omitempty" yaml:"types,omitempty"`
	Required *bool      `json:"required,omitempty" yaml:"required,omitempty"`
}
