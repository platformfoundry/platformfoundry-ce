package devex

import (
	"context"
	"fmt"
	"sort"
	"sync"
	"time"
)

// Manager handles developer experience analytics
type Manager struct {
	teams          map[string]*DeveloperMetrics
	doraMetrics    map[string]*DORAMetrics
	adoption       *PlatformAdoption
	journeys       map[string]*DeveloperJourney
	surveys        []Survey
	frictionPoints []FrictionPoint
	mu             sync.RWMutex
}

// NewManager creates a new DevEx manager
func NewManager() *Manager {
	m := &Manager{
		teams:          make(map[string]*DeveloperMetrics),
		doraMetrics:    make(map[string]*DORAMetrics),
		journeys:       make(map[string]*DeveloperJourney),
		surveys:        make([]Survey, 0),
		frictionPoints: make([]FrictionPoint, 0),
	}

	// Initialize with sample data for demonstration
	m.initializeSampleData()

	return m
}

// initializeSampleData creates sample metrics data
func (m *Manager) initializeSampleData() {
	// Sample DORA metrics
	m.doraMetrics["platform-team"] = &DORAMetrics{
		DeploymentFrequency: &DeploymentFrequency{
			Value:            4.5,
			Period:           "daily",
			TotalDeployments: 135,
			ByEnvironment: map[string]int{
				"dev":        60,
				"staging":    45,
				"production": 30,
			},
			Rating: "elite",
		},
		LeadTime: &LeadTime{
			Value:  2 * time.Hour,
			P50:    1 * time.Hour,
			P90:    4 * time.Hour,
			P95:    8 * time.Hour,
			Rating: "elite",
			Breakdown: &LeadTimeBreakdown{
				CodeReview: 30 * time.Minute,
				Build:      5 * time.Minute,
				Test:       15 * time.Minute,
				Staging:    30 * time.Minute,
				Approval:   20 * time.Minute,
				Production: 20 * time.Minute,
			},
		},
		ChangeFailureRate: &ChangeFailureRate{
			Value:         8.5,
			TotalChanges:  135,
			FailedChanges: 11,
			Rating:        "elite",
			TopFailures: []FailureCategory{
				{Category: "configuration", Count: 5, Percentage: 45.5},
				{Category: "integration", Count: 3, Percentage: 27.3},
				{Category: "resource_limit", Count: 2, Percentage: 18.2},
				{Category: "other", Count: 1, Percentage: 9.1},
			},
		},
		TimeToRestore: &TimeToRestore{
			Value:     30 * time.Minute,
			P50:       15 * time.Minute,
			P90:       1 * time.Hour,
			Rating:    "elite",
			Incidents: 8,
		},
		Rating: "elite",
	}

	m.doraMetrics["backend-team"] = &DORAMetrics{
		DeploymentFrequency: &DeploymentFrequency{
			Value:            2.1,
			Period:           "daily",
			TotalDeployments: 63,
			Rating:           "high",
		},
		LeadTime: &LeadTime{
			Value:  6 * time.Hour,
			P50:    4 * time.Hour,
			P90:    12 * time.Hour,
			Rating: "high",
		},
		ChangeFailureRate: &ChangeFailureRate{
			Value:         12.0,
			TotalChanges:  63,
			FailedChanges: 8,
			Rating:        "elite",
		},
		TimeToRestore: &TimeToRestore{
			Value:     2 * time.Hour,
			P50:       1 * time.Hour,
			P90:       4 * time.Hour,
			Rating:    "high",
			Incidents: 5,
		},
		Rating: "high",
	}

	// Sample platform adoption
	m.adoption = &PlatformAdoption{
		SelfServiceRatio:     85.0,
		GoldenPathAdoption:   72.0,
		AutomatedDeployments: 95.0,
		ActiveUsers:          45,
		TotalApplications:    28,
		FeatureUsage: map[string]int{
			"gitops":        25,
			"policies":      20,
			"secrets":       28,
			"workflows":     15,
			"observability": 22,
		},
	}

	// Sample friction points
	m.frictionPoints = []FrictionPoint{
		{
			ID:          "fp-1",
			Category:    "build",
			Description: "Long build times (>5 min) affecting 40% of builds",
			Impact:      "high",
			Occurrences: 156,
			Suggestion:  "Enable build caching and parallel test execution",
			DetectedAt:  time.Now().Add(-7 * 24 * time.Hour),
		},
		{
			ID:          "fp-2",
			Category:    "deploy",
			Description: "Manual approval delays averaging 2 hours",
			Impact:      "medium",
			Occurrences: 45,
			Suggestion:  "Implement auto-approval for non-production environments",
			DetectedAt:  time.Now().Add(-5 * 24 * time.Hour),
		},
		{
			ID:          "fp-3",
			Category:    "debug",
			Description: "Limited log retention making debugging difficult",
			Impact:      "medium",
			Occurrences: 23,
			Suggestion:  "Extend log retention to 30 days for production",
			DetectedAt:  time.Now().Add(-3 * 24 * time.Hour),
		},
	}
}

