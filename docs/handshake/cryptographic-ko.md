# Cryptographic

본 문서는 SAGE 프로토콜에서 DID 신원키 → 임시 키 교환 → 세션 키 파생 → AEAD/HMAC 보호 -> RFC 9421 통신으로 이어지는 흐름의 암호학적 안전성을 설명합니다.

## DID 신원키(Ed25519)를 이용한 임시(X25519) 키 교환 보호

핸드쉐이크 단계에서 임시키 교환을 보호하기 위해 상대 에이전트의 DID Document(verificationMethod)에서 Ed25519 공개키를 획득하여 암호화 하여 전송합니다.

핸드쉐이크에서 교환되는 임시 X25519 공개키 페이로드는:

- **서명**: 송신자가 Ed25519 신원키로 서명하여 출처/무결성을 보장

- **부트스트랩 암호화**: 수신자의 Ed25519 공개키를 X25519로 변환해 Ephemeral-Static ECDH → HKDF → AEAD(AES-GCM) 로 암호화

**보안 성질**

- **기밀성**: 부트스트랩 암호화로 임시키/핸드셰이크 내용을 제3자가 볼 수 없습니다(AES-GCM).
- **무결성/출처**: Ed25519 서명 검증 및 AEAD 인증태그로 변조·바꿔치기를 차단합니다.
- **신원 바인딩**: DID 문서의 공개키로 서명을 검증하므로, “누구와 통신하는지”가 고정됩니다.

### 가정(위협 모델과 전제)

- **유효한 DID → 올바른 공개키**: 상대 DID Document가 최신이며 변조되지 않았음을 전제(블록체인 앵커·검증 절차로 확보)
- **정상 난수원**: 임시키 생성, AEAD nonce 생성에 충분한 엔트로피가 제공

### 유의사항 & 대응

### 1. 중간자 공격 (MitM)

- **위험/고려**  
  공격자가 핸드셰이크 중 임시 공개키를 바꿔치기하면, 이후 파생되는 세션키 전체가 공격자 기준으로 고정될 수 있음
- **대응**
  - **신원 서명 검증**: 상위 A2A 레이어에서 메시지 전체를 Ed25519로 서명/검증(메타데이터의 DID와 서명)하여 발신자 신원 위조를 차단
  - **부트스트랩 암호화 무결성**: `EncryptWithEd25519Peer`가 AES-GCM을 사용하고, `transcript := appendPrefix(pubKey.Bytes(), peerX)`를 AAD(추가 인증 데이터)로 넣어 송신자 ephPub ↔ 수신자 변환키(peerX) 바인딩을 보장. 임시키 바꿔치기 시 GCM 태그가 즉시 파괴
- **유의 사항**

  - A2A 서명·검증 오류 시 사유(서명 불일치/미등록 DID/만료 등) 최소 노출
  - 핸드셰이크 로그에 DID·컨텍스트·eph 키 지문만 남기고 원문은 저장하지 않기

### 2. ECDH all-zero (RFC 7748 권고)

- **위험/고려**  
  비정상 공개키(저차점/identity 등) 입력 시 X25519 ECDH 결과가 올-제로(0x00…00) 가 될 수 있고, 이를 키 유도에 쓰면 취약
- **대응**

  - privKey.ECDH(peerPubKey) 결과를 반드시 sharedSecret(dh, err)로 넘겨 (a) 길이 32바이트 확인, (b) 상수시간 비교로 올-제로 거부를 수행해 즉시 에러로 차단

  - RFC 7748의 “all-zero shared secret MUST be rejected” 관례를 구현 수준에서 충족

- **유의 사항**
  - 거부 이벤트를 감사 로그에 남기고(지문만), 반복 발생 시 피어 차단 정책 연계

### 3. 부트스트랩 암호화 nonce/엔트로피

- **위험/고려**  
  AES-GCM nonce 재사용은 치명적이며, 난수 품질 저하도 위험
