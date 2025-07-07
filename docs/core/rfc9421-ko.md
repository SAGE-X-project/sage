# RFC-9421 HTTP 메시지 서명

SAGE (Secure Agent Guarantee Engine) 프로젝트를 위한 RFC-9421 HTTP 메시지 서명 구현으로, AI 에이전트 통신을 위한 안전한 HTTP 요청 서명 및 검증을 제공합니다.

## 개요

RFC-9421은 HTTP 메시지 서명을 생성, 인코딩 및 검증하는 메커니즘을 정의합니다. 이 구현을 통해 AI 에이전트는 HTTP 요청에 서명하여 메시지 무결성, 진정성을 보장하고 재전송 공격을 방지할 수 있습니다.

## 주요 기능

- **HTTP 요청 서명**: 다양한 서명 알고리즘으로 HTTP 요청 서명
- **서명 검증**: 수신된 HTTP 요청의 서명 검증
- **선택적 필드 서명**: 서명에 포함할 HTTP 구성 요소 선택
- **다중 알고리즘 지원**: Ed25519, ECDSA P-256, RSA (예정)
- **쿼리 매개변수 보호**: 쿼리 매개변수의 선택적 서명
- **타임스탬프 검증**: 재전송 공격 방지
- **메타데이터 통합**: DID 에이전트 메타데이터와의 통합

## 아키텍처

### 패키지 구조

```
core/rfc9421/
├── types.go              # 핵심 타입 정의
├── message.go            # 메시지 구조 및 빌더
├── parser.go             # Signature-Input 및 Signature 헤더 파서
├── canonicalizer.go      # HTTP 메시지 정규화
├── verifier.go           # 메시지 서명 검증
└── verifier_http.go      # HTTP 전용 검증
```

### 핵심 구성 요소

#### 1. 파서 (`parser.go`)
RFC-8941 구조화된 필드에 따라 RFC-9421 서명 헤더를 파싱:
- `ParseSignatureInput`: Signature-Input 헤더 파싱
- `ParseSignature`: base64 인코딩된 서명이 포함된 Signature 헤더 파싱

#### 2. 정규화기 (`canonicalizer.go`)
HTTP 요청에서 서명 베이스 문자열 생성:
- HTTP 서명 구성 요소 지원: `@method`, `@target-uri`, `@authority`, `@scheme`, `@request-target`, `@path`, `@query`
- 적절한 정규화와 함께 일반 HTTP 헤더 처리
- 선택적 쿼리 매개변수 서명을 위한 `@query-param` 구현

#### 3. HTTP 검증기 (`verifier_http.go`)
HTTP 요청 서명 및 검증 제공:
- `SignRequest`: 개인 키로 HTTP 요청 서명
- `VerifyRequest`: HTTP 요청 서명 검증

## 사용 예제

### HTTP 요청 서명하기

```go
package main

import (
    "crypto/ed25519"
    "crypto/rand"
    "net/http"
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
    
    // 서명 매개변수 정의
    params := &rfc9421.SignatureInputParams{
        CoveredComponents: []string{
            `"@method"`,
            `"@path"`,
            `"content-type"`,
            `"date"`,
        },
        KeyID:     "agent-key-1",
        Algorithm: "ed25519",
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
- `@status`: 응답 상태 (응답 전용)

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
4. **알고리즘 선택**: 새 구현에는 Ed25519 사용
5. **구성 요소 선택**: 중요한 구성 요소를 서명에 포함

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

## 표준 준수

이 구현은 다음을 따릅니다:
- [RFC-9421](https://datatracker.ietf.org/doc/rfc9421/): HTTP 메시지 서명
- [RFC-8941](https://datatracker.ietf.org/doc/rfc8941/): HTTP용 구조화된 필드 값
- [RFC-9110](https://datatracker.ietf.org/doc/rfc9110/): HTTP 의미론

## 향후 개선 사항

- [ ] 응답 서명 지원
- [ ] RSA-PSS 및 RSA-PKCS#1 v1.5 지원
- [ ] 서명 협상
- [ ] 성능 최적화
- [ ] 서명 검증을 위한 캐싱