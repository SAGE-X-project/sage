package logger

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"runtime"
	"strings"
	"sync"
	"time"
)

// Level represents the severity level of a log message
type Level int

const (
	DebugLevel Level = iota
	InfoLevel
	WarnLevel
	ErrorLevel
	FatalLevel
)

// String returns the string representation of a log level
func (l Level) String() string {
	switch l {
	case DebugLevel:
		return "DEBUG"
	case InfoLevel:
		return "INFO"
	case WarnLevel:
		return "WARN"
	case ErrorLevel:
		return "ERROR"
	case FatalLevel:
		return "FATAL"
	default:
		return "UNKNOWN"
	}
}

// Field represents a structured logging field
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

// Bool creates a boolean field
func Bool(key string, value bool) Field {
	return Field{Key: key, Value: value}
}

// Error creates an error field
func Error(err error) Field {
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

// Logger defines the interface for structured logging
type Logger interface {
	Debug(msg string, fields ...Field)
	Info(msg string, fields ...Field)
	Warn(msg string, fields ...Field)
	Error(msg string, fields ...Field)
	Fatal(msg string, fields ...Field)

	WithContext(ctx context.Context) Logger
	WithFields(fields ...Field) Logger
	SetLevel(level Level)
	GetLevel() Level
}

// StructuredLogger implements the Logger interface with JSON output
type StructuredLogger struct {
	mu          sync.RWMutex
	level       Level
	output      io.Writer
	context     context.Context
	baseFields  []Field
	timeFormat  string
	prettyPrint bool
}

// NewLogger creates a new structured logger
func NewLogger(output io.Writer, level Level) *StructuredLogger {
	return &StructuredLogger{
		level:      level,
		output:     output,
		timeFormat: time.RFC3339,
	}
}

// NewDefaultLogger creates a logger with default settings
func NewDefaultLogger() *StructuredLogger {
	level := InfoLevel
	if envLevel := os.Getenv("SAGE_LOG_LEVEL"); envLevel != "" {
		switch strings.ToUpper(envLevel) {
		case "DEBUG":
			level = DebugLevel
		case "INFO":
			level = InfoLevel
		case "WARN":
			level = WarnLevel
		case "ERROR":
			level = ErrorLevel
		}
	}

	return NewLogger(os.Stdout, level)
}

// SetPrettyPrint enables or disables pretty printing of JSON logs
func (l *StructuredLogger) SetPrettyPrint(pretty bool) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.prettyPrint = pretty
}

// SetTimeFormat sets the time format for log entries
func (l *StructuredLogger) SetTimeFormat(format string) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.timeFormat = format
}

// Debug logs a debug level message
func (l *StructuredLogger) Debug(msg string, fields ...Field) {
	l.log(DebugLevel, msg, fields...)
}

// Info logs an info level message
func (l *StructuredLogger) Info(msg string, fields ...Field) {
	l.log(InfoLevel, msg, fields...)
}

// Warn logs a warning level message
func (l *StructuredLogger) Warn(msg string, fields ...Field) {
	l.log(WarnLevel, msg, fields...)
}

// Error logs an error level message
func (l *StructuredLogger) Error(msg string, fields ...Field) {
	l.log(ErrorLevel, msg, fields...)
}

// Fatal logs a fatal level message and exits
func (l *StructuredLogger) Fatal(msg string, fields ...Field) {
	l.log(FatalLevel, msg, fields...)
	os.Exit(1)
}

// WithContext returns a new logger with the given context
func (l *StructuredLogger) WithContext(ctx context.Context) Logger {
	l.mu.RLock()
	defer l.mu.RUnlock()

	return &StructuredLogger{
		level:       l.level,
		output:      l.output,
		context:     ctx,
		baseFields:  l.baseFields,
		timeFormat:  l.timeFormat,
		prettyPrint: l.prettyPrint,
	}
}

// WithFields returns a new logger with additional fields
func (l *StructuredLogger) WithFields(fields ...Field) Logger {
	l.mu.RLock()
	defer l.mu.RUnlock()

	newFields := make([]Field, len(l.baseFields)+len(fields))
	copy(newFields, l.baseFields)
	copy(newFields[len(l.baseFields):], fields)

	return &StructuredLogger{
		level:       l.level,
		output:      l.output,
		context:     l.context,
		baseFields:  newFields,
		timeFormat:  l.timeFormat,
		prettyPrint: l.prettyPrint,
	}
}

// SetLevel sets the minimum log level
func (l *StructuredLogger) SetLevel(level Level) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.level = level
}

// GetLevel returns the current log level
func (l *StructuredLogger) GetLevel() Level {
	l.mu.RLock()
	defer l.mu.RUnlock()
	return l.level
}

