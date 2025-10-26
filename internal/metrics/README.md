# Metrics Package

## Overview

The `metrics` package provides comprehensive Prometheus-based metrics collection for SAGE. It tracks cryptographic operations, handshakes, message processing, session management, and server performance.

This package enables monitoring, alerting, and performance analysis of SAGE deployments.

## Features

- **Prometheus Integration**: Native Prometheus metrics export
- **Comprehensive Coverage**: Crypto, handshake, message, session, and server metrics
- **Custom Collector**: High-level metrics aggregation with percentiles
- **HTTP Endpoint**: `/metrics` endpoint for Prometheus scraping
- **Go Runtime Metrics**: Automatic Go runtime and process metrics
- **Type Safety**: Strongly-typed metric labels

## Architecture

```
┌─────────────────────────────────────────────┐
│         SAGE Operations                     │
│    - Crypto operations                      │
│    - Handshakes                             │
│    - Message processing                     │
│    - Session management                     │
└─────────────────┬───────────────────────────┘
                  │
                  ▼
┌─────────────────────────────────────────────┐
│    Metrics Package                          │
│    - Counters (operations, errors)          │
│    - Gauges (active sessions)               │
│    - Histograms (durations, sizes)          │
└─────────────────┬───────────────────────────┘
                  │
                  ▼
┌─────────────────────────────────────────────┐
│    Prometheus Registry                      │
│    - Metric aggregation                     │
│    - Label management                       │
└─────────────────┬───────────────────────────┘
                  │
                  ▼
┌─────────────────────────────────────────────┐
│    /metrics HTTP Endpoint                   │
│    - Prometheus scrape target               │
└─────────────────────────────────────────────┘
```

## Metric Categories

### 1. Cryptographic Operations (`sage_crypto_*`)

Tracks all cryptographic operations:

```
# Operation counts by algorithm
sage_crypto_operations_total{operation="sign",algorithm="ed25519"} 1234
sage_crypto_operations_total{operation="verify",algorithm="secp256k1"} 5678

# Error counts
sage_crypto_errors_total{operation="sign"} 3
sage_crypto_errors_total{operation="verify"} 12

# Operation durations (histogram)
sage_crypto_operation_duration_seconds_bucket{operation="sign",algorithm="ed25519",le="0.001"} 1200
sage_crypto_operation_duration_seconds_sum{operation="sign",algorithm="ed25519"} 0.456
sage_crypto_operation_duration_seconds_count{operation="sign",algorithm="ed25519"} 1234
```

**Labels:**
- `operation`: sign, verify, encrypt, decrypt
- `algorithm`: ed25519, secp256k1, chacha20

### 2. Handshake Metrics (`sage_handshakes_*`)

Tracks handshake lifecycle:

```
# Handshakes initiated
sage_handshakes_initiated_total{role="client"} 100
sage_handshakes_initiated_total{role="server"} 95

# Handshakes completed
sage_handshakes_completed_total{status="success"} 90
sage_handshakes_completed_total{status="failure"} 10

# Failures by error type
sage_handshakes_failed_total{error_type="timeout"} 5
sage_handshakes_failed_total{error_type="invalid"} 3
sage_handshakes_failed_total{error_type="network"} 2

# Duration by stage (histogram)
sage_handshakes_duration_seconds_bucket{stage="init",le="0.1"} 85
sage_handshakes_duration_seconds_bucket{stage="process",le="0.1"} 82
sage_handshakes_duration_seconds_bucket{stage="finalize",le="0.1"} 90
```

**Labels:**
- `role`: client, server
- `status`: success, failure
- `error_type`: timeout, invalid, network
- `stage`: init, process, finalize

### 3. Message Processing (`sage_messages_*`)

Tracks message handling and security:

