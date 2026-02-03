// Package dcm implements Dynamic Configuration Management for automatic
// config generation and resource provisioning per deployment.
package dcm

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"sort"
	"sync"
	"time"
)

// Engine is the Dynamic Configuration Management engine
type Engine struct {
	resourceGraph   *ResourceGraph
	driverRegistry  *DriverRegistry
	configGenerator *ConfigGenerator
	deltaProcessor  *DeltaProcessor
	stateBackend    StateBackend
	mu              sync.RWMutex
}

// StateBackend interface for persistence
type StateBackend interface {
	Get(ctx context.Context, kind, id string) (interface{}, error)
	Put(ctx context.Context, kind, id string, value interface{}) error
	Delete(ctx context.Context, kind, id string) error
	List(ctx context.Context, kind string) ([]interface{}, error)
}

// EngineConfig contains engine configuration
type EngineConfig struct {
	DefaultTimeout      time.Duration
	ParallelProvisioning int
	DryRunDefault       bool
}

// NewEngine creates a new DCM engine
func NewEngine(backend StateBackend, config EngineConfig) *Engine {
	if config.DefaultTimeout == 0 {
		config.DefaultTimeout = 5 * time.Minute
	}
	if config.ParallelProvisioning == 0 {
		config.ParallelProvisioning = 5
	}

	return &Engine{
		resourceGraph:   NewResourceGraph(),
		driverRegistry:  NewDriverRegistry(),
		configGenerator: NewConfigGenerator(),
		deltaProcessor:  NewDeltaProcessor(),
		stateBackend:    backend,
	}
}

// ResourceGraph tracks resource dependencies
type ResourceGraph struct {
	nodes map[string]*ResourceNode
	edges map[string][]string // from -> []to
	mu    sync.RWMutex
}

// ResourceNode represents a resource in the graph
type ResourceNode struct {
	ID           string                 `json:"id"`
	Type         string                 `json:"type"`
	Name         string                 `json:"name"`
	Class        string                 `json:"class,omitempty"`
	Params       map[string]interface{} `json:"params,omitempty"`
	Outputs      map[string]string      `json:"outputs,omitempty"`
	Status       ResourceStatus         `json:"status"`
	Dependencies []string               `json:"dependencies,omitempty"`
	Application  string                 `json:"application,omitempty"`
	Environment  string                 `json:"environment,omitempty"`
	CreatedAt    time.Time              `json:"createdAt"`
	UpdatedAt    time.Time              `json:"updatedAt"`
}

// ResourceStatus represents the state of a resource
type ResourceStatus string

const (
	StatusPending      ResourceStatus = "pending"
	StatusProvisioning ResourceStatus = "provisioning"
	StatusReady        ResourceStatus = "ready"
	StatusFailed       ResourceStatus = "failed"
	StatusDeleting     ResourceStatus = "deleting"
	StatusDeleted      ResourceStatus = "deleted"
)

// NewResourceGraph creates a new resource graph
func NewResourceGraph() *ResourceGraph {
	return &ResourceGraph{
		nodes: make(map[string]*ResourceNode),
		edges: make(map[string][]string),
	}
}

// AddNode adds a resource node to the graph
func (g *ResourceGraph) AddNode(node *ResourceNode) {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.nodes[node.ID] = node
}

// AddEdge adds a dependency edge
func (g *ResourceGraph) AddEdge(from, to string) {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.edges[from] = append(g.edges[from], to)
}

// GetNode returns a node by ID
func (g *ResourceGraph) GetNode(id string) *ResourceNode {
	g.mu.RLock()
	defer g.mu.RUnlock()
	return g.nodes[id]
}

// GetDependencies returns all dependencies of a node
func (g *ResourceGraph) GetDependencies(id string) []*ResourceNode {
	g.mu.RLock()
	defer g.mu.RUnlock()

	var deps []*ResourceNode
	for _, depID := range g.edges[id] {
		if node := g.nodes[depID]; node != nil {
			deps = append(deps, node)
		}
	}
	return deps
}

