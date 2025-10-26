#!/bin/bash
# SAGE Load Test Runner

set -e

# Colors
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
NC='\033[0m'

# Configuration
SAGE_BASE_URL="${SAGE_BASE_URL:-http://localhost:8080}"
SAGE_ENV="${SAGE_ENV:-local}"
SCENARIO="${1:-baseline}"

# k6 binary check
if ! command -v k6 &> /dev/null; then
    echo -e "${RED}Error: k6 is not installed${NC}"
    echo "Install k6:"
    echo "  macOS:   brew install k6"
    echo "  Linux:   sudo apt-get install k6"
    echo "  Windows: choco install k6"
    echo "  Or: https://k6.io/docs/getting-started/installation/"
    exit 1
fi

echo -e "${GREEN}SAGE Load Test Runner${NC}"
echo "================================"
echo "Scenario:    $SCENARIO"
echo "Base URL:    $SAGE_BASE_URL"
echo "Environment: $SAGE_ENV"
echo ""

# Validate scenario
VALID_SCENARIOS=("baseline" "stress" "soak" "spike" "concurrent-sessions" "did-operations" "hpke-operations" "mixed-workload" "all")
if [[ ! " ${VALID_SCENARIOS[@]} " =~ " ${SCENARIO} " ]]; then
    echo -e "${RED}Error: Invalid scenario '${SCENARIO}'${NC}"
    echo "Valid scenarios: ${VALID_SCENARIOS[*]}"
    exit 1
fi

# Create reports directory
mkdir -p tools/loadtest/reports

# Function to run a single test
run_test() {
    local test_name=$1
    local script_path="tools/loadtest/scenarios/${test_name}.js"

    if [ ! -f "$script_path" ]; then
        echo -e "${RED}Error: Test script not found: ${script_path}${NC}"
        return 1
    fi

    echo -e "${YELLOW}Running ${test_name} test...${NC}"
    echo ""

    # Set environment variables for k6
    export SAGE_BASE_URL
    export SAGE_ENV

    # Run k6 test
    k6 run \
        --out json="tools/loadtest/reports/${test_name}-results.json" \
        --summary-export="tools/loadtest/reports/${test_name}-summary.json" \
        "$script_path"

    local exit_code=$?

    if [ $exit_code -eq 0 ]; then
        echo -e "${GREEN}✅ ${test_name} test completed successfully${NC}"
    else
        echo -e "${RED}❌ ${test_name} test failed${NC}"
    fi

    echo ""
    return $exit_code
}

# Run tests
if [ "$SCENARIO" = "all" ]; then
    echo -e "${YELLOW}Running all load tests...${NC}"
    echo ""

    FAILED=0

    # Core tests
    run_test "baseline" || FAILED=$((FAILED + 1))
    run_test "stress" || FAILED=$((FAILED + 1))
    run_test "spike" || FAILED=$((FAILED + 1))

    # New specialized tests
    run_test "concurrent-sessions" || FAILED=$((FAILED + 1))
    run_test "did-operations" || FAILED=$((FAILED + 1))
    run_test "hpke-operations" || FAILED=$((FAILED + 1))
    run_test "mixed-workload" || FAILED=$((FAILED + 1))

    # Ask before running soak test (it takes hours)
    echo -e "${YELLOW}Soak test takes 2+ hours. Run it? (y/N)${NC}"
    read -r response
    if [[ "$response" =~ ^[Yy]$ ]]; then
        run_test "soak" || FAILED=$((FAILED + 1))
    else
        echo "Skipping soak test"
    fi

    echo ""
    echo "================================"
    if [ $FAILED -eq 0 ]; then
        echo -e "${GREEN}✅ All tests passed${NC}"
        exit 0
    else
        echo -e "${RED}❌ ${FAILED} test(s) failed${NC}"
        exit 1
    fi
else
    run_test "$SCENARIO"
    exit $?
fi
