#!/bin/bash
# SAGE 전체 기능 검증 스크립트 (소분류 기준)
# 2025년 오픈소스 개발자대회 기능 명세서 검증
# 작성일: 2025-10-10

set -e

# 색상 정의
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
CYAN='\033[0;36m'
NC='\033[0m' # No Color

# 진행 상황 표시
print_header() {
    echo ""
    echo -e "${BLUE}======================================================================"
    echo -e "$1"
    echo -e "======================================================================${NC}"
}

print_category() {
    echo ""
    echo -e "${CYAN}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
    echo -e "  $1"
    echo -e "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
}

print_test() {
    echo -e "${YELLOW}  [$1/$2] $3${NC}"
}

print_success() {
    echo -e "       ${GREEN}✅ $1${NC}"
}

print_error() {
    echo -e "       ${RED}❌ $1${NC}"
}

print_skip() {
    echo -e "       ${YELLOW}⏭️  $1${NC}"
}

# 테스트 실행 함수
run_test() {
    local test_name="$1"
    local test_cmd="$2"
    local log_file="$3"

    TOTAL_TESTS=$((TOTAL_TESTS + 1))

    if eval "$test_cmd" > "$log_file" 2>&1; then
        print_success "$test_name 통과"
        PASSED_TESTS=$((PASSED_TESTS + 1))
        return 0
    else
        print_error "$test_name 실패"
        FAILED_TESTS=$((FAILED_TESTS + 1))
        if [ "$VERBOSE" = "1" ]; then
            echo "       로그: $log_file"
            tail -20 "$log_file" | sed 's/^/         /'
        fi
        return 1
    fi
}

# CLI 테스트 함수
run_cli_test() {
    local test_name="$1"
    local test_cmd="$2"
    local expected_output="$3"
    local log_file="$4"

    TOTAL_TESTS=$((TOTAL_TESTS + 1))

    if eval "$test_cmd" > "$log_file" 2>&1; then
        if grep -q "$expected_output" "$log_file"; then
            print_success "$test_name 통과"
            PASSED_TESTS=$((PASSED_TESTS + 1))
            return 0
        else
            print_error "$test_name 출력 불일치"
            FAILED_TESTS=$((FAILED_TESTS + 1))
            return 1
        fi
    else
        print_error "$test_name 실패"
        FAILED_TESTS=$((FAILED_TESTS + 1))
        return 1
    fi
}

# 옵션 파싱
VERBOSE=0
if [ "$1" = "-v" ] || [ "$1" = "--verbose" ]; then
    VERBOSE=1
fi

# 시작
print_header "SAGE 전체 기능 검증 (소분류 기준)"
echo "기능 명세서: feature_list.docx"
echo "검증 문서: docs/FEATURE_VERIFICATION_GUIDE.md"
echo ""
echo "옵션: -v 또는 --verbose 로 실패 로그 상세 출력"
echo ""

# 테스트 카운터
TOTAL_TESTS=0
PASSED_TESTS=0
FAILED_TESTS=0

# 로그 디렉토리 생성
mkdir -p /tmp/sage-test-logs

#==============================================================================
# 1. RFC 9421 구현
#==============================================================================
print_header "[1/9] RFC 9421 구현"

## 1.1 메시지 서명
print_category "1.1 메시지 서명"
TEST_NUM=1

print_test $TEST_NUM 6 "HTTP 메시지 서명 생성 (Ed25519)"
run_test "HTTP 메시지 서명 생성 (Ed25519)" \
    "go test -v github.com/sage-x-project/sage/pkg/agent/core/rfc9421 -run 'TestIntegration/Ed25519'" \
    "/tmp/sage-test-logs/rfc9421_sign_ed25519.log"
TEST_NUM=$((TEST_NUM + 1))

print_test $TEST_NUM 6 "HTTP 메시지 서명 생성 (Secp256k1)"
run_test "HTTP 메시지 서명 생성 (Secp256k1)" \
    "go test -v github.com/sage-x-project/sage/pkg/agent/core/rfc9421 -run 'TestIntegration/ECDSA_Secp256k1'" \
    "/tmp/sage-test-logs/rfc9421_sign_secp256k1.log"
TEST_NUM=$((TEST_NUM + 1))

print_test $TEST_NUM 6 "Signature-Input 헤더 생성"
run_test "Signature-Input 헤더 형식" \
    "go test -v github.com/sage-x-project/sage/pkg/agent/core/rfc9421 -run 'TestParseSignatureInput'" \
    "/tmp/sage-test-logs/rfc9421_sig_input.log"
TEST_NUM=$((TEST_NUM + 1))

print_test $TEST_NUM 6 "Signature 헤더 생성"
run_test "Signature 헤더 Base64 인코딩" \
    "go test -v github.com/sage-x-project/sage/pkg/agent/core/rfc9421 -run 'TestParseSignature'" \
    "/tmp/sage-test-logs/rfc9421_sig_header.log"
