#!/bin/bash
# SAGE Docker Run Script
# Easy script to run SAGE containers with common configurations

set -e

# Colors
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

# Configuration
IMAGE_NAME="${IMAGE_NAME:-sage-backend}"
VERSION="${VERSION:-latest}"
CONTAINER_NAME="${CONTAINER_NAME:-sage-backend-dev}"

# Default environment variables
SAGE_PORT="${SAGE_PORT:-8080}"
SAGE_METRICS_PORT="${SAGE_METRICS_PORT:-9090}"
ETHEREUM_RPC_URL="${ETHEREUM_RPC_URL:-http://localhost:8545}"
SAGE_NETWORK="${SAGE_NETWORK:-local}"
LOG_LEVEL="${LOG_LEVEL:-info}"

echo -e "${GREEN}SAGE Docker Run${NC}"
echo "================================"
echo "Image: $IMAGE_NAME:$VERSION"
echo "Container: $CONTAINER_NAME"
echo "Ports: $SAGE_PORT (HTTP), $SAGE_METRICS_PORT (Metrics)"
echo "RPC URL: $ETHEREUM_RPC_URL"
echo ""

# Check if container already exists
if docker ps -a --format '{{.Names}}' | grep -q "^${CONTAINER_NAME}$"; then
    echo -e "${YELLOW}Container $CONTAINER_NAME already exists${NC}"
    read -p "Remove and recreate? (y/N): " -n 1 -r
    echo
    if [[ $REPLY =~ ^[Yy]$ ]]; then
        echo "Removing existing container..."
        docker rm -f "$CONTAINER_NAME"
    else
        echo "Aborting"
        exit 0
    fi
fi

# Parse command line arguments
MODE="interactive"
COMMAND=""

while [[ $# -gt 0 ]]; do
    case $1 in
        -d|--daemon)
            MODE="daemon"
            shift
            ;;
        -c|--command)
            COMMAND="$2"
            shift 2
            ;;
        -h|--help)
            echo "Usage: $0 [OPTIONS]"
            echo ""
            echo "Options:"
            echo "  -d, --daemon          Run in daemon mode"
            echo "  -c, --command CMD     Execute specific command"
            echo "  -h, --help            Show this help"
            echo ""
            echo "Environment variables:"
            echo "  IMAGE_NAME            Docker image name (default: sage-backend)"
            echo "  VERSION               Image version (default: latest)"
            echo "  CONTAINER_NAME        Container name (default: sage-backend-dev)"
            echo "  SAGE_PORT             HTTP port (default: 8080)"
            echo "  SAGE_METRICS_PORT     Metrics port (default: 9090)"
            echo "  ETHEREUM_RPC_URL      Blockchain RPC URL"
            echo "  SAGE_NETWORK          Network type (default: local)"
            echo "  LOG_LEVEL             Log level (default: info)"
            exit 0
            ;;
        *)
            echo "Unknown option: $1"
            echo "Use -h or --help for usage information"
            exit 1
            ;;
    esac
done

# Docker run arguments
DOCKER_ARGS=(
    --name "$CONTAINER_NAME"
    -p "$SAGE_PORT:8080"
    -p "$SAGE_METRICS_PORT:9090"
    -e "ETHEREUM_RPC_URL=$ETHEREUM_RPC_URL"
    -e "SAGE_NETWORK=$SAGE_NETWORK"
    -e "LOG_LEVEL=$LOG_LEVEL"
    -v sage-keys:/home/sage/.sage/keys
    -v sage-data:/home/sage/.sage/data
)

# Run based on mode
if [ "$MODE" = "daemon" ]; then
    echo -e "${BLUE}Starting container in daemon mode...${NC}"
    docker run -d "${DOCKER_ARGS[@]}" "$IMAGE_NAME:$VERSION" ${COMMAND:-sage-crypto help}
    echo -e "${GREEN}Container started successfully${NC}"
    echo ""
    echo "View logs: docker logs -f $CONTAINER_NAME"
    echo "Stop container: docker stop $CONTAINER_NAME"
    echo "Remove container: docker rm -f $CONTAINER_NAME"
else
    echo -e "${BLUE}Starting container in interactive mode...${NC}"
    docker run -it --rm "${DOCKER_ARGS[@]}" "$IMAGE_NAME:$VERSION" ${COMMAND:-/bin/sh}
fi