- **대응**

  - `nonce := make(...); io.ReadFull(rand.Reader, nonce)`로 매번 CSPRNG 기반 난수 nonce 생성

  - 핸드셰이크에서 ephemeral 키 매 세션 재생성 → 동일 키/nonce 중복 위험 추가 완화

- **유의 사항**
  - 초고속 트래픽 환경에서는 카운터 기반 nonce(세션당 카운터) 검토 가능
  - 난수 고갈/오류 감지 시 페일-클로즈(세션 중단) 정책 유지

### 4. DID 공개키 최신성/무결성

- **위험/고려**  
  회전/폐기된 키를 참조하거나 오래된 DID Document를 사용하면 검증 실패·혼동 가능
- **대응**  
   상위 레이어에서 DID Document 검증을 전제로 동작(네 문서/흐름에 명시). DID를 통해 신원키를 조회하고 A2A 서명 검증·부트스트랩 암호화에 활용
- **유의 사항**  
   DID 리졸버에 TTL/리프레시/해지(Revocation) 확인 정책을 적용하고, 체인 앵커 검증·캐시 무결성 보호(예: 서명된 캐시) 추가

## X25519 공유 비밀(shared secret) → HKDF-SHA256 → 세션 키 파생 (AEAD/HMAC)

핸드쉐이크의 키 교환에서 얻은 공유 비밀로 부터 세션내에서 사용할 암호화 키와 서명키를 생성합니다.

1. 공유 비밀로 부터 seed 생성

핸드쉐이크에서 얻은 공유 비밀을 입력으로 하여

```go
 hkdf.Extract(sha256.New, ikm, salt)
```

라벨/컨텍스트/임시키(정렬)로 만든 salt를 입력으로하여 PRK(sessionSeed) 를 얻습니다.

2. session seed로 부터 암호화/서명 키를 생성

```go
hkdfEnc := hkdf.New(sha256.New, s.sessionSeed, salt, []byte("encryption"))
s.encryptKey = make([]byte, 32)

hkdfSign := hkdf.New(sha256.New, s.sessionSeed, salt, []byte("signing"))
s.signingKey = make([]byte, 32)
```

salt=세션ID, info="encryption"/"signing" 로 각 키를 생성합니다.

**HKDF-Expand**:

- 암호화 키(32B) → ChaCha20-Poly1305 AEAD

  ```go
      // Initialize AEAD cipher
          aead, err := chacha20poly1305.New(sess.encryptKey)
          if err != nil {
              return nil, fmt.Errorf("failed to create AEAD: %w", err)
          }
          sess.aead = aead

          ...

          ciphertext := sess.aead.Seal(nil, nonce, plaintext, nil)
  ```

- 서명 키(32B) → HMAC-SHA256 (RFC 9421 스타일의 covered components 서명)
  ```go
  func (s *SecureSession) SignCovered(covered []byte) []byte {
      m := hmac.New(sha256.New, s.signingKey)
      m.Write(covered)
      s.UpdateLastUsed()
      return m.Sum(nil)
  }
  ```

### 유의사항 & 대응

### 1. 컨텍스트 바인딩 실패 (Context Binding Failure)

- **위험/고려**  
  동일한 공유 비밀이 다른 프로토콜/컨텍스트에서 재사용되는 경우 키/세션 충돌이나 교차-프로토콜 공격 위험 존재 (예: ctxID가 같은데 라벨이 다른 두 프로토콜이 같은 seed를 만들거나, 반대로 ctxID가 달라도 salt/label 설계가 부실해 충돌)
