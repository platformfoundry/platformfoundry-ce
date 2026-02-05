package telemetry

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewOTelProvider_Disabled(t *testing.T) {
	provider, err := NewOTelProvider(nil)
	require.NoError(t, err)
	assert.NotNil(t, provider)
}

func TestNewOTelProvider_DisabledExplicit(t *testing.T) {
	cfg := &OTelConfig{Enabled: false}
	provider, err := NewOTelProvider(cfg)
	require.NoError(t, err)
	assert.NotNil(t, provider)
}

func TestNewOTelProvider_Defaults(t *testing.T) {
	cfg := &OTelConfig{
		Enabled: true,
		// No endpoint, so no exporter
	}

	provider, err := NewOTelProvider(cfg)
	require.NoError(t, err)
	assert.NotNil(t, provider)
	assert.NotNil(t, provider.Tracer())

	// Cleanup
	provider.Shutdown(context.Background())
}

func TestNewOTelProvider_WithConfig(t *testing.T) {
	cfg := &OTelConfig{
		Enabled:        true,
		ServiceName:    "test-service",
		ServiceVersion: "1.2.3",
		Environment:    "testing",
		SampleRate:     0.5,
	}

	provider, err := NewOTelProvider(cfg)
	require.NoError(t, err)
	assert.NotNil(t, provider)
	assert.Equal(t, cfg, provider.config)

	provider.Shutdown(context.Background())
}

func TestOTelConfig_Defaults(t *testing.T) {
	cfg := &OTelConfig{
		Enabled: true,
	}

	provider, err := NewOTelProvider(cfg)
	require.NoError(t, err)

	// Defaults should be set
	assert.Equal(t, "platformfoundry", cfg.ServiceName)
	assert.Equal(t, "1.0.0", cfg.ServiceVersion)
	assert.Equal(t, "development", cfg.Environment)
	assert.Equal(t, 1.0, cfg.SampleRate)

	provider.Shutdown(context.Background())
}

func TestOTelSpan_BasicOperations(t *testing.T) {
	cfg := &OTelConfig{Enabled: true}
	provider, err := NewOTelProvider(cfg)
	require.NoError(t, err)
	defer provider.Shutdown(context.Background())

	ctx := context.Background()
	span := StartOTelSpan(ctx, "test-operation")
	assert.NotNil(t, span)
	assert.NotNil(t, span.Context())

	span.SetStringAttribute("key", "value")
	span.SetIntAttribute("count", 42)
	span.SetBoolAttribute("enabled", true)
	span.AddEvent("test-event")
	span.End()
}

func TestOTelSpan_WithError(t *testing.T) {
	cfg := &OTelConfig{Enabled: true}
	provider, err := NewOTelProvider(cfg)
	require.NoError(t, err)
	defer provider.Shutdown(context.Background())

	ctx := context.Background()
	span := StartOTelSpan(ctx, "operation-with-error")

	testErr := assert.AnError
	span.RecordError(testErr)
	span.EndWithError(testErr)
}

func TestOTelSpan_NilError(t *testing.T) {
	cfg := &OTelConfig{Enabled: true}
	provider, err := NewOTelProvider(cfg)
	require.NoError(t, err)
	defer provider.Shutdown(context.Background())

	ctx := context.Background()
	span := StartOTelSpan(ctx, "operation-no-error")

	span.RecordError(nil) // Should not panic
	span.EndWithError(nil)
}

func TestOTelSpan_DisabledProvider(t *testing.T) {
	// Clear global provider
	otelMu.Lock()
	globalOTelProvider = nil
	otelMu.Unlock()

	ctx := context.Background()
	span := StartOTelSpan(ctx, "test-operation")

	// Should work without panicking even with no provider
	assert.NotNil(t, span)
	span.SetStringAttribute("key", "value")
	span.End()
}

func TestTracePluginOp(t *testing.T) {
	cfg := &OTelConfig{Enabled: true}
	provider, err := NewOTelProvider(cfg)
	require.NoError(t, err)
	defer provider.Shutdown(context.Background())

	ctx := context.Background()
	span := TracePluginOp(ctx, "terraform", "apply")
	assert.NotNil(t, span)
	span.End()
}

func TestTraceStateOp(t *testing.T) {
	cfg := &OTelConfig{Enabled: true}
	provider, err := NewOTelProvider(cfg)
	require.NoError(t, err)
	defer provider.Shutdown(context.Background())

	ctx := context.Background()
	span := TraceStateOp(ctx, "postgres", "get", "state-123")
	assert.NotNil(t, span)
	span.End()
}

func TestTraceAuthOp(t *testing.T) {
	cfg := &OTelConfig{Enabled: true}
	provider, err := NewOTelProvider(cfg)
	require.NoError(t, err)
	defer provider.Shutdown(context.Background())

	ctx := context.Background()
	span := TraceAuthOp(ctx, "jwt", "validate")
	assert.NotNil(t, span)
	span.End()
}

func TestTracePolicyOp(t *testing.T) {
	cfg := &OTelConfig{Enabled: true}
	provider, err := NewOTelProvider(cfg)
	require.NoError(t, err)
	defer provider.Shutdown(context.Background())

	ctx := context.Background()
	span := TracePolicyOp(ctx, "deny-public-s3")
	assert.NotNil(t, span)
	span.End()
}

