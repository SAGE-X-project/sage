# SAGE Docker Deployment

Complete Docker containerization for SAGE with production-ready configurations.

## Quick Start

### Using Docker Compose (Recommended)

```bash
# Start all services (backend + blockchain + redis)
docker-compose up -d

# Start with monitoring stack
docker-compose --profile monitoring up -d

# View logs
docker-compose logs -f sage-backend

# Stop all services
docker-compose down

# Remove all data
docker-compose down -v
```

### Using Docker Directly

```bash
# Build image
./scripts/docker-build.sh

# Run interactively
./scripts/docker-run.sh

# Run in daemon mode
./scripts/docker-run.sh -d

# Run specific command
./scripts/docker-run.sh -c "sage-crypto --version"
```

## Architecture

```
┌─────────────────────────────────────────────────────────┐
│                    Docker Compose Stack                 │
├─────────────────────────────────────────────────────────┤
│                                                         │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐  │
│  │ sage-backend │  │  blockchain  │  │    redis     │  │
│  │   (Go App)   │──│  (Geth Dev)  │  │   (Cache)    │  │
│  │  Port: 8080  │  │  Port: 8545  │  │  Port: 6379  │  │
│  │  Port: 9090  │  │  Port: 8546  │  │              │  │
│  └──────────────┘  └──────────────┘  └──────────────┘  │
│         │                                               │
│         └───────────────────┬─────────────────────────  │
│                             │                           │
│  ┌──────────────┐  ┌──────────────┐                    │
│  │  prometheus  │  │   grafana    │  (monitoring)      │
│  │  Port: 9091  │──│  Port: 3000  │  (optional)        │
│  └──────────────┘  └──────────────┘                    │
│                                                         │
└─────────────────────────────────────────────────────────┘
```

## Services

### sage-backend
SAGE main application with cryptographic operations and session management.

**Ports:**
- `8080`: HTTP API
- `9090`: Prometheus metrics

**Volumes:**
- `sage-keys`: Private keys storage
- `sage-data`: Application data

