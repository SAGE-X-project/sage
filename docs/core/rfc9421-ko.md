# RFC-9421 HTTP 메시지 서명

SAGE (Secure Agent Guarantee Engine) 프로젝트를 위한 RFC-9421 HTTP 메시지 서명 구현으로, AI 에이전트 통신을 위한 안전한 HTTP 요청 서명 및 검증을 제공합니다.

## 개요

RFC-9421은 HTTP 메시지 서명을 생성, 인코딩 및 검증하는 메커니즘을 정의합니다. 이 구현을 통해 AI 에이전트는 HTTP 요청에 서명하여 메시지 무결성, 진정성을 보장하고 재전송 공격을 방지할 수 있습니다.

## 주요 기능

- **HTTP 요청 서명**: 다양한 서명 알고리즘으로 HTTP 요청 서명
- **서명 검증**: 수신된 HTTP 요청의 서명 검증
- **선택적 필드 서명**: 서명에 포함할 HTTP 구성 요소 선택
- **다중 알고리즘 지원**: Ed25519, ES256K (Secp256k1), RSA-PSS-SHA256
- **동적 알고리즘 레지스트리**: 중앙집중식 암호화 알고리즘 관리
- **쿼리 매개변수 보호**: 쿼리 매개변수의 선택적 서명
- **타임스탬프 검증**: 재전송 공격 방지
- **메타데이터 통합**: DID 에이전트 메타데이터와의 통합
- **메시지 빌더**: 메시지 구성을 위한 플루언트 API

## 아키텍처

### 패키지 구조

```
core/rfc9421/
├── types.go              # 핵심 타입 정의
├── message_builder.go    # 플루언트 API를 사용한 메시지 빌더
├── parser.go             # Signature-Input 및 Signature 헤더 파서
├── canonicalizer.go      # HTTP 메시지 정규화
├── verifier.go           # 메시지 서명 검증
└── verifier_http.go      # HTTP 전용 검증
```

### 알고리즘 레지스트리 통합

RFC-9421 구현은 SAGE의 중앙집중식 암호화 알고리즘 레지스트리(`crypto` 패키지)와 통합됩니다. 지원되는 알고리즘은 동적으로 등록되고 검증됩니다:

```go
// 지원되는 알고리즘 목록 조회
algorithms := rfc9421.GetSupportedAlgorithms()
// 반환값: ["ed25519", "es256k", "rsa-pss-sha256"]

// 알고리즘 지원 여부 확인
if rfc9421.IsAlgorithmSupported("ed25519") {
    // 알고리즘이 지원됩니다
}
```

**현재 지원되는 알고리즘**:
- **ed25519**: Edwards-curve 디지털 서명 알고리즘
- **es256k**: secp256k1 곡선을 사용하는 ECDSA (이더리움 호환)
- **rsa-pss-sha256**: PSS 패딩과 SHA-256을 사용하는 RSA

**참고**: ECDSA P-256 암호화 작업은 완전히 작동하고 테스트되었지만, 아직 별도의 RFC-9421 알고리즘 식별자로 등록되지 않았습니다. 현재 모든 ECDSA 작업은 알고리즘 레지스트리에서 `es256k` (secp256k1)로 매핑됩니다. 구현 상태는 `crypto/keys/algorithms.go`를 참조하세요.

### 핵심 구성 요소

#### 1. 파서 (`parser.go`)
RFC-8941 구조화된 필드에 따라 RFC-9421 서명 헤더를 파싱:
- `ParseSignatureInput`: Signature-Input 헤더 파싱
- `ParseSignature`: base64 인코딩된 서명이 포함된 Signature 헤더 파싱
- 잘못된 헤더 및 유효하지 않은 Base64 인코딩에 대한 오류 처리

#### 2. 정규화기 (`canonicalizer.go`)
HTTP 요청에서 서명 베이스 문자열 생성:
- HTTP 서명 구성 요소 지원: `@method`, `@target-uri`, `@authority`, `@scheme`, `@request-target`, `@path`, `@query`
- 적절한 정규화와 함께 일반 HTTP 헤더 처리
- 선택적 쿼리 매개변수 서명을 위한 `@query-param` 구현
- 구성 요소 정규화 및 순서 지정

#### 3. HTTP 검증기 (`verifier_http.go`)
HTTP 요청 서명 및 검증 제공:
- `SignRequest`: 개인 키로 HTTP 요청 서명
- `VerifyRequest`: 알고리즘 검증을 통한 HTTP 요청 서명 검증
- 검증을 위한 중앙집중식 알고리즘 레지스트리와의 통합

