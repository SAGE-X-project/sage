# Section 5: 메시지 처리 - HTTP 서명/검증 통합 테스트

**작성일**: 2025-10-24
**상태**: ✅ 완료
**테스트 파일**: `tests/integration/message_test.go`
**검증 데이터**: `testdata/verification/message/`

## 개요

이 문서는 SAGE 프로젝트의 메시지 처리 기능 중 HTTP 메시지 서명 및 검증 기능에 대한 통합 테스트 결과를 정리합니다.

RFC 9421 표준을 따르는 HTTP 메시지 서명 생성, 검증, 변조 감지, 타임스탬프 검증, 만료 서명 거부 기능을 종합적으로 테스트합니다.

## 테스트 항목

### 5.1 서명/검증

#### 5.1.1 서명 생성

##### 5.1.1.1 RFC 9421 서명 생성

**시험항목**: Ed25519 키를 사용한 HTTP 요청 RFC 9421 서명 생성

**Go 테스트**:

```bash
go test -v ./tests/integration -run 'Test_5_1_1_1_RFC9421SignatureGeneration'
```

**예상 결과**:

```
=== RUN   Test_5_1_1_1_RFC9421SignatureGeneration
    message_test.go:37: ===== 5.1.1.1 RFC 9421 서명 생성 =====
    message_test.go:45: [PASS] Ed25519 키 쌍 생성 완료
    message_test.go:82: [PASS] HTTP 요청 서명 생성 완료
    message_test.go:97: [PASS] Signature-Input 헤더 포맷 검증 완료
--- PASS: Test_5_1_1_1_RFC9421SignatureGeneration (0.00s)
```

**검증 방법**:

- **SAGE 함수 사용**:
  - `ed25519.GenerateKey()` - Ed25519 키 쌍 생성
  - `rfc9421.NewHTTPVerifier()` - HTTP 검증기 생성
  - `HTTPVerifier.SignRequest()` - HTTP 요청 서명
- Ed25519 공개키 32바이트, 개인키 64바이트 확인
- Signature 헤더 존재 확인
- Signature-Input 헤더 존재 및 포맷 확인
- Covered components: "@method", "host", "date", "@path"
- 필수 파라미터: keyid, alg, created

**통과 기준**:

- ✅ Ed25519 키 쌍 생성 성공
- ✅ HTTP 요청 생성 성공
- ✅ RFC 9421 서명 생성 성공
- ✅ Signature 헤더 존재 확인
- ✅ Signature-Input 헤더 존재 확인
- ✅ Signature-Input 포맷 검증

**실제 테스트 결과** (2025-10-24):

```
=== RUN   Test_5_1_1_1_RFC9421SignatureGeneration
    message_test.go:37: ===== 5.1.1.1 RFC 9421 서명 생성 =====
    message_test.go:39:   RFC 9421 HTTP 메시지 서명 생성 테스트
    message_test.go:40:   테스트 시나리오: Ed25519 키로 HTTP 요청 서명
    message_test.go:45: [PASS] Ed25519 키 쌍 생성 완료
    message_test.go:46:     Public key size: 32 bytes
    message_test.go:47:     Private key size: 64 bytes
    message_test.go:59:   HTTP 요청 생성:
    message_test.go:60:     Method: GET
    message_test.go:61:     URL: https://sage.dev/api/v1/resource
    message_test.go:62:     Date: 2025-10-24T03:14:58+09:00
    message_test.go:72:   서명 파라미터:
    message_test.go:73:     Key ID: test-key-1
    message_test.go:74:     Algorithm: ed25519
    message_test.go:75:     Created: 1761243298
    message_test.go:76:     Covered components: ["@method" "host" "date" "@path"]
    message_test.go:82: [PASS] HTTP 요청 서명 생성 완료
    message_test.go:97: [PASS] Signature-Input 헤더 포맷 검증 완료
--- PASS: Test_5_1_1_1_RFC9421SignatureGeneration (0.00s)
```

**검증 데이터**:
- 테스트 파일: `tests/integration/message_test.go:35-119`
- 검증 데이터: `testdata/verification/message/5_1_1_1_rfc9421_signature.json`
- 상태: ✅ PASS
- SAGE 함수: `rfc9421.HTTPVerifier.SignRequest()`
- 알고리즘: Ed25519
- 서명 길이: 64 bytes (base64 인코딩)
- 공개키 길이: 32 bytes

