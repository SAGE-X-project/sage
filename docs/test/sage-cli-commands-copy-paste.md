# SAGE CLI 명령어 - 복사 붙여넣기 버전

이 문서는 `docs/test/GO_TEST_COMMANDS.md`의 Chapter 8 명령어들을 복사-붙여넣기로 바로 실행할 수 있도록 정리한 버전입니다.

각 섹션의 명령어를 순서대로 복사하여 터미널에 붙여넣으면 됩니다.

---

## 준비 사항

```bash
# 임시 디렉토리 생성
mkdir -p /tmp/sage-test
cd /tmp/sage-test

# 바이너리 경로 확인 (프로젝트 루트에서 실행)
ls -la ./build/bin/sage-crypto
ls -la ./build/bin/sage-did
```

---

## 8.1 sage-crypto CLI 검증

### 8.1.1 키 생성

**Ed25519 키 생성**:

```bash
./build/bin/sage-crypto generate --type ed25519 --format jwk --output /tmp/sage-test/test-ed25519.jwk
cat /tmp/sage-test/test-ed25519.jwk | jq -r '.private_key.kty, .private_key.crv'
```

**예상 결과**:
```
OKP
Ed25519
```

**Secp256k1 키 생성**:

```bash
./build/bin/sage-crypto generate --type secp256k1 --format jwk --output /tmp/sage-test/test-secp256k1.jwk
cat /tmp/sage-test/test-secp256k1.jwk | jq -r '.key_type'
```

**예상 결과**:
```
Secp256k1
```

---

### 8.1.2 서명 생성

**테스트 메시지 작성 및 서명**:

```bash
echo "test message" > /tmp/sage-test/msg.txt
./build/bin/sage-crypto sign --key /tmp/sage-test/test-ed25519.jwk --message-file /tmp/sage-test/msg.txt --output /tmp/sage-test/sig.bin
ls -lh /tmp/sage-test/sig.bin
```

**예상 결과**:
```
Signature saved to: /tmp/sage-test/sig.bin
-rw-r--r-- 1 user group 190 Oct 24 10:00 /tmp/sage-test/sig.bin
```

---

### 8.1.3 서명 검증

```bash
./build/bin/sage-crypto verify --key /tmp/sage-test/test-ed25519.jwk --message-file /tmp/sage-test/msg.txt --signature-file /tmp/sage-test/sig.bin
```

**예상 결과**:
```
Signature verification PASSED
Key Type: Ed25519
Key ID: 67afcf6c322beb76
```

---

### 8.1.4 Ethereum 주소 생성

```bash
./build/bin/sage-crypto address generate --key /tmp/sage-test/test-secp256k1.jwk --chain ethereum
```

**예상 결과**:
```
Key Information:
  ID: cc4f0637f14b53ec
  Type: Secp256k1

Generated Addresses:

CHAIN     ADDRESS                                     NETWORK
-----     -------                                     -------
ethereum  0xaa20392db4cc515e58edcf6a0e9b748779fd0e7c  ethereum-mainnet
```

---

## 8.2 sage-did CLI 검증

### 전제 조건

⚠️ **이 섹션을 실행하기 전에 다음 작업이 필요합니다**:

**터미널 1 - Hardhat 노드 시작**:

```bash
cd contracts/ethereum
npx hardhat node
```

**터미널 2 - 컨트랙트 배포**:

```bash
cd contracts/ethereum
npx hardhat run scripts/deploy-v4-local.js --network localhost
```

배포 후 출력되는 컨트랙트 주소를 기록하세요. 예:
```
DIDRegistryV4 deployed to: 0xDc64a140Aa3E981100a9becA4E685f962f0cF6C9
```

---

### 8.2.1 DID 등록용 키 생성

**Secp256k1 키 생성 (Primary Key)**:

```bash
./build/bin/sage-crypto generate --type secp256k1 --format jwk --output /tmp/sage-test/eth-key.jwk
cat /tmp/sage-test/eth-key.jwk | jq -r '.key_type'
```

**Ethereum 주소 확인**:

```bash
./build/bin/sage-crypto address generate --key /tmp/sage-test/eth-key.jwk --chain ethereum
```

주소를 기록하세요. 예: `0xaa20392db4cc515e58edcf6a0e9b748779fd0e7c`

**JWK 구조 확인**:

```bash
cat /tmp/sage-test/eth-key.jwk | jq '.'
```

**Ed25519 키 생성 (추가 키용, 선택 사항)**:

```bash
./build/bin/sage-crypto generate --type ed25519 --format jwk --output /tmp/sage-test/did-key.jwk
```

---

### 8.2.3 DID 등록

⚠️ **주의**: 아래 명령어의 `--contract` 값을 위에서 배포한 실제 컨트랙트 주소로 변경하세요!

**단일 키로 등록**:

```bash
./build/bin/sage-did register \
  --chain ethereum \
  --name "SAGE Test Agent" \
  --endpoint "https://agent.example.com" \
  --key /tmp/sage-test/eth-key.jwk \
  --rpc http://localhost:8545 \
  --contract 0xDc64a140Aa3E981100a9becA4E685f962f0cF6C9 \
  --private-key 0xac0974bec39a17e36ba4a6b4d238ff944bacb478cbed5efcae784d7bf4f2ff80
```

**예상 결과**:
```
Registering agent on ethereum...
Transaction Hash: 0x1234567890abcdef...
Block Number: 15
Agent registered successfully!
DID: did:sage:ethereum:12345678-1234-1234-1234-123456789abc
```

