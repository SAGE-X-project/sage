# 📚 SAGE Contracts Local Testing Guide

로컬 환경에서 SAGE 컨트랙트를 배포하고 테스트하는 방법입니다.

## 🚀 Quick Start

가장 빠르게 시작하는 방법:

```bash
# 1. 전체 설정 (노드 시작 + 컨트랙트 배포)
./bin/deploy-local.sh
# 옵션 6 선택

# 2. 배포된 컨트랙트와 상호작용
./bin/deploy-local.sh
# 옵션 3 선택
```

## 📋 Prerequisites

- Node.js v18+ 
- npm or yarn
- 의존성 설치: `npm install`

## 🔧 Step-by-Step Guide

### 1️⃣ Hardhat 로컬 노드 시작

```bash
./bin/deploy-local.sh
# 옵션 1 선택
```

또는 수동으로:

```bash
npx hardhat node
```

이렇게 하면:
- 로컬 블록체인이 `http://localhost:8545`에서 실행됩니다
- 10개의 테스트 계정이 생성됩니다 (각각 10,000 ETH 보유)

### 2️⃣ 컨트랙트 배포

새 터미널에서:

```bash
./bin/deploy-local.sh
# 옵션 2 선택
```

또는 수동으로:

```bash
npx hardhat run scripts/deploy-local.js --network localhost
```

배포되는 컨트랙트:
- **SageRegistryV2**: 향상된 공개키 검증 기능이 있는 메인 레지스트리
- **SageVerificationHook**: DID 검증 및 rate limiting 기능

### 3️⃣ 컨트랙트와 상호작용

```bash
./bin/deploy-local.sh
# 옵션 3 선택
```

또는 수동으로:

```bash
npx hardhat run scripts/interact-local.js --network localhost
```

상호작용 메뉴:
1. **Register Agent**: 새 AI 에이전트 등록
2. **View Agent**: 에이전트 정보 조회
3. **Update Agent**: 에이전트 메타데이터 업데이트
4. **Deactivate Agent**: 에이전트 비활성화
5. **Revoke Key**: 공개키 취소 (관련 에이전트 자동 비활성화)
6. **List Agents**: 특정 소유자의 모든 에이전트 조회
7. **Check Hooks**: Hook 설정 확인
8. **Test Signature**: 서명 생성 테스트

### 4️⃣ 자동화된 테스트 시나리오

```bash
./bin/deploy-local.sh
# 옵션 4 선택
```

이렇게 하면 전체 통합 테스트가 실행됩니다.

## 📝 Example: Agent Registration

```javascript
// 1. 에이전트 데이터 준비
const agentData = {
  did: "did:sage:test:0x123...",
  name: "My AI Assistant",
  description: "A helpful AI agent",
  endpoint: "https://myagent.ai",
  publicKey: "0x04...", // 64 bytes for uncompressed secp256k1
  capabilities: '["chat", "code", "analysis"]'
};

// 2. 서명 생성
const messageHash = ethers.keccak256(
  ethers.solidityPacked(
    ["string", "string", "string", "string", "bytes", "string", "address", "uint256"],
    [did, name, description, endpoint, publicKey, capabilities, signerAddress, 0]
  )
);
const signature = await signer.signMessage(ethers.getBytes(messageHash));

// 3. 에이전트 등록
await registry.registerAgent(
  did, name, description, endpoint, 
  publicKey, capabilities, signature
);
```

## 🧪 Testing Features

### V2 향상된 기능들:

1. **공개키 검증 (5단계)**
   - 길이 검증 (33, 64, 65 bytes)
   - 형식 검증 (0x04, 0x02, 0x03 prefix)
   - Zero key 방지
   - 소유권 증명 (서명 검증)
   - 취소된 키 확인

2. **키 취소 기능**
   - 키 소유자만 취소 가능
   - 취소 시 관련 에이전트 자동 비활성화
   - 이중 취소 방지

3. **Hook 시스템**
   - BeforeRegisterHook: 등록 전 검증
   - AfterRegisterHook: 등록 후 처리
   - DID 형식 검증
   - Rate limiting (하루 5개 제한)
   - Blacklist 기능

## 🛠️ Troubleshooting

### 문제: "Hardhat node is not running"
```bash
# 노드 시작
npx hardhat node
# 또는
./bin/deploy-local.sh 옵션 1
```

### 문제: "Invalid signature"
- nonce 값 확인 (새 등록은 0, 업데이트는 1)
- 서명할 데이터 순서 확인
- `solidityPacked` 사용 확인 (not `encode`)

### 문제: "Registration cooldown active"
- 각 등록 사이 1분 대기
- 또는 다른 계정 사용

### 문제: Port 8545 already in use
```bash
# 기존 프로세스 종료
./bin/deploy-local.sh 옵션 5
# 또는
lsof -ti:8545 | xargs kill -9
```

## 📊 Gas Usage

대략적인 가스 사용량:
- Agent Registration: ~620,000 gas
- Agent Update: ~80,000 gas
- Key Revocation: ~66,000 gas
- Agent Deactivation: ~50,000 gas

## 🔍 Monitoring

로그 확인:
```bash
# Hardhat 노드 로그
tail -f hardhat-node.log

# 트랜잭션 모니터링
npx hardhat console --network localhost
> const registry = await ethers.getContractAt("SageRegistryV2", "0x...")
> await registry.queryFilter(registry.filters.AgentRegistered())
```

## 📚 Advanced Usage

### Custom Scripts

`scripts/` 디렉토리에 커스텀 스크립트를 추가할 수 있습니다:

```javascript
// scripts/my-test.js
const hre = require("hardhat");

async function main() {
  const registry = await hre.ethers.getContractAt(
    "SageRegistryV2", 
    "0x5FbDB2315678afecb367f032d93F642f64180aa3"
  );
  
  // Your custom logic here
}

main().catch(console.error);
```

실행:
```bash
npx hardhat run scripts/my-test.js --network localhost
```

### Hardhat Console

대화형 콘솔 사용:
```bash
npx hardhat console --network localhost

> const [owner, agent1] = await ethers.getSigners()
> const registry = await ethers.getContractAt("SageRegistryV2", "0x...")
> await registry.owner()
```

## 🎯 Next Steps

1. 로컬 테스트 완료 후 테스트넷 배포
2. Frontend 애플리케이션 연동
3. 추가 Hook 구현 (예: AI 모델 검증)
4. 성능 최적화 및 가스 효율성 개선

## 📞 Support

문제가 있으신가요?
- GitHub Issues: [프로젝트 저장소]
- Documentation: [이 가이드]
- Contract Source: `/contracts/`