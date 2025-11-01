# SAGE Contracts Quick Start Guide

## 빠른 배포 가이드

### 1. 로컬 개발 환경 설정

```bash
# 1. 의존성 설치
cd contracts/ethereum
npm install

# 2. 로컬 노드 실행 (새 터미널)
npm run node

# 3. 모든 컨트랙트 배포
npm run deploy:all
```

### 2. 배포 결과 확인

배포가 완료되면 다음과 같은 출력을 볼 수 있습니다:

```
================================================================================
 SAGE Complete Contract Deployment
================================================================================

 Network: localhost (Chain ID: 31337)
 Deployment ID: localhost
 Deployer: 0xf39Fd6e51aad88F6F4ce6aB8827279cffFb92266
 Balance: 10000.0 ETH
================================================================================

 [PHASE 1] Deploying Governance Contracts
--------------------------------------------------------------------------------

 [1/7] Deploying SimpleMultiSig...
    Address: 0x5FbDB2315678afecb367f032d93F642f64180aa3
    Gas: 653000
    Block: 1

 [2/7] Deploying TEEKeyRegistry...
    Address: 0xe7f1725E7734CE288F8367e1Bb143E90bb3F0512
    Gas: 2800000
    Block: 2

 [PHASE 2] Deploying ERC-8004 Registries
--------------------------------------------------------------------------------

 [3/7] Deploying ERC8004IdentityRegistry...
    Address: 0x9fE46736679d2D9a65F0992F2272dE9f3c7fa6e0
    Gas: 750000
    Block: 3

 [4/7] Deploying ERC8004ReputationRegistry...
    Address: 0xCf7Ed3AccA5a467e9e704C703E8D87F634fB0Fc9
    Gas: 1200000
    Block: 4

 [5/7] Deploying ERC8004ValidationRegistry...
    Address: 0xDc64a140Aa3E981100a9becA4E685f962f0cF6C9
    Gas: 2400000
    Block: 5

 [PHASE 3] Deploying AgentCard System
--------------------------------------------------------------------------------

 [6/7] Deploying AgentCardVerifyHook...
    Address: 0x5FC8d32690cc91D4c39d9d3abcBD16989F875707
    Gas: 450000
    Block: 6

 [7/7] Deploying AgentCardRegistry...
    Address: 0x0165878A594ca255338adfa4d48449f69242Eb8F
    Gas: 2600000
    Block: 7

================================================================================
 Deployment Complete!
================================================================================

 Governance Contracts:
   SimpleMultiSig:              0x5FbDB2315678afecb367f032d93F642f64180aa3
   TEEKeyRegistry:              0xe7f1725E7734CE288F8367e1Bb143E90bb3F0512

 ERC-8004 Registries:
   IdentityRegistry:            0x9fE46736679d2D9a65F0992F2272dE9f3c7fa6e0
   ReputationRegistry:          0xCf7Ed3AccA5a467e9e704C703E8D87F634fB0Fc9
   ValidationRegistry:          0xDc64a140Aa3E981100a9becA4E685f962f0cF6C9

 AgentCard System:
   AgentCardRegistry:           0x0165878A594ca255338adfa4d48449f69242Eb8F
   AgentCardVerifyHook:         0x5FC8d32690cc91D4c39d9d3abcBD16989F875707

 Total Gas Used:                10853000

 Deployment Files:
   Timestamped: deployments/localhost-complete-1234567890.json
   Latest:      deployments/localhost-complete-latest.json

 Next Steps:
   1. Verify contracts on block explorer
   2. Run integration tests
   3. Configure contract interactions
```

### 3. 배포된 컨트랙트 주소 확인

```bash
cat contracts/ethereum/deployments/localhost-complete-latest.json
```

## 주요 명령어

### 로컬 노드 관리

```bash
# 로컬 노드 실행
npm run node

# 로컬 노드 중지
npm run node:stop

# 로컬 노드 재시작
npm run node:restart

# 포트 상태 확인
npm run node:status
```

### 컨트랙트 배포

```bash
# 모든 컨트랙트 배포 (권장)
npm run deploy:all

# AgentCard만 배포
npm run deploy:localhost

# 테스트넷 배포
npm run deploy:all:ethereum:sepolia  # Ethereum Sepolia
npm run deploy:all:kaia:kairos       # Kaia Kairos
npm run deploy:all:bsc:testnet       # BSC Testnet
npm run deploy:all:base:sepolia      # Base Sepolia
npm run deploy:all:arbitrum:sepolia  # Arbitrum Sepolia
npm run deploy:all:optimism:sepolia  # Optimism Sepolia

# 메인넷 배포
npm run deploy:all:ethereum:mainnet  # Ethereum Mainnet
npm run deploy:all:kaia:mainnet      # Kaia Mainnet
npm run deploy:all:bsc:mainnet       # BSC Mainnet
npm run deploy:all:base:mainnet      # Base Mainnet
npm run deploy:all:arbitrum:mainnet  # Arbitrum Mainnet
npm run deploy:all:optimism:mainnet  # Optimism Mainnet
```

### 테스트

```bash
# 모든 테스트 실행
npm test

# 특정 테스트 파일 실행
npx hardhat test test/AgentCardRegistry.test.js --network localhost

# 커버리지 테스트
npm run coverage
```

## 배포된 컨트랙트 사용 예제

