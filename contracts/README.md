# SAGE Smart Contracts

Smart contracts for SAGE (Secure Agent Guarantee Engine) AI Agent Registry on Ethereum and Solana blockchains.

## Overview

The SAGE contracts provide a decentralized registry for AI agents with enhanced security features:

### Core Features
- **Enhanced Public Key Validation**: 5-step cryptographic validation for maximum security
- **Key Ownership Proof**: Challenge-response authentication prevents unauthorized registrations
- **Key Revocation**: Compromised keys can be revoked with automatic agent deactivation
- **Metadata Storage**: Store agent DID, name, description, endpoint, and capabilities
- **Update/Deactivation**: Owners can update metadata or deactivate agents
- **Hook System**: Extensible before/after registration hooks for additional verification
- **Multi-chain Support**: Implementations for both Ethereum and Solana

### V2 Enhancements (Ethereum)
-  **5-Step Public Key Validation**
  - Length validation (33, 64, or 65 bytes)
  - Format validation (0x04 for uncompressed, 0x02/0x03 for compressed)
  - Zero-key prevention
  - Ownership proof via signature challenge
  - Revocation status check
-  **Key Revocation System**: Revoke compromised keys and auto-deactivate affected agents
-  **Improved Gas Efficiency**: Optimized storage and validation logic
-  **Enhanced Security**: Prevention of key reuse attacks and zero-key exploits

## Architecture

### Ethereum Implementation

#### V2 Contracts (Recommended)
- **SageRegistryV2.sol**: Enhanced registry with 5-step public key validation
- **SageVerificationHook.sol**: Hook implementation with DID validation, rate limiting, and blacklist
- **IRegistryHook.sol**: Hook interface for extensibility
- **ISageRegistry.sol**: Registry interface

#### V1 Contracts (Legacy)
- **SageRegistry.sol**: Original registry contract with basic signature verification

### Solana Implementation

- **sage-registry**: Main Anchor program using PDAs for agent accounts
- **sage-verification-hook**: Verification hook with rate limiting

## Security Features

### V2 Security Enhancements
1. **5-Step Public Key Validation**
   - Length check: Validates key is 33, 64, or 65 bytes
   - Format verification: Ensures proper prefix (0x04, 0x02, 0x03)
   - Zero-key prevention: Rejects all-zero keys
   - Ownership proof: Challenge-response signature verification
   - Revocation check: Prevents use of revoked keys

2. **Key Revocation System**
   - Owner-controlled key revocation
   - Automatic agent deactivation on key revocation
   - Prevention of revoked key reuse
   - Key-to-owner mapping for secure revocation

3. **Enhanced Registration Security**
   - Challenge-based signature for key ownership proof
   - Separate validation from registration signature
   - Protection against key hijacking attacks

### Standard Security Features
- **Signature Verification**: All operations require valid signatures
- **Owner-only Operations**: Only agent owners can update/deactivate
- **Rate Limiting**: 1-minute cooldown between registrations
- **Daily Limits**: Maximum 5 registrations per address per day
- **Blacklist Support**: Block malicious actors
- **Nonce Tracking**: Prevent replay attacks
- **DID Format Validation**: Ensures proper DID structure

## Git Submodule Setup

To use these contracts in a separate repository:

```bash
# From your main project root
git submodule add https://github.com/your-org/sage-contracts.git contracts
git submodule update --init --recursive

# To update the submodule later
cd contracts
git pull origin main
cd ..
git add contracts
git commit -m "Update contracts submodule"
```

## Deployment

### Ethereum

#### Quick Start (Local Development)
```bash
cd contracts/ethereum

# Full setup: Start node + Deploy contracts
./bin/deploy-local.sh
# Choose option 6

# Or step by step:
# 1. Start Hardhat node
npx hardhat node

# 2. Deploy V2 contracts (recommended)
npx hardhat run scripts/deploy-v2.js --network localhost

# 3. Interact with contracts
npx hardhat run scripts/interact-local.js --network localhost
```

#### Testnet Deployment (Kaia/Klaytn)
```bash
cd contracts/ethereum
npm install

# Set up environment
cp .env.example .env
# Edit .env with your private key and RPC URLs

# Compile contracts
npx hardhat compile

# Deploy to Kaia testnet (Kairos)
npx hardhat run scripts/deploy-v2.js --network kairos

# Verify contracts
npx hardhat verify --network kairos <CONTRACT_ADDRESS>
```

#### Mainnet Deployment
```bash
# Deploy to Kaia mainnet (Cypress)
npx hardhat run scripts/deploy-v2.js --network cypress

# Deploy to Ethereum mainnet
npx hardhat run scripts/deploy-v2.js --network mainnet
```

### Solana

```bash
cd contracts/solana
anchor build
./scripts/deploy.sh

# Or deploy manually
anchor deploy
anchor run initialize
```

## Usage Examples

### Ethereum V2 Registration (Recommended)

```javascript
const { ethers } = require("ethers");
const registry = new ethers.Contract(registryAddress, registryABI, signer);

// Prepare agent data
const agentData = {
  did: `did:sage:agent:${signer.address}`,
  name: "My AI Agent",
  description: "An intelligent assistant",
  endpoint: "https://api.myagent.ai",
  publicKey: ethers.concat(["0x04", ethers.randomBytes(64)]), // Uncompressed secp256k1
  capabilities: JSON.stringify(["chat", "code", "analysis"])
};

// V2 requires key ownership proof via challenge signature
const keyHash = ethers.keccak256(agentData.publicKey);
const chainId = (await provider.getNetwork()).chainId;

// Create challenge for key ownership proof
const challenge = ethers.keccak256(
  ethers.solidityPacked(
    ["string", "uint256", "address", "address", "bytes32"],
    ["SAGE Key Registration:", chainId, registryAddress, signer.address, keyHash]
  )
);

// Sign the challenge
const signature = await signer.signMessage(ethers.getBytes(challenge));

// Register agent
const tx = await registry.registerAgent(
  agentData.did,
  agentData.name,
  agentData.description,
  agentData.endpoint,
  agentData.publicKey,
  agentData.capabilities,
  signature
);

await tx.wait();
console.log("Agent registered with ID:", tx.hash);
```

