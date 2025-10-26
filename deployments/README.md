# SAGE Deployment Configurations

This directory contains deployment configurations, infrastructure setup, and environment management for SAGE.

## Directory Structure

```
deployments/
├── config/              # Environment-specific configuration
│   ├── config.go        # Configuration data structures
│   ├── loader.go        # Configuration file loader
│   ├── env.go           # Environment variable parser
│   ├── validator.go     # Configuration validation
│   ├── blockchain.go    # Blockchain-specific config
│   ├── local.yaml       # Local development config
│   ├── development.yaml # Development environment config
│   ├── staging.yaml     # Staging environment config
│   └── production.yaml  # Production environment config
│
├── docker/              # Docker containerization
│   ├── Dockerfile       # Multi-stage production build
│   ├── docker-compose.yml # Complete service stack
│   ├── test-environment.yml # Testing environment
│   ├── grafana/         # Monitoring dashboards
│   ├── prometheus/      # Metrics collection
│   ├── scripts/         # Container scripts
│   └── README.md        # Docker deployment guide
│
└── migrations/          # Database schema migrations
    ├── 000001_initial_schema.up.sql
    ├── 000001_initial_schema.down.sql
    └── seeds/           # Test data seeds
        ├── dev.sql
        └── staging.sql
```

## Quick Start

### Local Development

```bash
# 1. Start local blockchain
cd contracts/ethereum
npx hardhat node

# 2. Build SAGE
cd ../..
make build

# 3. Run with local config
./build/bin/sage-server --config deployments/config/local.yaml
```

### Docker Deployment

```bash
# Start complete stack (backend + blockchain + redis)
cd deployments/docker
docker-compose up -d

# Start with monitoring
docker-compose --profile monitoring up -d

# View logs
docker-compose logs -f sage-backend
```

### Production Deployment

```bash
# Build production image
VERSION=v1.3.0 ./deployments/docker/scripts/docker-build.sh

# Deploy with production config
docker-compose -f docker-compose.yml up -d
```

## Configuration Management

### Environment-Specific Configs

The `config/` directory provides type-safe configuration management with support for multiple environments:

| Environment | Config File | Purpose |
|-------------|-------------|---------|
| **local** | local.yaml | Local development with Hardhat node |
| **development** | development.yaml | Development server deployment |
| **staging** | staging.yaml | Staging environment for testing |
| **production** | production.yaml | Production deployment |

### Loading Configuration

```go
import "github.com/sage-x-project/sage/deployments/config"

// Load from file
cfg, err := config.LoadFromFile("deployments/config/production.yaml")

// Load from environment
cfg, err := config.LoadFromEnv()

// Validate configuration
if err := cfg.Validate(); err != nil {
    log.Fatal(err)
}
```

### Configuration Structure

```yaml
# Example: local.yaml
version: "1.0"
environment: local

server:
  host: localhost
  port: 8080
  metrics_port: 9090

blockchain:
  network: local
  rpc_url: http://localhost:8545
  chain_id: 31337
  registry_address: "0x5FbDB2315678afecb367f032d93F642f64180aa3"

session:
  max_age: 1h
  idle_timeout: 10m
  cleanup_interval: 30s

security:
  nonce_ttl: 5m
  max_clock_skew: 5m

logging:
  level: debug
  format: json
```

## Docker Deployment

### Services Overview

The Docker setup includes:

- **sage-backend**: Main SAGE application (Go)
- **blockchain**: Local Ethereum node (Geth dev mode)
- **redis**: Session cache and nonce storage
- **prometheus**: Metrics collection (optional)
- **grafana**: Monitoring dashboards (optional)

See [docker/README.md](./docker/README.md) for detailed Docker documentation.

### Common Docker Commands

```bash
cd deployments/docker

# Development
docker-compose up                    # Start all services
docker-compose up --build            # Rebuild and start
docker-compose exec sage-backend sh  # Access container shell

# Production
VERSION=v1.3.0 docker-compose up -d  # Start with specific version
docker-compose logs -f sage-backend  # Follow logs
docker-compose ps                    # Check service status

# Monitoring
docker-compose --profile monitoring up -d  # Start with Grafana/Prometheus
open http://localhost:3000                 # Access Grafana (admin/admin)

# Cleanup
docker-compose down                  # Stop services
docker-compose down -v               # Stop and remove volumes
```

## Database Migrations

### Migration Files

Database schema migrations are managed using `.sql` files:

- `000001_initial_schema.up.sql` - Create initial schema
- `000001_initial_schema.down.sql` - Rollback initial schema
- `seeds/dev.sql` - Development test data
- `seeds/staging.sql` - Staging test data

### Running Migrations

```bash
# Using golang-migrate
migrate -path deployments/migrations \
        -database "postgres://user:pass@localhost:5432/sage?sslmode=disable" \
        up

# Rollback
migrate -path deployments/migrations \
        -database "postgres://user:pass@localhost:5432/sage?sslmode=disable" \
        down
```

### Creating New Migrations

```bash
# Install migrate CLI
go install -tags 'postgres' github.com/golang-migrate/migrate/v4/cmd/migrate@latest

# Create new migration
migrate create -ext sql -dir deployments/migrations -seq add_user_table
```

## Environment Variables

### Required Variables

```bash
# Server
export SAGE_PORT=8080
export SAGE_METRICS_PORT=9090

# Blockchain
export ETHEREUM_RPC_URL=http://localhost:8545
export SAGE_CHAIN_ID=31337
export SAGE_REGISTRY_ADDRESS=0x5FbDB...

# Security
export NONCE_TTL=5m
export MAX_CLOCK_SKEW=5m

# Logging
export LOG_LEVEL=info
export LOG_FORMAT=json
```

