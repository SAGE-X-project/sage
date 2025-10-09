# SAGE Monitoring & Observability Implementation

**Date:** 2025-10-08
**Status:** Implementation Plan
**Priority:** HIGH (Production Essential)

---

## 1. Architecture Overview

### 1.1 Monitoring Stack

```
┌─────────────────────────────────────────────────┐
│                 SAGE Backend                     │
│  ┌──────────┐  ┌──────────┐  ┌──────────┐      │
│  │ Session  │  │Handshake │  │  Crypto  │      │
│  │ Metrics  │  │ Metrics  │  │ Metrics  │      │
│  └─────┬────┘  └─────┬────┘  └─────┬────┘      │
│        │             │              │           │
│        └─────────────┴──────────────┘           │
│                      │                          │
│            ┌─────────▼─────────┐                │
│            │  Metrics Registry │                │
│            │   (Prometheus)    │                │
│            └─────────┬─────────┘                │
│                      │                          │
│            ┌─────────▼─────────┐                │
│            │  /metrics HTTP    │                │
│            │    Endpoint       │                │
│            └─────────┬─────────┘                │
└──────────────────────┼─────────────────────────┘
                       │
            ┌──────────▼──────────┐
            │   Prometheus        │
            │   (Scraper)         │
            └──────────┬──────────┘
                       │
            ┌──────────▼──────────┐
            │   Grafana           │
            │  (Visualization)    │
            └─────────────────────┘
```

### 1.2 Key Components

1. **Metrics Collection** - Prometheus client library
2. **Metrics Registry** - Central registry for all metrics
3. **HTTP Endpoints** - `/metrics` for Prometheus scraping
4. **Dashboard** - Grafana for visualization
5. **Alerts** (Future) - Alertmanager for notifications

---

## 2. Metrics Design

### 2.1 SAGE-Specific Metrics

#### Session Metrics
```
# Counter
sage_sessions_created_total{status="success|failure"}
sage_sessions_expired_total
sage_sessions_closed_total

# Gauge
sage_sessions_active

# Histogram
sage_session_duration_seconds{operation="create|encrypt|decrypt"}
sage_session_message_size_bytes{direction="inbound|outbound"}
```

#### Handshake Metrics
```
# Counter
sage_handshakes_initiated_total{role="client|server"}
sage_handshakes_completed_total{status="success|failure"}
sage_handshakes_failed_total{error_type="timeout|invalid|network"}

# Histogram
sage_handshake_duration_seconds{stage="init|process|finalize"}
```

#### Crypto Metrics
```
# Counter
sage_crypto_operations_total{operation="sign|verify|encrypt|decrypt", algorithm="ed25519|secp256k1|chacha20"}
sage_crypto_errors_total{operation="sign|verify|encrypt|decrypt"}

# Histogram
sage_crypto_operation_duration_seconds{operation="sign|verify|encrypt|decrypt"}
```

#### DID Metrics
```
# Counter
sage_did_resolutions_total{chain="ethereum|solana", status="success|failure"}
sage_did_registrations_total{chain="ethereum|solana", status="success|failure"}

# Histogram
sage_did_resolution_duration_seconds{chain="ethereum|solana"}
```

#### Message Processing Metrics
```
# Counter
sage_messages_processed_total{type="text|binary", status="success|failure"}
sage_replay_attacks_detected_total
sage_nonce_validations_total{status="valid|invalid|expired"}

# Histogram
sage_message_processing_duration_seconds
sage_message_size_bytes
```

### 2.2 Standard Go Metrics

```
# Process metrics
process_cpu_seconds_total
process_resident_memory_bytes
process_open_fds
go_goroutines
go_memstats_alloc_bytes
go_memstats_heap_inuse_bytes
```

---

## 3. Implementation Plan

### 3.1 Phase 1: Core Infrastructure (2-3 hours)

**Step 1: Add Dependencies**
```bash
go get github.com/prometheus/client_golang/prometheus
go get github.com/prometheus/client_golang/prometheus/promauto
go get github.com/prometheus/client_golang/prometheus/promhttp
```

**Step 2: Create Metrics Package**
```
internal/metrics/
├── registry.go      # Central metrics registry
├── session.go       # Session-specific metrics
├── handshake.go     # Handshake metrics
├── crypto.go        # Crypto metrics
├── did.go           # DID metrics
├── message.go       # Message processing metrics
└── server.go        # HTTP server for /metrics
```

