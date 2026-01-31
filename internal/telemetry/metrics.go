package telemetry

import (
	"fmt"
	"net/http"
	"sync"
	"time"
)

// MetricType represents the type of metric
type MetricType string

const (
	MetricCounter   MetricType = "counter"
	MetricGauge     MetricType = "gauge"
	MetricHistogram MetricType = "histogram"
)

// Metric represents a Prometheus-style metric
type Metric struct {
	Name   string
	Type   MetricType
	Help   string
	Labels map[string]string
	Value  float64
}

// Histogram represents a histogram metric with buckets
type Histogram struct {
	Name    string
	Help    string
	Labels  map[string]string
	Buckets []float64
	Counts  []int64
	Sum     float64
	Count   int64
}

// MetricsCollector collects and exposes Prometheus metrics
type MetricsCollector struct {
	counters   map[string]*Counter
	gauges     map[string]*Gauge
	histograms map[string]*HistogramMetric
	mu         sync.RWMutex
}

// Counter represents a counter metric
type Counter struct {
	name   string
	help   string
	labels map[string]string
	value  float64
	mu     sync.Mutex
}

// Gauge represents a gauge metric
type Gauge struct {
	name   string
	help   string
	labels map[string]string
	value  float64
	mu     sync.Mutex
}

// HistogramMetric represents a histogram metric
type HistogramMetric struct {
	name    string
	help    string
	labels  map[string]string
	buckets []float64
	counts  []int64
	sum     float64
	count   int64
	mu      sync.Mutex
}

// NewMetricsCollector creates a new metrics collector
func NewMetricsCollector() *MetricsCollector {
	mc := &MetricsCollector{
		counters:   make(map[string]*Counter),
		gauges:     make(map[string]*Gauge),
		histograms: make(map[string]*HistogramMetric),
	}

	// Register default metrics
	mc.registerDefaultMetrics()

	return mc
}

// registerDefaultMetrics registers the default Platform Foundry metrics
func (mc *MetricsCollector) registerDefaultMetrics() {
	// Apply duration histogram
	mc.RegisterHistogram("pf_apply_duration_seconds",
		"Time taken to apply resources",
		[]float64{1, 5, 10, 30, 60, 120, 300})

	// Plugin execution duration histogram
	mc.RegisterHistogram("pf_plugin_execution_seconds",
		"Time taken to execute plugins",
		[]float64{1, 5, 10, 30, 60, 120, 300})

	// Resources total gauge
	mc.RegisterGauge("pf_resources_total", "Total number of resources")

	// Errors total counter
	mc.RegisterCounter("pf_errors_total", "Total number of errors")

	// Jobs active gauge
	mc.RegisterGauge("pf_jobs_active", "Number of active jobs")

	// Jobs completed counter
	mc.RegisterCounter("pf_jobs_completed_total", "Total number of completed jobs")

	// Jobs failed counter
	mc.RegisterCounter("pf_jobs_failed_total", "Total number of failed jobs")

	// API requests counter
	mc.RegisterCounter("pf_api_requests_total", "Total number of API requests")

	// Resource operations counter
	mc.RegisterCounter("pf_resource_operations_total", "Total number of resource operations")
}

// RegisterCounter registers a new counter metric
func (mc *MetricsCollector) RegisterCounter(name, help string) {
	mc.mu.Lock()
	defer mc.mu.Unlock()

	if _, exists := mc.counters[name]; !exists {
		mc.counters[name] = &Counter{
			name:   name,
			help:   help,
			labels: make(map[string]string),
		}
	}
}

// RegisterGauge registers a new gauge metric
func (mc *MetricsCollector) RegisterGauge(name, help string) {
	mc.mu.Lock()
	defer mc.mu.Unlock()

	if _, exists := mc.gauges[name]; !exists {
		mc.gauges[name] = &Gauge{
			name:   name,
			help:   help,
			labels: make(map[string]string),
		}
	}
}

// RegisterHistogram registers a new histogram metric
func (mc *MetricsCollector) RegisterHistogram(name, help string, buckets []float64) {
	mc.mu.Lock()
	defer mc.mu.Unlock()

	if _, exists := mc.histograms[name]; !exists {
		mc.histograms[name] = &HistogramMetric{
			name:    name,
			help:    help,
			labels:  make(map[string]string),
			buckets: buckets,
			counts:  make([]int64, len(buckets)+1),
		}
	}
}

