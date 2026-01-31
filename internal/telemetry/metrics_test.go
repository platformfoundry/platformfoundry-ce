package telemetry

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestNewMetricsCollector(t *testing.T) {
	mc := NewMetricsCollector()
	if mc == nil {
		t.Fatal("NewMetricsCollector() returned nil")
	}

	if mc.counters == nil {
		t.Error("counters should be initialized")
	}

	if mc.gauges == nil {
		t.Error("gauges should be initialized")
	}

	if mc.histograms == nil {
		t.Error("histograms should be initialized")
	}

	// Check default metrics are registered
	if _, exists := mc.counters["pf_errors_total"]; !exists {
		t.Error("Default counter pf_errors_total should be registered")
	}

	if _, exists := mc.gauges["pf_resources_total"]; !exists {
		t.Error("Default gauge pf_resources_total should be registered")
	}

	if _, exists := mc.histograms["pf_apply_duration_seconds"]; !exists {
		t.Error("Default histogram pf_apply_duration_seconds should be registered")
	}
}

func TestRegisterCounter(t *testing.T) {
	mc := NewMetricsCollector()

	mc.RegisterCounter("test_counter", "Test counter")

	if _, exists := mc.counters["test_counter"]; !exists {
		t.Error("Counter should be registered")
	}
}

func TestRegisterGauge(t *testing.T) {
	mc := NewMetricsCollector()

	mc.RegisterGauge("test_gauge", "Test gauge")

	if _, exists := mc.gauges["test_gauge"]; !exists {
		t.Error("Gauge should be registered")
	}
}

func TestRegisterHistogram(t *testing.T) {
	mc := NewMetricsCollector()

	mc.RegisterHistogram("test_histogram", "Test histogram", []float64{1, 5, 10})

	if _, exists := mc.histograms["test_histogram"]; !exists {
		t.Error("Histogram should be registered")
	}

	histogram := mc.histograms["test_histogram"]
	if len(histogram.buckets) != 3 {
		t.Errorf("Expected 3 buckets, got %d", len(histogram.buckets))
	}
}

func TestIncrementCounter(t *testing.T) {
	mc := NewMetricsCollector()
	mc.RegisterCounter("test_counter", "Test counter")

	mc.IncrementCounter("test_counter", nil)
	mc.IncrementCounter("test_counter", nil)

	counter := mc.counters["test_counter"]
	if counter.value != 2 {
		t.Errorf("Expected counter value 2, got %.0f", counter.value)
	}
}

func TestAddToCounter(t *testing.T) {
	mc := NewMetricsCollector()
	mc.RegisterCounter("test_counter", "Test counter")

	mc.AddToCounter("test_counter", 5, nil)
	mc.AddToCounter("test_counter", 3, nil)

	counter := mc.counters["test_counter"]
	if counter.value != 8 {
		t.Errorf("Expected counter value 8, got %.0f", counter.value)
	}
}

func TestSetGauge(t *testing.T) {
	mc := NewMetricsCollector()
	mc.RegisterGauge("test_gauge", "Test gauge")

	mc.SetGauge("test_gauge", 42, nil)

	gauge := mc.gauges["test_gauge"]
	if gauge.value != 42 {
		t.Errorf("Expected gauge value 42, got %.0f", gauge.value)
	}

	mc.SetGauge("test_gauge", 10, nil)

	if gauge.value != 10 {
		t.Errorf("Expected gauge value 10, got %.0f", gauge.value)
	}
}

func TestIncrementGauge(t *testing.T) {
	mc := NewMetricsCollector()
	mc.RegisterGauge("test_gauge", "Test gauge")

	mc.IncrementGauge("test_gauge", 5)
	mc.IncrementGauge("test_gauge", 3)

	gauge := mc.gauges["test_gauge"]
	if gauge.value != 8 {
		t.Errorf("Expected gauge value 8, got %.0f", gauge.value)
	}
}

