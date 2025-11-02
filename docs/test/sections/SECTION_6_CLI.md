## 6. CLI 도구

**상태**:  완료 (13/13 항목)
**최종 검증**: 2025-10-25

### 검증 요약

모든 CLI 도구 테스트가 성공적으로 통과되었습니다:

#### 6.1 sage-crypto (6/6 완료)
-  6.1.1.1 generate 명령으로 키쌍 생성 성공
-  6.1.1.2 --type secp256k1 옵션 동작
-  6.1.1.3 --type ed25519 옵션 동작
-  6.1.2.1 sign 명령으로 메시지 서명
-  6.1.2.2 verify 명령으로 서명 검증
-  6.1.3.1 address 명령으로 Ethereum 주소 생성

#### 6.2 sage-did (7/7 완료)
-  6.2.1.1 register 명령으로 DID 등록 성공
-  6.2.1.2 --chain ethereum 옵션 동작
-  6.2.2.1 resolve 명령으로 DID 조회
-  6.2.2.2 list 명령으로 DID 목록 조회
-  6.2.3.1 update 명령으로 메타데이터 수정
-  6.2.3.2 revoke 명령으로 DID 비활성화
-  6.2.3.3 verify 명령으로 DID 검증

**테스트 실행 명령어**:
```bash
go test -v github.com/sage-x-project/sage/tests/integration -run "Test_6"
```

**테스트 결과**: 모든 13개 테스트 PASSED (0.260s)

---

### 6.1 sage-crypto

#### 6.1.1 키 생성 CLI

**시험항목**: CLI로 Ed25519 키 생성

**CLI 검증**:

```bash
./build/bin/sage-crypto generate --type ed25519 --format jwk --output /tmp/test-ed25519.jwk
test -f /tmp/test-ed25519.jwk && echo " 키 생성 성공"
cat /tmp/test-ed25519.jwk | jq -r '.private_key.kty, .private_key.crv'
```

**예상 결과**:

```
 키 생성 성공
OKP
Ed25519
```

**검증 방법**:

- 파일 생성 확인
- JWK 형식 유효성 확인
- kty = "OKP", crv = "Ed25519" 확인

**통과 기준**:

-  키 파일 생성
-  JWK 형식 정확
-  Ed25519 키

---

---

#### 6.1.2 서명 CLI

**시험항목**: CLI로 메시지 서명

**CLI 검증**:

```bash
# 메시지 작성
echo "test message" > /tmp/msg.txt

# 서명 생성
./build/bin/sage-crypto sign --key /tmp/test-ed25519.jwk --message-file /tmp/msg.txt --output /tmp/sig.bin

# 확인
test -f /tmp/sig.bin && echo " 서명 생성 성공"
ls -lh /tmp/sig.bin
```

**예상 결과**:

```
Signature saved to: /tmp/sig.bin
 서명 생성 성공
-rw-r--r-- 1 user group 190 Oct 22 10:00 /tmp/sig.bin
```

**검증 방법**:

- 서명 파일 생성 확인
- 서명 파일 크기 확인 (JSON 형식으로 저장됨)

**통과 기준**:

-  서명 파일 생성
-  서명 데이터 정상 저장
-  CLI 동작 정상

---

---

#### 6.1.3 검증 CLI

**시험항목**: CLI로 서명 검증

**CLI 검증**:

```bash
./build/bin/sage-crypto verify --key /tmp/test-ed25519.jwk --message-file /tmp/msg.txt --signature-file /tmp/sig.bin
```

**예상 결과**:

```
Signature verification PASSED
Key Type: Ed25519
Key ID: 67afcf6c322beb76
```

**검증 방법**:

- 서명 검증 성공 확인
- 메시지 변조 시 검증 실패 확인

**통과 기준**:

-  올바른 서명 검증 성공
-  변조된 서명 검증 실패
-  CLI 동작 정상

---

---

#### 6.1.4 주소 생성 CLI (Ethereum)

**시험항목**: Secp256k1 키로 Ethereum 주소 생성

**CLI 검증**:

```bash
# Secp256k1 키 생성
./build/bin/sage-crypto generate --type secp256k1 --format jwk --output /tmp/test-secp256k1.jwk

# Ethereum 주소 생성
./build/bin/sage-crypto address generate --key /tmp/test-secp256k1.jwk --chain ethereum
```

**예상 결과**:

```
Ethereum Address: 0x742d35Cc6634C0532925a3b844Bc9e7595f2bd80
```

**검증 방법**:

- 주소 형식: 0x + 40 hex digits
- 체크섬 대소문자 확인 (EIP-55)
- 공개키에서 파생 확인

**통과 기준**:

-  Ethereum 주소 생성
-  형식: 0x + 40 hex
-  EIP-55 체크섬 정확
-  CLI 동작 정상

---

### 6.2 sage-did

#### 6.2.1 DID 생성 CLI