// TopologicalSort returns nodes in dependency order
func (g *ResourceGraph) TopologicalSort() ([]*ResourceNode, error) {
	g.mu.RLock()
	defer g.mu.RUnlock()

	visited := make(map[string]bool)
	recStack := make(map[string]bool)
	var result []*ResourceNode

	var visit func(id string) error
	visit = func(id string) error {
		if recStack[id] {
			return fmt.Errorf("circular dependency detected: %s", id)
		}
		if visited[id] {
			return nil
		}

		recStack[id] = true

		for _, depID := range g.edges[id] {
			if err := visit(depID); err != nil {
				return err
			}
		}

		recStack[id] = false
		visited[id] = true

		if node := g.nodes[id]; node != nil {
			result = append([]*ResourceNode{node}, result...)
		}

		return nil
	}

	for id := range g.nodes {
		if err := visit(id); err != nil {
			return nil, err
		}
	}

	return result, nil
}

// DriverRegistry manages resource drivers
type DriverRegistry struct {
	drivers map[string]ResourceDriver
	mu      sync.RWMutex
}

// ResourceDriver provisions and manages a resource type
type ResourceDriver interface {
	Type() string
	Provision(ctx context.Context, node *ResourceNode) error
	Update(ctx context.Context, node *ResourceNode) error
	Delete(ctx context.Context, node *ResourceNode) error
	GetOutputs(ctx context.Context, node *ResourceNode) (map[string]string, error)
	Validate(node *ResourceNode) error
}

// NewDriverRegistry creates a new driver registry
func NewDriverRegistry() *DriverRegistry {
	return &DriverRegistry{
		drivers: make(map[string]ResourceDriver),
	}
}

// Register registers a driver
func (r *DriverRegistry) Register(driver ResourceDriver) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.drivers[driver.Type()] = driver
}

// Get returns a driver by type
func (r *DriverRegistry) Get(resourceType string) (ResourceDriver, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	driver, ok := r.drivers[resourceType]
	return driver, ok
}

// ConfigGenerator generates configuration for deployments
type ConfigGenerator struct {
	templates map[string]ConfigTemplate
}

// ConfigTemplate defines how to generate config for a resource type
type ConfigTemplate struct {
	Type           string            `json:"type"`
	OutputMappings map[string]string `json:"outputMappings"`
	EnvVarPrefix   string            `json:"envVarPrefix,omitempty"`
}

// NewConfigGenerator creates a new config generator
func NewConfigGenerator() *ConfigGenerator {
	return &ConfigGenerator{
		templates: make(map[string]ConfigTemplate),
	}
}

// GenerateConfig generates configuration from resource outputs
func (g *ConfigGenerator) GenerateConfig(resources []*ResourceNode) map[string]string {
	config := make(map[string]string)

	for _, resource := range resources {
		prefix := resource.Name
		for key, value := range resource.Outputs {
			configKey := fmt.Sprintf("%s_%s", toEnvVar(prefix), toEnvVar(key))
			config[configKey] = value
		}
	}

	return config
}

// DeltaProcessor handles incremental deployments
type DeltaProcessor struct{}

// NewDeltaProcessor creates a new delta processor
func NewDeltaProcessor() *DeltaProcessor {
	return &DeltaProcessor{}
}

// Delta represents changes between deployment sets
type Delta struct {
	Added    []*ResourceNode `json:"added"`
	Modified []*ResourceNode `json:"modified"`
	Removed  []*ResourceNode `json:"removed"`
}

// ComputeDelta computes the difference between two resource sets
func (p *DeltaProcessor) ComputeDelta(current, desired []*ResourceNode) *Delta {
	delta := &Delta{
		Added:    make([]*ResourceNode, 0),
		Modified: make([]*ResourceNode, 0),
		Removed:  make([]*ResourceNode, 0),
	}

	currentMap := make(map[string]*ResourceNode)
	for _, node := range current {
		currentMap[node.ID] = node
	}

	desiredMap := make(map[string]*ResourceNode)
	for _, node := range desired {
		desiredMap[node.ID] = node
	}

	// Find added and modified
	for id, desired := range desiredMap {
		if current, exists := currentMap[id]; exists {
			if !nodesEqual(current, desired) {
				delta.Modified = append(delta.Modified, desired)
			}
		} else {
			delta.Added = append(delta.Added, desired)
		}
	}

	// Find removed
	for id, current := range currentMap {
		if _, exists := desiredMap[id]; !exists {
			delta.Removed = append(delta.Removed, current)
		}
	}

	return delta
}

func nodesEqual(a, b *ResourceNode) bool {
	// Compare essential fields
	if a.Type != b.Type || a.Name != b.Name || a.Class != b.Class {
		return false
	}

	// Compare params by computing hash
	aHash := hashParams(a.Params)
	bHash := hashParams(b.Params)
	return aHash == bHash
}

