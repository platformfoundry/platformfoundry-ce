package platform

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// Manager handles platform-as-code operations
type Manager struct {
	platforms    map[string]*Platform
	applications map[string]*Application
	goldenPaths  map[string]*GoldenPath
	mu           sync.RWMutex
}

// NewManager creates a new platform manager
func NewManager() *Manager {
	m := &Manager{
		platforms:    make(map[string]*Platform),
		applications: make(map[string]*Application),
		goldenPaths:  make(map[string]*GoldenPath),
	}

	// Load built-in golden paths
	m.loadBuiltInGoldenPaths()

	return m
}

// loadBuiltInGoldenPaths creates default golden paths
func (m *Manager) loadBuiltInGoldenPaths() {
	paths := []*GoldenPath{
		{
			Name:        "microservice-go",
			Description: "Go microservice with standard tooling",
			Template:    "microservice",
			Language:    "go",
			Framework:   "gin",
			Resources: []ResourceType{
				{Type: "kubernetes-deployment", Required: true},
				{Type: "kubernetes-service", Required: true},
				{Type: "kubernetes-ingress", Required: false},
			},
			Pipelines:     []string{"build", "test", "security-scan", "deploy"},
			Observability: []string{"metrics", "traces", "logs"},
			Security: &SecurityConfig{
				ImageScanning:   true,
				DependencyCheck: true,
				SAST:            true,
			},
			Tags: []string{"backend", "api", "go"},
		},
		{
			Name:        "microservice-nodejs",
			Description: "Node.js microservice with Express",
			Template:    "microservice",
			Language:    "nodejs",
			Framework:   "express",
			Resources: []ResourceType{
				{Type: "kubernetes-deployment", Required: true},
				{Type: "kubernetes-service", Required: true},
			},
			Pipelines:     []string{"build", "test", "security-scan", "deploy"},
			Observability: []string{"metrics", "traces", "logs"},
			Security: &SecurityConfig{
				ImageScanning:   true,
				DependencyCheck: true,
			},
			Tags: []string{"backend", "api", "nodejs"},
		},
		{
			Name:        "microservice-python",
			Description: "Python microservice with FastAPI",
			Template:    "microservice",
			Language:    "python",
			Framework:   "fastapi",
			Resources: []ResourceType{
				{Type: "kubernetes-deployment", Required: true},
				{Type: "kubernetes-service", Required: true},
			},
			Pipelines:     []string{"build", "test", "security-scan", "deploy"},
			Observability: []string{"metrics", "traces", "logs"},
			Tags:          []string{"backend", "api", "python"},
		},
		{
			Name:        "web-app-react",
			Description: "React single-page application",
			Template:    "web-app",
			Language:    "typescript",
			Framework:   "react",
			Resources: []ResourceType{
				{Type: "s3-bucket", Required: true},
				{Type: "cloudfront", Required: true},
			},
			Pipelines:     []string{"build", "test", "deploy"},
			Observability: []string{"rum", "logs"},
			Tags:          []string{"frontend", "spa", "react"},
		},
		{
			Name:        "worker-go",
			Description: "Background worker in Go",
			Template:    "worker",
			Language:    "go",
			Resources: []ResourceType{
				{Type: "kubernetes-deployment", Required: true},
				{Type: "sqs-queue", Required: false},
				{Type: "redis", Required: false},
			},
			Pipelines:     []string{"build", "test", "deploy"},
			Observability: []string{"metrics", "logs"},
			Tags:          []string{"backend", "worker", "async"},
		},
		{
			Name:        "data-pipeline",
			Description: "Data processing pipeline",
			Template:    "data-pipeline",
			Language:    "python",
			Framework:   "spark",
			Resources: []ResourceType{
				{Type: "s3-bucket", Required: true},
				{Type: "glue-catalog", Required: false},
				{Type: "emr-cluster", Required: false},
			},
			Pipelines:     []string{"build", "test", "deploy"},
			Observability: []string{"metrics", "logs"},
			Tags:          []string{"data", "etl", "spark"},
		},
		{
			Name:        "api-with-database",
			Description: "API service with PostgreSQL database",
			Template:    "api-database",
			Language:    "go",
			Resources: []ResourceType{
				{Type: "kubernetes-deployment", Required: true},
				{Type: "kubernetes-service", Required: true},
				{Type: "postgres", Required: true},
				{Type: "redis", Required: false, Shareable: true},
			},
			Pipelines:     []string{"build", "test", "migrate", "deploy"},
			Observability: []string{"metrics", "traces", "logs"},
			Tags:          []string{"backend", "api", "database"},
		},
	}

	for _, p := range paths {
		m.goldenPaths[p.Name] = p
	}
}

