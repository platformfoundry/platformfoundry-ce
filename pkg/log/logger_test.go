package log

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
	"time"
)

func TestNewLogger(t *testing.T) {
	var buf bytes.Buffer
	logger := New(Config{
		Level:  LevelInfo,
		Format: FormatText,
		Output: &buf,
	})

	if logger == nil {
		t.Fatal("New returned nil")
	}
}

func TestDefault(t *testing.T) {
	logger := Default()
	if logger == nil {
		t.Fatal("Default returned nil")
	}
}

func TestLogLevels(t *testing.T) {
	var buf bytes.Buffer
	logger := New(Config{
		Level:    LevelDebug,
		Format:   FormatText,
		Output:   &buf,
		Colorize: false,
	})

	tests := []struct {
		name     string
		logFunc  func(string, ...Field)
		level    string
		expected bool
	}{
		{"debug", logger.Debug, "DEBUG", true},
		{"info", logger.Info, "INFO", true},
		{"warn", logger.Warn, "WARN", true},
		{"error", logger.Error, "ERROR", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			buf.Reset()
			tt.logFunc("test message")
			output := buf.String()

			if tt.expected && !strings.Contains(output, tt.level) {
				t.Errorf("Expected output to contain %s, got: %s", tt.level, output)
			}
			if tt.expected && !strings.Contains(output, "test message") {
				t.Errorf("Expected output to contain message, got: %s", output)
			}
		})
	}
}

func TestLogLevelFiltering(t *testing.T) {
	var buf bytes.Buffer
	logger := New(Config{
		Level:    LevelWarn,
		Format:   FormatText,
		Output:   &buf,
		Colorize: false,
	})

	// Debug and Info should be filtered
	logger.Debug("debug message")
	if buf.Len() > 0 {
		t.Error("Debug should be filtered at Warn level")
	}

	logger.Info("info message")
	if buf.Len() > 0 {
		t.Error("Info should be filtered at Warn level")
	}

	// Warn and Error should pass
	logger.Warn("warn message")
	if !strings.Contains(buf.String(), "warn message") {
		t.Error("Warn should not be filtered at Warn level")
	}

	buf.Reset()
	logger.Error("error message")
	if !strings.Contains(buf.String(), "error message") {
		t.Error("Error should not be filtered at Warn level")
	}
}

func TestLogWithFields(t *testing.T) {
	var buf bytes.Buffer
	logger := New(Config{
		Level:    LevelInfo,
		Format:   FormatText,
		Output:   &buf,
		Colorize: false,
	})

	logger.Info("test message",
		String("user", "testuser"),
		Int("count", 42),
		Bool("enabled", true),
	)

	output := buf.String()
	if !strings.Contains(output, "user=testuser") {
		t.Errorf("Expected user field, got: %s", output)
	}
	if !strings.Contains(output, "count=42") {
		t.Errorf("Expected count field, got: %s", output)
	}
	if !strings.Contains(output, "enabled=true") {
		t.Errorf("Expected enabled field, got: %s", output)
	}
}

func TestLogJSONFormat(t *testing.T) {
	var buf bytes.Buffer
	logger := New(Config{
		Level:  LevelInfo,
		Format: FormatJSON,
		Output: &buf,
	})

	logger.Info("test message",
		String("user", "testuser"),
		Int("count", 42),
	)

	var entry map[string]interface{}
	if err := json.Unmarshal(buf.Bytes(), &entry); err != nil {
		t.Fatalf("Failed to parse JSON: %v", err)
	}

	if entry["msg"] != "test message" {
		t.Errorf("Expected msg 'test message', got %v", entry["msg"])
	}
	if entry["level"] != "INFO" {
		t.Errorf("Expected level 'INFO', got %v", entry["level"])
	}
	if entry["user"] != "testuser" {
		t.Errorf("Expected user 'testuser', got %v", entry["user"])
	}
	if entry["count"] != float64(42) {
		t.Errorf("Expected count 42, got %v", entry["count"])
	}
}

func TestWithFields(t *testing.T) {
	var buf bytes.Buffer
	logger := New(Config{
		Level:    LevelInfo,
		Format:   FormatText,
		Output:   &buf,
		Colorize: false,
	})

	childLogger := logger.WithFields(
		String("component", "auth"),
		String("version", "1.0"),
	)

	childLogger.Info("test message")

	output := buf.String()
	if !strings.Contains(output, "component=auth") {
		t.Errorf("Expected component field, got: %s", output)
	}
	if !strings.Contains(output, "version=1.0") {
		t.Errorf("Expected version field, got: %s", output)
	}
}

func TestWithPrefix(t *testing.T) {
	var buf bytes.Buffer
	logger := New(Config{
		Level:    LevelInfo,
		Format:   FormatText,
		Output:   &buf,
		Colorize: false,
	})

	childLogger := logger.WithPrefix("AUTH")
	childLogger.Info("test message")

	output := buf.String()
	if !strings.Contains(output, "[AUTH]") {
		t.Errorf("Expected prefix [AUTH], got: %s", output)
	}
}

func TestFormattedLogging(t *testing.T) {
	var buf bytes.Buffer
	logger := New(Config{
		Level:    LevelInfo,
		Format:   FormatText,
		Output:   &buf,
		Colorize: false,
	})

	logger.Infof("User %s logged in from %s", "testuser", "192.168.1.1")

	output := buf.String()
	if !strings.Contains(output, "User testuser logged in from 192.168.1.1") {
		t.Errorf("Expected formatted message, got: %s", output)
	}
}

