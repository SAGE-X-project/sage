#!/bin/bash

# SAGE Port Management Script
# Manages ports for Hardhat nodes and agent servers.

# Color settings
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Default ports
HARDHAT_PORT=8545
AGENT_PORTS=(3001 3002 3003)
ALL_PORTS=(8545 3001 3002 3003)

print_header() {
    echo -e "${BLUE}=====================================${NC}"
    echo -e "${BLUE}     SAGE Port Management Tool${NC}"
    echo -e "${BLUE}=====================================${NC}"
}

check_port() {
    local port=$1
    local pids=$(lsof -ti:$port 2>/dev/null)
    
    if [ -z "$pids" ]; then
        echo -e "${GREEN} Port $port: Available${NC}"
        return 0
    else
        echo -e "${YELLOW}  Port $port: In use${NC}"
        for pid in $pids; do
            local process=$(ps -p $pid -o comm= 2>/dev/null || echo "Unknown")
            echo "     PID: $pid (Process: $process)"
        done
        return 1
    fi
}

kill_port() {
    local port=$1
    local pids=$(lsof -ti:$port 2>/dev/null)
    
    if [ -z "$pids" ]; then
        echo -e "${GREEN}Port $port is already free.${NC}"
        return 0
    fi
    
    echo -e "${YELLOW}Terminating processes using port $port...${NC}"
    for pid in $pids; do
        echo "  Terminating: PID $pid"
        kill -TERM $pid 2>/dev/null || true
        sleep 1
        
        # Force kill if still running
        if kill -0 $pid 2>/dev/null; then
            echo "  Force killing: PID $pid"
            kill -9 $pid 2>/dev/null || true
        fi
    done
    
    sleep 1
    
    # Verification
    if lsof -i:$port &>/dev/null; then
        echo -e "${RED} Failed to clean port $port${NC}"
        return 1
    else
        echo -e "${GREEN} Port $port cleaned successfully${NC}"
        return 0
    fi
}

case "$1" in
    status)
        print_header
        echo -e "\n${BLUE}[Port Status Check]${NC}"
        echo ""
        
        echo "Hardhat Node:"
        check_port $HARDHAT_PORT
        
        echo ""
        echo "Agent Servers:"
        check_port 3001 && echo "  Root Agent (3001)"
        check_port 3002 && echo "  Ordering Agent (3002)"
        check_port 3003 && echo "  Planning Agent (3003)"
        ;;
        
    clean)
        print_header
        echo -e "\n${BLUE}[Port Cleanup]${NC}"
        echo ""
        
        if [ "$2" == "--all" ]; then
            echo "Cleaning all ports..."
            for port in "${ALL_PORTS[@]}"; do
                kill_port $port
            done
        elif [ "$2" == "--hardhat" ]; then
            echo "Cleaning Hardhat port..."
            kill_port $HARDHAT_PORT
        elif [ "$2" == "--agents" ]; then
            echo "Cleaning agent ports..."
            for port in "${AGENT_PORTS[@]}"; do
                kill_port $port
            done
        elif [ ! -z "$2" ]; then
            echo "Cleaning port $2..."
            kill_port $2
        else
            echo -e "${RED}Usage: $0 clean [--all|--hardhat|--agents|port_number]${NC}"
            exit 1
        fi
        ;;
        
    check)
        print_header
        port=${2:-8545}
        echo -e "\n${BLUE}[Detailed Port $port Check]${NC}"
        echo ""
        
        if lsof -i:$port &>/dev/null; then
            echo -e "${YELLOW}Port $port in use:${NC}"
            lsof -i:$port
        else
            echo -e "${GREEN}Port $port is available.${NC}"
        fi
        ;;
        
    start-hardhat)
        print_header
        echo -e "\n${BLUE}[Starting Hardhat Node]${NC}"
        echo ""
        
        # Port check
        if lsof -i:$HARDHAT_PORT &>/dev/null; then
            echo -e "${YELLOW}Port $HARDHAT_PORT is in use. Cleaning...${NC}"
            kill_port $HARDHAT_PORT
        fi
        
        echo "Starting Hardhat node..."
        npx hardhat node &
        HARDHAT_PID=$!
        
        sleep 3
        
        if kill -0 $HARDHAT_PID 2>/dev/null; then
            echo -e "${GREEN} Hardhat node started successfully (PID: $HARDHAT_PID)${NC}"
            echo ""
            echo "To stop the node: $0 clean --hardhat"
        else
            echo -e "${RED} Failed to start Hardhat node${NC}"
        fi
        ;;
        
    help|*)
        print_header
        echo ""
        echo "Usage: $0 {command} [options]"
        echo ""
        echo "Commands:"
        echo "  status              - Check all port status"
        echo "  check [port]        - Detailed check of specific port (default: 8545)"
        echo "  clean --all         - Clean all ports"
        echo "  clean --hardhat     - Clean Hardhat port (8545)"
        echo "  clean --agents      - Clean agent ports (3001-3003)"
        echo "  clean [port]        - Clean specific port"
        echo "  start-hardhat       - Start Hardhat node (includes port cleanup)"
        echo "  help                - Show this help message"
        echo ""
        echo "Examples:"
        echo "  $0 status           # Check all port status"
        echo "  $0 clean --all      # Clean all ports"
        echo "  $0 clean 8545       # Clean only Hardhat port"
        echo "  $0 start-hardhat    # Start Hardhat node"
        ;;
esac

echo ""