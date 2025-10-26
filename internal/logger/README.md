# Logger Package

## Overview

The `logger` package provides a structured logging system for SAGE. It offers JSON-formatted logging with context support, log levels, and typed fields for better observability and debugging.

This package is designed for internal use within SAGE and provides a consistent logging interface across all components.

## Features

- **Structured Logging**: JSON output with typed fields
- **Log Levels**: Debug, Info, Warn, Error, Fatal
- **Context Integration**: Automatic request ID and trace propagation
- **Type-Safe Fields**: Strong typing for log fields
- **Performance**: Optimized for high-throughput scenarios
- **Error Handling**: Custom `SageError` type with structured details

## Architecture

```
┌─────────────────────────────────────────────┐
│         Application Code                    │
│    logger.Info("msg", fields...)            │
└─────────────────┬───────────────────────────┘
                  │
                  ▼
┌─────────────────────────────────────────────┐
│    Logger Interface                         │
│    - Level filtering                        │
│    - Context extraction                     │
│    - Field formatting                       │
└─────────────────┬───────────────────────────┘
                  │
                  ▼
┌─────────────────────────────────────────────┐
│    JSON Encoder                             │
│    - Structured output                      │
│    - Timestamp formatting                   │
└─────────────────┬───────────────────────────┘
                  │
                  ▼
┌─────────────────────────────────────────────┐
│    stdout / stderr                          │
└─────────────────────────────────────────────┘
```

## Core Types

### Logger Interface

```go
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
```

### Log Levels

```go
const (
    DebugLevel Level = iota  // -1: Detailed debug information
    InfoLevel                //  0: General informational messages
    WarnLevel                //  1: Warning messages
    ErrorLevel               //  2: Error messages
    FatalLevel               //  3: Fatal errors (calls os.Exit(1))
)
```

### Field Type

```go
type Field struct {
    Key   string
    Value interface{}
}
```

## Usage

### Basic Logging

```go
package main

import "github.com/sage-x-project/sage/internal/logger"

func main() {
    log := logger.New()

    // Simple message
    log.Info("Application started")

    // With fields
    log.Info("User logged in",
        logger.String("user_id", "12345"),
        logger.String("ip", "192.168.1.1"),
    )

    // Warning
    log.Warn("High memory usage",
        logger.Int("usage_mb", 1024),
        logger.Float64("usage_percent", 85.5),
    )

    // Error
    log.Error("Database connection failed",
        logger.Error(err),
        logger.String("host", "db.example.com"),
        logger.Int("retry_count", 3),
    )
}
```

Output:
```json
{"level":"info","ts":"2025-10-26T10:30:00Z","msg":"Application started"}
{"level":"info","ts":"2025-10-26T10:30:01Z","msg":"User logged in","user_id":"12345","ip":"192.168.1.1"}
{"level":"warn","ts":"2025-10-26T10:30:02Z","msg":"High memory usage","usage_mb":1024,"usage_percent":85.5}
{"level":"error","ts":"2025-10-26T10:30:03Z","msg":"Database connection failed","error":"connection refused","host":"db.example.com","retry_count":3}
```

### Context-Aware Logging

```go
import (
    "context"
    "github.com/sage-x-project/sage/internal/logger"
)

func HandleRequest(ctx context.Context, requestID string) {
    // Add request ID to context
    ctx = logger.WithRequestID(ctx, requestID)

    log := logger.New().WithContext(ctx)

    // All logs will include request_id
    log.Info("Request received")
    log.Debug("Processing request")
    log.Info("Request completed")
}
```

Output:
```json
{"level":"info","ts":"2025-10-26T10:30:00Z","msg":"Request received","request_id":"req-12345"}
{"level":"debug","ts":"2025-10-26T10:30:00Z","msg":"Processing request","request_id":"req-12345"}
{"level":"info","ts":"2025-10-26T10:30:01Z","msg":"Request completed","request_id":"req-12345"}
```

### Persistent Fields

```go
// Create logger with persistent fields
baseLog := logger.New().WithFields(
    logger.String("service", "sage-server"),
    logger.String("version", "1.0.0"),
)

// All logs from this logger will include service and version
baseLog.Info("Server started")
baseLog.Info("Listening on port 8080",
    logger.Int("port", 8080),
)
```

Output:
```json
{"level":"info","ts":"2025-10-26T10:30:00Z","msg":"Server started","service":"sage-server","version":"1.0.0"}
{"level":"info","ts":"2025-10-26T10:30:00Z","msg":"Listening on port 8080","service":"sage-server","version":"1.0.0","port":8080}
```

### Log Level Control

```go
log := logger.New()

// Set to Info level (Debug won't show)
log.SetLevel(logger.InfoLevel)

log.Debug("This won't appear")
log.Info("This will appear")

// Change to Debug level
log.SetLevel(logger.DebugLevel)
log.Debug("Now this appears")

// Check current level
if log.GetLevel() == logger.DebugLevel {
    // Expensive debug operation
}
```

