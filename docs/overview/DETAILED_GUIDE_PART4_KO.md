# SAGE 프로젝트 상세 가이드 - Part 4: HPKE 기반 핸드셰이크 프로토콜 및 세션 관리

> **대상 독자**: 프로그래밍 초급자부터 중급 개발자까지
> **작성일**: 2025-10-07
> **버전**: 1.2 (hpke 패키지 기준)
> **이전**: [Part 3 - DID 및 블록체인 통합](./DETAILED_GUIDE_PART3_KO.md)

---

## 목차

1. [핸드셰이크 프로토콜 개요](#1-핸드셰이크-프로토콜-개요)
2. [2단계 핸드셰이크 상세](#2-2단계-핸드셰이크-상세)
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
AI Agent A와 Agent B가 처음 만났을 때:

문제 1: 신원 확인

- B가 정말 B인지 어떻게 확인?
- A가 정말 A인지 어떻게 확인?

문제 2: 안전한 키 교환

- 대칭키를 어떻게 안전하게 공유?
- 중간자 공격 방지는?

문제 3: Forward Secrecy

- 나중에 개인키가 노출되어도 과거 대화는 안전?
- 세션마다 다른 키 사용?

문제 4: 재생 공격 방지

- 같은 메시지를 여러 번 보내는 것 방지?
- Nonce 관리는 어떻게?

**SAGE의 해결책: 2단계 핸드셰이크**

```
┌─────────────────────────────────────────────────────┐
│   Request = HPKE-Request (A → B)                    │
│   → B.kemPub로 HPKE(Base) 부트스트랩                │
│   → content = enc || Seal( ephC, ctxId, nonce, ts ) │
└─────────────────────┬───────────────────────────────┘
                      ↓
┌─────────────────────────────────────────────────────┐
│   Response = HPKE-Response (B → A)                  │
│   → A.kemPub로 HPKE(Base) 부트스트랩                │
│   → content = enc2 || Seal( ephS, ackTag, ts )      │
└─────────────────────┬───────────────────────────────┘
                      ↓
┌─────────────────────────────────────────────────────┐
│   Complete                 │
│   → 세션 확정: seed = HKDF(exporter||ssE2E, …)     │
│   → KeyID 발급 및 바인딩                            │
└─────────────────────────────────────────────────────┘
결과:
Yes 상호 인증 완료 (Mutual Authentication)
Yes 안전한 세션 키 공유 (Secure Key Exchange)
Yes Forward Secrecy 보장
Yes 재생 공격 방지 (Nonce)
```

### 1.2 TLS 1.3과 무엇이 다른가요?

```
TLS 1.3 핸드셰이크:
┌──────────────────────────────────────┐
│ 1. ClientHello                       │
│    - 지원하는 암호 스위트              │
│    - 랜덤 nonce                      │
│    - 임시 키 공유 (KeyShare)         │
├──────────────────────────────────────┤
│ 2. ServerHello                       │
│    - 선택한 암호 스위트                │
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

SAGE 핸드셰이크 (HPKE 기반):
┌──────────────────────────────────────┐
│ 1. Request = HPKE-Request (A → B)    │
│    - HPKE Base: SetupBaseS(B.kemPub)  │
│    - enc || Seal( ephC, ctxId, ts… ) │
│    - A의 DID 서명(메타데이터)         │
├──────────────────────────────────────┤
│ 2. Response = HPKE-Response (B → A)  │
│    - HPKE Base: SetupBaseS(A.kemPub)  │
│    - enc2 || Seal( ephS, ackTag… )   │
│    - B의 DID 서명(메타데이터)         │
├──────────────────────────────────────┤
│ 3. Complete (Control-plane)          │
│    - 세션 확정: exporter 파생          │
│    - KeyID 발급 및 바인딩              │
└──────────────────────────────────────┘

```

**차이점**

| 구분            | TLS 1.3      | SAGE HPKE                           |
| --------------- | ------------ | ----------------------------------- |
| 신원 증명       | X.509 인증서 | DID + Ed25519 서명                  |
| 인증 기관       | 중앙 CA      | 블록체인 Resolver                   |
| 키 합의         | ECDHE        | X25519 (HPKE Base)                  |
| 세션 암호화     | AES-GCM      | ChaCha20-Poly1305 (session.Manager) |
| 메시지 서명     | Finished MAC | RFC 9421 + HMAC-SHA256              |
| Forward Secrecy | Yes          | Yes                                 |
| 블록체인 통합   | No           | Yes DID Resolver 기반               |

### 1.3 A2A 프로토콜과의 연결

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

## 2. 2단계 핸드셰이크 상세

### 2.1 단계 0: 사전 준비 (Pre-flight)

```
[Client]
- Ed25519 메타 서명 키 생성
- DID 발급 (Part 3 참고)
- Resolver로 서버 KEM 키 조회 (ResolveKEMKey)

[Server]
- X25519 KEM 정적 키 생성
- DID Document에 공개키 게시
- session.Manager 준비 (MaxAge, IdleTimeout, MaxMessages)
```

### 2.2 단계 1: Init (Client → Server)

클라이언트는 `hpke.Client.Initialize`에서 Init 페이로드를 만든 뒤 A2A `SendMessage`를 호출합니다.

```go
info      := c.info.BuildInfo(ctxID, initDID, peerDID)
exportCtx := c.info.BuildExportContext(ctxID)

enc, exporter, _ := keys.HPKEDeriveSharedSecretToPeer(peerPub, info, exportCtx, 32)

nonce := uuid.NewString()
payload := {enc, info, exportCtx, nonce, ts}

msg := buildA2AMessage(ctxID, TaskHPKEComplete, payload)
meta := signStruct(c.key, deterministicBytes(msg), c.DID)
resp, err := c.a2a.SendMessage(ctx, &a2a.SendMessageRequest{Request: msg, Metadata: meta})
```

#### 처리 흐름

Init 페이로드
{ enc, info, exportCtx, nonce, ts }

```
┌─────────────────────────────────────────┐
│ Agent A (클라이언트)                     │
│                                         │
│ 1. DID/DID 레지스트리에서 B의 HPKE      │
│    정적 공개키 pkB(X25519) 조회         │
│ 2. info, exportCtx 구성                  │
│ 3. HPKE SetupBaseS(pkB, info) 실행      │
│    → enc = pkE(A의 임시 X25519 공개키)  │
│    → ctxS                                │
│ 4. exporter = ctxS.Export(exportCtx,32) │
│ 5. nonce, ts 생성(재생/시계 스큐 방지)  │
│ 6. payload = {enc, info, exportCtx, nonce, ts}   │
│ 7. A2A 메시지 결정론적 직렬화           │
│    → Ed25519 서명(meta)                  │
│ 8. gRPC SendMessage(Init) 전송          │
└──────────────┬──────────────────────────┘
               │ [네트워크]
               ↓
┌─────────────────────────────────────────┐
│ Agent B (서버)                           │
│                                         │
│ 1. Ed25519 서명 검증(결정론적 바이트)    │
│ 2. ts 검증(±maxSkew), nonce 재사용 차단  │
│ 3. HPKE SetupBaseR(skB, enc, info)      │
│    → ctxR                                │
│ 4. exporterB = ctxR.Export(exportCtx,32)│
│ 5. exporterB를 세션 시드 후보로 보관     │
│ 6. kid 생성 및 Ack 준비                  │
└─────────────────────────────────────────┘

```

### 2.3 단계 2: Ack (Server → Client)

서버는 `hpke.Server.SendMessage`에서 Init을 검증하고 Ack 응답 및 세션 생성

```go
_, sid, _, err := s.sessMgr.EnsureSessionFromExporterWithRole(
    combined,
    "sage/hpke+e2e v1",
    false, // receiver
    nil,
)
kid := "kid-" + uuid.NewString()
s.sessMgr.BindKeyID(kid, sid)

ack := MakeAckTag(
    combined,
    msg.ContextID,
    pl.Nonce,
    kid,
    pl.Info,
    pl.ExportCtx,
    pl.Enc,
    pl.EphC,
    ephSPubBytes,
    []byte(pl.InitDID),
    []byte(pl.RespDID),
)
```

Ack 메타데이터 예시

```json
{
  "kid": "kid-e0b6…",
  "ackTagB64": "t9uV4Ck+tER+VAWm…",
  "ts": "2025-10-07T09:00:00Z"
}
```

#### 처리 흐름

```
┌─────────────────────────────────────────┐
│ Agent B (서버)                           │
│                                         │
│ 1. ackTag = HMAC(exporterB,             │
│    ctxID || nonce || kid)               │
│ 2. metadata = { kid, ackTagB64, ts }    │
│ 3. A2A Ack 결정론적 직렬화 →  Ed25519 서명   │
│ 4. gRPC SendMessage(Ack) 전송            │
└──────────────┬──────────────────────────┘
               │ [네트워크]
               ↓
┌─────────────────────────────────────────┐
│ Agent A (클라이언트)                     │
│                                         │
│ 1. Ed25519 서명 검증                     │
│ 2. ackTagB64 decode                      │
│ 3. expect = HMAC(exporter,ctxID || nonce || kid) │
│ 4. constant-time 비교                    │
│ 5. 일치 시 → Complete 단계로 진행        │
└─────────────────────────────────────────┘

```

### 2.4 Complete (Client 확인)

클라이언트는 Ack를 받은 후 `ackTag` 검증 및 세션 생성

```go
ackTag, _ := base64.RawURLEncoding.DecodeString(ackTagB64)
expect := makeAckTag(exporter, ctxID, nonce, kid)
if !hmac.Equal(expect, ackTag) { return fmt.Errorf("ack tag mismatch") }
_, sid, _, _ := c.sessMgr.EnsureSessionFromExporterWithRole(exporter, "sage/hpke v1", true, nil)
c.sessMgr.BindKeyID(kid, sid)
```

```
┌─────────────────────────────────────────┐
│ Agent A (클라이언트)                     │
│                                         │
│ 1. AckTag 검증 성공                      │
│ 2. EnsureSessionFromExporterWithRole(   │
│    exporter, "sage/hpke v1",            │
│    initiator = true )                    │
│ 3. sid 계산 → 방향별 키 유도            │
│ 4. BindKeyID(kid, sid)                   │
│ 5. HPKE 핸드셰이크 종료                  │
│    (이후: 세션 키로 암호화·서명 사용)   │
└─────────────────────────────────────────┘

```

### 핸드셰이크 전체 과정

사전 조건

- **S**는 정적 X25519 KEM 키(HPKE 수신자용)와 정적 Ed25519 서명 키를 DID로 공개
- **C**는 S의 정적 키들을 DID Resolver로부터 얻을 수 있음

```
1) 준비(클라이언트)
   1. C가 DID Resolver로부터 S의 정적 X25519 KEM 공개키와 Ed25519 서명키를 조회.
   2. C가 `info`, `exportCtx`를 고정 형식으로 구성.
   3. C가 HPKE SetupBaseS(pkB_static, info) 실행 →
   `enc`(송신자 HPKE 임시 공개키)와 `exporter_HPKE` 획득.
   4. C가 ephC(클라이언트 E2E용 임시 X25519 공개키) 생성, `nonce`와 `ts` 준비.
   5. C가 `payload = {initDid, respDid, info, exportCtx, enc, ephC, nonce, ts}`를 Ed25519(클라이언트 DID)로 서명.
      (선택) 쿠키/퍼즐이 설정되어 있으면 `cookie`를 메타데이터에 첨부.

2) 전송
   6. C → S: `SecureMessage(payload, signature, [cookie])`

3) 1차 검증(서버: 저비용 먼저)

   7. (선택) 쿠키/퍼즐 확인: Verifier가 설정되어 있으면 `cookie` 유효성 검사를 가장 먼저 수행(누락/불일치 시 즉시 거절).
   8. 클라이언트 DID 서명 검증**(payload에 대한 Ed25519) → 발신자 무결성·인증.
   9. 타임스탬프/스큐/리플레이**(nonceStore), info/exportCtx 일치 검사.

4) 키 합의(서버)

   10. HPKE Open(skB_static, enc, info) → `exporter_HPKE'` 재현.
   11. ephS(서버 E2E용 임시 X25519 공개키) 생성.
   12. ssE2E = X25519(ephS, ephC) 계산.
   13. combined = HKDF-Extract(salt=exportCtx, IKM = exporter_HPKE || ssE2E)
    → 이 값을 세션 시드로 사용(사용 즉시 메모리에서 zeroize).

5) 세션 생성 + 응답(서버)

   14. 서버 세션 매니저에 `combined`로 수신자 세션 생성, `kid` 발급/바인딩.
   15. ackTag = HMAC(combined, ctxID|nonce|kid|TH(info,exportCtx,enc,ephC,ephS,IDs)) 계산.
   16. 응답(envelope) 구성: `{v, task, ctx, kid, ephS, ackTagB64, ts, did, infoHash, exportCtxHash, enc, ephC}`
    → 이를 서버 Ed25519로 서명(sigB64).
   17. S → C: `{envelope, sigB64}` 전송.

6) 2차 검증(클라이언트: 키 확인 → 신원 확인)

   18. C가 ssE2E = X25519(ephC_priv, ephS) → combined 재계산
   19. ackTag를 먼저 검증(키 일치·MITM/UKS 차단),
    `enc/ephC` 에코와 `infoHash/exportCtxHash`도 확인.
   20. DID Resolver로 서버 Ed25519 공개키 조회 → 서버 서명(sigB64) 검증(신원 바인딩).
   21. 클라이언트 세션 매니저에 `combined`로 발신자 세션 생성, `kid` 바인딩.

