package telemetry

import (
	"context"
	"fmt"
	"sync"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.24.0"
	"go.opentelemetry.io/otel/trace"
)

// OTelConfig configures OpenTelemetry integration
type OTelConfig struct {
	Enabled        bool              `yaml:"enabled" json:"enabled"`
	ServiceName    string            `yaml:"serviceName" json:"serviceName"`
	ServiceVersion string            `yaml:"serviceVersion" json:"serviceVersion"`
	Environment    string            `yaml:"environment" json:"environment"`
	OTLPEndpoint   string            `yaml:"otlpEndpoint" json:"otlpEndpoint"`
	OTLPInsecure   bool              `yaml:"otlpInsecure" json:"otlpInsecure"`
	OTLPHeaders    map[string]string `yaml:"otlpHeaders,omitempty" json:"otlpHeaders,omitempty"`
	SampleRate     float64           `yaml:"sampleRate" json:"sampleRate"`
}

// OTelProvider wraps OpenTelemetry tracer provider
type OTelProvider struct {
	config         *OTelConfig
	tracerProvider *sdktrace.TracerProvider
	tracer         trace.Tracer
	shutdown       func(context.Context) error
	mu             sync.RWMutex
}

var (
	globalOTelProvider *OTelProvider
	otelMu             sync.RWMutex
)

// NewOTelProvider creates a new OpenTelemetry provider
func NewOTelProvider(cfg *OTelConfig) (*OTelProvider, error) {
	if cfg == nil || !cfg.Enabled {
		return &OTelProvider{config: cfg}, nil
	}

	// Set defaults
	if cfg.ServiceName == "" {
		cfg.ServiceName = "platformfoundry"
	}
	if cfg.ServiceVersion == "" {
		cfg.ServiceVersion = "1.0.0"
	}
	if cfg.Environment == "" {
		cfg.Environment = "development"
	}
	if cfg.SampleRate == 0 {
		cfg.SampleRate = 1.0
	}

	ctx := context.Background()

	// Create resource
	res, err := resource.New(ctx,
		resource.WithAttributes(
			semconv.ServiceName(cfg.ServiceName),
			semconv.ServiceVersion(cfg.ServiceVersion),
			semconv.DeploymentEnvironment(cfg.Environment),
			attribute.String("platform", "platformfoundry"),
		),
		resource.WithHost(),
		resource.WithProcess(),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create resource: %w", err)
	}

	// Create OTLP exporter if endpoint is configured
	var exporter *otlptrace.Exporter
	if cfg.OTLPEndpoint != "" {
		opts := []otlptracegrpc.Option{
			otlptracegrpc.WithEndpoint(cfg.OTLPEndpoint),
		}
		if cfg.OTLPInsecure {
			opts = append(opts, otlptracegrpc.WithInsecure())
		}
		if len(cfg.OTLPHeaders) > 0 {
			opts = append(opts, otlptracegrpc.WithHeaders(cfg.OTLPHeaders))
		}

		exporter, err = otlptracegrpc.New(ctx, opts...)
		if err != nil {
			return nil, fmt.Errorf("failed to create OTLP exporter: %w", err)
		}
	}

	// Create sampler
	var sampler sdktrace.Sampler
	if cfg.SampleRate >= 1.0 {
		sampler = sdktrace.AlwaysSample()
	} else if cfg.SampleRate <= 0 {
		sampler = sdktrace.NeverSample()
	} else {
		sampler = sdktrace.TraceIDRatioBased(cfg.SampleRate)
	}

	// Create tracer provider
	tpOpts := []sdktrace.TracerProviderOption{
		sdktrace.WithResource(res),
		sdktrace.WithSampler(sampler),
	}
	if exporter != nil {
		tpOpts = append(tpOpts, sdktrace.WithBatcher(exporter))
	}

	tp := sdktrace.NewTracerProvider(tpOpts...)

	// Set global tracer provider
	otel.SetTracerProvider(tp)
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{},
		propagation.Baggage{},
	))

	provider := &OTelProvider{
		config:         cfg,
		tracerProvider: tp,
		tracer:         tp.Tracer(cfg.ServiceName),
		shutdown: func(ctx context.Context) error {
			return tp.Shutdown(ctx)
		},
	}

	// Set global provider
	otelMu.Lock()
	globalOTelProvider = provider
	otelMu.Unlock()

	return provider, nil
}