### Field Types

```go
import "time"

log := logger.New()

log.Info("All field types",
    // Strings
    logger.String("name", "Alice"),

    // Numbers
    logger.Int("age", 30),
    logger.Int64("large_number", 9223372036854775807),
    logger.Float64("price", 19.99),

    // Booleans
    logger.Bool("active", true),

    // Time
    logger.Time("timestamp", time.Now()),
    logger.Duration("elapsed", 150*time.Millisecond),

    // Errors
    logger.Error(err),

    // Complex objects (JSON serialized)
    logger.Any("metadata", map[string]interface{}{
        "key1": "value1",
        "key2": 42,
    }),
)
```

## SageError Type

The package provides a custom error type with structured details:

```go
type SageError struct {
    Code    string                 // Error code (e.g., "AUTH_FAILED")
    Message string                 // Human-readable message
    Details map[string]interface{} // Additional structured data
    Cause   error                  // Underlying error
}

func (e *SageError) Error() string
func (e *SageError) Unwrap() error
```

### Using SageError

```go
import "github.com/sage-x-project/sage/internal/logger"

// Create structured error
err := &logger.SageError{
    Code:    "DID_RESOLUTION_FAILED",
    Message: "Failed to resolve DID",
    Details: map[string]interface{}{
        "did":         "did:sage:ethereum:0x123...",
        "chain":       "ethereum",
        "retry_count": 3,
    },
    Cause: originalErr,
}

// Log with automatic field extraction
log.Error("DID resolution error",
    logger.SageError(err),
)
```

Output:
```json
{
  "level": "error",
  "ts": "2025-10-26T10:30:00Z",
  "msg": "DID resolution error",
  "error_code": "DID_RESOLUTION_FAILED",
  "error": "Failed to resolve DID",
  "did": "did:sage:ethereum:0x123...",
  "chain": "ethereum",
  "retry_count": 3,
  "cause": "connection timeout"
}
```

## Best Practices

### 1. Use Appropriate Log Levels

```go
// ✅ Correct usage
log.Debug("Detailed crypto operation", logger.Bytes("signature", sig))
log.Info("Session created", logger.String("session_id", sid))
log.Warn("Cache miss rate high", logger.Float64("rate", 0.85))
log.Error("Failed to verify signature", logger.Error(err))
log.Fatal("Cannot start server", logger.Error(err)) // Exits application

// ❌ Wrong usage
log.Info("Detailed loop iteration i=42")  // Too verbose, use Debug
log.Error("User logged out")              // Not an error, use Info
```

### 2. Use Typed Fields

```go
// ✅ Correct - type-safe fields
log.Info("Request completed",
    logger.Int("status_code", 200),
    logger.Duration("duration", elapsed),
)

// ❌ Wrong - untyped fields
log.Info("Request completed",
    logger.String("status_code", "200"),  // Should be Int
    logger.String("duration", "150ms"),   // Should be Duration
)
```

### 3. Consistent Field Names

```go
// ✅ Correct - snake_case, consistent naming
log.Info("DID resolved",
    logger.String("did", did),
    logger.String("chain_id", chainID),
    logger.Duration("resolve_time", elapsed),
)

// ❌ Wrong - inconsistent naming
log.Info("DID resolved",
    logger.String("DID", did),           // Use lowercase
    logger.String("chainID", chainID),   // Use snake_case
    logger.String("time", "150ms"),      // Use Duration type
)
```

### 4. Context Propagation

```go
// ✅ Correct - propagate context
func ProcessRequest(ctx context.Context) error {
    log := logger.New().WithContext(ctx)

    log.Info("Processing started")

    if err := step1(ctx); err != nil {
        log.Error("Step 1 failed", logger.Error(err))
        return err
    }

    log.Info("Processing completed")
    return nil
}

func step1(ctx context.Context) error {
    log := logger.New().WithContext(ctx)  // Same request_id
    log.Debug("Executing step 1")
    // ...
}

// ❌ Wrong - no context propagation
func ProcessRequest(ctx context.Context) error {
    log := logger.New()  // Missing context
    log.Info("Processing started")
    // ...
}
```

### 5. Error Logging

```go
// ✅ Correct - include context
log.Error("Database query failed",
    logger.Error(err),
    logger.String("query", "SELECT * FROM users"),
    logger.String("table", "users"),
    logger.Int("retry_attempt", 2),
)

// ❌ Wrong - insufficient context
log.Error("Error", logger.Error(err))
```

## Performance Considerations

### Conditional Debug Logging

```go
// Expensive debug logging
if log.GetLevel() == logger.DebugLevel {
    // Only compute expensive data if debug is enabled
    details := computeExpensiveDetails()
    log.Debug("Detailed information", logger.Any("details", details))
}
```