#### 4. 메시지 빌더 (`message_builder.go`)
RFC-9421 메시지 구성을 위한 플루언트 API 제공:
- `NewMessageBuilder()`: 새 메시지 빌더 생성
- 빌더 메서드: `WithAgentDID()`, `WithMessageID()`, `WithTimestamp()` 등
- `Build()`: 기본 서명 필드로 최종 메시지 구성
- `ParseMessageFromHeaders()`: HTTP 스타일 헤더에서 메시지 파싱

#### 5. 검증기 (`verifier.go`)
핵심 검증 로직:
- `VerifyWithMetadata()`: 메타데이터 제약 조건을 사용한 서명 검증
- `ConstructSignatureBase()`: 디버깅을 위한 서명 베이스 문자열 구성
- `VerifyHTTPRequest()`: HTTP 검증을 위한 래퍼
- 레지스트리를 통한 다중 서명 알고리즘 지원

## 사용 예제

### HTTP 요청 서명하기

```go
package main

import (
    "crypto/ed25519"
    "crypto/rand"
    "net/http"
    "strings"
    "time"

    "github.com/sage-x-project/sage/core/rfc9421"
)

func main() {
    // 키 쌍 생성
    publicKey, privateKey, _ := ed25519.GenerateKey(rand.Reader)

    // HTTP 요청 생성
    req, _ := http.NewRequest("POST", "https://api.example.com/agent/action",
        strings.NewReader(`{"action": "process"}`))

    req.Header.Set("Content-Type", "application/json")
    req.Header.Set("Date", time.Now().Format(http.TimeFormat))

    // 서명 매개변수 정의 (레지스트리 알고리즘 이름 사용)
    params := &rfc9421.SignatureInputParams{
        CoveredComponents: []string{
            `"@method"`,
            `"@path"`,
            `"content-type"`,
            `"date"`,
        },
        KeyID:     "agent-key-1",
        Algorithm: "ed25519",  // 레지스트리 알고리즘 이름 사용
        Created:   time.Now().Unix(),
    }

    // 요청 서명
    verifier := rfc9421.NewHTTPVerifier()
    err := verifier.SignRequest(req, "sig1", params, privateKey)
    if err != nil {
        panic(err)
    }

    // 이제 요청에 Signature-Input 및 Signature 헤더가 포함됨
    fmt.Println("Signature-Input:", req.Header.Get("Signature-Input"))
    fmt.Println("Signature:", req.Header.Get("Signature"))
}
```

### HTTP 요청 검증하기

```go
func verifyRequest(req *http.Request, publicKey ed25519.PublicKey) error {
    verifier := rfc9421.NewHTTPVerifier()

    // 기본 옵션으로 검증 (최대 5분 유효)
    err := verifier.VerifyRequest(req, publicKey, nil)
    if err != nil {
        return fmt.Errorf("서명 검증 실패: %w", err)
    }

    return nil
}
```

### 선택적 쿼리 매개변수 서명

```go
// 특정 쿼리 매개변수만 서명
params := &rfc9421.SignatureInputParams{
    CoveredComponents: []string{
        `"@method"`,
        `"@path"`,
        `"@query-param";name="api_key"`,  // api_key 매개변수만 서명
        `"@query-param";name="action"`,   // action 매개변수만 서명
    },
    Created: time.Now().Unix(),
}

// 다른 쿼리 매개변수는 서명을 무효화하지 않고 수정 가능
```

### MessageBuilder로 메시지 구성하기

```go
// 빌더를 사용하여 메시지 생성
builder := rfc9421.NewMessageBuilder()
message := builder.
    WithAgentDID("did:sage:ethereum:0x123...").
    WithMessageID("msg-001").
    WithTimestamp(time.Now()).
    WithNonce("random-nonce-123").
    WithBody([]byte(`{"action": "process"}`)).
    WithAlgorithm(rfc9421.AlgorithmEdDSA).
    WithKeyID("agent-key-1").
    WithSignedFields("agent_did", "message_id", "timestamp", "nonce", "body").
    AddHeader("Content-Type", "application/json").
    AddMetadata("capability", "signing").
    Build()

// HTTP 헤더에서 메시지 파싱
headers := map[string]string{
    "X-Agent-DID":            "did:sage:ethereum:0x123...",
    "X-Message-ID":           "msg-001",
    "X-Timestamp":            time.Now().Format(time.RFC3339),
    "X-Nonce":                "random-nonce-123",
    "X-Signature-Algorithm":  "ed25519",
    "X-Key-ID":               "agent-key-1",
    "X-Signed-Fields":        "agent_did,message_id,timestamp",
}
body := []byte(`{"action": "process"}`)
message, err := rfc9421.ParseMessageFromHeaders(headers, body)
if err != nil {
    panic(err)
}
```

### DID와의 통합