**Step 3: Define Core Metrics**
```go
// internal/metrics/registry.go
package metrics

import (
    "github.com/prometheus/client_golang/prometheus"
    "github.com/prometheus/client_golang/prometheus/promauto"
)

var (
    // Namespace for all SAGE metrics
    namespace = "sage"

    // Registry holds all metrics
    Registry = prometheus.NewRegistry()
)

func init() {
    // Register Go runtime metrics
    Registry.MustRegister(prometheus.NewGoCollector())
    Registry.MustRegister(prometheus.NewProcessCollector(prometheus.ProcessCollectorOpts{}))
}
```

### 3.2 Phase 2: Session Metrics (1 hour)

**File:** `internal/metrics/session.go`

```go
package metrics

import (
    "github.com/prometheus/client_golang/prometheus"
    "github.com/prometheus/client_golang/prometheus/promauto"
)

var (
    SessionsCreated = promauto.With(Registry).NewCounterVec(
        prometheus.CounterOpts{
            Namespace: namespace,
            Subsystem: "sessions",
            Name:      "created_total",
            Help:      "Total number of sessions created",
        },
        []string{"status"}, // success, failure
    )

    SessionsActive = promauto.With(Registry).NewGauge(
        prometheus.GaugeOpts{
            Namespace: namespace,
            Subsystem: "sessions",
            Name:      "active",
            Help:      "Number of currently active sessions",
        },
    )

    SessionDuration = promauto.With(Registry).NewHistogramVec(
        prometheus.HistogramOpts{
            Namespace: namespace,
            Subsystem: "sessions",
            Name:      "duration_seconds",
            Help:      "Session operation duration in seconds",
            Buckets:   prometheus.ExponentialBuckets(0.0001, 2, 15), // 0.1ms to 1.6s
        },
        []string{"operation"}, // create, encrypt, decrypt
    )

    SessionMessageSize = promauto.With(Registry).NewHistogramVec(
        prometheus.HistogramOpts{
            Namespace: namespace,
            Subsystem: "sessions",
            Name:      "message_size_bytes",
            Help:      "Size of messages processed by sessions",
            Buckets:   prometheus.ExponentialBuckets(64, 4, 10), // 64B to 16MB
        },
        []string{"direction"}, // inbound, outbound
    )
)
```

**Integration:**
```go
// session/manager.go - in CreateSession()
func (m *Manager) CreateSession(sessionID string, sharedSecret []byte) (Session, error) {
    start := time.Now()

    sess, err := m.CreateSessionWithConfig(sessionID, sharedSecret, m.defaultConfig)

    // Record metrics
    if err != nil {
        metrics.SessionsCreated.WithLabelValues("failure").Inc()
    } else {
        metrics.SessionsCreated.WithLabelValues("success").Inc()
        metrics.SessionsActive.Inc()
    }

    metrics.SessionDuration.WithLabelValues("create").Observe(time.Since(start).Seconds())

    return sess, err
}

// session/session.go - in Encrypt()
func (s *SecureSession) Encrypt(plaintext []byte) ([]byte, error) {
    start := time.Now()
    defer func() {
        metrics.SessionDuration.WithLabelValues("encrypt").Observe(time.Since(start).Seconds())
        metrics.SessionMessageSize.WithLabelValues("outbound").Observe(float64(len(plaintext)))
    }()

    // ... existing encryption logic
}
```

### 3.3 Phase 3: Handshake Metrics (1 hour)

**File:** `internal/metrics/handshake.go`

```go
package metrics

var (
    HandshakesInitiated = promauto.With(Registry).NewCounterVec(
        prometheus.CounterOpts{
            Namespace: namespace,
            Subsystem: "handshakes",
            Name:      "initiated_total",
            Help:      "Total number of handshakes initiated",
        },
        []string{"role"}, // client, server
    )

    HandshakesCompleted = promauto.With(Registry).NewCounterVec(
        prometheus.CounterOpts{
            Namespace: namespace,
            Subsystem: "handshakes",
            Name:      "completed_total",
            Help:      "Total number of handshakes completed",
        },
        []string{"status"}, // success, failure
    )

    HandshakeDuration = promauto.With(Registry).NewHistogramVec(
        prometheus.HistogramOpts{
            Namespace: namespace,
            Subsystem: "handshakes",
            Name:      "duration_seconds",
            Help:      "Handshake stage duration in seconds",
            Buckets:   prometheus.ExponentialBuckets(0.001, 2, 12), // 1ms to 4s
        },
        []string{"stage"}, // init, process, finalize
    )
)
```

### 3.4 Phase 4: HTTP Metrics Endpoint (30 min)