```
# Messages processed
sage_messages_processed_total{type="text",status="success"} 5000
sage_messages_processed_total{type="binary",status="failure"} 10

# Security: Replay attacks detected
sage_messages_replay_attacks_detected_total 15

# Nonce validations
sage_messages_nonce_validations_total{status="valid"} 4990
sage_messages_nonce_validations_total{status="invalid"} 5
sage_messages_nonce_validations_total{status="expired"} 5

# Processing duration (histogram)
sage_messages_processing_duration_seconds_bucket{le="0.001"} 4500
sage_messages_processing_duration_seconds_sum 2.345
sage_messages_processing_duration_seconds_count 5000

# Message size (histogram)
sage_messages_size_bytes_bucket{le="1024"} 4200
sage_messages_size_bytes_bucket{le="65536"} 4950
sage_messages_size_bytes_sum 2500000
sage_messages_size_bytes_count 5000
```

**Labels:**
- `type`: text, binary
- `status`: success, failure, valid, invalid, expired

### 4. Session Management (`sage_sessions_*`)

Tracks session lifecycle:

```
# Sessions created
sage_sessions_created_total{status="success"} 100
sage_sessions_created_total{status="failure"} 2

# Active sessions (gauge)
sage_sessions_active 45

# Sessions expired
sage_sessions_expired_total 30

# Sessions closed
sage_sessions_closed_total 25

# Session operation duration (histogram)
sage_sessions_duration_seconds_bucket{operation="create",le="0.01"} 95
sage_sessions_duration_seconds_bucket{operation="encrypt",le="0.001"} 4800
sage_sessions_duration_seconds_bucket{operation="decrypt",le="0.001"} 4750

# Message sizes by direction (histogram)
sage_sessions_message_size_bytes_bucket{direction="inbound",le="1024"} 2300
sage_sessions_message_size_bytes_bucket{direction="outbound",le="1024"} 2400
```

**Labels:**
- `status`: success, failure
- `operation`: create, encrypt, decrypt
- `direction`: inbound, outbound

### 5. Go Runtime Metrics

Automatically collected:

```
# Memory
go_memstats_alloc_bytes 45678912
go_memstats_heap_alloc_bytes 42345678
go_memstats_heap_inuse_bytes 48234567

# Goroutines
go_goroutines 42

# GC
go_gc_duration_seconds{quantile="0.5"} 0.000123
go_gc_duration_seconds{quantile="0.95"} 0.000456

# CPU
process_cpu_seconds_total 123.45
```

## Usage

### Recording Metrics

#### Crypto Operations

```go
import (
    "time"
    "github.com/sage-x-project/sage/internal/metrics"
)

func SignMessage(data []byte) ([]byte, error) {
    start := time.Now()

    sig, err := performSign(data)

    // Record operation
    duration := time.Since(start)
    metrics.CryptoOperationDuration.WithLabelValues("sign", "ed25519").Observe(duration.Seconds())

    if err != nil {
        metrics.CryptoErrors.WithLabelValues("sign").Inc()
        return nil, err
    }

    metrics.CryptoOperations.WithLabelValues("sign", "ed25519").Inc()
    return sig, nil
}
```

#### Handshake Tracking

```go
func PerformHandshake(role string) error {
    // Track initiation
    metrics.HandshakesInitiated.WithLabelValues(role).Inc()

    start := time.Now()

    err := doHandshake()

    // Track duration
    metrics.HandshakeDuration.WithLabelValues("init").Observe(time.Since(start).Seconds())

    if err != nil {
        metrics.HandshakesFailed.WithLabelValues("timeout").Inc()
        metrics.HandshakesCompleted.WithLabelValues("failure").Inc()
        return err
    }

    metrics.HandshakesCompleted.WithLabelValues("success").Inc()
    return nil
}
```

#### Message Processing

