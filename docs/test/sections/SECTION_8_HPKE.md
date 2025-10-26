## 8. HPKE

### 8.1 암호화/복호화

#### 8.1.1 DHKEM

##### 8.1.1.1 X25519 키 교환 성공

**시험항목**: X25519 기반 DHKEM 키 교환

**Go 테스트**:

```bash
go test -v github.com/sage-x-project/sage/pkg/agent/hpke -run 'Test_HPKE_Base_Exporter_To_Session'
```

**예상 결과**:

```
=== RUN   Test_HPKE_Base_Exporter_To_Session
[PASS] X25519 키 쌍 생성 성공 (Receiver: Bob)
[PASS] HPKE 키 파생 성공 (Sender: Alice)
  Encapsulated key: 32 bytes (예상값: 32)
[PASS] HPKE 키 개봉 성공 (Receiver: Bob)
```

**검증 방법**:

- X25519 키 쌍 생성 (Receiver)
- HPKE 키 파생 (Sender) - Encapsulated key 생성
- HPKE 키 개봉 (Receiver) - Encapsulated key로부터 복원
- Encapsulated key 크기 = 32 bytes 확인

**통과 기준**:

- ✅ X25519 키 생성 성공
- ✅ Encapsulated key = 32 bytes
- ✅ HPKE 키 파생 성공
- ✅ HPKE 키 개봉 성공

**실제 테스트 결과** (2025-10-24):

```
=== RUN   Test_HPKE_Base_Exporter_To_Session
[PASS] X25519 키 쌍 생성 성공 (Receiver: Bob)
  HPKE info context: sage/hpke-handshake v1|ctx:ctx-001|init:did:alice|resp:did:bob
  Export context: sage/session exporter v1
[PASS] HPKE 키 파생 성공 (Sender: Alice)
  Encapsulated key: 32 bytes (예상값: 32)
  Exporter secret: 32 bytes (예상값: 32)
[PASS] HPKE 키 개봉 성공 (Receiver: Bob)
  Shared secret 일치: true
```

**검증 데이터**:
- 테스트 파일: `pkg/agent/hpke/hpke_test.go:33-181`
- 테스트 데이터 파일: `testdata/verification/hpke/hpke_key_exchange_session.json`
- 상태: ✅ PASS
- SAGE 함수 사용:
  - ✅ `keys.GenerateX25519KeyPair()` - X25519 KEM 키 쌍 생성
  - ✅ `keys.HPKEDeriveSharedSecretToPeer()` - HPKE Sender 키 파생
  - ✅ `keys.HPKEOpenSharedSecretWithPriv()` - HPKE Receiver 키 개봉
- Encapsulated key: 32 bytes (X25519 공개키)
- Exporter secret: 32 bytes
- 모든 암호화 기능은 SAGE 내부 구현 사용

---

---

##### 8.1.1.2 공유 비밀 생성 확인

**시험항목**: HPKE 공유 비밀 생성 및 일치 확인

**Go 테스트**:

```bash
go test -v github.com/sage-x-project/sage/pkg/agent/hpke -run 'Test_HPKE_Base_Exporter_To_Session'
```

**예상 결과**:

```
[PASS] HPKE 키 개봉 성공 (Receiver: Bob)
  Shared secret 일치: true
[PASS] Session ID 결정적 파생
  Session ID (Alice): h5VqexSQWuM9qHMTDViJzw
  Session ID (Bob): h5VqexSQWuM9qHMTDViJzw
  Session ID 일치: true
```

**검증 방법**:

- Sender와 Receiver의 Shared secret 생성
- 양쪽 Shared secret 일치 확인 (`bytes.Equal(expA, expB)`)
- Session ID 결정적 파생 (`session.ComputeSessionIDFromSeed`)
- 양쪽 Session ID 일치 확인

**통과 기준**:

- ✅ Shared secret = 32 bytes
- ✅ Sender와 Receiver의 Shared secret 일치
- ✅ Session ID 결정적 파생 성공
- ✅ 양쪽 Session ID 일치

**실제 테스트 결과** (2025-10-24):

