# Secure Session Communication

SAGE 프로토콜은 종단간 보안을 제공하기 위해 기존 [에이전트 간 통신(Agent-to-Agent, A2A) 프로토콜](https://a2a-protocol.org/latest/topics/what-is-a2a/#a2a-request-lifecycle)에 [RFC 9180 HPKE(Hybrid Public Key Encryption)](https://www.rfc-editor.org/rfc/rfc9180) 기반의 핸드셰이크 단계를 추가하여 보호된 세션을 생성합니다. 세션이 성립되면 이후 모든 요청/응답은 대칭키 기반 AEAD 암호화로 보호되며, 헤더·메서드·경로 등의 메타데이터는 RFC 9421(HTTP Message Signatures) 스타일의 HMAC으로 무결성과 리플레이 방지를 보장합니다.

<img src="../assets/SAGE-E2EE-request-lifecycle.png" width="500" height="650"/>

## DID를 이용한 신원보장

에이전트의 신원 공개키는 DID Registry에 앵커되어 DID Document(verificationMethod)로 공개됩니다. 핸드셰이크에서 각 에이전트는 세션에서 사용할 임시(X25519) 공개키와 컨텍스트를 자신의 DID 신원 키(Ed25519)로 서명해 교환하고, 상대는 DID Document에서 얻은 공개키로 서명을 검증합니다.
이 과정으로 “누구와 통신하는지”가 암호학적으로 확정되며, 임시 키가 신원 키에 바인딩되기 때문에 중간자(MitM)가 임시 키를 바꿔치기 하면 서명이 깨져 차단됩니다.

검증된 신원의 에이전트끼리는 임시 키 교환으로 생성된 공유 비밀(shared secret)로부터 단기 세션 키로 파생합니다. 이 세션 키는 세션 동안 AEAD 암·복호화와 HMAC 서명에 사용되며, 결과적으로 세션 범위의 모든 통신이 신원이 보장된 상대와만 안전하게 이루어집니다.

## HPKE-based Handshake (2-Step)

SAGE의 HPKE 핸드셰이크는 **HPKE Base 모드(DHKEM(X25519), KDF=HKDF-SHA256, AEAD=ChaCha20-Poly1305)** 를 기반으로, **추가 E2E(ephemeral-ephemeral) X25519 교환**을 사용하여 **PFS를 보장**합니다.

단, 세션 당시의 임시 개인키(ephCpriv/ephSpriv)가 적시에 안전하게 폐기되며, 재사용/로그/유출이 없었다는 표준 가정하에서 성립합니다.

**용어**

> `enc`: HPKE Base에서 송신자가 만든 **HPKE 임시 공개키**(평문 전송 가능, 공개값)  
> `ephC`: 클라이언트 E2E 임시 X25519 **공개키**(평문 전송, 공개값)  
> `ephS`: 서버 E2E 임시 X25519 **공개키**(평문 전송, 공개값)  
> `exporterHPKE`: HPKE Base에서 양측이 얻는 **exporter secret**  
> `ssE2E`: E2E X25519 교환으로 얻는 **ephemeral-ephemeral DH 결과**  
> `combined`: `exporterHPKE || ssE2E` 를 HKDF로 결합한 **최종 세션 시드**  
> `kid`: 세션 식별자 (RFC 9421 `keyid`로 사용, 랜덤 문자열)  
> `ackTag`: 서버가 `combined`와 트랜스크립트를 HMAC해 만든 **키 확인 태그**

### 1) Client → Server : HPKE Initialize

클라이언트가 다음을 **A2A Message.Content**로 보냅니다(메타데이터에는 Ed25519 서명):

- `info`, `exportCtx`: 세션·컨텍스트/유도정보(서명·검증 대상에 포함되어 트랜스크립트에 고정)
- `enc`: HPKE Base 송신자 임시 공개키(공개값)
- `ephC`: 클라이언트 E2E 임시 X25519 공개키(공개값)
- `initDid`, `respDid`, `nonce`, `ts` 등

서버는 다음을 수행합니다.

1. **메타데이터 서명 검증(Ed25519, DID)**  
   발신자의 DID Document로 메시지(메타데이터 제외)를 검증 → **신원 바인딩**.
2. **윈도·재전송 검사**  
   `ts`가 허용 시간창(`±MaxSkew`) 안인지 확인, `(ctxID|nonce)` 캐시로 **Replay 차단**.
3. **HPKE exporter 재현**  
   서버의 **KEM 정적 개인키**와 클라이언트의 `enc`로 **`exporterHPKE`** 복원.
4. **E2E 임시 키 교환**  
   서버가 `ephS` 생성 → `ssE2E = X25519(ephSpriv, ephCpub)` 계산.
5. **최종 세션 시드**  
   `combined = HKDF(exporterHPKE || ssE2E ; salt=exportCtx)` 로 32바이트 시드 도출.
6. **세션 생성 및 kid 바인딩**  
   `combined`로 세션 키를 파생하고, 서버 쪽 세션을 만들고 **kid ↔ session** 바인딩.
7. **키 확인 태그(ackTag)**  
   `combined`와 트랜스크립트(`ctxID, nonce, kid, info, exportCtx, enc, ephC, ephS, initDid, respDid`)로 **HMAC** → `ackTag` 생성.

서버는 다음을 **A2A Message.Content**로 회신하며, **서버 DID 서명**을 **Message.Metadata**에 포함합니다.

- `kid`, `ephS`, `ackTagB64`, `ts`

### 2) Server → Client : Signed Response

클라이언트는 서버 응답에서:

1. **서버 DID 서명 검증** (메타데이터 제외 본문에 대한 Ed25519 검증)
2. **E2E DH 계산**  
   자신의 `ephCpriv`와 서버의 `ephS`로 **`ssE2E`** 계산.
3. **최종 시드·세션 합치**  
   자신이 만든 `exporterHPKE` 와 `ssE2E` 를 같은 방식으로 결합해 **`combined`** 재현
4. **`ackTag` 검증**  
   서버가 보낸 `ackTag` 와 자신의 계산값이 상수시간으로 일치하는지 확인 → **키 확인**
5. **세션 생성 및 kid 바인딩**  
   동일한 `combined`로 세션을 만들고 **kid ↔ session** 을 바인딩

> `enc`, `ephC`, `ephS` 를 **암호화 없이 평문**으로 보내도 안전한가?
>
> - 세 값은 **공개키**로, **기밀 데이터가 아니므로**, 공격자가 알아도 **비밀을 계산하려면 임시 개인키**가 필요합니다.
> - HPKE Base만 사용하면 수신자 KEM 정적 키 유출 시 과거 `exporterHPKE` 를 재현할 수 있지만, 본 설계는 **추가 E2E DH(ssE2E)** 를 결합하므로 **임시 개인키(ephCpriv 또는 ephSpriv)가 보존되지 않은 한** 과거 세션의 `combined` 를 재현할 수 없습니다. → **PFS 제공**.

## 보안 세션 통신

핸드셰이크가 끝나면 양측은 같은 `combined` 로부터 **AEAD 키**를 파생하여 **본문을 암호화(ChaCha20-Poly1305)** 합니다. 동시에 **RFC 9421** 으로 다음 컴포넌트를 서명합니다.

- `@method`, `@path`, `@authority(host)`, `date`, `content-digest`
- 헤더에는 `Signature-Input: sig1=(...);keyid="kid";nonce="...";created=...`
- **서명 알고리즘은 Ed25519**(세션 kid는 검증 시 세션/리플레이 캐시에 활용)

**추가 보안 사항**

- **Replay 방지**: `(kid, nonce)` 캐시로 **1회성 보장**.
- **세션 수명 정책**: `MaxAge`, `IdleTimeout`, `MaxMessages` 로 만료/폐기.
- **서버→클라이언트 응답 암호화**: 동일 세션키로 암호화한 `cipher_b64` 반환.

## 암호학적 성질 & 위협 대응

### 전방향 비밀성(PFS)

- **HPKE Base 단독**: 수신자 **정적 KEM 개인키**가 사후 유출되면 과거 캡처된 `(enc, ciphertext)` 로 `exporterHPKE` 재현 가능 → **과거 복호화 위험**.
- **본 설계(HPKE Base + E2E DH 결합)**:
  `combined = HKDF(exporterHPKE || ssE2E)` 이며, **ssE2E 계산에는 임시 개인키(ephCpriv 또는 ephSpriv)가 필요**.
  임시 개인키를 **메모리에서 즉시 폐기**하면, **수신자 KEM 키가 유출되어도 과거 `combined` 재현 불가** → **PFS 복원**.

### MitM(중간자) 공격

- **메타데이터 서명(DID/Ed25519)** 으로 **HPKE 파라미터(`info/exportCtx/enc/ephC`)** 가 발신자 신원에 바인딩됩니다.
- 서버 응답도 **서버 DID 서명**과 **ackTag(키 확인)** 를 포함 → 트랜스크립트 조작 불가.

### Replay 공격

- 핸드셰이크: `(ctxID|nonce)` 캐시로 **초기화 요청 재사용 차단**.
- 세션: `(kid, nonce)` 캐시로 **메시지 1회성 보장**.

### ECDH all-zero 방지

- X25519 **공유비밀이 all-zero** 인 경우 **즉시 오류로 거부**(RFC 7748 관례).
  (라이브러리 혹은 유효성 검사에서 강제)

### Nonce/엔트로피

- AEAD nonce 및 임시키는 **CSPRNG** 로 생성, 오류 시 **fail-closed**.

### DID 문서 최신성

- Resolver에 **TTL/리프레시/해지** 정책 적용, **체인 앵커 검증** 및 **캐시 무결성** 보장.

## 운영 권고

- **임시 개인키(ephCpriv/ephSpriv)** 는 사용 직후 **즉시 영(0)화 및 폐기**
- **서버 KEM 정적 키**는 **HSM/TPM** 보관, 가능하면 **주기적 회전**
- **로그**에는 DID·컨텍스트·키 지문만 남기고 **평문·세션키**는 기록 금지

## 프로토콜 요약(필드)

**Client → Server (Init / A2A Message)**

- Content: `{ info, exportCtx, enc, ephC, initDid, respDid, nonce, ts }`
- Metadata: `{ did, signature }` // 메시지(메타데이터 제외)에 대한 Ed25519 서명

**Server → Client (Ack / A2A Message)**

- Content: `{ kid, ephS, ackTagB64, ts }`
- Metadata: `{ did, signature }` // 메시지(메타데이터 제외)에 대한 Ed25519 서명

**Protected API (세션 통신)**

- Body: `AEAD(ciphertext)` (ChaCha20-Poly1305)
- Headers: `Signature-Input(sig1; keyid="kid"; nonce="..."; created=...)` 등
  (Ed25519로 서명, 서버는 DID Resolver로 공개키 확인)

---

## Cryptography

- **Identity / Signatures**: Ed25519 (DID verificationMethod)
- **HPKE Base**: DHKEM(X25519) + HKDF-SHA256 + ChaCha20-Poly1305
- **E2E Ephemeral**: X25519 (ephemeral-ephemeral)
- **Key Derivation**: HKDF-SHA256
- **Session AEAD**: ChaCha20-Poly1305
- **HTTP Message Signatures**: Ed25519 (RFC 9421 구성 요소 서명)
- **Replay Guards**: `(ctxID|nonce)` for handshake, `(kid, nonce)` for session
