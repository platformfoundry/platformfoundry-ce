// Package log provides structured logging for Platform Foundry.
// It supports multiple output formats (text, JSON) and log levels.
package log

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"runtime"
	"strings"
	"sync"
	"time"
)

// Level represents a log level
type Level int

const (
	LevelDebug Level = iota
	LevelInfo
	LevelWarn
	LevelError
)

func (l Level) String() string {
	switch l {
	case LevelDebug:
		return "DEBUG"
	case LevelInfo:
		return "INFO"
	case LevelWarn:
		return "WARN"
	case LevelError:
		return "ERROR"
	default:
		return "UNKNOWN"
	}
}

// Format represents the output format
type Format int

const (
	FormatText Format = iota
	FormatJSON
)

// Field represents a structured log field
type Field struct {
	Key   string
	Value interface{}
}

// String creates a string field
func String(key, value string) Field {
	return Field{Key: key, Value: value}
}

// Int creates an integer field
func Int(key string, value int) Field {
	return Field{Key: key, Value: value}
}

// Int64 creates an int64 field
func Int64(key string, value int64) Field {
	return Field{Key: key, Value: value}
}

// Float64 creates a float64 field
func Float64(key string, value float64) Field {
	return Field{Key: key, Value: value}
}

// Bool creates a boolean field
func Bool(key string, value bool) Field {
	return Field{Key: key, Value: value}
}

// Err creates an error field
func Err(err error) Field {
	if err == nil {
		return Field{Key: "error", Value: nil}
	}
	return Field{Key: "error", Value: err.Error()}
}

// Duration creates a duration field
func Duration(key string, value time.Duration) Field {
	return Field{Key: key, Value: value.String()}
}

// Any creates a field with any value
func Any(key string, value interface{}) Field {
	return Field{Key: key, Value: value}
}

// Logger provides structured logging
type Logger struct {
	mu       sync.Mutex
	out      io.Writer
	level    Level
	format   Format
	fields   []Field
	prefix   string
	colorize bool
}

// Config holds logger configuration
type Config struct {
	Level    Level
	Format   Format
	Output   io.Writer
	Colorize bool
}

// New creates a new logger with the given configuration
func New(cfg Config) *Logger {
	out := cfg.Output
	if out == nil {
		out = os.Stdout
	}

	return &Logger{
		out:      out,
		level:    cfg.Level,
		format:   cfg.Format,
		colorize: cfg.Colorize,
	}
}

// Default creates a logger with default settings
func Default() *Logger {
	return New(Config{
		Level:    LevelInfo,
		Format:   FormatText,
		Output:   os.Stdout,
		Colorize: true,
	})
}

// WithFields returns a new logger with additional fields
func (l *Logger) WithFields(fields ...Field) *Logger {
	newLogger := &Logger{
		out:      l.out,
		level:    l.level,
		format:   l.format,
		prefix:   l.prefix,
		colorize: l.colorize,
		fields:   make([]Field, len(l.fields)+len(fields)),
	}
	copy(newLogger.fields, l.fields)
	copy(newLogger.fields[len(l.fields):], fields)
	return newLogger
}

// WithPrefix returns a new logger with a prefix
func (l *Logger) WithPrefix(prefix string) *Logger {
	return &Logger{
		out:      l.out,
		level:    l.level,
		format:   l.format,
		prefix:   prefix,
		colorize: l.colorize,
		fields:   l.fields,
	}
}

// SetLevel sets the minimum log level
func (l *Logger) SetLevel(level Level) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.level = level
}

// SetFormat sets the output format
func (l *Logger) SetFormat(format Format) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.format = format
}

// SetOutput sets the output writer
func (l *Logger) SetOutput(w io.Writer) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.out = w
}

// Debug logs a debug message
func (l *Logger) Debug(msg string, fields ...Field) {
	l.log(LevelDebug, msg, fields)
}

// Info logs an info message
func (l *Logger) Info(msg string, fields ...Field) {
	l.log(LevelInfo, msg, fields)
}

// Warn logs a warning message
func (l *Logger) Warn(msg string, fields ...Field) {
	l.log(LevelWarn, msg, fields)
}

// Error logs an error message
func (l *Logger) Error(msg string, fields ...Field) {
	l.log(LevelError, msg, fields)
}

// Debugf logs a formatted debug message
func (l *Logger) Debugf(format string, args ...interface{}) {
	l.log(LevelDebug, fmt.Sprintf(format, args...), nil)
}

// Infof logs a formatted info message
func (l *Logger) Infof(format string, args ...interface{}) {
	l.log(LevelInfo, fmt.Sprintf(format, args...), nil)
}

// Warnf logs a formatted warning message
func (l *Logger) Warnf(format string, args ...interface{}) {
	l.log(LevelWarn, fmt.Sprintf(format, args...), nil)
}