```
[PASS] HPKE 키 개봉 성공 (Receiver: Bob)
  Shared secret 일치: true
[PASS] Session ID 결정적 파생
  Session ID (Alice): h5VqexSQWuM9qHMTDViJzw
  Session ID (Bob): h5VqexSQWuM9qHMTDViJzw
  Session ID 일치: true
```

**검증 데이터**:
- 테스트 파일: `pkg/agent/hpke/hpke_test.go:68-90`
- 상태: ✅ PASS
- SAGE 함수 사용:
  - ✅ `keys.HPKEOpenSharedSecretWithPriv()` - Shared secret 복원
  - ✅ `session.ComputeSessionIDFromSeed()` - 결정적 Session ID 파생
- Shared secret: 32 bytes (일치 확인)
- Session ID: Base64 인코딩 (양쪽 동일)
- 검증: `bytes.Equal(expA, expB)` 및 `sidA == sidB`

---

---

#### 8.1.2 AEAD

##### 8.1.2.1 ChaCha20Poly1305 암호화 성공

**시험항목**: HPKE exporter로부터 파생된 세션 키로 AEAD 암호화

**Go 테스트**:

```bash
go test -v github.com/sage-x-project/sage/pkg/agent/hpke -run 'Test_HPKE_Base_Exporter_To_Session'
```

**예상 결과**:

```
[PASS] HPKE exporter로부터 보안 세션 설정 완료
  Alice 세션 생성, ID: h5VqexSQWuM9qHMTDViJzw
  Bob 세션 생성, ID: h5VqexSQWuM9qHMTDViJzw
[PASS] 메시지 암호화 성공 (Alice)
  테스트 메시지: hello, secure world
  암호문 크기: 47 bytes
```

**검증 방법**:

- HPKE exporter로부터 보안 세션 생성 (`session.NewSecureSessionFromExporter`)
- 테스트 메시지 준비: "hello, secure world"
- Alice 세션으로 메시지 암호화 (`sA.Encrypt(msg)`)
- 암호문 크기 확인 (평문 + AEAD 오버헤드)

**통과 기준**:

- ✅ 보안 세션 생성 성공
- ✅ AEAD 암호화 성공
- ✅ 암호문 생성 확인 (크기 > 평문 크기)

**실제 테스트 결과** (2025-10-24):

```
[PASS] HPKE exporter로부터 보안 세션 설정 완료
  Alice 세션 생성, ID: h5VqexSQWuM9qHMTDViJzw
  Bob 세션 생성, ID: h5VqexSQWuM9qHMTDViJzw
[PASS] 메시지 암호화 성공 (Alice)
  테스트 메시지: hello, secure world
  메시지 크기: 19 bytes
  암호문 크기: 47 bytes
```

**검증 데이터**:
- 테스트 파일: `pkg/agent/hpke/hpke_test.go:93-111`
- 상태: ✅ PASS
- SAGE 함수 사용:
  - ✅ `session.NewSecureSessionFromExporter()` - HPKE exporter로부터 AEAD 세션 생성
  - ✅ `session.Encrypt()` - ChaCha20Poly1305 AEAD 암호화
- 평문: "hello, secure world" (19 bytes)
- 암호문: 47 bytes (19 bytes 평문 + AEAD 오버헤드)
- 알고리즘: ChaCha20Poly1305 (HPKE 기본 AEAD)

---

---

##### 8.1.2.2 복호화 후 평문과 일치

**시험항목**: AEAD 복호화 및 평문 일치 검증

**Go 테스트**:

```bash
go test -v github.com/sage-x-project/sage/pkg/agent/hpke -run 'Test_HPKE_Base_Exporter_To_Session'
```

**예상 결과**:

```
[PASS] 메시지 복호화 성공 (Bob)
  복호화된 메시지: hello, secure world
  평문 일치: true
```

**검증 방법**:

- Bob 세션으로 암호문 복호화 (`sB.Decrypt(ct)`)
- 복호화된 평문과 원본 메시지 비교 (`bytes.Equal(pt, msg)`)
- 평문 일치 확인

