#!/usr/bin/env bash
# Generate the Go bindings for the AgentCard contracts from the Hardhat
# artifacts with the abigen that matches go.mod's go-ethereum version.
#
# Usage:
#   tools/scripts/gen-bindings.sh [OUT_DIR]
#
# OUT_DIR defaults to pkg/blockchain/ethereum/contracts/agentcardregistry.
# Run `npx hardhat compile` in contracts/ethereum first; `make bindings` does both.
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/../.." && pwd)"
ARTIFACTS="$ROOT/contracts/ethereum/artifacts/contracts"
OUT="${1:-$ROOT/pkg/blockchain/ethereum/contracts/agentcardregistry}"
PKG=agentcardregistry

GETH_VERSION="$(cd "$ROOT" && go list -m -f '{{.Version}}' github.com/ethereum/go-ethereum)"
ABIGEN=(go run "github.com/ethereum/go-ethereum/cmd/abigen@${GETH_VERSION}")

# name -> artifact path (relative to ARTIFACTS)
CONTRACTS=(
  "AgentCardRegistry=AgentCardRegistry.sol/AgentCardRegistry.json"
  "AgentCardStorage=AgentCardStorage.sol/AgentCardStorage.json"
  "IRegistryHook=interfaces/IRegistryHook.sol/IRegistryHook.json"
)

mkdir -p "$OUT"
TMP="$(mktemp -d)"
trap 'rm -rf "$TMP"' EXIT

for entry in "${CONTRACTS[@]}"; do
  name="${entry%%=*}"
  artifact="$ARTIFACTS/${entry#*=}"
  if [ ! -f "$artifact" ]; then
    echo "gen-bindings: missing artifact $artifact (run 'npx hardhat compile' in contracts/ethereum)" >&2
    exit 1
  fi
  jq -c .abi "$artifact" > "$TMP/$name.abi"
  "${ABIGEN[@]}" --abi "$TMP/$name.abi" --pkg "$PKG" --type "$name" --out "$OUT/$name.go"
  echo "gen-bindings: $OUT/$name.go (abigen $GETH_VERSION)"
done
