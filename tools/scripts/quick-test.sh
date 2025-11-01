#!/bin/bash

# Quick test runner for SAGE core components
# Runs only essential tests for rapid feedback

# Get the sage directory (parent of scripts)
SAGE_DIR="$(cd "$(dirname "$0")/.." && pwd)"
cd "$SAGE_DIR"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

echo "================================================"
echo "SAGE Core Library - Quick Test"
echo "================================================"
echo ""

TOTAL=0
PASSED=0
FAILED=0

# Function to run test and track results
run_test() {
    local package=$1
    local description=$2

    echo -n "Testing $description... "
    TOTAL=$((TOTAL + 1))

    if go test -short -timeout 10s "$package" > /dev/null 2>&1; then
        echo -e "${GREEN}${NC}"
        PASSED=$((PASSED + 1))
    else
        echo -e "${RED}${NC}"
        FAILED=$((FAILED + 1))
    fi
}

# Core RFC9421 implementation (most critical)
run_test "./core/rfc9421" "RFC 9421 HTTP Message Signatures"

# Key management
run_test "./crypto/keys" "Cryptographic key management"

# Secure storage
run_test "./crypto/vault" "Secure vault storage"

# Message validation
run_test "./core/message/validator" "Message validation"

# DID management
run_test "./did" "DID core functionality"

echo ""
echo "================================================"
echo "Quick Test Summary"
echo "================================================"
echo -e "Total: ${TOTAL}, Passed: ${GREEN}${PASSED}${NC}, Failed: ${RED}${FAILED}${NC}"

if [ $FAILED -eq 0 ]; then
    echo -e "${GREEN} All essential tests passed!${NC}"
    exit 0
else
    echo -e "${RED} Some tests failed${NC}"
    exit 1
fi