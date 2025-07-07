# SAGE CLI 도구 문서

SAGE는 암호화 작업과 DID 관리를 위한 두 가지 커맨드라인 도구를 제공합니다:

- **sage-crypto**: 키 관리 및 암호화 작업
- **sage-did**: 분산 식별자(DID) 관리

## 설치

### 소스에서 설치

```bash
# 저장소 클론
git clone https://github.com/sage-x-project/sage.git
cd sage

# 빌드 및 설치
make build
make install  # $GOPATH/bin에 설치
```

### 사전 빌드된 바이너리 사용

GitHub 릴리스 페이지에서 최신 릴리스를 다운로드하세요.

## sage-crypto

`sage-crypto` 도구는 포괄적인 키 관리 및 암호화 작업을 제공합니다.

### 명령어

#### generate - 새 키 쌍 생성

```bash
# Ed25519 키를 JWK로 생성
sage-crypto generate --type ed25519 --format jwk

# Secp256k1 키를 생성하고 파일에 저장
sage-crypto generate --type secp256k1 --format pem --output mykey.pem

# 키 저장소에 생성하고 저장
sage-crypto generate --type ed25519 --format storage --storage-dir ./keys --key-id mykey
```

**옵션:**
- `--type, -t`: 키 타입 (ed25519, secp256k1)
- `--format, -f`: 출력 형식 (jwk, pem, storage)
- `--output, -o`: 출력 파일 경로
- `--storage-dir, -s`: storage 형식용 저장소 디렉토리
- `--key-id, -k`: storage 형식용 키 ID

#### sign - 메시지 서명

```bash
# JWK 키 파일로 서명
sage-crypto sign --key mykey.jwk --message "Hello, World!"

# 파일 내용 서명
sage-crypto sign --key mykey.pem --format pem --message-file document.txt

# 저장된 키로 서명
sage-crypto sign --storage-dir ./keys --key-id mykey --message "Test"

# base64 서명만 출력
sage-crypto sign --key mykey.jwk --message "Hello" --base64
```

**옵션:**
- `--key`: 키 파일 경로
- `--key-format`: 키 파일 형식 (jwk, pem)
- `--storage-dir, -s`: 저장소 디렉토리
- `--key-id, -k`: 저장소의 키 ID
- `--message, -m`: 서명할 메시지
- `--message-file`: 메시지가 포함된 파일
- `--output, -o`: 서명 출력 파일
- `--base64`: base64로만 서명 출력

#### verify - 서명 검증

```bash
# base64 서명으로 검증
sage-crypto verify --key public.jwk --message "Hello, World!" --signature-b64 "base64sig..."

# 서명 파일로 검증
sage-crypto verify --key mykey.pem --format pem --message-file document.txt --signature-file sig.json
```

**옵션:**
- `--key`: 공개 키 파일 (필수)
- `--key-format`: 키 형식 (jwk, pem)
- `--message, -m`: 검증할 메시지
- `--message-file`: 메시지가 포함된 파일
- `--signature-b64`: Base64 인코딩된 서명
- `--signature-file`: 서명 파일

#### list - 저장소의 키 목록 조회

```bash
sage-crypto list --storage-dir ./keys
```

**옵션:**
- `--storage-dir, -s`: 저장소 디렉토리 (필수)

#### rotate - 키 회전

```bash
sage-crypto rotate --storage-dir ./keys --key-id mykey
```

**옵션:**
- `--storage-dir, -s`: 저장소 디렉토리 (필수)
- `--key-id, -k`: 회전할 키 ID (필수)
- `--keep-old`: 이전 키 보관

#### address - 블록체인 주소 생성

```bash
# 키에서 이더리움 주소 생성
sage-crypto address generate --key mykey.pem --format pem --chain ethereum

# 솔라나 주소 생성
sage-crypto address generate --storage-dir ./keys --key-id mykey --chain solana

# 주소 파싱 및 검증
sage-crypto address parse 0x742d35Cc6634C0532925a3b844Bc9e7595f7F1a
```

**옵션:**
- `--key`: 키 파일 경로
- `--key-format`: 키 형식 (jwk, pem)
- `--storage-dir, -s`: 저장소 디렉토리
- `--key-id, -k`: 저장소의 키 ID
- `--chain, -c`: 블록체인 (ethereum, solana)
- `--all`: 모든 호환 체인의 주소 생성

