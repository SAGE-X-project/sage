## 2. 암호화 키 관리

### 2.1 키 생성

#### 2.1.1 Secp256k1 키 생성 (32바이트 개인키)

**시험항목**: Secp256k1 키 쌍 생성 (Ethereum 호환)

**Go 테스트**:

```bash
go test -v github.com/sage-x-project/sage/pkg/agent/crypto/keys -run 'TestSecp256k1KeyPair/Generate'
```

**CLI 검증**:

```bash
./build/bin/sage-crypto generate --type secp256k1 --format jwk --output /tmp/test-secp256k1.jwk
cat /tmp/test-secp256k1.jwk | jq '.'
```

**예상 결과**:

```
--- PASS: TestSecp256k1KeyPair/Generate (0.00s)
    keys_test.go:XX: Private key size: 32 bytes
    keys_test.go:XX: Public key size: 33/65 bytes (compressed/uncompressed)
```

**검증 방법**:

- 개인키 크기 = 32 bytes 확인
- 공개키 압축 형식 = 33 bytes 확인
- 공개키 비압축 형식 = 65 bytes 확인
- Ethereum 호환성 확인

**통과 기준**:

- ✅ Secp256k1 키 생성 성공
- ✅ 개인키 = 32 bytes
- ✅ 공개키 형식 정확
- ✅ Ethereum 호환

**실제 테스트 결과** (2025-10-23):

```
=== RUN   TestSecp256k1KeyPair/GenerateKeyPair
===== 2.1.1 Secp256k1 Complete Key Lifecycle (Generation + Secure Storage + Verification) =====
[PASS] Secp256k1 key pair generated successfully
[PASS] Key type confirmed: Secp256k1
[PASS] Private key size validated: 32 bytes
[PASS] Public key size validated: 65 bytes (uncompressed)
[PASS] Ethereum address generated
[PASS] Signature generated: 65 bytes (Ethereum format)
[PASS] Signature verification successful - Key is cryptographically valid
[PASS] FileVault initialized (AES-256-GCM + PBKDF2)
[PASS] Key encrypted and stored securely
[PASS] File permissions verified: 0600 (owner read/write only)
[PASS] Key decrypted successfully with correct passphrase
[PASS] Wrong passphrase correctly rejected - Security validated
[PASS] Secp256k1 key pair reconstructed from stored data
[PASS] Address recovery successful - Key fully functional after storage/loading
--- PASS: TestSecp256k1KeyPair/GenerateKeyPair (0.04s)
```

**검증 데이터**:
- 테스트 데이터 파일: `testdata/keys/secp256k1_key_generation.json`
- 상태: ✅ PASS
- Private key: 32 bytes (verified)
- Uncompressed public key: 65 bytes (verified)
- Signature size: 65 bytes (Ethereum format with recovery byte)
- Secure storage: AES-256-GCM + PBKDF2 (100,000 iterations)
- File permissions: 0600 (verified)
- Complete lifecycle: Generation → Storage → Loading → Reuse (verified)

---

---

#### 2.1.2 Ed25519 키 생성 (32바이트 공개키, 64바이트 비밀키)

**시험항목**: Ed25519 키 쌍 생성 및 크기 확인

**Go 테스트**:

```bash
go test -v github.com/sage-x-project/sage/pkg/agent/crypto/keys -run 'TestEd25519KeyPair/Generate'
```

**CLI 검증**:

```bash
./build/bin/sage-crypto generate --type ed25519 --format jwk --output /tmp/test-ed25519.jwk
cat /tmp/test-ed25519.jwk | jq '.'
```

**예상 결과**:

```
--- PASS: TestEd25519KeyPair/Generate (0.00s)
    keys_test.go:XX: Public key size: 32 bytes
    keys_test.go:XX: Private key size: 64 bytes
```

**검증 방법**:

- 공개키 크기 = 32 bytes 확인
- 비밀키 크기 = 64 bytes 확인
- JWK 형식 유효성 확인

**통과 기준**:

- ✅ Ed25519 키 생성 성공
- ✅ 공개키 = 32 bytes
- ✅ 비밀키 = 64 bytes
- ✅ JWK 형식 정확