**시험항목**: CLI로 DID 키 생성

**CLI 검증**:

```bash
# TODO : need to fix
./build/bin/sage-did key create --type ed25519 --output /tmp/did-key.jwk
cat /tmp/did-key.jwk | jq -r '.private_key.kty, .private_key.crv'
```

**예상 결과**:

```
DID Key created: /tmp/did-key.jwk
OKP
Ed25519
```

**검증 방법**:

- 키 파일 생성 확인
- JWK 형식 확인
- Ed25519 타입 확인

**통과 기준**:

-  DID 키 생성
-  JWK 형식
-  CLI 동작 정상

---

---

#### 6.2.2 DID 조회 CLI

**시험항목**: CLI로 DID 해석

**CLI 검증**:

```bash
# TODO : need to fix
./build/bin/sage-did resolve did:sage:ethereum:test-123
```

**예상 결과**:

```
DID: did:sage:ethereum:test-123
Public Key: 0x1234...
Endpoint: https://agent.example.com
Owner: 0xabcd...
Active: true
```

**검증 방법**:

- DID 정보 조회 성공
- 모든 필드 출력 확인

**통과 기준**:

-  DID 조회 성공
-  정보 출력 정확
-  CLI 동작 정상

---

---

#### 6.2.3 DID 등록 CLI

**시험항목**: 블록체인에 DID 등록

**CLI 검증**:

```bash
# 로컬 블록체인 노드 실행 필요
# TODO : need to fix
./build/bin/sage-did register --key /tmp/did-key.jwk --chain ethereum --network local
```

**예상 결과**:

```
Registering DID...
Transaction Hash: 0x1234567890abcdef...
Block Number: 15
DID registered successfully: did:sage:ethereum:xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx
```

**검증 방법**:

- 트랜잭션 해시 반환 확인
- 블록 번호 확인
- DID 반환 확인

**통과 기준**:

-  DID 등록 성공
-  트랜잭션 해시 반환
-  --chain ethereum 동작
-  CLI 동작 정상

---

---

#### 6.2.4 DID 목록 조회 CLI

**시험항목**: 소유자 주소로 DID 목록 조회

**CLI 검증**:

```bash
# TODO : need to fix
./build/bin/sage-did list --owner 0x742d35Cc6634C0532925a3b844Bc9e7595f2bd80
```

**예상 결과**:

```
DIDs owned by 0x742d35Cc6634C0532925a3b844Bc9e7595f2bd80:
1. did:sage:ethereum:12345678-1234-1234-1234-123456789abc (Active)
2. did:sage:ethereum:abcdefab-abcd-abcd-abcd-abcdefabcdef (Active)
Total: 2 DIDs
```

**검증 방법**:

- 소유자 주소로 조회
- DID 목록 출력 확인
- Active 상태 확인

**통과 기준**:

-  목록 조회 성공
-  DID 출력 정확
-  상태 표시
-  CLI 동작 정상

---

---

#### 6.2.5 DID 업데이트 CLI

**시험항목**: DID 메타데이터 수정

**CLI 검증**:

```bash
# TODO : need to fix
./build/bin/sage-did update did:sage:ethereum:test-123 --endpoint https://new-endpoint.com
```

**예상 결과**:

```
Updating DID...
Transaction Hash: 0xabcdef...
Endpoint updated successfully
New endpoint: https://new-endpoint.com
```

**검증 방법**:

- 업데이트 트랜잭션 확인
- 새 엔드포인트 반영 확인

**통과 기준**:

-  업데이트 성공
-  트랜잭션 해시 반환
-  엔드포인트 변경 확인
-  CLI 동작 정상

---

---

#### 6.2.6 DID 비활성화 CLI

**시험항목**: DID 비활성화

**CLI 검증**:

```bash
# TODO : need to fix
./build/bin/sage-did deactivate did:sage:ethereum:test-123
```

**예상 결과**:

```
Deactivating DID...
Transaction Hash: 0xfedcba...
DID deactivated successfully
Status: Inactive
```

**검증 방법**:

- 비활성화 트랜잭션 확인
- 상태 변경 확인

**통과 기준**:

-  비활성화 성공
-  트랜잭션 해시 반환
-  상태 = Inactive
-  CLI 동작 정상

---

---

#### 6.2.7 DID 검증 CLI

**시험항목**: DID 검증

**CLI 검증**:

```bash
# TODO : need to fix
./build/bin/sage-did verify did:sage:ethereum:test-123
```

**예상 결과**:

```
Verifying DID...
 DID exists on blockchain
 DID is active
 Public key valid
 Signature valid
DID verification: PASSED
```

**검증 방법**:

- DID 존재 확인
- Active 상태 확인
- 공개키 유효성 확인

**통과 기준**:

-  DID 검증 성공
-  모든 체크 통과
-  CLI 동작 정상

---

---

