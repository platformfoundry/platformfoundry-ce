package copilot

import (
	"context"
	"fmt"
	"strings"
	"time"
)

// SeverityLevel represents the severity of an issue
type SeverityLevel string

const (
	SeverityInfo     SeverityLevel = "info"
	SeverityWarning  SeverityLevel = "warning"
	SeverityError    SeverityLevel = "error"
	SeverityCritical SeverityLevel = "critical"
)

// DiagnosisResult represents the result of a troubleshooting diagnosis
type DiagnosisResult struct {
	ProbableRootCause string          `json:"probableRootCause"`
	Confidence        float64         `json:"confidence"`
	Evidence          []Evidence      `json:"evidence"`
	SuggestedFixes    []SuggestedFix  `json:"suggestedFixes"`
	RelatedIncidents  []Incident      `json:"relatedIncidents"`
	Timeline          []TimelineEvent `json:"timeline"`
	AffectedServices  []string        `json:"affectedServices"`
	Severity          SeverityLevel   `json:"severity"`
}

// Evidence represents evidence supporting a diagnosis
type Evidence struct {
	Type        string      `json:"type"` // log, metric, event, config
	Source      string      `json:"source"`
	Description string      `json:"description"`
	Timestamp   time.Time   `json:"timestamp"`
	Data        interface{} `json:"data,omitempty"`
	Relevance   float64     `json:"relevance"` // 0-1 how relevant this evidence is
}

// SuggestedFix represents a suggested fix for an issue
type SuggestedFix struct {
	Description   string        `json:"description"`
	Command       string        `json:"command,omitempty"`
	ActionPlan    *ActionPlan   `json:"actionPlan,omitempty"`
	Confidence    float64       `json:"confidence"`
	RiskLevel     RiskLevel     `json:"riskLevel"`
	EstimatedTime time.Duration `json:"estimatedTime"`
}

// Incident represents a related incident
type Incident struct {
	ID          string        `json:"id"`
	Title       string        `json:"title"`
	Description string        `json:"description"`
	Severity    SeverityLevel `json:"severity"`
	ResolvedAt  *time.Time    `json:"resolvedAt,omitempty"`
	Resolution  string        `json:"resolution,omitempty"`
	Similarity  float64       `json:"similarity"` // 0-1 how similar to current issue
}

// TimelineEvent represents an event in the troubleshooting timeline
type TimelineEvent struct {
	Timestamp   time.Time     `json:"timestamp"`
	Type        string        `json:"type"`
	Description string        `json:"description"`
	Source      string        `json:"source"`
	Severity    SeverityLevel `json:"severity"`
}

// HealthChecker interface for health checking
type HealthChecker interface {
	CheckAll(ctx context.Context) map[string]HealthStatus
	CheckService(ctx context.Context, service string) HealthStatus
}

// HealthStatus represents the health status of a component
type HealthStatus struct {
	Status    string             `json:"status"` // healthy, degraded, unhealthy
	Message   string             `json:"message,omitempty"`
	LastCheck time.Time          `json:"lastCheck"`
	Metrics   map[string]float64 `json:"metrics,omitempty"`
}

// EventStore interface for event storage
type EventStore interface {
	GetRecent(ctx context.Context, duration time.Duration) []Event
	GetByService(ctx context.Context, service string, duration time.Duration) []Event
	Search(ctx context.Context, query string, duration time.Duration) []Event
}

// MetricsClient interface for metrics queries
type MetricsClient interface {
	Query(ctx context.Context, query string, duration time.Duration) ([]MetricResult, error)
	GetAnomalies(ctx context.Context, service string, duration time.Duration) ([]Anomaly, error)
}

// MetricResult represents a metric query result
type MetricResult struct {
	Name      string            `json:"name"`
	Value     float64           `json:"value"`
	Timestamp time.Time         `json:"timestamp"`
	Labels    map[string]string `json:"labels"`
}

// Anomaly represents a detected anomaly
type Anomaly struct {
	Metric    string        `json:"metric"`
	Expected  float64       `json:"expected"`
	Actual    float64       `json:"actual"`
	Deviation float64       `json:"deviation"`
	Timestamp time.Time     `json:"timestamp"`
	Severity  SeverityLevel `json:"severity"`
}

