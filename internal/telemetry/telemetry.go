package telemetry

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
	"sync"
	"sync/atomic"
	"time"

	"github.com/platformfoundry/pf-ce/pkg/types"
)

// Provider provides telemetry capabilities for the platform
type Provider struct {
	mu        sync.RWMutex
	config    types.TelemetryConfig
	logger    *slog.Logger
	tracer    TracerInterface
	meter     MeterInterface
	stats     *types.TelemetryStats
	startTime time.Time
}

// TracerInterface interface for distributed tracing
type TracerInterface interface {
	// StartSpan starts a new span
	StartSpan(ctx context.Context, name string, opts ...SpanOption) (context.Context, SpanInterface)

	// SpanFromContext returns the current span from context
	SpanFromContext(ctx context.Context) SpanInterface

	// Shutdown gracefully shuts down the tracer
	Shutdown(ctx context.Context) error
}

// SpanInterface represents an active span
type SpanInterface interface {
	// End ends the span
	End()

	// SetStatus sets the span status
	SetStatus(code StatusCode, description string)

	// SetAttributes sets attributes on the span
	SetAttributes(attrs ...Attribute)

	// AddEvent adds an event to the span
	AddEvent(name string, attrs ...Attribute)

	// RecordError records an error
	RecordError(err error)

	// SpanContext returns the span context
	SpanContext() SpanContext
}

// SpanContext contains trace and span IDs
type SpanContext struct {
	TraceID string
	SpanID  string
}

// StatusCode represents span status
type StatusCode int

const (
	StatusUnset StatusCode = iota
	StatusOK
	StatusError
)

// Attribute represents a key-value attribute
type Attribute struct {
	Key   string
	Value interface{}
}

// SpanOption configures span creation
type SpanOption func(*spanConfig)

type spanConfig struct {
	kind       string
	attributes []Attribute
}

// WithSpanKind sets the span kind
func WithSpanKind(kind string) SpanOption {
	return func(c *spanConfig) {
		c.kind = kind
	}
}

// WithAttributes sets initial attributes
func WithAttributes(attrs ...Attribute) SpanOption {
	return func(c *spanConfig) {
		c.attributes = append(c.attributes, attrs...)
	}
}

// MeterInterface for metrics
type MeterInterface interface {
	// CreateCounter creates a new counter
	CreateCounter(name string, opts ...MetricOption) CounterInterface

	// CreateGauge creates a new gauge
	CreateGauge(name string, opts ...MetricOption) GaugeInterface

	// CreateHistogram creates a new histogram
	CreateHistogram(name string, opts ...MetricOption) HistogramInterface

	// Shutdown gracefully shuts down the meter
	Shutdown(ctx context.Context) error
}

// CounterInterface represents a monotonically increasing value
type CounterInterface interface {
	Add(ctx context.Context, value float64, attrs ...Attribute)
}

// GaugeInterface represents a value that can go up and down
type GaugeInterface interface {
	Record(ctx context.Context, value float64, attrs ...Attribute)
}

// HistogramInterface represents a distribution of values
type HistogramInterface interface {
	Record(ctx context.Context, value float64, attrs ...Attribute)
}

// MetricOption configures metric creation
type MetricOption func(*metricConfig)

type metricConfig struct {
	unit        string
	description string
}

// WithUnit sets the metric unit
func WithUnit(unit string) MetricOption {
	return func(c *metricConfig) {
		c.unit = unit
	}
}

// WithDescription sets the metric description
func WithDescription(desc string) MetricOption {
	return func(c *metricConfig) {
		c.description = desc
	}
}

// New creates a new telemetry provider
func New(config types.TelemetryConfig) (*Provider, error) {
	p := &Provider{
		config:    config,
		startTime: time.Now(),
		stats: &types.TelemetryStats{
			StartTime: time.Now(),
		},
	}

	// Setup logging
	if err := p.setupLogging(); err != nil {
		return nil, fmt.Errorf("failed to setup logging: %w", err)
	}

	// Setup tracing
	if config.Tracing.Enabled {
		if err := p.setupTracing(); err != nil {
			return nil, fmt.Errorf("failed to setup tracing: %w", err)
		}
	} else {
		p.tracer = &noopTracer{}
	}

	// Setup metrics
	if config.Metrics.Enabled {
		if err := p.setupMetrics(); err != nil {
			return nil, fmt.Errorf("failed to setup metrics: %w", err)
		}
	} else {
		p.meter = &noopMeter{}
	}

	return p, nil
}

