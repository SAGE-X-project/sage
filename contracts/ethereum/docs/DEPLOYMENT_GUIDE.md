# 🚀 SageRegistryV2 테스트 및 배포 가이드

이 가이드는 SageRegistryV2를 직접 테스트하고 배포하는 전체 과정을 단계별로 설명합니다.

## 📋 사전 준비사항

### 1. 필수 도구 설치
```bash
# Node.js 확인 (v18 권장, v16 이상 필수)
node --version

# npm 확인
npm --version

# Git 확인
git --version
```

### 2. 프로젝트 설정
```bash
# 프로젝트 디렉토리로 이동
cd /Users/0xtopaz/work/github/sage-x-project/sage/contracts/ethereum

# 의존성 설치
npm install

# 환경 변수 설정 (.env 파일 생성)
cp .env.example .env
```

### 3. 환경 변수 구성
`.env` 파일을 편집하여 필요한 값들을 설정:

```bash
# .env 파일 편집
nano .env  # 또는 원하는 편집기 사용
```

필수 환경 변수:
```env
# Kaia Testnet (Kairos) 설정
KAIROS_RPC_URL=https://public-en-kairos.node.kaia.io
PRIVATE_KEY=your_private_key_here  # 0x 없이 입력

# Kaia Mainnet (Cypress) 설정 (선택사항)
CYPRESS_RPC_URL=https://public-en-cypress.klaytn.net
# MAINNET_PRIVATE_KEY=your_mainnet_private_key  # 메인넷용

# 블록 익스플로러 API (선택사항)
KAIASCOPE_API_KEY=your_api_key_here
```

### 4. 테스트 계정 준비
```bash
# Kaia 테스트넷 Faucet에서 테스트 KLAY 받기
# https://kairos.wallet.kaia.io/faucet

# 계정 주소 확인 (Hardhat Console 사용)
npx hardhat console --network kairos
> const [deployer] = await ethers.getSigners()
> console.log("Address:", deployer.address)
> .exit
```

## 🧪 로컬 테스트

### 1. 컴파일
```bash
# 모든 컨트랙트 컴파일
npx hardhat compile

# 컴파일 확인
ls -la artifacts/contracts/
```

### 2. 단위 테스트 실행
```bash
# 모든 테스트 실행
npx hardhat test

# SageRegistryV2만 테스트
npx hardhat test test/SageRegistryV2.test.js

# 특정 테스트만 실행
npx hardhat test test/SageRegistryV2.test.js --grep "Should accept valid uncompressed"

# 가스 리포트 포함
REPORT_GAS=true npx hardhat test test/SageRegistryV2.test.js

# 커버리지 확인
npx hardhat coverage
```

### 3. 로컬 노드에서 테스트
```bash
# Terminal 1: 로컬 노드 실행
npx hardhat node

# Terminal 2: 로컬 노드에 배포 및 테스트
npx hardhat run scripts/deploy-v2.js --network localhost

# 로컬 노드에서 상호작용
npx hardhat console --network localhost
```

## 🌐 테스트넷 배포

### 1. 네트워크 연결 확인
```bash
# Kaia 테스트넷 연결 테스트
npx hardhat run scripts/test-connection.js --network kairos
```

`scripts/test-connection.js` 생성:
```javascript
async function main() {
  const [deployer] = await ethers.getSigners();
  const balance = await ethers.provider.getBalance(deployer.address);
  
  console.log("Network:", network.name);
  console.log("Chain ID:", (await ethers.provider.getNetwork()).chainId);
  console.log("Deployer:", deployer.address);
  console.log("Balance:", ethers.formatEther(balance), "KLAY");
}

main().catch(console.error);
```

### 2. 테스트넷 배포 (자동)
```bash
# 배포 스크립트 실행
./deploy-v2.sh

# 또는 직접 실행
npx hardhat run scripts/deploy-v2.js --network kairos
```

