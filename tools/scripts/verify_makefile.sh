#!/bin/bash

# Makefile Target Verification Script
# Tests all major Makefile targets and reports results

RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

PASSED=0
FAILED=0
SKIPPED=0

echo "========================================"
echo "SAGE Makefile Target Verification"
echo "========================================"
echo ""

# Function to test a target
test_target() {
    local target=$1
    local description=$2
    local skip_reason=$3

    if [ -n "$skip_reason" ]; then
        echo -e "${YELLOW}⊘ SKIP${NC} make $target - $skip_reason"
        ((SKIPPED++))
        return
    fi

    echo -n "Testing: make $target ... "

    if make $target > /tmp/make_test_$$.log 2>&1; then
        echo -e "${GREEN}✓ PASS${NC}"
        ((PASSED++))
    else
        echo -e "${RED}✗ FAIL${NC}"
        echo "  Error output:"
        tail -5 /tmp/make_test_$$.log | sed 's/^/    /'
        ((FAILED++))
    fi

    rm -f /tmp/make_test_$$.log
}

echo "=== BUILD TARGETS ==="
test_target "clean" "Clean build artifacts"
test_target "build-crypto" "Build sage-crypto"
test_target "build-did" "Build sage-did"
test_target "build-verify" "Build sage-verify"
test_target "build-binaries" "Build all binaries"
test_target "build-example-basic-demo" "Build basic-demo example"
test_target "build-example-basic-tool" "Build basic-tool example"
test_target "build-examples" "Build all examples"
test_target "build-random-test" "Build random-test"

echo ""
echo "=== LIBRARY BUILD TARGETS ==="
test_target "build-lib-static" "Build static library"
test_target "build-lib-shared" "Build shared library"
test_target "build-lib-darwin-arm64" "Build darwin-arm64 library"

echo ""
echo "=== TEST TARGETS ==="
test_target "test-crypto" "Test crypto package"
test_target "test-provider" "Test provider package"
test_target "test-vault" "Test vault package"
test_target "test-logger" "Test logger package"
test_target "test-health" "Test health package"
test_target "test-quick" "Quick component tests"

echo ""
echo "=== RANDOM TEST TARGETS ==="
test_target "random-test-quick" "Random test (10 iterations)"

echo ""
echo "=== UTILITY TARGETS ==="
test_target "fmt" "Format code"
test_target "tidy" "Run go mod tidy"
test_target "help" "Show help"

echo ""
echo "=== BLOCKCHAIN TARGETS ==="
test_target "blockchain-status" "Check blockchain status"

echo ""
echo "=== INTEGRATION TEST TARGETS (may fail without environment) ==="
test_target "test-e2e-local" "E2E local tests" "Requires test environment"

echo ""
echo "========================================"
echo "SUMMARY"
echo "========================================"
echo -e "${GREEN}Passed:${NC}  $PASSED"
echo -e "${RED}Failed:${NC}  $FAILED"
echo -e "${YELLOW}Skipped:${NC} $SKIPPED"
echo "Total:   $((PASSED + FAILED + SKIPPED))"
echo ""

if [ $FAILED -eq 0 ]; then
    echo -e "${GREEN}All tests passed!${NC}"
    exit 0
else
    echo -e "${RED}Some tests failed!${NC}"
    exit 1
fi
