#!/usr/bin/env bash
# Fail if any non-test file under pkg/ declares func init(). Library packages
# must not register themselves as an import side effect; wiring belongs to the
# composition root (internal/app) or the caller. Generated contract bindings
# are exempt.
set -euo pipefail
cd "$(dirname "$0")/../.."
hits=$(grep -rn "^func init()" pkg --include='*.go' | grep -v "_test.go" | grep -v "^pkg/blockchain/ethereum/contracts/" || true)
if [ -n "$hits" ]; then
  echo "init() is not allowed in pkg/ (REFACTORING_DESIGN.md Phase 1.5):"
  echo "$hits"
  exit 1
fi
echo "no init() in pkg/"