func TestTraceSecretsOp(t *testing.T) {
	cfg := &OTelConfig{Enabled: true}
	provider, err := NewOTelProvider(cfg)
	require.NoError(t, err)
	defer provider.Shutdown(context.Background())

	ctx := context.Background()
	span := TraceSecretsOp(ctx, "vault", "read")
	assert.NotNil(t, span)
	span.End()
}

func TestTraceOrchestrationOp(t *testing.T) {
	cfg := &OTelConfig{Enabled: true}
	provider, err := NewOTelProvider(cfg)
	require.NoError(t, err)
	defer provider.Shutdown(context.Background())

	ctx := context.Background()
	span := TraceOrchestrationOp(ctx, "infrastructure", 10)
	assert.NotNil(t, span)
	span.End()
}

func TestTraceCLI(t *testing.T) {
	cfg := &OTelConfig{Enabled: true}
	provider, err := NewOTelProvider(cfg)
	require.NoError(t, err)
	defer provider.Shutdown(context.Background())

	ctx := context.Background()
	span := TraceCLI(ctx, "apply", []string{"--auto-approve"})
	assert.NotNil(t, span)
	span.End()
}

func TestTraceHTTPRequest(t *testing.T) {
	cfg := &OTelConfig{Enabled: true}
	provider, err := NewOTelProvider(cfg)
	require.NoError(t, err)
	defer provider.Shutdown(context.Background())

	ctx := context.Background()
	span := TraceHTTPRequest(ctx, "GET", "/api/v1/resources")
	assert.NotNil(t, span)
	span.End()
}

func TestTraceDBQuery(t *testing.T) {
	cfg := &OTelConfig{Enabled: true}
	provider, err := NewOTelProvider(cfg)
	require.NoError(t, err)
	defer provider.Shutdown(context.Background())

	ctx := context.Background()
	span := TraceDBQuery(ctx, "postgres", "SELECT", "states")
	assert.NotNil(t, span)
	span.End()
}

func TestTraceCacheOp(t *testing.T) {
	cfg := &OTelConfig{Enabled: true}
	provider, err := NewOTelProvider(cfg)
	require.NoError(t, err)
	defer provider.Shutdown(context.Background())

	ctx := context.Background()
	span := TraceCacheOp(ctx, "redis", "GET", "session:123")
	assert.NotNil(t, span)
	span.End()
}

func TestGetOTelProvider(t *testing.T) {
	// Clear provider
	otelMu.Lock()
	globalOTelProvider = nil
	otelMu.Unlock()

	assert.Nil(t, GetOTelProvider())

	cfg := &OTelConfig{Enabled: true}
	provider, err := NewOTelProvider(cfg)
	require.NoError(t, err)
	defer provider.Shutdown(context.Background())

	assert.NotNil(t, GetOTelProvider())
	assert.Equal(t, provider, GetOTelProvider())
}

func TestInjectExtractTraceContext(t *testing.T) {
	cfg := &OTelConfig{Enabled: true}
	provider, err := NewOTelProvider(cfg)
	require.NoError(t, err)
	defer provider.Shutdown(context.Background())

	ctx := context.Background()

	// Start a span to have trace context
	span := StartOTelSpan(ctx, "parent-operation")
	ctxWithSpan := span.Context()

	// Inject trace context into carrier
	carrier := make(map[string]string)
	InjectTraceContext(ctxWithSpan, carrier)

	// Extract trace context from carrier
	extractedCtx := ExtractTraceContext(context.Background(), carrier)
	assert.NotNil(t, extractedCtx)

	span.End()
}

func TestOTelProvider_Shutdown(t *testing.T) {
	cfg := &OTelConfig{Enabled: true}
	provider, err := NewOTelProvider(cfg)
	require.NoError(t, err)

	err = provider.Shutdown(context.Background())
	assert.NoError(t, err)

	// Shutdown again should be safe
	err = provider.Shutdown(context.Background())
	assert.NoError(t, err)
}

func TestOTelProvider_ShutdownNil(t *testing.T) {
	provider := &OTelProvider{}
	err := provider.Shutdown(context.Background())
	assert.NoError(t, err)
}

func TestSampleRates(t *testing.T) {
	tests := []struct {
		name       string
		sampleRate float64
	}{
		{"always sample", 1.0},
		{"never sample", 0.0},
		{"half sample", 0.5},
		{"quarter sample", 0.25},
		{"over 1.0", 1.5},    // Should be clamped to always
		{"negative", -0.5},   // Should be clamped to never
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &OTelConfig{
				Enabled:    true,
				SampleRate: tt.sampleRate,
			}

			provider, err := NewOTelProvider(cfg)
			require.NoError(t, err)
			assert.NotNil(t, provider)
			provider.Shutdown(context.Background())
		})
	}
}

func BenchmarkStartOTelSpan(b *testing.B) {
	cfg := &OTelConfig{Enabled: true}
	provider, _ := NewOTelProvider(cfg)
	defer provider.Shutdown(context.Background())

	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		span := StartOTelSpan(ctx, "benchmark-operation")
		span.End()
	}
}

func BenchmarkStartOTelSpan_Disabled(b *testing.B) {
	otelMu.Lock()
	globalOTelProvider = nil
	otelMu.Unlock()

	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		span := StartOTelSpan(ctx, "benchmark-operation")
		span.End()
	}
}