**통과 기준**:

- ✅ AEAD 복호화 성공
- ✅ 복호화된 평문이 원본 메시지와 정확히 일치
- ✅ AEAD 인증 성공 (무결성 검증)

**실제 테스트 결과** (2025-10-24):

```
[PASS] 메시지 복호화 성공 (Bob)
  복호화된 메시지: hello, secure world
  평문 일치: true
```

**검증 데이터**:
- 테스트 파일: `pkg/agent/hpke/hpke_test.go:113-118`
- 상태: ✅ PASS
- SAGE 함수 사용:
  - ✅ `session.Decrypt()` - ChaCha20Poly1305 AEAD 복호화
- 복호화 결과: "hello, secure world" (원본과 일치)
- 검증: `bytes.Equal(pt, msg)` = true
- AEAD 인증: Poly1305 MAC 검증 성공

---

---

##### 8.1.2.3 암호문 처리 검증

**시험항목**: AEAD 암호문 크기 및 형식 검증

**Go 테스트**:

```bash
go test -v github.com/sage-x-project/sage/pkg/agent/hpke -run 'Test_HPKE_Base_Exporter_To_Session'
```

**예상 결과**:

```
[PASS] 메시지 암호화 성공 (Alice)
  암호문 크기: 47 bytes
```

**검증 방법**:

- 암호문 크기 확인 (평문 + AEAD 오버헤드)
- AEAD 오버헤드 = Nonce (12 bytes) + Poly1305 Tag (16 bytes)
- 암호문 형식: Nonce || Ciphertext || Tag

**통과 기준**:

- ✅ 암호문 크기 = 평문 크기 + AEAD 오버헤드
- ✅ 암호문이 유효한 AEAD 형식
- ✅ 복호화 가능

**실제 테스트 결과** (2025-10-24):

```
[PASS] 메시지 암호화 성공 (Alice)
  메시지 크기: 19 bytes
  암호문 크기: 47 bytes
```

**검증 데이터**:
- 테스트 파일: `pkg/agent/hpke/hpke_test.go:103-111`
- 상태: ✅ PASS
- SAGE 함수 사용:
  - ✅ `session.Encrypt()` - AEAD 암호화 및 형식화
- 평문: 19 bytes
- 암호문: 47 bytes
- AEAD 오버헤드: 28 bytes (Nonce 12 bytes + Poly1305 Tag 16 bytes)
- 형식: ChaCha20Poly1305 표준 AEAD 형식

---

---

#### 8.1.3 보안 검증

##### 8.1.3.1 서버 서명 및 Ack Tag

**시험항목**: HPKE 핸드셰이크 서버 서명 및 Ack Tag 검증

**Go 테스트**:

```bash
go test -v github.com/sage-x-project/sage/pkg/agent/hpke -run 'Test_ServerSignature_And_AckTag_HappyPath'
```

**예상 결과**:

```
--- PASS: Test_ServerSignature_And_AckTag_HappyPath (0.02s)
    hpke_test.go:XX: Server signature verified
    hpke_test.go:XX: Ack tag validated
```

**검증 방법**:

- HPKE 핸드셰이크 완료
- 서버 Ed25519 서명 검증 성공
- Ack Tag 검증 성공

**통과 기준**:

- ✅ 핸드셰이크 성공
- ✅ Ed25519 서명 검증
- ✅ Ack Tag 유효

**실제 테스트 결과** (2025-10-23):

```
=== RUN   Test_ServerSignature_And_AckTag_HappyPath
[PASS] HPKE 핸드셰이크 초기화 성공
  Client DID: did:sage:test:client-e53b90fe-51ef-4497-ba90-4f99e5734b4f
  Server DID: did:sage:test:server-e759b2d0-1eb1-447c-a1f4-b6b2b8df5283
  Context ID: ctx-d110dba4-7981-4075-91b3-0d47a72bddc0
[PASS] 서버 메시지 처리 성공
  Session ID: kid-31382254-3cb1-468d-9dea-1322966ed925
[PASS] Ed25519 서명 검증 성공
  서명 발견: Obx8QSXwuMLeQy6k05cz...
  서명 길이: 64 bytes (예상값: 64)
[PASS] Ack Tag 검증 성공
  Ack Tag: ZrfIAfV56Vzw_NmLSXWg...
[PASS] 세션 생성 완료
--- PASS: Test_ServerSignature_And_AckTag_HappyPath (0.00s)
```

