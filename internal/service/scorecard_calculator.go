package service

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/platformfoundry/platformfoundry-ce/internal/state"
	"github.com/platformfoundry/platformfoundry-ce/pkg/types"
)

// ScorecardCalculator computes and stores service scorecards
type ScorecardCalculator struct {
	backend state.Backend
	engine  *ScorecardEngine
}

// NewScorecardCalculator creates a new scorecard calculator
func NewScorecardCalculator(backend state.Backend) *ScorecardCalculator {
	return &ScorecardCalculator{
		backend: backend,
		engine:  NewScorecardEngine(),
	}
}

// Calculate computes a scorecard for a service
func (sc *ScorecardCalculator) Calculate(serviceName, organization string, context *CheckContext) (*types.ServiceScorecard, error) {
	// Get the service
	resourceName := sc.buildResourceName(serviceName, organization)
	resource, err := sc.backend.Get(resourceName)
	if err != nil {
		return nil, fmt.Errorf("failed to get service: %w", err)
	}

	// Convert resource to service
	service, err := sc.resourceToService(resource)
	if err != nil {
		return nil, fmt.Errorf("failed to parse service: %w", err)
	}

	// If no context provided, create a default one
	if context == nil {
		context = sc.buildDefaultContext(service)
	}

	// Evaluate using the engine
	scorecard, err := sc.engine.Evaluate(service, context)
	if err != nil {
		return nil, fmt.Errorf("failed to evaluate scorecard: %w", err)
	}

	// Store the scorecard
	if err := sc.Save(scorecard); err != nil {
		return nil, fmt.Errorf("failed to save scorecard: %w", err)
	}

	return scorecard, nil
}

// Get retrieves a stored scorecard
func (sc *ScorecardCalculator) Get(serviceName, organization string) (*types.ServiceScorecard, error) {
	resourceName := sc.buildResourceName(serviceName, organization)
	resource, err := sc.backend.Get(resourceName)
	if err != nil {
		return nil, fmt.Errorf("scorecard not found: %w", err)
	}

	// Only return if it's a scorecard resource
	if resource.Kind != "ServiceScorecard" {
		return nil, fmt.Errorf("resource is not a scorecard")
	}

	return sc.resourceToScorecard(resource, organization)
}

// Save stores a scorecard
func (sc *ScorecardCalculator) Save(scorecard *types.ServiceScorecard) error {
	resource, err := sc.scorecardToResource(scorecard)
	if err != nil {
		return fmt.Errorf("failed to convert scorecard: %w", err)
	}

	if err := sc.backend.Save(resource); err != nil {
		return fmt.Errorf("failed to save scorecard: %w", err)
	}

	return nil
}

// List retrieves all scorecards for an organization
func (sc *ScorecardCalculator) List(organization string) ([]*types.ServiceScorecard, error) {
	resources, err := sc.backend.List()
	if err != nil {
		return nil, fmt.Errorf("failed to list resources: %w", err)
	}

	scorecards := make([]*types.ServiceScorecard, 0)
	for _, resource := range resources {
		// Only process scorecard resources
		if resource.Kind != "ServiceScorecard" {
			continue
		}

		// Check if it belongs to the organization
		if org, ok := resource.Spec["organization"].(string); !ok || org != organization {
			continue
		}

		scorecard, err := sc.resourceToScorecard(resource, organization)
		if err != nil {
			continue
		}

		scorecards = append(scorecards, scorecard)
	}

	return scorecards, nil
}

// ListByGrade retrieves scorecards filtered by grade
func (sc *ScorecardCalculator) ListByGrade(grade types.ScorecardGrade, organization string) ([]*types.ServiceScorecard, error) {
	scorecards, err := sc.List(organization)
	if err != nil {
		return nil, err
	}

	filtered := make([]*types.ServiceScorecard, 0)
	for _, scorecard := range scorecards {
		if scorecard.Status.Grade == grade {
			filtered = append(filtered, scorecard)
		}
	}

	return filtered, nil
}

// RecalculateAll recalculates scorecards for all services in an organization
func (sc *ScorecardCalculator) RecalculateAll(organization string) error {
	// Get all resources
	resources, err := sc.backend.List()
	if err != nil {
		return fmt.Errorf("failed to list resources: %w", err)
	}

	for _, resource := range resources {
		// Only process service resources
		if resource.Kind != "Service" {
			continue
		}

		// Check if it belongs to the organization
		if org, ok := resource.Spec["organization"].(string); !ok || org != organization {
			continue
		}

		service, err := sc.resourceToService(resource)
		if err != nil {
			continue
		}

		// Calculate scorecard with default context
		context := sc.buildDefaultContext(service)
		scorecard, err := sc.engine.Evaluate(service, context)
		if err != nil {
			// Log error but continue
			continue
		}

		// Save the scorecard
		_ = sc.Save(scorecard)
	}

	return nil
}

// GetStats returns statistics about scorecards
func (sc *ScorecardCalculator) GetStats(organization string) (*ScorecardStats, error) {
	scorecards, err := sc.List(organization)
	if err != nil {
		return nil, err
	}

	stats := &ScorecardStats{
		Total:      len(scorecards),
		ByGrade:    make(map[types.ScorecardGrade]int),
		ByCategory: make(map[types.CheckCategory]CategoryStats),
	}

	totalScore := 0
	for _, scorecard := range scorecards {
		totalScore += scorecard.Status.Score
		stats.ByGrade[scorecard.Status.Grade]++

		// Aggregate category stats
		for _, check := range scorecard.Spec.Checks {
			catStats := stats.ByCategory[check.Category]
			catStats.Total++
			if check.Status == types.CheckStatusPassed {
				catStats.Passed++
			} else if check.Status == types.CheckStatusFailed {
				catStats.Failed++
			} else if check.Status == types.CheckStatusWarning {
				catStats.Warning++
			}
			stats.ByCategory[check.Category] = catStats
		}
	}

	if len(scorecards) > 0 {
		stats.AverageScore = float64(totalScore) / float64(len(scorecards))
	}

	return stats, nil
}

