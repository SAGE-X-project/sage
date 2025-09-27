#!/bin/bash

# Simple test runner for Phase 1 components
# Script should be run from the sage directory

echo "Testing Enhanced Provider..."
go test ./crypto/chain/ethereum -v -run TestRetryWithBackoff 2>&1 | grep -E "(PASS|FAIL|ok)" | tail -5
echo ""

echo "Testing SecureVault..."
go test ./crypto/vault -v -count=1 2>&1 | grep -E "(PASS|FAIL|ok)" | tail -5
echo ""

echo "Testing Logger..."
go test ./internal/logger -v -count=1 2>&1 | grep -E "(PASS|FAIL|ok)" | tail -5
echo ""

echo "Testing HealthChecker..."
go test ./health -v -count=1 2>&1 | grep -E "(PASS|FAIL|ok)" | tail -5
echo ""