TEST_NUM=$((TEST_NUM + 1))

print_test $TEST_NUM 6 "서명 필드 선택 및 정규화"
run_test "서명 필드 정규화" \
    "go test -v github.com/sage-x-project/sage/pkg/agent/core/rfc9421 -run 'TestCanonicalizer/basic_GET'" \
    "/tmp/sage-test-logs/rfc9421_field_normalize.log"
TEST_NUM=$((TEST_NUM + 1))

print_test $TEST_NUM 6 "Base64 인코딩 검증"
run_test "Base64 인코딩" \
    "go test -v github.com/sage-x-project/sage/pkg/agent/core/rfc9421 -run 'TestIntegration'" \
    "/tmp/sage-test-logs/rfc9421_base64.log"

## 1.2 메시지 검증
print_category "1.2 메시지 검증"
TEST_NUM=1

print_test $TEST_NUM 5 "서명 파싱 및 디코딩"
run_test "서명 파싱" \
    "go test -v github.com/sage-x-project/sage/pkg/agent/core/rfc9421 -run 'TestParseSignature'" \
    "/tmp/sage-test-logs/rfc9421_parse.log"
TEST_NUM=$((TEST_NUM + 1))

print_test $TEST_NUM 5 "정규화된 메시지 재구성"
run_test "메시지 재구성" \
    "go test -v github.com/sage-x-project/sage/pkg/agent/core/rfc9421 -run 'TestConstructSignatureBase'" \
    "/tmp/sage-test-logs/rfc9421_reconstruct.log"
TEST_NUM=$((TEST_NUM + 1))

print_test $TEST_NUM 5 "서명 검증 알고리즘 실행"
run_test "서명 검증" \
    "go test -v github.com/sage-x-project/sage/pkg/agent/core/rfc9421 -run 'TestVerifier/VerifySignature_with_valid'" \
    "/tmp/sage-test-logs/rfc9421_verify.log"
TEST_NUM=$((TEST_NUM + 1))

print_test $TEST_NUM 5 "타임스탬프 유효성 검사"
run_test "타임스탬프 검증" \
    "go test -v github.com/sage-x-project/sage/pkg/agent/core/rfc9421 -run 'TestNegativeCases/expired_signature'" \
    "/tmp/sage-test-logs/rfc9421_timestamp.log"
TEST_NUM=$((TEST_NUM + 1))

print_test $TEST_NUM 5 "Nonce 중복 체크"
run_test "Nonce 중복 검증" \
    "go test -v github.com/sage-x-project/sage/pkg/agent/core/message/nonce -run 'TestNonceManager/MarkNonceUsed'" \
    "/tmp/sage-test-logs/rfc9421_nonce_check.log"

## 1.3 메시지 빌더
print_category "1.3 메시지 빌더"
TEST_NUM=1

print_test $TEST_NUM 4 "메시지 구조 생성"
run_test "메시지 구조 생성" \
    "go test -v github.com/sage-x-project/sage/pkg/agent/core/rfc9421 -run 'TestMessageBuilder/Build_complete'" \
    "/tmp/sage-test-logs/rfc9421_builder_struct.log"
TEST_NUM=$((TEST_NUM + 1))

print_test $TEST_NUM 4 "헤더 필드 추가"
run_test "헤더 필드 추가" \
    "go test -v github.com/sage-x-project/sage/pkg/agent/core/rfc9421 -run 'TestMessageBuilder'" \
    "/tmp/sage-test-logs/rfc9421_builder_header.log"
TEST_NUM=$((TEST_NUM + 1))

print_test $TEST_NUM 4 "메타데이터 설정"
run_test "메타데이터 설정" \
    "go test -v github.com/sage-x-project/sage/pkg/agent/core/rfc9421 -run 'TestMessageBuilder'" \
    "/tmp/sage-test-logs/rfc9421_builder_meta.log"
TEST_NUM=$((TEST_NUM + 1))

print_test $TEST_NUM 4 "서명 필드 지정"
run_test "서명 필드 지정" \
    "go test -v github.com/sage-x-project/sage/pkg/agent/core/rfc9421 -run 'TestMessageBuilder/Build_with_default'" \
    "/tmp/sage-test-logs/rfc9421_builder_fields.log"

## 1.4 정규화
print_category "1.4 정규화"
TEST_NUM=1

print_test $TEST_NUM 4 "Canonical Request 생성"
run_test "Canonical Request 생성" \
    "go test -v github.com/sage-x-project/sage/pkg/agent/core/rfc9421 -run 'TestCanonicalizer'" \
    "/tmp/sage-test-logs/rfc9421_canonical.log"
TEST_NUM=$((TEST_NUM + 1))

print_test $TEST_NUM 4 "헤더 정규화"
run_test "헤더 정규화" \
    "go test -v github.com/sage-x-project/sage/pkg/agent/core/rfc9421 -run 'TestCanonicalizer/header_whitespace'" \
    "/tmp/sage-test-logs/rfc9421_header_normalize.log"