func TestDecrementGauge(t *testing.T) {
	mc := NewMetricsCollector()
	mc.RegisterGauge("test_gauge", "Test gauge")

	mc.SetGauge("test_gauge", 10, nil)
	mc.DecrementGauge("test_gauge", 3)

	gauge := mc.gauges["test_gauge"]
	if gauge.value != 7 {
		t.Errorf("Expected gauge value 7, got %.0f", gauge.value)
	}
}

func TestObserveHistogram(t *testing.T) {
	mc := NewMetricsCollector()
	mc.RegisterHistogram("test_histogram", "Test histogram", []float64{1, 5, 10})

	mc.ObserveHistogram("test_histogram", 3, nil)
	mc.ObserveHistogram("test_histogram", 7, nil)
	mc.ObserveHistogram("test_histogram", 12, nil)

	histogram := mc.histograms["test_histogram"]

	if histogram.count != 3 {
		t.Errorf("Expected count 3, got %d", histogram.count)
	}

	if histogram.sum != 22 {
		t.Errorf("Expected sum 22, got %.0f", histogram.sum)
	}

	// Check bucket counts
	// Value 3 falls in buckets: 5, 10, +Inf
	// Value 7 falls in buckets: 10, +Inf
	// Value 12 falls in buckets: +Inf
	// So counts should be: [0, 1, 2, 3]
	expectedCounts := []int64{0, 1, 2, 3}
	for i, expected := range expectedCounts {
		if histogram.counts[i] != expected {
			t.Errorf("Bucket %d: expected count %d, got %d", i, expected, histogram.counts[i])
		}
	}
}

func TestTimeDuration(t *testing.T) {
	mc := NewMetricsCollector()
	mc.RegisterHistogram("test_duration", "Test duration", []float64{0.1, 0.5, 1.0})

	mc.TimeDuration("test_duration", nil, func() {
		// Simulate some work
	})

	histogram := mc.histograms["test_duration"]

	if histogram.count != 1 {
		t.Errorf("Expected count 1, got %d", histogram.count)
	}
}

func TestGetMetrics(t *testing.T) {
	mc := NewMetricsCollector()

	mc.RegisterCounter("test_counter", "Test counter")
	mc.IncrementCounter("test_counter", nil)

	mc.RegisterGauge("test_gauge", "Test gauge")
	mc.SetGauge("test_gauge", 42, nil)

	mc.RegisterHistogram("test_histogram", "Test histogram", []float64{1, 5})
	mc.ObserveHistogram("test_histogram", 3, nil)

	output := mc.GetMetrics()

	// Check counter
	if !strings.Contains(output, "# TYPE test_counter counter") {
		t.Error("Metrics should contain counter type")
	}

	if !strings.Contains(output, "test_counter 1") {
		t.Error("Metrics should contain counter value")
	}

	// Check gauge
	if !strings.Contains(output, "# TYPE test_gauge gauge") {
		t.Error("Metrics should contain gauge type")
	}

	if !strings.Contains(output, "test_gauge 42") {
		t.Error("Metrics should contain gauge value")
	}

	// Check histogram
	if !strings.Contains(output, "# TYPE test_histogram histogram") {
		t.Error("Metrics should contain histogram type")
	}

	if !strings.Contains(output, "test_histogram_bucket") {
		t.Error("Metrics should contain histogram buckets")
	}

	if !strings.Contains(output, "test_histogram_sum") {
		t.Error("Metrics should contain histogram sum")
	}

	if !strings.Contains(output, "test_histogram_count") {
		t.Error("Metrics should contain histogram count")
	}
}

func TestGetMetricsWithLabels(t *testing.T) {
	mc := NewMetricsCollector()

	mc.RegisterCounter("test_counter", "Test counter")
	mc.IncrementCounter("test_counter", map[string]string{
		"status": "success",
		"type":   "platform",
	})

	output := mc.GetMetrics()

	if !strings.Contains(output, "test_counter{") {
		t.Error("Metrics should contain labels")
	}

	if !strings.Contains(output, "status=\"success\"") {
		t.Error("Metrics should contain status label")
	}

	if !strings.Contains(output, "type=\"platform\"") {
		t.Error("Metrics should contain type label")
	}
}

