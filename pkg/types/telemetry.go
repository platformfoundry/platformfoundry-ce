package types

import (
	"time"
)

// TelemetryConfig configures platform telemetry
type TelemetryConfig struct {
	// Enabled enables telemetry collection
	Enabled bool `yaml:"enabled" json:"enabled"`

	// ServiceName for this instance
	ServiceName string `yaml:"serviceName" json:"serviceName"`

	// ServiceVersion
	ServiceVersion string `yaml:"serviceVersion" json:"serviceVersion"`

	// Environment (production, staging, development)
	Environment string `yaml:"environment" json:"environment"`

	// Tracing configuration
	Tracing TracingConfig `yaml:"tracing" json:"tracing"`

	// Metrics configuration
	Metrics MetricsConfig `yaml:"metrics" json:"metrics"`

	// Logging configuration
	Logging TelemetryLoggingConfig `yaml:"logging" json:"logging"`
}

// TracingConfig configures distributed tracing
type TracingConfig struct {
	// Enabled enables tracing
	Enabled bool `yaml:"enabled" json:"enabled"`

	// Exporter type (jaeger, zipkin, otlp, stdout)
	Exporter string `yaml:"exporter" json:"exporter"`

	// Endpoint for the exporter
	Endpoint string `yaml:"endpoint" json:"endpoint"`

	// SampleRate (0.0 to 1.0)
	SampleRate float64 `yaml:"sampleRate" json:"sampleRate"`

	// Headers for authentication
	Headers map[string]string `yaml:"headers,omitempty" json:"headers,omitempty"`

	// Insecure allows insecure connections
	Insecure bool `yaml:"insecure" json:"insecure"`
}

// MetricsConfig configures metrics collection
type MetricsConfig struct {
	// Enabled enables metrics
	Enabled bool `yaml:"enabled" json:"enabled"`

	// Exporter type (prometheus, otlp, stdout)
	Exporter string `yaml:"exporter" json:"exporter"`

	// Endpoint for the exporter
	Endpoint string `yaml:"endpoint" json:"endpoint"`

	// PushInterval for push-based exporters
	PushInterval time.Duration `yaml:"pushInterval" json:"pushInterval"`

	// Port for Prometheus scraping
	Port int `yaml:"port" json:"port"`

	// Path for Prometheus scraping
	Path string `yaml:"path" json:"path"`
}

// TelemetryLoggingConfig configures structured logging
type TelemetryLoggingConfig struct {
	// Level (debug, info, warn, error)
	Level string `yaml:"level" json:"level"`

	// Format (json, text)
	Format string `yaml:"format" json:"format"`

	// Output (stdout, stderr, file)
	Output string `yaml:"output" json:"output"`

	// FilePath if output is file
	FilePath string `yaml:"filePath,omitempty" json:"filePath,omitempty"`

	// AddSource adds source file info to logs
	AddSource bool `yaml:"addSource" json:"addSource"`
}

// SpanInfo represents trace span information
type SpanInfo struct {
	TraceID    string                 `json:"trace_id"`
	SpanID     string                 `json:"span_id"`
	ParentID   string                 `json:"parent_id,omitempty"`
	Operation  string                 `json:"operation"`
	Service    string                 `json:"service"`
	StartTime  time.Time              `json:"start_time"`
	EndTime    time.Time              `json:"end_time"`
	Duration   time.Duration          `json:"duration"`
	Status     string                 `json:"status"`
	Attributes map[string]interface{} `json:"attributes,omitempty"`
	Events     []SpanEvent            `json:"events,omitempty"`
}

// SpanEvent represents an event within a span
type SpanEvent struct {
	Name       string                 `json:"name"`
	Timestamp  time.Time              `json:"timestamp"`
	Attributes map[string]interface{} `json:"attributes,omitempty"`
}

// MetricData represents a metric data point
type MetricData struct {
	Name       string                 `json:"name"`
	Type       string                 `json:"type"` // counter, gauge, histogram
	Value      float64                `json:"value"`
	Labels     map[string]string      `json:"labels,omitempty"`
	Timestamp  time.Time              `json:"timestamp"`
	Attributes map[string]interface{} `json:"attributes,omitempty"`
}

// TelemetryStats represents collected telemetry statistics
type TelemetryStats struct {
	// Tracing stats
	TracesCollected  int64 `json:"traces_collected"`
	SpansCollected   int64 `json:"spans_collected"`
	TracesExported   int64 `json:"traces_exported"`
	TraceExportErrors int64 `json:"trace_export_errors"`

	// Metrics stats
	MetricsCollected int64 `json:"metrics_collected"`
	MetricsExported  int64 `json:"metrics_exported"`
	MetricExportErrors int64 `json:"metric_export_errors"`

	// General
	StartTime        time.Time `json:"start_time"`
	Uptime           time.Duration `json:"uptime"`
}
