# SAGE Ethereum Contracts - Implementation Guide

## Overview

This directory contains the Ethereum/EVM implementation of SAGE smart contracts for decentralized AI agent identity management.

**For general SAGE contracts documentation, see [../README.md](../README.md)**

### Current Implementation Status

**V4 (Latest - Production Ready)** ✅
- Multi-key registry with Ed25519, ECDSA, X25519 support
- W3C Verifiable Credentials and Proof-of-Possession
- Enhanced validation framework (3-layer validation)
- Smart contract: `contracts/SageRegistryV4.sol`
- Test suite: 30 Solidity tests + 85+ Go tests (100% passing)
- Performance: 21 benchmarks, comprehensive gas analysis
- Development: Phase 1, 2 & 3 complete
- **Status**: ✅ Production Ready - Use for all new deployments

**V2 (Deprecated)** ⚠️
- Enhanced validation with 5-step public key verification
- Single-key limitation (superseded by V4)
- Smart contract: `contracts/deprecated/SageRegistryV2.sol`
- Legacy deployment: Sepolia testnet (0x487d45a678eb947bbF9d8f38a67721b13a0209BF)
- **Status**: DEPRECATED - Migrate to V4

**V1 & V3 (Deprecated)** ⚠️
- Legacy implementations with security vulnerabilities (V1) or single-key limitations (V3)
- Smart contracts: `contracts/deprecated/`
- **Status**: DEPRECATED - Do not use for new deployments

## Quick Start

### 1. Install Dependencies

```bash
npm install
```

### 2. Set Up Environment

```bash
# Copy example environment file
cp .env.example .env

# Edit .env and add your private key (without 0x prefix)
nano .env
```

### 3. Get Test KLAY

Get test KLAY from Kairos faucet: <https://kairos.wallet.kaia.io/faucet>

### 4. Run Quick Start Script

```bash
# Interactive setup and deployment
./bin/quick-start.sh
```

## Project Structure

```
sage/contracts/ethereum/
├── contracts/               # Smart contracts
│   ├── SageRegistryV4.sol       # ✅ V4 multi-key registry (PRODUCTION)
│   ├── SageVerificationHook.sol # Hook implementation (V4 compatible)
│   ├── deprecated/              # Legacy contracts (DO NOT USE)
│   │   ├── README.md               # Deprecation guide
│   │   ├── SageRegistry.sol        # V1 (DEPRECATED - security issues)
│   │   ├── SageRegistryV2.sol      # V2 (DEPRECATED - single-key only)
│   │   ├── SageRegistryTest.sol    # V2 test version
│   │   └── SageRegistryV3.sol      # V3 (DEPRECATED - superseded by V4)
│   └── interfaces/
│       ├── ISageRegistryV4.sol  # V4 interface
│       ├── ISageRegistry.sol    # V2 interface (legacy)
│       └── IRegistryHook.sol    # Hook interface
├── scripts/                 # Deployment and utility scripts
│   ├── deploy-v4.js            # V4 deployment
│   ├── deploy-unified.js       # Unified deployment script
│   ├── quick-test.js           # Quick testing
│   └── check-balance.js        # Balance checker
├── test/                    # Test files
│   ├── SageRegistryV4.test.js      # V4 multi-key tests (30 tests, 100% passing)
│   ├── SageRegistryV2.test.js      # V2 unit tests (legacy)
│   ├── integration-v2.test.js      # V2 integration tests (legacy)
│   └── SageRegistry.test.fixed.js  # V1 compatibility tests (legacy)
├── bin/                     # Shell scripts
│   ├── quick-start.sh          # Interactive setup
│   ├── deploy-v4.sh            # V4 deployment helper
│   ├── compile.sh              # Compilation script
│   ├── test.sh                 # Test runner
│   └── test-v4.sh              # V4-specific tests
├── deployments/             # Deployment records (auto-generated)
├── docs/                    # Documentation
│   ├── DEPLOYMENT_GUIDE_V4.md  # V4 deployment instructions
│   └── GAS_COST_ANALYSIS.md    # Gas optimization analysis
├── hardhat.config.js        # Hardhat configuration
└── package.json

```

## Available Scripts

### Shell Scripts (in `bin/` directory)

#### quick-start.sh - Interactive Setup Assistant

```bash
./bin/quick-start.sh
```

Features:

- Checks prerequisites
- Installs dependencies
- Configures environment
- Runs tests
- Provides deployment options

#### deploy-v2.sh - Deployment Helper

```bash
./bin/deploy-v2.sh
```

Options:

1. Local Hardhat Network
2. Kaia Testnet (Kairos)
3. Kaia Mainnet