// log is the internal logging method
func (l *StructuredLogger) log(level Level, msg string, fields ...Field) {
	l.mu.RLock()
	defer l.mu.RUnlock()

	if level < l.level {
		return
	}

	entry := make(map[string]interface{})
	entry["timestamp"] = time.Now().Format(l.timeFormat)
	entry["level"] = level.String()
	entry["message"] = msg

	// Add caller information
	if pc, file, line, ok := runtime.Caller(2); ok {
		entry["caller"] = fmt.Sprintf("%s:%d", file, line)
		if fn := runtime.FuncForPC(pc); fn != nil {
			entry["function"] = fn.Name()
		}
	}

	// Add context fields if available
	if l.context != nil {
		if requestID := l.context.Value("request_id"); requestID != nil {
			entry["request_id"] = requestID
		}
		if traceID := l.context.Value("trace_id"); traceID != nil {
			entry["trace_id"] = traceID
		}
	}

	// Add base fields
	for _, field := range l.baseFields {
		entry[field.Key] = field.Value
	}

	// Add provided fields
	for _, field := range fields {
		entry[field.Key] = field.Value
	}

	// Marshal to JSON
	var data []byte
	var err error
	if l.prettyPrint {
		data, err = json.MarshalIndent(entry, "", "  ")
	} else {
		data, err = json.Marshal(entry)
	}

	if err != nil {
		fmt.Fprintf(l.output, `{"level":"ERROR","message":"Failed to marshal log entry","error":"%v"}`+"\n", err)
		return
	}

	// Write to output
	fmt.Fprintf(l.output, "%s\n", data)
}

// SageError represents a structured error with additional context
type SageError struct {
	Code    string                 `json:"code"`
	Message string                 `json:"message"`
	Details map[string]interface{} `json:"details,omitempty"`
	Cause   error                  `json:"-"`
}

// Error implements the error interface
func (e *SageError) Error() string {
	if e.Cause != nil {
		return fmt.Sprintf("%s: %s (caused by: %v)", e.Code, e.Message, e.Cause)
	}
	return fmt.Sprintf("%s: %s", e.Code, e.Message)
}

// Unwrap returns the underlying error
func (e *SageError) Unwrap() error {
	return e.Cause
}

// WithDetails adds details to the error
func (e *SageError) WithDetails(key string, value interface{}) *SageError {
	if e.Details == nil {
		e.Details = make(map[string]interface{})
	}
	e.Details[key] = value
	return e
}

// NewSageError creates a new SAGE error
func NewSageError(code, message string, cause error) *SageError {
	return &SageError{
		Code:    code,
		Message: message,
		Cause:   cause,
	}
}

// Common error codes
const (
	ErrCodeInternal          = "INTERNAL_ERROR"
	ErrCodeInvalidInput      = "INVALID_INPUT"
	ErrCodeNotFound          = "NOT_FOUND"
	ErrCodeUnauthorized      = "UNAUTHORIZED"
	ErrCodeForbidden         = "FORBIDDEN"
	ErrCodeConflict          = "CONFLICT"
	ErrCodeTimeout           = "TIMEOUT"
	ErrCodeNetworkError      = "NETWORK_ERROR"
	ErrCodeBlockchainError   = "BLOCKCHAIN_ERROR"
	ErrCodeCryptoError       = "CRYPTO_ERROR"
	ErrCodeValidationError   = "VALIDATION_ERROR"
	ErrCodeConfigurationError = "CONFIGURATION_ERROR"
)

// Global logger instance
var defaultLogger = NewDefaultLogger()

// SetDefaultLogger sets the global default logger
func SetDefaultLogger(logger Logger) {
	if l, ok := logger.(*StructuredLogger); ok {
		defaultLogger = l
	}
}

// GetDefaultLogger returns the global default logger
func GetDefaultLogger() *StructuredLogger {
	return defaultLogger
}

// Package-level logging functions using the default logger

// Debug logs a debug message using the default logger
func Debug(msg string, fields ...Field) {
	defaultLogger.Debug(msg, fields...)
}

// Info logs an info message using the default logger
func Info(msg string, fields ...Field) {
	defaultLogger.Info(msg, fields...)
}

// Warn logs a warning message using the default logger
func Warn(msg string, fields ...Field) {
	defaultLogger.Warn(msg, fields...)
}

// ErrorMsg logs an error message using the default logger
func ErrorMsg(msg string, fields ...Field) {
	defaultLogger.Error(msg, fields...)
}

// Fatal logs a fatal message using the default logger and exits
func Fatal(msg string, fields ...Field) {
	defaultLogger.Fatal(msg, fields...)
}