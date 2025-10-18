# SAGE Smart Contracts

Smart contracts for SAGE (Secure Agent Guarantee Engine) AI Agent Registry on Ethereum-compatible blockchains.

## Overview

The SAGE contracts provide a decentralized registry for AI agents with enhanced security features and comprehensive validation mechanisms.

### Core Features

- **Enhanced Public Key Validation**: 5-step cryptographic validation for maximum security
- **Key Ownership Proof**: Challenge-response authentication prevents unauthorized registrations
- **Key Revocation**: Compromised keys can be revoked with automatic agent deactivation
- **Metadata Storage**: Store agent DID, name, description, endpoint, and capabilities
- **Update/Deactivation**: Owners can update metadata or deactivate agents
- **Hook System**: Extensible before/after registration hooks for additional verification
- **Multi-chain Support**: Ethereum, Kaia, and other EVM-compatible chains
- **Gas Optimized**: Efficient storage patterns and streamlined operations

### V4 Enhancements (Latest Version - In Development)

**Multi-Key Agent Support with A2A Protocol Compatibility**

SageRegistryV4 introduces a comprehensive multi-key architecture enabling agents to register and manage multiple cryptographic key types simultaneously. This design provides true multi-chain interoperability and compatibility with the Google A2A (Agent-to-Agent) protocol.

**Key Features:**

- **Multi-Key Architecture**: Single agent can register up to 10 keys across different types
  - Ed25519 (32 bytes) - Solana, Cardano, Polkadot compatibility
  - ECDSA/secp256k1 (33/65 bytes) - Ethereum, Bitcoin compatibility
  - X25519 (32 bytes) - HPKE key exchange for secure communication

- **Type-Specific Verification**:
  - ECDSA keys verified on-chain via ecrecover signature validation
  - Ed25519 keys require registry owner pre-approval (EVM native support pending)
  - X25519 keys automatically verified upon registration

- **Key Lifecycle Management**:
  - `registerAgent()` - Register agent with multiple keys in single transaction
  - `addKey()` - Add new keys to existing agent with signature proof
  - `revokeKey()` - Revoke compromised keys (owner-controlled)
  - `approveEd25519Key()` - Owner approves Ed25519 keys after verification

- **A2A Protocol Integration**:
  - Native support for A2A Agent Card generation
  - DID-based identity with multi-key credential verification
  - Service endpoint management for agent discovery
  - Capability declarations for interoperability

- **Gas Efficiency**:
  - ~875,000 gas for single-key registration
  - ~1,300,000 gas for three-key registration
  - ~50,000 gas per additional key via addKey()
  - Optimized storage with packed AgentKey structs

- **Backward Compatibility**:
  - Maintains support for legacy single-key agents
  - Automatic conversion from V2/V3 agent metadata format
  - Migration path from previous registry versions

**Status**: Smart contract implementation complete (30 tests passing), Go backend integration complete (77.6% coverage), CLI tools available (card generate/validate/show). Deployment to testnet/mainnet pending.

### V2 Features (Stable)

- **5-Step Public Key Validation**
  - Length validation (33, 64, or 65 bytes for secp256k1)
  - Format validation (0x04 for uncompressed, 0x02/0x03 for compressed)
  - Zero-key prevention
  - Ownership proof via signature challenge
  - Revocation status check
- **Key Revocation System**: Revoke compromised keys and auto-deactivate affected agents
- **Improved Gas Efficiency**: Optimized storage and validation logic (~620k gas for registration)
- **Enhanced Security**: Prevention of key reuse attacks and zero-key exploits
- **Unified Deployment**: Single deployment script supports all networks

## Architecture

### Ethereum Implementation

#### V4 Contracts (Latest - In Development)

- **SageRegistryV4.sol**: Multi-key registry with Ed25519, ECDSA, and X25519 support
- **ISageRegistryV4.sol**: V4 registry interface with multi-key operations
- **Types**: AgentKey, AgentMetadataV4, KeyType enum (Ed25519, ECDSA, X25519)
- **Features**: Multi-key registration, key lifecycle management, A2A protocol compatibility

**Status**: Smart contract complete (30 unit tests passing), pending deployment to testnet

#### V2 Contracts (Stable Production)

