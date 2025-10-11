# SAGE Session Management Example

This example demonstrates session lifecycle management in SAGE, including creation, usage, expiration, and cleanup.

## Overview

SAGE sessions provide secure, stateful communication channels between agents. Sessions are established through HPKE handshakes and maintain cryptographic context for efficient message exchange.

**Session Features:**
- Automatic expiration (configurable max age)
- Idle timeout (configurable inactivity period)
- Activity tracking (last_activity timestamp)
- Metadata storage (flexible JSONB field)
- Nonce-based replay attack prevention
- PostgreSQL persistence with connection pooling

---

## Session Lifecycle

```
┌─────────────┐
│  Initial    │
│  Handshake  │
└──────┬──────┘
       │
       v
┌─────────────┐     Activity      ┌─────────────┐
│   Active    │ ───────────────> │   Active    │
│  Session    │                   │ (Updated)   │
└──────┬──────┘                   └──────┬──────┘
       │                                 │
       │ Max Age / Idle Timeout          │
       v                                 v
┌─────────────┐                   ┌─────────────┐
│   Expired   │                   │   Cleanup   │
│  Session    │ ───────────────> │  (Deleted)  │
└─────────────┘                   └─────────────┘
```

---

## Configuration

### Environment Variables

```bash
# Session timing
SESSION_MAX_AGE=1h              # Maximum session lifetime
SESSION_IDLE_TIMEOUT=10m        # Inactivity timeout
SESSION_CLEANUP_INTERVAL=30s    # Cleanup job interval

# Security
NONCE_TTL=5m                    # Nonce expiration time
MAX_CLOCK_SKEW=5m               # Maximum allowed clock difference
```

### Configuration File (config.yaml)

```yaml
session:
  max_age: "1h"
  idle_timeout: "10m"
  cleanup_interval: "30s"

security:
  nonce_ttl: "5m"
  max_clock_skew: "5m"

storage:
  type: "postgres"  # or "memory" for testing
  postgres:
    host: "localhost"
    port: 5432
    user: "sage"
    password: "sage"
    database: "sage"
    ssl_mode: "require"
```

---

## Session Creation

### 1. Establish Session via Handshake

**Request:**
```bash
curl -X POST http://localhost:8080/v1/a2a:sendMessage \
  -H "Content-Type: application/json" \
  -d '{
    "sender_did": "did:sage:ethereum:0xAlice",
    "receiver_did": "did:sage:ethereum:0xServer",
    "message": "AgECA...encrypted_handshake...",
    "timestamp": 1234567890,
    "signature": "handshake_signature=="
  }'
```

**Response:**
```json
{
  "session_id": "550e8400-e29b-41d4-a716-446655440000",
  "response": "encrypted_session_confirmation=="
}
```

### 2. Session Metadata

Sessions are stored with metadata:

```sql
SELECT * FROM sessions WHERE id = '550e8400-e29b-41d4-a716-446655440000';
```

**Result:**
| Field | Value | Description |
|-------|-------|-------------|
| id | 550e8400-... | Session UUID |
| client_did | did:sage:ethereum:0xAlice | Client identifier |
| server_did | did:sage:ethereum:0xServer | Server identifier |
| session_key | \x01020304... | Encrypted session key material |
| created_at | 2025-10-10 12:00:00 | Creation timestamp |
| expires_at | 2025-10-10 13:00:00 | Expiration timestamp (created_at + max_age) |
| last_activity | 2025-10-10 12:00:00 | Last activity timestamp |
| metadata | {"purpose": "secure-messaging"} | Custom metadata (JSONB) |

---

## Session Usage

### Send Message in Existing Session

**Request:**
```bash
curl -X POST http://localhost:8080/v1/a2a:sendMessage \
  -H "Content-Type: application/json" \
  -H "X-Session-ID: 550e8400-e29b-41d4-a716-446655440000" \
  -d '{
    "sender_did": "did:sage:ethereum:0xAlice",
    "receiver_did": "did:sage:ethereum:0xServer",
    "message": "AgECA...encrypted_message...",
    "timestamp": 1234567900,
    "signature": "message_signature=="
  }'
```

**Response:**
```json
{
  "response": "encrypted_response==",
  "session_id": "550e8400-e29b-41d4-a716-446655440000"
}
```

**Activity Update:**
- `last_activity` timestamp is automatically updated
- Idle timeout timer is reset
- Session expiration time remains unchanged

---

## Session Expiration

### Max Age Expiration

