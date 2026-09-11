#!/usr/bin/env bash
# The Go toolchain version is declared once in go.mod (`toolchain goX.Y.Z`).
# The Dockerfiles and the GitHub workflows must build with the same version;
# this script fails when any of them drifts.
set -euo pipefail
cd "$(dirname "$0")/../.."

want=$(sed -n 's/^toolchain go\([0-9.]*\).*/\1/p' go.mod)
if [ -z "$want" ]; then
  echo "go.mod has no toolchain directive" >&2
  exit 1
fi

fail=0
for f in Dockerfile deployments/docker/Dockerfile; do
  [ -f "$f" ] || continue
  got=$(sed -n 's/^FROM golang:\([0-9.]*\)-alpine.*/\1/p' "$f" | head -1)
  if [ "$got" != "$want" ]; then echo "$f: golang image $got, go.mod toolchain $want" >&2; fail=1; fi
done
for f in .github/workflows/*.yml; do
  while IFS= read -r got; do
    if [ "$got" != "$want" ]; then echo "$f: GO_VERSION $got, go.mod toolchain $want" >&2; fail=1; fi
  done < <(sed -n "s/^ *GO_VERSION: *'\([0-9.]*\)'.*/\1/p" "$f")
  while IFS= read -r got; do
    if [ "$got" != "$want" ]; then echo "$f: go-version matrix $got, go.mod toolchain $want" >&2; fail=1; fi
  done < <(sed -n "s/^ *go-version: *\['\([0-9.]*\)'\].*/\1/p" "$f")
done
[ $fail -eq 0 ] && echo "Go version $want is consistent across go.mod, Dockerfiles and workflows"
exit $fail