// setupLogging configures the logger
func (p *Provider) setupLogging() error {
	var output io.Writer = os.Stdout
	if p.config.Logging.Output == "stderr" {
		output = os.Stderr
	} else if p.config.Logging.Output == "file" && p.config.Logging.FilePath != "" {
		f, err := os.OpenFile(p.config.Logging.FilePath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
		if err != nil {
			return err
		}
		output = f
	}

	level := slog.LevelInfo
	switch p.config.Logging.Level {
	case "debug":
		level = slog.LevelDebug
	case "warn":
		level = slog.LevelWarn
	case "error":
		level = slog.LevelError
	}

	opts := &slog.HandlerOptions{
		Level:     level,
		AddSource: p.config.Logging.AddSource,
	}

	var handler slog.Handler
	if p.config.Logging.Format == "json" {
		handler = slog.NewJSONHandler(output, opts)
	} else {
		handler = slog.NewTextHandler(output, opts)
	}

	p.logger = slog.New(handler).With(
		"service", p.config.ServiceName,
		"version", p.config.ServiceVersion,
		"env", p.config.Environment,
	)

	return nil
}

// setupTracing configures distributed tracing
func (p *Provider) setupTracing() error {
	// For now, use a simple tracer implementation
	// In production, this would integrate with OpenTelemetry
	p.tracer = &simpleTracer{
		serviceName: p.config.ServiceName,
		stats:       p.stats,
	}
	return nil
}

// setupMetrics configures metrics collection
func (p *Provider) setupMetrics() error {
	// For now, use a simple meter implementation
	// In production, this would integrate with OpenTelemetry
	p.meter = &simpleMeter{
		serviceName: p.config.ServiceName,
		stats:       p.stats,
	}
	return nil
}

// Logger returns the configured logger
func (p *Provider) Logger() *slog.Logger {
	return p.logger
}

// Tracer returns the configured tracer
func (p *Provider) Tracer() TracerInterface {
	return p.tracer
}

// Meter returns the configured meter
func (p *Provider) Meter() MeterInterface {
	return p.meter
}

// Stats returns telemetry statistics
func (p *Provider) Stats() *types.TelemetryStats {
	p.mu.RLock()
	defer p.mu.RUnlock()

	stats := *p.stats
	stats.Uptime = time.Since(p.startTime)
	return &stats
}

// Shutdown gracefully shuts down all telemetry components
func (p *Provider) Shutdown(ctx context.Context) error {
	var errs []error

	if p.tracer != nil {
		if err := p.tracer.Shutdown(ctx); err != nil {
			errs = append(errs, fmt.Errorf("tracer shutdown: %w", err))
		}
	}

	if p.meter != nil {
		if err := p.meter.Shutdown(ctx); err != nil {
			errs = append(errs, fmt.Errorf("meter shutdown: %w", err))
		}
	}

	if len(errs) > 0 {
		return fmt.Errorf("shutdown errors: %v", errs)
	}
	return nil
}

// simpleTracer implements TracerInterface
type simpleTracer struct {
	serviceName string
	stats       *types.TelemetryStats
}

func (t *simpleTracer) StartSpan(ctx context.Context, name string, opts ...SpanOption) (context.Context, SpanInterface) {
	atomic.AddInt64(&t.stats.SpansCollected, 1)

	span := &simpleSpan{
		name:      name,
		startTime: time.Now(),
		context: SpanContext{
			TraceID: generateTraceID(),
			SpanID:  generateSpanID(),
		},
	}

	// Apply options
	cfg := &spanConfig{}
	for _, opt := range opts {
		opt(cfg)
	}
	span.attributes = cfg.attributes

	return context.WithValue(ctx, spanKey{}, span), span
}

func (t *simpleTracer) SpanFromContext(ctx context.Context) SpanInterface {
	span, _ := ctx.Value(spanKey{}).(SpanInterface)
	if span == nil {
		return &noopSpan{}
	}
	return span
}

func (t *simpleTracer) Shutdown(ctx context.Context) error {
	return nil
}

type spanKey struct{}

// simpleSpan implements SpanInterface
type simpleSpan struct {
	name       string
	startTime  time.Time
	endTime    time.Time
	status     StatusCode
	statusDesc string
	attributes []Attribute
	events     []spanEvent
	context    SpanContext
}

type spanEvent struct {
	name       string
	timestamp  time.Time
	attributes []Attribute
}

func (s *simpleSpan) End() {
	s.endTime = time.Now()
}

func (s *simpleSpan) SetStatus(code StatusCode, description string) {
	s.status = code
	s.statusDesc = description
}

func (s *simpleSpan) SetAttributes(attrs ...Attribute) {
	s.attributes = append(s.attributes, attrs...)
}

func (s *simpleSpan) AddEvent(name string, attrs ...Attribute) {
	s.events = append(s.events, spanEvent{
		name:       name,
		timestamp:  time.Now(),
		attributes: attrs,
	})
}

func (s *simpleSpan) RecordError(err error) {
	if err != nil {
		s.AddEvent("exception", Attribute{Key: "exception.message", Value: err.Error()})
		s.SetStatus(StatusError, err.Error())
	}
}

func (s *simpleSpan) SpanContext() SpanContext {
	return s.context
}

// simpleMeter implements MeterInterface
type simpleMeter struct {
	serviceName string
	stats       *types.TelemetryStats
	mu          sync.RWMutex
	counters    map[string]*telemetryCounter
	gauges      map[string]*telemetryGauge
	histograms  map[string]*telemetryHistogram
}

func (m *simpleMeter) CreateCounter(name string, opts ...MetricOption) CounterInterface {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.counters == nil {
		m.counters = make(map[string]*telemetryCounter)
	}

	if c, exists := m.counters[name]; exists {
		return c
	}

	c := &telemetryCounter{name: name, stats: m.stats}
	m.counters[name] = c
	return c
}

func (m *simpleMeter) CreateGauge(name string, opts ...MetricOption) GaugeInterface {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.gauges == nil {
		m.gauges = make(map[string]*telemetryGauge)
	}

	if g, exists := m.gauges[name]; exists {
		return g
	}

	g := &telemetryGauge{name: name, stats: m.stats}
	m.gauges[name] = g
	return g
}

func (m *simpleMeter) CreateHistogram(name string, opts ...MetricOption) HistogramInterface {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.histograms == nil {
		m.histograms = make(map[string]*telemetryHistogram)
	}

	if h, exists := m.histograms[name]; exists {
		return h
	}

	h := &telemetryHistogram{name: name, stats: m.stats}
	m.histograms[name] = h
	return h
}

func (m *simpleMeter) Shutdown(ctx context.Context) error {
	return nil
}

type telemetryCounter struct {
	name  string
	value float64
	mu    sync.Mutex
	stats *types.TelemetryStats
}

func (c *telemetryCounter) Add(ctx context.Context, value float64, attrs ...Attribute) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.value += value
	atomic.AddInt64(&c.stats.MetricsCollected, 1)
}

