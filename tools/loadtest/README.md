# SAGE Load Testing

Comprehensive load testing suite for SAGE using [k6](https://k6.io/).

## Quick Start

### Install k6

```bash
# macOS
brew install k6

# Linux (Debian/Ubuntu)
sudo apt-key adv --keyserver hkp://keyserver.ubuntu.com:80 --recv-keys C5AD17C747E3415A3642D57D77C6C491D6AC1D69
echo "deb https://dl.k6.io/deb stable main" | sudo tee /etc/apt/sources.list.d/k6.list
sudo apt-get update
sudo apt-get install k6

# Windows (Chocolatey)
choco install k6
```

### Run Tests

```bash
# Start SAGE server first
go run tests/session/handshake/server/main.go

# Run baseline test
./scripts/run-loadtest.sh baseline

# Run stress test
./scripts/run-loadtest.sh stress

# Run spike test
./scripts/run-loadtest.sh spike

# Run soak test (long-running, 2+ hours)
./scripts/run-loadtest.sh soak

# Run all tests
./scripts/run-loadtest.sh all
```

## Test Scenarios

### 1. Baseline Test

**Purpose:** Establish performance baselines under normal load

**Profile:**
- **VUs:** 10 concurrent users
- **Duration:** ~2 minutes
- **Pattern:** Ramp up → Steady → Ramp down

**Thresholds:**
- 95% of requests < 500ms
- 99% of requests < 1s
- Error rate < 1%

**Run:**
```bash
./scripts/run-loadtest.sh baseline
```

**Use Case:**
- Daily regression testing
- Performance baseline documentation
- Pre-deployment validation

---

### 2. Stress Test

**Purpose:** Identify system breaking points

**Profile:**
- **VUs:** Ramps up to 200 users
- **Duration:** ~15 minutes
- **Pattern:** Gradual increase → Peak → Spike → Recovery

**Phases:**
1. Ramp to 50 users (2min)
2. Ramp to 100 users (3min)
3. Hold at 100 (5min)
4. Spike to 200 (2min)
5. Hold spike (3min)
6. Recovery (4min)

**Thresholds:**
- 95% of requests < 1s
- 99% of requests < 2s
- Error rate < 5%

**Run:**
```bash
./scripts/run-loadtest.sh stress
```

**Use Case:**
- Capacity planning
- Finding bottlenecks
- Validating error handling

---

### 3. Spike Test

**Purpose:** Test resilience to sudden traffic surges

**Profile:**
- **VUs:** 20 → 500 → 20 users
- **Duration:** ~5 minutes
- **Pattern:** Baseline → Sudden spike → Recovery

**Phases:**
1. Baseline: 20 users (1min)
2. **SPIKE:** 500 users in 30 seconds
3. Hold spike (2min)
4. Drop to baseline (30s)
5. Recovery period (1.5min)

**Thresholds:**
- 95% of requests < 2s (more lenient during spike)
- 99% of requests < 5s
- Error rate < 10%

**Run:**
```bash
./scripts/run-loadtest.sh spike
```

**Use Case:**
- Auto-scaling validation
- Rate limiting testing
- Traffic surge preparedness

---

### 4. Soak Test (Endurance)

**Purpose:** Detect memory leaks and performance degradation over time

**Profile:**
- **VUs:** 50 concurrent users
- **Duration:** 2-24 hours (configurable)
- **Pattern:** Ramp up → Long steady period → Ramp down

**Thresholds:**
- Performance should NOT degrade over time
- 95% of requests < 500ms (throughout)
- Error rate < 1%

**Run:**
```bash
# 2-hour soak test
./scripts/run-loadtest.sh soak

# 24-hour soak test
SOAK_DURATION=24h ./scripts/run-loadtest.sh soak
```

**Use Case:**
- Memory leak detection
- Long-term stability validation
- Resource exhaustion testing

**Monitor During Test:**
- Memory usage (should be stable)
- Database size (check cleanup jobs)
- Connection pool metrics
- Response times (should not increase)

---

## Test Results

Results are saved to `tools/loadtest/reports/`:

> **Note:** The `reports/` directory is auto-created during test execution and excluded from Git (`.gitignore`). Results are stored locally for analysis but not committed to the repository.

```
tools/loadtest/reports/
├── baseline-results.json       # Full k6 metrics
├── baseline-summary.json       # Summary only
├── stress-results.json
├── stress-summary.json
├── spike-results.json
├── spike-summary.json
├── soak-results.json
└── soak-summary.json
```

### View Results

```bash
# View summary
cat tools/loadtest/reports/baseline-summary.json | jq .

# View detailed metrics
cat tools/loadtest/reports/baseline-results.json | jq '.metrics'

# Extract specific metric
cat tools/loadtest/reports/baseline-summary.json | jq '.metrics.http_req_duration'
```

---

## Configuration

### Environment Variables

```bash
# Base URL (default: http://localhost:8080)
export SAGE_BASE_URL=https://staging-api.sage.example.com

# Environment (local, staging, production)
export SAGE_ENV=staging

# Soak test duration (default: 2h)
export SOAK_DURATION=24h
```

### Custom Thresholds

Edit `tools/loadtest/config.js`:

```javascript
thresholds: {
  baseline: {
    http_req_duration: ['p(95)<500', 'p(99)<1000'],
    http_req_failed: ['rate<0.01'],
  },
}
```

---

## Metrics

### Built-in k6 Metrics

- `http_req_duration`: Request duration
- `http_req_waiting`: Time waiting for response
- `http_req_sending`: Time sending request
- `http_req_receiving`: Time receiving response
- `http_req_failed`: Failed requests rate
- `http_reqs`: Total HTTP requests
- `iterations`: Total iterations
- `vus`: Virtual users
- `vus_max`: Max virtual users

### Custom SAGE Metrics

- `sage_handshake_duration`: Handshake operation duration
- `sage_message_duration`: Message send duration
- `sage_signature_verify_duration`: Signature verification duration
- `sage_success_rate`: Overall success rate
- `sage_sessions_created`: Number of sessions created
- `sage_messages_per_session`: Messages per session trend

---

## Continuous Integration

### GitHub Actions

Load tests run automatically:

**Schedule:**
- Baseline: Every push to main
- Stress: Daily at 2 AM
- Spike: Weekly (Sunday)
- Soak: Monthly (first Monday)

**Workflow:**
```yaml
# .github/workflows/loadtest.yml
name: Load Tests

on:
  push:
    branches: [main]
  schedule:
    - cron: '0 2 * * *'  # Daily at 2 AM

jobs:
  baseline:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: grafana/k6-action@v0.3.1
        with:
          filename: tools/loadtest/scenarios/baseline.js
```

---

## Analysis Tools

### Grafana Dashboard

Import the k6 dashboard:

1. Start Grafana: `docker-compose --profile monitoring up grafana`
2. Access: http://localhost:3000
3. Import dashboard: `docker/grafana/dashboards/k6-load-test.json`

### Prometheus Metrics

k6 can export to Prometheus:

```bash
k6 run --out experimental-prometheus-rw \
  tools/loadtest/scenarios/baseline.js
```

---

## Best Practices

### 1. Prepare Test Environment

```bash
# Clean database before tests
psql -h localhost -U sage -d sage -c "TRUNCATE sessions, nonces CASCADE;"

# Restart server
pkill sage-server
go run tests/session/handshake/server/main.go &

# Wait for healthy
until curl -s http://localhost:8080/debug/health | grep -q "healthy"; do
  sleep 1
done
```

### 2. Monitor During Tests

**Terminal 1: Server Logs**
```bash
go run tests/session/handshake/server/main.go
```

**Terminal 2: Database Monitoring**
```bash
watch -n 5 'psql -h localhost -U sage -d sage -c "
SELECT COUNT(*) as sessions FROM sessions;
SELECT COUNT(*) as nonces FROM nonces;
"'
```

**Terminal 3: System Resources**
```bash
htop
```

**Terminal 4: Run Test**
```bash
./scripts/run-loadtest.sh baseline
```

### 3. Post-Test Validation

```bash
# Check for errors in logs
grep -i error logs/sage-server.log

# Verify cleanup
psql -h localhost -U sage -d sage -c "
SELECT COUNT(*) FROM sessions WHERE expires_at < NOW();
"

# Check memory usage
ps aux | grep sage-server
```

---

## Troubleshooting

### k6 Connection Errors

**Problem:** `WARN[0001] Request Failed error="dial: i/o timeout"`

**Solutions:**
1. Increase timeout in `tools/loadtest/config.js`
2. Reduce concurrent VUs
3. Check server is running: `curl http://localhost:8080/debug/health`
4. Check firewall rules

### High Error Rates

**Problem:** Error rate > threshold

**Solutions:**
1. Check server logs for errors
2. Verify database is running
3. Check connection pool size
4. Reduce load or increase capacity

### Memory Issues

**Problem:** k6 process using excessive memory

**Solutions:**
1. Reduce `--out` frequency
2. Use `--discard-response-bodies`
3. Run tests separately instead of "all"

---

## References

- [k6 Documentation](https://k6.io/docs/)
- [k6 Best Practices](https://k6.io/docs/misc/fine-tuning-os/)
- [SAGE API Documentation](../docs/API.md)
- [SAGE Database Schema](../docs/DATABASE.md)

---

## Support

**Issues:** https://github.com/sage-x-project/sage/issues
**Label:** `performance` or `load-testing`
