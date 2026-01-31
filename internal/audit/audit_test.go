package audit

import (
	"os"
	"testing"
	"time"
)

func TestNewLogger(t *testing.T) {
	config := Config{
		Destination:   "",
		DestType:      "stdout",
		MaxBufferSize: 10,
		Retention:     30 * 24 * time.Hour,
	}

	logger, err := NewLogger(config)
	if err != nil {
		t.Fatalf("NewLogger() error = %v", err)
	}

	if logger == nil {
		t.Fatal("NewLogger() returned nil")
	}

	if logger.maxBufferSize != 10 {
		t.Errorf("Expected maxBufferSize 10, got %d", logger.maxBufferSize)
	}

	if logger.retention != 30*24*time.Hour {
		t.Errorf("Expected retention 30 days, got %v", logger.retention)
	}
}

func TestNewLoggerFileDestination(t *testing.T) {
	tmpDir := os.TempDir()
	logPath := tmpDir + "/audit_test.log"
	defer os.Remove(logPath)

	config := Config{
		Destination:   logPath,
		DestType:      "file",
		MaxBufferSize: 10,
	}

	logger, err := NewLogger(config)
	if err != nil {
		t.Fatalf("NewLogger() error = %v", err)
	}
	defer logger.Close()

	if logger.file == nil {
		t.Error("File should be initialized")
	}
}

func TestLog(t *testing.T) {
	config := Config{
		DestType:      "stdout",
		MaxBufferSize: 10,
	}

	logger, _ := NewLogger(config)

	event := Event{
		User:         "alice",
		EventType:    EventCreate,
		ResourceType: ResourcePlatform,
		ResourceName: "test-platform",
		Action:       "create",
		Status:       "success",
	}

	err := logger.Log(event)
	if err != nil {
		t.Fatalf("Log() error = %v", err)
	}

	if len(logger.buffer) != 1 {
		t.Errorf("Expected 1 event in buffer, got %d", len(logger.buffer))
	}

	// Check that ID and timestamp are set
	bufferedEvent := logger.buffer[0]
	if bufferedEvent.ID == "" {
		t.Error("Event ID should be set")
	}

	if bufferedEvent.Timestamp.IsZero() {
		t.Error("Event timestamp should be set")
	}
}

func TestLogCreate(t *testing.T) {
	config := Config{
		DestType:      "stdout",
		MaxBufferSize: 10,
	}

	logger, _ := NewLogger(config)

	err := logger.LogCreate("alice", ResourcePlatform, "test-platform", "success", "Created successfully")
	if err != nil {
		t.Fatalf("LogCreate() error = %v", err)
	}

	if len(logger.buffer) != 1 {
		t.Error("Should have 1 event in buffer")
	}

	event := logger.buffer[0]
	if event.EventType != EventCreate {
		t.Error("Event type should be create")
	}

	if event.User != "alice" {
		t.Error("User should be alice")
	}
}

func TestLogUpdate(t *testing.T) {
	config := Config{
		DestType:      "stdout",
		MaxBufferSize: 10,
	}

	logger, _ := NewLogger(config)

	err := logger.LogUpdate("bob", ResourceInfrastructure, "test-infra", "success", "Updated successfully")
	if err != nil {
		t.Fatalf("LogUpdate() error = %v", err)
	}

	event := logger.buffer[0]
	if event.EventType != EventUpdate {
		t.Error("Event type should be update")
	}
}

func TestLogDelete(t *testing.T) {
	config := Config{
		DestType:      "stdout",
		MaxBufferSize: 10,
	}

	logger, _ := NewLogger(config)

	err := logger.LogDelete("charlie", ResourceOrchestrator, "test-orch", "success", "Deleted successfully")
	if err != nil {
		t.Fatalf("LogDelete() error = %v", err)
	}

	// Delete events should flush immediately
	if len(logger.buffer) != 0 {
		t.Error("Buffer should be flushed after delete event")
	}
}

func TestLogApply(t *testing.T) {
	config := Config{
		DestType:      "stdout",
		MaxBufferSize: 10,
	}

	logger, _ := NewLogger(config)

	metadata := map[string]interface{}{
		"job_id": "job-123",
		"nodes":  5,
	}

	err := logger.LogApply("alice", ResourcePlatform, "test-platform", "success", "Applied successfully", metadata)
	if err != nil {
		t.Fatalf("LogApply() error = %v", err)
	}

	event := logger.buffer[0]
	if event.EventType != EventApply {
		t.Error("Event type should be apply")
	}

	if event.Metadata == nil {
		t.Error("Metadata should be set")
	}

	if event.Metadata["job_id"] != "job-123" {
		t.Error("Metadata should contain job_id")
	}
}