- **SageRegistryV2.sol**: Enhanced registry with 5-step public key validation
- **SageVerificationHook.sol**: Hook implementation with DID validation, rate limiting, and blacklist
- **IRegistryHook.sol**: Hook interface for extensibility
- **ISageRegistry.sol**: Registry interface

#### V1 Contracts (Legacy - Deprecated)

- **SageRegistry.sol**: Original registry contract with basic signature verification

### Contract Addresses

#### Testnet Deployments

**Kaia Testnet (Kairos)**

- SageRegistryV2: `[To be deployed]`
- SageVerificationHook: `[To be deployed]`

**Local Development**

- SageRegistryV2: `0x5FbDB2315678afecb367f032d93F642f64180aa3`
- SageVerificationHook: `0xe7f1725E7734CE288F8367e1Bb143E90bb3F0512`

## Installation

### Prerequisites

- Node.js 18+ and npm
- Git

### Quick Start

```bash
cd contracts/ethereum

# Install dependencies
npm install

# Compile contracts
npx hardhat compile

# Run tests
npm test
```

## Deployment

### Unified Deployment Script

The new unified deployment script (`deploy-unified.js`) supports all networks with a single interface:

```bash
# Local development (requires running Hardhat node)
npx hardhat node                                    # Terminal 1
npx hardhat run scripts/deploy-unified.js --network localhost  # Terminal 2

# Testnet deployment (Kaia Kairos)
npx hardhat run scripts/deploy-unified.js --network kairos

# Mainnet deployment (Kaia Cypress)
npx hardhat run scripts/deploy-unified.js --network cypress
```

**Features:**

- Automatic network detection and configuration
- Health check before deployment
- Test agent registration (testnet only)
- Automatic contract verification (on supported explorers)
- Deployment info saved to `deployments/` directory
- Progress indicators and detailed logging

### Legacy Deployment Scripts

For specific use cases, individual deployment scripts are available:

```bash
# V2 deployment (generic)
npx hardhat run scripts/deploy-v2.js --network <network>

# Kaia-specific deployments
npx hardhat run scripts/deploy-kaia-v2-latest.js --network kairos
npx hardhat run scripts/deploy-kaia-v2.js --network cypress

# Local testing
npx hardhat run scripts/deploy-local.js --network localhost
```

### Environment Configuration

Create `.env` file:

```env
# Network RPC URLs
ETHEREUM_RPC_URL=https://eth-mainnet.g.alchemy.com/v2/YOUR_KEY
KAIA_RPC_URL=https://public-en.node.kaia.io
KAIROS_RPC_URL=https://public-en-kairos.node.kaia.io

# Deployer Private Key (use test keys only for testnet!)
PRIVATE_KEY=your_private_key_here
MNEMONIC=your_twelve_word_mnemonic_here

# Optional: For contract verification
ETHERSCAN_API_KEY=your_etherscan_key
KAIASCAN_API_KEY=your_kaiascan_key
```

## Security Features

### V2 Security Enhancements

#### 1. 5-Step Public Key Validation

```solidity
function validatePublicKey(
    bytes memory publicKey,
    address owner,
    bytes memory signature
) internal view returns (bool) {
    // Step 1: Length check (33, 64, or 65 bytes)
    require(
        publicKey.length == 33 ||
        publicKey.length == 64 ||
        publicKey.length == 65,
        "Invalid key length"
    );

    // Step 2: Format verification
    if (publicKey.length == 65) {
        require(publicKey[0] == 0x04, "Invalid uncompressed key prefix");
    } else if (publicKey.length == 33) {
        require(
            publicKey[0] == 0x02 || publicKey[0] == 0x03,
            "Invalid compressed key prefix"
        );
    }

    // Step 3: Zero-key prevention
    require(!isZeroKey(publicKey), "Zero key not allowed");

    // Step 4: Ownership proof (challenge-response)
    bytes32 keyHash = keccak256(publicKey);
    bytes32 challenge = keccak256(abi.encodePacked(
        "SAGE Key Registration:",
        block.chainid,
        address(this),
        owner,
        keyHash
    ));
    address recovered = ECDSA.recover(
        ECDSA.toEthSignedMessageHash(challenge),
        signature
    );
    require(recovered == owner, "Invalid ownership proof");

    // Step 5: Revocation check
    require(!revokedKeys[keyHash], "Key has been revoked");

    return true;
}
```