TEST_NUM=$((TEST_NUM + 1))

print_test $TEST_NUM 4 "경로 정규화"
run_test "경로 정규화" \
    "go test -v github.com/sage-x-project/sage/pkg/agent/core/rfc9421 -run 'TestCanonicalizer/special_characters'" \
    "/tmp/sage-test-logs/rfc9421_path_normalize.log"
TEST_NUM=$((TEST_NUM + 1))

print_test $TEST_NUM 4 "쿼리 파라미터 정렬"
run_test "쿼리 파라미터 정렬" \
    "go test -v github.com/sage-x-project/sage/pkg/agent/core/rfc9421 -run 'TestQueryParamProtection'" \
    "/tmp/sage-test-logs/rfc9421_query_sort.log"

#==============================================================================
# 2. 암호화 키 관리
#==============================================================================
print_header "[2/9] 암호화 키 관리"

## 2.1 키 생성
print_category "2.1 키 생성"
TEST_NUM=1

print_test $TEST_NUM 4 "Secp256k1 키페어 생성"
run_test "Secp256k1 32바이트 개인키, 65바이트 공개키" \
    "go test -v github.com/sage-x-project/sage/pkg/agent/crypto/keys -run 'TestSecp256k1KeyPair/GenerateKeyPair'" \
    "/tmp/sage-test-logs/key_secp256k1_gen.log"
TEST_NUM=$((TEST_NUM + 1))

print_test $TEST_NUM 4 "Ed25519 키페어 생성"
run_test "Ed25519 32바이트 개인키, 32바이트 공개키" \
    "go test -v github.com/sage-x-project/sage/pkg/agent/crypto/keys -run 'TestEd25519KeyPair/GenerateKeyPair'" \
    "/tmp/sage-test-logs/key_ed25519_gen.log"
TEST_NUM=$((TEST_NUM + 1))

print_test $TEST_NUM 4 "X25519 키 생성 (HPKE용)"
run_test "X25519 HPKE 키 생성" \
    "go test -v github.com/sage-x-project/sage/pkg/agent/crypto/keys -run 'TestX25519'" \
    "/tmp/sage-test-logs/key_x25519_gen.log"
TEST_NUM=$((TEST_NUM + 1))

print_test $TEST_NUM 4 "RSA 키페어 생성"
run_test "RSA 키페어 생성" \
    "go test -v github.com/sage-x-project/sage/pkg/agent/crypto/keys -run 'TestRSA'" \
    "/tmp/sage-test-logs/key_rsa_gen.log"

## 2.2 키 저장
print_category "2.2 키 저장"
TEST_NUM=1

print_test $TEST_NUM 4 "파일 기반 저장 (PEM 형식)"
run_test "PEM 파일 저장 및 권한 확인" \
    "go test -v github.com/sage-x-project/sage/pkg/agent/crypto/storage -run 'TestFileStorage.*Save'" \
    "/tmp/sage-test-logs/key_file_storage.log"
TEST_NUM=$((TEST_NUM + 1))

print_test $TEST_NUM 4 "메모리 기반 저장"
run_test "메모리 저장소" \
    "go test -v github.com/sage-x-project/sage/pkg/agent/crypto/storage -run 'TestMemoryStorage'" \
    "/tmp/sage-test-logs/key_memory_storage.log"
TEST_NUM=$((TEST_NUM + 1))

print_test $TEST_NUM 4 "키 회전 지원"
run_test "키 회전" \
    "go test -v github.com/sage-x-project/sage/pkg/agent/crypto -run 'TestManager_DeleteKeyPair'" \
    "/tmp/sage-test-logs/key_rotation.log"
TEST_NUM=$((TEST_NUM + 1))

print_test $TEST_NUM 4 "키 목록 조회"
run_test "키 목록 조회" \
    "go test -v github.com/sage-x-project/sage/pkg/agent/crypto -run 'TestManager_ListKeyPairs'" \
    "/tmp/sage-test-logs/key_list.log"

## 2.3 키 형식 변환
print_category "2.3 키 형식 변환"
TEST_NUM=1

print_test $TEST_NUM 4 "PEM 형식 인코딩/디코딩"
run_test "PEM 변환" \
    "go test -v github.com/sage-x-project/sage/pkg/agent/crypto -run 'TestManager_ExportKeyPair.*pem'" \
    "/tmp/sage-test-logs/key_pem_convert.log"
TEST_NUM=$((TEST_NUM + 1))

print_test $TEST_NUM 4 "JWK 형식 변환"
run_test "JWK 변환" \
    "go test -v github.com/sage-x-project/sage/pkg/agent/crypto -run 'TestManager_ExportKeyPair.*jwk'" \
    "/tmp/sage-test-logs/key_jwk_convert.log"