// Tracer returns the OpenTelemetry tracer
func (p *OTelProvider) Tracer() trace.Tracer {
	if p == nil || p.tracer == nil {
		return otel.Tracer("platformfoundry")
	}
	return p.tracer
}

// Shutdown shuts down the OpenTelemetry provider
func (p *OTelProvider) Shutdown(ctx context.Context) error {
	if p.shutdown != nil {
		return p.shutdown(ctx)
	}
	return nil
}

// GetOTelProvider returns the global OpenTelemetry provider
func GetOTelProvider() *OTelProvider {
	otelMu.RLock()
	defer otelMu.RUnlock()
	return globalOTelProvider
}

// OTelSpan wraps an OpenTelemetry span with helper methods
type OTelSpan struct {
	ctx       context.Context
	span      trace.Span
	name      string
	startTime time.Time
}

// StartOTelSpan starts a new OpenTelemetry span
func StartOTelSpan(ctx context.Context, name string, opts ...trace.SpanStartOption) *OTelSpan {
	p := GetOTelProvider()
	if p == nil || p.tracer == nil {
		return &OTelSpan{ctx: ctx, name: name, startTime: time.Now()}
	}

	ctx, span := p.tracer.Start(ctx, name, opts...)
	return &OTelSpan{
		ctx:       ctx,
		span:      span,
		name:      name,
		startTime: time.Now(),
	}
}

// Context returns the span context
func (s *OTelSpan) Context() context.Context {
	return s.ctx
}

// SetStringAttribute sets a string attribute
func (s *OTelSpan) SetStringAttribute(key, value string) {
	if s.span != nil {
		s.span.SetAttributes(attribute.String(key, value))
	}
}

// SetIntAttribute sets an integer attribute
func (s *OTelSpan) SetIntAttribute(key string, value int64) {
	if s.span != nil {
		s.span.SetAttributes(attribute.Int64(key, value))
	}
}

// SetBoolAttribute sets a boolean attribute
func (s *OTelSpan) SetBoolAttribute(key string, value bool) {
	if s.span != nil {
		s.span.SetAttributes(attribute.Bool(key, value))
	}
}

// AddEvent adds an event to the span
func (s *OTelSpan) AddEvent(name string, attrs ...attribute.KeyValue) {
	if s.span != nil {
		s.span.AddEvent(name, trace.WithAttributes(attrs...))
	}
}

// RecordError records an error on the span
func (s *OTelSpan) RecordError(err error) {
	if s.span != nil && err != nil {
		s.span.RecordError(err)
	}
}

// End ends the span
func (s *OTelSpan) End() {
	if s.span != nil {
		s.span.End()
	}
}

// EndWithError ends the span and records any error
func (s *OTelSpan) EndWithError(err error) {
	if s.span != nil {
		if err != nil {
			s.span.RecordError(err)
		}
		s.span.End()
	}
}

// Operation-specific tracing helpers

// TracePluginOp traces a plugin operation using OpenTelemetry
func TracePluginOp(ctx context.Context, pluginName, operation string) *OTelSpan {
	span := StartOTelSpan(ctx, fmt.Sprintf("plugin.%s.%s", pluginName, operation),
		trace.WithAttributes(
			attribute.String("plugin.name", pluginName),
			attribute.String("plugin.operation", operation),
		),
	)
	return span
}

