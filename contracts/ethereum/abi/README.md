# 📜 SAGE Contract ABIs

## Overview
This directory contains the Application Binary Interface (ABI) files for SAGE smart contracts. ABIs are essential for interacting with deployed contracts from frontend applications or scripts.

## Available ABIs

### 1. **SageRegistryV2.abi.json**
Main registry contract for managing AI agents on the blockchain.

**Key Functions:**
- `registerAgent` - Register a new AI agent with public key validation
- `updateAgent` - Update agent metadata
- `deactivateAgent` - Deactivate an agent
- `revokeKey` - Revoke a compromised public key
- `getAgent` - Retrieve agent information
- `getAgentByDID` - Get agent by DID
- `getAgentsByOwner` - List all agents owned by an address
- `isKeyValid` - Check if a public key is valid

**Events:**
- `AgentRegistered` - Emitted when a new agent is registered
- `AgentUpdated` - Emitted when agent metadata is updated
- `AgentDeactivated` - Emitted when an agent is deactivated
- `KeyValidated` - Emitted when a key is validated
- `KeyRevoked` - Emitted when a key is revoked

### 2. **SageVerificationHook.abi.json**
Hook contract for additional verification during agent registration.

**Key Functions:**
- `beforeRegister` - Pre-registration validation hook
- `afterRegister` - Post-registration processing hook

## Usage Examples

### JavaScript/TypeScript (ethers.js v6)

```javascript
import { ethers } from 'ethers';
import SageRegistryV2ABI from './abi/SageRegistryV2.abi.json';

// Connect to provider
const provider = new ethers.JsonRpcProvider('https://public-en-kairos.node.kaia.io');

// Contract address (replace with actual deployed address)
const contractAddress = '0x...';

// Create contract instance
const registry = new ethers.Contract(
  contractAddress,
  SageRegistryV2ABI,
  provider
);

// Read operations
const agent = await registry.getAgentByDID('did:sage:example');
console.log('Agent:', agent);

// Write operations (requires signer)
const signer = new ethers.Wallet(privateKey, provider);
const registryWithSigner = registry.connect(signer);

// Register new agent
const tx = await registryWithSigner.registerAgent(
  did,
  name,
  description,
  endpoint,
  publicKey,
  capabilities,
  signature
);
await tx.wait();
```

### Web3.js

```javascript
import Web3 from 'web3';
import SageRegistryV2ABI from './abi/SageRegistryV2.abi.json';

const web3 = new Web3('https://public-en-kairos.node.kaia.io');

const registry = new web3.eth.Contract(
  SageRegistryV2ABI,
  contractAddress
);

// Read operations
const agent = await registry.methods.getAgentByDID('did:sage:example').call();
```

### Python (web3.py)

```python
from web3 import Web3
import json

# Load ABI
with open('abi/SageRegistryV2.abi.json', 'r') as f:
    abi = json.load(f)

# Connect to network
w3 = Web3(Web3.HTTPProvider('https://public-en-kairos.node.kaia.io'))

# Create contract instance
registry = w3.eth.contract(
    address=contract_address,
    abi=abi
)

# Read operations
agent = registry.functions.getAgentByDID('did:sage:example').call()
```

## ABI Generation

ABIs are automatically generated when contracts are compiled:

```bash
# Compile contracts
npm run compile

# ABIs will be extracted to this directory
# Source: artifacts/contracts/[ContractName].sol/[ContractName].json
```

## Network Deployments

Get the deployed contract addresses from:
- `deployments/kairos-v2-latest.json` - Kairos testnet
- `deployments/localhost-v2-latest.json` - Local development
- Environment variables: `SAGE_REGISTRY_ADDRESS`, `SAGE_VERIFICATION_HOOK_ADDRESS`

## Important Notes

1. **Version**: These ABIs are for SageRegistryV2 (enhanced security version)
2. **Chain**: Designed for Kaia network but compatible with EVM chains
3. **Updates**: Regenerate ABIs after contract modifications
4. **Security**: Never expose private keys in frontend applications

## Support

For questions or issues, refer to:
- [Main Documentation](../README.md)
- [Deployment Guide](../docs/DEPLOYMENT_GUIDE.md)
- [Contract Source](../contracts/)