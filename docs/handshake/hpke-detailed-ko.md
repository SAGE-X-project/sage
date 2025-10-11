# HPKE 핸드셰이크 메커니즘 상세 설명 (코드 기반)

> **작성일**: 2025-10-09
> **최종 업데이트**: 2025-10-09
> **목적**: SAGE의 HPKE 기반 핸드셰이크를 실제 코드를 통해 이해하기 쉽게 설명

> **중요**: 이 문서는 **HPKE 기반 핸드셰이크** (`/hpke` 패키지)를 설명합니다.
> **전통적 4단계 핸드셰이크** (`/handshake` 패키지)와는 다른 프로토콜입니다.
> 전통적 방식은 [handshake-ko.md](./handshake-ko.md)를 참조하세요.

> **문서 상태**: 이 문서는 현재 업데이트 중입니다. 일부 코드 예제가 `/handshake` 패키지와 `/hpke` 패키지를 혼용하고 있습니다.
> 정확한 HPKE 구현은 [hpke-based-handshake-ko.md](./hpke-based-handshake-ko.md)를 참조하세요.

## 목차
1. [기술 용어 해설](#기술-용어-해설)
2. [HPKE 핸드셰이크 과정 (2단계)](#hpke-핸드셰이크-과정-2단계)
3. [Forward Secrecy 구현](#forward-secrecy-구현)
4. [세션 암호화](#세션-암호화)

---

## 기술 용어 해설

HPKE 핸드셰이크를 이해하기 위해 필요한 핵심 용어들을 정리합니다.

### 암호화 프로토콜

| 용어 | 전체 이름 | 설명 |
|------|----------|------|
| **HPKE** | Hybrid Public Key Encryption | 공개키와 대칭키 암호화를 혼합한 방식. 공개키로 키 합의 → 대칭키로 빠른 암호화 |
| **DID** | Decentralized Identifier | 블록체인에 등록된 에이전트의 고유 식별자 (예: `did:sage:agent123`) |
| **HKDF** | HMAC-based Key Derivation Function | 하나의 비밀값(exporter)에서 여러 개의 키를 안전하게 생성하는 함수 |
| **HMAC** | Hash-based Message Authentication Code | 메시지가 변조되지 않았음을 증명하는 코드 |

### 핵심 데이터 값

| 용어 | 크기 | 설명 | 전송 여부 |
|------|------|------|----------|
| **enc** | 32 bytes | HPKE에서 생성되는 임시 공개키 (encapsulated key) | Yes (전송) |
| **exporter** | 32 bytes | 양쪽 에이전트가 동일하게 계산하는 공유 비밀값 | No (절대 전송 안함) |
| **ackTag** | 32 bytes | HMAC 기반 키 확인 태그 (상대방이 같은 키를 가졌는지 증명) | Yes (전송) |
| **kid** | variable | Key ID - 세션을 식별하는 ID (예: `"session:abc123"`) | Yes (전송) |
| **nonce** | variable | 재전송 공격 방지를 위한 일회용 난수 | Yes (전송) |

### 암호화 알고리즘

| 알고리즘 | 용도 | 특징 |
|---------|------|------|
| **X25519** | 타원곡선 키 교환 (ECDH) | Diffie-Hellman 키 합의에 사용, 32바이트 키 생성 |
| **Ed25519** | 전자 서명 | 메시지 서명 및 검증, 공개키 인증에 사용 |
| **ChaCha20-Poly1305** | AEAD 대칭키 암호화 | 실제 메시지 암호화, AES보다 빠름 |
| **SHA-256** | 해시 함수 | HMAC, HKDF에서 사용 |

---

## HPKE 핸드셰이크 과정 (2단계)

HPKE 기반 핸드셰이크는 **2단계 (1-RTT)** 프로토콜입니다:

1. **Initialize** (초기화): 클라이언트 → 서버 (HPKE `enc` + 임시 키 `ephC` 전송)
2. **Acknowledge** (확인): 서버 → 클라이언트 (세션 ID `kid` + 확인 태그 `ackTag` + 임시 키 `ephS` 응답)

```
클라이언트 (Initiator)                서버 (Responder)
     │                                        │
     │  1. Initialize                         │
     │  - HPKE enc (32바이트)                 │
     │  - ephC 공개키 (32바이트)              │
     │  - info, exportCtx, nonce              │
     │ ────────────────────────────────────>  │
     │                                        │
     │                                        ├─ enc로 HPKE 복호화
     │                                        ├─ ephC로 E2E DH 수행
     │                                        └─ 세션 생성
     │                                        │
     │  2. Acknowledge                        │
     │  <- kid (세션 ID)                      │
     │  <- ackTag (HMAC 확인)                 │
     │  <- ephS 공개키 (32바이트)             │
     │ <────────────────────────────────────  │
     │                                        │
     ├─ ackTag 검증                           │
     ├─ ephS로 E2E DH 완료                    │
     └─ 세션 시작                             │
     │                                        │
     │ [암호화된 세션 수립 완료]              │
```

각 단계를 실제 코드와 함께 살펴봅니다.

---

### Phase 1: Initialize (초기화 - 클라이언트 → 서버)

**목적**: HPKE 프로토콜로 공유 비밀 생성 및 E2E 임시 키 교환

**코드 위치**: `hpke/client.go:70-140`

```go
func (c *Client) Invitation(ctx context.Context, invMsg InvitationMessage, did string) (*a2a.SendMessageResponse, error) {
    // 1. JSON 메시지를 protobuf Struct로 변환
    payload, err := toStructPB(invMsg)
    if err != nil {
        return nil, fmt.Errorf("marshal invitation: %w", err)
    }

    // 2. A2A 메시지 구성
    msg := &a2a.Message{
        TaskId:    "handshake/invitation@v1",  // Phase 식별자
        ContextId: invMsg.ContextID,           // 이 핸드셰이크의 고유 ID
        Content: []*a2a.Part{{
            Part: &a2a.Part_Data{
                Data: &a2a.DataPart{Data: payload}
            }
        }},
    }

    // 3. 메시지를 deterministic하게 직렬화
    bytes, err := proto.MarshalOptions{Deterministic: true}.Marshal(msg)
    if err != nil {
        return nil, err
    }

    // 4. Ed25519로 서명 (위변조 방지)
    metadataStruct, err := signStruct(c.key, bytes, did)
    if err != nil {
        return nil, fmt.Errorf("sign: %w", err)
    }

    // 5. gRPC로 전송
    return c.A2AServiceClient.SendMessage(ctx, &a2a.SendMessageRequest{
        Message:  msg,
        Metadata: metadataStruct,
    })
}
```

**데이터 흐름**:
```
Agent A                                    Agent B
   │                                          │
   │  InvitationMessage (평문)                │
   │  + Ed25519 서명                          │
   │ ────────────────────────────────────>   │
   │                                          │
   │                                          ├─ 서명 검증
   │                                          └─ OnInvitation 이벤트 발생
```

**특징**:
- 평문 전송이지만 Ed25519 서명으로 보호
- 아직 암호화 없음 (공개키 교환 전)
- Agent B는 서명을 검증하여 Agent A의 신원 확인

---

### Phase 2: Request (요청 - 핵심 키 교환)

**목적**: HPKE 프로토콜을 사용하여 공유 비밀(exporter) 생성

**코드 위치**: `hpke/client.go:90-125`

```go
func (c *Client) Initialize(ctx context.Context, ctxID, initDID, peerDID string) (kid string, err error) {
    // 1. 블록체인에서 Agent B의 공개키 조회
    peerPub, err := c.resolver.ResolvePublicKey(ctx, did.AgentDID(peerDID))
    if err != nil {
        return "", fmt.Errorf("resolve peer public key: %w", err)
    }

    // 2. HPKE 프로토콜 정보 구성
    // info: 프로토콜 바인딩 정보 (양쪽이 동일한 값 사용)
    info := c.info.BuildInfo(ctxID, initDID, peerDID)
    // 실제 값: "sage/hpke v1|ctx=abc123|init=did:sage:A|resp=did:sage:B"

    // exportCtx: 키 확장용 컨텍스트
    exportCtx := c.info.BuildExportContext(ctxID)
    // 실제 값: "exporter:abc123"

    // 3. HPKE 키 합의 수행 (가장 중요한 부분!)
    enc, exporter, err := keys.HPKEDeriveSharedSecretToPeer(peerPub, info, exportCtx, 32)
    if err != nil {
        return "", fmt.Errorf("HPKE derive: %w", err)
    }
    // enc: 32바이트 임시 공개키 → Agent B에게 전송
    // exporter: 32바이트 공유 비밀 → 절대 전송하지 않음!

    // 4. 공유 비밀(exporter)로 세션 생성
    _, sid, _, err := c.sessMgr.EnsureSessionFromExporterWithRole(
        exporter,
        "sage/hpke v1", // 세션 ID 생성에 사용되는 레이블
        true,           // isInitiator = true (Agent A가 시작)
        nil,
    )
    if err != nil {
        return "", fmt.Errorf("create session: %w", err)
    }

    // 5. 재전송 공격 방지를 위한 nonce 생성
    nonce := uuid.NewString()

    // 6. Agent B에게 전송할 페이로드 구성
    payload := map[string]any{
        "initDid":   initDID,                                          // Agent A의 DID
        "respDid":   peerDID,                                          // Agent B의 DID
        "info":      string(info),                                     // 프로토콜 바인딩 정보
        "exportCtx": string(exportCtx),                                // 키 확장 컨텍스트
        "enc":       base64.RawURLEncoding.EncodeToString(enc),        // 임시 공개키 (전송)
        "nonce":     nonce,                                            // 재전송 공격 방지
        "ts":        time.Now().Format(time.RFC3339Nano),             // 타임스탬프
    }

    // 7. gRPC로 전송 (코드 생략)
    // ...
}
```

**HPKE 키 합의 내부 동작** (`keys.HPKEDeriveSharedSecretToPeer`):
```go
// 내부적으로 다음과 같은 작업 수행:
// 1. 임시 X25519 키쌍 생성
ephemeralPriv, ephemeralPub := x25519.GenerateKey(rand.Reader)

// 2. Agent B의 공개키와 ECDH 수행
sharedPoint := x25519.ECDH(ephemeralPriv, peerPub)

// 3. HKDF로 공유 비밀 추출
exporter := HKDF-Extract(sharedPoint, info)

// 반환값:
// enc = ephemeralPub  (32 bytes) - Agent B에게 전송
// exporter            (32 bytes) - 절대 전송 안함
```

**데이터 흐름**:
```
Agent A                                                Agent B
   │                                                      │
   ├─ X25519 임시 키쌍 생성                                │
   │  ephPriv (비밀), ephPub (공개)                        │
   │                                                      │
   ├─ Agent B의 공개키로 ECDH                              │
   │  shared = ECDH(ephPriv, B_pub)                      │
   │                                                      │
   ├─ HKDF로 exporter 추출                                │
   │  exporter = HKDF(shared, info)                      │
   │                                                      │
   ├─ 세션 생성 (exporter 사용)                            │
   │  sessionID = sid                                     │
   │                                                      │
   │  {enc, info, exportCtx, nonce, ts}                  │
   │ ─────────────────────────────────────────────────>  │
   │                                                      │
```

**핵심 포인트**:
- **enc만 전송**, exporter는 절대 전송 안함
- Agent B는 받은 enc와 자신의 개인키로 동일한 exporter 계산 가능
- info, exportCtx는 양쪽이 동일하게 사용 (프로토콜 바인딩)

---

### Phase 3: Response (응답 - 키 확인)

**목적**: Agent B가 동일한 exporter를 계산하고, ackTag로 증명

**코드 위치**: `hpke/server.go:104-143`

```go
func (s *Server) OnHandleTask(ctx context.Context, in *a2a.TaskRequest) (*a2a.TaskResponse, error) {
    // 1. Agent A가 보낸 페이로드 파싱
    st, err := firstDataPart(in.Message)
    if err != nil {
        return nil, err
    }

    pl, err := ParseHPKEInitPayload(st)
    if err != nil {
        return nil, fmt.Errorf("parse payload: %w", err)
    }
    // pl.Enc: Agent A의 임시 공개키 (32 bytes)
    // pl.Info: "sage/hpke v1|ctx=...|init=...|resp=..."
    // pl.ExportCtx: "exporter:..."
    // pl.Nonce: UUID 문자열

    // 2. Agent A의 서명 검증 (DID로 공개키 조회)
    senderPub, err := s.resolver.ResolvePublicKey(ctx, did.AgentDID(pl.InitDID))
    if err != nil {
        return nil, fmt.Errorf("resolve sender: %w", err)
    }

    if err := verifySenderSignature(in.Message, in.Metadata, senderPub); err != nil {
        return nil, fmt.Errorf("signature verification failed: %w", err)
    }

    // 3. 재전송 공격 방지 (nonce 중복 체크)
    if !s.nonces.checkAndMark(in.Message.ContextId + "|" + pl.Nonce) {
        return nil, errors.New("nonce reused - replay attack detected")
    }

    // 4. 타임스탬프 검증 (5분 이내 메시지만 허용)
    if time.Since(pl.Timestamp) > 5*time.Minute {
        return nil, errors.New("message too old")
    }

    // 5. 동일한 exporter 계산 (HPKE 키 합의)
    exporter, err := keys.HPKEDeriveSharedSecretFromPeer(
        s.key,          // Agent B의 개인키
        pl.Enc,         // Agent A가 보낸 임시 공개키
        pl.Info,        // 동일한 info
        pl.ExportCtx,   // 동일한 exportCtx
        32,             // 32 바이트 출력
    )
    if err != nil {
        return nil, fmt.Errorf("derive shared secret: %w", err)
    }
    // Agent A와 동일한 32바이트 exporter 획득!

    // 6. 세션 생성
    _, sid, _, err := s.sessMgr.EnsureSessionFromExporterWithRole(
        exporter,
        "sage/hpke v1",
        false,  // isInitiator = false (Agent B는 응답자)
        nil,
    )
    if err != nil {
        return nil, fmt.Errorf("create session: %w", err)
    }

    // 7. Key ID 생성 및 바인딩
    kid := "session:" + randBase64URL(12)  // 예: "session:xY3kL9mP2qR8"
    s.sessMgr.BindKeyID(kid, sid)

    // 8. ackTag 생성 (키 확인 증명)
    ackTag := makeAckTag(exporter, in.Message.ContextId, pl.Nonce, kid)
    // ackTag = HMAC(HKDF(exporter, "ack-key"), "hpke-ack|ctxID|nonce|kid")

    // 9. Agent A에게 응답
    return &a2a.TaskResponse{
        Metadata: map[string]string{
            "kid":       kid,
            "ackTagB64": base64.RawURLEncoding.EncodeToString(ackTag),
        },
    }, nil
}
```

**ackTag 생성 로직** (`hpke/common.go:180-190`):
```go
func makeAckTag(exporter []byte, ctxID, nonce, kid string) []byte {
    // 1. HKDF로 ack 전용 키 생성
    ackKey := hkdfExpand(exporter, "ack-key", 32)
    // ackKey = HKDF-Expand(exporter, "ack-key", 32 bytes)

    // 2. HMAC으로 태그 생성
    mac := hmac.New(sha256.New, ackKey)
    mac.Write([]byte("hpke-ack|"))
    mac.Write([]byte(ctxID))
    mac.Write([]byte("|"))
    mac.Write([]byte(nonce))
    mac.Write([]byte("|"))
    mac.Write([]byte(kid))

    return mac.Sum(nil)  // 32 bytes HMAC-SHA256
}
```

**데이터 흐름**:
```
Agent A                                                Agent B
   │                                                      │
   │  {enc, info, exportCtx, nonce}                      │
   │ ─────────────────────────────────────────────────>  │
   │                                                      │
   │                                                      ├─ enc + 자신의 개인키로 ECDH
   │                                                      │  shared = ECDH(B_priv, enc)
   │                                                      │
   │                                                      ├─ 동일한 exporter 계산
   │                                                      │  exporter = HKDF(shared, info)
   │                                                      │
   │                                                      ├─ 세션 생성 (sid)
   │                                                      │
   │                                                      ├─ kid 발급 및 바인딩
   │                                                      │
   │                                                      ├─ ackTag 생성
   │                                                      │  ackTag = HMAC(HKDF(exporter))
   │                                                      │
   │  {kid, ackTag}                                      │
   │ <─────────────────────────────────────────────────  │
```

**핵심 포인트**:
- Agent B는 **enc를 받아서 동일한 exporter 계산** (HPKE의 핵심)
- **ackTag**: Agent B가 올바른 exporter를 가졌음을 증명
- **nonce**: 재전송 공격 방지 (한 번만 사용)
- **timestamp**: 오래된 메시지 거부 (5분 제한)

---

### Phase 4: Complete (완료 - ackTag 검증)

**목적**: Agent A가 ackTag를 검증하여 Agent B도 같은 exporter를 가졌는지 확인

**코드 위치**: `hpke/client.go:148-165`

```go
func (c *Client) Initialize(ctx context.Context, ctxID, initDID, peerDID string) (kid string, err error) {
    // ... (Phase 2에서 계속) ...

    // Agent B로부터 응답 수신
    resp, err := c.a2a.SendMessage(ctx, signedMsg)
    if err != nil {
        return "", fmt.Errorf("send message: %w", err)
    }

    task := resp.GetTask()
    if task == nil {
        return "", errors.New("no task in response")
    }

    // 1. kid 및 ackTag 추출
    kid = task.Metadata["kid"]
    ackTagB64 := task.Metadata["ackTagB64"]
    if kid == "" || ackTagB64 == "" {
        return "", errors.New("missing kid or ackTag in response")
    }

    receivedAckTag, err := base64.RawURLEncoding.DecodeString(ackTagB64)
    if err != nil {
        return "", fmt.Errorf("decode ackTag: %w", err)
    }

    // 2. 동일한 방식으로 ackTag 계산
    expectedAckTag := makeAckTag(exporter, ctxID, nonce, kid)
    // 내부: HMAC(HKDF(exporter, "ack-key"), "hpke-ack|ctxID|nonce|kid")

    // 3. 시간 일정 비교 (타이밍 공격 방지)
    if !hmac.Equal(expectedAckTag, receivedAckTag) {
        return "", fmt.Errorf("ack tag mismatch - Agent B has different key")
    }
    // 검증 성공! Agent B도 동일한 exporter를 가짐

    // 4. kid를 세션에 바인딩
    c.sessMgr.BindKeyID(kid, sid)
    // 이제 kid로 메시지를 암호화/복호화 가능

    return kid, nil  // 성공!
}
```

**검증 흐름**:
```
Agent A                                                Agent B
   │                                                      │
   │  receivedAckTag ←─── {kid, ackTag}                 │
   │                                                      │
   ├─ expectedAckTag 계산                                 │
   │  HMAC(HKDF(exporter, "ack-key"), "...|kid")        │
   │                                                      │
   ├─ hmac.Equal(expected, received)                     │
   │  일치: Agent B도 같은 exporter 보유 확인              │
   │  불일치: 키 합의 실패                                 │
   │                                                      │
   ├─ BindKeyID(kid, sessionID)                          │
   │                                                      │
   │  [핸드셰이크 완료! 암호화 통신 시작]                   │
```

**핵심 포인트**:
- **ackTag 검증**: 암호문 없이도 키 일치 확인 (HMAC 사용)
- **hmac.Equal**: 타이밍 공격 방지 (상수 시간 비교)
- **kid 바인딩**: 이후 메시지에서 "Authorization: Bearer {kid}" 형태로 사용

---

## Forward Secrecy 구현

Forward Secrecy(전방향 비밀성)란 **현재 세션의 개인키가 노출되어도 과거 통신을 복호화할 수 없는** 특성입니다.

### 구현 방법

**코드 위치**: `internal/session_creator.go:93-100`

```go
func (a *Creator) OnComplete(ctx context.Context, ctxID string, comp CompleteMessage, p session.Params) error {
    // 1. 임시 개인키로 공유 비밀 계산
    a.mu.RLock()
    my := a.ephPrivByCtx[ctxID]  // X25519 임시 개인키
    a.mu.RUnlock()

    if my == nil {
        return fmt.Errorf("no ephemeral private for ctx=%s", ctxID)
    }

    shared, err := my.DeriveSharedSecret(p.PeerEph)
    if err != nil {
        return fmt.Errorf("derive shared: %w", err)
    }

    p.SharedSecret = shared

    // 2. 세션 생성
    _, sid, _, err := a.sessionMgr.EnsureSessionWithParams(p, nil)
    if err != nil {
        return fmt.Errorf("ensure session: %w", err)
    }

    // 3. 임시 개인키 즉시 삭제 (메모리에서 완전 제거)
    a.mu.Lock()
    delete(a.ephPrivByCtx, ctxID)  // 영구 삭제
    a.mu.Unlock()

    return nil
}
```

### Forward Secrecy 동작 원리

```
시간 →

[세션 1]
  ephPriv1, ephPub1 생성
  → 핸드셰이크 완료
  → exporter1 생성
  → 암호화 통신
  → ephPriv1 삭제

[세션 2]
  ephPriv2, ephPub2 생성  (새로운 키!)
  → 핸드셰이크 완료
  → exporter2 생성
  → 암호화 통신
  → ephPriv2 삭제

[미래에 장기 개인키 탈취됨]
  X ephPriv1, ephPriv2는 이미 삭제되어 복구 불가능
  X 과거 세션의 exporter1, exporter2 계산 불가능
  X 과거 통신 내용 복호화 불가능
```

**핵심**:
- 각 세션마다 **새로운 임시 키** 생성
- 세션 종료 시 **즉시 삭제** (메모리에서 완전 제거)
- 장기 개인키(DID 키)는 **서명 검증용**으로만 사용
- **HPKE는 임시 키만** 사용 → Forward Secrecy 보장

---

## 세션 암호화

핸드셰이크 완료 후 실제 메시지 암호화는 다음과 같이 이루어집니다.

### 키 유도 (Key Derivation)

```go
// session/manager.go (개념적 구현)

// 1. exporter에서 HKDF로 여러 키 생성
func deriveSessionKeys(exporter []byte, isInitiator bool) SessionKeys {
    // Client-to-Server 키
    c2sKey := HKDF-Expand(exporter, "c2s-key", 32)

    // Server-to-Client 키
    s2cKey := HKDF-Expand(exporter, "s2c-key", 32)

    if isInitiator {
        return SessionKeys{
            sendKey:    c2sKey,  // Agent A → Agent B
            receiveKey: s2cKey,  // Agent B → Agent A
        }
    } else {
        return SessionKeys{
            sendKey:    s2cKey,  // Agent B → Agent A
            receiveKey: c2sKey,  // Agent A → Agent B
        }
    }
}
```

### 암호화 흐름

```
exporter (32 bytes) - HPKE로 합의된 공유 비밀
  ↓
HKDF-Expand(exporter, "c2s-key", 32)
  → c2sKey (Client → Server 암호화 키)

HKDF-Expand(exporter, "s2c-key", 32)
  → s2cKey (Server → Client 암호화 키)

HKDF-Expand(exporter, "ack-key", 32)
  → ackKey (키 확인용)
```

### ChaCha20-Poly1305 AEAD 암호화

```go
// 메시지 암호화 (개념적 코드)
func encryptMessage(plaintext []byte, key []byte, nonce []byte) (ciphertext []byte, err error) {
    // ChaCha20-Poly1305 cipher 생성
    cipher, err := chacha20poly1305.New(key)
    if err != nil {
        return nil, err
    }

    // AEAD 암호화 (Authenticated Encryption with Associated Data)
    ciphertext = cipher.Seal(nil, nonce, plaintext, nil)

    return ciphertext, nil
}

// ciphertext 구조:
// [ 암호화된 데이터 ] + [ 16-byte Poly1305 인증 태그 ]
//   ↑ 기밀성               ↑ 무결성 + 인증
```

**AEAD 특징**:
- **기밀성** (Confidentiality): ChaCha20으로 암호화 → 내용 숨김
- **무결성** (Integrity): Poly1305 MAC → 변조 탐지
- **인증** (Authentication): 올바른 키 없이는 MAC 생성 불가

### 전체 통신 흐름

```
Agent A                                    Agent B
   │                                          │
   │ ─── HPKE 핸드셰이크 (4단계) ───────────> │
   │                                          │
   │ [동일한 exporter 공유]                    │
   │                                          │
   ├─ deriveSessionKeys(exporter, true)      ├─ deriveSessionKeys(exporter, false)
   │  sendKey = c2sKey                       │  sendKey = s2cKey
   │  recvKey = s2cKey                       │  recvKey = c2sKey
   │                                          │
   │  plaintext = "Hello"                    │
   ├─ ciphertext = Encrypt(plaintext, c2sKey)│
   │                                          │
   │  ciphertext                              │
   │ ─────────────────────────────────────>  │
   │                                          ├─ plaintext = Decrypt(ciphertext, c2sKey)
   │                                          │  "Hello"
   │                                          │
   │                                          │  plaintext = "World"
   │                                          ├─ ciphertext = Encrypt(plaintext, s2cKey)
   │  ciphertext                              │
   │ <─────────────────────────────────────  │
   ├─ plaintext = Decrypt(ciphertext, s2cKey)│
   │  "World"                                 │
```

---

## 요약

### HPKE 핸드셰이크의 핵심

1. **키 합의**: 공개키 암호화로 공유 비밀(exporter) 생성
   - Agent A: `exporter = HPKE-Seal(B_pub, info, exportCtx)` → `enc` 생성
   - Agent B: `exporter = HPKE-Open(B_priv, enc, info, exportCtx)` → 동일한 `exporter` 계산

2. **키 확인**: 암호문 없이 ackTag로 검증
   - `ackTag = HMAC(HKDF(exporter, "ack-key"), "hpke-ack|...")`
   - 양쪽이 동일한 ackTag 계산 → exporter 일치 확인

3. **Forward Secrecy**: 임시 키 사용 및 즉시 삭제
   - 각 세션마다 새로운 X25519 키쌍
   - 핸드셰이크 완료 후 즉시 삭제

4. **세션 암호화**: ChaCha20-Poly1305 AEAD
   - `c2sKey`, `s2cKey`를 exporter에서 유도
   - 양방향 독립 키 사용 (보안 강화)

### 보안 특성

| 특성 | 구현 방법 |
|------|----------|
| **기밀성** | ChaCha20-Poly1305 AEAD 암호화 |
| **무결성** | Poly1305 MAC, Ed25519 서명 |
| **인증** | DID 기반 Ed25519 서명 검증 |
| **Forward Secrecy** | 임시 X25519 키쌍 사용 및 즉시 삭제 |
| **재전송 공격 방지** | Nonce 중복 체크 |
| **타임스탬프 검증** | 5분 이내 메시지만 허용 |

---

**참고 자료**:
- HPKE RFC: [RFC 9180](https://www.rfc-editor.org/rfc/rfc9180.html)
- ChaCha20-Poly1305: [RFC 8439](https://www.rfc-editor.org/rfc/rfc8439.html)
- HKDF: [RFC 5869](https://www.rfc-editor.org/rfc/rfc5869.html)