// RegisterPlatform registers a platform configuration
func (m *Manager) RegisterPlatform(ctx context.Context, platform *Platform) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if platform.Metadata.Name == "" {
		return fmt.Errorf("platform name is required")
	}

	// Set defaults
	if platform.APIVersion == "" {
		platform.APIVersion = "platformfoundry.io/v1"
	}
	if platform.Kind == "" {
		platform.Kind = "Platform"
	}

	now := time.Now()
	platform.Metadata.CreatedAt = now
	platform.Metadata.UpdatedAt = now

	// Initialize status
	platform.Status = &PlatformStatus{
		Phase:       PlatformPhaseActive,
		LastApplied: now,
		Conditions: []PlatformCondition{
			{
				Type:               "Ready",
				Status:             "True",
				LastTransitionTime: now,
				Reason:             "PlatformRegistered",
				Message:            "Platform registered successfully",
			},
		},
	}

	// Register golden paths from this platform
	for i := range platform.Spec.GoldenPaths {
		gp := &platform.Spec.GoldenPaths[i]
		m.goldenPaths[gp.Name] = gp
	}

	m.platforms[platform.Metadata.Name] = platform
	return nil
}

// GetPlatform retrieves a platform by name
func (m *Manager) GetPlatform(name string) (*Platform, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	platform, ok := m.platforms[name]
	if !ok {
		return nil, fmt.Errorf("platform not found: %s", name)
	}
	return platform, nil
}

// ListPlatforms returns all platforms
func (m *Manager) ListPlatforms() []*Platform {
	m.mu.RLock()
	defer m.mu.RUnlock()

	result := make([]*Platform, 0, len(m.platforms))
	for _, p := range m.platforms {
		result = append(result, p)
	}
	return result
}

// UpdatePlatform updates a platform configuration
func (m *Manager) UpdatePlatform(ctx context.Context, platform *Platform) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	existing, ok := m.platforms[platform.Metadata.Name]
	if !ok {
		return fmt.Errorf("platform not found: %s", platform.Metadata.Name)
	}

	// Preserve creation time
	platform.Metadata.CreatedAt = existing.Metadata.CreatedAt
	platform.Metadata.UpdatedAt = time.Now()

	// Update golden paths
	for i := range platform.Spec.GoldenPaths {
		gp := &platform.Spec.GoldenPaths[i]
		m.goldenPaths[gp.Name] = gp
	}

	m.platforms[platform.Metadata.Name] = platform
	return nil
}

// DeletePlatform removes a platform
func (m *Manager) DeletePlatform(ctx context.Context, name string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, ok := m.platforms[name]; !ok {
		return fmt.Errorf("platform not found: %s", name)
	}

	delete(m.platforms, name)
	return nil
}

// GetGoldenPath retrieves a golden path by name
func (m *Manager) GetGoldenPath(name string) (*GoldenPath, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	gp, ok := m.goldenPaths[name]
	if !ok {
		return nil, fmt.Errorf("golden path not found: %s", name)
	}
	return gp, nil
}

// ListGoldenPaths returns all golden paths
func (m *Manager) ListGoldenPaths() []*GoldenPath {
	m.mu.RLock()
	defer m.mu.RUnlock()

	result := make([]*GoldenPath, 0, len(m.goldenPaths))
	for _, gp := range m.goldenPaths {
		result = append(result, gp)
	}
	return result
}

// ListGoldenPathsByTag returns golden paths matching a tag
func (m *Manager) ListGoldenPathsByTag(tag string) []*GoldenPath {
	m.mu.RLock()
	defer m.mu.RUnlock()

	result := make([]*GoldenPath, 0)
	for _, gp := range m.goldenPaths {
		for _, t := range gp.Tags {
			if t == tag {
				result = append(result, gp)
				break
			}
		}
	}
	return result
}

