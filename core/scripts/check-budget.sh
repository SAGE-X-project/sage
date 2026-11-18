#!/usr/bin/env bash
# Checks the rules of the core module that the compiler does not enforce:
# 1. the dependency budget: only the standard library and the modules below;
# 2. no console output, environment access, init functions or global logger
#    in non-test code.
# Run from the core/ directory.
set -euo pipefail

allowed='^(github\.com/sage-x-project/sage/core(/|$)|golang\.org/x/crypto/|golang\.org/x/sys/cpu$|github\.com/decred/dcrd/dcrec/secp256k1/v4(/|$)|filippo\.io/edwards25519(/|$))'

status=0

deps=$(go list -deps -test -f '{{if not .Standard}}{{.ImportPath}}{{end}}' ./... | sort -u)
while IFS= read -r pkg; do
  [ -z "$pkg" ] && continue
  # Test binaries of core packages are listed as "pkg.test" or "pkg [pkg.test]".
  base=${pkg%% *}
  base=${base%.test}
  if ! printf '%s\n' "$base" | grep -Eq "$allowed"; then
    printf 'dependency outside the budget: %s\n' "$pkg" >&2
    status=1
  fi
done <<<"$deps"

forbidden='fmt\.Print|os\.Getenv|os\.LookupEnv|os\.Environ|os\.ExpandEnv|syscall\.Getenv|os\.Stdout|os\.Stderr|^func init\(\)|(^|[^.A-Za-z_])print(ln)?\(|"log"$|log\.(Print|Fatal|Panic|Default)|slog\.(Default|SetDefault|Debug|Info|Warn|Error|Log)\('
matches=$(grep -rnE "$forbidden" --include='*.go' . | grep -v '_test\.go:' || true)
if [ -n "$matches" ]; then
  printf 'forbidden pattern in non-test code:\n%s\n' "$matches" >&2
  status=1
fi

# Package-level variables: only error sentinels made with errors.New and
# compile-time interface assertions (var _ T = ...) are allowed.
vars=$(grep -rnE '^var ' --include='*.go' . | grep -v '_test\.go:' \
  | grep -vE '^[^:]+:[0-9]+:var Err[A-Za-z0-9]* = errors\.New\(' \
  | grep -vE '^[^:]+:[0-9]+:var _ ' || true)
if [ -n "$vars" ]; then
  printf 'package-level variable in non-test code:\n%s\n' "$vars" >&2
  status=1
fi

if [ "$status" -eq 0 ]; then
  echo "core budget: ok"
fi
exit "$status"
