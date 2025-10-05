#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR=$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)
BIN_DIR=$(mktemp -d "${ROOT_DIR}/.tmp-e2e-XXXXXX")
SERVER_LOG="${BIN_DIR}/server.log"
CLIENT_LOG="${BIN_DIR}/client.log"
SERVER_PIPE="${BIN_DIR}/server.pipe"

SERVER_PID=""; TAIL_PID=""; READER_PID=""

# ----- colors (console only) -----
if [[ -t 1 ]]; then
  BLU=$'\033[34m'; GRN=$'\033[32m'; YLW=$'\033[33m'; RED=$'\033[31m'; NC=$'\033[0m'
else
  BLU=""; GRN=""; YLW=""; RED=""; NC=""
fi

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

echo "${YLW}[http e2e] building binaries...${NC}"
(
  cd "${ROOT_DIR}" && \
  go build -o "${BIN_DIR}/e2e-test-server" ./tests/handshake/server && \
  go build -o "${BIN_DIR}/e2e-test-client" ./tests/handshake/client
)

echo "${YLW}[http e2e] starting test server...${NC}"
"${BIN_DIR}/e2e-test-server" >"${SERVER_LOG}" 2>&1 &
SERVER_PID=$!

# ----- server log: 파일은 원본, 콘솔에는 색/접두어 -----
mkfifo "${SERVER_PIPE}"
{
  while IFS= read -r line; do
    printf '%s[server]%s %s\n' "$BLU" "$NC" "$line"
  done < "${SERVER_PIPE}"
} &
READER_PID=$!

tail -n0 -F "${SERVER_LOG}" > "${SERVER_PIPE}" &
TAIL_PID=$!

wait_for_http() {
  local url=$1 tries=${2:-60} delay=${3:-0.2}
  for ((i=0; i<tries; i++)); do
    if curl --silent --max-time 1 "$url" >/dev/null 2>&1; then
      return 0
    fi
    sleep "$delay"
  done
  return 1
}

if ! wait_for_http "http://127.0.0.1:8080/debug/server-pub" 100 0.2; then
  echo "${RED}[http e2e] server failed to start${NC}" >&2
  echo "--- server log ---" >&2
  cat "${SERVER_LOG}" >&2 || true
  exit 1
fi

echo "${YLW}[http e2e] running client scenario...${NC}"
: >"${CLIENT_LOG}"

# ----- client log: 파일은 원본, 콘솔에는 초록색/접두어 -----
set +e
"${BIN_DIR}/e2e-test-client" 2>&1 \
  | tee >(awk -v p="[client]" -v c="${GRN}" 'BEGIN{nc="\033[0m"} {printf "%s%s%s %s\n", c, p, nc, $0}') \
  >> "${CLIENT_LOG}"
CLIENT_STATUS=${PIPESTATUS[0]}
set -e

if [[ ${CLIENT_STATUS} -ne 0 ]]; then
  echo "${RED}[http e2e] client exited with status ${CLIENT_STATUS}${NC}" >&2
  echo "--- client log ---" >&2;  cat "${CLIENT_LOG}" >&2 || true
  echo "--- server log ---" >&2;  cat "${SERVER_LOG}" >&2 || true
  exit ${CLIENT_STATUS}
fi

# 성공 판정: 서버 로그 원본에서 STATUS: 200 확인 (색 없음)
if ! grep -q "STATUS: 200" "${SERVER_LOG}"; then
  echo "${RED}[http e2e] expected 200 not found in server log${NC}" >&2
  echo "--- client log ---" >&2;  cat "${CLIENT_LOG}" >&2 || true
  echo "--- server log ---" >&2;  cat "${SERVER_LOG}" >&2 || true
  exit 1
fi

# 종료 정리
kill_if_alive "$SERVER_PID"
kill_if_alive "$TAIL_PID"
kill_if_alive "$READER_PID"

echo "${YLW}[http e2e] success${NC}"