- **대응**

  - `DeriveSessionSeed`에서 다음과 같이 라벨/컨텍스트/임시키 정렬을 묶어 PRK(sessionSeed)를 추출.

  ```go
    h := sha256.New()
    h.Write([]byte(label))
    h.Write([]byte(p.ContextID))
    h.Write(lo)
    h.Write(hi)
    salt := h.Sum(nil)

    seed := hkdfExtractSHA256(sharedSecret, salt)
  ```

  또한 `ComputeSessionIDFromSeed(label || seed)`로 프로토콜 라벨을 추가하여 세션ID가 충돌하지 않도록 함.

- **유의 사항**
  - 라벨 값에 프로토콜 버전을 반드시 포함("a2a/handshake v1")해 버전/프로토콜 간 도메인 분리를 강화.
  - 추후 기능이 늘어나는 경우 info(HKDF Expand의 info) 문자열로 purpose label을 세분화(예: "encryption", "signing", "ack-key", "traffic-key/c2s", "traffic-key/s2c" 등).

### 2. 키 분리 미흡 (Key Separation)

- **위험/고려**  
  암호화 키와 서명 키를 동일 OKM에서 구분 없이 뽑는 경우 키 재사용에 따른 보안 경계가 약화
- **대응**  
  HKDF-Expand 시 info를 "encryption" / "signing"으로 구분해 도메인 분리

### 3. AEAD Nonce 재사용 (Nonce Misuse)

- **위험/고려**  
  ChaCha20-Poly1305는 96비트 nonce 재사용 시 보안 파괴. 랜덤 nonce는 충돌 확률이 낮지만, 고속/장수 세션에서 충돌 확률 누적 문제
- **대응**  
  `nonce := rand(12B)`로 CSPRNG 기반 랜덤 nonce 생성.

### 4. 세션 수명·사용량 초과 (Key Lifetime / Usage Limits)

- **위험/고려**  
  하나의 세션 키로 과도한 메시지/장시간 사용 시 누적 위험(통계적 공격 표면 증가)
- **대응**

  - `Config{ MaxAge, IdleTimeout, MaxMessages }`로 절대 수명/유휴 시간/메시지 개수 제한. 만료 시 Close()에서 키/seed 지우기 수행

  - 핸드셰이크에서 ephemeral 키 매 세션 재생성 → 동일 키/nonce 중복 위험 추가 완화

- **유의 사항**
  - 운영 환경에 맞춰 짧은 MaxMessages/MaxAge 설정(예: 10^5 메시지 이하, 수십 분~수 시간 단위)
  - 만료 임박/도달 시 ReKey(세션 재협상) 절차 도입

## 세션 계층

### 키 전제 (HKDF로 파생된 세션 키)

- 세션 키는 `sessionSeed = HKDF‐Extract(SHA-256, shared_secret, salt)` 에서 나옵니다. shared_secret(핸드셰이크 결과)와 세션 바인딩 salt(컨텍스트/라벨/임시키 정렬 등)를 쓰므로, 세션 간 키 분리(key separation) 가 성립합니다.

- 이후 HKDF‐Expand(…, “encryption”) / HKDF‐Expand(…, “signing”) 로 서로 다른 목적의 키를 파생합니다.  
  → 암호화 키와 서명 키는 독립이며, 한쪽이 노출되어도 다른 쪽의 보안성에 영향을 주지 않는 도메인 분리가 성립합니다.

### 본문 보호: ChaCha20-Poly1305 (Encrypt / Decrypt)

- **기밀성(Confidentiality)**: 256-bit 키, 96-bit 난수(Nonce), 스트림 + Poly1305 인증으로 IND-CPA 수준의 기밀성 보장.

- **무결성/인증(Integrity/Authentication)**: Poly1305 태그(128-bit)로 INT-CTXT 보장. 암/복호화 시 태그가 검증되며, 위조·변조는 실패합니다.

- **랜덤 Nonce**: 매 암호화마다 CSPRNG로 96-bit nonce 생성 → nonce 중복 확률이 매우 낮음.
  단, AEAD은 nonce 재사용에 취약하므로:

  - 동일 키로 절대 nonce 재사용 금지 (재사용 시 기밀성·무결성 모두 붕괴 가능).
  - 고속 시나리오에서는 “세션별 카운터 기반 nonce”도 고려 가능(중복 확률 0).

