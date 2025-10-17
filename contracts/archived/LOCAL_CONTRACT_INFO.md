#  로컬 배포 컨트랙트 정보

## 🔗 Network Information

| 항목 | 값 |
|------|-----|
| **RPC URL** | `http://localhost:8545` |
| **Chain ID** | `31337` |
| **Network Name** | `Hardhat Local Network` |

## 📍 Contract Addresses

| Contract | Address |
|----------|---------|
| **SageRegistryV2** | `0x5FbDB2315678afecb367f032d93F642f64180aa3` |
| **SageVerificationHook** | `0xe7f1725E7734CE288F8367e1Bb143E90bb3F0512` |

## 📄 ABI File Paths

| Contract | ABI File Path |
|----------|---------------|
| **SageRegistryV2** | `/Users/0xtopaz/work/github/sage-x-project/sage/contracts/ethereum/artifacts/contracts/SageRegistryV2.sol/SageRegistryV2.json` |
| **SageVerificationHook** | `/Users/0xtopaz/work/github/sage-x-project/sage/contracts/ethereum/artifacts/contracts/SageVerificationHook.sol/SageVerificationHook.json` |

##  Test Accounts (with Private Keys)

로컬 Hardhat 노드의 기본 테스트 계정들:

### Account #0 (Owner)
- **Address**: `0xf39Fd6e51aad88F6F4ce6aB8827279cffFb92266`
- **Private Key**: `0xac0974bec39a17e36ba4a6b4d238ff944bacb478cbed5efcae784d7bf4f2ff80`
- **Balance**: 10000 ETH

### Account #1 (Agent1)
- **Address**: `0x70997970C51812dc3A010C7d01b50e0d17dc79C8`
- **Private Key**: `0x59c6995e998f97a5a0044966f0945389dc9e86dae88c7a8412f4603b6b78690d`
- **Balance**: 10000 ETH

### Account #2 (Agent2)
- **Address**: `0x3C44CdDdB6a900fa2b585dd299e03d12FA4293BC`
- **Private Key**: `0x5de4111afa1a4b94908f83103eb1f1706367c2e68ca870fc3fb9a804cdab365a`
- **Balance**: 10000 ETH

## 🧑‍💻 Example: ethers.js v6 Connection

```javascript
const { ethers } = require("ethers");
const fs = require("fs");

// 1. Connect to local network
const provider = new ethers.JsonRpcProvider("http://localhost:8545");

// 2. Create signer with private key
const privateKey = "0x59c6995e998f97a5a0044966f0945389dc9e86dae88c7a8412f4603b6b78690d"; // Agent1
const signer = new ethers.Wallet(privateKey, provider);

// 3. Load ABI
const contractABI = JSON.parse(
  fs.readFileSync("/Users/0xtopaz/work/github/sage-x-project/sage/contracts/ethereum/artifacts/contracts/SageRegistryV2.sol/SageRegistryV2.json", "utf8")
).abi;

// 4. Connect to contract
const registryAddress = "0x5FbDB2315678afecb367f032d93F642f64180aa3";
const registry = new ethers.Contract(registryAddress, contractABI, signer);

// 5. Register agent
async function registerAgent() {
  // Generate public key with 0x04 prefix (uncompressed)
  const randomKey = ethers.randomBytes(64);
  const publicKey = ethers.concat(["0x04", randomKey]);
  
  // Agent data
  const did = `did:sage:test:${signer.address}_${Date.now()}`;
  const name = "My AI Agent";
  const description = "Test agent";
  const endpoint = "https://localhost:8080";
  const capabilities = JSON.stringify(["chat", "code"]);
  
  // Create signature for key ownership proof
  const keyHash = ethers.keccak256(publicKey);
  const chainId = (await provider.getNetwork()).chainId;
  
  const challenge = ethers.keccak256(
    ethers.solidityPacked(
      ["string", "uint256", "address", "address", "bytes32"],
      ["SAGE Key Registration:", chainId, registryAddress, signer.address, keyHash]
    )
  );
  
  const signature = await signer.signMessage(ethers.getBytes(challenge));
  
  // Register
  const tx = await registry.registerAgent(
    did, name, description, endpoint, publicKey, capabilities, signature
  );
  
  const receipt = await tx.wait();
  console.log("Agent registered! TX:", receipt.hash);
}

registerAgent().catch(console.error);
```

## 🐍 Example: Python (web3.py)

