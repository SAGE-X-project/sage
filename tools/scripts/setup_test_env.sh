#!/bin/bash
# SAGE Test Environment Setup Script
# Automatically starts local blockchain nodes and required services for integration testing

set -e

# Color output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Configuration
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "${SCRIPT_DIR}/../.." && pwd)"
DOCKER_COMPOSE_FILE="${PROJECT_ROOT}/deployments/docker/test-environment.yml"
TIMEOUT=60  # seconds to wait for services

# Logging functions
log_info() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

log_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

log_warn() {
    echo -e "${YELLOW}[WARN]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# Check dependencies
check_dependencies() {
    log_info "Checking dependencies..."

    if ! command -v docker &> /dev/null; then
        log_error "Docker is not installed"
        exit 1
    fi

    if ! command -v docker-compose &> /dev/null && ! docker compose version &> /dev/null; then
        log_error "Docker Compose is not installed"
        exit 1
    fi

    if ! docker info &> /dev/null; then
        log_error "Docker daemon is not running"
        exit 1
    fi

    log_success "All dependencies are installed"
}

# Cleanup existing test environment
cleanup_existing() {
    log_info "Cleaning up existing test environment..."

    if [ -f "${DOCKER_COMPOSE_FILE}" ]; then
        docker-compose -f "${DOCKER_COMPOSE_FILE}" down -v --remove-orphans 2>/dev/null || true
    fi

    # Remove any dangling containers
    docker ps -a --filter "name=sage-test" --format "{{.Names}}" | xargs -r docker rm -f 2>/dev/null || true

    log_success "Cleanup completed"
}

# Start test environment
start_services() {
    log_info "Starting test environment services..."

    if [ ! -f "${DOCKER_COMPOSE_FILE}" ]; then
        log_error "Docker Compose file not found: ${DOCKER_COMPOSE_FILE}"
        exit 1
    fi

    # Start services
    docker-compose -f "${DOCKER_COMPOSE_FILE}" up -d

    log_success "Services started"
}

# Wait for service to be healthy
wait_for_service() {
    local service_name=$1
    local max_wait=$2
    local elapsed=0

    log_info "Waiting for ${service_name} to be healthy..."

    while [ $elapsed -lt $max_wait ]; do
        if docker-compose -f "${DOCKER_COMPOSE_FILE}" ps | grep "${service_name}" | grep -q "Up (healthy)"; then
            log_success "${service_name} is healthy"
            return 0
        fi

        sleep 2
        elapsed=$((elapsed + 2))
        echo -n "."
    done

    echo ""
    log_error "${service_name} failed to become healthy within ${max_wait}s"
    return 1
}

# Wait for all services
wait_for_services() {
    log_info "Waiting for all services to be ready (timeout: ${TIMEOUT}s)..."

    wait_for_service "ethereum-node" $TIMEOUT || return 1
    wait_for_service "solana-node" $TIMEOUT || return 1
    wait_for_service "redis-test" $TIMEOUT || return 1

    log_success "All services are healthy and ready"
}

# Test connectivity
test_connectivity() {
    log_info "Testing service connectivity..."

    # Test Ethereum RPC
    if curl -s -X POST -H "Content-Type: application/json" \
        --data '{"jsonrpc":"2.0","method":"eth_blockNumber","params":[],"id":1}' \
        http://localhost:8545 | grep -q "result"; then
        log_success "Ethereum RPC: http://localhost:8545 ✓"
    else
        log_warn "Ethereum RPC: http://localhost:8545 ✗"
    fi

    # Test Solana RPC
    if curl -s -X POST -H "Content-Type: application/json" \
        --data '{"jsonrpc":"2.0","id":1,"method":"getHealth"}' \
        http://localhost:8899 | grep -q "result"; then
        log_success "Solana RPC: http://localhost:8899 ✓"
    else
        log_warn "Solana RPC: http://localhost:8899 ✗"
    fi

    # Test Redis
    if docker exec sage-test-redis redis-cli ping | grep -q "PONG"; then
        log_success "Redis: localhost:6380 ✓"
    else
        log_warn "Redis: localhost:6380 ✗"
    fi
}

# Print service URLs
print_service_urls() {
    echo ""
    echo -e "${GREEN}╔════════════════════════════════════════════════════╗${NC}"
    echo -e "${GREEN}║     SAGE Test Environment - Services Ready        ║${NC}"
    echo -e "${GREEN}╠════════════════════════════════════════════════════╣${NC}"
    echo -e "${GREEN}║${NC} Ethereum RPC:    http://localhost:8545           ${GREEN}║${NC}"
    echo -e "${GREEN}║${NC} Solana RPC:      http://localhost:8899           ${GREEN}║${NC}"
    echo -e "${GREEN}║${NC} Redis:           localhost:6380                  ${GREEN}║${NC}"
    echo -e "${GREEN}║${NC}                                                    ${GREEN}║${NC}"
    echo -e "${GREEN}║${NC} View logs:       docker-compose -f \\             ${GREEN}║${NC}"
    echo -e "${GREEN}║${NC}                  ${DOCKER_COMPOSE_FILE} logs -f   ${GREEN}║${NC}"
    echo -e "${GREEN}║${NC}                                                    ${GREEN}║${NC}"
    echo -e "${GREEN}║${NC} Stop services:   ./tools/scripts/                 ${GREEN}║${NC}"
    echo -e "${GREEN}║${NC}                  cleanup_test_env.sh              ${GREEN}║${NC}"
    echo -e "${GREEN}╚════════════════════════════════════════════════════╝${NC}"
    echo ""
}

# Export environment variables
export_env_vars() {
    log_info "Exporting environment variables..."

    cat > "${PROJECT_ROOT}/.env.test" <<EOF
# SAGE Test Environment Variables
# Generated by setup_test_env.sh on $(date)

# Blockchain RPC URLs
ETHEREUM_RPC_URL=http://localhost:8545
SOLANA_RPC_URL=http://localhost:8899

# Redis
REDIS_HOST=localhost
REDIS_PORT=6380

# Database (if using --with-db profile)
POSTGRES_HOST=localhost
POSTGRES_PORT=5433
POSTGRES_DB=sage_test
POSTGRES_USER=sage_test
POSTGRES_PASSWORD=sage_test_password

# Network
SAGE_NETWORK=test
SAGE_CHAIN_ID=1337

# Session (shorter timeouts for tests)
SESSION_MAX_AGE=5m
SESSION_IDLE_TIMEOUT=1m
SESSION_CLEANUP_INTERVAL=10s

# Security (relaxed for tests)
NONCE_TTL=2m
MAX_CLOCK_SKEW=2m

# Logging
LOG_LEVEL=debug
LOG_FORMAT=text
EOF

    log_success "Environment variables exported to .env.test"
}

# Main execution
main() {
    echo -e "${BLUE}╔════════════════════════════════════════════════════╗${NC}"
    echo -e "${BLUE}║         SAGE Test Environment Setup               ║${NC}"
    echo -e "${BLUE}╚════════════════════════════════════════════════════╝${NC}"
    echo ""

    # Parse arguments
    WITH_DB=false
    SKIP_CLEANUP=false

    while [[ $# -gt 0 ]]; do
        case $1 in
            --with-db)
                WITH_DB=true
                shift
                ;;
            --skip-cleanup)
                SKIP_CLEANUP=true
                shift
                ;;
            --help)
                echo "Usage: $0 [OPTIONS]"
                echo ""
                echo "Options:"
                echo "  --with-db          Include PostgreSQL database"
                echo "  --skip-cleanup     Skip cleanup of existing environment"
                echo "  --help             Show this help message"
                exit 0
                ;;
            *)
                log_error "Unknown option: $1"
                echo "Use --help for usage information"
                exit 1
                ;;
        esac
    done

    # Execute steps
    check_dependencies

    if [ "$SKIP_CLEANUP" != "true" ]; then
        cleanup_existing
    fi

    start_services
    wait_for_services || {
        log_error "Failed to start services. Check logs with:"
        echo "  docker-compose -f ${DOCKER_COMPOSE_FILE} logs"
        exit 1
    }

    test_connectivity
    export_env_vars
    print_service_urls

    log_success "Test environment is ready for integration tests!"
}

# Run main
main "$@"