### ERC8004IdentityRegistry - Agent 등록

```javascript
const { ethers } = require("hardhat");

async function registerAgent() {
  // 배포 정보 로드
  const deployment = require('./deployments/localhost-complete-latest.json');

  // 컨트랙트 인스턴스 가져오기
  const IdentityRegistry = await ethers.getContractAt(
    'ERC8004IdentityRegistry',
    deployment.contracts.ERC8004IdentityRegistry.address
  );

  // Agent 등록
  const agentDID = "did:sage:ethereum:0x1234...";
  const publicKey = "0x04abcd..."; // 65 bytes uncompressed public key
  const metadata = "ipfs://Qm...";

  const tx = await IdentityRegistry.registerAgent(
    agentDID,
    publicKey,
    metadata
  );

  await tx.wait();
  console.log("Agent registered:", agentDID);

  // Agent 조회
  const agent = await IdentityRegistry.getAgent(agentDID);
  console.log("Agent info:", agent);
}

registerAgent().catch(console.error);
```

### ERC8004ValidationRegistry - 검증 요청

```javascript
async function createValidationRequest() {
  const deployment = require('./deployments/localhost-complete-latest.json');

  const ValidationRegistry = await ethers.getContractAt(
    'ERC8004ValidationRegistry',
    deployment.contracts.ERC8004ValidationRegistry.address
  );

  // 검증 요청 생성
  const agentDID = "did:sage:ethereum:0x1234...";
  const validationType = 1; // Identity validation
  const payload = ethers.keccak256(ethers.toUtf8Bytes("validate this"));

  const tx = await ValidationRegistry.createValidationRequest(
    agentDID,
    validationType,
    payload,
    { value: ethers.parseEther("0.1") } // 보상 풀
  );

  const receipt = await tx.wait();
  const requestId = receipt.logs[0].topics[1]; // RequestCreated event

  console.log("Validation request created:", requestId);
}
```

### AgentCardRegistry - Commit-Reveal 등록

```javascript
async function registerWithCommitReveal() {
  const deployment = require('./deployments/localhost-complete-latest.json');

  const AgentCardRegistry = await ethers.getContractAt(
    'AgentCardRegistry',
    deployment.contracts.AgentCardRegistry.address
  );

  const agentDID = "did:sage:ethereum:0x1234...";
  const metadataHash = ethers.keccak256(ethers.toUtf8Bytes("metadata"));
  const keyTypes = ["ECDSA", "Ed25519"];
  const publicKeys = [
    "0x04abcd...", // ECDSA public key
    "0x1234..."   // Ed25519 public key
  ];
  const secret = ethers.randomBytes(32);

  // Step 1: Commit
  const commitment = ethers.keccak256(
    ethers.AbiCoder.defaultAbiCoder().encode(
      ["string", "bytes32", "string[]", "bytes[]", "bytes32"],
      [agentDID, metadataHash, keyTypes, publicKeys, secret]
    )
  );

  const commitTx = await AgentCardRegistry.commitAgentRegistration(commitment);
  await commitTx.wait();
  console.log("Committed");

  // Wait for block confirmation
  await new Promise(resolve => setTimeout(resolve, 2000));

  // Step 2: Reveal
  const revealTx = await AgentCardRegistry.revealAgentRegistration(
    agentDID,
    metadataHash,
    keyTypes,
    publicKeys,
    secret
  );
  await revealTx.wait();
  console.log("Agent registered:", agentDID);
}
```

## Hardhat Console 사용

```bash
npx hardhat console --network localhost
```

Console에서:

```javascript
// 배포 정보 로드
const deployment = require('./deployments/localhost-complete-latest.json');

// 컨트랙트 가져오기
const IdentityRegistry = await ethers.getContractAt(
  'ERC8004IdentityRegistry',
  deployment.contracts.ERC8004IdentityRegistry.address
);

// 상호작용
const agentDID = "did:sage:test:123";
const tx = await IdentityRegistry.registerAgent(
  agentDID,
  "0x04" + "ab".repeat(64),
  "metadata"
);
await tx.wait();

// 조회
const agent = await IdentityRegistry.getAgent(agentDID);
console.log(agent);
```

## 트러블슈팅

### 문제: "insufficient funds" 오류

**해결책**: 배포자 계정 잔액 확인
```bash
npm run check-balance
```

### 문제: "Nonce too high" 오류

**해결책**: Hardhat 노드 재시작
```bash
npm run node:restart
```

### 문제: 포트 8545가 이미 사용 중

**해결책**: 포트 정리
```bash
npm run port:clean
npm run node
```

### 문제: 컨트랙트 배포 실패

**해결책**: 컴파일 후 재배포
```bash
npm run clean
npm run compile
npm run deploy:all
```

## 더 자세한 정보

상세한 사용 방법은 다음 문서를 참고하세요:

- **전체 배포 가이드**: [docs/DEPLOYMENT_GUIDE.md](docs/DEPLOYMENT_GUIDE.md)
- **ERC-8004 명세**: [contracts/erc-8004/standalone/README.md](contracts/erc-8004/standalone/README.md)
- **Hardhat 설정**: [hardhat.config.js](hardhat.config.js)

## 지원

문제가 발생하면:
- GitHub Issues: https://github.com/sage-x-project/sage/issues
- Documentation: https://docs.sage-x-project.org