// IncrementCounter increments a counter by 1
func (mc *MetricsCollector) IncrementCounter(name string, labels map[string]string) {
	mc.mu.RLock()
	counter, exists := mc.counters[name]
	mc.mu.RUnlock()

	if !exists {
		return
	}

	counter.mu.Lock()
	counter.value++
	if labels != nil {
		counter.labels = labels
	}
	counter.mu.Unlock()
}

// AddToCounter adds a value to a counter
func (mc *MetricsCollector) AddToCounter(name string, value float64, labels map[string]string) {
	mc.mu.RLock()
	counter, exists := mc.counters[name]
	mc.mu.RUnlock()

	if !exists {
		return
	}

	counter.mu.Lock()
	counter.value += value
	if labels != nil {
		counter.labels = labels
	}
	counter.mu.Unlock()
}

// SetGauge sets a gauge to a specific value
func (mc *MetricsCollector) SetGauge(name string, value float64, labels map[string]string) {
	mc.mu.RLock()
	gauge, exists := mc.gauges[name]
	mc.mu.RUnlock()

	if !exists {
		return
	}

	gauge.mu.Lock()
	gauge.value = value
	if labels != nil {
		gauge.labels = labels
	}
	gauge.mu.Unlock()
}

// IncrementGauge increments a gauge by a value
func (mc *MetricsCollector) IncrementGauge(name string, value float64) {
	mc.mu.RLock()
	gauge, exists := mc.gauges[name]
	mc.mu.RUnlock()

	if !exists {
		return
	}

	gauge.mu.Lock()
	gauge.value += value
	gauge.mu.Unlock()
}

// DecrementGauge decrements a gauge by a value
func (mc *MetricsCollector) DecrementGauge(name string, value float64) {
	mc.mu.RLock()
	gauge, exists := mc.gauges[name]
	mc.mu.RUnlock()

	if !exists {
		return
	}

	gauge.mu.Lock()
	gauge.value -= value
	gauge.mu.Unlock()
}

// ObserveHistogram records a value in a histogram
func (mc *MetricsCollector) ObserveHistogram(name string, value float64, labels map[string]string) {
	mc.mu.RLock()
	histogram, exists := mc.histograms[name]
	mc.mu.RUnlock()

	if !exists {
		return
	}

	histogram.mu.Lock()
	defer histogram.mu.Unlock()

	// Update sum and count
	histogram.sum += value
	histogram.count++

	// Update bucket counts
	for i, bucket := range histogram.buckets {
		if value <= bucket {
			histogram.counts[i]++
		}
	}
	// Count for +Inf bucket
	histogram.counts[len(histogram.buckets)]++

	if labels != nil {
		histogram.labels = labels
	}
}

// TimeDuration measures and records duration for a histogram
func (mc *MetricsCollector) TimeDuration(name string, labels map[string]string, fn func()) {
	start := time.Now()
	fn()
	duration := time.Since(start).Seconds()
	mc.ObserveHistogram(name, duration, labels)
}

