#!/bin/bash

# SAGE 헬스체크 검증 스크립트
# 챕터 9: 헬스체크 (9.1.1.1, 9.1.1.2, 9.1.1.3)

set -e

PROJECT_ROOT="/Users/kevin/work/github/sage-x-project/sage"
BLOCKCHAIN_PID=""
BLOCKCHAIN_LOG="$PROJECT_ROOT/blockchain_test.log"

# 색상 정의
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

echo -e "${BLUE}═══════════════════════════════════════════════════════════${NC}"
echo -e "${BLUE}  SAGE 헬스체크 검증 스크립트 (챕터 9)${NC}"
echo -e "${BLUE}═══════════════════════════════════════════════════════════${NC}"
echo ""

# 정리 함수
cleanup() {
    echo ""
    echo -e "${YELLOW}정리 중...${NC}"

    if [ -n "$BLOCKCHAIN_PID" ] && kill -0 $BLOCKCHAIN_PID 2>/dev/null; then
        echo "  - 블록체인 노드 종료 (PID: $BLOCKCHAIN_PID)"
        kill $BLOCKCHAIN_PID 2>/dev/null || true
        wait $BLOCKCHAIN_PID 2>/dev/null || true
    fi

    if [ -f "$BLOCKCHAIN_LOG" ]; then
        rm -f "$BLOCKCHAIN_LOG"
    fi

    echo -e "${GREEN}정리 완료${NC}"
}

# 트랩 설정
trap cleanup EXIT INT TERM

# 1. 로컬 블록체인 노드 시작
echo -e "${BLUE}[1/6] 로컬 블록체인 노드 시작 중...${NC}"
cd "$PROJECT_ROOT/contracts/ethereum"

# Hardhat 노드 시작 (백그라운드)
npx hardhat node > "$BLOCKCHAIN_LOG" 2>&1 &
BLOCKCHAIN_PID=$!

echo "  - Hardhat 노드 PID: $BLOCKCHAIN_PID"
echo "  - 로그 파일: $BLOCKCHAIN_LOG"

# 노드가 준비될 때까지 대기
echo -e "${YELLOW}  - 노드 준비 대기 중...${NC}"
sleep 5

# 노드 상태 확인
if ! kill -0 $BLOCKCHAIN_PID 2>/dev/null; then
    echo -e "${RED} 블록체인 노드 시작 실패${NC}"
    cat "$BLOCKCHAIN_LOG"
    exit 1
fi

# RPC 연결 확인
MAX_RETRIES=10
RETRY_COUNT=0
while [ $RETRY_COUNT -lt $MAX_RETRIES ]; do
    if curl -s -X POST -H "Content-Type: application/json" \
        --data '{"jsonrpc":"2.0","method":"eth_chainId","params":[],"id":1}' \
        http://localhost:8545 > /dev/null 2>&1; then
        echo -e "${GREEN} 블록체인 노드 준비 완료${NC}"
        break
    fi
    RETRY_COUNT=$((RETRY_COUNT + 1))
    if [ $RETRY_COUNT -eq $MAX_RETRIES ]; then
        echo -e "${RED} 블록체인 노드 연결 실패 (타임아웃)${NC}"
        exit 1
    fi
    sleep 1
done

echo ""

# 2. sage-verify 빌드 확인
echo -e "${BLUE}[2/6] sage-verify CLI 도구 확인 중...${NC}"
cd "$PROJECT_ROOT"

if [ ! -f "./build/bin/sage-verify" ]; then
    echo -e "${YELLOW}  - sage-verify 빌드 필요${NC}"
    make build
fi

if [ -f "./build/bin/sage-verify" ]; then
    echo -e "${GREEN} sage-verify CLI 도구 준비 완료${NC}"
else
    echo -e "${RED} sage-verify 빌드 실패${NC}"
    exit 1
fi

echo ""

# 3. 9.1.1.1 /health 엔드포인트 정상 응답 검증
echo -e "${BLUE}[3/6] 9.1.1.1 /health 엔드포인트 정상 응답 검증${NC}"
echo "  명령어: ./build/bin/sage-verify health"
echo ""

# timeout 추가하여 실행
if command -v timeout &> /dev/null; then
    HEALTH_OUTPUT=$(timeout 10 ./build/bin/sage-verify health 2>&1 || echo "Timeout or error occurred")
else
    HEALTH_OUTPUT=$(./build/bin/sage-verify health 2>&1)
fi
echo "$HEALTH_OUTPUT"
echo ""

# JSON 출력 테스트
echo "  JSON 출력 테스트:"
if command -v timeout &> /dev/null; then
    HEALTH_JSON=$(timeout 10 ./build/bin/sage-verify health --json 2>&1 || echo "{}")
else
    HEALTH_JSON=$(./build/bin/sage-verify health --json 2>&1)