**실제 테스트 결과** (2025-10-23):

```
=== RUN   TestEd25519KeyPair/GenerateKeyPair
===== 2.1.2 Ed25519 Complete Key Lifecycle (Generation + Secure Storage + Verification) =====
[PASS] Ed25519 key pair generated successfully
[PASS] Key type confirmed: Ed25519
[PASS] Public key size validated: 32 bytes
[PASS] Private key size validated: 64 bytes
[PASS] Signature generated: 64 bytes (Ed25519 format)
[PASS] Signature verification successful - Key is cryptographically valid
[PASS] FileVault initialized (AES-256-GCM + PBKDF2)
[PASS] Key encrypted and stored securely
[PASS] File permissions verified: 0600 (owner read/write only)
[PASS] Key decrypted successfully with correct passphrase
[PASS] Wrong passphrase correctly rejected - Security validated
[PASS] Ed25519 key pair reconstructed from stored data
[PASS] Signature verified with reconstructed public key - Key fully functional after storage/loading
--- PASS: TestEd25519KeyPair/GenerateKeyPair (0.04s)
```

**검증 데이터**:
- 테스트 데이터 파일: `testdata/keys/ed25519_key_generation.json`
- 상태: ✅ PASS
- Public key: 32 bytes (verified)
- Private key: 64 bytes (verified)
- Signature size: 64 bytes (Ed25519 standard)
- Secure storage: AES-256-GCM + PBKDF2 (100,000 iterations)
- File permissions: 0600 (verified)
- Complete lifecycle: Generation → Storage → Loading → Reuse (verified)

---

---

### 2.2 키 저장

#### 2.2.1 PEM 형식 저장

**시험항목**: PEM 형식으로 키 저장/로드 (Ed25519만 지원)

**Go 테스트**:

```bash
go test -v github.com/sage-x-project/sage/pkg/agent/crypto/keys -run 'Test.*PEM'
```

**CLI 검증**:

```bash
./build/bin/sage-crypto generate --type ed25519 --format pem --output /tmp/test.pem
cat /tmp/test.pem
# 출력: -----BEGIN PRIVATE KEY----- ...
```

**예상 결과**:

```
--- PASS: TestEd25519KeyPairPEM (0.00s)
```

**검증 방법**:

- PEM 헤더/푸터 존재 확인
- Base64 인코딩 확인
- 저장 후 로드 가능 확인

**통과 기준**:

- ✅ PEM 형식 저장 성공
- ✅ PEM 형식 로드 성공
- ✅ 키 일치 확인

---

**실제 테스트 결과** (2025-10-23):

✅ **Ed25519 - PASS** (`TestEd25519KeyPairPEM`)
- PEM format: PKCS#8 DER encoding
- File permissions: 0600 (verified)
- Custom path support: ✅ (via `os.WriteFile(customPath, ...)`)
- Load and verify: ✅ (signature validation passed)
- Public key PEM export: ✅
- Data file: `testdata/keys/ed25519_pem_storage.json`

⚠️ **Secp256k1 - NOT SUPPORTED**
- **Reason**: x509 package only supports NIST curves (P-256, P-384, P-521)
- **Alternative**: Use FileVault encrypted storage (see 2.2.2)
- **Error**: `x509: unknown curve while marshaling to PKCS#8`

---

---

#### 2.2.2 암호화 저장

**시험항목**: 패스워드로 암호화된 키 저장 (Secp256k1, Ed25519 모두 지원)

**Go 테스트**:

```bash
go test -v github.com/sage-x-project/sage/pkg/agent/crypto/keys -run 'Test.*Encrypted'
```

**예상 결과**:

```
--- PASS: TestSecp256k1KeyPairEncrypted (0.11s)
--- PASS: TestEd25519KeyPairEncrypted (0.10s)
```

**검증 방법**:

- 패스워드로 키 암호화 확인
- 올바른 패스워드로 복호화 성공 확인
- 잘못된 패스워드로 복호화 실패 확인
- 복호화된 키로 서명/검증 확인

**통과 기준**:

- ✅ 암호화 저장 성공
- ✅ 올바른 패스워드로 로드 성공
- ✅ 잘못된 패스워드 거부
- ✅ 키 재사용 가능

---

**실제 테스트 결과** (2025-10-23):

✅ **Secp256k1 - PASS** (`TestSecp256k1KeyPairEncrypted`)
- Encryption: AES-256-GCM + PBKDF2 (100,000 iterations)
- File permissions: 0600 (verified)
- Custom path: ✅ (via `vault.NewFileVault(customPath)`)
- Correct passphrase: ✅ (decryption successful)
- Wrong passphrase: ✅ (correctly rejected)
- Key reconstruction: ✅ (32 bytes private key)
- Signature verification: ✅ (65 bytes Ethereum format)
- Ethereum address consistency: ✅
- Data file: `testdata/keys/secp256k1_encrypted_storage.json`

✅ **Ed25519 - PASS** (`TestEd25519KeyPairEncrypted`)
- Encryption: AES-256-GCM + PBKDF2 (100,000 iterations)
- File permissions: 0600 (verified)
- Custom path: ✅ (via `vault.NewFileVault(customPath)`)
- Correct passphrase: ✅ (decryption successful)
- Wrong passphrase: ✅ (correctly rejected)
- Key reconstruction: ✅ (64 bytes private key)
- Signature verification: ✅ (64 bytes signature)
- Data file: `testdata/keys/ed25519_encrypted_storage.json`

**암호화 저장 기능:**
- Storage: SAGE FileVault (애플리케이션 레벨 구현)
- Encryption: AES-256-GCM
- Key derivation: PBKDF2 with SHA-256 (100,000 iterations)
- Salt: 32 bytes random
- File permissions: 0600 (owner read/write only)
- Custom path support: ✅
- Empty passphrase: ✅ (handled correctly)
- Key overwrite: ✅ (with new passphrase)
- Key deletion: ✅

**Note**: 2.1.1 및 2.1.2의 Complete Lifecycle 테스트에도 암호화 저장이 포함되어 있으며, 2.2.2는 암호화 저장에 특화된 전용 테스트입니다.

---

### 2.3 서명/검증

#### 2.3.1 Secp256k1 서명/검증

**시험항목**: Secp256k1 ECDSA 서명/검증 및 주소 복구

**Go 테스트**:

```bash
go test -v github.com/sage-x-project/sage/pkg/agent/crypto/keys -run 'TestSecp256k1KeyPair/SignAndVerify'
```

**CLI 검증** (✅ 실제 동작 확인됨):

```bash
# 1. Secp256k1 키 생성
./build/bin/sage-crypto generate --type secp256k1 --format jwk --output secp256k1.jwk

# 2. 메시지 파일 생성
echo "test message for secp256k1" > message.txt

# 3. 서명 생성 (65 bytes: 64 bytes ECDSA + 1 byte recovery)
./build/bin/sage-crypto sign --key secp256k1.jwk --message-file message.txt --output signature.bin

# 4. 서명 검증 (주소 복구 포함)
./build/bin/sage-crypto verify --key secp256k1.jwk --message-file message.txt --signature-file signature.bin
# 출력: Signature verification PASSED
#       Key Type: Secp256k1
#       Key ID: [key_id]
```

**예상 결과**:

```
--- PASS: TestSecp256k1KeyPair/SignAndVerify (0.01s)
    secp256k1_test.go:308: [PASS] Signature generation successful
    secp256k1_test.go:309:   Signature size: 65 bytes (expected: 65 bytes)
    secp256k1_test.go:316: [PASS] Signature verification successful
    secp256k1_test.go:328: [PASS] Address recovery successful (Ethereum compatible)
```

**검증 방법**:

- ECDSA 서명 생성 확인 (65 bytes)
- 서명 검증 성공 확인 (`keyPair.Verify()`)
- Ethereum 주소 복구 확인 (`ethcrypto.SigToPub()`)
- 변조 탐지 확인

**통과 기준**:

- ✅ Secp256k1 서명 생성 (65 bytes)
- ✅ 검증 성공
- ✅ Ethereum 호환 (주소 복구)
- ✅ 변조 탐지

