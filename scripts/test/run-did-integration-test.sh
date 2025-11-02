#!/bin/bash
# SAGE DID Integration Test Script
# This script runs DID integration tests with local Hardhat node

set -e  # Exit on any error

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Project directories
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/../.." && pwd)"
CONTRACT_DIR="$PROJECT_ROOT/contracts/ethereum"

# Log file
LOG_DIR="$PROJECT_ROOT/logs"
mkdir -p "$LOG_DIR"
NODE_LOG="$LOG_DIR/hardhat-node.log"
DEPLOY_LOG="$LOG_DIR/contract-deploy.log"
TEST_LOG="$LOG_DIR/integration-test.log"

# PID file for Hardhat node
PID_FILE="/tmp/sage-hardhat-node.pid"

echo -e "${GREEN}========================================${NC}"
echo -e "${GREEN}SAGE DID Integration Test${NC}"
echo -e "${GREEN}========================================${NC}"
echo ""

# Function to cleanup on exit
cleanup() {
    echo -e "\n${YELLOW}Cleaning up...${NC}"

    # Kill Hardhat node if running
    if [ -f "$PID_FILE" ]; then
        NODE_PID=$(cat "$PID_FILE")
        if ps -p "$NODE_PID" > /dev/null 2>&1; then
            echo "Stopping Hardhat node (PID: $NODE_PID)..."
            kill "$NODE_PID" 2>/dev/null || true
            sleep 2
            # Force kill if still running
            if ps -p "$NODE_PID" > /dev/null 2>&1; then
                kill -9 "$NODE_PID" 2>/dev/null || true
            fi
        fi
        rm -f "$PID_FILE"
    fi

    # Also try pkill as backup
    pkill -f "hardhat node" 2>/dev/null || true

    echo -e "${GREEN}Cleanup complete${NC}"
}

# Set trap to cleanup on script exit
trap cleanup EXIT INT TERM

# Step 1: Check if contracts directory exists
echo -e "${YELLOW}[Step 1/5]${NC} Checking contract directory..."
if [ ! -d "$CONTRACT_DIR" ]; then
    echo -e "${RED}Error: Contract directory not found: $CONTRACT_DIR${NC}"
    exit 1
fi
echo -e "${GREEN}${NC} Contract directory found"
echo ""

# Step 2: Install npm dependencies if needed
echo -e "${YELLOW}[Step 2/5]${NC} Checking npm dependencies..."
if [ ! -d "$CONTRACT_DIR/node_modules" ]; then
    echo "Installing npm dependencies..."
    cd "$CONTRACT_DIR"
    npm install > /dev/null 2>&1
    echo -e "${GREEN}${NC} Dependencies installed"
else
    echo -e "${GREEN}${NC} Dependencies already installed"
fi
echo ""

# Step 3: Start Hardhat node
echo -e "${YELLOW}[Step 3/5]${NC} Starting Hardhat node..."
cd "$CONTRACT_DIR"
npx hardhat node > "$NODE_LOG" 2>&1 &
NODE_PID=$!
echo $NODE_PID > "$PID_FILE"
echo "Hardhat node started (PID: $NODE_PID)"
echo "Logs: $NODE_LOG"

# Wait for node to be ready
echo "Waiting for node to initialize..."
MAX_RETRIES=30
RETRY_COUNT=0
while [ $RETRY_COUNT -lt $MAX_RETRIES ]; do
    if curl -s -X POST http://localhost:8545 \
        -H "Content-Type: application/json" \
        -d '{"jsonrpc":"2.0","method":"eth_blockNumber","params":[],"id":1}' \
        > /dev/null 2>&1; then
        echo -e "${GREEN}${NC} Hardhat node is ready"
        break
    fi
    RETRY_COUNT=$((RETRY_COUNT + 1))
    if [ $RETRY_COUNT -eq $MAX_RETRIES ]; then
        echo -e "${RED}Error: Hardhat node failed to start${NC}"
        echo "Check logs at: $NODE_LOG"
        exit 1
    fi
    sleep 1
done
echo ""

# Step 4: Deploy V4 contract
echo -e "${YELLOW}[Step 4/5]${NC} Deploying V4 contract..."
cd "$CONTRACT_DIR"
npx hardhat run scripts/deploy_v4.js --network localhost > "$DEPLOY_LOG" 2>&1
if [ $? -eq 0 ]; then
    echo -e "${GREEN}${NC} V4 contract deployed"
    echo "Logs: $DEPLOY_LOG"

    # Extract contract address from deploy log
    CONTRACT_ADDRESS=$(grep -o "0x[a-fA-F0-9]\{40\}" "$DEPLOY_LOG" | head -1)
    if [ -n "$CONTRACT_ADDRESS" ]; then
        echo "Contract Address: $CONTRACT_ADDRESS"
    fi
else
    echo -e "${RED}Error: Contract deployment failed${NC}"
    echo "Check logs at: $DEPLOY_LOG"
    exit 1
fi
echo ""

# Step 5: Run integration tests
echo -e "${YELLOW}[Step 5/5]${NC} Running DID integration tests..."
cd "$PROJECT_ROOT"

# Run DID duplicate detection tests (both contract-level and pre-registration check)
echo ""
echo "Running: TestDIDDuplicateDetection and TestDIDPreRegistrationCheck"
echo "========================================"
SAGE_INTEGRATION_TEST=1 go test -v ./pkg/agent/did/ethereum \
    -run 'TestDIDDuplicateDetection|TestDIDPreRegistrationCheck' \
    2>&1 | tee "$TEST_LOG"

TEST_EXIT_CODE=${PIPESTATUS[0]}

echo ""
echo "========================================"
if [ $TEST_EXIT_CODE -eq 0 ]; then
    echo -e "${GREEN} All tests passed${NC}"
    echo "Test logs: $TEST_LOG"
else
    echo -e "${RED} Tests failed (exit code: $TEST_EXIT_CODE)${NC}"
    echo "Test logs: $TEST_LOG"
    exit $TEST_EXIT_CODE
fi

echo ""
echo -e "${GREEN}========================================${NC}"
echo -e "${GREEN}Integration test completed successfully${NC}"
echo -e "${GREEN}========================================${NC}"