type telemetryGauge struct {
	name  string
	value float64
	mu    sync.Mutex
	stats *types.TelemetryStats
}

func (g *telemetryGauge) Record(ctx context.Context, value float64, attrs ...Attribute) {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.value = value
	atomic.AddInt64(&g.stats.MetricsCollected, 1)
}

type telemetryHistogram struct {
	name   string
	values []float64
	mu     sync.Mutex
	stats  *types.TelemetryStats
}

func (h *telemetryHistogram) Record(ctx context.Context, value float64, attrs ...Attribute) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.values = append(h.values, value)
	atomic.AddInt64(&h.stats.MetricsCollected, 1)
}

// noopTracer implements TracerInterface with no-op operations
type noopTracer struct{}

func (t *noopTracer) StartSpan(ctx context.Context, name string, opts ...SpanOption) (context.Context, SpanInterface) {
	return ctx, &noopSpan{}
}

func (t *noopTracer) SpanFromContext(ctx context.Context) SpanInterface {
	return &noopSpan{}
}

func (t *noopTracer) Shutdown(ctx context.Context) error {
	return nil
}

// noopSpan implements SpanInterface with no-op operations
type noopSpan struct{}

func (s *noopSpan) End()                                         {}
func (s *noopSpan) SetStatus(code StatusCode, description string) {}
func (s *noopSpan) SetAttributes(attrs ...Attribute)              {}
func (s *noopSpan) AddEvent(name string, attrs ...Attribute)      {}
func (s *noopSpan) RecordError(err error)                         {}
func (s *noopSpan) SpanContext() SpanContext                      { return SpanContext{} }