**Environment Variables:**
See [Environment Configuration](#environment-configuration)

### blockchain
Local Ethereum development node (Geth in dev mode).

**Ports:**
- `8545`: HTTP JSON-RPC
- `8546`: WebSocket

**Features:**
- Auto-mining (1 second blocks)
- Pre-funded development accounts
- CORS enabled for local development

### redis
Session cache and nonce storage.

**Port:** `6379`

**Features:**
- AOF persistence enabled
- Health checks configured

### prometheus (monitoring profile)
Metrics collection and storage.

**Port:** `9091` (mapped from internal 9090)

**Configuration:** `docker/prometheus/prometheus.yml`

### grafana (monitoring profile)
Metrics visualization dashboard.

**Port:** `3000`

**Default Credentials:**
- Username: `admin`
- Password: `admin` (change on first login)

**Dashboards:**
- SAGE System Overview
- Session Management Metrics
- Cryptographic Operations

## Environment Configuration

Create a `.env` file in the project root:

```bash
# SAGE Configuration
VERSION=latest
SAGE_PORT=8080
SAGE_METRICS_PORT=9090
SAGE_NETWORK=local
SAGE_CHAIN_ID=1337

# Blockchain Configuration
BLOCKCHAIN_RPC_PORT=8545
BLOCKCHAIN_WS_PORT=8546
ETHEREUM_RPC_URL=http://blockchain:8545

# Network-specific RPC URLs (optional)
SEPOLIA_RPC_URL=https://sepolia.infura.io/v3/YOUR_KEY
KAIA_RPC_URL=https://kaia-rpc-url

# Smart Contract Addresses (after deployment)
SAGE_REGISTRY_ADDRESS=
ERC8004_REGISTRY_ADDRESS=

# Session Configuration
SESSION_MAX_AGE=1h
SESSION_IDLE_TIMEOUT=10m
SESSION_CLEANUP_INTERVAL=30s

# Security
NONCE_TTL=5m
MAX_CLOCK_SKEW=5m

# Logging
LOG_LEVEL=info
LOG_FORMAT=json

# Redis Configuration
REDIS_PORT=6379

# Monitoring (optional)
PROMETHEUS_PORT=9091
GRAFANA_PORT=3000
GRAFANA_USER=admin
GRAFANA_PASSWORD=admin

# Build Configuration
BUILD_DATE=
```

## Common Operations

### Development

```bash
# Start development environment
docker-compose up

# Rebuild after code changes
docker-compose up --build

# Run tests inside container
docker-compose exec sage-backend go test ./...

# Access container shell
docker-compose exec sage-backend /bin/sh

# View container logs
docker-compose logs -f sage-backend
```

### Production

```bash
# Build production image with version
VERSION=v1.0.0 ./scripts/docker-build.sh

# Tag for registry
docker tag sage-backend:v1.0.0 registry.example.com/sage-backend:v1.0.0

# Push to registry
docker push registry.example.com/sage-backend:v1.0.0

# Deploy with specific version
VERSION=v1.0.0 docker-compose up -d
```

### Monitoring

```bash
# Start with monitoring stack
docker-compose --profile monitoring up -d

# Access Grafana
open http://localhost:3000

# Access Prometheus
open http://localhost:9091

# View metrics directly
curl http://localhost:9090/metrics
```

### Debugging

```bash
# Check container health
docker-compose ps

# Inspect container
docker inspect sage-backend

# View resource usage
docker stats

# Execute health check manually
docker-compose exec sage-backend /bin/sh /usr/local/bin/healthcheck.sh

# Check logs for errors
docker-compose logs sage-backend | grep ERROR
```

## Multi-Architecture Builds

Build for multiple platforms (requires Docker Buildx):

```bash
# Build for AMD64 and ARM64
PLATFORMS=linux/amd64,linux/arm64 ./scripts/docker-build.sh

# Build and push to registry
DOCKER_REGISTRY=registry.example.com \
PLATFORMS=linux/amd64,linux/arm64 \
./scripts/docker-build.sh
```

## Volume Management

### Backup

```bash
# Backup keys
docker run --rm \
  -v sage-keys:/data \
  -v $(pwd):/backup \
  alpine tar czf /backup/sage-keys-backup.tar.gz /data

# Backup data
docker run --rm \
  -v sage-data:/data \
  -v $(pwd):/backup \
  alpine tar czf /backup/sage-data-backup.tar.gz /data
```

### Restore

```bash
# Restore keys
docker run --rm \
  -v sage-keys:/data \
  -v $(pwd):/backup \
  alpine tar xzf /backup/sage-keys-backup.tar.gz -C /

# Restore data
docker run --rm \
  -v sage-data:/data \
  -v $(pwd):/backup \
  alpine tar xzf /backup/sage-data-backup.tar.gz -C /
```

### Clean Up

```bash
# Remove all SAGE volumes
docker-compose down -v

# Remove specific volume
docker volume rm sage-keys
docker volume rm sage-data

# Prune unused volumes
docker volume prune
```

## Network Configuration

### Development (Local Blockchain)

```bash
# Use included Geth dev node
ETHEREUM_RPC_URL=http://blockchain:8545 \
SAGE_NETWORK=local \
docker-compose up
```

### Testnet (Sepolia)

```bash
# Configure for Sepolia
ETHEREUM_RPC_URL=https://sepolia.infura.io/v3/YOUR_KEY \
SAGE_NETWORK=sepolia \
SAGE_CHAIN_ID=11155111 \
SAGE_REGISTRY_ADDRESS=0x... \
docker-compose up sage-backend redis
```

### Production (Mainnet)

```bash
# Configure for mainnet
ETHEREUM_RPC_URL=https://mainnet.infura.io/v3/YOUR_KEY \
SAGE_NETWORK=mainnet \
SAGE_CHAIN_ID=1 \
SAGE_REGISTRY_ADDRESS=0x... \
docker-compose up -d sage-backend redis
```

## Security Best Practices

1. **Never commit `.env` files** - Add to `.gitignore`

2. **Use secrets management** for production:
   ```bash
   # Use Docker secrets
   docker secret create sage_rpc_url -
   # Update docker-compose.yml to use secrets
   ```

3. **Change default passwords**:
   - Grafana admin password
   - Any other default credentials

4. **Run as non-root** (already configured in Dockerfile)

5. **Limit container resources**:
   ```yaml
   services:
     sage-backend:
       deploy:
         resources:
           limits:
             cpus: '2'
             memory: 2G
   ```

6. **Use specific image tags** in production (not `latest`)

7. **Enable TLS** for production deployments

8. **Regular security updates**:
   ```bash
   # Rebuild with latest base images
   docker-compose build --pull --no-cache
   ```

## Troubleshooting

### Container won't start

```bash
# Check logs
docker-compose logs sage-backend

# Verify environment variables
docker-compose config

# Check if ports are already in use
lsof -i :8080
lsof -i :8545
```

### Blockchain connection issues

```bash
# Test RPC connectivity
docker-compose exec sage-backend wget -qO- http://blockchain:8545

# Verify blockchain is running
docker-compose ps blockchain

# Check blockchain logs
docker-compose logs blockchain
```

### Performance issues

```bash
# Check resource usage
docker stats

# Increase memory limits
# Edit docker-compose.yml deploy.resources section

# Check for memory leaks
docker-compose exec sage-backend ps aux
```

### Data persistence issues

```bash
# List volumes
docker volume ls | grep sage

# Inspect volume
docker volume inspect sage-keys

# Check volume mount inside container
docker-compose exec sage-backend ls -la ~/.sage
```

## CI/CD Integration

### GitHub Actions

```yaml
name: Docker Build

on:
  push:
    branches: [main]
    tags: ['v*']

jobs:
  build:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3

      - name: Set up Docker Buildx
        uses: docker/setup-buildx-action@v2

      - name: Build and push
        run: |
          VERSION=${GITHUB_REF#refs/tags/} ./scripts/docker-build.sh
```

### GitLab CI

```yaml
docker-build:
  stage: build
  image: docker:latest
  services:
    - docker:dind
  script:
    - ./scripts/docker-build.sh
  only:
    - main
    - tags
```

## Additional Resources

- [Main Documentation](../README.md)
- [Build Guide](../docs/BUILD.md)
- [Deployment Guide](../docs/DEPLOYMENT.md)
- [Dockerfile](../Dockerfile)
- [docker-compose.yml](../docker-compose.yml)

## Support

For issues and questions:
- GitHub Issues: https://github.com/sage-x-project/sage/issues
- Documentation: https://github.com/sage-x-project/sage/tree/main/docs
