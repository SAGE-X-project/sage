#!/bin/bash

# Integration Test Environment Setup Script
# Sets up local blockchain for SAGE testing

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Configuration
CHAIN_ID=31337
RPC_URL="http://localhost:8545"
MNEMONIC="test test test test test test test test test test test junk"
DEPLOY_ACCOUNT="0xf39Fd6e51aad88F6F4ce6aB8827279cffFb92266"
DEPLOY_KEY="0xac0974bec39a17e36ba4a6b4d238ff944bacb478cbed5efcae784d7bf4f2ff80"

echo -e "${BLUE}================================================${NC}"
echo -e "${BLUE}SAGE Integration Test Environment Setup${NC}"
echo -e "${BLUE}================================================${NC}"
echo ""

# Function to check if command exists
command_exists() {
    command -v "$1" >/dev/null 2>&1
}

# Function to wait for service
wait_for_service() {
    local url=$1
    local max_attempts=30
    local attempt=0

    echo -n "Waiting for service at $url"
    while [ $attempt -lt $max_attempts ]; do
        if curl -s -X POST "$url" -H "Content-Type: application/json" \
           -d '{"jsonrpc":"2.0","method":"eth_blockNumber","params":[],"id":1}' >/dev/null 2>&1; then
            echo -e " ${GREEN}✓${NC}"
            return 0
        fi
        echo -n "."
        sleep 1
        attempt=$((attempt + 1))
    done

    echo -e " ${RED}✗${NC}"
    return 1
}

# Function to start local blockchain
start_local_blockchain() {
    echo -e "${YELLOW}Starting local blockchain...${NC}"

    # Check if Hardhat is installed
    # Use separate check to prevent set -e from terminating script if hardhat is not found
    HARDHAT_AVAILABLE=false
    if command_exists npx; then
        # Temporarily disable exit-on-error for this check
        set +e
        npx hardhat --version >/dev/null 2>&1 && HARDHAT_AVAILABLE=true
        set -e
    fi

    if [ "$HARDHAT_AVAILABLE" = "true" ]; then
        echo "Using Hardhat node..."
        # Use fork if FORK_URL is set, otherwise run local network
        if [ -n "$FORK_URL" ]; then
            npx hardhat node --port 8545 --chain-id $CHAIN_ID --fork "$FORK_URL" >/dev/null 2>&1 &
        else
            npx hardhat node --port 8545 --chain-id $CHAIN_ID >/dev/null 2>&1 &
        fi
        BLOCKCHAIN_PID=$!

    # Check if Anvil (Foundry) is installed
    elif command_exists anvil; then
        echo "Using Anvil (Foundry)..."
        anvil --chain-id $CHAIN_ID --mnemonic "$MNEMONIC" --port 8545 &
        BLOCKCHAIN_PID=$!

    # Check if Ganache is installed
    elif command_exists ganache; then
        echo "Using Ganache..."
        ganache --chain.chainId $CHAIN_ID --wallet.mnemonic "$MNEMONIC" --server.port 8545 &
        BLOCKCHAIN_PID=$!

    # Check if ganache-cli is installed
    elif command_exists ganache-cli; then
        echo "Using ganache-cli..."
        ganache-cli --chainId $CHAIN_ID --mnemonic "$MNEMONIC" --port 8545 &
        BLOCKCHAIN_PID=$!

    else
        echo -e "${RED}Error: No local blockchain tool found!${NC}"
        echo "Please install one of the following:"
        echo "  - Hardhat: npm install --save-dev hardhat"
        echo "  - Foundry: curl -L https://foundry.paradigm.xyz | bash"
        echo "  - Ganache: npm install -g ganache"
        exit 1
    fi

    # Store PID for cleanup
    echo $BLOCKCHAIN_PID > .blockchain.pid

    # Wait for blockchain to be ready
    if wait_for_service "$RPC_URL"; then
        echo -e "${GREEN}Local blockchain started successfully (PID: $BLOCKCHAIN_PID)${NC}"
    else
        echo -e "${RED}Failed to start local blockchain${NC}"
        kill $BLOCKCHAIN_PID 2>/dev/null
        rm -f .blockchain.pid
        exit 1
    fi
}

# Function to stop local blockchain
stop_local_blockchain() {
    echo -e "${YELLOW}Stopping local blockchain...${NC}"

    if [ -f .blockchain.pid ]; then
        PID=$(cat .blockchain.pid)
        if kill -0 $PID 2>/dev/null; then
            kill $PID
            echo -e "${GREEN}Blockchain stopped${NC}"
        fi
        rm -f .blockchain.pid
    fi
}

