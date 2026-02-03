package audit

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// EventType represents the type of audit event
type EventType string

const (
	EventCreate   EventType = "create"
	EventUpdate   EventType = "update"
	EventDelete   EventType = "delete"
	EventRead     EventType = "read"
	EventApply    EventType = "apply"
	EventPlan     EventType = "plan"
	EventValidate EventType = "validate"
	EventRollback EventType = "rollback"
)

// ResourceType represents the type of resource
type ResourceType string

const (
	ResourcePlatform        ResourceType = "Platform"
	ResourceInfrastructure  ResourceType = "Infrastructure"
	ResourceOrchestrator    ResourceType = "Orchestrator"
	ResourceObservability   ResourceType = "Observability"
	ResourceDevEx           ResourceType = "DevEx"
	ResourcePipeline        ResourceType = "Pipeline"
	ResourceMesh            ResourceType = "Mesh"
	ResourceSecurity        ResourceType = "Security"
	ResourceCompliance      ResourceType = "Compliance"
	ResourceJob             ResourceType = "Job"
	ResourcePlugin          ResourceType = "Plugin"
	ResourceUser            ResourceType = "User"
	ResourceAPI             ResourceType = "API"
	ResourceService         ResourceType = "Service"
	ResourceServiceTemplate ResourceType = "ServiceTemplate"
)

// Event represents an audit log event
type Event struct {
	ID           string                 `json:"id"`
	Timestamp    time.Time              `json:"timestamp"`
	User         string                 `json:"user"`
	EventType    EventType              `json:"event_type"`
	ResourceType ResourceType           `json:"resource_type"`
	ResourceName string                 `json:"resource_name"`
	Action       string                 `json:"action"`
	Status       string                 `json:"status"` // success, failed
	Message      string                 `json:"message,omitempty"`
	Metadata     map[string]interface{} `json:"metadata,omitempty"`
	IPAddress    string                 `json:"ip_address,omitempty"`
	UserAgent    string                 `json:"user_agent,omitempty"`
}

// Logger manages audit logging
type Logger struct {
	destination   string // file path, syslog, etc.
	destType      string // file, syslog, cloud
	file          *os.File
	mu            sync.Mutex
	buffer        []Event
	maxBufferSize int
	retention     time.Duration
}

// Config represents audit logger configuration
type Config struct {
	Destination   string        // File path or endpoint
	DestType      string        // "file", "syslog", "cloud"
	MaxBufferSize int           // Max events to buffer before flush
	Retention     time.Duration // How long to keep audit logs
}

// NewLogger creates a new audit logger
func NewLogger(config Config) (*Logger, error) {
	logger := &Logger{
		destination:   config.Destination,
		destType:      config.DestType,
		buffer:        make([]Event, 0, config.MaxBufferSize),
		maxBufferSize: config.MaxBufferSize,
		retention:     config.Retention,
	}

	if config.MaxBufferSize == 0 {
		logger.maxBufferSize = 100
	}

	if config.Retention == 0 {
		logger.retention = 90 * 24 * time.Hour // 90 days default
	}

	// Initialize destination
	if config.DestType == "file" && config.Destination != "" {
		if err := logger.initFileDestination(); err != nil {
			return nil, err
		}
	}

	return logger, nil
}

// initFileDestination initializes file-based logging
func (l *Logger) initFileDestination() error {
	// Ensure directory exists
	dir := filepath.Dir(l.destination)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create audit log directory: %w", err)
	}

	// Open file in append mode
	file, err := os.OpenFile(l.destination, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("failed to open audit log file: %w", err)
	}

	l.file = file
	return nil
}

// Log logs an audit event
func (l *Logger) Log(event Event) error {
	// Set timestamp if not set
	if event.Timestamp.IsZero() {
		event.Timestamp = time.Now()
	}

	// Generate ID if not set
	if event.ID == "" {
		event.ID = generateEventID()
	}

	l.mu.Lock()
	defer l.mu.Unlock()

	// Add to buffer
	l.buffer = append(l.buffer, event)

	// Flush if buffer is full
	if len(l.buffer) >= l.maxBufferSize {
		return l.flushLocked()
	}

	// For critical events, flush immediately
	if event.EventType == EventDelete || event.Status == "failed" {
		return l.flushLocked()
	}

	return nil
}

