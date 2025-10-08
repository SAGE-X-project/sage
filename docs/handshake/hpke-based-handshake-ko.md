# SAGE HPKE-based Handshake

SAGE( Secure Agent Guarantee Engine )에서 **에이전트 간 안전한 세션**을 성립시키기 위해 [HPKE (RFC 9180)](https://www.rfc-editor.org/rfc/rfc9180) 기반 사전 합의를 제공하는 Go 패키지입니다.

기존 [A2A 프로토콜](https://a2a-protocol.org/latest/topics/what-is-a2a/#a2a-request-lifecycle)의 확장으로 **gRPC** 메시지 상에서 수행되며, 세션 수립 이후의 애플리케이션 메시지는 **세션 암호화 + HTTP Message Signatures (RFC 9421)** 로 보호됩니다.

<img src="../assets/SAGE-hpke-handshake.png" width="520" height="520"/>

클라이언트로부터 전달받은 임시 세션키로, 서버는 shared secret을 생성하고, 이를 기반으로 세션 seed를 생성합니다.

## 핵심 특징

- **DID 메타 검증 + 부트스트랩 무결성**  
  A2A 메타데이터의 DID와 Ed25519 서명을 검증합니다. 발신자 신원을 고정(anchoring)해 **MITM/악의적 발신자**를 차단합니다.

- **HPKE Base(단일 캡슐화)로 키 합의**  
  수신자 정적 X25519 KEM 공개키를 DID Document로 조회하고, 발신자가 생성한 `enc`(ephemeral encapsulation)와 함께 HPKE **exporter secret**을 파생합니다.

- **PFS 제공 (E2E add-on)**  
  Base 모드의 한계를 보완하기 위해 **ephemeral-ephemeral X25519 교환**(`ephC↔ephS`)을 추가로 수행하고, `exporterHPKE || ssE2E`를 HKDF로 결합해 최종 세션 시드를 만듭니다. 수신자 정적 KEM 개인키가 **사후 유출되더라도** 과거 트래픽 복호화 위험을 줄입니다.

- **세션·논스 관리**  
  `session.Manager`가 **MaxAge/IdleTimeout/MaxMessages** 정책과 **nonce 재사용(Replay) 방지**를 제공합니다. HPKE exporter 기반 **ackTag**로 키 확인(key confirmation)도 수행합니다.

- **간단한 1RTT 왕복**
  Base 모드는 **Client → Server(Init)**, **Server → Client(Ack)** 두 단계로 끝납니다.PFS 를 추가한 client와 server을 이용하는 경우에도 왕복 수는 같고, 서버가 `ephS`만 추가 반환합니다.

## 설치

```bash
go get github.com/sage-x-project/sage/hpke
```

## 아키텍처 & 구성요소

```text
hpke/
├── client.go               # HPKE 클라이언트(Init, ack 검증, 세션 바인딩)
├── server.go               # HPKE 서버(메타검증, exporter 재현, ack 생성)
├── common.go           # 필요 helper 함수들 집합
session/
├── manager.go              # 세션 생성/조회/만료/ReplayGuard
└── session.go              # AEAD 암복호화, 안전한 키 폐기
did/
└── resolver.go             # DID → (신원/HPKE KEM) 공개키 조회
```

### Info / Export Context (권장 규격)

```go
type InfoBuilder interface {
    BuildInfo(ctxID, initDID, respDID string) []byte
    BuildExportContext(ctxID string) []byte
}

type DefaultInfoBuilder struct{}

func (DefaultInfoBuilder) BuildInfo(ctxID, initDID, respDID string) []byte {
    return []byte("sage/hpke v1|ctx=" + ctxID + "|init=" + initDID + "|resp=" + respDID)
}
func (DefaultInfoBuilder) BuildExportContext(ctxID string) []byte {
    return []byte("exporter:" + ctxID)
}
```

- `info`는 **세션 문맥**(ctxID, 양측 DID)을 고정 문자열과 함께 포함해 **키 재사용/교차-문맥**을 방지합니다.
- `exportCtx`는 exporter의 **HKDF label(salt/ctx)** 로 쓰여, **세션 구분**과 **키 유출 영향 범위를 축소** 합니다.

## 핸드셰이크 흐름

### 0) 사전 조건

- 양측 에이전트의 DID는 체인에 등록되어 있고, 서버는 DID Document에 **정적 X25519 KEM 공개키**를 노출합니다.
- 클라이언트는 서버 DID로 **KEM 공개키를 Resolve** 합니다.

### 1) Initialize (Client → Server)

클라이언트는 다음을 포함해 `TaskHPKEComplete` 메시지를 전송합니다.

- `enc` : HPKE encapsulation (32B)
- `info`, `exportCtx` : InfoBuilder 산출값
- `nonce`, `ts` : 재생방지/신선도 확인용
- _(권장)_ `ephC` : PFS add-on 을 위한 클라 ephemeral X25519 공개키(32B)

메타데이터는 **발신자 DID 서명(Ed25519)** 으로 보호됩니다.

### 2) Ack (Server → Client)

서버는:

1. DID 메타 검증 및 **신선도(±maxSkew)**, **리플레이(nonce)**, **info/exportCtx 일치**를 점검
2. HPKE Base로 **exporterHPKE** 재현
3. 서버 **`ephS` 생성** 후 `ssE2E = ECDH(ephS, ephC)` 계산
4. 최종 세션 시드:

```
Base only:
    seed = exporterHPKE
Base + PFS add-on:
    seed = HKDF-Extract( salt = exportCtx,
                         IKM  = exporterHPKE || ssE2E )
           → HKDF-Expand(info = "SAGE-HPKE+E2E-Combiner", L=32)
```

- **서버 정적 KEM 개인키가 유출**  
  과거 캡처 트래픽의 `enc`로부터 **`exporterHPKE`는 재현**됩니다.  
  → 하지만 공격자는 과거시점의 클라이언트의 eph 개인키 **또는** 서버 eph 개인키가 필요로하며, 이들은 핸드셰이크 직후 파기되므로 **`ssE2E`는 만들수 없습니다.**  
  → HKDF로 **두 secret을 섞어** 쓰기 때문에 최종 `combined`는 여전히 안전하며 **PFS 를 보장합니다.**
- **반대로**, 클라이언트의 임시 개인키가 나중에 유출되더라도 서버 정적 KEM 개인키(블록체인 등록)가 안전하면 **`exporterHPKE`는 알 수 없어 최종 `combined`는 안전합니다**
- **둘 다 나중에 유출**(서버 정적 KEM sk + 클라 eph 개인키 **또는** 서버 eph 개인키)되면 당연히 과거 세션도 재현 가능합니다.

따라서 **서버와 클라이언트의 임시 세션키는 생성 즉시 사용 후 메모리에서 파기**한다면 안전한 세션 통신을 할 수 있습니다.

5. 세션 생성 및 `kid` 바인딩, **ackTag** 생성:

```
ackKey = HKDF-Expand(PRK=seed, info="ack-key", L=32)
ackTag = HMAC-SHA256(ackKey, "hpke-ack|"+ctxID+"|"+nonce+"|"+kid)
```

6. 응답 메타데이터로 **`kid`**, **`ackTagB64`**, _(선택)_ **`ephS`** 를 반환

클라이언트는 동일 절차로 `seed`를 계산해 `ackTag`를 검증하고, 세션을 생성·`kid`에 바인딩합니다.

> **왕복 수**: Base/추가 PFS 모두 **1RTT(Init/Ack)** 로 유지됩니다.

## 운영 모드

### A) HPKE Base

- 구현이 단순하고 상호운용성이 높습니다.
- **제약**: 수신자 **정적 KEM 개인키 사후 유출** 시, 과거 네트워크 캡처(`enc`, ciphertext)만으로 **과거 exporter 복원 → 과거 트래픽 복호화** 위험이 있습니다(RFC 9180의 FS 모델 한계).

→ 이 모드만 사용할 경우 **KEM 키 회전 주기 단축**(짧은 TTL), HSM 보관 등 운영적 보완이 필요합니다.

참고) hpke/hpke_test.go (블록체인 DID를 기반으로 하는 경우, 키 회전에 한계)

### B) HPKE Base + E2E PFS add-on (권장)

- Base exporter에 **ephemeral-ephemeral ECDH** 를 추가로 결합하여 **수신자 정적 KEM 키 유출에도** 과거 세션 seed를 복원하기 어렵게 만듭니다.
- 필드 추가: `ephC`(Init), `ephS`(Ack)
- 코드 변경은 **payload 파서와 seed 결합(HKDF) 함수**만으로 제한됩니다.

## 기존 HPKE 부적합 문제

- **Base vs Envelope 모드 불일치**  
  과거 서버가 “envelope(enc+ct)” 파싱을, 클라이언트는 Base(enc만)를 사용하던 **모드 불일치**가 있었습니다.  
  → **서버/클라이언트 모두 Base로 통일**하고, 선택적으로 **PFS add-on** 을 추가해 문제를 해소했습니다.  
  → 파서는 **`ParseHPKEBaseInitPayload`**(Base) 또는 **`ParseHPKEInitPayloadWithEphC`**(PFS add-on)을 사용합니다.

- **“HPKE Base로 PFS 보장” 오해**  
  RFC 9180 Base만으로는 **수신자 정적 KEM 키 유출**에 대한 PFS가 **보장되지 않습니다.**  
  → 문서/코드를 명확히 분리: **Base(단순/상호운용) vs Base+PFS add-on(강화 FS)**  
  → 운영 가이드(짧은 KEM TTL, HSM, encapsulation 은닉)와 함께 제시합니다.

## 보안 설계(문제 → 대응)

### 1) 수신자 KEM 키 사후 유출에 따른 과거 트래픽 복호화 위험

- **문제**: Base는 `enc`가 평문으로 전송되므로, 수신자 정적 KEM 개인키가 **나중에** 유출되면 과거 exporter를 재현할 수 있음.
- **대응(권장)**: **PFS add-on** 으로 `ssE2E` 결합 → seed를 **exporterHPKE ∥ ssE2E** 의 함수로 만들면, 수신자 정적키만으로는 seed를 재현 불가. 또한 **ephemeral 키 안전폐기**를 정책화.

### 2) 중간자 공격(MITM)·다운그레이드

- **위협**: ctxID/상대 DID를 바꿔치기하거나 info/exportCtx를 교란해 다른 문맥으로 exporter를 재사용 유도.
- **대응**:

  - **DID 서명 검증**(Ed25519)으로 발신자/무결성 보장
  - **canonical info/exportCtx**를 **ctxID, initDID, respDID**로 구성 → **교차-문맥/다운그레이드** 차단
  - **ackTag**(seed로부터 파생) 검증으로 **키 확인(key confirmation)** 수행

### 3) 재전송(Replay) 공격

- **위협**: 동일 패킷 혹은 **동일 Nonce** 재사용
- **대응**:

  - Init: **nonce + ts**(±maxSkew) 체크 및 **NonceStore** 기록
  - RFC 9421 요청: `Signature-Input`의 **nonce**를 `ReplayGuardSeenOnce(kid, nonce)` 로 거부
  - 세션 정책: **MaxMessages** 제한

### 4) 세션 탈취/키 재사용

- **대응**:

  - 세션은 **seed에서 HKDF** 로 파생되며, **MaxAge/IdleTimeout** 만료 → 즉시 폐기
  - **kid ↔ sessionID** 바인딩. `kid` 유출만으로는 메시지 복호화 불가(세션 키가 필요)

### 5) 무결성/기밀성

- **대응**:

  - 세션 내 메시지는 **AEAD(예: ChaCha20-Poly1305)** 로 암호화
  - HTTP 레벨에서 **RFC 9421** 서명(`content-digest`, `@method`, `@path`, `@authority` 등)으로 **헤더/본문 커버리지** 확보

## End-to-End 보장 방식

1. **핸드셰이크 단계**: HPKE(Base/추가 PFS)로 **공유 seed**를 안전히 합의하고, `ackTag` 로 **상호 키 확인**.
2. **세션 단계**: seed→HKDF→세션키. 모든 애플리케이션 페이로드는 **세션 레벨 AEAD** 로 암호화.
3. **HTTP 서명 단계**: RFC 9421로 요청 라인/헤더/본문 요약(**Content-Digest**)에 **서명**, `keyId=kid` 로 세션 조회·검증.
4. **운영 방어선**: Nonce 재사용 거부, 세션 만료, 키 안전폐기, DID 롤오버 정책 → **E2E 기밀성/무결성/재연불가능성** 달성.

## 메시지 스키마

### Init (Client → Server)

```json
{
  "initDid": "did:sage:...:client",
  "respDid": "did:sage:...:server",
  "info": "sage/hpke v1|ctx=CTX|init=...|resp=...",
  "exportCtx": "exporter:CTX",
  "enc": "<base64url 32B>",
  "nonce": "n-...",
  "ts": "RFC3339Nano",
  "ephC": "<base64url 32B>" // PFS add-on 일 때만
}
```

메타: `did`, `signature(Ed25519)`.

### Ack (Server → Client)

```json
{
  "kid": "kid-uuid",
  "ackTagB64": "<base64url HMAC(ackKey, 'hpke-ack|ctxID|nonce|kid')>",
  "ephS": "<base64url 32B>", // PFS add-on
  "ts": "RFC3339Nano"
}
```

메타: `did`, `signature(Ed25519)`.

## 구성/튜닝 포인트

- **ServerOpts.MaxSkew**: `ts` 수용 범위(기본 2분)
- **NonceStore TTL**: Init 재생 방지 창
- **session.Config**: `MaxAge`/`IdleTimeout`/`MaxMessages`
- **InfoBuilder**: 서비스 특화 컨텍스트 추가 가능(예: 테넌트·스코프)

## 에러 가이드

- `missing did` : 메타에 DID 없음 → 메타 서명 생성/전달 확인
- `signature verification failed` : DID Document 키 불일치/손상
- `ts out of window` : 시계/지연 문제 → NTP/MaxSkew 조정
- `replay detected` / `replay` : Nonce 재사용
- `info/exportCtx mismatch` : 서로 다른 InfoBuilder/ctxID
- `ack tag mismatch` : 양끝 seed 상이(HPKE 파라미터·eph 키 확인)
- `no session` : kid 미바인딩 또는 만료
- `sig verify failed`(9421) : Header 커버리지/서명 파라미터 오류

## 운영 권고(HPKE 운용 시)

- **Base만 사용하는 경우**

  - **KEM 키 회전 주기 단축**(짧은 TTL) 및 **HSM 격리**
  - Encapsulation(`enc`)을 상위 전송 채널에서 **은닉**(예: 추가 암호화 프레이밍)

- **강한 PFS가 필요한 경우(권장)**

  - **PFS add-on** 활성화(ephC/ephS + HKDF combiner)
  - ephemeral **개인키 즉시 폐기**와 **안전 메모리 처리**

## 예제(요약)

```go
// Client
enc, exporter := hpke.DeriveToPeer(serverKEMPub, info, exportCtx)
nonce := uuid.NewString()
payload := {enc, info, exportCtx, nonce, ts, ephC?}
resp := Send(TaskHPKEComplete, payload, metaSig(DID))

kid, ackTagB64, ephS? := resp.Meta...
seed := exporter                   // base
seed := Combine(exporter, ssE2E)   // base + PFS
checkAck(seed, ctxID, nonce, kid)
Bind(kid, session(seed))

// Server
exporter := hpke.OpenWithPriv(skR, enc, info, exportCtx)
if hasEphC { ephS, ssE2E := ECDH() }
seed := exporter or Combine(...)
kid := IssueKeyID(ctxID)
ackTag := HMAC(HKDF(seed,"ack-key"), "hpke-ack|ctx|nonce|kid")
return {kid, ackTag, ephS?}
```

## 로드맵

- DID 레이어 **키 롤오버 UX/정책** 개선 및 온체인 갱신 지연/비용 최적화
- **KEM 키 자동 회전**과 수명 단축을 통한 Base 운용 보안성 향상
