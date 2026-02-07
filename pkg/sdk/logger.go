// Package sdk provides the Plugin SDK for Platform Foundry.
package sdk

import (
	"fmt"
	"io"
	"os"
	"strings"
	"time"
)

// Logger is the logging interface for plugins
type Logger interface {
	// Trace logs a trace message
	Trace(msg string, args ...interface{})

	// Debug logs a debug message
	Debug(msg string, args ...interface{})

	// Info logs an info message
	Info(msg string, args ...interface{})

	// Warn logs a warning message
	Warn(msg string, args ...interface{})

	// Error logs an error message
	Error(msg string, args ...interface{})

	// With returns a logger with additional context
	With(args ...interface{}) Logger

	// Named returns a logger with a name prefix
	Named(name string) Logger
}

// LogLevel represents log severity
type LogLevel int

const (
	LogLevelTrace LogLevel = iota
	LogLevelDebug
	LogLevelInfo
	LogLevelWarn
	LogLevelError
)

func (l LogLevel) String() string {
	switch l {
	case LogLevelTrace:
		return "TRACE"
	case LogLevelDebug:
		return "DEBUG"
	case LogLevelInfo:
		return "INFO"
	case LogLevelWarn:
		return "WARN"
	case LogLevelError:
		return "ERROR"
	default:
		return "UNKNOWN"
	}
}

// DefaultLogger is a simple logger implementation
type DefaultLogger struct {
	level  LogLevel
	name   string
	fields []interface{}
	output io.Writer
}

// NewDefaultLogger creates a new default logger
func NewDefaultLogger(debug bool) *DefaultLogger {
	level := LogLevelInfo
	if debug {
		level = LogLevelDebug
	}
	return &DefaultLogger{
		level:  level,
		output: os.Stderr,
	}
}

// NewDefaultLoggerWithLevel creates a new default logger with a specific level
func NewDefaultLoggerWithLevel(level LogLevel) *DefaultLogger {
	return &DefaultLogger{
		level:  level,
		output: os.Stderr,
	}
}

func (l *DefaultLogger) log(level LogLevel, msg string, args ...interface{}) {
	if level < l.level {
		return
	}

	timestamp := time.Now().Format("2006-01-02T15:04:05.000Z07:00")
	prefix := ""
	if l.name != "" {
		prefix = "[" + l.name + "] "
	}

	var sb strings.Builder
	sb.WriteString(timestamp)
	sb.WriteString(" ")
	sb.WriteString(level.String())
	sb.WriteString(" ")
	sb.WriteString(prefix)
	sb.WriteString(msg)

	// Add fields from With()
	allArgs := append(l.fields, args...)
	if len(allArgs) > 0 {
		sb.WriteString(" ")
		for i := 0; i < len(allArgs); i += 2 {
			if i > 0 {
				sb.WriteString(" ")
			}
			key := fmt.Sprintf("%v", allArgs[i])
			var value interface{} = "MISSING"
			if i+1 < len(allArgs) {
				value = allArgs[i+1]
			}
			sb.WriteString(key)
			sb.WriteString("=")
			sb.WriteString(fmt.Sprintf("%v", value))
		}
	}
	sb.WriteString("\n")

	fmt.Fprint(l.output, sb.String())
}

// Trace logs a trace message
func (l *DefaultLogger) Trace(msg string, args ...interface{}) {
	l.log(LogLevelTrace, msg, args...)
}

// Debug logs a debug message
func (l *DefaultLogger) Debug(msg string, args ...interface{}) {
	l.log(LogLevelDebug, msg, args...)
}

// Info logs an info message
func (l *DefaultLogger) Info(msg string, args ...interface{}) {
	l.log(LogLevelInfo, msg, args...)
}

// Warn logs a warning message
func (l *DefaultLogger) Warn(msg string, args ...interface{}) {
	l.log(LogLevelWarn, msg, args...)
}

// Error logs an error message
func (l *DefaultLogger) Error(msg string, args ...interface{}) {
	l.log(LogLevelError, msg, args...)
}

// With returns a logger with additional context
func (l *DefaultLogger) With(args ...interface{}) Logger {
	newFields := make([]interface{}, len(l.fields)+len(args))
	copy(newFields, l.fields)
	copy(newFields[len(l.fields):], args)
	return &DefaultLogger{
		level:  l.level,
		name:   l.name,
		fields: newFields,
		output: l.output,
	}
}

// Named returns a logger with a name prefix
func (l *DefaultLogger) Named(name string) Logger {
	newName := name
	if l.name != "" {
		newName = l.name + "." + name
	}
	return &DefaultLogger{
		level:  l.level,
		name:   newName,
		fields: l.fields,
		output: l.output,
	}
}

// SetOutput sets the output writer
func (l *DefaultLogger) SetOutput(w io.Writer) {
	l.output = w
}

// SetLevel sets the log level
func (l *DefaultLogger) SetLevel(level LogLevel) {
	l.level = level
}

// NullLogger is a logger that discards all output
type NullLogger struct{}

func (NullLogger) Trace(msg string, args ...interface{}) {}
func (NullLogger) Debug(msg string, args ...interface{}) {}
func (NullLogger) Info(msg string, args ...interface{})  {}
func (NullLogger) Warn(msg string, args ...interface{})  {}
func (NullLogger) Error(msg string, args ...interface{}) {}
func (l NullLogger) With(args ...interface{}) Logger     { return l }
func (l NullLogger) Named(name string) Logger            { return l }