**검증 데이터**:
- 테스트 파일: `pkg/agent/hpke/security_test.go`
- 상태: ✅ PASS
- SAGE 함수 사용:
  - ✅ `keys.GenerateEd25519KeyPair()` - Ed25519 키 쌍 생성
  - ✅ `keys.GenerateX25519KeyPair()` - X25519 KEM 키 쌍 생성
  - ✅ `Client.Initialize()` - HPKE 클라이언트 핸드셰이크 초기화
  - ✅ `Server.HandleMessage()` - HPKE 서버 메시지 처리
- Ed25519 서명 길이: 64 bytes (verified)
- Ack Tag: Base64 인코딩된 키 확인 태그
- Mock 사용: DID Resolver만 mock (블록체인 의존성 제거), 모든 암호화 기능은 실제 구현 사용

---

---

##### 8.1.3.2 잘못된 키 거부

**시험항목**: MITM/UKS 공격 방어 - 잘못된 KEM 키 거부

**Go 테스트**:

```bash
go test -v github.com/sage-x-project/sage/pkg/agent/hpke -run 'Test_Client_ResolveKEM_WrongKey_Rejects'
```

**예상 결과**:

```
--- PASS: Test_Client_ResolveKEM_WrongKey_Rejects (0.01s)
    hpke_test.go:XX: Wrong key correctly rejected
```

**검증 방법**:

- 잘못된 X25519 KEM 키로 핸드셰이크 시도
- Ack Tag 불일치로 거부 확인
- MITM/UKS 공격 방어 확인

**통과 기준**:

- ✅ 잘못된 키 거부
- ✅ "ack tag mismatch" 에러 반환
- ✅ 핸드셰이크 실패로 보안 유지

**실제 테스트 결과** (2025-10-23):

```
=== RUN   Test_Client_ResolveKEM_WrongKey_Rejects
[PASS] 공격자 X25519 키 쌍 생성 성공
  Client DID: did:sage:test:client-6e63fe7f-11fe-4013-935d-61a8d7e90e2e
  Server DID: did:sage:test:server-fe52204a-232e-45d0-8b05-4b3b313a3c07
[PASS] 잘못된 KEM 키 리졸버 생성
  시나리오: MITM 공격 시뮬레이션 (잘못된 공개키 사용)
[PASS] 잘못된 키로 핸드셰이크 시도
  Context ID: ctx-70d9d12b-6138-4353-9074-efd039567108
[PASS] 잘못된 KEM 키 올바르게 거부됨
  에러: ack tag mismatch
[PASS] Ack Tag 키 확인으로 불일치 감지
[PASS] MITM/UKS 공격 방어 성공
--- PASS: Test_Client_ResolveKEM_WrongKey_Rejects (0.00s)
```

**검증 데이터**:
- 테스트 파일: `pkg/agent/hpke/security_test.go`
- 상태: ✅ PASS
- SAGE 함수 사용:
  - ✅ `keys.GenerateX25519KeyPair()` - 공격자의 잘못된 X25519 KEM 키 생성
  - ✅ `Client.Initialize()` - HPKE 클라이언트 핸드셰이크 초기화 시도
  - ✅ `Server.HandleMessage()` - HPKE 서버 메시지 처리
- 보안 기능: Ack Tag를 통한 키 확인 (Key Confirmation)
- 공격 시나리오: MITM/UKS (Man-in-the-Middle / Unknown Key-Share) 공격
- 에러 메시지: "ack tag mismatch" - 올바른 거부 동작 확인
- 보안 결과: ✅ 잘못된 KEM 키 사용 시 핸드셰이크 실패로 공격 방지

---

---