**File:** `internal/metrics/server.go`

```go
package metrics

import (
    "net/http"
    "github.com/prometheus/client_golang/prometheus/promhttp"
)

// Handler returns HTTP handler for Prometheus metrics
func Handler() http.Handler {
    return promhttp.HandlerFor(Registry, promhttp.HandlerOpts{
        EnableOpenMetrics: true,
    })
}

// StartServer starts standalone metrics server
func StartServer(addr string) error {
    http.Handle("/metrics", Handler())
    return http.ListenAndServe(addr, nil)
}
```

**Integration in main application:**
```go
// cmd/sage/main.go or appropriate entry point
import "github.com/sage-x-project/sage/internal/metrics"

func main() {
    // ... other initialization

    // Start metrics endpoint
    go func() {
        if err := metrics.StartServer(":9090"); err != nil {
            log.Printf("Metrics server error: %v", err)
        }
    }()

    // ... rest of application
}
```

---

## 4. Grafana Dashboard Configuration

### 4.1 Dashboard JSON

**File:** `docker/grafana/dashboards/sage-overview.json`

```json
{
  "dashboard": {
    "title": "SAGE Overview",
    "panels": [
      {
        "title": "Active Sessions",
        "targets": [{
          "expr": "sage_sessions_active"
        }],
        "type": "graph"
      },
      {
        "title": "Session Creation Rate",
        "targets": [{
          "expr": "rate(sage_sessions_created_total[5m])"
        }],
        "type": "graph"
      },
      {
        "title": "Handshake Success Rate",
        "targets": [{
          "expr": "rate(sage_handshakes_completed_total{status=\"success\"}[5m]) / rate(sage_handshakes_completed_total[5m])"
        }],
        "type": "graph"
      },
      {
        "title": "Encryption Latency (p95)",
        "targets": [{
          "expr": "histogram_quantile(0.95, rate(sage_session_duration_seconds_bucket{operation=\"encrypt\"}[5m]))"
        }],
        "type": "graph"
      }
    ]
  }
}
```

### 4.2 Key Queries

**Session Health:**
```promql
# Active sessions
sage_sessions_active

# Session creation rate (per second)
rate(sage_sessions_created_total[1m])

# Session failure rate
rate(sage_sessions_created_total{status="failure"}[5m]) /
rate(sage_sessions_created_total[5m])
```

**Performance:**
```promql
# p50/p95/p99 encryption latency
histogram_quantile(0.50, rate(sage_session_duration_seconds_bucket{operation="encrypt"}[5m]))
histogram_quantile(0.95, rate(sage_session_duration_seconds_bucket{operation="encrypt"}[5m]))
histogram_quantile(0.99, rate(sage_session_duration_seconds_bucket{operation="encrypt"}[5m]))

# Throughput (messages/sec)
rate(sage_messages_processed_total[1m])

# Average message size
rate(sage_session_message_size_bytes_sum[5m]) /
rate(sage_session_message_size_bytes_count[5m])
```

**Security:**
```promql
# Replay attacks detected
rate(sage_replay_attacks_detected_total[5m])

# Nonce validation failure rate
rate(sage_nonce_validations_total{status="invalid"}[5m]) /
rate(sage_nonce_validations_total[5m])
```

---

## 5. Alert Rules

### 5.1 Critical Alerts

**File:** `docker/prometheus/alerts/critical.yml`

```yaml
groups:
  - name: sage_critical
    interval: 30s
    rules:
      # High failure rate
      - alert: HighSessionFailureRate
        expr: |
          rate(sage_sessions_created_total{status="failure"}[5m]) /
          rate(sage_sessions_created_total[5m]) > 0.1
        for: 5m
        labels:
          severity: critical
        annotations:
          summary: "High session creation failure rate"
          description: "{{ $value | humanizePercentage }} of sessions are failing"

      # Handshake failures
      - alert: HandshakeFailures
        expr: |
          rate(sage_handshakes_completed_total{status="failure"}[5m]) > 10
        for: 2m
        labels:
          severity: warning
        annotations:
          summary: "High handshake failure rate"
          description: "{{ $value }} handshake failures per second"

      # High latency
      - alert: HighEncryptionLatency
        expr: |
          histogram_quantile(0.95,
            rate(sage_session_duration_seconds_bucket{operation="encrypt"}[5m])
          ) > 0.010
        for: 5m
        labels:
          severity: warning
        annotations:
          summary: "High encryption latency"
          description: "p95 latency is {{ $value }}s (>10ms)"

      # Replay attacks
      - alert: ReplayAttacksDetected
        expr: rate(sage_replay_attacks_detected_total[1m]) > 0
        for: 1m
        labels:
          severity: critical
        annotations:
          summary: "Replay attacks detected"
          description: "{{ $value }} replay attacks per second"
```