### Optional Variables

```bash
# Session Management
export SESSION_MAX_AGE=1h
export SESSION_IDLE_TIMEOUT=10m

# Redis
export REDIS_URL=redis://localhost:6379

# Monitoring
export ENABLE_METRICS=true
export PROMETHEUS_PORT=9091
```

## Network-Specific Deployment

### Local Development (Hardhat)

```yaml
# config/local.yaml
blockchain:
  network: local
  rpc_url: http://localhost:8545
  chain_id: 31337
  gas_limit: 6721975
```

```bash
# Terminal 1: Start Hardhat
cd contracts/ethereum && npx hardhat node

# Terminal 2: Deploy contracts
npx hardhat run scripts/deploy-v4-local.js --network localhost

# Terminal 3: Start SAGE
./build/bin/sage-server --config deployments/config/local.yaml
```

### Testnet (Sepolia)

```yaml
# config/staging.yaml
blockchain:
  network: sepolia
  rpc_url: https://sepolia.infura.io/v3/YOUR_KEY
  chain_id: 11155111
  registry_address: "0x..."
```

```bash
docker-compose up -d sage-backend redis
```

### Production (Mainnet)

```yaml
# config/production.yaml
blockchain:
  network: mainnet
  rpc_url: https://mainnet.infura.io/v3/YOUR_KEY
  chain_id: 1
  registry_address: "0x..."
```

```bash
VERSION=v1.3.0 docker-compose -f docker-compose.yml up -d
```

## Monitoring and Observability

### Prometheus Metrics

Access metrics at: `http://localhost:9090/metrics`

Key metrics:
- `sage_crypto_operations_total` - Crypto operation counts
- `sage_session_active` - Active session count
- `sage_handshake_duration_seconds` - Handshake latency
- `sage_message_validation_errors_total` - Validation errors

### Grafana Dashboards

Access Grafana at: `http://localhost:3000` (admin/admin)

Pre-configured dashboards:
- **SAGE System Overview** - Overall system health
- **Session Management** - Session lifecycle metrics
- **Cryptographic Operations** - Crypto performance

### Health Checks

```bash
# Check application health
curl http://localhost:8080/health

# Check blockchain connectivity
curl http://localhost:8080/health/blockchain

# Check metrics endpoint
curl http://localhost:9090/metrics
```

## Security Best Practices

### Production Deployment Checklist

- [ ] Use specific version tags (not `latest`)
- [ ] Configure TLS/HTTPS for all endpoints
- [ ] Use secrets management (Docker secrets, Vault, etc.)
- [ ] Change default Grafana password
- [ ] Enable rate limiting
- [ ] Configure firewall rules
- [ ] Set resource limits in docker-compose.yml
- [ ] Enable log rotation
- [ ] Regular security updates
- [ ] Backup private keys securely

### Secrets Management

```bash
# Using Docker secrets
echo "your_secret_value" | docker secret create sage_rpc_url -

# Update docker-compose.yml
services:
  sage-backend:
    secrets:
      - sage_rpc_url
    environment:
      ETHEREUM_RPC_URL_FILE: /run/secrets/sage_rpc_url
```

### Resource Limits

```yaml
# docker-compose.yml
services:
  sage-backend:
    deploy:
      resources:
        limits:
          cpus: '2'
          memory: 2G
        reservations:
          cpus: '1'
          memory: 1G
```

## Troubleshooting

### Configuration Issues

```bash
# Validate configuration file
go run ./cmd/config-validator deployments/config/production.yaml

# Check environment variables
env | grep SAGE
env | grep ETHEREUM
```

### Docker Issues

```bash
# Check service status
docker-compose ps

# View logs
docker-compose logs sage-backend
docker-compose logs blockchain

# Check resource usage
docker stats

# Test blockchain connectivity
docker-compose exec sage-backend wget -qO- http://blockchain:8545
```

### Database Issues

```bash
# Check migration status
migrate -path deployments/migrations \
        -database "postgres://..." \
        version

# Force migration version
migrate -path deployments/migrations \
        -database "postgres://..." \
        force 1
```

## CI/CD Integration

### GitHub Actions Example

```yaml
name: Deploy

on:
  push:
    tags:
      - 'v*'

jobs:
  deploy:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3

      - name: Build Docker image
        run: VERSION=${GITHUB_REF#refs/tags/} ./deployments/docker/scripts/docker-build.sh

      - name: Push to registry
        run: docker push registry.example.com/sage:${GITHUB_REF#refs/tags/}
```

## Additional Resources

### Documentation
- [Docker Deployment Guide](./docker/README.md)
- [Configuration Reference](./config/README.md) (coming soon)
- [Migration Guide](./migrations/README.md) (coming soon)
- [Main Documentation](../docs/)

### Related Files
- [Build Guide](../docs/BUILD.md)
- [V4 Update Deployment Guide](../docs/V4_UPDATE_DEPLOYMENT_GUIDE.md)
- [Contributing Guidelines](../CONTRIBUTING.md)

### External Resources
- [Docker Documentation](https://docs.docker.com/)
- [Docker Compose Reference](https://docs.docker.com/compose/)
- [golang-migrate](https://github.com/golang-migrate/migrate)
- [Prometheus](https://prometheus.io/docs/)
- [Grafana](https://grafana.com/docs/)

## Support

For deployment issues:
- GitHub Issues: https://github.com/sage-x-project/sage/issues
- Discussions: https://github.com/sage-x-project/sage/discussions
- Documentation: https://github.com/sage-x-project/sage/tree/main/docs

---

**Last Updated:** 2025-10-26
**SAGE Version:** 1.3.0