// Errorf logs a formatted error message
func (l *Logger) Errorf(format string, args ...interface{}) {
	l.log(LevelError, fmt.Sprintf(format, args...), nil)
}

func (l *Logger) log(level Level, msg string, fields []Field) {
	if level < l.level {
		return
	}

	l.mu.Lock()
	defer l.mu.Unlock()

	// Combine logger fields with call fields
	allFields := make([]Field, 0, len(l.fields)+len(fields))
	allFields = append(allFields, l.fields...)
	allFields = append(allFields, fields...)

	if l.format == FormatJSON {
		l.logJSON(level, msg, allFields)
	} else {
		l.logText(level, msg, allFields)
	}
}

func (l *Logger) logText(level Level, msg string, fields []Field) {
	var b strings.Builder

	// Timestamp
	b.WriteString(time.Now().Format("15:04:05"))
	b.WriteString(" ")

	// Level with optional color
	levelStr := level.String()
	if l.colorize {
		levelStr = l.colorLevel(level)
	}
	b.WriteString(fmt.Sprintf("%-5s", levelStr))
	b.WriteString(" ")

	// Prefix
	if l.prefix != "" {
		b.WriteString("[")
		b.WriteString(l.prefix)
		b.WriteString("] ")
	}

	// Message
	b.WriteString(msg)

	// Fields
	if len(fields) > 0 {
		b.WriteString(" ")
		for i, f := range fields {
			if i > 0 {
				b.WriteString(" ")
			}
			b.WriteString(f.Key)
			b.WriteString("=")
			b.WriteString(fmt.Sprintf("%v", f.Value))
		}
	}

	b.WriteString("\n")
	fmt.Fprint(l.out, b.String())
}

func (l *Logger) logJSON(level Level, msg string, fields []Field) {
	entry := map[string]interface{}{
		"time":  time.Now().Format(time.RFC3339),
		"level": level.String(),
		"msg":   msg,
	}

	if l.prefix != "" {
		entry["component"] = l.prefix
	}

	for _, f := range fields {
		entry[f.Key] = f.Value
	}

	data, err := json.Marshal(entry)
	if err != nil {
		fmt.Fprintf(l.out, `{"level":"ERROR","msg":"failed to marshal log entry: %s"}`+"\n", err)
		return
	}

	fmt.Fprintln(l.out, string(data))
}

func (l *Logger) colorLevel(level Level) string {
	var color string
	switch level {
	case LevelDebug:
		color = "\033[36m" // Cyan
	case LevelInfo:
		color = "\033[32m" // Green
	case LevelWarn:
		color = "\033[33m" // Yellow
	case LevelError:
		color = "\033[31m" // Red
	}
	return color + level.String() + "\033[0m"
}

// Global logger instance
var globalLogger = Default()

// SetDefault sets the global logger
func SetDefault(logger *Logger) {
	globalLogger = logger
}

// GetDefault returns the global logger
func GetDefault() *Logger {
	return globalLogger
}

// Package-level logging functions

// Debug logs a debug message using the global logger
func Debug(msg string, fields ...Field) {
	globalLogger.Debug(msg, fields...)
}

// Info logs an info message using the global logger
func Info(msg string, fields ...Field) {
	globalLogger.Info(msg, fields...)
}

// Warn logs a warning message using the global logger
func Warn(msg string, fields ...Field) {
	globalLogger.Warn(msg, fields...)
}

// Error logs an error message using the global logger
func Error(msg string, fields ...Field) {
	globalLogger.Error(msg, fields...)
}

// Debugf logs a formatted debug message using the global logger
func Debugf(format string, args ...interface{}) {
	globalLogger.Debugf(format, args...)
}

// Infof logs a formatted info message using the global logger
func Infof(format string, args ...interface{}) {
	globalLogger.Infof(format, args...)
}

// Warnf logs a formatted warning message using the global logger
func Warnf(format string, args ...interface{}) {
	globalLogger.Warnf(format, args...)
}

// Errorf logs a formatted error message using the global logger
func Errorf(format string, args ...interface{}) {
	globalLogger.Errorf(format, args...)
}

// ParseLevel parses a level string
func ParseLevel(s string) Level {
	switch strings.ToLower(s) {
	case "debug":
		return LevelDebug
	case "info":
		return LevelInfo
	case "warn", "warning":
		return LevelWarn
	case "error":
		return LevelError
	default:
		return LevelInfo
	}
}

// Caller returns file:line of the caller
func Caller(skip int) string {
	_, file, line, ok := runtime.Caller(skip + 1)
	if !ok {
		return "unknown"
	}
	// Get just the filename, not the full path
	short := file
	for i := len(file) - 1; i > 0; i-- {
		if file[i] == '/' || file[i] == '\\' {
			short = file[i+1:]
			break
		}
	}
	return fmt.Sprintf("%s:%d", short, line)
}