### 예제

#### 전체 워크플로우 예제

```bash
# 1. 새 Ed25519 키 쌍 생성
sage-crypto generate --type ed25519 --format jwk --output alice.jwk

# 2. 메시지 서명
MESSAGE="안녕하세요, SAGE!"
SIGNATURE=$(sage-crypto sign --key alice.jwk --message "$MESSAGE" --base64)
echo "서명: $SIGNATURE"

# 3. 서명 검증
sage-crypto verify --key alice.jwk --message "$MESSAGE" --signature-b64 "$SIGNATURE"
# 출력: ✅ Signature verification PASSED

# 4. 잘못된 메시지로 시도 (실패해야 함)
sage-crypto verify --key alice.jwk --message "잘못된 메시지" --signature-b64 "$SIGNATURE"
# 출력: ❌ Signature verification FAILED
```

#### 키 저장소 예제

```bash
# 저장소 디렉토리 생성
mkdir -p ~/.sage/keys

# 여러 키 생성 및 저장
sage-crypto generate --type ed25519 --format storage \
  --storage-dir ~/.sage/keys --key-id signing-key

sage-crypto generate --type secp256k1 --format storage \
  --storage-dir ~/.sage/keys --key-id ethereum-key

# 모든 키 목록 조회
sage-crypto list --storage-dir ~/.sage/keys

# 저장된 키로 서명
sage-crypto sign --storage-dir ~/.sage/keys --key-id signing-key \
  --message "저장된 키로 서명됨"

# 키 회전
sage-crypto rotate --storage-dir ~/.sage/keys --key-id signing-key --keep-old
```

#### 블록체인 주소 생성

```bash
# Secp256k1 키 생성 (이더리움용)
sage-crypto generate --type secp256k1 --format storage \
  --storage-dir ~/.sage/keys --key-id eth-key

# 이더리움 주소 생성
sage-crypto address generate --storage-dir ~/.sage/keys --key-id eth-key \
  --chain ethereum

# Ed25519 키 생성 (솔라나용)
sage-crypto generate --type ed25519 --format storage \
  --storage-dir ~/.sage/keys --key-id sol-key

# 솔라나 주소 생성
sage-crypto address generate --storage-dir ~/.sage/keys --key-id sol-key \
  --chain solana

# 모든 호환 체인의 주소 생성
sage-crypto address generate --storage-dir ~/.sage/keys --key-id eth-key --all
```

## sage-did

`sage-did` 도구는 블록체인상의 AI 에이전트를 위한 분산 식별자를 관리합니다.

### 명령어

#### register - 새 AI 에이전트 등록

```bash
# 이더리움에 등록
sage-did register --chain ethereum --name "나의 AI 에이전트" \
  --endpoint "https://api.myagent.com" \
  --key ethereum-key.pem --format pem \
  --description "코드 리뷰를 위한 AI 보조"

# 기능과 함께 솔라나에 등록
sage-did register --chain solana --name "거래 봇" \
  --endpoint "https://bot.example.com" \
  --storage-dir ~/.sage/keys --key-id bot-key \
  --capabilities '{"trading": true, "analysis": true}'
```

**옵션:**
- `--chain, -c`: 블록체인 (ethereum, solana) [필수]
- `--name, -n`: 에이전트 이름 [필수]
- `--endpoint`: 에이전트 API 엔드포인트 [필수]
- `--description, -d`: 에이전트 설명
- `--capabilities`: 에이전트 기능 (JSON)
- `--key, -k`: 키 파일 경로
- `--key-format`: 키 형식 (jwk, pem)
- `--storage-dir`: 키 저장소 디렉토리
- `--key-id`: 저장소의 키 ID
- `--rpc`: 블록체인 RPC 엔드포인트
- `--contract`: 레지스트리 컨트랙트 주소
- `--private-key`: 트랜잭션 서명자 개인 키

#### resolve - 에이전트 DID 조회

```bash
# 에이전트 메타데이터 조회
sage-did resolve did:sage:ethereum:agent_12345

# 파일에 저장
sage-did resolve did:sage:solana:bot_abc --output agent-info.json

# 텍스트 형식 출력
sage-did resolve did:sage:ethereum:agent_12345 --format text
```

