#!/bin/bash

# SAGE Agent Initialization Examples Runner
# This script helps you run the agent initialization examples

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Function to print colored output
print_status() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

print_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

print_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

print_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# Function to check if Go is installed
check_go() {
    if ! command -v go &> /dev/null; then
        print_error "Go is not installed. Please install Go 1.24.0 or higher."
        exit 1
    fi
    
    go_version=$(go version | cut -d' ' -f3 | cut -d'o' -f2)
    print_success "Go version: $go_version"
}

# Function to check if we're in the right directory
check_directory() {
    if [ ! -f "go.mod" ] || [ ! -d "pkg/agent" ]; then
        print_error "Please run this script from the SAGE project root directory"
        exit 1
    fi
    print_success "Running from SAGE project directory"
}

# Function to run simple agent init example
run_simple_example() {
    print_status "Running Simple Agent Initialization Example..."
    echo "=========================================="
    
    cd examples/simple-agent-init
    
    # First run - should generate keys
    print_status "First run (generating keys)..."
    go run main.go
    
    echo ""
    print_status "Second run (loading existing keys)..."
    go run main.go
    
    echo ""
    print_status "Third run (loading existing keys again)..."
    go run main.go
    
    cd ../..
    print_success "Simple example completed!"
}

# Function to run full agent init example
run_full_example() {
    print_status "Running Full Agent Initialization Example..."
    echo "=========================================="
    
    cd examples/agent-initialization
    
    # First run - should generate keys
    print_status "First run (generating keys)..."
    go run main.go
    
    echo ""
    print_status "Second run (loading existing keys)..."
    go run main.go
    
    echo ""
    print_status "Third run with blockchain registration..."
    REGISTER_ON_CHAIN=true go run main.go
    
    cd ../..
    print_success "Full example completed!"
}

# Function to clean up generated files
cleanup() {
    print_status "Cleaning up generated files..."
    
    # Remove key directories
    if [ -d "examples/simple-agent-init/keys" ]; then
        rm -rf examples/simple-agent-init/keys
        print_success "Removed simple-agent-init/keys"
    fi
    
    if [ -d "examples/agent-initialization/keys" ]; then
        rm -rf examples/agent-initialization/keys
        print_success "Removed agent-initialization/keys"
    fi
    
    # Remove agent card files
    if [ -f "examples/agent-initialization/agent-card.json" ]; then
        rm -f examples/agent-initialization/agent-card.json
        print_success "Removed agent-card.json"
    fi
    
    print_success "Cleanup completed!"
}

# Function to show help
show_help() {
    echo "SAGE Agent Initialization Examples Runner"
    echo ""
    echo "Usage: $0 [OPTION]"
    echo ""
    echo "Options:"
    echo "  simple    Run simple agent initialization example"
    echo "  full      Run full agent initialization example"
    echo "  both      Run both examples"
    echo "  clean     Clean up generated files"
    echo "  help      Show this help message"
    echo ""
    echo "Examples:"
    echo "  $0 simple     # Run simple example only"
    echo "  $0 full       # Run full example only"
    echo "  $0 both       # Run both examples"
    echo "  $0 clean      # Clean up generated files"
}

# Main script logic
main() {
    echo "╔═══════════════════════════════════════════════════════════╗"
    echo "║     SAGE Agent Initialization Examples Runner            ║"
    echo "╚═══════════════════════════════════════════════════════════╝"
    echo ""
    
    # Check prerequisites
    check_go
    check_directory
    
    # Parse command line arguments
    case "${1:-help}" in
        "simple")
            run_simple_example
            ;;
        "full")
            run_full_example
            ;;
        "both")
            run_simple_example
            echo ""
            run_full_example
            ;;
        "clean")
            cleanup
            ;;
        "help"|"-h"|"--help")
            show_help
            ;;
        *)
            print_error "Unknown option: $1"
            show_help
            exit 1
            ;;
    esac
    
    echo ""
    print_success "All done! 🎉"
}

# Run main function with all arguments
main "$@"