func TestBufferFlush(t *testing.T) {
	tmpDir := os.TempDir()
	logPath := tmpDir + "/audit_buffer_test.log"
	defer os.Remove(logPath)

	config := Config{
		Destination:   logPath,
		DestType:      "file",
		MaxBufferSize: 3,
	}

	logger, _ := NewLogger(config)
	defer logger.Close()

	// Add events to fill buffer
	logger.LogCreate("alice", ResourcePlatform, "p1", "success", "")
	logger.LogCreate("bob", ResourcePlatform, "p2", "success", "")

	// Buffer should have 2 events
	if len(logger.buffer) != 2 {
		t.Errorf("Expected 2 events in buffer, got %d", len(logger.buffer))
	}

	// Add third event to trigger flush
	logger.LogCreate("charlie", ResourcePlatform, "p3", "success", "")

	// Buffer should be flushed
	if len(logger.buffer) != 0 {
		t.Error("Buffer should be flushed when full")
	}

	// Check file exists and has content
	info, err := os.Stat(logPath)
	if err != nil {
		t.Error("Log file should exist")
	}

	if info.Size() == 0 {
		t.Error("Log file should have content")
	}
}

func TestManualFlush(t *testing.T) {
	tmpDir := os.TempDir()
	logPath := tmpDir + "/audit_manual_flush_test.log"
	defer os.Remove(logPath)

	config := Config{
		Destination:   logPath,
		DestType:      "file",
		MaxBufferSize: 10,
	}

	logger, _ := NewLogger(config)
	defer logger.Close()

	logger.LogCreate("alice", ResourcePlatform, "p1", "success", "")

	// Manual flush
	err := logger.Flush()
	if err != nil {
		t.Fatalf("Flush() error = %v", err)
	}

	if len(logger.buffer) != 0 {
		t.Error("Buffer should be empty after flush")
	}
}

func TestQuery(t *testing.T) {
	tmpDir := os.TempDir()
	logPath := tmpDir + "/audit_query_test.log"
	defer os.Remove(logPath)

	config := Config{
		Destination:   logPath,
		DestType:      "file",
		MaxBufferSize: 10,
	}

	logger, _ := NewLogger(config)
	defer logger.Close()

	// Add some events
	logger.LogCreate("alice", ResourcePlatform, "p1", "success", "")
	logger.LogCreate("bob", ResourceInfrastructure, "i1", "success", "")
	logger.LogUpdate("alice", ResourcePlatform, "p1", "success", "")
	logger.Flush()

	// Query all events
	events, err := logger.Query(map[string]interface{}{}, 0)
	if err != nil {
		t.Fatalf("Query() error = %v", err)
	}

	if len(events) != 3 {
		t.Errorf("Expected 3 events, got %d", len(events))
	}

	// Query by user
	events, err = logger.Query(map[string]interface{}{
		"user": "alice",
	}, 0)
	if err != nil {
		t.Fatalf("Query() error = %v", err)
	}

	if len(events) != 2 {
		t.Errorf("Expected 2 events for alice, got %d", len(events))
	}

	// Query by event type
	events, err = logger.Query(map[string]interface{}{
		"event_type": EventCreate,
	}, 0)
	if err != nil {
		t.Fatalf("Query() error = %v", err)
	}

	if len(events) != 2 {
		t.Errorf("Expected 2 create events, got %d", len(events))
	}

	// Query with limit
	events, err = logger.Query(map[string]interface{}{}, 2)
	if err != nil {
		t.Fatalf("Query() error = %v", err)
	}

	if len(events) != 2 {
		t.Errorf("Expected 2 events with limit, got %d", len(events))
	}
}

func TestQueryByResourceType(t *testing.T) {
	tmpDir := os.TempDir()
	logPath := tmpDir + "/audit_query_resource_test.log"
	defer os.Remove(logPath)

	config := Config{
		Destination:   logPath,
		DestType:      "file",
		MaxBufferSize: 10,
	}

	logger, _ := NewLogger(config)
	defer logger.Close()

	logger.LogCreate("alice", ResourcePlatform, "p1", "success", "")
	logger.LogCreate("alice", ResourceInfrastructure, "i1", "success", "")
	logger.Flush()

	events, err := logger.Query(map[string]interface{}{
		"resource_type": ResourcePlatform,
	}, 0)
	if err != nil {
		t.Fatalf("Query() error = %v", err)
	}

	if len(events) != 1 {
		t.Errorf("Expected 1 platform event, got %d", len(events))
	}

	if events[0].ResourceType != ResourcePlatform {
		t.Error("Event should be for Platform resource")
	}
}

