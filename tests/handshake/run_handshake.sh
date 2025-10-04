#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR=$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)
BIN_DIR=$(mktemp -d "${ROOT_DIR}/.tmp-e2e-XXXXXX")
SERVER_LOG="${BIN_DIR}/server.log"
CLIENT_LOG="${BIN_DIR}/client.log"
SERVER_PIPE="${BIN_DIR}/server.pipe"

SERVER_PID=""
TAIL_PID=""
READER_PID=""

kill_if_alive() {
  local pid="${1:-}"
  [[ -n "$pid" ]] || return 0
  if kill -0 "$pid" 2>/dev/null; then
    kill "$pid" 2>/dev/null || true
    wait "$pid" 2>/dev/null || true
  fi
}

cleanup() {
  kill_if_alive "$READER_PID"
  kill_if_alive "$TAIL_PID"
  kill_if_alive "$SERVER_PID"
  rm -rf "${BIN_DIR}" >/dev/null 2>&1 || true
}
trap cleanup EXIT INT TERM

echo "[http e2e] building binaries..."
(
  cd "${ROOT_DIR}" && \
  go build -o "${BIN_DIR}/e2e-test-server" ./tests/handshake/server && \
  go build -o "${BIN_DIR}/e2e-test-client" ./tests/handshake/client
)

echo "[http e2e] starting test server..."
"${BIN_DIR}/e2e-test-server" >"${SERVER_LOG}" 2>&1 &
SERVER_PID=$!

# show server log live using a named pipe (reliable PIDs)
mkfifo "${SERVER_PIPE}"

{
  while IFS= read -r line; do
    printf '[server] %s\n' "$line"
  done < "${SERVER_PIPE}"
} &
READER_PID=$!

tail -n0 -F "${SERVER_LOG}" > "${SERVER_PIPE}" &
TAIL_PID=$!

wait_for_http() {
  local url=$1
  local tries=${2:-60}
  local delay=${3:-0.2}
  for ((i=0; i < tries; i++)); do
    if curl --silent --max-time 1 "$url" >/dev/null 2>&1; then
      return 0
    fi
    sleep "$delay"
  done
  return 1
}

if ! wait_for_http "http://127.0.0.1:8080/debug/server-pub" 100 0.2; then
  echo "[http e2e] server failed to start" >&2
  echo "--- server log ---" >&2
  cat "${SERVER_LOG}" >&2 || true
  exit 1
fi

echo "[http e2e] running client scenario..."
: >"${CLIENT_LOG}"

set +e
"${BIN_DIR}/e2e-test-client" 2>&1 | tee -a "${CLIENT_LOG}"
CLIENT_STATUS=${PIPESTATUS[0]}
set -e

if [[ ${CLIENT_STATUS} -ne 0 ]]; then
  echo "[http e2e] client exited with status ${CLIENT_STATUS}" >&2
  echo "--- client log ---" >&2
  cat "${CLIENT_LOG}" >&2 || true
  echo "--- server log ---" >&2
  cat "${SERVER_LOG}" >&2 || true
  exit ${CLIENT_STATUS}
fi

if ! grep -q "STATUS: 200" "${SERVER_LOG}"; then
  echo "[http e2e] expected 200 not found in server log" >&2
  echo "--- client log ---" >&2
  cat "${CLIENT_LOG}" >&2 || true
  echo "--- server log ---" >&2
  cat "${SERVER_LOG}" >&2 || true
  exit 1
fi

# stop server and log tails cleanly
kill_if_alive "$SERVER_PID"
kill_if_alive "$TAIL_PID"
kill_if_alive "$READER_PID"

echo "[http e2e] success"