---

**실제 테스트 결과** (2025-10-23):

✅ **Secp256k1 - PASS** (`TestSecp256k1KeyPair/SignAndVerify`)
- Signature generation: ✅ (using `keyPair.Sign()` → ECDSA)
- Signature size: 65 bytes (64 bytes ECDSA + 1 byte recovery v)
- Signature verification: ✅ (using `keyPair.Verify()`)
- Address recovery: ✅ (Ethereum compatible via `ethcrypto.SigToPub()`)
- Tamper detection:
  - Wrong message: ✅ (correctly rejected with `crypto.ErrInvalidSignature`)
  - Modified signature: ✅ (correctly rejected with `crypto.ErrInvalidSignature`)
- Data file: `testdata/keys/secp256k1_sign_verify.json`

**기능 구현:**
- 서명 생성: `pkg/agent/crypto/keys/secp256k1.go` - `Sign()`
- 서명 검증: `pkg/agent/crypto/keys/secp256k1.go` - `Verify()`
- 주소 복구: `github.com/ethereum/go-ethereum/crypto` - `SigToPub()`
- CLI: `cmd/sage-crypto/sign.go`, `cmd/sage-crypto/verify.go`

---

---

#### 2.3.2 Ed25519 서명/검증 (64바이트 서명)

**시험항목**: Ed25519 서명 생성 및 검증

**Go 테스트**:

```bash
go test -v github.com/sage-x-project/sage/pkg/agent/crypto/keys -run 'TestEd25519KeyPair/SignAndVerify'
```

**CLI 검증** (✅ 실제 동작 확인됨):

```bash
# 1. Ed25519 키 생성
./build/bin/sage-crypto generate --type ed25519 --format jwk --output ed25519.jwk

# 2. 메시지 파일 생성
echo "test message" > message.txt

# 3. 서명 생성 (64 bytes)
./build/bin/sage-crypto sign --key ed25519.jwk --message-file message.txt --output signature.bin

# 4. 서명 검증
./build/bin/sage-crypto verify --key ed25519.jwk --message-file message.txt --signature-file signature.bin
# 출력: Signature verification PASSED
#       Key Type: Ed25519
#       Key ID: [key_id]
```

**예상 결과**:

```
--- PASS: TestEd25519KeyPair/SignAndVerify (0.00s)
    ed25519_test.go:284: [PASS] Signature generation successful
    ed25519_test.go:285:   Signature size: 64 bytes (expected: 64 bytes)
    ed25519_test.go:291: [PASS] Signature verification successful
    ed25519_test.go:298: [PASS] Tamper detection: Wrong message rejected
```

**검증 방법**:

- 서명 크기 = 64 bytes 확인
- 유효한 서명 검증 성공 확인 (`keyPair.Verify()`)
- 변조된 메시지 검증 실패 확인
- 변조된 서명 검증 실패 확인

**통과 기준**:

- ✅ 서명 생성 성공 (64 bytes)
- ✅ 검증 성공
- ✅ 변조 탐지

---

**실제 테스트 결과** (2025-10-23):

✅ **Ed25519 - PASS** (`TestEd25519KeyPair/SignAndVerify`)
- Signature generation: ✅ (using `keyPair.Sign()` → EdDSA)
- Signature size: 64 bytes (exactly)
- Signature verification: ✅ (using `keyPair.Verify()`)
- Tamper detection:
  - Wrong message: ✅ (correctly rejected with `crypto.ErrInvalidSignature`)
  - Modified signature: ✅ (correctly rejected with `crypto.ErrInvalidSignature`)
- Data file: `testdata/keys/ed25519_sign_verify.json`

**기능 구현:**
- 서명 생성: `pkg/agent/crypto/keys/ed25519.go` - `Sign()`
- 서명 검증: `pkg/agent/crypto/keys/ed25519.go` - `Verify()`
- Native: `crypto/ed25519` 표준 라이브러리 사용
- CLI: `cmd/sage-crypto/sign.go`, `cmd/sage-crypto/verify.go`

---

---