// KnownIssue represents a known issue pattern
type KnownIssue struct {
	ID         string        `json:"id"`
	Pattern    string        `json:"pattern"`
	Keywords   []string      `json:"keywords"`
	RootCause  string        `json:"rootCause"`
	Resolution string        `json:"resolution"`
	Commands   []string      `json:"commands,omitempty"`
	Severity   SeverityLevel `json:"severity"`
}

// KnowledgeBase interface for known issues
type KnowledgeBase interface {
	FindSimilar(symptom string, correlations []Correlation) *KnownIssue
	GetByKeywords(keywords []string) []KnownIssue
}

// Correlation represents a correlation between events
type Correlation struct {
	EventA      Event   `json:"eventA"`
	EventB      Event   `json:"eventB"`
	Correlation float64 `json:"correlation"` // -1 to 1
}

// TroubleshootEngine provides intelligent troubleshooting
type TroubleshootEngine struct {
	healthChecker HealthChecker
	eventStore    EventStore
	metricsClient MetricsClient
	knowledgeBase KnowledgeBase
}

// NewTroubleshootEngine creates a new troubleshoot engine
func NewTroubleshootEngine(health HealthChecker, events EventStore, metrics MetricsClient, kb KnowledgeBase) *TroubleshootEngine {
	return &TroubleshootEngine{
		healthChecker: health,
		eventStore:    events,
		metricsClient: metrics,
		knowledgeBase: kb,
	}
}

// Diagnose analyzes a symptom and produces a diagnosis
func (e *TroubleshootEngine) Diagnose(ctx context.Context, symptom string) (*DiagnosisResult, error) {
	result := &DiagnosisResult{
		Evidence:         make([]Evidence, 0),
		SuggestedFixes:   make([]SuggestedFix, 0),
		RelatedIncidents: make([]Incident, 0),
		Timeline:         make([]TimelineEvent, 0),
		AffectedServices: make([]string, 0),
		Confidence:       0.0,
	}

	// Step 1: Gather recent events
	events := e.gatherEvents(ctx, symptom)
	for _, event := range events {
		result.Evidence = append(result.Evidence, Evidence{
			Type:        "event",
			Source:      event.Source,
			Description: event.Message,
			Timestamp:   event.Timestamp,
			Relevance:   e.calculateRelevance(symptom, event.Message),
		})

		result.Timeline = append(result.Timeline, TimelineEvent{
			Timestamp:   event.Timestamp,
			Type:        event.Type,
			Description: event.Message,
			Source:      event.Source,
			Severity:    SeverityLevel(event.Severity),
		})
	}

	// Step 2: Check component health
	if e.healthChecker != nil {
		healthStatus := e.healthChecker.CheckAll(ctx)
		for service, status := range healthStatus {
			if status.Status != "healthy" {
				result.AffectedServices = append(result.AffectedServices, service)
				result.Evidence = append(result.Evidence, Evidence{
					Type:        "health",
					Source:      service,
					Description: fmt.Sprintf("%s is %s: %s", service, status.Status, status.Message),
					Timestamp:   status.LastCheck,
					Relevance:   0.8,
				})
			}
		}
	}

	// Step 3: Check for anomalies
	if e.metricsClient != nil {
		anomalies := e.detectAnomalies(ctx, result.AffectedServices)
		for _, anomaly := range anomalies {
			result.Evidence = append(result.Evidence, Evidence{
				Type:        "anomaly",
				Source:      "metrics",
				Description: fmt.Sprintf("Anomaly in %s: expected %.2f, got %.2f (%.1f%% deviation)", anomaly.Metric, anomaly.Expected, anomaly.Actual, anomaly.Deviation*100),
				Timestamp:   anomaly.Timestamp,
				Data:        anomaly,
				Relevance:   0.9,
			})
		}
	}

	// Step 4: Correlate events
	correlations := e.correlateEvents(events)

	// Step 5: Check knowledge base for known issues
	if e.knowledgeBase != nil {
		knownIssue := e.knowledgeBase.FindSimilar(symptom, correlations)
		if knownIssue != nil {
			result.ProbableRootCause = knownIssue.RootCause
			result.Confidence = 0.85
			result.Severity = knownIssue.Severity
			result.SuggestedFixes = append(result.SuggestedFixes, SuggestedFix{
				Description:   knownIssue.Resolution,
				Confidence:    0.85,
				RiskLevel:     RiskLow,
				EstimatedTime: 5 * time.Minute,
			})
			return result, nil
		}
	}

	// Step 6: Generate diagnosis based on evidence
	result.ProbableRootCause, result.Confidence = e.inferRootCause(symptom, result.Evidence, correlations)
	result.Severity = e.determineSeverity(result)
	result.SuggestedFixes = e.generateFixes(result)

	return result, nil
}