```go
func ProcessMessage(msg []byte, isText bool) error {
    start := time.Now()

    msgType := "binary"
    if isText {
        msgType = "text"
    }

    // Track message size
    metrics.MessageSize.Observe(float64(len(msg)))

    err := processMsg(msg)

    // Track processing time
    metrics.MessageProcessingDuration.Observe(time.Since(start).Seconds())

    if err != nil {
        metrics.MessagesProcessed.WithLabelValues(msgType, "failure").Inc()
        return err
    }

    metrics.MessagesProcessed.WithLabelValues(msgType, "success").Inc()
    return nil
}
```

#### Session Management

```go
func CreateSession() (string, error) {
    start := time.Now()

    sessionID, err := createNewSession()

    // Track creation duration
    metrics.SessionDuration.WithLabelValues("create").Observe(time.Since(start).Seconds())

    if err != nil {
        metrics.SessionsCreated.WithLabelValues("failure").Inc()
        return "", err
    }

    metrics.SessionsCreated.WithLabelValues("success").Inc()
    metrics.SessionsActive.Inc()

    return sessionID, nil
}

func CloseSession() {
    metrics.SessionsActive.Dec()
    metrics.SessionsClosed.Inc()
}
```

#### Security Events

```go
func ValidateNonce(nonce string) error {
    if isExpired(nonce) {
        metrics.NonceValidations.WithLabelValues("expired").Inc()
        return ErrNonceExpired
    }

    if !isValid(nonce) {
        metrics.NonceValidations.WithLabelValues("invalid").Inc()
        return ErrNonceInvalid
    }

    metrics.NonceValidations.WithLabelValues("valid").Inc()
    return nil
}

func DetectReplayAttack(msgID string) bool {
    if isDuplicate(msgID) {
        metrics.ReplayAttacksDetected.Inc()
        return true
    }
    return false
}
```

### Custom Metrics Collector

The package provides a custom collector for high-level metrics:

```go
import "github.com/sage-x-project/sage/internal/metrics"

func main() {
    collector := metrics.GetGlobalCollector()

    // Record operations
    collector.RecordSignature(150 * time.Microsecond)
    collector.RecordVerification(true, 200 * time.Microsecond)
    collector.RecordDIDResolution(true, 500 * time.Microsecond)
    collector.RecordBlockchainCall(true, 2 * time.Second)

    // Get snapshot
    snapshot := collector.GetSnapshot()

    fmt.Printf("Uptime: %v\n", snapshot.Uptime)
    fmt.Printf("Signatures: %d\n", snapshot.SignatureCount)
    fmt.Printf("Avg signature time: %.2f µs\n", snapshot.AvgSignatureTime)
    fmt.Printf("P95 signature time: %d µs\n", snapshot.P95SignatureTime)
    fmt.Printf("Cache hit rate: %.2f%%\n", snapshot.GetCacheHitRate())
    fmt.Printf("Verification success rate: %.2f%%\n", snapshot.GetVerificationSuccessRate())
}
```

### HTTP Metrics Endpoint

#### Embedded in Existing Server

```go
import (
    "net/http"
    "github.com/sage-x-project/sage/internal/metrics"
)

func main() {
    mux := http.NewServeMux()

    // Application routes
    mux.HandleFunc("/api/health", healthHandler)
    mux.HandleFunc("/api/sessions", sessionsHandler)

    // Metrics endpoint
    mux.Handle("/metrics", metrics.Handler())

    http.ListenAndServe(":8080", mux)
}
```

#### Standalone Metrics Server

```go
import "github.com/sage-x-project/sage/internal/metrics"

func main() {
    // Run metrics server on separate port
    go metrics.StartServer(":9090")

    // Main application
    runApp()
}
```

## Prometheus Configuration

### Scrape Configuration

Add to `prometheus.yml`:

```yaml
scrape_configs:
  - job_name: 'sage'
    scrape_interval: 15s
    static_configs:
      - targets: ['localhost:8080']
    metrics_path: '/metrics'
```

### Example Queries