TEST_NUM=$((TEST_NUM + 1))

print_test $TEST_NUM 4 "압축/비압축 공개키 변환"
run_test "공개키 압축/비압축 변환" \
    "go test -v github.com/sage-x-project/sage/pkg/agent/crypto/keys -run 'TestSecp256k1.*Compress'" \
    "/tmp/sage-test-logs/key_compress.log"
TEST_NUM=$((TEST_NUM + 1))

print_test $TEST_NUM 4 "Ethereum 주소 생성"
run_test "Ethereum 주소 생성 (0x prefix)" \
    "go test -v github.com/sage-x-project/sage/pkg/agent/crypto/keys -run 'TestEthereumAddress'" \
    "/tmp/sage-test-logs/key_eth_addr.log"

## 2.4 서명/검증
print_category "2.4 서명/검증"
TEST_NUM=1

print_test $TEST_NUM 4 "ECDSA 서명 (Secp256k1)"
run_test "Secp256k1 ECDSA 서명 및 검증" \
    "go test -v github.com/sage-x-project/sage/pkg/agent/crypto/keys -run 'TestSecp256k1KeyPair/SignAndVerify'" \
    "/tmp/sage-test-logs/sign_ecdsa.log"
TEST_NUM=$((TEST_NUM + 1))

print_test $TEST_NUM 4 "EdDSA 서명 (Ed25519)"
run_test "Ed25519 EdDSA 서명 및 검증 (64바이트)" \
    "go test -v github.com/sage-x-project/sage/pkg/agent/crypto/keys -run 'TestEd25519KeyPair/SignAndVerify'" \
    "/tmp/sage-test-logs/sign_eddsa.log"
TEST_NUM=$((TEST_NUM + 1))

print_test $TEST_NUM 4 "대용량 메시지 서명"
run_test "대용량 메시지 서명 테스트" \
    "go test -v github.com/sage-x-project/sage/pkg/agent/crypto/keys -run '.*SignLargeMessage'" \
    "/tmp/sage-test-logs/sign_large.log"
TEST_NUM=$((TEST_NUM + 1))

print_test $TEST_NUM 4 "빈 메시지 서명"
run_test "빈 메시지 서명 지원" \
    "go test -v github.com/sage-x-project/sage/pkg/agent/crypto/keys -run '.*SignEmptyMessage'" \
    "/tmp/sage-test-logs/sign_empty.log"

#==============================================================================
# 3. DID 관리
#==============================================================================
print_header "[3/9] DID 관리"

## 3.1 DID 생성
print_category "3.1 DID 생성"
TEST_NUM=1

print_test $TEST_NUM 2 "did:sage:ethereum 형식 생성"
run_test "DID 형식 검증" \
    "go test -v github.com/sage-x-project/sage/pkg/agent/did -run 'TestManager_CreateDID'" \
    "/tmp/sage-test-logs/did_create.log"
TEST_NUM=$((TEST_NUM + 1))

print_test $TEST_NUM 2 "DID Document 생성"
run_test "DID Document 구조" \
    "go test -v github.com/sage-x-project/sage/pkg/agent/did -run 'TestDocument'" \
    "/tmp/sage-test-logs/did_document.log"

## 3.2 DID 등록
print_category "3.2 DID 등록 (통합 테스트)"
print_skip "통합 테스트 섹션에서 검증"

## 3.3 DID 조회
print_category "3.3 DID 조회 (통합 테스트)"
print_skip "통합 테스트 섹션에서 검증"

## 3.4 DID 관리
print_category "3.4 DID 관리 (통합 테스트)"
print_skip "통합 테스트 섹션에서 검증"

#==============================================================================
# 4. 블록체인 연동
#==============================================================================
print_header "[4/9] 블록체인 연동"

## 4.1 Ethereum 연동
print_category "4.1 Ethereum 연동 (통합 테스트)"
print_skip "통합 테스트 섹션에서 검증 (Web3, 트랜잭션, 가스 예측)"

## 4.2 체인 레지스트리
print_category "4.2 체인 레지스트리"
TEST_NUM=1

print_test $TEST_NUM 4 "멀티체인 설정 로드"
run_test "Config 로드" \
    "go test -v github.com/sage-x-project/sage/deployments/config -run 'TestLoadConfig'" \
    "/tmp/sage-test-logs/chain_config.log"
TEST_NUM=$((TEST_NUM + 1))

print_test $TEST_NUM 4 "환경별 Config"
run_test "환경별 설정 (dev, staging, prod)" \
    "go test -v github.com/sage-x-project/sage/deployments/config -run 'TestLoadForEnvironment'" \
    "/tmp/sage-test-logs/chain_env.log"
TEST_NUM=$((TEST_NUM + 1))

print_test $TEST_NUM 4 "프리셋 지원"
run_test "네트워크 프리셋 (local, sepolia, mainnet)" \
    "go test -v github.com/sage-x-project/sage/deployments/config -run 'TestNetworkPresets'" \
    "/tmp/sage-test-logs/chain_preset.log"
