#!/bin/bash

# =========================================
# SAGE CLI 워크플로우 테스트 스크립트
# =========================================
#
# 이 스크립트는 docs/test/GO_TEST_COMMANDS.md의 Chapter 8 명령어들을
# 순차적으로 실행하여 SAGE CLI 도구를 검증합니다.
#
# 실행 방법:
#   chmod +x /tmp/sage-cli-workflow.sh
#   /tmp/sage-cli-workflow.sh
#

set -e  # 오류 발생 시 스크립트 중단

# 색상 출력 설정
GREEN='\033[0;32m'
BLUE='\033[0;34m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
NC='\033[0m' # No Color

# 로그 함수
log_section() {
    echo -e "\n${BLUE}=========================================${NC}"
    echo -e "${BLUE}$1${NC}"
    echo -e "${BLUE}=========================================${NC}\n"
}

log_step() {
    echo -e "${YELLOW}📌 $1${NC}"
}

log_success() {
    echo -e "${GREEN}✅ $1${NC}"
}

log_error() {
    echo -e "${RED}❌ $1${NC}"
}

log_info() {
    echo -e "${NC}ℹ️  $1${NC}"
}

# 환경 변수 설정
SAGE_CRYPTO="./build/bin/sage-crypto"
SAGE_DID="./build/bin/sage-did"
TMP_DIR="/tmp/sage-test"
HARDHAT_RPC="http://localhost:8545"
HARDHAT_PRIVATE_KEY="0xac0974bec39a17e36ba4a6b4d238ff944bacb478cbed5efcae784d7bf4f2ff80"
HARDHAT_ADDRESS="0xf39Fd6e51aad88F6F4ce6aB8827279cffFb92266"

# 임시 디렉토리 생성
mkdir -p $TMP_DIR

# 바이너리 존재 확인
if [ ! -f "$SAGE_CRYPTO" ]; then
    log_error "sage-crypto binary not found at $SAGE_CRYPTO"
    log_info "Please run: make build"
    exit 1
fi

if [ ! -f "$SAGE_DID" ]; then
    log_error "sage-did binary not found at $SAGE_DID"
    log_info "Please run: make build"
    exit 1
fi

# =========================================
# 8.1 sage-crypto CLI 검증
# =========================================

log_section "8.1 sage-crypto CLI 검증"

# 8.1.1 키 생성 CLI
log_step "8.1.1 키 생성 테스트"

# Ed25519 키 생성
log_info "Ed25519 키 생성 중..."
$SAGE_CRYPTO generate --type ed25519 --format jwk --output $TMP_DIR/test-ed25519.jwk

if [ -f "$TMP_DIR/test-ed25519.jwk" ]; then
    log_success "Ed25519 키 생성 성공"

    # 키 타입 확인
    KEY_TYPE=$(cat $TMP_DIR/test-ed25519.jwk | jq -r '.private_key.kty')
    CURVE=$(cat $TMP_DIR/test-ed25519.jwk | jq -r '.private_key.crv')
    log_info "Key Type: $KEY_TYPE"
    log_info "Curve: $CURVE"

    if [ "$KEY_TYPE" = "OKP" ] && [ "$CURVE" = "Ed25519" ]; then
        log_success "Ed25519 키 검증 성공"
    else
        log_error "Ed25519 키 검증 실패"
        exit 1
    fi
else
    log_error "Ed25519 키 생성 실패"
    exit 1
fi

echo ""

# Secp256k1 키 생성
log_info "Secp256k1 키 생성 중..."
$SAGE_CRYPTO generate --type secp256k1 --format jwk --output $TMP_DIR/test-secp256k1.jwk

if [ -f "$TMP_DIR/test-secp256k1.jwk" ]; then
    log_success "Secp256k1 키 생성 성공"

    # 키 타입 확인
    KEY_TYPE=$(cat $TMP_DIR/test-secp256k1.jwk | jq -r '.key_type')
    log_info "Key Type: $KEY_TYPE"

    if [ "$KEY_TYPE" = "Secp256k1" ]; then
        log_success "Secp256k1 키 검증 성공"
    else
        log_error "Secp256k1 키 검증 실패"
        exit 1
    fi
else
    log_error "Secp256k1 키 생성 실패"
    exit 1
fi

echo ""

# 8.1.2 서명 CLI
log_step "8.1.2 서명 생성 및 검증 테스트"

# 테스트 메시지 작성
echo "test message" > $TMP_DIR/msg.txt
log_info "테스트 메시지 작성: $(cat $TMP_DIR/msg.txt)"

# 서명 생성
log_info "Ed25519 키로 서명 생성 중..."
$SAGE_CRYPTO sign --key $TMP_DIR/test-ed25519.jwk --message-file $TMP_DIR/msg.txt --output $TMP_DIR/sig.bin

