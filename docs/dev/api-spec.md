# SAGE API 명세서

> **참고**: 이 문서는 SAGE 프로젝트의 전체 API 설계를 포함합니다.  
> 현재 구현된 부분과 향후 계획을 구분하여 표시합니다.

## 구현 상태

| 컴포넌트 | 상태 | 설명 |
|---------|------|------|
| **Core Modules** | ✅ 구현 완료 | crypto, did, rfc9421 모듈 |
| **CLI Tools** | ✅ 구현 완료 | sage-crypto, sage-did |
| **Agent SDK (Go)** | 📋 계획됨 | 별도 프로젝트로 구현 예정 |
| **Agent SDK (TypeScript)** | 📋 계획됨 | 별도 프로젝트로 구현 예정 |
| **Gateway REST API** | 📋 계획됨 | 향후 구현 예정 |
| **HTTP Server Integration** | 📋 계획됨 | 향후 구현 예정 |

## 목차

- [1. 현재 구현된 API](#1-현재-구현된-api)
  - [1.1 Crypto Module](#11-crypto-module)
  - [1.2 DID Module](#12-did-module)
  - [1.3 RFC9421 Module](#13-rfc9421-module)
- [2. 향후 구현 예정 API](#2-향후-구현-예정-api)
  - [2.1 Agent SDK API](#21-agent-sdk-api)
  - [2.2 Gateway REST API](#22-gateway-rest-api)
- [3. 메시지 형식](#3-메시지-형식)
- [4. 에러 코드](#4-에러-코드)

## 1. 현재 구현된 API

### 1.1 Crypto Module

```go
package crypto

// KeyPair 관리
type KeyPair interface {
    Generate(keyType string) error
    Sign(message []byte) ([]byte, error)
    Verify(message []byte, signature []byte) bool
    Export(format string) ([]byte, error)
}

// KeyStorage 인터페이스
type KeyStorage interface {
    Store(keyID string, keyPair KeyPair) error
    Load(keyID string) (KeyPair, error)
    List() ([]string, error)
    Delete(keyID string) error
}

// 지원 키 타입: ed25519, secp256k1
// 지원 형식: JWK, PEM
```

### 1.2 DID Module

```go
package did

// DID Manager
type Manager interface {
    Register(request RegistrationRequest) (*RegistrationResult, error)
    Resolve(did string) (*DIDDocument, error)
    Update(did string, updates map[string]interface{}) error
    Deactivate(did string) error
}

// RegistrationRequest 구조체
type RegistrationRequest struct {
    Name         string
    Endpoint     string
    PublicKey    []byte
    Capabilities map[string]interface{}
    Chain        string // ethereum, solana
}
```

### 1.3 RFC9421 Module

```go
package rfc9421

// HTTP 메시지 서명
type MessageSigner interface {
    SignRequest(req *http.Request, keyID string) error
    VerifyRequest(req *http.Request) error
}

// Canonicalizer - 메시지 정규화
type Canonicalizer interface {
    Canonicalize(components []string, req *http.Request) (string, error)
}
```

## 2. 향후 구현 예정 API

> **참고**: 아래 API들은 별도 프로젝트로 구현 예정이며,  
> 현재 문서는 향후 개발 시 참조용입니다.

### 2.1 Agent SDK API

#### 2.1.1 Go SDK

#### Agent 생성 및 초기화

```go
package sage

// Agent는 SAGE 에이전트의 핵심 인터페이스입니다
type Agent interface {
    // 서명된 메시지 생성
    CreateMessage(path string, headers map[string]string, body []byte) (*SignedMessage, error)
    
    // 메시지 서명 검증
    VerifyMessage(msg *SignedMessage, senderDID string) error
    
    // 메시지 전송
    SendRequest(to string, msg *SignedMessage) (*Response, error)
    
    // 서버 시작 (수신 모드)
    StartServer(port string) error
}

// NewAgent는 새로운 Agent 인스턴스를 생성합니다
func NewAgent(config AgentConfig) (Agent, error)

// AgentConfig는 Agent 설정을 정의합니다
type AgentConfig struct {
    DID        string          // Agent의 DID
    PrivateKey []byte          // 서명용 개인키
    Resolver   Resolver        // DID Resolver
    Transport  TransportConfig // 네트워크 설정
}
```

#### DID Resolver

```go
// Resolver는 DID를 해석하는 인터페이스입니다
type Resolver interface {
    // DID로부터 DID Document 조회
    Resolve(did string) (*DIDDocument, error)
    
    // 컨텍스트와 함께 조회 (타임아웃 지원)
    ResolveWithContext(ctx context.Context, did string) (*DIDDocument, error)
}

// DIDDocument는 W3C DID 문서 구조입니다
type DIDDocument struct {
    ID                 string               `json:"id"`
    VerificationMethod []VerificationMethod `json:"verificationMethod"`
    Authentication     []string             `json:"authentication,omitempty"`
    Service            []Service            `json:"service,omitempty"`
}

// VerificationMethod는 공개키 정보를 담습니다
type VerificationMethod struct {
    ID                 string `json:"id"`
    Type               string `json:"type"`
    Controller         string `json:"controller"`
    PublicKeyMultibase string `json:"publicKeyMultibase,omitempty"`
}
```

#### 메시지 서명

```go
// SignedMessage는 RFC 9421 서명된 메시지입니다
type SignedMessage struct {
    Path           string            `json:"path"`
    Headers        map[string]string `json:"headers"`
    Body           []byte            `json:"body"`
    Signature      []byte            `json:"signature"`
    SignatureInput string            `json:"signatureInput"`
    SenderDID      string            `json:"senderDID"`
}

// Signer는 메시지 서명 인터페이스입니다
type Signer interface {
    // 데이터 서명
    Sign(data []byte, privateKey []byte) ([]byte, error)
    
    // 서명 검증
    Verify(data []byte, signature []byte, publicKey []byte) bool
}
```

#### 사용 예시

```go
import (
    "github.com/sage/sdk"
    "time"
)

func main() {
    // Agent 생성
    agent, err := sdk.NewAgent(sdk.AgentConfig{
        DID:        "did:ethr:0xabc123",
        PrivateKey: loadPrivateKey(),
        Resolver:   sdk.NewHTTPResolver("https://resolver.sage.ai"),
    })
    
    // 메시지 생성 및 서명
    headers := map[string]string{
        "date": time.Now().UTC().Format(time.RFC1123),
        "host": "agent-b.ai",
    }
    
    msg, err := agent.CreateMessage("/task", headers, []byte(`{"action":"translate"}`))
    if err != nil {
        log.Fatal(err)
    }
    
    // 메시지 전송
    resp, err := agent.SendRequest("https://agent-b.ai/task", msg)
    if err != nil {
        log.Fatal(err)
    }
}
```

#### 2.1.2 TypeScript SDK

#### Agent 클래스

```typescript
// Agent 생성 및 초기화
export class Agent {
    constructor(config: AgentConfig);
    
    // 서명된 메시지 생성
    async createMessage(
        path: string, 
        headers: Record<string, string>, 
        body: Uint8Array
    ): Promise<SignedMessage>;
    
    // 메시지 검증
    async verifyMessage(
        message: SignedMessage, 
        senderDID: string
    ): Promise<boolean>;
    
    // 메시지 전송
    async sendRequest(
        to: string, 
        message: SignedMessage
    ): Promise<Response>;
}

// 설정 타입
export interface AgentConfig {
    did: string;
    privateKey: Uint8Array;
    resolver?: Resolver;
    transport?: TransportConfig;
}
```

#### TypeScript 타입 정의

```typescript
// 서명된 메시지
export interface SignedMessage {
    path: string;
    headers: Record<string, string>;
    body: Uint8Array;
    signature: Uint8Array;
    signatureInput: string;
    senderDID: string;
}

// DID Document
export interface DIDDocument {
    id: string;
    verificationMethod: VerificationMethod[];
    authentication?: string[];
    service?: Service[];
}

// 검증 메소드
export interface VerificationMethod {
    id: string;
    type: string;
    controller: string;
    publicKeyMultibase?: string;
}
```

#### 사용 예시

```typescript
import { Agent } from '@sage/sdk';

async function main() {
    // Agent 생성
    const agent = new Agent({
        did: "did:ethr:0xabc123",
        privateKey: await loadPrivateKey()
    });
    
    // 메시지 생성 및 서명
    const message = await agent.createMessage(
        "/task",
        {
            date: new Date().toUTCString(),
            host: "agent-b.ai"
        },
        new TextEncoder().encode('{"action":"translate"}')
    );
    
    // 메시지 전송
    const response = await agent.sendRequest(
        "https://agent-b.ai/task",
        message
    );
}
```

### 2.2 Gateway REST API

> **참고**: Gateway REST API는 향후 구현 예정입니다.  
> 아래 명세는 설계 참조용입니다.

#### 2.2.1 기본 정보

```yaml
openapi: 3.1.0
info:
  title: SAGE Gateway API
  version: 1.0.0
  description: RFC 9421 기반 메시지 라우팅 및 검증 Gateway

servers:
  - url: https://gateway.sage.ai
    description: Production server
  - url: http://localhost:8080
    description: Local development
```

### 2.2 엔드포인트

#### POST /relay

**설명**: 서명된 메시지를 수신하여 대상 에이전트로 라우팅

**요청**:
```http
POST /relay HTTP/1.1
Content-Type: application/json
Signature-Input: sig1=("@method" "@path" "host" "date");alg="ed25519";keyid="did:ethr:0xabc#key1"
Signature: sig1=:MEUCIQDkjN...=:

{
  "headers": {
    "host": "agent-b.ai",
    "date": "Tue, 24 Jun 2025 13:00:00 GMT"
  },
  "body": {
    "task": "translate",
    "content": "Hello world"
  },
  "signatureInput": "sig1=(\"@method\" \"@path\" \"host\" \"date\");alg=\"ed25519\";keyid=\"did:ethr:0xabc#key1\"",
  "signature": "MEUCIQDkjN...",
  "senderDID": "did:ethr:0xabc123",
  "receiverDID": "did:ethr:0xdef456"
}
```

**응답**:
```json
{
  "status": "accepted",
  "responseID": "550e8400-e29b-41d4-a716-446655440000",
  "timestamp": "2025-06-24T13:00:05Z"
}
```

#### GET /did/{did}

**설명**: DID Document 조회

**요청**:
```http
GET /did/did:ethr:0xabc123 HTTP/1.1
Accept: application/json
```

**응답**:
```json
{
  "id": "did:ethr:0xabc123",
  "verificationMethod": [
    {
      "id": "did:ethr:0xabc123#key-1",
      "type": "Ed25519VerificationKey2020",
      "controller": "did:ethr:0xabc123",
      "publicKeyMultibase": "z6MkhaXgBZD..."
    }
  ],
  "authentication": ["did:ethr:0xabc123#key-1"]
}
```

#### GET /health

**설명**: 서비스 상태 확인

**응답**:
```json
{
  "status": "healthy",
  "version": "1.0.0",
  "uptime": 3600,
  "components": {
    "resolver": "healthy",
    "cache": "healthy",
    "blockchain": "healthy"
  }
}
```

## 3. 메시지 형식

### 3.1 RFC 9421 서명 형식

```http
POST /invoke HTTP/1.1
Host: agent-b.ai
Date: Tue, 24 Jun 2025 13:00:00 GMT
Content-Type: application/json
Content-Digest: sha-256=:X48E9q...=:
Signature-Input: sig1=("@method" "@path" "host" "date" "content-digest");alg="ed25519";keyid="did:ethr:0xabc#key1";created=1719234000
Signature: sig1=:MEYCIQDTlI...=:

{
  "request": {
    "method": "translate",
    "params": {
      "text": "Hello world",
      "to": "ko"
    }
  }
}
```

### 3.2 서명 입력 구성

```
Signature-Input: sig1=(
  "@method"          # HTTP 메소드 (POST, GET 등)
  "@path"            # 요청 경로
  "host"             # Host 헤더
  "date"             # Date 헤더
  "content-digest"   # 본문 해시 (선택)
);
alg="ed25519";       # 서명 알고리즘
keyid="did:...#key1"; # 서명 키 ID
created=1719234000   # 서명 생성 시간 (Unix timestamp)
```

### 3.3 Content-Digest 계산

```
Content-Digest: sha-256=:<base64(sha256(body))>:
```

## 4. 에러 코드

### 4.1 HTTP 상태 코드

| 코드 | 설명 | 상황 |
|------|------|------|
| 200 | OK | 요청 성공 |
| 400 | Bad Request | 잘못된 요청 형식 |
| 401 | Unauthorized | 서명 검증 실패 |
| 403 | Forbidden | 정책에 의해 거부됨 |
| 404 | Not Found | 리소스 없음 |
| 422 | Unprocessable Entity | DID 해석 실패 |
| 429 | Too Many Requests | Rate limit 초과 |
| 500 | Internal Server Error | 서버 오류 |

### 4.2 에러 응답 형식

```json
{
  "error": {
    "code": "invalid_signature",
    "message": "Signature verification failed",
    "details": {
      "did": "did:ethr:0xabc123",
      "reason": "public key mismatch"
    }
  },
  "timestamp": "2025-06-24T13:00:05Z",
  "traceId": "550e8400-e29b-41d4-a716-446655440000"
}
```

### 4.3 에러 코드 목록

| 코드 | 설명 | HTTP 상태 |
|------|------|-----------|
| `invalid_signature` | 서명 검증 실패 | 401 |
| `did_not_found` | DID를 찾을 수 없음 | 422 |
| `invalid_did_format` | 잘못된 DID 형식 | 400 |
| `policy_denied` | 정책 위반 | 403 |
| `rate_limit_exceeded` | 요청 한도 초과 | 429 |
| `internal_error` | 내부 서버 오류 | 500 |
| `blockchain_error` | 블록체인 통신 오류 | 503 |

## SDK 설치 방법

### Go SDK
```bash
go get github.com/sage-project/sage-sdk-go
```

### TypeScript SDK
```bash
npm install @sage/sdk
# 또는
yarn add @sage/sdk
```

### Python SDK (예정)
```bash
pip install sage-sdk
```

## 추가 예제

전체 예제 코드는 다음 저장소에서 확인할 수 있습니다:
- [Go 예제](https://github.com/sage-project/sage-examples-go)
- [TypeScript 예제](https://github.com/sage-project/sage-examples-ts)
- [Gateway 구현 예제](https://github.com/sage-project/sage-gateway-example)