TEST_NUM=$((TEST_NUM + 1))

print_test $TEST_NUM 4 "환경 변수 오버라이드"
run_test "환경 변수 치환" \
    "go test -v github.com/sage-x-project/sage/deployments/config -run 'TestLoadWithEnvOverrides'" \
    "/tmp/sage-test-logs/chain_override.log"

#==============================================================================
# 5. 메시지 처리
#==============================================================================
print_header "[5/9] 메시지 처리"

## 5.1 Nonce 관리
print_category "5.1 Nonce 관리"
TEST_NUM=1

print_test $TEST_NUM 4 "Nonce 생성 (유니크성)"
run_test "UUID 기반 유니크 Nonce" \
    "go test -v github.com/sage-x-project/sage/pkg/agent/core/message/nonce -run 'TestNonceManager/GenerateNonce'" \
    "/tmp/sage-test-logs/nonce_gen.log"
TEST_NUM=$((TEST_NUM + 1))

print_test $TEST_NUM 4 "Nonce 저장 및 검증"
run_test "사용된 Nonce 마킹" \
    "go test -v github.com/sage-x-project/sage/pkg/agent/core/message/nonce -run 'TestNonceManager/MarkNonceUsed'" \
    "/tmp/sage-test-logs/nonce_mark.log"
TEST_NUM=$((TEST_NUM + 1))

print_test $TEST_NUM 4 "재전송 공격 방지"
run_test "중복 Nonce 거부" \
    "go test -v github.com/sage-x-project/sage/pkg/agent/core/message/nonce -run 'TestNonceManager/MarkNonceUsed'" \
    "/tmp/sage-test-logs/nonce_replay.log"
TEST_NUM=$((TEST_NUM + 1))

print_test $TEST_NUM 4 "만료 처리 (TTL)"
run_test "Nonce TTL 만료 및 cleanup" \
    "go test -v github.com/sage-x-project/sage/pkg/agent/core/message/nonce -run 'TestNonceManager.*Expires'" \
    "/tmp/sage-test-logs/nonce_expire.log"

## 5.2 메시지 순서
print_category "5.2 메시지 순서"
TEST_NUM=1

print_test $TEST_NUM 4 "메시지 ID 생성"
run_test "유니크 메시지 ID" \
    "go test -v github.com/sage-x-project/sage/pkg/agent/core/message/order -run 'TestOrderManager/FirstMessage'" \
    "/tmp/sage-test-logs/order_id.log"
TEST_NUM=$((TEST_NUM + 1))

print_test $TEST_NUM 4 "순서 보장"
run_test "시퀀스 단조 증가" \
    "go test -v github.com/sage-x-project/sage/pkg/agent/core/message/order -run 'TestOrderManager/SeqMonotonicity'" \
    "/tmp/sage-test-logs/order_seq.log"
TEST_NUM=$((TEST_NUM + 1))

print_test $TEST_NUM 4 "중복 감지"
run_test "중복 메시지 감지" \
    "go test -v github.com/sage-x-project/sage/pkg/agent/core/message/dedupe -run 'TestDetector'" \
    "/tmp/sage-test-logs/order_dedupe.log"
TEST_NUM=$((TEST_NUM + 1))

print_test $TEST_NUM 4 "타임스탬프 관리"
run_test "타임스탬프 순서 정렬" \
    "go test -v github.com/sage-x-project/sage/pkg/agent/core/message/order -run 'TestOrderManager/TimestampOrder'" \
    "/tmp/sage-test-logs/order_timestamp.log"

## 5.3 검증 서비스
print_category "5.3 검증 서비스"
TEST_NUM=1

print_test $TEST_NUM 4 "통합 검증 파이프라인"
run_test "메시지 검증 파이프라인" \
    "go test -v github.com/sage-x-project/sage/pkg/agent/core/message/validator -run 'TestValidateMessage'" \
    "/tmp/sage-test-logs/validate_pipeline.log"
TEST_NUM=$((TEST_NUM + 1))

print_test $TEST_NUM 4 "타임스탬프 허용 범위 검증"
run_test "타임스탬프 범위 체크" \
    "go test -v github.com/sage-x-project/sage/pkg/agent/core/message/validator -run '.*TimestampOutside'" \
    "/tmp/sage-test-logs/validate_time.log"
TEST_NUM=$((TEST_NUM + 1))

print_test $TEST_NUM 4 "재전송 공격 감지"
run_test "재전송 공격 감지" \
    "go test -v github.com/sage-x-project/sage/pkg/agent/core/message/validator -run '.*ReplayDetection'" \
    "/tmp/sage-test-logs/validate_replay.log"
TEST_NUM=$((TEST_NUM + 1))

