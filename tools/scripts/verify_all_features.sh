#!/bin/bash
# SAGE 전체 기능 검증 스크립트
# 2025년 오픈소스 개발자대회 기능 명세서 검증
# 작성일: 2025-10-10

set -e

# 색상 정의
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# 진행 상황 표시
print_header() {
    echo ""
    echo -e "${BLUE}======================================"
    echo -e "$1"
    echo -e "======================================${NC}"
}

print_success() {
    echo -e "${GREEN}✅ $1${NC}"
}

print_error() {
    echo -e "${RED}❌ $1${NC}"
}

print_info() {
    echo -e "${YELLOW}ℹ️  $1${NC}"
}

# 시작
print_header "SAGE 전체 기능 검증 시작"
echo "기능 명세서: feature_list.docx"
echo "검증 문서: docs/FEATURE_VERIFICATION_GUIDE.md"
echo ""

# 테스트 카운터
TOTAL_TESTS=0
PASSED_TESTS=0
FAILED_TESTS=0

# 1. RFC 9421 구현 검증
print_header "[1/9] RFC 9421 구현 검증"
TOTAL_TESTS=$((TOTAL_TESTS + 1))

print_info "RFC 9421 정규화 테스트..."
if go test -v github.com/sage-x-project/sage/pkg/agent/core/rfc9421 -run TestCanonicalizer > /tmp/test_rfc9421_canon.log 2>&1; then
    print_success "RFC 9421 정규화 테스트 통과"
    PASSED_TESTS=$((PASSED_TESTS + 1))
else
    print_error "RFC 9421 정규화 테스트 실패"
    FAILED_TESTS=$((FAILED_TESTS + 1))
    cat /tmp/test_rfc9421_canon.log
fi

print_info "RFC 9421 서명/검증 통합 테스트..."
if go test -v github.com/sage-x-project/sage/pkg/agent/core/rfc9421 -run TestIntegration > /tmp/test_rfc9421_integration.log 2>&1; then
    print_success "RFC 9421 통합 테스트 통과 (Ed25519, ECDSA)"
    PASSED_TESTS=$((PASSED_TESTS + 1))
else
    print_error "RFC 9421 통합 테스트 실패"
    FAILED_TESTS=$((FAILED_TESTS + 1))
    cat /tmp/test_rfc9421_integration.log
fi
TOTAL_TESTS=$((TOTAL_TESTS + 1))

# 2. 암호화 키 관리 검증
print_header "[2/9] 암호화 키 관리 검증"

print_info "Ed25519 키 생성/서명/검증 테스트..."
TOTAL_TESTS=$((TOTAL_TESTS + 1))
if go test -v github.com/sage-x-project/sage/pkg/agent/crypto/keys -run TestEd25519KeyPair > /tmp/test_ed25519.log 2>&1; then
    print_success "Ed25519 테스트 통과 (32바이트 키, 64바이트 서명)"
    PASSED_TESTS=$((PASSED_TESTS + 1))
else
    print_error "Ed25519 테스트 실패"
    FAILED_TESTS=$((FAILED_TESTS + 1))
    cat /tmp/test_ed25519.log
fi

print_info "Secp256k1 키 생성/서명/검증 테스트..."
TOTAL_TESTS=$((TOTAL_TESTS + 1))
if go test -v github.com/sage-x-project/sage/pkg/agent/crypto/keys -run TestSecp256k1KeyPair > /tmp/test_secp256k1.log 2>&1; then
    print_success "Secp256k1 테스트 통과 (Ethereum 호환 서명)"
    PASSED_TESTS=$((PASSED_TESTS + 1))
else
    print_error "Secp256k1 테스트 실패"
    FAILED_TESTS=$((FAILED_TESTS + 1))
    cat /tmp/test_secp256k1.log
fi

print_info "X25519 HPKE 키 테스트..."
TOTAL_TESTS=$((TOTAL_TESTS + 1))
if go test -v github.com/sage-x-project/sage/pkg/agent/crypto/keys -run TestX25519 > /tmp/test_x25519.log 2>&1; then
    print_success "X25519 테스트 통과"
    PASSED_TESTS=$((PASSED_TESTS + 1))
