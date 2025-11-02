#!/bin/bash

# SAGE Integration Test Script
# This script automatically tests the entire SAGE system.

set -e  # Exit on error

# Color settings
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Function definitions
print_step() {
    echo -e "\n${BLUE}[STEP]${NC} $1"
    echo "=================================================="
}

print_success() {
    echo -e "${GREEN} $1${NC}"
}

print_error() {
    echo -e "${RED} $1${NC}"
}

print_warning() {
    echo -e "${YELLOW}  $1${NC}"
}

# PID storage
PIDS=()

# Port cleanup function
kill_port() {
    local port=$1
    local pids=$(lsof -ti:$port 2>/dev/null || true)
    if [ ! -z "$pids" ]; then
        for pid in $pids; do
            print_warning "Killing process using port $port: PID $pid"
            kill -9 $pid 2>/dev/null || true
        done
        sleep 1
    fi
}

# Cleanup function
cleanup() {
    print_step "Cleaning up test environment"
    
    # Kill registered PIDs
    for pid in "${PIDS[@]}"; do
        if kill -0 $pid 2>/dev/null; then
            print_warning "Killing process: $pid"
            kill -TERM $pid 2>/dev/null || true
            sleep 1
            # Force kill if still running
            if kill -0 $pid 2>/dev/null; then
                kill -9 $pid 2>/dev/null || true
            fi
        fi
    done
    
    # Clean up Hardhat node port (8545)
    kill_port 8545
    
    # Clean up agent ports (3001, 3002, 3003)
    if [ "$1" == "--with-agents" ]; then
        kill_port 3001
        kill_port 3002
        kill_port 3003
    fi
    
    print_success "Cleanup completed"
}

# Cleanup on exit (both normal exit and error)
trap 'cleanup "$@"' EXIT INT TERM

# Main test start
echo "======================================================"
echo "              SAGE Integration Test Start"
echo "======================================================"
echo "Start time: $(date)"
echo ""

# 1. Environment check
print_step "1. Environment check"

# Node.js check
if command -v node &> /dev/null; then
    print_success "Node.js $(node --version)"
else
    print_error "Node.js is not installed"
    exit 1
fi

# Go check
if command -v go &> /dev/null; then
    print_success "Go $(go version)"
else
    print_error "Go is not installed"
    exit 1
fi

# 2. Dependency installation
print_step "2. Dependency installation"

echo "Installing contract dependencies..."
cd ../../contracts/ethereum
if [ ! -d "node_modules" ]; then
    npm install --silent
    print_success "Contract dependencies installation completed"
else
    print_success "Contract dependencies already installed"
fi

# 3. Contract compilation
print_step "3. Smart contract compilation"

npm run clean > /dev/null 2>&1
npm run compile > /dev/null 2>&1
print_success "Contract compilation completed"

# 4. Start Hardhat node
print_step "4. Starting local blockchain"

# Check and kill existing Hardhat process
if lsof -i:8545 &>/dev/null; then
    print_warning "Port 8545 is in use. Killing existing process."
    kill_port 8545
    sleep 2
fi

echo "Starting Hardhat node..."
npx hardhat node > hardhat.log 2>&1 &
HARDHAT_PID=$!
PIDS+=($HARDHAT_PID)
sleep 5

# Check node execution
if kill -0 $HARDHAT_PID 2>/dev/null; then
    # Additional port check
    if lsof -i:8545 &>/dev/null; then
        print_success "Hardhat node started (PID: $HARDHAT_PID)"
    else
        print_error "Hardhat node is running but not bound to port 8545"
        kill -9 $HARDHAT_PID 2>/dev/null || true
        cat hardhat.log | tail -20
        exit 1
    fi
else
    print_error "Hardhat node start failed"
    cat hardhat.log | tail -20
    exit 1
fi

# 5. Contract deployment
print_step "5. Smart contract deployment"

npm run deploy:unified:local > deploy.log 2>&1
if [ $? -eq 0 ]; then
    print_success "Contract deployment successful"
    
    # Extract deployment addresses
    REGISTRY_ADDR=$(grep "SageRegistryV2 deployed to:" deploy.log | awk '{print $NF}')
    HOOK_ADDR=$(grep "SageVerificationHook deployed to:" deploy.log | awk '{print $NF}')
    
    echo "  Registry: $REGISTRY_ADDR"
    echo "  Hook: $HOOK_ADDR"
    
    # Set environment variables
    export SAGE_REGISTRY_ADDRESS=$REGISTRY_ADDR
    export SAGE_VERIFICATION_HOOK_ADDRESS=$HOOK_ADDR
    export SAGE_NETWORK=localhost
    export SAGE_CHAIN_ID=31337