#### compile.sh - Compile Contracts

```bash
./bin/compile.sh
```

Compiles all contracts with size report.

#### test-v2.sh - Run V2 Tests

```bash
./bin/test-v2.sh
```

Runs comprehensive tests for SageRegistryV2.

### NPM Commands

#### Development

- `npm run compile` - Compile smart contracts
- `npm run clean` - Clean build artifacts
- `npm run test` - Run all tests
- `npm run test:v2` - Run V2 tests only
- `npm run coverage` - Generate coverage report
- `npm run size` - Check contract sizes

#### Deployment

- `npm run deploy:local` - Deploy to local network
- `npm run deploy:kairos` - Deploy to Kairos testnet
- `npm run deploy:kaia` - Deploy to Kaia mainnet

#### Utilities

- `npm run node` - Start local Hardhat node
- `npm run console` - Open Hardhat console
- `npm run quick-test` - Run quick test script

## Testing Guide

### Local Testing

```bash
# Start local node (Terminal 1)
npm run node

# Run tests (Terminal 2)
npm run test

# Quick test with deployment
npx hardhat run scripts/quick-test.js --network localhost
```

### Testnet Testing

```bash
# Check balance first
npx hardhat run scripts/check-balance.js --network kairos

# Deploy to testnet
npm run deploy:kairos

# Test deployed contract
npx hardhat run scripts/test-deployed.js --network kairos
```

### Specific Test Suites

```bash
# V2 tests only
./bin/test-v2.sh

# Integration tests
npx hardhat test test/integration-v2.test.js

# With gas reporting
REPORT_GAS=true npx hardhat test
```

## Deployment Process

### Quick Deployment Commands

#### Local Network

```bash
# Start local node
npm run node

# Deploy to local
npm run deploy:local
```

#### Kairos Testnet

```bash
# Check balance
npx hardhat run scripts/check-balance.js --network kairos

# Deploy to Kairos
npm run deploy:kairos
```

#### Kaia Mainnet

```bash
# Deploy to mainnet (use with caution!)
npm run deploy:kaia
```

### Detailed Deployment Guide

For comprehensive deployment instructions including Sepolia testnet support, see [📚 Deployment Guide](docs/DEPLOYMENT_GUIDE.md).

## Configuration

### Supported Networks

| Network | Chain ID | Type | RPC URL |
|---------|----------|------|---------|
| **Local** | 31337 | Development | <http://127.0.0.1:8545> |
| **Kairos** | 1001 | Testnet | <https://public-en-kairos.node.kaia.io> |
| **Kaia** | 8217 | Mainnet | <https://public-en.node.kaia.io> |
| **Sepolia** | 11155111 | Testnet | Configure in .env (optional) |

### Environment Setup

1. **Copy the example environment file:**

```bash
cp .env.example .env
```

2. **Edit `.env` with your configuration:**

```env
# Required for deployment
PRIVATE_KEY=your_private_key_without_0x
MAINNET_PRIVATE_KEY=your_mainnet_key_for_production

# Network RPC URLs (optional - defaults provided)
KAIROS_RPC_URL=https://public-en-kairos.node.kaia.io
KAIA_RPC_URL=https://public-en.node.kaia.io

# Optional: Sepolia testnet
# SEPOLIA_RPC_URL=https://sepolia.infura.io/v3/YOUR_KEY
# SEPOLIA_PRIVATE_KEY=your_sepolia_test_key

# Gas settings (optional)
GAS_PRICE_GWEI=250
GAS_LIMIT=3000000
```

3. **Get Test Tokens:**
   - **Kairos KLAY**: <https://kairos.wallet.kaia.io/faucet>
   - **Sepolia ETH**: <https://sepoliafaucet.com>

### Enhanced Configuration Features

The updated `hardhat.config.js` now includes:

- **Environment variable validation** with helpful warnings
- **Dynamic network configuration** based on .env settings
- **Sepolia testnet support** (automatically enabled if configured)
- **Gas price and limit customization**
- **Separate mainnet/testnet private keys**
- **Improved error messages** for missing configurations

## Security Features

### SageRegistryV4 (Latest)

1. **Multi-Key Validation**
   - Type-specific verification (Ed25519, ECDSA, X25519)
   - ECDSA keys verified on-chain via ecrecover
   - Ed25519 keys require registry owner pre-approval
   - X25519 keys automatically verified for encryption

2. **Key Lifecycle Management**
   - Add/revoke keys independently
   - Maximum 10 keys per agent (DoS prevention)
   - Key-specific verification status tracking
   - Owner-controlled approval for Ed25519 keys