---

## 6. Testing Strategy

### 6.1 Metrics Testing

**File:** `internal/metrics/metrics_test.go`

```go
package metrics_test

import (
    "testing"
    "github.com/prometheus/client_golang/prometheus/testutil"
    "github.com/sage-x-project/sage/internal/metrics"
)

func TestSessionMetrics(t *testing.T) {
    // Record a successful session creation
    metrics.SessionsCreated.WithLabelValues("success").Inc()

    // Verify counter increased
    count := testutil.ToFloat64(metrics.SessionsCreated.WithLabelValues("success"))
    if count != 1 {
        t.Errorf("Expected 1, got %f", count)
    }
}

func TestMetricsEndpoint(t *testing.T) {
    // Test that metrics endpoint returns valid data
    handler := metrics.Handler()

    // Make HTTP request to handler
    // Verify Prometheus format output
}
```

### 6.2 Integration Testing

```bash
# Start services with Docker Compose
docker-compose up -d

# Generate some traffic
go run examples/client/main.go

# Check metrics endpoint
curl http://localhost:9090/metrics | grep sage_

# Verify Prometheus is scraping
curl http://localhost:9091/api/v1/targets

# Check Grafana dashboard
open http://localhost:3000
```

---

## 7. Deployment Checklist

### 7.1 Code Changes
- [ ] Add Prometheus client library to go.mod
- [ ] Implement metrics package (registry, session, handshake, crypto, did, message)
- [ ] Add metrics calls to all critical paths
- [ ] Implement /metrics HTTP endpoint
- [ ] Add metrics server to main()

### 7.2 Infrastructure
- [ ] Update docker-compose.yml with Prometheus container
- [ ] Update docker-compose.yml with Grafana container
- [ ] Configure Prometheus scrape targets
- [ ] Import Grafana dashboards
- [ ] Set up alert rules (optional)

### 7.3 Documentation
- [ ] Document available metrics
- [ ] Create dashboard screenshots
- [ ] Write monitoring runbook
- [ ] Add troubleshooting guide

### 7.4 Testing
- [ ] Unit tests for metrics collection
- [ ] Integration test with Prometheus
- [ ] Load test to verify metrics accuracy
- [ ] Verify alerts fire correctly

---

## 8. Implementation Timeline

### Day 1 (Morning): Infrastructure Setup
- **Hour 1-2:** Add dependencies, create metrics package structure
- **Hour 2-3:** Implement core session metrics
- **Hour 3-4:** Implement handshake metrics

### Day 1 (Afternoon): Integration
- **Hour 1-2:** Add crypto and DID metrics
- **Hour 2-3:** Implement HTTP endpoint, integrate with main app
- **Hour 3-4:** Testing and bug fixes

### Day 2 (If needed): Dashboards & Alerts
- **Hour 1-2:** Create Grafana dashboards
- **Hour 2-3:** Configure alert rules
- **Hour 3-4:** Documentation

---

## 9. Success Criteria

### Must Have Yes
- [ ] All critical operations instrumented with metrics
- [ ] /metrics endpoint accessible and returning data
- [ ] Prometheus successfully scraping metrics
- [ ] Basic Grafana dashboard showing key metrics
- [ ] No performance degradation from metrics collection

### Should Have Target
- [ ] Alert rules for critical failures
- [ ] Complete dashboard with all subsystems
- [ ] Metrics documentation
- [ ] Integration tests

### Nice to Have Star
- [ ] Custom metrics for specific use cases
- [ ] Alertmanager integration
- [ ] Multiple dashboards (overview, detailed, debugging)
- [ ] Tracing integration (OpenTelemetry)

---

## 10. Maintenance Plan

### Regular Tasks
- **Daily:** Check alert firing, review dashboards
- **Weekly:** Review metric cardinality, optimize queries
- **Monthly:** Update dashboards based on usage patterns
- **Quarterly:** Review and update alert thresholds

### Metric Lifecycle
1. **Add:** New metrics when adding features
2. **Deprecate:** Mark old metrics as deprecated
3. **Remove:** Delete unused metrics after 1 release

---

**Status:** List READY TO IMPLEMENT
**Estimated Effort:** 6-8 hours (1 day)
**Priority:** HIGH
**Next Step:** Start Phase 1 - Core Infrastructure