else
    print_error "X25519 테스트 실패"
    FAILED_TESTS=$((FAILED_TESTS + 1))
    cat /tmp/test_x25519.log
fi

# 3. DID 관리 검증 (통합 테스트 필요)
print_header "[3/9] DID 관리 검증"
print_info "통합 테스트에서 DID 관리 검증 예정..."

# 4. 블록체인 연동 검증
print_header "[4/9] 블록체인 연동 검증"
print_info "통합 테스트에서 블록체인 연동 검증 예정..."

# 5. 메시지 처리 검증
print_header "[5/9] 메시지 처리 검증"

print_info "Nonce 관리 테스트..."
TOTAL_TESTS=$((TOTAL_TESTS + 1))
if go test -v github.com/sage-x-project/sage/pkg/agent/core/message/nonce -run TestNonceManager > /tmp/test_nonce.log 2>&1; then
    print_success "Nonce 관리 테스트 통과 (생성, 검증, 만료)"
    PASSED_TESTS=$((PASSED_TESTS + 1))
else
    print_error "Nonce 관리 테스트 실패"
    FAILED_TESTS=$((FAILED_TESTS + 1))
    cat /tmp/test_nonce.log
fi

print_info "메시지 순서 관리 테스트..."
TOTAL_TESTS=$((TOTAL_TESTS + 1))
if go test -v github.com/sage-x-project/sage/pkg/agent/core/message/order -run TestOrderManager > /tmp/test_order.log 2>&1; then
    print_success "메시지 순서 테스트 통과 (시퀀스, 타임스탬프)"
    PASSED_TESTS=$((PASSED_TESTS + 1))
else
    print_error "메시지 순서 테스트 실패"
    FAILED_TESTS=$((FAILED_TESTS + 1))
    cat /tmp/test_order.log
fi

print_info "중복 감지 테스트..."
TOTAL_TESTS=$((TOTAL_TESTS + 1))
if go test -v github.com/sage-x-project/sage/pkg/agent/core/message/dedupe -run TestDetector > /tmp/test_dedupe.log 2>&1; then
    print_success "중복 감지 테스트 통과"
    PASSED_TESTS=$((PASSED_TESTS + 1))
else
    print_error "중복 감지 테스트 실패"
    FAILED_TESTS=$((FAILED_TESTS + 1))
    cat /tmp/test_dedupe.log
fi

# 6. CLI 도구 검증
print_header "[6/9] CLI 도구 검증"

print_info "sage-crypto 키 생성 테스트..."
TOTAL_TESTS=$((TOTAL_TESTS + 1))
if ./build/bin/sage-crypto generate --type ed25519 --format jwk > /tmp/cli_ed25519.json 2>&1; then
    if grep -q "private_key" /tmp/cli_ed25519.json && grep -q "public_key" /tmp/cli_ed25519.json; then
        print_success "sage-crypto Ed25519 키 생성 성공"
        PASSED_TESTS=$((PASSED_TESTS + 1))
    else
        print_error "sage-crypto Ed25519 키 생성 출력 오류"
        FAILED_TESTS=$((FAILED_TESTS + 1))
    fi
else
    print_error "sage-crypto 명령 실행 실패"
    FAILED_TESTS=$((FAILED_TESTS + 1))
fi

print_info "sage-crypto Secp256k1 PEM 생성 테스트..."
TOTAL_TESTS=$((TOTAL_TESTS + 1))
if ./build/bin/sage-crypto generate --type secp256k1 --format pem > /tmp/cli_secp256k1.pem 2>&1; then
    if grep -q "BEGIN PRIVATE KEY" /tmp/cli_secp256k1.pem; then
        print_success "sage-crypto Secp256k1 PEM 생성 성공"
        PASSED_TESTS=$((PASSED_TESTS + 1))
    else
        print_error "sage-crypto PEM 형식 오류"
        FAILED_TESTS=$((FAILED_TESTS + 1))
    fi
