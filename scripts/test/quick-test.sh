#!/bin/bash

# SAGE Quick Test Script
# Simple version with minimal verification

echo "🚀 SAGE Quick Test"
echo "===================="

# Color settings
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
NC='\033[0m' # No Color

# 1. Compile
echo "📝 Compiling contracts..."
cd ../../contracts/ethereum

# Clean up ports (remove previous test residue)
if lsof -i:8545 &>/dev/null; then
    echo -e "${YELLOW}⚠️  Cleaning up port 8545...${NC}"
    lsof -ti:8545 | xargs kill -9 2>/dev/null || true
    sleep 1
fi

npm run compile > /dev/null 2>&1
if [ $? -eq 0 ]; then
    echo -e "${GREEN}✅ Compilation complete${NC}"
else
    echo -e "${RED}❌ Compilation failed${NC}"
    exit 1
fi

# 2. Deploy (using Hardhat network)
echo "🚀 Deploying contracts..."
npm run deploy:unified > deploy-quick.log 2>&1
if [ $? -eq 0 ]; then
    echo -e "${GREEN}✅ Deployment complete${NC}"
else
    echo -e "${RED}❌ Deployment failed${NC}"
    echo "Check logs: cat ../../contracts/ethereum/deploy-quick.log"
    exit 1
fi

# 3. Check deployment info
echo ""
echo "📊 Deployment Result:"
if [ -f "deployments/hardhat.json" ]; then
    echo "✅ Deployment info saved"
    
    # Use jq if available, otherwise use grep
    if command -v jq &> /dev/null; then
        REGISTRY=$(jq -r '.contracts.SageRegistryV2.address' deployments/hardhat.json)
        AGENTS=$(jq -r '.agents | length' deployments/hardhat.json)
    else
        REGISTRY=$(grep -o '"SageRegistryV2".*"address":"[^"]*"' deployments/hardhat.json | grep -o '0x[a-fA-F0-9]*')
        AGENTS=$(grep -c '"did":' deployments/hardhat.json || echo 0)
    fi
    
    echo "  Registry: $REGISTRY"
    echo "  Registered Agents: $AGENTS"
else
    echo "❌ Deployment info not found"
fi

echo ""
echo "✨ Quick test completed!"
echo ""
echo "For detailed testing:"
echo "  sage/scripts/test/test-sage.sh                    # Basic test"
echo "  sage/scripts/test/test-sage.sh --with-agents      # With agents"
echo "  sage/scripts/test/test-sage.sh --with-agents --keep-alive  # Keep running"