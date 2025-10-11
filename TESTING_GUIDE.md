# SAGE Testing Guide

## Skip되지 않도록 테스트 환경 구성하기

이 가이드는 SAGE 프로젝트의 모든 테스트를 Skip 없이 실행하기 위한 환경 구성 방법을 설명합니다.

## 1. Integration 테스트 실행하기

### Short Mode Skip 비활성화
Integration 테스트는 기본적으로 `go test -short` 모드에서 Skip됩니다.

**모든 테스트 실행하기:**
```bash
# Short flag 없이 실행 (전체 테스트)
go test ./...

# 특정 패키지의 Integration 테스트 실행
go test ./tests/integration/...
```

**Skip되는 테스트:**
- `TestDIDRegistration`
- `TestDIDLifecycle`
- `TestDIDVerification`
- `TestBlockchainConnection`
- `TestSmartContractInteraction`
- `TestMultiAgentScenario`

## 2. Ethereum/블록체인 테스트 환경 구성

### 로컬 Ethereum 노드 실행

**Hardhat 노드 시작:**
```bash
cd sage/contracts/ethereum
npm install
npx hardhat node

# 별도 터미널에서 컨트랙트 배포
npx hardhat deploy --network localhost
```

**환경변수 설정:**
```bash
export ETHEREUM_RPC_URL=http://localhost:8545
export ETHEREUM_CONTRACT_ADDRESS=<배포된_컨트랙트_주소>
export ETHEREUM_PRIVATE_KEY=<테스트용_프라이빗_키>
```

**관련 테스트:**
- `sage/did/ethereum/client_test.go` - Ethereum 클라이언트 테스트
- `sage/tests/integration/blockchain_test.go` - 블록체인 통합 테스트

## 3. Auth0 테스트 환경 구성

### Auth0 계정 설정

1. Auth0 계정이 없다면 [Auth0](https://auth0.com)에서 무료 계정 생성
2. 2개의 Application 생성 (Agent1, Agent2용)
3. 각 Application에서 Machine-to-Machine 타입 선택

### 환경 파일 구성

**`.env` 파일 생성:**
```bash
cd sage
cp .env.example .env
```

**`.env` 파일 편집:**
```env
# Agent 1 Configuration
AUTH0_DOMAIN_1=your-tenant.auth0.com
AUTH0_CLIENT_ID_1=your-client-id-1
AUTH0_CLIENT_SECRET_1=your-client-secret-1
TEST_DID_1=did:sage:agent1
IDENTIFIER_1=https://api.example.com/agent1
AUTH0_KEY_ID_1=key-1

# Agent 2 Configuration
AUTH0_DOMAIN_2=your-tenant.auth0.com
AUTH0_CLIENT_ID_2=your-client-id-2
AUTH0_CLIENT_SECRET_2=your-client-secret-2
TEST_DID_2=did:sage:agent2
IDENTIFIER_2=https://api.example.com/agent2
AUTH0_KEY_ID_2=key-2

# Token TTL 테스트용 (선택사항)
TEST_API_TOKEN_TTL_SECONDS=60
```

**관련 테스트:**
- `sage/oidc/auth0/auth0_integration_test.go`

## 4. 모든 테스트 실행 명령어

### 전체 테스트 실행 (Skip 없이)

```bash
# 1. 블록체인 노드 시작 (터미널 1)
cd sage/contracts/ethereum
npx hardhat node

# 2. 컨트랙트 배포 (터미널 2)
cd sage/contracts/ethereum
npx hardhat deploy --network localhost

# 3. 환경변수 설정 및 테스트 실행 (터미널 3)
cd sage
cp .env.example .env
# .env 파일 편집 (Auth0 정보 입력)

# 환경변수 export
export ETHEREUM_RPC_URL=http://localhost:8545
export TEST_API_TOKEN_TTL_SECONDS=60

# 전체 테스트 실행
go test ./... -v

# Integration 테스트만 실행
go test ./tests/integration/... -v -tags=integration

# 특정 테스트 함수 실행
go test -v -run TestDIDRegistration ./tests/integration/
```

### 스마트 컨트랙트 테스트

```bash
cd contracts/ethereum

# 전체 테스트
npm test

# 커버리지 포함
npm run coverage

# 특정 테스트
npm run test:v2
```

## 5. CI/CD에서 테스트 실행

### GitHub Actions 예시

```yaml
name: Full Test Suite

on: [push, pull_request]

jobs:
  test:
    runs-on: ubuntu-latest

    services:
      ethereum:
        image: ethereum/client-go:stable
        ports:
          - 8545:8545
        options: --dev --http --http.addr 0.0.0.0

    steps:
    - uses: actions/checkout@v3

    - name: Setup Go
      uses: actions/setup-go@v4
      with:
        go-version: '1.21'

    - name: Setup Node
      uses: actions/setup-node@v3
      with:
        node-version: '18'

    - name: Install dependencies
      run: |
        go mod download
        cd contracts/ethereum && npm install

    - name: Deploy contracts
      run: |
        cd contracts/ethereum
        npx hardhat deploy --network localhost

    - name: Run all tests
      env:
        ETHEREUM_RPC_URL: http://localhost:8545
        AUTH0_DOMAIN_1: ${{ secrets.AUTH0_DOMAIN_1 }}
        AUTH0_CLIENT_ID_1: ${{ secrets.AUTH0_CLIENT_ID_1 }}
        AUTH0_CLIENT_SECRET_1: ${{ secrets.AUTH0_CLIENT_SECRET_1 }}
        # ... 추가 환경변수
      run: |
        go test ./... -v -cover
```

## 6. 테스트 Skip 상태 확인

### Skip되는 테스트 목록 보기

```bash
# Skip 메시지가 있는 테스트 찾기
go test ./... -v 2>&1 | grep -i skip

# 특정 패키지에서 Skip 확인
go test ./tests/integration/... -v 2>&1 | grep -i skip
```

### Skip 이유별 분류

| Skip 이유 | 환경 구성 방법 |
|---------|---------------|
| `testing.Short()` | `-short` 플래그 없이 실행 |
| 블록체인 노드 없음 | Hardhat 노드 실행 |
| `.env` 파일 없음 | `.env` 파일 생성 및 설정 |
| 환경변수 없음 | 필요한 환경변수 export |

## 7. 트러블슈팅

### 문제: Ethereum 노드 연결 실패
```bash
# 해결방법
npx hardhat node --hostname 0.0.0.0  # 모든 인터페이스에서 접근 가능
export ETHEREUM_RPC_URL=http://127.0.0.1:8545
```

### 문제: Auth0 인증 실패
```bash
# Auth0 도메인 확인 (https:// 제외)
export AUTH0_DOMAIN_1=your-tenant.auth0.com  #  올바른 형식
# export AUTH0_DOMAIN_1=https://your-tenant.auth0.com  #  잘못된 형식
```

### 문제: 환경변수가 테스트에서 인식되지 않음
```bash
# 해결방법: godotenv 사용 또는 직접 export
source .env  # bash/zsh
go test ./...
```

## 8. 성능 고려사항

Integration 테스트는 시간이 오래 걸릴 수 있습니다:
- 블록체인 테스트: 블록 생성 대기 시간
- Auth0 테스트: 네트워크 지연
- 전체 테스트 실행 시간: 약 2-5분

**빠른 테스트를 위한 팁:**
```bash
# Unit 테스트만 실행
go test -short ./...

# 병렬 실행
go test -parallel 4 ./...

# 특정 패키지만 테스트
go test ./crypto/...
```

---

이 가이드를 따르면 SAGE 프로젝트의 모든 테스트를 Skip 없이 실행할 수 있습니다.