// buildDefaultContext creates a basic check context from service metadata
func (sc *ScorecardCalculator) buildDefaultContext(service *types.Service) *CheckContext {
	context := &CheckContext{
		DependencyCount: len(service.Spec.Dependencies),
	}

	// Infer some values from service spec
	if service.Spec.Repository != nil {
		context.HasReadme = true
		context.ReadmeLength = 500 // Assume reasonable README
	}

	// Check for observability config
	if len(service.Spec.Links) > 0 {
		for _, link := range service.Spec.Links {
			if link.Type == "metrics" || link.Type == "dashboard" {
				context.HasMetrics = true
			}
			if link.Type == "logs" {
				context.HasLogs = true
			}
		}
	}

	// Basic deployment info from status
	if service.Status.LastDeployed != nil {
		context.LastDeployTime = service.Status.LastDeployed
		// Assume weekly deploys if recently updated
		daysSinceUpdate := time.Since(*service.Status.LastDeployed).Hours() / 24
		if daysSinceUpdate < 7 {
			context.DeployFrequency = 2.0
		}
	}

	return context
}

// buildResourceName builds a unique resource name
func (sc *ScorecardCalculator) buildResourceName(name, organization string) string {
	if organization != "" {
		return fmt.Sprintf("%s/%s", organization, name)
	}
	return name
}

// ScorecardStats represents aggregated scorecard statistics
type ScorecardStats struct {
	Total        int
	AverageScore float64
	ByGrade      map[types.ScorecardGrade]int
	ByCategory   map[types.CheckCategory]CategoryStats
}

// CategoryStats represents statistics for a check category
type CategoryStats struct {
	Total   int
	Passed  int
	Failed  int
	Warning int
}

// Helper functions

func (sc *ScorecardCalculator) scorecardToResource(scorecard *types.ServiceScorecard) (*state.Resource, error) {
	// Marshal spec
	specMap, err := structToMap(scorecard.Spec)
	if err != nil {
		return nil, err
	}

	// Marshal status
	statusMap, err := structToMap(scorecard.Status)
	if err != nil {
		return nil, err
	}

	// Add metadata to spec for filtering
	specMap["organization"] = scorecard.Metadata.Organization
	specMap["name"] = scorecard.Metadata.Name

	now := time.Now()
	return &state.Resource{
		Name:       sc.buildResourceName(scorecard.Metadata.Name, scorecard.Metadata.Organization),
		Kind:       scorecard.Kind,
		APIVersion: scorecard.APIVersion,
		Spec:       specMap,
		Status:     statusMap,
		CreatedAt:  now,
		UpdatedAt:  now,
	}, nil
}

func (sc *ScorecardCalculator) resourceToScorecard(resource *state.Resource, organization string) (*types.ServiceScorecard, error) {
	// Marshal and unmarshal to convert maps to structs
	specBytes, err := json.Marshal(resource.Spec)
	if err != nil {
		return nil, err
	}

	statusBytes, err := json.Marshal(resource.Status)
	if err != nil {
		return nil, err
	}

	var spec types.ServiceScorecardSpec
	if err := json.Unmarshal(specBytes, &spec); err != nil {
		return nil, err
	}

	var status types.ServiceScorecardStatus
	if err := json.Unmarshal(statusBytes, &status); err != nil {
		return nil, err
	}

	// Extract name from resource name (organization/name format)
	name := resource.Name
	if strings.Contains(name, "/") {
		parts := strings.Split(name, "/")
		name = parts[len(parts)-1]
	}

	return &types.ServiceScorecard{
		APIVersion: resource.APIVersion,
		Kind:       resource.Kind,
		Metadata: types.Metadata{
			Name:         name,
			Organization: organization,
			Labels:       make(map[string]string),
		},
		Spec:   spec,
		Status: status,
	}, nil
}

func (sc *ScorecardCalculator) resourceToService(resource *state.Resource) (*types.Service, error) {
	// Marshal and unmarshal to convert maps to structs
	specBytes, err := json.Marshal(resource.Spec)
	if err != nil {
		return nil, err
	}

	statusBytes, err := json.Marshal(resource.Status)
	if err != nil {
		return nil, err
	}

	var spec types.ServiceSpec
	if err := json.Unmarshal(specBytes, &spec); err != nil {
		return nil, err
	}

	var status types.ServiceStatus
	if err := json.Unmarshal(statusBytes, &status); err != nil {
		return nil, err
	}

	// Extract organization and name from resource
	org := ""
	if orgVal, ok := resource.Spec["organization"].(string); ok {
		org = orgVal
	}

	// Extract name from resource name (organization/name format)
	name := resource.Name
	if strings.Contains(name, "/") {
		parts := strings.Split(name, "/")
		name = parts[len(parts)-1]
	}

	return &types.Service{
		APIVersion: resource.APIVersion,
		Kind:       resource.Kind,
		Metadata: types.Metadata{
			Name:         name,
			Organization: org,
			Labels:       make(map[string]string),
		},
		Spec:   spec,
		Status: status,
	}, nil
}