- 방향 분리(옵션): EncryptOutbound/DecryptInbound를 쓰면 클라→서버, 서버→클라 키를 분리 운용합니다.  
  → 한 방향 키가 노출돼도 반대 방향에는 영향이 없고, nonce 공간 충돌 가능성도 추가로 줄어듭니다.

### 메타 데이터 무결성: HMAC-SHA256 (SignCovered / VerifyCovered)

- **EUF-CMA 보안**: HMAC는 PRF로 모델링되며, SHA-256 기반의 HMAC는 알려진 실용적 공격이 없습니다.  
  → 선택-메시지 공격 하에서도 위조 불가능(키가 노출되지 않는 한).

- **covered 구성**: @method, @path, host, date, content-digest, @signature-params를 정규화된 순서/형식으로 직렬화해 서명합니다.  
  → 헤더·메서드·경로·시간·본문 다이제스트가 모두 HMAC에 바인딩되어, 중간자에 의한 헤더만 바꿔치기, 경로 다운그레이드, 날짜 조작 등이 차단됩니다.

- **Content-Digest 바인딩**: content-digest는 전송되는 바디 그대로(여기서는 cipher)를 해시해 헤더로 올리고, HMAC가 그 헤더를 다시 덮습니다.  
  → 헤더 ↔ 바디의 상호 바인딩이 생겨, 바디/헤더 중 하나만 조작해도 서명이 깨집니다.

- **검증 구현**: hmac.Equal을 사용한 상수시간 비교로 타이밍 기반 서명 유추 공격 방지.

### 실패 시 동작(보안 정책)

- 필수 헤더 누락/형식 오류 → 400
- 리플레이 감지((kid, nonce) 재사용) → 401
- HMAC 검증 실패 → 401
- AEAD 태그 실패/복호화 실패 → 401
- 세션 만료(MaxAge/IdleTimeout/MaxMessages) → 401 또는 정책 코드
- 로깅은 kid/ctxID/지문 수준으로만. 평문, 키, seed는 미기록(비노출).

### 키 수명·소거

- 세션마다 새 키: 핸드셰이크(임시키 교환)로 매 세션 새 sessionSeed → 새 암호화/서명 키.
  → **세션 단위 전방향 비밀성(PFS at session granularity)**에 기여.
- 소거(Zeroization): 세션 종료 시 메모리 상 키·seed를 0으로 덮어 사후 포렌식 위험 최소화.

## 엔드-투-엔드(E2E) 보장

- **키 보유 주체는 종단만**: 세션 키는 핸드쉐이크 양단에서만 파생·보유. 프록시/게이트웨이는 복호화·위조 불가. 데이터 + 메타데이터 이중 보호

- **AEAD**: 본문 기밀성 + 무결성(16바이트 인증태그)

- **HMAC 서명**: @method, @path, host, date, content-digest 등 covered components 무결성/재전송 방지

- **Replay 방지**: kid + nonce 캐시, Date 신선도, 세션 정책(IdleTimeout/MaxMessages)로 재사용 차단

## AEAD 인증태그, HMAC 서명 — 역할 분리

- **AEAD 인증태그(ChaCha20-Poly1305/GCM)**  
  복호 시 자동 검증되며, 본문 위변조 시 즉시 실패. 메시지 단위 무결성/진위를 보장합니다.

- **HMAC-SHA256 (RFC 9421 스타일)**  
  헤더/메서드/경로/날짜/콘텐츠 다이제스트 등 메타데이터 무결성을 보장하고 재전송 방지(nonce, Date)까지 담당합니다.

이러한 이중 보호는 프록시 개입, 본문 암호화 후 헤더 조작 같은 위협을 방어하는 데 유리합니다.