// LogCreate logs a create event
func (l *Logger) LogCreate(user string, resourceType ResourceType, resourceName string, status string, message string) error {
	return l.Log(Event{
		User:         user,
		EventType:    EventCreate,
		ResourceType: resourceType,
		ResourceName: resourceName,
		Action:       "create",
		Status:       status,
		Message:      message,
	})
}

// LogUpdate logs an update event
func (l *Logger) LogUpdate(user string, resourceType ResourceType, resourceName string, status string, message string) error {
	return l.Log(Event{
		User:         user,
		EventType:    EventUpdate,
		ResourceType: resourceType,
		ResourceName: resourceName,
		Action:       "update",
		Status:       status,
		Message:      message,
	})
}

// LogDelete logs a delete event
func (l *Logger) LogDelete(user string, resourceType ResourceType, resourceName string, status string, message string) error {
	return l.Log(Event{
		User:         user,
		EventType:    EventDelete,
		ResourceType: resourceType,
		ResourceName: resourceName,
		Action:       "delete",
		Status:       status,
		Message:      message,
	})
}

// LogApply logs an apply event
func (l *Logger) LogApply(user string, resourceType ResourceType, resourceName string, status string, message string, metadata map[string]interface{}) error {
	return l.Log(Event{
		User:         user,
		EventType:    EventApply,
		ResourceType: resourceType,
		ResourceName: resourceName,
		Action:       "apply",
		Status:       status,
		Message:      message,
		Metadata:     metadata,
	})
}

// LogAccess logs an API access event
func (l *Logger) LogAccess(user string, method string, path string, status string) error {
	return l.Log(Event{
		User:         user,
		EventType:    EventRead,
		ResourceType: ResourceAPI,
		ResourceName: path,
		Action:       method,
		Status:       status,
	})
}

// Flush flushes buffered events to the destination
func (l *Logger) Flush() error {
	l.mu.Lock()
	defer l.mu.Unlock()

	return l.flushLocked()
}

// flushLocked flushes buffered events (assumes lock is held)
func (l *Logger) flushLocked() error {
	if len(l.buffer) == 0 {
		return nil
	}

	switch l.destType {
	case "file":
		return l.flushToFile()
	case "syslog":
		return l.flushToSyslog()
	case "cloud":
		return l.flushToCloud()
	default:
		// Default to stdout
		return l.flushToStdout()
	}
}

// flushToFile writes events to file
func (l *Logger) flushToFile() error {
	if l.file == nil {
		return fmt.Errorf("file destination not initialized")
	}

	for _, event := range l.buffer {
		data, err := json.Marshal(event)
		if err != nil {
			return fmt.Errorf("failed to marshal event: %w", err)
		}

		if _, err := l.file.Write(data); err != nil {
			return fmt.Errorf("failed to write event: %w", err)
		}

		if _, err := l.file.Write([]byte("\n")); err != nil {
			return fmt.Errorf("failed to write newline: %w", err)
		}
	}

	// Sync to disk
	if err := l.file.Sync(); err != nil {
		return fmt.Errorf("failed to sync file: %w", err)
	}

	// Clear buffer
	l.buffer = l.buffer[:0]

	return nil
}

// flushToSyslog writes events to syslog (placeholder)
func (l *Logger) flushToSyslog() error {
	// In a real implementation, would integrate with syslog
	// For now, just clear the buffer
	l.buffer = l.buffer[:0]
	return nil
}

// flushToCloud writes events to cloud (placeholder)
func (l *Logger) flushToCloud() error {
	// In a real implementation, would send to cloud logging service
	// For now, just clear the buffer
	l.buffer = l.buffer[:0]
	return nil
}

// flushToStdout writes events to stdout
func (l *Logger) flushToStdout() error {
	for _, event := range l.buffer {
		data, err := json.Marshal(event)
		if err != nil {
			return fmt.Errorf("failed to marshal event: %w", err)
		}

		fmt.Println(string(data))
	}

	l.buffer = l.buffer[:0]
	return nil
}