7) 트래픽 키/논스(양방향)

   22. `DeriveTrafficKeys(combined)`로 c2s/s2c 각 키와 IV 파생.
    전송 시 seq 기반 nonce = IV XOR seq로 재사용 불가 보장.

8) 종료/폐기

   23. 세션 종료 시: ephC/ephS/combined/exporter/세션키 모두 zeroize(메모리 덮어쓰기).
```

결과(보안 성질)

- **전방 안전성(FS)**: ephC/ephS를 섞어 사용하므로 이후 정적 키 유출에도 과거 세션 복호화 불가
- **상호 인증**: 요청은 **클라이언트 DID 서명**, 응답은 **서버 DID 서명**으로 신원 바인딩
  (키 일치는 **ackTag**로 먼저 확인 → MITM/UKS 조기 차단)
- **세션 분리/키 분리**: `info/exportCtx` 고정 포맷, HKDF 라벨 기반 파생으로 방향/용도 분리
- **DoS 억제**: (설정 시) 쿠키/퍼즐 **초기에 검사**하여 고비용 HPKE/ECDH 전에 거절
- **리플레이/다운그레이드 방지**: nonceStore, `infoHash/exportCtxHash`/스위트 고정, 에코 검증

## 3. 클라이언트 구현

`hpke/client.go`의 구조는 다음과 같습니다.

```go
type Client struct {
    a2a      a2a.A2AServiceClient
    resolver did.Resolver
    key      sagecrypto.KeyPair // Ed25519 서명용
    DID      string
    info     InfoBuilder
    sessMgr  *session.Manager

    cookies CookieSource // optional
}