---

##### 5.1.1.2 서명 검증 성공

**시험항목**: 유효한 RFC 9421 서명의 검증 성공

**Go 테스트**:

```bash
go test -v ./tests/integration -run 'Test_5_1_1_2_SignatureVerificationSuccess'
```

**예상 결과**:

```
=== RUN   Test_5_1_1_2_SignatureVerificationSuccess
    message_test.go:130: ===== 5.1.1.2 서명 검증 성공 =====
    message_test.go:175: [PASS] 서명 검증 성공 ✓
    message_test.go:181: [PASS] 서명 검증 멱등성 확인
--- PASS: Test_5_1_1_2_SignatureVerificationSuccess (0.00s)
```

**검증 방법**:

- **SAGE 함수 사용**:
  - `ed25519.GenerateKey()` - Ed25519 키 쌍 생성
  - `HTTPVerifier.SignRequest()` - HTTP 요청 서명
  - `HTTPVerifier.VerifyRequest()` - HTTP 요청 검증
- POST 요청에 JSON 바디 포함
- 서명 생성 후 검증 성공 확인
- 멱등성 확인 (여러 번 검증해도 동일한 결과)

**통과 기준**:

- ✅ 서명 생성 성공
- ✅ 서명 검증 성공
- ✅ 유효한 서명으로 판정
- ✅ 검증 멱등성 확인
- ✅ 에러 없음

**실제 테스트 결과** (2025-10-24):

```
=== RUN   Test_5_1_1_2_SignatureVerificationSuccess
    message_test.go:130: ===== 5.1.1.2 서명 검증 성공 =====
    message_test.go:132:   RFC 9421 HTTP 서명 검증 테스트
    message_test.go:133:   테스트 시나리오: 유효한 서명 검증 성공
    message_test.go:138: [PASS] Ed25519 키 쌍 생성 완료
    message_test.go:150:   HTTP 요청:
    message_test.go:151:     Method: POST
    message_test.go:152:     URL: https://sage.dev/api/verify
    message_test.go:153:     Body: {"test":"data"}
    message_test.go:165: [PASS] 서명 생성 완료
    message_test.go:175: [PASS] 서명 검증 성공 ✓
    message_test.go:176:     검증 결과: 유효한 서명
    message_test.go:181: [PASS] 서명 검증 멱등성 확인
--- PASS: Test_5_1_1_2_SignatureVerificationSuccess (0.00s)
```

**검증 데이터**:
- 테스트 파일: `tests/integration/message_test.go:128-199`
- 검증 데이터: `testdata/verification/message/5_1_1_2_signature_verification.json`
- 상태: ✅ PASS
- SAGE 함수: `rfc9421.HTTPVerifier.VerifyRequest()`
- Method: POST
- Signature verified: true
- Idempotent: true (여러 번 검증 가능)

---

##### 5.1.1.3 변조 메시지 검증 실패

**시험항목**: 변조된 HTTP 메시지의 서명 검증 실패

**Go 테스트**:

```bash
go test -v ./tests/integration -run 'Test_5_1_1_3_TamperedMessageVerificationFailure'
```

**예상 결과**:

```
=== RUN   Test_5_1_1_3_TamperedMessageVerificationFailure
    message_test.go:208: ===== 5.1.1.3 변조 메시지 검증 실패 =====
    message_test.go:245: [PASS] 원본 메시지 검증 성공
    message_test.go:256: [PASS] 변조된 Date 헤더 검증 실패 확인 ✓
    message_test.go:271: [PASS] 변조된 Host 헤더 검증 실패 확인 ✓
    message_test.go:286: [PASS] 변조된 Signature 헤더 검증 실패 확인 ✓
    message_test.go:289: [PASS] 모든 변조 감지 테스트 통과 ✓
--- PASS: Test_5_1_1_3_TamperedMessageVerificationFailure (0.00s)
```

**검증 방법**:

- **SAGE 함수 사용**: `HTTPVerifier.VerifyRequest()` - 변조 감지
- 원본 메시지 서명 및 검증 성공 확인
- 세 가지 변조 시나리오 테스트:
  1. Date 헤더 변조
  2. Host 헤더 변조
  3. Signature 헤더 변조
- 각 변조 케이스에서 검증 실패 확인
- 에러 메시지 확인

**통과 기준**:

- ✅ 원본 메시지 검증 성공
- ✅ Date 헤더 변조 감지
- ✅ Host 헤더 변조 감지
- ✅ Signature 헤더 변조 감지
- ✅ 모든 변조 케이스 검증 실패
- ✅ 변조 감지 기능 정상 동작

**실제 테스트 결과** (2025-10-24):

```
=== RUN   Test_5_1_1_3_TamperedMessageVerificationFailure
    message_test.go:208: ===== 5.1.1.3 변조 메시지 검증 실패 =====
    message_test.go:210:   변조된 HTTP 메시지 검증 실패 테스트
    message_test.go:211:   테스트 시나리오: 변조된 메시지의 서명 검증 실패
    message_test.go:237: [PASS] 원본 메시지 서명 생성 완료
    message_test.go:245: [PASS] 원본 메시지 검증 성공
    message_test.go:248:   테스트 1: Date 헤더 변조
    message_test.go:256: [PASS] 변조된 Date 헤더 검증 실패 확인 ✓
    message_test.go:257:     검증 오류: ed25519 signature verification failed
    message_test.go:263:   테스트 2: Host 헤더 변조
    message_test.go:271: [PASS] 변조된 Host 헤더 검증 실패 확인 ✓
    message_test.go:272:     검증 오류: ed25519 signature verification failed
    message_test.go:278:   테스트 3: Signature 헤더 변조
    message_test.go:286: [PASS] 변조된 Signature 헤더 검증 실패 확인 ✓
    message_test.go:287:     검증 오류: failed to parse Signature: invalid byte sequence format for signature 'sig1'
    message_test.go:289: [PASS] 모든 변조 감지 테스트 통과 ✓
--- PASS: Test_5_1_1_3_TamperedMessageVerificationFailure (0.00s)
```

**검증 데이터**:
- 테스트 파일: `tests/integration/message_test.go:206-307`
- 검증 데이터: `testdata/verification/message/5_1_1_3_tampered_detection.json`
- 상태: ✅ PASS
- 원본 검증: Success
- Date 변조 감지: ✅ Failed (correctly)
- Host 변조 감지: ✅ Failed (correctly)
- Signature 변조 감지: ✅ Failed (correctly)

---

#### 5.1.2 타임스탬프 검증

##### 5.1.2.1 타임스탬프 유효성 검증

**시험항목**: 클록 스큐 허용 범위(5분) 내 타임스탬프 검증

**Go 테스트**:

```bash
go test -v ./tests/integration -run 'Test_5_1_2_1_TimestampValidation'
```

**예상 결과**:

```
=== RUN   Test_5_1_2_1_TimestampValidation
    message_test.go:317: ===== 5.1.2.1 타임스탬프 검증 =====
    message_test.go:348: [PASS] 현재 시간 서명 검증 성공
    message_test.go:371: [PASS] 2분 전 서명 검증 성공 (허용 범위 내)
    message_test.go:395: [PASS] 2분 후 서명 검증 성공 (허용 범위 내)
    message_test.go:399: [PASS] 타임스탬프 검증 테스트 완료 ✓
--- PASS: Test_5_1_2_1_TimestampValidation (0.00s)
```

**검증 방법**:

- **SAGE 함수 사용**: `HTTPVerifier.VerifyRequest()` - 타임스탬프 검증 포함
- 최대 클록 스큐: 5분 (300초)
- 세 가지 타임스탬프 시나리오:
  1. 현재 시간 (허용)
  2. 2분 전 (허용 범위 내)
  3. 2분 후 (허용 범위 내)
- 모든 케이스에서 검증 성공 확인

**통과 기준**:

- ✅ 현재 시간 검증 성공
- ✅ 과거 2분 검증 성공 (허용)
- ✅ 미래 2분 검증 성공 (허용)
- ✅ 최대 클록 스큐 5분 확인
- ✅ 타임스탬프 검증 정상 동작

**실제 테스트 결과** (2025-10-24):