// gatherEvents gathers relevant events
func (e *TroubleshootEngine) gatherEvents(ctx context.Context, symptom string) []Event {
	if e.eventStore == nil {
		return e.generateMockEvents(symptom)
	}

	// Get recent events
	events := e.eventStore.GetRecent(ctx, 1*time.Hour)

	// Also search for related events
	searchEvents := e.eventStore.Search(ctx, symptom, 24*time.Hour)
	events = append(events, searchEvents...)

	return events
}

// generateMockEvents generates mock events for demonstration
func (e *TroubleshootEngine) generateMockEvents(symptom string) []Event {
	now := time.Now()
	events := []Event{
		{
			Type:      "error",
			Source:    "api-gateway",
			Message:   "Connection timeout to backend service",
			Timestamp: now.Add(-10 * time.Minute),
			Severity:  "error",
		},
		{
			Type:      "warning",
			Source:    "database",
			Message:   "High connection pool usage (85%)",
			Timestamp: now.Add(-15 * time.Minute),
			Severity:  "warning",
		},
		{
			Type:      "info",
			Source:    "deployment",
			Message:   "Deployment completed for worker-service v2.3.1",
			Timestamp: now.Add(-30 * time.Minute),
			Severity:  "info",
		},
	}

	return events
}

// detectAnomalies detects metric anomalies
func (e *TroubleshootEngine) detectAnomalies(ctx context.Context, services []string) []Anomaly {
	var anomalies []Anomaly

	if e.metricsClient != nil {
		for _, service := range services {
			detected, _ := e.metricsClient.GetAnomalies(ctx, service, 1*time.Hour)
			anomalies = append(anomalies, detected...)
		}
	} else {
		// Generate mock anomalies for demonstration
		anomalies = append(anomalies, Anomaly{
			Metric:    "response_time_p99",
			Expected:  45.0,
			Actual:    350.0,
			Deviation: 6.78,
			Timestamp: time.Now().Add(-5 * time.Minute),
			Severity:  SeverityError,
		})
	}

	return anomalies
}

// correlateEvents finds correlations between events
func (e *TroubleshootEngine) correlateEvents(events []Event) []Correlation {
	var correlations []Correlation

	// Simple time-based correlation
	for i := 0; i < len(events); i++ {
		for j := i + 1; j < len(events); j++ {
			timeDiff := events[j].Timestamp.Sub(events[i].Timestamp)
			if timeDiff < 0 {
				timeDiff = -timeDiff
			}

			// Events within 5 minutes are likely correlated
			if timeDiff < 5*time.Minute {
				correlation := 1.0 - (float64(timeDiff) / float64(5*time.Minute))
				correlations = append(correlations, Correlation{
					EventA:      events[i],
					EventB:      events[j],
					Correlation: correlation,
				})
			}
		}
	}

	return correlations
}

// calculateRelevance calculates the relevance of evidence to the symptom
func (e *TroubleshootEngine) calculateRelevance(symptom, evidence string) float64 {
	symptomLower := strings.ToLower(symptom)
	evidenceLower := strings.ToLower(evidence)

	// Simple keyword matching
	words := strings.Fields(symptomLower)
	matches := 0
	for _, word := range words {
		if len(word) > 3 && strings.Contains(evidenceLower, word) {
			matches++
		}
	}

	if len(words) == 0 {
		return 0.5
	}

	return float64(matches) / float64(len(words))
}