// RegisterTeamMetrics registers metrics configuration for a team
func (m *Manager) RegisterTeamMetrics(ctx context.Context, metrics *DeveloperMetrics) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if metrics.Metadata.Name == "" {
		return fmt.Errorf("metrics name is required")
	}

	if metrics.APIVersion == "" {
		metrics.APIVersion = "platformfoundry.io/v1"
	}
	if metrics.Kind == "" {
		metrics.Kind = "DeveloperMetrics"
	}

	metrics.Metadata.CreatedAt = time.Now()

	// Initialize status
	metrics.Status = &DeveloperMetricsStatus{
		LastUpdated:    time.Now(),
		MetricValues:   make([]MetricValue, 0),
		FrictionPoints: make([]FrictionPoint, 0),
	}

	m.teams[metrics.Metadata.Name] = metrics
	return nil
}

// GetTeamMetrics returns metrics for a team
func (m *Manager) GetTeamMetrics(team string) (*DeveloperMetrics, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	metrics, ok := m.teams[team]
	if !ok {
		return nil, fmt.Errorf("metrics not found for team: %s", team)
	}
	return metrics, nil
}

// GetDORAMetrics returns DORA metrics for a team
func (m *Manager) GetDORAMetrics(team string) (*DORAMetrics, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	dora, ok := m.doraMetrics[team]
	if !ok {
		// Return default/organization-wide metrics
		return m.calculateOrgDORA(), nil
	}
	return dora, nil
}

// calculateOrgDORA calculates organization-wide DORA metrics
func (m *Manager) calculateOrgDORA() *DORAMetrics {
	// Aggregate all team metrics
	totalDeployments := 0
	totalFailures := 0
	var leadTimes []time.Duration
	var restoreTimes []time.Duration

	for _, dora := range m.doraMetrics {
		if dora.DeploymentFrequency != nil {
			totalDeployments += dora.DeploymentFrequency.TotalDeployments
		}
		if dora.ChangeFailureRate != nil {
			totalFailures += dora.ChangeFailureRate.FailedChanges
		}
		if dora.LeadTime != nil {
			leadTimes = append(leadTimes, dora.LeadTime.Value)
		}
		if dora.TimeToRestore != nil {
			restoreTimes = append(restoreTimes, dora.TimeToRestore.Value)
		}
	}

	// Calculate averages
	avgLeadTime := time.Duration(0)
	if len(leadTimes) > 0 {
		total := time.Duration(0)
		for _, lt := range leadTimes {
			total += lt
		}
		avgLeadTime = total / time.Duration(len(leadTimes))
	}

	avgRestoreTime := time.Duration(0)
	if len(restoreTimes) > 0 {
		total := time.Duration(0)
		for _, rt := range restoreTimes {
			total += rt
		}
		avgRestoreTime = total / time.Duration(len(restoreTimes))
	}

	failureRate := 0.0
	if totalDeployments > 0 {
		failureRate = float64(totalFailures) / float64(totalDeployments) * 100
	}

	return &DORAMetrics{
		DeploymentFrequency: &DeploymentFrequency{
			Value:            float64(totalDeployments) / 30, // Per day over 30 days
			TotalDeployments: totalDeployments,
			Rating:           rateDeplFreq(float64(totalDeployments) / 30),
		},
		LeadTime: &LeadTime{
			Value:  avgLeadTime,
			Rating: rateLeadTime(avgLeadTime),
		},
		ChangeFailureRate: &ChangeFailureRate{
			Value:         failureRate,
			TotalChanges:  totalDeployments,
			FailedChanges: totalFailures,
			Rating:        rateCFR(failureRate),
		},
		TimeToRestore: &TimeToRestore{
			Value:  avgRestoreTime,
			Rating: rateMTTR(avgRestoreTime),
		},
		Rating: "high", // Simplified
	}
}