// Query queries audit logs based on filters
func (l *Logger) Query(filters map[string]interface{}, limit int) ([]Event, error) {
	if l.destType != "file" || l.destination == "" {
		return nil, fmt.Errorf("query only supported for file destination")
	}

	file, err := os.Open(l.destination)
	if err != nil {
		return nil, fmt.Errorf("failed to open audit log: %w", err)
	}
	defer file.Close()

	events := make([]Event, 0)
	decoder := json.NewDecoder(file)

	for {
		var event Event
		if err := decoder.Decode(&event); err == io.EOF {
			break
		} else if err != nil {
			// Skip malformed lines
			continue
		}

		// Apply filters
		if matchesFilters(event, filters) {
			events = append(events, event)

			if limit > 0 && len(events) >= limit {
				break
			}
		}
	}

	return events, nil
}

// matchesFilters checks if an event matches the given filters
func matchesFilters(event Event, filters map[string]interface{}) bool {
	if user, ok := filters["user"].(string); ok && event.User != user {
		return false
	}

	if eventType, ok := filters["event_type"].(EventType); ok && event.EventType != eventType {
		return false
	}

	if resourceType, ok := filters["resource_type"].(ResourceType); ok && event.ResourceType != resourceType {
		return false
	}

	if resourceName, ok := filters["resource_name"].(string); ok && event.ResourceName != resourceName {
		return false
	}

	if status, ok := filters["status"].(string); ok && event.Status != status {
		return false
	}

	// Time range filters
	if after, ok := filters["after"].(time.Time); ok && event.Timestamp.Before(after) {
		return false
	}

	if before, ok := filters["before"].(time.Time); ok && event.Timestamp.After(before) {
		return false
	}

	return true
}

// CleanupOldLogs removes audit logs older than retention period
func (l *Logger) CleanupOldLogs() error {
	if l.destType != "file" || l.destination == "" {
		return nil
	}

	cutoffTime := time.Now().Add(-l.retention)

	// Read all events
	events, err := l.Query(map[string]interface{}{}, 0)
	if err != nil {
		return err
	}

	// Filter events within retention
	validEvents := make([]Event, 0)
	for _, event := range events {
		if event.Timestamp.After(cutoffTime) {
			validEvents = append(validEvents, event)
		}
	}

	// Rewrite file with valid events
	l.mu.Lock()
	defer l.mu.Unlock()

	// Close and reopen file
	if l.file != nil {
		l.file.Close()
	}

	file, err := os.OpenFile(l.destination, os.O_TRUNC|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("failed to reopen audit log: %w", err)
	}

	l.file = file

	// Write valid events
	for _, event := range validEvents {
		data, err := json.Marshal(event)
		if err != nil {
			continue
		}

		l.file.Write(data)
		l.file.Write([]byte("\n"))
	}

	l.file.Sync()

	return nil
}

// Close closes the audit logger
func (l *Logger) Close() error {
	// Flush remaining events
	if err := l.Flush(); err != nil {
		return err
	}

	// Close file if open
	if l.file != nil {
		return l.file.Close()
	}

	return nil
}

// GetStats returns statistics about audit logs
func (l *Logger) GetStats() (map[string]interface{}, error) {
	events, err := l.Query(map[string]interface{}{}, 0)
	if err != nil {
		return nil, err
	}

	stats := map[string]interface{}{
		"total_events": len(events),
		"by_type":      make(map[EventType]int),
		"by_resource":  make(map[ResourceType]int),
		"by_status":    make(map[string]int),
		"by_user":      make(map[string]int),
	}

	byType := stats["by_type"].(map[EventType]int)
	byResource := stats["by_resource"].(map[ResourceType]int)
	byStatus := stats["by_status"].(map[string]int)
	byUser := stats["by_user"].(map[string]int)

	for _, event := range events {
		byType[event.EventType]++
		byResource[event.ResourceType]++
		byStatus[event.Status]++
		byUser[event.User]++
	}

	return stats, nil
}

// generateEventID generates a unique event ID
func generateEventID() string {
	return fmt.Sprintf("evt_%d", time.Now().UnixNano())
}