**옵션:**
- `--rpc`: 블록체인 RPC 엔드포인트
- `--contract`: 레지스트리 컨트랙트 주소
- `--output, -o`: 출력 파일 경로
- `--format`: 출력 형식 (json, text)

#### list - 소유자별 에이전트 목록

```bash
# 주소가 소유한 모든 에이전트 목록
sage-did list --chain ethereum --owner 0x742d35Cc6634C0532925a3b844Bc9e7595f7F1a

# 사용자 정의 RPC와 함께
sage-did list --chain solana --owner AgentOwnerPubkey... \
  --rpc https://api.devnet.solana.com
```

**옵션:**
- `--chain, -c`: 블록체인 (ethereum, solana) [필수]
- `--owner`: 소유자 주소 [필수]
- `--rpc`: 블록체인 RPC 엔드포인트
- `--contract`: 레지스트리 컨트랙트 주소

#### update - 에이전트 메타데이터 업데이트

```bash
# 엔드포인트 업데이트
sage-did update did:sage:ethereum:agent_12345 \
  --endpoint "https://new-api.myagent.com" \
  --key owner-key.pem --format pem

# 기능 업데이트
sage-did update did:sage:solana:bot_abc \
  --capabilities '{"trading": true, "analysis": true, "reporting": true}' \
  --storage-dir ~/.sage/keys --key-id owner-key
```

**옵션:**
- `--endpoint`: 새 엔드포인트 URL
- `--description`: 새 설명
- `--capabilities`: 새 기능 (JSON)
- `--key`: 소유자 키 파일
- `--key-format`: 키 형식
- `--storage-dir`: 키 저장소 디렉토리
- `--key-id`: 저장소의 키 ID
- `--rpc`: 블록체인 RPC 엔드포인트
- `--contract`: 레지스트리 컨트랙트 주소

#### deactivate - 에이전트 비활성화

```bash
sage-did deactivate did:sage:ethereum:agent_12345 \
  --key owner-key.pem --format pem
```

**옵션:**
- `--key`: 소유자 키 파일 [필수]
- `--key-format`: 키 형식
- `--storage-dir`: 키 저장소 디렉토리
- `--key-id`: 저장소의 키 ID
- `--rpc`: 블록체인 RPC 엔드포인트
- `--contract`: 레지스트리 컨트랙트 주소

#### verify - 에이전트 메타데이터 검증

```bash
# 에이전트 활성 상태 및 엔드포인트 접근성 확인
sage-did verify did:sage:ethereum:agent_12345

# 엔드포인트 검사 건너뛰기
sage-did verify did:sage:solana:bot_abc --skip-endpoint
```

**옵션:**
- `--rpc`: 블록체인 RPC 엔드포인트
- `--contract`: 레지스트리 컨트랙트 주소
- `--skip-endpoint`: 엔드포인트 접근성 검사 건너뛰기

### 예제

#### 전체 에이전트 등록 워크플로우

```bash
# 1. 이더리움용 키 생성
sage-crypto generate --type secp256k1 --format storage \
  --storage-dir ~/.sage/keys --key-id agent-key

# 2. 이더리움 주소 확인
AGENT_ADDR=$(sage-crypto address generate --storage-dir ~/.sage/keys \
  --key-id agent-key --chain ethereum)
echo "에이전트 주소: $AGENT_ADDR"

# 3. 에이전트 등록 (주소에 자금 충전 후)
sage-did register --chain ethereum \
  --name "코드 리뷰 보조" \
  --description "자동 코드 리뷰를 위한 AI 에이전트" \
  --endpoint "https://api.codereview-bot.com" \
  --capabilities '{"review": true, "suggest": true, "lint": true}' \
  --storage-dir ~/.sage/keys --key-id agent-key \
  --private-key $DEPLOYER_PRIVATE_KEY

# 4. 등록 확인을 위해 조회
sage-did resolve did:sage:ethereum:agent_$AGENT_ADDR

# 5. 메타데이터 업데이트
sage-did update did:sage:ethereum:agent_$AGENT_ADDR \
  --description "향상된 코드 리뷰 AI 에이전트 v2" \
  --capabilities '{"review": true, "suggest": true, "lint": true, "security": true}' \
  --storage-dir ~/.sage/keys --key-id agent-key

# 6. 에이전트 상태 검증
sage-did verify did:sage:ethereum:agent_$AGENT_ADDR
```

