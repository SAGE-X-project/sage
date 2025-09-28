#!/bin/bash

# Phase 1 완료 테스트 스크립트
# 모든 구현된 컴포넌트의 테스트를 실행하고 결과를 검증

set -e

echo "================================================"
echo "SAGE Core Library - Phase 1 Completion Test"
echo "================================================"
echo ""

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Test results
TOTAL_TESTS=0
PASSED_TESTS=0
FAILED_TESTS=0
FAILED_PACKAGES=()

# Function to run tests for a package
run_test() {
    local package=$1
    local description=$2

    echo -e "${YELLOW}Testing:${NC} $description"
    echo "Package: $package"
    echo "----------------------------------------"

    TOTAL_TESTS=$((TOTAL_TESTS + 1))

    # Already in sage directory, no need to cd

    if go test -v -count=1 -timeout 30s "$package" 2>&1 | tee test_output.tmp; then
        echo -e "${GREEN}✓ PASSED${NC}: $description\n"
        PASSED_TESTS=$((PASSED_TESTS + 1))
    else
        echo -e "${RED}✗ FAILED${NC}: $description\n"
        FAILED_TESTS=$((FAILED_TESTS + 1))
        FAILED_PACKAGES+=("$package - $description")
    fi

    # Extract test statistics
    if grep -q "PASS" test_output.tmp; then
        grep -E "(PASS|ok)" test_output.tmp | tail -1
    fi

    echo ""
    rm -f test_output.tmp
}

# Function to check if package exists
check_package() {
    local package=$1
    local dir_path="${package#./}"
    if [ -d "$dir_path" ]; then
        return 0
    else
        echo -e "${RED}Package not found:${NC} $dir_path"
        return 1
    fi
}

echo "Starting Phase 1 component tests..."
echo ""

# 1. Test Enhanced Provider (블록체인 연동)
echo "================================================"
echo "1. BLOCKCHAIN INTEGRATION TESTS"
echo "================================================"
if check_package "./crypto/chain/ethereum"; then
    run_test "./crypto/chain/ethereum" "Enhanced Provider with retry logic"
fi

# 2. Test SecureVault (키 보안)
echo "================================================"
echo "2. SECURE VAULT TESTS"
echo "================================================"
if check_package "./crypto/vault"; then
    run_test "./crypto/vault" "AES-256 encrypted key storage"
fi

# 3. Test Logger (구조화된 로깅)
echo "================================================"
echo "3. STRUCTURED LOGGING TESTS"
echo "================================================"
if check_package "./internal/logger"; then
    run_test "./internal/logger" "Structured logging system"
fi

# 4. Test Health Checker (헬스체크)
echo "================================================"
echo "4. HEALTH CHECK TESTS"
echo "================================================"
if check_package "./health"; then
    run_test "./health" "System health monitoring"
fi

# 5. Test existing core components
echo "================================================"
echo "5. CORE COMPONENTS TESTS"
echo "================================================"

# Test blockchain config
if check_package "./config"; then
    run_test "./config" "Blockchain configuration"
fi

# Test RFC9421 implementation
if check_package "./core/rfc9421"; then
    run_test "./core/rfc9421" "RFC 9421 HTTP Message Signatures"
fi

# Test crypto keys
if check_package "./crypto/keys"; then
    run_test "./crypto/keys" "Cryptographic key management"
fi

# Test DID management
if check_package "./did/ethereum"; then
    run_test "./did/ethereum" "Ethereum DID resolver"
fi

echo ""
echo "================================================"
echo "TEST SUMMARY"
echo "================================================"
echo -e "Total Tests Run: ${TOTAL_TESTS}"
echo -e "Passed: ${GREEN}${PASSED_TESTS}${NC}"
echo -e "Failed: ${RED}${FAILED_TESTS}${NC}"

if [ ${FAILED_TESTS} -gt 0 ]; then
    echo ""
    echo -e "${RED}Failed Packages:${NC}"
    for pkg in "${FAILED_PACKAGES[@]}"; do
        echo "  - $pkg"
    done
fi

