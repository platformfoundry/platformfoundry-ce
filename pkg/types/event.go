package types

import (
	"time"
)

// EventType represents the type of platform event
type EventType string

const (
	// Resource lifecycle events
	EventResourceCreated EventType = "resource.created"
	EventResourceUpdated EventType = "resource.updated"
	EventResourceDeleted EventType = "resource.deleted"
	EventResourceFailed  EventType = "resource.failed"

	// Drift events
	EventDriftDetected   EventType = "drift.detected"
	EventDriftResolved   EventType = "drift.resolved"
	EventDriftRemediated EventType = "drift.remediated"

	// Health events
	EventHealthDegraded  EventType = "health.degraded"
	EventHealthRecovered EventType = "health.recovered"
	EventHealthCritical  EventType = "health.critical"

	// Promise events
	EventPromiseRequested EventType = "promise.requested"
	EventPromiseApproved  EventType = "promise.approved"
	EventPromiseRejected  EventType = "promise.rejected"
	EventPromiseFulfilled EventType = "promise.fulfilled"
	EventPromiseFailed    EventType = "promise.failed"

	// Workload events
	EventWorkloadDeployed EventType = "workload.deployed"
	EventWorkloadScaled   EventType = "workload.scaled"
	EventWorkloadFailed   EventType = "workload.failed"

	// Chaos events
	EventChaosStarted   EventType = "chaos.started"
	EventChaosCompleted EventType = "chaos.completed"
	EventChaosFailed    EventType = "chaos.failed"

	// Compliance events
	EventComplianceViolation EventType = "compliance.violation"
	EventCompliancePassed    EventType = "compliance.passed"

	// Cost events
	EventCostAlert   EventType = "cost.alert"
	EventCostAnomaly EventType = "cost.anomaly"

	// GitOps events
	EventGitOpsSyncStarted   EventType = "gitops.sync.started"
	EventGitOpsSyncCompleted EventType = "gitops.sync.completed"
	EventGitOpsSyncFailed    EventType = "gitops.sync.failed"
	EventGitOpsPRCreated     EventType = "gitops.pr.created"
	EventGitOpsPRMerged      EventType = "gitops.pr.merged"

	// System events
	EventSystemStartup  EventType = "system.startup"
	EventSystemShutdown EventType = "system.shutdown"
	EventSystemError    EventType = "system.error"
)

// EventSeverity represents the severity level of an event
type EventSeverity string

const (
	EventSeverityInfo     EventSeverity = "info"
	EventSeverityWarning  EventSeverity = "warning"
	EventSeverityError    EventSeverity = "error"
	EventSeverityCritical EventSeverity = "critical"
)

// Event represents a platform event
type Event struct {
	// ID is a unique identifier for this event
	ID string `json:"id" yaml:"id"`

	// Type is the event type
	Type EventType `json:"type" yaml:"type"`

	// Source is the component that emitted the event
	Source string `json:"source" yaml:"source"`

	// Subject is the resource affected by this event
	Subject string `json:"subject" yaml:"subject"`

	// SubjectKind is the kind of resource (Platform, Service, Promise, etc.)
	SubjectKind string `json:"subject_kind" yaml:"subject_kind"`

	// Severity indicates the importance of the event
	Severity EventSeverity `json:"severity" yaml:"severity"`

	// Data contains event-specific payload
	Data map[string]interface{} `json:"data,omitempty" yaml:"data,omitempty"`

	// Timestamp is when the event occurred
	Timestamp time.Time `json:"timestamp" yaml:"timestamp"`

	// CorrelationID links related events together
	CorrelationID string `json:"correlation_id,omitempty" yaml:"correlation_id,omitempty"`

	// CausationID is the ID of the event that caused this one
	CausationID string `json:"causation_id,omitempty" yaml:"causation_id,omitempty"`

	// Actor is who/what triggered this event
	Actor string `json:"actor,omitempty" yaml:"actor,omitempty"`

	// Organization scope
	Organization string `json:"organization,omitempty" yaml:"organization,omitempty"`

	// Environment scope
	Environment string `json:"environment,omitempty" yaml:"environment,omitempty"`

	// Metadata for additional context
	Metadata map[string]string `json:"metadata,omitempty" yaml:"metadata,omitempty"`
}

// EventFilter defines criteria for filtering events
type EventFilter struct {
	// Types to include (empty means all)
	Types []EventType `json:"types,omitempty" yaml:"types,omitempty"`

	// Sources to include
	Sources []string `json:"sources,omitempty" yaml:"sources,omitempty"`

	// SubjectKinds to include
	SubjectKinds []string `json:"subject_kinds,omitempty" yaml:"subject_kinds,omitempty"`

	// MinSeverity is the minimum severity to include
	MinSeverity EventSeverity `json:"min_severity,omitempty" yaml:"min_severity,omitempty"`

	// Since filters events after this time
	Since time.Time `json:"since,omitempty" yaml:"since,omitempty"`

	// Until filters events before this time
	Until time.Time `json:"until,omitempty" yaml:"until,omitempty"`

	// CorrelationID filters by correlation
	CorrelationID string `json:"correlation_id,omitempty" yaml:"correlation_id,omitempty"`

	// Subject filters by specific subject
	Subject string `json:"subject,omitempty" yaml:"subject,omitempty"`

	// Organization scope
	Organization string `json:"organization,omitempty" yaml:"organization,omitempty"`

	// Environment scope
	Environment string `json:"environment,omitempty" yaml:"environment,omitempty"`
}

