// Package poi defines the Platform Observability Interface (POI).
// This handles metrics, logs, and traces abstraction.
package poi

import (
	"context"
	"time"
)

// MetricsExporter exports metrics to an observability backend
type MetricsExporter interface {
	// Export sends metrics to the backend
	Export(ctx context.Context, metrics []Metric) error

	// Flush forces any buffered metrics to be sent
	Flush(ctx context.Context) error

	// Close releases exporter resources
	Close() error
}

// Metric represents a single metric data point
type Metric struct {
	// Name is the metric name
	Name string

	// Description describes what this metric measures
	Description string

	// Type is the metric type
	Type MetricType

	// Value is the metric value
	Value float64

	// Timestamp is when the metric was recorded
	Timestamp time.Time

	// Labels are key-value pairs for metric dimensions
	Labels map[string]string

	// Unit is the unit of measurement
	Unit string
}

// MetricType represents the type of metric
type MetricType string

const (
	MetricTypeCounter   MetricType = "counter"
	MetricTypeGauge     MetricType = "gauge"
	MetricTypeHistogram MetricType = "histogram"
	MetricTypeSummary   MetricType = "summary"
)

// MetricsCollector collects metrics during operations
type MetricsCollector interface {
	// Counter creates or gets a counter metric
	Counter(name, description string, labels ...string) Counter

	// Gauge creates or gets a gauge metric
	Gauge(name, description string, labels ...string) Gauge

	// Histogram creates or gets a histogram metric
	Histogram(name, description string, buckets []float64, labels ...string) Histogram
}

// Counter is a metric that only increases
type Counter interface {
	// Inc increments the counter by 1
	Inc(labels ...string)

	// Add adds the given value to the counter
	Add(value float64, labels ...string)
}

// Gauge is a metric that can increase and decrease
type Gauge interface {
	// Set sets the gauge to the given value
	Set(value float64, labels ...string)

	// Inc increments the gauge by 1
	Inc(labels ...string)

	// Dec decrements the gauge by 1
	Dec(labels ...string)

	// Add adds the given value to the gauge
	Add(value float64, labels ...string)

	// Sub subtracts the given value from the gauge
	Sub(value float64, labels ...string)
}

// Histogram records observations in buckets
type Histogram interface {
	// Observe records an observation
	Observe(value float64, labels ...string)
}

// MetricsQuery represents a query for metrics
type MetricsQuery struct {
	// Name is the metric name pattern (supports wildcards)
	Name string

	// Labels filters by label values
	Labels map[string]string

	// Start is the start of the query time range
	Start time.Time

	// End is the end of the query time range
	End time.Time

	// Step is the query resolution
	Step time.Duration

	// Aggregation specifies how to aggregate results
	Aggregation AggregationType
}

// AggregationType specifies how metrics are aggregated
type AggregationType string

const (
	AggregationSum   AggregationType = "sum"
	AggregationAvg   AggregationType = "avg"
	AggregationMin   AggregationType = "min"
	AggregationMax   AggregationType = "max"
	AggregationCount AggregationType = "count"
)

// MetricsReader reads metrics from a backend
type MetricsReader interface {
	// Query executes a metrics query
	Query(ctx context.Context, query *MetricsQuery) ([]MetricSeries, error)
}

// MetricSeries represents a time series of metric values
type MetricSeries struct {
	// Name is the metric name
	Name string

	// Labels are the series labels
	Labels map[string]string

	// Points are the data points
	Points []MetricPoint
}

// MetricPoint represents a single point in a time series
type MetricPoint struct {
	// Timestamp is when the point was recorded
	Timestamp time.Time

	// Value is the metric value
	Value float64
}