```promql
# Request rate
rate(sage_messages_processed_total[5m])

# Error rate
rate(sage_crypto_errors_total[5m]) / rate(sage_crypto_operations_total[5m])

# P95 signature latency
histogram_quantile(0.95, rate(sage_crypto_operation_duration_seconds_bucket{operation="sign"}[5m]))

# Active sessions trend
sage_sessions_active

# Handshake success rate
rate(sage_handshakes_completed_total{status="success"}[5m]) /
rate(sage_handshakes_initiated_total[5m])

# Replay attack rate
rate(sage_messages_replay_attacks_detected_total[5m])

# Message size P99
histogram_quantile(0.99, rate(sage_messages_size_bytes_bucket[5m]))
```

## Grafana Dashboards

### Key Metrics Dashboard

```json
{
  "dashboard": {
    "title": "SAGE Metrics",
    "panels": [
      {
        "title": "Request Rate",
        "targets": [
          {
            "expr": "rate(sage_messages_processed_total[5m])"
          }
        ]
      },
      {
        "title": "Error Rate",
        "targets": [
          {
            "expr": "rate(sage_crypto_errors_total[5m])"
          }
        ]
      },
      {
        "title": "Active Sessions",
        "targets": [
          {
            "expr": "sage_sessions_active"
          }
        ]
      },
      {
        "title": "P95 Latency",
        "targets": [
          {
            "expr": "histogram_quantile(0.95, rate(sage_crypto_operation_duration_seconds_bucket[5m]))"
          }
        ]
      }
    ]
  }
}
```

## Alerting Rules

### Example Alert Rules

```yaml
groups:
  - name: sage_alerts
    interval: 30s
    rules:
      # High error rate
      - alert: HighErrorRate
        expr: |
          rate(sage_crypto_errors_total[5m]) / rate(sage_crypto_operations_total[5m]) > 0.05
        for: 5m
        labels:
          severity: warning
        annotations:
          summary: "High crypto error rate"
          description: "Crypto error rate is {{ $value | humanizePercentage }}"

      # Replay attack detected
      - alert: ReplayAttackDetected
        expr: rate(sage_messages_replay_attacks_detected_total[1m]) > 0
        for: 1m
        labels:
          severity: critical
        annotations:
          summary: "Replay attack detected"
          description: "{{ $value }} replay attacks per second"

      # High handshake failure rate
      - alert: HighHandshakeFailureRate
        expr: |
          rate(sage_handshakes_completed_total{status="failure"}[5m]) /
          rate(sage_handshakes_initiated_total[5m]) > 0.10
        for: 5m
        labels:
          severity: warning
        annotations:
          summary: "High handshake failure rate"
          description: "{{ $value | humanizePercentage }} of handshakes failing"

      # Too many active sessions
      - alert: TooManyActiveSessions
        expr: sage_sessions_active > 1000
        for: 5m
        labels:
          severity: warning
        annotations:
          summary: "Too many active sessions"
          description: "{{ $value }} active sessions"

      # Slow crypto operations
      - alert: SlowCryptoOperations
        expr: |
          histogram_quantile(0.95,
            rate(sage_crypto_operation_duration_seconds_bucket[5m])
          ) > 0.1
        for: 5m
        labels:
          severity: warning
        annotations:
          summary: "Slow crypto operations"
          description: "P95 crypto latency is {{ $value }}s"
```

## Best Practices

### 1. Consistent Labeling

```go
// ✅ Correct - consistent label values
metrics.CryptoOperations.WithLabelValues("sign", "ed25519").Inc()
metrics.CryptoOperations.WithLabelValues("verify", "ed25519").Inc()

// ❌ Wrong - inconsistent label values
metrics.CryptoOperations.WithLabelValues("SIGN", "Ed25519").Inc()  // Wrong case
metrics.CryptoOperations.WithLabelValues("signing", "ed25519").Inc()  // Different term
```

### 2. Avoid High Cardinality Labels