### 3. 테스트넷 배포 (수동)
```bash
# Hardhat Console로 수동 배포
npx hardhat console --network kairos

# Console에서 실행:
> const SageRegistryV2 = await ethers.getContractFactory("SageRegistryV2")
> const registry = await SageRegistryV2.deploy()
> await registry.waitForDeployment()
> const address = await registry.getAddress()
> console.log("Deployed to:", address)

> const Hook = await ethers.getContractFactory("SageVerificationHook")
> const hook = await Hook.deploy()
> await hook.waitForDeployment()
> console.log("Hook deployed to:", await hook.getAddress())

> await registry.setBeforeRegisterHook(await hook.getAddress())
> console.log("Hook configured!")
> .exit
```

## 🔍 배포 검증

### 1. 컨트랙트 검증 (Kaiascope)
```bash
# 자동 검증
npx hardhat verify --network kairos DEPLOYED_CONTRACT_ADDRESS

# 수동 검증 (생성자 인자가 있는 경우)
npx hardhat verify --network kairos \
  --constructor-args arguments.js \
  DEPLOYED_CONTRACT_ADDRESS
```

### 2. 배포된 컨트랙트 테스트
```bash
# 테스트 스크립트 생성: scripts/test-deployed.js
```

```javascript
async function main() {
  const REGISTRY_ADDRESS = "YOUR_DEPLOYED_ADDRESS";
  const [signer] = await ethers.getSigners();
  
  // 컨트랙트 연결
  const registry = await ethers.getContractAt("SageRegistryV2", REGISTRY_ADDRESS);
  
  // 테스트 1: Owner 확인
  console.log("Owner:", await registry.owner());
  
  // 테스트 2: 에이전트 등록
  const publicKey = "0x04" + ethers.hexlify(ethers.randomBytes(64)).slice(2);
  const did = `did:sage:test:${signer.address}`;
  
  // 서명 생성
  const keyHash = ethers.keccak256(publicKey);
  const chainId = (await ethers.provider.getNetwork()).chainId;
  
  const packedData = ethers.solidityPacked(
    ["string", "uint256", "address", "address", "bytes32"],
    ["SAGE Key Registration:", chainId, REGISTRY_ADDRESS, signer.address, keyHash]
  );
  
  const challenge = ethers.keccak256(packedData);
  const signature = await signer.signMessage(ethers.getBytes(challenge));
  
  // 등록
  console.log("Registering agent...");
  const tx = await registry.registerAgent(
    did,
    "Test Agent",
    "Test Description",
    "https://test.com",
    publicKey,
    JSON.stringify(["test"]),
    signature
  );
  
  const receipt = await tx.wait();
  console.log("✅ Agent registered! Gas used:", receipt.gasUsed.toString());
  
  // 테스트 3: 조회
  const agent = await registry.getAgentByDID(did);
  console.log("Agent name:", agent.name);
  console.log("Agent active:", agent.active);
}

main().catch(console.error);
```

실행:
```bash
npx hardhat run scripts/test-deployed.js --network kairos
```

## 🛠 상호작용 및 관리

### 1. Hardhat Console 사용
```bash
# 테스트넷 연결
npx hardhat console --network kairos

# 컨트랙트 인스턴스 가져오기
> const registry = await ethers.getContractAt("SageRegistryV2", "DEPLOYED_ADDRESS")

# 함수 호출 예제
> await registry.owner()
> await registry.isKeyValid("0x04...")
> const agents = await registry.getAgentsByOwner("0x...")
```

### 2. 스크립트로 관리
```bash
# 키 폐기 스크립트: scripts/revoke-key.js
```

```javascript
async function revokeKey(registryAddress, publicKey) {
  const [signer] = await ethers.getSigners();
  const registry = await ethers.getContractAt("SageRegistryV2", registryAddress);
  
  console.log("Revoking key...");
  const tx = await registry.revokeKey(publicKey);
  await tx.wait();
  console.log("✅ Key revoked!");
  
  // 확인
  const isValid = await registry.isKeyValid(publicKey);
  console.log("Key valid?", isValid);
}

// 실행: npx hardhat run scripts/revoke-key.js --network kairos
```

