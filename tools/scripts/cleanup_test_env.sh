#!/bin/bash
# SAGE Test Environment Cleanup Script
# Stops and removes all test environment containers and volumes

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

# Stop services
stop_services() {
    log_info "Stopping test environment services..."

    if [ -f "${DOCKER_COMPOSE_FILE}" ]; then
        docker-compose -f "${DOCKER_COMPOSE_FILE}" stop 2>/dev/null || true
        log_success "Services stopped"
    else
        log_warn "Docker Compose file not found: ${DOCKER_COMPOSE_FILE}"
    fi
}

# Remove containers
remove_containers() {
    log_info "Removing test containers..."

    if [ -f "${DOCKER_COMPOSE_FILE}" ]; then
        docker-compose -f "${DOCKER_COMPOSE_FILE}" rm -f 2>/dev/null || true
    fi

    # Force remove any remaining sage-test containers
    docker ps -a --filter "name=sage-test" --format "{{.Names}}" | while read container; do
        if [ -n "$container" ]; then
            log_info "Removing container: $container"
            docker rm -f "$container" 2>/dev/null || true
        fi
    done

    log_success "Containers removed"
}

# Remove volumes
remove_volumes() {
    local REMOVE_VOLUMES=$1

    if [ "$REMOVE_VOLUMES" = "true" ]; then
        log_info "Removing test volumes..."

        if [ -f "${DOCKER_COMPOSE_FILE}" ]; then
            docker-compose -f "${DOCKER_COMPOSE_FILE}" down -v 2>/dev/null || true
        fi

        # Remove specific test volumes
        for volume in sage-test-solana-ledger sage-test-postgres-data; do
            if docker volume ls --format "{{.Name}}" | grep -q "^${volume}$"; then
                log_info "Removing volume: $volume"
                docker volume rm "$volume" 2>/dev/null || true
            fi
        done

        log_success "Volumes removed"
    else
        log_info "Skipping volume removal (use --remove-volumes to delete)"
    fi
}

# Remove network
remove_network() {
    log_info "Removing test network..."

    if docker network ls --format "{{.Name}}" | grep -q "^sage-test-network$"; then
        docker network rm sage-test-network 2>/dev/null || {
            log_warn "Network sage-test-network still in use, skipping removal"
        }
    fi

    log_success "Network cleanup completed"
}

# Clean environment file
clean_env_file() {
    if [ -f "${PROJECT_ROOT}/.env.test" ]; then
        log_info "Removing test environment file..."
        rm -f "${PROJECT_ROOT}/.env.test"
        log_success "Environment file removed"
    fi
}

# Show running containers
show_running() {
    echo ""
    log_info "Checking for remaining test containers..."

    local running=$(docker ps --filter "name=sage-test" --format "table {{.Names}}\t{{.Status}}" | tail -n +2)

    if [ -z "$running" ]; then
        log_success "No test containers running"
    else
        log_warn "Still running:"
        echo "$running"
    fi
}

# Main execution
main() {
    echo -e "${BLUE}╔════════════════════════════════════════════════════╗${NC}"
    echo -e "${BLUE}║       SAGE Test Environment Cleanup               ║${NC}"
    echo -e "${BLUE}╚════════════════════════════════════════════════════╝${NC}"
    echo ""

    # Parse arguments
    REMOVE_VOLUMES=false
    FORCE=false

    while [[ $# -gt 0 ]]; do
        case $1 in
            --remove-volumes|-v)
                REMOVE_VOLUMES=true
                shift
                ;;
            --force|-f)
                FORCE=true
                shift
                ;;
            --help|-h)
                echo "Usage: $0 [OPTIONS]"
                echo ""
                echo "Options:"
                echo "  -v, --remove-volumes    Remove data volumes (ledger, database)"
                echo "  -f, --force             Skip confirmation prompt"
                echo "  -h, --help              Show this help message"
                echo ""
                echo "Examples:"
                echo "  $0                      # Stop and remove containers only"
                echo "  $0 -v                   # Stop, remove containers and volumes"
                echo "  $0 -v -f                # Force cleanup with volumes"
                exit 0
                ;;
            *)
                log_error "Unknown option: $1"
                echo "Use --help for usage information"
                exit 1
                ;;
        esac
    done

    # Confirmation prompt
    if [ "$FORCE" != "true" ]; then
        echo -e "${YELLOW}This will stop and remove all test environment containers.${NC}"
        if [ "$REMOVE_VOLUMES" = "true" ]; then
            echo -e "${YELLOW}WARNING: This will also remove all data volumes!${NC}"
        fi
        echo ""
        read -p "Continue? (y/N) " -n 1 -r
        echo ""
        if [[ ! $REPLY =~ ^[Yy]$ ]]; then
            log_info "Cleanup cancelled"
            exit 0
        fi
    fi

    # Execute cleanup
    stop_services
    remove_containers
    remove_volumes "$REMOVE_VOLUMES"
    remove_network
    clean_env_file
    show_running

    echo ""
    log_success "Test environment cleanup completed!"

    if [ "$REMOVE_VOLUMES" != "true" ]; then
        echo ""
        log_info "Tip: Use --remove-volumes to also delete blockchain ledger and database"
    fi
}

# Run main
main "$@"
