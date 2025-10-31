#  Agent 조회 명령어 모음

**Version**: 2.0 (AgentCard Architecture)
**Last Updated**: 2025-11-01

## 1. 직접 JavaScript 실행 (Node.js)

### 모든 Agent 조회
```bash
cd /Users/0xtopaz/work/github/sage-x-project/sage/contracts/ethereum
node scripts/query-agents.js
```

### 특정 Owner의 Agent 조회
```bash
node scripts/query-agents.js by-owner 0x70997970C51812dc3A010C7d01b50e0d17dc79C8
```

### Agent ID로 조회
```bash
node scripts/query-agents.js by-id 0x5c7cdd064a1d14d8b9d6eae7e3ce2f3095e5b118d9c5fda8ef9567e4aebb9412
```

### 통계 정보
```bash
node scripts/query-agents.js stats
```

## 2. Curl을 사용한 직접 RPC 호출

### getAgentsByOwner 호출
```bash
# Test Account 1의 agents 조회
curl -X POST http://localhost:8545 \
  -H "Content-Type: application/json" \
  --data '{
    "jsonrpc": "2.0",
    "method": "eth_call",
    "params": [{
      "to": "0x5FbDB2315678afecb367f032d93F642f64180aa3",
      "data": "0x4b9f0cea00000000000000000000000070997970c51812dc3a010c7d01b50e0d17dc79c8"
    }, "latest"],
    "id": 1
  }'
```

### 이벤트 로그 조회
```bash
# AgentRegistered 이벤트 조회
curl -X POST http://localhost:8545 \
  -H "Content-Type: application/json" \
  --data '{
    "jsonrpc": "2.0",
    "method": "eth_getLogs",
    "params": [{
      "fromBlock": "0x0",
      "toBlock": "latest",
      "address": "0x5FbDB2315678afecb367f032d93F642f64180aa3",
      "topics": ["0x8a5c4c6e2f7a3b1d7e9c8f5a2b4d6e8f1a3c5e7b9d2f4a6c8e0a2c4e6a8c0e2a4e"]
    }],
    "id": 1
  }'
```

## 3. Hardhat Console 사용

```bash
cd /Users/0xtopaz/work/github/sage-x-project/sage/contracts/ethereum
npx hardhat console --network localhost
```

Console에서:
```javascript
// 컨트랙트 연결
const registry = await ethers.getContractAt("AgentCardRegistry", "0x5FbDB2315678afecb367f032d93F642f64180aa3")

// Test Account 1의 agents 조회
const agents = await registry.getAgentsByOwner("0x70997970C51812dc3A010C7d01b50e0d17dc79C8")
console.log("Agent IDs:", agents)

// 첫 번째 agent 상세 정보
if(agents.length > 0) {
  const agent = await registry.getAgent(agents[0])
  console.log("Agent metadata:", agent)
  console.log("DID:", agent.did)
  console.log("Name:", agent.name)
  console.log("Active:", agent.active)
  console.log("Keys:", agent.keyHashes)
  console.log("KME Public Key:", agent.kmePublicKey)
}

// 특정 키 조회
const keyHash = agent.keyHashes[0]
const keyInfo = await registry.getKey(keyHash)
console.log("Key type:", keyInfo.keyType) // 0=ECDSA, 1=Ed25519, 2=X25519
console.log("Key data:", keyInfo.keyData)
console.log("Verified:", keyInfo.verified)

// DID로 Agent 조회
const agentByDID = await registry.getAgentByDID("did:sage:ethereum:0x...")
console.log(agentByDID)

// 모든 이벤트 조회
const events = await registry.queryFilter(registry.filters.AgentRegistered())
events.forEach(e => {
  console.log(`Agent ID: ${e.topics[1]}`) // Indexed parameter
  console.log(`DID: ${e.args.did}`)
  console.log(`Owner: ${e.args.owner}`)
})
```

## 4. 간단한 One-liner 조회 스크립트

### 현재 등록된 모든 Agent 수 확인
```bash
cd /Users/0xtopaz/work/github/sage-x-project/sage/contracts/ethereum

node -e "
const ethers = require('ethers');
const provider = new ethers.JsonRpcProvider('http://localhost:8545');
const abi = [{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"name\":\"agentId\",\"type\":\"bytes32\"},{\"indexed\":true,\"name\":\"owner\",\"type\":\"address\"},{\"indexed\":false,\"name\":\"did\",\"type\":\"string\"},{\"indexed\":false,\"name\":\"timestamp\",\"type\":\"uint256\"}],\"name\":\"AgentRegistered\",\"type\":\"event\"}];
const registry = new ethers.Contract('0x5FbDB2315678afecb367f032d93F642f64180aa3', abi, provider);
registry.queryFilter(registry.filters.AgentRegistered()).then(events => {
  console.log('Total agents registered:', events.length);
  events.forEach((e, i) => {
    console.log(\`\${i+1}. Agent ID: \${e.args[0]}\`);
    console.log(\`   Owner: \${e.args[1]}\`);
    console.log(\`   DID: \${e.args[2]}\`);
  });
}).catch(console.error);
"
```

