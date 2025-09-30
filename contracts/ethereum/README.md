# SAGE Smart Contracts for Kaia Network

## Overview
SAGE Registry smart contracts for managing AI agent identities on Kaia blockchain.

### 🆕 Version 2 Features
- **Enhanced Public Key Validation**: 5-step verification process
- **Key Revocation**: Ability to revoke compromised keys
- **Zero-Key Prevention**: Protection against invalid keys
- **Ownership Proof**: Signature-based key ownership verification
- **Ed25519 Rejection**: Explicit rejection of unsupported key types

##  Quick Start

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
Get test KLAY from Kairos faucet: https://kairos.wallet.kaia.io/faucet

### 4. Run Quick Start Script
```bash
# Interactive setup and deployment
./bin/quick-start.sh
```

## 📁 Project Structure

```
sage/contracts/ethereum/
├── contracts/               # Smart contracts
│   ├── SageRegistry.sol     # V1 Registry (deprecated)
│   ├── SageRegistryV2.sol   # V2 Registry with enhanced validation
│   ├── SageVerificationHook.sol
│   └── interfaces/
├── scripts/                 # Deployment and utility scripts
│   ├── deploy-v2.js        # Main deployment script
│   ├── quick-test.js       # Quick testing script
│   └── check-balance.js   # Balance checker
├── test/                   # Test files
│   ├── SageRegistryV2.test.js
│   └── integration-v2.test.js
├── bin/                    # Shell scripts
│   ├── quick-start.sh      # Interactive setup
│   ├── deploy-v2.sh        # Deployment helper
│   ├── compile.sh          # Compilation script
│   ├── test.sh             # Test runner
│   └── test-v2.sh          # V2-specific tests
├── deployments/            # Deployment records
├── hardhat.config.js       # Hardhat configuration
└── package.json

```

## 🛠 Available Scripts

### Shell Scripts (in `bin/` directory)

#### **quick-start.sh** - Interactive Setup Assistant
```bash
./bin/quick-start.sh
```
Features:
- Checks prerequisites
- Installs dependencies
- Configures environment
- Runs tests
- Provides deployment options

#### **deploy-v2.sh** - Deployment Helper
```bash
./bin/deploy-v2.sh
```
Options:
1. Local Hardhat Network
2. Kaia Testnet (Kairos)
3. Kaia Mainnet

#### **compile.sh** - Compile Contracts
```bash
./bin/compile.sh
```
Compiles all contracts with size report.

#### **test-v2.sh** - Run V2 Tests
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

##  Testing Guide

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

##  Deployment Process

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

##  Configuration

### Supported Networks

| Network | Chain ID | Type | RPC URL |
|---------|----------|------|---------|
| **Local** | 31337 | Development | http://127.0.0.1:8545 |
| **Kairos** | 1001 | Testnet | https://public-en-kairos.node.kaia.io |
| **Kaia** | 8217 | Mainnet | https://public-en.node.kaia.io |
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
   - **Kairos KLAY**: https://kairos.wallet.kaia.io/faucet
   - **Sepolia ETH**: https://sepoliafaucet.com

### Enhanced Configuration Features

The updated `hardhat.config.js` now includes:
- **Environment variable validation** with helpful warnings
- **Dynamic network configuration** based on .env settings
- **Sepolia testnet support** (automatically enabled if configured)
- **Gas price and limit customization**
- **Separate mainnet/testnet private keys**
- **Improved error messages** for missing configurations

##  Security Features

### SageRegistryV2 Enhancements
1. **Public Key Validation**
   - Format checking (0x04, 0x02, 0x03)
   - Zero-key prevention
   - Length validation (32-65 bytes)

2. **Ownership Verification**
   - Challenge-response signature
   - Private key possession proof
   - Replay attack protection

3. **Key Management**
   - Key revocation capability
   - Automatic agent deactivation
   - Revoked key tracking

### Gas Usage
- Registration: ~620K gas
- Update: ~50K gas
- Revocation: ~30K gas

## 🐛 Troubleshooting

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

## 📚 Documentation

- [Deployment Guide](DEPLOYMENT_GUIDE.md) - Detailed deployment instructions
- [Migration Guide](MIGRATION_GUIDE_V2.md) - V1 to V2 migration
- [Code Review](CODE_REVIEW_V2.md) - Security analysis
- [Key Validation](KEY_VALIDATION_IMPROVEMENTS.md) - Technical improvements

##  Resources

- [Kaia Documentation](https://docs.kaia.io)
- [Kaia Explorer](https://kaiascope.com)
- [Hardhat Documentation](https://hardhat.org)
- [Ethers.js Documentation](https://docs.ethers.org)

## 📄 License
MIT