```

**주요 흐름**

1. **`NewClient`**: gRPC 커넥션, DID Resolver, session.Manager 등을 주입
2. **`Initialize`**: HPKE Base 수행 → `enc`/`exporter`, nonce 생성 → A2A로 전송 → Ack 검증
3. **세션 이용**: `sessMgr.GetByKeyID(kid)`로 세션을 찾아 `Encrypt`, `Decrypt`, `SignCovered`, `VerifyCovered` 등 호출하여 암복호화 수행

## 4. 서버 구현

```go
type Server struct {
    key      sagecrypto.KeyPair   // X25519 KEM 정적 키
    DID      string
    resolver did.Resolver         // DID → Ed25519 공개키
    sessMgr  *session.Manager
    info     InfoBuilder
    maxSkew  time.Duration
    nonces   *nonceStore
    binder   KeyIDBinder          //  커스텀 kid 발급
    cookies CookieVerifier        // optional anti-DoS
}
```

1. Task ID가 `hpke/complete@v1`인지 확인
2. 메타데이터에 DID가 있는지, Resolver로 Ed25519 키를 찾을 수 있는지 검사
3. `verifySenderSignature`로 Ed25519 서명을 검증
4. `ParseHPKEInitPayload`로 `info`, `exportCtx`, `enc`, `nonce`, `ts` 추출
5. `info/exportCtx`가 InfoBuilder 결과와 동일한지, 타임스탬프가 허용 범위인지, Nonce가 새 것인지 검증
6. `keys.HPKEOpenSharedSecretWithPriv`로 `exporter`를 재현
7. `session.Manager`에 세션을 만들고 `kid`를 바인딩(필요하면 `binder.IssueKeyID`).
8. `ackTag`를 계산하여 응답 메타데이터 생성

## 5. 세션 생성 및 키 유도

핵심 함수 `session.Manager.EnsureSessionFromExporterWithRole`

```go
sess, sid, _, err := mgr.EnsureSessionFromExporterWithRole(
    exporter,            // HPKE exporter (32 bytes)
    "sage/hpke v1",       // 레이블
    isInitiator,         // true=클라이언트, false=서버
    nil,                 // 추가 바인딩 없음
)
```

- HPKE exporter는 HKDF-Extract/Expand를 거쳐 ChaCha20-Poly1305 키와 HMAC 키로 분리됩니다.
- initiator(클라이언트)와 responder(서버)의 방향 키가 자동으로 나뉘어 저장됩니다.
- 반환된 `sid`는 내부 세션 ID이며 `BindKeyID`로 랜덤 문자열 `kid`에 맵핑합니다.

### 5.1 세션 파라미터

```go
// HPKE 권장 경로: exporter(32B)를 바로 사용
// (서버/클라이언트 양쪽에서 동일 값)