// GetPlatformAdoption returns platform adoption metrics
func (m *Manager) GetPlatformAdoption() *PlatformAdoption {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.adoption
}

// GetFrictionPoints returns identified friction points
func (m *Manager) GetFrictionPoints() []FrictionPoint {
	m.mu.RLock()
	defer m.mu.RUnlock()

	// Sort by impact (high first) then by occurrences
	sorted := make([]FrictionPoint, len(m.frictionPoints))
	copy(sorted, m.frictionPoints)

	sort.Slice(sorted, func(i, j int) bool {
		impactOrder := map[string]int{"high": 0, "medium": 1, "low": 2}
		if impactOrder[sorted[i].Impact] != impactOrder[sorted[j].Impact] {
			return impactOrder[sorted[i].Impact] < impactOrder[sorted[j].Impact]
		}
		return sorted[i].Occurrences > sorted[j].Occurrences
	})

	return sorted
}

// RecordFrictionPoint records a new friction point
func (m *Manager) RecordFrictionPoint(fp FrictionPoint) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if fp.ID == "" {
		fp.ID = fmt.Sprintf("fp-%d", time.Now().UnixNano())
	}
	fp.DetectedAt = time.Now()

	m.frictionPoints = append(m.frictionPoints, fp)
}

// GetDeveloperJourney returns journey metrics for a developer
func (m *Manager) GetDeveloperJourney(developerID string) (*DeveloperJourney, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	journey, ok := m.journeys[developerID]
	if !ok {
		return nil, fmt.Errorf("journey not found for developer: %s", developerID)
	}
	return journey, nil
}

// RecordJourneyMilestone records a developer journey milestone
func (m *Manager) RecordJourneyMilestone(developerID string, milestone JourneyMilestone) {
	m.mu.Lock()
	defer m.mu.Unlock()

	journey, ok := m.journeys[developerID]
	if !ok {
		journey = &DeveloperJourney{
			Milestones: make([]JourneyMilestone, 0),
		}
		m.journeys[developerID] = journey
	}

	now := time.Now()
	milestone.CompletedAt = &now
	journey.Milestones = append(journey.Milestones, milestone)
}

// GenerateReport generates a developer analytics report
func (m *Manager) GenerateReport(ctx context.Context, team string) (*AnalyticsReport, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	report := &AnalyticsReport{
		GeneratedAt:     time.Now(),
		Period:          "30d",
		Team:            team,
		FrictionPoints:  m.frictionPoints,
		Recommendations: m.generateRecommendations(),
	}

	// Get DORA metrics
	if team != "" {
		if dora, ok := m.doraMetrics[team]; ok {
			report.DORA = dora
		}
	} else {
		report.DORA = m.calculateOrgDORA()
	}

	report.Adoption = m.adoption

	// Calculate score
	report.Score = m.calculateScore(report.DORA, report.Adoption)

	return report, nil
}

// calculateScore calculates the developer experience score
func (m *Manager) calculateScore(dora *DORAMetrics, adoption *PlatformAdoption) *DeveloperScore {
	score := &DeveloperScore{
		Categories: make(map[string]float64),
	}

	// DORA score (40% weight)
	doraScore := 0.0
	doraCount := 0
	if dora != nil {
		if dora.DeploymentFrequency != nil {
			doraScore += ratingToScore(dora.DeploymentFrequency.Rating)
			doraCount++
		}
		if dora.LeadTime != nil {
			doraScore += ratingToScore(dora.LeadTime.Rating)
			doraCount++
		}
		if dora.ChangeFailureRate != nil {
			doraScore += ratingToScore(dora.ChangeFailureRate.Rating)
			doraCount++
		}
		if dora.TimeToRestore != nil {
			doraScore += ratingToScore(dora.TimeToRestore.Rating)
			doraCount++
		}
	}
	if doraCount > 0 {
		score.Categories["dora"] = doraScore / float64(doraCount)
	}

	// Adoption score (30% weight)
	if adoption != nil {
		adoptionScore := (adoption.SelfServiceRatio + adoption.GoldenPathAdoption + adoption.AutomatedDeployments) / 3
		score.Categories["adoption"] = adoptionScore / 100 * 5 // Scale to 5
	}

	// Self-service score (30% weight)
	selfServiceScore := 4.0 // Default good score
	if adoption != nil && adoption.SelfServiceRatio > 0 {
		selfServiceScore = adoption.SelfServiceRatio / 100 * 5
	}
	score.Categories["self-service"] = selfServiceScore

	// Calculate overall
	totalWeight := 0.0
	weightedSum := 0.0
	weights := map[string]float64{"dora": 0.4, "adoption": 0.3, "self-service": 0.3}

	for cat, catScore := range score.Categories {
		if weight, ok := weights[cat]; ok {
			weightedSum += catScore * weight
			totalWeight += weight
		}
	}

	if totalWeight > 0 {
		score.Overall = weightedSum / totalWeight
	}

	return score
}