// EventSubscription represents a subscription to events
type EventSubscription struct {
	// ID is a unique identifier for this subscription
	ID string `json:"id" yaml:"id"`

	// Name is a human-readable name
	Name string `json:"name" yaml:"name"`

	// Filter defines which events to receive
	Filter EventFilter `json:"filter" yaml:"filter"`

	// Webhook URL for HTTP delivery
	WebhookURL string `json:"webhook_url,omitempty" yaml:"webhook_url,omitempty"`

	// WebhookSecret for signing webhook payloads
	WebhookSecret string `json:"webhook_secret,omitempty" yaml:"webhook_secret,omitempty"`

	// Active indicates if the subscription is active
	Active bool `json:"active" yaml:"active"`

	// CreatedAt is when the subscription was created
	CreatedAt time.Time `json:"created_at" yaml:"created_at"`

	// Owner of the subscription
	Owner string `json:"owner,omitempty" yaml:"owner,omitempty"`
}

// EventBatch represents a batch of events for bulk operations
type EventBatch struct {
	Events []Event `json:"events" yaml:"events"`
}

// NewEvent creates a new event with the given parameters
func NewEvent(eventType EventType, source, subject, subjectKind string) *Event {
	return &Event{
		ID:          generateEventID(),
		Type:        eventType,
		Source:      source,
		Subject:     subject,
		SubjectKind: subjectKind,
		Severity:    EventSeverityInfo,
		Timestamp:   time.Now(),
		Data:        make(map[string]interface{}),
		Metadata:    make(map[string]string),
	}
}

// WithSeverity sets the severity
func (e *Event) WithSeverity(severity EventSeverity) *Event {
	e.Severity = severity
	return e
}

// WithData adds data to the event
func (e *Event) WithData(key string, value interface{}) *Event {
	e.Data[key] = value
	return e
}

// WithCorrelation sets the correlation ID
func (e *Event) WithCorrelation(correlationID string) *Event {
	e.CorrelationID = correlationID
	return e
}

// WithCausation sets the causation ID
func (e *Event) WithCausation(causationID string) *Event {
	e.CausationID = causationID
	return e
}

// WithActor sets the actor
func (e *Event) WithActor(actor string) *Event {
	e.Actor = actor
	return e
}

// WithOrganization sets the organization scope
func (e *Event) WithOrganization(org string) *Event {
	e.Organization = org
	return e
}

// WithEnvironment sets the environment scope
func (e *Event) WithEnvironment(env string) *Event {
	e.Environment = env
	return e
}

// WithMetadata adds metadata to the event
func (e *Event) WithMetadata(key, value string) *Event {
	e.Metadata[key] = value
	return e
}

// generateEventID generates a unique event ID
func generateEventID() string {
	return "evt_" + randomString(20)
}

// randomString generates a random alphanumeric string
func randomString(length int) string {
	const charset = "abcdefghijklmnopqrstuvwxyz0123456789"
	b := make([]byte, length)
	for i := range b {
		b[i] = charset[time.Now().UnixNano()%int64(len(charset))]
	}
	return string(b)
}

// Matches checks if an event matches a filter
func (f *EventFilter) Matches(e *Event) bool {
	// Check types
	if len(f.Types) > 0 {
		found := false
		for _, t := range f.Types {
			if t == e.Type {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}

	// Check sources
	if len(f.Sources) > 0 {
		found := false
		for _, s := range f.Sources {
			if s == e.Source {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}

	// Check subject kinds
	if len(f.SubjectKinds) > 0 {
		found := false
		for _, k := range f.SubjectKinds {
			if k == e.SubjectKind {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}

	// Check severity
	if f.MinSeverity != "" {
		if !severityAtLeast(e.Severity, f.MinSeverity) {
			return false
		}
	}

	// Check time range
	if !f.Since.IsZero() && e.Timestamp.Before(f.Since) {
		return false
	}
	if !f.Until.IsZero() && e.Timestamp.After(f.Until) {
		return false
	}

	// Check correlation
	if f.CorrelationID != "" && e.CorrelationID != f.CorrelationID {
		return false
	}

	// Check subject
	if f.Subject != "" && e.Subject != f.Subject {
		return false
	}

	// Check organization
	if f.Organization != "" && e.Organization != f.Organization {
		return false
	}

	// Check environment
	if f.Environment != "" && e.Environment != f.Environment {
		return false
	}

	return true
}

// severityAtLeast checks if severity a is at least severity b
func severityAtLeast(a, b EventSeverity) bool {
	severityOrder := map[EventSeverity]int{
		EventSeverityInfo:     0,
		EventSeverityWarning:  1,
		EventSeverityError:    2,
		EventSeverityCritical: 3,
	}
	return severityOrder[a] >= severityOrder[b]
}