```python
from web3 import Web3
import json

# 1. Connect to local network
w3 = Web3(Web3.HTTPProvider("http://localhost:8545"))

# 2. Load ABI
with open("/Users/0xtopaz/work/github/sage-x-project/sage/contracts/ethereum/artifacts/contracts/SageRegistryV2.sol/SageRegistryV2.json") as f:
    contract_json = json.load(f)
    abi = contract_json["abi"]

# 3. Contract instance
registry_address = "0x5FbDB2315678afecb367f032d93F642f64180aa3"
registry = w3.eth.contract(address=registry_address, abi=abi)

# 4. Set up account
private_key = "0x59c6995e998f97a5a0044966f0945389dc9e86dae88c7a8412f4603b6b78690d"
account = w3.eth.account.from_key(private_key)
```

##  Example: Using curl

```bash
# Get block number
curl -X POST http://localhost:8545 \
  -H "Content-Type: application/json" \
  --data '{
    "jsonrpc": "2.0",
    "method": "eth_blockNumber",
    "params": [],
    "id": 1
  }'

# Get balance
curl -X POST http://localhost:8545 \
  -H "Content-Type: application/json" \
  --data '{
    "jsonrpc": "2.0",
    "method": "eth_getBalance",
    "params": ["0x70997970C51812dc3A010C7d01b50e0d17dc79C8", "latest"],
    "id": 1
  }'
```

##  Important Notes

1. **Signature Requirements for V2**:
   - V2 requires a special challenge signature for public key ownership proof
   - The challenge format is: `"SAGE Key Registration:" + chainId + contractAddress + signerAddress + keyHash`
   - This is different from V1 which used registration data for signature

2. **Public Key Format**:
   - Must be 65 bytes with `0x04` prefix (uncompressed) OR
   - 33 bytes with `0x02` or `0x03` prefix (compressed)
   - Ed25519 keys (32 bytes) are NOT supported

3. **Hook Configuration**:
   - BeforeRegisterHook and AfterRegisterHook are already configured
   - DID validation and rate limiting are active
   - Daily limit: 5 registrations per address
   - Cooldown: 1 minute between registrations

4. **Gas Estimates**:
   - Agent Registration: ~627,000 gas
   - Agent Update: ~80,000 gas
   - Key Revocation: ~66,000 gas

##  Quick Test Script

Save as `test-registration.js`:

```javascript
const { ethers } = require("ethers");
const fs = require("fs");

async function main() {
  // Configuration
  const RPC_URL = "http://localhost:8545";
  const REGISTRY_ADDRESS = "0x5FbDB2315678afecb367f032d93F642f64180aa3";
  const PRIVATE_KEY = "0x5de4111afa1a4b94908f83103eb1f1706367c2e68ca870fc3fb9a804cdab365a"; // Account #2
  const ABI_PATH = "/Users/0xtopaz/work/github/sage-x-project/sage/contracts/ethereum/artifacts/contracts/SageRegistryV2.sol/SageRegistryV2.json";
  
  // Connect
  const provider = new ethers.JsonRpcProvider(RPC_URL);
  const signer = new ethers.Wallet(PRIVATE_KEY, provider);
  const abi = JSON.parse(fs.readFileSync(ABI_PATH, "utf8")).abi;
  const registry = new ethers.Contract(REGISTRY_ADDRESS, abi, signer);
  
  // Register agent
  const randomKey = ethers.randomBytes(64);
  const publicKey = ethers.concat(["0x04", randomKey]);
  
  const agentData = {
    did: `did:sage:test:${signer.address}_${Date.now()}`,
    name: "Test Agent from External Session",
    description: "Testing cross-session registration",
    endpoint: "https://example.com",
    capabilities: JSON.stringify(["chat", "code"])
  };
  
  // Create signature
  const keyHash = ethers.keccak256(publicKey);
  const chainId = (await provider.getNetwork()).chainId;
  
  const challenge = ethers.keccak256(
    ethers.solidityPacked(
      ["string", "uint256", "address", "address", "bytes32"],
      ["SAGE Key Registration:", chainId, REGISTRY_ADDRESS, signer.address, keyHash]
    )
  );
  
  const signature = await signer.signMessage(ethers.getBytes(challenge));
  
  console.log("Registering agent...");
  const tx = await registry.registerAgent(
    agentData.did,
    agentData.name,
    agentData.description,
    agentData.endpoint,
    publicKey,
    agentData.capabilities,
    signature
  );
  
  const receipt = await tx.wait();
  console.log(" Success! TX:", receipt.hash);
  
  // Get agent ID from event
  const logs = await registry.queryFilter(
    registry.filters.AgentRegistered(),
    receipt.blockNumber,
    receipt.blockNumber
  );
  
  if (logs.length > 0) {
    console.log("Agent ID:", logs[0].args[0]);
  }
}

main().catch(console.error);
```

Run with:
```bash
node test-registration.js
```