### Test Account 1의 Agent 확인
```bash
node -e "
const ethers = require('ethers');
const fs = require('fs');
const provider = new ethers.JsonRpcProvider('http://localhost:8545');
const abi = JSON.parse(fs.readFileSync('./artifacts/contracts/AgentCardRegistry.sol/AgentCardRegistry.json')).abi;
const registry = new ethers.Contract('0x5FbDB2315678afecb367f032d93F642f64180aa3', abi, provider);
registry.getAgentsByOwner('0x70997970C51812dc3A010C7d01b50e0d17dc79C8').then(async (ids) => {
  console.log('Found', ids.length, 'agent(s)');
  for(let id of ids) {
    const agent = await registry.getAgent(id);
    console.log('Name:', agent.name);
    console.log('DID:', agent.did);
    console.log('Active:', agent.active);
    console.log('Keys:', agent.keyHashes.length, 'keys');
    console.log('KME Key:', agent.kmePublicKey ? 'Yes' : 'No');
    console.log('---');
  }
}).catch(console.error);
"
```

## 5. AgentCard 특화 조회 명령어

### Multi-Key 정보 조회
```javascript
// Hardhat Console에서
const registry = await ethers.getContractAt("AgentCardRegistry", "0x...")

// Agent의 모든 키 조회
const agent = await registry.getAgent(agentId)
console.log("Total keys:", agent.keyHashes.length)

for (let keyHash of agent.keyHashes) {
  const keyInfo = await registry.getKey(keyHash)
  const keyTypeNames = ["ECDSA", "Ed25519", "X25519"]
  console.log(`Key Type: ${keyTypeNames[keyInfo.keyType]}`)
  console.log(`Key Data: ${keyInfo.keyData}`)
  console.log(`Verified: ${keyInfo.verified}`)
  console.log("---")
}
```

### KME Public Key 조회
```javascript
// KME 공개키 (X25519) 확인
const agent = await registry.getAgent(agentId)
if (agent.kmePublicKey && agent.kmePublicKey !== "0x") {
  console.log("KME Public Key (X25519):", agent.kmePublicKey)
  console.log("Length:", agent.kmePublicKey.length - 2, "bytes") // Should be 32
} else {
  console.log("No KME public key set")
}
```

### Activation Status 확인
```javascript
// Time-lock 활성화 상태 확인
const agent = await registry.getAgent(agentId)
const activationTime = await registry.agentActivationTime(agentId)
const now = Math.floor(Date.now() / 1000)

console.log("Active:", agent.active)
console.log("Activation Time:", new Date(activationTime * 1000))
console.log("Can activate:", now >= activationTime)
```

### Stake 정보 조회
```javascript
// Agent의 stake 금액 확인
const stake = await registry.agentStakes(agentId)
console.log("Stake:", ethers.formatEther(stake), "ETH")

// 전역 stake 요구사항
const requiredStake = await registry.registrationStake()
console.log("Required Stake:", ethers.formatEther(requiredStake), "ETH")
```

### Reputation 정보 조회
```javascript
// Agent owner의 reputation 확인
const reputation = await registry.agentReputations(ownerAddress)
console.log("Successful Interactions:", reputation.successfulInteractions)
console.log("Failed Interactions:", reputation.failedInteractions)
console.log("Reputation Score:", reputation.reputationScore, "/ 100")
console.log("Verified:", reputation.verified)
```

## 6. 현재 배포된 Agent 정보 (알려진 정보)

배포 시 자동으로 등록된 테스트 Agent:
- **Agent ID**: `0x5c7cdd064a1d14d8b9d6eae7e3ce2f3095e5b118d9c5fda8ef9567e4aebb9412`
- **Owner**: `0x70997970C51812dc3A010C7d01b50e0d17dc79C8` (Test Account 1)
- **DID**: `did:sage:ethereum:0x70997970C51812dc3A010C7d01b50e0d17dc79C8`
- **Name**: `Test AI Agent`
- **Keys**: 3 keys (ECDSA + Ed25519 + X25519)

### 이 Agent 직접 조회
```bash
node scripts/query-agents.js by-id 0x5c7cdd064a1d14d8b9d6eae7e3ce2f3095e5b118d9c5fda8ef9567e4aebb9412
```

또는

```bash
node scripts/query-agents.js by-did "did:sage:ethereum:0x70997970C51812dc3A010C7d01b50e0d17dc79C8"
```