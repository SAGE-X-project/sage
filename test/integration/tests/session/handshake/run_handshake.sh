#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "$0")" && pwd)"
BIN_DIR="$(mktemp -d "${ROOT_DIR}/.tmp-hpke-e2e-XXXXXX")"
SERVER_BIN="${BIN_DIR}/hpke-e2e-server"
CLIENT_BIN="${BIN_DIR}/hpke-e2e-client"
SERVER_LOG="${BIN_DIR}/server.log"

if [[ -t 1 ]]; then
  GRN=$'\033[32m'; RED=$'\033[31m'; YLW=$'\033[33m'; NC=$'\033[0m'
else
  GRN=""; RED=""; YLW=""; NC=""
fi

cleanup () {
  [[ -n "${SERVER_PID:-}" ]] && kill "${SERVER_PID}" 2>/dev/null || true
  rm -rf "${BIN_DIR}" || true
}
trap cleanup EXIT INT TERM

echo "${YLW}[build] building server/client...${NC}"
go build -tags="integration,a2a" -o "${SERVER_BIN}" "${ROOT_DIR}/server"
go build -tags="integration,a2a" -o "${CLIENT_BIN}" "${ROOT_DIR}/client"

echo "${YLW}[start] launching server...${NC}"
"${SERVER_BIN}" >"${SERVER_LOG}" 2>&1 &
SERVER_PID=$!

# wait server
for i in {1..60}; do
  if curl -fsS http://127.0.0.1:8080/debug/health >/dev/null 2>&1; then
    break
  fi
  sleep 0.2
done
if ! curl -fsS http://127.0.0.1:8080/debug/health >/dev/null 2>&1; then
  echo "${RED}[fatal] server not responding${NC}"
  cat "${SERVER_LOG}" || true
  exit 1
fi

echo "${YLW}[run] running client scenario...${NC}"
set +e
"${CLIENT_BIN}"
STATUS=$?
set -e

echo "----- server log (tail) -----"
tail -n 200 "${SERVER_LOG}" || true

if [[ $STATUS -ne 0 ]]; then
  echo "${RED}[fail] client exited with status ${STATUS}${NC}"
  exit $STATUS
fi

# Success criteria: ensure "server -> status 200" appears in the server log
if ! grep -q "server -> status 200" "${SERVER_LOG}"; then
  echo "${RED}[fail] expected 200 not found in server log${NC}"
  exit 1
fi

echo "${GRN}[ok] e2e finished successfully${NC}"
