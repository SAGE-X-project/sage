#!/usr/bin/env bash
# Generate the Go bindings for the AgentCard contracts from the ABI files
# published by the sage-contracts repository, with the abigen that matches
# go.mod's go-ethereum version.
#
# Usage:
#   tools/scripts/gen-bindings.sh [OUT_DIR]
#
# OUT_DIR defaults to pkg/blockchain/ethereum/contracts/agentcardregistry.
# The ABIs are read from $CONTRACTS_DIR/abi (default: .sage-contracts, a
# checkout of SAGE-X-project/sage-contracts at the commit in
# .contracts-version; `make contracts-checkout` creates it).
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/../.." && pwd)"
CONTRACTS_DIR="${CONTRACTS_DIR:-$ROOT/.sage-contracts}"
ABI_DIR="$CONTRACTS_DIR/abi"
OUT="${1:-$ROOT/pkg/blockchain/ethereum/contracts/agentcardregistry}"
PKG=agentcardregistry

GETH_VERSION="$(cd "$ROOT" && go list -m -f '{{.Version}}' github.com/ethereum/go-ethereum)"
ABIGEN=(go run "github.com/ethereum/go-ethereum/cmd/abigen@${GETH_VERSION}")

CONTRACTS=(AgentCardRegistry AgentCardStorage IRegistryHook)

if [ ! -d "$ABI_DIR" ]; then
  echo "gen-bindings: $ABI_DIR not found; run 'make contracts-checkout' or set CONTRACTS_DIR" >&2
  exit 1
fi

mkdir -p "$OUT"
for name in "${CONTRACTS[@]}"; do
  abi="$ABI_DIR/$name.json"
  if [ ! -f "$abi" ]; then
    echo "gen-bindings: missing ABI $abi" >&2
    exit 1
  fi
  "${ABIGEN[@]}" --abi "$abi" --pkg "$PKG" --type "$name" --out "$OUT/$name.go"
  echo "gen-bindings: $OUT/$name.go (abigen $GETH_VERSION)"
done