else
    print_error "Contract deployment failed"
    cat deploy.log
    exit 1
fi

# 6. Deployment verification
print_step "6. Deployment verification"

npx hardhat run scripts/verify-deployment.js --network localhost > verify.log 2>&1
if [ $? -eq 0 ]; then
    print_success "Deployment verification successful"
else
    print_warning "Deployment verification failed (continuing)"
    cat verify.log
fi

# 7. Go application test
print_step "7. Go application test"

cd ../../..
echo "Running Go verification tool..."
if go run sage/cmd/sage-verify/main.go > /dev/null 2>&1; then
    print_success "Go application integration confirmed"
else
    print_warning "Go application integration failed (continuing)"
fi

# 8. Unit test execution
print_step "8. Unit test execution"

cd ../../contracts/ethereum
echo "Running tests..."
npm test > test-results.log 2>&1
if [ $? -eq 0 ]; then
    print_success "All tests passed"
else
    print_warning "Some tests failed"
    tail -20 test-results.log
fi

# 9. Start agent servers (optional)
if [ "$1" == "--with-agents" ]; then
    print_step "9. Starting multi-agent system"
    
    cd ../../../../sage-multi-agent
    
    # Root Agent
    PORT=3001 AGENT_TYPE=root go run cli/root/main.go > root.log 2>&1 &
    ROOT_PID=$!
    PIDS+=($ROOT_PID)
    sleep 2
    
    if kill -0 $ROOT_PID 2>/dev/null; then
        print_success "Root Agent started (port 3001)"
    else
        print_error "Root Agent start failed"
    fi
    
    # Ordering Agent
    PORT=3002 AGENT_TYPE=ordering go run cli/ordering/main.go > ordering.log 2>&1 &
    ORDERING_PID=$!
    PIDS+=($ORDERING_PID)
    sleep 2
    
    if kill -0 $ORDERING_PID 2>/dev/null; then
        print_success "Ordering Agent started (port 3002)"
    else
        print_error "Ordering Agent start failed"
    fi
    
    # Planning Agent
    PORT=3003 AGENT_TYPE=planning go run cli/planning/main.go > planning.log 2>&1 &
    PLANNING_PID=$!
    PIDS+=($PLANNING_PID)
    sleep 2
    
    if kill -0 $PLANNING_PID 2>/dev/null; then
        print_success "Planning Agent started (port 3003)"
    else
        print_error "Planning Agent start failed"
    fi
    
    # Agent communication test
    print_step "10. Agent communication test"
    
    sleep 3
    
    # Root Agent 테스트
    response=$(curl -s -X POST http://localhost:3001/api/process \
        -H "Content-Type: application/json" \
        -d '{"message": "test", "sage_enabled": true}' 2>/dev/null || echo "failed")
    
    if [ "$response" != "failed" ]; then
        print_success "Agent communication successful"
    else
        print_error "Agent communication failed"
    fi
fi

# 10. Result summary
print_step "Test result summary"

echo ""
echo "======================================================"
echo "              Test Completed"
echo "======================================================"
echo "End time: $(date)"
echo ""

# Deployment information output
echo " Deployment Information:"
echo "  - Registry: $REGISTRY_ADDR"
echo "  - Hook: $HOOK_ADDR"
echo ""

# Next steps guide
echo " Next Steps:"
echo "  1. Frontend test: cd ../../../sage-fe && npm run dev"
echo "  2. Copy environment variables: cp ../../contracts/ethereum/deployments/localhost.env ../../../.env"
echo "  3. Check logs: tail -f ../../contracts/ethereum/hardhat.log"
echo ""

echo " Log Files:"
echo "  - Hardhat: ../../contracts/ethereum/hardhat.log"
echo "  - Deployment: ../../contracts/ethereum/deploy.log"
echo "  - Tests: ../../contracts/ethereum/test-results.log"

if [ "$1" == "--with-agents" ]; then
    echo "  - Root Agent: sage-multi-agent/root.log"
    echo "  - Ordering Agent: sage-multi-agent/ordering.log"
    echo "  - Planning Agent: sage-multi-agent/planning.log"
fi

echo ""
print_success "All tests completed!"

# Keep processes alive if --keep-alive option is present
if [ "$2" == "--keep-alive" ]; then
    echo ""
    print_warning "Keeping processes running. Press Ctrl+C to exit."
    wait
fi