```go
// DID 리졸버를 사용하여 검증 서비스 생성
verificationService := core.NewVerificationService(didManager)

// 메타데이터와 함께 에이전트 메시지 검증
message := &rfc9421.Message{
    AgentDID:  "did:sage:ethereum:0x123...",
    Body:      []byte("AI 응답"),
    Signature: signature,
    Algorithm: "ed25519",
}

result, err := verificationService.VerifyAgentMessage(
    ctx,
    message,
    &rfc9421.VerificationOptions{
        RequireActiveAgent: true,
        VerifyMetadata:     true,
    },
)

if result.Valid {
    fmt.Printf("에이전트로부터 메시지 검증됨: %s\n", result.AgentName)
}
```

## 고급 기능

### 메타데이터 검증

검증기는 고급 메타데이터 유효성 검사를 지원합니다:

```go
verifier := rfc9421.NewVerifier()

// 예상 메타데이터 정의
expectedMetadata := map[string]interface{}{
    "version": "1.0",
    "environment": "production",
}

// 필수 기능 정의
requiredCapabilities := []string{"signing", "verification"}

// 메타데이터와 함께 검증
result, err := verifier.VerifyWithMetadata(
    publicKey,
    message,
    expectedMetadata,
    requiredCapabilities,
    &rfc9421.VerificationOptions{
        RequireActiveAgent: true,
        VerifyMetadata:     true,
    },
)

if result.Valid {
    fmt.Println("메타데이터 제약 조건과 함께 메시지가 검증되었습니다")
}
```

### 서명 베이스 구성

디버깅 또는 사용자 정의 검증 흐름을 위해:

```go
verifier := rfc9421.NewVerifier()

// 서명 베이스 문자열 가져오기
signatureBase := verifier.ConstructSignatureBase(message)
fmt.Println("서명 베이스:", signatureBase)

// 출력 형식:
// agent_did: did:sage:ethereum:0x123...
// message_id: msg-001
// timestamp: 2025-01-15T10:30:00Z
// nonce: random-nonce-123
// body: {"action": "process"}
```

## 지원되는 HTTP 구성 요소

### 특수 구성 요소
- `@method`: HTTP 메서드 (GET, POST 등)
- `@target-uri`: 전체 대상 URI
- `@authority`: 호스트 및 포트
- `@scheme`: URI 스킴 (http/https)
- `@request-target`: 메서드 및 경로
- `@path`: URI 경로
- `@query`: 전체 쿼리 문자열
- `@query-param`: 선택적 쿼리 매개변수
- `@status`: 응답 상태 코드 (감지 구현됨, 응답 서명/검증은 아직 사용 불가)

### 헤더 구성 요소
소문자 이름을 사용하여 모든 HTTP 헤더 포함 가능:
- `date`
- `content-type`
- `content-length`
- `authorization`
- 기타

## 보안 고려사항

1. **타임스탬프 검증**: 항상 `created` 및 `expires` 타임스탬프 검증
2. **재전송 방지**: 중요한 작업에는 nonce 사용
3. **키 관리**: rotation 패키지를 사용하여 정기적으로 키 교체
4. **알고리즘 선택**: 새 구현에는 Ed25519 사용, 이더리움 호환성을 위해 ES256K 사용
5. **구성 요소 선택**: 중요한 구성 요소를 서명에 포함
6. **알고리즘 검증**: 구현은 공개 키와 알고리즘의 호환성을 자동으로 검증합니다

## 구성 옵션

### 검증 옵션

```go
opts := &rfc9421.HTTPVerificationOptions{
    // 서명 최대 유효 기간 (기본값: 5분)
    MaxAge: 10 * time.Minute,

    // 필수 서명 이름 (여러 서명이 존재하는 경우)
    SignatureName: "sig1",

    // 서명에 반드시 포함되어야 하는 구성 요소
    RequiredComponents: []string{`"@method"`, `"@path"`},
}
```

### 메시지 검증 옵션

```go
opts := &rfc9421.VerificationOptions{
    // 허용되는 최대 클럭 스큐
    MaxClockSkew: 5 * time.Minute,

    // 에이전트가 활성 상태여야 함
    RequireActiveAgent: true,

    // 메타데이터 필드 검증
    VerifyMetadata: true,

    // 필수 기능
    RequiredCapabilities: []string{"signing", "verification"},
}
```

## 테스트

구현에는 RFC-9421 테스트 벡터를 기반으로 한 포괄적인 테스트가 포함되어 있습니다:

```bash
# RFC-9421 테스트 실행
go test ./core/rfc9421/...

# 레이스 감지와 함께 실행
go test -race ./core/rfc9421/...

# 커버리지 확인
go test -cover ./core/rfc9421/...
```

### 테스트 커버리지

`rfc-9421-test.md`에 문서화된 테스트 계획의 **100% 커버리지**를 달성했습니다:

#### 단위 테스트
- ✅ **파서 테스트** (6/6 테스트 통과)
  - 기본 파싱, 다중 서명, 공백 처리
  - 오류 케이스: 잘못된 헤더, 유효하지 않은 Base64
