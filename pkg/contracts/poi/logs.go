// Package poi defines the Platform Observability Interface (POI).
package poi

import (
	"context"
	"time"
)

// LogExporter exports logs to an observability backend
type LogExporter interface {
	// Export sends logs to the backend
	Export(ctx context.Context, logs []LogRecord) error

	// Flush forces any buffered logs to be sent
	Flush(ctx context.Context) error

	// Close releases exporter resources
	Close() error
}

// LogRecord represents a single log entry
type LogRecord struct {
	// Timestamp is when the log was recorded
	Timestamp time.Time

	// Level is the log severity level
	Level LogLevel

	// Message is the log message
	Message string

	// Attributes contains structured log attributes
	Attributes map[string]interface{}

	// Resource identifies the source of the log
	Resource ResourceInfo

	// TraceID links this log to a trace (optional)
	TraceID string

	// SpanID links this log to a span (optional)
	SpanID string
}

// LogLevel represents log severity
type LogLevel string

const (
	LogLevelTrace LogLevel = "trace"
	LogLevelDebug LogLevel = "debug"
	LogLevelInfo  LogLevel = "info"
	LogLevelWarn  LogLevel = "warn"
	LogLevelError LogLevel = "error"
	LogLevelFatal LogLevel = "fatal"
)

// ResourceInfo identifies the source of telemetry data
type ResourceInfo struct {
	// ServiceName is the name of the service
	ServiceName string

	// ServiceVersion is the version of the service
	ServiceVersion string

	// ServiceInstance is the instance identifier
	ServiceInstance string

	// Environment is the deployment environment
	Environment string

	// Attributes contains additional resource attributes
	Attributes map[string]string
}

// Logger is the main logging interface
type Logger interface {
	// Log logs a message at the given level
	Log(level LogLevel, msg string, attrs ...interface{})

	// Trace logs a trace message
	Trace(msg string, attrs ...interface{})

	// Debug logs a debug message
	Debug(msg string, attrs ...interface{})

	// Info logs an info message
	Info(msg string, attrs ...interface{})

	// Warn logs a warning message
	Warn(msg string, attrs ...interface{})

	// Error logs an error message
	Error(msg string, attrs ...interface{})

	// Fatal logs a fatal message
	Fatal(msg string, attrs ...interface{})

	// With returns a logger with additional attributes
	With(attrs ...interface{}) Logger

	// WithContext returns a logger with context
	WithContext(ctx context.Context) Logger
}

// LogQuery represents a query for logs
type LogQuery struct {
	// Query is the search query string
	Query string

	// Start is the start of the query time range
	Start time.Time

	// End is the end of the query time range
	End time.Time

	// Levels filters by log level
	Levels []LogLevel

	// Services filters by service name
	Services []string

	// Limit is the maximum number of results
	Limit int

	// Offset is the result offset for pagination
	Offset int
}

// LogReader reads logs from a backend
type LogReader interface {
	// Query executes a log query
	Query(ctx context.Context, query *LogQuery) ([]LogRecord, error)

	// Tail streams logs in real-time
	Tail(ctx context.Context, query *LogQuery) (<-chan LogRecord, error)
}
