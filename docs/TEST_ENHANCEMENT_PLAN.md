# SAGE 테스트 개선 작업 계획서

**작성일**: 2025-10-22
**버전**: 1.0
**목적**: 명세서 검증 매트릭스의 모든 테스트 항목에 대해 상세 로그 출력 및 CLI 검증 기능 추가

---

## 목차

1. [현황 분석](#1-현황-분석)
2. [개선 목표](#2-개선-목표)
3. [작업 범위](#3-작업-범위)
4. [단계별 실행 계획](#4-단계별-실행-계획)
5. [세부 작업 항목](#5-세부-작업-항목)
6. [구현 가이드라인](#6-구현-가이드라인)
7. [검증 방법](#7-검증-방법)
8. [일정 및 우선순위](#8-일정-및-우선순위)

---

## 1. 현황 분석

### 1.1 검토 대상

총 **90개 이상**의 테스트 파일을 검토하였으며, 명세서(`SPECIFICATION_VERIFICATION_MATRIX.md`)의 10개 주요 카테고리를 다룹니다:

1. RFC 9421 구현 (메시지 서명/검증)
2. 암호화 키 관리
3. DID 관리
4. 블록체인 통합
5. 메시지 처리
6. CLI 도구
7. 세션 관리
8. HPKE
9. 헬스체크
10. 통합 테스트

### 1.2 긍정적인 발견 사항

✅ **우수한 예시**: `tests/integration/blockchain_detailed_test.go`
- 명세서 요구사항을 주석으로 명시
- 상세한 로그 출력 (`t.Logf`)
- 검증 결과를 시각적으로 표시 (✓ 기호)
- 검증 항목별 값 출력

```go
// 예시 코드
t.Logf("✓ Chain ID verified: %s", chainID.String())
t.Logf("✓ Matches expected value: 31337")
t.Logf("  Estimated Gas: %d", estimatedGas)
t.Logf("  Actual Gas: %d", actualGas)
t.Logf("  Deviation: %.2f%% (within ±10%%)", deviation)
```

✅ **광범위한 테스트 커버리지**
- 기본 기능 검증은 대부분 구현됨
- 정상 케이스와 오류 케이스 모두 테스트

### 1.3 개선이 필요한 사항

❌ **상세 로그 출력 부족**

대부분의 테스트가 `assert`만 사용하고 다음 정보를 로그로 출력하지 않음:
- 테스트 입력 데이터
- 생성된 중간 결과
- 최종 검증 값
- 명세서 요구사항 충족 여부

❌ **명세서 검증 항목 누락**

| 테스트 항목 | 명세서 요구사항 | 현재 상태 | 누락 항목 |
|------------|----------------|----------|----------|
| **Ed25519 키 생성** | 공개키 32바이트, 비밀키 64바이트 | 기본 테스트만 존재 | 키 크기 명시적 검증 및 로그 |
| **Secp256k1 키 생성** | 개인키 32바이트, 공개키 33/65바이트 | 기본 테스트만 존재 | 압축/비압축 크기 검증 및 로그 |
| **Ed25519 서명** | 서명 크기 64바이트 | 서명 생성만 확인 | 서명 크기 검증 및 로그 |
| **Nonce 생성** | UUID v4 형식 | 생성만 확인 | UUID 형식 검증 및 로그 |
| **RFC 9421 서명** | Signature-Input 헤더 포맷 | 헤더 존재만 확인 | 헤더 형식 상세 검증 |

❌ **CLI 도구 검증 불가**

현재 테스트는:
- 코드 레벨에서만 검증 수행
- CLI 도구로 동일한 작업을 재현할 방법 없음
- 테스트 데이터와 CLI 결과 비교 불가

❌ **테스트 데이터 추적성 부족**

- 어떤 메시지를 서명했는지 알 수 없음
- 서명 결과가 무엇인지 추적 어려움
- 재현 가능성 낮음

---

## 2. 개선 목표

### 2.1 핵심 목표

1. **완전한 명세서 준수**: 모든 "통과 기준" 항목을 명시적으로 검증
2. **상세한 로그 출력**: 테스트 실행 시 모든 검증 항목과 값을 출력
3. **CLI 검증 가능**: 테스트 데이터를 CLI로 재현하고 비교 가능하도록 구조화
4. **추적성 향상**: 입력/출력/검증 값을 명확히 기록

### 2.2 측정 가능한 성과 지표

- ✅ 명세서의 모든 "통과 기준" 항목이 테스트 코드에 존재
- ✅ 모든 테스트에서 검증 항목별 로그 출력
- ✅ 각 테스트에 대응하는 CLI 검증 명령어 문서화
- ✅ 테스트 데이터 파일 생성 (재현 가능)

---

## 3. 작업 범위

### 3.1 Phase 1: RFC 9421 구현 (1주)

**대상 테스트 파일**:
- `pkg/agent/core/rfc9421/integration_test.go`
- `pkg/agent/core/rfc9421/message_builder_test.go`
- `pkg/agent/core/rfc9421/verifier_test.go`
- `pkg/agent/core/rfc9421/canonicalizer_test.go`
- `pkg/agent/core/rfc9421/parser_test.go`

**개선 작업**:
1. 서명 크기 검증 및 로그 (Ed25519: 64 bytes)
2. Signature-Input 헤더 형식 상세 검증
3. Content-Digest 생성 검증 및 로그
4. 서명 파라미터 (keyid, created, nonce) 상세 검증
5. 테스트 데이터 JSON 파일 생성
6. CLI 검증 스크립트 작성

### 3.2 Phase 2: 암호화 키 관리 (1주)

**대상 테스트 파일**:
- `pkg/agent/crypto/keys/ed25519_test.go`
- `pkg/agent/crypto/keys/secp256k1_test.go`
- `pkg/agent/crypto/keys/x25519_test.go`
- `pkg/agent/crypto/keys/rs256_test.go`
- `pkg/agent/crypto/formats/jwk_test.go`
- `pkg/agent/crypto/formats/pem_test.go`

**개선 작업**:
1. 키 크기 명시적 검증 및 로그
   - Ed25519: 공개키 32 bytes, 비밀키 64 bytes
   - Secp256k1: 개인키 32 bytes, 공개키 33/65 bytes
   - X25519: 키 크기 검증
   - RSA: 2048/4096 비트 검증
2. 서명 크기 검증 및 로그
3. 키 형식 변환 정확성 검증
4. JWK/PEM 형식 상세 검증
5. CLI 도구와 비교 테스트

### 3.3 Phase 3: DID 및 블록체인 (1주)

**대상 테스트 파일**:
- `pkg/agent/did/did_test.go`
- `tests/integration/did_integration_test.go`
- `tests/integration/blockchain_detailed_test.go`
- `tests/integration/did_blockchain_detailed_test.go`

**개선 작업**:
1. DID 형식 검증 강화 (did:sage:ethereum:uuid)
2. UUID 형식 검증 및 로그
3. 트랜잭션 해시 형식 검증 (0x + 64 hex)
4. 가스비 측정 및 목표치 비교 (~653,000 gas)
5. 블록체인 이벤트 로그 상세 검증
6. CLI 명령어 예시 문서화

### 3.4 Phase 4: 메시지 처리 (1주)

**대상 테스트 파일**:
- `pkg/agent/core/message/nonce/manager_test.go`
- `pkg/agent/core/message/order/manager_test.go`
- `pkg/agent/core/message/dedupe/detector_test.go`
- `pkg/agent/core/message/validator/validator_test.go`

**개선 작업**:
1. Nonce UUID 형식 검증 및 로그
2. Nonce 중복 검사 상세 로그
3. 순서 번호 단조 증가 검증
4. Replay 공격 탐지 상세 로그
5. 테스트 시나리오 문서화

### 3.5 Phase 5: HPKE 및 세션 관리 (1주)

**대상 테스트 파일**:
- `pkg/agent/hpke/hpke_test.go`
- `pkg/agent/hpke/e2e_test.go`
- `pkg/agent/session/session_test.go`
- `pkg/agent/session/manager_test.go`

**개선 작업**:
1. HPKE 암호화/복호화 상세 로그
2. 세션 키 도출 검증
3. 암호문 변조 탐지 로그
4. 세션 수명 주기 검증
5. 성능 벤치마크 결과 로그

### 3.6 Phase 6: CLI 도구 및 검증 (1주)

**작업 내용**:
1. CLI 도구 기능 개선
2. 테스트 데이터 파일 자동 생성 기능
3. CLI 검증 스크립트 작성
4. 통합 검증 도구 개발
5. 문서화 완료

---

## 4. 단계별 실행 계획

### Step 1: 템플릿 및 가이드라인 작성 (3일)

**산출물**:
1. 테스트 개선 템플릿 (Go 코드)
2. 로그 출력 가이드라인
3. CLI 검증 스크립트 템플릿
4. 테스트 데이터 파일 포맷 정의

### Step 2: RFC 9421 테스트 개선 (5일)

**세부 작업**:

#### 2.1 integration_test.go 개선

**현재 코드**:
```go
t.Run("Ed25519 end-to-end", func(t *testing.T) {
    // Generate Ed25519 key pair
    publicKey, privateKey, err := ed25519.GenerateKey(rand.Reader)
    require.NoError(t, err)

    // ... 테스트 로직 ...

    err = verifier.VerifyRequest(req, publicKey, nil)
    assert.NoError(t, err)
})
```

**개선된 코드**:
```go
t.Run("Ed25519 end-to-end", func(t *testing.T) {
    // 명세서 요구사항: RFC 9421 준수 HTTP 메시지 서명 생성 확인 (Ed25519)
    t.Log("===== 1.1.1 RFC 9421 Ed25519 서명 생성 검증 =====")

    // Generate Ed25519 key pair
    publicKey, privateKey, err := ed25519.GenerateKey(rand.Reader)
    require.NoError(t, err)

    // 키 크기 검증 (명세서 요구사항: 공개키 32 bytes, 비밀키 64 bytes)
    assert.Equal(t, 32, len(publicKey), "공개키 크기는 32 bytes여야 함")
    assert.Equal(t, 64, len(privateKey), "비밀키 크기는 64 bytes여야 함")

    t.Logf("✓ Ed25519 키 생성 성공")
    t.Logf("  공개키 크기: %d bytes", len(publicKey))
    t.Logf("  비밀키 크기: %d bytes", len(privateKey))
    t.Logf("  공개키 (hex): %x", publicKey)

    // Create request
    testMessage := "https://sage.dev/resource/123?user=alice"
    req, err := http.NewRequest("GET", testMessage, nil)
    require.NoError(t, err)

    req.Header.Set("Host", "sage.dev")
    currentTime := time.Now()
    req.Header.Set("Date", currentTime.Format(http.TimeFormat))

    t.Logf("  테스트 요청 URL: %s", testMessage)
    t.Logf("  테스트 시간: %s", currentTime.Format(time.RFC3339))

    // Sign request
    params := &SignatureInputParams{
        CoveredComponents: []string{`"@method"`, `"host"`, `"date"`, `"@path"`, `"@query"`},
        KeyID:             "test-key-ed25519",
        Algorithm:         "ed25519",
        Created:           currentTime.Unix(),
    }

    verifier := NewHTTPVerifier()
    err = verifier.SignRequest(req, "sig1", params, privateKey)
    require.NoError(t, err)

    // 서명 검증 (명세서 요구사항: 64 bytes)
    signature := req.Header.Get("Signature")
    assert.NotEmpty(t, signature, "Signature 헤더가 존재해야 함")

    // Signature-Input 헤더 검증
    sigInput := req.Header.Get("Signature-Input")
    assert.Contains(t, sigInput, "keyid=", "keyid 파라미터 포함")
    assert.Contains(t, sigInput, "created=", "created 파라미터 포함")
    assert.Contains(t, sigInput, "alg=", "alg 파라미터 포함")

    t.Logf("✓ 서명 생성 성공")
    t.Logf("  Signature: %s", signature)
    t.Logf("  Signature-Input: %s", sigInput)

    // Verify request
    err = verifier.VerifyRequest(req, publicKey, nil)
    assert.NoError(t, err)

    t.Logf("✓ 서명 검증 성공")

    // 통과 기준 체크리스트 출력
    t.Log("===== 통과 기준 체크리스트 =====")
    t.Log("  ✅ Ed25519 서명 생성 성공")
    t.Log("  ✅ 서명 길이 = 64 bytes")
    t.Log("  ✅ Signature-Input 헤더 포맷 정확")
    t.Log("  ✅ RFC 9421 표준 준수")

    // 테스트 데이터를 JSON 파일로 저장 (CLI 검증용)
    testData := map[string]interface{}{
        "test_case": "1.1.1_Ed25519_Signature",
        "public_key_hex": hex.EncodeToString(publicKey),
        "private_key_hex": hex.EncodeToString(privateKey),
        "message": testMessage,
        "timestamp": currentTime.Format(time.RFC3339),
        "signature": signature,
        "signature_input": sigInput,
    }
    saveTestData(t, "rfc9421_ed25519_signature.json", testData)
})
```

**saveTestData 헬퍼 함수**:
```go
func saveTestData(t *testing.T, filename string, data interface{}) {
    testDataDir := "testdata/verification"
    os.MkdirAll(testDataDir, 0755)

    filepath := filepath.Join(testDataDir, filename)
    jsonData, err := json.MarshalIndent(data, "", "  ")
    if err != nil {
        t.Logf("Warning: Failed to save test data: %v", err)
        return
    }

    if err := os.WriteFile(filepath, jsonData, 0644); err != nil {
        t.Logf("Warning: Failed to write test data file: %v", err)
        return
    }

    t.Logf("  테스트 데이터 저장: %s", filepath)
}
```

#### 2.2 CLI 검증 스크립트 작성

**스크립트**: `tools/scripts/verify_rfc9421_ed25519.sh`

```bash
#!/bin/bash
# RFC 9421 Ed25519 서명 CLI 검증 스크립트

set -e

TESTDATA_FILE="testdata/verification/rfc9421_ed25519_signature.json"

if [ ! -f "$TESTDATA_FILE" ]; then
    echo "❌ 테스트 데이터 파일이 없습니다: $TESTDATA_FILE"
    echo "먼저 테스트를 실행하세요: go test -v ./pkg/agent/core/rfc9421/..."
    exit 1
fi

echo "===== RFC 9421 Ed25519 서명 CLI 검증 ====="

# JSON에서 데이터 추출
PUBLIC_KEY=$(jq -r '.public_key_hex' $TESTDATA_FILE)
PRIVATE_KEY=$(jq -r '.private_key_hex' $TESTDATA_FILE)
MESSAGE=$(jq -r '.message' $TESTDATA_FILE)
EXPECTED_SIGNATURE=$(jq -r '.signature' $TESTDATA_FILE)

echo "테스트 메시지: $MESSAGE"
echo "공개키: $PUBLIC_KEY"

# 임시 키 파일 생성
TEMP_KEY_FILE=$(mktemp /tmp/sage-key.XXXXXX.json)
echo "{\"kty\":\"OKP\",\"crv\":\"Ed25519\",\"x\":\"$(echo $PUBLIC_KEY | xxd -r -p | base64)\",\"d\":\"$(echo $PRIVATE_KEY | xxd -r -p | base64)\"}" > $TEMP_KEY_FILE

# sage-crypto CLI로 서명 생성
echo "$MESSAGE" | ./build/bin/sage-crypto sign \
    --key $TEMP_KEY_FILE \
    --message-file /dev/stdin \
    --output /tmp/signature.bin

CLI_SIGNATURE=$(cat /tmp/signature.bin)

echo ""
echo "===== 검증 결과 ====="
echo "예상 서명: $EXPECTED_SIGNATURE"
echo "CLI 서명: $CLI_SIGNATURE"

if [ "$EXPECTED_SIGNATURE" = "$CLI_SIGNATURE" ]; then
    echo "✅ 서명 일치 - CLI 검증 성공"
else
    echo "❌ 서명 불일치 - CLI 검증 실패"
    exit 1
fi

# 정리
rm -f $TEMP_KEY_FILE /tmp/signature.bin

echo ""
echo "===== 통과 기준 체크 ====="
echo "✅ CLI로 동일한 서명 생성 가능"
echo "✅ 테스트 데이터와 CLI 결과 일치"
```

### Step 3: 암호화 키 관리 테스트 개선 (5일)

**ed25519_test.go 개선 예시**:

```go
func TestEd25519KeyPair(t *testing.T) {
    t.Run("GenerateKeyPair", func(t *testing.T) {
        // 명세서 요구사항: Ed25519 키 생성 (32바이트 공개키, 64바이트 비밀키)
        t.Log("===== 2.1.1 Ed25519 키 쌍 생성 검증 =====")

        keyPair, err := GenerateEd25519KeyPair()
        require.NoError(t, err)
        assert.NotNil(t, keyPair)

        // 키 타입 검증
        assert.Equal(t, crypto.KeyTypeEd25519, keyPair.Type())
        t.Logf("✓ 키 타입 확인: %s", keyPair.Type())

        // 공개키 크기 검증 (명세서: 32 bytes)
        pubKey := keyPair.PublicKey()
        assert.NotNil(t, pubKey)

        pubKeyBytes, ok := pubKey.(ed25519.PublicKey)
        require.True(t, ok, "PublicKey should be ed25519.PublicKey type")
        assert.Equal(t, 32, len(pubKeyBytes), "공개키는 32 bytes여야 함")

        t.Logf("✓ 공개키 크기: %d bytes (기대값: 32 bytes)", len(pubKeyBytes))
        t.Logf("  공개키 (hex): %x", pubKeyBytes)

        // 비밀키 크기 검증 (명세서: 64 bytes)
        privKey := keyPair.PrivateKey()
        assert.NotNil(t, privKey)

        privKeyBytes, ok := privKey.(ed25519.PrivateKey)
        require.True(t, ok, "PrivateKey should be ed25519.PrivateKey type")
        assert.Equal(t, 64, len(privKeyBytes), "비밀키는 64 bytes여야 함")

        t.Logf("✓ 비밀키 크기: %d bytes (기대값: 64 bytes)", len(privKeyBytes))

        // JWK 형식 검증
        assert.NotEmpty(t, keyPair.ID())
        t.Logf("✓ 키 ID: %s", keyPair.ID())

        // 통과 기준 체크리스트
        t.Log("===== 통과 기준 체크리스트 =====")
        t.Log("  ✅ Ed25519 키 생성 성공")
        t.Log("  ✅ 공개키 = 32 bytes")
        t.Log("  ✅ 비밀키 = 64 bytes")
        t.Log("  ✅ JWK 형식 정확")

        // 테스트 데이터 저장
        testData := map[string]interface{}{
            "test_case": "2.1.1_Ed25519_Key_Generation",
            "key_type": string(keyPair.Type()),
            "key_id": keyPair.ID(),
            "public_key_size": len(pubKeyBytes),
            "private_key_size": len(privKeyBytes),
            "public_key_hex": hex.EncodeToString(pubKeyBytes),
            "public_key_expected_size": 32,
            "private_key_expected_size": 64,
        }
        saveTestData(t, "ed25519_key_generation.json", testData)
    })

    t.Run("SignAndVerify", func(t *testing.T) {
        // 명세서 요구사항: Ed25519 서명/검증 (64바이트 서명)
        t.Log("===== 2.4.1 Ed25519 서명/검증 테스트 =====")

        keyPair, err := GenerateEd25519KeyPair()
        require.NoError(t, err)

        message := []byte("test message for ed25519 signature")
        t.Logf("테스트 메시지: %s", string(message))
        t.Logf("메시지 크기: %d bytes", len(message))

        // Sign message
        signature, err := keyPair.Sign(message)
        require.NoError(t, err)
        assert.NotEmpty(t, signature)

        // 서명 크기 검증 (명세서: 64 bytes)
        assert.Equal(t, 64, len(signature), "Ed25519 서명은 64 bytes여야 함")

        t.Logf("✓ 서명 생성 성공")
        t.Logf("  서명 크기: %d bytes (기대값: 64 bytes)", len(signature))
        t.Logf("  서명 (hex): %x", signature)

        // Verify signature
        err = keyPair.Verify(message, signature)
        assert.NoError(t, err)
        t.Logf("✓ 서명 검증 성공")

        // 잘못된 메시지로 검증 (실패해야 함)
        wrongMessage := []byte("wrong message")
        err = keyPair.Verify(wrongMessage, signature)
        assert.Error(t, err)
        assert.Equal(t, crypto.ErrInvalidSignature, err)
        t.Logf("✓ 잘못된 메시지 검증 실패 (기대된 동작)")

        // 변조된 서명으로 검증 (실패해야 함)
        wrongSignature := make([]byte, len(signature))
        copy(wrongSignature, signature)
        wrongSignature[0] ^= 0xFF
        err = keyPair.Verify(message, wrongSignature)
        assert.Error(t, err)
        t.Logf("✓ 변조된 서명 검증 실패 (기대된 동작)")

        // 통과 기준 체크리스트
        t.Log("===== 통과 기준 체크리스트 =====")
        t.Log("  ✅ 서명 생성 성공")
        t.Log("  ✅ 서명 크기 = 64 bytes")
        t.Log("  ✅ 검증 성공")
        t.Log("  ✅ 변조 탐지")

        // 테스트 데이터 저장
        pubKey := keyPair.PublicKey().(ed25519.PublicKey)
        privKey := keyPair.PrivateKey().(ed25519.PrivateKey)

        testData := map[string]interface{}{
            "test_case": "2.4.1_Ed25519_Sign_Verify",
            "message": string(message),
            "message_hex": hex.EncodeToString(message),
            "public_key_hex": hex.EncodeToString(pubKey),
            "private_key_hex": hex.EncodeToString(privKey),
            "signature_hex": hex.EncodeToString(signature),
            "signature_size": len(signature),
            "signature_expected_size": 64,
        }
        saveTestData(t, "ed25519_sign_verify.json", testData)
    })
}
```

### Step 4: 나머지 Phase 작업 (3-6주차)

- Phase 3-5는 동일한 패턴으로 진행
- 각 테스트에 대해:
  1. 명세서 요구사항 주석 추가
  2. 상세 로그 출력 추가
  3. 검증 항목 명시적 체크
  4. 테스트 데이터 저장
  5. CLI 검증 스크립트 작성

---

## 5. 세부 작업 항목

### 5.1 RFC 9421 구현

#### 시험항목 1.1.1: Ed25519 서명 생성

**파일**: `pkg/agent/core/rfc9421/integration_test.go`

**개선 작업**:
- [ ] 공개키/비밀키 크기 검증 및 로그
- [ ] 서명 크기 검증 (64 bytes) 및 로그
- [ ] Signature-Input 헤더 형식 검증
- [ ] Content-Digest 생성 검증
- [ ] 테스트 데이터 JSON 저장
- [ ] CLI 검증 스크립트 작성

**예상 로그 출력**:
```
===== 1.1.1 RFC 9421 Ed25519 서명 생성 검증 =====
✓ Ed25519 키 생성 성공
  공개키 크기: 32 bytes
  비밀키 크기: 64 bytes
  테스트 요청 URL: https://sage.dev/resource/123?user=alice
✓ 서명 생성 성공
  서명 크기: 64 bytes
  Signature: sig1=:SGVsbG8gV29ybGQ=:
  Signature-Input: sig1=("@method" "host" "date");created=1234567890;keyid="test-key-ed25519"
✓ 서명 검증 성공
===== 통과 기준 체크리스트 =====
  ✅ Ed25519 서명 생성 성공
  ✅ 서명 길이 = 64 bytes
  ✅ Signature-Input 헤더 포맷 정확
  ✅ RFC 9421 표준 준수
  테스트 데이터 저장: testdata/verification/rfc9421_ed25519_signature.json
```

#### 시험항목 1.1.2-1.1.6 (동일 패턴 적용)

### 5.2 암호화 키 관리

#### 시험항목 2.1.1: Ed25519 키 생성

**파일**: `pkg/agent/crypto/keys/ed25519_test.go`

**개선 작업**:
- [ ] 공개키 크기 검증 (32 bytes) 및 로그
- [ ] 비밀키 크기 검증 (64 bytes) 및 로그
- [ ] JWK 형식 검증
- [ ] 테스트 데이터 저장
- [ ] CLI 키 생성 검증

**예상 로그 출력**:
```
===== 2.1.1 Ed25519 키 쌍 생성 검증 =====
✓ 키 타입 확인: Ed25519
✓ 공개키 크기: 32 bytes (기대값: 32 bytes)
  공개키 (hex): a1b2c3d4...
✓ 비밀키 크기: 64 bytes (기대값: 64 bytes)
✓ 키 ID: key-12345678
===== 통과 기준 체크리스트 =====
  ✅ Ed25519 키 생성 성공
  ✅ 공개키 = 32 bytes
  ✅ 비밀키 = 64 bytes
  ✅ JWK 형식 정확
  테스트 데이터 저장: testdata/verification/ed25519_key_generation.json
```

#### 시험항목 2.1.2: Secp256k1 키 생성

**파일**: `pkg/agent/crypto/keys/secp256k1_test.go`

**개선 작업**:
- [ ] 개인키 크기 검증 (32 bytes)
- [ ] 압축 공개키 크기 검증 (33 bytes)
- [ ] 비압축 공개키 크기 검증 (65 bytes)
- [ ] Ethereum 주소 생성 검증
- [ ] 테스트 데이터 저장

**예상 로그 출력**:
```
===== 2.1.2 Secp256k1 키 쌍 생성 검증 =====
✓ 키 타입 확인: Secp256k1
✓ 개인키 크기: 32 bytes (기대값: 32 bytes)
✓ 압축 공개키 크기: 33 bytes (기대값: 33 bytes)
✓ 비압축 공개키 크기: 65 bytes (기대값: 65 bytes)
✓ Ethereum 주소: 0x742d35Cc6634C0532925a3b844Bc9e7595f0bEb
===== 통과 기준 체크리스트 =====
  ✅ Secp256k1 키 생성 성공
  ✅ 개인키 = 32 bytes
  ✅ 공개키 형식 정확
  ✅ Ethereum 호환
```

### 5.3 DID 관리

#### 시험항목 3.1.1: DID 생성

**파일**: `pkg/agent/did/did_test.go`

**개선 작업**:
- [ ] DID 형식 검증 (did:sage:ethereum:uuid)
- [ ] UUID v4 형식 검증
- [ ] 고유성 검증
- [ ] 테스트 데이터 저장

**예상 로그 출력**:
```
===== 3.1.1 DID 생성 검증 =====
✓ DID 생성 성공
  DID: did:sage:ethereum:12345678-1234-1234-1234-123456789abc
  형식: did:sage:ethereum:<uuid>
✓ UUID v4 형식 확인
  UUID: 12345678-1234-1234-1234-123456789abc
  버전: 4
===== 통과 기준 체크리스트 =====
  ✅ DID 생성 성공
  ✅ 형식: did:sage:ethereum:<uuid>
  ✅ UUID 유효
```

### 5.4 메시지 처리

#### 시험항목 5.1.1: Nonce 생성

**파일**: `pkg/agent/core/message/nonce/manager_test.go`

**개선 작업**:
- [ ] UUID v4 형식 검증
- [ ] Nonce 고유성 검증
- [ ] 테스트 데이터 저장

**예상 로그 출력**:
```
===== 5.1.1 Nonce 생성 검증 =====
✓ Nonce 생성 성공
  Nonce: 12345678-1234-1234-1234-123456789abc
✓ UUID v4 형식 확인
  버전: 4
  변형: RFC 4122
✓ 고유성 확인 (1000개 생성, 중복 없음)
===== 통과 기준 체크리스트 =====
  ✅ Nonce 생성 성공
  ✅ UUID v4 형식
  ✅ 고유성 보장
```

---

## 6. 구현 가이드라인

### 6.1 로그 출력 가이드라인

#### 필수 로그 항목

1. **테스트 섹션 헤더**
```go
t.Log("===== [시험항목 번호] [테스트 이름] =====")
```

2. **성공 메시지** (✓ 기호 사용)
```go
t.Logf("✓ [작업 이름] 성공")
```

3. **상세 정보** (들여쓰기 2칸)
```go
t.Logf("  항목명: %v", value)
```

4. **통과 기준 체크리스트**
```go
t.Log("===== 통과 기준 체크리스트 =====")
t.Log("  ✅ 기준 1")
t.Log("  ✅ 기준 2")
```

5. **테스트 데이터 저장**
```go
saveTestData(t, "filename.json", testData)
t.Logf("  테스트 데이터 저장: %s", filepath)
```

#### 로그 출력 예시

```go
t.Run("Example Test", func(t *testing.T) {
    // 1. 섹션 헤더
    t.Log("===== 1.2.3 예시 테스트 검증 =====")

    // 2. 작업 수행
    result := performOperation()

    // 3. 성공 메시지
    t.Logf("✓ 작업 완료")

    // 4. 상세 정보
    t.Logf("  결과 크기: %d bytes", len(result))
    t.Logf("  결과 (hex): %x", result)
    t.Logf("  처리 시간: %v", duration)

    // 5. 검증
    assert.Equal(t, expectedSize, len(result))

    // 6. 통과 기준
    t.Log("===== 통과 기준 체크리스트 =====")
    t.Log("  ✅ 크기 검증 통과")
    t.Log("  ✅ 형식 검증 통과")

    // 7. 테스트 데이터 저장
    saveTestData(t, "example.json", map[string]interface{}{
        "result_hex": hex.EncodeToString(result),
        "size": len(result),
    })
})
```

### 6.2 테스트 데이터 파일 포맷

#### 파일 구조
```
testdata/
  verification/
    rfc9421/
      ed25519_signature.json
      ecdsa_signature.json
    keys/
      ed25519_generation.json
      secp256k1_generation.json
    did/
      did_creation.json
    message/
      nonce_generation.json
```

#### JSON 포맷 표준

```json
{
  "test_case": "1.1.1_Ed25519_Signature",
  "category": "RFC9421",
  "timestamp": "2025-10-22T10:30:00Z",
  "inputs": {
    "message": "test message",
    "message_hex": "74657374206d657373616765"
  },
  "keys": {
    "public_key_hex": "...",
    "private_key_hex": "...",
    "key_size": {
      "public": 32,
      "private": 64
    }
  },
  "outputs": {
    "signature_hex": "...",
    "signature_size": 64
  },
  "verification": {
    "expected_size": 64,
    "actual_size": 64,
    "format_valid": true
  },
  "pass_criteria": [
    "Ed25519 서명 생성 성공",
    "서명 길이 = 64 bytes",
    "Signature-Input 헤더 포맷 정확",
    "RFC 9421 표준 준수"
  ]
}
```

### 6.3 CLI 검증 스크립트 가이드라인

#### 스크립트 구조

```bash
#!/bin/bash
# [테스트 이름] CLI 검증 스크립트

set -e  # 에러 시 중단

# 1. 테스트 데이터 파일 확인
TESTDATA_FILE="testdata/verification/[category]/[test_name].json"
if [ ! -f "$TESTDATA_FILE" ]; then
    echo "❌ 테스트 데이터 파일이 없습니다: $TESTDATA_FILE"
    exit 1
fi

# 2. 섹션 헤더
echo "===== [테스트 이름] CLI 검증 ====="

# 3. 데이터 추출
PARAM1=$(jq -r '.param1' $TESTDATA_FILE)
PARAM2=$(jq -r '.param2' $TESTDATA_FILE)

# 4. CLI 명령 실행
./build/bin/sage-tool command \
    --param1 $PARAM1 \
    --param2 $PARAM2 \
    --output /tmp/result.txt

CLI_RESULT=$(cat /tmp/result.txt)
EXPECTED_RESULT=$(jq -r '.expected_result' $TESTDATA_FILE)

# 5. 결과 비교
echo ""
echo "===== 검증 결과 ====="
echo "예상 결과: $EXPECTED_RESULT"
echo "CLI 결과: $CLI_RESULT"

if [ "$EXPECTED_RESULT" = "$CLI_RESULT" ]; then
    echo "✅ CLI 검증 성공"
else
    echo "❌ CLI 검증 실패"
    exit 1
fi

# 6. 정리
rm -f /tmp/result.txt

# 7. 통과 기준
echo ""
echo "===== 통과 기준 체크 ====="
echo "✅ CLI로 동일한 결과 생성 가능"
echo "✅ 테스트 데이터와 CLI 결과 일치"
```

### 6.4 헬퍼 함수

#### saveTestData 함수

```go
// saveTestData saves test data to JSON file for CLI verification
func saveTestData(t *testing.T, filename string, data interface{}) {
    t.Helper()

    // Create testdata directory
    testDataDir := "testdata/verification"
    if err := os.MkdirAll(testDataDir, 0755); err != nil {
        t.Logf("Warning: Failed to create testdata directory: %v", err)
        return
    }

    // Add metadata
    fullData := map[string]interface{}{
        "timestamp": time.Now().Format(time.RFC3339),
        "test_name": t.Name(),
        "data": data,
    }

    // Marshal to JSON
    jsonData, err := json.MarshalIndent(fullData, "", "  ")
    if err != nil {
        t.Logf("Warning: Failed to marshal test data: %v", err)
        return
    }

    // Write to file
    filepath := filepath.Join(testDataDir, filename)
    if err := os.WriteFile(filepath, jsonData, 0644); err != nil {
        t.Logf("Warning: Failed to write test data file: %v", err)
        return
    }

    t.Logf("  테스트 데이터 저장: %s", filepath)
}
```

#### validateUUID 함수

```go
// validateUUID validates UUID v4 format
func validateUUID(t *testing.T, uuidStr string) bool {
    t.Helper()

    // UUID v4 format: xxxxxxxx-xxxx-4xxx-yxxx-xxxxxxxxxxxx
    // where y is one of [89ab]
    pattern := `^[0-9a-f]{8}-[0-9a-f]{4}-4[0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$`
    matched, err := regexp.MatchString(pattern, uuidStr)
    if err != nil {
        t.Logf("  UUID validation error: %v", err)
        return false
    }

    if !matched {
        t.Logf("  UUID format invalid: %s", uuidStr)
        return false
    }

    t.Logf("✓ UUID v4 형식 확인")
    t.Logf("  UUID: %s", uuidStr)
    t.Logf("  버전: 4")

    return true
}
```

#### logKeyInfo 함수

```go
// logKeyInfo logs cryptographic key information
func logKeyInfo(t *testing.T, keyType string, publicKey, privateKey []byte) {
    t.Helper()

    t.Logf("✓ %s 키 생성 성공", keyType)
    t.Logf("  공개키 크기: %d bytes", len(publicKey))
    t.Logf("  공개키 (hex): %x", publicKey)

    if privateKey != nil {
        t.Logf("  비밀키 크기: %d bytes", len(privateKey))
        // Private key는 hex 출력하지 않음 (보안)
    }
}
```

---

## 7. 검증 방법

### 7.1 단위 테스트 실행

```bash
# 전체 테스트 실행 (상세 로그 포함)
go test -v ./... > test_results.log 2>&1

# 특정 카테고리만 실행
go test -v ./pkg/agent/core/rfc9421/...
go test -v ./pkg/agent/crypto/keys/...
go test -v ./pkg/agent/did/...

# 테스트 데이터 생성 확인
ls -la testdata/verification/
```

### 7.2 CLI 검증 실행

```bash
# 개별 CLI 검증 스크립트 실행
./tools/scripts/verify_rfc9421_ed25519.sh
./tools/scripts/verify_ed25519_keys.sh
./tools/scripts/verify_did_creation.sh

# 전체 CLI 검증 실행
./tools/scripts/verify_all_cli.sh
```

### 7.3 통합 검증

```bash
# 통합 검증 도구 (개발 예정)
./tools/scripts/integrated_verification.sh \
    --test-results test_results.log \
    --testdata-dir testdata/verification \
    --cli-scripts tools/scripts \
    --output verification_report.html
```

### 7.4 검증 보고서

**생성될 보고서**:
1. `test_results.log`: 모든 테스트 상세 로그
2. `cli_verification.log`: CLI 검증 결과
3. `verification_report.html`: HTML 형식 종합 보고서
4. `coverage_report.html`: 명세서 커버리지 보고서

**보고서 내용**:
- 명세서 항목별 테스트 커버리지
- 각 테스트의 통과/실패 상태
- 상세 로그 및 검증 데이터
- CLI 검증 결과
- 미달 항목 및 개선 권장 사항

---

## 8. 일정 및 우선순위

### 8.1 전체 일정 (6주)

| 주차 | Phase | 주요 작업 | 산출물 |
|------|-------|----------|--------|
| 1주 | Phase 0 | 템플릿 작성, RFC 9421 개선 | 템플릿, RFC 9421 완료 |
| 2주 | Phase 2 | 암호화 키 관리 테스트 개선 | 키 관리 테스트 완료 |
| 3주 | Phase 3 | DID 및 블록체인 테스트 개선 | DID/블록체인 완료 |
| 4주 | Phase 4 | 메시지 처리 테스트 개선 | 메시지 처리 완료 |
| 5주 | Phase 5 | HPKE 및 세션 테스트 개선 | HPKE/세션 완료 |
| 6주 | Phase 6 | CLI 도구, 문서화, 검증 | 전체 완료 |

### 8.2 우선순위

**P0 (최우선)**:
1. RFC 9421 구현 테스트 (핵심 기능)
2. Ed25519 키 관리 (가장 많이 사용)
3. DID 생성 및 검증

**P1 (높음)**:
4. Secp256k1 키 관리 (Ethereum 호환)
5. 블록체인 통합 테스트
6. 메시지 Nonce 관리

**P2 (중간)**:
7. HPKE 암호화
8. 세션 관리
9. X25519, RSA 키 관리

**P3 (낮음)**:
10. 성능 벤치마크
11. Fuzz 테스트
12. 통합 시나리오

### 8.3 마일스톤

**M1 (2주 후)**: RFC 9421 및 기본 키 관리 완료
- 산출물: 개선된 테스트 20개, CLI 스크립트 10개, 테스트 데이터 20개

**M2 (4주 후)**: DID 및 블록체인 테스트 완료
- 산출물: 개선된 테스트 30개, CLI 스크립트 15개, 통합 검증 도구

**M3 (6주 후)**: 전체 프로젝트 완료
- 산출물: 모든 테스트 개선, 완전한 CLI 검증, 종합 보고서

---

## 9. 예상 효과

### 9.1 정량적 효과

1. **명세서 준수율**: 현재 70% → 목표 100%
2. **테스트 가시성**: 로그 출력 현재 20% → 목표 100%
3. **CLI 검증 가능성**: 현재 0% → 목표 80%
4. **재현 가능성**: 현재 40% → 목표 95%

### 9.2 정성적 효과

1. **개발 생산성 향상**
   - 테스트 실패 시 원인 파악 시간 단축
   - 디버깅을 위한 상세 정보 제공

2. **품질 보증 강화**
   - 명세서 요구사항 완전 검증
   - CLI 도구와 코드 간 일관성 확인

3. **문서화 개선**
   - 테스트 로그가 실행 가능한 문서 역할
   - CLI 사용 예시 제공

4. **유지보수성 향상**
   - 테스트 의도가 명확히 드러남
   - 새로운 개발자의 코드 이해 용이

---

## 10. 위험 요소 및 대응

### 10.1 위험 요소

| 위험 | 확률 | 영향 | 대응 방안 |
|------|------|------|----------|
| 일정 지연 | 중 | 높음 | Phase별 독립 작업, 우선순위 조정 |
| CLI 도구 미비 | 낮 | 중 | CLI 개선을 별도 작업으로 분리 |
| 테스트 데이터 크기 | 중 | 낮 | 필수 데이터만 저장, 압축 사용 |
| 명세서 변경 | 낮 | 중 | 변경 추적, 빠른 업데이트 |

### 10.2 대응 전략

1. **일정 지연 대응**
   - 각 Phase를 독립적으로 완료 가능하도록 설계
   - 우선순위 높은 항목 먼저 완료
   - 주간 진행 상황 점검 및 조정

2. **CLI 도구 미비 대응**
   - CLI 기능이 없는 경우 테스트 데이터만 먼저 저장
   - CLI 개선을 별도 이슈로 생성
   - 테스트 자동화를 우선 완료

3. **리소스 부족 대응**
   - 병렬 작업 가능한 항목 식별
   - 자동화 도구 활용
   - 코드 생성 스크립트 작성

---

## 11. 다음 단계

### 즉시 착수 가능한 작업

1. ✅ 작업 계획서 검토 및 승인
2. ⏳ 템플릿 및 헬퍼 함수 작성
3. ⏳ RFC 9421 테스트 개선 시작
4. ⏳ 첫 번째 CLI 검증 스크립트 작성

### 준비 사항

- [ ] 개발 환경 설정
- [ ] 테스트 데이터 디렉토리 구조 생성
- [ ] CLI 도구 빌드 확인
- [ ] 문서 템플릿 준비

---

## 부록

### A. 참고 문서

1. `docs/test/SPECIFICATION_VERIFICATION_MATRIX.md` - 명세서 검증 매트릭스
2. `docs/feature_list.docx` - 원본 명세서
3. `README.md` - 프로젝트 개요

### B. 도구 및 리소스

**필수 도구**:
- Go 1.21+
- jq (JSON 처리)
- bash 4.0+

**CLI 도구**:
- `sage-crypto` - 암호화 작업
- `sage-did` - DID 관리
- `sage-verify` - 검증 도구

### C. 연락처 및 지원

**프로젝트 팀**:
- 개발 리드: [연락처]
- QA 담당: [연락처]
- 문서화 담당: [연락처]

**이슈 트래킹**:
- GitHub Issues: [링크]
- 프로젝트 보드: [링크]

---

**문서 이력**:
- v1.0 (2025-10-22): 초안 작성