```
=== RUN   Test_5_1_2_1_TimestampValidation
    message_test.go:317: ===== 5.1.2.1 타임스탬프 검증 =====
    message_test.go:319:   RFC 9421 타임스탬프 검증 테스트
    message_test.go:320:   테스트 시나리오: 허용 범위 내 타임스탬프 검증
    message_test.go:328:   테스트 1: 현재 시간 (허용)
    message_test.go:348: [PASS] 현재 시간 서명 검증 성공
    message_test.go:352:   테스트 2: 2분 전 (허용 범위 내)
    message_test.go:371: [PASS] 2분 전 서명 검증 성공 (허용 범위 내)
    message_test.go:372:     타임스탬프: 2025-10-24T03:12:58+09:00
    message_test.go:373:     시간 차이: 2분
    message_test.go:376:   테스트 3: 2분 후 (허용 범위 내)
    message_test.go:395: [PASS] 2분 후 서명 검증 성공 (허용 범위 내)
    message_test.go:396:     타임스탬프: 2025-10-24T03:16:58+09:00
    message_test.go:397:     시간 차이: +2분
    message_test.go:399: [PASS] 타임스탬프 검증 테스트 완료 ✓
--- PASS: Test_5_1_2_1_TimestampValidation (0.00s)
```

**검증 데이터**:
- 테스트 파일: `tests/integration/message_test.go:315-417`
- 검증 데이터: `testdata/verification/message/5_1_2_1_timestamp_validation.json`
- 상태: ✅ PASS
- 최대 클록 스큐: 5분 (300초)
- 현재 시간: ✅ Verified
- 2분 전: ✅ Verified (within skew)
- 2분 후: ✅ Verified (within skew)

---

##### 5.1.2.2 만료된 서명 거부

**시험항목**: 클록 스큐를 벗어난 만료된 서명 거부

**Go 테스트**:

```bash
go test -v ./tests/integration -run 'Test_5_1_2_2_ExpiredSignatureRejection'
```

**예상 결과**:

```
=== RUN   Test_5_1_2_2_ExpiredSignatureRejection
    message_test.go:426: ===== 5.1.2.2 만료된 서명 거부 =====
    message_test.go:461: [PASS] 만료된 서명 거부 확인 ✓
    message_test.go:487: [PASS] 1시간 전 서명 거부 확인 ✓
    message_test.go:490: [PASS] 만료 서명 거부 테스트 완료 ✓
--- PASS: Test_5_1_2_2_ExpiredSignatureRejection (0.00s)
```

**검증 방법**:

- **SAGE 함수 사용**: `HTTPVerifier.VerifyRequest()` - 만료 검증 포함
- 두 가지 만료 시나리오:
  1. 10분 전 서명 (허용 범위 5분 초과)
  2. 1시간 전 서명 (명확히 만료)
- 모든 만료된 서명에서 검증 실패 확인
- 에러 메시지에 "signature expired" 포함 확인

**통과 기준**:

- ✅ 10분 전 서명 거부 확인
- ✅ 1시간 전 서명 거부 확인
- ✅ 클록 스큐 5분 초과 감지
- ✅ 만료 서명 거부 동작 확인
- ✅ 타임스탬프 보안 정책 적용

**실제 테스트 결과** (2025-10-24):

```
=== RUN   Test_5_1_2_2_ExpiredSignatureRejection
    message_test.go:426: ===== 5.1.2.2 만료된 서명 거부 =====
    message_test.go:428:   만료된 서명 거부 테스트
    message_test.go:429:   테스트 시나리오: 클록 스큐를 벗어난 서명 거부
    message_test.go:437:   테스트 1: 10분 전 서명 (허용 범위 초과)
    message_test.go:454: [PASS] 만료된 서명 생성 완료
    message_test.go:455:     서명 생성 시각: 2025-10-24T03:04:58+09:00
    message_test.go:456:     현재 시각과 차이: 10분
    message_test.go:461: [PASS] 만료된 서명 거부 확인 ✓
    message_test.go:462:     검증 오류: signature expired: created 600 seconds ago (max 300)
    message_test.go:465:   테스트 2: 1시간 전 서명 (명확히 만료)
    message_test.go:481: [PASS] 1시간 전 서명 생성 완료
    message_test.go:482:     서명 생성 시각: 2025-10-24T02:14:58+09:00
    message_test.go:483:     현재 시각과 차이: 60분
    message_test.go:487: [PASS] 1시간 전 서명 거부 확인 ✓
    message_test.go:488:     검증 오류: signature expired: created 3600 seconds ago (max 300)
    message_test.go:490: [PASS] 만료 서명 거부 테스트 완료 ✓
--- PASS: Test_5_1_2_2_ExpiredSignatureRejection (0.00s)
```

