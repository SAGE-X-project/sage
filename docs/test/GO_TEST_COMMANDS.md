# Go 테스트 명령어 모음

SPECIFICATION_VERIFICATION_MATRIX.md에서 추출한 모든 Go 테스트 명령어들을 카테고리별로 정리한 문서입니다.

## 목차

1. [RFC 9421 HTTP 서명 테스트](#1-rfc-9421-http-서명-테스트)
2. [암호화 키 관리 테스트](#2-암호화-키-관리-테스트)
3. [DID 테스트](#3-did-테스트)
4. [블록체인 통합 테스트](#4-블록체인-통합-테스트)
5. [메시지 처리 테스트](#5-메시지-처리-테스트)
6. [세션 관리 테스트](#6-세션-관리-테스트)
7. [HPKE 암호화 테스트](#7-hpke-암호화-테스트)
8. [CLI 도구 검증](#8-cli-도구-검증)
9. [전체 테스트 실행](#9-전체-테스트-실행)

---

## 1. RFC 9421 HTTP 서명 테스트

### 1.1 Ed25519 서명 통합 테스트

```bash
go test -v github.com/sage-x-project/sage/pkg/agent/core/rfc9421 -run 'TestIntegration/Ed25519'
```

### 1.2 ECDSA P-256 서명 통합 테스트

```bash
go test -v github.com/sage-x-project/sage/pkg/agent/core/rfc9421 -run 'TestIntegration/ECDSA_P-256'
```

### 1.3 ECDSA Secp256k1 서명 통합 테스트

```bash
go test -v github.com/sage-x-project/sage/pkg/agent/core/rfc9421 -run 'TestIntegration/ECDSA_Secp256k1'
```

### 1.4 메시지 빌더 테스트

```bash
go test -v github.com/sage-x-project/sage/pkg/agent/core/rfc9421 -run 'TestMessageBuilder'
```

### 1.5 서명자 파라미터 테스트

```bash
go test -v github.com/sage-x-project/sage/pkg/agent/core/rfc9421 -run 'TestSigner/.*Parameters'
```

### 1.6 검증자 Ed25519 테스트

```bash
go test -v github.com/sage-x-project/sage/pkg/agent/core/rfc9421 -run 'TestVerifier/.*Ed25519'
```

### 1.7 검증자 ECDSA 테스트

```bash
go test -v github.com/sage-x-project/sage/pkg/agent/core/rfc9421 -run 'TestVerifier/.*ECDSA'
```

### 1.8 변조된 서명 검증 테스트

```bash
go test -v github.com/sage-x-project/sage/pkg/agent/core/rfc9421 -run 'TestVerifier/VerifySignature_with_tampered'
```

### 1.9 Nonce 생성 테스트

```bash
go test -v github.com/sage-x-project/sage/pkg/agent/core/rfc9421 -run 'TestVerifier/NonceGeneration'
```

### 1.10 전체 RFC 9421 테스트

```bash
go test -v github.com/sage-x-project/sage/pkg/agent/core/rfc9421
```

---

## 2. 암호화 키 관리 테스트

### 2.1 Secp256k1 키 생성 테스트

```bash
go test -v github.com/sage-x-project/sage/pkg/agent/crypto/keys -run 'TestSecp256k1KeyPair/Generate'
```

### 2.2 Ed25519 키 생성 테스트

```bash
go test -v github.com/sage-x-project/sage/pkg/agent/crypto/keys -run 'TestEd25519KeyPair/Generate'
```

### 2.3 PEM 형식 테스트

```bash
go test -v github.com/sage-x-project/sage/pkg/agent/crypto/keys -run 'Test.*PEM'
```

### 2.4 암호화된 키 테스트

```bash
go test -v github.com/sage-x-project/sage/pkg/agent/crypto/keys -run 'Test.*Encrypted'
```

### 2.5 Secp256k1 서명 및 검증 테스트

```bash
go test -v github.com/sage-x-project/sage/pkg/agent/crypto/keys -run 'TestSecp256k1KeyPair/SignAndVerify'
```

### 2.6 Ed25519 서명 및 검증 테스트

```bash
go test -v github.com/sage-x-project/sage/pkg/agent/crypto/keys -run 'TestEd25519KeyPair/SignAndVerify'
```

### 2.7 전체 키 관리 테스트

```bash
go test -v github.com/sage-x-project/sage/pkg/agent/crypto/keys
```

---

## 3. DID 테스트

### 3.1 DID 생성 테스트

```bash
go test -v github.com/sage-x-project/sage/pkg/agent/did -run 'TestCreateDID'
```

### 3.2 DID 파싱 테스트

```bash
go test -v github.com/sage-x-project/sage/pkg/agent/did -run 'TestParseDID'
```

### 3.3 전체 DID 테스트

```bash
go test -v github.com/sage-x-project/sage/pkg/agent/did/...
```

---

## 4. 블록체인 통합 테스트

### 4.1 블록체인 프로바이더 설정 테스트

```bash
go test -v ./tests -run TestBlockchainProviderConfiguration
```

### 4.2 블록체인 체인 ID 테스트

```bash
go test -v ./tests -run TestBlockchainChainID
```

### 4.3 트랜잭션 서명 테스트

```bash
go test -v ./tests -run TestTransactionSigning
```

### 4.4 트랜잭션 전송 및 확인 테스트

```bash
go test -v ./tests -run TestTransactionSendAndConfirm
```

### 4.5 가스 추정 테스트

```bash
go test -v ./tests -run TestGasEstimation
```

### 4.6 컨트랙트 배포 테스트

```bash
go test -v ./tests -run TestContractDeployment
```

### 4.7 컨트랙트 상호작용 테스트

```bash
go test -v ./tests -run TestContractInteraction
```

### 4.8 컨트랙트 이벤트 테스트

```bash
go test -v ./tests -run TestContractEvents
```

### 4.9 전체 블록체인 테스트

```bash
go test -v ./tests -run TestBlockchain
```

---

## 5. 메시지 처리 테스트

### 5.1 Nonce 관리 테스트

#### 5.1.1 Nonce 생성 테스트

```bash
go test -v github.com/sage-x-project/sage/pkg/agent/core/message/nonce -run 'TestNonceManager/GenerateNonce'
```

#### 5.1.2 재생 공격 탐지 테스트

```bash
go test -v github.com/sage-x-project/sage/pkg/agent/core/message/nonce -run 'TestNonceManager/CheckReplay'
```

#### 5.1.3 Nonce 만료 테스트

```bash
go test -v github.com/sage-x-project/sage/pkg/agent/core/message/nonce -run 'TestNonceManager/Expiration'
```

### 5.2 메시지 순서 관리 테스트

#### 5.2.1 시퀀스 단조성 테스트

```bash
go test -v github.com/sage-x-project/sage/pkg/agent/core/message/order -run 'TestOrderManager/SeqMonotonicity'
```

#### 5.2.2 타임스탬프 순서 테스트

```bash
go test -v github.com/sage-x-project/sage/pkg/agent/core/message/order -run 'TestOrderManager/TimestampOrder'
```

#### 5.2.3 시퀀스 검증 테스트

```bash
go test -v github.com/sage-x-project/sage/pkg/agent/core/message/order -run 'TestOrderManager/ValidateSeq'
```

#### 5.2.4 순서 위반 테스트

```bash
go test -v github.com/sage-x-project/sage/pkg/agent/core/message/order -run 'TestOrderManager/OutOfOrder'
```

### 5.3 중복 메시지 탐지 테스트

```bash
go test -v github.com/sage-x-project/sage/pkg/agent/core/message/dedupe -run 'TestDetector/MarkAndDetectDuplicate'
```

### 5.4 메시지 검증 테스트

#### 5.4.1 재생 공격 탐지 테스트

```bash
go test -v github.com/sage-x-project/sage/pkg/agent/core/message/validator -run 'TestValidateMessage/ReplayDetection'
```

#### 5.4.2 유효성 검증 및 통계 테스트

```bash
go test -v github.com/sage-x-project/sage/pkg/agent/core/message/validator -run 'TestValidateMessage/ValidAndStats'
```

#### 5.4.3 순서 오류 테스트

```bash
go test -v github.com/sage-x-project/sage/pkg/agent/core/message/validator -run 'TestValidateMessage/OutOfOrderError'
```

### 5.5 전체 메시지 처리 테스트

```bash
go test -v github.com/sage-x-project/sage/pkg/agent/core/message/...
```

---

## 6. 세션 관리 테스트

### 6.1 세션 생성 및 관리 테스트

#### 6.1.1 중복 세션 ID 방지 테스트

```bash
go test -v github.com/sage-x-project/sage/pkg/agent/session -run 'Test_7_1_1_1_DuplicateSessionIDPrevention'
```

#### 6.1.2 세션 ID 형식 검증 테스트

```bash
go test -v github.com/sage-x-project/sage/pkg/agent/session -run 'Test_7_1_1_2_SessionIDFormatValidation'
```

#### 6.1.3 세션 메타데이터 설정 테스트

```bash
go test -v github.com/sage-x-project/sage/pkg/agent/session -run 'Test_7_1_1_3_SessionMetadataSetup'
```

### 6.2 세션 수명 주기 테스트

#### 6.2.1 세션 TTL 테스트

```bash
go test -v github.com/sage-x-project/sage/pkg/agent/session -run 'Test_7_2_1_1_SessionTTLTime'
```

#### 6.2.2 세션 정보 조회 테스트

```bash
go test -v github.com/sage-x-project/sage/pkg/agent/session -run 'Test_7_2_1_2_SessionInfoRetrieval'
```

#### 6.2.3 만료된 세션 삭제 테스트

```bash
go test -v github.com/sage-x-project/sage/pkg/agent/session -run 'Test_7_2_1_3_ExpiredSessionDeletion'
```

### 6.3 전체 세션 관리 테스트

```bash
go test -v github.com/sage-x-project/sage/pkg/agent/session
```

---

## 7. HPKE 암호화 테스트

### 7.1 HPKE 기본 내보내기 테스트

```bash
go test -v github.com/sage-x-project/sage/pkg/agent/hpke -run 'Test_HPKE_Base_Exporter_To_Session'
```

### 7.2 서버 서명 및 ACK 태그 테스트

```bash
go test -v github.com/sage-x-project/sage/pkg/agent/hpke -run 'Test_ServerSignature_And_AckTag_HappyPath'
```

### 7.3 잘못된 키 거부 테스트

```bash
go test -v github.com/sage-x-project/sage/pkg/agent/hpke -run 'Test_Client_ResolveKEM_WrongKey_Rejects'
```

### 7.4 전체 HPKE 테스트

```bash
go test -v github.com/sage-x-project/sage/pkg/agent/hpke
```

---

## 8. CLI 도구 검증

### 8.1 sage-crypto CLI

#### 8.1.1 키 생성 CLI

**Ed25519 키 생성**:

```bash
./build/bin/sage-crypto generate --type ed25519 --format jwk --output /tmp/test-ed25519.jwk
test -f /tmp/test-ed25519.jwk && echo "✓ 키 생성 성공"
cat /tmp/test-ed25519.jwk | jq -r '.private_key.kty, .private_key.crv'
```

**예상 결과**:
```
✓ 키 생성 성공
OKP
Ed25519
```

**Secp256k1 키 생성**:

```bash
./build/bin/sage-crypto generate --type secp256k1 --format jwk --output /tmp/test-secp256k1.jwk
```

#### 8.1.2 서명 CLI

**메시지 서명**:

```bash
# 메시지 작성
echo "test message" > /tmp/msg.txt

# 서명 생성
./build/bin/sage-crypto sign --key /tmp/test-ed25519.jwk --message-file /tmp/msg.txt --output /tmp/sig.bin

# 확인
test -f /tmp/sig.bin && echo "✓ 서명 생성 성공"
ls -lh /tmp/sig.bin
```

**예상 결과**:
```
Signature saved to: /tmp/sig.bin
✓ 서명 생성 성공
-rw-r--r-- 1 user group 190 Oct 22 10:00 /tmp/sig.bin
```

#### 8.1.3 검증 CLI

**서명 검증**:

```bash
./build/bin/sage-crypto verify --key /tmp/test-ed25519.jwk --message-file /tmp/msg.txt --signature-file /tmp/sig.bin
```

**예상 결과**:
```
Signature verification PASSED
Key Type: Ed25519
Key ID: 67afcf6c322beb76
```

#### 8.1.4 주소 생성 CLI (Ethereum)

**Ethereum 주소 생성**:

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

### 8.2 sage-did CLI

**주요 명령어**:
- `register`: 새로운 AI 에이전트 등록
- `resolve`: DID로 에이전트 메타데이터 조회
- `list`: 소유자 주소로 에이전트 목록 조회
- `update`: 에이전트 메타데이터 업데이트
- `deactivate`: 에이전트 비활성화
- `key`: 에이전트 키 관리 (add, list, revoke, verify-pop)

#### 8.2.1 DID 키 생성 (sage-crypto 사용)

**Ed25519 키 생성**:

```bash
# sage-crypto를 사용하여 Ed25519 키 생성
./build/bin/sage-crypto generate --type ed25519 --format jwk --output /tmp/did-key.jwk

# 키 정보 확인
cat /tmp/did-key.jwk | jq -r '.private_key.kty, .private_key.crv'
```

**예상 결과**:
```
Key saved to: /tmp/did-key.jwk
OKP
Ed25519
```

**Secp256k1 키 생성 (Ethereum 호환)**:

```bash
./build/bin/sage-crypto generate --type secp256k1 --format jwk --output /tmp/eth-key.jwk
```

#### 8.2.2 DID 조회 CLI

**기본 조회 (JSON 형식)**:

```bash
./build/bin/sage-did resolve did:sage:ethereum:<agent-id> \
  --rpc http://localhost:8545 \
  --contract 0xYourContractAddress
```

**텍스트 형식으로 조회**:

```bash
./build/bin/sage-did resolve did:sage:ethereum:<agent-id> \
  --rpc http://localhost:8545 \
  --contract 0xYourContractAddress \
  --format text
```

**예상 결과** (JSON):
```json
{
  "did": "did:sage:ethereum:xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx",
  "publicKey": "0x1234...",
  "endpoint": "https://agent.example.com",
  "owner": "0xabcd...",
  "isActive": true,
  "name": "My Agent",
  "capabilities": {}
}
```

#### 8.2.3 DID 등록 CLI

**단일 키로 등록**:

```bash
# 로컬 블록체인 노드 실행 필요 (Hardhat/Anvil)
./build/bin/sage-did register \
  --chain ethereum \
  --name "My SAGE Agent" \
  --endpoint "https://agent.example.com" \
  --key /tmp/eth-key.jwk \
  --rpc http://localhost:8545 \
  --contract 0xYourContractAddress \
  --private-key 0xYourPrivateKeyForGasFees
```

**다중 키로 등록 (Multi-Key Registration)**:

```bash
# 여러 타입의 키로 등록 (Ed25519, ECDSA, X25519)
./build/bin/sage-did register \
  --chain ethereum \
  --name "Multi-Key Agent" \
  --endpoint "https://agent.example.com" \
  --key /tmp/eth-key.jwk \
  --additional-keys /tmp/ed25519.jwk,/tmp/x25519.key \
  --key-types ed25519,x25519 \
  --rpc http://localhost:8545 \
  --contract 0xYourContractAddress \
  --private-key 0xYourPrivateKeyForGasFees
```

**예상 결과**:
```
Registering agent on ethereum...
Transaction Hash: 0x1234567890abcdef...
Block Number: 15
Agent registered successfully!
DID: did:sage:ethereum:xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx
```

#### 8.2.4 DID 목록 조회 CLI

**테이블 형식으로 조회**:

```bash
./build/bin/sage-did list \
  --chain ethereum \
  --owner 0x742d35Cc6634C0532925a3b844Bc9e7595f2bd80 \
  --rpc http://localhost:8545 \
  --contract 0xYourContractAddress \
  --format table
```

**JSON 형식으로 조회**:

```bash
./build/bin/sage-did list \
  --chain ethereum \
  --owner 0x742d35Cc6634C0532925a3b844Bc9e7595f2bd80 \
  --rpc http://localhost:8545 \
  --contract 0xYourContractAddress \
  --format json
```

**예상 결과** (테이블):
```
Owner: 0x742d35Cc6634C0532925a3b844Bc9e7595f2bd80

DID                                                            | Name        | Active
---------------------------------------------------------------|-------------|--------
did:sage:ethereum:12345678-1234-1234-1234-123456789abc       | Agent 1     | true
did:sage:ethereum:abcdefab-abcd-abcd-abcd-abcdefabcdef       | Agent 2     | true

Total: 2 agents
```

#### 8.2.5 키 관리 CLI

**에이전트에 추가 키 등록**:

```bash
./build/bin/sage-did key add \
  --chain ethereum \
  did:sage:ethereum:<agent-id> \
  --key /tmp/new-key.jwk \
  --rpc http://localhost:8545 \
  --contract 0xYourContractAddress
```

**에이전트의 모든 키 조회**:

```bash
./build/bin/sage-did key list \
  --chain ethereum \
  did:sage:ethereum:<agent-id> \
  --rpc http://localhost:8545 \
  --contract 0xYourContractAddress
```

**키 해지**:

```bash
./build/bin/sage-did key revoke \
  --chain ethereum \
  did:sage:ethereum:<agent-id> \
  --key-id <key-id> \
  --rpc http://localhost:8545 \
  --contract 0xYourContractAddress
```

---

## 9. 전체 테스트 실행

### 9.1 모든 테스트 실행 (간략)

```bash
go test ./...
```

### 9.2 모든 테스트 실행 (상세)

```bash
go test -v ./...
```

### 9.3 커버리지 포함 테스트

```bash
go test -cover ./...
```

### 9.4 헬스 체크 테스트

```bash
go test -v github.com/sage-x-project/sage/pkg/health
```

### 9.5 통합 테스트

```bash
go test -v ./tests/integration
```

---

## 테스트 실행 팁

### 특정 패키지만 테스트

```bash
# 특정 패키지 경로 지정
go test -v github.com/sage-x-project/sage/pkg/agent/[패키지명]
```

### 특정 테스트만 실행

```bash
# -run 플래그로 정규표현식 패턴 지정
go test -v [패키지경로] -run '[테스트명패턴]'
```

### 병렬 실행

```bash
# -parallel 플래그로 병렬 실행 수 지정
go test -v -parallel 4 ./...
```

### 타임아웃 설정

```bash
# -timeout 플래그로 전체 테스트 타임아웃 설정
go test -v -timeout 30m ./...
```

### 실패한 테스트만 재실행

```bash
# -failfast 플래그로 첫 실패 시 중단
go test -v -failfast ./...
```

---

## 테스트 카테고리별 실행 순서 권장사항

### 1단계: 기본 단위 테스트

```bash
go test -v github.com/sage-x-project/sage/pkg/agent/crypto/keys
go test -v github.com/sage-x-project/sage/pkg/agent/core/rfc9421
```

### 2단계: 메시지 처리 테스트

```bash
go test -v github.com/sage-x-project/sage/pkg/agent/core/message/...
```

### 3단계: 세션 및 암호화 테스트

```bash
go test -v github.com/sage-x-project/sage/pkg/agent/session
go test -v github.com/sage-x-project/sage/pkg/agent/hpke
```

### 4단계: DID 및 블록체인 통합 테스트

```bash
go test -v github.com/sage-x-project/sage/pkg/agent/did/...
go test -v ./tests -run TestBlockchain
```

### 5단계: 전체 통합 테스트

```bash
go test -v ./tests/integration
go test -cover ./...
```

### 6단계: CLI 도구 검증

**sage-crypto CLI 검증**:

```bash
# 키 생성
./build/bin/sage-crypto generate --type ed25519 --format jwk --output /tmp/test-ed25519.jwk
./build/bin/sage-crypto generate --type secp256k1 --format jwk --output /tmp/test-secp256k1.jwk

# 서명 및 검증
echo "test message" > /tmp/msg.txt
./build/bin/sage-crypto sign --key /tmp/test-ed25519.jwk --message-file /tmp/msg.txt --output /tmp/sig.bin
./build/bin/sage-crypto verify --key /tmp/test-ed25519.jwk --message-file /tmp/msg.txt --signature-file /tmp/sig.bin

# Ethereum 주소 생성
./build/bin/sage-crypto address generate --key /tmp/test-secp256k1.jwk --chain ethereum
```

**sage-did CLI 검증** (TODO: 수정 필요):

```bash
# DID 키 생성
./build/bin/sage-did key create --type ed25519 --output /tmp/did-key.jwk

# DID 조회
./build/bin/sage-did resolve did:sage:ethereum:test-123

# DID 등록 (로컬 블록체인 필요)
./build/bin/sage-did register --key /tmp/did-key.jwk --chain ethereum --network local

# DID 목록 조회
./build/bin/sage-did list --owner 0x742d35Cc6634C0532925a3b844Bc9e7595f2bd80
```

---

## 문서 정보

- **생성일**: 2025-10-24
- **최종 업데이트**: 2025-10-24
- **출처**: SPECIFICATION_VERIFICATION_MATRIX.md
- **목적**: Go 테스트 및 CLI 명령어 체계적 정리 및 참조 문서
- **추가된 내용**: 6장 CLI 도구 검증 (sage-crypto, sage-did)