```go
// ❌ Wrong - unbounded label values
metrics.SomeMetric.WithLabelValues(userID).Inc()  // Could be millions of users
metrics.SomeMetric.WithLabelValues(sessionID).Inc()  // Thousands of sessions

// ✅ Correct - bounded label values
metrics.SomeMetric.WithLabelValues("user_activity").Inc()
metrics.SomeMetric.WithLabelValues("session_created").Inc()
```

### 3. Use Appropriate Metric Types

```go
// ✅ Correct
metrics.SessionsActive.Inc()                    // Gauge - can go up/down
metrics.SessionsCreated.Inc()                   // Counter - only increases
metrics.SessionDuration.Observe(duration)       // Histogram - distribution

// ❌ Wrong
metrics.SessionsActive.Observe(45)              // Should be Gauge, not Histogram
metrics.SessionDuration.Inc()                   // Should be Histogram, not Counter
```

### 4. Timing Measurements

```go
// ✅ Correct - measure around operation
start := time.Now()
result, err := performOperation()
metrics.OperationDuration.Observe(time.Since(start).Seconds())

// ❌ Wrong - missing failure case
start := time.Now()
result, err := performOperation()
if err == nil {  // Only records successful operations!
    metrics.OperationDuration.Observe(time.Since(start).Seconds())
}
```

## Performance Considerations

### Metric Collection Overhead

```go
// Minimal overhead - direct counter increment
metrics.Counter.Inc()  // ~100ns

// Low overhead - labeled counter
metrics.CounterVec.WithLabelValues("label").Inc()  // ~200ns

// Moderate overhead - histogram observation
metrics.Histogram.Observe(value)  // ~500ns

// Cache label combinations
counter := metrics.CounterVec.WithLabelValues("frequent", "labels")
for i := 0; i < 1000; i++ {
    counter.Inc()  // Faster - no label lookup
}
```

### Memory Usage

```go
// Each unique label combination creates a new time series
// Cardinality = product of all label value counts

// ✅ Low cardinality (3 operations × 3 algorithms = 9 time series)
metrics.CryptoOperations.WithLabelValues(operation, algorithm).Inc()

// ❌ High cardinality (unlimited)
metrics.SomeMetric.WithLabelValues(userID, sessionID, requestID).Inc()
```

## Testing

```go
package mypackage_test

import (
    "testing"
    "github.com/prometheus/client_golang/prometheus/testutil"
    "github.com/sage-x-project/sage/internal/metrics"
)

func TestMetrics(t *testing.T) {
    // Reset metrics before test
    metrics.Registry.Unregister(metrics.CryptoOperations)
    metrics.Registry.MustRegister(metrics.CryptoOperations)

    // Perform operation
    metrics.CryptoOperations.WithLabelValues("sign", "ed25519").Inc()

    // Verify metric
    count := testutil.ToFloat64(
        metrics.CryptoOperations.WithLabelValues("sign", "ed25519"),
    )

    if count != 1 {
        t.Errorf("Expected count=1, got %v", count)
    }
}
```

## File Structure

```
internal/metrics/
├── README.md           # This file
├── registry.go         # Prometheus registry setup
├── collector.go        # Custom metrics collector
├── crypto.go           # Crypto operation metrics
├── handshake.go        # Handshake metrics
├── message.go          # Message processing metrics
├── session.go          # Session management metrics
├── server.go           # HTTP metrics server
└── verify_test.go      # Tests
```

## Related Packages

- `internal/logger` - Structured logging (complements metrics)
- `pkg/agent` - Uses metrics for agent operations
- `pkg/server` - Exposes metrics endpoint

## References

- [Prometheus Documentation](https://prometheus.io/docs/)
- [Prometheus Best Practices](https://prometheus.io/docs/practices/naming/)
- [Grafana Dashboards](https://grafana.com/docs/grafana/latest/dashboards/)
- [Go Prometheus Client](https://github.com/prometheus/client_golang)