### Key Revocation (V2 Only)

```javascript
// Revoke a compromised key
const tx = await registry.revokeKey(publicKey);
await tx.wait();
// All agents using this key are automatically deactivated
```

### Query Agents

```javascript
// Get agents by owner
const agentIds = await registry.getAgentsByOwner(ownerAddress);

// Get agent details
const agent = await registry.getAgent(agentId);
console.log("Name:", agent.name);
console.log("Active:", agent.active);

// Check if key is valid
const isValid = await registry.isKeyValid(publicKey);
```

### Solana Registration

```typescript
const program = anchor.workspace.SageRegistry;

// Sign with Ed25519
const signature = await ed.sign(messageBytes, wallet.secretKey);

// Register agent
await program.methods
  .registerAgent(
    did,
    name,
    description,
    endpoint,
    capabilities,
    Buffer.from(signature)
  )
  .accounts({
    agent: agentPDA,
    registry: registryPDA,
    owner: wallet.publicKey,
    systemProgram: SystemProgram.programId,
    ed25519Program: ED25519_PROGRAM_ID,
  })
  .rpc();
```

## Gas Optimization & Performance

### Ethereum V2 Improvements
- **Registration**: ~620,000 gas (includes enhanced validation)
- **Update**: ~80,000 gas
- **Key Revocation**: ~66,000 gas
- **Deactivation**: ~50,000 gas

#### Optimization Techniques
- Efficient storage packing in AgentMetadata struct
- Minimal storage writes with batched updates
- Events for off-chain indexing
- Optimized signature verification using ecrecover
- Reduced external calls in validation flow

### Solana
- PDA-based accounts minimize rent
- Efficient account sizing
- Native Ed25519 support
- Minimal compute units

## Testing

### Ethereum

#### Run All Tests
```bash
cd contracts/ethereum

# Run all tests (V1 + V2 + Integration)
./bin/test-all.sh

# Or run specific test suites
npx hardhat test test/SageRegistryV2.test.js        # V2 tests
npx hardhat test test/SageRegistry.test.fixed.js    # V1 tests
npx hardhat test test/integration-v2.test.js        # Integration tests

# With coverage
npx hardhat coverage
```

#### Test Coverage
-  V2 Public Key Validation (5-step process)
-  Key Revocation & Auto-deactivation
-  Hook Integration (DID validation, rate limiting)
-  Signature Verification (challenge-response)
-  Access Control & Ownership
-  Gas Usage Optimization

### Solana
```bash
cd contracts/solana
anchor test
```

## Project Structure

```
contracts/
├── ethereum/
│   ├── contracts/
│   │   ├── SageRegistryV2.sol          # V2 registry with enhanced validation
│   │   ├── SageRegistry.sol            # V1 registry (legacy)
│   │   ├── SageVerificationHook.sol    # Hook implementation
│   │   └── interfaces/
│   │       ├── ISageRegistry.sol       # Registry interface
│   │       └── IRegistryHook.sol       # Hook interface
│   ├── test/
│   │   ├── SageRegistryV2.test.js      # V2 unit tests
│   │   ├── SageRegistry.test.fixed.js  # V1 compatibility tests
│   │   ├── integration-v2.test.js      # Integration tests
│   │   └── integration.test.js         # Legacy integration tests
│   ├── scripts/
│   │   ├── deploy-v2.js                # V2 deployment script
│   │   ├── deploy-local.js             # Local deployment
│   │   ├── interact-local.js           # Interactive CLI
│   │   └── query-agents.js             # Query utilities
│   ├── bin/
│   │   ├── deploy-local.sh             # Local deployment helper
│   │   ├── test-all.sh                 # Run all tests
│   │   ├── test-v2.sh                  # V2 tests only
│   │   └── query-agents.sh             # Query helper
│   └── hardhat.config.js               # Hardhat configuration
└── solana/
    └── programs/
        └── sage-registry/               # Solana implementation

```

## Contract Addresses (Testnet)

### Kaia Testnet (Kairos)
- **SageRegistryV2**: `[To be deployed]`
- **SageVerificationHook**: `[To be deployed]`

### Local Development
- **SageRegistryV2**: `0x5FbDB2315678afecb367f032d93F642f64180aa3`
- **SageVerificationHook**: `0xe7f1725E7734CE288F8367e1Bb143E90bb3F0512`

## Migration Guide (V1 → V2)

### Key Changes
1. **Signature Method**: V2 uses challenge-response for key ownership proof
2. **Public Key Format**: Must include proper prefix (0x04, 0x02, 0x03)
3. **New Features**: Key revocation, enhanced validation
4. **Gas Cost**: Slightly higher due to enhanced security (~620k vs ~400k)

### Migration Steps
1. Deploy V2 contracts
2. Update frontend to use new signature method
3. Migrate existing agents (optional - V1 remains functional)
4. Update monitoring for new events

## Contributing

1. Fork the repository
2. Create your feature branch (`git checkout -b feature/amazing-feature`)
3. Commit your changes (`git commit -m 'Add amazing feature'`)
4. Push to the branch (`git push origin feature/amazing-feature`)
5. Open a Pull Request

## License

MIT