#### 2. Key Revocation System

- Owner-controlled key revocation
- Automatic agent deactivation on key revocation
- Prevention of revoked key reuse
- Key-to-owner mapping for secure revocation

#### 3. Enhanced Registration Security

- Challenge-based signature for key ownership proof
- Separate validation from registration signature
- Protection against key hijacking attacks

### Standard Security Features

- **Signature Verification**: All operations require valid signatures
- **Owner-only Operations**: Only agent owners can update/deactivate
- **Rate Limiting**: 1-minute cooldown between registrations (via hook)
- **Daily Limits**: Maximum 5 registrations per address per day (via hook)
- **Blacklist Support**: Block malicious actors (via hook)
- **Nonce Tracking**: Prevent replay attacks
- **DID Format Validation**: Ensures proper DID structure

## Usage Examples

### V2 Registration (Recommended)

```javascript
const { ethers } = require("ethers");
const registry = new ethers.Contract(registryAddress, registryABI, signer);

// Prepare agent data
const agentData = {
  did: `did:sage:ethereum:${signer.address}`,
  name: "My AI Agent",
  description: "An intelligent assistant",
  endpoint: "https://api.myagent.ai",
  publicKey: ethers.concat(["0x04", ethers.randomBytes(64)]), // Uncompressed secp256k1
  capabilities: JSON.stringify(["chat", "code", "analysis"]),
};

// V2 requires key ownership proof via challenge signature
const keyHash = ethers.keccak256(agentData.publicKey);
const chainId = (await provider.getNetwork()).chainId;

// Create challenge for key ownership proof
const challenge = ethers.keccak256(
  ethers.solidityPacked(
    ["string", "uint256", "address", "address", "bytes32"],
    [
      "SAGE Key Registration:",
      chainId,
      registryAddress,
      signer.address,
      keyHash,
    ]
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
console.log("Agent registered:", tx.hash);
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
console.log("DID:", agent.did);

// Check if key is valid
const isValid = await registry.isKeyValid(publicKey);
```

### Update Agent Metadata

```javascript
const tx = await registry.updateAgent(
  agentId,
  "Updated Name",
  "Updated description",
  "https://new-endpoint.com",
  JSON.stringify(["new", "capabilities"])
);
await tx.wait();
```

### Deactivate Agent

```javascript
const tx = await registry.deactivateAgent(agentId);
await tx.wait();
```

## Gas Optimization & Performance

### V2 Gas Usage

| Operation             | Gas Used | USD (@ 30 gwei) |
| --------------------- | -------- | --------------- |
| Register Agent        | ~620,000 | ~$48            |
| Update Agent Metadata | ~80,000  | ~$6             |
| Revoke Key            | ~66,000  | ~$5             |
| Deactivate Agent      | ~50,000  | ~$3.8           |
| Query Agent (view)    | 0        | $0              |

### Optimization Techniques

- Efficient storage packing in AgentMetadata struct
- Minimal storage writes with batched updates
- Events for off-chain indexing
- Optimized signature verification using ecrecover
- Reduced external calls in validation flow
- Gas-efficient key revocation mapping

## Testing

### Run All Tests

```bash
cd contracts/ethereum

# Run all test suites
npm test

# Run with coverage
npm run coverage

# Run specific test suite
npx hardhat test test/SageRegistryV2.test.js        # V2 tests
npx hardhat test test/integration-v2.test.js        # Integration tests
```

### Test Coverage

**V2 Test Suite Covers:**

- ✅ 5-Step Public Key Validation (all steps)
- ✅ Key Revocation & Auto-deactivation
- ✅ Hook Integration (DID validation, rate limiting, blacklist)
- ✅ Signature Verification (challenge-response)
- ✅ Access Control & Ownership
- ✅ Gas Usage Optimization
- ✅ Edge Cases (zero keys, invalid formats, replay attacks)

### Integration Tests

```bash
# Full integration test with deployment
npx hardhat test test/integration-v2.test.js

# Test with Hardhat node
npx hardhat node                                    # Terminal 1
npx hardhat test --network localhost                # Terminal 2
```

## Deployment Scripts

### Available Scripts (24 total)

#### Core Deployment