📝 **등록된 DID를 기록하세요!** 다음 단계에서 사용합니다.

---

### 8.2.2 DID 조회

⚠️ **주의**: 아래 명령어의 DID와 `--contract` 값을 실제 값으로 변경하세요!

**JSON 형식으로 조회**:

```bash
./build/bin/sage-did resolve did:sage:ethereum:12345678-1234-1234-1234-123456789abc \
  --rpc http://localhost:8545 \
  --contract 0xDc64a140Aa3E981100a9becA4E685f962f0cF6C9
```

**텍스트 형식으로 조회**:

```bash
./build/bin/sage-did resolve did:sage:ethereum:12345678-1234-1234-1234-123456789abc \
  --rpc http://localhost:8545 \
  --contract 0xDc64a140Aa3E981100a9becA4E685f962f0cF6C9 \
  --format text
```

**파일로 저장**:

```bash
./build/bin/sage-did resolve did:sage:ethereum:12345678-1234-1234-1234-123456789abc \
  --rpc http://localhost:8545 \
  --contract 0xDc64a140Aa3E981100a9becA4E685f962f0cF6C9 \
  --output /tmp/sage-test/agent-metadata.json

cat /tmp/sage-test/agent-metadata.json | jq '.'
```

---

### 8.2.4 DID 목록 조회

⚠️ **주의**: `--owner` 값을 8.2.1에서 확인한 실제 Ethereum 주소로 변경하세요!

**테이블 형식**:

```bash
./build/bin/sage-did list \
  --chain ethereum \
  --owner 0xaa20392db4cc515e58edcf6a0e9b748779fd0e7c \
  --rpc http://localhost:8545 \
  --contract 0xDc64a140Aa3E981100a9becA4E685f962f0cF6C9 \
  --format table
```

**JSON 형식**:

```bash
./build/bin/sage-did list \
  --chain ethereum \
  --owner 0xaa20392db4cc515e58edcf6a0e9b748779fd0e7c \
  --rpc http://localhost:8545 \
  --contract 0xDc64a140Aa3E981100a9becA4E685f962f0cF6C9 \
  --format json
```

**또는 Hardhat 기본 계정으로 조회**:

```bash
./build/bin/sage-did list \
  --chain ethereum \
  --owner 0xf39Fd6e51aad88F6F4ce6aB8827279cffFb92266 \
  --rpc http://localhost:8545 \
  --contract 0xDc64a140Aa3E981100a9becA4E685f962f0cF6C9 \
  --format table
```

---

### 8.2.5 키 관리 (선택 사항)

**에이전트의 모든 키 조회**:

```bash
./build/bin/sage-did key list \
  --chain ethereum \
  did:sage:ethereum:12345678-1234-1234-1234-123456789abc \
  --rpc http://localhost:8545 \
  --contract 0xDc64a140Aa3E981100a9becA4E685f962f0cF6C9
```

**에이전트에 추가 키 등록**:

```bash
./build/bin/sage-did key add \
  --chain ethereum \
  did:sage:ethereum:12345678-1234-1234-1234-123456789abc \
  --key /tmp/sage-test/did-key.jwk \
  --rpc http://localhost:8545 \
  --contract 0xDc64a140Aa3E981100a9becA4E685f962f0cF6C9
```

**키 해지**:

```bash
./build/bin/sage-did key revoke \
  --chain ethereum \
  did:sage:ethereum:12345678-1234-1234-1234-123456789abc \
  --key-id <key-id> \
  --rpc http://localhost:8545 \
  --contract 0xDc64a140Aa3E981100a9becA4E685f962f0cF6C9
```

---

## 정리

**생성된 파일 확인**:

```bash
ls -la /tmp/sage-test/
```

**파일 삭제** (원하는 경우):

```bash
rm -rf /tmp/sage-test
```

---

## 빠른 참조 - 환경 변수 설정

복사-붙여넣기를 더 쉽게 하려면 환경 변수를 설정하세요:

```bash
# 컨트랙트 주소 (배포 후 실제 값으로 변경)
export CONTRACT_ADDRESS="0xDc64a140Aa3E981100a9becA4E685f962f0cF6C9"

# RPC 엔드포인트
export RPC_URL="http://localhost:8545"

# Hardhat 기본 계정
export HARDHAT_PRIVATE_KEY="0xac0974bec39a17e36ba4a6b4d238ff944bacb478cbed5efcae784d7bf4f2ff80"
export HARDHAT_ADDRESS="0xf39Fd6e51aad88F6F4ce6aB8827279cffFb92266"

# Owner 주소 (8.2.1에서 생성한 키의 주소)
export OWNER_ADDRESS="0xaa20392db4cc515e58edcf6a0e9b748779fd0e7c"

# 등록된 DID (8.2.3에서 등록 후 실제 값으로 변경)
export REGISTERED_DID="did:sage:ethereum:12345678-1234-1234-1234-123456789abc"
```

**환경 변수를 사용한 명령어 예시**:

```bash
# DID 조회
./build/bin/sage-did resolve $REGISTERED_DID \
  --rpc $RPC_URL \
  --contract $CONTRACT_ADDRESS

# 목록 조회
./build/bin/sage-did list \
  --chain ethereum \
  --owner $OWNER_ADDRESS \
  --rpc $RPC_URL \
  --contract $CONTRACT_ADDRESS \
  --format table
```

---

## 문서 정보

- **생성일**: 2025-10-24
- **출처**: docs/test/GO_TEST_COMMANDS.md Chapter 8
- **목적**: CLI 명령어 복사-붙여넣기 실행 가이드
