# SAGE Contracts Local Testing Guide

**Version**: 2.0 (AgentCard Architecture)
**Last Updated**: 2025-11-01

로컬 환경에서 SAGE 컨트랙트를 배포하고 테스트하는 방법입니다.

##  Quick Start

가장 빠르게 시작하는 방법:

```bash
# 1. 전체 설정 (노드 시작 + 컨트랙트 배포)
./bin/deploy-local.sh
# 옵션 6 선택

# 2. 배포된 컨트랙트와 상호작용
./bin/deploy-local.sh
# 옵션 3 선택
```

##  Prerequisites

- Node.js v18+ 
- npm or yarn
- 의존성 설치: `npm install`

##  Step-by-Step Guide

### 1. Hardhat 로컬 노드 시작

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

### 2. 컨트랙트 배포

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
- **AgentCardRegistry**: Multi-key 지원 및 ERC-8004 네이티브 구현
- **AgentCardStorage**: 분리된 스토리지 레이어
- **AgentCardVerifyHook**: DID 검증, rate limiting, blacklist 기능
- **ERC8004ValidationRegistry**: Task 검증 레지스트리
- **ERC8004ReputationRegistryV2**: Reputation 관리

### 3. 컨트랙트와 상호작용

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

### 4. 자동화된 테스트 시나리오

```bash
./bin/deploy-local.sh
# 옵션 4 선택
```

이렇게 하면 전체 통합 테스트가 실행됩니다.

##  Example: Agent Registration (Multi-Key)

```javascript
// 1. Multi-key 데이터 준비
const ecdsaWallet = ethers.Wallet.createRandom();
const ed25519Key = ethers.hexlify(ethers.randomBytes(32)); // 실제로는 proper Ed25519 lib 사용
const x25519Key = ethers.hexlify(ethers.randomBytes(32));   // KME public key for HPKE

const keys = [
  ecdsaWallet.publicKey,  // ECDSA (65 bytes uncompressed)
  ed25519Key,             // Ed25519 (32 bytes)
  x25519Key,              // X25519 (32 bytes)
];

const keyTypes = [0, 1, 2]; // ECDSA, Ed25519, X25519

// 2. Commit 생성
const salt = ethers.hexlify(ethers.randomBytes(32));
const network = await provider.getNetwork();
const commitHash = ethers.keccak256(
  ethers.AbiCoder.defaultAbiCoder().encode(
    ["string", "bytes[]", "address", "bytes32", "uint256"],
    [did, keys, ecdsaWallet.address, salt, network.chainId]
  )
);

// 3. Commit 제출 (0.01 ETH stake)
await registry.commitRegistration(commitHash, {
  value: ethers.parseEther("0.01")
});

// 4. 60초 대기 후 Reveal
await new Promise(resolve => setTimeout(resolve, 61000));

// 5. 서명 생성 (ECDSA만)
const messageHash = ethers.keccak256(
  ethers.AbiCoder.defaultAbiCoder().encode(
    ["string", "uint256", "address", "address"],
    ["SAGE Agent Registration:", network.chainId, registryAddress, ecdsaWallet.address]
  )
);
const ecdsaSignature = await ecdsaWallet.signMessage(ethers.getBytes(messageHash));

// 6. Registration params 준비
const params = {
  did: "did:sage:ethereum:0x123...",
  name: "My AI Assistant",
  description: "A helpful AI agent",
  endpoint: "https://myagent.ai",
  capabilities: JSON.stringify(["chat", "code", "analysis"]),
  keys,
  keyTypes,
  signatures: [ecdsaSignature, ethers.hexlify(ethers.randomBytes(64)), "0x"],
  salt
};

// 7. 에이전트 등록
const tx = await registry.registerAgent(params);
await tx.wait();

// 8. 1시간 후 활성화
setTimeout(async () => {
  await registry.activateAgent(agentId);
}, 3600000);
```

##  Testing Features

### AgentCard 아키텍처 주요 기능:

1. **Multi-Key 지원**
   - ECDSA (secp256k1): Ethereum 호환 서명
   - Ed25519: 고성능 EdDSA 서명
   - X25519: HPKE 암호화 (KME public key)
   - 최대 10개 키 등록 가능

2. **강화된 보안**
   - Commit-reveal 패턴 (front-running 방지)
   - 0.01 ETH stake 요구사항
   - 1시간 time-lock 활성화
   - Rate limiting (하루 24개 제한)
   - Public key 재사용 방지
   - Blacklist/Whitelist

3. **ERC-8004 네이티브 구현**
   - Identity Registry (native)
   - Validation Registry
   - Reputation Registry
   - AgentDomain 지원

4. **스토리지 분리**
   - AgentCardStorage: 독립된 스토리지 레이어
   - 가스 최적화
   - 업그레이드 용이성

## Troubleshooting

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

##  Gas Usage

대략적인 가스 사용량 (AgentCard):
- Commit Registration: ~50,000 gas
- Register Agent (3 keys): ~450,000-650,000 gas
- Activate Agent: ~50,000 gas
- Add Key: ~100,000 gas
- Revoke Key: ~70,000 gas
- Update Agent: ~80,000 gas
- Deactivate Agent: ~50,000 gas

##  Monitoring

로그 확인:
```bash
# Hardhat 노드 로그
tail -f hardhat-node.log

# 트랜잭션 모니터링
npx hardhat console --network localhost
> const registry = await ethers.getContractAt("AgentCardRegistry", "0x...")
> await registry.queryFilter(registry.filters.AgentRegistered())
```

## Advanced Usage

### Custom Scripts

`scripts/` 디렉토리에 커스텀 스크립트를 추가할 수 있습니다:

```javascript
// scripts/my-test.js
const hre = require("hardhat");

async function main() {
  const registry = await hre.ethers.getContractAt(
    "AgentCardRegistry",
    "0x5FbDB2315678afecb367f032d93F642f64180aa3"
  );

  // Your custom logic here
  // Example: Query agent by DID
  const agent = await registry.getAgentByDID("did:sage:ethereum:0x...");
  console.log("Agent:", agent);
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
> const registry = await ethers.getContractAt("AgentCardRegistry", "0x...")
> await registry.owner()
> await registry.registrationStake() // 0.01 ETH
> await registry.activationDelay()   // 3600 seconds (1 hour)
```

##  Next Steps

1. 로컬 테스트 완료 후 테스트넷 배포
2. Frontend 애플리케이션 연동
3. 추가 Hook 구현 (예: AI 모델 검증)
4. 성능 최적화 및 가스 효율성 개선

## Support

문제가 있으신가요?
- GitHub Issues: [프로젝트 저장소]
- Documentation: [이 가이드]
- Contract Source: `/contracts/`