// generateRecommendations generates platform improvement recommendations
func (m *Manager) generateRecommendations() []Recommendation {
	recommendations := []Recommendation{}
	priority := 1

	// Check friction points
	for _, fp := range m.frictionPoints {
		if fp.Impact == "high" {
			recommendations = append(recommendations, Recommendation{
				ID:          fmt.Sprintf("rec-%d", priority),
				Category:    fp.Category,
				Title:       fmt.Sprintf("Address %s friction", fp.Category),
				Description: fp.Suggestion,
				Impact:      "high",
				Effort:      "medium",
				Priority:    priority,
			})
			priority++
		}
	}

	// Check adoption rates
	if m.adoption != nil {
		if m.adoption.GoldenPathAdoption < 80 {
			recommendations = append(recommendations, Recommendation{
				ID:          fmt.Sprintf("rec-%d", priority),
				Category:    "adoption",
				Title:       "Increase golden path adoption",
				Description: "Promote golden paths through documentation and training to improve standardization",
				Impact:      "high",
				Effort:      "low",
				Priority:    priority,
			})
			priority++
		}

		if m.adoption.SelfServiceRatio < 90 {
			recommendations = append(recommendations, Recommendation{
				ID:          fmt.Sprintf("rec-%d", priority),
				Category:    "self-service",
				Title:       "Improve self-service capabilities",
				Description: "Identify manual processes and automate them to increase self-service ratio",
				Impact:      "medium",
				Effort:      "medium",
				Priority:    priority,
			})
			priority++
		}
	}

	return recommendations
}

// RecordSurvey records a developer survey
func (m *Manager) RecordSurvey(survey Survey) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if survey.ID == "" {
		survey.ID = fmt.Sprintf("survey-%d", time.Now().UnixNano())
	}
	survey.CollectedAt = time.Now()

	m.surveys = append(m.surveys, survey)
}

// GetSurveys returns recorded surveys
func (m *Manager) GetSurveys(surveyType string) []Survey {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if surveyType == "" {
		return m.surveys
	}

	filtered := make([]Survey, 0)
	for _, s := range m.surveys {
		if s.Type == surveyType {
			filtered = append(filtered, s)
		}
	}
	return filtered
}

// ListTeams returns all teams with metrics
func (m *Manager) ListTeams() []string {
	m.mu.RLock()
	defer m.mu.RUnlock()

	teams := make([]string, 0)
	for team := range m.doraMetrics {
		teams = append(teams, team)
	}
	return teams
}

// Helper functions

func rateDeplFreq(deploysPerDay float64) string {
	if deploysPerDay >= 1 {
		return "elite"
	} else if deploysPerDay >= 1.0/7 {
		return "high"
	} else if deploysPerDay >= 1.0/30 {
		return "medium"
	}
	return "low"
}

func rateLeadTime(lt time.Duration) string {
	if lt < time.Hour {
		return "elite"
	} else if lt < 24*time.Hour {
		return "high"
	} else if lt < 7*24*time.Hour {
		return "medium"
	}
	return "low"
}

func rateCFR(rate float64) string {
	if rate < 15 {
		return "elite"
	} else if rate < 30 {
		return "high"
	} else if rate < 45 {
		return "medium"
	}
	return "low"
}

func rateMTTR(mttr time.Duration) string {
	if mttr < time.Hour {
		return "elite"
	} else if mttr < 24*time.Hour {
		return "high"
	} else if mttr < 7*24*time.Hour {
		return "medium"
	}
	return "low"
}

func ratingToScore(rating string) float64 {
	switch rating {
	case "elite":
		return 5.0
	case "high":
		return 4.0
	case "medium":
		return 3.0
	case "low":
		return 2.0
	default:
		return 2.5
	}
}