Sessions expire after `max_age` regardless of activity:

```
created_at: 2025-10-10 12:00:00
max_age:    1 hour
expires_at: 2025-10-10 13:00:00  (created_at + max_age)
```

After 13:00:00, requests with this session will fail:

**Error Response:**
```json
{
  "error": "session not found or expired",
  "code": "SESSION_EXPIRED",
  "details": {
    "session_id": "550e8400-e29b-41d4-a716-446655440000",
    "expired_at": "2025-10-10T13:00:00Z"
  }
}
```

### Idle Timeout Expiration

Sessions expire after `idle_timeout` of inactivity:

```
last_activity:   2025-10-10 12:30:00
idle_timeout:    10 minutes
will_expire_at:  2025-10-10 12:40:00  (last_activity + idle_timeout)
```

**Check Session Activity:**
```sql
SELECT
    id,
    client_did,
    last_activity,
    last_activity + interval '10 minutes' as idle_expires_at,
    expires_at as max_age_expires_at,
    CASE
        WHEN last_activity + interval '10 minutes' < NOW() THEN 'idle_expired'
        WHEN expires_at < NOW() THEN 'max_age_expired'
        ELSE 'active'
    END as status
FROM sessions
WHERE id = '550e8400-e29b-41d4-a716-446655440000';
```

---

## Session Cleanup

### Automatic Cleanup

SAGE runs a periodic cleanup job:

```go
// Server-side cleanup (automatic)
ticker := time.NewTicker(30 * time.Second)
defer ticker.Stop()

for range ticker.C {
    // Clean expired sessions
    deleted, _ := sessionStore.DeleteExpired(ctx)
    log.Printf("Deleted %d expired sessions", deleted)

    // Clean expired nonces
    noncesDeleted, _ := nonceStore.DeleteExpired(ctx)
    log.Printf("Deleted %d expired nonces", noncesDeleted)
}
```

**Cleanup SQL:**
```sql
-- Delete sessions past max age
DELETE FROM sessions WHERE expires_at < NOW();

-- Delete sessions past idle timeout
DELETE FROM sessions
WHERE last_activity + interval '10 minutes' < NOW();
```

### Manual Cleanup

**Development/Testing:**
```bash
# Via psql
psql -h localhost -U sage -d sage -c "DELETE FROM sessions WHERE expires_at < NOW();"

# Via Go code
ctx := context.Background()
deleted, err := sessionStore.DeleteExpired(ctx)
fmt.Printf("Deleted %d sessions\n", deleted)
```

---

## Monitoring Sessions

### Health Check Endpoint

**Request:**
```bash
curl http://localhost:8080/debug/health
```

**Response:**
```json
{
  "status": "healthy",
  "timestamp": "2025-10-10T12:00:00Z",
  "sessions": {
    "active": 42,
    "total": 150
  }
}
```

### Database Queries

**Active sessions by client:**
```sql
SELECT
    client_did,
    COUNT(*) as session_count,
    MAX(last_activity) as last_seen
FROM sessions
WHERE expires_at > NOW()
    AND last_activity + interval '10 minutes' > NOW()
GROUP BY client_did
ORDER BY session_count DESC;
```

**Session statistics:**
```sql
SELECT
    COUNT(*) FILTER (WHERE expires_at > NOW()) as active_by_max_age,
    COUNT(*) FILTER (WHERE last_activity + interval '10 minutes' > NOW()) as active_by_idle,
    COUNT(*) FILTER (WHERE expires_at < NOW()) as expired_by_max_age,
    COUNT(*) FILTER (WHERE last_activity + interval '10 minutes' < NOW()) as expired_by_idle,
    COUNT(*) as total
FROM sessions;
```

**Average session duration:**
```sql
SELECT
    AVG(EXTRACT(EPOCH FROM (expires_at - created_at))) as avg_max_age_seconds,
    AVG(EXTRACT(EPOCH FROM (last_activity - created_at))) as avg_active_duration_seconds,
    PERCENTILE_CONT(0.5) WITHIN GROUP (ORDER BY EXTRACT(EPOCH FROM (last_activity - created_at))) as median_active_duration_seconds
FROM sessions
WHERE expires_at < NOW();
```

---

## Session Metrics

SAGE exposes Prometheus metrics for session monitoring:

**Available Metrics:**
```
# Active sessions gauge
sage_sessions_active{client_did="did:sage:ethereum:0xAlice"} 3

# Session creation counter
sage_sessions_created_total 150

# Session expiration counter
sage_sessions_expired_total{reason="max_age"} 45
sage_sessions_expired_total{reason="idle_timeout"} 23

# Session duration histogram
sage_session_duration_seconds_bucket{le="300"} 10
sage_session_duration_seconds_bucket{le="600"} 25
sage_session_duration_seconds_bucket{le="3600"} 100

# Activity latency histogram
sage_session_activity_update_seconds 0.002
```

**Query Grafana:**
```promql
# Session creation rate (per minute)
rate(sage_sessions_created_total[5m]) * 60

# Session expiration by reason
sum by (reason) (rate(sage_sessions_expired_total[5m]))

# Average active sessions
avg(sage_sessions_active)

# 95th percentile session duration
histogram_quantile(0.95, rate(sage_session_duration_seconds_bucket[5m]))
```

---

## Storage Backends

### PostgreSQL (Production)

**Connection:**
```go
import "github.com/sage-x-project/sage/pkg/storage/postgres"

store, err := postgres.NewStore(ctx, &postgres.Config{
    Host:     "localhost",
    Port:     5432,
    User:     "sage",
    Password: "secure-password",
    Database: "sage",
    SSLMode:  "require",
})
defer store.Close()

sessionStore := store.SessionStore()
```

**Features:**
- ACID transactions
- Connection pooling (pgxpool)
- JSONB for flexible metadata
- Automatic index optimization
- Foreign key constraints (nonces → sessions)

### In-Memory (Testing)

**Connection:**
```go
import "github.com/sage-x-project/sage/pkg/storage/memory"

store := memory.NewStore()
defer store.Close()

sessionStore := store.SessionStore()
```

**Features:**
- Thread-safe (sync.RWMutex)
- Same interface as PostgreSQL
- No persistence (data lost on restart)
- Fast for unit tests

---

## Best Practices

### 1. Session Tuning

Choose `max_age` based on security requirements:
- **High security**: 15-30 minutes
- **Balanced**: 1 hour (default)
- **Long-lived**: 24 hours (only for trusted environments)

Choose `idle_timeout` based on usage patterns:
- **Interactive**: 5-10 minutes
- **Automated agents**: 30 minutes - 1 hour
- **Batch processing**: Disable (set to max_age)

### 2. Nonce Management

- Nonce TTL should be ≥ 2 × MAX_CLOCK_SKEW
- Cleanup nonces frequently to prevent database bloat
- Monitor nonce table size

### 3. Database Maintenance

```bash
# Regular vacuum (PostgreSQL)
psql -h localhost -U sage -d sage -c "VACUUM ANALYZE sessions;"

# Check table bloat
psql -h localhost -U sage -d sage -c "SELECT pg_size_pretty(pg_total_relation_size('sessions'));"

# Reindex if needed
psql -h localhost -U sage -d sage -c "REINDEX TABLE sessions;"
```

### 4. Connection Pooling

```go
// Tune pool size based on load
config.MaxConns = 50        // Maximum connections
config.MinConns = 5         // Minimum idle connections
config.MaxConnIdleTime = 30 * time.Minute
config.HealthCheckPeriod = 1 * time.Minute
```

---

## Troubleshooting

### Sessions Not Expiring

**Check cleanup job:**
```bash
# Look for cleanup logs
docker logs sage-backend 2>&1 | grep "Deleted.*expired sessions"
```

**Manually trigger cleanup:**
```sql
SELECT COUNT(*) FROM sessions WHERE expires_at < NOW();
DELETE FROM sessions WHERE expires_at < NOW();
```

### High Database Load

**Check slow queries:**
```sql
SELECT query, mean_exec_time, calls
FROM pg_stat_statements
WHERE query LIKE '%sessions%'
ORDER BY mean_exec_time DESC
LIMIT 10;
```

**Verify indexes:**
```sql
SELECT schemaname, tablename, indexname, idx_scan
FROM pg_stat_user_indexes
WHERE tablename = 'sessions';
```

### Session Conflicts

**Check for duplicate sessions:**
```sql
SELECT client_did, server_did, COUNT(*) as session_count
FROM sessions
WHERE expires_at > NOW()
GROUP BY client_did, server_did
HAVING COUNT(*) > 1;
```

---

## References

- [SAGE Database Documentation](../../docs/DATABASE.md)
- [Storage Interface](../../pkg/storage/interface.go)
- [PostgreSQL Storage](../../pkg/storage/postgres/)
- [Session Tests](../../tests/session/)