// noopMeter implements MeterInterface with no-op operations
type noopMeter struct{}

func (m *noopMeter) CreateCounter(name string, opts ...MetricOption) CounterInterface   { return &noopCounter{} }
func (m *noopMeter) CreateGauge(name string, opts ...MetricOption) GaugeInterface       { return &noopGauge{} }
func (m *noopMeter) CreateHistogram(name string, opts ...MetricOption) HistogramInterface { return &noopHistogram{} }
func (m *noopMeter) Shutdown(ctx context.Context) error                                   { return nil }

type noopCounter struct{}

func (c *noopCounter) Add(ctx context.Context, value float64, attrs ...Attribute) {}

type noopGauge struct{}

func (g *noopGauge) Record(ctx context.Context, value float64, attrs ...Attribute) {}

type noopHistogram struct{}

func (h *noopHistogram) Record(ctx context.Context, value float64, attrs ...Attribute) {}

// Helper functions
func generateTraceID() string {
	return fmt.Sprintf("%016x%016x", time.Now().UnixNano(), time.Now().UnixNano()>>8)
}

func generateSpanID() string {
	return fmt.Sprintf("%016x", time.Now().UnixNano())
}

// Predefined metrics for Platform Foundry
const (
	MetricOperationsTotal     = "pf.operations.total"
	MetricOperationsDuration  = "pf.operations.duration"
	MetricPluginCalls         = "pf.plugin.calls"
	MetricDriftDetected       = "pf.drift.detected"
	MetricPromiseRequests     = "pf.promise.requests"
	MetricHealthScore         = "pf.health.score"
	MetricResourceCount       = "pf.resources.count"
	MetricEventPublished      = "pf.events.published"
	MetricAPIRequests         = "pf.api.requests"
	MetricAPILatency          = "pf.api.latency"
)

// PlatformMetrics provides platform-specific metrics
type PlatformMetrics struct {
	OperationsTotal    CounterInterface
	OperationsDuration HistogramInterface
	PluginCalls        CounterInterface
	DriftDetected      CounterInterface
	PromiseRequests    CounterInterface
	HealthScore        GaugeInterface
	ResourceCount      GaugeInterface
	EventPublished     CounterInterface
	APIRequests        CounterInterface
	APILatency         HistogramInterface
}

// NewPlatformMetrics creates platform metrics from a meter
func NewPlatformMetrics(meter MeterInterface) *PlatformMetrics {
	return &PlatformMetrics{
		OperationsTotal:    meter.CreateCounter(MetricOperationsTotal, WithDescription("Total platform operations")),
		OperationsDuration: meter.CreateHistogram(MetricOperationsDuration, WithUnit("ms"), WithDescription("Operation duration")),
		PluginCalls:        meter.CreateCounter(MetricPluginCalls, WithDescription("Plugin invocations")),
		DriftDetected:      meter.CreateCounter(MetricDriftDetected, WithDescription("Drift detections")),
		PromiseRequests:    meter.CreateCounter(MetricPromiseRequests, WithDescription("Promise requests")),
		HealthScore:        meter.CreateGauge(MetricHealthScore, WithDescription("Platform health score")),
		ResourceCount:      meter.CreateGauge(MetricResourceCount, WithDescription("Managed resource count")),
		EventPublished:     meter.CreateCounter(MetricEventPublished, WithDescription("Events published")),
		APIRequests:        meter.CreateCounter(MetricAPIRequests, WithDescription("API requests")),
		APILatency:         meter.CreateHistogram(MetricAPILatency, WithUnit("ms"), WithDescription("API latency")),
	}
}