// inferRootCause infers the root cause from evidence
func (e *TroubleshootEngine) inferRootCause(symptom string, evidence []Evidence, correlations []Correlation) (string, float64) {
	// Sort evidence by relevance
	var mostRelevant *Evidence
	for i := range evidence {
		if mostRelevant == nil || evidence[i].Relevance > mostRelevant.Relevance {
			mostRelevant = &evidence[i]
		}
	}

	// Check for common patterns
	symptomLower := strings.ToLower(symptom)

	if containsAny(symptomLower, []string{"slow", "timeout", "latency"}) {
		return "Performance degradation likely caused by resource contention or increased load", 0.7
	}

	if containsAny(symptomLower, []string{"error", "fail", "crash"}) {
		return "Service failure detected, possibly due to a recent deployment or configuration change", 0.65
	}

	if containsAny(symptomLower, []string{"connection", "network", "unreachable"}) {
		return "Network connectivity issue between services", 0.6
	}

	if mostRelevant != nil && mostRelevant.Relevance > 0.5 {
		return fmt.Sprintf("Issue likely related to: %s", mostRelevant.Description), mostRelevant.Relevance
	}

	return "Unable to determine root cause with high confidence. Manual investigation recommended.", 0.3
}

// determineSeverity determines the severity of the issue
func (e *TroubleshootEngine) determineSeverity(result *DiagnosisResult) SeverityLevel {
	// Check affected services count
	if len(result.AffectedServices) > 3 {
		return SeverityCritical
	}

	// Check evidence severity
	for _, ev := range result.Evidence {
		if ev.Type == "anomaly" {
			return SeverityError
		}
	}

	if len(result.AffectedServices) > 0 {
		return SeverityWarning
	}

	return SeverityInfo
}

// generateFixes generates suggested fixes
func (e *TroubleshootEngine) generateFixes(result *DiagnosisResult) []SuggestedFix {
	fixes := []SuggestedFix{}

	// Generic fixes based on symptom analysis
	if len(result.AffectedServices) > 0 {
		fixes = append(fixes, SuggestedFix{
			Description:   fmt.Sprintf("Restart affected services: %s", strings.Join(result.AffectedServices, ", ")),
			Command:       fmt.Sprintf("pf workload restart %s", result.AffectedServices[0]),
			Confidence:    0.6,
			RiskLevel:     RiskMedium,
			EstimatedTime: 2 * time.Minute,
		})
	}

	// Check for deployment-related issues
	for _, ev := range result.Timeline {
		if ev.Type == "deployment" || strings.Contains(ev.Description, "deploy") {
			fixes = append(fixes, SuggestedFix{
				Description:   "Rollback to previous version",
				Command:       "pf rollback --to-previous",
				Confidence:    0.7,
				RiskLevel:     RiskHigh,
				EstimatedTime: 3 * time.Minute,
			})
			break
		}
	}

	// Generic investigative steps
	fixes = append(fixes, SuggestedFix{
		Description:   "Check service logs for detailed errors",
		Command:       "pf logs --service all --since 1h --level error",
		Confidence:    0.8,
		RiskLevel:     RiskLow,
		EstimatedTime: 30 * time.Second,
	})

	return fixes
}

// QuickDiagnose performs a quick diagnosis without full analysis
func (e *TroubleshootEngine) QuickDiagnose(ctx context.Context, service string) (*DiagnosisResult, error) {
	result := &DiagnosisResult{
		AffectedServices: []string{service},
		Evidence:         make([]Evidence, 0),
		SuggestedFixes:   make([]SuggestedFix, 0),
	}

	// Quick health check
	if e.healthChecker != nil {
		status := e.healthChecker.CheckService(ctx, service)
		if status.Status != "healthy" {
			result.ProbableRootCause = fmt.Sprintf("Service %s is %s: %s", service, status.Status, status.Message)
			result.Confidence = 0.8
			result.Severity = SeverityWarning
			if status.Status == "unhealthy" {
				result.Severity = SeverityError
			}
		} else {
			result.ProbableRootCause = fmt.Sprintf("Service %s appears healthy", service)
			result.Confidence = 0.9
			result.Severity = SeverityInfo
		}
	}

	return result, nil
}