# Function to deploy contracts
deploy_contracts() {
    echo -e "${YELLOW}Deploying smart contracts...${NC}"

    # Check if contracts directory exists
    if [ -d "../contracts/ethereum" ]; then
        # Change to contracts directory
        cd ../contracts/ethereum
    else
        echo -e "${YELLOW}No contracts directory found, skipping deployment${NC}"
        return 0
    fi

    # Check if deployment script exists
    if [ -f scripts/deploy.js ] || [ -f scripts/deploy.ts ]; then
        echo "Running deployment script..."
        npx hardhat run scripts/deploy.* --network localhost
    elif [ -f deploy.sh ]; then
        echo "Running deployment shell script..."
        ./deploy.sh
    else
        echo -e "${YELLOW}Warning: No deployment script found${NC}"
        echo "Creating basic deployment..."

        # Create a basic deployment script
        cat > temp_deploy.js << 'EOF'
const hre = require("hardhat");

async function main() {
    console.log("Deploying SAGE Registry...");

    // Get deployer account
    const [deployer] = await hre.ethers.getSigners();
    console.log("Deploying with account:", deployer.address);

    // Deploy registry contract
    const Registry = await hre.ethers.getContractFactory("SAGERegistry");
    const registry = await Registry.deploy();
    await registry.deployed();

    console.log("SAGE Registry deployed to:", registry.address);

    // Save deployment info
    const fs = require("fs");
    const deploymentInfo = {
        network: "localhost",
        chainId: 31337,
        contracts: {
            SAGERegistry: registry.address
        },
        deployer: deployer.address,
        timestamp: new Date().toISOString()
    };

    fs.writeFileSync(
        "deployments/localhost.json",
        JSON.stringify(deploymentInfo, null, 2)
    );

    console.log("Deployment info saved to deployments/localhost.json");
}

main()
    .then(() => process.exit(0))
    .catch((error) => {
        console.error(error);
        process.exit(1);
    });
EOF

        # Try to run the temporary deployment
        if command_exists npx && [ -f hardhat.config.js ] || [ -f hardhat.config.ts ]; then
            npx hardhat run temp_deploy.js --network localhost
        fi

        rm -f temp_deploy.js
    fi

    cd - > /dev/null

    echo -e "${GREEN}Contracts deployed${NC}"
}

# Function to setup test accounts
setup_test_accounts() {
    echo -e "${YELLOW}Setting up test accounts...${NC}"

    # Create test accounts file
    cat > test_accounts.json << EOF
{
    "deployer": {
        "address": "$DEPLOY_ACCOUNT",
        "privateKey": "$DEPLOY_KEY"
    },
    "alice": {
        "address": "0x70997970C51812dc3A010C7d01b50e0d17dc79C8",
        "privateKey": "0x59c6995e998f97a5a0044966f0945389dc9e86dae88c7a8412f4603b6b78690d"
    },
    "bob": {
        "address": "0x3C44CdDdB6a900fa2b585dd299e03d12FA4293BC",
        "privateKey": "0x5de4111afa1a4b94908f83103eb1f1706367c2e68ca870fc3fb9a804cdab365a"
    },
    "charlie": {
        "address": "0x90F79bf6EB2c4f870365E785982E1f101E93b906",
        "privateKey": "0x7c852118294e51e653712a81e05800f419141751be58f605c371e15141b007a6"
    }
}
EOF

    echo -e "${GREEN}Test accounts created${NC}"
    echo "Test accounts saved to: test_accounts.json"
}

# Function to run integration tests
run_integration_tests() {
    echo -e "${YELLOW}Running integration tests...${NC}"

    # Export environment variables
    export SAGE_RPC_URL=$RPC_URL
    export SAGE_CHAIN_ID=$CHAIN_ID
    export SAGE_PRIVATE_KEY=$DEPLOY_KEY

    # Run Go integration tests
    echo "Running Go integration tests..."
    go test -v ./tests/integration/... -tags=integration

    echo -e "${GREEN}Integration tests completed${NC}"
}

# Main execution
main() {
    case "${1:-}" in
        start)
            start_local_blockchain
            setup_test_accounts
            deploy_contracts
            echo ""
            echo -e "${GREEN}Test environment ready!${NC}"
            echo "RPC URL: $RPC_URL"
            echo "Chain ID: $CHAIN_ID"
            echo ""
            ;;

        stop)
            stop_local_blockchain
            ;;

        restart)
            stop_local_blockchain
            sleep 2
            start_local_blockchain
            setup_test_accounts
            deploy_contracts
            ;;

        test)
            # Set trap only for test command to cleanup on exit
            trap 'stop_local_blockchain' EXIT INT TERM

            if ! curl -s -X POST "$RPC_URL" >/dev/null 2>&1; then
                echo "Starting test environment first..."
                start_local_blockchain
                setup_test_accounts
                deploy_contracts
            fi
            run_integration_tests
            ;;

        status)
            if curl -s -X POST "$RPC_URL" -H "Content-Type: application/json" \
               -d '{"jsonrpc":"2.0","method":"eth_blockNumber","params":[],"id":1}' >/dev/null 2>&1; then
                echo -e "${GREEN}✓${NC} Local blockchain is running at $RPC_URL"

                # Get block number
                BLOCK=$(curl -s -X POST "$RPC_URL" -H "Content-Type: application/json" \
                    -d '{"jsonrpc":"2.0","method":"eth_blockNumber","params":[],"id":1}' \
                    | grep -o '"result":"[^"]*"' | cut -d'"' -f4)
                echo "  Current block: $((16#${BLOCK#0x}))"
            else
                echo -e "${RED}✗${NC} Local blockchain is not running"
            fi
            ;;

        clean)
            stop_local_blockchain
            rm -f test_accounts.json
            rm -f .blockchain.pid
            echo -e "${GREEN}Cleaned up test environment${NC}"
            ;;

        *)
            echo "SAGE Integration Test Environment"
            echo ""
            echo "Usage: $0 {start|stop|restart|test|status|clean}"
            echo ""
            echo "Commands:"
            echo "  start   - Start local blockchain and deploy contracts"
            echo "  stop    - Stop local blockchain"
            echo "  restart - Restart local blockchain"
            echo "  test    - Run integration tests"
            echo "  status  - Check blockchain status"
            echo "  clean   - Clean up test environment"
            echo ""
            ;;
    esac
}

# Run main function (trap is set inside main for specific commands)
main "$@"