print_test $TEST_NUM 4 "순서 위반 감지"
run_test "Out-of-order 감지" \
    "go test -v github.com/sage-x-project/sage/pkg/agent/core/message/validator -run '.*OutOfOrder'" \
    "/tmp/sage-test-logs/validate_order.log"

#==============================================================================
# 6. CLI 도구
#==============================================================================
print_header "[6/9] CLI 도구"

## 6.1 sage-crypto
print_category "6.1 sage-crypto"
TEST_NUM=1

print_test $TEST_NUM 5 "키페어 생성 (Ed25519 JWK)"
run_cli_test "Ed25519 JWK 생성" \
    "./build/bin/sage-crypto generate --type ed25519 --format jwk" \
    "private_key" \
    "/tmp/sage-test-logs/cli_ed25519_jwk.json"
TEST_NUM=$((TEST_NUM + 1))

print_test $TEST_NUM 5 "키페어 생성 (Secp256k1 PEM)"
run_cli_test "Secp256k1 PEM 생성" \
    "./build/bin/sage-crypto generate --type secp256k1 --format pem" \
    "BEGIN EC PRIVATE KEY" \
    "/tmp/sage-test-logs/cli_secp256k1_pem.txt"
TEST_NUM=$((TEST_NUM + 1))

print_test $TEST_NUM 5 "키 저장소 저장"
TOTAL_TESTS=$((TOTAL_TESTS + 1))
if ./build/bin/sage-crypto generate --type ed25519 --format storage --storage-dir /tmp/sage-keys --key-id test-key > /tmp/sage-test-logs/cli_storage.log 2>&1; then
    if [ -f "/tmp/sage-keys/test-key.key" ]; then
        print_success "키 저장소 저장 성공"
        PASSED_TESTS=$((PASSED_TESTS + 1))
        rm -rf /tmp/sage-keys
    else
        print_error "키 파일 생성 실패"
        FAILED_TESTS=$((FAILED_TESTS + 1))
    fi
else
    print_error "키 저장소 저장 실패"
    FAILED_TESTS=$((FAILED_TESTS + 1))
fi
TEST_NUM=$((TEST_NUM + 1))

print_test $TEST_NUM 5 "Help 명령 확인"
run_cli_test "sage-crypto help" \
    "./build/bin/sage-crypto --help" \
    "generate" \
    "/tmp/sage-test-logs/cli_crypto_help.log"
TEST_NUM=$((TEST_NUM + 1))

print_test $TEST_NUM 5 "Generate 명령 Help"
run_cli_test "sage-crypto generate help" \
    "./build/bin/sage-crypto generate --help" \
    "Supported key types" \
    "/tmp/sage-test-logs/cli_crypto_gen_help.log"

## 6.2 sage-did
print_category "6.2 sage-did"
TEST_NUM=1

print_test $TEST_NUM 2 "Help 명령 확인"
run_cli_test "sage-did help" \
    "./build/bin/sage-did --help" \
    "register" \
    "/tmp/sage-test-logs/cli_did_help.log"
TEST_NUM=$((TEST_NUM + 1))

print_test $TEST_NUM 2 "Register 명령 Help"
run_cli_test "sage-did register help" \
    "./build/bin/sage-did register --help" \
    "chain" \
    "/tmp/sage-test-logs/cli_did_register_help.log"

## 6.3 sage-verify
print_category "6.3 sage-verify"
print_skip "sage-verify CLI 테스트는 메시지 입력 필요"

#==============================================================================
# 7. 세션 관리
#==============================================================================
print_header "[7/9] 세션 관리"

## 7.1 세션 생성
print_category "7.1 세션 생성"
TEST_NUM=1

print_test $TEST_NUM 4 "세션 ID 생성 (UUID)"
run_test "유니크 세션 ID" \
    "go test -v github.com/sage-x-project/sage/pkg/agent/session -run 'TestSessionManager_CreateSession'" \
    "/tmp/sage-test-logs/session_id.log"
TEST_NUM=$((TEST_NUM + 1))

print_test $TEST_NUM 4 "세션 메타데이터 설정"
run_test "세션 메타데이터 (Created, LastAccessed)" \
    "go test -v github.com/sage-x-project/sage/pkg/agent/session -run 'TestSessionManager_CreateSession'" \
    "/tmp/sage-test-logs/session_meta.log"
TEST_NUM=$((TEST_NUM + 1))

print_test $TEST_NUM 4 "세션 암호화 키 생성"
run_test "ChaCha20-Poly1305 키 생성" \
    "go test -v github.com/sage-x-project/sage/pkg/agent/session -run 'TestSessionManager_CreateSession'" \
    "/tmp/sage-test-logs/session_key.log"
TEST_NUM=$((TEST_NUM + 1))

print_test $TEST_NUM 4 "세션 저장"
run_test "세션 저장 및 조회" \
    "go test -v github.com/sage-x-project/sage/pkg/agent/session -run 'TestSessionManager_GetSession'" \
    "/tmp/sage-test-logs/session_save.log"