- `deploy-unified.js` - **Recommended**: Single script for all networks
- `deploy-v2.js` - V2 registry deployment (generic)
- `deploy-local.js` - Local development deployment
- `deploy-kaia-v2.js` - Kaia mainnet deployment
- `deploy-kaia-v2-latest.js` - Latest Kaia deployment

#### Utilities

- `interact-local.js` - Interactive CLI for testing
- `query-agents.js` - Query and inspect agents
- `register-agents.js` - Batch agent registration
- `verify-deployment.js` - Verify deployment health
- `verify-contracts.js` - Contract verification on explorer

#### Network-Specific

- `deploy-kaia.js` - Legacy Kaia deployment
- `interact-kaia.js` - Kaia network interaction
- `verify-kaia.js` - Kaia contract verification

#### Helper Scripts

- `check-balance.js` - Check deployer balance
- `extract-abi.js` - Extract contract ABIs
- `generate-go-bindings.js` - Generate Go bindings
- `generate-verification-info.js` - Generate verification data
- `port-manager.sh` - Manage local node ports
- `full-test.sh` - Complete test suite

### Shell Scripts

```bash
# Test scripts
./bin/test-all.sh       # Run all tests (V1 + V2 + Integration)
./bin/test-v2.sh        # Run V2 tests only
./bin/quick-test.js     # Quick smoke test

# Deployment helpers
./bin/deploy-local.sh   # Interactive local deployment
./bin/query-agents.sh   # Query agents from CLI
```

## Project Structure

```
contracts/ethereum/
├── contracts/
│   ├── SageRegistryV4.sol          # V4 multi-key registry (latest)
│   ├── SageRegistryV3.sol          # V3 registry (legacy)
│   ├── SageRegistryV2.sol          # V2 registry (stable production)
│   ├── SageRegistry.sol            # V1 registry (deprecated)
│   ├── SageVerificationHook.sol    # Hook implementation
│   └── interfaces/
│       ├── ISageRegistryV4.sol     # V4 multi-key interface
│       ├── ISageRegistry.sol       # V2 registry interface
│       └── IRegistryHook.sol       # Hook interface
├── test/
│   ├── SageRegistryV4.test.js      # V4 multi-key tests (30 tests)
│   ├── SageRegistryV2.test.js      # V2 unit tests
│   ├── integration-v2.test.js      # V2 integration tests
│   └── SageRegistry.test.fixed.js  # V1 compatibility tests
├── scripts/
│   ├── deploy-unified.js           # Unified deployment (NEW)
│   ├── deploy-v2.js                # V2 deployment
│   ├── deploy-local.js             # Local deployment
│   ├── interact-local.js           # Interactive CLI
│   ├── query-agents.js             # Query utilities
│   ├── register-agents.js          # Batch registration
│   └── verify-deployment.js        # Deployment verification
├── bin/
│   ├── deploy-local.sh             # Local deployment helper
│   ├── test-all.sh                 # Run all tests
│   ├── test-v2.sh                  # V2 tests only
│   └── query-agents.sh             # Query helper
├── deployments/                    # Deployment artifacts (auto-generated)
├── bindings/                       # Language bindings (auto-generated)
│   ├── go/                         # Go bindings
│   └── python/                     # Python bindings
└── hardhat.config.js               # Hardhat configuration
```

## Migration Guide (V1 → V2)

### Key Changes

1. **Signature Method**: V2 uses challenge-response for key ownership proof
2. **Public Key Format**: Must include proper prefix (0x04, 0x02, 0x03)
3. **New Features**: Key revocation, enhanced validation
4. **Gas Cost**: Slightly higher due to enhanced security (~620k vs ~400k)

### Migration Steps

1. Deploy V2 contracts using `deploy-unified.js`
2. Update frontend to use new signature method (challenge-response)
3. Migrate existing agents (optional - V1 remains functional)
4. Update monitoring for new events (`AgentRegistered`, `KeyRevoked`)
5. Test thoroughly on testnet before mainnet migration

### Code Changes Required

