#!/bin/bash
# SAGE Fuzzing Test Runner
# Runs Go fuzzing tests and Solidity fuzzing tests

set -e

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

echo -e "${GREEN}SAGE Fuzzing Test Suite${NC}"
echo "================================"
echo ""

# Parse arguments
FUZZ_TIME="30s"
FUZZ_TYPE="all"
PARALLEL=""

while [[ $# -gt 0 ]]; do
    case $1 in
        --time)
            FUZZ_TIME="$2"
            shift 2
            ;;
        --type)
            FUZZ_TYPE="$2"
            shift 2
            ;;
        --parallel)
            PARALLEL="-parallel $2"
            shift 2
            ;;
        -h|--help)
            echo "Usage: $0 [OPTIONS]"
            echo ""
            echo "Options:"
            echo "  --time TIME       Fuzz time per test (default: 30s)"
            echo "  --type TYPE       Test type: all, go, solidity (default: all)"
            echo "  --parallel N      Run N tests in parallel"
            echo "  -h, --help        Show this help"
            echo ""
            echo "Examples:"
            echo "  $0                              # Run all fuzz tests for 30s"
            echo "  $0 --time 5m --type go          # Run Go fuzz tests for 5 minutes"
            echo "  $0 --time 1m --parallel 4       # Run 4 tests in parallel"
            exit 0
            ;;
        *)
            echo "Unknown option: $1"
            echo "Use -h or --help for usage information"
            exit 1
            ;;
    esac
done

echo "Configuration:"
echo "  Fuzz time: $FUZZ_TIME"
echo "  Test type: $FUZZ_TYPE"
echo ""

# Run Go fuzzing tests
if [ "$FUZZ_TYPE" = "all" ] || [ "$FUZZ_TYPE" = "go" ]; then
    echo -e "${BLUE}Running Go Fuzzing Tests${NC}"
    echo "================================"
    echo ""

    # Crypto fuzzing
    echo -e "${YELLOW}Fuzzing crypto package...${NC}"
    go test -fuzz=FuzzKeyPairGeneration -fuzztime=$FUZZ_TIME $PARALLEL ./crypto || true
    go test -fuzz=FuzzSignAndVerify -fuzztime=$FUZZ_TIME $PARALLEL ./crypto || true
    go test -fuzz=FuzzKeyExportImport -fuzztime=$FUZZ_TIME $PARALLEL ./crypto || true
    go test -fuzz=FuzzSignatureWithDifferentKeys -fuzztime=$FUZZ_TIME $PARALLEL ./crypto || true
    go test -fuzz=FuzzInvalidSignatureData -fuzztime=$FUZZ_TIME $PARALLEL ./crypto || true
    go test -fuzz=FuzzKeyDerivation -fuzztime=$FUZZ_TIME $PARALLEL ./crypto || true
    echo ""

    # Session fuzzing
    echo -e "${YELLOW}Fuzzing session package...${NC}"
    go test -fuzz=FuzzSessionCreation -fuzztime=$FUZZ_TIME $PARALLEL ./session || true
    go test -fuzz=FuzzSessionEncryptDecrypt -fuzztime=$FUZZ_TIME $PARALLEL ./session || true
    go test -fuzz=FuzzNonceValidation -fuzztime=$FUZZ_TIME $PARALLEL ./session || true
    go test -fuzz=FuzzSessionExpiration -fuzztime=$FUZZ_TIME $PARALLEL ./session || true
    go test -fuzz=FuzzConcurrentSessionAccess -fuzztime=$FUZZ_TIME $PARALLEL ./session || true
    go test -fuzz=FuzzInvalidSessionData -fuzztime=$FUZZ_TIME $PARALLEL ./session || true
    echo ""

    echo -e "${GREEN}Go fuzzing complete${NC}"
    echo ""
fi

# Run Solidity fuzzing tests
if [ "$FUZZ_TYPE" = "all" ] || [ "$FUZZ_TYPE" = "solidity" ]; then
    echo -e "${BLUE}Running Solidity Fuzzing Tests (Foundry)${NC}"
    echo "================================"
    echo ""

    # Check if foundry is installed
    if ! command -v forge &> /dev/null; then
        echo -e "${YELLOW}Foundry not installed. Skipping Solidity fuzzing.${NC}"
        echo "Install Foundry: curl -L https://foundry.paradigm.xyz | bash"
        echo ""
    else
        cd contracts/ethereum

        echo -e "${YELLOW}Running Foundry fuzz tests...${NC}"
        forge test --match-test "testFuzz_" -vv || true

        echo ""
        echo -e "${YELLOW}Running invariant tests...${NC}"
        forge test --match-test "invariant_" -vv || true

        cd ../..
        echo ""
        echo -e "${GREEN}Solidity fuzzing complete${NC}"
    fi
fi

echo ""
echo "================================"
echo -e "${GREEN}Fuzzing Summary${NC}"
echo "================================"
echo ""

# Check for crash files
CRASH_FILES=$(find . -name "testdata" -type d 2>/dev/null || true)

if [ -z "$CRASH_FILES" ]; then
    echo -e "${GREEN}No crashes found${NC}"
else
    echo -e "${YELLOW}Crash files found in:${NC}"
    echo "$CRASH_FILES"
    echo ""
    echo "Review crash files with:"
    echo "  go test -fuzz=FuzzTestName -run=FuzzTestName/CRASHHASH"
fi

echo ""
echo -e "${GREEN}Fuzzing complete!${NC}"
echo ""
echo "Tips:"
echo "  - Increase fuzz time for more thorough testing: --time 1h"
echo "  - Use corpus from previous runs: testdata/fuzz/FuzzTestName/"
echo "  - Run specific fuzzer: go test -fuzz=FuzzTestName ./package"
echo "  - Minimize crash: go test -fuzz=FuzzTestName -run=FuzzTestName/CRASHHASH"