func TestErrorField(t *testing.T) {
	var buf bytes.Buffer
	logger := New(Config{
		Level:    LevelInfo,
		Format:   FormatText,
		Output:   &buf,
		Colorize: false,
	})

	err := &testError{msg: "connection refused"}
	logger.Error("failed to connect", Err(err))

	output := buf.String()
	if !strings.Contains(output, "error=connection refused") {
		t.Errorf("Expected error field, got: %s", output)
	}
}

func TestNilErrorField(t *testing.T) {
	var buf bytes.Buffer
	logger := New(Config{
		Level:    LevelInfo,
		Format:   FormatText,
		Output:   &buf,
		Colorize: false,
	})

	logger.Info("success", Err(nil))

	output := buf.String()
	if !strings.Contains(output, "error=<nil>") {
		t.Errorf("Expected error=<nil>, got: %s", output)
	}
}

func TestDurationField(t *testing.T) {
	var buf bytes.Buffer
	logger := New(Config{
		Level:    LevelInfo,
		Format:   FormatText,
		Output:   &buf,
		Colorize: false,
	})

	logger.Info("operation completed", Duration("elapsed", 1500*time.Millisecond))

	output := buf.String()
	if !strings.Contains(output, "elapsed=1.5s") {
		t.Errorf("Expected elapsed field, got: %s", output)
	}
}

func TestSetLevel(t *testing.T) {
	var buf bytes.Buffer
	logger := New(Config{
		Level:  LevelInfo,
		Format: FormatText,
		Output: &buf,
	})

	logger.Debug("debug message")
	if buf.Len() > 0 {
		t.Error("Debug should be filtered initially")
	}

	logger.SetLevel(LevelDebug)
	logger.Debug("debug message")
	if buf.Len() == 0 {
		t.Error("Debug should pass after SetLevel")
	}
}

func TestSetFormat(t *testing.T) {
	var buf bytes.Buffer
	logger := New(Config{
		Level:  LevelInfo,
		Format: FormatText,
		Output: &buf,
	})

	logger.Info("text message")
	if strings.HasPrefix(buf.String(), "{") {
		t.Error("Expected text format initially")
	}

	buf.Reset()
	logger.SetFormat(FormatJSON)
	logger.Info("json message")
	if !strings.HasPrefix(buf.String(), "{") {
		t.Error("Expected JSON format after SetFormat")
	}
}

func TestParseLevel(t *testing.T) {
	tests := []struct {
		input    string
		expected Level
	}{
		{"debug", LevelDebug},
		{"DEBUG", LevelDebug},
		{"info", LevelInfo},
		{"INFO", LevelInfo},
		{"warn", LevelWarn},
		{"warning", LevelWarn},
		{"error", LevelError},
		{"ERROR", LevelError},
		{"invalid", LevelInfo},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := ParseLevel(tt.input)
			if result != tt.expected {
				t.Errorf("ParseLevel(%s) = %v, want %v", tt.input, result, tt.expected)
			}
		})
	}
}

func TestLevelString(t *testing.T) {
	tests := []struct {
		level    Level
		expected string
	}{
		{LevelDebug, "DEBUG"},
		{LevelInfo, "INFO"},
		{LevelWarn, "WARN"},
		{LevelError, "ERROR"},
		{Level(99), "UNKNOWN"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			if tt.level.String() != tt.expected {
				t.Errorf("Level(%d).String() = %s, want %s", tt.level, tt.level.String(), tt.expected)
			}
		})
	}
}

func TestGlobalLogger(t *testing.T) {
	var buf bytes.Buffer
	logger := New(Config{
		Level:    LevelInfo,
		Format:   FormatText,
		Output:   &buf,
		Colorize: false,
	})

	oldLogger := GetDefault()
	SetDefault(logger)
	defer SetDefault(oldLogger)

	Info("global message")
	if !strings.Contains(buf.String(), "global message") {
		t.Error("Expected global logger to work")
	}
}

func TestFieldTypes(t *testing.T) {
	var buf bytes.Buffer
	logger := New(Config{
		Level:    LevelInfo,
		Format:   FormatText,
		Output:   &buf,
		Colorize: false,
	})

	logger.Info("types test",
		String("str", "value"),
		Int("int", 42),
		Int64("int64", 9223372036854775807),
		Float64("float", 3.14),
		Bool("bool", true),
		Any("any", map[string]int{"a": 1}),
	)

	output := buf.String()
	if !strings.Contains(output, "str=value") {
		t.Errorf("Missing string field: %s", output)
	}
	if !strings.Contains(output, "int=42") {
		t.Errorf("Missing int field: %s", output)
	}
	if !strings.Contains(output, "int64=9223372036854775807") {
		t.Errorf("Missing int64 field: %s", output)
	}
	if !strings.Contains(output, "float=3.14") {
		t.Errorf("Missing float field: %s", output)
	}
	if !strings.Contains(output, "bool=true") {
		t.Errorf("Missing bool field: %s", output)
	}
}

func TestCaller(t *testing.T) {
	caller := Caller(0)
	if !strings.Contains(caller, "logger_test.go") {
		t.Errorf("Expected caller to contain logger_test.go, got: %s", caller)
	}
}

// testError is a simple error for testing
type testError struct {
	msg string
}

func (e *testError) Error() string {
	return e.msg
}