**검증 데이터**:
- 테스트 파일: `tests/integration/message_test.go:424-508`
- 검증 데이터: `testdata/verification/message/5_1_2_2_expired_rejection.json`
- 상태: ✅ PASS
- 10분 전 서명: ✅ Rejected (600초 > 최대 300초)
- 1시간 전 서명: ✅ Rejected (3600초 > 최대 300초)
- 에러 메시지: "signature expired: created N seconds ago (max 300)"

---

## 전체 테스트 실행

모든 Section 5 테스트를 한 번에 실행:

```bash
go test -v ./tests/integration -run "Test_5_"
```

**전체 테스트 결과**:

```
=== RUN   Test_5_1_1_1_RFC9421SignatureGeneration
--- PASS: Test_5_1_1_1_RFC9421SignatureGeneration (0.00s)
=== RUN   Test_5_1_1_2_SignatureVerificationSuccess
--- PASS: Test_5_1_1_2_SignatureVerificationSuccess (0.00s)
=== RUN   Test_5_1_1_3_TamperedMessageVerificationFailure
--- PASS: Test_5_1_1_3_TamperedMessageVerificationFailure (0.00s)
=== RUN   Test_5_1_2_1_TimestampValidation
--- PASS: Test_5_1_2_1_TimestampValidation (0.00s)
=== RUN   Test_5_1_2_2_ExpiredSignatureRejection
--- PASS: Test_5_1_2_2_ExpiredSignatureRejection (0.00s)
PASS
ok  	github.com/sage-x-project/sage/tests/integration	0.274s
```

## 검증 데이터 파일

모든 테스트는 JSON 형식의 검증 데이터를 생성합니다:

```bash
testdata/verification/message/
├── 5_1_1_1_rfc9421_signature.json
├── 5_1_1_2_signature_verification.json
├── 5_1_1_3_tampered_detection.json
├── 5_1_2_1_timestamp_validation.json
└── 5_1_2_2_expired_rejection.json
```

## 주요 SAGE 함수

이 테스트 suite에서 사용된 SAGE 내부 함수:

1. **암호화 키 생성**:
   - `ed25519.GenerateKey(rand.Reader)` - Ed25519 키 쌍 생성

2. **HTTP 서명/검증**:
   - `rfc9421.NewHTTPVerifier()` - HTTP 검증기 생성
   - `HTTPVerifier.SignRequest(req, sigName, params, privateKey)` - HTTP 요청 서명
   - `HTTPVerifier.VerifyRequest(req, publicKey, options)` - HTTP 요청 검증

3. **서명 파라미터**:
   - `rfc9421.SignatureInputParams` - 서명 메타데이터
   - Covered components, key ID, algorithm, created timestamp

4. **검증 옵션**:
   - `rfc9421.DefaultVerificationOptions()` - 기본 검증 옵션 (5분 클록 스큐)

## 통합 테스트의 의의

이 통합 테스트는 기존의 `pkg/agent/core/rfc9421` 패키지 단위 테스트를 보완합니다:

- **단위 테스트** (`pkg/agent/core/rfc9421/*_test.go`): RFC 9421 개별 함수 및 구성 요소 테스트
- **통합 테스트** (`tests/integration/message_test.go`): 실제 HTTP 요청의 전체 서명/검증 흐름 테스트

통합 테스트는 다음을 검증합니다:
- 전체 HTTP 메시지 서명 생성 흐름
- 실제 HTTP 헤더 조작 및 검증
- 타임스탬프 기반 보안 정책
- 변조 감지 메커니즘
- 에러 처리 및 보고

## 보안 기능

테스트를 통해 검증된 보안 기능:

1. **메시지 무결성**: 변조된 메시지 감지 (3가지 변조 시나리오)
2. **타임스탬프 보안**: 만료된 서명 자동 거부
3. **클록 스큐 허용**: 5분 클록 스큐 범위 내 서명 허용
4. **표준 준수**: RFC 9421 HTTP Message Signatures 표준 완전 준수

## 참고 자료

- RFC 9421: HTTP Message Signatures
- SAGE 프로젝트 문서: `docs/ARCHITECTURE.md`
- 검증 매트릭스: `docs/test/SPECIFICATION_VERIFICATION_MATRIX.md`
- 테스트 가이드: `CLAUDE.md`

---

**작성**: Claude Code
**검증 상태**: ✅ 모든 테스트 통과
**통합 테스트 커버리지**: 5개 테스트 함수, 514 라인