- ✅ **정규화기 테스트** (10/10 테스트 통과)
  - HTTP 컴포넌트 (`@method`, `@path`, `@query` 등)
  - 헤더 정규화 및 공백 처리
  - 쿼리 매개변수 보호 (`@query-param`)
- ✅ **메시지 빌더 테스트** (3/3 테스트 통과)
  - 플루언트 API 구성, 헤더 파싱

#### 통합 테스트
- ✅ **종단 간 테스트** (2/2 테스트 통과)
  - Ed25519 서명 및 검증
  - ECDSA P-256 서명 및 검증
- ✅ **부정 테스트** (5/5 테스트 통과)
  - 서명 변조 감지
  - 서명된 헤더 수정 감지
  - 서명되지 않은 헤더 수정 (통과해야 함)
  - 만료 검증 (`created` + `MaxAge`, `expires`)

#### 고급 테스트
- ✅ **쿼리 매개변수 테스트** (5/5 테스트 통과)
  - 선택적 매개변수 서명 및 보호
  - 매개변수 대소문자 구분
  - 존재하지 않는 매개변수 처리
- ✅ **엣지 케이스 테스트** (3/3 테스트 통과)
  - 빈 경로, 특수 문자, 프록시 요청

**총계: 26/26 테스트 통과 (100% 커버리지)**

### 테스트 파일
- `parser_test.go` - 헤더 파싱 및 오류 처리 (6개 테스트)
- `canonicalizer_test.go` - 서명 베이스 구성 (10개 테스트)
- `verifier_test.go` - 서명 검증 로직 (다양함)
- `integration_test.go` - 종단 간 및 부정 테스트 케이스 (7개 테스트)
- `message_builder_test.go` - 메시지 구성 API (3개 테스트)

## 표준 준수

이 구현은 다음을 따릅니다:
- [RFC-9421](https://datatracker.ietf.org/doc/rfc9421/): HTTP 메시지 서명
- [RFC-8941](https://datatracker.ietf.org/doc/rfc8941/): HTTP용 구조화된 필드 값
- [RFC-9110](https://datatracker.ietf.org/doc/rfc9110/): HTTP 의미론

## 알고리즘 지원 상태

| 알고리즘 | 상태 | RFC-9421 이름 | 참고 |
|---------|------|--------------|------|
| Ed25519 | ✅ 완전 지원 | `ed25519` | 새 구현에 권장 |
| ES256K (Secp256k1) | ✅ 완전 지원 | `es256k` | 이더리움 호환 |
| RSA-PSS-SHA256 | ✅ 완전 지원 | `rsa-pss-sha256` | PSS 패딩을 사용하는 RSA |
| ECDSA P-256 | ⚠️ 암호화만 | 해당 없음 | 암호화 작업은 작동하나 별도 알고리즘으로 등록되지 않음 |
| RSA-PKCS#1 v1.5 | ❌ 미지원 | `rsa-v1_5-sha256` | 레거시 RSA (계획됨) |

## 구현 상태 및 로드맵

### 완료된 기능
- ✅ **RSA-PSS-SHA256 지원** - 알고리즘 레지스트리에 완전히 구현 및 등록됨
- ✅ **핵심 RFC-9421 준수** - Ed25519, ES256K, RSA-PSS-SHA256를 사용한 HTTP 요청 서명
- ✅ **포괄적인 테스트 커버리지** - 문서화된 테스트 계획의 100% 커버리지 (26/26 테스트 통과)

### 부분 구현
- ⚠️ **응답 서명 지원** - `@status` 컴포넌트 감지 구현됨, 서명/검증 메서드는 대기 중
- ⚠️ **ECDSA P-256 지원** - 암호화 작업이 완전히 작동하고 테스트됨, 별도 식별자로서의 알고리즘 등록은 대기 중

### 계획된 개선 사항
- **RSA-PKCS#1 v1.5 지원** - 레거시 RSA 알고리즘 (`rsa-v1_5-sha256`)
- **ECDSA P-256 등록 완료** - secp256k1과 별도의 알고리즘 (`ecdsa-p256-sha256`)으로 등록
- **응답 서명 메서드** - HTTP 응답을 위한 `SignResponse()` 및 `VerifyResponse()`
- **서명 협상** - Accept-Signature 헤더, 알고리즘 기능 광고
- **성능 최적화** - 버퍼 풀링, 고루틴 풀, 사전 할당 전략
- **캐싱 레이어** - 공개키 캐시, DID 해석 캐시, 파싱된 서명 캐시

### 기술 부채
- 알고리즘 레지스트리에서 ECDSA P-256 등록 완료 (`crypto/keys/algorithms.go:58-60` 참조)
- `@status` 컴포넌트를 위한 응답 정규화 구현