### Field Allocation

```go
// ✅ Efficient - reuse field slices
fields := make([]logger.Field, 0, 5)
fields = append(fields, logger.String("key1", val1))
fields = append(fields, logger.String("key2", val2))
log.Info("Message", fields...)

// ❌ Less efficient - many small allocations
log.Info("Message",
    logger.String("key1", val1),
    logger.String("key2", val2),
    logger.String("key3", val3),
    // ... many more fields
)
```

## Testing

### Example Test

```go
package mypackage_test

import (
    "bytes"
    "encoding/json"
    "testing"

    "github.com/sage-x-project/sage/internal/logger"
)

func TestLogging(t *testing.T) {
    // Capture log output
    var buf bytes.Buffer
    log := logger.NewWithWriter(&buf)

    log.Info("test message",
        logger.String("key", "value"),
    )

    // Parse JSON output
    var entry map[string]interface{}
    if err := json.Unmarshal(buf.Bytes(), &entry); err != nil {
        t.Fatal(err)
    }

    // Verify fields
    if entry["msg"] != "test message" {
        t.Errorf("Expected msg='test message', got %v", entry["msg"])
    }
    if entry["key"] != "value" {
        t.Errorf("Expected key='value', got %v", entry["key"])
    }
}
```

## Common Patterns

### HTTP Request Logging

```go
func LogHTTPRequest(log logger.Logger, r *http.Request, status int, duration time.Duration) {
    log.Info("HTTP request",
        logger.String("method", r.Method),
        logger.String("path", r.URL.Path),
        logger.String("remote_addr", r.RemoteAddr),
        logger.Int("status", status),
        logger.Duration("duration", duration),
        logger.String("user_agent", r.UserAgent()),
    )
}
```

### Crypto Operation Logging

```go
func logCryptoOperation(log logger.Logger, operation string, duration time.Duration, err error) {
    if err != nil {
        log.Error("Crypto operation failed",
            logger.String("operation", operation),
            logger.Duration("duration", duration),
            logger.Error(err),
        )
    } else {
        log.Debug("Crypto operation completed",
            logger.String("operation", operation),
            logger.Duration("duration", duration),
        )
    }
}
```

### Session Logging

```go
func logSessionEvent(log logger.Logger, event string, sessionID string, fields ...logger.Field) {
    allFields := make([]logger.Field, 0, len(fields)+2)
    allFields = append(allFields, logger.String("event", event))
    allFields = append(allFields, logger.String("session_id", sessionID))
    allFields = append(allFields, fields...)

    log.Info("Session event", allFields...)
}
```

## Environment Configuration

Set log level via environment variable:

```bash
# Development
export SAGE_LOG_LEVEL=debug
./sage-server

# Production
export SAGE_LOG_LEVEL=info
./sage-server

# Quiet mode
export SAGE_LOG_LEVEL=warn
./sage-server
```

## Integration with Components

### Agent Package

```go
type Agent struct {
    log logger.Logger
}

func NewAgent(log logger.Logger) *Agent {
    return &Agent{
        log: log.WithFields(
            logger.String("component", "agent"),
        ),
    }
}

func (a *Agent) ProcessMessage(msg []byte) error {
    a.log.Debug("Processing message",
        logger.Int("size", len(msg)),
    )
    // ...
}
```

### Middleware

```go
func LoggingMiddleware(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        start := time.Now()

        // Generate request ID
        requestID := generateRequestID()
        ctx := logger.WithRequestID(r.Context(), requestID)

        log := logger.New().WithContext(ctx)
        log.Info("Request started",
            logger.String("method", r.Method),
            logger.String("path", r.URL.Path),
        )

        // Wrap response writer to capture status
        lrw := &loggingResponseWriter{ResponseWriter: w, statusCode: 200}

        next.ServeHTTP(lrw, r.WithContext(ctx))

        log.Info("Request completed",
            logger.Int("status", lrw.statusCode),
            logger.Duration("duration", time.Since(start)),
        )
    })
}
```

## File Structure

```
internal/logger/
├── README.md           # This file
├── logger.go           # Logger interface and implementation
├── fields.go           # Field constructors
├── error.go            # SageError type
└── logger_test.go      # Tests
```

## Related Packages

- `internal/metrics` - Metrics collection (complements logging)
- `pkg/agent` - Uses logger for agent operations
- `pkg/server` - HTTP server logging

## References

- [Structured Logging Best Practices](https://www.thoughtworks.com/insights/blog/structured-logging)
- [The Log/Event Processing Pipeline](https://engineering.linkedin.com/distributed-systems/log-what-every-software-engineer-should-know-about-real-time-datas-unifying)
- [Go context package](https://pkg.go.dev/context)
