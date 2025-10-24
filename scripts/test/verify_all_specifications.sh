#!/bin/bash

# SAGE Specification Verification Script
# Runs all tests defined in SPECIFICATION_VERIFICATION_MATRIX.md

set -e

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
PURPLE='\033[0;35m'
CYAN='\033[0;36m'
NC='\033[0m'
BOLD='\033[1m'

# Setup
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/../.." && pwd)"
LOG_DIR="$PROJECT_ROOT/logs/verification"
TIMESTAMP=$(date +%Y%m%d_%H%M%S)
REPORT_FILE="$LOG_DIR/verification_report_${TIMESTAMP}.md"

mkdir -p "$LOG_DIR"

# Counters
TOTAL_CHAPTERS=9
PASSED=0
FAILED=0
SKIPPED=0

# Print functions
print_header() {
    echo ""
    echo -e "${BOLD}${CYAN}╔══════════════════════════════════════════════════════════════╗${NC}"
    printf "${BOLD}${CYAN}║  %-58s  ║${NC}\n" "$1"
    echo -e "${BOLD}${CYAN}╚══════════════════════════════════════════════════════════════╝${NC}"
    echo ""
}

print_section() {
    echo ""
    echo -e "${BOLD}${PURPLE}═══ $1 ═══${NC}"
}

print_pass() { echo -e "${GREEN}✅ PASS:${NC} $1"; }
print_fail() { echo -e "${RED}❌ FAIL:${NC} $1"; }
print_skip() { echo -e "${YELLOW}⏭️  SKIP:${NC} $1"; }
print_info() { echo -e "${CYAN}ℹ️  INFO:${NC} $1"; }

# Check prerequisites
check_prereqs() {
    print_section "Checking Prerequisites"
    
    command -v go >/dev/null 2>&1 || { print_fail "Go not installed"; exit 1; }
    print_pass "Go $(go version | awk '{print $3}')"
    
    [ -f "$PROJECT_ROOT/go.mod" ] || { print_fail "Not in SAGE project root"; exit 1; }
    print_pass "In SAGE project root"
    
    if curl -s -X POST http://localhost:8545 -H "Content-Type: application/json" \
        -d '{"jsonrpc":"2.0","method":"eth_chainId","params":[],"id":1}' >/dev/null 2>&1; then
        print_pass "Hardhat node running"
    else
        print_skip "Hardhat node not running"
    fi
}

# Run chapter test
run_chapter() {
    local num=$1
    local name=$2
    shift 2
    local packages="$@"
    
    print_section "Chapter $num: $name"
    
    local log_file="$LOG_DIR/chapter${num}.log"
    
    if go test -v $packages > "$log_file" 2>&1; then
        print_pass "Chapter $num: $name"
        PASSED=$((PASSED + 1))
        return 0
    else
        print_fail "Chapter $num: $name (see $log_file)"
        FAILED=$((FAILED + 1))
        return 1
    fi
}

# Run all chapters
run_all_tests() {
    cd "$PROJECT_ROOT" || exit 1
    
    run_chapter 1 "RFC 9421" github.com/sage-x-project/sage/pkg/agent/core/rfc9421 || true
    
    run_chapter 2 "Key Management" \
        github.com/sage-x-project/sage/pkg/agent/crypto/keys \
        github.com/sage-x-project/sage/pkg/agent/crypto/rotation \
        github.com/sage-x-project/sage/pkg/agent/crypto/storage \
        github.com/sage-x-project/sage/pkg/agent/crypto/vault || true
    
    run_chapter 3 "DID Management" \
        github.com/sage-x-project/sage/pkg/agent/did/... || true
    
    # Check Hardhat for Chapter 4
    if curl -s -X POST http://localhost:8545 -H "Content-Type: application/json" \
        -d '{"jsonrpc":"2.0","method":"eth_chainId","params":[],"id":1}' >/dev/null 2>&1; then
        run_chapter 4 "Blockchain Integration" \
            ./tests -run "TestBlockchain|TestProvider|TestChainID|TestTransaction|TestGas|TestContract" || true
    else
        print_skip "Chapter 4: Blockchain Integration (Hardhat not running)"
        SKIPPED=$((SKIPPED + 1))
    fi
    
    run_chapter 5 "Message Processing" \
        github.com/sage-x-project/sage/pkg/agent/core/message/... \
        github.com/sage-x-project/sage/pkg/agent/transport/... || true
    
    run_chapter 6 "CLI Tools" \
        github.com/sage-x-project/sage/cmd/... || true
    
    run_chapter 7 "Session Management" \
        github.com/sage-x-project/sage/pkg/agent/session \
        github.com/sage-x-project/sage/pkg/agent/handshake || true
    
    run_chapter 8 "HPKE" \
        github.com/sage-x-project/sage/pkg/agent/hpke || true
    
    run_chapter 9 "Health Check" \
        github.com/sage-x-project/sage/pkg/health || true
}

# Generate report
generate_report() {
    local tested=$((TOTAL_CHAPTERS - SKIPPED))
    local success_rate="N/A"
    if [ $tested -gt 0 ]; then
        success_rate=$(awk "BEGIN {printf \"%.1f\", ($PASSED/$tested)*100}")
    fi
    
    cat > "$REPORT_FILE" << EOF
# SAGE Specification Verification Report

**Date**: $(date '+%Y-%m-%d %H:%M:%S')
**Project**: SAGE (Secure Agent Guarantee Engine)

## Summary

\`\`\`
Total Chapters:     $TOTAL_CHAPTERS
✅ Passed:          $PASSED
❌ Failed:          $FAILED
⏭️  Skipped:         $SKIPPED
Success Rate:       ${success_rate}%
\`\`\`

## Logs

All test logs are in: \`$LOG_DIR\`

EOF
    
    print_info "Report: $REPORT_FILE"
}

# Print summary
print_summary() {
    print_header "VERIFICATION SUMMARY"
    
    echo -e "${BOLD}Results:${NC}"
    echo -e "  Total:    $TOTAL_CHAPTERS"
    echo -e "  ${GREEN}Passed:${NC}   $PASSED"
    echo -e "  ${RED}Failed:${NC}   $FAILED"
    echo -e "  ${YELLOW}Skipped:${NC}  $SKIPPED"
    
    local tested=$((TOTAL_CHAPTERS - SKIPPED))
    if [ $tested -gt 0 ]; then
        local rate=$(awk "BEGIN {printf \"%.1f\", ($PASSED/$tested)*100}")
        echo -e "  ${BOLD}Rate:${NC}     ${rate}%"
    fi
    
    echo ""
    echo -e "${BOLD}Files:${NC}"
    echo -e "  Report: $REPORT_FILE"
    echo -e "  Logs:   $LOG_DIR"
    echo ""
}

# Main
main() {
    print_header "SAGE Specification Verification"
    
    print_info "Starting at $(date)"
    print_info "Project: $PROJECT_ROOT"
    print_info "Logs: $LOG_DIR"
    
    check_prereqs
    run_all_tests
    generate_report
    print_summary
    
    if [ $FAILED -gt 0 ]; then
        print_fail "Completed with $FAILED failure(s)"
        exit 1
    else
        print_pass "All tests passed!"
        exit 0
    fi
}

main "$@"