if [ -f "$TMP_DIR/sig.bin" ]; then
    log_success "서명 생성 성공"
    ls -lh $TMP_DIR/sig.bin
else
    log_error "서명 생성 실패"
    exit 1
fi

echo ""

# 8.1.3 검증 CLI
log_step "8.1.3 서명 검증 테스트"

log_info "서명 검증 중..."
$SAGE_CRYPTO verify --key $TMP_DIR/test-ed25519.jwk --message-file $TMP_DIR/msg.txt --signature-file $TMP_DIR/sig.bin

log_success "서명 검증 성공"

echo ""

# 8.1.4 주소 생성 CLI
log_step "8.1.4 Ethereum 주소 생성 테스트"

log_info "Secp256k1 키로 Ethereum 주소 생성 중..."
ETH_ADDRESS=$($SAGE_CRYPTO address generate --key $TMP_DIR/test-secp256k1.jwk --chain ethereum | grep "ethereum" | awk '{print $2}')

if [ -n "$ETH_ADDRESS" ]; then
    log_success "Ethereum 주소 생성 성공: $ETH_ADDRESS"
else
    log_error "Ethereum 주소 생성 실패"
    exit 1
fi

echo ""

# =========================================
# 8.2 sage-did CLI 검증
# =========================================

log_section "8.2 sage-did CLI 검증"

log_info "이 섹션은 로컬 블록체인 노드(Hardhat)가 필요합니다."
log_info "다음 명령어로 Hardhat 노드를 시작하세요:"
echo ""
echo "  cd contracts/ethereum"
echo "  npx hardhat node"
echo ""
log_info "그리고 다른 터미널에서 컨트랙트를 배포하세요:"
echo ""
echo "  npx hardhat run scripts/deploy-v4-local.js --network localhost"
echo ""

# 사용자에게 계속 진행할지 물어보기
read -p "Hardhat 노드와 컨트랙트 배포가 완료되었습니까? (y/N) " -n 1 -r
echo
if [[ ! $REPLY =~ ^[Yy]$ ]]; then
    log_info "스크립트를 중단합니다. Hardhat 노드와 컨트랙트를 먼저 준비하세요."
    exit 0
fi

# 컨트랙트 주소 입력 받기
read -p "배포된 DID Registry 컨트랙트 주소를 입력하세요: " CONTRACT_ADDRESS

if [ -z "$CONTRACT_ADDRESS" ]; then
    log_error "컨트랙트 주소가 입력되지 않았습니다."
    exit 1
fi

log_info "컨트랙트 주소: $CONTRACT_ADDRESS"

echo ""

# 8.2.1 DID 키 생성
log_step "8.2.1 DID 등록용 키 생성"

# Secp256k1 키 생성 (Ethereum 호환)
log_info "DID 등록용 Secp256k1 키 생성 중..."
$SAGE_CRYPTO generate --type secp256k1 --format jwk --output $TMP_DIR/eth-key.jwk

if [ -f "$TMP_DIR/eth-key.jwk" ]; then
    log_success "Secp256k1 키 생성 성공"

    # 키 정보 확인
    KEY_ID=$(cat $TMP_DIR/eth-key.jwk | jq -r '.key_id')
    KEY_TYPE=$(cat $TMP_DIR/eth-key.jwk | jq -r '.key_type')
    log_info "Key ID: $KEY_ID"
    log_info "Key Type: $KEY_TYPE"

    # Ethereum 주소 생성
    log_info "Ethereum 주소 생성 중..."
    OWNER_ADDRESS=$($SAGE_CRYPTO address generate --key $TMP_DIR/eth-key.jwk --chain ethereum | grep "ethereum" | awk '{print $2}')
    log_success "Owner Address: $OWNER_ADDRESS"

    # JWK 구조 확인
    log_info "JWK 파일 구조:"
    cat $TMP_DIR/eth-key.jwk | jq '{key_id, key_type, private_key: {kty: .private_key.kty, crv: .private_key.crv}}'
else
    log_error "Secp256k1 키 생성 실패"
    exit 1
fi

echo ""

# Ed25519 키 생성 (추가 키용)
log_info "추가 키용 Ed25519 키 생성 중..."
$SAGE_CRYPTO generate --type ed25519 --format jwk --output $TMP_DIR/did-key.jwk

if [ -f "$TMP_DIR/did-key.jwk" ]; then
    log_success "Ed25519 키 생성 성공"
else
    log_error "Ed25519 키 생성 실패"
    exit 1
fi

echo ""

# 8.2.3 DID 등록
log_step "8.2.3 DID 등록 (단일 키)"

log_info "Secp256k1 키로 에이전트 등록 중..."
log_info "트랜잭션 가스비 지불 계정: $HARDHAT_ADDRESS"