## 📊 모니터링

### 1. 이벤트 모니터링
```javascript
// scripts/monitor-events.js
async function monitor(registryAddress) {
  const registry = await ethers.getContractAt("SageRegistryV2", registryAddress);
  
  // 에이전트 등록 이벤트 모니터링
  registry.on("AgentRegistered", (agentId, owner, did, timestamp) => {
    console.log(`New Agent: ${agentId}`);
    console.log(`  Owner: ${owner}`);
    console.log(`  DID: ${did}`);
    console.log(`  Time: ${new Date(timestamp * 1000)}`);
  });
  
  // 키 폐기 이벤트 모니터링
  registry.on("KeyRevoked", (keyHash, owner) => {
    console.log(`Key Revoked: ${keyHash} by ${owner}`);
  });
  
  console.log("Monitoring events... Press Ctrl+C to stop");
}
```

### 2. 상태 확인
```javascript
// scripts/check-status.js
async function checkStatus(registryAddress) {
  const registry = await ethers.getContractAt("SageRegistryV2", registryAddress);
  const [signer] = await ethers.getSigners();
  
  // 내 에이전트들 확인
  const myAgents = await registry.getAgentsByOwner(signer.address);
  console.log(`You have ${myAgents.length} agents`);
  
  for (const agentId of myAgents) {
    const agent = await registry.getAgent(agentId);
    console.log(`- ${agent.name}: ${agent.active ? "✅ Active" : "❌ Inactive"}`);
  }
}
```

## 🚨 문제 해결

### 일반적인 오류와 해결방법

#### 1. "Insufficient funds" 오류
```bash
# 잔액 확인
npx hardhat run scripts/check-balance.js --network kairos

# Faucet에서 테스트 KLAY 받기
# https://kairos.wallet.kaia.io/faucet
```

#### 2. "Nonce too high" 오류
```bash
# 로컬 nonce 리셋
npx hardhat clean
```

#### 3. "Key ownership not proven" 오류
```javascript
// 올바른 서명 생성 확인
const challenge = ethers.keccak256(
  ethers.solidityPacked(
    ["string", "uint256", "address", "address", "bytes32"],
    ["SAGE Key Registration:", chainId, contractAddress, signerAddress, keyHash]
  )
);
```

#### 4. 가스 부족
```javascript
// 가스 리밋 수동 설정
const tx = await registry.registerAgent(...params, {
  gasLimit: 800000
});
```

## 📝 체크리스트

### 테스트넷 배포 전
- [ ] 로컬 테스트 모두 통과
- [ ] 환경 변수 설정 완료
- [ ] 테스트 KLAY 충분히 보유
- [ ] 네트워크 연결 확인

### 배포 후
- [ ] 컨트랙트 주소 기록
- [ ] 블록 익스플로러에서 확인
- [ ] 기본 기능 테스트
- [ ] 이벤트 발생 확인

### 메인넷 배포 전
- [ ] 테스트넷에서 충분한 테스트
- [ ] 보안 감사 완료
- [ ] 가스 최적화 확인
- [ ] 백업 및 복구 계획

## 🆘 도움말

### 유용한 명령어 모음
```bash
# 컴파일 및 크기 확인
npx hardhat compile --force
npx hardhat size-contracts

# 계정 목록
npx hardhat accounts

# 네트워크 확인
npx hardhat run scripts/check-network.js --network kairos

# 클린 빌드
npx hardhat clean
rm -rf artifacts cache
npx hardhat compile
```

### 추가 리소스
- [Kaia Docs](https://docs.kaia.io)
- [Hardhat Docs](https://hardhat.org/docs)
- [Ethers.js Docs](https://docs.ethers.org)
- [Kaia Faucet](https://kairos.wallet.kaia.io/faucet)
- [Kaiascope](https://kairos.kaiascope.com)

---

**중요**: 메인넷 배포 전에는 반드시 테스트넷에서 모든 기능을 충분히 테스트하세요!