func hashParams(params map[string]interface{}) string {
	data, _ := json.Marshal(params)
	hash := sha256.Sum256(data)
	return hex.EncodeToString(hash[:8])
}

// DeploymentSet represents an immutable set of resources for a deployment
type DeploymentSet struct {
	ID          string                 `json:"id"`
	Application string                 `json:"application"`
	Environment string                 `json:"environment"`
	Version     int                    `json:"version"`
	Resources   []*ResourceNode        `json:"resources"`
	Config      map[string]interface{} `json:"config,omitempty"`
	Status      DeploymentStatus       `json:"status"`
	CreatedAt   time.Time              `json:"createdAt"`
	DeployedAt  *time.Time             `json:"deployedAt,omitempty"`
	Hash        string                 `json:"hash"`
}

// DeploymentStatus represents the status of a deployment
type DeploymentStatus string

const (
	DeploymentPending    DeploymentStatus = "pending"
	DeploymentInProgress DeploymentStatus = "in_progress"
	DeploymentSucceeded  DeploymentStatus = "succeeded"
	DeploymentFailed     DeploymentStatus = "failed"
	DeploymentRolledBack DeploymentStatus = "rolled_back"
)

// CreateDeploymentSet creates a new deployment set
func (e *Engine) CreateDeploymentSet(ctx context.Context, app, env string, resources []*ResourceNode) (*DeploymentSet, error) {
	e.mu.Lock()
	defer e.mu.Unlock()

	// Get current version
	version := 1
	if e.stateBackend != nil {
		existing, _ := e.listDeploymentSets(ctx, app, env)
		if len(existing) > 0 {
			version = existing[len(existing)-1].Version + 1
		}
	}

	// Compute hash
	hash := e.computeSetHash(resources)

	set := &DeploymentSet{
		ID:          fmt.Sprintf("%s-%s-v%d", app, env, version),
		Application: app,
		Environment: env,
		Version:     version,
		Resources:   resources,
		Status:      DeploymentPending,
		CreatedAt:   time.Now(),
		Hash:        hash,
	}

	// Persist
	if e.stateBackend != nil {
		if err := e.stateBackend.Put(ctx, "DeploymentSet", set.ID, set); err != nil {
			return nil, err
		}
	}

	return set, nil
}

func (e *Engine) listDeploymentSets(ctx context.Context, app, env string) ([]*DeploymentSet, error) {
	items, err := e.stateBackend.List(ctx, "DeploymentSet")
	if err != nil {
		return nil, err
	}

	var sets []*DeploymentSet
	for _, item := range items {
		if set, ok := item.(*DeploymentSet); ok {
			if set.Application == app && set.Environment == env {
				sets = append(sets, set)
			}
		}
	}

	sort.Slice(sets, func(i, j int) bool {
		return sets[i].Version < sets[j].Version
	})

	return sets, nil
}

func (e *Engine) computeSetHash(resources []*ResourceNode) string {
	data, _ := json.Marshal(resources)
	hash := sha256.Sum256(data)
	return hex.EncodeToString(hash[:])
}

// Deploy deploys a deployment set
func (e *Engine) Deploy(ctx context.Context, set *DeploymentSet) error {
	set.Status = DeploymentInProgress

	// Build resource graph
	for _, node := range set.Resources {
		e.resourceGraph.AddNode(node)
		for _, dep := range node.Dependencies {
			e.resourceGraph.AddEdge(node.ID, dep)
		}
	}

	// Get resources in dependency order
	ordered, err := e.resourceGraph.TopologicalSort()
	if err != nil {
		set.Status = DeploymentFailed
		return fmt.Errorf("failed to order resources: %w", err)
	}

	// Provision resources
	for _, node := range ordered {
		if err := e.provisionResource(ctx, node); err != nil {
			set.Status = DeploymentFailed
			return fmt.Errorf("failed to provision %s: %w", node.ID, err)
		}
	}

	// Generate config
	set.Config = make(map[string]interface{})
	config := e.configGenerator.GenerateConfig(ordered)
	for k, v := range config {
		set.Config[k] = v
	}

	// Mark as deployed
	now := time.Now()
	set.DeployedAt = &now
	set.Status = DeploymentSucceeded

	// Persist
	if e.stateBackend != nil {
		if err := e.stateBackend.Put(ctx, "DeploymentSet", set.ID, set); err != nil {
			return err
		}
	}

	return nil
}