// GetMetrics returns all metrics in Prometheus exposition format
func (mc *MetricsCollector) GetMetrics() string {
	mc.mu.RLock()
	defer mc.mu.RUnlock()

	output := ""

	// Export counters
	for _, counter := range mc.counters {
		counter.mu.Lock()
		output += fmt.Sprintf("# HELP %s %s\n", counter.name, counter.help)
		output += fmt.Sprintf("# TYPE %s counter\n", counter.name)
		if len(counter.labels) > 0 {
			output += fmt.Sprintf("%s%s %.0f\n", counter.name, formatLabels(counter.labels), counter.value)
		} else {
			output += fmt.Sprintf("%s %.0f\n", counter.name, counter.value)
		}
		counter.mu.Unlock()
	}

	// Export gauges
	for _, gauge := range mc.gauges {
		gauge.mu.Lock()
		output += fmt.Sprintf("# HELP %s %s\n", gauge.name, gauge.help)
		output += fmt.Sprintf("# TYPE %s gauge\n", gauge.name)
		if len(gauge.labels) > 0 {
			output += fmt.Sprintf("%s%s %.0f\n", gauge.name, formatLabels(gauge.labels), gauge.value)
		} else {
			output += fmt.Sprintf("%s %.0f\n", gauge.name, gauge.value)
		}
		gauge.mu.Unlock()
	}

	// Export histograms
	for _, histogram := range mc.histograms {
		histogram.mu.Lock()
		output += fmt.Sprintf("# HELP %s %s\n", histogram.name, histogram.help)
		output += fmt.Sprintf("# TYPE %s histogram\n", histogram.name)

		labels := formatLabels(histogram.labels)

		// Bucket counts
		for i, bucket := range histogram.buckets {
			bucketLabel := fmt.Sprintf("{le=\"%.1f\"}", bucket)
			if labels != "" {
				bucketLabel = fmt.Sprintf("{le=\"%.1f\",%s}", bucket, labels[1:])
			}
			output += fmt.Sprintf("%s_bucket%s %d\n", histogram.name, bucketLabel, histogram.counts[i])
		}

		// +Inf bucket
		infLabel := "{le=\"+Inf\"}"
		if labels != "" {
			infLabel = fmt.Sprintf("{le=\"+Inf\",%s}", labels[1:])
		}
		output += fmt.Sprintf("%s_bucket%s %d\n", histogram.name, infLabel, histogram.counts[len(histogram.buckets)])

		// Sum and count
		output += fmt.Sprintf("%s_sum%s %.2f\n", histogram.name, labels, histogram.sum)
		output += fmt.Sprintf("%s_count%s %d\n", histogram.name, labels, histogram.count)
		histogram.mu.Unlock()
	}

	return output
}

// formatLabels formats labels for Prometheus exposition format
func formatLabels(labels map[string]string) string {
	if len(labels) == 0 {
		return ""
	}

	result := "{"
	first := true
	for k, v := range labels {
		if !first {
			result += ","
		}
		result += fmt.Sprintf("%s=\"%s\"", k, v)
		first = false
	}
	result += "}"

	return result
}

// Handler returns an HTTP handler for the /metrics endpoint
func (mc *MetricsCollector) Handler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain; version=0.0.4")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(mc.GetMetrics()))
	}
}

// StartMetricsServer starts an HTTP server to expose metrics
func (mc *MetricsCollector) StartMetricsServer(port int) error {
	http.HandleFunc("/metrics", mc.Handler())

	addr := fmt.Sprintf(":%d", port)
	return http.ListenAndServe(addr, nil)
}

// RecordApplyDuration records the duration of an apply operation
func (mc *MetricsCollector) RecordApplyDuration(resourceType string, duration float64, status string) {
	mc.ObserveHistogram("pf_apply_duration_seconds", duration, map[string]string{
		"resource_type": resourceType,
		"status":        status,
	})
}

// RecordPluginExecution records the duration of a plugin execution
func (mc *MetricsCollector) RecordPluginExecution(pluginName string, duration float64, status string) {
	mc.ObserveHistogram("pf_plugin_execution_seconds", duration, map[string]string{
		"plugin": pluginName,
		"status": status,
	})
}

// RecordError increments the error counter
func (mc *MetricsCollector) RecordError(errorType string) {
	mc.IncrementCounter("pf_errors_total", map[string]string{
		"type": errorType,
	})
}

// RecordResourceOperation records a resource operation
func (mc *MetricsCollector) RecordResourceOperation(resourceType, operation string) {
	mc.IncrementCounter("pf_resource_operations_total", map[string]string{
		"resource_type": resourceType,
		"operation":     operation,
	})
}

// UpdateActiveJobs updates the active jobs gauge
func (mc *MetricsCollector) UpdateActiveJobs(count float64) {
	mc.SetGauge("pf_jobs_active", count, nil)
}

// RecordJobCompleted increments the completed jobs counter
func (mc *MetricsCollector) RecordJobCompleted(status string) {
	if status == "success" {
		mc.IncrementCounter("pf_jobs_completed_total", map[string]string{
			"status": status,
		})
	} else {
		mc.IncrementCounter("pf_jobs_failed_total", map[string]string{
			"status": status,
		})
	}
}
