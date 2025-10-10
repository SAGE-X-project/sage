# SAGE Database Documentation

**Last Updated:** 2025-10-10
**Database:** PostgreSQL 14+
**Migration Tool:** golang-migrate/migrate

---

## Table of Contents

1. [Overview](#overview)
2. [Schema](#schema)
3. [Setup](#setup)
4. [Migrations](#migrations)
5. [Storage Interface](#storage-interface)
6. [Backup & Restore](#backup--restore)
7. [Development](#development)
8. [Production](#production)

---

## Overview

SAGE uses PostgreSQL for persistent storage of:
- **Sessions**: Active secure sessions between clients and servers
- **Nonces**: Replay attack prevention through nonce tracking
- **DIDs**: Cached DID information from blockchain

### Why PostgreSQL?

-  **ACID compliance** for session consistency
-  **JSONB support** for flexible metadata
-  **Advanced indexing** for performance
-  **Mature ecosystem** with excellent tooling
-  **Horizontal scaling** via connection pooling

---

## Schema

### Tables

#### `sessions`

Stores active secure sessions.

| Column | Type | Description |
|--------|------|-------------|
| `id` | UUID | Primary key, session identifier |
| `client_did` | TEXT | Client DID |
| `server_did` | TEXT | Server DID |
| `session_key` | BYTEA | Encrypted session key material |
| `created_at` | TIMESTAMP | Session creation time |
| `expires_at` | TIMESTAMP | Session expiration time |
| `last_activity` | TIMESTAMP | Last activity timestamp |
| `metadata` | JSONB | Additional session metadata |

**Indexes:**
- `idx_sessions_client_did` on `client_did`
- `idx_sessions_server_did` on `server_did`
- `idx_sessions_expires_at` on `expires_at`
- `idx_sessions_created_at` on `created_at`

**Constraints:**
- `expires_at > created_at`

#### `nonces`

Tracks used nonces for replay attack prevention.

| Column | Type | Description |
|--------|------|-------------|
| `nonce` | TEXT | Primary key, nonce value |
| `session_id` | UUID | Foreign key to sessions |
| `used_at` | TIMESTAMP | When nonce was used |
| `expires_at` | TIMESTAMP | Nonce expiration time |

**Indexes:**
- `idx_nonces_session_id` on `session_id`
- `idx_nonces_expires_at` on `expires_at`
- `idx_nonces_used_at` on `used_at`

**Constraints:**
- `expires_at > used_at`
- Foreign key to `sessions(id)` with `ON DELETE CASCADE`

#### `dids`

Caches DID information from blockchain.

| Column | Type | Description |
|--------|------|-------------|
| `did` | TEXT | Primary key, DID identifier |
| `public_key` | BYTEA | Public key bytes |
| `owner_address` | TEXT | Blockchain owner address |
| `key_type` | TEXT | Key type (Ed25519, Secp256k1, etc.) |
| `revoked` | BOOLEAN | Revocation status |
| `created_at` | TIMESTAMP | Creation timestamp |
| `updated_at` | TIMESTAMP | Last update timestamp |

**Indexes:**
- `idx_dids_owner_address` on `owner_address`
- `idx_dids_revoked` on `revoked` (WHERE revoked = TRUE)
- `idx_dids_key_type` on `key_type`

**Triggers:**
- `update_dids_updated_at` - Automatically updates `updated_at`

---

## Setup

### Prerequisites

```bash
# Install PostgreSQL
# Ubuntu/Debian
sudo apt-get install postgresql-14

# macOS
brew install postgresql@14

# Start PostgreSQL
sudo systemctl start postgresql  # Linux
brew services start postgresql@14  # macOS
```

### Create Database

```sql
-- Connect as postgres user
sudo -u postgres psql

-- Create database and user
CREATE DATABASE sage;
CREATE USER sage WITH ENCRYPTED PASSWORD 'your-secure-password';
GRANT ALL PRIVILEGES ON DATABASE sage TO sage;

-- Connect to sage database
\c sage

-- Grant schema permissions
GRANT ALL ON SCHEMA public TO sage;
```

### Environment Variables

Create `.env` file:

```bash
# Database Configuration
DB_HOST=localhost
DB_PORT=5432
DB_USER=sage
DB_PASSWORD=your-secure-password
DB_NAME=sage
DB_SSLMODE=disable  # use 'require' in production
```

---

## Migrations

### Install Migration Tool

```bash
# Install golang-migrate CLI
go install -tags 'postgres' github.com/golang-migrate/migrate/v4/cmd/migrate@latest
```

### Run Migrations

```bash
# Apply all migrations
./scripts/migrate-up.sh

# Or manually
export DB_HOST=localhost
export DB_PORT=5432
export DB_USER=sage
export DB_PASSWORD=sage
export DB_NAME=sage

migrate -path ./migrations \
  -database "postgres://sage:sage@localhost:5432/sage?sslmode=disable" \
  up
```

### Rollback Migrations

```bash
# Rollback 1 migration
./scripts/migrate-down.sh 1

# Rollback all
./scripts/migrate-down.sh
```

### Check Migration Status

```bash
migrate -path ./migrations \
  -database "postgres://sage:sage@localhost:5432/sage?sslmode=disable" \
  version
```

### Create New Migration

```bash
migrate create -ext sql -dir ./migrations -seq add_session_ttl_column
```

---

## Storage Interface

### Usage in Go

```go
package main

import (
    "context"
    "time"

    "github.com/sage-x-project/sage/pkg/storage"
    "github.com/sage-x-project/sage/pkg/storage/postgres"
)

func main() {
    ctx := context.Background()

    // Create PostgreSQL store
    store, err := postgres.NewStore(ctx, &postgres.Config{
        Host:     "localhost",
        Port:     5432,
        User:     "sage",
        Password: "sage",
        Database: "sage",
        SSLMode:  "disable",
    })
    if err != nil {
        panic(err)
    }
    defer store.Close()

    // Use session store
    session := &storage.Session{
        ID:        "550e8400-e29b-41d4-a716-446655440000",
        ClientDID: "did:sage:alice",
        ServerDID: "did:sage:bob",
        SessionKey: []byte("encrypted-key"),
        CreatedAt: time.Now(),
        ExpiresAt: time.Now().Add(1 * time.Hour),
        LastActivity: time.Now(),
        Metadata: map[string]interface{}{
            "purpose": "secure-messaging",
        },
    }

    err = store.SessionStore().Create(ctx, session)
    if err != nil {
        panic(err)
    }

    // Retrieve session
    retrieved, err := store.SessionStore().Get(ctx, session.ID)
    if err != nil {
        panic(err)
    }
}
```

### In-Memory Store (for Testing)

```go
import "github.com/sage-x-project/sage/pkg/storage/memory"

func TestSomething(t *testing.T) {
    store := memory.NewStore()
    defer store.Close()

    // Use same interface as PostgreSQL store
    err := store.SessionStore().Create(ctx, session)
    // ...
}
```

---

## Backup & Restore

### Create Backup

```bash
# Using script (recommended)
./scripts/backup-db.sh

# Manual backup
pg_dump -h localhost -U sage -d sage > backup.sql

# Compressed backup
pg_dump -h localhost -U sage -d sage | gzip > backup.sql.gz
```

Backups are stored in `./backups/` with timestamp:
```
sage_backup_20251010_121500.sql.gz
```

### Restore from Backup

```bash
# Using script (recommended)
./scripts/restore-db.sh ./backups/sage_backup_20251010_121500.sql.gz

# Manual restore
gunzip -c backup.sql.gz | psql -h localhost -U sage -d sage
```

** Warning:** Restore will DROP all existing data!

---

## Development

### Load Seed Data

```bash
# Load development seed data
SAGE_ENV=dev ./scripts/seed-db.sh

# Load staging seed data
SAGE_ENV=staging ./scripts/seed-db.sh
```

Development seed includes:
- 4 test DIDs (3 active, 1 revoked)
- 3 test sessions
- 3 test nonces

### Cleanup Expired Data

```go
// Automatic cleanup in application
func cleanupExpired(ctx context.Context, store storage.Store) {
    ticker := time.NewTicker(5 * time.Minute)
    defer ticker.Stop()

    for range ticker.C {
        // Clean expired sessions
        sessionsDeleted, _ := store.SessionStore().DeleteExpired(ctx)
        log.Printf("Deleted %d expired sessions", sessionsDeleted)

        // Clean expired nonces
        noncesDeleted, _ := store.NonceStore().DeleteExpired(ctx)
        log.Printf("Deleted %d expired nonces", noncesDeleted)
    }
}
```

### Database Monitoring

```sql
-- Active sessions
SELECT COUNT(*) FROM sessions WHERE expires_at > NOW();

-- Sessions by client
SELECT client_did, COUNT(*) as session_count
FROM sessions
WHERE expires_at > NOW()
GROUP BY client_did
ORDER BY session_count DESC;

-- Nonce usage
SELECT COUNT(*) FROM nonces WHERE expires_at > NOW();

-- Database size
SELECT pg_size_pretty(pg_database_size('sage'));

-- Table sizes
SELECT
    schemaname,
    tablename,
    pg_size_pretty(pg_total_relation_size(schemaname||'.'||tablename)) as size
FROM pg_tables
WHERE schemaname = 'public'
ORDER BY pg_total_relation_size(schemaname||'.'||tablename) DESC;
```

---

## Production

### Configuration

```bash
# Production environment variables
DB_HOST=your-db-host.example.com
DB_PORT=5432
DB_USER=sage
DB_PASSWORD=strong-random-password
DB_NAME=sage_prod
DB_SSLMODE=require  # Always use SSL in production
```

### Connection Pooling

The PostgreSQL store uses `pgxpool` for connection pooling:

```go
// Default pool settings (automatically configured)
// - Max connections: 25
// - Min connections: 2
// - Max idle time: 30 minutes
// - Health check period: 1 minute

// Custom pool configuration
config, _ := pgxpool.ParseConfig("postgres://...")
config.MaxConns = 50
config.MinConns = 5
pool, _ := pgxpool.NewWithConfig(ctx, config)
```

### Monitoring

Key metrics to monitor:
- Active session count
- Database connection pool usage
- Query performance (slow queries > 100ms)
- Database size growth
- Table bloat

### Security

 **Required in Production:**
1. Use SSL/TLS (`sslmode=require`)
2. Strong passwords (rotate regularly)
3. Network firewall rules
4. Regular backups (automated daily)
5. Encrypted backups
6. Audit logging enabled
7. Read-only replica for queries

 **Never in Production:**
1. Default passwords
2. Public database access
3. Unencrypted connections
4. Missing backups

---

## Troubleshooting

### Connection Issues

```bash
# Test connection
psql -h localhost -U sage -d sage -c "SELECT 1"

# Check PostgreSQL is running
sudo systemctl status postgresql  # Linux
brew services list  # macOS

# Check logs
tail -f /var/log/postgresql/postgresql-14-main.log  # Linux
tail -f /usr/local/var/log/postgres.log  # macOS
```

### Migration Failures

```bash
# Check current version
migrate -path ./migrations -database $DB_URL version

# Force to specific version (use with caution)
migrate -path ./migrations -database $DB_URL force 1

# Fix dirty state
migrate -path ./migrations -database $DB_URL force -1
migrate -path ./migrations -database $DB_URL up
```

### Performance Issues

```sql
-- Find slow queries
SELECT query, mean_exec_time, calls
FROM pg_stat_statements
ORDER BY mean_exec_time DESC
LIMIT 10;

-- Check index usage
SELECT schemaname, tablename, indexname, idx_scan
FROM pg_stat_user_indexes
WHERE idx_scan = 0
AND indexname NOT LIKE 'pg%';

-- Table bloat
SELECT schemaname, tablename,
    pg_size_pretty(pg_total_relation_size(schemaname||'.'||tablename)) as total_size
FROM pg_tables
WHERE schemaname = 'public';
```

---

## References

- [PostgreSQL Documentation](https://www.postgresql.org/docs/)
- [golang-migrate](https://github.com/golang-migrate/migrate)
- [pgx Documentation](https://github.com/jackc/pgx)
- [SAGE Storage Interface](../pkg/storage/interface.go)

---

**Need Help?**
- Check logs: `docker-compose logs postgres`
- Run tests: `go test ./pkg/storage/...`
- Report issues: GitHub Issues