```javascript
// V1 (OLD)
const tx = await registry.registerAgent(did, name, desc, endpoint, pubkey, caps);

// V2 (NEW) - Requires ownership proof
const challenge = ethers.keccak256(
  ethers.solidityPacked(
    ["string", "uint256", "address", "address", "bytes32"],
    ["SAGE Key Registration:", chainId, registryAddress, owner, keyHash]
  )
);
const signature = await signer.signMessage(ethers.getBytes(challenge));
const tx = await registry.registerAgent(
  did,
  name,
  desc,
  endpoint,
  pubkey,
  caps,
  signature
);
```

## Network Configuration

### Supported Networks

**Mainnet:**

- Ethereum Mainnet
- Kaia (Cypress)

**Testnet:**

- Sepolia (Ethereum)
- Kairos (Kaia)
- Localhost (Hardhat)

### Hardhat Configuration

```javascript
module.exports = {
  networks: {
    hardhat: {
      chainId: 31337,
    },
    localhost: {
      url: "http://127.0.0.1:8545",
      chainId: 31337,
    },
    kairos: {
      url: process.env.KAIROS_RPC_URL || "https://public-en-kairos.node.kaia.io",
      chainId: 1001,
      accounts: [process.env.PRIVATE_KEY],
    },
    cypress: {
      url: process.env.KAIA_RPC_URL || "https://public-en.node.kaia.io",
      chainId: 8217,
      accounts: [process.env.PRIVATE_KEY],
    },
  },
  solidity: {
    version: "0.8.19",
    settings: {
      optimizer: {
        enabled: true,
        runs: 200,
      },
    },
  },
};
```

## Troubleshooting

### Common Issues

**1. Deployment fails with "insufficient funds"**

```bash
# Check balance
npx hardhat run scripts/check-balance.js --network <network>

# Ensure deployer has enough native tokens (ETH, KAIA, etc.)
```

**2. Contract verification fails**

```bash
# Manual verification
npx hardhat verify --network <network> <CONTRACT_ADDRESS> <CONSTRUCTOR_ARGS>

# Check API keys in .env
ETHERSCAN_API_KEY=...
KAIASCAN_API_KEY=...
```

**3. Hardhat node connection issues**

```bash
# Check if node is running
lsof -i:8545

# Restart node
npx hardhat node --reset

# Check port in hardhat.config.js
```

**4. Test failures**

```bash
# Clean artifacts
npx hardhat clean

# Recompile
npx hardhat compile

# Run tests with verbose output
npx hardhat test --verbose
```

## Contributing

We welcome contributions to the SAGE smart contracts!

### Development Workflow

1. Fork the repository
2. Create feature branch (`git checkout -b feature/amazing-feature`)
3. Make changes and add tests
4. Run test suite (`npm test`)
5. Check gas usage (`npx hardhat test`)
6. Commit changes (`git commit -m 'Add amazing feature'`)
7. Push to branch (`git push origin feature/amazing-feature`)
8. Open Pull Request

### Code Standards

- Follow Solidity style guide
- Add NatSpec comments for all public functions
- Write comprehensive tests (>90% coverage)
- Optimize for gas efficiency
- Document all breaking changes

## License

**MIT License** - See [LICENSE](ethereum/LICENSE) file for details.

This differs from the main SAGE repository (LGPL-3.0) to align with blockchain ecosystem standards and maximize adoption.

## Security

### Audit Status

- V1: Community reviewed
- V2: Internal review completed, external audit pending

### Bug Bounty

We welcome security researchers to review our contracts. Please report vulnerabilities responsibly to security@sage-x-project.org.

### Security Best Practices

1. Never expose private keys
2. Test thoroughly on testnet before mainnet
3. Use hardware wallets for mainnet deployments
4. Monitor for unusual activity
5. Keep dependencies updated

## Resources

- [Hardhat Documentation](https://hardhat.org/docs)
- [OpenZeppelin Contracts](https://docs.openzeppelin.com/contracts)
- [Solidity Documentation](https://docs.soliditylang.org/)
- [Kaia Developer Docs](https://docs.kaia.io)
- [Ethereum Development Resources](https://ethereum.org/developers)

## Support

- **Issues**: [GitHub Issues](https://github.com/SAGE-X-project/sage/issues)
- **Discussions**: [GitHub Discussions](https://github.com/SAGE-X-project/sage/discussions)
- **Documentation**: See [main README](../README.md)

---

**For more information about SAGE, see the [main repository README](../README.md).**