# Calculate success rate
if [ ${TOTAL_TESTS} -gt 0 ]; then
    SUCCESS_RATE=$((PASSED_TESTS * 100 / TOTAL_TESTS))
    echo ""
    echo -e "Success Rate: ${SUCCESS_RATE}%"

    if [ ${SUCCESS_RATE} -ge 90 ]; then
        echo -e "${GREEN}✓ Phase 1 testing PASSED with ${SUCCESS_RATE}% success rate${NC}"
    elif [ ${SUCCESS_RATE} -ge 70 ]; then
        echo -e "${YELLOW}⚠ Phase 1 testing PARTIALLY PASSED with ${SUCCESS_RATE}% success rate${NC}"
    else
        echo -e "${RED}✗ Phase 1 testing FAILED with only ${SUCCESS_RATE}% success rate${NC}"
    fi
fi

echo ""
echo "================================================"
echo "PHASE 1 COMPLETION CHECK"
echo "================================================"

# Check for required implementations
echo "Checking Phase 1 requirements..."
echo ""

REQUIREMENTS_MET=0
REQUIREMENTS_TOTAL=0

# Already in sage directory

# Check 1: Enhanced Provider
REQUIREMENTS_TOTAL=$((REQUIREMENTS_TOTAL + 1))
if [ -f "crypto/chain/ethereum/enhanced_provider.go" ] && [ -f "crypto/chain/ethereum/enhanced_provider_test.go" ]; then
    echo -e "${GREEN}✓${NC} Enhanced Provider implementation and tests exist"
    REQUIREMENTS_MET=$((REQUIREMENTS_MET + 1))
else
    echo -e "${RED}✗${NC} Enhanced Provider implementation or tests missing"
fi

# Check 2: SecureVault
REQUIREMENTS_TOTAL=$((REQUIREMENTS_TOTAL + 1))
if [ -f "crypto/vault/secure_storage.go" ] && [ -f "crypto/vault/secure_storage_test.go" ]; then
    echo -e "${GREEN}✓${NC} SecureVault implementation and tests exist"
    REQUIREMENTS_MET=$((REQUIREMENTS_MET + 1))
else
    echo -e "${RED}✗${NC} SecureVault implementation or tests missing"
fi

# Check 3: Logger
REQUIREMENTS_TOTAL=$((REQUIREMENTS_TOTAL + 1))
if [ -f "internal/logger/logger.go" ] && [ -f "internal/logger/logger_test.go" ]; then
    echo -e "${GREEN}✓${NC} Structured Logger implementation and tests exist"
    REQUIREMENTS_MET=$((REQUIREMENTS_MET + 1))
else
    echo -e "${RED}✗${NC} Structured Logger implementation or tests missing"
fi

# Check 4: Health Checker
REQUIREMENTS_TOTAL=$((REQUIREMENTS_TOTAL + 1))
if [ -f "health/checker.go" ] && [ -f "health/checker_test.go" ]; then
    echo -e "${GREEN}✓${NC} Health Checker implementation and tests exist"
    REQUIREMENTS_MET=$((REQUIREMENTS_MET + 1))
else
    echo -e "${RED}✗${NC} Health Checker implementation or tests missing"
fi

# Check 5: Blockchain Config
REQUIREMENTS_TOTAL=$((REQUIREMENTS_TOTAL + 1))
if [ -f "config/blockchain.go" ]; then
    echo -e "${GREEN}✓${NC} Blockchain configuration exists"
    REQUIREMENTS_MET=$((REQUIREMENTS_MET + 1))
else
    echo -e "${RED}✗${NC} Blockchain configuration missing"
fi

echo ""
echo "----------------------------------------"
echo -e "Requirements Met: ${REQUIREMENTS_MET}/${REQUIREMENTS_TOTAL}"

if [ ${REQUIREMENTS_MET} -eq ${REQUIREMENTS_TOTAL} ]; then
    echo -e "${GREEN}✓ All Phase 1 requirements are implemented${NC}"
    echo ""
    echo "Phase 1 Features Completed:"
    echo "  • Blockchain connection with retry logic"
    echo "  • Gas estimation and optimization"
    echo "  • AES-256 encrypted key storage"
    echo "  • Structured JSON logging"
    echo "  • Comprehensive health checks"
    echo "  • Configuration management"

    # Final exit code based on test results
    if [ ${FAILED_TESTS} -eq 0 ]; then
        echo ""
        echo -e "${GREEN}🎉 Phase 1 FULLY COMPLETED AND TESTED!${NC}"
        exit 0
    else
        echo ""
        echo -e "${YELLOW}⚠ Phase 1 implemented but some tests failed${NC}"
        exit 1
    fi
else
    echo -e "${RED}✗ Phase 1 requirements not fully met${NC}"
    exit 1
fi