3. **Enhanced Validation Framework** (Phase 3 - Complete)
   - **Level 1**: Basic A2A card structure validation
   - **Level 2**: W3C Verifiable Credentials with cryptographic proof
   - **Level 3**: DID cross-validation against blockchain
   - Proof-of-Possession mechanism (challenge-response)
   - Prevents card tampering and ensures key ownership

4. **Smart Contract Security**
   - ReentrancyGuard protection
   - Ownable2Step for safe ownership transfer
   - Pausable for emergency situations
   - Per-key verification status
   - Hook support for custom validation logic

### Gas Usage

See [GAS_COST_ANALYSIS.md](GAS_COST_ANALYSIS.md) for detailed analysis.

**V4 Gas Costs (Ethereum Mainnet):**
- Single-key registration: ~907K gas (~$40 at 50 gwei, $2000 ETH)
- Dual-key registration: ~1.1M gas (~$48)
- Triple-key registration: ~1.3M gas (~$57)
- Add key: ~50K gas per key (~$2.20)
- Revoke key: ~30K gas (~$1.32)

**Layer 2 Optimization (Optimism/Arbitrum):**
- 97% cost reduction vs Ethereum mainnet
- Single-key: ~$1.20 (vs $40 on mainnet)
- Production recommendation: Deploy to L2 for cost efficiency

**V2 Gas Costs (Legacy - Single Key):**
- Registration: ~734K gas (~$32 at 50 gwei, $2000 ETH)
- V4 single-key overhead: +23.6% vs V2 (for multi-key flexibility)

## Troubleshooting

### Common Issues

#### Compilation Warnings

Fixed in latest version - no warnings

#### "Insufficient funds" Error

```bash
# Check balance
npx hardhat run scripts/check-balance.js --network kairos

# Get test KLAY
# Visit: https://kairos.wallet.kaia.io/faucet
```

#### "Key ownership not proven" Error

Ensure signature generation matches contract expectations:

```javascript
const challenge = keccak256(solidityPacked(
  ["string", "uint256", "address", "address", "bytes32"],
  ["SAGE Key Registration:", chainId, contractAddress, signerAddress, keyHash]
));
```

#### Network Connection Issues

```bash
# Test connection
npx hardhat run scripts/test-connection.js --network kairos

# Check RPC status
curl -X POST -H "Content-Type: application/json" \
  --data '{"jsonrpc":"2.0","method":"eth_blockNumber","params":[],"id":1}' \
  https://public-en-kairos.node.kaia.io
```

## Documentation

### Core Documentation
- [Contracts Overview](../README.md) - General SAGE contracts documentation
- [Roadmap](../ROADMAP.md) - Feature roadmap (Phase 1, 2 & 3 complete)
- [TODO](../TODO.md) - Current tasks and progress tracking

### V4 Documentation
- [Multi-Key Design](../MULTI_KEY_DESIGN.md) - V4 architecture specification
- [Deployment Guide V4](DEPLOYMENT_GUIDE_V4.md) - V4 deployment instructions
- [Gas Cost Analysis](GAS_COST_ANALYSIS.md) - V2 vs V4 gas optimization
- [Performance Benchmarks](../../docs/PERFORMANCE_BENCHMARKS.md) - 21 benchmark functions

### Legacy Documentation
- [Deprecated Contracts](contracts/deprecated/README.md) - V1/V2/V3 deprecation guide
- [Security Audit (Legacy)](../archived/SECURITY_AUDIT_LEGACY.md) - Historical security findings
- [Code Analysis V1/V2/V3](../archived/CODE_ANALYSIS_V1_V2_V3.md) - Legacy contract analysis

## Resources

- [Kaia Documentation](https://docs.kaia.io)
- [Kaia Explorer](https://kaiascope.com)
- [Hardhat Documentation](https://hardhat.org)
- [Ethers.js Documentation](https://docs.ethers.org)

## License

This directory contains SAGE smart contracts licensed under the **MIT License**.

**Note**: The main SAGE Go backend (parent directory) is licensed under LGPL-3.0. Smart contracts are separately licensed as MIT to align with blockchain ecosystem standards and OpenZeppelin compatibility.

See [LICENSE](LICENSE) for the full MIT license text.

### Why MIT for Smart Contracts?

- **Industry Standard**: Most DeFi and blockchain projects use MIT
- **OpenZeppelin Compatible**: Our contracts use OpenZeppelin (MIT)
- **Audit Friendly**: Standard license for security auditors
- **On-Chain Transparency**: Smart contracts are publicly verifiable on-chain

### Dependencies

- OpenZeppelin Contracts: MIT License
- Hardhat: MIT License
- Ethers.js: MIT License