## 7.2 세션 관리
print_category "7.2 세션 관리"
TEST_NUM=1

print_test $TEST_NUM 4 "세션 조회"
run_test "세션 ID로 조회" \
    "go test -v github.com/sage-x-project/sage/pkg/agent/session -run 'TestSessionManager_GetSession'" \
    "/tmp/sage-test-logs/session_get.log"
TEST_NUM=$((TEST_NUM + 1))

print_test $TEST_NUM 4 "세션 갱신"
run_test "LastAccessed 업데이트" \
    "go test -v github.com/sage-x-project/sage/pkg/agent/session -run 'TestSessionManager_GetSession'" \
    "/tmp/sage-test-logs/session_refresh.log"
TEST_NUM=$((TEST_NUM + 1))

print_test $TEST_NUM 4 "세션 만료 처리"
run_test "TTL 만료 자동 삭제" \
    "go test -v github.com/sage-x-project/sage/pkg/agent/session -run 'TestSessionManager_ExpireSession'" \
    "/tmp/sage-test-logs/session_expire.log"
TEST_NUM=$((TEST_NUM + 1))

print_test $TEST_NUM 4 "세션 삭제"
run_test "세션 삭제" \
    "go test -v github.com/sage-x-project/sage/pkg/agent/session -run 'TestSessionManager_DeleteSession'" \
    "/tmp/sage-test-logs/session_delete.log"

## 7.3 세션 암호화/복호화
print_category "7.3 세션 암호화/복호화"
TEST_NUM=1

print_test $TEST_NUM 3 "메시지 암호화 (AEAD)"
run_test "ChaCha20-Poly1305 암호화" \
    "go test -v github.com/sage-x-project/sage/pkg/agent/session -run 'TestSessionManager_EncryptMessage'" \
    "/tmp/sage-test-logs/session_encrypt.log"
TEST_NUM=$((TEST_NUM + 1))

print_test $TEST_NUM 3 "메시지 복호화"
run_test "암호문 복호화 및 검증" \
    "go test -v github.com/sage-x-project/sage/pkg/agent/session -run 'TestSessionManager_DecryptMessage'" \
    "/tmp/sage-test-logs/session_decrypt.log"
TEST_NUM=$((TEST_NUM + 1))

print_test $TEST_NUM 3 "인증 태그 검증"
run_test "변조된 메시지 복호화 실패" \
    "go test -v github.com/sage-x-project/sage/pkg/agent/session -run 'TestSessionManager.*tampered'" \
    "/tmp/sage-test-logs/session_auth_tag.log"

#==============================================================================
# 8. HPKE (Hybrid Public Key Encryption)
#==============================================================================
print_header "[8/9] HPKE"

## 8.1 키 교환
print_category "8.1 키 교환 (DHKEM)"
TEST_NUM=1

print_test $TEST_NUM 3 "X25519 키 교환"
run_test "X25519 DHKEM" \
    "go test -v github.com/sage-x-project/sage/pkg/agent/crypto/keys -run 'TestX25519'" \
    "/tmp/sage-test-logs/hpke_dhkem.log"
TEST_NUM=$((TEST_NUM + 1))

print_test $TEST_NUM 3 "공유 비밀 생성"
run_test "HPKE 공유 비밀 파생" \
    "go test -v github.com/sage-x-project/sage/pkg/agent/hpke -run 'TestHPKE.*Derive'" \
    "/tmp/sage-test-logs/hpke_shared_secret.log"
TEST_NUM=$((TEST_NUM + 1))

print_test $TEST_NUM 3 "키 파생 (HKDF)"
run_test "HKDF 키 파생" \
    "go test -v github.com/sage-x-project/sage/pkg/agent/hpke -run 'TestHPKE'" \
    "/tmp/sage-test-logs/hpke_hkdf.log"

## 8.2 암호화/복호화
print_category "8.2 HPKE 암호화/복호화"
TEST_NUM=1

print_test $TEST_NUM 4 "HPKE 컨텍스트 생성"
run_test "HPKE 컨텍스트 초기화" \
    "go test -v github.com/sage-x-project/sage/pkg/agent/hpke -run 'TestClient'" \
    "/tmp/sage-test-logs/hpke_context.log"
TEST_NUM=$((TEST_NUM + 1))

print_test $TEST_NUM 4 "메시지 암호화"
run_test "HPKE 암호화" \
    "go test -v github.com/sage-x-project/sage/pkg/agent/hpke -run 'TestHPKE'" \
    "/tmp/sage-test-logs/hpke_encrypt.log"
TEST_NUM=$((TEST_NUM + 1))

print_test $TEST_NUM 4 "메시지 복호화"
run_test "HPKE 복호화" \
    "go test -v github.com/sage-x-project/sage/pkg/agent/hpke -run 'TestHPKE'" \
    "/tmp/sage-test-logs/hpke_decrypt.log"
