# SAGE Ethereum Contracts - Implementation Guide

## Overview

This directory contains the Ethereum/EVM implementation of SAGE smart contracts for decentralized AI agent identity management.

**For general SAGE contracts documentation, see [../README.md](../README.md)**

### Current Implementation Status

**V4 (Latest - In Development)**
- Multi-key registry with Ed25519, ECDSA, X25519 support
- Smart contract: `contracts/SageRegistryV4.sol`
- Test suite: `test/SageRegistryV4.test.js` (30 tests, 100% passing)
- Status: Contract complete, pending deployment

**V2 (Stable Production)**
- Enhanced validation with 5-step public key verification
- Deployed on Sepolia testnet: `0x487d45a678eb947bbF9d8f38a67721b13a0209BF`
- Smart contract: `contracts/SageRegistryV2.sol`
- Status: Production ready

**V1 (Deprecated)**
- Legacy implementation with basic signature verification
- Smart contract: `contracts/SageRegistry.sol`
- Status: Archived, not recommended for new deployments

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
│   ├── SageRegistryV4.sol       # V4 multi-key registry (latest)
│   ├── SageRegistryV3.sol       # V3 registry (legacy)
│   ├── SageRegistryV2.sol       # V2 registry with enhanced validation (stable)
│   ├── SageRegistry.sol         # V1 registry (deprecated)
│   ├── SageVerificationHook.sol # Hook implementation
│   └── interfaces/
│       ├── ISageRegistryV4.sol  # V4 interface
│       ├── ISageRegistry.sol    # V2 interface
│       └── IRegistryHook.sol    # Hook interface
├── scripts/                 # Deployment and utility scripts
│   ├── deploy-unified.js        # Unified deployment script
│   ├── deploy-v2.js            # V2 deployment
│   ├── quick-test.js           # Quick testing
│   └── check-balance.js        # Balance checker
├── test/                    # Test files
│   ├── SageRegistryV4.test.js      # V4 multi-key tests (30 tests)
│   ├── SageRegistryV2.test.js      # V2 unit tests
│   ├── integration-v2.test.js      # V2 integration tests
│   └── SageRegistry.test.fixed.js  # V1 compatibility tests
├── bin/                     # Shell scripts
│   ├── quick-start.sh          # Interactive setup
│   ├── deploy-v2.sh            # Deployment helper
│   ├── compile.sh              # Compilation script
│   ├── test.sh                 # Test runner
│   └── test-v2.sh              # V2-specific tests
├── deployments/             # Deployment records (auto-generated)
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

3. **Enhanced Security**
   - ReentrancyGuard protection
   - Ownable2Step for safe ownership transfer
   - Pausable for emergency situations
   - Per-key verification status

### Gas Usage

**V4 Gas Costs:**
- Single-key registration: ~875K gas
- Three-key registration: ~1.3M gas
- Add key: ~50K gas per key
- Revoke key: ~30K gas

**V2 Gas Costs:**
- Registration: ~620K gas
- Update: ~50K gas
- Revocation: ~30K gas

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

See the main contracts documentation:
- [Contracts Overview](../README.md) - General SAGE contracts documentation
- [Deployment Guide](../DEPLOYMENT_GUIDE.md) - Deployment instructions
- [Roadmap](../ROADMAP.md) - Planned features and enhancements
- [Multi-Key Design](../MULTI_KEY_DESIGN.md) - V4 multi-key architecture specification
- [Security Audit (Legacy)](../archived/SECURITY_AUDIT_LEGACY.md) - Historical security findings

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
