# SAGE 아키텍처 리팩토링 실행 계획

## 문서 정보

- **작성일**: 2025-10-10
- **목적**: a2a-go 의존성 제거를 위한 구체적 실행 계획
- **예상 기간**: 2주 (10 영업일)
- **담당**: SAGE 개발팀

---

## 목차

1. [Overview](#1-overview)
2. [Phase 1: sage 리팩토링](#2-phase-1-sage-리팩토링)
3. [Phase 2: A2A Transport Adapter](#3-phase-2-a2a-transport-adapter)
4. [Phase 3: sage-adk 통합](#4-phase-3-sage-adk-통합)
5. [Phase 4: 문서화](#5-phase-4-문서화)
6. [Phase 5: 배포](#6-phase-5-배포)
7. [체크리스트](#7-체크리스트)

---

## 1. Overview

### 1.1 목표

- ✅ sage에서 a2a-go 의존성 완전 제거
- ✅ Go 버전 요구사항 1.24.4+ → 1.23.0+ 복원
- ✅ 전송 레이어 추상화 (인터페이스 기반)
- ✅ A2A 프로젝트에 gRPC 어댑터 구현
- ✅ 모든 테스트 통과 (89/89)

### 1.2 Timeline

```
Week 1:
  Day 1-2: Phase 1 시작 (인터페이스 설계)
  Day 3-4: Phase 1 계속 (코드 리팩토링)
  Day 5:   Phase 1 완료 (테스트)

Week 2:
  Day 6-7: Phase 2 (A2A Adapter)
  Day 8:   Phase 3 (sage-adk 통합)
  Day 9:   Phase 4 (문서화)
  Day 10:  Phase 5 (배포)
```

### 1.3 성공 기준

| 항목 | 기준 |
|------|------|
| **테스트** | 89/89 통과 (100%) |
| **Go 버전** | go.mod에 `go 1.23.0` |
| **의존성** | go.mod에 a2a-go 없음 |
| **빌드** | `make build` 성공 |
| **문서** | 마이그레이션 가이드 완성 |

---

## 2. Phase 1: sage 리팩토링

### 2.1 Day 1-2: 인터페이스 설계

#### Step 1.1: 인터페이스 파일 생성

```bash
# 디렉토리 생성
mkdir -p pkg/agent/transport

# 인터페이스 파일 생성
touch pkg/agent/transport/interface.go
touch pkg/agent/transport/interface_test.go
```

**파일 내용**: `pkg/agent/transport/interface.go`

```go
// SAGE - Secure Agent Guarantee Engine
// Copyright (C) 2025 SAGE-X-project
//
// This file is part of SAGE.
// SAGE is free software under LGPL-3.0.

package transport

import "context"

// MessageTransport는 전송 프로토콜 추상화 인터페이스
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

**체크리스트**:
- [ ] `pkg/agent/transport/interface.go` 생성
- [ ] `MessageTransport` 인터페이스 정의
- [ ] `SecureMessage` 구조체 정의
- [ ] `Response` 구조체 정의
- [ ] 주석 및 문서화 완료

#### Step 1.2: Mock Transport 구현 (테스트용)

**파일 내용**: `pkg/agent/transport/mock.go`

```go
package transport

import "context"

// MockTransport는 테스트용 Mock 구현
type MockTransport struct {
    SendFunc func(ctx context.Context, msg *SecureMessage) (*Response, error)
}

func (m *MockTransport) Send(ctx context.Context, msg *SecureMessage) (*Response, error) {
    if m.SendFunc != nil {
        return m.SendFunc(ctx, msg)
    }
    return &Response{Success: true, MessageID: msg.ID}, nil
}
```

**체크리스트**:
- [ ] `pkg/agent/transport/mock.go` 생성
- [ ] `MockTransport` 구조체 정의
- [ ] 테스트에서 사용 가능 확인

### 2.2 Day 3-4: 코드 리팩토링

#### Step 2.1: handshake/client.go 리팩토링

**작업**:
1. import 변경: `a2a` → `transport`
2. Client 구조체 변경
3. NewClient 함수 시그니처 변경
4. 각 메서드 리팩토링

**변경 파일**: `pkg/agent/handshake/client.go`

**체크리스트**:
- [ ] import 문 수정
  - [ ] `a2a "github.com/a2aproject/a2a/grpc"` 제거
  - [ ] `"github.com/sage-x-project/sage/pkg/agent/transport"` 추가
- [ ] Client 구조체 수정
  - [ ] `a2a.A2AServiceClient` → `transport transport.MessageTransport`
- [ ] NewClient 함수 수정
  - [ ] 파라미터: `grpc.ClientConnInterface` → `transport.MessageTransport`
- [ ] Invitation 메서드 리팩토링
  - [ ] 반환 타입: `*a2a.SendMessageResponse` → `*transport.Response`
  - [ ] a2a.Message 생성 → transport.SecureMessage 생성
  - [ ] c.SendMessage(a2a) → c.transport.Send(transport)
- [ ] Request 메서드 리팩토링 (동일 패턴)
- [ ] Response 메서드 리팩토링 (동일 패턴)
- [ ] Complete 메서드 리팩토링 (동일 패턴)

**코드 예시 (Invitation 메서드)**:

```go
// BEFORE
func (c *Client) Invitation(ctx context.Context, invMsg InvitationMessage, did string) (*a2a.SendMessageResponse, error) {
    payload, _ := toStructPB(invMsg)
    msg := &a2a.Message{
        MessageId: uuid.NewString(),
        ContextId: invMsg.ContextID,
        TaskId:    GenerateTaskID(Invitation),
        Role:      a2a.Role_ROLE_USER,
        Content:   []*a2a.Part{{Part: &a2a.Part_Data{Data: &a2a.DataPart{Data: payload}}}},
    }
    bytes, _ := proto.Marshal(msg)
    meta, _ := signStruct(c.key, bytes, did)
    resp, err := c.SendMessage(ctx, &a2a.SendMessageRequest{Request: msg, Metadata: meta})
    return resp, err
}

// AFTER
func (c *Client) Invitation(ctx context.Context, invMsg InvitationMessage, did string) (*transport.Response, error) {
    // 1. 페이로드 직렬화
    payloadBytes, err := json.Marshal(invMsg)
    if err != nil {
        return nil, fmt.Errorf("marshal invitation: %w", err)
    }

    // 2. 서명 생성
    signature, err := c.key.Sign(payloadBytes)
    if err != nil {
        return nil, fmt.Errorf("sign: %w", err)
    }

    // 3. SecureMessage 생성
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

    // 4. 전송
    return c.transport.Send(ctx, msg)
}
```

#### Step 2.2: handshake/server.go 리팩토링

**체크리스트**:
- [ ] import 문 수정
- [ ] Server 구조체 수정
- [ ] ProcessInvitation 메서드 리팩토링
- [ ] ProcessRequest 메서드 리팩토링
- [ ] ProcessResponse 메서드 리팩토링
- [ ] ProcessComplete 메서드 리팩토링

#### Step 2.3: hpke/client.go 리팩토링

**체크리스트**:
- [ ] import 문 수정
- [ ] Client 구조체 수정
- [ ] NewClient 함수 수정
- [ ] Initialize 메서드 리팩토링
- [ ] SendEncrypted 메서드 리팩토링

#### Step 2.4: hpke/server.go 리팩토링

**체크리스트**:
- [ ] import 문 수정
- [ ] Server 구조체 수정
- [ ] HandleInit 메서드 리팩토링
- [ ] SendAcknowledge 메서드 리팩토링

#### Step 2.5: hpke/common.go 리팩토링

**체크리스트**:
- [ ] import 문 수정
- [ ] 공통 유틸 함수 수정

### 2.3 Day 5: 테스트 및 정리

#### Step 3.1: 테스트 코드 업데이트

**파일**: `pkg/agent/handshake/server_test.go`

```go
// BEFORE
mockA2A := &mockA2AServiceClient{...}

// AFTER
mockTransport := &transport.MockTransport{
    SendFunc: func(ctx context.Context, msg *transport.SecureMessage) (*transport.Response, error) {
        // 테스트 로직
        return &transport.Response{Success: true, MessageID: msg.ID}, nil
    },
}
client := handshake.NewClient(mockTransport, keyPair)
```

**체크리스트**:
- [ ] `handshake/server_test.go` 업데이트
- [ ] `hpke/server_test.go` 업데이트
- [ ] 통합 테스트 파일 업데이트

#### Step 3.2: go.mod 정리

```bash
# a2a-go 의존성 제거
go mod edit -droprequire github.com/a2aproject/a2a
go mod edit -dropreplace github.com/a2aproject/a2a

# Go 버전 변경
go mod edit -go=1.23.0

# 정리
go mod tidy
```

**체크리스트**:
- [ ] `go.mod`에서 a2a-go 제거
- [ ] `go 1.23.0` 설정
- [ ] `go mod tidy` 실행
- [ ] `go.sum` 확인

#### Step 3.3: 전체 테스트 실행

```bash
# 유닛 테스트
make test

# 통합 테스트 (블록체인 제외)
go test ./pkg/agent/handshake/...
go test ./pkg/agent/hpke/...

# 전체 검증
./tools/scripts/verify_all_features.sh
```

**체크리스트**:
- [ ] 유닛 테스트 89/89 통과
- [ ] handshake 테스트 통과
- [ ] hpke 테스트 통과
- [ ] 전체 검증 스크립트 통과

#### Step 3.4: 빌드 확인

```bash
# CLI 빌드
make build

# 확인
./build/bin/sage-crypto --version
./build/bin/sage-did --version
```

**체크리스트**:
- [ ] `make build` 성공
- [ ] sage-crypto 실행 확인
- [ ] sage-did 실행 확인

---

## 3. Phase 2: A2A Transport Adapter

### 3.1 Day 6: 프로젝트 설정

#### Step 4.1: A2A 저장소 설정

```bash
# A2A 저장소로 이동 (또는 클론)
cd ../A2A

# transport 디렉토리 생성
mkdir -p transport/grpc
cd transport/grpc
```

#### Step 4.2: go.mod 초기화

```bash
go mod init github.com/SAGE-X-project/A2A

# sage 의존성 추가
go get github.com/sage-x-project/sage/pkg/agent/transport@latest

# a2a-go 의존성 추가 (Go 1.24.4+ 필요)
go get github.com/a2aproject/a2a-go@latest

# protobuf 추가
go get google.golang.org/protobuf
go get google.golang.org/grpc
```

**체크리스트**:
- [ ] A2A 프로젝트 go.mod 생성
- [ ] sage transport 패키지 import
- [ ] a2a-go 의존성 추가
- [ ] Go 1.24.4+ 확인

### 3.2 Day 7: Adapter 구현

#### Step 5.1: gRPC Adapter 코드 작성

**파일**: `A2A/transport/grpc/adapter.go`

```go
// Copyright (C) 2025 SAGE-X-project
// Licensed under MIT

package grpc

import (
    "context"
    "fmt"

    "github.com/sage-x-project/sage/pkg/agent/transport"
    a2a "github.com/a2aproject/a2a/grpc"
    "google.golang.org/grpc"
)

// A2ATransport는 sage MessageTransport의 A2A gRPC 구현
type A2ATransport struct {
    client a2a.A2AServiceClient
}

// NewA2ATransport는 A2A gRPC Transport를 생성
func NewA2ATransport(conn grpc.ClientConnInterface) transport.MessageTransport {
    return &A2ATransport{
        client: a2a.NewA2AServiceClient(conn),
    }
}

// Send는 SecureMessage를 A2A 프로토콜로 전송
func (t *A2ATransport) Send(ctx context.Context, msg *transport.SecureMessage) (*transport.Response, error) {
    // 1. SecureMessage → a2a.Message 변환
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
    metadata := make(map[string]string)
    metadata["did"] = msg.DID
    metadata["signature"] = string(msg.Signature)
    for k, v := range msg.Metadata {
        metadata[k] = v
    }

    // 3. gRPC 전송
    a2aResp, err := t.client.SendMessage(ctx, &a2a.SendMessageRequest{
        Request:  a2aMsg,
        Metadata: metadata,
    })
    if err != nil {
        return &transport.Response{
            Success: false,
            Error:   err,
        }, fmt.Errorf("a2a send failed: %w", err)
    }

    // 4. a2a.Response → transport.Response 변환
    return &transport.Response{
        Success:   true,
        MessageID: a2aResp.GetMsg().GetMessageId(),
        TaskID:    a2aResp.GetTask().GetTaskId(),
        Data:      a2aResp.GetMsg().GetContent()[0].GetData().GetData(),
    }, nil
}

func (t *A2ATransport) convertRole(role string) a2a.Role {
    switch role {
    case "agent":
        return a2a.Role_ROLE_AGENT
    default:
        return a2a.Role_ROLE_USER
    }
}
```

**체크리스트**:
- [ ] `adapter.go` 파일 생성
- [ ] `A2ATransport` 구조체 정의
- [ ] `NewA2ATransport` 생성자
- [ ] `Send` 메서드 구현
- [ ] 타입 변환 로직 (`convertRole` 등)

#### Step 5.2: 테스트 작성

**파일**: `A2A/transport/grpc/adapter_test.go`

```go
package grpc_test

import (
    "context"
    "testing"

    "github.com/SAGE-X-project/A2A/transport/grpc"
    "github.com/sage-x-project/sage/pkg/agent/transport"
    "github.com/stretchr/testify/assert"
)

func TestA2ATransport_Send(t *testing.T) {
    // Mock gRPC connection
    // ...

    adapter := grpc.NewA2ATransport(conn)

    msg := &transport.SecureMessage{
        ID:        "test-id",
        ContextID: "ctx-123",
        TaskID:    "task-456",
        Payload:   []byte("encrypted payload"),
        DID:       "did:sage:ethereum:alice",
        Signature: []byte("signature"),
        Role:      "user",
    }

    resp, err := adapter.Send(context.Background(), msg)

    assert.NoError(t, err)
    assert.True(t, resp.Success)
}
```

**체크리스트**:
- [ ] `adapter_test.go` 파일 생성
- [ ] Send 메서드 테스트
- [ ] 타입 변환 테스트
- [ ] 에러 핸들링 테스트

---

## 4. Phase 3: sage-adk 통합

### 4.1 Day 8: 통합 예제

#### Step 6.1: 예제 코드 작성

**파일**: `sage-adk/examples/handshake_with_a2a.go`

```go
package main

import (
    "context"
    "log"

    "github.com/sage-x-project/sage/pkg/agent/handshake"
    "github.com/sage-x-project/sage/pkg/agent/crypto/keys"
    a2aTransport "github.com/SAGE-X-project/A2A/transport/grpc"

    "google.golang.org/grpc"
)

func main() {
    // 1. 키 생성
    keyPair, err := keys.GenerateEd25519()
    if err != nil {
        log.Fatal(err)
    }

    // 2. gRPC 연결
    conn, err := grpc.Dial("localhost:50051", grpc.WithInsecure())
    if err != nil {
        log.Fatal(err)
    }
    defer conn.Close()

    // 3. A2A Transport 생성
    transport := a2aTransport.NewA2ATransport(conn)

    // 4. Handshake 클라이언트
    client := handshake.NewClient(transport, keyPair)

    // 5. Invitation 전송
    invMsg := handshake.InvitationMessage{
        ContextID: "ctx-123",
        From:      "did:sage:ethereum:alice",
        To:        "did:sage:ethereum:bob",
    }

    resp, err := client.Invitation(context.Background(), invMsg, "did:sage:ethereum:alice")
    if err != nil {
        log.Fatal(err)
    }

    log.Printf("Handshake success: %v", resp.Success)
}
```

**체크리스트**:
- [ ] 예제 코드 작성
- [ ] 컴파일 확인
- [ ] 실행 테스트 (로컬 gRPC 서버 필요)

---

## 5. Phase 4: 문서화

### 5.1 Day 9: 문서 업데이트

#### Step 7.1: README 업데이트

**파일**: `sage/README.md`

**변경사항**:
```diff
### Prerequisites

- **Go 1.24.4 or higher** (required by a2a-go dependency)
+ **Go 1.23.0 or higher**
```

**체크리스트**:
- [ ] README.md Go 버전 업데이트
- [ ] 사용 예시 업데이트
- [ ] 의존성 섹션 업데이트

#### Step 7.2: 마이그레이션 가이드 작성

**파일**: `sage/docs/MIGRATION_GUIDE_V2.md`

```markdown
# SAGE v2.0 마이그레이션 가이드

## Breaking Changes

### handshake/hpke 클라이언트 생성자 변경

**Before (v1.x)**:
```go
conn, _ := grpc.Dial("localhost:50051", grpc.WithInsecure())
client := handshake.NewClient(conn, keyPair)
```

**After (v2.0)**:
```go
import a2aTransport "github.com/SAGE-X-project/A2A/transport/grpc"

conn, _ := grpc.Dial("localhost:50051", grpc.WithInsecure())
transport := a2aTransport.NewA2ATransport(conn)
client := handshake.NewClient(transport, keyPair)
```

## 마이그레이션 단계

1. A2A transport 패키지 추가
2. NewClient 호출 시 transport 주입
3. 테스트 실행
```

**체크리스트**:
- [ ] 마이그레이션 가이드 작성
- [ ] Breaking changes 문서화
- [ ] 예제 코드 제공

#### Step 7.3: API 문서 업데이트

**체크리스트**:
- [ ] `docs/handshake/*.md` 업데이트
- [ ] `docs/API.md` 업데이트
- [ ] 코드 주석 업데이트

---

## 6. Phase 5: 배포

### 6.1 Day 10: 릴리스

#### Step 8.1: 버전 태그

```bash
# sage v2.0.0
cd sage
git tag -a v2.0.0 -m "feat: Remove a2a-go dependency, restore Go 1.23.0"
git push origin v2.0.0

# A2A transport v1.0.0
cd ../A2A
git tag -a transport/v1.0.0 -m "feat: Initial A2A gRPC transport adapter"
git push origin transport/v1.0.0
```

**체크리스트**:
- [ ] sage v2.0.0 태그
- [ ] A2A transport v1.0.0 태그
- [ ] GitHub Release 노트 작성

#### Step 8.2: 릴리스 노트

**sage v2.0.0 Release Notes**:

```markdown
# SAGE v2.0.0 - Architecture Refactoring

## Breaking Changes

- **Removed a2a-go dependency**: sage no longer directly depends on a2a-go
- **Go version restored**: Now requires Go 1.23.0+ (down from 1.24.4+)
- **Constructor signature changed**: handshake/hpke clients now accept `transport.MessageTransport` interface

## New Features

- **Transport abstraction**: Support for multiple transport protocols (gRPC, HTTP, WebSocket)
- **Better testability**: Simple mock interfaces for testing
- **Modular architecture**: Clear separation between security and transport layers

## Migration

See [MIGRATION_GUIDE_V2.md](docs/MIGRATION_GUIDE_V2.md)

## Dependencies

- A2A Transport adapter: `github.com/SAGE-X-project/A2A/transport/grpc@v1.0.0`
```

**체크리스트**:
- [ ] 릴리스 노트 작성
- [ ] CHANGELOG.md 업데이트
- [ ] GitHub Release 게시

---

## 7. 체크리스트

### 7.1 Phase 1 완료 조건

- [ ] `pkg/agent/transport/interface.go` 생성
- [ ] `pkg/agent/transport/mock.go` 생성
- [ ] `handshake/client.go` 리팩토링
- [ ] `handshake/server.go` 리팩토링
- [ ] `hpke/client.go` 리팩토링
- [ ] `hpke/server.go` 리팩토링
- [ ] `hpke/common.go` 리팩토링
- [ ] 테스트 파일 업데이트 (5개)
- [ ] `go.mod`에서 a2a-go 제거
- [ ] `go.mod`에 `go 1.23.0` 설정
- [ ] 전체 테스트 통과 (89/89)
- [ ] `make build` 성공

### 7.2 Phase 2 완료 조건

- [ ] A2A 프로젝트 `go.mod` 생성
- [ ] `transport/grpc/adapter.go` 구현
- [ ] `transport/grpc/adapter_test.go` 작성
- [ ] 테스트 통과
- [ ] README.md 작성

### 7.3 Phase 3 완료 조건

- [ ] sage-adk 예제 작성
- [ ] 통합 테스트 통과

### 7.4 Phase 4 완료 조건

- [ ] `sage/README.md` 업데이트
- [ ] `MIGRATION_GUIDE_V2.md` 작성
- [ ] API 문서 업데이트
- [ ] handshake 문서 업데이트

### 7.5 Phase 5 완료 조건

- [ ] sage v2.0.0 태그
- [ ] A2A transport v1.0.0 태그
- [ ] GitHub Release 게시
- [ ] CHANGELOG 업데이트

---

## 8. 롤백 계획

### 8.1 Phase 1 실패 시

```bash
# 변경사항 되돌리기
git checkout HEAD -- pkg/agent/handshake/
git checkout HEAD -- pkg/agent/hpke/
git checkout HEAD -- go.mod

# transport 디렉토리 삭제
rm -rf pkg/agent/transport/
```

### 8.2 Phase 2 실패 시

- A2A 프로젝트는 별도이므로 sage에 영향 없음
- sage는 Phase 1 완료 상태 유지

### 8.3 전체 롤백

```bash
# sage 저장소
git reset --hard <commit-before-refactoring>

# go.mod 복원
go mod edit -require github.com/a2aproject/a2a@v0.2.6
go mod edit -replace github.com/a2aproject/a2a=github.com/a2aproject/a2a-go@v0.0.0-20250723091033-2993b9830c07
go mod edit -go=1.24.4
go mod tidy
```

---

## 9. 리스크 관리

| 리스크 | 확률 | 영향 | 대응 |
|--------|------|------|------|
| 테스트 실패 | 중 | 높음 | 철저한 단위 테스트 |
| 성능 저하 | 낮음 | 중 | 벤치마크 테스트 |
| 호환성 문제 | 낮음 | 높음 | 마이그레이션 가이드 |
| 일정 지연 | 중 | 중 | 우선순위 조정 |

---

## 10. 연락처 및 지원

**문제 발생 시**:
- GitHub Issues: https://github.com/SAGE-X-project/sage/issues
- 개발팀 이메일: dev@sage-x-project.org

---

**문서 버전**: 1.0
**최종 업데이트**: 2025-10-10
**담당자**: SAGE 개발팀