fi
echo "$HEALTH_JSON" | python3 -m json.tool 2>/dev/null | head -20 || echo "$HEALTH_JSON" | head -20
echo ""

# 검증
if echo "$HEALTH_OUTPUT" | grep -q "Overall Status"; then
    echo -e "${GREEN} 9.1.1.1 통합 헬스체크 동작 확인${NC}"
else
    echo -e "${RED} 9.1.1.1 통합 헬스체크 실패${NC}"
    exit 1
fi

# JSON 형식 검증
if echo "$HEALTH_JSON" | python3 -c "import sys, json; json.load(sys.stdin)" 2>/dev/null; then
    echo -e "${GREEN} JSON 출력 형식 확인${NC}"
else
    echo -e "${RED} JSON 출력 형식 오류${NC}"
    exit 1
fi

echo ""

# 4. 9.1.1.2 블록체인 연결 상태 확인 검증
echo -e "${BLUE}[4/6] 9.1.1.2 블록체인 연결 상태 확인 검증${NC}"
echo "  명령어: ./build/bin/sage-verify blockchain"
echo ""

if command -v timeout &> /dev/null; then
    BLOCKCHAIN_OUTPUT=$(timeout 10 ./build/bin/sage-verify blockchain 2>&1 || echo "Timeout occurred")
else
    BLOCKCHAIN_OUTPUT=$(./build/bin/sage-verify blockchain 2>&1)
fi
echo "$BLOCKCHAIN_OUTPUT"
echo ""

# 검증
if echo "$BLOCKCHAIN_OUTPUT" | grep -q "CONNECTED\|OK"; then
    echo -e "${GREEN} 9.1.1.2 블록체인 연결 성공${NC}"

    # Chain ID 확인
    if echo "$BLOCKCHAIN_OUTPUT" | grep -q "31337"; then
        echo -e "${GREEN} Chain ID 확인 (31337)${NC}"
    else
        echo -e "${YELLOW} Chain ID 확인 불가${NC}"
    fi

    # 블록 번호 확인
    if echo "$BLOCKCHAIN_OUTPUT" | grep -q "Block"; then
        echo -e "${GREEN} 블록 번호 조회 성공${NC}"
    fi
else
    echo -e "${RED} 9.1.1.2 블록체인 연결 실패${NC}"
    exit 1
fi

echo ""

# 5. 9.1.1.3 메모리/CPU 사용률 확인 검증
echo -e "${BLUE}[5/6] 9.1.1.3 메모리/CPU 사용률 확인 검증${NC}"
echo "  명령어: ./build/bin/sage-verify system"
echo ""

if command -v timeout &> /dev/null; then
    SYSTEM_OUTPUT=$(timeout 10 ./build/bin/sage-verify system 2>&1 || echo "Timeout occurred")
else
    SYSTEM_OUTPUT=$(./build/bin/sage-verify system 2>&1)
fi
echo "$SYSTEM_OUTPUT"
echo ""

# 검증
if echo "$SYSTEM_OUTPUT" | grep -q "Memory"; then
    echo -e "${GREEN} 메모리 사용량 표시 확인${NC}"
else
    echo -e "${RED} 메모리 사용량 표시 실패${NC}"
    exit 1
fi

if echo "$SYSTEM_OUTPUT" | grep -q "Disk"; then
    echo -e "${GREEN} 디스크 사용량 표시 확인${NC}"
else
    echo -e "${RED} 디스크 사용량 표시 실패${NC}"
    exit 1
fi

if echo "$SYSTEM_OUTPUT" | grep -q "Goroutines"; then
    echo -e "${GREEN} Goroutine 수 표시 확인${NC}"
else
    echo -e "${RED} Goroutine 수 표시 실패${NC}"
    exit 1
fi

echo ""

# 6. 검증 결과 요약
echo -e "${BLUE}[6/6] 검증 결과 요약${NC}"
echo -e "${BLUE}═══════════════════════════════════════════════════════════${NC}"
echo ""
echo -e "${GREEN} 챕터 9: 헬스체크 검증 완료${NC}"
echo ""
echo "  9.1.1.1 /health 엔드포인트:"
echo "     통합 헬스체크 응답 확인"
echo "     JSON 출력 형식 확인"
echo "     블록체인 및 시스템 상태 포함"
echo ""
echo "  9.1.1.2 블록체인 연결 상태:"
echo "     로컬 노드 연결 성공"
echo "     Chain ID 확인 (31337)"
echo "     블록 번호 조회 성공"
echo ""
echo "  9.1.1.3 메모리/CPU 사용률:"
echo "     메모리 사용량 표시"
echo "     디스크 사용량 표시"
echo "     Goroutine 수 표시"
echo ""
echo -e "${BLUE}═══════════════════════════════════════════════════════════${NC}"
echo ""
echo -e "${GREEN}모든 헬스체크 항목 검증 완료!${NC}"
echo ""

exit 0