func TestMetricsHandler(t *testing.T) {
	mc := NewMetricsCollector()

	mc.RegisterCounter("test_counter", "Test counter")
	mc.IncrementCounter("test_counter", nil)

	req := httptest.NewRequest("GET", "/metrics", nil)
	w := httptest.NewRecorder()

	mc.Handler()(w, req)

	resp := w.Result()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}

	contentType := resp.Header.Get("Content-Type")
	if !strings.Contains(contentType, "text/plain") {
		t.Errorf("Expected content type text/plain, got %s", contentType)
	}
}

func TestRecordApplyDuration(t *testing.T) {
	mc := NewMetricsCollector()

	mc.RecordApplyDuration("Platform", 5.5, "success")

	histogram := mc.histograms["pf_apply_duration_seconds"]
	if histogram.count != 1 {
		t.Error("Apply duration should be recorded")
	}

	if histogram.sum != 5.5 {
		t.Errorf("Expected sum 5.5, got %.1f", histogram.sum)
	}
}

func TestRecordPluginExecution(t *testing.T) {
	mc := NewMetricsCollector()

	mc.RecordPluginExecution("terraform", 12.3, "success")

	histogram := mc.histograms["pf_plugin_execution_seconds"]
	if histogram.count != 1 {
		t.Error("Plugin execution should be recorded")
	}

	if histogram.sum != 12.3 {
		t.Errorf("Expected sum 12.3, got %.1f", histogram.sum)
	}
}

func TestRecordError(t *testing.T) {
	mc := NewMetricsCollector()

	mc.RecordError("validation")
	mc.RecordError("plugin")

	counter := mc.counters["pf_errors_total"]
	if counter.value != 2 {
		t.Errorf("Expected 2 errors, got %.0f", counter.value)
	}
}

func TestRecordResourceOperation(t *testing.T) {
	mc := NewMetricsCollector()

	mc.RecordResourceOperation("Platform", "create")
	mc.RecordResourceOperation("Platform", "update")

	counter := mc.counters["pf_resource_operations_total"]
	if counter.value != 2 {
		t.Errorf("Expected 2 operations, got %.0f", counter.value)
	}
}

func TestUpdateActiveJobs(t *testing.T) {
	mc := NewMetricsCollector()

	mc.UpdateActiveJobs(5)

	gauge := mc.gauges["pf_jobs_active"]
	if gauge.value != 5 {
		t.Errorf("Expected 5 active jobs, got %.0f", gauge.value)
	}

	mc.UpdateActiveJobs(3)

	if gauge.value != 3 {
		t.Errorf("Expected 3 active jobs, got %.0f", gauge.value)
	}
}

func TestRecordJobCompleted(t *testing.T) {
	mc := NewMetricsCollector()

	mc.RecordJobCompleted("success")
	mc.RecordJobCompleted("success")

	successCounter := mc.counters["pf_jobs_completed_total"]
	if successCounter.value != 2 {
		t.Errorf("Expected 2 completed jobs, got %.0f", successCounter.value)
	}

	mc.RecordJobCompleted("failed")

	failedCounter := mc.counters["pf_jobs_failed_total"]
	if failedCounter.value != 1 {
		t.Errorf("Expected 1 failed job, got %.0f", failedCounter.value)
	}
}

func TestFormatLabels(t *testing.T) {
	tests := []struct {
		name     string
		labels   map[string]string
		expected string
	}{
		{
			name:     "empty labels",
			labels:   map[string]string{},
			expected: "",
		},
		{
			name: "single label",
			labels: map[string]string{
				"status": "success",
			},
			expected: "{status=\"success\"}",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := formatLabels(tt.labels)
			if tt.name == "empty labels" && result != tt.expected {
				t.Errorf("Expected '%s', got '%s'", tt.expected, result)
			}
			if tt.name == "single label" && !strings.Contains(result, "status=\"success\"") {
				t.Errorf("Expected label in result, got '%s'", result)
			}
		})
	}
}
