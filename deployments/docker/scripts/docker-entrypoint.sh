#!/bin/sh
# SAGE Docker Entrypoint Script
# Initializes the container environment before starting the main process

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

echo "${GREEN}SAGE Container Initialization${NC}"
echo "================================"

# Environment validation
echo "${YELLOW}Checking environment...${NC}"

if [ -z "$ETHEREUM_RPC_URL" ]; then
    echo "${RED}ERROR: ETHEREUM_RPC_URL not set${NC}"
    exit 1
fi

echo "RPC URL: $ETHEREUM_RPC_URL"
echo "Network: ${SAGE_NETWORK:-local}"
echo "Chain ID: ${SAGE_CHAIN_ID:-1337}"

# Directory setup
echo "${YELLOW}Setting up directories...${NC}"
mkdir -p ~/.sage/keys
mkdir -p ~/.sage/data
chmod 700 ~/.sage/keys
chmod 755 ~/.sage/data

# Configuration file check
if [ ! -f "/home/sage/config.yaml" ]; then
    echo "${YELLOW}No config.yaml found, using defaults${NC}"
    if [ -f "/home/sage/config.yaml.example" ]; then
        echo "Copying example config..."
        cp /home/sage/config.yaml.example /home/sage/config.yaml
    fi
fi

# Wait for blockchain node (if local)
if [ "$SAGE_NETWORK" = "local" ]; then
    echo "${YELLOW}Waiting for blockchain node...${NC}"

    MAX_RETRIES=30
    RETRY_COUNT=0

    while [ $RETRY_COUNT -lt $MAX_RETRIES ]; do
        if wget -q -O- "$ETHEREUM_RPC_URL" > /dev/null 2>&1; then
            echo "${GREEN}Blockchain node is ready${NC}"
            break
        fi

        RETRY_COUNT=$((RETRY_COUNT + 1))
        echo "Attempt $RETRY_COUNT/$MAX_RETRIES..."
        sleep 2
    done

    if [ $RETRY_COUNT -eq $MAX_RETRIES ]; then
        echo "${RED}ERROR: Blockchain node not responding${NC}"
        exit 1
    fi
fi

# Wait for Redis (if configured)
if [ -n "$REDIS_URL" ]; then
    echo "${YELLOW}Waiting for Redis...${NC}"

    MAX_RETRIES=15
    RETRY_COUNT=0

    while [ $RETRY_COUNT -lt $MAX_RETRIES ]; do
        if nc -z redis 6379 > /dev/null 2>&1; then
            echo "${GREEN}Redis is ready${NC}"
            break
        fi

        RETRY_COUNT=$((RETRY_COUNT + 1))
        echo "Attempt $RETRY_COUNT/$MAX_RETRIES..."
        sleep 1
    done

    if [ $RETRY_COUNT -eq $MAX_RETRIES ]; then
        echo "${YELLOW}WARNING: Redis not responding, proceeding without cache${NC}"
    fi
fi

# Display version information
echo "${YELLOW}Verifying binaries:${NC}"
sage-crypto help >/dev/null 2>&1 && echo "sage-crypto: OK" || echo "sage-crypto: FAILED"
sage-did help >/dev/null 2>&1 && echo "sage-did: OK" || echo "sage-did: Not installed"

echo "${GREEN}Initialization complete${NC}"
echo "================================"
echo ""

# Execute the main command
exec "$@"
