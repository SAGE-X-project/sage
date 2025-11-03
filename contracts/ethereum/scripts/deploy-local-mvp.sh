#!/bin/bash

# SAGE MVP - Local Blockchain Setup & Deploy Script
# This script starts a local Hardhat node and deploys AgentCardRegistry

set -e

echo ""
echo "═══════════════════════════════════════════════════════"
echo " SAGE MVP - Local Blockchain Setup"
echo "═══════════════════════════════════════════════════════"
echo ""

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Navigate to contracts directory
SCRIPT_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"
cd "$SCRIPT_DIR/.."

echo -e "${BLUE}📂 Working directory: $(pwd)${NC}"
echo ""

# Check if Hardhat is installed
if ! command -v npx &> /dev/null; then
    echo -e "${RED}❌ Error: npx not found. Please install Node.js and npm${NC}"
    exit 1
fi

# Check for existing Hardhat node
HARDHAT_PID=$(lsof -ti:8545 2>/dev/null || echo "")
if [ -n "$HARDHAT_PID" ]; then
    echo -e "${YELLOW}⚠️  Port 8545 is already in use (PID: $HARDHAT_PID)${NC}"
    echo -e "${YELLOW}   Stopping existing process...${NC}"
    kill -9 $HARDHAT_PID 2>/dev/null || true
    sleep 2
fi

# Start Hardhat node in background
echo -e "${BLUE}🚀 Starting Hardhat local node on port 8545...${NC}"
npx hardhat node > /tmp/hardhat-node.log 2>&1 &
HARDHAT_NODE_PID=$!

echo -e "${GREEN}✓ Hardhat node started (PID: $HARDHAT_NODE_PID)${NC}"
echo -e "${BLUE}   Log file: /tmp/hardhat-node.log${NC}"
echo ""

# Wait for node to be ready
echo -e "${BLUE}⏳ Waiting for node to be ready...${NC}"
for i in {1..30}; do
    if curl -s -X POST -H "Content-Type: application/json" --data '{"jsonrpc":"2.0","method":"eth_blockNumber","params":[],"id":1}' http://localhost:8545 > /dev/null 2>&1; then
        echo -e "${GREEN}✓ Node is ready!${NC}"
        break
    fi
    sleep 1
    if [ $i -eq 30 ]; then
        echo -e "${RED}❌ Timeout waiting for node${NC}"
        kill -9 $HARDHAT_NODE_PID 2>/dev/null || true
        exit 1
    fi
done
echo ""

# Deploy contracts
echo -e "${BLUE}📝 Deploying AgentCardRegistry contracts...${NC}"
echo ""

npx hardhat run scripts/deploy-agentcard.js --network localhost

echo ""
echo -e "${GREEN}✅ Deployment complete!${NC}"
echo ""

# Read deployment info
LATEST_DEPLOYMENT="deployments/localhost-latest.json"
if [ -f "$LATEST_DEPLOYMENT" ]; then
    REGISTRY_ADDRESS=$(node -p "require('./$LATEST_DEPLOYMENT').contracts.AgentCardRegistry.address")
    HOOK_ADDRESS=$(node -p "require('./$LATEST_DEPLOYMENT').contracts.AgentCardVerifyHook.address")

    echo "═══════════════════════════════════════════════════════"
    echo " Contract Addresses (Local Network)"
    echo "═══════════════════════════════════════════════════════"
    echo ""
    echo -e "${GREEN}AgentCardRegistry:   ${REGISTRY_ADDRESS}${NC}"
    echo -e "${GREEN}AgentCardVerifyHook: ${HOOK_ADDRESS}${NC}"
    echo ""
    echo "═══════════════════════════════════════════════════════"
    echo " Environment Variables"
    echo "═══════════════════════════════════════════════════════"
    echo ""
    echo "Add these to your .env files:"
    echo ""
    echo "# sage-fe/.env.local"
    echo "NEXT_PUBLIC_ETHEREUM_RPC_URL=http://localhost:8545"
    echo "NEXT_PUBLIC_AGENT_REGISTRY_ADDRESS=${REGISTRY_ADDRESS}"
    echo "NEXT_PUBLIC_CHAIN_ID=31337"
    echo ""
    echo "# sage-payment-agent-for-demo/.env.local"
    echo "BLOCKCHAIN_RPC_URL=http://localhost:8545"
    echo "CONTRACT_ADDRESS=${REGISTRY_ADDRESS}"
    echo "CHAIN_ID=31337"
    echo ""
    echo "═══════════════════════════════════════════════════════"
    echo " Hardhat Node Info"
    echo "═══════════════════════════════════════════════════════"
    echo ""
    echo -e "${BLUE}Process ID:${NC} $HARDHAT_NODE_PID"
    echo -e "${BLUE}RPC URL:${NC}    http://localhost:8545"
    echo -e "${BLUE}Chain ID:${NC}   31337"
    echo -e "${BLUE}Log file:${NC}   /tmp/hardhat-node.log"
    echo ""
    echo -e "${YELLOW}To stop the node:${NC}"
    echo "  kill -9 $HARDHAT_NODE_PID"
    echo ""
    echo -e "${GREEN}✅ Setup complete! The local blockchain is running.${NC}"
    echo ""
else
    echo -e "${RED}❌ Error: Deployment info not found${NC}"
    kill -9 $HARDHAT_NODE_PID 2>/dev/null || true
    exit 1
fi

# Save PID for later cleanup
echo $HARDHAT_NODE_PID > /tmp/hardhat-node.pid
echo -e "${BLUE}ℹ️  Node PID saved to /tmp/hardhat-node.pid${NC}"
echo ""
