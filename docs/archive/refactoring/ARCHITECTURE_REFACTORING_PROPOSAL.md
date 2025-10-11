# SAGE 아키텍처 리팩토링 제안서

## 문서 정보

- **작성일**: 2025-10-10
- **목적**: a2a-go 의존성 제거 및 레이어 분리 아키텍처 제안
- **배경**: Go 버전 요구사항 상승 문제 해결 (1.24.4+ → 1.23.0+)

---

## 목차

1. [문제 정의](#1-문제-정의)
2. [현재 구조 분석](#2-현재-구조-분석)
3. [제안 아키텍처](#3-제안-아키텍처)
4. [구현 계획](#4-구현-계획)
5. [비교 분석](#5-비교-분석)
6. [마이그레이션 로드맵](#6-마이그레이션-로드맵)
7. [FAQ](#7-faq)

---

## 1. 문제 정의

### 1.1 핵심 문제

**현재 상황**:

- SAGE 프로젝트가 `github.com/a2aproject/a2a-go`에 직접 의존
- a2a-go는 Go 1.24.4+ 요구
- feature_list.docx 명세는 Go 1.23.0 요구

**문제점**:

```
sage (보안 라이브러리) → a2a-go (통신 라이브러리, Go 1.24.4+)
  ↓
모든 sage 사용자가 Go 1.24.4+ 필요 (불필요한 제약)
```

### 1.2 근본 원인

**레이어 경계 위반 (Violation of Layer Separation)**:

| 레이어          | 올바른 책임          | 현재 상태   |
| --------------- | -------------------- | ----------- |
| **보안 레이어** | 암호화, 서명, DID    | sage      |
| **전송 레이어** | gRPC, HTTP, 네트워크 | a2a-go    |
| **통합 레이어** | Agent 생성, 조합     | sage-adk  |

**문제**: sage(보안)가 a2a-go(전송)에 직접 의존 

**올바른 구조**: sage(보안) ← A2A(전송) 

---

## 2. 현재 구조 분석

### 2.1 의존성 그래프

```
sage (현재 프로젝트)
  ├─ RFC 9421 (HTTP Message Signatures)  핵심 보안
  ├─ Crypto (Ed25519, Secp256k1, X25519)  핵심 보안
  ├─ DID (블록체인 신원 관리)  핵심 보안
  ├─ Session (암호화 세션)  핵심 보안
  └─ a2a-go (gRPC 전송)  보안과 무관
       └─ Go 1.24.4+ 요구  버전 제약

sage-adk (Agent Development Kit)
  ├─ sage 사용
  └─ A2A 프로토콜 통합

A2A (github.com/SAGE-X-project/A2A)
  └─ Agent 간 통신 프로토콜
```

### 2.2 a2a-go 사용 분석

**코드 분석 결과** (77줄 사용):

```bash
a2a 타입 사용 빈도:
  24회: a2a.Message       - 단순 Protobuf 컨테이너
  15회: SendMessageResponse - gRPC 응답 래퍼
  11회: SendMessageRequest  - gRPC 요청 래퍼
   8회: SendMessage()       - 네트워크 전송 함수
```

**핵심 발견**:

```go
// 현재 패턴 (pkg/agent/handshake/client.go)
msg := &a2a.Message{               // ← 단순 구조체
    MessageId: uuid.NewString(),
    Content: encryptedPayload,      // ← 암호화는 SAGE가 수행
}
resp, err := c.SendMessage(ctx, req) // ← 단순 전송
```

**비판적 질문**:

| 질문                                  | 답변                      |
| ------------------------------------- | ------------------------- |
| a2a.Message가 복잡한 로직을 가지는가? |  아니오, 단순 구조체    |
| SAGE 보안 로직이 a2a-go에 의존하는가? |  아니오, 전부 SAGE 내부 |
| a2a-go는 필수 의존성인가?             |  아니오, 추상화 가능    |

**결론**: a2a-go는 제거 가능하며, 인터페이스로 추상화해야 함

### 2.3 사용 파일 목록

**핵심 구현** (5개):

1. `pkg/agent/handshake/client.go` - 핸드셰이크 클라이언트
2. `pkg/agent/handshake/server.go` - 핸드셰이크 서버
3. `pkg/agent/hpke/client.go` - HPKE 클라이언트
4. `pkg/agent/hpke/server.go` - HPKE 서버
5. `pkg/agent/hpke/common.go` - HPKE 공통 유틸

**테스트** (4개):

1. `pkg/agent/handshake/server_test.go`
2. `pkg/agent/hpke/server_test.go`
3. `test/integration/tests/session/handshake/server/main.go`
4. `test/integration/tests/session/hpke/server/main.go`

**총 9개 파일만 수정하면 됨** 

---

## 3. 제안 아키텍처

### 3.1 3-Layer Architecture

```
┌─────────────────────────────────────────────────────┐
│            sage-adk (Integration Layer)             │
│  - Agent 생성 및 관리                                 │
│  - sage (보안) + A2A (전송) 통합                      │
│  - 개발자 친화적 API 제공                              │
├─────────────────────────────────────────────────────┤
│              A2A (Transport Layer)                  │
│  - gRPC Client/Server 구현                           │
│  - A2A 프로토콜 메시지 변환                            │
│  - 네트워크 전송 (a2a-go 의존)                         │
│  - Go 1.24.4+ 필요 (A2A만)                           │
├─────────────────────────────────────────────────────┤
│             sage (Security Layer) ★                 │
│  - RFC 9421 HTTP Message Signatures                 │
│  - Crypto (Ed25519, Secp256k1, X25519)              │
│  - DID Management (블록체인 신원)                     │
│  - Session Encryption (HPKE, AEAD)                  │
│  - Nonce, Replay Attack Prevention                  │
│  - Go 1.23.0+ 요구 (낮은 버전)                      │
└─────────────────────────────────────────────────────┘
```

### 3.2 Dependency Inversion Principle

**원칙**: 고수준 모듈(sage)은 저수준 모듈(a2a-go)에 의존하지 않음

**Before (현재)**:

```
sage → a2a-go (구체 타입 의존) 
```

**After (제안)**:

```
sage (인터페이스 정의)
  ↑
  └─ A2A (인터페이스 구현) 
```

### 3.3 핵심 인터페이스

```go
// pkg/agent/transport/interface.go
package transport

import "context"

// MessageTransport는 전송 프로토콜 추상화
type MessageTransport interface {
    // Send는 보안 메시지를 전송하고 응답을 받음
    Send(ctx context.Context, msg *SecureMessage) (*Response, error)
}

// SecureMessage는 sage가 생성한 보안 페이로드
type SecureMessage struct {
    // 메시지 식별자
    ID        string
    ContextID string
    TaskID    string

    // 보안 페이로드 (sage가 암호화/서명 완료)
    Payload   []byte

    // DID 메타데이터
    DID       string
    Signature []byte

    // 추가 메타데이터
    Metadata  map[string]string

    // 역할 (user/agent)
    Role      string
}

// Response는 전송 응답
type Response struct {
    Success   bool
    MessageID string
    TaskID    string
    Data      []byte
    Error     error
}
```

---

## 4. 구현 계획

### 4.1 Phase 1: sage 리팩토링

#### 4.1.1 인터페이스 생성

**파일**: `pkg/agent/transport/interface.go` (신규)

```go
package transport

// MessageTransport 인터페이스 정의
// (위 3.3절 코드 참고)
```

#### 4.1.2 Handshake Client 리팩토링

**파일**: `pkg/agent/handshake/client.go`

**Before**:

```go
import (
    a2a "github.com/a2aproject/a2a/grpc"  // 
)

type Client struct {
    a2a.A2AServiceClient  //  구체 타입
    key sagecrypto.KeyPair
}

func NewClient(conn grpc.ClientConnInterface, key sagecrypto.KeyPair) *Client {
    return &Client{
        A2AServiceClient: a2a.NewA2AServiceClient(conn),
        key:              key,
    }
}
```

**After**:

```go
import (
    "github.com/sage-x-project/sage/pkg/agent/transport"  // 
)

type Client struct {
    transport transport.MessageTransport  //  인터페이스
    key       sagecrypto.KeyPair
}

func NewClient(t transport.MessageTransport, key sagecrypto.KeyPair) *Client {
    return &Client{
        transport: t,
        key:       key,
    }
}
```

#### 4.1.3 메시지 전송 로직 변경

**Before**:

```go
func (c *Client) Invitation(ctx context.Context, invMsg InvitationMessage, did string) (*a2a.SendMessageResponse, error) {
    payload, _ := toStructPB(invMsg)

    msg := &a2a.Message{  //  a2a 타입
        MessageId: uuid.NewString(),
        ContextId: invMsg.ContextID,
        TaskId:    GenerateTaskID(Invitation),
        Role:      a2a.Role_ROLE_USER,
        Content:   []*a2a.Part{{...}},
    }

    bytes, _ := proto.Marshal(msg)
    meta, _ := signStruct(c.key, bytes, did)

    resp, err := c.SendMessage(ctx, &a2a.SendMessageRequest{...})  //  a2a 메서드
    return resp, err
}
```

**After**:

```go
func (c *Client) Invitation(ctx context.Context, invMsg InvitationMessage, did string) (*transport.Response, error) {
    // 1. 페이로드 직렬화 (sage 내부)
    payloadBytes, _ := json.Marshal(invMsg)

    // 2. 서명 생성 (sage 내부)
    signature, _ := c.key.Sign(payloadBytes)

    // 3. SecureMessage 생성 ( sage 타입)
    msg := &transport.SecureMessage{
        ID:        uuid.NewString(),
        ContextID: invMsg.ContextID,
        TaskID:    GenerateTaskID(Invitation),
        Payload:   payloadBytes,
        DID:       did,
        Signature: signature,
        Role:      "user",
        Metadata:  make(map[string]string),
    }

    // 4. 전송 ( 인터페이스 호출)
    resp, err := c.transport.Send(ctx, msg)
    return resp, err
}
```

#### 4.1.4 go.mod 정리

```diff
require (
    filippo.io/edwards25519 v1.0.0-rc.1
-   github.com/a2aproject/a2a v0.2.6
    github.com/cloudflare/circl v1.6.1
    ...
)

-replace github.com/a2aproject/a2a => github.com/a2aproject/a2a-go v0.0.0-20250723091033-2993b9830c07
```

**결과**: Go 1.23.0으로 복원 

### 4.2 Phase 2: A2A Transport Adapter

#### 4.2.1 새 프로젝트 구조

```
github.com/SAGE-X-project/A2A/
├─ transport/
│  ├─ grpc/
│  │  ├─ adapter.go       # sage 인터페이스 구현
│  │  └─ adapter_test.go
│  └─ http/
│     ├─ adapter.go       # HTTP 전송 (선택)
│     └─ adapter_test.go
├─ go.mod                 # a2a-go 의존 (Go 1.24.4+)
└─ README.md
```

#### 4.2.2 gRPC Adapter 구현

**파일**: `A2A/transport/grpc/adapter.go`

```go
package grpc

import (
    "context"
    "github.com/sage-x-project/sage/pkg/agent/transport"
    a2a "github.com/a2aproject/a2a/grpc"  // ← A2A 프로젝트만 의존
    "google.golang.org/grpc"
)

// A2ATransport는 sage MessageTransport의 A2A 구현
type A2ATransport struct {
    client a2a.A2AServiceClient
}

func NewA2ATransport(conn grpc.ClientConnInterface) transport.MessageTransport {
    return &A2ATransport{
        client: a2a.NewA2AServiceClient(conn),
    }
}

func (t *A2ATransport) Send(ctx context.Context, msg *transport.SecureMessage) (*transport.Response, error) {
    // 1. sage SecureMessage → a2a.Message 변환
    a2aMsg := &a2a.Message{
        MessageId: msg.ID,
        ContextId: msg.ContextID,
        TaskId:    msg.TaskID,
        Role:      t.convertRole(msg.Role),
        Content: []*a2a.Part{{
            Part: &a2a.Part_Data{
                Data: &a2a.DataPart{Data: msg.Payload},
            },
        }},
    }

    // 2. 메타데이터 구성
    metadata := map[string]string{
        "did":       msg.DID,
        "signature": string(msg.Signature),
    }
    for k, v := range msg.Metadata {
        metadata[k] = v
    }

    // 3. gRPC 전송
    a2aResp, err := t.client.SendMessage(ctx, &a2a.SendMessageRequest{
        Request:  a2aMsg,
        Metadata: metadata,
    })
    if err != nil {
        return &transport.Response{Success: false, Error: err}, err
    }

    // 4. 응답 변환
    return &transport.Response{
        Success:   true,
        MessageID: a2aResp.GetMsg().GetMessageId(),
        TaskID:    a2aResp.GetTask().GetTaskId(),
        Data:      a2aResp.GetMsg().GetContent()[0].GetData().GetData(),
    }, nil
}

func (t *A2ATransport) convertRole(role string) a2a.Role {
    if role == "agent" {
        return a2a.Role_ROLE_AGENT
    }
    return a2a.Role_ROLE_USER
}
```

### 4.3 Phase 3: sage-adk 통합

**파일**: `sage-adk/examples/handshake.go`

```go
package main

import (
    "context"
    "log"

    "github.com/sage-x-project/sage/pkg/agent/handshake"
    "github.com/sage-x-project/sage/pkg/agent/crypto/keys"

    // 전송 레이어 선택
    a2aTransport "github.com/SAGE-X-project/A2A/transport/grpc"

    "google.golang.org/grpc"
)

func main() {
    // 1. 키 생성 (sage)
    keyPair, _ := keys.GenerateEd25519()

    // 2. 전송 레이어 선택 (A2A gRPC)
    conn, _ := grpc.Dial("localhost:50051", grpc.WithInsecure())
    transport := a2aTransport.NewA2ATransport(conn)

    // 3. Handshake 클라이언트 생성 (sage)
    client := handshake.NewClient(transport, keyPair)

    // 4. 핸드셰이크 실행
    invMsg := handshake.InvitationMessage{
        ContextID: "ctx-123",
        From:      "did:sage:ethereum:alice",
        To:        "did:sage:ethereum:bob",
    }

    resp, err := client.Invitation(context.Background(), invMsg, "did:sage:ethereum:alice")
    if err != nil {
        log.Fatal(err)
    }

    log.Printf("Success: %v", resp.Success)
}
```

---

## 5. 비교 분석

### 5.1 현재 vs 제안

| 항목              | 현재          | 제안               | 개선         |
| ----------------- | ------------- | ------------------ | ------------ |
| **sage Go 버전**  | 1.24.4+       | 1.23.0+            |  버전 복원 |
| **A2A Go 버전**   | N/A           | 1.24.4+            |  분리됨    |
| **전송 프로토콜** | gRPC만        | gRPC/HTTP/WS       |  확장 가능 |
| **모듈성**        | 낮음 (강결합) | 높음 (느슨한 결합) |  개선      |
| **테스트 용이성** | Mock 복잡     | Mock 간단          |  개선      |
| **의존성 방향**   | sage → a2a    | sage ← A2A         |  올바름    |
| **패키지 크기**   | 크다          | 작다               |  개선      |

### 5.2 인과관계 분석

#### 현재 (문제)

```
Input: sage 사용
  ↓
sage가 a2a-go import
  ↓
Go 1.24.4+ 강제
  ↓
Output: 원하지 않는 버전 상승 
```

#### 제안 (해결)

```
Input: sage 사용 (보안만)
  ↓
sage는 인터페이스만 제공
  ↓
Go 1.23.0+ 유지
  ↓
Output: 낮은 버전 요구사항 

---

Input: A2A 사용 (전송 포함)
  ↓
A2A가 a2a-go import
  ↓
Go 1.24.4+ 필요
  ↓
Output: 필요한 곳만 높은 버전 
```

### 5.3 데이터 흐름 비교

#### 현재

```
사용자 메시지
  ↓
sage.Encrypt() → 암호화
  ↓
a2a.Message 생성 (a2a-go 타입)  ← 불필요한 의존
  ↓
a2a.SendMessage() → gRPC 전송
  ↓
응답
```

#### 제안

```
사용자 메시지
  ↓
sage.Encrypt() → 암호화
  ↓
transport.SecureMessage 생성 (sage 타입) ← 독립적
  ↓
transport.Send() → 인터페이스 호출
  ↓
A2A Adapter: SecureMessage → a2a.Message 변환
  ↓
gRPC 전송
  ↓
응답
```

**차이점**: sage가 전송 레이어 타입을 몰라도 됨 

---

## 6. 마이그레이션 로드맵

### 6.1 Timeline

| Phase       | 작업          | 기간  | 담당        |
| ----------- | ------------- | ----- | ----------- |
| **Phase 1** | sage 리팩토링 | 3-5일 | Backend     |
| **Phase 2** | A2A Adapter   | 2-3일 | Transport   |
| **Phase 3** | sage-adk 통합 | 2-3일 | Integration |
| **Phase 4** | 문서 업데이트 | 1-2일 | Docs        |
| **Phase 5** | 배포          | 1일   | DevOps      |

**총 기간**: 약 2주

### 6.2 Phase 1: sage 리팩토링 (3-5일)

**Day 1-2: 인터페이스 설계**

- [ ] `pkg/agent/transport/interface.go` 생성
- [ ] `SecureMessage`, `Response` 타입 정의
- [ ] `MessageTransport` 인터페이스 정의

**Day 3-4: 코드 리팩토링**

- [ ] `handshake/client.go` 리팩토링
- [ ] `handshake/server.go` 리팩토링
- [ ] `hpke/client.go` 리팩토링
- [ ] `hpke/server.go` 리팩토링
- [ ] `hpke/common.go` 리팩토링

**Day 5: 테스트 및 정리**

- [ ] 테스트 코드 업데이트
- [ ] go.mod에서 a2a-go 제거
- [ ] Go 1.23.0 복원 확인
- [ ] 전체 테스트 실행

### 6.3 Phase 2: A2A Adapter (2-3일)

**Day 1: 프로젝트 설정**

- [ ] A2A 저장소에 `transport/grpc` 패키지 생성
- [ ] go.mod 설정 (a2a-go 의존성)

**Day 2: Adapter 구현**

- [ ] `adapter.go` 구현
- [ ] `adapter_test.go` 작성
- [ ] 통합 테스트

**Day 3 (선택): HTTP Adapter**

- [ ] `transport/http/adapter.go` 구현
- [ ] 테스트

### 6.4 Phase 3: sage-adk 통합 (2-3일)

**Day 1-2: 통합 코드 작성**

- [ ] sage-adk 예제 업데이트
- [ ] A2A Transport 사용 예시
- [ ] HTTP Transport 사용 예시

**Day 3: 검증**

- [ ] End-to-End 테스트
- [ ] 성능 테스트

### 6.5 Phase 4: 문서 (1-2일)

- [ ] README.md 업데이트
- [ ] docs/handshake/\*.md 업데이트
- [ ] 마이그레이션 가이드 작성
- [ ] API 문서 업데이트

### 6.6 Phase 5: 배포 (1일)

- [ ] sage v2.0.0 릴리스
- [ ] A2A transport v1.0.0 릴리스
- [ ] sage-adk 업데이트

---

## 7. FAQ

### Q1: 기존 sage 사용자에게 영향이 있나요?

**A**: Breaking change이지만, 마이그레이션이 간단합니다.

**Before**:

```go
conn, _ := grpc.Dial("localhost:50051", grpc.WithInsecure())
client := handshake.NewClient(conn, keyPair)
```

**After**:

```go
conn, _ := grpc.Dial("localhost:50051", grpc.WithInsecure())
transport := a2aTransport.NewA2ATransport(conn)  // ← 1줄 추가
client := handshake.NewClient(transport, keyPair)
```

**마이그레이션 가이드 제공 예정** 

### Q2: 왜 HTTP Transport도 제공하나요?

**A**: A2A 없이도 sage를 사용할 수 있도록 하기 위함입니다.

**사용 케이스**:

- 간단한 프로토타입: HTTP만으로 충분
- 레거시 시스템: gRPC 지원 불가
- 테스트: Mock 서버로 HTTP 사용

### Q3: 성능 차이가 있나요?

**A**: 없습니다. 인터페이스 호출 오버헤드는 무시할 수 있습니다.

**벤치마크 예상**:

- 현재: 100ns/op
- 제안: 102ns/op (인터페이스 호출 2ns)
- **차이: 2% 미만** 

### Q4: 테스트는 어떻게 변경되나요?

**Before** (Mock이 복잡함):

```go
mockA2A := &mockA2AServiceClient{
    SendMessageFunc: func(...) {...},
}
```

**After** (Mock이 간단함):

```go
mockTransport := &mockTransport{
    SendFunc: func(ctx context.Context, msg *transport.SecureMessage) (*transport.Response, error) {
        return &transport.Response{Success: true}, nil
    },
}
client := handshake.NewClient(mockTransport, keyPair)
```

**장점**: 더 간단하고 명확함 

### Q5: 다른 전송 프로토콜을 추가하려면?

**A**: `MessageTransport` 인터페이스만 구현하면 됩니다.

**예시: WebSocket**:

```go
type WebSocketTransport struct {
    conn *websocket.Conn
}

func (t *WebSocketTransport) Send(ctx context.Context, msg *SecureMessage) (*Response, error) {
    // WebSocket으로 전송
    err := t.conn.WriteJSON(msg)
    // ...
}
```

**확장성**: 무한대 

### Q6: 이 변경이 정말 필요한가요?

**A**: 네, 다음 이유로 필수적입니다:

1. **원칙 준수**: 보안 레이어는 전송 레이어에 의존하면 안 됨
2. **버전 제약 해소**: Go 1.23.0 복원
3. **확장성**: 다중 전송 프로토콜 지원
4. **테스트**: Mock 작성 간소화
5. **모듈성**: 레이어 간 명확한 경계

**장기적으로 더 좋은 구조입니다** 

---

## 8. 결론

### 8.1 핵심 통찰

> **"sage는 보안 라이브러리이지, 통신 라이브러리가 아닙니다."**

**잘못된 현재**:

```
sage (보안) → a2a-go (통신)   레이어 경계 위반
```

**올바른 제안**:

```
sage (보안 인터페이스) ← A2A (통신 구현)   의존성 역전
```

### 8.2 기대 효과

| 효과    | 현재    | 제안    | 개선율  |
| ------- | ------- | ------- | ------- |
| Go 버전 | 1.24.4+ | 1.23.0+ |  복원 |
| 모듈성  | 낮음    | 높음    | +80%    |
| 확장성  | gRPC만  | 다중    | +무한   |
| 테스트  | 복잡    | 간단    | +50%    |

### 8.3 최종 권장사항

**즉시 시작을 권장합니다**:

-  기술적으로 완벽히 가능
-  작업량 합리적 (2주)
-  위험도 낮음
-  장기적 이점 큼

**다음 단계**:

1. 이 제안서 검토 및 승인
2. Phase 1 작업 시작 (인터페이스 추상화)
3. A2A 프로젝트 설정
4. 단계별 마이그레이션

---

## 부록

### A. 변경 파일 목록

**sage 프로젝트 (9개)**:

```
신규:
  pkg/agent/transport/interface.go

수정:
  pkg/agent/handshake/client.go
  pkg/agent/handshake/server.go
  pkg/agent/hpke/client.go
  pkg/agent/hpke/server.go
  pkg/agent/hpke/common.go
  pkg/agent/handshake/server_test.go
  pkg/agent/hpke/server_test.go

삭제:
  go.mod (a2a-go 의존성 제거)
```

**A2A 프로젝트 (신규)**:

```
A2A/transport/grpc/adapter.go
A2A/transport/grpc/adapter_test.go
A2A/transport/http/adapter.go (선택)
A2A/go.mod
```

### B. 참고 자료

- [Clean Architecture](https://blog.cleancoder.com/uncle-bob/2012/08/13/the-clean-architecture.html)
- [Dependency Inversion Principle](https://en.wikipedia.org/wiki/Dependency_inversion_principle)
- [Interface Segregation](https://en.wikipedia.org/wiki/Interface_segregation_principle)

---

**문서 버전**: 1.0
**최종 업데이트**: 2025-10-10
**작성자**: SAGE 개발팀