REGISTER_OUTPUT=$($SAGE_DID register \
  --chain ethereum \
  --name "SAGE Test Agent" \
  --endpoint "https://agent.example.com" \
  --key $TMP_DIR/eth-key.jwk \
  --rpc $HARDHAT_RPC \
  --contract $CONTRACT_ADDRESS \
  --private-key $HARDHAT_PRIVATE_KEY 2>&1)

echo "$REGISTER_OUTPUT"

# DID 추출
REGISTERED_DID=$(echo "$REGISTER_OUTPUT" | grep -oP 'DID: \K[^\s]+' || echo "")

if [ -n "$REGISTERED_DID" ]; then
    log_success "DID 등록 성공!"
    log_success "등록된 DID: $REGISTERED_DID"
else
    log_error "DID 등록 실패 - DID를 추출할 수 없습니다."
    log_info "등록 출력:"
    echo "$REGISTER_OUTPUT"
    exit 1
fi

echo ""

# 8.2.2 DID 조회
log_step "8.2.2 등록한 DID 조회 (JSON 형식)"

log_info "DID 조회 중: $REGISTERED_DID"

$SAGE_DID resolve $REGISTERED_DID \
  --rpc $HARDHAT_RPC \
  --contract $CONTRACT_ADDRESS

log_success "DID 조회 성공"

echo ""

# 텍스트 형식으로 조회
log_info "DID 조회 (텍스트 형식):"

$SAGE_DID resolve $REGISTERED_DID \
  --rpc $HARDHAT_RPC \
  --contract $CONTRACT_ADDRESS \
  --format text

echo ""

# 파일로 저장
log_info "DID 메타데이터를 파일로 저장 중..."

$SAGE_DID resolve $REGISTERED_DID \
  --rpc $HARDHAT_RPC \
  --contract $CONTRACT_ADDRESS \
  --output $TMP_DIR/agent-metadata.json

if [ -f "$TMP_DIR/agent-metadata.json" ]; then
    log_success "메타데이터 저장 성공: $TMP_DIR/agent-metadata.json"
    cat $TMP_DIR/agent-metadata.json | jq '.'
else
    log_error "메타데이터 저장 실패"
fi

echo ""

# 8.2.4 DID 목록 조회
log_step "8.2.4 소유자 주소로 DID 목록 조회"

log_info "소유자 주소: $OWNER_ADDRESS"

# 테이블 형식
log_info "목록 조회 (테이블 형식):"

$SAGE_DID list \
  --chain ethereum \
  --owner $OWNER_ADDRESS \
  --rpc $HARDHAT_RPC \
  --contract $CONTRACT_ADDRESS \
  --format table

echo ""

# JSON 형식
log_info "목록 조회 (JSON 형식):"

$SAGE_DID list \
  --chain ethereum \
  --owner $OWNER_ADDRESS \
  --rpc $HARDHAT_RPC \
  --contract $CONTRACT_ADDRESS \
  --format json

log_success "DID 목록 조회 성공"

echo ""

# 8.2.5 키 관리 (선택적)
log_step "8.2.5 키 관리 (선택 사항 - 스킵)"

log_info "키 관리 명령어 예시:"
echo ""
echo "  # 에이전트의 모든 키 조회"
echo "  $SAGE_DID key list \\"
echo "    --chain ethereum \\"
echo "    $REGISTERED_DID \\"
echo "    --rpc $HARDHAT_RPC \\"
echo "    --contract $CONTRACT_ADDRESS"
echo ""
echo "  # 에이전트에 추가 키 등록"
echo "  $SAGE_DID key add \\"
echo "    --chain ethereum \\"
echo "    $REGISTERED_DID \\"
echo "    --key $TMP_DIR/did-key.jwk \\"
echo "    --rpc $HARDHAT_RPC \\"
echo "    --contract $CONTRACT_ADDRESS"
echo ""

# =========================================
# 최종 요약
# =========================================

log_section "테스트 완료!"

echo ""
log_success "모든 CLI 워크플로우 테스트가 성공적으로 완료되었습니다!"
echo ""
log_info "생성된 파일들:"
echo "  - Ed25519 키: $TMP_DIR/test-ed25519.jwk"
echo "  - Secp256k1 키: $TMP_DIR/test-secp256k1.jwk"
echo "  - DID 등록용 키: $TMP_DIR/eth-key.jwk"
echo "  - 추가 키: $TMP_DIR/did-key.jwk"
echo "  - 서명: $TMP_DIR/sig.bin"
echo "  - 메타데이터: $TMP_DIR/agent-metadata.json"
echo ""
log_info "등록된 정보:"
echo "  - DID: $REGISTERED_DID"
echo "  - Owner Address: $OWNER_ADDRESS"
echo "  - Contract: $CONTRACT_ADDRESS"
echo ""
log_info "파일 정리를 원하시면 다음 명령어를 실행하세요:"
echo "  rm -rf $TMP_DIR"
echo ""