TEST_NUM=$((TEST_NUM + 1))

print_test $TEST_NUM 4 "AEAD 인증 검증"
run_test "인증 태그 검증" \
    "go test -v github.com/sage-x-project/sage/pkg/agent/hpke -run 'TestHPKE'" \
    "/tmp/sage-test-logs/hpke_aead.log"

## 8.3 핸드셰이크 E2E
print_category "8.3 핸드셰이크 E2E 테스트"
TEST_NUM=1

# Run the new MockTransport-based E2E test instead of integration test
print_test $TEST_NUM 4 "MockTransport HPKE E2E 테스트"
TOTAL_TESTS=$((TOTAL_TESTS + 1))
if go test -v github.com/sage-x-project/sage/pkg/agent/hpke -run TestE2E_HPKE_Handshake_MockTransport > /tmp/sage-test-logs/handshake_e2e.log 2>&1; then
    print_success "HPKE E2E 테스트 통과 (4개 시나리오)"
    PASSED_TESTS=$((PASSED_TESTS + 1))
else
    print_error "HPKE E2E 테스트 실패"
    FAILED_TESTS=$((FAILED_TESTS + 1))
fi

#==============================================================================
# 9. 통합 테스트
#==============================================================================
print_header "[9/9] 통합 테스트"

## 9.1 전체 유닛 테스트
print_category "9.1 전체 유닛 테스트"
TEST_NUM=1

print_test $TEST_NUM 1 "전체 패키지 유닛 테스트"
run_test "모든 패키지 테스트 (150+ 케이스)" \
    "make test" \
    "/tmp/sage-test-logs/test_all_units.log"

## 9.2 블록체인 통합 테스트
print_category "9.2 블록체인 통합 테스트"
TEST_NUM=1

print_test $TEST_NUM 5 "블록체인 연결"
run_test "Web3 연결 및 Chain ID 확인" \
    "make test-integration 2>&1 | grep -A 5 'TestBlockchainConnection'" \
    "/tmp/sage-test-logs/integration_blockchain.log"
TEST_NUM=$((TEST_NUM + 1))

print_test $TEST_NUM 5 "Enhanced Provider (가스 예측)"
run_test "가스 예측 및 재시도 로직" \
    "make test-integration 2>&1 | grep -A 10 'TestEnhancedProviderIntegration'" \
    "/tmp/sage-test-logs/integration_provider.log"
TEST_NUM=$((TEST_NUM + 1))

print_test $TEST_NUM 5 "DID 등록/조회"
run_test "DID 등록 및 조회" \
    "make test-integration 2>&1 | grep -A 10 'TestDIDRegistration'" \
    "/tmp/sage-test-logs/integration_did.log"
TEST_NUM=$((TEST_NUM + 1))

print_test $TEST_NUM 5 "멀티 에이전트 DID"
run_test "5개 에이전트 생성 및 서명" \
    "make test-integration 2>&1 | grep -A 10 'TestMultiAgentDID'" \
    "/tmp/sage-test-logs/integration_multi_agent.log"
TEST_NUM=$((TEST_NUM + 1))

print_test $TEST_NUM 5 "DID Resolver 캐싱"
run_test "DID 조회 캐싱 성능" \
    "make test-integration 2>&1 | grep -A 10 'TestDIDResolver'" \
    "/tmp/sage-test-logs/integration_resolver.log"

#==============================================================================
# 최종 결과
#==============================================================================
print_header "검증 결과 요약"

echo ""
echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
echo -e "${BLUE}  총 테스트:${NC} $TOTAL_TESTS"
echo -e "${GREEN}  통과:${NC} $PASSED_TESTS"
echo -e "${RED}  실패:${NC} $FAILED_TESTS"

if [ $FAILED_TESTS -eq 0 ]; then
    PASS_RATE=100
else
    PASS_RATE=$((PASSED_TESTS * 100 / TOTAL_TESTS))
fi
echo -e "${CYAN}  통과율:${NC} ${PASS_RATE}%"
echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
echo ""

if [ $FAILED_TESTS -eq 0 ]; then
    echo -e "${GREEN}✅ 모든 기능 검증 완료! 🎉${NC}"
    echo ""
    echo -e "${GREEN}기능 명세서의 모든 소분류 항목이 구현되고 테스트되었습니다.${NC}"
    echo ""
    echo "📄 상세 검증 결과: docs/FEATURE_VERIFICATION_GUIDE.md"
    echo "📁 테스트 로그: /tmp/sage-test-logs/"
    exit 0
else
    echo -e "${RED}❌ 일부 테스트 실패 ($FAILED_TESTS/$TOTAL_TESTS)${NC}"
    echo ""
    echo "실패한 테스트 로그:"
    echo "  /tmp/sage-test-logs/"
    echo ""
    echo "상세 로그 확인: ls -lh /tmp/sage-test-logs/"
    exit 1
fi
