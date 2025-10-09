#!/bin/bash

# SAGE MCP Examples - Compilation Test Script
# Tests that all examples compile successfully

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
cd "$SCRIPT_DIR"

echo "======================================"
echo "SAGE MCP Examples - Compilation Test"
echo "======================================"
echo ""

# Colors for output
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

EXAMPLES=(
    "basic-demo"
    "basic-tool"
    "client"
    "simple-standalone"
    "vulnerable-vs-secure/vulnerable-chat"
    "vulnerable-vs-secure/secure-chat"
    "vulnerable-vs-secure/attacker"
)

PASSED=0
FAILED=0
TOTAL=${#EXAMPLES[@]}

for example in "${EXAMPLES[@]}"; do
    echo -n "Testing $example... "

    if [ ! -d "$example" ]; then
        echo -e "${RED}[SKIP - Directory not found]${NC}"
        ((FAILED++))
        continue
    fi

    cd "$example"

    # Try to build
    if go build -o /tmp/sage-test-build . > /dev/null 2>&1; then
        echo -e "${GREEN}[PASS]${NC}"
        ((PASSED++))
        rm -f /tmp/sage-test-build
    else
        echo -e "${RED}[FAIL]${NC}"
        ((FAILED++))
        echo "  Error output:"
        go build . 2>&1 | head -5 | sed 's/^/    /'
    fi

    cd "$SCRIPT_DIR"
done

echo ""
echo "======================================"
echo "Test Results:"
echo "  Total:  $TOTAL"
echo -e "  ${GREEN}Passed: $PASSED${NC}"
if [ $FAILED -gt 0 ]; then
    echo -e "  ${RED}Failed: $FAILED${NC}"
else
    echo "  Failed: 0"
fi
echo "======================================"

if [ $FAILED -gt 0 ]; then
    exit 1
fi

exit 0