exporter := HPKE Export(..., 32)  // 코드상: keys.HPKEDerive... / HPKEOpen... 결과
label    := "sage/hpke v1"        // 기본값
```

```go
// 호환 경로(기본 hadnshake): ECDH 공유비밀 + 임시키 솔트
type Params struct {
    ContextID    string  // 핸드셰이크 컨텍스트 ID
    SelfEph      []byte  // 내 임시 공개키(32B)
    PeerEph      []byte  // 상대 임시 공개키(32B)
    Label        string  // 도메인 분리 (기본: "a2a/handshake v1")
    SharedSecret []byte  // (기본 hadnshake) ECDH 공유 비밀(32B)
}
```

역할:

- exporter(32바이트) 자체가 PRK에 해당하는 세션 시드 역할
- 양측이 동일 exporter를 얻으면 같은 세션 ID/키를 결정론적으로 생성
- label로 도메인 분리

### 5.2 세션 시드 유도

**HPKE 권장 경로**

```go
// sessionSeed := exporter (32B PRK 유사물)
sessionSeed := exporter
```

**기본 hadnshake ECDH 경로 (호환)**

```go
func DeriveSessionSeed(sharedSecret []byte, p Params) ([]byte, error) {
    // label, contextID, 임시공개키(lo,hi)로 salt 생성 → HKDF-Extract
    // 코드: session/DeriveSessionSeed (동일)
}
```

- HPKE 경로에서는 **별도 솔트/정렬 없이** exporter를 그대로 세션 시드로 사용

### 5.3 세션 ID 계산

```go
func ComputeSessionIDFromSeed(seed []byte, label string) (string, error) {
    // SHA256(label || seed)의 앞 16바이트를 Base64 URL-safe로 인코딩
    // 결과: 22자 내외, 충돌확률 ≈ 2^-128
}
```

- 세션 ID는 label과 “같은 시드(seed)”를 넣어 만든 결정론적 해시 값으로, 양쪽이 같은 시드를 갖게되면 SID도 동일해짐

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
Yes 방향별 독립 키 (크로스 공격 방지)
Yes 암호화/서명 분리 (도메인 분리)
Yes 버전 관리 가능 (v1, v2...)
Yes 확장 용이 (새 키 타입 추가 가능)
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

### 5.6 키 확인(Key Confirmation)

**개념**
서버가 진짜 같은 exporter를 파생했는지 **HMAC 태그(ackTag)** 로 확인

**프로토콜**

1. 클라이언트: `nonce` 생성 후 `TaskHPKEComplete` 전송(서명 포함)
   페이로드: `enc`, `info`, `exportCtx`, `nonce`, `ts` …

2. 서버:

   - 메타서명 검증, 타임스큐(기본 ±2분) 확인, `(ctxID|nonce)` 리플레이 차단
   - `HPKE Open`으로 exporter 재현 → 서버 로컬 세션 생성
   - `kid` 발급/바인딩
   - `ackTag = HMAC-SHA256(K_conf, "hpke-ack|ctxID|nonce|kid")` 계산 후 b64로 되돌림
     _(코드: `makeAckTag(se, ctxID, nonce, kid)`; `se(exporter)`에서 파생된 확인키 사용)_

3. 클라이언트: 수신한 `ackTagB64`를 **HMAC 상수시간 비교**로 검증 → 성공 시 `kid→sid` 바인딩

## 6. 세션 관리자 (HPKE 경로 우선)

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
// 양쪽 동일:
EnsureSessionFromExporterWithRole(exporter, "sage/hpke v1", initiator, cfg)
// → (session, sid, existed, err)

// 필요 시 kid까지 즉시 바인딩:
EnsureAndBindFromExporterWithRole(exporter, "sage/hpke v1", initiator, kid, cfg)
```

