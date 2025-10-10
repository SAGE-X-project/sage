# SAGE Load Testing Guide

**Last Updated:** 2025-10-10
**Tool:** k6 (https://k6.io/)
**Version:** v0.48.0+

---

## Table of Contents

1. [Overview](#overview)
2. [Quick Start](#quick-start)
3. [Test Scenarios](#test-scenarios)
4. [Performance Baselines](#performance-baselines)
5. [Continuous Testing](#continuous-testing)
6. [Analysis](#analysis)
7. [Troubleshooting](#troubleshooting)
8. [Best Practices](#best-practices)

---

## Overview

SAGE uses [k6](https://k6.io/) for load testing and performance benchmarking. Our load testing strategy covers four key scenarios:

| Scenario | Purpose | Duration | VUs | Frequency |
|----------|---------|----------|-----|-----------|
| **Baseline** | Performance baselines | ~2 min | 10 | Every push |
| **Stress** | Breaking points | ~15 min | 50-200 | Daily |
| **Spike** | Traffic surges | ~5 min | 20-500 | Weekly |
| **Soak** | Memory leaks | 2-24 hrs | 50 | Monthly |

### Why Load Testing?

-  **Catch Performance Regressions:** Detect slowdowns before production
-  **Capacity Planning:** Understand system limits
-  **Reliability:** Verify error handling under stress
-  **Optimization:** Identify bottlenecks
-  **SLA Validation:** Ensure performance commitments

---

## Quick Start

### Prerequisites

```bash
# Install k6
brew install k6  # macOS
# or
sudo apt-get install k6  # Ubuntu/Debian

# Verify installation
k6 version
```

### Run Your First Test

```bash
# 1. Start SAGE server
go run tests/session/handshake/server/main.go

# 2. In another terminal, run baseline test
./scripts/run-loadtest.sh baseline

# Expected output:
# ✓ http_req_duration..........: avg=45.23ms  p(95)=123.45ms
# ✓ http_req_failed............: 0.00%
# ✓ sage_success_rate..........: 100.00%
```

### Expected Results

**Baseline Test (10 VUs):**
```
Average Response Time: ~50ms
95th Percentile: <500ms
99th Percentile: <1000ms
Error Rate: <1%
Throughput: >100 req/s
```

If your results differ significantly, see [Troubleshooting](#troubleshooting).

---

## Test Scenarios

### 1. Baseline Test

**Purpose:** Establish performance baselines under normal conditions

#### Profile

```javascript
stages: [
  { duration: '30s', target: 10 },  // Ramp up
  { duration: '1m', target: 10 },   // Steady state
  { duration: '30s', target: 0 },   // Ramp down
]
```

#### Thresholds

```javascript
thresholds: {
  http_req_duration: ['p(95)<500', 'p(99)<1000'],
  http_req_failed: ['rate<0.01'],
  http_reqs: ['rate>10'],
}
```

#### Usage

```bash
# Local test
./scripts/run-loadtest.sh baseline

# Against staging
SAGE_BASE_URL=https://staging-api.sage.example.com \
  ./scripts/run-loadtest.sh baseline

# CI environment
SAGE_ENV=ci ./scripts/run-loadtest.sh baseline
```

#### What It Tests

- Agent registration
- HPKE handshake flow
- Message sending (3-5 messages per session)
- Health checks (10% of iterations)

#### Interpretation

| Metric | Good | Acceptable | Poor |
|--------|------|------------|------|
| p(95) latency | <200ms | <500ms | >500ms |
| p(99) latency | <500ms | <1000ms | >1000ms |
| Error rate | <0.1% | <1% | >1% |
| Throughput | >100 req/s | >50 req/s | <50 req/s |

---

### 2. Stress Test

**Purpose:** Find system breaking points and bottlenecks

#### Profile

```javascript
stages: [
  { duration: '2m', target: 50 },    // Warm up
  { duration: '3m', target: 100 },   // Increase load
  { duration: '5m', target: 100 },   // Sustained load
  { duration: '2m', target: 200 },   // Spike
  { duration: '3m', target: 200 },   // Hold spike
  { duration: '2m', target: 100 },   // Recovery
  { duration: '2m', target: 0 },     // Cooldown
]
```

#### Thresholds

```javascript
thresholds: {
  http_req_duration: ['p(95)<1000', 'p(99)<2000'],
  http_req_failed: ['rate<0.05'],
  http_reqs: ['rate>50'],
}
```

#### Usage

```bash
# Run stress test
./scripts/run-loadtest.sh stress

# Monitor during test
watch -n 5 'curl -s http://localhost:8080/debug/health'
```

#### What It Tests

- System behavior under sustained high load
- Error handling when resources are strained
- Recovery after load spike
- Connection pool behavior
- Database performance under pressure

#### Interpretation

**Healthy System:**
- Error rate stays below 5%
- p(95) latency increases but stays under 1s
- System recovers quickly after spike

**Warning Signs:**
- Error rate > 5%
- p(95) latency > 2s
- Slow recovery after spike
- Memory usage climbing

**Action Items if Failed:**
- Check CPU usage (may need more cores)
- Review database query performance
- Increase connection pool size
- Consider horizontal scaling

---

### 3. Spike Test

**Purpose:** Validate resilience to sudden traffic surges

#### Profile

```javascript
stages: [
  { duration: '1m', target: 20 },     // Baseline
  { duration: '30s', target: 500 },   // SPIKE!
  { duration: '2m', target: 500 },    // Hold spike
  { duration: '30s', target: 20 },    // Drop
  { duration: '1m', target: 20 },     // Recovery
]
```

#### Thresholds

```javascript
thresholds: {
  http_req_duration: ['p(95)<2000', 'p(99)<5000'],
  http_req_failed: ['rate<0.10'],
}
```

#### Usage

```bash
./scripts/run-loadtest.sh spike
```

#### What It Tests

- Auto-scaling responsiveness
- Rate limiting effectiveness
- Queue management
- Circuit breaker behavior
- Graceful degradation

#### Interpretation

**Good Performance:**
- Error rate < 10% during spike
- p(95) latency < 2s
- System recovers within 30s after spike

**Poor Performance:**
- Error rate > 20%
- Many timeout errors
- System doesn't recover

**Recommendations:**
- Implement rate limiting
- Add request queue/throttling
- Configure auto-scaling
- Set up traffic spike alerts

---

### 4. Soak Test (Endurance)

**Purpose:** Detect memory leaks and degradation over time

#### Profile

```javascript
stages: [
  { duration: '5m', target: 50 },     // Ramp up
  { duration: '2h', target: 50 },     // Soak (adjustable)
  { duration: '5m', target: 0 },      // Ramp down
]
```

#### Thresholds

```javascript
thresholds: {
  http_req_duration: ['p(95)<500'],  // Should NOT degrade
  http_req_failed: ['rate<0.01'],
}
```

#### Usage

```bash
# 2-hour soak test (default)
./scripts/run-loadtest.sh soak

# 24-hour soak test
SOAK_DURATION=24h ./scripts/run-loadtest.sh soak
```

#### What It Tests

- Memory leaks
- Resource exhaustion
- Database bloat
- Connection pool leaks
- Long-term performance stability

#### Monitoring During Soak Test

**System Metrics (Prometheus/Grafana):**
```promql
# Memory usage (should be stable)
process_resident_memory_bytes

# Active connections (should not grow)
sage_db_connections_active

# Response time (should not increase)
histogram_quantile(0.95, rate(http_request_duration_seconds_bucket[5m]))
```

**Database Checks:**
```sql
-- Check table sizes (should not grow excessively)
SELECT
    schemaname,
    tablename,
    pg_size_pretty(pg_total_relation_size(schemaname||'.'||tablename)) as size
FROM pg_tables
WHERE schemaname = 'public'
ORDER BY pg_total_relation_size(schemaname||'.'||tablename) DESC;

-- Check expired sessions cleanup
SELECT COUNT(*) FROM sessions WHERE expires_at < NOW();
-- Should be 0 or low (cleanup job working)
```

#### Interpretation

**Healthy System:**
- Memory usage stable throughout
- Response times consistent
- No increase in error rate
- Cleanup jobs working (expired data removed)

**Warning Signs:**
- Memory usage steadily increasing
- Response times degrading over time
- Database size growing unbounded
- Connection pool exhausted

**Action Items if Failed:**
- Profile for memory leaks (pprof)
- Review cleanup job effectiveness
- Check for goroutine leaks
- Verify database indexes
- Review connection pool configuration

---

## Performance Baselines

### Target Metrics (Baseline Test)

| Metric | Target | Acceptable | Critical |
|--------|--------|------------|----------|
| **Latency (p50)** | <50ms | <100ms | >200ms |
| **Latency (p95)** | <200ms | <500ms | >1000ms |
| **Latency (p99)** | <500ms | <1000ms | >2000ms |
| **Error Rate** | <0.1% | <1% | >5% |
| **Throughput** | >100 req/s | >50 req/s | <20 req/s |

### Hardware Reference

These baselines were established on:

```
CPU: 4 cores @ 2.4 GHz
RAM: 8 GB
Database: PostgreSQL 14, local
Network: localhost
Go Version: 1.22
```

**Note:** Performance will vary with hardware. Establish your own baselines.

### Tracking Performance Over Time

```bash
# Run baseline and save results
./scripts/run-loadtest.sh baseline

# Compare with previous run
node loadtest/analysis/compare.js \
  loadtest/reports/baseline-2025-10-01.json \
  loadtest/reports/baseline-2025-10-10.json

# Expected output:
# Latency (p95): 423ms → 387ms (↓ 8.5%) 
# Error rate: 0.5% → 0.3% (↓ 40%) 
# Throughput: 87 req/s → 95 req/s (↑ 9.2%) 
```

---

## Continuous Testing

### GitHub Actions Integration

Load tests run automatically:

**Triggers:**
-  Every push to `main` branch (baseline only)
-  Daily at 2 AM UTC (stress test)
-  Manual trigger via workflow_dispatch

**Workflow:**
```yaml
# .github/workflows/loadtest.yml
name: Load Tests

on:
  push:
    branches: [main]
  schedule:
    - cron: '0 2 * * *'
  workflow_dispatch:
    inputs:
      scenario:
        type: choice
        options: [baseline, stress, spike, all]
```

### View Results

**GitHub Actions UI:**
1. Go to Actions tab
2. Select "Load Tests" workflow
3. Click on run to view details
4. Download artifacts (JSON results)

**Download Results Programmatically:**
```bash
# Using GitHub CLI
gh run list --workflow=loadtest.yml
gh run download <run-id>
```

### Setting Up Performance Alerts

**Grafana Alerts:**
```promql
# Alert if p95 latency > 1s
histogram_quantile(0.95, rate(http_request_duration_seconds_bucket[5m])) > 1

# Alert if error rate > 1%
rate(http_requests_total{status=~"5.."}[5m]) / rate(http_requests_total[5m]) > 0.01
```

---

## Analysis

### Reading k6 Output

```
✓ http_req_duration..........: avg=123.45ms min=45.12ms med=98.23ms max=987.65ms p(90)=234.56ms p(95)=345.67ms
✓ http_req_failed............: 0.12% ✓ 12 ✗ 9988
✓ http_reqs..................: 10000 166.67/s
✓ iterations.................: 1000 16.67/s
  sage_handshake_duration....: avg=234.56ms min=123.45ms med=198.76ms max=1234.56ms p(90)=345.67ms p(95)=456.78ms
  sage_sessions_created......: 987 16.45/s
  vus........................: 10 min=0 max=10
  vus_max....................: 10 min=10 max=10
```

**Key Metrics:**
- `avg`: Average response time
- `p(95)`: 95th percentile (95% of requests faster than this)
- `p(99)`: 99th percentile (99% of requests faster than this)
- `http_req_failed`: Percentage of failed requests
- `http_reqs`: Total requests and requests/second

### Analyzing Results

```bash
# Extract key metrics from summary
cat loadtest/reports/baseline-summary.json | jq '.metrics.http_req_duration.values'

# Compare error rates
cat loadtest/reports/baseline-summary.json | jq '.metrics.http_req_failed.values.rate'

# Check if thresholds passed
cat loadtest/reports/baseline-summary.json | jq '.metrics.http_req_duration.thresholds'
```

### Grafana Dashboards

Import k6 dashboard for visualization:

```bash
# Start Grafana
docker-compose --profile monitoring up grafana

# Access: http://localhost:3000
# Import: docker/grafana/dashboards/k6-load-test.json
```

Dashboard shows:
- Request rate over time
- Response time distribution
- Error rate trend
- Virtual users
- Custom SAGE metrics

---

## Troubleshooting

### High Latency

**Symptoms:**
- p(95) > 1s
- Average response time increasing

**Diagnosis:**
```bash
# Check server CPU
top

# Check database performance
psql -h localhost -U sage -d sage -c "
SELECT query, mean_exec_time, calls
FROM pg_stat_statements
ORDER BY mean_exec_time DESC
LIMIT 10;
"

# Check connection pool
curl http://localhost:9090/metrics | grep sage_db_connections
```

**Solutions:**
- Scale vertically (more CPU/RAM)
- Optimize database queries
- Add database indexes
- Increase connection pool size
- Enable query caching

---

### High Error Rate

**Symptoms:**
- Error rate > 1%
- Many 500/503 responses

**Diagnosis:**
```bash
# Check server logs
tail -f logs/sage-server.log | grep ERROR

# Check database connections
psql -h localhost -U sage -d sage -c "
SELECT count(*) FROM pg_stat_activity WHERE datname='sage';
"
```

**Solutions:**
- Check for connection pool exhaustion
- Review error logs for patterns
- Verify database is healthy
- Check for resource limits (ulimit)
- Ensure cleanup jobs are running

---

### k6 Errors

**Problem:** `WARN Request Failed error="dial tcp: connect: connection refused"`

**Solution:**
```bash
# Verify server is running
curl http://localhost:8080/debug/health

# Check port
lsof -i :8080

# Restart server
pkill sage-server
go run tests/session/handshake/server/main.go
```

---

**Problem:** `ERRO thresholds on metrics 'http_req_duration' failed`

**Solution:**
- This means test failed due to slow responses
- Review system resources
- Check for bottlenecks
- Consider adjusting thresholds if unrealistic

---

## Best Practices

### 1. Test in Isolation

```bash
# Stop other services
docker-compose down

# Clean database
psql -h localhost -U sage -d sage -c "TRUNCATE sessions, nonces CASCADE;"

# Restart server fresh
go run tests/session/handshake/server/main.go
```

### 2. Warm Up Period

Always include a ramp-up period to avoid cold start effects:

```javascript
stages: [
  { duration: '30s', target: 10 },  // ← Warm up
  { duration: '1m', target: 10 },   // Actual test
]
```

### 3. Monitor During Tests

Keep these running in separate terminals:

```bash
# Terminal 1: Server logs
go run tests/session/handshake/server/main.go

# Terminal 2: System resources
htop

# Terminal 3: Database monitoring
watch -n 5 'psql -h localhost -U sage -d sage -c "SELECT COUNT(*) FROM sessions;"'

# Terminal 4: Run test
./scripts/run-loadtest.sh baseline
```

### 4. Document Changes

When making performance changes:

```bash
# Before change
./scripts/run-loadtest.sh baseline
cp loadtest/reports/baseline-summary.json baseline-before.json

# After change
./scripts/run-loadtest.sh baseline
cp loadtest/reports/baseline-summary.json baseline-after.json

# Compare
node loadtest/analysis/compare.js baseline-before.json baseline-after.json
```

### 5. Regular Testing

- **Daily:** Automated baseline tests via CI
- **Weekly:** Manual stress test review
- **Monthly:** Full soak test (24h)
- **Before Release:** All scenarios

---

## References

- [k6 Documentation](https://k6.io/docs/)
- [k6 Best Practices](https://k6.io/docs/misc/fine-tuning-os/)
- [Performance Testing Patterns](https://k6.io/docs/test-types/introduction/)
- [SAGE API Documentation](./API.md)
- [SAGE Monitoring](../docker/grafana/dashboards/)

---

**Need Help?**

- GitHub Issues: https://github.com/sage-x-project/sage/issues
- Label: `performance` or `load-testing`
- Include: Test scenario, results JSON, system specs

---

**Last Updated:** 2025-10-10
**Maintainer:** SAGE Team