func TestQueryByStatus(t *testing.T) {
	tmpDir := os.TempDir()
	logPath := tmpDir + "/audit_query_status_test.log"
	defer os.Remove(logPath)

	config := Config{
		Destination:   logPath,
		DestType:      "file",
		MaxBufferSize: 10,
	}

	logger, _ := NewLogger(config)
	defer logger.Close()

	logger.LogCreate("alice", ResourcePlatform, "p1", "success", "")
	logger.LogCreate("bob", ResourcePlatform, "p2", "failed", "Error")
	logger.Flush()

	events, err := logger.Query(map[string]interface{}{
		"status": "failed",
	}, 0)
	if err != nil {
		t.Fatalf("Query() error = %v", err)
	}

	if len(events) != 1 {
		t.Errorf("Expected 1 failed event, got %d", len(events))
	}

	if events[0].Status != "failed" {
		t.Error("Event should have failed status")
	}
}

func TestGetStats(t *testing.T) {
	tmpDir := os.TempDir()
	logPath := tmpDir + "/audit_stats_test.log"
	defer os.Remove(logPath)

	config := Config{
		Destination:   logPath,
		DestType:      "file",
		MaxBufferSize: 10,
	}

	logger, _ := NewLogger(config)
	defer logger.Close()

	logger.LogCreate("alice", ResourcePlatform, "p1", "success", "")
	logger.LogCreate("bob", ResourceInfrastructure, "i1", "success", "")
	logger.LogUpdate("alice", ResourcePlatform, "p1", "success", "")
	logger.LogDelete("charlie", ResourceOrchestrator, "o1", "failed", "Error")
	logger.Flush()

	stats, err := logger.GetStats()
	if err != nil {
		t.Fatalf("GetStats() error = %v", err)
	}

	totalEvents := stats["total_events"].(int)
	if totalEvents != 4 {
		t.Errorf("Expected 4 total events, got %d", totalEvents)
	}

	byType := stats["by_type"].(map[EventType]int)
	if byType[EventCreate] != 2 {
		t.Errorf("Expected 2 create events, got %d", byType[EventCreate])
	}

	byUser := stats["by_user"].(map[string]int)
	if byUser["alice"] != 2 {
		t.Errorf("Expected 2 events for alice, got %d", byUser["alice"])
	}

	byStatus := stats["by_status"].(map[string]int)
	if byStatus["success"] != 3 {
		t.Errorf("Expected 3 success events, got %d", byStatus["success"])
	}

	if byStatus["failed"] != 1 {
		t.Errorf("Expected 1 failed event, got %d", byStatus["failed"])
	}
}

func TestCleanupOldLogs(t *testing.T) {
	tmpDir := os.TempDir()
	logPath := tmpDir + "/audit_cleanup_test.log"
	defer os.Remove(logPath)

	config := Config{
		Destination:   logPath,
		DestType:      "file",
		MaxBufferSize: 10,
		Retention:     1 * time.Hour,
	}

	logger, _ := NewLogger(config)
	defer logger.Close()

	// Create an old event
	oldEvent := Event{
		ID:           "old-1",
		User:         "alice",
		EventType:    EventCreate,
		ResourceType: ResourcePlatform,
		ResourceName: "old",
		Status:       "success",
		Timestamp:    time.Now().Add(-2 * time.Hour),
	}
	logger.Log(oldEvent)

	// Create a recent event
	logger.LogCreate("bob", ResourcePlatform, "new", "success", "")
	logger.Flush()

	// Cleanup old logs
	err := logger.CleanupOldLogs()
	if err != nil {
		t.Fatalf("CleanupOldLogs() error = %v", err)
	}

	// Query events
	events, _ := logger.Query(map[string]interface{}{}, 0)

	if len(events) != 1 {
		t.Errorf("Expected 1 event after cleanup, got %d", len(events))
	}

	if events[0].ResourceName != "new" {
		t.Error("Only recent event should remain")
	}
}