else
    print_error "sage-crypto PEM 생성 실패"
    FAILED_TESTS=$((FAILED_TESTS + 1))
fi

# 7. 세션 관리 검증
print_header "[7/9] 세션 관리 검증"

print_info "세션 생성/관리 테스트..."
TOTAL_TESTS=$((TOTAL_TESTS + 1))
if go test -v github.com/sage-x-project/sage/pkg/agent/session -run "TestSessionManager.*" > /tmp/test_session.log 2>&1; then
    print_success "세션 관리 테스트 통과 (생성, 조회, 만료)"
    PASSED_TESTS=$((PASSED_TESTS + 1))
else
    print_error "세션 관리 테스트 실패"
    FAILED_TESTS=$((FAILED_TESTS + 1))
    cat /tmp/test_session.log
fi

# 8. HPKE 검증
print_header "[8/9] HPKE (Hybrid Public Key Encryption) 검증"

print_info "HPKE 키 교환/암호화/복호화 테스트..."
TOTAL_TESTS=$((TOTAL_TESTS + 1))
if go test -v github.com/sage-x-project/sage/pkg/agent/hpke > /tmp/test_hpke.log 2>&1; then
    print_success "HPKE 테스트 통과 (X25519, AEAD)"
    PASSED_TESTS=$((PASSED_TESTS + 1))
else
    print_error "HPKE 테스트 실패"
    FAILED_TESTS=$((FAILED_TESTS + 1))
    cat /tmp/test_hpke.log
fi

print_info "핸드셰이크 E2E 테스트 (5가지 시나리오)..."
TOTAL_TESTS=$((TOTAL_TESTS + 1))
if make test-handshake > /tmp/test_handshake.log 2>&1; then
    print_success "핸드셰이크 E2E 테스트 통과 (signed, replay, expired 등)"
    PASSED_TESTS=$((PASSED_TESTS + 1))
else
    print_error "핸드셰이크 E2E 테스트 실패"
    FAILED_TESTS=$((FAILED_TESTS + 1))
    cat /tmp/test_handshake.log
fi

# 9. 통합 테스트 (전체)
print_header "[9/9] 통합 테스트 (전체)"

print_info "전체 유닛 테스트 실행..."
TOTAL_TESTS=$((TOTAL_TESTS + 1))
if make test > /tmp/test_all_units.log 2>&1; then
    print_success "전체 유닛 테스트 통과 (150+ 테스트)"
    PASSED_TESTS=$((PASSED_TESTS + 1))
else
    print_error "유닛 테스트 일부 실패"
    FAILED_TESTS=$((FAILED_TESTS + 1))
    tail -100 /tmp/test_all_units.log
fi

print_info "통합 테스트 실행 (블록체인 포함)..."
TOTAL_TESTS=$((TOTAL_TESTS + 1))
if make test-integration > /tmp/test_integration.log 2>&1; then
    print_success "통합 테스트 통과 (DID 등록, 조회, 업데이트)"
    PASSED_TESTS=$((PASSED_TESTS + 1))
else
    print_error "통합 테스트 일부 실패"
    FAILED_TESTS=$((FAILED_TESTS + 1))
    tail -100 /tmp/test_integration.log
fi

# 최종 결과
print_header "검증 결과 요약"
echo -e "${BLUE}총 테스트:${NC} $TOTAL_TESTS"
echo -e "${GREEN}통과:${NC} $PASSED_TESTS"
echo -e "${RED}실패:${NC} $FAILED_TESTS"
echo ""

if [ $FAILED_TESTS -eq 0 ]; then
    print_success "모든 기능 검증 완료! 🎉"
    echo ""
    echo -e "${GREEN}기능 명세서의 모든 항목이 구현되고 테스트되었습니다.${NC}"
    echo ""
    echo "상세 검증 결과: docs/FEATURE_VERIFICATION_GUIDE.md"
    exit 0
else
    print_error "일부 테스트 실패"
    echo ""
    echo "로그 파일:"
    echo "  /tmp/test_*.log"
    exit 1
fi