func (e *Engine) provisionResource(ctx context.Context, node *ResourceNode) error {
	driver, ok := e.driverRegistry.Get(node.Type)
	if !ok {
		// No driver, mark as ready (placeholder)
		node.Status = StatusReady
		node.Outputs = generatePlaceholderOutputs(node)
		return nil
	}

	// Validate
	if err := driver.Validate(node); err != nil {
		return fmt.Errorf("validation failed: %w", err)
	}

	// Provision
	node.Status = StatusProvisioning
	if err := driver.Provision(ctx, node); err != nil {
		node.Status = StatusFailed
		return err
	}

	// Get outputs
	outputs, err := driver.GetOutputs(ctx, node)
	if err != nil {
		return fmt.Errorf("failed to get outputs: %w", err)
	}
	node.Outputs = outputs
	node.Status = StatusReady
	node.UpdatedAt = time.Now()

	return nil
}

func generatePlaceholderOutputs(node *ResourceNode) map[string]string {
	outputs := make(map[string]string)

	switch node.Type {
	case "postgres", "mysql", "mongodb":
		outputs["host"] = fmt.Sprintf("%s.%s.svc.cluster.local", node.Name, node.Environment)
		outputs["port"] = "5432"
		outputs["name"] = node.Name
		outputs["username"] = node.Name
		outputs["password"] = fmt.Sprintf("${secret:%s-password}", node.Name)
		outputs["connection_string"] = fmt.Sprintf("postgres://%s:${secret:%s-password}@%s:5432/%s",
			node.Name, node.Name, outputs["host"], node.Name)
	case "redis":
		outputs["host"] = fmt.Sprintf("%s.%s.svc.cluster.local", node.Name, node.Environment)
		outputs["port"] = "6379"
		outputs["url"] = fmt.Sprintf("redis://%s:6379", outputs["host"])
	case "s3":
		outputs["bucket"] = fmt.Sprintf("%s-%s", node.Application, node.Name)
		outputs["region"] = "us-east-1"
		outputs["endpoint"] = "s3.amazonaws.com"
	}

	return outputs
}

// Rollback rolls back to a previous deployment set
func (e *Engine) Rollback(ctx context.Context, app, env string, version int) (*DeploymentSet, error) {
	sets, err := e.listDeploymentSets(ctx, app, env)
	if err != nil {
		return nil, err
	}

	var targetSet *DeploymentSet
	for _, set := range sets {
		if set.Version == version {
			targetSet = set
			break
		}
	}

	if targetSet == nil {
		return nil, fmt.Errorf("deployment set version %d not found", version)
	}

	// Create a new set from the target
	newSet, err := e.CreateDeploymentSet(ctx, app, env, targetSet.Resources)
	if err != nil {
		return nil, err
	}

	// Deploy
	if err := e.Deploy(ctx, newSet); err != nil {
		return nil, err
	}

	newSet.Status = DeploymentRolledBack
	return newSet, nil
}

// GetDeploymentSet retrieves a deployment set
func (e *Engine) GetDeploymentSet(ctx context.Context, id string) (*DeploymentSet, error) {
	if e.stateBackend == nil {
		return nil, fmt.Errorf("no state backend configured")
	}

	item, err := e.stateBackend.Get(ctx, "DeploymentSet", id)
	if err != nil {
		return nil, err
	}

	set, ok := item.(*DeploymentSet)
	if !ok {
		return nil, fmt.Errorf("invalid deployment set data")
	}

	return set, nil
}

// ListDeploymentSets lists deployment sets for an application
func (e *Engine) ListDeploymentSets(ctx context.Context, app, env string) ([]*DeploymentSet, error) {
	return e.listDeploymentSets(ctx, app, env)
}

// DiffDeploymentSets compares two deployment sets
func (e *Engine) DiffDeploymentSets(set1, set2 *DeploymentSet) *Delta {
	return e.deltaProcessor.ComputeDelta(set1.Resources, set2.Resources)
}

// RegisterDriver registers a resource driver
func (e *Engine) RegisterDriver(driver ResourceDriver) {
	e.driverRegistry.Register(driver)
}

// Helper functions

func toEnvVar(s string) string {
	result := ""
	for _, c := range s {
		if (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9') {
			result += string(c)
		} else if c >= 'a' && c <= 'z' {
			result += string(c - 32) // to uppercase
		} else if c == '-' || c == '.' || c == '_' {
			result += "_"
		}
	}
	return result
}
