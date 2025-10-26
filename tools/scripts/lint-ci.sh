#!/bin/bash
# Local CI Lint Check Script
# Runs the same lint checks as GitHub Actions

set -e

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

# Get script directory and project root
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/../.." && pwd)"

# Ensure we're in project root
cd "$PROJECT_ROOT"

echo -e "${GREEN}==================================${NC}"
echo -e "${GREEN}Local CI Lint Validation${NC}"
echo -e "${GREEN}==================================${NC}"
echo ""
echo "Simulating GitHub Actions lint workflow..."
echo ""

FAILED=0

# Step 1: golangci-lint (same as GitHub Actions)
echo -e "${BLUE}[1/3] Running golangci-lint...${NC}"
echo "Command: golangci-lint run --timeout=5m"
echo ""

if golangci-lint run --timeout=5m; then
    echo -e "${GREEN}✓ golangci-lint passed${NC}"
else
    echo -e "${RED}✗ golangci-lint failed${NC}"
    FAILED=1
fi
echo ""

# Step 2: go vet (same as GitHub Actions)
echo -e "${BLUE}[2/3] Running go vet...${NC}"
echo "Command: go vet ./..."
echo ""

if go vet ./...; then
    echo -e "${GREEN}✓ go vet passed${NC}"
else
    echo -e "${RED}✗ go vet failed${NC}"
    FAILED=1
fi
echo ""

# Step 3: gofmt check (same as GitHub Actions)
echo -e "${BLUE}[3/3] Checking code formatting...${NC}"
echo "Command: gofmt -l . | grep -v 'contracts/ethereum/bindings/go/example.go'"
echo ""

UNFORMATTED=$(gofmt -l . | grep -v "contracts/ethereum/bindings/go/example.go" || true)
if [ -n "$UNFORMATTED" ]; then
    echo -e "${RED}✗ Go code is not formatted:${NC}"
    echo "$UNFORMATTED"
    echo ""
    echo "Run 'make fmt' to format the code"
    FAILED=1
else
    echo -e "${GREEN}✓ Code formatting check passed${NC}"
fi
echo ""

# Summary
echo -e "${GREEN}==================================${NC}"
if [ $FAILED -eq 0 ]; then
    echo -e "${GREEN}All CI lint checks passed! ✓${NC}"
    echo -e "${GREEN}Safe to commit.${NC}"
    exit 0
else
    echo -e "${RED}Some CI lint checks failed! ✗${NC}"
    echo -e "${YELLOW}Please fix the issues above before committing.${NC}"
    exit 1
fi
