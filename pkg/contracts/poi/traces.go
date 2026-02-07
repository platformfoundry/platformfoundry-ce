// Package poi defines the Platform Observability Interface (POI).
package poi

import (
	"context"
	"time"
)

// TraceExporter exports traces to an observability backend
type TraceExporter interface {
	// Export sends spans to the backend
	Export(ctx context.Context, spans []Span) error

	// Flush forces any buffered spans to be sent
	Flush(ctx context.Context) error

	// Close releases exporter resources
	Close() error
}

// Span represents a single span in a distributed trace
type Span struct {
	// TraceID is the unique trace identifier
	TraceID string

	// SpanID is the unique span identifier
	SpanID string

	// ParentSpanID is the parent span identifier (empty for root spans)
	ParentSpanID string

	// Name is the operation name
	Name string

	// Kind is the span kind
	Kind SpanKind

	// StartTime is when the span started
	StartTime time.Time

	// EndTime is when the span ended
	EndTime time.Time

	// Status is the span status
	Status SpanStatus

	// Attributes contains span attributes
	Attributes map[string]interface{}

	// Events contains span events
	Events []SpanEvent

	// Links contains links to other spans
	Links []SpanLink

	// Resource identifies the source of the span
	Resource ResourceInfo
}

// SpanKind represents the type of span
type SpanKind string

const (
	SpanKindUnspecified SpanKind = "unspecified"
	SpanKindInternal    SpanKind = "internal"
	SpanKindServer      SpanKind = "server"
	SpanKindClient      SpanKind = "client"
	SpanKindProducer    SpanKind = "producer"
	SpanKindConsumer    SpanKind = "consumer"
)

// SpanStatus represents the status of a span
type SpanStatus struct {
	// Code is the status code
	Code SpanStatusCode

	// Message is the status message
	Message string
}

// SpanStatusCode represents a span status code
type SpanStatusCode string

const (
	SpanStatusUnset SpanStatusCode = "unset"
	SpanStatusOK    SpanStatusCode = "ok"
	SpanStatusError SpanStatusCode = "error"
)

// SpanEvent represents an event within a span
type SpanEvent struct {
	// Name is the event name
	Name string

	// Timestamp is when the event occurred
	Timestamp time.Time

	// Attributes contains event attributes
	Attributes map[string]interface{}
}

// SpanLink represents a link to another span
type SpanLink struct {
	// TraceID is the linked trace identifier
	TraceID string

	// SpanID is the linked span identifier
	SpanID string

	// Attributes contains link attributes
	Attributes map[string]interface{}
}

// Tracer creates spans for distributed tracing
type Tracer interface {
	// Start starts a new span
	Start(ctx context.Context, name string, opts ...SpanOption) (context.Context, SpanHandle)

	// SpanFromContext returns the current span from context
	SpanFromContext(ctx context.Context) SpanHandle
}

// SpanHandle provides operations on an active span
type SpanHandle interface {
	// End ends the span
	End(opts ...SpanEndOption)

	// SetAttributes sets span attributes
	SetAttributes(attrs ...interface{})

	// SetStatus sets the span status
	SetStatus(code SpanStatusCode, message string)

	// AddEvent adds an event to the span
	AddEvent(name string, attrs ...interface{})

	// RecordError records an error on the span
	RecordError(err error, attrs ...interface{})

	// SpanContext returns the span context
	SpanContext() SpanContext
}

// SpanContext contains the identifying information for a span
type SpanContext struct {
	// TraceID is the trace identifier
	TraceID string

	// SpanID is the span identifier
	SpanID string

	// TraceFlags contains trace flags
	TraceFlags byte

	// TraceState contains trace state
	TraceState string
}

// SpanOption configures span creation
type SpanOption interface {
	applySpan(*spanConfig)
}

// SpanEndOption configures span ending
type SpanEndOption interface {
	applySpanEnd(*spanEndConfig)
}

type spanConfig struct {
	Kind       SpanKind
	Attributes map[string]interface{}
	Links      []SpanLink
	StartTime  time.Time
}

type spanEndConfig struct {
	EndTime time.Time
}

// WithSpanKind sets the span kind
func WithSpanKind(kind SpanKind) SpanOption {
	return spanKindOption{kind: kind}
}

type spanKindOption struct {
	kind SpanKind
}

func (o spanKindOption) applySpan(c *spanConfig) {
	c.Kind = o.kind
}

// TraceQuery represents a query for traces
type TraceQuery struct {
	// TraceID filters by trace ID
	TraceID string

	// ServiceName filters by service name
	ServiceName string

	// OperationName filters by operation name
	OperationName string

	// Start is the start of the query time range
	Start time.Time

	// End is the end of the query time range
	End time.Time

	// MinDuration filters by minimum duration
	MinDuration time.Duration

	// MaxDuration filters by maximum duration
	MaxDuration time.Duration

	// Limit is the maximum number of traces
	Limit int
}

// TraceReader reads traces from a backend
type TraceReader interface {
	// GetTrace retrieves a trace by ID
	GetTrace(ctx context.Context, traceID string) (*Trace, error)

	// Query executes a trace query
	Query(ctx context.Context, query *TraceQuery) ([]Trace, error)
}

// Trace represents a complete distributed trace
type Trace struct {
	// TraceID is the unique trace identifier
	TraceID string

	// Spans contains all spans in the trace
	Spans []Span

	// Duration is the total trace duration
	Duration time.Duration

	// Services contains service names in this trace
	Services []string
}