**기본 handshake 호환 경로**

```go
// Params로 PRK(seed) 유도 → SID → 세션
EnsureSessionWithParams(p Params, cfg *Config)
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

## 7. 이벤트 기반 아키텍처 (기본 handshake)

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

## 8. 보안 고려사항

### 8.1 타이밍 공격 방지

```go
// hpke/common.go
func verifySignature(payload, signature []byte, senderPub crypto.PublicKey) error {
if len(signature) == 0 {
    return errors.New("missing signature")
}

// Support either a custom Verify interface or raw ed25519.PublicKey
type verifyKey interface {
    Verify(msg, sig []byte) error
}

switch pk := senderPub.(type) {
case verifyKey:
    if err := pk.Verify(payload, signature); err != nil {
        return fmt.Errorf("signature verify failed: %w", err)
    }
    return nil
case ed25519.PublicKey:
    if !ed25519.Verify(pk, payload, signature) {
        return errors.New("signature verify failed: invalid ed25519 signature")
    }
    return nil
default:
    return fmt.Errorf("unsupported public key type: %T", senderPub)
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
// hpke/types.go
type CookieSource interface {
    GetCookie(ctxID, initDID, respDID string) (string, bool)
}

type CookieVerifier interface {
    Verify(cookie, ctxID, initDID, respDID string) bool
}
```

```go
// 클라이언트에 통합
func (c *Client) WithCookieSource(src CookieSource) *Client {
    c.cookies = src
    return c
}

if c.cookies != nil {
    if cookie, ok := c.cookies.GetCookie(ctxID, initDID, peerDID); ok && cookie != "" {
        if msg.Metadata == nil {
            msg.Metadata = map[string]string{}
        }
        msg.Metadata["cookie"] = cookie
    }
}

// 서버에 통합
type ServerOpts struct {
    // ...
    Cookies CookieVerifier // optional DoS cookie/puzzle verifier
}

// In HandleMessage(...) BEFORE heavy crypto:
if s.cookies != nil {
    cookie := msg.Metadata["cookie"]
    if !s.cookies.Verify(cookie, msg.ContextID, pl.InitDID, pl.RespDID) {
        return nil, fmt.Errorf("cookie required or invalid") // early reject
    }
}
```

**Cookie Verifier 예시**

1. HMAC 쿠키 (고속·저비용)

```go
// "hmac:<b64url(HMAC_SHA256('SAGE-Cookie|v1|'||ctxID||'|'||initDID||'|'||respDID))>"

type hmacCookieVerifier struct{ secret []byte }

func (v *hmacCookieVerifier) Verify(cookie, ctxID, initDID, respDID string) bool {
    const prefix = "hmac:"
    if len(cookie) <= len(prefix) || cookie[:len(prefix)] != prefix {
        return false
    }
    gotRaw, err := base64.RawURLEncoding.DecodeString(cookie[len(prefix):])
    if err != nil {
        return false
    }
    m := hmac.New(sha256.New, v.secret)
    m.Write([]byte("SAGE-Cookie|v1|"))
    m.Write([]byte(ctxID)); m.Write([]byte("|"))
    m.Write([]byte(initDID)); m.Write([]byte("|"))
    m.Write([]byte(respDID))
    exp := m.Sum(nil)
    return hmac.Equal(gotRaw, exp) // constant-time compare
}

type hmacCookieSource struct{ secret []byte }

func (s *hmacCookieSource) GetCookie(ctxID, initDID, respDID string) (string, bool) {
    m := hmac.New(sha256.New, s.secret)
    m.Write([]byte("SAGE-Cookie|v1|"))
    m.Write([]byte(ctxID)); m.Write([]byte("|"))
    m.Write([]byte(initDID)); m.Write([]byte("|"))
    m.Write([]byte(respDID))
    out := base64.RawURLEncoding.EncodeToString(m.Sum(nil))
    return "hmac:" + out, true
}
```

2. PoW 퍼즐 쿠키 (봇·스팸 트래픽 감쇠)

```go
// "pow:<nonceHex>:<hex(sha256('SAGE-PoW|'||ctxID||'|'||initDID||'|'||respDID||'|'||nonce))>"
// difficulty = number of leading zero nibbles in SHA-256 digest

type powCookieVerifier struct{ difficulty int }

func leadingZeroNibbles(sum []byte) int {
    n := 0
    for _, b := range sum {
        if b>>4 == 0 { n++ } else { return n }
        if b&0x0F == 0 { n++ } else { return n }
    }
    return n
}

func (v *powCookieVerifier) Verify(cookie, ctxID, initDID, respDID string) bool {
    var nonce, hexHash string
    _, err := fmt.Sscanf(cookie, "pow:%s:%s", &nonce, &hexHash)
    if err != nil { return false }
    sum := sha256.Sum256([]byte("SAGE-PoW|" + ctxID + "|" + initDID + "|" + respDID + "|" + nonce))
    if hex.EncodeToString(sum[:]) != hexHash {
        return false
    }
    return leadingZeroNibbles(sum[:]) >= v.difficulty
}

type powCookieSource struct{ difficulty int }

func (s *powCookieSource) GetCookie(ctxID, initDID, respDID string) (string, bool) {
    for nonce := 0; nonce < 1<<24; nonce++ { // bounded search for tests
        ns := fmt.Sprintf("%x", nonce)
        sum := sha256.Sum256([]byte("SAGE-PoW|" + ctxID + "|" + initDID + "|" + respDID + "|" + ns))
        if leadingZeroNibbles(sum[:]) >= s.difficulty {
            return fmt.Sprintf("pow:%s:%s", ns, hex.EncodeToString(sum[:])), true
        }
    }
    return "", false
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

## 9. 실전 예제 (HPKE로 치환)

### 9.1 전체 흐름 요약

- **클라이언트**

  1. `resolver.ResolvePublicKey(peerDID)`
  2. `keys.HPKEDeriveSharedSecretToPeer(peerPub, info, exportCtx, 32)` → `(enc, exporter)`
  3. `EnsureSessionFromExporterWithRole(exporter, "sage/hpke v1", true, cfg)`
  4. `TaskHPKEComplete` 전송(서명/nonce/ts 포함, `enc`, `info`, `exportCtx`)
  5. 응답 `kid`, `ackTagB64` 수신 → `makeAckTag(exporter, ctxID, nonce, kid)`로 검증
  6. `BindKeyID(kid, sid)` → 완료

- **서버**

  1. 메타 DID/서명 검증, 시간창/리플레이 검사
  2. `keys.HPKEOpenSharedSecretWithPriv(priv, enc, info, exportCtx, 32)` → `exporter`
  3. `EnsureSessionFromExporterWithRole(exporter, "sage/hpke v1", false, cfg)`
  4. `kid` 발급/바인딩(`KeyIDBinder` 선택)
  5. `ackTag := makeAckTag(exporter, ctxID, nonce, kid)` → `ackTagB64`로 응답

### 9.2 gRPC 배선/로그 예시

- gRPC 서비스는 그대로 `A2AService`를 사용
- 태스크 ID는 `TaskHPKEComplete` 하나로 처리
- `make test-hpke` 로 예제 확인 가능

## 요약

Part 4에서 다룬 내용:

1. **핸드셰이크 프로토콜 개요**: 필요성, TLS 비교, A2A 통합
2. **2단계 핸드셰이크**: Request, Response, Complete 상세
3. **클라이언트 구현**: 구조, 시퀀스, 에러 처리
4. **서버 구현**: 구조, 상태 관리, 자동 정리
5. **세션 생성**: 시드 유도, 세션 ID 계산, 방향별 키 유도, AEAD 초기화
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