## 환경 변수

두 도구 모두 다음 환경 변수를 지원합니다:

```bash
# 기본 저장소 디렉토리
export SAGE_KEY_STORAGE="$HOME/.sage/keys"

# 기본 RPC 엔드포인트
export SAGE_ETH_RPC="https://eth-mainnet.g.alchemy.com/v2/your-key"
export SAGE_SOL_RPC="https://api.mainnet-beta.solana.com"

# 기본 컨트랙트 주소
export SAGE_ETH_CONTRACT="0x..."
export SAGE_SOL_CONTRACT="..."
```

## 보안 모범 사례

1. **키 저장소**: 프로덕션에서는 항상 암호화된 저장소나 하드웨어 보안 모듈을 사용하세요
2. **개인 키**: 개인 키를 절대 공유하거나 커밋하지 마세요
3. **RPC 엔드포인트**: 프로덕션에서는 인증된 RPC 엔드포인트를 사용하세요
4. **권한**: 키 파일에 적절한 파일 권한(0600)을 설정하세요
5. **백업**: 키 저장소 디렉토리를 정기적으로 백업하세요

## 문제 해결

### 일반적인 문제

1. **"Key not found" 오류**
   - 키 파일 경로 또는 저장소 디렉토리 확인
   - 키 ID가 일치하는지 확인

2. **"Invalid signature" 오류**
   - 메시지가 정확히 일치하는지 확인 (공백 포함)
   - 올바른 키를 사용하고 있는지 확인

3. **블록체인 작업에서 "Connection refused"**
   - RPC 엔드포인트가 접근 가능한지 확인
   - 네트워크 연결 확인

4. **DID 등록에서 "Insufficient funds"**
   - 계정에 가스 수수료를 위한 충분한 네이티브 토큰이 있는지 확인

### 디버그 모드

`SAGE_DEBUG` 환경 변수로 디버그 출력 활성화:

```bash
SAGE_DEBUG=1 sage-crypto sign --key mykey.jwk --message "test"
```

## 도움말

```bash
# 일반 도움말
sage-crypto --help
sage-did --help

# 명령별 도움말
sage-crypto sign --help
sage-did register --help
```

## 추가 예제

### 멀티체인 에이전트 관리

```bash
# 1. 두 체인용 키 생성
sage-crypto generate --type secp256k1 --format storage \
  --storage-dir ~/.sage/keys --key-id eth-agent-key

sage-crypto generate --type ed25519 --format storage \
  --storage-dir ~/.sage/keys --key-id sol-agent-key

# 2. 이더리움에 에이전트 등록
sage-did register --chain ethereum \
  --name "크로스체인 AI 에이전트" \
  --endpoint "https://api.multichain-agent.com" \
  --capabilities '{"crosschain": true, "bridge": true}' \
  --storage-dir ~/.sage/keys --key-id eth-agent-key

# 3. 솔라나에 동일 에이전트 등록
sage-did register --chain solana \
  --name "크로스체인 AI 에이전트" \
  --endpoint "https://api.multichain-agent.com" \
  --capabilities '{"crosschain": true, "bridge": true}' \
  --storage-dir ~/.sage/keys --key-id sol-agent-key

# 4. 두 체인에서 에이전트 검색
sage-did list --chain ethereum --owner $ETH_OWNER_ADDR
sage-did list --chain solana --owner $SOL_OWNER_ADDR
```

### 백업 및 복원

```bash
# 키 저장소 백업
tar -czf sage-keys-backup-$(date +%Y%m%d).tar.gz ~/.sage/keys

# 키 저장소 복원
tar -xzf sage-keys-backup-20240107.tar.gz -C ~/

# 개별 키 내보내기
sage-crypto generate --type ed25519 --format storage \
  --storage-dir ~/.sage/keys --key-id important-key

# JWK로 내보내기 (백업용)
sage-crypto sign --storage-dir ~/.sage/keys --key-id important-key \
  --message "" --output important-key-backup.jwk
```

문제나 기능 요청은 다음을 방문하세요: https://github.com/sage-x-project/sage/issues