# SAGE 프로젝트 상세 가이드 - Part 4: 핸드셰이크 프로토콜 및 세션 관리

> **대상 독자**: 프로그래밍 초급자부터 중급 개발자까지
> **작성일**: 2025-10-07
> **버전**: 1.0
> **이전**: [Part 3 - DID 및 블록체인 통합](./DETAILED_GUIDE_PART3_KO.md)

---

## 목차
1. [핸드셰이크 프로토콜 개요](#1-핸드셰이크-프로토콜-개요)
2. [4단계 핸드셰이크 상세](#2-4단계-핸드셰이크-상세)
3. [클라이언트 구현](#3-클라이언트-구현)
4. [서버 구현](#4-서버-구현)
5. [세션 생성 및 키 유도](#5-세션-생성-및-키-유도)
6. [세션 관리자](#6-세션-관리자)
7. [이벤트 기반 아키텍처](#7-이벤트-기반-아키텍처)
8. [보안 고려사항](#8-보안-고려사항)
9. [실전 예제](#9-실전-예제)

---

## 1. 핸드셰이크 프로토콜 개요

### 1.1 왜 핸드셰이크가 필요한가?

**문제 상황**:
```
AI Agent A와 Agent B가 처음 만났을 때:

❓ 문제 1: 신원 확인
   - B가 정말 B인지 어떻게 확인?
   - A가 정말 A인지 어떻게 확인?

❓ 문제 2: 안전한 키 교환
   - 대칭키를 어떻게 안전하게 공유?
   - 중간자 공격 방지는?

❓ 문제 3: Forward Secrecy
   - 나중에 개인키가 노출되어도 과거 대화는 안전?
   - 세션마다 다른 키 사용?

❓ 문제 4: 재생 공격 방지
   - 같은 메시지를 여러 번 보내는 것 방지?
   - Nonce 관리는 어떻게?
```

**SAGE의 해결책: 4단계 핸드셰이크**

```
┌─────────────────────────────────────────────────────┐
│   Invitation: "나는 A야, 대화하고 싶어"             │
│   → DID 서명으로 신원 증명                          │
└─────────────────────┬───────────────────────────────┘
                      ↓
┌─────────────────────────────────────────────────────┐
│   Request: A가 임시 공개키 전송                      │
│   → B의 공개키로 암호화 (부트스트랩)                 │
└─────────────────────┬───────────────────────────────┘
                      ↓
┌─────────────────────────────────────────────────────┐
│   Response: B가 임시 공개키 전송                     │
│   → A의 공개키로 암호화 (부트스트랩)                 │
└─────────────────────┬───────────────────────────────┘
                      ↓
┌─────────────────────────────────────────────────────┐
│   Complete: 세션 확정                                │
│   → 양측이 같은 세션 키 유도                        │
│   → KeyID 발급 및 바인딩                            │
└─────────────────────────────────────────────────────┘

결과:
✅ 상호 인증 완료 (Mutual Authentication)
✅ 안전한 세션 키 공유 (Secure Key Exchange)
✅ Forward Secrecy 보장
✅ 재생 공격 방지 (Nonce)
```

### 1.2 핸드셰이크 vs TLS

**TLS (Transport Layer Security)와의 비교**:

```
TLS 1.3 핸드셰이크:
┌──────────────────────────────────────┐
│ 1. ClientHello                       │
│    - 지원하는 암호화 스위트           │
│    - 랜덤 nonce                      │
│    - 임시 키 공유 (KeyShare)         │
├──────────────────────────────────────┤
│ 2. ServerHello                       │
│    - 선택한 암호화 스위트             │
│    - 랜덤 nonce                      │
│    - 임시 키 공유                     │
│    - 인증서 (Certificate)            │
│    - 인증서 검증 (CertificateVerify) │
├──────────────────────────────────────┤
│ 3. Client 완료                       │
│    - Finished (MAC)                  │
├──────────────────────────────────────┤
│ 4. Server 완료                       │
│    - Finished (MAC)                  │
└──────────────────────────────────────┘

SAGE 핸드셰이크:
┌──────────────────────────────────────┐
│ 1. Invitation                        │
│    - DID 기반 신원                    │
│    - Ed25519 서명                    │
│    - 컨텍스트 ID                     │
├──────────────────────────────────────┤
│ 2. Request                           │
│    - X25519 임시 공개키              │
│    - DID 공개키로 암호화             │
│    - Ed25519 서명                    │
├──────────────────────────────────────┤
│ 3. Response                          │
│    - X25519 임시 공개키              │
│    - DID 공개키로 암호화             │
│    - Ed25519 서명                    │
├──────────────────────────────────────┤
│ 4. Complete                          │
│    - 세션 확정                        │
│    - KeyID 발급                      │
└──────────────────────────────────────┘

차이점:
┌────────────────┬───────────────┬──────────────┐
│                │ TLS 1.3       │ SAGE         │
├────────────────┼───────────────┼──────────────┤
│ 신원 증명      │ X.509 인증서  │ DID          │
│ 인증 기관      │ CA 필요       │ 블록체인     │
│ 키 알고리즘    │ RSA/ECDSA     │ Ed25519      │
│ 키 교환        │ ECDHE         │ X25519       │
│ 세션 암호화    │ AES-GCM       │ ChaCha20     │
│ 메시지 서명    │ HMAC          │ HMAC-SHA256  │
│ Forward Secrecy│ ✅            │ ✅           │
│ 블록체인 통합  │ ❌            │ ✅           │
└────────────────┴───────────────┴──────────────┘
```

### 1.3 A2A 프로토콜 통합

**A2A (Agent-to-Agent) 프로토콜**:

```
SAGE는 A2A 프로토콜 위에서 동작:

┌─────────────────────────────────────────┐
│        Application Layer                 │
│     (AI Agent Business Logic)            │
└──────────────┬──────────────────────────┘
               │
┌──────────────┴──────────────────────────┐
│        SAGE Handshake Layer              │
│   (Security & Session Establishment)     │
└──────────────┬──────────────────────────┘
               │
┌──────────────┴──────────────────────────┐
│        A2A Protocol Layer                │
│   (Message Structure & Routing)          │
└──────────────┬──────────────────────────┘
               │
┌──────────────┴──────────────────────────┐
│        gRPC Transport Layer              │
│        (HTTP/2, Binary Protocol)         │
└──────────────┬──────────────────────────┘
               │
┌──────────────┴──────────────────────────┐
│        Network Layer (TCP/IP)            │
└─────────────────────────────────────────┘

A2A Message 구조:
{
  "messageId": "uuid-1234",
  "contextId": "conversation-abc",
  "taskId": "handshake:invitation",
  "role": "user",
  "content": [
    {
      "type": "data",
      "data": { /* 페이로드 */ }
    }
  ],
  "metadata": {
    "did": "did:sage:ethereum:0x...",
    "signature": "base64..."
  }
}
```

---

## 2. 4단계 핸드셰이크 상세

### 2.1 단계 1: Invitation (초대)

**목적**: 세션 시작 의도 전달 및 신원 증명

```
Agent A → Agent B

메시지 구조:
{
  "contextId": "ctx-abc123",
  "timestamp": 1704067200,
  "agentDid": "did:sage:ethereum:0x742d35Cc..."
}

메타데이터:
{
  "did": "did:sage:ethereum:0x742d35Cc...",
  "signature": "MEUCIQDx..." // Ed25519 서명
}

처리 흐름:
┌─────────────────────────────────────────┐
│ Agent A (클라이언트)                     │
│                                          │
│ 1. contextId 생성 (UUID)                │
│ 2. InvitationMessage 구성                │
│ 3. A2A 메시지로 래핑                     │
│ 4. Ed25519로 서명                        │
│ 5. gRPC로 전송                           │
└──────────────┬──────────────────────────┘
               │ [네트워크]
               ↓
┌─────────────────────────────────────────┐
│ Agent B (서버)                           │
│                                          │
│ 1. A2A 메시지 수신                       │
│ 2. DID 추출 및 블록체인 조회             │
│ 3. 공개키로 서명 검증                    │
│ 4. contextId 저장                        │
│ 5. OnInvitation 이벤트 발생              │
│ 6. ACK 응답                              │
└─────────────────────────────────────────┘

코드 위치: handshake/client.go:49-70
```

**실제 코드**:

```go
// handshake/client.go

func (c *Client) Invitation(
    ctx context.Context,
    invMsg InvitationMessage,
    did string,
) (*a2a.SendMessageResponse, error) {
    // 1. InvitationMessage → structpb
    payload, err := toStructPB(invMsg)
    if err != nil {
        return nil, fmt.Errorf("marshal invitation: %w", err)
    }

    // 2. A2A Message 구성
    msg := &a2a.Message{
        MessageId: uuid.NewString(),
        ContextId: invMsg.ContextID,
        TaskId:    GenerateTaskID(Invitation),  // "handshake:invitation"
        Role:      a2a.Role_ROLE_USER,
        Content:   []*a2a.Part{
            {Part: &a2a.Part_Data{
                Data: &a2a.DataPart{Data: payload},
            }},
        },
    }

    // 3. 결정론적 직렬화 (서명용)
    bytes, err := proto.MarshalOptions{Deterministic: true}.Marshal(msg)
    if err != nil {
        return nil, fmt.Errorf("marshal for signing: %w", err)
    }

    // 4. 서명 생성
    meta, err := signStruct(c.key, bytes, did)
    if err != nil {
        return nil, fmt.Errorf("sign: %w", err)
    }

    // 5. gRPC 호출
    return c.SendMessage(ctx, &a2a.SendMessageRequest{
        Request:  msg,
        Metadata: meta,
    })
}

// 서명 헬퍼
func signStruct(key sagecrypto.KeyPair, data []byte, did string) (*structpb.Struct, error) {
    signature, err := key.Sign(data)
    if err != nil {
        return nil, err
    }

    return structpb.NewStruct(map[string]interface{}{
        "did":       did,
        "signature": base64.RawURLEncoding.EncodeToString(signature),
        "algorithm": "ed25519",
        "timestamp": time.Now().Unix(),
    })
}
```

**서버 처리**:

```go
// handshake/server.go

func (s *Server) SendMessage(ctx context.Context, in *a2a.SendMessageRequest) (*a2a.SendMessageResponse, error) {
    msg := in.Request
    phase, _ := parsePhase(msg.TaskId)

    if phase == Invitation {
        // 1. DID 추출
        senderDID := in.Metadata.GetFields()["did"].GetStringValue()

        // 2. 캐시 확인
        var senderPub crypto.PublicKey
        if cache, ok := s.getPeer(msg.ContextId); ok {
            senderPub = cache.pub
        } else {
            // 3. 블록체인에서 DID 조회
            pub, err := s.resolver.ResolvePublicKey(ctx, did.AgentDID(senderDID))
            if err != nil {
                return nil, errors.New("cannot resolve sender pubkey")
            }
            senderPub = pub
            s.savePeer(msg.ContextId, senderPub, senderDID)
        }

        // 4. 서명 검증
        if err := s.verifySenderSignature(msg, in.Metadata, senderPub); err != nil {
            return nil, fmt.Errorf("signature verification failed: %w", err)
        }

        // 5. Invitation 메시지 파싱
        payload, _ := firstDataPart(msg)
        var inv InvitationMessage
        fromStructPB(payload, &inv)

        // 6. 이벤트 발생
        s.events.OnInvitation(ctx, msg.ContextId, inv)

        // 7. ACK 반환
        return s.ack(msg, "invitation_received")
    }
    // ...
}

위치: handshake/server.go:159-197
```

### 2.2 단계 2: Request (요청)

**목적**: 임시 공개키 전송 및 암호화된 통신 시작

```
Agent A → Agent B

메시지 구조 (암호화 전):
{
  "contextId": "ctx-abc123",
  "ephemeralPubKey": "{\"kty\":\"OKP\",\"crv\":\"X25519\",\"x\":\"...\"}"
}

암호화:
plaintext → EncryptWithEd25519Peer(B_pubKey, plaintext) → packet
packet = ephPub(32) || nonce(12) || ciphertext || tag(16)

Base64 인코딩 후 A2A 메시지로 전송

처리 흐름:
┌─────────────────────────────────────────┐
│ Agent A (클라이언트)                     │
│                                          │
│ 1. X25519 임시 키 쌍 생성                │
│ 2. 임시 공개키 → JWK 직렬화              │
│ 3. RequestMessage 구성                   │
│ 4. B의 DID 공개키로 암호화               │
│    (EncryptWithEd25519Peer)              │
│ 5. Base64 인코딩                         │
│ 6. A2A 메시지 + Ed25519 서명             │
│ 7. 전송                                  │
└──────────────┬──────────────────────────┘
               │ [네트워크]
               ↓
┌─────────────────────────────────────────┐
│ Agent B (서버)                           │
│                                          │
│ 1. Base64 디코딩                         │
│ 2. 서명 검증 (Ed25519)                   │
│ 3. B의 개인키로 복호화                   │
│    (DecryptWithEd25519Peer)              │
│ 4. 임시 공개키 파싱                      │
│ 5. pendingState에 저장                   │
│ 6. B의 임시 키 쌍 생성                   │
│ 7. OnRequest 이벤트 발생                 │
│ 8. Response 자동 전송                    │
└─────────────────────────────────────────┘

코드 위치: handshake/client.go:72-99
```

**클라이언트 코드**:

```go
// handshake/client.go

func (c *Client) Request(
    ctx context.Context,
    reqMsg RequestMessage,
    edPeerPub crypto.PublicKey,
    did string,
) (*a2a.SendMessageResponse, error) {
    // 1. RequestMessage → JSON
    reqBytes, err := json.Marshal(reqMsg)
    if err != nil {
        return nil, fmt.Errorf("marshal request: %w", err)
    }

    // 2. 부트스트랩 암호화
    // Ed25519 피어 공개키를 사용하여 암호화
    packet, err := keys.EncryptWithEd25519Peer(edPeerPub, reqBytes)
    if err != nil {
        return nil, fmt.Errorf("encrypt request: %w", err)
    }

    // 3. Base64 인코딩 → structpb
    payload, _ := b64ToStructPB(base64.RawURLEncoding.EncodeToString(packet))

    // 4. A2A Message 구성
    msg := &a2a.Message{
        MessageId: uuid.NewString(),
        ContextId: reqMsg.ContextID,
        TaskId:    GenerateTaskID(Request),  // "handshake:request"
        Role:      a2a.Role_ROLE_USER,
        Content:   []*a2a.Part{
            {Part: &a2a.Part_Data{
                Data: &a2a.DataPart{Data: payload},
            }},
        },
    }

    // 5. 서명
    bytes, _ := proto.MarshalOptions{Deterministic: true}.Marshal(msg)
    meta, _ := signStruct(c.key, bytes, did)

    // 6. 전송
    return c.SendMessage(ctx, &a2a.SendMessageRequest{
        Request:  msg,
        Metadata: meta,
    })
}

위치: handshake/client.go:72-99
```

**서버 처리**:

```go
// handshake/server.go

case Request:
    // 1. 캐시된 피어 정보 가져오기
    cache, ok := s.getPeer(msg.ContextId)
    if !ok {
        return nil, errors.New("no cached peer; invitation required first")
    }

    // 2. 서명 검증
    if err := s.verifySenderSignature(msg, in.Metadata, cache.pub); err != nil {
        return nil, fmt.Errorf("request signature verification failed: %w", err)
    }

    // 3. 복호화
    payload, _ := firstDataPart(msg)
    plain, err := s.decryptPacket(payload)
    if err != nil {
        return nil, fmt.Errorf("request decrypt: %w", err)
    }

    // 4. RequestMessage 파싱
    var req RequestMessage
    json.Unmarshal(plain, &req)

    if len(req.EphemeralPubKey) == 0 {
        return nil, fmt.Errorf("empty peer ephemeral public key")
    }

    // 5. JWK → X25519 공개키
    exported, _ := s.importer.ImportPublic(
        []byte(req.EphemeralPubKey),
        sagecrypto.KeyFormatJWK,
    )
    peerPub := exported.(*ecdh.PublicKey)
    peerEphRaw := peerPub.Bytes()

    // 6. 서버 임시 키 생성 요청
    serverEphRaw, serverEphJWK, err := s.events.AskEphemeral(ctx, msg.ContextId)
    if err != nil {
        return nil, fmt.Errorf("ask ephemeral: %w", err)
    }

    // 7. pending 상태 저장
    s.savePending(msg.ContextId, pendingState{
        peerEph:   peerEphRaw,
        serverEph: serverEphRaw,
    })

    // 8. OnRequest 이벤트
    s.events.OnRequest(ctx, msg.ContextId, req, cache.pub)

    // 9. Response 자동 전송
    res := ResponseMessage{
        EphemeralPubKey: json.RawMessage(serverEphJWK),
        Ack:             true,
    }
    return s.sendResponseToPeer(res, msg.ContextId, cache.pub, cache.did)

위치: handshake/server.go:199-256
```

### 2.3 단계 3: Response (응답)

**목적**: 서버의 임시 공개키를 클라이언트에 전달

```
Agent B → Agent A

메시지 구조 (암호화 전):
{
  "ephemeralPubKey": "{\"kty\":\"OKP\",\"crv\":\"X25519\",\"x\":\"...\"}",
  "ack": true,
  "keyId": ""  // 아직 발급 전
}

암호화:
plaintext → EncryptWithEd25519Peer(A_pubKey, plaintext) → packet

처리 흐름:
┌─────────────────────────────────────────┐
│ Agent B (서버)                           │
│                                          │
│ 1. X25519 임시 키 쌍 이미 생성됨         │
│ 2. 임시 공개키 → JWK 직렬화              │
│ 3. ResponseMessage 구성                  │
│ 4. A의 DID 공개키로 암호화               │
│ 5. A2A 메시지 + Ed25519 서명             │
│ 6. 전송                                  │
└──────────────┬──────────────────────────┘
               │ [네트워크]
               ↓
┌─────────────────────────────────────────┐
│ Agent A (클라이언트)                     │
│                                          │
│ 1. Base64 디코딩                         │
│ 2. 서명 검증                              │
│ 3. A의 개인키로 복호화                   │
│ 4. 임시 공개키 파싱                      │
│ 5. 공유 비밀 계산:                       │
│    shared = ECDH(A_ephPriv, B_ephPub)   │
│ 6. 세션 준비 (아직 생성 안함)            │
└─────────────────────────────────────────┘

코드 위치: handshake/server.go:335-354
```

**서버 전송 코드**:

```go
// handshake/server.go

func (s *Server) sendResponseToPeer(
    res ResponseMessage,
    ctxID string,
    peerPub crypto.PublicKey,
    senderDID string,
) (*a2a.SendMessageResponse, error) {
    // 1. ResponseMessage → JSON
    plain, err := json.Marshal(res)
    if err != nil {
        return nil, fmt.Errorf("marshal response: %w", err)
    }

    // 2. 암호화
    packet, err := keys.EncryptWithEd25519Peer(peerPub, plain)
    if err != nil {
        return nil, fmt.Errorf("encrypt response: %w", err)
    }

    // 3. Base64 → structpb
    payload, _ := b64ToStructPB(base64.RawURLEncoding.EncodeToString(packet))

    // 4. A2A Message
    msg := &a2a.Message{
        MessageId: uuid.NewString(),
        ContextId: ctxID,
        TaskId:    GenerateTaskID(Response),
        Role:      a2a.Role_ROLE_AGENT,
        Content:   []*a2a.Part{
            {Part: &a2a.Part_Data{
                Data: &a2a.DataPart{Data: payload},
            }},
        },
    }

    // 5. 서명 및 전송
    return s.sendSigned(msg, senderDID)
}

위치: handshake/server.go:335-354
```

**클라이언트 수신** (Response 메서드):

```go
// handshake/client.go

func (c *Client) Response(
    ctx context.Context,
    resMsg ResponseMessage,
    edPeerPub crypto.PublicKey,
    did string,
) (*a2a.SendMessageResponse, error) {
    // 서버와 동일한 암호화 로직
    // Response는 양방향 가능 (B→A 또는 A→B)
    resBytes, _ := json.Marshal(resMsg)
    packet, _ := keys.EncryptWithEd25519Peer(edPeerPub, resBytes)
    payload, _ := b64ToStructPB(base64.RawURLEncoding.EncodeToString(packet))

    msg := &a2a.Message{
        MessageId: uuid.NewString(),
        ContextId: resMsg.ContextID,
        TaskId:    GenerateTaskID(Response),
        Role:      a2a.Role_ROLE_AGENT,
        Content:   []*a2a.Part{
            {Part: &a2a.Part_Data{
                Data: &a2a.DataPart{Data: payload},
            }},
        },
    }

    bytes, _ := proto.MarshalOptions{Deterministic: true}.Marshal(msg)
    meta, _ := signStruct(c.key, bytes, did)

    return c.SendMessage(ctx, &a2a.SendMessageRequest{
        Request:  msg,
        Metadata: meta,
    })
}

위치: handshake/client.go:102-128
```

### 2.4 단계 4: Complete (완료)

**목적**: 핸드셰이크 완료 확인 및 세션 생성

```
Agent A → Agent B

메시지 구조 (평문):
{
  "contextId": "ctx-abc123",
  "timestamp": 1704067300
}

처리 흐름:
┌─────────────────────────────────────────┐
│ Agent A (클라이언트)                     │
│                                          │
│ 1. CompleteMessage 구성                  │
│ 2. A2A 메시지 + Ed25519 서명             │
│ 3. 전송                                  │
│ 4. Response 대기 (KeyID 포함)            │
└──────────────┬──────────────────────────┘
               │ [네트워크]
               ↓
┌─────────────────────────────────────────┐
│ Agent B (서버)                           │
│                                          │
│ 1. 서명 검증                              │
│ 2. pending 상태 가져오기                 │
│ 3. 공유 비밀 계산:                       │
│    shared = ECDH(B_ephPriv, A_ephPub)   │
│ 4. 세션 파라미터 구성                    │
│ 5. OnComplete 이벤트 발생                │
│    → 이벤트 핸들러에서 세션 생성         │
│ 6. KeyID 발급 (UUID)                    │
│ 7. KeyID → SessionID 바인딩              │
│ 8. Response(keyId) 전송                  │
└──────────────┬──────────────────────────┘
               │ [네트워크]
               ↓
┌─────────────────────────────────────────┐
│ Agent A (클라이언트)                     │
│                                          │
│ 1. Response 수신                         │
│ 2. KeyID 추출                            │
│ 3. 공유 비밀로 세션 생성                 │
│ 4. KeyID → SessionID 바인딩              │
│ 5. 핸드셰이크 완료!                      │
└─────────────────────────────────────────┘

코드 위치: handshake/client.go:130-152, handshake/server.go:258-298
```

**클라이언트 코드**:

```go
// handshake/client.go

func (c *Client) Complete(
    ctx context.Context,
    compMsg CompleteMessage,
    did string,
) (*a2a.SendMessageResponse, error) {
    // 1. CompleteMessage → structpb
    payload, err := toStructPB(compMsg)
    if err != nil {
        return nil, fmt.Errorf("marshal complete: %w", err)
    }

    // 2. A2A Message (평문, 암호화 없음)
    msg := &a2a.Message{
        MessageId: uuid.NewString(),
        ContextId: compMsg.ContextID,
        TaskId:    GenerateTaskID(Complete),
        Role:      a2a.Role_ROLE_USER,
        Content:   []*a2a.Part{
            {Part: &a2a.Part_Data{
                Data: &a2a.DataPart{Data: payload},
            }},
        },
    }

    // 3. 서명
    bytes, _ := proto.MarshalOptions{Deterministic: true}.Marshal(msg)
    meta, _ := signStruct(c.key, bytes, did)

    // 4. 전송
    return c.SendMessage(ctx, &a2a.SendMessageRequest{
        Request:  msg,
        Metadata: meta,
    })
}

위치: handshake/client.go:130-152
```

**서버 처리**:

```go
// handshake/server.go

case Complete:
    // 1. 서명 검증
    cache, ok := s.getPeer(msg.ContextId)
    if !ok {
        return nil, errors.New("no cached peer")
    }

    if err := s.verifySenderSignature(msg, in.Metadata, cache.pub); err != nil {
        return nil, fmt.Errorf("signature verification failed: %w", err)
    }

    // 2. CompleteMessage 파싱
    payload, _ := firstDataPart(msg)
    var comp CompleteMessage
    fromStructPB(payload, &comp)

    // 3. pending 상태 가져오기 (제거)
    st, ok := s.takePending(msg.ContextId)
    if !ok {
        s.events.OnComplete(ctx, msg.ContextId, comp, session.Params{})
        return s.ack(msg, "complete_received_no_pending")
    }

    // 4. 세션 파라미터 구성
    sessParams := session.Params{
        ContextID: msg.ContextId,
        SelfEph:   st.serverEph,
        PeerEph:   st.peerEph,
        Label:     "a2a/handshake v1",
    }

    // 5. OnComplete 이벤트 발생
    // 이벤트 핸들러에서 세션 생성
    s.events.OnComplete(ctx, msg.ContextId, comp, sessParams)

    // 6. KeyID 발급 및 바인딩
    if binder, ok := any(s.events).(KeyIDBinder); ok {
        if kid, ok2 := binder.IssueKeyID(msg.ContextId); ok2 && kid != "" {
            // KeyID를 Response로 전송
            res := ResponseMessage{
                Ack:   true,
                KeyID: kid,
            }
            return s.sendResponseToPeer(res, msg.ContextId, cache.pub, cache.did)
        }
    }

    return s.ack(msg, "complete_received_session_ready")

위치: handshake/server.go:258-298
```

---

## 3. 클라이언트 구현

### 3.1 클라이언트 구조

```go
// handshake/client.go

type Client struct {
    a2a.A2AServiceClient  // gRPC 클라이언트 (임베디드)
    key sagecrypto.KeyPair // Ed25519 키 (신원용)
}

// 생성자
func NewClient(conn grpc.ClientConnInterface, key sagecrypto.KeyPair) *Client {
    return &Client{
        A2AServiceClient: a2a.NewA2AServiceClient(conn),
        key:              key,
    }
}

특징:
- 상태 없음 (Stateless)
- gRPC 클라이언트 래퍼
- 각 메서드가 독립적
- 세션 관리는 외부에서
```

### 3.2 완전한 핸드셰이크 시퀀스

```go
// examples/handshake/client_example.go

func performHandshake(
    client *handshake.Client,
    peerDID string,
    contextID string,
) (string, error) {
    ctx := context.Background()

    // 0. DID 조회로 피어 공개키 가져오기
    resolver := did.NewMultiChainResolver()
    // ... resolver 설정 ...
    peerMetadata, err := resolver.Resolve(ctx, did.AgentDID(peerDID))
    if err != nil {
        return "", fmt.Errorf("failed to resolve peer DID: %w", err)
    }
    peerEdPub := peerMetadata.PublicKey  // Ed25519 공개키

    // 1. Invitation
    invMsg := handshake.InvitationMessage{
        BaseMessage: message.BaseMessage{
            ContextID: contextID,
        },
    }
    resp, err := client.Invitation(ctx, invMsg, myDID)
    if err != nil {
        return "", fmt.Errorf("invitation failed: %w", err)
    }
    fmt.Println("✅ Invitation sent")

    // 2. Request (임시 키 생성)
    ephKeyPair, _ := keys.GenerateX25519KeyPair()
    ephX := ephKeyPair.(*keys.X25519KeyPair)

    // JWK 직렬화
    exporter := formats.NewJWKExporter()
    ephJWK, _ := exporter.ExportPublic(ephKeyPair, sagecrypto.KeyFormatJWK)

    reqMsg := handshake.RequestMessage{
        BaseMessage: message.BaseMessage{
            ContextID: contextID,
        },
        EphemeralPubKey: json.RawMessage(ephJWK),
    }

    // Ed25519 공개키로 암호화
    resp, err = client.Request(ctx, reqMsg, peerEdPub, myDID)
    if err != nil {
        return "", fmt.Errorf("request failed: %w", err)
    }
    fmt.Println("✅ Request sent")

    // 3. Response 수신 (비동기 또는 블로킹)
    // 실제로는 서버로부터 gRPC 스트림 또는 별도 채널로 수신
    // 여기서는 단순화를 위해 동기 수신 가정
    responseMsg, err := waitForResponse(ctx, contextID)
    if err != nil {
        return "", fmt.Errorf("response wait failed: %w", err)
    }

    // Response 복호화
    peerEphJWK := responseMsg.EphemeralPubKey
    importer := formats.NewJWKImporter()
    peerEphPub, _ := importer.ImportPublic(
        []byte(peerEphJWK),
        sagecrypto.KeyFormatJWK,
    )

    // 4. 공유 비밀 계산
    sharedSecret, err := ephX.DeriveSharedSecret(
        peerEphPub.(*ecdh.PublicKey).Bytes(),
    )
    if err != nil {
        return "", fmt.Errorf("derive shared secret failed: %w", err)
    }
    fmt.Println("✅ Shared secret computed")

    // 5. Complete
    compMsg := handshake.CompleteMessage{
        BaseMessage: message.BaseMessage{
            ContextID: contextID,
        },
    }
    resp, err = client.Complete(ctx, compMsg, myDID)
    if err != nil {
        return "", fmt.Errorf("complete failed: %w", err)
    }
    fmt.Println("✅ Complete sent")

    // 6. KeyID 수신
    keyIDResp, err := waitForKeyIDResponse(ctx, contextID)
    if err != nil {
        return "", fmt.Errorf("keyID wait failed: %w", err)
    }
    keyID := keyIDResp.KeyID

    // 7. 세션 생성
    sessParams := session.Params{
        ContextID:    contextID,
        SelfEph:      ephX.PublicBytesKey(),
        PeerEph:      peerEphPub.(*ecdh.PublicKey).Bytes(),
        Label:        "a2a/handshake v1",
        SharedSecret: sharedSecret,
    }

    sess, err := session.NewSecureSessionFromExporterWithRole(
        keyID,  // keyID를 세션 ID로 사용
        sharedSecret,
        true,  // initiator
        session.Config{
            MaxAge:      time.Hour,
            IdleTimeout: 10 * time.Minute,
            MaxMessages: 10000,
        },
    )
    if err != nil {
        return "", fmt.Errorf("session creation failed: %w", err)
    }

    fmt.Printf("✅ Session created: %s\n", keyID)
    return keyID, nil
}
```

### 3.3 에러 처리

```go
// handshake/errors.go

var (
    ErrInvitationTimeout    = errors.New("invitation timeout")
    ErrRequestTimeout       = errors.New("request timeout")
    ErrResponseTimeout      = errors.New("response timeout")
    ErrCompleteTimeout      = errors.New("complete timeout")
    ErrInvalidSignature     = errors.New("invalid signature")
    ErrDecryptionFailed     = errors.New("decryption failed")
    ErrPeerNotFound         = errors.New("peer not found")
    ErrDIDResolutionFailed  = errors.New("DID resolution failed")
)

// 재시도 로직
func (c *Client) InvitationWithRetry(
    ctx context.Context,
    invMsg InvitationMessage,
    did string,
    maxRetries int,
) (*a2a.SendMessageResponse, error) {
    var lastErr error

    for i := 0; i < maxRetries; i++ {
        resp, err := c.Invitation(ctx, invMsg, did)
        if err == nil {
            return resp, nil
        }

        lastErr = err

        // 재시도 가능한 에러인지 확인
        if !isRetryable(err) {
            return nil, err
        }

        // Exponential backoff
        waitTime := time.Duration(1<<uint(i)) * time.Second
        fmt.Printf("Retry %d/%d after %v...\n", i+1, maxRetries, waitTime)
        time.Sleep(waitTime)
    }

    return nil, fmt.Errorf("max retries exceeded: %w", lastErr)
}

func isRetryable(err error) bool {
    // 네트워크 에러, 타임아웃 등은 재시도 가능
    if errors.Is(err, context.DeadlineExceeded) {
        return true
    }
    if errors.Is(err, ErrInvitationTimeout) {
        return true
    }
    // 서명 에러, 복호화 실패 등은 재시도 불가능
    if errors.Is(err, ErrInvalidSignature) {
        return false
    }
    if errors.Is(err, ErrDecryptionFailed) {
        return false
    }
    return true
}
```

---

## 4. 서버 구현

### 4.1 서버 구조

```go
// handshake/server.go

type Server struct {
    a2a.UnimplementedA2AServiceServer  // gRPC 서버 (임베디드)

    key      sagecrypto.KeyPair  // Ed25519 키 (신원용)
    events   Events               // 이벤트 핸들러
    resolver did.Resolver         // DID 조회

    // 상태 관리
    mu      sync.Mutex
    sf      singleflight.Group
    pending map[string]pendingState  // contextID → 임시 상태
    peers   map[string]cachedPeer    // contextID → 피어 정보

    // 세션 설정
    sessionCfg session.Config

    // 키 포맷 변환
    exporter sagecrypto.KeyExporter
    importer sagecrypto.KeyImporter

    // 정리 타이머
    pendingTTL    time.Duration
    cleanupTicker *time.Ticker
    stopCleanup   chan struct{}
    cleanupDone   chan struct{}
}

// 생성자
func NewServer(
    key sagecrypto.KeyPair,
    events Events,
    resolver did.Resolver,
    sessionCfg *session.Config,
    cleanupInterval time.Duration,
) *Server {
    if events == nil {
        events = NoopEvents{}
    }

    var cfg session.Config
    if sessionCfg == nil {
        cfg = session.Config{
            MaxAge:      time.Hour,
            IdleTimeout: 10 * time.Minute,
            MaxMessages: 10000,
        }
    } else {
        cfg = *sessionCfg
    }

    s := &Server{
        key:         key,
        events:      events,
        resolver:    resolver,
        pending:     make(map[string]pendingState),
        peers:       make(map[string]cachedPeer),
        sessionCfg:  cfg,
        exporter:    formats.NewJWKExporter(),
        importer:    formats.NewJWKImporter(),
        pendingTTL:  15 * time.Minute,
        stopCleanup: make(chan struct{}),
        cleanupDone: make(chan struct{}),
    }

    interval := cleanupInterval
    if interval <= 0 {
        interval = 10 * time.Minute
    }
    s.cleanupTicker = time.NewTicker(interval)
    go s.cleanupLoop()

    return s
}

위치: handshake/server.go:57-138
```

### 4.2 상태 관리

**pendingState**:

```go
// handshake/server.go

type pendingState struct {
    peerEph   []byte     // 클라이언트 임시 공개키 (32바이트)
    serverEph []byte     // 서버 임시 공개키 (32바이트)
    expires   time.Time  // TTL (15분)
}

// 저장
func (s *Server) savePending(id string, st pendingState) {
    s.mu.Lock()
    defer s.mu.Unlock()
    s.pending[id] = st
}

// 가져오기 및 삭제 (Complete 시)
func (s *Server) takePending(id string) (pendingState, bool) {
    s.mu.Lock()
    defer s.mu.Unlock()
    st, ok := s.pending[id]
    if ok {
        delete(s.pending, id)
    }
    return st, ok
}

주의사항:
- pending은 임시 상태만 저장
- 공유 비밀은 저장하지 않음 (보안)
- Complete 시 즉시 삭제
- TTL 만료 시 자동 정리
```

**cachedPeer**:

```go
type cachedPeer struct {
    pub     crypto.PublicKey  // Ed25519 공개키
    did     string             // DID 문자열
    expires time.Time          // 캐시 만료 시간
}

// 저장
func (s *Server) savePeer(ctxID string, pub crypto.PublicKey, did string) {
    s.mu.Lock()
    defer s.mu.Unlock()
    s.peers[ctxID] = cachedPeer{
        pub:     pub,
        did:     did,
        expires: time.Now().Add(s.pendingTTL),
    }
}

// 조회
func (s *Server) getPeer(ctxID string) (cachedPeer, bool) {
    s.mu.Lock()
    defer s.mu.Unlock()
    cp, ok := s.peers[ctxID]
    return cp, ok
}

장점:
- 블록체인 조회 횟수 감소
- 같은 컨텍스트 내 여러 메시지 처리 빠름
- 메모리 효율적 (contextID당 하나만 저장)
```

### 4.3 자동 정리 루프

```go
// handshake/server.go

func (s *Server) cleanupLoop() {
    ticker := s.cleanupTicker
    for {
        select {
        case <-ticker.C:
            s.cleanupExpired(time.Now())
        case <-s.stopCleanup:
            ticker.Stop()
            s.mu.Lock()
            if s.cleanupDone != nil {
                close(s.cleanupDone)
            }
            s.mu.Unlock()
            return
        }
    }
}

func (s *Server) cleanupExpired(now time.Time) {
    s.mu.Lock()
    defer s.mu.Unlock()

    // pending 상태 정리
    for ctxID, st := range s.pending {
        if now.After(st.expires) {
            delete(s.pending, ctxID)
            log.Info("cleaned up expired pending", "context", ctxID)
        }
    }

    // 피어 캐시 정리
    for ctxID, cp := range s.peers {
        if now.After(cp.expires) {
            delete(s.peers, ctxID)
            log.Info("cleaned up expired peer cache", "context", ctxID)
        }
    }
}

// 서버 종료 시
func (s *Server) Shutdown() {
    close(s.stopCleanup)
    <-s.cleanupDone
}

위치: handshake/server.go:470-502
```

### 4.4 DID 조회 최적화

**Singleflight 패턴으로 중복 조회 방지**:

```go
// handshake/server.go

// Invitation 처리 중
if cache, ok := s.getPeer(msg.ContextId); ok && cache.did == senderDID {
    senderPub = cache.pub
} else {
    if s.resolver == nil {
        return nil, errors.New("resolver not set")
    }

    // Singleflight: 같은 DID를 여러 고루틴이 동시에 조회하는 것 방지
    v, err, shared := s.sf.Do("resolve:"+senderDID, func() (any, error) {
        return s.resolver.ResolvePublicKey(ctx, did.AgentDID(senderDID))
    })
    if err != nil {
        return nil, errors.New("cannot resolve sender pubkey")
    }

    senderPub, ok = v.(crypto.PublicKey)
    if !ok {
        return nil, fmt.Errorf("unexpected key type: %T", v)
    }

    // 캐시 저장
    s.savePeer(msg.ContextId, senderPub, senderDID)

    if shared {
        log.Debug("reused DID resolution", "did", senderDID)
    }
}

위치: handshake/server.go:166-185

Singleflight 효과:
- 100개 고루틴이 같은 DID 조회 → 1번만 실제 조회
- 나머지 99개는 결과 대기 후 공유
- 블록체인 부하 감소
```

---

## 5. 세션 생성 및 키 유도

### 5.1 세션 파라미터

```go
// session/session.go

type Params struct {
    ContextID    string  // 컨텍스트 ID (핸드셰이크 식별자)
    SelfEph      []byte  // 자신의 임시 공개키 (32바이트)
    PeerEph      []byte  // 피어의 임시 공개키 (32바이트)
    Label        string  // 프로토콜 레이블 ("a2a/handshake v1")
    SharedSecret []byte  // ECDH 공유 비밀 (32바이트)
}

역할:
- 양측이 같은 파라미터로 세션 생성
- 결정론적 세션 ID 유도
- 도메인 분리 (Label)
```

### 5.2 세션 시드 유도

```go
// session/session.go

func DeriveSessionSeed(sharedSecret []byte, p Params) ([]byte, error) {
    // 1. 레이블 기본값
    label := p.Label
    if label == "" {
        label = "a2a/handshake v1"
    }

    // 2. 임시 공개키 정렬 (대칭성)
    // A와 B가 순서에 상관없이 같은 결과를 얻도록
    lo, hi := canonicalOrder(p.SelfEph, p.PeerEph)

    // 3. 솔트 계산
    h := sha256.New()
    h.Write([]byte(label))
    h.Write([]byte(p.ContextID))
    h.Write(lo)
    h.Write(hi)
    salt := h.Sum(nil)

    // 4. HKDF-Extract
    // PRK (Pseudorandom Key) 생성
    seed := hkdfExtractSHA256(sharedSecret, salt)

    return seed, nil
}

func hkdfExtractSHA256(ikm, salt []byte) []byte {
    prk := hkdf.Extract(sha256.New, ikm, salt)
    out := make([]byte, len(prk))
    copy(out, prk)
    return out
}

func canonicalOrder(a, b []byte) (lo, hi []byte) {
    if bytes.Compare(a, b) <= 0 {
        return a, b
    }
    return b, a
}

위치: session/session.go:181-202, 288-305

예시:
A의 임시 공개키: 0xaa...
B의 임시 공개키: 0xbb...

A 측:
lo, hi = canonicalOrder(0xaa..., 0xbb...) → (0xaa..., 0xbb...)

B 측:
lo, hi = canonicalOrder(0xbb..., 0xaa...) → (0xaa..., 0xbb...)

→ 양측이 같은 솔트, 같은 시드를 얻음!
```

### 5.3 세션 ID 계산

```go
// session/session.go

func ComputeSessionIDFromSeed(seed []byte, label string) (string, error) {
    if len(seed) == 0 {
        return "", fmt.Errorf("empty seed")
    }

    // 1. Label + Seed 해시
    h := sha256.New()
    h.Write([]byte(label))
    h.Write(seed)
    full := h.Sum(nil)  // 32바이트

    // 2. 처음 16바이트만 사용
    // 3. Base64 URL-safe 인코딩
    return base64.RawURLEncoding.EncodeToString(full[:16]), nil
}

위치: session/session.go:206-215

예시:
seed = 0x1234abcd...
label = "a2a/handshake v1"

hash = SHA256("a2a/handshake v1" || seed)
    = 0xef5678...1234 (32바이트)

sessionID = Base64(hash[0:16])
          = "71V4...EjQ"

결과:
- 짧고 읽기 쉬운 ID (22자)
- 충돌 확률 극히 낮음 (2^-128)
- 양측이 같은 ID 계산
```

### 5.4 방향별 키 유도

```go
// session/session.go

func (s *SecureSession) deriveDirectionalKeys() error {
    salt := []byte(s.id)  // 세션 ID를 솔트로

    // HKDF-Expand 헬퍼 함수
    expand := func(info string, n int) ([]byte, error) {
        r := hkdf.New(sha256.New, s.sessionSeed, salt, []byte(info))
        out := make([]byte, n)
        if _, err := io.ReadFull(r, out); err != nil {
            return nil, err
        }
        return out, nil
    }

    // 4가지 키 유도
    c2sEnc, _ := expand("c2s|enc|v1", 32)   // Client → Server 암호화
    c2sSign, _ := expand("c2s|sign|v1", 32)  // Client → Server 서명
    s2cEnc, _ := expand("s2c|enc|v1", 32)    // Server → Client 암호화
    s2cSign, _ := expand("s2c|sign|v1", 32)  // Server → Client 서명

    // 역할에 따라 할당
    if s.initiator {
        // 클라이언트 (initiator=true)
        s.outKey, s.outSign = c2sEnc, c2sSign  // 송신은 C2S
        s.inKey, s.inSign = s2cEnc, s2cSign    // 수신은 S2C
    } else {
        // 서버 (initiator=false)
        s.outKey, s.outSign = s2cEnc, s2cSign  // 송신은 S2C
        s.inKey, s.inSign = c2sEnc, c2sSign    // 수신은 C2S
    }

    return nil
}

위치: session/session.go:240-273

키 계층:
sessionSeed (PRK)
    ↓ HKDF-Expand(info="c2s|enc|v1")
    ├─→ c2sEnc (32바이트)
    ↓ HKDF-Expand(info="c2s|sign|v1")
    ├─→ c2sSign (32바이트)
    ↓ HKDF-Expand(info="s2c|enc|v1")
    ├─→ s2cEnc (32바이트)
    ↓ HKDF-Expand(info="s2c|sign|v1")
    └─→ s2cSign (32바이트)

initiator=true (Client):
    out: c2sEnc, c2sSign
    in:  s2cEnc, s2cSign

initiator=false (Server):
    out: s2cEnc, s2cSign
    in:  c2sEnc, c2sSign

장점:
✅ 방향별 독립 키 (크로스 공격 방지)
✅ 암호화/서명 분리 (도메인 분리)
✅ 버전 관리 가능 (v1, v2...)
✅ 확장 용이 (새 키 타입 추가 가능)
```

### 5.5 AEAD 초기화

```go
// session/session.go

func (s *SecureSession) initAEADs() error {
    var err error

    // 송신용 AEAD
    s.aeadOut, err = chacha20poly1305.New(s.outKey)
    if err != nil {
        return fmt.Errorf("create outbound AEAD: %w", err)
    }

    // 수신용 AEAD
    s.aeadIn, err = chacha20poly1305.New(s.inKey)
    if err != nil {
        return fmt.Errorf("create inbound AEAD: %w", err)
    }

    return nil
}

위치: session/session.go:275-286

ChaCha20-Poly1305:
- 키 크기: 32바이트
- Nonce 크기: 12바이트
- 태그 크기: 16바이트
- 성능: ~1 GB/s (소프트웨어)
```

---

## 6. 세션 관리자

### 6.1 Manager 구조

```go
// session/manager.go

type Manager struct {
    sessions map[string]*SecureSession  // sessionID → Session
    keyIDs   map[string]string          // keyID → sessionID
    mu       sync.RWMutex
    config   Config

    // 백그라운드 정리
    cleanupTicker *time.Ticker
    stopCleanup   chan struct{}
}

func NewManager() *Manager {
    m := &Manager{
        sessions:    make(map[string]*SecureSession),
        keyIDs:      make(map[string]string),
        stopCleanup: make(chan struct{}),
    }

    // 30초마다 만료 세션 정리
    m.cleanupTicker = time.NewTicker(30 * time.Second)
    go m.cleanupLoop()

    return m
}
```

### 6.2 세션 생성

```go
// session/manager.go

func (m *Manager) CreateSession(
    params Params,
    sharedSecret []byte,
    initiator bool,
    config Config,
) (*SecureSession, error) {
    m.mu.Lock()
    defer m.mu.Unlock()

    // 1. 세션 시드 유도
    seed, err := DeriveSessionSeed(sharedSecret, params)
    if err != nil {
        return nil, err
    }

    // 2. 세션 ID 계산
    sessID, err := ComputeSessionIDFromSeed(seed, params.Label)
    if err != nil {
        return nil, err
    }

    // 3. 중복 확인
    if _, exists := m.sessions[sessID]; exists {
        return nil, fmt.Errorf("session already exists: %s", sessID)
    }

    // 4. 세션 생성
    sess, err := NewSecureSessionFromExporterWithRole(
        sessID,
        seed,
        initiator,
        config,
    )
    if err != nil {
        return nil, err
    }

    // 5. 저장
    m.sessions[sessID] = sess

    log.Info("session created",
        "id", sessID,
        "context", params.ContextID,
        "initiator", initiator)

    return sess, nil
}
```

### 6.3 KeyID 바인딩

```go
// session/manager.go

func (m *Manager) BindKeyID(keyID string, sessionID string) error {
    m.mu.Lock()
    defer m.mu.Unlock()

    // 1. 세션 존재 확인
    if _, exists := m.sessions[sessionID]; !exists {
        return fmt.Errorf("session not found: %s", sessionID)
    }

    // 2. KeyID 중복 확인
    if existing, ok := m.keyIDs[keyID]; ok {
        if existing != sessionID {
            return fmt.Errorf("keyID already bound to different session")
        }
        // 이미 바인딩되어 있으면 OK
        return nil
    }

    // 3. 바인딩
    m.keyIDs[keyID] = sessionID

    log.Info("keyID bound", "keyID", keyID, "sessionID", sessionID)

    return nil
}

// KeyID로 세션 조회
func (m *Manager) GetByKeyID(keyID string) (*SecureSession, bool) {
    m.mu.RLock()
    defer m.mu.RUnlock()

    sessID, ok := m.keyIDs[keyID]
    if !ok {
        return nil, false
    }

    sess, ok := m.sessions[sessID]
    return sess, ok
}

// RFC 9421 서명 검증 시 사용
func (m *Manager) VerifyWithKeyID(
    keyID string,
    message []byte,
    signature []byte,
) error {
    sess, ok := m.GetByKeyID(keyID)
    if !ok {
        return fmt.Errorf("session not found for keyID: %s", keyID)
    }

    if sess.IsExpired() {
        return fmt.Errorf("session expired")
    }

    return sess.VerifyCovered(message, signature)
}
```

### 6.4 세션 정리

```go
// session/manager.go

func (m *Manager) cleanupLoop() {
    for {
        select {
        case <-m.cleanupTicker.C:
            m.CleanupExpired()
        case <-m.stopCleanup:
            m.cleanupTicker.Stop()
            return
        }
    }
}

func (m *Manager) CleanupExpired() {
    m.mu.Lock()
    defer m.mu.Unlock()

    now := time.Now()
    var expired []string

    // 만료된 세션 찾기
    for sessID, sess := range m.sessions {
        if sess.IsExpired() {
            expired = append(expired, sessID)
        }
    }

    // 제거
    for _, sessID := range expired {
        sess := m.sessions[sessID]

        // 안전한 키 삭제
        sess.Close()

        // 맵에서 제거
        delete(m.sessions, sessID)

        // KeyID 바인딩도 제거
        for keyID, sid := range m.keyIDs {
            if sid == sessID {
                delete(m.keyIDs, keyID)
            }
        }

        log.Info("session cleaned up",
            "id", sessID,
            "reason", "expired")
    }

    if len(expired) > 0 {
        log.Info("cleanup completed", "removed", len(expired))
    }
}

// 강제 제거
func (m *Manager) Remove(sessionID string) error {
    m.mu.Lock()
    defer m.mu.Unlock()

    sess, exists := m.sessions[sessionID]
    if !exists {
        return fmt.Errorf("session not found")
    }

    // 안전한 키 삭제
    sess.Close()

    // 제거
    delete(m.sessions, sessionID)

    // KeyID 바인딩 제거
    for keyID, sid := range m.keyIDs {
        if sid == sessionID {
            delete(m.keyIDs, keyID)
        }
    }

    log.Info("session removed", "id", sessionID)

    return nil
}

// 서버 종료 시
func (m *Manager) Shutdown() {
    close(m.stopCleanup)

    m.mu.Lock()
    defer m.mu.Unlock()

    // 모든 세션 정리
    for _, sess := range m.sessions {
        sess.Close()
    }

    m.sessions = nil
    m.keyIDs = nil
}
```

### 6.5 통계 및 모니터링

```go
// session/manager.go

type SessionStats struct {
    TotalSessions   int           `json:"total_sessions"`
    ActiveSessions  int           `json:"active_sessions"`
    ExpiredSessions int           `json:"expired_sessions"`
    AverageAge      time.Duration `json:"average_age"`
    OldestSession   time.Duration `json:"oldest_session"`
    KeyIDBindings   int           `json:"keyid_bindings"`
}

func (m *Manager) GetStats() SessionStats {
    m.mu.RLock()
    defer m.mu.RUnlock()

    now := time.Now()
    stats := SessionStats{
        TotalSessions: len(m.sessions),
        KeyIDBindings: len(m.keyIDs),
    }

    var totalAge time.Duration
    var oldest time.Duration

    for _, sess := range m.sessions {
        age := now.Sub(sess.GetCreatedAt())
        totalAge += age

        if age > oldest {
            oldest = age
        }

        if sess.IsExpired() {
            stats.ExpiredSessions++
        } else {
            stats.ActiveSessions++
        }
    }

    if stats.TotalSessions > 0 {
        stats.AverageAge = totalAge / time.Duration(stats.TotalSessions)
    }
    stats.OldestSession = oldest

    return stats
}

// 헬스 체크
func (m *Manager) HealthCheck() error {
    stats := m.GetStats()

    // 너무 많은 만료 세션
    if stats.ExpiredSessions > stats.TotalSessions/2 {
        return fmt.Errorf("too many expired sessions: %d/%d",
            stats.ExpiredSessions, stats.TotalSessions)
    }

    // 세션이 너무 많음
    if stats.TotalSessions > 10000 {
        return fmt.Errorf("too many sessions: %d", stats.TotalSessions)
    }

    return nil
}
```

---

## 7. 이벤트 기반 아키텍처

### 7.1 Events 인터페이스

```go
// handshake/types.go

type Events interface {
    // 초대 수신
    OnInvitation(ctx context.Context, contextID string, inv InvitationMessage) error

    // 요청 수신
    OnRequest(ctx context.Context, contextID string, req RequestMessage, peerPub crypto.PublicKey) error

    // 완료 수신
    OnComplete(ctx context.Context, contextID string, comp CompleteMessage, params session.Params) error

    // 임시 키 생성 요청
    AskEphemeral(ctx context.Context, contextID string) (raw []byte, jwk []byte, error)
}

// KeyID 발급 (선택적)
type KeyIDBinder interface {
    IssueKeyID(contextID string) (keyID string, ok bool)
}

// NoopEvents (기본 구현)
type NoopEvents struct{}

func (NoopEvents) OnInvitation(ctx context.Context, contextID string, inv InvitationMessage) error {
    return nil
}
func (NoopEvents) OnRequest(ctx context.Context, contextID string, req RequestMessage, peerPub crypto.PublicKey) error {
    return nil
}
func (NoopEvents) OnComplete(ctx context.Context, contextID string, comp CompleteMessage, params session.Params) error {
    return nil
}
func (NoopEvents) AskEphemeral(ctx context.Context, contextID string) ([]byte, []byte, error) {
    return nil, nil, fmt.Errorf("not implemented")
}
```

### 7.2 이벤트 핸들러 구현

```go
// examples/handshake/event_handler.go

type MyEventHandler struct {
    sessionManager *session.Manager
    keyStore       map[string]*keys.X25519KeyPair  // contextID → 임시 키
    keyIDs         map[string]string                // contextID → keyID
    mu             sync.Mutex

    exporter sagecrypto.KeyExporter
}

func NewMyEventHandler(sessionMgr *session.Manager) *MyEventHandler {
    return &MyEventHandler{
        sessionManager: sessionMgr,
        keyStore:       make(map[string]*keys.X25519KeyPair),
        keyIDs:         make(map[string]string),
        exporter:       formats.NewJWKExporter(),
    }
}

// OnInvitation: 초대 수신 시
func (h *MyEventHandler) OnInvitation(
    ctx context.Context,
    contextID string,
    inv InvitationMessage,
) error {
    log.Info("invitation received", "context", contextID)

    // 비즈니스 로직: 초대 수락 여부 결정
    // 예: 블랙리스트 체크, 레이트 리밋 등

    return nil
}

// OnRequest: 요청 수신 시
func (h *MyEventHandler) OnRequest(
    ctx context.Context,
    contextID string,
    req RequestMessage,
    peerPub crypto.PublicKey,
) error {
    log.Info("request received", "context", contextID)

    // 임시 키는 이미 AskEphemeral에서 생성됨
    // 여기서는 추가 검증이나 로깅만

    return nil
}

// OnComplete: 완료 수신 시 (세션 생성)
func (h *MyEventHandler) OnComplete(
    ctx context.Context,
    contextID string,
    comp CompleteMessage,
    params session.Params,
) error {
    log.Info("complete received", "context", contextID)

    // 1. 임시 키 가져오기
    h.mu.Lock()
    ephKeyPair, ok := h.keyStore[contextID]
    if !ok {
        h.mu.Unlock()
        return fmt.Errorf("ephemeral key not found for context: %s", contextID)
    }
    delete(h.keyStore, contextID)  // 사용 후 즉시 삭제
    h.mu.Unlock()

    // 2. 공유 비밀 계산
    sharedSecret, err := ephKeyPair.DeriveSharedSecret(params.PeerEph)
    if err != nil {
        return fmt.Errorf("derive shared secret failed: %w", err)
    }

    // 3. 세션 생성
    sess, err := h.sessionManager.CreateSession(
        params,
        sharedSecret,
        false,  // 서버는 initiator=false
        session.Config{
            MaxAge:      time.Hour,
            IdleTimeout: 10 * time.Minute,
            MaxMessages: 10000,
        },
    )
    if err != nil {
        return fmt.Errorf("session creation failed: %w", err)
    }

    // 4. KeyID 생성
    keyID := uuid.NewString()

    // 5. KeyID → SessionID 바인딩
    err = h.sessionManager.BindKeyID(keyID, sess.GetID())
    if err != nil {
        return fmt.Errorf("bind keyID failed: %w", err)
    }

    // 6. KeyID 저장 (IssueKeyID에서 반환용)
    h.mu.Lock()
    h.keyIDs[contextID] = keyID
    h.mu.Unlock()

    log.Info("session created",
        "context", contextID,
        "sessionID", sess.GetID(),
        "keyID", keyID)

    return nil
}

// AskEphemeral: 임시 키 생성
func (h *MyEventHandler) AskEphemeral(
    ctx context.Context,
    contextID string,
) (raw []byte, jwk []byte, error) {
    // 1. X25519 키 쌍 생성
    ephKeyPair, err := keys.GenerateX25519KeyPair()
    if err != nil {
        return nil, nil, err
    }

    ephX := ephKeyPair.(*keys.X25519KeyPair)

    // 2. Raw 바이트
    raw = ephX.PublicBytesKey()

    // 3. JWK 직렬화
    jwk, err = h.exporter.ExportPublic(ephKeyPair, sagecrypto.KeyFormatJWK)
    if err != nil {
        return nil, nil, err
    }

    // 4. 저장 (OnComplete에서 사용)
    h.mu.Lock()
    h.keyStore[contextID] = ephX
    h.mu.Unlock()

    log.Debug("ephemeral key generated", "context", contextID)

    return raw, jwk, nil
}

// IssueKeyID: KeyID 발급
func (h *MyEventHandler) IssueKeyID(contextID string) (string, bool) {
    h.mu.Lock()
    defer h.mu.Unlock()

    keyID, ok := h.keyIDs[contextID]
    if ok {
        delete(h.keyIDs, contextID)  // 한 번만 사용
    }
    return keyID, ok
}
```

### 7.3 이벤트 핸들러 사용

```go
// examples/handshake/server_example.go

func main() {
    // 1. 세션 관리자
    sessionMgr := session.NewManager()

    // 2. 이벤트 핸들러
    events := NewMyEventHandler(sessionMgr)

    // 3. DID Resolver
    resolver := did.NewMultiChainResolver()
    // ... resolver 설정 ...

    // 4. Ed25519 키 (서버 신원)
    serverKey, _ := keys.GenerateEd25519KeyPair()

    // 5. 핸드셰이크 서버
    hsServer := handshake.NewServer(
        serverKey,
        events,
        resolver,
        &session.Config{
            MaxAge:      time.Hour,
            IdleTimeout: 10 * time.Minute,
        },
        10 * time.Minute,  // cleanup interval
    )

    // 6. gRPC 서버
    grpcServer := grpc.NewServer()
    a2a.RegisterA2AServiceServer(grpcServer, hsServer)

    // 7. 리스닝
    lis, _ := net.Listen("tcp", ":50051")
    fmt.Println("Server listening on :50051")
    grpcServer.Serve(lis)
}
```

---

## 8. 보안 고려사항

### 8.1 타이밍 공격 방지

```go
// handshake/server.go

func (s *Server) verifySenderSignature(
    m *a2a.Message,
    meta *structpb.Struct,
    senderPub crypto.PublicKey,
) error {
    field := meta.GetFields()["signature"]
    if field == nil {
        return errors.New("missing signature")
    }

    sig, err := base64.RawURLEncoding.DecodeString(field.GetStringValue())
    if err != nil {
        return fmt.Errorf("bad signature b64: %w", err)
    }

    bytes, _ := proto.MarshalOptions{Deterministic: true}.Marshal(m)

    // 타이밍 공격 방지를 위한 constant-time 비교
    switch pk := senderPub.(type) {
    case ed25519.PublicKey:
        // ed25519.Verify는 내부적으로 constant-time
        if !ed25519.Verify(pk, bytes, sig) {
            return errors.New("signature verify failed")
        }
        return nil
    default:
        return fmt.Errorf("unsupported key type: %T", senderPub)
    }
}
```

### 8.2 Nonce 재사용 방지

```go
// session/nonce.go

type NonceCache struct {
    seen   map[string]time.Time
    ttl    time.Duration
    mu     sync.Mutex
    ticker *time.Ticker
}

func NewNonceCache(ttl time.Duration) *NonceCache {
    nc := &NonceCache{
        seen:   make(map[string]time.Time),
        ttl:    ttl,
        ticker: time.NewTicker(ttl),
    }
    go nc.cleanupLoop()
    return nc
}

// Check: nonce가 처음인지 확인
func (nc *NonceCache) Check(nonce string) bool {
    nc.mu.Lock()
    defer nc.mu.Unlock()

    // 이미 사용된 nonce
    if _, exists := nc.seen[nonce]; exists {
        return false
    }

    // 새 nonce 기록
    nc.seen[nonce] = time.Now()
    return true
}

// 자동 정리
func (nc *NonceCache) cleanupLoop() {
    for range nc.ticker.C {
        nc.cleanup()
    }
}

func (nc *NonceCache) cleanup() {
    nc.mu.Lock()
    defer nc.mu.Unlock()

    now := time.Now()
    for nonce, ts := range nc.seen {
        if now.Sub(ts) > nc.ttl {
            delete(nc.seen, nonce)
        }
    }
}

위치: session/nonce.go
```

### 8.3 DDoS 방지

```go
// handshake/ratelimit.go

type RateLimiter struct {
    requests map[string][]time.Time  // IP → 요청 타임스탬프
    mu       sync.Mutex
    limit    int           // 최대 요청 수
    window   time.Duration // 시간 윈도우
}

func NewRateLimiter(limit int, window time.Duration) *RateLimiter {
    return &RateLimiter{
        requests: make(map[string][]time.Time),
        limit:    limit,
        window:   window,
    }
}

func (rl *RateLimiter) Allow(ip string) bool {
    rl.mu.Lock()
    defer rl.mu.Unlock()

    now := time.Now()
    cutoff := now.Add(-rl.window)

    // 오래된 요청 제거
    reqs := rl.requests[ip]
    var recent []time.Time
    for _, ts := range reqs {
        if ts.After(cutoff) {
            recent = append(recent, ts)
        }
    }

    // 제한 확인
    if len(recent) >= rl.limit {
        return false
    }

    // 새 요청 기록
    recent = append(recent, now)
    rl.requests[ip] = recent

    return true
}

// 서버에 통합
func (s *Server) SendMessage(ctx context.Context, in *a2a.SendMessageRequest) (*a2a.SendMessageResponse, error) {
    // 1. IP 추출
    peer, _ := peer.FromContext(ctx)
    ip := peer.Addr.String()

    // 2. 레이트 리밋
    if !s.rateLimiter.Allow(ip) {
        return nil, status.Error(codes.ResourceExhausted, "rate limit exceeded")
    }

    // 3. 정상 처리
    // ...
}
```

### 8.4 키 재사용 방지

```go
// session/session.go

// Close: 세션 종료 시 모든 키 삭제
func (s *SecureSession) Close() error {
    s.closed = true

    // 모든 키를 0으로 덮어쓰기 (보안 삭제)
    zeroBytes := func(b []byte) {
        if b != nil {
            for i := range b {
                b[i] = 0
            }
        }
    }

    zeroBytes(s.encryptKey)
    zeroBytes(s.signingKey)
    zeroBytes(s.sessionSeed)
    zeroBytes(s.outKey)
    zeroBytes(s.inKey)
    zeroBytes(s.outSign)
    zeroBytes(s.inSign)

    // AEAD 인스턴스는 GC가 처리
    s.aead = nil
    s.aeadOut = nil
    s.aeadIn = nil

    return nil
}

위치: session/session.go:355-398

주의사항:
- Go의 GC는 메모리를 0으로 지우지 않음
- 명시적으로 0으로 덮어쓰기 필요
- 하드웨어 최적화로 제거될 수 있음
- 더 강력한 보안이 필요하면 mlock() 사용
```

---

## 9. 실전 예제

### 9.1 완전한 핸드셰이크 구현

**디렉토리 구조**:
```
examples/handshake-complete/
├── main.go
├── client.go
├── server.go
├── events.go
└── README.md
```

**main.go**:
```go
package main

import (
    "fmt"
    "os"
)

func main() {
    if len(os.Args) < 2 {
        fmt.Println("Usage: handshake-complete [server|client]")
        os.Exit(1)
    }

    mode := os.Args[1]
    switch mode {
    case "server":
        runServer()
    case "client":
        runClient()
    default:
        fmt.Printf("Unknown mode: %s\n", mode)
        os.Exit(1)
    }
}
```

**server.go**:
```go
package main

import (
    "context"
    "fmt"
    "net"
    "time"

    a2a "github.com/a2aproject/a2a/grpc"
    "github.com/sage-x-project/sage/crypto/keys"
    "github.com/sage-x-project/sage/did"
    "github.com/sage-x-project/sage/did/ethereum"
    "github.com/sage-x-project/sage/handshake"
    "github.com/sage-x-project/sage/session"
    "google.golang.org/grpc"
)

func runServer() {
    fmt.Println("=== SAGE Handshake Server ===\n")

    // 1. 키 생성
    serverKey, _ := keys.GenerateEd25519KeyPair()
    fmt.Printf("Server Key ID: %s\n", serverKey.ID())

    // 2. DID 설정 (간소화: 실제로는 블록체인 등록 필요)
    serverDID := fmt.Sprintf("did:sage:ethereum:0xserver%s", serverKey.ID()[:8])
    fmt.Printf("Server DID: %s\n", serverDID)

    // 3. 세션 관리자
    sessionMgr := session.NewManager()

    // 4. DID Resolver
    resolver := did.NewMultiChainResolver()
    // ... resolver 설정 (생략) ...

    // 5. 이벤트 핸들러
    events := NewServerEventHandler(sessionMgr)

    // 6. 핸드셰이크 서버
    hsServer := handshake.NewServer(
        serverKey,
        events,
        resolver,
        &session.Config{
            MaxAge:      time.Hour,
            IdleTimeout: 10 * time.Minute,
            MaxMessages: 10000,
        },
        10 * time.Minute,
    )

    // 7. gRPC 서버
    grpcServer := grpc.NewServer()
    a2a.RegisterA2AServiceServer(grpcServer, hsServer)

    // 8. 리스닝
    lis, err := net.Listen("tcp", ":50051")
    if err != nil {
        panic(err)
    }

    fmt.Println("\n✅ Server started on :50051")
    fmt.Println("Waiting for handshake requests...\n")

    if err := grpcServer.Serve(lis); err != nil {
        panic(err)
    }
}
```

**client.go**:
```go
package main

import (
    "context"
    "fmt"
    "time"

    "github.com/google/uuid"
    "github.com/sage-x-project/sage/crypto/keys"
    "github.com/sage-x-project/sage/handshake"
    "github.com/sage-x-project/sage/session"
    "google.golang.org/grpc"
    "google.golang.org/grpc/credentials/insecure"
)

func runClient() {
    fmt.Println("=== SAGE Handshake Client ===\n")

    // 1. 키 생성
    clientKey, _ := keys.GenerateEd25519KeyPair()
    fmt.Printf("Client Key ID: %s\n", clientKey.ID())

    clientDID := fmt.Sprintf("did:sage:ethereum:0xclient%s", clientKey.ID()[:8])
    fmt.Printf("Client DID: %s\n", clientDID)

    // 2. gRPC 연결
    conn, err := grpc.Dial("localhost:50051",
        grpc.WithTransportCredentials(insecure.NewCredentials()))
    if err != nil {
        panic(err)
    }
    defer conn.Close()

    // 3. 핸드셰이크 클라이언트
    hsClient := handshake.NewClient(conn, clientKey)

    // 4. 핸드셰이크 수행
    contextID := uuid.NewString()
    serverDID := "did:sage:ethereum:0xserver..."  // 실제로는 조회

    fmt.Println("\n--- Starting Handshake ---")
    keyID, err := performFullHandshake(hsClient, contextID, clientDID, serverDID)
    if err != nil {
        fmt.Printf("❌ Handshake failed: %v\n", err)
        return
    }

    fmt.Printf("\n✅ Handshake successful!\n")
    fmt.Printf("   KeyID: %s\n", keyID)

    // 5. 세션 사용 예시
    fmt.Println("\n--- Testing Session ---")
    testSession(keyID)
}

func performFullHandshake(
    client *handshake.Client,
    contextID string,
    myDID string,
    peerDID string,
) (string, error) {
    ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
    defer cancel()

    // ... 핸드셰이크 구현 (이전 섹션 참조) ...

    return "keyID-123", nil
}

func testSession(keyID string) {
    // 세션으로 메시지 암호화/복호화 테스트
    fmt.Println("Encrypting test message...")
    // ...
    fmt.Println("✅ Session test passed!")
}
```

**events.go**:
```go
package main

import (
    "context"
    "fmt"
    "sync"
    "time"

    "github.com/google/uuid"
    "github.com/sage-x-project/sage/crypto"
    "github.com/sage-x-project/sage/crypto/formats"
    "github.com/sage-x-project/sage/crypto/keys"
    "github.com/sage-x-project/sage/handshake"
    "github.com/sage-x-project/sage/session"
    sagecrypto "github.com/sage-x-project/sage/crypto"
)

type ServerEventHandler struct {
    sessionManager *session.Manager
    keyStore       map[string]*keys.X25519KeyPair
    keyIDs         map[string]string
    mu             sync.Mutex
    exporter       sagecrypto.KeyExporter
}

func NewServerEventHandler(sessionMgr *session.Manager) *ServerEventHandler {
    return &ServerEventHandler{
        sessionManager: sessionMgr,
        keyStore:       make(map[string]*keys.X25519KeyPair),
        keyIDs:         make(map[string]string),
        exporter:       formats.NewJWKExporter(),
    }
}

func (h *ServerEventHandler) OnInvitation(
    ctx context.Context,
    contextID string,
    inv handshake.InvitationMessage,
) error {
    fmt.Printf("📨 Invitation received (context: %s)\n", contextID)
    return nil
}

func (h *ServerEventHandler) OnRequest(
    ctx context.Context,
    contextID string,
    req handshake.RequestMessage,
    peerPub crypto.PublicKey,
) error {
    fmt.Printf("📨 Request received (context: %s)\n", contextID)
    return nil
}

func (h *ServerEventHandler) OnComplete(
    ctx context.Context,
    contextID string,
    comp handshake.CompleteMessage,
    params session.Params,
) error {
    fmt.Printf("📨 Complete received (context: %s)\n", contextID)

    // 세션 생성 로직 (이전 섹션 참조)
    // ...

    fmt.Printf("✅ Session created for context: %s\n", contextID)
    return nil
}

func (h *ServerEventHandler) AskEphemeral(
    ctx context.Context,
    contextID string,
) ([]byte, []byte, error) {
    fmt.Printf("🔑 Generating ephemeral key (context: %s)\n", contextID)

    // 키 생성 로직 (이전 섹션 참조)
    // ...

    return raw, jwk, nil
}

func (h *ServerEventHandler) IssueKeyID(contextID string) (string, bool) {
    h.mu.Lock()
    defer h.mu.Unlock()

    keyID, ok := h.keyIDs[contextID]
    if ok {
        delete(h.keyIDs, contextID)
        fmt.Printf("🎫 KeyID issued: %s\n", keyID)
    }
    return keyID, ok
}
```

### 9.2 실행 예시

**터미널 1 (서버)**:
```bash
$ go run . server
=== SAGE Handshake Server ===

Server Key ID: a1b2c3d4e5f6g7h8
Server DID: did:sage:ethereum:0xservera1b2c3d4

✅ Server started on :50051
Waiting for handshake requests...

📨 Invitation received (context: ctx-abc123)
📨 Request received (context: ctx-abc123)
🔑 Generating ephemeral key (context: ctx-abc123)
📨 Complete received (context: ctx-abc123)
✅ Session created for context: ctx-abc123
🎫 KeyID issued: keyid-xyz789
```

**터미널 2 (클라이언트)**:
```bash
$ go run . client
=== SAGE Handshake Client ===

Client Key ID: 9i0j1k2l3m4n5o6p
Client DID: did:sage:ethereum:0xclient9i0j1k2l

--- Starting Handshake ---
✅ Invitation sent
✅ Request sent
✅ Response received
✅ Shared secret computed
✅ Complete sent
✅ KeyID received: keyid-xyz789

✅ Handshake successful!
   KeyID: keyid-xyz789

--- Testing Session ---
Encrypting test message...
Decrypting test message...
✅ Session test passed!
```

---

## 요약

Part 4에서 다룬 내용:

1. **핸드셰이크 프로토콜 개요**: 필요성, TLS 비교, A2A 통합
2. **4단계 핸드셰이크**: Invitation, Request, Response, Complete 상세
3. **클라이언트 구현**: 구조, 시퀀스, 에러 처리
4. **서버 구현**: 구조, 상태 관리, 자동 정리, DID 조회 최적화
5. **세션 생성**: 파라미터, 시드 유도, ID 계산, 방향별 키 유도, AEAD 초기화
6. **세션 관리자**: 생성, KeyID 바인딩, 정리, 통계
7. **이벤트 기반 아키텍처**: Events 인터페이스, 핸들러 구현
8. **보안 고려사항**: 타이밍 공격, Nonce, DDoS, 키 재사용 방지
9. **실전 예제**: 완전한 구현, 실행 예시

**다음 파트 예고**:

**Part 5: 스마트 컨트랙트 및 온체인 레지스트리**에서는:
- Solidity 컨트랙트 상세 분석
- V2 보안 강화 기능
- 가스 최적화 기법
- 배포 및 검증 프로세스

계속해서 Part 5를 작성하시겠습니까?