// TraceStateOp traces a state backend operation
func TraceStateOp(ctx context.Context, backend, operation, stateID string) *OTelSpan {
	span := StartOTelSpan(ctx, fmt.Sprintf("state.%s.%s", backend, operation),
		trace.WithAttributes(
			attribute.String("state.backend", backend),
			attribute.String("state.operation", operation),
			attribute.String("state.id", stateID),
		),
	)
	return span
}

// TraceAuthOp traces an authentication operation
func TraceAuthOp(ctx context.Context, authType, operation string) *OTelSpan {
	span := StartOTelSpan(ctx, fmt.Sprintf("auth.%s.%s", authType, operation),
		trace.WithAttributes(
			attribute.String("auth.type", authType),
			attribute.String("auth.operation", operation),
		),
	)
	return span
}

// TracePolicyOp traces a policy evaluation
func TracePolicyOp(ctx context.Context, policyName string) *OTelSpan {
	span := StartOTelSpan(ctx, "policy.evaluate",
		trace.WithAttributes(
			attribute.String("policy.name", policyName),
		),
	)
	return span
}

// TraceSecretsOp traces a secrets operation
func TraceSecretsOp(ctx context.Context, provider, operation string) *OTelSpan {
	span := StartOTelSpan(ctx, fmt.Sprintf("secrets.%s.%s", provider, operation),
		trace.WithAttributes(
			attribute.String("secrets.provider", provider),
			attribute.String("secrets.operation", operation),
		),
	)
	return span
}

// TraceOrchestrationOp traces an orchestration operation
func TraceOrchestrationOp(ctx context.Context, orchestrationType string, resourceCount int) *OTelSpan {
	span := StartOTelSpan(ctx, "orchestration.execute",
		trace.WithAttributes(
			attribute.String("orchestration.type", orchestrationType),
			attribute.Int("orchestration.resource_count", resourceCount),
		),
	)
	return span
}

// TraceCLI traces a CLI command execution
func TraceCLI(ctx context.Context, command string, args []string) *OTelSpan {
	span := StartOTelSpan(ctx, fmt.Sprintf("cli.%s", command),
		trace.WithAttributes(
			attribute.String("cli.command", command),
			attribute.Int("cli.args_count", len(args)),
		),
	)
	return span
}

// TraceHTTPRequest traces an HTTP request
func TraceHTTPRequest(ctx context.Context, method, path string) *OTelSpan {
	span := StartOTelSpan(ctx, fmt.Sprintf("http.%s", method),
		trace.WithAttributes(
			attribute.String("http.method", method),
			attribute.String("http.path", path),
		),
	)
	return span
}

// TraceDBQuery traces a database query
func TraceDBQuery(ctx context.Context, dbType, operation, table string) *OTelSpan {
	span := StartOTelSpan(ctx, fmt.Sprintf("db.%s.%s", dbType, operation),
		trace.WithAttributes(
			attribute.String("db.type", dbType),
			attribute.String("db.operation", operation),
			attribute.String("db.table", table),
		),
	)
	return span
}

// TraceCacheOp traces a cache operation
func TraceCacheOp(ctx context.Context, cacheType, operation, key string) *OTelSpan {
	span := StartOTelSpan(ctx, fmt.Sprintf("cache.%s.%s", cacheType, operation),
		trace.WithAttributes(
			attribute.String("cache.type", cacheType),
			attribute.String("cache.operation", operation),
			attribute.String("cache.key", key),
		),
	)
	return span
}

// InjectTraceContext injects trace context into a map for propagation
func InjectTraceContext(ctx context.Context, carrier map[string]string) {
	propagator := otel.GetTextMapPropagator()
	propagator.Inject(ctx, propagation.MapCarrier(carrier))
}

// ExtractTraceContext extracts trace context from a map
func ExtractTraceContext(ctx context.Context, carrier map[string]string) context.Context {
	propagator := otel.GetTextMapPropagator()
	return propagator.Extract(ctx, propagation.MapCarrier(carrier))
}