// CreateApplication creates a new application from a golden path
func (m *Manager) CreateApplication(ctx context.Context, app *Application) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if app.Metadata.Name == "" {
		return fmt.Errorf("application name is required")
	}

	if app.Spec.GoldenPath == "" {
		return fmt.Errorf("golden path is required")
	}

	// Validate golden path exists
	if _, ok := m.goldenPaths[app.Spec.GoldenPath]; !ok {
		return fmt.Errorf("golden path not found: %s", app.Spec.GoldenPath)
	}

	// Set defaults
	if app.APIVersion == "" {
		app.APIVersion = "platformfoundry.io/v1"
	}
	if app.Kind == "" {
		app.Kind = "Application"
	}

	// Initialize status
	app.Status = &ApplicationStatus{
		Phase:       "Pending",
		Deployments: make(map[string]DeploymentInfo),
		Resources:   make([]ResourceStatus, 0),
	}

	m.applications[app.Metadata.Name] = app
	return nil
}

// GetApplication retrieves an application by name
func (m *Manager) GetApplication(name string) (*Application, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	app, ok := m.applications[name]
	if !ok {
		return nil, fmt.Errorf("application not found: %s", name)
	}
	return app, nil
}

// ListApplications returns all applications
func (m *Manager) ListApplications() []*Application {
	m.mu.RLock()
	defer m.mu.RUnlock()

	result := make([]*Application, 0, len(m.applications))
	for _, app := range m.applications {
		result = append(result, app)
	}
	return result
}

// ListApplicationsByGoldenPath returns applications using a golden path
func (m *Manager) ListApplicationsByGoldenPath(goldenPath string) []*Application {
	m.mu.RLock()
	defer m.mu.RUnlock()

	result := make([]*Application, 0)
	for _, app := range m.applications {
		if app.Spec.GoldenPath == goldenPath {
			result = append(result, app)
		}
	}
	return result
}

// DeleteApplication removes an application
func (m *Manager) DeleteApplication(ctx context.Context, name string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, ok := m.applications[name]; !ok {
		return fmt.Errorf("application not found: %s", name)
	}

	delete(m.applications, name)
	return nil
}

// DetectDrift checks for configuration drift
func (m *Manager) DetectDrift(ctx context.Context, platformName string) ([]DriftDetail, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	platform, ok := m.platforms[platformName]
	if !ok {
		return nil, fmt.Errorf("platform not found: %s", platformName)
	}

	// In a real implementation, this would compare actual state with desired state
	// For now, return empty (no drift)
	drifts := make([]DriftDetail, 0)

	platform.Status.DriftDetected = len(drifts) > 0
	platform.Status.DriftDetails = drifts

	return drifts, nil
}

// ValidatePlatform validates a platform configuration
func (m *Manager) ValidatePlatform(platform *Platform) []string {
	errors := make([]string, 0)

	if platform.Metadata.Name == "" {
		errors = append(errors, "platform name is required")
	}

	// Validate golden paths
	for i, gp := range platform.Spec.GoldenPaths {
		if gp.Name == "" {
			errors = append(errors, fmt.Sprintf("golden path %d: name is required", i))
		}
		if gp.Template == "" {
			errors = append(errors, fmt.Sprintf("golden path %s: template is required", gp.Name))
		}
	}

	// Validate environments
	for i, env := range platform.Spec.Environments {
		if env.Name == "" {
			errors = append(errors, fmt.Sprintf("environment %d: name is required", i))
		}
	}

	return errors
}

// ValidateApplication validates an application configuration
func (m *Manager) ValidateApplication(app *Application) []string {
	errors := make([]string, 0)

	if app.Metadata.Name == "" {
		errors = append(errors, "application name is required")
	}

	if app.Spec.GoldenPath == "" {
		errors = append(errors, "golden path is required")
	} else {
		if _, ok := m.goldenPaths[app.Spec.GoldenPath]; !ok {
			errors = append(errors, fmt.Sprintf("golden path not found: %s", app.Spec.GoldenPath))
		}
	}

	return errors
}

// GetPlatformStats returns statistics about a platform
func (m *Manager) GetPlatformStats(platformName string) map[string]interface{} {
	m.mu.RLock()
	defer m.mu.RUnlock()

	stats := map[string]interface{}{
		"goldenPaths":  len(m.goldenPaths),
		"applications": len(m.applications),
	}

	// Count applications by golden path
	byGoldenPath := make(map[string]int)
	for _, app := range m.applications {
		byGoldenPath[app.Spec.GoldenPath]++
	}
	stats["applicationsByGoldenPath"] = byGoldenPath

	// Count applications by team
	byTeam := make(map[string]int)
	for _, app := range m.applications {
		if app.Metadata.Team != "" {
			byTeam[app.Metadata.Team]++
		}
	}
	stats["applicationsByTeam"] = byTeam

	return stats
}
